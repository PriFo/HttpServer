package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

const migrationsTableName = "schema_migrations"

// ensureMigrationTable создает таблицу schema_migrations при необходимости.
func ensureMigrationTable(db *sql.DB) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			name TEXT PRIMARY KEY,
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`, migrationsTableName)

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to ensure schema_migrations table: %w", err)
	}
	return nil
}

// isMigrationApplied проверяет, была ли уже применена миграция.
func isMigrationApplied(db *sql.DB, name string) (bool, error) {
	if err := ensureMigrationTable(db); err != nil {
		return false, err
	}

	var appliedAt sql.NullTime
	query := fmt.Sprintf(`SELECT applied_at FROM %s WHERE name = ?`, migrationsTableName)
	err := db.QueryRow(query, name).Scan(&appliedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to check migration %s: %w", name, err)
	}

	return appliedAt.Valid, nil
}

// markMigrationApplied сохраняет информацию о примененной миграции.
func markMigrationApplied(db *sql.DB, name string) error {
	if err := ensureMigrationTable(db); err != nil {
		return err
	}

	query := fmt.Sprintf(`INSERT OR REPLACE INTO %s(name, applied_at) VALUES(?, ?)`, migrationsTableName)
	_, err := db.Exec(query, name, time.Now())
	if err != nil {
		return fmt.Errorf("failed to mark migration %s as applied: %w", name, err)
	}
	return nil
}

// ensureMigrationApplied выполняет миграцию только один раз.
func ensureMigrationApplied(db *sql.DB, name string, migration func(*sql.DB) error) error {
	applied, err := isMigrationApplied(db, name)
	if err != nil {
		return err
	}
	if applied {
		log.Printf("[Migrations] Skipping %s - already applied", name)
		return nil
	}

	if err := migration(db); err != nil {
		return err
	}

	if err := markMigrationApplied(db, name); err != nil {
		return err
	}

	log.Printf("[Migrations] %s applied successfully", name)
	return nil
}
