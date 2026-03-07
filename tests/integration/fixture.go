// Package integration provides shared test fixtures for integration tests.
//
// It uses testcontainers-go to spin up a real PostgreSQL database in Docker,
// create a GORM connection, and run AutoMigrate — exactly like production.
//
// Usage in tests:
//
//	func TestSomething(t *testing.T) {
//	    if testing.Short() {
//	        t.Skip("skipping integration test in short mode")
//	    }
//	    ctx := context.Background()
//	    fixture := integration.NewFixture(t, ctx)
//	    defer fixture.Teardown()
//
//	    // Use fixture.DB for database operations
//	    // Use fixture.Router for HTTP testing
//	}
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/database"
)

const (
	testDBName = "vcstack_test"
	testDBUser = "vcstack_test"
	testDBPass = "test_password" // #nosec G101 -- test-only
)

// Fixture holds shared test infrastructure.
type Fixture struct {
	t         *testing.T
	ctx       context.Context
	container testcontainers.Container
	DB        *gorm.DB
	Logger    *zap.Logger
	Router    *gin.Engine
	DSN       string
}

// NewFixture creates a new integration test fixture with a real PostgreSQL database.
// It spins up a container, connects GORM, runs AutoMigrate, and sets up a Gin router.
func NewFixture(t *testing.T, ctx context.Context) *Fixture {
	t.Helper()

	// Start PostgreSQL container.
	pgContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	// Get connection details.
	host, err := pgContainer.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get container port: %v", err)
	}

	// Connect via GORM.
	db, err := database.New(database.Config{
		Host:            host,
		Port:            port.Int(),
		Name:            testDBName,
		Username:        testDBUser,
		Password:        testDBPass,
		SSLMode:         "disable",
		MaxIdleConns:    2,
		MaxOpenConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		LogLevel:        "warn",
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	// Run migrations.
	if err := database.AutoMigrate(db); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// Setup logger and router.
	logger := zap.NewNop()
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(gin.Recovery())

	return &Fixture{
		t:         t,
		ctx:       ctx,
		container: pgContainer,
		DB:        db,
		Logger:    logger,
		Router:    router,
		DSN:       fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable", testDBUser, testDBPass, host, port.Int(), testDBName),
	}
}

// Teardown cleans up the test fixture by stopping the PostgreSQL container.
func (f *Fixture) Teardown() {
	f.t.Helper()

	// Close DB.
	if f.DB != nil {
		sqlDB, err := f.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}

	// Stop container.
	if f.container != nil {
		if err := f.container.Terminate(f.ctx); err != nil {
			f.t.Logf("warning: failed to terminate postgres container: %v", err)
		}
	}
}
