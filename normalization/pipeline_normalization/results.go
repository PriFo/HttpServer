package pipeline_normalization

// SimilarityScore оценка схожести с детализацией по алгоритмам
type SimilarityScore struct {
	OverallSimilarity float64            `json:"overall_similarity"` // Общая схожесть (0.0 - 1.0)
	AlgorithmScores   map[string]float64 `json:"algorithm_scores"`  // Схожесть по каждому алгоритму
	IsDuplicate       bool               `json:"is_duplicate"`       // Являются ли строки дубликатами
	Confidence        float64            `json:"confidence"`         // Уверенность в результате (0.0 - 1.0)
}

// NormalizationResult результат нормализации двух строк
type NormalizationResult struct {
	Text1            string         `json:"text1"`             // Первая строка
	Text2            string         `json:"text2"`             // Вторая строка
	Similarity       SimilarityScore `json:"similarity"`       // Оценка схожести
	NormalizedText1  string         `json:"normalized_text1"`  // Нормализованная первая строка
	NormalizedText2  string         `json:"normalized_text2"`  // Нормализованная вторая строка
	ProcessingTime   int64          `json:"processing_time"`  // Время обработки в наносекундах
	AlgorithmsUsed   []string       `json:"algorithms_used"`  // Список использованных алгоритмов
}

// QualityMetrics метрики качества алгоритмов
type QualityMetrics struct {
	Precision float64 `json:"precision"` // Точность (доля найденных дублей, которые действительно дубли)
	Recall    float64 `json:"recall"`    // Полнота (доля всех дублей, которые были найдены)
	FMeasure  float64 `json:"f_measure"` // F-мера (гармоническое среднее точности и полноты)
	
	// Детализация по алгоритмам
	AlgorithmMetrics map[string]AlgorithmQualityMetrics `json:"algorithm_metrics"`
}

// AlgorithmQualityMetrics метрики качества для отдельного алгоритма
type AlgorithmQualityMetrics struct {
	Precision     float64 `json:"precision"`
	Recall        float64 `json:"recall"`
	FMeasure      float64 `json:"f_measure"`
	TruePositives int     `json:"true_positives"`  // Правильно найденные дубли
	FalsePositives int    `json:"false_positives"` // Ложные срабатывания
	FalseNegatives int    `json:"false_negatives"` // Пропущенные дубли
	TrueNegatives  int    `json:"true_negatives"`  // Правильно определенные как не дубли
}

// BatchResult результат обработки батча строк
type BatchResult struct {
	Results        []NormalizationResult `json:"results"`         // Результаты для каждой пары
	TotalProcessed int                   `json:"total_processed"` // Всего обработано пар
	DuplicatesFound int                  `json:"duplicates_found"` // Найдено дубликатов
	ProcessingTime int64                 `json:"processing_time"`  // Общее время обработки в наносекундах
	AverageSimilarity float64            `json:"average_similarity"` // Средняя схожесть
	QualityMetrics *QualityMetrics      `json:"quality_metrics,omitempty"` // Метрики качества (если вычислялись)
}

// NewSimilarityScore создает новую оценку схожести
func NewSimilarityScore() *SimilarityScore {
	return &SimilarityScore{
		OverallSimilarity: 0.0,
		AlgorithmScores:   make(map[string]float64),
		IsDuplicate:       false,
		Confidence:        0.0,
	}
}

// AddAlgorithmScore добавляет оценку от алгоритма
func (s *SimilarityScore) AddAlgorithmScore(algorithmName string, score float64) {
	s.AlgorithmScores[algorithmName] = score
}

// CalculateOverall вычисляет общую схожесть на основе оценок алгоритмов
func (s *SimilarityScore) CalculateOverall(weights map[string]float64, combineMethod string, threshold float64) {
	if len(s.AlgorithmScores) == 0 {
		s.OverallSimilarity = 0.0
		s.IsDuplicate = false
		return
	}

	switch combineMethod {
	case "weighted":
		s.calculateWeighted(weights)
	case "max":
		s.calculateMax()
	case "min":
		s.calculateMin()
	case "average":
		s.calculateAverage()
	default:
		s.calculateWeighted(weights)
	}

	// Определяем, являются ли строки дубликатами
	s.IsDuplicate = s.OverallSimilarity >= threshold

	// Вычисляем уверенность на основе разброса оценок
	s.Confidence = s.calculateConfidence()
}

// calculateWeighted вычисляет взвешенную схожесть
func (s *SimilarityScore) calculateWeighted(weights map[string]float64) {
	totalWeight := 0.0
	weightedSum := 0.0

	for algName, score := range s.AlgorithmScores {
		weight := weights[algName]
		if weight > 0 {
			weightedSum += score * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		s.OverallSimilarity = weightedSum / totalWeight
	} else {
		s.calculateAverage()
	}
}

// calculateMax вычисляет максимальную схожесть
func (s *SimilarityScore) calculateMax() {
	maxScore := 0.0
	for _, score := range s.AlgorithmScores {
		if score > maxScore {
			maxScore = score
		}
	}
	s.OverallSimilarity = maxScore
}

// calculateMin вычисляет минимальную схожесть
func (s *SimilarityScore) calculateMin() {
	minScore := 1.0
	for _, score := range s.AlgorithmScores {
		if score < minScore {
			minScore = score
		}
	}
	s.OverallSimilarity = minScore
}

// calculateAverage вычисляет среднюю схожесть
func (s *SimilarityScore) calculateAverage() {
	sum := 0.0
	count := 0
	for _, score := range s.AlgorithmScores {
		sum += score
		count++
	}
	if count > 0 {
		s.OverallSimilarity = sum / float64(count)
	}
}

// calculateConfidence вычисляет уверенность на основе согласованности оценок
func (s *SimilarityScore) calculateConfidence() float64 {
	if len(s.AlgorithmScores) == 0 {
		return 0.0
	}

	// Вычисляем стандартное отклонение оценок
	scores := make([]float64, 0, len(s.AlgorithmScores))
	for _, score := range s.AlgorithmScores {
		scores = append(scores, score)
	}

	mean := s.OverallSimilarity
	variance := 0.0
	for _, score := range scores {
		diff := score - mean
		variance += diff * diff
	}
	variance /= float64(len(scores))
	stdDev := variance

	// Уверенность обратно пропорциональна стандартному отклонению
	// Чем больше согласованность, тем выше уверенность
	confidence := 1.0 - stdDev
	if confidence < 0.0 {
		confidence = 0.0
	}
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// NewQualityMetrics создает новые метрики качества
func NewQualityMetrics() *QualityMetrics {
	return &QualityMetrics{
		Precision:        0.0,
		Recall:           0.0,
		FMeasure:         0.0,
		AlgorithmMetrics: make(map[string]AlgorithmQualityMetrics),
	}
}

// Calculate вычисляет метрики качества
func (qm *QualityMetrics) Calculate(truePositives, falsePositives, falseNegatives int) {
	// Precision = TP / (TP + FP)
	if truePositives+falsePositives > 0 {
		qm.Precision = float64(truePositives) / float64(truePositives+falsePositives)
	}

	// Recall = TP / (TP + FN)
	if truePositives+falseNegatives > 0 {
		qm.Recall = float64(truePositives) / float64(truePositives+falseNegatives)
	}

	// F-measure = 2 * (Precision * Recall) / (Precision + Recall)
	if qm.Precision+qm.Recall > 0 {
		qm.FMeasure = 2.0 * (qm.Precision * qm.Recall) / (qm.Precision + qm.Recall)
	}
}

// CalculateWeighted вычисляет взвешенную F-меру
func (qm *QualityMetrics) CalculateWeighted(precisionWeight, recallWeight float64) {
	if qm.Precision+qm.Recall > 0 {
		denominator := precisionWeight*qm.Precision + recallWeight*qm.Recall
		if denominator > 0 {
			qm.FMeasure = (1.0 + precisionWeight*recallWeight) * (qm.Precision * qm.Recall) / denominator
		}
	}
}

// AddAlgorithmMetrics добавляет метрики для алгоритма
func (qm *QualityMetrics) AddAlgorithmMetrics(algorithmName string, metrics AlgorithmQualityMetrics) {
	qm.AlgorithmMetrics[algorithmName] = metrics
}

// CalculateAlgorithmMetrics вычисляет метрики для алгоритма
func CalculateAlgorithmMetrics(truePositives, falsePositives, falseNegatives, trueNegatives int) AlgorithmQualityMetrics {
	metrics := AlgorithmQualityMetrics{
		TruePositives:  truePositives,
		FalsePositives: falsePositives,
		FalseNegatives: falseNegatives,
		TrueNegatives:  trueNegatives,
	}

	// Precision
	if truePositives+falsePositives > 0 {
		metrics.Precision = float64(truePositives) / float64(truePositives+falsePositives)
	}

	// Recall
	if truePositives+falseNegatives > 0 {
		metrics.Recall = float64(truePositives) / float64(truePositives+falseNegatives)
	}

	// F-measure
	if metrics.Precision+metrics.Recall > 0 {
		metrics.FMeasure = 2.0 * (metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall)
	}

	return metrics
}

