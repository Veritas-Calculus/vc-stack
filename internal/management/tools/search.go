// Package tools provides the global resource search endpoint.
package tools

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SearchResult represents a single search result item.
type SearchResult struct {
	Type     string `json:"type"`     // instance, volume, network, image, security_group, project
	ID       string `json:"id"`       // resource ID
	Name     string `json:"name"`     // display name
	Subtitle string `json:"subtitle"` // additional context (status, CIDR, etc.)
}

// setupSearchRoutes registers the global search endpoint.
func (s *Service) setupSearchRoutes(router *gin.Engine) {
	router.GET("/api/v1/search", s.globalSearch)
}

// globalSearch handles GET /api/v1/search?q=<term>&limit=<n>.
// It searches across instances, volumes, networks, images, and security groups
// using a case-insensitive LIKE query on the name or ID column.
func (s *Service) globalSearch(c *gin.Context) {
	q := strings.TrimSpace(c.Query("q"))
	if q == "" || len(q) < 2 {
		c.JSON(http.StatusOK, gin.H{"results": []SearchResult{}, "total": 0})
		return
	}

	limit := 10
	if l := c.Query("limit"); l != "" {
		// Simple parse; ignore errors, default to 10.
		var parsed int
		if _, err := parseIntSafe(l); err == nil {
			parsed = int(mustParseInt(l))
		}
		if parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	pattern := "%" + q + "%"
	var results []SearchResult

	// Search instances (Model: pkg/models.Instance or compute table).
	results = append(results, s.searchTable("instances", "instance", pattern, limit)...)

	// Search volumes.
	results = append(results, s.searchTable("storage_volumes", "volume", pattern, limit)...)

	// Search networks.
	results = append(results, s.searchTable("net_networks", "network", pattern, limit)...)

	// Search images.
	results = append(results, s.searchTable("images", "image", pattern, limit)...)

	// Search security groups.
	results = append(results, s.searchTable("net_security_groups", "security_group", pattern, limit)...)

	// Trim total results.
	if len(results) > limit {
		results = results[:limit]
	}

	c.JSON(http.StatusOK, gin.H{"results": results, "total": len(results)})
}

// searchTable runs a generic name/id search on a table with id+name columns.
func (s *Service) searchTable(table, resourceType, pattern string, limit int) []SearchResult {
	type row struct {
		ID     string `gorm:"column:id"`
		Name   string `gorm:"column:name"`
		Status string `gorm:"column:status"`
	}

	var rows []row

	// Try name + id search. Use a raw query to handle tables that might not
	// have a status column gracefully.
	q := s.db.Table(table).
		Select("id, name").
		Where("LOWER(name) LIKE LOWER(?) OR LOWER(CAST(id AS TEXT)) LIKE LOWER(?)", pattern, pattern).
		Limit(limit)

	// Attempt to also select status if the column exists.
	if hasColumn(s.db, table, "status") {
		q = s.db.Table(table).
			Select("id, name, status").
			Where("LOWER(name) LIKE LOWER(?) OR LOWER(CAST(id AS TEXT)) LIKE LOWER(?)", pattern, pattern).
			Limit(limit)
	}

	if err := q.Find(&rows).Error; err != nil {
		// Table might not exist in this deployment; silently skip.
		return nil
	}

	var results []SearchResult
	for _, r := range rows {
		subtitle := resourceType
		if r.Status != "" {
			subtitle = r.Status
		}
		results = append(results, SearchResult{
			Type:     resourceType,
			ID:       r.ID,
			Name:     r.Name,
			Subtitle: subtitle,
		})
	}
	return results
}

// hasColumn checks if a table has a specific column.
func hasColumn(db *gorm.DB, table, column string) bool {
	var count int64
	db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_name = ? AND column_name = ?",
		table, column).Scan(&count)
	return count > 0
}

// parseIntSafe is a simple int parser that returns error on failure.
func parseIntSafe(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

// mustParseInt parses a string as int, returning 0 on failure.
func mustParseInt(s string) int64 {
	n, _ := parseIntSafe(s)
	return n
}
