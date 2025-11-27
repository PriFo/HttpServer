//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"
)

// BenchmarkResult результат бенчмарка для одного этапа
type BenchmarkResult struct {
	Stage             string        `json:"stage"`
	RecordCount       int           `json:"record_count"`
	Duration          time.Duration `json:"duration_ms"`
	RecordsPerSecond  float64       `json:"records_per_second"`
	MemoryUsedMB      float64       `json:"memory_used_mb,omitempty"`
	DuplicateGroups   int           `json:"duplicate_groups,omitempty"`
	TotalDuplicates   int           `json:"total_duplicates,omitempty"`
	BenchmarkMatches  int           `json:"benchmark_matches,omitempty"`
	EnrichedCount     int           `json:"enriched_count,omitempty"`
	CreatedBenchmarks int           `json:"created_benchmarks,omitempty"`
	ProcessedCount    int           `json:"processed_count,omitempty"`
	ErrorCount        int           `json:"error_count,omitempty"`
	Stopped           bool          `json:"stopped,omitempty"`
	StopLatency       time.Duration `json:"stop_latency_ms,omitempty"`
}

// FullBenchmarkReport полный отчет о бенчмарке
type FullBenchmarkReport struct {
	Timestamp     string                 `json:"timestamp"`
	TestName      string                 `json:"test_name"`
	RecordCount   int                    `json:"record_count"`
	DuplicateRate float64                `json:"duplicate_rate"`
	Workers       int                    `json:"workers"`
	Results       []BenchmarkResult      `json:"results"`
	TotalDuration time.Duration          `json:"total_duration_ms"`
	AverageSpeed  float64                `json:"average_speed_records_per_sec"`
	Summary       map[string]interface{} `json:"summary"`
}

// ComparisonResult результат сравнения
type ComparisonResult struct {
	Stage           string
	Baseline        BenchmarkResult
	Current         BenchmarkResult
	SpeedChange     float64 // Процент изменения скорости
	DurationChange  float64 // Процент изменения времени
	MemoryChange    float64 // Процент изменения памяти
	Improvement     bool    // Улучшение или ухудшение
}

func main() {
	var (
		baselineFile = flag.String("baseline", "", "Путь к базовому JSON отчету")
		currentFile  = flag.String("current", "", "Путь к текущему JSON отчету")
		outputFile   = flag.String("output", "", "Путь к файлу для сохранения сравнения (опционально)")
	)
	flag.Parse()

	if *baselineFile == "" || *currentFile == "" {
		log.Fatal("Использование: compare_benchmark_results.go -baseline <файл> -current <файл> [-output <файл>]")
	}

	fmt.Println("=== Сравнение результатов бенчмарков ===")
	fmt.Printf("Базовый отчет: %s\n", *baselineFile)
	fmt.Printf("Текущий отчет: %s\n", *currentFile)
	fmt.Println()

	// Загружаем отчеты
	baseline, err := loadReport(*baselineFile)
	if err != nil {
		log.Fatalf("Ошибка загрузки базового отчета: %v", err)
	}

	current, err := loadReport(*currentFile)
	if err != nil {
		log.Fatalf("Ошибка загрузки текущего отчета: %v", err)
	}

	// Сравниваем
	comparisons := compareReports(baseline, current)

	// Выводим результаты
	printComparison(comparisons, baseline, current)

	// Сохраняем, если указан файл
	if *outputFile != "" {
		saveComparison(comparisons, baseline, current, *outputFile)
	}
}

// FullBenchmarkReportJSON структура для JSON десериализации
type FullBenchmarkReportJSON struct {
	Timestamp     string                 `json:"timestamp"`
	TestName      string                 `json:"test_name"`
	RecordCount   int                    `json:"record_count"`
	DuplicateRate float64                `json:"duplicate_rate"`
	Workers       int                    `json:"workers"`
	Results       []BenchmarkResultJSON  `json:"results"`
	TotalDuration int64                  `json:"total_duration_ms"`
	AverageSpeed  float64                `json:"average_speed_records_per_sec"`
	Summary       map[string]interface{} `json:"summary"`
}

// BenchmarkResultJSON структура для JSON десериализации результата
type BenchmarkResultJSON struct {
	Stage             string  `json:"stage"`
	RecordCount       int     `json:"record_count"`
	Duration          int64   `json:"duration_ms"`
	RecordsPerSecond  float64 `json:"records_per_second"`
	MemoryUsedMB      float64 `json:"memory_used_mb,omitempty"`
	DuplicateGroups   int     `json:"duplicate_groups,omitempty"`
	TotalDuplicates   int     `json:"total_duplicates,omitempty"`
	BenchmarkMatches  int     `json:"benchmark_matches,omitempty"`
	EnrichedCount     int     `json:"enriched_count,omitempty"`
	CreatedBenchmarks int     `json:"created_benchmarks,omitempty"`
	ProcessedCount    int     `json:"processed_count,omitempty"`
	ErrorCount        int     `json:"error_count,omitempty"`
	Stopped           bool    `json:"stopped,omitempty"`
	StopLatency       int64   `json:"stop_latency_ms,omitempty"`
}

func (r *FullBenchmarkReportJSON) ToFullBenchmarkReport() *FullBenchmarkReport {
	results := make([]BenchmarkResult, len(r.Results))
	for i, res := range r.Results {
		results[i] = BenchmarkResult{
			Stage:             res.Stage,
			RecordCount:       res.RecordCount,
			Duration:          time.Duration(res.Duration) * time.Millisecond,
			RecordsPerSecond:  res.RecordsPerSecond,
			MemoryUsedMB:      res.MemoryUsedMB,
			DuplicateGroups:   res.DuplicateGroups,
			TotalDuplicates:   res.TotalDuplicates,
			BenchmarkMatches:  res.BenchmarkMatches,
			EnrichedCount:     res.EnrichedCount,
			CreatedBenchmarks: res.CreatedBenchmarks,
			ProcessedCount:    res.ProcessedCount,
			ErrorCount:        res.ErrorCount,
			Stopped:           res.Stopped,
			StopLatency:       time.Duration(res.StopLatency) * time.Millisecond,
		}
	}

	return &FullBenchmarkReport{
		Timestamp:     r.Timestamp,
		TestName:      r.TestName,
		RecordCount:   r.RecordCount,
		DuplicateRate: r.DuplicateRate,
		Workers:       r.Workers,
		Results:       results,
		TotalDuration: time.Duration(r.TotalDuration) * time.Millisecond,
		AverageSpeed:  r.AverageSpeed,
		Summary:       r.Summary,
	}
}

func loadReport(filename string) (*FullBenchmarkReport, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var reportJSON FullBenchmarkReportJSON
	if err := json.Unmarshal(data, &reportJSON); err != nil {
		return nil, err
	}

	return reportJSON.ToFullBenchmarkReport(), nil
}

func compareReports(baseline, current *FullBenchmarkReport) []ComparisonResult {
	comparisons := make([]ComparisonResult, 0)

	// Создаем карту результатов по этапам
	baselineMap := make(map[string]BenchmarkResult)
	for _, r := range baseline.Results {
		baselineMap[r.Stage] = r
	}

	currentMap := make(map[string]BenchmarkResult)
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

func printComparison(comparisons []ComparisonResult, baseline, current *FullBenchmarkReport) {
	fmt.Println("=" + strings.Repeat("=", 120))
	fmt.Println("СРАВНЕНИЕ РЕЗУЛЬТАТОВ")
	fmt.Println("=" + strings.Repeat("=", 120))
	fmt.Println()

	// Общая статистика
	fmt.Printf("Базовый отчет: %s (%d записей, %.1f%% дубликатов)\n",
		baseline.Timestamp, baseline.RecordCount, baseline.DuplicateRate*100)
	fmt.Printf("Текущий отчет: %s (%d записей, %.1f%% дубликатов)\n",
		current.Timestamp, current.RecordCount, current.DuplicateRate*100)
	fmt.Println()

	// Изменение общей скорости
	if baseline.AverageSpeed > 0 {
		speedChange := ((current.AverageSpeed - baseline.AverageSpeed) / baseline.AverageSpeed) * 100
		fmt.Printf("Общая скорость: %.2f -> %.2f записей/сек (изменение: %+.2f%%)\n",
			baseline.AverageSpeed, current.AverageSpeed, speedChange)
		if speedChange > 0 {
			fmt.Printf("  ✓ Улучшение на %.2f%%\n", speedChange)
		} else if speedChange < 0 {
			fmt.Printf("  ✗ Ухудшение на %.2f%%\n", -speedChange)
		}
		fmt.Println()
	}

	// Детальное сравнение по этапам
	fmt.Println("Детальное сравнение по этапам:")
	fmt.Println(strings.Repeat("-", 120))
	fmt.Printf("%-30s | %-12s | %-12s | %-12s | %-12s\n",
		"Этап", "Скорость", "Время", "Память", "Статус")
	fmt.Println(strings.Repeat("-", 120))

	for _, comp := range comparisons {
		status := "➡ Без изменений"
		if comp.Improvement {
			status = "✓ Улучшение"
		} else if comp.SpeedChange < 0 || comp.DurationChange > 0 {
			status = "✗ Ухудшение"
		}

		fmt.Printf("%-30s | %+8.2f%% | %+8.2f%% | %+8.2f%% | %s\n",
			comp.Stage,
			comp.SpeedChange,
			comp.DurationChange,
			comp.MemoryChange,
			status)
	}

	fmt.Println(strings.Repeat("-", 120))
	fmt.Println()

	// Рекомендации
	fmt.Println("Рекомендации:")
	improvements := 0
	regressions := 0
	for _, comp := range comparisons {
		if comp.Improvement {
			improvements++
		} else if comp.SpeedChange < -5 || comp.DurationChange > 5 {
			regressions++
		}
	}

	if improvements > 0 {
		fmt.Printf("  ✓ Найдено %d улучшений\n", improvements)
	}
	if regressions > 0 {
		fmt.Printf("  ⚠ Найдено %d ухудшений (требуется внимание)\n", regressions)
	}
	if improvements == 0 && regressions == 0 {
		fmt.Println("  ➡ Значительных изменений не обнаружено")
	}
}

func saveComparison(comparisons []ComparisonResult, baseline, current *FullBenchmarkReport, filename string) {
	report := map[string]interface{}{
		"baseline": map[string]interface{}{
			"timestamp":     baseline.Timestamp,
			"record_count":  baseline.RecordCount,
			"duplicate_rate": baseline.DuplicateRate,
			"average_speed": baseline.AverageSpeed,
		},
		"current": map[string]interface{}{
			"timestamp":     current.Timestamp,
			"record_count":  current.RecordCount,
			"duplicate_rate": current.DuplicateRate,
			"average_speed": current.AverageSpeed,
		},
		"comparisons": comparisons,
		"summary": map[string]interface{}{
			"total_stages":    len(comparisons),
			"improvements":    0,
			"regressions":     0,
			"no_changes":      0,
		},
	}

	// Подсчитываем статистику
	summary := report["summary"].(map[string]interface{})
	improvements := 0
	regressions := 0
	noChanges := 0
	for _, comp := range comparisons {
		if comp.Improvement {
			improvements++
		} else if comp.SpeedChange < -5 || comp.DurationChange > 5 {
			regressions++
		} else {
			noChanges++
		}
	}
	summary["improvements"] = improvements
	summary["regressions"] = regressions
	summary["no_changes"] = noChanges

	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("Ошибка при сериализации JSON: %v", err)
		return
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Printf("Ошибка при сохранении файла: %v", err)
		return
	}

	fmt.Printf("✓ Сравнение сохранено в: %s\n", filename)
}

