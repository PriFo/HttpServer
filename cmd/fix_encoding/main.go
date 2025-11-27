package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Открываем базу данных
	db, err := sql.Open("sqlite3", "./gosts.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Получаем ВСЕ записи и проверяем их при чтении
	// SQL LIKE может не работать с этими символами правильно
	rows, err := db.Query(`
		SELECT id, gost_number, title, status, description, keywords 
		FROM gosts 
		LIMIT 10000
	`)
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	var fixedCount int
	var totalCount int

	for rows.Next() {
		totalCount++
		var id int
		var gostNumber, title, status, description, keywords sql.NullString

		if err := rows.Scan(&id, &gostNumber, &title, &status, &description, &keywords); err != nil {
			log.Printf("Failed to scan: %v", err)
			continue
		}

		// Проверяем, нужно ли исправлять эту запись
		needsFix := false
		if gostNumber.Valid && (strings.Contains(gostNumber.String, "╨У") || strings.Contains(gostNumber.String, "╨Ю")) {
			needsFix = true
		}
		if title.Valid && (strings.Contains(title.String, "╨У") || strings.Contains(title.String, "╨Ю")) {
			needsFix = true
		}
		if status.Valid && (strings.Contains(status.String, "╨У") || strings.Contains(status.String, "╨Ю")) {
			needsFix = true
		}

		if !needsFix {
			continue // Пропускаем записи без проблем
		}

		// Исправляем каждое поле
		fields := map[string]string{
			"gost_number": fixEncoding(gostNumber.String),
			"title":       fixEncoding(title.String),
			"status":      fixEncoding(status.String),
			"description": fixEncoding(description.String),
			"keywords":    fixEncoding(keywords.String),
		}

		// Обновляем запись
		query := `
			UPDATE gosts 
			SET gost_number = ?, title = ?, status = ?, description = ?, keywords = ?
			WHERE id = ?
		`
		_, err := db.Exec(query,
			fields["gost_number"],
			fields["title"],
			fields["status"],
			fields["description"],
			fields["keywords"],
			id,
		)
		if err != nil {
			log.Printf("Failed to update record %d: %v", id, err)
			continue
		}

		fixedCount++
		if fixedCount%100 == 0 {
			fmt.Printf("Fixed %d records...\n", fixedCount)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Row error: %v", err)
	}

	fmt.Printf("\nFixed %d out of %d records with encoding issues\n", fixedCount, totalCount)
}

func fixEncoding(text string) string {
	if text == "" {
		return text
	}

	// Проверяем, есть ли некорректные символы
	if !strings.Contains(text, "╨У") && !strings.Contains(text, "╨Ю") {
		return text
	}

	// Проблема: текст уже в UTF-8, но содержит символы "╨У" (0xD0 0xA3 в UTF-8)
	// Эти байты должны быть интерпретированы как Windows-1251 символ "Г" (0xC3)
	// Решение: берем UTF-8 байты и декодируем их как Windows-1251
	decoder := charmap.Windows1251.NewDecoder()
	
	// Конвертируем UTF-8 строку в байты
	utf8Bytes := []byte(text)
	
	// Декодируем эти байты как Windows-1251
	decoded, _, err := transform.Bytes(decoder, utf8Bytes)
	if err != nil {
		return text // Если не удалось, возвращаем как есть
	}

	// Проверяем результат
	if utf8.Valid(decoded) {
		result := string(decoded)
		// Проверяем, что после декодирования нет некорректных символов и есть правильные
		if !strings.Contains(result, "╨У") && !strings.Contains(result, "╨Ю") {
			// Проверяем, что есть правильные кириллические символы
			if strings.Contains(result, "ГОСТ") || strings.Contains(result, "Стандарт") {
				return result
			}
		}
	}

	return text
}

