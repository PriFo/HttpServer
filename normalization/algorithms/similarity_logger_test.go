package algorithms

import (
	"os"
	"testing"
	"time"
)

func TestSimilarityLogger(t *testing.T) {
	logFile := "test_similarity.log"
	defer os.Remove(logFile)

	logger := NewSimilarityLogger(logFile, LogLevelDebug, true)
	defer logger.Close()

	t.Run("Debug", func(t *testing.T) {
		logger.Debug("Test debug message")
	})

	t.Run("Info", func(t *testing.T) {
		logger.Info("Test info message")
	})

	t.Run("Warn", func(t *testing.T) {
		logger.Warn("Test warn message")
	})

	t.Run("Error", func(t *testing.T) {
		logger.Error("Test error message")
	})

	t.Run("LogComparison", func(t *testing.T) {
		logger.LogComparison("test1", "test2", 0.85, "hybrid")
	})

	t.Run("LogBatchProcessing", func(t *testing.T) {
		logger.LogBatchProcessing(100, time.Second, 50)
	})

	t.Run("LogTraining", func(t *testing.T) {
		weights := DefaultSimilarityWeights()
		logger.LogTraining(100, 50, weights)
	})

	t.Run("LogPerformance", func(t *testing.T) {
		logger.LogPerformance("test_operation", time.Millisecond*100, 10)
	})

	t.Run("GetLogStats", func(t *testing.T) {
		stats := logger.GetLogStats()
		if stats["enabled"] != true {
			t.Error("Logger should be enabled")
		}
	})
}

func TestGlobalLogger(t *testing.T) {
	logger1 := GetLogger()
	logger2 := GetLogger()

	if logger1 != logger2 {
		t.Error("GetLogger should return the same instance")
	}
}

func TestLoggerLevels(t *testing.T) {
	logger := NewSimilarityLogger("", LogLevelWarn, true)
	defer logger.Close()

	// Debug и Info не должны логироваться
	logger.Debug("Should not appear")
	logger.Info("Should not appear")
	
	// Warn и Error должны логироваться
	logger.Warn("Should appear")
	logger.Error("Should appear")
}

func TestLoggerEnableDisable(t *testing.T) {
	logger := NewSimilarityLogger("", LogLevelDebug, false)
	defer logger.Close()

	// Логирование отключено
	logger.Info("Should not appear")

	// Включаем
	logger.Enable(true)
	logger.Info("Should appear")

	// Выключаем
	logger.Enable(false)
	logger.Info("Should not appear")
}

