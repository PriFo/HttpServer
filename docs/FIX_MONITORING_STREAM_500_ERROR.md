# Исправление ошибки 500 для `/api/monitoring/providers/stream`

## Проблема
При обращении к эндпоинту `/api/monitoring/providers/stream` возникала ошибка 500 Internal Server Error. Ошибка происходила из-за паники при преобразовании данных мониторинга или при отсутствии инициализации компонентов.

## Причина
1. **Паника при преобразовании данных**: При преобразовании `MonitoringData` из пакета `server` в `handlers.MonitoringData` могла возникать паника, если данные были некорректными или `nil`.
2. **Отсутствие обработки паники на верхнем уровне**: Обработчики не перехватывали паники, которые происходили до начала цикла SSE или при установке заголовков.
3. **Отсутствие проверки на `nil`**: Не было проверки на `nil` для `serverData.Providers` перед преобразованием.

## Исправления

### 1. `server/server.go` - Функция `getMonitoringMetrics`

#### Добавлена обработка паники при преобразовании провайдеров:
```go
// Преобразуем провайдеры с обработкой паники
var providers []handlers.ProviderMetrics
func() {
    defer func() {
        if r := recover(); r != nil {
            s.log(LogEntry{
                Timestamp: time.Now(),
                Level:     "ERROR",
                Message:   fmt.Sprintf("Panic converting providers: %v", r),
                Endpoint:  "/api/monitoring/providers/stream",
            })
            providers = []handlers.ProviderMetrics{}
        }
    }()
    if serverData.Providers != nil {
        providers = make([]handlers.ProviderMetrics, len(serverData.Providers))
        // ... преобразование ...
    } else {
        providers = []handlers.ProviderMetrics{}
    }
}()
```

#### Добавлена обработка паники при преобразовании временной метки:
```go
// Преобразуем системную статистику с обработкой паники
var timestampStr string
func() {
    defer func() {
        if r := recover(); r != nil {
            s.log(LogEntry{
                Timestamp: time.Now(),
                Level:     "ERROR",
                Message:   fmt.Sprintf("Panic converting timestamp: %v", r),
                Endpoint:  "/api/monitoring/providers/stream",
            })
            timestampStr = time.Now().Format(time.RFC3339)
        }
    }()
    if !serverData.System.Timestamp.IsZero() {
        timestampStr = serverData.System.Timestamp.Format(time.RFC3339)
    } else {
        timestampStr = time.Now().Format(time.RFC3339)
    }
}()
```

### 2. `server/monitoring_handlers.go` - Старый обработчик

#### Добавлена обработка паники на верхнем уровне:
```go
func (s *Server) handleMonitoringProvidersStream(w http.ResponseWriter, r *http.Request) {
    // Обработка паники на верхнем уровне
    defer func() {
        if panicVal := recover(); panicVal != nil {
            log.Printf("[Monitoring] Panic in handleMonitoringProvidersStream: %v, stack: %s", panicVal, debug.Stack())
            // Если заголовки еще не установлены, отправляем обычную ошибку
            if w.Header().Get("Content-Type") != "text/event-stream" {
                http.Error(w, fmt.Sprintf("Internal server error: %v", panicVal), http.StatusInternalServerError)
            }
        }
    }()
    // ... остальной код ...
}
```

#### Улучшена проверка поддержки Flusher:
- Проверка перенесена ДО установки заголовков SSE
- Добавлена обработка ошибок при отправке начального сообщения

### 3. `server/handlers/monitoring.go` - Новый обработчик

#### `HandleMonitoringProvidersStream`:
- ✅ Обработка паники на верхнем уровне (уже была)
- ✅ Обработка ошибок при отправке начального сообщения (уже была)
- ✅ Проверка на `nil` для `getMonitoringMetrics` (уже была)

#### `HandleMonitoringEvents` - улучшен:
- ✅ Добавлена обработка паники на верхнем уровне
- ✅ Проверка Flusher перенесена ДО установки заголовков
- ✅ Добавлена обработка ошибок при отправке начального сообщения
- ✅ Улучшена обработка ошибок при маршалинге метрик
- ✅ Удален лишний `w.WriteHeader(http.StatusOK)` (не нужен для SSE)

### 4. `server/handlers/normalization.go` - `HandleNormalizationEvents`

#### Улучшения:
- ✅ Добавлена обработка паники на верхнем уровне
- ✅ Проверка Flusher перенесена ДО установки заголовков
- ✅ Добавлена обработка ошибок при отправке начального сообщения
- ✅ Удален лишний `w.WriteHeader(http.StatusOK)`

### 5. `server/handlers/reclassification.go` - `HandleEvents`

#### Улучшения:
- ✅ Добавлена обработка паники на верхнем уровне
- ✅ Проверка Flusher перенесена ДО установки заголовков
- ✅ Добавлена обработка ошибок при отправке начального сообщения
- ✅ Удален лишний `w.WriteHeader(http.StatusOK)`

## Результат

После исправлений:
- ✅ Все паники перехватываются и логируются во всех SSE обработчиках
- ✅ При ошибках возвращаются валидные (пустые) данные вместо 500
- ✅ Клиент получает корректный SSE поток даже при ошибках
- ✅ Все ошибки логируются для отладки
- ✅ Все SSE обработчики защищены от паники на верхнем уровне
- ✅ Проверка Flusher выполняется до установки заголовков (избегает проблем с заголовками)
- ✅ Удалены лишние `WriteHeader` вызовы для SSE потоков

## Тестирование

1. Перезапустите сервер
2. Откройте страницу с мониторингом провайдеров
3. Проверьте, что SSE поток подключается без ошибок 500
4. Проверьте логи сервера на наличие сообщений об ошибках

## Дополнительные улучшения

Если проблема сохраняется:
1. Проверьте логи сервера на наличие сообщений о панике
2. Убедитесь, что `monitoringManager` инициализирован корректно
3. Проверьте, что все провайдеры зарегистрированы в `monitoringManager`

