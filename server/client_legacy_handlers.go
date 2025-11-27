package server

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
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"httpserver/database"
	"httpserver/normalization"
	"httpserver/server/services"

	"github.com/google/uuid"
)

// Legacy client handlers - перемещены из server.go для рефакторинга
// TODO: Заменить на новые handlers из internal/api/handlers/

// handleClients обрабатывает запросы к /api/clients
func (s *Server) handleClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleGetClients(w, r)
	case http.MethodPost:
		s.handleCreateClient(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleClientRoutes обрабатывает запросы к /api/clients/{id} и вложенным маршрутам
func (s *Server) handleClientRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/clients/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, "Client ID required", http.StatusBadRequest)
		return
	}

	clientID, err := ValidateIntPathParam(parts[0], "client_id")
	if err != nil {
		s.HandleValidationError(w, r, err)
		return
	}

	// Обработка вложенных маршрутов
	if len(parts) > 1 {
		switch parts[1] {
		case "statistics":
			// GET /api/clients/{id}/statistics
			if r.Method == http.MethodGet {
				s.handleGetClientStatistics(w, r, clientID)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		case "nomenclature":
			// GET /api/clients/{id}/nomenclature
			if r.Method == http.MethodGet {
				s.handleGetClientNomenclature(w, r, clientID)
				return
			}
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		case "databases":
			// GET /api/clients/{id}/databases
			if len(parts) == 2 {
				if r.Method == http.MethodGet {
					s.handleGetClientDatabases(w, r, clientID)
					return
				}
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			// POST /api/clients/{id}/databases/auto-link
			if len(parts) == 3 && parts[2] == "auto-link" {
				if r.Method == http.MethodPost {
					s.handleAutoLinkClientDatabases(w, r, clientID)
					return
				}
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			// PUT /api/clients/{id}/databases/{databaseId}/link
			if len(parts) == 4 && parts[3] == "link" {
				dbID, err := ValidateIntPathParam(parts[2], "database_id")
				if err != nil {
					s.HandleValidationError(w, r, err)
					return
				}
				if r.Method == http.MethodPut {
					s.handleLinkClientDatabase(w, r, clientID, dbID)
					return
				}
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
			return
		case "documents":
			// /api/clients/{id}/documents
			if len(parts) == 2 {
				if s.clientHandler != nil {
					switch r.Method {
					case http.MethodGet:
						s.clientHandler.HandleGetClientDocuments(w, r, clientID)
					case http.MethodPost:
						s.clientHandler.HandleUploadClientDocument(w, r, clientID)
					default:
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				} else {
					http.Error(w, "Client handler not available", http.StatusInternalServerError)
				}
				return
			}
			// /api/clients/{id}/documents/{docId}
			if len(parts) == 3 {
				docID, err := ValidateIntPathParam(parts[2], "document_id")
				if err != nil {
					s.HandleValidationError(w, r, err)
					return
				}

				if s.clientHandler != nil {
					switch r.Method {
					case http.MethodGet:
						s.clientHandler.HandleDownloadClientDocument(w, r, clientID, docID)
					case http.MethodDelete:
						s.clientHandler.HandleDeleteClientDocument(w, r, clientID, docID)
					default:
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
				} else {
					http.Error(w, "Client handler not available", http.StatusInternalServerError)
				}
				return
			}
			http.Error(w, "Not found", http.StatusNotFound)
			return
		case "projects":
			if len(parts) == 2 {
				// GET/POST /api/clients/{id}/projects
				if r.Method == http.MethodGet {
					s.handleGetClientProjects(w, r, clientID)
				} else if r.Method == http.MethodPost {
					s.handleCreateClientProject(w, r, clientID)
				} else {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				}
				return
			}
			// Обработка /api/clients/{id}/projects/{projectId}...
			if len(parts) >= 3 {
				projectID, err := ValidateIntPathParam(parts[2], "project_id")
				if err != nil {
					s.HandleValidationError(w, r, err)
					return
				}

				if len(parts) == 3 {
					// GET/PUT/DELETE /api/clients/{id}/projects/{projectId}
					switch r.Method {
					case http.MethodGet:
						s.handleGetClientProject(w, r, clientID, projectID)
					case http.MethodPut:
						s.handleUpdateClientProject(w, r, clientID, projectID)
					case http.MethodDelete:
						s.handleDeleteClientProject(w, r, clientID, projectID)
					default:
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
					return
				}

				if len(parts) == 4 && parts[3] == "benchmarks" {
					// GET/POST /api/clients/{id}/projects/{projectId}/benchmarks
					if r.Method == http.MethodGet {
						s.handleGetClientBenchmarks(w, r, clientID, projectID)
					} else if r.Method == http.MethodPost {
						s.handleCreateClientBenchmark(w, r, clientID, projectID)
					} else {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
					return
				}

				if len(parts) == 4 && parts[3] == "nomenclature" {
					// GET /api/clients/{id}/projects/{projectId}/nomenclature
					if r.Method == http.MethodGet {
						s.handleGetProjectNomenclature(w, r, clientID, projectID)
					} else {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
					return
				}

				// Обработка /api/clients/{id}/projects/{projectId}/databases
				if parts[3] == "databases" {
					if len(parts) == 4 {
						// GET/POST /api/clients/{id}/projects/{projectId}/databases
						if r.Method == http.MethodGet {
							s.handleGetProjectDatabases(w, r, clientID, projectID)
						} else if r.Method == http.MethodPost {
							// Проверяем Content-Type для определения типа запроса
							contentType := r.Header.Get("Content-Type")
							if strings.HasPrefix(contentType, "multipart/form-data") {
								// Загрузка файла
								s.handleUploadProjectDatabase(w, r, clientID, projectID)
							} else {
								// Обычный JSON запрос
								s.handleCreateProjectDatabase(w, r, clientID, projectID)
							}
						} else {
							http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						}
						return
					}

					if len(parts) == 5 {
						// GET/PUT/DELETE /api/clients/{id}/projects/{projectId}/databases/{dbId}
						dbID, err := ValidateIntPathParam(parts[4], "database_id")
						if err != nil {
							s.HandleValidationError(w, r, err)
							return
						}

						switch r.Method {
						case http.MethodGet:
							s.handleGetProjectDatabase(w, r, clientID, projectID, dbID)
						case http.MethodPut:
							s.handleUpdateProjectDatabase(w, r, clientID, projectID, dbID)
						case http.MethodDelete:
							s.handleDeleteProjectDatabase(w, r, clientID, projectID, dbID)
						default:
							http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						}
						return
					}

					// Обработка /api/clients/{id}/projects/{projectId}/databases/{dbId}/tables
					if len(parts) >= 6 && parts[5] == "tables" {
						dbID, err := ValidateIntPathParam(parts[4], "database_id")
						if err != nil {
							s.HandleValidationError(w, r, err)
							return
						}

						if len(parts) == 6 {
							// GET /api/clients/{id}/projects/{projectId}/databases/{dbId}/tables
							if r.Method == http.MethodGet {
								s.handleGetProjectDatabaseTables(w, r, clientID, projectID, dbID)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						}

						if len(parts) == 7 {
							// GET /api/clients/{id}/projects/{projectId}/databases/{dbId}/tables/{tableName}
							tableName := parts[6]
							if r.Method == http.MethodGet {
								s.handleGetProjectDatabaseTableData(w, r, clientID, projectID, dbID, tableName)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						}
					}
				}

				if len(parts) >= 4 && parts[3] == "normalization" {
					// Обработка /api/clients/{id}/projects/{projectId}/normalization/...
					if len(parts) == 5 {
						switch parts[4] {
						case "start":
							if r.Method == http.MethodPost {
								// Используем новый обработчик из NormalizationHandler
								if s.normalizationHandler != nil {
									// Добавляем clientID и projectID в контекст для обработчика
									ctx := context.WithValue(r.Context(), "clientId", clientID)
									ctx = context.WithValue(ctx, "projectId", projectID)
									r = r.WithContext(ctx)
									s.normalizationHandler.HandleStartClientProjectNormalization(w, r)
								} else {
									s.handleStartClientNormalization(w, r, clientID, projectID)
								}
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						case "stop":
							if r.Method == http.MethodPost {
								s.handleStopClientNormalization(w, r, clientID, projectID)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						case "status":
							if r.Method == http.MethodGet {
								// Используем новый обработчик из NormalizationHandler
								if s.normalizationHandler != nil {
									// Добавляем clientID и projectID в контекст для обработчика
									ctx := context.WithValue(r.Context(), "clientId", clientID)
									ctx = context.WithValue(ctx, "projectId", projectID)
									r = r.WithContext(ctx)
									s.normalizationHandler.HandleGetClientProjectNormalizationStatus(w, r)
								} else {
									s.handleGetClientNormalizationStatus(w, r, clientID, projectID)
								}
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						case "stats":
							if r.Method == http.MethodGet {
								s.handleGetClientNormalizationStats(w, r, clientID, projectID)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						case "preview-stats":
							if r.Method == http.MethodGet {
								// Используем новый обработчик из NormalizationHandler
								if s.normalizationHandler != nil {
									// Добавляем clientID и projectID в контекст для обработчика
									ctx := context.WithValue(r.Context(), "clientId", clientID)
									ctx = context.WithValue(ctx, "projectId", projectID)
									r = r.WithContext(ctx)
									s.normalizationHandler.HandleGetClientProjectNormalizationPreviewStats(w, r)
								} else {
									http.Error(w, "Normalization handler not available", http.StatusInternalServerError)
								}
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						case "groups":
							if r.Method == http.MethodGet {
								s.handleGetClientNormalizationGroups(w, r, clientID, projectID)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						case "sessions":
							if r.Method == http.MethodGet {
								s.handleGetNormalizationSessions(w, r, clientID, projectID)
							} else if r.Method == http.MethodPost {
								s.handleUpdateSessionPriority(w, r, clientID, projectID)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
							return
						}
					}
					// Обработка /api/clients/{id}/projects/{projectId}/normalization/sessions/{sessionId}
					if len(parts) == 6 && parts[4] == "sessions" {
						sessionIDStr := parts[5]
						sessionID, err := ValidateIDPathParam(sessionIDStr, "session_id")
						if err != nil {
							s.writeJSONError(w, r, fmt.Sprintf("Invalid session ID: %s", err.Error()), http.StatusBadRequest)
							return
						}

						// Проверяем путь для возобновления или остановки
						if strings.HasSuffix(r.URL.Path, "/resume") {
							if r.Method == http.MethodPost {
								s.handleResumeNormalizationSession(w, r, clientID, projectID, sessionID)
							} else {
								http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
							}
						} else if r.Method == http.MethodPost {
							s.handleStopNormalizationSession(w, r, clientID, projectID, sessionID)
						} else {
							http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
						}
						return
					}
					http.Error(w, "Invalid route", http.StatusNotFound)
					return
				}

				if len(parts) == 4 && parts[3] == "pipeline-stats" {
					// Обработка /api/clients/{id}/projects/{projectId}/pipeline-stats
					if r.Method == http.MethodGet {
						s.handleGetProjectPipelineStatsWithParams(w, r, clientID, projectID)
					} else {
						http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					}
					return
				}
			}
		}
		http.Error(w, "Invalid route", http.StatusNotFound)
		return
	}

	// Обработка /api/clients/{id}
	switch r.Method {
	case http.MethodGet:
		s.handleGetClient(w, r, clientID)
	case http.MethodPut:
		s.handleUpdateClient(w, r, clientID)
	case http.MethodDelete:
		s.handleDeleteClient(w, r, clientID)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleGetClients получает список клиентов
func (s *Server) handleGetClients(w http.ResponseWriter, r *http.Request) {
	if s.serviceDB == nil {
		LogError(r.Context(), nil, "Service database not available for get clients")
		s.handleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	clients, err := s.serviceDB.GetClientsWithStats()
	if err != nil {
		LogError(r.Context(), err, "Failed to get clients")
		s.handleHTTPError(w, r, NewInternalError("не удалось получить список клиентов", err))
		return
	}

	// Возвращаем пустой массив если клиентов нет (это нормально)
	if clients == nil {
		clients = []map[string]interface{}{}
	}

	LogInfo(r.Context(), "Clients retrieved successfully", "count", len(clients))
	s.writeJSONResponse(w, r, clients, http.StatusOK)
}

// handleCreateClient создает нового клиента
func (s *Server) handleCreateClient(w http.ResponseWriter, r *http.Request) {
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
		LogError(r.Context(), err, "Failed to decode request body for create client")
		s.handleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.Name == "" {
		LogWarn(r.Context(), "Missing required field 'name' for create client")
		s.handleHTTPError(w, r, NewValidationError("поле 'name' обязательно для заполнения", nil))
		return
	}

	LogInfo(r.Context(), "Creating client", "name", req.Name, "legal_name", req.LegalName)
	client, err := s.serviceDB.CreateClient(req.Name, req.LegalName, req.Description, req.ContactEmail, req.ContactPhone, req.TaxID, req.Country, "system")
	if err != nil {
		LogError(r.Context(), err, "Failed to create client", "name", req.Name)
		s.handleHTTPError(w, r, NewInternalError("не удалось создать клиента", err))
		return
	}

	LogInfo(r.Context(), "Client created successfully", "client_id", client.ID, "name", client.Name)
	s.writeJSONResponse(w, r, client, http.StatusCreated)
}

// handleGetClient получает клиента по ID
func (s *Server) handleGetClient(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		LogError(r.Context(), nil, "Service database not available for get client", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	client, err := s.serviceDB.GetClient(clientID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get client", "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Клиент не найден", err))
		return
	}

	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get projects for client", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	// Получаем документы клиента
	var clientDocuments []*database.ClientDocument
	docs, err := s.serviceDB.GetClientDocuments(clientID)
	if err == nil && docs != nil {
		clientDocuments = docs
	}

	// Преобразуем документы в формат для ответа
	documentsResponse := make([]ClientDocument, 0, len(clientDocuments))
	for _, doc := range clientDocuments {
		if doc != nil {
			documentsResponse = append(documentsResponse, ClientDocument{
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

	// Подсчет статистики
	var totalBenchmarks int
	var activeSessions int
	var totalQualityScore float64
	var qualityCount int

	if s.serviceDB != nil {
		conn := s.serviceDB.GetConnection()

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

	avgQualityScore := func() float64 {
		if qualityCount > 0 {
			return totalQualityScore
		}
		return 0.0
	}()

	response := ClientDetailResponse{
		Client: Client{
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
		Projects:  make([]ClientProject, len(projects)),
		Documents: documentsResponse,
		Statistics: ClientStatistics{
			TotalProjects:   len(projects),
			TotalBenchmarks: totalBenchmarks,
			ActiveSessions:  activeSessions,
			AvgQualityScore: avgQualityScore,
		},
	}

	for i, p := range projects {
		response.Projects[i] = ClientProject{
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

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetClientStatistics получает расширенную статистику клиента
func (s *Server) handleGetClientStatistics(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		LogError(r.Context(), nil, "Service database not available for get client statistics", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get projects for client statistics", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	// Подсчет общей статистики
	var totalNomenclature int
	var totalCounterparties int
	var totalDatabases int
	var totalQualityScore float64
	var qualityCount int

	// Статистика по проектам
	projectStats := make([]map[string]interface{}, 0, len(projects))

	for _, project := range projects {
		// Получаем базы данных проекта
		projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
		if err == nil {
			totalDatabases += len(projectDBs)
		}

		// Собираем информацию о конфигурациях 1С для баз данных проекта
		configurations := make([]string, 0)
		for _, db := range projectDBs {
			if db.FilePath != "" {
				fileName := filepath.Base(db.FilePath)
				fileInfo := ParseDatabaseFileInfo(fileName)
				if fileInfo.ConfigName != "" && fileInfo.ConfigName != "Unknown" {
					configurations = append(configurations, fileInfo.ConfigName)
				}
			}
		}

		// Подсчитываем номенклатуру и контрагентов из нормализованной БД
		// Это упрощенная версия - в реальности нужно запрашивать из normalized_data
		projectStat := map[string]interface{}{
			"project_id":           project.ID,
			"project_name":         project.Name,
			"project_type":         project.ProjectType,
			"status":               project.Status,
			"total_nomenclature":   0, // Будет заполнено из БД
			"total_counterparties": 0, // Будет заполнено из БД
			"total_databases":      len(projectDBs),
			"avg_quality_score":    0.0,
			"last_updated":         project.UpdatedAt.Format(time.RFC3339),
			"configurations":       configurations, // Конфигурации 1С
		}

		// Запрос к normalized_data для подсчета номенклатуры через сессии
		if s.normalizedDB != nil && s.serviceDB != nil {
			// Получаем session IDs для проекта
			projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
			if err == nil && len(projectDBs) > 0 {
				dbIDs := make([]interface{}, len(projectDBs))
				for i, db := range projectDBs {
					dbIDs[i] = db.ID
				}

				query := "SELECT id FROM normalization_sessions WHERE project_database_id IN (" + strings.Repeat("?,", len(dbIDs)-1) + "?)"
				rows, err := s.serviceDB.Query(query, dbIDs...)
				if err == nil {
					var sessionIDs []interface{}
					defer rows.Close()
					for rows.Next() {
						var sessionID int
						if err := rows.Scan(&sessionID); err == nil {
							sessionIDs = append(sessionIDs, sessionID)
						}
					}

					if len(sessionIDs) > 0 {
						// Подсчет номенклатуры
						var nomCount int
						countQuery := "SELECT COUNT(*) FROM normalized_data WHERE normalization_session_id IN (" + strings.Repeat("?,", len(sessionIDs)-1) + "?)"
						err = s.normalizedDB.QueryRow(countQuery, sessionIDs...).Scan(&nomCount)
						if err == nil {
							projectStat["total_nomenclature"] = nomCount
							totalNomenclature += nomCount
						}

						// Подсчет среднего качества
						var avgQuality sql.NullFloat64
						qualityQuery := "SELECT AVG(quality_score) FROM normalized_data WHERE normalization_session_id IN (" + strings.Repeat("?,", len(sessionIDs)-1) + "?) AND quality_score IS NOT NULL"
						err = s.normalizedDB.QueryRow(qualityQuery, sessionIDs...).Scan(&avgQuality)
						if err == nil && avgQuality.Valid {
							projectStat["avg_quality_score"] = avgQuality.Float64
							totalQualityScore += avgQuality.Float64
							qualityCount++
						}
					}
				}
			}
		}

		projectStats = append(projectStats, projectStat)
	}

	// Поиск и обработка непривязанных баз данных
	unlinkedDatabases := make([]map[string]interface{}, 0)
	unlinkedNomenclature := 0
	unlinkedCounterparties := 0
	unlinkedDBCount := 0

	uploadsDir, err := EnsureUploadsDirectory(".")
	if err == nil {
		files, err := filepath.Glob(filepath.Join(uploadsDir, "*.db"))
		if err == nil {
			// Получаем все привязанные базы данных
			allLinkedDBs := make(map[string]bool)
			for _, project := range projects {
				projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
				if err == nil {
					for _, db := range projectDBs {
						if db.FilePath != "" {
							absPath, _ := filepath.Abs(db.FilePath)
							allLinkedDBs[absPath] = true
							// Также проверяем по имени файла
							allLinkedDBs[filepath.Base(db.FilePath)] = true
						}
					}
				}
			}

			// Проверяем каждую базу данных в папке uploads
			for _, filePath := range files {
				absPath, _ := filepath.Abs(filePath)
				fileName := filepath.Base(filePath)

				// Проверяем, привязана ли база
				isLinked := allLinkedDBs[absPath] || allLinkedDBs[fileName]

				// Также проверяем через GetProjectDatabaseByFilePath для всех проектов
				if !isLinked {
					for _, project := range projects {
						db, err := s.serviceDB.GetProjectDatabaseByFilePath(project.ID, filePath)
						if err == nil && db != nil {
							isLinked = true
							break
						}
					}
				}

				if !isLinked {
					// Это непривязанная база данных
					unlinkedDBCount++

					// Парсим информацию из имени файла
					fileInfo := ParseDatabaseFileInfo(fileName)

					// Получаем размер файла
					var size int64
					if info, err := os.Stat(filePath); err == nil {
						size = info.Size()
					}

					// Подсчитываем записи из базы данных (упрощенная версия)
					// В реальности нужно открывать базу и считать записи
					dbInfo := map[string]interface{}{
						"file_path":     filePath,
						"file_name":     fileName,
						"display_name":  fileInfo.DisplayName,
						"config_name":   fileInfo.ConfigName,
						"database_type": fileInfo.DatabaseType,
						"data_type":     fileInfo.DataType,
						"size":          size,
					}

					unlinkedDatabases = append(unlinkedDatabases, dbInfo)

					// Подсчет номенклатуры и контрагентов из непривязанных баз
					// Это упрощенная версия - в реальности нужно открывать базу и считать
					// Пока оставляем 0, так как для точного подсчета нужно открывать SQLite базу
				}
			}
		}
	}

	// Обновляем общее количество баз данных
	totalDatabases += unlinkedDBCount

	// Вычисляем среднее качество
	avgQuality := 0.0
	if qualityCount > 0 {
		avgQuality = totalQualityScore / float64(qualityCount)
	}

	response := map[string]interface{}{
		"total_projects":       len(projects),
		"total_benchmarks":     0, // Можно добавить подсчет
		"active_sessions":      0, // Можно добавить подсчет
		"avg_quality_score":    avgQuality,
		"total_nomenclature":   totalNomenclature + unlinkedNomenclature,
		"total_counterparties": totalCounterparties + unlinkedCounterparties,
		"total_databases":      totalDatabases,
		"projects":             projectStats,
		"unlinked_databases":   unlinkedDatabases, // Непривязанные базы данных
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// NomenclatureResult представляет результат номенклатуры из разных источников
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

// getNomenclatureFromMainDB получает номенклатуру из основной базы данных
func (s *Server) getNomenclatureFromMainDB(dbPath string, clientID int, projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*NomenclatureResult, int, error) {
	// Используем кэш подключений для оптимизации
	db, err := s.dbConnectionCache.GetConnection(dbPath)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open database %s: %w", dbPath, err)
	}
	defer s.dbConnectionCache.ReleaseConnection(dbPath)

	// Проверяем наличие таблиц
	var hasCatalogItems, hasNomenclatureItems bool
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='catalog_items'").Scan(&hasCatalogItems)
	if err == nil && hasCatalogItems {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&count)
		hasCatalogItems = (err == nil && count > 0)
	}

	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='nomenclature_items'").Scan(&hasNomenclatureItems)
	if err == nil && hasNomenclatureItems {
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM nomenclature_items").Scan(&count)
		hasNomenclatureItems = (err == nil && count > 0)
	}

	if !hasCatalogItems && !hasNomenclatureItems {
		return []*NomenclatureResult{}, 0, nil
	}

	// Получаем upload_id для проектов клиента
	uploadIDs := make([]interface{}, 0)
	uploadQuery := "SELECT DISTINCT id FROM uploads WHERE client_id = ?"
	uploadArgs := []interface{}{clientID}

	if len(projectIDs) > 0 {
		uploadQuery += " AND project_id IN (" + strings.Repeat("?,", len(projectIDs)-1) + "?)"
		for _, pid := range projectIDs {
			uploadArgs = append(uploadArgs, pid)
		}
	}

	rows, err := db.Query(uploadQuery, uploadArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query uploads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var uploadID int
		if err := rows.Scan(&uploadID); err == nil {
			uploadIDs = append(uploadIDs, uploadID)
		}
	}

	// Если upload записей нет, но есть данные в catalog_items, пытаемся получить их напрямую
	// Это fallback для случаев, когда upload записи еще не созданы
	if len(uploadIDs) == 0 {
		// Пытаемся получить все catalog_items без фильтрации по upload_id
		// Это менее точный способ, но позволяет показать данные, если они есть
		if hasCatalogItems {
			fallbackQuery := `
				SELECT ci.id, ci.code, ci.name, ci.reference, c.id as catalog_id
				FROM catalog_items ci
				INNER JOIN catalogs c ON ci.catalog_id = c.id
				WHERE 1=1
			`
			fallbackArgs := []interface{}{}
			
			if search != "" {
				fallbackQuery += " AND (ci.name LIKE ? OR ci.code LIKE ?)"
				searchPattern := "%" + search + "%"
				fallbackArgs = append(fallbackArgs, searchPattern, searchPattern)
			}
			
			fallbackQuery += " ORDER BY ci.id LIMIT 100000"
			
			fallbackRows, err := db.Query(fallbackQuery, fallbackArgs...)
			if err == nil {
				defer fallbackRows.Close()
				fallbackResults := make([]*NomenclatureResult, 0)
				for fallbackRows.Next() {
					var id int
					var code, name, reference sql.NullString
					var catalogID int
					
					if err := fallbackRows.Scan(&id, &code, &name, &reference, &catalogID); err == nil {
						// Используем первый projectID из списка, так как не можем определить точно
						pid := 0
						if len(projectIDs) > 0 {
							pid = projectIDs[0]
						}
						
						result := &NomenclatureResult{
							ID:              id,
							Code:            code.String,
							Name:            name.String,
							NormalizedName:  name.String,
							SourceDatabase:  dbPath,
							SourceType:      "main",
							ProjectID:       pid,
							ProjectName:     projectNames[pid],
							SourceReference: reference.String,
							SourceName:      name.String,
						}
						fallbackResults = append(fallbackResults, result)
					}
				}
				
				if len(fallbackResults) > 0 {
					log.Printf("Found %d catalog items without upload records in %s, using fallback method", len(fallbackResults), dbPath)
					return fallbackResults, len(fallbackResults), nil
				}
			}
		}
		
		// Если fallback не дал результатов, возвращаем пустой список
		return []*NomenclatureResult{}, 0, nil
	}

	// Собираем результаты из catalog_items и nomenclature_items
	results := make([]*NomenclatureResult, 0)
	projectIDMap := make(map[int]int) // upload_id -> project_id

	// Получаем маппинг upload_id -> project_id
	uploadProjectQuery := "SELECT id, project_id FROM uploads WHERE id IN (" + strings.Repeat("?,", len(uploadIDs)-1) + "?)"
	uploadProjectRows, err := db.Query(uploadProjectQuery, uploadIDs...)
	if err == nil {
		defer uploadProjectRows.Close()
		for uploadProjectRows.Next() {
			var uid, pid sql.NullInt64
			if err := uploadProjectRows.Scan(&uid, &pid); err == nil && uid.Valid && pid.Valid {
				projectIDMap[int(uid.Int64)] = int(pid.Int64)
			}
		}
	}

	// Запрос из catalog_items
	if hasCatalogItems {
		query := `
			SELECT ci.id, ci.code, ci.name, ci.reference,
			       c.upload_id, u.project_id
			FROM catalog_items ci
			INNER JOIN catalogs c ON ci.catalog_id = c.id
			INNER JOIN uploads u ON c.upload_id = u.id
			WHERE c.upload_id IN (` + strings.Repeat("?,", len(uploadIDs)-1) + "?)"
		args := append([]interface{}{}, uploadIDs...)

		if search != "" {
			query += " AND (ci.name LIKE ? OR ci.code LIKE ?)"
			searchPattern := "%" + search + "%"
			args = append(args, searchPattern, searchPattern)
		}

		query += " ORDER BY ci.id LIMIT 100000"

		catalogRows, err := db.Query(query, args...)
		if err == nil {
			defer catalogRows.Close()
			for catalogRows.Next() {
				var id int
				var code, name, reference sql.NullString
				var uploadID, projectID sql.NullInt64

				if err := catalogRows.Scan(&id, &code, &name, &reference, &uploadID, &projectID); err == nil {
					pid := 0
					if projectID.Valid {
						pid = int(projectID.Int64)
					} else if uploadID.Valid {
						if mappedPID, ok := projectIDMap[int(uploadID.Int64)]; ok {
							pid = mappedPID
						}
					}

					result := &NomenclatureResult{
						ID:              id,
						Code:            code.String,
						Name:            name.String,
						NormalizedName:  name.String,
						SourceDatabase:  dbPath,
						SourceType:      "main",
						ProjectID:       pid,
						ProjectName:     projectNames[pid],
						SourceReference: reference.String,
						SourceName:      name.String,
					}
					results = append(results, result)
				}
			}
		}
	}

	// Запрос из nomenclature_items
	if hasNomenclatureItems {
		query := `
			SELECT id, nomenclature_code, nomenclature_name, nomenclature_reference,
			       upload_id
			FROM nomenclature_items
			WHERE upload_id IN (` + strings.Repeat("?,", len(uploadIDs)-1) + "?)"
		args := append([]interface{}{}, uploadIDs...)

		if search != "" {
			query += " AND (nomenclature_name LIKE ? OR nomenclature_code LIKE ?)"
			searchPattern := "%" + search + "%"
			args = append(args, searchPattern, searchPattern)
		}

		query += " ORDER BY id LIMIT 100000"

		nomenclatureRows, err := db.Query(query, args...)
		if err == nil {
			defer nomenclatureRows.Close()
			for nomenclatureRows.Next() {
				var id int
				var code, name, reference sql.NullString
				var uploadID sql.NullInt64

				if err := nomenclatureRows.Scan(&id, &code, &name, &reference, &uploadID); err == nil {
					pid := 0
					if uploadID.Valid {
						if mappedPID, ok := projectIDMap[int(uploadID.Int64)]; ok {
							pid = mappedPID
						}
					}

					result := &NomenclatureResult{
						ID:              id,
						Code:            code.String,
						Name:            name.String,
						NormalizedName:  name.String,
						SourceDatabase:  dbPath,
						SourceType:      "main",
						ProjectID:       pid,
						ProjectName:     projectNames[pid],
						SourceReference: reference.String,
						SourceName:      name.String,
					}
					results = append(results, result)
				}
			}
		}
	}

	// Подсчет общего количества (упрощенный, так как мы объединяем две таблицы)
	total := len(results)
	return results, total, nil
}

// getNomenclatureFromNormalizedDB получает номенклатуру из нормализованной базы данных
func (s *Server) getNomenclatureFromNormalizedDB(projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*NomenclatureResult, int, error) {
	if s.normalizedDB == nil {
		return []*NomenclatureResult{}, 0, nil
	}

	if len(projectIDs) == 0 {
		return []*NomenclatureResult{}, 0, nil
	}

	query := `
		SELECT id, code, normalized_name, category, quality_score, 
		       kpved_code, kpved_name, source_reference, source_name,
		       ai_confidence, ai_reasoning, processing_level, merged_count, project_id
		FROM normalized_data
		WHERE project_id IN (` + strings.Repeat("?,", len(projectIDs)-1) + "?)"
	args := append([]interface{}{}, make([]interface{}, len(projectIDs))...)
	for i, pid := range projectIDs {
		args[i] = pid
	}

	if search != "" {
		query += " AND (normalized_name LIKE ? OR code LIKE ?)"
		searchPattern := "%" + search + "%"
		args = append(args, searchPattern, searchPattern)
	}

	// Подсчет общего количества
	var total int
	countQuery := "SELECT COUNT(*) FROM normalized_data WHERE project_id IN (" + strings.Repeat("?,", len(projectIDs)-1) + "?)"
	countArgs := append([]interface{}{}, make([]interface{}, len(projectIDs))...)
	for i, pid := range projectIDs {
		countArgs[i] = pid
	}
	if search != "" {
		countQuery += " AND (normalized_name LIKE ? OR code LIKE ?)"
		searchPattern := "%" + search + "%"
		countArgs = append(countArgs, searchPattern, searchPattern)
	}
	err := s.normalizedDB.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		log.Printf("Error counting normalized nomenclature: %v", err)
		total = 0
	}

	query += " ORDER BY id LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.normalizedDB.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query normalized data: %w", err)
	}
	defer rows.Close()

	results := make([]*NomenclatureResult, 0)
	for rows.Next() {
		var id, projectID int
		var code, normalizedName, category sql.NullString
		var qualityScore sql.NullFloat64
		var kpvedCode, kpvedName sql.NullString
		var sourceRef, sourceName sql.NullString
		var aiConfidence sql.NullFloat64
		var aiReasoning, processingLevel sql.NullString
		var mergedCount int

		err := rows.Scan(&id, &code, &normalizedName, &category, &qualityScore,
			&kpvedCode, &kpvedName, &sourceRef, &sourceName,
			&aiConfidence, &aiReasoning, &processingLevel, &mergedCount, &projectID)
		if err != nil {
			log.Printf("Error scanning normalized nomenclature row: %v", err)
			continue
		}

		result := &NomenclatureResult{
			ID:              id,
			Code:            code.String,
			Name:            sourceName.String,
			NormalizedName:  normalizedName.String,
			Category:        category.String,
			QualityScore:    0.0,
			SourceDatabase:  "normalized_data.db",
			SourceType:      "normalized",
			ProjectID:       projectID,
			ProjectName:     projectNames[projectID],
			SourceReference: sourceRef.String,
			SourceName:      sourceName.String,
			MergedCount:     mergedCount,
		}

		if qualityScore.Valid {
			result.QualityScore = qualityScore.Float64
		}
		if kpvedCode.Valid {
			result.KpvedCode = kpvedCode.String
		}
		if kpvedName.Valid {
			result.KpvedName = kpvedName.String
		}
		if aiConfidence.Valid {
			result.AIConfidence = aiConfidence.Float64
		}
		if aiReasoning.Valid {
			result.AIReasoning = aiReasoning.String
		}
		if processingLevel.Valid {
			result.ProcessingLevel = processingLevel.String
		}

		results = append(results, result)
	}

	return results, total, nil
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

// handleGetClientNomenclature получает номенклатуру клиента из всех баз данных
func (s *Server) handleGetClientNomenclature(w http.ResponseWriter, r *http.Request, clientID int) {
	const maxRecordsPerDB = 100000

	// Параметры пагинации
	page, limit, err := ValidatePaginationParams(r, 1, 20, 100)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}
	search := r.URL.Query().Get("search")

	offset := (page - 1) * limit

	// Получаем проекты клиента
	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	if len(projects) == 0 {
		s.writeJSONResponse(w, r, map[string]interface{}{
			"items": []interface{}{},
			"total": 0,
			"page":  page,
			"limit": limit,
		}, http.StatusOK)
		return
	}

	// Создаем маппинг project_id -> project_name
	projectIDs := make([]int, len(projects))
	projectNames := make(map[int]string)
	for i, p := range projects {
		projectIDs[i] = p.ID
		projectNames[p.ID] = p.Name
	}

	// Собираем результаты из всех баз данных
	allResults := make([]*NomenclatureResult, 0)

	// 1. Получаем данные из нормализованной базы
	// ВАЖНО: Используем большой лимит для получения всех записей, необходимых для правильной дедупликации
	// При больших объемах данных это может быть медленно. В будущем можно оптимизировать:
	// - Использовать индексы для быстрого поиска уникальных комбинаций код+название
	// - Применять пагинацию на уровне БД с последующей дедупликацией только на границах страниц
	// - Кэшировать результаты дедупликации
	normalizedResults, _, err := s.getNomenclatureFromNormalizedDB(projectIDs, projectNames, search, maxRecordsPerDB, 0)
	if err != nil {
		log.Printf("Error getting normalized nomenclature: %v (continuing with other databases)", err)
		// Продолжаем работу, даже если нормализованная база недоступна
	} else {
		allResults = append(allResults, normalizedResults...)
	}

	// 2. Получаем данные из всех баз данных проектов
	if s.serviceDB != nil {
		for _, project := range projects {
			projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
			if err != nil {
				log.Printf("Error getting databases for project %d: %v", project.ID, err)
				continue
			}

			for _, projectDB := range projectDBs {
				if !projectDB.IsActive {
					continue
				}

				// Проверяем, что файл существует
				if _, err := os.Stat(projectDB.FilePath); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						log.Printf("Database file not found: %s", projectDB.FilePath)
					} else {
						log.Printf("Error checking database file %s: %v", projectDB.FilePath, err)
					}
					continue
				}

				// Получаем данные из основной базы
				// Метод getNomenclatureFromMainDB сам проверяет наличие нужных таблиц
				// (catalog_items или nomenclature_items) и возвращает пустой результат, если их нет
				// Используем большой лимит, чтобы получить все записи для последующего объединения
				mainResults, _, err := s.getNomenclatureFromMainDB(projectDB.FilePath, clientID, []int{project.ID}, projectNames, search, 100000, 0)
				if err != nil {
					log.Printf("Error getting nomenclature from main DB %s: %v (skipping this database)", projectDB.FilePath, err)
					// Продолжаем работу с другими базами, даже если одна недоступна
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

	// Объединяем и применяем пагинацию
	paginatedResults, total := mergeNomenclatureResults(allResults, limit, offset)

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

	s.writeJSONResponse(w, r, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	}, http.StatusOK)
}

// handleGetProjectNomenclature получает номенклатуру проекта из всех баз данных
func (s *Server) handleGetProjectNomenclature(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	startTime := time.Now()
	const maxRecordsPerDB = 100000

	// Параметры пагинации
	page, limit, err := ValidatePaginationParams(r, 1, 20, 100)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}
	search := r.URL.Query().Get("search")

	offset := (page - 1) * limit

	// Получаем информацию о проекте
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	projectNames := map[int]string{projectID: project.Name}

	// Собираем результаты из всех баз данных
	allResults := make([]*NomenclatureResult, 0)

	// 1. Получаем данные из нормализованной базы для этого проекта
	// ВАЖНО: Используем большой лимит для получения всех записей, необходимых для правильной дедупликации
	normalizedResults, _, err := s.getNomenclatureFromNormalizedDB([]int{projectID}, projectNames, search, maxRecordsPerDB, 0)
	if err != nil {
		log.Printf("Error getting normalized nomenclature for project %d: %v (continuing with other databases)", projectID, err)
	} else {
		allResults = append(allResults, normalizedResults...)
	}

	// 2. Получаем данные из всех баз данных проекта
	if s.serviceDB != nil {
		projectDBs, err := s.serviceDB.GetProjectDatabases(projectID, false)
		if err != nil {
			log.Printf("Error getting databases for project %d: %v", projectID, err)
		} else {
			for _, projectDB := range projectDBs {
				if !projectDB.IsActive {
					continue
				}

				// Проверяем, что файл существует
				if _, err := os.Stat(projectDB.FilePath); err != nil {
					if errors.Is(err, os.ErrNotExist) {
						log.Printf("Database file not found: %s", projectDB.FilePath)
					} else {
						log.Printf("Error checking database file %s: %v", projectDB.FilePath, err)
					}
					continue
				}

				// Получаем данные из основной базы
				// Метод getNomenclatureFromMainDB сам проверяет наличие нужных таблиц
				// (catalog_items или nomenclature_items) и возвращает пустой результат, если их нет
				// ВАЖНО: Используем большой лимит для получения всех записей, необходимых для правильной дедупликации
				mainResults, _, err := s.getNomenclatureFromMainDB(projectDB.FilePath, clientID, []int{projectID}, projectNames, search, maxRecordsPerDB, 0)
				if err != nil {
					log.Printf("Error getting nomenclature from main DB %s: %v (skipping this database)", projectDB.FilePath, err)
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

	// Объединяем и применяем пагинацию
	paginatedResults, total := mergeNomenclatureResults(allResults, limit, offset)

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

	duration := time.Since(startTime)
	log.Printf("handleGetProjectNomenclature: clientID=%d, projectID=%d, total=%d, duration=%v", clientID, projectID, total, duration)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"items": items,
		"total": total,
		"page":  page,
		"limit": limit,
	}, http.StatusOK)
}

// handleGetClientDatabases получает все базы данных клиента
func (s *Server) handleGetClientDatabases(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	allDatabases := make([]map[string]interface{}, 0)

	for _, project := range projects {
		projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			log.Printf("Error getting databases for project %d: %v", project.ID, err)
			continue
		}

		for _, db := range projectDBs {
			// Получаем размер файла, если возможно
			var size int64
			if db.FilePath != "" {
				if info, err := os.Stat(db.FilePath); err == nil {
					size = info.Size()
				}
			}

			status := "active"
			if !db.IsActive {
				status = "inactive"
			}

			dbInfo := map[string]interface{}{
				"id":           db.ID,
				"name":         db.Name,
				"path":         db.FilePath,
				"size":         size,
				"created_at":   db.CreatedAt.Format(time.RFC3339),
				"status":       status,
				"project_id":   project.ID,
				"project_name": project.Name,
			}

			allDatabases = append(allDatabases, dbInfo)
		}
	}

	s.writeJSONResponse(w, r, allDatabases, http.StatusOK)
}

// handleLinkClientDatabase привязывает базу данных к проекту
// PUT /api/clients/{clientId}/databases/{databaseId}/link
func (s *Server) handleLinkClientDatabase(w http.ResponseWriter, r *http.Request, clientID, databaseID int) {
	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		ProjectID int `json:"project_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ProjectID <= 0 {
		s.writeJSONError(w, r, "project_id is required and must be positive", http.StatusBadRequest)
		return
	}

	// Проверяем, что проект принадлежит клиенту
	project, err := s.serviceDB.GetClientProject(req.ProjectID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Project not found: %v", err), http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusForbidden)
		return
	}

	// Проверяем, что база данных существует и принадлежит клиенту (через проекты)
	dbRecord, err := s.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Database not found: %v", err), http.StatusNotFound)
		return
	}

	// Проверяем, что база данных принадлежит одному из проектов клиента
	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get client projects: %v", err), http.StatusInternalServerError)
		return
	}

	belongsToClient := false
	for _, p := range projects {
		if dbRecord.ClientProjectID == p.ID {
			belongsToClient = true
			break
		}
	}

	// Если база данных непривязана (client_project_id = 0 или NULL), разрешаем привязку
	if dbRecord.ClientProjectID == 0 {
		belongsToClient = true
	}

	if !belongsToClient {
		s.writeJSONError(w, r, "Database does not belong to this client", http.StatusForbidden)
		return
	}

	// Привязываем базу данных к проекту
	if err := s.serviceDB.LinkProjectDatabase(databaseID, req.ProjectID); err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to link database: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем обновленную информацию о базе данных
	updatedDB, err := s.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get updated database: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, updatedDB, http.StatusOK)
}

// handleAutoLinkClientDatabases автоматически привязывает непривязанные базы данных к проектам
// POST /api/clients/{clientId}/databases/auto-link
func (s *Server) handleAutoLinkClientDatabases(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Получаем все проекты клиента
	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get client projects: %v", err), http.StatusInternalServerError)
		return
	}

	// Получаем все базы данных клиента (через проекты)
	allDatabases := make([]*database.ProjectDatabase, 0)
	for _, project := range projects {
		projectDBs, err := s.serviceDB.GetProjectDatabases(project.ID, false)
		if err != nil {
			log.Printf("Error getting databases for project %d: %v", project.ID, err)
			continue
		}
		allDatabases = append(allDatabases, projectDBs...)
	}

	// Находим непривязанные базы (client_project_id = 0 или NULL)
	unlinkedDatabases := make([]*database.ProjectDatabase, 0)
	for _, db := range allDatabases {
		if db.ClientProjectID == 0 {
			unlinkedDatabases = append(unlinkedDatabases, db)
		}
	}

	// Также ищем базы данных в папке data/uploads, которые еще не добавлены в БД
	uploadsDir, err := EnsureUploadsDirectory(".")
	if err == nil {
		files, err := filepath.Glob(filepath.Join(uploadsDir, "*.db"))
		if err == nil {
			for _, filePath := range files {
				fileName := filepath.Base(filePath)
				// Проверяем, есть ли уже эта база в БД
				found := false
				for _, db := range allDatabases {
					if db.FilePath == filePath || strings.HasSuffix(db.FilePath, fileName) {
						found = true
						break
					}
				}
				if !found {
					// Парсим информацию из имени файла
					fileInfo := ParseDatabaseFileInfo(fileName)
					// Создаем временную запись для обработки
					tempDB := &database.ProjectDatabase{
						ID:              0, // Временный ID
						FilePath:        filePath,
						Name:            fileInfo.DisplayName,
						ClientProjectID: 0, // Непривязанная
					}
					unlinkedDatabases = append(unlinkedDatabases, tempDB)
				}
			}
		}
	}

	linkedCount := 0
	errors := make([]string, 0)

	// Пытаемся автоматически привязать каждую непривязанную базу
	for _, db := range unlinkedDatabases {
		if db.ID == 0 {
			// База еще не добавлена в БД, пропускаем (нужно сначала добавить через upload)
			continue
		}

		// Парсим информацию из имени файла
		fileName := filepath.Base(db.FilePath)
		fileInfo := ParseDatabaseFileInfo(fileName)

		// Ищем подходящий проект по типу данных
		var targetProject *database.ClientProject
		for _, project := range projects {
			if project.ProjectType == fileInfo.DataType ||
				(fileInfo.DataType == "nomenclature" && project.ProjectType == "nomenclature_counterparties") ||
				(fileInfo.DataType == "counterparties" && project.ProjectType == "nomenclature_counterparties") {
				targetProject = project
				break
			}
		}

		if targetProject == nil {
			// Если не нашли по типу, берем первый проект клиента
			if len(projects) > 0 {
				targetProject = projects[0]
			} else {
				errors = append(errors, fmt.Sprintf("No projects found for database %s", db.Name))
				continue
			}
		}

		// Привязываем базу к проекту
		if err := s.serviceDB.LinkProjectDatabase(db.ID, targetProject.ID); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to link database %s: %v", db.Name, err))
			continue
		}

		linkedCount++
	}

	response := map[string]interface{}{
		"linked_count":   linkedCount,
		"total_unlinked": len(unlinkedDatabases),
		"errors":         errors,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleUpdateClient обновляет клиента
func (s *Server) handleUpdateClient(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for update client %d", clientID)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		Name         string `json:"name"`
		LegalName    string `json:"legal_name"`
		Description  string `json:"description"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
		TaxID        string `json:"tax_id"`
		Country      string `json:"country"`
		Status       string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body for update client %d: %v", clientID, err)
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Updating client %d: name=%q, legal_name=%q", clientID, req.Name, req.LegalName)
	if err := s.serviceDB.UpdateClient(clientID, req.Name, req.LegalName, req.Description, req.ContactEmail, req.ContactPhone, req.TaxID, req.Country, req.Status); err != nil {
		log.Printf("Error updating client %d: %v", clientID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	client, err := s.serviceDB.GetClient(clientID)
	if err != nil {
		log.Printf("Error getting updated client %d: %v", clientID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Client %d updated successfully", clientID)
	s.writeJSONResponse(w, r, client, http.StatusOK)
}

// handleDeleteClient удаляет клиента
func (s *Server) handleDeleteClient(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		LogError(r.Context(), nil, "Service database not available for delete client", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	LogInfo(r.Context(), "Deleting client", "client_id", clientID)
	if err := s.serviceDB.DeleteClient(clientID); err != nil {
		LogError(r.Context(), err, "Failed to delete client", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("не удалось удалить клиента", err))
		return
	}

	LogInfo(r.Context(), "Client deleted successfully", "client_id", clientID)
	s.writeJSONResponse(w, r, map[string]string{"message": "Client deleted"}, http.StatusOK)
}

// handleGetClientProjects получает проекты клиента
func (s *Server) handleGetClientProjects(w http.ResponseWriter, r *http.Request, clientID int) {
	if s.serviceDB == nil {
		LogError(r.Context(), nil, "Service database not available for get projects", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("Service database not available", nil))
		return
	}

	projects, err := s.serviceDB.GetClientProjects(clientID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get projects for client", "client_id", clientID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить проекты клиента", err))
		return
	}

	LogInfo(r.Context(), "Projects retrieved for client", "client_id", clientID, "count", len(projects))

	response := make([]ClientProject, len(projects))
	for i, p := range projects {
		response[i] = ClientProject{
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

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleCreateClientProject создает проект клиента
func (s *Server) handleCreateClientProject(w http.ResponseWriter, r *http.Request, clientID int) {
	var req struct {
		Name               string  `json:"name"`
		ProjectType        string  `json:"project_type"`
		Description        string  `json:"description"`
		SourceSystem       string  `json:"source_system"`
		TargetQualityScore float64 `json:"target_quality_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogError(r.Context(), err, "Failed to decode request body for create project", "client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.Name == "" || req.ProjectType == "" {
		LogWarn(r.Context(), "Missing required fields for create project", "client_id", clientID, "name", req.Name, "project_type", req.ProjectType)
		s.handleHTTPError(w, r, NewValidationError("поля 'name' и 'project_type' обязательны", nil))
		return
	}

	LogInfo(r.Context(), "Creating project", "client_id", clientID, "name", req.Name, "type", req.ProjectType)
	project, err := s.serviceDB.CreateClientProject(clientID, req.Name, req.ProjectType, req.Description, req.SourceSystem, req.TargetQualityScore)
	if err != nil {
		LogError(r.Context(), err, "Failed to create project", "client_id", clientID, "name", req.Name)
		s.handleHTTPError(w, r, NewInternalError("не удалось создать проект", err))
		return
	}

	LogInfo(r.Context(), "Project created successfully", "project_id", project.ID, "client_id", clientID, "name", project.Name)

	response := ClientProject{
		ID:                 project.ID,
		ClientID:           project.ClientID,
		Name:               project.Name,
		ProjectType:        project.ProjectType,
		Description:        project.Description,
		SourceSystem:       project.SourceSystem,
		Status:             project.Status,
		TargetQualityScore: project.TargetQualityScore,
		CreatedAt:          project.CreatedAt,
		UpdatedAt:          project.UpdatedAt,
	}

	s.writeJSONResponse(w, r, response, http.StatusCreated)
}

// handleGetClientProject получает проект по ID
func (s *Server) handleGetClientProject(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	// Получаем эталоны проекта
	benchmarks, err := s.serviceDB.GetClientBenchmarks(projectID, "", false)
	if err != nil {
		log.Printf("Error fetching benchmarks: %v", err)
		benchmarks = []*database.ClientBenchmark{}
	}

	// Вычисляем статистику
	totalBenchmarks := len(benchmarks)
	approvedBenchmarks := 0
	totalQuality := 0.0
	for _, b := range benchmarks {
		if b.IsApproved {
			approvedBenchmarks++
		}
		totalQuality += b.QualityScore
	}
	avgQuality := 0.0
	if totalBenchmarks > 0 {
		avgQuality = totalQuality / float64(totalBenchmarks)
	}

	response := map[string]interface{}{
		"project": ClientProject{
			ID:                 project.ID,
			ClientID:           project.ClientID,
			Name:               project.Name,
			ProjectType:        project.ProjectType,
			Description:        project.Description,
			SourceSystem:       project.SourceSystem,
			Status:             project.Status,
			TargetQualityScore: project.TargetQualityScore,
			CreatedAt:          project.CreatedAt,
			UpdatedAt:          project.UpdatedAt,
		},
		"benchmarks": benchmarks[:min(10, len(benchmarks))], // Первые 10 эталонов
		"statistics": map[string]interface{}{
			"total_benchmarks":    totalBenchmarks,
			"approved_benchmarks": approvedBenchmarks,
			"avg_quality_score":   avgQuality,
		},
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// handleUpdateClientProject обновляет проект клиента
func (s *Server) handleUpdateClientProject(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for update", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (update)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	var req struct {
		Name               string  `json:"name"`
		ProjectType        string  `json:"project_type"`
		Description        string  `json:"description"`
		SourceSystem       string  `json:"source_system"`
		Status             string  `json:"status"`
		TargetQualityScore float64 `json:"target_quality_score"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body for update project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Updating project %d for client %d: name=%q, type=%q, status=%q", projectID, clientID, req.Name, req.ProjectType, req.Status)
	if err := s.serviceDB.UpdateClientProject(projectID, req.Name, req.ProjectType, req.Description, req.SourceSystem, req.Status, req.TargetQualityScore); err != nil {
		log.Printf("Error updating project %d: %v", projectID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Получаем обновленный проект
	updated, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting updated project %d: %v", projectID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Project %d updated successfully for client %d", projectID, clientID)

	response := ClientProject{
		ID:                 updated.ID,
		ClientID:           updated.ClientID,
		Name:               updated.Name,
		ProjectType:        updated.ProjectType,
		Description:        updated.Description,
		SourceSystem:       updated.SourceSystem,
		Status:             updated.Status,
		TargetQualityScore: updated.TargetQualityScore,
		CreatedAt:          updated.CreatedAt,
		UpdatedAt:          updated.UpdatedAt,
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleDeleteClientProject удаляет проект клиента
func (s *Server) handleDeleteClientProject(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for delete", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (delete)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	LogInfo(r.Context(), "Deleting project", "project_id", projectID, "client_id", clientID)
	if err := s.serviceDB.DeleteClientProject(projectID); err != nil {
		LogError(r.Context(), err, "Failed to delete project", "project_id", projectID)
		s.handleHTTPError(w, r, NewInternalError("не удалось удалить проект", err))
		return
	}

	LogInfo(r.Context(), "Project deleted successfully", "project_id", projectID, "client_id", clientID)
	s.writeJSONResponse(w, r, map[string]string{"message": "Project deleted"}, http.StatusOK)
}

// handleGetClientBenchmarks получает эталоны проекта
func (s *Server) handleGetClientBenchmarks(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for get benchmarks", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (get benchmarks)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	category := r.URL.Query().Get("category")
	approvedOnly := r.URL.Query().Get("approved_only") == "true"

	log.Printf("[Benchmarks] Getting benchmarks for project %d, client %d, category: %s, approvedOnly: %v",
		projectID, clientID, category, approvedOnly)

	benchmarks, err := s.serviceDB.GetClientBenchmarks(projectID, category, approvedOnly)
	if err != nil {
		log.Printf("[Benchmarks] Error getting benchmarks: %v", err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[Benchmarks] Found %d benchmarks for project %d", len(benchmarks), projectID)

	responseBenchmarks := make([]ClientBenchmark, len(benchmarks))
	for i, b := range benchmarks {
		responseBenchmarks[i] = ClientBenchmark{
			ID:              b.ID,
			ClientProjectID: b.ClientProjectID,
			OriginalName:    b.OriginalName,
			NormalizedName:  b.NormalizedName,
			Category:        b.Category,
			Subcategory:     b.Subcategory,
			Attributes:      b.Attributes,
			QualityScore:    b.QualityScore,
			IsApproved:      b.IsApproved,
			ApprovedBy:      b.ApprovedBy,
			ApprovedAt:      b.ApprovedAt,
			SourceDatabase:  b.SourceDatabase,
			UsageCount:      b.UsageCount,
			CreatedAt:       b.CreatedAt,
			UpdatedAt:       b.UpdatedAt,
		}
	}

	log.Printf("[Benchmarks] Returning %d benchmarks to client", len(responseBenchmarks))

	s.writeJSONResponse(w, r, map[string]interface{}{
		"benchmarks": responseBenchmarks,
		"total":      len(responseBenchmarks),
	}, http.StatusOK)
}

// handleCreateClientBenchmark создает эталон
func (s *Server) handleCreateClientBenchmark(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for create benchmark", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (create benchmark)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

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
		LogError(r.Context(), err, "Failed to decode request body for create benchmark", "project_id", projectID)
		s.handleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.OriginalName == "" || req.NormalizedName == "" || req.Category == "" {
		LogWarn(r.Context(), "Missing required fields for create benchmark", "project_id", projectID, "original_name", req.OriginalName, "normalized_name", req.NormalizedName, "category", req.Category)
		s.handleHTTPError(w, r, NewValidationError("поля 'original_name', 'normalized_name' и 'category' обязательны", nil))
		return
	}

	LogInfo(r.Context(), "Creating benchmark", "project_id", projectID, "original_name", req.OriginalName, "normalized_name", req.NormalizedName, "category", req.Category)
	benchmark, err := s.serviceDB.CreateClientBenchmark(projectID, req.OriginalName, req.NormalizedName, req.Category, req.Subcategory, req.Attributes, req.SourceDatabase, req.QualityScore)
	if err != nil {
		LogError(r.Context(), err, "Failed to create benchmark", "project_id", projectID)
		s.handleHTTPError(w, r, NewInternalError("не удалось создать эталон", err))
		return
	}

	LogInfo(r.Context(), "Benchmark created successfully", "benchmark_id", benchmark.ID, "project_id", projectID)

	response := ClientBenchmark{
		ID:              benchmark.ID,
		ClientProjectID: benchmark.ClientProjectID,
		OriginalName:    benchmark.OriginalName,
		NormalizedName:  benchmark.NormalizedName,
		Category:        benchmark.Category,
		Subcategory:     benchmark.Subcategory,
		Attributes:      benchmark.Attributes,
		QualityScore:    benchmark.QualityScore,
		IsApproved:      benchmark.IsApproved,
		ApprovedBy:      benchmark.ApprovedBy,
		ApprovedAt:      benchmark.ApprovedAt,
		SourceDatabase:  benchmark.SourceDatabase,
		UsageCount:      benchmark.UsageCount,
		CreatedAt:       benchmark.CreatedAt,
		UpdatedAt:       benchmark.UpdatedAt,
	}

	s.writeJSONResponse(w, r, response, http.StatusCreated)
}

// handleGetProjectDatabases получает список баз данных проекта
func (s *Server) handleGetProjectDatabases(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for get databases", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (get databases)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	activeOnly := r.URL.Query().Get("active_only") == "true"
	LogInfo(r.Context(), "Getting databases for project", "project_id", projectID, "client_id", clientID, "active_only", activeOnly)

	databases, err := s.serviceDB.GetProjectDatabases(projectID, activeOnly)
	if err != nil {
		LogError(r.Context(), err, "Failed to get databases for project", "project_id", projectID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить базы данных проекта", err))
		return
	}

	LogInfo(r.Context(), "Databases retrieved for project", "project_id", projectID, "count", len(databases))

	// Получаем статистику для всех баз данных одним batch-запросом (оптимизация N+1)
	var statsMap map[int]map[string]interface{}
	if s.db != nil && len(databases) > 0 {
		databaseIDs := make([]int, 0, len(databases))
		for _, db := range databases {
			databaseIDs = append(databaseIDs, db.ID)
		}

		var statsErr error
		statsMap, statsErr = s.db.GetUploadStatsByDatabaseIDs(databaseIDs)
		if statsErr != nil {
			// Игнорируем ошибки получения статистики - это не критично
			// Логируем предупреждение, но продолжаем работу без статистики
			s.logWarnf("Failed to get upload stats for project %d: %v", projectID, statsErr)
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

		// Добавляем статистику из batch-запроса
		if statsMap != nil {
			if stats, exists := statsMap[db.ID]; exists && stats != nil {
				dbInfo["stats"] = stats
			}
		}

		databasesWithStats = append(databasesWithStats, dbInfo)
	}

	s.writeJSONResponse(w, r, map[string]interface{}{
		"databases": databasesWithStats,
		"total":     len(databasesWithStats),
	}, http.StatusOK)
}

// handleCreateProjectDatabase создает новую базу данных для проекта
func (s *Server) handleCreateProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	var req struct {
		Name        string `json:"name"`
		FilePath    string `json:"file_path"`
		Description string `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		LogError(r.Context(), err, "Failed to decode request body for create database", "project_id", projectID)
		s.handleHTTPError(w, r, NewValidationError("неверный формат запроса", err))
		return
	}

	if req.Name == "" || req.FilePath == "" {
		LogWarn(r.Context(), "Missing required fields for create database", "project_id", projectID, "name", req.Name, "file_path", req.FilePath)
		s.handleHTTPError(w, r, NewValidationError("поля 'name' и 'file_path' обязательны", nil))
		return
	}

	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for create database", "project_id", projectID, "client_id", clientID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (create database)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	LogInfo(r.Context(), "Creating database for project", "project_id", projectID, "name", req.Name, "file_path", req.FilePath)

	// Проверяем, не существует ли уже база данных с таким именем для этого проекта
	existingDatabases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err == nil {
		for _, existingDB := range existingDatabases {
			// Проверяем по имени или по пути к файлу
			if existingDB.Name == req.Name || (req.FilePath != "" && existingDB.FilePath == req.FilePath) {
				// Возвращаем существующую базу данных вместо создания дубликата
				s.writeJSONResponse(w, r, existingDB, http.StatusOK)
				return
			}
		}
	}

	// Проверяем существование файла
	fileInfo, err := os.Stat(req.FilePath)
	if err != nil {
		LogError(r.Context(), err, "File not found for create database", "project_id", projectID, "file_path", req.FilePath)
		s.handleHTTPError(w, r, NewValidationError(fmt.Sprintf("Файл не найден: %s", req.FilePath), err))
		return
	}

	fileSize := fileInfo.Size()
	finalPath := req.FilePath

	// Если файл не в data/uploads/, перемещаем его туда
	uploadsDir, err := EnsureUploadsDirectory(".")
	if err == nil {
		// Проверяем, находится ли файл уже в uploads
		absFilePath, _ := filepath.Abs(req.FilePath)
		absUploadsDir, _ := filepath.Abs(uploadsDir)

		if !strings.HasPrefix(absFilePath, absUploadsDir) {
			// Перемещаем файл
			newPath, moveErr := MoveDatabaseToUploads(req.FilePath, uploadsDir)
			if moveErr != nil {
				LogError(r.Context(), moveErr, "Failed to move file to uploads", "project_id", projectID, "file_path", req.FilePath)
				s.handleHTTPError(w, r, NewInternalError("не удалось переместить файл в uploads", moveErr))
				return
			}
			finalPath = newPath
			LogInfo(r.Context(), "File moved to uploads", "old_path", req.FilePath, "new_path", finalPath)
		}
	}

	database, err := s.serviceDB.CreateProjectDatabase(projectID, req.Name, finalPath, req.Description, fileSize)
	if err != nil {
		log.Printf("Error creating database for project_id=%d: %v", projectID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Database created successfully: database_id=%d, project_id=%d, name=%q", database.ID, projectID, database.Name)

	// Создаем или обновляем upload записи в исходной базе данных
	// Это необходимо для того, чтобы getNomenclatureFromMainDB мог найти данные
	// Выполняем синхронно, чтобы данные были доступны сразу после добавления БД
	if err := s.ensureUploadRecordsForDatabase(finalPath, clientID, projectID, database.ID); err != nil {
		log.Printf("Warning: Failed to ensure upload records for database %d: %v (database was still created)", database.ID, err)
		// Не возвращаем ошибку, так как база данных уже создана
		// Upload записи можно будет создать позже
	}

	s.writeJSONResponse(w, r, database, http.StatusCreated)
}

// handleGetProjectDatabase получает базу данных проекта
func (s *Server) handleGetProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for get database (client_id=%d, project_id=%d, db_id=%d)", clientID, projectID, dbID)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		LogError(r.Context(), err, "Project not found for get database", "project_id", projectID, "client_id", clientID, "db_id", dbID)
		s.handleHTTPError(w, r, NewNotFoundError("Проект не найден", err))
		return
	}

	if project.ClientID != clientID {
		LogWarn(r.Context(), "Project does not belong to client (get database)", "project_id", projectID, "project_client_id", project.ClientID, "requested_client_id", clientID, "db_id", dbID)
		s.handleHTTPError(w, r, NewValidationError("Проект не принадлежит данному клиенту", nil))
		return
	}

	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		LogError(r.Context(), err, "Failed to get database", "db_id", dbID)
		s.handleHTTPError(w, r, NewInternalError("не удалось получить базу данных", err))
		return
	}

	if database == nil {
		LogWarn(r.Context(), "Database not found", "db_id", dbID)
		s.handleHTTPError(w, r, NewNotFoundError("База данных не найдена", nil))
		return
	}

	if database.ClientProjectID != projectID {
		log.Printf("Database %d does not belong to project %d", dbID, projectID)
		s.writeJSONError(w, r, "Database does not belong to this project", http.StatusBadRequest)
		return
	}

	log.Printf("Retrieved database %d for project %d", dbID, projectID)
	s.writeJSONResponse(w, r, database, http.StatusOK)
}

// handleUpdateProjectDatabase обновляет базу данных проекта
func (s *Server) handleUpdateProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for update database (client_id=%d, project_id=%d, db_id=%d)", clientID, projectID, dbID)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	var req struct {
		Name        string `json:"name"`
		FilePath    string `json:"file_path"`
		Description string `json:"description"`
		IsActive    bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body for update database %d: %v", dbID, err)
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for update database (client_id=%d, db_id=%d): %v", projectID, clientID, dbID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (update database %d)", projectID, clientID, dbID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		log.Printf("Error getting database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	if database == nil {
		log.Printf("Database %d not found", dbID)
		s.writeJSONError(w, r, "Database not found", http.StatusNotFound)
		return
	}

	if database.ClientProjectID != projectID {
		log.Printf("Database %d does not belong to project %d", dbID, projectID)
		s.writeJSONError(w, r, "Database does not belong to this project", http.StatusBadRequest)
		return
	}

	log.Printf("Updating database %d: name=%q, is_active=%v", dbID, req.Name, req.IsActive)
	err = s.serviceDB.UpdateProjectDatabase(dbID, req.Name, req.FilePath, req.Description, req.IsActive)
	if err != nil {
		log.Printf("Error updating database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	updatedDatabase, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		log.Printf("Error getting updated database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Database %d updated successfully", dbID)
	s.writeJSONResponse(w, r, updatedDatabase, http.StatusOK)
}

// handleDeleteProjectDatabase удаляет базу данных проекта
func (s *Server) handleDeleteProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for delete database (client_id=%d, project_id=%d, db_id=%d)", clientID, projectID, dbID)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for delete database (client_id=%d, db_id=%d): %v", projectID, clientID, dbID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (delete database %d)", projectID, clientID, dbID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		log.Printf("Error getting database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	if database == nil {
		log.Printf("Database %d not found", dbID)
		s.writeJSONError(w, r, "Database not found", http.StatusNotFound)
		return
	}

	if database.ClientProjectID != projectID {
		log.Printf("Database %d does not belong to project %d", dbID, projectID)
		s.writeJSONError(w, r, "Database does not belong to this project", http.StatusBadRequest)
		return
	}

	log.Printf("Deleting database %d from project %d", dbID, projectID)

	// Сохраняем путь к файлу перед удалением записи
	filePath := database.FilePath

	// Удаляем запись из БД
	err = s.serviceDB.DeleteProjectDatabase(dbID)
	if err != nil {
		log.Printf("Error deleting database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	// Удаляем физический файл, если он существует
	fileDeleted := false
	if filePath != "" {
		absPath, err := filepath.Abs(filePath)
		if err == nil {
			if _, err := os.Stat(absPath); err == nil {
				// Проверяем, что файл не защищен
				fileName := filepath.Base(absPath)
				protectedFiles := map[string]bool{
					"service.db":         true,
					"1c_data.db":         true,
					"data.db":            true,
					"normalized_data.db": true,
				}

				if !protectedFiles[fileName] {
					if err := os.Remove(absPath); err == nil {
						fileDeleted = true
						log.Printf("Deleted physical file: %s", absPath)
					} else {
						log.Printf("Warning: Failed to delete physical file %s: %v", absPath, err)
					}
				} else {
					log.Printf("Skipped deletion of protected file: %s", fileName)
				}
			}
		}
	}

	log.Printf("Database %d deleted successfully (file deleted: %v)", dbID, fileDeleted)
	response := map[string]interface{}{
		"message":      "Database deleted successfully",
		"file_deleted": fileDeleted,
	}
	if filePath != "" {
		response["file_path"] = filePath
	}
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// isValidTableName проверяет, что имя таблицы безопасно (защита от SQL-инъекций)
func isValidTableName(name string) bool {
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

// buildSafeTableQuery создает безопасный SQL запрос с именем таблицы
// ВАЖНО: tableName ДОЛЖЕН быть валидирован через isValidTableName перед вызовом этой функции
func buildSafeTableQuery(queryTemplate string, tableName string) string {
	// Дополнительная проверка на всякий случай
	if !isValidTableName(tableName) {
		// В production это должно логироваться как критическая ошибка
		log.Printf("SECURITY WARNING: Attempted to build query with invalid table name: %s", tableName)
		return ""
	}
	return fmt.Sprintf(queryTemplate, tableName)
}

// handleGetProjectDatabaseTables получает список таблиц базы данных проекта
func (s *Server) handleGetProjectDatabaseTables(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int) {
	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for get tables (client_id=%d, project_id=%d, db_id=%d)", clientID, projectID, dbID)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for get tables (client_id=%d, db_id=%d): %v", projectID, clientID, dbID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (get tables, db_id=%d)", projectID, clientID, dbID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Получаем информацию о базе данных
	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		log.Printf("Error getting database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	if database == nil {
		log.Printf("Database %d not found", dbID)
		s.writeJSONError(w, r, "Database not found", http.StatusNotFound)
		return
	}

	if database.ClientProjectID != projectID {
		log.Printf("Database %d does not belong to project %d", dbID, projectID)
		s.writeJSONError(w, r, "Database does not belong to this project", http.StatusBadRequest)
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(database.FilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeJSONError(w, r, fmt.Sprintf("Database file not found: %s", database.FilePath), http.StatusNotFound)
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Error checking database file: %v", err), http.StatusInternalServerError)
		return
	}

	// Открываем базу данных проекта
	conn, err := sql.Open("sqlite3", database.FilePath)
	if err != nil {
		s.logError(fmt.Sprintf("Error opening database %s: %v", database.FilePath, err), r.URL.Path)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Получаем список таблиц
	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		log.Printf("Error querying tables: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to query tables: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type ColumnInfo struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		Nullable bool   `json:"nullable"`
		Default  string `json:"default"`
	}

	type TableInfo struct {
		Name         string       `json:"name"`
		RecordsCount int          `json:"records_count"`
		Columns      []ColumnInfo `json:"columns"`
	}

	tables := []TableInfo{}
	totalRecords := 0

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Валидация имени таблицы из БД (защита от потенциально скомпрометированных данных)
		if !isValidTableName(tableName) {
			log.Printf("Warning: Invalid table name from database: %s, skipping", tableName)
			continue
		}

		tableInfo := TableInfo{Name: tableName}

		// Получаем количество записей (безопасный запрос)
		var count int
		query := buildSafeTableQuery("SELECT COUNT(*) FROM %s", tableName)
		if query == "" {
			log.Printf("Error: Failed to build safe query for table %s", tableName)
			count = 0
		} else {
			err := conn.QueryRow(query).Scan(&count)
			if err != nil {
				log.Printf("Error counting records in table %s: %v", tableName, err)
				count = 0
			}
		}
		tableInfo.RecordsCount = count
		totalRecords += count

		// Получаем информацию о колонках (безопасный запрос)
		pragmaQuery := buildSafeTableQuery("PRAGMA table_info(%s)", tableName)
		if pragmaQuery == "" {
			log.Printf("Error: Failed to build safe PRAGMA query for table %s", tableName)
			continue
		}
		colRows, err := conn.Query(pragmaQuery)
		if err == nil {
			columns := []ColumnInfo{}
			for colRows.Next() {
				var col ColumnInfo
				var cid int
				var notNull int
				var pk int
				var dfltValue sql.NullString

				if err := colRows.Scan(&cid, &col.Name, &col.Type, &notNull, &dfltValue, &pk); err == nil {
					col.Nullable = (notNull == 0)
					if dfltValue.Valid {
						col.Default = dfltValue.String
					}
					columns = append(columns, col)
				}
			}
			colRows.Close()
			tableInfo.Columns = columns
		}

		tables = append(tables, tableInfo)
	}

	// Формируем ответ с информацией о базе данных и таблицах
	response := map[string]interface{}{
		"database": map[string]interface{}{
			"id":            database.ID,
			"name":          database.Name,
			"path":          database.FilePath,
			"size":          database.FileSize,
			"tables_count":  len(tables),
			"total_records": totalRecords,
			"created_at":    database.CreatedAt.Format(time.RFC3339),
		},
		"tables": tables,
	}

	log.Printf("Retrieved %d tables from database %d (project_id=%d, total_records=%d)", len(tables), dbID, projectID, totalRecords)
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetProjectDatabaseTableData получает данные из таблицы базы данных проекта с пагинацией
func (s *Server) handleGetProjectDatabaseTableData(w http.ResponseWriter, r *http.Request, clientID, projectID, dbID int, tableName string) {
	if s.serviceDB == nil {
		log.Printf("Error: Service database not available for get table data (client_id=%d, project_id=%d, db_id=%d, table=%s)", clientID, projectID, dbID, tableName)
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	// Валидация имени таблицы
	if !isValidTableName(tableName) {
		log.Printf("Invalid table name: %s (client_id=%d, project_id=%d, db_id=%d)", tableName, clientID, projectID, dbID)
		s.writeJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}

	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for get table data (client_id=%d, db_id=%d, table=%s): %v", projectID, clientID, dbID, tableName, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (get table data, db_id=%d, table=%s)", projectID, clientID, dbID, tableName)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Получаем информацию о базе данных
	database, err := s.serviceDB.GetProjectDatabase(dbID)
	if err != nil {
		log.Printf("Error getting database %d: %v", dbID, err)
		s.writeJSONError(w, r, err.Error(), http.StatusInternalServerError)
		return
	}

	if database == nil {
		log.Printf("Database %d not found", dbID)
		s.writeJSONError(w, r, "Database not found", http.StatusNotFound)
		return
	}

	if database.ClientProjectID != projectID {
		log.Printf("Database %d does not belong to project %d", dbID, projectID)
		s.writeJSONError(w, r, "Database does not belong to this project", http.StatusBadRequest)
		return
	}

	// Проверяем существование файла
	if _, err := os.Stat(database.FilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.writeJSONError(w, r, fmt.Sprintf("Database file not found: %s", database.FilePath), http.StatusNotFound)
			return
		}
		s.writeJSONError(w, r, fmt.Sprintf("Error checking database file: %v", err), http.StatusInternalServerError)
		return
	}

	// Параметры пагинации с валидацией
	page, err := ValidateIntParam(r, "page", 1, 1, 0)
	if err != nil {
		page = 1 // Используем значение по умолчанию при ошибке
	}

	pageSize, err := ValidateIntParam(r, "pageSize", 50, 1, 100)
	if err != nil {
		pageSize = 50 // Используем значение по умолчанию при ошибке
	}

	offset := (page - 1) * pageSize

	// Открываем базу данных проекта
	conn, err := sql.Open("sqlite3", database.FilePath)
	if err != nil {
		log.Printf("Error opening database %s: %v", database.FilePath, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Проверяем существование таблицы
	var tableExists bool
	err = conn.QueryRow(`
		SELECT COUNT(*) > 0 
		FROM sqlite_master 
		WHERE type='table' AND name=?
	`, tableName).Scan(&tableExists)
	if err != nil || !tableExists {
		s.writeJSONError(w, r, "Table not found", http.StatusNotFound)
		return
	}

	// Получаем общее количество записей (безопасный запрос)
	var total int
	countQuery := buildSafeTableQuery("SELECT COUNT(*) FROM %s", tableName)
	if countQuery == "" {
		s.writeJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}
	err = conn.QueryRow(countQuery).Scan(&total)
	if err != nil {
		s.logError(fmt.Sprintf("Error counting records in table %s: %v", tableName, err), r.URL.Path)
		total = 0
	}

	// Получаем информацию о колонках (безопасный запрос)
	pragmaQuery := buildSafeTableQuery("PRAGMA table_info(%s)", tableName)
	if pragmaQuery == "" {
		s.writeJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}
	colRows, err := conn.Query(pragmaQuery)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get table structure: %v", err), http.StatusInternalServerError)
		return
	}
	defer colRows.Close()

	columns := []string{}
	for colRows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := colRows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err == nil {
			columns = append(columns, name)
		}
	}

	if len(columns) == 0 {
		s.writeJSONError(w, r, "Table has no columns", http.StatusBadRequest)
		return
	}

	// Получаем данные с пагинацией (безопасный запрос)
	selectQuery := buildSafeTableQuery("SELECT * FROM %s LIMIT ? OFFSET ?", tableName)
	if selectQuery == "" {
		s.writeJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}
	rows, err := conn.Query(selectQuery, pageSize, offset)
	if err != nil {
		log.Printf("Error querying table data: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to query table data: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Читаем данные
	var data []map[string]interface{}
	for rows.Next() {
		// Создаем срез для значений
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Преобразуем в map
		rowData := make(map[string]interface{})
		for i, colName := range columns {
			val := values[i]
			// Преобразуем []byte в string для читаемости
			if b, ok := val.([]byte); ok {
				rowData[colName] = string(b)
			} else if val == nil {
				rowData[colName] = nil
			} else {
				rowData[colName] = val
			}
		}
		data = append(data, rowData)
	}

	totalPages := (total + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}

	response := map[string]interface{}{
		"table_name":  tableName,
		"columns":     columns,
		"data":        data,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
	}

	log.Printf("Retrieved table data from %s (db_id=%d, project_id=%d, page=%d, page_size=%d, total=%d, returned=%d)", tableName, dbID, projectID, page, pageSize, total, len(data))
	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleUploadProjectDatabase обрабатывает загрузку файла базы данных через multipart/form-data
// Enterprise-level security: валидация, санитизация, проверка прав доступа, защита от path traversal
func (s *Server) handleUploadProjectDatabase(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	requestID := r.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = fmt.Sprintf("req_%d_%d", time.Now().Unix(), projectID)
	}
	clientIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		clientIP = strings.Split(forwarded, ",")[0]
	}

	log.Printf("[handleUploadProjectDatabase] [%s] Начало обработки загрузки файла для проекта %d клиента %d (IP: %s)", requestID, projectID, clientID, clientIP)

	// Проверяем Content-Type (может быть с boundary параметром)
	contentType := r.Header.Get("Content-Type")
	log.Printf("[handleUploadProjectDatabase] [%s] Content-Type: %s", requestID, contentType)

	// Проверяем, что это multipart/form-data (может быть с boundary)
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		log.Printf("[handleUploadProjectDatabase] Ошибка: неверный Content-Type, ожидается multipart/form-data, получен: %s", contentType)
		s.writeJSONError(w, r, fmt.Sprintf("Неверный Content-Type: ожидается multipart/form-data, получен: %s", contentType), http.StatusBadRequest)
		return
	}

	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка получения проекта %d: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("[handleUploadProjectDatabase] Ошибка: проект %d не принадлежит клиенту %d (принадлежит клиенту %d)", projectID, clientID, project.ClientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Проверяем Content-Length перед парсингом
	contentLengthStr := r.Header.Get("Content-Length")
	var contentLength int64
	if contentLengthStr != "" {
		if parsedLen, err := strconv.ParseInt(contentLengthStr, 10, 64); err == nil {
			contentLength = parsedLen
			log.Printf("[handleUploadProjectDatabase] [%s] Content-Length: %d байт (~%.2f MB)", requestID, contentLength, float64(contentLength)/(1024*1024))
		} else {
			log.Printf("[handleUploadProjectDatabase] [%s] Предупреждение: не удалось распарсить Content-Length: %s", requestID, contentLengthStr)
		}
	} else {
		log.Printf("[handleUploadProjectDatabase] [%s] Предупреждение: Content-Length не указан", requestID)
	}

	// Проверяем размер файла перед парсингом
	maxSize := int64(500 << 20) // 500MB
	if contentLength > 0 && contentLength > maxSize {
		log.Printf("[handleUploadProjectDatabase] [%s] Ошибка: файл слишком большой (%d байт, максимум %d байт)", requestID, contentLength, maxSize)
		s.writeJSONError(w, r, fmt.Sprintf("Файл слишком большой: %d байт (максимум %d байт)", contentLength, maxSize), http.StatusRequestEntityTooLarge)
		return
	}

	// Парсим multipart форму (максимальный размер 500MB для баз данных)
	log.Printf("[handleUploadProjectDatabase] [%s] Начало парсинга multipart формы (макс. размер: 500MB, ожидаемый размер: %d байт)", requestID, contentLength)
	err = r.ParseMultipartForm(500 << 20)
	if err != nil {
		// Детальная диагностика ошибки
		log.Printf("[handleUploadProjectDatabase] [%s] ❌ ОШИБКА парсинга multipart формы: %v", requestID, err)
		log.Printf("[handleUploadProjectDatabase] [%s] Детали: Content-Type=%s, Content-Length=%s, Method=%s, URL=%s",
			requestID, contentType, contentLengthStr, r.Method, r.URL.String())

		// Специальная обработка для "Unexpected end of multipart data"
		if strings.Contains(err.Error(), "Unexpected end") || strings.Contains(err.Error(), "multipart data") {
			log.Printf("[handleUploadProjectDatabase] [%s] Обнаружена ошибка 'Unexpected end of multipart data'", requestID)
			log.Printf("[handleUploadProjectDatabase] [%s] Возможные причины:", requestID)
			log.Printf("[handleUploadProjectDatabase] [%s]   1. Соединение прервано во время передачи", requestID)
			log.Printf("[handleUploadProjectDatabase] [%s]   2. Таймаут при передаче большого файла", requestID)
			log.Printf("[handleUploadProjectDatabase] [%s]   3. Проблема с проксированием через Next.js", requestID)
			log.Printf("[handleUploadProjectDatabase] [%s]   4. Файл не полностью передан", requestID)

			s.writeJSONError(w, r, fmt.Sprintf("Ошибка передачи файла: соединение прервано или файл не полностью передан. Попробуйте загрузить файл снова или уменьшите размер файла. Детали: %v", err), http.StatusBadRequest)
		} else {
			s.writeJSONError(w, r, fmt.Sprintf("Ошибка парсинга формы: %v. Проверьте, что файл отправлен правильно.", err), http.StatusBadRequest)
		}
		return
	}
	log.Printf("[handleUploadProjectDatabase] [%s] ✅ Multipart форма успешно распарсена", requestID)

	// Получаем файл из формы
	log.Printf("[handleUploadProjectDatabase] Попытка получить файл из формы")
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка получения файла из формы: %v. Доступные поля формы: %v", err, r.MultipartForm.Value)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка получения файла: %v. Убедитесь, что файл отправлен с полем 'file'.", err), http.StatusBadRequest)
		return
	}
	defer file.Close()
	log.Printf("[handleUploadProjectDatabase] Файл получен: %s (размер: %d байт)", header.Filename, header.Size)
	log.Printf("[handleUploadProjectDatabase] Информация об имени файла: длина=%d, байт=%d, первые 50 символов=%s",
		len(header.Filename), len([]byte(header.Filename)), header.Filename[:min(50, len(header.Filename))])

	// Проверяем расширение файла
	fileName := header.Filename
	if !strings.HasSuffix(strings.ToLower(fileName), ".db") {
		log.Printf("[handleUploadProjectDatabase] Ошибка: файл не имеет расширение .db: %s", fileName)
		s.writeJSONError(w, r, fmt.Sprintf("Файл должен иметь расширение .db. Получен файл: %s", fileName), http.StatusBadRequest)
		return
	}

	// Очищаем имя файла от потенциально опасных символов
	// Заменяем недопустимые символы в имени файла
	fileName = strings.ReplaceAll(fileName, "..", "_")
	fileName = strings.ReplaceAll(fileName, "/", "_")
	fileName = strings.ReplaceAll(fileName, "\\", "_")
	fileName = strings.ReplaceAll(fileName, ":", "_")
	fileName = strings.ReplaceAll(fileName, "*", "_")
	fileName = strings.ReplaceAll(fileName, "?", "_")
	fileName = strings.ReplaceAll(fileName, "\"", "_")
	fileName = strings.ReplaceAll(fileName, "<", "_")
	fileName = strings.ReplaceAll(fileName, ">", "_")
	fileName = strings.ReplaceAll(fileName, "|", "_")

	// Ограничиваем длину имени файла (максимум 255 символов)
	if len(fileName) > 255 {
		ext := filepath.Ext(fileName)
		nameWithoutExt := strings.TrimSuffix(fileName, ext)
		if len(nameWithoutExt) > 250 {
			nameWithoutExt = nameWithoutExt[:250]
		}
		fileName = nameWithoutExt + ext
	}

	log.Printf("Received database file: %s (size: %d bytes, cleaned filename: %s)", header.Filename, header.Size, fileName)

	// Создаем папку uploads, если её нет
	uploadStartTime := time.Now() // Время начала загрузки
	uploadsDir, err := EnsureUploadsDirectory(".")
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка создания папки uploads: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка создания папки uploads: %v", err), http.StatusInternalServerError)
		return
	}
	log.Printf("[handleUploadProjectDatabase] Папка uploads проверена/создана: %s", uploadsDir)

	// Сохраняем файл в data/uploads/
	// Используем filepath.Clean для защиты от path traversal
	filePath := filepath.Join(uploadsDir, fileName)
	filePath = filepath.Clean(filePath)

	// Дополнительная проверка на path traversal - убеждаемся, что путь находится внутри uploadsDir
	absUploadsDir, err := filepath.Abs(uploadsDir)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка получения абсолютного пути uploadsDir: %v", err)
		s.writeJSONError(w, r, "Internal server error", http.StatusInternalServerError)
		return
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка получения абсолютного пути filePath: %v", err)
		s.writeJSONError(w, r, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Проверяем, что файл находится внутри uploadsDir
	if !strings.HasPrefix(absFilePath, absUploadsDir+string(filepath.Separator)) && absFilePath != absUploadsDir {
		log.Printf("[handleUploadProjectDatabase] Ошибка безопасности: попытка path traversal. uploadsDir: %s, filePath: %s", absUploadsDir, absFilePath)
		s.writeJSONError(w, r, "Invalid file path", http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли файл с таким именем
	if _, err := os.Stat(filePath); err == nil {
		// Файл уже существует, добавляем timestamp к имени
		ext := filepath.Ext(fileName)
		nameWithoutExt := strings.TrimSuffix(fileName, ext)
		timestamp := time.Now().Format("20060102_150405")
		oldFileName := fileName
		fileName = fmt.Sprintf("%s_%s%s", nameWithoutExt, timestamp, ext)
		filePath = filepath.Join(uploadsDir, fileName)
		filePath = filepath.Clean(filePath) // Нормализуем путь после переименования
		log.Printf("[handleUploadProjectDatabase] Файл с именем '%s' уже существует, переименован в '%s'", oldFileName, fileName)
	}

	// Создаем файл для сохранения
	log.Printf("[handleUploadProjectDatabase] Создание файла для сохранения: %s", filePath)
	dst, err := os.Create(filePath)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка создания файла %s: %v", filePath, err)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка создания файла: %v", err), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Копируем содержимое загруженного файла
	copyStartTime := time.Now()
	log.Printf("[handleUploadProjectDatabase] Начало копирования файла (размер: %d байт, ~%.2f MB)", header.Size, float64(header.Size)/(1024*1024))
	bytesWritten, err := io.Copy(dst, file)
	copyDuration := time.Since(copyStartTime)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка копирования файла: %v (записано %d из %d байт)", err, bytesWritten, header.Size)
		os.Remove(filePath) // Удаляем файл при ошибке
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка сохранения файла: %v", err), http.StatusInternalServerError)
		return
	}

	// Вычисляем скорость загрузки (избегаем деления на ноль и Inf)
	var speedMBps float64
	nanos := float64(copyDuration.Nanoseconds())
	if nanos > 0 {
		// Используем наносекунды для более точного расчета
		speedMBps = float64(bytesWritten) / (1024 * 1024) / (nanos / 1e9)
		// Проверяем на разумные значения (не Inf и не NaN)
		if !(speedMBps >= 0 && speedMBps < 1e10) {
			// Если скорость неразумно высокая, показываем "очень быстро"
			speedMBps = -1 // Специальное значение для "очень быстро"
		}
	} else {
		speedMBps = 0 // Если время равно 0, скорость не определена
	}

	// Форматируем скорость для лога
	var speedStr string
	if speedMBps < 0 {
		speedStr = "очень быстро"
	} else if speedMBps > 0 {
		speedStr = fmt.Sprintf("%.2f MB/s", speedMBps)
	} else {
		speedStr = "N/A"
	}

	log.Printf("[handleUploadProjectDatabase] Файл сохранен: %s (записано %d байт, ~%.2f MB, время: %v, скорость: %s)",
		filePath, bytesWritten, float64(bytesWritten)/(1024*1024), copyDuration, speedStr)

	// Проверяем, что файл не пустой
	if bytesWritten == 0 {
		log.Printf("[handleUploadProjectDatabase] Ошибка: файл пустой")
		os.Remove(filePath)
		s.writeJSONError(w, r, "Загруженный файл пустой", http.StatusBadRequest)
		return
	}

	// Проверяем, что файл является валидным SQLite файлом
	// SQLite файлы начинаются с заголовка "SQLite format 3\000"
	dst.Close() // Закрываем файл перед проверкой
	validationFile, err := os.Open(filePath)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка открытия файла для валидации: %v", err)
		os.Remove(filePath)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка валидации файла: %v", err), http.StatusInternalServerError)
		return
	}
	defer validationFile.Close()

	fileHeader := make([]byte, 16)
	n, err := validationFile.Read(fileHeader)
	if err != nil && err != io.EOF {
		log.Printf("[handleUploadProjectDatabase] Ошибка чтения заголовка файла: %v", err)
		os.Remove(filePath)
		s.writeJSONError(w, r, fmt.Sprintf("Ошибка чтения файла: %v", err), http.StatusInternalServerError)
		return
	}

	if n < 16 {
		log.Printf("[handleUploadProjectDatabase] Ошибка: файл слишком маленький для SQLite базы данных")
		os.Remove(filePath)
		s.writeJSONError(w, r, "Файл слишком маленький. Минимальный размер SQLite файла - 16 байт", http.StatusBadRequest)
		return
	}

	// Проверяем SQLite заголовок
	sqliteHeader := "SQLite format 3\x00"
	if string(fileHeader) != sqliteHeader {
		log.Printf("[handleUploadProjectDatabase] Ошибка: файл не является валидным SQLite файлом. Заголовок: %q (ожидается: %q)", string(fileHeader), sqliteHeader)
		os.Remove(filePath)
		s.writeJSONError(w, r, fmt.Sprintf("Файл не является валидным SQLite файлом. Заголовок файла: %q. Убедитесь, что вы загружаете файл базы данных SQLite.", string(fileHeader)), http.StatusBadRequest)
		return
	}

	// Дополнительная проверка безопасности: проверяем, что файл не является исполняемым
	// Это защита от загрузки вредоносных файлов под видом SQLite
	executableSignatures := [][]byte{
		[]byte{0x7F, 0x45, 0x4C, 0x46}, // ELF (Linux executable)
		[]byte{0x4D, 0x5A},             // PE/COFF (Windows executable)
		[]byte{0xCA, 0xFE, 0xBA, 0xBE}, // Java class file
		[]byte{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O (macOS executable)
		[]byte{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O (macOS executable, little-endian)
	}

	for _, sig := range executableSignatures {
		if len(fileHeader) >= len(sig) {
			match := true
			for i := 0; i < len(sig); i++ {
				if fileHeader[i] != sig[i] {
					match = false
					break
				}
			}
			if match {
				log.Printf("[handleUploadProjectDatabase] ОШИБКА БЕЗОПАСНОСТИ: файл похож на исполняемый (сигнатура: %x). Файл: %s, размер: %d байт", sig, fileName, header.Size)
				os.Remove(filePath)
				s.writeJSONError(w, r, "Файл не является валидным SQLite файлом. Обнаружена подозрительная сигнатура. Загрузка отклонена по соображениям безопасности.", http.StatusBadRequest)
				return
			}
		}
	}

	log.Printf("[handleUploadProjectDatabase] Файл успешно валидирован как SQLite база данных (размер: %d байт, ~%.2f MB)",
		bytesWritten, float64(bytesWritten)/(1024*1024))

	// Проверяем размер файла - должен совпадать с заявленным
	if header.Size > 0 && bytesWritten != header.Size {
		sizeDiff := bytesWritten - header.Size
		log.Printf("[handleUploadProjectDatabase] Предупреждение: размер записанного файла (%d байт) не совпадает с заявленным (%d байт), разница: %d байт",
			bytesWritten, header.Size, sizeDiff)
		// Это не критично, но стоит залогировать
	}

	// Дополнительная проверка целостности SQLite файла через sql.Open и Ping
	log.Printf("[handleUploadProjectDatabase] Проверка целостности SQLite файла...")
	if err := ValidateSQLiteDatabase(filePath); err != nil {
		log.Printf("[handleUploadProjectDatabase] Ошибка проверки целостности: %v", err)
		os.Remove(filePath)
		s.writeJSONError(w, r, fmt.Sprintf("Файл не является валидной SQLite базой данных или поврежден: %v", err), http.StatusBadRequest)
		return
	}
	log.Printf("[handleUploadProjectDatabase] ✅ Файл успешно прошел проверку целостности")

	// Проверяем лимит на количество баз данных в проекте
	const MAX_DATABASES_PER_PROJECT = 100
	dbCount, err := s.serviceDB.GetProjectDatabaseCount(projectID, false)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Предупреждение: не удалось получить количество баз данных: %v", err)
		// Продолжаем, так как это не критично
	} else if dbCount >= MAX_DATABASES_PER_PROJECT {
		log.Printf("[handleUploadProjectDatabase] Ошибка: превышен лимит баз данных в проекте (текущее количество: %d, лимит: %d)", dbCount, MAX_DATABASES_PER_PROJECT)
		os.Remove(filePath)
		s.writeJSONError(w, r, fmt.Sprintf("Превышен лимит баз данных в проекте. Максимальное количество: %d, текущее: %d", MAX_DATABASES_PER_PROJECT, dbCount), http.StatusBadRequest)
		return
	}

	// Парсим название из имени файла
	suggestedName := ParseDatabaseNameFromFilename(fileName)
	fileNameWithoutExt := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	nameRequired := false

	if suggestedName == "" || suggestedName == fileNameWithoutExt {
		// Если не удалось распарсить или имя совпадает с именем файла, требуется ввод от пользователя
		suggestedName = fileNameWithoutExt
		nameRequired = true
	}

	// Получаем описание из формы, если есть
	description := r.FormValue("description")
	if description == "" {
		description = fmt.Sprintf("Загружено: %s", time.Now().Format("02.01.2006 15:04"))
	}

	// Проверяем, не является ли файл дубликатом (по пути к файлу)
	existingDB, err := s.serviceDB.GetProjectDatabaseByFilePath(projectID, filePath)
	if err != nil {
		log.Printf("[handleUploadProjectDatabase] Предупреждение: не удалось проверить дубликаты: %v", err)
		// Продолжаем, так как это не критично
	} else if existingDB != nil {
		log.Printf("[handleUploadProjectDatabase] Обнаружен дубликат: база данных с таким же путем уже существует (ID=%d, путь=%s)", existingDB.ID, filePath)
		// Не удаляем файл, так как он может быть нужен
		// Возвращаем существующую базу данных
		s.writeJSONResponse(w, r, map[string]interface{}{
			"success":        true,
			"message":        "База данных с таким путем уже существует",
			"database":       existingDB,
			"file_path":      filePath,
			"file_name":      fileName,
			"suggested_name": suggestedName,
			"name_required":  nameRequired,
			"is_duplicate":   true,
		}, http.StatusOK)
		return
	}

	// Проверяем, нужно ли автоматически создать базу данных
	autoCreate := r.FormValue("auto_create") == "true"

	totalDuration := time.Since(uploadStartTime)
	log.Printf("[handleUploadProjectDatabase] Общее время обработки загрузки: %v", totalDuration)

	// Вычисляем скорость загрузки для ответа
	var uploadSpeedMBps float64
	if totalDuration.Seconds() > 0 {
		uploadSpeedMBps = float64(bytesWritten) / (1024 * 1024) / totalDuration.Seconds()
	} else {
		nanos := float64(totalDuration.Nanoseconds())
		if nanos > 0 {
			uploadSpeedMBps = float64(bytesWritten) / (1024 * 1024) / (nanos / 1e9)
		}
	}

	// Формируем метрики загрузки
	uploadMetrics := map[string]interface{}{
		"start_time":      uploadStartTime.Format(time.RFC3339),
		"duration_ms":     totalDuration.Milliseconds(),
		"duration_sec":    totalDuration.Seconds(),
		"speed_mbps":      uploadSpeedMBps,
		"file_size_bytes": bytesWritten,
		"file_size_mb":    float64(bytesWritten) / (1024 * 1024),
	}

	if autoCreate {
		// Автоматически создаем базу данных
		log.Printf("[handleUploadProjectDatabase] Автоматическое создание базы данных: название='%s', путь='%s'", suggestedName, filePath)
		database, err := s.serviceDB.CreateProjectDatabase(projectID, suggestedName, filePath, description, header.Size)
		if err != nil {
			log.Printf("[handleUploadProjectDatabase] Ошибка создания базы данных в БД: %v (проект=%d, название='%s')", err, projectID, suggestedName)
			// Удаляем загруженный файл при ошибке создания записи в БД
			if removeErr := os.Remove(filePath); removeErr != nil {
				log.Printf("[handleUploadProjectDatabase] Предупреждение: не удалось удалить файл после ошибки: %v (файл: %s)", removeErr, filePath)
			} else {
				log.Printf("[handleUploadProjectDatabase] Загруженный файл удален из-за ошибки создания записи в БД: %s", filePath)
			}
			s.writeJSONError(w, r, fmt.Sprintf("Ошибка создания базы данных: %v. Загруженный файл был удален.", err), http.StatusInternalServerError)
			return
		}
		log.Printf("[handleUploadProjectDatabase] База данных успешно создана: ID=%d, название='%s'", database.ID, database.Name)
		log.Printf("[handleUploadProjectDatabase] ✅ Успешно завершено: файл загружен и база данных создана (ID=%d, время: %v)",
			database.ID, time.Since(uploadStartTime))

		// База данных успешно создана, имя больше не требуется
		s.writeJSONResponse(w, r, map[string]interface{}{
			"success":        true,
			"message":        "База данных успешно загружена и добавлена",
			"database":       database,
			"file_path":      filePath,
			"file_name":      fileName,
			"suggested_name": suggestedName,
			"name_required":  false,
			"upload_metrics": uploadMetrics,
		}, http.StatusCreated)
	} else {
		// Возвращаем информацию о файле для подтверждения
		log.Printf("[handleUploadProjectDatabase] ✅ Успешно завершено: файл загружен, ожидается подтверждение (время: %v)", time.Since(uploadStartTime))
		s.writeJSONResponse(w, r, map[string]interface{}{
			"success":        true,
			"message":        "Файл успешно загружен",
			"file_path":      filePath,
			"file_name":      fileName,
			"suggested_name": suggestedName,
			"name_required":  nameRequired,
			"file_size":      header.Size,
			"description":    description,
			"upload_metrics": uploadMetrics,
		}, http.StatusOK)
	}
}

// startProjectNormalization запускает нормализацию для конкретного проекта (внутренний метод)
func (s *Server) startProjectNormalization(clientID, projectID int, options map[string]interface{}) error {
	if s.serviceDB == nil {
		return fmt.Errorf("service database not available")
	}

	s.normalizerMutex.Lock()
	if s.normalizerRunning {
		s.normalizerMutex.Unlock()
		return fmt.Errorf("normalization is already running")
	}

	// Извлекаем опции
	// По умолчанию all_active = true (обрабатываем все БД проекта)
	allActive := true
	if val, ok := options["all_active"].(bool); ok {
		allActive = val
	}
	databasePath := ""
	if val, ok := options["database_path"].(string); ok {
		databasePath = val
	}

	// Извлекаем список ID выбранных БД (если указан)
	var databaseIDs []int
	if val, ok := options["database_ids"]; ok {
		if ids, ok := val.([]interface{}); ok {
			for _, id := range ids {
				if idNum, ok := id.(float64); ok {
					databaseIDs = append(databaseIDs, int(idNum))
				} else if idNum, ok := id.(int); ok {
					databaseIDs = append(databaseIDs, idNum)
				}
			}
		} else if ids, ok := val.([]int); ok {
			databaseIDs = ids
		}
	}

	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.normalizerMutex.Unlock()
		return fmt.Errorf("project not found: %w", err)
	}

	if project.ClientID != clientID {
		s.normalizerMutex.Unlock()
		return fmt.Errorf("project does not belong to this client")
	}

	var databasesToProcess []*database.ProjectDatabase

	// Приоритет: database_ids > database_path > all_active (по умолчанию true)
	if len(databaseIDs) > 0 {
		// Используем выбранные БД по ID
		allDatabases, err := s.serviceDB.GetProjectDatabases(projectID, false)
		if err != nil {
			s.normalizerMutex.Unlock()
			return fmt.Errorf("failed to get project databases: %w", err)
		}

		// Создаем map для быстрого поиска
		dbMap := make(map[int]*database.ProjectDatabase)
		for _, db := range allDatabases {
			dbMap[db.ID] = db
		}

		// Фильтруем по выбранным ID
		invalidIDs := make([]int, 0)
		for _, id := range databaseIDs {
			if id <= 0 {
				invalidIDs = append(invalidIDs, id)
				continue
			}
			if db, exists := dbMap[id]; exists {
				// Проверяем, что БД активна
				if db.IsActive {
					databasesToProcess = append(databasesToProcess, db)
				} else {
					log.Printf("Database ID %d is not active, skipping", id)
				}
			} else {
				invalidIDs = append(invalidIDs, id)
			}
		}

		// Логируем невалидные ID
		if len(invalidIDs) > 0 {
			log.Printf("Warning: Invalid or not found database IDs: %v", invalidIDs)
		}

		if len(databasesToProcess) == 0 {
			s.normalizerMutex.Unlock()
			if len(invalidIDs) > 0 {
				return fmt.Errorf("no valid active databases found. Invalid IDs: %v", invalidIDs) // nolint:errorlint // not wrapping error, just formatting IDs
			}
			return fmt.Errorf("no valid databases found for selected IDs")
		}
	} else if allActive {
		// Получаем все активные БД проекта (по умолчанию)
		databases, err := s.serviceDB.GetProjectDatabases(projectID, true)
		if err != nil {
			s.normalizerMutex.Unlock()
			return fmt.Errorf("failed to get project databases: %w", err)
		}
		if len(databases) == 0 {
			s.normalizerMutex.Unlock()
			return fmt.Errorf("no active databases found for this project")
		}
		databasesToProcess = databases
	} else {
		// Используем конкретную БД по пути (только если явно указано all_active=false)
		if databasePath == "" {
			s.normalizerMutex.Unlock()
			return fmt.Errorf("database_path is required when all_active is false and database_ids is not provided")
		}

		// Получаем все БД проекта для проверки принадлежности
		allDatabases, err := s.serviceDB.GetProjectDatabases(projectID, false)
		if err != nil {
			s.normalizerMutex.Unlock()
			return fmt.Errorf("failed to get project databases: %w", err)
		}

		var foundDB *database.ProjectDatabase
		for _, db := range allDatabases {
			if pathsMatch(db.FilePath, databasePath) {
				foundDB = db
				break
			}
		}

		if foundDB == nil {
			s.normalizerMutex.Unlock()
			return fmt.Errorf("database does not belong to this project")
		}

		databasesToProcess = []*database.ProjectDatabase{foundDB}
	}

	// Создаем context для управления жизненным циклом нормализации
	// Отменяем предыдущий context, если он существует
	if s.normalizerCancel != nil {
		s.normalizerCancel()
	}
	s.normalizerCtx, s.normalizerCancel = context.WithCancel(context.Background())
	s.normalizerRunning = true
	s.normalizerMutex.Unlock()

	// Используем структурированное логирование
	normType := "nomenclature"
	if database.IsCounterpartyProjectType(project.ProjectType) {
		normType = "counterparty"
	}
	LogNormalizationStart(clientID, projectID, len(databasesToProcess), normType)

	// Запускаем нормализацию для всех выбранных БД в горутине
	go func() {
		startTime := time.Now()
		defer func() {
			// Обработка паники и очистка состояния
			if rec := recover(); rec != nil {
				LogNormalizationPanic(projectID, rec, string(debug.Stack()))
				// Уведомляем о критической ошибке
				select {
				case s.normalizerEvents <- fmt.Sprintf("Критическая ошибка нормализации: %v", rec):
				default:
				}
				// Отменяем контекст для остановки всех дочерних горутин
				s.normalizerMutex.Lock()
				if s.normalizerCancel != nil {
					s.normalizerCancel()
					s.normalizerCancel = nil
				}
				s.normalizerMutex.Unlock()
			}
			// Всегда сбрасываем флаг running при выходе
			s.normalizerMutex.Lock()
			s.normalizerRunning = false
			s.normalizerMutex.Unlock()
			LogNormalizationComplete(clientID, projectID, s.normalizerProcessed, s.normalizerSuccess, s.normalizerErrors, time.Since(startTime))
		}()

		// Для контрагентов и номенклатуры используем параллельную обработку БД
		if database.IsCounterpartyProjectType(project.ProjectType) {
			s.processCounterpartyDatabasesParallel(databasesToProcess, clientID, projectID)
		} else {
			// Для номенклатуры также используем параллельную обработку
			// Извлекаем параметры из options для передачи в processNomenclatureDatabasesParallel
			useKpved := false
			if val, ok := options["use_kpved"].(bool); ok {
				useKpved = val
			}
			useOkpd2 := false
			if val, ok := options["use_okpd2"].(bool); ok {
				useOkpd2 = val
			}

			req := struct {
				DatabasePath string `json:"database_path"`
				AllActive    bool   `json:"all_active"`
				UseKpved     bool   `json:"use_kpved"`
				UseOkpd2     bool   `json:"use_okpd2"`
			}{
				DatabasePath: databasePath,
				AllActive:    allActive,
				UseKpved:     useKpved,
				UseOkpd2:     useOkpd2,
			}

			s.processNomenclatureDatabasesParallel(databasesToProcess, clientID, projectID, project, req)
		}
	}()

	// Client normalization handlers перемещены в server/normalization_legacy_handlers.go
	return nil
}

// processNomenclatureDatabase обрабатывает нормализацию одной БД номенклатуры
// Вынесено в отдельную функцию для корректной работы defer при закрытии БД
func (s *Server) processNomenclatureDatabase(ctx context.Context, projectDB *database.ProjectDatabase, clientID, projectID int) {
	// Валидация входных параметров
	if projectDB == nil {
		LogNormalizationError(clientID, projectID, fmt.Errorf("projectDB is nil"), "Invalid database parameter")
		return
	}
	if projectDB.ID <= 0 {
		LogNormalizationError(clientID, projectID, fmt.Errorf("invalid projectDB.ID: %d", projectDB.ID), "Invalid database ID")
		return
	}
	if projectDB.FilePath == "" {
		LogNormalizationError(clientID, projectID, fmt.Errorf("empty file path"), "Empty database file path")
		return
	}
	if clientID <= 0 || projectID <= 0 {
		LogNormalizationError(clientID, projectID, fmt.Errorf("invalid clientID or projectID: clientID=%d, projectID=%d", clientID, projectID), "Invalid client or project ID")
		return
	}

	// Проверяем контекст перед началом работы
	select {
	case <-ctx.Done():
		LogNormalizationStopped(clientID, projectID, "context cancelled before start", projectDB.ID)
		return
	default:
	}

	// Проверяем доступность файла БД перед обработкой
	if _, err := os.Stat(projectDB.FilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			LogNormalizationError(clientID, projectID, err, "Database file not found")
			select {
			case s.normalizerEvents <- fmt.Sprintf("Файл БД %s не найден, пропущена", projectDB.Name):
			default:
			}
			return
		}
		LogNormalizationError(clientID, projectID, err, "Error checking database file")
		return
	}

	// Пытаемся создать сессию нормализации для этой базы данных
	// Используем атомарную функцию, которая создаст сессию только если нет активных
	sessionID, created, err := s.serviceDB.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		LogNormalizationError(clientID, projectID, err, "Failed to create normalization session")
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка создания сессии для БД %s: %v", projectDB.FilePath, err):
		default:
		}
		return
	}
	if !created {
		log.Printf("[Nomenclature] Database %s already has active session, skipping", projectDB.Name)
		LogNormalizationStopped(clientID, projectID, "database already has active session", projectDB.ID)
		select {
		case s.normalizerEvents <- fmt.Sprintf("БД %s уже обрабатывается, пропущена", projectDB.Name):
		default:
		}
		return
	}

	// Обновляем last_used_at
	if err := s.serviceDB.UpdateProjectDatabaseLastUsed(projectDB.ID); err != nil {
		LogNormalizationError(clientID, projectID, err, "Failed to update database last used")
		// Не критично, продолжаем
	}

	// Проверяем контекст перед открытием БД
	select {
	case <-ctx.Done():
		s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", nil)
		LogNormalizationStopped(clientID, projectID, "context cancelled before DB open", projectDB.ID)
		return
	default:
	}

	// Открываем подключение к базе данных
	sourceDB, err := database.NewDB(projectDB.FilePath)
	if err != nil {
		// Обновляем сессию как failed
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		LogNormalizationError(clientID, projectID, err, "Failed to open database")
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка открытия БД %s: %v", projectDB.FilePath, err):
		default:
		}
		return
	}
	// Гарантируем закрытие базы данных при выходе из функции
	defer func() {
		if err := sourceDB.Close(); err != nil {
			log.Printf("Error closing database %s: %v", projectDB.FilePath, err)
		}
	}()

	// Для номенклатуры получаем все записи из catalog_items
	items, err := sourceDB.GetAllCatalogItems()
	if err != nil {
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка чтения данных из %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Failed to read data from %s: %v", projectDB.FilePath, err)
		return
	}

	log.Printf("Starting normalization for project %d with %d items from %s", projectID, len(items), projectDB.FilePath)
	select {
	case s.normalizerEvents <- fmt.Sprintf("Начало нормализации БД: %s (%d записей)", projectDB.Name, len(items)):
	default:
	}

	// Создаем клиентский нормализатор
	clientNormalizer := normalization.NewClientNormalizerWithConfig(clientID, projectID, sourceDB, s.serviceDB, s.normalizerEvents, s.workerConfigManager)

	// Устанавливаем sessionID для нормализатора
	clientNormalizer.SetSessionID(sessionID)

	// Проверяем статус сессии перед запуском
	session, err := s.serviceDB.GetNormalizationSession(sessionID)
	if err != nil || session == nil || session.Status != "running" {
		log.Printf("Session %d is not running, skipping normalization", sessionID)
		return
	}

	// Запускаем нормализацию с периодической проверкой остановки
	s.serviceDB.UpdateSessionActivity(sessionID)

	// Запускаем горутину для периодического обновления активности
	activityTicker := time.NewTicker(30 * time.Second)
	activityDone := make(chan bool)
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Panic in session activity goroutine (session_id=%d): %v\n%s", sessionID, rec, debug.Stack())
			}
		}()
		for {
			select {
			case <-activityTicker.C:
				s.serviceDB.UpdateSessionActivity(sessionID)
			case <-activityDone:
				return
			}
		}
	}()

	result, err := clientNormalizer.ProcessWithClientBenchmarks(items)
	activityTicker.Stop()
	activityDone <- true
	finishedAt := time.Now()

	// Проверяем, не была ли сессия остановлена
	session, _ = s.serviceDB.GetNormalizationSession(sessionID)
	if session != nil && session.Status == "stopped" {
		log.Printf("Session %d was stopped during normalization", sessionID)
		return
	}

	if err != nil {
		// Обновляем сессию как failed
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка нормализации БД %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Ошибка клиентской нормализации для %s: %v", projectDB.FilePath, err)
	} else {
		// Обновляем сессию как completed
		s.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
		// Сохраняем результаты нормализации в normalized_data
		if result != nil && result.Groups != nil && len(result.Groups) > 0 {
			normalizedItems, itemAttributes := s.convertClientGroupsToNormalizedItems(result.Groups, projectID, sessionID)
			if len(normalizedItems) > 0 && s.normalizedDB != nil {
				_, saveErr := s.normalizedDB.InsertNormalizedItemsWithAttributesBatch(normalizedItems, itemAttributes, &sessionID, &projectID)
				if saveErr != nil {
					log.Printf("Ошибка сохранения нормализованных данных для БД %s: %v", projectDB.FilePath, saveErr)
					// Отправляем уведомление об ошибке
					if s.notificationService != nil {
						clientIDPtr := &clientID
						projectIDPtr := &projectID
						_, _ = s.notificationService.AddNotification(ctx, services.NotificationTypeError, "Ошибка сохранения результатов", fmt.Sprintf("Не удалось сохранить нормализованные данные для БД %s: %v", projectDB.Name, saveErr), clientIDPtr, projectIDPtr, map[string]interface{}{"database_id": projectDB.ID, "session_id": sessionID})
					}
				} else {
					log.Printf("Сохранено %d нормализованных записей для проекта %d из БД %s", len(normalizedItems), projectID, projectDB.Name)
					// Отправляем уведомление об успешном завершении
					if s.notificationService != nil {
						clientIDPtr := &clientID
						projectIDPtr := &projectID
						_, _ = s.notificationService.AddNotification(ctx, services.NotificationTypeSuccess, "Нормализация завершена", fmt.Sprintf("Успешно нормализовано %d записей из БД %s", len(normalizedItems), projectDB.Name), clientIDPtr, projectIDPtr, map[string]interface{}{"database_id": projectDB.ID, "session_id": sessionID, "items_count": len(normalizedItems)})
					}
				}
			}
		}
	}
}

// EnsureUploadRecordsForDatabase создает или обновляет upload записи в исходной базе данных
// Это необходимо для того, чтобы getNomenclatureFromMainDB мог найти данные через uploads таблицу
// Публичный метод для использования в инструментах и тестах
func (s *Server) EnsureUploadRecordsForDatabase(dbPath string, clientID, projectID, databaseID int) error {
	return s.ensureUploadRecordsForDatabase(dbPath, clientID, projectID, databaseID)
}

// ensureUploadRecordsForDatabase создает или обновляет upload записи в исходной базе данных
// Это необходимо для того, чтобы getNomenclatureFromMainDB мог найти данные через uploads таблицу
func (s *Server) ensureUploadRecordsForDatabase(dbPath string, clientID, projectID, databaseID int) error {
	// Открываем исходную базу данных
	sourceDB, err := database.NewDB(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open source database %s: %w", dbPath, err)
	}
	defer sourceDB.Close()

	// Получаем все существующие upload записи
	uploads, err := sourceDB.GetAllUploads()
	if err != nil {
		// Если таблица uploads не существует, это нормально - база может быть пустой
		log.Printf("Note: Could not get uploads from %s (table may not exist): %v", dbPath, err)
		uploads = []*database.Upload{}
	}

	// Проверяем, есть ли upload записи с правильными client_id и project_id
	needsUpdate := false
	needsCreate := false

	if len(uploads) == 0 {
		needsCreate = true
	} else {
		// Проверяем, есть ли хотя бы одна запись с правильными client_id и project_id
		hasCorrectUpload := false
		for _, upload := range uploads {
			if upload.ClientID != nil && *upload.ClientID == clientID &&
				upload.ProjectID != nil && *upload.ProjectID == projectID {
				hasCorrectUpload = true
				break
			}
		}

		if !hasCorrectUpload {
			needsUpdate = true
			// Если все upload записи не имеют правильных client_id/project_id, создаем новую
			allMissingIDs := true
			for _, upload := range uploads {
				if upload.ClientID != nil || upload.ProjectID != nil {
					allMissingIDs = false
					break
				}
			}
			if allMissingIDs {
				needsCreate = true
			}
		}
	}

	// Обновляем существующие upload записи
	if needsUpdate {
		for _, upload := range uploads {
			// Обновляем только если client_id или project_id отсутствуют или неверны
			shouldUpdate := false
			if upload.ClientID == nil || *upload.ClientID != clientID {
				shouldUpdate = true
			}
			if upload.ProjectID == nil || *upload.ProjectID != projectID {
				shouldUpdate = true
			}

			if shouldUpdate {
				err := sourceDB.UpdateUploadClientProject(upload.ID, clientID, projectID)
				if err != nil {
					log.Printf("Warning: Failed to update upload %d in %s: %v", upload.ID, dbPath, err)
				} else {
					log.Printf("Updated upload %d in %s with client_id=%d, project_id=%d", upload.ID, dbPath, clientID, projectID)
				}
			}
		}
	}

	// Создаем новую upload запись, если нужно
	if needsCreate {
		uploadUUID := uuid.New().String()
		dbID := databaseID
		
		// Пытаемся определить версию 1С и имя конфигурации из метаданных или имени файла
		version1C := "8.3"
		configName := "Unknown"
		
		// Парсим имя файла для получения информации
		fileName := filepath.Base(dbPath)
		fileInfo := database.ParseDatabaseFileInfo(fileName)
		if fileInfo.ConfigName != "" && fileInfo.ConfigName != "Unknown" {
			configName = fileInfo.ConfigName
		}

		upload, err := sourceDB.CreateUploadWithDatabase(
			uploadUUID,
			version1C,
			configName,
			&dbID,
			"", // computerName
			"", // userName
			"", // configVersion
			1,  // iterationNumber
			"", // iterationLabel
			"", // programmerName
			"", // uploadPurpose
			nil, // parentUploadID
		)
		if err != nil {
			return fmt.Errorf("failed to create upload in %s: %w", dbPath, err)
		}

		// Обновляем client_id и project_id
		err = sourceDB.UpdateUploadClientProject(upload.ID, clientID, projectID)
		if err != nil {
			log.Printf("Warning: Failed to update new upload %d with client_id/project_id: %v", upload.ID, err)
		} else {
			log.Printf("Created and updated upload %d in %s with client_id=%d, project_id=%d", upload.ID, dbPath, clientID, projectID)
		}
	}

	return nil
}

// handleKpvedHierarchy возвращает иерархию КПВЭД классификатора
