package apierrors

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAPIError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *APIError
		expected string
	}{
		{
			name:     "simple error",
			err:      New(400, "ValidationFailed", "invalid input"),
			expected: "[ValidationFailed] invalid input",
		},
		{
			name:     "error with detail",
			err:      New(404, "ResourceNotFound", "instance not found").WithDetail("id: 42"),
			expected: "[ResourceNotFound] instance not found: id: 42",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestRespond(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Respond(c, ErrNotFound("instance", "42"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}

	var resp APIError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if resp.Code != CodeInstanceNotFound {
		t.Errorf("expected code %s, got %s", CodeInstanceNotFound, resp.Code)
	}
	if resp.ErrorCompat == "" {
		t.Error("error field should be set for backward compatibility")
	}
	if resp.Detail != "id: 42" {
		t.Errorf("expected detail 'id: 42', got %q", resp.Detail)
	}
}

func TestRespond_WithRequestID(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		c.Set("request_id", "req-abc-123")
		Respond(c, ErrValidation("bad input"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp APIError
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.RequestID != "req-abc-123" {
		t.Errorf("expected request_id 'req-abc-123', got %q", resp.RequestID)
	}
}

func TestRespondWithData(t *testing.T) {
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		RespondWithData(c, ErrResourceInUse("image", 5), map[string]interface{}{
			"instance_count": 5,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}

	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["code"] != CodeResourceInUse {
		t.Errorf("expected code %s, got %v", CodeResourceInUse, resp["code"])
	}
	if resp["instance_count"] != float64(5) {
		t.Errorf("expected instance_count 5, got %v", resp["instance_count"])
	}
}

func TestAuthErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    *APIError
		status int
		code   string
	}{
		{"auth required", ErrAuthRequired(""), http.StatusUnauthorized, CodeAuthRequired},
		{"invalid creds", ErrInvalidCredentials(), http.StatusUnauthorized, CodeInvalidCredentials},
		{"token expired", ErrTokenExpired(), http.StatusUnauthorized, CodeTokenExpired},
		{"token invalid", ErrTokenInvalid(), http.StatusUnauthorized, CodeTokenInvalid},
		{"access denied", ErrAccessDenied("admin only"), http.StatusForbidden, CodeAccessDenied},
		{"rate limited", ErrRateLimited(), http.StatusTooManyRequests, CodeRateLimitExceeded},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.HTTPStatus != tt.status {
				t.Errorf("expected status %d, got %d", tt.status, tt.err.HTTPStatus)
			}
			if tt.err.Code != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, tt.err.Code)
			}
		})
	}
}

func TestValidationErrors(t *testing.T) {
	err := ErrInvalidParam("name", "must be alphanumeric")
	if err.HTTPStatus != 400 {
		t.Error("should be 400")
	}
	if err.Field != "name" {
		t.Errorf("field should be 'name', got %q", err.Field)
	}
	if err.Detail != "must be alphanumeric" {
		t.Errorf("detail should be set, got %q", err.Detail)
	}

	err2 := ErrMissingParam("image_id")
	if err2.Field != "image_id" {
		t.Error("field should be 'image_id'")
	}
	if !strings.Contains(err2.Message, "image_id") {
		t.Error("message should mention the field")
	}
}

func TestNotFoundErrors_TypeSpecificCodes(t *testing.T) {
	tests := []struct {
		resourceType string
		expectedCode string
	}{
		{"instance", CodeInstanceNotFound},
		{"flavor", CodeFlavorNotFound},
		{"image", CodeImageNotFound},
		{"host", CodeHostNotFound},
		{"network", CodeNetworkNotFound},
		{"subnet", CodeSubnetNotFound},
		{"port", CodePortNotFound},
		{"volume", CodeVolumeNotFound},
		{"snapshot", CodeSnapshotNotFound},
		{"security group", CodeSecGroupNotFound},
		{"floating ip", CodeFloatingIPNotFound},
		{"unknown-type", CodeResourceNotFound}, // fallback to generic
	}
	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			err := ErrNotFound(tt.resourceType, "123")
			if err.Code != tt.expectedCode {
				t.Errorf("for %s: expected code %s, got %s", tt.resourceType, tt.expectedCode, err.Code)
			}
			if err.HTTPStatus != 404 {
				t.Error("all not-found errors should be 404")
			}
		})
	}
}

func TestResourceErrors(t *testing.T) {
	t.Run("already exists", func(t *testing.T) {
		err := ErrAlreadyExists("image", "ubuntu-22.04")
		if err.HTTPStatus != 409 {
			t.Error("should be 409")
		}
		if !strings.Contains(err.Message, "ubuntu-22.04") {
			t.Error("should mention the name")
		}
	})

	t.Run("resource in use", func(t *testing.T) {
		err := ErrResourceInUse("image", 3)
		if err.HTTPStatus != 409 {
			t.Error("should be 409")
		}
		if !strings.Contains(err.Detail, "3") {
			t.Error("should mention count")
		}
	})

	t.Run("resource protected", func(t *testing.T) {
		err := ErrResourceProtected("image")
		if err.HTTPStatus != 403 {
			t.Error("should be 403")
		}
	})

	t.Run("state conflict", func(t *testing.T) {
		err := ErrStateConflict("instance", "stopped", "running")
		if err.HTTPStatus != 409 {
			t.Error("should be 409")
		}
		if err.Code != CodeStateConflict {
			t.Error("should use StateConflict code")
		}
	})
}

func TestQuotaErrors(t *testing.T) {
	err := ErrQuotaExceeded("vcpus", 32, 48)
	if err.HTTPStatus != 403 {
		t.Error("should be 403")
	}
	if err.Code != CodeQuotaExceeded {
		t.Error("should use QuotaExceeded code")
	}
	if !strings.Contains(err.Detail, "32") || !strings.Contains(err.Detail, "48") {
		t.Error("should include limit and requested in detail")
	}
}

func TestOperationErrors(t *testing.T) {
	t.Run("internal", func(t *testing.T) {
		err := ErrInternal("unexpected nil pointer")
		if err.HTTPStatus != 500 {
			t.Error("should be 500")
		}
		if err.Detail != "unexpected nil pointer" {
			t.Error("should include detail")
		}
		// Message should NOT leak internal details.
		if strings.Contains(err.Message, "nil pointer") {
			t.Error("message should not expose internal details")
		}
	})

	t.Run("database", func(t *testing.T) {
		err := ErrDatabase("create instance")
		if err.HTTPStatus != 500 {
			t.Error("should be 500")
		}
	})

	t.Run("no host", func(t *testing.T) {
		err := ErrNoHostAvailable()
		if err.HTTPStatus != 503 {
			t.Error("should be 503")
		}
	})

	t.Run("service unavailable", func(t *testing.T) {
		err := ErrServiceUnavailable("scheduler")
		if err.HTTPStatus != 503 {
			t.Error("should be 503")
		}
	})
}

func TestBackwardCompatibility(t *testing.T) {
	// Ensure the "error" field is always present for existing frontend clients.
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		Respond(c, ErrNotFound("instance", "99"))
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var raw map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &raw)

	// Must have both "error" (old format) and "code" (new format).
	if _, ok := raw["error"]; !ok {
		t.Error("response must include 'error' field for backward compatibility")
	}
	if _, ok := raw["code"]; !ok {
		t.Error("response must include 'code' field")
	}
	if _, ok := raw["message"]; !ok {
		t.Error("response must include 'message' field")
	}
}

func TestWithDetail_Chaining(t *testing.T) {
	err := ErrNotFound("instance", "").WithDetail("checked all regions").WithField("instance_id")
	if err.Detail != "checked all regions" {
		t.Error("detail should be set via chaining")
	}
	if err.Field != "instance_id" {
		t.Error("field should be set via chaining")
	}
}
