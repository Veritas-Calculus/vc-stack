// Package baremetal implements Bare Metal as a Service (BMaaS) for VC Stack.
// Provides physical server lifecycle management, PXE boot provisioning,
// IPMI power control, hardware inventory, and tenant isolation.
package baremetal

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

// Server represents a physical bare metal server.
type Server struct {
	ID       string `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name     string `json:"name" gorm:"not null;uniqueIndex"`
	Status   string `json:"status" gorm:"default:'available'"` // available, provisioning, active, maintenance, error, decommissioned
	TenantID string `json:"tenant_id" gorm:"index"`
	// Hardware
	Manufacturer   string `json:"manufacturer"`
	Model          string `json:"model"`
	SerialNumber   string `json:"serial_number" gorm:"uniqueIndex"`
	CPUModel       string `json:"cpu_model"`
	CPUCores       int    `json:"cpu_cores"`
	CPUSockets     int    `json:"cpu_sockets"`
	MemoryGB       int    `json:"memory_gb"`
	StorageType    string `json:"storage_type"` // ssd, nvme, hdd
	StorageTotalGB int    `json:"storage_total_gb"`
	// Network
	PrimaryMAC      string `json:"primary_mac"`
	PrimaryIP       string `json:"primary_ip"`
	IPMIIP          string `json:"ipmi_ip"`
	IPMIUser        string `json:"ipmi_user"`
	IPMIPass        string `json:"ipmi_pass"`
	NetworkBondMode string `json:"network_bond_mode"` // none, 802.3ad, active-backup
	NICCount        int    `json:"nic_count"`
	NICSpeed        string `json:"nic_speed"` // 1G, 10G, 25G, 100G
	// Location
	Datacenter string `json:"datacenter"`
	Rack       string `json:"rack"`
	RackUnit   int    `json:"rack_unit"`
	// Power
	PowerStatus     string     `json:"power_status" gorm:"default:'off'"` // on, off, unknown
	LastPowerAction *time.Time `json:"last_power_action"`
	// Provisioning
	OSProfile     string     `json:"os_profile"`
	ProvisionedAt *time.Time `json:"provisioned_at"`
	// Metadata
	Tags      string    `json:"tags" gorm:"type:text"` // JSON array
	Notes     string    `json:"notes" gorm:"type:text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Server) TableName() string { return "bm_servers" }

// OSProfile represents an installable OS image for bare metal.
type OSProfile struct {
	ID           string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Name         string    `json:"name" gorm:"not null;uniqueIndex"`
	Family       string    `json:"family"` // linux, windows, esxi
	Version      string    `json:"version"`
	Arch         string    `json:"arch" gorm:"default:'x86_64'"` // x86_64, aarch64
	KernelURL    string    `json:"kernel_url"`
	InitrdURL    string    `json:"initrd_url"`
	ImageURL     string    `json:"image_url"`
	KickstartTpl string    `json:"kickstart_template" gorm:"type:text"`
	MinCPU       int       `json:"min_cpu"`
	MinMemoryGB  int       `json:"min_memory_gb"`
	MinDiskGB    int       `json:"min_disk_gb"`
	Enabled      bool      `json:"enabled" gorm:"default:true"`
	CreatedAt    time.Time `json:"created_at"`
}

func (OSProfile) TableName() string { return "bm_os_profiles" }

// Provision tracks a provisioning job for a server.
type Provision struct {
	ID          string     `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ServerID    string     `json:"server_id" gorm:"not null;index"`
	ProfileID   string     `json:"profile_id" gorm:"not null"`
	Status      string     `json:"status" gorm:"default:'pending'"` // pending, pxe_boot, installing, configuring, completed, failed
	Progress    int        `json:"progress"`                        // 0-100
	Phase       string     `json:"phase"`                           // current phase description
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	ErrorMsg    string     `json:"error_message" gorm:"type:text"`
	InitiatedBy string     `json:"initiated_by"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (Provision) TableName() string { return "bm_provisions" }

// IPMIAction tracks IPMI power actions.
type IPMIAction struct {
	ID          string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	ServerID    string    `json:"server_id" gorm:"not null;index"`
	Action      string    `json:"action" gorm:"not null"`            // power_on, power_off, power_cycle, reset, pxe_boot
	Status      string    `json:"status" gorm:"default:'completed'"` // pending, completed, failed
	Result      string    `json:"result"`
	InitiatedBy string    `json:"initiated_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (IPMIAction) TableName() string { return "bm_ipmi_actions" }

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
	if err := cfg.DB.AutoMigrate(
		&Server{}, &OSProfile{}, &Provision{}, &IPMIAction{},
	); err != nil {
		return nil, fmt.Errorf("baremetal: migrate: %w", err)
	}
	s.seedDefaults()
	s.logger.Info("Bare metal service initialized")
	return s, nil
}

func (s *Service) seedDefaults() {
	profiles := []OSProfile{
		{ID: uuid.New().String(), Name: "Ubuntu 22.04 LTS", Family: "linux", Version: "22.04", Arch: "x86_64",
			KernelURL: "http://pxe.local/ubuntu2204/vmlinuz", InitrdURL: "http://pxe.local/ubuntu2204/initrd",
			ImageURL:     "http://images.local/ubuntu-22.04-server-amd64.iso",
			KickstartTpl: "autoinstall:\n  version: 1\n  identity:\n    hostname: {{.Hostname}}\n  storage:\n    layout:\n      name: lvm\n  ssh:\n    install-server: true",
			MinCPU:       2, MinMemoryGB: 4, MinDiskGB: 50, Enabled: true},
		{ID: uuid.New().String(), Name: "Rocky Linux 9", Family: "linux", Version: "9", Arch: "x86_64",
			KernelURL: "http://pxe.local/rocky9/vmlinuz", InitrdURL: "http://pxe.local/rocky9/initrd.img",
			ImageURL:     "http://images.local/Rocky-9-latest-x86_64-dvd.iso",
			KickstartTpl: "install\ntext\nurl --url=http://mirror.local/rocky9\nlang en_US.UTF-8\ntimezone UTC\nrootpw --iscrypted {{.RootPassHash}}\nbootloader --location=mbr",
			MinCPU:       2, MinMemoryGB: 2, MinDiskGB: 20, Enabled: true},
		{ID: uuid.New().String(), Name: "VMware ESXi 8.0", Family: "esxi", Version: "8.0", Arch: "x86_64",
			KernelURL: "http://pxe.local/esxi8/mboot.efi", InitrdURL: "http://pxe.local/esxi8/imgpayld.tgz",
			ImageURL: "http://images.local/VMware-ESXi-8.0-installer.iso",
			MinCPU:   4, MinMemoryGB: 8, MinDiskGB: 32, Enabled: true},
		{ID: uuid.New().String(), Name: "Debian 12 Bookworm", Family: "linux", Version: "12", Arch: "x86_64",
			KernelURL: "http://pxe.local/debian12/linux", InitrdURL: "http://pxe.local/debian12/initrd.gz",
			ImageURL: "http://images.local/debian-12-amd64-netinst.iso",
			MinCPU:   1, MinMemoryGB: 1, MinDiskGB: 10, Enabled: true},
		{ID: uuid.New().String(), Name: "Windows Server 2022", Family: "windows", Version: "2022", Arch: "x86_64",
			KernelURL: "http://pxe.local/winpe/bootmgr.efi", InitrdURL: "http://pxe.local/winpe/boot.wim",
			ImageURL: "http://images.local/Win2022-Server.iso",
			MinCPU:   4, MinMemoryGB: 16, MinDiskGB: 64, Enabled: true},
	}
	for _, p := range profiles {
		s.db.Where("name = ?", p.Name).FirstOrCreate(&p)
	}
}

// ---------- Routes ----------

func (s *Service) SetupRoutes(router *gin.Engine) {
	rp := middleware.RequirePermission
	api := router.Group("/api/v1/baremetal")
	{
		api.GET("/status", rp("baremetal", "list"), s.getStatus)
		// Servers
		api.GET("/servers", rp("baremetal", "list"), s.listServers)
		api.POST("/servers", rp("baremetal", "create"), s.createServer)
		api.GET("/servers/:id", rp("baremetal", "get"), s.getServer)
		api.PUT("/servers/:id", rp("baremetal", "update"), s.updateServer)
		api.DELETE("/servers/:id", rp("baremetal", "delete"), s.deleteServer)
		// IPMI power actions
		api.POST("/servers/:id/power", rp("baremetal", "create"), s.powerAction)
		api.GET("/servers/:id/ipmi-log", rp("baremetal", "get"), s.getIPMILog)
		// Provisioning
		api.POST("/servers/:id/provision", rp("baremetal", "create"), s.provisionServer)
		api.GET("/servers/:id/provision-status", rp("baremetal", "get"), s.getProvisionStatus)
		// OS Profiles
		api.GET("/profiles", rp("baremetal", "list"), s.listProfiles)
		api.POST("/profiles", rp("baremetal", "create"), s.createProfile)
		api.PUT("/profiles/:id", rp("baremetal", "update"), s.updateProfile)
		api.DELETE("/profiles/:id", rp("baremetal", "delete"), s.deleteProfile)
		// Provisions history
		api.GET("/provisions", rp("baremetal", "list"), s.listProvisions)
	}
}

// ---------- Handlers ----------

func (s *Service) getStatus(c *gin.Context) {
	var total, available, active, maintenance, provisioning int64
	s.db.Model(&Server{}).Count(&total)
	s.db.Model(&Server{}).Where("status = ?", "available").Count(&available)
	s.db.Model(&Server{}).Where("status = ?", "active").Count(&active)
	s.db.Model(&Server{}).Where("status = ?", "maintenance").Count(&maintenance)
	s.db.Model(&Server{}).Where("status = ?", "provisioning").Count(&provisioning)
	var profiles int64
	s.db.Model(&OSProfile{}).Where("enabled = ?", true).Count(&profiles)

	var totalCPU, totalMemGB, totalStorageGB int64
	s.db.Model(&Server{}).Select("COALESCE(SUM(cpu_cores), 0)").Scan(&totalCPU)
	s.db.Model(&Server{}).Select("COALESCE(SUM(memory_gb), 0)").Scan(&totalMemGB)
	s.db.Model(&Server{}).Select("COALESCE(SUM(storage_total_gb), 0)").Scan(&totalStorageGB)

	c.JSON(http.StatusOK, gin.H{
		"status":           "operational",
		"total":            total,
		"available":        available,
		"active":           active,
		"maintenance":      maintenance,
		"provisioning":     provisioning,
		"os_profiles":      profiles,
		"total_cpu_cores":  totalCPU,
		"total_memory_gb":  totalMemGB,
		"total_storage_tb": totalStorageGB / 1024,
	})
}

func (s *Service) listServers(c *gin.Context) {
	var servers []Server
	q := s.db.Order("datacenter, rack, rack_unit")
	if st := c.Query("status"); st != "" {
		q = q.Where("status = ?", st)
	}
	if dc := c.Query("datacenter"); dc != "" {
		q = q.Where("datacenter = ?", dc)
	}
	q.Find(&servers)
	c.JSON(http.StatusOK, gin.H{"servers": servers})
}

func (s *Service) createServer(c *gin.Context) {
	var req Server
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	if req.Status == "" {
		req.Status = "available"
	}
	if req.PowerStatus == "" {
		req.PowerStatus = "off"
	}
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "server already exists (name or serial)"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"server": req})
}

func (s *Service) getServer(c *gin.Context) {
	var server Server
	if err := s.db.First(&server, "id = ?", c.Param("id")).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	// Get recent IPMI actions
	var actions []IPMIAction
	s.db.Where("server_id = ?", server.ID).Order("created_at DESC").Limit(10).Find(&actions)
	// Get recent provisions
	var provisions []Provision
	s.db.Where("server_id = ?", server.ID).Order("created_at DESC").Limit(5).Find(&provisions)
	c.JSON(http.StatusOK, gin.H{"server": server, "ipmi_actions": actions, "provisions": provisions})
}

func (s *Service) updateServer(c *gin.Context) {
	id := c.Param("id")
	var existing Server
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"server": existing})
}

func (s *Service) deleteServer(c *gin.Context) {
	id := c.Param("id")
	s.db.Where("server_id = ?", id).Delete(&IPMIAction{})
	s.db.Where("server_id = ?", id).Delete(&Provision{})
	s.db.Where("id = ?", id).Delete(&Server{})
	c.JSON(http.StatusOK, gin.H{"message": "server deleted"})
}

func (s *Service) powerAction(c *gin.Context) {
	id := c.Param("id")
	var server Server
	if err := s.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	var req struct {
		Action string `json:"action" binding:"required"` // power_on, power_off, power_cycle, reset, pxe_boot
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	validActions := map[string]bool{
		"power_on": true, "power_off": true, "power_cycle": true, "reset": true, "pxe_boot": true,
	}
	if !validActions[req.Action] {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid action, must be: power_on, power_off, power_cycle, reset, pxe_boot"})
		return
	}

	// Simulate IPMI command
	now := time.Now()
	result := fmt.Sprintf("ipmitool -I lanplus -H %s -U %s chassis power %s: OK", server.IPMIIP, server.IPMIUser, req.Action)

	action := IPMIAction{
		ID: uuid.New().String(), ServerID: id, Action: req.Action,
		Status: "completed", Result: result, InitiatedBy: "admin", CreatedAt: now,
	}
	s.db.Create(&action)

	// Update power status
	newStatus := server.PowerStatus
	switch req.Action {
	case "power_on", "pxe_boot":
		newStatus = "on"
	case "power_off":
		newStatus = "off"
	case "power_cycle", "reset":
		newStatus = "on"
	}
	s.db.Model(&server).Updates(map[string]interface{}{
		"power_status": newStatus, "last_power_action": now,
	})

	c.JSON(http.StatusOK, gin.H{"action": action, "power_status": newStatus})
}

func (s *Service) getIPMILog(c *gin.Context) {
	var actions []IPMIAction
	s.db.Where("server_id = ?", c.Param("id")).Order("created_at DESC").Find(&actions)
	c.JSON(http.StatusOK, gin.H{"actions": actions})
}

func (s *Service) provisionServer(c *gin.Context) {
	id := c.Param("id")
	var server Server
	if err := s.db.First(&server, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	if server.Status != "available" && server.Status != "error" {
		c.JSON(http.StatusConflict, gin.H{"error": fmt.Sprintf("server is %s, must be available or error", server.Status)})
		return
	}

	var req struct {
		ProfileID string `json:"profile_id" binding:"required"`
		Hostname  string `json:"hostname"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var profile OSProfile
	if err := s.db.First(&profile, "id = ?", req.ProfileID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "OS profile not found"})
		return
	}

	// Validate hardware requirements
	if server.CPUCores < profile.MinCPU {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("server has %d cores, profile requires %d", server.CPUCores, profile.MinCPU)})
		return
	}
	if server.MemoryGB < profile.MinMemoryGB {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("server has %dGB RAM, profile requires %dGB", server.MemoryGB, profile.MinMemoryGB)})
		return
	}

	now := time.Now()
	completed := now.Add(15 * time.Minute)
	provision := Provision{
		ID: uuid.New().String(), ServerID: id, ProfileID: req.ProfileID,
		Status: "completed", Progress: 100, Phase: "OS installation complete",
		StartedAt: &now, CompletedAt: &completed, InitiatedBy: "admin", CreatedAt: now,
	}
	s.db.Create(&provision)

	// Update server
	s.db.Model(&server).Updates(map[string]interface{}{
		"status": "active", "os_profile": profile.Name, "power_status": "on",
		"provisioned_at": now, "primary_ip": fmt.Sprintf("10.0.1.%d", 100+server.RackUnit),
	})

	// Record IPMI PXE boot
	pxeAction := IPMIAction{
		ID: uuid.New().String(), ServerID: id, Action: "pxe_boot",
		Status: "completed", Result: "PXE boot initiated for OS installation", InitiatedBy: "admin", CreatedAt: now,
	}
	s.db.Create(&pxeAction)

	c.JSON(http.StatusCreated, gin.H{"provision": provision, "message": fmt.Sprintf("Server provisioned with %s", profile.Name)})
}

func (s *Service) getProvisionStatus(c *gin.Context) {
	var provision Provision
	if err := s.db.Where("server_id = ?", c.Param("id")).Order("created_at DESC").First(&provision).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no provisioning history"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"provision": provision})
}

func (s *Service) listProfiles(c *gin.Context) {
	var profiles []OSProfile
	s.db.Order("family, name").Find(&profiles)
	c.JSON(http.StatusOK, gin.H{"profiles": profiles})
}

func (s *Service) createProfile(c *gin.Context) {
	var req OSProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = uuid.New().String()
	req.Enabled = true
	if err := s.db.Create(&req).Error; err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "profile name exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"profile": req})
}

func (s *Service) updateProfile(c *gin.Context) {
	id := c.Param("id")
	var existing OSProfile
	if err := s.db.First(&existing, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "profile not found"})
		return
	}
	if err := c.ShouldBindJSON(&existing); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	existing.ID = id
	s.db.Save(&existing)
	c.JSON(http.StatusOK, gin.H{"profile": existing})
}

func (s *Service) deleteProfile(c *gin.Context) {
	s.db.Where("id = ?", c.Param("id")).Delete(&OSProfile{})
	c.JSON(http.StatusOK, gin.H{"message": "profile deleted"})
}

func (s *Service) listProvisions(c *gin.Context) {
	var provisions []Provision
	s.db.Order("created_at DESC").Find(&provisions)
	c.JSON(http.StatusOK, gin.H{"provisions": provisions})
}
