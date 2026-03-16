package vcstack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAndGetInstance(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/instances":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"instance": Instance{ID: 42, Name: "new-vm", Status: "building"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/instances/42":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"instance": Instance{ID: 42, Name: "new-vm", Status: "running"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/instances/42":
			w.WriteHeader(204)
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	// Create
	inst, err := c.Instances.Create(context.Background(), &CreateInstanceRequest{
		Name: "new-vm", FlavorID: 1, ImageID: 1,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if inst.ID != 42 || inst.Name != "new-vm" {
		t.Errorf("unexpected instance: %+v", inst)
	}

	// Get
	got, err := c.Instances.Get(context.Background(), "42")
	if err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if got.Status != "running" {
		t.Errorf("expected running, got %s", got.Status)
	}

	// Delete
	if err := c.Instances.Delete(context.Background(), "42"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
}

func TestInstanceAction(t *testing.T) {
	var actionPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actionPath = r.URL.Path
		w.WriteHeader(204)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")
	if err := c.Instances.Action(context.Background(), "5", "stop"); err != nil {
		t.Fatalf("Action error: %v", err)
	}
	if actionPath != "/v1/instances/5/stop" {
		t.Errorf("expected /v1/instances/5/stop, got %s", actionPath)
	}
}

func TestListRouters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/routers" {
			http.Error(w, "not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"routers": []Router{
				{ID: 1, Name: "router-01", Status: "ACTIVE"},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")
	routers, err := c.Routers.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(routers) != 1 || routers[0].Name != "router-01" {
		t.Errorf("unexpected routers: %+v", routers)
	}
}

func TestCreateAndDeleteRouter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/routers":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"router": Router{ID: 10, Name: "my-router", Status: "ACTIVE"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/routers/10":
			w.WriteHeader(204)
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	router, err := c.Routers.Create(context.Background(), &CreateRouterRequest{Name: "my-router"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if router.Name != "my-router" {
		t.Errorf("unexpected name: %s", router.Name)
	}

	if err := c.Routers.Delete(context.Background(), "10"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
}

func TestListSecurityGroups(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"security_groups": []SecurityGroup{
				{ID: 1, Name: "default", Description: "Default SG"},
				{ID: 2, Name: "web", Description: "Web servers"},
			},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")
	sgs, err := c.SecurityGroups.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(sgs) != 2 {
		t.Fatalf("expected 2 security groups, got %d", len(sgs))
	}
}

func TestCreateAndDeleteSecurityGroup(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/security-groups":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"security_group": SecurityGroup{ID: 5, Name: "api-sg"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/security-groups/5":
			w.WriteHeader(204)
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	sg, err := c.SecurityGroups.Create(context.Background(), &CreateSecurityGroupRequest{Name: "api-sg"})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if sg.Name != "api-sg" {
		t.Errorf("unexpected name: %s", sg.Name)
	}
	if err := c.SecurityGroups.Delete(context.Background(), "5"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
}

func TestSSHKeyOperations(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/ssh-keys":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ssh_keys": []SSHKey{{ID: 1, Name: "my-key", Fingerprint: "aa:bb:cc"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/ssh-keys":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"ssh_key": SSHKey{ID: 2, Name: "new-key", PublicKey: "ssh-ed25519 AAAA"},
			})
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/ssh-keys/2":
			w.WriteHeader(204)
		default:
			http.Error(w, "not found", 404)
		}
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	keys, err := c.SSHKeys.List(context.Background())
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	key, err := c.SSHKeys.Create(context.Background(), "new-key", "ssh-ed25519 AAAA")
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if key.Name != "new-key" {
		t.Errorf("unexpected: %+v", key)
	}

	if err := c.SSHKeys.Delete(context.Background(), "2"); err != nil {
		t.Fatalf("Delete error: %v", err)
	}
}

func TestCreateSubnet(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"subnet": Subnet{ID: 3, Name: "sub-1", CIDR: "10.0.0.0/24", NetworkID: 1},
		})
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")
	subnet, err := c.Subnets.Create(context.Background(), &CreateSubnetRequest{
		Name: "sub-1", CIDR: "10.0.0.0/24", NetworkID: 1,
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if subnet.CIDR != "10.0.0.0/24" {
		t.Errorf("unexpected CIDR: %s", subnet.CIDR)
	}
}

func TestVolumeAttachDetach(t *testing.T) {
	var lastPath, lastMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		lastMethod = r.Method
		w.WriteHeader(204)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	if err := c.Volumes.Attach(context.Background(), "10", 5); err != nil {
		t.Fatalf("Attach error: %v", err)
	}
	if lastPath != "/v1/storage/volumes/10/attach" || lastMethod != http.MethodPost {
		t.Errorf("unexpected: %s %s", lastMethod, lastPath)
	}

	if err := c.Volumes.Detach(context.Background(), "10", "5"); err != nil {
		t.Fatalf("Detach error: %v", err)
	}
	if lastPath != "/v1/storage/volumes/10/detach" {
		t.Errorf("unexpected path: %s", lastPath)
	}
}

func TestFloatingIPAssociate(t *testing.T) {
	var lastPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		w.WriteHeader(204)
	}))
	defer ts.Close()

	c := NewClient(ts.URL)
	c.SetToken("test")

	if err := c.FloatingIPs.Associate(context.Background(), "7", 42); err != nil {
		t.Fatalf("Associate error: %v", err)
	}
	if lastPath != "/v1/floating-ips/7/associate" {
		t.Errorf("unexpected path: %s", lastPath)
	}

	if err := c.FloatingIPs.Disassociate(context.Background(), "7"); err != nil {
		t.Fatalf("Disassociate error: %v", err)
	}
	if lastPath != "/v1/floating-ips/7/disassociate" {
		t.Errorf("unexpected path: %s", lastPath)
	}
}

func TestRouterInitialized(t *testing.T) {
	c := NewClient("http://localhost")
	if c.Routers == nil {
		t.Error("expected Routers client to be initialized")
	}
}
