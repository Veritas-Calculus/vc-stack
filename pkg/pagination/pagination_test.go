package pagination

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestParseParams_Defaults(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) {
		p := ParseParams(c)
		if p.Limit != DefaultLimit {
			t.Errorf("expected default limit %d, got %d", DefaultLimit, p.Limit)
		}
		if p.SortDir != "desc" {
			t.Errorf("expected default sort_dir desc, got %s", p.SortDir)
		}
		if p.SortKey != "created_at" {
			t.Errorf("expected default sort_key created_at, got %s", p.SortKey)
		}
		if p.Marker != "" {
			t.Errorf("expected empty marker, got %s", p.Marker)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)
}

func TestParseParams_CustomValues(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) {
		p := ParseParams(c)
		if p.Limit != 50 {
			t.Errorf("expected limit 50, got %d", p.Limit)
		}
		if p.SortKey != "name" {
			t.Errorf("expected sort_key name, got %s", p.SortKey)
		}
		if p.SortDir != "asc" {
			t.Errorf("expected sort_dir asc, got %s", p.SortDir)
		}
		if p.Marker != "abc-123" {
			t.Errorf("expected marker abc-123, got %s", p.Marker)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?limit=50&sort_key=name&sort_dir=asc&marker=abc-123", nil)
	r.ServeHTTP(w, req)
}

func TestParseParams_MaxLimit(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) {
		p := ParseParams(c)
		if p.Limit != MaxLimit {
			t.Errorf("expected max limit %d, got %d", MaxLimit, p.Limit)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?limit=9999", nil)
	r.ServeHTTP(w, req)
}

func TestParseParams_InvalidSortDir(t *testing.T) {
	r := setupRouter()
	r.GET("/test", func(c *gin.Context) {
		p := ParseParams(c)
		if p.SortDir != "desc" {
			t.Errorf("expected fallback sort_dir desc, got %s", p.SortDir)
		}
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?sort_dir=invalid", nil)
	r.ServeHTTP(w, req)
}

func TestBuildResult_HasMore(t *testing.T) {
	ids := []string{"a", "b", "c", "d", "e", "f"}
	p := Params{Limit: 5}

	result := BuildResult(ids, 100, p, func(i int) string { return ids[i] }, 6)

	if !result.HasMore {
		t.Error("expected has_more=true when length > limit")
	}
	if result.NextMarker != "e" {
		t.Errorf("expected next_marker=e, got %s", result.NextMarker)
	}
	if result.TotalCount != 100 {
		t.Errorf("expected total_count=100, got %d", result.TotalCount)
	}
}

func TestBuildResult_LastPage(t *testing.T) {
	ids := []string{"a", "b", "c"}
	p := Params{Limit: 5}

	result := BuildResult(ids, 3, p, func(i int) string { return ids[i] }, 3)

	if result.HasMore {
		t.Error("expected has_more=false on last page")
	}
	if result.NextMarker != "" {
		t.Errorf("expected empty next_marker, got %s", result.NextMarker)
	}
}
