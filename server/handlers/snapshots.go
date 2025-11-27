package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/server/services"
)

// convertCreatedByToString конвертирует *int в string, обрабатывая nil
func convertCreatedByToString(createdBy *int) string {
	if createdBy == nil {
		return ""
	}
	return fmt.Sprintf("%d", *createdBy)
}

// SnapshotHandler обработчик для работы со срезами данных
type SnapshotHandler struct {
	*BaseHandler
	snapshotService *services.SnapshotService
	logFunc         func(entry interface{}) // server.LogEntry, но без прямого импорта
	serviceDB       interface {
		GetClientProject(projectID int) (*database.ClientProject, error)
	}
	// Функции от Server
	normalizeSnapshotFunc      func(int, interface{}) (interface{}, error)
	compareSnapshotIterations  func(int) (interface{}, error)
	calculateSnapshotMetrics  func(int) (interface{}, error)
	getSnapshotEvolution       func(int) (interface{}, error)
	createAutoSnapshotFunc     func(int, int, string, string) (*database.DataSnapshot, error)
}

// NewSnapshotHandler создает новый обработчик срезов
func NewSnapshotHandler(
	baseHandler *BaseHandler,
	snapshotService *services.SnapshotService,
	logFunc func(entry interface{}), // server.LogEntry, но без прямого импорта
	serviceDB interface {
		GetClientProject(projectID int) (*database.ClientProject, error)
	},
	normalizeSnapshotFunc func(int, interface{}) (interface{}, error),
	compareSnapshotIterations func(int) (interface{}, error),
	calculateSnapshotMetrics func(int) (interface{}, error),
	getSnapshotEvolution func(int) (interface{}, error),
	createAutoSnapshotFunc func(int, int, string, string) (*database.DataSnapshot, error),
) *SnapshotHandler {
	return &SnapshotHandler{
		BaseHandler:              baseHandler,
		snapshotService:         snapshotService,
		logFunc:                  logFunc,
		serviceDB:                serviceDB,
		normalizeSnapshotFunc:    normalizeSnapshotFunc,
		compareSnapshotIterations: compareSnapshotIterations,
		calculateSnapshotMetrics: calculateSnapshotMetrics,
		getSnapshotEvolution:     getSnapshotEvolution,
		createAutoSnapshotFunc:   createAutoSnapshotFunc,
	}
}

// HandleSnapshotsRoutes обрабатывает запросы к /api/snapshots
func (h *SnapshotHandler) HandleSnapshotsRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.HandleListSnapshots(w, r)
	case http.MethodPost:
		h.HandleCreateSnapshot(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleListSnapshots обрабатывает запрос получения списка срезов
func (h *SnapshotHandler) HandleListSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := h.snapshotService.GetAllSnapshots()
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting snapshots: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	// Преобразуем в формат ответа
	type SnapshotResponse struct {
		ID           int        `json:"id"`
		Name         string     `json:"name"`
		Description  string     `json:"description"`
		CreatedBy    *int       `json:"created_by,omitempty"`
		CreatedAt    time.Time  `json:"created_at"`
		SnapshotType string     `json:"snapshot_type"`
		ProjectID    *int       `json:"project_id,omitempty"`
		ClientID     *int       `json:"client_id,omitempty"`
	}

	response := struct {
		Snapshots []SnapshotResponse `json:"snapshots"`
		Total     int                `json:"total"`
	}{
		Snapshots: make([]SnapshotResponse, 0, len(snapshots)),
		Total:     len(snapshots),
	}

	for _, snapshot := range snapshots {
		response.Snapshots = append(response.Snapshots, SnapshotResponse{
			ID:           snapshot.ID,
			Name:         snapshot.Name,
			Description:  snapshot.Description,
			CreatedBy:    snapshot.CreatedBy,
			CreatedAt:    snapshot.CreatedAt,
			SnapshotType: snapshot.SnapshotType,
			ProjectID:    snapshot.ProjectID,
			ClientID:     snapshot.ClientID,
		})
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleCreateSnapshot обрабатывает запрос создания среза
func (h *SnapshotHandler) HandleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name            string `json:"name"`
		Description     string `json:"description"`
		SnapshotType    string `json:"snapshot_type"`
		ProjectID       *int   `json:"project_id,omitempty"`
		ClientID        *int   `json:"client_id,omitempty"`
		IncludedUploads []struct {
			UploadID       int    `json:"upload_id"`
			IterationLabel string `json:"iteration_label,omitempty"`
			UploadOrder    int    `json:"upload_order,omitempty"`
		} `json:"included_uploads"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Name == "" {
		h.WriteJSONError(w, r, "Name is required", http.StatusBadRequest)
		return
	}

	if req.SnapshotType == "" {
		req.SnapshotType = "manual"
	}

	// Преобразуем IncludedUploads в []database.SnapshotUpload
	var snapshotUploads []database.SnapshotUpload
	for _, u := range req.IncludedUploads {
		snapshotUploads = append(snapshotUploads, database.SnapshotUpload{
			UploadID:       u.UploadID,
			IterationLabel: u.IterationLabel,
			UploadOrder:    u.UploadOrder,
		})
	}

	// Создаем срез
	snapshot := &database.DataSnapshot{
		Name:         req.Name,
		Description:  req.Description,
		SnapshotType: req.SnapshotType,
		ProjectID:    req.ProjectID,
		ClientID:     req.ClientID,
	}

	createdSnapshot, err := h.snapshotService.CreateSnapshot(snapshot, snapshotUploads)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error creating snapshot: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to create snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		ID           int       `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		CreatedBy    string    `json:"created_by"`
		CreatedAt    time.Time `json:"created_at"`
		SnapshotType string    `json:"snapshot_type"`
		ProjectID    *int      `json:"project_id,omitempty"`
		ClientID     *int      `json:"client_id,omitempty"`
		UploadCount  int       `json:"upload_count"`
	}{
		ID:           createdSnapshot.ID,
		Name:         createdSnapshot.Name,
		Description:  createdSnapshot.Description,
		CreatedBy:    func() string {
			if createdSnapshot.CreatedBy != nil {
				return fmt.Sprintf("%d", *createdSnapshot.CreatedBy)
			}
			return ""
		}(),
		CreatedAt:    createdSnapshot.CreatedAt,
		SnapshotType: createdSnapshot.SnapshotType,
		ProjectID:    createdSnapshot.ProjectID,
		ClientID:     createdSnapshot.ClientID,
		UploadCount:  len(snapshotUploads),
	}

	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Created snapshot: %s (ID: %d, uploads: %d)", createdSnapshot.Name, createdSnapshot.ID, len(snapshotUploads)),
		Endpoint:  "/api/snapshots",
	})

	h.WriteJSONResponse(w, r, response, http.StatusCreated)
}

// HandleCreateAutoSnapshot обрабатывает запрос создания автоматического среза
func (h *SnapshotHandler) HandleCreateAutoSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Type              string `json:"type"`
		ProjectID         int    `json:"project_id"`
		UploadsPerDatabase int  `json:"uploads_per_database"`
		Name              string `json:"name,omitempty"`
		Description       string `json:"description,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.WriteJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type != "latest_per_database" {
		h.WriteJSONError(w, r, "Unsupported auto snapshot type", http.StatusBadRequest)
		return
	}

	if req.UploadsPerDatabase <= 0 {
		req.UploadsPerDatabase = 3 // Значение по умолчанию
	}

	createdSnapshot, err := h.snapshotService.CreateAutoSnapshot(
		req.ProjectID,
		req.UploadsPerDatabase,
		req.Name,
		req.Description,
		h.createAutoSnapshotFunc,
	)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error creating auto snapshot: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to create auto snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	response := struct {
		ID           int       `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		CreatedBy    string    `json:"created_by"`
		CreatedAt    time.Time `json:"created_at"`
		SnapshotType string    `json:"snapshot_type"`
		ProjectID    *int      `json:"project_id,omitempty"`
		ClientID     *int      `json:"client_id,omitempty"`
	}{
		ID:           createdSnapshot.ID,
		Name:         createdSnapshot.Name,
		Description:  createdSnapshot.Description,
		CreatedBy:    func() string {
			if createdSnapshot.CreatedBy != nil {
				return fmt.Sprintf("%d", *createdSnapshot.CreatedBy)
			}
			return ""
		}(),
		CreatedAt:    createdSnapshot.CreatedAt,
		SnapshotType: createdSnapshot.SnapshotType,
		ProjectID:    createdSnapshot.ProjectID,
		ClientID:     createdSnapshot.ClientID,
	}

	h.WriteJSONResponse(w, r, response, http.StatusCreated)
}

// HandleSnapshotRoutes обрабатывает запросы к /api/snapshots/{id} и вложенным маршрутам
func (h *SnapshotHandler) HandleSnapshotRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/snapshots/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.WriteJSONError(w, r, "Snapshot ID required", http.StatusBadRequest)
		return
	}

	snapshotID, err := ValidateIDPathParam(parts[0], "snapshot_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid snapshot ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "normalize":
			if r.Method == http.MethodPost {
				h.HandleNormalizeSnapshot(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "comparison":
			if r.Method == http.MethodGet {
				h.HandleSnapshotComparison(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "metrics":
			if r.Method == http.MethodGet {
				h.HandleSnapshotMetrics(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "evolution":
			if r.Method == http.MethodGet {
				h.HandleSnapshotEvolution(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		default:
			h.WriteJSONError(w, r, "Unknown snapshot route", http.StatusNotFound)
			return
		}
	}

	// Обработка основных операций со срезом
	switch r.Method {
	case http.MethodGet:
		h.HandleGetSnapshot(w, r, snapshotID)
	case http.MethodDelete:
		h.HandleDeleteSnapshot(w, r, snapshotID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleGetSnapshot обрабатывает запрос получения среза
func (h *SnapshotHandler) HandleGetSnapshot(w http.ResponseWriter, r *http.Request, snapshotID int) {
	snapshot, uploads, err := h.snapshotService.GetSnapshotWithUploads(snapshotID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.WriteJSONError(w, r, "Snapshot not found", http.StatusNotFound)
		} else {
			h.WriteJSONError(w, r, fmt.Sprintf("Failed to get snapshot: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Преобразуем uploads в формат ответа
	type UploadListItem struct {
		UploadUUID     string     `json:"upload_uuid"`
		StartedAt      *time.Time `json:"started_at,omitempty"`
		CompletedAt    *time.Time `json:"completed_at,omitempty"`
		Status         string     `json:"status"`
		Version1C      string     `json:"version_1c,omitempty"`
		ConfigName     string     `json:"config_name,omitempty"`
		TotalConstants int        `json:"total_constants,omitempty"`
		TotalCatalogs  int        `json:"total_catalogs,omitempty"`
		TotalItems     int        `json:"total_items,omitempty"`
	}

	uploadList := make([]UploadListItem, 0, len(uploads))
	for _, upload := range uploads {
		// Конвертируем time.Time в *time.Time
		var startedAt *time.Time
		if !upload.StartedAt.IsZero() {
			startedAt = &upload.StartedAt
		}
		var completedAt *time.Time
		if upload.CompletedAt != nil {
			completedAt = upload.CompletedAt
		}
		uploadList = append(uploadList, UploadListItem{
			UploadUUID:     upload.UploadUUID,
			StartedAt:      startedAt,
			CompletedAt:    completedAt,
			Status:         upload.Status,
			Version1C:      upload.Version1C,
			ConfigName:     upload.ConfigName,
			TotalConstants: upload.TotalConstants,
			TotalCatalogs:  upload.TotalCatalogs,
			TotalItems:     upload.TotalItems,
		})
	}

	response := struct {
		ID           int            `json:"id"`
		Name         string         `json:"name"`
		Description  string         `json:"description"`
		CreatedBy    string         `json:"created_by"`
		CreatedAt    time.Time      `json:"created_at"`
		SnapshotType string         `json:"snapshot_type"`
		ProjectID    *int           `json:"project_id,omitempty"`
		ClientID     *int           `json:"client_id,omitempty"`
		Uploads      []UploadListItem `json:"uploads"`
		UploadCount  int            `json:"upload_count"`
	}{
		ID:           snapshot.ID,
		Name:         snapshot.Name,
		Description:  snapshot.Description,
		CreatedBy:    convertCreatedByToString(snapshot.CreatedBy),
		CreatedAt:    snapshot.CreatedAt,
		SnapshotType: snapshot.SnapshotType,
		ProjectID:    snapshot.ProjectID,
		ClientID:     snapshot.ClientID,
		Uploads:      uploadList,
		UploadCount:  len(uploadList),
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

// HandleDeleteSnapshot обрабатывает запрос удаления среза
func (h *SnapshotHandler) HandleDeleteSnapshot(w http.ResponseWriter, r *http.Request, snapshotID int) {
	err := h.snapshotService.DeleteSnapshot(snapshotID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error deleting snapshot: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to delete snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Deleted snapshot ID: %d", snapshotID),
		Endpoint:  "/api/snapshots",
	})

	h.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Snapshot deleted successfully",
	}, http.StatusOK)
}

// HandleNormalizeSnapshot обрабатывает запрос нормализации среза
func (h *SnapshotHandler) HandleNormalizeSnapshot(w http.ResponseWriter, r *http.Request, snapshotID int) {
	var req interface{}
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Если тело запроса пустое, используем значения по умолчанию
			req = map[string]interface{}{
				"use_ai":             false,
				"min_confidence":     0.7,
				"rate_limit_delay_ms": 100,
				"max_retries":        3,
			}
		}
	}

	result, err := h.snapshotService.NormalizeSnapshot(snapshotID, h.normalizeSnapshotFunc, req)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error normalizing snapshot %d: %v", snapshotID, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Normalization failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.logFunc(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Normalized snapshot %d", snapshotID),
		Endpoint:  "/api/snapshots/normalize",
	})

	h.WriteJSONResponse(w, r, result, http.StatusOK)
}

// HandleSnapshotComparison обрабатывает запрос сравнения итераций
func (h *SnapshotHandler) HandleSnapshotComparison(w http.ResponseWriter, r *http.Request, snapshotID int) {
	comparison, err := h.snapshotService.CompareSnapshotIterations(snapshotID, h.compareSnapshotIterations)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error comparing snapshot iterations %d: %v", snapshotID, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Comparison failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, comparison, http.StatusOK)
}

// HandleSnapshotMetrics обрабатывает запрос метрик среза
func (h *SnapshotHandler) HandleSnapshotMetrics(w http.ResponseWriter, r *http.Request, snapshotID int) {
	metrics, err := h.snapshotService.CalculateSnapshotMetrics(snapshotID, h.calculateSnapshotMetrics)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error calculating snapshot metrics %d: %v", snapshotID, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Metrics calculation failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, metrics, http.StatusOK)
}

// HandleSnapshotEvolution обрабатывает запрос эволюции среза
func (h *SnapshotHandler) HandleSnapshotEvolution(w http.ResponseWriter, r *http.Request, snapshotID int) {
	evolution, err := h.snapshotService.GetSnapshotEvolution(snapshotID, h.getSnapshotEvolution)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting snapshot evolution %d: %v", snapshotID, err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Evolution data failed: %v", err), http.StatusInternalServerError)
		return
	}

	h.WriteJSONResponse(w, r, evolution, http.StatusOK)
}

// HandleProjectSnapshotsRoutes обрабатывает запросы к /api/projects/{project_id}/snapshots
func (h *SnapshotHandler) HandleProjectSnapshotsRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[0] == "" || parts[1] != "snapshots" {
		// Это не маршрут для срезов проекта, передаем дальше
		http.NotFound(w, r)
		return
	}

	projectID, err := ValidateIDPathParam(parts[0], "project_id")
	if err != nil {
		h.WriteJSONError(w, r, fmt.Sprintf("Invalid project ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		h.HandleGetProjectSnapshots(w, r, projectID)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// HandleGetProjectSnapshots обрабатывает запрос получения срезов проекта
func (h *SnapshotHandler) HandleGetProjectSnapshots(w http.ResponseWriter, r *http.Request, projectID int) {
	snapshots, err := h.snapshotService.GetSnapshotsByProject(projectID)
	if err != nil {
		h.logFunc(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Error getting project snapshots: %v", err),
			Endpoint:  r.URL.Path,
		})
		h.WriteJSONError(w, r, fmt.Sprintf("Failed to get project snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	type SnapshotResponse struct {
		ID           int       `json:"id"`
		Name         string    `json:"name"`
		Description  string    `json:"description"`
		CreatedBy    string    `json:"created_by"`
		CreatedAt    time.Time `json:"created_at"`
		SnapshotType string    `json:"snapshot_type"`
		ProjectID    *int      `json:"project_id,omitempty"`
		ClientID     *int      `json:"client_id,omitempty"`
	}

	response := struct {
		Snapshots []SnapshotResponse `json:"snapshots"`
		Total     int                `json:"total"`
	}{
		Snapshots: make([]SnapshotResponse, 0, len(snapshots)),
		Total:     len(snapshots),
	}

	for _, snapshot := range snapshots {
		createdByStr := ""
		if snapshot.CreatedBy != nil {
			createdByStr = fmt.Sprintf("%d", *snapshot.CreatedBy)
		}
		response.Snapshots = append(response.Snapshots, SnapshotResponse{
			ID:           snapshot.ID,
			Name:         snapshot.Name,
			Description:  snapshot.Description,
			CreatedBy:    createdByStr,
			CreatedAt:    snapshot.CreatedAt,
			SnapshotType: snapshot.SnapshotType,
			ProjectID:    snapshot.ProjectID,
			ClientID:     snapshot.ClientID,
		})
	}

	h.WriteJSONResponse(w, r, response, http.StatusOK)
}

