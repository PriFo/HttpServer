package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Открываем базу данных
	db, err := sql.Open("sqlite3", "./gosts.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Проверяем кодировку базы данных
	var encoding string
	err = db.QueryRow("PRAGMA encoding").Scan(&encoding)
	if err != nil {
		log.Fatalf("Failed to check encoding: %v", err)
	}
	fmt.Printf("Database encoding: %s\n\n", encoding)

	// Получаем несколько записей
	rows, err := db.Query("SELECT gost_number, title FROM gosts LIMIT 5")
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	fmt.Println("Sample records from database:")
	fmt.Println(strings.Repeat("=", 80))

	for rows.Next() {
		var gostNumber, title string
		if err := rows.Scan(&gostNumber, &title); err != nil {
			log.Printf("Failed to scan: %v", err)
			continue
		}

		// Проверяем, является ли строка валидным UTF-8
		isValidUTF8 := utf8.ValidString(gostNumber) && utf8.ValidString(title)
		
		// Проверяем наличие некорректных символов
		hasInvalidChars := false
		invalidChars := []string{"╨У", "╨Ю", "╨б", "╨в"}
		for _, char := range invalidChars {
			if contains(gostNumber, char) || contains(title, char) {
				hasInvalidChars = true
				break
			}
		}

		fmt.Printf("\nGOST Number: %s\n", gostNumber)
		fmt.Printf("Title: %s\n", truncate(title, 60))
		fmt.Printf("Valid UTF-8: %v\n", isValidUTF8)
		fmt.Printf("Has invalid encoding chars: %v\n", hasInvalidChars)
		
		if hasInvalidChars {
			fmt.Printf("⚠️  WARNING: Contains invalid encoding characters!\n")
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatalf("Row error: %v", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		indexOfSubstring(s, substr) >= 0)
}

func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

