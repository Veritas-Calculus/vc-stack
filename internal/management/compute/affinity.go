package compute

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AffinityGroup represents a VM placement constraint group.
type AffinityGroup struct {
	ID          uint                  `gorm:"primaryKey" json:"id"`
	Name        string                `gorm:"not null;uniqueIndex" json:"name"`
	Description string                `json:"description"`
	Type        string                `gorm:"not null;default:'host-anti-affinity'" json:"type"` // host-affinity, host-anti-affinity
	UserID      uint                  `json:"user_id"`
	ProjectID   uint                  `json:"project_id"`
	Members     []AffinityGroupMember `gorm:"foreignKey:GroupID" json:"members,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// AffinityGroupMember links an instance to an affinity group.
type AffinityGroupMember struct {
	ID         uint `gorm:"primaryKey" json:"id"`
	GroupID    uint `gorm:"not null;index;uniqueIndex:idx_ag_member" json:"group_id"`
	InstanceID uint `gorm:"not null;index;uniqueIndex:idx_ag_member" json:"instance_id"`
}

// migrateAffinityGroups runs auto-migration for affinity group tables.
func (s *Service) migrateAffinityGroups() error {
	return s.db.AutoMigrate(&AffinityGroup{}, &AffinityGroupMember{})
}

// --- AffinityGroup handlers ---

func (s *Service) listAffinityGroups(c *gin.Context) {
	var groups []AffinityGroup
	query := s.db.Preload("Members").Order("name")
	if projectID := c.Query("project_id"); projectID != "" {
		query = query.Where("project_id = ?", projectID)
	}
	if err := query.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list affinity groups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"affinity_groups": groups})
}

func (s *Service) createAffinityGroup(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		Type        string `json:"type"`
		ProjectID   uint   `json:"project_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	agType := req.Type
	if agType == "" {
		agType = "host-anti-affinity"
	}
	if agType != "host-affinity" && agType != "host-anti-affinity" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type must be 'host-affinity' or 'host-anti-affinity'"})
		return
	}

	group := AffinityGroup{
		Name:        req.Name,
		Description: req.Description,
		Type:        agType,
		ProjectID:   req.ProjectID,
	}
	if err := s.db.Create(&group).Error; err != nil {
		s.logger.Error("failed to create affinity group", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create affinity group"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"affinity_group": group})
}

func (s *Service) deleteAffinityGroup(c *gin.Context) {
	id := c.Param("id")
	// Remove all members first
	s.db.Where("group_id = ?", id).Delete(&AffinityGroupMember{})
	if err := s.db.Delete(&AffinityGroup{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete affinity group"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) addAffinityGroupMember(c *gin.Context) {
	groupID := c.Param("id")
	var req struct {
		InstanceID uint `json:"instance_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var gid uint
	if err := s.db.Model(&AffinityGroup{}).Select("id").Where("id = ?", groupID).Scan(&gid).Error; err != nil || gid == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "affinity group not found"})
		return
	}

	member := AffinityGroupMember{GroupID: gid, InstanceID: req.InstanceID}
	if err := s.db.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add member"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"member": member})
}

func (s *Service) removeAffinityGroupMember(c *gin.Context) {
	memberID := c.Param("memberId")
	if err := s.db.Delete(&AffinityGroupMember{}, memberID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
