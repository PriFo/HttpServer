package monitoring

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"
	"sync"
	"time"
)

// HealthStatus статус здоровья компонента
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// ComponentHealth здоровье отдельного компонента
type ComponentHealth struct {
	Name      string       `json:"name"`
	Status    HealthStatus `json:"status"`
	Message   string       `json:"message,omitempty"`
	Timestamp time.Time    `json:"timestamp"`
	Latency   time.Duration `json:"latency,omitempty"`
}

// HealthCheckResult результат проверки здоровья системы
type HealthCheckResult struct {
	Status    HealthStatus              `json:"status"`
	Timestamp time.Time                 `json:"timestamp"`
	Uptime    time.Duration             `json:"uptime"`
	Version   string                    `json:"version"`
	Components map[string]ComponentHealth `json:"components"`
	System    SystemHealth              `json:"system"`
}

// SystemHealth системные метрики
type SystemHealth struct {
	CPUUsage    float64 `json:"cpu_usage_percent"`
	MemoryUsage float64 `json:"memory_usage_percent"`
	Goroutines  int     `json:"goroutines"`
}

// HealthChecker проверяет здоровье системы
type HealthChecker struct {
	mu              sync.RWMutex
	components      map[string]HealthCheckFunc
	startTime       time.Time
	version         string
	db              *sql.DB
	serviceDB       interface {
		Ping() error
	}
}

// HealthCheckFunc функция проверки здоровья компонента
type HealthCheckFunc func(ctx context.Context) ComponentHealth

// NewHealthChecker создает новый HealthChecker
func NewHealthChecker(version string, db *sql.DB, serviceDB interface{ Ping() error }) *HealthChecker {
	return &HealthChecker{
		components: make(map[string]HealthCheckFunc),
		startTime:  time.Now(),
		version:    version,
		db:         db,
		serviceDB:  serviceDB,
	}
}

// RegisterComponent регистрирует компонент для проверки здоровья
func (hc *HealthChecker) RegisterComponent(name string, checkFunc HealthCheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.components[name] = checkFunc
}

// Check выполняет проверку здоровья всех компонентов
func (hc *HealthChecker) Check(ctx context.Context) HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	components := make(map[string]ComponentHealth)
	overallStatus := HealthStatusHealthy

	// Проверяем базу данных
	if hc.db != nil {
		start := time.Now()
		err := hc.db.PingContext(ctx)
		latency := time.Since(start)
		status := HealthStatusHealthy
		message := "Database is healthy"
		if err != nil {
			status = HealthStatusUnhealthy
			message = fmt.Sprintf("Database error: %v", err)
			overallStatus = HealthStatusUnhealthy
		}
		components["database"] = ComponentHealth{
			Name:      "database",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Latency:   latency,
		}
	}

	// Проверяем service database
	if hc.serviceDB != nil {
		start := time.Now()
		err := hc.serviceDB.Ping()
		latency := time.Since(start)
		status := HealthStatusHealthy
		message := "Service database is healthy"
		if err != nil {
			status = HealthStatusUnhealthy
			message = fmt.Sprintf("Service database error: %v", err)
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
		components["service_database"] = ComponentHealth{
			Name:      "service_database",
			Status:    status,
			Message:   message,
			Timestamp: time.Now(),
			Latency:   latency,
		}
	}

	// Проверяем зарегистрированные компоненты
	for name, checkFunc := range hc.components {
		componentHealth := checkFunc(ctx)
		components[name] = componentHealth
		if componentHealth.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if componentHealth.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	// Собираем системные метрики
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryUsage := float64(m.Alloc) / float64(m.Sys) * 100
	if memoryUsage > 100 {
		memoryUsage = 100
	}

	return HealthCheckResult{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Uptime:     time.Since(hc.startTime),
		Version:    hc.version,
		Components: components,
		System: SystemHealth{
			MemoryUsage: memoryUsage,
			Goroutines:  runtime.NumGoroutine(),
		},
	}
}

// HTTPHandler возвращает HTTP handler для health check endpoint
func (hc *HealthChecker) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		result := hc.Check(ctx)

		// Определяем HTTP статус код
		statusCode := http.StatusOK
		if result.Status == HealthStatusUnhealthy {
			statusCode = http.StatusServiceUnavailable
		} else if result.Status == HealthStatusDegraded {
			statusCode = http.StatusOK // 200, но с предупреждением
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(result)
	}
}

// LivenessHandler возвращает простой liveness probe (для Kubernetes)
func (hc *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Простая проверка - сервер работает
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// ReadinessHandler возвращает readiness probe (для Kubernetes)
func (hc *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		result := hc.Check(ctx)

		// Готов, если база данных доступна
		if dbHealth, ok := result.Components["database"]; ok {
			if dbHealth.Status == HealthStatusHealthy {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("Ready"))
				return
			}
		}

		// Не готов
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not Ready"))
	}
}

// LogHealthStatus логирует статус здоровья
func (hc *HealthChecker) LogHealthStatus() {
	result := hc.Check(context.Background())
	
	slog.Info("Health check",
		"status", result.Status,
		"uptime", result.Uptime,
		"components", len(result.Components),
		"goroutines", result.System.Goroutines,
		"memory_usage", fmt.Sprintf("%.2f%%", result.System.MemoryUsage),
	)

	// Логируем проблемные компоненты
	for name, component := range result.Components {
		if component.Status != HealthStatusHealthy {
			slog.Warn("Component health issue",
				"component", name,
				"status", component.Status,
				"message", component.Message,
			)
		}
	}
}


