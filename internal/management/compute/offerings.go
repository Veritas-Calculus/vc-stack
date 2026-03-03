package compute

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/Veritas-Calculus/vc-stack/pkg/models"
)

// DiskOffering is an alias for the canonical model.
type DiskOffering = models.DiskOffering

// NetworkOffering is an alias for the canonical model.
type NetworkOffering = models.NetworkOffering

// migrateOfferings runs auto-migration for offering tables.
func (s *Service) migrateOfferings() error {
	return s.db.AutoMigrate(&DiskOffering{}, &NetworkOffering{})
}

// seedDefaultDiskOfferings seeds starter disk offerings.
func (s *Service) seedDefaultDiskOfferings() {
	var count int64
	s.db.Model(&DiskOffering{}).Count(&count)
	if count > 0 {
		return
	}
	defaults := []DiskOffering{
		{Name: "Standard", DisplayText: "Standard HDD storage", DiskSizeGB: 50, StorageType: "shared", MinIOPS: 100, MaxIOPS: 500},
		{Name: "Performance", DisplayText: "SSD-backed storage", DiskSizeGB: 100, StorageType: "ssd", MinIOPS: 3000, MaxIOPS: 10000, Throughput: 200},
		{Name: "Enterprise", DisplayText: "NVMe high-performance", DiskSizeGB: 200, StorageType: "nvme", MinIOPS: 10000, MaxIOPS: 50000, Throughput: 500},
		{Name: "Custom", DisplayText: "User-defined size", DiskSizeGB: 0, IsCustom: true, StorageType: "shared"},
	}
	for _, d := range defaults {
		if err := s.db.Create(&d).Error; err != nil {
			s.logger.Warn("failed to seed disk offering", zap.String("name", d.Name), zap.Error(err))
		}
	}
}

// seedDefaultNetworkOfferings seeds starter network offerings.
func (s *Service) seedDefaultNetworkOfferings() {
	var count int64
	s.db.Model(&NetworkOffering{}).Count(&count)
	if count > 0 {
		return
	}
	defaults := []NetworkOffering{
		{Name: "DefaultIsolated", DisplayText: "Default Isolated Network", GuestIPType: "isolated", EnableDHCP: true, EnableFirewall: true, EnableSourceNAT: true, IsDefault: true},
		{Name: "DefaultShared", DisplayText: "Default Shared Network", GuestIPType: "shared", EnableDHCP: true, EnableFirewall: true},
		{Name: "L2Network", DisplayText: "Layer 2 passthrough network", GuestIPType: "l2", EnableDHCP: false, EnableFirewall: false},
	}
	for _, d := range defaults {
		if err := s.db.Create(&d).Error; err != nil {
			s.logger.Warn("failed to seed network offering", zap.String("name", d.Name), zap.Error(err))
		}
	}
}

// --- Disk Offering handlers ---

func (s *Service) listDiskOfferings(c *gin.Context) {
	var offerings []DiskOffering
	if err := s.db.Order("id").Find(&offerings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list disk offerings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"disk_offerings": offerings})
}

type CreateDiskOfferingRequest struct {
	Name        string `json:"name" binding:"required"`
	DisplayText string `json:"display_text"`
	DiskSizeGB  int    `json:"disk_size_gb"`
	IsCustom    bool   `json:"is_custom"`
	StorageType string `json:"storage_type"`
	MinIOPS     int    `json:"min_iops"`
	MaxIOPS     int    `json:"max_iops"`
	BurstIOPS   int    `json:"burst_iops"`
	Throughput  int    `json:"throughput"`
}

func (s *Service) createDiskOffering(c *gin.Context) {
	var req CreateDiskOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	offering := DiskOffering{
		Name:        req.Name,
		DisplayText: req.DisplayText,
		DiskSizeGB:  req.DiskSizeGB,
		IsCustom:    req.IsCustom,
		StorageType: req.StorageType,
		MinIOPS:     req.MinIOPS,
		MaxIOPS:     req.MaxIOPS,
		BurstIOPS:   req.BurstIOPS,
		Throughput:  req.Throughput,
	}
	if offering.StorageType == "" {
		offering.StorageType = "shared"
	}

	if err := s.db.Create(&offering).Error; err != nil {
		s.logger.Error("failed to create disk offering", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create disk offering"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"disk_offering": offering})
}

func (s *Service) deleteDiskOffering(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&DiskOffering{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete disk offering"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Network Offering handlers ---

func (s *Service) listNetworkOfferings(c *gin.Context) {
	var offerings []NetworkOffering
	if err := s.db.Order("id").Find(&offerings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list network offerings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"network_offerings": offerings})
}

type CreateNetworkOfferingRequest struct {
	Name            string `json:"name" binding:"required"`
	DisplayText     string `json:"display_text"`
	GuestIPType     string `json:"guest_ip_type"`
	TrafficType     string `json:"traffic_type"`
	EnableDHCP      bool   `json:"enable_dhcp"`
	EnableFirewall  bool   `json:"enable_firewall"`
	EnableLB        bool   `json:"enable_lb"`
	EnableVPN       bool   `json:"enable_vpn"`
	EnableSourceNAT bool   `json:"enable_source_nat"`
	MaxConnections  int    `json:"max_connections"`
}

func (s *Service) createNetworkOffering(c *gin.Context) {
	var req CreateNetworkOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	offering := NetworkOffering{
		Name:            req.Name,
		DisplayText:     req.DisplayText,
		GuestIPType:     req.GuestIPType,
		TrafficType:     req.TrafficType,
		EnableDHCP:      req.EnableDHCP,
		EnableFirewall:  req.EnableFirewall,
		EnableLB:        req.EnableLB,
		EnableVPN:       req.EnableVPN,
		EnableSourceNAT: req.EnableSourceNAT,
		MaxConnections:  req.MaxConnections,
	}
	if offering.GuestIPType == "" {
		offering.GuestIPType = "isolated"
	}
	if offering.TrafficType == "" {
		offering.TrafficType = "guest"
	}

	if err := s.db.Create(&offering).Error; err != nil {
		s.logger.Error("failed to create network offering", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create network offering"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"network_offering": offering})
}

func (s *Service) deleteNetworkOffering(c *gin.Context) {
	id := c.Param("id")
	if err := s.db.Delete(&NetworkOffering{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete network offering"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
