package server

// Re-export для обратной совместимости
import apperrors "httpserver/server/errors"

// AppError представляет ошибку приложения с HTTP статусом и контекстом
// Deprecated: используйте errors.AppError напрямую
type AppError = apperrors.AppError

// NewNotFoundError создает ошибку 404 Not Found
// Deprecated: используйте errors.NewNotFoundError
var NewNotFoundError = apperrors.NewNotFoundError

// NewValidationError создает ошибку 400 Bad Request
// Deprecated: используйте errors.NewValidationError
var NewValidationError = apperrors.NewValidationError

// NewInternalError создает ошибку 500 Internal Server Error
// Deprecated: используйте errors.NewInternalError
var NewInternalError = apperrors.NewInternalError

// NewConflictError создает ошибку 409 Conflict
// Deprecated: используйте errors.NewConflictError
var NewConflictError = apperrors.NewConflictError

// NewUnauthorizedError создает ошибку 401 Unauthorized
// Deprecated: используйте errors.NewUnauthorizedError
var NewUnauthorizedError = apperrors.NewUnauthorizedError

// NewForbiddenError создает ошибку 403 Forbidden
// Deprecated: используйте errors.NewForbiddenError
var NewForbiddenError = apperrors.NewForbiddenError

// NewBadGatewayError создает ошибку 502 Bad Gateway
// Deprecated: используйте errors.NewBadGatewayError
var NewBadGatewayError = apperrors.NewBadGatewayError

// NewServiceUnavailableError создает ошибку 503 Service Unavailable
// Deprecated: используйте errors.NewServiceUnavailableError
var NewServiceUnavailableError = apperrors.NewServiceUnavailableError

// WrapError оборачивает существующую ошибку с контекстом
// Deprecated: используйте errors.WrapError
var WrapError = apperrors.WrapError

