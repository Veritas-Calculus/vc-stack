package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/spf13/cobra"

	dbmigrate "github.com/Veritas-Calculus/vc-stack/pkg/database/migrate"
)

// newDBCommand creates the `vcctl db` subcommand tree for database operations.
func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
		Long:  "Manage the VC Stack PostgreSQL database: migrations, backups, and restores.",
	}

	cmd.AddCommand(newDBMigrateCommand())
	cmd.AddCommand(newDBBackupCommand())
	cmd.AddCommand(newDBRestoreCommand())
	cmd.AddCommand(newDBVersionCommand())

	return cmd
}

// ── Migrate ────────────────────────────────────────────────────────────

func newDBMigrateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Run database migrations",
		Long:  "Apply pending SQL migrations from the migrations/ directory.",
	}

	var migrationsPath string
	var dsn string

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Apply all pending migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := dbmigrate.NewRunner(dbmigrate.Config{
				DatabaseDSN:    dsn,
				MigrationsPath: "file://" + migrationsPath,
			})
			if err != nil {
				return fmt.Errorf("failed to initialize migrations: %w", err)
			}
			defer runner.Close() //nolint:errcheck

			return runner.Up()
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Roll back the last migration",
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := dbmigrate.NewRunner(dbmigrate.Config{
				DatabaseDSN:    dsn,
				MigrationsPath: "file://" + migrationsPath,
			})
			if err != nil {
				return fmt.Errorf("failed to initialize migrations: %w", err)
			}
			defer runner.Close() //nolint:errcheck

			return runner.Down()
		},
	}

	cmd.PersistentFlags().StringVar(&migrationsPath, "path", "./migrations", "Path to migration files")
	cmd.PersistentFlags().StringVar(&dsn, "dsn", "", "PostgreSQL DSN (e.g., postgres://user:pass@host/db?sslmode=disable)")
	_ = cmd.MarkPersistentFlagRequired("dsn")

	cmd.AddCommand(upCmd, downCmd)
	return cmd
}

// ── Backup ─────────────────────────────────────────────────────────────

func newDBBackupCommand() *cobra.Command {
	var (
		host     string
		port     int
		dbName   string
		user     string
		outFile  string
		compress bool
	)

	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup the database using pg_dump",
		Long: `Create a PostgreSQL backup using pg_dump.

Requires pg_dump to be installed and accessible in PATH.
The DB password should be set via PGPASSWORD environment variable.

Examples:
  # Backup to a timestamped file
  PGPASSWORD=mypass vcctl db backup --host localhost --db vcstack --user vcstack

  # Backup to a specific file with compression
  PGPASSWORD=mypass vcctl db backup --host db.example.com --db vcstack --user vcstack --output /backups/vcstack.sql.gz --compress`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default output file with timestamp.
			if outFile == "" {
				ts := time.Now().Format("20060102-150405")
				ext := ".sql"
				if compress {
					ext = ".sql.gz"
				}
				outFile = fmt.Sprintf("vcstack-backup-%s%s", ts, ext)
			}

			pgArgs := []string{
				"-h", host,
				"-p", fmt.Sprintf("%d", port),
				"-U", user,
				"-d", dbName,
				"--format=plain",
				"--no-owner",
				"--no-privileges",
			}

			if compress {
				pgArgs = append(pgArgs, "--compress=9")
			}

			pgArgs = append(pgArgs, "-f", outFile)

			fmt.Printf("Backing up database '%s' on %s:%d to %s...\n", dbName, host, port, outFile)

			pgDump := exec.Command("pg_dump", pgArgs...) // #nosec G204
			pgDump.Stdout = os.Stdout
			pgDump.Stderr = os.Stderr

			if err := pgDump.Run(); err != nil {
				return fmt.Errorf("pg_dump failed: %w", err)
			}

			fmt.Printf("Backup completed: %s\n", outFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&port, "port", 5432, "Database port")
	cmd.Flags().StringVar(&dbName, "db", "vcstack", "Database name")
	cmd.Flags().StringVarP(&user, "user", "U", "vcstack", "Database user")
	cmd.Flags().StringVarP(&outFile, "output", "o", "", "Output file (default: vcstack-backup-TIMESTAMP.sql)")
	cmd.Flags().BoolVar(&compress, "compress", false, "Compress the backup with gzip")

	return cmd
}

// ── Restore ────────────────────────────────────────────────────────────

func newDBRestoreCommand() *cobra.Command {
	var (
		host   string
		port   int
		dbName string
		user   string
	)

	cmd := &cobra.Command{
		Use:   "restore [file]",
		Short: "Restore the database from a pg_dump backup",
		Long: `Restore a PostgreSQL database from a backup file created by 'vcctl db backup'.

Requires psql to be installed and accessible in PATH.
The DB password should be set via PGPASSWORD environment variable.

Examples:
  PGPASSWORD=mypass vcctl db restore vcstack-backup-20260307.sql`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			inputFile := args[0]

			if _, err := os.Stat(inputFile); os.IsNotExist(err) {
				return fmt.Errorf("backup file not found: %s", inputFile)
			}

			fmt.Printf("Restoring database '%s' on %s:%d from %s...\n", dbName, host, port, inputFile)

			psqlArgs := []string{
				"-h", host,
				"-p", fmt.Sprintf("%d", port),
				"-U", user,
				"-d", dbName,
				"-f", inputFile,
				"--single-transaction",
			}

			psql := exec.Command("psql", psqlArgs...) // #nosec G204
			psql.Stdout = os.Stdout
			psql.Stderr = os.Stderr

			if err := psql.Run(); err != nil {
				return fmt.Errorf("psql restore failed: %w", err)
			}

			fmt.Printf("Restore completed from %s\n", inputFile)
			return nil
		},
	}

	cmd.Flags().StringVar(&host, "host", "localhost", "Database host")
	cmd.Flags().IntVar(&port, "port", 5432, "Database port")
	cmd.Flags().StringVar(&dbName, "db", "vcstack", "Database name")
	cmd.Flags().StringVarP(&user, "user", "U", "vcstack", "Database user")

	return cmd
}

// ── Version ────────────────────────────────────────────────────────────

func newDBVersionCommand() *cobra.Command {
	var dsn string

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show current migration version",
		RunE: func(cmd *cobra.Command, args []string) error {
			runner, err := dbmigrate.NewRunner(dbmigrate.Config{
				DatabaseDSN:    dsn,
				MigrationsPath: "file://./migrations",
			})
			if err != nil {
				return fmt.Errorf("failed to initialize migrations: %w", err)
			}
			defer runner.Close() //nolint:errcheck

			version, dirty, err := runner.Version()
			if err != nil {
				return fmt.Errorf("failed to get version: %w", err)
			}

			status := "clean"
			if dirty {
				status = "DIRTY"
			}

			fmt.Printf("Migration version: %d (%s)\n", version, status)
			return nil
		},
	}

	cmd.Flags().StringVar(&dsn, "dsn", "", "PostgreSQL DSN")
	_ = cmd.MarkFlagRequired("dsn")

	return cmd
}
