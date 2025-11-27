package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/server/services"
	"httpserver/server/types"
)

// NomenclatureResult представляет результат номенклатуры из базы данных
type NomenclatureResult struct {
	ID              int
	Code            string
	Name            string
	NormalizedName  string
	Category        string
	QualityScore    float64
	SourceDatabase  string
	SourceType      string // "main" или "normalized"
	ProjectID       int
	ProjectName     string
	KpvedCode       string
	KpvedName       string
	AIConfidence    float64
	AIReasoning     string
	ProcessingLevel string
	MergedCount     int
	SourceReference string
	SourceName      string
}

// ClientHandler обработчик для работы с клиентами, проектами и базами данных
type ClientHandler struct {
	clientService   *services.ClientService
	databaseService *services.DatabaseService
	baseHandler     *BaseHandler
	// Функции для получения данных из баз uploads
	getNomenclatureFromNormalizedDB func(projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*NomenclatureResult, int, error)
	getNomenclatureFromMainDB       func(dbPath string, clientID int, projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*NomenclatureResult, int, error)
	getProjectDatabases             func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error)
	dbConnectionCache               interface {
		GetConnection(dbPath string) (*sql.DB, error)
		ReleaseConnection(dbPath string)
	}
	// Функции для работы с базами данных (чтобы избежать циклического импорта)
	// Переименованы, чтобы избежать конфликта с методами
	findMatchingProjectForDatabaseFunc func(serviceDB *database.ServiceDB, clientID int, filePath string) (*database.ClientProject, error)
	parseDatabaseFileInfoFunc          func(fileName string) database.DatabaseFilenameInfo
	// Опциональные handlers для вложенных маршрутов
	normalizationHandler *NormalizationHandler // Handler для маршрутов нормализации
}

// SetNormalizationHandler устанавливает normalizationHandler
func (h *ClientHandler) SetNormalizationHandler(handler *NormalizationHandler) {
	h.normalizationHandler = handler
}

// NewClientHandler создает новый обработчик для работы с клиентами
func NewClientHandler(
	clientService *services.ClientService,
	baseHandler *BaseHandler,
) *ClientHandler {
	return &ClientHandler{
		clientService: clientService,
		baseHandler:   baseHandler,
	}
}

// SetDatabaseService устанавливает databaseService для получения статистики из uploads
func (h *ClientHandler) SetDatabaseService(databaseService *services.DatabaseService) {
	h.databaseService = databaseService
}

// SetNomenclatureDataFunctions устанавливает функции для получения данных из баз uploads
func (h *ClientHandler) SetNomenclatureDataFunctions(
	getNomenclatureFromNormalizedDB func(projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*NomenclatureResult, int, error),
	getNomenclatureFromMainDB func(dbPath string, clientID int, projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*NomenclatureResult, int, error),
	getProjectDatabases func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error),
	dbConnectionCache interface {
		GetConnection(dbPath string) (*sql.DB, error)
		ReleaseConnection(dbPath string)
	},
) {
	h.getNomenclatureFromNormalizedDB = getNomenclatureFromNormalizedDB
	h.getNomenclatureFromMainDB = getNomenclatureFromMainDB
	h.getProjectDatabases = getProjectDatabases
	h.dbConnectionCache = dbConnectionCache
}

// HandleClients обрабатывает запросы к /api/clients
func (h *ClientHandler) HandleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetClients(w, r)
	case http.MethodPost:
		h.CreateClient(w, r)
	default:
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
	}
}

// GetClients возвращает список всех клиентов
func (h *ClientHandler) GetClients(w http.ResponseWriter, r *http.Request) {
	clients, err := h.clientService.GetAllClients(r.Context())
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить список клиентов", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, clients, http.StatusOK)
}

// CreateClient создает нового клиента
func (h *ClientHandler) CreateClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		LegalName    string `json:"legal_name"`
		Description  string `json:"description"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
		TaxID        string `json:"tax_id"`
		Country      string `json:"country"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.Name == "" {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("поле 'name' обязательно для заполнения", nil))
		return
	}

	client, err := h.clientService.CreateClient(r.Context(), req.Name, req.LegalName, req.Description, req.ContactEmail, req.ContactPhone, req.TaxID, req.Country)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось создать клиента", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, client, http.StatusCreated)
}

// HandleClientRoutes обрабатывает запросы к /api/clients/{id} и вложенным маршрутам
func (h *ClientHandler) HandleClientRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/clients/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		h.baseHandler.WriteJSONError(w, r, "Client ID required", http.StatusBadRequest)
		return
	}

	clientID, err := ValidateIntPathParam(parts[0], "client_id")
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("Invalid client ID", err))
		return
	}

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "statistics":
			if r.Method == http.MethodGet {
				h.GetClientStatistics(w, r, clientID)
				return
			}
			h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
			return
		case "nomenclature":
			if r.Method == http.MethodGet {
				h.GetClientNomenclature(w, r, clientID)
				return
			}
			h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
			return
		case "databases":
			if len(parts) == 2 {
				// GET /api/clients/{id}/databases
				if r.Method == http.MethodGet {
					h.GetClientDatabases(w, r, clientID)
					return
				}
				h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
				return
			}
			// Обработка /api/clients/{id}/databases/auto-link
		case "documents":
			// /api/clients/{id}/documents
			if len(parts) == 2 {
				switch r.Method {
				case http.MethodGet:
					h.HandleGetClientDocuments(w, r, clientID)
				case http.MethodPost:
					h.HandleUploadClientDocument(w, r, clientID)
				default:
					h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
				}
				return
			}
			// /api/clients/{id}/documents/{docId}
			if len(parts) == 3 {
				docID, err := ValidateIntPathParam(parts[2], "document_id")
				if err != nil {
					h.baseHandler.HandleHTTPError(w, r, NewValidationError("Invalid document ID", err))
					return
				}

				switch r.Method {
				case http.MethodGet:
					h.HandleDownloadClientDocument(w, r, clientID, docID)
				case http.MethodDelete:
					h.HandleDeleteClientDocument(w, r, clientID, docID)
				default:
					h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodDelete)
				}
				return
			}
			if len(parts) == 3 && parts[2] == "auto-link" {
				if r.Method == http.MethodPost {
					h.AutoLinkClientDatabases(w, r, clientID)
					return
				}
				h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
				return
			}
			// Обработка /api/clients/{id}/databases/update-metadata
			if len(parts) == 3 && parts[2] == "update-metadata" {
				if r.Method == http.MethodPost {
					h.UpdateAllDatabasesMetadata(w, r, clientID)
					return
				}
				h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
				return
			}
			// Обработка /api/clients/{id}/databases/{dbId}/link
			if len(parts) == 4 && parts[3] == "link" {
				databaseID, err := ValidateIntPathParam(parts[2], "database_id")
				if err != nil {
					h.baseHandler.HandleHTTPError(w, r, NewValidationError("Invalid database ID", err))
					return
				}
				if r.Method == http.MethodPut {
					h.LinkDatabaseToProject(w, r, clientID, databaseID)
					return
				}
				if r.Method == http.MethodDelete {
					h.UnlinkDatabaseFromProject(w, r, clientID, databaseID)
					return
				}
				h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPut, http.MethodDelete)
				return
			}
			// По умолчанию возвращаем список баз данных
			if r.Method == http.MethodGet {
				h.GetClientDatabases(w, r, clientID)
				return
			}
			h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
			return
		case "projects":
			if len(parts) == 2 {
				if r.Method == http.MethodGet {
					h.GetClientProjects(w, r, clientID)
				} else if r.Method == http.MethodPost {
					h.CreateClientProject(w, r, clientID)
				} else {
					h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
				}
				return
			}
			// Обработка /api/clients/{id}/projects/{projectId}...
			if len(parts) >= 3 {
				projectID, err := ValidateIntPathParam(parts[2], "project_id")
				if err != nil {
					h.baseHandler.HandleHTTPError(w, r, NewValidationError("Invalid project ID", err))
					return
				}

				if len(parts) == 3 {
					switch r.Method {
					case http.MethodGet:
						h.GetClientProject(w, r, clientID, projectID)
					case http.MethodPut:
						h.UpdateClientProject(w, r, clientID, projectID)
					case http.MethodDelete:
						h.DeleteClientProject(w, r, clientID, projectID)
					default:
						h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodDelete)
					}
					return
				}

				// Обработка вложенных маршрутов проектов
				if len(parts) == 4 && parts[3] == "databases" {
					if r.Method == http.MethodGet {
						h.GetProjectDatabases(w, r, clientID, projectID)
					} else if r.Method == http.MethodPost {
						h.CreateProjectDatabase(w, r, clientID, projectID)
					} else {
						h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
					}
					return
				}

				// Обработка /api/clients/{id}/projects/{projectId}/databases/{dbId}
				if len(parts) == 5 && parts[3] == "databases" {
					dbID, err := ValidateIntPathParam(parts[4], "database_id")
					if err != nil {
						h.baseHandler.HandleHTTPError(w, r, NewValidationError("Invalid database ID", err))
						return
					}

					switch r.Method {
					case http.MethodGet:
						h.GetProjectDatabase(w, r, clientID, projectID, dbID)
					case http.MethodPut:
						h.UpdateProjectDatabase(w, r, clientID, projectID, dbID)
					case http.MethodDelete:
						h.DeleteProjectDatabase(w, r, clientID, projectID, dbID)
					default:
						h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodDelete)
					}
					return
				}

				// Обработка вложенных маршрутов normalization
				if len(parts) >= 4 && parts[3] == "normalization" {
					if h.normalizationHandler != nil {
						// Добавляем clientID и projectID в контекст для обработчика
						ctx := context.WithValue(r.Context(), "clientId", clientID)
						ctx = context.WithValue(ctx, "projectId", projectID)
						r = r.WithContext(ctx)

						if len(parts) == 5 {
							switch parts[4] {
							case "start":
								if r.Method == http.MethodPost {
									h.normalizationHandler.HandleStartClientProjectNormalization(w, r)
									return
								}
								h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
								return
							case "status":
								if r.Method == http.MethodGet {
									h.normalizationHandler.HandleGetClientProjectNormalizationStatus(w, r)
									return
								}
								h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
								return
							}
						}
					}
					// Если handler не установлен, возвращаем ошибку
					h.baseHandler.WriteJSONError(w, r, "Normalization handler not available", http.StatusNotImplemented)
					return
				}

				// Обработка вложенных маршрутов benchmarks
				if len(parts) == 4 && parts[3] == "benchmarks" {
					if r.Method == http.MethodGet {
						h.GetProjectBenchmarks(w, r, clientID, projectID)
						return
					} else if r.Method == http.MethodPost {
						h.CreateProjectBenchmark(w, r, clientID, projectID)
						return
					}
					h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPost)
					return
				}

				// Обработка вложенных маршрутов nomenclature
				if len(parts) == 4 && parts[3] == "nomenclature" {
					if r.Method == http.MethodGet {
						// Используем существующий метод GetClientNomenclature с фильтрацией по projectID
						h.GetClientNomenclature(w, r, clientID)
						return
					}
					h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
					return
				}
			}
		}
		h.baseHandler.WriteJSONError(w, r, "Invalid route", http.StatusNotFound)
		return
	}

	// Обработка /api/clients/{id}
	switch r.Method {
	case http.MethodGet:
		h.GetClient(w, r, clientID)
	case http.MethodPut:
		h.UpdateClient(w, r, clientID)
	case http.MethodDelete:
		h.DeleteClient(w, r, clientID)
	default:
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet, http.MethodPut, http.MethodDelete)
	}
}

// GetClient получает клиента по ID
func (h *ClientHandler) GetClient(w http.ResponseWriter, r *http.Request, clientID int) {
	client, err := h.clientService.GetClient(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Клиент не найден", err))
		return
	}

	projects, err := h.clientService.GetClientProjects(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	// Подсчитываем статистику
	var totalBenchmarks int
	var activeSessions int
	var totalQualityScore float64
	var qualityCount int

	// Получаем serviceDB для подсчета статистики
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB != nil {
		conn := serviceDB.GetConnection()

		// Получаем project IDs
		projectIDs := make([]interface{}, len(projects))
		for i, project := range projects {
			projectIDs[i] = project.ID
		}

		// Подсчитываем benchmarks
		if len(projectIDs) > 0 {
			placeholders := strings.Repeat("?,", len(projectIDs)-1) + "?"
			query := fmt.Sprintf(`
				SELECT COUNT(*) as total_benchmarks,
				       AVG(CASE WHEN quality_score IS NOT NULL THEN quality_score ELSE NULL END) as avg_quality_score
				FROM client_benchmarks
				WHERE client_project_id IN (%s)
			`, placeholders)

			var avgQuality sql.NullFloat64
			err := conn.QueryRow(query, projectIDs...).Scan(&totalBenchmarks, &avgQuality)
			if err == nil {
				if avgQuality.Valid {
					totalQualityScore = avgQuality.Float64
					qualityCount = 1
				}
			}

			// Подсчитываем активные сессии
			query = fmt.Sprintf(`
				SELECT COUNT(*) 
				FROM normalization_sessions ns
				INNER JOIN project_databases pd ON ns.project_database_id = pd.id
				WHERE pd.client_project_id IN (%s) AND ns.status = 'running'
			`, placeholders)

			err = conn.QueryRow(query, projectIDs...).Scan(&activeSessions)
			if err != nil {
				activeSessions = 0
			}
		}
	}

	// Получаем документы клиента
	var clientDocuments []*database.ClientDocument
	if serviceDB != nil {
		docs, err := serviceDB.GetClientDocuments(clientID)
		if err == nil && docs != nil {
			clientDocuments = docs
		}
	}

	// Преобразуем документы в формат для ответа
	documentsResponse := make([]types.ClientDocument, 0, len(clientDocuments))
	for _, doc := range clientDocuments {
		if doc != nil {
			documentsResponse = append(documentsResponse, types.ClientDocument{
				ID:          doc.ID,
				ClientID:    doc.ClientID,
				FileName:    doc.FileName,
				FilePath:    doc.FilePath,
				FileType:    doc.FileType,
				FileSize:    doc.FileSize,
				Category:    doc.Category,
				Description: doc.Description,
				UploadedBy:  doc.UploadedBy,
				UploadedAt:  doc.UploadedAt,
			})
		}
	}

	// Формируем ответ
	response := types.ClientDetailResponse{
		Client: types.Client{
			ID:                   client.ID,
			Name:                 client.Name,
			LegalName:            client.LegalName,
			Description:          client.Description,
			ContactEmail:         client.ContactEmail,
			ContactPhone:         client.ContactPhone,
			TaxID:                client.TaxID,
			Country:              client.Country,
			Status:               client.Status,
			CreatedBy:            client.CreatedBy,
			Industry:             client.Industry,
			CompanySize:          client.CompanySize,
			LegalForm:            client.LegalForm,
			ContactPerson:        client.ContactPerson,
			ContactPosition:      client.ContactPosition,
			AlternatePhone:       client.AlternatePhone,
			Website:              client.Website,
			OGRN:                 client.OGRN,
			KPP:                  client.KPP,
			LegalAddress:         client.LegalAddress,
			PostalAddress:        client.PostalAddress,
			BankName:             client.BankName,
			BankAccount:          client.BankAccount,
			CorrespondentAccount: client.CorrespondentAccount,
			BIK:                  client.BIK,
			ContractNumber:       client.ContractNumber,
			ContractDate:         client.ContractDate,
			ContractTerms:        client.ContractTerms,
			ContractExpiresAt:    client.ContractExpiresAt,
			CreatedAt:            client.CreatedAt,
			UpdatedAt:            client.UpdatedAt,
		},
		Projects:  make([]types.ClientProject, len(projects)),
		Documents: documentsResponse,
		Statistics: types.ClientStatistics{
			TotalProjects:   len(projects),
			TotalBenchmarks: totalBenchmarks,
			ActiveSessions:  activeSessions,
			AvgQualityScore: func() float64 {
				if qualityCount > 0 {
					return totalQualityScore
				}
				return 0.0
			}(),
		},
	}

	for i, p := range projects {
		response.Projects[i] = types.ClientProject{
			ID:                 p.ID,
			ClientID:           p.ClientID,
			Name:               p.Name,
			ProjectType:        p.ProjectType,
			Description:        p.Description,
			SourceSystem:       p.SourceSystem,
			Status:             p.Status,
			TargetQualityScore: p.TargetQualityScore,
			CreatedAt:          p.CreatedAt,
			UpdatedAt:          p.UpdatedAt,
		}
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// UpdateClient обновляет клиента
func (h *ClientHandler) UpdateClient(w http.ResponseWriter, r *http.Request, clientID int) {
	var req database.Client

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	// Используем UpdateClientFields для поддержки всех новых полей
	client, err := h.clientService.UpdateClientFields(r.Context(), clientID, &req)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось обновить клиента", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, client, http.StatusOK)
}

// DeleteClient удаляет клиента
func (h *ClientHandler) DeleteClient(w http.ResponseWriter, r *http.Request, clientID int) {
	if err := h.clientService.DeleteClient(r.Context(), clientID); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось удалить клиента", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{"message": "Client deleted successfully"}, http.StatusOK)
}

// GetClientProjects возвращает проекты клиента
func (h *ClientHandler) GetClientProjects(w http.ResponseWriter, r *http.Request, clientID int) {
	projects, err := h.clientService.GetClientProjects(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, projects, http.StatusOK)
}

// CreateClientProject создает новый проект для клиента
func (h *ClientHandler) CreateClientProject(w http.ResponseWriter, r *http.Request, clientID int) {
	var req struct {
		Name               string  `json:"name"`
		ProjectType        string  `json:"project_type"`
		Description        string  `json:"description"`
		SourceSystem       string  `json:"source_system"`
		TargetQualityScore float64 `json:"target_quality_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.Name == "" {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("поле 'name' обязательно для заполнения", nil))
		return
	}

	if req.ProjectType == "" {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("поле 'project_type' обязательно для заполнения", nil))
		return
	}

	project, err := h.clientService.CreateClientProject(r.Context(), clientID, req.Name, req.ProjectType, req.Description, req.SourceSystem, req.TargetQualityScore)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось создать проект", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusCreated)
}

// GetClientProject возвращает проект клиента
func (h *ClientHandler) GetClientProject(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	project, err := h.clientService.GetClientProject(r.Context(), clientID, projectID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	// Проверяем, что проект принадлежит указанному клиенту
	if project.ClientID != clientID {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	// Получаем информацию о клиенте
	client, err := h.clientService.GetClient(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Клиент не найден", err))
		return
	}

	// Подсчитываем статистику
	stats := map[string]interface{}{
		"total_benchmarks":    0,
		"approved_benchmarks": 0,
		"avg_quality_score":   0.0,
	}

	serviceDB := h.clientService.GetServiceDB()
	if serviceDB != nil {
		conn := serviceDB.GetConnection()
		if conn != nil {
			// Статистика по эталонам
			query := `
				SELECT 
					COUNT(*) as total_benchmarks,
					SUM(CASE WHEN is_approved = 1 THEN 1 ELSE 0 END) as approved_benchmarks,
					AVG(CASE WHEN quality_score IS NOT NULL THEN quality_score ELSE NULL END) as avg_quality_score
				FROM client_benchmarks
				WHERE client_project_id = ?
			`
			var totalBenchmarks int
			var approvedBenchmarks sql.NullInt64
			var avgQualityScore sql.NullFloat64

			err = conn.QueryRow(query, projectID).Scan(&totalBenchmarks, &approvedBenchmarks, &avgQualityScore)
			if err == nil {
				stats["total_benchmarks"] = totalBenchmarks
				if approvedBenchmarks.Valid {
					stats["approved_benchmarks"] = int(approvedBenchmarks.Int64)
				}
				if avgQualityScore.Valid {
					stats["avg_quality_score"] = avgQualityScore.Float64
				}
			}
		}
	}

	// Формируем расширенный ответ
	response := map[string]interface{}{
		"project":     project,
		"client_name": client.Name,
		"statistics":  stats,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// UpdateClientProject обновляет проект клиента
func (h *ClientHandler) UpdateClientProject(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	var req struct {
		Name               string  `json:"name"`
		ProjectType        string  `json:"project_type"`
		Description        string  `json:"description"`
		SourceSystem       string  `json:"source_system"`
		TargetQualityScore float64 `json:"target_quality_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	// Получаем текущий проект для сохранения статуса
	currentProject, err := h.clientService.GetClientProject(r.Context(), clientID, projectID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}
	status := currentProject.Status
	if status == "" {
		status = "active"
	}
	project, err := h.clientService.UpdateClientProject(r.Context(), clientID, projectID, req.Name, req.ProjectType, req.Description, req.SourceSystem, status, req.TargetQualityScore)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось обновить проект", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, project, http.StatusOK)
}

// DeleteClientProject удаляет проект клиента
func (h *ClientHandler) DeleteClientProject(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	if err := h.clientService.DeleteClientProject(r.Context(), clientID, projectID); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось удалить проект", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{"message": "Project deleted successfully"}, http.StatusOK)
}

// GetClientDatabases возвращает базы данных клиента
// @Summary Получить базы данных клиента
// @Description Возвращает список всех баз данных, связанных с клиентом, включая непривязанные базы
// @Tags clients
// @Produce json
// @Param clientId path int true "ID клиента"
// @Success 200 {array} database.ProjectDatabase "Список баз данных"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Клиент не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/clients/{clientId}/databases [get]
func (h *ClientHandler) GetClientDatabases(w http.ResponseWriter, r *http.Request, clientID int) {
	databases, err := h.clientService.GetClientDatabases(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить базы данных клиента", err))
		return
	}

	// Также получаем непривязанные базы данных из БД
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB != nil {
		unlinkedDatabases, err := serviceDB.GetUnlinkedDatabases()
		if err == nil {
			// Добавляем непривязанные базы, которых еще нет в списке
			existingPaths := make(map[string]bool)
			existingIDs := make(map[int]bool)
			for _, db := range databases {
				if db.FilePath != "" {
					absPath, _ := filepath.Abs(db.FilePath)
					existingPaths[absPath] = true
					existingPaths[db.FilePath] = true
				}
				existingIDs[db.ID] = true
			}

			for _, unlinkedDB := range unlinkedDatabases {
				if !existingIDs[unlinkedDB.ID] {
					absPath, _ := filepath.Abs(unlinkedDB.FilePath)
					if !existingPaths[absPath] && !existingPaths[unlinkedDB.FilePath] {
						// Проверяем, что файл существует
						if _, err := os.Stat(unlinkedDB.FilePath); err == nil {
							databases = append(databases, unlinkedDB)
						}
					}
				}
			}
		}
	}

	// Получаем статистику для всех баз данных одним batch-запросом (оптимизация N+1)
	var statsMap map[int]map[string]interface{}
	if h.databaseService != nil && h.databaseService.GetDB() != nil && len(databases) > 0 {
		databaseIDs := make([]int, 0, len(databases))
		for _, db := range databases {
			databaseIDs = append(databaseIDs, db.ID)
		}

		var statsErr error
		statsMap, statsErr = h.databaseService.GetDB().GetUploadStatsByDatabaseIDs(databaseIDs)
		if statsErr != nil {
			// Игнорируем ошибки получения статистики - это не критично
			statsMap = make(map[int]map[string]interface{})
		}
	}

	// Получаем serviceDB для метаданных
	serviceDB = h.clientService.GetServiceDB()

	getIntStat := func(stats map[string]interface{}, keys ...string) int {
		if stats == nil {
			return 0
		}
		for _, key := range keys {
			if value, ok := stats[key]; ok {
				switch v := value.(type) {
				case int:
					return v
				case int64:
					return int(v)
				case float64:
					return int(v)
				}
			}
		}
		return 0
	}

	mergeStatsPreferingNonZero := func(base map[string]interface{}, overrides map[string]interface{}) map[string]interface{} {
		if overrides == nil {
			return base
		}
		if base == nil {
			base = make(map[string]interface{}, len(overrides))
		}
		for key, value := range overrides {
			switch v := value.(type) {
			case int:
				if v > 0 {
					base[key] = v
				}
			case int64:
				if v > 0 {
					base[key] = int(v)
				}
			case float64:
				if v > 0 {
					base[key] = v
				}
			case string:
				if v != "" {
					base[key] = v
				}
			default:
				if value != nil {
					base[key] = value
				}
			}
		}
		return base
	}

	fileStatsCache := make(map[string]map[string]interface{})

	// Форматируем данные с добавлением статистики и конфигурации 1С
	databasesWithStats := make([]map[string]interface{}, 0, len(databases))
	for _, db := range databases {
		dbInfo := map[string]interface{}{
			"id":                db.ID,
			"client_project_id": db.ClientProjectID,
			"project_id":        db.ClientProjectID,
			"name":              db.Name,
			"path":              db.FilePath,
			"size":              db.FileSize,
			"created_at":        db.CreatedAt.Format(time.RFC3339),
			"updated_at":        db.UpdatedAt.Format(time.RFC3339),
			"status":            "active",
		}

		if !db.IsActive {
			dbInfo["status"] = "inactive"
		}

		if db.LastUsedAt != nil {
			dbInfo["last_used_at"] = db.LastUsedAt.Format(time.RFC3339)
		}

		// Получаем размер файла, если возможно
		if db.FilePath != "" {
			if info, err := os.Stat(db.FilePath); err == nil {
				dbInfo["size"] = info.Size()
			}
		}

		// Получаем информацию о проекте
		if db.ClientProjectID > 0 && serviceDB != nil {
			project, err := serviceDB.GetClientProject(db.ClientProjectID)
			if err == nil && project != nil {
				dbInfo["project_name"] = project.Name
			}
		}

		// Получаем метаданные для конфигурации 1С
		if serviceDB != nil && db.FilePath != "" {
			metadata, err := serviceDB.GetDatabaseMetadata(db.FilePath)
			if err == nil && metadata != nil && metadata.MetadataJSON != "" {
				var metadataMap map[string]interface{}
				if err := json.Unmarshal([]byte(metadata.MetadataJSON), &metadataMap); err == nil {
					if configName, ok := metadataMap["config_name"].(string); ok && configName != "" {
						dbInfo["config_name"] = configName
					}
					if displayName, ok := metadataMap["display_name"].(string); ok && displayName != "" {
						dbInfo["display_name"] = displayName
					}
				}
			}
		}

		// Добавляем статистику из batch-запроса
		var dbStats map[string]interface{}
		if statsMap != nil {
			if stats, exists := statsMap[db.ID]; exists && stats != nil {
				dbStats = stats
			}
		}

		// Если статистика из service_db равна 0 или отсутствует, проверяем файл БД проекта
		if db.FilePath != "" && getIntStat(dbStats, "total_uploads", "uploads_count") == 0 {
			var fileStats map[string]interface{}
			var found bool

			if fileStats, found = fileStatsCache[db.FilePath]; !found {
				if stats, err := database.GetUploadStatsFromDatabaseFile(db.FilePath); err == nil {
					fileStats = stats
				} else {
					// Логируем только полезные ошибки (отсутствие файла не критично)
					if !errors.Is(err, os.ErrNotExist) {
						log.Printf("failed to read stats from database file %s: %v", db.FilePath, err)
					}
				}
				fileStatsCache[db.FilePath] = fileStats
			}

			if fileStats != nil {
				dbStats = mergeStatsPreferingNonZero(dbStats, fileStats)
			}
		}

		// Добавляем статистику в dbInfo
		if dbStats != nil {
			dbInfo["stats"] = dbStats
		}

		databasesWithStats = append(databasesWithStats, dbInfo)
	}

	h.baseHandler.WriteJSONResponse(w, r, databasesWithStats, http.StatusOK)
}

// LinkDatabaseToProject привязывает базу данных к проекту
// PUT /api/clients/{clientId}/databases/{databaseId}/link
func (h *ClientHandler) LinkDatabaseToProject(w http.ResponseWriter, r *http.Request, clientID, databaseID int) {
	var req struct {
		ProjectID int `json:"project_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("Invalid request body", err))
		return
	}

	if req.ProjectID == 0 {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("project_id is required", nil))
		return
	}

	// Проверяем, что проект принадлежит клиенту
	project, err := h.clientService.GetClientProject(r.Context(), clientID, req.ProjectID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проект", err))
		return
	}

	if project == nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Проект не найден", nil))
		return
	}

	// Получаем serviceDB
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	// Проверяем, что база данных существует и принадлежит клиенту
	db, err := serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		if err == sql.ErrNoRows {
			h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("База данных не найдена", err))
			return
		}
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить базу данных", err))
		return
	}

	// Проверяем, что база данных не привязана к другому проекту
	if db.ClientProjectID > 0 && db.ClientProjectID != req.ProjectID {
		existingProject, err := serviceDB.GetClientProject(db.ClientProjectID)
		projectName := "неизвестный проект"
		if err == nil && existingProject != nil {
			projectName = existingProject.Name
		}
		h.baseHandler.HandleHTTPError(w, r, NewValidationError(
			fmt.Sprintf("База данных уже привязана к проекту '%s'", projectName),
			nil))
		return
	}

	// Проверяем, что файл базы данных существует (если указан путь)
	if db.FilePath != "" {
		if _, err := os.Stat(db.FilePath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				h.baseHandler.HandleHTTPError(w, r, NewValidationError(
					fmt.Sprintf("Файл базы данных не найден: %s", db.FilePath),
					err))
				return
			}
			h.baseHandler.HandleHTTPError(w, r, NewValidationError(
				fmt.Sprintf("Ошибка проверки файла базы данных: %v", err),
				err))
			return
		}
	}

	// Привязываем базу данных к проекту
	if err := serviceDB.LinkProjectDatabaseToProject(databaseID, req.ProjectID); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось привязать базу данных к проекту", err))
		return
	}

	log.Printf("[LinkDatabaseToProject] База данных %d успешно привязана к проекту %d", databaseID, req.ProjectID)

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"message":     "База данных успешно привязана к проекту",
		"database_id": databaseID,
		"project_id":  req.ProjectID,
	}, http.StatusOK)
}

// UnlinkDatabaseFromProject отвязывает базу данных от проекта
// DELETE /api/clients/{clientId}/databases/{databaseId}/link
func (h *ClientHandler) UnlinkDatabaseFromProject(w http.ResponseWriter, r *http.Request, clientID, databaseID int) {
	// Получаем serviceDB
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	// Проверяем, что база данных существует
	db, err := serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("База данных не найдена", err))
		return
	}

	// Проверяем, что база данных принадлежит клиенту (через проекты)
	if db.ClientProjectID > 0 {
		project, err := serviceDB.GetClientProject(db.ClientProjectID)
		if err == nil && project.ClientID != clientID {
			h.baseHandler.HandleHTTPError(w, r, NewForbiddenError("База данных принадлежит другому клиенту", nil))
			return
		}
	}

	// Отвязываем базу данных от проекта
	if err := serviceDB.UnlinkProjectDatabase(databaseID); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось отвязать базу данных от проекта", err))
		return
	}

	log.Printf("[UnlinkDatabaseFromProject] База данных %d успешно отвязана от проекта", databaseID)

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"message":     "База данных успешно отвязана от проекта",
		"database_id": databaseID,
	}, http.StatusOK)
}

// AutoLinkClientDatabases автоматически привязывает непривязанные базы данных к проектам
// POST /api/clients/{clientId}/databases/auto-link
func (h *ClientHandler) AutoLinkClientDatabases(w http.ResponseWriter, r *http.Request, clientID int) {
	// Получаем serviceDB
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	linkedCount := 0
	errors := []string{}

	// Получаем все проекты клиента
	projects, err := h.clientService.GetClientProjects(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	// Получаем все базы данных клиента (включая непривязанные)
	databases, err := h.clientService.GetClientDatabases(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить базы данных клиента", err))
		return
	}

	// Также получаем непривязанные базы данных из БД, которые еще не в списке
	unlinkedDatabases, err := serviceDB.GetUnlinkedDatabases()
	if err != nil {
		log.Printf("[AutoLinkClientDatabases] Ошибка получения непривязанных баз из БД: %v", err)
	} else {
		log.Printf("[AutoLinkClientDatabases] Найдено непривязанных баз в БД: %d", len(unlinkedDatabases))
		// Добавляем непривязанные базы, которых еще нет в списке
		existingPaths := make(map[string]bool)
		for _, db := range databases {
			if db.FilePath != "" {
				absPath, _ := filepath.Abs(db.FilePath)
				existingPaths[absPath] = true
				existingPaths[db.FilePath] = true
			}
		}

		addedCount := 0
		for _, unlinkedDB := range unlinkedDatabases {
			if unlinkedDB.FilePath != "" {
				absPath, _ := filepath.Abs(unlinkedDB.FilePath)
				if !existingPaths[absPath] && !existingPaths[unlinkedDB.FilePath] {
					// Проверяем, что файл существует
					if _, err := os.Stat(unlinkedDB.FilePath); err == nil {
						databases = append(databases, unlinkedDB)
						addedCount++
					} else {
						log.Printf("[AutoLinkClientDatabases] Файл не существует: %s", unlinkedDB.FilePath)
					}
				}
			}
		}
		if addedCount > 0 {
			log.Printf("[AutoLinkClientDatabases] Добавлено непривязанных баз из БД: %d", addedCount)
		}
	}

	// Также ищем базы данных в папке uploads, которые еще не добавлены в project_databases
	uploadsDir := "data/uploads"
	if _, err := os.Stat(uploadsDir); err == nil {
		log.Printf("[AutoLinkClientDatabases] Сканируем папку uploads: %s", uploadsDir)
		// Сканируем папку uploads
		files, err := filepath.Glob(filepath.Join(uploadsDir, "*.db"))
		if err != nil {
			log.Printf("[AutoLinkClientDatabases] Ошибка сканирования папки uploads: %v", err)
		} else {
			log.Printf("[AutoLinkClientDatabases] Найдено файлов в uploads: %d", len(files))
			for _, filePath := range files {
				absPath, err := filepath.Abs(filePath)
				if err != nil {
					continue
				}

				// Проверяем, есть ли уже эта база в project_databases
				_, projectID, err := serviceDB.FindClientAndProjectByDatabasePath(absPath)
				if err == nil && projectID > 0 {
					// База уже привязана, пропускаем
					continue
				}

				// Используем унифицированную функцию для поиска подходящего проекта
				matchingProject, err := h.findMatchingProjectForDatabaseFunc(serviceDB, clientID, absPath)
				if err != nil {
					log.Printf("[AutoLinkClientDatabases] Ошибка поиска проекта для базы %s: %v", absPath, err)
					continue
				}

				if matchingProject != nil {
					// Проверяем, что файл существует
					_, err := os.Stat(absPath)
					if err != nil {
						errors = append(errors, fmt.Sprintf("Файл не существует: %s", absPath))
						log.Printf("[AutoLinkClientDatabases] Файл не существует: %s", absPath)
						continue
					}

					// Получаем читаемое название из имени файла
					fileName := filepath.Base(absPath)
					fileInfo := h.parseDatabaseFileInfoFunc(fileName)
					displayName := fileInfo.DisplayName
					if displayName == "" {
						displayName = fileName
					}

					log.Printf("[AutoLinkClientDatabases] Привязываем базу %s к проекту %d (%s)", fileName, matchingProject.ID, matchingProject.Name)
					projectDB, err := serviceDB.LinkDatabaseByPathToProject(absPath, matchingProject.ID, displayName)
					if err == nil && projectDB != nil {
						linkedCount++
						log.Printf("[AutoLinkClientDatabases] Успешно привязана база %s (ID: %d)", fileName, projectDB.ID)
					} else {
						errorMsg := fmt.Sprintf("Ошибка создания записи для базы %s: %v", fileName, err)
						errors = append(errors, errorMsg)
						log.Printf("[AutoLinkClientDatabases] %s", errorMsg)
					}
				} else {
					fileName := filepath.Base(absPath)
					fileInfo := h.parseDatabaseFileInfoFunc(fileName)
					errorMsg := fmt.Sprintf("Не найден подходящий проект для базы '%s' (тип данных: %s, конфигурация: %s). Убедитесь, что существует проект с типом '%s'",
						fileName,
						fileInfo.DataType,
						fileInfo.ConfigName,
						fileInfo.DataType)
					errors = append(errors, errorMsg)
					log.Printf("[AutoLinkClientDatabases] %s", errorMsg)
				}
			}
		}
	}

	// Для каждой базы данных пытаемся найти подходящий проект
	for _, db := range databases {
		// Если база уже привязана, пропускаем
		if db.ClientProjectID > 0 {
			continue
		}

		// Пытаемся найти подходящий проект автоматически
		var matchingProject *database.ClientProject

		// Сначала пробуем через метаданные
		metadata, err := serviceDB.GetDatabaseMetadata(db.FilePath)
		if err == nil && metadata != nil && metadata.MetadataJSON != "" {
			var metadataMap map[string]interface{}
			if err := json.Unmarshal([]byte(metadata.MetadataJSON), &metadataMap); err == nil {
				if dataType, ok := metadataMap["data_type"].(string); ok && dataType != "" {
					// Ищем проект с подходящим типом
					for _, project := range projects {
						if project.ProjectType == dataType ||
							(dataType == "nomenclature" && project.ProjectType == "nomenclature_counterparties") ||
							(dataType == "counterparties" && project.ProjectType == "nomenclature_counterparties") {
							matchingProject = project
							break
						}
					}
				}
			}
		}

		// Если не нашли через метаданные, пробуем парсить имя файла
		if matchingProject == nil {
			fileName := filepath.Base(db.FilePath)
			nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
			parts := strings.Split(nameWithoutExt, "_")

			if len(parts) >= 3 {
				dbType := parts[1] // Номенклатура или Контрагенты
				var dataType string
				if dbType == "Номенклатура" {
					dataType = "nomenclature"
				} else if dbType == "Контрагенты" {
					dataType = "counterparties"
				}

				if dataType != "" {
					// Ищем проект с подходящим типом
					for _, project := range projects {
						if project.ProjectType == dataType ||
							(dataType == "nomenclature" && project.ProjectType == "nomenclature_counterparties") ||
							(dataType == "counterparties" && project.ProjectType == "nomenclature_counterparties") {
							matchingProject = project
							break
						}
					}
				}
			}
		}

		if matchingProject != nil {
			// Проверяем, что файл существует перед привязкой
			if db.FilePath != "" {
				if _, err := os.Stat(db.FilePath); err != nil {
					errors = append(errors, fmt.Sprintf("Файл не существует: %s (ID: %d)", db.Name, db.ID))
					log.Printf("[AutoLinkClientDatabases] Файл не существует: %s", db.FilePath)
					continue
				}
			}

			// Привязываем базу данных к проекту
			log.Printf("[AutoLinkClientDatabases] Привязываем базу %s (ID: %d) к проекту %d (%s)", db.Name, db.ID, matchingProject.ID, matchingProject.Name)
			if err := serviceDB.LinkProjectDatabaseToProject(db.ID, matchingProject.ID); err != nil {
				errorMsg := fmt.Sprintf("Ошибка привязки базы '%s' (ID: %d) к проекту '%s' (ID: %d): %v",
					db.Name, db.ID, matchingProject.Name, matchingProject.ID, err)
				errors = append(errors, errorMsg)
				log.Printf("[AutoLinkClientDatabases] %s", errorMsg)
				continue
			}
			linkedCount++
			log.Printf("[AutoLinkClientDatabases] Успешно привязана база %s (ID: %d) к проекту %s (ID: %d)",
				db.Name, db.ID, matchingProject.Name, matchingProject.ID)
		} else {
			// Получаем информацию о типе базы данных для более информативного сообщения
			fileName := filepath.Base(db.FilePath)
			fileInfo := h.parseDatabaseFileInfo(fileName)
			errorMsg := fmt.Sprintf("Не найден подходящий проект для базы '%s' (ID: %d). Тип данных: %s, конфигурация: %s. Убедитесь, что существует проект с типом '%s'",
				db.Name,
				db.ID,
				fileInfo.DataType,
				fileInfo.ConfigName,
				fileInfo.DataType)
			errors = append(errors, errorMsg)
			log.Printf("[AutoLinkClientDatabases] %s", errorMsg)
		}
	}

	log.Printf("[AutoLinkClientDatabases] Автоматическая привязка завершена. Привязано: %d из %d баз, ошибок: %d", linkedCount, len(databases), len(errors))

	response := map[string]interface{}{
		"message":         fmt.Sprintf("Автоматическая привязка завершена. Привязано баз: %d", linkedCount),
		"linked_count":    linkedCount,
		"total_databases": len(databases),
		"unlinked_count":  len(databases) - linkedCount,
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["error_count"] = len(errors)
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// GetProjectDatabases возвращает базы данных проекта
func (h *ClientHandler) GetProjectDatabases(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	databases, err := h.clientService.GetProjectDatabases(r.Context(), clientID, projectID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить базы данных проекта", err))
		return
	}

	// Получаем статистику для всех баз данных одним batch-запросом (оптимизация N+1)
	var statsMap map[int]map[string]interface{}
	if h.databaseService != nil && h.databaseService.GetDB() != nil && len(databases) > 0 {
		databaseIDs := make([]int, 0, len(databases))
		for _, db := range databases {
			databaseIDs = append(databaseIDs, db.ID)
		}

		var statsErr error
		statsMap, statsErr = h.databaseService.GetDB().GetUploadStatsByDatabaseIDs(databaseIDs)
		if statsErr != nil {
			// Игнорируем ошибки получения статистики - это не критично
			statsMap = make(map[int]map[string]interface{})
		}
	}

	// Форматируем данные с добавлением статистики
	databasesWithStats := make([]map[string]interface{}, 0, len(databases))
	for _, db := range databases {
		dbInfo := map[string]interface{}{
			"id":                db.ID,
			"client_project_id": db.ClientProjectID,
			"name":              db.Name,
			"file_path":         db.FilePath,
			"description":       db.Description,
			"is_active":         db.IsActive,
			"file_size":         db.FileSize,
			"created_at":        db.CreatedAt.Format(time.RFC3339),
			"updated_at":        db.UpdatedAt.Format(time.RFC3339),
		}

		if db.LastUsedAt != nil {
			dbInfo["last_used_at"] = db.LastUsedAt.Format(time.RFC3339)
		}

		// Получаем размер файла, если возможно
		if db.FilePath != "" {
			if info, err := os.Stat(db.FilePath); err == nil {
				dbInfo["file_size"] = info.Size()
			}
		}

		// Получаем метаданные для конфигурации 1С
		serviceDB := h.clientService.GetServiceDB()
		if serviceDB != nil && db.FilePath != "" {
			metadata, err := serviceDB.GetDatabaseMetadata(db.FilePath)
			if err == nil && metadata != nil && metadata.MetadataJSON != "" {
				var metadataMap map[string]interface{}
				if err := json.Unmarshal([]byte(metadata.MetadataJSON), &metadataMap); err == nil {
					if configName, ok := metadataMap["config_name"].(string); ok && configName != "" {
						dbInfo["config_name"] = configName
					}
					if displayName, ok := metadataMap["display_name"].(string); ok && displayName != "" {
						dbInfo["display_name"] = displayName
					}
				}
			}
		}

		// Добавляем статистику из batch-запроса
		if statsMap != nil {
			if stats, exists := statsMap[db.ID]; exists && stats != nil {
				dbInfo["stats"] = stats
			}
		}

		databasesWithStats = append(databasesWithStats, dbInfo)
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"databases": databasesWithStats,
		"total":     len(databasesWithStats),
	}, http.StatusOK)
}

// GetProjectDatabase возвращает базу данных проекта
func (h *ClientHandler) GetProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	projectDB, err := h.clientService.GetProjectDatabase(r.Context(), clientID, projectID, dbID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("База данных не найдена", err))
		return
	}

	// Форматируем данные для ответа
	dbInfo := map[string]interface{}{
		"id":                projectDB.ID,
		"client_project_id": projectDB.ClientProjectID,
		"name":              projectDB.Name,
		"path":              projectDB.FilePath,
		"file_path":         projectDB.FilePath,
		"description":       projectDB.Description,
		"is_active":         projectDB.IsActive,
		"status":            "active",
		"file_size":         projectDB.FileSize,
		"size":              projectDB.FileSize,
		"created_at":        projectDB.CreatedAt.Format(time.RFC3339),
		"updated_at":        projectDB.UpdatedAt.Format(time.RFC3339),
	}

	if !projectDB.IsActive {
		dbInfo["status"] = "inactive"
	}

	if projectDB.LastUsedAt != nil {
		dbInfo["last_used_at"] = projectDB.LastUsedAt.Format(time.RFC3339)
	}

	// Получаем размер файла, если возможно
	if projectDB.FilePath != "" {
		if info, err := os.Stat(projectDB.FilePath); err == nil {
			dbInfo["file_size"] = info.Size()
			dbInfo["size"] = info.Size()
		}
	}

	// Получаем статистику из uploads, если доступна основная БД
	if h.databaseService != nil && h.databaseService.GetDB() != nil {
		stats, err := h.databaseService.GetDB().GetUploadStatsByDatabaseID(projectDB.ID)
		if err == nil && stats != nil {
			// Если статистика нулевая, пытаемся получить из файла БД проекта
			totalUploads, _ := stats["total_uploads"].(int)
			totalCatalogs, _ := stats["total_catalogs"].(int)
			totalItems, _ := stats["total_items"].(int)
			if totalUploads == 0 && totalCatalogs == 0 && totalItems == 0 && projectDB.FilePath != "" {
				fileStats, fileErr := database.GetUploadStatsFromDatabaseFile(projectDB.FilePath)
				if fileErr == nil && fileStats != nil {
					// Используем статистику из файла БД, если она больше нуля
					fileUploads, _ := fileStats["total_uploads"].(int)
					fileCatalogs, _ := fileStats["total_catalogs"].(int)
					fileItems, _ := fileStats["total_items"].(int)
					if fileUploads > 0 || fileCatalogs > 0 || fileItems > 0 {
						stats = fileStats
					}
				}
			}
			dbInfo["stats"] = stats
		}
		// Игнорируем ошибки получения статистики - это не критично
	}

	// Получаем список таблиц с количеством записей
	if projectDB.FilePath != "" {
		if tables, err := h.getDatabaseTables(projectDB.FilePath); err == nil && len(tables) > 0 {
			dbInfo["tables"] = tables
			// Подсчитываем общую статистику
			totalRows := 0
			for _, table := range tables {
				if rowCount, ok := table["row_count"].(int); ok {
					totalRows += rowCount
				}
			}
			dbInfo["statistics"] = map[string]interface{}{
				"total_tables": len(tables),
				"total_rows":   totalRows,
				"total_size":   dbInfo["size"],
			}
		}
		// Игнорируем ошибки получения таблиц - это не критично
	}

	h.baseHandler.WriteJSONResponse(w, r, dbInfo, http.StatusOK)
}

// getDatabaseTables получает список таблиц базы данных с количеством записей
func (h *ClientHandler) getDatabaseTables(dbPath string) ([]map[string]interface{}, error) {
	// Проверяем существование файла
	if _, err := os.Stat(dbPath); err != nil {
		return nil, fmt.Errorf("database file not found: %w", err)
	}

	// Открываем базу данных
	conn, err := sql.Open("sqlite3", dbPath+"?_timeout=2000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer conn.Close()

	// Получаем список таблиц
	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []map[string]interface{}
	var priorityTables []map[string]interface{} // Приоритетные таблицы (nomenclature_items, counterparties)
	var otherTables []map[string]interface{}     // Остальные таблицы

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Валидация имени таблицы
		if !isValidTableNameForQuery(tableName) {
			continue
		}

		// Получаем количество записей
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if err := conn.QueryRow(query).Scan(&count); err != nil {
			count = 0
		}

		tableInfo := map[string]interface{}{
			"name":      tableName,
			"row_count": count,
		}

		// Приоритетные таблицы идут первыми
		if tableName == "nomenclature_items" || tableName == "counterparties" || 
		   tableName == "catalog_items" {
			priorityTables = append(priorityTables, tableInfo)
		} else {
			otherTables = append(otherTables, tableInfo)
		}
	}

	// Объединяем: сначала приоритетные, потом остальные
	tables = append(tables, priorityTables...)
	tables = append(tables, otherTables...)

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating tables: %w", err)
	}

	return tables, nil
}

// isValidTableNameForQuery проверяет, что имя таблицы безопасно для использования в SQL запросах
func isValidTableNameForQuery(name string) bool {
	if len(name) == 0 || len(name) > 100 {
		return false
	}
	// Разрешаем только буквы, цифры и подчеркивания
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}
	// Проверяем, что имя не начинается с цифры (SQLite ограничение)
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		return false
	}
	// Проверяем зарезервированные слова SQLite (базовый список)
	reservedWords := []string{"SELECT", "FROM", "WHERE", "INSERT", "UPDATE", "DELETE", "CREATE", "DROP", "ALTER", "INDEX", "TABLE", "VIEW", "TRIGGER", "PRAGMA"}
	upperName := strings.ToUpper(name)
	for _, reserved := range reservedWords {
		if upperName == reserved {
			return false
		}
	}
	return true
}

// UpdateProjectDatabase обновляет базу данных проекта
func (h *ClientHandler) UpdateProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	var req struct {
		Name   string `json:"name"`
		DBPath string `json:"db_path"`
		DBType string `json:"db_type"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	// Получаем текущую базу данных для сохранения isActive
	_, err := h.clientService.GetProjectDatabase(r.Context(), clientID, projectID, dbID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("База данных не найдена", err))
		return
	}
	isActive := true // По умолчанию активна
	// UpdateProjectDatabase требует description, но в запросе его нет, используем пустую строку
	database, err := h.clientService.UpdateProjectDatabase(r.Context(), clientID, projectID, dbID, req.Name, req.DBPath, "", isActive)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось обновить базу данных", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, database, http.StatusOK)
}

// DeleteProjectDatabase удаляет базу данных проекта
func (h *ClientHandler) DeleteProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	if err := h.clientService.DeleteProjectDatabase(r.Context(), clientID, projectID, dbID); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось удалить базу данных", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{"message": "Database deleted successfully"}, http.StatusOK)
}

// GetClientStatistics получает расширенную статистику клиента
// @Summary Получить статистику клиента
// @Description Возвращает расширенную статистику по клиенту: проекты, базы данных, эталоны, номенклатуру, контрагенты и т.д.
// @Tags clients
// @Produce json
// @Param clientId path int true "ID клиента"
// @Success 200 {object} map[string]interface{} "Статистика клиента"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Клиент не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/clients/{clientId}/statistics [get]
func (h *ClientHandler) GetClientStatistics(w http.ResponseWriter, r *http.Request, clientID int) {
	projects, err := h.clientService.GetClientProjects(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	// Подсчитываем статистику
	var totalBenchmarks int
	var totalNomenclature int
	var totalCounterparties int
	var totalDatabases int
	var activeSessions int
	var totalQualityScore float64
	var qualityCount int
	var totalDuplicateGroups int
	var totalDuplicates int
	var totalMultiDatabaseCounterparties int

	// Получаем serviceDB для оптимизированных запросов
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		// Если serviceDB недоступен, возвращаем базовую статистику
		response := map[string]interface{}{
			"total_projects":       len(projects),
			"total_benchmarks":     0,
			"active_sessions":      0,
			"avg_quality_score":    0.0,
			"total_nomenclature":   0,
			"total_counterparties": 0,
			"total_databases":      0,
			"projects":             []map[string]interface{}{},
		}
		h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
		return
	}

	conn := serviceDB.GetConnection()

	// Оптимизация: получаем все project IDs для batch запросов
	projectIDs := make([]interface{}, len(projects))
	projectMap := make(map[int]*database.ClientProject)
	for i, project := range projects {
		projectIDs[i] = project.ID
		projectMap[project.ID] = project
	}

	// Получаем статистику по benchmarks одним запросом
	benchmarkStats := make(map[int]map[string]interface{})
	if len(projectIDs) > 0 {
		placeholders := strings.Repeat("?,", len(projectIDs)-1) + "?"
		query := fmt.Sprintf(`
			SELECT 
				client_project_id,
				COUNT(*) as total_benchmarks,
				SUM(CASE WHEN category = 'nomenclature' THEN 1 ELSE 0 END) as total_nomenclature,
				SUM(CASE WHEN category = 'counterparty' THEN 1 ELSE 0 END) as total_counterparties,
				AVG(CASE WHEN quality_score IS NOT NULL THEN quality_score ELSE NULL END) as avg_quality_score
			FROM client_benchmarks
			WHERE client_project_id IN (%s)
			GROUP BY client_project_id
		`, placeholders)

		rows, err := conn.Query(query, projectIDs...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var projectID int
				var totalBench, totalNom, totalCounter int
				var avgQuality sql.NullFloat64

				if err := rows.Scan(&projectID, &totalBench, &totalNom, &totalCounter, &avgQuality); err == nil {
					benchmarkStats[projectID] = map[string]interface{}{
						"total_benchmarks":     totalBench,
						"total_nomenclature":   totalNom,
						"total_counterparties": totalCounter,
						"avg_quality_score": func() float64 {
							if avgQuality.Valid {
								return avgQuality.Float64
							}
							return 0.0
						}(),
					}
					totalBenchmarks += totalBench
					totalNomenclature += totalNom
					totalCounterparties += totalCounter
					if avgQuality.Valid {
						totalQualityScore += avgQuality.Float64
						qualityCount++
					}
				}
			}
		}
	}

	// Получаем количество баз данных по проектам одним запросом
	dbCounts := make(map[int]int)
	if len(projectIDs) > 0 {
		placeholders := strings.Repeat("?,", len(projectIDs)-1) + "?"
		query := fmt.Sprintf(`
			SELECT client_project_id, COUNT(*) as db_count
			FROM project_databases
			WHERE client_project_id IN (%s)
			GROUP BY client_project_id
		`, placeholders)

		rows, err := conn.Query(query, projectIDs...)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var projectID, dbCount int
				if err := rows.Scan(&projectID, &dbCount); err == nil {
					dbCounts[projectID] = dbCount
					totalDatabases += dbCount
				}
			}
		}
	}

	// Получаем активные сессии одним запросом
	if len(projectIDs) > 0 {
		// Получаем все database IDs для проектов
		placeholders := strings.Repeat("?,", len(projectIDs)-1) + "?"
		query := fmt.Sprintf(`
			SELECT COUNT(*) 
			FROM normalization_sessions ns
			INNER JOIN project_databases pd ON ns.project_database_id = pd.id
			WHERE pd.client_project_id IN (%s) AND ns.status = 'running'
		`, placeholders)

		err = conn.QueryRow(query, projectIDs...).Scan(&activeSessions)
		if err != nil {
			activeSessions = 0
		}
	}

	// Получаем все базы данных клиента для подсчета непривязанных и конфигураций
	allDatabases, err := h.clientService.GetClientDatabases(r.Context(), clientID)

	// Также получаем непривязанные базы данных из БД
	if err == nil && serviceDB != nil {
		unlinkedDatabasesFromDB, err := serviceDB.GetUnlinkedDatabases()
		if err == nil {
			// Добавляем непривязанные базы, которых еще нет в списке
			existingPaths := make(map[string]bool)
			existingIDs := make(map[int]bool)
			for _, db := range allDatabases {
				if db.FilePath != "" {
					absPath, _ := filepath.Abs(db.FilePath)
					existingPaths[absPath] = true
					existingPaths[db.FilePath] = true
				}
				existingIDs[db.ID] = true
			}

			for _, unlinkedDB := range unlinkedDatabasesFromDB {
				if !existingIDs[unlinkedDB.ID] {
					absPath, _ := filepath.Abs(unlinkedDB.FilePath)
					if !existingPaths[absPath] && !existingPaths[unlinkedDB.FilePath] {
						// Проверяем, что файл существует
						if _, err := os.Stat(unlinkedDB.FilePath); err == nil {
							allDatabases = append(allDatabases, unlinkedDB)
						}
					}
				}
			}
		}
	}

	unlinkedDatabases := []map[string]interface{}{}
	unlinkedCount := 0
	if err == nil {
		for _, db := range allDatabases {
			// Если база не привязана к проекту
			if db.ClientProjectID == 0 {
				unlinkedCount++
				dbInfo := map[string]interface{}{
					"id":   db.ID,
					"name": db.Name,
					"path": db.FilePath,
					"size": db.FileSize,
				}

				// Получаем метаданные для конфигурации 1С
				metadata, err := serviceDB.GetDatabaseMetadata(db.FilePath)
				if err == nil && metadata != nil && metadata.MetadataJSON != "" {
					var metadataMap map[string]interface{}
					if err := json.Unmarshal([]byte(metadata.MetadataJSON), &metadataMap); err == nil {
						if configName, ok := metadataMap["config_name"].(string); ok && configName != "" {
							dbInfo["config_name"] = configName
						}
						if displayName, ok := metadataMap["display_name"].(string); ok && displayName != "" {
							dbInfo["display_name"] = displayName
						}
					}
				}

				unlinkedDatabases = append(unlinkedDatabases, dbInfo)
			}
		}
	}

	// Формируем статистику по проектам
	projectStats := make([]map[string]interface{}, 0, len(projects))
	for _, project := range projects {
		stats := benchmarkStats[project.ID]
		if stats == nil {
			stats = map[string]interface{}{
				"total_benchmarks":     0,
				"total_nomenclature":   0,
				"total_counterparties": 0,
				"avg_quality_score":    0.0,
			}
		}

		// Получаем базы данных проекта для конфигураций
		projectDBs, _ := serviceDB.GetProjectDatabases(project.ID, false)
		configs := make(map[string]int)
		for _, db := range projectDBs {
			metadata, err := serviceDB.GetDatabaseMetadata(db.FilePath)
			if err == nil && metadata != nil && metadata.MetadataJSON != "" {
				var metadataMap map[string]interface{}
				if err := json.Unmarshal([]byte(metadata.MetadataJSON), &metadataMap); err == nil {
					if configName, ok := metadataMap["config_name"].(string); ok && configName != "" && configName != "Unknown" {
						configs[configName]++
					}
				}
			}
		}

		projectStat := map[string]interface{}{
			"project_id":           project.ID,
			"project_name":         project.Name, // Используем project_name для соответствия frontend
			"project_type":         project.ProjectType,
			"status":               project.Status,
			"total_benchmarks":     stats["total_benchmarks"],
			"total_nomenclature":   stats["total_nomenclature"],
			"total_counterparties": stats["total_counterparties"],
			"total_databases":      dbCounts[project.ID],
			"avg_quality_score":    stats["avg_quality_score"],
			"last_updated":         project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}

		// Добавляем информацию о конфигурациях 1С
		if len(configs) > 0 {
			projectStat["configs"] = configs
		}

		// Получаем дополнительные метрики по контрагентам (дубликаты, связи с базами данных)
		if counterpartyStats, err := serviceDB.GetNormalizedCounterpartyStats(project.ID); err == nil && counterpartyStats != nil {
			toInt := func(value interface{}) int {
				switch v := value.(type) {
				case int:
					return v
				case int64:
					return int(v)
				case float64:
					return int(v)
				default:
					return 0
				}
			}

			if duplicateGroups, ok := counterpartyStats["duplicate_groups"]; ok {
				count := toInt(duplicateGroups)
				projectStat["duplicate_groups"] = count
				totalDuplicateGroups += count
			}
			if duplicatesCount, ok := counterpartyStats["duplicates_count"]; ok {
				count := toInt(duplicatesCount)
				projectStat["duplicates_count"] = count
				totalDuplicates += count
			}
			if multiDatabaseCount, ok := counterpartyStats["multi_database_count"]; ok {
				count := toInt(multiDatabaseCount)
				projectStat["multi_database_count"] = count
				totalMultiDatabaseCounterparties += count
			}
		}

		projectStats = append(projectStats, projectStat)
	}

	// Формируем ответ
	response := map[string]interface{}{
		"total_projects":   len(projects),
		"total_benchmarks": totalBenchmarks,
		"active_sessions":  activeSessions,
		"avg_quality_score": func() float64 {
			if qualityCount > 0 {
				return totalQualityScore / float64(qualityCount)
			}
			return 0.0
		}(),
		"total_nomenclature":       totalNomenclature,
		"total_counterparties":     totalCounterparties,
		"total_databases":          totalDatabases,
		"projects":                 projectStats,
		"unlinked_databases":       unlinkedDatabases,
		"unlinked_databases_count": unlinkedCount,
		"duplicate_summary": map[string]int{
			"total_groups":                  totalDuplicateGroups,
			"total_records":                 totalDuplicates,
			"multi_database_counterparties": totalMultiDatabaseCounterparties,
		},
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// mergeNomenclatureResults объединяет результаты из разных баз данных с дедупликацией
func mergeNomenclatureResults(allResults []*NomenclatureResult, limit, offset int) ([]*NomenclatureResult, int) {
	if len(allResults) == 0 {
		return []*NomenclatureResult{}, 0
	}

	// Дедупликация по коду и названию
	seen := make(map[string]*NomenclatureResult)
	deduplicated := make([]*NomenclatureResult, 0)

	for _, result := range allResults {
		key := result.Code + "|" + result.NormalizedName
		if key == "|" {
			// Если нет кода и названия, используем ID и источник
			key = fmt.Sprintf("%d|%s|%s", result.ID, result.SourceType, result.SourceDatabase)
		}

		if existing, exists := seen[key]; exists {
			// Если уже есть запись, приоритет нормализованной базе
			if result.SourceType == "normalized" && existing.SourceType == "main" {
				seen[key] = result
			}
		} else {
			seen[key] = result
			deduplicated = append(deduplicated, result)
		}
	}

	total := len(deduplicated)

	// Применяем пагинацию
	start := offset
	end := offset + limit
	if start > len(deduplicated) {
		return []*NomenclatureResult{}, total
	}
	if end > len(deduplicated) {
		end = len(deduplicated)
	}

	return deduplicated[start:end], total
}

// GetClientNomenclature получает номенклатуру клиента из всех баз данных uploads
// @Summary Получить номенклатуру клиента
// @Description Возвращает номенклатуру клиента из всех баз данных с поддержкой пагинации и фильтрации по проекту
// @Tags clients
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param page query int false "Номер страницы" default(1)
// @Param limit query int false "Количество записей на странице" default(20)
// @Param project_id query int false "Фильтр по ID проекта"
// @Param search query string false "Поисковый запрос"
// @Success 200 {object} map[string]interface{} "Номенклатура с пагинацией"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Клиент не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/clients/{clientId}/nomenclature [get]
func (h *ClientHandler) GetClientNomenclature(w http.ResponseWriter, r *http.Request, clientID int) {
	const maxRecordsPerDB = 100000

	// Получаем параметры пагинации
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	projectId := r.URL.Query().Get("project_id")
	search := r.URL.Query().Get("search")

	pageNum := 1
	limitNum := 20
	if page != "" {
		if p, err := ValidateIntParam(r, "page", 1, 1, 1000); err == nil && p > 0 {
			pageNum = p
		}
	}
	if limit != "" {
		if l, err := ValidateIntParam(r, "limit", 20, 1, 100); err == nil && l > 0 && l <= 100 {
			limitNum = l
		}
	}

	offset := (pageNum - 1) * limitNum

	// Получаем проекты клиента
	projects, err := h.clientService.GetClientProjects(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	if len(projects) == 0 {
		response := map[string]interface{}{
			"items":      []interface{}{},
			"total":      0,
			"page":       pageNum,
			"limit":      limitNum,
			"has_more":   false,
			"project_id": projectId,
		}
		h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
		return
	}

	// Фильтруем проекты, если указан project_id
	var filteredProjects []*database.ClientProject
	projectIDs := make([]int, 0)
	projectNames := make(map[int]string)

	if projectId != "" {
		projectIDNum, err := ValidateIntPathParam(projectId, "project_id")
		if err != nil {
			h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный project_id", err))
			return
		}
		// Проверяем, что проект принадлежит клиенту
		projectFound := false
		for _, p := range projects {
			if p.ID == projectIDNum {
				projectFound = true
				filteredProjects = []*database.ClientProject{p}
				projectIDs = []int{p.ID}
				projectNames[p.ID] = p.Name
				break
			}
		}
		if !projectFound {
			h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("проект не найден", nil))
			return
		}
	} else {
		filteredProjects = projects
		projectIDs = make([]int, len(projects))
		for i, p := range projects {
			projectIDs[i] = p.ID
			projectNames[p.ID] = p.Name
		}
	}

	// Собираем результаты из всех баз данных
	allResults := make([]*NomenclatureResult, 0)

	// 1. Получаем данные из нормализованной базы (если функции установлены)
	if h.getNomenclatureFromNormalizedDB != nil {
		normalizedResults, _, err := h.getNomenclatureFromNormalizedDB(projectIDs, projectNames, search, maxRecordsPerDB, 0)
		if err != nil {
			// Продолжаем работу, даже если нормализованная база недоступна
		} else {
			allResults = append(allResults, normalizedResults...)
		}
	}

	// 2. Получаем данные из всех баз данных проектов (если функции установлены)
	if h.getProjectDatabases != nil && h.getNomenclatureFromMainDB != nil && h.dbConnectionCache != nil {
		for _, project := range filteredProjects {
			projectDBs, err := h.getProjectDatabases(project.ID, false)
			if err != nil {
				// Продолжаем работу с другими проектами
				continue
			}

			for _, projectDB := range projectDBs {
				if !projectDB.IsActive {
					continue
				}

				// Проверяем, что файл существует
				if _, err := os.Stat(projectDB.FilePath); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						continue
					}
					// Другие ошибки тоже пропускаем
					continue
				}

				// Получаем данные из основной базы
				// Метод getNomenclatureFromMainDB сам проверяет наличие нужных таблиц
				// (catalog_items или nomenclature_items) и возвращает пустой результат, если их нет
				mainResults, _, err := h.getNomenclatureFromMainDB(projectDB.FilePath, clientID, []int{project.ID}, projectNames, search, maxRecordsPerDB, 0)
				if err != nil {
					// Продолжаем работу с другими базами
					continue
				}

				// Обновляем source_database для результатов
				for _, result := range mainResults {
					result.SourceDatabase = projectDB.FilePath
					result.ProjectName = project.Name
				}

				allResults = append(allResults, mainResults...)
			}
		}
	}

	// Если функции не установлены, получаем данные из client_benchmarks (fallback)
	if h.getNomenclatureFromNormalizedDB == nil {
		serviceDB := h.clientService.GetServiceDB()
		if serviceDB != nil {
			conn := serviceDB.GetConnection()
			projectIDsForQuery := make([]interface{}, len(projectIDs))
			for i, pid := range projectIDs {
				projectIDsForQuery[i] = pid
			}
			placeholders := strings.Repeat("?,", len(projectIDsForQuery)-1) + "?"

			query := fmt.Sprintf(`SELECT id, original_name, normalized_name, category, subcategory, quality_score, is_approved, created_at 
			                    FROM client_benchmarks 
			                    WHERE client_project_id IN (%s) AND category = 'nomenclature'
			                    ORDER BY created_at DESC 
			                    LIMIT ? OFFSET ?`, placeholders)
			args := make([]interface{}, len(projectIDsForQuery)+2)
			copy(args, projectIDsForQuery)
			args[len(projectIDsForQuery)] = limitNum
			args[len(projectIDsForQuery)+1] = offset

			rows, err := conn.Query(query, args...)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var id int
					var originalName, normalizedName, category, subcategory string
					var qualityScore sql.NullFloat64
					var isApproved bool
					var createdAt time.Time

					if err := rows.Scan(&id, &originalName, &normalizedName, &category, &subcategory, &qualityScore, &isApproved, &createdAt); err == nil {
						result := &NomenclatureResult{
							ID:             id,
							Code:           "",
							Name:           originalName,
							NormalizedName: normalizedName,
							Category:       category,
							SourceType:     "benchmark",
							ProjectID:      0, // Будет определено из projectIDs
						}
						if qualityScore.Valid {
							result.QualityScore = qualityScore.Float64
						}
						allResults = append(allResults, result)
					}
				}
			}
		}
	}

	// Объединяем и применяем пагинацию
	paginatedResults, total := mergeNomenclatureResults(allResults, limitNum, offset)

	// Преобразуем в формат ответа
	items := make([]map[string]interface{}, 0)
	for _, result := range paginatedResults {
		item := map[string]interface{}{
			"id":              result.ID,
			"code":            result.Code,
			"name":            result.Name,
			"normalized_name": result.NormalizedName,
			"category":        result.Category,
			"quality_score":   result.QualityScore,
			"status":          "active",
			"merged_count":    result.MergedCount,
			"source_database": result.SourceDatabase,
			"source_type":     result.SourceType,
			"project_id":      result.ProjectID,
			"project_name":    result.ProjectName,
		}

		if result.KpvedCode != "" {
			item["kpved_code"] = result.KpvedCode
		}
		if result.KpvedName != "" {
			item["kpved_name"] = result.KpvedName
		}
		if result.AIConfidence > 0 {
			item["ai_confidence"] = result.AIConfidence
		}
		if result.AIReasoning != "" {
			item["ai_reasoning"] = result.AIReasoning
		}
		if result.ProcessingLevel != "" {
			item["processing_level"] = result.ProcessingLevel
		}

		items = append(items, item)
	}

	response := map[string]interface{}{
		"items":      items,
		"total":      total,
		"page":       pageNum,
		"limit":      limitNum,
		"has_more":   (offset + limitNum) < total,
		"project_id": projectId,
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusOK)
}

// CreateProjectDatabase создает новую базу данных для проекта
func (h *ClientHandler) CreateProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	var req struct {
		Name        string `json:"name"`
		FilePath    string `json:"file_path"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.Name == "" {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("поле 'name' обязательно для заполнения", nil))
		return
	}

	// Если file_path не указан, используем пустую строку (файл может быть создан позже)
	filePath := req.FilePath
	fileSize := int64(0)

	// Если file_path указан, проверяем существование файла и получаем его размер
	if filePath != "" {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			h.baseHandler.HandleHTTPError(w, r, NewValidationError(fmt.Sprintf("Файл не найден: %s", filePath), err))
			return
		}
		fileSize = fileInfo.Size()
	}

	// Проверяем, не существует ли уже база данных с таким именем или путем для этого проекта
	existingDatabases, err := h.clientService.GetProjectDatabases(r.Context(), clientID, projectID)
	if err == nil {
		for _, existingDB := range existingDatabases {
			// Проверяем по имени или по пути к файлу
			if existingDB.Name == req.Name || (filePath != "" && existingDB.FilePath == filePath) {
				// Возвращаем существующую базу данных вместо создания дубликата
				h.baseHandler.WriteJSONResponse(w, r, existingDB, http.StatusOK)
				return
			}
		}
	}

	// Создаем базу данных через сервис
	database, err := h.clientService.CreateProjectDatabase(r.Context(), clientID, projectID, req.Name, filePath, req.Description, fileSize)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось создать базу данных", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, database, http.StatusCreated)
}

// GetProjectBenchmarks получает эталоны проекта
// @Summary Получить эталоны проекта
// @Description Возвращает список эталонов для указанного проекта с возможностью фильтрации по категории и статусу одобрения
// @Tags projects
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Param category query string false "Фильтр по категории"
// @Param approved_only query bool false "Только одобренные эталоны" default(false)
// @Success 200 {object} map[string]interface{} "Список эталонов"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Проект не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/clients/{clientId}/projects/{projectId}/benchmarks [get]
func (h *ClientHandler) GetProjectBenchmarks(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Получаем serviceDB
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	// Проверяем существование проекта
	project, err := serviceDB.GetClientProject(projectID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	// Проверяем, что проект принадлежит клиенту
	if project.ClientID != clientID {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	// Получаем параметры запроса
	category := r.URL.Query().Get("category")
	approvedOnly := r.URL.Query().Get("approved_only") == "true"

	log.Printf("[GetProjectBenchmarks] Getting benchmarks for project %d, client %d, category: %s, approvedOnly: %v",
		projectID, clientID, category, approvedOnly)

	// Получаем эталоны из БД
	benchmarks, err := serviceDB.GetClientBenchmarks(projectID, category, approvedOnly)
	if err != nil {
		log.Printf("[GetProjectBenchmarks] Error getting benchmarks: %v", err)
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить эталоны проекта", err))
		return
	}

	log.Printf("[GetProjectBenchmarks] Found %d benchmarks for project %d", len(benchmarks), projectID)

	// Формируем ответ
	responseBenchmarks := make([]map[string]interface{}, len(benchmarks))
	for i, b := range benchmarks {
		responseBenchmarks[i] = map[string]interface{}{
			"id":                b.ID,
			"client_project_id": b.ClientProjectID,
			"original_name":     b.OriginalName,
			"normalized_name":   b.NormalizedName,
			"category":          b.Category,
			"subcategory":       b.Subcategory,
			"attributes":        b.Attributes,
			"quality_score":     b.QualityScore,
			"is_approved":       b.IsApproved,
			"approved_by":       b.ApprovedBy,
			"approved_at": func() interface{} {
				if b.ApprovedAt != nil {
					return b.ApprovedAt.Format("2006-01-02T15:04:05Z07:00")
				}
				return nil
			}(),
			"source_database": b.SourceDatabase,
			"usage_count":     b.UsageCount,
			"created_at":      b.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			"updated_at":      b.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}

	log.Printf("[GetProjectBenchmarks] Returning %d benchmarks to client", len(responseBenchmarks))

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"benchmarks": responseBenchmarks,
		"total":      len(responseBenchmarks),
	}, http.StatusOK)
}

// CreateProjectBenchmark создает эталон для проекта
// @Summary Создать эталон для проекта
// @Description Создает новый эталон (benchmark) для указанного проекта клиента
// @Tags projects
// @Accept json
// @Produce json
// @Param clientId path int true "ID клиента"
// @Param projectId path int true "ID проекта"
// @Param payload body map[string]interface{} true "Данные эталона"
// @Success 201 {object} database.ClientBenchmark "Созданный эталон"
// @Failure 400 {object} ErrorResponse "Некорректный запрос"
// @Failure 404 {object} ErrorResponse "Проект не найден"
// @Failure 500 {object} ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/clients/{clientId}/projects/{projectId}/benchmarks [post]
func (h *ClientHandler) CreateProjectBenchmark(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Получаем serviceDB
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	// Проверяем существование проекта
	project, err := serviceDB.GetClientProject(projectID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	// Проверяем, что проект принадлежит клиенту
	if project.ClientID != clientID {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	// Парсим тело запроса
	var req struct {
		OriginalName   string  `json:"original_name"`
		NormalizedName string  `json:"normalized_name"`
		Category       string  `json:"category"`
		Subcategory    string  `json:"subcategory"`
		Attributes     string  `json:"attributes"`
		QualityScore   float64 `json:"quality_score"`
		SourceDatabase string  `json:"source_database"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	// Валидация обязательных полей
	if req.OriginalName == "" || req.NormalizedName == "" || req.Category == "" {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("поля 'original_name', 'normalized_name' и 'category' обязательны", nil))
		return
	}

	log.Printf("[CreateProjectBenchmark] Creating benchmark for project %d, original_name: %s, normalized_name: %s, category: %s",
		projectID, req.OriginalName, req.NormalizedName, req.Category)

	// Создаем эталон
	benchmark, err := serviceDB.CreateClientBenchmark(projectID, req.OriginalName, req.NormalizedName, req.Category, req.Subcategory, req.Attributes, req.SourceDatabase, req.QualityScore)
	if err != nil {
		log.Printf("[CreateProjectBenchmark] Error creating benchmark: %v", err)
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось создать эталон", err))
		return
	}

	log.Printf("[CreateProjectBenchmark] Benchmark created successfully with ID: %d", benchmark.ID)

	// Формируем ответ
	response := map[string]interface{}{
		"id":                benchmark.ID,
		"client_project_id": benchmark.ClientProjectID,
		"original_name":     benchmark.OriginalName,
		"normalized_name":   benchmark.NormalizedName,
		"category":          benchmark.Category,
		"subcategory":       benchmark.Subcategory,
		"attributes":        benchmark.Attributes,
		"quality_score":     benchmark.QualityScore,
		"is_approved":       benchmark.IsApproved,
		"source_database":   benchmark.SourceDatabase,
		"usage_count":       benchmark.UsageCount,
		"created_at":        benchmark.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		"updated_at":        benchmark.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	h.baseHandler.WriteJSONResponse(w, r, response, http.StatusCreated)
}

// databaseFileInfo содержит информацию, извлеченную из имени файла базы данных
type databaseFileInfo struct {
	DisplayName  string // Читаемое название (например, "ERP WE Номенклатура")
	ConfigName   string // Название конфигурации 1С (например, "ERPWE")
	DatabaseType string // Тип базы данных (Номенклатура или Контрагенты)
	DataType     string // Тип данных для project_type (nomenclature, counterparties)
}

// parseDatabaseFileInfo извлекает полную информацию из имени файла базы данных
func (h *ClientHandler) parseDatabaseFileInfo(fileName string) databaseFileInfo {
	info := databaseFileInfo{
		DisplayName:  fileName,
		ConfigName:   "",
		DatabaseType: "",
		DataType:     "",
	}

	// Убираем расширение
	nameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	// Разбиваем по подчеркиваниям
	parts := strings.Split(nameWithoutExt, "_")

	if len(parts) < 3 {
		// Если формат не соответствует ожидаемому, возвращаем имя файла без расширения
		info.DisplayName = nameWithoutExt
		return info
	}

	// Формат: Выгрузка_<Тип>_<Конфигурация>_...
	// Тип: Номенклатура или Контрагенты
	// Конфигурация: например, ERPWE, БухгалтерияДляКазахстана

	dbType := parts[1]     // Номенклатура или Контрагенты
	configName := parts[2] // Название конфигурации

	// Если конфигурация "Unknown", пробуем взять следующую часть
	if configName == "Unknown" && len(parts) > 3 {
		configName = parts[3]
	}

	// Сохраняем исходное название конфигурации
	info.ConfigName = configName
	info.DatabaseType = dbType

	// Определяем тип данных для project_type
	if dbType == "Номенклатура" {
		info.DataType = "nomenclature"
	} else if dbType == "Контрагенты" {
		info.DataType = "counterparties"
	}

	// Формируем читаемое название
	var result strings.Builder

	// Добавляем название конфигурации, разделяя заглавные буквы пробелами
	// Например, "ERPWE" -> "ERP WE", "БухгалтерияДляКазахстана" -> "БухгалтерияДляКазахстана"
	if configName != "Unknown" && configName != "" {
		// Для латинских букв: разделяем по заглавным
		if strings.ContainsAny(configName, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
			var formattedConfig strings.Builder
			for i, r := range configName {
				if i > 0 && r >= 'A' && r <= 'Z' {
					formattedConfig.WriteRune(' ')
				}
				formattedConfig.WriteRune(r)
			}
			result.WriteString(formattedConfig.String())
		} else {
			result.WriteString(configName)
		}
		result.WriteString(" ")
	}

	// Добавляем тип
	result.WriteString(dbType)

	info.DisplayName = strings.TrimSpace(result.String())

	return info
}

// findMatchingProjectForDatabase находит подходящий проект для базы данных на основе имени файла
func (h *ClientHandler) findMatchingProjectForDatabase(serviceDB *database.ServiceDB, clientID int, filePath string) (*database.ClientProject, error) {
	if serviceDB == nil {
		return nil, fmt.Errorf("serviceDB is nil")
	}

	fileName := filepath.Base(filePath)
	fileInfo := h.parseDatabaseFileInfo(fileName)

	// Если не удалось определить тип данных, возвращаем nil
	if fileInfo.DataType == "" {
		return nil, nil
	}

	// Получаем все проекты клиента
	projects, err := h.clientService.GetClientProjects(context.Background(), clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client projects: %w", err)
	}

	// Ищем проект с подходящим типом
	for _, project := range projects {
		// Точное совпадение типа
		if project.ProjectType == fileInfo.DataType {
			return project, nil
		}

		// Для nomenclature_counterparties принимаем и nomenclature, и counterparties
		if project.ProjectType == "nomenclature_counterparties" {
			if fileInfo.DataType == "nomenclature" || fileInfo.DataType == "counterparties" {
				return project, nil
			}
		}
	}

	return nil, nil
}

// UpdateAllDatabasesMetadata обновляет метаданные для всех баз данных клиента
func (h *ClientHandler) UpdateAllDatabasesMetadata(w http.ResponseWriter, r *http.Request, clientID int) {
	// Получаем serviceDB
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.WriteJSONError(w, r, "сервисная база данных недоступна", http.StatusInternalServerError)
		return
	}

	// Получаем все проекты клиента
	projects, err := h.clientService.GetClientProjects(r.Context(), clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	updatedCount := 0
	errorCount := 0
	errors := make([]string, 0)

	// Для каждого проекта получаем базы данных и обновляем метаданные
	for _, project := range projects {
		databases, err := serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			errorCount++
			errors = append(errors, fmt.Sprintf("проект %d: не удалось получить базы данных: %v", project.ID, err))
			continue
		}

		for _, db := range databases {
			if db.FilePath == "" {
				continue
			}

			// Определяем тип базы данных
			dbType := "1C"
			if db.Description != "" {
				dbType = db.Description
			}

			// Обновляем метаданные
			fileName := filepath.Base(db.FilePath)
			fileInfo := database.ParseDatabaseFileInfo(fileName)

			// Получаем существующие метаданные
			existingMetadata, err := serviceDB.GetDatabaseMetadata(db.FilePath)
			if err != nil {
				errorCount++
				errors = append(errors, fmt.Sprintf("база данных %d: не удалось получить метаданные: %v", db.ID, err))
				continue
			}

			// Создаем структуру для метаданных
			metadataMap := make(map[string]interface{})
			if existingMetadata != nil && existingMetadata.MetadataJSON != "" {
				// Парсим существующие метаданные
				if err := json.Unmarshal([]byte(existingMetadata.MetadataJSON), &metadataMap); err != nil {
					// Если не удалось распарсить, начинаем с пустой карты
					metadataMap = make(map[string]interface{})
				}
			}

			// Обновляем информацию о конфигурации 1С
			metadataMap["config_name"] = fileInfo.ConfigName
			metadataMap["database_type"] = fileInfo.DatabaseType
			metadataMap["data_type"] = fileInfo.DataType
			metadataMap["display_name"] = fileInfo.DisplayName

			// Сериализуем обратно в JSON
			metadataJSON, err := json.Marshal(metadataMap)
			if err != nil {
				errorCount++
				errors = append(errors, fmt.Sprintf("база данных %d: не удалось сериализовать метаданные: %v", db.ID, err))
				continue
			}

			// Формируем описание
			description := fmt.Sprintf("База данных типа %s", dbType)
			if fileInfo.ConfigName != "" && fileInfo.ConfigName != "Unknown" {
				description = fmt.Sprintf("%s, конфигурация: %s", description, fileInfo.DisplayName)
			}

			// Обновляем метаданные
			if err := serviceDB.UpsertDatabaseMetadata(db.FilePath, dbType, description, string(metadataJSON)); err != nil {
				errorCount++
				errors = append(errors, fmt.Sprintf("база данных %d: не удалось обновить метаданные: %v", db.ID, err))
				continue
			}

			updatedCount++
		}
	}

	// Формируем ответ
	response := map[string]interface{}{
		"client_id":      clientID,
		"updated_count":  updatedCount,
		"error_count":    errorCount,
		"total_projects": len(projects),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	statusCode := http.StatusOK
	if errorCount > 0 && updatedCount == 0 {
		statusCode = http.StatusInternalServerError
	} else if errorCount > 0 {
		statusCode = http.StatusPartialContent
	}

	h.baseHandler.WriteJSONResponse(w, r, response, statusCode)
}

// HandleUploadClientDocument обрабатывает загрузку документа клиента
// POST /api/clients/{clientId}/documents
func (h *ClientHandler) HandleUploadClientDocument(w http.ResponseWriter, r *http.Request, clientID int) {
	if r.Method != http.MethodPost {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodPost)
		return
	}

	// Проверяем существование клиента
	ctx := r.Context()
	_, err := h.clientService.GetClient(ctx, clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Клиент не найден", err))
		return
	}

	// Парсим multipart form
	err = r.ParseMultipartForm(100 << 20) // 100 MB
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("Failed to parse multipart form", err))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewValidationError("File is required", err))
		return
	}
	defer file.Close()

	category := r.FormValue("category")
	if category == "" {
		category = "technical"
	}

	description := r.FormValue("description")
	uploadedBy := r.FormValue("uploaded_by")
	if uploadedBy == "" {
		uploadedBy = "system"
	}

	// Создаем директорию для документов клиента
	documentsDir := filepath.Join("data", "client_documents", fmt.Sprintf("%d", clientID))
	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to create documents directory", err))
		return
	}

	// Генерируем безопасное имя файла
	timestamp := time.Now().Format("20060102_150405")
	safeName := strings.ReplaceAll(header.Filename, " ", "_")
	fileName := fmt.Sprintf("%s_%s", timestamp, safeName)
	filePath := filepath.Join(documentsDir, fileName)

	// Сохраняем файл
	dst, err := os.Create(filePath)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to create file", err))
		return
	}
	defer dst.Close()

	written, err := io.Copy(dst, file)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to save file", err))
		return
	}

	// Определяем тип файла
	fileType := header.Header.Get("Content-Type")
	if fileType == "" {
		fileType = "application/octet-stream"
	}

	// Сохраняем информацию в БД
	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	document, err := serviceDB.UploadClientDocument(clientID, header.Filename, filePath, fileType, written, category, description, uploadedBy)
	if err != nil {
		// Удаляем файл при ошибке сохранения в БД
		os.Remove(filePath)
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to save document info", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, document, http.StatusCreated)
}

// HandleGetClientDocuments обрабатывает получение списка документов клиента
// GET /api/clients/{clientId}/documents
func (h *ClientHandler) HandleGetClientDocuments(w http.ResponseWriter, r *http.Request, clientID int) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	documents, err := serviceDB.GetClientDocuments(clientID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to get documents", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"documents": documents,
		"total":     len(documents),
	}, http.StatusOK)
}

// HandleDownloadClientDocument обрабатывает скачивание документа клиента
// GET /api/clients/{clientId}/documents/{docId}
func (h *ClientHandler) HandleDownloadClientDocument(w http.ResponseWriter, r *http.Request, clientID, docID int) {
	if r.Method != http.MethodGet {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodGet)
		return
	}

	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	document, err := serviceDB.GetClientDocument(docID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to get document", err))
		return
	}

	if document == nil || document.ClientID != clientID {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Document not found", nil))
		return
	}

	// Открываем файл
	file, err := os.Open(document.FilePath)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("File not found", err))
		return
	}
	defer file.Close()

	// Устанавливаем заголовки
	w.Header().Set("Content-Type", document.FileType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, document.FileName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", document.FileSize))

	// Отправляем файл
	http.ServeContent(w, r, document.FileName, document.UploadedAt, file)
}

// HandleDeleteClientDocument обрабатывает удаление документа клиента
// DELETE /api/clients/{clientId}/documents/{docId}
func (h *ClientHandler) HandleDeleteClientDocument(w http.ResponseWriter, r *http.Request, clientID, docID int) {
	if r.Method != http.MethodDelete {
		h.baseHandler.HandleMethodNotAllowed(w, r, http.MethodDelete)
		return
	}

	serviceDB := h.clientService.GetServiceDB()
	if serviceDB == nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	document, err := serviceDB.GetClientDocument(docID)
	if err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to get document", err))
		return
	}

	if document == nil || document.ClientID != clientID {
		h.baseHandler.HandleHTTPError(w, r, NewNotFoundError("Document not found", nil))
		return
	}

	// Удаляем файл
	if err := os.Remove(document.FilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("Failed to delete file %s: %v", document.FilePath, err)
	}

	// Удаляем запись из БД
	if err := serviceDB.DeleteClientDocument(docID); err != nil {
		h.baseHandler.HandleHTTPError(w, r, NewInternalError("Failed to delete document", err))
		return
	}

	h.baseHandler.WriteJSONResponse(w, r, map[string]interface{}{
		"success": true,
		"message": "Document deleted successfully",
	}, http.StatusOK)
}
