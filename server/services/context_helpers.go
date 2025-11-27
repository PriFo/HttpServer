package services

import (
	"context"

	apperrors "httpserver/server/errors"
)

// ValidateContext проверяет, что context не nil и не отменен.
// Возвращает ошибку, если context невалиден.
// Используется для единообразной валидации контекста во всех сервисах.
func ValidateContext(ctx context.Context) error {
	if ctx == nil {
		return apperrors.NewValidationError("context не может быть nil", nil)
	}

	select {
	case <-ctx.Done():
		return apperrors.NewServiceUnavailableError("контекст отменен", ctx.Err())
	default:
		return nil
	}
}

