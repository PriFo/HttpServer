package services

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"httpserver/database"
	"httpserver/nomenclature"
	apperrors "httpserver/server/errors"
)

// NomenclatureService сервис для работы с номенклатурой
type NomenclatureService struct {
	db                  *database.DB
	normalizedDB        *database.DB
	serviceDB           *database.ServiceDB
	workerConfigManager interface {
		GetNomenclatureConfig() (nomenclature.Config, error)
	}
	processorGetter func() *nomenclature.NomenclatureProcessor
	processorSetter func(*nomenclature.NomenclatureProcessor)
}

// NewNomenclatureService создает новый сервис для работы с номенклатурой
func NewNomenclatureService(
	db *database.DB,
	normalizedDB *database.DB,
	serviceDB *database.ServiceDB,
	workerConfigManager interface {
		GetNomenclatureConfig() (nomenclature.Config, error)
	},
	processorGetter func() *nomenclature.NomenclatureProcessor,
	processorSetter func(*nomenclature.NomenclatureProcessor),
) *NomenclatureService {
	return &NomenclatureService{
		db:                  db,
		normalizedDB:        normalizedDB,
		serviceDB:           serviceDB,
		workerConfigManager: workerConfigManager,
		processorGetter:     processorGetter,
		processorSetter:     processorSetter,
	}
}

// StartProcessing запускает обработку номенклатуры
func (s *NomenclatureService) StartProcessing() error {
	// Получаем конфигурацию из менеджера воркеров, если доступен
	var config nomenclature.Config
	if s.workerConfigManager != nil {
		workerConfig, err := s.workerConfigManager.GetNomenclatureConfig()
		if err == nil {
			config = workerConfig
			config.DatabasePath = "./normalized_data.db"
		} else {
			// Fallback на дефолтную конфигурацию
			apiKey := os.Getenv("ARLIAI_API_KEY")
			if apiKey == "" {
				return apperrors.NewInternalError("переменная окружения ARLIAI_API_KEY не установлена", nil)
			}
			config = nomenclature.DefaultConfig()
			config.ArliaiAPIKey = apiKey
			config.DatabasePath = "./normalized_data.db"
		}
	} else {
		// Fallback на дефолтную конфигурацию
		apiKey := os.Getenv("ARLIAI_API_KEY")
		if apiKey == "" {
			return apperrors.NewInternalError("переменная окружения ARLIAI_API_KEY не установлена", nil)
		}
		config = nomenclature.DefaultConfig()
		config.ArliaiAPIKey = apiKey
		config.DatabasePath = "./normalized_data.db"
	}

	// Создаем процессор
	processor, err := nomenclature.NewProcessor(config)
	if err != nil {
		return apperrors.NewInternalError("не удалось создать процессор", err)
	}

	// Сохраняем процессор
	if s.processorSetter != nil {
		s.processorSetter(processor)
	}

	// Запускаем обработку в горутине
	go func() {
		defer func() {
			processor.Close()
		}()
		if err := processor.ProcessAll(); err != nil {
			fmt.Printf("Ошибка обработки номенклатуры: %v\n", err)
		}
	}()

	return nil
}

// GetStatus возвращает статус обработки номенклатуры
func (s *NomenclatureService) GetStatus() (map[string]interface{}, error) {
	processor := s.processorGetter()
	if processor == nil {
		return map[string]interface{}{
			"status":  "not_started",
			"message": "Обработка номенклатуры не запущена",
		}, nil
	}

	stats := processor.GetStats()
	return map[string]interface{}{
		"status":        "processing",
		"total":         stats.Total,
		"processed":     stats.Processed,
		"successful":    stats.Successful,
		"failed":        stats.Failed,
		"start_time":    stats.StartTime,
	}, nil
}

// GetRecentRecords возвращает недавние записи номенклатуры
func (s *NomenclatureService) GetRecentRecords(limit int) ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, apperrors.NewInternalError("база данных недоступна", nil)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	// Получаем недавно обработанные записи (processed_at не NULL)
	query := `
		SELECT ci.id, ci.catalog_id, c.name as catalog_name,
		       ci.reference, ci.code, ci.name,
		       COALESCE(ci.normalized_name, '') as normalized_name,
		       COALESCE(ci.processing_status, 'pending') as processing_status,
		       ci.processed_at, ci.created_at,
		       COALESCE(ci.kpved_code, '') as kpved_code,
		       COALESCE(ci.kpved_name, '') as kpved_name
		FROM catalog_items ci
		LEFT JOIN catalogs c ON ci.catalog_id = c.id
		WHERE ci.processed_at IS NOT NULL
		ORDER BY ci.processed_at DESC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить недавние записи", err)
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var id, catalogID int
		var reference, code, name, normalizedName, processingStatus, kpvedCode, kpvedName string
		var catalogName sql.NullString
		var processedAt, createdAt time.Time

		err := rows.Scan(
			&id, &catalogID, &catalogName, &reference, &code, &name,
			&normalizedName, &processingStatus, &processedAt, &createdAt,
			&kpvedCode, &kpvedName,
		)
		if err != nil {
			continue
		}

		record := map[string]interface{}{
			"id":               id,
			"catalog_id":       catalogID,
			"reference":        reference,
			"code":             code,
			"name":             name,
			"normalized_name":  normalizedName,
			"processing_status": processingStatus,
			"processed_at":      processedAt,
			"created_at":        createdAt,
			"kpved_code":        kpvedCode,
			"kpved_name":        kpvedName,
		}

		if catalogName.Valid {
			record["catalog_name"] = catalogName.String
		}

		records = append(records, record)
	}

	return records, nil
}

// GetPendingRecords возвращает ожидающие обработки записи
func (s *NomenclatureService) GetPendingRecords(limit int) ([]map[string]interface{}, error) {
	if s.db == nil {
		return nil, apperrors.NewInternalError("база данных недоступна", nil)
	}

	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	// Получаем записи со статусом 'pending' или NULL
	query := `
		SELECT ci.id, ci.catalog_id, c.name as catalog_name,
		       ci.reference, ci.code, ci.name,
		       COALESCE(ci.normalized_name, '') as normalized_name,
		       COALESCE(ci.processing_status, 'pending') as processing_status,
		       ci.processed_at, ci.created_at,
		       COALESCE(ci.processing_attempts, 0) as processing_attempts,
		       COALESCE(ci.error_message, '') as error_message
		FROM catalog_items ci
		LEFT JOIN catalogs c ON ci.catalog_id = c.id
		WHERE ci.processing_status = 'pending' OR ci.processing_status IS NULL
		ORDER BY ci.created_at ASC, ci.id ASC
		LIMIT ?
	`

	rows, err := s.db.Query(query, limit)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить ожидающие записи", err)
	}
	defer rows.Close()

	var records []map[string]interface{}
	for rows.Next() {
		var id, catalogID, processingAttempts int
		var reference, code, name, normalizedName, processingStatus, errorMessage string
		var catalogName sql.NullString
		var processedAt sql.NullTime
		var createdAt time.Time

		err := rows.Scan(
			&id, &catalogID, &catalogName, &reference, &code, &name,
			&normalizedName, &processingStatus, &processedAt, &createdAt,
			&processingAttempts, &errorMessage,
		)
		if err != nil {
			continue
		}

		record := map[string]interface{}{
			"id":                 id,
			"catalog_id":         catalogID,
			"reference":          reference,
			"code":               code,
			"name":               name,
			"normalized_name":    normalizedName,
			"processing_status":  processingStatus,
			"created_at":         createdAt,
			"processing_attempts": processingAttempts,
		}

		if catalogName.Valid {
			record["catalog_name"] = catalogName.String
		}
		if processedAt.Valid {
			record["processed_at"] = processedAt.Time
		}
		if errorMessage != "" {
			record["error_message"] = errorMessage
		}

		records = append(records, record)
	}

	return records, nil
}

// GetDBStats возвращает статистику из базы данных
func (s *NomenclatureService) GetDBStats(db *database.DB) (map[string]interface{}, error) {
	var stats struct {
		Total     int
		Completed int
		Errors    int
	}

	// Общее количество записей
	row := db.QueryRow("SELECT COUNT(*) FROM catalog_items")
	if err := row.Scan(&stats.Total); err != nil {
		return nil, apperrors.NewInternalError("не удалось получить общее количество", err)
	}

	// Количество обработанных
	row = db.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE processing_status = 'completed'")
	if err := row.Scan(&stats.Completed); err != nil {
		return nil, apperrors.NewInternalError("не удалось получить количество завершенных", err)
	}

	// Количество с ошибками
	row = db.QueryRow("SELECT COUNT(*) FROM catalog_items WHERE processing_status = 'error'")
	if err := row.Scan(&stats.Errors); err != nil {
		return nil, apperrors.NewInternalError("не удалось получить количество ошибок", err)
	}

	return map[string]interface{}{
		"total":     stats.Total,
		"completed": stats.Completed,
		"errors":    stats.Errors,
		"pending":   stats.Total - stats.Completed - stats.Errors,
	}, nil
}

