package sentry

import (
	"fmt"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

// GinMiddleware returns a Gin middleware that captures errors and sends them to Sentry.
func GinMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a new Sentry hub for this request.
		hub := sentry.CurrentHub().Clone()
		hub.Scope().SetRequest(c.Request)
		hub.Scope().SetTag("request_id", c.GetString("request_id"))

		// Add request context.
		hub.Scope().SetContext("request", map[string]interface{}{
			"method":      c.Request.Method,
			"url":         c.Request.URL.String(),
			"headers":     c.Request.Header,
			"remote_addr": c.ClientIP(),
			"user_agent":  c.Request.UserAgent(),
		})

		// Store hub in context.
		c.Set("sentry_hub", hub)

		// Recover from panics.
		defer func() {
			if err := recover(); err != nil {
				hub.RecoverWithContext(
					c.Request.Context(),
					err,
				)

				// Re-throw panic to let Gin's recovery middleware handle it.
				panic(err)
			}
		}()

		// Process request.
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		// Capture errors if status code >= 500.
		if c.Writer.Status() >= 500 {
			hub.WithScope(func(scope *sentry.Scope) {
				scope.SetTag("http_status", fmt.Sprintf("%d", c.Writer.Status()))
				scope.SetExtra("duration_ms", duration.Milliseconds())
				scope.SetLevel(sentry.LevelError)

				// Get error from context if available.
				if len(c.Errors) > 0 {
					for _, e := range c.Errors {
						hub.CaptureException(e.Err)
					}
				} else {
					hub.CaptureMessage(fmt.Sprintf("HTTP %d: %s %s",
						c.Writer.Status(),
						c.Request.Method,
						c.Request.URL.Path))
				}
			})
		}

		// Add breadcrumb for successful requests.
		if c.Writer.Status() < 400 {
			hub.AddBreadcrumb(&sentry.Breadcrumb{
				Type:     "http",
				Category: "http.request",
				Message:  fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path),
				Data: map[string]interface{}{
					"status":      c.Writer.Status(),
					"duration_ms": duration.Milliseconds(),
				},
				Level: sentry.LevelInfo,
			}, nil)
		}
	}
}

// CaptureGinError captures a Gin error and sends it to Sentry.
func CaptureGinError(c *gin.Context, err error, tags map[string]string) {
	if err == nil {
		return
	}

	hub := getHubFromContext(c)
	hub.WithScope(func(scope *sentry.Scope) {
		// Add tags.
		for k, v := range tags {
			scope.SetTag(k, v)
		}

		// Add request context.
		scope.SetContext("gin", map[string]interface{}{
			"path":      c.Request.URL.Path,
			"method":    c.Request.Method,
			"status":    c.Writer.Status(),
			"client_ip": c.ClientIP(),
			"params":    c.Params,
			"query":     c.Request.URL.Query(),
		})

		hub.CaptureException(err)
	})
}

// getHubFromContext retrieves Sentry hub from Gin context.
func getHubFromContext(c *gin.Context) *sentry.Hub {
	if hub, exists := c.Get("sentry_hub"); exists {
		if sentryHub, ok := hub.(*sentry.Hub); ok {
			return sentryHub
		}
	}
	return sentry.CurrentHub()
}

// RecordTransaction creates a Sentry transaction for performance monitoring.
func RecordTransaction(c *gin.Context, operation string) *sentry.Span {
	hub := getHubFromContext(c)

	ctx := c.Request.Context()
	transaction := sentry.StartTransaction(ctx,
		fmt.Sprintf("%s %s", c.Request.Method, c.FullPath()),
		sentry.WithOpName(operation),
		sentry.WithTransactionSource(sentry.SourceRoute),
	)

	transaction.SetTag("http.method", c.Request.Method)
	transaction.SetTag("http.route", c.FullPath())
	transaction.SetData("http.query", c.Request.URL.RawQuery)

	hub.Scope().SetTag("trace_id", transaction.TraceID.String())

	return transaction
}
