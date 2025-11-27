# Пример интеграции новой архитектуры в server.go

## Быстрая интеграция (минимальные изменения)

Добавить в `server/server.go`:

### 1. Добавить поле в структуру Server

```go
type Server struct {
    // ... существующие поля ...
    
    // Новая архитектура (опционально, для постепенной миграции)
    newContainer *container.Container
}
```

### 2. В методе Start() или NewServer() добавить создание нового контейнера

```go
// После инициализации существующих компонентов:
// Создаем новый container для enterprise-архитектуры
newContainer, err := container.NewContainer(s.config)
if err != nil {
    log.Printf("Warning: failed to create new container: %v", err)
} else {
    if err := newContainer.Initialize(); err != nil {
        log.Printf("Warning: failed to initialize new container: %v", err)
    } else {
        s.newContainer = newContainer
        log.Println("New architecture container initialized successfully")
    }
}
```

### 3. В методе setupRouter() добавить регистрацию новых маршрутов

```go
// После создания mux := http.NewServeMux()

// Регистрируем новые маршруты через Router (новая архитектура)
if s.newContainer != nil {
    newRouter, err := routes.NewRouter(mux, s.newContainer)
    if err == nil {
        newRouter.RegisterAllRoutes()
        log.Println("New architecture routes registered")
    }
}

// Старые маршруты продолжают работать как обычно
if s.uploadHandler != nil {
    mux.HandleFunc("/handshake", s.uploadHandler.HandleHandshake)
    // ...
}
```

### 4. Альтернатива: Прямая регистрация только upload маршрутов

Если не нужен полноценный Router, можно напрямую зарегистрировать только upload:

```go
// После создания mux := http.NewServeMux()

// Регистрируем новые upload маршруты (параллельно со старыми)
if s.newContainer != nil {
    newUploadHandler, err := s.newContainer.GetUploadHandler()
    if err == nil && newUploadHandler != nil {
        // Используем префикс /v2/ для новых endpoints (опционально)
        // Или используем те же пути, но новые handlers будут иметь приоритет
        routes.RegisterUploadRoutes(mux, newUploadHandler)
        log.Println("New upload handlers registered")
    }
}
```

## Полная интеграция с переключением через feature flag

### 1. Добавить в config.go

```go
type Config struct {
    // ... существующие поля ...
    
    // Feature flags
    UseNewArchitecture bool `mapstructure:"use_new_architecture"`
}
```

### 2. В setupRouter() использовать feature flag

```go
// Регистрируем маршруты в зависимости от feature flag
if s.config.UseNewArchitecture && s.newContainer != nil {
    // Используем новую архитектуру
    newRouter, err := routes.NewRouter(mux, s.newContainer)
    if err == nil {
        newRouter.RegisterAllRoutes()
        log.Println("Using new architecture for all routes")
    }
} else {
    // Используем старые handlers
    if s.uploadHandler != nil {
        mux.HandleFunc("/handshake", s.uploadHandler.HandleHandshake)
        // ...
    }
}
```

## Постепенная миграция отдельных endpoints

### Пример миграции /handshake endpoint

```go
// В setupRouter():

// Определяем, какой handler использовать для /handshake
if s.config.UseNewArchitecture && s.newContainer != nil {
    newUploadHandler, err := s.newContainer.GetUploadHandler()
    if err == nil && newUploadHandler != nil {
        // Используем новый handler
        mux.HandleFunc("/handshake", newUploadHandler.HandleHandshake)
        log.Println("Using new handler for /handshake")
    } else {
        // Fallback к старому handler
        if s.uploadHandler != nil {
            mux.HandleFunc("/handshake", s.uploadHandler.HandleHandshake)
        }
    }
} else {
    // Используем старый handler
    if s.uploadHandler != nil {
        mux.HandleFunc("/handshake", s.uploadHandler.HandleHandshake)
    }
}
```

## Проверка интеграции

После добавления кода:

1. Компиляция:
```bash
go build ./server
```

2. Запуск сервера:
```bash
./server
```

3. Проверка логов:
   - Должно появиться сообщение о инициализации нового контейнера
   - Должно появиться сообщение о регистрации новых маршрутов

4. Тестирование endpoints:
   - Старые endpoints должны работать как обычно
   - Новые endpoints должны работать (если используете их)

## Откат изменений

Если что-то пошло не так, можно легко откатить:

1. Закомментировать создание нового контейнера
2. Закомментировать регистрацию новых маршрутов
3. Старые handlers продолжат работать

## Следующие шаги

После успешной интеграции:

1. Постепенно мигрировать endpoints на новые handlers
2. Тестировать каждый endpoint отдельно
3. Мониторить логи и ошибки
4. После полной миграции удалить старый код

