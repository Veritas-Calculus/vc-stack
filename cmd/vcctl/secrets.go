package main

import (
	"fmt"
	"os"

	"github.com/Veritas-Calculus/vc-stack/pkg/security"
	"github.com/spf13/cobra"
)

func newSecretsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "secrets",
		Short: "Manage encrypted secrets for VC Stack",
		Long: `Provides tools to generate master keys and encrypt/decrypt
sensitive strings (like database passwords) for configuration files.
This mimics CloudStack's secure credential management approach.`,
	}

	cmd.AddCommand(newSecretsInitCommand())
	cmd.AddCommand(newSecretsEncryptCommand())
	cmd.AddCommand(newSecretsDecryptCommand())

	return cmd
}

func newSecretsInitCommand() *cobra.Command {
	var keyPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a new master encryption key",
		RunE: func(cmd *cobra.Command, args []string) error {
			key, err := security.GenerateMasterKey()
			if err != nil {
				return fmt.Errorf("failed to generate master key: %w", err)
			}

			if keyPath == "" {
				// Output to stdout
				fmt.Println("Generated Master Key:")
				fmt.Println(key)
				fmt.Println("\nTo use this key, either:")
				fmt.Printf("1. Save it to %s with 0400 permissions\n", security.DefaultMasterKeyPath)
				fmt.Println("2. Set the VC_MASTER_KEY environment variable")
				return nil
			}

			// Write to file with 0400 permissions
			if err := os.WriteFile(keyPath, []byte(key+"\n"), 0400); err != nil {
				return fmt.Errorf("failed to write key file: %w", err)
			}
			fmt.Printf("Master key successfully saved to %s\n", keyPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&keyPath, "file", "f", "", "Output file path (default prints to stdout)")

	return cmd
}

func newSecretsEncryptCommand() *cobra.Command {
	var keyPath string

	cmd := &cobra.Command{
		Use:   "encrypt [plaintext]",
		Short: "Encrypt a plaintext string using the master key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			plaintext := args[0]
			key, err := security.GetMasterKey(keyPath)
			if err != nil {
				return fmt.Errorf("failed to load master key: %w", err)
			}

			encrypted, err := security.Encrypt(plaintext, key)
			if err != nil {
				return fmt.Errorf("encryption failed: %w", err)
			}

			fmt.Println(encrypted)
			return nil
		},
	}
	cmd.Flags().StringVarP(&keyPath, "key-file", "k", "", "Path to master key file")

	return cmd
}

func newSecretsDecryptCommand() *cobra.Command {
	var keyPath string

	cmd := &cobra.Command{
		Use:   "decrypt [encrypted_string]",
		Short: "Decrypt an ENC(...) string using the master key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			encrypted := args[0]
			key, err := security.GetMasterKey(keyPath)
			if err != nil {
				return fmt.Errorf("failed to load master key: %w", err)
			}

			plaintext, err := security.Decrypt(encrypted, key)
			if err != nil {
				return fmt.Errorf("decryption failed: %w", err)
			}

			fmt.Println(plaintext)
			return nil
		},
	}
	cmd.Flags().StringVarP(&keyPath, "key-file", "k", "", "Path to master key file")

	return cmd
}
