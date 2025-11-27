package models

import (
	"time"
)

// Benchmark основная структура эталона
type Benchmark struct {
	ID             string                 `json:"id"`
	EntityType     string                 `json:"entity_type"`
	Name           string                 `json:"name"`
	Data           map[string]interface{} `json:"data"`
	SourceUploadID *int                   `json:"source_upload_id,omitempty"`
	SourceClientID *int                   `json:"source_client_id,omitempty"`
	IsActive       bool                   `json:"is_active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Variations     []string               `json:"variations,omitempty"`
}

// BenchmarkVariation структура для вариаций эталона
type BenchmarkVariation struct {
	ID          int    `json:"id"`
	BenchmarkID string `json:"benchmark_id"`
	Variation   string `json:"variation"`
}

// CreateBenchmarkRequest запрос на создание эталона
type CreateBenchmarkRequest struct {
	EntityType     string                 `json:"entity_type" binding:"required"`
	Name           string                 `json:"name" binding:"required"`
	Data           map[string]interface{} `json:"data"`
	SourceUploadID *int                   `json:"source_upload_id,omitempty"`
	SourceClientID *int                   `json:"source_client_id,omitempty"`
	Variations     []string               `json:"variations,omitempty"`
}

// CreateBenchmarkFromUploadRequest запрос на создание эталона из загрузки
type CreateBenchmarkFromUploadRequest struct {
	UploadID   string   `json:"upload_id" binding:"required"`
	ItemIDs    []string `json:"item_ids" binding:"required"`
	EntityType string   `json:"entity_type" binding:"required"`
}

// UpdateBenchmarkRequest запрос на обновление эталона
type UpdateBenchmarkRequest struct {
	EntityType string                 `json:"entity_type,omitempty"`
	Name       string                 `json:"name,omitempty"`
	Data       map[string]interface{} `json:"data,omitempty"`
	IsActive   *bool                  `json:"is_active,omitempty"`
	Variations []string               `json:"variations,omitempty"`
}

// SearchBenchmarkRequest запрос на поиск эталона
type SearchBenchmarkRequest struct {
	Name       string `json:"name" binding:"required"`
	EntityType string `json:"entity_type" binding:"required"`
}

// ListBenchmarksRequest запрос на получение списка эталонов
type ListBenchmarksRequest struct {
	EntityType string `json:"entity_type,omitempty"`
	ActiveOnly bool   `json:"active_only,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
}

// BenchmarkListResponse ответ со списком эталонов
type BenchmarkListResponse struct {
	Benchmarks []*Benchmark `json:"benchmarks"`
	Total      int          `json:"total"`
	Limit      int          `json:"limit"`
	Offset     int          `json:"offset"`
}
