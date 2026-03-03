// Package database provides database migration utilities.
package database

import (
	"fmt"

	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// AutoMigrate runs automatic database migrations for all models.
func AutoMigrate(db *gorm.DB) error {
	// Enable required PostgreSQL extensions.
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error; err != nil {
		return fmt.Errorf("failed to create uuid-ossp extension: %w", err)
	}

	// Create custom types if they don't exist.
	if err := createCustomTypes(db); err != nil {
		return fmt.Errorf("failed to create custom types: %w", err)
	}

	// Auto-migrate all shared models.
	// This is the single source of truth for database schema.
	// Node services must NOT run their own AutoMigrate.
	if err := db.AutoMigrate(
		// Infrastructure models
		&models.Host{},
		// Compute models
		&models.Flavor{},
		&models.Image{},
		&models.Instance{},
		&models.Volume{},
		&models.Snapshot{},
		&models.SSHKey{},
		&models.VolumeAttachment{},
		&models.AuditLog{},
	); err != nil {
		return fmt.Errorf("failed to auto-migrate: %w", err)
	}

	return nil
}

// createCustomTypes creates PostgreSQL custom types.
func createCustomTypes(db *gorm.DB) error {
	types := []string{
		`DO $$ BEGIN
			CREATE TYPE host_type AS ENUM ('compute', 'storage', 'network', 'routing');
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;`,
		`DO $$ BEGIN
			CREATE TYPE host_status AS ENUM (
				'up', 'down', 'error', 'maintenance',
				'disabled', 'connecting', 'disconnected'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;`,
		`DO $$ BEGIN
			CREATE TYPE host_resource_state AS ENUM (
				'enabled', 'disabled', 'maintenance', 'error'
			);
		EXCEPTION
			WHEN duplicate_object THEN null;
		END $$;`,
	}

	for _, typeSQL := range types {
		if err := db.Exec(typeSQL).Error; err != nil {
			return err
		}
	}

	return nil
}
