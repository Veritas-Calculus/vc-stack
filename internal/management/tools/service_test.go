package tools

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

func TestNewService_NilDB(t *testing.T) {
	svc, err := NewService(Config{DB: nil})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if svc != nil {
		t.Fatal("expected nil service when DB is nil")
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

// --- Comment handler tests ---

func TestCreateComment(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.POST("/api/v1/comments", svc.createComment)

	body := `{"resource_type":"instance","resource_id":"inst-001","body":"test comment"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createComment status = %d, want %d, body: %s", w.Code, http.StatusCreated, w.Body.String())
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	comment := resp["comment"].(map[string]interface{})
	if comment["author"] != "system" {
		t.Errorf("author = %v, want 'system' (default)", comment["author"])
	}
}

func TestListComments(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	db.Create(&Comment{ResourceType: "instance", ResourceID: "inst-001", Author: "admin", Body: "note"})

	router := gin.New()
	router.GET("/api/v1/comments", svc.listComments)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/comments?resource_type=instance&resource_id=inst-001", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listComments status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestDeleteComment(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	c := Comment{ResourceType: "instance", ResourceID: "inst-001", Author: "admin", Body: "note"}
	db.Create(&c)

	router := gin.New()
	router.DELETE("/api/v1/comments/:id", svc.deleteComment)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/v1/comments/1", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("deleteComment status = %d, want %d", w.Code, http.StatusOK)
	}
}

// --- Webhook handler tests ---

func TestCreateWebhook(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.POST("/api/v1/webhooks", svc.createWebhook)

	body := `{"name":"deploy-hook","url":"https://example.com/webhook","events":"instance.create,instance.delete"}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("createWebhook status = %d, want %d", w.Code, http.StatusCreated)
	}
}

func TestListWebhooks_SecretMasked(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	db.Create(&Webhook{Name: "test", URL: "https://example.com", Secret: "super-secret", Enabled: true})

	router := gin.New()
	router.GET("/api/v1/webhooks", svc.listWebhooks)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/webhooks", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("listWebhooks status = %d, want %d", w.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	webhooks := resp["webhooks"].([]interface{})
	if len(webhooks) == 0 {
		t.Fatal("expected at least one webhook")
	}
	wh := webhooks[0].(map[string]interface{})
	if wh["secret"] != "***" {
		t.Errorf("secret = %v, expected '***' (masked)", wh["secret"])
	}
}

func TestTestWebhook(t *testing.T) {
	db := testDB(t)
	svc, _ := NewService(Config{DB: db, Logger: zap.NewNop()})

	router := gin.New()
	router.POST("/api/v1/webhooks/:id/test", svc.testWebhook)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/webhooks/1/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("testWebhook status = %d, want %d", w.Code, http.StatusOK)
	}
}
