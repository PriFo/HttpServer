package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
)

// RequestIDKey ключ для request ID в контексте
type RequestIDKey struct{}

// RequestIDMiddleware добавляет уникальный request ID к каждому запросу
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Генерируем или получаем request ID из заголовка
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = uuid.New().String()
		}

		// Добавляем request ID в контекст
		ctx := SetRequestID(r.Context(), reqID)

		// Добавляем request ID в заголовок ответа
		w.Header().Set("X-Request-ID", reqID)

		// Передаем запрос с обновленным контекстом
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID извлекает request ID из контекста
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	
	reqID, ok := ctx.Value(RequestIDKey{}).(string)
	if !ok {
		return ""
	}
	return reqID
}

// SetRequestID устанавливает request ID в контекст
func SetRequestID(ctx context.Context, reqID string) context.Context {
	return context.WithValue(ctx, RequestIDKey{}, reqID)
}

