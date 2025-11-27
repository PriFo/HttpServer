package errors

import (
	"sync"
	"time"
)

// ErrorMetricsCollector собирает метрики ошибок для мониторинга
type ErrorMetricsCollector struct {
	mu sync.RWMutex

	// Общие метрики
	totalErrors       int64
	errorsByType      map[string]int64      // По типу ошибки (ValidationError, InternalError и т.д.)
	errorsByCode      map[int]int64         // По HTTP статус коду
	errorsByEndpoint  map[string]int64      // По эндпоинту
	errorsByTime      []ErrorTimeBucket     // По времени (последний час)

	// Детальные метрики
	lastErrors        []ErrorRecord         // Последние N ошибок
	maxLastErrors     int                   // Максимальное количество сохраняемых ошибок

	// Временные метрики
	startTime         time.Time
}

// ErrorTimeBucket метрики за временной интервал
type ErrorTimeBucket struct {
	Time      time.Time
	Count     int64
	ByType    map[string]int64
	ByCode    map[int]int64
}

// ErrorRecord запись об ошибке
type ErrorRecord struct {
	Timestamp   time.Time
	Type        string
	Code        int
	Message     string
	Endpoint    string
	RequestID   string
	UserMessage string
}

// NewErrorMetricsCollector создает новый сборщик метрик ошибок
func NewErrorMetricsCollector() *ErrorMetricsCollector {
	return &ErrorMetricsCollector{
		errorsByType:     make(map[string]int64),
		errorsByCode:     make(map[int]int64),
		errorsByEndpoint: make(map[string]int64),
		errorsByTime:     make([]ErrorTimeBucket, 0),
		lastErrors:       make([]ErrorRecord, 0),
		maxLastErrors:    100,
		startTime:        time.Now(),
	}
}

// RecordError записывает ошибку в метрики
func (emc *ErrorMetricsCollector) RecordError(err *AppError, endpoint, requestID string) {
	emc.mu.Lock()
	defer emc.mu.Unlock()

	emc.totalErrors++

	// Определяем тип ошибки
	errorType := emc.getErrorType(err)
	emc.errorsByType[errorType]++

	// Записываем по коду
	emc.errorsByCode[err.Code]++

	// Записываем по эндпоинту
	if endpoint != "" {
		emc.errorsByEndpoint[endpoint]++
	}

	// Добавляем во временные метрики
	emc.addToTimeBucket(errorType, err.Code)

	// Добавляем в последние ошибки
	errorRecord := ErrorRecord{
		Timestamp:   time.Now(),
		Type:        errorType,
		Code:        err.Code,
		Message:     err.Error(),
		Endpoint:    endpoint,
		RequestID:   requestID,
		UserMessage: err.UserMessage(),
	}
	emc.lastErrors = append([]ErrorRecord{errorRecord}, emc.lastErrors...)
	if len(emc.lastErrors) > emc.maxLastErrors {
		emc.lastErrors = emc.lastErrors[:emc.maxLastErrors]
	}
}

// getErrorType определяет тип ошибки по коду
func (emc *ErrorMetricsCollector) getErrorType(err *AppError) string {
	switch err.Code {
	case 400:
		return "ValidationError"
	case 401:
		return "UnauthorizedError"
	case 403:
		return "ForbiddenError"
	case 404:
		return "NotFoundError"
	case 409:
		return "ConflictError"
	case 500:
		return "InternalError"
	case 502:
		return "BadGatewayError"
	case 503:
		return "ServiceUnavailableError"
	default:
		return "UnknownError"
	}
}

// addToTimeBucket добавляет ошибку во временной интервал
func (emc *ErrorMetricsCollector) addToTimeBucket(errorType string, code int) {
	now := time.Now()
	currentMinute := now.Truncate(time.Minute)

	// Ищем существующий интервал
	found := false
	for i := range emc.errorsByTime {
		if emc.errorsByTime[i].Time.Equal(currentMinute) {
			emc.errorsByTime[i].Count++
			emc.errorsByTime[i].ByType[errorType]++
			emc.errorsByTime[i].ByCode[code]++
			found = true
			break
		}
	}

	// Создаем новый интервал, если не найден
	if !found {
		emc.errorsByTime = append([]ErrorTimeBucket{{
			Time:   currentMinute,
			Count:  1,
			ByType: map[string]int64{errorType: 1},
			ByCode: map[int]int64{code: 1},
		}}, emc.errorsByTime...)
	}

	// Оставляем только последний час
	oneHourAgo := now.Add(-1 * time.Hour)
	filtered := make([]ErrorTimeBucket, 0)
	for _, bucket := range emc.errorsByTime {
		if bucket.Time.After(oneHourAgo) {
			filtered = append(filtered, bucket)
		}
	}
	emc.errorsByTime = filtered
}

// GetMetrics возвращает все метрики ошибок
func (emc *ErrorMetricsCollector) GetMetrics() map[string]interface{} {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	// Копируем данные для безопасного возврата
	errorsByType := make(map[string]int64)
	for k, v := range emc.errorsByType {
		errorsByType[k] = v
	}

	errorsByCode := make(map[int]int64)
	for k, v := range emc.errorsByCode {
		errorsByCode[k] = v
	}

	errorsByEndpoint := make(map[string]int64)
	for k, v := range emc.errorsByEndpoint {
		errorsByEndpoint[k] = v
	}

	// Копируем временные метрики
	timeBuckets := make([]ErrorTimeBucket, len(emc.errorsByTime))
	copy(timeBuckets, emc.errorsByTime)

	// Копируем последние ошибки
	lastErrors := make([]ErrorRecord, len(emc.lastErrors))
	copy(lastErrors, emc.lastErrors)

	return map[string]interface{}{
		"total_errors":      emc.totalErrors,
		"errors_by_type":    errorsByType,
		"errors_by_code":    errorsByCode,
		"errors_by_endpoint": errorsByEndpoint,
		"time_buckets":      timeBuckets,
		"last_errors":       lastErrors,
		"uptime_seconds":    time.Since(emc.startTime).Seconds(),
		"errors_per_minute": emc.calculateErrorsPerMinute(),
	}
}

// calculateErrorsPerMinute рассчитывает среднее количество ошибок в минуту
func (emc *ErrorMetricsCollector) calculateErrorsPerMinute() float64 {
	if len(emc.errorsByTime) == 0 {
		return 0
	}

	totalCount := int64(0)
	for _, bucket := range emc.errorsByTime {
		totalCount += bucket.Count
	}

	minutes := float64(len(emc.errorsByTime))
	if minutes == 0 {
		return 0
	}

	return float64(totalCount) / minutes
}

// GetErrorsByType возвращает количество ошибок по типу
func (emc *ErrorMetricsCollector) GetErrorsByType() map[string]int64 {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range emc.errorsByType {
		result[k] = v
	}
	return result
}

// GetErrorsByCode возвращает количество ошибок по HTTP коду
func (emc *ErrorMetricsCollector) GetErrorsByCode() map[int]int64 {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	result := make(map[int]int64)
	for k, v := range emc.errorsByCode {
		result[k] = v
	}
	return result
}

// GetErrorsByEndpoint возвращает количество ошибок по эндпоинту
func (emc *ErrorMetricsCollector) GetErrorsByEndpoint() map[string]int64 {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	result := make(map[string]int64)
	for k, v := range emc.errorsByEndpoint {
		result[k] = v
	}
	return result
}

// GetLastErrors возвращает последние N ошибок
func (emc *ErrorMetricsCollector) GetLastErrors(limit int) []ErrorRecord {
	emc.mu.RLock()
	defer emc.mu.RUnlock()

	if limit <= 0 || limit > len(emc.lastErrors) {
		limit = len(emc.lastErrors)
	}

	result := make([]ErrorRecord, limit)
	copy(result, emc.lastErrors[:limit])
	return result
}

// Reset сбрасывает все метрики
func (emc *ErrorMetricsCollector) Reset() {
	emc.mu.Lock()
	defer emc.mu.Unlock()

	emc.totalErrors = 0
	emc.errorsByType = make(map[string]int64)
	emc.errorsByCode = make(map[int]int64)
	emc.errorsByEndpoint = make(map[string]int64)
	emc.errorsByTime = make([]ErrorTimeBucket, 0)
	emc.lastErrors = make([]ErrorRecord, 0)
	emc.startTime = time.Now()
}

