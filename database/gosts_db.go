package database

import (
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// GostsDB обертка для работы с базой данных ГОСТов
type GostsDB struct {
	conn             *sql.DB
	tableCreateMutex sync.Mutex
}

// NewGostsDB создает новое подключение к базе данных ГОСТов
func NewGostsDB(dbPath string) (*GostsDB, error) {
	return NewGostsDBWithConfig(dbPath, DBConfig{})
}

// NewGostsDBWithConfig создает новое подключение к базе данных ГОСТов с конфигурацией
func NewGostsDBWithConfig(dbPath string, config DBConfig) (*GostsDB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open gosts database: %w", err)
	}

	// Настройка connection pooling
	if config.MaxOpenConns > 0 {
		conn.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		conn.SetMaxOpenConns(25)
	}

	if config.MaxIdleConns > 0 {
		conn.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		conn.SetMaxIdleConns(5)
	}

	if config.ConnMaxLifetime > 0 {
		conn.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		conn.SetConnMaxLifetime(5 * time.Minute)
	}

	// Проверяем подключение
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping gosts database: %w", err)
	}

	// Включаем поддержку FOREIGN KEY constraints в SQLite
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Убеждаемся, что SQLite использует UTF-8 для текстовых данных
	// SQLite по умолчанию использует UTF-8, но явно указываем это
	if _, err := conn.Exec("PRAGMA encoding = 'UTF-8'"); err != nil {
		// Это не критично, но логируем
		log.Printf("Warning: failed to set UTF-8 encoding: %v", err)
	}

	gostsDB := &GostsDB{conn: conn}

	// Инициализируем схему
	if err := InitGostsSchema(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize gosts schema: %w", err)
	}

	// Выполняем миграции для существующих баз данных
	if err := MigrateGostsSchema(conn); err != nil {
		// Логируем ошибку, но не прерываем инициализацию
		// Миграции могут быть идемпотентными и не критичными
		log.Printf("Warning: failed to run GOSTs migrations: %v", err)
	}

	return gostsDB, nil
}

// Close закрывает подключение к базе данных ГОСТов
func (db *GostsDB) Close() error {
	return db.conn.Close()
}

// GetDB возвращает указатель на sql.DB для прямого доступа
func (db *GostsDB) GetDB() *sql.DB {
	return db.conn
}

// GetConnection возвращает указатель на sql.DB для прямого доступа
func (db *GostsDB) GetConnection() *sql.DB {
	return db.conn
}

// Gost структура ГОСТа
type Gost struct {
	ID            int        `json:"id"`
	GostNumber    string     `json:"gost_number"`
	Title         string     `json:"title"`
	AdoptionDate  *time.Time `json:"adoption_date"`
	EffectiveDate *time.Time `json:"effective_date"`
	Status        string     `json:"status"`
	SourceType    string     `json:"source_type"`
	SourceID      *int       `json:"source_id"`
	SourceURL     string     `json:"source_url"`
	Description   string     `json:"description"`
	Keywords      string     `json:"keywords"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// GostDocument структура документа ГОСТа
type GostDocument struct {
	ID         int       `json:"id"`
	GostID     int       `json:"gost_id"`
	FilePath   string    `json:"file_path"`
	FileType   string    `json:"file_type"`
	FileSize   int64     `json:"file_size"`
	UploadedAt time.Time `json:"uploaded_at"`
}

// GostSource структура источника данных ГОСТов
type GostSource struct {
	ID           int        `json:"id"`
	SourceName   string     `json:"source_name"`
	SourceURL    string     `json:"source_url"`
	LastSyncDate *time.Time `json:"last_sync_date"`
	RecordsCount int        `json:"records_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateOrUpdateGost создает или обновляет ГОСТ
func (db *GostsDB) CreateOrUpdateGost(gost *Gost) (*Gost, error) {
	query := `
		INSERT INTO gosts (gost_number, title, adoption_date, effective_date, status, 
		                   source_type, source_id, source_url, description, keywords, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(gost_number) DO UPDATE SET
			title = excluded.title,
			adoption_date = excluded.adoption_date,
			effective_date = excluded.effective_date,
			status = excluded.status,
			source_type = excluded.source_type,
			source_id = excluded.source_id,
			source_url = excluded.source_url,
			description = excluded.description,
			keywords = excluded.keywords,
			updated_at = CURRENT_TIMESTAMP
	`

	result, err := db.conn.Exec(query,
		gost.GostNumber, gost.Title, gost.AdoptionDate, gost.EffectiveDate,
		gost.Status, gost.SourceType, gost.SourceID, gost.SourceURL,
		gost.Description, gost.Keywords)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update gost: %w", err)
	}

	// Получаем ID записи (либо новый, либо существующий)
	var id int64
	insertID, err := result.LastInsertId()
	if err == nil && insertID > 0 {
		id = insertID
	} else {
		// Если это UPDATE, получаем ID по номеру ГОСТа
		err := db.conn.QueryRow("SELECT id FROM gosts WHERE gost_number = ?", gost.GostNumber).Scan(&id)
		if err != nil {
			// Если не удалось получить ID, возвращаем ошибку
			return nil, fmt.Errorf("failed to get gost ID: %w", err)
		}
	}

	// Получаем полную запись
	return db.GetGost(int(id))
}

// GetGost получает ГОСТ по ID
func (db *GostsDB) GetGost(id int) (*Gost, error) {
	query := `
		SELECT id, gost_number, title, adoption_date, effective_date, status,
		       source_type, source_id, source_url, description, keywords,
		       created_at, updated_at
		FROM gosts WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	gost := &Gost{}

	var adoptionDate, effectiveDate sql.NullTime
	var sourceID sql.NullInt64

	var createdAt sql.NullTime

	err := row.Scan(
		&gost.ID, &gost.GostNumber, &gost.Title,
		&adoptionDate, &effectiveDate,
		&gost.Status, &gost.SourceType, &sourceID,
		&gost.SourceURL, &gost.Description, &gost.Keywords,
		&createdAt, &gost.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get gost: %w", err)
	}

	// Обрабатываем created_at
	if createdAt.Valid {
		gost.CreatedAt = createdAt.Time
	} else {
		gost.CreatedAt = gost.UpdatedAt // Используем updated_at если created_at NULL
	}

	if adoptionDate.Valid {
		gost.AdoptionDate = &adoptionDate.Time
	}
	if effectiveDate.Valid {
		gost.EffectiveDate = &effectiveDate.Time
	}
	if sourceID.Valid {
		id := int(sourceID.Int64)
		gost.SourceID = &id
	}

	return gost, nil
}

// GetGostByNumber получает ГОСТ по номеру
func (db *GostsDB) GetGostByNumber(gostNumber string) (*Gost, error) {
	query := `
		SELECT id, gost_number, title, adoption_date, effective_date, status,
		       source_type, source_id, source_url, description, keywords,
		       created_at, updated_at
		FROM gosts WHERE gost_number = ?
	`

	row := db.conn.QueryRow(query, gostNumber)
	gost := &Gost{}

	var adoptionDate, effectiveDate sql.NullTime
	var sourceID sql.NullInt64
	var createdAt sql.NullTime

	err := row.Scan(
		&gost.ID, &gost.GostNumber, &gost.Title,
		&adoptionDate, &effectiveDate,
		&gost.Status, &gost.SourceType, &sourceID,
		&gost.SourceURL, &gost.Description, &gost.Keywords,
		&createdAt, &gost.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get gost by number: %w", err)
	}

	// Обрабатываем created_at
	if createdAt.Valid {
		gost.CreatedAt = createdAt.Time
	} else {
		gost.CreatedAt = gost.UpdatedAt
	}

	if adoptionDate.Valid {
		gost.AdoptionDate = &adoptionDate.Time
	}
	if effectiveDate.Valid {
		gost.EffectiveDate = &effectiveDate.Time
	}
	if sourceID.Valid {
		id := int(sourceID.Int64)
		gost.SourceID = &id
	}

	return gost, nil
}

// SearchGosts выполняет поиск ГОСТов
func (db *GostsDB) SearchGosts(
	query string,
	limit, offset int,
	status, sourceType,
	adoptionFrom, adoptionTo, effectiveFrom, effectiveTo string,
) ([]*Gost, int, error) {
	whereClause := "(gost_number LIKE ? OR title LIKE ? OR keywords LIKE ?)"
	args := []interface{}{}

	searchPattern := "%" + query + "%"
	args = append(args, searchPattern, searchPattern, searchPattern)

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}
	if sourceType != "" {
		whereClause += " AND source_type = ?"
		args = append(args, sourceType)
	}
	if adoptionFrom != "" {
		whereClause += " AND adoption_date IS NOT NULL AND date(adoption_date) >= date(?)"
		args = append(args, adoptionFrom)
	}
	if adoptionTo != "" {
		whereClause += " AND adoption_date IS NOT NULL AND date(adoption_date) <= date(?)"
		args = append(args, adoptionTo)
	}
	if effectiveFrom != "" {
		whereClause += " AND effective_date IS NOT NULL AND date(effective_date) >= date(?)"
		args = append(args, effectiveFrom)
	}
	if effectiveTo != "" {
		whereClause += " AND effective_date IS NOT NULL AND date(effective_date) <= date(?)"
		args = append(args, effectiveTo)
	}

	searchQuery := fmt.Sprintf(`
		SELECT id, gost_number, title, adoption_date, effective_date, status,
		       source_type, source_id, source_url, description, keywords,
		       created_at, updated_at
		FROM gosts
		WHERE %s
		ORDER BY gost_number
		LIMIT ? OFFSET ?
	`, whereClause)

	argsWithPagination := append(append([]interface{}{}, args...), limit, offset)
	rows, err := db.conn.Query(searchQuery, argsWithPagination...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search gosts: %w", err)
	}
	defer rows.Close()

	var gosts []*Gost
	for rows.Next() {
		gost := &Gost{}
		var adoptionDate, effectiveDate sql.NullTime
		var sourceID sql.NullInt64

		err := rows.Scan(
			&gost.ID, &gost.GostNumber, &gost.Title,
			&adoptionDate, &effectiveDate,
			&gost.Status, &gost.SourceType, &sourceID,
			&gost.SourceURL, &gost.Description, &gost.Keywords,
			&gost.CreatedAt, &gost.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan gost: %w", err)
		}

		if adoptionDate.Valid {
			gost.AdoptionDate = &adoptionDate.Time
		}
		if effectiveDate.Valid {
			gost.EffectiveDate = &effectiveDate.Time
		}
		if sourceID.Valid {
			id := int(sourceID.Int64)
			gost.SourceID = &id
		}

		gosts = append(gosts, gost)
	}

	// Получаем общее количество
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM gosts
		WHERE %s
	`, whereClause)
	var total int
	err = db.conn.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count gosts: %w", err)
	}

	return gosts, total, nil
}

// ListGosts возвращает список ГОСТов с пагинацией
func (db *GostsDB) ListGosts(
	limit, offset int,
	status, sourceType,
	adoptionFrom, adoptionTo, effectiveFrom, effectiveTo string,
) ([]*Gost, int, error) {
	whereClause := "1=1"
	args := []interface{}{}

	if status != "" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}
	if sourceType != "" {
		whereClause += " AND source_type = ?"
		args = append(args, sourceType)
	}
	if adoptionFrom != "" {
		whereClause += " AND adoption_date IS NOT NULL AND date(adoption_date) >= date(?)"
		args = append(args, adoptionFrom)
	}
	if adoptionTo != "" {
		whereClause += " AND adoption_date IS NOT NULL AND date(adoption_date) <= date(?)"
		args = append(args, adoptionTo)
	}
	if effectiveFrom != "" {
		whereClause += " AND effective_date IS NOT NULL AND date(effective_date) >= date(?)"
		args = append(args, effectiveFrom)
	}
	if effectiveTo != "" {
		whereClause += " AND effective_date IS NOT NULL AND date(effective_date) <= date(?)"
		args = append(args, effectiveTo)
	}

	query := fmt.Sprintf(`
		SELECT id, gost_number, title, adoption_date, effective_date, status,
		       source_type, source_id, source_url, description, keywords,
		       created_at, updated_at
		FROM gosts
		WHERE %s
		ORDER BY gost_number
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list gosts: %w", err)
	}
	defer rows.Close()

	var gosts []*Gost
	for rows.Next() {
		gost := &Gost{}
		var adoptionDate, effectiveDate sql.NullTime
		var sourceID sql.NullInt64
		var createdAt sql.NullTime

		err := rows.Scan(
			&gost.ID, &gost.GostNumber, &gost.Title,
			&adoptionDate, &effectiveDate,
			&gost.Status, &gost.SourceType, &sourceID,
			&gost.SourceURL, &gost.Description, &gost.Keywords,
			&createdAt, &gost.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan gost: %w", err)
		}

		// Обрабатываем created_at
		if createdAt.Valid {
			gost.CreatedAt = createdAt.Time
		} else {
			gost.CreatedAt = gost.UpdatedAt
		}

		if adoptionDate.Valid {
			gost.AdoptionDate = &adoptionDate.Time
		}
		if effectiveDate.Valid {
			gost.EffectiveDate = &effectiveDate.Time
		}
		if sourceID.Valid {
			id := int(sourceID.Int64)
			gost.SourceID = &id
		}

		gosts = append(gosts, gost)
	}

	// Получаем общее количество
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM gosts WHERE %s", whereClause)
	countArgs := args[:len(args)-2] // Убираем limit и offset
	var total int
	err = db.conn.QueryRow(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count gosts: %w", err)
	}

	return gosts, total, nil
}

// CreateOrUpdateSource создает или обновляет источник данных
func (db *GostsDB) CreateOrUpdateSource(source *GostSource) (*GostSource, error) {
	// Проверяем наличие колонок created_at и updated_at
	var hasUpdatedAt bool
	db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('gost_sources')
			WHERE name='updated_at'
		)
	`).Scan(&hasUpdatedAt)

	var query string
	if hasUpdatedAt {
		query = `
			INSERT INTO gost_sources (source_name, source_url, last_sync_date, records_count, updated_at)
			VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(source_name) DO UPDATE SET
				source_url = excluded.source_url,
				last_sync_date = excluded.last_sync_date,
				records_count = excluded.records_count,
				updated_at = CURRENT_TIMESTAMP
		`
	} else {
		query = `
			INSERT INTO gost_sources (source_name, source_url, last_sync_date, records_count)
			VALUES (?, ?, ?, ?)
			ON CONFLICT(source_name) DO UPDATE SET
				source_url = excluded.source_url,
				last_sync_date = excluded.last_sync_date,
				records_count = excluded.records_count
		`
	}

	result, err := db.conn.Exec(query, source.SourceName, source.SourceURL, source.LastSyncDate, source.RecordsCount)
	if err != nil {
		return nil, fmt.Errorf("failed to create or update source: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		// Если это UPDATE, получаем ID по имени
		err := db.conn.QueryRow("SELECT id FROM gost_sources WHERE source_name = ?", source.SourceName).Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to get source ID: %w", err)
		}
	}

	return db.GetSource(int(id))
}

// GetSource получает источник по ID
func (db *GostsDB) GetSource(id int) (*GostSource, error) {
	// Проверяем наличие колонки updated_at
	var hasUpdatedAt bool
	db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('gost_sources')
			WHERE name='updated_at'
		)
	`).Scan(&hasUpdatedAt)

	var query string
	if hasUpdatedAt {
		query = `
			SELECT id, source_name, source_url, last_sync_date, records_count, created_at, updated_at
			FROM gost_sources WHERE id = ?
		`
	} else {
		query = `
			SELECT id, source_name, source_url, last_sync_date, records_count
			FROM gost_sources WHERE id = ?
		`
	}

	row := db.conn.QueryRow(query, id)
	source := &GostSource{}

	var lastSyncDate sql.NullTime
	var createdAt sql.NullTime

	var err error
	if hasUpdatedAt {
		err = row.Scan(
			&source.ID, &source.SourceName, &source.SourceURL,
			&lastSyncDate, &source.RecordsCount,
			&createdAt, &source.UpdatedAt,
		)
		if createdAt.Valid {
			source.CreatedAt = createdAt.Time
		} else {
			source.CreatedAt = source.UpdatedAt
		}
	} else {
		err = row.Scan(
			&source.ID, &source.SourceName, &source.SourceURL,
			&lastSyncDate, &source.RecordsCount,
		)
		source.CreatedAt = time.Now()
		source.UpdatedAt = time.Now()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get source: %w", err)
	}

	if lastSyncDate.Valid {
		source.LastSyncDate = &lastSyncDate.Time
	}

	return source, nil
}

// GetSourceByName получает источник по имени
func (db *GostsDB) GetSourceByName(sourceName string) (*GostSource, error) {
	query := `
		SELECT id, source_name, source_url, last_sync_date, records_count, updated_at
		FROM gost_sources WHERE source_name = ?
	`

	row := db.conn.QueryRow(query, sourceName)
	source := &GostSource{}

	var lastSyncDate sql.NullTime
	var createdAt sql.NullTime

	err := row.Scan(
		&source.ID, &source.SourceName, &source.SourceURL,
		&lastSyncDate, &source.RecordsCount,
		&source.UpdatedAt,
	)

	if createdAt.Valid {
		source.CreatedAt = createdAt.Time
	} else {
		source.CreatedAt = source.UpdatedAt
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get source by name: %w", err)
	}

	if lastSyncDate.Valid {
		source.LastSyncDate = &lastSyncDate.Time
	}

	return source, nil
}

// AddDocument добавляет документ к ГОСТу
func (db *GostsDB) AddDocument(doc *GostDocument) (*GostDocument, error) {
	query := `
		INSERT INTO gost_documents (gost_id, file_path, file_type, file_size)
		VALUES (?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, doc.GostID, doc.FilePath, doc.FileType, doc.FileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to add document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get document ID: %w", err)
	}

	return db.GetDocument(int(id))
}

// GetDocument получает документ по ID
func (db *GostsDB) GetDocument(id int) (*GostDocument, error) {
	query := `
		SELECT id, gost_id, file_path, file_type, file_size, uploaded_at
		FROM gost_documents WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	doc := &GostDocument{}

	err := row.Scan(
		&doc.ID, &doc.GostID, &doc.FilePath, &doc.FileType, &doc.FileSize, &doc.UploadedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	return doc, nil
}

// GetDocumentsByGostID получает все документы для ГОСТа
func (db *GostsDB) GetDocumentsByGostID(gostID int) ([]*GostDocument, error) {
	query := `
		SELECT id, gost_id, file_path, file_type, file_size, uploaded_at
		FROM gost_documents WHERE gost_id = ?
		ORDER BY uploaded_at DESC
	`

	rows, err := db.conn.Query(query, gostID)
	if err != nil {
		return nil, fmt.Errorf("failed to get documents: %w", err)
	}
	defer rows.Close()

	var documents []*GostDocument
	for rows.Next() {
		doc := &GostDocument{}
		err := rows.Scan(
			&doc.ID, &doc.GostID, &doc.FilePath, &doc.FileType, &doc.FileSize, &doc.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}
		documents = append(documents, doc)
	}

	return documents, nil
}

// GetStatistics возвращает статистику по базе ГОСТов
func (db *GostsDB) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Общее количество ГОСТов
	var totalGosts int
	err := db.conn.QueryRow("SELECT COUNT(*) FROM gosts").Scan(&totalGosts)
	if err != nil {
		return nil, fmt.Errorf("failed to count gosts: %w", err)
	}
	stats["total_gosts"] = totalGosts

	// Количество по статусам
	statusQuery := `
		SELECT status, COUNT(*) as count
		FROM gosts
		WHERE status IS NOT NULL AND status != ''
		GROUP BY status
	`
	rows, err := db.conn.Query(statusQuery)
	if err == nil {
		defer rows.Close()
		statusCounts := make(map[string]int)
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err == nil {
				statusCounts[status] = count
			}
		}
		stats["by_status"] = statusCounts
	}

	// Количество по типам источников
	sourceTypeQuery := `
		SELECT source_type, COUNT(*) as count
		FROM gosts
		WHERE source_type IS NOT NULL AND source_type != ''
		GROUP BY source_type
	`
	rows, err = db.conn.Query(sourceTypeQuery)
	if err == nil {
		defer rows.Close()
		sourceTypeCounts := make(map[string]int)
		for rows.Next() {
			var sourceType string
			var count int
			if err := rows.Scan(&sourceType, &count); err == nil {
				sourceTypeCounts[sourceType] = count
			}
		}
		stats["by_source_type"] = sourceTypeCounts
	}

	// Количество документов
	var totalDocuments int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM gost_documents").Scan(&totalDocuments)
	if err == nil {
		stats["total_documents"] = totalDocuments
	}

	// Количество источников
	var totalSources int
	err = db.conn.QueryRow("SELECT COUNT(*) FROM gost_sources").Scan(&totalSources)
	if err == nil {
		stats["total_sources"] = totalSources
	}

	return stats, nil
}
