package services

import (
	"context"
	"fmt"

	normalizationdomain "httpserver/internal/domain/normalization"
	"httpserver/server/services"
)

// benchmarkAdapter адаптер для services.BenchmarkService
type benchmarkAdapter struct {
	benchmarkService *services.BenchmarkService
}

// NewBenchmarkAdapter создает новый адаптер для сервиса эталонов
func NewBenchmarkAdapter(benchmarkService *services.BenchmarkService) normalizationdomain.BenchmarkServiceInterface {
	return &benchmarkAdapter{
		benchmarkService: benchmarkService,
	}
}

// FindBestMatch находит лучший эталон для заданного имени и типа сущности
func (a *benchmarkAdapter) FindBestMatch(ctx context.Context, name string, entityType string) (*normalizationdomain.BenchmarkMatch, error) {
	if a.benchmarkService == nil {
		return nil, fmt.Errorf("benchmark service is not initialized")
	}

	benchmark, err := a.benchmarkService.FindBestMatch(name, entityType)
	if err != nil {
		return nil, fmt.Errorf("failed to find benchmark: %w", err)
	}

	if benchmark == nil {
		return nil, fmt.Errorf("benchmark not found")
	}

	return &normalizationdomain.BenchmarkMatch{
		Name:       benchmark.Name,
		EntityType: entityType,
		Confidence: 1.0, // BenchmarkService возвращает точное совпадение
		Source:     "benchmark",
	}, nil
}

