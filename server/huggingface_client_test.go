package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"httpserver/nomenclature"
)

// TestNewHuggingFaceClient проверяет создание нового клиента Hugging Face
func TestNewHuggingFaceClient(t *testing.T) {
	apiKey := "test-api-key"
	baseURL := "https://api-inference.huggingface.co"
	client := NewHuggingFaceClient(apiKey, baseURL)

	if client == nil {
		t.Fatal("NewHuggingFaceClient() returned nil")
	}

	if client.apiKey != apiKey {
		t.Errorf("HuggingFaceClient.apiKey = %v, want %v", client.apiKey, apiKey)
	}

	if client.baseURL != baseURL {
		t.Errorf("HuggingFaceClient.baseURL = %v, want %v", client.baseURL, baseURL)
	}

	if client.httpClient == nil {
		t.Error("HuggingFaceClient.httpClient is nil")
	}

	if client.httpClient.Timeout != 60*time.Second {
		t.Errorf("HTTP client timeout = %v, want 60s", client.httpClient.Timeout)
	}
}

// TestNewHuggingFaceClient_DefaultURL проверяет использование дефолтного URL
func TestNewHuggingFaceClient_DefaultURL(t *testing.T) {
	client := NewHuggingFaceClient("test-key", "")
	
	expectedURL := "https://api-inference.huggingface.co"
	if client.baseURL != expectedURL {
		t.Errorf("HuggingFaceClient.baseURL = %v, want %v", client.baseURL, expectedURL)
	}
}

// TestChatCompletion_Success проверяет успешный сценарий отправки запроса
func TestChatCompletion_Success(t *testing.T) {
	// Мок сервер для тестирования
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Проверяем метод и путь
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		expectedPath := "/models/test-model"
		if !strings.HasSuffix(r.URL.Path, expectedPath) {
			t.Errorf("Expected path ending with %s, got %s", expectedPath, r.URL.Path)
		}

		// Проверяем заголовки
		authHeader := r.Header.Get("Authorization")
		expectedAuth := "Bearer test-api-key"
		if authHeader != expectedAuth {
			t.Errorf("Expected Authorization header %s, got %s", expectedAuth, authHeader)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", contentType)
		}

		// Проверяем тело запроса
		var request HuggingFaceRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("Failed to decode request: %v", err)
		}

		if request.Inputs == "" {
			t.Error("Expected non-empty inputs in request")
		}

		// Проверяем, что inputs содержит контент из messages
		if !strings.Contains(request.Inputs, "System: Test system prompt") {
			t.Error("Expected inputs to contain system prompt")
		}

		if !strings.Contains(request.Inputs, "User: Test user message") {
			t.Error("Expected inputs to contain user message")
		}

		// Возвращаем успешный ответ
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := HuggingFaceResponse{
			{GeneratedText: "This is a test response from Hugging Face"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Создаем клиент с мок-сервером
	client := &HuggingFaceClient{
		baseURL:    server.URL,
		apiKey:     "test-api-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	// Создаем тестовые сообщения
	messages := []nomenclature.Message{
		{Role: "system", Content: "Test system prompt"},
		{Role: "user", Content: "Test user message"},
	}

	// Вызываем ChatCompletion
	result, err := client.ChatCompletion("test-model", messages)

	if err != nil {
		t.Fatalf("ChatCompletion() returned error: %v", err)
	}

	expectedResult := "This is a test response from Hugging Face"
	if result != expectedResult {
		t.Errorf("ChatCompletion() = %v, want %v", result, expectedResult)
	}
}

// TestChatCompletion_SingleObjectResponse проверяет обработку ответа в формате одного объекта
func TestChatCompletion_SingleObjectResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Возвращаем ответ в формате одного объекта (не массива)
		w.Write([]byte(`{"generated_text": "Single object response"}`))
	}))
	defer server.Close()

	client := &HuggingFaceClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	messages := []nomenclature.Message{
		{Role: "user", Content: "Test"},
	}

	result, err := client.ChatCompletion("test-model", messages)

	if err != nil {
		t.Fatalf("ChatCompletion() returned error: %v", err)
	}

	if result != "Single object response" {
		t.Errorf("ChatCompletion() = %v, want 'Single object response'", result)
	}
}

// TestChatCompletion_API_Error проверяет обработку ошибок API
func TestChatCompletion_API_Error(t *testing.T) {
	testCases := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "401 Unauthorized",
			statusCode:     http.StatusUnauthorized,
			responseBody:   `{"error": "Invalid API key"}`,
			expectedErrMsg: "401",
		},
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error": "Model not found"}`,
			expectedErrMsg: "404",
		},
		{
			name:           "500 Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error": "Internal server error"}`,
			expectedErrMsg: "500",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			client := &HuggingFaceClient{
				baseURL:    server.URL,
				apiKey:     "test-key",
				httpClient: &http.Client{Timeout: 5 * time.Second},
			}

			messages := []nomenclature.Message{
				{Role: "user", Content: "Test"},
			}

			_, err := client.ChatCompletion("test-model", messages)

			if err == nil {
				t.Fatal("ChatCompletion() should return error for API error")
			}

			if !strings.Contains(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Error message should contain %s, got: %v", tc.expectedErrMsg, err)
			}
		})
	}
}

// TestChatCompletion_InvalidJSON проверяет обработку невалидного JSON
func TestChatCompletion_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Возвращаем невалидный JSON
		w.Write([]byte(`{"generated_text":}`))
	}))
	defer server.Close()

	client := &HuggingFaceClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	messages := []nomenclature.Message{
		{Role: "user", Content: "Test"},
	}

	_, err := client.ChatCompletion("test-model", messages)

	if err == nil {
		t.Fatal("ChatCompletion() should return error for invalid JSON")
	}

	if !strings.Contains(err.Error(), "decode") && !strings.Contains(err.Error(), "json") {
		t.Errorf("Error should mention JSON decoding, got: %v", err)
	}
}

// TestChatCompletion_EmptyResponse проверяет обработку пустого ответа
func TestChatCompletion_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Возвращаем пустой массив
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := &HuggingFaceClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	messages := []nomenclature.Message{
		{Role: "user", Content: "Test"},
	}

	_, err := client.ChatCompletion("test-model", messages)

	if err == nil {
		t.Fatal("ChatCompletion() should return error for empty response")
	}

	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("Error should mention empty response, got: %v", err)
	}
}

// TestMessagesToPrompt проверяет преобразование messages в промпт
func TestMessagesToPrompt(t *testing.T) {
	client := &HuggingFaceClient{}

	messages := []nomenclature.Message{
		{Role: "system", Content: "You are a helpful assistant"},
		{Role: "user", Content: "What is 2+2?"},
		{Role: "assistant", Content: "2+2 equals 4"},
		{Role: "user", Content: "Thank you"},
	}

	prompt := client.messagesToPrompt(messages)

	// Проверяем, что все сообщения присутствуют
	if !strings.Contains(prompt, "You are a helpful assistant") {
		t.Error("Prompt should contain system message")
	}

	if !strings.Contains(prompt, "What is 2+2?") {
		t.Error("Prompt should contain first user message")
	}

	if !strings.Contains(prompt, "2+2 equals 4") {
		t.Error("Prompt should contain assistant message")
	}

	if !strings.Contains(prompt, "Thank you") {
		t.Error("Prompt should contain second user message")
	}

	// Проверяем формат (должны быть префиксы System:, User:, Assistant:)
	if !strings.Contains(prompt, "System:") {
		t.Error("Prompt should contain 'System:' prefix")
	}

	if !strings.Contains(prompt, "User:") {
		t.Error("Prompt should contain 'User:' prefix")
	}

	if !strings.Contains(prompt, "Assistant:") {
		t.Error("Prompt should contain 'Assistant:' prefix")
	}
}

// TestGetModels проверяет, что метод возвращает курированный список моделей
func TestGetModels(t *testing.T) {
	client := NewHuggingFaceClient("test-key", "")

	ctx := context.Background()
	models, err := client.GetModels(ctx, "test-request-id")

	if err != nil {
		t.Fatalf("GetModels() returned error: %v", err)
	}

	// Проверяем количество моделей
	expectedCount := 7
	if len(models) != expectedCount {
		t.Errorf("GetModels() returned %d models, want %d", len(models), expectedCount)
	}

	// Проверяем структуру моделей
	for i, model := range models {
		if model.ID == "" {
			t.Errorf("Model %d ID should not be empty", i)
		}

		if model.Name == "" {
			t.Errorf("Model %d Name should not be empty", i)
		}

		if model.Status != "active" {
			t.Errorf("Model %d Status should be 'active', got %s", i, model.Status)
		}

		if model.MaxTokens <= 0 {
			t.Errorf("Model %d MaxTokens should be positive, got %d", i, model.MaxTokens)
		}
	}

	// Проверяем наличие конкретных моделей
	expectedModels := []string{
		"microsoft/DialoGPT-medium",
		"google/flan-t5-base",
		"meta-llama/Llama-2-7b-chat-hf",
		"mistralai/Mistral-7B-Instruct-v0.1",
		"google/gemma-7b-it",
		"HuggingFaceH4/zephyr-7b-beta",
		"tiiuae/falcon-7b-instruct",
	}

	modelIDs := make(map[string]bool)
	for _, model := range models {
		modelIDs[model.ID] = true
	}

	for _, expectedID := range expectedModels {
		if !modelIDs[expectedID] {
			t.Errorf("Expected model %s not found in results", expectedID)
		}
	}
}

// TestGetModels_Context проверяет обработку контекста
func TestGetModels_Context(t *testing.T) {
	client := NewHuggingFaceClient("test-key", "")

	// Тест с отмененным контекстом
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем сразу

	models, err := client.GetModels(ctx, "test-request-id")

	// GetModels не использует контекст для HTTP запросов (возвращает статический список)
	// Поэтому он должен вернуть успешный результат даже с отмененным контекстом
	if err != nil {
		t.Logf("GetModels() returned error (may be expected): %v", err)
	}

	// Проверяем, что модели все равно возвращаются (так как это статический список)
	if len(models) == 0 {
		t.Error("GetModels() should return models even with cancelled context (static list)")
	}
}

// TestHuggingFaceRequest проверяет структуру запроса
func TestHuggingFaceRequest(t *testing.T) {
	request := HuggingFaceRequest{
		Inputs: "Test prompt",
		Parameters: map[string]interface{}{
			"max_new_tokens": 512,
			"temperature":    0.7,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var decoded HuggingFaceRequest
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	if decoded.Inputs != request.Inputs {
		t.Errorf("Decoded Inputs = %v, want %v", decoded.Inputs, request.Inputs)
	}
}

// TestHuggingFaceResponse проверяет структуру ответа
func TestHuggingFaceResponse(t *testing.T) {
	responseJSON := `[{"generated_text": "Test response"}]`

	var response HuggingFaceResponse
	if err := json.Unmarshal([]byte(responseJSON), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Response length = %d, want 1", len(response))
	}

	if response[0].GeneratedText != "Test response" {
		t.Errorf("GeneratedText = %v, want 'Test response'", response[0].GeneratedText)
	}
}

// TestHandleGetModels_HuggingFace_Integration проверяет полный путь от HTTP-запроса до ответа
func TestHandleGetModels_HuggingFace_Integration(t *testing.T) {
	// Создаем тестовый сервер
	srv, cleanup := setupTestServer(t)
	defer cleanup()

	// Создаем HTTP запрос с параметром provider=huggingface
	req := httptest.NewRequest("GET", "/api/workers/models?provider=huggingface", nil)
	w := httptest.NewRecorder()

	// Вызываем обработчик
	srv.handleGetModels(w, req)

	// Проверяем статус код
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d. Body: %s", http.StatusOK, w.Code, w.Body.String())
	}

	// Парсим ответ
	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Models []struct {
				ID     string `json:"id"`
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"models"`
			Provider string `json:"provider"`
			Total    int    `json:"total"`
		} `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v. Body: %s", err, w.Body.String())
	}

	// Проверяем успешность запроса
	if !response.Success {
		t.Error("Expected success=true in response")
	}

	// Проверяем, что провайдер указан правильно
	if response.Data.Provider != "huggingface" {
		t.Errorf("Expected provider 'huggingface', got '%s'", response.Data.Provider)
	}

	// Проверяем количество моделей (должно быть 7)
	expectedCount := 7
	if len(response.Data.Models) != expectedCount {
		t.Errorf("Expected %d models, got %d", expectedCount, len(response.Data.Models))
	}

	// Проверяем наличие конкретных моделей
	expectedModels := map[string]bool{
		"microsoft/DialoGPT-medium":           false,
		"google/flan-t5-base":                 false,
		"meta-llama/Llama-2-7b-chat-hf":      false,
		"mistralai/Mistral-7B-Instruct-v0.1": false,
		"google/gemma-7b-it":                 false,
		"HuggingFaceH4/zephyr-7b-beta":       false,
		"tiiuae/falcon-7b-instruct":          false,
	}

	for _, model := range response.Data.Models {
		if expectedModels[model.ID] {
			expectedModels[model.ID] = true
		}
		// Также проверяем по Name, если ID не совпадает
		for expectedID := range expectedModels {
			if model.Name != "" && strings.Contains(model.Name, expectedID) {
				expectedModels[expectedID] = true
			}
		}
	}

	// Проверяем, что все ожидаемые модели найдены
	for modelID, found := range expectedModels {
		if !found {
			t.Errorf("Expected model %s not found in response", modelID)
		}
	}

	// Проверяем структуру моделей
	for i, model := range response.Data.Models {
		if model.ID == "" {
			t.Errorf("Model %d ID should not be empty", i)
		}
		if model.Status == "" {
			t.Errorf("Model %d Status should not be empty", i)
		}
	}
}

// TestChatCompletion_RequestFormat проверяет формат запроса
func TestChatCompletion_RequestFormat(t *testing.T) {
	var capturedRequest HuggingFaceRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Сохраняем запрос для проверки
		json.NewDecoder(r.Body).Decode(&capturedRequest)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := HuggingFaceResponse{
			{GeneratedText: "OK"},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &HuggingFaceClient{
		baseURL:    server.URL,
		apiKey:     "test-key",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}

	messages := []nomenclature.Message{
		{Role: "system", Content: "System"},
		{Role: "user", Content: "User"},
	}

	_, err := client.ChatCompletion("test-model", messages)
	if err != nil {
		t.Fatalf("ChatCompletion() returned error: %v", err)
	}

	// Проверяем формат запроса
	if capturedRequest.Inputs == "" {
		t.Error("Request Inputs should not be empty")
	}

	// Проверяем наличие параметров
	if capturedRequest.Parameters == nil {
		t.Error("Request Parameters should not be nil")
	}

	maxTokens, ok := capturedRequest.Parameters["max_new_tokens"].(float64)
	if !ok || maxTokens != 512 {
		t.Errorf("Expected max_new_tokens=512, got %v", maxTokens)
	}
}

