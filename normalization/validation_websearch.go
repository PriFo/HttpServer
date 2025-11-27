package normalization

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"httpserver/database"
	"httpserver/websearch"
)

// SetWebSearchValidators добавляет правила валидации с использованием веб-поиска
// Если validators nil, правила не добавляются
// Правила добавляются только если они включены в конфигурации
func (ve *ValidationEngine) SetWebSearchValidators(
	existenceValidator *websearch.ProductExistenceValidator,
	accuracyValidator *websearch.ProductAccuracyValidator,
	rulesConfig map[string]interface{},
) {
	if existenceValidator == nil && accuracyValidator == nil {
		return
	}

	// Проверяем конфигурацию правил
	existenceEnabled := IsRuleEnabled(rulesConfig, "product_name", "existence")
	accuracyEnabled := IsRuleEnabled(rulesConfig, "product_code", "accuracy")

	// Правило: Проверка существования товара через веб-поиск
	if existenceValidator != nil && existenceEnabled {
		ve.AddRule(ValidationRule{
			Name:        "web_search_product_exists",
			Description: "Проверка существования товара в интернете",
			Severity:    ValidationSeverityMedium,
			Validator: func(item *database.CatalogItem) error {
				// Создаем контекст с таймаутом для веб-поиска
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Выполняем валидацию существования
				result, err := existenceValidator.Validate(ctx, item.Name)
				if err != nil {
					// Не блокируем валидацию при ошибке веб-поиска
					return nil
				}

				// Если товар не найден, добавляем предупреждение
				if !result.Found || result.Status == "not_found" {
					return fmt.Errorf("товар не найден в интернете: %s", result.Message)
				}

				return nil
			},
		})
	}

	// Правило: Проверка точности данных через веб-поиск
	if accuracyValidator != nil && accuracyEnabled {
		ve.AddRule(ValidationRule{
			Name:        "web_search_data_accuracy",
			Description: "Проверка точности данных через веб-поиск",
			Severity:    ValidationSeverityLow,
			Validator: func(item *database.CatalogItem) error {
				// Создаем контекст с таймаутом
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				// Выполняем валидацию точности (название + код)
				result, err := accuracyValidator.Validate(ctx, item.Name, item.Code)
				if err != nil {
					// Не блокируем валидацию при ошибке
					return nil
				}

				// Если точность низкая, добавляем предупреждение
				if result.Score < 0.5 {
					return fmt.Errorf("низкая точность данных (%.2f): %s", result.Score, result.Message)
				}

				return nil
			},
		})
	}
}

// IsRuleEnabled проверяет, включено ли правило в конфигурации
// Экспортируется для использования в других пакетах
func IsRuleEnabled(rulesConfig map[string]interface{}, fieldName, validatorType string) bool {
	if rulesConfig == nil {
		// По умолчанию правила выключены, если конфигурация не задана
		return false
	}

	fieldConfig, exists := rulesConfig[fieldName]
	if !exists {
		return false
	}

	// Преобразуем в map для проверки
	fieldMap, ok := fieldConfig.(map[string]interface{})
	if !ok {
		return false
	}

	// Проверяем, что валидатор совпадает и правило включено
	validator, _ := fieldMap["validator"].(string)
	enabled, _ := fieldMap["enabled"].(bool)

	return validator == validatorType && enabled
}

// LoadWebSearchRulesConfig загружает конфигурацию правил веб-поиска из базы данных
func LoadWebSearchRulesConfig(db *database.ServiceDB) (map[string]interface{}, error) {
	query := `SELECT websearch_rules FROM normalization_config WHERE id = 1`

	var rulesJSON string
	err := db.GetDB().QueryRow(query).Scan(&rulesJSON)
	if err != nil {
		// Если конфигурация не найдена, возвращаем пустую конфигурацию
		return make(map[string]interface{}), nil
	}

	if rulesJSON == "" || rulesJSON == "{}" {
		return make(map[string]interface{}), nil
	}

	var rules map[string]interface{}
	if err := json.Unmarshal([]byte(rulesJSON), &rules); err != nil {
		return nil, fmt.Errorf("failed to parse websearch_rules JSON: %w", err)
	}

	return rules, nil
}
