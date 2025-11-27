package normalization

import (
	"fmt"

	"httpserver/normalization/pipeline_normalization"
)

// AlgorithmSelector фабрика для создания и выбора алгоритмов нормализации
type AlgorithmSelector struct{}

// NewAlgorithmSelector создает новый селектор алгоритмов
func NewAlgorithmSelector() *AlgorithmSelector {
	return &AlgorithmSelector{}
}

// CreatePipeline создает pipeline с указанной конфигурацией
func (as *AlgorithmSelector) CreatePipeline(configType string) (*pipeline_normalization.NormalizationPipeline, error) {
	var config *pipeline_normalization.NormalizationPipelineConfig

	switch configType {
	case "default":
		config = pipeline_normalization.NewDefaultConfig()
	case "fast":
		config = pipeline_normalization.NewFastConfig()
	case "precise":
		config = pipeline_normalization.NewPreciseConfig()
	default:
		return nil, fmt.Errorf("unknown configuration type: %s", configType)
	}

	return pipeline_normalization.NewNormalizationPipeline(config)
}

// CreateCustomPipeline создает pipeline с пользовательской конфигурацией
func (as *AlgorithmSelector) CreateCustomPipeline(config *pipeline_normalization.NormalizationPipelineConfig) (*pipeline_normalization.NormalizationPipeline, error) {
	return pipeline_normalization.NewNormalizationPipeline(config)
}

// RecommendConfiguration рекомендует конфигурацию на основе требований
func (as *AlgorithmSelector) RecommendConfiguration(requirements PipelineRequirements) *pipeline_normalization.NormalizationPipelineConfig {
	config := pipeline_normalization.NewDefaultConfig()

	// Настройка на основе требований
	if requirements.Speed > requirements.Accuracy {
		// Приоритет скорости
		config = pipeline_normalization.NewFastConfig()
		config.ParallelExecution = true
	} else if requirements.Accuracy > requirements.Speed {
		// Приоритет точности
		config = pipeline_normalization.NewPreciseConfig()
		config.ParallelExecution = false
	}

	// Настройка метрик
	if requirements.CalculateMetrics {
		config.CalculateMetrics = true
		config.PrecisionWeight = requirements.PrecisionWeight
		config.RecallWeight = requirements.RecallWeight
	}

	// Настройка порогов
	if requirements.MinSimilarity > 0 {
		config.MinSimilarity = requirements.MinSimilarity
	}

	// Включение/отключение алгоритмов на основе требований
	if requirements.UsePhonetic {
		config.EnableAlgorithm(pipeline_normalization.AlgorithmSoundex)
		config.EnableAlgorithm(pipeline_normalization.AlgorithmMetaphone)
	} else {
		config.DisableAlgorithm(pipeline_normalization.AlgorithmSoundex)
		config.DisableAlgorithm(pipeline_normalization.AlgorithmMetaphone)
	}

	if requirements.UseNGrams {
		config.EnableAlgorithm(pipeline_normalization.AlgorithmNGrams)
		config.EnableAlgorithm(pipeline_normalization.AlgorithmJaccard)
	} else {
		config.DisableAlgorithm(pipeline_normalization.AlgorithmNGrams)
	}

	return config
}

// PipelineRequirements требования к pipeline
type PipelineRequirements struct {
	Speed            float64 // Приоритет скорости (0.0 - 1.0)
	Accuracy         float64 // Приоритет точности (0.0 - 1.0)
	CalculateMetrics bool    // Вычислять метрики качества
	PrecisionWeight  float64 // Вес точности
	RecallWeight     float64 // Вес полноты
	MinSimilarity    float64 // Минимальная схожесть
	UsePhonetic      bool    // Использовать фонетические алгоритмы
	UseNGrams        bool    // Использовать N-граммы
}

// GetPresetConfigurations возвращает список предустановленных конфигураций
func (as *AlgorithmSelector) GetPresetConfigurations() []PresetConfiguration {
	return []PresetConfiguration{
		{
			Name:        "default",
			Description: "Сбалансированная конфигурация (скорость и точность)",
			Config:      pipeline_normalization.NewDefaultConfig(),
		},
		{
			Name:        "fast",
			Description: "Быстрая конфигурация (приоритет скорости)",
			Config:      pipeline_normalization.NewFastConfig(),
		},
		{
			Name:        "precise",
			Description: "Точная конфигурация (приоритет точности, все алгоритмы)",
			Config:      pipeline_normalization.NewPreciseConfig(),
		},
	}
}

// PresetConfiguration предустановленная конфигурация
type PresetConfiguration struct {
	Name        string
	Description string
	Config      *pipeline_normalization.NormalizationPipelineConfig
}

// CompareConfigurations сравнивает две конфигурации
func (as *AlgorithmSelector) CompareConfigurations(
	config1, config2 *pipeline_normalization.NormalizationPipelineConfig,
) ConfigurationComparison {
	comparison := ConfigurationComparison{
		Config1Algorithms: len(config1.GetEnabledAlgorithms()),
		Config2Algorithms: len(config2.GetEnabledAlgorithms()),
		Config1Parallel:   config1.ParallelExecution,
		Config2Parallel:   config2.ParallelExecution,
		Config1Cache:      config1.CacheEnabled,
		Config2Cache:      config2.CacheEnabled,
	}

	// Определяем, какая конфигурация быстрее
	if config1.ParallelExecution && !config2.ParallelExecution {
		comparison.FasterConfig = "config1"
	} else if !config1.ParallelExecution && config2.ParallelExecution {
		comparison.FasterConfig = "config2"
	} else {
		comparison.FasterConfig = "equal"
	}

	// Определяем, какая конфигурация точнее (больше алгоритмов)
	if len(config1.GetEnabledAlgorithms()) > len(config2.GetEnabledAlgorithms()) {
		comparison.MoreAccurateConfig = "config1"
	} else if len(config1.GetEnabledAlgorithms()) < len(config2.GetEnabledAlgorithms()) {
		comparison.MoreAccurateConfig = "config2"
	} else {
		comparison.MoreAccurateConfig = "equal"
	}

	return comparison
}

// ConfigurationComparison сравнение конфигураций
type ConfigurationComparison struct {
	Config1Algorithms  int
	Config2Algorithms  int
	Config1Parallel    bool
	Config2Parallel    bool
	Config1Cache       bool
	Config2Cache       bool
	FasterConfig       string
	MoreAccurateConfig string
}

// GetAlgorithmInfo возвращает информацию об алгоритме
func (as *AlgorithmSelector) GetAlgorithmInfo(algType pipeline_normalization.AlgorithmType) AlgorithmInfo {
	info := AlgorithmInfo{
		Type:        string(algType),
		Description: "",
		UseCases:    []string{},
		Complexity:  "",
	}

	switch algType {
	case pipeline_normalization.AlgorithmSoundex:
		info.Description = "Фонетический алгоритм Soundex для русского языка. Хорош для поиска похожих по звучанию слов."
		info.UseCases = []string{"Поиск опечаток", "Фонетическое сравнение", "Быстрый поиск"}
		info.Complexity = "O(n)"
	case pipeline_normalization.AlgorithmMetaphone:
		info.Description = "Улучшенный фонетический алгоритм Metaphone. Более точный чем Soundex."
		info.UseCases = []string{"Точный фонетический поиск", "Обработка сложных слов"}
		info.Complexity = "O(n)"
	case pipeline_normalization.AlgorithmJaccard:
		info.Description = "Индекс Жаккара для сравнения множеств токенов или N-грамм."
		info.UseCases = []string{"Сравнение по общим словам", "Анализ текстовой схожести"}
		info.Complexity = "O(n+m)"
	case pipeline_normalization.AlgorithmNGrams:
		info.Description = "Сравнение на основе N-грамм (биграммы, триграммы)."
		info.UseCases = []string{"Детальное сравнение", "Учет порядка символов"}
		info.Complexity = "O(n*m)"
	case pipeline_normalization.AlgorithmDamerauLevenshtein:
		info.Description = "Расстояние Дамерау-Левенштейна с учетом транспозиций."
		info.UseCases = []string{"Поиск опечаток", "Точное сравнение строк", "Учет перестановок"}
		info.Complexity = "O(n*m)"
	case pipeline_normalization.AlgorithmCosine:
		info.Description = "Косинусная близость на основе TF-IDF векторов."
		info.UseCases = []string{"Семантическое сравнение", "Анализ документов"}
		info.Complexity = "O(n+m)"
	case pipeline_normalization.AlgorithmToken:
		info.Description = "Токен-ориентированное сравнение с взвешиванием."
		info.UseCases = []string{"Сравнение по словам", "Учет частоты токенов"}
		info.Complexity = "O(n+m)"
	}

	return info
}

// AlgorithmInfo информация об алгоритме
type AlgorithmInfo struct {
	Type        string
	Description string
	UseCases    []string
	Complexity  string
}
