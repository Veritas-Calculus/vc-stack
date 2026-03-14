package identity

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// Old IDP handlers removed — see federation.go for full implementation.

// Profile Handlers.
func (s *Service) getProfileHandler(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	uid, ok := uidVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}
	var user User
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (s *Service) updateProfileHandler(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	uid, ok := uidVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}
	var user User
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	var req struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	updates := map[string]interface{}{}
	if req.FirstName != "" {
		updates["first_name"] = req.FirstName
	}
	if req.LastName != "" {
		updates["last_name"] = req.LastName
	}
	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			s.logger.Error("Failed to update profile", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (s *Service) changePasswordHandler(c *gin.Context) {
	uidVal, _ := c.Get("user_id")
	uid, ok := uidVal.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}
	var user User
	if err := s.db.First(&user, uid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	var req struct {
		Current string `json:"current" binding:"required"`
		New     string `json:"new" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Current)) != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Current password incorrect"})
		return
	}
	// Use cost 12 for better security (default is 10)
	hashed, err := bcrypt.GenerateFromPassword([]byte(req.New), 12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}
	if err := s.db.Model(&user).Update("password", string(hashed)).Error; err != nil {
		s.logger.Error("Failed to change password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password updated"})
}

// Policy handlers.

func (s *Service) listPoliciesHandler(c *gin.Context) {
	var policies []Policy
	if err := s.db.Find(&policies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list policies"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policies": policies})
}

func (s *Service) createPolicyHandler(c *gin.Context) {
	var req struct {
		Name        string  `json:"name" binding:"required"`
		Description string  `json:"description"`
		Document    JSONMap `json:"document" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	policy := &Policy{
		Name:        req.Name,
		Description: req.Description,
		Document:    req.Document,
		Type:        "custom",
	}

	if err := s.db.Create(policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) getPolicyHandler(c *gin.Context) {
	id := c.Param("id")
	var policy Policy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) updatePolicyHandler(c *gin.Context) {
	id := c.Param("id")
	var policy Policy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	if policy.Type == "system" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system policy"})
		return
	}

	var req struct {
		Description string  `json:"description"`
		Document    JSONMap `json:"document"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Document != nil {
		updates["document"] = req.Document
	}

	if err := s.db.Model(&policy).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"policy": policy})
}

func (s *Service) deletePolicyHandler(c *gin.Context) {
	id := c.Param("id")
	var policy Policy
	if err := s.db.First(&policy, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "policy not found"})
		return
	}

	if policy.Type == "system" {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system policy"})
		return
	}

	if err := s.db.Delete(&policy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) attachUserPolicyHandler(c *gin.Context) {
	userID := c.Param("id")
	policyID := c.Param("policyId")

	if err := s.db.Exec("INSERT INTO user_policies (user_id, policy_id, created_at) VALUES (?, ?, ?)", userID, policyID, time.Now()).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to attach policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (s *Service) detachUserPolicyHandler(c *gin.Context) {
	userID := c.Param("id")
	policyID := c.Param("policyId")

	if err := s.db.Exec("DELETE FROM user_policies WHERE user_id = ? AND policy_id = ?", userID, policyID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to detach policy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
