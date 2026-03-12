package logging

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	return db
}

func setupTestService(t *testing.T) *Service {
	t.Helper()
	svc, err := NewService(Config{DB: setupTestDB(t), Logger: zap.NewNop()})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	return svc
}

func TestIngest(t *testing.T) {
	svc := setupTestService(t)

	err := svc.Ingest(&LogEntry{
		Level:     "info",
		Source:    "vc-management",
		Component: "identity",
		Message:   "user logged in: admin",
	})
	if err != nil {
		t.Fatalf("Ingest error: %v", err)
	}
}

func TestIngestBatch(t *testing.T) {
	svc := setupTestService(t)

	entries := []LogEntry{
		{Level: "info", Source: "vc-management", Message: "msg 1", Timestamp: time.Now()},
		{Level: "warn", Source: "vc-compute", Message: "msg 2", Timestamp: time.Now()},
		{Level: "error", Source: "vc-management", Message: "msg 3", Timestamp: time.Now()},
	}
	if err := svc.IngestBatch(entries); err != nil {
		t.Fatalf("IngestBatch error: %v", err)
	}
}

func TestQueryByLevel(t *testing.T) {
	svc := setupTestService(t)

	svc.Ingest(&LogEntry{Level: "info", Source: "vc-management", Message: "info msg", Timestamp: time.Now()})
	svc.Ingest(&LogEntry{Level: "error", Source: "vc-management", Message: "error msg", Timestamp: time.Now()})
	svc.Ingest(&LogEntry{Level: "info", Source: "vc-compute", Message: "info 2", Timestamp: time.Now()})

	entries, total, err := svc.Query(&LogQueryRequest{Level: "error"})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 error log, got %d", total)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(entries))
	}
}

func TestQueryBySource(t *testing.T) {
	svc := setupTestService(t)

	svc.Ingest(&LogEntry{Level: "info", Source: "vc-management", Message: "m1", Timestamp: time.Now()})
	svc.Ingest(&LogEntry{Level: "info", Source: "vc-compute", Message: "c1", Timestamp: time.Now()})

	entries, total, err := svc.Query(&LogQueryRequest{Source: "vc-compute"})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1, got %d", total)
	}
	if entries[0].Source != "vc-compute" {
		t.Errorf("got source %q", entries[0].Source)
	}
}

func TestQueryFullTextSearch(t *testing.T) {
	svc := setupTestService(t)

	svc.Ingest(&LogEntry{Level: "info", Source: "vc-management", Message: "instance i-abc123 created", Timestamp: time.Now()})
	svc.Ingest(&LogEntry{Level: "info", Source: "vc-management", Message: "network created", Timestamp: time.Now()})

	entries, _, err := svc.Query(&LogQueryRequest{Search: "i-abc123"})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 match, got %d", len(entries))
	}
}

func TestQueryPagination(t *testing.T) {
	svc := setupTestService(t)

	for i := 0; i < 20; i++ {
		svc.Ingest(&LogEntry{Level: "info", Source: "vc-management", Message: "msg", Timestamp: time.Now()})
	}

	entries, total, err := svc.Query(&LogQueryRequest{Limit: 5, Offset: 0})
	if err != nil {
		t.Fatalf("Query error: %v", err)
	}
	if total != 20 {
		t.Errorf("expected total 20, got %d", total)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}
