package services

import (
	"httpserver/database"
)

// SnapshotService сервис для работы со срезами данных
type SnapshotService struct {
	db *database.DB
}

// NewSnapshotService создает новый сервис срезов
func NewSnapshotService(db *database.DB) *SnapshotService {
	return &SnapshotService{
		db: db,
	}
}

// GetAllSnapshots получает список всех срезов
func (ss *SnapshotService) GetAllSnapshots() ([]*database.DataSnapshot, error) {
	return ss.db.GetAllSnapshots()
}

// GetSnapshotWithUploads получает срез с выгрузками
func (ss *SnapshotService) GetSnapshotWithUploads(snapshotID int) (*database.DataSnapshot, []*database.Upload, error) {
	return ss.db.GetSnapshotWithUploads(snapshotID)
}

// GetSnapshotsByProject получает все срезы проекта
func (ss *SnapshotService) GetSnapshotsByProject(projectID int) ([]*database.DataSnapshot, error) {
	return ss.db.GetSnapshotsByProject(projectID)
}

// CreateSnapshot создает новый срез
func (ss *SnapshotService) CreateSnapshot(snapshot *database.DataSnapshot, uploads []database.SnapshotUpload) (*database.DataSnapshot, error) {
	return ss.db.CreateSnapshot(snapshot, uploads)
}

// DeleteSnapshot удаляет срез
func (ss *SnapshotService) DeleteSnapshot(snapshotID int) error {
	return ss.db.DeleteSnapshot(snapshotID)
}

// NormalizeSnapshot выполняет нормализацию среза
// Принимает функцию нормализации от Server
func (ss *SnapshotService) NormalizeSnapshot(snapshotID int, normalizeFunc func(int, interface{}) (interface{}, error), req interface{}) (interface{}, error) {
	return normalizeFunc(snapshotID, req)
}

// CompareSnapshotIterations сравнивает итерации среза
// Принимает функцию сравнения от Server
func (ss *SnapshotService) CompareSnapshotIterations(snapshotID int, compareFunc func(int) (interface{}, error)) (interface{}, error) {
	return compareFunc(snapshotID)
}

// CalculateSnapshotMetrics вычисляет метрики среза
// Принимает функцию вычисления от Server
func (ss *SnapshotService) CalculateSnapshotMetrics(snapshotID int, calculateFunc func(int) (interface{}, error)) (interface{}, error) {
	return calculateFunc(snapshotID)
}

// GetSnapshotEvolution получает эволюцию номенклатуры среза
// Принимает функцию получения эволюции от Server
func (ss *SnapshotService) GetSnapshotEvolution(snapshotID int, evolutionFunc func(int) (interface{}, error)) (interface{}, error) {
	return evolutionFunc(snapshotID)
}

// CreateAutoSnapshot создает срез автоматически по критериям
// Принимает функцию создания от Server
func (ss *SnapshotService) CreateAutoSnapshot(projectID int, uploadsPerDatabase int, name, description string, createFunc func(int, int, string, string) (*database.DataSnapshot, error)) (*database.DataSnapshot, error) {
	return createFunc(projectID, uploadsPerDatabase, name, description)
}

