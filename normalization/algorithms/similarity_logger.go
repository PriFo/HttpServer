package algorithms

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel уровень логирования
type LogLevel int

const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// SimilarityLogger логгер для системы схожести
type SimilarityLogger struct {
	logger   *log.Logger
	file     *os.File
	level    LogLevel
	mu       sync.RWMutex
	enabled  bool
	logFile  string
}

var (
	globalLogger *SimilarityLogger
	loggerOnce   sync.Once
)

// GetLogger возвращает глобальный логгер
func GetLogger() *SimilarityLogger {
	loggerOnce.Do(func() {
		globalLogger = NewSimilarityLogger("similarity.log", LogLevelInfo, true)
	})
	return globalLogger
}

// NewSimilarityLogger создает новый логгер
func NewSimilarityLogger(logFile string, level LogLevel, enabled bool) *SimilarityLogger {
	sl := &SimilarityLogger{
		level:   level,
		enabled: enabled,
		logFile: logFile,
	}

	if enabled && logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			sl.file = file
			sl.logger = log.New(file, "", log.LstdFlags)
		} else {
			// Fallback to stdout
			sl.logger = log.New(os.Stdout, "[SIMILARITY] ", log.LstdFlags)
		}
	} else {
		sl.logger = log.New(os.Stdout, "[SIMILARITY] ", log.LstdFlags)
	}

	return sl
}

// SetLevel устанавливает уровень логирования
func (sl *SimilarityLogger) SetLevel(level LogLevel) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.level = level
}

// Enable включает/выключает логирование
func (sl *SimilarityLogger) Enable(enabled bool) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.enabled = enabled
}

// Debug логирует отладочное сообщение
func (sl *SimilarityLogger) Debug(format string, v ...interface{}) {
	sl.log(LogLevelDebug, "DEBUG", format, v...)
}

// Info логирует информационное сообщение
func (sl *SimilarityLogger) Info(format string, v ...interface{}) {
	sl.log(LogLevelInfo, "INFO", format, v...)
}

// Warn логирует предупреждение
func (sl *SimilarityLogger) Warn(format string, v ...interface{}) {
	sl.log(LogLevelWarn, "WARN", format, v...)
}

// Error логирует ошибку
func (sl *SimilarityLogger) Error(format string, v ...interface{}) {
	sl.log(LogLevelError, "ERROR", format, v...)
}

// log внутренний метод логирования
func (sl *SimilarityLogger) log(level LogLevel, prefix, format string, v ...interface{}) {
	sl.mu.RLock()
	enabled := sl.enabled
	currentLevel := sl.level
	logger := sl.logger
	sl.mu.RUnlock()

	if !enabled || level < currentLevel {
		return
	}

	message := fmt.Sprintf("[%s] %s", prefix, fmt.Sprintf(format, v...))
	logger.Println(message)
}

// LogComparison логирует сравнение двух строк
func (sl *SimilarityLogger) LogComparison(s1, s2 string, similarity float64, method string) {
	sl.Debug("Comparison: '%s' vs '%s' = %.4f (method: %s)", s1, s2, similarity, method)
}

// LogBatchProcessing логирует пакетную обработку
func (sl *SimilarityLogger) LogBatchProcessing(count int, duration time.Duration, cacheHits int) {
	sl.Info("Batch processing: %d pairs in %v, cache hits: %d", count, duration, cacheHits)
}

// LogTraining логирует процесс обучения
func (sl *SimilarityLogger) LogTraining(iterations int, pairsCount int, finalWeights *SimilarityWeights) {
	sl.Info("Training completed: %d iterations, %d pairs, weights: J-W=%.2f, LCS=%.2f, Phon=%.2f, Ngram=%.2f, Jacc=%.2f",
		iterations, pairsCount,
		finalWeights.JaroWinkler, finalWeights.LCS, finalWeights.Phonetic,
		finalWeights.Ngram, finalWeights.Jaccard)
}

// LogPerformance логирует метрики производительности
func (sl *SimilarityLogger) LogPerformance(operation string, duration time.Duration, items int) {
	avgTime := duration / time.Duration(items)
	sl.Info("Performance: %s - %d items in %v (avg: %v/item)", operation, items, duration, avgTime)
}

// Close закрывает файл лога
func (sl *SimilarityLogger) Close() error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.file != nil {
		return sl.file.Close()
	}
	return nil
}

// GetLogStats возвращает статистику логирования
func (sl *SimilarityLogger) GetLogStats() map[string]interface{} {
	sl.mu.RLock()
	defer sl.mu.RUnlock()

	stats := map[string]interface{}{
		"enabled":  sl.enabled,
		"level":    sl.level,
		"log_file": sl.logFile,
	}

	if sl.file != nil {
		if info, err := sl.file.Stat(); err == nil {
			stats["file_size"] = info.Size()
			stats["file_modified"] = info.ModTime().Format(time.RFC3339)
		}
	}

	return stats
}

