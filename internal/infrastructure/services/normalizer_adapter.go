package services

import (
	"context"
	"fmt"

	normalizationdomain "httpserver/internal/domain/normalization"
	"httpserver/normalization"
)

// normalizerAdapter адаптер для normalization.Normalizer
type normalizerAdapter struct {
	normalizer *normalization.Normalizer
}

// NewNormalizerAdapter создает новый адаптер для нормализатора
func NewNormalizerAdapter(normalizer *normalization.Normalizer) normalizationdomain.NormalizerInterface {
	return &normalizerAdapter{
		normalizer: normalizer,
	}
}

// ProcessNormalization выполняет полный процесс нормализации данных
func (a *normalizerAdapter) ProcessNormalization(ctx context.Context, uploadID int) error {
	if a.normalizer == nil {
		return fmt.Errorf("normalizer is not initialized")
	}
	return a.normalizer.ProcessNormalization(uploadID)
}

// NormalizeName нормализует название с использованием AI и эталонов
func (a *normalizerAdapter) NormalizeName(ctx context.Context, name string, entityType string) (string, error) {
	if a.normalizer == nil {
		return "", fmt.Errorf("normalizer is not initialized")
	}

	// Используем AI нормализатор, если доступен
	if aiNormalizer := a.normalizer.GetAINormalizer(); aiNormalizer != nil {
		result, err := aiNormalizer.NormalizeWithAI(name)
		if err != nil {
			return "", fmt.Errorf("AI normalization failed: %w", err)
		}
		return result.NormalizedName, nil
	}

	return "", fmt.Errorf("AI normalizer is not available")
}

// GetAINormalizer возвращает AI нормализатор для прямого доступа
func (a *normalizerAdapter) GetAINormalizer() interface{} {
	if a.normalizer == nil {
		return nil
	}
	return a.normalizer.GetAINormalizer()
}

