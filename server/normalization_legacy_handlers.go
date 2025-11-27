package server

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"database/sql"

	_ "github.com/mattn/go-sqlite3"

	"httpserver/database"
	"httpserver/normalization"
	"httpserver/server/services"
)

// Legacy normalization handlers - перемещены из server.go для рефакторинга
// TODO: Заменить на новые handlers из internal/api/handlers/

// handleNormalizeStart обрабатывает запрос на запуск нормализации
func (s *Server) handleNormalizeStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Парсим тело запроса для получения AI конфигурации
	type NormalizeRequest struct {
		UseAI            bool    `json:"use_ai"`
		MinConfidence    float64 `json:"min_confidence"`
		RateLimitDelayMS int     `json:"rate_limit_delay_ms"`
		MaxRetries       int     `json:"max_retries"`
		Model            string  `json:"model"`     // Выбранная модель AI
		Database         string  `json:"database"`  // База данных для нормализации
		UseKpved         bool    `json:"use_kpved"` // Включить КПВЭД классификацию
		UseOkpd2         bool    `json:"use_okpd2"` // Включить ОКПД2 классификацию
		UploadID         int     `json:"upload_id"` // ID выгрузки для привязки checkpoint
	}

	var req NormalizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Если тело пустое или некорректное, используем значения по умолчанию
		req.UseAI = false
		req.MinConfidence = 0.8
		req.RateLimitDelayMS = 100
		req.MaxRetries = 3
	}

	// Проверяем, не запущен ли уже процесс
	s.normalizerMutex.Lock()
	if s.normalizerRunning {
		s.normalizerMutex.Unlock()
		LogWarn(r.Context(), "Normalization start requested but already running")
		s.handleHTTPError(w, r, NewConflictError("Нормализация уже выполняется", nil))
		return
	}
	s.normalizerRunning = true
	s.normalizerStartTime = time.Now()
	s.normalizerProcessed = 0
	s.normalizerSuccess = 0
	s.normalizerErrors = 0
	s.normalizerMutex.Unlock()

	// Загружаем конфигурацию нормализации из serviceDB
	var config *database.NormalizationConfig
	if s.serviceDB != nil {
		var err error
		config, err = s.serviceDB.GetNormalizationConfig()
		if err != nil {
			LogWarn(r.Context(), "Failed to get normalization config, using defaults", "error", err)
			config = nil
		}
	}

	// Используем дефолтные значения, если конфигурация не загружена
	if config == nil {
		config = &database.NormalizationConfig{
			SourceTable:     "catalog_items",
			ReferenceColumn: "reference",
			CodeColumn:      "code",
			NameColumn:      "name",
		}
	}

	// Применяем конфигурацию к нормализатору
	s.normalizer.SetSourceConfig(
		config.SourceTable,
		config.ReferenceColumn,
		config.CodeColumn,
		config.NameColumn,
	)

	log.Printf("Запуск нормализации: таблица=%s, ref=%s, code=%s, name=%s",
		config.SourceTable, config.ReferenceColumn, config.CodeColumn, config.NameColumn)
	if req.UseAI {
		log.Printf("AI параметры из запроса игнорируются, используется конфигурация сервера")
	}

	// Если указана модель, устанавливаем её как активную через WorkerConfigManager
	if req.Model != "" && s.workerConfigManager != nil {
		// Валидируем имя модели
		if err := ValidateModelName(req.Model); err != nil {
			log.Printf("[normalize-start] Warning: Invalid model name %s: %v", req.Model, err)
			s.normalizerEvents <- fmt.Sprintf("⚠ Предупреждение: неверное имя модели %s: %v", req.Model, err)
			// Санитизируем и продолжаем
			req.Model = SanitizeModelName(req.Model)
		}

		provider, err := s.workerConfigManager.GetActiveProvider()
		if err == nil {
			// Устанавливаем модель как активную для провайдера
			if err := s.workerConfigManager.SetDefaultModel(provider.Name, req.Model); err != nil {
				log.Printf("[normalize-start] Warning: Failed to set model %s: %v", req.Model, err)
				s.normalizerEvents <- fmt.Sprintf("⚠ Предупреждение: не удалось установить модель %s: %v", req.Model, err)
			} else {
				log.Printf("[normalize-start] Установлена модель для нормализации: %s (провайдер: %s)", req.Model, provider.Name)
				s.normalizerEvents <- fmt.Sprintf("✓ Модель установлена: %s", req.Model)

				// Получаем информацию о модели для логирования
				model, modelErr := s.workerConfigManager.GetActiveModel(provider.Name)
				if modelErr == nil {
					log.Printf("[normalize-start] Параметры модели: скорость=%s, качество=%s, max_tokens=%d, temperature=%.2f",
						model.Speed, model.Quality, model.MaxTokens, model.Temperature)
				}
			}
		} else {
			log.Printf("[normalize-start] Warning: Failed to get active provider: %v", err)
		}
	}

	// Если указана база данных в запросе, открываем её и создаем новый normalizer
	var normalizerToUse *normalization.Normalizer
	var tempDB *database.DB

	if req.Database != "" {
		log.Printf("Открытие базы данных: %s", req.Database)
		s.normalizerEvents <- fmt.Sprintf("Открытие базы данных: %s", req.Database)

		var err error
		tempDB, err = database.NewDB(req.Database)
		if err != nil {
			s.normalizerMutex.Lock()
			s.normalizerRunning = false
			s.normalizerMutex.Unlock()
			LogError(r.Context(), err, "Failed to open database for normalization", "database", req.Database)
			s.handleHTTPError(w, r, NewInternalError("не удалось открыть базу данных", err))
			return
		}

		// Создаем временный normalizer для указанной БД
		aiConfig := &normalization.AIConfig{
			Enabled:        true,
			MinConfidence:  0.7,
			RateLimitDelay: 100 * time.Millisecond,
			MaxRetries:     3,
		}
		// Создаем функцию проверки остановки
		stopCheck := s.createStopCheckFunction()
		// Создаем функцию получения API ключа из конфигурации воркеров
		var getAPIKey func() string
		if s.workerConfigManager != nil {
			getAPIKey = func() string {
				apiKey, _, err := s.workerConfigManager.GetModelAndAPIKey()
				if err != nil {
					return "" // Fallback на переменную окружения в NewNormalizer
				}
				return apiKey
			}
		}
		// Создаем временный normalizer для БД с проверкой остановки
		normalizerToUse = normalization.NewNormalizerWithStopCheck(tempDB, s.normalizerEvents, aiConfig, stopCheck, getAPIKey)
		normalizerToUse.SetSourceConfig(
			config.SourceTable,
			config.ReferenceColumn,
			config.CodeColumn,
			config.NameColumn,
		)
		LogInfo(r.Context(), "Created temporary normalizer for database", "database", req.Database)
	} else {
		// Используем стандартный normalizer
		if s.normalizer == nil {
			s.normalizerMutex.Lock()
			s.normalizerRunning = false
			s.normalizerMutex.Unlock()
			s.writeJSONError(w, r, "Normalizer not initialized", http.StatusInternalServerError)
			return
		}
		// Обновляем normalizer с проверкой остановки, если возможно
		// Если normalizer уже создан, используем его, но проверка остановки будет работать через флаг
		// Создаем функцию проверки остановки
		stopCheck := s.createStopCheckFunction()
		// Используем стандартный normalizer и устанавливаем функцию проверки остановки
		normalizerToUse = s.normalizer
		normalizerToUse.SetStopCheck(stopCheck)
		LogInfo(r.Context(), "Используется стандартный normalizer")
	}

	// Возвращаем успешный ответ перед запуском горутины
	s.writeJSONResponse(w, r, map[string]interface{}{
		"status":  "started",
		"message": "Normalization started",
	}, http.StatusOK)

	// Запускаем нормализацию в горутине
	go func() {
		defer func() {
			// Обработка паники и очистка состояния
			if rec := recover(); rec != nil {
				log.Printf("Panic in normalization goroutine: %v\n%s", rec, debug.Stack())
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
			// Закрываем временную БД если она была открыта
			if tempDB != nil {
				tempDB.Close()
				log.Printf("Временная БД %s закрыта", req.Database)
			}

			// Всегда сбрасываем флаг running при выходе
			s.normalizerMutex.Lock()
			s.normalizerRunning = false
			s.normalizerMutex.Unlock()
			log.Println("Normalization goroutine finished")
		}()

		log.Println("Запуск процесса нормализации в горутине...")
		s.normalizerEvents <- "Начало нормализации данных..."

		// Отслеживаем события для обновления статистики
		eventTicker := time.NewTicker(2 * time.Second)
		defer eventTicker.Stop()

		go func() {
			defer func() {
				if rec := recover(); rec != nil {
					log.Printf("Panic in normalization stats goroutine: %v\n%s", rec, debug.Stack())
				}
			}()
			for range eventTicker.C {
				s.normalizerMutex.RLock()
				isRunning := s.normalizerRunning
				s.normalizerMutex.RUnlock()
				if !isRunning {
					return
				}
				// Обновляем processed из БД
				var count int
				if err := s.db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&count); err == nil {
					s.normalizerMutex.Lock()
					s.normalizerProcessed = count
					s.normalizerMutex.Unlock()
				}
			}
		}()

		// Получаем uploadID из запроса, если указан
		uploadID := req.UploadID
		if uploadID == 0 {
			uploadID = 1 // Значение по умолчанию, если не указано
		}
		if err := normalizerToUse.ProcessNormalization(uploadID); err != nil {
			log.Printf("Ошибка нормализации данных: %v", err)
			s.normalizerEvents <- fmt.Sprintf("Ошибка нормализации: %v", err)
			s.normalizerMutex.Lock()
			s.normalizerErrors++
			s.normalizerMutex.Unlock()
		} else {
			log.Println("Нормализация завершена успешно")
			s.normalizerEvents <- "Нормализация завершена успешно"
			// Обновляем финальную статистику
			var finalCount int
			if err := s.db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&finalCount); err == nil {
				s.normalizerMutex.Lock()
				s.normalizerProcessed = finalCount
				s.normalizerSuccess = finalCount
				s.normalizerMutex.Unlock()
			}

			// КПВЭД классификация после нормализации
			if req.UseKpved && s.hierarchicalClassifier != nil {
				log.Println("Начинаем КПВЭД классификацию...")
				s.normalizerEvents <- "Начало КПВЭД классификации"

				s.kpvedClassifierMutex.RLock()
				classifier := s.hierarchicalClassifier
				s.kpvedClassifierMutex.RUnlock()

				if classifier != nil {
					// Определяем какую БД использовать: временную или стандартную
					dbToUse := s.normalizedDB
					if tempDB != nil {
						dbToUse = tempDB
					}

					// Получаем записи без КПВЭД классификации
					rows, err := dbToUse.Query(`
						SELECT id, normalized_name, category
						FROM normalized_data
						WHERE (kpved_code IS NULL OR kpved_code = '' OR TRIM(kpved_code) = '')
					`)
					if err != nil {
						log.Printf("Ошибка получения записей для КПВЭД классификации: %v", err)
						s.normalizerEvents <- fmt.Sprintf("Ошибка КПВЭД: %v", err)
					} else {
						defer rows.Close()

						var recordsToClassify []struct {
							ID             int
							NormalizedName string
							Category       string
						}

						for rows.Next() {
							var record struct {
								ID             int
								NormalizedName string
								Category       string
							}
							if err := rows.Scan(&record.ID, &record.NormalizedName, &record.Category); err != nil {
								log.Printf("Ошибка сканирования записи: %v", err)
								continue
							}
							recordsToClassify = append(recordsToClassify, record)
						}

						totalToClassify := len(recordsToClassify)
						if totalToClassify == 0 {
							log.Println("Нет записей для КПВЭД классификации")
							s.normalizerEvents <- "Все записи уже классифицированы по КПВЭД"
						} else {
							log.Printf("Найдено записей для КПВЭД классификации: %d", totalToClassify)
							s.normalizerEvents <- fmt.Sprintf("Классификация %d записей по КПВЭД", totalToClassify)

							classified := 0
							failed := 0
							for i, record := range recordsToClassify {
								// Классифицируем запись
								result, err := classifier.Classify(record.NormalizedName, record.Category)
								if err != nil {
									log.Printf("Ошибка классификации записи %d: %v", record.ID, err)
									failed++
									continue
								}

								// Обновляем запись с результатами классификации
								_, err = dbToUse.Exec(`
									UPDATE normalized_data
									SET kpved_code = ?, kpved_name = ?, kpved_confidence = ?,
									    stage11_kpved_code = ?, stage11_kpved_name = ?, stage11_kpved_confidence = ?,
									    stage11_kpved_completed = 1, stage11_kpved_completed_at = CURRENT_TIMESTAMP
									WHERE id = ?
								`, result.FinalCode, result.FinalName, result.FinalConfidence,
									result.FinalCode, result.FinalName, result.FinalConfidence, record.ID)

								if err != nil {
									log.Printf("Ошибка обновления КПВЭД для записи %d: %v", record.ID, err)
									failed++
									continue
								}

								classified++

								// Логируем прогресс каждые 10 записей или на последней записи
								if (i+1)%10 == 0 || i+1 == totalToClassify {
									progress := float64(i+1) / float64(totalToClassify) * 100
									log.Printf("КПВЭД классификация: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
									s.normalizerEvents <- fmt.Sprintf("КПВЭД: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
								}
							}

							log.Printf("КПВЭД классификация завершена: классифицировано %d из %d записей (ошибок: %d)", classified, totalToClassify, failed)
							s.normalizerEvents <- fmt.Sprintf("КПВЭД классификация завершена: %d/%d (ошибок: %d)", classified, totalToClassify, failed)
						}
					}
				} else {
					log.Println("КПВЭД классификатор недоступен")
					s.normalizerEvents <- "КПВЭД классификатор недоступен"
				}
			} else if req.UseKpved {
				log.Println("КПВЭД классификация запрошена, но классификатор не инициализирован")
				s.normalizerEvents <- "КПВЭД классификатор не инициализирован. Проверьте ARLIAI_API_KEY"
			}

			// ОКПД2 классификация после нормализации (и после КПВЭД, если она была)
			if req.UseOkpd2 {
				log.Println("Начинаем ОКПД2 классификацию...")
				s.normalizerEvents <- "Начало ОКПД2 классификации"

				// Определяем какую БД использовать: временную или стандартную
				dbToUse := s.normalizedDB
				if tempDB != nil {
					dbToUse = tempDB
				}

				// Получаем записи без ОКПД2 классификации
				rows, err := dbToUse.Query(`
					SELECT id, normalized_name, category
					FROM normalized_data
					WHERE (stage12_okpd2_code IS NULL OR stage12_okpd2_code = '' OR TRIM(stage12_okpd2_code) = '')
				`)
				if err != nil {
					log.Printf("Ошибка получения записей для ОКПД2 классификации: %v", err)
					s.normalizerEvents <- fmt.Sprintf("Ошибка ОКПД2: %v", err)
				} else {
					defer rows.Close()

					var recordsToClassify []struct {
						ID             int
						NormalizedName string
						Category       string
					}

					for rows.Next() {
						var record struct {
							ID             int
							NormalizedName string
							Category       string
						}
						if err := rows.Scan(&record.ID, &record.NormalizedName, &record.Category); err != nil {
							log.Printf("Ошибка сканирования записи: %v", err)
							continue
						}
						recordsToClassify = append(recordsToClassify, record)
					}

					totalToClassify := len(recordsToClassify)
					if totalToClassify == 0 {
						log.Println("Нет записей для ОКПД2 классификации")
						s.normalizerEvents <- "Все записи уже классифицированы по ОКПД2"
					} else {
						log.Printf("Найдено записей для ОКПД2 классификации: %d", totalToClassify)
						s.normalizerEvents <- fmt.Sprintf("Классификация %d записей по ОКПД2", totalToClassify)

						classified := 0
						failed := 0
						serviceDB := s.serviceDB.GetDB()

						for i, record := range recordsToClassify {
							// Простой поиск по ключевым словам в ОКПД2
							// Извлекаем ключевые слова из нормализованного имени
							searchTerms := extractKeywords(record.NormalizedName)
							if len(searchTerms) == 0 {
								searchTerms = []string{record.NormalizedName}
							}

							var bestMatch struct {
								Code       string
								Name       string
								Confidence float64
							}
							bestMatch.Confidence = 0.0

							// Ищем совпадения по каждому ключевому слову
							for _, term := range searchTerms {
								if len(term) < 3 {
									continue // Пропускаем слишком короткие слова
								}

								searchPattern := "%" + term + "%"
								query := `
									SELECT code, name, level
									FROM okpd2_classifier
									WHERE name LIKE ?
									ORDER BY 
										CASE 
											WHEN name LIKE ? THEN 1
											WHEN name LIKE ? THEN 2
											ELSE 3
										END,
										level DESC
									LIMIT 5
								`
								exactPattern := term
								startPattern := term + "%"

								rows, err := serviceDB.Query(query, searchPattern, exactPattern, startPattern)
								if err != nil {
									log.Printf("Ошибка поиска ОКПД2 для '%s': %v", term, err)
									continue
								}

								for rows.Next() {
									var code, name string
									var level int
									if err := rows.Scan(&code, &name, &level); err != nil {
										continue
									}

									// Вычисляем уверенность на основе совпадения
									confidence := calculateOkpd2Confidence(term, name, level)
									if confidence > bestMatch.Confidence {
										bestMatch.Code = code
										bestMatch.Name = name
										bestMatch.Confidence = confidence
									}
								}
								rows.Close()
							}

							// Если нашли совпадение с достаточной уверенностью, сохраняем
							if bestMatch.Confidence >= 0.3 && bestMatch.Code != "" {
								_, err = dbToUse.Exec(`
									UPDATE normalized_data
									SET stage12_okpd2_code = ?, stage12_okpd2_name = ?, stage12_okpd2_confidence = ?,
									    stage12_okpd2_completed = 1, stage12_okpd2_completed_at = CURRENT_TIMESTAMP
									WHERE id = ?
								`, bestMatch.Code, bestMatch.Name, bestMatch.Confidence, record.ID)

								if err != nil {
									log.Printf("Ошибка обновления ОКПД2 для записи %d: %v", record.ID, err)
									failed++
									continue
								}

								classified++
							} else {
								// Отмечаем как обработанное, но без результата
								_, err = dbToUse.Exec(`
									UPDATE normalized_data
									SET stage12_okpd2_completed = 1, stage12_okpd2_completed_at = CURRENT_TIMESTAMP
									WHERE id = ?
								`, record.ID)
								if err != nil {
									log.Printf("Ошибка обновления статуса ОКПД2 для записи %d: %v", record.ID, err)
								}
								failed++
							}

							// Логируем прогресс каждые 10 записей или на последней записи
							if (i+1)%10 == 0 || i+1 == totalToClassify {
								progress := float64(i+1) / float64(totalToClassify) * 100
								log.Printf("ОКПД2 классификация: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
								s.normalizerEvents <- fmt.Sprintf("ОКПД2: %d/%d (%.1f%%)", i+1, totalToClassify, progress)
							}
						}

						log.Printf("ОКПД2 классификация завершена: классифицировано %d из %d записей (не найдено: %d)", classified, totalToClassify, failed)
						s.normalizerEvents <- fmt.Sprintf("ОКПД2 классификация завершена: %d/%d (не найдено: %d)", classified, totalToClassify, failed)
					}
				}
			}
		}
	}()

	s.log(LogEntry{
		Timestamp: time.Now(),
		Level:     "INFO",
		Message:   "Нормализация данных запущена",
		Endpoint:  "/api/normalize/start",
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":   true,
		"message":   "Нормализация данных запущена",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// handleNormalizationEvents обрабатывает SSE соединение для событий нормализации
func (s *Server) handleNormalizationEvents(w http.ResponseWriter, r *http.Request) {
	// Устанавливаем заголовки для SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")

	// Проверяем поддержку Flusher
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Отправляем начальное событие
	fmt.Fprintf(w, "data: %s\n\n", "{\"type\":\"connected\",\"message\":\"Connected to normalization events\"}")
	flusher.Flush()

	// Слушаем события из канала
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event := <-s.normalizerEvents:
			// Форматируем событие как JSON
			eventJSON := fmt.Sprintf("{\"type\":\"log\",\"message\":%q,\"timestamp\":%q}",
				event, time.Now().Format(time.RFC3339))
			if _, err := fmt.Fprintf(w, "data: %s\n\n", eventJSON); err != nil {
				log.Printf("Ошибка отправки SSE события: %v", err)
				return
			}
			flusher.Flush()
		case <-ticker.C:
			// Отправляем heartbeat для поддержания соединения
			if _, err := fmt.Fprintf(w, ": heartbeat\n\n"); err != nil {
				log.Printf("Ошибка отправки heartbeat: %v", err)
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			// Клиент отключился
			log.Printf("SSE клиент отключился: %v", r.Context().Err())
			return
		}
	}
}

// NormalizationStatus теперь определен в internal/domain/models и доступен через алиас в server/models.go

// handleNormalizationStatus возвращает текущий статус нормализации
func (s *Server) handleNormalizationStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.normalizerMutex.RLock()
	isRunning := s.normalizerRunning
	startTime := s.normalizerStartTime
	processed := s.normalizerProcessed
	success := s.normalizerSuccess
	errors := s.normalizerErrors
	s.normalizerMutex.RUnlock()

	// Получаем реальное количество записей в catalog_items для расчета total
	var totalCatalogItems int
	err := s.db.QueryRow("SELECT COUNT(*) FROM catalog_items").Scan(&totalCatalogItems)
	if err != nil {
		// Если не удалось получить, используем значение по умолчанию
		log.Printf("Ошибка получения количества записей из catalog_items: %v", err)
		totalCatalogItems = 0
	}

	// Получаем количество записей в normalized_data
	var totalNormalized int
	err = s.db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&totalNormalized)
	if err != nil {
		// Таблица может не существовать или быть пустой - это нормально
		log.Printf("Ошибка получения количества нормализованных записей: %v", err)
		totalNormalized = 0
	}

	// Получаем метрики КПВЭД классификации (считаем группы, а не записи)
	// ВАЖНО: normalized_data находится в основной БД (s.db), а не в normalizedDB
	var kpvedClassified, kpvedTotal int
	var kpvedProgress float64
	if s.db != nil {
		// Считаем количество групп с классификацией КПВЭД
		err = s.db.QueryRow(`
			SELECT COUNT(DISTINCT normalized_name || '|' || category)
			FROM normalized_data
			WHERE kpved_code IS NOT NULL AND kpved_code != '' AND TRIM(kpved_code) != ''
		`).Scan(&kpvedClassified)
		if err != nil {
			log.Printf("Ошибка получения количества классифицированных по КПВЭД: %v", err)
			kpvedClassified = 0
		}

		// Считаем общее количество групп
		err = s.db.QueryRow(`
			SELECT COUNT(DISTINCT normalized_name || '|' || category)
			FROM normalized_data
		`).Scan(&kpvedTotal)
		if err != nil {
			log.Printf("Ошибка получения общего количества групп для КПВЭД: %v", err)
			kpvedTotal = 0
		}

		if kpvedTotal > 0 {
			kpvedProgress = float64(kpvedClassified) / float64(kpvedTotal) * 100
		}
	}

	// Используем processed из мьютекса, если процесс запущен, иначе из БД
	if !isRunning {
		processed = totalNormalized
	}

	// Используем реальное количество записей из catalog_items для расчета total
	// Если totalCatalogItems = 0, используем processed как total (для случая когда БД пустая)
	total := totalCatalogItems
	if total == 0 && processed > 0 {
		total = processed
	}

	// Проверяем, действительно ли процесс завершился
	progressPercent := 0.0
	if total > 0 {
		progressPercent = float64(processed) / float64(total) * 100
		if progressPercent > 100 {
			progressPercent = 100
		}
	}

	// Если processed >= total, процесс завершен
	if isRunning && total > 0 && processed >= total {
		// Завершаем процесс сразу, если все записи обработаны
		s.normalizerMutex.Lock()
		s.normalizerRunning = false
		s.normalizerMutex.Unlock()
		isRunning = false
		progressPercent = 100.0
	} else if isRunning && progressPercent >= 100 {
		// Если прогресс 100% и процесс "запущен", но нет активности - завершаем через таймаут
		if !startTime.IsZero() {
			elapsed := time.Since(startTime)
			// Если прошло более 10 секунд и прогресс 100%, считаем завершенным
			if elapsed > 10*time.Second {
				s.normalizerMutex.Lock()
				s.normalizerRunning = false
				s.normalizerMutex.Unlock()
				isRunning = false
			}
		}
	}

	status := NormalizationStatus{
		IsRunning:       isRunning,
		Progress:        progressPercent,
		Processed:       processed,
		Total:           total,
		Success:         success,
		Errors:          errors,
		CurrentStep:     "Не запущено",
		Logs:            []string{},
		KpvedClassified: kpvedClassified,
		KpvedTotal:      kpvedTotal,
		KpvedProgress:   kpvedProgress,
	}

	if isRunning {
		status.CurrentStep = "Выполняется нормализация..."

		// Добавляем время начала и прошедшее время
		if !startTime.IsZero() {
			status.StartTime = startTime.Format(time.RFC3339)
			elapsed := time.Since(startTime)
			status.ElapsedTime = elapsed.String()

			// Рассчитываем скорость обработки
			if elapsed.Seconds() > 0 {
				status.Rate = float64(processed) / elapsed.Seconds()
			}
		}
	} else if progressPercent >= 100 {
		status.CurrentStep = "Нормализация завершена"
		// Добавляем финальную статистику
		if !startTime.IsZero() {
			status.StartTime = startTime.Format(time.RFC3339)
			elapsed := time.Since(startTime)
			status.ElapsedTime = elapsed.String()
			if elapsed.Seconds() > 0 {
				status.Rate = float64(processed) / elapsed.Seconds()
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// handleNormalizationStop останавливает процесс нормализации
func (s *Server) handleNormalizationStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.normalizerMutex.Lock()
	wasRunning := s.normalizerRunning
	s.normalizerRunning = false
	s.normalizerMutex.Unlock()

	if wasRunning {
		s.normalizerEvents <- "Нормализация остановлена пользователем"
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "Normalization stopped",
	})
}

// handleNormalizationStats возвращает статистику нормализации
func (s *Server) handleNormalizationStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем статистику из normalized_data
	// Статистика показывает количество исправленных элементов (каждая запись - это исправленный элемент
	// с разложенными по колонкам/атрибутам размерами, брендами и т.д.)
	var totalItems int
	var totalItemsWithAttributes int // Количество элементов с извлеченными атрибутами
	var lastNormalizedAt sql.NullString
	var categoryStats map[string]int = make(map[string]int)

	// Считаем все исправленные элементы (записи в normalized_data)
	err := s.db.QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&totalItems)
	if err != nil {
		log.Printf("Ошибка получения количества исправленных элементов: %v", err)
		totalItems = 0
	}

	// Считаем количество элементов, у которых есть извлеченные атрибуты (размеры, бренды и т.д.)
	err = s.db.QueryRow(`
		SELECT COUNT(DISTINCT normalized_item_id) 
		FROM normalized_item_attributes
	`).Scan(&totalItemsWithAttributes)
	if err != nil {
		// Таблица атрибутов может не существовать - это нормально
		log.Printf("Ошибка получения количества элементов с атрибутами: %v", err)
		totalItemsWithAttributes = 0
	}

	// Получаем время последней нормализации
	err = s.db.QueryRow("SELECT MAX(created_at) FROM normalized_data").Scan(&lastNormalizedAt)
	if err != nil {
		log.Printf("Ошибка получения времени последней нормализации: %v", err)
	}

	// Получаем статистику по категориям (из поля category)
	rows, err := s.db.Query(`
		SELECT 
			category,
			COUNT(*) as count
		FROM normalized_data
		WHERE category IS NOT NULL AND category != ''
		GROUP BY category
		ORDER BY count DESC
	`)
	if err != nil {
		log.Printf("Ошибка получения статистики по категориям: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var category string
			var count int
			if err := rows.Scan(&category, &count); err != nil {
				log.Printf("Ошибка сканирования категории: %v", err)
				continue
			}
			categoryStats[category] = count
		}
		if err := rows.Err(); err != nil {
			log.Printf("Ошибка при итерации по категориям: %v", err)
		}
	}

	// Вычисляем количество объединенных элементов (дубликатов, которые были объединены)
	// mergedItems = общее количество - количество уникальных групп по normalized_reference
	var uniqueGroups int
	if err := s.db.QueryRow("SELECT COUNT(DISTINCT normalized_reference) FROM normalized_data").Scan(&uniqueGroups); err != nil {
		log.Printf("Ошибка получения количества уникальных групп: %v", err)
		uniqueGroups = 0
	}
	mergedItems := totalItems - uniqueGroups
	if mergedItems < 0 {
		mergedItems = 0
	}

	stats := map[string]interface{}{
		"totalItems":               totalItems,               // Количество исправленных элементов
		"totalItemsWithAttributes": totalItemsWithAttributes, // Количество элементов с извлеченными атрибутами
		"totalGroups":              uniqueGroups,             // Количество уникальных групп (для совместимости)
		"categories":               categoryStats,
		"mergedItems":              mergedItems, // Количество объединенных дубликатов
	}

	// Добавляем timestamp последней нормализации, если он есть
	if lastNormalizedAt.Valid && lastNormalizedAt.String != "" {
		stats["last_normalized_at"] = lastNormalizedAt.String
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleNormalizationGroups возвращает список уникальных групп с пагинацией
func (s *Server) handleNormalizationGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	category := query.Get("category")
	search := query.Get("search")
	kpvedCode := query.Get("kpved_code")
	includeAI := query.Get("include_ai") == "true"

	// Валидация параметров пагинации
	page, limit, err := ValidatePaginationParams(r, 1, 20, 100)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	offset := (page - 1) * limit

	// Строим SQL запрос для получения уникальных групп
	baseQuery := `
		SELECT normalized_name, normalized_reference, category, COUNT(*) as merged_count`

	if includeAI {
		baseQuery += `, AVG(COALESCE(NULLIF(quality_score, 0), NULLIF(ai_confidence, 0))) as avg_confidence, MAX(processing_level) as processing_level`
	}

	// Всегда включаем КПВЭД поля (берем первое значение из группы)
	baseQuery += `, MAX(kpved_code) as kpved_code, MAX(kpved_name) as kpved_name, AVG(kpved_confidence) as kpved_confidence`

	// Добавляем timestamp последней нормализации для группы
	baseQuery += `, MAX(created_at) as last_normalized_at`

	baseQuery += `
		FROM normalized_data
		WHERE 1=1
	`
	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT normalized_name, category
			FROM normalized_data
			WHERE 1=1
	`

	// Параметры для prepared statement
	var args []interface{}
	var countArgs []interface{}

	// Добавляем фильтр по категории
	if category != "" {
		baseQuery += " AND category = ?"
		countQuery += " AND category = ?"
		args = append(args, category)
		countArgs = append(countArgs, category)
	}

	// Добавляем поиск по нормализованному имени
	if search != "" {
		baseQuery += " AND normalized_name LIKE ?"
		countQuery += " AND normalized_name LIKE ?"
		searchParam := "%" + search + "%"
		args = append(args, searchParam)
		countArgs = append(countArgs, searchParam)
	}

	// Добавляем фильтр по КПВЭД коду
	if kpvedCode != "" {
		baseQuery += " AND kpved_code = ?"
		countQuery += " AND kpved_code = ?"
		args = append(args, kpvedCode)
		countArgs = append(countArgs, kpvedCode)
	}

	// Группировка и сортировка для основного запроса
	baseQuery += " GROUP BY normalized_name, normalized_reference, category"
	baseQuery += " ORDER BY merged_count DESC, normalized_name ASC"
	baseQuery += " LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	// Закрываем подзапрос для подсчета
	countQuery += " GROUP BY normalized_name, category)"

	// Получаем общее количество групп
	var totalGroups int
	err = s.db.QueryRow(countQuery, countArgs...).Scan(&totalGroups)
	if err != nil {
		log.Printf("Ошибка получения количества групп: %v", err)
		http.Error(w, "Failed to count groups", http.StatusInternalServerError)
		return
	}

	// Получаем группы
	rows, err := s.db.Query(baseQuery, args...)
	if err != nil {
		log.Printf("Ошибка выполнения запроса групп: %v", err)
		http.Error(w, "Failed to fetch groups", http.StatusInternalServerError)
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
				log.Printf("Ошибка сканирования группы: %v", err)
				continue
			}
		} else {
			if err := rows.Scan(&g.NormalizedName, &g.NormalizedReference, &g.Category, &g.MergedCount,
				&g.KpvedCode, &g.KpvedName, &g.KpvedConfidence, &lastNormalizedAt); err != nil {
				log.Printf("Ошибка сканирования группы: %v", err)
				continue
			}
		}
		if lastNormalizedAt.Valid && lastNormalizedAt.String != "" {
			g.LastNormalizedAt = &lastNormalizedAt.String
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при итерации по группам: %v", err)
	}

	// Вычисляем общее количество страниц
	totalPages := (totalGroups + limit - 1) / limit

	response := map[string]interface{}{
		"groups":     groups,
		"total":      totalGroups,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleNormalizationGroupItems возвращает исходные записи для конкретной группы
func (s *Server) handleNormalizationGroupItems(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	normalizedName := query.Get("normalized_name")
	category := query.Get("category")
	includeAI := query.Get("include_ai") == "true"
	includeAttributes := true
	if val := query.Get("include_attributes"); val != "" {
		includeAttributes = val == "true"
	}

	page := 1
	if pageParam := query.Get("page"); pageParam != "" {
		if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
			page = parsed
		}
	}

	var (
		limit           int
		applyPagination bool
	)
	if limitParam := query.Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil {
			if parsed < 1 {
				parsed = 1
			}
			if parsed > 500 {
				parsed = 500
			}
			limit = parsed
			applyPagination = true
		}
	}

	search := strings.TrimSpace(query.Get("search"))
	sortBy := strings.ToLower(query.Get("sort_by"))
	sortOrder := strings.ToLower(query.Get("sort_order"))

	validSortColumns := map[string]string{
		"source_name":      "source_name",
		"source_reference": "source_reference",
		"code":             "code",
		"created_at":       "created_at",
		"ai_confidence":    "ai_confidence",
	}

	sortColumn := "source_name"
	if column, ok := validSortColumns[sortBy]; ok {
		sortColumn = column
	}

	if sortColumn == "ai_confidence" {
		includeAI = true
	}

	sortDirection := "ASC"
	if sortOrder == "desc" {
		sortDirection = "DESC"
	}

	// Получаем базовую информацию о группе (без учета поиска)
	var normalizedRef sql.NullString
	var mergedCount sql.NullInt64
	var groupKpvedCode, groupKpvedName sql.NullString
	var groupKpvedConfidence sql.NullFloat64

	err := s.db.QueryRow(`
		SELECT normalized_reference, merged_count, kpved_code, kpved_name, kpved_confidence
		FROM normalized_data
		WHERE normalized_name = ? AND category = ?
		ORDER BY id
		LIMIT 1
	`, normalizedName, category).Scan(&normalizedRef, &mergedCount, &groupKpvedCode, &groupKpvedName, &groupKpvedConfidence)
	if err == sql.ErrNoRows {
		http.Error(w, "Group not found", http.StatusNotFound)
		return
	}
	if err != nil {
		log.Printf("Ошибка получения базовой информации группы: %v", err)
		http.Error(w, "Failed to fetch group info", http.StatusInternalServerError)
		return
	}

	whereClauses := []string{"normalized_name = ?", "category = ?"}
	params := []interface{}{normalizedName, category}
	if search != "" {
		like := "%" + search + "%"
		whereClauses = append(whereClauses, "(source_reference LIKE ? OR source_name LIKE ? OR code LIKE ?)")
		params = append(params, like, like, like)
	}
	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM normalized_data WHERE %s`, whereSQL)
	countParams := append([]interface{}{}, params...)
	var totalItems int
	if err := s.db.QueryRow(countQuery, countParams...).Scan(&totalItems); err != nil {
		log.Printf("Ошибка подсчета элементов группы: %v", err)
		http.Error(w, "Failed to count group items", http.StatusInternalServerError)
		return
	}

	if totalItems == 0 && search == "" {
		http.Error(w, "Group items not found", http.StatusNotFound)
		return
	}

	if !applyPagination {
		if totalItems > 0 {
			limit = totalItems
		} else {
			limit = 0
		}
		page = 1
	}

	offset := 0
	if applyPagination && limit > 0 {
		offset = (page - 1) * limit
		if offset >= totalItems {
			// Пересчитываем страницу, если пользователь запросил страницу вне диапазона
			lastPage := totalItems / limit
			if totalItems%limit != 0 {
				lastPage++
			}
			if lastPage < 1 {
				lastPage = 1
			}
			page = lastPage
			offset = (page - 1) * limit
		}
	}

	if normalizedName == "" || category == "" {
		http.Error(w, "normalized_name and category are required", http.StatusBadRequest)
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

	// Всегда включаем КПВЭД поля
	sqlQuery += `, kpved_code, kpved_name, kpved_confidence`

	sqlQuery += `
		FROM normalized_data
		WHERE normalized_name = ? AND category = ?
		ORDER BY source_name
	`

	var queryParams []interface{}
	queryParams = append(queryParams, params...)

	orderClause := fmt.Sprintf(" ORDER BY %s %s", sortColumn, sortDirection)
	if applyPagination && limit > 0 {
		sqlQuery += orderClause + " LIMIT ? OFFSET ?"
		queryParams = append(queryParams, limit, offset)
	} else {
		sqlQuery += orderClause
	}

	rows, err := s.db.Query(sqlQuery, queryParams...)
	if err != nil {
		log.Printf("Ошибка получения записей группы: %v", err)
		http.Error(w, "Failed to fetch group items", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []map[string]interface{}{}

	for rows.Next() {
		var id int
		var sourceRef, sourceName, code, normName, normRef, cat string
		var mCount int
		var createdAt time.Time
		var aiConfidence *float64
		var aiReasoning *string
		var processingLevel *string
		var kpvedCode, kpvedName *string
		var kpvedConfidence *float64

		item := map[string]interface{}{}

		if includeAI {
			if err := rows.Scan(&id, &sourceRef, &sourceName, &code, &normName, &normRef, &cat, &mCount, &createdAt,
				&aiConfidence, &aiReasoning, &processingLevel, &kpvedCode, &kpvedName, &kpvedConfidence); err != nil {
				log.Printf("Ошибка сканирования записи: %v", err)
				continue
			}
			item = map[string]interface{}{
				"id":               id,
				"source_reference": sourceRef,
				"source_name":      sourceName,
				"code":             code,
				"created_at":       createdAt,
			}
			if aiConfidence != nil {
				item["ai_confidence"] = *aiConfidence
			}
			if aiReasoning != nil {
				item["ai_reasoning"] = *aiReasoning
			}
			if processingLevel != nil {
				item["processing_level"] = *processingLevel
			}
		} else {
			if err := rows.Scan(&id, &sourceRef, &sourceName, &code, &normName, &normRef, &cat, &mCount, &createdAt,
				&kpvedCode, &kpvedName, &kpvedConfidence); err != nil {
				log.Printf("Ошибка сканирования записи: %v", err)
				continue
			}
			item = map[string]interface{}{
				"id":               id,
				"source_reference": sourceRef,
				"source_name":      sourceName,
				"code":             code,
				"created_at":       createdAt,
			}
		}

		// Добавляем КПВЭД поля если они есть
		if kpvedCode != nil {
			item["kpved_code"] = *kpvedCode
		}
		if kpvedName != nil {
			item["kpved_name"] = *kpvedName
		}
		if kpvedConfidence != nil {
			item["kpved_confidence"] = *kpvedConfidence
		}

		normalizedRef = sql.NullString{String: normRef, Valid: true}
		mergedCount = sql.NullInt64{Int64: int64(mCount), Valid: true}

		if includeAttributes {
			attributes, err := s.db.GetItemAttributes(id)
			if err != nil {
				log.Printf("Ошибка получения атрибутов для элемента %d: %v", id, err)
			} else if len(attributes) > 0 {
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
					})
				}
				item["attributes"] = attrsJSON
			}
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при итерации по записям: %v", err)
	}

	totalPages := 1
	if applyPagination && limit > 0 {
		totalPages = (totalItems + limit - 1) / limit
		if totalPages < 1 {
			totalPages = 1
		}
	}

	response := map[string]interface{}{
		"normalized_name":      normalizedName,
		"normalized_reference": normalizedRef.String,
		"category":             category,
		"merged_count":         mergedCount.Int64,
		"items":                items,
		"total":                totalItems,
		"page":                 page,
		"total_pages":          totalPages,
		"limit":                limit,
		"search":               search,
	}

	if groupKpvedCode.Valid {
		response["kpved_code"] = groupKpvedCode.String
	}
	if groupKpvedName.Valid {
		response["kpved_name"] = groupKpvedName.String
	}
	if groupKpvedConfidence.Valid {
		response["kpved_confidence"] = groupKpvedConfidence.Float64
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleNormalizationItemAttributes возвращает атрибуты для конкретного нормализованного товара
func (s *Server) handleNormalizationItemAttributes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Извлекаем ID из пути /api/normalization/item-attributes/{id}
	path := r.URL.Path
	parts := strings.Split(path, "/")
	if len(parts) < 5 {
		http.Error(w, "Item ID is required", http.StatusBadRequest)
		return
	}

	itemIDStr := parts[len(parts)-1]
	itemID, err := ValidateIDPathParam(itemIDStr, "item_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid item ID: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем атрибуты
	attributes, err := s.db.GetItemAttributes(itemID)
	if err != nil {
		log.Printf("Ошибка получения атрибутов для элемента %d: %v", itemID, err)
		http.Error(w, "Failed to fetch attributes", http.StatusInternalServerError)
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

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleNormalizationExportGroup экспортирует группу в CSV или JSON формате
func (s *Server) handleNormalizationExportGroup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	normalizedName := query.Get("normalized_name")
	category := query.Get("category")
	format := query.Get("format")

	if normalizedName == "" || category == "" {
		http.Error(w, "normalized_name and category are required", http.StatusBadRequest)
		return
	}

	// По умолчанию CSV формат
	if format == "" {
		format = "csv"
	}

	// Получаем данные группы
	sqlQuery := `
		SELECT id, source_reference, source_name, code,
		       normalized_name, normalized_reference, category,
		       created_at, ai_confidence, ai_reasoning, processing_level
		FROM normalized_data
		WHERE normalized_name = ? AND category = ?
		ORDER BY source_name
	`

	rows, err := s.db.Query(sqlQuery, normalizedName, category)
	if err != nil {
		log.Printf("Ошибка получения записей группы для экспорта: %v", err)
		http.Error(w, "Failed to fetch group items", http.StatusInternalServerError)
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
			log.Printf("Ошибка сканирования записи для экспорта: %v", err)
			continue
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при итерации по записям: %v", err)
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
		http.Error(w, "Invalid format. Supported formats: csv, json", http.StatusBadRequest)
	}
}

// Обработчики баз данных перенесены в server_database_handlers.go

// getNomenclatureStatus возвращает статус обработки номенклатуры
func (s *Server) getNomenclatureStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем статистику из БД
	dbStats, err := s.getNomenclatureDBStats(s.normalizedDB)
	if err != nil {
		LogError(r.Context(), err, "Failed to get nomenclature DB stats")
		s.handleHTTPError(w, r, NewInternalError("не удалось получить статистику БД", err))
		return
	}

	// Получаем статистику из процессора, если он запущен
	var response NomenclatureStatusResponse
	response.DBStats = dbStats

	s.processorMutex.RLock()
	processor := s.nomenclatureProcessor
	if processor == nil {
		s.processorMutex.RUnlock()
		response.Processing = false
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}
	s.processorMutex.RUnlock()

	// processor уже проверен выше, здесь он гарантированно != nil
	stats := processor.GetStats()
	if stats != nil && stats.Total > 0 {
		// Проверяем, идет ли обработка
		// Обработка активна, если:
		// 1. Есть необработанные записи (Processed < Total)
		// 2. Время начала установлено
		// 3. С момента начала прошло не более 5 минут без обновлений (защита от зависших процессов)
		isProcessing := stats.Processed < stats.Total && !stats.StartTime.IsZero()

		// Дополнительная проверка: если прошло более 5 минут без обновлений, считаем обработку завершенной
		if isProcessing {
			elapsed := time.Since(stats.StartTime)
			// Если прошло более 5 минут и нет прогресса, считаем обработку завершенной
			if elapsed > 5*time.Minute && stats.Processed == 0 {
				isProcessing = false
			}
		}

		response.Processing = isProcessing
		response.CurrentStats = &ProcessingStatsResponse{
			Total:      stats.Total,
			Processed:  stats.Processed,
			Successful: stats.Successful,
			Failed:     stats.Failed,
			StartTime:  stats.StartTime,
			MaxWorkers: processor.GetConfig().MaxWorkers,
		}
	} else {
		response.Processing = false
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getProcessingStatus был удален - используйте getNomenclatureStatus напрямую
// Эта функция больше не используется и была заменена на getNomenclatureStatus
//nolint:unused // Функция удалена, комментарий оставлен для справки

// getNomenclatureRecentRecords возвращает последние обработанные записи
func (s *Server) getNomenclatureRecentRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметр limit из запроса (по умолчанию 15)
	limit, err := ValidateIntParam(r, "limit", 15, 1, 100)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Запрос для получения последних записей
	query := `
		SELECT id, name, normalized_name, kpved_code, kpved_name, 
		       processing_status, processed_at
		FROM catalog_items
		WHERE processing_status IN ('completed', 'error')
		ORDER BY COALESCE(processed_at, last_processed_at, created_at) DESC
		LIMIT ?
	`

	// Используем Query для получения нескольких строк
	rows, err := s.normalizedDB.Query(query, limit)
	if err != nil {
		LogError(r.Context(), err, "Failed to get recent nomenclature records")
		s.handleHTTPError(w, r, NewInternalError("не удалось получить последние записи", err))
		return
	}
	defer rows.Close()

	var records []RecentRecord
	for rows.Next() {
		var record RecentRecord
		var processedAtStr sql.NullString

		err := rows.Scan(
			&record.ID,
			&record.OriginalName,
			&record.NormalizedName,
			&record.KpvedCode,
			&record.KpvedName,
			&record.Status,
			&processedAtStr,
		)
		if err != nil {
			log.Printf("Error scanning recent record: %v", err)
			continue
		}

		// Парсим время обработки, если оно есть
		if processedAtStr.Valid && processedAtStr.String != "" {
			if parsedTime, err := time.Parse(time.RFC3339, processedAtStr.String); err == nil {
				record.ProcessedAt = &parsedTime
			}
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		LogError(r.Context(), err, "Error iterating recent nomenclature records")
		s.handleHTTPError(w, r, NewInternalError("ошибка при итерации последних записей", err))
		return
	}

	response := RecentRecordsResponse{
		Records: records,
		Total:   len(records),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// getNomenclaturePendingRecords возвращает необработанные записи
func (s *Server) getNomenclaturePendingRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметры из запроса
	limit := 50
	limit, err := ValidateIntParam(r, "limit", 50, 1, 500)
	if err != nil {
		limit = 50 // Используем значение по умолчанию при ошибке
	}

	offset, err := ValidateIntParam(r, "offset", 0, 0, 0)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	// Запрос для получения необработанных записей
	query := `
		SELECT id, name, 
		       COALESCE(processing_status, 'pending') as status,
		       created_at
		FROM catalog_items
		WHERE processing_status IS NULL OR processing_status = 'pending'
		ORDER BY id ASC
		LIMIT ? OFFSET ?
	`

	rows, err := s.normalizedDB.Query(query, limit, offset)
	if err != nil {
		LogError(r.Context(), err, "Failed to get pending nomenclature records")
		s.handleHTTPError(w, r, NewInternalError("не удалось получить необработанные записи", err))
		return
	}
	defer rows.Close()

	var records []PendingRecord
	for rows.Next() {
		var record PendingRecord
		var createdAtStr string

		err := rows.Scan(
			&record.ID,
			&record.OriginalName,
			&record.Status,
			&createdAtStr,
		)
		if err != nil {
			LogWarn(r.Context(), "Error scanning pending record", "error", err)
			continue
		}

		// Парсим время создания
		if parsedTime, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
			record.CreatedAt = parsedTime
		} else if parsedTime, err := time.Parse("2006-01-02 15:04:05", createdAtStr); err == nil {
			record.CreatedAt = parsedTime
		} else {
			record.CreatedAt = time.Now()
		}

		records = append(records, record)
	}

	if err = rows.Err(); err != nil {
		LogError(r.Context(), err, "Error iterating pending nomenclature records")
		s.handleHTTPError(w, r, NewInternalError("ошибка при итерации необработанных записей", err))
		return
	}

	// Получаем общее количество необработанных записей
	var total int
	err = s.normalizedDB.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE processing_status IS NULL OR processing_status = 'pending'").Scan(&total)
	if err != nil {
		LogWarn(r.Context(), "Error getting total pending count, using records length", "error", err)
		total = len(records)
	}

	response := PendingRecordsResponse{
		Records: records,
		Total:   total,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// serveNomenclatureStatusPage отдает HTML страницу мониторинга
func (s *Server) serveNomenclatureStatusPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Пытаемся прочитать файл из static директории
	htmlContent, err := os.ReadFile("./static/nomenclature_status.html")
	if err != nil {
		// Если файл не найден, используем встроенную версию
		htmlContent = []byte(getNomenclatureStatusPageHTML())
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(htmlContent)
}

// getNomenclatureStatusPageHTML возвращает встроенную HTML страницу мониторинга
func getNomenclatureStatusPageHTML() string {
	return `<!DOCTYPE html>
<html lang="ru">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Мониторинг обработки номенклатуры</title>
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css" rel="stylesheet">
    <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.4.0/css/all.min.css">
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        .card { transition: transform 0.2s; height: 100%; }
        .card:hover { transform: translateY(-5px); }
        .progress { height: 10px; }
        #progressChart { max-height: 300px; }
        .table-responsive { max-height: 400px; }
        .last-updated { font-size: 0.8rem; color: #6c757d; }
        .thread-status { display: flex; align-items: center; margin-bottom: 10px; }
        .thread-indicator { width: 12px; height: 12px; border-radius: 50%; margin-right: 10px; }
        .thread-active { background-color: #28a745; animation: pulse 1.5s infinite; }
        .thread-idle { background-color: #6c757d; }
        @keyframes pulse { 0% { opacity: 1; } 50% { opacity: 0.5; } 100% { opacity: 1; } }
        .dark-mode { background-color: #212529; color: #f8f9fa; }
        .dark-mode .card { background-color: #343a40; color: #f8f9fa; }
        .dark-mode .table { color: #f8f9fa; }
    </style>
</head>
<body>
    <div class="container-fluid py-4">
        <div class="d-flex justify-content-between align-items-center mb-4">
            <h1 class="h3"><i class="fas fa-database me-2"></i>Мониторинг обработки номенклатуры</h1>
            <div>
                <button class="btn btn-sm btn-outline-secondary me-2" id="themeToggle"><i class="fas fa-moon"></i> Тема</button>
                <button class="btn btn-sm btn-outline-primary" id="refreshBtn"><i class="fas fa-sync-alt"></i> Обновить</button>
            </div>
        </div>
        <div class="row mb-4">
            <div class="col-md-3 col-sm-6 mb-3">
                <div class="card bg-primary text-white">
                    <div class="card-body">
                        <div class="d-flex justify-content-between">
                            <div><h4 class="card-title" id="totalRecords">0</h4><p class="card-text">Всего записей</p></div>
                            <div class="align-self-center"><i class="fas fa-table fa-2x"></i></div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-3 col-sm-6 mb-3">
                <div class="card bg-success text-white">
                    <div class="card-body">
                        <div class="d-flex justify-content-between">
                            <div><h4 class="card-title" id="processedRecords">0</h4><p class="card-text">Обработано</p></div>
                            <div class="align-self-center"><i class="fas fa-check-circle fa-2x"></i></div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-3 col-sm-6 mb-3">
                <div class="card bg-warning text-dark">
                    <div class="card-body">
                        <div class="d-flex justify-content-between">
                            <div><h4 class="card-title" id="pendingRecords">0</h4><p class="card-text">Ожидают обработки</p></div>
                            <div class="align-self-center"><i class="fas fa-clock fa-2x"></i></div>
                        </div>
                    </div>
                </div>
            </div>
            <div class="col-md-3 col-sm-6 mb-3">
                <div class="card bg-danger text-white">
                    <div class="card-body">
                        <div class="d-flex justify-content-between">
                            <div><h4 class="card-title" id="errorRecords">0</h4><p class="card-text">С ошибками</p></div>
                            <div class="align-self-center"><i class="fas fa-exclamation-circle fa-2x"></i></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <div class="row">
            <div class="col-lg-8">
                <div class="card mb-4">
                    <div class="card-header d-flex justify-content-between align-items-center">
                        <h5 class="card-title mb-0"><i class="fas fa-tasks me-2"></i>Статус обработки</h5>
                        <span class="badge bg-success d-none" id="processingBadge">Активна</span>
                        <span class="badge bg-secondary" id="idleBadge">Не активна</span>
                    </div>
                    <div class="card-body">
                        <div class="mb-3">
                            <div class="d-flex justify-content-between mb-1">
                                <span>Прогресс обработки</span>
                                <span id="progressPercent">0%</span>
                            </div>
                            <div class="progress">
                                <div id="progressBar" class="progress-bar progress-bar-striped progress-bar-animated" role="progressbar" style="width: 0%"></div>
                            </div>
                            <div class="text-center mt-2">
                                <span id="progressText">Обработано <strong>0</strong> из <strong>0</strong> записей</span>
                            </div>
                        </div>
                        <div class="row">
                            <div class="col-md-4">
                                <div class="card bg-light">
                                    <div class="card-body text-center py-3">
                                        <h6>Время начала</h6>
                                        <p class="mb-0" id="startTime">-</p>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-4">
                                <div class="card bg-light">
                                    <div class="card-body text-center py-3">
                                        <h6>Прошедшее время</h6>
                                        <p class="mb-0" id="elapsedTime">-</p>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-4">
                                <div class="card bg-light">
                                    <div class="card-body text-center py-3">
                                        <h6>Оставшееся время</h6>
                                        <p class="mb-0" id="remainingTime">-</p>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="card mb-4">
                    <div class="card-header">
                        <h5 class="card-title mb-0"><i class="fas fa-chart-line me-2"></i>График прогресса обработки</h5>
                    </div>
                    <div class="card-body">
                        <canvas id="progressChart"></canvas>
                    </div>
                </div>
            </div>
            <div class="col-lg-4">
                <div class="card mb-4">
                    <div class="card-header">
                        <h5 class="card-title mb-0"><i class="fas fa-microchip me-2"></i>Потоки обработки</h5>
                    </div>
                    <div class="card-body">
                        <div class="thread-status">
                            <div class="thread-indicator thread-idle" id="thread1Indicator"></div>
                            <div><strong>Поток 1</strong><div class="text-muted small" id="thread1Status">Ожидание</div></div>
                        </div>
                        <div class="thread-status">
                            <div class="thread-indicator thread-idle" id="thread2Indicator"></div>
                            <div><strong>Поток 2</strong><div class="text-muted small" id="thread2Status">Ожидание</div></div>
                        </div>
                        <div class="mt-3">
                            <div class="d-flex justify-content-between">
                                <span>Скорость обработки:</span>
                                <span id="processingSpeed">0 записей/мин</span>
                            </div>
                        </div>
                    </div>
                </div>
                <div class="card mb-4">
                    <div class="card-header">
                        <h5 class="card-title mb-0"><i class="fas fa-cogs me-2"></i>Управление обработкой</h5>
                    </div>
                    <div class="card-body">
                        <div class="d-grid gap-2">
                            <button class="btn btn-success" id="startBtn"><i class="fas fa-play me-2"></i>Запустить обработку</button>
                            <button class="btn btn-warning" id="stopBtn" disabled><i class="fas fa-stop me-2"></i>Остановить обработку</button>
                        </div>
                        <div class="mt-3">
                            <div class="form-check form-switch">
                                <input class="form-check-input" type="checkbox" id="autoRefresh" checked>
                                <label class="form-check-label" for="autoRefresh">Автообновление (каждые 5 сек)</label>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        <div class="card mb-4">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="card-title mb-0"><i class="fas fa-clock me-2"></i>Необработанные номенклатуры</h5>
                <div>
                    <span class="badge bg-warning me-2" id="pendingCountBadge">0 записей</span>
                    <button class="btn btn-sm btn-success" id="startProcessingBtn"><i class="fas fa-play me-1"></i>Запустить обработку</button>
                </div>
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped table-hover">
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Наименование</th>
                                <th>Статус</th>
                                <th>Дата создания</th>
                            </tr>
                        </thead>
                        <tbody id="pendingRecords">
                            <tr><td colspan="4" class="text-center text-muted">Загрузка данных...</td></tr>
                        </tbody>
                    </table>
                </div>
                <div class="mt-3 d-flex justify-content-between align-items-center">
                    <div>
                        <button class="btn btn-sm btn-outline-secondary" id="prevPageBtn" disabled><i class="fas fa-chevron-left"></i> Назад</button>
                        <span class="mx-2" id="pageInfo">Страница 1</span>
                        <button class="btn btn-sm btn-outline-secondary" id="nextPageBtn">Вперед <i class="fas fa-chevron-right"></i></button>
                    </div>
                    <div>
                        <span class="text-muted small">Показано: <span id="pendingShown">0</span> из <span id="pendingTotal">0</span></span>
                    </div>
                </div>
            </div>
        </div>
        <div class="card">
            <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="card-title mb-0"><i class="fas fa-history me-2"></i>Последние обработанные записи</h5>
                <span class="last-updated">Обновлено: <span id="lastUpdateTime">-</span></span>
            </div>
            <div class="card-body">
                <div class="table-responsive">
                    <table class="table table-striped table-hover">
                        <thead>
                            <tr>
                                <th>ID</th>
                                <th>Исходное наименование</th>
                                <th>Нормализованное наименование</th>
                                <th>Код КПВЭД</th>
                                <th>Статус</th>
                                <th>Время обработки</th>
                            </tr>
                        </thead>
                        <tbody id="recentRecords">
                            <tr><td colspan="6" class="text-center text-muted">Загрузка данных...</td></tr>
                        </tbody>
                    </table>
                </div>
            </div>
        </div>
    </div>
    <script>
        let refreshInterval, progressChart, processingActive = false, progressData = { labels: [], values: [] };
        let pendingPage = 0;
        const pendingPageSize = 50;
        document.addEventListener('DOMContentLoaded', function() {
            initializeChart();
            loadData();
            setupEventListeners();
            startAutoRefresh();
        });
        function initializeChart() {
            const ctx = document.getElementById('progressChart').getContext('2d');
            progressChart = new Chart(ctx, {
                type: 'line',
                data: { labels: progressData.labels, datasets: [{
                    label: 'Прогресс обработки (%)',
                    data: progressData.values,
                    borderColor: 'rgb(75, 192, 192)',
                    backgroundColor: 'rgba(75, 192, 192, 0.1)',
                    tension: 0.3,
                    fill: true
                }]},
                options: {
                    responsive: true,
                    maintainAspectRatio: false,
                    scales: { y: { beginAtZero: true, max: 100, ticks: { callback: function(value) { return value + '%'; } } } }
                }
            });
        }
        function setupEventListeners() {
            document.getElementById('refreshBtn').addEventListener('click', loadData);
            document.getElementById('startBtn').addEventListener('click', startProcessing);
            document.getElementById('startProcessingBtn').addEventListener('click', startProcessing);
            document.getElementById('stopBtn').addEventListener('click', stopProcessing);
            document.getElementById('themeToggle').addEventListener('click', toggleTheme);
            document.getElementById('autoRefresh').addEventListener('change', toggleAutoRefresh);
            document.getElementById('prevPageBtn').addEventListener('click', () => {
                if (pendingPage > 0) {
                    pendingPage--;
                    loadPendingRecords();
                }
            });
            document.getElementById('nextPageBtn').addEventListener('click', () => {
                pendingPage++;
                loadPendingRecords();
            });
        }
        function loadData() {
            fetch('/api/nomenclature/status')
                .then(response => response.json())
                .then(data => updateUI(data))
                .catch(error => { console.error('Ошибка загрузки данных:', error); showError('Не удалось загрузить данные'); });
            loadRecentRecords();
            loadPendingRecords();
        }
        function loadPendingRecords() {
            const offset = pendingPage * pendingPageSize;
            fetch('/api/nomenclature/pending?limit=' + pendingPageSize + '&offset=' + offset)
                .then(response => {
                    if (!response.ok) throw new Error('Ошибка загрузки необработанных записей');
                    return response.json();
                })
                .then(data => {
                    updatePendingRecordsTable(data.records);
                    updatePendingPagination(data.total, data.records.length);
                })
                .catch(error => {
                    console.error('Ошибка загрузки необработанных записей:', error);
                    const tbody = document.getElementById('pendingRecords');
                    tbody.innerHTML = '<tr><td colspan="4" class="text-center text-danger">Ошибка загрузки данных</td></tr>';
                });
        }
        function updatePendingRecordsTable(records) {
            const tbody = document.getElementById('pendingRecords');
            if (!records || records.length === 0) {
                tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted">Нет необработанных записей</td></tr>';
                return;
            }
            tbody.innerHTML = '';
            records.forEach(record => {
                const statusBadge = '<span class="badge bg-warning">Ожидает обработки</span>';
                const createdAt = new Date(record.created_at).toLocaleString();
                const row = document.createElement('tr');
                row.innerHTML = '<td>' + record.id + '</td>' +
                    '<td>' + escapeHtml(record.original_name || '-') + '</td>' +
                    '<td>' + statusBadge + '</td>' +
                    '<td>' + createdAt + '</td>';
                tbody.appendChild(row);
            });
        }
        function updatePendingPagination(total, shown) {
            document.getElementById('pendingTotal').textContent = total;
            document.getElementById('pendingShown').textContent = shown;
            document.getElementById('pendingCountBadge').textContent = total + ' записей';
            const totalPages = Math.ceil(total / pendingPageSize);
            const currentPage = pendingPage + 1;
            document.getElementById('pageInfo').textContent = 'Страница ' + currentPage + ' из ' + (totalPages || 1);
            document.getElementById('prevPageBtn').disabled = pendingPage === 0;
            document.getElementById('nextPageBtn').disabled = currentPage >= totalPages || shown < pendingPageSize;
        }
        function loadRecentRecords() {
            fetch('/api/nomenclature/recent?limit=15')
                .then(response => {
                    if (!response.ok) throw new Error('Ошибка загрузки последних записей');
                    return response.json();
                })
                .then(data => updateRecentRecordsTable(data.records))
                .catch(error => {
                    console.error('Ошибка загрузки последних записей:', error);
                    const tbody = document.getElementById('recentRecords');
                    tbody.innerHTML = '<tr><td colspan="6" class="text-center text-danger">Ошибка загрузки данных</td></tr>';
                });
        }
        function updateRecentRecordsTable(records) {
            const tbody = document.getElementById('recentRecords');
            if (!records || records.length === 0) {
                tbody.innerHTML = '<tr><td colspan="6" class="text-center text-muted">Нет обработанных записей</td></tr>';
                return;
            }
            tbody.innerHTML = '';
            records.forEach(record => {
                const statusBadge = record.status === 'completed' ? 
                    '<span class="badge bg-success">Успешно</span>' : 
                    '<span class="badge bg-danger">Ошибка</span>';
                const processedAt = record.processed_at ? 
                    new Date(record.processed_at).toLocaleString() : '-';
                const row = document.createElement('tr');
                row.innerHTML = '<td>' + record.id + '</td>' +
                    '<td>' + escapeHtml(record.original_name || '-') + '</td>' +
                    '<td>' + escapeHtml(record.normalized_name || '-') + '</td>' +
                    '<td>' + escapeHtml(record.kpved_code || '-') + (record.kpved_name ? '<br><small class="text-muted">' + escapeHtml(record.kpved_name) + '</small>' : '') + '</td>' +
                    '<td>' + statusBadge + '</td>' +
                    '<td>' + processedAt + '</td>';
                tbody.appendChild(row);
            });
        }
        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
        function updateUI(data) {
            document.getElementById('totalRecords').textContent = data.db_stats.total.toLocaleString();
            document.getElementById('processedRecords').textContent = data.db_stats.completed.toLocaleString();
            document.getElementById('pendingRecords').textContent = data.db_stats.pending.toLocaleString();
            document.getElementById('errorRecords').textContent = data.db_stats.errors.toLocaleString();
            processingActive = data.processing;
            if (processingActive) {
                document.getElementById('processingBadge').classList.remove('d-none');
                document.getElementById('idleBadge').classList.add('d-none');
                document.getElementById('startBtn').disabled = true;
                document.getElementById('startProcessingBtn').disabled = true;
                document.getElementById('stopBtn').disabled = false;
            } else {
                document.getElementById('processingBadge').classList.add('d-none');
                document.getElementById('idleBadge').classList.remove('d-none');
                document.getElementById('startBtn').disabled = false;
                document.getElementById('startProcessingBtn').disabled = false;
                document.getElementById('stopBtn').disabled = true;
            }
            if (data.current_stats) {
                const stats = data.current_stats;
                const progressPercent = stats.total > 0 ? Math.floor((stats.processed / stats.total) * 100) : 0;
                document.getElementById('progressPercent').textContent = progressPercent + '%';
                document.getElementById('progressBar').style.width = progressPercent + '%';
                document.getElementById('progressText').innerHTML = 'Обработано <strong>' + stats.processed.toLocaleString() + '</strong> из <strong>' + stats.total.toLocaleString() + '</strong> записей';
                document.getElementById('startTime').textContent = new Date(stats.start_time).toLocaleString();
                const elapsed = Math.floor((Date.now() - new Date(stats.start_time).getTime()) / 1000);
                document.getElementById('elapsedTime').textContent = formatTime(elapsed);
                if (progressPercent < 100 && stats.processed > 0) {
                    const remaining = Math.floor((elapsed * (stats.total - stats.processed)) / stats.processed);
                    document.getElementById('remainingTime').textContent = formatTime(remaining);
                } else {
                    document.getElementById('remainingTime').textContent = '-';
                }
                const speed = stats.processed > 0 ? Math.floor((stats.processed / elapsed) * 60) : 0;
                document.getElementById('processingSpeed').textContent = speed > 0 ? speed + ' записей/мин' : '0 записей/мин';
                updateProgressChart(progressPercent);
                const threadActive = processingActive;
                const maxWorkers = stats.max_workers || 2;
                // Обновляем статус потоков в зависимости от max_workers
                for (let i = 1; i <= 2; i++) {
                    const indicator = document.getElementById('thread' + i + 'Indicator');
                    const status = document.getElementById('thread' + i + 'Status');
                    if (i <= maxWorkers) {
                        if (threadActive) {
                            indicator.classList.add('thread-active');
                            indicator.classList.remove('thread-idle');
                            status.textContent = 'Активен';
                        } else {
                            indicator.classList.remove('thread-active');
                            indicator.classList.add('thread-idle');
                            status.textContent = 'Ожидание';
                        }
                    } else {
                        indicator.classList.remove('thread-active');
                        indicator.classList.add('thread-idle');
                        status.textContent = 'Не используется';
                    }
                }
            } else {
                document.getElementById('progressPercent').textContent = '0%';
                document.getElementById('progressBar').style.width = '0%';
                document.getElementById('progressText').innerHTML = 'Обработано <strong>0</strong> из <strong>0</strong> записей';
                document.getElementById('startTime').textContent = '-';
                document.getElementById('elapsedTime').textContent = '-';
                document.getElementById('remainingTime').textContent = '-';
                document.getElementById('processingSpeed').textContent = '0 записей/мин';
            }
            document.getElementById('lastUpdateTime').textContent = new Date().toLocaleTimeString();
        }
        function updateProgressChart(progress) {
            const now = new Date();
            const timeLabel = now.getHours() + ':' + now.getMinutes().toString().padStart(2, '0');
            progressData.labels.push(timeLabel);
            progressData.values.push(progress);
            if (progressData.labels.length > 10) {
                progressData.labels.shift();
                progressData.values.shift();
            }
            progressChart.update();
        }
        function startProcessing() {
            if (processingActive) {
                showWarning('Обработка уже запущена');
                return;
            }
            if (!confirm('Запустить обработку необработанных номенклатур?')) {
                return;
            }
            const startBtn = document.getElementById('startBtn');
            const startProcessingBtn = document.getElementById('startProcessingBtn');
            const originalText = startBtn.innerHTML;
            const originalText2 = startProcessingBtn.innerHTML;
            startBtn.disabled = true;
            startProcessingBtn.disabled = true;
            startBtn.innerHTML = '<i class="fas fa-spinner fa-spin me-2"></i>Запуск...';
            startProcessingBtn.innerHTML = '<i class="fas fa-spinner fa-spin me-1"></i>Запуск...';
            fetch('/api/nomenclature/process', { method: 'POST', headers: { 'Content-Type': 'application/json' } })
                .then(response => {
                    if (response.ok) {
                        showSuccess('Обработка запущена');
                        setTimeout(() => loadData(), 1000);
                    } else {
                        throw new Error('Ошибка запуска обработки');
                    }
                })
                .catch(error => {
                    console.error('Ошибка:', error);
                    showError('Не удалось запустить обработку');
                    startBtn.disabled = false;
                    startProcessingBtn.disabled = false;
                    startBtn.innerHTML = originalText;
                    startProcessingBtn.innerHTML = originalText2;
                });
        }
        function stopProcessing() {
            showWarning('Функция остановки обработки будет реализована в будущем');
        }
        function toggleTheme() {
            document.body.classList.toggle('dark-mode');
            const themeIcon = document.querySelector('#themeToggle i');
            if (document.body.classList.contains('dark-mode')) {
                themeIcon.classList.remove('fa-moon');
                themeIcon.classList.add('fa-sun');
            } else {
                themeIcon.classList.remove('fa-sun');
                themeIcon.classList.add('fa-moon');
            }
        }
        function startAutoRefresh() {
            refreshInterval = setInterval(() => {
                if (document.getElementById('autoRefresh').checked) { loadData(); }
            }, 5000);
        }
        function toggleAutoRefresh() {
            if (document.getElementById('autoRefresh').checked) { startAutoRefresh(); } else { clearInterval(refreshInterval); }
        }
        function formatTime(seconds) {
            const hours = Math.floor(seconds / 3600);
            const minutes = Math.floor((seconds % 3600) / 60);
            if (hours > 0) { return hours + 'ч ' + minutes + 'м'; } else { return minutes + 'м'; }
        }
        function showSuccess(message) { console.log('Успех:', message); }
        function showError(message) { console.error('Ошибка:', message); }
        function showWarning(message) { console.warn('Предупреждение:', message); }
    </script>
</body>
</html>`
}

// handleStartClientNormalization запускает нормализацию для клиента
func (s *Server) handleStartClientNormalization(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	if s.serviceDB == nil {
		s.writeJSONError(w, r, "Service database not available", http.StatusInternalServerError)
		return
	}

	s.normalizerMutex.Lock()
	if s.normalizerRunning {
		s.normalizerMutex.Unlock()
		s.writeJSONError(w, r, "Normalization is already running", http.StatusBadRequest)
		return
	}

	// Читаем параметры из запроса
	var req struct {
		DatabasePath string `json:"database_path"`
		AllActive    bool   `json:"all_active"`
		UseKpved     bool   `json:"use_kpved"`
		UseOkpd2     bool   `json:"use_okpd2"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Если тело пустое, используем значения по умолчанию
		// По умолчанию обрабатываем все активные БД проекта
		req.AllActive = true
		req.UseKpved = false
		req.UseOkpd2 = false
	}

	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		s.normalizerMutex.Unlock()
		log.Printf("Error getting project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		s.normalizerMutex.Unlock()
		log.Printf("Project %d does not belong to client %d", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	var databasesToProcess []*database.ProjectDatabase

	// По умолчанию обрабатываем все активные БД проекта (all_active = true)
	// Если явно указано all_active = false, тогда требуется database_path
	if !req.AllActive && req.DatabasePath != "" {
		// Используем конкретную БД по пути (только если явно указано all_active=false)
		// Получаем все БД проекта для проверки принадлежности
		allDatabases, err := s.serviceDB.GetProjectDatabases(projectID, false)
		if err != nil {
			s.normalizerMutex.Unlock()
			log.Printf("Error getting all project databases for project %d: %v", projectID, err)
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get project databases: %v", err), http.StatusInternalServerError)
			return
		}

		var foundDB *database.ProjectDatabase
		for _, db := range allDatabases {
			if pathsMatch(db.FilePath, req.DatabasePath) {
				foundDB = db
				break
			}
		}

		if foundDB == nil {
			s.normalizerMutex.Unlock()
			log.Printf("Database %s does not belong to project %d", req.DatabasePath, projectID)
			s.writeJSONError(w, r, "Database does not belong to this project", http.StatusBadRequest)
			return
		}

		databasesToProcess = []*database.ProjectDatabase{foundDB}
	} else {
		// По умолчанию получаем все активные БД проекта
		databases, err := s.serviceDB.GetProjectDatabases(projectID, true)
		if err != nil {
			s.normalizerMutex.Unlock()
			log.Printf("Error getting active project databases for project %d: %v", projectID, err)
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get project databases: %v", err), http.StatusInternalServerError)
			return
		}
		if len(databases) == 0 {
			s.normalizerMutex.Unlock()
			log.Printf("No active databases found for project %d", projectID)
			s.writeJSONError(w, r, "No active databases found for this project", http.StatusBadRequest)
			return
		}
		databasesToProcess = databases
	}

	// Создаем context для управления жизненным циклом нормализации
	// Отменяем предыдущий context, если он существует
	s.normalizerMutex.Lock()
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

	// Возвращаем успешный ответ перед запуском горутины
	s.writeJSONResponse(w, r, map[string]interface{}{
		"status":          "started",
		"message":         "Normalization started",
		"databases_count": len(databasesToProcess),
	}, http.StatusOK)

	// Запускаем нормализацию для всех выбранных БД
	go func() {
		startTime := time.Now()
		ctx := s.normalizerCtx // Используем контекст для graceful shutdown
		defer func() {
			// Обработка паники и очистка состояния
			if rec := recover(); rec != nil {
				LogNormalizationPanic(projectID, rec, string(debug.Stack()))
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
		// Проверка контекста в начале для раннего выхода
		select {
		case <-ctx.Done():
			LogNormalizationStopped(clientID, projectID, "context cancelled before start", 0)
			return
		default:
		}

		// Для контрагентов и номенклатуры используем параллельную обработку БД
		if database.IsCounterpartyProjectType(project.ProjectType) {
			s.processCounterpartyDatabasesParallel(databasesToProcess, clientID, projectID)
		} else {
			// Для номенклатуры также используем параллельную обработку
			s.processNomenclatureDatabasesParallel(databasesToProcess, clientID, projectID, project, req)
		}
	}()
	// Ответ уже был отправлен ранее
}

// processLegacyNormalizationDatabase обрабатывает нормализацию одной БД номенклатуры
// Вынесено в отдельную функцию для корректной работы defer при закрытии БД
func (s *Server) processLegacyNormalizationDatabase(clientID, projectID int, projectDB *database.ProjectDatabase, sessionID int, project *database.ClientProject, req struct {
	DatabasePath string `json:"database_path"`
	AllActive    bool   `json:"all_active"`
	UseKpved     bool   `json:"use_kpved"`
	UseOkpd2     bool   `json:"use_okpd2"`
}) {
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
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		return
	}
	if sessionID <= 0 {
		LogNormalizationError(clientID, projectID, fmt.Errorf("invalid sessionID: %d", sessionID), "Invalid session ID")
		return
	}

	// Проверяем доступность файла БД перед обработкой
	if _, err := os.Stat(projectDB.FilePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			LogNormalizationError(clientID, projectID, err, "Database file not found")
			s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
			select {
			case s.normalizerEvents <- fmt.Sprintf("Файл БД %s не найден, пропущена", projectDB.Name):
			default:
			}
			return
		}
		LogNormalizationError(clientID, projectID, err, "Error checking database file")
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		return
	}

	// Открываем подключение к базе данных
	sourceDB, err := database.NewDB(projectDB.FilePath)
	if err != nil {
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

	// Нормализация номенклатуры
	// Получаем все записи из catalog_items
	items, err := sourceDB.GetAllCatalogItems()
	if err != nil {
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		LogNormalizationError(clientID, projectID, err, "Failed to read data from database")
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка чтения данных из %s: %v", projectDB.FilePath, err):
		default:
		}
		return
	}

	if len(items) == 0 {
		s.serviceDB.UpdateNormalizationSession(sessionID, "completed", nil)
		LogInfo(context.Background(), "No items to normalize",
			"client_id", clientID,
			"project_id", projectID,
			"database_id", projectDB.ID)
		select {
		case s.normalizerEvents <- fmt.Sprintf("База данных %s пуста, нормализация завершена", projectDB.Name):
		default:
		}
		return
	}

	LogInfo(context.Background(), "Starting normalization",
		"client_id", clientID,
		"project_id", projectID,
		"database_id", projectDB.ID,
		"database_name", projectDB.Name,
		"items_count", len(items))
	select {
	case s.normalizerEvents <- fmt.Sprintf("Начало нормализации БД: %s (%d записей)", projectDB.Name, len(items)):
	default:
	}

	// Создаем клиентский нормализатор
	clientNormalizer := normalization.NewClientNormalizerWithConfig(clientID, projectID, sourceDB, s.serviceDB, s.normalizerEvents, s.workerConfigManager)
	clientNormalizer.SetSessionID(sessionID)

	// Проверяем статус сессии перед запуском
	session, err := s.serviceDB.GetNormalizationSession(sessionID)
	if err != nil || session == nil || session.Status != "running" {
		log.Printf("Session %d is not running, skipping normalization", sessionID)
		return
	}

	// Обновляем активность сессии перед началом
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
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка нормализации БД %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Ошибка клиентской нормализации для %s: %v", projectDB.FilePath, err)
		if s.notificationService != nil {
			ctx := context.Background()
			clientIDPtr := &clientID
			projectIDPtr := &projectID
			_, _ = s.notificationService.AddNotification(ctx, services.NotificationTypeError, "Ошибка нормализации", fmt.Sprintf("Ошибка нормализации БД %s: %v", projectDB.Name, err), clientIDPtr, projectIDPtr, map[string]interface{}{"database_id": projectDB.ID, "session_id": sessionID, "error": err.Error()})
		}
		return
	}

	// Сохраняем результаты нормализации в normalized_data
	if result != nil && result.Groups != nil && len(result.Groups) > 0 {
		normalizedItems, itemAttributes := s.convertClientGroupsToNormalizedItems(result.Groups, projectID, sessionID)
		if len(normalizedItems) > 0 && s.normalizedDB != nil {
			_, saveErr := s.normalizedDB.InsertNormalizedItemsWithAttributesBatch(normalizedItems, itemAttributes, &sessionID, &projectID)
			if saveErr != nil {
				log.Printf("Ошибка сохранения нормализованных данных для БД %s: %v", projectDB.FilePath, saveErr)
				select {
				case s.normalizerEvents <- fmt.Sprintf("⚠ Ошибка сохранения результатов нормализации БД %s: %v", projectDB.Name, saveErr):
				default:
				}
			} else {
				log.Printf("Сохранено %d нормализованных записей для проекта %d из БД %s", len(normalizedItems), projectID, projectDB.Name)
				select {
				case s.normalizerEvents <- fmt.Sprintf("✓ Сохранено %d нормализованных записей из БД %s", len(normalizedItems), projectDB.Name):
				default:
				}
			}
		}
	}

	// Обновляем сессию как completed
	s.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
	select {
	case s.normalizerEvents <- fmt.Sprintf("Нормализация БД %s завершена успешно", projectDB.Name):
	default:
	}

	// Отправляем уведомление об успешном завершении
	if s.notificationService != nil {
		ctx := context.Background()
		clientIDPtr := &clientID
		projectIDPtr := &projectID
		_, _ = s.notificationService.AddNotification(ctx, services.NotificationTypeSuccess, "Нормализация завершена", fmt.Sprintf("Нормализация БД %s завершена успешно", projectDB.Name), clientIDPtr, projectIDPtr, map[string]interface{}{
			"database_id":   projectDB.ID,
			"database_name": projectDB.Name,
			"session_id":    sessionID,
		})
	}

	// Классификация КПВЭД и ОКПД2 после нормализации
	normalizedDBPath := filepath.Join(filepath.Dir(projectDB.FilePath), "normalized_"+filepath.Base(projectDB.FilePath))
	if _, err := os.Stat(normalizedDBPath); err == nil {
		normalizedDB, err := database.NewDB(normalizedDBPath)
		if err == nil {
			defer normalizedDB.Close()

			if req.UseKpved && s.hierarchicalClassifier != nil {
				s.classifyKpvedForDatabase(normalizedDB, projectDB.Name)
			}

			if req.UseOkpd2 {
				s.classifyOkpd2ForDatabase(normalizedDB, projectDB.Name)
			}
		}
	}
}

// isNormalizationStopped проверяет, была ли нормализация остановлена пользователем
// по сообщению об ошибке в результате нормализации
func (s *Server) isNormalizationStopped(result *normalization.CounterpartyNormalizationResult) bool {
	if result == nil {
		return false
	}
	for _, errMsg := range result.Errors {
		if errMsg == normalization.ErrMsgNormalizationStopped {
			return true
		}
	}
	return false
}

// handleCounterpartyNormalizationResult обрабатывает результат нормализации контрагентов:
// обновляет сессию, отправляет события и логирует результаты
func (s *Server) handleCounterpartyNormalizationResult(
	sessionID int,
	result *normalization.CounterpartyNormalizationResult,
	projectDB *database.ProjectDatabase,
	counterparties []*database.CatalogItem,
	clientID, projectID int,
	isResume bool,
) {
	finishedAt := time.Now()
	wasStopped := s.isNormalizationStopped(result)

	if wasStopped {
		// Обновляем сессию как stopped с временем завершения
		s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
		progressPercent := 0.0
		if len(counterparties) > 0 {
			progressPercent = float64(result.TotalProcessed) / float64(len(counterparties)) * 100
		}

		// Формируем сообщение в зависимости от контекста
		contextMsg := "остановлена пользователем"
		if isResume {
			contextMsg = "остановлена пользователем при возобновлении"
		}

		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация контрагентов БД %s %s: обработано %d из %d (%.1f%%)",
			projectDB.Name, contextMsg, result.TotalProcessed, len(counterparties), progressPercent):
		default:
		}

		// Отправляем структурированное событие об остановке (только для processCounterpartyDatabase)
		if !isResume {
			select {
			case s.normalizerEvents <- fmt.Sprintf(`{"type":"database_stopped","database_id":%d,"database_name":%q,"processed":%d,"total":%d,"progress_percent":%.1f,"benchmark_matches":%d,"enriched_count":%d,"duplicate_groups":%d,"client_id":%d,"project_id":%d,"timestamp":%q}`,
				projectDB.ID, projectDB.Name, result.TotalProcessed, len(counterparties), progressPercent,
				result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups, clientID, projectID, time.Now().Format(time.RFC3339)):
			default:
			}
		}

		log.Printf("Нормализация контрагентов %s: обработано %d из %d (%.1f%%) для БД %s",
			contextMsg, result.TotalProcessed, len(counterparties), progressPercent, projectDB.FilePath)
	} else {
		// Обновляем сессию как completed
		s.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация контрагентов БД %s завершена: обработано %d, найдено эталонов %d, дозаполнено %d, групп дублей %d",
			projectDB.Name, result.TotalProcessed, result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups):
		default:
		}

		// Отправляем уведомление об успешном завершении
		if s.notificationService != nil {
			ctx := context.Background()
			clientIDPtr := &clientID
			projectIDPtr := &projectID
			message := fmt.Sprintf("Обработано %d контрагентов, найдено эталонов: %d, дозаполнено: %d, групп дублей: %d",
				result.TotalProcessed, result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups)
			_, _ = s.notificationService.AddNotification(ctx, services.NotificationTypeSuccess, "Нормализация контрагентов завершена", message, clientIDPtr, projectIDPtr, map[string]interface{}{
				"database_id":       projectDB.ID,
				"database_name":     projectDB.Name,
				"session_id":        sessionID,
				"processed":         result.TotalProcessed,
				"benchmark_matches": result.BenchmarkMatches,
				"enriched_count":    result.EnrichedCount,
				"duplicate_groups":  result.DuplicateGroups,
			})
		}

		// Отправляем структурированное событие о завершении (только для processCounterpartyDatabase)
		if !isResume {
			select {
			case s.normalizerEvents <- fmt.Sprintf(`{"type":"database_completed","database_id":%d,"database_name":%q,"processed":%d,"total":%d,"benchmark_matches":%d,"enriched_count":%d,"duplicate_groups":%d,"client_id":%d,"project_id":%d,"timestamp":%q}`,
				projectDB.ID, projectDB.Name, result.TotalProcessed, len(counterparties),
				result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups, clientID, projectID, time.Now().Format(time.RFC3339)):
			default:
			}
		}

	log.Printf("Нормализация контрагентов завершена: обработано %d, найдено эталонов %d, дозаполнено %d, групп дублей %d",
		result.TotalProcessed, result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups)
	}
}

// processNomenclatureDatabasesParallel обрабатывает БД номенклатуры параллельно
// Каждая БД обрабатывается в отдельной горутине с использованием пула воркеров.
//
// Параметры:
//   - databases: список баз данных проекта для обработки
//   - clientID: ID клиента (должен быть > 0)
//   - projectID: ID проекта (должен быть > 0)
//   - project: информация о проекте
//   - req: параметры запроса (UseKpved, UseOkpd2)
//
// Особенности:
//   - Использует семафор для ограничения параллелизма (максимум из конфигурации или 5)
//   - Проверяет наличие активных сессий перед началом обработки
//   - Логирует время обработки каждой БД
//   - Собирает статистику по завершенным, провалившимся и остановленным сессиям
//   - Проверяет остановку нормализации перед обработкой каждой БД
func (s *Server) processNomenclatureDatabasesParallel(databases []*database.ProjectDatabase, clientID, projectID int, project *database.ClientProject, req struct {
	DatabasePath string `json:"database_path"`
	AllActive    bool   `json:"all_active"`
	UseKpved     bool   `json:"use_kpved"`
	UseOkpd2     bool   `json:"use_okpd2"`
}) {
	// Проверяем входные параметры
	if len(databases) == 0 {
		log.Printf("[Nomenclature] No databases to process for project %d", projectID)
		return
	}
	if clientID <= 0 || projectID <= 0 {
		log.Printf("[Nomenclature] Invalid clientID (%d) or projectID (%d), skipping", clientID, projectID)
		return
	}

	// Получаем максимальное количество воркеров из конфигурации или используем дефолт
	maxWorkers := 5 // Дефолтное значение
	if s.workerConfigManager != nil {
		globalMaxWorkers := s.workerConfigManager.GetGlobalMaxWorkers()
		if globalMaxWorkers > 0 {
			maxWorkers = globalMaxWorkers
			log.Printf("[Nomenclature] Using global MaxWorkers=%d for parallel processing", maxWorkers)
		}
	}

	// Ограничиваем максимальным количеством БД
	if len(databases) < maxWorkers {
		maxWorkers = len(databases)
	}

	log.Printf("[Nomenclature] Starting parallel normalization for project %d: %d databases, %d workers", projectID, len(databases), maxWorkers)
	startTime := time.Now()

	// Фильтруем БД без активных сессий перед параллельной обработкой
	var databasesToProcess []*database.ProjectDatabase
	skippedCount := 0
	if s.serviceDB != nil {
		for _, db := range databases {
			session, err := s.serviceDB.GetLastNormalizationSession(db.ID)
			if err == nil && session != nil && session.Status == "running" {
				log.Printf("[Nomenclature] Skipping database %s (active session ID: %d)", db.Name, session.ID)
				skippedCount++
				select {
				case s.normalizerEvents <- fmt.Sprintf("БД %s уже обрабатывается (сессия %d), пропущена", db.Name, session.ID):
				default:
				}
				continue
			}
			databasesToProcess = append(databasesToProcess, db)
		}
		if skippedCount > 0 {
			log.Printf("[Nomenclature] Filtered out %d databases with active sessions, will process %d databases", skippedCount, len(databasesToProcess))
		}
	} else {
		databasesToProcess = databases
	}

	if len(databasesToProcess) == 0 {
		log.Printf("[Nomenclature] No databases to process (all have active sessions or none available)")
		select {
		case s.normalizerEvents <- "Нет доступных БД для обработки (все уже обрабатываются)":
		default:
		}
		return
	}

	// Семафор для ограничения параллелизма
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, projectDB := range databasesToProcess {
		// Проверяем, не нужно ли остановить нормализацию
		if s.shouldStopNormalization() {
			log.Printf("Normalization stopped, skipping database %s", projectDB.FilePath)
			break
		}

		// Захватываем семафор
		semaphore <- struct{}{}
		wg.Add(1)

		// Запускаем обработку БД в отдельной горутине
		go func(db *database.ProjectDatabase) {
			dbStartTime := time.Now()
			defer func() {
				// Освобождаем семафор
				<-semaphore
				wg.Done()

				dbDuration := time.Since(dbStartTime)
				log.Printf("[Nomenclature] Database %s processing completed in %v", db.Name, dbDuration)

				if rec := recover(); rec != nil {
					log.Printf("Panic in nomenclature normalization goroutine for DB %s: %v\n%s", db.FilePath, rec, debug.Stack())
					select {
					case s.normalizerEvents <- fmt.Sprintf("Критическая ошибка нормализации БД %s: %v", db.Name, rec):
					default:
					}
				}
			}()

			// Проверяем доступность файла БД перед обработкой
			if _, err := os.Stat(db.FilePath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					log.Printf("Database file not found: %s, skipping", db.FilePath)
					select {
					case s.normalizerEvents <- fmt.Sprintf("Файл БД %s не найден, пропущена", db.Name):
					default:
					}
					return
				}
				// Другие ошибки тоже пропускаем
				log.Printf("Error checking database file %s: %v, skipping", db.FilePath, err)
				return
			}

			// Пытаемся создать сессию нормализации для этой базы данных (приоритет 0 по умолчанию, таймаут 1 час)
			// Используем атомарную функцию, которая создаст сессию только если нет активных
			sessionID, created, err := s.serviceDB.TryCreateNormalizationSession(db.ID, 0, 3600)
			if err != nil {
				select {
				case s.normalizerEvents <- fmt.Sprintf("Ошибка создания сессии для БД %s: %v", db.FilePath, err):
				default:
				}
				log.Printf("Failed to create normalization session for database %s: %v", db.FilePath, err)
				return
			}
			if !created {
				log.Printf("[Nomenclature] Database %s already has active session, skipping", db.Name)
				select {
				case s.normalizerEvents <- fmt.Sprintf("БД %s уже обрабатывается, пропущена", db.Name):
				default:
				}
				return
			}

			// Обновляем last_used_at
			s.serviceDB.UpdateProjectDatabaseLastUsed(db.ID)

			// Обрабатываем БД в отдельной функции для корректной работы defer
			s.processLegacyNormalizationDatabase(clientID, projectID, db, sessionID, project, req)
		}(projectDB)
	}

	// Ждем завершения всех воркеров
	wg.Wait()
	duration := time.Since(startTime)

	// Проверяем финальный статус остановки
	wasStopped := s.shouldStopNormalization()

	// Получаем статистику по сессиям для этого проекта
	var completedCount, failedCount, stoppedCount int
	if s.serviceDB != nil {
		// Получаем все сессии для баз данных этого проекта
		// Используем databasesToProcess для статистики по обработанным БД
		for _, db := range databasesToProcess {
			session, err := s.serviceDB.GetLastNormalizationSession(db.ID)
			if err == nil && session != nil {
				switch session.Status {
				case "completed":
					completedCount++
				case "failed":
					failedCount++
				case "stopped":
					stoppedCount++
				}
			}
		}
	}

	if wasStopped {
		log.Printf("[Nomenclature] Normalization was stopped for project %d - some databases may not have been processed (duration: %v, completed: %d, failed: %d, stopped: %d)",
			projectID, duration, completedCount, failedCount, stoppedCount)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация остановлена для проекта %d (время работы: %v, завершено БД: %d, ошибок: %d, остановлено: %d)",
			projectID, duration, completedCount, failedCount, stoppedCount):
		default:
		}
	} else {
		// Вычисляем среднее время обработки на БД
		avgTimePerDB := time.Duration(0)
		if completedCount+failedCount > 0 {
			avgTimePerDB = duration / time.Duration(completedCount+failedCount)
		}

		log.Printf("[Nomenclature] All databases processed for project %d: total=%d (skipped=%d), duration=%v, avg_time_per_db=%v, completed: %d, failed: %d",
			projectID, len(databasesToProcess), skippedCount, duration, avgTimePerDB, completedCount, failedCount)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Параллельная нормализация номенклатуры завершена: обработано БД %d/%d (пропущено: %d) за %v (среднее время на БД: %v, успешно: %d, ошибок: %d)",
			len(databasesToProcess), len(databases), skippedCount, duration, avgTimePerDB, completedCount, failedCount):
		default:
		}
	}
}

// processCounterpartyDatabasesParallel обрабатывает БД контрагентов параллельно
// Каждая БД обрабатывается в отдельной горутине с использованием пула воркеров.
// Нормализация контрагентов не зависит от внешних источников обогащения.
//
// Параметры:
//   - databases: список баз данных проекта для обработки
//   - clientID: ID клиента (должен быть > 0)
//   - projectID: ID проекта (должен быть > 0)
//
// Особенности:
//   - Использует семафор для ограничения параллелизма (максимум из конфигурации или 5)
//   - Проверяет наличие активных сессий перед началом обработки
//   - Логирует время обработки каждой БД
//   - Собирает статистику по завершенным, провалившимся и остановленным сессиям
//   - Проверяет остановку нормализации перед обработкой каждой БД
func (s *Server) processCounterpartyDatabasesParallel(databases []*database.ProjectDatabase, clientID, projectID int) {
	// Проверяем входные параметры
	if len(databases) == 0 {
		log.Printf("[Counterparty] No databases to process for project %d", projectID)
		return
	}
	if clientID <= 0 || projectID <= 0 {
		log.Printf("[Counterparty] Invalid clientID (%d) or projectID (%d), skipping", clientID, projectID)
		return
	}

	// Получаем максимальное количество воркеров из конфигурации или используем дефолт
	maxWorkers := 5 // Дефолтное значение
	if s.workerConfigManager != nil {
		globalMaxWorkers := s.workerConfigManager.GetGlobalMaxWorkers()
		if globalMaxWorkers > 0 {
			maxWorkers = globalMaxWorkers
			log.Printf("[Counterparty] Using global MaxWorkers=%d for parallel processing", maxWorkers)
		}
	}

	// Ограничиваем максимальным количеством БД
	if len(databases) < maxWorkers {
		maxWorkers = len(databases)
	}

	log.Printf("[Counterparty] Starting parallel normalization for project %d: %d databases, %d workers", projectID, len(databases), maxWorkers)
	startTime := time.Now()

	// Фильтруем БД без активных сессий перед параллельной обработкой
	var databasesToProcess []*database.ProjectDatabase
	skippedCount := 0
	if s.serviceDB != nil {
		for _, db := range databases {
			session, err := s.serviceDB.GetLastNormalizationSession(db.ID)
			if err == nil && session != nil && session.Status == "running" {
				log.Printf("[Counterparty] Skipping database %s (active session ID: %d)", db.Name, session.ID)
				skippedCount++
				select {
				case s.normalizerEvents <- fmt.Sprintf("БД %s уже обрабатывается (сессия %d), пропущена", db.Name, session.ID):
				default:
				}
				continue
			}
			databasesToProcess = append(databasesToProcess, db)
		}
		if skippedCount > 0 {
			log.Printf("[Counterparty] Filtered out %d databases with active sessions, will process %d databases", skippedCount, len(databasesToProcess))
		}
	} else {
		databasesToProcess = databases
	}

	if len(databasesToProcess) == 0 {
		log.Printf("[Counterparty] No databases to process (all have active sessions or none available)")
		select {
		case s.normalizerEvents <- "Нет доступных БД для обработки (все уже обрабатываются)":
		default:
		}
		return
	}

	// Семафор для ограничения параллелизма
	semaphore := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, projectDB := range databasesToProcess {
		// Проверяем, не нужно ли остановить нормализацию
		if s.shouldStopNormalization() {
			log.Printf("Normalization stopped, skipping database %s", projectDB.FilePath)
			break
		}

		// Захватываем семафор
		semaphore <- struct{}{}
		wg.Add(1)

		// Запускаем обработку БД в отдельной горутине
		go func(db *database.ProjectDatabase) {
			dbStartTime := time.Now()
			defer func() {
				// Освобождаем семафор
				<-semaphore
				wg.Done()

				dbDuration := time.Since(dbStartTime)
				log.Printf("[Counterparty] Database %s processing completed in %v", db.Name, dbDuration)

				if rec := recover(); rec != nil {
					log.Printf("Panic in counterparty normalization goroutine for DB %s: %v\n%s", db.FilePath, rec, debug.Stack())
					select {
					case s.normalizerEvents <- fmt.Sprintf("Критическая ошибка нормализации БД %s: %v", db.Name, rec):
					default:
					}
				}
			}()

			// Проверяем доступность файла БД перед обработкой
			if _, err := os.Stat(db.FilePath); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					log.Printf("Database file not found: %s, skipping", db.FilePath)
					select {
					case s.normalizerEvents <- fmt.Sprintf("Файл БД %s не найден, пропущена", db.Name):
					default:
					}
					return
				}
				// Другие ошибки тоже пропускаем
				log.Printf("Error checking database file %s: %v, skipping", db.FilePath, err)
				return
			}

			s.processCounterpartyDatabase(db, clientID, projectID)
		}(projectDB)
	}

	// Ждем завершения всех воркеров
	wg.Wait()
	duration := time.Since(startTime)

	// Проверяем финальный статус остановки
	wasStopped := s.shouldStopNormalization()

	// Получаем статистику по сессиям для этого проекта
	var completedCount, failedCount, stoppedCount int
	if s.serviceDB != nil {
		// Получаем все сессии для баз данных этого проекта
		// Используем databasesToProcess для статистики по обработанным БД
		for _, db := range databasesToProcess {
			session, err := s.serviceDB.GetLastNormalizationSession(db.ID)
			if err == nil && session != nil {
				switch session.Status {
				case "completed":
					completedCount++
				case "failed":
					failedCount++
				case "stopped":
					stoppedCount++
				}
			}
		}
	}

	if wasStopped {
		log.Printf("[Counterparty] Normalization was stopped for project %d - some databases may not have been processed (duration: %v, completed: %d, failed: %d, stopped: %d)",
			projectID, duration, completedCount, failedCount, stoppedCount)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация остановлена для проекта %d (время работы: %v, завершено БД: %d, ошибок: %d, остановлено: %d)",
			projectID, duration, completedCount, failedCount, stoppedCount):
		default:
		}
	} else {
		// Вычисляем среднее время обработки на БД
		avgTimePerDB := time.Duration(0)
		if completedCount+failedCount > 0 {
			avgTimePerDB = duration / time.Duration(completedCount+failedCount)
		}

		log.Printf("[Counterparty] All databases processed for project %d: total=%d (skipped=%d), duration=%v, avg_time_per_db=%v, completed: %d, failed: %d",
			projectID, len(databasesToProcess), skippedCount, duration, avgTimePerDB, completedCount, failedCount)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Параллельная нормализация контрагентов завершена: обработано БД %d/%d (пропущено: %d) за %v (среднее время на БД: %v, успешно: %d, ошибок: %d)",
			len(databasesToProcess), len(databases), skippedCount, duration, avgTimePerDB, completedCount, failedCount):
		default:
		}
	}
}

// processCounterpartyDatabase обрабатывает одну БД контрагентов
func (s *Server) processCounterpartyDatabase(projectDB *database.ProjectDatabase, clientID, projectID int) {
	// Проверяем, что projectDB.ID валиден
	if projectDB.ID <= 0 {
		log.Printf("Invalid project database ID (%d) for database %s, skipping", projectDB.ID, projectDB.FilePath)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка: невалидный ID базы данных %s", projectDB.Name):
		default:
		}
		return
	}

	// Пытаемся создать сессию нормализации для этой базы данных (приоритет 0 по умолчанию, таймаут 1 час)
	// Используем атомарную функцию, которая создаст сессию только если нет активных
	sessionID, created, err := s.serviceDB.TryCreateNormalizationSession(projectDB.ID, 0, 3600)
	if err != nil {
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка создания сессии для БД %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Failed to create normalization session for database %s: %v", projectDB.FilePath, err)
		return
	}
	if !created {
		log.Printf("[Counterparty] Database %s already has active session, skipping", projectDB.Name)
		select {
		case s.normalizerEvents <- fmt.Sprintf("БД %s уже обрабатывается, пропущена", projectDB.Name):
		default:
		}
		return
	}

	// Обновляем last_used_at
	s.serviceDB.UpdateProjectDatabaseLastUsed(projectDB.ID)

	// Открываем подключение к базе данных
	// Проверяем, что путь к файлу не пустой
	if projectDB.FilePath == "" {
		log.Printf("Empty file path for database ID %d, skipping", projectDB.ID)
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка: пустой путь к файлу БД %s", projectDB.Name):
		default:
		}
		return
	}

	sourceDB, err := database.NewDB(projectDB.FilePath)
	if err != nil {
		// Обновляем сессию как failed
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка открытия БД %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Failed to open database %s: %v", projectDB.FilePath, err)
		return
	}
	defer func() {
		if err := sourceDB.Close(); err != nil {
			log.Printf("Error closing database %s: %v", projectDB.FilePath, err)
		}
	}()

	// Получаем контрагентов из справочника "Контрагенты"
	uploads, err := sourceDB.GetAllUploads()
	if err != nil {
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка получения выгрузок из %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Failed to get uploads from %s: %v", projectDB.FilePath, err)
		return
	}

	var counterparties []*database.CatalogItem
	for _, upload := range uploads {
		items, _, err := sourceDB.GetCatalogItemsByUpload(upload.ID, []string{"Контрагенты"}, 0, 0)
		if err != nil {
			log.Printf("Failed to get counterparties from upload %d: %v", upload.ID, err)
			select {
			case s.normalizerEvents <- fmt.Sprintf("Ошибка получения контрагентов из выгрузки %d БД %s: %v", upload.ID, projectDB.Name, err):
			default:
			}
			continue
		}
		counterparties = append(counterparties, items...)
	}

	if len(counterparties) == 0 {
		log.Printf("No counterparties found in database %s (project %d)", projectDB.Name, projectID)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Контрагенты не найдены в БД %s", projectDB.Name):
		default:
		}
		// Обновляем сессию как completed (нет данных для обработки)
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
		return
	}

	// Фильтруем пустые контрагенты (без названия)
	validCounterparties := make([]*database.CatalogItem, 0, len(counterparties))
	emptyCount := 0
	for _, cp := range counterparties {
		if cp != nil && cp.Name != "" && strings.TrimSpace(cp.Name) != "" {
			validCounterparties = append(validCounterparties, cp)
		} else {
			emptyCount++
		}
	}

	if emptyCount > 0 {
		log.Printf("Filtered out %d empty counterparties from database %s (project %d)", emptyCount, projectDB.Name, projectID)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Отфильтровано %d пустых контрагентов из БД %s", emptyCount, projectDB.Name):
		default:
		}
	}

	if len(validCounterparties) == 0 {
		log.Printf("No valid counterparties found in database %s (project %d) after filtering", projectDB.Name, projectID)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Валидные контрагенты не найдены в БД %s после фильтрации", projectDB.Name):
		default:
		}
		// Обновляем сессию как completed (нет валидных данных для обработки)
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
		return
	}

	counterparties = validCounterparties

	// Логируем предупреждение для больших объемов данных
	if len(counterparties) > 10000 {
		log.Printf("[Counterparty] Warning: Large dataset detected for database %s: %d counterparties. Processing may take significant time.", projectDB.Name, len(counterparties))
		select {
		case s.normalizerEvents <- fmt.Sprintf("Внимание: большой объем данных в БД %s (%d контрагентов). Обработка может занять значительное время.", projectDB.Name, len(counterparties)):
		default:
		}
	}

	log.Printf("Starting counterparty normalization for project %d with %d counterparties from %s", projectID, len(counterparties), projectDB.FilePath)
	select {
	case s.normalizerEvents <- fmt.Sprintf("Начало нормализации контрагентов БД: %s (%d записей)", projectDB.Name, len(counterparties)):
	default:
	}

	// Проверяем, не нужно ли остановить нормализацию перед началом обработки
	if s.shouldStopNormalization() {
		log.Printf("Normalization stopped before processing counterparties from %s", projectDB.FilePath)
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация остановлена перед обработкой БД %s", projectDB.Name):
		default:
		}
		return
	}

	// Создаем нормализатор через сервис, который правильно настроен с benchmarkFinder
	var counterpartyNormalizer *normalization.CounterpartyNormalizer
	if s.counterpartyService != nil && s.multiProviderClient != nil {
		// Используем сервис, который правильно настроен с benchmarkFinder
		counterpartyNormalizer = s.counterpartyService.CreateNormalizer(clientID, projectID, s.multiProviderClient)
	} else if s.serviceDB != nil && s.multiProviderClient != nil {
		// Fallback: создаем напрямую без benchmarkFinder
		ctx := context.Background()
		counterpartyNormalizer = normalization.NewCounterpartyNormalizer(s.serviceDB, clientID, projectID, s.normalizerEvents, ctx, s.multiProviderClient, nil)
		log.Printf("Warning: Using fallback counterparty normalizer (without benchmarkFinder) for database %s", projectDB.Name)
	} else {
		// Критическая ошибка: невозможно создать нормализатор
		log.Printf("Error: Cannot create counterparty normalizer - counterpartyService, serviceDB or multiProviderClient is nil for database %s", projectDB.Name)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка: невозможно создать нормализатор контрагентов для БД %s - отсутствуют необходимые сервисы", projectDB.Name):
		default:
		}
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", &finishedAt)
		return
	}

	// Проверяем остановку перед началом нормализации
	if counterpartyNormalizer.IsStopped() {
		log.Printf("[Counterparty] Normalization stopped before ProcessNormalization (resume) for database %s (project %d)", projectDB.Name, projectID)
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация контрагентов БД %s остановлена пользователем до начала обработки (возобновление)", projectDB.Name):
		default:
		}
		// Отправляем структурированное событие об остановке
		select {
		case s.normalizerEvents <- fmt.Sprintf(`{"type":"database_stopped","database_id":%d,"database_name":%q,"processed":0,"total":%d,"progress_percent":0.0,"benchmark_matches":0,"enriched_count":0,"duplicate_groups":0,"client_id":%d,"project_id":%d,"timestamp":%q,"reason":"stopped_before_start_resume"}`,
			projectDB.ID, projectDB.Name, len(counterparties), clientID, projectID, time.Now().Format(time.RFC3339)):
		default:
		}
		return
	}

	// Запускаем нормализацию контрагентов (skipNormalized = false для новой сессии)
	result, err := counterpartyNormalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		// Обновляем сессию как failed
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка нормализации контрагентов БД %s: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Ошибка нормализации контрагентов для %s: %v", projectDB.FilePath, err)
	} else {
		// Проверяем, была ли нормализация остановлена пользователем
		wasStopped := false
		for _, errMsg := range result.Errors {
			if errMsg == "Нормализация остановлена пользователем" {
				wasStopped = true
				break
			}
		}

		if wasStopped {
			// Обновляем сессию как stopped с временем завершения
			finishedAt := time.Now()
			s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
			progressPercent := 0.0
			if len(counterparties) > 0 {
				progressPercent = float64(result.TotalProcessed) / float64(len(counterparties)) * 100
			}
			select {
			case s.normalizerEvents <- fmt.Sprintf("Нормализация контрагентов БД %s остановлена пользователем: обработано %d из %d (%.1f%%)",
				projectDB.Name, result.TotalProcessed, len(counterparties), progressPercent):
			default:
			}

			// Отправляем структурированное событие об остановке
			select {
			case s.normalizerEvents <- fmt.Sprintf(`{"type":"database_stopped","database_id":%d,"database_name":%q,"processed":%d,"total":%d,"progress_percent":%.1f,"benchmark_matches":%d,"enriched_count":%d,"duplicate_groups":%d,"client_id":%d,"project_id":%d,"timestamp":%q}`,
				projectDB.ID, projectDB.Name, result.TotalProcessed, len(counterparties), progressPercent,
				result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups, clientID, projectID, time.Now().Format(time.RFC3339)):
			default:
			}
			log.Printf("Нормализация контрагентов остановлена: обработано %d из %d (%.1f%%) для БД %s",
				result.TotalProcessed, len(counterparties), progressPercent, projectDB.FilePath)
		} else {
			// Обновляем сессию как completed
			finishedAt := time.Now()
			s.serviceDB.UpdateNormalizationSession(sessionID, "completed", &finishedAt)
			select {
			case s.normalizerEvents <- fmt.Sprintf("Нормализация контрагентов БД %s завершена: обработано %d, найдено эталонов %d, дозаполнено %d, групп дублей %d",
				projectDB.Name, result.TotalProcessed, result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups):
			default:
			}

			// Отправляем структурированное событие о завершении
			select {
			case s.normalizerEvents <- fmt.Sprintf(`{"type":"database_completed","database_id":%d,"database_name":%q,"processed":%d,"total":%d,"benchmark_matches":%d,"enriched_count":%d,"duplicate_groups":%d,"created_benchmarks":%d,"errors_count":%d,"client_id":%d,"project_id":%d,"timestamp":%q}`,
				projectDB.ID, projectDB.Name, result.TotalProcessed, len(counterparties),
				result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups, result.CreatedBenchmarks, len(result.Errors),
				clientID, projectID, time.Now().Format(time.RFC3339)):
			default:
			}
			log.Printf("Нормализация контрагентов завершена: обработано %d, найдено эталонов %d, дозаполнено %d, групп дублей %d",
				result.TotalProcessed, result.BenchmarkMatches, result.EnrichedCount, result.DuplicateGroups)
		}
	}
}

// handleStopClientNormalization останавливает нормализацию для клиента
func (s *Server) handleStopClientNormalization(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for stop normalization: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (stop normalization)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	s.normalizerMutex.Lock()
	wasRunning := s.normalizerRunning
	if !wasRunning {
		s.normalizerMutex.Unlock()
		log.Printf("Stop normalization requested for client_id=%d, project_id=%d, but normalization is not running", clientID, projectID)
		s.writeJSONResponse(w, r, map[string]interface{}{
			"status":  "not_running",
			"message": "Normalization is not running",
		}, http.StatusOK) // Возвращаем 200, так как желаемое состояние уже достигнуто
		return
	}

	// Останавливаем нормализацию через отмену context
	if s.normalizerCancel != nil {
		s.normalizerCancel()
		s.normalizerCancel = nil
		log.Printf("Cancelled normalization context for client_id=%d, project_id=%d", clientID, projectID)
	}
	s.normalizerRunning = false
	s.normalizerMutex.Unlock()

	log.Printf("Normalization stop signal set for client_id=%d, project_id=%d", clientID, projectID)

	// Останавливаем все активные сессии нормализации для этого проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err == nil {
		runningSessions, err := s.serviceDB.GetRunningSessions()
		if err == nil {
			stoppedCount := 0
			for _, session := range runningSessions {
				// Проверяем, принадлежит ли сессия этому проекту
				for _, db := range databases {
					if session.ProjectDatabaseID == db.ID {
						if err := s.serviceDB.StopNormalizationSession(session.ID); err == nil {
							stoppedCount++
							log.Printf("Stopped normalization session %d for database %d", session.ID, db.ID)
						}
						break
					}
				}
			}
			if stoppedCount > 0 {
				log.Printf("Stopped %d normalization sessions for project %d", stoppedCount, projectID)
			}
		}
	}

	log.Printf("Normalization stopped for client_id=%d, project_id=%d", clientID, projectID)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"status":  "stopped",
		"message": "Normalization stopped",
	}, http.StatusOK)
}

// handleGetClientNormalizationStatus получает статус нормализации для клиента
func (s *Server) handleGetClientNormalizationStatus(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for normalization status: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (get status)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	s.normalizerMutex.RLock()
	isRunning := s.normalizerRunning
	startTime := s.normalizerStartTime
	processed := s.normalizerProcessed
	success := s.normalizerSuccess
	errors := s.normalizerErrors
	s.normalizerMutex.RUnlock()

	// Получаем события нормализации (последние 50)
	var logs []string
	for i := 0; i < 50; i++ {
		select {
		case event := <-s.normalizerEvents:
			logs = append(logs, event)
		default:
			goto done
		}
	}
done:

	// Вычисляем прогресс и время
	var progress float64
	var elapsedTime string
	var rate float64
	var currentStep string

	if isRunning {
		if startTime.IsZero() {
			currentStep = "Инициализация..."
		} else {
			elapsed := time.Since(startTime)
			elapsedTime = elapsed.Round(time.Second).String()
			if processed > 0 && elapsed.Seconds() > 0 {
				rate = float64(processed) / elapsed.Seconds()
			}
			currentStep = "Нормализация в процессе"
			if len(logs) > 0 {
				currentStep = logs[len(logs)-1]
			}
		}
	} else {
		if processed > 0 {
			currentStep = "Нормализация завершена"
			// Вычисляем скорость для завершенной нормализации
			if !startTime.IsZero() {
				elapsed := time.Since(startTime)
				elapsedTime = elapsed.Round(time.Second).String()
				if elapsed.Seconds() > 0 {
					rate = float64(processed) / elapsed.Seconds()
				}
			}
		} else {
			currentStep = "Не запущено"
		}
	}

	// Получаем общее количество для расчета прогресса
	var total int
	var sessionsInfo []map[string]interface{}
	var databasesInfo []map[string]interface{}

	// project уже получен выше, используем его
	if project != nil {
		if database.IsCounterpartyProjectType(project.ProjectType) {
			// Для контрагентов получаем статистику
			stats, err := s.serviceDB.GetNormalizedCounterpartyStats(projectID)
			if err == nil {
				if totalCountVal, ok := stats["total_count"]; ok {
					switch v := totalCountVal.(type) {
					case int:
						total = v
					case int64:
						total = int(v)
					case float64:
						total = int(v)
					}
				}
			}

			// Получаем информацию о сессиях нормализации
			databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
			if err == nil {
				for _, db := range databases {
					dbInfo := map[string]interface{}{
						"id":        db.ID,
						"name":      db.Name,
						"file_path": db.FilePath,
						"is_active": db.IsActive,
					}

					// Получаем последнюю сессию для этой БД
					lastSession, err := s.serviceDB.GetLastNormalizationSession(db.ID)
					if err == nil && lastSession != nil {
						dbInfo["last_session"] = map[string]interface{}{
							"id":         lastSession.ID,
							"status":     lastSession.Status,
							"created_at": lastSession.CreatedAt.Format(time.RFC3339),
							"finished_at": func() interface{} {
								if lastSession.FinishedAt != nil {
									return lastSession.FinishedAt.Format(time.RFC3339)
								}
								return nil
							}(),
						}
					}
					databasesInfo = append(databasesInfo, dbInfo)
				}

				// Получаем активные сессии
				runningSessions, err := s.serviceDB.GetRunningSessions()
				if err == nil {
					for _, session := range runningSessions {
						// Проверяем, принадлежит ли сессия этому проекту
						for _, db := range databases {
							if session.ProjectDatabaseID == db.ID {
								sessionInfo := map[string]interface{}{
									"id":                  session.ID,
									"project_database_id": session.ProjectDatabaseID,
									"database_name":       db.Name,
									"status":              session.Status,
									"created_at":          session.CreatedAt.Format(time.RFC3339),
								}
								if session.FinishedAt != nil {
									sessionInfo["finished_at"] = session.FinishedAt.Format(time.RFC3339)
								}
								sessionsInfo = append(sessionsInfo, sessionInfo)
								break
							}
						}
					}
				}
			}
		} else {
			// Для номенклатуры получаем из БД проекта
			databases, err := s.serviceDB.GetProjectDatabases(projectID, true)
			if err == nil && len(databases) > 0 {
				// Берем первую активную БД для подсчета
				db, err := database.NewDB(databases[0].FilePath)
				if err == nil {
					defer func() {
						if err := db.Close(); err != nil {
							log.Printf("Error closing database in getNormalizationProgress: %v", err)
						}
					}()
					items, err := db.GetAllCatalogItems()
					if err == nil {
						total = len(items)
					}
				}
			}
		}
	}

	if total > 0 {
		progress = float64(processed) / float64(total) * 100
	}

	response := map[string]interface{}{
		"isRunning":   isRunning,
		"progress":    progress,
		"processed":   processed,
		"total":       total,
		"success":     success,
		"errors":      errors,
		"currentStep": currentStep,
		"logs":        logs,
		"startTime":   startTime.Format(time.RFC3339),
		"elapsedTime": elapsedTime,
		"rate":        rate,
		"client_id":   clientID,
		"project_id":  projectID,
	}

	// Добавляем информацию о сессиях и БД для контрагентов
	if project != nil && database.IsCounterpartyProjectType(project.ProjectType) {
		response["sessions"] = sessionsInfo
		response["databases"] = databasesInfo
		response["active_sessions_count"] = len(sessionsInfo)
		response["total_databases_count"] = len(databasesInfo)
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleGetNormalizationSessions получает список сессий нормализации для проекта
func (s *Server) handleGetNormalizationSessions(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for normalization sessions: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (get sessions)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Получаем все базы данных проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		log.Printf("Error getting project databases for project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Failed to get project databases", http.StatusInternalServerError)
		return
	}

	var allSessions []map[string]interface{}
	for _, db := range databases {
		session, err := s.serviceDB.GetLastNormalizationSession(db.ID)
		if err == nil && session != nil {
			sessionData := map[string]interface{}{
				"id":                  session.ID,
				"project_database_id": session.ProjectDatabaseID,
				"database_name":       db.Name,
				"database_path":       db.FilePath,
				"started_at":          session.StartedAt.Format(time.RFC3339),
				"status":              session.Status,
				"priority":            session.Priority,
				"timeout_seconds":     session.TimeoutSeconds,
				"last_activity_at":    session.LastActivityAt.Format(time.RFC3339),
			}
			if session.FinishedAt != nil {
				sessionData["finished_at"] = session.FinishedAt.Format(time.RFC3339)
			}
			allSessions = append(allSessions, sessionData)
		}
	}

	// Получаем остановленные сессии для отображения возможности возобновления
	stoppedSessions, err := s.serviceDB.GetStoppedSessions()
	stoppedSessionsInfo := make([]map[string]interface{}, 0)
	if err == nil {
		for _, session := range stoppedSessions {
			// Проверяем, принадлежит ли сессия этому проекту
			for _, db := range databases {
				if session.ProjectDatabaseID == db.ID {
					sessionInfo := map[string]interface{}{
						"id":                  session.ID,
						"project_database_id": session.ProjectDatabaseID,
						"database_name":       db.Name,
						"database_path":       db.FilePath,
						"status":              session.Status,
						"started_at":          session.StartedAt.Format(time.RFC3339),
						"priority":            session.Priority,
						"can_resume":          true, // Остановленные сессии можно возобновить
					}
					if session.FinishedAt != nil {
						sessionInfo["finished_at"] = session.FinishedAt.Format(time.RFC3339)
					}
					stoppedSessionsInfo = append(stoppedSessionsInfo, sessionInfo)
					break
				}
			}
		}
	}

	// Также получаем все активные сессии
	runningSessions, err := s.serviceDB.GetRunningSessions()
	if err == nil {
		for _, session := range runningSessions {
			// Проверяем, принадлежит ли сессия этому проекту
			for _, db := range databases {
				if session.ProjectDatabaseID == db.ID {
					// Уже добавлена выше
					break
				}
			}
		}
	}

	response := map[string]interface{}{
		"sessions":         allSessions,
		"stopped_sessions": stoppedSessionsInfo,
		"total":            len(allSessions),
	}

	s.writeJSONResponse(w, r, response, http.StatusOK)
}

// handleUpdateSessionPriority обновляет приоритет сессии
func (s *Server) handleUpdateSessionPriority(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for update session priority: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (update priority)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	var req struct {
		SessionID int `json:"session_id"`
		Priority  int `json:"priority"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Error decoding request body for update session priority: %v", err)
		s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверяем, что сессия принадлежит проекту
	session, err := s.serviceDB.GetNormalizationSession(req.SessionID)
	if err != nil || session == nil {
		log.Printf("Session %d not found for update priority", req.SessionID)
		s.writeJSONError(w, r, "Session not found", http.StatusNotFound)
		return
	}

	// Проверяем, что база данных принадлежит проекту
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		log.Printf("Error getting project databases for project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Failed to get project databases", http.StatusInternalServerError)
		return
	}

	found := false
	for _, db := range databases {
		if db.ID == session.ProjectDatabaseID {
			found = true
			break
		}
	}

	if !found {
		log.Printf("Session %d does not belong to project %d", req.SessionID, projectID)
		s.writeJSONError(w, r, "Session does not belong to this project", http.StatusForbidden)
		return
	}

	// Обновляем приоритет
	err = s.serviceDB.UpdateSessionPriority(req.SessionID, req.Priority)
	if err != nil {
		log.Printf("Error updating session %d priority: %v", req.SessionID, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to update priority: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Session %d priority updated to %d for project %d", req.SessionID, req.Priority, projectID)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":    true,
		"session_id": req.SessionID,
		"priority":   req.Priority,
	}, http.StatusOK)
}

// handleStopNormalizationSession останавливает конкретную сессию нормализации
func (s *Server) handleStopNormalizationSession(w http.ResponseWriter, r *http.Request, clientID, projectID, sessionID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for stop session: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (stop session)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Проверяем, что сессия принадлежит проекту
	session, err := s.serviceDB.GetNormalizationSession(sessionID)
	if err != nil || session == nil {
		log.Printf("Session %d not found for stop", sessionID)
		s.writeJSONError(w, r, "Session not found", http.StatusNotFound)
		return
	}

	// Проверяем, что база данных принадлежит проекту
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		log.Printf("Error getting project databases for project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Failed to get project databases", http.StatusInternalServerError)
		return
	}

	found := false
	for _, db := range databases {
		if db.ID == session.ProjectDatabaseID {
			found = true
			break
		}
	}

	if !found {
		log.Printf("Session %d does not belong to project %d", sessionID, projectID)
		s.writeJSONError(w, r, "Session does not belong to this project", http.StatusForbidden)
		return
	}

	// Останавливаем сессию
	err = s.serviceDB.StopNormalizationSession(sessionID)
	if err != nil {
		log.Printf("Error stopping session %d: %v", sessionID, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to stop session: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Session %d stopped successfully for project %d", sessionID, projectID)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"message":    "Session stopped successfully",
	}, http.StatusOK)
}

// handleResumeNormalizationSession возобновляет остановленную сессию нормализации
func (s *Server) handleResumeNormalizationSession(w http.ResponseWriter, r *http.Request, clientID, projectID, sessionID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for resume session: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (resume session)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Проверяем, что сессия принадлежит проекту
	session, err := s.serviceDB.GetNormalizationSession(sessionID)
	if err != nil || session == nil {
		log.Printf("Session %d not found for resume", sessionID)
		s.writeJSONError(w, r, "Session not found", http.StatusNotFound)
		return
	}

	// Проверяем, что база данных принадлежит проекту
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		log.Printf("Error getting project databases for project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Failed to get project databases", http.StatusInternalServerError)
		return
	}

	var projectDB *database.ProjectDatabase
	for _, db := range databases {
		if db.ID == session.ProjectDatabaseID {
			projectDB = db
			break
		}
	}

	if projectDB == nil {
		log.Printf("Session %d does not belong to project %d", sessionID, projectID)
		s.writeJSONError(w, r, "Session does not belong to this project", http.StatusForbidden)
		return
	}

	// Проверяем, что сессия остановлена
	if session.Status != "stopped" {
		s.writeJSONError(w, r, fmt.Sprintf("Session is not stopped (status: %s)", session.Status), http.StatusBadRequest)
		return
	}

	// Возобновляем сессию
	err = s.serviceDB.ResumeNormalizationSession(sessionID)
	if err != nil {
		log.Printf("Error resuming session %d: %v", sessionID, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to resume session: %v", err), http.StatusBadRequest)
		return
	}

	// Запускаем нормализацию в горутине
	go func() {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("Panic in resume normalization goroutine for session %d: %v\n%s", sessionID, rec, debug.Stack())
				select {
				case s.normalizerEvents <- fmt.Sprintf("Критическая ошибка при возобновлении сессии %d: %v", sessionID, rec):
				default:
				}
			}
		}()

		s.resumeCounterpartyDatabase(projectDB, clientID, projectID, sessionID)
	}()

	log.Printf("Session %d resumed successfully for project %d", sessionID, projectID)

	s.writeJSONResponse(w, r, map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"message":    "Normalization resumed",
	}, http.StatusOK)
}

// resumeCounterpartyDatabase возобновляет нормализацию контрагентов для остановленной сессии
func (s *Server) resumeCounterpartyDatabase(projectDB *database.ProjectDatabase, clientID, projectID, sessionID int) {
	// Обновляем last_used_at
	s.serviceDB.UpdateProjectDatabaseLastUsed(projectDB.ID)

	// Открываем подключение к базе данных
	sourceDB, err := database.NewDB(projectDB.FilePath)
	if err != nil {
		// Обновляем сессию как failed
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка открытия БД %s при возобновлении: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Failed to open database %s for resume: %v", projectDB.FilePath, err)
		return
	}
	defer func() {
		if err := sourceDB.Close(); err != nil {
			log.Printf("Error closing database %s: %v", projectDB.FilePath, err)
		}
	}()

	// Получаем контрагентов из справочника "Контрагенты"
	uploads, err := sourceDB.GetAllUploads()
	if err != nil {
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка получения выгрузок из %s при возобновлении: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Failed to get uploads from %s for resume: %v", projectDB.FilePath, err)
		return
	}

	var counterparties []*database.CatalogItem
	for _, upload := range uploads {
		items, _, err := sourceDB.GetCatalogItemsByUpload(upload.ID, []string{"Контрагенты"}, 0, 0)
		if err != nil {
			log.Printf("Failed to get counterparties from upload %d for resume: %v", upload.ID, err)
			continue
		}
		counterparties = append(counterparties, items...)
	}

	if len(counterparties) == 0 {
		select {
		case s.normalizerEvents <- fmt.Sprintf("Контрагенты не найдены в БД %s при возобновлении", projectDB.Name):
		default:
		}
		return
	}

	log.Printf("Resuming counterparty normalization for session %d, project %d with %d counterparties from %s", sessionID, projectID, len(counterparties), projectDB.FilePath)
	select {
	case s.normalizerEvents <- fmt.Sprintf("Возобновление нормализации контрагентов БД: %s (%d записей)", projectDB.Name, len(counterparties)):
	default:
	}

	// Проверяем, не нужно ли остановить нормализацию перед началом обработки
	if s.shouldStopNormalization() {
		log.Printf("Normalization stopped before resuming counterparties from %s", projectDB.FilePath)
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация остановлена перед возобновлением БД %s", projectDB.Name):
		default:
		}
		return
	}

	// Проверяем остановку перед началом нормализации
	if s.shouldStopNormalization() {
		log.Printf("[Counterparty] Normalization stopped before starting (resume) for database %s (project %d)", projectDB.Name, projectID)
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "stopped", &finishedAt)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Нормализация контрагентов БД %s остановлена пользователем до начала обработки (возобновление)", projectDB.Name):
		default:
		}
		return
	}

	// Создаем нормализатор через сервис, который правильно настроен с benchmarkFinder
	var counterpartyNormalizer *normalization.CounterpartyNormalizer
	if s.counterpartyService != nil && s.multiProviderClient != nil {
		counterpartyNormalizer = s.counterpartyService.CreateNormalizer(clientID, projectID, s.multiProviderClient)
	} else if s.serviceDB != nil && s.multiProviderClient != nil {
		// Fallback: создаем напрямую без benchmarkFinder
		ctx := context.Background()
		counterpartyNormalizer = normalization.NewCounterpartyNormalizer(s.serviceDB, clientID, projectID, s.normalizerEvents, ctx, s.multiProviderClient, nil)
		log.Printf("Warning: Using fallback counterparty normalizer (without benchmarkFinder) for database %s (resume)", projectDB.Name)
	} else {
		// Критическая ошибка: невозможно создать нормализатор
		log.Printf("Error: Cannot create counterparty normalizer - counterpartyService, serviceDB or multiProviderClient is nil for database %s (resume)", projectDB.Name)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка: невозможно создать нормализатор контрагентов для БД %s при возобновлении - отсутствуют необходимые сервисы", projectDB.Name):
		default:
		}
		finishedAt := time.Now()
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", &finishedAt)
		return
	}

	// Запускаем нормализацию контрагентов с пропуском уже нормализованных (skipNormalized = true)
	result, err := counterpartyNormalizer.ProcessNormalization(counterparties, true)
	if err != nil {
		// Обновляем сессию как failed
		s.serviceDB.UpdateNormalizationSession(sessionID, "failed", nil)
		select {
		case s.normalizerEvents <- fmt.Sprintf("Ошибка нормализации контрагентов БД %s при возобновлении: %v", projectDB.FilePath, err):
		default:
		}
		log.Printf("Ошибка нормализации контрагентов при возобновлении для %s: %v", projectDB.FilePath, err)
	} else {
		// Обрабатываем результат нормализации (при возобновлении)
		s.handleCounterpartyNormalizationResult(sessionID, result, projectDB, counterparties, clientID, projectID, true)
	}
}

// handleGetClientNormalizationStats получает статистику нормализации для клиента
func (s *Server) handleGetClientNormalizationStats(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for normalization stats: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (get stats)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	s.normalizerMutex.RLock()
	isRunning := s.normalizerRunning
	s.normalizerMutex.RUnlock()

	stats := map[string]interface{}{
		"is_running":        isRunning,
		"total_processed":   0,
		"total_groups":      0,
		"benchmark_matches": 0,
		"ai_enhanced":       0,
		"basic_normalized":  0,
	}

	// Если проект типа контрагентов, получаем статистику по контрагентам
	if database.IsCounterpartyProjectType(project.ProjectType) {
		counterpartyStats, err := s.serviceDB.GetNormalizedCounterpartyStats(projectID)
		if err == nil {
			stats["counterparty_stats"] = counterpartyStats
			stats["total_processed"] = counterpartyStats["total_count"]
			if withBenchmark, ok := counterpartyStats["with_benchmark"].(int); ok {
				stats["benchmark_matches"] = withBenchmark
			}
			if enriched, ok := counterpartyStats["enriched"].(int); ok {
				stats["enriched_count"] = enriched
			}
		}

		// Получаем статистику по дубликатам
		databases, err := s.serviceDB.GetProjectDatabases(projectID, true)
		if err == nil && len(databases) > 0 {
			var allCounterparties []*database.CatalogItem
			for _, dbInfo := range databases {
				if !dbInfo.IsActive {
					continue
				}
				db, err := database.NewDB(dbInfo.FilePath)
				if err != nil {
					continue
				}
				defer func() {
					if err := db.Close(); err != nil {
						log.Printf("Error closing database in getAllCounterparties: %v", err)
					}
				}()
				uploads, err := db.GetAllUploads()
				if err != nil {
					continue
				}
				for _, upload := range uploads {
					items, _, err := db.GetCatalogItemsByUpload(upload.ID, []string{"Контрагенты"}, 0, 0)
					if err == nil {
						allCounterparties = append(allCounterparties, items...)
					}
				}
			}

			if len(allCounterparties) > 0 {
				duplicateAnalyzer := normalization.NewCounterpartyDuplicateAnalyzer()
				duplicateGroups := duplicateAnalyzer.AnalyzeDuplicates(allCounterparties)
				summary := duplicateAnalyzer.GetDuplicateSummary(duplicateGroups)
				stats["duplicate_stats"] = summary
			}
		}
	}

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleQualityStats возвращает статистику качества нормализации
func (s *Server) handleQualityStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем параметр database из query
	databasePath := r.URL.Query().Get("database")
	if databasePath == "" {
		// Если не указан, используем normalizedDB по умолчанию
		databasePath = s.currentNormalizedDBPath
	}

	// Открываем нужную БД
	var db *database.DB
	var err error
	if databasePath != "" && databasePath != s.currentNormalizedDBPath {
		db, err = database.NewDB(databasePath)
		if err != nil {
			log.Printf("Error opening database %s: %v", databasePath, err)
			s.writeJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
			return
		}
		defer db.Close()
	} else {
		// Используем текущую БД
		db = s.db
	}

	stats, err := db.GetQualityStats()
	if err != nil {
		log.Printf("Error getting quality stats: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get quality stats: %v", err), http.StatusInternalServerError)
		return
	}

	s.writeJSONResponse(w, r, stats, http.StatusOK)
}

// handleGetClientNormalizationGroups возвращает группы нормализации для базы данных проекта
func (s *Server) handleGetClientNormalizationGroups(w http.ResponseWriter, r *http.Request, clientID, projectID int) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем существование проекта
	project, err := s.serviceDB.GetClientProject(projectID)
	if err != nil {
		log.Printf("Error getting project %d for normalization groups: %v", projectID, err)
		s.writeJSONError(w, r, "Project not found", http.StatusNotFound)
		return
	}

	if project.ClientID != clientID {
		log.Printf("Project %d does not belong to client %d (get groups)", projectID, clientID)
		s.writeJSONError(w, r, "Project does not belong to this client", http.StatusBadRequest)
		return
	}

	// Получаем параметры запроса
	query := r.URL.Query()
	dbIDStr := query.Get("db_id")

	if dbIDStr == "" {
		s.writeJSONError(w, r, "db_id parameter is required", http.StatusBadRequest)
		return
	}

	dbID, err := ValidateIDParam(r, "db_id")
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Invalid db_id parameter: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// Получаем информацию о базе данных
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		log.Printf("Error getting project databases for project %d: %v", projectID, err)
		s.writeJSONError(w, r, "Failed to get project databases", http.StatusInternalServerError)
		return
	}

	var targetDB *database.ProjectDatabase
	for _, db := range databases {
		if db.ID == dbID {
			targetDB = db
			break
		}
	}

	if targetDB == nil {
		log.Printf("Database %d not found in project %d", dbID, projectID)
		s.writeJSONError(w, r, "Database not found", http.StatusNotFound)
		return
	}

	// Получаем последнюю сессию нормализации для этой базы
	session, err := s.serviceDB.GetLastNormalizationSession(dbID)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to get normalization session: %v", err), http.StatusInternalServerError)
		return
	}

	if session == nil {
		// Нет сессий для этой базы - возвращаем пустой список
		s.writeJSONResponse(w, r, map[string]interface{}{
			"groups":     []interface{}{},
			"total":      0,
			"page":       1,
			"limit":      20,
			"totalPages": 0,
		}, http.StatusOK)
		return
	}

	// Открываем проектную БД
	projectDB, err := database.NewDB(targetDB.FilePath)
	if err != nil {
		s.writeJSONError(w, r, fmt.Sprintf("Failed to open project database: %v", err), http.StatusInternalServerError)
		return
	}
	defer projectDB.Close()

	// Получаем параметры пагинации
	page, limit, err := ValidatePaginationParams(r, 1, 20, 100)
	if err != nil {
		if s.HandleValidationError(w, r, err) {
			return
		}
	}

	offset := (page - 1) * limit

	// Строим SQL запрос с фильтрацией по session_id
	baseQuery := `
		SELECT normalized_name, normalized_reference, category, COUNT(*) as merged_count,
		       MAX(kpved_code) as kpved_code, MAX(kpved_name) as kpved_name, 
		       AVG(kpved_confidence) as kpved_confidence, MAX(created_at) as last_normalized_at
		FROM normalized_data
		WHERE normalization_session_id = ?
		GROUP BY normalized_name, normalized_reference, category
		ORDER BY merged_count DESC, normalized_name ASC
		LIMIT ? OFFSET ?
	`

	countQuery := `
		SELECT COUNT(*) FROM (
			SELECT normalized_name, category
			FROM normalized_data
			WHERE normalization_session_id = ?
			GROUP BY normalized_name, category
		)
	`

	// Получаем общее количество групп
	var totalGroups int
	err = projectDB.QueryRow(countQuery, session.ID).Scan(&totalGroups)
	if err != nil {
		log.Printf("Ошибка получения количества групп: %v", err)
		s.writeJSONError(w, r, "Failed to count groups", http.StatusInternalServerError)
		return
	}

	// Получаем группы
	rows, err := projectDB.Query(baseQuery, session.ID, limit, offset)
	if err != nil {
		log.Printf("Ошибка выполнения запроса групп: %v", err)
		s.writeJSONError(w, r, "Failed to fetch groups", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	type Group struct {
		NormalizedName      string   `json:"normalized_name"`
		NormalizedReference string   `json:"normalized_reference"`
		Category            string   `json:"category"`
		MergedCount         int      `json:"merged_count"`
		KpvedCode           *string  `json:"kpved_code,omitempty"`
		KpvedName           *string  `json:"kpved_name,omitempty"`
		KpvedConfidence     *float64 `json:"kpved_confidence,omitempty"`
		LastNormalizedAt    *string  `json:"last_normalized_at,omitempty"`
	}

	groups := []Group{}
	for rows.Next() {
		var g Group
		var lastNormalizedAt sql.NullString
		if err := rows.Scan(&g.NormalizedName, &g.NormalizedReference, &g.Category, &g.MergedCount,
			&g.KpvedCode, &g.KpvedName, &g.KpvedConfidence, &lastNormalizedAt); err != nil {
			log.Printf("Ошибка сканирования группы: %v", err)
			continue
		}
		if lastNormalizedAt.Valid && lastNormalizedAt.String != "" {
			g.LastNormalizedAt = &lastNormalizedAt.String
		}
		groups = append(groups, g)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Ошибка при итерации по группам: %v", err)
	}

	// Вычисляем общее количество страниц
	totalPages := (totalGroups + limit - 1) / limit

	response := map[string]interface{}{
		"groups":     groups,
		"total":      totalGroups,
		"page":       page,
		"limit":      limit,
		"totalPages": totalPages,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Обработчики мониторинга перенесены в server_monitoring_handlers.go
// handleMonitoringMetrics возвращает общую статистику производительности
func (s *Server) _handleMonitoringMetrics_OLD(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем реальную статистику от нормализатора
	var statsCollector *normalization.StatsCollector
	var cacheStats normalization.CacheStats
	hasCacheStats := false

	if s.normalizer != nil && s.normalizer.GetAINormalizer() != nil {
		statsCollector = s.normalizer.GetAINormalizer().GetStatsCollector()
		cacheStats = s.normalizer.GetAINormalizer().GetCacheStats()
		hasCacheStats = true
	}

	// Получаем статистику качества из БД (используем db, так как записи сохраняются туда)
	qualityStatsMap, err := s.db.GetQualityStats()
	if err != nil {
		log.Printf("Error getting quality stats: %v", err)
		qualityStatsMap = make(map[string]interface{})
	}

	// Извлекаем значения из карты БД (используем как fallback)
	dbTotalNormalized := int64(0)
	dbBasicNormalized := int64(0)
	dbAIEnhanced := int64(0)
	dbBenchmarkQuality := int64(0)
	dbAverageQualityScore := 0.0

	if total, ok := qualityStatsMap["total_items"].(int); ok {
		dbTotalNormalized = int64(total)
	}
	if avg, ok := qualityStatsMap["average_quality"].(float64); ok {
		dbAverageQualityScore = avg
	}
	if benchmark, ok := qualityStatsMap["benchmark_count"].(int); ok {
		dbBenchmarkQuality = int64(benchmark)
	}
	if byLevel, ok := qualityStatsMap["by_level"].(map[string]map[string]interface{}); ok {
		if basicStats, ok := byLevel["basic"]; ok {
			if count, ok := basicStats["count"].(int); ok {
				dbBasicNormalized = int64(count)
			}
		}
		if aiStats, ok := byLevel["ai_enhanced"]; ok {
			if count, ok := aiStats["count"].(int); ok {
				dbAIEnhanced = int64(count)
			}
		}
	}

	// Используем метрики из StatsCollector как основной источник, БД как fallback
	totalNormalized := dbTotalNormalized
	basicNormalized := dbBasicNormalized
	aiEnhanced := dbAIEnhanced
	benchmarkQuality := dbBenchmarkQuality
	averageQualityScore := dbAverageQualityScore

	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		// Используем метрики из StatsCollector если они доступны
		if perfMetrics.TotalNormalized > 0 {
			totalNormalized = perfMetrics.TotalNormalized
			basicNormalized = perfMetrics.BasicNormalized
			aiEnhanced = perfMetrics.AIEnhanced
			benchmarkQuality = perfMetrics.BenchmarkQuality
			if perfMetrics.AverageQualityScore > 0 {
				averageQualityScore = perfMetrics.AverageQualityScore
			} else if dbAverageQualityScore > 0 {
				averageQualityScore = dbAverageQualityScore
			}
		}
	}

	// Рассчитываем uptime
	uptime := time.Since(s.startTime).Seconds()

	// Рассчитываем throughput (за всё время работы)
	throughput := 0.0
	if uptime > 0 && totalNormalized > 0 {
		throughput = float64(totalNormalized) / uptime
	}

	// Формируем ответ
	summary := map[string]interface{}{
		"uptime_seconds":              uptime,
		"throughput_items_per_second": throughput,
		"ai": map[string]interface{}{
			"total_requests":     0,
			"successful":         0,
			"failed":             0,
			"success_rate":       0.0,
			"average_latency_ms": 0.0,
		},
		"cache": map[string]interface{}{
			"hits":            0,
			"misses":          0,
			"hit_rate":        0.0,
			"size":            0,
			"memory_usage_kb": 0.0,
		},
		"quality": map[string]interface{}{
			"total_normalized":      totalNormalized,
			"basic":                 basicNormalized,
			"ai_enhanced":           aiEnhanced,
			"benchmark":             benchmarkQuality,
			"average_quality_score": averageQualityScore,
		},
	}

	// Добавляем реальные AI метрики если доступны
	if statsCollector != nil {
		perfMetrics := statsCollector.GetMetrics()
		successRate := 0.0
		if perfMetrics.TotalAIRequests > 0 {
			successRate = float64(perfMetrics.SuccessfulAIRequest) / float64(perfMetrics.TotalAIRequests)
		}
		avgLatencyMs := float64(perfMetrics.AverageAILatency.Milliseconds())

		summary["ai"] = map[string]interface{}{
			"total_requests":     perfMetrics.TotalAIRequests,
			"successful":         perfMetrics.SuccessfulAIRequest,
			"failed":             perfMetrics.FailedAIRequests,
			"success_rate":       successRate,
			"average_latency_ms": avgLatencyMs,
		}
	}

	// Добавляем реальные cache метрики если доступны
	if hasCacheStats {
		summary["cache"] = map[string]interface{}{
			"hits":            cacheStats.Hits,
			"misses":          cacheStats.Misses,
			"hit_rate":        cacheStats.HitRate,
			"size":            cacheStats.Entries,
			"memory_usage_kb": float64(cacheStats.MemoryUsageB) / 1024.0,
		}
	}

	// Добавляем метрики Circuit Breaker
	summary["circuit_breaker"] = s.GetCircuitBreakerState()

	// Добавляем метрики Batch Processor
	summary["batch_processor"] = s.GetBatchProcessorStats()

	// Добавляем статус Checkpoint
	summary["checkpoint"] = s.GetCheckpointStatus()

	s.writeJSONResponse(w, r, summary, http.StatusOK)
}

// handleNormalizationConfig обрабатывает GET/POST запросы конфигурации нормализации
func (s *Server) handleNormalizationConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Получить текущую конфигурацию
		config, err := s.serviceDB.GetNormalizationConfig()
		if err != nil {
			log.Printf("Error getting normalization config: %v", err)
			s.writeJSONError(w, r, fmt.Sprintf("Failed to get config: %v", err), http.StatusInternalServerError)
			return
		}
		s.writeJSONResponse(w, r, config, http.StatusOK)

	case http.MethodPost:
		// Обновить конфигурацию
		var req struct {
			DatabasePath    string `json:"database_path"`
			SourceTable     string `json:"source_table"`
			ReferenceColumn string `json:"reference_column"`
			CodeColumn      string `json:"code_column"`
			NameColumn      string `json:"name_column"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeJSONError(w, r, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Валидация
		if req.SourceTable == "" || req.ReferenceColumn == "" || req.CodeColumn == "" || req.NameColumn == "" {
			s.writeJSONError(w, r, "All fields are required", http.StatusBadRequest)
			return
		}

		err := s.serviceDB.UpdateNormalizationConfig(
			req.DatabasePath,
			req.SourceTable,
			req.ReferenceColumn,
			req.CodeColumn,
			req.NameColumn,
		)
		if err != nil {
			log.Printf("Error updating normalization config: %v", err)
			s.writeJSONError(w, r, fmt.Sprintf("Failed to update config: %v", err), http.StatusInternalServerError)
			return
		}

		// Возвращаем обновленную конфигурацию
		config, err := s.serviceDB.GetNormalizationConfig()
		if err != nil {
			log.Printf("Error getting updated config: %v", err)
			s.writeJSONError(w, r, fmt.Sprintf("Config saved but failed to retrieve: %v", err), http.StatusInternalServerError)
			return
		}

		s.writeJSONResponse(w, r, config, http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleNormalizationDatabases возвращает список доступных баз данных
func (s *Server) handleNormalizationDatabases(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Получаем список БД файлов в текущей директории
	databases := []map[string]interface{}{}

	// Добавляем текущую БД
	if s.currentDBPath != "" {
		databases = append(databases, map[string]interface{}{
			"path": s.currentDBPath,
			"name": filepath.Base(s.currentDBPath),
			"type": "current",
		})
	}

	// Сканируем директорию с БД файлами
	files, err := filepath.Glob("*.db")
	if err == nil {
		for _, file := range files {
			// Пропускаем если это текущая БД или service.db
			if file == s.currentDBPath || file == "service.db" {
				continue
			}

			databases = append(databases, map[string]interface{}{
				"path": file,
				"name": filepath.Base(file),
				"type": "available",
			})
		}
	}

	s.writeJSONResponse(w, r, databases, http.StatusOK)
}

// handleNormalizationTables возвращает список таблиц в указанной БД
func (s *Server) handleNormalizationTables(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbPath := r.URL.Query().Get("database")
	if dbPath == "" {
		// Если путь не указан, используем текущую БД
		dbPath = s.currentDBPath
	}

	// Открываем БД
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database %s: %v", dbPath, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Получаем список таблиц
	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		log.Printf("Error querying tables: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to query tables: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	tables := []map[string]interface{}{}
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Пропускаем системные таблицы SQLite
		if strings.HasPrefix(tableName, "sqlite_") {
			continue
		}

		// Валидация имени таблицы из БД (защита от потенциально скомпрометированных данных)
		if !isValidTableName(tableName) {
			log.Printf("Warning: Invalid table name from database: %s, skipping", tableName)
			continue
		}

		// Получаем количество записей (безопасный запрос)
		var count int
		query := buildSafeTableQuery("SELECT COUNT(*) FROM %s", tableName)
		if query == "" {
			log.Printf("Error: Failed to build safe query for table %s", tableName)
			count = 0
		} else {
			conn.QueryRow(query).Scan(&count)
		}

		tables = append(tables, map[string]interface{}{
			"name":  tableName,
			"count": count,
		})
	}

	s.writeJSONResponse(w, r, tables, http.StatusOK)
}

// handleNormalizationColumns возвращает список колонок в указанной таблице
func (s *Server) handleNormalizationColumns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	dbPath := r.URL.Query().Get("database")
	tableName := r.URL.Query().Get("table")

	if tableName == "" {
		s.writeJSONError(w, r, "Table name is required", http.StatusBadRequest)
		return
	}

	// Валидация имени таблицы
	if !isValidTableName(tableName) {
		log.Printf("Invalid table name: %s", tableName)
		s.writeJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}

	if dbPath == "" {
		dbPath = s.currentDBPath
	}

	// Открываем БД
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Printf("Error opening database %s: %v", dbPath, err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to open database: %v", err), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Получаем информацию о колонках (безопасный запрос)
	pragmaQuery := buildSafeTableQuery("PRAGMA table_info(%s)", tableName)
	if pragmaQuery == "" {
		s.writeJSONError(w, r, "Invalid table name", http.StatusBadRequest)
		return
	}
	rows, err := conn.Query(pragmaQuery)
	if err != nil {
		log.Printf("Error querying columns: %v", err)
		s.writeJSONError(w, r, fmt.Sprintf("Failed to query columns: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	columns := []map[string]interface{}{}
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			continue
		}

		columns = append(columns, map[string]interface{}{
			"name":        name,
			"type":        colType,
			"primary_key": pk == 1,
			"not_null":    notNull == 1,
		})
	}

	s.writeJSONResponse(w, r, columns, http.StatusOK)
}
