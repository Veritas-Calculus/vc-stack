package gateway

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// sessionRecorder records SSH session for audit and replay.
type sessionRecorder struct {
	db        *gorm.DB
	sessionID string
	startTime time.Time
	events    []models.WebShellEvent
	mu        sync.Mutex
	bytesSent int64
	bytesRecv int64
}

// newSessionRecorder creates a new session recorder.
func newSessionRecorder(db *gorm.DB) *sessionRecorder {
	return &sessionRecorder{
		db:        db,
		sessionID: generateSessionID(),
		startTime: time.Now(),
		events:    make([]models.WebShellEvent, 0, 1000),
	}
}

// generateSessionID creates a unique session ID.
func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("sess_%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// recordEvent records a session event.
func (r *sessionRecorder) recordEvent(eventType, data string, cols, rows *int) {
	if r.db == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	offset := time.Since(r.startTime).Milliseconds()
	event := models.WebShellEvent{
		SessionID:  r.sessionID,
		EventType:  eventType,
		EventTime:  time.Now(),
		Data:       data,
		DataSize:   len(data),
		Cols:       cols,
		Rows:       rows,
		TimeOffset: offset,
	}

	r.events = append(r.events, event)

	// Batch save every 100 events
	if len(r.events) >= 100 {
		r.flushEvents()
	}
}

// flushEvents saves buffered events to database.
func (r *sessionRecorder) flushEvents() {
	if len(r.events) == 0 {
		return
	}

	events := make([]models.WebShellEvent, len(r.events))
	copy(events, r.events)
	r.events = r.events[:0]

	// Save asynchronously
	go func() {
		if err := r.db.Create(&events).Error; err != nil {
			// Log error but don't fail the session
			fmt.Printf("Failed to save session events: %v\n", err)
		}
	}()
}

// close finalizes the session recording.
func (r *sessionRecorder) close(status string) {
	if r.db == nil {
		return
	}

	r.mu.Lock()
	r.flushEvents()
	r.mu.Unlock()

	// Update session record
	endTime := time.Now()
	duration := int(endTime.Sub(r.startTime).Seconds())

	_ = r.db.Model(&models.WebShellSession{}).
		Where("session_id = ?", r.sessionID).
		Updates(map[string]interface{}{
			"status":        status,
			"ended_at":      endTime,
			"duration_secs": duration,
			"bytes_sent":    r.bytesSent,
			"bytes_recv":    r.bytesRecv,
		}).Error
}

// WebShellConnectRequest represents the connection request payload.
type WebShellConnectRequest struct {
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port"`
	User       string `json:"user" binding:"required"`
	AuthMethod string `json:"auth_method" binding:"required,oneof=password key"`
	Password   string `json:"password"`
	PrivateKey string `json:"private_key"`
}

// WebShellMessage represents WebSocket message structure.
type WebShellMessage struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// webShellHandler handles WebSocket connections for SSH sessions.
func (s *Service) webShellHandler(c *gin.Context) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		HandshakeTimeout: 10 * time.Second,
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("Failed to upgrade WebSocket connection", zap.Error(err))
		return
	}
	defer conn.Close()

	// Read connection request.
	var req WebShellConnectRequest
	if err := conn.ReadJSON(&req); err != nil {
		s.logger.Error("Failed to read connection request", zap.Error(err))
		s.sendWebShellError(conn, "Invalid connection request")
		return
	}

	// Log received parameters (without sensitive data)
	s.logger.Info("WebShell connection request received",
		zap.String("host", req.Host),
		zap.Int("port", req.Port),
		zap.String("user", req.User),
		zap.String("auth_method", req.AuthMethod),
		zap.Bool("has_password", req.Password != ""),
		zap.Bool("has_key", req.PrivateKey != ""))

	// Validate request.
	if req.Port == 0 {
		req.Port = 22
	}
	if req.AuthMethod == "password" && req.Password == "" {
		s.logger.Error("Password authentication requested but password is empty")
		s.sendWebShellError(conn, "Password is required for password authentication")
		return
	}
	if req.AuthMethod == "key" && req.PrivateKey == "" {
		s.logger.Error("Key authentication requested but private key is empty")
		s.sendWebShellError(conn, "Private key is required for key authentication")
		return
	}

	// Initialize session recorder.
	recorder := newSessionRecorder(s.db)
	currentCols, currentRows := 80, 24

	// Get client IP for audit.
	clientIP := c.ClientIP()

	// Parse user ID from context.
	var userID *uint
	if userIDStr := c.GetString("user_id"); userIDStr != "" {
		// Convert string to uint if needed
		// For now, leave it nil if not properly set
	}

	// Create session record.
	sessionRecord := models.WebShellSession{
		SessionID:  recorder.sessionID,
		UserID:     userID,
		Username:   req.User,
		RemoteHost: req.Host,
		RemotePort: req.Port,
		RemoteUser: req.User,
		ClientIP:   clientIP,
		AuthMethod: req.AuthMethod,
		Status:     "connecting",
		StartedAt:  time.Now(),
	}

	if err := s.db.Create(&sessionRecord).Error; err != nil {
		s.logger.Error("Failed to create session record", zap.Error(err))
		// Continue even if recording fails
	}

	defer func() {
		recorder.close("closed")
	}()

	// Create SSH client config.
	config := &ssh.ClientConfig{
		User:            req.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
		ClientVersion:   "SSH-2.0-OpenSSH_8.0", // Improve compatibility
	}

	// Set auth method.
	if req.AuthMethod == "password" {
		// Try both password and keyboard-interactive for better compatibility
		config.Auth = []ssh.AuthMethod{
			ssh.Password(req.Password),
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) ([]string, error) {
				// Answer all questions with the password
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = req.Password
				}
				return answers, nil
			}),
			ssh.PasswordCallback(func() (string, error) {
				return req.Password, nil
			}),
		}
		s.logger.Info("Using password authentication",
			zap.String("user", req.User),
			zap.String("host", req.Host),
			zap.Int("password_length", len(req.Password)))
	} else {
		signer, err := ssh.ParsePrivateKey([]byte(req.PrivateKey))
		if err != nil {
			s.logger.Error("Failed to parse private key", zap.Error(err))
			recorder.close("auth_failed")
			s.sendWebShellError(conn, fmt.Sprintf("Invalid private key: %v", err))
			return
		}
		config.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
		s.logger.Info("Using public key authentication", zap.String("user", req.User), zap.String("host", req.Host))
	}

	// Connect to SSH server.
	addr := fmt.Sprintf("%s:%d", req.Host, req.Port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		s.logger.Error("Failed to connect to SSH server", zap.Error(err), zap.String("addr", addr), zap.String("user", req.User))
		recorder.close("connection_failed")

		// Provide user-friendly error message
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "unable to authenticate") {
			s.sendWebShellError(conn, fmt.Sprintf("Authentication failed for user '%s'@%s. Please verify your password or SSH key.", req.User, req.Host))
		} else if strings.Contains(errorMsg, "connection refused") {
			s.sendWebShellError(conn, fmt.Sprintf("Connection refused by %s:%d. SSH service may not be running.", req.Host, req.Port))
		} else if strings.Contains(errorMsg, "timeout") || strings.Contains(errorMsg, "deadline exceeded") {
			s.sendWebShellError(conn, fmt.Sprintf("Connection timeout to %s:%d. Host may be unreachable or firewall is blocking.", req.Host, req.Port))
		} else if strings.Contains(errorMsg, "no route to host") {
			s.sendWebShellError(conn, fmt.Sprintf("No route to host %s. Please check network connectivity.", req.Host))
		} else {
			s.sendWebShellError(conn, fmt.Sprintf("Connection failed: %v", err))
		}
		return
	}
	defer sshClient.Close()

	// Create SSH session.
	session, err := sshClient.NewSession()
	if err != nil {
		s.logger.Error("Failed to create SSH session", zap.Error(err))
		recorder.close("session_failed")
		s.sendWebShellError(conn, fmt.Sprintf("Failed to create session: %v", err))
		return
	}
	defer session.Close()

	// Get I/O pipes.
	stdin, err := session.StdinPipe()
	if err != nil {
		s.logger.Error("Failed to get stdin pipe", zap.Error(err))
		recorder.close("io_error")
		s.sendWebShellError(conn, "Failed to setup session I/O")
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		s.logger.Error("Failed to get stdout pipe", zap.Error(err))
		recorder.close("io_error")
		s.sendWebShellError(conn, "Failed to setup session I/O")
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		s.logger.Error("Failed to get stderr pipe", zap.Error(err))
		recorder.close("io_error")
		s.sendWebShellError(conn, "Failed to setup session I/O")
		return
	}

	// Request PTY with default terminal size.
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", currentRows, currentCols, modes); err != nil {
		s.logger.Error("Failed to request PTY", zap.Error(err))
		recorder.close("pty_failed")
		s.sendWebShellError(conn, fmt.Sprintf("Failed to request PTY: %v", err))
		return
	}

	// Start shell.
	if err := session.Shell(); err != nil {
		s.logger.Error("Failed to start shell", zap.Error(err))
		recorder.close("shell_failed")
		s.sendWebShellError(conn, fmt.Sprintf("Failed to start shell: %v", err))
		return
	}

	// Update session status to active.
	_ = s.db.Model(&models.WebShellSession{}).
		Where("session_id = ?", recorder.sessionID).
		Update("status", "active").Error

	s.logger.Info("WebShell session established",
		zap.String("session_id", recorder.sessionID),
		zap.String("user", req.User),
		zap.String("host", req.Host))

	// Send success message with session ID.
	if err := conn.WriteJSON(WebShellMessage{
		Type: "connected",
		Data: recorder.sessionID,
	}); err != nil {
		s.logger.Error("Failed to send connected message", zap.Error(err))
		return
	}

	// Record connection event.
	recorder.recordEvent("connected", fmt.Sprintf("Connected to %s@%s:%d", req.User, req.Host, req.Port), &currentCols, &currentRows)

	// Proxy between WebSocket and SSH session.
	var wg sync.WaitGroup
	wg.Add(3)

	// Read from SSH stdout and send to WebSocket.
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					s.logger.Error("Error reading stdout", zap.Error(err))
				}
				return
			}
			if n > 0 {
				data := string(buf[:n])
				msg := WebShellMessage{
					Type: "output",
					Data: data,
				}
				if err := conn.WriteJSON(msg); err != nil {
					s.logger.Error("Error sending output to WebSocket", zap.Error(err))
					return
				}

				// Record output event.
				recorder.recordEvent("output", data, nil, nil)
				recorder.mu.Lock()
				recorder.bytesRecv += int64(n)
				recorder.mu.Unlock()
			}
		}
	}()

	// Read from SSH stderr and send to WebSocket.
	go func() {
		defer wg.Done()
		buf := make([]byte, 32*1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if err != io.EOF {
					s.logger.Error("Error reading stderr", zap.Error(err))
				}
				return
			}
			if n > 0 {
				data := string(buf[:n])
				msg := WebShellMessage{
					Type: "output",
					Data: data,
				}
				if err := conn.WriteJSON(msg); err != nil {
					s.logger.Error("Error sending stderr to WebSocket", zap.Error(err))
					return
				}

				// Record output event.
				recorder.recordEvent("output", data, nil, nil)
				recorder.mu.Lock()
				recorder.bytesRecv += int64(n)
				recorder.mu.Unlock()
			}
		}
	}()

	// Read from WebSocket and send to SSH stdin.
	go func() {
		defer wg.Done()
		defer stdin.Close()
		for {
			var msg WebShellMessage
			if err := conn.ReadJSON(&msg); err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					s.logger.Error("Error reading from WebSocket", zap.Error(err))
				}
				return
			}

			switch msg.Type {
			case "input":
				data := []byte(msg.Data)
				if _, err := stdin.Write(data); err != nil {
					s.logger.Error("Error writing to stdin", zap.Error(err))
					return
				}

				// Record input event.
				recorder.recordEvent("input", msg.Data, nil, nil)
				recorder.mu.Lock()
				recorder.bytesSent += int64(len(data))
				recorder.mu.Unlock()

			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					if err := session.WindowChange(msg.Rows, msg.Cols); err != nil {
						s.logger.Error("Error resizing terminal", zap.Error(err))
					} else {
						currentCols = msg.Cols
						currentRows = msg.Rows

						// Record resize event.
						recorder.recordEvent("resize", "", &currentCols, &currentRows)
					}
				}

			case "ping":
				// Respond to ping to keep connection alive.
				if err := conn.WriteJSON(WebShellMessage{Type: "pong"}); err != nil {
					s.logger.Error("Error sending pong", zap.Error(err))
					return
				}
			}
		}
	}()

	wg.Wait()
	s.logger.Info("WebShell session closed",
		zap.String("session_id", recorder.sessionID),
		zap.String("user", req.User),
		zap.String("host", req.Host))
}

// sendWebShellError sends an error message to the WebSocket client.
func (s *Service) sendWebShellError(conn *websocket.Conn, message string) {
	msg := WebShellMessage{
		Type: "error",
		Data: message,
	}
	_ = conn.WriteJSON(msg)
}
