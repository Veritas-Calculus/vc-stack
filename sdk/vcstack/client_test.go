package vcstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("https://api.example.com")
	if c.BaseURL != "https://api.example.com" {
		t.Errorf("expected base URL, got %s", c.BaseURL)
	}
	if c.Instances == nil || c.Volumes == nil || c.Networks == nil {
		t.Error("expected resource clients to be initialized")
	}
}

func TestSetAPIKey(t *testing.T) {
	c := NewClient("http://localhost")
	c.SetAPIKey("VC-AKIA-test", "secret123")
	if c.accessKeyID != "VC-AKIA-test" {
		t.Errorf("expected access key ID to be set")
	}
	if c.secretKey != "secret123" {
		t.Errorf("expected secret key to be set")
	}
}

func TestSetToken(t *testing.T) {
	c := NewClient("http://localhost")
	c.SetToken("jwt-token-here")
	if c.bearerToken != "jwt-token-here" {
		t.Errorf("expected bearer token to be set")
	}
}

func TestLogin(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/auth/login" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{
			AccessToken:  "test-token-abc",
			RefreshToken: "refresh-xyz",
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	resp, err := c.Login(context.Background(), "admin", "password")
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if resp.AccessToken != "test-token-abc" {
		t.Errorf("expected test-token-abc, got %s", resp.AccessToken)
	}
	if c.bearerToken != "test-token-abc" {
		t.Error("expected bearer token to be stored after login")
	}
}

func TestListInstances(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/instances" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"instances": []Instance{
				{ID: 1, Name: "web-1", Status: "running"},
				{ID: 2, Name: "db-1", Status: "stopped"},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	instances, err := c.Instances.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(instances) != 2 {
		t.Fatalf("expected 2 instances, got %d", len(instances))
	}
	if instances[0].Name != "web-1" {
		t.Errorf("expected name 'web-1', got %q", instances[0].Name)
	}
}

func TestAPIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	_, err := c.Instances.List(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 403 {
		t.Errorf("expected 403, got %d", apiErr.StatusCode)
	}
	if !strings.Contains(apiErr.Message, "forbidden") {
		t.Errorf("expected 'forbidden' in message, got %q", apiErr.Message)
	}
}

func TestHMACAuthHeader(t *testing.T) {
	var authHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"instances": []Instance{}})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetAPIKey("VC-AKIA-test123", "secret456")

	_, _ = c.Instances.List(context.Background())

	if !strings.HasPrefix(authHeader, "VC-HMAC-SHA256 ") {
		t.Errorf("expected VC-HMAC-SHA256 prefix, got %q", authHeader)
	}
	if !strings.Contains(authHeader, "AccessKeyId=VC-AKIA-test123") {
		t.Errorf("expected AccessKeyId in header, got %q", authHeader)
	}
	if !strings.Contains(authHeader, "Timestamp=") {
		t.Errorf("expected Timestamp in header, got %q", authHeader)
	}
	if !strings.Contains(authHeader, "Signature=") {
		t.Errorf("expected Signature in header, got %q", authHeader)
	}
}

func TestBearerAuthHeader(t *testing.T) {
	var authHeader string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"flavors": []Flavor{}})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("my-jwt-token")

	_, _ = c.Flavors.List(context.Background())

	if authHeader != "Bearer my-jwt-token" {
		t.Errorf("expected 'Bearer my-jwt-token', got %q", authHeader)
	}
}

func TestListVolumes(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"volumes": []Volume{
				{ID: 1, Name: "data-vol", SizeGB: 100, Status: "available"},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")
	volumes, err := c.Volumes.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(volumes) != 1 || volumes[0].Name != "data-vol" {
		t.Errorf("unexpected volumes: %+v", volumes)
	}
}

func TestListNetworks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"networks": []Network{
				{ID: 1, Name: "mgmt-net", External: true},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")
	networks, err := c.Networks.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(networks) != 1 || networks[0].Name != "mgmt-net" {
		t.Errorf("unexpected networks: %+v", networks)
	}
}
