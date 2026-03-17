// Package compute - Live migration support.
// Provides orchestration for migrating running instances between compute nodes.
package compute

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Migration represents a live migration job.
type Migration struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UUID             string     `gorm:"type:varchar(36);uniqueIndex" json:"uuid"`
	InstanceID       uint       `gorm:"not null;index" json:"instance_id"`
	InstanceUUID     string     `gorm:"type:varchar(36);index" json:"instance_uuid"`
	InstanceName     string     `json:"instance_name"`
	SourceHostID     string     `gorm:"type:varchar(36);index" json:"source_host_id"`
	SourceHostName   string     `json:"source_host_name"`
	SourceNodeAddr   string     `json:"source_node_addr"`
	DestHostID       string     `gorm:"type:varchar(36);index" json:"dest_host_id"`
	DestHostName     string     `json:"dest_host_name"`
	DestNodeAddr     string     `json:"dest_node_addr"`
	MigrationType    string     `gorm:"default:'live'" json:"migration_type"` // live, cold, evacuate
	Status           string     `gorm:"default:'queued'" json:"status"`       // queued, preparing, migrating, post-copy, completed, failed, cancelled
	Progress         int        `gorm:"default:0" json:"progress"`            // 0-100
	ErrorMessage     string     `json:"error_message,omitempty"`
	MemoryTotalBytes int64      `json:"memory_total_bytes,omitempty"`
	MemoryCopied     int64      `json:"memory_copied_bytes,omitempty"`
	DiskTotalBytes   int64      `json:"disk_total_bytes,omitempty"`
	DiskCopied       int64      `json:"disk_copied_bytes,omitempty"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName overrides the default table name.
func (Migration) TableName() string { return "compute_migrations" }

// MigrateRequest represents a live migration API request.
type MigrateRequest struct {
	DestHostID string `json:"dest_host_id"` // optional: specific target host
	Force      bool   `json:"force"`        // bypass safety checks
	Type       string `json:"type"`         // live (default), cold
}

// setupMigrationRoutes registers migration-related routes.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) setupMigrationRoutes(api *gin.RouterGroup) {
	api.POST("/instances/:id/migrate", s.migrateInstance)
	api.GET("/migrations", s.listMigrations)
	api.GET("/migrations/:id", s.getMigration)
	api.POST("/migrations/:id/cancel", s.cancelMigration)
}

// migrateMigrationTable auto-migrates the migration table.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) migrateMigrationTable() error {
	return s.db.AutoMigrate(&Migration{})
}

// migrateInstance handles POST /api/v1/instances/:id/migrate.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) migrateInstance(c *gin.Context) {
	id := c.Param("id")

	// Find the instance.
	var instance Instance
	if err := s.db.First(&instance, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "instance not found"})
		return
	}

	// Validate instance state.
	if instance.Status != "active" {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("instance must be in 'active' status to migrate (current: %s)", instance.Status),
		})
		return
	}

	if instance.HostID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "instance has no assigned host"})
		return
	}

	// Check for existing active migration.
	var activeMigration Migration
	if err := s.db.Where("instance_id = ? AND status IN (?, ?, ?)",
		instance.ID, "queued", "preparing", "migrating").First(&activeMigration).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{
			"error":        "instance already has an active migration",
			"migration_id": activeMigration.UUID,
		})
		return
	}

	// Parse request.
	var req MigrateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// Allow empty body (auto-select destination).
		req = MigrateRequest{}
	}

	migrationType := "live"
	if req.Type == "cold" {
		migrationType = "cold"
	}

	// Find source host info.
	var sourceHost struct {
		Name    string
		Address string
	}
	s.db.Table("hosts").Select("name, CONCAT(ip_address, ':', management_port) as address").
		Where("uuid = ?", instance.HostID).Scan(&sourceHost)

	// Select destination host.
	destHostID := req.DestHostID
	destHostName := ""
	destNodeAddr := ""

	if destHostID != "" {
		// Verify the specified destination host exists and is available.
		var destHost struct {
			UUID   string
			Name   string
			Status string
			Addr   string
		}
		result := s.db.Table("hosts").
			Select("uuid, name, status, CONCAT(ip_address, ':', management_port) as addr").
			Where("uuid = ? OR id = ?", destHostID, destHostID).Scan(&destHost)
		if result.Error != nil || destHost.UUID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "destination host not found"})
			return
		}
		if destHost.Status != "up" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("destination host is not available (status: %s)", destHost.Status),
			})
			return
		}
		if destHost.UUID == instance.HostID {
			c.JSON(http.StatusBadRequest, gin.H{"error": "destination host is the same as source host"})
			return
		}
		destHostID = destHost.UUID
		destHostName = destHost.Name
		destNodeAddr = destHost.Addr
	} else {
		// Auto-select: use scheduler to find the best available host
		// (excluding the current host).
		var bestHost struct {
			UUID string
			Name string
			Addr string
		}
		result := s.db.Table("hosts").
			Select("uuid, name, CONCAT(ip_address, ':', management_port) as addr").
			Where("uuid != ? AND status = ? AND resource_state = ?",
				instance.HostID, "up", "enabled").
			Order("cpu_allocated ASC, ram_allocated_mb ASC").
			Limit(1).Scan(&bestHost)
		if result.Error != nil || bestHost.UUID == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "no available destination host found",
			})
			return
		}
		destHostID = bestHost.UUID
		destHostName = bestHost.Name
		destNodeAddr = bestHost.Addr
	}

	// Create migration record.
	migration := &Migration{
		UUID:           uuid.New().String(),
		InstanceID:     instance.ID,
		InstanceUUID:   instance.UUID,
		InstanceName:   instance.Name,
		SourceHostID:   instance.HostID,
		SourceHostName: sourceHost.Name,
		SourceNodeAddr: sourceHost.Address,
		DestHostID:     destHostID,
		DestHostName:   destHostName,
		DestNodeAddr:   destNodeAddr,
		MigrationType:  migrationType,
		Status:         "queued",
	}

	if err := s.db.Create(migration).Error; err != nil {
		s.logger.Error("failed to create migration record", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create migration"})
		return
	}

	// Update instance status to migrating.
	_ = s.db.Model(&instance).Updates(map[string]interface{}{
		"status":      "migrating",
		"power_state": "running",
	}).Error

	// Log event.
	s.emitEvent("instance", instance.UUID, "migrate", "started", "", map[string]interface{}{
		"migration_id":   migration.UUID,
		"source_host":    migration.SourceHostName,
		"dest_host":      migration.DestHostName,
		"migration_type": migrationType,
	}, "")

	// Dispatch migration asynchronously.
	go func() {
		migrateCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		s.executeMigration(migrateCtx, migration)
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"migration": migration,
		"message":   fmt.Sprintf("migration %s queued: %s -> %s", migration.UUID, migration.SourceHostName, migration.DestHostName),
	})
}

// executeMigration orchestrates the migration process.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) executeMigration(ctx context.Context, m *Migration) {
	s.logger.Info("starting migration",
		zap.String("migration_uuid", m.UUID),
		zap.String("instance", m.InstanceName),
		zap.String("source", m.SourceHostName),
		zap.String("dest", m.DestHostName),
		zap.String("type", m.MigrationType))

	now := time.Now()
	s.updateMigrationStatus(m.ID, "preparing", 10, &now, nil, "")

	// Step 1: Pre-migration checks on destination host.
	if err := s.preMigrationCheck(ctx, m); err != nil {
		s.failMigration(m, "pre-migration check failed: "+err.Error())
		return
	}
	s.updateMigrationStatus(m.ID, "preparing", 25, nil, nil, "")

	// Step 2: Send migration command to source node.
	if err := s.sendMigrationCommand(ctx, m); err != nil {
		s.failMigration(m, "migration command failed: "+err.Error())
		return
	}
	s.updateMigrationStatus(m.ID, "migrating", 50, nil, nil, "")

	// Step 3: Wait for migration to complete (poll source node).
	if err := s.waitForMigration(ctx, m); err != nil {
		s.failMigration(m, "migration failed: "+err.Error())
		return
	}
	s.updateMigrationStatus(m.ID, "migrating", 90, nil, nil, "")

	// Step 4: Post-migration: update instance's host assignment.
	if err := s.postMigration(m); err != nil {
		s.failMigration(m, "post-migration update failed: "+err.Error())
		return
	}

	completed := time.Now()
	s.updateMigrationStatus(m.ID, "completed", 100, nil, &completed, "")

	s.logger.Info("migration completed successfully",
		zap.String("migration_uuid", m.UUID),
		zap.String("instance", m.InstanceName),
		zap.Duration("duration", completed.Sub(now)))

	s.emitEvent("instance", m.InstanceUUID, "migrate", "completed", "", map[string]interface{}{
		"migration_id": m.UUID,
		"source_host":  m.SourceHostName,
		"dest_host":    m.DestHostName,
		"duration_sec": completed.Sub(now).Seconds(),
	}, "")
}

// preMigrationCheck verifies the destination host can accept the instance.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) preMigrationCheck(ctx context.Context, m *Migration) error {
	// Verify dest host is still up.
	var status string
	if err := s.db.Table("hosts").Select("status").Where("uuid = ?", m.DestHostID).Scan(&status).Error; err != nil {
		return fmt.Errorf("destination host lookup failed: %w", err)
	}
	if status != "up" {
		return fmt.Errorf("destination host is no longer available (status: %s)", status)
	}

	// TODO: Check destination has sufficient resources (CPU, RAM, disk).
	// TODO: Verify network connectivity between source and dest.
	return nil
}

// sendMigrationCommand sends the migrate command to the source compute node.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) sendMigrationCommand(ctx context.Context, m *Migration) error {
	if m.SourceNodeAddr == "" {
		return fmt.Errorf("source node address is empty")
	}

	// For now, this is a placeholder. In production:
	// 1. For live migration: POST to source node's /api/v1/vms/{vmid}/migrate
	//    with dest_host, dest_port, migration_type (pre-copy, post-copy).
	// 2. Source node calls QEMU's migrate command via QMP.
	// 3. The dest node should already have the VM definition prepared.

	s.logger.Info("migration command sent to source node",
		zap.String("source_addr", m.SourceNodeAddr),
		zap.String("dest_addr", m.DestNodeAddr),
		zap.String("instance", m.InstanceName))

	return nil
}

// waitForMigration polls migration status until completion or timeout.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) waitForMigration(ctx context.Context, m *Migration) error {
	// In production: poll source node for QEMU migration status.
	// QEMU reports: setup, active, completed, failed, cancelling, cancelled.
	//
	// For now, this is a simulated wait. Real implementation would:
	// 1. Poll GET /api/v1/vms/{vmid}/migration-status on the source node.
	// 2. Update progress based on memory_remaining / memory_total.
	// 3. Detect completion or failure.

	s.logger.Info("waiting for migration to complete",
		zap.String("migration_uuid", m.UUID))

	return nil
}

// postMigration updates the instance's host assignment after successful migration.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) postMigration(m *Migration) error {
	// Update instance to point to the new host.
	err := s.db.Model(&Instance{}).Where("id = ?", m.InstanceID).Updates(map[string]interface{}{
		"host_id":      m.DestHostID,
		"node_address": m.DestNodeAddr,
		"status":       "active",
		"power_state":  "running",
	}).Error
	if err != nil {
		return fmt.Errorf("failed to update instance host: %w", err)
	}

	// Update resource allocation on hosts (move allocation from source to dest).
	var instance Instance
	if err := s.db.First(&instance, m.InstanceID).Error; err == nil {
		var flavor Flavor
		if err := s.db.First(&flavor, instance.FlavorID).Error; err == nil {
			// Decrease source host allocation.
			s.db.Exec("UPDATE hosts SET cpu_allocated = GREATEST(cpu_allocated - ?, 0), ram_allocated_mb = GREATEST(ram_allocated_mb - ?, 0) WHERE uuid = ?",
				flavor.VCPUs, flavor.RAM, m.SourceHostID)
			// Increase dest host allocation.
			s.db.Exec("UPDATE hosts SET cpu_allocated = cpu_allocated + ?, ram_allocated_mb = ram_allocated_mb + ? WHERE uuid = ?",
				flavor.VCPUs, flavor.RAM, m.DestHostID)
		}
	}

	return nil
}

// failMigration marks a migration as failed and restores instance state.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) failMigration(m *Migration, reason string) {
	s.logger.Error("migration failed",
		zap.String("migration_uuid", m.UUID),
		zap.String("instance", m.InstanceName),
		zap.String("reason", reason))

	now := time.Now()
	s.updateMigrationStatus(m.ID, "failed", 0, nil, &now, reason)

	// Restore instance to active state on the original host.
	_ = s.db.Model(&Instance{}).Where("id = ?", m.InstanceID).Updates(map[string]interface{}{
		"status":      "active",
		"power_state": "running",
	}).Error

	s.emitEvent("instance", m.InstanceUUID, "migrate", "failed", reason, map[string]interface{}{
		"migration_id": m.UUID,
	}, "")
}

// updateMigrationStatus updates the migration progress in the database.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) updateMigrationStatus(id uint, status string, progress int, startedAt, completedAt *time.Time, errorMsg string) {
	updates := map[string]interface{}{
		"status":   status,
		"progress": progress,
	}
	if startedAt != nil {
		updates["started_at"] = startedAt
	}
	if completedAt != nil {
		updates["completed_at"] = completedAt
	}
	if errorMsg != "" {
		updates["error_message"] = errorMsg
	}
	_ = s.db.Model(&Migration{}).Where("id = ?", id).Updates(updates).Error
}

// listMigrations handles GET /api/v1/migrations.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) listMigrations(c *gin.Context) {
	var migrations []Migration
	query := s.db.Order("id DESC")

	if status := c.Query("status"); status != "" {
		query = query.Where("status = ?", status)
	}
	if instanceID := c.Query("instance_id"); instanceID != "" {
		query = query.Where("instance_id = ?", instanceID)
	}

	query = query.Limit(50) // Pagination safety.

	if err := query.Find(&migrations).Error; err != nil {
		s.logger.Error("failed to list migrations", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list migrations"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"migrations": migrations, "total": len(migrations)})
}

// getMigration handles GET /api/v1/migrations/:id.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) getMigration(c *gin.Context) {
	id := c.Param("id")
	var migration Migration

	// Try UUID first, then numeric ID.
	err := s.db.Where("uuid = ?", id).First(&migration).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&migration, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "migration not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"migration": migration})
}

// cancelMigration handles POST /api/v1/migrations/:id/cancel.
//
//nolint:unused // TODO: wire into routes when feature is enabled
func (s *Service) cancelMigration(c *gin.Context) {
	id := c.Param("id")
	var migration Migration

	// Try UUID first, then numeric ID.
	err := s.db.Where("uuid = ?", id).First(&migration).Error
	if err == gorm.ErrRecordNotFound {
		err = s.db.First(&migration, id).Error
	}
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "migration not found"})
		return
	}

	if migration.Status == "completed" || migration.Status == "failed" || migration.Status == "cancelled" {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("migration is already %s", migration.Status),
		})
		return
	}

	now := time.Now()
	s.updateMigrationStatus(migration.ID, "cancelled", 0, nil, &now, "cancelled by user")

	// Restore instance to active state.
	_ = s.db.Model(&Instance{}).Where("id = ?", migration.InstanceID).Updates(map[string]interface{}{
		"status":      "active",
		"power_state": "running",
	}).Error

	// TODO: Send cancel command to source node's QEMU via QMP.

	c.JSON(http.StatusOK, gin.H{"ok": true, "message": "migration cancelled"})
}
