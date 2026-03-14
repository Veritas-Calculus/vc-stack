package identity

import (
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// listUsersHandler returns a list of users.
func (s *Service) listUsersHandler(c *gin.Context) {
	var users []User
	if err := s.db.Preload("Roles").Find(&users).Error; err != nil {
		s.logger.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// createUserHandler creates a new user.
func (s *Service) createUserHandler(c *gin.Context) {
	var req struct {
		Username  string `json:"username" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=8"` // #nosec // This is a password field in a request struct
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsAdmin   bool   `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &User{
		Username:  req.Username,
		Email:     req.Email,
		FirstName: req.FirstName,
		LastName:  req.LastName,
		IsAdmin:   req.IsAdmin,
		IsActive:  true,
	}

	if err := s.db.Create(user).Error; err != nil {
		s.logger.Error("Failed to create user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"user": user})
}

// getUserHandler returns a specific user.
func (s *Service) getUserHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user User
	if err := s.db.Preload("Roles.Permissions").First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// updateUserHandler updates a user.
func (s *Service) updateUserHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		IsActive  *bool  `json:"is_active"`
		IsAdmin   *bool  `json:"is_admin"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields.
	if req.Email != "" {
		user.Email = req.Email
	}
	if req.FirstName != "" {
		user.FirstName = req.FirstName
	}
	if req.LastName != "" {
		user.LastName = req.LastName
	}
	if req.IsActive != nil {
		user.IsActive = *req.IsActive
	}
	if req.IsAdmin != nil {
		user.IsAdmin = *req.IsAdmin
	}

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to update user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// deleteUserHandler deletes a user.
func (s *Service) deleteUserHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := s.db.Delete(&User{}, id).Error; err != nil {
		s.logger.Error("Failed to delete user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// updateUserStatusHandler updates a user's status (active/inactive/suspended).
func (s *Service) updateUserStatusHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Map status to IsActive flag; 'suspended' and 'inactive' both mean inactive for now.
	switch req.Status {
	case "active":
		user.IsActive = true
	case "inactive", "suspended":
		user.IsActive = false
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status"})
		return
	}

	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to update user status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user": user})
}

// resetUserPasswordHandler resets a user's password to a default for dev.
func (s *Service) resetUserPasswordHandler(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var user User
	if err := s.db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// For development, we set a known password. In production, generate and email.
	// Get default password from environment
	defaultPassword := os.Getenv("ADMIN_DEFAULT_PASSWORD")
	if defaultPassword == "" {
		defaultPassword = "ChangeMe123!" // This should be set via environment variable
		s.logger.Warn("SECURITY WARNING: Using fallback default password. Set ADMIN_DEFAULT_PASSWORD environment variable!")
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), 12) // Use cost 12 for better security
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}
	user.Password = string(hashed)
	if err := s.db.Save(&user).Error; err != nil {
		s.logger.Error("Failed to save reset password", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Password reset to default"})
}
