package normalization

import (
	"strings"
	"testing"

	"httpserver/database"
)

// setupTestServiceDB создает тестовую ServiceDB
func setupTestServiceDBForDuplicate(t *testing.T) *database.ServiceDB {
	serviceDB, err := database.NewServiceDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test service DB: %v", err)
	}
	return serviceDB
}

// createTestCounterpartyItem создает тестовый элемент контрагента
func createTestCounterpartyItem(id int, name, inn, kpp, bin string) *database.CatalogItem {
	attributes := ""
	if inn != "" {
		attributes += `<ИНН>` + inn + `</ИНН>`
	}
	if kpp != "" {
		attributes += `<КПП>` + kpp + `</КПП>`
	}
	if bin != "" {
		attributes += `<БИН>` + bin + `</БИН>`
	}

	return &database.CatalogItem{
		ID:         id,
		Reference:  "ref_" + name,
		Code:       "code_" + name,
		Name:       name,
		Attributes: attributes,
	}
}

// TestNewCounterpartyDuplicateAnalyzer проверяет создание анализатора
func TestNewCounterpartyDuplicateAnalyzer(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	if analyzer == nil {
		t.Error("NewCounterpartyDuplicateAnalyzer returned nil")
	}
}

// TestGroupByINNKPP_Basic проверяет базовую группировку по ИНН/КПП
func TestGroupByINNKPP_Basic(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567891", "123456789", ""),
	}

	groups := analyzer.groupByINNKPP(counterparties)

	if len(groups) == 0 {
		t.Error("Expected to find duplicate groups")
	}

	// Должна быть найдена группа с двумя элементами (ID 1 и 2)
	found := false
	for _, group := range groups {
		if len(group.Items) >= 2 {
			found = true
			if group.KeyType != "inn_kpp" {
				t.Errorf("Expected KeyType 'inn_kpp', got '%s'", group.KeyType)
			}
			if group.Confidence != 1.0 {
				t.Errorf("Expected Confidence 1.0, got %f", group.Confidence)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find group with at least 2 items")
	}
}

// TestGroupByINNKPP_WithKPP проверяет группировку с КПП
func TestGroupByINNKPP_WithKPP(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567890", "987654321", ""), // Другой КПП
	}

	groups := analyzer.groupByINNKPP(counterparties)

	// Должны быть две группы: одна с ИНН/КПП 1234567890/123456789 (2 элемента), другая с 1234567890/987654321 (1 элемент, не дубликат)
	groupsWithDuplicates := 0
	for _, group := range groups {
		if len(group.Items) >= 2 {
			groupsWithDuplicates++
		}
	}

	if groupsWithDuplicates == 0 {
		t.Error("Expected to find at least one group with duplicates")
	}
}

// TestGroupByINNKPP_WithoutKPP проверяет группировку только по ИНН
func TestGroupByINNKPP_WithoutKPP(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567891", "", ""),
	}

	groups := analyzer.groupByINNKPP(counterparties)

	if len(groups) == 0 {
		t.Error("Expected to find duplicate groups")
	}

	// Должна быть найдена группа с двумя элементами
	found := false
	for _, group := range groups {
		if len(group.Items) >= 2 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find group with at least 2 items")
	}
}

// TestGroupByINNKPP_NoDuplicates проверяет случай без дублей
func TestGroupByINNKPP_NoDuplicates(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567891", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567892", "123456789", ""),
	}

	groups := analyzer.groupByINNKPP(counterparties)

	if len(groups) > 0 {
		t.Errorf("Expected no duplicate groups, got %d", len(groups))
	}
}

// TestGroupByINNKPP_EmptyINN проверяет обработку пустого ИНН
func TestGroupByINNKPP_EmptyINN(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "", "123456789", ""),
	}

	groups := analyzer.groupByINNKPP(counterparties)

	// Элементы без ИНН должны быть пропущены
	if len(groups) > 0 {
		t.Errorf("Expected no groups for items without INN, got %d", len(groups))
	}
}

// TestGroupByBIN_Basic проверяет базовую группировку по БИН
func TestGroupByBIN_Basic(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ТОО Тест 1", "", "", "123456789012"),
		createTestCounterpartyItem(2, "ТОО Тест 2", "", "", "123456789012"),
		createTestCounterpartyItem(3, "ТОО Тест 3", "", "", "123456789013"),
	}

	groups := analyzer.groupByBIN(counterparties)

	if len(groups) == 0 {
		t.Error("Expected to find duplicate groups")
	}

	// Должна быть найдена группа с двумя элементами
	found := false
	for _, group := range groups {
		if len(group.Items) >= 2 {
			found = true
			if !strings.Contains(group.KeyType, "bin") {
				t.Errorf("Expected KeyType to contain 'bin', got '%s'", group.KeyType)
			}
			if group.Confidence != 1.0 {
				t.Errorf("Expected Confidence 1.0, got %f", group.Confidence)
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find group with at least 2 items")
	}
}

// TestGroupByBIN_NoDuplicates проверяет случай без дублей по БИН
func TestGroupByBIN_NoDuplicates(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ТОО Тест 1", "", "", "123456789012"),
		createTestCounterpartyItem(2, "ТОО Тест 2", "", "", "123456789013"),
		createTestCounterpartyItem(3, "ТОО Тест 3", "", "", "123456789014"),
	}

	groups := analyzer.groupByBIN(counterparties)

	if len(groups) > 0 {
		t.Errorf("Expected no duplicate groups, got %d", len(groups))
	}
}

// TestGroupByBIN_EmptyBIN проверяет обработку пустого БИН
func TestGroupByBIN_EmptyBIN(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ТОО Тест 1", "", "", ""),
		createTestCounterpartyItem(2, "ТОО Тест 2", "", "", ""),
	}

	groups := analyzer.groupByBIN(counterparties)

	// Элементы без БИН должны быть пропущены
	if len(groups) > 0 {
		t.Errorf("Expected no groups for items without BIN, got %d", len(groups))
	}
}

// TestMergeOverlappingGroups_NoOverlap проверяет случай без пересечений
func TestMergeOverlappingGroups_NoOverlap(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{
		{
			Key:        "1234567890",
			KeyType:    "inn_kpp",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 2}},
			Confidence: 1.0,
		},
		{
			Key:        "123456789012",
			KeyType:    "bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 3}, {ID: 4}},
			Confidence: 1.0,
		},
	}

	merged := analyzer.mergeOverlappingGroups(groups)

	if len(merged) != 2 {
		t.Errorf("Expected 2 groups after merge (no overlap), got %d", len(merged))
	}
}

// TestMergeOverlappingGroups_WithOverlap проверяет объединение пересекающихся групп
func TestMergeOverlappingGroups_WithOverlap(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	// Создаем группы с общим элементом (ID: 1)
	groups := []CounterpartyDuplicateGroup{
		{
			Key:        "1234567890",
			KeyType:    "inn_kpp",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 2}},
			Confidence: 1.0,
		},
		{
			Key:        "123456789012",
			KeyType:    "bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 3}},
			Confidence: 1.0,
		},
	}

	merged := analyzer.mergeOverlappingGroups(groups)

	// Должна быть одна объединенная группа
	if len(merged) != 1 {
		t.Errorf("Expected 1 merged group, got %d", len(merged))
	}

	if len(merged[0].Items) != 3 {
		t.Errorf("Expected 3 items in merged group, got %d", len(merged[0].Items))
	}
}

// TestMergeOverlappingGroups_Empty проверяет обработку пустого списка
func TestMergeOverlappingGroups_Empty(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{}

	merged := analyzer.mergeOverlappingGroups(groups)

	if len(merged) != 0 {
		t.Errorf("Expected empty result, got %d groups", len(merged))
	}
}

// TestHasCommonItems проверяет проверку общих элементов
func TestHasCommonItems(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items1 := []*CounterpartyDuplicateItem{
		{ID: 1}, {ID: 2}, {ID: 3},
	}
	items2 := []*CounterpartyDuplicateItem{
		{ID: 3}, {ID: 4}, {ID: 5},
	}
	items3 := []*CounterpartyDuplicateItem{
		{ID: 6}, {ID: 7}, {ID: 8},
	}

	if !analyzer.hasCommonItems(items1, items2) {
		t.Error("Expected items1 and items2 to have common items (ID: 3)")
	}

	if analyzer.hasCommonItems(items1, items3) {
		t.Error("Expected items1 and items3 to have no common items")
	}
}

// TestMergeItems проверяет объединение элементов
func TestMergeItems(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items1 := []*CounterpartyDuplicateItem{
		{ID: 1}, {ID: 2}, {ID: 3},
	}
	items2 := []*CounterpartyDuplicateItem{
		{ID: 3}, {ID: 4}, {ID: 5},
	}

	merged := analyzer.mergeItems(items1, items2)

	if len(merged) != 5 {
		t.Errorf("Expected 5 unique items, got %d", len(merged))
	}

	// Проверяем, что ID: 3 не дублируется
	idCount := 0
	for _, item := range merged {
		if item.ID == 3 {
			idCount++
		}
	}
	if idCount != 1 {
		t.Errorf("Expected ID 3 to appear once, got %d times", idCount)
	}
}

// TestSelectMasterRecord_SingleItem проверяет выбор master record для одного элемента
func TestSelectMasterRecord_SingleItem(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{ID: 1, Name: "ООО Тест", INN: "1234567890", QualityScore: 0.8},
	}

	master := analyzer.selectMasterRecord(items)

	if master == nil {
		t.Error("Expected master record, got nil")
	}
	if master.ID != 1 {
		t.Errorf("Expected master ID 1, got %d", master.ID)
	}
}

// TestSelectMasterRecord_ByQualityScore проверяет выбор по качеству
func TestSelectMasterRecord_ByQualityScore(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{ID: 1, Name: "ООО Тест 1", INN: "1234567890", QualityScore: 0.5},
		{ID: 2, Name: "ООО Тест 2", INN: "1234567890", QualityScore: 0.9},
		{ID: 3, Name: "ООО Тест 3", INN: "1234567890", QualityScore: 0.7},
	}

	master := analyzer.selectMasterRecord(items)

	if master == nil {
		t.Error("Expected master record, got nil")
	}
	if master.ID != 2 {
		t.Errorf("Expected master ID 2 (highest quality), got %d", master.ID)
	}
}

// TestSelectMasterRecord_ByCompleteness проверяет выбор по полноте данных
func TestSelectMasterRecord_ByCompleteness(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{ID: 1, Name: "ООО Тест 1", INN: "1234567890", KPP: "", LegalAddress: "", QualityScore: 0.5},
		{ID: 2, Name: "ООО Тест 2", INN: "1234567890", KPP: "123456789", LegalAddress: "г. Москва", QualityScore: 0.5},
		{ID: 3, Name: "ООО Тест 3", INN: "1234567890", KPP: "", LegalAddress: "", QualityScore: 0.5},
	}

	master := analyzer.selectMasterRecord(items)

	if master == nil {
		t.Error("Expected master record, got nil")
	}
	// Должен быть выбран элемент с наибольшей полнотой данных (ID: 2)
	if master.ID != 2 {
		t.Errorf("Expected master ID 2 (most complete), got %d", master.ID)
	}
}

// TestSelectMasterRecord_WithOPF проверяет выбор с ОПФ в названии
func TestSelectMasterRecord_WithOPF(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	items := []*CounterpartyDuplicateItem{
		{ID: 1, Name: "Тест Компания", INN: "1234567890", QualityScore: 0.5},
		{ID: 2, Name: "ООО Тест Компания", INN: "1234567890", QualityScore: 0.5},
		{ID: 3, Name: "Компания", INN: "1234567890", QualityScore: 0.5},
	}

	master := analyzer.selectMasterRecord(items)

	if master == nil {
		t.Error("Expected master record, got nil")
	}
	// Должен быть выбран элемент с ОПФ (ID: 2)
	if master.ID != 2 {
		t.Errorf("Expected master ID 2 (with OPF), got %d", master.ID)
	}
}

// TestCalculateMasterScore проверяет расчет оценки master record
func TestCalculateMasterScore(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	item := &CounterpartyDuplicateItem{
		ID:           1,
		Name:         "ООО Тестовая Компания",
		INN:          "1234567890",
		KPP:          "123456789",
		LegalAddress: "г. Москва, ул. Тестовая, д. 1",
		QualityScore: 0.9,
	}

	score := analyzer.calculateMasterScore(item)

	if score <= 0 {
		t.Error("Expected score > 0")
	}

	// Проверяем, что полные данные дают высокий балл
	if score < 50.0 {
		t.Errorf("Expected score >= 50.0 for complete data, got %f", score)
	}
}

// TestAnalyzeDuplicates_ByINNKPP проверяет анализ по ИНН/КПП
func TestAnalyzeDuplicates_ByINNKPP(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567891", "123456789", ""),
	}

	groups := analyzer.AnalyzeDuplicates(counterparties)

	if len(groups) == 0 {
		t.Error("Expected to find duplicate groups")
	}

	// Проверяем, что есть группа по ИНН/КПП
	found := false
	for _, group := range groups {
		if group.KeyType == "inn_kpp" && len(group.Items) >= 2 {
			found = true
			if group.MasterItem == nil {
				t.Error("Expected MasterItem to be set")
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find group by INN/KPP")
	}
}

// TestAnalyzeDuplicates_ByBIN проверяет анализ по БИН
func TestAnalyzeDuplicates_ByBIN(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ТОО Тест 1", "", "", "123456789012"),
		createTestCounterpartyItem(2, "ТОО Тест 2", "", "", "123456789012"),
		createTestCounterpartyItem(3, "ТОО Тест 3", "", "", "123456789013"),
	}

	groups := analyzer.AnalyzeDuplicates(counterparties)

	if len(groups) == 0 {
		t.Error("Expected to find duplicate groups")
	}

	// Проверяем, что есть группа по БИН
	found := false
	for _, group := range groups {
		if strings.Contains(group.KeyType, "bin") && len(group.Items) >= 2 {
			found = true
			if group.MasterItem == nil {
				t.Error("Expected MasterItem to be set")
			}
			break
		}
	}

	if !found {
		t.Error("Expected to find group by BIN")
	}
}

// TestAnalyzeDuplicates_Both проверяет анализ по обоим критериям
func TestAnalyzeDuplicates_Both(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "123456789", ""),
		createTestCounterpartyItem(3, "ТОО Тест 3", "", "", "123456789012"),
		createTestCounterpartyItem(4, "ТОО Тест 4", "", "", "123456789012"),
	}

	groups := analyzer.AnalyzeDuplicates(counterparties)

	if len(groups) < 2 {
		t.Errorf("Expected at least 2 groups (by INN/KPP and by BIN), got %d", len(groups))
	}
}

// TestAnalyzeDuplicates_NoDuplicates проверяет случай без дублей
func TestAnalyzeDuplicates_NoDuplicates(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparties := []*database.CatalogItem{
		createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", ""),
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567891", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567892", "123456789", ""),
	}

	groups := analyzer.AnalyzeDuplicates(counterparties)

	if len(groups) > 0 {
		t.Errorf("Expected no duplicate groups, got %d", len(groups))
	}
}

// TestAnalyzeDuplicates_EmptyList проверяет обработку пустого списка
func TestAnalyzeDuplicates_EmptyList(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := analyzer.AnalyzeDuplicates([]*database.CatalogItem{})

	if len(groups) != 0 {
		t.Errorf("Expected no groups for empty list, got %d", len(groups))
	}
}

// TestFindDuplicatesForCounterparty_ByINN проверяет поиск по ИНН
func TestFindDuplicatesForCounterparty_ByINN(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparty := createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", "")
	allCounterparties := []*database.CatalogItem{
		counterparty,
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567891", "123456789", ""),
	}

	duplicates := analyzer.FindDuplicatesForCounterparty(counterparty, allCounterparties)

	if len(duplicates) == 0 {
		t.Error("Expected to find duplicates")
	}

	// Проверяем, что сам элемент не включен в дубликаты
	for _, dup := range duplicates {
		if dup.ID == counterparty.ID {
			t.Error("Counterparty itself should not be in duplicates")
		}
	}
}

// TestFindDuplicatesForCounterparty_ByBIN проверяет поиск по БИН
func TestFindDuplicatesForCounterparty_ByBIN(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparty := createTestCounterpartyItem(1, "ТОО Тест 1", "", "", "123456789012")
	allCounterparties := []*database.CatalogItem{
		counterparty,
		createTestCounterpartyItem(2, "ТОО Тест 2", "", "", "123456789012"),
		createTestCounterpartyItem(3, "ТОО Тест 3", "", "", "123456789013"),
	}

	duplicates := analyzer.FindDuplicatesForCounterparty(counterparty, allCounterparties)

	if len(duplicates) == 0 {
		t.Error("Expected to find duplicates")
	}

	// Проверяем, что сам элемент не включен в дубликаты
	for _, dup := range duplicates {
		if dup.ID == counterparty.ID {
			t.Error("Counterparty itself should not be in duplicates")
		}
	}
}

// TestFindDuplicatesForCounterparty_NoDuplicates проверяет случай без дублей
func TestFindDuplicatesForCounterparty_NoDuplicates(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparty := createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", "")
	allCounterparties := []*database.CatalogItem{
		counterparty,
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567891", "123456789", ""),
		createTestCounterpartyItem(3, "ООО Тест 3", "1234567892", "123456789", ""),
	}

	duplicates := analyzer.FindDuplicatesForCounterparty(counterparty, allCounterparties)

	if len(duplicates) > 0 {
		t.Errorf("Expected no duplicates, got %d", len(duplicates))
	}
}

// TestFindDuplicatesForCounterparty_ExcludesSelf проверяет исключение самого элемента
func TestFindDuplicatesForCounterparty_ExcludesSelf(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	counterparty := createTestCounterpartyItem(1, "ООО Тест 1", "1234567890", "123456789", "")
	allCounterparties := []*database.CatalogItem{
		counterparty,
		createTestCounterpartyItem(2, "ООО Тест 2", "1234567890", "123456789", ""),
	}

	duplicates := analyzer.FindDuplicatesForCounterparty(counterparty, allCounterparties)

	// Проверяем, что сам элемент не включен
	for _, dup := range duplicates {
		if dup.ID == counterparty.ID {
			t.Error("Counterparty itself should not be in duplicates")
		}
	}

	// Должен быть найден только один дубликат (ID: 2)
	if len(duplicates) != 1 {
		t.Errorf("Expected 1 duplicate, got %d", len(duplicates))
	}
}

// TestGetDuplicateSummary_Basic проверяет базовую сводку
func TestGetDuplicateSummary_Basic(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{
		{
			Key:        "1234567890",
			KeyType:    "inn_kpp",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 2}},
			Confidence: 1.0,
		},
		{
			Key:        "123456789012",
			KeyType:    "bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 3}, {ID: 4}},
			Confidence: 1.0,
		},
	}

	summary := analyzer.GetDuplicateSummary(groups)

	if summary["total_groups"].(int) != 2 {
		t.Errorf("Expected total_groups 2, got %v", summary["total_groups"])
	}
	if summary["total_duplicates"].(int) != 4 {
		t.Errorf("Expected total_duplicates 4, got %v", summary["total_duplicates"])
	}
}

// TestGetDuplicateSummary_ByINNKPP проверяет сводку только по ИНН/КПП
func TestGetDuplicateSummary_ByINNKPP(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{
		{
			Key:        "1234567890",
			KeyType:    "inn_kpp",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 2}},
			Confidence: 1.0,
		},
		{
			Key:        "1234567891",
			KeyType:    "inn_kpp",
			Items:      []*CounterpartyDuplicateItem{{ID: 3}, {ID: 4}},
			Confidence: 1.0,
		},
	}

	summary := analyzer.GetDuplicateSummary(groups)

	if summary["duplicates_by_inn_kpp"].(int) != 2 {
		t.Errorf("Expected duplicates_by_inn_kpp 2, got %v", summary["duplicates_by_inn_kpp"])
	}
	if summary["duplicates_by_bin"].(int) != 0 {
		t.Errorf("Expected duplicates_by_bin 0, got %v", summary["duplicates_by_bin"])
	}
}

// TestGetDuplicateSummary_ByBIN проверяет сводку только по БИН
func TestGetDuplicateSummary_ByBIN(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{
		{
			Key:        "123456789012",
			KeyType:    "bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 2}},
			Confidence: 1.0,
		},
		{
			Key:        "123456789013",
			KeyType:    "bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 3}, {ID: 4}},
			Confidence: 1.0,
		},
	}

	summary := analyzer.GetDuplicateSummary(groups)

	if summary["duplicates_by_bin"].(int) != 2 {
		t.Errorf("Expected duplicates_by_bin 2, got %v", summary["duplicates_by_bin"])
	}
	if summary["duplicates_by_inn_kpp"].(int) != 0 {
		t.Errorf("Expected duplicates_by_inn_kpp 0, got %v", summary["duplicates_by_inn_kpp"])
	}
}

// TestGetDuplicateSummary_Both проверяет сводку по обоим типам
func TestGetDuplicateSummary_Both(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{
		{
			Key:        "1234567890",
			KeyType:    "inn_kpp",
			Items:      []*CounterpartyDuplicateItem{{ID: 1}, {ID: 2}},
			Confidence: 1.0,
		},
		{
			Key:        "123456789012",
			KeyType:    "bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 3}, {ID: 4}},
			Confidence: 1.0,
		},
		{
			Key:        "1234567890|123456789012",
			KeyType:    "inn_kpp+bin",
			Items:      []*CounterpartyDuplicateItem{{ID: 5}, {ID: 6}},
			Confidence: 1.0,
		},
	}

	summary := analyzer.GetDuplicateSummary(groups)

	if summary["duplicates_by_both"].(int) != 1 {
		t.Errorf("Expected duplicates_by_both 1, got %v", summary["duplicates_by_both"])
	}
}

// TestGetDuplicateSummary_Empty проверяет обработку пустого списка
func TestGetDuplicateSummary_Empty(t *testing.T) {
	analyzer := NewCounterpartyDuplicateAnalyzer()

	groups := []CounterpartyDuplicateGroup{}

	summary := analyzer.GetDuplicateSummary(groups)

	if summary["total_groups"].(int) != 0 {
		t.Errorf("Expected total_groups 0, got %v", summary["total_groups"])
	}
	if summary["total_duplicates"].(int) != 0 {
		t.Errorf("Expected total_duplicates 0, got %v", summary["total_duplicates"])
	}
}
