package handlers

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/enrichment"
	"httpserver/normalization"
)

const (
	maxCounterpartiesLimit       = 100000
	heavyCounterpartiesThreshold = 5000
)

// CounterpartyService описывает методы сервиса контрагентов, используемые обработчиком.
type CounterpartyService interface {
	GetServiceDB() *database.ServiceDB
	GetNormalizedCounterpartyStats(projectID int) (map[string]interface{}, error)
	GetNormalizedCounterparty(id int) (*database.NormalizedCounterparty, error)
	UpdateNormalizedCounterparty(id int, normalizedName string, taxID, kpp, bin string,
		legalAddress, postalAddress string, contactPhone, contactEmail string,
		contactPerson, legalForm string, bankName, bankAccount string,
		correspondentAccount, bik string, qualityScore float64,
		sourceEnrichment, subcategory string) error
	GetCounterpartyDuplicates(projectID int) ([]map[string]interface{}, error)
	MergeCounterpartyDuplicates(masterID int, mergeIDs []int) (*database.NormalizedCounterparty, error)
	GetNormalizedCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error)
	GetNormalizedCounterparties(projectID int, limit, offset int, search, taxID, bin string) ([]*database.NormalizedCounterparty, int, error)
	GetClientProject(projectID int) (*database.ClientProject, error)
	GetAllCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, source, sortBy, order string, minQuality, maxQuality *float64) (*database.GetAllCounterpartiesByClientResult, error)
	BulkUpdateCounterparties(ids []int, updates map[string]interface{}) (map[string]interface{}, error)
	BulkDeleteCounterparties(ids []int) (map[string]interface{}, error)
	DeleteCounterpartyDuplicateGroup(projectID int, groupID string) error
	ResolveCounterpartyDuplicateGroup(projectID int, groupID string) (*database.NormalizedCounterparty, error)
}

// CounterpartyHandler обработчик для контрагентов
type CounterpartyHandler struct {
	*BaseHandler
	counterpartyService CounterpartyService
	enrichmentFactory   *enrichment.EnricherFactory
	exportManager       *CounterpartyExportManager
	logFunc             func(entry interface{}) // server.LogEntry, но без прямого импорта
}

// NewCounterpartyHandler создает новый обработчик контрагентов
func NewCounterpartyHandler(baseHandler *BaseHandler, counterpartyService CounterpartyService, logFunc func(entry interface{})) *CounterpartyHandler {
	return &CounterpartyHandler{
		BaseHandler:         baseHandler,
		counterpartyService: counterpartyService,
		logFunc:             logFunc,
	}
}

// SetEnrichmentFactory устанавливает фабрику обогащения
func (h *CounterpartyHandler) SetEnrichmentFactory(factory *enrichment.EnricherFactory) {
	h.enrichmentFactory = factory
}

// SetExportManager подключает очередь тяжелых выгрузок.
func (h *CounterpartyHandler) SetExportManager(manager *CounterpartyExportManager) {
	h.exportManager = manager
}

// HandleNormalizedCounterpartyStats обрабатывает запрос статистики нормализованных контрагентов
func (h *CounterpartyHandler) HandleNormalizedCounterpartyStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	stats, err := h.counterpartyService.GetNormalizedCounterpartyStats(projectID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting counterparty stats: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleGetNormalizedCounterparty обрабатывает запрос получения контрагента по ID
func (h *CounterpartyHandler) HandleGetNormalizedCounterparty(w http.ResponseWriter, r *http.Request, id int) {
	counterparty, err := h.counterpartyService.GetNormalizedCounterparty(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			h.WriteJSONError(w, r, "Counterparty not found", http.StatusNotFound)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get counterparty: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSONResponse(w, r, counterparty, http.StatusOK)
}

// HandleUpdateNormalizedCounterparty обрабатывает запрос обновления контрагента
func (h *CounterpartyHandler) HandleUpdateNormalizedCounterparty(w http.ResponseWriter, r *http.Request, id int) {
	var req struct {
		NormalizedName       string  `json:"normalized_name"`
		TaxID                string  `json:"tax_id"`
		KPP                  string  `json:"kpp"`
		BIN                  string  `json:"bin"`
		LegalAddress         string  `json:"legal_address"`
		PostalAddress        string  `json:"postal_address"`
		ContactPhone         string  `json:"contact_phone"`
		ContactEmail         string  `json:"contact_email"`
		ContactPerson        string  `json:"contact_person"`
		LegalForm            string  `json:"legal_form"`
		BankName             string  `json:"bank_name"`
		BankAccount          string  `json:"bank_account"`
		CorrespondentAccount string  `json:"correspondent_account"`
		BIK                  string  `json:"bik"`
		QualityScore         float64 `json:"quality_score"`
		SourceEnrichment     string  `json:"source_enrichment"`
		Subcategory          string  `json:"subcategory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем существование контрагента
	_, err := h.counterpartyService.GetNormalizedCounterparty(id)
	if err != nil {
		h.WriteJSONError(w, r, "Counterparty not found", http.StatusNotFound)
		return
	}

	// Обновляем контрагента
	err = h.counterpartyService.UpdateNormalizedCounterparty(
		id,
		req.NormalizedName,
		req.TaxID, req.KPP, req.BIN,
		req.LegalAddress, req.PostalAddress,
		req.ContactPhone, req.ContactEmail,
		req.ContactPerson, req.LegalForm,
		req.BankName, req.BankAccount,
		req.CorrespondentAccount, req.BIK,
		req.QualityScore,
		req.SourceEnrichment,
		req.Subcategory,
	)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error updating counterparty: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to update counterparty: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем обновленного контрагента
	updated, err := h.counterpartyService.GetNormalizedCounterparty(id)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get updated counterparty: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":      true,
		"message":      "Counterparty updated successfully",
		"counterparty": updated,
	}, http.StatusOK)
}

// HandleGetCounterpartyDuplicates обрабатывает запрос получения дубликатов контрагентов
func (h *CounterpartyHandler) HandleGetCounterpartyDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	duplicateGroups, err := h.counterpartyService.GetCounterpartyDuplicates(projectID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting counterparty duplicates: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get duplicates: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"total_groups": len(duplicateGroups),
		"groups":       duplicateGroups,
	}, http.StatusOK)
}

// HandleMergeCounterpartyDuplicates обрабатывает запрос слияния дубликатов контрагентов
func (h *CounterpartyHandler) HandleMergeCounterpartyDuplicates(w http.ResponseWriter, r *http.Request, groupID int) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MasterID int   `json:"master_id"`
		MergeIDs []int `json:"merge_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.MasterID == 0 {
		h.WriteJSONError(w, r, "master_id is required", http.StatusBadRequest)
		return
	}

	updated, err := h.counterpartyService.MergeCounterpartyDuplicates(req.MasterID, req.MergeIDs)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error merging counterparty duplicates: %v", err),
			Endpoint:  r.URL.Path,
		})
		if strings.Contains(err.Error(), "not found") {
			h.WriteJSONError(w, r, "Master counterparty not found", http.StatusNotFound)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to merge duplicates: %v", err), http.StatusInternalServerError)
		}
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":      true,
		"message":      "Duplicates merged successfully",
		"counterparty": updated,
	}, http.StatusOK)
}

// HandleAutoMapCounterparties обрабатывает запрос на автоматический мэппинг контрагентов
func (h *CounterpartyHandler) HandleAutoMapCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Запускаем мэппинг в фоне
	go func() {
		serviceDB := h.counterpartyService.GetServiceDB()
		if serviceDB == nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   "ServiceDB is nil, cannot start auto-mapping",
				Endpoint:  r.URL.Path,
			})
			return
		}

		mapper := normalization.NewCounterpartyMapper(serviceDB)
		if err := mapper.MapAllCounterpartiesForProject(projectID); err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Error auto-mapping counterparties for project %d: %v", projectID, err),
				Endpoint:  r.URL.Path,
			})
		} else {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "INFO",
				Message:   fmt.Sprintf("Successfully completed auto-mapping for project %d", projectID),
				Endpoint:  r.URL.Path,
			})
		}
	}()

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Auto-mapping started",
	}, http.StatusAccepted)
}

// HandleMappingStatus обрабатывает запрос на получение статуса мэппинга
func (h *CounterpartyHandler) HandleMappingStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем статистику по контрагентам проекта
	stats, err := h.counterpartyService.GetNormalizedCounterpartyStats(projectID)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get mapping status: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем конфигурацию проекта
	serviceDB := h.counterpartyService.GetServiceDB()
	if serviceDB == nil {
		h.WriteJSONError(w, r, "ServiceDB is not available", http.StatusInternalServerError)
		return
	}

	config, err := serviceDB.GetProjectNormalizationConfig(projectID)
	if err != nil {
		// Если конфигурация не найдена, используем значения по умолчанию
		config = &database.ProjectNormalizationConfig{
			ClientProjectID:         projectID,
			AutoMapCounterparties:   true,
			AutoMergeDuplicates:     true,
			MasterSelectionStrategy: "max_data",
		}
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"project_id":                projectID,
		"auto_map_counterparties":   config.AutoMapCounterparties,
		"auto_merge_duplicates":     config.AutoMergeDuplicates,
		"master_selection_strategy": config.MasterSelectionStrategy,
		"stats":                     stats,
	}, http.StatusOK)
}

// HandleUpdateNormalizationConfig обрабатывает запрос на обновление конфигурации нормализации проекта
func (h *CounterpartyHandler) HandleUpdateNormalizationConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	var req struct {
		AutoMapCounterparties   *bool   `json:"auto_map_counterparties"`
		AutoMergeDuplicates     *bool   `json:"auto_merge_duplicates"`
		MasterSelectionStrategy *string `json:"master_selection_strategy"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	serviceDB := h.counterpartyService.GetServiceDB()
	if serviceDB == nil {
		h.WriteJSONError(w, r, "ServiceDB is not available", http.StatusInternalServerError)
		return
	}

	// Получаем текущую конфигурацию
	config, err := serviceDB.GetProjectNormalizationConfig(projectID)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get current config: %v", err), http.StatusInternalServerError)
		return
	}

	// Обновляем только переданные поля
	if req.AutoMapCounterparties != nil {
		config.AutoMapCounterparties = *req.AutoMapCounterparties
	}
	if req.AutoMergeDuplicates != nil {
		config.AutoMergeDuplicates = *req.AutoMergeDuplicates
	}
	if req.MasterSelectionStrategy != nil {
		// Валидация стратегии
		validStrategies := map[string]bool{
			"max_data":      true,
			"max_quality":   true,
			"max_databases": true,
		}
		if !validStrategies[*req.MasterSelectionStrategy] {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid master_selection_strategy: %s. Valid values: max_data, max_quality, max_databases", *req.MasterSelectionStrategy), http.StatusBadRequest)
			return
		}
		config.MasterSelectionStrategy = *req.MasterSelectionStrategy
	}

	// Сохраняем обновленную конфигурацию
	if err := serviceDB.UpdateProjectNormalizationConfig(projectID, config); err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error updating normalization config: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to update config: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success":                   true,
		"message":                   "Configuration updated successfully",
		"project_id":                projectID,
		"auto_map_counterparties":   config.AutoMapCounterparties,
		"auto_merge_duplicates":     config.AutoMergeDuplicates,
		"master_selection_strategy": config.MasterSelectionStrategy,
	}, http.StatusOK)
}

// HandleNormalizedCounterpartyRoutes обрабатывает вложенные маршруты для нормализованных контрагентов
func (h *CounterpartyHandler) HandleNormalizedCounterpartyRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/counterparties/normalized/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.WriteJSONError(w, r, "Invalid request path", http.StatusBadRequest)
		return
	}

	// Обработка stats
	if len(parts) == 1 && parts[0] == "stats" {
		h.HandleNormalizedCounterpartyStats(w, r)
		return
	}

	// Обработка duplicates - получение групп дубликатов
	if len(parts) == 1 && parts[0] == "duplicates" {
		if r.Method == http.MethodGet {
			h.HandleGetCounterpartyDuplicates(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка auto-map: /api/counterparties/normalized/auto-map
	if len(parts) == 1 && parts[0] == "auto-map" {
		h.HandleAutoMapCounterparties(w, r)
		return
	}

	// Обработка mapping-status: /api/counterparties/normalized/mapping-status
	if len(parts) == 1 && parts[0] == "mapping-status" {
		h.HandleMappingStatus(w, r)
		return
	}

	// Обработка merge дубликатов: /api/counterparties/normalized/duplicates/{groupId}/merge
	if len(parts) == 3 && parts[0] == "duplicates" && parts[2] == "merge" {
		if r.Method == http.MethodPost {
			groupId, err := ValidateIDPathParam(parts[1], "group_id")
			if err != nil {
				h.WriteJSONError(w, r, fmt.Sprintf("Invalid duplicate group ID: %s", err.Error()), http.StatusBadRequest)
				return
			}
			h.HandleMergeCounterpartyDuplicates(w, r, groupId)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}
}

// HandleCounterpartyDuplicatesRoutes обрабатывает маршруты для дубликатов контрагентов
func (h *CounterpartyHandler) HandleCounterpartyDuplicatesRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/counterparties/duplicates")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	// Если путь пустой или только "duplicates", возвращаем список дубликатов
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		if r.Method == http.MethodGet {
			h.HandleGetCounterpartyDuplicates(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка merge: /api/counterparties/duplicates/{groupId}/merge
	if len(parts) == 2 && parts[1] == "merge" {
		if r.Method == http.MethodPost {
			groupId, err := ValidateIDPathParam(parts[0], "group_id")
			if err != nil {
				h.WriteJSONError(w, r, fmt.Sprintf("Invalid duplicate group ID: %s", err.Error()), http.StatusBadRequest)
				return
			}
			h.HandleMergeCounterpartyDuplicates(w, r, groupId)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Если путь не распознан, возвращаем 404
	http.NotFound(w, r)

	// Обработка конкретного контрагента по ID: /api/counterparties/normalized/{id}
	if len(parts) == 1 {
		id, err := ValidateIDPathParam(parts[0], "counterparty_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid counterparty ID: %s", err.Error()), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			h.HandleGetNormalizedCounterparty(w, r, id)
		case http.MethodPut, http.MethodPatch:
			h.HandleUpdateNormalizedCounterparty(w, r, id)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	h.WriteJSONError(w, r, "Not found", http.StatusNotFound)
}

// HandleNormalizedCounterparties обрабатывает запросы к /api/counterparties/normalized
func (h *CounterpartyHandler) HandleNormalizedCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Поддерживаем два режима: по проекту или по клиенту
	projectIDStr := r.URL.Query().Get("project_id")
	clientIDStr := r.URL.Query().Get("client_id")

	// Получаем параметры пагинации
	page, limit, err := ValidatePaginationParams(r, 1, 100, 1000)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid pagination params: %v", err), http.StatusBadRequest)
		return
	}

	// Поддержка offset для обратной совместимости
	offsetStr := r.URL.Query().Get("offset")
	offset := 0
	if offsetStr != "" {
		offset, err = ValidateIntParam(r, "offset", 0, 0, 0)
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid offset: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// Вычисляем offset из page
		offset = (page - 1) * limit
	}

	// Получаем параметры фильтрации
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err := ValidateSearchQuery(search, 500); err != nil {
		search = ""
	}
	enrichment := strings.TrimSpace(r.URL.Query().Get("enrichment"))
	subcategory := strings.TrimSpace(r.URL.Query().Get("subcategory"))

	var counterparties []*database.NormalizedCounterparty
	var projects []*database.ClientProject
	var totalCount int

	if clientIDStr != "" {
		// Режим получения по клиенту (все проекты)
		clientID, err := ValidateIDParam(r, "client_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
			return
		}

		var projectID *int
		if projectIDStr != "" {
			pID, err := ValidateIDParam(r, "project_id")
			if err == nil {
				projectID = &pID
			}
		}

		counterparties, projects, totalCount, err = h.counterpartyService.GetNormalizedCounterpartiesByClient(clientID, projectID, offset, limit, search, enrichment, subcategory)
		if err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to get normalized counterparties for client_id %d: %v", clientID, err),
				Endpoint:  "/api/counterparties/normalized",
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
			return
		}
	} else if projectIDStr != "" {
		// Режим получения по проекту
		projectID, err := ValidateIDParam(r, "project_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid project_id: %v", err), http.StatusBadRequest)
			return
		}

		// Получаем нормализованных контрагентов
		// GetNormalizedCounterparties принимает: projectID, limit, offset, search, taxID, bin
		// Но мы используем enrichment и subcategory, поэтому передаем пустые строки для taxID и bin
		counterparties, totalCount, err = h.counterpartyService.GetNormalizedCounterparties(projectID, limit, offset, search, "", "")
		if err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to get normalized counterparties for project_id %d: %v", projectID, err),
				Endpoint:  "/api/counterparties/normalized",
			})
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get normalized counterparties: %v", err), http.StatusInternalServerError)
			return
		}

		// Получаем информацию о проекте
		project, err := h.counterpartyService.GetClientProject(projectID)
		if err != nil {
			h.WriteJSONError(w, r, "Project not found", http.StatusNotFound)
			return
		}
		projects = []*database.ClientProject{project}
	} else {
		h.WriteJSONError(w, r, "Either project_id or client_id is required", http.StatusBadRequest)
		return
	}

	// Формируем ответ с информацией о проектах
	projectsInfo := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		projectsInfo[i] = map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		}
	}

	// Логирование успешного ответа
	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("GetNormalizedCounterparties success - total: %d, returned: %d, page: %d, limit: %d",
			totalCount, len(counterparties), page, limit),
		Endpoint: "/api/counterparties/normalized",
	})

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"counterparties": counterparties,
		"projects":       projectsInfo,
		"total":          totalCount,
		"offset":         offset,
		"limit":          limit,
		"page":           page,
	}, http.StatusOK)
}

// HandleGetAllCounterparties обрабатывает запросы к /api/counterparties/all
func (h *CounterpartyHandler) HandleGetAllCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Валидация обязательного параметра client_id
	clientID, err := ValidateIntParam(r, "client_id", 0, 1, 0)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
		return
	}
	if clientID <= 0 {
		h.WriteJSONError(w, r, "client_id is required and must be positive", http.StatusBadRequest)
		return
	}

	// Валидация опционального параметра project_id
	var projectID *int
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr != "" {
		pID, err := ValidateIntParam(r, "project_id", 0, 1, 0)
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid project_id: %v", err), http.StatusBadRequest)
			return
		}
		if pID > 0 {
			projectID = &pID
		}
	}

	// Валидация параметров пагинации
	offset, err := ValidateIntParam(r, "offset", 0, 0, 0)
	if err != nil {
		offset = 0
	}
	if offset < 0 {
		offset = 0
	}

	limit, err := ValidateIntParam(r, "limit", 100, 1, 100000)
	if err != nil {
		limit = 100
	}

	limitClamped := false
	if limit > maxCounterpartiesLimit {
		limit = maxCounterpartiesLimit
		limitClamped = true
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("GetAllCounterparties limit parameter truncated to %d", maxCounterpartiesLimit),
			Endpoint:  "/api/counterparties/all",
		})
	}

	loadAllParam := strings.EqualFold(r.URL.Query().Get("load_all"), "true") || r.URL.Query().Get("load_all") == "1"
	if loadAllParam {
		if limit < maxCounterpartiesLimit {
			limit = maxCounterpartiesLimit
		}
		limitClamped = true
	}

	// Валидация параметра поиска
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err := ValidateSearchQuery(search, 500); err != nil {
		search = ""
	}

	// Валидация параметра source
	source := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
	if err := ValidateEnumParam(source, "source", []string{"database", "normalized"}, false); err != nil {
		source = ""
	}

	// Валидация параметров сортировки
	sortBy := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_by")))
	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if err := ValidateSortParams(sortBy, order, []string{"name", "quality", "source", "id", ""}); err != nil {
		sortBy = ""
		order = ""
	}

	// Получаем параметры фильтрации по качеству
	var minQuality, maxQuality *float64
	if minQualityStr := r.URL.Query().Get("min_quality"); minQualityStr != "" {
		if q, err := strconv.ParseFloat(minQualityStr, 64); err == nil {
			minQuality = &q
		}
	}
	if maxQualityStr := r.URL.Query().Get("max_quality"); maxQualityStr != "" {
		if q, err := strconv.ParseFloat(maxQualityStr, 64); err == nil {
			maxQuality = &q
		}
	}

	// Логирование запроса
	minQStr := "nil"
	maxQStr := "nil"
	if minQuality != nil {
		minQStr = fmt.Sprintf("%.2f", *minQuality)
	}
	if maxQuality != nil {
		maxQStr = fmt.Sprintf("%.2f", *maxQuality)
	}
	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("GetAllCounterparties request - client_id: %d, project_id: %v, offset: %d, limit: %d, search: %q, source: %q, sort_by: %q, order: %q, min_quality: %s, max_quality: %s",
			clientID, projectID, offset, limit, search, source, sortBy, order, minQStr, maxQStr),
		Endpoint: "/api/counterparties/all",
	})

	isHeavyRequest := loadAllParam || limit >= heavyCounterpartiesThreshold
	if isHeavyRequest {
		h.handleStreamAllCounterparties(w, r, clientID, projectID, offset, limit, search, source, minQuality, maxQuality, limitClamped)
		return
	}

	// Получаем всех контрагентов через сервис
	result, err := h.counterpartyService.GetAllCounterpartiesByClient(clientID, projectID, offset, limit, search, source, sortBy, order, minQuality, maxQuality)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get counterparties for client_id %d: %v", clientID, err),
			Endpoint:  "/api/counterparties/all",
		})
		// Проверяем тип ошибки для более точного HTTP статуса
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not exist") {
			h.WriteJSONError(w, r, fmt.Sprintf("Client or project not found: %v", err), http.StatusNotFound)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Проверяем, что result не nil
	if result == nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "WARN",
			Message:   fmt.Sprintf("GetAllCounterpartiesByClient returned nil result for client_id %d", clientID),
			Endpoint:  "/api/counterparties/all",
		})
		// Возвращаем пустой результат вместо ошибки
		h.WriteJSONResponse(w, r, map[string]interface{}{
			"counterparties": []interface{}{},
			"projects":       []interface{}{},
			"total":          0,
			"offset":         offset,
			"limit":          limit,
			"stats":          map[string]interface{}{},
		}, http.StatusOK)
		return
	}

	// Формируем ответ с информацией о проектах
	projectsInfo := make([]map[string]interface{}, len(result.Projects))
	for i, p := range result.Projects {
		projectsInfo[i] = map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		}
	}

	// Логирование успешного ответа
	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("GetAllCounterparties success - client_id: %d, total: %d, returned: %d, processing_time: %dms",
			clientID, result.TotalCount, len(result.Counterparties), result.Stats.ProcessingTimeMs),
		Endpoint: "/api/counterparties/all",
	})

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"counterparties": result.Counterparties,
		"projects":       projectsInfo,
		"total":          result.TotalCount,
		"offset":         offset,
		"limit":          limit,
		"limit_clamped":  limitClamped,
		"stats":          result.Stats,
	}, http.StatusOK)
}

func (h *CounterpartyHandler) handleStreamAllCounterparties(
	w http.ResponseWriter,
	r *http.Request,
	clientID int,
	projectID *int,
	offset, limit int,
	search, source string,
	minQuality, maxQuality *float64,
	limitClamped bool,
) {
	ctx := r.Context()

	if h.exportManager != nil {
		if err := h.exportManager.Acquire(ctx); err != nil {
			if errors.Is(err, ErrExportQueueBusy) {
				h.WriteJSONError(w, r, "Слишком много одновременных выгрузок. Попробуйте повторить позже.", http.StatusTooManyRequests)
			} else if errors.Is(err, context.Canceled) {
				// Клиент отменил запрос — просто выходим
			} else {
				h.WriteJSONError(w, r, fmt.Sprintf("Не удалось занять слот выгрузки: %v", err), http.StatusServiceUnavailable)
			}
			return
		}
		defer h.exportManager.Release()
	}

	serviceDB := h.counterpartyService.GetServiceDB()
	if serviceDB == nil {
		h.WriteJSONError(w, r, "Service database is not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(`{"counterparties":[`)); err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to start streaming counterparties: %v", err),
			Endpoint:  "/api/counterparties/all",
		})
		return
	}

	flusher, _ := w.(http.Flusher)
	firstRecord := true
	consumer := func(batch []*database.UnifiedCounterparty) error {
		for _, item := range batch {
			if !firstRecord {
				if _, err := w.Write([]byte(",")); err != nil {
					return err
				}
			} else {
				firstRecord = false
			}
			data, err := json.Marshal(item)
			if err != nil {
				return err
			}
			if _, err := w.Write(data); err != nil {
				return err
			}
		}
		if flusher != nil {
			flusher.Flush()
		}
		return nil
	}

	streamOpts := &database.CounterpartyStreamOptions{
		ClientID:           clientID,
		ProjectID:          projectID,
		Offset:             offset,
		Limit:              limit,
		Search:             search,
		Source:             source,
		MinQuality:         minQuality,
		MaxQuality:         maxQuality,
		BatchSize:          1000,
		ApplyQualityFilter: true,
		ApplyPagination:    true,
	}

	stats, projects, totalCount, err := serviceDB.StreamAllCounterpartiesByClient(ctx, streamOpts, consumer)
	if stats == nil {
		stats = &database.CounterpartiesStats{}
	}
	if err != nil && !errors.Is(err, database.ErrCounterpartyStreamLimitReached) {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Streaming counterparties failed: %v", err),
			Endpoint:  "/api/counterparties/all",
		})
		// Завершаем объект с описанием ошибки
		fmt.Fprintf(w, `],"error":%q}`, err.Error())
		return
	}

	projectsInfo := make([]map[string]interface{}, len(projects))
	for i, p := range projects {
		projectsInfo[i] = map[string]interface{}{
			"id":   p.ID,
			"name": p.Name,
		}
	}

	projectsJSON, _ := json.Marshal(projectsInfo)
	statsJSON, _ := json.Marshal(stats)

	if _, err := w.Write([]byte(`],"projects":`)); err != nil {
		return
	}
	if _, err := w.Write(projectsJSON); err != nil {
		return
	}
	meta := fmt.Sprintf(`,"total":%d,"offset":%d,"limit":%d,"limit_clamped":%t`, totalCount, offset, limit, limitClamped)
	if _, err := w.Write([]byte(meta)); err != nil {
		return
	}
	if _, err := w.Write([]byte(`,"stats":`)); err != nil {
		return
	}
	if _, err := w.Write(statsJSON); err != nil {
		return
	}
	if _, err := w.Write([]byte(`}`)); err != nil {
		return
	}
	if flusher != nil {
		flusher.Flush()
	}

	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("Streamed counterparties - client_id: %d, total: %d, returned_limit: %d",
			clientID, totalCount, limit),
		Endpoint: "/api/counterparties/all",
	})
}

// HandleExportAllCounterparties обрабатывает запросы к /api/counterparties/all/export
func (h *CounterpartyHandler) HandleExportAllCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Валидация обязательного параметра client_id
	clientID, err := ValidateIntParam(r, "client_id", 0, 1, 0)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid client_id: %v", err), http.StatusBadRequest)
		return
	}
	if clientID <= 0 {
		h.WriteJSONError(w, r, "client_id is required and must be positive", http.StatusBadRequest)
		return
	}

	// Валидация опционального параметра project_id
	var projectID *int
	projectIDStr := r.URL.Query().Get("project_id")
	if projectIDStr != "" {
		pID, err := ValidateIntParam(r, "project_id", 0, 1, 0)
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid project_id: %v", err), http.StatusBadRequest)
			return
		}
		if pID > 0 {
			projectID = &pID
		}
	}

	// Получаем параметры фильтрации (те же, что и в HandleGetAllCounterparties)
	search := strings.TrimSpace(r.URL.Query().Get("search"))
	if err := ValidateSearchQuery(search, 500); err != nil {
		search = ""
	}

	source := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("source")))
	if err := ValidateEnumParam(source, "source", []string{"database", "normalized"}, false); err != nil {
		source = ""
	}

	// Параметры сортировки
	sortBy := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("sort_by")))
	order := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("order")))
	if err := ValidateSortParams(sortBy, order, []string{"name", "quality", "source", "id", ""}); err != nil {
		sortBy = ""
		order = ""
	}

	// Валидация фильтров по качеству
	var minQuality, maxQuality *float64
	if minQualityStr := r.URL.Query().Get("min_quality"); minQualityStr != "" {
		if q, err := strconv.ParseFloat(minQualityStr, 64); err == nil && q >= 0 && q <= 1 {
			minQuality = &q
		}
	}
	if maxQualityStr := r.URL.Query().Get("max_quality"); maxQualityStr != "" {
		if q, err := strconv.ParseFloat(maxQualityStr, 64); err == nil && q >= 0 && q <= 1 {
			maxQuality = &q
		}
	}

	// Проверяем логику фильтров
	if minQuality != nil && maxQuality != nil && *minQuality > *maxQuality {
		h.WriteJSONError(w, r, "min_quality must be less than or equal to max_quality", http.StatusBadRequest)
		return
	}

	// Определяем формат экспорта (из query параметра или Accept header)
	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "text/csv") || strings.Contains(acceptHeader, "application/csv") {
			format = "csv"
		} else if strings.Contains(acceptHeader, "application/json") {
			format = "json"
		} else {
			format = "json" // по умолчанию JSON
		}
	}

	if format != "csv" && format != "json" {
		h.WriteJSONError(w, r, "Invalid format parameter. Must be 'csv' or 'json'", http.StatusBadRequest)
		return
	}

	// Получаем ВСЕ контрагенты (без пагинации для экспорта)
	result, err := h.counterpartyService.GetAllCounterpartiesByClient(clientID, projectID, 0, 1000000, search, source, sortBy, order, minQuality, maxQuality)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get counterparties for export (client_id %d): %v", clientID, err),
			Endpoint:  "/api/counterparties/all/export",
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	// Логирование экспорта
	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message: fmt.Sprintf("ExportAllCounterparties - client_id: %d, project_id: %v, format: %s, total: %d",
			clientID, projectID, format, result.TotalCount),
		Endpoint: "/api/counterparties/all/export",
	})

	// Генерируем имя файла
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("counterparties_client_%d_%s", clientID, timestamp)
	if projectID != nil {
		filename = fmt.Sprintf("counterparties_client_%d_project_%d_%s", clientID, *projectID, timestamp)
	}

	switch format {
	case "csv":
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.csv", filename))
		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		// Заголовки CSV
		headers := []string{
			"ID", "Name", "Source", "Project ID", "Project Name",
			"Database ID", "Database Name", "Tax ID (INN)", "KPP", "BIN",
			"Legal Address", "Postal Address", "Contact Phone", "Contact Email", "Contact Person",
			"Quality Score", "Reference", "Code", "Normalized Name", "Source Name", "Source Reference",
		}
		if err := csvWriter.Write(headers); err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to write CSV headers: %v", err),
				Endpoint:  "/api/counterparties/all/export",
			})
			return
		}

		// Данные
		for _, cp := range result.Counterparties {
			qualityScore := ""
			if cp.QualityScore != nil {
				qualityScore = fmt.Sprintf("%.2f", *cp.QualityScore)
			}
			databaseID := ""
			if cp.DatabaseID != nil {
				databaseID = fmt.Sprintf("%d", *cp.DatabaseID)
			}
			row := []string{
				fmt.Sprintf("%d", cp.ID),
				cp.Name,
				cp.Source,
				fmt.Sprintf("%d", cp.ProjectID),
				cp.ProjectName,
				databaseID,
				cp.DatabaseName,
				cp.TaxID,
				cp.KPP,
				cp.BIN,
				cp.LegalAddress,
				cp.PostalAddress,
				cp.ContactPhone,
				cp.ContactEmail,
				cp.ContactPerson,
				qualityScore,
				cp.Reference,
				cp.Code,
				cp.NormalizedName,
				cp.SourceName,
				cp.SourceReference,
			}
			if err := csvWriter.Write(row); err != nil {
				h.logFunc(LogEntry{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Message:   fmt.Sprintf("Failed to write CSV row: %v", err),
					Endpoint:  "/api/counterparties/all/export",
				})
				return
			}
		}

	default: // json
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.json", filename))

		exportData := map[string]interface{}{
			"client_id":      clientID,
			"project_id":     projectID,
			"export_date":    time.Now().Format(time.RFC3339),
			"format_version": "1.0",
			"total":          result.TotalCount,
			"stats":          result.Stats,
			"counterparties": result.Counterparties,
			"projects":       result.Projects,
		}

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(exportData); err != nil {
			h.logFunc(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to encode JSON: %v", err),
				Endpoint:  "/api/counterparties/all/export",
			})
			return
		}
	}
}

// HandleBulkUpdateCounterparties обрабатывает запросы к /api/counterparties/bulk/update
func (h *CounterpartyHandler) HandleBulkUpdateCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		IDs     []int                  `json:"ids"`
		Updates map[string]interface{} `json:"updates"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		h.WriteJSONError(w, r, "ids array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	result, err := h.counterpartyService.BulkUpdateCounterparties(req.IDs, req.Updates)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error bulk updating counterparties: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to bulk update counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	statusCode := http.StatusOK
	if failedCount, ok := result["failed_count"].(int); ok && failedCount > 0 {
		if successCount, ok := result["success_count"].(int); ok && successCount == 0 {
			statusCode = http.StatusInternalServerError
		}
	}

	h.WriteJSONResponse(w, r, result, statusCode)
}

// HandleBulkDeleteCounterparties обрабатывает запросы к /api/counterparties/bulk/delete
func (h *CounterpartyHandler) HandleBulkDeleteCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		IDs []int `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		h.WriteJSONError(w, r, "ids array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	result, err := h.counterpartyService.BulkDeleteCounterparties(req.IDs)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error bulk deleting counterparties: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to bulk delete counterparties: %v", err), http.StatusInternalServerError)
		return
	}

	statusCode := http.StatusOK
	if failedCount, ok := result["failed_count"].(int); ok && failedCount > 0 {
		if successCount, ok := result["success_count"].(int); ok && successCount == 0 {
			statusCode = http.StatusInternalServerError
		}
	}

	h.WriteJSONResponse(w, r, result, statusCode)
}

// HandleBulkEnrichCounterparties обрабатывает запросы к /api/counterparties/bulk/enrich
func (h *CounterpartyHandler) HandleBulkEnrichCounterparties(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	var req struct {
		IDs []int `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		h.WriteJSONError(w, r, "ids array is required and cannot be empty", http.StatusBadRequest)
		return
	}

	if h.enrichmentFactory == nil {
		h.WriteJSONError(w, r, "Enrichment is not configured", http.StatusServiceUnavailable)
		return
	}

	serviceDB := h.counterpartyService.GetServiceDB()
	if serviceDB == nil {
		h.WriteJSONError(w, r, "ServiceDB is not available", http.StatusInternalServerError)
		return
	}

	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, id := range req.IDs {
		// Получаем контрагента
		cp, err := serviceDB.GetNormalizedCounterparty(id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: not found", id))
			continue
		}

		// Определяем ИНН/БИН для обогащения
		inn := cp.TaxID
		bin := cp.BIN
		if inn == "" && bin == "" {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: INN or BIN is required", id))
			continue
		}

		// Выполняем обогащение
		response := h.enrichmentFactory.Enrich(inn, bin)
		if !response.Success {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: enrichment failed: %v", id, response.Errors))
			continue
		}

		// Берем лучший результат
		bestResult := h.enrichmentFactory.GetBestResult(response.Results)
		if bestResult == nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: no enrichment results available", id))
			continue
		}

		// Объединяем данные из обогащения
		normalizedName := cp.NormalizedName
		if bestResult.FullName != "" {
			normalizedName = bestResult.FullName
		}

		if bestResult.INN != "" {
			inn = bestResult.INN
		}
		if bestResult.BIN != "" {
			bin = bestResult.BIN
		}

		legalAddress := cp.LegalAddress
		if bestResult.LegalAddress != "" {
			legalAddress = bestResult.LegalAddress
		}

		contactPhone := cp.ContactPhone
		if bestResult.Phone != "" {
			contactPhone = bestResult.Phone
		}

		contactEmail := cp.ContactEmail
		if bestResult.Email != "" {
			contactEmail = bestResult.Email
		}

		// Обновляем контрагента
		err = serviceDB.UpdateNormalizedCounterparty(
			id,
			normalizedName,
			inn, cp.KPP, bin,
			legalAddress, cp.PostalAddress,
			contactPhone, contactEmail,
			cp.ContactPerson, cp.LegalForm,
			cp.BankName, cp.BankAccount,
			cp.CorrespondentAccount, cp.BIK,
			cp.QualityScore,
			bestResult.Source,
			cp.Subcategory,
		)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: failed to update: %v", id, err))
			continue
		}

		successCount++
	}

	response := map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(req.IDs),
		"success_count": successCount,
		"failed_count":  failedCount,
	}
	if len(errors) > 0 {
		response["errors"] = errors
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleCounterpartyDuplicates обрабатывает запросы к /api/counterparties/duplicates
func (h *CounterpartyHandler) HandleCounterpartyDuplicates(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	// Получаем параметры запроса
	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %v", err), http.StatusBadRequest)
		return
	}

	// Логирование запроса
	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("GetCounterpartyDuplicates request - project_id: %d", projectID),
		Endpoint:  "/api/counterparties/duplicates",
	})

	// Получаем дубликаты через сервис
	groups, err := h.counterpartyService.GetCounterpartyDuplicates(projectID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get counterparty duplicates for project_id %d: %v", projectID, err),
			Endpoint:  "/api/counterparties/duplicates",
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get duplicates: %v", err), http.StatusInternalServerError)
		return
	}

	// Подсчитываем общее количество дубликатов
	totalDuplicates := 0
	for _, group := range groups {
		if count, ok := group["count"].(int); ok {
			totalDuplicates += count
		}
	}

	// Логирование успешного ответа
	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("GetCounterpartyDuplicates success - project_id: %d, groups: %d, total_duplicates: %d", projectID, len(groups), totalDuplicates),
		Endpoint:  "/api/counterparties/duplicates",
	})

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"groups":           groups,
		"total_groups":     len(groups),
		"total_duplicates": totalDuplicates,
	}, http.StatusOK)
}

// HandleCounterpartyDuplicateRoutes обрабатывает запросы к /api/counterparties/duplicates/
// Это альтернативный обработчик для вложенных маршрутов дубликатов
func (h *CounterpartyHandler) HandleCounterpartyDuplicateRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/counterparties/duplicates")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")

	// Если путь пустой, возвращаем список дубликатов
	if len(parts) == 0 || (len(parts) == 1 && parts[0] == "") {
		if r.Method == http.MethodGet {
			h.HandleGetCounterpartyDuplicates(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка конкретной группы дубликатов: /api/counterparties/duplicates/{groupId}
	if len(parts) == 1 {
		groupId, err := ValidateIDPathParam(parts[0], "group_id")
		if err != nil {
			h.WriteJSONError(w, r, fmt.Sprintf("Invalid duplicate group ID: %s", err.Error()), http.StatusBadRequest)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Получаем информацию о конкретной группе дубликатов
			h.HandleGetCounterpartyDuplicateGroup(w, r, groupId)
		case http.MethodDelete:
			// Удаление группы дубликатов (если нужно)
			h.HandleDeleteCounterpartyDuplicateGroup(w, r, groupId)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка merge: /api/counterparties/duplicates/{groupId}/merge
	if len(parts) == 2 && parts[1] == "merge" {
		if r.Method == http.MethodPost {
			groupId, err := ValidateIDPathParam(parts[0], "group_id")
			if err != nil {
				h.WriteJSONError(w, r, fmt.Sprintf("Invalid duplicate group ID: %s", err.Error()), http.StatusBadRequest)
				return
			}
			h.HandleMergeCounterpartyDuplicates(w, r, groupId)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Обработка resolve: /api/counterparties/duplicates/{groupId}/resolve
	if len(parts) == 2 && parts[1] == "resolve" {
		if r.Method == http.MethodPut || r.Method == http.MethodPost {
			groupId, err := ValidateIDPathParam(parts[0], "group_id")
			if err != nil {
				h.WriteJSONError(w, r, fmt.Sprintf("Invalid duplicate group ID: %s", err.Error()), http.StatusBadRequest)
				return
			}
			h.HandleResolveCounterpartyDuplicateGroup(w, r, groupId)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
		return
	}

	// Если путь не распознан, возвращаем 404
	h.WriteJSONError(w, r, "Not found", http.StatusNotFound)
}

// HandleGetCounterpartyDuplicateGroup получает информацию о конкретной группе дубликатов
func (h *CounterpartyHandler) HandleGetCounterpartyDuplicateGroup(w http.ResponseWriter, r *http.Request, groupID int) {
	// Получаем project_id из query параметров
	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем все группы дубликатов
	duplicateGroups, err := h.counterpartyService.GetCounterpartyDuplicates(projectID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting counterparty duplicates: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get duplicates: %v", err), http.StatusInternalServerError)
		return
	}

	// Ищем нужную группу по индексу (groupID - это индекс в массиве групп)
	if groupID < 0 || groupID >= len(duplicateGroups) {
		h.WriteJSONError(w, r, fmt.Sprintf("Duplicate group %d not found", groupID), http.StatusNotFound)
		return
	}

	foundGroup := duplicateGroups[groupID]

	h.WriteJSONResponse(w, r, foundGroup, http.StatusOK)
}

// HandleDeleteCounterpartyDuplicateGroup удаляет группу дубликатов
func (h *CounterpartyHandler) HandleDeleteCounterpartyDuplicateGroup(w http.ResponseWriter, r *http.Request, groupID int) {
	// Получаем project_id из query параметров
	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем все группы дубликатов, чтобы найти нужную группу
	duplicateGroups, err := h.counterpartyService.GetCounterpartyDuplicates(projectID)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get duplicates: %v", err), http.StatusInternalServerError)
		return
	}

	// Проверяем, что groupID валиден
	if groupID < 0 || groupID >= len(duplicateGroups) {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid group ID: %d", groupID), http.StatusBadRequest)
		return
	}

	// Получаем tax_id из группы
	group := duplicateGroups[groupID]
	taxID, ok := group["tax_id"].(string)
	if !ok || taxID == "" {
		h.WriteJSONError(w, r, "Group tax_id not found", http.StatusInternalServerError)
		return
	}

	// Удаляем группу дубликатов
	if err := h.counterpartyService.DeleteCounterpartyDuplicateGroup(projectID, taxID); err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to delete duplicate group: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"message":  "Duplicate group deleted successfully",
		"group_id": groupID,
		"tax_id":   taxID,
	}, http.StatusOK)
}

// HandleResolveCounterpartyDuplicateGroup разрешает группу дубликатов (объединяет их в одного контрагента)
func (h *CounterpartyHandler) HandleResolveCounterpartyDuplicateGroup(w http.ResponseWriter, r *http.Request, groupID int) {
	// Получаем project_id из query параметров
	projectID, err := ValidateIDParam(r, "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("project_id is required: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем все группы дубликатов, чтобы найти нужную группу
	duplicateGroups, err := h.counterpartyService.GetCounterpartyDuplicates(projectID)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get duplicates: %v", err), http.StatusInternalServerError)
		return
	}

	// Проверяем, что groupID валиден
	if groupID < 0 || groupID >= len(duplicateGroups) {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid group ID: %d", groupID), http.StatusBadRequest)
		return
	}

	// Получаем tax_id из группы
	group := duplicateGroups[groupID]
	taxID, ok := group["tax_id"].(string)
	if !ok || taxID == "" {
		h.WriteJSONError(w, r, "Group tax_id not found", http.StatusInternalServerError)
		return
	}

	// Разрешаем группу дубликатов (объединяем их)
	master, err := h.counterpartyService.ResolveCounterpartyDuplicateGroup(projectID, taxID)
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to resolve duplicate group: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"message":             "Duplicate group resolved successfully",
		"group_id":            groupID,
		"tax_id":              taxID,
		"master_counterparty": master,
	}, http.StatusOK)
}

// HandleCounterpartyNormalizationStopCheckPerformance обрабатывает запросы к /api/counterparty/normalization/stop-check/performance
func (h *CounterpartyHandler) HandleCounterpartyNormalizationStopCheckPerformance(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	stats := normalization.GetStopCheckStats()

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Counterparty stop-check performance metrics requested",
			Endpoint:  r.URL.Path,
		})
	}

	h.WriteJSONResponse(w, r, stats, http.StatusOK)
}

// HandleCounterpartyNormalizationStopCheckPerformanceReset обрабатывает запросы к /api/counterparty/normalization/stop-check/performance/reset
func (h *CounterpartyHandler) HandleCounterpartyNormalizationStopCheckPerformanceReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodDelete {
		h.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	normalization.ResetStopCheckMetrics()

	if h.logFunc != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "INFO",
			Message:   "Counterparty stop-check performance metrics reset",
			Endpoint:  r.URL.Path,
		})
	}

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"message": "Stop check performance metrics reset successfully",
	}, http.StatusOK)
}
