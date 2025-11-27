package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BottleneckAnalysis анализ узких мест
type BottleneckAnalysis struct {
	Stage            string   `json:"stage"`
	Duration         int64    `json:"duration_ms"`
	Percentage       float64  `json:"percentage"`
	RecordsPerSecond float64  `json:"records_per_second"`
	MemoryUsedMB     float64  `json:"memory_used_mb"`
	Recommendations  []string `json:"recommendations"`
	Severity         string   `json:"severity"` // "critical", "high", "medium", "low"
}

// handleNormalizationBenchmarkAnalyze анализирует узкие места в бенчмарке
func (s *Server) handleNormalizationBenchmarkAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем JSON из тела запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка чтения тела запроса: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Парсим JSON
	var report NormalizationBenchmarkReport
	if err := json.Unmarshal(body, &report); err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка парсинга JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Анализируем узкие места
	bottlenecks := analyzeBottlenecks(&report)

	// Возвращаем результат
	s.writeJSONResponse(w, r, map[string]interface{}{
		"bottlenecks": bottlenecks,
		"summary": map[string]interface{}{
			"total_stages":         len(bottlenecks),
			"critical_bottlenecks": countBySeverity(bottlenecks, "critical"),
			"high_bottlenecks":     countBySeverity(bottlenecks, "high"),
			"medium_bottlenecks":   countBySeverity(bottlenecks, "medium"),
			"low_bottlenecks":      countBySeverity(bottlenecks, "low"),
		},
	}, http.StatusOK)
}

// analyzeBottlenecks анализирует узкие места в отчете
func analyzeBottlenecks(report *NormalizationBenchmarkReport) []BottleneckAnalysis {
	analyses := make([]BottleneckAnalysis, 0, len(report.Results))

	// Вычисляем общее время
	totalDuration := int64(0)
	for _, r := range report.Results {
		totalDuration += r.Duration
	}

	if totalDuration == 0 {
		return analyses
	}

	// Анализируем каждый этап
	for _, result := range report.Results {
		percentage := (float64(result.Duration) / float64(totalDuration)) * 100

		analysis := BottleneckAnalysis{
			Stage:            result.Stage,
			Duration:         result.Duration,
			Percentage:       percentage,
			RecordsPerSecond: result.RecordsPerSecond,
			MemoryUsedMB:     result.MemoryUsedMB,
			Recommendations:  make([]string, 0),
		}

		// Определяем серьезность
		if percentage > 50 {
			analysis.Severity = "critical"
		} else if percentage > 30 {
			analysis.Severity = "high"
		} else if percentage > 15 {
			analysis.Severity = "medium"
		} else {
			analysis.Severity = "low"
		}

		// Генерируем рекомендации
		if result.RecordsPerSecond < 50 {
			analysis.Recommendations = append(analysis.Recommendations,
				"Низкая скорость обработки - рассмотрите оптимизацию алгоритма")
		}

		if result.MemoryUsedMB > 500 {
			analysis.Recommendations = append(analysis.Recommendations,
				"Высокое использование памяти - проверьте утечки памяти")
		}

		if percentage > 40 {
			analysis.Recommendations = append(analysis.Recommendations,
				"Этап занимает большую часть времени - приоритетная цель для оптимизации")
		}

		if result.ErrorCount > 0 {
			analysis.Recommendations = append(analysis.Recommendations,
				fmt.Sprintf("Обнаружено %d ошибок - требуется исправление", result.ErrorCount))
		}

		if result.Stage == "Full Normalization" && result.RecordsPerSecond < 100 {
			analysis.Recommendations = append(analysis.Recommendations,
				"Рассмотрите увеличение количества воркеров для параллельной обработки")
		}

		if result.Stage == "Duplicate Detection" && result.DuplicateGroups > 0 {
			duplicateRate := float64(result.TotalDuplicates) / float64(result.RecordCount) * 100
			if duplicateRate > 30 {
				analysis.Recommendations = append(analysis.Recommendations,
					"Высокий процент дубликатов - рассмотрите предварительную фильтрацию")
			}
		}

		analyses = append(analyses, analysis)
	}

	// Сортируем по проценту времени (самые медленные первыми)
	sort.Slice(analyses, func(i, j int) bool {
		return analyses[i].Percentage > analyses[j].Percentage
	})

	return analyses
}

func countBySeverity(bottlenecks []BottleneckAnalysis, severity string) int {
	count := 0
	for _, b := range bottlenecks {
		if b.Severity == severity {
			count++
		}
	}
	return count
}

// ComparisonResult результат сравнения двух бенчмарков
type ComparisonResult struct {
	Stage          string                        `json:"stage"`
	Baseline       NormalizationBenchmarkResult  `json:"baseline"`
	Current        NormalizationBenchmarkResult  `json:"current"`
	SpeedChange    float64                       `json:"speed_change_percent"`
	DurationChange float64                       `json:"duration_change_percent"`
	MemoryChange   float64                       `json:"memory_change_percent"`
	Improvement    bool                          `json:"improvement"`
}

// handleNormalizationBenchmarkCompare сравнивает два бенчмарка
func (s *Server) handleNormalizationBenchmarkCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Читаем JSON из тела запроса
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка чтения тела запроса: %v", err), http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Парсим JSON
	var request struct {
		BaselineID string `json:"baseline_id"`
		CurrentID  string `json:"current_id"`
		Baseline   *NormalizationBenchmarkReport `json:"baseline,omitempty"`
		Current    *NormalizationBenchmarkReport `json:"current,omitempty"`
	}

	if err := json.Unmarshal(body, &request); err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка парсинга JSON: %v", err), http.StatusBadRequest)
		return
	}

	var baseline, current *NormalizationBenchmarkReport

	// Загружаем бенчмарки по ID, если указаны
	if request.BaselineID != "" {
		baseline, err = s.loadBenchmarkByID(request.BaselineID)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Ошибка загрузки базового бенчмарка: %v", err), http.StatusNotFound)
			return
		}
	} else if request.Baseline != nil {
		baseline = request.Baseline
	} else {
		s.writeJSONError(w, r, "Не указан базовый бенчмарк", http.StatusBadRequest)
		return
	}

	if request.CurrentID != "" {
		current, err = s.loadBenchmarkByID(request.CurrentID)
		if err != nil {
			s.writeJSONError(w, r, fmt.Sprintf("Ошибка загрузки текущего бенчмарка: %v", err), http.StatusNotFound)
			return
		}
	} else if request.Current != nil {
		current = request.Current
	} else {
		s.writeJSONError(w, r, "Не указан текущий бенчмарк", http.StatusBadRequest)
		return
	}

	// Сравниваем
	comparisons := compareBenchmarks(baseline, current)

	// Вычисляем общие изменения
	var speedChange, durationChange float64
	if baseline.AverageSpeed > 0 {
		speedChange = ((current.AverageSpeed - baseline.AverageSpeed) / baseline.AverageSpeed) * 100
	}
	if baseline.TotalDuration > 0 {
		durationChange = ((float64(current.TotalDuration) - float64(baseline.TotalDuration)) / float64(baseline.TotalDuration)) * 100
	}

	// Возвращаем результат
	s.writeJSONResponse(w, r, map[string]interface{}{
		"baseline": map[string]interface{}{
			"timestamp":      baseline.Timestamp,
			"test_name":      baseline.TestName,
			"record_count":   baseline.RecordCount,
			"duplicate_rate": baseline.DuplicateRate,
			"workers":        baseline.Workers,
			"average_speed":  baseline.AverageSpeed,
			"total_duration": baseline.TotalDuration,
		},
		"current": map[string]interface{}{
			"timestamp":      current.Timestamp,
			"test_name":      current.TestName,
			"record_count":   current.RecordCount,
			"duplicate_rate": current.DuplicateRate,
			"workers":        current.Workers,
			"average_speed":  current.AverageSpeed,
			"total_duration": current.TotalDuration,
		},
		"comparisons": comparisons,
		"summary": map[string]interface{}{
			"speed_change":    speedChange,
			"duration_change": durationChange,
			"improvements":    countImprovements(comparisons),
			"regressions":     countRegressions(comparisons),
			"no_changes":      len(comparisons) - countImprovements(comparisons) - countRegressions(comparisons),
		},
	}, http.StatusOK)
}

// loadBenchmarkByID загружает бенчмарк по ID
func (s *Server) loadBenchmarkByID(id string) (*NormalizationBenchmarkReport, error) {
	benchmarksDir := "./benchmarks"
	files, err := os.ReadDir(benchmarksDir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		if strings.Contains(file.Name(), id) {
			filePath := filepath.Join(benchmarksDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, err
			}

			var report NormalizationBenchmarkReport
			if err := json.Unmarshal(data, &report); err != nil {
				return nil, err
			}

			return &report, nil
		}
	}

	return nil, fmt.Errorf("benchmark not found: %s", id)
}

// compareBenchmarks сравнивает два бенчмарка
func compareBenchmarks(baseline, current *NormalizationBenchmarkReport) []ComparisonResult {
	comparisons := make([]ComparisonResult, 0)

	// Создаем карту результатов по этапам
	baselineMap := make(map[string]NormalizationBenchmarkResult)
	for _, r := range baseline.Results {
		baselineMap[r.Stage] = r
	}

	currentMap := make(map[string]NormalizationBenchmarkResult)
	for _, r := range current.Results {
		currentMap[r.Stage] = r
	}

	// Сравниваем общие метрики
	allStages := make(map[string]bool)
	for stage := range baselineMap {
		allStages[stage] = true
	}
	for stage := range currentMap {
		allStages[stage] = true
	}

	for stage := range allStages {
		baselineResult, hasBaseline := baselineMap[stage]
		currentResult, hasCurrent := currentMap[stage]

		if !hasBaseline || !hasCurrent {
			continue
		}

		comparison := ComparisonResult{
			Stage:    stage,
			Baseline: baselineResult,
			Current:  currentResult,
		}

		// Вычисляем изменения
		if baselineResult.RecordsPerSecond > 0 {
			comparison.SpeedChange = ((currentResult.RecordsPerSecond - baselineResult.RecordsPerSecond) / baselineResult.RecordsPerSecond) * 100
		}

		if baselineResult.Duration > 0 {
			comparison.DurationChange = ((float64(currentResult.Duration) - float64(baselineResult.Duration)) / float64(baselineResult.Duration)) * 100
		}

		if baselineResult.MemoryUsedMB > 0 {
			comparison.MemoryChange = ((currentResult.MemoryUsedMB - baselineResult.MemoryUsedMB) / baselineResult.MemoryUsedMB) * 100
		}

		// Улучшение = больше скорость или меньше время
		comparison.Improvement = comparison.SpeedChange > 0 || comparison.DurationChange < 0

		comparisons = append(comparisons, comparison)
	}

	// Сортируем по этапам
	sort.Slice(comparisons, func(i, j int) bool {
		return comparisons[i].Stage < comparisons[j].Stage
	})

	return comparisons
}

func countImprovements(comparisons []ComparisonResult) int {
	count := 0
	for _, c := range comparisons {
		if c.Improvement {
			count++
		}
	}
	return count
}

func countRegressions(comparisons []ComparisonResult) int {
	count := 0
	for _, c := range comparisons {
		if !c.Improvement && (c.SpeedChange < -5 || c.DurationChange > 5) {
			count++
		}
	}
	return count
}

