package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// DaDataClient клиент для работы с DaData API
type DaDataClient struct {
	apiKey        string
	secretKey     string
	baseURL       string
	httpClient    *http.Client
	circuitBreaker *HTTPCircuitBreaker // Circuit breaker для защиты от каскадных сбоев
}

// DaDataPartyName структура названия компании в ответе DaData
type DaDataPartyName struct {
	Full  string `json:"full_with_opf"`
	Short string `json:"short_with_opf"`
}

// DaDataAddress структура адреса в ответе DaData
type DaDataAddress struct {
	Value string `json:"value"`
}

// DaDataPartyData структура данных компании в ответе DaData
type DaDataPartyData struct {
	Name    DaDataPartyName `json:"name"`
	INN     string          `json:"inn"`
	KPP     string          `json:"kpp"`
	Address DaDataAddress   `json:"address"`
	OGRN   string          `json:"ogrn"`
	Status  string          `json:"state"`
}

// DaDataSuggestion структура одного предложения от DaData
type DaDataSuggestion struct {
	Value string          `json:"value"`
	Data  DaDataPartyData `json:"data"`
}

// DaDataSuggestionsResponse ответ от DaData API
type DaDataSuggestionsResponse struct {
	Suggestions []DaDataSuggestion `json:"suggestions"`
}

// PartySuggestion результат поиска компании
type PartySuggestion struct {
	FullName    string
	ShortName   string
	INN         string
	KPP         string
	Address     string
	OGRN        string
	Status      string
}

// NewDaDataClient создает новый клиент DaData
func NewDaDataClient(apiKey, secretKey, baseURL string) *DaDataClient {
	if baseURL == "" {
		baseURL = "https://suggestions.dadata.ru/suggestions/api/4_1/rs"
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

	return &DaDataClient{
		apiKey:         apiKey,
		secretKey:      secretKey,
		baseURL:        baseURL,
		circuitBreaker: NewHTTPCircuitBreaker(),
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		},
	}
}

// SuggestParty выполняет поиск компании по названию и/или ИНН
func (c *DaDataClient) SuggestParty(query string, inn string) (*PartySuggestion, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("DaData API key is not set")
	}

	// Проверяем Circuit Breaker перед запросом
	if !c.circuitBreaker.CanProceed() {
		return nil, fmt.Errorf("circuit breaker is open (state: %s), DaData API calls are temporarily blocked", c.circuitBreaker.GetState())
	}

	url := fmt.Sprintf("%s/suggest/party", c.baseURL)

	// Формируем тело запроса
	requestBody := map[string]interface{}{
		"query": query,
		"count": 1,
	}

	// Если указан ИНН, добавляем его в фильтр
	if inn != "" {
		requestBody["filters"] = []map[string]string{
			{"inn": inn},
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// DaData использует заголовки Authorization с Token и X-Secret
	req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.apiKey))
	if c.secretKey != "" {
		req.Header.Set("X-Secret", c.secretKey)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Проверяем статус ответа для circuit breaker
	if resp.StatusCode >= 500 {
		c.circuitBreaker.RecordFailure()
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		c.circuitBreaker.RecordSuccess()
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("DaData API returned status %d: %s", resp.StatusCode, string(body))
	}

	var response DaDataSuggestionsResponse
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("[DaData] Failed to decode response: %v, body: %s", err, string(body))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Suggestions) == 0 {
		return nil, fmt.Errorf("no suggestions found for query: %s", query)
	}

	// Берем первое предложение
	suggestion := response.Suggestions[0]
	data := suggestion.Data

	result := &PartySuggestion{
		FullName:  data.Name.Full,
		ShortName: data.Name.Short,
		INN:       data.INN,
		KPP:       data.KPP,
		Address:   data.Address.Value,
		OGRN:      data.OGRN,
		Status:   data.Status,
	}

	// Если полное название пустое, используем короткое
	if result.FullName == "" {
		result.FullName = result.ShortName
	}

	return result, nil
}

