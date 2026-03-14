package metadata

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() { gin.SetMode(gin.TestMode) }

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test DB: %v", err)
	}
	if err := db.AutoMigrate(&Metadata{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return db
}

func TestNewService(t *testing.T) {
	db := testDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if svc == nil {
		t.Fatal("NewService() returned nil")
	}
}

func TestNewService_NilLogger(t *testing.T) {
	db := testDB(t)
	svc, err := NewService(Config{DB: db})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc == nil {
		t.Fatal("expected non-nil service")
	}
}

func TestMetadata_TableName(t *testing.T) {
	m := Metadata{}
	if m.TableName() != "sys_metadata" {
		t.Errorf("TableName() = %q, want %q", m.TableName(), "sys_metadata")
	}
}

func TestSetupRoutes(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	svc.SetupRoutes(router)

	routes := router.Routes()
	if len(routes) == 0 {
		t.Fatal("no routes registered")
	}
}

func TestCreateAndGetMetadata(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.POST("/api/v1/metadata/instances", svc.createMetadata)
	router.GET("/api/v1/metadata/instances/:id", svc.getInstanceMetadata)

	// Create
	body := `{"instance_id":"inst-001","hostname":"web-1","user_data":"#!/bin/bash"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/metadata/instances", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createMetadata status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	// Get
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v1/metadata/instances/inst-001", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("getInstanceMetadata status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetMetadata_NotFound(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/latest/meta-data", svc.getMetadata)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/latest/meta-data?instance_id=nonexistent", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("getMetadata status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestGetMetadata_MissingInstanceID(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.GET("/latest/meta-data", svc.getMetadata)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/latest/meta-data", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDeleteMetadata(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	db.Create(&Metadata{InstanceID: "inst-del", Hostname: "test"})

	router := gin.New()
	router.DELETE("/api/v1/metadata/instances/:id", svc.deleteMetadata)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/metadata/instances/inst-del", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("deleteMetadata status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "metadata deleted" {
		t.Errorf("unexpected response: %v", resp)
	}
}
