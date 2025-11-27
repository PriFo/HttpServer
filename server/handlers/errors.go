package handlers

import (
	"httpserver/server/errors"
)

// Реэкспорт типов и функций из server/errors для обратной совместимости
type AppError = errors.AppError

var (
	NewValidationError = errors.NewValidationError
	NewInternalError   = errors.NewInternalError
	NewNotFoundError   = errors.NewNotFoundError
	NewConflictError   = errors.NewConflictError
	NewUnauthorizedError = errors.NewUnauthorizedError
	NewForbiddenError   = errors.NewForbiddenError
	NewBadGatewayError  = errors.NewBadGatewayError
	NewServiceUnavailableError = errors.NewServiceUnavailableError
	WrapError           = errors.WrapError
)

