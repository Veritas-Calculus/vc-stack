package sentry

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// Config holds Sentry configuration
type Config struct {
	DSN              string
	Environment      string // dev, staging, production
	Release          string // version string
	SampleRate       float64
	TracesSampleRate float64
	Debug            bool
	ServerName       string
}

// Init initializes Sentry SDK with configuration
func Init(cfg Config) error {
	if cfg.DSN == "" {
		return fmt.Errorf("sentry DSN is required")
	}

	// Set defaults
	if cfg.Environment == "" {
		cfg.Environment = "production"
	}
	if cfg.SampleRate == 0 {
		cfg.SampleRate = 1.0
	}
	if cfg.TracesSampleRate == 0 {
		cfg.TracesSampleRate = 0.2 // 20% for performance monitoring
	}
	if cfg.ServerName == "" {
		hostname, _ := os.Hostname()
		cfg.ServerName = hostname
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		Release:          cfg.Release,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		Debug:            cfg.Debug,
		ServerName:       cfg.ServerName,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// Add custom logic before sending events
			// For example, filter sensitive data
			return event
		},
		AttachStacktrace: true,
		MaxBreadcrumbs:   30,
	})

	if err != nil {
		return fmt.Errorf("failed to initialize sentry: %w", err)
	}

	return nil
}

// Flush waits for Sentry to send all pending events
func Flush(timeout time.Duration) {
	sentry.Flush(timeout)
}

// Close flushes and closes the Sentry client
func Close() {
	sentry.Flush(2 * time.Second)
}

// CaptureError captures an error and sends it to Sentry
func CaptureError(err error, tags map[string]string, extra map[string]interface{}) {
	if err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		// Add tags
		for k, v := range tags {
			scope.SetTag(k, v)
		}

		// Add extra context
		for k, v := range extra {
			scope.SetExtra(k, v)
		}

		sentry.CaptureException(err)
	})
}

// CaptureMessage captures a message and sends it to Sentry
func CaptureMessage(message string, level sentry.Level, tags map[string]string) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetLevel(level)

		for k, v := range tags {
			scope.SetTag(k, v)
		}

		sentry.CaptureMessage(message)
	})
}

// SetUser sets the user context for Sentry events
func SetUser(userID, username, email string) {
	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:       userID,
			Username: username,
			Email:    email,
		})
	})
}

// AddBreadcrumb adds a breadcrumb for debugging
func AddBreadcrumb(category, message string, data map[string]interface{}) {
	sentry.AddBreadcrumb(&sentry.Breadcrumb{
		Category: category,
		Message:  message,
		Data:     data,
		Level:    sentry.LevelInfo,
	})
}

// ZapToSentryLevel converts zap log level to Sentry level
func ZapToSentryLevel(zapLevel string) sentry.Level {
	switch zapLevel {
	case "debug":
		return sentry.LevelDebug
	case "info":
		return sentry.LevelInfo
	case "warn", "warning":
		return sentry.LevelWarning
	case "error":
		return sentry.LevelError
	case "fatal", "panic":
		return sentry.LevelFatal
	default:
		return sentry.LevelInfo
	}
}

// ZapSentryHook is a zap core that sends errors to Sentry
type ZapSentryHook struct {
	logger *zap.Logger
}

// NewZapSentryHook creates a new Zap-Sentry integration
func NewZapSentryHook(logger *zap.Logger) *ZapSentryHook {
	return &ZapSentryHook{logger: logger}
}
