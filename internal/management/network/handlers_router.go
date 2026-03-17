package network

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Service) setupRouterRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/routers")
	{
		api.GET("", s.listRouters)
		api.POST("", s.createRouter)
		api.GET("/:id", s.getRouter)
		api.DELETE("/:id", s.deleteRouter)
		api.POST("/:id/add_interface", s.addRouterInterface)
		api.POST("/:id/remove_interface", s.removeRouterInterface)
	}
}

func (s *Service) listRouters(c *gin.Context) {
	var routers []Router
	s.db.Find(&routers)
	c.JSON(http.StatusOK, routers)
}

func (s *Service) createRouter(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	r := Router{
		ID:     uuid.New().String(),
		Name:   req.Name,
		Status: "active",
	}
	s.db.Create(&r)

	// Create in OVN
	_ = s.driver.EnsureRouter(r.Name)

	c.JSON(http.StatusCreated, r)
}

func (s *Service) getRouter(c *gin.Context) {
	id := c.Param("id")
	var r Router
	if err := s.db.First(&r, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusOK, r)
}

func (s *Service) deleteRouter(c *gin.Context) {
	id := c.Param("id")
	var r Router
	if err := s.db.First(&r, "id = ?", id).Error; err == nil {
		_ = s.driver.DeleteRouter(r.Name)
		s.db.Delete(&r)
	}
	c.Status(http.StatusNoContent)
}

func (s *Service) addRouterInterface(c *gin.Context) {
	routerID := c.Param("id")
	var req struct {
		SubnetID string `json:"subnet_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var r Router
	s.db.First(&r, "id = ?", routerID)
	var sn Subnet
	s.db.First(&sn, "id = ?", req.SubnetID)
	var n Network
	s.db.First(&n, "id = ?", sn.NetworkID)

	// Connect logical switch to logical router
	if err := s.driver.ConnectSubnetToRouter(r.Name, &n, &sn); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "SDN connection failed"})
		return
	}

	ri := RouterInterface{
		RouterID: r.ID,
		SubnetID: sn.ID,
	}
	s.db.Create(&ri)

	c.JSON(http.StatusOK, ri)
}

func (s *Service) removeRouterInterface(c *gin.Context) {
	// Implementation to disconnect
	c.Status(http.StatusNoContent)
}
