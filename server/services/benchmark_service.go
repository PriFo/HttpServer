package services

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"httpserver/database"
	apperrors "httpserver/server/errors"
	"httpserver/server/models"

	"github.com/google/uuid"
)

// BenchmarkService сервис для управления эталонами
type BenchmarkService struct {
	benchmarksDB *database.BenchmarksDB
	db           *database.DB
	serviceDB    *database.ServiceDB
}

// NewBenchmarkService создает новый сервис эталонов
func NewBenchmarkService(
	benchmarksDB *database.BenchmarksDB,
	db *database.DB,
	serviceDB *database.ServiceDB,
) *BenchmarkService {
	return &BenchmarkService{
		benchmarksDB: benchmarksDB,
		db:           db,
		serviceDB:    serviceDB,
	}
}

// CreateFromUpload создает эталон из выбранных элементов загрузки
func (bs *BenchmarkService) CreateFromUpload(uploadID string, itemIDs []string, entityType string) (*models.Benchmark, error) {
	// 1. Получить информацию о загрузке
	upload, err := bs.db.GetUploadByUUID(uploadID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("загрузка не найдена", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить информацию о загрузке", err)
	}

	// 2. Получить элементы из catalog_items по их ID
	items, err := bs.getCatalogItemsByIDs(itemIDs)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить элементы каталога", err)
	}

	if len(items) == 0 {
		return nil, apperrors.NewValidationError("элементы каталога не найдены", nil)
	}

	// 3. Сформировать каноническое имя и данные
	// Используем имя первого элемента как основное
	canonicalName := items[0].Name
	if canonicalName == "" {
		canonicalName = items[0].Code
	}

	// Собираем данные из всех элементов
	data := make(map[string]interface{})
	variations := []string{canonicalName}

	// Добавляем уникальные имена как вариации
	nameSet := make(map[string]bool)
	nameSet[canonicalName] = true

	for _, item := range items {
		if item.Name != "" && !nameSet[item.Name] {
			variations = append(variations, item.Name)
			nameSet[item.Name] = true
		}
		if item.Code != "" {
			data["code"] = item.Code
		}
		if item.Reference != "" {
			data["reference"] = item.Reference
		}
		// Парсим attributes_xml если есть
		if item.Attributes != "" {
			var attrs map[string]interface{}
			if err := json.Unmarshal([]byte(item.Attributes), &attrs); err == nil {
				for k, v := range attrs {
					data[k] = v
				}
			}
		}
	}

	// 4. Создать запись в таблице benchmarks
	benchmark := &models.Benchmark{
		ID:             uuid.New().String(),
		EntityType:     entityType,
		Name:           canonicalName,
		Data:           data,
		SourceUploadID: &upload.ID,
		IsActive:       true,
		Variations:     variations,
	}

	if upload.ClientID != nil {
		benchmark.SourceClientID = upload.ClientID
	}

	// Конвертируем в database.Benchmark
	dbBenchmark := &database.Benchmark{
		ID:             benchmark.ID,
		EntityType:     benchmark.EntityType,
		Name:           benchmark.Name,
		Data:           benchmark.Data,
		SourceUploadID: benchmark.SourceUploadID,
		SourceClientID: benchmark.SourceClientID,
		IsActive:       benchmark.IsActive,
		Variations:     benchmark.Variations,
	}

	if err := bs.benchmarksDB.CreateBenchmark(dbBenchmark); err != nil {
		return nil, apperrors.NewInternalError("не удалось создать эталон", err)
	}

	return benchmark, nil
}

// getCatalogItemsByIDs получает элементы каталога по их ID
func (bs *BenchmarkService) getCatalogItemsByIDs(itemIDs []string) ([]*database.CatalogItem, error) {
	if len(itemIDs) == 0 {
		return nil, apperrors.NewValidationError("не указаны ID элементов", nil)
	}

	// Строим запрос с IN clause
	placeholders := make([]string, len(itemIDs))
	args := make([]interface{}, len(itemIDs))
	for i, id := range itemIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT ci.id, ci.catalog_id, c.name as catalog_name,
		       ci.reference, ci.code, ci.name,
		       COALESCE(ci.attributes_xml, '') as attributes,
		       COALESCE(ci.table_parts_xml, '') as table_parts,
		       ci.created_at
		FROM catalog_items ci
		LEFT JOIN catalogs c ON ci.catalog_id = c.id
		WHERE ci.id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := bs.db.GetConnection().Query(query, args...)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось выполнить запрос элементов каталога", err)
	}
	defer rows.Close()

	var items []*database.CatalogItem
	for rows.Next() {
		item := &database.CatalogItem{}
		var catalogName sql.NullString
		err := rows.Scan(
			&item.ID, &item.CatalogID, &catalogName, &item.Reference, &item.Code, &item.Name,
			&item.Attributes, &item.TableParts, &item.CreatedAt,
		)
		if err != nil {
			return nil, apperrors.NewInternalError("не удалось прочитать элемент каталога", err)
		}
		if catalogName.Valid {
			item.CatalogName = catalogName.String
		}
		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		return nil, apperrors.NewInternalError("ошибка при итерации элементов каталога", err)
	}

	return items, nil
}

// FindBestMatch ищет лучший эталон для данного имени
func (bs *BenchmarkService) FindBestMatch(name string, entityType string) (*models.Benchmark, error) {
	dbBenchmark, err := bs.benchmarksDB.FindBestMatch(name, entityType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("эталон не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось найти эталон", err)
	}

	if dbBenchmark == nil {
		return nil, nil
	}

	// Конвертируем в models.Benchmark
	benchmark := &models.Benchmark{
		ID:             dbBenchmark.ID,
		EntityType:     dbBenchmark.EntityType,
		Name:           dbBenchmark.Name,
		Data:           dbBenchmark.Data,
		SourceUploadID: dbBenchmark.SourceUploadID,
		SourceClientID: dbBenchmark.SourceClientID,
		IsActive:       dbBenchmark.IsActive,
		CreatedAt:      dbBenchmark.CreatedAt,
		UpdatedAt:      dbBenchmark.UpdatedAt,
		Variations:     dbBenchmark.Variations,
	}

	return benchmark, nil
}

// GetByType возвращает все активные эталоны для заданного типа
func (bs *BenchmarkService) GetByType(entityType string) ([]*models.Benchmark, error) {
	dbBenchmarks, err := bs.benchmarksDB.ListBenchmarks(entityType, true, 1000, 0)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить эталоны по типу", err)
	}

	benchmarks := make([]*models.Benchmark, len(dbBenchmarks))
	for i, dbBenchmark := range dbBenchmarks {
		benchmarks[i] = &models.Benchmark{
			ID:             dbBenchmark.ID,
			EntityType:     dbBenchmark.EntityType,
			Name:           dbBenchmark.Name,
			Data:           dbBenchmark.Data,
			SourceUploadID: dbBenchmark.SourceUploadID,
			SourceClientID: dbBenchmark.SourceClientID,
			IsActive:       dbBenchmark.IsActive,
			CreatedAt:      dbBenchmark.CreatedAt,
			UpdatedAt:      dbBenchmark.UpdatedAt,
			Variations:     dbBenchmark.Variations,
		}
	}

	return benchmarks, nil
}

// GetByID получает эталон по ID
func (bs *BenchmarkService) GetByID(id string) (*models.Benchmark, error) {
	dbBenchmark, err := bs.benchmarksDB.GetBenchmark(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperrors.NewNotFoundError("эталон не найден", err)
		}
		return nil, apperrors.NewInternalError("не удалось получить эталон", err)
	}

	benchmark := &models.Benchmark{
		ID:             dbBenchmark.ID,
		EntityType:     dbBenchmark.EntityType,
		Name:           dbBenchmark.Name,
		Data:           dbBenchmark.Data,
		SourceUploadID: dbBenchmark.SourceUploadID,
		SourceClientID: dbBenchmark.SourceClientID,
		IsActive:       dbBenchmark.IsActive,
		CreatedAt:      dbBenchmark.CreatedAt,
		UpdatedAt:      dbBenchmark.UpdatedAt,
		Variations:     dbBenchmark.Variations,
	}

	return benchmark, nil
}

// List возвращает список эталонов с фильтрацией
func (bs *BenchmarkService) List(entityType string, activeOnly bool, limit, offset int) (*models.BenchmarkListResponse, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	dbBenchmarks, err := bs.benchmarksDB.ListBenchmarks(entityType, activeOnly, limit, offset)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось получить список эталонов", err)
	}

	total, err := bs.benchmarksDB.CountBenchmarks(entityType, activeOnly)
	if err != nil {
		return nil, apperrors.NewInternalError("не удалось подсчитать количество эталонов", err)
	}

	benchmarks := make([]*models.Benchmark, len(dbBenchmarks))
	for i, dbBenchmark := range dbBenchmarks {
		benchmarks[i] = &models.Benchmark{
			ID:             dbBenchmark.ID,
			EntityType:     dbBenchmark.EntityType,
			Name:           dbBenchmark.Name,
			Data:           dbBenchmark.Data,
			SourceUploadID: dbBenchmark.SourceUploadID,
			SourceClientID: dbBenchmark.SourceClientID,
			IsActive:       dbBenchmark.IsActive,
			CreatedAt:      dbBenchmark.CreatedAt,
			UpdatedAt:      dbBenchmark.UpdatedAt,
			Variations:     dbBenchmark.Variations,
		}
	}

	return &models.BenchmarkListResponse{
		Benchmarks: benchmarks,
		Total:      total,
		Limit:      limit,
		Offset:     offset,
	}, nil
}

// Update обновляет эталон
func (bs *BenchmarkService) Update(benchmark *models.Benchmark) error {
	// Получаем существующий эталон
	dbBenchmark, err := bs.benchmarksDB.GetBenchmark(benchmark.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperrors.NewNotFoundError("эталон не найден", err)
		}
		return apperrors.NewInternalError("не удалось получить эталон", err)
	}

	// Обновляем поля
	if benchmark.EntityType != "" {
		dbBenchmark.EntityType = benchmark.EntityType
	}
	if benchmark.Name != "" {
		dbBenchmark.Name = benchmark.Name
	}
	if benchmark.Data != nil {
		dbBenchmark.Data = benchmark.Data
	}
	if benchmark.IsActive != dbBenchmark.IsActive {
		dbBenchmark.IsActive = benchmark.IsActive
	}
	if benchmark.Variations != nil {
		dbBenchmark.Variations = benchmark.Variations
	}

	// Сохраняем
	if err := bs.benchmarksDB.UpdateBenchmark(dbBenchmark); err != nil {
		return apperrors.NewInternalError("не удалось обновить эталон", err)
	}

	return nil
}

// Delete удаляет эталон (мягкое удаление)
func (bs *BenchmarkService) Delete(id string) error {
	return bs.benchmarksDB.DeleteBenchmark(id)
}

// Create создает новый эталон
func (bs *BenchmarkService) Create(req *models.CreateBenchmarkRequest) (*models.Benchmark, error) {
	benchmark := &models.Benchmark{
		ID:             uuid.New().String(),
		EntityType:     req.EntityType,
		Name:           req.Name,
		Data:           req.Data,
		SourceUploadID: req.SourceUploadID,
		SourceClientID: req.SourceClientID,
		IsActive:       true,
		Variations:     req.Variations,
	}

	if benchmark.Data == nil {
		benchmark.Data = make(map[string]interface{})
	}

	// Конвертируем в database.Benchmark
	dbBenchmark := &database.Benchmark{
		ID:             benchmark.ID,
		EntityType:     benchmark.EntityType,
		Name:           benchmark.Name,
		Data:           benchmark.Data,
		SourceUploadID: benchmark.SourceUploadID,
		SourceClientID: benchmark.SourceClientID,
		IsActive:       benchmark.IsActive,
		Variations:     benchmark.Variations,
	}

	if err := bs.benchmarksDB.CreateBenchmark(dbBenchmark); err != nil {
		return nil, apperrors.NewInternalError("не удалось создать эталон", err)
	}

	return benchmark, nil
}
