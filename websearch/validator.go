package websearch

import (
	"context"
	"fmt"
	"strings"
	"time"

	"httpserver/websearch/types"
)

// SearchClientInterface импортируется из search_client_interface.go

// ProductExistenceValidator валидатор для проверки существования товара
type ProductExistenceValidator struct {
	client SearchClientInterface
}

// NewProductExistenceValidator создает новый валидатор существования товара
func NewProductExistenceValidator(client SearchClientInterface) *ProductExistenceValidator {
	return &ProductExistenceValidator{
		client: client,
	}
}

// Validate проверяет существование товара по названию
func (v *ProductExistenceValidator) Validate(ctx context.Context, name string) (*types.ValidationResult, error) {
	if strings.TrimSpace(name) == "" {
		return &types.ValidationResult{
			Status:    "error",
			Message:   "Название товара не может быть пустым",
			Score:     0.0,
			Found:     false,
			Timestamp: time.Now(),
		}, nil
	}

	// Выполняем поиск
	result, err := v.client.Search(ctx, name)
	if err != nil {
		return &types.ValidationResult{
			Status:    "error",
			Message:   fmt.Sprintf("Ошибка поиска: %v", err),
			Score:     0.0,
			Found:     false,
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}, nil
	}

	// Анализируем результаты
	return v.analyzeResults(result, name), nil
}

// analyzeResults анализирует результаты поиска
func (v *ProductExistenceValidator) analyzeResults(result *types.SearchResult, name string) *types.ValidationResult {
	validation := &types.ValidationResult{
		Status:    "success",
		Found:     result.Found,
		Results:   result.Results,
		Provider:  result.Source,
		Timestamp: result.Timestamp,
		Details:   make(map[string]interface{}),
	}

	if !result.Found {
		validation.Status = "not_found"
		validation.Message = "Товар не найден в интернете"
		validation.Score = 0.0
		validation.Details["total_results"] = 0
		return validation
	}

	// Проверяем релевантность результатов
	nameLower := strings.ToLower(name)
	matchCount := 0

	for _, item := range result.Results {
		itemText := strings.ToLower(item.Title + " " + item.Snippet)
		if strings.Contains(itemText, nameLower) {
			matchCount++
		}
	}

	if matchCount > 0 {
		validation.Found = true
		score := float64(matchCount) / float64(len(result.Results))
		if score > 1.0 {
			score = 1.0
		}
		validation.Score = score
		validation.Message = fmt.Sprintf("Найдено %d релевантных результатов из %d", matchCount, len(result.Results))
		validation.Details["match_count"] = matchCount
		validation.Details["total_results"] = len(result.Results)
		validation.Details["confidence"] = result.Confidence
	} else {
		validation.Status = "not_found"
		validation.Found = false
		validation.Score = 0.0
		validation.Message = "Найдены результаты, но они не релевантны"
		validation.Details["match_count"] = 0
		validation.Details["total_results"] = len(result.Results)
	}

	return validation
}

// ProductAccuracyValidator валидатор для проверки точности данных товара
type ProductAccuracyValidator struct {
	client SearchClientInterface
}

// NewProductAccuracyValidator создает новый валидатор точности данных
func NewProductAccuracyValidator(client SearchClientInterface) *ProductAccuracyValidator {
	return &ProductAccuracyValidator{
		client: client,
	}
}

// Validate проверяет точность данных товара (название + код)
func (v *ProductAccuracyValidator) Validate(ctx context.Context, name, code string) (*types.ValidationResult, error) {
	if strings.TrimSpace(name) == "" {
		return &types.ValidationResult{
			Status:    "error",
			Message:   "Название товара не может быть пустым",
			Score:     0.0,
			Found:     false,
			Timestamp: time.Now(),
		}, nil
	}

	// Формируем поисковый запрос
	query := v.buildQuery(name, code)

	// Выполняем поиск
	result, err := v.client.Search(ctx, query)
	if err != nil {
		return &types.ValidationResult{
			Status:    "error",
			Message:   fmt.Sprintf("Ошибка поиска: %v", err),
			Score:     0.0,
			Found:     false,
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}, nil
	}

	// Анализируем точность данных
	return v.analyzeAccuracy(result, name, code), nil
}

// buildQuery формирует поисковый запрос из названия и кода
func (v *ProductAccuracyValidator) buildQuery(name, code string) string {
	parts := make([]string, 0, 2)

	if strings.TrimSpace(name) != "" {
		parts = append(parts, strings.TrimSpace(name))
	}

	if strings.TrimSpace(code) != "" {
		parts = append(parts, strings.TrimSpace(code))
	}

	return strings.Join(parts, " ")
}

// analyzeAccuracy анализирует точность данных на основе результатов поиска
func (v *ProductAccuracyValidator) analyzeAccuracy(result *types.SearchResult, name, code string) *types.ValidationResult {
	validation := &types.ValidationResult{
		Status:    "success",
		Found:     result.Found,
		Results:   result.Results,
		Provider:  result.Source,
		Timestamp: result.Timestamp,
		Details:   make(map[string]interface{}),
	}

	if !result.Found {
		validation.Status = "not_found"
		validation.Message = "Недостаточно данных для проверки точности"
		validation.Score = 0.0
		validation.Details["total_results"] = 0
		return validation
	}

	// Проверяем соответствие данных
	nameLower := strings.ToLower(name)
	codeLower := strings.ToLower(code)

	score := 0.0
	totalChecks := 0.0
	matchedItems := 0

	for _, item := range result.Results {
		itemText := strings.ToLower(item.Title + " " + item.Snippet)
		itemScore := 0.0
		itemChecks := 0.0

		// Проверка названия
		if nameLower != "" {
			totalChecks++
			itemChecks++
			if strings.Contains(itemText, nameLower) {
				score += 1.0
				itemScore += 1.0
			}
		}

		// Проверка кода
		if codeLower != "" {
			totalChecks++
			itemChecks++
			if strings.Contains(itemText, codeLower) {
				score += 1.0
				itemScore += 1.0
			}
		}

		if itemChecks > 0 && itemScore/itemChecks > 0.3 {
			matchedItems++
		}
	}

	if totalChecks > 0 {
		accuracyScore := score / totalChecks
		validation.Score = accuracyScore
		validation.Found = accuracyScore > 0.3 // Порог существования
		validation.Message = fmt.Sprintf("Оценка точности: %.2f (проверено %d параметров)", accuracyScore, int(totalChecks))
		validation.Details["accuracy_score"] = accuracyScore
		validation.Details["total_checks"] = totalChecks
		validation.Details["matched_items"] = matchedItems
		validation.Details["confidence"] = result.Confidence
		if !validation.Found {
			validation.Status = "low_accuracy"
		}
	} else {
		validation.Status = "not_found"
		validation.Score = 0.0
		validation.Found = false
		validation.Message = "Нет данных для проверки"
		validation.Details["total_checks"] = 0
	}

	return validation
}

// ProductValidator универсальный валидатор для проверки товаров
// Обёртка над ProductExistenceValidator и ProductAccuracyValidator
type ProductValidator struct {
	existenceValidator *ProductExistenceValidator
	accuracyValidator  *ProductAccuracyValidator
}

// NewProductValidator создает новый универсальный валидатор товаров
func NewProductValidator(client SearchClientInterface) *ProductValidator {
	return &ProductValidator{
		existenceValidator: NewProductExistenceValidator(client),
		accuracyValidator:  NewProductAccuracyValidator(client),
	}
}

// ValidateProductExists проверяет существование товара по названию и коду
func (v *ProductValidator) ValidateProductExists(ctx context.Context, name, code string) (*types.ValidationResult, error) {
	if code != "" {
		// Если есть код, используем валидатор точности для более полной проверки
		return v.accuracyValidator.Validate(ctx, name, code)
	}
	// Если кода нет, проверяем только существование
	return v.existenceValidator.Validate(ctx, name)
}

// ValidateDataAccuracy проверяет точность данных товара (название, код, категория)
func (v *ProductValidator) ValidateDataAccuracy(ctx context.Context, name, code, category string) (*types.ValidationResult, error) {
	// Формируем поисковый запрос с категорией
	query := name
	if code != "" {
		query = name + " " + code
	}
	if category != "" {
		query = query + " " + category
	}
	
	// Используем валидатор точности с расширенным запросом
	// Сначала проверяем точность по названию и коду
	result, err := v.accuracyValidator.Validate(ctx, name, code)
	if err != nil {
		return result, err
	}
	
	// Если категория указана, добавляем дополнительную проверку
	if category != "" && result != nil {
		// Проверяем наличие категории в результатах
		categoryLower := strings.ToLower(category)
		for _, item := range result.Results {
			itemText := strings.ToLower(item.Title + " " + item.Snippet)
			if strings.Contains(itemText, categoryLower) {
				// Категория найдена - повышаем оценку
				if result.Details == nil {
					result.Details = make(map[string]interface{})
				}
				result.Details["category_found"] = true
				result.Score = result.Score * 1.1 // Небольшое повышение за совпадение категории
				if result.Score > 1.0 {
					result.Score = 1.0
				}
				break
			}
		}
	}
	
	return result, nil
}