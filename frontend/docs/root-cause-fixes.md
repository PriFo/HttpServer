# Исправление первопричин ошибок

## Найденные первопричины

### 1. SSE Stream таймаут 15 секунд

**Проблема**: 
- `WriteTimeout: 15 * time.Second` в `server_start_shutdown.go`
- Heartbeat отправлялся каждые 30 секунд
- Если за 15 секунд не было записи, соединение закрывалось
- Heartbeat не успевал отправиться до таймаута

**Решение**:
- ✅ Увеличен `WriteTimeout` до 60 секунд
- ✅ Увеличен `IdleTimeout` до 120 секунд
- ✅ Уменьшен интервал heartbeat до 10 секунд во всех SSE обработчиках
- ✅ Heartbeat теперь гарантированно отправляется до таймаута

**Измененные файлы**:
- `server/server_start_shutdown.go` - увеличены таймауты
- `server/handlers/monitoring.go` - уменьшен интервал heartbeat (2 места)
- `server/handlers/normalization.go` - уменьшен интервал heartbeat

### 2. 404 для `/api/kpved/workers/status`

**Проблема**: 
- Обработчик `HandleKpvedWorkersStatus` существует
- Но роут не был зарегистрирован в `registerGinHandlers`

**Решение**:
- ✅ Добавлена регистрация роутов для KPVED Workers API:
  - `/api/kpved/workers/status` - GET
  - `/api/kpved/workers/stop` - POST
  - `/api/kpved/workers/resume` - POST

**Измененный файл**:
- `server/server_start_shutdown.go` - добавлена регистрация роутов

### 3. 404 для `/api/logs/client-error`

**Проблема**: 
- Console interceptor отправляет на `/api/logs/error`
- Но в логах видно запросы на `/api/logs/client-error`
- Возможно, это из другого места или изменилось

**Решение**:
- ✅ Создан роут `/api/logs/client-error` для совместимости
- ✅ Создан роут `/api/logs/error` для console-interceptor
- ✅ Оба роута логируют ошибки одинаково

**Измененные файлы**:
- `frontend/app/api/logs/client-error/route.ts` - новый роут
- `frontend/app/api/logs/error/route.ts` - новый роут

## Результаты

1. **SSE соединения стабильны** - heartbeat отправляется каждые 10 секунд, таймаут 60 секунд
2. **KPVED workers API работает** - роуты зарегистрированы
3. **Клиентские ошибки логируются** - оба эндпоинта доступны

## Технические детали

### Таймауты сервера

```go
ReadTimeout:  15 * time.Second  // Время чтения запроса
WriteTimeout: 60 * time.Second  // Время записи ответа (увеличено для SSE)
IdleTimeout:  120 * time.Second  // Время простоя соединения (увеличено для SSE)
```

### Heartbeat интервалы

- **Было**: 30 секунд
- **Стало**: 10 секунд
- **Причина**: WriteTimeout 60 секунд, heartbeat каждые 10 секунд гарантирует активность

### Регистрация роутов

```go
// KPVED Workers API
kpvedAPI := api.Group("/kpved")
{
    kpvedAPI.GET("/workers/status", ...)
    kpvedAPI.POST("/workers/stop", ...)
    kpvedAPI.POST("/workers/resume", ...)
}
```

