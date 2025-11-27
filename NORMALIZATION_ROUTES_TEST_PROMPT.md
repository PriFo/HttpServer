# Промпт для генерации интеграционных тестов для новых роутов нормализации

Скопируй этот промпт в новый чат для автоматической генерации интеграционных тестов, проверяющих все изменения из отчета `NORMALIZATION_ROUTES_CODE_REVIEW.md`:

---

## Роль

Ты — Senior Go Developer и QA Automation Engineer. Твоя задача — создать исчерпывающий набор интеграционных тестов для проверки всех изменений, описанных в отчете `NORMALIZATION_ROUTES_CODE_REVIEW.md`.

## Контекст

Ты только что сгенерировал подробный отчет `NORMALIZATION_ROUTES_CODE_REVIEW.md`, в котором описано:

1. **Добавление 7 новых роутов** для API нормализации в приложении на Go/Gin:
   - `POST /api/normalization/stop` - остановка нормализации
   - `GET /api/normalization/pipeline/stage-details` - детали этапа pipeline
   - `GET /api/normalization/export` - экспорт нормализованных данных
   - `GET /api/normalization/config` - получение конфигурации
   - `PUT /api/normalization/config` - обновление конфигурации
   - `POST /api/normalization/config` - создание/обновление конфигурации
   - `GET /api/normalization/databases` - список баз данных
   - `GET /api/normalization/tables` - список таблиц
   - `GET /api/normalization/columns` - список колонок

2. **Рекомендации по добавлению Swagger-документации** для существующих роутов.

Твоя цель — написать тесты, которые подтвердят, что все эти изменения реализованы корректно и работают как положено.

## Основная задача

Сгенерируй один Go-файл `normalization_routes_integration_test.go`, который будет содержать полный набор интеграционных тестов для проверки роутов нормализации. Тесты должны использовать `testify/suite` для организации и `httptest` для HTTP-запросов.

## Детальные требования к тестам

### 1. Настройка тестового окружения

#### 1.1. Структура TestSuite

Создай структуру `NormalizationRoutesIntegrationTestSuite` со следующими полями:

```go
type NormalizationRoutesIntegrationTestSuite struct {
    suite.Suite
    router          *gin.Engine
    normalizationHandler *handlers.NormalizationHandler
    normalizationService *services.NormalizationService
    baseHandler     *handlers.BaseHandler
    // Добавь другие необходимые зависимости
}
```

#### 1.2. SetupSuite()

В методе `SetupSuite()`:

1. **Инициализируй Gin роутер:**
   ```go
   gin.SetMode(gin.TestMode)
   suite.router = gin.New()
   ```

2. **Создай моки или реальные зависимости:**
   - Создай мок `NormalizationService` или используй реальный сервис с тестовой БД
   - Создай `BaseHandler`
   - Инициализируй `NormalizationHandler` с этими зависимостями

3. **Зарегистрируй все роуты нормализации:**
   - Создай группу `/api/normalization`
   - Зарегистрируй все существующие роуты (17 штук)
   - Зарегистрируй все 7 новых роутов из отчета:
     ```go
     normalizationAPI := suite.router.Group("/api/normalization")
     {
         // Существующие роуты
         normalizationAPI.GET("/status", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationStatus))
         normalizationAPI.POST("/start", httpHandlerToGin(suite.normalizationHandler.HandleStartVersionedNormalization))
         // ... остальные существующие ...
         
         // Новые роуты (7 штук)
         normalizationAPI.POST("/stop", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationStop))
         normalizationAPI.GET("/pipeline/stage-details", httpHandlerToGin(suite.normalizationHandler.HandleStageDetails))
         normalizationAPI.GET("/export", httpHandlerToGin(suite.normalizationHandler.HandleExport))
         normalizationAPI.GET("/config", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationConfig))
         normalizationAPI.PUT("/config", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationConfig))
         normalizationAPI.POST("/config", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationConfig))
         normalizationAPI.GET("/databases", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationDatabases))
         normalizationAPI.GET("/tables", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationTables))
         normalizationAPI.GET("/columns", httpHandlerToGin(suite.normalizationHandler.HandleNormalizationColumns))
     }
     ```

4. **Зарегистрируй Swagger endpoint** (если используется):
   ```go
   suite.router.GET("/swagger/doc.json", ginSwagger.WrapHandler(swaggerFiles.Handler))
   ```

#### 1.3. SetupTest()

В методе `SetupTest()`:
- Создай `httptest.NewRecorder()` для каждого теста (или создавай в самом тесте)
- Инициализируй тестовые данные, если необходимо

### 2. Тестирование 7 новых роутов (Позитивные и негативные сценарии)

Для каждого из 7 новых роутов напиши отдельный тестовый метод с позитивными и негативными сценариями.

#### 2.1. POST /api/normalization/stop

**Тест:** `TestRoute_PostNormalizationStop`

**Позитивные сценарии:**
- Успешная остановка нормализации (даже если ничего не было запущено)
- Проверка структуры ответа: должен содержать `was_running` (boolean)

**Негативные сценарии:**
- Неправильный HTTP метод (GET вместо POST) - должен вернуть 405

**Пример кода:**
```go
func (suite *NormalizationRoutesIntegrationTestSuite) TestRoute_PostNormalizationStop() {
    // Позитивный сценарий: успешная остановка
    req, _ := http.NewRequest("POST", "/api/normalization/stop", nil)
    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)

    assert.Equal(suite.T(), http.StatusOK, w.Code)
    
    var response map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &response)
    assert.NoError(suite.T(), err)
    assert.Contains(suite.T(), response, "was_running")
    assert.Contains(suite.T(), response, "success")
    assert.Contains(suite.T(), response, "message")

    // Негативный сценарий: неправильный метод
    req, _ = http.NewRequest("GET", "/api/normalization/stop", nil)
    w = httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)

    assert.Equal(suite.T(), http.StatusMethodNotAllowed, w.Code)
}
```

#### 2.2. GET /api/normalization/pipeline/stage-details

**Тест:** `TestRoute_GetPipelineStageDetails`

**Позитивные сценарии:**
- Успешное получение деталей этапа (200 OK)
- Проверка структуры JSON ответа:
  - Должен содержать поля: `stage`, `current_step`, `is_running`, `processed`, `success`, `errors`
  - Типы данных должны быть корректными

**Негативные сценарии:**
- Неправильный HTTP метод (POST вместо GET) - должен вернуть 405
- Если сервис недоступен - должен вернуть 503 (если обработчик проверяет это)

#### 2.3. GET /api/normalization/export

**Тест:** `TestRoute_GetNormalizationExport`

**Позитивные сценарии:**
- Экспорт в CSV формате (по умолчанию):
  - Проверка статуса 200
  - Проверка заголовка `Content-Type: text/csv; charset=utf-8`
  - Проверка заголовка `Content-Disposition` с именем файла
  - Проверка наличия UTF-8 BOM (первые 3 байта: `0xEF, 0xBB, 0xBF`)
  
- Экспорт в JSON формате:
  - Проверка статуса 200
  - Проверка заголовка `Content-Type: application/json`
  - Проверка структуры JSON ответа

- Обработка query-параметров:
  - `format=json` - должен вернуть JSON
  - `format=csv` - должен вернуть CSV
  - `category=...` - должен фильтровать по категории
  - `search=...` - должен фильтровать по поиску
  - `kpved_code=...` - должен фильтровать по КПВЭД коду
  - `limit=...` - должен ограничивать количество записей

**Негативные сценарии:**
- Неправильный формат (`format=invalid`) - должен вернуть 400
- Неправильный HTTP метод - должен вернуть 405

#### 2.4. GET /api/normalization/config

**Тест:** `TestRoute_GetNormalizationConfig`

**Позитивные сценарии:**
- Успешное получение конфигурации (200 OK)
- Проверка структуры JSON ответа:
  - Должен содержать поля: `id`, `database_path`, `source_table`, `reference_column`, `code_column`, `name_column`, `created_at`, `updated_at`

**Негативные сценарии:**
- Если сервис недоступен - должен вернуть 503
- Неправильный HTTP метод - должен вернуть 405

#### 2.5. PUT /api/normalization/config

**Тест:** `TestRoute_PutNormalizationConfig`

**Позитивные сценарии:**
- Успешное обновление конфигурации с валидным телом запроса (200 OK)
- Проверка структуры ответа: должен содержать `message` и обновленную `config`

**Негативные сценарии:**
- Невалидное тело запроса (не JSON) - должен вернуть 400
- Отсутствие обязательных полей (`source_table`, `reference_column`, `code_column`, `name_column`) - должен вернуть 400
- Неправильный HTTP метод - должен вернуть 405
- Если сервис недоступен - должен вернуть 503

#### 2.6. POST /api/normalization/config

**Тест:** `TestRoute_PostNormalizationConfig`

**Позитивные и негативные сценарии:** Аналогично PUT (см. раздел 2.5)

#### 2.7. GET /api/normalization/databases

**Тест:** `TestRoute_GetNormalizationDatabases`

**Позитивные сценарии:**
- Успешное получение списка баз данных (200 OK)
- Проверка, что ответ - это массив объектов
- Проверка структуры каждого объекта: должен содержать `name`, `path`, `size`

**Негативные сценарии:**
- Неправильный HTTP метод - должен вернуть 405

#### 2.8. GET /api/normalization/tables

**Тест:** `TestRoute_GetNormalizationTables`

**Позитивные сценарии:**
- Успешное получение списка таблиц с query-параметром `database` (200 OK)
- Проверка, что ответ - это массив объектов
- Проверка структуры каждого объекта: должен содержать `name`, `count`

**Негативные сценарии:**
- Отсутствие параметра `database` (если требуется) - проверить поведение
- Неправильный HTTP метод - должен вернуть 405
- Некорректный путь к БД - должен вернуть 500

#### 2.9. GET /api/normalization/columns

**Тест:** `TestRoute_GetNormalizationColumns`

**Позитивные сценарии:**
- Успешное получение списка колонок с параметрами `database` и `table` (200 OK)
- Проверка, что ответ - это массив объектов
- Проверка структуры каждого объекта: должен содержать `name`, `type`, `nullable`, `primary`, `default`

**Негативные сценарии:**
- Отсутствие обязательного параметра `table` - должен вернуть 400
- Некорректное имя таблицы (SQL injection попытка) - должен вернуть 400
- Неправильный HTTP метод - должен вернуть 405
- Некорректный путь к БД - должен вернуть 500

### 3. Тестирование Swagger-документации

**Тест:** `TestSwagger_NewRoutesDocumented`

Этот тест проверяет, что все новые роуты появились в Swagger-спецификации.

**Шаги:**
1. Отправь GET-запрос на эндпоинт Swagger JSON (обычно `/swagger/doc.json` или `/swagger/index.html`)
2. Распарси полученный JSON (если доступен)
3. Проверь, что в `paths` присутствуют все новые пути:
   - `/api/normalization/stop` с методом `post`
   - `/api/normalization/pipeline/stage-details` с методом `get`
   - `/api/normalization/export` с методом `get`
   - `/api/normalization/config` с методами `get`, `put`, `post`
   - `/api/normalization/databases` с методом `get`
   - `/api/normalization/tables` с методом `get`
   - `/api/normalization/columns` с методом `get`

4. Для каждого пути проверь наличие правильных HTTP-методов

**Пример структуры:**
```go
func (suite *NormalizationRoutesIntegrationTestSuite) TestSwagger_NewRoutesDocumented() {
    // Отправляем запрос на Swagger endpoint
    req, _ := http.NewRequest("GET", "/swagger/doc.json", nil)
    w := httptest.NewRecorder()
    suite.router.ServeHTTP(w, req)

    assert.Equal(suite.T(), http.StatusOK, w.Code)

    var swaggerDoc map[string]interface{}
    err := json.Unmarshal(w.Body.Bytes(), &swaggerDoc)
    assert.NoError(suite.T(), err)

    paths, ok := swaggerDoc["paths"].(map[string]interface{})
    assert.True(suite.T(), ok, "Swagger doc should contain paths")

    // Проверяем наличие новых роутов
    newRoutes := []struct {
        path   string
        method string
    }{
        {"/api/normalization/stop", "post"},
        {"/api/normalization/pipeline/stage-details", "get"},
        {"/api/normalization/export", "get"},
        {"/api/normalization/config", "get"},
        {"/api/normalization/config", "put"},
        {"/api/normalization/config", "post"},
        {"/api/normalization/databases", "get"},
        {"/api/normalization/tables", "get"},
        {"/api/normalization/columns", "get"},
    }

    for _, route := range newRoutes {
        pathInfo, exists := paths[route.path]
        assert.True(suite.T(), exists, "Route %s should be documented", route.path)
        
        if exists {
            pathMap := pathInfo.(map[string]interface{})
            methodInfo, methodExists := pathMap[route.method]
            assert.True(suite.T(), methodExists, "Route %s should have %s method", route.path, route.method)
            
            if methodExists {
                methodMap := methodInfo.(map[string]interface{})
                assert.Contains(suite.T(), methodMap, "summary", "Route %s %s should have summary", route.path, route.method)
                assert.Contains(suite.T(), methodMap, "tags", "Route %s %s should have tags", route.path, route.method)
            }
        }
    }
}
```

**Примечание:** Если Swagger endpoint недоступен или не настроен, тест должен быть пропущен с соответствующим сообщением.

### 4. Регрессионное тестирование существующих роутов

Напиши 2-3 теста для существующих роутов, чтобы убедиться, что они не были сломаны после добавления новых.

#### 4.1. TestRoute_ExistingStartNormalization

**Тест:** Проверь, что `POST /api/normalization/start` все еще работает.

**Проверки:**
- Успешный ответ (200 или 201)
- Структура ответа содержит `session_id` или аналогичное поле
- Неправильный метод возвращает 405

#### 4.2. TestRoute_ExistingNormalizationStatus

**Тест:** Проверь, что `GET /api/normalization/status` возвращает корректный статус.

**Проверки:**
- Успешный ответ (200)
- Структура ответа содержит поля: `is_running`, `progress`, `processed`, `total`
- Типы данных корректные

#### 4.3. TestRoute_ExistingNormalizationHistory

**Тест:** Проверь, что `GET /api/normalization/history` работает.

**Проверки:**
- Успешный ответ с query-параметром `session_id` (200)
- Отсутствие `session_id` возвращает 400
- Структура ответа корректна

## Технические детали

### Фреймворк тестирования:
- **testify/suite** - для организации тестов
- **testify/assert** - для assertions
- **net/http/httptest** - для HTTP-тестирования
- **encoding/json** - для парсинга JSON ответов

### Имя файла:
`normalization_routes_integration_test.go`

### Пакет:
`server_test` или `integration` (в зависимости от структуры проекта)

### Импорты (пример):
```go
import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    
    "httpserver/server/handlers"
    "httpserver/server/services"
    // ... другие необходимые импорты
)
```

### Структура файла:

```go
package server_test // или integration

import (
    // ... импорты ...
)

type NormalizationRoutesIntegrationTestSuite struct {
    suite.Suite
    // ... поля ...
}

func (suite *NormalizationRoutesIntegrationTestSuite) SetupSuite() {
    // ... настройка ...
}

func (suite *NormalizationRoutesIntegrationTestSuite) SetupTest() {
    // ... настройка для каждого теста ...
}

// Тесты для новых роутов
func (suite *NormalizationRoutesIntegrationTestSuite) TestRoute_PostNormalizationStop() {
    // ...
}

func (suite *NormalizationRoutesIntegrationTestSuite) TestRoute_GetPipelineStageDetails() {
    // ...
}

// ... остальные тесты для новых роутов ...

// Тест Swagger
func (suite *NormalizationRoutesIntegrationTestSuite) TestSwagger_NewRoutesDocumented() {
    // ...
}

// Регрессионные тесты
func (suite *NormalizationRoutesIntegrationTestSuite) TestRoute_ExistingStartNormalization() {
    // ...
}

func (suite *NormalizationRoutesIntegrationTestSuite) TestRoute_ExistingNormalizationStatus() {
    // ...
}

func (suite *NormalizationRoutesIntegrationTestSuite) TestRoute_ExistingNormalizationHistory() {
    // ...
}

// Запуск тестов
func TestNormalizationRoutesIntegrationSuite(t *testing.T) {
    suite.Run(t, new(NormalizationRoutesIntegrationTestSuite))
}
```

## Дополнительные требования

### Комментарии:
- Добавь четкие комментарии, объясняющие что делает каждый тест
- Комментируй неочевидные проверки
- Объясни, почему используются моки или реальные зависимости

### Обработка ошибок:
- Все тесты должны корректно обрабатывать ошибки
- Используй `assert.NoError()` для проверки отсутствия ошибок
- Используй `assert.Error()` для проверки наличия ошибок в негативных сценариях

### Изоляция тестов:
- Каждый тест должен быть независимым
- Не используй общее состояние между тестами
- Используй `SetupTest()` для подготовки данных для каждого теста

### Покрытие:
- Тесты должны покрывать как позитивные, так и негативные сценарии
- Проверяй валидацию входных данных
- Проверяй обработку ошибок

## Ожидаемый результат

Предоставь один готовый к запуску Go-файл `normalization_routes_integration_test.go`, содержащий:

1. ✅ Структуру `NormalizationRoutesIntegrationTestSuite` с методами `SetupSuite` и `SetupTest`
2. ✅ Набор тестовых методов для проверки 7 новых роутов (позитивные и негативные сценарии)
3. ✅ Специальный тест для проверки наличия новых роутов в Swagger-спецификации
4. ✅ Несколько регрессионных тестов для существующих роутов
5. ✅ Необходимые импорты
6. ✅ Хорошо прокомментированный код, следующий лучшим практикам Go

Код должен быть готов к запуску командой:
```bash
go test -v ./server_test -run TestNormalizationRoutesIntegrationSuite
```
или
```bash
go test -v ./integration -run TestNormalizationRoutesIntegrationSuite
```

---

## Инструкция по использованию:

1. Скопируй промпт выше (текст между "---" и "---")
2. Вставь в новый чат с AI
3. AI автоматически сгенерирует полный файл интеграционных тестов

## Что будет создано:

- `normalization_routes_integration_test.go` - полный файл с интеграционными тестами
- Тесты для всех 7 новых роутов (позитивные и негативные сценарии)
- Тест для проверки Swagger-документации
- Регрессионные тесты для существующих роутов
- Готовый к запуску код с необходимыми зависимостями

