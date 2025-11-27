package pipeline_normalization

import "errors"

// Ошибки конфигурации
var (
	ErrNoAlgorithmsEnabled = errors.New("no algorithms enabled in configuration")
	ErrInvalidWeight        = errors.New("algorithm weight must be between 0.0 and 1.0")
	ErrInvalidThreshold     = errors.New("threshold must be between 0.0 and 1.0")
	ErrInvalidCombineMethod = errors.New("invalid combine method")
)

// AlgorithmType тип алгоритма нормализации
type AlgorithmType string

const (
	// AlgorithmSoundex Soundex для русского языка
	AlgorithmSoundex AlgorithmType = "soundex"
	// AlgorithmMetaphone Metaphone для русского языка
	AlgorithmMetaphone AlgorithmType = "metaphone"
	// AlgorithmJaccard индекс Жаккара
	AlgorithmJaccard AlgorithmType = "jaccard"
	// AlgorithmNGrams N-граммы
	AlgorithmNGrams AlgorithmType = "ngrams"
	// AlgorithmDamerauLevenshtein расстояние Дамерау-Левенштейна
	AlgorithmDamerauLevenshtein AlgorithmType = "damerau_levenshtein"
	// AlgorithmCosine косинусная близость
	AlgorithmCosine AlgorithmType = "cosine"
	// AlgorithmToken токен-ориентированные методы
	AlgorithmToken AlgorithmType = "token"
	// AlgorithmJaro алгоритм Jaro
	AlgorithmJaro AlgorithmType = "jaro"
	// AlgorithmJaroWinkler алгоритм Jaro-Winkler
	AlgorithmJaroWinkler AlgorithmType = "jaro_winkler"
	// AlgorithmLCS алгоритм на основе наибольшей общей подпоследовательности
	AlgorithmLCS AlgorithmType = "lcs"
)

// AlgorithmConfig конфигурация для отдельного алгоритма
type AlgorithmConfig struct {
	Type      AlgorithmType `json:"type"`
	Enabled   bool          `json:"enabled"`
	Weight    float64       `json:"weight"`    // Вес алгоритма при комбинировании (0.0 - 1.0)
	Threshold float64       `json:"threshold"` // Порог срабатывания (0.0 - 1.0)
	Params    map[string]interface{} `json:"params"` // Параметры алгоритма
}

// NormalizationPipelineConfig конфигурация pipeline нормализации
type NormalizationPipelineConfig struct {
	// Алгоритмы и их конфигурации
	Algorithms []AlgorithmConfig `json:"algorithms"`

	// Общие настройки
	MinSimilarity     float64 `json:"min_similarity"`     // Минимальная схожесть для определения дубликата
	CombineMethod     string  `json:"combine_method"`     // Метод комбинирования: "weighted", "max", "min", "average"
	ParallelExecution bool    `json:"parallel_execution"` // Параллельное выполнение алгоритмов
	CacheEnabled      bool    `json:"cache_enabled"`      // Включить кэширование результатов

	// Метрики качества
	CalculateMetrics bool    `json:"calculate_metrics"` // Вычислять метрики качества
	PrecisionWeight  float64 `json:"precision_weight"`  // Вес точности (precision)
	RecallWeight     float64 `json:"recall_weight"`     // Вес полноты (recall)
}

// NewDefaultConfig создает конфигурацию по умолчанию
func NewDefaultConfig() *NormalizationPipelineConfig {
	return &NormalizationPipelineConfig{
		Algorithms: []AlgorithmConfig{
			{
				Type:      AlgorithmDamerauLevenshtein,
				Enabled:   true,
				Weight:    0.3,
				Threshold: 0.85,
				Params:    make(map[string]interface{}),
			},
			{
				Type:      AlgorithmJaccard,
				Enabled:   true,
				Weight:    0.2,
				Threshold: 0.75,
				Params: map[string]interface{}{
					"use_ngrams": false,
				},
			},
			{
				Type:      AlgorithmNGrams,
				Enabled:   true,
				Weight:    0.2,
				Threshold: 0.80,
				Params: map[string]interface{}{
					"n": 2, // биграммы
				},
			},
			{
				Type:      AlgorithmCosine,
				Enabled:   true,
				Weight:    0.15,
				Threshold: 0.80,
				Params: map[string]interface{}{
					"use_tfidf": true,
				},
			},
			{
				Type:      AlgorithmToken,
				Enabled:   true,
				Weight:    0.15,
				Threshold: 0.75,
				Params: map[string]interface{}{
					"use_weighted": true,
				},
			},
		},
		MinSimilarity:     0.85,
		CombineMethod:     "weighted",
		ParallelExecution: true,
		CacheEnabled:      true,
		CalculateMetrics:  true,
		PrecisionWeight:  0.5,
		RecallWeight:     0.5,
	}
}

// NewFastConfig создает быструю конфигурацию (меньше алгоритмов)
func NewFastConfig() *NormalizationPipelineConfig {
	return &NormalizationPipelineConfig{
		Algorithms: []AlgorithmConfig{
			{
				Type:      AlgorithmDamerauLevenshtein,
				Enabled:   true,
				Weight:    0.5,
				Threshold: 0.85,
				Params:    make(map[string]interface{}),
			},
			{
				Type:      AlgorithmJaccard,
				Enabled:   true,
				Weight:    0.5,
				Threshold: 0.75,
				Params: map[string]interface{}{
					"use_ngrams": false,
				},
			},
		},
		MinSimilarity:     0.85,
		CombineMethod:     "weighted",
		ParallelExecution: true,
		CacheEnabled:      true,
		CalculateMetrics:  false,
		PrecisionWeight:  0.5,
		RecallWeight:     0.5,
	}
}

// NewPreciseConfig создает точную конфигурацию (все алгоритмы)
func NewPreciseConfig() *NormalizationPipelineConfig {
	return &NormalizationPipelineConfig{
		Algorithms: []AlgorithmConfig{
			{
				Type:      AlgorithmSoundex,
				Enabled:   true,
				Weight:    0.1,
				Threshold: 0.90,
				Params:    make(map[string]interface{}),
			},
			{
				Type:      AlgorithmMetaphone,
				Enabled:   true,
				Weight:    0.1,
				Threshold: 0.90,
				Params:    make(map[string]interface{}),
			},
			{
				Type:      AlgorithmDamerauLevenshtein,
				Enabled:   true,
				Weight:    0.2,
				Threshold: 0.85,
				Params:    make(map[string]interface{}),
			},
			{
				Type:      AlgorithmJaccard,
				Enabled:   true,
				Weight:    0.15,
				Threshold: 0.75,
				Params: map[string]interface{}{
					"use_ngrams": true,
					"n_gram_size": 2,
				},
			},
			{
				Type:      AlgorithmNGrams,
				Enabled:   true,
				Weight:    0.15,
				Threshold: 0.80,
				Params: map[string]interface{}{
					"n": 2,
				},
			},
			{
				Type:      AlgorithmCosine,
				Enabled:   true,
				Weight:    0.15,
				Threshold: 0.80,
				Params: map[string]interface{}{
					"use_tfidf": true,
				},
			},
			{
				Type:      AlgorithmToken,
				Enabled:   true,
				Weight:    0.15,
				Threshold: 0.75,
				Params: map[string]interface{}{
					"use_weighted": true,
					"use_positional": false,
				},
			},
		},
		MinSimilarity:     0.85,
		CombineMethod:     "weighted",
		ParallelExecution: false, // Последовательное выполнение для точности
		CacheEnabled:      true,
		CalculateMetrics:  true,
		PrecisionWeight:  0.5,
		RecallWeight:     0.5,
	}
}

// Validate проверяет корректность конфигурации
func (c *NormalizationPipelineConfig) Validate() error {
	// Проверяем, что есть хотя бы один включенный алгоритм
	hasEnabled := false
	for _, alg := range c.Algorithms {
		if alg.Enabled {
			hasEnabled = true
			break
		}
	}
	if !hasEnabled {
		return ErrNoAlgorithmsEnabled
	}

	// Проверяем веса алгоритмов
	totalWeight := 0.0
	for _, alg := range c.Algorithms {
		if alg.Enabled {
			if alg.Weight < 0 || alg.Weight > 1 {
				return ErrInvalidWeight
			}
			totalWeight += alg.Weight
		}
	}

	// Нормализуем веса если они не в сумме дают 1.0
	if totalWeight > 0 && totalWeight != 1.0 {
		// Нормализуем веса
		for i := range c.Algorithms {
			if c.Algorithms[i].Enabled {
				c.Algorithms[i].Weight /= totalWeight
			}
		}
	}

	// Проверяем пороги
	if c.MinSimilarity < 0 || c.MinSimilarity > 1 {
		return ErrInvalidThreshold
	}

	// Проверяем метод комбинирования
	validMethods := map[string]bool{
		"weighted": true,
		"max":      true,
		"min":      true,
		"average":  true,
	}
	if !validMethods[c.CombineMethod] {
		return ErrInvalidCombineMethod
	}

	return nil
}

// GetEnabledAlgorithms возвращает список включенных алгоритмов
func (c *NormalizationPipelineConfig) GetEnabledAlgorithms() []AlgorithmConfig {
	enabled := make([]AlgorithmConfig, 0)
	for _, alg := range c.Algorithms {
		if alg.Enabled {
			enabled = append(enabled, alg)
		}
	}
	return enabled
}

// SetAlgorithmWeight устанавливает вес алгоритма
func (c *NormalizationPipelineConfig) SetAlgorithmWeight(algType AlgorithmType, weight float64) {
	for i := range c.Algorithms {
		if c.Algorithms[i].Type == algType {
			c.Algorithms[i].Weight = weight
			break
		}
	}
}

// SetAlgorithmThreshold устанавливает порог для алгоритма
func (c *NormalizationPipelineConfig) SetAlgorithmThreshold(algType AlgorithmType, threshold float64) {
	for i := range c.Algorithms {
		if c.Algorithms[i].Type == algType {
			c.Algorithms[i].Threshold = threshold
			break
		}
	}
}

// EnableAlgorithm включает алгоритм
func (c *NormalizationPipelineConfig) EnableAlgorithm(algType AlgorithmType) {
	for i := range c.Algorithms {
		if c.Algorithms[i].Type == algType {
			c.Algorithms[i].Enabled = true
			break
		}
	}
}

// DisableAlgorithm отключает алгоритм
func (c *NormalizationPipelineConfig) DisableAlgorithm(algType AlgorithmType) {
	for i := range c.Algorithms {
		if c.Algorithms[i].Type == algType {
			c.Algorithms[i].Enabled = false
			break
		}
	}
}

