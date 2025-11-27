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

// BottleneckAnalysis анализ узких мест
type BottleneckAnalysis struct {
	Stage              string
	Duration           time.Duration
	Percentage         float64
	RecordsPerSecond   float64
	MemoryUsedMB       float64
	Recommendations    []string
	Severity           string // "critical", "high", "medium", "low"
}

func main() {
	var (
		reportFile = flag.String("report", "", "Путь к JSON отчету бенчмарка")
		outputFile = flag.String("output", "", "Путь к файлу для сохранения анализа (опционально)")
	)
	flag.Parse()

	if *reportFile == "" {
		log.Fatal("Использование: analyze_benchmark.go -report <файл> [-output <файл>]")
	}

	fmt.Println("=== Анализ узких мест в нормализации ===")
	fmt.Printf("Отчет: %s\n", *reportFile)
	fmt.Println()

	// Загружаем отчет
	report, err := loadReport(*reportFile)
	if err != nil {
		log.Fatalf("Ошибка загрузки отчета: %v", err)
	}

	// Анализируем узкие места
	bottlenecks := analyzeBottlenecks(report)

	// Выводим результаты
	printAnalysis(bottlenecks, report)

	// Сохраняем, если указан файл
	if *outputFile != "" {
		saveAnalysis(bottlenecks, report, *outputFile)
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

func analyzeBottlenecks(report *FullBenchmarkReport) []BottleneckAnalysis {
	analyses := make([]BottleneckAnalysis, 0, len(report.Results))

	// Вычисляем общее время
	totalDuration := time.Duration(0)
	for _, r := range report.Results {
		totalDuration += r.Duration
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

func printAnalysis(bottlenecks []BottleneckAnalysis, report *FullBenchmarkReport) {
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println("АНАЛИЗ УЗКИХ МЕСТ")
	fmt.Println("=" + strings.Repeat("=", 100))
	fmt.Println()

	fmt.Printf("Общая статистика:\n")
	fmt.Printf("  Записей: %d\n", report.RecordCount)
	fmt.Printf("  Общее время: %v\n", report.TotalDuration.Round(time.Millisecond))
	fmt.Printf("  Средняя скорость: %.2f записей/сек\n", report.AverageSpeed)
	fmt.Println()

	fmt.Println("Узкие места (отсортированы по времени выполнения):")
	fmt.Println(strings.Repeat("-", 100))
	fmt.Printf("%-30s | %-12s | %-10s | %-12s | %-12s | %-10s\n",
		"Этап", "Время", "% времени", "Скорость", "Память", "Серьезность")
	fmt.Println(strings.Repeat("-", 100))

	for _, b := range bottlenecks {
		severityIcon := "✓"
		if b.Severity == "critical" {
			severityIcon = "🔴"
		} else if b.Severity == "high" {
			severityIcon = "🟠"
		} else if b.Severity == "medium" {
			severityIcon = "🟡"
		}

		fmt.Printf("%-30s | %-12v | %-10.1f%% | %-12.2f | %-12.2f | %-10s\n",
			b.Stage,
			b.Duration.Round(time.Millisecond),
			b.Percentage,
			b.RecordsPerSecond,
			b.MemoryUsedMB,
			severityIcon+" "+b.Severity)
	}

	fmt.Println(strings.Repeat("-", 100))
	fmt.Println()

	// Рекомендации
	fmt.Println("Рекомендации по оптимизации:")
	fmt.Println()

	criticalCount := 0
	for _, b := range bottlenecks {
		if b.Severity == "critical" || b.Severity == "high" {
			criticalCount++
			fmt.Printf("%d. %s (%s)\n", criticalCount, b.Stage, b.Severity)
			fmt.Printf("   Время: %v (%.1f%% от общего времени)\n", b.Duration.Round(time.Millisecond), b.Percentage)
			if len(b.Recommendations) > 0 {
				for _, rec := range b.Recommendations {
					fmt.Printf("   • %s\n", rec)
				}
			}
			fmt.Println()
		}
	}

	if criticalCount == 0 {
		fmt.Println("  ✓ Критических узких мест не обнаружено")
		fmt.Println("  Система работает эффективно")
	}
}

func saveAnalysis(bottlenecks []BottleneckAnalysis, report *FullBenchmarkReport, filename string) {
	analysisReport := map[string]interface{}{
		"timestamp":     time.Now().Format(time.RFC3339),
		"source_report": report.Timestamp,
		"summary": map[string]interface{}{
			"total_stages":      len(bottlenecks),
			"critical_bottlenecks": 0,
			"high_bottlenecks":     0,
			"medium_bottlenecks":   0,
			"low_bottlenecks":      0,
		},
		"bottlenecks": make([]map[string]interface{}, 0, len(bottlenecks)),
	}

	summary := analysisReport["summary"].(map[string]interface{})
	for _, b := range bottlenecks {
		bottleneckData := map[string]interface{}{
			"stage":              b.Stage,
			"duration_ms":        b.Duration.Milliseconds(),
			"percentage":         b.Percentage,
			"records_per_second": b.RecordsPerSecond,
			"memory_used_mb":     b.MemoryUsedMB,
			"severity":           b.Severity,
			"recommendations":    b.Recommendations,
		}
		analysisReport["bottlenecks"] = append(analysisReport["bottlenecks"].([]map[string]interface{}), bottleneckData)

		switch b.Severity {
		case "critical":
			summary["critical_bottlenecks"] = summary["critical_bottlenecks"].(int) + 1
		case "high":
			summary["high_bottlenecks"] = summary["high_bottlenecks"].(int) + 1
		case "medium":
			summary["medium_bottlenecks"] = summary["medium_bottlenecks"].(int) + 1
		case "low":
			summary["low_bottlenecks"] = summary["low_bottlenecks"].(int) + 1
		}
	}

	jsonData, err := json.MarshalIndent(analysisReport, "", "  ")
	if err != nil {
		log.Printf("Ошибка при сериализации JSON: %v", err)
		return
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Printf("Ошибка при сохранении файла: %v", err)
		return
	}

	fmt.Printf("✓ Анализ сохранен в: %s\n", filename)
}

