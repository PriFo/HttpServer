package normalization

import (
	"strings"
)

// MethodSelector автоматически выбирает оптимальный метод для данных
type MethodSelector struct {
	universalMatcher *UniversalMatcher
	recommendations  map[string]string // характеристики -> рекомендуемый метод
}

// NewMethodSelector создает новый селектор методов
func NewMethodSelector(universalMatcher *UniversalMatcher) *MethodSelector {
	ms := &MethodSelector{
		universalMatcher: universalMatcher,
		recommendations:  make(map[string]string),
	}

	// Инициализируем рекомендации
	ms.initializeRecommendations()

	return ms
}

// initializeRecommendations инициализирует рекомендации по выбору метода
func (ms *MethodSelector) initializeRecommendations() {
	// Рекомендации на основе характеристик данных
	ms.recommendations["short_text"] = "jaro_winkler"            // Короткие тексты
	ms.recommendations["long_text"] = "jaccard"                  // Длинные тексты
	ms.recommendations["typos_expected"] = "damerau_levenshtein" // Ожидаются опечатки
	ms.recommendations["phonetic_similar"] = "soundex"           // Фонетическое сходство
	ms.recommendations["word_order_variation"] = "jaccard"       // Вариации порядка слов
	ms.recommendations["character_variation"] = "char_trigram"   // Вариации символов
}

// SelectMethod автоматически выбирает оптимальный метод на основе характеристик данных
func (ms *MethodSelector) SelectMethod(s1, s2 string) (string, []string, error) {
	characteristics := ms.analyzeCharacteristics(s1, s2)

	// Определяем приоритетные методы на основе характеристик
	methods := ms.selectMethodsByCharacteristics(characteristics)

	if len(methods) == 0 {
		// Используем методы по умолчанию
		methods = []string{"levenshtein", "jaccard", "jaro_winkler"}
	}

	primaryMethod := methods[0]

	return primaryMethod, methods, nil
}

// analyzeCharacteristics анализирует характеристики пары строк
func (ms *MethodSelector) analyzeCharacteristics(s1, s2 string) map[string]bool {
	characteristics := make(map[string]bool)

	len1 := len([]rune(s1))
	len2 := len([]rune(s2))
	avgLen := (len1 + len2) / 2

	// Короткие тексты
	if avgLen < 10 {
		characteristics["short_text"] = true
	}

	// Длинные тексты
	if avgLen > 50 {
		characteristics["long_text"] = true
	}

	// Проверяем на возможные опечатки (расстояние Левенштейна небольшое)
	if ms.universalMatcher != nil {
		levenshtein, err := ms.universalMatcher.Similarity(s1, s2, "levenshtein")
		if err == nil && levenshtein > 0.7 && levenshtein < 0.95 {
			characteristics["typos_expected"] = true
		}

		// Проверяем фонетическое сходство
		soundex1, _ := ms.universalMatcher.Similarity(s1, s2, "soundex")
		if soundex1 > 0.8 {
			characteristics["phonetic_similar"] = true
		}

		// Проверяем вариации порядка слов
		jaccard, err := ms.universalMatcher.Similarity(s1, s2, "jaccard")
		if err == nil && jaccard > 0.5 {
			characteristics["word_order_variation"] = true
		}
	} else {
		// Fallback на базовые проверки
		if len([]rune(s1)) > 0 && len([]rune(s2)) > 0 {
			// Простая проверка длины
			lenDiff := float64(abs(len([]rune(s1))-len([]rune(s2)))) / float64(max(len([]rune(s1)), len([]rune(s2))))
			if lenDiff < 0.2 {
				characteristics["typos_expected"] = true
			}
		}
	}

	return characteristics
}

// selectMethodsByCharacteristics выбирает методы на основе характеристик
func (ms *MethodSelector) selectMethodsByCharacteristics(characteristics map[string]bool) []string {
	methods := make([]string, 0)

	// Приоритет 1: Фонетическое сходство
	if characteristics["phonetic_similar"] {
		methods = append(methods, "soundex", "metaphone", "phonetic")
	}

	// Приоритет 2: Опечатки
	if characteristics["typos_expected"] {
		methods = append(methods, "damerau_levenshtein", "jaro_winkler")
	}

	// Приоритет 3: Короткие тексты
	if characteristics["short_text"] {
		methods = append(methods, "jaro_winkler", "levenshtein")
	}

	// Приоритет 4: Длинные тексты
	if characteristics["long_text"] {
		methods = append(methods, "jaccard", "lcs", "ngram")
	}

	// Приоритет 5: Вариации порядка слов
	if characteristics["word_order_variation"] {
		methods = append(methods, "jaccard", "dice")
	}

	// Приоритет 6: Вариации символов
	if characteristics["character_variation"] {
		methods = append(methods, "char_trigram", "char_bigram")
	}

	// Если ничего не подошло, используем универсальные методы
	if len(methods) == 0 {
		methods = []string{"levenshtein", "jaccard", "jaro_winkler"}
	}

	// Удаляем дубликаты
	uniqueMethods := make(map[string]bool)
	result := make([]string, 0)
	for _, method := range methods {
		if !uniqueMethods[method] {
			uniqueMethods[method] = true
			result = append(result, method)
		}
	}

	return result
}

// abs возвращает абсолютное значение
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// max возвращает максимальное значение
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// RecommendHybridMethod рекомендует гибридный метод на основе данных
func (ms *MethodSelector) RecommendHybridMethod(s1, s2 string) ([]string, []float64, error) {
	characteristics := ms.analyzeCharacteristics(s1, s2)
	methods := ms.selectMethodsByCharacteristics(characteristics)

	// Ограничиваем до 3-5 методов для гибрида
	if len(methods) > 5 {
		methods = methods[:5]
	}

	// Определяем веса на основе характеристик
	weights := ms.determineWeights(methods, characteristics)

	return methods, weights, nil
}

// determineWeights определяет веса для методов
func (ms *MethodSelector) determineWeights(methods []string, characteristics map[string]bool) []float64 {
	weights := make([]float64, len(methods))

	// Базовые веса
	for i := range weights {
		weights[i] = 1.0 / float64(len(methods))
	}

	// Корректируем веса на основе характеристик
	if characteristics["phonetic_similar"] {
		for i, method := range methods {
			if strings.Contains(method, "soundex") || strings.Contains(method, "metaphone") || strings.Contains(method, "phonetic") {
				weights[i] *= 1.5
			}
		}
	}

	if characteristics["typos_expected"] {
		for i, method := range methods {
			if method == "damerau_levenshtein" || method == "jaro_winkler" {
				weights[i] *= 1.3
			}
		}
	}

	if characteristics["short_text"] {
		for i, method := range methods {
			if method == "jaro_winkler" {
				weights[i] *= 1.2
			}
		}
	}

	if characteristics["long_text"] {
		for i, method := range methods {
			if method == "jaccard" || method == "lcs" {
				weights[i] *= 1.2
			}
		}
	}

	// Нормализуем веса
	total := 0.0
	for _, w := range weights {
		total += w
	}
	if total > 0 {
		for i := range weights {
			weights[i] /= total
		}
	}

	return weights
}

// GetBestMethodForDataset рекомендует лучший метод для набора данных
func (ms *MethodSelector) GetBestMethodForDataset(samples []string) (string, error) {
	if len(samples) < 2 {
		return "levenshtein", nil // Метод по умолчанию
	}

	// Анализируем характеристики набора данных
	avgLength := 0
	hasShort := false
	hasLong := false

	for _, sample := range samples {
		length := len([]rune(sample))
		avgLength += length
		if length < 10 {
			hasShort = true
		}
		if length > 50 {
			hasLong = true
		}
	}

	avgLength /= len(samples)

	// Рекомендации на основе анализа
	if hasShort && !hasLong {
		return "jaro_winkler", nil
	}
	if hasLong && !hasShort {
		return "jaccard", nil
	}
	if avgLength < 15 {
		return "jaro_winkler", nil
	}
	if avgLength > 30 {
		return "jaccard", nil
	}

	// Универсальный метод
	return "levenshtein", nil
}
