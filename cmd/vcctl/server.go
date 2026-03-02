package main

import (
	"fmt"

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

	cmd.AddCommand(newServerManagementCommand())
	cmd.AddCommand(newServerComputeCommand())

	return cmd
}

// newServerManagementCommand creates the management server command.
func newServerManagementCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "management",
		Short: "Start the management plane server",
		Long:  `Start the VC Stack management plane server (includes gateway, identity, network, scheduler)`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting VC Stack Management...")
			fmt.Println("TODO: Launch vc-management server")
			fmt.Println("Hint: Use 'vc-management' binary directly or implement server start logic here")
		},
	}
}

// newServerComputeCommand creates the compute server command.
func newServerComputeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "compute",
		Short: "Start the compute node server",
		Long:  `Start the VC Stack compute node server (includes VM management, network agent, storage agent)`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Starting VC Stack Compute...")
			fmt.Println("TODO: Launch vc-compute server")
			fmt.Println("Hint: Use 'vc-compute' binary directly or implement server start logic here")
		},
	}
}
