package identity

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (s *Service) listProjectsHandler(c *gin.Context) {
	var projects []Project
	// Optional filter: owner=true returns only projects owned by current user.
	q := s.db.Model(&Project{})
	if c.Query("owner") == "true" {
		if uidVal, exists := c.Get("user_id"); exists {
			if uid, ok := uidVal.(uint); ok {
				q = q.Where("user_id = ?", uid)
			}
		}
	}
	if err := q.Find(&projects).Error; err != nil {
		s.logger.Error("Failed to list projects", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list projects"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"projects": projects})
}

func (s *Service) createProjectHandler(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		UserID      uint   `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Default to current user if not provided.
	if req.UserID == 0 {
		if uidVal, exists := c.Get("user_id"); exists {
			if uid, ok := uidVal.(uint); ok {
				req.UserID = uid
			}
		}
	}
	p := &Project{Name: req.Name, Description: req.Description, UserID: req.UserID}
	if err := s.db.Create(p).Error; err != nil {
		s.logger.Error("Failed to create project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project": p})
}

func (s *Service) deleteProjectHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	if err := s.db.Delete(&Project{}, id).Error; err != nil {
		s.logger.Error("Failed to delete project", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete project"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Project deleted"})
}

// getProjectDetailHandler returns a project with its members.
func (s *Service) getProjectDetailHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var project Project
	if err := s.db.Preload("Members.User").First(&project, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Project not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project": project})
}

// listProjectMembersHandler returns members of a project.
func (s *Service) listProjectMembersHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var members []ProjectMember
	if err := s.db.Preload("User").Where("project_id = ?", id).Find(&members).Error; err != nil {
		s.logger.Error("Failed to list project members", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list members"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

// addProjectMemberHandler adds a user as a member to a project.
func (s *Service) addProjectMemberHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	role := req.Role
	if role == "" {
		role = "member"
	}
	member := &ProjectMember{
		ProjectID: uint(id),
		UserID:    req.UserID,
		Role:      role,
	}
	if err := s.db.Create(member).Error; err != nil {
		s.logger.Error("Failed to add project member", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add member"})
		return
	}
	// Reload with user
	s.db.Preload("User").First(member, member.ID)
	c.JSON(http.StatusCreated, gin.H{"member": member})
}

// updateProjectMemberHandler updates a member's role.
func (s *Service) updateProjectMemberHandler(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}
	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := s.db.Model(&ProjectMember{}).Where("id = ?", memberID).Update("role", req.Role).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update member role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// removeProjectMemberHandler removes a member from a project.
func (s *Service) removeProjectMemberHandler(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid member ID"})
		return
	}
	if err := s.db.Delete(&ProjectMember{}, memberID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove member"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// Quota handlers.
func (s *Service) getDefaultQuotaHandler(c *gin.Context) {
	var q Quota
	if err := s.db.Where("project_id IS NULL").First(&q).Error; err != nil {
		// Return sensible defaults if not set.
		q = Quota{VCPUs: 16, RAMMB: 32768, DiskGB: 500, Instances: 20}
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

func (s *Service) updateDefaultQuotaHandler(c *gin.Context) {
	var req Quota
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	var q Quota
	if err := s.db.Where("project_id IS NULL").First(&q).Error; err != nil {
		q = Quota{ProjectID: nil}
		q.VCPUs, q.RAMMB, q.DiskGB, q.Instances = req.VCPUs, req.RAMMB, req.DiskGB, req.Instances
		if err := s.db.Create(&q).Error; err != nil {
			s.logger.Error("Failed to set default quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default quota"})
			return
		}
	} else {
		updates := map[string]interface{}{"vcpus": req.VCPUs, "ram_mb": req.RAMMB, "disk_gb": req.DiskGB, "instances": req.Instances}
		if err := s.db.Model(&q).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update default quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update default quota"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

func (s *Service) getProjectQuotaHandler(c *gin.Context) {
	pid, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var q Quota
	if err := s.db.Where("project_id = ?", uint(pid)).First(&q).Error; err != nil {
		// inherit default.
		s.getDefaultQuotaHandler(c)
		return
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}

func (s *Service) updateProjectQuotaHandler(c *gin.Context) {
	pid, err := strconv.ParseUint(c.Param("projectId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}
	var req Quota
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	projectID := uint(pid)
	var q Quota
	if err := s.db.Where("project_id = ?", projectID).First(&q).Error; err != nil {
		q = Quota{ProjectID: &projectID, VCPUs: req.VCPUs, RAMMB: req.RAMMB, DiskGB: req.DiskGB, Instances: req.Instances}
		if err := s.db.Create(&q).Error; err != nil {
			s.logger.Error("Failed to create project quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project quota"})
			return
		}
	} else {
		updates := map[string]interface{}{"vcpus": req.VCPUs, "ram_mb": req.RAMMB, "disk_gb": req.DiskGB, "instances": req.Instances}
		if err := s.db.Model(&q).Where("project_id = ?", projectID).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update project quota", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update project quota"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"quota": q})
}
