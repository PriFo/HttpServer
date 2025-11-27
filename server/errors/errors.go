package errors

import (
	"errors"
	"fmt"
	"net/http"
)

// AppError представляет ошибку приложения с HTTP статусом и контекстом
type AppError struct {
	Code    int    `json:"status_code"` // HTTP статус код
	Message string `json:"message"`     // Сообщение для пользователя
	Err     error  `json:"-"`           // Внутренняя ошибка для логов, не сериализуется
	Context string `json:"-"`           // Дополнительный контекст (функция, параметры)
}

// Error реализует интерфейс error
func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap возвращает вложенную ошибку для errors.Is и errors.As
func (e *AppError) Unwrap() error {
	return e.Err
}

// StatusCode возвращает HTTP статус код ошибки
// Реализует интерфейс middleware.HTTPError
func (e *AppError) StatusCode() int {
	return e.Code
}

// UserMessage возвращает сообщение для пользователя
// Реализует интерфейс middleware.HTTPError
func (e *AppError) UserMessage() string {
	return e.Message
}

// GetContext возвращает контекст ошибки
// Реализует интерфейс middleware.HTTPError
func (e *AppError) GetContext() string {
	return e.Context
}

// WithContext добавляет контекст к ошибке
func (e *AppError) WithContext(context string) *AppError {
	e.Context = context
	return e
}

// NewNotFoundError создает ошибку 404 Not Found
func NewNotFoundError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusNotFound,
		Message: message,
		Err:     err,
	}
}

// NewValidationError создает ошибку 400 Bad Request
func NewValidationError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusBadRequest,
		Message: message,
		Err:     err,
	}
}

// NewInternalError создает ошибку 500 Internal Server Error
// Для пользователя возвращается общее сообщение, детали только в логах
func NewInternalError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusInternalServerError,
		Message: "Внутренняя ошибка сервера", // Общее сообщение для пользователя
		Err:     errors.Join(errors.New(message), err), // Детали для лога
	}
}

// NewConflictError создает ошибку 409 Conflict
func NewConflictError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusConflict,
		Message: message,
		Err:     err,
	}
}

// NewUnauthorizedError создает ошибку 401 Unauthorized
func NewUnauthorizedError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusUnauthorized,
		Message: message,
		Err:     err,
	}
}

// NewForbiddenError создает ошибку 403 Forbidden
func NewForbiddenError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusForbidden,
		Message: message,
		Err:     err,
	}
}

// NewBadGatewayError создает ошибку 502 Bad Gateway
func NewBadGatewayError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusBadGateway,
		Message: message,
		Err:     err,
	}
}

// NewServiceUnavailableError создает ошибку 503 Service Unavailable
func NewServiceUnavailableError(message string, err error) *AppError {
	return &AppError{
		Code:    http.StatusServiceUnavailable,
		Message: message,
		Err:     err,
	}
}

// WrapError оборачивает существующую ошибку с контекстом
// Если ошибка уже AppError, добавляет контекст. Иначе создает новую InternalError
func WrapError(err error, message string) *AppError {
	if err == nil {
		return nil
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		// Если это уже AppError, добавляем контекст к сообщению
		return &AppError{
			Code:    appErr.Code,
			Message: fmt.Sprintf("%s: %s", message, appErr.Message),
			Err:     appErr.Err,
			Context: appErr.Context,
		}
	}

	// Иначе создаем новую InternalError
	return NewInternalError(message, err)
}

// Проверка интерфейса middleware.HTTPError перенесена в errors_interface_test.go
// для избежания циклических импортов
