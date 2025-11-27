package main

import (
	"fmt"
	"log"
	"strings"

	"httpserver/database"
)

func main() {
	// Инициализируем базу данных
	gostsDB, err := database.NewGostsDB("./gosts.db")
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer gostsDB.Close()

	// Получаем несколько записей для проверки
	gosts, total, err := gostsDB.ListGosts(10, 0, "", "", "", "", "", "")
	if err != nil {
		log.Fatalf("Failed to list gosts: %v", err)
	}

	fmt.Printf("Total GOSTs in database: %d\n\n", total)
	fmt.Printf("First 10 GOSTs:\n")
	fmt.Println(strings.Repeat("=", 80))

	for i, gost := range gosts {
		fmt.Printf("\n%d. %s\n", i+1, gost.GostNumber)
		fmt.Printf("   Title: %s\n", gost.Title)
		if gost.AdoptionDate != nil {
			fmt.Printf("   Adoption Date: %s\n", gost.AdoptionDate.Format("2006-01-02"))
		}
		if gost.Status != "" {
			fmt.Printf("   Status: %s\n", gost.Status)
		}
		fmt.Printf("   Source: %s\n", gost.SourceType)
	}

	// Проверяем конкретный ГОСТ
	if len(gosts) > 0 {
		testNumber := gosts[0].GostNumber
		gost, err := gostsDB.GetGostByNumber(testNumber)
		if err != nil {
			log.Printf("Failed to get GOST by number: %v", err)
		} else {
			fmt.Printf("\n\nDetailed info for GOST %s:\n", testNumber)
			fmt.Println(strings.Repeat("=", 80))
			fmt.Printf("ID: %d\n", gost.ID)
			fmt.Printf("Number: %s\n", gost.GostNumber)
			fmt.Printf("Title: %s\n", gost.Title)
			if gost.Description != "" {
				fmt.Printf("Description: %s\n", gost.Description)
			}
		}
	}
}
