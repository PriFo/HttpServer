package algorithms

import (
	"fmt"
	"strings"
)

// MatchingRule определяет правило сопоставления для поиска дублей
type MatchingRule struct {
	ID          string            `json:"id"`           // Уникальный идентификатор правила
	Name        string            `json:"name"`         // Название правила
	Description string            `json:"description"`  // Описание правила
	Fields      []string          `json:"fields"`       // Поля для сравнения
	Algorithm   string            `json:"algorithm"`    // Алгоритм сравнения: "exact", "fuzzy", "phonetic", "ngram", "combined"
	Threshold   float64           `json:"threshold"`    // Порог схожести (0.0 - 1.0)
	Weight      float64           `json:"weight"`       // Вес правила при комбинировании
	Enabled     bool              `json:"enabled"`       // Включено ли правило
	Config      map[string]interface{} `json:"config"` // Дополнительная конфигурация
}

// RuleSet набор правил для конкретного справочника
type RuleSet struct {
	ID          string        `json:"id"`           // Уникальный идентификатор набора
	Name        string        `json:"name"`         // Название набора правил
	ReferenceID string        `json:"reference_id"`  // ID справочника, к которому применяется
	Rules       []MatchingRule `json:"rules"`      // Список правил
	Priority    int           `json:"priority"`     // Приоритет набора (чем выше, тем раньше применяется)
	Enabled     bool          `json:"enabled"`      // Включен ли набор
}

// RuleEngine движок правил сопоставления
type RuleEngine struct {
	ruleSets    map[string]*RuleSet // Наборы правил по ID справочника
	algorithms  *AlgorithmRegistry  // Реестр алгоритмов
}

// AlgorithmRegistry реестр доступных алгоритмов
type AlgorithmRegistry struct {
	similarity  *SimilarityMetrics
	phonetic    *PhoneticMatcher
	ngram       *NGramGenerator
}

// NewRuleEngine создает новый движок правил
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		ruleSets: make(map[string]*RuleSet),
		algorithms: &AlgorithmRegistry{
			similarity: NewSimilarityMetrics(),
			phonetic:   NewPhoneticMatcher(),
			ngram:      NewNGramGenerator(2), // биграммы по умолчанию
		},
	}
}

// RegisterRuleSet регистрирует набор правил для справочника
func (re *RuleEngine) RegisterRuleSet(ruleSet *RuleSet) {
	re.ruleSets[ruleSet.ReferenceID] = ruleSet
}

// GetRuleSet возвращает набор правил для справочника
func (re *RuleEngine) GetRuleSet(referenceID string) (*RuleSet, bool) {
	ruleSet, exists := re.ruleSets[referenceID]
	return ruleSet, exists
}

// MatchRecords проверяет, являются ли две записи дублями согласно правилам
// Возвращает схожесть и причину совпадения
func (re *RuleEngine) MatchRecords(record1, record2 map[string]string, referenceID string) (float64, string, bool) {
	ruleSet, exists := re.GetRuleSet(referenceID)
	if !exists || !ruleSet.Enabled {
		// Используем правила по умолчанию
		return re.defaultMatch(record1, record2)
	}

	var totalSimilarity float64
	var totalWeight float64
	var reasons []string

	for _, rule := range ruleSet.Rules {
		if !rule.Enabled {
			continue
		}

		similarity, reason := re.applyRule(rule, record1, record2)
		if similarity >= rule.Threshold {
			weightedSimilarity := similarity * rule.Weight
			totalSimilarity += weightedSimilarity
			totalWeight += rule.Weight
			reasons = append(reasons, reason)
		}
	}

	if totalWeight == 0 {
		return 0.0, "", false
	}

	// Нормализуем взвешенную схожесть
	finalSimilarity := totalSimilarity / totalWeight
	reason := strings.Join(reasons, "; ")

	// Считаем дублем, если финальная схожесть >= 0.7
	isDuplicate := finalSimilarity >= 0.7

	return finalSimilarity, reason, isDuplicate
}

// applyRule применяет одно правило к записям
func (re *RuleEngine) applyRule(rule MatchingRule, record1, record2 map[string]string) (float64, string) {
	// Извлекаем значения полей
	values1 := make([]string, 0, len(rule.Fields))
	values2 := make([]string, 0, len(rule.Fields))

	for _, field := range rule.Fields {
		val1, ok1 := record1[field]
		val2, ok2 := record2[field]

		if ok1 && val1 != "" {
			values1 = append(values1, strings.ToLower(strings.TrimSpace(val1)))
		}
		if ok2 && val2 != "" {
			values2 = append(values2, strings.ToLower(strings.TrimSpace(val2)))
		}
	}

	if len(values1) == 0 || len(values2) == 0 {
		return 0.0, fmt.Sprintf("rule %s: missing fields", rule.Name)
	}

	// Объединяем значения полей
	text1 := strings.Join(values1, " ")
	text2 := strings.Join(values2, " ")

	// Применяем выбранный алгоритм
	var similarity float64
	var reason string

	switch rule.Algorithm {
	case "exact":
		similarity, reason = re.exactMatch(text1, text2, rule.Name)
	case "fuzzy":
		similarity, reason = re.fuzzyMatch(text1, text2, rule.Name)
	case "phonetic":
		similarity, reason = re.phoneticMatch(text1, text2, rule.Name)
	case "ngram":
		similarity, reason = re.ngramMatch(text1, text2, rule.Name)
	case "combined":
		similarity, reason = re.combinedMatch(text1, text2, rule.Name)
	default:
		similarity, reason = re.fuzzyMatch(text1, text2, rule.Name)
	}

	return similarity, reason
}

// exactMatch точное совпадение
func (re *RuleEngine) exactMatch(text1, text2, ruleName string) (float64, string) {
	if text1 == text2 {
		return 1.0, fmt.Sprintf("%s: exact match", ruleName)
	}
	return 0.0, fmt.Sprintf("%s: no exact match", ruleName)
}

// fuzzyMatch нечеткое совпадение (Левенштейна)
func (re *RuleEngine) fuzzyMatch(text1, text2, ruleName string) (float64, string) {
	similarity := re.algorithms.similarity.LevenshteinSimilarity(text1, text2)
	return similarity, fmt.Sprintf("%s: fuzzy similarity %.2f", ruleName, similarity)
}

// phoneticMatch фонетическое совпадение
func (re *RuleEngine) phoneticMatch(text1, text2, ruleName string) (float64, string) {
	similarity := re.algorithms.phonetic.Similarity(text1, text2)
	return similarity, fmt.Sprintf("%s: phonetic similarity %.2f", ruleName, similarity)
}

// ngramMatch совпадение на основе N-грамм
func (re *RuleEngine) ngramMatch(text1, text2, ruleName string) (float64, string) {
	similarity := re.algorithms.ngram.Similarity(text1, text2)
	return similarity, fmt.Sprintf("%s: ngram similarity %.2f", ruleName, similarity)
}

// combinedMatch комбинированное совпадение
func (re *RuleEngine) combinedMatch(text1, text2, ruleName string) (float64, string) {
	similarity := re.algorithms.similarity.CombinedSimilarity(text1, text2)
	return similarity, fmt.Sprintf("%s: combined similarity %.2f", ruleName, similarity)
}

// defaultMatch правила по умолчанию (если нет набора правил)
func (re *RuleEngine) defaultMatch(record1, record2 map[string]string) (float64, string, bool) {
	// Проверяем основные поля
	name1, ok1 := record1["name"]
	name2, ok2 := record2["name"]

	if !ok1 || !ok2 {
		return 0.0, "missing name field", false
	}

	similarity := re.algorithms.similarity.CombinedSimilarity(
		strings.ToLower(strings.TrimSpace(name1)),
		strings.ToLower(strings.TrimSpace(name2)),
	)

	isDuplicate := similarity >= 0.85
	reason := fmt.Sprintf("default: combined similarity %.2f", similarity)

	return similarity, reason, isDuplicate
}

// CreateDefaultRuleSet создает набор правил по умолчанию для справочника
func CreateDefaultRuleSet(referenceID, name string) *RuleSet {
	return &RuleSet{
		ID:          fmt.Sprintf("default_%s", referenceID),
		Name:        name,
		ReferenceID: referenceID,
		Priority:    1,
		Enabled:     true,
		Rules: []MatchingRule{
			{
				ID:          "exact_code",
				Name:        "Точное совпадение кода",
				Description: "Точное совпадение по полю code",
				Fields:      []string{"code"},
				Algorithm:   "exact",
				Threshold:   1.0,
				Weight:      1.0,
				Enabled:     true,
			},
			{
				ID:          "fuzzy_name",
				Name:        "Нечеткое совпадение имени",
				Description: "Нечеткое совпадение по полю name",
				Fields:      []string{"name"},
				Algorithm:   "fuzzy",
				Threshold:   0.85,
				Weight:      0.8,
				Enabled:     true,
			},
			{
				ID:          "combined_fields",
				Name:        "Комбинированное совпадение",
				Description: "Комбинированное сравнение по нескольким полям",
				Fields:      []string{"name", "category"},
				Algorithm:   "combined",
				Threshold:   0.80,
				Weight:      0.6,
				Enabled:     true,
			},
		},
	}
}

// CreateNomenclatureRuleSet создает специализированный набор правил для номенклатуры
func CreateNomenclatureRuleSet(referenceID string) *RuleSet {
	return &RuleSet{
		ID:          fmt.Sprintf("nomenclature_%s", referenceID),
		Name:        "Правила для номенклатуры",
		ReferenceID: referenceID,
		Priority:    2,
		Enabled:     true,
		Rules: []MatchingRule{
			{
				ID:          "exact_code",
				Name:        "Точное совпадение кода",
				Description: "Точное совпадение по коду номенклатуры",
				Fields:      []string{"code"},
				Algorithm:   "exact",
				Threshold:   1.0,
				Weight:      1.0,
				Enabled:     true,
			},
			{
				ID:          "normalized_name",
				Name:        "Нормализованное имя",
				Description: "Сравнение нормализованных имен",
				Fields:      []string{"normalized_name"},
				Algorithm:   "ngram",
				Threshold:   0.90,
				Weight:      0.9,
				Enabled:     true,
			},
			{
				ID:          "phonetic_name",
				Name:        "Фонетическое совпадение",
				Description: "Фонетическое сравнение для обнаружения опечаток",
				Fields:      []string{"name"},
				Algorithm:   "phonetic",
				Threshold:   0.85,
				Weight:      0.7,
				Enabled:     true,
			},
		},
	}
}

