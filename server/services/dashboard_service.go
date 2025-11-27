package services

import (
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"httpserver/database"
	apperrors "httpserver/server/errors"
)

// ActivityLog представляет запись активности
type ActivityLog struct {
	ID        int
	Type      string
	Timestamp time.Time
	Message   string
	ClientID  *int
	ProjectID *int
}

// DashboardService сервис для работы с дашбордом
type DashboardService struct {
	db                     *database.DB
	normalizedDB           *database.DB
	serviceDB              *database.ServiceDB
	getStatsFunc           func() map[string]interface{} // Функция для получения статистики от Server
	getNormalizationStatusFunc func() map[string]interface{} // Функция для получения статуса нормализации
}

// NewDashboardService создает новый сервис для работы с дашбордом
func NewDashboardService(
	db *database.DB,
	normalizedDB *database.DB,
	serviceDB *database.ServiceDB,
	getStatsFunc func() map[string]interface{},
	getNormalizationStatusFunc func() map[string]interface{},
) *DashboardService {
	return &DashboardService{
		db:                        db,
		normalizedDB:              normalizedDB,
		serviceDB:                 serviceDB,
		getStatsFunc:              getStatsFunc,
		getNormalizationStatusFunc: getNormalizationStatusFunc,
	}
}

// GetStats возвращает общую статистику для дашборда
func (s *DashboardService) GetStats() (map[string]interface{}, error) {
	if s.getStatsFunc != nil {
		return s.getStatsFunc(), nil
	}
	
	// Fallback: получаем базовую статистику из БД
	qualityStats, err := s.db.GetQualityStats()
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить статистику качества", err)
	}
	
	return qualityStats, nil
}

// GetNormalizationStatus возвращает статус нормализации для дашборда
func (s *DashboardService) GetNormalizationStatus() (map[string]interface{}, error) {
	if s.getNormalizationStatusFunc != nil {
		return s.getNormalizationStatusFunc(), nil
	}
	
	// Fallback
	return map[string]interface{}{
		"status":  "not_running",
		"message": "Нормализация не запущена",
	}, nil
}

// QualityMetrics метрики качества (копия из server_dashboard.go для использования в сервисе)
type QualityMetrics struct {
	OverallQuality   float64 `json:"overallQuality"`   // Общее качество (0-1)
	HighConfidence   float64 `json:"highConfidence"`   // Процент записей с высокой уверенностью (0-1)
	MediumConfidence float64 `json:"mediumConfidence"`  // Процент записей со средней уверенностью (0-1)
	LowConfidence    float64 `json:"lowConfidence"`    // Процент записей с низкой уверенностью (0-1)
	TotalRecords     int     `json:"totalRecords"`     // Общее количество записей
}

// GetQualityMetrics возвращает метрики качества данных
func (s *DashboardService) GetQualityMetrics() (*QualityMetrics, error) {
	if s.db == nil {
		return nil, apperrors.NewInternalError("база данных недоступна", nil)
	}

	metrics := &QualityMetrics{
		OverallQuality:   0,
		HighConfidence:   0,
		MediumConfidence: 0,
		LowConfidence:    0,
		TotalRecords:     0,
	}

	// Получаем общее количество записей
	var totalRecords int
	err := s.db.GetDB().QueryRow("SELECT COUNT(*) FROM normalized_data").Scan(&totalRecords)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return metrics, apperrors.NewNotFoundError("записи не найдены", err)
		}
		return metrics, apperrors.NewInternalError("не удалось получить общее количество записей", err)
	}
	metrics.TotalRecords = totalRecords

	if totalRecords > 0 {
		// Получаем метрики качества из normalized_data
		// Используем quality_score если есть, иначе ai_confidence
		rows, err := s.db.GetDB().Query(`
			SELECT 
				CASE 
					WHEN COALESCE(NULLIF(quality_score, 0), NULLIF(ai_confidence, 0), 0) >= 0.9 THEN 'high'
					WHEN COALESCE(NULLIF(quality_score, 0), NULLIF(ai_confidence, 0), 0) >= 0.7 THEN 'medium'
					WHEN COALESCE(NULLIF(quality_score, 0), NULLIF(ai_confidence, 0), 0) > 0 THEN 'low'
					ELSE NULL
				END as level,
				COUNT(*) as count
			FROM normalized_data
			WHERE (quality_score IS NOT NULL AND quality_score > 0) OR (ai_confidence IS NOT NULL AND ai_confidence > 0)
			GROUP BY level
		`)
		if err == nil {
			defer rows.Close()
			var highCount, mediumCount, lowCount int
			var weightedSum float64
			var recordsWithConfidence int
			
			for rows.Next() {
				var level sql.NullString
				var count int
				if err := rows.Scan(&level, &count); err != nil {
					continue
				}
				
				if !level.Valid {
					continue
				}
				
				switch level.String {
				case "high":
					highCount = count
					weightedSum += float64(count) * 0.95
				case "medium":
					mediumCount = count
					weightedSum += float64(count) * 0.8
				case "low":
					lowCount = count
					weightedSum += float64(count) * 0.35
				}
				recordsWithConfidence += count
			}
			
			if err := rows.Err(); err != nil {
				// Логируем ошибку, но продолжаем
			}
			
			// Рассчитываем проценты от общего количества записей
			if totalRecords > 0 {
				metrics.HighConfidence = float64(highCount) / float64(totalRecords)
				metrics.MediumConfidence = float64(mediumCount) / float64(totalRecords)
				metrics.LowConfidence = float64(lowCount) / float64(totalRecords)
				
				// Рассчитываем общее качество как средневзвешенное
				if recordsWithConfidence > 0 {
					metrics.OverallQuality = weightedSum / float64(recordsWithConfidence)
				} else {
					// Если нет записей с уверенностью, используем среднее на основе processing_level
					var avgQuality sql.NullFloat64
					err = s.db.GetDB().QueryRow(`
						SELECT AVG(CASE 
							WHEN processing_level = 'benchmark' THEN 0.95
							WHEN processing_level = 'ai_enhanced' THEN 0.85
							WHEN processing_level = 'enhanced' THEN 0.70
							ELSE 0.50
						END)
						FROM normalized_data
					`).Scan(&avgQuality)
					if err == nil && avgQuality.Valid {
						metrics.OverallQuality = avgQuality.Float64
					}
				}
			}
		} else {
			// Если запрос не удался, используем значения по умолчанию на основе processing_level
			var avgQuality sql.NullFloat64
			err = s.db.GetDB().QueryRow(`
				SELECT AVG(CASE 
					WHEN processing_level = 'benchmark' THEN 0.95
					WHEN processing_level = 'ai_enhanced' THEN 0.85
					WHEN processing_level = 'enhanced' THEN 0.70
					ELSE 0.50
				END)
				FROM normalized_data
			`).Scan(&avgQuality)
			if err == nil && avgQuality.Valid {
				metrics.OverallQuality = avgQuality.Float64
			}
		}
	}

	return metrics, nil
}

// GetRecentActivity возвращает последние записи активности из разных источников
func (s *DashboardService) GetRecentActivity(limit int) ([]ActivityLog, error) {
	if limit <= 0 {
		limit = 20 // Значение по умолчанию
	}
	if limit > 100 {
		limit = 100 // Максимальное значение
	}

	var activities []ActivityLog

	// 1. Получаем последние загрузки из uploads
	if s.db != nil {
		uploads, err := s.db.GetAllUploads()
		if err == nil && len(uploads) > 0 {
			// Берем последние загрузки
			maxUploads := limit / 2
			if maxUploads > len(uploads) {
				maxUploads = len(uploads)
			}
			for i := 0; i < maxUploads; i++ {
				upload := uploads[i]
				activity := ActivityLog{
					ID:        upload.ID,
					Type:      "upload",
					Timestamp: upload.StartedAt,
				}

				// Формируем сообщение
				if upload.Status == "completed" {
					activity.Message = fmt.Sprintf("Загрузка %s завершена (конфигурация: %s)", upload.UploadUUID[:8], upload.ConfigName)
				} else if upload.Status == "in_progress" {
					activity.Message = fmt.Sprintf("Загрузка %s в процессе (конфигурация: %s)", upload.UploadUUID[:8], upload.ConfigName)
				} else {
					activity.Message = fmt.Sprintf("Загрузка %s: %s (конфигурация: %s)", upload.UploadUUID[:8], upload.Status, upload.ConfigName)
				}

				// Добавляем clientID и projectID если они есть
				if upload.ClientID != nil {
					clientID := int(*upload.ClientID)
					activity.ClientID = &clientID
				}
				if upload.ProjectID != nil {
					projectID := int(*upload.ProjectID)
					activity.ProjectID = &projectID
				}

				activities = append(activities, activity)
			}
		}
	}

	// 2. Получаем последние обновления баз данных из project_databases
	if s.serviceDB != nil {
		query := `
			SELECT pd.id, pd.name, pd.updated_at, pd.client_project_id, cp.client_id
			FROM project_databases pd
			JOIN client_projects cp ON pd.client_project_id = cp.id
			WHERE pd.updated_at IS NOT NULL
			ORDER BY pd.updated_at DESC
			LIMIT ?
		`
		rows, err := s.serviceDB.GetDB().Query(query, limit/4)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, projectID, clientID int
				var name string
				var updatedAt time.Time
				if err := rows.Scan(&id, &name, &updatedAt, &projectID, &clientID); err == nil {
					activity := ActivityLog{
						ID:        id,
						Type:      "database",
						Message:   fmt.Sprintf("База данных '%s' обновлена", name),
						Timestamp: updatedAt,
						ClientID:  &clientID,
						ProjectID: &projectID,
					}
					activities = append(activities, activity)
				}
			}
		}
	}

	// 3. Получаем последние обновления проектов из client_projects
	if s.serviceDB != nil {
		query := `
			SELECT id, name, updated_at, client_id
			FROM client_projects
			WHERE updated_at IS NOT NULL
			ORDER BY updated_at DESC
			LIMIT ?
		`
		rows, err := s.serviceDB.GetDB().Query(query, limit/4)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var id, clientID int
				var name string
				var updatedAt time.Time
				if err := rows.Scan(&id, &name, &updatedAt, &clientID); err == nil {
					activity := ActivityLog{
						ID:        id,
						Type:      "project",
						Message:   fmt.Sprintf("Проект '%s' обновлен", name),
						Timestamp: updatedAt,
						ClientID:  &clientID,
						ProjectID: &id,
					}
					activities = append(activities, activity)
				}
			}
		}
	}

	// Сортируем по времени (самые новые первыми)
	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Timestamp.After(activities[j].Timestamp)
	})

	// Ограничиваем количество результатов
	if len(activities) > limit {
		activities = activities[:limit]
	}

	return activities, nil
}

