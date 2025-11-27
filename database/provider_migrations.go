package database

import (
	"database/sql"
	"fmt"
	"strings"
)

// CreateProvidersTable создает таблицу providers если её нет
func CreateProvidersTable(db *sql.DB) error {
	// Проверяем существование таблицы
	var tableExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='providers'
		)
	`).Scan(&tableExists)
	if err != nil {
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		// Создаем таблицу providers с новой структурой
		createTable := `
			CREATE TABLE providers (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				type TEXT NOT NULL,
				config TEXT,
				is_active BOOLEAN NOT NULL DEFAULT 1,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`

		_, err = db.Exec(createTable)
		if err != nil {
			return fmt.Errorf("failed to create providers table: %w", err)
		}

		// Создаем индексы
		indexes := []string{
			`CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(type)`,
			`CREATE INDEX IF NOT EXISTS idx_providers_is_active ON providers(is_active)`,
			`CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name)`,
		}

		for _, indexSQL := range indexes {
			if _, err := db.Exec(indexSQL); err != nil {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}

		// Вставляем дефолтные провайдеры
		defaultProviders := []struct {
			Name     string
			Type     string
			IsActive bool
		}{
			{"OpenRouter", "openrouter", false},
			{"Hugging Face", "huggingface", false},
			{"Arliai", "arliai", false},
			{"Eden AI", "edenai", false},
		}

		for _, p := range defaultProviders {
			_, err = db.Exec(`
				INSERT OR IGNORE INTO providers (name, type, is_active)
				VALUES (?, ?, ?)
			`, p.Name, p.Type, p.IsActive)
			if err != nil {
				return fmt.Errorf("failed to insert default provider %s: %w", p.Name, err)
			}
		}
	} else {
		// Таблица существует - выполняем миграцию для обновления структуры
		if err := migrateProvidersTable(db); err != nil {
			return fmt.Errorf("failed to migrate providers table: %w", err)
		}
	}

	return nil
}

// migrateProvidersTable мигрирует существующую таблицу providers на новую структуру
func migrateProvidersTable(db *sql.DB) error {
	// Проверяем структуру таблицы, чтобы определить, нужна ли миграция
	var hasTypeColumn bool
	var hasConfigColumn bool
	var hasIsActiveColumn bool
	var hasIntegerID bool

	// Проверяем наличие колонки type
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('providers')
			WHERE name='type'
		)
	`).Scan(&hasTypeColumn)
	if err != nil {
		return fmt.Errorf("failed to check type column: %w", err)
	}

	// Проверяем наличие колонки config
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('providers')
			WHERE name='config'
		)
	`).Scan(&hasConfigColumn)
	if err != nil {
		return fmt.Errorf("failed to check config column: %w", err)
	}

	// Проверяем наличие колонки is_active
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('providers')
			WHERE name='is_active'
		)
	`).Scan(&hasIsActiveColumn)
	if err != nil {
		return fmt.Errorf("failed to check is_active column: %w", err)
	}

	// Проверяем тип колонки id
	var idType string
	err = db.QueryRow(`
		SELECT type FROM pragma_table_info('providers')
		WHERE name='id'
	`).Scan(&idType)
	if err == nil {
		hasIntegerID = strings.ToUpper(idType) == "INTEGER"
	}

	// Если таблица уже имеет новую структуру, ничего не делаем
	if hasTypeColumn && hasConfigColumn && hasIsActiveColumn && hasIntegerID {
		return nil
	}

	// Если таблица имеет старую структуру (id TEXT), нужно пересоздать таблицу
	if !hasIntegerID {
		// Создаем временную таблицу с новой структурой
		_, err = db.Exec(`
			CREATE TABLE providers_new (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name TEXT NOT NULL UNIQUE,
				type TEXT NOT NULL,
				config TEXT,
				is_active BOOLEAN NOT NULL DEFAULT 1,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
			)
		`)
		if err != nil {
			return fmt.Errorf("failed to create new providers table: %w", err)
		}

		// Копируем данные из старой таблицы в новую
		// Маппинг старых полей на новые
		_, err = db.Exec(`
			INSERT INTO providers_new (name, type, config, is_active, created_at, updated_at)
			SELECT 
				name,
				COALESCE(id, LOWER(REPLACE(name, ' ', '_'))) as type,
				CASE 
					WHEN api_key IS NOT NULL OR base_url IS NOT NULL THEN
						json_object('api_key', COALESCE(api_key, ''), 'base_url', COALESCE(base_url, ''))
					ELSE NULL
				END as config,
				COALESCE(enabled, 1) as is_active,
				COALESCE(created_at, CURRENT_TIMESTAMP) as created_at,
				COALESCE(updated_at, CURRENT_TIMESTAMP) as updated_at
			FROM providers
		`)
		if err != nil {
			// Если не удалось скопировать данные, удаляем новую таблицу
			db.Exec(`DROP TABLE IF EXISTS providers_new`)
			return fmt.Errorf("failed to migrate data: %w", err)
		}

		// Удаляем старую таблицу
		_, err = db.Exec(`DROP TABLE providers`)
		if err != nil {
			return fmt.Errorf("failed to drop old providers table: %w", err)
		}

		// Переименовываем новую таблицу
		_, err = db.Exec(`ALTER TABLE providers_new RENAME TO providers`)
		if err != nil {
			return fmt.Errorf("failed to rename providers table: %w", err)
		}
	} else {
		// Таблица уже имеет INTEGER id, просто добавляем недостающие колонки
		migrations := []string{}

		if !hasTypeColumn {
			migrations = append(migrations, `ALTER TABLE providers ADD COLUMN type TEXT NOT NULL DEFAULT 'unknown'`)
		}

		if !hasConfigColumn {
			migrations = append(migrations, `ALTER TABLE providers ADD COLUMN config TEXT`)
		}

		if !hasIsActiveColumn {
			// Если есть enabled, мигрируем его в is_active
			var hasEnabledColumn bool
			err = db.QueryRow(`
				SELECT EXISTS (
					SELECT 1 FROM pragma_table_info('providers')
					WHERE name='enabled'
				)
			`).Scan(&hasEnabledColumn)
			if err == nil && hasEnabledColumn {
				migrations = append(migrations, `ALTER TABLE providers ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT 1`)
				// Копируем значения из enabled в is_active
				migrations = append(migrations, `UPDATE providers SET is_active = enabled WHERE is_active IS NULL`)
			} else {
				migrations = append(migrations, `ALTER TABLE providers ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT 1`)
			}
		}

		// Выполняем миграции
		for _, migration := range migrations {
			_, err = db.Exec(migration)
			if err != nil {
				errStr := strings.ToLower(err.Error())
				// Игнорируем ошибки о существующих колонках
				if !strings.Contains(errStr, "duplicate column") &&
					!strings.Contains(errStr, "already exists") {
					return fmt.Errorf("failed to execute migration: %s, error: %w", migration, err)
				}
			}
		}

		// Устанавливаем type для существующих записей, если он не установлен
		_, err = db.Exec(`
			UPDATE providers 
			SET type = LOWER(REPLACE(name, ' ', '_'))
			WHERE type IS NULL OR type = '' OR type = 'unknown'
		`)
		if err != nil {
			// Игнорируем ошибки, если колонка type еще не существует
		}
	}

	// Создаем индексы, если их нет
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_providers_type ON providers(type)`,
		`CREATE INDEX IF NOT EXISTS idx_providers_is_active ON providers(is_active)`,
		`CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name)`,
	}

	for _, indexSQL := range indexes {
		if _, err := db.Exec(indexSQL); err != nil {
			// Игнорируем ошибки о существующих индексах
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "duplicate index") &&
				!strings.Contains(errStr, "already exists") {
				return fmt.Errorf("failed to create index: %w", err)
			}
		}
	}

	return nil
}

// addChannelsColumnIfNotExists добавляет колонку channels если её нет (legacy, для обратной совместимости)
func addChannelsColumnIfNotExists(db *sql.DB) error {
	// Проверяем существование колонки
	var columnExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM pragma_table_info('providers')
			WHERE name='channels'
		)
	`).Scan(&columnExists)
	if err != nil {
		return fmt.Errorf("failed to check column existence: %w", err)
	}

	if !columnExists {
		// Добавляем колонку channels
		_, err = db.Exec(`ALTER TABLE providers ADD COLUMN channels INTEGER DEFAULT 1`)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			if !strings.Contains(errStr, "duplicate column") &&
				!strings.Contains(errStr, "already exists") {
				return fmt.Errorf("failed to add channels column: %w", err)
			}
		}

		// Устанавливаем значения каналов для существующих провайдеров
		updates := []struct {
			Name     string
			Channels int
		}{
			{"OpenRouter", 1},
			{"openrouter", 1},
			{"Hugging Face", 1},
			{"huggingface", 1},
			{"Arliai", 2},
			{"arliai", 2},
		}

		for _, update := range updates {
			_, err = db.Exec(`
				UPDATE providers 
				SET channels = ?, updated_at = CURRENT_TIMESTAMP
				WHERE LOWER(name) = LOWER(?)
			`, update.Channels, update.Name)
			if err != nil {
				// Игнорируем ошибки, если провайдер не найден
				continue
			}
		}
	}

	return nil
}

// Provider структура провайдера из БД
type Provider struct {
	ID        int
	Name      string
	Type      string
	Config    string // JSON с конфигурацией (API ключи, URL и т.д.)
	IsActive  bool
	CreatedAt string
	UpdatedAt string
}

// GetProviders получает список всех провайдеров из БД
func (db *ServiceDB) GetProviders() ([]*Provider, error) {
	query := `
		SELECT id, name, type, config, is_active, created_at, updated_at
		FROM providers
		ORDER BY name ASC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get providers: %w", err)
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		var config sql.NullString
		var createdAt sql.NullString
		var updatedAt sql.NullString
		err := rows.Scan(
			&p.ID, &p.Name, &p.Type, &config,
			&p.IsActive, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		if config.Valid {
			p.Config = config.String
		}
		if createdAt.Valid {
			p.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			p.UpdatedAt = updatedAt.String
		}
		providers = append(providers, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating providers: %w", err)
	}

	return providers, nil
}

// GetActiveProviders получает только активные провайдеры
func (db *ServiceDB) GetActiveProviders() ([]*Provider, error) {
	query := `
		SELECT id, name, type, config, is_active, created_at, updated_at
		FROM providers
		WHERE is_active = 1
		ORDER BY name ASC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get active providers: %w", err)
	}
	defer rows.Close()

	var providers []*Provider
	for rows.Next() {
		p := &Provider{}
		var config sql.NullString
		var createdAt sql.NullString
		var updatedAt sql.NullString
		err := rows.Scan(
			&p.ID, &p.Name, &p.Type, &config,
			&p.IsActive, &createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan provider: %w", err)
		}
		if config.Valid {
			p.Config = config.String
		}
		if createdAt.Valid {
			p.CreatedAt = createdAt.String
		}
		if updatedAt.Valid {
			p.UpdatedAt = updatedAt.String
		}
		providers = append(providers, p)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating providers: %w", err)
	}

	return providers, nil
}

// UpdateProviderChannels обновляет количество каналов для провайдера (legacy метод для обратной совместимости)
// В новой структуре каналы хранятся в config JSON
func (db *ServiceDB) UpdateProviderChannels(providerID int, channels int) error {
	// Получаем текущий config
	var config sql.NullString
	err := db.conn.QueryRow(`
		SELECT config FROM providers WHERE id = ?
	`, providerID).Scan(&config)
	if err != nil {
		return fmt.Errorf("failed to get provider config: %w", err)
	}

	// Обновляем config с новым значением channels
	// Это упрощенная версия - в реальности нужно парсить и обновлять JSON
	query := `
		UPDATE providers
		SET config = json_set(COALESCE(config, '{}'), '$.channels', ?),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	result, err := db.conn.Exec(query, channels, providerID)
	if err != nil {
		return fmt.Errorf("failed to update provider channels: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("provider %d not found", providerID)
	}

	return nil
}

// AddDataStandardizationProviders добавляет провайдеров DaData и Adata для стандартизации данных
func AddDataStandardizationProviders(db *sql.DB) error {
	// Провайдеры для стандартизации контрагентов
	standardizationProviders := []struct {
		Name     string
		Type     string
		BaseURL  string
		IsActive bool
	}{
		{"DaData.ru", "dadata", "https://suggestions.dadata.ru/suggestions/api/4_1/rs", false},
		{"Adata.kz", "adata", "https://api.adata.kz", false},
	}

	for _, p := range standardizationProviders {
		// Проверяем, существует ли провайдер по типу
		var exists bool
		err := db.QueryRow(`
			SELECT EXISTS (
				SELECT 1 FROM providers WHERE type = ?
			)
		`, p.Type).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check provider existence %s: %w", p.Type, err)
		}

		configJSON := fmt.Sprintf(`{"base_url": "%s"}`, p.BaseURL)

		if !exists {
			// Вставляем нового провайдера (по умолчанию disabled, пока не настроен API ключ)
			_, err = db.Exec(`
				INSERT INTO providers (name, type, config, is_active)
				VALUES (?, ?, ?, ?)
			`, p.Name, p.Type, configJSON, p.IsActive)
			if err != nil {
				return fmt.Errorf("failed to insert provider %s: %w", p.Name, err)
			}
		} else {
			// Обновляем существующего провайдера
			_, err = db.Exec(`
				UPDATE providers 
				SET name = ?, config = json_set(COALESCE(config, '{}'), '$.base_url', ?), 
				    updated_at = CURRENT_TIMESTAMP
				WHERE type = ?
			`, p.Name, p.BaseURL, p.Type)
			if err != nil {
				return fmt.Errorf("failed to update provider %s: %w", p.Name, err)
			}
		}
	}

	return nil
}

