package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// InitBenchmarksSchema создает все необходимые таблицы в БД эталонов
func InitBenchmarksSchema(db *sql.DB) error {
	schema := `
	-- Таблица эталонных записей
	CREATE TABLE IF NOT EXISTS benchmarks (
		id TEXT PRIMARY KEY,
		entity_type TEXT NOT NULL, -- 'counterparty', 'nomenclature'
		name TEXT NOT NULL,
		data TEXT NOT NULL,        -- JSON с полями: inn, address, article, brand и т.д.
		source_upload_id INTEGER,
		source_client_id INTEGER,
		is_active BOOLEAN NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	-- Таблица вариаций названий для эталонов
	CREATE TABLE IF NOT EXISTS benchmark_variations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		benchmark_id TEXT NOT NULL,
		variation TEXT NOT NULL,
		FOREIGN KEY (benchmark_id) REFERENCES benchmarks(id) ON DELETE CASCADE
	);

	-- Индексы для оптимизации запросов
	CREATE INDEX IF NOT EXISTS idx_benchmarks_entity_type ON benchmarks(entity_type);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_name ON benchmarks(name);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_is_active ON benchmarks(is_active);
	CREATE INDEX IF NOT EXISTS idx_benchmarks_source_upload_id ON benchmarks(source_upload_id);
	CREATE INDEX IF NOT EXISTS idx_benchmark_variations_benchmark_id ON benchmark_variations(benchmark_id);
	CREATE INDEX IF NOT EXISTS idx_benchmark_variations_variation ON benchmark_variations(variation);
	`

	_, err := db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create benchmarks schema: %w", err)
	}

	// Включаем поддержку FOREIGN KEY constraints в SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	return nil
}

// CreateBenchmarksDatabase создает или открывает БД эталонов
func CreateBenchmarksDatabase(path string) (*sql.DB, error) {
	// Создаем директорию, если её нет
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create benchmarks database directory: %w", err)
	}

	// Открываем БД
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open benchmarks database: %w", err)
	}

	// Настройка connection pooling
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем подключение
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping benchmarks database: %w", err)
	}

	// Инициализируем схему
	if err := InitBenchmarksSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize benchmarks schema: %w", err)
	}

	return db, nil
}

