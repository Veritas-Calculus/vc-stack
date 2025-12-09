// vc-node: combined node binary (compute + lite + netplugin)
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
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/node"
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
	rootCmd := &cobra.Command{
		Use:   "vc-node",
		Short: "VC Stack compute node",
		Long:  "VC Stack compute node - manages virtual machines and provides compute services",
	}

	// Server command (default behavior).
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run vc-node server",
		Run:   runServer,
	}

	// VM management commands.
	vmCmd := &cobra.Command{
		Use:   "vm",
		Short: "Manage virtual machines",
	}

	var configDir string
	vmCmd.PersistentFlags().StringVar(&configDir, "config-dir", "/var/lib/vc-node/configs", "Configuration directory")

	vmCmd.AddCommand(vmListCmd())
	vmCmd.AddCommand(vmShowCmd())
	vmCmd.AddCommand(vmStopCmd())
	vmCmd.AddCommand(vmDeleteCmd())
	vmCmd.AddCommand(vmConsoleCmd())

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(vmCmd)

	// Default to server if no command specified.
	if len(os.Args) == 1 {
		os.Args = append(os.Args, "server")
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runServer(cmd *cobra.Command, args []string) {
	flag.Parse()

	// Initialize logger.
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
			Release:          fmt.Sprintf("vc-node@%s", Version),
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

	// Initialize database (optional for node features).
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
	db, dErr := database.New(database.Config{Host: dbHost, Port: dbPort, Name: dbName, Username: dbUser, Password: dbPass, SSLMode: "disable", MaxIdleConns: 1, MaxOpenConns: 2, ConnMaxLifetime: time.Minute * 5})
	if dErr != nil {
		zapLogger.Warn("database connect failed, node features limited", zap.Error(dErr))
		db = nil
	}

	// Compose node services via aggregator.
	nSvc, err := node.New(node.Config{DB: db, Logger: zapLogger})
	if err != nil {
		zapLogger.Fatal("failed to initialize node services", zap.Error(err))
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes from all node components via aggregator.
	nSvc.SetupRoutes(router)

	// Port selection.
	port := 8081
	if v := os.Getenv("VC_NODE_PORT"); v != "" {
		fmt.Sscanf(v, "%d", &port)
	}

	srv := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: router}
	go func() {
		zapLogger.Info("starting vc-node", zap.Int("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Error("vc-node server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down vc-node")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("vc-node graceful shutdown failed", zap.Error(err))
	}
	zapLogger.Info("vc-node stopped")
}
