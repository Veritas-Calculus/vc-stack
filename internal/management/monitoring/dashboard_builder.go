package monitoring

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ──────────────────────────────────────────────────────────────────────
// Custom Dashboard Builder
//
// Drag-and-drop widget-based dashboards with grid layout persistence.
// ──────────────────────────────────────────────────────────────────────

// CustomDashboard represents a user-created monitoring dashboard.
type CustomDashboard struct {
	ID          uint              `json:"id" gorm:"primarykey"`
	Name        string            `json:"name" gorm:"not null"`
	Description string            `json:"description"`
	OwnerID     string            `json:"owner_id" gorm:"index"`
	TenantID    string            `json:"tenant_id" gorm:"index"`
	IsShared    bool              `json:"is_shared" gorm:"default:false"`
	IsDefault   bool              `json:"is_default" gorm:"default:false"`
	Tags        string            `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Widgets     []DashboardWidget `json:"widgets,omitempty" gorm:"foreignKey:DashboardID"`
}

func (CustomDashboard) TableName() string { return "mon_dashboards" }

// DashboardWidget represents a single widget on a dashboard.
type DashboardWidget struct {
	ID          uint   `json:"id" gorm:"primarykey"`
	DashboardID uint   `json:"dashboard_id" gorm:"index;not null"`
	Title       string `json:"title" gorm:"not null"`
	Type        string `json:"type" gorm:"not null"` // line, bar, gauge, table, text, alert_status, pie, heatmap
	DataSource  string `json:"data_source"`          // prometheus, custom_metrics, influxdb
	Query       string `json:"query" gorm:"type:text"`
	// Grid position (12-column grid).
	PosX   int `json:"pos_x" gorm:"default:0"`
	PosY   int `json:"pos_y" gorm:"default:0"`
	Width  int `json:"width" gorm:"default:6"`
	Height int `json:"height" gorm:"default:4"`
	// Display config.
	Config    string    `json:"config,omitempty" gorm:"type:text"` // JSON: colors, thresholds, legends
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (DashboardWidget) TableName() string { return "mon_dashboard_widgets" }

// ── Route Setup ──

func SetupDashboardBuilderRoutes(api *gin.RouterGroup, db *gorm.DB, logger *zap.Logger) {
	svc := &dashboardBuilderSvc{db: db, logger: logger}
	d := api.Group("/dashboards")
	{
		d.GET("", svc.listDashboards)
		d.POST("", svc.createDashboard)
		d.GET("/:id", svc.getDashboard)
		d.PUT("/:id", svc.updateDashboard)
		d.DELETE("/:id", svc.deleteDashboard)
		d.POST("/:id/clone", svc.cloneDashboard)
		// Widgets.
		d.POST("/:id/widgets", svc.addWidget)
		d.PUT("/:id/widgets/:widgetId", svc.updateWidget)
		d.DELETE("/:id/widgets/:widgetId", svc.deleteWidget)
	}
}

type dashboardBuilderSvc struct {
	db     *gorm.DB
	logger *zap.Logger
}

func (s *dashboardBuilderSvc) listDashboards(c *gin.Context) {
	var dashboards []CustomDashboard
	query := s.db
	if tid := c.Query("tenant_id"); tid != "" {
		query = query.Where("tenant_id = ? OR is_shared = true", tid)
	}
	query.Find(&dashboards)
	c.JSON(http.StatusOK, gin.H{"dashboards": dashboards})
}

func (s *dashboardBuilderSvc) createDashboard(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
		TenantID    string `json:"tenant_id"`
		IsShared    bool   `json:"is_shared"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	d := CustomDashboard{
		Name: req.Name, Description: req.Description,
		TenantID: req.TenantID, IsShared: req.IsShared,
	}
	s.db.Create(&d)
	c.JSON(http.StatusCreated, gin.H{"dashboard": d})
}

func (s *dashboardBuilderSvc) getDashboard(c *gin.Context) {
	id := c.Param("id")
	var d CustomDashboard
	if err := s.db.Preload("Widgets").First(&d, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dashboard not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"dashboard": d})
}

func (s *dashboardBuilderSvc) updateDashboard(c *gin.Context) {
	id := c.Param("id")
	var d CustomDashboard
	if err := s.db.First(&d, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dashboard not found"})
		return
	}
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsShared    *bool  `json:"is_shared"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Name != "" {
		d.Name = req.Name
	}
	if req.Description != "" {
		d.Description = req.Description
	}
	if req.IsShared != nil {
		d.IsShared = *req.IsShared
	}
	s.db.Save(&d)
	c.JSON(http.StatusOK, gin.H{"dashboard": d})
}

func (s *dashboardBuilderSvc) deleteDashboard(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("dashboard_id = ?", id).Delete(&DashboardWidget{})
	s.db.Delete(&CustomDashboard{}, id)
	c.JSON(http.StatusOK, gin.H{"message": "Dashboard deleted"})
}

func (s *dashboardBuilderSvc) cloneDashboard(c *gin.Context) {
	id := c.Param("id")
	var src CustomDashboard
	if err := s.db.Preload("Widgets").First(&src, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Dashboard not found"})
		return
	}
	clone := CustomDashboard{
		Name: src.Name + " (copy)", Description: src.Description,
		TenantID: src.TenantID, IsShared: false,
	}
	s.db.Create(&clone)
	for _, w := range src.Widgets {
		wc := w
		wc.ID = 0
		wc.DashboardID = clone.ID
		s.db.Create(&wc)
	}
	s.db.Preload("Widgets").First(&clone, clone.ID)
	c.JSON(http.StatusCreated, gin.H{"dashboard": clone})
}

func (s *dashboardBuilderSvc) addWidget(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Title      string          `json:"title" binding:"required"`
		Type       string          `json:"type" binding:"required"`
		DataSource string          `json:"data_source"`
		Query      string          `json:"query"`
		PosX       int             `json:"pos_x"`
		PosY       int             `json:"pos_y"`
		Width      int             `json:"width"`
		Height     int             `json:"height"`
		Config     json.RawMessage `json:"config"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	dashID := uint(0)
	for _, ch := range id {
		if ch >= '0' && ch <= '9' {
			dashID = dashID*10 + uint(ch-'0')
		}
	}
	w := DashboardWidget{
		DashboardID: dashID, Title: req.Title, Type: req.Type,
		DataSource: req.DataSource, Query: req.Query,
		PosX: req.PosX, PosY: req.PosY,
		Width: req.Width, Height: req.Height,
		Config: string(req.Config),
	}
	if w.Width == 0 {
		w.Width = 6
	}
	if w.Height == 0 {
		w.Height = 4
	}
	s.db.Create(&w)
	c.JSON(http.StatusCreated, gin.H{"widget": w})
}

func (s *dashboardBuilderSvc) updateWidget(c *gin.Context) {
	wid := c.Param("widgetId")
	var w DashboardWidget
	if err := s.db.First(&w, wid).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Widget not found"})
		return
	}
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	s.db.Model(&w).Updates(req)
	s.db.First(&w, wid)
	c.JSON(http.StatusOK, gin.H{"widget": w})
}

func (s *dashboardBuilderSvc) deleteWidget(c *gin.Context) {
	wid := c.Param("widgetId")
	s.db.Delete(&DashboardWidget{}, wid)
	c.JSON(http.StatusOK, gin.H{"message": "Widget deleted"})
}
