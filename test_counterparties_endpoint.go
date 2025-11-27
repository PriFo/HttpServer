//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	baseURL := "http://localhost:3000"
	if len(os.Args) > 1 {
		baseURL = os.Args[1]
	}

	fmt.Println("Тестирование эндпоинта /api/counterparties/normalized")
	fmt.Println("==================================================")

	// Тест 1: Получение по client_id
	fmt.Println("\n1. Тест получения по client_id=1")
	testGetByClientID(baseURL, 1)

	// Тест 2: Получение по project_id
	fmt.Println("\n2. Тест получения по project_id=1")
	testGetByProjectID(baseURL, 1)

	// Тест 3: Получение с пагинацией
	fmt.Println("\n3. Тест получения с пагинацией (page=1, limit=10)")
	testGetWithPagination(baseURL, 1, 1, 10)

	// Тест 4: Получение с поиском
	fmt.Println("\n4. Тест получения с поиском (search=test)")
	testGetWithSearch(baseURL, 1, "test")

	// Тест 5: Получение без параметров (должна быть ошибка)
	fmt.Println("\n5. Тест получения без параметров (ожидается ошибка)")
	testGetWithoutParams(baseURL)

	// Тест 6: Неверный HTTP метод
	fmt.Println("\n6. Тест неверного HTTP метода (POST вместо GET)")
	testInvalidMethod(baseURL)
}

func testGetByClientID(baseURL string, clientID int) {
	u, _ := url.Parse(fmt.Sprintf("%s/api/counterparties/normalized", baseURL))
	q := u.Query()
	q.Set("client_id", fmt.Sprintf("%d", clientID))
	q.Set("page", "1")
	q.Set("limit", "20")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Printf("  ❌ Ошибка запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	printResponse("GET", u.String(), resp.StatusCode, body)
}

func testGetByProjectID(baseURL string, projectID int) {
	u, _ := url.Parse(fmt.Sprintf("%s/api/counterparties/normalized", baseURL))
	q := u.Query()
	q.Set("project_id", fmt.Sprintf("%d", projectID))
	q.Set("page", "1")
	q.Set("limit", "20")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Printf("  ❌ Ошибка запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	printResponse("GET", u.String(), resp.StatusCode, body)
}

func testGetWithPagination(baseURL string, clientID, page, limit int) {
	u, _ := url.Parse(fmt.Sprintf("%s/api/counterparties/normalized", baseURL))
	q := u.Query()
	q.Set("client_id", fmt.Sprintf("%d", clientID))
	q.Set("page", fmt.Sprintf("%d", page))
	q.Set("limit", fmt.Sprintf("%d", limit))
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Printf("  ❌ Ошибка запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	printResponse("GET", u.String(), resp.StatusCode, body)
}

func testGetWithSearch(baseURL string, clientID int, search string) {
	u, _ := url.Parse(fmt.Sprintf("%s/api/counterparties/normalized", baseURL))
	q := u.Query()
	q.Set("client_id", fmt.Sprintf("%d", clientID))
	q.Set("search", search)
	q.Set("page", "1")
	q.Set("limit", "20")
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Printf("  ❌ Ошибка запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	printResponse("GET", u.String(), resp.StatusCode, body)
}

func testGetWithoutParams(baseURL string) {
	u := fmt.Sprintf("%s/api/counterparties/normalized", baseURL)

	resp, err := http.Get(u)
	if err != nil {
		fmt.Printf("  ❌ Ошибка запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	printResponse("GET", u, resp.StatusCode, body)
}

func testInvalidMethod(baseURL string) {
	u := fmt.Sprintf("%s/api/counterparties/normalized?client_id=1", baseURL)

	req, _ := http.NewRequest("POST", u, nil)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("  ❌ Ошибка запроса: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	printResponse("POST", u, resp.StatusCode, body)
}

func printResponse(method, url string, statusCode int, body []byte) {
	statusIcon := "✅"
	if statusCode >= 400 {
		statusIcon = "❌"
	}
	fmt.Printf("  %s %s %s -> %d\n", statusIcon, method, url, statusCode)

	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		prettyJSON, _ := json.MarshalIndent(jsonData, "    ", "  ")
		if len(prettyJSON) > 500 {
			fmt.Printf("    %s...\n", string(prettyJSON[:500]))
		} else {
			fmt.Printf("    %s\n", string(prettyJSON))
		}
	} else {
		if len(body) > 200 {
			fmt.Printf("    %s...\n", string(body[:200]))
		} else {
			fmt.Printf("    %s\n", string(body))
		}
	}
}

