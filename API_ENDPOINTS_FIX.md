# Исправление проблем с API endpoints

## Проблемы

1. **404 Not Found** для `/api/counterparties/all?client_id=1&offset=0&limit=1000`
2. **500 Internal Server Error** для `/api/quality/stats?project=1:3`
3. **500 Internal Server Error** для `/api/quality/stats?database=...&project=1:3`

## Причины

1. **`/api/counterparties/all`** - роут был зарегистрирован только в Gin, но не было fallback роута через mux
2. **`/api/quality/stats`** - роут вообще не был зарегистрирован ни в Gin, ни в mux fallback

## Решение

### 1. Добавлен роут `/api/quality/stats` в Gin

В `server/server_start_shutdown.go` добавлен роут:
```go
qualityAPI.GET("/stats", func(c *gin.Context) {
    currentDB := s.db
    if currentDB == nil {
        currentDB = s.normalizedDB
    }
    s.qualityHandler.HandleQualityStats(c.Writer, c.Request, currentDB, s.currentNormalizedDBPath)
})
```

### 2. Добавлены fallback роуты через mux

Добавлены fallback роуты для совместимости:
- `/api/quality/stats` - с оберткой для передачи currentDB
- `/api/counterparties/all` - прямой вызов handler'а
- `/api/counterparties/all/export` - прямой вызов handler'а

### 3. Улучшена обработка ошибок в `aggregateProjectStats`

Добавлена проверка на `nil` для `currentDB`:
```go
if currentDB == nil {
    if h.normalizedDB == nil {
        return nil, apperrors.NewInternalError("database connection is not available", nil)
    }
    currentDB = h.normalizedDB
}
```

## Изменения в коде

### server/server_start_shutdown.go
- Добавлен роут `/api/quality/stats` в Gin
- Добавлены fallback роуты через mux для `/api/quality/stats` и `/api/counterparties/all`

### server/handlers/quality.go
- Улучшена обработка `nil` для `currentDB` в `aggregateProjectStats`

## Результат

Теперь оба endpoint'а должны работать корректно:
- `/api/counterparties/all` - доступен через Gin и mux fallback
- `/api/quality/stats` - доступен через Gin и mux fallback

## Следующие шаги

1. Пересобрать Go backend: `go build`
2. Перезапустить сервер
3. Проверить работу endpoints в интерфейсе

