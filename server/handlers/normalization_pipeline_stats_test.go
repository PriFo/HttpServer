package handlers_test

import (
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"httpserver/database"
	"httpserver/server/handlers"
)

// TestHandlePipelineStats verifies that the legacy handler returns rich stage data
// that matches what the frontend expects.
func TestHandlePipelineStats_ReturnsStageStats(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "pipeline_stats.db")

	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("failed to create test database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	sqlDB := db.GetDB()

	// Seed two normalized records with different stage progress to exercise the aggregations.
	insert := `
		INSERT INTO normalized_data (
			source_reference, source_name, code, normalized_name,
			stage05_completed, stage1_completed, stage2_completed, stage25_completed,
			stage3_completed, stage35_completed, stage4_completed, stage5_completed,
			stage6_completed, stage6_classifier_confidence, stage65_completed,
			stage7_ai_processed, stage8_completed, stage8_manual_review_required,
			stage9_completed, stage10_exported, stage11_kpved_completed,
			stage12_okpd2_completed, final_completed, final_confidence, final_completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, datetime('now'))
	`

	_, err = sqlDB.Exec(insert,
		"ref-1", "Item 1", "CODE1", "Normalized 1",
		1, 1, 1, 1,
		1, 1, 1, 1,
		1, 0.92, 1,
		1, 1, 0,
		1, 1, 1,
		1, 1, 0.95,
	)
	if err != nil {
		t.Fatalf("failed to seed normalized_data: %v", err)
	}

	_, err = sqlDB.Exec(`
		INSERT INTO normalized_data (
			source_reference, source_name, code, normalized_name,
			stage05_completed, stage1_completed, stage2_completed,
			stage7_ai_processed, stage6_classifier_confidence,
			stage8_manual_review_required, final_confidence
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		"ref-2", "Item 2", "CODE2", "Normalized 2",
		1, 0, 0,
		0, 0.0,
		1, 0.40,
	)
	if err != nil {
		t.Fatalf("failed to seed second normalized_data row: %v", err)
	}

	handler := handlers.NewNormalizationHandler(nil, handlers.NewBaseHandlerFromMiddleware(), nil)
	handler.SetDatabase(db, dbPath, nil, "")

	req := httptest.NewRequest("GET", "/api/normalization/pipeline/stats", nil)
	rr := httptest.NewRecorder()

	handler.HandlePipelineStats(rr, req)

	if rr.Code != 200 {
		t.Fatalf("unexpected status %d, body=%s", rr.Code, rr.Body.String())
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if payload["total_records"] != float64(2) {
		t.Fatalf("expected total_records=2, got %v", payload["total_records"])
	}

	stageStats, ok := payload["stage_stats"].([]interface{})
	if !ok || len(stageStats) == 0 {
		t.Fatalf("stage_stats is missing or empty: %v", payload["stage_stats"])
	}

	if _, ok := payload["quality_metrics"].(map[string]interface{}); !ok {
		t.Fatalf("quality_metrics should be present: %v", payload["quality_metrics"])
	}

	if _, ok := payload["overall_progress"]; !ok {
		t.Fatalf("overall_progress should be present in response")
	}
}
