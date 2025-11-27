package database

import (
	"database/sql"
	"log"
	"strings"
)

// MigrateClientEnhancements добавляет расширенные поля для клиентов и таблицу документов
func MigrateClientEnhancements(db *sql.DB) error {
	log.Println("Starting client enhancements migration...")

	migrations := []string{
		// Бизнес-информация
		`ALTER TABLE clients ADD COLUMN industry TEXT`,
		`ALTER TABLE clients ADD COLUMN company_size TEXT`,
		`ALTER TABLE clients ADD COLUMN legal_form TEXT`,

		// Расширенные контакты
		`ALTER TABLE clients ADD COLUMN contact_person TEXT`,
		`ALTER TABLE clients ADD COLUMN contact_position TEXT`,
		`ALTER TABLE clients ADD COLUMN alternate_phone TEXT`,
		`ALTER TABLE clients ADD COLUMN website TEXT`,

		// Юридические данные
		`ALTER TABLE clients ADD COLUMN ogrn TEXT`,
		`ALTER TABLE clients ADD COLUMN kpp TEXT`,
		`ALTER TABLE clients ADD COLUMN legal_address TEXT`,
		`ALTER TABLE clients ADD COLUMN postal_address TEXT`,
		`ALTER TABLE clients ADD COLUMN bank_name TEXT`,
		`ALTER TABLE clients ADD COLUMN bank_account TEXT`,
		`ALTER TABLE clients ADD COLUMN correspondent_account TEXT`,
		`ALTER TABLE clients ADD COLUMN bik TEXT`,

		// Договорные данные
		`ALTER TABLE clients ADD COLUMN contract_number TEXT`,
		`ALTER TABLE clients ADD COLUMN contract_date TIMESTAMP`,
		`ALTER TABLE clients ADD COLUMN contract_terms TEXT`,
		`ALTER TABLE clients ADD COLUMN contract_expires_at TIMESTAMP`,
	}

	successCount := 0
	skipCount := 0

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки о существующих колонках
			if strings.Contains(errStr, "duplicate column") ||
				strings.Contains(errStr, "already exists") {
				skipCount++
				continue
			}
			return err
		}
		successCount++
	}

	log.Printf("Client enhancements migration completed: %d operations successful, %d skipped", successCount, skipCount)

	// Создаем таблицу документов клиента
	createDocumentsTableSQL := `
		CREATE TABLE IF NOT EXISTS client_documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_id INTEGER NOT NULL,
			file_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			file_type TEXT NOT NULL,
			file_size INTEGER NOT NULL,
			category TEXT DEFAULT 'technical',
			description TEXT,
			uploaded_by TEXT,
			uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(client_id) REFERENCES clients(id) ON DELETE CASCADE
		);
	`

	if _, err := db.Exec(createDocumentsTableSQL); err != nil {
		errStr := strings.ToLower(err.Error())
		if !strings.Contains(errStr, "already exists") {
			return err
		}
	}

	// Создаем индексы для таблицы документов
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_client_documents_client_id ON client_documents(client_id)`,
		`CREATE INDEX IF NOT EXISTS idx_client_documents_category ON client_documents(category)`,
		`CREATE INDEX IF NOT EXISTS idx_client_documents_uploaded_at ON client_documents(uploaded_at DESC)`,
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "duplicate index") &&
				!strings.Contains(errStr, "already exists") {
				log.Printf("Warning: failed to create index: %v", err)
			}
		}
	}

	log.Println("✅ Client enhancements migration completed successfully")
	return nil
}

