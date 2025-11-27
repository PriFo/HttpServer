package normalization

import (
	"fmt"
	"log/slog"

	"httpserver/database"
	"httpserver/extractors"
)

// CounterpartyMapper сервис для автоматического мэппинга контрагентов
type CounterpartyMapper struct {
	serviceDB *database.ServiceDB
	logger    *slog.Logger
	analyzer  *CounterpartyDuplicateAnalyzer
}

// NewCounterpartyMapper создает новый сервис мэппинга контрагентов
func NewCounterpartyMapper(serviceDB *database.ServiceDB) *CounterpartyMapper {
	logger := slog.Default().With("component", "counterparty_mapper")
	return &CounterpartyMapper{
		serviceDB: serviceDB,
		logger:    logger,
		analyzer:  NewCounterpartyDuplicateAnalyzer(),
	}
}

// MapCounterpartiesFromDatabase выполняет мэппинг контрагентов из одной базы данных
func (cm *CounterpartyMapper) MapCounterpartiesFromDatabase(projectID, databaseID int) error {
	if cm.serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	cm.logger.Info("Starting counterparty mapping from database", "project_id", projectID, "database_id", databaseID)

	// Получаем базу данных проекта
	dbInfo, err := cm.serviceDB.GetProjectDatabase(databaseID)
	if err != nil {
		cm.logger.Error("Failed to get project database", "database_id", databaseID, "error", err)
		return fmt.Errorf("failed to get project database %d: %w", databaseID, err)
	}

	if dbInfo == nil {
		return fmt.Errorf("project database %d not found", databaseID)
	}

	if dbInfo.ClientProjectID != projectID {
		cm.logger.Error("Database does not belong to project",
			"database_id", databaseID,
			"database_project_id", dbInfo.ClientProjectID,
			"expected_project_id", projectID)
		return fmt.Errorf("database %d does not belong to project %d (belongs to project %d)", databaseID, projectID, dbInfo.ClientProjectID)
	}

	// Получаем все контрагенты из этой базы данных
	catalogItems, err := cm.serviceDB.GetCatalogItemsByDatabase(databaseID)
	if err != nil {
		return fmt.Errorf("failed to get catalog items: %w", err)
	}

	cm.logger.Info("Found catalog items", "count", len(catalogItems), "database_id", databaseID)

	if len(catalogItems) == 0 {
		cm.logger.Info("No catalog items found, skipping mapping", "database_id", databaseID)
		return nil
	}

	// Получаем конфигурацию проекта
	config, err := cm.serviceDB.GetProjectNormalizationConfig(projectID)
	if err != nil {
		cm.logger.Warn("Failed to get project normalization config, using defaults",
			"project_id", projectID,
			"error", err)
		config = &database.ProjectNormalizationConfig{
			ClientProjectID:         projectID,
			AutoMapCounterparties:   true,
			AutoMergeDuplicates:     true,
			MasterSelectionStrategy: "max_data",
		}
	}

	// Выполняем мэппинг и объединение дубликатов
	if err := cm.findAndMergeDuplicatesWithConfig(projectID, catalogItems, databaseID, config); err != nil {
		return fmt.Errorf("failed to merge duplicates: %w", err)
	}

	cm.logger.Info("Completed counterparty mapping", "project_id", projectID, "database_id", databaseID)
	return nil
}

// MapAllCounterpartiesForProject выполняет мэппинг всех контрагентов проекта
func (cm *CounterpartyMapper) MapAllCounterpartiesForProject(projectID int) error {
	if cm.serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	cm.logger.Info("Starting counterparty mapping for project", "project_id", projectID)

	// Убеждаемся, что проект существует
	if _, err := cm.serviceDB.GetClientProject(projectID); err != nil {
		return fmt.Errorf("project %d not found: %w", projectID, err)
	}

	// Проверяем конфигурацию проекта
	config, err := cm.serviceDB.GetProjectNormalizationConfig(projectID)
	if err != nil {
		cm.logger.Warn("Failed to get project normalization config, using defaults",
			"project_id", projectID,
			"error", err)
		// Продолжаем с настройками по умолчанию
		config = &database.ProjectNormalizationConfig{
			ClientProjectID:         projectID,
			AutoMapCounterparties:   true,
			AutoMergeDuplicates:     true,
			MasterSelectionStrategy: "max_data",
		}
	}

	// Если автоматический мэппинг отключен, пропускаем
	if !config.AutoMapCounterparties {
		cm.logger.Info("Auto-mapping is disabled for project", "project_id", projectID)
		return nil
	}

	// Получаем все базы данных проекта
	databases, err := cm.serviceDB.GetProjectDatabases(projectID, false)
	if err != nil {
		cm.logger.Error("Failed to get project databases", "project_id", projectID, "error", err)
		return fmt.Errorf("failed to get project databases for project %d: %w", projectID, err)
	}

	if len(databases) == 0 {
		cm.logger.Info("No databases found for project", "project_id", projectID)
		return nil
	}

	// Собираем все контрагенты из всех баз данных
	// Обрабатываем каждую базу отдельно, чтобы сохранить связи с базами данных
	totalMapped := 0
	successfulDatabases := 0
	failedDatabases := 0
	for _, dbInfo := range databases {
		items, err := cm.serviceDB.GetCatalogItemsByDatabase(dbInfo.ID)
		if err != nil {
			cm.logger.Warn("Failed to get catalog items from database",
				"database_id", dbInfo.ID,
				"database_name", dbInfo.Name,
				"error", err)
			failedDatabases++
			continue
		}

		if len(items) == 0 {
			cm.logger.Debug("No catalog items in database",
				"database_id", dbInfo.ID,
				"database_name", dbInfo.Name)
			continue
		}

		// Выполняем мэппинг для каждой базы отдельно, чтобы сохранить связи
		// Используем конфигурацию проекта для настройки процесса
		if err := cm.findAndMergeDuplicatesWithConfig(projectID, items, dbInfo.ID, config); err != nil {
			cm.logger.Warn("Failed to map counterparties from database",
				"database_id", dbInfo.ID,
				"database_name", dbInfo.Name,
				"items_count", len(items),
				"error", err)
			failedDatabases++
			continue
		}

		totalMapped += len(items)
		successfulDatabases++
		cm.logger.Info("Mapped counterparties from database",
			"database_id", dbInfo.ID,
			"database_name", dbInfo.Name,
			"items_count", len(items))
	}

	cm.logger.Info("Completed counterparty mapping for project",
		"project_id", projectID,
		"total_databases", len(databases),
		"successful_databases", successfulDatabases,
		"failed_databases", failedDatabases,
		"total_mapped", totalMapped)

	if failedDatabases > 0 {
		return fmt.Errorf("failed to map counterparties from %d out of %d databases", failedDatabases, len(databases))
	}

	return nil
}

// findAndMergeDuplicates находит и объединяет дубликаты контрагентов (использует конфигурацию по умолчанию)
func (cm *CounterpartyMapper) findAndMergeDuplicates(projectID int, counterparties []*database.CatalogItem, databaseID int) error {
	config := &database.ProjectNormalizationConfig{
		ClientProjectID:         projectID,
		AutoMapCounterparties:   true,
		AutoMergeDuplicates:     true,
		MasterSelectionStrategy: "max_data",
	}
	return cm.findAndMergeDuplicatesWithConfig(projectID, counterparties, databaseID, config)
}

// findAndMergeDuplicatesWithConfig находит и объединяет дубликаты контрагентов с учетом конфигурации
func (cm *CounterpartyMapper) findAndMergeDuplicatesWithConfig(projectID int, counterparties []*database.CatalogItem, databaseID int, config *database.ProjectNormalizationConfig) error {
	if cm.serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	if len(counterparties) == 0 {
		return nil
	}

	// Анализируем дубликаты
	duplicateGroups := cm.analyzer.AnalyzeDuplicates(counterparties)

	totalDuplicates := 0
	for _, group := range duplicateGroups {
		if len(group.Items) > 1 {
			totalDuplicates += len(group.Items)
		}
	}
	cm.logger.Info("Found duplicate groups", "groups_count", len(duplicateGroups), "total_duplicates", totalDuplicates, "project_id", projectID)

	// Если автоматическое объединение дубликатов отключено, только логируем группы
	if config != nil && !config.AutoMergeDuplicates {
		cm.logger.Info("Auto-merge is disabled, only analyzing duplicates",
			"project_id", projectID,
			"duplicate_groups", len(duplicateGroups))
		return nil
	}

	// Обрабатываем каждую группу дубликатов
	processedGroups := 0
	failedGroups := 0
	mergedDuplicates := 0

	for _, group := range duplicateGroups {
		if len(group.Items) < 2 {
			// Нет дубликатов, пропускаем
			continue
		}

		processedGroups++

		// Подсчитываем количество связанных баз данных для каждого элемента перед выбором эталона
		// Это нужно для правильного расчета DatabaseCount в calculateMasterScore
		for _, item := range group.Items {
			if item == nil {
				cm.logger.Warn("Nil item in duplicate group", "group_key", group.Key)
				continue
			}

			// Проверяем существующие связи для каждого элемента
			existing, err := cm.serviceDB.GetNormalizedCounterpartyBySourceReference(projectID, item.Reference)
			if err == nil && existing != nil {
				databases, err := cm.serviceDB.GetCounterpartyDatabases(existing.ID)
				if err == nil && databases != nil {
					item.DatabaseCount = len(databases)
				} else {
					// Если не удалось получить связи, считаем что есть хотя бы одна (текущая)
					if err != nil {
						cm.logger.Debug("Failed to get counterparty databases",
							"counterparty_id", existing.ID,
							"error", err)
					}
					if databaseID > 0 {
						item.DatabaseCount = 1 // Будет создана связь при сохранении
					} else {
						item.DatabaseCount = 0
					}
				}
				// Также обновляем данные об обогащении, если они есть
				if existing.EnrichmentApplied {
					item.EnrichmentApplied = existing.EnrichmentApplied
					item.SourceEnrichment = existing.SourceEnrichment
				}
				// Обновляем quality_score из существующей записи
				if existing.QualityScore > item.QualityScore {
					item.QualityScore = existing.QualityScore
				}
			} else {
				// Если контрагент еще не создан, DatabaseCount = 0 (или 1, если databaseID > 0)
				if err != nil {
					cm.logger.Debug("Error getting normalized counterparty",
						"project_id", projectID,
						"reference", item.Reference,
						"error", err)
				}
				if databaseID > 0 {
					item.DatabaseCount = 1 // Будет создана связь при сохранении
				} else {
					item.DatabaseCount = 0
				}
			}
		}

		// Выбираем эталонного контрагента с учетом DatabaseCount и других факторов
		masterItem := group.MasterItem
		if masterItem == nil {
			// Используем стратегию из конфигурации проекта
			strategy := "max_data"
			if config != nil && config.MasterSelectionStrategy != "" {
				strategy = config.MasterSelectionStrategy
			}
			masterItem = cm.analyzer.SelectMasterRecordWithStrategy(group.Items, strategy)
		}

		if masterItem == nil {
			cm.logger.Warn("No master item selected for duplicate group", "key", group.Key)
			continue
		}

		// Получаем или создаем нормализованного контрагента для эталона
		masterNormalized, err := cm.getOrCreateNormalizedCounterparty(projectID, masterItem, databaseID)
		if err != nil {
			cm.logger.Warn("Failed to get or create master normalized counterparty",
				"error", err,
				"item_id", masterItem.ID,
				"reference", masterItem.Reference,
				"group_key", group.Key)
			failedGroups++
			continue
		}

		if masterNormalized == nil {
			cm.logger.Warn("Master normalized counterparty is nil",
				"item_id", masterItem.ID,
				"reference", masterItem.Reference,
				"group_key", group.Key)
			failedGroups++
			continue
		}

		// Объединяем данные из дубликатов в эталонного
		for _, duplicateItem := range group.Items {
			if duplicateItem.ID == masterItem.ID {
				continue // Пропускаем эталонного
			}

			// Получаем или создаем нормализованного контрагента для дубликата
			duplicateNormalized, err := cm.getOrCreateNormalizedCounterparty(projectID, duplicateItem, databaseID)
			if err != nil {
				cm.logger.Warn("Failed to get or create duplicate normalized counterparty", "error", err, "item_id", duplicateItem.ID)
				continue
			}

			// Объединяем данные
			mergedData := cm.mergeCounterpartyData(masterNormalized, duplicateNormalized, duplicateItem, databaseID)
			if mergedData == nil {
				cm.logger.Warn("Failed to merge counterparty data, merged result is nil",
					"master_id", masterNormalized.ID,
					"duplicate_id", duplicateNormalized.ID)
				continue
			}
			masterNormalized = mergedData

			// Обновляем эталонного контрагента
			if err := cm.updateNormalizedCounterparty(masterNormalized); err != nil {
				cm.logger.Warn("Failed to update master normalized counterparty",
					"error", err,
					"master_id", masterNormalized.ID,
					"duplicate_id", duplicateNormalized.ID)
				continue
			}

			// Переносим все связи с базами данных из дубликата в эталон
			duplicateDatabases, err := cm.serviceDB.GetCounterpartyDatabases(duplicateNormalized.ID)
			databasesTransferred := 0
			if err == nil && duplicateDatabases != nil {
				for _, dbSource := range duplicateDatabases {
					// Создаем связь для эталона, если её еще нет
					if err := cm.serviceDB.SaveCounterpartyDatabaseLink(
						masterNormalized.ID,
						dbSource.DatabaseID,
						dbSource.SourceReference,
						dbSource.SourceName,
					); err != nil {
						cm.logger.Warn("Failed to transfer database link from duplicate to master",
							"error", err,
							"duplicate_id", duplicateNormalized.ID,
							"master_id", masterNormalized.ID,
							"database_id", dbSource.DatabaseID)
					} else {
						databasesTransferred++
					}
				}
			}

			// Удаляем дубликат (или помечаем как объединенный)
			// Вместо удаления, можно добавить поле merged_into_id
			// Пока оставляем дубликат в базе, но все связи перенесены в эталон
			cm.logger.Info("Merged duplicate into master",
				"duplicate_id", duplicateNormalized.ID,
				"duplicate_reference", duplicateItem.Reference,
				"master_id", masterNormalized.ID,
				"master_name", masterNormalized.NormalizedName,
				"database_id", databaseID,
				"databases_transferred", databasesTransferred)
			mergedDuplicates++
		}
	}

	cm.logger.Info("Completed duplicate merging",
		"project_id", projectID,
		"database_id", databaseID,
		"processed_groups", processedGroups,
		"failed_groups", failedGroups,
		"merged_duplicates", mergedDuplicates)

	if failedGroups > 0 && failedGroups == processedGroups {
		// Если все группы провалились, возвращаем ошибку
		return fmt.Errorf("failed to process all %d duplicate groups", processedGroups)
	}

	return nil
}

// getOrCreateNormalizedCounterparty получает или создает нормализованного контрагента
func (cm *CounterpartyMapper) getOrCreateNormalizedCounterparty(projectID int, item *CounterpartyDuplicateItem, databaseID int) (*database.NormalizedCounterparty, error) {
	if cm.serviceDB == nil {
		return nil, fmt.Errorf("serviceDB is nil")
	}

	if item == nil {
		return nil, fmt.Errorf("item is nil")
	}

	// Пробуем найти существующего нормализованного контрагента
	normalized, err := cm.serviceDB.GetNormalizedCounterpartyBySourceReference(projectID, item.Reference)
	if err == nil && normalized != nil {
		return normalized, nil
	}

	// Если ошибка не связана с отсутствием записи, логируем её
	if err != nil {
		cm.logger.Debug("Error getting normalized counterparty by reference",
			"project_id", projectID,
			"reference", item.Reference,
			"error", err)
		// Продолжаем создание нового контрагента
	}

	// Используем данные из CounterpartyDuplicateItem (уже извлечены при создании)
	inn := item.INN
	kpp := item.KPP
	bin := item.BIN

	// Если данные не были извлечены, пробуем извлечь из атрибутов
	if inn == "" && bin == "" && item.Attributes != "" {
		if extractedINN, err := extractors.ExtractINNFromAttributes(item.Attributes); err == nil {
			inn = extractedINN
		}
		if extractedBIN, err := extractors.ExtractBINFromAttributes(item.Attributes); err == nil {
			bin = extractedBIN
		}
	}
	if kpp == "" && item.Attributes != "" {
		if extractedKPP, err := extractors.ExtractKPPFromAttributes(item.Attributes); err == nil {
			kpp = extractedKPP
		}
	}

	// Создаем нового нормализованного контрагента
	normalizedName := item.Name
	if normalizedName == "" {
		normalizedName = "Без названия"
	}

	// Получаем имя базы данных для source_database
	sourceDatabase := ""
	if databaseID > 0 {
		dbInfo, err := cm.serviceDB.GetProjectDatabase(databaseID)
		if err != nil {
			cm.logger.Warn("Failed to get project database for source_database",
				"database_id", databaseID,
				"error", err)
			// Продолжаем без имени базы данных
		} else if dbInfo != nil {
			sourceDatabase = dbInfo.Name
		}
	}

	// Извлекаем дополнительные данные из атрибутов, если они не были заполнены
	legalAddress := item.LegalAddress
	postalAddress := item.PostalAddress
	contactPhone := item.ContactPhone
	contactEmail := item.ContactEmail
	contactPerson := item.ContactPerson
	bankName := item.BankName
	bankAccount := item.BankAccount
	correspondentAccount := item.CorrespondentAccount
	bik := item.BIK

	// Если данные не были извлечены, пробуем извлечь из атрибутов
	if item.Attributes != "" {
		if legalAddress == "" {
			if addr, err := extractors.ExtractAddressFromAttributes(item.Attributes); err == nil {
				legalAddress = addr
				postalAddress = addr
			}
		}
		if contactPhone == "" {
			if phone, err := extractors.ExtractContactPhoneFromAttributes(item.Attributes); err == nil {
				contactPhone = phone
			}
		}
		if contactEmail == "" {
			if email, err := extractors.ExtractContactEmailFromAttributes(item.Attributes); err == nil {
				contactEmail = email
			}
		}
		if contactPerson == "" {
			if person, err := extractors.ExtractContactPersonFromAttributes(item.Attributes); err == nil {
				contactPerson = person
			}
		}
		if bankName == "" {
			if bank, err := extractors.ExtractBankNameFromAttributes(item.Attributes); err == nil {
				bankName = bank
			}
		}
		if bankAccount == "" {
			if account, err := extractors.ExtractBankAccountFromAttributes(item.Attributes); err == nil {
				bankAccount = account
			}
		}
		if correspondentAccount == "" {
			if corrAccount, err := extractors.ExtractCorrespondentAccountFromAttributes(item.Attributes); err == nil {
				correspondentAccount = corrAccount
			}
		}
		if bik == "" {
			if bikCode, err := extractors.ExtractBIKFromAttributes(item.Attributes); err == nil {
				bik = bikCode
			}
		}
	}

	// Сохраняем нормализованного контрагента
	err = cm.serviceDB.SaveNormalizedCounterparty(
		projectID,
		item.Reference,
		item.Name,
		normalizedName,
		inn,
		kpp,
		bin,
		legalAddress,
		postalAddress,
		contactPhone,
		contactEmail,
		contactPerson,
		"", // legal_form
		bankName,
		bankAccount,
		correspondentAccount,
		bik,
		0, // benchmark_id
		item.QualityScore,
		false, // enrichment_applied
		"",    // source_enrichment
		sourceDatabase,
		"", // subcategory
	)
	if err != nil {
		return nil, fmt.Errorf("failed to save normalized counterparty: %w", err)
	}

	// Получаем созданного контрагента
	normalized, err = cm.serviceDB.GetNormalizedCounterpartyBySourceReference(projectID, item.Reference)
	if err != nil {
		cm.logger.Error("Failed to get created normalized counterparty",
			"project_id", projectID,
			"reference", item.Reference,
			"error", err)
		return nil, fmt.Errorf("failed to get created normalized counterparty (reference: %s): %w", item.Reference, err)
	}

	if normalized == nil {
		return nil, fmt.Errorf("created normalized counterparty is nil (reference: %s)", item.Reference)
	}

	// Создаем связь с базой данных
	if databaseID > 0 {
		if err := cm.serviceDB.SaveCounterpartyDatabaseLink(normalized.ID, databaseID, item.Reference, item.Name); err != nil {
			cm.logger.Warn("Failed to save counterparty database link", "error", err)
		}
	}

	return normalized, nil
}

// mergeCounterpartyData объединяет данные из дубликата в эталонного контрагента
func (cm *CounterpartyMapper) mergeCounterpartyData(master, duplicate *database.NormalizedCounterparty, duplicateItem *CounterpartyDuplicateItem, databaseID int) *database.NormalizedCounterparty {
	if master == nil {
		cm.logger.Error("Master counterparty is nil in mergeCounterpartyData")
		return nil
	}

	if duplicate == nil {
		cm.logger.Warn("Duplicate counterparty is nil in mergeCounterpartyData, returning master")
		return master
	}

	merged := &database.NormalizedCounterparty{
		ID:                   master.ID,
		ClientProjectID:      master.ClientProjectID,
		SourceReference:      master.SourceReference,
		SourceName:           master.SourceName,
		NormalizedName:       master.NormalizedName,
		TaxID:                master.TaxID,
		KPP:                  master.KPP,
		BIN:                  master.BIN,
		LegalAddress:         master.LegalAddress,
		PostalAddress:        master.PostalAddress,
		ContactPhone:         master.ContactPhone,
		ContactEmail:         master.ContactEmail,
		ContactPerson:        master.ContactPerson,
		LegalForm:            master.LegalForm,
		BankName:             master.BankName,
		BankAccount:          master.BankAccount,
		CorrespondentAccount: master.CorrespondentAccount,
		BIK:                  master.BIK,
		BenchmarkID:          master.BenchmarkID,
		QualityScore:         master.QualityScore,
		EnrichmentApplied:    master.EnrichmentApplied,
		SourceEnrichment:     master.SourceEnrichment,
		SourceDatabase:       master.SourceDatabase,
		Subcategory:          master.Subcategory,
	}

	// Объединяем поля: используем более полные данные
	// Если поле пустое в эталоне, но заполнено в дубликате, используем значение из дубликата
	// Если оба заполнены, выбираем более полное значение (длиннее или более информативное)
	if merged.TaxID == "" && duplicate.TaxID != "" {
		merged.TaxID = duplicate.TaxID
	} else if merged.TaxID != "" && duplicate.TaxID != "" && len(duplicate.TaxID) > len(merged.TaxID) {
		// Используем более длинный ИНН (может быть 12-значный вместо 10-значного)
		merged.TaxID = duplicate.TaxID
	}
	if merged.BIN == "" && duplicate.BIN != "" {
		merged.BIN = duplicate.BIN
	}
	if merged.KPP == "" && duplicate.KPP != "" {
		merged.KPP = duplicate.KPP
	}
	if merged.LegalAddress == "" && duplicate.LegalAddress != "" {
		merged.LegalAddress = duplicate.LegalAddress
	} else if merged.LegalAddress != "" && duplicate.LegalAddress != "" && len(duplicate.LegalAddress) > len(merged.LegalAddress) {
		// Используем более полный адрес
		merged.LegalAddress = duplicate.LegalAddress
	}
	if merged.PostalAddress == "" && duplicate.PostalAddress != "" {
		merged.PostalAddress = duplicate.PostalAddress
	} else if merged.PostalAddress != "" && duplicate.PostalAddress != "" && len(duplicate.PostalAddress) > len(merged.PostalAddress) {
		merged.PostalAddress = duplicate.PostalAddress
	}
	if merged.ContactPhone == "" && duplicate.ContactPhone != "" {
		merged.ContactPhone = duplicate.ContactPhone
	} else if merged.ContactPhone != "" && duplicate.ContactPhone != "" {
		// Если оба заполнены, проверяем формат (предпочитаем более полный формат с кодом страны)
		if len(duplicate.ContactPhone) > len(merged.ContactPhone) {
			merged.ContactPhone = duplicate.ContactPhone
		}
	}
	if merged.ContactEmail == "" && duplicate.ContactEmail != "" {
		merged.ContactEmail = duplicate.ContactEmail
	}
	if merged.ContactPerson == "" && duplicate.ContactPerson != "" {
		merged.ContactPerson = duplicate.ContactPerson
	} else if merged.ContactPerson != "" && duplicate.ContactPerson != "" && len(duplicate.ContactPerson) > len(merged.ContactPerson) {
		// Используем более полное имя контактного лица
		merged.ContactPerson = duplicate.ContactPerson
	}
	if merged.LegalForm == "" && duplicate.LegalForm != "" {
		merged.LegalForm = duplicate.LegalForm
	}
	if merged.BankName == "" && duplicate.BankName != "" {
		merged.BankName = duplicate.BankName
	} else if merged.BankName != "" && duplicate.BankName != "" && len(duplicate.BankName) > len(merged.BankName) {
		// Используем более полное название банка
		merged.BankName = duplicate.BankName
	}
	if merged.BankAccount == "" && duplicate.BankAccount != "" {
		merged.BankAccount = duplicate.BankAccount
	}
	if merged.CorrespondentAccount == "" && duplicate.CorrespondentAccount != "" {
		merged.CorrespondentAccount = duplicate.CorrespondentAccount
	}
	if merged.BIK == "" && duplicate.BIK != "" {
		merged.BIK = duplicate.BIK
	}

	// Используем максимальный quality_score
	if duplicate.QualityScore > merged.QualityScore {
		merged.QualityScore = duplicate.QualityScore
	}

	// Если дубликат имеет обогащенные данные, а эталон нет, используем данные дубликата
	if duplicate.EnrichmentApplied && !merged.EnrichmentApplied {
		merged.EnrichmentApplied = duplicate.EnrichmentApplied
		merged.SourceEnrichment = duplicate.SourceEnrichment
	}

	// Создаем связь с базой данных для дубликата
	if databaseID > 0 {
		if err := cm.serviceDB.SaveCounterpartyDatabaseLink(merged.ID, databaseID, duplicateItem.Reference, duplicateItem.Name); err != nil {
			cm.logger.Warn("Failed to save duplicate counterparty database link", "error", err)
		}
	}

	return merged
}

// updateNormalizedCounterparty обновляет нормализованного контрагента
func (cm *CounterpartyMapper) updateNormalizedCounterparty(nc *database.NormalizedCounterparty) error {
	if cm.serviceDB == nil {
		return fmt.Errorf("serviceDB is nil")
	}

	if nc == nil {
		return fmt.Errorf("normalized counterparty is nil")
	}

	err := cm.serviceDB.UpdateNormalizedCounterparty(
		nc.ID,
		nc.NormalizedName,
		nc.TaxID,
		nc.KPP,
		nc.BIN,
		nc.LegalAddress,
		nc.PostalAddress,
		nc.ContactPhone,
		nc.ContactEmail,
		nc.ContactPerson,
		nc.LegalForm,
		nc.BankName,
		nc.BankAccount,
		nc.CorrespondentAccount,
		nc.BIK,
		nc.QualityScore,
		nc.SourceEnrichment,
		nc.Subcategory,
	)

	if err != nil {
		cm.logger.Error("Failed to update normalized counterparty",
			"counterparty_id", nc.ID,
			"reference", nc.SourceReference,
			"error", err)
		return fmt.Errorf("failed to update normalized counterparty %d: %w", nc.ID, err)
	}

	return nil
}
