package server

import (
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestApplyFilters_StatusFilter(t *testing.T) {
	summary := &SystemSummary{
		UploadDetails: []UploadSummary{
			{Status: "completed", Name: "Test1"},
			{Status: "failed", Name: "Test2"},
			{Status: "in_progress", Name: "Test3"},
			{Status: "completed", Name: "Test4"},
		},
	}

	filter := SystemSummaryFilter{
		Status: []string{"completed"},
	}

	result := ApplyFilters(summary, filter)

	if len(result.UploadDetails) != 2 {
		t.Errorf("Expected 2 completed uploads, got %d", len(result.UploadDetails))
	}

	for _, upload := range result.UploadDetails {
		if upload.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", upload.Status)
		}
	}
}

func TestApplyFilters_SearchFilter(t *testing.T) {
	summary := &SystemSummary{
		UploadDetails: []UploadSummary{
			{Name: "БухгалтерияДляКазахстана", UploadUUID: "uuid1"},
			{Name: "БухгалтерияДляРоссии", UploadUUID: "uuid2"},
			{Name: "Казахстан", UploadUUID: "uuid3"},
		},
	}

	filter := SystemSummaryFilter{
		Search: "Казахстан",
	}

	result := ApplyFilters(summary, filter)

	if len(result.UploadDetails) != 2 {
		t.Errorf("Expected 2 uploads matching 'Казахстан', got %d", len(result.UploadDetails))
	}
}

func TestApplyFilters_DateFilter(t *testing.T) {
	now := time.Now()
	summary := &SystemSummary{
		UploadDetails: []UploadSummary{
			{CreatedAt: now.Add(-2 * time.Hour)},
			{CreatedAt: now.Add(-1 * time.Hour)},
			{CreatedAt: now},
		},
	}

	createdAfter := now.Add(-90 * time.Minute)
	filter := SystemSummaryFilter{
		CreatedAfter: &createdAfter,
	}

	result := ApplyFilters(summary, filter)

	if len(result.UploadDetails) != 2 {
		t.Errorf("Expected 2 uploads after %v, got %d", createdAfter, len(result.UploadDetails))
	}
}

func TestApplyFilters_SortBy(t *testing.T) {
	summary := &SystemSummary{
		UploadDetails: []UploadSummary{
			{Name: "C", CreatedAt: time.Now().Add(-3 * time.Hour)},
			{Name: "A", CreatedAt: time.Now().Add(-1 * time.Hour)},
			{Name: "B", CreatedAt: time.Now().Add(-2 * time.Hour)},
		},
	}

	filter := SystemSummaryFilter{
		SortBy: "name",
		Order:  "asc",
	}

	result := ApplyFilters(summary, filter)

	if len(result.UploadDetails) != 3 {
		t.Fatalf("Expected 3 uploads, got %d", len(result.UploadDetails))
	}

	if result.UploadDetails[0].Name != "A" {
		t.Errorf("Expected first upload name 'A', got '%s'", result.UploadDetails[0].Name)
	}
	if result.UploadDetails[1].Name != "B" {
		t.Errorf("Expected second upload name 'B', got '%s'", result.UploadDetails[1].Name)
	}
	if result.UploadDetails[2].Name != "C" {
		t.Errorf("Expected third upload name 'C', got '%s'", result.UploadDetails[2].Name)
	}
}

func TestApplyFilters_Pagination(t *testing.T) {
	summary := &SystemSummary{
		UploadDetails: make([]UploadSummary, 10),
	}

	for i := 0; i < 10; i++ {
		summary.UploadDetails[i] = UploadSummary{Name: "Test"}
	}

	filter := SystemSummaryFilter{
		Limit:  3,
		Offset: 2,
	}

	result := ApplyFilters(summary, filter)

	if len(result.UploadDetails) != 3 {
		t.Errorf("Expected 3 uploads with limit=3, offset=2, got %d", len(result.UploadDetails))
	}
}

func TestApplyFilters_RecalculateStats(t *testing.T) {
	summary := &SystemSummary{
		UploadDetails: []UploadSummary{
			{Status: "completed", NomenclatureCount: 100, CounterpartyCount: 50},
			{Status: "failed", NomenclatureCount: 200, CounterpartyCount: 75},
			{Status: "completed", NomenclatureCount: 150, CounterpartyCount: 60},
		},
	}

	filter := SystemSummaryFilter{
		Status: []string{"completed"},
	}

	result := ApplyFilters(summary, filter)

	if result.CompletedUploads != 2 {
		t.Errorf("Expected 2 completed uploads, got %d", result.CompletedUploads)
	}
	if result.TotalNomenclature != 250 {
		t.Errorf("Expected 250 total nomenclature, got %d", result.TotalNomenclature)
	}
	if result.TotalCounterparties != 110 {
		t.Errorf("Expected 110 total counterparties, got %d", result.TotalCounterparties)
	}
}

func TestParseSystemSummaryFilterFromRequest(t *testing.T) {
	req := &http.Request{
		URL: &url.URL{
			RawQuery: "status=completed&search=Test&sort_by=name&order=asc&limit=10&page=2",
		},
	}

	filter := ParseSystemSummaryFilterFromRequest(req)

	if len(filter.Status) != 1 || filter.Status[0] != "completed" {
		t.Errorf("Expected status 'completed', got %v", filter.Status)
	}
	if filter.Search != "Test" {
		t.Errorf("Expected search 'Test', got '%s'", filter.Search)
	}
	if filter.SortBy != "name" {
		t.Errorf("Expected sort_by 'name', got '%s'", filter.SortBy)
	}
	if filter.Order != "asc" {
		t.Errorf("Expected order 'asc', got '%s'", filter.Order)
	}
	if filter.Limit != 10 {
		t.Errorf("Expected limit 10, got %d", filter.Limit)
	}
	if filter.Offset != 10 { // page 2 with limit 10 = offset 10
		t.Errorf("Expected offset 10, got %d", filter.Offset)
	}
}

func TestParseSystemSummaryFilterFromRequest_MultipleStatus(t *testing.T) {
	req := &http.Request{
		URL: &url.URL{
			RawQuery: "status=completed,failed",
		},
	}

	filter := ParseSystemSummaryFilterFromRequest(req)

	if len(filter.Status) != 2 {
		t.Errorf("Expected 2 statuses, got %d", len(filter.Status))
	}
}

func TestParseSystemSummaryFilterFromRequest_DateFilter(t *testing.T) {
	req := &http.Request{
		URL: &url.URL{
			RawQuery: "created_after=2025-01-01T00:00:00Z&created_before=2025-01-31T23:59:59Z",
		},
	}

	filter := ParseSystemSummaryFilterFromRequest(req)

	if filter.CreatedAfter == nil {
		t.Error("Expected CreatedAfter to be set")
	}
	if filter.CreatedBefore == nil {
		t.Error("Expected CreatedBefore to be set")
	}
}

func TestCompareUploadSummary(t *testing.T) {
	now := time.Now()
	a := UploadSummary{Name: "A", CreatedAt: now.Add(-1 * time.Hour)}
	b := UploadSummary{Name: "B", CreatedAt: now}

	// Test by name ascending
	if !compareUploadSummary(a, b, "name", false) {
		t.Error("Expected A < B by name")
	}

	// Test by created_at ascending
	if !compareUploadSummary(a, b, "created_at", false) {
		t.Error("Expected a.CreatedAt < b.CreatedAt")
	}

	// Test descending
	if compareUploadSummary(a, b, "name", true) {
		t.Error("Expected B > A by name (descending)")
	}
}

