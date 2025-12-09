// Package main implements the VC Stack CLI - a unified command-line interface.
// for managing VC Stack infrastructure, inspired by AWS CLI.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information, set via ldflags during build.
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"

	// Global flags.
	apiEndpoint string
	output      string
	profile     string
	debug       bool
)

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// newRootCommand creates the root command for vcctl.
func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vcctl",
		Short: "VC Stack CLI - Manage your cloud infrastructure",
		Long: `vcctl is a unified command-line interface for VC Stack.
It provides commands to manage compute instances, networks, storage,
and other cloud resources.

Examples:
  # List all instances
  vcctl compute instances list

  # Create a new instance
  vcctl compute instances create --name my-vm --vcpus 2 --memory 4096

  # List networks
  vcctl network list

  # Show cluster nodes
  vcctl cluster nodes list

For more information, visit: https://github.com/Veritas-Calculus/vc-stack`,
		SilenceUsage: true,
	}

	// Global flags.
	cmd.PersistentFlags().StringVar(&apiEndpoint, "endpoint", "", "API endpoint URL (default from config or env)")
	cmd.PersistentFlags().StringVarP(&output, "output", "o", "table", "Output format: table, json, yaml")
	cmd.PersistentFlags().StringVar(&profile, "profile", "default", "Configuration profile to use")
	cmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	// Add subcommands.
	cmd.AddCommand(newComputeCommand())
	cmd.AddCommand(newNetworkCommand())
	cmd.AddCommand(newStorageCommand())
	cmd.AddCommand(newClusterCommand())
	cmd.AddCommand(newIdentityCommand())
	cmd.AddCommand(newServerCommand())
	cmd.AddCommand(newVersionCommand())
	cmd.AddCommand(newConfigCommand())

	return cmd
}

// newVersionCommand creates the version command.
func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("vcctl version %s\n", Version)
			fmt.Printf("Commit: %s\n", Commit)
			fmt.Printf("Build Time: %s\n", BuildTime)
		},
	}
}
