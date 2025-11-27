# 🚀 Реализация мониторинга провайдеров AI

## 📋 Обзор

Создана система мониторинга в реальном времени для отслеживания использования всех AI-провайдеров (OpenRouter, Hugging Face, Arliai) в процессах нормализации.

## ✅ Выполненные задачи

### 1. Backend (Go)

#### ✅ Создан `server/monitoring.go`
- `MonitoringManager` - центральный менеджер для сбора метрик
- `ProviderMetrics` - структура метрик для каждого провайдера
- `SystemStats` - общая статистика системы
- Методы:
  - `RegisterProvider()` - регистрация провайдера
  - `IncrementRequest()` - запись начала запроса
  - `RecordResponse()` - запись завершения запроса
  - `GetAllMetrics()` - получение всех метрик

#### ✅ Создан `server/monitoring_handlers.go`
- `handleMonitoringProvidersStream()` - SSE эндпоинт для трансляции метрик
- `handleMonitoringProviders()` - одноразовый запрос метрик
- Обновление каждую секунду через SSE

#### ✅ Интеграция в `server/multi_provider_client.go`
- Добавлен `monitoringManager` в структуру `MultiProviderClient`
- Интегрирован сбор метрик в `NormalizeName()`:
  - `RecordRequest()` перед запросом
  - `RecordResponse()` после получения ответа

#### ✅ Интеграция в `server/server.go`
- Добавлено поле `monitoringManager` в структуру `Server`
- Инициализация в `NewServerWithConfig()`
- Регистрация провайдеров:
  - Arliai: 2 канала
  - OpenRouter: 1 канал
  - Hugging Face: 1 канал
- Добавлены эндпоинты:
  - `/api/monitoring/providers/stream` - SSE поток
  - `/api/monitoring/providers` - одноразовый запрос

## 📊 Структура данных

### ProviderMetrics
```go
type ProviderMetrics struct {
    ID                string    // "openrouter", "huggingface", "arliai"
    Name              string    // "OpenRouter", "Hugging Face", "Arliai"
    ActiveChannels    int       // Количество активных каналов
    CurrentRequests   int       // Текущие активные запросы
    TotalRequests     int64     // Всего запросов
    SuccessfulRequests int64    // Успешных запросов
    FailedRequests    int64     // Неудачных запросов
    AverageLatencyMs  float64   // Средняя задержка в мс
    Status            string    // "active", "idle", "error"
    RequestsPerSecond float64   // Запросов в секунду
}
```

### SystemStats
```go
type SystemStats struct {
    TotalProviders      int
    ActiveProviders     int
    TotalRequests       int64
    TotalSuccessful     int64
    TotalFailed         int64
    SystemRequestsPerSecond float64
    Timestamp           time.Time
}
```

## 🔧 API Эндпоинты

### GET `/api/monitoring/providers/stream`
**SSE поток метрик провайдеров**

**Заголовки:**
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`
- `Connection: keep-alive`

**Формат данных:**
```
data: {"providers": [...], "system": {...}}

```

**Обновление:** Каждую секунду

### GET `/api/monitoring/providers`
**Одноразовый запрос текущих метрик**

**Ответ:**
```json
{
  "providers": [
    {
      "id": "arliai",
      "name": "Arliai",
      "active_channels": 2,
      "current_requests": 1,
      "total_requests": 150,
      "successful_requests": 145,
      "failed_requests": 5,
      "average_latency_ms": 234.5,
      "status": "active",
      "requests_per_second": 2.5
    }
  ],
  "system": {
    "total_providers": 3,
    "active_providers": 2,
    "total_requests": 450,
    "total_successful": 430,
    "total_failed": 20,
    "system_requests_per_second": 7.5,
    "timestamp": "2025-11-21T22:00:00Z"
  }
}
```

## 🎯 Следующие шаги (Frontend)

### Требуется реализовать:

1. **Обновить `frontend/app/monitoring/page.tsx`**
   - Добавить секцию для мониторинга провайдеров
   - Использовать SSE хук для получения данных
   - Отобразить карточки для каждого провайдера

2. **Создать компоненты визуализации**
   - `ProviderCard` - карточка провайдера с метриками
   - `ProviderComparisonChart` - сравнение провайдеров (BarChart)
   - `RequestsTimelineChart` - временной график запросов (LineChart)
   - `SuccessRatePieChart` - соотношение успех/ошибки (PieChart)

3. **Установить зависимости**
   ```bash
   npm install recharts
   ```

4. **Структура компонента**
   ```tsx
   interface ProviderMetrics {
     id: string
     name: string
     active_channels: number
     current_requests: number
     total_requests: number
     successful_requests: number
     failed_requests: number
     average_latency_ms: number
     status: 'active' | 'idle' | 'error'
     requests_per_second: number
   }
   ```

## 📝 Примечания

- Мониторинг автоматически собирает метрики при использовании `MultiProviderClient`
- История запросов хранится в памяти (последние 60 секунд)
- Метрики обновляются в реальном времени через SSE
- Статус провайдера определяется автоматически на основе ошибок

## 🔄 Интеграция с существующим кодом

Мониторинг интегрирован в:
- ✅ `MultiProviderClient.NormalizeName()` - основной метод нормализации
- ✅ `Server` - инициализация и регистрация провайдеров
- ✅ SSE эндпоинт для фронтенда

**Готово к использованию!** 🎉

