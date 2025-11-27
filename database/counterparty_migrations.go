package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// MigrateCounterpartyEnrichmentSource добавляет поле source_enrichment для хранения источника нормализации
func MigrateCounterpartyEnrichmentSource(db *sql.DB) error {
	// Проверяем существование таблицы normalized_counterparties перед миграцией
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='normalized_counterparties'
		)
	`).Scan(&tableExists)
	if err != nil {
		// Если не удалось проверить, продолжаем (возможно, это не критично)
		tableExists = false
	}

	// Выполняем миграцию только если таблица существует
	if !tableExists {
		// Таблица не существует, пропускаем миграцию
		return nil
	}

	migrations := []string{
		`ALTER TABLE normalized_counterparties ADD COLUMN source_enrichment TEXT DEFAULT ''`,
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_source_enrichment ON normalized_counterparties(source_enrichment)`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если поле уже существует
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "duplicate index") {
				return fmt.Errorf("migration failed: %s, error: %w", migration, err)
			}
		}
	}

	return nil
}

// MigrateBenchmarkCounterpartyFields добавляет поля для контрагентов в таблицу client_benchmarks
// если их нет (для старых баз данных)
func MigrateBenchmarkCounterpartyFields(db *sql.DB) error {
	migrations := []string{
		`ALTER TABLE client_benchmarks ADD COLUMN tax_id TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN kpp TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN legal_address TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN postal_address TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN contact_phone TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN contact_email TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN contact_person TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN legal_form TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN bank_name TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN bank_account TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN correspondent_account TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN bik TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_client_benchmarks_tax_id ON client_benchmarks(tax_id)`,
		`CREATE INDEX IF NOT EXISTS idx_client_benchmarks_kpp ON client_benchmarks(kpp)`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если поле уже существует
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "duplicate index") {
				return fmt.Errorf("migration failed: %s, error: %w", migration, err)
			}
		}
	}

	return nil
}

// MigrateBenchmarkOGRNRegion добавляет поля ogrn и region в таблицу client_benchmarks
func MigrateBenchmarkOGRNRegion(db *sql.DB) error {
	migrations := []string{
		`ALTER TABLE client_benchmarks ADD COLUMN ogrn TEXT`,
		`ALTER TABLE client_benchmarks ADD COLUMN region TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_client_benchmarks_ogrn ON client_benchmarks(ogrn)`,
		`CREATE INDEX IF NOT EXISTS idx_client_benchmarks_region ON client_benchmarks(region)`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если поле уже существует
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "duplicate index") {
				return fmt.Errorf("migration failed: %s, error: %w", migration, err)
			}
		}
	}

	return nil
}

// CreateNormalizedCounterpartiesTable создает таблицу normalized_counterparties если её нет
func CreateNormalizedCounterpartiesTable(db *sql.DB) error {
	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='normalized_counterparties'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if tableExists {
		// Таблица уже существует, пропускаем создание
		return nil
	}

	// Создаем таблицу
	// ВАЖНО: Включаем все поля сразу, чтобы не требовались миграции для новых таблиц
	createTable := `
		CREATE TABLE IF NOT EXISTS normalized_counterparties (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			client_project_id INTEGER NOT NULL,
			source_reference TEXT,
			source_name TEXT,
			normalized_name TEXT NOT NULL,
			tax_id TEXT,
			kpp TEXT,
			bin TEXT,
			legal_address TEXT,
			postal_address TEXT,
			contact_phone TEXT,
			contact_email TEXT,
			contact_person TEXT,
			legal_form TEXT,
			bank_name TEXT,
			bank_account TEXT,
			correspondent_account TEXT,
			bik TEXT,
			benchmark_id INTEGER,
			quality_score REAL DEFAULT 0.0,
			enrichment_applied BOOLEAN DEFAULT FALSE,
			source_enrichment TEXT DEFAULT '',
			source_database TEXT,
			subcategory TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(client_project_id) REFERENCES client_projects(id) ON DELETE CASCADE,
			FOREIGN KEY(benchmark_id) REFERENCES client_benchmarks(id) ON DELETE SET NULL,
			UNIQUE(client_project_id, source_reference)
		)
	`

	_, err = db.Exec(createTable)
	if err != nil {
		return fmt.Errorf("failed to create normalized_counterparties table: %w", err)
	}

	// Создаем индексы (с повторной проверкой существования таблицы для защиты от race condition)
	// Проверяем, что таблица все еще существует после создания
	var tableStillExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='normalized_counterparties'
		)
	`).Scan(&tableStillExists)
	if err != nil || !tableStillExists {
		// Таблица не была создана или была удалена - это может быть race condition
		// Пробуем еще раз проверить существование
		err = db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM sqlite_master
				WHERE type='table' AND name='normalized_counterparties'
			)
		`).Scan(&tableStillExists)
		if err != nil || !tableStillExists {
			return fmt.Errorf("table normalized_counterparties was not created successfully")
		}
	}

	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_project_id ON normalized_counterparties(client_project_id)`,
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_tax_id ON normalized_counterparties(tax_id)`,
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_benchmark_id ON normalized_counterparties(benchmark_id)`,
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_subcategory ON normalized_counterparties(subcategory)`,
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_source_enrichment ON normalized_counterparties(source_enrichment)`,
	}

	for _, indexSQL := range indexes {
		_, err = db.Exec(indexSQL)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если индекс уже существует или таблица не существует (может быть race condition)
			if !strings.Contains(errStr, "duplicate index") && 
			   !strings.Contains(errStr, "already exists") &&
			   !strings.Contains(errStr, "no such table") {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}
	}

	return nil
}

// MigrateBenchmarkManufacturerLink добавляет поле manufacturer_benchmark_id для связи номенклатур с производителями
func MigrateBenchmarkManufacturerLink(db *sql.DB) error {
	migrations := []string{
		`ALTER TABLE client_benchmarks ADD COLUMN manufacturer_benchmark_id INTEGER`,
		`CREATE INDEX IF NOT EXISTS idx_client_benchmarks_manufacturer_id ON client_benchmarks(manufacturer_benchmark_id)`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если поле уже существует
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "duplicate index") {
				return fmt.Errorf("migration failed: %s, error: %w", migration, err)
			}
		}
	}

	return nil
}

// MigrateNormalizedCounterpartiesSubcategory добавляет поле subcategory в таблицу normalized_counterparties
func MigrateNormalizedCounterpartiesSubcategory(db *sql.DB) error {
	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='normalized_counterparties'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		// Таблица не существует, создаем её с полем subcategory
		return CreateNormalizedCounterpartiesTable(db)
	}

	// Добавляем поле subcategory если его нет
	migrations := []string{
		`ALTER TABLE normalized_counterparties ADD COLUMN subcategory TEXT`,
		`CREATE INDEX IF NOT EXISTS idx_normalized_counterparties_subcategory ON normalized_counterparties(subcategory)`,
	}

	for _, migration := range migrations {
		_, err := db.Exec(migration)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если поле уже существует
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "duplicate index") {
				return fmt.Errorf("migration failed: %s, error: %w", migration, err)
			}
		}
	}

	return nil
}

// CreateCounterpartyDatabasesTable создает таблицу counterparty_databases для связи many-to-many
// между нормализованными контрагентами и базами данных проекта
func CreateCounterpartyDatabasesTable(db *sql.DB) error {
	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='counterparty_databases'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if tableExists {
		// Таблица уже существует, пропускаем создание
		return nil
	}

	// Создаем таблицу
	createTable := `
		CREATE TABLE IF NOT EXISTS counterparty_databases (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			normalized_counterparty_id INTEGER NOT NULL,
			project_database_id INTEGER NOT NULL,
			source_reference TEXT,
			source_name TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(normalized_counterparty_id) REFERENCES normalized_counterparties(id) ON DELETE CASCADE,
			FOREIGN KEY(project_database_id) REFERENCES project_databases(id) ON DELETE CASCADE,
			UNIQUE(normalized_counterparty_id, project_database_id, source_reference)
		)
	`

	_, err = db.Exec(createTable)
	if err != nil {
		return fmt.Errorf("failed to create counterparty_databases table: %w", err)
	}

	// Проверяем, что таблица была создана
	var tableStillExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='counterparty_databases'
		)
	`).Scan(&tableStillExists)
	if err != nil || !tableStillExists {
		return fmt.Errorf("table counterparty_databases was not created successfully")
	}

	// Создаем индексы
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_counterparty_databases_counterparty_id ON counterparty_databases(normalized_counterparty_id)`,
		`CREATE INDEX IF NOT EXISTS idx_counterparty_databases_database_id ON counterparty_databases(project_database_id)`,
		`CREATE INDEX IF NOT EXISTS idx_counterparty_databases_source_reference ON counterparty_databases(source_reference)`,
		`CREATE INDEX IF NOT EXISTS idx_counterparty_databases_counterparty_database ON counterparty_databases(normalized_counterparty_id, project_database_id)`,
	}

	for _, indexSQL := range indexes {
		_, err = db.Exec(indexSQL)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			// Игнорируем ошибки, если индекс уже существует
			if !strings.Contains(errStr, "duplicate index") &&
				!strings.Contains(errStr, "already exists") &&
				!strings.Contains(errStr, "no such table") {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}
	}

	return nil
}

// MigrateCounterpartyDatabases заполняет таблицу counterparty_databases из существующих данных
// Извлекает информацию о базах данных из поля source_database в normalized_counterparties
func MigrateCounterpartyDatabases(db *sql.DB) error {
	// Проверяем существование таблицы counterparty_databases
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='counterparty_databases'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		// Таблица не существует, создаем её
		if err := CreateCounterpartyDatabasesTable(db); err != nil {
			return fmt.Errorf("failed to create counterparty_databases table: %w", err)
		}
	}

	// Проверяем существование таблицы normalized_counterparties
	var normalizedTableExists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='normalized_counterparties'
		)
	`).Scan(&normalizedTableExists)
	if err != nil || !normalizedTableExists {
		// Таблица normalized_counterparties не существует, нечего мигрировать
		return nil
	}

	// Получаем все нормализованные контрагенты с source_database
	rows, err := db.Query(`
		SELECT id, client_project_id, source_reference, source_name, source_database
		FROM normalized_counterparties
		WHERE source_database IS NOT NULL AND source_database != ''
	`)
	if err != nil {
		return fmt.Errorf("failed to query normalized counterparties: %w", err)
	}
	defer rows.Close()

	migratedCount := 0
	skippedCount := 0
	errorCount := 0

	for rows.Next() {
		var counterpartyID, projectID int
		var sourceReference, sourceName, sourceDatabase sql.NullString

		err := rows.Scan(&counterpartyID, &projectID, &sourceReference, &sourceName, &sourceDatabase)
		if err != nil {
			errorCount++
			continue
		}

		if !sourceDatabase.Valid || sourceDatabase.String == "" {
			skippedCount++
			continue
		}

		// Парсим source_database - может быть одно имя или несколько через разделитель
		dbNames := ParseDatabaseNames(sourceDatabase.String)

		// Для каждого имени базы данных ищем соответствующую запись в project_databases
		for _, dbName := range dbNames {
			if dbName == "" {
				continue
			}

			// Ищем базу данных по имени в рамках проекта
			var databaseID int
			err := db.QueryRow(`
				SELECT id FROM project_databases
				WHERE client_project_id = ? AND name = ?
				LIMIT 1
			`, projectID, dbName).Scan(&databaseID)

			if err == sql.ErrNoRows {
				// База данных не найдена, пробуем найти по file_path
				err = db.QueryRow(`
					SELECT id FROM project_databases
					WHERE client_project_id = ? AND (file_path LIKE ? OR file_path LIKE ?)
					LIMIT 1
				`, projectID, "%"+dbName+"%", dbName).Scan(&databaseID)
			}

			if err != nil {
				// База данных не найдена, пропускаем
				skippedCount++
				continue
			}

			// Создаем связь в counterparty_databases
			sourceRef := ""
			if sourceReference.Valid {
				sourceRef = sourceReference.String
			}
			sourceNameStr := ""
			if sourceName.Valid {
				sourceNameStr = sourceName.String
			}

			_, err = db.Exec(`
				INSERT OR IGNORE INTO counterparty_databases
				(normalized_counterparty_id, project_database_id, source_reference, source_name)
				VALUES (?, ?, ?, ?)
			`, counterpartyID, databaseID, sourceRef, sourceNameStr)

			if err != nil {
				errorCount++
				continue
			}

			migratedCount++
		}
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating normalized counterparties: %w", err)
	}

	// Логируем результаты миграции
	if migratedCount > 0 || skippedCount > 0 || errorCount > 0 {
		fmt.Printf("Counterparty databases migration: migrated=%d, skipped=%d, errors=%d\n", migratedCount, skippedCount, errorCount)
	}

	return nil
}
