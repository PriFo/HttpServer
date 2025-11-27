# Промпт для генерации интеграционных тестов CRUD API уведомлений

Скопируй этот промпт в новый чат для автоматической генерации интеграционных тестов, проверяющих все CRUD операции API уведомлений с обязательной проверкой персистентности данных:

---

## Роль

Ты — Senior Go Developer и QA Automation Engineer. Твоя задача — создать исчерпывающий набор интеграционных тестов для проверки всех CRUD операций API уведомлений с обязательной проверкой персистентности данных в базе данных.

## Контекст

**Проект:** Go-приложение с использованием фреймворка Gin и SQLite базы данных

**API уведомлений:** `/api/notifications`

**Структура БД:** Таблица `notifications` в `service_db` со следующими полями:
- `id` (INTEGER PRIMARY KEY AUTOINCREMENT)
- `type` (TEXT NOT NULL) - тип уведомления: 'info', 'success', 'warning', 'error'
- `title` (TEXT NOT NULL) - заголовок уведомления
- `message` (TEXT NOT NULL) - текст уведомления
- `timestamp` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP) - время создания
- `read` (BOOLEAN DEFAULT FALSE) - флаг прочитанности
- `client_id` (INTEGER) - ID клиента (опционально)
- `project_id` (INTEGER) - ID проекта (опционально)
- `metadata_json` (TEXT) - JSON с дополнительными данными
- `created_at` (TIMESTAMP DEFAULT CURRENT_TIMESTAMP)

**Обработчики:**
- `HandleGetNotifications` - GET /api/notifications
- `HandleGetUnreadCount` - GET /api/notifications/unread-count
- `HandleMarkAsRead` - POST /api/notifications/{id}/read
- `HandleMarkAllAsRead` - POST /api/notifications/read-all
- `HandleDeleteNotification` - DELETE /api/notifications/{id}

**Примечание:** Создание уведомлений может происходить через POST /api/notifications или через сервис напрямую. Если обработчика создания нет, создай его или используй прямой вызов сервиса для тестов.

## Основная задача

Сгенерируй один Go-файл `notifications_crud_integration_test.go`, который будет содержать полный набор интеграционных тестов для проверки всех CRUD операций API уведомлений. Тесты должны использовать `testify/suite` для организации и `httptest` для HTTP-запросов.

**Критически важно:** Каждый тест должен включать прямые SQL-запросы к тестовой БД для проверки персистентности данных после каждой операции.

## Детальные требования к тестам

### 1. Настройка тестового окружения

#### 1.1. Структура TestSuite

Создай структуру `NotificationsCRUDIntegrationTestSuite` со следующими полями:

```go
type NotificationsCRUDIntegrationTestSuite struct {
    suite.Suite
    serviceDB            *database.ServiceDB
    tx                   *sql.Tx
    router               *gin.Engine
    notificationHandler  *handlers.NotificationHandler
    notificationService  *services.NotificationService
    baseHandler          *handlers.BaseHandler
}
```

#### 1.2. SetupSuite()

В методе `SetupSuite()`:

1. **Инициализируй тестовую БД:**
   ```go
   suite.serviceDB, err = database.NewServiceDB(":memory:")
   assert.NoError(suite.T(), err)
   defer suite.serviceDB.Close()
   ```

2. **Выполни миграции:**
   - Убедись, что таблица `notifications` создана
   - Можно использовать `database.InitServiceSchema()` или выполнить SQL напрямую

3. **Инициализируй зависимости:**
   - Создай `BaseHandler`
   - Создай `NotificationService` с `serviceDB`
   - Создай `NotificationHandler` с сервисом и baseHandler

4. **Инициализируй Gin роутер:**
   ```go
   gin.SetMode(gin.TestMode)
   suite.router = gin.New()
   api := suite.router.Group("/api")
   notificationsAPI := api.Group("/notifications")
   {
       notificationsAPI.GET("", httpHandlerToGin(suite.notificationHandler.HandleGetNotifications))
       notificationsAPI.GET("/unread-count", httpHandlerToGin(suite.notificationHandler.HandleGetUnreadCount))
       notificationsAPI.POST("/:id/read", httpHandlerToGin(suite.notificationHandler.HandleMarkAsRead))
       notificationsAPI.POST("/read-all", httpHandlerToGin(suite.notificationHandler.HandleMarkAllAsRead))
       notificationsAPI.DELETE("/:id", httpHandlerToGin(suite.notificationHandler.HandleDeleteNotification))
       // Если есть обработчик создания:
       notificationsAPI.POST("", httpHandlerToGin(suite.notificationHandler.HandleCreateNotification))
   }
   ```

#### 1.3. SetupTest() и TearDownTest()

**SetupTest():**
- Начинай транзакцию: `suite.tx, err = suite.serviceDB.DB.Begin()`
- Используй транзакцию для всех операций в тесте (для изоляции)

**TearDownTest():**
- Откатывай транзакцию: `suite.tx.Rollback()`
- Это обеспечит чистое состояние БД для каждого теста

### 2. Тестирование CRUD-операций и персистентности

Для каждого эндпоинта напиши отдельный тестовый метод с проверкой персистентности.

#### 2.1. POST /api/notifications (Создание)

**Тест:** `TestNotification_Create_Success`

**Проверки:**

1. **HTTP-уровень:**
   - Отправь валидный JSON для создания уведомления:
     ```json
     {
       "type": "info",
       "title": "Test Notification",
       "message": "This is a test notification",
       "client_id": 1,
       "project_id": 1,
       "metadata_json": "{\"key\":\"value\"}"
     }
     ```
   - Проверь, что ответ имеет статус `201 Created` (или `200 OK`, если используется другой код)
   - Проверь, что ответ содержит `id` созданного уведомления

2. **Уровень базы данных (критично!):**
   - Сделай прямой SQL-запрос к тестовой БД в рамках транзакции:
     ```go
     var id int
     var notificationType, title, message string
     var read bool
     var clientID, projectID sql.NullInt64
     var metadataJSON sql.NullString
     var timestamp, createdAt time.Time
     
     err := suite.tx.QueryRow(`
         SELECT id, type, title, message, read, client_id, project_id, 
                metadata_json, timestamp, created_at
         FROM notifications 
         WHERE id = ?
     `, createdID).Scan(&id, &notificationType, &title, &message, &read, 
         &clientID, &projectID, &metadataJSON, &timestamp, &createdAt)
     
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), "info", notificationType)
     assert.Equal(suite.T(), "Test Notification", title)
     assert.Equal(suite.T(), "This is a test notification", message)
     assert.False(suite.T(), read, "Notification should be unread by default")
     assert.True(suite.T(), clientID.Valid)
     assert.Equal(suite.T(), int64(1), clientID.Int64)
     assert.True(suite.T(), projectID.Valid)
     assert.Equal(suite.T(), int64(1), projectID.Int64)
     assert.True(suite.T(), metadataJSON.Valid)
     assert.NotZero(suite.T(), timestamp)
     assert.NotZero(suite.T(), createdAt)
     ```
   - Убедись, что все поля сохранены корректно
   - Проверь, что `read=false` по умолчанию
   - Проверь, что `timestamp` и `created_at` установлены

**Тест:** `TestNotification_Create_InvalidData`

**Проверки:**

1. **Отсутствующие обязательные поля:**
   - Отправь JSON без поля `title`
   - Ожидай статус `400 Bad Request`
   - Проверь, что в БД ничего не добавилось:
     ```go
     var count int
     err := suite.tx.QueryRow("SELECT COUNT(*) FROM notifications").Scan(&count)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 0, count, "No notifications should be created with invalid data")
     ```

2. **Невалидный JSON:**
   - Отправь невалидный JSON (например, `{"title":}`)
   - Ожидай статус `400 Bad Request`
   - Проверь, что в БД ничего не добавилось

3. **Невалидный тип уведомления:**
   - Отправь JSON с `type: "invalid"`
   - Ожидай статус `400 Bad Request` (если есть валидация)
   - Проверь, что в БД ничего не добавилось

#### 2.2. GET /api/notifications (Чтение/Фильтрация)

**Тест:** `TestNotification_GetAll_Success`

**Проверки:**

1. **Подготовка данных:**
   - В рамках тестовой транзакции создай несколько уведомлений напрямую через SQL:
     ```go
     // Создаем уведомления для разных client_id
     suite.tx.Exec(`
         INSERT INTO notifications (type, title, message, client_id, project_id, read)
         VALUES 
         ('info', 'Notification 1', 'Message 1', 123, 1, 0),
         ('warning', 'Notification 2', 'Message 2', 123, 1, 1),
         ('error', 'Notification 3', 'Message 3', 456, 2, 0),
         ('success', 'Notification 4', 'Message 4', 456, 2, 0)
     `)
     ```

2. **GET без фильтров:**
   - Отправь `GET /api/notifications` без query-параметров
   - Проверь статус `200 OK`
   - Проверь, что ответ содержит все созданные уведомления
   - Проверь структуру ответа: `{"notifications": [...], "count": 4}`

3. **GET с фильтром client_id:**
   - Отправь `GET /api/notifications?client_id=123`
   - Проверь статус `200 OK`
   - Проверь, что в ответе только уведомления для `client_id=123` (2 штуки)
   - Сделай прямой SQL-запрос для сравнения:
     ```go
     var count int
     err := suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications WHERE client_id = 123
     `).Scan(&count)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 2, count)
     ```

4. **GET с фильтром read=false:**
   - Отправь `GET /api/notifications?unread_only=true` или `GET /api/notifications?read=false`
   - Проверь статус `200 OK`
   - Проверь, что в ответе только непрочитанные уведомления (3 штуки)
   - Сделай прямой SQL-запрос для сравнения:
     ```go
     var count int
     err := suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications WHERE read = 0
     `).Scan(&count)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 3, count)
     ```

5. **GET с комбинированными фильтрами:**
   - Отправь `GET /api/notifications?client_id=123&unread_only=true`
   - Проверь, что возвращаются только непрочитанные уведомления для client_id=123 (1 штука)

#### 2.3. PUT /api/notifications/{id}/read (Обновление)

**Тест:** `TestNotification_MarkAsRead_Success`

**Проверки:**

1. **Подготовка данных:**
   - Создай уведомление напрямую в БД с `read=false`:
     ```go
     result, err := suite.tx.Exec(`
         INSERT INTO notifications (type, title, message, read)
         VALUES ('info', 'Test', 'Test message', 0)
     `)
     assert.NoError(suite.T(), err)
     notificationID, _ := result.LastInsertId()
     ```

2. **HTTP-уровень:**
   - Отправь `POST /api/notifications/{id}/read` (обрати внимание: в коде используется POST, не PUT)
   - Проверь статус `200 OK`
   - Проверь структуру ответа: `{"success": true}`

3. **Уровень базы данных (критично!):**
   - Сделай прямой SQL-запрос, чтобы убедиться, что поле `read` изменилось на `true`:
     ```go
     var read bool
     err := suite.tx.QueryRow(`
         SELECT read FROM notifications WHERE id = ?
     `, notificationID).Scan(&read)
     assert.NoError(suite.T(), err)
     assert.True(suite.T(), read, "Notification should be marked as read")
     ```

**Тест:** `TestNotification_MarkAsRead_NotFound`

**Проверки:**

1. **Несуществующий ID:**
   - Отправь `POST /api/notifications/99999/read`
   - Ожидай статус `404 Not Found` (или `500 Internal Server Error`, если обработчик не проверяет существование)
   - Проверь, что в БД ничего не изменилось:
     ```go
     var count int
     err := suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications WHERE read = 1
     `).Scan(&count)
     assert.NoError(suite.T(), err)
     // count должен остаться прежним
     ```

2. **Невалидный ID:**
   - Отправь `POST /api/notifications/invalid/read`
   - Ожидай статус `400 Bad Request`

#### 2.4. POST /api/notifications/read-all (Массовое обновление)

**Тест:** `TestNotification_MarkAllAsRead_Success`

**Проверки:**

1. **Подготовка данных:**
   - Создай несколько непрочитанных уведомлений для `client_id=111`:
     ```go
     suite.tx.Exec(`
         INSERT INTO notifications (type, title, message, client_id, read)
         VALUES 
         ('info', 'Notif 1', 'Message 1', 111, 0),
         ('warning', 'Notif 2', 'Message 2', 111, 0),
         ('error', 'Notif 3', 'Message 3', 111, 0)
     `)
     ```
   - Создай несколько непрочитанных уведомлений для `client_id=222`:
     ```go
     suite.tx.Exec(`
         INSERT INTO notifications (type, title, message, client_id, read)
         VALUES 
         ('info', 'Notif 4', 'Message 4', 222, 0),
         ('success', 'Notif 5', 'Message 5', 222, 0)
     `)
     ```

2. **HTTP-уровень:**
   - Отправь `POST /api/notifications/read-all?client_id=111`
   - Проверь статус `200 OK`
   - Проверь структуру ответа: `{"success": true}`

3. **Уровень базы данных (критично!):**
   - Сделай прямой SQL-запрос, чтобы убедиться, что только уведомления для `client_id=111` пометились как прочитанные:
     ```go
     // Проверяем, что уведомления для client_id=111 прочитаны
     var readCount111 int
     err := suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications 
         WHERE client_id = 111 AND read = 1
     `).Scan(&readCount111)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 3, readCount111, "All notifications for client_id=111 should be marked as read")
     
     // Проверяем, что уведомления для client_id=222 остались непрочитанными
     var unreadCount222 int
     err = suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications 
         WHERE client_id = 222 AND read = 0
     `).Scan(&unreadCount222)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 2, unreadCount222, "Notifications for client_id=222 should remain unread")
     ```

4. **Без фильтра client_id:**
   - Создай новые непрочитанные уведомления
   - Отправь `POST /api/notifications/read-all` без параметров
   - Проверь, что все уведомления пометились как прочитанные

#### 2.5. GET /api/notifications/unread-count (Агрегация)

**Тест:** `TestNotification_GetUnreadCount_Success`

**Проверки:**

1. **Подготовка данных:**
   - Создай смесь из прочитанных и непрочитанных уведомлений:
     ```go
     suite.tx.Exec(`
         INSERT INTO notifications (type, title, message, read, client_id)
         VALUES 
         ('info', 'Unread 1', 'Message 1', 0, 100),
         ('warning', 'Read 1', 'Message 2', 1, 100),
         ('error', 'Unread 2', 'Message 3', 0, 100),
         ('success', 'Unread 3', 'Message 4', 0, 200),
         ('info', 'Read 2', 'Message 5', 1, 200)
     `)
     ```

2. **HTTP-уровень:**
   - Отправь `GET /api/notifications/unread-count`
   - Проверь статус `200 OK`
   - Проверь структуру ответа: `{"count": 3}` (3 непрочитанных)

3. **Уровень базы данных (критично!):**
   - Сделай прямой SQL-запрос для сравнения:
     ```go
     var dbCount int
     err := suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications WHERE read = 0
     `).Scan(&dbCount)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 3, dbCount)
     
     // Сравниваем с ответом API
     var response map[string]interface{}
     err = json.Unmarshal(w.Body.Bytes(), &response)
     assert.NoError(suite.T(), err)
     apiCount := int(response["count"].(float64))
     assert.Equal(suite.T(), dbCount, apiCount, "API count should match database count")
     ```

4. **С фильтром client_id:**
   - Отправь `GET /api/notifications/unread-count?client_id=100`
   - Проверь, что возвращается количество непрочитанных только для этого клиента (2 штуки)
   - Сравни с прямым SQL-запросом:
     ```go
     var dbCount int
     err := suite.tx.QueryRow(`
         SELECT COUNT(*) FROM notifications 
         WHERE read = 0 AND client_id = 100
     `).Scan(&dbCount)
     assert.NoError(suite.T(), err)
     assert.Equal(suite.T(), 2, dbCount)
     ```

#### 2.6. DELETE /api/notifications/{id} (Удаление)

**Тест:** `TestNotification_Delete_Success`

**Проверки:**

1. **Подготовка данных:**
   - Создай уведомление напрямую в БД:
     ```go
     result, err := suite.tx.Exec(`
         INSERT INTO notifications (type, title, message)
         VALUES ('info', 'To Delete', 'This will be deleted')
     `)
     assert.NoError(suite.T(), err)
     notificationID, _ := result.LastInsertId()
     ```

2. **HTTP-уровень:**
   - Отправь `DELETE /api/notifications/{id}`
   - Проверь статус `200 OK` или `204 No Content`
   - Проверь структуру ответа (если есть): `{"success": true}`

3. **Уровень базы данных (критично!):**
   - Сделай прямой SQL-запрос, чтобы убедиться, что запись была удалена:
     ```go
     var id int
     err := suite.tx.QueryRow(`
         SELECT id FROM notifications WHERE id = ?
     `, notificationID).Scan(&id)
     assert.Error(suite.T(), err)
     assert.Equal(suite.T(), sql.ErrNoRows, err, "Notification should be deleted from database")
     ```

4. **Проверка количества:**
   - Перед удалением запомни количество записей
   - После удаления проверь, что количество уменьшилось на 1:
     ```go
     var countBefore int
     suite.tx.QueryRow("SELECT COUNT(*) FROM notifications").Scan(&countBefore)
     
     // Выполни DELETE запрос
     
     var countAfter int
     suite.tx.QueryRow("SELECT COUNT(*) FROM notifications").Scan(&countAfter)
     assert.Equal(suite.T(), countBefore-1, countAfter, "Count should decrease by 1 after deletion")
     ```

**Тест:** `TestNotification_Delete_NotFound`

**Проверки:**

1. **Несуществующий ID:**
   - Отправь `DELETE /api/notifications/99999`
   - Ожидай статус `404 Not Found` (или `500`, если обработчик не проверяет)
   - Проверь, что количество записей в БД не изменилось

2. **Невалидный ID:**
   - Отправь `DELETE /api/notifications/invalid`
   - Ожидай статус `400 Bad Request`

### 3. Дополнительные тесты

#### 3.1. Тест на изоляцию транзакций

**Тест:** `TestNotification_TransactionIsolation`

**Проверки:**

- Создай уведомление в одном тесте
- В следующем тесте проверь, что это уведомление отсутствует (благодаря Rollback в TearDownTest)

#### 3.2. Тест на валидацию типов

**Тест:** `TestNotification_TypeValidation`

**Проверки:**

- Попробуй создать уведомление с невалидным типом
- Проверь, что БД отклоняет запрос (CHECK constraint) или обработчик валидирует

## Технические детали

### Фреймворк тестирования:
- **testify/suite** - для организации тестов
- **testify/assert** - для assertions
- **net/http/httptest** - для HTTP-тестирования
- **database/sql** - для прямых SQL-запросов
- **encoding/json** - для парсинга JSON ответов

### Имя файла:
`notifications_crud_integration_test.go`

### Пакет:
`server_test` или `integration` (в зависимости от структуры проекта)

### Импорты (пример):
```go
import (
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/gin-gonic/gin"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
    
    "httpserver/database"
    "httpserver/server/handlers"
    "httpserver/server/services"
    _ "github.com/mattn/go-sqlite3"
)
```

### Структура файла:

```go
package server_test // или integration

import (
    // ... импорты ...
)

type NotificationsCRUDIntegrationTestSuite struct {
    suite.Suite
    serviceDB            *database.ServiceDB
    tx                   *sql.Tx
    router               *gin.Engine
    notificationHandler  *handlers.NotificationHandler
    notificationService  *services.NotificationService
    baseHandler          *handlers.BaseHandler
}

func (suite *NotificationsCRUDIntegrationTestSuite) SetupSuite() {
    // ... настройка ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) SetupTest() {
    // Начало транзакции
    var err error
    suite.tx, err = suite.serviceDB.DB.Begin()
    assert.NoError(suite.T(), err)
}

func (suite *NotificationsCRUDIntegrationTestSuite) TearDownTest() {
    // Откат транзакции
    if suite.tx != nil {
        suite.tx.Rollback()
    }
}

// Тесты для CRUD операций
func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_Create_Success() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_Create_InvalidData() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_GetAll_Success() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_MarkAsRead_Success() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_MarkAsRead_NotFound() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_MarkAllAsRead_Success() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_GetUnreadCount_Success() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_Delete_Success() {
    // ...
}

func (suite *NotificationsCRUDIntegrationTestSuite) TestNotification_Delete_NotFound() {
    // ...
}

// Запуск тестов
func TestNotificationsCRUDIntegrationSuite(t *testing.T) {
    suite.Run(t, new(NotificationsCRUDIntegrationTestSuite))
}
```

## Дополнительные требования

### Комментарии:
- Добавь четкие комментарии, объясняющие что делает каждый тест
- Комментируй неочевидные проверки
- Объясни, почему используются транзакции для изоляции

### Обработка ошибок:
- Все тесты должны корректно обрабатывать ошибки
- Используй `assert.NoError()` для проверки отсутствия ошибок
- Используй `assert.Error()` для проверки наличия ошибок в негативных сценариях
- Используй `assert.Equal(suite.T(), sql.ErrNoRows, err)` для проверки отсутствия записей

### Изоляция тестов:
- Каждый тест должен быть независимым благодаря транзакциям
- Используй `SetupTest()` для начала транзакции
- Используй `TearDownTest()` для отката транзакции
- Не используй общее состояние между тестами

### Покрытие:
- Тесты должны покрывать как позитивные, так и негативные сценарии
- Проверяй валидацию входных данных
- Проверяй обработку ошибок
- Проверяй фильтрацию и агрегацию

### Критически важно:
- **Каждый тест должен включать прямые SQL-запросы для проверки персистентности**
- Не полагайся только на HTTP-ответы
- Проверяй реальное состояние БД после каждой операции

## Ожидаемый результат

Предоставь один готовый к запуску Go-файл `notifications_crud_integration_test.go`, содержащий:

1. ✅ Структуру `NotificationsCRUDIntegrationTestSuite` с методами `SetupSuite`, `SetupTest`, `TearDownTest`
2. ✅ Набор тестовых методов для проверки всех CRUD операций:
   - `TestNotification_Create_Success` - создание с проверкой БД
   - `TestNotification_Create_InvalidData` - невалидные данные
   - `TestNotification_GetAll_Success` - чтение с фильтрацией
   - `TestNotification_MarkAsRead_Success` - обновление с проверкой БД
   - `TestNotification_MarkAsRead_NotFound` - несуществующий ID
   - `TestNotification_MarkAllAsRead_Success` - массовое обновление с проверкой БД
   - `TestNotification_GetUnreadCount_Success` - агрегация с проверкой БД
   - `TestNotification_Delete_Success` - удаление с проверкой БД
   - `TestNotification_Delete_NotFound` - несуществующий ID
3. ✅ Прямые SQL-запросы для проверки персистентности в каждом тесте
4. ✅ Необходимые импорты
5. ✅ Хорошо прокомментированный код, следующий лучшим практикам Go

Код должен быть готов к запуску командой:
```bash
go test -v ./server_test -run TestNotificationsCRUDIntegrationSuite
```
или
```bash
go test -v ./integration -run TestNotificationsCRUDIntegrationSuite
```

---

## Инструкция по использованию:

1. Скопируй промпт выше (текст между "---" и "---")
2. Вставь в новый чат с AI
3. AI автоматически сгенерирует полный файл интеграционных тестов

## Что будет создано:

- `notifications_crud_integration_test.go` - полный файл с интеграционными тестами
- Тесты для всех CRUD операций (Create, Read, Update, Delete)
- Тесты для фильтрации и агрегации
- Прямые SQL-запросы для проверки персистентности в каждом тесте
- Позитивные и негативные сценарии
- Готовый к запуску код с необходимыми зависимостями

