package services

import (
	"sync"
	"time"

	apperrors "httpserver/server/errors"
)

// ReclassificationStatus статус процесса переклассификации
type ReclassificationStatus struct {
	IsRunning   bool     `json:"isRunning"`
	Progress    float64  `json:"progress"`
	Processed   int      `json:"processed"`
	Total       int      `json:"total"`
	Success     int      `json:"success"`
	Errors      int      `json:"errors"`
	Skipped     int      `json:"skipped"`
	CurrentStep string   `json:"currentStep"`
	Logs        []string `json:"logs"`
	StartTime   string   `json:"startTime,omitempty"`
	ElapsedTime string   `json:"elapsedTime,omitempty"`
	Rate        float64  `json:"rate"` // записей в секунду
}

// ReclassificationRequest запрос на запуск переклассификации
type ReclassificationRequest struct {
	ClassifierID int    `json:"classifier_id"`
	StrategyID   string `json:"strategy_id"`
	Limit        int    `json:"limit,omitempty"` // 0 = без лимита
}

// ReclassificationService сервис для переклассификации
type ReclassificationService struct {
	running     bool
	runningMu   sync.RWMutex
	status      ReclassificationStatus
	statusMu    sync.RWMutex
	events      chan string
	stopChan    chan bool
}

// NewReclassificationService создает новый сервис для переклассификации
func NewReclassificationService() *ReclassificationService {
	return &ReclassificationService{
		running:  false,
		status: ReclassificationStatus{
			IsRunning: false,
			Logs:      make([]string, 0),
		},
		events:   make(chan string, 1000),
		stopChan: make(chan bool, 1),
	}
}

// Start запускает процесс переклассификации
func (s *ReclassificationService) Start(req ReclassificationRequest) error {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()

	if s.running {
		return apperrors.NewConflictError("переклассификация уже запущена", nil)
	}

	s.running = true

	// Валидация
	if req.ClassifierID <= 0 {
		req.ClassifierID = 1 // По умолчанию КПВЭД
	}
	if req.StrategyID == "" {
		req.StrategyID = "top_priority"
	}

	// Инициализация статуса
	s.statusMu.Lock()
	s.status = ReclassificationStatus{
		IsRunning:   true,
		Processed:   0,
		Total:       0,
		Success:     0,
		Errors:      0,
		Skipped:     0,
		CurrentStep: "Инициализация...",
		Logs:        make([]string, 0),
		StartTime:   time.Now().Format(time.RFC3339),
	}
	s.statusMu.Unlock()

	// TODO: Запустить переклассификацию в отдельной горутине
	// Это требует доступа к db, normalizedDB и другим зависимостям из Server

	return nil
}

// Stop останавливает процесс переклассификации
func (s *ReclassificationService) Stop() bool {
	s.runningMu.Lock()
	defer s.runningMu.Unlock()

	wasRunning := s.running
	s.running = false

	if wasRunning {
		select {
		case s.stopChan <- true:
		default:
		}
		s.sendEvent("⚠ Процесс переклассификации остановлен пользователем")
	}

	return wasRunning
}

// GetStatus получает текущий статус переклассификации
func (s *ReclassificationService) GetStatus() ReclassificationStatus {
	s.statusMu.RLock()
	defer s.statusMu.RUnlock()
	return s.status
}

// IsRunning проверяет, запущена ли переклассификация
func (s *ReclassificationService) IsRunning() bool {
	s.runningMu.RLock()
	defer s.runningMu.RUnlock()
	return s.running
}

// GetEvents возвращает канал событий
func (s *ReclassificationService) GetEvents() <-chan string {
	return s.events
}

// sendEvent отправляет событие
func (s *ReclassificationService) sendEvent(message string) {
	select {
	case s.events <- message:
	default:
		// Канал полон, пропускаем событие
	}
}

