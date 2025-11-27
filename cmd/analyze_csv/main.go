package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func main() {
	// Ищем CSV файл
	csvFiles, err := filepath.Glob("data/temp/gost_*.csv")
	if err != nil || len(csvFiles) == 0 {
		log.Fatalf("No CSV files found in data/temp/")
	}
	
	filePath := csvFiles[len(csvFiles)-1] // Берем последний файл
	fmt.Printf("Analyzing CSV file: %s\n\n", filePath)
	
	// Читаем файл
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()
	
	// Читаем первые 2000 байт
	data := make([]byte, 2000)
	n, err := io.ReadFull(file, data)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		log.Fatalf("Failed to read: %v", err)
	}
	data = data[:n]
	
	fmt.Printf("Read %d bytes\n\n", len(data))
	
	// Показываем первые байты в hex
	fmt.Println("First 100 bytes (hex):")
	for i := 0; i < len(data) && i < 100; i++ {
		if i%16 == 0 {
			fmt.Printf("\n%04x: ", i)
		}
		fmt.Printf("%02x ", data[i])
	}
	fmt.Println()
	
	// Пробуем разные кодировки
	fmt.Println("Trying different encodings:")
	fmt.Println(strings.Repeat("=", 80))
	
	// 1. Как UTF-8
	if utf8.Valid(data) {
		str := string(data)
		fmt.Printf("\n1. As UTF-8:\n")
		fmt.Printf("   Valid UTF-8: yes\n")
		fmt.Printf("   Contains ╨У: %v\n", strings.Contains(str, "╨У"))
		fmt.Printf("   Contains ГОСТ: %v\n", strings.Contains(str, "ГОСТ"))
		fmt.Printf("   Sample (first 150 chars): %s\n", truncate(str, 150))
	}
	
	// 2. Как Windows-1251
	decoder := charmap.Windows1251.NewDecoder()
	decoded, _, err := transform.Bytes(decoder, data)
	if err == nil && len(decoded) > 0 && utf8.Valid(decoded) {
		str := string(decoded)
		fmt.Printf("\n2. As Windows-1251:\n")
		fmt.Printf("   Valid UTF-8 after decode: yes\n")
		fmt.Printf("   Contains ╨У: %v\n", strings.Contains(str, "╨У"))
		fmt.Printf("   Contains ГОСТ: %v\n", strings.Contains(str, "ГОСТ"))
		fmt.Printf("   Sample (first 150 chars): %s\n", truncate(str, 150))
		
		// Если все еще есть ╨У, пробуем декодировать еще раз
		if strings.Contains(str, "╨У") {
			fmt.Printf("\n   ⚠️  Still contains ╨У, trying double decode...\n")
			decoded2, _, err2 := transform.Bytes(decoder, decoded)
			if err2 == nil && len(decoded2) > 0 && utf8.Valid(decoded2) {
				str2 := string(decoded2)
				fmt.Printf("\n3. As Windows-1251 (double decode):\n")
				fmt.Printf("   Valid UTF-8 after decode: yes\n")
				fmt.Printf("   Contains ╨У: %v\n", strings.Contains(str2, "╨У"))
				fmt.Printf("   Contains ГОСТ: %v\n", strings.Contains(str2, "ГОСТ"))
				fmt.Printf("   Sample (first 150 chars): %s\n", truncate(str2, 150))
			}
		}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

