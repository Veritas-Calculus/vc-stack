package management

import "github.com/gin-gonic/gin"

// Module is the interface that all management plane modules must implement.
// It provides a uniform contract for route registration, allowing the
// management service to auto-discover and register modules without
// maintaining explicit references to each one.
//
// To add a new module to the management plane:
//  1. Implement Module on your service type.
//  2. In RegisterOptionalModules (modules.go), add a ModuleDescriptor whose
//     Factory stores the service via svc.RegisterModule(s).
//  3. That's it. No need to modify management.Service struct or routes.go.
type Module interface {
	// Name returns the unique identifier for this module (e.g., "kms", "dns").
	Name() string

	// SetupRoutes registers HTTP routes on the provided Gin engine.
	SetupRoutes(router *gin.Engine)
}

// MiddlewareProvider is an optional interface that modules can implement
// to provide Gin middleware that should be applied to all routes.
// Middleware is applied in module registration order.
type MiddlewareProvider interface {
	Module
	// Middleware returns a Gin middleware handler.
	Middleware() gin.HandlerFunc
}

// GroupRouteRegistrar is an optional interface for modules that register
// routes on a RouterGroup (e.g., under /api/v1) instead of the root Engine.
// If a module implements both Module and GroupRouteRegistrar, SetupRoutes
// will be called (for backward compatibility) AND SetupGroupRoutes will
// be called with a /api/v1 group.
type GroupRouteRegistrar interface {
	// SetupGroupRoutes registers routes on a specific router group.
	SetupGroupRoutes(rg *gin.RouterGroup)
}

// EngineRouteRegistrar is a minimal interface for services that have
// SetupRoutes(*gin.Engine). Used by moduleAdapter.
type EngineRouteRegistrar interface {
	SetupRoutes(router *gin.Engine)
}

// moduleAdapter wraps an existing service (with SetupRoutes) into a Module
// without requiring the service to implement Name(). This allows gradual
// migration: existing 35+ modules work via the adapter, new modules can
// implement Module directly.
type moduleAdapter struct {
	name string
	svc  EngineRouteRegistrar
}

func (a *moduleAdapter) Name() string                   { return a.name }
func (a *moduleAdapter) SetupRoutes(router *gin.Engine) { a.svc.SetupRoutes(router) }

// WrapModule creates a Module from any service that has SetupRoutes(*gin.Engine).
// This is the recommended way to register existing modules during the gradual
// migration period.
func WrapModule(name string, svc EngineRouteRegistrar) Module {
	return &moduleAdapter{name: name, svc: svc}
}
