package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/management/compute"
	"github.com/Veritas-Calculus/vc-stack/tests/integration"
)

// TestComputeFlavorCRUD verifies full CRUD lifecycle for compute flavors
// against a real PostgreSQL database.
func TestComputeFlavorCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := compute.NewService(compute.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	t.Run("default flavors are seeded", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/flavors", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		flavors, ok := resp["flavors"].([]interface{})
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(flavors), 10, "should have seeded default flavors")
	})

	t.Run("create custom flavor", func(t *testing.T) {
		body := `{"name":"test.custom","vcpus":4,"ram":8192,"disk":100}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/flavors", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		flavor := resp["flavor"].(map[string]interface{})
		assert.Equal(t, "test.custom", flavor["name"])
		assert.Equal(t, float64(4), flavor["vcpus"])
		assert.Equal(t, float64(8192), flavor["ram"])
	})

	t.Run("get flavor by id", func(t *testing.T) {
		// First create one so we know the ID.
		body := `{"name":"test.get","vcpus":2,"ram":4096,"disk":50}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/flavors", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var createResp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &createResp))
		flavor := createResp["flavor"].(map[string]interface{})
		id := flavor["id"]

		// Now get it.
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/api/v1/flavors/"+jsonID(id), http.NoBody)
		fixture.Router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("delete flavor", func(t *testing.T) {
		body := `{"name":"test.delete","vcpus":1,"ram":512,"disk":10}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/flavors", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		flavor := resp["flavor"].(map[string]interface{})
		id := flavor["id"]

		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("DELETE", "/api/v1/flavors/"+jsonID(id), http.NoBody)
		fixture.Router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
	})

	t.Run("duplicate flavor name rejected", func(t *testing.T) {
		body := `{"name":"test.dup","vcpus":1,"ram":512,"disk":10}`

		// First create.
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/flavors", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		// Second create with same name should fail.
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("POST", "/api/v1/flavors", bytes.NewBufferString(body))
		req2.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w2, req2)

		assert.NotEqual(t, http.StatusOK, w2.Code, "duplicate flavor should be rejected")
	})
}

// TestComputeSSHKeyCRUD verifies SSH key management operations.
func TestComputeSSHKeyCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := compute.NewService(compute.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	t.Run("create and list ssh keys", func(t *testing.T) {
		body := `{"name":"test-key","public_key":"ssh-rsa AAAAB3NzaC1yc2EAAA test@test"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/ssh-keys", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// List keys.
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/api/v1/ssh-keys", http.NoBody)
		fixture.Router.ServeHTTP(w2, req2)

		assert.Equal(t, http.StatusOK, w2.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w2.Body.Bytes(), &resp))
		keys, ok := resp["ssh_keys"].([]interface{})
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(keys), 1)
	})
}

// jsonID converts a JSON number/string to a string ID for URL construction.
func jsonID(v interface{}) string {
	switch id := v.(type) {
	case float64:
		return fmt.Sprintf("%d", int(id))
	case string:
		return id
	default:
		return "0"
	}
}
