// Package apiversion provides API version negotiation middleware.
//
// VC Stack uses URL-based versioning (/api/v1/...) as the primary strategy.
// This middleware adds version headers and supports deprecation notices.
package apiversion

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	// CurrentVersion is the latest stable API version.
	CurrentVersion = "v1"

	// HeaderAPIVersion is the response header indicating the API version used.
	HeaderAPIVersion = "X-API-Version"

	// HeaderDeprecation is the response header for deprecation notices.
	HeaderDeprecation = "X-API-Deprecated"

	// HeaderSunset is the response header indicating when an API version will be removed.
	HeaderSunset = "Sunset"
)

// VersionInfo describes a supported API version.
type VersionInfo struct {
	Version    string // e.g. "v1"
	Status     string // "current", "supported", "deprecated"
	SunsetDate string // ISO 8601 date when deprecated version will be removed (empty if active)
}

// SupportedVersions lists all API versions and their status.
var SupportedVersions = []VersionInfo{
	{Version: "v1", Status: "current"},
	// Future: {Version: "v2", Status: "current"},
	//         {Version: "v1", Status: "deprecated", SunsetDate: "2027-01-01"},
}

// Middleware adds API version headers to all responses and handles
// deprecation warnings for older API versions.
func Middleware() gin.HandlerFunc {
	// Build a lookup of deprecated versions.
	deprecated := make(map[string]VersionInfo)
	for _, v := range SupportedVersions {
		if v.Status == "deprecated" {
			deprecated[v.Version] = v
		}
	}

	return func(c *gin.Context) {
		// Determine which API version is being used from the URL path.
		path := c.Request.URL.Path
		version := extractVersion(path)
		if version == "" {
			version = CurrentVersion
		}

		// Set version header on all API responses.
		c.Header(HeaderAPIVersion, version)

		// Check for deprecated versions.
		if info, ok := deprecated[version]; ok {
			c.Header(HeaderDeprecation, "true")
			if info.SunsetDate != "" {
				c.Header(HeaderSunset, info.SunsetDate)
			}
		}

		c.Next()
	}
}

// VersionsHandler returns a handler that lists all supported API versions.
// GET /api/versions.
func VersionsHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"versions": SupportedVersions,
			"current":  CurrentVersion,
		})
	}
}

// extractVersion pulls the version segment from a URL path.
// e.g. "/api/v1/instances" → "v1".
func extractVersion(path string) string {
	parts := strings.Split(strings.TrimLeft(path, "/"), "/")
	for i, p := range parts {
		if p == "api" && i+1 < len(parts) {
			v := parts[i+1]
			if strings.HasPrefix(v, "v") {
				return v
			}
		}
	}
	return ""
}
