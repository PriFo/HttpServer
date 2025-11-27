# Управление кэшем статистики качества проектов

## Обзор

Кэш `ProjectQualityStatsCache` используется для хранения результатов агрегации статистики качества по проектам. Это позволяет значительно ускорить повторные запросы и снизить нагрузку на сервер.

## Основные возможности

### Автоматическая очистка

Кэш автоматически очищает устаревшие записи:
- **TTL**: 5 минут (настраивается)
- **Фоновая очистка**: каждую минуту
- **Проверка при чтении**: устаревшие записи удаляются при попытке чтения

### Методы управления

#### Получение данных
```go
stats, found := cache.Get("project:123")
if found {
    // Использовать кэшированные данные
}
```

#### Сохранение данных
```go
cache.Set("project:123", aggregatedStats)
```

#### Инвалидация
```go
// Удалить конкретный проект
cache.Invalidate("project:123")
cache.InvalidateProject(123) // Альтернативный способ

// Очистить весь кэш
cache.Clear()
```

#### Статистика кэша
```go
stats := cache.GetStats()
// Возвращает:
// - total_entries: общее количество записей
// - valid_entries: количество валидных записей
// - expired_entries: количество устаревших записей
// - ttl_seconds: TTL в секундах
// - total_hits / total_misses: количество попаданий и промахов
// - hit_rate: коэффициент попаданий (0..1)
// - entries: массив с подробной информацией по каждой записи (project_id, cached_at, last_access, hit_count, возраст, время до истечения)
```

## Когда инвалидировать кэш

### Автоматическая инвалидация

Кэш автоматически инвалидируется через TTL (5 минут). Это означает, что:
- Статистика обновляется не чаще, чем раз в 5 минут
- Для большинства случаев этого достаточно

### Ручная инвалидация

Рекомендуется инвалидировать кэш в следующих случаях:

1. **После обновления данных проекта**
   - Добавление новой базы данных
   - Удаление базы данных
   - Изменение статуса базы данных (активна/неактивна)

2. **После запуска анализа качества**
   - Когда завершается анализ качества для проекта
   - Когда обновляются эталоны

3. **После массовых операций**
   - Импорт данных
   - Нормализация данных
   - Слияние дубликатов

## Примеры использования

### Инвалидация после обновления базы данных

```go
// В handler для обновления базы данных
func (h *DatabaseHandler) HandleUpdateDatabase(w http.ResponseWriter, r *http.Request) {
    // ... обновление базы данных ...
    
    // Инвалидируем кэш для всех проектов, использующих эту БД
    if h.qualityHandler != nil && h.qualityHandler.projectStatsCache != nil {
        // Получаем список проектов, использующих БД
        projects := getProjectsForDatabase(dbID)
        for _, projectID := range projects {
            h.qualityHandler.projectStatsCache.InvalidateProject(projectID)
        }
    }
}
```

### Инвалидация после анализа качества

```go
// В handler для запуска анализа качества
func (h *QualityHandler) HandleQualityAnalysis(w http.ResponseWriter, r *http.Request) {
    projectID := extractProjectID(r)
    
    // ... запуск анализа ...
    
    // Инвалидируем кэш после завершения
    if h.projectStatsCache != nil {
        h.projectStatsCache.InvalidateProject(projectID)
    }
}
```

### Мониторинг кэша

```go
// Endpoint для мониторинга кэша
func (h *QualityHandler) HandleCacheStats(w http.ResponseWriter, r *http.Request) {
    if h.projectStatsCache == nil {
        h.WriteJSONResponse(w, r, map[string]interface{}{
            "enabled": false,
        }, http.StatusOK)
        return
    }
    
    stats := h.projectStatsCache.GetStats()
    h.WriteJSONResponse(w, r, map[string]interface{}{
        "enabled": true,
        "stats": stats,
    }, http.StatusOK)
}
```

## Настройка TTL

TTL можно настроить при создании кэша:

```go
// Короткий TTL для часто обновляемых данных
cache := handlers.NewProjectQualityStatsCache(1 * time.Minute)

// Длинный TTL для стабильных данных
cache := handlers.NewProjectQualityStatsCache(15 * time.Minute)

// По умолчанию: 5 минут
cache := handlers.NewProjectQualityStatsCache(0) // Использует дефолт
```

## Производительность

### Ожидаемые показатели

- **Hit rate**: > 50% для активных проектов
- **Время чтения из кэша**: < 1ms
- **Время записи в кэш**: < 1ms
- **Память**: ~1-5 KB на проект (зависит от размера статистики)

### Оптимизация

1. **Увеличить TTL** для стабильных проектов
2. **Уменьшить TTL** для часто обновляемых проектов
3. **Использовать инвалидацию** вместо очистки всего кэша
4. **Мониторить размер кэша** через `GetStats()`

## Отладка

### Логирование

Кэш логирует следующие события:
- Возврат из кэша: `INFO: Returning cached stats for project X`
- Сохранение в кэш: автоматически при агрегации
- Очистка: автоматически в фоне

### Проверка состояния

```go
// Проверить размер кэша
size := cache.Size()

// Получить статистику
stats := cache.GetStats()
log.Printf("Cache stats: %+v", stats)
```

## Рекомендации

1. **Не инвалидировать слишком часто** - это снижает эффективность кэша
2. **Использовать инвалидацию по проектам** вместо очистки всего кэша
3. **Мониторить hit rate** - если < 30%, возможно TTL слишком короткий
4. **Настроить TTL** в зависимости от частоты обновления данных

