package normalization

import (
	"context"
	"fmt"
	"time"

	"httpserver/internal/domain/repositories"

	"github.com/google/uuid"
)

// service реализация domain service для normalization
type service struct {
	normalizationRepo repositories.NormalizationRepository
	normalizer        NormalizerInterface
	benchmarkService  BenchmarkServiceInterface
}

// NewService создает новый domain service для normalization
func NewService(
	normalizationRepo repositories.NormalizationRepository,
	normalizer NormalizerInterface,
	benchmarkService BenchmarkServiceInterface,
) Service {
	return &service{
		normalizationRepo: normalizationRepo,
		normalizer:        normalizer,
		benchmarkService:  benchmarkService,
	}
}

// StartProcess запускает процесс нормализации
func (s *service) StartProcess(ctx context.Context, uploadID string) (*NormalizationProcess, error) {
	if uploadID == "" {
		return nil, ErrInvalidUploadID
	}

	processID := uuid.New().String()

	// Создаем процесс нормализации
	process := &repositories.NormalizationProcess{
		ID:        processID,
		UploadID:  uploadID,
		Status:    "pending",
		Progress:  0.0,
		Processed: 0,
		Total:     0,
		StartedAt: time.Now(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Сохраняем в репозитории
	if err := s.normalizationRepo.Create(ctx, process); err != nil {
		return nil, fmt.Errorf("failed to create normalization process: %w", err)
	}

	// Преобразуем uploadID в int для нормализатора
	var uploadIDInt int
	if _, err := fmt.Sscanf(uploadID, "%d", &uploadIDInt); err != nil {
		uploadIDInt = 0 // Если не удалось преобразовать, используем 0
	}

	// Запускаем нормализацию через адаптер
	if s.normalizer != nil {
		if err := s.normalizer.ProcessNormalization(ctx, uploadIDInt); err != nil {
			// Обновляем статус на failed
			process.Status = "failed"
			process.Error = err.Error()
			now := time.Now()
			process.CompletedAt = &now
			process.UpdatedAt = now
			s.normalizationRepo.Update(ctx, process)
			return nil, fmt.Errorf("failed to start normalization: %w", err)
		}
	}

	// Обновляем статус на running
	process.Status = "running"
	process.UpdatedAt = time.Now()
	if err := s.normalizationRepo.Update(ctx, process); err != nil {
		return nil, fmt.Errorf("failed to update normalization process: %w", err)
	}

	return s.toDomainProcess(process), nil
}

// GetProcessStatus возвращает статус процесса нормализации
func (s *service) GetProcessStatus(ctx context.Context, processID string) (*NormalizationProcess, error) {
	process, err := s.normalizationRepo.GetByID(ctx, processID)
	if err != nil {
		return nil, fmt.Errorf("failed to get normalization process: %w", err)
	}

	return s.toDomainProcess(process), nil
}

// StopProcess останавливает процесс нормализации
func (s *service) StopProcess(ctx context.Context, processID string) error {
	process, err := s.normalizationRepo.GetByID(ctx, processID)
	if err != nil {
		return fmt.Errorf("failed to get normalization process: %w", err)
	}

	if process.Status != "running" && process.Status != "pending" {
		return ErrProcessNotRunning
	}

	// Обновляем статус
	now := time.Now()
	process.Status = "cancelled"
	process.CompletedAt = &now
	process.UpdatedAt = now

	if err := s.normalizationRepo.Update(ctx, process); err != nil {
		return fmt.Errorf("failed to stop normalization process: %w", err)
	}

	return nil
}

// GetActiveProcesses возвращает активные процессы нормализации
func (s *service) GetActiveProcesses(ctx context.Context) ([]*NormalizationProcess, error) {
	processes, err := s.normalizationRepo.GetActiveProcesses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active processes: %w", err)
	}

	result := make([]*NormalizationProcess, 0, len(processes))
	for _, p := range processes {
		result = append(result, s.toDomainProcess(&p))
	}

	return result, nil
}

// NormalizeName нормализует название
func (s *service) NormalizeName(ctx context.Context, name string, entityType string) (string, error) {
	// Сначала ищем в эталонах
	if s.benchmarkService != nil {
		benchmark, err := s.benchmarkService.FindBestMatch(ctx, name, entityType)
		if err == nil && benchmark != nil {
			// Найден эталон, возвращаем каноническое имя
			return benchmark.Name, nil
		}
		// Если эталон не найден, продолжаем с AI нормализацией
	}

	// Используем AI нормализатор
	if s.normalizer != nil {
		return s.normalizer.NormalizeName(ctx, name, entityType)
	}

	return "", fmt.Errorf("normalization service is not available")
}

// NormalizeEntity нормализует сущность
func (s *service) NormalizeEntity(ctx context.Context, entityID string, entityType string) (*NormalizedEntity, error) {
	// TODO: Реализовать нормализацию сущности
	return nil, fmt.Errorf("not implemented yet")
}

// BatchNormalize выполняет пакетную нормализацию
func (s *service) BatchNormalize(ctx context.Context, entityIDs []string, entityType string) (*BatchNormalizationResult, error) {
	// TODO: Реализовать пакетную нормализацию
	return nil, fmt.Errorf("not implemented yet")
}

// StartVersionedNormalization начинает версионированную нормализацию
func (s *service) StartVersionedNormalization(ctx context.Context, itemID int, originalName string) (*NormalizationSession, error) {
	// TODO: Реализовать через normalization pipeline
	return nil, fmt.Errorf("not implemented yet")
}

// ApplyPatterns применяет алгоритмические паттерны
func (s *service) ApplyPatterns(ctx context.Context, sessionID int) (*NormalizationSession, error) {
	// TODO: Реализовать применение паттернов
	return nil, fmt.Errorf("not implemented yet")
}

// ApplyAI применяет AI коррекцию
func (s *service) ApplyAI(ctx context.Context, sessionID int, useChat bool) (*NormalizationSession, error) {
	// TODO: Реализовать AI коррекцию
	return nil, fmt.Errorf("not implemented yet")
}

// GetSessionHistory возвращает историю сессии нормализации
func (s *service) GetSessionHistory(ctx context.Context, sessionID int) ([]*NormalizationStage, error) {
	// TODO: Реализовать получение истории сессии
	return nil, fmt.Errorf("not implemented yet")
}

// GetStatistics возвращает статистику нормализации
func (s *service) GetStatistics(ctx context.Context) (*NormalizationStatistics, error) {
	stats, err := s.normalizationRepo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	if stats == nil {
		return &NormalizationStatistics{}, nil
	}

	return &NormalizationStatistics{
		TotalProcesses:      stats.TotalProcesses,
		SuccessfulProcesses: stats.SuccessfulProcesses,
		FailedProcesses:     stats.FailedProcesses,
		AverageDuration:     stats.AverageDuration,
		TotalProcessed:      stats.TotalProcessed,
		AverageProgress:     stats.AverageProgress,
	}, nil
}

// GetProcessHistory возвращает историю процессов нормализации
func (s *service) GetProcessHistory(ctx context.Context, uploadID string) ([]*NormalizationProcess, error) {
	processes, err := s.normalizationRepo.GetProcessHistory(ctx, uploadID)
	if err != nil {
		return nil, fmt.Errorf("failed to get process history: %w", err)
	}

	result := make([]*NormalizationProcess, 0, len(processes))
	for _, p := range processes {
		result = append(result, s.toDomainProcess(&p))
	}

	return result, nil
}

// toDomainProcess преобразует repositories.NormalizationProcess в domain NormalizationProcess
func (s *service) toDomainProcess(p *repositories.NormalizationProcess) *NormalizationProcess {
	return &NormalizationProcess{
		ID:          p.ID,
		UploadID:    p.UploadID,
		Status:      p.Status,
		Progress:    p.Progress,
		Processed:   p.Processed,
		Total:       p.Total,
		StartedAt:   p.StartedAt,
		CompletedAt: p.CompletedAt,
		Error:       p.Error,
		Config:      p.Config,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

