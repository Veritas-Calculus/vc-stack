// vc-management: combined management plane binary.
//
// @title           VC Stack Management API
// @version         1.0.0
// @description     VC Stack Management Plane — IaaS API for compute, network, storage, and identity management.
// @termsOfService  https://github.com/Veritas-Calculus/vc-stack
//
// @contact.name   VC Stack Team
// @contact.url    https://github.com/Veritas-Calculus/vc-stack/issues
//
// @license.name  Apache 2.0
// @license.url   https://www.apache.org/licenses/LICENSE-2.0.html
//
// @host      localhost:8080
// @BasePath  /api/v1
//
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT Bearer token. Format: "Bearer {token}"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // #nosec G108 -- pprof bound to localhost:6060, not exposed externally
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/management"
	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/pkg/appconfig"
	"github.com/Veritas-Calculus/vc-stack/pkg/database"
	"github.com/Veritas-Calculus/vc-stack/pkg/dlock"
	_ "github.com/Veritas-Calculus/vc-stack/pkg/iam" // Register IAM permission mappings
	"github.com/Veritas-Calculus/vc-stack/pkg/logger"
	"github.com/Veritas-Calculus/vc-stack/pkg/metrics"
	"github.com/Veritas-Calculus/vc-stack/pkg/mq"
	pkgsentry "github.com/Veritas-Calculus/vc-stack/pkg/sentry"
	"github.com/Veritas-Calculus/vc-stack/pkg/telemetry"
	"github.com/Veritas-Calculus/vc-stack/pkg/vcredis"
)

var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	flag.Parse()

	// --- Load centralized configuration ---
	appCfg, err := appconfig.Load("vc-management")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize logger.
	zapLogger, err := logger.New(logger.Config{
		Level:  appCfg.Logging.Level,
		Format: appCfg.Logging.Format,
		Output: "stdout",
	})
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}
	defer func() { _ = zapLogger.Sync() }()

	zapLogger.Info("configuration loaded",
		zap.Int("server.port", appCfg.Server.Port),
		zap.String("database.host", appCfg.Database.Host),
		zap.String("logging.level", appCfg.Logging.Level))

	// Initialize Sentry for error tracking.
	if appCfg.SentryDSN != "" {
		sentryEnv := appCfg.SentryEnvironment
		if sentryEnv == "" {
			sentryEnv = "production"
		}
		err := pkgsentry.Init(pkgsentry.Config{
			DSN:              appCfg.SentryDSN,
			Environment:      sentryEnv,
			Release:          fmt.Sprintf("vc-management@%s", Version),
			SampleRate:       1.0,
			TracesSampleRate: 0.2,
			Debug:            false,
		})
		if err != nil {
			zapLogger.Warn("failed to initialize sentry", zap.Error(err))
		} else {
			zapLogger.Info("sentry initialized", zap.String("environment", sentryEnv))
			defer pkgsentry.Close()
		}
	} else {
		zapLogger.Info("sentry DSN not configured, error tracking disabled")
	}

	// Initialize database.
	db, dErr := database.New(database.Config{
		Host:            appCfg.Database.Host,
		Port:            appCfg.Database.Port,
		Name:            appCfg.Database.Name,
		Username:        appCfg.Database.Username,
		Password:        appCfg.Database.Password,
		SSLMode:         appCfg.Database.SSLMode,
		MaxIdleConns:    appCfg.Database.MaxIdleConns,
		MaxOpenConns:    appCfg.Database.MaxOpenConns,
		ConnMaxLifetime: appCfg.Database.ConnMaxLifetime,
	})
	if dErr != nil {
		zapLogger.Warn("database connect failed, continuing with limited functionality", zap.Error(dErr))
		db = nil
	}

	// Run database migrations if available.
	if db != nil {
		zapLogger.Info("running database migrations")
		if err := database.AutoMigrate(db); err != nil {
			zapLogger.Fatal("database migration failed", zap.Error(err))
		}
		if !db.Migrator().HasColumn(&compute.Instance{}, "DeletedAt") {
			if err := db.Migrator().AddColumn(&compute.Instance{}, "DeletedAt"); err != nil {
				zapLogger.Fatal("failed to add deleted_at to instances", zap.Error(err))
			}
		}
		zapLogger.Info("database migrations completed")
	}

	// --- Initialize etcd (distributed lock / leader election) ---
	var dlockMgr *dlock.Manager
	if len(appCfg.Etcd.Endpoints) > 0 {
		dlockMgr, err = dlock.NewManager(appCfg.Etcd, zapLogger.Named("dlock"))
		if err != nil {
			zapLogger.Warn("etcd connect failed, running in single-instance mode", zap.Error(err))
			dlockMgr = nil
		} else {
			zapLogger.Info("etcd connected for distributed locking",
				zap.Strings("endpoints", appCfg.Etcd.Endpoints))
			defer func() { _ = dlockMgr.Close() }()
		}
	} else {
		zapLogger.Info("etcd not configured, running in single-instance mode")
	}

	// --- Initialize Redis (session / cache) ---
	var redisMgr *vcredis.Manager
	if appCfg.Redis.Addr != "" || len(appCfg.Redis.SentinelAddrs) > 0 {
		redisMgr, err = vcredis.NewManager(appCfg.Redis, zapLogger.Named("redis"))
		if err != nil {
			zapLogger.Warn("Redis connect failed, using in-memory fallback", zap.Error(err))
			redisMgr = nil
		} else {
			zapLogger.Info("Redis connected for session/cache")
			defer func() { _ = redisMgr.Close() }()
		}
	} else {
		zapLogger.Info("Redis not configured, using in-memory session/cache")
	}

	// --- Initialize Kafka (message bus) ---
	var mqBus mq.MessageBus
	if len(appCfg.Kafka.Brokers) > 0 {
		kafkaBus, kErr := mq.NewKafkaBus(appCfg.Kafka, zapLogger.Named("kafka"))
		if kErr != nil {
			zapLogger.Warn("Kafka connect failed, using synchronous REST dispatch", zap.Error(kErr))
		} else {
			zapLogger.Info("Kafka connected for message bus",
				zap.Strings("brokers", appCfg.Kafka.Brokers))
			mqBus = kafkaBus
			defer func() { _ = kafkaBus.Close() }()
		}
	} else {
		zapLogger.Info("Kafka not configured, using synchronous REST dispatch")
	}

	// --- Compose management plane services ---
	jwtSecret := appCfg.Identity.JWTSecret
	if jwtSecret == "" {
		if appCfg.Server.GinMode == "release" {
			zapLogger.Fatal("JWT_SECRET is required in production (GIN_MODE=release). " +
				"Generate a strong secret with: openssl rand -base64 64")
		}
		zapLogger.Warn("JWT_SECRET not set — using insecure default. DO NOT use in production!")
		jwtSecret = "vc-stack-jwt-secret-change-me-in-production" // #nosec G101 -- dev-only fallback
	}

	mgmtCfg := management.Config{
		DB:        db,
		Logger:    zapLogger,
		JWTSecret: jwtSecret,
		DLock:     dlockMgr,
		Redis:     redisMgr,
		MQ:        mqBus,
		AppCfg:    appCfg,
	}
	mgmtCfg.SchedulerOvercommit.CPURatio = appCfg.Scheduler.CPUOvercommitRatio
	mgmtCfg.SchedulerOvercommit.RAMRatio = appCfg.Scheduler.RAMOvercommitRatio
	mgmtCfg.SchedulerOvercommit.DiskRatio = appCfg.Scheduler.DiskOvercommitRatio

	mgmtSvc, err := management.New(mgmtCfg)
	if err != nil {
		zapLogger.Fatal("failed to initialize management services", zap.Error(err))
	}

	// --- Initialize OpenTelemetry ---
	otelEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	otelShutdown, otelErr := telemetry.Init(telemetry.Config{
		ServiceName:    "vc-management",
		ServiceVersion: Version,
		Enabled:        otelEndpoint != "",
		Endpoint:       otelEndpoint,
		SampleRate:     0.1, // 10% sampling in production
		Insecure:       true,
	})
	if otelErr != nil {
		zapLogger.Warn("OpenTelemetry init failed", zap.Error(otelErr))
	} else if otelEndpoint != "" {
		zapLogger.Info("OpenTelemetry tracing enabled", zap.String("endpoint", otelEndpoint))
		defer func() { _ = otelShutdown(context.Background()) }()
	}

	// Setup HTTP router.
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(metrics.GinMiddleware())
	if otelEndpoint != "" {
		router.Use(telemetry.GinMiddleware("vc-management"))
	}
	// Note: /metrics is registered by the monitoring module via SetupRoutes.
	mgmtSvc.SetupRoutes(router)

	// Serve frontend Web Console.
	webDir := appCfg.WebConsoleDir
	if webDir == "" {
		if fi, err := os.Stat("/opt/vc-stack/web/console/dist/index.html"); err == nil && !fi.IsDir() {
			webDir = "/opt/vc-stack/web/console/dist"
		} else {
			webDir = "./web/console/dist"
		}
	}
	indexHTML := filepath.Join(webDir, "index.html")
	if _, err := os.Stat(indexHTML); err == nil { // #nosec G703
		zapLogger.Info("serving web console", zap.String("dir", webDir))
		router.Static("/assets", filepath.Join(webDir, "assets"))
		router.Static("/config", filepath.Join(webDir, "config"))
		router.StaticFile("/favicon.ico", filepath.Join(webDir, "favicon.ico"))
		router.StaticFile("/logo-42.svg", filepath.Join(webDir, "logo-42.svg"))
		router.NoRoute(func(c *gin.Context) {
			p := c.Request.URL.Path
			if strings.HasPrefix(p, "/api/") || strings.HasPrefix(p, "/ws/") {
				c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
				return
			}
			cleaned := filepath.Clean(p)
			filePath := filepath.Join(webDir, cleaned)
			absFile, err := filepath.Abs(filePath)
			if err != nil {
				c.File(indexHTML)
				return
			}
			absWeb, _ := filepath.Abs(webDir)
			if !strings.HasPrefix(absFile, absWeb+string(filepath.Separator)) {
				c.File(indexHTML)
				return
			}
			if fi, err := os.Stat(absFile); err == nil && !fi.IsDir() { // #nosec G703
				c.File(absFile)
				return
			}
			c.File(indexHTML)
		})
	} else {
		zapLogger.Warn("web console dist not found", zap.String("expected", indexHTML))
	}

	port := appCfg.Server.Port
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       appCfg.Server.ReadTimeout,
		WriteTimeout:      appCfg.Server.WriteTimeout,
		IdleTimeout:       appCfg.Server.IdleTimeout,
	}

	// Start pprof debug server on a separate port (non-production only).
	if appCfg.Server.GinMode != "release" || os.Getenv("VC_ENABLE_PPROF") == "true" {
		go func() {
			pprofAddr := ":6060"
			zapLogger.Info("starting pprof debug server", zap.String("addr", pprofAddr))
			pprofSrv := &http.Server{
				Addr:              pprofAddr,
				Handler:           http.DefaultServeMux,
				ReadHeaderTimeout: 10 * time.Second,
			}
			if err := pprofSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				zapLogger.Warn("pprof server failed", zap.Error(err))
			}
		}()
	}

	go func() {
		zapLogger.Info("starting vc-management", zap.Int("port", port), zap.String("version", Version))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zapLogger.Error("server failed", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	zapLogger.Info("shutting down vc-management")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zapLogger.Error("graceful shutdown failed", zap.Error(err))
	}
	zapLogger.Info("vc-management stopped")
}
