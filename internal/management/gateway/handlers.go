package gateway

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/Veritas-Calculus/vc-stack/pkg/circuitbreaker"
)

// rateLimitMiddleware implements rate limiting.
func (s *Service) rateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		s.mu.Lock()
		limiter, exists := s.limiters[clientIP]
		if !exists {
			limiter = rate.NewLimiter(
				rate.Limit(s.config.Security.RateLimit.RequestsPerMinute)/60,
				s.config.Security.RateLimit.RequestsPerMinute)
			s.limiters[clientIP] = limiter
		}
		s.mu.Unlock()

		if !limiter.Allow() {
			s.logger.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP))
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// loggingMiddleware logs requests.
func (s *Service) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		s.logger.Info("HTTP Request",
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.Int("status", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		)
		return ""
	})
}

// healthHandler returns the gateway health status.
func (s *Service) healthHandler(c *gin.Context) {
	s.mu.RLock()
	services := make(map[string]bool)
	for name, proxy := range s.services {
		services[name] = proxy.HealthOK
	}
	s.mu.RUnlock()

	allHealthy := true
	for _, healthy := range services {
		if !healthy {
			allHealthy = false
			break
		}
	}

	status := "healthy"
	httpStatus := http.StatusOK
	if !allHealthy {
		status = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":   status,
		"gateway":  "healthy",
		"services": services,
	})
}

// statusHandler returns detailed gateway status.
func (s *Service) statusHandler(c *gin.Context) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make(map[string]interface{})
	for name, proxy := range s.services {
		services[name] = map[string]interface{}{
			"healthy": proxy.HealthOK,
			"target":  proxy.Target.String(),
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"gateway": map[string]interface{}{
			"version":    "v1.0.0",
			"uptime":     time.Since(s.startTime).String(),
			"rate_limit": s.config.Security.RateLimit.Enabled,
		},
		"services": services,
	})
}

// proxyHandler creates a proxy handler for a specific service.
func (s *Service) proxyHandler(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		s.mu.RLock()
		proxy, exists := s.services[serviceName]
		s.mu.RUnlock()

		if !exists {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s not available", serviceName),
			})
			return
		}

		if !proxy.HealthOK {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": fmt.Sprintf("Service %s is unhealthy", serviceName),
			})
			return
		}

		// Check circuit breaker before forwarding.
		cb := s.cbManager.Get(serviceName)
		if cb.State() == circuitbreaker.StateOpen {
			s.logger.Warn("circuit breaker open, rejecting request",
				zap.String("service", serviceName),
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": circuitbreaker.FormatError(serviceName, circuitbreaker.ErrCircuitOpen).Error(),
			})
			return
		}

		c.Request.URL.Host = proxy.Target.Host
		c.Request.URL.Scheme = proxy.Target.Scheme
		c.Request.Header.Set("X-Forwarded-For", c.ClientIP())
		c.Request.Header.Set("X-Forwarded-Proto", "http")

		proxy.Proxy.ServeHTTP(c.Writer, c.Request) // #nosec G107 -- reverse proxy to vetted backend
	}
}

// listGatewayCircuitBreakers returns circuit breaker metrics for all proxied services.
func (s *Service) listGatewayCircuitBreakers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"circuit_breakers": s.cbManager.AllMetrics()})
}

// metricsHandler returns Prometheus metrics for standalone gateway mode.
func (s *Service) metricsHandler(c *gin.Context) {
	out := "# HELP vc_gateway_up Gateway is up and running.\n"
	out += "# TYPE vc_gateway_up gauge\n"
	out += "vc_gateway_up 1\n"

	out += "# HELP vc_gateway_uptime_seconds Gateway uptime in seconds.\n"
	out += "# TYPE vc_gateway_uptime_seconds gauge\n"
	out += fmt.Sprintf("vc_gateway_uptime_seconds %.0f\n", time.Since(s.startTime).Seconds())

	// Service health gauges.
	s.mu.RLock()
	for name, proxy := range s.services {
		val := 0
		if proxy.HealthOK {
			val = 1
		}
		out += fmt.Sprintf("vc_gateway_service_healthy{service=\"%s\"} %d\n", name, val)
	}
	s.mu.RUnlock()

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(out))
}
