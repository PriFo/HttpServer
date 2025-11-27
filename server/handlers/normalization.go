package handlers

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"httpserver/database"
	apperrors "httpserver/server/errors"
	"httpserver/server/services"
	"httpserver/server/types"

	_ "github.com/mattn/go-sqlite3"
)

// NormalizationHandler обработчик для работы с нормализацией данных
type NormalizationHandler struct {
	normalizationService   *services.NormalizationService
	clientService          *services.ClientService
	baseHandler            *BaseHandler
	normalizerEvents       <-chan string
	startNormalizationFunc func(clientID, projectID int, options map[string]interface{}) error // Функция для запуска нормализации проекта
	getArliaiAPIKey        func() string                                                       // Функция для получения API ключа Arliai из конфигурации
	// Доступ к базам данных
	db                      *database.DB // Основная БД (содержит normalized_data)
	currentDBPath           string
	normalizedDB            *database.DB // Нормализованная БД (если отличается)
	currentNormalizedDBPath string
}

// NormalizationStartRequest описывает тело запроса на запуск версионированной нормализации.
type NormalizationStartRequest struct {
	ItemID       int    `json:"item_id" example:"1001"`
	OriginalName string `json:"original_name" example:"Труба стальная 20мм"`
}

// NormalizationSessionRequest описывает запросы, требующие идентификатора сессии.
type NormalizationSessionRequest struct {
	SessionID int `json:"session_id" example:"42"`
}

// CompletenessMetrics метрики заполненности данных
type CompletenessMetrics struct {
	NomenclatureCompleteness struct {
		ArticlesPercent      float64 `json:"articles_percent"`
		UnitsPercent         float64 `json:"units_percent"`
		DescriptionsPercent  float64 `json:"descriptions_percent"`
		OverallCompleteness  float64 `json:"overall_completeness"`
	} `json:"nomenclature_completeness,omitempty"`
	CounterpartyCompleteness struct {
		INNPercent          float64 `json:"inn_percent"`
		AddressPercent      float64 `json:"address_percent"`
		ContactsPercent     float64 `json:"contacts_percent"`
		OverallCompleteness float64 `json:"overall_completeness"`
	} `json:"counterparty_completeness,omitempty"`
}

// DatabasePreviewStats статистика по базе данных для предпросмотра
type DatabasePreviewStats struct {
	DatabaseID        int                 `json:"database_id"`
	DatabaseName      string              `json:"database_name"`
	FilePath          string              `json:"file_path"`
	NomenclatureCount int64               `json:"nomenclature_count"`
	CounterpartyCount int64               `json:"counterparty_count"`
	TotalRecords      int64               `json:"total_records"`
	DatabaseSize      int64               `json:"database_size"`
	Error             string              `json:"error,omitempty"`
	IsAccessible      bool                `json:"is_accessible"`
	IsValid           bool                `json:"is_valid"`
	Completeness      *CompletenessMetrics `json:"completeness,omitempty"`
}

// NormalizationAIRequest описывает запросы на применение AI-коррекции.
type NormalizationAIRequest struct {
	SessionID int      `json:"session_id" example:"42"`
	UseChat   bool     `json:"use_chat" example:"false"`
	Context   []string `json:"context,omitempty" example:"\"Предыдущие варианты\""`
}

// NormalizationRevertRequest описывает запрос на откат стадии нормализации.
type NormalizationRevertRequest struct {
	SessionID  int `json:"session_id" example:"42"`
	StageIndex int `json:"stage_index" example:"1"`
}

// NormalizationCategoryRequest описывает запрос с идентификатором сессии и категорией.
type NormalizationCategoryRequest struct {
	SessionID int    `json:"session_id" example:"42"`
	Category  string `json:"category" example:"materials"`
}

// NewNormalizationHandler создает новый обработчик для работы с нормализацией
func NewNormalizationHandler(
	normalizationService *services.NormalizationService,
	baseHandler *BaseHandler,
	normalizerEvents <-chan string,
) *NormalizationHandler {
	return &NormalizationHandler{
		normalizationService: normalizationService,
		baseHandler:          baseHandler,
		normalizerEvents:     normalizerEvents,
	}
}

// NewNormalizationHandlerWithServices создает новый обработчик с полным набором сервисов
func NewNormalizationHandlerWithServices(
	normalizationService *services.NormalizationService,
	clientService *services.ClientService,
	baseHandler *BaseHandler,
	normalizerEvents <-chan string,
	startNormalizationFunc func(clientID, projectID int, options map[string]interface{}) error,
	getArliaiAPIKey func() string, // Функция для получения API ключа Arliai из конфигурации
) *NormalizationHandler {
	return &NormalizationHandler{
		normalizationService:   normalizationService,
		clientService:          clientService,
		baseHandler:            baseHandler,
		normalizerEvents:       normalizerEvents,
		startNormalizationFunc: startNormalizationFunc,
		getArliaiAPIKey:        getArliaiAPIKey,
	}
}

// SetDatabase устанавливает доступ к базам данных
func (h *NormalizationHandler) SetDatabase(db *database.DB, currentDBPath string, normalizedDB *database.DB, currentNormalizedDBPath string) {
	h.db = db
	h.currentDBPath = currentDBPath
	h.normalizedDB = normalizedDB
	h.currentNormalizedDBPath = currentNormalizedDBPath
}

// SetGetArliaiAPIKey устанавливает функцию для получения API ключа Arliai из конфигурации
func (h *NormalizationHandler) SetGetArliaiAPIKey(getArliaiAPIKey func() string) {
	h.getArliaiAPIKey = getArliaiAPIKey
}

// getAPIKey получает API ключ Arliai с fallback на переменную окружения
func (h *NormalizationHandler) getAPIKey() string {
	if h.getArliaiAPIKey != nil {
		if apiKey := h.getArliaiAPIKey(); apiKey != "" {
			return apiKey
		}
	}
	// Fallback на переменную окружения
	return os.Getenv("ARLIAI_API_KEY")
}

// SetStartNormalizationFunc устанавливает функцию для запуска нормализации проекта
func (h *NormalizationHandler) SetStartNormalizationFunc(startNormalizationFunc func(clientID, projectID int, options map[string]interface{}) error) {
	h.startNormalizationFunc = startNormalizationFunc
}

// SetClientService устанавливает сервис клиентов
func (h *NormalizationHandler) SetClientService(clientService *services.ClientService) {
	h.clientService = clientService
}

// getDB получает БД по пути, используя db по умолчанию
func (h *NormalizationHandler) getDB(databasePath string) (*database.DB, error) {
	if databasePath == "" {
		return h.db, nil
	}

	if databasePath != h.currentDBPath {
		db, err := database.NewDB(databasePath)
		if err != nil {
			return nil, apperrors.NewInternalError(fmt.Sprintf("failed to open database %s", databasePath), err)
		}
		return db, nil
	}

	return h.db, nil
}

// HandleNormalizationEvents обрабатывает SSE соединение для событий нормализации
// HandleNormalizationEvents обрабатывает SSE-события нормализации
// @Summary Получить поток событий нормализации
// @Description Возвращает Server-Sent Events (SSE) поток с событиями процесса нормализации в реальном времени.
// @Tags normalization
// @Produce text/event-stream
// @Success 200 {string} string "SSE поток событий"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/normalization/events [get]
func (h *NormalizationHandler) HandleNormalizationEvents(w http.ResponseWriter, r *http.Request) {
	// Обработка паники на верхнем уровне
	defer func() {
		if panicVal := recover(); panicVal != nil {
			slog.Error("[Normalization] Panic in HandleNormalizationEvents",
				"panic", panicVal,
				"stack", string(debug.Stack()),
				"path", r.URL.Path,
			)
			// Если заголовки еще не установлены, отправляем обычный HTTP ответ
			if w.Header().Get("Content-Type") != "text/event-stream" {
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}
	}()

	// Проверяем поддержку Flusher ДО установки заголовков
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Отправляем начальное событие с обработкой ошибок
	if _, err := fmt.Fprintf(w, "data: %s\n\n", "{\"type\":\"connected\",\"message\":\"Connected to normalization events\"}"); err != nil {
		slog.Error("[Normalization] Error sending initial connection message",
			"error", err,
			"path", r.URL.Path,
		)
		return
	}
	flusher.Flush()

	// Слушаем события из канала
	// Heartbeat каждые 10 секунд для предотвращения таймаута (WriteTimeout 60 секунд)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-h.normalizerEvents:
			// Обработка события с защитой от паники
			func() {
				defer func() {
					if panicVal := recover(); panicVal != nil {
						slog.Error("[Normalization] Panic in HandleNormalizationEvents",
							"panic", panicVal,
							"stack", string(debug.Stack()),
							"path", r.URL.Path,
						)
						errorMsg := fmt.Sprintf(`{"error":"internal error processing event","details":"%v"}`, panicVal)
						fmt.Fprintf(w, "data: %s\n\n", errorMsg)
						flusher.Flush()
					}
				}()

				// Форматируем событие как JSON
				eventJSON := fmt.Sprintf("{\"type\":\"log\",\"message\":%q,\"timestamp\":%q}",
					event, time.Now().Format(time.RFC3339))
				if _, err := fmt.Fprintf(w, "data: %s\n\n", eventJSON); err != nil {
					slog.Error("[Normalization] Error sending SSE event", "error", err, "path", r.URL.Path)
					return
				}
				flusher.Flush()
			}()
		case <-ticker.C:
			// Отправляем heartbeat для поддержания соединения
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				slog.Error("[Normalization] Error sending heartbeat", "error", err, "path", r.URL.Path)
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			// Клиент отключился
			slog.Info("[Normalization] Client disconnected", "error", r.Context().Err(), "path", r.URL.Path)
			return
		}
	}
}

// HandleNormalizationStatus возвращает текущий статус нормализации
// @Summary Получить общий статус нормализации
// @Description Возвращает агрегированную информацию о текущем состоянии нормализационной сессии.
// @Tags normalization
// @Produce json
// @Success 200 {object} types.NormalizationStatus
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/status [get]
func (h *NormalizationHandler) HandleNormalizationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	status := h.normalizationService.GetStatus()

	// Получаем общее количество элементов для нормализации из БД
	var total int
	if h.db != nil {
		err := h.db.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&total)
		if err != nil {
			// Если не удалось получить из БД, используем 0
			total = 0
		}
	}

	// Рассчитываем прогресс в процентах
	var progress float64
	if total > 0 && status.Processed > 0 {
		progress = float64(status.Processed) / float64(total) * 100
		if progress > 100 {
			progress = 100
		}
	}

	response := types.NormalizationStatus{
		IsRunning:   status.IsRunning,
		Progress:    progress,
		Processed:   status.Processed,
		Total:       total,
		Success:     status.Success,
		Errors:      status.Errors,
		CurrentStep: "Не запущено",
		Logs:        []string{},
	}

	if status.IsRunning {
		response.CurrentStep = "Выполняется нормализация..."
		if status.StartTime != "" {
			response.StartTime = status.StartTime
			response.ElapsedTime = status.ElapsedTime
			// Парсим ElapsedTime для расчета rate
			if elapsed, err := time.ParseDuration(status.ElapsedTime); err == nil && elapsed.Seconds() > 0 {
				response.Rate = float64(status.Processed) / elapsed.Seconds()
			}
		}
	} else if status.Processed > 0 {
		response.CurrentStep = "Нормализация завершена"
		if status.StartTime != "" {
			response.StartTime = status.StartTime
			response.ElapsedTime = status.ElapsedTime
		}
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleStartClientProjectNormalization запускает нормализацию для конкретного проекта клиента
// @Summary Запустить нормализацию проекта клиента
// @Description Запускает процесс нормализации контрагентов для заданного клиента и проекта.
// @Tags normalization
// @Accept json
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Param payload body map[string]interface{} false "Дополнительные опции запуска"
// @Success 200 {object} map[string]interface{} "Статус запуска"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Клиент или проект не найдены"
// @Failure 500 {object} ErrorResponse "Не удалось запустить нормализацию"
// @Router /api/clients/{clientId}/projects/{projectId}/normalization/start [post]
func (h *NormalizationHandler) HandleStartClientProjectNormalization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Извлекаем clientId и projectId из контекста (установлены в Gin wrapper)
	// PathValue не работает с Gin router, поэтому используем только контекст
	var clientIDStr, projectIDStr string
	if ctxClientID := r.Context().Value("clientId"); ctxClientID != nil {
		clientIDStr = fmt.Sprintf("%v", ctxClientID)
	}
	if ctxProjectID := r.Context().Value("projectId"); ctxProjectID != nil {
		projectIDStr = fmt.Sprintf("%v", ctxProjectID)
	}

	if clientIDStr == "" || projectIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "clientId and projectId are required in URL path", http.StatusBadRequest)
		return
	}

	var clientID, projectID int
	var err error
	if clientID, err = strconv.Atoi(clientIDStr); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid clientId: %s", clientIDStr), http.StatusBadRequest)
		return
	}
	if projectID, err = strconv.Atoi(projectIDStr); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid projectId: %s", projectIDStr), http.StatusBadRequest)
		return
	}

	// Валидируем существование клиента и проекта
	if h.clientService != nil {
		ctx := r.Context()
		client, err := h.clientService.GetClient(ctx, clientID)
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Client not found: %v", err), http.StatusNotFound)
			return
		}
		if client == nil {
			h.baseHandler.WriteJSONError(w, r, "Client not found", http.StatusNotFound)
			return
		}

		project, err := h.clientService.GetClientProject(ctx, clientID, projectID)
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Project not found: %v", err), http.StatusNotFound)
			return
		}
		if project == nil {
			h.baseHandler.WriteJSONError(w, r, "Project not found", http.StatusNotFound)
			return
		}
	}

	// Читаем опции из тела запроса
	var options map[string]interface{}
	if r.Body != nil {
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&options); err != nil && err.Error() != "EOF" {
			// Если тело не пустое и не удалось распарсить, возвращаем ошибку
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}
	}
	if options == nil {
		options = make(map[string]interface{})
	}

	// Запускаем нормализацию через функцию от Server
	if h.startNormalizationFunc != nil {
		if err := h.startNormalizationFunc(clientID, projectID, options); err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to start normalization: %v", err), http.StatusInternalServerError)
			return
		}
	} else {
		h.baseHandler.WriteJSONError(w, r, "Normalization start function not available", http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success":    true,
		"message":    "Normalization started for project",
		"client_id":  clientID,
		"project_id": projectID,
	}, http.StatusOK)
}

// HandleGetClientProjectNormalizationStatus возвращает статус нормализации для конкретного проекта
// @Summary Получить статус нормализации проекта клиента
// @Description Возвращает прогресс и состояние текущей сессии нормализации.
// @Tags normalization
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Success 200 {object} types.NormalizationStatus
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Клиент или проект не найдены"
// @Router /api/clients/{clientId}/projects/{projectId}/normalization/status [get]
func (h *NormalizationHandler) HandleGetClientProjectNormalizationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Извлекаем clientId и projectId из контекста (установлены в Gin wrapper)
	// PathValue не работает с Gin router, поэтому используем только контекст
	var clientIDStr, projectIDStr string
	if ctxClientID := r.Context().Value("clientId"); ctxClientID != nil {
		clientIDStr = fmt.Sprintf("%v", ctxClientID)
	}
	if ctxProjectID := r.Context().Value("projectId"); ctxProjectID != nil {
		projectIDStr = fmt.Sprintf("%v", ctxProjectID)
	}

	if clientIDStr == "" || projectIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "clientId and projectId are required in URL path", http.StatusBadRequest)
		return
	}

	var clientID, projectID int
	var err error
	if clientID, err = strconv.Atoi(clientIDStr); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid clientId: %s", clientIDStr), http.StatusBadRequest)
		return
	}
	if projectID, err = strconv.Atoi(projectIDStr); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid projectId: %s", projectIDStr), http.StatusBadRequest)
		return
	}

	// Валидируем существование проекта
	if h.clientService != nil {
		ctx := r.Context()
		project, err := h.clientService.GetClientProject(ctx, clientID, projectID)
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Project not found: %v", err), http.StatusNotFound)
			return
		}
		if project == nil {
			h.baseHandler.WriteJSONError(w, r, "Project not found", http.StatusNotFound)
			return
		}
	}

	// Получаем статус нормализации (пока глобальный, в будущем можно фильтровать по проекту)
	status := h.normalizationService.GetStatus()

	// Рассчитываем прогресс
	progress := 0.0
	total := status.Processed + status.Success + status.Errors
	if total > 0 {
		progress = float64(status.Processed) / float64(total) * 100
	}

	response := types.NormalizationStatus{
		IsRunning:   status.IsRunning,
		Progress:    progress,
		Processed:   status.Processed,
		Total:       total,
		Success:     status.Success,
		Errors:      status.Errors,
		CurrentStep: "Не запущено",
		Logs:        []string{},
	}

	if status.IsRunning {
		response.CurrentStep = "Выполняется нормализация..."
		if status.StartTime != "" {
			response.StartTime = status.StartTime
			response.ElapsedTime = status.ElapsedTime
			// Парсим ElapsedTime для расчета rate
			if elapsed, err := time.ParseDuration(status.ElapsedTime); err == nil && elapsed.Seconds() > 0 {
				response.Rate = float64(status.Processed) / elapsed.Seconds()
			}
		}
	} else if status.Processed > 0 {
		response.CurrentStep = "Нормализация завершена"
		if status.StartTime != "" {
			response.StartTime = status.StartTime
			response.ElapsedTime = status.ElapsedTime
		}
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleNormalizationStop останавливает процесс нормализации
// @Summary Остановить нормализацию
// @Description Останавливает текущий процесс нормализации и возвращает статус операции.
// @Tags normalization
// @Produce json
// @Success 200 {object} map[string]interface{} "Статус остановки с полем was_running"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/stop [post]
func (h *NormalizationHandler) HandleNormalizationStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	wasRunning := h.normalizationService.Stop()

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success":     true,
		"message":     "Normalization stopped",
		"was_running": wasRunning,
	}, http.StatusOK)
}

// HandleNormalizationStats возвращает статистику нормализации
// @Summary Получить статистику нормализации
// @Description Возвращает агрегированную статистику по процессу нормализации: количество обработанных записей, групп, успешных операций и ошибок.
// @Tags normalization
// @Produce json
// @Success 200 {object} map[string]interface{} "Статистика нормализации"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/stats [get]
func (h *NormalizationHandler) HandleNormalizationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	databasePath := query.Get("database")
	categoryFilter := strings.TrimSpace(query.Get("category_filter"))
	if categoryFilter == "" {
		categoryFilter = "Номенклатура"
	}

	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем статистику из normalized_data
	var totalItems int
	var totalItemsWithAttributes int
	var lastNormalizedAt sql.NullString
	categoryStats := make(map[string]int)

	// Считаем все исправленные элементы
	err = db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&totalItems)
	if err != nil {
		totalItems = 0
	}

	// Считаем количество элементов с атрибутами
	err = db.QueryRow(`
		SELECT COUNT(DISTINCT normalized_item_id) 
		FROM normalized_item_attributes
	`).Scan(&totalItemsWithAttributes)
	if err != nil {
		totalItemsWithAttributes = 0
	}

	// Получаем время последней нормализации
	err = db.QueryRow("SELECT MAX(created_at) FROM normalized_data").Scan(&lastNormalizedAt)
	if err != nil {
		// Игнорируем ошибку
	}

	// Получаем статистику по категориям
	rows, err := db.Query(`
		SELECT 
			category,
			COUNT(*) as count
		FROM normalized_data
		WHERE category IS NOT NULL AND category != ''
		GROUP BY category
		ORDER BY count DESC
	`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int
			if err := rows.Scan(&category, &count); err != nil {
				continue
			}
			categoryStats[category] = count
		}
	}

	// Вычисляем количество объединенных элементов
	var uniqueGroups int
	if err := db.QueryRow("SELECT COUNT(DISTINCT normalized_reference) FROM normalized_data").Scan(&uniqueGroups); err != nil {
		uniqueGroups = 0
	}
	mergedItems := totalItems - uniqueGroups
	if mergedItems < 0 {
		mergedItems = 0
	}

	// Вычисляем количество уникальных групп номенклатур (по фильтру категории)
	filterPattern := "%" + strings.ToLower(categoryFilter) + "%"
	nomenclatureGroupsQuery := `
		SELECT COUNT(*) FROM (
			SELECT normalized_name, category
			FROM normalized_data
			WHERE category IS NOT NULL AND category != ''
			  AND normalized_name IS NOT NULL AND normalized_name != ''
			  AND LOWER(category) LIKE ?
			GROUP BY normalized_name, category
		)
	`
	var nomenclatureGroups int
	if err := db.QueryRow(nomenclatureGroupsQuery, filterPattern).Scan(&nomenclatureGroups); err != nil {
		nomenclatureGroups = 0
	}

	stats := map[string]interface{}{
		"totalItems":               totalItems,
		"totalItemsWithAttributes": totalItemsWithAttributes,
		"totalGroups":              uniqueGroups,
		"categories":               categoryStats,
		"mergedItems":              mergedItems,
		"categoryFilter":           categoryFilter,
		"nomenclatureGroups":       nomenclatureGroups,
	}

	if lastNormalizedAt.Valid && lastNormalizedAt.String != "" {
		stats["last_normalized_at"] = lastNormalizedAt.String
	}

	h.baseHandler.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleNormalizationGroups возвращает группы нормализованных данных
// @Summary Получить группы нормализованных данных
// @Description Возвращает список всех групп нормализованных данных с агрегированной информацией по каждой группе.
// @Tags normalization
// @Produce json
// @Param database query string false "Путь к базе данных"
// @Param category query string false "Фильтр по категории"
// @Success 200 {object} map[string]interface{} "Список групп нормализованных данных"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/groups [get]
func (h *NormalizationHandler) HandleNormalizationGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметр database или db_id из query
	databasePath := r.URL.Query().Get("database")
	dbIDStr := r.URL.Query().Get("db_id")
	
	// Если передан db_id, получаем путь через clientService
	if dbIDStr != "" && h.clientService != nil {
		dbID, err := strconv.Atoi(dbIDStr)
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid db_id: %v", err), http.StatusBadRequest)
			return
		}
		
		// Получаем clientId и projectId из контекста (устанавливаются в wrapper)
		clientID, _ := r.Context().Value("clientId").(int)
		projectID, _ := r.Context().Value("projectId").(int)
		
		if clientID > 0 && projectID > 0 {
			// Получаем информацию о базе данных проекта
			projectDB, err := h.clientService.GetProjectDatabase(r.Context(), clientID, projectID, dbID)
			if err != nil {
				h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get project database: %v", err), http.StatusNotFound)
				return
			}
			databasePath = projectDB.FilePath
		}
	}
	
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем параметры запроса
	query := r.URL.Query()
	category := query.Get("category")
	search := query.Get("search")
	kpvedCode := query.Get("kpved_code")
	includeAI := query.Get("include_ai") == "true"

	// Валидация параметров пагинации
	page, err := ValidateIntParam(r, "page", 1, 1, 1000)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	limit, err := ValidateIntParam(r, "limit", 20, 1, 100)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, err.Error(), http.StatusBadRequest)
		return
	}
	offset := (page - 1) * limit

	// Строим SQL запрос
	baseQuery := `
		SELECT normalized_name, normalized_reference, category, COUNT(*) as merged_count`
	if includeAI {
		baseQuery += `, AVG(COALESCE(NULLIF(quality_score, 0), NULLIF(ai_confidence, 0))) as avg_confidence, MAX(processing_level) as processing_level`
	}
	baseQuery += `, MAX(kpved_code) as kpved_code, MAX(kpved_name) as kpved_name, AVG(kpved_confidence) as kpved_confidence`
	baseQuery += `, MAX(created_at) as last_normalized_at`
	baseQuery += ` FROM normalized_data WHERE 1=1`

	countQuery := `SELECT COUNT(*) FROM (SELECT normalized_name, category FROM normalized_data WHERE 1=1`

	var args []interface{}
	var countArgs []interface{}

	if category != "" {
		baseQuery += " AND category = ?"
		countQuery += " AND category = ?"
		args = append(args, category)
		countArgs = append(countArgs, category)
	}

	if search != "" {
		baseQuery += " AND normalized_name LIKE ?"
		countQuery += " AND normalized_name LIKE ?"
		searchParam := "%" + search + "%"
		args = append(args, searchParam)
		countArgs = append(countArgs, searchParam)
	}

	if kpvedCode != "" {
		baseQuery += " AND kpved_code = ?"
		countQuery += " AND kpved_code = ?"
		args = append(args, kpvedCode)
		countArgs = append(countArgs, kpvedCode)
	}

	baseQuery += " GROUP BY normalized_name, normalized_reference, category"
	baseQuery += " ORDER BY merged_count DESC, normalized_name ASC"
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	countQuery += " GROUP BY normalized_name, category)"

	// Получаем общее количество групп
	var totalGroups int
	err = db.QueryRow(countQuery, countArgs...).Scan(&totalGroups)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Failed to count groups", http.StatusInternalServerError)
		return
	}

	// Получаем группы
	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Failed to fetch groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Group struct {
		NormalizedName      string   `json:"normalized_name"`
		NormalizedReference string   `json:"normalized_reference"`
		Category            string   `json:"category"`
		MergedCount         int      `json:"merged_count"`
		AvgConfidence       *float64 `json:"avg_confidence,omitempty"`
		ProcessingLevel     *string  `json:"processing_level,omitempty"`
		KpvedCode           *string  `json:"kpved_code,omitempty"`
		KpvedName           *string  `json:"kpved_name,omitempty"`
		KpvedConfidence     *float64 `json:"kpved_confidence,omitempty"`
		LastNormalizedAt    *string  `json:"last_normalized_at,omitempty"`
	}

	groups := []Group{}
	for rows.Next() {
		var g Group
		var lastNormalizedAt sql.NullString
		if includeAI {
			if err := rows.Scan(&g.NormalizedName, &g.NormalizedReference, &g.Category, &g.MergedCount,
				&g.AvgConfidence, &g.ProcessingLevel, &g.KpvedCode, &g.KpvedName, &g.KpvedConfidence, &lastNormalizedAt); err != nil {
				continue
			}
		} else {
			if err := rows.Scan(&g.NormalizedName, &g.NormalizedReference, &g.Category, &g.MergedCount,
				&g.KpvedCode, &g.KpvedName, &g.KpvedConfidence, &lastNormalizedAt); err != nil {
				continue
			}
		}
		if lastNormalizedAt.Valid && lastNormalizedAt.String != "" {
			g.LastNormalizedAt = &lastNormalizedAt.String
		}
		groups = append(groups, g)
	}

	totalPages := (totalGroups + limit - 1) / limit

	response := map[string]interface{}{
		"groups":     groups,
		"total":      totalGroups,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleNormalizationGroupItems возвращает элементы группы нормализованных данных
// @Summary Получить элементы группы нормализованных данных
// @Description Возвращает все исходные записи, объединенные в указанную группу.
// @Tags normalization
// @Produce json
// @Param database query string false "Путь к базе данных"
// @Param normalized_name query string true "Нормализованное название группы"
// @Param category query string true "Категория группы"
// @Success 200 {object} map[string]interface{} "Список элементов группы"
// @Failure 400 {object} ErrorResponse "Отсутствуют обязательные параметры"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/group-items [get]
func (h *NormalizationHandler) HandleNormalizationGroupItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметр database из query
	databasePath := r.URL.Query().Get("database")
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем параметры запроса
	query := r.URL.Query()
	normalizedName := query.Get("normalized_name")
	category := query.Get("category")
	includeAI := query.Get("include_ai") == "true"

	if normalizedName == "" || category == "" {
		h.baseHandler.WriteJSONError(w, r, "normalized_name and category are required", http.StatusBadRequest)
		return
	}

	// Запрос для получения всех исходных записей группы
	sqlQuery := `
		SELECT id, source_reference, source_name, code,
		       normalized_name, normalized_reference, category,
		       merged_count, created_at`
	if includeAI {
		sqlQuery += `, ai_confidence, ai_reasoning, processing_level`
	}
	sqlQuery += `, kpved_code, kpved_name, kpved_confidence`
	sqlQuery += ` FROM normalized_data WHERE normalized_name = ? AND category = ? ORDER BY source_name`

	rows, err := db.Query(sqlQuery, normalizedName, category)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, "Failed to fetch group items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}
	for rows.Next() {
		item := make(map[string]interface{})
		var id, mergedCount int
		var sourceRef, sourceName, code, normName, normRef, cat, createdAt string
		var kpvedCode, kpvedName sql.NullString
		var kpvedConf sql.NullFloat64

		if includeAI {
			var aiConf sql.NullFloat64
			var aiReasoning, procLevel sql.NullString
			if err := rows.Scan(&id, &sourceRef, &sourceName, &code, &normName, &normRef, &cat, &mergedCount, &createdAt,
				&aiConf, &aiReasoning, &procLevel, &kpvedCode, &kpvedName, &kpvedConf); err != nil {
				continue
			}
			item["id"] = id
			item["source_reference"] = sourceRef
			item["source_name"] = sourceName
			item["code"] = code
			item["normalized_name"] = normName
			item["normalized_reference"] = normRef
			item["category"] = cat
			item["merged_count"] = mergedCount
			item["created_at"] = createdAt
			if aiConf.Valid {
				item["ai_confidence"] = aiConf.Float64
			}
			if aiReasoning.Valid {
				item["ai_reasoning"] = aiReasoning.String
			}
			if procLevel.Valid {
				item["processing_level"] = procLevel.String
			}
		} else {
			if err := rows.Scan(&id, &sourceRef, &sourceName, &code, &normName, &normRef, &cat, &mergedCount, &createdAt,
				&kpvedCode, &kpvedName, &kpvedConf); err != nil {
				continue
			}
			item["id"] = id
			item["source_reference"] = sourceRef
			item["source_name"] = sourceName
			item["code"] = code
			item["normalized_name"] = normName
			item["normalized_reference"] = normRef
			item["category"] = cat
			item["merged_count"] = mergedCount
			item["created_at"] = createdAt
		}

		if kpvedCode.Valid {
			item["kpved_code"] = kpvedCode.String
		}
		if kpvedName.Valid {
			item["kpved_name"] = kpvedName.String
		}
		if kpvedConf.Valid {
			item["kpved_confidence"] = kpvedConf.Float64
		}

		items = append(items, item)
	}

	response := map[string]interface{}{
		"items": items,
		"total": len(items),
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleStartVersionedNormalization обрабатывает запросы к /api/normalization/start
// @Summary Запустить версионированную нормализацию
// @Description Создает новую сессию нормализации для указанного элемента и возвращает session_id.
// @Tags normalization
// @Accept json
// @Produce json
// @Param payload body NormalizationStartRequest true "Параметры запуска нормализации"
// @Success 200 {object} map[string]interface{} "Информация о сессии"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/start [post]
func (h *NormalizationHandler) HandleStartVersionedNormalization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req NormalizationStartRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ItemID == 0 || req.OriginalName == "" {
		h.baseHandler.WriteJSONError(w, r, "item_id and original_name are required", http.StatusBadRequest)
		return
	}

	// Получаем API ключ из конфигурации
	result, err := h.normalizationService.StartVersionedNormalization(req.ItemID, req.OriginalName, h.getAPIKey)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to start normalization: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleApplyPatterns обрабатывает запросы к /api/normalization/apply-patterns
// @Summary Применить алгоритмические паттерны
// @Description Применяет корректирующие паттерны к текущей сессии нормализации и сохраняет результат.
// @Tags normalization
// @Accept json
// @Produce json
// @Param payload body NormalizationSessionRequest true "Идентификатор сессии"
// @Success 200 {object} map[string]interface{} "Обновленная информация о сессии"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/apply-patterns [post]
func (h *NormalizationHandler) HandleApplyPatterns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req NormalizationSessionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Получаем API ключ из конфигурации
	result, err := h.normalizationService.ApplyPatterns(req.SessionID, h.getAPIKey)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to apply patterns: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleApplyAI обрабатывает запросы к /api/normalization/apply-ai
// @Summary Применить AI-коррекцию
// @Description Запускает AI-коррекцию для текущей сессии нормализации и фиксирует результат.
// @Tags normalization
// @Accept json
// @Produce json
// @Param payload body NormalizationAIRequest true "Параметры AI-коррекции"
// @Success 200 {object} map[string]interface{} "Обновленная информация о сессии"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/apply-ai [post]
func (h *NormalizationHandler) HandleApplyAI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req NormalizationAIRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Получаем API ключ из конфигурации
	result, err := h.normalizationService.ApplyAI(req.SessionID, req.UseChat, req.Context, h.getAPIKey)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to apply AI: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleGetSessionHistory обрабатывает запросы к /api/normalization/history
// @Summary Получить историю сессии нормализации
// @Description Возвращает стадии нормализации и список примененных шагов для указанной сессии.
// @Tags normalization
// @Produce json
// @Param session_id query int true "Идентификатор сессии"
// @Success 200 {object} map[string]interface{} "История стадий"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/history [get]
func (h *NormalizationHandler) HandleGetSessionHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "session_id is required", http.StatusBadRequest)
		return
	}

	sessionID, err := ValidateIntPathParam(sessionIDStr, "session_id")
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid session_id: %v", err), http.StatusBadRequest)
		return
	}

	result, err := h.normalizationService.GetSessionHistory(sessionID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get history: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleRevertStage обрабатывает запросы к /api/normalization/revert
// @Summary Откатить стадию нормализации
// @Description Откатывает указанную стадию и восстанавливает предыдущее значение в normalized_data.
// @Tags normalization
// @Accept json
// @Produce json
// @Param payload body NormalizationRevertRequest true "Параметры отката"
// @Success 200 {object} map[string]interface{} "Актуальное состояние сессии"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/revert [post]
func (h *NormalizationHandler) HandleRevertStage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req NormalizationRevertRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.normalizationService.RevertStage(req.SessionID, req.StageIndex, h.getAPIKey)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to revert stage: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleApplyCategorization обрабатывает запросы к /api/normalization/apply-categorization
// @Summary Применить категорию для сессии нормализации
// @Description Позволяет вручную задать категорию для результатов нормализации в рамках указанной сессии.
// @Tags normalization
// @Accept json
// @Produce json
// @Param payload body NormalizationCategoryRequest true "Идентификатор сессии и категория"
// @Success 200 {object} map[string]interface{} "Обновленная информация"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/apply-categorization [post]
func (h *NormalizationHandler) HandleApplyCategorization(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req NormalizationCategoryRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.normalizationService.ApplyCategorization(req.SessionID, req.Category, h.getAPIKey)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to apply categorization: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleNormalizeStart обрабатывает запросы к /api/normalize/start
// Использует HandleStartClientProjectNormalization для обратной совместимости
// HandleNormalizeStart - legacy wrapper для HandleStartClientProjectNormalization
// @Summary (Legacy) Запустить нормализацию для проекта клиента
// @Description Устаревший метод. Рекомендуется использовать POST /api/clients/:clientId/projects/:projectId/normalization/start
// @Tags normalization
// @Deprecated
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Результат запуска нормализации"
// @Failure 400 {object} ErrorResponse "Некорректные данные запроса"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalize/start [post]
func (h *NormalizationHandler) HandleNormalizeStart(w http.ResponseWriter, r *http.Request) {
	// Перенаправляем на HandleStartClientProjectNormalization для обратной совместимости
	h.HandleStartClientProjectNormalization(w, r)
}

// HandleNormalizationItemAttributes обрабатывает запросы к /api/normalization/item-attributes/{id}
// @Summary Получить атрибуты элемента нормализации
// @Description Возвращает детальную информацию об атрибутах указанного элемента нормализации по его ID.
// @Tags normalization
// @Produce json
// @Param id path int true "ID элемента нормализации"
// @Success 200 {object} map[string]interface{} "Атрибуты элемента"
// @Failure 400 {object} ErrorResponse "Некорректный ID"
// @Failure 404 {object} ErrorResponse "Элемент не найден"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/item-attributes/{id} [get]
func (h *NormalizationHandler) HandleNormalizationItemAttributes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Извлекаем ID из пути /api/normalization/item-attributes/{id}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		h.baseHandler.WriteJSONError(w, r, "Item ID is required", http.StatusBadRequest)
		return
	}

	itemIDStr := parts[len(parts)-1]
	itemID, err := strconv.Atoi(itemIDStr)
	if err != nil || itemID <= 0 {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid item ID: %s", itemIDStr), http.StatusBadRequest)
		return
	}

	// Получаем параметр database из query
	databasePath := r.URL.Query().Get("database")
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем атрибуты
	attributes, err := db.GetItemAttributes(itemID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to fetch attributes: %v", err), http.StatusInternalServerError)
		return
	}

	// Преобразуем в JSON-совместимый формат
	attrsJSON := make([]map[string]interface{}, 0, len(attributes))
	for _, attr := range attributes {
		attrsJSON = append(attrsJSON, map[string]interface{}{
			"id":              attr.ID,
			"attribute_type":  attr.AttributeType,
			"attribute_name":  attr.AttributeName,
			"attribute_value": attr.AttributeValue,
			"unit":            attr.Unit,
			"original_text":   attr.OriginalText,
			"confidence":      attr.Confidence,
			"created_at":      attr.CreatedAt,
		})
	}

	response := map[string]interface{}{
		"item_id":    itemID,
		"attributes": attrsJSON,
		"count":      len(attrsJSON),
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleNormalizationExportGroup обрабатывает запросы к /api/normalization/export-group
// @Summary Экспортировать группу нормализованных данных
// @Description Экспортирует указанную группу нормализованных данных в выбранном формате (CSV, JSON и т.д.).
// @Tags normalization
// @Produce application/octet-stream
// @Param normalized_name query string true "Нормализованное название группы"
// @Param category query string true "Категория группы"
// @Param format query string false "Формат экспорта (csv, json)"
// @Param database query string false "Путь к базе данных"
// @Success 200 {file} file "Экспортированные данные"
// @Failure 400 {object} ErrorResponse "Отсутствуют обязательные параметры"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/export-group [get]
func (h *NormalizationHandler) HandleNormalizationExportGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	normalizedName := query.Get("normalized_name")
	category := query.Get("category")
	format := query.Get("format")

	if normalizedName == "" || category == "" {
		h.baseHandler.WriteJSONError(w, r, "normalized_name and category are required", http.StatusBadRequest)
		return
	}

	// По умолчанию CSV формат
	if format == "" {
		format = "csv"
	}

	// Получаем параметр database из query
	databasePath := query.Get("database")
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем данные группы
	sqlQuery := `
		SELECT id, source_reference, source_name, code,
		       normalized_name, normalized_reference, category,
		       created_at, ai_confidence, ai_reasoning, processing_level
		FROM normalized_data
		WHERE normalized_name = ? AND category = ?
		ORDER BY source_name
	`

	rows, err := db.Query(sqlQuery, normalizedName, category)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to fetch group items: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ExportItem struct {
		ID                  int       `json:"id"`
		Code                string    `json:"code"`
		SourceName          string    `json:"source_name"`
		SourceReference     string    `json:"source_reference"`
		NormalizedName      string    `json:"normalized_name"`
		NormalizedReference string    `json:"normalized_reference"`
		Category            string    `json:"category"`
		CreatedAt           time.Time `json:"created_at"`
		AIConfidence        *float64  `json:"ai_confidence,omitempty"`
		AIReasoning         *string   `json:"ai_reasoning,omitempty"`
		ProcessingLevel     *string   `json:"processing_level,omitempty"`
	}

	items := []ExportItem{}
	for rows.Next() {
		var item ExportItem
		if err := rows.Scan(
			&item.ID,
			&item.SourceReference,
			&item.SourceName,
			&item.Code,
			&item.NormalizedName,
			&item.NormalizedReference,
			&item.Category,
			&item.CreatedAt,
			&item.AIConfidence,
			&item.AIReasoning,
			&item.ProcessingLevel,
		); err != nil {
			slog.Warn("Error scanning export item", "error", err)
			continue
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Error iterating records: %v", err), http.StatusInternalServerError)
		return
	}

	// Формируем имя файла
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("group_%s_%s_%s.%s", normalizedName, category, timestamp, format)

	if format == "csv" {
		// Экспорт в CSV
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		// UTF-8 BOM для корректного отображения в Excel
		w.Write([]byte{0xEF, 0xBB, 0xBF})

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Заголовки
		headers := []string{
			"ID", "Код", "Исходное название", "Исходный reference",
			"Нормализованное название", "Нормализованный reference",
			"Категория", "AI Confidence", "Processing Level", "Дата создания",
		}
		writer.Write(headers)

		// Данные
		for _, item := range items {
			confidence := ""
			if item.AIConfidence != nil {
				confidence = fmt.Sprintf("%.2f", *item.AIConfidence)
			}

			processingLevel := ""
			if item.ProcessingLevel != nil {
				processingLevel = *item.ProcessingLevel
			}

			record := []string{
				fmt.Sprintf("%d", item.ID),
				item.Code,
				item.SourceName,
				item.SourceReference,
				item.NormalizedName,
				item.NormalizedReference,
				item.Category,
				confidence,
				processingLevel,
				item.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			writer.Write(record)
		}
	} else if format == "json" {
		// Экспорт в JSON
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		exportData := map[string]interface{}{
			"group_name":  normalizedName,
			"category":    category,
			"export_date": time.Now().Format(time.RFC3339),
			"item_count":  len(items),
			"items":       items,
		}

		json.NewEncoder(w).Encode(exportData)
	} else {
		h.baseHandler.WriteJSONError(w, r, "Invalid format. Supported formats: csv, json", http.StatusBadRequest)
	}
}

// HandlePipelineStats обрабатывает запросы к /api/normalization/pipeline/stats
// Возвращает статистику pipeline нормализации
// @Summary Получить статистику pipeline нормализации
// @Description Возвращает детальную статистику по этапам pipeline нормализации, включая информацию о стадиях обработки.
// @Tags normalization
// @Produce json
// @Success 200 {object} map[string]interface{} "Статистика pipeline"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 503 {object} ErrorResponse "Сервис недоступен"
// @Router /api/normalization/pipeline/stats [get]
func (h *NormalizationHandler) HandlePipelineStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметр database из query
	databasePath := r.URL.Query().Get("database")
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusServiceUnavailable)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем расширенную статистику этапов из normalized_data
	stats, err := database.GetStageProgress(db)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get pipeline stats: %v", err), http.StatusInternalServerError)
		return
	}

	// Добавляем текущий статус выполнения, если сервис доступен
	if h.normalizationService != nil {
		status := h.normalizationService.GetStatus()
		stats["is_running"] = status.IsRunning
		stats["processed"] = status.Processed
		stats["success"] = status.Success
		stats["errors"] = status.Errors
		stats["start_time"] = status.StartTime
		stats["elapsed_time"] = status.ElapsedTime
		if status.Processed > 0 {
			successRate := float64(status.Success) / float64(status.Processed) * 100
			errorRate := float64(status.Errors) / float64(status.Processed) * 100
			stats["success_rate"] = successRate
			stats["error_rate"] = errorRate
		}
	}

	h.baseHandler.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleStageDetails обрабатывает запросы к /api/normalization/pipeline/stage-details
// Возвращает детали текущего этапа нормализации
// @Summary Получить детали текущего этапа нормализации
// @Description Возвращает подробную информацию о текущем этапе pipeline нормализации, включая прогресс и статус обработки.
// @Tags normalization
// @Produce json
// @Success 200 {object} map[string]interface{} "Детали этапа нормализации"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 503 {object} ErrorResponse "Сервис недоступен"
// @Router /api/normalization/pipeline/stage-details [get]
func (h *NormalizationHandler) HandleStageDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	if h.normalizationService == nil {
		h.baseHandler.WriteJSONError(w, r, "Normalization service is not available", http.StatusServiceUnavailable)
		return
	}

	// Получаем статус нормализации
	status := h.normalizationService.GetStatus()

	// Определяем текущий этап на основе статуса
	stage := "idle"
	currentStep := "waiting"
	if status.IsRunning {
		stage = "processing"
		currentStep = "normalizing"
		if status.Processed > 0 {
			currentStep = "processing_records"
		}
	}

	// Формируем детали этапа
	response := map[string]interface{}{
		"stage":        stage,
		"current_step": currentStep,
		"is_running":   status.IsRunning,
		"processed":    status.Processed,
		"success":      status.Success,
		"errors":       status.Errors,
		"start_time":   status.StartTime,
		"elapsed_time": status.ElapsedTime,
	}

	// Добавляем прогресс, если есть обработанные элементы
	if status.Processed > 0 {
		response["progress"] = map[string]interface{}{
			"total":   status.Processed,
			"success": status.Success,
			"errors":  status.Errors,
		}
		// Вычисляем процент успешных операций
		if status.Processed > 0 {
			successRate := float64(status.Success) / float64(status.Processed) * 100
			response["success_rate"] = successRate
		}
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleExport обрабатывает запросы к /api/normalization/export
// Реализует общий экспорт нормализованных данных с возможностью фильтрации
// @Summary Экспортировать нормализованные данные
// @Description Экспортирует нормализованные данные с возможностью фильтрации по категории, поисковому запросу, коду КПВЭД и другим параметрам.
// @Tags normalization
// @Produce application/octet-stream
// @Param format query string false "Формат экспорта (csv, json). По умолчанию csv"
// @Param category query string false "Фильтр по категории"
// @Param search query string false "Поисковый запрос"
// @Param kpved_code query string false "Фильтр по коду КПВЭД"
// @Param limit query int false "Лимит записей (по умолчанию 10000)"
// @Param database query string false "Путь к базе данных"
// @Success 200 {file} file "Экспортированные данные"
// @Failure 400 {object} ErrorResponse "Некорректные параметры запроса"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/export [get]
func (h *NormalizationHandler) HandleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	format := query.Get("format")
	category := query.Get("category")
	search := query.Get("search")
	kpvedCode := query.Get("kpved_code")
	limitStr := query.Get("limit")
	limit := 10000 // По умолчанию максимум 10000 записей
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	// По умолчанию CSV формат
	if format == "" {
		format = "csv"
	}

	// Получаем параметр database из query
	databasePath := query.Get("database")
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Формируем SQL запрос с фильтрами
	sqlQuery := `
		SELECT id, source_reference, source_name, code,
		       normalized_name, normalized_reference, category,
		       created_at, ai_confidence, ai_reasoning, processing_level,
		       kpved_code, kpved_name, kpved_confidence
		FROM normalized_data
		WHERE 1=1
	`
	var args []interface{}

	if category != "" {
		sqlQuery += " AND category = ?"
		args = append(args, category)
	}

	if search != "" {
		sqlQuery += " AND (normalized_name LIKE ? OR source_name LIKE ?)"
		searchParam := "%" + search + "%"
		args = append(args, searchParam, searchParam)
	}

	if kpvedCode != "" {
		sqlQuery += " AND kpved_code = ?"
		args = append(args, kpvedCode)
	}

	sqlQuery += " ORDER BY normalized_name, source_name LIMIT ?"
	args = append(args, limit)

	rows, err := db.Query(sqlQuery, args...)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to fetch export data: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ExportItem struct {
		ID                  int       `json:"id"`
		Code                string    `json:"code"`
		SourceName          string    `json:"source_name"`
		SourceReference     string    `json:"source_reference"`
		NormalizedName      string    `json:"normalized_name"`
		NormalizedReference string    `json:"normalized_reference"`
		Category            string    `json:"category"`
		CreatedAt           time.Time `json:"created_at"`
		AIConfidence        *float64  `json:"ai_confidence,omitempty"`
		AIReasoning         *string   `json:"ai_reasoning,omitempty"`
		ProcessingLevel     *string   `json:"processing_level,omitempty"`
		KpvedCode           *string   `json:"kpved_code,omitempty"`
		KpvedName           *string   `json:"kpved_name,omitempty"`
		KpvedConfidence     *float64  `json:"kpved_confidence,omitempty"`
	}

	items := []ExportItem{}
	for rows.Next() {
		var item ExportItem
		if err := rows.Scan(
			&item.ID,
			&item.SourceReference,
			&item.SourceName,
			&item.Code,
			&item.NormalizedName,
			&item.NormalizedReference,
			&item.Category,
			&item.CreatedAt,
			&item.AIConfidence,
			&item.AIReasoning,
			&item.ProcessingLevel,
			&item.KpvedCode,
			&item.KpvedName,
			&item.KpvedConfidence,
		); err != nil {
			slog.Warn("Error scanning export item", "error", err)
			continue
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Error iterating records: %v", err), http.StatusInternalServerError)
		return
	}

	// Формируем имя файла
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("normalized_export_%s.%s", timestamp, format)

	if format == "csv" {
		// Экспорт в CSV
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		// UTF-8 BOM для корректного отображения в Excel
		w.Write([]byte{0xEF, 0xBB, 0xBF})

		writer := csv.NewWriter(w)
		defer writer.Flush()

		// Заголовки
		headers := []string{
			"ID", "Код", "Исходное название", "Исходный reference",
			"Нормализованное название", "Нормализованный reference",
			"Категория", "AI Confidence", "Processing Level",
			"КПВЭД Код", "КПВЭД Название", "КПВЭД Confidence",
			"Дата создания",
		}
		writer.Write(headers)

		// Данные
		for _, item := range items {
			confidence := ""
			if item.AIConfidence != nil {
				confidence = fmt.Sprintf("%.2f", *item.AIConfidence)
			}

			processingLevel := ""
			if item.ProcessingLevel != nil {
				processingLevel = *item.ProcessingLevel
			}

			kpvedCode := ""
			if item.KpvedCode != nil {
				kpvedCode = *item.KpvedCode
			}

			kpvedName := ""
			if item.KpvedName != nil {
				kpvedName = *item.KpvedName
			}

			kpvedConfidence := ""
			if item.KpvedConfidence != nil {
				kpvedConfidence = fmt.Sprintf("%.2f", *item.KpvedConfidence)
			}

			record := []string{
				fmt.Sprintf("%d", item.ID),
				item.Code,
				item.SourceName,
				item.SourceReference,
				item.NormalizedName,
				item.NormalizedReference,
				item.Category,
				confidence,
				processingLevel,
				kpvedCode,
				kpvedName,
				kpvedConfidence,
				item.CreatedAt.Format("2006-01-02 15:04:05"),
			}
			writer.Write(record)
		}
	} else if format == "json" {
		// Экспорт в JSON
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))

		exportData := map[string]interface{}{
			"export_date": time.Now().Format(time.RFC3339),
			"item_count":  len(items),
			"filters": map[string]interface{}{
				"category":   category,
				"search":     search,
				"kpved_code": kpvedCode,
				"limit":      limit,
			},
			"items": items,
		}

		json.NewEncoder(w).Encode(exportData)
	} else {
		h.baseHandler.WriteJSONError(w, r, "Invalid format. Supported formats: csv, json", http.StatusBadRequest)
	}
}

// HandleNormalizationConfig обрабатывает запросы к /api/normalization/config
// Реализует получение и обновление конфигурации нормализации
// HandleNormalizationConfig обрабатывает запросы к /api/normalization/config
// @Summary Получить/Обновить конфигурацию нормализации
// @Description Возвращает или обновляет текущую конфигурацию нормализации. Поддерживает GET (получение), PUT/POST (обновление).
// @Tags normalization
// @Accept json
// @Produce json
// @Param config body object false "Конфигурация нормализации (для PUT/POST)"
// @Success 200 {object} map[string]interface{} "Конфигурация или сообщение об успехе"
// @Failure 400 {object} ErrorResponse "Некорректные данные запроса"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 503 {object} ErrorResponse "Сервис недоступен"
// @Router /api/normalization/config [get]
// @Router /api/normalization/config [put]
// @Router /api/normalization/config [post]
func (h *NormalizationHandler) HandleNormalizationConfig(w http.ResponseWriter, r *http.Request) {
	if h.normalizationService == nil {
		h.baseHandler.WriteJSONError(w, r, "Normalization service is not available", http.StatusServiceUnavailable)
		return
	}

	if r.Method == http.MethodGet {
		// Получение конфигурации
		config, err := h.normalizationService.GetNormalizationConfig()
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get normalization config: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"id":               config.ID,
			"database_path":    config.DatabasePath,
			"source_table":     config.SourceTable,
			"reference_column": config.ReferenceColumn,
			"code_column":      config.CodeColumn,
			"name_column":      config.NameColumn,
			"created_at":       config.CreatedAt.Format(time.RFC3339),
			"updated_at":       config.UpdatedAt.Format(time.RFC3339),
		}
		h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
	} else if r.Method == http.MethodPut || r.Method == http.MethodPost {
		// Обновление конфигурации
		var config struct {
			DatabasePath    string `json:"database_path"`
			SourceTable     string `json:"source_table"`
			ReferenceColumn string `json:"reference_column"`
			CodeColumn      string `json:"code_column"`
			NameColumn      string `json:"name_column"`
		}

		if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
			h.baseHandler.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация обязательных полей
		if config.SourceTable == "" {
			h.baseHandler.WriteJSONError(w, r, "source_table is required", http.StatusBadRequest)
			return
		}
		if config.ReferenceColumn == "" {
			h.baseHandler.WriteJSONError(w, r, "reference_column is required", http.StatusBadRequest)
			return
		}
		if config.CodeColumn == "" {
			h.baseHandler.WriteJSONError(w, r, "code_column is required", http.StatusBadRequest)
			return
		}
		if config.NameColumn == "" {
			h.baseHandler.WriteJSONError(w, r, "name_column is required", http.StatusBadRequest)
			return
		}

		// Обновляем конфигурацию
		err := h.normalizationService.UpdateNormalizationConfig(
			config.DatabasePath,
			config.SourceTable,
			config.ReferenceColumn,
			config.CodeColumn,
			config.NameColumn,
		)
		if err != nil {
			h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to update normalization config: %v", err), http.StatusInternalServerError)
			return
		}

		// Получаем обновленную конфигурацию для ответа
		updatedConfig, err := h.normalizationService.GetNormalizationConfig()
		if err != nil {
			// Если не удалось получить обновленную конфигурацию, возвращаем успех с полученными данными
			response := map[string]interface{}{
				"message": "Configuration updated successfully",
				"config":  config,
			}
			h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
			return
		}

		response := map[string]interface{}{
			"message": "Configuration updated successfully",
			"config": map[string]interface{}{
				"id":               updatedConfig.ID,
				"database_path":    updatedConfig.DatabasePath,
				"source_table":     updatedConfig.SourceTable,
				"reference_column": updatedConfig.ReferenceColumn,
				"code_column":      updatedConfig.CodeColumn,
				"name_column":      updatedConfig.NameColumn,
				"created_at":       updatedConfig.CreatedAt.Format(time.RFC3339),
				"updated_at":       updatedConfig.UpdatedAt.Format(time.RFC3339),
			},
		}
		h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
	} else {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodPost)
	}
}

// HandleNormalizationDatabases обрабатывает запросы к /api/normalization/databases
// @Summary Получить список баз данных
// @Description Возвращает список доступных баз данных для нормализации с информацией о размере и пути.
// @Tags normalization
// @Produce json
// @Success 200 {array} map[string]interface{} "Список баз данных"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Router /api/normalization/databases [get]
func (h *NormalizationHandler) HandleNormalizationDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем список баз данных из текущей БД
	// Используем информацию о доступных БД из serviceDB через normalizationService
	// Для простоты возвращаем информацию о текущей БД
	databases := []map[string]interface{}{}

	if h.db != nil && h.currentDBPath != "" {
		// Получаем информацию о текущей БД
		var fileSize int64
		if stat, err := os.Stat(h.currentDBPath); err == nil {
			fileSize = stat.Size()
		}

		databases = append(databases, map[string]interface{}{
			"name": filepath.Base(h.currentDBPath),
			"path": h.currentDBPath,
			"size": fileSize,
		})
	}

	h.baseHandler.WriteJSONResponse(w, r, databases, http.StatusOK)
}

// HandleNormalizationTables обрабатывает запросы к /api/normalization/tables
// @Summary Получить список таблиц базы данных
// @Description Возвращает список всех таблиц в указанной базе данных, исключая системные таблицы SQLite.
// @Tags normalization
// @Produce json
// @Param database query string false "Путь к базе данных"
// @Success 200 {array} string "Список имен таблиц"
// @Failure 400 {object} ErrorResponse "Некорректные параметры запроса"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 500 {object} ErrorResponse "Ошибка при запросе к базе данных"
// @Router /api/normalization/tables [get]
func (h *NormalizationHandler) HandleNormalizationTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем путь к БД из query параметра
	databasePath := r.URL.Query().Get("database")
	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем список таблиц
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to query tables: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tables := []map[string]interface{}{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Валидация имени таблицы
		if !isValidTableName(tableName) {
			slog.Warn("Invalid table name from database", "table", tableName)
			continue
		}

		// Получаем количество записей
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := db.QueryRow(query).Scan(&count); err != nil {
			count = 0
		}

		tables = append(tables, map[string]interface{}{
			"name":  tableName,
			"count": count,
		})
	}

	if err = rows.Err(); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Error iterating tables: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, tables, http.StatusOK)
}

// HandleNormalizationColumns обрабатывает запросы к /api/normalization/columns
// @Summary Получить список колонок таблицы
// @Description Возвращает список всех колонок указанной таблицы с информацией о типе данных и других атрибутах.
// @Tags normalization
// @Produce json
// @Param database query string false "Путь к базе данных"
// @Param table query string true "Имя таблицы"
// @Success 200 {array} map[string]interface{} "Список колонок с их атрибутами"
// @Failure 400 {object} ErrorResponse "Отсутствует имя таблицы или некорректные параметры"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 500 {object} ErrorResponse "Ошибка при запросе к базе данных"
// @Router /api/normalization/columns [get]
func (h *NormalizationHandler) HandleNormalizationColumns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметры из query
	databasePath := r.URL.Query().Get("database")
	tableName := r.URL.Query().Get("table")

	if tableName == "" {
		h.baseHandler.WriteJSONError(w, r, "Table name is required", http.StatusBadRequest)
		return
	}

	// Валидация имени таблицы
	if !isValidTableName(tableName) {
		h.baseHandler.WriteJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}

	db, err := h.getDB(databasePath)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			db.Close()
		}
	}()

	// Получаем информацию о колонках
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to query columns: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns := []map[string]interface{}{}
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int

		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			continue
		}

		col := map[string]interface{}{
			"name":     name,
			"type":     colType,
			"nullable": notNull == 0,
			"primary":  pk == 1,
		}

		if defaultValue.Valid {
			col["default"] = defaultValue.String
		} else {
			col["default"] = nil
		}

		columns = append(columns, col)
	}

	if err = rows.Err(); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Error iterating columns: %v", err), http.StatusInternalServerError)
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, columns, http.StatusOK)
}

// isValidTableName проверяет, что имя таблицы безопасно для использования в SQL запросах
func isValidTableName(name string) bool {
	if name == "" {
		return false
	}
	// Проверяем, что имя содержит только буквы, цифры и подчеркивания
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	// Проверяем, что имя не начинается с цифры
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		return false
	}
	// Проверяем, что имя не является зарезервированным словом SQLite
	reservedWords := []string{"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "INDEX", "TABLE", "VIEW"}
	nameUpper := strings.ToUpper(name)
	for _, word := range reservedWords {
		if nameUpper == word {
			return false
		}
	}
	return true
}

// HandleGetClientProjectNormalizationPreviewStats возвращает предварительную статистику перед запуском нормализации
// @Summary Получить превью статистику нормализации проекта
// @Description Возвращает агрегированные показатели по выбранным базам перед запуском нормализации.
// @Tags normalization
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Success 200 {object} map[string]interface{} "Предварительная статистика"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Клиент или проект не найдены"
// @Router /api/clients/{clientId}/projects/{projectId}/normalization/preview-stats [get]
func (h *NormalizationHandler) HandleGetClientProjectNormalizationPreviewStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Извлекаем clientId и projectId из контекста (установлены в Gin wrapper)
	// PathValue не работает с Gin router, поэтому используем только контекст
	var clientIDStr, projectIDStr string
	if ctxClientID := r.Context().Value("clientId"); ctxClientID != nil {
		clientIDStr = fmt.Sprintf("%v", ctxClientID)
	}
	if ctxProjectID := r.Context().Value("projectId"); ctxProjectID != nil {
		projectIDStr = fmt.Sprintf("%v", ctxProjectID)
	}

	if clientIDStr == "" || projectIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "clientId and projectId are required in URL path", http.StatusBadRequest)
		return
	}

	var clientID, projectID int
	var err error
	if clientID, err = strconv.Atoi(clientIDStr); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid clientId: %s", clientIDStr), http.StatusBadRequest)
		return
	}
	if projectID, err = strconv.Atoi(projectIDStr); err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Invalid projectId: %s", projectIDStr), http.StatusBadRequest)
		return
	}

	// Валидируем существование клиента и проекта
	if h.clientService == nil {
		h.baseHandler.WriteJSONError(w, r, "Client service not available", http.StatusInternalServerError)
		return
	}

	ctx := r.Context()
	client, err := h.clientService.GetClient(ctx, clientID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Client not found: %v", err), http.StatusNotFound)
		return
	}
	if client == nil {
		h.baseHandler.WriteJSONError(w, r, "Client not found", http.StatusNotFound)
		return
	}

	project, err := h.clientService.GetClientProject(ctx, clientID, projectID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Project not found: %v", err), http.StatusNotFound)
		return
	}
	if project == nil {
		h.baseHandler.WriteJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	// Получаем активные базы данных проекта
	databases, err := h.clientService.GetProjectDatabases(ctx, clientID, projectID)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to get project databases: %v", err), http.StatusInternalServerError)
		return
	}

	// Фильтруем только активные базы данных
	activeDatabases := make([]*database.ProjectDatabase, 0)
	for _, db := range databases {
		if db.IsActive {
			activeDatabases = append(activeDatabases, db)
		}
	}

	// Собираем статистику по каждой базе данных
	stats := make([]DatabasePreviewStats, 0, len(activeDatabases))
	var totalNomenclature, totalCounterparties, totalRecords int64

	// Используем контекст с таймаутом для каждой БД (2 секунды на БД)
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second*time.Duration(len(activeDatabases)))
	defer cancel()

	var accessibleCount, validCount int

	for _, db := range activeDatabases {
		dbStats := DatabasePreviewStats{
			DatabaseID:   db.ID,
			DatabaseName: db.Name,
			FilePath:     db.FilePath,
			IsAccessible: false,
			IsValid:      false,
		}

		// Проверяем доступность файла
		if _, err := os.Stat(db.FilePath); err != nil {
			dbStats.Error = fmt.Sprintf("Файл недоступен: %v", err)
			stats = append(stats, dbStats)
			continue
		}
		dbStats.IsAccessible = true
		accessibleCount++

		// Подсчитываем записи в базе данных
		nomenclatureCount, counterpartyCount, dbSize, err := h.countDatabaseRecords(ctxWithTimeout, db.FilePath)
		if err != nil {
			dbStats.Error = err.Error()
			stats = append(stats, dbStats)
			continue
		}

		dbStats.IsValid = true
		validCount++

		dbStats.NomenclatureCount = nomenclatureCount
		dbStats.CounterpartyCount = counterpartyCount
		dbStats.TotalRecords = nomenclatureCount + counterpartyCount
		dbStats.DatabaseSize = dbSize

		// Рассчитываем метрики заполненности
		if dbStats.IsValid && dbStats.TotalRecords > 0 {
			completenessMetrics, err := h.calculateCompletenessMetrics(ctxWithTimeout, db.FilePath, nomenclatureCount, counterpartyCount)
			if err != nil {
				// Логируем ошибку, но продолжаем обработку других БД
				slog.Warn("Не удалось рассчитать метрики заполненности для БД",
					"database_id", db.ID,
					"database_name", db.Name,
					"file_path", db.FilePath,
					"error", err)
				// Не добавляем ошибку в dbStats.Error, чтобы не помечать БД как невалидную
				// Метрики просто не будут включены в общий расчет
			} else {
				dbStats.Completeness = completenessMetrics
			}
		}

		totalNomenclature += nomenclatureCount
		totalCounterparties += counterpartyCount
		totalRecords += dbStats.TotalRecords

		stats = append(stats, dbStats)
	}

	// Подсчитываем потенциальные дубликаты (быстрый подсчет по именам)
	estimatedDuplicates := int64(0)
	duplicateGroups := int64(0)

	// Быстрый подсчет дублей по именам для каждой БД (только если не слишком много записей)
	if totalRecords > 0 && totalRecords < 100000 { // Ограничиваем для производительности
		for _, db := range activeDatabases {
			if db.FilePath == "" {
				continue
			}

			// Быстрый подсчет дублей по именам в БД
			dbDuplicates, dbGroups := h.countQuickDuplicates(ctxWithTimeout, db.FilePath)
			estimatedDuplicates += dbDuplicates
			duplicateGroups += dbGroups
		}
	} else if totalRecords > 0 {
		// Для больших БД используем оценку
		estimatedDuplicates = totalRecords / 20   // ~5%
		duplicateGroups = estimatedDuplicates / 3 // Примерно 3 записи на группу
	}

	// Рассчитываем общие метрики заполненности по всем БД
	var overallCompleteness CompletenessMetrics
	if totalNomenclature > 0 || totalCounterparties > 0 {
		overallCompleteness = h.calculateOverallCompleteness(stats)
	}

	response := map[string]interface{}{
		"total_databases":      len(activeDatabases),
		"accessible_databases": accessibleCount,
		"valid_databases":      validCount,
		"total_nomenclature":   totalNomenclature,
		"total_counterparties": totalCounterparties,
		"total_records":        totalRecords,
		"estimated_duplicates": estimatedDuplicates,
		"duplicate_groups":     duplicateGroups,
		"completeness_metrics": overallCompleteness,
		"databases":            stats,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// countDatabaseRecords подсчитывает количество записей в таблицах nomenclature_items и counterparties
// Возвращает также размер файла БД в байтах
func (h *NormalizationHandler) countDatabaseRecords(ctx context.Context, dbFilePath string) (nomenclatureCount, counterpartyCount int64, dbSize int64, err error) {
	// Проверяем существование файла и получаем размер
	fileInfo, err := os.Stat(dbFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, 0, 0, fmt.Errorf("файл БД не найден: %s: %w", dbFilePath, err)
		}
		return 0, 0, 0, fmt.Errorf("не удалось получить информацию о файле БД: %w", err)
	}
	dbSize = fileInfo.Size()

	// Открываем базу данных
	conn, err := sql.Open("sqlite3", dbFilePath+"?_timeout=2000")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("не удалось открыть БД: %w", err)
	}
	defer conn.Close()

	// Настраиваем таймаут для запросов
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(5 * time.Second)

	// Проверяем подключение
	if err := conn.PingContext(ctx); err != nil {
		return 0, 0, 0, fmt.Errorf("не удалось подключиться к БД: %w", err)
	}

	// Подсчитываем номенклатуру
	var tableExists bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='nomenclature_items'
		)
	`).Scan(&tableExists)
	if err == nil && tableExists {
		err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM nomenclature_items").Scan(&nomenclatureCount)
		if err != nil {
			// Игнорируем ошибку, продолжаем
		}
	}

	// Подсчитываем контрагентов
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='counterparties'
		)
	`).Scan(&tableExists)
	if err == nil && tableExists {
		err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM counterparties").Scan(&counterpartyCount)
		if err != nil {
			// Игнорируем ошибку, продолжаем
		}
	} else {
		// Если таблицы counterparties нет, проверяем catalog_items
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='catalog_items'
			)
		`).Scan(&tableExists)
		if err == nil && tableExists {
			// Определяем тип данных по каталогу или имени файла
			dataType := ""
			
			// Сначала пробуем определить по таблице catalogs
			var catalogsTableExists bool
			err = conn.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM sqlite_master 
					WHERE type='table' AND name='catalogs'
				)
			`).Scan(&catalogsTableExists)
			
			if err == nil && catalogsTableExists {
				// Проверяем, есть ли каталог "Номенклатура"
				var nomenclatureCatalogExists bool
				err = conn.QueryRowContext(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM catalogs 
						WHERE name = 'Номенклатура' OR name LIKE '%оменклатур%'
					)
				`).Scan(&nomenclatureCatalogExists)
				if err == nil && nomenclatureCatalogExists {
					dataType = "nomenclature"
				} else {
					// Проверяем, есть ли каталог "Контрагенты"
					var counterpartyCatalogExists bool
					err = conn.QueryRowContext(ctx, `
						SELECT EXISTS (
							SELECT 1 FROM catalogs 
							WHERE name = 'Контрагенты' OR name LIKE '%онтрагент%'
						)
					`).Scan(&counterpartyCatalogExists)
					if err == nil && counterpartyCatalogExists {
						dataType = "counterparties"
					}
				}
			}
			
			// Если не удалось определить по каталогу, используем имя файла
			if dataType == "" {
				fileName := filepath.Base(dbFilePath)
				fileInfo := database.ParseDatabaseFileInfo(fileName)
				dataType = fileInfo.DataType
			}
			
			// Подсчитываем записи в зависимости от типа данных
			var catalogItemsCount int64
			if dataType == "nomenclature" {
				// Считаем номенклатуру из catalog_items
				err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_items").Scan(&catalogItemsCount)
				if err == nil {
					nomenclatureCount = catalogItemsCount
				}
			} else if dataType == "counterparties" {
				// Считаем контрагентов из catalog_items
				err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_items").Scan(&catalogItemsCount)
				if err == nil {
					counterpartyCount = catalogItemsCount
				}
			} else {
				// Если тип не определен, считаем как контрагентов (по умолчанию)
				err = conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM catalog_items").Scan(&catalogItemsCount)
				if err == nil {
					counterpartyCount = catalogItemsCount
				}
			}
		}
	}

	return nomenclatureCount, counterpartyCount, dbSize, nil
}

// calculateCompletenessMetrics рассчитывает метрики заполненности для базы данных
func (h *NormalizationHandler) calculateCompletenessMetrics(ctx context.Context, dbFilePath string, nomenclatureCount, counterpartyCount int64) (*CompletenessMetrics, error) {
	// Валидация входных параметров
	if dbFilePath == "" {
		return nil, fmt.Errorf("путь к базе данных не может быть пустым")
	}

	if nomenclatureCount < 0 || counterpartyCount < 0 {
		return nil, fmt.Errorf("количество записей не может быть отрицательным: nomenclature=%d, counterparties=%d", nomenclatureCount, counterpartyCount)
	}

	metrics := &CompletenessMetrics{}

	// Проверяем существование файла перед открытием
	if _, err := os.Stat(dbFilePath); err != nil {
		return nil, fmt.Errorf("файл базы данных недоступен: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbFilePath+"?_timeout=2000&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть БД %s: %w", dbFilePath, err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			slog.Warn("Ошибка при закрытии соединения с БД", "error", closeErr, "db_path", dbFilePath)
		}
	}()

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(5 * time.Second)

	// Проверяем подключение с контекстом таймаута
	if err := conn.PingContext(ctx); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("таймаут подключения к БД %s", dbFilePath)
		}
		if ctx.Err() == context.Canceled {
			return nil, fmt.Errorf("подключение к БД %s было отменено", dbFilePath)
		}
		return nil, fmt.Errorf("не удалось подключиться к БД %s: %w", dbFilePath, err)
	}

	// Рассчитываем метрики для номенклатуры
	if nomenclatureCount > 0 {
		var tableExists bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='nomenclature_items'
			)
		`).Scan(&tableExists)

		if err != nil {
			slog.Warn("Ошибка при проверке существования таблицы nomenclature_items",
				"error", err, "db_path", dbFilePath)
		}

		if err == nil && tableExists {
			var total, withCode, withName, withChar, withUnit int64
			err = conn.QueryRowContext(ctx, `
				SELECT 
					COUNT(*) as total,
					COUNT(CASE WHEN nomenclature_code IS NOT NULL AND TRIM(nomenclature_code) != '' THEN 1 END) as with_code,
					COUNT(CASE WHEN nomenclature_name IS NOT NULL AND TRIM(nomenclature_name) != '' THEN 1 END) as with_name,
					COUNT(CASE WHEN characteristic_name IS NOT NULL AND TRIM(characteristic_name) != '' THEN 1 END) as with_char,
					COUNT(CASE WHEN attributes_xml IS NOT NULL AND TRIM(attributes_xml) != '' 
						AND (attributes_xml LIKE '%ЕдиницаИзмерения%' OR attributes_xml LIKE '%Единица%' 
						OR attributes_xml LIKE '%Unit%' OR attributes_xml LIKE '%единица%') THEN 1 END) as with_unit
				FROM nomenclature_items
			`).Scan(&total, &withCode, &withName, &withChar, &withUnit)

			if err != nil {
				slog.Warn("Ошибка при расчете метрик номенклатуры",
					"error", err, "db_path", dbFilePath)
			} else if total > 0 {
				metrics.NomenclatureCompleteness.ArticlesPercent = float64(withCode) / float64(total) * 100
				metrics.NomenclatureCompleteness.UnitsPercent = float64(withUnit) / float64(total) * 100
				metrics.NomenclatureCompleteness.DescriptionsPercent = float64(withChar) / float64(total) * 100
				// Общая заполненность = среднее от артикулов, единиц измерения и описаний
				metrics.NomenclatureCompleteness.OverallCompleteness = (metrics.NomenclatureCompleteness.ArticlesPercent +
					metrics.NomenclatureCompleteness.UnitsPercent +
					metrics.NomenclatureCompleteness.DescriptionsPercent) / 3
			} else if total == 0 && nomenclatureCount > 0 {
				slog.Warn("Несоответствие: nomenclatureCount > 0, но total = 0",
					"nomenclature_count", nomenclatureCount, "db_path", dbFilePath)
			}
		}
	}

	// Рассчитываем метрики для контрагентов
	if counterpartyCount > 0 {
		var tableExists bool
		var withINN, withAddress, withContacts int64
		var total int64

		// Проверяем таблицу counterparties
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='counterparties'
			)
		`).Scan(&tableExists)

		if err != nil {
			slog.Warn("Ошибка при проверке существования таблицы counterparties",
				"error", err, "db_path", dbFilePath)
		}

		if err == nil && tableExists {
			err = conn.QueryRowContext(ctx, `
				SELECT 
					COUNT(*) as total,
					COUNT(CASE WHEN (inn IS NOT NULL AND TRIM(inn) != '') OR (bin IS NOT NULL AND TRIM(bin) != '') THEN 1 END) as with_inn,
					COUNT(CASE WHEN (legal_address IS NOT NULL AND TRIM(legal_address) != '') 
						OR (postal_address IS NOT NULL AND TRIM(postal_address) != '') THEN 1 END) as with_address,
					COUNT(CASE WHEN (contact_phone IS NOT NULL AND TRIM(contact_phone) != '') 
						OR (contact_email IS NOT NULL AND TRIM(contact_email) != '') THEN 1 END) as with_contacts
				FROM counterparties
			`).Scan(&total, &withINN, &withAddress, &withContacts)
		} else if err == nil {
			// Проверяем catalog_items для контрагентов
			var catalogItemsErr error
			catalogItemsErr = conn.QueryRowContext(ctx, `
				SELECT EXISTS (
					SELECT 1 FROM sqlite_master 
					WHERE type='table' AND name='catalog_items'
				)
			`).Scan(&tableExists)

			if catalogItemsErr != nil {
				slog.Warn("Ошибка при проверке существования таблицы catalog_items",
					"error", catalogItemsErr, "db_path", dbFilePath)
			}

			if catalogItemsErr == nil && tableExists {
				// Проверяем наличие каталога "Контрагенты"
				var catalogExists bool
				err = conn.QueryRowContext(ctx, `
					SELECT EXISTS (
						SELECT 1 FROM catalogs 
						WHERE name = 'Контрагенты' OR name LIKE '%онтрагент%'
					)
				`).Scan(&catalogExists)

				var catalogCheckErr error
				if catalogItemsErr == nil {
					catalogCheckErr = conn.QueryRowContext(ctx, `
						SELECT EXISTS (
							SELECT 1 FROM catalogs 
							WHERE name = 'Контрагенты' OR name LIKE '%онтрагент%'
						)
					`).Scan(&catalogExists)
				}

				if catalogCheckErr != nil {
					slog.Warn("Ошибка при проверке наличия каталога контрагентов",
						"error", catalogCheckErr, "db_path", dbFilePath)
				}

				if catalogCheckErr == nil && catalogExists {
					catalogQueryErr := conn.QueryRowContext(ctx, `
						SELECT 
							COUNT(*) as total,
							COUNT(CASE WHEN ci.attributes_xml IS NOT NULL AND TRIM(ci.attributes_xml) != ''
								AND (ci.attributes_xml LIKE '%ИНН%' OR ci.attributes_xml LIKE '%БИН%' 
								OR ci.attributes_xml LIKE '%INN%' OR ci.attributes_xml LIKE '%BIN%') THEN 1 END) as with_inn,
							COUNT(CASE WHEN ci.attributes_xml IS NOT NULL AND TRIM(ci.attributes_xml) != ''
								AND (ci.attributes_xml LIKE '%адрес%' OR ci.attributes_xml LIKE '%address%'
								OR ci.attributes_xml LIKE '%Адрес%') THEN 1 END) as with_address,
							COUNT(CASE WHEN ci.attributes_xml IS NOT NULL AND TRIM(ci.attributes_xml) != ''
								AND (ci.attributes_xml LIKE '%телефон%' OR ci.attributes_xml LIKE '%phone%'
								OR ci.attributes_xml LIKE '%email%' OR ci.attributes_xml LIKE '%почта%'
								OR ci.attributes_xml LIKE '%контакт%') THEN 1 END) as with_contacts
						FROM catalog_items ci
						INNER JOIN catalogs c ON ci.catalog_id = c.id
						WHERE c.name = 'Контрагенты' OR c.name LIKE '%онтрагент%'
					`).Scan(&total, &withINN, &withAddress, &withContacts)
					
					if catalogQueryErr != nil {
						slog.Warn("Ошибка при запросе метрик контрагентов из catalog_items",
							"error", catalogQueryErr, "db_path", dbFilePath)
						err = catalogQueryErr
					}
				}
			}
		}

		if err != nil {
			slog.Warn("Ошибка при расчете метрик контрагентов",
				"error", err, "db_path", dbFilePath)
		} else if total > 0 {
			metrics.CounterpartyCompleteness.INNPercent = float64(withINN) / float64(total) * 100
			metrics.CounterpartyCompleteness.AddressPercent = float64(withAddress) / float64(total) * 100
			metrics.CounterpartyCompleteness.ContactsPercent = float64(withContacts) / float64(total) * 100
			// Общая заполненность = среднее от ИНН, адресов и контактов
			metrics.CounterpartyCompleteness.OverallCompleteness = (metrics.CounterpartyCompleteness.INNPercent +
				metrics.CounterpartyCompleteness.AddressPercent +
				metrics.CounterpartyCompleteness.ContactsPercent) / 3
		} else if total == 0 && counterpartyCount > 0 {
			slog.Warn("Несоответствие: counterpartyCount > 0, но total = 0",
				"counterparty_count", counterpartyCount, "db_path", dbFilePath)
		}
	}

	// Возвращаем метрики даже если были ошибки - частичные данные лучше чем ничего
	return metrics, nil
}

// calculateOverallCompleteness рассчитывает общие метрики заполненности по всем БД
func (h *NormalizationHandler) calculateOverallCompleteness(stats []DatabasePreviewStats) CompletenessMetrics {
	var overall CompletenessMetrics
	
	if len(stats) == 0 {
		return overall
	}

	var nomTotal, nomWithArticles, nomWithUnits, nomWithDesc int64
	var cpTotal, cpWithINN, cpWithAddress, cpWithContacts int64

	validDatabases := 0
	for _, db := range stats {
		if db.Completeness == nil {
			continue
		}
		validDatabases++

		// Агрегируем метрики номенклатуры
		if db.NomenclatureCount > 0 {
			// Используем метрики даже если OverallCompleteness = 0 (могут быть частичные данные)
			nomTotal += db.NomenclatureCount
			// Защита от деления на ноль и некорректных процентов
			articlesPercent := db.Completeness.NomenclatureCompleteness.ArticlesPercent
			if articlesPercent < 0 {
				articlesPercent = 0
			}
			if articlesPercent > 100 {
				articlesPercent = 100
			}
			unitsPercent := db.Completeness.NomenclatureCompleteness.UnitsPercent
			if unitsPercent < 0 {
				unitsPercent = 0
			}
			if unitsPercent > 100 {
				unitsPercent = 100
			}
			descriptionsPercent := db.Completeness.NomenclatureCompleteness.DescriptionsPercent
			if descriptionsPercent < 0 {
				descriptionsPercent = 0
			}
			if descriptionsPercent > 100 {
				descriptionsPercent = 100
			}
			
			nomWithArticles += int64(float64(db.NomenclatureCount) * articlesPercent / 100)
			nomWithUnits += int64(float64(db.NomenclatureCount) * unitsPercent / 100)
			nomWithDesc += int64(float64(db.NomenclatureCount) * descriptionsPercent / 100)
		}

		// Агрегируем метрики контрагентов
		if db.CounterpartyCount > 0 {
			// Используем метрики даже если OverallCompleteness = 0 (могут быть частичные данные)
			cpTotal += db.CounterpartyCount
			// Защита от деления на ноль и некорректных процентов
			innPercent := db.Completeness.CounterpartyCompleteness.INNPercent
			if innPercent < 0 {
				innPercent = 0
			}
			if innPercent > 100 {
				innPercent = 100
			}
			addressPercent := db.Completeness.CounterpartyCompleteness.AddressPercent
			if addressPercent < 0 {
				addressPercent = 0
			}
			if addressPercent > 100 {
				addressPercent = 100
			}
			contactsPercent := db.Completeness.CounterpartyCompleteness.ContactsPercent
			if contactsPercent < 0 {
				contactsPercent = 0
			}
			if contactsPercent > 100 {
				contactsPercent = 100
			}
			
			cpWithINN += int64(float64(db.CounterpartyCount) * innPercent / 100)
			cpWithAddress += int64(float64(db.CounterpartyCount) * addressPercent / 100)
			cpWithContacts += int64(float64(db.CounterpartyCount) * contactsPercent / 100)
		}
	}

	if validDatabases == 0 {
		slog.Warn("calculateOverallCompleteness: нет БД с валидными метриками заполненности",
			"total_databases", len(stats))
		return overall
	}

	// Рассчитываем общие проценты
	if nomTotal > 0 {
		overall.NomenclatureCompleteness.ArticlesPercent = float64(nomWithArticles) / float64(nomTotal) * 100
		overall.NomenclatureCompleteness.UnitsPercent = float64(nomWithUnits) / float64(nomTotal) * 100
		overall.NomenclatureCompleteness.DescriptionsPercent = float64(nomWithDesc) / float64(nomTotal) * 100
		overall.NomenclatureCompleteness.OverallCompleteness = (overall.NomenclatureCompleteness.ArticlesPercent +
			overall.NomenclatureCompleteness.UnitsPercent +
			overall.NomenclatureCompleteness.DescriptionsPercent) / 3
	}

	if cpTotal > 0 {
		overall.CounterpartyCompleteness.INNPercent = float64(cpWithINN) / float64(cpTotal) * 100
		overall.CounterpartyCompleteness.AddressPercent = float64(cpWithAddress) / float64(cpTotal) * 100
		overall.CounterpartyCompleteness.ContactsPercent = float64(cpWithContacts) / float64(cpTotal) * 100
		overall.CounterpartyCompleteness.OverallCompleteness = (overall.CounterpartyCompleteness.INNPercent +
			overall.CounterpartyCompleteness.AddressPercent +
			overall.CounterpartyCompleteness.ContactsPercent) / 3
	}

	return overall
}

// countQuickDuplicates быстро подсчитывает дубликаты по именам в базе данных
// Возвращает количество записей-дубликатов и количество групп дубликатов
func (h *NormalizationHandler) countQuickDuplicates(ctx context.Context, dbFilePath string) (duplicateRecords, duplicateGroups int64) {
	conn, err := sql.Open("sqlite3", dbFilePath+"?_timeout=2000")
	if err != nil {
		return 0, 0
	}
	defer conn.Close()

	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(3 * time.Second)

	if err := conn.PingContext(ctx); err != nil {
		return 0, 0
	}

	// Подсчитываем дубликаты номенклатуры по именам
	var nomDuplicates, nomGroups int64
	var tableExists bool

	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='nomenclature_items'
		)
	`).Scan(&tableExists)

	if err == nil && tableExists {
		// Подсчитываем группы дубликатов по именам (игнорируя регистр и пробелы)
		err = conn.QueryRowContext(ctx, `
			SELECT 
				COUNT(*) - COUNT(DISTINCT LOWER(TRIM(name))) as duplicates,
				COUNT(DISTINCT LOWER(TRIM(name))) as unique_names
			FROM nomenclature_items
			WHERE name IS NOT NULL AND name != ''
		`).Scan(&nomDuplicates, &nomGroups)
		if err != nil {
			nomDuplicates = 0
			nomGroups = 0
		}
		// Вычисляем количество групп (группы = записи с одинаковыми именами)
		if nomDuplicates > 0 {
			nomGroups = nomDuplicates / 2 // Примерно 2 записи на группу
			if nomGroups < 1 {
				nomGroups = 1
			}
		}
	}

	// Подсчитываем дубликаты контрагентов
	var cpDuplicates, cpGroups int64

	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='counterparties'
		)
	`).Scan(&tableExists)

	if err == nil && tableExists {
		err = conn.QueryRowContext(ctx, `
			SELECT 
				COUNT(*) - COUNT(DISTINCT LOWER(TRIM(name))) as duplicates,
				COUNT(DISTINCT LOWER(TRIM(name))) as unique_names
			FROM counterparties
			WHERE name IS NOT NULL AND name != ''
		`).Scan(&cpDuplicates, &cpGroups)
		if err != nil {
			cpDuplicates = 0
			cpGroups = 0
		}
		if cpDuplicates > 0 {
			cpGroups = cpDuplicates / 2
			if cpGroups < 1 {
				cpGroups = 1
			}
		}
	} else {
		// Проверяем catalog_items для контрагентов
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='catalog_items'
			)
		`).Scan(&tableExists)

		if err == nil && tableExists {
			// Подсчитываем дубликаты контрагентов из catalog_items
			err = conn.QueryRowContext(ctx, `
				SELECT 
					COUNT(*) - COUNT(DISTINCT LOWER(TRIM(ci.name))) as duplicates
				FROM catalog_items ci
				INNER JOIN catalogs c ON ci.catalog_id = c.id
				WHERE c.name = 'Контрагенты' AND ci.name IS NOT NULL AND ci.name != ''
			`).Scan(&cpDuplicates)
			if err != nil {
				// Если не получилось с фильтром, пробуем без него
				conn.QueryRowContext(ctx, `
					SELECT 
						COUNT(*) - COUNT(DISTINCT LOWER(TRIM(name))) as duplicates
					FROM catalog_items
					WHERE name IS NOT NULL AND name != ''
				`).Scan(&cpDuplicates)
			}
			if cpDuplicates > 0 {
				cpGroups = cpDuplicates / 2
				if cpGroups < 1 {
					cpGroups = 1
				}
			}
		}
	}

	return nomDuplicates + cpDuplicates, nomGroups + cpGroups
}

// HandleBenchmarkDataset возвращает выборку данных из нормализации для бэнчмарка
// @Summary Получить датасет для бэнчмарка
// @Description Возвращает выборку названий товаров из catalog_items или nomenclature_items для использования в бэнчмарке моделей
// @Tags normalization
// @Produce json
// @Param limit query int false "Количество записей (по умолчанию 50)"
// @Param database query string false "Путь к базе данных (опционально)"
// @Success 200 {object} map[string]interface{} "Массив названий товаров"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка"
// @Router /api/normalization/benchmark-dataset [get]
func (h *NormalizationHandler) HandleBenchmarkDataset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	ctx := r.Context()

	// Получаем параметры запроса с валидацией
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // По умолчанию 50 записей
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			// Ограничиваем максимум 500 записей для безопасности
			if limit > 500 {
				limit = 500
				slog.Warn("[BenchmarkDataset] Limit exceeded maximum, capped at 500", "requested", parsedLimit)
			}
		} else {
			slog.Warn("[BenchmarkDataset] Invalid limit parameter", "limit", limitStr, "error", err)
		}
	}

	databasePath := r.URL.Query().Get("database")
	// Валидация databasePath для предотвращения path traversal
	if databasePath != "" {
		if strings.Contains(databasePath, "..") || strings.Contains(databasePath, "~") {
			slog.Error("[BenchmarkDataset] Invalid database path detected", "path", databasePath)
			h.baseHandler.WriteJSONError(w, r, "Invalid database path", http.StatusBadRequest)
			return
		}
	}

	// Получаем базу данных
	db, err := h.getDB(databasePath)
	if err != nil {
		slog.Error("[BenchmarkDataset] Failed to open database", "error", err, "database", databasePath)
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	if db == nil {
		slog.Error("[BenchmarkDataset] Database is nil")
		h.baseHandler.WriteJSONError(w, r, "Database is not available", http.StatusInternalServerError)
		return
	}
	defer func() {
		if databasePath != "" && databasePath != h.currentDBPath {
			if err := db.Close(); err != nil {
				slog.Warn("[BenchmarkDataset] Error closing database", "error", err)
			}
		}
	}()

	var names []string

	// Пробуем получить данные из catalog_items
	conn := db.GetConnection()
	if conn == nil {
		slog.Error("[BenchmarkDataset] Database connection is nil")
		h.baseHandler.WriteJSONError(w, r, "Database connection is nil", http.StatusInternalServerError)
		return
	}

	// Проверяем наличие таблицы catalog_items
	var hasCatalogItems bool
	err = conn.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM sqlite_master 
			WHERE type='table' AND name='catalog_items'
		)
	`).Scan(&hasCatalogItems)

	if err != nil {
		slog.Warn("[BenchmarkDataset] Error checking catalog_items table", "error", err)
	} else if hasCatalogItems {
		// Получаем выборку из catalog_items
		rows, err := conn.QueryContext(ctx, `
			SELECT DISTINCT name
			FROM catalog_items
			WHERE name IS NOT NULL AND name != '' AND TRIM(name) != ''
			ORDER BY RANDOM()
			LIMIT ?
		`, limit)
		if err != nil {
			slog.Warn("[BenchmarkDataset] Error querying catalog_items", "error", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var name string
				if err := rows.Scan(&name); err != nil {
					slog.Warn("[BenchmarkDataset] Error scanning catalog_items row", "error", err)
					continue
				}
				if name != "" {
					names = append(names, name)
				}
			}
			if err := rows.Err(); err != nil {
				slog.Warn("[BenchmarkDataset] Error iterating catalog_items rows", "error", err)
			}
		}
	}

	// Если не получили достаточно данных, пробуем nomenclature_items
	if len(names) < limit {
		var hasNomenclatureItems bool
		err = conn.QueryRowContext(ctx, `
			SELECT EXISTS(
				SELECT 1 FROM sqlite_master 
				WHERE type='table' AND name='nomenclature_items'
			)
		`).Scan(&hasNomenclatureItems)

		if err != nil {
			slog.Warn("[BenchmarkDataset] Error checking nomenclature_items table", "error", err)
		} else if hasNomenclatureItems {
			needed := limit - len(names)
			rows, err := conn.QueryContext(ctx, `
				SELECT DISTINCT nomenclature_name
				FROM nomenclature_items
				WHERE nomenclature_name IS NOT NULL AND nomenclature_name != '' AND TRIM(nomenclature_name) != ''
				ORDER BY RANDOM()
				LIMIT ?
			`, needed)
			if err != nil {
				slog.Warn("[BenchmarkDataset] Error querying nomenclature_items", "error", err)
			} else {
				defer rows.Close()
				for rows.Next() {
					var name string
					if err := rows.Scan(&name); err != nil {
						slog.Warn("[BenchmarkDataset] Error scanning nomenclature_items row", "error", err)
						continue
					}
					if name != "" {
						names = append(names, name)
					}
				}
				if err := rows.Err(); err != nil {
					slog.Warn("[BenchmarkDataset] Error iterating nomenclature_items rows", "error", err)
				}
			}
		}
	}

	slog.Info("[BenchmarkDataset] Dataset retrieved", "count", len(names), "limit", limit, "database", databasePath)

	// Если все еще нет данных, возвращаем пустой массив (frontend может использовать дефолтные данные)
	response := map[string]interface{}{
		"data":   names,
		"count":  len(names),
		"limit":  limit,
		"source": "normalization",
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleDeleteAllNormalizedData удаляет все результаты нормализации из базы данных.
// @Summary Удалить все результаты нормализации
// @Description Удаляет все записи из normalized_data и связанные атрибуты (через CASCADE)
// @Tags normalization
// @Accept json
// @Produce json
// @Param confirm query bool true "Подтверждение удаления (обязательно true)"
// @Success 200 {object} map[string]interface{} "Успешное удаление"
// @Failure 400 {object} ErrorResponse "Неверный запрос или отсутствует подтверждение"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/normalization/data/all [delete]
func (h *NormalizationHandler) HandleDeleteAllNormalizedData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	// Проверяем подтверждение
	confirmParam := r.URL.Query().Get("confirm")
	if confirmParam != "true" {
		h.baseHandler.WriteJSONError(w, r, "Подтверждение обязательно. Добавьте ?confirm=true к URL", http.StatusBadRequest)
		return
	}

	// Используем normalizedDB если доступен, иначе основную БД
	db := h.normalizedDB
	if db == nil {
		db = h.db
	}

	if db == nil {
		slog.Error("[DeleteAllNormalizedData] Database is not available")
		h.baseHandler.WriteJSONError(w, r, "База данных недоступна", http.StatusInternalServerError)
		return
	}

	// Получаем статистику до удаления для логирования
	var countBefore int64
	err := db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&countBefore)
	if err != nil {
		slog.Warn("[DeleteAllNormalizedData] Failed to count records before deletion", "error", err)
		countBefore = 0
	}

	// Выполняем удаление
	rowsAffected, err := db.DeleteAllNormalizedData()
	if err != nil {
		slog.Error("[DeleteAllNormalizedData] Failed to delete normalized data", "error", err)
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Ошибка удаления данных: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("[DeleteAllNormalizedData] Successfully deleted all normalized data", 
		"rows_affected", rowsAffected, 
		"count_before", countBefore)

	// Возвращаем результат
	response := map[string]interface{}{
		"success":       true,
		"message":       "Все результаты нормализации успешно удалены",
		"rows_affected": rowsAffected,
		"count_before":   countBefore,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleDeleteNormalizedDataByProject удаляет результаты нормализации для указанного проекта.
// @Summary Удалить результаты нормализации по проекту
// @Description Удаляет все записи из normalized_data для указанного проекта и связанные атрибуты (через CASCADE)
// @Tags normalization
// @Accept json
// @Produce json
// @Param project_id query int true "ID проекта"
// @Param confirm query bool true "Подтверждение удаления (обязательно true)"
// @Success 200 {object} map[string]interface{} "Успешное удаление"
// @Failure 400 {object} ErrorResponse "Неверный запрос или отсутствует подтверждение"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/normalization/data/project [delete]
func (h *NormalizationHandler) HandleDeleteNormalizedDataByProject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	// Проверяем подтверждение
	confirmParam := r.URL.Query().Get("confirm")
	if confirmParam != "true" {
		h.baseHandler.WriteJSONError(w, r, "Подтверждение обязательно. Добавьте ?confirm=true к URL", http.StatusBadRequest)
		return
	}

	// Получаем project_id из query параметров
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "Параметр project_id обязателен", http.StatusBadRequest)
		return
	}

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Неверный формат project_id: %s", projectIDStr), http.StatusBadRequest)
		return
	}

	if projectID <= 0 {
		h.baseHandler.WriteJSONError(w, r, "project_id должен быть положительным числом", http.StatusBadRequest)
		return
	}

	// Используем normalizedDB если доступен, иначе основную БД
	db := h.normalizedDB
	if db == nil {
		db = h.db
	}

	if db == nil {
		slog.Error("[DeleteNormalizedDataByProject] Database is not available")
		h.baseHandler.WriteJSONError(w, r, "База данных недоступна", http.StatusInternalServerError)
		return
	}

	// Получаем статистику до удаления для логирования
	var countBefore int64
	err = db.QueryRow("SELECT COUNT(*) FROM normalized_data WHERE project_id = ?", projectID).Scan(&countBefore)
	if err != nil {
		slog.Warn("[DeleteNormalizedDataByProject] Failed to count records before deletion", "error", err, "project_id", projectID)
		countBefore = 0
	}

	if countBefore == 0 {
		h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
			"success":       true,
			"message":       fmt.Sprintf("Нет данных для проекта %d", projectID),
			"rows_affected": 0,
			"count_before":  0,
			"project_id":    projectID,
		}, http.StatusOK)
		return
	}

	// Выполняем удаление
	rowsAffected, err := db.DeleteNormalizedDataByProjectID(projectID)
	if err != nil {
		slog.Error("[DeleteNormalizedDataByProject] Failed to delete normalized data", "error", err, "project_id", projectID)
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Ошибка удаления данных: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("[DeleteNormalizedDataByProject] Successfully deleted normalized data for project", 
		"rows_affected", rowsAffected, 
		"count_before", countBefore,
		"project_id", projectID)

	// Возвращаем результат
	response := map[string]interface{}{
		"success":       true,
		"message":       fmt.Sprintf("Результаты нормализации для проекта %d успешно удалены", projectID),
		"rows_affected": rowsAffected,
		"count_before":   countBefore,
		"project_id":    projectID,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleDeleteNormalizedDataBySession удаляет результаты нормализации для указанной сессии.
// @Summary Удалить результаты нормализации по сессии
// @Description Удаляет все записи из normalized_data для указанной сессии и связанные атрибуты (через CASCADE)
// @Tags normalization
// @Accept json
// @Produce json
// @Param session_id query int true "ID сессии нормализации"
// @Param confirm query bool true "Подтверждение удаления (обязательно true)"
// @Success 200 {object} map[string]interface{} "Успешное удаление"
// @Failure 400 {object} ErrorResponse "Неверный запрос или отсутствует подтверждение"
// @Failure 405 {object} ErrorResponse "Метод не поддерживается"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/normalization/data/session [delete]
func (h *NormalizationHandler) HandleDeleteNormalizedDataBySession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	// Проверяем подтверждение
	confirmParam := r.URL.Query().Get("confirm")
	if confirmParam != "true" {
		h.baseHandler.WriteJSONError(w, r, "Подтверждение обязательно. Добавьте ?confirm=true к URL", http.StatusBadRequest)
		return
	}

	// Получаем session_id из query параметров
	sessionIDStr := r.URL.Query().Get("session_id")
	if sessionIDStr == "" {
		h.baseHandler.WriteJSONError(w, r, "Параметр session_id обязателен", http.StatusBadRequest)
		return
	}

	sessionID, err := strconv.Atoi(sessionIDStr)
	if err != nil {
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Неверный формат session_id: %s", sessionIDStr), http.StatusBadRequest)
		return
	}

	if sessionID <= 0 {
		h.baseHandler.WriteJSONError(w, r, "session_id должен быть положительным числом", http.StatusBadRequest)
		return
	}

	// Используем normalizedDB если доступен, иначе основную БД
	db := h.normalizedDB
	if db == nil {
		db = h.db
	}

	if db == nil {
		slog.Error("[DeleteNormalizedDataBySession] Database is not available")
		h.baseHandler.WriteJSONError(w, r, "База данных недоступна", http.StatusInternalServerError)
		return
	}

	// Получаем статистику до удаления для логирования
	var countBefore int64
	err = db.QueryRow("SELECT COUNT(*) FROM normalized_data WHERE normalization_session_id = ?", sessionID).Scan(&countBefore)
	if err != nil {
		slog.Warn("[DeleteNormalizedDataBySession] Failed to count records before deletion", "error", err, "session_id", sessionID)
		countBefore = 0
	}

	if countBefore == 0 {
		h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
			"success":       true,
			"message":       fmt.Sprintf("Нет данных для сессии %d", sessionID),
			"rows_affected": 0,
			"count_before":  0,
			"session_id":    sessionID,
		}, http.StatusOK)
		return
	}

	// Выполняем удаление
	rowsAffected, err := db.DeleteNormalizedDataBySessionID(sessionID)
	if err != nil {
		slog.Error("[DeleteNormalizedDataBySession] Failed to delete normalized data", "error", err, "session_id", sessionID)
		h.baseHandler.WriteJSONError(w, r, fmt.Sprintf("Ошибка удаления данных: %v", err), http.StatusInternalServerError)
		return
	}

	slog.Info("[DeleteNormalizedDataBySession] Successfully deleted normalized data for session", 
		"rows_affected", rowsAffected, 
		"count_before", countBefore,
		"session_id", sessionID)

	// Возвращаем результат
	response := map[string]interface{}{
		"success":       true,
		"message":       fmt.Sprintf("Результаты нормализации для сессии %d успешно удалены", sessionID),
		"rows_affected": rowsAffected,
		"count_before":   countBefore,
		"session_id":    sessionID,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}
