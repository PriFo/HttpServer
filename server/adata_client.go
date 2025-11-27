package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AdataClient клиент для работы с Adata.kz API
type AdataClient struct {
	apiToken string
	baseURL  string
	httpClient *http.Client
}

// AdataCompanyInfo структура данных компании в ответе Adata.kz
// Структура может отличаться в зависимости от версии API
type AdataCompanyInfo struct {
	// Основная информация
	Name        string `json:"name"`
	ShortName   string `json:"short_name,omitempty"`
	BIN         string `json:"bin,omitempty"`
	IIN         string `json:"iin,omitempty"`
	
	// Адреса
	LegalAddress   string `json:"legal_address,omitempty"`
	ActualAddress  string `json:"actual_address,omitempty"`
	
	// Контакты
	Phone         string `json:"phone,omitempty"`
	Email         string `json:"email,omitempty"`
	
	// Дополнительная информация
	Status        string `json:"status,omitempty"`
	ActivityType  string `json:"activity_type,omitempty"`
	
	// Для гибкости - храним весь ответ
	RawData       map[string]interface{} `json:"-"`
}

// CompanyInfo результат поиска компании в Adata
type CompanyInfo struct {
	FullName      string
	ShortName     string
	BIN           string
	IIN           string
	LegalAddress  string
	ActualAddress string
	Phone         string
	Email         string
	Status        string
}

// NewAdataClient создает новый клиент Adata.kz
func NewAdataClient(apiToken, baseURL string) *AdataClient {
	if baseURL == "" {
		baseURL = "https://api.adata.kz"
	}

	// Оптимизированный HTTP Transport с connection pooling
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxConnsPerHost:     5,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		MaxIdleConnsPerHost: 5,
	}

	return &AdataClient{
		apiToken: apiToken,
		baseURL:  baseURL,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// FindCompany выполняет поиск компании по названию и/или БИН/ИИН
// Adata.kz API может иметь разные форматы, поэтому реализация гибкая
func (c *AdataClient) FindCompany(name string, bin string) (*CompanyInfo, error) {
	if c.apiToken == "" {
		return nil, fmt.Errorf("Adata API token is not set")
	}

	// Пробуем несколько вариантов API в зависимости от доступных данных
	var companyInfo *CompanyInfo
	var err error

	// Если есть БИН/ИИН, используем поиск по нему
	if bin != "" {
		companyInfo, err = c.findCompanyByBIN(bin)
		if err == nil && companyInfo != nil {
			return companyInfo, nil
		}
		log.Printf("[Adata] Failed to find company by BIN %s: %v, trying by name", bin, err)
	}

	// Если поиск по БИН не удался или БИН не указан, пробуем поиск по названию
	if name != "" {
		companyInfo, err = c.findCompanyByName(name)
		if err == nil && companyInfo != nil {
			return companyInfo, nil
		}
		log.Printf("[Adata] Failed to find company by name %s: %v", name, err)
	}

	return nil, fmt.Errorf("failed to find company: no results from Adata API")
}

// findCompanyByBIN ищет компанию по БИН/ИИН
func (c *AdataClient) findCompanyByBIN(bin string) (*CompanyInfo, error) {
	// Вариант 1: API с токеном в URL (как указано в документации)
	// GET /api/company/info/{TOKEN}?iinBin={BIN}
	apiURL := fmt.Sprintf("%s/api/company/info/%s", c.baseURL, c.apiToken)
	
	params := url.Values{}
	params.Add("iinBin", bin)
	
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Если этот вариант не работает, пробуем альтернативный
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusUnauthorized {
			return c.findCompanyByBINAlternative(bin)
		}
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Adata API returned status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseCompanyResponse(resp.Body)
}

// findCompanyByBINAlternative альтернативный вариант поиска по БИН
func (c *AdataClient) findCompanyByBINAlternative(bin string) (*CompanyInfo, error) {
	// Вариант 2: API с токеном в заголовке
	apiURL := fmt.Sprintf("%s/api/company/info", c.baseURL)
	
	params := url.Values{}
	params.Add("iinBin", bin)
	
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Adata API returned status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseCompanyResponse(resp.Body)
}

// findCompanyByName ищет компанию по названию
func (c *AdataClient) findCompanyByName(name string) (*CompanyInfo, error) {
	// Adata может иметь эндпоинт для поиска по названию
	// Пробуем несколько вариантов
	
	// Вариант 1: /api/company/search
	apiURL := fmt.Sprintf("%s/api/company/search", c.baseURL)
	
	params := url.Values{}
	params.Add("name", name)
	params.Add("limit", "1")
	
	fullURL := fmt.Sprintf("%s?%s", apiURL, params.Encode())
	
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		// Если эндпоинт не найден, возвращаем ошибку
		return nil, fmt.Errorf("Adata API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Парсим ответ (может быть массив или объект)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Пробуем распарсить как массив
	var companies []AdataCompanyInfo
	if err := json.Unmarshal(body, &companies); err == nil && len(companies) > 0 {
		return c.convertToCompanyInfo(&companies[0]), nil
	}

	// Пробуем распарсить как объект
	var company AdataCompanyInfo
	if err := json.Unmarshal(body, &company); err == nil {
		return c.convertToCompanyInfo(&company), nil
	}

	return nil, fmt.Errorf("failed to parse Adata response")
}

// parseCompanyResponse парсит ответ от Adata API
func (c *AdataClient) parseCompanyResponse(body io.Reader) (*CompanyInfo, error) {
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var company AdataCompanyInfo
	if err := json.Unmarshal(bodyBytes, &company); err != nil {
		log.Printf("[Adata] Failed to decode response: %v, body: %s", err, string(bodyBytes))
		
		// Пробуем распарсить как объект с данными внутри
		var wrapper struct {
			Data AdataCompanyInfo `json:"data"`
		}
		if err := json.Unmarshal(bodyBytes, &wrapper); err == nil {
			company = wrapper.Data
		} else {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return c.convertToCompanyInfo(&company), nil
}

// convertToCompanyInfo конвертирует AdataCompanyInfo в CompanyInfo
func (c *AdataClient) convertToCompanyInfo(adata *AdataCompanyInfo) *CompanyInfo {
	result := &CompanyInfo{
		FullName:      adata.Name,
		ShortName:     adata.ShortName,
		BIN:           adata.BIN,
		IIN:           adata.IIN,
		LegalAddress:  adata.LegalAddress,
		ActualAddress: adata.ActualAddress,
		Phone:         adata.Phone,
		Email:         adata.Email,
		Status:        adata.Status,
	}

	// Если полное название пустое, используем короткое
	if result.FullName == "" {
		result.FullName = result.ShortName
	}

	// Если БИН пустой, используем ИИН
	if result.BIN == "" {
		result.BIN = result.IIN
	}

	return result
}

// ExtractBINFromString извлекает БИН из строки (12 цифр)
func ExtractBINFromString(s string) string {
	// Убираем все нецифровые символы
	digits := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	
	// БИН должен быть 12 цифр
	if len(digits) == 12 {
		return digits
	}
	
	return ""
}

