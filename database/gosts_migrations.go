package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// MigrateGostsSourceID добавляет поле source_id в таблицу gosts для связи с источниками данных
// Это миграция для существующих баз данных, которые были созданы до добавления source_id
func MigrateGostsSourceID(db *sql.DB) error {
	log.Println("Running migration: adding source_id field to gosts table...")

	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='gosts'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		// Таблица не существует, схема будет создана при инициализации
		log.Println("gosts table does not exist, skipping migration")
		return nil
	}

	// Проверяем, существует ли уже поле source_id
	var columnExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('gosts')
			WHERE name='source_id'
		)
	`).Scan(&columnExists)
	if err != nil {
		// Если не удалось проверить, пробуем добавить (SQLite может не поддерживать pragma_table_info в некоторых версиях)
		columnExists = false
	}

	if columnExists {
		log.Println("source_id column already exists, skipping migration")
		return nil
	}

	// Добавляем поле source_id
	migrations := []string{
		`ALTER TABLE gosts ADD COLUMN source_id INTEGER`,
		`CREATE INDEX IF NOT EXISTS idx_gosts_source_id ON gosts(source_id)`,
	}

	successCount := 0
	skipCount := 0

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки о существующих колонках/индексах
			if strings.Contains(errStr, "duplicate column") ||
				strings.Contains(errStr, "already exists") ||
				strings.Contains(errStr, "duplicate index") {
				skipCount++
				continue
			}
			return fmt.Errorf("migration failed: %s, error: %w", migration, err)
		}
		successCount++
	}

	log.Printf("GOSTs migration completed: %d operations successful, %d skipped", successCount, skipCount)

	// Добавляем поля created_at и updated_at в таблицу gost_sources, если их нет
	if err := migrateGostSourcesTimestamps(db); err != nil {
		log.Printf("Warning: failed to migrate gost_sources timestamps: %v", err)
	}

	return nil
}

// migrateGostSourcesTimestamps добавляет поля created_at и updated_at в таблицу gost_sources
func migrateGostSourcesTimestamps(db *sql.DB) error {
	// Проверяем существование колонки updated_at
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('gost_sources')
			WHERE name='updated_at'
		)
	`).Scan(&columnExists)
	if err != nil {
		// Если не удалось проверить, пробуем добавить
		columnExists = false
	}

	if !columnExists {
		migrations := []string{
			`ALTER TABLE gost_sources ADD COLUMN created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
			`ALTER TABLE gost_sources ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`,
		}

		for _, migration := range migrations {
			_, err := db.Exec(migration)
			if err != nil {
				errStr := strings.ToLower(err.Error())
				if !strings.Contains(errStr, "duplicate column") &&
					!strings.Contains(errStr, "already exists") {
					return fmt.Errorf("migration failed: %s, error: %w", migration, err)
				}
			}
		}
		log.Println("Added created_at and updated_at columns to gost_sources table")
	}

	return nil
}

// MigrateGostsSchema выполняет все миграции для таблиц ГОСТов
func MigrateGostsSchema(db *sql.DB) error {
	// Выполняем миграцию для source_id
	if err := MigrateGostsSourceID(db); err != nil {
		return fmt.Errorf("failed to migrate gosts source_id: %w", err)
	}

	return nil
}

