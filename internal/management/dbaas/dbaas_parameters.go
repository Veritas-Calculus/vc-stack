package dbaas

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ──────────────────────────────────────────────────────────────────────
// DB Parameter Group Models
//
// Customizable configuration templates for PostgreSQL/MySQL instances.
// Equivalent to AWS RDS Parameter Groups.
// ──────────────────────────────────────────────────────────────────────

// DBParameterGroup is a named collection of database engine parameters.
type DBParameterGroup struct {
	ID          uint      `json:"id" gorm:"primarykey"`
	Name        string    `json:"name" gorm:"uniqueIndex;not null"`
	Engine      string    `json:"engine" gorm:"not null"` // postgresql, mysql
	Family      string    `json:"family" gorm:"not null"` // pg16, pg15, mysql8, mysql5.7
	Description string    `json:"description"`
	IsDefault   bool      `json:"is_default" gorm:"default:false"`
	ProjectID   uint      `json:"project_id" gorm:"index"`
	Parameters  string    `json:"parameters" gorm:"type:text"` // JSON map of key->DBParameterValue
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// DBParameterValue describes a single parameter.
type DBParameterValue struct {
	Value           string   `json:"value"`
	DataType        string   `json:"data_type"`                // string, integer, boolean, enum
	AllowedValues   []string `json:"allowed_values,omitempty"` // For enum type
	MinValue        *int     `json:"min_value,omitempty"`
	MaxValue        *int     `json:"max_value,omitempty"`
	RequiresRestart bool     `json:"requires_restart"`
	Description     string   `json:"description,omitempty"`
}

// ──────────────────────────────────────────────────────────────────────
// Handlers
// ──────────────────────────────────────────────────────────────────────

func (s *Service) handleListParameterGroups(c *gin.Context) {
	var groups []DBParameterGroup
	query := s.db
	if pid := c.Query("project_id"); pid != "" {
		query = query.Where("project_id = ? OR is_default = true", pid)
	}
	if engine := c.Query("engine"); engine != "" {
		query = query.Where("engine = ?", engine)
	}
	if err := query.Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list parameter groups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"parameter_groups": groups})
}

func (s *Service) handleCreateParameterGroup(c *gin.Context) {
	var req struct {
		Name        string                      `json:"name" binding:"required"`
		Engine      string                      `json:"engine" binding:"required"`
		Family      string                      `json:"family" binding:"required"`
		Description string                      `json:"description"`
		Parameters  map[string]DBParameterValue `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectID := extractProjectID(c)
	paramsJSON, _ := json.Marshal(req.Parameters)

	pg := DBParameterGroup{
		Name:        req.Name,
		Engine:      req.Engine,
		Family:      req.Family,
		Description: req.Description,
		ProjectID:   projectID,
		Parameters:  string(paramsJSON),
	}
	if err := s.db.Create(&pg).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create parameter group"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"parameter_group": pg})
}

func (s *Service) handleGetParameterGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("pgId"), 10, 32)
	var pg DBParameterGroup
	if err := s.db.First(&pg, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Parameter group not found"})
		return
	}
	// Deserialize parameters for response.
	var params map[string]DBParameterValue
	_ = json.Unmarshal([]byte(pg.Parameters), &params)
	c.JSON(http.StatusOK, gin.H{"parameter_group": pg, "parameters_parsed": params})
}

func (s *Service) handleUpdateParameterGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("pgId"), 10, 32)
	var pg DBParameterGroup
	if err := s.db.First(&pg, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Parameter group not found"})
		return
	}
	if pg.IsDefault {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot modify default parameter group; create a copy"})
		return
	}

	var req struct {
		Description string                      `json:"description"`
		Parameters  map[string]DBParameterValue `json:"parameters"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Description != "" {
		pg.Description = req.Description
	}
	if req.Parameters != nil {
		paramsJSON, _ := json.Marshal(req.Parameters)
		pg.Parameters = string(paramsJSON)
	}
	s.db.Save(&pg)
	c.JSON(http.StatusOK, gin.H{"parameter_group": pg})
}

func (s *Service) handleDeleteParameterGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("pgId"), 10, 32)
	var pg DBParameterGroup
	if err := s.db.First(&pg, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Parameter group not found"})
		return
	}
	if pg.IsDefault {
		c.JSON(http.StatusForbidden, gin.H{"error": "Cannot delete default parameter group"})
		return
	}
	s.db.Delete(&pg)
	c.JSON(http.StatusOK, gin.H{"message": "Parameter group deleted"})
}

// handleApplyParameterGroup applies a parameter group to DB instances.
func (s *Service) handleApplyParameterGroup(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("pgId"), 10, 32)
	var req struct {
		InstanceIDs []uint `json:"instance_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pg DBParameterGroup
	if err := s.db.First(&pg, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Parameter group not found"})
		return
	}

	// In production, this would generate config files and trigger a reload/restart.
	s.logger.Info("Applying parameter group to instances",
		zap.Uint("pg_id", pg.ID),
		zap.String("pg_name", pg.Name),
		zap.Int("instance_count", len(req.InstanceIDs)),
	)

	c.JSON(http.StatusOK, gin.H{
		"message":          "Parameter group applied",
		"applied_to":       req.InstanceIDs,
		"requires_restart": true,
	})
}

// seedDefaultParameterGroups creates built-in parameter groups if they don't exist.
func (s *Service) seedDefaultParameterGroups() {
	pgDefaults := map[string]DBParameterValue{
		"max_connections":            {Value: "200", DataType: "integer", MinValue: intPtr(10), MaxValue: intPtr(10000), RequiresRestart: true, Description: "Maximum concurrent connections"},
		"shared_buffers":             {Value: "256MB", DataType: "string", RequiresRestart: true, Description: "Memory for shared buffer cache"},
		"work_mem":                   {Value: "4MB", DataType: "string", RequiresRestart: false, Description: "Memory per sort/hash operation"},
		"maintenance_work_mem":       {Value: "64MB", DataType: "string", RequiresRestart: false, Description: "Memory for maintenance operations"},
		"effective_cache_size":       {Value: "1GB", DataType: "string", RequiresRestart: false, Description: "OS cache size estimate for planner"},
		"wal_level":                  {Value: "replica", DataType: "enum", AllowedValues: []string{"minimal", "replica", "logical"}, RequiresRestart: true},
		"max_wal_senders":            {Value: "10", DataType: "integer", MinValue: intPtr(0), MaxValue: intPtr(100), RequiresRestart: true},
		"log_min_duration_statement": {Value: "1000", DataType: "integer", RequiresRestart: false, Description: "Log statements exceeding N ms"},
	}
	pgJSON, _ := json.Marshal(pgDefaults)

	mysqlDefaults := map[string]DBParameterValue{
		"max_connections":         {Value: "200", DataType: "integer", MinValue: intPtr(10), MaxValue: intPtr(10000), RequiresRestart: false},
		"innodb_buffer_pool_size": {Value: "256M", DataType: "string", RequiresRestart: true, Description: "InnoDB buffer pool size"},
		"innodb_log_file_size":    {Value: "48M", DataType: "string", RequiresRestart: true},
		"slow_query_log":          {Value: "ON", DataType: "enum", AllowedValues: []string{"ON", "OFF"}, RequiresRestart: false},
		"long_query_time":         {Value: "1", DataType: "integer", RequiresRestart: false, Description: "Slow query threshold in seconds"},
		"binlog_format":           {Value: "ROW", DataType: "enum", AllowedValues: []string{"ROW", "STATEMENT", "MIXED"}, RequiresRestart: true},
	}
	mysqlJSON, _ := json.Marshal(mysqlDefaults)

	defaults := []DBParameterGroup{
		{Name: "pg16-default", Engine: "postgresql", Family: "pg16", Description: "Default PostgreSQL 16 parameters", IsDefault: true, Parameters: string(pgJSON)},
		{Name: "mysql8-default", Engine: "mysql", Family: "mysql8", Description: "Default MySQL 8.0 parameters", IsDefault: true, Parameters: string(mysqlJSON)},
	}

	for _, d := range defaults {
		var existing DBParameterGroup
		if err := s.db.Where("name = ?", d.Name).First(&existing).Error; err != nil {
			_ = s.db.Create(&d).Error
		}
	}
}

func intPtr(v int) *int { return &v }
