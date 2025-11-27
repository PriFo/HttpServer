package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит Snapshots handlers, извлеченные из server.go
// для сокращения размера server.go

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/normalization"
)

func (s *Server) handleSnapshotsRoutes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListSnapshots(w, r)
	case http.MethodPost:
		s.handleCreateSnapshot(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleSnapshotRoutes обрабатывает запросы к /api/snapshots/{id} и вложенным маршрутам
func (s *Server) handleSnapshotRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/snapshots/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Snapshot ID required", http.StatusBadRequest)
		return
	}

	snapshotID, err := ValidateIDPathParam(parts[0], "snapshot_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid snapshot ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "normalize":
			if r.Method == http.MethodPost {
				s.handleNormalizeSnapshot(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "comparison":
			if r.Method == http.MethodGet {
				s.handleSnapshotComparison(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "metrics":
			if r.Method == http.MethodGet {
				s.handleSnapshotMetrics(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		case "evolution":
			if r.Method == http.MethodGet {
				s.handleSnapshotEvolution(w, r, snapshotID)
			} else {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			}
			return
		default:
			http.Error(w, "Unknown snapshot route", http.StatusNotFound)
			return
		}
	}

	// Обработка основных операций со срезом
	switch r.Method {
	case http.MethodGet:
		s.handleGetSnapshot(w, r, snapshotID)
	case http.MethodDelete:
		s.handleDeleteSnapshot(w, r, snapshotID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleProjectSnapshotsRoutes обрабатывает запросы к /api/projects/{project_id}/snapshots
func (s *Server) handleProjectSnapshotsRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	parts := strings.Split(path, "/")

	if len(parts) < 2 || parts[0] == "" || parts[1] != "snapshots" {
		// Это не маршрут для срезов проекта, передаем дальше
		http.NotFound(w, r)
		return
	}

	projectID, err := ValidateIDPathParam(parts[0], "project_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid project ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		s.handleGetProjectSnapshots(w, r, projectID)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleListSnapshots получает список всех срезов
func (s *Server) handleListSnapshots(w http.ResponseWriter, r *http.Request) {
	snapshots, err := s.db.GetAllSnapshots()
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	response := SnapshotListResponse{
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

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleCreateSnapshot создает новый срез вручную
func (s *Server) handleCreateSnapshot(w http.ResponseWriter, r *http.Request) {
	var req SnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Валидация
	if req.Name == "" {
		s.writeJSONError(w, r, "Name is required", http.StatusBadRequest)
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

	createdSnapshot, err := s.db.CreateSnapshot(snapshot, snapshotUploads)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to create snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	response := SnapshotResponse{
		ID:           createdSnapshot.ID,
		Name:         createdSnapshot.Name,
		Description:  createdSnapshot.Description,
		CreatedBy:    createdSnapshot.CreatedBy,
		CreatedAt:    createdSnapshot.CreatedAt,
		SnapshotType: createdSnapshot.SnapshotType,
		ProjectID:    createdSnapshot.ProjectID,
		ClientID:     createdSnapshot.ClientID,
		UploadCount:  len(snapshotUploads),
	}

	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Created snapshot: %s (ID: %d, uploads: %d)", createdSnapshot.Name, createdSnapshot.ID, len(snapshotUploads)),
		Endpoint:  "/api/snapshots",
	})

	s.writeJSONResponse(w, r, response, http.StatusCreated)
}

// handleCreateAutoSnapshot создает срез автоматически по критериям
func (s *Server) handleCreateAutoSnapshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AutoSnapshotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Type != "latest_per_database" {
		s.writeJSONError(w, r, "Unsupported auto snapshot type", http.StatusBadRequest)
		return
	}

	if req.UploadsPerDatabase <= 0 {
		req.UploadsPerDatabase = 3 // Значение по умолчанию
	}

	createdSnapshot, err := s.createAutoSnapshot(req.ProjectID, req.UploadsPerDatabase, req.Name, req.Description)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to create auto snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	response := SnapshotResponse{
		ID:           createdSnapshot.ID,
		Name:         createdSnapshot.Name,
		Description:  createdSnapshot.Description,
		CreatedBy:    createdSnapshot.CreatedBy,
		CreatedAt:    createdSnapshot.CreatedAt,
		SnapshotType: createdSnapshot.SnapshotType,
		ProjectID:    createdSnapshot.ProjectID,
		ClientID:     createdSnapshot.ClientID,
	}

	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Created auto snapshot: %s (ID: %d, project: %d)", createdSnapshot.Name, createdSnapshot.ID, req.ProjectID),
		Endpoint:  "/api/snapshots/auto",
	})

	s.writeJSONResponse(w, r, response, http.StatusCreated)
}

// createAutoSnapshot создает срез автоматически для проекта
func (s *Server) createAutoSnapshot(projectID int, uploadsPerDatabase int, name, description string) (*database.DataSnapshot, error) {
	if s.serviceDB == nil {
		return nil, fmt.Errorf("service database not available")
	}

	// Получаем все базы данных проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	if len(databases) == 0 {
		return nil, fmt.Errorf("no databases found for project %d", projectID)
	}

	var snapshotUploads []database.SnapshotUpload
	uploadOrder := 0

	// Для каждой базы получаем N последних выгрузок
	for _, db := range databases {
		uploads, err := s.db.GetLatestUploads(db.ID, uploadsPerDatabase)
		if err != nil {
			log.Printf("Failed to get latest uploads for database %d: %v", db.ID, err)
			continue
		}

		for _, upload := range uploads {
			snapshotUploads = append(snapshotUploads, database.SnapshotUpload{
				UploadID:       upload.ID,
				IterationLabel: upload.IterationLabel,
				UploadOrder:    uploadOrder,
			})
			uploadOrder++
		}
	}

	if len(snapshotUploads) == 0 {
		return nil, fmt.Errorf("no uploads found for project %d", projectID)
	}

	// Получаем информацию о проекте для имени среза
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	// Формируем имя среза
	if name == "" {
		name = fmt.Sprintf("Авто-срез проекта '%s' (%d выгрузок)", project.Name, len(snapshotUploads))
	}
	if description == "" {
		description = fmt.Sprintf("Автоматически созданный срез: последние %d выгрузок от каждой базы данных проекта", uploadsPerDatabase)
	}

	// Создаем срез
	snapshot := &database.DataSnapshot{
		Name:         name,
		Description:  description,
		SnapshotType: "auto_latest",
		ProjectID:    &projectID,
		ClientID:     &project.ClientID,
	}

	return s.db.CreateSnapshot(snapshot, snapshotUploads)
}

// handleGetSnapshot получает детали среза
func (s *Server) handleGetSnapshot(w http.ResponseWriter, r *http.Request, snapshotID int) {
	snapshot, uploads, err := s.db.GetSnapshotWithUploads(snapshotID)
	if err != nil {
		if err == sql.ErrNoRows {
			s.writeJSONError(w, r, "Snapshot not found", http.StatusNotFound)
		} else {
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get snapshot: %v", err), http.StatusInternalServerError)
		}
		return
	}

	// Преобразуем uploads в UploadListItem
	uploadList := make([]UploadListItem, 0, len(uploads))
	for _, upload := range uploads {
		uploadList = append(uploadList, UploadListItem{
			UploadUUID:     upload.UploadUUID,
			StartedAt:      upload.StartedAt,
			CompletedAt:    upload.CompletedAt,
			Status:         upload.Status,
			Version1C:      upload.Version1C,
			ConfigName:     upload.ConfigName,
			TotalConstants: upload.TotalConstants,
			TotalCatalogs:  upload.TotalCatalogs,
			TotalItems:     upload.TotalItems,
		})
	}

	response := SnapshotResponse{
		ID:           snapshot.ID,
		Name:         snapshot.Name,
		Description:  snapshot.Description,
		CreatedBy:    snapshot.CreatedBy,
		CreatedAt:    snapshot.CreatedAt,
		SnapshotType: snapshot.SnapshotType,
		ProjectID:    snapshot.ProjectID,
		ClientID:     snapshot.ClientID,
		Uploads:      uploadList,
		UploadCount:  len(uploadList),
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetProjectSnapshots получает все срезы проекта
func (s *Server) handleGetProjectSnapshots(w http.ResponseWriter, r *http.Request, projectID int) {
	snapshots, err := s.db.GetSnapshotsByProject(projectID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get project snapshots: %v", err), http.StatusInternalServerError)
		return
	}

	response := SnapshotListResponse{
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

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleDeleteSnapshot удаляет срез
func (s *Server) handleDeleteSnapshot(w http.ResponseWriter, r *http.Request, snapshotID int) {
	err := s.db.DeleteSnapshot(snapshotID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to delete snapshot: %v", err), http.StatusInternalServerError)
		return
	}

	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Deleted snapshot ID: %d", snapshotID),
		Endpoint:  "/api/snapshots",
	})

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Snapshot deleted successfully",
	}, http.StatusOK)
}

// handleNormalizeSnapshot запускает нормализацию среза
func (s *Server) handleNormalizeSnapshot(w http.ResponseWriter, r *http.Request, snapshotID int) {
	var req SnapshotNormalizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Если тело запроса пустое, используем значения по умолчанию
		req = SnapshotNormalizationRequest{
			UseAI:            false,
			MinConfidence:    0.7,
			RateLimitDelayMS: 100,
			MaxRetries:       3,
		}
	}

	result, err := s.normalizeSnapshot(snapshotID, req)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to normalize snapshot %d: %v", snapshotID, err),
			Endpoint:  "/api/snapshots/normalize",
		})
		s.writeJSONError(w, r, fmt.Sprintf("Normalization failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   fmt.Sprintf("Normalized snapshot %d: processed %d items, %d groups", snapshotID, result.TotalProcessed, result.TotalGroups),
		Endpoint:  "/api/snapshots/normalize",
	})

	s.writeJSONResponse(w, r, result, http.StatusOK)
}

// normalizeSnapshot выполняет сквозную нормализацию среза
func (s *Server) normalizeSnapshot(snapshotID int, req SnapshotNormalizationRequest) (*SnapshotNormalizationResult, error) {
	// Получаем срез со всеми выгрузками
	snapshot, uploads, err := s.db.GetSnapshotWithUploads(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot: %w", err)
	}

	if len(uploads) == 0 {
		return nil, fmt.Errorf("snapshot has no uploads")
	}

	// Создаем нормализатор срезов
	snapshotNormalizer := normalization.NewSnapshotNormalizer()

	// Выполняем нормализацию
	result, err := snapshotNormalizer.NormalizeSnapshot(s.db, snapshot, uploads)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize snapshot: %w", err)
	}

	// Сохраняем результаты нормализации для каждой выгрузки
	for uploadID, uploadResult := range result.UploadResults {
		if uploadResult.Error != "" {
			continue // Пропускаем выгрузки с ошибками
		}

		// Преобразуем NormalizedItem в map[string]interface{} для сохранения
		dataToSave := make([]map[string]interface{}, 0, len(uploadResult.NormalizedData))
		for _, item := range uploadResult.NormalizedData {
			dataToSave = append(dataToSave, map[string]interface{}{
				"source_reference":        item.SourceReference,
				"source_name":             item.SourceName,
				"code":                    item.Code,
				"normalized_name":         item.NormalizedName,
				"normalized_reference":    item.NormalizedReference,
				"category":                item.Category,
				"merged_count":            item.MergedCount,
				"source_database_id":      item.SourceDatabaseID,
				"source_iteration_number": item.SourceIterationNumber,
			})
		}

		// Сохраняем данные
		err = s.db.SaveSnapshotNormalizedDataItems(snapshotID, uploadID, dataToSave)
		if err != nil {
			s.log(LogEntry{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Message:   fmt.Sprintf("Failed to save normalized data for upload %d: %v", uploadID, err),
				Endpoint:  "/api/snapshots/normalize",
			})
			// Продолжаем обработку других выгрузок
			continue
		}
	}

	// Формируем ответ
	response := &SnapshotNormalizationResult{
		SnapshotID:      result.SnapshotID,
		MasterReference: result.MasterReference,
		UploadResults:   make(map[int]*UploadNormalizationResult),
		TotalProcessed:  result.TotalProcessed,
		TotalGroups:     result.TotalGroups,
		CompletedAt:     time.Now(),
	}

	// Преобразуем результаты
	for uploadID, uploadResult := range result.UploadResults {
		var changes *NormalizationChanges
		if uploadResult.Changes != nil {
			changes = &NormalizationChanges{
				Added:   uploadResult.Changes.Added,
				Updated: uploadResult.Changes.Updated,
				Deleted: uploadResult.Changes.Deleted,
			}
		}

		response.UploadResults[uploadID] = &UploadNormalizationResult{
			UploadID:       uploadResult.UploadID,
			ProcessedCount: uploadResult.ProcessedCount,
			GroupCount:     uploadResult.GroupCount,
			Error:          uploadResult.Error,
			Changes:        changes,
		}
	}

	return response, nil
}

// handleSnapshotComparison получает сравнение итераций
func (s *Server) handleSnapshotComparison(w http.ResponseWriter, r *http.Request, snapshotID int) {
	comparison, err := s.compareSnapshotIterations(snapshotID)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to compare snapshot iterations %d: %v", snapshotID, err),
			Endpoint:  "/api/snapshots/comparison",
		})
		s.writeJSONError(w, r, fmt.Sprintf("Comparison failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, comparison, http.StatusOK)
}

// handleSnapshotMetrics получает метрики улучшения данных
func (s *Server) handleSnapshotMetrics(w http.ResponseWriter, r *http.Request, snapshotID int) {
	metrics, err := s.calculateSnapshotMetrics(snapshotID)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to calculate snapshot metrics %d: %v", snapshotID, err),
			Endpoint:  "/api/snapshots/metrics",
		})
		s.writeJSONError(w, r, fmt.Sprintf("Metrics calculation failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, metrics, http.StatusOK)
}

// handleSnapshotEvolution получает эволюцию номенклатуры
func (s *Server) handleSnapshotEvolution(w http.ResponseWriter, r *http.Request, snapshotID int) {
	evolution, err := s.getSnapshotEvolution(snapshotID)
	if err != nil {
		s.log(LogEntry{
			Timestamp: time.Now(),
			Level:     "ERROR",
			Message:   fmt.Sprintf("Failed to get snapshot evolution %d: %v", snapshotID, err),
			Endpoint:  "/api/snapshots/evolution",
		})
		s.writeJSONError(w, r, fmt.Sprintf("Evolution data failed: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, evolution, http.StatusOK)
}
