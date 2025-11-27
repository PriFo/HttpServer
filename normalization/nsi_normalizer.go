package normalization

import (
	"fmt"
	"sync"
)

// NSINormalizer унифицированный интерфейс для всех методов нормализации и поиска дублей НСИ
type NSINormalizer struct {
	advancedNormalizer *AdvancedNormalizer
	fuzzyAlgorithms    *FuzzyAlgorithms
	nameNormalizer     *NameNormalizer
	duplicateAnalyzer  *DuplicateAnalyzer
	evaluationMetrics  *EvaluationMetrics
	cache              map[string]string
	cacheMutex         sync.RWMutex
}

// NewNSINormalizer создает новый унифицированный нормализатор НСИ
func NewNSINormalizer() *NSINormalizer {
	return &NSINormalizer{
		advancedNormalizer: NewAdvancedNormalizer(),
		fuzzyAlgorithms:    NewFuzzyAlgorithms(),
		nameNormalizer:     NewNameNormalizer(),
		duplicateAnalyzer:  NewDuplicateAnalyzer(),
		evaluationMetrics:  NewEvaluationMetrics(),
		cache:              make(map[string]string),
	}
}

// NormalizeName выполняет комплексную нормализацию наименования
// Использует все доступные методы нормализации
func (nsi *NSINormalizer) NormalizeName(name string, options NormalizationOptions) string {
	if name == "" {
		return ""
	}

	// Проверяем кэш
	nsi.cacheMutex.RLock()
	if cached, ok := nsi.cache[name]; ok {
		nsi.cacheMutex.RUnlock()
		return cached
	}
	nsi.cacheMutex.RUnlock()

	// 1. Базовая нормализация (удаление кодов, размеров и т.д.)
	normalized := nsi.nameNormalizer.NormalizeName(name)

	// 2. Расширенная нормализация
	normalized = nsi.advancedNormalizer.AdvancedNormalize(normalized, options)

	// Сохраняем в кэш
	nsi.cacheMutex.Lock()
	nsi.cache[name] = normalized
	nsi.cacheMutex.Unlock()

	return normalized
}

// FindDuplicates находит дубликаты используя все доступные алгоритмы
func (nsi *NSINormalizer) FindDuplicates(items []DuplicateItem, config DuplicateDetectionConfig) []DuplicateGroup {
	var allGroups []DuplicateGroup

	// 1. Exact matching (если включен)
	if config.UseExactMatching {
		exactGroups := nsi.duplicateAnalyzer.AnalyzeDuplicates(items)
		allGroups = append(allGroups, exactGroups...)
	}

	// 2. Fuzzy matching с различными алгоритмами
	if config.UseFuzzyMatching {
		fuzzyGroups := nsi.findFuzzyDuplicates(items, config)
		allGroups = append(allGroups, fuzzyGroups...)
	}

	// 3. Объединяем пересекающиеся группы
	if config.MergeOverlapping {
		allGroups = nsi.duplicateAnalyzer.MergeOverlappingGroups(allGroups)
	}

	// 4. Фильтруем по порогу уверенности
	if config.MinConfidence > 0 {
		filteredGroups := make([]DuplicateGroup, 0)
		for _, group := range allGroups {
			if group.Confidence >= config.MinConfidence {
				filteredGroups = append(filteredGroups, group)
			}
		}
		allGroups = filteredGroups
	}

	return allGroups
}

// findFuzzyDuplicates находит дубликаты используя нечеткие алгоритмы
func (nsi *NSINormalizer) findFuzzyDuplicates(items []DuplicateItem, config DuplicateDetectionConfig) []DuplicateGroup {
	var groups []DuplicateGroup
	processed := make(map[int]bool)
	groupCounter := 0

	weights := config.SimilarityWeights
	if weights.Levenshtein == 0 && weights.DamerauLevenshtein == 0 &&
		weights.Bigram == 0 && weights.Trigram == 0 &&
		weights.Jaccard == 0 && weights.Soundex == 0 && weights.Metaphone == 0 {
		weights = DefaultSimilarityWeights()
	}

	for i := 0; i < len(items); i++ {
		if processed[items[i].ID] {
			continue
		}

		var duplicates []DuplicateItem
		var itemIDs []int
		duplicates = append(duplicates, items[i])
		itemIDs = append(itemIDs, items[i].ID)

		for j := i + 1; j < len(items); j++ {
			if processed[items[j].ID] {
				continue
			}

			// Вычисляем комбинированную схожесть
			similarity := nsi.fuzzyAlgorithms.CombinedSimilarity(
				items[i].NormalizedName,
				items[j].NormalizedName,
				weights,
			)

			if similarity >= config.Threshold {
				duplicates = append(duplicates, items[j])
				itemIDs = append(itemIDs, items[j].ID)
				processed[items[j].ID] = true
			}
		}

		if len(duplicates) >= 2 {
			processed[items[i].ID] = true

			// Вычисляем среднюю схожесть в группе
			avgSimilarity := 0.0
			pairCount := 0
			for k := 0; k < len(duplicates); k++ {
				for l := k + 1; l < len(duplicates); l++ {
					avgSimilarity += nsi.fuzzyAlgorithms.CombinedSimilarity(
						duplicates[k].NormalizedName,
						duplicates[l].NormalizedName,
						weights,
					)
					pairCount++
				}
			}
			if pairCount > 0 {
				avgSimilarity /= float64(pairCount)
			}

			groups = append(groups, DuplicateGroup{
				GroupID:         fmt.Sprintf("fuzzy_%d", groupCounter),
				Type:            DuplicateTypeSemantic,
				SimilarityScore: avgSimilarity,
				ItemIDs:         itemIDs,
				Items:           duplicates,
				Confidence:      avgSimilarity,
				Reason:          "Fuzzy matching with combined algorithms",
			})
			groupCounter++
		}
	}

	return groups
}

// EvaluateAlgorithm оценивает качество алгоритма поиска дублей
func (nsi *NSINormalizer) EvaluateAlgorithm(
	predicted []DuplicateGroup,
	actual []DuplicateGroup,
) MetricsResult {
	return nsi.evaluationMetrics.EvaluateAlgorithm(predicted, actual)
}

// FindOptimalThreshold находит оптимальный порог схожести
func (nsi *NSINormalizer) FindOptimalThreshold(
	items []DuplicateItem,
	actualPairs map[Pair]bool,
	thresholds []float64,
	weights SimilarityWeights,
) (float64, MetricsResult) {
	similarityFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.CombinedSimilarity(
			item1.NormalizedName,
			item2.NormalizedName,
			weights,
		)
	}

	return nsi.evaluationMetrics.CalculateOptimalThreshold(
		items,
		actualPairs,
		similarityFunc,
		thresholds,
	)
}

// CompareAlgorithms сравнивает различные алгоритмы поиска дублей
func (nsi *NSINormalizer) CompareAlgorithms(
	items []DuplicateItem,
	actualPairs map[Pair]bool,
	threshold float64,
) AlgorithmComparison {
	comparison := AlgorithmComparison{
		Threshold: threshold,
		Results:   make(map[string]MetricsResult),
	}

	// Levenshtein
	levFunc := func(item1, item2 DuplicateItem) float64 {
		return 1.0 - float64(levenshteinDistance(item1.NormalizedName, item2.NormalizedName))/
			float64(max(len([]rune(item1.NormalizedName)), len([]rune(item2.NormalizedName))))
	}
	levGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, levFunc, threshold)
	comparison.Results["Levenshtein"] = nsi.evaluationMetrics.EvaluateAlgorithm(levGroups, []DuplicateGroup{})

	// Damerau-Levenshtein
	dlFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.DamerauLevenshteinSimilarity(item1.NormalizedName, item2.NormalizedName)
	}
	dlGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, dlFunc, threshold)
	comparison.Results["DamerauLevenshtein"] = nsi.evaluationMetrics.EvaluateAlgorithm(dlGroups, []DuplicateGroup{})

	// Bigram
	bigramFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.BigramSimilarity(item1.NormalizedName, item2.NormalizedName)
	}
	bigramGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, bigramFunc, threshold)
	comparison.Results["Bigram"] = nsi.evaluationMetrics.EvaluateAlgorithm(bigramGroups, []DuplicateGroup{})

	// Trigram
	trigramFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.TrigramSimilarity(item1.NormalizedName, item2.NormalizedName)
	}
	trigramGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, trigramFunc, threshold)
	comparison.Results["Trigram"] = nsi.evaluationMetrics.EvaluateAlgorithm(trigramGroups, []DuplicateGroup{})

	// Jaccard
	jaccardFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.JaccardIndex(item1.NormalizedName, item2.NormalizedName)
	}
	jaccardGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, jaccardFunc, threshold)
	comparison.Results["Jaccard"] = nsi.evaluationMetrics.EvaluateAlgorithm(jaccardGroups, []DuplicateGroup{})

	// Soundex
	soundexFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.SoundexSimilarity(item1.NormalizedName, item2.NormalizedName)
	}
	soundexGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, soundexFunc, threshold)
	comparison.Results["Soundex"] = nsi.evaluationMetrics.EvaluateAlgorithm(soundexGroups, []DuplicateGroup{})

	// Metaphone
	metaphoneFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.MetaphoneSimilarity(item1.NormalizedName, item2.NormalizedName)
	}
	metaphoneGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, metaphoneFunc, threshold)
	comparison.Results["Metaphone"] = nsi.evaluationMetrics.EvaluateAlgorithm(metaphoneGroups, []DuplicateGroup{})

	// Combined
	combinedFunc := func(item1, item2 DuplicateItem) float64 {
		return nsi.fuzzyAlgorithms.CombinedSimilarity(
			item1.NormalizedName,
			item2.NormalizedName,
			DefaultSimilarityWeights(),
		)
	}
	combinedGroups := nsi.evaluationMetrics.findDuplicatesWithThreshold(items, combinedFunc, threshold)
	comparison.Results["Combined"] = nsi.evaluationMetrics.EvaluateAlgorithm(combinedGroups, []DuplicateGroup{})

	return comparison
}

// AlgorithmComparison результат сравнения алгоритмов
type AlgorithmComparison struct {
	Threshold float64
	Results   map[string]MetricsResult
}

// GetBestAlgorithm возвращает название лучшего алгоритма по F1-мере
func (ac AlgorithmComparison) GetBestAlgorithm() string {
	bestAlgorithm := ""
	bestF1 := 0.0

	for name, metrics := range ac.Results {
		if metrics.F1Score > bestF1 {
			bestF1 = metrics.F1Score
			bestAlgorithm = name
		}
	}

	return bestAlgorithm
}

// DuplicateDetectionConfig конфигурация для поиска дубликатов
type DuplicateDetectionConfig struct {
	UseExactMatching  bool              // Использовать точное совпадение
	UseFuzzyMatching  bool              // Использовать нечеткий поиск
	Threshold         float64           // Порог схожести (0.0 - 1.0)
	MinConfidence     float64           // Минимальная уверенность
	MergeOverlapping  bool              // Объединять пересекающиеся группы
	SimilarityWeights SimilarityWeights // Веса для алгоритмов схожести
}

// DefaultDuplicateDetectionConfig возвращает конфигурацию по умолчанию
func DefaultDuplicateDetectionConfig() DuplicateDetectionConfig {
	return DuplicateDetectionConfig{
		UseExactMatching:  true,
		UseFuzzyMatching:  true,
		Threshold:         0.85,
		MinConfidence:     0.0,
		MergeOverlapping:  true,
		SimilarityWeights: DefaultSimilarityWeights(),
	}
}

// ClearCache очищает кэш нормализации
func (nsi *NSINormalizer) ClearCache() {
	nsi.cacheMutex.Lock()
	defer nsi.cacheMutex.Unlock()
	nsi.cache = make(map[string]string)
}

// GetCacheSize возвращает размер кэша
func (nsi *NSINormalizer) GetCacheSize() int {
	nsi.cacheMutex.RLock()
	defer nsi.cacheMutex.RUnlock()
	return len(nsi.cache)
}

// BatchNormalize выполняет нормализацию для батча наименований
func (nsi *NSINormalizer) BatchNormalize(names []string, options NormalizationOptions) []string {
	results := make([]string, len(names))
	for i, name := range names {
		results[i] = nsi.NormalizeName(name, options)
	}
	return results
}

// BatchFindDuplicates выполняет поиск дубликатов для батча записей
func (nsi *NSINormalizer) BatchFindDuplicates(
	batches [][]DuplicateItem,
	config DuplicateDetectionConfig,
) [][]DuplicateGroup {
	results := make([][]DuplicateGroup, len(batches))
	for i, batch := range batches {
		results[i] = nsi.FindDuplicates(batch, config)
	}
	return results
}
