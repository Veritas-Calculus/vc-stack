// Package catalog implements the unified Service Catalog for VC Stack.
// Provides a browsable marketplace of infrastructure services, curated
// offerings, quick-deploy templates, and tenant self-service provisioning.
package catalog

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ---------- Models ----------

// Category groups related catalog items.
type Category struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name        string    `json:"name" gorm:"not null;uniqueIndex"`
	Icon        string    `json:"icon"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sort_order" gorm:"default:0"`
	ItemCount   int       `json:"item_count" gorm:"-"`
	CreatedAt   time.Time `json:"created_at"`
}

func (Category) TableName() string { return "catalog_categories" }

// CatalogItem is a service offering in the catalog.
type CatalogItem struct {
	ID          string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	CategoryID  string `json:"category_id" gorm:"not null;index"`
	Name        string `json:"name" gorm:"not null;uniqueIndex"`
	DisplayName string `json:"display_name"`
	Description string `json:"description" gorm:"type:text"`
	Icon        string `json:"icon"`
	// Pricing
	PriceUnit   string  `json:"price_unit"` // hour, month, request, GB
	PriceAmount float64 `json:"price_amount"`
	// Specs
	Specs string `json:"specs" gorm:"type:text"` // JSON: {"vcpu":4,"ram_gb":8,...}
	Tags  string `json:"tags" gorm:"type:text"`  // JSON array
	// Provisioning
	ProvisionType string `json:"provision_type"` // instant, approval_required, manual
	TemplateID    string `json:"template_id"`    // orchestration stack template
	// Status
	Status      string    `json:"status" gorm:"default:'published'"` // published, draft, deprecated, disabled
	Featured    bool      `json:"featured" gorm:"default:false"`
	Popular     bool      `json:"popular" gorm:"default:false"`
	Deployments int       `json:"deployments" gorm:"default:0"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CatalogItem) TableName() string { return "catalog_items" }

// CatalogRequest tracks user requests to provision catalog items.
type CatalogRequest struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ItemID      string    `json:"item_id" gorm:"not null;index"`
	ItemName    string    `json:"item_name"`
	TenantID    string    `json:"tenant_id" gorm:"index"`
	RequestedBy string    `json:"requested_by"`
	Status      string    `json:"status" gorm:"default:'pending'"` // pending, approved, provisioning, completed, rejected, failed
	Parameters  string    `json:"parameters" gorm:"type:text"`     // JSON parameters
	Notes       string    `json:"notes" gorm:"type:text"`
	ApprovedBy  string    `json:"approved_by"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (CatalogRequest) TableName() string { return "catalog_requests" }

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
	if err := cfg.DB.AutoMigrate(&Category{}, &CatalogItem{}, &CatalogRequest{}); err != nil {
		return nil, fmt.Errorf("catalog: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Service catalog initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	cats := []Category{
		{ID: uuid.New().String(), Name: "compute", Icon: "🖥️", Description: "Virtual machines, containers, and bare metal servers", SortOrder: 1},
		{ID: uuid.New().String(), Name: "storage", Icon: "💾", Description: "Block, object, and file storage services", SortOrder: 2},
		{ID: uuid.New().String(), Name: "networking", Icon: "🌐", Description: "Virtual networks, load balancers, DNS, and VPN", SortOrder: 3},
		{ID: uuid.New().String(), Name: "databases", Icon: "🗄️", Description: "Managed database instances", SortOrder: 4},
		{ID: uuid.New().String(), Name: "security", Icon: "🔒", Description: "Identity management, encryption, and compliance", SortOrder: 5},
		{ID: uuid.New().String(), Name: "containers", Icon: "📦", Description: "Kubernetes clusters and container orchestration", SortOrder: 6},
	}
	catMap := map[string]string{}
	for i := range cats {
		s.db.Where("name = ?", cats[i].Name).FirstOrCreate(&cats[i])
		catMap[cats[i].Name] = cats[i].ID
	}

	items := []CatalogItem{
		// Compute
		{ID: uuid.New().String(), CategoryID: catMap["compute"], Name: "vm-standard", DisplayName: "Standard VM",
			Description: "General-purpose virtual machine with balanced CPU/memory ratio. Ideal for web applications, dev/test environments, and small databases.",
			Icon:        "🖥️", PriceUnit: "hour", PriceAmount: 0.05, Specs: `{"vcpu":2,"ram_gb":4,"disk_gb":40}`,
			Tags: `["general","web","dev"]`, ProvisionType: "instant", Status: "published", Featured: true, Popular: true, Deployments: 142},
		{ID: uuid.New().String(), CategoryID: catMap["compute"], Name: "vm-performance", DisplayName: "Performance VM",
			Description: "High-performance virtual machine with dedicated CPU cores and NVMe storage for production workloads.",
			Icon:        "⚡", PriceUnit: "hour", PriceAmount: 0.20, Specs: `{"vcpu":8,"ram_gb":32,"disk_gb":200,"disk_type":"nvme"}`,
			Tags: `["production","high-perf"]`, ProvisionType: "instant", Status: "published", Featured: true, Deployments: 67},
		{ID: uuid.New().String(), CategoryID: catMap["compute"], Name: "vm-gpu", DisplayName: "GPU Accelerated VM",
			Description: "GPU-equipped virtual machine for AI/ML training, rendering, and HPC workloads. NVIDIA A100 attached.",
			Icon:        "🎮", PriceUnit: "hour", PriceAmount: 2.50, Specs: `{"vcpu":16,"ram_gb":64,"disk_gb":500,"gpu":"A100-40GB"}`,
			Tags: `["gpu","ai","ml","hpc"]`, ProvisionType: "approval_required", Status: "published", Deployments: 12},
		{ID: uuid.New().String(), CategoryID: catMap["compute"], Name: "bare-metal-standard", DisplayName: "Bare Metal Server",
			Description: "Dedicated physical server with full hardware access. No hypervisor overhead.",
			Icon:        "🏗️", PriceUnit: "month", PriceAmount: 450.00, Specs: `{"cpu_cores":32,"ram_gb":256,"disk_tb":4,"disk_type":"nvme"}`,
			Tags: `["bare-metal","dedicated"]`, ProvisionType: "approval_required", Status: "published", Deployments: 5},
		// Storage
		{ID: uuid.New().String(), CategoryID: catMap["storage"], Name: "block-ssd", DisplayName: "SSD Block Storage",
			Description: "High-performance SSD block storage volume. Hot-attachable to any VM instance.",
			Icon:        "💿", PriceUnit: "GB", PriceAmount: 0.10, Specs: `{"iops_max":10000,"throughput_mbps":250,"type":"ssd"}`,
			Tags: `["block","ssd"]`, ProvisionType: "instant", Status: "published", Popular: true, Deployments: 230},
		{ID: uuid.New().String(), CategoryID: catMap["storage"], Name: "object-s3", DisplayName: "S3 Object Storage",
			Description: "S3-compatible object storage with 99.999999999% durability. Ideal for backups, media, and archives.",
			Icon:        "🪣", PriceUnit: "GB", PriceAmount: 0.023, Specs: `{"durability":"11-nines","api":"s3-compatible"}`,
			Tags: `["object","s3","backup"]`, ProvisionType: "instant", Status: "published", Deployments: 89},
		// Networking
		{ID: uuid.New().String(), CategoryID: catMap["networking"], Name: "vpc-network", DisplayName: "Virtual Private Cloud",
			Description: "Isolated virtual network with customizable CIDR, subnets, and routing. OVN-powered SDN.",
			Icon:        "🌐", PriceUnit: "month", PriceAmount: 0, Specs: `{"max_subnets":16,"max_security_groups":50}`,
			Tags: `["network","vpc","sdn"]`, ProvisionType: "instant", Status: "published", Deployments: 95},
		{ID: uuid.New().String(), CategoryID: catMap["networking"], Name: "load-balancer", DisplayName: "Load Balancer",
			Description: "Layer 4/7 load balancer with health checks, SSL termination, and sticky sessions.",
			Icon:        "⚖️", PriceUnit: "hour", PriceAmount: 0.025, Specs: `{"layers":["L4","L7"],"ssl":true,"health_checks":true}`,
			Tags: `["lb","ha","load-balancer"]`, ProvisionType: "instant", Status: "published", Deployments: 38},
		// Databases
		{ID: uuid.New().String(), CategoryID: catMap["databases"], Name: "db-postgresql", DisplayName: "Managed PostgreSQL",
			Description: "Fully managed PostgreSQL with automated backups, replication, and point-in-time recovery.",
			Icon:        "🐘", PriceUnit: "hour", PriceAmount: 0.08, Specs: `{"engine":"PostgreSQL 16","ha":true,"backup":"daily","pitr":true}`,
			Tags: `["database","postgresql","managed"]`, ProvisionType: "instant", Status: "published", Featured: true, Deployments: 55},
		{ID: uuid.New().String(), CategoryID: catMap["databases"], Name: "db-redis", DisplayName: "Managed Redis",
			Description: "In-memory cache/store with cluster mode, persistence, and automatic failover.",
			Icon:        "🔴", PriceUnit: "hour", PriceAmount: 0.04, Specs: `{"engine":"Redis 7","cluster":true,"persistence":true}`,
			Tags: `["cache","redis","nosql"]`, ProvisionType: "instant", Status: "published", Deployments: 42},
		// Security
		{ID: uuid.New().String(), CategoryID: catMap["security"], Name: "kms-key", DisplayName: "Encryption Key (KMS)",
			Description: "Create and manage AES-256 encryption keys for data-at-rest protection.",
			Icon:        "🔑", PriceUnit: "month", PriceAmount: 1.0, Specs: `{"algorithm":"AES-256-GCM","rotation":"automatic"}`,
			Tags: `["encryption","kms","security"]`, ProvisionType: "instant", Status: "published", Deployments: 18},
		// Containers
		{ID: uuid.New().String(), CategoryID: catMap["containers"], Name: "k8s-cluster", DisplayName: "Kubernetes Cluster",
			Description: "Managed Kubernetes cluster with Calico CNI, OVN integration, and auto-scaling node pools.",
			Icon:        "☸️", PriceUnit: "hour", PriceAmount: 0.10, Specs: `{"k8s_version":"v1.29","cni":"calico","autoscale":true}`,
			Tags: `["kubernetes","k8s","containers"]`, ProvisionType: "approval_required", Status: "published", Featured: true, Deployments: 22},
	}
	for i := range items {
		s.db.Where("name = ?", items[i].Name).FirstOrCreate(&items[i])
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	api := router.Group("/api/v1/catalog")
	{
		api.GET("/status", s.getStatus)
		api.GET("/categories", s.listCategories)
		api.GET("/items", s.listItems)
		api.GET("/items/:id", s.getItem)
		api.GET("/featured", s.listFeatured)
		api.GET("/popular", s.listPopular)
		api.POST("/items", s.createItem)
		api.PUT("/items/:id", s.updateItem)
		api.DELETE("/items/:id", s.deleteItem)
		// Requests
		api.GET("/requests", s.listRequests)
		api.POST("/requests", s.createRequest)
		api.PUT("/requests/:id/approve", s.approveRequest)
		api.PUT("/requests/:id/reject", s.rejectRequest)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var catCount, itemCount, reqCount int64
	s.db.Model(&Category{}).Count(&catCount)
	s.db.Model(&CatalogItem{}).Where("status = ?", "published").Count(&itemCount)
	s.db.Model(&CatalogRequest{}).Count(&reqCount)
	var pendingReqs int64
	s.db.Model(&CatalogRequest{}).Where("status = ?", "pending").Count(&pendingReqs)
	var totalDeploy int64
	s.db.Model(&CatalogItem{}).Select("COALESCE(SUM(deployments), 0)").Scan(&totalDeploy)
	c.JSON(http.StatusOK, gin.H{
		"status":            "operational",
		"categories":        catCount,
		"published_items":   itemCount,
		"total_requests":    reqCount,
		"pending_requests":  pendingReqs,
		"total_deployments": totalDeploy,
	})
}

func (s *Service) listCategories(c *gin.Context) {
	var cats []Category
	s.db.Order("sort_order").Find(&cats)
	// Count items per category
	for i := range cats {
		var count int64
		s.db.Model(&CatalogItem{}).Where("category_id = ? AND status = ?", cats[i].ID, "published").Count(&count)
		cats[i].ItemCount = int(count)
	}
	c.JSON(http.StatusOK, gin.H{"categories": cats})
}

func (s *Service) listItems(c *gin.Context) {
	var items []CatalogItem
	q := s.db.Where("status = ?", "published")
	if cat := c.Query("category"); cat != "" {
		var catRec Category
		if err := s.db.Where("name = ?", cat).First(&catRec).Error; err == nil {
			q = q.Where("category_id = ?", catRec.ID)
		}
	}
	if search := c.Query("search"); search != "" {
		q = q.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%", "%"+search+"%")
	}
	q.Order("deployments DESC, name").Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Service) getItem(c *gin.Context) {
	var item CatalogItem
	if err := s.db.First(&item, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"item": item})
}

func (s *Service) listFeatured(c *gin.Context) {
	var items []CatalogItem
	s.db.Where("featured = ? AND status = ?", true, "published").Order("deployments DESC").Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Service) listPopular(c *gin.Context) {
	var items []CatalogItem
	s.db.Where("popular = ? AND status = ?", true, "published").Order("deployments DESC").Find(&items)
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (s *Service) createItem(c *gin.Context) {
	var req CatalogItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if req.Status == "" {
		req.Status = "published"
	}
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "item name exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"item": req})
}

func (s *Service) updateItem(c *gin.Context) {
	id := c.Param("id")
	var existing CatalogItem
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"item": existing})
}

func (s *Service) deleteItem(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&CatalogItem{})
	c.JSON(http.StatusOK, gin.H{"message": "item deleted"})
}

func (s *Service) listRequests(c *gin.Context) {
	var reqs []CatalogRequest
	s.db.Order("created_at DESC").Find(&reqs)
	c.JSON(http.StatusOK, gin.H{"requests": reqs})
}

func (s *Service) createRequest(c *gin.Context) {
	var req struct {
		ItemID     string `json:"item_id" binding:"required"`
		Parameters string `json:"parameters"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var item CatalogItem
	if err := s.db.First(&item, "id = ?", req.ItemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "catalog item not found"})
		return
	}

	status := "pending"
	if item.ProvisionType == "instant" {
		status = "completed"
		s.db.Model(&item).Update("deployments", item.Deployments+1)
	}

	catReq := CatalogRequest{
		ID: uuid.New().String(), ItemID: req.ItemID, ItemName: item.DisplayName,
		RequestedBy: "admin", Status: status,
		Parameters: req.Parameters, Notes: req.Notes,
	}
	s.db.Create(&catReq)
	c.JSON(http.StatusCreated, gin.H{"request": catReq})
}

func (s *Service) approveRequest(c *gin.Context) {
	var req CatalogRequest
	if err := s.db.First(&req, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}
	if req.Status != "pending" {
		c.JSON(http.StatusConflict, gin.H{"error": "request is not pending"})
		return
	}
	req.Status = "completed"
	req.ApprovedBy = "admin"
	s.db.Save(&req)

	// Increment deployment count
	s.db.Model(&CatalogItem{}).Where("id = ?", req.ItemID).UpdateColumn("deployments", gorm.Expr("deployments + 1"))

	c.JSON(http.StatusOK, gin.H{"request": req})
}

func (s *Service) rejectRequest(c *gin.Context) {
	var req CatalogRequest
	if err := s.db.First(&req, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "request not found"})
		return
	}
	req.Status = "rejected"
	s.db.Save(&req)
	c.JSON(http.StatusOK, gin.H{"request": req})
}
