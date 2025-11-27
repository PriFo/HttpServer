package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// NormalizationBenchmarkReport структура отчета бенчмарка
type NormalizationBenchmarkReport struct {
	Timestamp     string                 `json:"timestamp"`
	TestName      string                 `json:"test_name"`
	RecordCount   int                    `json:"record_count"`
	DuplicateRate float64                `json:"duplicate_rate"`
	Workers       int                    `json:"workers"`
	Results       []NormalizationBenchmarkResult `json:"results"`
	TotalDuration int64                  `json:"total_duration_ms"`
	AverageSpeed  float64                `json:"average_speed_records_per_sec"`
	Summary       map[string]interface{} `json:"summary"`
}

// NormalizationBenchmarkResult результат одного этапа
type NormalizationBenchmarkResult struct {
	Stage             string  `json:"stage"`
	RecordCount       int     `json:"record_count"`
	Duration          int64   `json:"duration_ms"`
	RecordsPerSecond  float64 `json:"records_per_second"`
	MemoryUsedMB      float64 `json:"memory_used_mb,omitempty"`
	DuplicateGroups   int     `json:"duplicate_groups,omitempty"`
	TotalDuplicates   int     `json:"total_duplicates,omitempty"`
	ProcessedCount    int     `json:"processed_count,omitempty"`
	BenchmarkMatches  int     `json:"benchmark_matches,omitempty"`
	EnrichedCount     int     `json:"enriched_count,omitempty"`
	CreatedBenchmarks int     `json:"created_benchmarks,omitempty"`
	ErrorCount        int     `json:"error_count,omitempty"`
	Stopped           bool    `json:"stopped,omitempty"`
}

// handleNormalizationBenchmarkUpload обрабатывает загрузку результатов бенчмарка
func (s *Server) handleNormalizationBenchmarkUpload(w http.ResponseWriter, r *http.Request) {
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

	// Валидация
	if report.Timestamp == "" {
		report.Timestamp = time.Now().Format(time.RFC3339)
	}
	if len(report.Results) == 0 {
		s.writeJSONError(w, r, "Отчет не содержит результатов", http.StatusBadRequest)
		return
	}
	
	// Дополнительная валидация структуры
	if report.RecordCount <= 0 {
		s.writeJSONError(w, r, "Количество записей должно быть больше нуля", http.StatusBadRequest)
		return
	}
	if report.DuplicateRate < 0 || report.DuplicateRate > 1 {
		s.writeJSONError(w, r, "Процент дубликатов должен быть от 0.0 до 1.0", http.StatusBadRequest)
		return
	}
	if report.Workers <= 0 {
		s.writeJSONError(w, r, "Количество воркеров должно быть больше нуля", http.StatusBadRequest)
		return
	}
	
	// Валидация результатов
	for i, result := range report.Results {
		if result.Stage == "" {
			s.writeJSONError(w, r, fmt.Sprintf("Результат %d не содержит названия этапа", i+1), http.StatusBadRequest)
			return
		}
		if result.Duration < 0 {
			s.writeJSONError(w, r, fmt.Sprintf("Длительность этапа '%s' не может быть отрицательной", result.Stage), http.StatusBadRequest)
			return
		}
		if result.RecordsPerSecond < 0 {
			s.writeJSONError(w, r, fmt.Sprintf("Скорость этапа '%s' не может быть отрицательной", result.Stage), http.StatusBadRequest)
			return
		}
	}

	// Сохраняем в файл
	benchmarksDir := filepath.Join(".", "benchmarks")
	if err := os.MkdirAll(benchmarksDir, 0755); err != nil {
		log.Printf("[Benchmark] Ошибка создания директории: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка создания директории: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Проверяем доступность директории для записи
	if info, err := os.Stat(benchmarksDir); err != nil || !info.IsDir() {
		log.Printf("[Benchmark] Директория недоступна: %v", err)
		s.writeJSONError(w, r, "Директория для сохранения бенчмарков недоступна", http.StatusInternalServerError)
		return
	}

	// Генерируем имя файла из timestamp
	timestamp := report.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format("20060102_150405")
	} else {
		// Парсим timestamp и преобразуем в формат имени файла
		if t, err := time.Parse(time.RFC3339, timestamp); err == nil {
			timestamp = t.Format("20060102_150405")
		} else {
			timestamp = time.Now().Format("20060102_150405")
		}
	}

	filename := fmt.Sprintf("normalization_benchmark_%s.json", timestamp)
	filePath := filepath.Join(benchmarksDir, filename)

	// Сохраняем JSON
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка сериализации JSON: %v", err), http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		log.Printf("[Benchmark] Ошибка сохранения файла: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка сохранения файла: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("[Benchmark] Бенчмарк сохранен: %s", filePath)

	// Возвращаем успешный ответ
	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Benchmark uploaded successfully",
		"id":      timestamp,
		"file":    filename,
		"path":    filePath,
	}, http.StatusOK)
}

// handleNormalizationBenchmarkList возвращает список доступных бенчмарков
func (s *Server) handleNormalizationBenchmarkList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	benchmarksDir := filepath.Join(".", "benchmarks")
	
	// Проверяем существование директории
	if _, err := os.Stat(benchmarksDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeJSONResponse(w, r, map[string]interface{}{
				"benchmarks": []interface{}{},
				"total":      0,
			}, http.StatusOK)
			return
		}
		// Другие ошибки - возвращаем пустой список
		s.writeJSONResponse(w, r, map[string]interface{}{
			"benchmarks": []interface{}{},
			"total":      0,
		}, http.StatusOK)
		return
	}

	// Читаем файлы
	files, err := os.ReadDir(benchmarksDir)
	if err != nil {
		log.Printf("[Benchmark] Ошибка чтения директории: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка чтения директории: %v", err), http.StatusInternalServerError)
		return
	}

	benchmarks := make([]map[string]interface{}, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		if !strings.HasPrefix(file.Name(), "normalization_benchmark_") {
			continue
		}

		filePath := filepath.Join(benchmarksDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Printf("[Benchmark] Ошибка чтения файла %s: %v", file.Name(), err)
			continue
		}

		var report NormalizationBenchmarkReport
		if err := json.Unmarshal(data, &report); err != nil {
			log.Printf("[Benchmark] Ошибка парсинга файла %s: %v", file.Name(), err)
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		benchmarks = append(benchmarks, map[string]interface{}{
			"id":            report.Timestamp,
			"filename":      file.Name(),
			"timestamp":    report.Timestamp,
			"test_name":     report.TestName,
			"record_count":  report.RecordCount,
			"duplicate_rate": report.DuplicateRate,
			"workers":       report.Workers,
			"average_speed": report.AverageSpeed,
			"total_duration_ms": report.TotalDuration,
			"file_size":     info.Size(),
			"created_at":    info.ModTime().Format(time.RFC3339),
		})
	}

	// Сортируем по timestamp (новые первыми)
	for i := 0; i < len(benchmarks)-1; i++ {
		for j := i + 1; j < len(benchmarks); j++ {
			ts1, ok1 := benchmarks[i]["timestamp"].(string)
			ts2, ok2 := benchmarks[j]["timestamp"].(string)
			if ok1 && ok2 {
				t1, _ := time.Parse(time.RFC3339, ts1)
				t2, _ := time.Parse(time.RFC3339, ts2)
				if t1.Before(t2) {
					benchmarks[i], benchmarks[j] = benchmarks[j], benchmarks[i]
				}
			}
		}
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"benchmarks": benchmarks,
		"total":      len(benchmarks),
	}, http.StatusOK)
}

// handleNormalizationBenchmarkGet возвращает конкретный бенчмарк по ID
func (s *Server) handleNormalizationBenchmarkGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		s.writeJSONError(w, r, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	var id string
	for i, part := range pathParts {
		if part == "benchmark" && i+1 < len(pathParts) {
			id = pathParts[i+1]
			break
		}
	}

	if id == "" {
		s.writeJSONError(w, r, "Benchmark ID not provided", http.StatusBadRequest)
		return
	}

	benchmarksDir := filepath.Join(".", "benchmarks")
	
	// Ищем файл по ID (timestamp)
	files, err := os.ReadDir(benchmarksDir)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка чтения директории: %v", err), http.StatusInternalServerError)
		return
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Проверяем, содержит ли имя файла ID
		if strings.Contains(file.Name(), id) {
			filePath := filepath.Join(benchmarksDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				s.writeJSONError(w, r, fmt.Sprintf("Ошибка чтения файла: %v", err), http.StatusInternalServerError)
				return
			}

			var report NormalizationBenchmarkReport
			if err := json.Unmarshal(data, &report); err != nil {
				s.writeJSONError(w, r, fmt.Sprintf("Ошибка парсинга файла: %v", err), http.StatusInternalServerError)
				return
			}

			s.writeJSONResponse(w, r, report, http.StatusOK)
			return
		}
	}

	s.writeJSONError(w, r, "Benchmark not found", http.StatusNotFound)
}

