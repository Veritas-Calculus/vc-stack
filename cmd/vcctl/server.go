package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

// newServerCommand creates the server management command.
// This command starts the actual VC Stack server components.
func newServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Manage VC Stack server components",
		Long:  `Start and manage VC Stack server components (management, compute)`,
	}

	mgmt := newServerManagementCommand()
	mgmt.Flags().StringP("config", "c", "/etc/vc-stack/management.yaml", "Path to management config file")
	cmd.AddCommand(mgmt)

	comp := newServerComputeCommand()
	comp.Flags().StringP("config", "c", "/etc/vc-stack/compute.yaml", "Path to compute node config file")
	cmd.AddCommand(comp)

	return cmd
}

// newServerManagementCommand creates the management server command.
func newServerManagementCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "management",
		Short: "Start the management plane server",
		Long:  `Start the VC Stack management plane server (includes gateway, identity, network, scheduler)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			return launchBinary("vc-management", configPath)
		},
	}
}

// newServerComputeCommand creates the compute server command.
func newServerComputeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "compute",
		Short: "Start the compute node server",
		Long:  `Start the VC Stack compute node server (includes VM management, network agent, storage agent)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, _ := cmd.Flags().GetString("config")
			return launchBinary("vc-compute", configPath)
		},
	}
}

// launchBinary finds and exec's the given VC Stack binary.
func launchBinary(name, configPath string) error {
	// 1. Look in the same directory as vcctl
	selfPath, _ := os.Executable()
	selfDir := filepath.Dir(selfPath)
	candidate := filepath.Join(selfDir, name)

	// 2. Fall back to PATH lookup
	binPath, err := exec.LookPath(candidate)
	if err != nil {
		binPath, err = exec.LookPath(name)
		if err != nil {
			return fmt.Errorf("cannot find %q binary — ensure it is installed or in the same directory as vcctl", name)
		}
	}

	fmt.Printf("Starting %s (binary: %s)\n", name, binPath)
	if configPath != "" {
		fmt.Printf("Config: %s\n", configPath)
	}

	// Build args: pass --config if provided
	execArgs := []string{name}
	if configPath != "" {
		execArgs = append(execArgs, "--config", configPath)
	}

	// Replace this process with the target binary (exec)
	return syscall.Exec(binPath, execArgs, os.Environ()) // #nosec G204 -- binPath from exec.LookPath
}
