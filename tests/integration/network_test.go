package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/management/network"
	"github.com/Veritas-Calculus/vc-stack/tests/integration"
)

// TestNetworkCRUD verifies full CRUD lifecycle for networks and subnets
// using the NoopDriver (no real OVN required).
func TestNetworkCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := network.NewService(network.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
		SDN: network.SDNConfig{
			Provider: "noop", // No real OVN needed for DB-level tests.
		},
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	var networkID string

	t.Run("create network", func(t *testing.T) {
		body := `{"name":"test-net","cidr":"10.0.0.0/24","gateway":"10.0.0.1","zone":"zone-1","tenant_id":"tenant-1"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/networks", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		require.Equal(t, http.StatusCreated, w.Code, "response: %s", w.Body.String())

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		net, ok := resp["network"].(map[string]interface{})
		require.True(t, ok, "expected network in response")
		assert.Equal(t, "test-net", net["name"])
		assert.Equal(t, "10.0.0.0/24", net["cidr"])
		networkID = net["id"].(string)
	})

	t.Run("list networks", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/networks", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		networks := resp["networks"].([]interface{})
		assert.GreaterOrEqual(t, len(networks), 1)
	})

	t.Run("get network by id", func(t *testing.T) {
		if networkID == "" {
			t.Skip("create_network failed, skipping")
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/networks/"+networkID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("update network", func(t *testing.T) {
		if networkID == "" {
			t.Skip("create_network failed, skipping")
		}
		body := `{"description":"integration test network"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/networks/"+networkID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete network", func(t *testing.T) {
		if networkID == "" {
			t.Skip("create_network failed, skipping")
		}
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/networks/"+networkID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestSecurityGroupCRUD verifies security group and rule lifecycle.
func TestSecurityGroupCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := network.NewService(network.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
		SDN:    network.SDNConfig{Provider: "noop"},
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	var sgID string

	t.Run("create security group", func(t *testing.T) {
		body := `{"name":"test-sg","description":"test security group","tenant_id":"tenant-1"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/security-groups", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		sg := resp["security_group"].(map[string]interface{})
		assert.Equal(t, "test-sg", sg["name"])
		sgID = sg["id"].(string)
	})

	t.Run("add security group rule", func(t *testing.T) {
		require.NotEmpty(t, sgID)
		body := `{"security_group_id":"` + sgID + `","direction":"ingress","protocol":"tcp","port_range_min":22,"port_range_max":22,"remote_ip_prefix":"0.0.0.0/0"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/security-group-rules", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("list security groups", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/security-groups", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("get security group with rules", func(t *testing.T) {
		require.NotEmpty(t, sgID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/security-groups/"+sgID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		sg := resp["security_group"].(map[string]interface{})
		rules, ok := sg["rules"].([]interface{})
		require.True(t, ok)
		assert.GreaterOrEqual(t, len(rules), 1)
	})

	t.Run("delete security group", func(t *testing.T) {
		require.NotEmpty(t, sgID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/security-groups/"+sgID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}

// TestZoneCRUD verifies zone CRUD operations.
func TestZoneCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := network.NewService(network.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
		SDN:    network.SDNConfig{Provider: "noop"},
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	var zoneID string

	t.Run("create zone", func(t *testing.T) {
		body := `{"name":"zone-1","type":"core"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/zones", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		zone := resp["zone"].(map[string]interface{})
		zoneID = zone["id"].(string)
	})

	t.Run("list zones", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/zones", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete zone", func(t *testing.T) {
		require.NotEmpty(t, zoneID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/zones/"+zoneID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
	})
}
