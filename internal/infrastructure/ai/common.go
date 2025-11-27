package ai

import "time"

// RetryConfig конфигурация повторных попыток
type RetryConfig struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// DefaultRetryConfig возвращает конфигурацию повторных попыток по умолчанию
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:        3,
		InitialDelay:      500 * time.Millisecond,
		MaxDelay:          10 * time.Second,
		BackoffMultiplier: 2.0,
	}
}


