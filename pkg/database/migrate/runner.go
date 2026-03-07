// Package migrate provides versioned database migration support for VC Stack.
//
// It wraps golang-migrate to provide a unified migration interface that supports:
//   - Forward (up) and backward (down) migrations
//   - Migration version tracking via schema_migrations table
//   - File-based SQL migrations from the migrations/ directory
//   - Integration with the existing GORM setup
//
// Migration files follow the naming convention:
//
//	{version}_{description}.up.sql     — forward migration
//	{version}_{description}.down.sql   — backward migration (optional)
//
// Usage:
//
//	runner, err := migrate.NewRunner(migrate.Config{
//	    DatabaseDSN:    "postgres://user:pass@host/db?sslmode=disable",
//	    MigrationsPath: "./migrations",
//	    Logger:         zapLogger,
//	})
//	if err != nil { ... }
//	defer runner.Close()
//
//	if err := runner.Up(); err != nil { ... }
package migrate

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // PostgreSQL driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // File source driver
	"go.uber.org/zap"
)

// Config holds configuration for the migration runner.
type Config struct {
	// DatabaseDSN is the PostgreSQL connection string.
	// Format: postgres://user:password@host:port/dbname?sslmode=disable
	DatabaseDSN string

	// MigrationsPath is the path to the migration files directory.
	// The path should use the file:// scheme, e.g., "file://./migrations"
	MigrationsPath string

	// Logger is the structured logger for migration events.
	Logger *zap.Logger
}

// Runner manages database migrations using golang-migrate.
type Runner struct {
	m      *migrate.Migrate
	logger *zap.Logger
}

// NewRunner creates a new migration runner.
func NewRunner(cfg Config) (*Runner, error) {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	sourcePath := cfg.MigrationsPath
	if sourcePath == "" {
		sourcePath = "file://./migrations"
	}

	m, err := migrate.New(sourcePath, cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Runner{m: m, logger: cfg.Logger}, nil
}

// Up applies all pending migrations.
func (r *Runner) Up() error {
	r.logger.Info("applying database migrations")

	err := r.m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration up failed: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		r.logger.Info("database is up to date, no migrations applied")
	} else {
		version, dirty, _ := r.m.Version()
		r.logger.Info("migrations applied successfully",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty))
	}

	return nil
}

// Down rolls back the last migration.
func (r *Runner) Down() error {
	r.logger.Info("rolling back last migration")

	err := r.m.Steps(-1)
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migration down failed: %w", err)
	}

	version, dirty, _ := r.m.Version()
	r.logger.Info("migration rolled back",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty))

	return nil
}

// Version returns the current migration version.
func (r *Runner) Version() (uint, bool, error) {
	return r.m.Version()
}

// Force sets the migration version without running any migration.
// This is useful for fixing a dirty database state.
func (r *Runner) Force(version int) error {
	r.logger.Warn("forcing migration version", zap.Int("version", version))
	return r.m.Force(version)
}

// Close closes the migration runner and releases resources.
func (r *Runner) Close() error {
	srcErr, dbErr := r.m.Close()
	if srcErr != nil {
		return srcErr
	}
	return dbErr
}
