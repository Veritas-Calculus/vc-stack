package management

import (
	"sort"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/apidocs"
	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/Veritas-Calculus/vc-stack/pkg/apiversion"
)

// SetupRoutes registers all management plane routes onto the provided Gin router.
//
// Route registration follows a specific order to ensure correct middleware layering:
//  1. Global middleware (versioning, tracing)
//  2. API docs (public, no auth)
//  3. Gateway middleware (CORS, rate limiting)
//  4. Module routes (via Module interface, sorted by name for determinism)
//  5. Gateway proxy routes (wildcard, must be last)
func (s *Service) SetupRoutes(router *gin.Engine) {
	// ── Phase 1: Global middleware ────────────────────────────────────
	router.Use(apiversion.Middleware())
	router.Use(apidocs.VersionMiddleware())
	router.Use(middleware.RequestTracing(s.logger))

	// ── Phase 2: API docs (public, no auth) ──────────────────────────
	if s.APIDocs != nil {
		s.APIDocs.SetupRoutes(router)
	}

	// ── Phase 3: Gateway middleware (CORS, rate limiting, logging) ────
	s.Gateway.SetupMiddleware(router)
	if s.RateLimit != nil {
		router.Use(s.RateLimit.Middleware())
	}

	// ── Phase 4: Module routes via interface ──────────────────────────
	// All registered modules get their SetupRoutes called here, EXCEPT those
	// that are explicitly handled in other phases to avoid duplicate routes.
	skipModules := map[string]bool{
		"gateway": true, // handled in Phase 3 (middleware) + Phase 5 (proxy)
		"apidocs": true, // handled in Phase 2
	}

	// Sort module names for deterministic route registration order.
	names := make([]string, 0, len(s.modules))
	for name := range s.modules {
		if skipModules[name] {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	// Register each module's routes.
	v1 := router.Group("/api/v1")
	for _, name := range names {
		mod := s.modules[name]
		mod.SetupRoutes(router)

		// If the module also implements GroupRouteRegistrar, call it too.
		if grr, ok := mod.(GroupRouteRegistrar); ok {
			grr.SetupGroupRoutes(v1)
		}
	}

	// ── Phase 4b: RouterGroup-only modules (DNS, ObjStorage, Orchestration) ─
	// These modules use SetupRoutes(*gin.RouterGroup) instead of *gin.Engine.
	// They are NOT in the Module map because they have a different signature.
	// TODO(P3-02): Migrate these to the Module interface.
	if s.DNS != nil {
		s.DNS.SetupRoutes(v1)
	}
	if s.ObjStorage != nil {
		s.ObjStorage.SetupRoutes(v1)
	}
	if s.Orchestration != nil {
		s.Orchestration.SetupRoutes(v1)
	}

	// ── Phase 5: Gateway proxy routes (wildcard, must be last) ────────
	s.Gateway.SetupComputeProxyRoutes(router)
}
