package server

import (
	"fmt"
	"regexp"
	"strings"
)

// DaDataProviderAdapter адаптер для DaDataClient, реализующий интерфейс ProviderClient
type DaDataProviderAdapter struct {
	client *DaDataClient
	name   string
}

// NewDaDataProviderAdapter создает новый адаптер для DaData
func NewDaDataProviderAdapter(client *DaDataClient) *DaDataProviderAdapter {
	return &DaDataProviderAdapter{
		client: client,
		name:   "DaData",
	}
}

// GetCompletion реализует интерфейс ProviderClient
// Извлекает название компании из промпта и вызывает DaData API
func (a *DaDataProviderAdapter) GetCompletion(systemPrompt, userPrompt string) (string, error) {
	if a.client == nil {
		return "", fmt.Errorf("DaData client is not initialized")
	}

	// Извлекаем название компании из userPrompt
	// Промпт может быть в формате: "Нормализуй название компании: \"ООО Ромашка\""
	companyName := extractCompanyNameFromPrompt(userPrompt)
	if companyName == "" {
		return "", fmt.Errorf("failed to extract company name from prompt")
	}

	// Пытаемся извлечь ИНН из промпта, если он есть
	inn := extractINNFromPrompt(userPrompt)

	// Вызываем DaData API
	suggestion, err := a.client.SuggestParty(companyName, inn)
	if err != nil {
		return "", fmt.Errorf("DaData API error: %w", err)
	}

	// Форматируем результат в строку
	result := formatPartySuggestion(suggestion)
	return result, nil
}

// GetProviderName возвращает имя провайдера
func (a *DaDataProviderAdapter) GetProviderName() string {
	return a.name
}

// IsEnabled проверяет, активен ли провайдер
func (a *DaDataProviderAdapter) IsEnabled() bool {
	return a.client != nil && a.client.apiKey != ""
}

// extractCompanyNameFromPrompt извлекает название компании из промпта
func extractCompanyNameFromPrompt(prompt string) string {
	// Пробуем найти название в кавычках
	quotedPattern := regexp.MustCompile(`["']([^"']+)["']`)
	matches := quotedPattern.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	// Пробуем найти после ключевых слов
	keywords := []string{
		"название компании:",
		"компания:",
		"контрагент:",
		"организация:",
	}
	
	for _, keyword := range keywords {
		idx := strings.Index(strings.ToLower(prompt), strings.ToLower(keyword))
		if idx != -1 {
			start := idx + len(keyword)
			rest := prompt[start:]
			// Берем текст до конца строки или до следующего ключевого слова
			parts := strings.Fields(rest)
			if len(parts) > 0 {
				// Берем первые несколько слов (до 10)
				maxWords := 10
				if len(parts) < maxWords {
					maxWords = len(parts)
				}
				return strings.Join(parts[:maxWords], " ")
			}
		}
	}

	// Если ничего не найдено, возвращаем весь промпт (убираем ключевые слова)
	cleaned := prompt
	for _, keyword := range keywords {
		cleaned = strings.ReplaceAll(strings.ToLower(cleaned), strings.ToLower(keyword), "")
	}
	cleaned = strings.TrimSpace(cleaned)
	if cleaned != "" {
		return cleaned
	}

	return ""
}

// extractINNFromPrompt извлекает ИНН из промпта
func extractINNFromPrompt(prompt string) string {
	// ИНН может быть 10 или 12 цифр
	innPattern := regexp.MustCompile(`(?:ИНН|INN)[\s:]*(\d{10,12})`)
	matches := innPattern.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		return matches[1]
	}

	// Пробуем найти просто последовательность из 10-12 цифр
	digitsPattern := regexp.MustCompile(`\b(\d{10,12})\b`)
	matches = digitsPattern.FindStringSubmatch(prompt)
	if len(matches) > 1 {
		// Проверяем, что это не часть другого числа
		inn := matches[1]
		if len(inn) == 10 || len(inn) == 12 {
			return inn
		}
	}

	return ""
}

// formatPartySuggestion форматирует результат DaData в строку
func formatPartySuggestion(suggestion *PartySuggestion) string {
	if suggestion == nil {
		return ""
	}

	// Используем полное название, если оно есть, иначе короткое
	name := suggestion.FullName
	if name == "" {
		name = suggestion.ShortName
	}

	// Формируем строку с информацией
	parts := []string{name}
	
	if suggestion.INN != "" {
		parts = append(parts, fmt.Sprintf("ИНН: %s", suggestion.INN))
	}
	
	if suggestion.KPP != "" {
		parts = append(parts, fmt.Sprintf("КПП: %s", suggestion.KPP))
	}

	// Возвращаем только нормализованное название (для совместимости с интерфейсом)
	// Дополнительная информация (ИНН, КПП) может быть использована в будущем
	return name
}

