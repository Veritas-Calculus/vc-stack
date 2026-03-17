package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (s *Service) setupSecurityRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/security-groups")
	{
		api.GET("", s.listSecurityGroups)
		api.POST("", s.createSecurityGroup)
		api.POST("/:id/rules", s.addSecurityGroupRule)
		api.DELETE("/:id/rules/:ruleId", s.removeSecurityGroupRule)
	}
}

func (s *Service) listSecurityGroups(c *gin.Context) {
	var groups []SecurityGroup
	s.db.Preload("Rules").Find(&groups)
	c.JSON(http.StatusOK, groups)
}

func (s *Service) createSecurityGroup(c *gin.Context) {
	var sg SecurityGroup
	if err := c.ShouldBindJSON(&sg); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	sg.ID = uuid.New().String()
	s.db.Create(&sg)
	c.JSON(http.StatusCreated, sg)
}

func (s *Service) addSecurityGroupRule(c *gin.Context) {
	sgID := c.Param("id")
	var rule SecurityGroupRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	rule.ID = uuid.New().String()
	rule.SecurityGroupID = sgID
	s.db.Create(&rule)

	// In a real implementation, this would trigger a re-sync of all ports using this group
	s.syncSecurityGroup(sgID)

	c.JSON(http.StatusCreated, rule)
}

func (s *Service) syncSecurityGroup(id string) {
	var sg SecurityGroup
	if err := s.db.Preload("Rules").First(&sg, "id = ?", id).Error; err != nil {
		s.logger.Error("failed to load SG for sync", zap.String("id", id), zap.Error(err))
		return
	}

	// 1. Ensure the Port Group exists in OVN
	_ = s.driver.EnsurePortGroup(sg.Name)

	// 2. Compile rules
	aclRules := CompileGroup(&sg)

	// 3. Update ACLs in OVN
	if err := s.driver.SetPortGroupACLs(sg.Name, aclRules); err != nil {
		s.logger.Error("failed to sync ACLs to OVN", zap.String("sg", sg.Name), zap.Error(err))
	}
}

func (s *Service) removeSecurityGroupRule(c *gin.Context) {
	ruleID := c.Param("ruleId")
	s.db.Delete(&SecurityGroupRule{}, "id = ?", ruleID)
	c.Status(http.StatusNoContent)
}
