// vc-compute: combined compute node binary
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

	computenode "github.com/Veritas-Calculus/vc-stack/internal/compute"
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
		Use:   "vc-compute",
		Short: "VC Stack compute node",
		Long:  "VC Stack compute node - manages virtual machines, networking and storage on physical hosts",
	}

	// Server command (default behavior).
	serverCmd := &cobra.Command{
		Use:   "server",
		Short: "Run vc-compute server",
		Run:   runServer,
	}

	// VM management commands.
	vmCmd := &cobra.Command{
		Use:   "vm",
		Short: "Manage virtual machines",
	}

	var configDir string
	vmCmd.PersistentFlags().StringVar(&configDir, "config-dir", "/var/lib/vc-compute/configs", "Configuration directory")

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

func runServer(_ *cobra.Command, args []string) {
	flag.Parse()

	// Initialize logger.
	zapLogger, err := logger.New(logger.Config{Level: "info", Format: "json", Output: "stdout"})
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() {
		if err := zapLogger.Sync(); err != nil {
			log.Printf("failed to sync logger: %v", err)
		}
	}()

	// Initialize Sentry for error tracking.
	sentryDSN := os.Getenv("SENTRY_DSN")
	if sentryDSN != "" {
		sentryEnv := os.Getenv("SENTRY_ENVIRONMENT")
		if sentryEnv == "" {
			sentryEnv = "production"
		}

		err := pkgsentry.Init(pkgsentry.Config{
			DSN:              sentryDSN,
			Environment:      sentryEnv,
			Release:          fmt.Sprintf("vc-compute@%s", Version),
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

	// Initialize database (optional for compute node features).
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
		zapLogger.Warn("database connect failed, compute features limited", zap.Error(dErr))
		db = nil
	}

	// Compose compute node services via aggregator.
	nSvc, err := computenode.NewNode(computenode.NodeConfig{DB: db, Logger: zapLogger})
	if err != nil {
		zapLogger.Fatal("failed to initialize compute services", zap.Error(err))
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())

	// Register routes from all compute components via aggregator.
	nSvc.SetupRoutes(router)

	// Port selection.
	port := 8081
	if v := os.Getenv("VC_COMPUTE_PORT"); v != "" {
		if _, err := fmt.Sscanf(v, "%d", &port); err != nil {
			zapLogger.Warn("invalid VC_COMPUTE_PORT, using default 8081", zap.String("value", v), zap.Error(err))
			port = 8081
		}
	}

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		zapLogger.Info("starting vc-compute", zap.Int("port", port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Error("vc-compute server failed", zap.Error(err))
		}
	}()

	// Auto-register with management controller if CONTROLLER_URL is set.
	if controllerURL := os.Getenv("CONTROLLER_URL"); controllerURL != "" {
		go func() {
			time.Sleep(2 * time.Second) // Wait for server to fully start
			zapLogger.Info("auto-registering with controller", zap.String("url", controllerURL))

			nodeInfo := computenode.CollectNodeInfo(zapLogger)
			client := computenode.NewControllerClient(controllerURL, nodeInfo.Hostname, zapLogger)

			zoneID := os.Getenv("ZONE_ID")
			clusterID := os.Getenv("CLUSTER_ID")

			regCtx, regCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer regCancel()

			uuid, regErr := client.RegisterNode(regCtx, nodeInfo, port, zoneID, clusterID)
			if regErr != nil {
				zapLogger.Warn("auto-registration failed, will retry on next heartbeat",
					zap.Error(regErr))
			} else {
				zapLogger.Info("auto-registration successful",
					zap.String("uuid", uuid),
					zap.String("ip", nodeInfo.IPAddress),
					zap.Int("cpu_cores", nodeInfo.CPUCores),
					zap.Int64("ram_mb", nodeInfo.RAMMB),
					zap.Int64("disk_gb", nodeInfo.DiskGB))
			}
		}()
	} else {
		zapLogger.Info("CONTROLLER_URL not set, skipping auto-registration")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down vc-compute")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("vc-compute graceful shutdown failed", zap.Error(err))
	}
	zapLogger.Info("vc-compute stopped")
}
