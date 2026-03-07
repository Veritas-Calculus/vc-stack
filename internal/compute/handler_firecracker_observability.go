package compute

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// --- Phase 3: Serial Console Log ---

// consoleFirecrackerHandler returns the serial console output for a Firecracker microVM.
// GET /v1/firecracker/:id/console?lines=100&follow=false
func (s *Service) consoleFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Determine log file path.
	logPath := fmt.Sprintf("/srv/firecracker/logs/fc-%d.log", instance.ID)

	// Read requested number of tail lines.
	lines := 100
	if l := c.Query("lines"); l != "" {
		if v, err := strconv.Atoi(l); err == nil && v > 0 && v <= 10000 {
			lines = v
		}
	}

	content, err := tailFile(logPath, lines)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"instance_id": instance.ID,
			"log":         "",
			"error":       "Log file not available: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": instance.ID,
		"log":         content,
		"lines":       lines,
		"path":        logPath,
	})
}

// --- Phase 3: Metrics ---

// metricsFirecrackerHandler returns runtime metrics for a Firecracker microVM.
// GET /v1/firecracker/:id/metrics
func (s *Service) metricsFirecrackerHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var instance FirecrackerInstance
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).First(&instance).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Firecracker instance not found"})
		return
	}

	// Try to flush and read metrics from the Firecracker metrics file.
	metricsPath := fmt.Sprintf("/srv/firecracker/logs/fc-%d.metrics", instance.ID)

	// Flush metrics via the client if VM is running.
	if client, ok := s.fcRegistry.Get(instance.ID); ok && client.IsRunning() {
		if _, err := client.GetMetrics(c.Request.Context()); err != nil {
			s.logger.Debug("Metrics flush failed", zap.Error(err))
		}
	}

	// Read the last metrics JSON line from the file.
	metricsJSON, err := readLastJSONLine(metricsPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"instance_id": instance.ID,
			"status":      instance.PowerState,
			"metrics":     nil,
			"error":       "Metrics not available: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"instance_id": instance.ID,
		"status":      instance.PowerState,
		"metrics":     metricsJSON,
	})
}

// --- Phase 3: WebSocket Real-time Status ---

// fcStatusHub manages WebSocket connections for Firecracker status updates.
type fcStatusHub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]bool
}

var fcHub = &fcStatusHub{
	clients: make(map[*websocket.Conn]bool),
}

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// FCStatusEvent is broadcast to WebSocket clients on state changes.
type FCStatusEvent struct {
	Type       string `json:"type"` // "status_change"
	InstanceID uint   `json:"instance_id"`
	Name       string `json:"name"`
	Status     string `json:"status"`
	PowerState string `json:"power_state"`
	PID        int    `json:"pid,omitempty"`
	Timestamp  string `json:"timestamp"`
}

// wsFirecrackerStatus handles WebSocket connections for real-time status.
// WS /ws/firecracker/status
func (s *Service) wsFirecrackerStatus(c *gin.Context) {
	conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	fcHub.mu.Lock()
	fcHub.clients[conn] = true
	fcHub.mu.Unlock()

	s.logger.Info("WebSocket client connected for Firecracker status")

	// Send initial state: all running VMs.
	running := s.fcRegistry.AllRunning()
	for vmID, pid := range running {
		var inst FirecrackerInstance
		if err := s.db.First(&inst, vmID).Error; err == nil {
			evt := FCStatusEvent{
				Type:       "initial",
				InstanceID: inst.ID,
				Name:       inst.Name,
				Status:     inst.Status,
				PowerState: inst.PowerState,
				PID:        pid,
				Timestamp:  time.Now().Format(time.RFC3339),
			}
			msg, _ := json.Marshal(evt)
			_ = conn.WriteMessage(websocket.TextMessage, msg)
		}
	}

	// Keep connection alive, read messages (pings) from client.
	defer func() {
		fcHub.mu.Lock()
		delete(fcHub.clients, conn)
		fcHub.mu.Unlock()
		_ = conn.Close()
		s.logger.Info("WebSocket client disconnected")
	}()

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

// BroadcastFCStatus sends a status event to all connected WebSocket clients.
func BroadcastFCStatus(event FCStatusEvent) {
	event.Timestamp = time.Now().Format(time.RFC3339)
	msg, err := json.Marshal(event)
	if err != nil {
		return
	}

	fcHub.mu.RLock()
	defer fcHub.mu.RUnlock()

	for conn := range fcHub.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			// Mark for removal but don't modify map during iteration.
			go func(c *websocket.Conn) {
				fcHub.mu.Lock()
				delete(fcHub.clients, c)
				fcHub.mu.Unlock()
				_ = c.Close()
			}(conn)
		}
	}
}

// --- Helpers ---

// tailFile reads the last N lines from a file.
func tailFile(path string, n int) (string, error) {
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	var lines []string
	scanner := bufio.NewScanner(f)
	// Use a larger buffer for potentially long log lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return strings.Join(lines, "\n"), nil
}

// readLastJSONLine reads the last non-empty line from a file and parses it as JSON.
func readLastJSONLine(path string) (interface{}, error) {
	f, err := os.Open(path) // #nosec G304
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var lastLine string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lastLine = line
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if lastLine == "" {
		return nil, fmt.Errorf("no metrics data")
	}

	var result interface{}
	if err := json.Unmarshal([]byte(lastLine), &result); err != nil {
		return nil, fmt.Errorf("invalid metrics JSON: %w", err)
	}
	return result, nil
}
