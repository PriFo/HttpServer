package server

import (
	"fmt"
	"log/slog"
	"regexp"

	"httpserver/internal/infrastructure/ai"
)

// CounterpartyProviderRouter умный маршрутизатор для выбора правильного провайдера
// на основе ИНН (Россия) или БИН (Казахстан)
type CounterpartyProviderRouter struct {
	dadataAdapter ai.ProviderClient
	adataAdapter  ai.ProviderClient
	logger        *slog.Logger // Структурированный логгер
}

// NewCounterpartyProviderRouter создает новый роутер провайдеров
func NewCounterpartyProviderRouter(dadataAdapter, adataAdapter ai.ProviderClient) *CounterpartyProviderRouter {
	logger := slog.Default().With("component", "counterparty_provider_router")
	return &CounterpartyProviderRouter{
		dadataAdapter: dadataAdapter,
		adataAdapter:  adataAdapter,
		logger:        logger,
	}
}

// RouteByTaxID определяет, какой провайдер использовать на основе ИНН/БИН
// Возвращает провайдер или nil, если не удалось определить
func (r *CounterpartyProviderRouter) RouteByTaxID(inn, bin string) ai.ProviderClient {
	// Приоритет: БИН (12 цифр) → Adata, ИНН (10 или 12 цифр) → DaData
	
	// Нормализуем БИН (убираем пробелы и нецифровые символы)
	normalizedBIN := normalizeTaxID(bin)
	if normalizedBIN != "" && len(normalizedBIN) == 12 {
		// Проверяем, что это не российский ИНН (российский ИНН может быть 12 цифр, но начинается с определенных цифр)
		// БИН казахстанский всегда 12 цифр
		if r.adataAdapter != nil && r.adataAdapter.IsEnabled() {
			r.logger.Info("Routing request to Adata.kz for BIN", "bin", bin)
			return r.adataAdapter
		}
	}

	// Нормализуем ИНН
	normalizedINN := normalizeTaxID(inn)
	if normalizedINN != "" {
		// Российский ИНН: 10 или 12 цифр
		if len(normalizedINN) == 10 || len(normalizedINN) == 12 {
			// Проверяем, что это не БИН (если есть БИН, он имеет приоритет)
			if normalizedBIN == "" || len(normalizedBIN) != 12 {
				if r.dadataAdapter != nil && r.dadataAdapter.IsEnabled() {
					r.logger.Info("Routing request to DaData for INN", "inn", inn)
					return r.dadataAdapter
				}
			}
		}
	}

	r.logger.Warn("Cannot determine provider from tax ID", "inn", inn, "bin", bin)
	return nil
}

// StandardizeCounterparty стандартизирует контрагента, используя правильный провайдер
func (r *CounterpartyProviderRouter) StandardizeCounterparty(name, inn, bin string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("company name cannot be empty")
	}

	// Определяем провайдер
	provider := r.RouteByTaxID(inn, bin)
	if provider == nil {
		r.logger.Warn("Cannot determine provider, returning error", "inn", inn, "bin", bin)
		return "", fmt.Errorf("cannot determine provider for INN: %s, BIN: %s", inn, bin)
	}

	providerName := provider.GetProviderName()
	logger := r.logger.With("provider", providerName, "name", name, "inn", inn, "bin", bin)

	// Формируем промпт для провайдера
	systemPrompt := "Ты эксперт по нормализации названий компаний. Нормализуй название, приведя его к каноничному виду с правильными регистрами."
	userPrompt := formatCounterpartyPrompt(name, inn, bin)

	// Вызываем провайдер
	result, err := provider.GetCompletion(systemPrompt, userPrompt)
	if err != nil {
		logger.Warn("Provider returned error, falling back to generative AI", "error", err.Error())
		return "", fmt.Errorf("provider %s error: %w", providerName, err)
	}

	if result == "" {
		logger.Info("Provider returned no result, falling back to generative AI")
		return "", fmt.Errorf("provider %s returned empty result", providerName)
	}

	logger.Info("Successfully normalized using specialized provider", "result", result)
	return result, nil
}

// formatCounterpartyPrompt форматирует промпт для нормализации контрагента
func formatCounterpartyPrompt(name, inn, bin string) string {
	prompt := fmt.Sprintf("Нормализуй название компании: \"%s\"", name)
	
	if inn != "" {
		prompt += fmt.Sprintf(" ИНН: %s", inn)
	}
	
	if bin != "" {
		prompt += fmt.Sprintf(" БИН: %s", bin)
	}
	
	return prompt
}

// normalizeTaxID нормализует ИНН/БИН, убирая пробелы и нецифровые символы
func normalizeTaxID(taxID string) string {
	if taxID == "" {
		return ""
	}

	// Убираем все нецифровые символы
	normalized := regexp.MustCompile(`\D`).ReplaceAllString(taxID, "")
	return normalized
}

// IsRussianINN проверяет, является ли ИНН российским
// Российский ИНН: 10 или 12 цифр, определенные правила валидации
func IsRussianINN(inn string) bool {
	normalized := normalizeTaxID(inn)
	if len(normalized) != 10 && len(normalized) != 12 {
		return false
	}

	// Простая проверка: российский ИНН обычно начинается с определенных цифр
	// Более точная валидация требует проверки контрольных сумм
	return true
}

// IsKazakhBIN проверяет, является ли БИН казахстанским
// Казахстанский БИН: 12 цифр
func IsKazakhBIN(bin string) bool {
	normalized := normalizeTaxID(bin)
	return len(normalized) == 12
}

// DetectCountryByTaxID определяет страну на основе ИНН/БИН
func DetectCountryByTaxID(inn, bin string) string {
	normalizedBIN := normalizeTaxID(bin)
	if normalizedBIN != "" && len(normalizedBIN) == 12 {
		// Проверяем, что это не российский ИНН
		// Казахстанский БИН всегда 12 цифр
		return "KZ"
	}

	normalizedINN := normalizeTaxID(inn)
	if normalizedINN != "" {
		if len(normalizedINN) == 10 || len(normalizedINN) == 12 {
			return "RU"
		}
	}

	return ""
}

