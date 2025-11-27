package server

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"httpserver/database"
	_ "github.com/mattn/go-sqlite3"
)

// DatabaseDiagnostic структура для диагностики базы данных
type DatabaseDiagnostic struct {
	FilePath            string            `json:"file_path"`
	DatabaseID          int               `json:"database_id"`
	DatabaseName        string            `json:"database_name"`
	Exists              bool              `json:"exists"`
	Tables              []string          `json:"tables"`
	RecordCounts        map[string]int    `json:"record_counts"`
	HasCatalogItems     bool              `json:"has_catalog_items"`
	HasNomenclatureItems bool              `json:"has_nomenclature_items"`
	Has1CTables         bool              `json:"has_1c_tables"`
	Issues              []string          `json:"issues"`
	IsActive            bool              `json:"is_active"`
}

// UploadStatus структура для статуса upload записей
type UploadStatus struct {
	DatabaseID   int      `json:"database_id"`
	FileName     string   `json:"file_name"`
	UploadID     *int     `json:"upload_id"`
	ClientID     *int     `json:"client_id"`
	ProjectID    *int     `json:"project_id"`
	Status       string   `json:"status"` // 'missing', 'invalid', 'valid'
	CreatedAt    string   `json:"created_at,omitempty"`
	RecordsCount int      `json:"records_count,omitempty"`
	Issues       []string `json:"issues,omitempty"`
}

// ExtractionStatus структура для статуса извлечения данных
type ExtractionStatus struct {
	DatabaseID            int      `json:"database_id"`
	DatabaseName          string   `json:"database_name"`
	SourceTables          []string `json:"source_tables"`
	CatalogItemsCount     int      `json:"catalog_items_count"`
	NomenclatureItemsCount int     `json:"nomenclature_items_count"`
	ExtractionMethod      string   `json:"extraction_method"` // 'auto', 'manual', 'none'
	LastExtraction        string   `json:"last_extraction,omitempty"`
	Issues                []string `json:"issues"`
}

// DiagnosticNormalizationStatus структура для статуса нормализации в диагностике
type DiagnosticNormalizationStatus struct {
	ProjectID            int                    `json:"project_id"`
	NormalizedRecordsCount int                  `json:"normalized_records_count"`
	NormalizationSessions []NormalizationSession `json:"normalization_sessions"`
	ProjectDatabaseLinks  []ProjectDatabaseLink `json:"project_database_links"`
	Issues               []string               `json:"issues"`
}

// NormalizationSession информация о сессии нормализации
type NormalizationSession struct {
	ID              int    `json:"id"`
	Status          string `json:"status"`
	RecordsProcessed int   `json:"records_processed"`
	CreatedAt       string `json:"created_at"`
	FinishedAt      *string `json:"finished_at,omitempty"`
}

// ProjectDatabaseLink связь проекта с базой данных
type ProjectDatabaseLink struct {
	DatabaseID        int `json:"database_id"`
	NormalizedRecords int `json:"normalized_records"`
}

// CheckAllProjectDatabases проверяет все базы данных проекта
// Возвращает []interface{} для соответствия интерфейсу DiagnosticsServer
func (s *Server) CheckAllProjectDatabases(projectID, clientID int) ([]interface{}, error) {
	if s.serviceDB == nil {
		return nil, fmt.Errorf("service database not available")
	}

	// Получаем все базы данных проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	var diagnostics []interface{}
	for _, db := range databases {
		diagnostic := s.CheckDatabaseHealth(db)
		diagnostics = append(diagnostics, diagnostic)
	}

	return diagnostics, nil
}

// CheckDatabaseHealth проверяет здоровье базы данных
func (s *Server) CheckDatabaseHealth(db *database.ProjectDatabase) DatabaseDiagnostic {
	diagnostic := DatabaseDiagnostic{
		FilePath:     db.FilePath,
		DatabaseID:   db.ID,
		DatabaseName: db.Name,
		IsActive:     db.IsActive,
		RecordCounts: make(map[string]int),
		Issues:       []string{},
	}

	// Проверяем существование файла
	if _, err := os.Stat(db.FilePath); err != nil {
		diagnostic.Exists = false
		diagnostic.Issues = append(diagnostic.Issues, fmt.Sprintf("Файл базы данных не существует: %v", err))
		return diagnostic
	}

	diagnostic.Exists = true

	// Подключаемся к базе данных
	conn, err := sql.Open("sqlite3", db.FilePath)
	if err != nil {
		diagnostic.Issues = append(diagnostic.Issues, fmt.Sprintf("Не удалось подключиться к БД: %v", err))
		return diagnostic
	}
	defer conn.Close()

	// Получаем список таблиц
	tables, err := s.getDatabaseTables(conn)
	if err != nil {
		diagnostic.Issues = append(diagnostic.Issues, fmt.Sprintf("Не удалось получить список таблиц: %v", err))
		return diagnostic
	}

	diagnostic.Tables = tables

	// Проверяем наличие ключевых таблиц
	diagnostic.HasCatalogItems = containsTable(tables, "catalog_items")
	diagnostic.HasNomenclatureItems = containsTable(tables, "nomenclature_items")
	diagnostic.Has1CTables = s.has1CTables(tables)

	// Подсчитываем записи в ключевых таблицах
	keyTables := []string{"catalog_items", "nomenclature_items", "normalized_data"}
	for _, table := range keyTables {
		if containsTable(tables, table) {
			count, err := s.countTableRecords(conn, table)
			if err == nil {
				diagnostic.RecordCounts[table] = count
			}
		}
	}

	// Выявляем проблемы
	if !diagnostic.HasCatalogItems && !diagnostic.HasNomenclatureItems && !diagnostic.Has1CTables {
		diagnostic.Issues = append(diagnostic.Issues, "Не найдены таблицы catalog_items, nomenclature_items или исходные таблицы 1С")
	}

	if diagnostic.HasCatalogItems && diagnostic.RecordCounts["catalog_items"] == 0 {
		diagnostic.Issues = append(diagnostic.Issues, "Таблица catalog_items пуста")
	}

	if diagnostic.HasNomenclatureItems && diagnostic.RecordCounts["nomenclature_items"] == 0 {
		diagnostic.Issues = append(diagnostic.Issues, "Таблица nomenclature_items пуста")
	}

	return diagnostic
}

// getDatabaseTables получает список таблиц в базе данных
func (s *Server) getDatabaseTables(conn *sql.DB) ([]string, error) {
	rows, err := conn.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		tables = append(tables, name)
	}

	return tables, nil
}

// countTableRecords подсчитывает записи в таблице
func (s *Server) countTableRecords(conn *sql.DB, tableName string) (int, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err := conn.QueryRow(query).Scan(&count)
	return count, err
}

// containsTable проверяет наличие таблицы в списке
func containsTable(tables []string, tableName string) bool {
	for _, t := range tables {
		if strings.EqualFold(t, tableName) {
			return true
		}
	}
	return false
}

// has1CTables проверяет наличие таблиц 1С
func (s *Server) has1CTables(tables []string) bool {
	// Типичные префиксы таблиц 1С
	oneCPrefixes := []string{"_1C", "Catalog", "Document", "InformationRegister", "AccumulationRegister", "Enum"}
	for _, table := range tables {
		for _, prefix := range oneCPrefixes {
			if strings.HasPrefix(table, prefix) {
				return true
			}
		}
	}
	return false
}

// CheckUploadRecords проверяет upload записи для проекта
// Возвращает []interface{} для соответствия интерфейсу DiagnosticsServer
func (s *Server) CheckUploadRecords(projectID, clientID int) ([]interface{}, error) {
	if s.serviceDB == nil {
		return nil, fmt.Errorf("service database not available")
	}

	// Получаем все базы данных проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	var uploadStatuses []interface{}
	for _, db := range databases {
		status := s.checkUploadForDatabase(db, projectID, clientID)
		uploadStatuses = append(uploadStatuses, interface{}(status))
	}

	return uploadStatuses, nil
}

// checkUploadForDatabase проверяет upload запись для конкретной базы данных
func (s *Server) checkUploadForDatabase(db *database.ProjectDatabase, projectID, clientID int) UploadStatus {
	status := UploadStatus{
		DatabaseID: db.ID,
		FileName:   db.Name,
		Status:     "missing",
		Issues:     []string{},
	}

	if s.db == nil {
		status.Status = "error"
		return status
	}

	// Ищем upload записи для этой базы данных (может быть несколько)
	uploads, err := s.db.GetUploadsByDatabaseID(db.ID)
	if err != nil {
		status.Status = "error"
		status.Issues = []string{fmt.Sprintf("Ошибка получения upload записей: %v", err)}
		return status
	}

	if len(uploads) == 0 {
		status.Status = "missing"
		status.Issues = []string{"Upload запись отсутствует для этой базы данных"}
		return status
	}

	// Берем последнюю upload запись
	upload := uploads[0]
	status.UploadID = &upload.ID
	if upload.ClientID != nil {
		status.ClientID = upload.ClientID
	}
	if upload.ProjectID != nil {
		status.ProjectID = upload.ProjectID
	}
	status.CreatedAt = upload.StartedAt.Format(time.RFC3339)

	// Проверяем валидность
	if upload.ClientID != nil && *upload.ClientID != clientID {
		status.Status = "invalid"
		status.Issues = []string{fmt.Sprintf("Несоответствие client_id: ожидается %d, получено %d", clientID, *upload.ClientID)}
		return status
	}
	if upload.ProjectID != nil && *upload.ProjectID != projectID {
		status.Status = "invalid"
		status.Issues = []string{fmt.Sprintf("Несоответствие project_id: ожидается %d, получено %d", projectID, *upload.ProjectID)}
		return status
	}

	// Получаем количество записей из статистики
	stats, err := s.db.GetUploadStatsByDatabaseID(db.ID)
	if err == nil {
		if totalItems, ok := stats["total_items"].(int); ok {
			status.RecordsCount = totalItems
		}
	}

	status.Status = "valid"
	return status
}

// CreateMissingUploads создает недостающие upload записи
func (s *Server) CreateMissingUploads(projectID, clientID int) (int, error) {
	if s.serviceDB == nil || s.db == nil {
		return 0, fmt.Errorf("databases not available")
	}

	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get project databases: %w", err)
	}

	fixedCount := 0
	for _, db := range databases {
		// Проверяем существование upload записи
		uploads, err := s.db.GetUploadsByDatabaseID(db.ID)
		if err != nil || len(uploads) == 0 {
			// Создаем upload запись
			// Используем CreateUploadWithDatabase с database_id
			dbID := db.ID
			uploadUUID := fmt.Sprintf("diagnostic-%d-%d", db.ID, time.Now().Unix())
			upload, err := s.db.CreateUploadWithDatabase(
				uploadUUID,
				"8.3", // версия по умолчанию
				"Diagnostic Import",
				&dbID,
				"", "", "", 1, "", "", "", nil,
			)
			if err != nil {
				log.Printf("Failed to create upload for database %d: %v", db.ID, err)
				continue
			}
			log.Printf("Created upload record %d for database %d (project %d, client %d)", upload.ID, db.ID, projectID, clientID)
			fixedCount++
		}
	}

	return fixedCount, nil
}

// CheckExtractionStatus проверяет статус извлечения данных
// Возвращает []interface{} для соответствия интерфейсу DiagnosticsServer
func (s *Server) CheckExtractionStatus(projectID, clientID int) ([]interface{}, error) {
	if s.serviceDB == nil {
		return nil, fmt.Errorf("service database not available")
	}

	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	var statuses []interface{}
	for _, db := range databases {
		status := s.checkExtractionForDatabase(db)
		statuses = append(statuses, status)
	}

	return statuses, nil
}

// checkExtractionForDatabase проверяет извлечение данных для конкретной базы
func (s *Server) checkExtractionForDatabase(db *database.ProjectDatabase) ExtractionStatus {
	status := ExtractionStatus{
		DatabaseID:   db.ID,
		DatabaseName: db.Name,
		Issues:       []string{},
	}

	if _, err := os.Stat(db.FilePath); err != nil {
		status.Issues = append(status.Issues, "Файл базы данных не существует")
		return status
	}

	conn, err := sql.Open("sqlite3", db.FilePath)
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Не удалось подключиться к БД: %v", err))
		return status
	}
	defer conn.Close()

	// Получаем список таблиц
	tables, err := s.getDatabaseTables(conn)
	if err != nil {
		status.Issues = append(status.Issues, fmt.Sprintf("Не удалось получить список таблиц: %v", err))
		return status
	}

	status.SourceTables = tables

	// Проверяем наличие данных в catalog_items и nomenclature_items
	if containsTable(tables, "catalog_items") {
		count, err := s.countTableRecords(conn, "catalog_items")
		if err == nil {
			status.CatalogItemsCount = count
		}
	} else {
		status.Issues = append(status.Issues, "Таблица catalog_items не найдена")
	}

	if containsTable(tables, "nomenclature_items") {
		count, err := s.countTableRecords(conn, "nomenclature_items")
		if err == nil {
			status.NomenclatureItemsCount = count
		}
	} else {
		status.Issues = append(status.Issues, "Таблица nomenclature_items не найдена")
	}

	// Определяем метод извлечения
	if status.CatalogItemsCount > 0 || status.NomenclatureItemsCount > 0 {
		status.ExtractionMethod = "auto"
	} else if s.has1CTables(tables) {
		status.ExtractionMethod = "none"
		status.Issues = append(status.Issues, "Данные не извлечены из исходных таблиц 1С")
	} else {
		status.ExtractionMethod = "none"
		status.Issues = append(status.Issues, "Исходные таблицы не найдены")
	}

	return status
}

// CheckNormalizationStatus проверяет статус нормализации
func (s *Server) CheckNormalizationStatus(projectID int) (interface{}, error) {
	if s.serviceDB == nil {
		return nil, fmt.Errorf("service database not available")
	}

	status := &DiagnosticNormalizationStatus{
		ProjectID:            projectID,
		NormalizationSessions: []NormalizationSession{},
		ProjectDatabaseLinks:  []ProjectDatabaseLink{},
		Issues:               []string{},
	}

	// Получаем базы данных проекта
	databases, err := s.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	// Получаем сессии нормализации
	for _, db := range databases {
		session, err := s.serviceDB.GetLastNormalizationSession(db.ID)
		if err == nil && session != nil {
			finishedAt := ""
			if session.FinishedAt != nil {
				finishedAt = session.FinishedAt.Format(time.RFC3339)
			}
			// Получаем количество обработанных записей из normalized_data
			recordsProcessed := 0
			if s.db != nil {
				count, err := s.getNormalizedDataCountByProjectAndDatabase(projectID, db.ID)
				if err == nil {
					recordsProcessed = count
				}
			}
			status.NormalizationSessions = append(status.NormalizationSessions, NormalizationSession{
				ID:              session.ID,
				Status:          session.Status,
				RecordsProcessed: recordsProcessed,
				CreatedAt:       session.StartedAt.Format(time.RFC3339),
				FinishedAt:      &finishedAt,
			})
		}

		// Подсчитываем нормализованные записи для этой БД
		if s.db != nil {
			count, err := s.getNormalizedDataCountByProjectAndDatabase(projectID, db.ID)
			if err == nil {
				status.ProjectDatabaseLinks = append(status.ProjectDatabaseLinks, ProjectDatabaseLink{
					DatabaseID:        db.ID,
					NormalizedRecords: count,
				})
				status.NormalizedRecordsCount += count
			}
		}
	}

	// Выявляем проблемы
	if status.NormalizedRecordsCount == 0 {
		status.Issues = append(status.Issues, "Нет нормализованных записей для проекта")
	}

	if len(status.NormalizationSessions) == 0 {
		status.Issues = append(status.Issues, "Нет сессий нормализации для проекта")
	}

	return status, nil
}

// getNormalizedDataCountByProjectAndDatabase подсчитывает нормализованные записи для проекта и базы данных
func (s *Server) getNormalizedDataCountByProjectAndDatabase(projectID, databaseID int) (int, error) {
	if s.db == nil {
		return 0, fmt.Errorf("database not available")
	}

	// Получаем путь к базе данных проекта
	dbInfo, err := s.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		return 0, fmt.Errorf("failed to get database info: %w", err)
	}

	// Подключаемся к базе данных проекта
	conn, err := sql.Open("sqlite3", dbInfo.FilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open database: %w", err)
	}
	defer conn.Close()

	// Подсчитываем записи в normalized_data с project_id
	var count int
	query := "SELECT COUNT(*) FROM normalized_data WHERE project_id = ?"
	err = conn.QueryRow(query, projectID).Scan(&count)
	if err != nil {
		// Если таблицы нет или project_id не заполнен, проверяем общее количество
		query = "SELECT COUNT(*) FROM normalized_data"
		err = conn.QueryRow(query).Scan(&count)
		if err != nil {
			return 0, nil // Таблица может отсутствовать
		}
	}

	return count, nil
}

