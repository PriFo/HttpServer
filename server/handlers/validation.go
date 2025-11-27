package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// ValidationError представляет ошибку валидации
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
}

// ValidateIntParam валидирует целочисленный параметр из query string
func ValidateIntParam(r *http.Request, paramName string, defaultValue, min, max int) (int, error) {
	valueStr := r.URL.Query().Get(paramName)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, &ValidationError{
			Field:   paramName,
			Message: fmt.Sprintf("must be an integer, got: %s", valueStr),
		}
	}

	if min > 0 && value < min {
		return 0, &ValidationError{
			Field:   paramName,
			Message: fmt.Sprintf("must be at least %d, got: %d", min, value),
		}
	}

	if max > 0 && value > max {
		return 0, &ValidationError{
			Field:   paramName,
			Message: fmt.Sprintf("must be at most %d, got: %d", max, value),
		}
	}

	return value, nil
}

// ValidateIntPathParam валидирует целочисленный параметр из path
func ValidateIntPathParam(paramStr, paramName string) (int, error) {
	if paramStr == "" {
		return 0, &ValidationError{
			Field:   paramName,
			Message: "is required",
		}
	}

	value, err := strconv.Atoi(paramStr)
	if err != nil {
		return 0, &ValidationError{
			Field:   paramName,
			Message: fmt.Sprintf("must be an integer, got: %s", paramStr),
		}
	}

	if value <= 0 {
		return 0, &ValidationError{
			Field:   paramName,
			Message: fmt.Sprintf("must be positive, got: %d", value),
		}
	}

	return value, nil
}

// ValidateStringParam валидирует строковый параметр
func ValidateStringParam(value, paramName string, required bool, maxLength int) error {
	if required && strings.TrimSpace(value) == "" {
		return &ValidationError{
			Field:   paramName,
			Message: "is required",
		}
	}

	if maxLength > 0 && len(value) > maxLength {
		return &ValidationError{
			Field:   paramName,
			Message: fmt.Sprintf("must be at most %d characters, got: %d", maxLength, len(value)),
		}
	}

	return nil
}

// ValidateSearchQuery валидирует поисковый запрос
func ValidateSearchQuery(query string, maxLength int) error {
	if maxLength > 0 && len(query) > maxLength {
		return &ValidationError{
			Field:   "search",
			Message: fmt.Sprintf("search query must be at most %d characters", maxLength),
		}
	}
	return nil
}

// ValidatePaginationParams валидирует параметры пагинации
func ValidatePaginationParams(r *http.Request, defaultPage, defaultLimit, maxLimit int) (page, limit int, err error) {
	page, err = ValidateIntParam(r, "page", defaultPage, 1, 1000)
	if err != nil {
		return 0, 0, err
	}

	limit, err = ValidateIntParam(r, "limit", defaultLimit, 1, maxLimit)
	if err != nil {
		return 0, 0, err
	}

	return page, limit, nil
}

// ValidateIDParam валидирует ID параметр из query string
func ValidateIDParam(r *http.Request, paramName string) (int, error) {
	return ValidateIntParam(r, paramName, 0, 1, 0)
}

// ValidateIDPathParam валидирует ID параметр из path
func ValidateIDPathParam(paramStr, paramName string) (int, error) {
	return ValidateIntPathParam(paramStr, paramName)
}

// ValidateEnumParam валидирует параметр из списка допустимых значений
func ValidateEnumParam(value, paramName string, allowedValues []string, required bool) error {
	if required && value == "" {
		return &ValidationError{
			Field:   paramName,
			Message: "is required",
		}
	}

	if value == "" {
		return nil // Пустое значение разрешено, если не требуется
	}

	for _, allowed := range allowedValues {
		if strings.EqualFold(value, allowed) {
			return nil
		}
	}

	return &ValidationError{
		Field:   paramName,
		Message: fmt.Sprintf("must be one of: %s, got: %s", strings.Join(allowedValues, ", "), value),
	}
}

// ValidateSortParams валидирует параметры сортировки
func ValidateSortParams(sortBy, order string, allowedFields []string) error {
	if sortBy != "" {
		allowed := false
		for _, field := range allowedFields {
			if strings.EqualFold(sortBy, field) {
				allowed = true
				break
			}
		}
		if !allowed {
			return &ValidationError{
				Field:   "sort_by",
				Message: fmt.Sprintf("must be one of: %s, got: %s", strings.Join(allowedFields, ", "), sortBy),
			}
		}
	}

	if order != "" && order != "asc" && order != "desc" {
		return &ValidationError{
			Field:   "order",
			Message: "must be 'asc' or 'desc', got: " + order,
		}
	}

	return nil
}
