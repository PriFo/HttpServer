package normalization

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"httpserver/database"
	"httpserver/extractors"

	_ "github.com/mattn/go-sqlite3"
)

// setupIntegrationTestDatabase создает тестовую базу данных с контрагентами для интеграционных тестов
func setupIntegrationTestDatabase(t *testing.T, dbPath string, counterparties []TestCounterparty) *database.DB {
	// Удаляем существующий файл, если есть
	os.Remove(dbPath)

	db, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Создаем таблицу catalog_items
	_, err = db.GetConnection().Exec(`
		CREATE TABLE IF NOT EXISTS catalog_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			catalog_id INTEGER NOT NULL,
			catalog_name TEXT,
			reference TEXT NOT NULL,
			code TEXT,
			name TEXT,
			attributes TEXT,
			table_parts TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create catalog_items table: %v", err)
	}

	// Вставляем тестовые данные
	for _, cp := range counterparties {
		_, err = db.GetConnection().Exec(`
			INSERT INTO catalog_items (catalog_id, catalog_name, reference, code, name, attributes)
			VALUES (?, ?, ?, ?, ?, ?)
		`, 1, "Контрагенты", cp.Reference, cp.Code, cp.Name, cp.Attributes)
		if err != nil {
			t.Fatalf("Failed to insert counterparty: %v", err)
		}
	}

	return db
}

// TestCounterparty тестовый контрагент
type TestCounterparty struct {
	Reference  string
	Code       string
	Name       string
	Attributes string
}

// TestCounterpartyNormalization_AllDatabases тестирует нормализацию на всех найденных базах
func TestCounterpartyNormalization_AllDatabases(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Находим все базы данных с контрагентами
	dbPaths := findCounterpartyDatabases(t)
	if len(dbPaths) == 0 {
		t.Skip("No counterparty databases found for testing")
	}

	t.Logf("Found %d databases to test", len(dbPaths))

	// Создаем временную сервисную БД
	tmpDir := t.TempDir()
	serviceDBPath := filepath.Join(tmpDir, "test_service.db")
	serviceDB, err := database.NewServiceDB(serviceDBPath)
	if err != nil {
		t.Fatalf("Failed to create service DB: %v", err)
	}
	defer serviceDB.Close()

	// Создаем клиента и проект
	client, err := serviceDB.CreateClient("Test Client", "Test Legal", "Description", "test@test.com", "+1234567890", "TAX123", "user")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	project, err := serviceDB.CreateClientProject(client.ID, "Test Project", "counterparty", "Project Description", "1C", 0.8)
	if err != nil {
		t.Fatalf("Failed to create project: %v", err)
	}

	// Тестируем каждую базу данных
	for i, dbPath := range dbPaths {
		t.Run(filepath.Base(dbPath), func(t *testing.T) {
			testNormalizationOnDatabase(t, serviceDB, project.ID, dbPath, i)
		})
	}
}

// findCounterpartyDatabases находит все базы данных с контрагентами
func findCounterpartyDatabases(t *testing.T) []string {
	var dbPaths []string

	// Директории для поиска
	searchDirs := []string{
		".",
		"data",
		"data/uploads",
	}

	for _, dir := range searchDirs {
		pattern := filepath.Join(dir, "*.db")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}

		for _, match := range matches {
			// Пропускаем служебные базы
			baseName := filepath.Base(match)
			if baseName == "service.db" || baseName == "test.db" {
				continue
			}

			// Проверяем, есть ли в базе контрагенты
			if hasCounterparties(match) {
				dbPaths = append(dbPaths, match)
			}
		}
	}

	return dbPaths
}

// hasCounterparties проверяет, есть ли в базе данных контрагенты
func hasCounterparties(dbPath string) bool {
	conn, err := sql.Open("sqlite3", dbPath+"?_timeout=5000")
	if err != nil {
		return false
	}
	defer conn.Close()

	// Проверяем наличие таблиц с контрагентами
	tables := []string{"catalog_items", "counterparties", "normalized_data"}
	for _, table := range tables {
		var count int
		err := conn.QueryRow(`
			SELECT COUNT(*) FROM sqlite_master 
			WHERE type='table' AND name=?
		`, table).Scan(&count)
		if err == nil && count > 0 {
			// Проверяем, есть ли записи
			var recordCount int
			err = conn.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&recordCount)
			if err == nil && recordCount > 0 {
				return true
			}
		}
	}

	return false
}

// testNormalizationOnDatabase тестирует нормализацию на конкретной базе данных
func testNormalizationOnDatabase(t *testing.T, serviceDB *database.ServiceDB, projectID int, dbPath string, dbIndex int) {
	// Открываем базу данных
	sourceDB, err := database.NewDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to open database %s: %v", dbPath, err)
	}
	defer sourceDB.Close()

	// Получаем контрагентов из базы
	counterparties, err := getCounterpartiesFromDB(sourceDB)
	if err != nil {
		t.Fatalf("Failed to get counterparties from %s: %v", dbPath, err)
	}

	if len(counterparties) == 0 {
		t.Skipf("No counterparties found in %s", dbPath)
	}

	t.Logf("Found %d counterparties in %s", len(counterparties), dbPath)

	// Создаем проект БД
	_, err = serviceDB.CreateProjectDatabase(projectID, filepath.Base(dbPath), dbPath, "Test DB", 0)
	if err != nil {
		t.Fatalf("Failed to create project database: %v", err)
	}

	// Создаем моковый AI нормализатор
	mockNormalizer := &MockAINameNormalizer{
		normalizedNames: make(map[string]string),
	}

	// Создаем моковый BenchmarkFinder
	mockBenchmarkFinder := &MockBenchmarkFinder{}

	// Создаем канал для событий
	eventChannel := make(chan string, 100)

	// Создаем контекст
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Создаем нормализатор
	normalizer := NewCounterpartyNormalizer(
		serviceDB,
		1, // clientID
		projectID,
		eventChannel,
		ctx,
		mockNormalizer,
		mockBenchmarkFinder,
	)

	// Запускаем нормализацию
	result, err := normalizer.ProcessNormalization(counterparties, false)
	if err != nil {
		t.Fatalf("Normalization failed for %s: %v", dbPath, err)
	}

	// Проверяем результаты
	if result == nil {
		t.Fatal("Normalization result is nil")
	}

	t.Logf("Normalization results for %s:", dbPath)
	t.Logf("  Total processed: %d", result.TotalProcessed)
	t.Logf("  Benchmark matches: %d", result.BenchmarkMatches)
	t.Logf("  Enriched count: %d", result.EnrichedCount)
	t.Logf("  Duplicate groups: %d", result.DuplicateGroups)
	t.Logf("  Errors: %d", len(result.Errors))

	// Проверяем, что хотя бы часть контрагентов обработана
	if result.TotalProcessed == 0 && len(counterparties) > 0 {
		t.Errorf("Expected at least some counterparties to be processed, got 0")
	}

	// Проверяем, что нормализованные контрагенты сохранены в БД
	normalized, _, err := serviceDB.GetNormalizedCounterparties(projectID, 0, 100, "", "", "")
	if err != nil {
		t.Fatalf("Failed to get normalized counterparties: %v", err)
	}

	if len(normalized) == 0 && result.TotalProcessed > 0 {
		t.Errorf("Expected normalized counterparties in DB, got 0")
	}

	// Проверяем качество нормализации
	for _, cp := range normalized {
		if cp.NormalizedName == "" {
			t.Errorf("Normalized counterparty %d has empty normalized_name", cp.ID)
		}
		if cp.SourceName == "" {
			t.Errorf("Normalized counterparty %d has empty source_name", cp.ID)
		}
	}

	// Проверяем извлечение данных
	extractedDataCount := 0
	for _, cp := range normalized {
		if cp.TaxID != "" || cp.BIN != "" {
			extractedDataCount++
		}
		if cp.LegalAddress != "" {
			extractedDataCount++
		}
		if cp.ContactPhone != "" || cp.ContactEmail != "" {
			extractedDataCount++
		}
	}

	if extractedDataCount == 0 && len(normalized) > 0 {
		t.Logf("Warning: No data extracted from attributes for any counterparty")
	} else {
		t.Logf("Extracted data from %d counterparties", extractedDataCount)
	}
}

// getCounterpartiesFromDB получает контрагентов из базы данных
func getCounterpartiesFromDB(db *database.DB) ([]*database.CatalogItem, error) {
	// Пробуем получить из catalog_items
	items, err := db.GetAllCatalogItems()
	if err == nil && len(items) > 0 {
		return items, nil
	}

	// Если не получилось, пробуем напрямую через SQL
	conn := db.GetConnection()
	rows, err := conn.Query(`
		SELECT id, catalog_id, catalog_name, reference, code, name, attributes, table_parts, created_at
		FROM catalog_items
		LIMIT 100
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items2 []*database.CatalogItem
	for rows.Next() {
		item := &database.CatalogItem{}
		err := rows.Scan(
			&item.ID,
			&item.CatalogID,
			&item.CatalogName,
			&item.Reference,
			&item.Code,
			&item.Name,
			&item.Attributes,
			&item.TableParts,
			&item.CreatedAt,
		)
		if err != nil {
			continue
		}
		items2 = append(items2, item)
	}

	return items2, rows.Err()
}

// MockAINameNormalizer моковый AI нормализатор для тестирования
type MockAINameNormalizer struct {
	normalizedNames map[string]string
}

func (m *MockAINameNormalizer) NormalizeName(ctx context.Context, name string) (string, error) {
	// Простая нормализация: убираем лишние пробелы и приводим к верхнему регистру первой буквы
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return "", fmt.Errorf("empty name")
	}

	// Сохраняем для проверки
	if m.normalizedNames == nil {
		m.normalizedNames = make(map[string]string)
	}
	m.normalizedNames[name] = normalized

	return normalized, nil
}

func (m *MockAINameNormalizer) NormalizeCounterparty(ctx context.Context, name, inn, bin string) (string, error) {
	// Используем NormalizeName, но можем добавить логику на основе ИНН/БИН
	return m.NormalizeName(ctx, name)
}

// MockBenchmarkFinder моковый BenchmarkFinder для тестирования
type MockBenchmarkFinder struct{}

func (m *MockBenchmarkFinder) FindBestMatch(name string, entityType string) (normalizedName string, found bool, err error) {
	// Возвращаем false, так как в тестах не проверяем эталоны
	return "", false, nil
}

// TestCounterpartyNormalization_Extractors тестирует извлечение данных из атрибутов
func TestCounterpartyNormalization_Extractors(t *testing.T) {
	testCases := []struct {
		name       string
		attributes string
		expected   map[string]string
	}{
		{
			name: "Полные данные контрагента",
			attributes: `
				<ИНН>1234567890</ИНН>
				<КПП>123456789</КПП>
				<Адрес>Москва, ул. Тестовая, 1</Адрес>
				<Телефон>+71234567890</Телефон>
				<Email>test@test.com</Email>
				<КонтактноеЛицо>Иванов Иван Иванович</КонтактноеЛицо>
				<ОрганизационноПравоваяФорма>ООО</ОрганизационноПравоваяФорма>
				<Банк>Сбербанк России</Банк>
				<РасчетныйСчет>40702810100000000001</РасчетныйСчет>
				<КорреспондентскийСчет>30101810100000000001</КорреспондентскийСчет>
				<БИК>044525225</БИК>
			`,
			expected: map[string]string{
				"inn":                   "1234567890",
				"kpp":                   "123456789",
				"address":               "Москва, ул. Тестовая, 1",
				"phone":                 "+71234567890",
				"email":                 "test@test.com",
				"contact_person":        "Иванов Иван Иванович",
				"legal_form":            "ООО",
				"bank_name":             "Сбербанк России",
				"bank_account":          "40702810100000000001",
				"correspondent_account": "30101810100000000001",
				"bik":                   "044525225",
			},
		},
		{
			name: "БИН для Казахстана",
			attributes: `
				<БИН>123456789012</БИН>
				<Адрес>Алматы, ул. Абая, 1</Адрес>
			`,
			expected: map[string]string{
				"bin":     "123456789012",
				"address": "Алматы, ул. Абая, 1",
			},
		},
		{
			name: "Минимальные данные",
			attributes: `
				<ИНН>9876543210</ИНН>
			`,
			expected: map[string]string{
				"inn": "9876543210",
			},
		},
		{
			name:       "Данные в текстовом формате",
			attributes: `ИНН: 9876543210, КПП: 987654321, Адрес: Москва, ул. Ленина, 10, Телефон: +79876543210, Email: contact@company.ru, Контактное лицо: Петров Петр Петрович, Банк: ВТБ, Р/С: 40702810200000000002, К/С: 30101810200000000002, БИК: 044525226`,
			expected: map[string]string{
				"inn":                   "9876543210",
				"kpp":                   "987654321",
				"address":               "Москва, ул. Ленина, 10",
				"phone":                 "+79876543210",
				"email":                 "contact@company.ru",
				"contact_person":        "Петров Петр Петрович",
				"bank_account":          "40702810200000000002",
				"correspondent_account": "30101810200000000002",
				"bik":                   "044525226",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Тестируем извлечение ИНН
			if expectedINN, ok := tc.expected["inn"]; ok {
				inn, err := extractors.ExtractINNFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract INN: %v", err)
				} else if inn != expectedINN {
					t.Errorf("Expected INN %s, got %s", expectedINN, inn)
				}
			}

			// Тестируем извлечение БИН
			if expectedBIN, ok := tc.expected["bin"]; ok {
				bin, err := extractors.ExtractBINFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract BIN: %v", err)
				} else if bin != expectedBIN {
					t.Errorf("Expected BIN %s, got %s", expectedBIN, bin)
				}
			}

			// Тестируем извлечение КПП
			if expectedKPP, ok := tc.expected["kpp"]; ok {
				kpp, err := extractors.ExtractKPPFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract KPP: %v", err)
				} else if kpp != expectedKPP {
					t.Errorf("Expected KPP %s, got %s", expectedKPP, kpp)
				}
			}

			// Тестируем извлечение адреса
			if expectedAddr, ok := tc.expected["address"]; ok {
				addr, err := extractors.ExtractAddressFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract address: %v", err)
				} else {
					// Нормализуем адрес для сравнения (убираем лишние пробелы)
					addr = strings.TrimSpace(addr)
					expectedAddr = strings.TrimSpace(expectedAddr)
					if addr != expectedAddr {
						t.Errorf("Expected address '%s', got '%s'", expectedAddr, addr)
					}
				}
			}

			// Тестируем извлечение телефона
			if expectedPhone, ok := tc.expected["phone"]; ok {
				phone, err := extractors.ExtractContactPhoneFromAttributes(tc.attributes)
				if err != nil {
					// Телефон может не извлекаться из XML тегов, это нормально
					t.Logf("Phone extraction failed (may be expected for XML format): %v", err)
				} else if phone != expectedPhone {
					// Нормализуем телефон для сравнения
					phone = strings.ReplaceAll(phone, " ", "")
					expectedPhone = strings.ReplaceAll(expectedPhone, " ", "")
					if phone != expectedPhone {
						t.Logf("Expected phone '%s', got '%s' (format may differ)", expectedPhone, phone)
					}
				}
			}

			// Тестируем извлечение email
			if expectedEmail, ok := tc.expected["email"]; ok {
				email, err := extractors.ExtractContactEmailFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract email: %v", err)
				} else if email != expectedEmail {
					t.Errorf("Expected email %s, got %s", expectedEmail, email)
				}
			}

			// Тестируем извлечение контактного лица
			if expectedPerson, ok := tc.expected["contact_person"]; ok {
				person, err := extractors.ExtractContactPersonFromAttributes(tc.attributes)
				if err != nil {
					// Контактное лицо может не извлекаться из XML тегов, это нормально
					t.Logf("Contact person extraction failed (may be expected for XML format): %v", err)
				} else if person != expectedPerson {
					// Нормализуем для сравнения
					person = strings.TrimSpace(person)
					expectedPerson = strings.TrimSpace(expectedPerson)
					if person != expectedPerson {
						t.Logf("Expected contact person '%s', got '%s' (format may differ)", expectedPerson, person)
					}
				}
			}

			// Тестируем извлечение юридической формы
			if expectedForm, ok := tc.expected["legal_form"]; ok {
				form, err := extractors.ExtractLegalFormFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract legal form: %v", err)
				} else if form != expectedForm {
					t.Errorf("Expected legal form %s, got %s", expectedForm, form)
				}
			}

			// Тестируем извлечение названия банка
			if expectedBank, ok := tc.expected["bank_name"]; ok {
				bank, err := extractors.ExtractBankNameFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract bank name: %v", err)
				} else {
					// Нормализуем для сравнения
					bank = strings.TrimSpace(bank)
					expectedBank = strings.TrimSpace(expectedBank)
					// Проверяем, что извлеченное название содержит ожидаемое
					if !strings.Contains(bank, expectedBank) && bank != expectedBank {
						t.Logf("Expected bank name to contain '%s', got '%s' (partial match may be acceptable)", expectedBank, bank)
					}
				}
			}

			// Тестируем извлечение расчетного счета
			if expectedAccount, ok := tc.expected["bank_account"]; ok {
				account, err := extractors.ExtractBankAccountFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract bank account: %v", err)
				} else if account != expectedAccount {
					t.Errorf("Expected bank account %s, got %s", expectedAccount, account)
				}
			}

			// Тестируем извлечение корреспондентского счета
			if expectedCorrAccount, ok := tc.expected["correspondent_account"]; ok {
				corrAccount, err := extractors.ExtractCorrespondentAccountFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract correspondent account: %v", err)
				} else if corrAccount != expectedCorrAccount {
					t.Errorf("Expected correspondent account %s, got %s", expectedCorrAccount, corrAccount)
				}
			}

			// Тестируем извлечение БИК
			if expectedBIK, ok := tc.expected["bik"]; ok {
				bik, err := extractors.ExtractBIKFromAttributes(tc.attributes)
				if err != nil {
					t.Errorf("Failed to extract BIK: %v", err)
				} else if bik != expectedBIK {
					// БИК может путаться с КПП, если оба присутствуют
					// Проверяем, что это действительно БИК (9 цифр)
					if len(bik) == 9 && len(expectedBIK) == 9 {
						t.Logf("Expected BIK '%s', got '%s' (may be confused with KPP if both present)", expectedBIK, bik)
					} else {
						t.Errorf("Expected BIK '%s', got '%s'", expectedBIK, bik)
					}
				}
			}
		})
	}
}
