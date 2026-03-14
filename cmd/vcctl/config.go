package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// CLIConfig represents the full vcctl configuration file.
type CLIConfig struct {
	// ActiveProfile is the currently selected profile name.
	ActiveProfile string `yaml:"active_profile"`
	// Profiles maps profile names to their settings.
	Profiles map[string]Profile `yaml:"profiles"`
}

// Profile represents a single named configuration profile.
type Profile struct {
	Endpoint string `yaml:"endpoint"`
	Output   string `yaml:"output,omitempty"`   // table | json | yaml
	Token    string `yaml:"token,omitempty"`    // cached auth token
	Project  string `yaml:"project,omitempty"`  // default project ID
	Insecure bool   `yaml:"insecure,omitempty"` // skip TLS verification
}

// configDir returns the path to ~/.vcctl/.
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".vcctl"
	}
	return filepath.Join(home, ".vcctl")
}

// configFile returns the path to ~/.vcctl/config.yaml.
func configFile() string {
	return filepath.Join(configDir(), "config.yaml")
}

// loadConfig reads the config file from disk.
func loadConfig() (*CLIConfig, error) {
	data, err := os.ReadFile(configFile())
	if err != nil {
		if os.IsNotExist(err) {
			return &CLIConfig{
				ActiveProfile: "default",
				Profiles:      map[string]Profile{},
			}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg CLIConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	return &cfg, nil
}

// saveConfig writes the config file to disk.
func saveConfig(cfg *CLIConfig) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configFile(), data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

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
		Short: "Initialize configuration with a default profile",
		RunE:  runConfigInit,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List configuration profiles",
		RunE:  runConfigList,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value in the active profile",
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <key>",
		Short: "Get a configuration value from the active profile",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigGet,
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "use-profile <name>",
		Short: "Switch the active configuration profile",
		Args:  cobra.ExactArgs(1),
		RunE:  runConfigUseProfile,
	})

	return cmd
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	path := configFile()
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Configuration already exists at %s\n", path)
		fmt.Println("Use 'vcctl config set <key> <value>' to modify settings.")
		return nil
	}

	cfg := &CLIConfig{
		ActiveProfile: "default",
		Profiles: map[string]Profile{
			"default": {
				Endpoint: "http://127.0.0.1:8080",
				Output:   "table",
			},
		},
	}

	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Configuration initialized at %s\n", path)
	fmt.Println()
	fmt.Println("Active profile: default")
	fmt.Println("  endpoint = http://127.0.0.1:8080")
	fmt.Println("  output   = table")
	fmt.Println()
	fmt.Println("Run 'vcctl config set endpoint <url>' to configure your API endpoint.")
	return nil
}

func runConfigList(_ *cobra.Command, _ []string) error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Run 'vcctl config init' first.")
		return nil
	}

	// Sort profile names for stable output
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	tw := newTabWriter()
	fmt.Fprintln(tw, "PROFILE\tENDPOINT\tOUTPUT\tACTIVE")
	for _, name := range names {
		p := cfg.Profiles[name]
		active := ""
		if name == cfg.ActiveProfile {
			active = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", name, p.Endpoint, p.Output, active)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	profile := cfg.ActiveProfile
	p := cfg.Profiles[profile]

	switch key {
	case "endpoint":
		p.Endpoint = value
	case "output":
		if value != "table" && value != "json" && value != "yaml" {
			return fmt.Errorf("invalid output format %q: must be table, json, or yaml", value)
		}
		p.Output = value
	case "token":
		p.Token = value
	case "project":
		p.Project = value
	case "insecure":
		p.Insecure = value == "true" || value == "1"
	default:
		return fmt.Errorf("unknown config key %q (valid: endpoint, output, token, project, insecure)", key)
	}

	cfg.Profiles[profile] = p
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("[%s] %s = %s\n", profile, key, value)
	return nil
}

func runConfigGet(_ *cobra.Command, args []string) error {
	key := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	profile := cfg.ActiveProfile
	p, ok := cfg.Profiles[profile]
	if !ok {
		return fmt.Errorf("active profile %q not found — run 'vcctl config init'", profile)
	}

	switch key {
	case "endpoint":
		fmt.Println(p.Endpoint)
	case "output":
		fmt.Println(p.Output)
	case "token":
		if p.Token == "" {
			fmt.Println("(not set)")
		} else {
			fmt.Println(p.Token[:8] + "...")
		}
	case "project":
		fmt.Println(p.Project)
	case "insecure":
		fmt.Println(p.Insecure)
	default:
		return fmt.Errorf("unknown config key %q (valid: endpoint, output, token, project, insecure)", key)
	}
	return nil
}

func runConfigUseProfile(_ *cobra.Command, args []string) error {
	name := args[0]

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	if _, ok := cfg.Profiles[name]; !ok {
		// Create the profile if it doesn't exist
		cfg.Profiles[name] = Profile{
			Endpoint: "http://127.0.0.1:8080",
			Output:   "table",
		}
		fmt.Printf("Created new profile %q\n", name)
	}

	cfg.ActiveProfile = name
	if err := saveConfig(cfg); err != nil {
		return err
	}

	fmt.Printf("Switched to profile %q\n", name)
	return nil
}
