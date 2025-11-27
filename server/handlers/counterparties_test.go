package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"httpserver/database"
)

// mockCounterpartyService мок для CounterpartyService
type mockCounterpartyService struct {
	getServiceDBFunc                        func() *database.ServiceDB
	getNormalizedCounterpartyStatsFunc      func(projectID int) (map[string]interface{}, error)
	getNormalizedCounterpartyFunc           func(id int) (*database.NormalizedCounterparty, error)
	updateNormalizedCounterpartyFunc        func(id int, normalizedName, taxID, kpp, bin, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm, bankName, bankAccount, correspondentAccount, bik string, qualityScore float64, sourceEnrichment, subcategory string) error
	getCounterpartyDuplicatesFunc           func(projectID int) ([]map[string]interface{}, error)
	mergeCounterpartyDuplicatesFunc         func(masterID int, mergeIDs []int) (*database.NormalizedCounterparty, error)
	getNormalizedCounterpartiesByClientFunc func(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error)
	getNormalizedCounterpartiesFunc         func(projectID int, limit, offset int, search, taxID, bin string) ([]*database.NormalizedCounterparty, int, error)
	getClientProjectFunc                    func(projectID int) (*database.ClientProject, error)
	getAllCounterpartiesByClientFunc        func(clientID int, projectID *int, offset, limit int, search, source, sortBy, order string, minQuality, maxQuality *float64) (*database.GetAllCounterpartiesByClientResult, error)
	bulkUpdateCounterpartiesFunc            func(ids []int, updates map[string]interface{}) (map[string]interface{}, error)
	bulkDeleteCounterpartiesFunc            func(ids []int) (map[string]interface{}, error)
	deleteCounterpartyDuplicateGroupFunc    func(projectID int, groupID string) error
	resolveCounterpartyDuplicateGroupFunc   func(projectID int, groupID string) (*database.NormalizedCounterparty, error)
}

func (m *mockCounterpartyService) GetServiceDB() *database.ServiceDB {
	if m.getServiceDBFunc != nil {
		return m.getServiceDBFunc()
	}
	return nil
}

func (m *mockCounterpartyService) GetNormalizedCounterpartyStats(projectID int) (map[string]interface{}, error) {
	if m.getNormalizedCounterpartyStatsFunc != nil {
		return m.getNormalizedCounterpartyStatsFunc(projectID)
	}
	return map[string]interface{}{}, nil
}

func (m *mockCounterpartyService) GetNormalizedCounterparty(id int) (*database.NormalizedCounterparty, error) {
	if m.getNormalizedCounterpartyFunc != nil {
		return m.getNormalizedCounterpartyFunc(id)
	}
	return nil, nil
}

func (m *mockCounterpartyService) UpdateNormalizedCounterparty(id int, normalizedName, taxID, kpp, bin, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm, bankName, bankAccount, correspondentAccount, bik string, qualityScore float64, sourceEnrichment, subcategory string) error {
	if m.updateNormalizedCounterpartyFunc != nil {
		return m.updateNormalizedCounterpartyFunc(id, normalizedName, taxID, kpp, bin, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm, bankName, bankAccount, correspondentAccount, bik, qualityScore, sourceEnrichment, subcategory)
	}
	return nil
}

func (m *mockCounterpartyService) GetCounterpartyDuplicates(projectID int) ([]map[string]interface{}, error) {
	if m.getCounterpartyDuplicatesFunc != nil {
		return m.getCounterpartyDuplicatesFunc(projectID)
	}
	return []map[string]interface{}{}, nil
}

func (m *mockCounterpartyService) MergeCounterpartyDuplicates(masterID int, mergeIDs []int) (*database.NormalizedCounterparty, error) {
	if m.mergeCounterpartyDuplicatesFunc != nil {
		return m.mergeCounterpartyDuplicatesFunc(masterID, mergeIDs)
	}
	return nil, nil
}

func (m *mockCounterpartyService) GetNormalizedCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error) {
	if m.getNormalizedCounterpartiesByClientFunc != nil {
		return m.getNormalizedCounterpartiesByClientFunc(clientID, projectID, offset, limit, search, enrichment, subcategory)
	}
	return nil, nil, 0, nil
}

func (m *mockCounterpartyService) GetNormalizedCounterparties(projectID int, limit, offset int, search, taxID, bin string) ([]*database.NormalizedCounterparty, int, error) {
	if m.getNormalizedCounterpartiesFunc != nil {
		return m.getNormalizedCounterpartiesFunc(projectID, limit, offset, search, taxID, bin)
	}
	return nil, 0, nil
}

func (m *mockCounterpartyService) GetClientProject(projectID int) (*database.ClientProject, error) {
	if m.getClientProjectFunc != nil {
		return m.getClientProjectFunc(projectID)
	}
	return nil, nil
}

func (m *mockCounterpartyService) GetAllCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, source, sortBy, order string, minQuality, maxQuality *float64) (*database.GetAllCounterpartiesByClientResult, error) {
	if m.getAllCounterpartiesByClientFunc != nil {
		return m.getAllCounterpartiesByClientFunc(clientID, projectID, offset, limit, search, source, sortBy, order, minQuality, maxQuality)
	}
	return nil, nil
}

func (m *mockCounterpartyService) BulkUpdateCounterparties(ids []int, updates map[string]interface{}) (map[string]interface{}, error) {
	if m.bulkUpdateCounterpartiesFunc != nil {
		return m.bulkUpdateCounterpartiesFunc(ids, updates)
	}
	return map[string]interface{}{}, nil
}

func (m *mockCounterpartyService) BulkDeleteCounterparties(ids []int) (map[string]interface{}, error) {
	if m.bulkDeleteCounterpartiesFunc != nil {
		return m.bulkDeleteCounterpartiesFunc(ids)
	}
	return map[string]interface{}{}, nil
}

func (m *mockCounterpartyService) DeleteCounterpartyDuplicateGroup(projectID int, groupID string) error {
	if m.deleteCounterpartyDuplicateGroupFunc != nil {
		return m.deleteCounterpartyDuplicateGroupFunc(projectID, groupID)
	}
	return nil
}

func (m *mockCounterpartyService) ResolveCounterpartyDuplicateGroup(projectID int, groupID string) (*database.NormalizedCounterparty, error) {
	if m.resolveCounterpartyDuplicateGroupFunc != nil {
		return m.resolveCounterpartyDuplicateGroupFunc(projectID, groupID)
	}
	return nil, nil
}

// setupTestHandler создает тестовый обработчик
func setupTestCounterpartyHandler(svc CounterpartyService) *CounterpartyHandler {
	baseHandler := NewBaseHandler(
		func(w http.ResponseWriter, r *http.Request, data interface{}, statusCode int) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(data)
		},
		func(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": message})
		},
		func(w http.ResponseWriter, r *http.Request, err error) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		},
	)

	logFunc := func(entry interface{}) {
		// Мок логирования
	}

	return NewCounterpartyHandler(baseHandler, svc, logFunc)
}

// TestHandleNormalizedCounterparties_ByClientID тестирует получение по client_id
func TestHandleNormalizedCounterparties_ByClientID(t *testing.T) {
	mockService := &mockCounterpartyService{
		getNormalizedCounterpartiesByClientFunc: func(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error) {
			if clientID != 1 {
				t.Errorf("Expected clientID=1, got %d", clientID)
			}
			return []*database.NormalizedCounterparty{
					{
						ID:              1,
						NormalizedName:  "Test Counterparty",
						ClientProjectID: 1,
					},
				}, []*database.ClientProject{
					{
						ID:   1,
						Name: "Test Project",
					},
				}, 1, nil
		},
	}

	handler := setupTestCounterpartyHandler(mockService)

	req := httptest.NewRequest("GET", "/api/counterparties/normalized?client_id=1&page=1&limit=20", nil)
	w := httptest.NewRecorder()

	handler.HandleNormalizedCounterparties(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["total"] != float64(1) {
		t.Errorf("Expected total=1, got %v", response["total"])
	}

	counterparties, ok := response["counterparties"].([]interface{})
	if !ok || len(counterparties) != 1 {
		t.Errorf("Expected 1 counterparty, got %v", counterparties)
	}
}

// TestHandleNormalizedCounterparties_ByProjectID тестирует получение по project_id
func TestHandleNormalizedCounterparties_ByProjectID(t *testing.T) {
	mockService := &mockCounterpartyService{
		getNormalizedCounterpartiesFunc: func(projectID int, limit, offset int, search, taxID, bin string) ([]*database.NormalizedCounterparty, int, error) {
			if projectID != 1 {
				t.Errorf("Expected projectID=1, got %d", projectID)
			}
			return []*database.NormalizedCounterparty{
				{
					ID:              1,
					NormalizedName:  "Test Counterparty",
					ClientProjectID: 1,
				},
			}, 1, nil
		},
		getClientProjectFunc: func(projectID int) (*database.ClientProject, error) {
			if projectID != 1 {
				t.Errorf("Expected projectID=1, got %d", projectID)
			}
			return &database.ClientProject{
				ID:   1,
				Name: "Test Project",
			}, nil
		},
	}

	handler := setupTestCounterpartyHandler(mockService)

	req := httptest.NewRequest("GET", "/api/counterparties/normalized?project_id=1&page=1&limit=20", nil)
	w := httptest.NewRecorder()

	handler.HandleNormalizedCounterparties(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["total"] != float64(1) {
		t.Errorf("Expected total=1, got %v", response["total"])
	}
}

// TestHandleNormalizedCounterparties_MissingParams тестирует отсутствие обязательных параметров
func TestHandleNormalizedCounterparties_MissingParams(t *testing.T) {
	handler := setupTestCounterpartyHandler(&mockCounterpartyService{})

	req := httptest.NewRequest("GET", "/api/counterparties/normalized", nil)
	w := httptest.NewRecorder()

	handler.HandleNormalizedCounterparties(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestHandleNormalizedCounterparties_InvalidMethod тестирует неверный HTTP метод
func TestHandleNormalizedCounterparties_InvalidMethod(t *testing.T) {
	handler := setupTestCounterpartyHandler(&mockCounterpartyService{})

	req := httptest.NewRequest("POST", "/api/counterparties/normalized?client_id=1", nil)
	w := httptest.NewRecorder()

	handler.HandleNormalizedCounterparties(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestHandleNormalizedCounterparties_WithSearch тестирует поиск
func TestHandleNormalizedCounterparties_WithSearch(t *testing.T) {
	mockService := &mockCounterpartyService{
		getNormalizedCounterpartiesByClientFunc: func(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error) {
			if search != "test" {
				t.Errorf("Expected search='test', got '%s'", search)
			}
			return []*database.NormalizedCounterparty{}, []*database.ClientProject{}, 0, nil
		},
	}

	handler := setupTestCounterpartyHandler(mockService)

	req := httptest.NewRequest("GET", "/api/counterparties/normalized?client_id=1&search=test", nil)
	w := httptest.NewRecorder()

	handler.HandleNormalizedCounterparties(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

// TestHandleNormalizedCounterparties_WithPagination тестирует пагинацию
func TestHandleNormalizedCounterparties_WithPagination(t *testing.T) {
	mockService := &mockCounterpartyService{
		getNormalizedCounterpartiesByClientFunc: func(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*database.NormalizedCounterparty, []*database.ClientProject, int, error) {
			if offset != 20 {
				t.Errorf("Expected offset=20, got %d", offset)
			}
			if limit != 10 {
				t.Errorf("Expected limit=10, got %d", limit)
			}
			return []*database.NormalizedCounterparty{}, []*database.ClientProject{}, 0, nil
		},
	}

	handler := setupTestCounterpartyHandler(mockService)

	req := httptest.NewRequest("GET", "/api/counterparties/normalized?client_id=1&page=3&limit=10", nil)
	w := httptest.NewRecorder()

	handler.HandleNormalizedCounterparties(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}
