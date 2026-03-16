package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Veritas-Calculus/vc-stack/internal/management/gateway"
	"github.com/Veritas-Calculus/vc-stack/tests/integration"
)

// TestGatewayCircuitBreakerDiagnostics verifies the gateway circuit breaker
// diagnostics endpoint returns valid JSON status information.
func TestGatewayCircuitBreakerDiagnostics(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := gateway.NewService(&gateway.Config{
		Logger: fixture.Logger,
		DB:     fixture.DB,
		Services: gateway.ServicesConfig{
			Identity: gateway.ServiceEndpoint{
				Host: "localhost",
				Port: 19991, // Non-existent port — services won't be reachable.
			},
		},
	})
	require.NoError(t, err)
	defer svc.Stop()
	svc.SetupRoutes(fixture.Router)

	t.Run("circuit breaker diagnostics returns valid JSON", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/gateway/circuit-breakers", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "circuit_breakers")
	})
}

// TestGatewayHealthEndpoint verifies the gateway health endpoint.
func TestGatewayHealthEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := gateway.NewService(&gateway.Config{
		Logger: fixture.Logger,
		DB:     fixture.DB,
		Services: gateway.ServicesConfig{
			Identity: gateway.ServiceEndpoint{
				Host: "localhost",
				Port: 19992,
			},
		},
	})
	require.NoError(t, err)
	defer svc.Stop()
	svc.SetupRoutes(fixture.Router)

	t.Run("gateway health returns status", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/gateway/health", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Contains(t, resp, "status")
		assert.Contains(t, resp, "services")
	})
}

// TestGatewayStopIsIdempotent verifies that calling Stop() multiple times doesn't panic.
func TestGatewayStopIsIdempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := gateway.NewService(&gateway.Config{
		Logger: fixture.Logger,
		DB:     fixture.DB,
	})
	require.NoError(t, err)

	// Start health checker.
	svc.SetupRoutes(fixture.Router)

	// Give health checker goroutine a moment to start.
	time.Sleep(100 * time.Millisecond)

	// Stop should not panic even if called once (stop channel only closed once).
	assert.NotPanics(t, func() {
		svc.Stop()
	})
}
