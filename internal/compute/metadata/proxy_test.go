package metadata

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func TestNewProxy_Defaults(t *testing.T) {
	p, err := NewProxy(ProxyConfig{})
	if err != nil {
		t.Fatalf("NewProxy failed: %v", err)
	}
	if p.port != "8082" {
		t.Errorf("expected default port 8082, got %q", p.port)
	}
	if p.mux == nil {
		t.Error("expected non-nil mux")
	}
}

func TestNewProxy_CustomPort(t *testing.T) {
	p, err := NewProxy(ProxyConfig{
		Logger: zap.NewNop(),
		Port:   "9090",
	})
	if err != nil {
		t.Fatalf("NewProxy failed: %v", err)
	}
	if p.port != "9090" {
		t.Errorf("expected port 9090, got %q", p.port)
	}
}

func TestHandleRoot(t *testing.T) {
	p, _ := NewProxy(ProxyConfig{Logger: zap.NewNop()})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	p.mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if body == "" {
		t.Error("expected non-empty response body")
	}
	if ct := w.Header().Get("Content-Type"); ct != "text/plain" {
		t.Errorf("expected Content-Type text/plain, got %q", ct)
	}
}

func TestHandleRoot_NonRootPath(t *testing.T) {
	p, _ := NewProxy(ProxyConfig{Logger: zap.NewNop()})

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()
	p.mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestInstanceInfo_Fields(t *testing.T) {
	info := instanceInfo{
		UUID:       "uuid-1",
		Name:       "test-vm",
		FlavorName: "m1.small",
		ImageUUID:  "img-1",
		SSHKey:     "ssh-rsa AAAA...",
		UserData:   "#cloud-config",
		IPAddress:  "10.0.0.5",
		Metadata:   map[string]interface{}{"key": "value"},
	}
	if info.UUID != "uuid-1" {
		t.Errorf("expected uuid-1, got %q", info.UUID)
	}
	if info.FlavorName != "m1.small" {
		t.Errorf("expected m1.small, got %q", info.FlavorName)
	}
}

func TestOpenStackMeta_Fields(t *testing.T) {
	meta := openStackMeta{
		UUID:             "uuid-1",
		Hostname:         "my-vm",
		Name:             "my-vm",
		AvailabilityZone: "az-1",
		PublicKeys:       map[string]string{"default": "ssh-rsa AAAA..."},
	}
	if meta.UUID != "uuid-1" {
		t.Errorf("expected uuid-1, got %q", meta.UUID)
	}
	if meta.AvailabilityZone != "az-1" {
		t.Errorf("expected az-1, got %q", meta.AvailabilityZone)
	}
}
