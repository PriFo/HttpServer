package database

import (
	"database/sql"
	"log"
	"strings"
)

// MigrateAppConfigVersioning добавляет версионирование в таблицу app_config
func MigrateAppConfigVersioning(db *sql.DB) error {
	migrations := []string{
		// Добавляем поле version в app_config, если его нет
		`ALTER TABLE app_config ADD COLUMN version INTEGER DEFAULT 1`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если поле уже существует
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") {
				// Не возвращаем ошибку, если это просто дубликат колонки
				continue
			}
		} else {
			// Если миграция прошла успешно, устанавливаем версию для существующих записей
			if strings.Contains(migration, "version") {
				_, updateErr := db.Exec(`UPDATE app_config SET version = 1 WHERE version IS NULL`)
				if updateErr != nil {
					log.Printf("Warning: failed to set default version: %v", updateErr)
				}
			}
		}
	}

	return nil
}

