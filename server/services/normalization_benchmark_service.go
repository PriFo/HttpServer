package services

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	apperrors "httpserver/server/errors"
)

// NormalizationBenchmarkReport отчет о бенчмарке нормализации
type NormalizationBenchmarkReport struct {
	Timestamp   string                 `json:"timestamp"`
	TotalItems  int                    `json:"total_items"`
	Results     map[string]interface{} `json:"results"`
	Metrics     map[string]interface{} `json:"metrics"`
	Config      map[string]interface{} `json:"config"`
	Environment map[string]interface{} `json:"environment"`
}

// NormalizationBenchmarkService сервис для работы с бенчмарками нормализации
type NormalizationBenchmarkService struct {
	benchmarksDir string
}

// NewNormalizationBenchmarkService создает новый сервис для работы с бенчмарками нормализации
func NewNormalizationBenchmarkService() *NormalizationBenchmarkService {
	benchmarksDir := filepath.Join(".", "benchmarks")
	return &NormalizationBenchmarkService{
		benchmarksDir: benchmarksDir,
	}
}

// UploadBenchmark загружает и сохраняет бенчмарк
func (s *NormalizationBenchmarkService) UploadBenchmark(report NormalizationBenchmarkReport) (map[string]interface{}, error) {
	// Создаем директорию, если её нет
	if err := os.MkdirAll(s.benchmarksDir, 0755); err != nil {
		return nil, apperrors.NewInternalError("не удалось создать директорию бенчмарков", err)
	}

	// Генерируем имя файла на основе timestamp
	timestamp := report.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format("20060102_150405")
	}

	// Очищаем timestamp для использования в имени файла
	cleanTimestamp := strings.ReplaceAll(timestamp, ":", "")
	cleanTimestamp = strings.ReplaceAll(cleanTimestamp, "-", "")
	cleanTimestamp = strings.ReplaceAll(cleanTimestamp, "T", "_")
	cleanTimestamp = strings.ReplaceAll(cleanTimestamp, "Z", "")

	filename := "benchmark_" + cleanTimestamp + ".json"
	filePath := filepath.Join(s.benchmarksDir, filename)

	// Сохраняем отчет в JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось сериализовать отчет бенчмарка", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, apperrors.NewInternalError("не удалось записать файл бенчмарка", err)
	}

	return map[string]interface{}{
		"success":    true,
		"message":    "Benchmark uploaded successfully",
		"id":         cleanTimestamp,
		"file_path":  filePath,
		"timestamp":  timestamp,
		"total_items": report.TotalItems,
	}, nil
}

// ListBenchmarks возвращает список доступных бенчмарков
func (s *NormalizationBenchmarkService) ListBenchmarks() (map[string]interface{}, error) {
	// Проверяем существование директории
	if _, err := os.Stat(s.benchmarksDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return map[string]interface{}{
				"benchmarks": []interface{}{},
				"total":      0,
			}, nil
		}
		// Другие ошибки - возвращаем пустой список с ошибкой
		return map[string]interface{}{
			"benchmarks": []interface{}{},
			"total":      0,
		}, nil
	}

	// Читаем файлы
	files, err := os.ReadDir(s.benchmarksDir)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось прочитать директорию бенчмарков", err)
	}

	benchmarks := make([]map[string]interface{}, 0)
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(s.benchmarksDir, file.Name())
		fileInfo, err := file.Info()
		if err != nil {
			continue
		}

		// Читаем файл для получения метаданных
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var report NormalizationBenchmarkReport
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}

		// Извлекаем ID из имени файла
		id := strings.TrimPrefix(file.Name(), "benchmark_")
		id = strings.TrimSuffix(id, ".json")

		benchmark := map[string]interface{}{
			"id":          id,
			"timestamp":   report.Timestamp,
			"total_items":  report.TotalItems,
			"file_name":    file.Name(),
			"file_size":    fileInfo.Size(),
			"created_at":   fileInfo.ModTime().Format(time.RFC3339),
		}

		// Добавляем метрики, если они есть
		if report.Metrics != nil {
			benchmark["metrics"] = report.Metrics
		}

		benchmarks = append(benchmarks, benchmark)
	}

	return map[string]interface{}{
		"benchmarks": benchmarks,
		"total":      len(benchmarks),
	}, nil
}

// GetBenchmark возвращает конкретный бенчмарк по ID
func (s *NormalizationBenchmarkService) GetBenchmark(id string) (*NormalizationBenchmarkReport, error) {
	// Ищем файл по ID (timestamp)
	files, err := os.ReadDir(s.benchmarksDir)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось прочитать директорию бенчмарков", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Проверяем, содержит ли имя файла ID
		if strings.Contains(file.Name(), id) {
			filePath := filepath.Join(s.benchmarksDir, file.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				return nil, apperrors.NewInternalError("не удалось прочитать файл бенчмарка", err)
			}

			var report NormalizationBenchmarkReport
			if err := json.Unmarshal(data, &report); err != nil {
				return nil, apperrors.NewInternalError("не удалось распарсить файл бенчмарка", err)
			}

			return &report, nil
		}
	}

	return nil, apperrors.NewNotFoundError("бенчмарк не найден", nil)
}

