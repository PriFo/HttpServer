package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Открываем базу данных
	db, err := sql.Open("sqlite3", "./gosts.db")
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Удаляем все записи из таблицы gosts
	fmt.Println("Deleting all GOST records...")
	result, err := db.Exec("DELETE FROM gosts")
	if err != nil {
		log.Fatalf("Failed to delete records: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("Failed to get rows affected: %v", err)
	}

	fmt.Printf("Deleted %d records from gosts table\n", rowsAffected)
	fmt.Println("\nNow you can reimport GOSTs using:")
	fmt.Println("  go run cmd/import_gosts/main.go -download -source-url <url> -source-type <type>")
}

