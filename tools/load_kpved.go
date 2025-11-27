package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"httpserver/database"
)

func main() {
	filePath := flag.String("file", "КПВЭД.txt", "Path to KPVED file")
	dbPath := flag.String("db", "service.db", "Path to service database")
	flag.Parse()

	if *filePath == "" {
		log.Fatal("File path is required")
	}

	// Проверяем существование файла
	if _, err := os.Stat(*filePath); err != nil {
		log.Fatalf("File not found: %s", *filePath)
	}

	// Открываем базу данных
	serviceDB, err := database.NewServiceDB(*dbPath)
	if err != nil {
		log.Fatalf("Failed to open service database: %v", err)
	}
	defer serviceDB.Close()

	log.Printf("Loading KPVED from file: %s", *filePath)

	// Загружаем данные
	if err := database.LoadKpvedFromFile(serviceDB, *filePath); err != nil {
		log.Fatalf("Failed to load KPVED: %v", err)
	}

	// Проверяем результат
	var count int
	err = serviceDB.GetDB().QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&count)
	if err != nil {
		log.Fatalf("Failed to count loaded codes: %v", err)
	}

	fmt.Printf("✓ Successfully loaded %d KPVED codes from %s\n", count, *filePath)
}

