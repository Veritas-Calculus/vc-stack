package selfheal

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Service) getStatus(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	healthy := 0
	for _, ch := range s.checks {
		if ch.Status == "healthy" {
			healthy++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           "operational",
		"total_checks":     len(s.checks),
		"healthy":          healthy,
		"degraded":         len(s.checks) - healthy,
		"active_policies":  len(s.policies),
		"total_events":     len(s.events),
		"healing_rate_pct": s.healingRate(),
	})
}

func (s *Service) listChecks(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resourceType := c.Query("resource_type")
	var result []HealthCheck
	for _, ch := range s.checks {
		if resourceType == "" || ch.ResourceType == resourceType {
			result = append(result, ch)
		}
	}

	c.JSON(http.StatusOK, gin.H{"checks": result})
}

func (s *Service) runCheck(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := c.Param("id")
	ch := s.findCheck(id)
	if ch == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check not found"})
		return
	}

	// Simulate running the check — mark as healthy and update timestamp.
	ch.Status = "healthy"
	ch.LastChecked = time.Now()

	c.JSON(http.StatusOK, gin.H{"check": *ch})
}

func (s *Service) listPolicies(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"policies": s.policies})
}

func (s *Service) simulateIncident(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var req struct {
		CheckID string `json:"check_id"`
		Type    string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	action, valid := actionForIncident(req.Type)
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown incident type"})
		return
	}

	// If a check_id was specified, degrade it then heal it.
	if req.CheckID != "" {
		if ch := s.findCheck(req.CheckID); ch != nil {
			ch.Status = "healthy" // healed after remediation
			ch.LastChecked = time.Now()
		}
	}

	event := HealingEvent{
		ID:        s.nextEventID(),
		CheckID:   req.CheckID,
		Type:      req.Type,
		Action:    action,
		Status:    "success",
		Details:   "Simulated incident resolved",
		CreatedAt: time.Now(),
	}
	s.events = append(s.events, event)

	c.JSON(http.StatusCreated, gin.H{"event": event})
}

func (s *Service) listEvents(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	c.JSON(http.StatusOK, gin.H{"events": s.events})
}
