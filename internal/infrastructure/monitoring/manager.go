package monitoring

import (
	"sync"
	"time"
)

// ProviderMetrics метрики для одного провайдера
type ProviderMetrics struct {
	ID                 string    `json:"id"`                  // "openrouter", "huggingface", "arliai"
	Name               string    `json:"name"`                // "OpenRouter", "Hugging Face", "Arliai"
	ActiveChannels     int       `json:"active_channels"`     // Количество активных каналов
	CurrentRequests    int       `json:"current_requests"`    // Текущие активные запросы
	TotalRequests      int64     `json:"total_requests"`      // Всего запросов с момента запуска
	SuccessfulRequests int64     `json:"successful_requests"` // Успешных запросов
	FailedRequests     int64     `json:"failed_requests"`     // Неудачных запросов
	AverageLatencyMs   float64   `json:"average_latency_ms"`  // Средняя задержка в миллисекундах
	LastRequestTime    time.Time `json:"last_request_time"`   // Время последнего запроса
	Status             string    `json:"status"`              // "active", "idle", "error"
	RequestsPerSecond  float64   `json:"requests_per_second"` // Запросов в секунду (скользящее среднее)
}

// SystemStats общая статистика системы
type SystemStats struct {
	TotalProviders          int       `json:"total_providers"`
	ActiveProviders         int       `json:"active_providers"`
	TotalRequests           int64     `json:"total_requests"`
	TotalSuccessful         int64     `json:"total_successful"`
	TotalFailed             int64     `json:"total_failed"`
	SystemRequestsPerSecond float64   `json:"system_requests_per_second"`
	Timestamp               time.Time `json:"timestamp"`
}

// MonitoringData данные для отправки клиенту
type MonitoringData struct {
	Providers []ProviderMetrics `json:"providers"`
	System    SystemStats       `json:"system"`
}

// Manager потокобезопасный менеджер для сбора статистики
type Manager struct {
	metrics        map[string]*ProviderMetrics
	mu             sync.RWMutex
	requestHistory map[string][]time.Time // История запросов для расчета RPS (последние 60 секунд)
	historyMu      sync.RWMutex
}

// NewManager создает новый менеджер мониторинга
func NewManager() *Manager {
	mm := &Manager{
		metrics:        make(map[string]*ProviderMetrics),
		requestHistory: make(map[string][]time.Time),
	}

	return mm
}

// RegisterProvider регистрирует провайдера в системе мониторинга
func (mm *Manager) RegisterProvider(providerID, name string, channels int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.metrics[providerID] = &ProviderMetrics{
		ID:             providerID,
		Name:           name,
		ActiveChannels: channels,
		Status:         "idle",
	}
}

// StartHistoryProcessor запускает фоновый процессор для очистки истории запросов
func (mm *Manager) StartHistoryProcessor() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			mm.historyMu.Lock()
			cutoff := time.Now().Add(-60 * time.Second)
			for providerID, history := range mm.requestHistory {
				filtered := []time.Time{}
				for _, t := range history {
					if t.After(cutoff) {
						filtered = append(filtered, t)
					}
				}
				mm.requestHistory[providerID] = filtered
			}
			mm.historyMu.Unlock()
		}
	}()
}

// SetProviderChannels устанавливает количество каналов для провайдера
func (mm *Manager) SetProviderChannels(providerID string, channels int) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	metric, exists := mm.metrics[providerID]
	if !exists {
		return
	}

	metric.ActiveChannels = channels
}

// IncrementRequest вызывается перед отправкой запроса
func (mm *Manager) IncrementRequest(providerID string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	metric, exists := mm.metrics[providerID]
	if !exists {
		// Создаем новую метрику, если провайдер не найден
		metric = &ProviderMetrics{
			ID:             providerID,
			Name:           providerID,
			ActiveChannels: 1,
			Status:         "active",
		}
		mm.metrics[providerID] = metric
	}

	metric.CurrentRequests++
	metric.TotalRequests++
	metric.LastRequestTime = time.Now()
	metric.Status = "active"

	// Добавляем в историю для расчета RPS
	mm.historyMu.Lock()
	mm.requestHistory[providerID] = append(mm.requestHistory[providerID], time.Now())
	// Оставляем только последние 60 секунд
	cutoff := time.Now().Add(-60 * time.Second)
	filtered := []time.Time{}
	for _, t := range mm.requestHistory[providerID] {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	mm.requestHistory[providerID] = filtered
	mm.historyMu.Unlock()
}

// RecordResponse вызывается после получения ответа
func (mm *Manager) RecordResponse(providerID string, latencyMs float64, err error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	metric, exists := mm.metrics[providerID]
	if !exists {
		return
	}

	metric.CurrentRequests--
	if metric.CurrentRequests < 0 {
		metric.CurrentRequests = 0
	}

	if err != nil {
		metric.FailedRequests++
		// Если много ошибок подряд, устанавливаем статус "error"
		if metric.FailedRequests > 0 && metric.TotalRequests > 10 {
			errorRate := float64(metric.FailedRequests) / float64(metric.TotalRequests)
			if errorRate > 0.5 {
				metric.Status = "error"
			}
		}
	} else {
		metric.SuccessfulRequests++
		// Если были ошибки, но последний запрос успешен, возвращаем статус "active"
		if metric.Status == "error" && metric.CurrentRequests == 0 {
			metric.Status = "idle"
		}
	}

	// Обновление средней задержки (скользящее среднее)
	if metric.TotalRequests > 0 {
		metric.AverageLatencyMs = (metric.AverageLatencyMs * 0.9) + (latencyMs * 0.1)
	} else {
		metric.AverageLatencyMs = latencyMs
	}

	// Обновление статуса
	if metric.CurrentRequests == 0 {
		if time.Since(metric.LastRequestTime) > 30*time.Second {
			metric.Status = "idle"
		}
	}

	// Расчет RPS
	mm.historyMu.RLock()
	history := mm.requestHistory[providerID]
	mm.historyMu.RUnlock()

	if len(history) > 0 {
		timeSpan := time.Since(history[0]).Seconds()
		if timeSpan > 0 {
			metric.RequestsPerSecond = float64(len(history)) / timeSpan
		}
	}
}

// GetAllMetrics возвращает все метрики для отправки клиенту
func (mm *Manager) GetAllMetrics() MonitoringData {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	providers := make([]ProviderMetrics, 0, len(mm.metrics))
	totalRequests := int64(0)
	totalSuccessful := int64(0)
	totalFailed := int64(0)
	activeProviders := 0
	systemRPS := 0.0

	for _, metric := range mm.metrics {
		providers = append(providers, *metric)
		totalRequests += metric.TotalRequests
		totalSuccessful += metric.SuccessfulRequests
		totalFailed += metric.FailedRequests
		systemRPS += metric.RequestsPerSecond
		if metric.Status == "active" {
			activeProviders++
		}
	}

	return MonitoringData{
		Providers: providers,
		System: SystemStats{
			TotalProviders:          len(mm.metrics),
			ActiveProviders:         activeProviders,
			TotalRequests:           totalRequests,
			TotalSuccessful:         totalSuccessful,
			TotalFailed:             totalFailed,
			SystemRequestsPerSecond: systemRPS,
			Timestamp:               time.Now(),
		},
	}
}
