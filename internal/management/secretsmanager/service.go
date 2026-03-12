// Package secretsmanager provides an independent Secrets Manager service.
//
// Unlike the ENC() flat-reference pattern, this provides full API-driven
// secret lifecycle management with versioning and automatic rotation.
package secretsmanager

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
)

// ──────────────────────────────────────────────────────────────────────
// Models
// ──────────────────────────────────────────────────────────────────────

// Secret represents a managed secret.
type Secret struct {
	ID          uint            `json:"id" gorm:"primarykey"`
	Name        string          `json:"name" gorm:"uniqueIndex;not null"`
	Description string          `json:"description"`
	ProjectID   uint            `json:"project_id" gorm:"index"`
	VersionID   int             `json:"version_id" gorm:"default:1"`
	RotateAfter int             `json:"rotate_after_days,omitempty"` // 0 = no auto-rotate
	LastRotated *time.Time      `json:"last_rotated,omitempty"`
	Versions    []SecretVersion `json:"versions,omitempty" gorm:"foreignKey:SecretID"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// SecretVersion stores a versioned secret value.
type SecretVersion struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	SecretID  uint      `json:"secret_id" gorm:"index;not null"`
	Version   int       `json:"version"`
	Value     string    `json:"-" gorm:"type:text;not null"`     // never serialized to JSON
	Status    string    `json:"status" gorm:"default:'current'"` // current, previous, deprecated
	CreatedAt time.Time `json:"created_at"`
}

// ──────────────────────────────────────────────────────────────────────
// Service
// ──────────────────────────────────────────────────────────────────────

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	if err := cfg.DB.AutoMigrate(&Secret{}, &SecretVersion{}); err != nil {
		return nil, fmt.Errorf("secretsmanager auto-migrate: %w", err)
	}
	return &Service{db: cfg.DB, logger: cfg.Logger}, nil
}

// ── Secret CRUD ──────────────────────────────────────────────

func (s *Service) Create(projectID uint, name, description, value string, rotateDays int) (*Secret, error) {
	sec := &Secret{
		Name: name, Description: description, ProjectID: projectID,
		VersionID: 1, RotateAfter: rotateDays,
	}
	return sec, s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(sec).Error; err != nil {
			return err
		}
		v := &SecretVersion{SecretID: sec.ID, Version: 1, Value: value, Status: "current"}
		return tx.Create(v).Error
	})
}

func (s *Service) List(projectID uint) ([]Secret, error) {
	var secrets []Secret
	q := s.db.Order("created_at DESC")
	if projectID > 0 {
		q = q.Where("project_id = ?", projectID)
	}
	return secrets, q.Find(&secrets).Error
}

func (s *Service) Get(id uint) (*Secret, error) {
	var sec Secret
	return &sec, s.db.First(&sec, id).Error
}

// GetValue retrieves the current version's value.
func (s *Service) GetValue(id uint) (string, int, error) {
	var v SecretVersion
	if err := s.db.Where("secret_id = ? AND status = ?", id, "current").First(&v).Error; err != nil {
		return "", 0, err
	}
	return v.Value, v.Version, nil
}

// PutValue creates a new version of the secret.
func (s *Service) PutValue(id uint, newValue string) (*SecretVersion, error) {
	var sec Secret
	if err := s.db.First(&sec, id).Error; err != nil {
		return nil, err
	}

	var v SecretVersion
	return &v, s.db.Transaction(func(tx *gorm.DB) error {
		// Mark current -> previous
		tx.Model(&SecretVersion{}).Where("secret_id = ? AND status = ?", id, "current").Update("status", "previous")
		// Create new
		newVer := sec.VersionID + 1
		v = SecretVersion{SecretID: id, Version: newVer, Value: newValue, Status: "current"}
		tx.Model(&Secret{}).Where("id = ?", id).Updates(map[string]interface{}{"version_id": newVer, "last_rotated": time.Now()})
		return tx.Create(&v).Error
	})
}

// Rotate generates a random new value for the secret.
func (s *Service) Rotate(id uint) (*SecretVersion, error) {
	b := make([]byte, 32)
	rand.Read(b)
	return s.PutValue(id, hex.EncodeToString(b))
}

func (s *Service) Delete(id uint) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("secret_id = ?", id).Delete(&SecretVersion{})
		return tx.Delete(&Secret{}, id).Error
	})
}

// ──────────────────────────────────────────────────────────────────────
// HTTP Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) SetupRoutes(router *gin.Engine, jwtSecret string) {
	api := router.Group("/api/v1/secrets")
	api.Use(middleware.AuthMiddleware(jwtSecret, s.logger))
	{
		api.GET("", s.handleList)
		api.POST("", s.handleCreate)
		api.GET("/:id", s.handleGet)
		api.DELETE("/:id", s.handleDelete)
		api.GET("/:id/value", s.handleGetValue)
		api.PUT("/:id/value", s.handlePutValue)
		api.POST("/:id/rotate", s.handleRotate)
	}
}

func (s *Service) handleList(c *gin.Context) {
	secs, err := s.List(0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secrets": secs})
}

func (s *Service) handleCreate(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Value       string `json:"value" binding:"required"`
		RotateDays  int    `json:"rotate_after_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sec, err := s.Create(0, req.Name, req.Description, req.Value, req.RotateDays)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"secret": sec})
}

func (s *Service) handleGet(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	sec, err := s.Get(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"secret": sec})
}

func (s *Service) handleDelete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if err := s.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusNoContent, nil)
}

func (s *Service) handleGetValue(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	val, ver, err := s.GetValue(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"value": val, "version": ver})
}

func (s *Service) handlePutValue(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := s.PutValue(uint(id), req.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}

func (s *Service) handleRotate(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	v, err := s.Rotate(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": v})
}
