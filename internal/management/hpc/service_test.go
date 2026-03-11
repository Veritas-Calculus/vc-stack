package hpc

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

// stubIdentity is a minimal RequirePermission middleware that always allows.
type stubIdentity struct{}

func (s *stubIdentity) RequirePermission(_, _ string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("is_admin", true)
		c.Next()
	}
}

func setupTestHPC(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	logger := zap.NewNop()

	// Create roles and permissions tables so seedHPCRoles doesn't crash.
	_ = db.Exec("CREATE TABLE IF NOT EXISTS roles (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE, description TEXT, created_at DATETIME, updated_at DATETIME)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS permissions (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT UNIQUE, resource TEXT, action TEXT, description TEXT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS role_permissions (role_id INTEGER, permission_id INTEGER, created_at DATETIME, PRIMARY KEY(role_id, permission_id))")

	svc, err := NewService(Config{
		DB:        db,
		Logger:    logger,
		JWTSecret: "", // empty = skip AuthMiddleware in tests
		Identity:  &stubIdentity{},
	})
	if err != nil {
		t.Fatalf("failed to create hpc service: %v", err)
	}

	r := gin.New()
	// Set user context for handlers.
	r.Use(func(c *gin.Context) {
		c.Set("user_id", float64(1))
		c.Set("project_id", "proj-test")
		c.Set("is_admin", true)
		c.Next()
	})
	svc.SetupRoutes(r)
	return r, svc
}

func TestHPCStatus(t *testing.T) {
	r, _ := setupTestHPC(t)

	req, _ := http.NewRequest("GET", "/api/v1/hpc/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "operational" {
		t.Errorf("expected status=operational, got %v", resp["status"])
	}
}

func TestHPCK8sClusterCRUD(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create
	body, _ := json.Marshal(map[string]interface{}{
		"name":               "gpu-cluster-1",
		"kubernetes_version": "1.30",
		"worker_count":       2,
		"gpu_scheduler":      "volcano",
		"enable_mpi":         true,
	})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	cluster := createResp["cluster"].(map[string]interface{})
	clusterID := cluster["id"].(string)

	if cluster["name"] != "gpu-cluster-1" {
		t.Errorf("expected name=gpu-cluster-1, got %v", cluster["name"])
	}
	if cluster["gpu_scheduler"] != "volcano" {
		t.Errorf("expected gpu_scheduler=volcano, got %v", cluster["gpu_scheduler"])
	}

	// List
	req2, _ := http.NewRequest("GET", "/api/v1/hpc/kubernetes/clusters", nil)
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w2.Code)
	}
	var listResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &listResp)
	clusters := listResp["clusters"].([]interface{})
	if len(clusters) != 1 {
		t.Errorf("expected 1 cluster, got %d", len(clusters))
	}

	// Get
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/kubernetes/clusters/"+clusterID, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w3.Code)
	}
	var getResp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &getResp)
	nodes := getResp["nodes"].([]interface{})
	if len(nodes) == 0 {
		t.Error("expected provisioned nodes, got 0")
	}

	// Update
	updateBody, _ := json.Marshal(map[string]interface{}{"description": "Updated GPU cluster"})
	req4, _ := http.NewRequest("PUT", "/api/v1/hpc/kubernetes/clusters/"+clusterID, bytes.NewBuffer(updateBody))
	req4.Header.Set("Content-Type", "application/json")
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("update: expected 200, got %d", w4.Code)
	}

	// Delete
	req5, _ := http.NewRequest("DELETE", "/api/v1/hpc/kubernetes/clusters/"+clusterID, nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Errorf("delete: expected 200, got %d", w5.Code)
	}
}

func TestSlurmClusterCRUD(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create
	body, _ := json.Marshal(map[string]interface{}{
		"name":          "slurm-prod",
		"slurm_version": "24.05",
	})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("create slurm: expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var createResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	cluster := createResp["cluster"].(map[string]interface{})
	clusterID := cluster["id"].(string)

	// Create partition
	partBody, _ := json.Marshal(map[string]interface{}{
		"name":          "gpu-partition",
		"gpu_type":      "A100",
		"gpus_per_node": 4,
		"priority":      10,
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/slurm/clusters/"+clusterID+"/partitions", bytes.NewBuffer(partBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create partition: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	// List partitions
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/slurm/clusters/"+clusterID+"/partitions", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("list partitions: expected 200, got %d", w3.Code)
	}
	var partResp map[string]interface{}
	json.Unmarshal(w3.Body.Bytes(), &partResp)
	partitions := partResp["partitions"].([]interface{})
	if len(partitions) != 1 {
		t.Errorf("expected 1 partition, got %d", len(partitions))
	}

	// Get slurm cluster (includes partitions)
	req4, _ := http.NewRequest("GET", "/api/v1/hpc/slurm/clusters/"+clusterID, nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Fatalf("get slurm: expected 200, got %d", w4.Code)
	}

	// Delete
	req5, _ := http.NewRequest("DELETE", "/api/v1/hpc/slurm/clusters/"+clusterID, nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Errorf("delete slurm: expected 200, got %d", w5.Code)
	}
}

func TestHPCJobSubmitAndCancel(t *testing.T) {
	r, _ := setupTestHPC(t)

	// First create a cluster that the job targets.
	body, _ := json.Marshal(map[string]interface{}{"name": "job-cluster"})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Submit job
	jobBody, _ := json.Marshal(map[string]interface{}{
		"name":       "train-llama",
		"scheduler":  "kubernetes",
		"cluster_id": clusterID,
		"gpus":       4,
		"gpu_type":   "A100",
		"image":      "pytorch:2.2-cuda12.1",
		"script":     "python train.py",
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/jobs", bytes.NewBuffer(jobBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusCreated {
		t.Fatalf("submit job: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var jobResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &jobResp)
	job := jobResp["job"].(map[string]interface{})
	jobID := job["id"].(string)

	if job["scheduler"] != "kubernetes" {
		t.Errorf("expected scheduler=kubernetes, got %v", job["scheduler"])
	}
	if int(job["gpus"].(float64)) != 4 {
		t.Errorf("expected 4 gpus, got %v", job["gpus"])
	}

	// Get job
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/jobs/"+jobID, nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("get job: expected 200, got %d", w3.Code)
	}

	// Cancel job
	req4, _ := http.NewRequest("DELETE", "/api/v1/hpc/jobs/"+jobID, nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("cancel job: expected 200, got %d", w4.Code)
	}

	// Verify cancelled via GET
	req5, _ := http.NewRequest("GET", "/api/v1/hpc/jobs/"+jobID, nil)
	w5 := httptest.NewRecorder()
	r.ServeHTTP(w5, req5)
	if w5.Code != http.StatusOK {
		t.Fatalf("get cancelled job: expected 200, got %d", w5.Code)
	}
	var cancelledResp map[string]interface{}
	json.Unmarshal(w5.Body.Bytes(), &cancelledResp)
	cancelledJob := cancelledResp["job"].(map[string]interface{})
	if cancelledJob["status"] != "cancelled" {
		t.Errorf("expected status=cancelled, got %v", cancelledJob["status"])
	}

	// Cannot cancel again
	req6, _ := http.NewRequest("DELETE", "/api/v1/hpc/jobs/"+jobID, nil)
	w6 := httptest.NewRecorder()
	r.ServeHTTP(w6, req6)
	if w6.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for double cancel, got %d", w6.Code)
	}
}

func TestGPUPoolCRUD(t *testing.T) {
	r, _ := setupTestHPC(t)

	// Create cluster first
	body, _ := json.Marshal(map[string]interface{}{"name": "pool-cluster"})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	var cr map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &cr)
	clusterID := cr["cluster"].(map[string]interface{})["id"].(string)

	// Create GPU pool
	poolBody, _ := json.Marshal(map[string]interface{}{
		"name":        "a100-pool",
		"gpu_type":    "A100-80GB",
		"gpu_count":   8,
		"mig_enabled": true,
		"mig_profile": "1g.5gb",
	})
	req2, _ := http.NewRequest("POST", "/api/v1/hpc/kubernetes/clusters/"+clusterID+"/gpu-pools", bytes.NewBuffer(poolBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusCreated {
		t.Fatalf("create gpu pool: expected 201, got %d: %s", w2.Code, w2.Body.String())
	}

	var poolResp map[string]interface{}
	json.Unmarshal(w2.Body.Bytes(), &poolResp)
	pool := poolResp["gpu_pool"].(map[string]interface{})
	poolID := pool["id"].(string)

	if pool["gpu_type"] != "A100-80GB" {
		t.Errorf("expected gpu_type=A100-80GB, got %v", pool["gpu_type"])
	}
	if int(pool["available"].(float64)) != 8 {
		t.Errorf("expected available=8, got %v", pool["available"])
	}

	// List GPU pools
	req3, _ := http.NewRequest("GET", "/api/v1/hpc/kubernetes/clusters/"+clusterID+"/gpu-pools", nil)
	w3 := httptest.NewRecorder()
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusOK {
		t.Fatalf("list gpu pools: expected 200, got %d", w3.Code)
	}

	// Delete GPU pool
	req4, _ := http.NewRequest("DELETE", "/api/v1/hpc/kubernetes/clusters/"+clusterID+"/gpu-pools/"+poolID, nil)
	w4 := httptest.NewRecorder()
	r.ServeHTTP(w4, req4)
	if w4.Code != http.StatusOK {
		t.Errorf("delete gpu pool: expected 200, got %d", w4.Code)
	}
}

func TestHPCSeedRoles(t *testing.T) {
	_, svc := setupTestHPC(t)

	// Verify hpc_admin and hpc_user roles were seeded.
	var count int64
	svc.db.Table("roles").Where("name IN ?", []string{"hpc_admin", "hpc_user"}).Count(&count)
	if count != 2 {
		t.Errorf("expected 2 HPC roles (hpc_admin, hpc_user), got %d", count)
	}
}

func TestInvalidScheduler(t *testing.T) {
	r, _ := setupTestHPC(t)

	body, _ := json.Marshal(map[string]interface{}{
		"name":       "bad-job",
		"scheduler":  "invalid",
		"cluster_id": "any",
	})
	req, _ := http.NewRequest("POST", "/api/v1/hpc/jobs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid scheduler, got %d", w.Code)
	}
}

func TestK8sNotFound(t *testing.T) {
	r, _ := setupTestHPC(t)

	req, _ := http.NewRequest("GET", "/api/v1/hpc/kubernetes/clusters/non-existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}
