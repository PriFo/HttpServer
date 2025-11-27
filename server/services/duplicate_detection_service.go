package services

import (
	"fmt"
	"sync"
	"time"

	"httpserver/normalization/algorithms"
	apperrors "httpserver/server/errors"
)

// DuplicateDetectionTask задача обнаружения дублей
type DuplicateDetectionTask struct {
	ID          string
	Status      string // "running", "completed", "failed"
	Progress    int    // 0-100
	TotalItems  int
	Processed   int
	FoundGroups int
	Error       string
	StartedAt   time.Time
	CompletedAt *time.Time
	// mu зарезервировано для будущего использования
	// mu          sync.RWMutex
}

// DuplicateDetectionService сервис для обнаружения дубликатов
type DuplicateDetectionService struct {
	tasks     map[string]*DuplicateDetectionTask
	tasksMu   sync.RWMutex
	taskCounter int
	taskCounterMu sync.Mutex
}

// NewDuplicateDetectionService создает новый сервис для обнаружения дубликатов
func NewDuplicateDetectionService() *DuplicateDetectionService {
	return &DuplicateDetectionService{
		tasks: make(map[string]*DuplicateDetectionTask),
	}
}

// StartDetection запускает обнаружение дубликатов
func (s *DuplicateDetectionService) StartDetection(projectID int, threshold float64, batchSize int, useAdvanced bool, weights *algorithms.SimilarityWeights, maxItems int) (string, error) {
	if projectID <= 0 {
		return "", apperrors.NewValidationError("project_id обязателен", nil)
	}

	if threshold <= 0 || threshold > 1 {
		threshold = 0.75
	}

	if batchSize <= 0 {
		batchSize = 100
	}

	// TODO: weights будет использоваться при реализации обнаружения дубликатов
	if weights == nil {
		_ = algorithms.DefaultSimilarityWeights() // Заглушка для будущего использования
		weights = algorithms.DefaultSimilarityWeights()
	}

	// Генерируем ID задачи
	taskID := s.generateTaskID()
	task := &DuplicateDetectionTask{
		ID:         taskID,
		Status:     "running",
		Progress:   0,
		StartedAt:  time.Now(),
	}

	s.tasksMu.Lock()
	s.tasks[taskID] = task
	s.tasksMu.Unlock()

	// TODO: Запустить обнаружение в отдельной горутине
	// Это требует доступа к normalizedDB и другим зависимостям из Server

	return taskID, nil
}

// GetTaskStatus получает статус задачи
func (s *DuplicateDetectionService) GetTaskStatus(taskID string) (*DuplicateDetectionTask, error) {
	s.tasksMu.RLock()
	defer s.tasksMu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, apperrors.NewNotFoundError("задача не найдена", nil)
	}

	return task, nil
}

// generateTaskID генерирует уникальный ID задачи
func (s *DuplicateDetectionService) generateTaskID() string {
	s.taskCounterMu.Lock()
	s.taskCounter++
	id := fmt.Sprintf("duplicate_%d_%d", time.Now().Unix(), s.taskCounter)
	s.taskCounterMu.Unlock()
	return id
}

