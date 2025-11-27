package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"httpserver/database"
	"httpserver/importer"
)

// Список всех источников данных Росстандарта (50 источников)
var gostSources = []struct {
	name string
	url  string
}{
	{"tulist", "https://www.rst.gov.ru/opendata/7706406291-tulist"},
	{"nationalstandards", "https://www.rst.gov.ru/opendata/7706406291-nationalstandards"},
	{"interstatestandards", "https://www.rst.gov.ru/opendata/7706406291-interstatestandards"},
	{"techcommit", "https://www.rst.gov.ru/opendata/7706406291-techcommit"},
	{"publiccouncils", "https://www.rst.gov.ru/opendata/7706406291-publiccouncils"},
	{"gosuslugi", "https://www.rst.gov.ru/opendata/7706406291-gosuslugi"},
	{"npa", "https://www.rst.gov.ru/opendata/7706406291-npa"},
	{"zased", "https://www.rst.gov.ru/opendata/7706406291-zased"},
	{"plan", "https://www.rst.gov.ru/opendata/7706406291-plan"},
	{"reqsmi", "https://www.rst.gov.ru/opendata/7706406291-reqsmi"},
	{"listinstrros", "https://www.rst.gov.ru/opendata/7706406291-listinstrros"},
	{"vniiftricouncilplan", "https://www.rst.gov.ru/opendata/7706406291-vniiftricouncilplan"},
	{"memoranda", "https://www.rst.gov.ru/opendata/7706406291-memoranda"},
	{"regulations", "https://www.rst.gov.ru/opendata/7706406291-regulations"},
	{"publiccouncilplan", "https://www.rst.gov.ru/opendata/7706406291-publiccouncilplan"},
	{"rosstandartsystems", "https://www.rst.gov.ru/opendata/7706406291-rosstandartsystems"},
	{"incomeemployees", "https://www.rst.gov.ru/opendata/7706406291-incomeemployees"},
	{"efficiencyfact", "https://www.rst.gov.ru/opendata/7706406291-efficiencyfact"},
	{"indicatorsindustry", "https://www.rst.gov.ru/opendata/7706406291-indicatorsindustry"},
	{"targetsindustry", "https://www.rst.gov.ru/opendata/7706406291-targetsindustry"},
	{"efficiencyplan", "https://www.rst.gov.ru/opendata/7706406291-efficiencyplan"},
	{"budgetappropriations", "https://www.rst.gov.ru/opendata/7706406291-budgetappropriations"},
	{"gostevents", "https://www.rst.gov.ru/opendata/7706406291-gostevents"},
	{"establishedmedia", "https://www.rst.gov.ru/opendata/7706406291-establishedmedia"},
	{"standartcouncil", "https://www.rst.gov.ru/opendata/7706406291-standartcouncil"},
	{"vacanciesinfo", "https://www.rst.gov.ru/opendata/7706406291-vacanciesinfo"},
	{"anticorruption", "https://www.rst.gov.ru/opendata/7706406291-anticorruption"},
	{"stateprograms", "https://www.rst.gov.ru/opendata/7706406291-stateprograms"},
	{"etalonros", "https://www.rst.gov.ru/opendata/7706406291-etalonros"},
	{"citizensappeals", "https://www.rst.gov.ru/opendata/7706406291-citizensappeals"},
	{"rosstandartstructure", "https://www.rst.gov.ru/opendata/7706406291-rosstandartstructure"},
	{"informationsystems", "https://www.rst.gov.ru/opendata/7706406291-informationsystems"},
	{"podved", "https://www.rst.gov.ru/opendata/7706406291-podved"},
	{"rules", "https://www.rst.gov.ru/opendata/7706406291-rules"},
	{"interaction", "https://www.rst.gov.ru/opendata/7706406291-interaction"},
	{"controlresult", "https://www.rst.gov.ru/opendata/7706406291-controlresult"},
	{"controlplan", "https://www.rst.gov.ru/opendata/7706406291-controlplan"},
	{"incomepodved", "https://www.rst.gov.ru/opendata/7706406291-incomepodved"},
	{"ndtlist", "https://www.rst.gov.ru/opendata/7706406291-ndtlist"},
	{"verification", "https://www.rst.gov.ru/opendata/7706406291-verification"},
	{"nssregistry", "https://www.rst.gov.ru/opendata/7706406291-nssregistry"},
	{"nssblacklist", "https://www.rst.gov.ru/opendata/7706406291--nssblacklist"},
	{"rstmaingoals", "https://www.rst.gov.ru/opendata/7706406291-rstmaingoals"},
	{"koomet", "https://www.rst.gov.ru/opendata/7706406291-koomet"},
	{"eaeutechregs", "https://www.rst.gov.ru/opendata/7706406291-eaeutechregs"},
	{"orglist", "https://www.rst.gov.ru/opendata/7706406291-orglist"},
	{"timezones", "https://www.rst.gov.ru/opendata/7706406291-timezones"},
	{"declaredproducts", "https://www.rst.gov.ru/opendata/7706406291-declaredproducts"},
	{"productcommoncertification", "https://www.rst.gov.ru/opendata/7706406291-productcommoncertification"},
	{"listnationalstandarts", "https://www.rst.gov.ru/opendata/7706406291-listnationalstandarts"},
}

func main() {
	var (
		filePath  = flag.String("file", "", "Path to the GOST CSV file")
		dbPath    = flag.String("db", "./gosts.db", "Path to GOSTs database")
		sourceURL = flag.String("source-url", "", "Source URL for the GOST data")
		sourceType = flag.String("source-type", "", "Source type (nationalstandards, interstatestandards, etc.)")
		download   = flag.Bool("download", false, "Download CSV files from Rosstandart")
		allSources = flag.Bool("all", false, "Download and import from all available sources")
		verbose    = flag.Bool("verbose", false, "Verbose output")
	)
	flag.Parse()

	// Проверяем существование БД или создаем директорию
	dbDir := filepath.Dir(*dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Открываем базу данных
	gostsDB, err := database.NewGostsDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer gostsDB.Close()

	if *verbose {
		log.Printf("Using database: %s", *dbPath)
	}

	// Если нужно скачать файлы
	if *download || *allSources {
		if *allSources {
			// Скачиваем и импортируем из всех источников
			for _, source := range gostSources {
				if *verbose {
					log.Printf("Downloading from source: %s", source.name)
				}
				if err := downloadAndImport(gostsDB, source.url, source.name, *verbose); err != nil {
					log.Printf("Error importing from %s: %v", source.name, err)
					continue
				}
			}
		} else {
			// Скачиваем из указанного источника
			if *sourceURL == "" || *sourceType == "" {
				log.Fatal("source-url and source-type are required when using -download")
			}
			if err := downloadAndImport(gostsDB, *sourceURL, *sourceType, *verbose); err != nil {
				log.Fatalf("Failed to download and import: %v", err)
			}
		}
		return
	}

	// Импорт из локального файла
	if *filePath == "" {
		fmt.Println("Usage: import_gosts [options]")
		fmt.Println("\nOptions:")
		fmt.Println("  -file <path>          Path to CSV file with GOSTs")
		fmt.Println("  -db <path>            Path to GOSTs database (default: ./gosts.db)")
		fmt.Println("  -source-type <type>    Source type (nationalstandards, interstatestandards, etc.)")
		fmt.Println("  -source-url <url>     Source URL")
		fmt.Println("  -download             Download CSV from source URL")
		fmt.Println("  -all                  Download and import from all available sources")
		fmt.Println("  -verbose              Verbose output")
		fmt.Println("\nExamples:")
		fmt.Println("  import_gosts -file gosts.csv -source-type nationalstandards")
		fmt.Println("  import_gosts -download -source-url https://www.rst.gov.ru/opendata/7706406291-nationalstandards -source-type nationalstandards")
		fmt.Println("  import_gosts -all")
		os.Exit(1)
	}

	// Проверяем существование файла
	if _, err := os.Stat(*filePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Fatalf("File not found: %s", *filePath)
		}
		log.Fatalf("Error checking file %s: %v", *filePath, err)
	}

	// Определяем тип источника, если не указан
	if *sourceType == "" {
		*sourceType = "unknown"
	}
	if *sourceURL == "" {
		*sourceURL = ""
	}

	// Парсим CSV файл используя новый парсер
	if *verbose {
		log.Printf("Parsing CSV file: %s", *filePath)
	}
	
	// Открываем файл
	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()
	
	// Используем ParseGostCSVFromReader
	records, err := importer.ParseGostCSVFromReader(file)
	if err != nil {
		log.Fatalf("Failed to parse CSV file: %v", err)
	}

	if *verbose {
		log.Printf("Parsed %d records from CSV file", len(records))
	}

	if len(records) == 0 {
		log.Fatalf("No records found in CSV file")
	}

	// Создаем или обновляем источник данных
	source := &database.GostSource{
		SourceName:   *sourceType,
		SourceURL:    *sourceURL,
		LastSyncDate: timePtr(time.Now()),
		RecordsCount: len(records),
	}

	sourceRecord, err := gostsDB.CreateOrUpdateSource(source)
	if err != nil {
		log.Fatalf("Failed to create or update source: %v", err)
	}

	if *verbose {
		log.Printf("Using source ID: %d", sourceRecord.ID)
	}

	// Импортируем данные
	successCount := 0
	errorCount := 0
	errors := []string{}

	if *verbose {
		log.Printf("Starting import of %d GOST records...", len(records))
	}

	for i, record := range records {
		if *verbose && (i+1)%100 == 0 {
			log.Printf("Processed %d/%d records...", i+1, len(records))
		}

		gost := &database.Gost{
			GostNumber:    record.GostNumber,
			Title:         record.Title,
			AdoptionDate:  record.AdoptionDate,
			EffectiveDate: record.EffectiveDate,
			Status:        record.Status,
			SourceType:    *sourceType,
			SourceID:      &sourceRecord.ID,
			SourceURL:     *sourceURL,
			Description:   record.Description,
			Keywords:      record.Keywords,
		}

		_, err := gostsDB.CreateOrUpdateGost(gost)
		if err != nil {
			errorCount++
			errorMsg := fmt.Sprintf("ГОСТ %s: %v", record.GostNumber, err)
			errors = append(errors, errorMsg)
			if *verbose {
				log.Printf("Error importing GOST %s: %v", record.GostNumber, err)
			}
			continue
		}

		successCount++
	}

	// Выводим результаты
	fmt.Printf("\n=== Import Results ===\n")
	fmt.Printf("Total records: %d\n", len(records))
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Errors: %d\n", errorCount)
	fmt.Printf("Source ID: %d\n", sourceRecord.ID)

	if errorCount > 0 && *verbose {
		fmt.Printf("\n=== Errors (first 20) ===\n")
		maxErrors := 20
		if len(errors) < maxErrors {
			maxErrors = len(errors)
		}
		for i := 0; i < maxErrors; i++ {
			fmt.Printf(" - %s\n", errors[i])
		}
		if len(errors) > maxErrors {
			fmt.Printf("... and %d more errors\n", len(errors)-maxErrors)
		}
	}

	// Сохраняем результаты в JSON файл
	result := map[string]interface{}{
		"total":      len(records),
		"success":    successCount,
		"errors":     errorCount,
		"error_list": errors,
		"source_id":  sourceRecord.ID,
		"timestamp":  time.Now().Format(time.RFC3339),
	}

	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err == nil {
		reportPath := filepath.Join(filepath.Dir(*dbPath), "gost_import_report.json")
		if err := os.WriteFile(reportPath, resultJSON, 0644); err == nil {
			if *verbose {
				log.Printf("Import report saved to: %s", reportPath)
			}
		}
	}

	// Получаем статистику
	stats, err := gostsDB.GetStatistics()
	if err == nil {
		fmt.Printf("\n=== Database Statistics ===\n")
		if total, ok := stats["total_gosts"].(int); ok {
			fmt.Printf("Total GOSTs in database: %d\n", total)
		}
		if byStatus, ok := stats["by_status"].(map[string]int); ok {
			fmt.Printf("\nBy status:\n")
			for status, count := range byStatus {
				fmt.Printf("  %s: %d\n", status, count)
			}
		}
		if bySourceType, ok := stats["by_source_type"].(map[string]int); ok {
			fmt.Printf("\nBy source type:\n")
			for sourceType, count := range bySourceType {
				fmt.Printf("  %s: %d\n", sourceType, count)
			}
		}
	}

	if errorCount > 0 {
		fmt.Printf("\nWarning: Import completed with %d errors\n", errorCount)
		os.Exit(1)
	}

	fmt.Printf("\nImport completed successfully!\n")
}

// downloadAndImport скачивает CSV файл и импортирует его
func downloadAndImport(gostsDB *database.GostsDB, url, sourceType string, verbose bool) error {
	if verbose {
		log.Printf("Downloading CSV from: %s", url)
	}

	// Создаем HTTP клиент с таймаутом
	client := &http.Client{
		Timeout: 5 * time.Minute, // Таймаут 5 минут для больших файлов
	}

	// Скачиваем файл
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file: status code %d", resp.StatusCode)
	}

	// Проверяем Content-Type (может быть не CSV)
	contentType := resp.Header.Get("Content-Type")
	if verbose && contentType != "" {
		log.Printf("Content-Type: %s", contentType)
	}

	// Если сервер вернул HTML вместо CSV, пытаемся найти ссылку на CSV в HTML
	if strings.Contains(contentType, "text/html") {
		if verbose {
			log.Printf("Server returned HTML, searching for CSV download links...")
		}
		
		// Читаем HTML для парсинга
		htmlContent, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return fmt.Errorf("failed to read HTML content: %w", err)
		}
		
		// Парсим HTML и ищем ссылки на CSV
		csvURL, err := findCSVLinkInHTML(string(htmlContent), url)
		if err != nil {
			return fmt.Errorf("failed to find CSV link in HTML: %w", err)
		}
		
		if csvURL == "" {
			return fmt.Errorf("no CSV download link found in HTML page")
		}
		
		if verbose {
			log.Printf("Found CSV link: %s", csvURL)
		}
		
		// Скачиваем CSV файл по найденной ссылке
		resp, err = client.Get(csvURL)
		if err != nil {
			return fmt.Errorf("failed to download CSV from found link: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("failed to download CSV: status code %d", resp.StatusCode)
		}
		
		// Обновляем Content-Type
		contentType = resp.Header.Get("Content-Type")
		if verbose && contentType != "" {
			log.Printf("CSV Content-Type: %s", contentType)
		}
	}

	// Читаем данные напрямую из HTTP ответа в байты
	// Это позволяет правильно определить кодировку до парсинга
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}
	resp.Body.Close()

	// Проверяем размер данных
	if len(data) == 0 {
		return fmt.Errorf("downloaded file is empty")
	}
	if verbose {
		log.Printf("Downloaded file size: %d bytes", len(data))
	}

	// Используем парсер напрямую с данными в байтах
	// Парсер сам определит и исправит кодировку
	config := importer.DefaultParserConfig()
	logger := &importLogger{verbose: verbose}
	parser := importer.NewGostParser(config, logger)
	
	records, err := parser.ParseCSVData(data)
	if err != nil {
		// Если парсинг не удался, возможно это HTML или другой формат
		if strings.Contains(contentType, "text/html") {
			return fmt.Errorf("server returned HTML instead of CSV. This source may not provide CSV format or URL format is incorrect: %w", err)
		}
		return fmt.Errorf("failed to parse CSV: %w", err)
	}

	if verbose {
		log.Printf("Parsed %d records from %s", len(records), sourceType)
	}

	// Создаем или обновляем источник (до импорта, чтобы знать source_id)
	source := &database.GostSource{
		SourceName:   sourceType,
		SourceURL:    url,
		LastSyncDate: timePtr(time.Now()),
		RecordsCount: len(records),
	}

	sourceRecord, err := gostsDB.CreateOrUpdateSource(source)
	if err != nil {
		// Если не удалось создать источник, пробуем продолжить без него
		log.Printf("Warning: failed to create source: %v, continuing without source_id", err)
		sourceRecord = &database.GostSource{
			ID: 0, // Используем 0 если источник не создан
		}
	}

	// Импортируем данные
	successCount := 0
	errorCount := 0
	for _, record := range records {
		// Пропускаем записи без номера ГОСТа
		if record.GostNumber == "" {
			errorCount++
			if verbose {
				log.Printf("Skipping record without GOST number")
			}
			continue
		}
		
		// Если название пустое, используем номер ГОСТа
		title := record.Title
		if title == "" {
			title = record.GostNumber
		}
		
		// Создаем запись ГОСТа
		var sourceID *int
		if sourceRecord != nil && sourceRecord.ID > 0 {
			sourceID = &sourceRecord.ID
		}
		
		gost := &database.Gost{
			GostNumber:    record.GostNumber,
			Title:         title,
			AdoptionDate:  record.AdoptionDate,
			EffectiveDate: record.EffectiveDate,
			Status:        record.Status,
			SourceType:    sourceType,
			SourceID:      sourceID,
			SourceURL:     url,
			Description:   record.Description,
			Keywords:      record.Keywords,
		}

		_, err := gostsDB.CreateOrUpdateGost(gost)
		if err != nil {
			errorCount++
			if verbose {
				log.Printf("Error importing GOST %s: %v", record.GostNumber, err)
			}
			continue
		}

		successCount++
	}

	if verbose {
		log.Printf("Imported %d/%d GOSTs from %s (errors: %d)", successCount, len(records), sourceType, errorCount)
	} else {
		log.Printf("Imported %d GOSTs from %s", successCount, sourceType)
	}

	return nil
}

// findCSVLinkInHTML парсит HTML и ищет ссылки на CSV файлы
func findCSVLinkInHTML(htmlContent, baseURL string) (string, error) {
	// Парсим HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Парсим базовый URL для разрешения относительных ссылок
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}
	
	var csvURL string
	
	// Стратегия 1: Ищем ссылки с расширением .csv или параметрами формата
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		if csvURL != "" {
			return // Уже нашли
		}
		
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		
		hrefLower := strings.ToLower(href)
		// Проверяем, содержит ли ссылка .csv или указывает на CSV
		if strings.Contains(hrefLower, ".csv") || 
		   strings.Contains(hrefLower, "format=csv") ||
		   strings.Contains(hrefLower, "export=csv") ||
		   strings.Contains(hrefLower, "download") {
			
			// Разрешаем относительные ссылки
			parsedURL, err := url.Parse(href)
			if err != nil {
				return
			}
			
			resolvedURL := base.ResolveReference(parsedURL)
			csvURL = resolvedURL.String()
		}
	})
	
	// Стратегия 2: Ищем кнопки или элементы с data-атрибутами
	if csvURL == "" {
		doc.Find("[data-format='csv'], [data-export='csv'], [data-download='csv']").Each(func(i int, s *goquery.Selection) {
			if csvURL != "" {
				return
			}
			
			href, exists := s.Attr("href")
			if !exists {
				href, exists = s.Attr("data-url")
			}
			if !exists {
				href, exists = s.Attr("data-link")
			}
			
			if exists && href != "" {
				parsedURL, err := url.Parse(href)
				if err == nil {
					resolvedURL := base.ResolveReference(parsedURL)
					csvURL = resolvedURL.String()
				}
			}
		})
	}
	
	// Стратегия 3: Ищем ссылки с текстом "CSV", "Скачать", "Download"
	if csvURL == "" {
		doc.Find("a").Each(func(i int, s *goquery.Selection) {
			if csvURL != "" {
				return
			}
			
			text := strings.ToLower(strings.TrimSpace(s.Text()))
			if strings.Contains(text, "csv") || 
			   strings.Contains(text, "скачать") ||
			   strings.Contains(text, "download") ||
			   strings.Contains(text, "экспорт") {
				
				href, exists := s.Attr("href")
				if exists && href != "" {
					parsedURL, err := url.Parse(href)
					if err == nil {
						resolvedURL := base.ResolveReference(parsedURL)
						// Проверяем, что это действительно CSV
						resolvedStr := strings.ToLower(resolvedURL.String())
						if strings.Contains(resolvedStr, ".csv") || 
						   strings.Contains(resolvedStr, "format=csv") ||
						   strings.Contains(resolvedStr, "export=csv") {
							csvURL = resolvedURL.String()
						}
					}
				}
			}
		})
	}
	
	// Стратегия 4: Пробуем добавить .csv к базовому URL
	if csvURL == "" {
		if !strings.HasSuffix(baseURL, ".csv") {
			csvURL = baseURL + ".csv"
		}
	}
	
	return csvURL, nil
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// importLogger простой логгер для импорта
type importLogger struct {
	verbose bool
}

func (l *importLogger) Printf(format string, v ...interface{}) {
	if l.verbose {
		log.Printf(format, v...)
	}
}

