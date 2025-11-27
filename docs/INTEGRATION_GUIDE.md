# Руководство по интеграции новой Upload архитектуры

## Дата: 2025-01-21

## ✅ Статус интеграции

Новая архитектура Upload Domain (Clean Architecture) полностью интегрирована в DI Container.

### Компоненты

1. **Domain Layer** (`internal/domain/upload/`)
   - ✅ `Service` - интерфейс бизнес-логики
   - ✅ `service` - реализация domain service

2. **Application Layer** (`internal/application/upload/`)
   - ✅ `UseCase` - координация между domain и infrastructure

3. **Infrastructure Layer** (`internal/infrastructure/`)
   - ✅ `persistence/upload_repository.go` - репозиторий
   - ✅ `services/database_info_adapter.go` - адаптер для DatabaseInfoService

4. **Presentation Layer** (`internal/api/handlers/upload/`)
   - ✅ `Handler` - HTTP обработчики
   - ✅ Все endpoints реализованы

5. **Routes** (`internal/api/routes/`)
   - ✅ `RegisterUploadRoutes` - регистрация маршрутов

6. **DI Container** (`internal/container/`)
   - ✅ Поля для новых компонентов добавлены
   - ✅ `initUploadComponents()` вызывается в `Initialize()`
   - ✅ `GetUploadHandler()` для получения handler

## 📋 Пример интеграции в server.go

### Вариант 1: Использование нового handler параллельно со старым

```go
// В server.go, в методе Start() или NewServerWithConfig()

// Инициализируем контейнер
container, err := container.NewContainer(config)
if err != nil {
    return fmt.Errorf("failed to create container: %w", err)
}

if err := container.Initialize(); err != nil {
    return fmt.Errorf("failed to initialize container: %w", err)
}

// Получаем новый upload handler
uploadHandlerV2, err := container.GetUploadHandler()
if err != nil {
    log.Printf("Warning: Failed to get new upload handler: %v", err)
} else {
    // Регистрируем новые маршруты
    routes.RegisterUploadRoutes(mux, uploadHandlerV2)
    
    // Старые маршруты продолжают работать
    // Можно постепенно мигрировать endpoints
}
```

### Вариант 2: Полная замена старого handler

```go
// В server.go

// 1. Инициализация контейнера
container, err := container.NewContainer(config)
if err != nil {
    return fmt.Errorf("failed to create container: %w", err)
}

if err := container.Initialize(); err != nil {
    return fmt.Errorf("failed to initialize container: %w", err)
}

// 2. Получаем новый handler
uploadHandlerV2, err := container.GetUploadHandler()
if err != nil {
    return fmt.Errorf("failed to get upload handler: %w", err)
}

// 3. Регистрируем маршруты
routes.RegisterUploadRoutes(mux, uploadHandlerV2)

// 4. Старый handler можно оставить для обратной совместимости
// или удалить после тестирования
```

### Вариант 3: Постепенная миграция endpoints

```go
// В server.go

// Регистрируем новые endpoints с префиксом /api/v2
mux.HandleFunc("/api/v2/upload/handshake", uploadHandlerV2.HandleHandshake)
mux.HandleFunc("/api/v2/upload/metadata", uploadHandlerV2.HandleMetadata)
// ... остальные endpoints

// Старые endpoints продолжают работать через старый handler
mux.HandleFunc("/handshake", oldUploadHandler.HandleHandshake)
mux.HandleFunc("/metadata", oldUploadHandler.HandleMetadata)
// ... остальные endpoints

// После тестирования можно переключить старые endpoints на новый handler
```

## 🔄 Миграция endpoints

### Этап 1: Параллельная работа (текущий)
- ✅ Новые handlers готовы
- ✅ Старые handlers продолжают работать
- ✅ Можно тестировать новые endpoints параллельно

### Этап 2: Постепенная миграция
1. Переключить один endpoint на новый handler
2. Протестировать
3. Переключить следующий endpoint
4. Повторить для всех endpoints

### Этап 3: Полная замена
1. Удалить старый handler
2. Удалить старый UploadService
3. Обновить все ссылки на новый handler

## 📝 Преимущества новой архитектуры

1. **Clean Architecture**
   - Четкое разделение слоев
   - Независимость от инфраструктуры
   - Легкое тестирование

2. **DDD (Domain-Driven Design)**
   - Bounded context для Upload
   - Бизнес-логика в domain layer
   - Явные domain модели

3. **Dependency Injection**
   - Все зависимости через конструкторы
   - Легкая замена реализаций
   - Упрощенное тестирование

4. **Модульность**
   - Каждый компонент в отдельном пакете
   - Низкая связанность
   - Высокая связность внутри модуля

## 🧪 Тестирование

### Unit тесты
```go
// Пример теста для domain service
func TestUploadService_ProcessHandshake(t *testing.T) {
    // Arrange
    mockRepo := &MockUploadRepository{}
    mockDBInfo := &MockDatabaseInfoService{}
    service := uploaddomain.NewService(mockRepo, mockDBInfo)
    
    // Act
    result, err := service.ProcessHandshake(ctx, req)
    
    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Integration тесты
```go
// Пример интеграционного теста
func TestUploadHandler_HandleHandshake(t *testing.T) {
    // Arrange
    container := setupTestContainer(t)
    handler, _ := container.GetUploadHandler()
    
    // Act
    req := httptest.NewRequest("POST", "/handshake", body)
    w := httptest.NewRecorder()
    handler.HandleHandshake(w, req)
    
    // Assert
    assert.Equal(t, http.StatusOK, w.Code)
}
```

## 🚀 Следующие шаги

1. **Интегрировать в server.go**
   - Добавить инициализацию контейнера
   - Зарегистрировать маршруты
   - Протестировать endpoints

2. **Рефакторинг других доменов**
   - Normalization domain
   - Quality domain
   - Classification domain
   - Counterparty domain

3. **Добавить тесты**
   - Unit тесты для domain services
   - Integration тесты для handlers
   - E2E тесты для endpoints

4. **Документация**
   - API документация
   - Архитектурная документация
   - Руководства для разработчиков

## 📚 Дополнительные ресурсы

- `docs/REFACTORING_PLAN.md` - план рефакторинга
- `docs/REFACTORING_PROGRESS.md` - прогресс рефакторинга
- `docs/REFACTORING_NEXT_STEPS.md` - следующие шаги
- `docs/COMPILATION_FIXES_COMPLETE.md` - исправления ошибок компиляции
