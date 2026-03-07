package orchestration

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

func init() { gin.SetMode(gin.TestMode) }

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) (*Service, *gin.Engine) {
	t.Helper()
	db := setupTestDB(t)
	svc, err := NewService(Config{DB: db, Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	router := gin.New()
	v1 := router.Group("/api/v1")
	svc.SetupRoutes(v1)
	return svc, router
}

func doRequest(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func parseJSON(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("parse json: %v (body: %s)", err, w.Body.String())
	}
	return result
}

// Simple two-tier web app template (network -> instance).
var sampleTemplate = `{
	"description": "Simple web application stack",
	"parameters": {
		"instance_name": {"type": "string", "default": "web-server-1", "description": "Name of the instance"}
	},
	"resources": {
		"app_network": {
			"type": "VC::Network::Network",
			"properties": {"name": "app-net", "cidr": "10.0.1.0/24"}
		},
		"app_subnet": {
			"type": "VC::Network::Subnet",
			"properties": {"name": "app-subnet", "network_id": {"ref": "app_network"}, "cidr": "10.0.1.0/24"},
			"depends_on": ["app_network"]
		},
		"web_server": {
			"type": "VC::Compute::Instance",
			"properties": {"name": "web-1", "flavor": "m1.small", "image": "ubuntu-22.04"},
			"depends_on": ["app_subnet"]
		}
	},
	"outputs": {
		"server_id": {"description": "ID of the web server", "value": {"ref": "web_server"}}
	}
}`

func createTestStack(t *testing.T, router *gin.Engine, name string) string {
	t.Helper()
	w := doRequest(t, router, http.MethodPost, "/api/v1/stacks", CreateStackRequest{
		Name:     name,
		Template: sampleTemplate,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create stack %s: %d %s", name, w.Code, w.Body.String())
	}
	return parseJSON(t, w)["stack"].(map[string]interface{})["id"].(string)
}

// TestCreateStack verifies stack creation with template parsing and resource creation.
func TestCreateStack(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/stacks", CreateStackRequest{
		Name:        "web-app-stack",
		Description: "My web application",
		Template:    sampleTemplate,
		Tags:        "env=dev,team=platform",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	stack := resp["stack"].(map[string]interface{})

	if stack["name"] != "web-app-stack" {
		t.Errorf("expected 'web-app-stack', got %s", stack["name"])
	}
	if stack["status"] != StackStatusCreateComplete {
		t.Errorf("expected CREATE_COMPLETE, got %s", stack["status"])
	}
	if stack["resource_count"].(float64) != 3 {
		t.Errorf("expected 3 resources, got %v", stack["resource_count"])
	}
	if stack["template_description"] != "Simple web application stack" {
		t.Errorf("expected template description, got %s", stack["template_description"])
	}
}

// TestCreateStack_InvalidTemplate tests template validation.
func TestCreateStack_InvalidTemplate(t *testing.T) {
	_, router := setupTestService(t)

	tests := []struct {
		name     string
		template string
	}{
		{"empty template", ""},
		{"invalid json", "{bad json"},
		{"no resources", `{"description":"empty","resources":{}}`},
		{"invalid resource type", `{"resources":{"x":{"type":"Invalid::Type"}}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := doRequest(t, router, http.MethodPost, "/api/v1/stacks", CreateStackRequest{
				Name:     "test-" + tt.name,
				Template: tt.template,
			})
			if w.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %q, got %d: %s", tt.name, w.Code, w.Body.String())
			}
		})
	}
}

// TestCircularDependency tests cycle detection.
func TestCircularDependency(t *testing.T) {
	_, router := setupTestService(t)

	circular := `{
		"resources": {
			"a": {"type": "VC::Network::Network", "depends_on": ["b"]},
			"b": {"type": "VC::Network::Subnet", "depends_on": ["a"]}
		}
	}`

	w := doRequest(t, router, http.MethodPost, "/api/v1/stacks", CreateStackRequest{
		Name:     "cycle-test",
		Template: circular,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for circular deps, got %d: %s", w.Code, w.Body.String())
	}
}

// TestUndefinedDependency tests referencing a non-existent resource.
func TestUndefinedDependency(t *testing.T) {
	_, router := setupTestService(t)

	tpl := `{
		"resources": {
			"a": {"type": "VC::Network::Network", "depends_on": ["nonexistent"]}
		}
	}`

	w := doRequest(t, router, http.MethodPost, "/api/v1/stacks", CreateStackRequest{
		Name:     "undef-dep-test",
		Template: tpl,
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for undefined dep, got %d", w.Code)
	}
}

// TestDuplicateStack tests uniqueness enforcement.
func TestDuplicateStack(t *testing.T) {
	_, router := setupTestService(t)

	createTestStack(t, router, "unique-stack")
	w := doRequest(t, router, http.MethodPost, "/api/v1/stacks", CreateStackRequest{
		Name:     "unique-stack",
		Template: sampleTemplate,
	})
	if w.Code != http.StatusConflict {
		t.Errorf("expected 409 for duplicate, got %d", w.Code)
	}
}

// TestGetStack verifies fetching stack details.
func TestGetStack(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestStack(t, router, "get-test-stack")

	w := doRequest(t, router, http.MethodGet, "/api/v1/stacks/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	stack := resp["stack"].(map[string]interface{})
	if stack["name"] != "get-test-stack" {
		t.Errorf("expected 'get-test-stack', got %s", stack["name"])
	}
	if resp["resource_count"].(float64) != 3 {
		t.Errorf("expected 3 resources, got %v", resp["resource_count"])
	}
}

// TestListResources verifies resource listing with dependency info.
func TestListResources(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestStack(t, router, "resource-list-stack")

	w := doRequest(t, router, http.MethodGet, "/api/v1/stacks/"+id+"/resources", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseJSON(t, w)
	resources := resp["resources"].([]interface{})
	if len(resources) != 3 {
		t.Fatalf("expected 3 resources, got %d", len(resources))
	}

	// Check types.
	types := map[string]bool{}
	for _, r := range resources {
		res := r.(map[string]interface{})
		types[res["type"].(string)] = true
		// All should be CREATE_COMPLETE.
		if res["status"] != ResourceStatusCreateComplete {
			t.Errorf("expected CREATE_COMPLETE, got %s", res["status"])
		}
		// All should have physical IDs.
		if res["physical_id"] == nil || res["physical_id"] == "" {
			t.Error("expected physical_id to be set")
		}
	}
	if !types[ResourceTypeNetwork] {
		t.Error("expected VC::Network::Network resource")
	}
	if !types[ResourceTypeSubnet] {
		t.Error("expected VC::Network::Subnet resource")
	}
	if !types[ResourceTypeInstance] {
		t.Error("expected VC::Compute::Instance resource")
	}
}

// TestListEvents verifies event timeline.
func TestListEvents(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestStack(t, router, "event-test-stack")

	w := doRequest(t, router, http.MethodGet, "/api/v1/stacks/"+id+"/events", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	resp := parseJSON(t, w)
	events := resp["events"].([]interface{})
	// 3 resource events + 1 stack event = 4.
	if len(events) < 4 {
		t.Errorf("expected at least 4 events, got %d", len(events))
	}

	// Check that stack-level event exists.
	found := false
	for _, e := range events {
		evt := e.(map[string]interface{})
		if evt["resource_type"] == "VC::Orchestration::Stack" {
			found = true
			if evt["status"] != StackStatusCreateComplete {
				t.Errorf("expected CREATE_COMPLETE event, got %s", evt["status"])
			}
		}
	}
	if !found {
		t.Error("expected stack-level CREATE_COMPLETE event")
	}
}

// TestUpdateStack verifies stack update.
func TestUpdateStack(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestStack(t, router, "update-test-stack")

	w := doRequest(t, router, http.MethodPut, "/api/v1/stacks/"+id, UpdateStackRequest{
		Description: "Updated description",
		Tags:        "env=prod",
		TimeoutMins: 120,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	stack := resp["stack"].(map[string]interface{})
	if stack["status"] != StackStatusUpdateComplete {
		t.Errorf("expected UPDATE_COMPLETE, got %s", stack["status"])
	}
	if stack["description"] != "Updated description" {
		t.Errorf("expected updated description")
	}
}

// TestDeleteStack verifies stack deletion with resource cleanup.
func TestDeleteStack(t *testing.T) {
	svc, router := setupTestService(t)
	id := createTestStack(t, router, "delete-test-stack")

	w := doRequest(t, router, http.MethodDelete, "/api/v1/stacks/"+id, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Verify soft-deleted.
	var stack Stack
	svc.db.First(&stack, "id = ?", id)
	if stack.Status != StackStatusDeleteComplete {
		t.Errorf("expected DELETE_COMPLETE, got %s", stack.Status)
	}

	// GET should now 404.
	w = doRequest(t, router, http.MethodGet, "/api/v1/stacks/"+id, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 after delete, got %d", w.Code)
	}

	// Resources should be cleaned up.
	var resourceCount int64
	svc.db.Model(&StackResource{}).Where("stack_id = ?", id).Count(&resourceCount)
	if resourceCount != 0 {
		t.Errorf("expected 0 resources after delete, got %d", resourceCount)
	}
}

// TestPreviewStack verifies dry-run template analysis.
func TestPreviewStack(t *testing.T) {
	_, router := setupTestService(t)

	w := doRequest(t, router, http.MethodPost, "/api/v1/stacks/preview", CreateStackRequest{
		Name:     "preview-test",
		Template: sampleTemplate,
	})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	preview := resp["preview"].(map[string]interface{})
	resources := preview["resources"].([]interface{})
	if len(resources) != 3 {
		t.Errorf("expected 3 preview resources, got %d", len(resources))
	}

	// First resource should be app_network (no deps).
	first := resources[0].(map[string]interface{})
	if first["logical_id"] != "app_network" {
		t.Errorf("expected app_network first (topological order), got %s", first["logical_id"])
	}
	if first["action"] != "CREATE" {
		t.Errorf("expected CREATE action, got %s", first["action"])
	}
}

// TestTemplateLibrary verifies the reusable template CRUD.
func TestTemplateLibrary(t *testing.T) {
	_, router := setupTestService(t)

	// Create template.
	w := doRequest(t, router, http.MethodPost, "/api/v1/templates", CreateTemplateRequest{
		Name:        "Basic Web App",
		Description: "A simple web application template",
		Template:    sampleTemplate,
		Version:     "2.0",
		IsPublic:    true,
		Category:    "web",
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON(t, w)
	tpl := resp["template"].(map[string]interface{})
	tplID := tpl["id"].(string)
	if tpl["name"] != "Basic Web App" {
		t.Errorf("expected 'Basic Web App', got %s", tpl["name"])
	}
	if tpl["version"] != "2.0" {
		t.Errorf("expected version 2.0, got %s", tpl["version"])
	}

	// List templates.
	w = doRequest(t, router, http.MethodGet, "/api/v1/templates?category=web", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp = parseJSON(t, w)
	templates := resp["templates"].([]interface{})
	if len(templates) != 1 {
		t.Errorf("expected 1 template, got %d", len(templates))
	}

	// Get template.
	w = doRequest(t, router, http.MethodGet, "/api/v1/templates/"+tplID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	// Delete template.
	w = doRequest(t, router, http.MethodDelete, "/api/v1/templates/"+tplID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// TestTopologicalSort verifies resource ordering.
func TestTopologicalSort(t *testing.T) {
	resources := map[string]TemplateResource{
		"c": {Type: ResourceTypeInstance, DependsOn: []string{"b"}},
		"a": {Type: ResourceTypeNetwork},
		"b": {Type: ResourceTypeSubnet, DependsOn: []string{"a"}},
	}

	order := topologicalSort(resources)
	if len(order) != 3 {
		t.Fatalf("expected 3, got %d", len(order))
	}

	// Check that dependencies come before dependents.
	indexOf := map[string]int{}
	for i, v := range order {
		indexOf[v] = i
	}

	if indexOf["a"] >= indexOf["b"] {
		t.Error("a must come before b")
	}
	if indexOf["b"] >= indexOf["c"] {
		t.Error("b must come before c")
	}
}

// TestResourceTypeValidation verifies supported types.
func TestResourceTypeValidation(t *testing.T) {
	valid := []string{
		ResourceTypeInstance, ResourceTypeVolume, ResourceTypeNetwork,
		ResourceTypeSubnet, ResourceTypeSecurityGroup, ResourceTypeFloatingIP,
		ResourceTypeDNSZone, ResourceTypeDNSRecord, ResourceTypeBucket,
		ResourceTypeRouter, ResourceTypeKeypair,
	}
	for _, rt := range valid {
		if !isValidResourceType(rt) {
			t.Errorf("expected %q to be valid", rt)
		}
	}

	invalid := []string{"Invalid::Type", "AWS::EC2::Instance", "VC::Unknown::Thing"}
	for _, rt := range invalid {
		if isValidResourceType(rt) {
			t.Errorf("expected %q to be invalid", rt)
		}
	}
}

// TestGetStackTemplate verifies template retrieval from running stack.
func TestGetStackTemplate(t *testing.T) {
	_, router := setupTestService(t)
	id := createTestStack(t, router, "template-retrieval-stack")

	w := doRequest(t, router, http.MethodGet, "/api/v1/stacks/"+id+"/template", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	resp := parseJSON(t, w)
	if resp["template"] == nil || resp["template"] == "" {
		t.Error("expected template content")
	}
}
