package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// consoleWebSocketHandler handles WebSocket console requests with dynamic node routing.
// URL format: /ws/console/{node_id}?token=xxx.
func (s *Service) consoleWebSocketHandler(c *gin.Context) {
	nodeID := c.Param("node_id")
	token := c.Query("token")

	if nodeID == "" || token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing node_id or token"})
		return
	}

	// Try to lookup node address from scheduler.
	nodeAddr, err := s.lookupNodeAddress(c.Request.Context(), nodeID)
	if err != nil {
		// Fallback: if scheduler lookup fails, use lite service directly.
		s.logger.Warn("Scheduler lookup failed, using lite service fallback",
			zap.String("node_id", nodeID),
			zap.Error(err))

		// Use lite service proxy if available.
		s.mu.RLock()
		liteProxy, hasLite := s.services["lite"]
		s.mu.RUnlock()

		if hasLite && liteProxy.Target != nil {
			nodeAddr = liteProxy.Target.String()
		} else {
			s.logger.Error("No lite service configured and scheduler lookup failed",
				zap.String("node_id", nodeID))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "cannot route to node: scheduler unavailable and no lite service configured"})
			return
		}
	}

	// Parse node address.
	targetURL, err := url.Parse(nodeAddr)
	if err != nil {
		s.logger.Error("Invalid node address",
			zap.String("node_id", nodeID),
			zap.String("address", nodeAddr),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "invalid node address"})
		return
	}

	// Build WebSocket URL to node's VM driver.
	wsURL := url.URL{
		Scheme:   "ws",
		Host:     targetURL.Host,
		Path:     "/ws/console",
		RawQuery: "token=" + token,
	}
	if targetURL.Scheme == "https" {
		wsURL.Scheme = "wss"
	}

	s.logger.Info("Proxying console WebSocket",
		zap.String("node_id", nodeID),
		zap.String("target", wsURL.String()))

	// Upgrade client connection.
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade client connection", zap.Error(err))
		return
	}
	defer func() { _ = clientConn.Close() }()

	// Dial backend WebSocket.
	backendConn, resp, err := websocket.DefaultDialer.Dial(wsURL.String(), nil)
	if err != nil {
		s.logger.Error("Failed to dial backend WebSocket",
			zap.String("url", wsURL.String()),
			zap.Error(err))
		if err := clientConn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "backend connection failed")); err != nil {
			s.logger.Warn("failed to send close message", zap.Error(err))
		}
		return
	}
	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	defer func() { _ = backendConn.Close() }()

	// Bidirectional proxy.
	errChan := make(chan error, 2)

	// Client -> Backend.
	go func() {
		for {
			msgType, data, err := clientConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := backendConn.WriteMessage(msgType, data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Backend -> Client.
	go func() {
		for {
			msgType, data, err := backendConn.ReadMessage()
			if err != nil {
				errChan <- err
				return
			}
			if err := clientConn.WriteMessage(msgType, data); err != nil {
				errChan <- err
				return
			}
		}
	}()

	// Wait for error or completion.
	err = <-errChan
	if err != nil && err != io.EOF {
		s.logger.Debug("Console WebSocket proxy closed", zap.Error(err))
	}
}

// lookupNodeAddress queries the scheduler for a node's address.
func (s *Service) lookupNodeAddress(ctx context.Context, nodeID string) (string, error) {
	schedProxy, ok := s.services["scheduler"]
	if !ok {
		return "", fmt.Errorf("scheduler service not configured")
	}

	// Call scheduler API: GET /api/v1/nodes/{nodeID}.
	reqURL := fmt.Sprintf("%s/api/v1/nodes/%s", schedProxy.Target.String(), nodeID)
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, http.NoBody)
	if err != nil {
		return "", err
	}

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req) //nolint:bodyclose // closed below
	if err != nil {
		return "", fmt.Errorf("scheduler request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("scheduler returned %d and failed to read body: %w", resp.StatusCode, err)
		}
		return "", fmt.Errorf("scheduler returned %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Node struct {
			ID      string `json:"id"`
			Address string `json:"address"`
		} `json:"node"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode scheduler response: %w", err)
	}

	if result.Node.Address == "" {
		return "", fmt.Errorf("node has no address")
	}

	return strings.TrimSpace(result.Node.Address), nil
}
