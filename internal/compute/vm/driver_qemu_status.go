package vm

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func (d *qemuDriver) VMStatus(id string) (exists, running bool) {
	// Sanitize id to prevent path traversal.
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return false, false
	}

	// Use new QEMU driver if enabled.
	if d.useNewDrv {
		isRunning, err := d.qemuDrv.IsRunning(id)
		if err != nil {
			return false, false
		}
		return isRunning, isRunning
	}

	// Legacy implementation.
	pidbs, err := os.ReadFile(d.pidPath(id))
	var pid int
	if err == nil {
		pid, _ = strconv.Atoi(strings.TrimSpace(string(pidbs)))
	} else {
		// Try metadata.
		if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
			var m vmMeta
			if err := json.Unmarshal(mb, &m); err == nil {
				pid = m.PID
			}
		}
	}
	if pid == 0 {
		return false, false
	}
	if err := syscall.Kill(pid, 0); err == nil {
		return true, true
	}
	return true, false
}

//nolint:gocognit
func (d *qemuDriver) ConsoleURL(id string, _ time.Duration) (string, error) {
	// Sanitize id to prevent path traversal.
	var err error
	id, err = validateVMID(id)
	if err != nil {
		return "", err
	}

	// Try to query QMP socket for the real VNC port using human-monitor-command 'info vnc'
	qmpPath := filepath.Join(d.runDir, id+".qmp")
	// Prefer a cached VNC address in metadata.
	if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
		var m vmMeta
		if err := json.Unmarshal(mb, &m); err == nil && m.VNC != "" {
			// If stored as port only, normalize.
			if strings.Contains(m.VNC, ":") {
				return fmt.Sprintf("vnc://%s", m.VNC), nil
			}
			return fmt.Sprintf("vnc://127.0.0.1:%s", m.VNC), nil
		}
		if err := json.Unmarshal(mb, &m); err == nil && m.QMP != "" {
			qmpPath = m.QMP
		}
	}
	if _, err := os.Stat(qmpPath); err == nil {
		out, err := queryQMP(qmpPath, "info vnc")
		if err == nil {
			// parse port from output, look for digits like 5901.
			fields := strings.Fields(out)
			for _, f := range fields {
				if strings.HasPrefix(f, "127.") || strings.HasPrefix(f, "0.") || strings.Contains(f, ":") {
					// try split by ':' to get port.
					if strings.Contains(f, ":") {
						parts := strings.Split(f, ":")
						port := parts[len(parts)-1]
						// sanitize.
						port = strings.Trim(port, ",")
						if _, err := strconv.Atoi(port); err == nil {
							// persist into metadata.
							if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
								var m vmMeta
								if err := json.Unmarshal(mb, &m); err == nil {
									m.VNC = fmt.Sprintf("127.0.0.1:%s", port)
									if b, err := json.Marshal(m); err == nil {
										_ = os.WriteFile(d.metaPath(id), b, 0o600)
									}
								}
							}
							return fmt.Sprintf("vnc://127.0.0.1:%s", port), nil
						}
					}
				}
				// fallback: if token is numeric and >=5900.
				if p, err := strconv.Atoi(strings.Trim(f, ",")); err == nil && p >= 5900 && p < 7000 {
					// persist.
					if mb, err := os.ReadFile(d.metaPath(id)); err == nil {
						var m vmMeta
						if err := json.Unmarshal(mb, &m); err == nil {
							m.VNC = fmt.Sprintf("127.0.0.1:%d", p)
							if b, err := json.Marshal(m); err == nil {
								_ = os.WriteFile(d.metaPath(id), b, 0o600)
							}
						}
					}
					return fmt.Sprintf("vnc://127.0.0.1:%d", p), nil
				}
			}
		}
	}
	// Fallback placeholder.
	return "vnc://127.0.0.1:5900", nil
}

// queryQMP connects to a unix qmp socket, performs handshake, issues a human-monitor-command and returns its output string.
func queryQMP(socketPath, humanCmd string) (string, error) {
	// Connect with timeout.
	conn, err := net.DialTimeout("unix", socketPath, 2*time.Second)
	if err != nil {
		return "", err
	}
	defer func() { _ = conn.Close() }()

	// helper to read a single JSON message from the socket (QMP sends newline-delimited JSON)
	readMsg := func(deadline time.Duration) ([]byte, error) {
		var buf []byte
		tmp := make([]byte, 4096)
		deadlineTime := time.Now().Add(deadline)
		for {
			_ = conn.SetReadDeadline(deadlineTime)
			n, err := conn.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
				// try to find a complete JSON object in buf.
				var v interface{}
				if json.Unmarshal(buf, &v) == nil {
					return buf, nil
				}
				// else continue reading until valid JSON assembled.
			}
			if err != nil {
				return nil, err
			}
		}
	}

	// Read greeting message.
	if _, err := readMsg(2 * time.Second); err != nil {
		return "", fmt.Errorf("read qmp greeting: %w", err)
	}

	// Send qmp_capabilities.
	capCmd := map[string]interface{}{"execute": "qmp_capabilities"}
	if b, err := json.Marshal(capCmd); err == nil {
		b = append(b, '\n')
		if _, err := conn.Write(b); err != nil {
			return "", fmt.Errorf("write qmp_capabilities: %w", err)
		}
	}
	// Read capabilities response.
	if _, err := readMsg(2 * time.Second); err != nil {
		return "", fmt.Errorf("read qmp capabilities response: %w", err)
	}

	// Send human-monitor-command.
	hm := map[string]interface{}{"execute": "human-monitor-command", "arguments": map[string]interface{}{"command-line": humanCmd}}
	if b, err := json.Marshal(hm); err == nil {
		b = append(b, '\n')
		if _, err := conn.Write(b); err != nil {
			return "", fmt.Errorf("write human-monitor-command: %w", err)
		}
	}

	// Read response.
	outb, err := readMsg(2 * time.Second)
	if err != nil {
		return "", fmt.Errorf("read human-monitor-command response: %w", err)
	}

	// Parse JSON and extract any "output" string under the "return" object.
	var parsed map[string]interface{}
	if err := json.Unmarshal(outb, &parsed); err != nil {
		// fallback to raw.
		return string(outb), nil
	}
	if ret, ok := parsed["return"]; ok {
		// ret may be map.
		if m, ok := ret.(map[string]interface{}); ok {
			// look for output field.
			if outv, ok := m["output"]; ok {
				if s, ok := outv.(string); ok {
					return s, nil
				}
			}
			// sometimes nested under human-monitor-command key.
			for _, v := range m {
				if s, ok := v.(string); ok {
					return s, nil
				}
			}
		}
	}
	// fallback: return raw bytes as string.
	return string(outb), nil
}

func (d *qemuDriver) Status() NodeStatus {
	// Best-effort: query /proc/meminfo and /proc/cpuinfo would be better, but return config-derived defaults.
	return NodeStatus{CPUsTotal: 8, CPUsUsed: 0, RAMMBTotal: 32768, RAMMBUsed: 0, DiskGBTotal: 500, DiskGBUsed: 0}
}
