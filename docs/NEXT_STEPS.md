# Следующие шаги рефакторинга

## Дата: 2025-01-21

## ✅ Текущий статус

### Завершено:
1. **Upload Domain** - полностью реализован по Clean Architecture
   - ✅ Domain, Application, Infrastructure, Presentation слои
   - ✅ Интегрирован в Container
   - ✅ Интегрирован в server.go
   - ✅ Endpoints зарегистрированы с префиксом `/api/v2`

2. **Исправления ошибок компиляции**
   - ✅ Все критические ошибки исправлены
   - ✅ Использование ValidationResult обновлено

### В процессе:
1. **Normalization Domain** - частично реализован
   - ✅ Domain Layer
   - ✅ Application Layer
   - ✅ Presentation Layer (handler создан)
   - ⏳ Интеграция в Container
   - ⏳ Интеграция в server.go
   - ⏳ Регистрация маршрутов

2. **Quality Domain** - частично реализован
   - ✅ Domain Layer
   - ✅ Application Layer
   - ✅ Presentation Layer (handler создан)
   - ⏳ Интеграция в Container
   - ⏳ Интеграция в server.go
   - ⏳ Регистрация маршрутов

## 🎯 Приоритетные задачи

### 1. Завершить интеграцию Normalization Domain

**Шаги:**
1. Создать `initNormalizationComponents()` в Container
2. Добавить поля для Normalization компонентов в Container
3. Вызвать `initNormalizationComponents()` в `Initialize()`
4. Создать метод `GetNormalizationHandler()` в Container
5. Интегрировать в server.go (аналогично Upload)
6. Зарегистрировать маршруты с префиксом `/api/v2/normalization`

**Файлы для создания/изменения:**
- `internal/container/normalization_init.go` (новый)
- `internal/container/container.go` (добавить поля и вызов)
- `server/server.go` (интеграция)
- `internal/api/routes/normalization_routes.go` (новый)

### 2. Завершить интеграцию Quality Domain

**Шаги:**
1. Создать `initQualityComponents()` в Container
2. Добавить поля для Quality компонентов в Container
3. Вызвать `initQualityComponents()` в `Initialize()`
4. Создать метод `GetQualityHandler()` в Container
5. Интегрировать в server.go
6. Зарегистрировать маршруты с префиксом `/api/v2/quality`

**Файлы для создания/изменения:**
- `internal/container/quality_init.go` (новый)
- `internal/container/container.go` (добавить поля и вызов)
- `server/server.go` (интеграция)
- `internal/api/routes/quality_routes.go` (новый)

### 3. Решить циклическую зависимость в websearch

**Проблема:**
- `websearch/router.go` → `websearch/providers`
- `websearch/providers/*.go` → `websearch`

**Решение:**
1. Вынести общие типы (`SearchResult`, `SearchItem`) в `websearch/types` (уже есть)
2. Или изменить структуру: `websearch/providers` не должен импортировать `websearch`
3. Использовать интерфейсы вместо конкретных типов

**Приоритет:** Низкий (не блокирует основную функциональность)

## 📋 Шаблон для интеграции нового домена

### Шаг 1: Создать init файл в Container

```go
// internal/container/{domain}_init.go
package container

import (
    "httpserver/internal/api/handlers/{domain}"
    "{domain}app" "httpserver/internal/application/{domain}"
    "{domain}domain" "httpserver/internal/domain/{domain}"
    "httpserver/internal/infrastructure/persistence"
    "httpserver/server/handlers"
)

func (c *Container) init{Domain}Components() error {
    // 1. Создаем репозиторий
    repo := persistence.New{Domain}Repository(c.DB)
    
    // 2. Создаем domain service
    domainService := {domain}domain.NewService(repo, ...)
    
    // 3. Создаем application use case
    useCase := {domain}app.NewUseCase(repo, domainService)
    
    // 4. Создаем base handler
    baseHandler := handlers.NewBaseHandlerFromMiddleware()
    
    // 5. Создаем HTTP handler
    handler := {domain}.NewHandler(baseHandler, useCase)
    
    // 6. Сохраняем в контейнере
    c.{Domain}HandlerV2 = handler
    c.{Domain}UseCase = useCase
    c.{Domain}DomainService = domainService
    
    return nil
}

func (c *Container) Get{Domain}Handler() (*{domain}.Handler, error) {
    // ... аналогично GetUploadHandler
}
```

### Шаг 2: Добавить поля в Container

```go
// internal/container/container.go
type Container struct {
    // ...
    {Domain}HandlerV2     interface{} // *{domain}.Handler
    {Domain}UseCase       interface{} // *{domain}app.UseCase
    {Domain}DomainService interface{} // *{domain}domain.Service
}
```

### Шаг 3: Вызвать в Initialize()

```go
// internal/container/container.go
func (c *Container) Initialize() error {
    // ...
    if err := c.init{Domain}Components(); err != nil {
        return fmt.Errorf("failed to initialize {domain} components: %w", err)
    }
    // ...
}
```

### Шаг 4: Интегрировать в server.go

```go
// server/server.go
func (s *Server) initNew{Domain}Architecture() {
    // Аналогично initNewUploadArchitecture
}

func (s *Server) Start() error {
    s.initNew{Domain}Architecture()
    // ...
}

func (s *Server) setupRouter() *gin.Engine {
    // ...
    if s.{domain}HandlerV2 != nil {
        // Регистрируем endpoints с префиксом /api/v2/{domain}
    }
}
```

### Шаг 5: Создать routes файл

```go
// internal/api/routes/{domain}_routes.go
package routes

func Register{Domain}Routes(mux *http.ServeMux, handler *{domain}.Handler) {
    mux.HandleFunc("/api/v2/{domain}/...", handler.Handle...)
}
```

## 🧪 Тестирование

### После интеграции каждого домена:

1. **Компиляция**
   ```bash
   go build ./server
   ```

2. **Запуск сервера**
   ```bash
   go run cmd/server/main.go
   ```

3. **Тестирование endpoints**
   ```bash
   curl -X POST http://localhost:9999/api/v2/{domain}/...
   ```

4. **Сравнение со старыми endpoints**
   ```bash
   curl -X POST http://localhost:9999/api/v1/{domain}/...
   ```

## 📊 Прогресс

- ✅ Upload Domain: 100%
- ⏳ Normalization Domain: 60% (Domain, Application, Presentation готовы, нужна интеграция)
- ⏳ Quality Domain: 60% (Domain, Application, Presentation готовы, нужна интеграция)
- ⏳ Classification Domain: 0%
- ⏳ Counterparty Domain: 0%

## 🎯 Цель

Довести все домены до уровня Upload Domain:
- Clean Architecture
- Полная интеграция в Container
- Полная интеграция в server.go
- Endpoints с префиксом `/api/v2`
- Параллельная работа со старыми endpoints

