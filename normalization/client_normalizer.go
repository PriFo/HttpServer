package normalization

import (
	"fmt"
	"log"
	"os"
	"time"

	"httpserver/database"
	"httpserver/nomenclature"
)

// ClientNormalizationResult результат нормализации для клиента
type ClientNormalizationResult struct {
	ClientID             int
	ProjectID            int
	ProcessedAt          time.Time
	TotalProcessed       int
	TotalGroups          int
	BenchmarkMatches     int
	AIEnhancedItems      int
	BasicNormalizedItems int
	NewBenchmarksCreated int
	// Группы с метаданными для сохранения
	Groups map[string]*ClientNormalizationGroup
}

// ClientNormalizationGroup группа нормализованных записей с метаданными
type ClientNormalizationGroup struct {
	Items           []*database.CatalogItem
	Category        string
	NormalizedName  string
	AIConfidence    float64
	AIReasoning     string
	ProcessingLevel string
	KpvedCode       string
	KpvedName       string
	KpvedConfidence float64
	Attributes      map[string][]*database.ItemAttribute // code -> attributes
}

// ClientNormalizer нормализатор с поддержкой клиентских эталонов
type ClientNormalizer struct {
	clientID        int
	projectID       int
	db              *database.DB
	serviceDB       *database.ServiceDB
	benchmarkStore  *ClientBenchmarkStore
	aiClient        *nomenclature.AIClient
	basicNormalizer *Normalizer
	events          chan<- string
	sessionID       *int // ID сессии нормализации
}

// WorkerConfigManagerInterface интерфейс для получения конфигурации модели
type WorkerConfigManagerInterface interface {
	GetModelAndAPIKey() (apiKey string, modelName string, err error)
}

// NewClientNormalizer создает новый клиентский нормализатор
func NewClientNormalizer(clientID, projectID int, db *database.DB, serviceDB *database.ServiceDB, events chan<- string) *ClientNormalizer {
	return NewClientNormalizerWithConfig(clientID, projectID, db, serviceDB, events, nil)
}

// NewClientNormalizerWithConfig создает новый клиентский нормализатор с конфигурацией модели
func NewClientNormalizerWithConfig(clientID, projectID int, db *database.DB, serviceDB *database.ServiceDB, events chan<- string, configManager WorkerConfigManagerInterface) *ClientNormalizer {
	normalizer := &ClientNormalizer{
		clientID:       clientID,
		projectID:      projectID,
		db:             db,
		serviceDB:      serviceDB,
		benchmarkStore: NewClientBenchmarkStore(serviceDB, projectID),
		events:         events,
	}

	// Инициализация базового нормализатора
	aiConfig := &AIConfig{
		Enabled:        true,
		MinConfidence:  0.7,
		RateLimitDelay: 100 * time.Millisecond,
		MaxRetries:     3,
	}
	
	// Создаем функцию получения API ключа для базового нормализатора
	var getAPIKey func() string
	if configManager != nil {
		getAPIKey = func() string {
			apiKey, _, err := configManager.GetModelAndAPIKey()
			if err != nil {
				return "" // Fallback на переменную окружения в NewNormalizer
			}
			return apiKey
		}
	}
	
	normalizer.basicNormalizer = NewNormalizerWithStopCheck(db, events, aiConfig, nil, getAPIKey)

	// Инициализация AI клиента
	var apiKey, model string
	if configManager != nil {
		var err error
		apiKey, model, err = configManager.GetModelAndAPIKey()
		if err != nil {
			// Fallback на переменные окружения
			apiKey = os.Getenv("ARLIAI_API_KEY")
			model = os.Getenv("ARLIAI_MODEL")
		}
	} else {
		// Fallback на переменные окружения, если конфигурация не доступна
		apiKey = os.Getenv("ARLIAI_API_KEY")
		model = os.Getenv("ARLIAI_MODEL")
	}

	if model == "" {
		model = "gpt-4o-mini" // Последний fallback
	}

	if apiKey != "" {
		normalizer.aiClient = nomenclature.NewAIClient(apiKey, model)
	}

	return normalizer
}

// ProcessWithClientBenchmarks выполняет нормализацию с использованием эталонов клиента
func (c *ClientNormalizer) ProcessWithClientBenchmarks(items []*database.CatalogItem) (*ClientNormalizationResult, error) {
	result := &ClientNormalizationResult{
		ClientID:    c.clientID,
		ProjectID:   c.projectID,
		ProcessedAt: time.Now(),
	}

	c.sendEvent("Начало нормализации с использованием эталонов клиента...")
	log.Printf("Начало нормализации для клиента %d, проекта %d", c.clientID, c.projectID)

	groups := make(map[string]*ClientNormalizationGroup)
	processedCount := 0

	for _, item := range items {
		// 1. Проверка против эталонов клиента
		benchmark, found := c.benchmarkStore.FindBenchmark(item.Name)
		if found {
			// Используем эталонную запись
			result.BenchmarkMatches++
			c.sendEvent(fmt.Sprintf("✓ Найдено совпадение с эталоном: %s -> %s", item.Name, benchmark.NormalizedName))

			// Увеличиваем счетчик использования
			if err := c.benchmarkStore.UpdateUsage(benchmark.ID); err != nil {
				log.Printf("Ошибка обновления счетчика эталона: %v", err)
			}

			// Группируем по нормализованному имени из эталона
			key := fmt.Sprintf("%s|%s", benchmark.Category, benchmark.NormalizedName)
			if groups[key] == nil {
				groups[key] = &ClientNormalizationGroup{
					Category:        benchmark.Category,
					NormalizedName:  benchmark.NormalizedName,
					ProcessingLevel: "benchmark",
					AIConfidence:    1.0, // Эталоны имеют максимальную уверенность
					Items:           make([]*database.CatalogItem, 0),
					Attributes:      make(map[string][]*database.ItemAttribute),
				}
			}
			groups[key].Items = append(groups[key].Items, item)
			processedCount++
			continue
		}

		// 2. Базовая нормализация с извлечением атрибутов
		category := c.basicNormalizer.categorizer.Categorize(item.Name)
		var normalizedName string
		var attributes []*database.ItemAttribute
		if c.basicNormalizer.nameNormalizer != nil {
			normalizedName, attributes = c.basicNormalizer.nameNormalizer.ExtractAttributes(item.Name)
		} else {
			// Fallback на простую нормализацию
			normalizedName = item.Name
			attributes = []*database.ItemAttribute{}
		}
		if normalizedName == "" {
			normalizedName = item.Name // Используем исходное имя, если нормализация дала пустую строку
		}
		aiConfidence := 0.0
		aiReasoning := ""
		processingLevel := "basic"

		// 3. AI-усиление если требуется
		if c.basicNormalizer.useAI && c.basicNormalizer.aiNormalizer != nil &&
			c.basicNormalizer.aiNormalizer.RequiresAI(item.Name, category) {
			aiResult, err := c.basicNormalizer.processWithAI(item.Name)
			if err == nil && aiResult.Confidence >= c.basicNormalizer.aiConfig.MinConfidence {
				category = aiResult.Category
				normalizedName = aiResult.NormalizedName
				aiConfidence = aiResult.Confidence
				aiReasoning = aiResult.Reasoning
				processingLevel = "ai_enhanced"
				result.AIEnhancedItems++

				// Сохраняем как потенциальный эталон
				if aiConfidence >= 0.9 {
					if err := c.benchmarkStore.SavePotentialBenchmark(
						item.Name,
						normalizedName,
						category,
						"",
						aiConfidence,
					); err == nil {
						result.NewBenchmarksCreated++
					}
				}
			} else {
				result.BasicNormalizedItems++
			}
		} else {
			result.BasicNormalizedItems++
		}

		// Группируем записи
		key := fmt.Sprintf("%s|%s", category, normalizedName)
		if groups[key] == nil {
			groups[key] = &ClientNormalizationGroup{
				Category:        category,
				NormalizedName:  normalizedName,
				AIConfidence:    aiConfidence,
				AIReasoning:     aiReasoning,
				ProcessingLevel: processingLevel,
				Items:           make([]*database.CatalogItem, 0),
				Attributes:      make(map[string][]*database.ItemAttribute),
			}
		}
		groups[key].Items = append(groups[key].Items, item)
		if len(attributes) > 0 && item.Code != "" {
			groups[key].Attributes[item.Code] = attributes
		}
		processedCount++

		// Отправляем событие каждые 1000 записей
		if processedCount%1000 == 0 {
			c.sendEvent(fmt.Sprintf("Обработано %d из %d записей", processedCount, len(items)))
		}
	}

	result.TotalProcessed = processedCount
	result.TotalGroups = len(groups)
	result.Groups = groups

	c.sendEvent(fmt.Sprintf("Нормализация завершена. Обработано: %d, Групп: %d, Эталонов использовано: %d",
		result.TotalProcessed, result.TotalGroups, result.BenchmarkMatches))

	return result, nil
}

// SetSessionID устанавливает ID сессии нормализации
func (c *ClientNormalizer) SetSessionID(sessionID int) {
	c.sessionID = &sessionID
	if c.basicNormalizer != nil {
		c.basicNormalizer.SetSessionID(sessionID)
	}
}

// sendEvent отправляет событие в канал
func (c *ClientNormalizer) sendEvent(message string) {
	if c.events != nil {
		select {
		case c.events <- message:
		default:
			// Канал полон, пропускаем событие
		}
	}
}
