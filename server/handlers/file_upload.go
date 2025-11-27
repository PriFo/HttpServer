package handlers

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"httpserver/server/errors"
)

// FileUploadHandler обрабатывает загрузку файлов базы данных
type FileUploadHandler struct {
	*BaseHandler
	validateSQLite    func(string) error
	ensureUploadsDir  func(string) (string, error)
	parseDBName       func(string) string
}

// NewFileUploadHandler создает новый обработчик загрузки файлов
func NewFileUploadHandler(
	base *BaseHandler,
	validateSQLite func(string) error,
	ensureUploadsDir func(string) (string, error),
	parseDBName func(string) string,
) *FileUploadHandler {
	return &FileUploadHandler{
		BaseHandler:      base,
		validateSQLite:   validateSQLite,
		ensureUploadsDir: ensureUploadsDir,
		parseDBName:      parseDBName,
	}
}

// UploadRequest запрос на загрузку файла
type UploadRequest struct {
	File        multipart.File
	Header      *multipart.FileHeader
	Description string
	AutoCreate  bool
}

// UploadResult результат загрузки файла
type UploadResult struct {
	FilePath    string
	FileName    string
	FileSize    int64
	ContentType string
}

// FileHeader заголовок файла для валидации
type FileHeader struct {
	Filename string
	Size     int64
}

// ValidateUploadRequest валидирует запрос на загрузку
func (h *FileUploadHandler) ValidateUploadRequest(r *http.Request) (*UploadRequest, error) {
	// Проверяем Content-Type
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data") {
		return nil, NewValidationError(
			fmt.Sprintf("invalid Content-Type: expected multipart/form-data, got: %s", contentType),
			nil,
		)
	}

	// Парсим multipart форму (максимальный размер 500MB)
	if err := r.ParseMultipartForm(500 << 20); err != nil {
		return nil, NewValidationError("failed to parse multipart form", err)
	}

	// Получаем файл из формы
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, NewValidationError("failed to get file from form", err)
	}

	// Проверяем расширение файла
	fileName := header.Filename
	if !strings.HasSuffix(strings.ToLower(fileName), ".db") {
		return nil, errors.NewValidationError(
			fmt.Sprintf("file must have .db extension, got: %s", fileName),
			nil,
		)
	}

	description := r.FormValue("description")
	autoCreate := r.FormValue("auto_create") == "true"

	return &UploadRequest{
		File:        file,
		Header:      header,
		Description: description,
		AutoCreate:  autoCreate,
	}, nil
}

// ProcessFileUpload обрабатывает загрузку и валидацию файла
func (h *FileUploadHandler) ProcessFileUpload(req *UploadRequest) (*UploadResult, error) {
	// Получаем директорию для загрузки
	uploadsDir, err := h.ensureUploadsDir("uploads")
	if err != nil {
		return nil, NewInternalError("failed to ensure uploads directory", err)
	}

	// Создаем путь к файлу
	filePath := filepath.Join(uploadsDir, req.Header.Filename)

	// Сохраняем файл
	bytesWritten, err := SaveFile(req.File, filePath, req.Header.Size)
	if err != nil {
		return nil, NewInternalError("failed to save file", err)
	}

	// Валидируем SQLite файл
	if err := h.validateSQLite(filePath); err != nil {
		os.Remove(filePath)
		return nil, WrapError(err, "SQLite validation failed")
	}

	return &UploadResult{
		FilePath:    filePath,
		FileName:    req.Header.Filename,
		FileSize:    bytesWritten,
		ContentType: req.Header.Header.Get("Content-Type"),
	}, nil
}

// GetSuggestedDatabaseName получает предложенное имя базы данных из имени файла
func (h *FileUploadHandler) GetSuggestedDatabaseName(fileName string) string {
	if h.parseDBName != nil {
		name := h.parseDBName(fileName)
		if name != "" {
			return name
		}
	}
	// Если не удалось распарсить, используем имя файла без расширения
	return strings.TrimSuffix(fileName, filepath.Ext(fileName))
}

// GetDefaultDescription получает описание по умолчанию
func (h *FileUploadHandler) GetDefaultDescription() string {
	return fmt.Sprintf("Загружено: %s", time.Now().Format("02.01.2006 15:04"))
}

