package task

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDBCounter uint64

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	n := atomic.AddUint64(&testDBCounter, 1)
	path := fmt.Sprintf("/tmp/vc_task_test_%d.db", n)
	t.Cleanup(func() { _ = os.Remove(path) })
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("failed to create task service: %v", err)
	}
	return svc, db
}

func TestCreateTask(t *testing.T) {
	svc, db := setupTestService(t)
	task, err := svc.CreateTask("create_instance", "instance", "inst-uuid-1", "my-vm", 1, 10)
	if err != nil {
		t.Fatalf("CreateTask error: %v", err)
	}
	if task.UUID == "" {
		t.Error("task UUID should not be empty")
	}
	if task.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", task.Status)
	}

	var count int64
	db.Model(&Task{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 task, got %d", count)
	}
}

func TestTaskLifecycle(t *testing.T) {
	svc, db := setupTestService(t)
	task, _ := svc.CreateTask("snapshot", "volume", "vol-1", "snap-vol", 1, 1)

	// Start.
	svc.StartTask(task.UUID)
	var started Task
	db.Where("uuid = ?", task.UUID).First(&started)
	if started.Status != "running" {
		t.Errorf("expected 'running', got '%s'", started.Status)
	}
	if started.StartedAt == nil {
		t.Error("started_at should be set")
	}

	// Progress.
	svc.UpdateProgress(task.UUID, 50, "halfway done")
	var progressed Task
	db.Where("uuid = ?", task.UUID).First(&progressed)
	if progressed.Progress != 50 {
		t.Errorf("expected progress 50, got %d", progressed.Progress)
	}

	// Complete.
	svc.CompleteTask(task.UUID, `{"snapshot_id": 42}`)
	var completed Task
	db.Where("uuid = ?", task.UUID).First(&completed)
	if completed.Status != "completed" {
		t.Errorf("expected 'completed', got '%s'", completed.Status)
	}
	if completed.Progress != 100 {
		t.Errorf("expected progress 100, got %d", completed.Progress)
	}
	if completed.Result != `{"snapshot_id": 42}` {
		t.Error("result should contain snapshot_id")
	}
}

func TestFailTask(t *testing.T) {
	svc, db := setupTestService(t)
	task, _ := svc.CreateTask("backup", "instance", "inst-1", "backup-vm", 1, 1)
	svc.StartTask(task.UUID)
	svc.FailTask(task.UUID, "disk full")

	var failed Task
	db.Where("uuid = ?", task.UUID).First(&failed)
	if failed.Status != "failed" {
		t.Errorf("expected 'failed', got '%s'", failed.Status)
	}
	if failed.ErrorMessage != "disk full" {
		t.Errorf("expected error 'disk full', got '%s'", failed.ErrorMessage)
	}
}

func TestCancelTask(t *testing.T) {
	svc, _ := setupTestService(t)
	task, _ := svc.CreateTask("migrate", "instance", "inst-1", "cancel-vm", 1, 1)
	svc.StartTask(task.UUID)

	err := svc.CancelTask(task.UUID)
	if err != nil {
		t.Fatalf("CancelTask error: %v", err)
	}

	// Cannot cancel again.
	err = svc.CancelTask(task.UUID)
	if err == nil {
		t.Error("should not be able to cancel an already cancelled task")
	}
}

func TestListTasks_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	svc.CreateTask("create_instance", "instance", "i-1", "vm-1", 1, 1)
	svc.CreateTask("snapshot", "volume", "v-1", "vol-snap", 1, 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "\"total\":2") {
		t.Error("response should report 2 tasks")
	}
}

func TestListTasks_FilterByStatus(t *testing.T) {
	svc, _ := setupTestService(t)
	task1, _ := svc.CreateTask("a", "instance", "i-1", "vm", 1, 1)
	svc.CreateTask("b", "instance", "i-2", "vm2", 1, 1)
	svc.StartTask(task1.UUID)
	svc.CompleteTask(task1.UUID, "")

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks?status=pending", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	// Only task2 should be pending.
	if !strings.Contains(w.Body.String(), "\"total\":1") {
		t.Errorf("expected 1 pending task, got: %s", w.Body.String())
	}
}

func TestGetTask_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	task, _ := svc.CreateTask("test", "instance", "i-1", "test-vm", 1, 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+task.UUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCancelTask_HTTP(t *testing.T) {
	svc, _ := setupTestService(t)
	task, _ := svc.CreateTask("test", "instance", "i-1", "test-vm", 1, 1)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.UUID+"/cancel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteTask_BlocksActive(t *testing.T) {
	svc, _ := setupTestService(t)
	task, _ := svc.CreateTask("test", "instance", "i-1", "test-vm", 1, 1)
	svc.StartTask(task.UUID)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	svc.SetupRoutes(router)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/tasks/"+task.UUID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected 409, got %d", w.Code)
	}
}
