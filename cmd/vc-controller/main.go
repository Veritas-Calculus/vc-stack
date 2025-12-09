// vc-controller: combined control-plane binary
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/controlplane"
	"github.com/Veritas-Calculus/vc-stack/pkg/database"
	"github.com/Veritas-Calculus/vc-stack/pkg/logger"
	pkgsentry "github.com/Veritas-Calculus/vc-stack/pkg/sentry"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	flag.Parse()

	// Initialize logger
	zapLogger, err := logger.New(logger.Config{Level: "info", Format: "json", Output: "stdout"})
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() { _ = zapLogger.Sync() }()

	// Initialize Sentry for error tracking
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		sentryEnv := os.Getenv("SENTRY_ENVIRONMENT")
		if sentryEnv == "" {
			sentryEnv = "production"
		}

		err := pkgsentry.Init(pkgsentry.Config{
			DSN:              sentryDSN,
			Environment:      sentryEnv,
			Release:          fmt.Sprintf("vc-controller@%s", Version),
			SampleRate:       1.0,
			TracesSampleRate: 0.2,
			Debug:            false,
		})
		if err != nil {
			zapLogger.Warn("failed to initialize sentry", zap.Error(err))
		} else {
			zapLogger.Info("sentry initialized successfully",
				zap.String("environment", sentryEnv),
				zap.String("release", Version))
			defer pkgsentry.Close()
		}
	} else {
		zapLogger.Info("sentry DSN not configured, error tracking disabled")
	}

	// Initialize database (best-effort). Uses env with sensible defaults.
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := 5432
	dbName := os.Getenv("DB_NAME")
	if dbName == "" {
		dbName = "vcstack"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "vcstack"
	}
	dbPass := os.Getenv("DB_PASS")
	db, dErr := database.New(database.Config{Host: dbHost, Port: dbPort, Name: dbName, Username: dbUser, Password: dbPass, SSLMode: "disable", MaxIdleConns: 2, MaxOpenConns: 4, ConnMaxLifetime: time.Minute * 5})
	if dErr != nil {
		zapLogger.Warn("database connect failed, continuing with limited functionality", zap.Error(dErr))
		db = nil
	}

	// Run database migrations if database is available
	if db != nil {
		zapLogger.Info("running database migrations")
		if err := database.AutoMigrate(db); err != nil {
			zapLogger.Fatal("database migration failed", zap.Error(err))
		}
		zapLogger.Info("database migrations completed successfully")
	}

	// Compose controlplane services via aggregator
	cpSvc, err := controlplane.New(controlplane.Config{DB: db, Logger: zapLogger})
	if err != nil {
		zapLogger.Fatal("failed to initialize controlplane services", zap.Error(err))
	}

	// Shared router: register all control-plane routes on one HTTP server.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register all control-plane routes via aggregator
	cpSvc.SetupRoutes(router)

	port := 8080
	if v := os.Getenv("VC_CONTROLLER_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}

	go func() {
		zapLogger.Info("starting vc-controller", zap.Int("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Error("server failed", zap.Error(err))
		}
	}()

	// Wait for termination
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down vc-controller")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("graceful shutdown failed", zap.Error(err))
	}
	zapLogger.Info("vc-controller stopped")
}
