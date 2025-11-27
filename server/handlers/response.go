package handlers

import (
	"encoding/json"
	"net/http"
	"time"
)

// JSONResponse стандартная структура JSON ответа
type JSONResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp string      `json:"timestamp"`
}

// WriteJSONResponse записывает JSON ответ
func WriteJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	response := JSONResponse{
		Success:   statusCode >= 200 && statusCode < 300,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

// WriteJSONError записывает JSON ошибку
func WriteJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)

	response := JSONResponse{
		Success:   false,
		Error:     message,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode JSON error", http.StatusInternalServerError)
	}
}

// WriteJSONSuccess записывает успешный JSON ответ
func WriteJSONSuccess(w http.ResponseWriter, message string, data interface{}) {
	WriteJSONResponse(w, map[string]interface{}{
		"message": message,
		"data":    data,
	}, http.StatusOK)
}

// WriteJSONPaginatedResponse записывает пагинированный JSON ответ
func WriteJSONPaginatedResponse(w http.ResponseWriter, data interface{}, total, page, limit int) {
	response := map[string]interface{}{
		"data":  data,
		"total": total,
		"page":  page,
		"limit": limit,
	}

	if total > 0 {
		response["pages"] = (total + limit - 1) / limit
	}

	WriteJSONResponse(w, response, http.StatusOK)
}
