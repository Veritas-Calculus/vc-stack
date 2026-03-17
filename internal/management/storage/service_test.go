package storage

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

func setupTestService() (*Service, *gin.Engine) {
	db, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	logger := zap.NewNop()

	// Perform migration for tests
	db.AutoMigrate(&models.Volume{}, &models.VolumeAttachment{})

	svc, _ := NewService(Config{
		DB:     db,
		Logger: logger,
	})

	// Force Noop driver for unit tests
	svc.driver = &NoopStorageDriver{logger: logger}

	r := gin.New()
	svc.SetupRoutes(r)

	return svc, r
}

func TestListVolumes(t *testing.T) {
	svc, r := setupTestService()

	// Seed a volume
	svc.db.Create(&models.Volume{Name: "test-vol", SizeGB: 10, Status: "available"})

	req, _ := http.NewRequest("GET", "/api/v1/storage/volumes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var volumes []models.Volume
	json.Unmarshal(w.Body.Bytes(), &volumes)
	assert.Len(t, volumes, 1)
	assert.Equal(t, "test-vol", volumes[0].Name)
}

func TestCreateVolume(t *testing.T) {
	_, r := setupTestService()

	payload := models.Volume{
		Name:   "new-volume",
		SizeGB: 20,
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/v1/storage/volumes", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}
