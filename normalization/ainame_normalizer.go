// Package normalization предоставляет функциональность нормализации данных контрагентов.
// Основные компоненты:
//   - CounterpartyNormalizer: основной нормализатор контрагентов
//   - AINameNormalizer: интерфейс для AI-нормализации имен
package normalization

import (
	"context"
)

// AINameNormalizer интерфейс для нормализации имен контрагентов с использованием AI.
// Реализуется MultiProviderClient и другими AI-провайдерами.
//
// Интерфейс обеспечивает единообразный способ нормализации имен компаний
// с использованием различных AI-провайдеров (OpenRouter, HuggingFace, Arliai и др.).
type AINameNormalizer interface {
	// NormalizeName нормализует название контрагента, используя AI провайдеры.
	// Параметры:
	//   - ctx: контекст для управления отменой операции
	//   - name: исходное название компании
	// Возвращает нормализованное имя или ошибку.
	NormalizeName(ctx context.Context, name string) (string, error)

	// NormalizeCounterparty нормализует контрагента с учетом ИНН/БИН.
	// Сначала пытается использовать специализированные провайдеры (DaData/Adata),
	// затем fallback на генеративные AI провайдеры.
	//
	// Параметры:
	//   - ctx: контекст для управления отменой операции
	//   - name: название компании
	//   - inn: ИНН (для российских компаний, 10 или 12 цифр)
	//   - bin: БИН (для казахстанских компаний, 12 цифр)
	//
	// Возвращает нормализованное имя или ошибку.
	//
	// Пример использования:
	//   result, err := normalizer.NormalizeCounterparty(ctx, "ООО Ромашка", "1234567890", "")
	NormalizeCounterparty(ctx context.Context, name, inn, bin string) (string, error)
}
