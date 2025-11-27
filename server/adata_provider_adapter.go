package server

import (
	"fmt"
	"regexp"
)

// AdataProviderAdapter адаптер для AdataClient, реализующий интерфейс ProviderClient
type AdataProviderAdapter struct {
	client *AdataClient
	name   string
}

// NewAdataProviderAdapter создает новый адаптер для Adata
func NewAdataProviderAdapter(client *AdataClient) *AdataProviderAdapter {
	return &AdataProviderAdapter{
		client: client,
		name:   "Adata",
	}
}

// GetCompletion реализует интерфейс ProviderClient
// Извлекает название компании из промпта и вызывает Adata API
func (a *AdataProviderAdapter) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("Adata client is not initialized")
	}

	// Извлекаем название компании из userPrompt
	companyName := extractCompanyNameFromPrompt(userPrompt)
	if companyName == "" {
		return "", fmt.Errorf("failed to extract company name from prompt")
	}

	// Пытаемся извлечь БИН из промпта, если он есть
	bin := extractBINFromPrompt(userPrompt)

	// Вызываем Adata API
	companyInfo, err := a.client.FindCompany(companyName, bin)
	if err != nil {
		return "", fmt.Errorf("Adata API error: %w", err)
	}

	// Форматируем результат в строку
	result := formatCompanyInfo(companyInfo)
	return result, nil
}

// GetProviderName возвращает имя провайдера
func (a *AdataProviderAdapter) GetProviderName() string {
	return a.name
}

// IsEnabled проверяет, активен ли провайдер
func (a *AdataProviderAdapter) IsEnabled() bool {
	return a.client != nil && a.client.apiToken != ""
}

// extractBINFromPrompt извлекает БИН из промпта
func extractBINFromPrompt(prompt string) string {
	// БИН - это 12 цифр
	binPattern := regexp.MustCompile(`(?:БИН|BIN|ИИН|IIN)[\s:]*(\d{12})`)
	matches := binPattern.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		return matches[1]
	}

	// Пробуем найти последовательность из 12 цифр
	digitsPattern := regexp.MustCompile(`\b(\d{12})\b`)
	matches = digitsPattern.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// formatCompanyInfo форматирует результат Adata в строку
func formatCompanyInfo(info *CompanyInfo) string {
	if info == nil {
		return ""
	}

	// Используем полное название, если оно есть, иначе короткое
	name := info.FullName
	if name == "" {
		name = info.ShortName
	}

	// Возвращаем только нормализованное название (для совместимости с интерфейсом)
	return name
}

