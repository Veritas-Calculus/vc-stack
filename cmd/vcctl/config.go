package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newConfigCommand creates the configuration management command.
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"configure"},
		Short:   "Manage CLI configuration",
		Long:    `Manage vcctl configuration profiles and settings.`,
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize configuration",
		Run:   runConfigInit,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configuration profiles",
		Run:   runConfigList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		Run:   runConfigSet,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value",
		Args:  cobra.ExactArgs(1),
		Run:   runConfigGet,
	})

	return cmd
}

func runConfigInit(cmd *cobra.Command, args []string) {
	fmt.Println("Initializing vcctl configuration...")
	fmt.Println("TODO: Create ~/.vcctl/config")
	fmt.Println()
	fmt.Println("Configuration will be stored in: ~/.vcctl/config")
	fmt.Println()
	fmt.Println("Example configuration:")
	fmt.Println("  [default]")
	fmt.Println("  endpoint = http://localhost:8080")
	fmt.Println("  output = table")
	fmt.Println()
	fmt.Println("  [production]")
	fmt.Println("  endpoint = https://vcstack.example.com")
	fmt.Println("  output = json")
}

func runConfigList(cmd *cobra.Command, args []string) {
	fmt.Println("Configuration profiles:")
	fmt.Println("TODO: List profiles from ~/.vcctl/config")
}

func runConfigSet(cmd *cobra.Command, args []string) {
	key := args[0]
	value := args[1]
	fmt.Printf("Setting %s = %s\n", key, value)
	fmt.Println("TODO: Implement config set")
}

func runConfigGet(cmd *cobra.Command, args []string) {
	key := args[0]
	fmt.Printf("Getting %s\n", key)
	fmt.Println("TODO: Implement config get")
}
