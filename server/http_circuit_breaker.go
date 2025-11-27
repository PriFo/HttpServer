package server

import (
	"sync"
	"time"
)

// HTTPCircuitBreakerState состояние Circuit Breaker для HTTP-клиентов
type HTTPCircuitBreakerState int

const (
	HTTPStateClosed HTTPCircuitBreakerState = iota // Нормальная работа
	HTTPStateOpen                                  // Breaker открыт - запросы блокируются
	HTTPStateHalfOpen                              // Пробуем восстановить соединение
)

// HTTPCircuitBreaker защита от каскадных сбоев для HTTP-клиентов провайдеров
type HTTPCircuitBreaker struct {
	mu              sync.RWMutex
	state           HTTPCircuitBreakerState
	failureCount    int           // Счетчик неудачных запросов
	successCount    int           // Счетчик успешных запросов в half-open состоянии
	failureThreshold int          // Порог ошибок для открытия breaker (по умолчанию 5)
	successThreshold int          // Порог успехов для закрытия breaker (по умолчанию 2)
	timeout         time.Duration // Время ожидания перед переходом в half-open (по умолчанию 30 секунд)
	lastFailureTime time.Time     // Время последней ошибки
}

// NewHTTPCircuitBreaker создает новый circuit breaker для HTTP-клиентов
func NewHTTPCircuitBreaker() *HTTPCircuitBreaker {
	return &HTTPCircuitBreaker{
		state:            HTTPStateClosed,
		failureThreshold: 5,              // Открываем после 5 ошибок
		successThreshold: 2,              // Закрываем после 2 успехов в half-open
		timeout:          30 * time.Second, // Ждем 30 секунд перед попыткой восстановления
	}
}

// CanProceed проверяет, можно ли выполнить запрос
// Возвращает false если Circuit Breaker открыт (слишком много ошибок)
func (cb *HTTPCircuitBreaker) CanProceed() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case HTTPStateClosed:
		// Нормальная работа - пропускаем запрос
		return true

	case HTTPStateOpen:
		// Проверяем, прошло ли время для попытки восстановления
		if time.Since(cb.lastFailureTime) > cb.timeout {
			// Переходим в half-open для пробного запроса
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = HTTPStateHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		// Breaker все еще открыт - блокируем запрос
		return false

	case HTTPStateHalfOpen:
		// В half-open состоянии пропускаем ограниченное количество запросов
		return true

	default:
		return false
	}
}

// RecordSuccess записывает успешный запрос
func (cb *HTTPCircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case HTTPStateClosed:
		// Сбрасываем счетчик ошибок при успешном запросе
		cb.failureCount = 0

	case HTTPStateHalfOpen:
		// Увеличиваем счетчик успехов в half-open
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			// Достаточно успехов - закрываем breaker
			cb.state = HTTPStateClosed
			cb.failureCount = 0
			cb.successCount = 0
		}
	}
}

// RecordFailure записывает неудачный запрос
func (cb *HTTPCircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case HTTPStateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			// Слишком много ошибок - открываем breaker
			cb.state = HTTPStateOpen
		}

	case HTTPStateHalfOpen:
		// Ошибка в half-open - возвращаемся в open
		cb.state = HTTPStateOpen
		cb.failureCount = cb.failureThreshold // Устанавливаем максимальное значение
		cb.successCount = 0
	}
}

// GetState возвращает текущее состояние Circuit Breaker (для логирования)
func (cb *HTTPCircuitBreaker) GetState() string {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case HTTPStateClosed:
		return "closed"
	case HTTPStateOpen:
		return "open"
	case HTTPStateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// GetStateDetails возвращает детальное состояние Circuit Breaker для мониторинга
func (cb *HTTPCircuitBreaker) GetStateDetails() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	canProceed := false
	switch cb.state {
	case HTTPStateClosed:
		canProceed = true
	case HTTPStateOpen:
		canProceed = time.Since(cb.lastFailureTime) > cb.timeout
	case HTTPStateHalfOpen:
		canProceed = true
	}

	stateStr := "closed"
	switch cb.state {
	case HTTPStateClosed:
		stateStr = "closed"
	case HTTPStateOpen:
		stateStr = "open"
	case HTTPStateHalfOpen:
		stateStr = "half-open"
	}

	result := map[string]interface{}{
		"state":         stateStr,
		"can_proceed":   canProceed,
		"failure_count": cb.failureCount,
		"success_count": cb.successCount,
	}

	if !cb.lastFailureTime.IsZero() {
		result["last_failure_time"] = cb.lastFailureTime.Format(time.RFC3339)
	}

	return result
}

