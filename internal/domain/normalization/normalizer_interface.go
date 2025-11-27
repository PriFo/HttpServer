package normalization

import (
	"context"
)

// NormalizerInterface интерфейс для нормализатора данных
// Абстрагирует работу с normalization.Normalizer
type NormalizerInterface interface {
	// ProcessNormalization выполняет полный процесс нормализации данных
	ProcessNormalization(ctx context.Context, uploadID int) error
	
	// NormalizeName нормализует название с использованием AI и эталонов
	NormalizeName(ctx context.Context, name string, entityType string) (string, error)
	
	// GetAINormalizer возвращает AI нормализатор для прямого доступа (опционально)
	GetAINormalizer() interface{} // *normalization.AINormalizer
}

// BenchmarkServiceInterface интерфейс для сервиса эталонов
// Абстрагирует работу с BenchmarkService
type BenchmarkServiceInterface interface {
	// FindBestMatch находит лучший эталон для заданного имени и типа сущности
	FindBestMatch(ctx context.Context, name string, entityType string) (*BenchmarkMatch, error)
}

// BenchmarkMatch результат поиска эталона
type BenchmarkMatch struct {
	Name         string
	EntityType   string
	Confidence   float64
	Source       string // "benchmark", "ai"
}

