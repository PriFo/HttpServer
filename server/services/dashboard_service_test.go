package services

import (
	"path/filepath"
	"testing"

	"httpserver/database"
)

func TestDashboardService_GetStats_UsesCustomFunc(t *testing.T) {
	service := NewDashboardService(
		nil,
		nil,
		nil,
		func() map[string]interface{} {
			return map[string]interface{}{
				"totalRecords": 42,
				"isMock":       true,
			}
		},
		nil,
	)

	stats, err := service.GetStats()
	if err != nil {
		t.Fatalf("GetStats() returned error: %v", err)
	}

	if stats["totalRecords"] != 42 {
		t.Fatalf("expected totalRecords=42, got %v", stats["totalRecords"])
	}
	if stats["isMock"] != true {
		t.Fatalf("expected isMock flag in stats, got %v", stats["isMock"])
	}
}

func TestDashboardService_GetStats_FallbackToDatabase(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "dashboard.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	defer db.Close()

	service := NewDashboardService(db, nil, nil, nil, nil)

	stats, err := service.GetStats()
	if err != nil {
		t.Fatalf("GetStats() returned error: %v", err)
	}

	if stats == nil {
		t.Fatal("expected stats map, got nil")
	}
	if _, ok := stats["total_items"]; !ok {
		t.Fatalf("expected total_items in stats, got %v", stats)
	}
}

