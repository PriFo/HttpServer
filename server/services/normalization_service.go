package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"httpserver/database"
	"httpserver/internal/domain/models"
	"httpserver/normalization"
	apperrors "httpserver/server/errors"
)

// NormalizationStateManager - интерфейс для управления состоянием нормализации
// АРХИТЕКТУРНАЯ ЗАМЕТКА: В текущей реализации состояние нормализации дублируется в нескольких местах:
// - Server.normalizerRunning (глобальное состояние)
// - NormalizationService.normalizerRunning (состояние сервиса)
// - CounterpartyService.normalizerRunning (состояние для контрагентов)
//
// TODO: Для централизации рекомендуется:
// 1. Сделать NormalizationService единственным источником правды о состоянии
// 2. Удалить normalizerRunning из Server и CounterpartyService
// 3. Использовать этот интерфейс для проверки состояния из других компонентов
type NormalizationStateManager interface {
	// IsRunning возвращает true, если нормализация запущена
	IsRunning() bool
	// Start помечает нормализацию как запущенную
	Start() error
	// Stop останавливает нормализацию
	Stop() bool
	// GetStatus возвращает текущий статус нормализации
	GetStatus() NormalizationStatus
}

// NormalizationService сервис для управления нормализацией
// Реализует интерфейс NormalizationStateManager
type NormalizationService struct {
	db                  *database.DB
	serviceDB           *database.ServiceDB
	normalizer          *normalization.Normalizer
	benchmarkService    *BenchmarkService
	normalizerMutex     sync.RWMutex
	normalizerRunning   bool
	normalizerStartTime time.Time
	normalizerProcessed int
	normalizerSuccess   int
	normalizerErrors    int
	normalizerEvents    chan<- string
	normalizerCtx       context.Context
	normalizerCancel    context.CancelFunc
}

// NewNormalizationService создает новый сервис нормализации
func NewNormalizationService(
	db *database.DB,
	serviceDB *database.ServiceDB,
	normalizer *normalization.Normalizer,
	benchmarkService *BenchmarkService,
	normalizerEvents chan<- string,
) *NormalizationService {
	return &NormalizationService{
		db:                db,
		serviceDB:         serviceDB,
		normalizer:        normalizer,
		benchmarkService:  benchmarkService,
		normalizerEvents:  normalizerEvents,
		normalizerRunning: false,
	}
}

// Compile-time проверка, что NormalizationService реализует интерфейс NormalizationStateManager
var _ NormalizationStateManager = (*NormalizationService)(nil)

// IsRunning проверяет, запущена ли нормализация
func (ns *NormalizationService) IsRunning() bool {
	ns.normalizerMutex.RLock()
	defer ns.normalizerMutex.RUnlock()
	return ns.normalizerRunning
}

// Start запускает нормализацию
func (ns *NormalizationService) Start() error {
	ns.normalizerMutex.Lock()
	defer ns.normalizerMutex.Unlock()

	if ns.normalizerRunning {
		return apperrors.NewConflictError("нормализация уже запущена", nil)
	}

	ns.normalizerRunning = true
	ns.normalizerStartTime = time.Now()
	ns.normalizerProcessed = 0
	ns.normalizerSuccess = 0
	ns.normalizerErrors = 0

	return nil
}

// Stop останавливает нормализацию
func (ns *NormalizationService) Stop() bool {
	ns.normalizerMutex.Lock()
	defer ns.normalizerMutex.Unlock()

	wasRunning := ns.normalizerRunning
	ns.normalizerRunning = false

	if ns.normalizerCancel != nil {
		ns.normalizerCancel()
		ns.normalizerCancel = nil
	}

	return wasRunning
}

// GetStatus возвращает статус нормализации
func (ns *NormalizationService) GetStatus() NormalizationStatus {
	ns.normalizerMutex.RLock()
	defer ns.normalizerMutex.RUnlock()

	elapsedTime := time.Since(ns.normalizerStartTime)
	var startTimeStr string
	if !ns.normalizerStartTime.IsZero() {
		startTimeStr = ns.normalizerStartTime.Format(time.RFC3339)
	}

	return NormalizationStatus{
		IsRunning:   ns.normalizerRunning,
		Processed:   ns.normalizerProcessed,
		Success:     ns.normalizerSuccess,
		Errors:      ns.normalizerErrors,
		StartTime:   startTimeStr,
		ElapsedTime: elapsedTime.String(),
	}
}

// NormalizeName нормализует название с приоритетом поиска в эталонах
func (ns *NormalizationService) NormalizeName(name string, entityType string) (string, error) {
	// Сначала ищем в эталонах
	if ns.benchmarkService != nil {
		benchmark, err := ns.benchmarkService.FindBestMatch(name, entityType)
		if err != nil {
			// NotFoundError не критична - просто продолжаем поиск
			var appErr *apperrors.AppError
			if errors.As(err, &appErr) && appErr.Code == 404 {
				// Эталон не найден - это нормально, продолжаем
			} else {
				return "", apperrors.WrapError(err, "не удалось выполнить поиск в эталонах")
			}
		}
		if benchmark != nil {
			// Найден эталон, возвращаем каноническое имя
			return benchmark.Name, nil
		}
	}

	// Эталон не найден, используем дорогие AI сервисы
	if ns.normalizer != nil && ns.normalizer.GetAINormalizer() != nil {
		aiNormalizer := ns.normalizer.GetAINormalizer()
		// AINormalizer использует другой интерфейс, нужно использовать nameNormalizer
		// или напрямую через aiNormalizer
		result, err := aiNormalizer.NormalizeWithAI(name)
		if err != nil {
			return "", apperrors.NewInternalError("ошибка AI нормализации", err)
		}
		return result.NormalizedName, nil
	}

	return "", apperrors.NewServiceUnavailableError("сервис нормализации недоступен", nil)
}

// NormalizeCounterparty нормализует контрагента с приоритетом поиска в эталонах
func (ns *NormalizationService) NormalizeCounterparty(name string, inn string, bin string) (string, error) {
	// Сначала ищем в эталонах для контрагентов
	if ns.benchmarkService != nil {
		benchmark, err := ns.benchmarkService.FindBestMatch(name, "counterparty")
		if err != nil {
			// NotFoundError не критична - просто продолжаем поиск
			var appErr *apperrors.AppError
			if errors.As(err, &appErr) && appErr.Code == 404 {
				// Эталон не найден - это нормально, продолжаем
			} else {
				return "", apperrors.WrapError(err, "не удалось выполнить поиск в эталонах")
			}
		}
		if benchmark != nil {
			// Найден эталон, возвращаем каноническое имя
			return benchmark.Name, nil
		}
	}

	// Эталон не найден, используем дорогие AI сервисы
	if ns.normalizer != nil && ns.normalizer.GetAINormalizer() != nil {
		aiNormalizer := ns.normalizer.GetAINormalizer()
		// AINormalizer использует NormalizeWithAI, который работает с одним именем
		// Для контрагентов можно использовать комбинацию имени и ИНН/БИН
		combinedName := name
		if inn != "" {
			combinedName = fmt.Sprintf("%s ИНН:%s", combinedName, inn)
		}
		if bin != "" {
			combinedName = fmt.Sprintf("%s БИН:%s", combinedName, bin)
		}
		result, err := aiNormalizer.NormalizeWithAI(combinedName)
		if err != nil {
			return "", apperrors.NewInternalError("ошибка AI нормализации", err)
		}
		return result.NormalizedName, nil
	}

	return "", apperrors.NewServiceUnavailableError("сервис нормализации недоступен", nil)
}

// StartVersionedNormalization начинает новую сессию версионированной нормализации
func (ns *NormalizationService) StartVersionedNormalization(itemID int, originalName string, getArliaiAPIKey func() string) (map[string]interface{}, error) {
	if ns.db == nil {
		return nil, apperrors.NewServiceUnavailableError("база данных недоступна", nil)
	}

	// Создаем компоненты для пайплайна
	patternDetector := normalization.NewPatternDetector()

	var aiIntegrator *normalization.PatternAIIntegrator
	apiKey := getArliaiAPIKey()
	if apiKey == "" {
		// Fallback на переменную окружения
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey != "" {
		aiNormalizer := normalization.NewAINormalizer(apiKey)
		aiIntegrator = normalization.NewPatternAIIntegrator(patternDetector, aiNormalizer)
	}

	// Создаем пайплайн
	pipeline := normalization.NewVersionedNormalizationPipeline(
		ns.db,
		patternDetector,
		aiIntegrator,
	)
	// Все шаги пайплайна фиксируют актуальное canonical name через database.UpdateNormalizedName.

	// Начинаем сессию
	if err := pipeline.StartSession(itemID, originalName); err != nil {
		return nil, apperrors.NewInternalError("не удалось начать сессию", err)
	}

	return map[string]interface{}{
		"session_id":    pipeline.GetSessionID(),
		"current_name":  pipeline.GetCurrentName(),
		"original_name": originalName,
	}, nil
}

// ApplyPatterns применяет алгоритмические паттерны к сессии
func (ns *NormalizationService) ApplyPatterns(sessionID int, getArliaiAPIKey func() string) (map[string]interface{}, error) {
	if ns.db == nil {
		return nil, apperrors.NewServiceUnavailableError("база данных недоступна", nil)
	}

	// Получаем сессию
	session, err := ns.db.GetNormalizationSession(sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("сессия не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить сессию", err)
	}

	// Создаем пайплайн и восстанавливаем состояние
	patternDetector := normalization.NewPatternDetector()
	var aiIntegrator *normalization.PatternAIIntegrator
	apiKey := getArliaiAPIKey()
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey != "" {
		aiNormalizer := normalization.NewAINormalizer(apiKey)
		aiIntegrator = normalization.NewPatternAIIntegrator(patternDetector, aiNormalizer)
	}

	pipeline := normalization.NewVersionedNormalizationPipeline(
		ns.db,
		patternDetector,
		aiIntegrator,
	)
	// Все шаги пайплайна фиксируют актуальное canonical name через database.UpdateNormalizedName.

	// Восстанавливаем сессию
	if err := pipeline.StartSession(session.CatalogItemID, session.OriginalName); err != nil {
		return nil, apperrors.NewInternalError("не удалось восстановить сессию", err)
	}

	// Применяем паттерны
	if err := pipeline.ApplyPatterns(); err != nil {
		return nil, apperrors.NewInternalError("не удалось применить паттерны", err)
	}

	history, _ := pipeline.GetHistory()

	return map[string]interface{}{
		"session_id":   pipeline.GetSessionID(),
		"current_name": pipeline.GetCurrentName(),
		"stage_count":  len(history),
	}, nil
}

// ApplyAI применяет AI коррекцию к сессии
func (ns *NormalizationService) ApplyAI(sessionID int, useChat bool, context []string, getArliaiAPIKey func() string) (map[string]interface{}, error) {
	if ns.db == nil {
		return nil, apperrors.NewServiceUnavailableError("база данных недоступна", nil)
	}

	// Получаем сессию
	session, err := ns.db.GetNormalizationSession(sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("сессия не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить сессию", err)
	}

	// Создаем пайплайн
	patternDetector := normalization.NewPatternDetector()
	var aiIntegrator *normalization.PatternAIIntegrator
	apiKey := getArliaiAPIKey()
	if apiKey == "" {
		apiKey = os.Getenv("ARLIAI_API_KEY")
	}
	if apiKey == "" {
		return nil, apperrors.NewValidationError("API ключ не установлен", nil)
	}

	aiNormalizer := normalization.NewAINormalizer(apiKey)
	aiIntegrator = normalization.NewPatternAIIntegrator(patternDetector, aiNormalizer)

	pipeline := normalization.NewVersionedNormalizationPipeline(
		ns.db,
		patternDetector,
		aiIntegrator,
	)
	// Все шаги пайплайна фиксируют актуальное canonical name через database.UpdateNormalizedName.

	// Восстанавливаем сессию
	if err := pipeline.StartSession(session.CatalogItemID, session.OriginalName); err != nil {
		return nil, apperrors.NewInternalError("не удалось восстановить сессию", err)
	}

	// Применяем AI коррекцию
	if err := pipeline.ApplyAICorrection(useChat, context...); err != nil {
		return nil, apperrors.NewInternalError("не удалось применить AI коррекцию", err)
	}

	history, _ := pipeline.GetHistory()
	lastStage := history[len(history)-1]

	return map[string]interface{}{
		"session_id":   pipeline.GetSessionID(),
		"current_name": pipeline.GetCurrentName(),
		"stage_count":  len(history),
		"last_stage": map[string]interface{}{
			"type":       lastStage.StageType,
			"confidence": lastStage.Confidence,
			"output":     lastStage.OutputName,
		},
	}, nil
}

// GetSessionHistory получает историю сессии нормализации
func (ns *NormalizationService) GetSessionHistory(sessionID int) (map[string]interface{}, error) {
	if ns.db == nil {
		return nil, apperrors.NewServiceUnavailableError("база данных недоступна", nil)
	}

	session, err := ns.db.GetNormalizationSession(sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("сессия не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить сессию", err)
	}

	history, err := ns.db.GetSessionHistory(sessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить историю сессии", err)
	}

	return map[string]interface{}{
		"session": session,
		"history": history,
	}, nil
}

// RevertStage откатывает сессию к указанной стадии
func (ns *NormalizationService) RevertStage(sessionID, stageIndex int, getArliaiAPIKey func() string) (map[string]interface{}, error) {
	if ns.db == nil {
		return nil, apperrors.NewServiceUnavailableError("база данных недоступна", nil)
	}

	// Получаем сессию для проверки существования
	_, err := ns.db.GetNormalizationSession(sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("сессия не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить сессию", err)
	}

	// Получаем историю стадий
	history, err := ns.db.GetSessionHistory(sessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить историю сессии", err)
	}

	// Проверяем валидность индекса
	if stageIndex < 0 || stageIndex >= len(history) {
		return nil, apperrors.NewValidationError(
			fmt.Sprintf("недопустимый индекс стадии: %d (всего стадий: %d)", stageIndex, len(history)),
			nil,
		)
	}

	// Получаем ID целевой стадии
	targetStageID := history[stageIndex].ID

	// Выполняем откат
	if err := ns.db.RevertToStage(sessionID, targetStageID); err != nil {
		return nil, apperrors.NewInternalError("не удалось откатить сессию", err)
	}

	// Обновляем статус сессии на "reverted"
	if err := ns.db.UpdateSessionStatus(sessionID, "reverted"); err != nil {
		// Логируем, но не возвращаем ошибку - откат уже выполнен
	}

	// Получаем обновленную информацию
	updatedSession, err := ns.db.GetNormalizationSession(sessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить обновленную сессию", err)
	}

	updatedHistory, err := ns.db.GetSessionHistory(sessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить обновленную историю", err)
	}

	return map[string]interface{}{
		"session_id":   sessionID,
		"reverted_to":  stageIndex,
		"current_name": updatedSession.CurrentName,
		"stages_count": updatedSession.StagesCount,
		"status":       updatedSession.Status,
		"session":      updatedSession,
		"history":      updatedHistory,
	}, nil
}

// ApplyCategorization применяет категоризацию к сессии
func (ns *NormalizationService) ApplyCategorization(sessionID int, category string, getArliaiAPIKey func() string) (map[string]interface{}, error) {
	if ns.db == nil {
		return nil, apperrors.NewServiceUnavailableError("база данных недоступна", nil)
	}

	if category == "" {
		return nil, apperrors.NewValidationError("категория не может быть пустой", nil)
	}

	// Получаем сессию
	session, err := ns.db.GetNormalizationSession(sessionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("сессия не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить сессию", err)
	}

	// Получаем историю стадий
	history, err := ns.db.GetSessionHistory(sessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить историю сессии", err)
	}

	if len(history) == 0 {
		return nil, apperrors.NewValidationError("сессия не содержит стадий для категоризации", nil)
	}

	// Получаем последнюю стадию
	lastStage := history[len(history)-1]

	// Обновляем категорию в последней стадии
	// Если категория не была установлена ранее, используем category_folded
	// Иначе обновляем оба поля
	var categoryJSON []byte
	if lastStage.CategoryFolded != "" {
		var existingCategory map[string]interface{}
		if err := json.Unmarshal([]byte(lastStage.CategoryFolded), &existingCategory); err == nil {
			categoryJSON, _ = json.Marshal(map[string]interface{}{
				"category":      category,
				"original_path": existingCategory,
			})
		}
	}
	if categoryJSON == nil {
		categoryJSON, _ = json.Marshal(map[string]interface{}{
			"category": category,
		})
	}

	// Обновляем стадию в БД
	updateQuery := `
		UPDATE normalization_stages 
		SET category_folded = ?,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	if _, err := ns.db.Exec(updateQuery, string(categoryJSON), lastStage.ID); err != nil {
		return nil, apperrors.NewInternalError("не удалось обновить категорию стадии", err)
	}

	// Если есть catalog_item_id, обновляем категорию в catalog_items или normalized_data
	if session.CatalogItemID > 0 {
		// Пытаемся обновить в normalized_data, если есть связь
		updateNormalizedQuery := `
			UPDATE normalized_data 
			SET category = ? 
			WHERE id = (
				SELECT normalized_item_id 
				FROM catalog_items 
				WHERE id = ?
			)
		`
		_, _ = ns.db.Exec(updateNormalizedQuery, category, session.CatalogItemID)
		// Игнорируем ошибку - возможно, запись еще не нормализована
	}

	// Получаем обновленную историю
	updatedHistory, err := ns.db.GetSessionHistory(sessionID)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить обновленную историю", err)
	}

	return map[string]interface{}{
		"session_id":   sessionID,
		"category":     category,
		"current_name": session.CurrentName,
		"stage_id":     lastStage.ID,
		"history":      updatedHistory,
	}, nil
}

// CreateStopCheck создает функцию проверки остановки
func (ns *NormalizationService) CreateStopCheck() func() bool {
	return func() bool {
		ns.normalizerMutex.RLock()
		shouldStop := !ns.normalizerRunning
		ns.normalizerMutex.RUnlock()
		return shouldStop
	}
}

// NormalizationStatus статус нормализации
// Используем алиас к models.NormalizationStatus для единообразия
type NormalizationStatus = models.NormalizationStatus

// GetNormalizationConfig получает конфигурацию нормализации из serviceDB
func (ns *NormalizationService) GetNormalizationConfig() (*database.NormalizationConfig, error) {
	if ns.serviceDB == nil {
		return nil, apperrors.NewServiceUnavailableError("serviceDB недоступна", nil)
	}
	return ns.serviceDB.GetNormalizationConfig()
}

// UpdateNormalizationConfig обновляет конфигурацию нормализации в serviceDB
func (ns *NormalizationService) UpdateNormalizationConfig(databasePath, sourceTable, referenceColumn, codeColumn, nameColumn string) error {
	if ns.serviceDB == nil {
		return apperrors.NewServiceUnavailableError("serviceDB недоступна", nil)
	}
	return ns.serviceDB.UpdateNormalizationConfig(databasePath, sourceTable, referenceColumn, codeColumn, nameColumn)
}
