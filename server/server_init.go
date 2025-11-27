package server

// TODO:legacy-migration revisit dependencies after handler extraction
// Файл содержит методы инициализации и построения функций Server, извлеченные из server.go
// для сокращения размера server.go

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"httpserver/database"
	inframonitoring "httpserver/internal/infrastructure/monitoring"
	"httpserver/internal/infrastructure/workers"
	"httpserver/server/handlers"
	"httpserver/server/services"
)

// initHandlers инициализирует все handlers сервера
func (s *Server) initHandlers() {
	// Устанавливаем logFunc для upload legacy handler
	if s.uploadLegacyHandler != nil {
		// Пересоздаем upload legacy handler с правильным logFunc
		s.uploadLegacyHandler = handlers.NewUploadLegacyHandler(
			s.db,
			s.serviceDB,
			s.dbInfoCache,
			s.qualityAnalyzer,
			func(entry LogEntry) {
				s.log(LogEntry{
					Timestamp:  entry.Timestamp,
					Level:      entry.Level,
					Message:    entry.Message,
					UploadUUID: entry.UploadUUID,
					Endpoint:   entry.Endpoint,
				})
			},
		)
	}

	// Устанавливаем logFunc и generateQualityReport для quality handler
	if s.qualityHandler != nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.qualityHandler = handlers.NewQualityHandler(baseHandler, s.qualityService, func(entry interface{}) {
			// Преобразуем interface{} в LogEntry
			if logEntry, ok := entry.(LogEntry); ok {
				s.log(logEntry)
			}
		}, s.normalizedDB, s.currentNormalizedDBPath)

		// Устанавливаем кэш для статистики проектов (TTL: 5 минут)
		projectStatsCache := handlers.NewProjectQualityStatsCache(5 * time.Minute)
		s.qualityHandler.SetProjectStatsCache(projectStatsCache)
		// Устанавливаем функцию генерации отчета
		// generateQualityReport определен в handlers/quality_legacy.go
		// Но это файл пакета handlers, а метод должен быть в пакете server
		// Временно используем заглушку, пока метод не перемещен в правильный пакет
		if s.qualityHandler != nil {
			s.qualityHandler.SetGenerateQualityReport(func(databasePath string) (interface{}, error) {
				// TODO: Переместить generateQualityReport из handlers/quality_legacy.go в server/quality_legacy_handlers.go
				// Временно возвращаем ошибку, так как метод недоступен
				return nil, fmt.Errorf("generateQualityReport not yet moved to server package")
			})
			// Устанавливаем функцию для получения баз данных проекта
			if s.serviceDB != nil {
				s.qualityHandler.SetGetProjectDatabases(func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error) {
					return s.serviceDB.GetProjectDatabases(projectID, activeOnly)
				})
			}

			// Устанавливаем кэш для статистики проектов (TTL: 5 минут)
			projectStatsCache := handlers.NewProjectQualityStatsCache(5 * time.Minute)
			s.qualityHandler.SetProjectStatsCache(projectStatsCache)
		}
	}

	// Устанавливаем normalizedDB для работы с нормализованными выгрузками
	if s.uploadHandler != nil && s.normalizedDB != nil {
		s.uploadHandler.SetNormalizedDB(s.normalizedDB, s.currentNormalizedDBPath)
	}

	// Устанавливаем databaseService для clientHandler для получения статистики из uploads
	if s.clientHandler != nil && s.databaseService != nil {
		s.clientHandler.SetDatabaseService(s.databaseService)
	}

	// Устанавливаем callback для обновления БД в Server при переключении через DatabaseService
	if s.databaseService != nil {
		s.databaseService.SetOnDBUpdate(func(newDB *database.DB, newPath string) error {
			// Проверяем, что нормализация не запущена
			s.normalizerMutex.RLock()
			isRunning := s.normalizerRunning
			s.normalizerMutex.RUnlock()

			if isRunning {
				return fmt.Errorf("cannot switch database while normalization is running")
			}

			s.dbMutex.Lock()
			defer s.dbMutex.Unlock()

			// Закрываем текущую БД
			if s.db != nil {
				if err := s.db.Close(); err != nil {
					log.Printf("Ошибка закрытия текущей БД: %v", err)
					return fmt.Errorf("failed to close current database: %w", err)
				}
			}

			// Обновляем БД в Server
			s.db = newDB
			s.currentDBPath = newPath

			// Обновляем БД в других сервисах, которые используют её
			if s.uploadService != nil {
				// UploadService может иметь ссылку на БД, нужно обновить если есть метод
				// Пока оставляем как есть, так как uploadService может использовать кэш
			}

			// Обновляем БД в normalization handler
			if s.normalizationHandler != nil {
				s.normalizationHandler.SetDatabase(newDB, newPath, s.normalizedDB, s.currentNormalizedDBPath)
			}

			log.Printf("База данных переключена на: %s", newPath)
			return nil
		})
	}

	// Устанавливаем функции для получения данных из баз uploads для clientHandler
	if s.clientHandler != nil {
		s.clientHandler.SetNomenclatureDataFunctions(
			// getNomenclatureFromNormalizedDB
			func(projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*handlers.NomenclatureResult, int, error) {
				results, total, err := s.getNomenclatureFromNormalizedDB(projectIDs, projectNames, search, limit, offset)
				if err != nil {
					return nil, 0, err
				}
				// Преобразуем NomenclatureResult из server в handlers.NomenclatureResult
				handlerResults := make([]*handlers.NomenclatureResult, len(results))
				for i, r := range results {
					handlerResults[i] = &handlers.NomenclatureResult{
						ID:              r.ID,
						Code:            r.Code,
						Name:            r.Name,
						NormalizedName:  r.NormalizedName,
						Category:        r.Category,
						QualityScore:    r.QualityScore,
						SourceDatabase:  r.SourceDatabase,
						SourceType:      r.SourceType,
						ProjectID:       r.ProjectID,
						ProjectName:     r.ProjectName,
						KpvedCode:       r.KpvedCode,
						KpvedName:       r.KpvedName,
						AIConfidence:    r.AIConfidence,
						AIReasoning:     r.AIReasoning,
						ProcessingLevel: r.ProcessingLevel,
						MergedCount:     r.MergedCount,
						SourceReference: r.SourceReference,
						SourceName:      r.SourceName,
					}
				}
				return handlerResults, total, nil
			},
			// getNomenclatureFromMainDB
			func(dbPath string, clientID int, projectIDs []int, projectNames map[int]string, search string, limit, offset int) ([]*handlers.NomenclatureResult, int, error) {
				results, total, err := s.getNomenclatureFromMainDB(dbPath, clientID, projectIDs, projectNames, search, limit, offset)
				if err != nil {
					return nil, 0, err
				}
				// Преобразуем NomenclatureResult из server в handlers.NomenclatureResult
				handlerResults := make([]*handlers.NomenclatureResult, len(results))
				for i, r := range results {
					handlerResults[i] = &handlers.NomenclatureResult{
						ID:              r.ID,
						Code:            r.Code,
						Name:            r.Name,
						NormalizedName:  r.NormalizedName,
						Category:        r.Category,
						QualityScore:    r.QualityScore,
						SourceDatabase:  r.SourceDatabase,
						SourceType:      r.SourceType,
						ProjectID:       r.ProjectID,
						ProjectName:     r.ProjectName,
						KpvedCode:       r.KpvedCode,
						KpvedName:       r.KpvedName,
						AIConfidence:    r.AIConfidence,
						AIReasoning:     r.AIReasoning,
						ProcessingLevel: r.ProcessingLevel,
						MergedCount:     r.MergedCount,
						SourceReference: r.SourceReference,
						SourceName:      r.SourceName,
					}
				}
				return handlerResults, total, nil
			},
			// getProjectDatabases
			func(projectID int, activeOnly bool) ([]*database.ProjectDatabase, error) {
				return s.serviceDB.GetProjectDatabases(projectID, activeOnly)
			},
			// dbConnectionCache
			s.dbConnectionCache,
		)
	}

	// Устанавливаем logFunc и getModelFromConfig для classification handler
	if s.classificationHandler != nil && s.classificationService != nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		// Пересоздаем classification service с правильным getModelFromConfig и getAPIKeyFromConfig
		var getAPIKey func() string
		if s.workerConfigManager != nil {
			getAPIKey = func() string {
				apiKey, _, err := s.workerConfigManager.GetModelAndAPIKey()
				if err == nil && apiKey != "" {
					return apiKey
				}
				return os.Getenv("ARLIAI_API_KEY")
			}
		}
		s.classificationService = services.NewClassificationService(s.db, s.normalizedDB, s.serviceDB, s.getModelFromConfig, getAPIKey)
		s.classificationHandler = handlers.NewClassificationHandler(baseHandler, s.classificationService, func(entry interface{}) {
			// Преобразуем interface{} в LogEntry
			if logEntry, ok := entry.(LogEntry); ok {
				s.log(logEntry)
			} else if handlersLogEntry, ok := entry.(handlers.LogEntry); ok {
				s.log(LogEntry{
					Timestamp: handlersLogEntry.Timestamp,
					Level:     handlersLogEntry.Level,
					Message:   handlersLogEntry.Message,
					Endpoint:  handlersLogEntry.Endpoint,
				})
			}
		})
	}

	// Устанавливаем logFunc для counterparty handler
	if s.counterpartyHandler != nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.counterpartyHandler = handlers.NewCounterpartyHandler(baseHandler, s.counterpartyService, func(entry interface{}) {
			// Преобразуем interface{} в LogEntry
			if logEntry, ok := entry.(LogEntry); ok {
				s.log(logEntry)
			} else if handlersLogEntry, ok := entry.(handlers.LogEntry); ok {
				s.log(LogEntry{
					Timestamp: handlersLogEntry.Timestamp,
					Level:     handlersLogEntry.Level,
					Message:   handlersLogEntry.Message,
					Endpoint:  handlersLogEntry.Endpoint,
				})
			}
		})
		s.counterpartyHandler.SetExportManager(handlers.NewDefaultCounterpartyExportManager())
		// Устанавливаем enrichmentFactory для массового обогащения
		if s.enrichmentFactory != nil {
			s.counterpartyHandler.SetEnrichmentFactory(s.enrichmentFactory)
		}
	}

	// Устанавливаем logFunc для similarity handler
	if s.similarityHandler != nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.similarityHandler = handlers.NewSimilarityHandler(baseHandler, s.similarityService, func(entry interface{}) {
			// Преобразуем interface{} в LogEntry
			if logEntry, ok := entry.(LogEntry); ok {
				s.log(logEntry)
			} else if handlersLogEntry, ok := entry.(handlers.LogEntry); ok {
				s.log(LogEntry{
					Timestamp: handlersLogEntry.Timestamp,
					Level:     handlersLogEntry.Level,
					Message:   handlersLogEntry.Message,
					Endpoint:  handlersLogEntry.Endpoint,
				})
			}
		})
	}

	// Инициализируем worker handler и сервис
	if s.workerConfigManager != nil {
		if s.workerService == nil {
			workerAdapter := &workers.Adapter{Wcm: s.workerConfigManager}
			s.workerService = services.NewWorkerService(workerAdapter)
		}

		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.workerHandler = handlers.NewWorkerHandler(
			baseHandler,
			s.workerService,
			func(entry interface{}) {
				switch v := entry.(type) {
				case LogEntry:
					s.log(v)
				case handlers.LogEntry:
					s.log(LogEntry{
						Timestamp: v.Timestamp,
						Level:     v.Level,
						Message:   v.Message,
						Endpoint:  v.Endpoint,
					})
				}
			},
			s.checkArliaiConnectionWrapper,
			func(ctx context.Context, traceID string, apiKey string) (interface{}, error) {
				if apiKey == "" && s.workerConfigManager != nil {
					if provider, err := s.workerConfigManager.GetActiveProvider(); err == nil && provider != nil && provider.Name == "openrouter" {
						apiKey = provider.APIKey
					}
				}
				if apiKey == "" {
					apiKey = os.Getenv("OPENROUTER_API_KEY")
				}
				return s.checkOpenRouterConnectionWrapper(ctx, traceID, apiKey)
			},
			func(ctx context.Context, traceID string, apiKey string, baseURL string) (interface{}, error) {
				if apiKey == "" {
					apiKey = os.Getenv("HUGGINGFACE_API_KEY")
				}
				return s.checkHuggingFaceConnectionWrapper(ctx, traceID, apiKey, baseURL)
			},
			s.getModelsWrapper,
			s.getOrchestratorStrategyWrapper,
			s.setOrchestratorStrategyWrapper,
			s.getOrchestratorStatsWrapper,
			s.updateHuggingFaceClientWrapper,
			s.updateProviderOrchestratorWrapper,
		)
	}

	// Устанавливаем logFunc для worker trace handler
	if s.workerTraceHandler != nil {
		s.workerTraceHandler.SetLogFunc(func(entry interface{}) {
			// Преобразуем interface{} в LogEntry
			if logEntry, ok := entry.(LogEntry); ok {
				s.log(logEntry)
			} else if handlersLogEntry, ok := entry.(handlers.LogEntry); ok {
				s.log(LogEntry{
					Timestamp: handlersLogEntry.Timestamp,
					Level:     handlersLogEntry.Level,
					Message:   handlersLogEntry.Message,
					Endpoint:  handlersLogEntry.Endpoint,
				})
			}
		})
	}

	// Устанавливаем функцию получения API ключа для normalization handler
	if s.normalizationHandler != nil && s.workerConfigManager != nil {
		s.normalizationHandler.SetGetArliaiAPIKey(func() string {
			apiKey, _, err := s.workerConfigManager.GetModelAndAPIKey()
			if err != nil {
				// Fallback на переменную окружения
				return os.Getenv("ARLIAI_API_KEY")
			}
			return apiKey
		})
	}

	// Устанавливаем функцию запуска нормализации для normalization handler
	if s.normalizationHandler != nil {
		s.normalizationHandler.SetStartNormalizationFunc(s.startProjectNormalization)
		// Устанавливаем clientService для normalization handler
		if s.clientService != nil {
			s.normalizationHandler.SetClientService(s.clientService)
		}
	}

	// Logs handler уже создан в container и присвоен в NewServerWithConfig
	// Если по какой-то причине он nil, создаем его здесь как fallback
	if s.logsHandler == nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.logsHandler = handlers.NewLogsHandler(baseHandler)
		log.Printf("[initHandlers] LogsHandler был nil, создан fallback handler")
	}

	// Устанавливаем функции для monitoring handler
	if s.monitoringHandler != nil && s.monitoringService != nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.monitoringHandler = handlers.NewMonitoringHandler(
			baseHandler,
			s.monitoringService,
			func(entry interface{}) {
				// Преобразуем interface{} в LogEntry
				if logEntry, ok := entry.(LogEntry); ok {
					s.log(logEntry)
				} else if handlersLogEntry, ok := entry.(handlers.LogEntry); ok {
					s.log(LogEntry{
						Timestamp: handlersLogEntry.Timestamp,
						Level:     handlersLogEntry.Level,
						Message:   handlersLogEntry.Message,
						Endpoint:  handlersLogEntry.Endpoint,
					})
				}
			},
			func() map[string]interface{} {
				// getCircuitBreakerState - будет реализовано при необходимости
				return map[string]interface{}{"state": "closed"}
			},
			func() map[string]interface{} {
				// getBatchProcessorStats - будет реализовано при необходимости
				return map[string]interface{}{}
			},
			func() map[string]interface{} {
				// getCheckpointStatus - будет реализовано при необходимости
				return map[string]interface{}{}
			},
			func() *database.PerformanceMetricsSnapshot {
				// collectMetricsSnapshot - будет реализовано при необходимости
				return nil
			},
			func() handlers.MonitoringData {
				// getMonitoringMetrics - преобразуем MonitoringData из server пакета в handlers.MonitoringData
				// Обработка паники для безопасности
				defer func() {
					if r := recover(); r != nil {
						s.log(LogEntry{
							Timestamp: time.Now(),
							Level:     "ERROR",
							Message:   fmt.Sprintf("Panic in getMonitoringMetrics: %v", r),
							Endpoint:  "/api/monitoring/providers/stream",
						})
					}
				}()

				if s.monitoringManager == nil {
					return handlers.MonitoringData{
						Providers: []handlers.ProviderMetrics{},
						System: handlers.SystemStats{
							Timestamp: time.Now().Format(time.RFC3339),
						},
					}
				}

				// Безопасно получаем метрики с обработкой паники
				var serverData inframonitoring.MonitoringData
				func() {
					defer func() {
						if r := recover(); r != nil {
							s.log(LogEntry{
								Timestamp: time.Now(),
								Level:     "ERROR",
								Message:   fmt.Sprintf("Panic in monitoringManager.GetAllMetrics: %v", r),
								Endpoint:  "/api/monitoring/providers/stream",
							})
							// Возвращаем пустые метрики при панике
							serverData = inframonitoring.MonitoringData{
								Providers: []inframonitoring.ProviderMetrics{},
								System: inframonitoring.SystemStats{
									Timestamp: time.Now(),
								},
							}
						}
					}()
					serverData = s.monitoringManager.GetAllMetrics()
				}()

				// Преобразуем провайдеры с обработкой паники
				var providers []handlers.ProviderMetrics
				func() {
					defer func() {
						if r := recover(); r != nil {
							s.log(LogEntry{
								Timestamp: time.Now(),
								Level:     "ERROR",
								Message:   fmt.Sprintf("Panic converting providers: %v", r),
								Endpoint:  "/api/monitoring/providers/stream",
							})
							providers = []handlers.ProviderMetrics{}
						}
					}()
					if serverData.Providers != nil {
						providers = make([]handlers.ProviderMetrics, len(serverData.Providers))
						for i, p := range serverData.Providers {
							lastRequestTimeStr := ""
							if !p.LastRequestTime.IsZero() {
								lastRequestTimeStr = p.LastRequestTime.Format(time.RFC3339)
							}
							providers[i] = handlers.ProviderMetrics{
								ID:                 p.ID,
								Name:               p.Name,
								ActiveChannels:     p.ActiveChannels,
								CurrentRequests:    p.CurrentRequests,
								TotalRequests:      p.TotalRequests,
								SuccessfulRequests: p.SuccessfulRequests,
								FailedRequests:     p.FailedRequests,
								AverageLatencyMs:   p.AverageLatencyMs,
								LastRequestTime:    lastRequestTimeStr,
								Status:             p.Status,
								RequestsPerSecond:  p.RequestsPerSecond,
							}
						}
					} else {
						providers = []handlers.ProviderMetrics{}
					}
				}()

				// Преобразуем системную статистику с обработкой паники
				var timestampStr string
				func() {
					defer func() {
						if r := recover(); r != nil {
							s.log(LogEntry{
								Timestamp: time.Now(),
								Level:     "ERROR",
								Message:   fmt.Sprintf("Panic converting timestamp: %v", r),
								Endpoint:  "/api/monitoring/providers/stream",
							})
							timestampStr = time.Now().Format(time.RFC3339)
						}
					}()
					if !serverData.System.Timestamp.IsZero() {
						timestampStr = serverData.System.Timestamp.Format(time.RFC3339)
					} else {
						timestampStr = time.Now().Format(time.RFC3339)
					}
				}()

				return handlers.MonitoringData{
					Providers: providers,
					System: handlers.SystemStats{
						TotalProviders:          serverData.System.TotalProviders,
						ActiveProviders:         serverData.System.ActiveProviders,
						TotalRequests:           serverData.System.TotalRequests,
						TotalSuccessful:         serverData.System.TotalSuccessful,
						TotalFailed:             serverData.System.TotalFailed,
						SystemRequestsPerSecond: serverData.System.SystemRequestsPerSecond,
						Timestamp:               timestampStr,
					},
				}
			},
		)
	}

	s.ensureDashboardComponents()

	// Устанавливаем logFunc и функции для snapshot handler
	if s.snapshotHandler != nil {
		baseHandler := handlers.NewBaseHandlerFromMiddleware()
		s.snapshotHandler = handlers.NewSnapshotHandler(
			baseHandler,
			s.snapshotService,
			func(entry interface{}) {
				// Преобразуем interface{} в LogEntry
				if logEntry, ok := entry.(LogEntry); ok {
					s.log(logEntry)
				} else if handlersLogEntry, ok := entry.(handlers.LogEntry); ok {
					s.log(LogEntry{
						Timestamp: handlersLogEntry.Timestamp,
						Level:     handlersLogEntry.Level,
						Message:   handlersLogEntry.Message,
						Endpoint:  handlersLogEntry.Endpoint,
					})
				}
			},
			s.serviceDB,
			func(snapshotID int, req interface{}) (interface{}, error) {
				// Преобразуем req в SnapshotNormalizationRequest
				reqMap, ok := req.(map[string]interface{})
				if !ok {
					return nil, fmt.Errorf("invalid request format")
				}
				normalizeReq := SnapshotNormalizationRequest{
					UseAI:            false,
					MinConfidence:    0.7,
					RateLimitDelayMS: 100,
					MaxRetries:       3,
				}
				if useAI, ok := reqMap["use_ai"].(bool); ok {
					normalizeReq.UseAI = useAI
				}
				if minConf, ok := reqMap["min_confidence"].(float64); ok {
					normalizeReq.MinConfidence = minConf
				}
				if delay, ok := reqMap["rate_limit_delay_ms"].(float64); ok {
					normalizeReq.RateLimitDelayMS = int(delay)
				}
				if retries, ok := reqMap["max_retries"].(float64); ok {
					normalizeReq.MaxRetries = int(retries)
				}
				return s.normalizeSnapshot(snapshotID, normalizeReq)
			},
			func(snapshotID int) (interface{}, error) {
				result, err := s.compareSnapshotIterations(snapshotID)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
			func(snapshotID int) (interface{}, error) {
				result, err := s.calculateSnapshotMetrics(snapshotID)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
			func(snapshotID int) (interface{}, error) {
				result, err := s.getSnapshotEvolution(snapshotID)
				if err != nil {
					return nil, err
				}
				return result, nil
			},
			s.createAutoSnapshot,
		)
	}

	// Инициализируем дефолтные привязки классификаторов к типам проектов
	s.initDefaultProjectTypeClassifiers()
}

func (s *Server) ensureDashboardComponents() {
	statsFunc := s.buildDashboardStatsFunc()
	normalizationStatusFunc := s.buildNormalizationStatusFunc()

	s.dashboardService = services.NewDashboardService(
		s.db,
		s.normalizedDB,
		s.serviceDB,
		statsFunc,
		normalizationStatusFunc,
	)

	if serverContainer, ok := s.container.(*Container); ok {
		serverContainer.DashboardService = s.dashboardService
	}

	baseHandler := handlers.NewBaseHandlerFromMiddleware()
	s.dashboardHandler = handlers.NewDashboardHandlerWithServices(
		s.dashboardService,
		s.clientService,
		s.normalizationService,
		s.qualityService,
		baseHandler,
		s.buildMonitoringMetricsFunc(),
	)
}

func (s *Server) buildDashboardStatsFunc() func() map[string]interface{} {
	return func() map[string]interface{} {
		totalRecords := s.queryMainDBCount("SELECT COUNT(*) FROM normalized_data")
		createdGroups := s.queryMainDBCount("SELECT COUNT(*) FROM catalogs")
		mergedRecords := s.queryMainDBCount("SELECT COALESCE(SUM(COALESCE(merged_count, 0)), 0) FROM normalized_data")
		totalDatabases := s.queryServiceDBCount("SELECT COUNT(*) FROM project_databases")

		stats := map[string]interface{}{
			"totalRecords":     totalRecords,
			"processedRecords": s.computeProcessedRecords(totalRecords),
			"createdGroups":    createdGroups,
			"mergedRecords":    mergedRecords,
			"totalDatabases":   totalDatabases,
			"systemVersion":    s.resolveSystemVersion(),
			"currentDatabase":  s.buildCurrentDatabaseInfo(),
			"timestamp":        time.Now().Format(time.RFC3339),
		}

		if statusFunc := s.buildNormalizationStatusFunc(); statusFunc != nil {
			stats["normalizationStatus"] = statusFunc()
		}

		return stats
	}
}

func (s *Server) buildNormalizationStatusFunc() func() map[string]interface{} {
	return func() map[string]interface{} {
		s.normalizerMutex.RLock()
		isRunning := s.normalizerRunning
		processed := s.normalizerProcessed
		startTime := s.normalizerStartTime
		s.normalizerMutex.RUnlock()

		status := map[string]interface{}{
			"status":       "idle",
			"progress":     0,
			"currentStage": "Ожидание",
			"currentStep":  "Ожидание",
			"processed":    processed,
			"total":        0,
			"isRunning":    isRunning,
		}

		var totalCatalogItems int
		if s.db != nil {
			if err := s.db.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&totalCatalogItems); err != nil {
				log.Printf("Error getting total catalog items: %v", err)
				totalCatalogItems = 0
			}
		}
		status["total"] = totalCatalogItems

		if isRunning {
			status["status"] = "running"
			status["currentStage"] = "Обработка данных..."
			status["currentStep"] = "Обработка данных..."

			if totalCatalogItems > 0 {
				progress := float64(processed) / float64(totalCatalogItems) * 100
				if progress > 100 {
					progress = 100
				}
				status["progress"] = progress
			}

			if !startTime.IsZero() {
				status["startTime"] = startTime.Format(time.RFC3339)
				elapsed := time.Since(startTime)
				status["elapsedTime"] = elapsed.Round(time.Second).String()
				if elapsed.Seconds() > 0 && processed > 0 {
					status["rate"] = float64(processed) / elapsed.Seconds()
				}
			}
		} else if processed > 0 && totalCatalogItems > 0 && processed >= totalCatalogItems {
			status["status"] = "completed"
			status["progress"] = 100
			status["currentStage"] = "Завершено"
			status["currentStep"] = "Завершено"
			if !startTime.IsZero() {
				elapsed := time.Since(startTime)
				status["elapsedTime"] = elapsed.Round(time.Second).String()
				if elapsed.Seconds() > 0 {
					status["rate"] = float64(processed) / elapsed.Seconds()
				}
			}
		}

		return status
	}
}

func (s *Server) buildMonitoringMetricsFunc() func() handlers.MonitoringData {
	return func() handlers.MonitoringData {
		if s.monitoringManager == nil {
			return handlers.MonitoringData{
				Providers: []handlers.ProviderMetrics{},
				System:    handlers.SystemStats{Timestamp: time.Now().Format(time.RFC3339)},
			}
		}

		serverData := s.monitoringManager.GetAllMetrics()
		providers := make([]handlers.ProviderMetrics, len(serverData.Providers))
		for i, p := range serverData.Providers {
			lastRequestTimeStr := ""
			if !p.LastRequestTime.IsZero() {
				lastRequestTimeStr = p.LastRequestTime.Format(time.RFC3339)
			}
			providers[i] = handlers.ProviderMetrics{
				ID:                 p.ID,
				Name:               p.Name,
				ActiveChannels:     p.ActiveChannels,
				CurrentRequests:    p.CurrentRequests,
				TotalRequests:      p.TotalRequests,
				SuccessfulRequests: p.SuccessfulRequests,
				FailedRequests:     p.FailedRequests,
				AverageLatencyMs:   p.AverageLatencyMs,
				LastRequestTime:    lastRequestTimeStr,
				Status:             p.Status,
				RequestsPerSecond:  p.RequestsPerSecond,
			}
		}

		timestampStr := time.Now().Format(time.RFC3339)
		if !serverData.System.Timestamp.IsZero() {
			timestampStr = serverData.System.Timestamp.Format(time.RFC3339)
		}

		return handlers.MonitoringData{
			Providers: providers,
			System: handlers.SystemStats{
				TotalProviders:          serverData.System.TotalProviders,
				ActiveProviders:         serverData.System.ActiveProviders,
				TotalRequests:           serverData.System.TotalRequests,
				TotalSuccessful:         serverData.System.TotalSuccessful,
				TotalFailed:             serverData.System.TotalFailed,
				SystemRequestsPerSecond: serverData.System.SystemRequestsPerSecond,
				Timestamp:               timestampStr,
			},
		}
	}
}

func (s *Server) queryMainDBCount(query string) int {
	if s.db == nil || s.db.GetDB() == nil {
		return 0
	}
	var count sql.NullInt64
	if err := s.db.GetDB().QueryRow(query).Scan(&count); err != nil {
		log.Printf("Dashboard stats main DB query failed (%s): %v", query, err)
		return 0
	}
	if count.Valid {
		return int(count.Int64)
	}
	return 0
}

func (s *Server) queryServiceDBCount(query string) int {
	if s.serviceDB == nil || s.serviceDB.GetDB() == nil {
		return 0
	}
	var count sql.NullInt64
	if err := s.serviceDB.GetDB().QueryRow(query).Scan(&count); err != nil {
		log.Printf("Dashboard stats service DB query failed (%s): %v", query, err)
		return 0
	}
	if count.Valid {
		return int(count.Int64)
	}
	return 0
}

func (s *Server) computeProcessedRecords(total int) int {
	s.normalizerMutex.RLock()
	processed := s.normalizerProcessed
	s.normalizerMutex.RUnlock()
	if processed > 0 {
		return processed
	}
	return total
}

func (s *Server) buildCurrentDatabaseInfo() map[string]interface{} {
	info := map[string]interface{}{
		"path":           s.currentDBPath,
		"normalizedPath": s.currentNormalizedDBPath,
	}

	if s.currentDBPath != "" {
		if stat, err := os.Stat(s.currentDBPath); err == nil {
			info["sizeBytes"] = stat.Size()
			info["updatedAt"] = stat.ModTime().Format(time.RFC3339)
			info["name"] = filepath.Base(s.currentDBPath)
		}
	}

	return info
}

func (s *Server) resolveSystemVersion() string {
	if v := os.Getenv("APP_VERSION"); v != "" {
		return v
	}
	if s.config != nil && s.config.DatabasePath != "" {
		return filepath.Base(s.config.DatabasePath)
	}
	return "unknown"
}

// startSessionTimeoutChecker уже объявлен в server.go:209, не дублируем
