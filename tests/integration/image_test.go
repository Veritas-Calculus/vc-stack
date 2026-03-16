package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/internal/management/image"
	"github.com/Veritas-Calculus/vc-stack/tests/integration"
)

// TestImageCRUD verifies full CRUD lifecycle for images.
func TestImageCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := image.NewService(image.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	var createdUUID string

	t.Run("create image", func(t *testing.T) {
		body := `{"name":"ubuntu-22.04","disk_format":"qcow2","os_type":"linux","os_version":"ubuntu-22.04","min_disk":10}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/v1/images", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		img := resp["image"].(map[string]interface{})
		assert.Equal(t, "ubuntu-22.04", img["name"])
		assert.Equal(t, "qcow2", img["disk_format"])
		assert.Equal(t, "private", img["visibility"])
		assert.Equal(t, "queued", img["status"])
		assert.NotEmpty(t, img["uuid"])
		createdUUID = img["uuid"].(string)
	})

	t.Run("list images", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/images", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		images := resp["images"].([]interface{})
		assert.GreaterOrEqual(t, len(images), 1)
	})

	t.Run("list images with filter", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/images?os_type=linux&disk_format=qcow2", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		images := resp["images"].([]interface{})
		assert.GreaterOrEqual(t, len(images), 1)
	})

	t.Run("get image by uuid", func(t *testing.T) {
		require.NotEmpty(t, createdUUID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/images/"+createdUUID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		img := resp["image"].(map[string]interface{})
		assert.Equal(t, createdUUID, img["uuid"])
	})

	t.Run("update image metadata", func(t *testing.T) {
		require.NotEmpty(t, createdUUID)
		body := `{"description":"Ubuntu 22.04 LTS Cloud Image","visibility":"public"}`
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("PUT", "/api/v1/images/"+createdUUID, bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]interface{}
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		img := resp["image"].(map[string]interface{})
		assert.Equal(t, "public", img["visibility"])
	})

	t.Run("delete image", func(t *testing.T) {
		require.NotEmpty(t, createdUUID)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("DELETE", "/api/v1/images/"+createdUUID, http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		// Confirm it's gone.
		w2 := httptest.NewRecorder()
		req2, _ := http.NewRequest("GET", "/api/v1/images/"+createdUUID, http.NoBody)
		fixture.Router.ServeHTTP(w2, req2)
		assert.Equal(t, http.StatusNotFound, w2.Code)
	})

	t.Run("get nonexistent image returns 404", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/images/nonexistent-uuid", http.NoBody)
		fixture.Router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

// TestImageProtectedDelete verifies that protected images cannot be deleted.
func TestImageProtectedDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	fixture := integration.NewFixture(t, ctx)
	defer fixture.Teardown()

	svc, err := image.NewService(image.Config{
		DB:     fixture.DB,
		Logger: zap.NewNop(),
	})
	require.NoError(t, err)
	svc.SetupRoutes(fixture.Router)

	// Create a protected image.
	body := `{"name":"protected-img","disk_format":"qcow2","protected":true}`
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/images", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	fixture.Router.ServeHTTP(w, req)
	require.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	img := resp["image"].(map[string]interface{})
	uuid := img["uuid"].(string)

	// Attempt delete — should be rejected.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("DELETE", "/api/v1/images/"+uuid, http.NoBody)
	fixture.Router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusForbidden, w2.Code)
}
