package gateway

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// listWebShellSessions lists all WebShell sessions with filtering.
func (s *Service) listWebShellSessions(c *gin.Context) {
	// Parse query parameters.
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	// Build query.
	query := s.db.Model(&models.WebShellSession{})

	// Filter by username.
	if username := c.Query("username"); username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}

	// Filter by remote host.
	if remoteHost := c.Query("remote_host"); remoteHost != "" {
		query = query.Where("remote_host LIKE ?", "%"+remoteHost+"%")
	}

	// Filter by status.
	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total.
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.logger.Error("Failed to count sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count sessions"})
		return
	}

	// Fetch sessions.
	var sessions []models.WebShellSession
	if err := query.Order("started_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&sessions).Error; err != nil {
		s.logger.Error("Failed to fetch sessions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch sessions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     total,
		"page":      page,
		"page_size": pageSize,
		"data":      sessions,
	})
}

// getWebShellSession gets a single session by ID or session_id.
func (s *Service) getWebShellSession(c *gin.Context) {
	sessionIDParam := c.Param("id")

	var session models.WebShellSession

	// Try as session_id first (hex string), then as numeric ID.
	err := s.db.Where("session_id = ?", sessionIDParam).First(&session).Error
	if err != nil {
		// Try as numeric ID.
		if id, parseErr := strconv.ParseUint(sessionIDParam, 10, 32); parseErr == nil {
			err = s.db.First(&session, id).Error
		}
	}

	if err != nil {
		s.logger.Error("Session not found", zap.String("id", sessionIDParam), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// getWebShellSessionEvents gets events for a session (for replay).
func (s *Service) getWebShellSessionEvents(c *gin.Context) {
	sessionIDParam := c.Param("id")

	// Verify session exists and get session_id.
	var session models.WebShellSession
	err := s.db.Where("session_id = ?", sessionIDParam).First(&session).Error
	if err != nil {
		if id, parseErr := strconv.ParseUint(sessionIDParam, 10, 32); parseErr == nil {
			err = s.db.First(&session, id).Error
		}
	}

	if err != nil {
		s.logger.Error("Session not found", zap.String("id", sessionIDParam), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Fetch events for this session.
	var events []models.WebShellEvent
	if err := s.db.Where("session_id = ?", session.SessionID).
		Order("time_offset ASC").
		Find(&events).Error; err != nil {
		s.logger.Error("Failed to fetch session events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"session": session,
		"events":  events,
	})
}

// exportWebShellSession exports a session in asciinema format.
func (s *Service) exportWebShellSession(c *gin.Context) {
	sessionIDParam := c.Param("id")

	// Verify session exists.
	var session models.WebShellSession
	err := s.db.Where("session_id = ?", sessionIDParam).First(&session).Error
	if err != nil {
		if id, parseErr := strconv.ParseUint(sessionIDParam, 10, 32); parseErr == nil {
			err = s.db.First(&session, id).Error
		}
	}

	if err != nil {
		s.logger.Error("Session not found", zap.String("id", sessionIDParam), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Fetch events.
	var events []models.WebShellEvent
	if err := s.db.Where("session_id = ?", session.SessionID).
		Order("time_offset ASC").
		Find(&events).Error; err != nil {
		s.logger.Error("Failed to fetch session events", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch events"})
		return
	}

	// Generate asciinema format.
	// Header line.
	header := map[string]interface{}{
		"version":   2,
		"width":     80,
		"height":    24,
		"timestamp": session.StartedAt.Unix(),
		"env": map[string]string{
			"SHELL": "/bin/bash",
			"TERM":  "xterm-256color",
		},
		"title": fmt.Sprintf("%s@%s:%d", session.Username, session.RemoteHost, session.RemotePort),
	}

	headerJSON, err := json.Marshal(header)
	if err != nil {
		s.logger.Error("Failed to marshal header", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to prepare export"})
		return
	}
	c.Writer.Header().Set("Content-Type", "application/x-asciicast")
	c.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=webshell-%s.cast", session.SessionID))
	c.Writer.WriteHeader(http.StatusOK)

	// Write header.
	if _, err := c.Writer.Write(headerJSON); err != nil {
		s.logger.Warn("Failed to write header", zap.Error(err))
		return
	}
	if _, err := c.Writer.WriteString("\n"); err != nil {
		s.logger.Warn("Failed to write newline", zap.Error(err))
		return
	}

	// Write events.
	for _, event := range events {
		if event.EventType != "output" || event.Data == "" {
			continue
		}
		timestamp := float64(event.TimeOffset) / 1000.0 // Convert to seconds
		eventLine := []interface{}{timestamp, "o", event.Data}
		eventJSON, err := json.Marshal(eventLine)
		if err != nil {
			s.logger.Warn("Failed to marshal event", zap.Error(err))
			continue
		}
		if _, err := c.Writer.Write(eventJSON); err != nil {
			s.logger.Warn("Failed to write event", zap.Error(err))
			return
		}
		if _, err := c.Writer.WriteString("\n"); err != nil {
			s.logger.Warn("Failed to write newline", zap.Error(err))
			return
		}
	}
}
