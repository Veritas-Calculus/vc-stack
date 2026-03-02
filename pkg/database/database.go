// Package database provides database connection and management utilities.
// It follows Google's database best practices using GORM.
package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/Veritas-Calculus/vc-stack/pkg/security"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config represents the database configuration.
type Config struct {
	Host            string
	Port            int
	Name            string
	Username        string
	Password        string // #nosec // This is a configuration field
	SSLMode         string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// New creates a new database connection using the provided configuration.
func New(config Config) (*gorm.DB, error) {
	// Attempt to decrypt credentials if wrapped in ENC(...)
	username := config.Username
	password := config.Password // #nosec

	// If at least one of them looks like it's encrypted, look for a master key
	if strings.HasPrefix(username, "ENC(") || strings.HasPrefix(password, "ENC(") {
		key, err := security.GetMasterKey("")
		if err == nil && len(key) >= 16 {
			if strings.HasPrefix(username, "ENC(") {
				if dec, err := security.Decrypt(username, key); err == nil {
					username = dec
				}
			}
			if strings.HasPrefix(password, "ENC(") {
				if dec, err := security.Decrypt(password, key); err == nil {
					password = dec
				}
			}
		}
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, username, password, config.Name, config.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	// Configure connection pool.
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)

	// Test the connection.
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
