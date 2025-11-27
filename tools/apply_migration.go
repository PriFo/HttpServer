//go:build tool_apply_migration
// +build tool_apply_migration

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	fmt.Println("Применение миграции database_table_metadata...")

	db, err := sql.Open("sqlite3", "data/service.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Создаем таблицу
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS database_table_metadata (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		database_id INTEGER NOT NULL,
		table_name TEXT NOT NULL,
		entity_type TEXT NOT NULL DEFAULT 'counterparty',
		column_mappings TEXT NOT NULL,
		detection_confidence REAL NOT NULL DEFAULT 0.0,
		last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(database_id, table_name, entity_type),
		FOREIGN KEY (database_id) REFERENCES project_databases(id) ON DELETE CASCADE
	);
	`

	if _, err := db.Exec(createTableSQL); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	fmt.Println("✅ Таблица создана")

	// Создаем индексы
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_database_table_metadata_database_id ON database_table_metadata(database_id);`,
		`CREATE INDEX IF NOT EXISTS idx_database_table_metadata_entity_type ON database_table_metadata(entity_type);`,
		`CREATE INDEX IF NOT EXISTS idx_database_table_metadata_confidence ON database_table_metadata(detection_confidence);`,
	}

	for i, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			log.Fatalf("Failed to create index %d: %v", i+1, err)
		}
		fmt.Printf("✅ Индекс %d создан\n", i+1)
	}

	fmt.Println("\n✅ Миграция успешно применена!")
}

