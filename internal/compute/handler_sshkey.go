package compute

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SSH Key Handlers.
func (s *Service) listSSHKeysHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	q := s.db.Model(&SSHKey{}).Where("user_id = ?", userID)
	if pid := s.getProjectIDFromContext(c); pid != 0 {
		q = q.Where("project_id = ?", pid)
	}
	var keys []SSHKey
	if err := q.Find(&keys).Error; err != nil {
		s.logger.Error("Failed to list ssh keys", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list ssh keys"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ssh_keys": keys})
}

func (s *Service) createSSHKeyHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	projectID := s.getProjectIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	var req struct {
		Name      string `json:"name" binding:"required"`
		PublicKey string `json:"public_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}
	// Basic key validation: starts with ssh- or ecdsa/ed25519... and has at least 2 parts
	if len(req.PublicKey) < 20 || (!startsWithAny(req.PublicKey, "ssh-", "ecdsa-", "sk-", "ed25519")) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid SSH public key"})
		return
	}
	key := &SSHKey{Name: req.Name, PublicKey: req.PublicKey, UserID: userID, ProjectID: projectID}
	if err := s.db.Create(key).Error; err != nil {
		s.logger.Error("Failed to create ssh key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create ssh key"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ssh_key": key})
}

func (s *Service) deleteSSHKeyHandler(c *gin.Context) {
	userID := s.getUserIDFromContext(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).Delete(&SSHKey{}).Error; err != nil {
		s.logger.Error("Failed to delete ssh key", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete ssh key"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "SSH key deleted"})
}
