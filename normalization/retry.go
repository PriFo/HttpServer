package normalization

import (
	"log"
	"strings"
	"time"
)

const (
	// DefaultRetryAttempts количество попыток повтора по умолчанию
	DefaultRetryAttempts = 3
	// DefaultRetryDelay задержка между попытками по умолчанию
	DefaultRetryDelay = 100 * time.Millisecond
	// MaxRetryDelay максимальная задержка между попытками
	MaxRetryDelay = 2 * time.Second
)

// RetryConfig конфигурация для retry логики
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64 // Множитель для экспоненциальной задержки
}

// DefaultRetryConfig возвращает конфигурацию retry по умолчанию
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  DefaultRetryAttempts,
		InitialDelay: DefaultRetryDelay,
		MaxDelay:     MaxRetryDelay,
		Multiplier:   2.0,
	}
}

// RetryableFunc функция, которую можно повторить при ошибке
type RetryableFunc func() error

// IsRetryableError проверяет, можно ли повторить операцию при данной ошибке
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	// Список ошибок, при которых стоит повторить операцию
	retryableErrors := []string{
		"database is locked",
		"busy",
		"timeout",
		"connection",
		"temporary",
		"network",
		"deadline exceeded",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// contains проверяет, содержит ли строка подстроку (case-insensitive)
func contains(s, substr string) bool {
	sLower := strings.ToLower(s)
	substrLower := strings.ToLower(substr)
	return strings.Contains(sLower, substrLower)
}

// Retry выполняет функцию с retry логикой
func Retry(fn RetryableFunc, config RetryConfig) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Проверяем, стоит ли повторить операцию
		if !IsRetryableError(err) {
			return err
		}

		// Если это не последняя попытка, ждем перед повтором
		if attempt < config.MaxAttempts {
			time.Sleep(delay)
			// Увеличиваем задержку экспоненциально
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		}
	}

	return lastErr
}

// RetryWithLog выполняет функцию с retry логикой и логированием
func RetryWithLog(fn RetryableFunc, config RetryConfig, operationName string, clientID, projectID int) error {
	var lastErr error
	delay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		err := fn()
		if err == nil {
			if attempt > 1 {
				log.Printf("[Counterparty] [ClientID:%d] [ProjectID:%d] INFO: %s succeeded after %d attempts",
					clientID, projectID, operationName, attempt)
			}
			return nil
		}

		lastErr = err

		// Проверяем, стоит ли повторить операцию
		if !IsRetryableError(err) {
			log.Printf("[Counterparty] [ClientID:%d] [ProjectID:%d] ERROR: %s failed with non-retryable error: %v",
				clientID, projectID, operationName, err)
			return err
		}

		// Если это не последняя попытка, ждем перед повтором
		if attempt < config.MaxAttempts {
			log.Printf("[Counterparty] [ClientID:%d] [ProjectID:%d] WARN: %s failed (attempt %d/%d), retrying in %v: %v",
				clientID, projectID, operationName, attempt, config.MaxAttempts, delay, err)
			time.Sleep(delay)
			// Увеличиваем задержку экспоненциально
			delay = time.Duration(float64(delay) * config.Multiplier)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}
		} else {
			log.Printf("[Counterparty] [ClientID:%d] [ProjectID:%d] ERROR: %s failed after %d attempts: %v",
				clientID, projectID, operationName, config.MaxAttempts, err)
		}
	}

	return lastErr
}
