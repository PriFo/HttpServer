package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/quality"
	apperrors "httpserver/server/errors"
	"httpserver/server/types"
)

// QualityAnalyzerInterface интерфейс для анализатора качества.
// Используется для улучшения тестируемости и возможности замены реализации.
type QualityAnalyzerInterface interface {
	AnalyzeUpload(uploadID int, databaseID int) error
}

// LoggerInterface интерфейс для логирования.
// Используется для улучшения тестируемости и возможности замены реализации.
type LoggerInterface interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
}

// DatabaseInterface интерфейс для работы с базой данных.
// Используется для улучшения тестируемости и возможности замены реализации.
type DatabaseInterface interface {
	GetUploadByUUID(uuid string) (*database.Upload, error)
	GetQualityMetrics(uploadID int) ([]database.DataQualityMetric, error)
	GetQualityIssues(uploadID int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error)
	GetQualityIssuesByUploadIDs(uploadIDs []int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error)
	GetQualityTrends(databaseID int, days int) ([]database.QualityTrend, error)
	GetCurrentQualityMetrics(databaseID int) ([]database.DataQualityMetric, error)
	GetTopQualityIssues(databaseID int, limit int) ([]database.DataQualityIssue, error)
	GetAllUploads() ([]*database.Upload, error)
	GetUploadsByDatabaseID(databaseID int) ([]*database.Upload, error)
	GetQualityStats() (interface{}, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Close() error
}

// DatabaseFactory интерфейс для создания подключений к БД.
// Используется для создания новых подключений в GetQualityStats.
type DatabaseFactory interface {
	NewDB(path string) (DatabaseInterface, error)
}

// defaultLogger стандартная реализация логгера с использованием slog
type defaultLogger struct {
	logger *slog.Logger
}

func newDefaultLogger() *defaultLogger {
	return &defaultLogger{
		logger: slog.Default(),
	}
}

func (l *defaultLogger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		// Преобразуем args в пары ключ-значение для slog
		attrs := make([]interface{}, 0, len(args))
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				attrs = append(attrs, args[i], args[i+1])
			} else {
				attrs = append(attrs, args[i])
			}
		}
		l.logger.Info(msg, attrs...)
	} else {
		l.logger.Info(msg)
	}
}

func (l *defaultLogger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		attrs := make([]interface{}, 0, len(args))
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				attrs = append(attrs, args[i], args[i+1])
			} else {
				attrs = append(attrs, args[i])
			}
		}
		l.logger.Error(msg, attrs...)
	} else {
		l.logger.Error(msg)
	}
}

func (l *defaultLogger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		attrs := make([]interface{}, 0, len(args))
		for i := 0; i < len(args); i += 2 {
			if i+1 < len(args) {
				attrs = append(attrs, args[i], args[i+1])
			} else {
				attrs = append(attrs, args[i])
			}
		}
		l.logger.Warn(msg, attrs...)
	} else {
		l.logger.Warn(msg)
	}
}

// DatabaseAdapter адаптер для *database.DB, реализующий DatabaseInterface
// Экспортирован для использования в handlers
type DatabaseAdapter struct {
	DB *database.DB
}

// databaseAdapter внутренний адаптер (алиас для обратной совместимости)
type databaseAdapter = DatabaseAdapter

func (a *DatabaseAdapter) GetUploadByUUID(uuid string) (*database.Upload, error) {
	return a.DB.GetUploadByUUID(uuid)
}

func (a *DatabaseAdapter) GetQualityMetrics(uploadID int) ([]database.DataQualityMetric, error) {
	return a.DB.GetQualityMetrics(uploadID)
}

func (a *DatabaseAdapter) GetQualityIssues(uploadID int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error) {
	return a.DB.GetQualityIssues(uploadID, filters, limit, offset)
}

func (a *DatabaseAdapter) GetQualityIssuesByUploadIDs(uploadIDs []int, filters map[string]interface{}, limit, offset int) ([]database.DataQualityIssue, int, error) {
	return a.DB.GetQualityIssuesByUploadIDs(uploadIDs, filters, limit, offset)
}

func (a *DatabaseAdapter) GetQualityTrends(databaseID int, days int) ([]database.QualityTrend, error) {
	return a.DB.GetQualityTrends(databaseID, days)
}

func (a *DatabaseAdapter) GetCurrentQualityMetrics(databaseID int) ([]database.DataQualityMetric, error) {
	return a.DB.GetCurrentQualityMetrics(databaseID)
}

func (a *DatabaseAdapter) GetTopQualityIssues(databaseID int, limit int) ([]database.DataQualityIssue, error) {
	return a.DB.GetTopQualityIssues(databaseID, limit)
}

func (a *DatabaseAdapter) GetAllUploads() ([]*database.Upload, error) {
	return a.DB.GetAllUploads()
}

func (a *DatabaseAdapter) GetUploadsByDatabaseID(databaseID int) ([]*database.Upload, error) {
	return a.DB.GetUploadsByDatabaseID(databaseID)
}

func (a *DatabaseAdapter) GetQualityStats() (interface{}, error) {
	stats, err := a.DB.GetQualityStats()
	if err != nil {
		return nil, err
	}
	return stats, nil
}

func (a *DatabaseAdapter) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return a.DB.Query(query, args...)
}

func (a *DatabaseAdapter) Close() error {
	return a.DB.Close()
}

// NewDatabaseAdapter создает новый адаптер для *database.DB
// Используется в handlers для преобразования *database.DB в DatabaseInterface
func NewDatabaseAdapter(db *database.DB) *DatabaseAdapter {
	if db == nil {
		return nil
	}
	return &DatabaseAdapter{DB: db}
}

// databaseFactoryImpl реализация DatabaseFactory для создания подключений к БД
type databaseFactoryImpl struct{}

func (f *databaseFactoryImpl) NewDB(path string) (DatabaseInterface, error) {
	db, err := database.NewDB(path)
	if err != nil {
		return nil, err
	}
	return NewDatabaseAdapter(db), nil
}

// cachedConnection хранит подключение к БД с временем последнего использования
type cachedConnection struct {
	db       DatabaseInterface
	lastUsed time.Time
	refCount int
	mutex    sync.RWMutex
}

// QualityService сервис для работы с качеством данных.
// Предоставляет методы для получения статистики, отчетов, дашбордов и запуска анализа качества.
type QualityService struct {
	db              DatabaseInterface
	dbFactory       DatabaseFactory
	qualityAnalyzer QualityAnalyzerInterface
	logger          LoggerInterface
	connectionCache sync.Map // map[string]*cachedConnection - кэш подключений по пути к БД
	cacheMutex      sync.RWMutex
	maxCacheAge     time.Duration // Максимальное время жизни подключения в кэше
}

// NewQualityService создает новый сервис качества.
// Принимает подключение к базе данных и анализатор качества.
// Возвращает ошибку, если db равен nil.
func NewQualityService(db *database.DB, qualityAnalyzer *quality.QualityAnalyzer) (*QualityService, error) {
	if db == nil {
		return nil, errors.New("database.DB cannot be nil")
	}
	return &QualityService{
		db:              NewDatabaseAdapter(db),
		dbFactory:       &databaseFactoryImpl{},
		qualityAnalyzer: qualityAnalyzer,
		logger:          newDefaultLogger(),
		maxCacheAge:     5 * time.Minute, // Подключения кэшируются на 5 минут
	}, nil
}

// NewQualityServiceWithDeps создает новый сервис качества с возможностью внедрения зависимостей.
// Используется для тестирования и позволяет передать моки для анализатора и логгера.
// Если logger равен nil, используется defaultLogger.
// Если dbFactory равен nil, используется databaseFactoryImpl.
func NewQualityServiceWithDeps(db DatabaseInterface, qualityAnalyzer QualityAnalyzerInterface, logger LoggerInterface, dbFactory DatabaseFactory) (*QualityService, error) {
	if db == nil {
		return nil, errors.New("database.DB cannot be nil")
	}
	if logger == nil {
		logger = newDefaultLogger()
	}
	if dbFactory == nil {
		dbFactory = &databaseFactoryImpl{}
	}
	return &QualityService{
		db:              db,
		dbFactory:       dbFactory,
		qualityAnalyzer: qualityAnalyzer,
		logger:          logger,
		maxCacheAge:     5 * time.Minute,
	}, nil
}

// getCachedConnection получает подключение из кэша или создает новое
func (qs *QualityService) getCachedConnection(databasePath string) (DatabaseInterface, error) {
	// Проверяем кэш
	if cached, ok := qs.connectionCache.Load(databasePath); ok {
		conn := cached.(*cachedConnection)
		conn.mutex.Lock()
		defer conn.mutex.Unlock()

		// Проверяем, не устарело ли подключение
		if time.Since(conn.lastUsed) < qs.maxCacheAge {
			conn.lastUsed = time.Now()
			conn.refCount++
			qs.logger.Info("Using cached database connection", "path", databasePath)
			return conn.db, nil
		}

		// Подключение устарело, закрываем его
		if closeErr := conn.db.Close(); closeErr != nil {
			qs.logger.Warn("Failed to close stale connection", "error", closeErr)
		}
		qs.connectionCache.Delete(databasePath)
	}

	// Создаем новое подключение
	qs.logger.Info("Creating new database connection", "path", databasePath)
	db, err := qs.dbFactory.NewDB(databasePath)
	if err != nil {
		qs.logger.Error("Failed to create database connection", "path", databasePath, "error", err)
		return nil, apperrors.NewInternalError("не удалось создать подключение к базе данных", err)
	}

	// Сохраняем в кэш
	cached := &cachedConnection{
		db:       db,
		lastUsed: time.Now(),
		refCount: 1,
	}
	qs.connectionCache.Store(databasePath, cached)

	return db, nil
}

// releaseCachedConnection уменьшает счетчик ссылок на подключение
func (qs *QualityService) releaseCachedConnection(databasePath string) {
	if cached, ok := qs.connectionCache.Load(databasePath); ok {
		conn := cached.(*cachedConnection)
		conn.mutex.Lock()
		defer conn.mutex.Unlock()

		conn.refCount--
		if conn.refCount <= 0 {
			// Если нет активных ссылок, можно закрыть подключение
			// Но оставляем его в кэше на случай повторного использования
			conn.lastUsed = time.Now()
		}
	}
}

// CleanupStaleConnections очищает устаревшие подключения из кэша
func (qs *QualityService) CleanupStaleConnections() {
	now := time.Now()
	qs.connectionCache.Range(func(key, value interface{}) bool {
		conn := value.(*cachedConnection)
		conn.mutex.RLock()
		age := now.Sub(conn.lastUsed)
		refCount := conn.refCount
		conn.mutex.RUnlock()

		// Удаляем подключения, которые не использовались более maxCacheAge и не имеют активных ссылок
		if age > qs.maxCacheAge && refCount <= 0 {
			conn.mutex.Lock()
			if closeErr := conn.db.Close(); closeErr != nil {
				qs.logger.Warn("Failed to close stale connection during cleanup", "error", closeErr)
			}
			conn.mutex.Unlock()
			qs.connectionCache.Delete(key)
			qs.logger.Info("Cleaned up stale database connection", "path", key)
		}
		return true
	})
}

// CloseAllConnections закрывает все подключения в кэше
// Должен вызываться при завершении работы сервиса
func (qs *QualityService) CloseAllConnections() {
	qs.connectionCache.Range(func(key, value interface{}) bool {
		conn := value.(*cachedConnection)
		conn.mutex.Lock()
		if closeErr := conn.db.Close(); closeErr != nil {
			qs.logger.Warn("Failed to close connection", "path", key, "error", closeErr)
		}
		conn.mutex.Unlock()
		qs.connectionCache.Delete(key)
		qs.logger.Info("Closed database connection", "path", key)
		return true
	})
}

// GetQualityStats получает статистику качества для базы данных.
// Если databasePath не пустой, использует кэшированное подключение или создает новое.
// Если databasePath пустой, использует currentDB (который не должен быть nil).
// Возвращает статистику качества или ошибку при неудаче.
func (qs *QualityService) GetQualityStats(ctx context.Context, databasePath string, currentDB DatabaseInterface) (interface{}, error) {
	if err := ValidateContext(ctx); err != nil {
		return nil, err
	}

	var db DatabaseInterface
	var err error
	var shouldRelease bool

	if databasePath != "" {
		// Используем кэшированное подключение
		db, err = qs.getCachedConnection(databasePath)
		if err != nil {
			return nil, err
		}
		shouldRelease = true
	} else {
		if currentDB == nil {
			return nil, errors.New("currentDB cannot be nil when databasePath is empty")
		}
		db = currentDB
		shouldRelease = false
	}

	// Освобождаем ссылку на подключение после использования
	if shouldRelease {
		defer qs.releaseCachedConnection(databasePath)
	}

	stats, err := db.GetQualityStats()
	if err != nil {
		qs.logger.Error("Failed to get quality stats", "error", err)
		return nil, apperrors.NewInternalError("не удалось получить статистику качества", err)
	}

	qs.logger.Info("Successfully retrieved quality stats")
	return stats, nil
}

// GetQualityReport получает отчет о качестве для выгрузки.
// uploadUUID - UUID выгрузки для которой нужно получить отчет.
// summaryOnly - если true, возвращает только сводку без детальных проблем.
// limit и offset используются для пагинации проблем качества.
// Возвращает отчет о качестве или ошибку при неудаче.
func (qs *QualityService) GetQualityReport(ctx context.Context, uploadUUID string, summaryOnly bool, limit, offset int) (*types.QualityReport, error) {
	if ctx == nil {
		return nil, errors.New("context cannot be nil")
	}

	// Проверяем отмену контекста
	select {
	case <-ctx.Done():
		return nil, apperrors.NewServiceUnavailableError("операция отменена", ctx.Err())
	default:
	}

	if uploadUUID == "" {
		return nil, errors.New("uploadUUID cannot be empty")
	}

	if limit < 0 {
		return nil, errors.New("limit cannot be negative")
	}

	if offset < 0 {
		return nil, errors.New("offset cannot be negative")
	}

	qs.logger.Info("Getting quality report", "upload_uuid", uploadUUID, "summary_only", summaryOnly, "limit", limit, "offset", offset)

	upload, err := qs.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		qs.logger.Error("Failed to get upload", "upload_uuid", uploadUUID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	// Получаем метрики качества
	metrics, err := qs.db.GetQualityMetrics(upload.ID)
	if err != nil {
		qs.logger.Error("Failed to get quality metrics", "upload_id", upload.ID, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить метрики качества", err)
	}

	// Получаем проблемы качества с пагинацией
	var issues []database.DataQualityIssue
	var totalIssuesCount int
	if !summaryOnly {
		issues, totalIssuesCount, err = qs.db.GetQualityIssues(upload.ID, map[string]interface{}{}, limit, offset)
		if err != nil {
			qs.logger.Error("Failed to get quality issues", "upload_id", upload.ID, "error", err)
			return nil, apperrors.NewInternalError("не удалось получить проблемы качества", err)
		}
	} else {
		// Для сводки получаем только количество без деталей
		_, totalIssuesCount, err = qs.db.GetQualityIssues(upload.ID, map[string]interface{}{}, 0, 0)
		if err != nil {
			qs.logger.Error("Failed to count quality issues", "upload_id", upload.ID, "error", err)
			return nil, apperrors.NewInternalError("не удалось подсчитать проблемы качества", err)
		}
		issues = []database.DataQualityIssue{} // Пустой список
	}

	// Формируем сводку
	summary, err := qs.buildQualitySummary(upload.ID, issues, totalIssuesCount, metrics, summaryOnly)
	if err != nil {
		qs.logger.Error("Failed to build quality summary", "upload_id", upload.ID, "error", err)
		return nil, apperrors.NewInternalError("не удалось построить сводку качества", err)
	}

	databaseID := 0
	if upload.DatabaseID != nil {
		databaseID = *upload.DatabaseID
	}

	// Рассчитываем общий балл
	overallScore := qs.calculateOverallScore(metrics)

	report := &types.QualityReport{
		UploadUUID:   uploadUUID,
		DatabaseID:   databaseID,
		AnalyzedAt:   upload.CompletedAt,
		OverallScore: overallScore,
		Metrics:      metrics,
		Issues:       issues,
		Summary:      summary,
	}

	qs.logger.Info("Successfully generated quality report", "upload_uuid", uploadUUID, "total_issues", totalIssuesCount)
	return report, nil
}

// getIssuesSeverityStats получает статистику по уровням серьезности проблем
func (qs *QualityService) getIssuesSeverityStats(uploadID int) (map[string]int, error) {
	query := `
		SELECT issue_severity, COUNT(*) as count
		FROM data_quality_issues
		WHERE upload_id = ?
		GROUP BY issue_severity
	`

	// Используем prepared statement через db.Query
	rows, err := qs.db.Query(query, uploadID)
	if err != nil {
		qs.logger.Error("Failed to query severity stats", "upload_id", uploadID, "error", err)
		return nil, apperrors.NewInternalError("не удалось выполнить запрос статистики серьезности", err)
	}
	defer rows.Close()

	stats := map[string]int{
		"CRITICAL": 0,
		"HIGH":     0,
		"MEDIUM":   0,
		"LOW":      0,
	}

	scanErrors := 0
	for rows.Next() {
		var severity string
		var count int
		if err := rows.Scan(&severity, &count); err != nil {
			// Логируем ошибки сканирования, но продолжаем обработку
			scanErrors++
			qs.logger.Warn("Failed to scan severity stats row", "upload_id", uploadID, "error", err)
			continue
		}
		// Валидируем severity перед добавлением
		if _, exists := stats[severity]; exists {
			stats[severity] = count
		} else {
			qs.logger.Warn("Unknown severity level", "severity", severity, "upload_id", uploadID)
		}
	}

	if err := rows.Err(); err != nil {
		qs.logger.Error("Error iterating severity stats rows", "upload_id", uploadID, "error", err)
		return nil, apperrors.NewInternalError("ошибка при итерации статистики серьезности", err)
	}

	if scanErrors > 0 {
		qs.logger.Warn("Some rows failed to scan", "upload_id", uploadID, "errors_count", scanErrors)
	}

	return stats, nil
}

// AnalyzeQuality запускает анализ качества для выгрузки.
// uploadUUID - UUID выгрузки для которой нужно запустить анализ.
// Возвращает ошибку если анализ не может быть запущен (выгрузка не найдена, анализатор не инициализирован и т.д.).
func (qs *QualityService) AnalyzeQuality(ctx context.Context, uploadUUID string) error {
	if err := ValidateContext(ctx); err != nil {
		return err
	}

	if uploadUUID == "" {
		return errors.New("uploadUUID cannot be empty")
	}

	// Проверяем, что qualityAnalyzer не nil (включая случай, когда интерфейс содержит nil указатель)
	if qs.qualityAnalyzer == nil || (reflect.ValueOf(qs.qualityAnalyzer).Kind() == reflect.Ptr && reflect.ValueOf(qs.qualityAnalyzer).IsNil()) {
		qs.logger.Error("Quality analyzer is not initialized", "upload_uuid", uploadUUID)
		return errors.New("quality analyzer is not initialized")
	}

	qs.logger.Info("Starting quality analysis", "upload_uuid", uploadUUID)

	upload, err := qs.db.GetUploadByUUID(uploadUUID)
	if err != nil {
		qs.logger.Error("Failed to get upload for analysis", "upload_uuid", uploadUUID, "error", err)
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.NewNotFoundError("выгрузка не найдена", err)
		}
		return apperrors.NewInternalError("не удалось получить выгрузку", err)
	}

	databaseID := 0
	if upload.DatabaseID != nil {
		databaseID = *upload.DatabaseID
	}

	if databaseID <= 0 {
		qs.logger.Error("Database ID not set for upload", "upload_uuid", uploadUUID, "upload_id", upload.ID)
		return apperrors.NewValidationError(fmt.Sprintf("database_id не установлен для выгрузки %s", uploadUUID), nil)
	}

	qs.logger.Info("Calling quality analyzer", "upload_id", upload.ID, "database_id", databaseID)
	if err := qs.qualityAnalyzer.AnalyzeUpload(upload.ID, databaseID); err != nil {
		qs.logger.Error("Quality analysis failed", "upload_uuid", uploadUUID, "upload_id", upload.ID, "database_id", databaseID, "error", err)
		return apperrors.NewInternalError("анализ качества не удался", err)
	}

	qs.logger.Info("Quality analysis completed successfully", "upload_uuid", uploadUUID, "upload_id", upload.ID, "database_id", databaseID)
	return nil
}

// GetQualityDashboard получает дашборд качества для базы данных.
// databaseID - ID базы данных.
// days - количество дней для расчета трендов.
// limit - максимальное количество топ проблем для возврата.
// Возвращает дашборд с трендами, метриками и проблемами или ошибку при неудаче.
func (qs *QualityService) GetQualityDashboard(ctx context.Context, databaseID int, days int, limit int) (*types.QualityDashboard, error) {
	if err := ValidateContext(ctx); err != nil {
		return nil, err
	}

	if databaseID <= 0 {
		return nil, errors.New("databaseID must be positive")
	}

	if days <= 0 {
		return nil, errors.New("days must be positive")
	}

	if limit < 0 {
		return nil, errors.New("limit cannot be negative")
	}

	qs.logger.Info("Getting quality dashboard", "database_id", databaseID, "days", days, "limit", limit)
	// Получаем тренды качества
	trends, err := qs.db.GetQualityTrends(databaseID, days)
	if err != nil {
		qs.logger.Error("Failed to get quality trends", "database_id", databaseID, "days", days, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить тренды качества", err)
	}

	// Текущие метрики
	currentMetrics, err := qs.db.GetCurrentQualityMetrics(databaseID)
	if err != nil {
		qs.logger.Error("Failed to get current metrics", "database_id", databaseID, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить текущие метрики", err)
	}

	// Топ проблем
	topIssues, err := qs.db.GetTopQualityIssues(databaseID, limit)
	if err != nil {
		qs.logger.Error("Failed to get top issues", "database_id", databaseID, "limit", limit, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить топ проблем", err)
	}

	// Группируем метрики по сущностям
	metricsByEntity := qs.groupMetricsByEntity(currentMetrics)

	// Рассчитываем текущий общий балл
	currentScore := qs.calculateCurrentScore(trends, currentMetrics)

	dashboard := &types.QualityDashboard{
		DatabaseID:      databaseID,
		CurrentScore:    currentScore,
		Trends:          trends,
		TopIssues:       topIssues,
		MetricsByEntity: metricsByEntity,
	}

	qs.logger.Info("Successfully generated quality dashboard", "database_id", databaseID, "trends_count", len(trends), "top_issues_count", len(topIssues))
	return dashboard, nil
}

// GetQualityIssues получает проблемы качества для базы данных.
// databaseID - ID базы данных.
// filters - карта фильтров для проблем (entity_type, severity, status и т.д.).
//
// Возвращает список проблем качества или ошибку при неудаче.
func (qs *QualityService) GetQualityIssues(ctx context.Context, databaseID int, filters map[string]interface{}) ([]database.DataQualityIssue, error) {
	if err := ValidateContext(ctx); err != nil {
		return nil, err
	}

	if databaseID <= 0 {
		return nil, errors.New("databaseID must be positive")
	}

	qs.logger.Info("Getting quality issues", "database_id", databaseID)

	// Получаем выгрузки для базы данных напрямую (исправление N+1 проблемы)
	relevantUploads, err := qs.db.GetUploadsByDatabaseID(databaseID)
	if err != nil {
		qs.logger.Error("Failed to get uploads by database id", "database_id", databaseID, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить выгрузки по ID базы данных", err)
	}

	qs.logger.Info("Found relevant uploads", "database_id", databaseID, "count", len(relevantUploads))

	if len(relevantUploads) == 0 {
		qs.logger.Info("No uploads found for database", "database_id", databaseID)
		return []database.DataQualityIssue{}, nil
	}

	// Собираем все upload IDs для batch-запроса (оптимизация N+1 проблемы)
	uploadIDs := make([]int, 0, len(relevantUploads))
	for _, upload := range relevantUploads {
		uploadIDs = append(uploadIDs, upload.ID)
	}

	// Получаем все issues одним batch-запросом (оптимизация N+1 проблемы)
	allIssues, _, err := qs.db.GetQualityIssuesByUploadIDs(uploadIDs, filters, 0, 0)
	if err != nil {
		qs.logger.Error("Failed to get quality issues by upload IDs", "database_id", databaseID, "upload_count", len(uploadIDs), "error", err)
		return nil, apperrors.NewInternalError("не удалось получить проблемы качества по ID выгрузок", err)
	}

	qs.logger.Info("Retrieved quality issues", "database_id", databaseID, "total_issues", len(allIssues))
	// Гарантируем, что возвращаем пустой слайс, а не nil
	if allIssues == nil {
		allIssues = []database.DataQualityIssue{}
	}
	return allIssues, nil
}

// GetQualityTrends получает тренды качества для базы данных.
// databaseID - ID базы данных.
// days - количество дней для расчета трендов.
// Возвращает список трендов качества или ошибку при неудаче.
func (qs *QualityService) GetQualityTrends(ctx context.Context, databaseID int, days int) ([]database.QualityTrend, error) {
	if err := ValidateContext(ctx); err != nil {
		return nil, err
	}

	if databaseID <= 0 {
		return nil, errors.New("databaseID must be positive")
	}

	if days <= 0 {
		return nil, errors.New("days must be positive")
	}

	qs.logger.Info("Getting quality trends", "database_id", databaseID, "days", days)

	trends, err := qs.db.GetQualityTrends(databaseID, days)
	if err != nil {
		qs.logger.Error("Failed to get quality trends", "database_id", databaseID, "days", days, "error", err)
		return nil, apperrors.NewInternalError("не удалось получить тренды качества", err)
	}

	qs.logger.Info("Successfully retrieved quality trends", "database_id", databaseID, "trends_count", len(trends))
	// Гарантируем, что возвращаем пустой слайс, а не nil
	if trends == nil {
		trends = []database.QualityTrend{}
	}
	return trends, nil
}

// buildQualitySummary формирует сводку по качеству
func (qs *QualityService) buildQualitySummary(uploadID int, issues []database.DataQualityIssue, totalIssuesCount int, metrics []database.DataQualityMetric, summaryOnly bool) (types.QualitySummary, error) {
	summary := types.QualitySummary{
		TotalIssues:       totalIssuesCount,
		MetricsByCategory: make(map[string]float64),
	}

	// Подсчитываем проблемы по уровням серьезности
	if !summaryOnly {
		summary.CriticalIssues = qs.countIssuesBySeverity(issues, "CRITICAL")
		summary.HighIssues = qs.countIssuesBySeverity(issues, "HIGH")
		summary.MediumIssues = qs.countIssuesBySeverity(issues, "MEDIUM")
		summary.LowIssues = qs.countIssuesBySeverity(issues, "LOW")
	} else {
		// Для summary_only получаем статистику по уровням отдельным запросом
		severityStats, err := qs.getIssuesSeverityStats(uploadID)
		if err != nil {
			// Не критично, продолжаем с нулевыми значениями
			qs.logger.Warn("Failed to get severity stats, using zero values", "upload_id", uploadID, "error", err)
			severityStats = map[string]int{
				"CRITICAL": 0,
				"HIGH":     0,
				"MEDIUM":   0,
				"LOW":      0,
			}
		}
		summary.CriticalIssues = severityStats["CRITICAL"]
		summary.HighIssues = severityStats["HIGH"]
		summary.MediumIssues = severityStats["MEDIUM"]
		summary.LowIssues = severityStats["LOW"]
	}

	// Группируем метрики по категориям
	qs.groupMetricsByCategory(metrics, &summary)

	return summary, nil
}

// countIssuesBySeverity подсчитывает количество проблем по уровню серьезности
func (qs *QualityService) countIssuesBySeverity(issues []database.DataQualityIssue, severity string) int {
	count := 0
	for _, issue := range issues {
		if issue.IssueSeverity == severity {
			count++
		}
	}
	return count
}

// groupMetricsByCategory группирует метрики по категориям и рассчитывает средние значения
func (qs *QualityService) groupMetricsByCategory(metrics []database.DataQualityMetric, summary *types.QualitySummary) {
	// Сначала суммируем значения по категориям
	categoryCounts := make(map[string]int)
	for _, metric := range metrics {
		if _, exists := summary.MetricsByCategory[metric.MetricCategory]; !exists {
			summary.MetricsByCategory[metric.MetricCategory] = 0.0
		}
		summary.MetricsByCategory[metric.MetricCategory] += metric.MetricValue
		categoryCounts[metric.MetricCategory]++
	}

	// Рассчитываем средние значения
	for category := range summary.MetricsByCategory {
		if count := categoryCounts[category]; count > 0 {
			summary.MetricsByCategory[category] = summary.MetricsByCategory[category] / float64(count)
		}
	}
}

// calculateOverallScore рассчитывает общий балл качества
func (qs *QualityService) calculateOverallScore(metrics []database.DataQualityMetric) float64 {
	if len(metrics) == 0 {
		return 0.0
	}

	var totalScore float64
	for _, metric := range metrics {
		totalScore += metric.MetricValue
	}

	return totalScore / float64(len(metrics))
}

// convertMetricsToInterface конвертирует метрики в []interface{}
func (qs *QualityService) convertMetricsToInterface(metrics []database.DataQualityMetric) []interface{} {
	result := make([]interface{}, len(metrics))
	for i, m := range metrics {
		result[i] = m
	}
	return result
}

// convertIssuesToInterface конвертирует проблемы в []interface{}
func (qs *QualityService) convertIssuesToInterface(issues []database.DataQualityIssue) []interface{} {
	result := make([]interface{}, len(issues))
	for i, issue := range issues {
		result[i] = issue
	}
	return result
}

// groupMetricsByEntity группирует метрики по типам сущностей
func (qs *QualityService) groupMetricsByEntity(metrics []database.DataQualityMetric) map[string]types.EntityMetrics {
	metricsByEntity := make(map[string]types.EntityMetrics)

	for _, metric := range metrics {
		// Определяем тип сущности из имени метрики
		entityType := qs.determineEntityType(metric.MetricName)

		if _, exists := metricsByEntity[entityType]; !exists {
			metricsByEntity[entityType] = types.EntityMetrics{}
		}

		entityMetrics := metricsByEntity[entityType]
		qs.updateEntityMetrics(&entityMetrics, metric)
		metricsByEntity[entityType] = entityMetrics
	}

	// Рассчитываем общие баллы для каждой сущности
	for entityType := range metricsByEntity {
		entityMetrics := metricsByEntity[entityType]
		entityMetrics.OverallScore = qs.calculateEntityOverallScore(entityMetrics)
		metricsByEntity[entityType] = entityMetrics
	}

	return metricsByEntity
}

// determineEntityType определяет тип сущности по имени метрики
func (qs *QualityService) determineEntityType(metricName string) string {
	metricNameLower := strings.ToLower(metricName)
	if strings.Contains(metricNameLower, "nomenclature") {
		return "nomenclature"
	}
	if strings.Contains(metricNameLower, "counterparty") {
		return "counterparty"
	}
	return "unknown"
}

// updateEntityMetrics обновляет метрики сущности на основе метрики
func (qs *QualityService) updateEntityMetrics(entityMetrics *types.EntityMetrics, metric database.DataQualityMetric) {
	switch metric.MetricCategory {
	case "completeness":
		entityMetrics.Completeness = metric.MetricValue
	case "consistency":
		entityMetrics.Consistency = metric.MetricValue
	case "uniqueness":
		entityMetrics.Uniqueness = metric.MetricValue
	case "validity":
		entityMetrics.Validity = metric.MetricValue
	}
}

// calculateEntityOverallScore рассчитывает общий балл для сущности
func (qs *QualityService) calculateEntityOverallScore(entityMetrics types.EntityMetrics) float64 {
	count := 0
	total := 0.0

	if entityMetrics.Completeness > 0 {
		total += entityMetrics.Completeness
		count++
	}
	if entityMetrics.Consistency > 0 {
		total += entityMetrics.Consistency
		count++
	}
	if entityMetrics.Uniqueness > 0 {
		total += entityMetrics.Uniqueness
		count++
	}
	if entityMetrics.Validity > 0 {
		total += entityMetrics.Validity
		count++
	}

	if count > 0 {
		return total / float64(count)
	}

	return 0.0
}

// calculateCurrentScore рассчитывает текущий общий балл
func (qs *QualityService) calculateCurrentScore(trends []database.QualityTrend, currentMetrics []database.DataQualityMetric) float64 {
	if len(trends) > 0 {
		return trends[0].OverallScore
	}

	if len(currentMetrics) > 0 {
		return qs.calculateOverallScore(currentMetrics)
	}

	return 0.0
}

// convertTrendsToInterface конвертирует тренды в []interface{}
func (qs *QualityService) convertTrendsToInterface(trends []database.QualityTrend) []interface{} {
	result := make([]interface{}, len(trends))
	for i, t := range trends {
		result[i] = t
	}
	return result
}
