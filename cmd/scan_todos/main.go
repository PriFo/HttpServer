package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TodoTask представляет задачу TODO
type TodoTask struct {
	ID             string    `json:"id"`
	File           string    `json:"file"`
	Line           int       `json:"line"`
	Type           string    `json:"type"`     // TODO, FIXME, HACK, REFACTOR
	Priority       string    `json:"priority"` // CRITICAL, HIGH, MEDIUM, LOW
	Description    string    `json:"description"`
	Status         string    `json:"status"` // OPEN, IN_PROGRESS, RESOLVED, TESTING
	AssignedTo     string    `json:"assignedTo,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
	EstimatedHours int       `json:"estimatedHours"`
	ActualHours    *int      `json:"actualHours,omitempty"`
	Dependencies   []string  `json:"dependencies"`
	RelatedFiles   []string  `json:"relatedFiles"`
}

// TodoDB представляет базу данных задач
type TodoDB struct {
	Tasks    []TodoTask `json:"tasks"`
	LastScan *time.Time `json:"lastScan"`
	Version  string     `json:"version"`
}

// SmartTodoScanner сканирует код на наличие TODO
type SmartTodoScanner struct {
	patterns map[string]*regexp.Regexp
	dbPath   string
	db       *TodoDB
}

// NewSmartTodoScanner создает новый сканер
func NewSmartTodoScanner(dbPath string) *SmartTodoScanner {
	return &SmartTodoScanner{
		patterns: map[string]*regexp.Regexp{
			"CRITICAL": regexp.MustCompile(`(?i)(TODO\s*\(\s*CRITICAL\s*\)|FIXME|HACK|panic\(|not\s+implemented)`),
			"HIGH":     regexp.MustCompile(`(?i)(TODO\s*\(\s*HIGH\s*\)|implement|not\s+implemented)`),
			"MEDIUM":   regexp.MustCompile(`(?i)(TODO\s*\(\s*MEDIUM\s*\)|optimize|refactor)`),
			"LOW":      regexp.MustCompile(`(?i)(TODO\s*\(\s*LOW\s*\)|cleanup|document)`),
		},
		dbPath: dbPath,
	}
}

// LoadDB загружает базу данных задач
func (s *SmartTodoScanner) LoadDB() error {
	data, err := os.ReadFile(s.dbPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.db = &TodoDB{
				Tasks:   []TodoTask{},
				Version: "1.0.0",
			}
			return nil
		}
		return err
	}

	// Инициализируем db перед unmarshal
	s.db = &TodoDB{}
	if err := json.Unmarshal(data, s.db); err != nil {
		return err
	}

	return nil
}

// SaveDB сохраняет базу данных задач
func (s *SmartTodoScanner) SaveDB() error {
	now := time.Now()
	s.db.LastScan = &now

	data, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(s.dbPath, data, 0644)
}

// ScanFile сканирует файл на наличие TODO
func (s *SmartTodoScanner) ScanFile(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	// Определяем тип файла
	fileType := s.determineFileType(ext, filePath)
	if fileType == "" {
		return nil // Пропускаем файлы, которые не нужно сканировать
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")

	for lineNum, line := range lines {
		lineNum++ // Нумерация с 1

		// Определяем тип и приоритет
		todoType, priority := s.classifyLine(line)
		if todoType == "" {
			continue
		}

		// Извлекаем описание
		description := s.extractDescription(line)

		// Пропускаем задачи без описания (ложные срабатывания)
		if description == "" {
			continue
		}

		// Создаем ID задачи
		taskID := fmt.Sprintf("%s:%d", filePath, lineNum)

		// Проверяем, существует ли уже задача
		exists := false
		for i := range s.db.Tasks {
			if s.db.Tasks[i].ID == taskID {
				// Обновляем существующую задачу
				s.db.Tasks[i].UpdatedAt = time.Now()
				s.db.Tasks[i].Description = description
				s.db.Tasks[i].Priority = priority
				s.db.Tasks[i].Type = todoType
				exists = true
				break
			}
		}

		if !exists {
			// Создаем новую задачу
			task := TodoTask{
				ID:             taskID,
				File:           filePath,
				Line:           lineNum,
				Type:           todoType,
				Priority:       priority,
				Description:    description,
				Status:         "OPEN",
				AssignedTo:     s.autoAssign(fileType, priority),
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
				EstimatedHours: s.estimateHours(priority),
				Dependencies:   []string{},
				RelatedFiles:   []string{},
			}
			s.db.Tasks = append(s.db.Tasks, task)
		}
	}

	return nil
}

// determineFileType определяет тип файла
func (s *SmartTodoScanner) determineFileType(ext, filePath string) string {
	switch ext {
	case ".go":
		return "backend"
	case ".ts", ".tsx", ".js", ".jsx":
		return "frontend"
	case ".sh", ".ps1", ".bat":
		return "devops"
	case ".yml", ".yaml", ".dockerfile":
		return "devops"
	default:
		// Проверяем по пути
		if strings.Contains(filePath, "frontend") {
			return "frontend"
		}
		if strings.Contains(filePath, "server") || strings.Contains(filePath, "cmd") {
			return "backend"
		}
		return ""
	}
}

// classifyLine классифицирует строку и определяет приоритет
func (s *SmartTodoScanner) classifyLine(line string) (todoType, priority string) {
	lineUpper := strings.ToUpper(line)

	// Игнорируем строки, которые являются комментариями к структурам или типам
	// Например: "// Type string `json:"type"` // TODO, FIXME, HACK, REFACTOR"
	if s.isFalsePositive(line) {
		return "", ""
	}

	// Определяем тип - ищем только явные маркеры TODO/FIXME/HACK/REFACTOR
	// которые не являются частью комментариев к коду
	todoMatch := regexp.MustCompile(`(?i)\b(TODO|FIXME|HACK|REFACTOR)\s*\(?`)
	if !todoMatch.MatchString(line) {
		return "", ""
	}

	// Определяем тип
	if strings.Contains(lineUpper, "FIXME") {
		todoType = "FIXME"
	} else if strings.Contains(lineUpper, "HACK") {
		todoType = "HACK"
	} else if strings.Contains(lineUpper, "REFACTOR") {
		todoType = "REFACTOR"
	} else if strings.Contains(lineUpper, "TODO") {
		todoType = "TODO"
	} else {
		return "", ""
	}

	// Определяем приоритет
	if s.patterns["CRITICAL"].MatchString(line) {
		priority = "CRITICAL"
	} else if s.patterns["HIGH"].MatchString(line) {
		priority = "HIGH"
	} else if s.patterns["MEDIUM"].MatchString(line) {
		priority = "MEDIUM"
	} else {
		priority = "LOW"
	}

	return todoType, priority
}

// isFalsePositive проверяет, является ли строка ложным срабатыванием
func (s *SmartTodoScanner) isFalsePositive(line string) bool {
	// Игнорируем комментарии к структурам Go
	// Например: "Type string `json:"type"` // TODO, FIXME, HACK, REFACTOR"
	if regexp.MustCompile(`(?i)(type|struct|interface|func|var|const)\s+\w+\s+.*(TODO|FIXME|HACK)`).MatchString(line) {
		return true
	}

	// Игнорируем строки, где TODO/FIXME/HACK упоминаются только в комментариях к типам
	// Например: "// TODO, FIXME, HACK, REFACTOR" без описания задачи
	if regexp.MustCompile(`(?i)^\s*//\s*(TODO|FIXME|HACK|REFACTOR)\s*[,:]?\s*(TODO|FIXME|HACK|REFACTOR|panic|implement|optimize|refactor|cleanup|document)\s*$`).MatchString(line) {
		return true
	}

	// Игнорируем строки с регулярными выражениями, содержащими TODO/FIXME/HACK
	if regexp.MustCompile(`(?i)regexp|MustCompile|MatchString|ReplaceAllString`).MatchString(line) &&
		regexp.MustCompile(`(?i)(TODO|FIXME|HACK)`).MatchString(line) {
		return true
	}

	// Игнорируем строки, где TODO/FIXME/HACK упоминаются в контексте поиска/сканирования
	if regexp.MustCompile(`(?i)(scan|search|find|look|поиск|сканирование|скрипт|script)`).MatchString(line) &&
		regexp.MustCompile(`(?i)(TODO|FIXME|HACK)`).MatchString(line) {
		return true
	}

	// Игнорируем строки, где TODO/FIXME/HACK упоминаются в контексте обработки строк
	if regexp.MustCompile(`(?i)(contains|match|replace|extract|убираем|маркер)`).MatchString(line) &&
		regexp.MustCompile(`(?i)(TODO|FIXME|HACK)`).MatchString(line) {
		return true
	}

	return false
}

// extractDescription извлекает описание из строки
func (s *SmartTodoScanner) extractDescription(line string) string {
	// Убираем комментарии и лишние пробелы
	line = strings.TrimSpace(line)

	// Убираем маркеры TODO/FIXME/HACK с приоритетом
	line = regexp.MustCompile(`(?i)(TODO|FIXME|HACK|REFACTOR)\s*\([^)]*\)\s*:?\s*`).ReplaceAllString(line, "")
	// Убираем простые маркеры TODO/FIXME/HACK
	line = regexp.MustCompile(`(?i)(TODO|FIXME|HACK|REFACTOR)\s*:?\s*`).ReplaceAllString(line, "")
	// Убираем символы комментариев
	line = regexp.MustCompile(`(?i)(//|#|/\*|\*/|\*)`).ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	// Убираем технические маркеры, которые не являются описанием задачи
	line = regexp.MustCompile(`(?i)^\s*(type|struct|interface|func|var|const|json:|`).ReplaceAllString(line, "")
	line = strings.TrimSpace(line)

	// Если описание слишком короткое или содержит только технические термины, возвращаем пустую строку
	if len(line) < 5 {
		return ""
	}

	// Игнорируем описания, которые являются только списком типов
	if regexp.MustCompile(`(?i)^\s*(TODO|FIXME|HACK|REFACTOR|panic|implement|optimize|refactor|cleanup|document)\s*[,:]?\s*(TODO|FIXME|HACK|REFACTOR|panic|implement|optimize|refactor|cleanup|document)?\s*$`).MatchString(line) {
		return ""
	}

	// Ограничиваем длину
	if len(line) > 200 {
		line = line[:197] + "..."
	}

	return line
}

// autoAssign автоматически назначает задачу
func (s *SmartTodoScanner) autoAssign(fileType, priority string) string {
	switch fileType {
	case "backend":
		return "backend-team"
	case "frontend":
		return "frontend-team"
	case "devops":
		return "devops"
	default:
		return "unassigned"
	}
}

// estimateHours оценивает время выполнения
func (s *SmartTodoScanner) estimateHours(priority string) int {
	switch priority {
	case "CRITICAL":
		return 4
	case "HIGH":
		return 2
	case "MEDIUM":
		return 1
	case "LOW":
		return 0
	default:
		return 1
	}
}

// ScanDirectory рекурсивно сканирует директорию
func (s *SmartTodoScanner) ScanDirectory(rootDir string) error {
	extensions := map[string]bool{
		".go": true, ".ts": true, ".tsx": true, ".js": true, ".jsx": true,
		".sh": true, ".ps1": true, ".bat": true,
		".yml": true, ".yaml": true,
	}

	skipDirs := map[string]bool{
		".git": true, "node_modules": true, ".next": true, "dist": true,
		"build": true, "vendor": true, ".todos": true, "logs": true,
		"tmp": true, "checkpoints": true, "exports": true,
	}

	// Игнорируем файлы сканирования TODO
	skipFiles := map[string]bool{
		"scan_todos":           true,
		"generate_todo_report": true,
		"scan-todos":           true,
		"scan_todos_simple":    true,
	}

	return filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Пропускаем ошибки доступа
		}

		// Пропускаем директории
		if d.IsDir() {
			if skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		// Проверяем расширение
		ext := strings.ToLower(filepath.Ext(path))
		if !extensions[ext] {
			return nil
		}

		// Пропускаем файлы сканирования TODO
		baseName := filepath.Base(path)
		dirName := filepath.Dir(path)
		shouldSkip := false
		for skipFile := range skipFiles {
			if strings.Contains(baseName, skipFile) || strings.Contains(dirName, skipFile) {
				shouldSkip = true
				break
			}
		}
		if shouldSkip {
			return nil
		}

		// Сканируем файл
		if err := s.ScanFile(path); err != nil {
			log.Printf("Ошибка сканирования %s: %v", path, err)
		}

		return nil
	})
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: scan_todos <директория>")
		os.Exit(1)
	}

	rootDir := os.Args[1]
	if rootDir == "" {
		rootDir = "."
	}

	dbPath := ".todos/tasks.json"
	scanner := NewSmartTodoScanner(dbPath)

	// Загружаем БД
	if err := scanner.LoadDB(); err != nil {
		log.Fatalf("Ошибка загрузки БД: %v", err)
	}

	fmt.Println("🔄 Начинаю сканирование TODO...")
	fmt.Printf("📁 Директория: %s\n", rootDir)

	// Сканируем
	if err := scanner.ScanDirectory(rootDir); err != nil {
		log.Fatalf("Ошибка сканирования: %v", err)
	}

	// Сохраняем БД
	if err := scanner.SaveDB(); err != nil {
		log.Fatalf("Ошибка сохранения БД: %v", err)
	}

	// Выводим статистику
	total := len(scanner.db.Tasks)
	open := 0
	critical := 0

	for _, task := range scanner.db.Tasks {
		if task.Status == "OPEN" {
			open++
		}
		if task.Priority == "CRITICAL" {
			critical++
		}
	}

	fmt.Println("\n✅ Сканирование завершено!")
	fmt.Printf("📊 Статистика:\n")
	fmt.Printf("   Всего задач: %d\n", total)
	fmt.Printf("   Открытых: %d\n", open)
	fmt.Printf("   Критических: %d\n", critical)
	fmt.Printf("   Завершенных: %d\n", total-open)

	if scanner.db.LastScan != nil {
		fmt.Printf("\n📅 Последнее сканирование: %s\n", scanner.db.LastScan.Format("2006-01-02 15:04:05"))
	}
}
