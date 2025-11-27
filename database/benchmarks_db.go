package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// BenchmarksDB обертка для работы с БД эталонов
type BenchmarksDB struct {
	conn *sql.DB
}

// NewBenchmarksDB создает новое подключение к БД эталонов
func NewBenchmarksDB(path string) (*BenchmarksDB, error) {
	db, err := CreateBenchmarksDatabase(path)
	if err != nil {
		return nil, err
	}

	return &BenchmarksDB{conn: db}, nil
}

// Close закрывает подключение к БД эталонов
func (db *BenchmarksDB) Close() error {
	return db.conn.Close()
}

// GetConnection возвращает указатель на sql.DB для прямого доступа
func (db *BenchmarksDB) GetConnection() *sql.DB {
	return db.conn
}

// Benchmark структура эталонной записи
type Benchmark struct {
	ID             string                 `json:"id"`
	EntityType     string                 `json:"entity_type"`
	Name           string                 `json:"name"`
	Data           map[string]interface{} `json:"data"`
	SourceUploadID *int                   `json:"source_upload_id,omitempty"`
	SourceClientID *int                   `json:"source_client_id,omitempty"`
	IsActive       bool                   `json:"is_active"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Variations     []string               `json:"variations,omitempty"`
}

// BenchmarkVariation структура вариации названия
type BenchmarkVariation struct {
	ID          int    `json:"id"`
	BenchmarkID string `json:"benchmark_id"`
	Variation   string `json:"variation"`
}

// CreateBenchmark создает новый эталон
func (db *BenchmarksDB) CreateBenchmark(benchmark *Benchmark) error {
	// Сериализуем data в JSON
	dataJSON, err := json.Marshal(benchmark.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark data: %w", err)
	}

	query := `
		INSERT INTO benchmarks (id, entity_type, name, data, source_upload_id, source_client_id, is_active, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	now := time.Now()
	_, err = db.conn.Exec(query,
		benchmark.ID,
		benchmark.EntityType,
		benchmark.Name,
		string(dataJSON),
		benchmark.SourceUploadID,
		benchmark.SourceClientID,
		benchmark.IsActive,
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to create benchmark: %w", err)
	}

	// Добавляем вариации, если они есть
	if len(benchmark.Variations) > 0 {
		if err := db.AddVariations(benchmark.ID, benchmark.Variations); err != nil {
			return fmt.Errorf("failed to add variations: %w", err)
		}
	}

	return nil
}

// GetBenchmark получает эталон по ID
func (db *BenchmarksDB) GetBenchmark(id string) (*Benchmark, error) {
	query := `
		SELECT id, entity_type, name, data, source_upload_id, source_client_id, is_active, created_at, updated_at
		FROM benchmarks
		WHERE id = ?
	`

	var benchmark Benchmark
	var dataJSON string
	var sourceUploadID, sourceClientID sql.NullInt64
	var createdAt, updatedAt time.Time

	err := db.conn.QueryRow(query, id).Scan(
		&benchmark.ID,
		&benchmark.EntityType,
		&benchmark.Name,
		&dataJSON,
		&sourceUploadID,
		&sourceClientID,
		&benchmark.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("benchmark not found")
		}
		return nil, fmt.Errorf("failed to get benchmark: %w", err)
	}

	// Десериализуем JSON
	if err := json.Unmarshal([]byte(dataJSON), &benchmark.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal benchmark data: %w", err)
	}

	if sourceUploadID.Valid {
		id := int(sourceUploadID.Int64)
		benchmark.SourceUploadID = &id
	}
	if sourceClientID.Valid {
		id := int(sourceClientID.Int64)
		benchmark.SourceClientID = &id
	}

	benchmark.CreatedAt = createdAt
	benchmark.UpdatedAt = updatedAt

	// Загружаем вариации
	variations, err := db.GetVariations(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get variations: %w", err)
	}
	benchmark.Variations = variations

	return &benchmark, nil
}

// UpdateBenchmark обновляет эталон
func (db *BenchmarksDB) UpdateBenchmark(benchmark *Benchmark) error {
	// Сериализуем data в JSON
	dataJSON, err := json.Marshal(benchmark.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal benchmark data: %w", err)
	}

	query := `
		UPDATE benchmarks
		SET entity_type = ?, name = ?, data = ?, source_upload_id = ?, source_client_id = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`

	_, err = db.conn.Exec(query,
		benchmark.EntityType,
		benchmark.Name,
		string(dataJSON),
		benchmark.SourceUploadID,
		benchmark.SourceClientID,
		benchmark.IsActive,
		time.Now(),
		benchmark.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update benchmark: %w", err)
	}

	// Обновляем вариации
	// Сначала удаляем старые
	if _, err := db.conn.Exec("DELETE FROM benchmark_variations WHERE benchmark_id = ?", benchmark.ID); err != nil {
		return fmt.Errorf("failed to delete old variations: %w", err)
	}

	// Добавляем новые
	if len(benchmark.Variations) > 0 {
		if err := db.AddVariations(benchmark.ID, benchmark.Variations); err != nil {
			return fmt.Errorf("failed to add variations: %w", err)
		}
	}

	return nil
}

// DeleteBenchmark удаляет эталон (мягкое удаление - устанавливает is_active = 0)
func (db *BenchmarksDB) DeleteBenchmark(id string) error {
	query := `UPDATE benchmarks SET is_active = 0, updated_at = ? WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete benchmark: %w", err)
	}
	return nil
}

// ListBenchmarks получает список эталонов с фильтрацией
func (db *BenchmarksDB) ListBenchmarks(entityType string, activeOnly bool, limit, offset int) ([]*Benchmark, error) {
	query := `
		SELECT id, entity_type, name, data, source_upload_id, source_client_id, is_active, created_at, updated_at
		FROM benchmarks
		WHERE 1=1
	`
	args := []interface{}{}

	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}

	if activeOnly {
		query += " AND is_active = 1"
	}

	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list benchmarks: %w", err)
	}
	defer rows.Close()

	var benchmarks []*Benchmark
	for rows.Next() {
		var benchmark Benchmark
		var dataJSON string
		var sourceUploadID, sourceClientID sql.NullInt64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&benchmark.ID,
			&benchmark.EntityType,
			&benchmark.Name,
			&dataJSON,
			&sourceUploadID,
			&sourceClientID,
			&benchmark.IsActive,
			&createdAt,
			&updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan benchmark: %w", err)
		}

		// Десериализуем JSON
		if err := json.Unmarshal([]byte(dataJSON), &benchmark.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal benchmark data: %w", err)
		}

		if sourceUploadID.Valid {
			id := int(sourceUploadID.Int64)
			benchmark.SourceUploadID = &id
		}
		if sourceClientID.Valid {
			id := int(sourceClientID.Int64)
			benchmark.SourceClientID = &id
		}

		benchmark.CreatedAt = createdAt
		benchmark.UpdatedAt = updatedAt

		benchmarks = append(benchmarks, &benchmark)
	}

	return benchmarks, nil
}

// FindBestMatch ищет лучший эталон для данного имени
func (db *BenchmarksDB) FindBestMatch(name string, entityType string) (*Benchmark, error) {
	// Сначала ищем точное совпадение по имени в таблице benchmarks
	query := `
		SELECT id, entity_type, name, data, source_upload_id, source_client_id, is_active, created_at, updated_at
		FROM benchmarks
		WHERE name = ? AND entity_type = ? AND is_active = 1
		LIMIT 1
	`

	var benchmark Benchmark
	var dataJSON string
	var sourceUploadID, sourceClientID sql.NullInt64
	var createdAt, updatedAt time.Time

	err := db.conn.QueryRow(query, name, entityType).Scan(
		&benchmark.ID,
		&benchmark.EntityType,
		&benchmark.Name,
		&dataJSON,
		&sourceUploadID,
		&sourceClientID,
		&benchmark.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err == nil {
		// Найдено точное совпадение
		if err := json.Unmarshal([]byte(dataJSON), &benchmark.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal benchmark data: %w", err)
		}

		if sourceUploadID.Valid {
			id := int(sourceUploadID.Int64)
			benchmark.SourceUploadID = &id
		}
		if sourceClientID.Valid {
			id := int(sourceClientID.Int64)
			benchmark.SourceClientID = &id
		}

		benchmark.CreatedAt = createdAt
		benchmark.UpdatedAt = updatedAt

		return &benchmark, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to search benchmark: %w", err)
	}

	// Если точного совпадения нет, ищем по вариациям
	query = `
		SELECT b.id, b.entity_type, b.name, b.data, b.source_upload_id, b.source_client_id, b.is_active, b.created_at, b.updated_at
		FROM benchmarks b
		INNER JOIN benchmark_variations bv ON b.id = bv.benchmark_id
		WHERE bv.variation = ? AND b.entity_type = ? AND b.is_active = 1
		LIMIT 1
	`

	err = db.conn.QueryRow(query, name, entityType).Scan(
		&benchmark.ID,
		&benchmark.EntityType,
		&benchmark.Name,
		&dataJSON,
		&sourceUploadID,
		&sourceClientID,
		&benchmark.IsActive,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Не найдено
		}
		return nil, fmt.Errorf("failed to search benchmark by variation: %w", err)
	}

	// Найдено по вариации
	if err := json.Unmarshal([]byte(dataJSON), &benchmark.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal benchmark data: %w", err)
	}

	if sourceUploadID.Valid {
		id := int(sourceUploadID.Int64)
		benchmark.SourceUploadID = &id
	}
	if sourceClientID.Valid {
		id := int(sourceClientID.Int64)
		benchmark.SourceClientID = &id
	}

	benchmark.CreatedAt = createdAt
	benchmark.UpdatedAt = updatedAt

	return &benchmark, nil
}

// AddVariations добавляет вариации названия для эталона
func (db *BenchmarksDB) AddVariations(benchmarkID string, variations []string) error {
	query := `INSERT INTO benchmark_variations (benchmark_id, variation) VALUES (?, ?)`

	for _, variation := range variations {
		if _, err := db.conn.Exec(query, benchmarkID, variation); err != nil {
			return fmt.Errorf("failed to add variation: %w", err)
		}
	}

	return nil
}

// GetVariations получает все вариации для эталона
func (db *BenchmarksDB) GetVariations(benchmarkID string) ([]string, error) {
	query := `SELECT variation FROM benchmark_variations WHERE benchmark_id = ?`

	rows, err := db.conn.Query(query, benchmarkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get variations: %w", err)
	}
	defer rows.Close()

	var variations []string
	for rows.Next() {
		var variation string
		if err := rows.Scan(&variation); err != nil {
			return nil, fmt.Errorf("failed to scan variation: %w", err)
		}
		variations = append(variations, variation)
	}

	return variations, nil
}

// CountBenchmarks возвращает количество эталонов
func (db *BenchmarksDB) CountBenchmarks(entityType string, activeOnly bool) (int, error) {
	query := `SELECT COUNT(*) FROM benchmarks WHERE 1=1`
	args := []interface{}{}

	if entityType != "" {
		query += " AND entity_type = ?"
		args = append(args, entityType)
	}

	if activeOnly {
		query += " AND is_active = 1"
	}

	var count int
	err := db.conn.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count benchmarks: %w", err)
	}

	return count, nil
}


