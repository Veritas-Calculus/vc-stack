package management

import (
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/pkg/apiversion"
)

// SetupRoutes registers all management plane routes onto the provided Gin router.
func (s *Service) SetupRoutes(router *gin.Engine) {
	// ── Phase 1: Global middleware ────────────────────────────────────
	router.Use(apiversion.Middleware())
	router.Use(middleware.RequestTracing(s.logger))

	// ── Phase 2: Gateway & Auth Middleware ────────────────────────────
	// Gateway is a special module, we get it from registry
	if gatewaySvc := s.GetModule("gateway"); gatewaySvc != nil {
		// Use type assertion or interface if needed
		if provider, ok := gatewaySvc.(interface{ SetupMiddleware(*gin.Engine) }); ok {
			provider.SetupMiddleware(router)
		}
	}

	if s.jwtSecret != "" {
		router.Use(middleware.OptionalAuthMiddleware(s.jwtSecret, s.logger))
	}

	// ── Phase 3: Dynamic Module Routes ────────────────────────────────
	names := make([]string, 0, len(s.modules))
	for name := range s.modules {
		names = append(names, name)
	}
	sort.Strings(names)

	v1 := router.Group("/api/v1")
	for _, name := range names {
		mod := s.modules[name]

		// Skip special handling if already done
		if name == "gateway" {
			continue
		}

		// Basic route registration
		mod.SetupRoutes(router)

		// Group-based route registration
		if grr, ok := mod.(GroupRouteRegistrar); ok {
			grr.SetupGroupRoutes(v1)
		}
	}

	// ── Phase 4: Final Proxy ──────────────────────────────────────────
	if gatewaySvc := s.GetModule("gateway"); gatewaySvc != nil {
		if provider, ok := gatewaySvc.(interface{ SetupComputeProxyRoutes(*gin.Engine) }); ok {
			provider.SetupComputeProxyRoutes(router)
		}
	}
}
