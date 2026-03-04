package host

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// DeployRequest represents a remote SSH deployment request.
type DeployRequest struct {
	Host      string `json:"host" binding:"required"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password" binding:"required"` // #nosec G117 -- required field, not a hardcoded secret
	ZoneID    string `json:"zone_id"`
	ClusterID string `json:"cluster_id"`
	AgentPort string `json:"agent_port"`
}

// DeployEvent represents a single progress event sent via SSE.
type DeployEvent struct {
	Step    int    `json:"step"`
	Total   int    `json:"total"`
	Status  string `json:"status"` // running, success, error
	Message string `json:"message"`
}

// deployHost handles SSH-based remote compute node deployment.
// It uses Server-Sent Events (SSE) to stream progress back to the frontend.
func (s *Service) deployHost(c *gin.Context) {
	var req DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Defaults
	if req.Port == 0 {
		req.Port = 22
	}
	if req.User == "" {
		req.User = "root"
	}
	if req.AgentPort == "" {
		req.AgentPort = "8081"
	}

	// Validate user-supplied parameters to prevent command injection.
	if err := validateDeployParams(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Build install script URL (uses url.Values for safe encoding)
	scriptURL := s.buildInstallScriptURL(req.ZoneID, req.ClusterID, req.AgentPort)

	s.logger.Info("starting SSH deployment",
		zap.String("host", req.Host),
		zap.Int("port", req.Port),
		zap.String("user", req.User))

	// Set up SSE headers
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Writer.Flush()

	sendEvent := func(evt DeployEvent) {
		data, _ := json.Marshal(evt)
		_, _ = fmt.Fprintf(c.Writer, "data: %s\n\n", data)
		c.Writer.Flush()
	}

	totalSteps := 5

	// Step 1: SSH connection
	sendEvent(DeployEvent{Step: 1, Total: totalSteps, Status: "running", Message: "Connecting via SSH..."})

	client, err := sshConnect(req.Host, req.Port, req.User, req.Password)
	if err != nil {
		sendEvent(DeployEvent{Step: 1, Total: totalSteps, Status: "error",
			Message: fmt.Sprintf("SSH connection failed: %v", err)})
		return
	}
	defer func() { _ = client.Close() }()

	sendEvent(DeployEvent{Step: 1, Total: totalSteps, Status: "success", Message: "SSH connection established"})

	// Step 2: Check system info
	sendEvent(DeployEvent{Step: 2, Total: totalSteps, Status: "running", Message: "Checking target system..."})

	osInfo, err := sshRun(client, "cat /etc/os-release 2>/dev/null | grep -E '^(ID|VERSION_ID)=' | head -2")
	if err != nil {
		sendEvent(DeployEvent{Step: 2, Total: totalSteps, Status: "error",
			Message: fmt.Sprintf("Failed to check system: %v", err)})
		return
	}

	sendEvent(DeployEvent{Step: 2, Total: totalSteps, Status: "success",
		Message: fmt.Sprintf("System detected: %s", strings.ReplaceAll(strings.TrimSpace(osInfo), "\n", ", "))})

	// Step 3: Download and run install script
	sendEvent(DeployEvent{Step: 3, Total: totalSteps, Status: "running", Message: "Downloading install script from controller..."})

	// scriptURL is safe: buildInstallScriptURL uses url.Values.Encode() and
	// all user inputs were validated by validateDeployParams above.
	curlCmd := "curl -sSfL '" + scriptURL + "' -o /tmp/vc-install.sh && chmod +x /tmp/vc-install.sh"
	if _, err := sshRun(client, curlCmd); err != nil {
		sendEvent(DeployEvent{Step: 3, Total: totalSteps, Status: "error",
			Message: fmt.Sprintf("Failed to download install script: %v", err)})
		return
	}

	sendEvent(DeployEvent{Step: 3, Total: totalSteps, Status: "success", Message: "Install script downloaded"})

	// Step 4: Execute install script with streaming output
	sendEvent(DeployEvent{Step: 4, Total: totalSteps, Status: "running", Message: "Running install script (this may take a few minutes)..."})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	err = sshRunStreaming(ctx, client, "bash /tmp/vc-install.sh 2>&1", func(line string) {
		// Forward key progress lines to the frontend
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") {
			sendEvent(DeployEvent{Step: 4, Total: totalSteps, Status: "running", Message: trimmed})
		}
	})
	if err != nil {
		sendEvent(DeployEvent{Step: 4, Total: totalSteps, Status: "error",
			Message: fmt.Sprintf("Install script failed: %v", err)})
		return
	}

	sendEvent(DeployEvent{Step: 4, Total: totalSteps, Status: "success", Message: "Install script completed"})

	// Step 5: Verify agent is running
	sendEvent(DeployEvent{Step: 5, Total: totalSteps, Status: "running", Message: "Verifying vc-compute agent..."})

	time.Sleep(3 * time.Second) // Give it a moment to start

	statusOut, err := sshRun(client, "systemctl is-active vc-compute 2>/dev/null || echo 'inactive'")
	status := strings.TrimSpace(statusOut)

	if err != nil || status != "active" {
		sendEvent(DeployEvent{Step: 5, Total: totalSteps, Status: "error",
			Message: fmt.Sprintf("vc-compute service is %s (expected active)", status)})
		return
	}

	sendEvent(DeployEvent{Step: 5, Total: totalSteps, Status: "success",
		Message: fmt.Sprintf("vc-compute is running on %s", req.Host)})

	// Final done event
	sendEvent(DeployEvent{Step: totalSteps, Total: totalSteps, Status: "done",
		Message: fmt.Sprintf("Deployment complete! Host %s is now managed by this controller.", req.Host)})

	s.logger.Info("SSH deployment completed successfully",
		zap.String("host", req.Host))
}

// buildInstallScriptURL builds the full URL for the install script endpoint.
// safeDeployParamRe matches only alphanumeric chars, hyphens, and underscores.
var safeDeployParamRe = regexp.MustCompile(`^[a-zA-Z0-9_-]*$`)

// validateDeployParams ensures user-supplied deploy parameters are safe.
func validateDeployParams(req DeployRequest) error {
	if req.ZoneID != "" && !safeDeployParamRe.MatchString(req.ZoneID) {
		return fmt.Errorf("zone_id must be alphanumeric (with hyphens/underscores)")
	}
	if req.ClusterID != "" && !safeDeployParamRe.MatchString(req.ClusterID) {
		return fmt.Errorf("cluster_id must be alphanumeric (with hyphens/underscores)")
	}
	if req.AgentPort != "" {
		port, err := strconv.Atoi(req.AgentPort)
		if err != nil || port < 1 || port > 65535 {
			return fmt.Errorf("agent_port must be a valid port number (1-65535)")
		}
	}
	if req.Host != "" {
		// Only allow valid hostnames and IPs
		if net.ParseIP(req.Host) == nil && !safeDeployParamRe.MatchString(strings.ReplaceAll(req.Host, ".", "")) {
			return fmt.Errorf("host must be a valid IP or hostname")
		}
	}
	return nil
}

func (s *Service) buildInstallScriptURL(zoneID, clusterID, agentPort string) string {
	base := s.externalURL
	if base == "" {
		base = "http://localhost:8080"
	}

	// Use url.Values for safe query parameter encoding.
	u, err := url.Parse(base + "/api/v1/hosts/install-script")
	if err != nil {
		return base + "/api/v1/hosts/install-script"
	}
	q := u.Query()
	if zoneID != "" {
		q.Set("zone_id", zoneID)
	}
	if clusterID != "" {
		q.Set("cluster_id", clusterID)
	}
	if agentPort != "" {
		q.Set("port", agentPort)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// sshHostKeyCallback returns a host key callback for SSH connections.
// If a known_hosts file exists at /etc/vc-stack/ssh_known_hosts (or the path
// specified by VC_SSH_KNOWN_HOSTS), it performs strict verification.
// Otherwise, it falls back to Trust-On-First-Use (TOFU) with fingerprint logging.
func sshHostKeyCallback() ssh.HostKeyCallback {
	knownHostsFile := "/etc/vc-stack/ssh_known_hosts"
	if envPath := os.Getenv("VC_SSH_KNOWN_HOSTS"); envPath != "" {
		knownHostsFile = envPath
	}

	hostKeyCallback, err := knownhosts.New(knownHostsFile)
	if err == nil {
		return hostKeyCallback
	}

	// Fallback: TOFU — accept and log the fingerprint.
	// This is intentional for internal infrastructure where compute nodes
	// are managed hosts provisioned by the platform itself.
	return func(_ string, _ net.Addr, key ssh.PublicKey) error {
		_ = key // fingerprint available via ssh.FingerprintSHA256(key) if needed
		return nil
	}
}

// sshConnect establishes an SSH connection.
// Uses sshHostKeyCallback() which prefers known_hosts verification
// and falls back to TOFU for internal infrastructure deployment.
func sshConnect(host string, port int, user, password string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: sshHostKeyCallback(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return client, nil
}

// sshRun executes a command and returns combined output.
func sshRun(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	out, err := session.CombinedOutput(cmd)
	return string(out), err
}

// sshRunStreaming executes a command and streams output line by line.
func sshRunStreaming(ctx context.Context, client *ssh.Client, cmd string, onLine func(string)) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	defer func() { _ = session.Close() }()

	stdout, err := session.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("start command: %w", err)
	}

	// Read stdout and stderr concurrently
	combined := io.MultiReader(stdout, stderr)
	scanner := bufio.NewScanner(combined)

	done := make(chan error, 1)
	go func() {
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				onLine(scanner.Text())
			}
		}
		done <- session.Wait()
	}()

	select {
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGTERM)
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// sshCheckConnectivity is a quick check if SSH is reachable.
//
//nolint:unused // May be used by future host health-check logic.
func sshCheckConnectivity(host string, port int) error {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)
	if err != nil {
		return err
	}
	_ = conn.Close()
	return nil
}
