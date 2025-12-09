//go:build !ovn_sdk && !ovn_libovsdb

package network

import (
	"os"
	"os/exec"
)

// runNetCommand executes a network command with sudo if not running as root
// This is shared between namespace and localport implementations
func runNetCommand(name string, args ...string) error {
	// Check if running as root
	if os.Geteuid() == 0 {
		return exec.Command(name, args...).Run()
	}
	// Not root, use sudo
	cmdArgs := append([]string{name}, args...)
	return exec.Command("sudo", cmdArgs...).Run()
}
