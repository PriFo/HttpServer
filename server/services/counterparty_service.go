package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"httpserver/database"
	"httpserver/normalization"
	apperrors "httpserver/server/errors"
)

// BenchmarkFinderAdapter адаптер для BenchmarkService, реализующий интерфейс BenchmarkFinder
type BenchmarkFinderAdapter struct {
	BenchmarkService *BenchmarkService
}

// FindBestMatch реализует интерфейс normalization.BenchmarkFinder
func (a *BenchmarkFinderAdapter) FindBestMatch(name string, entityType string) (normalizedName string, found bool, err error) {
	if a.BenchmarkService == nil {
		return "", false, nil
	}
	benchmark, err := a.BenchmarkService.FindBestMatch(name, entityType)
	if err != nil {
		return "", false, err
	}
	if benchmark == nil {
		return "", false, nil
	}
	return benchmark.Name, true, nil
}

// CounterpartyService сервис для управления нормализацией контрагентов
// АРХИТЕКТУРНАЯ ЗАМЕТКА: normalizerRunning дублируется в Server и NormalizationService.
// TODO: Централизовать управление состоянием через services.NormalizationStateManager интерфейс.
// См. services/normalization_service.go для деталей рефакторинга.
type CounterpartyService struct {
	serviceDB         *database.ServiceDB
	benchmarkService  *BenchmarkService
	detector          *database.CounterpartyDetector
	normalizerMutex   sync.RWMutex
	normalizerRunning bool
	normalizerEvents  chan<- string
	normalizerCtx     context.Context
	normalizerCancel  context.CancelFunc
}

// NewCounterpartyService создает новый сервис нормализации контрагентов
func NewCounterpartyService(
	serviceDB *database.ServiceDB,
	normalizerEvents chan<- string,
	benchmarkService *BenchmarkService,
) *CounterpartyService {
	return &CounterpartyService{
		serviceDB:         serviceDB,
		benchmarkService:  benchmarkService,
		detector:          database.NewCounterpartyDetector(serviceDB),
		normalizerEvents:  normalizerEvents,
		normalizerRunning: false,
	}
}

// GetServiceDB возвращает ServiceDB для доступа к базе данных
func (cs *CounterpartyService) GetServiceDB() *database.ServiceDB {
	return cs.serviceDB
}

// IsRunning проверяет, запущена ли нормализация контрагентов
func (cs *CounterpartyService) IsRunning() bool {
	cs.normalizerMutex.RLock()
	defer cs.normalizerMutex.RUnlock()
	return cs.normalizerRunning
}

// Start запускает нормализацию контрагентов
func (cs *CounterpartyService) Start() error {
	cs.normalizerMutex.Lock()
	defer cs.normalizerMutex.Unlock()

	if cs.normalizerRunning {
		return apperrors.NewConflictError("нормализация контрагентов уже запущена", nil)
	}

	cs.normalizerRunning = true
	return nil
}

// Stop останавливает нормализацию контрагентов
func (cs *CounterpartyService) Stop() bool {
	cs.normalizerMutex.Lock()
	defer cs.normalizerMutex.Unlock()

	wasRunning := cs.normalizerRunning
	cs.normalizerRunning = false

	if cs.normalizerCancel != nil {
		cs.normalizerCancel()
		cs.normalizerCancel = nil
	}

	return wasRunning
}

// CreateNormalizer создает нормализатор контрагентов с контекстом для управления отменой
// Если контекст не установлен, создается новый контекст с возможностью отмены
func (cs *CounterpartyService) CreateNormalizer(
	clientID, projectID int,
	nameNormalizer normalization.AINameNormalizer,
) *normalization.CounterpartyNormalizer {
	// Используем существующий контекст или создаем новый
	cs.normalizerMutex.Lock()
	if cs.normalizerCtx == nil {
		cs.normalizerCtx, cs.normalizerCancel = context.WithCancel(context.Background())
	}
	ctx := cs.normalizerCtx
	cs.normalizerMutex.Unlock()

	var benchmarkFinder normalization.BenchmarkFinder
	if cs.benchmarkService != nil {
		benchmarkFinder = &BenchmarkFinderAdapter{BenchmarkService: cs.benchmarkService}
	}

	return normalization.NewCounterpartyNormalizer(
		cs.serviceDB,
		clientID,
		projectID,
		cs.normalizerEvents,
		ctx,
		nameNormalizer,
		benchmarkFinder,
	)
}

// GetNormalizedCounterpartyStats получает статистику по нормализованным контрагентам проекта
func (cs *CounterpartyService) GetNormalizedCounterpartyStats(projectID int) (map[string]interface{}, error) {
	return cs.serviceDB.GetNormalizedCounterpartyStats(projectID)
}

// GetNormalizedCounterparty получает контрагента по ID
func (cs *CounterpartyService) GetNormalizedCounterparty(id int) (*database.NormalizedCounterparty, error) {
	return cs.serviceDB.GetNormalizedCounterparty(id)
}

// UpdateNormalizedCounterparty обновляет контрагента
func (cs *CounterpartyService) UpdateNormalizedCounterparty(
	id int,
	normalizedName string,
	taxID, kpp, bin string,
	legalAddress, postalAddress string,
	contactPhone, contactEmail string,
	contactPerson, legalForm string,
	bankName, bankAccount string,
	correspondentAccount, bik string,
	qualityScore float64,
	sourceEnrichment, subcategory string,
) error {
	return cs.serviceDB.UpdateNormalizedCounterparty(
		id,
		normalizedName,
		taxID, kpp, bin,
		legalAddress, postalAddress,
		contactPhone, contactEmail,
		contactPerson, legalForm,
		bankName, bankAccount,
		correspondentAccount, bik,
		qualityScore,
		sourceEnrichment,
		subcategory,
	)
}

// GetNormalizedCounterparties получает список нормализованных контрагентов проекта
func (cs *CounterpartyService) GetNormalizedCounterparties(projectID int, limit, offset int, search, taxID, bin string) ([]*database.NormalizedCounterparty, int, error) {
	return cs.serviceDB.GetNormalizedCounterparties(projectID, limit, offset, search, taxID, bin)
}

// GetCounterpartyDuplicates получает группы дубликатов контрагентов по проекту
func (cs *CounterpartyService) GetCounterpartyDuplicates(projectID int) ([]map[string]interface{}, error) {
	// Получаем всех контрагентов проекта
	counterparties, _, err := cs.serviceDB.GetNormalizedCounterparties(projectID, 0, 10000, "", "", "")
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить контрагентов", err)
	}

	// Группируем по ИНН/БИН
	groups := make(map[string][]*database.NormalizedCounterparty)
	for _, cp := range counterparties {
		key := cp.TaxID
		if key == "" {
			key = cp.BIN
		}
		if key != "" {
			groups[key] = append(groups[key], cp)
		}
	}

	// Фильтруем только группы с дубликатами
	duplicateGroups := []map[string]interface{}{}
	for key, items := range groups {
		if len(items) > 1 {
			duplicateGroups = append(duplicateGroups, map[string]interface{}{
				"tax_id": key,
				"count":  len(items),
				"items":  items,
			})
		}
	}

	return duplicateGroups, nil
}

// MergeCounterpartyDuplicates выполняет слияние дубликатов контрагентов
// Сохраняет все связи с базами данных из дубликатов в эталонного контрагента
func (cs *CounterpartyService) MergeCounterpartyDuplicates(masterID int, mergeIDs []int) (*database.NormalizedCounterparty, error) {
	// Получаем мастер-контрагента
	master, err := cs.serviceDB.GetNormalizedCounterparty(masterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("мастер-контрагент не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить мастер-контрагента", err)
	}

	// Получаем связи мастер-контрагента с базами данных
	masterDatabases, err := cs.serviceDB.GetCounterpartyDatabases(masterID)
	if err != nil {
		// Если ошибка, продолжаем с пустым списком
		masterDatabases = []database.DatabaseSource{}
	}

	// Создаем map для быстрой проверки существующих связей
	existingLinks := make(map[int]bool) // key: databaseID
	for _, dbSource := range masterDatabases {
		existingLinks[dbSource.DatabaseID] = true
	}

	// Объединяем данные из дубликатов в мастер
	for _, mergeID := range mergeIDs {
		if mergeID == masterID {
			continue
		}

		duplicate, err := cs.serviceDB.GetNormalizedCounterparty(mergeID)
		if err != nil {
			continue
		}

		// Получаем связи дубликата с базами данных
		duplicateDatabases, err := cs.serviceDB.GetCounterpartyDatabases(mergeID)
		if err == nil {
			// Сохраняем все связи дубликата в мастер-контрагента
			for _, dbSource := range duplicateDatabases {
				if !existingLinks[dbSource.DatabaseID] {
					// Создаем новую связь, если её еще нет
					err := cs.serviceDB.SaveCounterpartyDatabaseLink(
						masterID,
						dbSource.DatabaseID,
						dbSource.SourceReference,
						dbSource.SourceName,
					)
					if err == nil {
						existingLinks[dbSource.DatabaseID] = true
					}
				}
			}
		}

		// Объединяем данные (выбираем максимальный набор данных)
		// Если поле пустое в эталоне, но заполнено в дубликате, используем значение из дубликата
		// Если оба заполнены, выбираем более полное значение
		if master.TaxID == "" && duplicate.TaxID != "" {
			master.TaxID = duplicate.TaxID
		}
		if master.BIN == "" && duplicate.BIN != "" {
			master.BIN = duplicate.BIN
		}
		if master.KPP == "" && duplicate.KPP != "" {
			master.KPP = duplicate.KPP
		}
		if master.LegalAddress == "" && duplicate.LegalAddress != "" {
			master.LegalAddress = duplicate.LegalAddress
		} else if master.LegalAddress != "" && duplicate.LegalAddress != "" && len(duplicate.LegalAddress) > len(master.LegalAddress) {
			// Используем более полный адрес
			master.LegalAddress = duplicate.LegalAddress
		}
		if master.PostalAddress == "" && duplicate.PostalAddress != "" {
			master.PostalAddress = duplicate.PostalAddress
		} else if master.PostalAddress != "" && duplicate.PostalAddress != "" && len(duplicate.PostalAddress) > len(master.PostalAddress) {
			master.PostalAddress = duplicate.PostalAddress
		}
		if master.ContactPhone == "" && duplicate.ContactPhone != "" {
			master.ContactPhone = duplicate.ContactPhone
		}
		if master.ContactEmail == "" && duplicate.ContactEmail != "" {
			master.ContactEmail = duplicate.ContactEmail
		}
		if master.ContactPerson == "" && duplicate.ContactPerson != "" {
			master.ContactPerson = duplicate.ContactPerson
		} else if master.ContactPerson != "" && duplicate.ContactPerson != "" && len(duplicate.ContactPerson) > len(master.ContactPerson) {
			master.ContactPerson = duplicate.ContactPerson
		}
		if master.BankName == "" && duplicate.BankName != "" {
			master.BankName = duplicate.BankName
		}
		if master.BankAccount == "" && duplicate.BankAccount != "" {
			master.BankAccount = duplicate.BankAccount
		}
		if master.CorrespondentAccount == "" && duplicate.CorrespondentAccount != "" {
			master.CorrespondentAccount = duplicate.CorrespondentAccount
		}
		if master.BIK == "" && duplicate.BIK != "" {
			master.BIK = duplicate.BIK
		}
		if master.LegalForm == "" && duplicate.LegalForm != "" {
			master.LegalForm = duplicate.LegalForm
		}
	}

	// Обновляем мастер-контрагента
	err = cs.serviceDB.UpdateNormalizedCounterparty(
		masterID,
		master.NormalizedName,
		master.TaxID, master.KPP, master.BIN,
		master.LegalAddress, master.PostalAddress,
		master.ContactPhone, master.ContactEmail,
		master.ContactPerson, master.LegalForm,
		master.BankName, master.BankAccount,
		master.CorrespondentAccount, master.BIK,
		master.QualityScore,
		master.SourceEnrichment,
		master.Subcategory,
	)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось обновить мастер-контрагента", err)
	}

	// Удаляем дубликаты
	for _, mergeID := range mergeIDs {
		if mergeID != masterID {
			if err := cs.serviceDB.DeleteNormalizedCounterparty(mergeID); err != nil {
				// Логируем ошибку, но продолжаем
				continue
			}
		}
	}

	// Получаем обновленного мастер-контрагента
	updated, err := cs.serviceDB.GetNormalizedCounterparty(masterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("обновленный мастер-контрагент не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить обновленного мастер-контрагента", err)
	}

	return updated, nil
}

// GetNormalizedCounterpartiesByClient получает нормализованных контрагентов по клиенту
func (cs *CounterpartyService) GetNormalizedCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error) {
	return cs.serviceDB.GetNormalizedCounterpartiesByClient(clientID, projectID, offset, limit, search, enrichment, subcategory)
}

// GetClientProject получает проект по ID
func (cs *CounterpartyService) GetClientProject(projectID int) (*database.ClientProject, error) {
	return cs.serviceDB.GetClientProject(projectID)
}

// GetCounterpartiesFromDatabase получает контрагентов из конкретной БД используя детектор
func (cs *CounterpartyService) GetCounterpartiesFromDatabase(databaseID int, dbPath string, limit, offset int) ([]map[string]interface{}, error) {
	// 1. Проверяем кэш метаданных
	structure, err := cs.detector.GetCachedMetadata(databaseID)
	if err != nil || structure == nil {
		// 2. Если кэша нет - обнаруживаем структуру
		structure, err = cs.detector.DetectStructure(databaseID, dbPath)
		if err != nil {
			return nil, fmt.Errorf("failed to detect counterparty structure: %w", err)
		}
	}
	
	// 3. Проверяем confidence
	if structure.Confidence < 0.7 {
		return nil, fmt.Errorf("low confidence score (%.2f) for database %d", structure.Confidence, databaseID)
	}
	
	// 4. Получаем контрагентов используя обнаруженную структуру
	counterparties, err := cs.detector.GetCounterparties(dbPath, structure, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get counterparties: %w", err)
	}
	
	return counterparties, nil
}

// GetAllCounterpartiesByClient получает все контрагенты (из баз и нормализованных) по клиенту
func (cs *CounterpartyService) GetAllCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, source, sortBy, order string, minQuality, maxQuality *float64) (*database.GetAllCounterpartiesByClientResult, error) {
	// Если source == "database", используем детектор для загрузки из исходных БД
	if source == "database" {
		// Получаем все проекты клиента
		var projects []*database.ClientProject
		var err error
		
		if projectID != nil {
			project, err := cs.serviceDB.GetClientProject(*projectID)
			if err != nil {
				return nil, fmt.Errorf("failed to get project: %w", err)
			}
			projects = []*database.ClientProject{project}
		} else {
			projects, err = cs.serviceDB.GetClientProjects(clientID)
			if err != nil {
				return nil, fmt.Errorf("failed to get client projects: %w", err)
			}
		}
		
		// Собираем контрагентов из всех БД всех проектов
		allCounterparties := []map[string]interface{}{}
		for _, project := range projects {
			// Получаем БД проекта
			databases, err := cs.serviceDB.GetProjectDatabases(project.ID, true)
			if err != nil {
				continue
			}
			
			for _, db := range databases {
				counterparties, err := cs.GetCounterpartiesFromDatabase(db.ID, db.FilePath, 1000, 0)
				if err != nil {
					// Логируем, но продолжаем
					fmt.Printf("Warning: failed to get counterparties from DB %d: %v\n", db.ID, err)
					continue
				}
				
				// Добавляем информацию об источнике
				for i := range counterparties {
					counterparties[i]["source"] = "database"
					counterparties[i]["database_id"] = db.ID
					counterparties[i]["database_name"] = db.Name
					counterparties[i]["project_id"] = project.ID
					counterparties[i]["project_name"] = project.Name
				}
				
				allCounterparties = append(allCounterparties, counterparties...)
			}
		}
		
		// TODO: Применить фильтры (search, качество)
		// TODO: Применить сортировку
		// TODO: Дедупликация по ИНН/БИН
		
		// Преобразуем в UnifiedCounterparty
		unified := make([]*database.UnifiedCounterparty, 0, len(allCounterparties))
		for _, cp := range allCounterparties {
			// Извлекаем значения из map
			name, _ := cp["name"].(string)
			inn, _ := cp["inn"].(string)
			bin, _ := cp["bin"].(string)
			// ogrn not used in current implementation
			kpp, _ := cp["kpp"].(string)
			address, _ := cp["address"].(string)
			phone, _ := cp["phone"].(string)
			email, _ := cp["email"].(string)
			
			dbID, _ := cp["database_id"].(int)
			dbName, _ := cp["database_name"].(string)
			projID, _ := cp["project_id"].(int)
			projName, _ := cp["project_name"].(string)
			
			unified = append(unified, &database.UnifiedCounterparty{
				Name:          name,
				Source:        "database",
				ProjectID:     projID,
				ProjectName:   projName,
				DatabaseID:    &dbID,
				DatabaseName:  dbName,
				TaxID:         inn,
				BIN:           bin,
				KPP:           kpp,
				LegalAddress:  address,
				ContactPhone:  phone,
				ContactEmail:  email,
			})
		}
		
		// Применяем пагинацию
		total := len(unified)
		start := offset
		end := offset + limit
		
		if start > total {
			start = total
		}
		if end > total {
			end = total
		}
		
		paginatedCounterparties := unified[start:end]
		
		return &database.GetAllCounterpartiesByClientResult{
			Counterparties: paginatedCounterparties,
			TotalCount:     total,
			Stats: &database.CounterpartiesStats{
				TotalFromDatabase: total,
				TotalNormalized:   0,
			},
		}, nil
	}
	
	// Для остальных source используем существующую логику (normalized)
	return cs.serviceDB.GetAllCounterpartiesByClient(clientID, projectID, offset, limit, search, source, sortBy, order, minQuality, maxQuality)
}

// BulkUpdateCounterparties выполняет массовое обновление контрагентов
func (cs *CounterpartyService) BulkUpdateCounterparties(ids []int, updates map[string]interface{}) (map[string]interface{}, error) {
	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, id := range ids {
		cp, err := cs.serviceDB.GetNormalizedCounterparty(id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: %v", id, err))
			continue
		}

		// Применяем обновления
		normalizedName := cp.NormalizedName
		if v, ok := updates["normalized_name"].(*string); ok && v != nil {
			normalizedName = *v
		}
		taxID := cp.TaxID
		if v, ok := updates["tax_id"].(*string); ok && v != nil {
			taxID = *v
		}
		kpp := cp.KPP
		if v, ok := updates["kpp"].(*string); ok && v != nil {
			kpp = *v
		}
		bin := cp.BIN
		if v, ok := updates["bin"].(*string); ok && v != nil {
			bin = *v
		}
		legalAddress := cp.LegalAddress
		if v, ok := updates["legal_address"].(*string); ok && v != nil {
			legalAddress = *v
		}
		postalAddress := cp.PostalAddress
		if v, ok := updates["postal_address"].(*string); ok && v != nil {
			postalAddress = *v
		}
		contactPhone := cp.ContactPhone
		if v, ok := updates["contact_phone"].(*string); ok && v != nil {
			contactPhone = *v
		}
		contactEmail := cp.ContactEmail
		if v, ok := updates["contact_email"].(*string); ok && v != nil {
			contactEmail = *v
		}
		contactPerson := cp.ContactPerson
		if v, ok := updates["contact_person"].(*string); ok && v != nil {
			contactPerson = *v
		}
		legalForm := cp.LegalForm
		if v, ok := updates["legal_form"].(*string); ok && v != nil {
			legalForm = *v
		}
		bankName := cp.BankName
		if v, ok := updates["bank_name"].(*string); ok && v != nil {
			bankName = *v
		}
		bankAccount := cp.BankAccount
		if v, ok := updates["bank_account"].(*string); ok && v != nil {
			bankAccount = *v
		}
		correspondentAccount := cp.CorrespondentAccount
		if v, ok := updates["correspondent_account"].(*string); ok && v != nil {
			correspondentAccount = *v
		}
		bik := cp.BIK
		if v, ok := updates["bik"].(*string); ok && v != nil {
			bik = *v
		}
		qualityScore := cp.QualityScore
		if v, ok := updates["quality_score"].(*float64); ok && v != nil {
			qualityScore = *v
		}
		sourceEnrichment := cp.SourceEnrichment
		if v, ok := updates["source_enrichment"].(*string); ok && v != nil {
			sourceEnrichment = *v
		}
		subcategory := cp.Subcategory
		if v, ok := updates["subcategory"].(*string); ok && v != nil {
			subcategory = *v
		}

		err = cs.serviceDB.UpdateNormalizedCounterparty(
			id,
			normalizedName,
			taxID, kpp, bin,
			legalAddress, postalAddress,
			contactPhone, contactEmail,
			contactPerson, legalForm,
			bankName, bankAccount,
			correspondentAccount, bik,
			qualityScore,
			sourceEnrichment,
			subcategory,
		)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: %v", id, err))
			continue
		}

		successCount++
	}

	result := map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(ids),
		"success_count": successCount,
		"failed_count":  failedCount,
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}

	return result, nil
}

// BulkDeleteCounterparties выполняет массовое удаление контрагентов
func (cs *CounterpartyService) BulkDeleteCounterparties(ids []int) (map[string]interface{}, error) {
	successCount := 0
	failedCount := 0
	errors := []string{}

	for _, id := range ids {
		err := cs.serviceDB.DeleteNormalizedCounterparty(id)
		if err != nil {
			failedCount++
			errors = append(errors, fmt.Sprintf("Counterparty %d: %v", id, err))
			continue
		}
		successCount++
	}

	result := map[string]interface{}{
		"success":       failedCount == 0,
		"total":         len(ids),
		"success_count": successCount,
		"failed_count":  failedCount,
	}
	if len(errors) > 0 {
		result["errors"] = errors
	}

	return result, nil
}

// BulkEnrichCounterparties выполняет массовое обогащение контрагентов
func (cs *CounterpartyService) BulkEnrichCounterparties(ids []int, enrichmentFactory interface {
	Enrich(inn, bin string) interface {
		Success() bool
		Errors() []string
		Results() []interface {
			FullName() string
			INN() string
			BIN() string
			LegalAddress() string
			Phone() string
			Email() string
		}
	}
	GetBestResult(results []interface {
		FullName() string
		INN() string
		BIN() string
		LegalAddress() string
		Phone() string
		Email() string
	}) interface {
		FullName() string
		INN() string
		BIN() string
		LegalAddress() string
		Phone() string
		Email() string
	}
}) (map[string]interface{}, error) {
	// TODO: Реализовать массовое обогащение контрагентов
	// Это требует доступа к enrichmentFactory, который находится в Server
	return nil, apperrors.NewServiceUnavailableError("функция не реализована", nil)
}

// DeleteCounterpartyDuplicateGroup удаляет всех контрагентов из группы дубликатов
// groupID - это tax_id или bin, по которому группируются дубликаты
func (cs *CounterpartyService) DeleteCounterpartyDuplicateGroup(projectID int, groupID string) error {
	// Получаем всех контрагентов проекта
	counterparties, _, err := cs.serviceDB.GetNormalizedCounterparties(projectID, 0, 10000, "", "", "")
	if err != nil {
		return apperrors.NewInternalError("не удалось получить контрагентов", err)
	}

	// Находим всех контрагентов из группы
	var idsToDelete []int
	for _, cp := range counterparties {
		key := cp.TaxID
		if key == "" {
			key = cp.BIN
		}
		if key == groupID {
			idsToDelete = append(idsToDelete, cp.ID)
		}
	}

	if len(idsToDelete) == 0 {
		return apperrors.NewNotFoundError("группа дубликатов не найдена", nil)
	}

	// Удаляем всех контрагентов из группы
	for _, id := range idsToDelete {
		if err := cs.serviceDB.DeleteNormalizedCounterparty(id); err != nil {
			// Логируем ошибку, но продолжаем удаление остальных
			continue
		}
	}

	return nil
}

// ResolveCounterpartyDuplicateGroup разрешает группу дубликатов, объединяя их в одного контрагента
// groupID - это tax_id или bin, по которому группируются дубликаты
// Выбирает контрагента с лучшим quality_score как мастер-контрагента
func (cs *CounterpartyService) ResolveCounterpartyDuplicateGroup(projectID int, groupID string) (*database.NormalizedCounterparty, error) {
	// Получаем всех контрагентов проекта
	counterparties, _, err := cs.serviceDB.GetNormalizedCounterparties(projectID, 0, 10000, "", "", "")
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить контрагентов", err)
	}

	// Находим всех контрагентов из группы
	var groupCounterparties []*database.NormalizedCounterparty
	for _, cp := range counterparties {
		key := cp.TaxID
		if key == "" {
			key = cp.BIN
		}
		if key == groupID {
			groupCounterparties = append(groupCounterparties, cp)
		}
	}

	if len(groupCounterparties) == 0 {
		return nil, apperrors.NewNotFoundError("группа дубликатов не найдена", nil)
	}

	if len(groupCounterparties) == 1 {
		// Если только один контрагент, возвращаем его
		return groupCounterparties[0], nil
	}

	// Выбираем мастер-контрагента (с лучшим quality_score или первый, если все равны)
	masterIndex := 0
	masterScore := groupCounterparties[0].QualityScore
	for i, cp := range groupCounterparties {
		if cp.QualityScore > masterScore {
			masterIndex = i
			masterScore = cp.QualityScore
		}
	}

	master := groupCounterparties[masterIndex]
	
	// Собираем ID всех дубликатов (кроме мастера)
	mergeIDs := make([]int, 0, len(groupCounterparties)-1)
	for i, cp := range groupCounterparties {
		if i != masterIndex {
			mergeIDs = append(mergeIDs, cp.ID)
		}
	}

	// Объединяем дубликаты в мастер-контрагента
	return cs.MergeCounterpartyDuplicates(master.ID, mergeIDs)
}