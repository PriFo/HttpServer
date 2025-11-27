package handlers

import (
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	apperrors "httpserver/server/errors"
	"httpserver/server/middleware"

	_ "github.com/mattn/go-sqlite3"
)

// FileValidator валидатор файлов
type FileValidator struct {
	AllowedExtensions []string
	MaxSize           int64
	MinSize           int64
}

// NewFileValidator создает новый валидатор файлов
func NewFileValidator(extensions []string, maxSize, minSize int64) *FileValidator {
	return &FileValidator{
		AllowedExtensions: extensions,
		MaxSize:           maxSize,
		MinSize:           minSize,
	}
}

// ValidateExtension проверяет расширение файла
func (fv *FileValidator) ValidateExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range fv.AllowedExtensions {
		if ext == strings.ToLower(allowed) {
			return nil
		}
	}
	return apperrors.NewValidationError(
		fmt.Sprintf("file extension %s is not allowed. Allowed extensions: %v", ext, fv.AllowedExtensions),
		nil,
	)
}

// ValidateSize проверяет размер файла
func (fv *FileValidator) ValidateSize(size int64) error {
	if fv.MinSize > 0 && size < fv.MinSize {
		return apperrors.NewValidationError(
			fmt.Sprintf("file size %d is less than minimum %d", size, fv.MinSize),
			nil,
		)
	}
	if fv.MaxSize > 0 && size > fv.MaxSize {
		return apperrors.NewValidationError(
			fmt.Sprintf("file size %d exceeds maximum %d", size, fv.MaxSize),
			nil,
		)
	}
	return nil
}

// SanitizeFilename очищает имя файла от опасных символов
func SanitizeFilename(filename string) string {
	// Удаляем путь, оставляем только имя файла
	filename = filepath.Base(filename)

	// Заменяем опасные символы
	dangerous := []string{"..", "/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	for _, char := range dangerous {
		filename = strings.ReplaceAll(filename, char, "_")
	}

	// Ограничиваем длину
	if len(filename) > 255 {
		filename = filename[:255]
	}

	return filename
}

// ValidatePathTraversal проверяет, что путь не содержит path traversal
func ValidatePathTraversal(path string) error {
	if strings.Contains(path, "..") || strings.Contains(path, "/") || strings.Contains(path, "\\") {
		return apperrors.NewValidationError(
			fmt.Sprintf("path traversal detected in path: %s", path),
			nil,
		)
	}
	return nil
}

// EnsureDirectory создает директорию, если она не существует
func EnsureDirectory(dirPath string) error {
	if _, err := os.Stat(dirPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return apperrors.NewInternalError(
					fmt.Sprintf("failed to create directory %s", dirPath),
					err,
				)
			}
			slog.Info("Created directory", "path", dirPath)
			return nil
		}
		return apperrors.NewInternalError(
			fmt.Sprintf("failed to check directory %s", dirPath),
			err,
		)
	}
	return nil
}

// ValidateSQLiteFile проверяет, что файл является валидным SQLite файлом
func ValidateSQLiteFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return apperrors.NewInternalError("failed to open file", err)
	}
	defer file.Close()

	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil && err.Error() != "EOF" {
		return apperrors.NewInternalError("failed to read file header", err)
	}

	if n < 16 {
		return apperrors.NewValidationError(
			"file too small to be a SQLite database (minimum 16 bytes)",
			nil,
		)
	}

	sqliteHeader := "SQLite format 3\x00"
	if string(header) != sqliteHeader {
		return apperrors.NewValidationError(
			fmt.Sprintf("file is not a valid SQLite database. Header: %q", string(header)),
			nil,
		)
	}

	// Проверяем, что файл не является исполняемым
	executableSignatures := [][]byte{
		{0x7F, 0x45, 0x4C, 0x46}, // ELF
		{0x4D, 0x5A},             // PE/COFF
		{0xCA, 0xFE, 0xBA, 0xBE}, // Java class
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O
		{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O little-endian
	}

	for _, sig := range executableSignatures {
		if len(header) >= len(sig) {
			match := true
			for i := 0; i < len(sig); i++ {
				if header[i] != sig[i] {
					match = false
					break
				}
			}
			if match {
				return apperrors.NewValidationError(
					"file appears to be an executable (signature detected), rejected for security",
					nil,
				)
			}
		}
	}

	return nil
}

// ValidateSQLiteConnection проверяет подключение к SQLite базе данных
func ValidateSQLiteConnection(dbPath string) error {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return apperrors.NewInternalError("failed to open database", err)
	}
	defer conn.Close()

	if err := conn.Ping(); err != nil {
		return apperrors.NewInternalError("failed to ping database", err)
	}

	return nil
}

// IsValidTableName проверяет, что имя таблицы валидно (защита от SQL-инъекций)
func IsValidTableName(tableName string) bool {
	if tableName == "" {
		return false
	}

	// Проверяем на опасные символы
	dangerous := []string{";", "'", "\"", "--", "/*", "*/", " ", "-", ".."}
	for _, char := range dangerous {
		if strings.Contains(tableName, char) {
			return false
		}
	}

	// Проверяем, что имя содержит только буквы, цифры и подчеркивания
	for _, r := range tableName {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_') {
			return false
		}
	}

	return true
}

// BuildSafeTableQuery строит безопасный SQL запрос с именем таблицы
func BuildSafeTableQuery(queryTemplate, tableName string) string {
	if !IsValidTableName(tableName) {
		slog.Warn("SECURITY WARNING: Attempted to build query with invalid table name",
			"table_name", tableName,
		)
		return ""
	}
	return fmt.Sprintf(queryTemplate, tableName)
}

// WriteXMLResponse записывает XML ответ
func WriteXMLResponse(w http.ResponseWriter, data interface{}) error {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	xmlData, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return apperrors.NewInternalError("failed to marshal XML", err)
	}

	w.Write([]byte(xml.Header))
	_, err = w.Write(xmlData)
	if err != nil {
		return apperrors.NewInternalError("failed to write XML response", err)
	}
	return nil
}

// WriteXMLError записывает ошибку в XML формате и логирует её
func WriteXMLError(w http.ResponseWriter, r *http.Request, message string, err error) {
	// Логируем ошибку
	if r != nil {
		reqID := middleware.GetRequestID(r.Context())
		slog.Error("XML HTTP error",
			"error", message,
			"underlying_error", err,
			"status_code", http.StatusInternalServerError,
			"request_id", reqID,
			"method", r.Method,
			"path", r.URL.Path,
		)
	} else {
		slog.Error("XML HTTP error",
			"error", message,
			"underlying_error", err,
			"status_code", http.StatusInternalServerError,
		)
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)

	type ErrorResponse struct {
		XMLName   xml.Name `xml:"error_response"`
		Success   bool     `xml:"success"`
		Error     string   `xml:"error"`
		Message   string   `xml:"message"`
		Timestamp string   `xml:"timestamp"`
	}

	response := ErrorResponse{
		Success:   false,
		Error:     "",
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err != nil {
		response.Error = err.Error()
	}

	xmlData, _ := xml.MarshalIndent(response, "", "  ")
	w.Write([]byte(xml.Header))
	w.Write(xmlData)
}
