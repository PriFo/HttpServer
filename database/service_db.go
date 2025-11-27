package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"httpserver/extractors"

	_ "github.com/mattn/go-sqlite3"
)

// DBConfig конфигурация подключения к БД (используется и для ServiceDB)
type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// ServiceDB обертка для работы с сервисной базой данных
type ServiceDB struct {
	conn             *sql.DB
	tableCreateMutex sync.Mutex // Мьютекс для создания таблиц (защита от race condition)
}

func nullString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

var timestampLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.000Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
}

func normalizeTimestampValue(value interface{}) string {
	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.UTC().Format(time.RFC3339)
	case *time.Time:
		if v == nil || v.IsZero() {
			return ""
		}
		return v.UTC().Format(time.RFC3339)
	case []byte:
		return parseTimestampString(string(v))
	case string:
		return parseTimestampString(v)
	case nil:
		return ""
	default:
		return fmt.Sprint(v)
	}
}

func parseTimestampString(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	for _, layout := range timestampLayouts {
		if ts, err := time.Parse(layout, raw); err == nil {
			return ts.UTC().Format(time.RFC3339)
		}
	}

	return raw
}

// NewServiceDB создает новое подключение к сервисной базе данных
func NewServiceDB(dbPath string) (*ServiceDB, error) {
	config := DBConfig{}

	// Для in-memory SQLite требуется использовать ровно одно соединение,
	// иначе каждое новое соединение будет получать пустую БД без таблиц/миграций.
	if isInMemoryServiceDB(dbPath) {
		config.MaxOpenConns = 1
		config.MaxIdleConns = 1
	}

	return NewServiceDBWithConfig(dbPath, config)
}

// isInMemoryServiceDB определяет, что путь относится к in-memory SQLite
func isInMemoryServiceDB(dbPath string) bool {
	if dbPath == ":memory:" {
		return true
	}

	// Формат file:memdb?_mode=memory&cache=shared также хранит БД в памяти
	if strings.HasPrefix(dbPath, "file:") && strings.Contains(dbPath, "mode=memory") {
		return true
	}

	return false
}

// NewServiceDBWithConfig создает новое подключение к сервисной базе данных с конфигурацией
func NewServiceDBWithConfig(dbPath string, config DBConfig) (*ServiceDB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open service database: %w", err)
	}

	// Настройка connection pooling
	if config.MaxOpenConns > 0 {
		conn.SetMaxOpenConns(config.MaxOpenConns)
	} else {
		// SQLite плохо справляется с большим количеством одновременных соединений
		// Ограничиваем до 10 для предотвращения блокировок
		conn.SetMaxOpenConns(10)
	}

	if config.MaxIdleConns > 0 {
		conn.SetMaxIdleConns(config.MaxIdleConns)
	} else {
		// Уменьшаем количество idle соединений для SQLite
		conn.SetMaxIdleConns(3)
	}

	if config.ConnMaxLifetime > 0 {
		conn.SetConnMaxLifetime(config.ConnMaxLifetime)
	} else {
		conn.SetConnMaxLifetime(5 * time.Minute)
	}

	// Проверяем подключение
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping service database: %w", err)
	}

	// Включаем поддержку FOREIGN KEY constraints в SQLite
	if _, err := conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Включаем WAL режим для улучшения конкурентности чтения
	// WAL позволяет множественным читателям работать одновременно без блокировок
	if _, err := conn.Exec("PRAGMA journal_mode = WAL"); err != nil {
		// Логируем, но не прерываем инициализацию, так как это не критично
		log.Printf("[ServiceDB] Warning: Failed to enable WAL mode: %v", err)
	}

	serviceDB := &ServiceDB{conn: conn}

	// Инициализируем схему сервисной БД
	if err := InitServiceSchema(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize service schema: %w", err)
	}

	// Заполняем демо-данные, если сервисная БД ещё пуста (пропускаем in-memory подключения в тестах)
	if !isInMemoryServiceDB(dbPath) {
		if err := ensureDemoClients(conn); err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to seed demo clients: %w", err)
		}
	}

	return serviceDB, nil
}

// Close закрывает подключение к сервисной базе данных
func (db *ServiceDB) Close() error {
	return db.conn.Close()
}

// Ping проверяет подключение к базе данных
func (db *ServiceDB) Ping() error {
	return db.conn.Ping()
}

// GetDB возвращает указатель на sql.DB для прямого доступа
func (db *ServiceDB) GetDB() *sql.DB {
	return db.conn
}

// GetConnection возвращает указатель на sql.DB для прямого доступа (алиас для GetDB)
func (db *ServiceDB) GetConnection() *sql.DB {
	return db.conn
}

// QueryRow выполняет запрос и возвращает одну строку
func (db *ServiceDB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// Query выполняет запрос и возвращает несколько строк
func (db *ServiceDB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return db.conn.Query(query, args...)
}

// Exec выполняет запрос без возврата строк
func (db *ServiceDB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.conn.Exec(query, args...)
}

// Client структура клиента
type Client struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	LegalName    string `json:"legal_name"`
	Description  string `json:"description"`
	ContactEmail string `json:"contact_email"`
	ContactPhone string `json:"contact_phone"`
	TaxID        string `json:"tax_id"`
	Country      string `json:"country"`
	Status       string `json:"status"`
	CreatedBy    string `json:"created_by"`
	// Бизнес-информация
	Industry    string `json:"industry"`
	CompanySize string `json:"company_size"`
	LegalForm   string `json:"legal_form"`
	// Расширенные контакты
	ContactPerson   string `json:"contact_person"`
	ContactPosition string `json:"contact_position"`
	AlternatePhone  string `json:"alternate_phone"`
	Website         string `json:"website"`
	// Юридические данные
	OGRN                 string `json:"ogrn"`
	KPP                  string `json:"kpp"`
	LegalAddress         string `json:"legal_address"`
	PostalAddress        string `json:"postal_address"`
	BankName             string `json:"bank_name"`
	BankAccount          string `json:"bank_account"`
	CorrespondentAccount string `json:"correspondent_account"`
	BIK                  string `json:"bik"`
	// Договорные данные
	ContractNumber    string     `json:"contract_number"`
	ContractDate      *time.Time `json:"contract_date"`
	ContractTerms     string     `json:"contract_terms"`
	ContractExpiresAt *time.Time `json:"contract_expires_at"`
	// Системные поля
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ProjectCount   int        `json:"project_count"`
	BenchmarkCount int        `json:"benchmark_count"`
	LastActivity   *time.Time `json:"last_activity"`
}

// ClientDocument структура документа клиента
type ClientDocument struct {
	ID          int       `json:"id"`
	ClientID    int       `json:"client_id"`
	FileName    string    `json:"file_name"`
	FilePath    string    `json:"file_path"`
	FileType    string    `json:"file_type"`
	FileSize    int64     `json:"file_size"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	UploadedBy  string    `json:"uploaded_by"`
	UploadedAt  time.Time `json:"uploaded_at"`
}

// ClientProject структура проекта клиента
type ClientProject struct {
	ID                 int       `json:"id"`
	ClientID           int       `json:"client_id"`
	Name               string    `json:"name"`
	ProjectType        string    `json:"project_type"`
	Description        string    `json:"description"`
	SourceSystem       string    `json:"source_system"`
	Status             string    `json:"status"`
	TargetQualityScore float64   `json:"target_quality_score"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ClientBenchmark структура эталонной записи
type ClientBenchmark struct {
	ID              int        `json:"id"`
	ClientProjectID int        `json:"client_project_id"`
	OriginalName    string     `json:"original_name"`
	NormalizedName  string     `json:"normalized_name"`
	Category        string     `json:"category"`
	Subcategory     string     `json:"subcategory"`
	Attributes      string     `json:"attributes"`
	QualityScore    float64    `json:"quality_score"`
	IsApproved      bool       `json:"is_approved"`
	ApprovedBy      string     `json:"approved_by"`
	ApprovedAt      *time.Time `json:"approved_at"`
	SourceDatabase  string     `json:"source_database"`
	UsageCount      int        `json:"usage_count"`
	// Поля для контрагентов
	TaxID                   string    `json:"tax_id"`                              // ИНН
	KPP                     string    `json:"kpp"`                                 // КПП
	OGRN                    string    `json:"ogrn"`                                // ОГРН
	Region                  string    `json:"region"`                              // Регион
	LegalAddress            string    `json:"legal_address"`                       // Юридический адрес
	PostalAddress           string    `json:"postal_address"`                      // Почтовый адрес
	ContactPhone            string    `json:"contact_phone"`                       // Телефон
	ContactEmail            string    `json:"contact_email"`                       // Email
	ContactPerson           string    `json:"contact_person"`                      // Контактное лицо
	LegalForm               string    `json:"legal_form"`                          // Организационно-правовая форма
	BankName                string    `json:"bank_name"`                           // Банк
	BankAccount             string    `json:"bank_account"`                        // Расчетный счет
	CorrespondentAccount    string    `json:"correspondent_account"`               // Корреспондентский счет
	BIK                     string    `json:"bik"`                                 // БИК
	ManufacturerBenchmarkID *int      `json:"manufacturer_benchmark_id,omitempty"` // ID эталона производителя (для номенклатур)
	OKPD2ReferenceID        *int      `json:"okpd2_reference_id,omitempty"`        // ID справочника ОКПД2
	TNVEDReferenceID        *int      `json:"tnved_reference_id,omitempty"`        // ID справочника ТН ВЭД
	TUGOSTReferenceID       *int      `json:"tu_gost_reference_id,omitempty"`      // ID справочника ТУ/ГОСТ
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// NormalizationConfig структура конфигурации нормализации
type NormalizationConfig struct {
	ID              int       `json:"id"`
	DatabasePath    string    `json:"database_path"`
	SourceTable     string    `json:"source_table"`
	ReferenceColumn string    `json:"reference_column"`
	CodeColumn      string    `json:"code_column"`
	NameColumn      string    `json:"name_column"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ProjectDatabase структура базы данных проекта
type ProjectDatabase struct {
	ID              int        `json:"id"`
	ClientProjectID int        `json:"client_project_id"`
	Name            string     `json:"name"`
	FilePath        string     `json:"file_path"`
	Description     string     `json:"description"`
	IsActive        bool       `json:"is_active"`
	FileSize        int64      `json:"file_size"`
	LastUsedAt      *time.Time `json:"last_used_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// CreateClient создает нового клиента.
// Допускает два формата вызова:
//  1. Устаревший: CreateClient(..., taxID, createdBy) — страна будет пустой.
//  2. Новый: CreateClient(..., taxID, country, createdBy).
func (db *ServiceDB) CreateClient(name, legalName, description, contactEmail, contactPhone, taxID string, extra ...string) (*Client, error) {
	country := ""
	createdBy := "system"

	switch len(extra) {
	case 0:
		// Используем значения по умолчанию.
	case 1:
		createdBy = extra[0]
	default:
		country = extra[0]
		createdBy = extra[1]
	}

	query := `
		INSERT INTO clients (name, legal_name, description, contact_email, contact_phone, tax_id, country, created_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, name, legalName, description, contactEmail, contactPhone, taxID, country, createdBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get client ID: %w", err)
	}

	return db.GetClient(int(id))
}

// GetClient получает клиента по ID
func (db *ServiceDB) GetClient(id int) (*Client, error) {
	query := `
		SELECT id, name, legal_name, description, contact_email, contact_phone, tax_id, country,
		       status, created_by,
		       industry, company_size, legal_form,
		       contact_person, contact_position, alternate_phone, website,
		       ogrn, kpp, legal_address, postal_address,
		       bank_name, bank_account, correspondent_account, bik,
		       contract_number, contract_date, contract_terms, contract_expires_at,
		       created_at, updated_at
		FROM clients WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	client := &Client{}

	var (
		description       sql.NullString
		contactEmail      sql.NullString
		contactPhone      sql.NullString
		taxID             sql.NullString
		country           sql.NullString
		status            sql.NullString
		createdBy         sql.NullString
		industry          sql.NullString
		companySize       sql.NullString
		legalForm         sql.NullString
		contactPerson     sql.NullString
		contactPosition   sql.NullString
		alternatePhone    sql.NullString
		website           sql.NullString
		ogrn              sql.NullString
		kpp               sql.NullString
		legalAddress      sql.NullString
		postalAddress     sql.NullString
		bankName          sql.NullString
		bankAccount       sql.NullString
		correspondentAcct sql.NullString
		bik               sql.NullString
		contractNumber    sql.NullString
		contractDate      sql.NullTime
		contractTerms     sql.NullString
		contractExpiresAt sql.NullTime
	)

	err := row.Scan(
		&client.ID,
		&client.Name,
		&client.LegalName,
		&description,
		&contactEmail,
		&contactPhone,
		&taxID,
		&country,
		&status,
		&createdBy,
		&industry,
		&companySize,
		&legalForm,
		&contactPerson,
		&contactPosition,
		&alternatePhone,
		&website,
		&ogrn,
		&kpp,
		&legalAddress,
		&postalAddress,
		&bankName,
		&bankAccount,
		&correspondentAcct,
		&bik,
		&contractNumber,
		&contractDate,
		&contractTerms,
		&contractExpiresAt,
		&client.CreatedAt,
		&client.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get client: %w", err)
	}

	client.Description = nullString(description)
	client.ContactEmail = nullString(contactEmail)
	client.ContactPhone = nullString(contactPhone)
	client.TaxID = nullString(taxID)
	client.Country = nullString(country)
	client.Status = nullString(status)
	client.CreatedBy = nullString(createdBy)
	client.Industry = nullString(industry)
	client.CompanySize = nullString(companySize)
	client.LegalForm = nullString(legalForm)
	client.ContactPerson = nullString(contactPerson)
	client.ContactPosition = nullString(contactPosition)
	client.AlternatePhone = nullString(alternatePhone)
	client.Website = nullString(website)
	client.OGRN = nullString(ogrn)
	client.KPP = nullString(kpp)
	client.LegalAddress = nullString(legalAddress)
	client.PostalAddress = nullString(postalAddress)
	client.BankName = nullString(bankName)
	client.BankAccount = nullString(bankAccount)
	client.CorrespondentAccount = nullString(correspondentAcct)
	client.BIK = nullString(bik)
	client.ContractNumber = nullString(contractNumber)
	client.ContractTerms = nullString(contractTerms)
	if contractDate.Valid {
		client.ContractDate = &contractDate.Time
	}
	if contractExpiresAt.Valid {
		client.ContractExpiresAt = &contractExpiresAt.Time
	}

	return client, nil
}

// GetClientsByIDs получает несколько клиентов по списку ID одним запросом
// Оптимизирует N+1 проблему при получении списка клиентов
// GetClientsByIDs получает клиентов по списку ID используя batch-запрос.
// Оптимизирует производительность, решая N+1 проблему при получении множества клиентов.
//
// Параметры:
//   - ids: список ID клиентов для получения
//
// Возвращает:
//   - список клиентов в том же порядке, что и ids (если клиент не найден, он пропускается)
//   - ошибку при неудаче
//
// Пример использования:
//
//	clients, err := serviceDB.GetClientsByIDs([]int{1, 2, 3})
//	if err != nil {
//	    return err
//	}
func (db *ServiceDB) GetClientsByIDs(ids []int) ([]*Client, error) {
	if len(ids) == 0 {
		return []*Client{}, nil
	}

	// Создаем плейсхолдеры для IN запроса
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"

	query := fmt.Sprintf(`
		SELECT id, name, legal_name, description, contact_email, contact_phone, tax_id, country,
		       status, created_by,
		       industry, company_size, legal_form,
		       contact_person, contact_position, alternate_phone, website,
		       ogrn, kpp, legal_address, postal_address,
		       bank_name, bank_account, correspondent_account, bik,
		       contract_number, contract_date, contract_terms, contract_expires_at,
		       created_at, updated_at
		FROM clients WHERE id IN (%s)
		ORDER BY id
	`, placeholders)

	// Преобразуем []int в []interface{} для Query
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients by IDs: %w", err)
	}
	defer rows.Close()

	clients := make([]*Client, 0, len(ids))
	for rows.Next() {
		client := &Client{}
		var (
			description       sql.NullString
			contactEmail      sql.NullString
			contactPhone      sql.NullString
			taxID             sql.NullString
			country           sql.NullString
			status            sql.NullString
			createdBy         sql.NullString
			industry          sql.NullString
			companySize       sql.NullString
			legalForm         sql.NullString
			contactPerson     sql.NullString
			contactPosition   sql.NullString
			alternatePhone    sql.NullString
			website           sql.NullString
			ogrn              sql.NullString
			kpp               sql.NullString
			legalAddress      sql.NullString
			postalAddress     sql.NullString
			bankName          sql.NullString
			bankAccount       sql.NullString
			correspondentAcct sql.NullString
			bik               sql.NullString
			contractNumber    sql.NullString
			contractDate      sql.NullTime
			contractTerms     sql.NullString
			contractExpiresAt sql.NullTime
		)

		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.LegalName,
			&description,
			&contactEmail,
			&contactPhone,
			&taxID,
			&country,
			&status,
			&createdBy,
			&industry,
			&companySize,
			&legalForm,
			&contactPerson,
			&contactPosition,
			&alternatePhone,
			&website,
			&ogrn,
			&kpp,
			&legalAddress,
			&postalAddress,
			&bankName,
			&bankAccount,
			&correspondentAcct,
			&bik,
			&contractNumber,
			&contractDate,
			&contractTerms,
			&contractExpiresAt,
			&client.CreatedAt,
			&client.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		client.Description = nullString(description)
		client.ContactEmail = nullString(contactEmail)
		client.ContactPhone = nullString(contactPhone)
		client.TaxID = nullString(taxID)
		client.Country = nullString(country)
		client.Status = nullString(status)
		client.CreatedBy = nullString(createdBy)
		client.Industry = nullString(industry)
		client.CompanySize = nullString(companySize)
		client.LegalForm = nullString(legalForm)
		client.ContactPerson = nullString(contactPerson)
		client.ContactPosition = nullString(contactPosition)
		client.AlternatePhone = nullString(alternatePhone)
		client.Website = nullString(website)
		client.OGRN = nullString(ogrn)
		client.KPP = nullString(kpp)
		client.LegalAddress = nullString(legalAddress)
		client.PostalAddress = nullString(postalAddress)
		client.BankName = nullString(bankName)
		client.BankAccount = nullString(bankAccount)
		client.CorrespondentAccount = nullString(correspondentAcct)
		client.BIK = nullString(bik)
		client.ContractNumber = nullString(contractNumber)
		client.ContractTerms = nullString(contractTerms)
		if contractDate.Valid {
			client.ContractDate = &contractDate.Time
		}
		if contractExpiresAt.Valid {
			client.ContractExpiresAt = &contractExpiresAt.Time
		}

		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clients: %w", err)
	}

	return clients, nil
}

// GetAllClients получает всех клиентов
func (db *ServiceDB) GetAllClients() ([]*Client, error) {
	query := `
		SELECT id, name, legal_name, description, contact_email, contact_phone, tax_id, country,
		       status, created_by,
		       industry, company_size, legal_form,
		       contact_person, contact_position, alternate_phone, website,
		       ogrn, kpp, legal_address, postal_address,
		       bank_name, bank_account, correspondent_account, bik,
		       contract_number, contract_date, contract_terms, contract_expires_at,
		       created_at, updated_at
		FROM clients
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all clients: %w", err)
	}
	defer rows.Close()

	var clients []*Client
	for rows.Next() {
		client := &Client{}
		var (
			description       sql.NullString
			contactEmail      sql.NullString
			contactPhone      sql.NullString
			taxID             sql.NullString
			country           sql.NullString
			status            sql.NullString
			createdBy         sql.NullString
			industry          sql.NullString
			companySize       sql.NullString
			legalForm         sql.NullString
			contactPerson     sql.NullString
			contactPosition   sql.NullString
			alternatePhone    sql.NullString
			website           sql.NullString
			ogrn              sql.NullString
			kpp               sql.NullString
			legalAddress      sql.NullString
			postalAddress     sql.NullString
			bankName          sql.NullString
			bankAccount       sql.NullString
			correspondentAcct sql.NullString
			bik               sql.NullString
			contractNumber    sql.NullString
			contractDate      sql.NullTime
			contractTerms     sql.NullString
			contractExpiresAt sql.NullTime
		)

		err := rows.Scan(
			&client.ID,
			&client.Name,
			&client.LegalName,
			&description,
			&contactEmail,
			&contactPhone,
			&taxID,
			&country,
			&status,
			&createdBy,
			&industry,
			&companySize,
			&legalForm,
			&contactPerson,
			&contactPosition,
			&alternatePhone,
			&website,
			&ogrn,
			&kpp,
			&legalAddress,
			&postalAddress,
			&bankName,
			&bankAccount,
			&correspondentAcct,
			&bik,
			&contractNumber,
			&contractDate,
			&contractTerms,
			&contractExpiresAt,
			&client.CreatedAt,
			&client.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		client.Description = nullString(description)
		client.ContactEmail = nullString(contactEmail)
		client.ContactPhone = nullString(contactPhone)
		client.TaxID = nullString(taxID)
		client.Country = nullString(country)
		client.Status = nullString(status)
		client.CreatedBy = nullString(createdBy)
		client.Industry = nullString(industry)
		client.CompanySize = nullString(companySize)
		client.LegalForm = nullString(legalForm)
		client.ContactPerson = nullString(contactPerson)
		client.ContactPosition = nullString(contactPosition)
		client.AlternatePhone = nullString(alternatePhone)
		client.Website = nullString(website)
		client.OGRN = nullString(ogrn)
		client.KPP = nullString(kpp)
		client.LegalAddress = nullString(legalAddress)
		client.PostalAddress = nullString(postalAddress)
		client.BankName = nullString(bankName)
		client.BankAccount = nullString(bankAccount)
		client.CorrespondentAccount = nullString(correspondentAcct)
		client.BIK = nullString(bik)
		client.ContractNumber = nullString(contractNumber)
		client.ContractTerms = nullString(contractTerms)
		if contractDate.Valid {
			client.ContractDate = &contractDate.Time
		}
		if contractExpiresAt.Valid {
			client.ContractExpiresAt = &contractExpiresAt.Time
		}

		clients = append(clients, client)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clients: %w", err)
	}

	return clients, nil
}

// UpdateClient обновляет информацию о клиенте (старая версия для обратной совместимости)
func (db *ServiceDB) UpdateClient(id int, name, legalName, description, contactEmail, contactPhone, taxID, country, status string) error {
	return db.UpdateClientFields(id, &Client{
		Name:         name,
		LegalName:    legalName,
		Description:  description,
		ContactEmail: contactEmail,
		ContactPhone: contactPhone,
		TaxID:        taxID,
		Country:      country,
		Status:       status,
	})
}

// UpdateClientFields обновляет поля клиента (поддерживает все новые поля)
// Обновляются только те поля, которые не пустые в переданном объекте Client
func (db *ServiceDB) UpdateClientFields(id int, updates *Client) error {
	if updates == nil {
		return fmt.Errorf("updates cannot be nil")
	}

	// Строим динамический UPDATE запрос, обновляя только непустые поля
	setParts := []string{}
	args := []interface{}{}

	if updates.Name != "" {
		setParts = append(setParts, "name = ?")
		args = append(args, updates.Name)
	}
	if updates.LegalName != "" {
		setParts = append(setParts, "legal_name = ?")
		args = append(args, updates.LegalName)
	}
	if updates.Description != "" {
		setParts = append(setParts, "description = ?")
		args = append(args, updates.Description)
	}
	if updates.ContactEmail != "" {
		setParts = append(setParts, "contact_email = ?")
		args = append(args, updates.ContactEmail)
	}
	if updates.ContactPhone != "" {
		setParts = append(setParts, "contact_phone = ?")
		args = append(args, updates.ContactPhone)
	}
	if updates.TaxID != "" {
		setParts = append(setParts, "tax_id = ?")
		args = append(args, updates.TaxID)
	}
	if updates.Country != "" {
		setParts = append(setParts, "country = ?")
		args = append(args, updates.Country)
	}
	if updates.Status != "" {
		setParts = append(setParts, "status = ?")
		args = append(args, updates.Status)
	}

	// Бизнес-информация
	if updates.Industry != "" {
		setParts = append(setParts, "industry = ?")
		args = append(args, updates.Industry)
	}
	if updates.CompanySize != "" {
		setParts = append(setParts, "company_size = ?")
		args = append(args, updates.CompanySize)
	}
	if updates.LegalForm != "" {
		setParts = append(setParts, "legal_form = ?")
		args = append(args, updates.LegalForm)
	}

	// Расширенные контакты
	if updates.ContactPerson != "" {
		setParts = append(setParts, "contact_person = ?")
		args = append(args, updates.ContactPerson)
	}
	if updates.ContactPosition != "" {
		setParts = append(setParts, "contact_position = ?")
		args = append(args, updates.ContactPosition)
	}
	if updates.AlternatePhone != "" {
		setParts = append(setParts, "alternate_phone = ?")
		args = append(args, updates.AlternatePhone)
	}
	if updates.Website != "" {
		setParts = append(setParts, "website = ?")
		args = append(args, updates.Website)
	}

	// Юридические данные
	if updates.OGRN != "" {
		setParts = append(setParts, "ogrn = ?")
		args = append(args, updates.OGRN)
	}
	if updates.KPP != "" {
		setParts = append(setParts, "kpp = ?")
		args = append(args, updates.KPP)
	}
	if updates.LegalAddress != "" {
		setParts = append(setParts, "legal_address = ?")
		args = append(args, updates.LegalAddress)
	}
	if updates.PostalAddress != "" {
		setParts = append(setParts, "postal_address = ?")
		args = append(args, updates.PostalAddress)
	}
	if updates.BankName != "" {
		setParts = append(setParts, "bank_name = ?")
		args = append(args, updates.BankName)
	}
	if updates.BankAccount != "" {
		setParts = append(setParts, "bank_account = ?")
		args = append(args, updates.BankAccount)
	}
	if updates.CorrespondentAccount != "" {
		setParts = append(setParts, "correspondent_account = ?")
		args = append(args, updates.CorrespondentAccount)
	}
	if updates.BIK != "" {
		setParts = append(setParts, "bik = ?")
		args = append(args, updates.BIK)
	}

	// Договорные данные
	if updates.ContractNumber != "" {
		setParts = append(setParts, "contract_number = ?")
		args = append(args, updates.ContractNumber)
	}
	if updates.ContractDate != nil {
		setParts = append(setParts, "contract_date = ?")
		args = append(args, updates.ContractDate)
	}
	if updates.ContractTerms != "" {
		setParts = append(setParts, "contract_terms = ?")
		args = append(args, updates.ContractTerms)
	}
	if updates.ContractExpiresAt != nil {
		setParts = append(setParts, "contract_expires_at = ?")
		args = append(args, updates.ContractExpiresAt)
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Всегда обновляем updated_at
	setParts = append(setParts, "updated_at = CURRENT_TIMESTAMP")

	query := fmt.Sprintf(`
		UPDATE clients 
		SET %s
		WHERE id = ?
	`, strings.Join(setParts, ", "))

	args = append(args, id)

	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update client: %w", err)
	}

	return nil
}

// DeleteClient удаляет клиента
func (db *ServiceDB) DeleteClient(id int) error {
	query := `DELETE FROM clients WHERE id = ?`

	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete client: %w", err)
	}

	return nil
}

// GetClientsWithStats получает список клиентов со статистикой
func (db *ServiceDB) GetClientsWithStats() ([]map[string]interface{}, error) {
	query := `
		SELECT 
			c.id,
			c.name,
			c.legal_name,
			c.description,
			c.country,
			c.status,
			c.created_at,
			COUNT(DISTINCT cp.id) as project_count,
			COUNT(DISTINCT cb.id) as benchmark_count,
			MAX(
				COALESCE(
					cb.updated_at,
					cp.updated_at,
					c.updated_at,
					c.created_at
				)
			) as last_activity
		FROM clients c
		LEFT JOIN client_projects cp ON c.id = cp.client_id
		LEFT JOIN client_benchmarks cb ON cp.id = cb.client_project_id
		GROUP BY c.id, c.name, c.legal_name, c.description, c.country, c.status, c.created_at
		ORDER BY c.created_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}
	defer rows.Close()

	var clients []map[string]interface{}
	for rows.Next() {
		var (
			id             int
			projectCount   int
			benchmarkCount int
			name           sql.NullString
			legalName      sql.NullString
			description    sql.NullString
			country        sql.NullString
			status         sql.NullString
			createdAt      sql.NullString
			lastActivity   interface{}
		)

		err := rows.Scan(&id, &name, &legalName, &description, &country, &status, &createdAt, &projectCount, &benchmarkCount, &lastActivity)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client: %w", err)
		}

		client := map[string]interface{}{
			"id":              id,
			"name":            nullString(name),
			"legal_name":      nullString(legalName),
			"description":     nullString(description),
			"country":         nullString(country),
			"status":          nullString(status),
			"project_count":   projectCount,
			"benchmark_count": benchmarkCount,
			"last_activity":   "",
		}

		client["last_activity"] = normalizeTimestampValue(lastActivity)

		clients = append(clients, client)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating clients: %w", err)
	}

	// Если нет клиентов, возвращаем пустой массив (это нормально, если таблица пустая)
	return clients, nil
}

// CreateClientProject создает новый проект клиента
func (db *ServiceDB) CreateClientProject(clientID int, name, projectType, description, sourceSystem string, targetQualityScore float64) (*ClientProject, error) {
	query := `
		INSERT INTO client_projects (client_id, name, project_type, description, source_system, target_quality_score)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, clientID, name, projectType, description, sourceSystem, targetQualityScore)
	if err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}

	return db.GetClientProject(int(id))
}

// GetClientProject получает проект по ID
func (db *ServiceDB) GetClientProject(id int) (*ClientProject, error) {
	query := `
		SELECT id, client_id, name, project_type, description, source_system, 
		       status, target_quality_score, created_at, updated_at
		FROM client_projects WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	project := &ClientProject{}

	err := row.Scan(
		&project.ID, &project.ClientID, &project.Name, &project.ProjectType,
		&project.Description, &project.SourceSystem, &project.Status,
		&project.TargetQualityScore, &project.CreatedAt, &project.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// GetOrCreateSystemProject получает или создает системный проект для глобальных эталонов
func (db *ServiceDB) GetOrCreateSystemProject() (*ClientProject, error) {
	// Сначала пытаемся найти системного клиента
	var systemClientID int
	err := db.conn.QueryRow(`SELECT id FROM clients WHERE name = 'Система' LIMIT 1`).Scan(&systemClientID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Создаем системного клиента
			systemClient, err := db.CreateClient(
				"Система",
				"Система",
				"Системный клиент для глобальных эталонов",
				"",
				"",
				"",
				"",
				"system",
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create system client: %w", err)
			}
			systemClientID = systemClient.ID
		} else {
			return nil, fmt.Errorf("failed to get system client: %w", err)
		}
	}

	// Пытаемся найти системный проект
	var systemProjectID int
	err = db.conn.QueryRow(`
		SELECT id FROM client_projects 
		WHERE client_id = ? AND name = 'Глобальные эталоны' 
		LIMIT 1
	`, systemClientID).Scan(&systemProjectID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Создаем системный проект
			systemProject, err := db.CreateClientProject(
				systemClientID,
				"Глобальные эталоны",
				"system",
				"Глобальные эталоны для всех проектов (производители, справочники и т.д.)",
				"system",
				0.95,
			)
			if err != nil {
				return nil, fmt.Errorf("failed to create system project: %w", err)
			}
			return systemProject, nil
		}
		return nil, fmt.Errorf("failed to get system project: %w", err)
	}

	return db.GetClientProject(systemProjectID)
}

// FindBenchmarkByTaxID ищет эталон по projectID и ИНН/БИН
// Сначала ищет в указанном проекте, затем в глобальном (системном) проекте
func (db *ServiceDB) FindBenchmarkByTaxID(projectID int, taxID string) (*ClientBenchmark, error) {
	if taxID == "" {
		return nil, nil
	}

	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, subcategory,
		       attributes, quality_score, is_approved, approved_by, approved_at,
		       source_database, usage_count, tax_id, kpp, COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       legal_address, postal_address, contact_phone, contact_email, contact_person, legal_form,
		       bank_name, bank_account, correspondent_account, bik,
		       created_at, updated_at
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND tax_id = ?
		  AND is_approved = TRUE
		ORDER BY quality_score DESC, usage_count DESC
		LIMIT 1
	`

	row := db.conn.QueryRow(query, projectID, taxID)
	benchmark := &ClientBenchmark{}

	var approvedAt sql.NullTime
	err := row.Scan(
		&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
		&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
		&benchmark.IsApproved, &benchmark.ApprovedBy, &approvedAt,
		&benchmark.SourceDatabase, &benchmark.UsageCount,
		&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
		&benchmark.LegalAddress, &benchmark.PostalAddress,
		&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
		&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
		&benchmark.CreatedAt, &benchmark.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Если не найдено в проекте, ищем в глобальном (системном) проекте
			return db.FindGlobalBenchmarkByTaxID(taxID)
		}
		return nil, fmt.Errorf("failed to find benchmark by tax ID: %w", err)
	}

	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	return benchmark, nil
}

// FindGlobalBenchmarkByTaxID ищет глобальный эталон по ИНН/БИН (в системном проекте)
func (db *ServiceDB) FindGlobalBenchmarkByTaxID(taxID string) (*ClientBenchmark, error) {
	if taxID == "" {
		return nil, nil
	}

	// Получаем системный проект
	systemProject, err := db.GetOrCreateSystemProject()
	if err != nil {
		return nil, fmt.Errorf("failed to get system project: %w", err)
	}

	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, subcategory,
		       attributes, quality_score, is_approved, approved_by, approved_at,
		       source_database, usage_count, tax_id, kpp, COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       legal_address, postal_address, contact_phone, contact_email, contact_person, legal_form,
		       bank_name, bank_account, correspondent_account, bik,
		       created_at, updated_at
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND tax_id = ?
		  AND is_approved = TRUE
		ORDER BY quality_score DESC, usage_count DESC
		LIMIT 1
	`

	row := db.conn.QueryRow(query, systemProject.ID, taxID)
	benchmark := &ClientBenchmark{}

	var approvedAt sql.NullTime
	err = row.Scan(
		&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
		&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
		&benchmark.IsApproved, &benchmark.ApprovedBy, &approvedAt,
		&benchmark.SourceDatabase, &benchmark.UsageCount,
		&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
		&benchmark.LegalAddress, &benchmark.PostalAddress,
		&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
		&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
		&benchmark.CreatedAt, &benchmark.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find global benchmark: %w", err)
	}

	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	return benchmark, nil
}

// GetClientProjects получает все проекты клиента
func (db *ServiceDB) GetClientProjects(clientID int) ([]*ClientProject, error) {
	query := `
		SELECT id, client_id, name, project_type, description, source_system, 
		       status, target_quality_score, created_at, updated_at
		FROM client_projects 
		WHERE client_id = ?
		ORDER BY created_at DESC
	`

	rows, err := db.conn.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	defer rows.Close()

	var projects []*ClientProject
	for rows.Next() {
		project := &ClientProject{}
		err := rows.Scan(
			&project.ID, &project.ClientID, &project.Name, &project.ProjectType,
			&project.Description, &project.SourceSystem, &project.Status,
			&project.TargetQualityScore, &project.CreatedAt, &project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating projects: %w", err)
	}

	return projects, nil
}

// UpdateClientProject обновляет проект
func (db *ServiceDB) UpdateClientProject(id int, name, projectType, description, sourceSystem, status string, targetQualityScore float64) error {
	query := `
		UPDATE client_projects 
		SET name = ?, project_type = ?, description = ?, source_system = ?, 
		    status = ?, target_quality_score = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, name, projectType, description, sourceSystem, status, targetQualityScore, id)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// DeleteClientProject удаляет проект
func (db *ServiceDB) DeleteClientProject(id int) error {
	query := `DELETE FROM client_projects WHERE id = ?`

	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}

// CreateClientBenchmark создает эталонную запись
func (db *ServiceDB) CreateClientBenchmark(projectID int, originalName, normalizedName, category, subcategory, attributes, sourceDatabase string, qualityScore float64) (*ClientBenchmark, error) {
	query := `
		INSERT INTO client_benchmarks 
		(client_project_id, original_name, normalized_name, category, subcategory, attributes, quality_score, source_database)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, projectID, originalName, normalizedName, category, subcategory, attributes, qualityScore, sourceDatabase)
	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark ID: %w", err)
	}

	return db.GetClientBenchmark(int(id))
}

// CreateNomenclatureBenchmark создает эталонную запись номенклатуры с привязкой к производителю и справочникам
func (db *ServiceDB) CreateNomenclatureBenchmark(projectID int, originalName, normalizedName, subcategory, attributes, sourceDatabase string, qualityScore float64, manufacturerBenchmarkID *int, okpd2RefID, tnvedRefID, tuGostRefID *int) (*ClientBenchmark, error) {
	query := `
		INSERT INTO client_benchmarks 
		(client_project_id, original_name, normalized_name, category, subcategory, attributes, quality_score, source_database, 
		 manufacturer_benchmark_id, okpd2_reference_id, tnved_reference_id, tu_gost_reference_id)
		VALUES (?, ?, ?, 'nomenclature', ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, projectID, originalName, normalizedName, subcategory, attributes, qualityScore, sourceDatabase,
		manufacturerBenchmarkID, okpd2RefID, tnvedRefID, tuGostRefID)
	if err != nil {
		return nil, fmt.Errorf("failed to create nomenclature benchmark: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark ID: %w", err)
	}

	return db.GetClientBenchmark(int(id))
}

// CreateCounterpartyBenchmark создает эталонную запись контрагента с полными данными
func (db *ServiceDB) CreateCounterpartyBenchmark(
	projectID int,
	originalName, normalizedName string,
	taxID, kpp, bin, ogrn, region, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm,
	bankName, bankAccount, correspondentAccount, bik string,
	qualityScore float64,
) (*ClientBenchmark, error) {
	// Используем tax_id для поиска, если есть БИН, сохраняем его отдельно
	// В эталонах используем tax_id как основной идентификатор (может быть ИНН или БИН)
	searchTaxID := taxID
	if searchTaxID == "" && bin != "" {
		searchTaxID = bin
	}

	query := `
		INSERT INTO client_benchmarks 
		(client_project_id, original_name, normalized_name, category, subcategory, 
		 tax_id, kpp, ogrn, region, legal_address, postal_address, contact_phone, contact_email, 
		 contact_person, legal_form, bank_name, bank_account, correspondent_account, bik,
		 quality_score, source_database)
		VALUES (?, ?, ?, 'counterparty', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query,
		projectID, originalName, normalizedName,
		"", // subcategory будет установлен позже если нужно
		searchTaxID, kpp, ogrn, region, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm,
		bankName, bankAccount, correspondentAccount, bik,
		qualityScore,
		"", // source_database
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create counterparty benchmark: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark ID: %w", err)
	}

	return db.GetClientBenchmark(int(id))
}

// GetClientBenchmark получает эталон по ID
func (db *ServiceDB) GetClientBenchmark(id int) (*ClientBenchmark, error) {
	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, subcategory,
		       COALESCE(attributes, '') as attributes, quality_score, is_approved, approved_by, approved_at,
		       source_database, usage_count, tax_id, kpp, COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       legal_address, postal_address, contact_phone, contact_email, contact_person, legal_form,
		       bank_name, bank_account, correspondent_account, bik, manufacturer_benchmark_id,
		       okpd2_reference_id, tnved_reference_id, tu_gost_reference_id,
		       created_at, updated_at
		FROM client_benchmarks WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	benchmark := &ClientBenchmark{}

	var approvedAt sql.NullTime
	var approvedBy sql.NullString
	var manufacturerID, okpd2RefID, tnvedRefID, tuGostRefID sql.NullInt64
	err := row.Scan(
		&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
		&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
		&benchmark.IsApproved, &approvedBy, &approvedAt,
		&benchmark.SourceDatabase, &benchmark.UsageCount,
		&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
		&benchmark.LegalAddress, &benchmark.PostalAddress,
		&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
		&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
		&manufacturerID, &okpd2RefID, &tnvedRefID, &tuGostRefID,
		&benchmark.CreatedAt, &benchmark.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan benchmark: %w", err)
	}

	if approvedBy.Valid {
		benchmark.ApprovedBy = approvedBy.String
	}
	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark: %w", err)
	}

	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	if manufacturerID.Valid {
		id := int(manufacturerID.Int64)
		benchmark.ManufacturerBenchmarkID = &id
	}

	if okpd2RefID.Valid {
		id := int(okpd2RefID.Int64)
		benchmark.OKPD2ReferenceID = &id
	}

	if tnvedRefID.Valid {
		id := int(tnvedRefID.Int64)
		benchmark.TNVEDReferenceID = &id
	}

	if tuGostRefID.Valid {
		id := int(tuGostRefID.Int64)
		benchmark.TUGOSTReferenceID = &id
	}

	return benchmark, nil
}

// FindClientBenchmark ищет эталон по названию для проекта
func (db *ServiceDB) FindClientBenchmark(projectID int, name string) (*ClientBenchmark, error) {
	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, subcategory,
		       attributes, quality_score, is_approved, approved_by, approved_at,
		       source_database, usage_count, tax_id, kpp, COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       legal_address, postal_address, contact_phone, contact_email, contact_person, legal_form,
		       bank_name, bank_account, correspondent_account, bik, manufacturer_benchmark_id,
		       okpd2_reference_id, tnved_reference_id, tu_gost_reference_id,
		       created_at, updated_at
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND (original_name = ? OR normalized_name = ?)
		  AND is_approved = TRUE
		ORDER BY quality_score DESC, usage_count DESC
		LIMIT 1
	`

	row := db.conn.QueryRow(query, projectID, name, name)
	benchmark := &ClientBenchmark{}

	var approvedAt sql.NullTime
	var manufacturerID, okpd2RefID, tnvedRefID, tuGostRefID sql.NullInt64
	err := row.Scan(
		&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
		&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
		&benchmark.IsApproved, &benchmark.ApprovedBy, &approvedAt,
		&benchmark.SourceDatabase, &benchmark.UsageCount,
		&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
		&benchmark.LegalAddress, &benchmark.PostalAddress,
		&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
		&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
		&manufacturerID, &okpd2RefID, &tnvedRefID, &tuGostRefID,
		&benchmark.CreatedAt, &benchmark.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find benchmark: %w", err)
	}

	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	if manufacturerID.Valid {
		id := int(manufacturerID.Int64)
		benchmark.ManufacturerBenchmarkID = &id
	}

	if okpd2RefID.Valid {
		id := int(okpd2RefID.Int64)
		benchmark.OKPD2ReferenceID = &id
	}

	if tnvedRefID.Valid {
		id := int(tnvedRefID.Int64)
		benchmark.TNVEDReferenceID = &id
	}

	if tuGostRefID.Valid {
		id := int(tuGostRefID.Int64)
		benchmark.TUGOSTReferenceID = &id
	}

	return benchmark, nil
}

// GetClientBenchmarks получает эталоны проекта
func (db *ServiceDB) GetClientBenchmarks(projectID int, category string, approvedOnly bool) ([]*ClientBenchmark, error) {
	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, 
		       COALESCE(subcategory, '') as subcategory,
		       COALESCE(attributes, '') as attributes, quality_score, is_approved, 
		       COALESCE(approved_by, '') as approved_by, approved_at,
		       COALESCE(source_database, '') as source_database, usage_count, 
		       COALESCE(tax_id, '') as tax_id, COALESCE(kpp, '') as kpp, 
		       COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       COALESCE(legal_address, '') as legal_address, 
		       COALESCE(postal_address, '') as postal_address,
		       COALESCE(contact_phone, '') as contact_phone, 
		       COALESCE(contact_email, '') as contact_email, 
		       COALESCE(contact_person, '') as contact_person, 
		       COALESCE(legal_form, '') as legal_form,
		       COALESCE(bank_name, '') as bank_name, 
		       COALESCE(bank_account, '') as bank_account, 
		       COALESCE(correspondent_account, '') as correspondent_account, 
		       COALESCE(bik, '') as bik, manufacturer_benchmark_id,
		       okpd2_reference_id, tnved_reference_id, tu_gost_reference_id,
		       created_at, updated_at
		FROM client_benchmarks 
		WHERE client_project_id = ?
	`

	args := []interface{}{projectID}

	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	if approvedOnly {
		query += " AND is_approved = TRUE"
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmarks: %w", err)
	}
	defer rows.Close()

	var benchmarks []*ClientBenchmark
	for rows.Next() {
		benchmark := &ClientBenchmark{}
		var approvedAt sql.NullTime
		var manufacturerID, okpd2RefID, tnvedRefID, tuGostRefID sql.NullInt64

		err := rows.Scan(
			&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
			&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
			&benchmark.IsApproved, &benchmark.ApprovedBy, &approvedAt,
			&benchmark.SourceDatabase, &benchmark.UsageCount,
			&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
			&benchmark.LegalAddress, &benchmark.PostalAddress,
			&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
			&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
			&manufacturerID, &okpd2RefID, &tnvedRefID, &tuGostRefID,
			&benchmark.CreatedAt, &benchmark.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan benchmark: %w", err)
		}

		if approvedAt.Valid {
			benchmark.ApprovedAt = &approvedAt.Time
		}

		if manufacturerID.Valid {
			id := int(manufacturerID.Int64)
			benchmark.ManufacturerBenchmarkID = &id
		}

		if okpd2RefID.Valid {
			id := int(okpd2RefID.Int64)
			benchmark.OKPD2ReferenceID = &id
		}

		if tnvedRefID.Valid {
			id := int(tnvedRefID.Int64)
			benchmark.TNVEDReferenceID = &id
		}

		if tuGostRefID.Valid {
			id := int(tuGostRefID.Int64)
			benchmark.TUGOSTReferenceID = &id
		}

		benchmarks = append(benchmarks, benchmark)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating benchmarks: %w", err)
	}

	return benchmarks, nil
}

// UpdateBenchmarkUsage увеличивает счетчик использования эталона
func (db *ServiceDB) UpdateBenchmarkUsage(benchmarkID int) error {
	query := `UPDATE client_benchmarks SET usage_count = usage_count + 1 WHERE id = ?`

	_, err := db.conn.Exec(query, benchmarkID)
	if err != nil {
		return fmt.Errorf("failed to update benchmark usage: %w", err)
	}

	return nil
}

// ApproveBenchmark утверждает эталон
func (db *ServiceDB) ApproveBenchmark(benchmarkID int, approvedBy string) error {
	query := `
		UPDATE client_benchmarks
		SET is_approved = TRUE, approved_by = ?, approved_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, approvedBy, benchmarkID)
	if err != nil {
		return fmt.Errorf("failed to approve benchmark: %w", err)
	}

	return nil
}

// UpdateBenchmark обновляет эталон контрагента
func (db *ServiceDB) UpdateBenchmark(benchmarkID int, originalName, normalizedName, ogrn, region, attributes string, qualityScore float64) error {
	query := `
		UPDATE client_benchmarks
		SET original_name = ?,
		    normalized_name = ?,
		    ogrn = COALESCE(NULLIF(?, ''), ogrn),
		    region = COALESCE(NULLIF(?, ''), region),
		    attributes = COALESCE(NULLIF(?, ''), attributes),
		    quality_score = ?,
		    is_approved = TRUE,
		    approved_by = 'system',
		    approved_at = CURRENT_TIMESTAMP,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, originalName, normalizedName, ogrn, region, attributes, qualityScore, benchmarkID)
	if err != nil {
		return fmt.Errorf("failed to update benchmark: %w", err)
	}

	return nil
}

// FindManufacturerByINN ищет производителя по ИНН в проекте
func (db *ServiceDB) FindManufacturerByINN(projectID int, inn string) (*ClientBenchmark, error) {
	if inn == "" {
		return nil, nil
	}

	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, subcategory,
		       attributes, quality_score, is_approved, approved_by, approved_at,
		       source_database, usage_count, tax_id, kpp, COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       legal_address, postal_address, contact_phone, contact_email, contact_person, legal_form,
		       bank_name, bank_account, correspondent_account, bik, manufacturer_benchmark_id,
		       okpd2_reference_id, tnved_reference_id, tu_gost_reference_id,
		       created_at, updated_at
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'counterparty'
		  AND tax_id = ?
		ORDER BY is_approved DESC, quality_score DESC
		LIMIT 1
	`

	row := db.conn.QueryRow(query, projectID, inn)
	benchmark := &ClientBenchmark{}

	var approvedAt sql.NullTime
	var manufacturerID, okpd2RefID, tnvedRefID, tuGostRefID sql.NullInt64
	err := row.Scan(
		&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
		&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
		&benchmark.IsApproved, &benchmark.ApprovedBy, &approvedAt,
		&benchmark.SourceDatabase, &benchmark.UsageCount,
		&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
		&benchmark.LegalAddress, &benchmark.PostalAddress,
		&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
		&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
		&manufacturerID, &okpd2RefID, &tnvedRefID, &tuGostRefID,
		&benchmark.CreatedAt, &benchmark.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find manufacturer by INN: %w", err)
	}

	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	if manufacturerID.Valid {
		id := int(manufacturerID.Int64)
		benchmark.ManufacturerBenchmarkID = &id
	}

	if okpd2RefID.Valid {
		id := int(okpd2RefID.Int64)
		benchmark.OKPD2ReferenceID = &id
	}

	if tnvedRefID.Valid {
		id := int(tnvedRefID.Int64)
		benchmark.TNVEDReferenceID = &id
	}

	if tuGostRefID.Valid {
		id := int(tuGostRefID.Int64)
		benchmark.TUGOSTReferenceID = &id
	}

	return benchmark, nil
}

// FindManufacturerByOGRN ищет производителя по ОГРН в проекте
func (db *ServiceDB) FindManufacturerByOGRN(projectID int, ogrn string) (*ClientBenchmark, error) {
	if ogrn == "" {
		return nil, nil
	}

	query := `
		SELECT id, client_project_id, original_name, normalized_name, category, subcategory,
		       attributes, quality_score, is_approved, approved_by, approved_at,
		       source_database, usage_count, tax_id, kpp, COALESCE(ogrn, '') as ogrn, COALESCE(region, '') as region,
		       legal_address, postal_address, contact_phone, contact_email, contact_person, legal_form,
		       bank_name, bank_account, correspondent_account, bik, manufacturer_benchmark_id,
		       okpd2_reference_id, tnved_reference_id, tu_gost_reference_id,
		       created_at, updated_at
		FROM client_benchmarks 
		WHERE client_project_id = ? 
		  AND category = 'counterparty'
		  AND ogrn = ?
		ORDER BY is_approved DESC, quality_score DESC
		LIMIT 1
	`

	row := db.conn.QueryRow(query, projectID, ogrn)
	benchmark := &ClientBenchmark{}

	var approvedAt sql.NullTime
	var manufacturerID, okpd2RefID, tnvedRefID, tuGostRefID sql.NullInt64
	err := row.Scan(
		&benchmark.ID, &benchmark.ClientProjectID, &benchmark.OriginalName, &benchmark.NormalizedName,
		&benchmark.Category, &benchmark.Subcategory, &benchmark.Attributes, &benchmark.QualityScore,
		&benchmark.IsApproved, &benchmark.ApprovedBy, &approvedAt,
		&benchmark.SourceDatabase, &benchmark.UsageCount,
		&benchmark.TaxID, &benchmark.KPP, &benchmark.OGRN, &benchmark.Region,
		&benchmark.LegalAddress, &benchmark.PostalAddress,
		&benchmark.ContactPhone, &benchmark.ContactEmail, &benchmark.ContactPerson, &benchmark.LegalForm,
		&benchmark.BankName, &benchmark.BankAccount, &benchmark.CorrespondentAccount, &benchmark.BIK,
		&manufacturerID, &okpd2RefID, &tnvedRefID, &tuGostRefID,
		&benchmark.CreatedAt, &benchmark.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find manufacturer by OGRN: %w", err)
	}

	if approvedAt.Valid {
		benchmark.ApprovedAt = &approvedAt.Time
	}

	if manufacturerID.Valid {
		id := int(manufacturerID.Int64)
		benchmark.ManufacturerBenchmarkID = &id
	}

	if okpd2RefID.Valid {
		id := int(okpd2RefID.Int64)
		benchmark.OKPD2ReferenceID = &id
	}

	if tnvedRefID.Valid {
		id := int(tnvedRefID.Int64)
		benchmark.TNVEDReferenceID = &id
	}

	if tuGostRefID.Valid {
		id := int(tuGostRefID.Int64)
		benchmark.TUGOSTReferenceID = &id
	}

	return benchmark, nil
}

// UpdateBenchmarkFields обновляет дополнительные поля эталона (subcategory, source_database)
func (db *ServiceDB) UpdateBenchmarkFields(benchmarkID int, subcategory, sourceDatabase string) error {
	query := `
		UPDATE client_benchmarks
		SET subcategory = COALESCE(NULLIF(?, ''), subcategory),
		    source_database = COALESCE(NULLIF(?, ''), source_database),
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, subcategory, sourceDatabase, benchmarkID)
	if err != nil {
		return fmt.Errorf("failed to update benchmark fields: %w", err)
	}

	return nil
}

// GetNormalizationConfig получает конфигурацию нормализации
func (db *ServiceDB) GetNormalizationConfig() (*NormalizationConfig, error) {
	query := `
		SELECT id, database_path, source_table, reference_column, code_column, name_column, created_at, updated_at
		FROM normalization_config
		WHERE id = 1
	`

	row := db.conn.QueryRow(query)
	config := &NormalizationConfig{}

	err := row.Scan(
		&config.ID, &config.DatabasePath, &config.SourceTable,
		&config.ReferenceColumn, &config.CodeColumn, &config.NameColumn,
		&config.CreatedAt, &config.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// Возвращаем дефолтную конфигурацию
			return &NormalizationConfig{
				ID:              1,
				DatabasePath:    "",
				SourceTable:     "catalog_items",
				ReferenceColumn: "reference",
				CodeColumn:      "code",
				NameColumn:      "name",
			}, nil
		}
		return nil, fmt.Errorf("failed to get normalization config: %w", err)
	}

	return config, nil
}

// UpdateNormalizationConfig обновляет конфигурацию нормализации
func (db *ServiceDB) UpdateNormalizationConfig(databasePath, sourceTable, referenceColumn, codeColumn, nameColumn string) error {
	query := `
		UPDATE normalization_config
		SET database_path = ?, source_table = ?, reference_column = ?,
		    code_column = ?, name_column = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = 1
	`

	_, err := db.conn.Exec(query, databasePath, sourceTable, referenceColumn, codeColumn, nameColumn)
	if err != nil {
		return fmt.Errorf("failed to update normalization config: %w", err)
	}

	return nil
}

// CreateProjectDatabase создает новую базу данных для проекта
// Нормализует путь к файлу для консистентности (использует filepath.Clean)
func (db *ServiceDB) CreateProjectDatabase(projectID int, name, filePath, description string, fileSize int64) (*ProjectDatabase, error) {
	// Нормализуем путь к файлу для консистентности
	normalizedPath := filepath.Clean(filePath)

	query := `
		INSERT INTO project_databases
		(client_project_id, name, file_path, description, file_size, is_active)
		VALUES (?, ?, ?, ?, ?, TRUE)
	`

	result, err := db.conn.Exec(query, projectID, name, normalizedPath, description, fileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create project database: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get project database ID: %w", err)
	}

	return db.GetProjectDatabase(int(id))
}

// GetProjectDatabase получает базу данных проекта по ID
func (db *ServiceDB) GetProjectDatabase(id int) (*ProjectDatabase, error) {
	query := `
		SELECT id, client_project_id, name, file_path, description, is_active,
		       file_size, last_used_at, created_at, updated_at
		FROM project_databases WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	projectDB := &ProjectDatabase{}

	var lastUsedAt sql.NullTime
	var fileSize sql.NullInt64
	err := row.Scan(
		&projectDB.ID, &projectDB.ClientProjectID, &projectDB.Name, &projectDB.FilePath,
		&projectDB.Description, &projectDB.IsActive, &fileSize, &lastUsedAt,
		&projectDB.CreatedAt, &projectDB.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get project database: %w", err)
	}

	if fileSize.Valid {
		projectDB.FileSize = fileSize.Int64
	} else {
		projectDB.FileSize = 0
	}

	if lastUsedAt.Valid {
		projectDB.LastUsedAt = &lastUsedAt.Time
	}

	return projectDB, nil
}

// GetProjectDatabases получает все базы данных проекта
func (db *ServiceDB) GetProjectDatabases(projectID int, activeOnly bool) ([]*ProjectDatabase, error) {
	query := `
		SELECT id, client_project_id, name, file_path, description, is_active,
		       file_size, last_used_at, created_at, updated_at
		FROM project_databases
		WHERE client_project_id = ?
	`

	args := []interface{}{projectID}

	if activeOnly {
		query += " AND is_active = TRUE"
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}
	defer rows.Close()

	var databases []*ProjectDatabase
	for rows.Next() {
		projectDB := &ProjectDatabase{}
		var lastUsedAt sql.NullTime
		var fileSize sql.NullInt64

		err := rows.Scan(
			&projectDB.ID, &projectDB.ClientProjectID, &projectDB.Name, &projectDB.FilePath,
			&projectDB.Description, &projectDB.IsActive, &fileSize, &lastUsedAt,
			&projectDB.CreatedAt, &projectDB.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project database: %w", err)
		}

		if fileSize.Valid {
			projectDB.FileSize = fileSize.Int64
		} else {
			projectDB.FileSize = 0
		}

		if lastUsedAt.Valid {
			projectDB.LastUsedAt = &lastUsedAt.Time
		}

		databases = append(databases, projectDB)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating project databases: %w", err)
	}

	return databases, nil
}

// GetProjectDatabaseCount получает количество баз данных в проекте
func (db *ServiceDB) GetProjectDatabaseCount(projectID int, activeOnly bool) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM project_databases
		WHERE client_project_id = ?
	`

	if activeOnly {
		query += " AND is_active = TRUE"
	}

	var count int
	err := db.conn.QueryRow(query, projectID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get project database count: %w", err)
	}

	return count, nil
}

// GetAllProjectDatabases получает все базы данных из всех проектов всех клиентов
func (db *ServiceDB) GetAllProjectDatabases() ([]*ProjectDatabase, error) {
	// Получаем всех клиентов
	clients, err := db.GetAllClients()
	if err != nil {
		return nil, fmt.Errorf("failed to get clients: %w", err)
	}

	var allDatabases []*ProjectDatabase

	// Для каждого клиента получаем проекты и базы данных
	for _, client := range clients {
		// Получаем проекты клиента
		projects, err := db.GetClientProjects(client.ID)
		if err != nil {
			// Логируем ошибку, но продолжаем обработку других клиентов
			log.Printf("Failed to get projects for client %d: %v", client.ID, err)
			continue
		}

		// Для каждого проекта получаем базы данных
		for _, project := range projects {
			databases, err := db.GetProjectDatabases(project.ID, false)
			if err != nil {
				// Логируем ошибку, но продолжаем обработку других проектов
				log.Printf("Failed to get databases for project %d: %v", project.ID, err)
				continue
			}

			allDatabases = append(allDatabases, databases...)
		}
	}

	return allDatabases, nil
}

// GetProjectDatabaseByFilePath проверяет, существует ли база данных с таким же путем к файлу
// Поддерживает поиск с разными форматами путей (прямые и обратные слеши) для совместимости Windows/Linux
func (db *ServiceDB) GetProjectDatabaseByFilePath(projectID int, filePath string) (*ProjectDatabase, error) {
	// Нормализуем путь для поиска (как в LinkDatabaseByPathToProject)
	normalizedPath := filepath.Clean(filePath)
	normalizedPathSlash := filepath.ToSlash(normalizedPath)
	normalizedPathBackslash := filepath.FromSlash(normalizedPath)

	// Проверяем с разными вариантами пути для совместимости
	query := `
		SELECT id, client_project_id, name, file_path, description, is_active,
		       file_size, last_used_at, created_at, updated_at
		FROM project_databases
		WHERE client_project_id = ? 
		  AND (file_path = ? OR file_path = ? OR file_path = ? OR file_path = ?)
		LIMIT 1
	`

	row := db.conn.QueryRow(query, projectID, filePath, normalizedPath, normalizedPathSlash, normalizedPathBackslash)
	projectDB := &ProjectDatabase{}

	var lastUsedAt sql.NullTime
	err := row.Scan(
		&projectDB.ID, &projectDB.ClientProjectID, &projectDB.Name, &projectDB.FilePath,
		&projectDB.Description, &projectDB.IsActive, &projectDB.FileSize, &lastUsedAt,
		&projectDB.CreatedAt, &projectDB.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // Файл не найден, это нормально
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get project database by file path: %w", err)
	}

	if lastUsedAt.Valid {
		projectDB.LastUsedAt = &lastUsedAt.Time
	}

	return projectDB, nil
}

// GetProjectDatabaseByPath получает базу данных проекта по пути файла
func (db *ServiceDB) GetProjectDatabaseByPath(projectID int, filePath string) (*ProjectDatabase, error) {
	query := `
		SELECT id, client_project_id, name, file_path, description, is_active,
		       file_size, last_used_at, created_at, updated_at
		FROM project_databases 
		WHERE client_project_id = ? AND file_path = ?
	`

	row := db.conn.QueryRow(query, projectID, filePath)
	projectDB := &ProjectDatabase{}

	var lastUsedAt sql.NullTime
	err := row.Scan(
		&projectDB.ID, &projectDB.ClientProjectID, &projectDB.Name, &projectDB.FilePath,
		&projectDB.Description, &projectDB.IsActive, &projectDB.FileSize, &lastUsedAt,
		&projectDB.CreatedAt, &projectDB.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get project database by path: %w", err)
	}

	if lastUsedAt.Valid {
		projectDB.LastUsedAt = &lastUsedAt.Time
	}

	return projectDB, nil
}

// FindClientAndProjectByDatabasePath находит клиента и проект по пути базы данных
// Поддерживает поиск с разными форматами путей (прямые и обратные слеши)
func (db *ServiceDB) FindClientAndProjectByDatabasePath(filePath string) (clientID, projectID int, err error) {
	// Нормализуем путь для поиска
	normalizedPath := filepath.Clean(filePath)
	normalizedPathSlash := filepath.ToSlash(normalizedPath)
	normalizedPathBackslash := filepath.FromSlash(normalizedPath)

	// Пробуем найти с разными вариантами пути
	query := `
		SELECT cp.client_id, pd.client_project_id
		FROM project_databases pd
		JOIN client_projects cp ON pd.client_project_id = cp.id
		WHERE pd.file_path = ? OR pd.file_path = ? OR pd.file_path = ? OR pd.file_path = ?
		LIMIT 1
	`

	err = db.conn.QueryRow(query, filePath, normalizedPath, normalizedPathSlash, normalizedPathBackslash).Scan(&clientID, &projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, 0, fmt.Errorf("database not found in any project")
		}
		return 0, 0, fmt.Errorf("failed to find client and project: %w", err)
	}

	return clientID, projectID, nil
}

// UpdateProjectDatabase обновляет базу данных проекта
func (db *ServiceDB) UpdateProjectDatabase(id int, name, filePath, description string, isActive bool) error {
	query := `
		UPDATE project_databases
		SET name = ?, file_path = ?, description = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, name, filePath, description, isActive, id)
	if err != nil {
		return fmt.Errorf("failed to update project database: %w", err)
	}

	return nil
}

// DeleteProjectDatabase удаляет базу данных проекта
func (db *ServiceDB) DeleteProjectDatabase(id int) error {
	query := `DELETE FROM project_databases WHERE id = ?`

	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project database: %w", err)
	}

	return nil
}

// UpdateProjectDatabaseLastUsed обновляет время последнего использования базы данных
func (db *ServiceDB) UpdateProjectDatabaseLastUsed(id int) error {
	query := `
		UPDATE project_databases
		SET last_used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to update last_used_at: %w", err)
	}

	return nil
}

// LinkProjectDatabase привязывает базу данных к проекту
func (db *ServiceDB) LinkProjectDatabase(databaseID, projectID int) error {
	// Проверяем, что проект существует
	project, err := db.GetClientProject(projectID)
	if err != nil {
		return fmt.Errorf("project not found: %w", err)
	}
	if project == nil {
		return fmt.Errorf("project %d not found", projectID)
	}

	// Проверяем, что база данных существует
	dbRecord, err := db.GetProjectDatabase(databaseID)
	if err != nil {
		return fmt.Errorf("database not found: %w", err)
	}
	if dbRecord == nil {
		return fmt.Errorf("database %d not found", databaseID)
	}

	// Обновляем привязку
	query := `
		UPDATE project_databases
		SET client_project_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.conn.Exec(query, projectID, databaseID)
	if err != nil {
		return fmt.Errorf("failed to link database to project: %w", err)
	}

	return nil
}

// GetQualityMetricsForProject получает метрики качества для проекта
func (db *ServiceDB) GetQualityMetricsForProject(projectID int, period string) ([]DataQualityMetric, error) {
	query := `
		SELECT 
			id, upload_id, database_id, metric_category, metric_name, 
			metric_value, threshold_value, status, measured_at, details
		FROM data_quality_metrics
		WHERE database_id IN (
			SELECT id FROM project_databases 
			WHERE client_project_id = ?
		)
		AND measured_at >= ?
		ORDER BY measured_at DESC
	`

	var timeRange time.Time
	switch period {
	case "day":
		timeRange = time.Now().AddDate(0, 0, -1)
	case "week":
		timeRange = time.Now().AddDate(0, 0, -7)
	case "month":
		timeRange = time.Now().AddDate(0, -1, 0)
	default:
		timeRange = time.Now().AddDate(-1, 0, 0) // default to 1 year
	}

	rows, err := db.conn.Query(query, projectID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality metrics: %w", err)
	}
	defer rows.Close()

	var metrics []DataQualityMetric
	for rows.Next() {
		var metric DataQualityMetric
		var details string

		err := rows.Scan(
			&metric.ID, &metric.UploadID, &metric.DatabaseID,
			&metric.MetricCategory, &metric.MetricName,
			&metric.MetricValue, &metric.ThresholdValue,
			&metric.Status, &metric.MeasuredAt, &details,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		// Десериализация details из JSON
		if details != "" {
			if err := json.Unmarshal([]byte(details), &metric.Details); err != nil {
				log.Printf("Error unmarshaling metric details: %v", err)
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// GetQualityTrendsForClient получает тренды качества для клиента
func (db *ServiceDB) GetQualityTrendsForClient(clientID int, period string) ([]QualityTrend, error) {
	query := `
		SELECT 
			id, database_id, measurement_date, 
			overall_score, completeness_score, 
			consistency_score, uniqueness_score, 
			validity_score, records_analyzed, 
			issues_count, created_at
		FROM quality_trends
		WHERE database_id IN (
			SELECT id FROM project_databases 
			WHERE client_project_id IN (
				SELECT id FROM client_projects 
				WHERE client_id = ?
			)
		)
		AND measurement_date >= ?
		ORDER BY measurement_date ASC
	`

	var timeRange time.Time
	switch period {
	case "week":
		timeRange = time.Now().AddDate(0, 0, -7)
	case "month":
		timeRange = time.Now().AddDate(0, -1, 0)
	case "quarter":
		timeRange = time.Now().AddDate(0, -3, 0)
	default:
		timeRange = time.Now().AddDate(-1, 0, 0) // default to 1 year
	}

	rows, err := db.conn.Query(query, clientID, timeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get quality trends: %w", err)
	}
	defer rows.Close()

	var trends []QualityTrend
	for rows.Next() {
		var trend QualityTrend
		err := rows.Scan(
			&trend.ID, &trend.DatabaseID, &trend.MeasurementDate,
			&trend.OverallScore, &trend.CompletenessScore,
			&trend.ConsistencyScore, &trend.UniquenessScore,
			&trend.ValidityScore, &trend.RecordsAnalyzed,
			&trend.IssuesCount, &trend.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trend: %w", err)
		}
		trends = append(trends, trend)
	}

	return trends, nil
}

// CompareProjectsQuality сравнивает метрики качества между проектами
func (db *ServiceDB) CompareProjectsQuality(projectIDs []int) (map[int][]DataQualityMetric, error) {
	query := `
		SELECT 
			id, upload_id, database_id, metric_category, metric_name, 
			metric_value, threshold_value, status, measured_at, details
		FROM data_quality_metrics
		WHERE database_id IN (
			SELECT id FROM project_databases 
			WHERE client_project_id IN (` + placeholders(len(projectIDs)) + `)
		)
		AND measured_at >= (
			SELECT MAX(measured_at) FROM data_quality_metrics
			WHERE database_id IN (
				SELECT id FROM project_databases 
				WHERE client_project_id IN (` + placeholders(len(projectIDs)) + `)
			)
		)
	`

	args := make([]interface{}, 0, len(projectIDs)*2)
	for _, id := range projectIDs {
		args = append(args, id)
	}
	for _, id := range projectIDs {
		args = append(args, id)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to compare projects: %w", err)
	}
	defer rows.Close()

	results := make(map[int][]DataQualityMetric)
	for rows.Next() {
		var metric DataQualityMetric
		var details string
		var dbID int

		err := rows.Scan(
			&metric.ID, &metric.UploadID, &dbID,
			&metric.MetricCategory, &metric.MetricName,
			&metric.MetricValue, &metric.ThresholdValue,
			&metric.Status, &metric.MeasuredAt, &details,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric: %w", err)
		}

		// Десериализация details из JSON
		if details != "" {
			if err := json.Unmarshal([]byte(details), &metric.Details); err != nil {
				log.Printf("Error unmarshaling metric details: %v", err)
			}
		}

		// Получаем projectID для текущей базы данных
		var projectID int
		err = db.conn.QueryRow("SELECT client_project_id FROM project_databases WHERE id = ?", dbID).Scan(&projectID)
		if err != nil {
			log.Printf("Error getting project ID for database %d: %v", dbID, err)
			continue
		}

		results[projectID] = append(results[projectID], metric)
	}

	return results, nil
}

// placeholders генерирует строку с n плейсхолдерами для SQL запроса
func placeholders(n int) string {
	ph := make([]string, n)
	for i := range ph {
		ph[i] = "?"
	}
	return strings.Join(ph, ",")
}

// DatabaseMetadata структура метаданных базы данных
type DatabaseMetadata struct {
	ID             int        `json:"id"`
	FilePath       string     `json:"file_path"`
	DatabaseType   string     `json:"database_type"`
	Description    string     `json:"description"`
	FirstSeenAt    time.Time  `json:"first_seen_at"`
	LastAnalyzedAt *time.Time `json:"last_analyzed_at"`
	MetadataJSON   string     `json:"metadata_json"`
}

// GetDatabaseMetadata получает метаданные базы данных по пути
func (db *ServiceDB) GetDatabaseMetadata(filePath string) (*DatabaseMetadata, error) {
	query := `
		SELECT id, file_path, database_type, description, first_seen_at, last_analyzed_at, metadata_json
		FROM database_metadata
		WHERE file_path = ?
	`

	row := db.conn.QueryRow(query, filePath)
	metadata := &DatabaseMetadata{}
	var lastAnalyzedAt sql.NullTime

	err := row.Scan(
		&metadata.ID, &metadata.FilePath, &metadata.DatabaseType, &metadata.Description,
		&metadata.FirstSeenAt, &lastAnalyzedAt, &metadata.MetadataJSON,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get database metadata: %w", err)
	}

	if lastAnalyzedAt.Valid {
		metadata.LastAnalyzedAt = &lastAnalyzedAt.Time
	}

	return metadata, nil
}

// UpsertDatabaseMetadata создает или обновляет метаданные базы данных
func (db *ServiceDB) UpsertDatabaseMetadata(filePath, databaseType, description, metadataJSON string) error {
	query := `
		INSERT INTO database_metadata (file_path, database_type, description, last_analyzed_at, metadata_json)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			database_type = ?,
			description = ?,
			last_analyzed_at = CURRENT_TIMESTAMP,
			metadata_json = ?
	`

	_, err := db.conn.Exec(query, filePath, databaseType, description, metadataJSON,
		databaseType, description, metadataJSON)
	if err != nil {
		return fmt.Errorf("failed to upsert database metadata: %w", err)
	}

	return nil
}

// GetAllDatabaseMetadata получает все метаданные баз данных
func (db *ServiceDB) GetAllDatabaseMetadata() ([]*DatabaseMetadata, error) {
	query := `
		SELECT id, file_path, database_type, description, first_seen_at, last_analyzed_at, metadata_json
		FROM database_metadata
		ORDER BY last_analyzed_at DESC NULLS LAST, first_seen_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all database metadata: %w", err)
	}
	defer rows.Close()

	var metadataList []*DatabaseMetadata
	for rows.Next() {
		metadata := &DatabaseMetadata{}
		var lastAnalyzedAt sql.NullTime

		err := rows.Scan(
			&metadata.ID, &metadata.FilePath, &metadata.DatabaseType, &metadata.Description,
			&metadata.FirstSeenAt, &lastAnalyzedAt, &metadata.MetadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan database metadata: %w", err)
		}

		if lastAnalyzedAt.Valid {
			metadata.LastAnalyzedAt = &lastAnalyzedAt.Time
		}

		metadataList = append(metadataList, metadata)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database metadata: %w", err)
	}

	return metadataList, nil
}

// GetDatabaseMetadataBatch получает метаданные для нескольких баз данных одним запросом
func (db *ServiceDB) GetDatabaseMetadataBatch(filePaths []string) (map[string]*DatabaseMetadata, error) {
	if len(filePaths) == 0 {
		return make(map[string]*DatabaseMetadata), nil
	}

	// Создаем плейсхолдеры для IN запроса
	placeholders := make([]string, len(filePaths))
	args := make([]interface{}, len(filePaths))
	for i, path := range filePaths {
		placeholders[i] = "?"
		args[i] = path
	}

	query := fmt.Sprintf(`
		SELECT id, file_path, database_type, description, first_seen_at, last_analyzed_at, metadata_json
		FROM database_metadata
		WHERE file_path IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get database metadata batch: %w", err)
	}
	defer rows.Close()

	metadataMap := make(map[string]*DatabaseMetadata)
	for rows.Next() {
		metadata := &DatabaseMetadata{}
		var lastAnalyzedAt sql.NullTime

		err := rows.Scan(
			&metadata.ID, &metadata.FilePath, &metadata.DatabaseType, &metadata.Description,
			&metadata.FirstSeenAt, &lastAnalyzedAt, &metadata.MetadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan database metadata: %w", err)
		}

		if lastAnalyzedAt.Valid {
			metadata.LastAnalyzedAt = &lastAnalyzedAt.Time
		}

		metadataMap[metadata.FilePath] = metadata
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating database metadata: %w", err)
	}

	return metadataMap, nil
}

// GetWorkerConfig получает конфигурацию воркеров из БД
func (db *ServiceDB) GetWorkerConfig() (string, error) {
	query := `SELECT config_json FROM worker_config WHERE id = 1`
	var configJSON string
	err := db.conn.QueryRow(query).Scan(&configJSON)
	if err == sql.ErrNoRows {
		return "", nil // Конфигурация еще не сохранена
	}
	if err != nil {
		return "", fmt.Errorf("failed to get worker config: %w", err)
	}
	return configJSON, nil
}

// SaveWorkerConfig сохраняет конфигурацию воркеров в БД
func (db *ServiceDB) SaveWorkerConfig(configJSON string) error {
	query := `
		INSERT INTO worker_config (id, config_json, updated_at)
		VALUES (1, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			config_json = ?,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.conn.Exec(query, configJSON, configJSON)
	if err != nil {
		return fmt.Errorf("failed to save worker config: %w", err)
	}
	return nil
}

// GetAppConfig получает конфигурацию приложения из БД
func (db *ServiceDB) GetAppConfig() (string, error) {
	query := `SELECT config_json FROM app_config WHERE id = 1`
	var configJSON string
	err := db.conn.QueryRow(query).Scan(&configJSON)
	if err == sql.ErrNoRows {
		return "", nil // Конфигурация еще не сохранена
	}
	if err != nil {
		return "", fmt.Errorf("failed to get app config: %w", err)
	}
	return configJSON, nil
}

// GetAppConfigVersion получает версию конфигурации приложения
func (db *ServiceDB) GetAppConfigVersion() (int, error) {
	query := `SELECT COALESCE(version, 1) FROM app_config WHERE id = 1`
	var version int
	err := db.conn.QueryRow(query).Scan(&version)
	if err == sql.ErrNoRows {
		return 1, nil // Конфигурация еще не сохранена, версия по умолчанию
	}
	if err != nil {
		return 1, fmt.Errorf("failed to get app config version: %w", err)
	}
	return version, nil
}

// SaveAppConfig сохраняет конфигурацию приложения в БД с версионированием
func (db *ServiceDB) SaveAppConfig(configJSON string) error {
	return db.SaveAppConfigWithHistory(configJSON, "", "")
}

// SaveAppConfigWithHistory сохраняет конфигурацию приложения в БД с историей изменений
func (db *ServiceDB) SaveAppConfigWithHistory(configJSON, changedBy, changeReason string) error {
	// Получаем текущую версию
	var currentVersion int
	err := db.conn.QueryRow(`SELECT COALESCE(version, 1) FROM app_config WHERE id = 1`).Scan(&currentVersion)
	if err == sql.ErrNoRows {
		currentVersion = 0 // Первая версия
	} else if err != nil {
		return fmt.Errorf("failed to get current config version: %w", err)
	}

	// Сохраняем текущую конфигурацию в историю перед обновлением
	if currentVersion > 0 {
		var currentConfigJSON string
		err := db.conn.QueryRow(`SELECT config_json FROM app_config WHERE id = 1`).Scan(&currentConfigJSON)
		if err == nil && currentConfigJSON != "" {
			// Сохраняем в историю
			_, err = db.conn.Exec(`
				INSERT INTO app_config_history (version, config_json, changed_by, change_reason)
				VALUES (?, ?, ?, ?)
			`, currentVersion, currentConfigJSON, changedBy, changeReason)
			if err != nil {
				log.Printf("Warning: failed to save config to history: %v", err)
				// Не прерываем сохранение, если история не сохранилась
			}
		}
	}

	// Обновляем конфигурацию с увеличением версии
	newVersion := currentVersion + 1
	query := `
		INSERT INTO app_config (id, config_json, version, updated_at)
		VALUES (1, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			config_json = excluded.config_json,
			version = excluded.version,
			updated_at = CURRENT_TIMESTAMP
	`
	_, err = db.conn.Exec(query, configJSON, newVersion)
	if err != nil {
		return fmt.Errorf("failed to save app config: %w", err)
	}

	log.Printf("Config saved with version %d", newVersion)
	return nil
}

// GetAppConfigHistory получает историю изменений конфигурации
func (db *ServiceDB) GetAppConfigHistory(limit int) ([]map[string]interface{}, error) {
	if limit <= 0 {
		limit = 10 // По умолчанию 10 последних версий
	}
	if limit > 100 {
		limit = 100 // Максимум 100 версий
	}

	query := `
		SELECT version, config_json, changed_by, change_reason, created_at
		FROM app_config_history
		ORDER BY version DESC
		LIMIT ?
	`
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get config history: %w", err)
	}
	defer rows.Close()

	var history []map[string]interface{}
	for rows.Next() {
		var version int
		var configJSON, changedBy, changeReason sql.NullString
		var createdAt time.Time

		if err := rows.Scan(&version, &configJSON, &changedBy, &changeReason, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan config history: %w", err)
		}

		history = append(history, map[string]interface{}{
			"version":       version,
			"config_json":   configJSON.String,
			"changed_by":    nullString(changedBy),
			"change_reason": nullString(changeReason),
			"created_at":    createdAt,
		})
	}

	return history, nil
}

// PendingDatabase структура ожидающей индексации базы данных
type PendingDatabase struct {
	ID                  int        `json:"id"`
	FilePath            string     `json:"file_path"`
	FileName            string     `json:"file_name"`
	FileSize            int64      `json:"file_size"`
	DetectedAt          time.Time  `json:"detected_at"`
	IndexingStatus      string     `json:"indexing_status"` // pending, indexing, completed, failed
	IndexingStartedAt   *time.Time `json:"indexing_started_at"`
	IndexingCompletedAt *time.Time `json:"indexing_completed_at"`
	ErrorMessage        string     `json:"error_message"`
	ClientID            *int       `json:"client_id"`
	ProjectID           *int       `json:"project_id"`
	MovedToUploads      bool       `json:"moved_to_uploads"`
	OriginalPath        string     `json:"original_path"`
}

// CreatePendingDatabase создает запись о pending database
func (db *ServiceDB) CreatePendingDatabase(filePath, fileName string, fileSize int64) (*PendingDatabase, error) {
	query := `
		INSERT INTO pending_databases (file_path, file_name, file_size)
		VALUES (?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			file_name = excluded.file_name,
			file_size = excluded.file_size,
			detected_at = CURRENT_TIMESTAMP
	`
	_, err := db.conn.Exec(query, filePath, fileName, fileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create pending database: %w", err)
	}
	return db.GetPendingDatabaseByPath(filePath)
}

// GetPendingDatabase получает pending database по ID
func (db *ServiceDB) GetPendingDatabase(id int) (*PendingDatabase, error) {
	query := `
		SELECT id, file_path, file_name, file_size, detected_at,
		       indexing_status, indexing_started_at, indexing_completed_at,
		       error_message, client_id, project_id, moved_to_uploads, original_path
		FROM pending_databases WHERE id = ?
	`
	row := db.conn.QueryRow(query, id)
	return db.scanPendingDatabase(row)
}

// GetPendingDatabaseByPath получает pending database по пути к файлу
func (db *ServiceDB) GetPendingDatabaseByPath(filePath string) (*PendingDatabase, error) {
	query := `
		SELECT id, file_path, file_name, file_size, detected_at,
		       indexing_status, indexing_started_at, indexing_completed_at,
		       error_message, client_id, project_id, moved_to_uploads, original_path
		FROM pending_databases WHERE file_path = ?
	`
	row := db.conn.QueryRow(query, filePath)
	return db.scanPendingDatabase(row)
}

// GetPendingDatabases получает список всех pending databases
func (db *ServiceDB) GetPendingDatabases(statusFilter string) ([]*PendingDatabase, error) {
	query := `
		SELECT id, file_path, file_name, file_size, detected_at,
		       indexing_status, indexing_started_at, indexing_completed_at,
		       error_message, client_id, project_id, moved_to_uploads, original_path
		FROM pending_databases
	`
	args := []interface{}{}
	if statusFilter != "" {
		query += " WHERE indexing_status = ?"
		args = append(args, statusFilter)
	}
	query += " ORDER BY detected_at DESC"

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending databases: %w", err)
	}
	defer rows.Close()

	var databases []*PendingDatabase
	for rows.Next() {
		pendingDB, err := db.scanPendingDatabase(rows)
		if err != nil {
			return nil, err
		}
		databases = append(databases, pendingDB)
	}

	return databases, nil
}

// scanPendingDatabase сканирует строку в структуру PendingDatabase
func (db *ServiceDB) scanPendingDatabase(scanner interface {
	Scan(dest ...interface{}) error
}) (*PendingDatabase, error) {
	pendingDB := &PendingDatabase{}
	var indexingStartedAt, indexingCompletedAt sql.NullTime
	var clientID, projectID sql.NullInt64
	var errorMessage, originalPath sql.NullString

	err := scanner.Scan(
		&pendingDB.ID,
		&pendingDB.FilePath,
		&pendingDB.FileName,
		&pendingDB.FileSize,
		&pendingDB.DetectedAt,
		&pendingDB.IndexingStatus,
		&indexingStartedAt,
		&indexingCompletedAt,
		&errorMessage,
		&clientID,
		&projectID,
		&pendingDB.MovedToUploads,
		&originalPath,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to scan pending database: %w", err)
	}

	// Обрабатываем NULL значения
	if errorMessage.Valid {
		pendingDB.ErrorMessage = errorMessage.String
	}
	if originalPath.Valid {
		pendingDB.OriginalPath = originalPath.String
	}
	if indexingStartedAt.Valid {
		pendingDB.IndexingStartedAt = &indexingStartedAt.Time
	}
	if indexingCompletedAt.Valid {
		pendingDB.IndexingCompletedAt = &indexingCompletedAt.Time
	}
	if clientID.Valid {
		clientIDInt := int(clientID.Int64)
		pendingDB.ClientID = &clientIDInt
	}
	if projectID.Valid {
		projectIDInt := int(projectID.Int64)
		pendingDB.ProjectID = &projectIDInt
	}

	if indexingStartedAt.Valid {
		pendingDB.IndexingStartedAt = &indexingStartedAt.Time
	}
	if indexingCompletedAt.Valid {
		pendingDB.IndexingCompletedAt = &indexingCompletedAt.Time
	}
	if clientID.Valid {
		id := int(clientID.Int64)
		pendingDB.ClientID = &id
	}
	if projectID.Valid {
		id := int(projectID.Int64)
		pendingDB.ProjectID = &id
	}

	return pendingDB, nil
}

// UpdatePendingDatabaseStatus обновляет статус индексации
func (db *ServiceDB) UpdatePendingDatabaseStatus(id int, status string, errorMessage string) error {
	query := `
		UPDATE pending_databases
		SET indexing_status = ?,
		    error_message = ?,
		    indexing_started_at = CASE WHEN ? = 'indexing' AND indexing_started_at IS NULL THEN CURRENT_TIMESTAMP ELSE indexing_started_at END,
		    indexing_completed_at = CASE WHEN ? IN ('completed', 'failed') THEN CURRENT_TIMESTAMP ELSE indexing_completed_at END
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, status, errorMessage, status, status, id)
	if err != nil {
		return fmt.Errorf("failed to update pending database status: %w", err)
	}
	return nil
}

// BindPendingDatabaseToProject привязывает pending database к проекту
func (db *ServiceDB) BindPendingDatabaseToProject(id, clientID, projectID int, newFilePath string, movedToUploads bool) error {
	query := `
		UPDATE pending_databases
		SET client_id = ?,
		    project_id = ?,
		    file_path = ?,
		    moved_to_uploads = ?,
		    original_path = CASE WHEN ? = TRUE THEN file_path ELSE original_path END
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, clientID, projectID, newFilePath, movedToUploads, movedToUploads, id)
	if err != nil {
		return fmt.Errorf("failed to bind pending database to project: %w", err)
	}
	return nil
}

// DeletePendingDatabase удаляет pending database
func (db *ServiceDB) DeletePendingDatabase(id int) error {
	query := `DELETE FROM pending_databases WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete pending database: %w", err)
	}
	return nil
}

// CleanupOldPendingDatabases удаляет старые pending databases (старше указанного количества дней)
func (db *ServiceDB) CleanupOldPendingDatabases(daysOld int) (int, error) {
	cutoffDate := time.Now().AddDate(0, 0, -daysOld)

	query := `
		DELETE FROM pending_databases 
		WHERE detected_at < ? 
		AND indexing_status = 'pending'
		AND client_id IS NULL
		AND project_id IS NULL
	`

	result, err := db.conn.Exec(query, cutoffDate)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old pending databases: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(deleted), nil
}

// NormalizedCounterparty структура нормализованного контрагента из БД
type NormalizedCounterparty struct {
	ID                   int       `json:"id"`
	ClientProjectID      int       `json:"client_project_id"`
	SourceReference      string    `json:"source_reference"`
	SourceName           string    `json:"source_name"`
	NormalizedName       string    `json:"normalized_name"`
	TaxID                string    `json:"tax_id"`
	KPP                  string    `json:"kpp"`
	BIN                  string    `json:"bin"`
	LegalAddress         string    `json:"legal_address"`
	PostalAddress        string    `json:"postal_address"`
	ContactPhone         string    `json:"contact_phone"`
	ContactEmail         string    `json:"contact_email"`
	ContactPerson        string    `json:"contact_person"`
	LegalForm            string    `json:"legal_form"`
	BankName             string    `json:"bank_name"`
	BankAccount          string    `json:"bank_account"`
	CorrespondentAccount string    `json:"correspondent_account"`
	BIK                  string    `json:"bik"`
	BenchmarkID          *int      `json:"benchmark_id"`
	QualityScore         float64   `json:"quality_score"`
	EnrichmentApplied    bool      `json:"enrichment_applied"`
	SourceEnrichment     string    `json:"source_enrichment"` // Источник нормализации: Adata.kz, Dadata.ru, gisp.gov.ru
	SourceDatabase       string    `json:"source_database"`
	Subcategory          string    `json:"subcategory"` // Подкатегория (например, "производитель")
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

// DatabaseSource представляет источник данных контрагента из конкретной базы данных
type DatabaseSource struct {
	DatabaseID      int    `json:"database_id"`
	DatabaseName    string `json:"database_name"`
	SourceReference string `json:"source_reference,omitempty"`
	SourceName      string `json:"source_name,omitempty"`
}

// UnifiedCounterparty объединенная структура контрагента из всех источников
type UnifiedCounterparty struct {
	// Общие поля
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Source       string `json:"source"` // "database" или "normalized"
	ProjectID    int    `json:"project_id"`
	ProjectName  string `json:"project_name"`
	DatabaseID   *int   `json:"database_id,omitempty"`
	DatabaseName string `json:"database_name,omitempty"`

	// Поля из исходных баз (CatalogItem)
	Reference  string `json:"reference,omitempty"`
	Code       string `json:"code,omitempty"`
	Attributes string `json:"attributes,omitempty"`

	// Поля из нормализованных записей
	NormalizedName  string `json:"normalized_name,omitempty"`
	SourceName      string `json:"source_name,omitempty"`
	SourceReference string `json:"source_reference,omitempty"`

	// Общие поля для обоих типов
	TaxID         string   `json:"tax_id,omitempty"`
	KPP           string   `json:"kpp,omitempty"`
	BIN           string   `json:"bin,omitempty"`
	LegalAddress  string   `json:"legal_address,omitempty"`
	PostalAddress string   `json:"postal_address,omitempty"`
	ContactPhone  string   `json:"contact_phone,omitempty"`
	ContactEmail  string   `json:"contact_email,omitempty"`
	ContactPerson string   `json:"contact_person,omitempty"`
	QualityScore  *float64 `json:"quality_score,omitempty"`

	// Связи с базами данных (many-to-many)
	SourceDatabases []DatabaseSource `json:"source_databases,omitempty"`
}

// SaveNormalizedCounterparty сохраняет нормализованного контрагента
func (db *ServiceDB) SaveNormalizedCounterparty(
	projectID int,
	sourceReference, sourceName, normalizedName string,
	taxID, kpp, bin, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm,
	bankName, bankAccount, correspondentAccount, bik string,
	benchmarkID int,
	qualityScore float64,
	enrichmentApplied bool,
	sourceEnrichment, sourceDatabase, subcategory string,
) error {
	// Проверяем существование таблицы перед сохранением (с защитой от race condition)
	db.tableCreateMutex.Lock()
	var tableExists bool
	err := db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='normalized_counterparties'
		)
	`).Scan(&tableExists)
	if err != nil {
		db.tableCreateMutex.Unlock()
		return fmt.Errorf("failed to check table existence: %w", err)
	}

	if !tableExists {
		// Таблица не существует, пытаемся создать её
		if err := CreateNormalizedCounterpartiesTable(db.conn); err != nil {
			db.tableCreateMutex.Unlock()
			return fmt.Errorf("table normalized_counterparties does not exist and failed to create: %w", err)
		}
	}
	db.tableCreateMutex.Unlock()

	var benchmarkIDValue interface{}
	if benchmarkID > 0 {
		benchmarkIDValue = benchmarkID
	} else {
		benchmarkIDValue = nil
	}

	// Используем INSERT с ON CONFLICT для обновления существующих записей по (client_project_id, source_reference)
	// Отладочный вывод для диагностики проблем с сохранением
	if sourceReference == "" {
		// В SQLite UNIQUE constraint не работает для NULL значений, поэтому пустые строки могут вызывать проблемы
		// Используем специальное значение для пустых reference
		sourceReference = fmt.Sprintf("__empty_ref_%d__", projectID)
	}

	query := `
		INSERT INTO normalized_counterparties
		(client_project_id, source_reference, source_name, normalized_name,
		 tax_id, kpp, bin, legal_address, postal_address, contact_phone, contact_email,
		 contact_person, legal_form, bank_name, bank_account, correspondent_account, bik,
		 benchmark_id, quality_score, enrichment_applied, source_enrichment, source_database, subcategory, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(client_project_id, source_reference) DO UPDATE SET
			source_name = excluded.source_name,
			normalized_name = excluded.normalized_name,
			tax_id = excluded.tax_id,
			kpp = excluded.kpp,
			bin = excluded.bin,
			legal_address = excluded.legal_address,
			postal_address = excluded.postal_address,
			contact_phone = excluded.contact_phone,
			contact_email = excluded.contact_email,
			contact_person = excluded.contact_person,
			legal_form = excluded.legal_form,
			bank_name = excluded.bank_name,
			bank_account = excluded.bank_account,
			correspondent_account = excluded.correspondent_account,
			bik = excluded.bik,
			benchmark_id = excluded.benchmark_id,
			quality_score = excluded.quality_score,
			enrichment_applied = excluded.enrichment_applied,
			source_enrichment = excluded.source_enrichment,
			source_database = excluded.source_database,
			subcategory = excluded.subcategory,
			updated_at = CURRENT_TIMESTAMP
	`

	result, err := db.conn.Exec(query,
		projectID, sourceReference, sourceName, normalizedName,
		taxID, kpp, bin, legalAddress, postalAddress, contactPhone, contactEmail,
		contactPerson, legalForm, bankName, bankAccount, correspondentAccount, bik,
		benchmarkIDValue, qualityScore, enrichmentApplied, sourceEnrichment, sourceDatabase, subcategory,
	)
	if err != nil {
		return fmt.Errorf("failed to save normalized counterparty: %w", err)
	}

	// Получаем ID сохраненного контрагента
	var counterpartyID int64
	if id, err := result.LastInsertId(); err == nil {
		counterpartyID = id
	} else {
		// Если не удалось получить ID через LastInsertId (например, при UPDATE),
		// получаем его через запрос
		err := db.conn.QueryRow(`
			SELECT id FROM normalized_counterparties
			WHERE client_project_id = ? AND source_reference = ?
		`, projectID, sourceReference).Scan(&counterpartyID)
		if err != nil {
			// Если не удалось получить ID, пропускаем создание связи
			return nil
		}
	}

	// Создаем связи с базами данных, если указан sourceDatabase
	if sourceDatabase != "" {
		dbNames := ParseDatabaseNames(sourceDatabase)
		for _, dbName := range dbNames {
			if dbName == "" {
				continue
			}

			// Ищем базу данных по имени в рамках проекта
			var databaseID int
			err := db.conn.QueryRow(`
				SELECT id FROM project_databases
				WHERE client_project_id = ? AND name = ?
				LIMIT 1
			`, projectID, dbName).Scan(&databaseID)

			if err == sql.ErrNoRows {
				// База данных не найдена, пробуем найти по file_path
				err = db.conn.QueryRow(`
					SELECT id FROM project_databases
					WHERE client_project_id = ? AND (file_path LIKE ? OR file_path LIKE ?)
					LIMIT 1
				`, projectID, "%"+dbName+"%", dbName).Scan(&databaseID)
			}

			if err == nil {
				// Создаем связь
				_ = db.SaveCounterpartyDatabaseLink(int(counterpartyID), databaseID, sourceReference, sourceName)
			}
		}
	}

	return nil
}

// NormalizedCounterpartyBatchItem элемент для batch сохранения контрагентов
type NormalizedCounterpartyBatchItem struct {
	SourceReference      string
	SourceName           string
	NormalizedName       string
	INN                  string
	KPP                  string
	BIN                  string
	LegalAddress         string
	PostalAddress        string
	ContactPhone         string
	ContactEmail         string
	ContactPerson        string
	LegalForm            string
	BankName             string
	BankAccount          string
	CorrespondentAccount string
	BIK                  string
	BenchmarkID          int
	QualityScore         float64
	EnrichmentApplied    bool
	SourceEnrichment     string
	Subcategory          string
	SourceDatabase       string
}

// SaveNormalizedCounterpartiesBatch сохраняет пакет нормализованных контрагентов в одной транзакции
// Это оптимизация для устранения N+1 запросов - вместо N отдельных INSERT выполняется один batch insert
func (db *ServiceDB) SaveNormalizedCounterpartiesBatch(
	projectID int,
	items []*NormalizedCounterpartyBatchItem,
) error {
	if len(items) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Подготавливаем statement для batch insert
	stmt, err := tx.Prepare(`
		INSERT OR REPLACE INTO normalized_counterparties
		(client_project_id, source_reference, source_name, normalized_name,
		 tax_id, kpp, bin, legal_address, postal_address, contact_phone, contact_email,
		 contact_person, legal_form, bank_name, bank_account, correspondent_account, bik,
		 benchmark_id, quality_score, enrichment_applied, source_enrichment, source_database, subcategory, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Выполняем batch insert
	for i, item := range items {
		var benchmarkIDValue interface{}
		if item.BenchmarkID > 0 {
			benchmarkIDValue = item.BenchmarkID
		} else {
			benchmarkIDValue = nil
		}

		_, err = stmt.Exec(
			projectID,
			item.SourceReference,
			item.SourceName,
			item.NormalizedName,
			item.INN,
			item.KPP,
			item.BIN,
			item.LegalAddress,
			item.PostalAddress,
			item.ContactPhone,
			item.ContactEmail,
			item.ContactPerson,
			item.LegalForm,
			item.BankName,
			item.BankAccount,
			item.CorrespondentAccount,
			item.BIK,
			benchmarkIDValue,
			item.QualityScore,
			item.EnrichmentApplied,
			item.SourceEnrichment,
			item.SourceDatabase,
			item.Subcategory,
		)
		if err != nil {
			return fmt.Errorf("failed to insert normalized counterparty at index %d: %w", i, err)
		}
	}

	// Подтверждаем транзакцию
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetNormalizedCounterpartiesBySourceReferences получает уже нормализованных контрагентов по списку source_reference
// Возвращает map[source_reference] = true для быстрой проверки
func (db *ServiceDB) GetNormalizedCounterpartiesBySourceReferences(projectID int, sourceReferences []string) (map[string]bool, error) {
	if len(sourceReferences) == 0 {
		return make(map[string]bool), nil
	}

	// Создаем плейсхолдеры для IN запроса
	placeholders := strings.Repeat("?,", len(sourceReferences)-1) + "?"

	query := fmt.Sprintf(`
		SELECT DISTINCT source_reference
		FROM normalized_counterparties
		WHERE client_project_id = ? AND source_reference IN (%s)
	`, placeholders)

	args := make([]interface{}, 0, len(sourceReferences)+1)
	args = append(args, projectID)
	for _, ref := range sourceReferences {
		args = append(args, ref)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get normalized counterparties by source references: %w", err)
	}
	defer rows.Close()

	normalized := make(map[string]bool)
	for rows.Next() {
		var sourceRef string
		if err := rows.Scan(&sourceRef); err != nil {
			return nil, fmt.Errorf("failed to scan source reference: %w", err)
		}
		normalized[sourceRef] = true
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating normalized counterparties: %w", err)
	}

	return normalized, nil
}

// ResumeNormalizationSession возобновляет остановленную сессию нормализации
func (db *ServiceDB) ResumeNormalizationSession(sessionID int) error {
	query := `
		UPDATE normalization_sessions
		SET status = 'running', finished_at = NULL, last_activity_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'stopped'
	`

	result, err := db.conn.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to resume normalization session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session %d not found or not stopped", sessionID)
	}

	return nil
}

// UpdateNormalizedCounterparty обновляет нормализованного контрагента
func (db *ServiceDB) UpdateNormalizedCounterparty(
	id int,
	normalizedName string,
	taxID, kpp, bin, legalAddress, postalAddress, contactPhone, contactEmail, contactPerson, legalForm,
	bankName, bankAccount, correspondentAccount, bik string,
	qualityScore float64,
	sourceEnrichment, subcategory string,
) error {
	query := `
		UPDATE normalized_counterparties
		SET normalized_name = ?, tax_id = ?, kpp = ?, bin = ?,
		    legal_address = ?, postal_address = ?, contact_phone = ?, contact_email = ?,
		    contact_person = ?, legal_form = ?, bank_name = ?, bank_account = ?,
		    correspondent_account = ?, bik = ?, quality_score = ?,
		    source_enrichment = ?, subcategory = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err := db.conn.Exec(query,
		normalizedName, taxID, kpp, bin,
		legalAddress, postalAddress, contactPhone, contactEmail,
		contactPerson, legalForm, bankName, bankAccount,
		correspondentAccount, bik, qualityScore,
		sourceEnrichment, subcategory, id,
	)
	if err != nil {
		return fmt.Errorf("failed to update normalized counterparty: %w", err)
	}

	return nil
}

// GetNormalizedCounterparty получает контрагента по ID
func (db *ServiceDB) GetNormalizedCounterparty(id int) (*NormalizedCounterparty, error) {
	query := `
		SELECT id, client_project_id, source_reference, source_name, normalized_name,
		       tax_id, kpp, bin, legal_address, postal_address, contact_phone, contact_email,
		       contact_person, legal_form, bank_name, bank_account, correspondent_account, bik,
		       benchmark_id, quality_score, enrichment_applied, COALESCE(source_enrichment, ''), source_database, 
		       COALESCE(subcategory, '') as subcategory, created_at, updated_at
		FROM normalized_counterparties
		WHERE id = ?
	`

	cp := &NormalizedCounterparty{}
	var benchmarkID sql.NullInt64

	err := db.conn.QueryRow(query, id).Scan(
		&cp.ID, &cp.ClientProjectID, &cp.SourceReference, &cp.SourceName, &cp.NormalizedName,
		&cp.TaxID, &cp.KPP, &cp.BIN, &cp.LegalAddress, &cp.PostalAddress,
		&cp.ContactPhone, &cp.ContactEmail, &cp.ContactPerson, &cp.LegalForm,
		&cp.BankName, &cp.BankAccount, &cp.CorrespondentAccount, &cp.BIK,
		&benchmarkID, &cp.QualityScore, &cp.EnrichmentApplied, &cp.SourceEnrichment, &cp.SourceDatabase,
		&cp.Subcategory, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("counterparty not found")
		}
		return nil, fmt.Errorf("failed to get normalized counterparty: %w", err)
	}

	if benchmarkID.Valid {
		id := int(benchmarkID.Int64)
		cp.BenchmarkID = &id
	}

	return cp, nil
}

// DeleteNormalizedCounterparty удаляет нормализованного контрагента
func (db *ServiceDB) DeleteNormalizedCounterparty(id int) error {
	query := `DELETE FROM normalized_counterparties WHERE id = ?`

	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete normalized counterparty: %w", err)
	}

	return nil
}

// GetNormalizedCounterpartiesByClient получает всех контрагентов клиента по всем проектам
// search - поисковый запрос (поиск по имени, ИНН, БИН, адресу, email, телефону)
// enrichment - фильтр по источнику обогащения (пустая строка = все, "none" = без обогащения, иначе конкретное значение)
// subcategory - фильтр по подкатегории (пустая строка = все, "none" = без подкатегории, "manufacturer" = производитель, иначе конкретное значение)
func (db *ServiceDB) GetNormalizedCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, enrichment, subcategory string) ([]*NormalizedCounterparty, []*ClientProject, int, error) {
	// Получаем проекты клиента
	var projects []*ClientProject
	var err error

	if projectID != nil {
		// Если указан конкретный проект, получаем только его
		project, err := db.GetClientProject(*projectID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to get project: %w", err)
		}
		if project.ClientID != clientID {
			return nil, nil, 0, fmt.Errorf("project does not belong to client")
		}
		projects = []*ClientProject{project}
	} else {
		// Получаем все проекты клиента
		projects, err = db.GetClientProjects(clientID)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to get projects: %w", err)
		}
	}

	if len(projects) == 0 {
		return []*NormalizedCounterparty{}, projects, 0, nil
	}

	// Собираем ID проектов
	projectIDs := make([]interface{}, len(projects))
	placeholders := make([]string, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
		placeholders[i] = "?"
	}

	// Формируем условия поиска
	whereConditions := []string{fmt.Sprintf("client_project_id IN (%s)", strings.Join(placeholders, ","))}
	args := append([]interface{}{}, projectIDs...)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		whereConditions = append(whereConditions, `(
			LOWER(nc.normalized_name) LIKE ? OR
			LOWER(nc.source_name) LIKE ? OR
			LOWER(nc.tax_id) LIKE ? OR
			LOWER(nc.bin) LIKE ? OR
			LOWER(nc.legal_address) LIKE ? OR
			LOWER(nc.postal_address) LIKE ? OR
			LOWER(nc.contact_email) LIKE ? OR
			LOWER(nc.contact_phone) LIKE ? OR
			LOWER(nc.contact_person) LIKE ?
		)`)
		// Добавляем паттерн для каждого поля поиска
		for i := 0; i < 9; i++ {
			args = append(args, searchPattern)
		}
	}

	// Фильтр по источнику обогащения
	if enrichment != "" {
		if enrichment == "none" {
			whereConditions = append(whereConditions, "(COALESCE(nc.source_enrichment, '') = '')")
		} else {
			whereConditions = append(whereConditions, "COALESCE(nc.source_enrichment, '') = ?")
			args = append(args, enrichment)
		}
	}

	// Фильтр по подкатегории
	if subcategory != "" {
		if subcategory == "none" {
			whereConditions = append(whereConditions, "(COALESCE(nc.subcategory, '') = '')")
		} else if subcategory == "manufacturer" {
			whereConditions = append(whereConditions, "COALESCE(nc.subcategory, '') = ?")
			args = append(args, "производитель")
		} else {
			whereConditions = append(whereConditions, "COALESCE(nc.subcategory, '') = ?")
			args = append(args, subcategory)
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Получаем общее количество
	var totalCount int
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*) FROM normalized_counterparties nc WHERE %s`,
		whereClause,
	)
	err = db.conn.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Получаем записи с пагинацией
	query := fmt.Sprintf(`
		SELECT nc.id, nc.client_project_id, nc.source_reference, nc.source_name, nc.normalized_name,
		       nc.tax_id, nc.kpp, nc.bin, nc.legal_address, nc.postal_address, nc.contact_phone, nc.contact_email,
		       nc.contact_person, nc.legal_form, nc.bank_name, nc.bank_account, nc.correspondent_account, nc.bik,
		       nc.benchmark_id, nc.quality_score, nc.enrichment_applied, COALESCE(nc.source_enrichment, ''), nc.source_database, 
		       COALESCE(nc.subcategory, '') as subcategory, nc.created_at, nc.updated_at
		FROM normalized_counterparties nc
		WHERE %s
		ORDER BY nc.normalized_name, nc.created_at DESC
	`, whereClause)

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, nil, 0, fmt.Errorf("failed to query normalized counterparties: %w", err)
	}
	defer rows.Close()

	var counterparties []*NormalizedCounterparty
	for rows.Next() {
		cp := &NormalizedCounterparty{}
		var benchmarkID sql.NullInt64

		err := rows.Scan(
			&cp.ID, &cp.ClientProjectID, &cp.SourceReference, &cp.SourceName, &cp.NormalizedName,
			&cp.TaxID, &cp.KPP, &cp.BIN, &cp.LegalAddress, &cp.PostalAddress,
			&cp.ContactPhone, &cp.ContactEmail, &cp.ContactPerson, &cp.LegalForm,
			&cp.BankName, &cp.BankAccount, &cp.CorrespondentAccount, &cp.BIK,
			&benchmarkID, &cp.QualityScore, &cp.EnrichmentApplied, &cp.SourceEnrichment, &cp.SourceDatabase,
			&cp.Subcategory, &cp.CreatedAt, &cp.UpdatedAt,
		)
		if err != nil {
			return nil, nil, 0, fmt.Errorf("failed to scan normalized counterparty: %w", err)
		}

		if benchmarkID.Valid {
			id := int(benchmarkID.Int64)
			cp.BenchmarkID = &id
		}

		counterparties = append(counterparties, cp)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, 0, fmt.Errorf("error iterating normalized counterparties: %w", err)
	}

	return counterparties, projects, totalCount, nil
}

// GetNormalizedCounterparties получает нормализованных контрагентов проекта
// search - поисковый запрос (поиск по имени, ИНН, БИН, адресу, email, телефону)
// enrichment - фильтр по источнику обогащения (пустая строка = все, "none" = без обогащения, иначе конкретное значение)
// subcategory - фильтр по подкатегории (пустая строка = все, "none" = без подкатегории, "manufacturer" = производитель, иначе конкретное значение)
func (db *ServiceDB) GetNormalizedCounterparties(projectID int, offset, limit int, search, enrichment, subcategory string) ([]*NormalizedCounterparty, int, error) {
	// Формируем условия поиска
	whereConditions := []string{"client_project_id = ?"}
	args := []interface{}{projectID}

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		whereConditions = append(whereConditions, `(
			LOWER(normalized_name) LIKE ? OR
			LOWER(source_name) LIKE ? OR
			LOWER(tax_id) LIKE ? OR
			LOWER(bin) LIKE ? OR
			LOWER(legal_address) LIKE ? OR
			LOWER(postal_address) LIKE ? OR
			LOWER(contact_email) LIKE ? OR
			LOWER(contact_phone) LIKE ? OR
			LOWER(contact_person) LIKE ?
		)`)
		// Добавляем паттерн для каждого поля поиска
		for i := 0; i < 9; i++ {
			args = append(args, searchPattern)
		}
	}

	// Фильтр по источнику обогащения
	if enrichment != "" {
		if enrichment == "none" {
			whereConditions = append(whereConditions, "(COALESCE(source_enrichment, '') = '')")
		} else {
			whereConditions = append(whereConditions, "COALESCE(source_enrichment, '') = ?")
			args = append(args, enrichment)
		}
	}

	// Фильтр по подкатегории
	if subcategory != "" {
		if subcategory == "none" {
			whereConditions = append(whereConditions, "(COALESCE(subcategory, '') = '')")
		} else if subcategory == "manufacturer" {
			whereConditions = append(whereConditions, "COALESCE(subcategory, '') = ?")
			args = append(args, "производитель")
		} else {
			whereConditions = append(whereConditions, "COALESCE(subcategory, '') = ?")
			args = append(args, subcategory)
		}
	}

	whereClause := strings.Join(whereConditions, " AND ")

	// Получаем общее количество
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM normalized_counterparties WHERE %s`, whereClause)
	var totalCount int
	err := db.conn.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %w", err)
	}

	// Получаем записи с пагинацией
	query := fmt.Sprintf(`
		SELECT id, client_project_id, source_reference, source_name, normalized_name,
		       tax_id, kpp, bin, legal_address, postal_address, contact_phone, contact_email,
		       contact_person, legal_form, bank_name, bank_account, correspondent_account, bik,
		       benchmark_id, quality_score, enrichment_applied, COALESCE(source_enrichment, ''), source_database, 
		       COALESCE(subcategory, '') as subcategory, created_at, updated_at
		FROM normalized_counterparties
		WHERE %s
		ORDER BY normalized_name, created_at DESC
	`, whereClause)

	if limit > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, limit, offset)
	}

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query normalized counterparties: %w", err)
	}
	defer rows.Close()

	var counterparties []*NormalizedCounterparty
	for rows.Next() {
		cp := &NormalizedCounterparty{}
		var benchmarkID sql.NullInt64

		err := rows.Scan(
			&cp.ID, &cp.ClientProjectID, &cp.SourceReference, &cp.SourceName, &cp.NormalizedName,
			&cp.TaxID, &cp.KPP, &cp.BIN, &cp.LegalAddress, &cp.PostalAddress,
			&cp.ContactPhone, &cp.ContactEmail, &cp.ContactPerson, &cp.LegalForm,
			&cp.BankName, &cp.BankAccount, &cp.CorrespondentAccount, &cp.BIK,
			&benchmarkID, &cp.QualityScore, &cp.EnrichmentApplied, &cp.SourceEnrichment, &cp.SourceDatabase,
			&cp.Subcategory, &cp.CreatedAt, &cp.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan normalized counterparty: %w", err)
		}

		if benchmarkID.Valid {
			id := int(benchmarkID.Int64)
			cp.BenchmarkID = &id
		}

		counterparties = append(counterparties, cp)
	}

	if err = rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating normalized counterparties: %w", err)
	}

	return counterparties, totalCount, nil
}

// GetAllCounterpartiesByClientResult результат получения всех контрагентов с метаданными
type GetAllCounterpartiesByClientResult struct {
	Counterparties []*UnifiedCounterparty
	Projects       []*ClientProject
	TotalCount     int
	Stats          *CounterpartiesStats
}

// CounterpartiesStats статистика по контрагентам
type CounterpartiesStats struct {
	TotalFromDatabase  int     `json:"total_from_database"`
	TotalNormalized    int     `json:"total_normalized"`
	TotalWithQuality   int     `json:"total_with_quality"`
	AverageQuality     float64 `json:"average_quality,omitempty"`
	DatabasesProcessed int     `json:"databases_processed,omitempty"`
	ProjectsProcessed  int     `json:"projects_processed,omitempty"`
	ProcessingTimeMs   int64   `json:"processing_time_ms,omitempty"`
}

// CounterpartyStreamOptions описывает параметры потоковой выдачи контрагентов.
type CounterpartyStreamOptions struct {
	ClientID           int
	ProjectID          *int
	Offset             int
	Limit              int
	Search             string
	Source             string
	SortBy             string
	Order              string
	MinQuality         *float64
	MaxQuality         *float64
	BatchSize          int
	ApplyQualityFilter bool
	ApplyPagination    bool
}

var ErrCounterpartyStreamLimitReached = errors.New("counterparty stream limit reached")

// GetAllCounterpartiesByClient получает всех контрагентов клиента из всех баз данных и нормализованных записей
// search - поисковый запрос (поиск по имени, ИНН, БИН)
// source - фильтр по источнику: "database", "normalized" или пусто (все)
// sortBy - поле для сортировки: "name", "quality", "source" или пусто (по умолчанию)
// order - порядок сортировки: "asc", "desc" или пусто (по умолчанию)
func (db *ServiceDB) GetAllCounterpartiesByClient(clientID int, projectID *int, offset, limit int, search, source, sortBy, order string, minQuality, maxQuality *float64) (*GetAllCounterpartiesByClientResult, error) {
	startTime := time.Now()

	streamOpts := &CounterpartyStreamOptions{
		ClientID:           clientID,
		ProjectID:          projectID,
		Search:             search,
		Source:             source,
		MinQuality:         minQuality,
		MaxQuality:         maxQuality,
		BatchSize:          1000,
		ApplyQualityFilter: false,
		ApplyPagination:    false,
		// Offset/Limit применяются позже после сортировки
	}

	var allCounterparties []*UnifiedCounterparty
	consumer := func(batch []*UnifiedCounterparty) error {
		allCounterparties = append(allCounterparties, batch...)
		return nil
	}

	stats, projects, _, err := db.StreamAllCounterpartiesByClient(context.Background(), streamOpts, consumer)
	if err != nil && !errors.Is(err, ErrCounterpartyStreamLimitReached) {
		return nil, err
	}

	if stats == nil {
		stats = &CounterpartiesStats{}
	}
	if projects == nil {
		projects = []*ClientProject{}
	}

	allCounterparties = append([]*UnifiedCounterparty(nil), allCounterparties...)

	// 3. Применяем фильтры по качеству
	if minQuality != nil || maxQuality != nil {
		filtered := make([]*UnifiedCounterparty, 0, len(allCounterparties))
		for _, cp := range allCounterparties {
			// Пропускаем записи без качества, если указан minQuality
			if cp.QualityScore == nil {
				if minQuality != nil {
					continue // Пропускаем записи без качества, если требуется минимальное качество
				}
				filtered = append(filtered, cp)
				continue
			}

			quality := *cp.QualityScore

			// Проверяем минимальное качество
			if minQuality != nil && quality < *minQuality {
				continue
			}

			// Проверяем максимальное качество
			if maxQuality != nil && quality > *maxQuality {
				continue
			}

			filtered = append(filtered, cp)
		}
		allCounterparties = filtered
	}

	// 4. Сортируем объединенный список
	// Определяем порядок сортировки
	isDesc := strings.ToLower(order) == "desc"

	// Определяем поле сортировки
	sortField := strings.ToLower(sortBy)
	if sortField == "" {
		sortField = "default" // Сортировка по умолчанию
	}

	sort.Slice(allCounterparties, func(i, j int) bool {
		var result bool

		switch sortField {
		case "quality":
			// Сортировка по качеству
			if allCounterparties[i].QualityScore == nil && allCounterparties[j].QualityScore == nil {
				result = false
			} else if allCounterparties[i].QualityScore == nil {
				result = false // Записи без качества внизу
			} else if allCounterparties[j].QualityScore == nil {
				result = true // Записи с качеством вверху
			} else {
				result = *allCounterparties[i].QualityScore > *allCounterparties[j].QualityScore
			}
		case "source":
			// Сортировка по источнику
			result = allCounterparties[i].Source < allCounterparties[j].Source
		case "name":
			// Сортировка по имени
			nameI := strings.ToLower(allCounterparties[i].Name)
			nameJ := strings.ToLower(allCounterparties[j].Name)
			result = nameI < nameJ
		case "id":
			// Сортировка по ID
			result = allCounterparties[i].ID < allCounterparties[j].ID
		default:
			// Сортировка по умолчанию: качество -> имя -> ID
			if allCounterparties[i].QualityScore != nil && allCounterparties[j].QualityScore == nil {
				result = true
			} else if allCounterparties[i].QualityScore == nil && allCounterparties[j].QualityScore != nil {
				result = false
			} else if allCounterparties[i].QualityScore != nil && allCounterparties[j].QualityScore != nil {
				if *allCounterparties[i].QualityScore != *allCounterparties[j].QualityScore {
					result = *allCounterparties[i].QualityScore > *allCounterparties[j].QualityScore
				} else {
					nameI := strings.ToLower(allCounterparties[i].Name)
					nameJ := strings.ToLower(allCounterparties[j].Name)
					if nameI != nameJ {
						result = nameI < nameJ
					} else {
						result = allCounterparties[i].ID < allCounterparties[j].ID
					}
				}
			} else {
				nameI := strings.ToLower(allCounterparties[i].Name)
				nameJ := strings.ToLower(allCounterparties[j].Name)
				if nameI != nameJ {
					result = nameI < nameJ
				} else {
					result = allCounterparties[i].ID < allCounterparties[j].ID
				}
			}
		}

		// Применяем порядок сортировки
		if isDesc {
			return !result
		}
		return result
	})

	// 5. Применяем пагинацию
	totalCount := len(allCounterparties)
	if totalCount == 0 {
		return &GetAllCounterpartiesByClientResult{
			Counterparties: []*UnifiedCounterparty{},
			Projects:       projects,
			TotalCount:     0,
			Stats:          stats,
		}, nil
	}

	start := offset
	end := offset + limit
	if limit > 0 {
		if start > totalCount {
			start = totalCount
		}
		if end > totalCount {
			end = totalCount
		}
		if start < end {
			allCounterparties = allCounterparties[start:end]
		} else {
			allCounterparties = []*UnifiedCounterparty{}
		}
	} else {
		if start > totalCount {
			allCounterparties = []*UnifiedCounterparty{}
		} else {
			allCounterparties = allCounterparties[start:]
		}
	}

	// Вычисляем время обработки
	processingTime := time.Since(startTime)
	stats.ProcessingTimeMs = processingTime.Milliseconds()

	return &GetAllCounterpartiesByClientResult{
		Counterparties: allCounterparties,
		Projects:       projects,
		TotalCount:     totalCount,
		Stats:          stats,
	}, nil
}

// StreamAllCounterpartiesByClient потоково передает контрагентов батчами потребителю
func (db *ServiceDB) StreamAllCounterpartiesByClient(ctx context.Context, opts *CounterpartyStreamOptions, consumer func([]*UnifiedCounterparty) error) (*CounterpartiesStats, []*ClientProject, int, error) {
	if opts == nil {
		return nil, nil, 0, fmt.Errorf("stream options are required")
	}
	if consumer == nil {
		return nil, nil, 0, fmt.Errorf("stream consumer is required")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.BatchSize <= 0 {
		opts.BatchSize = 1000
	}

	start := time.Now()

	projects, err := db.getProjectsForClient(opts.ClientID, opts.ProjectID)
	if err != nil {
		return nil, nil, 0, err
	}

	stats := &CounterpartiesStats{
		ProjectsProcessed: len(projects),
	}
	if len(projects) == 0 {
		stats.ProcessingTimeMs = time.Since(start).Milliseconds()
		return stats, []*ClientProject{}, 0, nil
	}

	acc := newCounterpartyStreamAccumulator(ctx, opts, stats, consumer)

	limitReached := false

	if opts.Source == "" || opts.Source == "database" {
		if err := db.streamCounterpartiesFromDatabases(ctx, projects, opts, acc); err != nil {
			if errors.Is(err, ErrCounterpartyStreamLimitReached) {
				limitReached = true
			} else {
				return nil, nil, 0, err
			}
		}
	}

	if !limitReached && (opts.Source == "" || opts.Source == "normalized") {
		if err := db.streamNormalizedCounterparties(ctx, projects, opts, acc); err != nil {
			if !errors.Is(err, ErrCounterpartyStreamLimitReached) {
				return nil, nil, 0, err
			}
		}
	}

	if err := acc.finalize(); err != nil {
		if errors.Is(err, ErrCounterpartyStreamLimitReached) {
			// Нормально завершаем при достижении лимита
		} else {
			return nil, nil, 0, err
		}
	}

	stats.AverageQuality = acc.averageQuality()
	stats.ProcessingTimeMs = time.Since(start).Milliseconds()

	return stats, projects, acc.total(), nil
}

func (db *ServiceDB) getProjectsForClient(clientID int, projectID *int) ([]*ClientProject, error) {
	if projectID != nil {
		project, err := db.GetClientProject(*projectID)
		if err != nil {
			return nil, fmt.Errorf("failed to get project: %w", err)
		}
		if project.ClientID != clientID {
			return nil, fmt.Errorf("project does not belong to client")
		}
		return []*ClientProject{project}, nil
	}

	projects, err := db.GetClientProjects(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get projects: %w", err)
	}
	return projects, nil
}

func (db *ServiceDB) streamCounterpartiesFromDatabases(ctx context.Context, projects []*ClientProject, opts *CounterpartyStreamOptions, acc *counterpartyStreamAccumulator) error {
	type dbTask struct {
		project *ClientProject
		dbInfo  *ProjectDatabase
	}

	const catalogBatchSize = 2000

	var tasks []dbTask
	for _, project := range projects {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		databases, err := db.GetProjectDatabases(project.ID, true)
		if err != nil {
			log.Printf("Failed to get databases for project %d: %v", project.ID, err)
			continue
		}
		for _, info := range databases {
			if info.IsActive {
				tasks = append(tasks, dbTask{project: project, dbInfo: info})
			}
		}
	}

	for _, task := range tasks {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := db.streamSingleDatabase(ctx, task.project, task.dbInfo, opts, acc, catalogBatchSize); err != nil {
			if errors.Is(err, ErrCounterpartyStreamLimitReached) {
				return err
			}
			log.Printf("Failed to stream database %s: %v", task.dbInfo.FilePath, err)
		}
	}
	return nil
}

func (db *ServiceDB) streamSingleDatabase(ctx context.Context, project *ClientProject, dbInfo *ProjectDatabase, opts *CounterpartyStreamOptions, acc *counterpartyStreamAccumulator, batchSize int) error {
	sourceDB, err := NewDB(dbInfo.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open database %s: %w", dbInfo.FilePath, err)
	}
	defer sourceDB.Close()

	acc.stats.DatabasesProcessed++

	uploads, err := sourceDB.GetAllUploads()
	if err != nil {
		return fmt.Errorf("failed to get uploads from %s: %w", dbInfo.FilePath, err)
	}

	for _, upload := range uploads {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := db.streamCatalogItems(ctx, sourceDB, upload.ID, project, dbInfo, opts, acc, batchSize); err != nil {
			if errors.Is(err, ErrCounterpartyStreamLimitReached) {
				return err
			}
			log.Printf("Failed to stream catalog items from upload %d: %v", upload.ID, err)
		}
	}
	return nil
}

func (db *ServiceDB) streamCatalogItems(ctx context.Context, sourceDB *DB, uploadID int, project *ClientProject, dbInfo *ProjectDatabase, opts *CounterpartyStreamOptions, acc *counterpartyStreamAccumulator, batchSize int) error {
	offset := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		items, _, err := sourceDB.GetCatalogItemsByUpload(uploadID, []string{"Контрагенты"}, offset, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get catalog items: %w", err)
		}
		if len(items) == 0 {
			break
		}

		for _, item := range items {
			unified := db.catalogItemToUnified(item, project, dbInfo)
			if opts.Search != "" && !db.matchesSearch(unified, opts.Search) {
				continue
			}
			if err := acc.addCandidate(unified, false); err != nil {
				return err
			}
		}

		if len(items) < batchSize {
			break
		}
		offset += len(items)
	}
	return nil
}

func (db *ServiceDB) streamNormalizedCounterparties(ctx context.Context, projects []*ClientProject, opts *CounterpartyStreamOptions, acc *counterpartyStreamAccumulator) error {
	if len(projects) == 0 {
		return nil
	}

	whereClause, args := buildNormalizedWhereClause(projects, opts.Search)
	query := fmt.Sprintf(`
		SELECT nc.id, nc.client_project_id, nc.source_reference, nc.source_name, nc.normalized_name,
		       nc.tax_id, nc.kpp, nc.bin, nc.legal_address, nc.postal_address, nc.contact_phone, nc.contact_email,
		       nc.contact_person, nc.legal_form, nc.bank_name, nc.bank_account, nc.correspondent_account, nc.bik,
		       nc.benchmark_id, nc.quality_score, nc.enrichment_applied, COALESCE(nc.source_enrichment, ''), nc.source_database,
		       COALESCE(nc.subcategory, '') as subcategory, nc.created_at, nc.updated_at
		FROM normalized_counterparties nc
		WHERE %s
		ORDER BY nc.normalized_name, nc.created_at DESC
	`, whereClause)

	projectMap := make(map[int]*ClientProject, len(projects))
	for _, p := range projects {
		projectMap[p.ID] = p
	}

	pageSize := opts.BatchSize
	offset := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pageArgs := append([]interface{}{}, args...)
		pageArgs = append(pageArgs, pageSize, offset)
		rows, err := db.conn.QueryContext(ctx, query+" LIMIT ? OFFSET ?", pageArgs...)
		if err != nil {
			return fmt.Errorf("failed to query normalized counterparties: %w", err)
		}

		count := 0
		for rows.Next() {
			cp := &NormalizedCounterparty{}
			var benchmarkID sql.NullInt64
			if err := rows.Scan(
				&cp.ID, &cp.ClientProjectID, &cp.SourceReference, &cp.SourceName, &cp.NormalizedName,
				&cp.TaxID, &cp.KPP, &cp.BIN, &cp.LegalAddress, &cp.PostalAddress,
				&cp.ContactPhone, &cp.ContactEmail, &cp.ContactPerson, &cp.LegalForm,
				&cp.BankName, &cp.BankAccount, &cp.CorrespondentAccount, &cp.BIK,
				&benchmarkID, &cp.QualityScore, &cp.EnrichmentApplied, &cp.SourceEnrichment, &cp.SourceDatabase,
				&cp.Subcategory, &cp.CreatedAt, &cp.UpdatedAt,
			); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan normalized counterparty: %w", err)
			}
			if benchmarkID.Valid {
				id := int(benchmarkID.Int64)
				cp.BenchmarkID = &id
			}

			project := projectMap[cp.ClientProjectID]
			if project == nil {
				continue
			}

			unified := db.normalizedToUnified(cp, project)
			if err := acc.addCandidate(unified, true); err != nil {
				rows.Close()
				return err
			}
			count++
		}
		rows.Close()

		if count == 0 || count < pageSize {
			break
		}
		offset += count
	}
	return nil
}

func buildNormalizedWhereClause(projects []*ClientProject, search string) (string, []interface{}) {
	projectIDs := make([]interface{}, len(projects))
	placeholders := make([]string, len(projects))
	for i, p := range projects {
		projectIDs[i] = p.ID
		placeholders[i] = "?"
	}

	whereConditions := []string{fmt.Sprintf("client_project_id IN (%s)", strings.Join(placeholders, ","))}
	args := append([]interface{}{}, projectIDs...)

	if search != "" {
		searchPattern := "%" + strings.ToLower(search) + "%"
		whereConditions = append(whereConditions, `(
			LOWER(nc.normalized_name) LIKE ? OR
			LOWER(nc.source_name) LIKE ? OR
			LOWER(nc.tax_id) LIKE ? OR
			LOWER(nc.bin) LIKE ? OR
			LOWER(nc.legal_address) LIKE ? OR
			LOWER(nc.postal_address) LIKE ? OR
			LOWER(nc.contact_email) LIKE ? OR
			LOWER(nc.contact_phone) LIKE ? OR
			LOWER(nc.contact_person) LIKE ?
		)`)
		for i := 0; i < 9; i++ {
			args = append(args, searchPattern)
		}
	}

	return strings.Join(whereConditions, " AND "), args
}

type counterpartyStreamAccumulator struct {
	ctx             context.Context
	opts            *CounterpartyStreamOptions
	stats           *CounterpartiesStats
	consumer        func([]*UnifiedCounterparty) error
	batch           []*UnifiedCounterparty
	filteredCount   int
	emitted         int
	avgQualitySum   float64
	avgQualityCount int
}

func newCounterpartyStreamAccumulator(ctx context.Context, opts *CounterpartyStreamOptions, stats *CounterpartiesStats, consumer func([]*UnifiedCounterparty) error) *counterpartyStreamAccumulator {
	return &counterpartyStreamAccumulator{
		ctx:      ctx,
		opts:     opts,
		stats:    stats,
		consumer: consumer,
		batch:    make([]*UnifiedCounterparty, 0, opts.BatchSize),
	}
}

func (a *counterpartyStreamAccumulator) addCandidate(unified *UnifiedCounterparty, isNormalized bool) error {
	if err := a.ctx.Err(); err != nil {
		return err
	}

	if isNormalized {
		a.stats.TotalNormalized++
		if unified.QualityScore != nil {
			a.stats.TotalWithQuality++
		}
	} else {
		a.stats.TotalFromDatabase++
	}

	if unified.QualityScore != nil {
		a.avgQualitySum += *unified.QualityScore
		a.avgQualityCount++
	}

	if a.opts.ApplyQualityFilter {
		if !a.passesQuality(unified) {
			return nil
		}
	}

	a.filteredCount++

	if a.opts.ApplyPagination {
		if a.filteredCount <= a.opts.Offset {
			return nil
		}
		if a.opts.Limit > 0 && a.emitted >= a.opts.Limit {
			return ErrCounterpartyStreamLimitReached
		}
	}

	a.batch = append(a.batch, unified)
	a.emitted++

	if len(a.batch) >= a.opts.BatchSize {
		return a.flush()
	}
	return nil
}

func (a *counterpartyStreamAccumulator) passesQuality(unified *UnifiedCounterparty) bool {
	if a.opts.MinQuality == nil && a.opts.MaxQuality == nil {
		return true
	}
	if unified.QualityScore == nil {
		return a.opts.MinQuality == nil
	}
	value := *unified.QualityScore
	if a.opts.MinQuality != nil && value < *a.opts.MinQuality {
		return false
	}
	if a.opts.MaxQuality != nil && value > *a.opts.MaxQuality {
		return false
	}
	return true
}

func (a *counterpartyStreamAccumulator) flush() error {
	if len(a.batch) == 0 {
		return nil
	}
	if err := a.consumer(a.batch); err != nil {
		return err
	}
	a.batch = a.batch[:0]
	return nil
}

func (a *counterpartyStreamAccumulator) finalize() error {
	if err := a.flush(); err != nil {
		return err
	}
	if a.opts.ApplyPagination && a.opts.Limit > 0 && a.emitted >= a.opts.Limit {
		return ErrCounterpartyStreamLimitReached
	}
	return nil
}

func (a *counterpartyStreamAccumulator) averageQuality() float64 {
	if a.avgQualityCount == 0 {
		return 0
	}
	return a.avgQualitySum / float64(a.avgQualityCount)
}

func (a *counterpartyStreamAccumulator) total() int {
	return a.filteredCount
}

// catalogItemToUnified преобразует CatalogItem в UnifiedCounterparty
func (db *ServiceDB) catalogItemToUnified(item *CatalogItem, project *ClientProject, dbInfo *ProjectDatabase) *UnifiedCounterparty {
	unified := &UnifiedCounterparty{
		ID:          item.ID,
		Name:        item.Name,
		Source:      "database",
		ProjectID:   project.ID,
		ProjectName: project.Name,
		Reference:   item.Reference,
		Code:        item.Code,
		Attributes:  item.Attributes,
	}

	if dbInfo != nil {
		dbID := dbInfo.ID
		unified.DatabaseID = &dbID
		unified.DatabaseName = dbInfo.Name
	}

	// Извлекаем данные из атрибутов используя extractors
	// Извлекаем ИНН
	if inn, err := extractors.ExtractINNFromAttributes(item.Attributes); err == nil {
		unified.TaxID = inn
	}

	// Извлекаем КПП
	if kpp, err := extractors.ExtractKPPFromAttributes(item.Attributes); err == nil {
		unified.KPP = kpp
	}

	// Извлекаем БИН
	if bin, err := extractors.ExtractBINFromAttributes(item.Attributes); err == nil {
		unified.BIN = bin
	}

	// Если ИНН/БИН не найдены в атрибутах, пробуем извлечь из других полей
	if unified.TaxID == "" && unified.BIN == "" {
		// Пробуем извлечь из Code
		if item.Code != "" {
			if inn, err := extractors.ExtractINNFromAttributes(item.Code); err == nil {
				unified.TaxID = inn
			} else if bin, err := extractors.ExtractBINFromAttributes(item.Code); err == nil {
				unified.BIN = bin
			}
		}

		// Пробуем извлечь из Reference
		if unified.TaxID == "" && unified.BIN == "" && item.Reference != "" {
			if inn, err := extractors.ExtractINNFromAttributes(item.Reference); err == nil {
				unified.TaxID = inn
			} else if bin, err := extractors.ExtractBINFromAttributes(item.Reference); err == nil {
				unified.BIN = bin
			}
		}

		// Пробуем извлечь из Name (иногда ИНН/БИН может быть в названии)
		if unified.TaxID == "" && unified.BIN == "" && item.Name != "" {
			// Ищем паттерны типа "ООО Компания ИНН 1234567890" или "ТОО БИН 123456789012"
			if inn, err := extractors.ExtractINNFromAttributes(item.Name); err == nil {
				unified.TaxID = inn
			} else if bin, err := extractors.ExtractBINFromAttributes(item.Name); err == nil {
				unified.BIN = bin
			}
		}
	}

	// Если нашли БИН, но не нашли TaxID, используем БИН как TaxID для отображения
	if unified.TaxID == "" && unified.BIN != "" {
		unified.TaxID = unified.BIN
	}

	// Извлекаем адрес
	if address, err := extractors.ExtractAddressFromAttributes(item.Attributes); err == nil {
		unified.LegalAddress = address
		unified.PostalAddress = address
	}

	// Извлекаем телефон
	if phone, err := extractors.ExtractContactPhoneFromAttributes(item.Attributes); err == nil {
		unified.ContactPhone = phone
	}

	// Извлекаем email
	if email, err := extractors.ExtractContactEmailFromAttributes(item.Attributes); err == nil {
		unified.ContactEmail = email
	}

	// Извлекаем контактное лицо
	if person, err := extractors.ExtractContactPersonFromAttributes(item.Attributes); err == nil {
		unified.ContactPerson = person
	}

	// Добавляем информацию о базе данных в SourceDatabases
	if dbInfo != nil {
		dbID := dbInfo.ID
		unified.SourceDatabases = []DatabaseSource{
			{
				DatabaseID:      dbID,
				DatabaseName:    dbInfo.Name,
				SourceReference: item.Reference,
				SourceName:      item.Name,
			},
		}
	}

	return unified
}

// normalizedToUnified преобразует NormalizedCounterparty в UnifiedCounterparty
func (db *ServiceDB) normalizedToUnified(nc *NormalizedCounterparty, project *ClientProject) *UnifiedCounterparty {
	qualityScore := nc.QualityScore
	taxID := nc.TaxID
	bin := nc.BIN

	// Если TaxID пустой, но BIN есть, используем BIN как TaxID для отображения
	if taxID == "" && bin != "" {
		taxID = bin
	}

	// Загружаем информацию о базах данных из таблицы counterparty_databases
	sourceDatabases := db.getCounterpartyDatabases(nc.ID)

	unified := &UnifiedCounterparty{
		ID:              nc.ID,
		Name:            nc.NormalizedName,
		Source:          "normalized",
		ProjectID:       project.ID,
		ProjectName:     project.Name,
		NormalizedName:  nc.NormalizedName,
		SourceName:      nc.SourceName,
		SourceReference: nc.SourceReference,
		TaxID:           taxID,
		KPP:             nc.KPP,
		BIN:             bin,
		LegalAddress:    nc.LegalAddress,
		PostalAddress:   nc.PostalAddress,
		ContactPhone:    nc.ContactPhone,
		ContactEmail:    nc.ContactEmail,
		ContactPerson:   nc.ContactPerson,
		QualityScore:    &qualityScore,
		SourceDatabases: sourceDatabases,
	}

	// Если есть базы данных, устанавливаем первую как основную для обратной совместимости
	if len(sourceDatabases) > 0 {
		dbID := sourceDatabases[0].DatabaseID
		unified.DatabaseID = &dbID
		unified.DatabaseName = sourceDatabases[0].DatabaseName
	}

	return unified
}

// matchesSearch проверяет, соответствует ли контрагент поисковому запросу
func (db *ServiceDB) matchesSearch(cp *UnifiedCounterparty, search string) bool {
	searchLower := strings.ToLower(search)
	return strings.Contains(strings.ToLower(cp.Name), searchLower) ||
		strings.Contains(strings.ToLower(cp.TaxID), searchLower) ||
		strings.Contains(strings.ToLower(cp.BIN), searchLower) ||
		strings.Contains(strings.ToLower(cp.NormalizedName), searchLower) ||
		strings.Contains(strings.ToLower(cp.SourceName), searchLower)
}

// getCounterpartyDatabases получает список баз данных для контрагента (приватная функция)
func (db *ServiceDB) getCounterpartyDatabases(counterpartyID int) []DatabaseSource {
	query := `
		SELECT pd.id, pd.name, cd.source_reference, cd.source_name
		FROM counterparty_databases cd
		JOIN project_databases pd ON cd.project_database_id = pd.id
		WHERE cd.normalized_counterparty_id = ?
		ORDER BY cd.created_at ASC
	`

	rows, err := db.conn.Query(query, counterpartyID)
	if err != nil {
		// Если таблица не существует или произошла ошибка, возвращаем пустой список
		return []DatabaseSource{}
	}
	defer rows.Close()

	var sources []DatabaseSource
	for rows.Next() {
		var dbID int
		var dbName, sourceRef, sourceName sql.NullString

		if err := rows.Scan(&dbID, &dbName, &sourceRef, &sourceName); err != nil {
			continue
		}

		source := DatabaseSource{
			DatabaseID: dbID,
		}
		if dbName.Valid {
			source.DatabaseName = dbName.String
		}
		if sourceRef.Valid {
			source.SourceReference = sourceRef.String
		}
		if sourceName.Valid {
			source.SourceName = sourceName.String
		}

		sources = append(sources, source)
	}

	return sources
}

// SaveCounterpartyDatabaseLink сохраняет связь между контрагентом и базой данных
func (db *ServiceDB) SaveCounterpartyDatabaseLink(counterpartyID, databaseID int, sourceReference, sourceName string) error {
	// Проверяем существование таблицы
	var tableExists bool
	err := db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM sqlite_master
			WHERE type='table' AND name='counterparty_databases'
		)
	`).Scan(&tableExists)
	if err != nil || !tableExists {
		// Таблица не существует, создаем её
		if err := CreateCounterpartyDatabasesTable(db.conn); err != nil {
			return fmt.Errorf("failed to create counterparty_databases table: %w", err)
		}
	}

	query := `
		INSERT OR IGNORE INTO counterparty_databases
		(normalized_counterparty_id, project_database_id, source_reference, source_name)
		VALUES (?, ?, ?, ?)
	`

	_, err = db.conn.Exec(query, counterpartyID, databaseID, sourceReference, sourceName)
	if err != nil {
		return fmt.Errorf("failed to save counterparty database link: %w", err)
	}

	return nil
}

// GetCounterpartyDatabases получает список баз данных для контрагента
func (db *ServiceDB) GetCounterpartyDatabases(counterpartyID int) ([]DatabaseSource, error) {
	sources := db.getCounterpartyDatabases(counterpartyID)
	return sources, nil
}

// GetDatabaseCounterparties получает список контрагентов для базы данных
func (db *ServiceDB) GetDatabaseCounterparties(databaseID int) ([]*NormalizedCounterparty, error) {
	query := `
		SELECT nc.id, nc.client_project_id, nc.source_reference, nc.source_name, nc.normalized_name,
		       nc.tax_id, nc.kpp, nc.bin, nc.legal_address, nc.postal_address, nc.contact_phone, nc.contact_email,
		       nc.contact_person, nc.legal_form, nc.bank_name, nc.bank_account, nc.correspondent_account, nc.bik,
		       nc.benchmark_id, nc.quality_score, nc.enrichment_applied, COALESCE(nc.source_enrichment, ''), nc.source_database, 
		       COALESCE(nc.subcategory, '') as subcategory, nc.created_at, nc.updated_at
		FROM normalized_counterparties nc
		JOIN counterparty_databases cd ON nc.id = cd.normalized_counterparty_id
		WHERE cd.project_database_id = ?
		ORDER BY nc.normalized_name
	`

	rows, err := db.conn.Query(query, databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to query database counterparties: %w", err)
	}
	defer rows.Close()

	var counterparties []*NormalizedCounterparty
	for rows.Next() {
		cp := &NormalizedCounterparty{}
		var benchmarkID sql.NullInt64

		err := rows.Scan(
			&cp.ID, &cp.ClientProjectID, &cp.SourceReference, &cp.SourceName, &cp.NormalizedName,
			&cp.TaxID, &cp.KPP, &cp.BIN, &cp.LegalAddress, &cp.PostalAddress,
			&cp.ContactPhone, &cp.ContactEmail, &cp.ContactPerson, &cp.LegalForm,
			&cp.BankName, &cp.BankAccount, &cp.CorrespondentAccount, &cp.BIK,
			&benchmarkID, &cp.QualityScore, &cp.EnrichmentApplied, &cp.SourceEnrichment, &cp.SourceDatabase,
			&cp.Subcategory, &cp.CreatedAt, &cp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan counterparty: %w", err)
		}

		if benchmarkID.Valid {
			id := int(benchmarkID.Int64)
			cp.BenchmarkID = &id
		}

		counterparties = append(counterparties, cp)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating counterparties: %w", err)
	}

	return counterparties, nil
}

// ParseDatabaseNames парсит строку с именами баз данных
// Поддерживает форматы: JSON массив, разделители ",", "|", или просто одно имя
func ParseDatabaseNames(sourceDatabase string) []string {
	if sourceDatabase == "" {
		return []string{}
	}

	// Пробуем разделитель "|"
	if strings.Contains(sourceDatabase, "|") {
		parts := strings.Split(sourceDatabase, "|")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// Пробуем разделитель ","
	if strings.Contains(sourceDatabase, ",") {
		parts := strings.Split(sourceDatabase, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
		if len(result) > 0 {
			return result
		}
	}

	// Просто одно имя
	return []string{strings.TrimSpace(sourceDatabase)}
}

// GetCatalogItemsByDatabase получает все catalog items из базы данных проекта
func (db *ServiceDB) GetCatalogItemsByDatabase(databaseID int) ([]*CatalogItem, error) {
	// Получаем информацию о базе данных
	dbInfo, err := db.GetProjectDatabase(databaseID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project database: %w", err)
	}

	// Открываем базу данных
	sourceDB, err := NewDB(dbInfo.FilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer sourceDB.Close()

	// Получаем все выгрузки из этой базы
	uploads, err := sourceDB.GetAllUploads()
	if err != nil {
		return nil, fmt.Errorf("failed to get uploads: %w", err)
	}

	// Собираем все catalog items из всех uploads
	var allItems []*CatalogItem
	for _, upload := range uploads {
		items, _, err := sourceDB.GetCatalogItemsByUpload(upload.ID, []string{"Контрагенты"}, 0, 0)
		if err != nil {
			// Пропускаем ошибки для отдельных uploads
			continue
		}
		allItems = append(allItems, items...)
	}

	return allItems, nil
}

// GetNormalizedCounterpartyBySourceReference получает нормализованного контрагента по source_reference
func (db *ServiceDB) GetNormalizedCounterpartyBySourceReference(projectID int, sourceReference string) (*NormalizedCounterparty, error) {
	query := `
		SELECT id, client_project_id, source_reference, source_name, normalized_name,
		       tax_id, kpp, bin, legal_address, postal_address, contact_phone, contact_email,
		       contact_person, legal_form, bank_name, bank_account, correspondent_account, bik,
		       benchmark_id, quality_score, enrichment_applied, COALESCE(source_enrichment, ''), source_database, 
		       COALESCE(subcategory, '') as subcategory, created_at, updated_at
		FROM normalized_counterparties
		WHERE client_project_id = ? AND source_reference = ?
		LIMIT 1
	`

	nc := &NormalizedCounterparty{}
	var benchmarkID sql.NullInt64

	err := db.conn.QueryRow(query, projectID, sourceReference).Scan(
		&nc.ID, &nc.ClientProjectID, &nc.SourceReference, &nc.SourceName, &nc.NormalizedName,
		&nc.TaxID, &nc.KPP, &nc.BIN, &nc.LegalAddress, &nc.PostalAddress,
		&nc.ContactPhone, &nc.ContactEmail, &nc.ContactPerson, &nc.LegalForm,
		&nc.BankName, &nc.BankAccount, &nc.CorrespondentAccount, &nc.BIK,
		&benchmarkID, &nc.QualityScore, &nc.EnrichmentApplied, &nc.SourceEnrichment, &nc.SourceDatabase,
		&nc.Subcategory, &nc.CreatedAt, &nc.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Не найдено, это нормально
		}
		return nil, fmt.Errorf("failed to get normalized counterparty: %w", err)
	}

	if benchmarkID.Valid {
		id := int(benchmarkID.Int64)
		nc.BenchmarkID = &id
	}

	return nc, nil
}

// ProjectNormalizationConfig конфигурация автоматического мэппинга для проекта
type ProjectNormalizationConfig struct {
	ID                      int       `json:"id"`
	ClientProjectID         int       `json:"client_project_id"`
	AutoMapCounterparties   bool      `json:"auto_map_counterparties"`
	AutoMergeDuplicates     bool      `json:"auto_merge_duplicates"`
	MasterSelectionStrategy string    `json:"master_selection_strategy"` // max_data, max_quality, max_databases
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

// GetProjectNormalizationConfig получает конфигурацию нормализации для проекта
func (db *ServiceDB) GetProjectNormalizationConfig(projectID int) (*ProjectNormalizationConfig, error) {
	query := `
		SELECT id, client_project_id, auto_map_counterparties, auto_merge_duplicates,
		       master_selection_strategy, created_at, updated_at
		FROM project_normalization_config
		WHERE client_project_id = ?
		LIMIT 1
	`

	config := &ProjectNormalizationConfig{}
	err := db.conn.QueryRow(query, projectID).Scan(
		&config.ID, &config.ClientProjectID, &config.AutoMapCounterparties,
		&config.AutoMergeDuplicates, &config.MasterSelectionStrategy,
		&config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// Возвращаем конфигурацию по умолчанию
			return &ProjectNormalizationConfig{
				ClientProjectID:         projectID,
				AutoMapCounterparties:   true,
				AutoMergeDuplicates:     true,
				MasterSelectionStrategy: "max_data",
			}, nil
		}
		return nil, fmt.Errorf("failed to get project normalization config: %w", err)
	}

	return config, nil
}

// UpdateProjectNormalizationConfig обновляет конфигурацию нормализации для проекта
func (db *ServiceDB) UpdateProjectNormalizationConfig(projectID int, config *ProjectNormalizationConfig) error {
	// Проверяем существование записи
	var exists bool
	err := db.conn.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM project_normalization_config
			WHERE client_project_id = ?
		)
	`, projectID).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check config existence: %w", err)
	}

	if !exists {
		// Создаем новую запись
		query := `
			INSERT INTO project_normalization_config
			(client_project_id, auto_map_counterparties, auto_merge_duplicates, master_selection_strategy)
			VALUES (?, ?, ?, ?)
		`
		_, err = db.conn.Exec(query, projectID, config.AutoMapCounterparties,
			config.AutoMergeDuplicates, config.MasterSelectionStrategy)
		if err != nil {
			return fmt.Errorf("failed to create project normalization config: %w", err)
		}
	} else {
		// Обновляем существующую запись
		query := `
			UPDATE project_normalization_config
			SET auto_map_counterparties = ?, auto_merge_duplicates = ?,
			    master_selection_strategy = ?, updated_at = CURRENT_TIMESTAMP
			WHERE client_project_id = ?
		`
		_, err = db.conn.Exec(query, config.AutoMapCounterparties,
			config.AutoMergeDuplicates, config.MasterSelectionStrategy, projectID)
		if err != nil {
			return fmt.Errorf("failed to update project normalization config: %w", err)
		}
	}

	return nil
}

// GetNormalizedCounterpartyStats получает статистику по нормализованным контрагентам проекта
func (db *ServiceDB) GetNormalizedCounterpartyStats(projectID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Общее количество
	var totalCount int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ?`, projectID).Scan(&totalCount)
	if err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	stats["total_count"] = totalCount
	stats["total"] = totalCount      // Для совместимости с фронтендом
	stats["normalized"] = totalCount // Для совместимости с фронтендом

	// С эталонами
	var withBenchmark int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND benchmark_id IS NOT NULL`, projectID).Scan(&withBenchmark)
	if err == nil {
		stats["with_benchmark"] = withBenchmark
	}

	// С дозаполнением
	var enriched int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND enrichment_applied = 1`, projectID).Scan(&enriched)
	if err == nil {
		stats["enriched"] = enriched
	}

	// Средний quality score
	var avgQuality sql.NullFloat64
	err = db.conn.QueryRow(`SELECT AVG(quality_score) FROM normalized_counterparties WHERE client_project_id = ?`, projectID).Scan(&avgQuality)
	if err == nil && avgQuality.Valid {
		stats["average_quality_score"] = avgQuality.Float64
	}

	// С ИНН
	var withINN int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND tax_id != '' AND tax_id IS NOT NULL`, projectID).Scan(&withINN)
	if err == nil {
		stats["with_inn"] = withINN
	}

	// С адресами
	var withAddress int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND (legal_address != '' OR postal_address != '')`, projectID).Scan(&withAddress)
	if err == nil {
		stats["with_address"] = withAddress
	}

	// С контактами
	var withContacts int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND (contact_phone != '' OR contact_email != '')`, projectID).Scan(&withContacts)
	if err == nil {
		stats["with_contacts"] = withContacts
	}

	// Статистика по источникам обогащения
	enrichmentStats := make(map[string]int)
	rows, err := db.conn.Query(`SELECT source_enrichment, COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND source_enrichment != '' AND source_enrichment IS NOT NULL GROUP BY source_enrichment`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var source string
			var count int
			if err := rows.Scan(&source, &count); err == nil {
				enrichmentStats[source] = count
			}
		}
		stats["enrichment_by_source"] = enrichmentStats
	}

	// Общее количество обогащенных
	var totalEnriched int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND source_enrichment != '' AND source_enrichment IS NOT NULL`, projectID).Scan(&totalEnriched)
	if err == nil {
		stats["total_enriched"] = totalEnriched
	}

	// Количество производителей (из глобальных эталонов)
	var manufacturersCount int
	err = db.conn.QueryRow(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND subcategory = 'производитель'`, projectID).Scan(&manufacturersCount)
	if err == nil {
		stats["manufacturers_count"] = manufacturersCount
	}

	// Статистика по подкатегориям
	subcategoryStats := make(map[string]int)
	rows, err = db.conn.Query(`SELECT subcategory, COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND subcategory != '' AND subcategory IS NOT NULL GROUP BY subcategory`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var subcategory string
			var count int
			if err := rows.Scan(&subcategory, &count); err == nil {
				subcategoryStats[subcategory] = count
			}
		}
		stats["subcategory_stats"] = subcategoryStats
	}

	// Статистика по дубликатам
	// Подсчитываем дубликаты по ИНН/БИН
	var duplicatesCount int
	var duplicateGroupsCount int
	duplicateGroups := make(map[string]int)
	rows, err = db.conn.Query(`
		SELECT tax_id, COUNT(*) as cnt 
		FROM normalized_counterparties 
		WHERE client_project_id = ? AND tax_id != '' AND tax_id IS NOT NULL
		GROUP BY tax_id
		HAVING cnt > 1
	`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var taxID string
			var count int
			if err := rows.Scan(&taxID, &count); err == nil {
				duplicateGroups[taxID] = count
				duplicateGroupsCount++
				duplicatesCount += count
			}
		}
	}

	// Также проверяем по БИН
	rows, err = db.conn.Query(`
		SELECT bin, COUNT(*) as cnt 
		FROM normalized_counterparties 
		WHERE client_project_id = ? AND bin != '' AND bin IS NOT NULL
		GROUP BY bin
		HAVING cnt > 1
	`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var bin string
			var count int
			if err := rows.Scan(&bin, &count); err == nil {
				// Проверяем, не добавлена ли уже эта группа по ИНН
				if _, exists := duplicateGroups[bin]; !exists {
					duplicateGroups[bin] = count
					duplicateGroupsCount++
					duplicatesCount += count
				}
			}
		}
	}

	stats["duplicate_groups"] = duplicateGroupsCount
	stats["duplicates_count"] = duplicatesCount - duplicateGroupsCount // Общее количество дубликатов (без эталонов)

	// Количество контрагентов, связанных с несколькими базами данных
	var multiDatabaseCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM (
			SELECT cd.normalized_counterparty_id
			FROM counterparty_databases cd
			INNER JOIN normalized_counterparties nc ON nc.id = cd.normalized_counterparty_id
			WHERE nc.client_project_id = ?
			GROUP BY cd.normalized_counterparty_id
			HAVING COUNT(cd.project_database_id) > 1
		)
	`, projectID).Scan(&multiDatabaseCount)
	if err == nil {
		stats["multi_database_count"] = multiDatabaseCount
	}

	// Статистика по качеству (распределение по диапазонам)
	qualityDistribution := make(map[string]int)
	qualityRanges := []struct {
		label string
		query string
	}{
		{"excellent", "quality_score >= 0.9"},
		{"good", "quality_score >= 0.7 AND quality_score < 0.9"},
		{"fair", "quality_score >= 0.5 AND quality_score < 0.7"},
		{"poor", "quality_score < 0.5"},
	}
	for _, r := range qualityRanges {
		var count int
		query := fmt.Sprintf(`SELECT COUNT(*) FROM normalized_counterparties WHERE client_project_id = ? AND %s`, r.query)
		err = db.conn.QueryRow(query, projectID).Scan(&count)
		if err == nil {
			qualityDistribution[r.label] = count
		}
	}
	stats["quality_distribution"] = qualityDistribution

	// Статистика по датам создания (последние 30 дней)
	dateStats := make([]map[string]interface{}, 0)
	rows, err = db.conn.Query(`
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as count
		FROM normalized_counterparties 
		WHERE client_project_id = ? 
			AND created_at >= datetime('now', '-30 days')
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var date string
			var count int
			if err := rows.Scan(&date, &count); err == nil {
				dateStats = append(dateStats, map[string]interface{}{
					"date":  date,
					"count": count,
				})
			}
		}
		stats["creation_timeline"] = dateStats
	}

	// Статистика по регионам (если есть в эталонах)
	regionStats := make(map[string]int)
	rows, err = db.conn.Query(`
		SELECT 
			COALESCE(cb.region, 'Не указан') as region,
			COUNT(DISTINCT nc.id) as count
		FROM normalized_counterparties nc
		LEFT JOIN client_benchmarks cb ON nc.benchmark_id = cb.id
		WHERE nc.client_project_id = ?
		GROUP BY region
		ORDER BY count DESC
		LIMIT 10
	`, projectID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var region string
			var count int
			if err := rows.Scan(&region, &count); err == nil {
				regionStats[region] = count
			}
		}
		stats["region_distribution"] = regionStats
	}

	// Статистика по полноте данных
	completenessStats := make(map[string]int)

	// Полностью заполненные (все основные поля)
	var completeCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM normalized_counterparties 
		WHERE client_project_id = ?
			AND tax_id != '' AND tax_id IS NOT NULL
			AND legal_address != '' AND legal_address IS NOT NULL
			AND (contact_phone != '' OR contact_email != '')
	`, projectID).Scan(&completeCount)
	if err == nil {
		completenessStats["complete"] = completeCount
	}

	// Частично заполненные
	var partialCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM normalized_counterparties 
		WHERE client_project_id = ?
			AND (
				(tax_id != '' AND tax_id IS NOT NULL AND (legal_address = '' OR legal_address IS NULL))
				OR (legal_address != '' AND legal_address IS NOT NULL AND (tax_id = '' OR tax_id IS NULL))
			)
	`, projectID).Scan(&partialCount)
	if err == nil {
		completenessStats["partial"] = partialCount
	}

	// Минимально заполненные (только название)
	var minimalCount int
	err = db.conn.QueryRow(`
		SELECT COUNT(*) FROM normalized_counterparties 
		WHERE client_project_id = ?
			AND (tax_id = '' OR tax_id IS NULL)
			AND (legal_address = '' OR legal_address IS NULL)
	`, projectID).Scan(&minimalCount)
	if err == nil {
		completenessStats["minimal"] = minimalCount
	}
	stats["completeness_stats"] = completenessStats

	return stats, nil
}

// ProjectTypeClassifier структура связи типа проекта с классификатором
type ProjectTypeClassifier struct {
	ID           int       `json:"id"`
	ProjectType  string    `json:"project_type"`
	ClassifierID int       `json:"classifier_id"`
	IsDefault    bool      `json:"is_default"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CreateProjectTypeClassifier создает связь типа проекта с классификатором
func (db *ServiceDB) CreateProjectTypeClassifier(projectType string, classifierID int, isDefault bool) (*ProjectTypeClassifier, error) {
	query := `
		INSERT INTO project_type_classifiers (project_type, classifier_id, is_default)
		VALUES (?, ?, ?)
	`

	result, err := db.conn.Exec(query, projectType, classifierID, isDefault)
	if err != nil {
		return nil, fmt.Errorf("failed to create project type classifier: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get project type classifier ID: %w", err)
	}

	return db.GetProjectTypeClassifier(int(id))
}

// GetProjectTypeClassifier получает связь по ID
func (db *ServiceDB) GetProjectTypeClassifier(id int) (*ProjectTypeClassifier, error) {
	query := `
		SELECT id, project_type, classifier_id, is_default, created_at, updated_at
		FROM project_type_classifiers WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	ptc := &ProjectTypeClassifier{}

	err := row.Scan(
		&ptc.ID, &ptc.ProjectType, &ptc.ClassifierID, &ptc.IsDefault,
		&ptc.CreatedAt, &ptc.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get project type classifier: %w", err)
	}

	return ptc, nil
}

// GetClassifiersByProjectType получает классификаторы для типа проекта
// Используем структуру из db_classification.go через алиас
func (db *ServiceDB) GetClassifiersByProjectType(projectType string) ([]map[string]interface{}, error) {
	query := `
		SELECT c.id, c.name, c.description, c.max_depth, c.tree_structure,
		       c.client_id, c.project_id, c.is_active, c.created_at, c.updated_at
		FROM category_classifiers c
		INNER JOIN project_type_classifiers ptc ON c.id = ptc.classifier_id
		WHERE ptc.project_type = ? AND c.is_active = TRUE
		ORDER BY ptc.is_default DESC, c.name ASC
	`

	rows, err := db.conn.Query(query, projectType)
	if err != nil {
		return nil, fmt.Errorf("failed to get classifiers by project type: %w", err)
	}
	defer rows.Close()

	var classifiers []map[string]interface{}
	for rows.Next() {
		var id, maxDepth int
		var name, description, treeStructure string
		var isActive bool
		var clientID, projectID sql.NullInt64
		var createdAt, updatedAt time.Time

		err := rows.Scan(
			&id, &name, &description, &maxDepth,
			&treeStructure, &clientID, &projectID, &isActive,
			&createdAt, &updatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan classifier: %w", err)
		}

		classifier := map[string]interface{}{
			"id":             id,
			"name":           name,
			"description":    description,
			"max_depth":      maxDepth,
			"tree_structure": treeStructure,
			"is_active":      isActive,
			"created_at":     createdAt,
			"updated_at":     updatedAt,
		}

		if clientID.Valid {
			classifier["client_id"] = int(clientID.Int64)
		}
		if projectID.Valid {
			classifier["project_id"] = int(projectID.Int64)
		}

		classifiers = append(classifiers, classifier)
	}

	return classifiers, nil
}

// DeleteProjectTypeClassifier удаляет связь типа проекта с классификатором
func (db *ServiceDB) DeleteProjectTypeClassifier(id int) error {
	query := `DELETE FROM project_type_classifiers WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project type classifier: %w", err)
	}
	return nil
}

// GetAllProjectTypeClassifiers получает все связи типов проектов с классификаторами
func (db *ServiceDB) GetAllProjectTypeClassifiers() ([]*ProjectTypeClassifier, error) {
	query := `
		SELECT id, project_type, classifier_id, is_default, created_at, updated_at
		FROM project_type_classifiers
		ORDER BY project_type, is_default DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all project type classifiers: %w", err)
	}
	defer rows.Close()

	var ptcs []*ProjectTypeClassifier
	for rows.Next() {
		ptc := &ProjectTypeClassifier{}
		err := rows.Scan(
			&ptc.ID, &ptc.ProjectType, &ptc.ClassifierID, &ptc.IsDefault,
			&ptc.CreatedAt, &ptc.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project type classifier: %w", err)
		}
		ptcs = append(ptcs, ptc)
	}

	return ptcs, nil
}

// ProjectNormalizationSession представляет сессию нормализации для базы данных проекта
type ProjectNormalizationSession struct {
	ID                int
	ProjectDatabaseID int
	StartedAt         time.Time
	FinishedAt        *time.Time
	Status            string
	Priority          int
	TimeoutSeconds    int
	LastActivityAt    time.Time
	CreatedAt         time.Time
}

// CreateNormalizationSession создает новую сессию нормализации для базы данных проекта
func (db *ServiceDB) CreateNormalizationSession(projectDatabaseID int, priority int, timeoutSeconds int) (int, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 3600 // Дефолтный таймаут 1 час
	}

	query := `
		INSERT INTO normalization_sessions (project_database_id, status, started_at, priority, timeout_seconds, last_activity_at)
		VALUES (?, 'running', CURRENT_TIMESTAMP, ?, ?, CURRENT_TIMESTAMP)
	`
	result, err := db.conn.Exec(query, projectDatabaseID, priority, timeoutSeconds)
	if err != nil {
		return 0, fmt.Errorf("failed to create normalization session: %w", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get session ID: %w", err)
	}

	return int(sessionID), nil
}

// TryCreateNormalizationSession пытается создать новую сессию нормализации
// только если для данной БД нет активных сессий (status = 'running')
// Возвращает (sessionID, true) если сессия создана, (0, false) если уже есть активная
func (db *ServiceDB) TryCreateNormalizationSession(projectDatabaseID int, priority int, timeoutSeconds int) (int, bool, error) {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 3600 // Дефолтный таймаут 1 час
	}

	// Используем транзакцию для атомарности проверки и создания
	// SQLite автоматически блокирует таблицу при записи, что предотвращает race conditions
	// Используем обычную транзакцию - SQLite гарантирует атомарность операций внутри транзакции
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, false, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Проверяем наличие активных сессий для этой БД
	// Используем индекс idx_normalization_sessions_db_status для быстрого поиска
	var activeCount int
	checkQuery := `
		SELECT COUNT(*) FROM normalization_sessions
		WHERE project_database_id = ? AND status = 'running'
	`
	err = tx.QueryRow(checkQuery, projectDatabaseID).Scan(&activeCount)
	if err != nil {
		return 0, false, fmt.Errorf("failed to check active sessions: %w", err)
	}

	// Если есть активная сессия, не создаем новую
	if activeCount > 0 {
		log.Printf("[TryCreateNormalizationSession] Database %d already has %d active session(s), skipping creation", projectDatabaseID, activeCount)
		return 0, false, nil
	}

	// Создаем новую сессию
	insertQuery := `
		INSERT INTO normalization_sessions (project_database_id, status, started_at, priority, timeout_seconds, last_activity_at)
		VALUES (?, 'running', CURRENT_TIMESTAMP, ?, ?, CURRENT_TIMESTAMP)
	`
	result, err := tx.Exec(insertQuery, projectDatabaseID, priority, timeoutSeconds)
	if err != nil {
		return 0, false, fmt.Errorf("failed to create normalization session: %w", err)
	}

	sessionID, err := result.LastInsertId()
	if err != nil {
		return 0, false, fmt.Errorf("failed to get session ID: %w", err)
	}

	// Коммитим транзакцию
	if err = tx.Commit(); err != nil {
		return 0, false, fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("[TryCreateNormalizationSession] Successfully created normalization session %d for database %d", sessionID, projectDatabaseID)
	return int(sessionID), true, nil
}

// GetNormalizationSession получает сессию нормализации по ID
func (db *ServiceDB) GetNormalizationSession(sessionID int) (*ProjectNormalizationSession, error) {
	query := `
		SELECT id, project_database_id, started_at, finished_at, status, priority, timeout_seconds, last_activity_at, created_at
		FROM normalization_sessions
		WHERE id = ?
	`

	session := &ProjectNormalizationSession{}
	var finishedAt sql.NullTime

	err := db.conn.QueryRow(query, sessionID).Scan(
		&session.ID, &session.ProjectDatabaseID, &session.StartedAt, &finishedAt,
		&session.Status, &session.Priority, &session.TimeoutSeconds, &session.LastActivityAt, &session.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("normalization session not found")
		}
		return nil, fmt.Errorf("failed to get normalization session: %w", err)
	}

	if finishedAt.Valid {
		session.FinishedAt = &finishedAt.Time
	}

	return session, nil
}

// UpdateNormalizationSession обновляет статус сессии нормализации
func (db *ServiceDB) UpdateNormalizationSession(sessionID int, status string, finishedAt *time.Time) error {
	var query string
	var args []interface{}

	if finishedAt != nil {
		query = `
			UPDATE normalization_sessions
			SET status = ?, finished_at = ?, last_activity_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		args = []interface{}{status, finishedAt, sessionID}
	} else {
		query = `
			UPDATE normalization_sessions
			SET status = ?, last_activity_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`
		args = []interface{}{status, sessionID}
	}

	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update normalization session: %w", err)
	}

	return nil
}

// UpdateSessionActivity обновляет время последней активности сессии
func (db *ServiceDB) UpdateSessionActivity(sessionID int) error {
	query := `
		UPDATE normalization_sessions
		SET last_activity_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'running'
	`
	_, err := db.conn.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session activity: %w", err)
	}
	return nil
}

// StopNormalizationSession останавливает сессию нормализации
func (db *ServiceDB) StopNormalizationSession(sessionID int) error {
	finishedAt := time.Now()
	query := `
		UPDATE normalization_sessions
		SET status = 'stopped', finished_at = ?, last_activity_at = CURRENT_TIMESTAMP
		WHERE id = ? AND status = 'running'
	`
	result, err := db.conn.Exec(query, finishedAt, sessionID)
	if err != nil {
		return fmt.Errorf("failed to stop normalization session: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("session %d not found or already stopped", sessionID)
	}

	return nil
}

// CheckAndMarkTimeoutSessions проверяет и помечает зависшие сессии как timeout
func (db *ServiceDB) CheckAndMarkTimeoutSessions() (int, error) {
	query := `
		UPDATE normalization_sessions
		SET status = 'timeout', finished_at = CURRENT_TIMESTAMP
		WHERE status = 'running' 
		  AND (julianday('now') - julianday(last_activity_at)) * 86400 > timeout_seconds
	`
	result, err := db.conn.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to check timeout sessions: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	return int(rowsAffected), nil
}

// GetRunningSessions получает все активные сессии
func (db *ServiceDB) GetRunningSessions() ([]*ProjectNormalizationSession, error) {
	query := `
		SELECT id, project_database_id, started_at, finished_at, status, 
		       priority, timeout_seconds, last_activity_at, created_at
		FROM normalization_sessions
		WHERE status = 'running'
		ORDER BY priority DESC, started_at ASC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get running sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*ProjectNormalizationSession
	for rows.Next() {
		session := &ProjectNormalizationSession{}
		var finishedAt sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.ProjectDatabaseID,
			&session.StartedAt,
			&finishedAt,
			&session.Status,
			&session.Priority,
			&session.TimeoutSeconds,
			&session.LastActivityAt,
			&session.CreatedAt,
		)
		if err != nil {
			continue
		}

		if finishedAt.Valid {
			session.FinishedAt = &finishedAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetStoppedSessions получает все остановленные сессии нормализации
func (db *ServiceDB) GetStoppedSessions() ([]*ProjectNormalizationSession, error) {
	query := `
		SELECT id, project_database_id, started_at, finished_at, status, 
		       priority, timeout_seconds, last_activity_at, created_at
		FROM normalization_sessions
		WHERE status = 'stopped'
		ORDER BY finished_at DESC
	`

	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get stopped sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*ProjectNormalizationSession
	for rows.Next() {
		session := &ProjectNormalizationSession{}
		var finishedAt sql.NullTime

		err := rows.Scan(
			&session.ID,
			&session.ProjectDatabaseID,
			&session.StartedAt,
			&finishedAt,
			&session.Status,
			&session.Priority,
			&session.TimeoutSeconds,
			&session.LastActivityAt,
			&session.CreatedAt,
		)
		if err != nil {
			continue
		}

		if finishedAt.Valid {
			session.FinishedAt = &finishedAt.Time
		}

		sessions = append(sessions, session)
	}

	return sessions, nil
}

// GetSessionStatistics получает статистику по сессиям нормализации для проекта
func (db *ServiceDB) GetSessionStatistics(projectID int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Получаем все базы данных проекта
	databases, err := db.GetProjectDatabases(projectID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get project databases: %w", err)
	}

	if len(databases) == 0 {
		stats["total_sessions"] = 0
		stats["running_sessions"] = 0
		stats["stopped_sessions"] = 0
		stats["completed_sessions"] = 0
		stats["failed_sessions"] = 0
		return stats, nil
	}

	dbIDs := make([]interface{}, len(databases))
	for i, db := range databases {
		dbIDs[i] = db.ID
	}

	placeholders := strings.Repeat("?,", len(dbIDs)-1) + "?"

	// Общее количество сессий
	var totalSessions int
	query := fmt.Sprintf(`
		SELECT COUNT(*) FROM normalization_sessions
		WHERE project_database_id IN (%s)
	`, placeholders)
	err = db.conn.QueryRow(query, dbIDs...).Scan(&totalSessions)
	if err == nil {
		stats["total_sessions"] = totalSessions
	}

	// Сессии по статусам
	statusQuery := fmt.Sprintf(`
		SELECT status, COUNT(*) as count
		FROM normalization_sessions
		WHERE project_database_id IN (%s)
		GROUP BY status
	`, placeholders)

	rows, err := db.conn.Query(statusQuery, dbIDs...)
	if err == nil {
		defer rows.Close()
		statusCounts := make(map[string]int)
		for rows.Next() {
			var status string
			var count int
			if err := rows.Scan(&status, &count); err == nil {
				statusCounts[status] = count
			}
		}
		stats["running_sessions"] = statusCounts["running"]
		stats["stopped_sessions"] = statusCounts["stopped"]
		stats["completed_sessions"] = statusCounts["completed"]
		stats["failed_sessions"] = statusCounts["failed"]
		stats["timeout_sessions"] = statusCounts["timeout"]
	}

	// Последняя сессия
	lastSessionQuery := fmt.Sprintf(`
		SELECT id, status, started_at, finished_at
		FROM normalization_sessions
		WHERE project_database_id IN (%s)
		ORDER BY started_at DESC
		LIMIT 1
	`, placeholders)

	var lastSessionID sql.NullInt64
	var lastStatus sql.NullString
	var lastStartedAt sql.NullTime
	var lastFinishedAt sql.NullTime
	err = db.conn.QueryRow(lastSessionQuery, dbIDs...).Scan(&lastSessionID, &lastStatus, &lastStartedAt, &lastFinishedAt)
	if err == nil && lastSessionID.Valid {
		stats["last_session"] = map[string]interface{}{
			"id":         lastSessionID.Int64,
			"status":     lastStatus.String,
			"started_at": lastStartedAt.Time.Format(time.RFC3339),
		}
		if lastFinishedAt.Valid {
			stats["last_session"].(map[string]interface{})["finished_at"] = lastFinishedAt.Time.Format(time.RFC3339)
		}
	}

	return stats, nil
}

// GetLastNormalizationSession получает последнюю сессию нормализации для базы данных проекта
func (db *ServiceDB) GetLastNormalizationSession(projectDatabaseID int) (*ProjectNormalizationSession, error) {
	query := `
		SELECT id, project_database_id, started_at, finished_at, status, priority, timeout_seconds, last_activity_at, created_at
		FROM normalization_sessions
		WHERE project_database_id = ?
		ORDER BY started_at DESC
		LIMIT 1
	`

	var session ProjectNormalizationSession
	var finishedAt sql.NullTime

	err := db.conn.QueryRow(query, projectDatabaseID).Scan(
		&session.ID,
		&session.ProjectDatabaseID,
		&session.StartedAt,
		&finishedAt,
		&session.Status,
		&session.Priority,
		&session.TimeoutSeconds,
		&session.LastActivityAt,
		&session.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Нет сессий для этой базы
		}
		return nil, fmt.Errorf("failed to get normalization session: %w", err)
	}

	if finishedAt.Valid {
		session.FinishedAt = &finishedAt.Time
	}

	return &session, nil
}

// UpdateSessionPriority обновляет приоритет сессии нормализации
func (db *ServiceDB) UpdateSessionPriority(sessionID int, priority int) error {
	query := `UPDATE normalization_sessions SET priority = ? WHERE id = ?`
	_, err := db.conn.Exec(query, priority, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update session priority: %w", err)
	}
	return nil
}

// LinkProjectDatabaseToProject привязывает базу данных к проекту
// Если база данных уже существует в project_databases, обновляет client_project_id
// Если базы данных нет, создает новую запись
func (db *ServiceDB) LinkProjectDatabaseToProject(databaseID int, projectID int) error {
	// Проверяем, существует ли база данных
	existingDB, err := db.GetProjectDatabase(databaseID)
	if err != nil {
		return fmt.Errorf("failed to get project database: %w", err)
	}

	if existingDB == nil {
		return fmt.Errorf("database with id %d not found", databaseID)
	}

	// Обновляем client_project_id
	query := `
		UPDATE project_databases
		SET client_project_id = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.conn.Exec(query, projectID, databaseID)
	if err != nil {
		return fmt.Errorf("failed to link database to project: %w", err)
	}

	return nil
}

// LinkDatabaseByPathToProject привязывает базу данных к проекту по пути файла
// Если база данных уже существует в project_databases, обновляет client_project_id
// Если базы данных нет, создает новую запись
func (db *ServiceDB) LinkDatabaseByPathToProject(filePath string, projectID int, name string) (*ProjectDatabase, error) {
	// Нормализуем путь
	normalizedPath := filepath.Clean(filePath)
	normalizedPathSlash := filepath.ToSlash(normalizedPath)
	normalizedPathBackslash := filepath.FromSlash(normalizedPath)

	// Ищем существующую базу данных по пути
	query := `
		SELECT id, client_project_id, name, file_path, description, is_active,
		       file_size, last_used_at, created_at, updated_at
		FROM project_databases
		WHERE file_path = ? OR file_path = ? OR file_path = ? OR file_path = ?
		LIMIT 1
	`

	row := db.conn.QueryRow(query, filePath, normalizedPath, normalizedPathSlash, normalizedPathBackslash)
	existingDB := &ProjectDatabase{}
	var lastUsedAt sql.NullTime

	err := row.Scan(
		&existingDB.ID, &existingDB.ClientProjectID, &existingDB.Name, &existingDB.FilePath,
		&existingDB.Description, &existingDB.IsActive, &existingDB.FileSize, &lastUsedAt,
		&existingDB.CreatedAt, &existingDB.UpdatedAt,
	)

	if err == nil {
		// База данных существует, обновляем client_project_id
		if existingDB.ClientProjectID != projectID {
			if err := db.LinkProjectDatabaseToProject(existingDB.ID, projectID); err != nil {
				return nil, err
			}
			existingDB.ClientProjectID = projectID
		}
		if lastUsedAt.Valid {
			existingDB.LastUsedAt = &lastUsedAt.Time
		}
		return existingDB, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to check existing database: %w", err)
	}

	// База данных не существует, создаем новую запись
	// Получаем размер файла
	var fileSize int64
	if info, err := os.Stat(filePath); err == nil {
		fileSize = info.Size()
	}

	// Если имя не указано, используем имя файла
	if name == "" {
		name = filepath.Base(filePath)
	}

	return db.CreateProjectDatabase(projectID, name, normalizedPath, "", fileSize)
}

// GetUnlinkedDatabases возвращает все базы данных, которые не привязаны ни к одному проекту
// (client_project_id IS NULL OR client_project_id = 0)
func (db *ServiceDB) GetUnlinkedDatabases() ([]*ProjectDatabase, error) {
	query := `
		SELECT id, client_project_id, name, file_path, description, is_active,
		       file_size, last_used_at, created_at, updated_at
		FROM project_databases
		WHERE client_project_id IS NULL OR client_project_id = 0
		ORDER BY created_at DESC
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get unlinked databases: %w", err)
	}
	defer rows.Close()

	var databases []*ProjectDatabase
	for rows.Next() {
		dbRecord := &ProjectDatabase{}
		var clientProjectID sql.NullInt64
		var lastUsedAt sql.NullTime
		err := rows.Scan(
			&dbRecord.ID, &clientProjectID, &dbRecord.Name, &dbRecord.FilePath,
			&dbRecord.Description, &dbRecord.IsActive, &dbRecord.FileSize,
			&lastUsedAt, &dbRecord.CreatedAt, &dbRecord.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan unlinked database: %w", err)
		}
		if clientProjectID.Valid {
			dbRecord.ClientProjectID = int(clientProjectID.Int64)
		}
		if lastUsedAt.Valid {
			dbRecord.LastUsedAt = &lastUsedAt.Time
		}
		databases = append(databases, dbRecord)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating unlinked databases: %w", err)
	}

	return databases, nil
}

// UnlinkProjectDatabase отвязывает базу данных от проекта
// Устанавливает client_project_id в NULL
func (db *ServiceDB) UnlinkProjectDatabase(databaseID int) error {
	// Проверяем, что база данных существует
	dbRecord, err := db.GetProjectDatabase(databaseID)
	if err != nil {
		return fmt.Errorf("database not found: %w", err)
	}
	if dbRecord == nil {
		return fmt.Errorf("database %d not found", databaseID)
	}

	// Отвязываем базу данных от проекта
	query := `
		UPDATE project_databases
		SET client_project_id = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`

	_, err = db.conn.Exec(query, databaseID)
	if err != nil {
		return fmt.Errorf("failed to unlink database from project: %w", err)
	}

	return nil
}

// SaveNotification сохраняет уведомление в БД
func (db *ServiceDB) SaveNotification(notificationType, title, message string, clientID, projectID *int, metadata map[string]interface{}) (int, error) {
	var metadataJSON string
	if metadata != nil {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return 0, fmt.Errorf("failed to marshal metadata: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	query := `
		INSERT INTO notifications (type, title, message, client_id, project_id, metadata_json, timestamp)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`

	result, err := db.conn.Exec(query, notificationType, title, message, clientID, projectID, metadataJSON)
	if err != nil {
		return 0, fmt.Errorf("failed to save notification: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get notification ID: %w", err)
	}

	return int(id), nil
}

// GetNotificationsFromDB получает уведомления из БД
func (db *ServiceDB) GetNotificationsFromDB(limit, offset int, unreadOnly bool, clientID, projectID *int) ([]map[string]interface{}, error) {
	whereClause := "1=1"
	args := []interface{}{}

	if unreadOnly {
		whereClause += " AND read = FALSE"
	}

	if clientID != nil {
		whereClause += " AND client_id = ?"
		args = append(args, *clientID)
	}

	if projectID != nil {
		whereClause += " AND project_id = ?"
		args = append(args, *projectID)
	}

	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, type, title, message, timestamp, read, client_id, project_id, metadata_json
		FROM notifications
		WHERE %s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var id int
		var notificationType, title, message string
		var timestamp time.Time
		var read bool
		var clientID, projectID sql.NullInt64
		var metadataJSON sql.NullString

		err := rows.Scan(&id, &notificationType, &title, &message, &timestamp, &read, &clientID, &projectID, &metadataJSON)
		if err != nil {
			continue
		}

		notification := map[string]interface{}{
			"id":        id,
			"type":      notificationType,
			"title":     title,
			"message":   message,
			"timestamp": timestamp,
			"read":      read,
		}

		if clientID.Valid {
			clientIDVal := int(clientID.Int64)
			notification["client_id"] = &clientIDVal
		}
		if projectID.Valid {
			projectIDVal := int(projectID.Int64)
			notification["project_id"] = &projectIDVal
		}
		if metadataJSON.Valid && metadataJSON.String != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(metadataJSON.String), &metadata); err == nil {
				notification["metadata"] = metadata
			}
		}

		notifications = append(notifications, notification)
	}

	return notifications, nil
}

// MarkNotificationAsRead помечает уведомление как прочитанное в БД
func (db *ServiceDB) MarkNotificationAsRead(notificationID int) error {
	query := `UPDATE notifications SET read = TRUE WHERE id = ?`
	result, err := db.conn.Exec(query, notificationID)
	if err != nil {
		return fmt.Errorf("failed to mark notification as read: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("notification with id %d not found", notificationID)
	}
	return nil
}

// MarkAllNotificationsAsRead помечает все уведомления как прочитанные в БД
func (db *ServiceDB) MarkAllNotificationsAsRead(clientID, projectID *int) error {
	whereClause := "1=1"
	args := []interface{}{}

	if clientID != nil {
		whereClause += " AND client_id = ?"
		args = append(args, *clientID)
	}

	if projectID != nil {
		whereClause += " AND project_id = ?"
		args = append(args, *projectID)
	}

	query := fmt.Sprintf(`UPDATE notifications SET read = TRUE WHERE %s`, whereClause)
	_, err := db.conn.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to mark all notifications as read: %w", err)
	}
	return nil
}

// DeleteNotification удаляет уведомление из БД
func (db *ServiceDB) DeleteNotification(notificationID int) error {
	query := `DELETE FROM notifications WHERE id = ?`
	result, err := db.conn.Exec(query, notificationID)
	if err != nil {
		return fmt.Errorf("failed to delete notification: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("notification with id %d not found", notificationID)
	}
	return nil
}

// GetUnreadNotificationsCount получает количество непрочитанных уведомлений из БД
func (db *ServiceDB) GetUnreadNotificationsCount(clientID, projectID *int) (int, error) {
	whereClause := "read = FALSE"
	args := []interface{}{}

	if clientID != nil {
		whereClause += " AND client_id = ?"
		args = append(args, *clientID)
	}

	if projectID != nil {
		whereClause += " AND project_id = ?"
		args = append(args, *projectID)
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM notifications WHERE %s`, whereClause)
	var count int
	err := db.conn.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get unread notifications count: %w", err)
	}
	return count, nil
}

// GetNotificationsCount получает общее количество уведомлений с учетом фильтров
func (db *ServiceDB) GetNotificationsCount(unreadOnly bool, clientID, projectID *int) (int, error) {
	whereClause := "1=1"
	args := []interface{}{}

	if unreadOnly {
		whereClause += " AND read = FALSE"
	}

	if clientID != nil {
		whereClause += " AND client_id = ?"
		args = append(args, *clientID)
	}

	if projectID != nil {
		whereClause += " AND project_id = ?"
		args = append(args, *projectID)
	}

	query := fmt.Sprintf(`SELECT COUNT(*) FROM notifications WHERE %s`, whereClause)
	var count int
	err := db.conn.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get notifications count: %w", err)
	}
	return count, nil
}

// UploadClientDocument сохраняет информацию о загруженном документе клиента
func (db *ServiceDB) UploadClientDocument(clientID int, fileName, filePath, fileType string, fileSize int64, category, description, uploadedBy string) (*ClientDocument, error) {
	query := `
		INSERT INTO client_documents (client_id, file_name, file_path, file_type, file_size, category, description, uploaded_by)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := db.conn.Exec(query, clientID, fileName, filePath, fileType, fileSize, category, description, uploadedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to upload client document: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get document ID: %w", err)
	}

	return db.GetClientDocument(int(id))
}

// GetClientDocument получает документ клиента по ID
func (db *ServiceDB) GetClientDocument(id int) (*ClientDocument, error) {
	query := `
		SELECT id, client_id, file_name, file_path, file_type, file_size, category, description, uploaded_by, uploaded_at
		FROM client_documents WHERE id = ?
	`

	row := db.conn.QueryRow(query, id)
	doc := &ClientDocument{}

	var (
		description sql.NullString
		uploadedBy  sql.NullString
	)

	err := row.Scan(
		&doc.ID,
		&doc.ClientID,
		&doc.FileName,
		&doc.FilePath,
		&doc.FileType,
		&doc.FileSize,
		&doc.Category,
		&description,
		&uploadedBy,
		&doc.UploadedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get client document: %w", err)
	}

	doc.Description = nullString(description)
	doc.UploadedBy = nullString(uploadedBy)

	return doc, nil
}

// GetClientDocuments получает все документы клиента
func (db *ServiceDB) GetClientDocuments(clientID int) ([]*ClientDocument, error) {
	query := `
		SELECT id, client_id, file_name, file_path, file_type, file_size, category, description, uploaded_by, uploaded_at
		FROM client_documents
		WHERE client_id = ?
		ORDER BY uploaded_at DESC
	`

	rows, err := db.conn.Query(query, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to get client documents: %w", err)
	}
	defer rows.Close()

	var documents []*ClientDocument
	for rows.Next() {
		doc := &ClientDocument{}
		var (
			description sql.NullString
			uploadedBy  sql.NullString
		)

		err := rows.Scan(
			&doc.ID,
			&doc.ClientID,
			&doc.FileName,
			&doc.FilePath,
			&doc.FileType,
			&doc.FileSize,
			&doc.Category,
			&description,
			&uploadedBy,
			&doc.UploadedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan client document: %w", err)
		}

		doc.Description = nullString(description)
		doc.UploadedBy = nullString(uploadedBy)

		documents = append(documents, doc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating client documents: %w", err)
	}

	return documents, nil
}

// DeleteClientDocument удаляет документ клиента
func (db *ServiceDB) DeleteClientDocument(id int) error {
	query := `DELETE FROM client_documents WHERE id = ?`
	result, err := db.conn.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete client document: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("document with id %d not found", id)
	}

	return nil
}
