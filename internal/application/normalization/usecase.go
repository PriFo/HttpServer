package normalization

import (
	"context"
	"fmt"

	normalizationdomain "httpserver/internal/domain/normalization"
	"httpserver/internal/domain/repositories"
)

// UseCase представляет use case для работы с нормализацией
// Координирует выполнение бизнес-логики между domain и infrastructure слоями
type UseCase struct {
	normalizationRepo repositories.NormalizationRepository
	normalizationService normalizationdomain.Service
}

// NewUseCase создает новый use case для нормализации
func NewUseCase(
	normalizationRepo repositories.NormalizationRepository,
	normalizationService normalizationdomain.Service,
) *UseCase {
	return &UseCase{
		normalizationRepo:    normalizationRepo,
		normalizationService: normalizationService,
	}
}

// StartProcess запускает процесс нормализации
func (uc *UseCase) StartProcess(ctx context.Context, uploadID string) (*normalizationdomain.NormalizationProcess, error) {
	result, err := uc.normalizationService.StartProcess(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to start normalization process: %w", err)
	}
	return result, nil
}

// GetProcessStatus возвращает статус процесса нормализации
func (uc *UseCase) GetProcessStatus(ctx context.Context, processID string) (*normalizationdomain.NormalizationProcess, error) {
	result, err := uc.normalizationService.GetProcessStatus(ctx, processID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process status: %w", err)
	}
	return result, nil
}

// StopProcess останавливает процесс нормализации
func (uc *UseCase) StopProcess(ctx context.Context, processID string) error {
	if err := uc.normalizationService.StopProcess(ctx, processID); err != nil {
		return fmt.Errorf("failed to stop normalization process: %w", err)
	}
	return nil
}

// GetActiveProcesses возвращает активные процессы нормализации
func (uc *UseCase) GetActiveProcesses(ctx context.Context) ([]*normalizationdomain.NormalizationProcess, error) {
	result, err := uc.normalizationService.GetActiveProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active processes: %w", err)
	}
	return result, nil
}

// NormalizeName нормализует название
func (uc *UseCase) NormalizeName(ctx context.Context, name string, entityType string) (string, error) {
	result, err := uc.normalizationService.NormalizeName(ctx, name, entityType)
	if err != nil {
		return "", fmt.Errorf("failed to normalize name: %w", err)
	}
	return result, nil
}

// GetStatistics возвращает статистику нормализации
func (uc *UseCase) GetStatistics(ctx context.Context) (*normalizationdomain.NormalizationStatistics, error) {
	result, err := uc.normalizationService.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	return result, nil
}

// GetProcessHistory возвращает историю процессов нормализации
func (uc *UseCase) GetProcessHistory(ctx context.Context, uploadID string) ([]*normalizationdomain.NormalizationProcess, error) {
	result, err := uc.normalizationService.GetProcessHistory(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process history: %w", err)
	}
	return result, nil
}

