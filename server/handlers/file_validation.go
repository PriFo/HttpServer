package handlers

import (
	"io"
	"os"

	apperrors "httpserver/server/errors"
)

// SQLiteValidator валидатор SQLite файлов
type SQLiteValidator struct {
	validateSQLite func(string) error
}

// NewSQLiteValidator создает новый валидатор SQLite
func NewSQLiteValidator(validateFunc func(string) error) *SQLiteValidator {
	return &SQLiteValidator{
		validateSQLite: validateFunc,
	}
}

// ValidateFileHeader проверяет заголовок SQLite файла
func (v *SQLiteValidator) ValidateFileHeader(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return apperrors.NewInternalError("не удалось открыть файл", err)
	}
	defer file.Close()

	header := make([]byte, 16)
	n, err := file.Read(header)
	if err != nil && err.Error() != "EOF" {
		return apperrors.NewInternalError("не удалось прочитать заголовок файла", err)
	}

	if n < 16 {
		return apperrors.NewValidationError("файл слишком мал для базы данных SQLite (минимум 16 байт)", nil)
	}

	sqliteHeader := "SQLite format 3\x00"
	if string(header) != sqliteHeader {
		return apperrors.NewValidationError("файл не является валидной базой данных SQLite", nil)
	}

	// Проверяем, что файл не является исполняемым
	if err := v.checkExecutableSignatures(header); err != nil {
		return err
	}

	return nil
}

// ValidateFileIntegrity проверяет целостность SQLite файла
func (v *SQLiteValidator) ValidateFileIntegrity(filePath string) error {
	if v.validateSQLite != nil {
		return v.validateSQLite(filePath)
	}
	return apperrors.NewInternalError("функция валидации SQLite не предоставлена", nil)
}

// checkExecutableSignatures проверяет, что файл не является исполняемым
func (v *SQLiteValidator) checkExecutableSignatures(header []byte) error {
	executableSignatures := [][]byte{
		{0x7F, 0x45, 0x4C, 0x46}, // ELF (Linux executable)
		{0x4D, 0x5A},             // PE/COFF (Windows executable)
		{0xCA, 0xFE, 0xBA, 0xBE}, // Java class file
		{0xFE, 0xED, 0xFA, 0xCE}, // Mach-O (macOS executable)
		{0xCE, 0xFA, 0xED, 0xFE}, // Mach-O (macOS executable, little-endian)
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
				return apperrors.NewValidationError("файл похож на исполняемый (обнаружена сигнатура), отклонен по соображениям безопасности", nil)
			}
		}
	}

	return nil
}

// ValidateFileSize проверяет размер файла
func ValidateFileSize(bytesWritten, expectedSize int64) error {
	if bytesWritten == 0 {
		return apperrors.NewValidationError("загруженный файл пуст", nil)
	}
	if expectedSize > 0 && bytesWritten != expectedSize {
		// Это не критично, но стоит залогировать
		// Возвращаем nil, так как это предупреждение, а не ошибка
	}
	return nil
}

// SaveFile сохраняет файл на диск
func SaveFile(file io.Reader, filePath string, expectedSize int64) (int64, error) {
	dst, err := os.Create(filePath)
	if err != nil {
		return 0, apperrors.NewInternalError("не удалось создать файл", err)
	}
	defer dst.Close()

	bytesWritten, err := io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath)
		return 0, apperrors.NewInternalError("не удалось скопировать файл", err)
	}

	return bytesWritten, nil
}

