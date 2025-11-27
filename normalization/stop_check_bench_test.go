package normalization

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// BenchmarkStopCheck_Simple проверяет производительность простой проверки остановки
func BenchmarkStopCheck_Simple(b *testing.B) {
	stopCheck := func() bool {
		return false
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stopCheck()
	}
}

// BenchmarkStopCheck_WithMutex проверяет производительность проверки с мьютексом
func BenchmarkStopCheck_WithMutex(b *testing.B) {
	var mu sync.RWMutex
	running := true

	stopCheck := func() bool {
		mu.RLock()
		defer mu.RUnlock()
		return !running
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stopCheck()
	}
}

// BenchmarkStopCheck_WithAtomic проверяет производительность проверки с atomic
func BenchmarkStopCheck_WithAtomic(b *testing.B) {
	var running int32 = 1

	stopCheck := func() bool {
		return atomic.LoadInt32(&running) == 0
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stopCheck()
	}
}

// BenchmarkStopCheck_WithMetrics проверяет производительность проверки с метриками
func BenchmarkStopCheck_WithMetrics(b *testing.B) {
	stopCheck := func() bool {
		checkStart := time.Now()
		result := false
		checkDuration := time.Since(checkStart)
		recordStopCheck(checkDuration, result)
		return result
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = stopCheck()
	}
}

// BenchmarkStopCheck_Realistic симулирует реальный сценарий с проверкой каждые 50 записей
func BenchmarkStopCheck_Realistic(b *testing.B) {
	var mu sync.RWMutex
	running := true

	stopCheck := func() bool {
		mu.RLock()
		defer mu.RUnlock()
		return !running
	}

	// Симулируем обработку с проверкой каждые 50 записей
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Симулируем обработку записи
		_ = i * 2

		// Проверка каждые 50 записей
		if i > 0 && i%50 == 0 {
			_ = stopCheck()
		}
	}
}

// BenchmarkStopCheck_Parallel проверяет производительность в параллельном режиме
func BenchmarkStopCheck_Parallel(b *testing.B) {
	var mu sync.RWMutex
	running := true

	stopCheck := func() bool {
		mu.RLock()
		defer mu.RUnlock()
		return !running
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = stopCheck()
		}
	})
}

// BenchmarkStopCheck_WithMetrics_Parallel проверяет производительность с метриками в параллельном режиме
func BenchmarkStopCheck_WithMetrics_Parallel(b *testing.B) {
	var mu sync.RWMutex
	running := true

	stopCheck := func() bool {
		checkStart := time.Now()
		mu.RLock()
		result := !running
		mu.RUnlock()
		checkDuration := time.Since(checkStart)
		recordStopCheck(checkDuration, result)
		return result
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = stopCheck()
		}
	})
}
