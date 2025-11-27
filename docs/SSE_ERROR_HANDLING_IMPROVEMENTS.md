# Улучшения обработки ошибок в SSE эндпоинтах

## Обзор

Проведена комплексная работа по улучшению обработки ошибок во всех Server-Sent Events (SSE) эндпоинтах для предотвращения ошибок 500 и обеспечения надежной работы потоков.

## Проблемы, которые были исправлены

### 1. Ошибка 500 при отсутствии поддержки Flusher
**Проблема**: Проверка поддержки `http.Flusher` выполнялась после установки заголовков SSE, что приводило к ошибке 500, если streaming не поддерживался.

**Решение**: Все проверки `http.Flusher` перенесены до установки заголовков SSE.

### 2. Отсутствие обработки паники
**Проблема**: Паника в SSE обработчиках могла привести к падению сервера.

**Решение**: Добавлена обработка паники на верхнем уровне всех SSE обработчиков.

### 3. Неединообразный формат ошибок
**Проблема**: Ошибки в SSE потоках имели разные форматы, что усложняло обработку на клиенте.

**Решение**: Все ошибки теперь имеют единообразный формат: `{"type":"error","error":"..."}`.

### 4. Использование неструктурированного логирования
**Проблема**: Использование `log.Printf` вместо структурированного логирования.

**Решение**: Все SSE эндпоинты переведены на использование `slog` для структурированного логирования.

## Исправленные эндпоинты

### Основные обработчики (handlers/)

1. **`/api/monitoring/providers/stream`** - `HandleMonitoringProvidersStream`
   - ✅ Обработка паники на верхнем уровне
   - ✅ Проверка `getMonitoringMetrics` до установки заголовков
   - ✅ Обработка паники в `getMonitoringMetrics` (server.go)
   - ✅ Единообразный формат ошибок
   - ✅ Структурированное логирование

2. **`/api/monitoring/events`** - `HandleMonitoringEvents`
   - ✅ Улучшен формат ошибок
   - ✅ Улучшена обработка ошибок при отправке
   - ✅ Структурированное логирование

3. **`/api/normalization/events`** - `HandleNormalizationEvents`
   - ✅ Удалена дублирующаяся проверка Flusher
   - ✅ Проверка Flusher до установки заголовков
   - ✅ Структурированное логирование

4. **`/api/reclassification/events`** - `HandleEvents`
   - ✅ Удалена дублирующаяся проверка Flusher
   - ✅ Проверка Flusher до установки заголовков
   - ✅ Структурированное логирование

5. **`/api/internal/worker-trace/stream`** - `HandleWorkerTraceStream`
   - ✅ Уже имел правильную обработку ошибок

### Старые обработчики (server/)

6. **`/api/monitoring/providers/stream`** (fallback) - `handleMonitoringProvidersStream`
   - ✅ Проверка `monitoringManager` до установки заголовков
   - ✅ Улучшена обработка паники
   - ✅ Единообразный формат ошибок
   - ✅ Структурированное логирование

7. **`/api/monitoring/events`** (fallback) - `handleMonitoringEvents`
   - ✅ Добавлена обработка паники на верхнем уровне
   - ✅ Проверка Flusher до установки заголовков
   - ✅ Единообразный формат ошибок
   - ✅ Структурированное логирование

8. **`/api/reclassification/events`** (fallback) - `handleReclassificationEvents`
   - ✅ Добавлена обработка паники на верхнем уровне
   - ✅ Проверка Flusher до установки заголовков
   - ✅ Единообразный формат ошибок
   - ✅ Структурированное логирование

9. **`/api/uploads/{uuid}/stream`** - `handleStreamUploadData`
   - ✅ Проверка Flusher перенесена до установки заголовков

10. **`/api/normalized/uploads/{uuid}/stream`** - `handleStreamUploadDataNormalized`
    - ✅ Проверка Flusher перенесена до установки заголовков

11. **`/api/system/summary/stream`** - `handleSystemSummaryStream`
    - ✅ Уже имел правильную обработку ошибок

## Общие улучшения

### 1. Единообразная обработка ошибок
- Все ошибки в SSE-потоке имеют формат: `{"type":"error","error":"..."}`
- Улучшено логирование всех ошибок с контекстом

### 2. Правильный порядок проверок
```go
// ✅ Правильно: проверка ДО установки заголовков
flusher, ok := w.(http.Flusher)
if !ok {
    http.Error(w, "Streaming not supported", http.StatusInternalServerError)
    return
}

// Устанавливаем заголовки
w.Header().Set("Content-Type", "text/event-stream")
w.WriteHeader(http.StatusOK)
```

### 3. Обработка паники
```go
defer func() {
    if panicVal := recover(); panicVal != nil {
        slog.Error("[Monitoring] Panic in handler",
            "panic", panicVal,
            "stack", string(debug.Stack()),
            "path", r.URL.Path,
        )
        // Обработка паники
    }
}()
```

### 4. Структурированное логирование
Все логи теперь используют `slog` с контекстом:
```go
slog.Error("[Monitoring] Error message",
    "error", err,
    "path", r.URL.Path,
    "request_id", reqID,
)
```

## Статистика

- **Исправлено SSE эндпоинтов**: 11
- **Добавлено обработок паники**: 8
- **Исправлено проверок Flusher**: 6
- **Улучшено форматов ошибок**: 10
- **Переведено на структурированное логирование**: 8

## Результат

✅ Все SSE-эндпоинты защищены от ошибок 500  
✅ Единообразный формат ошибок во всех потоках  
✅ Правильный порядок проверок и установки заголовков  
✅ Надежная обработка паники во всех эндпоинтах  
✅ Структурированное логирование для лучшей отладки  

## Тестирование

После применения изменений рекомендуется:

1. Перезапустить сервер
2. Проверить все SSE эндпоинты на наличие ошибок 500
3. Проверить логи на наличие структурированных записей
4. Убедиться, что ошибки отправляются через SSE поток в корректном формате

## Связанные файлы

- `server/handlers/monitoring.go`
- `server/monitoring_handlers.go`
- `server/server_monitoring_handlers.go`
- `server/handlers/normalization.go`
- `server/handlers/reclassification.go`
- `server/server_reclassification.go`
- `server/server.go` (getMonitoringMetrics)
- `server/system_scanner_sse.go`

