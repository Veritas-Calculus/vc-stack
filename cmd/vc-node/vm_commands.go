package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/Veritas-Calculus/vc-stack/internal/node/compute"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var configDir string

func vmListCmd() *cobra.Command {
	var allFlag bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List virtual machines",
		RunE: func(cmd *cobra.Command, args []string) error {
			logger, _ := zap.NewProduction()
			defer logger.Sync()

			store, err := compute.NewQEMUConfigStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("init config store: %w", err)
			}

			configs, err := store.List()
			if err != nil {
				return fmt.Errorf("list VMs: %w", err)
			}

			var filtered []*compute.QEMUConfig
			if allFlag {
				filtered = configs
			} else {
				for _, cfg := range configs {
					if cfg.Status == "running" {
						filtered = append(filtered, cfg)
					}
				}
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tSTATUS\tVCPUs\tMEMORY\tPID\tCREATED")

			for _, cfg := range filtered {
				created := cfg.CreatedAt.Format("2006-01-02 15:04:05")
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%dM\t%d\t%s\n",
					cfg.ID[:12], cfg.Name, cfg.Status, cfg.VCPUs, cfg.MemoryMB, cfg.PID, created)
			}

			w.Flush()
			fmt.Printf("\nTotal: %d VMs\n", len(filtered))
			return nil
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Show all VMs including stopped")
	return cmd
}

func vmShowCmd() *cobra.Command {
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:   "show <vm-id>",
		Short: "Show VM details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmID := args[0]
			logger, _ := zap.NewProduction()
			defer logger.Sync()

			store, err := compute.NewQEMUConfigStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("init config store: %w", err)
			}

			config, err := store.Load(vmID)
			if err != nil {
				return fmt.Errorf("load VM: %w", err)
			}

			if jsonFlag {
				data, err := json.MarshalIndent(config, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal JSON: %w", err)
				}
				fmt.Println(string(data))
				return nil
			}

			fmt.Printf("VM Details:\n")
			fmt.Printf("  ID:           %s\n", config.ID)
			fmt.Printf("  Name:         %s\n", config.Name)
			fmt.Printf("  Status:       %s\n", config.Status)
			fmt.Printf("  Tenant ID:    %s\n", config.TenantID)
			fmt.Printf("  Project ID:   %s\n", config.ProjectID)
			fmt.Printf("\n")
			fmt.Printf("Resources:\n")
			fmt.Printf("  VCPUs:        %d\n", config.VCPUs)
			fmt.Printf("  Memory:       %d MB\n", config.MemoryMB)
			fmt.Printf("  Disk:         %d GB\n", config.DiskGB)
			fmt.Printf("\n")
			fmt.Printf("Storage:\n")
			fmt.Printf("  Image ID:     %s\n", config.ImageID)
			fmt.Printf("  Image Path:   %s\n", config.ImagePath)
			fmt.Printf("  Disk Path:    %s\n", config.DiskPath)
			fmt.Printf("\n")
			fmt.Printf("Process:\n")
			fmt.Printf("  PID:          %d\n", config.PID)
			fmt.Printf("  Socket:       %s\n", config.SocketPath)
			fmt.Printf("  VNC Port:     %d\n", config.VNCPort)
			fmt.Printf("\n")
			fmt.Printf("Timestamps:\n")
			fmt.Printf("  Created:      %s\n", config.CreatedAt.Format(time.RFC3339))
			fmt.Printf("  Updated:      %s\n", config.UpdatedAt.Format(time.RFC3339))
			if !config.StartedAt.IsZero() {
				fmt.Printf("  Started:      %s\n", config.StartedAt.Format(time.RFC3339))
			}
			fmt.Printf("\n")

			if len(config.Networks) > 0 {
				fmt.Printf("Networks:\n")
				for i, net := range config.Networks {
					fmt.Printf("  [%d] Network ID: %s\n", i, net.NetworkID)
					fmt.Printf("      Port ID:    %s\n", net.PortID)
					fmt.Printf("      MAC:        %s\n", net.MAC)
					fmt.Printf("      IP:         %s\n", net.IP)
					fmt.Printf("      Interface:  %s\n", net.Interface)
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output as JSON")
	return cmd
}

func vmStopCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "stop <vm-id>",
		Short: "Stop a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmID := args[0]
			logger, _ := zap.NewProduction()
			defer logger.Sync()

			managerConfig := compute.QEMUManagerConfig{
				ConfigDir: configDir,
			}

			manager, err := compute.NewQEMUManager(managerConfig, logger)
			if err != nil {
				return fmt.Errorf("init manager: %w", err)
			}

			ctx := context.Background()
			if err := manager.StopVM(ctx, vmID, force); err != nil {
				return fmt.Errorf("stop VM: %w", err)
			}

			fmt.Printf("VM %s stopped successfully\n", vmID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force stop (SIGKILL)")
	return cmd
}

func vmDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <vm-id>",
		Short: "Delete a VM",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmID := args[0]
			logger, _ := zap.NewProduction()
			defer logger.Sync()

			if !force {
				fmt.Printf("Are you sure you want to delete VM %s? (yes/no): ", vmID)
				var response string
				fmt.Scanln(&response)
				if response != "yes" {
					fmt.Println("Cancelled")
					return nil
				}
			}

			managerConfig := compute.QEMUManagerConfig{
				ConfigDir: configDir,
			}

			manager, err := compute.NewQEMUManager(managerConfig, logger)
			if err != nil {
				return fmt.Errorf("init manager: %w", err)
			}

			ctx := context.Background()
			if err := manager.DeleteVM(ctx, vmID); err != nil {
				return fmt.Errorf("delete VM: %w", err)
			}

			fmt.Printf("VM %s deleted successfully\n", vmID)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete without confirmation")
	return cmd
}

func vmConsoleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "console <vm-id>",
		Short: "Show VM console info",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			vmID := args[0]
			logger, _ := zap.NewProduction()
			defer logger.Sync()

			store, err := compute.NewQEMUConfigStore(configDir, logger)
			if err != nil {
				return fmt.Errorf("init config store: %w", err)
			}

			config, err := store.Load(vmID)
			if err != nil {
				return fmt.Errorf("load VM: %w", err)
			}

			if config.Status != "running" {
				return fmt.Errorf("VM is not running (status: %s)", config.Status)
			}

			fmt.Printf("VM Console Information:\n")
			fmt.Printf("  VNC Port:     %d\n", config.VNCPort)
			fmt.Printf("  VNC URL:      vnc://localhost:%d\n", config.VNCPort)
			fmt.Printf("  QMP Socket:   %s\n", config.SocketPath)
			fmt.Printf("\nUse a VNC client to connect to the console.\n")
			fmt.Printf("Example: vncviewer localhost:%d\n", config.VNCPort)

			return nil
		},
	}
}
