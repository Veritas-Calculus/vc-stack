package caas

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTest(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	logger, _ := zap.NewDevelopment()
	svc, err := NewService(Config{DB: db, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	r := gin.New()
	svc.SetupRoutes(r)
	return svc, r
}

func doReq(r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func parseJSON(w *httptest.ResponseRecorder) map[string]interface{} {
	var result map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &result)
	return result
}

func TestGetStatus(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "GET", "/api/v1/kubernetes/status", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	data := parseJSON(w)
	if data["status"] != "operational" {
		t.Errorf("expected operational status")
	}
}

func TestCreateAndListClusters(t *testing.T) {
	_, r := setupTest(t)
	body := map[string]interface{}{
		"name":         "test-k8s",
		"pod_cidr":     "10.244.0.0/16",
		"service_cidr": "10.96.0.0/16",
		"worker_count": 2,
	}
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseJSON(w)
	cluster := data["cluster"].(map[string]interface{})
	if cluster["name"] != "test-k8s" {
		t.Errorf("expected test-k8s, got %v", cluster["name"])
	}
	if cluster["status"] != "active" {
		t.Errorf("expected active, got %v", cluster["status"])
	}
	if cluster["cni_provider"] != "calico" {
		t.Errorf("expected calico CNI")
	}

	// List
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data = parseJSON(w)
	clusters := data["clusters"].([]interface{})
	if len(clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(clusters))
	}
}

func TestClusterWithBGP(t *testing.T) {
	_, r := setupTest(t)
	body := map[string]interface{}{
		"name":         "bgp-cluster",
		"calico_mode":  "bgp",
		"bgp_enabled":  true,
		"bgp_peer_asn": 65000,
		"bgp_node_asn": 65001,
		"worker_count": 2,
	}
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseJSON(w)
	cluster := data["cluster"].(map[string]interface{})
	clusterID := cluster["id"].(string)

	if cluster["calico_backend"] != "bird" {
		t.Errorf("expected bird backend for BGP mode, got %v", cluster["calico_backend"])
	}
	if cluster["vxlan_mode"] != "Never" {
		t.Errorf("expected VXLAN mode Never for BGP, got %v", cluster["vxlan_mode"])
	}

	// Check BGP peers were auto-created
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/bgp-peers", nil)
	data = parseJSON(w)
	peers := data["bgp_peers"].([]interface{})
	if len(peers) == 0 {
		t.Error("expected auto-created BGP peer for OVN router")
	}
}

func TestClusterNodes(t *testing.T) {
	_, r := setupTest(t)
	// Create cluster
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "node-test", "control_plane_count": 1, "worker_count": 2,
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	// List nodes
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/nodes", nil)
	data = parseJSON(w)
	nodes := data["nodes"].([]interface{})
	if len(nodes) != 3 { // 1 CP + 2 workers
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	// Add worker
	w = doReq(r, "POST", "/api/v1/kubernetes/clusters/"+clusterID+"/nodes", map[string]interface{}{
		"count": 1, "role": "worker",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201")
	}

	// Verify 4 nodes
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/nodes", nil)
	data = parseJSON(w)
	nodes = data["nodes"].([]interface{})
	if len(nodes) != 4 {
		t.Errorf("expected 4 nodes after add, got %d", len(nodes))
	}

	// Find a worker node to remove
	var workerID string
	for _, n := range nodes {
		node := n.(map[string]interface{})
		if node["role"] == "worker" {
			workerID = node["id"].(string)
			break
		}
	}

	// Remove worker
	w = doReq(r, "DELETE", "/api/v1/kubernetes/clusters/"+clusterID+"/nodes/"+workerID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}

	// Cannot remove control-plane
	for _, n := range nodes {
		node := n.(map[string]interface{})
		if node["role"] == "control-plane" {
			w = doReq(r, "DELETE", "/api/v1/kubernetes/clusters/"+clusterID+"/nodes/"+node["id"].(string), nil)
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for CP removal, got %d", w.Code)
			}
			break
		}
	}
}

func TestLoadBalancer(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "lb-test", "worker_count": 2,
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	// Create LB
	lbReq := map[string]interface{}{
		"service_name":  "nginx-svc",
		"namespace":     "default",
		"external_port": 80,
		"node_port":     30080,
		"protocol":      "TCP",
	}
	w = doReq(r, "POST", "/api/v1/kubernetes/clusters/"+clusterID+"/loadbalancers", lbReq)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	data = parseJSON(w)
	lb := data["loadbalancer"].(map[string]interface{})
	if lb["external_ip"] == "" {
		t.Error("expected external IP allocation")
	}
	if lb["status"] != "active" {
		t.Errorf("expected active status")
	}
	lbID := lb["id"].(string)

	// List LBs
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/loadbalancers", nil)
	data = parseJSON(w)
	lbs := data["loadbalancers"].([]interface{})
	if len(lbs) != 1 {
		t.Errorf("expected 1 LB")
	}

	// Delete LB
	w = doReq(r, "DELETE", "/api/v1/kubernetes/clusters/"+clusterID+"/loadbalancers/"+lbID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func TestIPPools(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "pool-test",
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	// Default pool should exist
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/ippools", nil)
	data = parseJSON(w)
	pools := data["ip_pools"].([]interface{})
	if len(pools) != 1 {
		t.Errorf("expected 1 default pool, got %d", len(pools))
	}

	// Create additional pool
	w = doReq(r, "POST", "/api/v1/kubernetes/clusters/"+clusterID+"/ippools", map[string]interface{}{
		"name": "extra-pool", "cidr": "10.245.0.0/16", "encapsulation": "VXLAN",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201")
	}

	// Delete pool
	data = parseJSON(w)
	poolID := data["ip_pool"].(map[string]interface{})["id"].(string)
	w = doReq(r, "DELETE", "/api/v1/kubernetes/clusters/"+clusterID+"/ippools/"+poolID, nil)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200")
	}
}

func TestNetworkPolicies(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "policy-test",
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	// Create policy
	w = doReq(r, "POST", "/api/v1/kubernetes/clusters/"+clusterID+"/network-policies", map[string]interface{}{
		"name": "deny-all", "policy_type": "calico", "namespace": "production",
		"spec": "apiVersion: projectcalico.org/v3\nkind: NetworkPolicy\nmetadata:\n  name: deny-all",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201")
	}

	// List
	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/network-policies", nil)
	data = parseJSON(w)
	policies := data["network_policies"].([]interface{})
	if len(policies) != 1 {
		t.Errorf("expected 1 policy")
	}
}

func TestClusterUpgrade(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "upgrade-test", "kubernetes_version": "1.29",
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	w = doReq(r, "POST", "/api/v1/kubernetes/clusters/"+clusterID+"/upgrade", map[string]interface{}{
		"target_version": "1.30",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data = parseJSON(w)
	cluster := data["cluster"].(map[string]interface{})
	if cluster["kubernetes_version"] != "1.30" {
		t.Errorf("expected 1.30, got %v", cluster["kubernetes_version"])
	}
}

func TestGetNetworking(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "net-test", "calico_mode": "bgp", "bgp_enabled": true,
		"bgp_peer_asn": 65000, "bgp_node_asn": 65001,
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID+"/networking", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}
	data = parseJSON(w)
	cni := data["cni"].(map[string]interface{})
	if cni["provider"] != "calico" {
		t.Errorf("expected calico")
	}
	if cni["bgp_enabled"] != true {
		t.Errorf("expected BGP enabled")
	}
}

func TestDeleteCluster(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "del-test",
	})
	data := parseJSON(w)
	clusterID := data["cluster"].(map[string]interface{})["id"].(string)

	w = doReq(r, "DELETE", "/api/v1/kubernetes/clusters/"+clusterID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200")
	}

	w = doReq(r, "GET", "/api/v1/kubernetes/clusters/"+clusterID, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete")
	}
}

func TestHACluster(t *testing.T) {
	_, r := setupTest(t)
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{
		"name": "ha-test", "ha_enabled": true, "control_plane_count": 1,
	})
	data := parseJSON(w)
	cluster := data["cluster"].(map[string]interface{})
	// HA should force 3 control planes
	if cluster["control_plane_count"].(float64) != 3 {
		t.Errorf("expected 3 CP nodes for HA, got %v", cluster["control_plane_count"])
	}
}

func TestDuplicateClusterName(t *testing.T) {
	_, r := setupTest(t)
	doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{"name": "dup-test"})
	w := doReq(r, "POST", "/api/v1/kubernetes/clusters", map[string]interface{}{"name": "dup-test"})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d", w.Code)
	}
}
