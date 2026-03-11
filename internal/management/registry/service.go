// Package registry implements a lightweight service discovery registry
// for VC Stack. Services register themselves with health endpoints,
// metadata, and tags. The registry tracks heartbeats and automatically
// deregisters unhealthy instances.
package registry

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/Veritas-Calculus/vc-stack/internal/management/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// ServiceInstance represents a registered service endpoint.
type ServiceInstance struct {
	ID            string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ServiceName   string     `json:"service_name" gorm:"not null;index"`
	Version       string     `json:"version"`
	Host          string     `json:"host" gorm:"not null"`
	Port          int        `json:"port" gorm:"not null"`
	Scheme        string     `json:"scheme" gorm:"default:'http'"`
	HealthPath    string     `json:"health_path" gorm:"default:'/health'"`
	Tags          string     `json:"tags" gorm:"type:text"`      // JSON array
	Metadata      string     `json:"metadata" gorm:"type:text"`  // JSON object
	Status        string     `json:"status" gorm:"default:'up'"` // up, down, draining, starting
	Weight        int        `json:"weight" gorm:"default:100"`  // load-balancing weight
	Zone          string     `json:"zone"`                       // availability zone
	Region        string     `json:"region"`
	LastHeartbeat *time.Time `json:"last_heartbeat"`
	RegisteredAt  time.Time  `json:"registered_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (ServiceInstance) TableName() string { return "registry_instances" }

// ServiceRoute defines an API route provided by a service.
type ServiceRoute struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ServiceName  string    `json:"service_name" gorm:"not null;index"`
	Method       string    `json:"method"` // GET, POST, PUT, DELETE, *
	PathPrefix   string    `json:"path_prefix" gorm:"not null"`
	Description  string    `json:"description"`
	AuthRequired bool      `json:"auth_required" gorm:"default:true"`
	RateLimit    int       `json:"rate_limit"` // requests per minute, 0 = unlimited
	CreatedAt    time.Time `json:"created_at"`
}

func (ServiceRoute) TableName() string { return "registry_routes" }

// ---------- Service ----------

type Config struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

type Service struct {
	db     *gorm.DB
	logger *zap.Logger
}

func NewService(cfg Config) (*Service, error) {
	s := &Service{db: cfg.DB, logger: cfg.Logger}
	if err := cfg.DB.AutoMigrate(&ServiceInstance{}, &ServiceRoute{}); err != nil {
		return nil, fmt.Errorf("registry: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Service registry initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	now := time.Now()
	instances := []ServiceInstance{
		{ID: uuid.New().String(), ServiceName: "vc-management", Version: "1.0.0",
			Host: "localhost", Port: 8080, Scheme: "http", HealthPath: "/health",
			Tags: `["management","api","web"]`, Metadata: `{"binary":"vc-management","pid":1}`,
			Status: "up", Weight: 100, Zone: "az-1", Region: "dc-primary",
			LastHeartbeat: &now, RegisteredAt: now},
		{ID: uuid.New().String(), ServiceName: "vc-compute", Version: "1.0.0",
			Host: "compute-01", Port: 8081, Scheme: "http", HealthPath: "/health",
			Tags: `["compute","hypervisor","kvm"]`, Metadata: `{"binary":"vc-compute","hypervisor":"qemu-kvm","cpu_cores":64}`,
			Status: "up", Weight: 100, Zone: "az-1", Region: "dc-primary",
			LastHeartbeat: &now, RegisteredAt: now},
		{ID: uuid.New().String(), ServiceName: "vc-compute", Version: "1.0.0",
			Host: "compute-02", Port: 8081, Scheme: "http", HealthPath: "/health",
			Tags: `["compute","hypervisor","kvm","gpu"]`, Metadata: `{"binary":"vc-compute","hypervisor":"qemu-kvm","cpu_cores":128,"gpu":"A100"}`,
			Status: "up", Weight: 100, Zone: "az-2", Region: "dc-primary",
			LastHeartbeat: &now, RegisteredAt: now},
		{ID: uuid.New().String(), ServiceName: "postgres", Version: "16.2",
			Host: "postgres", Port: 5432, Scheme: "tcp", HealthPath: "",
			Tags: `["database","primary"]`, Metadata: `{"engine":"PostgreSQL","role":"primary"}`,
			Status: "up", Weight: 100, Zone: "az-1", Region: "dc-primary",
			LastHeartbeat: &now, RegisteredAt: now},
		{ID: uuid.New().String(), ServiceName: "ovn-central", Version: "23.09",
			Host: "ovn-central-01", Port: 6641, Scheme: "tcp", HealthPath: "",
			Tags: `["sdn","ovn","networking"]`, Metadata: `{"component":"ovn-northd","nb_port":6641,"sb_port":6642}`,
			Status: "up", Weight: 100, Zone: "az-1", Region: "dc-primary",
			LastHeartbeat: &now, RegisteredAt: now},
	}
	for i := range instances {
		s.db.Where("service_name = ? AND host = ? AND port = ?",
			instances[i].ServiceName, instances[i].Host, instances[i].Port).FirstOrCreate(&instances[i])
	}

	routes := []ServiceRoute{
		{ID: uuid.New().String(), ServiceName: "vc-management", Method: "*", PathPrefix: "/api/v1/auth", Description: "Authentication", AuthRequired: false, RateLimit: 30},
		{ID: uuid.New().String(), ServiceName: "vc-management", Method: "*", PathPrefix: "/api/v1/projects", Description: "Project Management", AuthRequired: true, RateLimit: 100},
		{ID: uuid.New().String(), ServiceName: "vc-management", Method: "*", PathPrefix: "/api/v1/identity", Description: "Identity & RBAC", AuthRequired: true, RateLimit: 100},
		{ID: uuid.New().String(), ServiceName: "vc-management", Method: "*", PathPrefix: "/api/v1/networks", Description: "Network Management", AuthRequired: true, RateLimit: 60},
		{ID: uuid.New().String(), ServiceName: "vc-management", Method: "*", PathPrefix: "/api/v1/dns", Description: "DNS Management", AuthRequired: true, RateLimit: 60},
		{ID: uuid.New().String(), ServiceName: "vc-compute", Method: "*", PathPrefix: "/api/v1/instances", Description: "VM Lifecycle", AuthRequired: true, RateLimit: 30},
		{ID: uuid.New().String(), ServiceName: "vc-compute", Method: "*", PathPrefix: "/api/v1/volumes", Description: "Volume Management", AuthRequired: true, RateLimit: 60},
	}
	for i := range routes {
		s.db.Where("service_name = ? AND path_prefix = ?", routes[i].ServiceName, routes[i].PathPrefix).FirstOrCreate(&routes[i])
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/registry")
	{
		api.GET("/status", rp("registry", "list"), s.getStatus)
		api.GET("/services", rp("registry", "list"), s.listServices)
		api.GET("/services/:name", rp("registry", "get"), s.getService)
		api.POST("/register", rp("registry", "create"), s.register)
		api.DELETE("/deregister/:id", rp("registry", "delete"), s.deregister)
		api.PUT("/heartbeat/:id", rp("registry", "update"), s.heartbeat)
		api.PUT("/instances/:id/drain", rp("registry", "update"), s.drain)
		api.GET("/routes", rp("registry", "list"), s.listRoutes)
		api.GET("/topology", rp("registry", "list"), s.getTopology)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var total, up, down int64
	s.db.Model(&ServiceInstance{}).Count(&total)
	s.db.Model(&ServiceInstance{}).Where("status = ?", "up").Count(&up)
	s.db.Model(&ServiceInstance{}).Where("status = ?", "down").Count(&down)
	var services int64
	s.db.Model(&ServiceInstance{}).Distinct("service_name").Count(&services)
	var routes int64
	s.db.Model(&ServiceRoute{}).Count(&routes)

	c.JSON(http.StatusOK, gin.H{
		"status":              "operational",
		"unique_services":     services,
		"total_instances":     total,
		"healthy_instances":   up,
		"unhealthy_instances": down,
		"registered_routes":   routes,
	})
}

func (s *Service) listServices(c *gin.Context) {
	type svcSummary struct {
		ServiceName string `json:"service_name"`
		Instances   int64  `json:"instances"`
		Healthy     int64  `json:"healthy"`
	}

	var names []string
	s.db.Model(&ServiceInstance{}).Distinct("service_name").Pluck("service_name", &names)

	summaries := make([]svcSummary, 0, len(names))
	for _, n := range names {
		var cnt, healthy int64
		s.db.Model(&ServiceInstance{}).Where("service_name = ?", n).Count(&cnt)
		s.db.Model(&ServiceInstance{}).Where("service_name = ? AND status = ?", n, "up").Count(&healthy)
		summaries = append(summaries, svcSummary{ServiceName: n, Instances: cnt, Healthy: healthy})
	}
	c.JSON(http.StatusOK, gin.H{"services": summaries})
}

func (s *Service) getService(c *gin.Context) {
	name := c.Param("name")
	var instances []ServiceInstance
	s.db.Where("service_name = ?", name).Order("zone, host").Find(&instances)
	if len(instances) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "service not found"})
		return
	}
	var routes []ServiceRoute
	s.db.Where("service_name = ?", name).Find(&routes)
	c.JSON(http.StatusOK, gin.H{"service": name, "instances": instances, "routes": routes})
}

func (s *Service) register(c *gin.Context) {
	var req ServiceInstance
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.Status = "starting"
	now := time.Now()
	req.LastHeartbeat = &now
	req.RegisteredAt = now
	s.db.Create(&req)

	// Auto-promote to up
	s.db.Model(&req).Update("status", "up")
	req.Status = "up"

	s.logger.Info("Service registered", zap.String("service", req.ServiceName),
		zap.String("host", req.Host), zap.Int("port", req.Port))
	c.JSON(http.StatusCreated, gin.H{"instance": req})
}

func (s *Service) deregister(c *gin.Context) {
	id := c.Param("id")
	var inst ServiceInstance
	if err := s.db.First(&inst, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	s.db.Delete(&inst)
	s.logger.Info("Service deregistered", zap.String("service", inst.ServiceName), zap.String("host", inst.Host))
	c.JSON(http.StatusOK, gin.H{"message": "deregistered"})
}

func (s *Service) heartbeat(c *gin.Context) {
	id := c.Param("id")
	now := time.Now()
	result := s.db.Model(&ServiceInstance{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_heartbeat": now, "status": "up",
	})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "heartbeat received", "timestamp": now})
}

func (s *Service) drain(c *gin.Context) {
	id := c.Param("id")
	result := s.db.Model(&ServiceInstance{}).Where("id = ?", id).Update("status", "draining")
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "instance set to draining"})
}

func (s *Service) listRoutes(c *gin.Context) {
	var routes []ServiceRoute
	s.db.Order("service_name, path_prefix").Find(&routes)
	c.JSON(http.StatusOK, gin.H{"routes": routes})
}

func (s *Service) getTopology(c *gin.Context) {
	var instances []ServiceInstance
	s.db.Order("region, zone, service_name").Find(&instances)

	// Build topology tree: region → zone → services
	type zoneInfo struct {
		Zone      string            `json:"zone"`
		Instances []ServiceInstance `json:"instances"`
	}
	type regionInfo struct {
		Region string     `json:"region"`
		Zones  []zoneInfo `json:"zones"`
	}

	regionMap := map[string]map[string][]ServiceInstance{}
	for _, inst := range instances {
		r := inst.Region
		if r == "" {
			r = "default"
		}
		z := inst.Zone
		if z == "" {
			z = "default"
		}
		if regionMap[r] == nil {
			regionMap[r] = map[string][]ServiceInstance{}
		}
		regionMap[r][z] = append(regionMap[r][z], inst)
	}

	var topology []regionInfo
	for rName, zones := range regionMap {
		ri := regionInfo{Region: rName}
		for zName, insts := range zones {
			ri.Zones = append(ri.Zones, zoneInfo{Zone: zName, Instances: insts})
		}
		topology = append(topology, ri)
	}
	c.JSON(http.StatusOK, gin.H{"topology": topology})
}
