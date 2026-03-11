// Package pagination provides standardized cursor-based pagination for
// VC Stack List APIs. It supports marker/limit paging with sort options.
package pagination

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Default and limit constants.
const (
	DefaultLimit = 20
	MaxLimit     = 1000
)

// Params holds parsed pagination parameters from a request.
type Params struct {
	Marker  string `json:"marker,omitempty"`   // ID of last item from previous page
	Limit   int    `json:"limit"`              // max items to return
	SortKey string `json:"sort_key,omitempty"` // column to sort by
	SortDir string `json:"sort_dir,omitempty"` // asc or desc
}

// PageResult wraps a paginated response with metadata.
type PageResult struct {
	Items      interface{} `json:"items"`
	NextMarker string      `json:"next_marker,omitempty"` // empty = last page
	TotalCount int64       `json:"total_count"`
	Limit      int         `json:"limit"`
	HasMore    bool        `json:"has_more"`
}

// ParseParams extracts pagination parameters from query string.
// Defaults: limit=20, sort_dir=desc, sort_key=created_at.
func ParseParams(c *gin.Context) Params {
	p := Params{
		Marker:  c.Query("marker"),
		SortKey: c.DefaultQuery("sort_key", "created_at"),
		SortDir: c.DefaultQuery("sort_dir", "desc"),
	}

	if l, err := strconv.Atoi(c.DefaultQuery("limit", "20")); err == nil {
		p.Limit = l
	} else {
		p.Limit = DefaultLimit
	}

	// Clamp limit.
	if p.Limit <= 0 {
		p.Limit = DefaultLimit
	}
	if p.Limit > MaxLimit {
		p.Limit = MaxLimit
	}

	// Validate sort direction.
	if p.SortDir != "asc" && p.SortDir != "desc" {
		p.SortDir = "desc"
	}

	return p
}

// ApplyToQuery adds ORDER BY, LIMIT, and marker-based WHERE to a GORM query.
// The markerColumn is the column used for marker-based pagination (usually "id").
func ApplyToQuery(db *gorm.DB, p Params, markerColumn string) *gorm.DB {
	q := db

	// Sort.
	if p.SortKey != "" {
		order := p.SortKey + " " + p.SortDir
		q = q.Order(order)
	}

	// Marker-based cursor: if marker is set, skip records before/after the marker.
	if p.Marker != "" && markerColumn != "" {
		if p.SortDir == "asc" {
			q = q.Where(markerColumn+" > ?", p.Marker)
		} else {
			q = q.Where(markerColumn+" < ?", p.Marker)
		}
	}

	// Limit (fetch one extra to determine has_more).
	q = q.Limit(p.Limit + 1)

	return q
}

// BuildResult constructs a PageResult from the query results.
// items should be a pointer to a slice. The function checks if there are
// more items beyond the limit by inspecting the extra fetched item.
func BuildResult(items interface{}, totalCount int64, p Params, getID func(i int) string, length int) PageResult {
	hasMore := length > p.Limit
	nextMarker := ""

	if hasMore {
		nextMarker = getID(p.Limit - 1)
	}

	return PageResult{
		Items:      items,
		NextMarker: nextMarker,
		TotalCount: totalCount,
		Limit:      p.Limit,
		HasMore:    hasMore,
	}
}
