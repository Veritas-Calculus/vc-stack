package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

const (
	// RequestIDHeader is the header name for request tracing.
	RequestIDHeader = "X-Request-ID"
	// RequestIDKey is the gin context key for the request ID.
	RequestIDKey = "request_id"
)

// RequestTracing adds a unique request ID to every request for distributed tracing.
// If the client provides X-Request-ID, it is preserved; otherwise a new UUID is generated.
// The ID is set in the response header, gin context, and included in structured logs.
func RequestTracing(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Get or generate request ID.
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set in context and response header.
		c.Set(RequestIDKey, requestID)
		c.Header(RequestIDHeader, requestID)

		// Process request.
		c.Next()

		// Log with request ID for traceability.
		duration := time.Since(start)
		status := c.Writer.Status()

		if logger != nil {
			fields := []zap.Field{
				zap.String("request_id", requestID),
				zap.String("method", c.Request.Method),
				zap.String("path", c.Request.URL.Path),
				zap.Int("status", status),
				zap.Duration("duration", duration),
				zap.String("client_ip", c.ClientIP()),
			}

			if status >= 500 {
				logger.Error("request completed with server error", fields...)
			} else if status >= 400 {
				logger.Warn("request completed with client error", fields...)
			} else if duration > 5*time.Second {
				logger.Warn("slow request detected", fields...)
			}
			// Normal requests logged at debug to avoid noise.
		}
	}
}

// GetRequestID extracts the request ID from gin context.
func GetRequestID(c *gin.Context) string {
	if id, exists := c.Get(RequestIDKey); exists {
		if s, ok := id.(string); ok {
			return s
		}
	}
	return ""
}
