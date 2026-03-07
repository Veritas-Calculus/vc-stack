// Package apidocs provides API version discovery, OpenAPI spec serving,
// and Swagger UI for the management plane.
package apidocs

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Version constants.
const (
	CurrentVersion = "v1"
	APIPrefix      = "/api"
	BuildVersion   = "1.0.0"
)

// VersionInfo describes a single API version.
type VersionInfo struct {
	Version    string `json:"version"`
	Status     string `json:"status"` // current, deprecated, experimental
	MinVersion string `json:"min_version,omitempty"`
	UpdatedAt  string `json:"updated_at"`
}

// Config contains the API docs service configuration.
type Config struct {
	Logger *zap.Logger
}

// Service provides API version discovery and documentation.
type Service struct {
	logger *zap.Logger
}

// NewService creates a new API docs service.
func NewService(cfg Config) *Service {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &Service{logger: cfg.Logger}
}

// SetupRoutes registers API discovery, OpenAPI spec, and Swagger UI routes.
// These routes are public (no auth required).
func (s *Service) SetupRoutes(router *gin.Engine) {
	// API version discovery (no auth).
	router.GET("/api/versions", s.listVersions)
	router.GET("/api/v1", s.versionDetail)

	// OpenAPI spec endpoints (no auth).
	router.GET("/api/v1/openapi.json", s.serveOpenAPIJSON)
	router.GET("/api/v1/openapi.yaml", s.serveOpenAPIYAML)

	// Legacy Swagger UI (static HTML).
	router.GET("/api/docs", s.serveSwaggerUI)
	router.GET("/api/docs/*any", s.serveSwaggerUI)

	// gin-swagger interactive Swagger UI.
	// After running `swag init -g cmd/vc-management/main.go`, this serves
	// the auto-generated spec with full interactivity.
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler,
		ginSwagger.URL("/api/v1/openapi.json"),
		ginSwagger.DefaultModelsExpandDepth(-1),
	))
}

// VersionMiddleware adds API version headers to all responses.
func VersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-API-Version", CurrentVersion)
		c.Header("X-API-Build", BuildVersion)
		c.Header("X-API-Deprecation", "") // Empty = not deprecated
		c.Next()
	}
}

// listVersions handles GET /api/versions - lists all available API versions.
func (s *Service) listVersions(c *gin.Context) {
	versions := []VersionInfo{
		{
			Version:   "v1",
			Status:    "current",
			UpdatedAt: time.Now().Format("2006-01-02"),
		},
	}
	c.JSON(http.StatusOK, gin.H{
		"versions":        versions,
		"default_version": CurrentVersion,
		"build":           BuildVersion,
		"links": gin.H{
			"self":    "/api/versions",
			"current": "/api/v1",
			"docs":    "/api/docs",
			"spec":    "/api/v1/openapi.json",
		},
	})
}

// versionDetail handles GET /api/v1 - details about v1 API.
func (s *Service) versionDetail(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": "v1",
		"status":  "current",
		"build":   BuildVersion,
		"resources": []string{
			"auth", "users", "roles", "permissions", "policies", "projects",
			"instances", "flavors", "images", "volumes", "snapshots",
			"networks", "subnets", "security-groups", "floating-ips", "routers", "vpcs", "ports", "asns",
			"hosts", "zones", "clusters",
			"tasks", "tags", "events", "quotas",
			"notifications", "storage", "ssh-keys",
			"migrations", "backup", "autoscale",
			"mfa", "kms", "encryption", "compliance",
			"ha", "dr", "self-heal",
			"caas", "baremetal", "object-storage", "dns",
			"catalog", "firecracker",
		},
		"links": gin.H{
			"spec": "/api/v1/openapi.json",
			"docs": "/api/docs",
		},
	})
}

// serveOpenAPIJSON serves the OpenAPI spec as JSON.
func (s *Service) serveOpenAPIJSON(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.String(http.StatusOK, openAPISpecJSON)
}

// serveOpenAPIYAML serves the OpenAPI spec as YAML.
func (s *Service) serveOpenAPIYAML(c *gin.Context) {
	c.Header("Content-Type", "application/x-yaml")
	c.String(http.StatusOK, openAPISpecYAML)
}

// serveSwaggerUI serves the Swagger UI HTML page.
func (s *Service) serveSwaggerUI(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, swaggerUIHTML)
}
