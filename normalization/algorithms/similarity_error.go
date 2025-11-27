package algorithms

import (
	"fmt"
)

// SimilarityError ошибка системы схожести
type SimilarityError struct {
	Code    string
	Message string
	Details map[string]interface{}
	Err     error
}

// Error реализует интерфейс error
func (e *SimilarityError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap возвращает вложенную ошибку
func (e *SimilarityError) Unwrap() error {
	return e.Err
}

// Коды ошибок
const (
	ErrCodeInvalidInput      = "INVALID_INPUT"
	ErrCodeInvalidWeights    = "INVALID_WEIGHTS"
	ErrCodeInvalidThreshold  = "INVALID_THRESHOLD"
	ErrCodeEmptyData         = "EMPTY_DATA"
	ErrCodeTrainingFailed    = "TRAINING_FAILED"
	ErrCodeExportFailed      = "EXPORT_FAILED"
	ErrCodeImportFailed      = "IMPORT_FAILED"
	ErrCodeCacheError        = "CACHE_ERROR"
	ErrCodePerformanceError  = "PERFORMANCE_ERROR"
)

// NewSimilarityError создает новую ошибку
func NewSimilarityError(code, message string, err error) *SimilarityError {
	return &SimilarityError{
		Code:    code,
		Message: message,
		Err:     err,
		Details: make(map[string]interface{}),
	}
}

// WithDetail добавляет детали к ошибке
func (e *SimilarityError) WithDetail(key string, value interface{}) *SimilarityError {
	e.Details[key] = value
	return e
}

// IsInvalidInput проверяет, является ли ошибка ошибкой неверного ввода
func IsInvalidInput(err error) bool {
	if se, ok := err.(*SimilarityError); ok {
		return se.Code == ErrCodeInvalidInput
	}
	return false
}

// IsTrainingFailed проверяет, является ли ошибка ошибкой обучения
func IsTrainingFailed(err error) bool {
	if se, ok := err.(*SimilarityError); ok {
		return se.Code == ErrCodeTrainingFailed
	}
	return false
}

// ValidateWeights проверяет валидность весов
func ValidateWeights(weights *SimilarityWeights) error {
	if weights == nil {
		return NewSimilarityError(ErrCodeInvalidWeights, "weights cannot be nil", nil)
	}

	total := weights.JaroWinkler + weights.LCS + weights.Phonetic + weights.Ngram + weights.Jaccard
	if total <= 0 {
		return NewSimilarityError(ErrCodeInvalidWeights, "total weight must be greater than 0", nil).
			WithDetail("total", total)
	}

	// Проверяем, что все веса неотрицательны
	if weights.JaroWinkler < 0 || weights.LCS < 0 || weights.Phonetic < 0 ||
		weights.Ngram < 0 || weights.Jaccard < 0 {
		return NewSimilarityError(ErrCodeInvalidWeights, "all weights must be non-negative", nil).
			WithDetail("weights", weights)
	}

	return nil
}

// ValidateThreshold проверяет валидность порога
func ValidateThreshold(threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return NewSimilarityError(ErrCodeInvalidThreshold,
			fmt.Sprintf("threshold must be between 0 and 1, got %.2f", threshold), nil).
			WithDetail("threshold", threshold)
	}
	return nil
}

// ValidatePair проверяет валидность пары строк
func ValidatePair(s1, s2 string) error {
	if s1 == "" && s2 == "" {
		return NewSimilarityError(ErrCodeInvalidInput, "both strings cannot be empty", nil)
	}
	return nil
}

// ValidatePairs проверяет валидность массива пар
func ValidatePairs(pairs []SimilarityPair) error {
	if len(pairs) == 0 {
		return NewSimilarityError(ErrCodeEmptyData, "pairs array cannot be empty", nil)
	}

	for i, pair := range pairs {
		if err := ValidatePair(pair.S1, pair.S2); err != nil {
			return NewSimilarityError(ErrCodeInvalidInput,
				fmt.Sprintf("invalid pair at index %d", i), err).
				WithDetail("index", i).
				WithDetail("pair", pair)
		}
	}

	return nil
}

// ValidateTestPairs проверяет валидность тестовых пар
func ValidateTestPairs(pairs []SimilarityTestPair) error {
	if len(pairs) == 0 {
		return NewSimilarityError(ErrCodeEmptyData, "test pairs array cannot be empty", nil)
	}

	for i, pair := range pairs {
		if err := ValidatePair(pair.S1, pair.S2); err != nil {
			return NewSimilarityError(ErrCodeInvalidInput,
				fmt.Sprintf("invalid test pair at index %d", i), err).
				WithDetail("index", i).
				WithDetail("pair", pair)
		}
	}

	return nil
}

