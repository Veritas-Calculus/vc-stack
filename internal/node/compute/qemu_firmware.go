// Package compute provides UEFI and TPM support utilities.
package compute

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

// prepareUEFIVars creates a copy of OVMF_VARS for the VM.
func (m *QEMUManager) prepareUEFIVars(varsPath string) error {
	// Copy OVMF_VARS template to instance directory.
	if err := copyFile(m.config.OVMFVarsPath, varsPath); err != nil {
		return fmt.Errorf("copy UEFI vars: %w", err)
	}

	m.logger.Debug("Prepared UEFI variables", zap.String("path", varsPath))
	return nil
}

// prepareTPM initializes TPM state directory and starts swtpm.
func (m *QEMUManager) prepareTPM(tpmDir string) error {
	// Create TPM state directory.
	if err := os.MkdirAll(tpmDir, 0750); err != nil {
		return fmt.Errorf("create TPM directory: %w", err)
	}

	// Initialize TPM state using swtpm_setup.
	setupCmd := exec.Command("swtpm_setup",
		"--tpm2",
		"--tpmstate", tpmDir,
		"--create-ek-cert",
		"--create-platform-cert",
		"--lock-nvram",
	)

	if output, err := setupCmd.CombinedOutput(); err != nil {
		m.logger.Warn("swtpm_setup failed",
			zap.Error(err),
			zap.String("output", string(output)))
		return fmt.Errorf("swtpm_setup: %w", err)
	}

	// Start swtpm in background.
	socketPath := filepath.Join(tpmDir, "swtpm-sock")
	swtpmCmd := exec.Command("swtpm",
		"socket",
		"--tpm2",
		"--tpmstate", fmt.Sprintf("dir=%s", tpmDir),
		"--ctrl", fmt.Sprintf("type=unixio,path=%s", socketPath),
		"--log", "level=20",
	)

	if err := swtpmCmd.Start(); err != nil {
		return fmt.Errorf("start swtpm: %w", err)
	}

	// Store swtpm PID for cleanup.
	pidPath := filepath.Join(tpmDir, "swtpm.pid")
	if err := os.WriteFile(pidPath, []byte(fmt.Sprintf("%d", swtpmCmd.Process.Pid)), 0644); err != nil {
		m.logger.Warn("Failed to write swtpm PID", zap.Error(err))
	}

	m.logger.Debug("Prepared TPM",
		zap.String("dir", tpmDir),
		zap.Int("pid", swtpmCmd.Process.Pid))

	return nil
}

// cleanupTPM stops swtpm process.
func (m *QEMUManager) cleanupTPM(tpmDir string) error {
	pidPath := filepath.Join(tpmDir, "swtpm.pid")

	data, err := os.ReadFile(pidPath)
	if err != nil {
		return nil // No PID file, nothing to clean up.
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return nil
	}

	// Kill swtpm process.
	process, err := os.FindProcess(pid)
	if err != nil {
		return nil
	}

	if err := process.Kill(); err != nil {
		m.logger.Warn("Failed to kill swtpm", zap.Int("pid", pid), zap.Error(err))
	}

	_ = os.Remove(pidPath)
	return nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	input, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	if err := os.WriteFile(dst, input, 0644); err != nil {
		return fmt.Errorf("write destination: %w", err)
	}

	return nil
}

// checkUEFISupport checks if UEFI firmware is available.
func (m *QEMUManager) checkUEFISupport() bool {
	if _, err := os.Stat(m.config.OVMFCodePath); os.IsNotExist(err) {
		return false
	}
	if _, err := os.Stat(m.config.OVMFVarsPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// checkTPMSupport checks if TPM (swtpm) is available.
func (m *QEMUManager) checkTPMSupport() bool {
	if _, err := exec.LookPath("swtpm"); err != nil {
		return false
	}
	if _, err := exec.LookPath("swtpm_setup"); err != nil {
		return false
	}
	return true
}

// GetCapabilities returns supported VM capabilities.
func (m *QEMUManager) GetCapabilities() map[string]bool {
	return map[string]bool{
		"kvm":         m.config.EnableKVM,
		"uefi":        m.checkUEFISupport(),
		"tpm":         m.checkTPMSupport(),
		"secure_boot": m.checkUEFISupport(), // Secure boot requires UEFI
	}
}
