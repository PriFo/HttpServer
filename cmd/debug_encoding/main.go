package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/transform"
)

func main() {
	// Скачиваем CSV файл напрямую (используем прямую ссылку на CSV)
	// Попробуем несколько вариантов URL
	urls := []string{
		"https://www.rst.gov.ru/opendata/7706406291-nationalstandards/data-20160517T000000-structure-20160517T000000.csv",
		"https://www.rst.gov.ru/opendata/7706406291-nationalstandards/data.csv",
		"https://www.rst.gov.ru/opendata/7706406291-nationalstandards",
	}
	
	var data []byte
	var err error
	
	for _, url := range urls {
		fmt.Printf("Trying URL: %s\n", url)
		client := &http.Client{}
		resp, err := client.Get(url)
		if err != nil {
			fmt.Printf("  Error: %v\n", err)
			continue
		}
		
		contentType := resp.Header.Get("Content-Type")
		fmt.Printf("  Content-Type: %s\n", contentType)
		
		if strings.Contains(contentType, "text/csv") || strings.Contains(contentType, "application/csv") {
			// Это CSV файл
			data, err = io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				fmt.Printf("  Error reading: %v\n", err)
				continue
			}
			fmt.Printf("  Successfully downloaded %d bytes\n", len(data))
			break
		} else {
			// Это HTML, читаем только первые байты для анализа
			data = make([]byte, 1000)
			n, _ := io.ReadFull(resp.Body, data)
			resp.Body.Close()
			data = data[:n]
			fmt.Printf("  Got HTML, read %d bytes\n", len(data))
		}
	}
	
	if len(data) == 0 {
		log.Fatalf("Failed to download any data")
	}

	fmt.Printf("Downloaded %d bytes\n\n", len(data))

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
		fmt.Printf("   Sample: %s\n", truncate(str, 100))
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
		fmt.Printf("   Sample: %s\n", truncate(str, 100))
		
		// Если все еще есть ╨У, пробуем декодировать еще раз
		if strings.Contains(str, "╨У") {
			decoded2, _, err2 := transform.Bytes(decoder, decoded)
			if err2 == nil && len(decoded2) > 0 && utf8.Valid(decoded2) {
				str2 := string(decoded2)
				fmt.Printf("\n3. As Windows-1251 (double decode):\n")
				fmt.Printf("   Valid UTF-8 after decode: yes\n")
				fmt.Printf("   Contains ╨У: %v\n", strings.Contains(str2, "╨У"))
				fmt.Printf("   Contains ГОСТ: %v\n", strings.Contains(str2, "ГОСТ"))
				fmt.Printf("   Sample: %s\n", truncate(str2, 100))
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

