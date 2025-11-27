
package importer

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

// GostCSVRecord represents a GOST record from CSV file
type GostCSVRecord struct {
	Number        string `csv:"number"`
	Title         string `csv:"title"`
	AdoptionDate  string `csv:"adoption_date"`
	EffectiveDate string `csv:"effective_date"`
	Status        string `csv:"status"`
	SourceType    string `csv:"source_type"`
	SourceURL     string `csv:"source_url"`
	Description   string `csv:"description"`
	Keywords      string `csv:"keywords"`
}

// GostRecord представляет запись ГОСТа из CSV файла Росстандарта
type GostRecord struct {
	GostNumber    string
	Title         string
	AdoptionDate  *time.Time
	EffectiveDate *time.Time
	Status        string
	SourceType    string
	SourceURL     string
	Description   string
	Keywords      string
}

// Gost represents a parsed GOST standard
type Gost struct {
	ID            int        `json:"id"`
	GostNumber    string     `json:"gost_number"`
	Title         string     `json:"title"`
	AdoptionDate  *time.Time `json:"adoption_date"`
	EffectiveDate *time.Time `json:"effective_date"`
	Status        string     `json:"status"`
	SourceType    string     `json:"source_type"`
	SourceURL     string     `json:"source_url"`
	Description   string     `json:"description"`
	Keywords      string     `json:"keywords"`
}

// GostColumnIndices holds column indices for GOST CSV parsing
type GostColumnIndices struct {
	gostNumber    int
	title         int
	adoptionDate  int
	effectiveDate int
	status        int
	description   int
	keywords      int
}

// ParserConfig holds configuration options for the GOST parser
type ParserConfig struct {
	Delimiter      rune   // CSV delimiter (default: comma)
	HasHeader      bool   // Whether CSV has header row
	Encoding       string // Expected encoding (default: utf-8)
	SkipEmptyRows  bool   // Skip empty rows
	NormalizeDates bool   // Normalize date formats
	MaxErrors      int    // Max parsing errors before stopping
	ErrorCallback  func(error) // Callback for parsing errors
}

// DefaultParserConfig returns default configuration for the parser
func DefaultParserConfig() ParserConfig {
	return ParserConfig{
		Delimiter:      ';', // Росстандарт использует точку с запятой
		HasHeader:      true,
		Encoding:       "utf-8",
		SkipEmptyRows:  true,
		NormalizeDates: true,
		MaxErrors:      100,
		ErrorCallback:  func(err error) { fmt.Printf("Parsing error: %v\n", err) },
	}
}

// GostParser handles parsing of CSV files from Rosstandart sources
type GostParser struct {
	config   ParserConfig
	logger   interface{ Printf(format string, v ...interface{}) }
	errorCount int
}

// NewGostParser creates a new GOST parser with the given configuration
func NewGostParser(config ParserConfig, logger interface{ Printf(format string, v ...interface{}) }) *GostParser {
	if config.Delimiter == 0 {
		config.Delimiter = ','
	}
	if config.ErrorCallback == nil {
		config.ErrorCallback = func(err error) { fmt.Printf("Parsing error: %v\n", err) }
	}
	
	return &GostParser{
		config: config,
		logger: logger,
	}
}

// ParseCSVFile parses a single CSV file and returns GOST records
func (p *GostParser) ParseCSVFile(filePath string) ([]*Gost, error) {
	if p.logger == nil {
		return nil, fmt.Errorf("logger is not configured")
	}

	p.logger.Printf("Starting to parse file: %s", filePath)
	p.errorCount = 0

	file, err := os.Open(filePath)
	if err != nil {
		p.logger.Printf("Failed to open file %s: %v", filePath, err)
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return p.ParseCSVData(data)
}

// ParseCSVData parses CSV data from byte slice and returns GOST records
func (p *GostParser) ParseCSVData(data []byte) ([]*Gost, error) {
	// Detect and convert encoding if necessary
	convertedData, err := p.detectAndConvertEncoding(data)
	if err != nil {
		return nil, fmt.Errorf("failed to detect/convert encoding: %w", err)
	}
	
	// КРИТИЧЕСКАЯ ПРОВЕРКА: Проверяем, что после конвертации нет некорректных символов
	convertedStr := string(convertedData)
	hasInvalidChars := strings.Contains(convertedStr, "╨У") || strings.Contains(convertedStr, "╨Ю") || 
	                   strings.Contains(convertedStr, "╨б") || strings.Contains(convertedStr, "╨в")
	
	if hasInvalidChars {
		// КРИТИЧЕСКАЯ ОШИБКА: Результат все еще содержит некорректные символы
		// Принудительно декодируем исходные данные как Windows-1251
		if p.logger != nil {
			p.logger.Printf("CRITICAL: Converted data still contains invalid chars, forcing Windows-1251 decode on raw data")
		}
		// Если все еще есть некорректные символы, пробуем более агрессивное исправление
		if p.logger != nil {
			p.logger.Printf("Warning: still detecting invalid encoding characters, attempting aggressive fix")
		}
		
		// Пробуем декодировать исходные данные как Windows-1251 напрямую
		decoder := charmap.Windows1251.NewDecoder()
		fixed, _, err := transform.Bytes(decoder, data)
		if err == nil && len(fixed) > 0 && utf8.Valid(fixed) {
			fixedStr := string(fixed)
			// Если после декодирования все еще есть некорректные символы, значит файл уже был в UTF-8
			// с неправильными символами - пробуем декодировать еще раз (двойное декодирование)
			if strings.Contains(fixedStr, "╨У") || strings.Contains(fixedStr, "╨Ю") {
				// Декодируем еще раз - это двойная конвертация
				fixed2, _, err2 := transform.Bytes(decoder, fixed)
				if err2 == nil && len(fixed2) > 0 && utf8.Valid(fixed2) {
					fixedStr2 := string(fixed2)
					if hasCyrillicRunes(fixedStr2) && strings.Contains(fixedStr2, "ГОСТ") && !strings.Contains(fixedStr2, "╨У") {
						convertedData = fixed2
						if p.logger != nil {
							p.logger.Printf("Successfully fixed encoding using double-decoding method")
						}
					}
				}
			} else if hasCyrillicRunes(fixedStr) && strings.Contains(fixedStr, "ГОСТ") {
				// Первое декодирование помогло
				convertedData = fixed
				if p.logger != nil {
					p.logger.Printf("Successfully fixed encoding using Windows-1251 decoder")
				}
			}
		}
	}

	// Create CSV reader
	// Use converted data directly (already in UTF-8 from detectAndConvertEncoding)
	reader := csv.NewReader(strings.NewReader(string(convertedData)))
	reader.Comma = p.config.Delimiter
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	// Read headers if present
	var headers []string
	if p.config.HasHeader {
		headers, err = reader.Read()
		if err != nil {
			return nil, fmt.Errorf("failed to read CSV headers: %w", err)
		}
	}

	// Auto-detect format if headers are present
	var sourceType string
	if p.config.HasHeader && len(headers) > 0 {
		sourceType, err = p.AutoDetectFormat(convertedData)
		if err != nil {
			p.logger.Printf("Failed to auto-detect format: %v", err)
			// Continue with default format
			sourceType = "national"
		}
	} else {
		sourceType = "national"
	}

	// Создаем карту индексов колонок по заголовкам
	headerMap := make(map[string]int)
	if p.config.HasHeader && len(headers) > 0 {
		for i, header := range headers {
			headerMap[strings.ToLower(strings.TrimSpace(header))] = i
		}
	}

	// Определяем индексы нужных колонок
	colIndices := findGostColumnIndices(headerMap)

	var gosts []*Gost
	recordCount := 0

	// Parse data rows
	for {
		recordCount++
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			// Не останавливаем парсинг при ошибках чтения строк - просто пропускаем
			if p.errorCount < p.config.MaxErrors {
				p.config.ErrorCallback(fmt.Errorf("failed to read CSV row %d: %w", recordCount, err))
				p.errorCount++
			}
			// Продолжаем парсинг даже при ошибках
			continue
		}

		// Skip empty rows if configured
		if p.config.SkipEmptyRows && p.isEmptyRow(row) {
			continue
		}

		// Парсим строку используя маппинг колонок
		gost := &Gost{}
		
		// Извлекаем номер ГОСТа
		if colIndices.gostNumber >= 0 && colIndices.gostNumber < len(row) {
			gost.GostNumber = p.normalizeGostNumber(strings.TrimSpace(row[colIndices.gostNumber]))
		} else if len(row) > 0 {
			// Fallback на первую колонку
			gost.GostNumber = p.normalizeGostNumber(strings.TrimSpace(row[0]))
		}
		
		// Извлекаем название
		if colIndices.title >= 0 && colIndices.title < len(row) {
			gost.Title = strings.TrimSpace(row[colIndices.title])
		} else if len(row) > 1 {
			// Fallback на вторую колонку
			gost.Title = strings.TrimSpace(row[1])
		}
		
		// Извлекаем дату принятия
		if colIndices.adoptionDate >= 0 && colIndices.adoptionDate < len(row) {
			dateStr := strings.TrimSpace(row[colIndices.adoptionDate])
			if dateStr != "" {
				date, parseErr := p.parseDate(dateStr)
				if parseErr == nil && date != nil {
					gost.AdoptionDate = date
				}
			}
		}
		
		// Извлекаем дату вступления
		if colIndices.effectiveDate >= 0 && colIndices.effectiveDate < len(row) {
			dateStr := strings.TrimSpace(row[colIndices.effectiveDate])
			if dateStr != "" {
				date, parseErr := p.parseDate(dateStr)
				if parseErr == nil && date != nil {
					gost.EffectiveDate = date
				}
			}
		}
		
		// Извлекаем статус
		if colIndices.status >= 0 && colIndices.status < len(row) {
			gost.Status = strings.TrimSpace(row[colIndices.status])
		}
		
		// Извлекаем описание
		if colIndices.description >= 0 && colIndices.description < len(row) {
			gost.Description = strings.TrimSpace(row[colIndices.description])
		}
		
		// Извлекаем ключевые слова
		if colIndices.keywords >= 0 && colIndices.keywords < len(row) {
			gost.Keywords = strings.TrimSpace(row[colIndices.keywords])
		}
		
		// Устанавливаем тип источника
		gost.SourceType = sourceType
		
		// Проверяем, что обязательные поля заполнены (только номер ГОСТа обязателен)
		if gost.GostNumber == "" {
			if p.errorCount >= p.config.MaxErrors {
				// Не останавливаем парсинг, просто логируем
				p.logger.Printf("Warning: reached max error count (%d), continuing with warnings only", p.config.MaxErrors)
				p.errorCount = 0 // Сбрасываем счетчик, чтобы продолжить
			}
			p.config.ErrorCallback(fmt.Errorf("skipping row %d: missing GOST number", recordCount))
			p.errorCount++
			continue
		}
		
		// Название не является обязательным - некоторые записи могут его не иметь
		if gost.Title == "" {
			// Используем номер ГОСТа как название, если название отсутствует
			gost.Title = gost.GostNumber
		}

		// Normalize the GOST data
		if err := p.NormalizeGostData(gost); err != nil {
			p.config.ErrorCallback(fmt.Errorf("failed to normalize GOST data: %w", err))
			p.errorCount++
			continue
		}

		// Validate the GOST record
		if err := p.ValidateGostRecord(gost); err != nil {
			p.config.ErrorCallback(fmt.Errorf("invalid GOST record: %w", err))
			p.errorCount++
			continue
		}

		gosts = append(gosts, gost)
	}

	p.logger.Printf("Successfully parsed %d records from CSV", len(gosts))
	return gosts, nil
}

// ParseMultipleCSVFiles parses multiple CSV files and returns combined GOST records
func (p *GostParser) ParseMultipleCSVFiles(filePaths []string) ([]*Gost, error) {
	var allGosts []*Gost
	errorCount := 0

	for _, filePath := range filePaths {
		gosts, err := p.ParseCSVFile(filePath)
		if err != nil {
			p.config.ErrorCallback(fmt.Errorf("failed to parse file %s: %w", filePath, err))
			errorCount++
			continue
		}

		allGosts = append(allGosts, gosts...)
	}

	if errorCount > 0 {
		p.config.ErrorCallback(fmt.Errorf("completed with %d parsing errors", errorCount))
	}

	// Remove duplicates
	allGosts = p.DeduplicateGosts(allGosts)

	return allGosts, nil
}

// ParseNationalStandards parses CSV data in national standards format
func (p *GostParser) ParseNationalStandards(row []string) (*Gost, error) {
	if len(row) < 2 {
		return nil, errors.New("insufficient columns in national standards format")
	}

	gost := &Gost{
		GostNumber: p.normalizeGostNumber(strings.TrimSpace(row[0])),
		Title:      strings.TrimSpace(row[1]),
	}

	// Parse dates if available
	if len(row) > 2 && row[2] != "" {
		date, err := p.parseDate(strings.TrimSpace(row[2]))
		if err != nil {
			// Если дата не парсится, просто пропускаем её
			p.logger.Printf("Warning: failed to parse adoption date '%s': %v", row[2], err)
		} else if date != nil {
			gost.AdoptionDate = date
		}
	}

	if len(row) > 3 && row[3] != "" {
		date, err := p.parseDate(strings.TrimSpace(row[3]))
		if err != nil {
			// Если дата не парсится, просто пропускаем её
			p.logger.Printf("Warning: failed to parse effective date '%s': %v", row[3], err)
		} else if date != nil {
			gost.EffectiveDate = date
		}
	}

	// Parse status - может быть в разных позициях в зависимости от формата
	// Если есть дата вступления (row[3]), то статус в row[4]
	// Если нет даты вступления, то статус может быть в row[3]
	if len(row) > 3 {
		// Проверяем, является ли row[3] датой или статусом
		if gost.EffectiveDate == nil {
			// Если дата вступления не была распарсена, возможно row[3] - это статус
			// Проверяем, не является ли это датой
			if _, err := p.parseDate(strings.TrimSpace(row[3])); err != nil {
				// Если это не дата, то это статус
				gost.Status = strings.TrimSpace(row[3])
			}
		}
	}
	if len(row) > 4 {
		// Если есть row[4], это может быть статус (если была дата вступления) или URL
		if gost.Status == "" {
			gost.Status = strings.TrimSpace(row[4])
		} else {
			gost.SourceType = "national"
			gost.SourceURL = strings.TrimSpace(row[4])
		}
	}
	if len(row) > 5 {
		if gost.SourceURL == "" {
			gost.SourceType = "national"
			gost.SourceURL = strings.TrimSpace(row[5])
		} else {
			gost.Description = strings.TrimSpace(row[5])
		}
	}
	if len(row) > 6 {
		if gost.Description == "" {
			gost.Description = strings.TrimSpace(row[6])
		} else {
			gost.Keywords = strings.TrimSpace(row[6])
		}
	}
	if len(row) > 7 {
		if gost.Keywords == "" {
			gost.Keywords = strings.TrimSpace(row[7])
		}
	}
	
	// Устанавливаем SourceType по умолчанию
	if gost.SourceType == "" {
		gost.SourceType = "national"
	}

	return gost, nil
}

// ParseInterstateStandards parses CSV data in interstate standards format
func (p *GostParser) ParseInterstateStandards(row []string) (*Gost, error) {
	if len(row) < 2 {
		return nil, errors.New("insufficient columns in interstate standards format")
	}

	// Нормализуем номер ГОСТа, но если нормализация вернула пустую строку, используем исходный номер
	gostNumber := strings.TrimSpace(row[0])
	normalized := p.normalizeGostNumber(gostNumber)
	if normalized == "" {
		normalized = gostNumber // Используем исходный номер, если нормализация не удалась
	}

	gost := &Gost{
		GostNumber: normalized,
		Title:      strings.TrimSpace(row[1]),
	}

	// Parse dates if available
	if len(row) > 2 && row[2] != "" {
		date, err := p.parseDate(strings.TrimSpace(row[2]))
		if err != nil {
			// Если дата не парсится, просто пропускаем её
			p.logger.Printf("Warning: failed to parse adoption date '%s': %v", row[2], err)
		} else if date != nil {
			gost.AdoptionDate = date
		}
	}

	if len(row) > 3 && row[3] != "" {
		date, err := p.parseDate(strings.TrimSpace(row[3]))
		if err != nil {
			// Если дата не парсится, просто пропускаем её
			p.logger.Printf("Warning: failed to parse effective date '%s': %v", row[3], err)
		} else if date != nil {
			gost.EffectiveDate = date
		}
	}

	// Parse other fields
	if len(row) > 4 {
		gost.Status = strings.TrimSpace(row[4])
	}
	if len(row) > 5 {
		gost.SourceType = "interstate"
		gost.SourceURL = strings.TrimSpace(row[5])
	}
	if len(row) > 6 {
		gost.Description = strings.TrimSpace(row[6])
	}
	if len(row) > 7 {
		gost.Keywords = strings.TrimSpace(row[7])
	}

	return gost, nil
}

// ParseTechCommit parses CSV data in technical committee format
func (p *GostParser) ParseTechCommit(row []string) (*Gost, error) {
	if len(row) < 2 {
		return nil, errors.New("insufficient columns in tech committee format")
	}

	gost := &Gost{
		GostNumber: p.normalizeGostNumber(strings.TrimSpace(row[0])),
		Title:      strings.TrimSpace(row[1]),
	}

	// Parse dates if available
	if len(row) > 2 && row[2] != "" {
		date, err := p.parseDate(strings.TrimSpace(row[2]))
		if err != nil {
			// Если дата не парсится, просто пропускаем её
			p.logger.Printf("Warning: failed to parse adoption date '%s': %v", row[2], err)
		} else if date != nil {
			gost.AdoptionDate = date
		}
	}

	if len(row) > 3 && row[3] != "" {
		date, err := p.parseDate(strings.TrimSpace(row[3]))
		if err != nil {
			// Если дата не парсится, просто пропускаем её
			p.logger.Printf("Warning: failed to parse effective date '%s': %v", row[3], err)
		} else if date != nil {
			gost.EffectiveDate = date
		}
	}

	// Parse other fields
	if len(row) > 4 {
		gost.Status = strings.TrimSpace(row[4])
	}
	if len(row) > 5 {
		gost.SourceType = "tech_commit"
		gost.SourceURL = strings.TrimSpace(row[5])
	}
	if len(row) > 6 {
		gost.Description = strings.TrimSpace(row[6])
	}
	if len(row) > 7 {
		gost.Keywords = strings.TrimSpace(row[7])
	}

	return gost, nil
}

// AutoDetectFormat automatically detects the CSV format based on content
func (p *GostParser) AutoDetectFormat(data []byte) (string, error) {
	// Convert to string for analysis
	str := strings.ToLower(string(data))

	// Check for interstate format first (more specific patterns)
	if strings.Contains(str, "сн ") ||
		strings.Contains(str, "снип") ||
		strings.Contains(str, "межгосударственный") ||
		strings.Contains(str, "союз") {
		return "interstate", nil
	}

	// Check for tech commit format
	if strings.Contains(str, "технический") ||
		strings.Contains(str, "комитет") ||
		strings.Contains(str, "тк") {
		return "tech_commit", nil
	}

	// Check for national format
	if strings.Contains(str, "гост р") ||
		strings.Contains(str, "российский") ||
		strings.Contains(str, "национальный") ||
		strings.Contains(str, "гост") {
		return "national", nil
	}

	// Default to national format
	return "national", nil
}

// NormalizeGostData normalizes and cleans parsed GOST data
func (p *GostParser) NormalizeGostData(gost *Gost) error {
	if gost == nil {
		return errors.New("GOST record is nil")
	}

	// Normalize GOST number
	originalNumber := gost.GostNumber
	gost.GostNumber = p.normalizeGostNumber(gost.GostNumber)
	if gost.GostNumber == "" {
		// Если нормализация не удалась, но исходный номер содержит "ГОСТ", используем его
		if strings.Contains(strings.ToUpper(originalNumber), "ГОСТ") {
			gost.GostNumber = strings.TrimSpace(originalNumber)
		} else {
			return errors.New("normalized GOST number is empty")
		}
	}

	// Normalize whitespace in title
	gost.Title = strings.TrimSpace(gost.Title)
	// Название не обязательно - если пустое, используем номер ГОСТа
	if gost.Title == "" {
		gost.Title = gost.GostNumber
	}

	// Normalize dates if configured
	if p.config.NormalizeDates {
		if gost.AdoptionDate != nil {
			normalized := gost.AdoptionDate.Format("2006-01-02")
			date, err := p.parseDate(normalized)
			if err == nil && date != nil {
				gost.AdoptionDate = date
			}
		}

		if gost.EffectiveDate != nil {
			normalized := gost.EffectiveDate.Format("2006-01-02")
			date, err := p.parseDate(normalized)
			if err == nil && date != nil {
				gost.EffectiveDate = date
			}
		}
	}

	// Normalize status
	gost.Status = strings.TrimSpace(gost.Status)
	gost.Status = strings.ToLower(gost.Status)

	// Normalize source type
	gost.SourceType = strings.TrimSpace(gost.SourceType)
	gost.SourceType = strings.ToLower(gost.SourceType)

	// Normalize other text fields
	gost.Description = strings.TrimSpace(gost.Description)
	gost.Keywords = strings.TrimSpace(gost.Keywords)
	gost.SourceURL = strings.TrimSpace(gost.SourceURL)

	return nil
}

// ValidateGostRecord validates a parsed GOST record
// Возвращает ошибки только для критических проблем, предупреждения для некритичных
func (p *GostParser) ValidateGostRecord(gost *Gost) error {
	if gost == nil {
		return errors.New("GOST record is nil")
	}

	if gost.GostNumber == "" {
		return errors.New("GOST number is required")
	}

	// Проверяем формат номера ГОСТа, но не блокируем импорт для нестандартных форматов
	if !p.isValidGostNumber(gost.GostNumber) {
		// Это предупреждение, а не критическая ошибка
		return fmt.Errorf("non-standard GOST number format: %s", gost.GostNumber)
	}

	// Название не является обязательным - некоторые записи могут его не иметь
	if gost.Title == "" {
		// Это предупреждение, не ошибка
		return fmt.Errorf("GOST title is empty for %s", gost.GostNumber)
	}

	// Validate status if present
	if gost.Status != "" {
		validStatuses := map[string]bool{
			"действующий": true,
			"утвержден":   true,
			"отменен":     true,
			"заменен":     true,
			"введен":      true,
			"взамен":      true,
			"cancelled":   true,
			"replaced":    true,
			"active":      true,
			"approved":    true,
			"deprecated":  true,
		}
		if !validStatuses[gost.Status] {
			p.logger.Printf("Warning: unknown status '%s' for GOST %s", gost.Status, gost.GostNumber)
		}
	}

	return nil
}

// DeduplicateGosts removes duplicate records from a slice of GOSTs
func (p *GostParser) DeduplicateGosts(gosts []*Gost) []*Gost {
	if len(gosts) == 0 {
		return gosts
	}

	seen := make(map[string]bool)
	var unique []*Gost

	for _, gost := range gosts {
		if !seen[gost.GostNumber] {
			seen[gost.GostNumber] = true
			unique = append(unique, gost)
		}
	}

	p.logger.Printf("Removed %d duplicate records, %d unique records remain",
		len(gosts)-len(unique), len(unique))

	return unique
}

// hasCyrillicRunes checks if string contains Cyrillic characters
func hasCyrillicRunes(s string) bool {
	for _, r := range s {
		if r >= 0x0400 && r <= 0x04FF { // Cyrillic Unicode range
			return true
		}
	}
	return false
}

// detectAndConvertEncoding detects file encoding and converts to UTF-8 if necessary
func (p *GostParser) detectAndConvertEncoding(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Сначала проверяем, может быть файл уже в UTF-8, но содержит некорректные символы
	// Это может быть признак того, что файл был неправильно декодирован ранее
	if utf8.Valid(data) {
		str := string(data)
		// Если файл содержит некорректные символы типа "╨У╨Ю╨б╨в", это значит,
		// что UTF-8 файл был декодирован как Windows-1251, а потом снова сохранен как UTF-8
		// В этом случае нужно попробовать декодировать как Windows-1251
		// Проверяем наличие некорректных символов двойной конвертации
		hasDoubleEncoding := strings.Contains(str, "╨У") || strings.Contains(str, "╨Ю") || 
		                     strings.Contains(str, "╨б") || strings.Contains(str, "╨в")
		
		if hasDoubleEncoding {
			if p.logger != nil {
				p.logger.Printf("Detected double-encoded file (contains ╨У╨Ю╨б╨в), attempting aggressive fix...")
			}
			
			// Файл содержит символы двойной конвертации
			// Пробуем декодировать исходные данные как Windows-1251
			decoder := charmap.Windows1251.NewDecoder()
			decoded, _, err := transform.Bytes(decoder, data)
			if err == nil && len(decoded) > 0 && utf8.Valid(decoded) {
				decodedStr := string(decoded)
				
				// Если после первого декодирования все еще есть некорректные символы,
				// значит файл был декодирован дважды - декодируем еще раз
				if strings.Contains(decodedStr, "╨У") || strings.Contains(decodedStr, "╨Ю") {
					if p.logger != nil {
						p.logger.Printf("First decode still has invalid chars, attempting second decode...")
					}
					// Декодируем еще раз - это двойное декодирование
					decoded2, _, err2 := transform.Bytes(decoder, decoded)
					if err2 == nil && len(decoded2) > 0 && utf8.Valid(decoded2) {
						decodedStr2 := string(decoded2)
						// Если после второго декодирования все еще есть некорректные символы,
						// пробуем третий раз (тройное декодирование)
						if strings.Contains(decodedStr2, "╨У") || strings.Contains(decodedStr2, "╨Ю") {
							if p.logger != nil {
								p.logger.Printf("Second decode still has invalid chars, attempting third decode...")
							}
							decoded3, _, err3 := transform.Bytes(decoder, decoded2)
							if err3 == nil && len(decoded3) > 0 && utf8.Valid(decoded3) {
								decodedStr3 := string(decoded3)
								if hasCyrillicRunes(decodedStr3) && 
								   !strings.Contains(decodedStr3, "╨У") && 
								   !strings.Contains(decodedStr3, "╨Ю") &&
								   strings.Contains(decodedStr3, "ГОСТ") {
									if p.logger != nil {
										p.logger.Printf("Successfully fixed using triple Windows-1251 decode")
									}
									return decoded3, nil
								}
							}
						}
						
						// Проверяем результат второго декодирования
						if hasCyrillicRunes(decodedStr2) && 
						   !strings.Contains(decodedStr2, "╨У") && 
						   !strings.Contains(decodedStr2, "╨Ю") &&
						   strings.Contains(decodedStr2, "ГОСТ") {
							if p.logger != nil {
								p.logger.Printf("Successfully fixed using double Windows-1251 decode")
							}
							return decoded2, nil
						}
					}
				}
				
				// Проверяем результат первого декодирования
				if hasCyrillicRunes(decodedStr) && 
				   !strings.Contains(decodedStr, "╨У") && 
				   !strings.Contains(decodedStr, "╨Ю") &&
				   strings.Contains(decodedStr, "ГОСТ") {
					if p.logger != nil {
						p.logger.Printf("Successfully fixed double-encoded file using Windows-1251 decoder")
					}
					return decoded, nil
				}
			}
		}
	}

	// Список кодировок для проверки (в порядке приоритета)
	// Windows-1251 проверяем первым, так как это наиболее вероятная кодировка для русских данных
	encodings := []struct {
		name string
		enc  encoding.Encoding
	}{
		{"Windows-1251", charmap.Windows1251}, // Проверяем первым для русских данных
		{"UTF-8", encoding.Nop}, // UTF-8 без преобразования
		{"KOI8-R", charmap.KOI8R},
		{"ISO-8859-5", charmap.ISO8859_5},
	}

	bestResult := data
	bestScore := 0
	bestEncoding := "unknown"

	// Пробуем каждую кодировку и оцениваем результат
	for _, enc := range encodings {
		var decoded []byte
		var err error

		if enc.name == "UTF-8" {
			// Для UTF-8 просто проверяем валидность
			if utf8.Valid(data) {
				decoded = data
				// ВАЖНО: Если файл уже в UTF-8, но содержит некорректные символы "╨У╨Ю╨б╨в",
				// это означает, что файл был неправильно сохранен - пропускаем этот вариант
				decodedStrCheck := string(decoded)
				if strings.Contains(decodedStrCheck, "╨У") || strings.Contains(decodedStrCheck, "╨Ю") {
					// Файл в UTF-8, но содержит некорректные символы - пропускаем
					if p.logger != nil {
						p.logger.Printf("Skipping UTF-8: contains invalid characters (╨У╨Ю╨б╨в)")
					}
					continue
				}
			} else {
				continue
			}
		} else {
			// Декодируем через трансформацию
			decoder := enc.enc.NewDecoder()
			decoded, _, err = transform.Bytes(decoder, data)
			if err != nil || len(decoded) == 0 {
				continue
			}
		}

		// Проверяем, что результат валидный UTF-8
		if !utf8.Valid(decoded) {
			continue
		}

		decodedStr := string(decoded)
		
		// КРИТИЧЕСКИ ВАЖНО: Проверяем наличие некорректных символов СРАЗУ после декодирования
		// Символы "╨У╨Ю╨б╨в" появляются когда UTF-8 файл декодируется как Windows-1251
		// Если они есть, этот вариант НЕПРАВИЛЬНЫЙ и должен быть полностью исключен
		hasInvalidChars := strings.Contains(decodedStr, "╨У") || strings.Contains(decodedStr, "╨Ю") || 
		                   strings.Contains(decodedStr, "╨б") || strings.Contains(decodedStr, "╨в")
		
		// Если есть некорректные символы, полностью пропускаем этот вариант
		// Неважно какой у него score - он неправильный
		if hasInvalidChars {
			if p.logger != nil {
				// Показываем пример некорректного текста для отладки
				sample := decodedStr
				if len(sample) > 100 {
					sample = sample[:100]
				}
				p.logger.Printf("Skipping %s encoding: contains invalid characters (╨У╨Ю╨б╨в) - sample: %s", enc.name, sample)
			}
			continue // Пропускаем этот вариант полностью
		}
		
		// Только если нет некорректных символов, оцениваем score
		score := p.scoreEncoding(decodedStr)
		
		// Дополнительная проверка: если score хороший, но есть подозрительные символы, пропускаем
		if score > 0 {
			// Проверяем, что результат содержит правильные кириллические символы
			if !hasCyrillicRunes(decodedStr) {
				if p.logger != nil {
					p.logger.Printf("Skipping %s encoding: no Cyrillic characters found", enc.name)
				}
				continue
			}
		}

		// Если это лучший результат, сохраняем его
		// Но только если нет некорректных символов (дополнительная проверка)
		if score > bestScore {
			// Двойная проверка на некорректные символы
			if !strings.Contains(decodedStr, "╨У") && !strings.Contains(decodedStr, "╨Ю") {
				bestScore = score
				bestResult = decoded
				bestEncoding = enc.name
			} else {
				if p.logger != nil {
					p.logger.Printf("Skipping %s encoding: score %d but contains invalid characters", enc.name, score)
				}
			}
		}
	}

	// Если нашли хороший вариант, используем его
	if bestScore > 0 {
		// КРИТИЧЕСКАЯ ПРОВЕРКА: Проверяем финальный результат на некорректные символы
		finalStr := string(bestResult)
		hasFinalInvalidChars := strings.Contains(finalStr, "╨У") || strings.Contains(finalStr, "╨Ю") || 
		                        strings.Contains(finalStr, "╨б") || strings.Contains(finalStr, "╨в")
		
		// ВСЕГДА принудительно декодируем как Windows-1251, если есть некорректные символы
		// Это даст нам правильные кириллические символы в UTF-8 для хранения в базе
		if hasFinalInvalidChars {
			// Финальный результат содержит некорректные символы - это КРИТИЧЕСКАЯ ОШИБКА
			// Пробуем декодировать исходные данные как Windows-1251 напрямую (принудительно)
			// Это должно дать нам правильные кириллические символы в UTF-8
			if p.logger != nil {
				p.logger.Printf("CRITICAL: Final result contains invalid chars (╨У╨Ю╨б╨в), forcing Windows-1251 decode to get proper Cyrillic in UTF-8")
			}
			decoder := charmap.Windows1251.NewDecoder()
			fixed, _, err := transform.Bytes(decoder, data)
			if err == nil && len(fixed) > 0 && utf8.Valid(fixed) {
				fixedStr := string(fixed)
				// Проверяем, что после принудительного декодирования нет некорректных символов
				// и есть правильные кириллические символы в UTF-8
				if !strings.Contains(fixedStr, "╨У") && !strings.Contains(fixedStr, "╨Ю") &&
				   hasCyrillicRunes(fixedStr) && strings.Contains(fixedStr, "ГОСТ") {
					if p.logger != nil {
						p.logger.Printf("Successfully fixed: Windows-1251 -> UTF-8, now we have proper Cyrillic characters for database storage")
					}
					return fixed, nil
				} else {
					// Если после первого декодирования все еще есть проблемы, пробуем еще раз
					if strings.Contains(fixedStr, "╨У") || strings.Contains(fixedStr, "╨Ю") {
						if p.logger != nil {
							p.logger.Printf("First forced decode still has invalid chars, trying double decode...")
						}
						fixed2, _, err2 := transform.Bytes(decoder, fixed)
						if err2 == nil && len(fixed2) > 0 && utf8.Valid(fixed2) {
							fixedStr2 := string(fixed2)
							if !strings.Contains(fixedStr2, "╨У") && !strings.Contains(fixedStr2, "╨Ю") &&
							   hasCyrillicRunes(fixedStr2) && strings.Contains(fixedStr2, "ГОСТ") {
								if p.logger != nil {
									p.logger.Printf("Successfully fixed using double Windows-1251 decode -> UTF-8 with proper Cyrillic")
								}
								return fixed2, nil
							}
						}
					}
					if p.logger != nil {
						p.logger.Printf("WARNING: Forced decode still has issues - fixedStr contains ГОСТ: %v, hasCyrillic: %v, hasInvalid: %v",
							strings.Contains(fixedStr, "ГОСТ"), hasCyrillicRunes(fixedStr),
							strings.Contains(fixedStr, "╨У") || strings.Contains(fixedStr, "╨Ю"))
					}
				}
			}
			// Если принудительное декодирование не помогло, возвращаем как есть (но это ошибка)
			if p.logger != nil {
				p.logger.Printf("WARNING: Could not fix encoding, returning result with invalid chars")
			}
		}
		
		if p.logger != nil {
			p.logger.Printf("Detected encoding: %s (score: %d)", bestEncoding, bestScore)
		}
		return bestResult, nil
	}

	// Если ничего не подошло, возвращаем оригинальные данные
	return data, nil
}

// scoreEncoding оценивает качество декодирования по количеству валидных кириллических символов
func (p *GostParser) scoreEncoding(text string) int {
	score := 0
	textLower := strings.ToLower(text)

	// Проверяем наличие ключевых русских слов
	keywords := []string{
		"гост",
		"название",
		"наименование",
		"дата",
		"принятия",
		"утверждения",
		"действующий",
		"отменен",
		"заменен",
		"стандарт",
		"номер",
		"обозначение",
	}

	for _, keyword := range keywords {
		if strings.Contains(textLower, keyword) {
			score += 10
		}
	}

	// Подсчитываем количество кириллических символов
	cyrillicCount := 0
	for _, r := range text {
		if r >= 0x0400 && r <= 0x04FF { // Cyrillic Unicode range
			cyrillicCount++
		}
	}

	// Добавляем баллы за кириллицу (но не слишком много, чтобы не перевесить ключевые слова)
	score += cyrillicCount / 100

	// Штраф за некорректные символы (например, "╨У╨Ю╨б╨в" вместо "ГОСТ")
	invalidPatterns := []string{
		"╨У", "╨Ю", "╨б", "╨в", // Типичные артефакты неправильной кодировки
		"Ð", "Ñ", "Ð", "Ñ", // Другие возможные артефакты
	}

	for _, pattern := range invalidPatterns {
		if strings.Contains(text, pattern) {
			score -= 200 // Очень большой штраф за некорректные символы
		}
	}

	// Проверяем, что текст не содержит только мусор
	if len(text) > 100 && cyrillicCount < 10 {
		score -= 20 // Штраф за малое количество кириллицы в большом тексте
	}

	return score
}

// parseDate parses a date string in various formats
func (p *GostParser) parseDate(dateStr string) (*time.Time, error) {
	// Remove any extra whitespace
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return nil, nil
	}

	// Try various date formats
	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"02/01/2006",
		"2006-01-02 15:04:05",
		"02.01.2006 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006/01/02",
		"02-01-2006",
		"2006.01.02",
		"01/02/2006", // American format
		"20060102",   // Compact format
		"02.01.06",   // Short year format
		"02/01/06",   // Short year format
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unsupported date format: %s", dateStr)
}

// normalizeGostNumber normalizes GOST number to standard format
func (p *GostParser) normalizeGostNumber(number string) string {
	number = strings.TrimSpace(number)
	if number == "" {
		return ""
	}

	// Remove extra whitespace
	number = regexp.MustCompile(`\s+`).ReplaceAllString(number, " ")

		// Pattern for GOST number: ГОСТ (Р)? number-year
		// Support different delimiters: -, –, —
		// Также поддерживаем форматы с точками: ГОСТ Р 1.1-2020, ГОСТ Р 2.001-2023
		gostPattern := regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*)?(\d+(?:\.\d+)*)\s*[-–—]\s*(\d{4})$`)
		matches := gostPattern.FindStringSubmatch(number)

		if len(matches) == 4 {
			// Format as standard: ГОСТ number-year (сохраняем точки в номере)
			return fmt.Sprintf("ГОСТ %s-%s", matches[2], matches[3])
		}

		// Try pattern without GOST prefix (number-year only)
		simplePattern := regexp.MustCompile(`(?i)^(\d+(?:\.\d+)*)\s*[-–—]\s*(\d{4})$`)
		simpleMatches := simplePattern.FindStringSubmatch(number)
		if len(simpleMatches) == 3 {
			return fmt.Sprintf("ГОСТ %s-%s", simpleMatches[1], simpleMatches[2])
		}
		
		// Try pattern with 2-digit year
		shortYearPattern := regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*)?(\d+(?:\.\d+)*)\s*[-–—]\s*(\d{2})$`)
		shortMatches := shortYearPattern.FindStringSubmatch(number)
		if len(shortMatches) == 4 {
			year := shortMatches[3]
			if len(year) == 2 {
				// Преобразуем 2-значный год в 4-значный (предполагаем 20xx)
				year = "20" + year
			}
			return fmt.Sprintf("ГОСТ %s-%s", shortMatches[2], year)
		}

	// If doesn't match pattern but contains "ГОСТ", return as is (normalize spaces)
	if strings.Contains(strings.ToUpper(number), "ГОСТ") {
		// Normalize spaces around dash
		normalized := regexp.MustCompile(`\s*[-–—]\s*`).ReplaceAllString(number, "-")
		return normalized
	}

	return ""
}

// isEmptyRow checks if a row is empty
func (p *GostParser) isEmptyRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// isValidGostNumber checks if a GOST number is valid
func (p *GostParser) isValidGostNumber(number string) bool {
	if number == "" {
		return false
	}

	// Паттерны для различных форматов номеров ГОСТов:
	// - ГОСТ Р 12345-2020
	// - ГОСТ 12345-2020
	// - ГОСТ Р ИСО 12345-2020
	// - ГОСТ Р 1.1-2020 (с точками)
	// - ГОСТ Р 2.001-2023 (с точками и нулями)
	// - ГОСТ Р 7.0.0-2024 (с несколькими точками)
	// - ГОСТ Р 2.901-99 (старый формат без 20xx)
	
	patterns := []*regexp.Regexp{
		// Стандартный формат: ГОСТ Р 12345-2020
		regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*(ИСО\s*)?\d+)\s*[-–—]\s*(\d{4})$`),
		// Формат с точками: ГОСТ Р 1.1-2020
		regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*\d+\.\d+)\s*[-–—]\s*(\d{4})$`),
		// Формат с несколькими точками: ГОСТ Р 7.0.0-2024
		regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*\d+\.\d+\.\d+)\s*[-–—]\s*(\d{4})$`),
		// Формат с нулями: ГОСТ Р 2.001-2023
		regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*\d+\.\d{3})\s*[-–—]\s*(\d{4})$`),
		// Старый формат: ГОСТ Р 2.901-99
		regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*\d+\.\d+)\s*[-–—]\s*(\d{2})$`),
	}
	
	for _, pattern := range patterns {
		if pattern.MatchString(number) {
			return true
		}
	}
	
	return false
}

// parseDate парсит дату из строки в различных форматах (обычная функция для использования вне парсера)
func parseDate(dateStr string) *time.Time {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return nil
	}

	formats := []string{
		"2006-01-02",
		"02.01.2006",
		"02/01/2006",
		"2006-01-02 15:04:05",
		"02.01.2006 15:04:05",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006/01/02",
		"02-01-2006",
		"2006.01.02",
		"01/02/2006",
		"20060102",
		"02.01.06",
		"02/01/06",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return &t
		}
	}

	return nil
}

// ParseGostCSVFromReader парсит CSV из io.Reader (для использования с HTTP запросами)
func ParseGostCSVFromReader(reader io.Reader) ([]GostRecord, error) {
	// Читаем данные для определения формата
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	
	// Используем GostParser для парсинга (он умеет определять кодировку и формат)
	logger := &simpleLogger{}
	config := DefaultParserConfig()
	parser := NewGostParser(config, logger)
	
	// Парсим данные
	gosts, err := parser.ParseCSVData(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV data: %w", err)
	}
	
	// Конвертируем из []*Gost в []GostRecord
	records := make([]GostRecord, 0, len(gosts))
	for _, gost := range gosts {
		record := GostRecord{
			GostNumber:    gost.GostNumber,
			Title:         gost.Title,
			Status:        gost.Status,
			Description:   gost.Description,
			Keywords:      gost.Keywords,
		}
		if gost.AdoptionDate != nil {
			record.AdoptionDate = gost.AdoptionDate
		}
		if gost.EffectiveDate != nil {
			record.EffectiveDate = gost.EffectiveDate
		}
		records = append(records, record)
	}
	
	return records, nil
}

// simpleLogger простой логгер для парсера
type simpleLogger struct{}

func (l *simpleLogger) Printf(format string, v ...interface{}) {
	// Игнорируем логи при парсинге через ParseGostCSVFromReader
}

// ParseGostCSV парсит CSV файл с ГОСТами из Росстандарта (обратная совместимость)
func ParseGostCSV(filePath string) ([]GostRecord, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	return ParseGostCSVFromReader(file)
}

// isEmptyGostRow проверяет, является ли строка пустой (обычная функция для обратной совместимости)
func isEmptyGostRow(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

// normalizeGostNumber нормализует номер ГОСТа (обычная функция для обратной совместимости)
func normalizeGostNumber(number string) string {
	number = strings.TrimSpace(number)
	if number == "" {
		return ""
	}

	number = regexp.MustCompile(`\s+`).ReplaceAllString(number, " ")
	
	gostPattern := regexp.MustCompile(`(?i)^(ГОСТ\s*Р?\s*)?(\d+)\s*[-–—]\s*(\d{4})$`)
	matches := gostPattern.FindStringSubmatch(number)
	
	if len(matches) == 4 {
		return fmt.Sprintf("ГОСТ %s-%s", matches[2], matches[3])
	}
	
	if strings.Contains(strings.ToUpper(number), "ГОСТ") {
		return number
	}
	
	return ""
}

// findGostColumnIndices определяет индексы колонок для парсинга ГОСТов
func findGostColumnIndices(headerMap map[string]int) GostColumnIndices {
	indices := GostColumnIndices{
		gostNumber:    -1,
		title:         -1,
		adoptionDate:  -1,
		effectiveDate: -1,
		status:        -1,
		description:   -1,
		keywords:      -1,
	}

	// Ищем номер ГОСТа
	for key, val := range headerMap {
		if strings.Contains(key, "номер") || strings.Contains(key, "обозначение") {
			indices.gostNumber = val
			break
		}
	}

	// Ищем название
	for key, val := range headerMap {
		if strings.Contains(key, "название") || strings.Contains(key, "наименование") {
			indices.title = val
			break
		}
	}

	// Ищем дату принятия
	for key, val := range headerMap {
		if strings.Contains(key, "дата принятия") || strings.Contains(key, "дата утверждения") {
			indices.adoptionDate = val
			break
		}
	}

	// Ищем дату вступления в силу
	for key, val := range headerMap {
		if strings.Contains(key, "дата вступления") || strings.Contains(key, "дата введения") {
			indices.effectiveDate = val
			break
		}
	}

	// Ищем статус
	for key, val := range headerMap {
		if strings.Contains(key, "статус") {
			indices.status = val
			break
		}
	}

	// Ищем описание
	for key, val := range headerMap {
		if strings.Contains(key, "описание") || strings.Contains(key, "область применения") {
			indices.description = val
			break
		}
	}

	// Ищем ключевые слова
	for key, val := range headerMap {
		if strings.Contains(key, "ключевые слова") || strings.Contains(key, "коды") {
			indices.keywords = val
			break
		}
	}

	return indices
}

// parseGostRow парсит строку CSV в структуру GostRecord (обычная функция для обратной совместимости)
func parseGostRow(row []string, indices GostColumnIndices) GostRecord {
	record := GostRecord{}

	// Извлекаем номер ГОСТа
	if indices.gostNumber >= 0 && indices.gostNumber < len(row) {
		record.GostNumber = strings.TrimSpace(row[indices.gostNumber])
	}

	// Извлекаем название
	if indices.title >= 0 && indices.title < len(row) {
		record.Title = strings.TrimSpace(row[indices.title])
	}

	// Извлекаем дату принятия
	if indices.adoptionDate >= 0 && indices.adoptionDate < len(row) {
		dateStr := strings.TrimSpace(row[indices.adoptionDate])
		if dateStr != "" {
			if date, err := time.Parse("2006-01-02", dateStr); err == nil {
				record.AdoptionDate = &date
			} else if date, err := time.Parse("02.01.2006", dateStr); err == nil {
				record.AdoptionDate = &date
			}
		}
	}

	// Извлекаем дату вступления в силу
	if indices.effectiveDate >= 0 && indices.effectiveDate < len(row) {
		dateStr := strings.TrimSpace(row[indices.effectiveDate])
		if dateStr != "" {
			if date, err := time.Parse("2006-01-02", dateStr); err == nil {
				record.EffectiveDate = &date
			} else if date, err := time.Parse("02.01.2006", dateStr); err == nil {
				record.EffectiveDate = &date
			}
		}
	}

	// Извлекаем статус
	if indices.status >= 0 && indices.status < len(row) {
		record.Status = strings.TrimSpace(row[indices.status])
	}

	// Извлекаем описание
	if indices.description >= 0 && indices.description < len(row) {
		record.Description = strings.TrimSpace(row[indices.description])
	}

	// Извлекаем ключевые слова
	if indices.keywords >= 0 && indices.keywords < len(row) {
		record.Keywords = strings.TrimSpace(row[indices.keywords])
	}

	return record
}

// ParseCSVFromReader parses CSV data from io.Reader and returns GOST records
func (p *GostParser) ParseCSVFromReader(reader io.Reader) ([]*Gost, error) {
	// Read all data from reader
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read from reader: %w", err)
	}
	
	// Use ParseCSVData to parse the data
	return p.ParseCSVData(data)
}

// ParseCSVFromJSON parses JSON data and returns GOST records
func (p *GostParser) ParseCSVFromJSON(jsonData string) ([]*Gost, error) {
	var records []struct {
		Number       string `json:"number"`
		Title        string `json:"title"`
		AdoptionDate string `json:"adoption_date"`
		Status       string `json:"status"`
	}
	
	if err := json.Unmarshal([]byte(jsonData), &records); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	var gosts []*Gost
	for _, record := range records {
		gost := &Gost{
			GostNumber: record.Number,
			Title:      record.Title,
			Status:     record.Status,
		}
		
		// Parse adoption date if present
		if record.AdoptionDate != "" {
			if date, err := p.parseDate(record.AdoptionDate); err == nil && date != nil {
				gost.AdoptionDate = date
			}
		}
		
		// Normalize and validate
		if err := p.NormalizeGostData(gost); err != nil {
			p.logger.Printf("Warning: failed to normalize GOST %s: %v", record.Number, err)
			continue
		}
		
		if err := p.ValidateGostRecord(gost); err != nil {
			p.logger.Printf("Warning: failed to validate GOST %s: %v", record.Number, err)
			continue
		}
		
		gosts = append(gosts, gost)
	}
	
	return gosts, nil
}

