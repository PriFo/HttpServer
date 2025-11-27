package websearch

import (
	"time"
)

// ProviderStats статистика провайдера
type ProviderStats struct {
	ProviderName      string
	RequestsTotal     int64
	RequestsSuccess   int64
	RequestsFailed    int64
	FailureRate       float64
	AvgResponseTimeMs int64
	LastSuccess       *time.Time
	LastFailure       *time.Time
	LastError         string
	UpdatedAt         time.Time
}

// ReliabilityManagerInterface интерфейс для управления надежностью провайдеров
type ReliabilityManagerInterface interface {
	// RecordSuccess записывает успешный запрос
	RecordSuccess(providerName string, responseTime time.Duration) error
	
	// RecordSuccessWithTime записывает успешный запрос с временем ответа (алиас для RecordSuccess)
	RecordSuccessWithTime(providerName string, responseTime time.Duration) error
	
	// RecordFailure записывает неуспешный запрос без ошибки
	RecordFailure(providerName string) error
	
	// RecordFailureWithError записывает неуспешный запрос с ошибкой
	RecordFailureWithError(providerName string, err error) error
	
	// GetStats возвращает статистику провайдера
	GetStats(providerName string) *ProviderStats
	
	// GetAllStats возвращает статистику всех провайдеров
	GetAllStats() map[string]*ProviderStats
	
	// GetWeight возвращает вес провайдера для выбора
	GetWeight(providerName string, basePriority int) float64
}

