package qemu

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GenerateCloudInitISO generates a cloud-init ISO file.
// It creates meta-data and user-data files, then uses genisoimage/mkisofs to create the ISO.
func GenerateCloudInitISO(dir string, id, hostname, userData string, sshKeys []string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create dir: %w", err)
	}

	// Generate meta-data.
	metaData := fmt.Sprintf("instance-id: %s\nlocal-hostname: %s\n", id, hostname)
	metaPath := filepath.Join(dir, "meta-data")
	if err := os.WriteFile(metaPath, []byte(metaData), 0o644); err != nil {
		return "", fmt.Errorf("write meta-data: %w", err)
	}
	defer os.Remove(metaPath)

	// Generate user-data.
	if userData == "" {
		userData = "#cloud-config\n"
	}
	if !strings.HasPrefix(userData, "#cloud-config") {
		// If not a full cloud-config, assume it's just script or empty, prepend header if needed.
		if len(sshKeys) > 0 {
			userData = "#cloud-config\n"
		}
	}

	if len(sshKeys) > 0 {
		// Append ssh keys if not already present.
		if userData == "#cloud-config\n" {
			userData += "ssh_authorized_keys:\n"
			for _, key := range sshKeys {
				userData += fmt.Sprintf("  - %s\n", key)
			}
		} else {
			if !strings.Contains(userData, "ssh_authorized_keys:") {
				userData += "\nssh_authorized_keys:\n"
				for _, key := range sshKeys {
					userData += fmt.Sprintf("  - %s\n", key)
				}
			}
		}
	}

	userPath := filepath.Join(dir, "user-data")
	if err := os.WriteFile(userPath, []byte(userData), 0o644); err != nil {
		return "", fmt.Errorf("write user-data: %w", err)
	}
	defer os.Remove(userPath)

	// ISO path.
	isoPath := filepath.Join(dir, id+"-cidata.iso")

	// Try genisoimage first, then mkisofs.
	// -output <iso> -volid cidata -joliet -rock <user-data> <meta-data>
	cmd := exec.Command("genisoimage", "-output", isoPath, "-volid", "cidata", "-joliet", "-rock", userPath, metaPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		// Try mkisofs
		cmd = exec.Command("mkisofs", "-output", isoPath, "-volid", "cidata", "-joliet", "-rock", userPath, metaPath)
		if output2, err2 := cmd.CombinedOutput(); err2 != nil {
			return "", fmt.Errorf("failed to create ISO: genisoimage: %v (%s), mkisofs: %v (%s)", err, string(output), err2, string(output2))
		}
	}

	return isoPath, nil
}
