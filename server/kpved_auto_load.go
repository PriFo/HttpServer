package server

import (
	"log"
	"os"
	"path/filepath"

	"httpserver/database"
)

// ensureKpvedLoaded проверяет наличие данных КПВЭД в базе и загружает их при необходимости
func (s *Server) ensureKpvedLoaded() {
	if s.serviceDB == nil {
		log.Printf("[KPVED] Service database not available, skipping KPVED auto-load")
		return
	}

	// Проверяем наличие данных в таблице
	var count int
	err := s.serviceDB.GetDB().QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&count)
	if err != nil {
		// Таблица может не существовать, это нормально
		log.Printf("[KPVED] Error checking KPVED table: %v", err)
		count = 0
	}

	// Если данные уже есть, пропускаем загрузку
	if count > 0 {
		log.Printf("[KPVED] KPVED classifier already loaded: %d codes", count)
		return
	}

	log.Printf("[KPVED] KPVED classifier not found in database, attempting auto-load...")

	// Ищем файл КПВЭД.txt в корне проекта
	possiblePaths := []string{
		"КПВЭД.txt",
		"kpved.txt",
		"KPVED.txt",
		filepath.Join(".", "КПВЭД.txt"),
		filepath.Join("..", "КПВЭД.txt"),
	}

	var kpvedFilePath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			kpvedFilePath = path
			break
		}
	}

	if kpvedFilePath == "" {
		log.Printf("[KPVED] KPVED file not found in standard locations, skipping auto-load")
		log.Printf("[KPVED] Please load KPVED manually via POST /api/kpved/load or /api/kpved/load-from-file")
		return
	}

	log.Printf("[KPVED] Found KPVED file: %s, loading into database...", kpvedFilePath)

	// Загружаем данные
	if err := database.LoadKpvedFromFile(s.serviceDB, kpvedFilePath); err != nil {
		log.Printf("[KPVED] Failed to auto-load KPVED: %v", err)
		log.Printf("[KPVED] Please load KPVED manually via POST /api/kpved/load or /api/kpved/load-from-file")
		return
	}

	// Проверяем результат
	var loadedCount int
	err = s.serviceDB.GetDB().QueryRow("SELECT COUNT(*) FROM kpved_classifier").Scan(&loadedCount)
	if err != nil {
		log.Printf("[KPVED] Error counting loaded codes: %v", err)
	} else {
		log.Printf("[KPVED] ✓ Successfully auto-loaded %d KPVED codes from %s", loadedCount, kpvedFilePath)
	}
}

