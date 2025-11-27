package server

import (
	"fmt"
	"log"
	"strings"

	"httpserver/internal/infrastructure/ai"
	"httpserver/normalization"
)

// MultiProviderAINormalizerInterface интерфейс для совместимости с AINormalizer
type MultiProviderAINormalizerInterface interface {
	NormalizeWithAI(name string) (*normalization.AIResult, error)
	RequiresAI(name, category string) bool
}

// MultiProviderAINormalizer обертка для использования оркестратора провайдеров в нормализации
type MultiProviderAINormalizer struct {
	orchestrator *ai.ProviderOrchestrator
	systemPrompt string
	cache        *normalization.AICache
}

// NewMultiProviderAINormalizer создает новый мульти-провайдерный AI нормализатор
func NewMultiProviderAINormalizer(orchestrator *ai.ProviderOrchestrator, cache *normalization.AICache) *MultiProviderAINormalizer {
	systemPrompt := `Ты - эксперт по нормализации наименований товаров и их категоризации.

ТВОЯ ЗАДАЧА:
1. НОРМАЛИЗОВАТЬ наименование товара:
   - Исправить опечатки и грамматические ошибки
   - Привести к стандартной форме
   - Удалить технические коды, артикулы, размеры (но сохранить смысл)
   - Унифицировать синонимы (например: "молоток" вместо "молотак", "отвертка" вместо "отвертка крестовая №2")
   - Использовать единообразную терминологию

2. ОПРЕДЕЛИТЬ КАТЕГОРИЮ товара из списка:
   - инструмент
   - медикаменты
   - стройматериалы
   - электроника
   - оборудование
   - расходники
   - автоаксессуары
   - канцелярия
   - средства очистки
   - продукты
   - сельское хозяйство
   - связь
   - сантехника
   - мебель
   - инструменты измерительные
   - программное обеспечение
   - упаковка
   - другое

ВАЖНЫЕ ПРАВИЛА:
- Нормализованное имя должно быть лаконичным и понятным (2-100 символов)
- Сохраняй ключевые характеристики товара (материал, назначение)
- Категория должна точно соответствовать товару
- Если не уверен в категории - выбирай "другое"
- Уверенность (confidence) от 0.0 до 1.0 (0.9+ только если полностью уверен)

ФОРМАТ ОТВЕТА - СТРОГО JSON:
{
    "normalized_name": "нормализованное наименование",
    "category": "категория из списка",
    "confidence": 0.95,
    "reasoning": "краткое объяснение нормализации и выбора категории"
}

Отвечай ТОЛЬКО JSON, без дополнительных пояснений.`

	return &MultiProviderAINormalizer{
		orchestrator: orchestrator,
		systemPrompt: systemPrompt,
		cache:        cache,
	}
}

// NormalizeWithAI нормализует название товара с помощью всех активных провайдеров
func (m *MultiProviderAINormalizer) NormalizeWithAI(name string) (*normalization.AIResult, error) {
	// Проверяем кэш
	if m.cache != nil {
		sourceName := strings.ToLower(strings.TrimSpace(name))
		if cached, exists := m.cache.Get(sourceName); exists {
			return &normalization.AIResult{
				NormalizedName: cached.NormalizedName,
				Category:       cached.Category,
				Confidence:     cached.Confidence,
				Reasoning:      cached.Reasoning,
			}, nil
		}
	}

	// Используем оркестратор для запроса ко всем провайдерам
	userPrompt := fmt.Sprintf("НАИМЕНОВАНИЕ ТОВАРА ДЛЯ ОБРАБОТКИ: \"%s\"", name)

	aggregated, err := m.orchestrator.Normalize(m.systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("orchestrator failed: %w", err)
	}

	if aggregated.FinalResult == nil {
		return nil, fmt.Errorf("no successful result from any provider (tried %d providers, %d succeeded, %d failed)",
			aggregated.TotalProviders, aggregated.SuccessCount, aggregated.ErrorCount)
	}

	// Преобразуем результат в формат AIResult
	result := &normalization.AIResult{
		NormalizedName: aggregated.FinalResult.NormalizedName,
		Category:       aggregated.FinalResult.KpvedName, // Используем KpvedName как категорию
		Confidence:     aggregated.FinalResult.Confidence,
		Reasoning:      aggregated.FinalResult.Reasoning,
	}

	// Если категория пустая, используем дефолтную
	if result.Category == "" {
		result.Category = "другое"
	}

	// Сохраняем в кэш
	if m.cache != nil {
		sourceName := strings.ToLower(strings.TrimSpace(name))
		m.cache.Set(sourceName, result.NormalizedName, result.Category, result.Confidence, result.Reasoning)
	}

	// Логируем информацию о стратегии и результатах
	log.Printf("[MultiProvider] Normalized '%s' using strategy '%s': %d/%d providers succeeded, final confidence: %.2f",
		name, aggregated.Strategy, aggregated.SuccessCount, aggregated.TotalProviders, result.Confidence)

	return result, nil
}

// RequiresAI определяет, требует ли товар AI обработки
func (m *MultiProviderAINormalizer) RequiresAI(name, category string) bool {
	// Используем ту же логику, что и в обычном AINormalizer
	// Товары с простыми названиями и известными категориями не требуют AI
	if category != "" && category != "другое" {
		// Если категория уже определена и не "другое", возможно не нужен AI
		nameLower := strings.ToLower(name)
		if len(nameLower) < 20 && !strings.Contains(nameLower, "?") {
			return false
		}
	}
	return true
}
