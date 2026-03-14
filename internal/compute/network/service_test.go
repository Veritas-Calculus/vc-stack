package network

import (
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestNewService_Defaults(t *testing.T) {
	svc, err := NewService(Config{})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	if svc.integrationBridge != "br-int" {
		t.Errorf("expected default bridge br-int, got %q", svc.integrationBridge)
	}
}

func TestNewService_CustomBridge(t *testing.T) {
	svc, err := NewService(Config{
		Logger:            zap.NewNop(),
		IntegrationBridge: "br-custom",
	})
	if err != nil {
		t.Fatalf("NewService failed: %v", err)
	}
	if svc.integrationBridge != "br-custom" {
		t.Errorf("expected br-custom, got %q", svc.integrationBridge)
	}
}

func TestSetupRoutes_RegistersEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc, _ := NewService(Config{Logger: zap.NewNop()})
	r := gin.New()
	svc.SetupRoutes(r)

	routes := r.Routes()
	expected := map[string]bool{
		"GET /api/v1/network-agent/health":        false,
		"POST /api/v1/network-agent/ports/attach": false,
		"POST /api/v1/network-agent/ports/detach": false,
		"GET /api/v1/network-agent/ports":         false,
		"GET /api/v1/network-agent/bridge/status": false,
	}

	for _, route := range routes {
		key := route.Method + " " + route.Path
		if _, ok := expected[key]; ok {
			expected[key] = true
		}
	}

	for route, found := range expected {
		if !found {
			t.Errorf("route not registered: %s", route)
		}
	}
}

func TestAttachPortRequest_Fields(t *testing.T) {
	req := AttachPortRequest{
		PortID:        "port-123",
		InterfaceName: "tap0",
		MACAddress:    "aa:bb:cc:dd:ee:ff",
		InstanceID:    "vm-456",
	}
	if req.PortID != "port-123" {
		t.Errorf("expected port-123, got %q", req.PortID)
	}
	if req.InterfaceName != "tap0" {
		t.Errorf("expected tap0, got %q", req.InterfaceName)
	}
}

func TestDetachPortRequest_Fields(t *testing.T) {
	req := DetachPortRequest{InterfaceName: "vnet0"}
	if req.InterfaceName != "vnet0" {
		t.Errorf("expected vnet0, got %q", req.InterfaceName)
	}
}
