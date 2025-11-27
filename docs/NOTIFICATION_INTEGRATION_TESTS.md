# Notification System Integration Tests

## Обзор

Создан полный набор интеграционных тестов для системы уведомлений с персистентностью в базе данных. Тесты проверяют все CRUD-операции, фильтрацию, fallback-логику и синхронизацию между БД и памятью.

## Файл тестов

**`server/handlers/notification_service_integration_test.go`**

## Структура тестов

### Test Suite

Используется `testify/suite` для организации тестов:

- **NotificationIntegrationTestSuite** - основная структура тестового набора
- **SetupSuite()** - инициализация тестового окружения (БД, сервисы, роутер)
- **SetupTest()** - очистка данных перед каждым тестом
- **TearDownTest()** - очистка после каждого теста
- **TearDownSuite()** - закрытие соединений после всех тестов

### Тестовая база данных

- Используется in-memory SQLite (`:memory:`)
- Автоматическое создание схемы через `InitServiceSchema`
- Прямой доступ к БД через `ServiceDB.GetDB()` для проверки состояния

## Покрытие тестами

### 1. CRUD операции

#### Создание уведомлений
- ✅ **TestNotification_Create_Success** - успешное создание с валидными данными
  - Проверка HTTP статуса 201 Created
  - Проверка наличия ID в ответе
  - Прямая проверка записи в БД (все поля)
- ✅ **TestNotification_Create_InvalidData** - создание с невалидными данными
  - Отсутствие обязательных полей (title)
  - Невалидный JSON

#### Чтение уведомлений
- ✅ **TestNotification_GetAll_Success** - получение всех уведомлений
  - Проверка возврата всех созданных уведомлений
- ✅ **TestNotification_GetWithFilters** - фильтрация
  - По `client_id`
  - По `unread_only`

#### Обновление уведомлений
- ✅ **TestNotification_MarkAsRead_Success** - пометка как прочитанного
  - Проверка HTTP статуса 200 OK
  - Прямая проверка изменения поля `read` в БД
- ✅ **TestNotification_MarkAsRead_NotFound** - несуществующее уведомление
  - Проверка обработки ошибки
- ✅ **TestNotification_MarkAllAsRead_Success** - массовая пометка
  - Фильтрация по `client_id`
  - Проверка, что только нужные уведомления помечены

#### Удаление уведомлений
- ✅ **TestNotification_Delete_Success** - успешное удаление
  - Проверка HTTP статуса 200/204
  - Прямая проверка удаления из БД
- ✅ **TestNotification_Delete_NotFound** - несуществующее уведомление

### 2. Агрегация

- ✅ **TestNotification_GetUnreadCount_Success** - подсчет непрочитанных
  - Проверка корректности подсчета
  - Сравнение с прямым SQL-запросом

### 3. Синхронизация и персистентность

- ✅ **TestNotification_SyncBetweenDBAndService** - синхронизация БД и сервиса
  - Создание в БД → проверка через API
  - Создание через API → проверка в БД
- ✅ **TestNotification_PersistenceAcrossRestarts** - персистентность при перезапуске
  - Создание уведомления
  - Реинициализация сервиса
  - Проверка доступности уведомления

### 4. Fallback-логика

- ✅ **TestNotification_FallbackToMemoryOnDBFailure** - fallback на память
  - Создание сервиса без БД
  - Проверка, что сервис не падает
  - Проверка работы через память

## Технические детали

### Используемые технологии

- **testify/suite** - организация тестов
- **testify/assert** - assertions
- **net/http/httptest** - HTTP тестирование
- **database/sql** - прямые SQL-запросы для проверки БД
- **gin-gonic/gin** - HTTP роутер
- **SQLite :memory:** - изолированная тестовая БД

### Адаптер для Gin

Создан адаптер `httpHandlerToGin` для преобразования `http.HandlerFunc` в `gin.HandlerFunc`:

```go
func httpHandlerToGin(handler http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := c.Request
		// Прокидываем path-параметры в контекст
		if len(c.Params) > 0 {
			ctx := req.Context()
			for _, param := range c.Params {
				ctx = context.WithValue(ctx, param.Key, param.Value)
			}
			req = req.WithContext(ctx)
		}
		handler(c.Writer, req)
	}
}
```

### Изоляция тестов

- Каждый тест выполняется с чистой БД (очистка в `SetupTest`)
- Использование транзакций не требуется, так как используется in-memory БД
- Полная изоляция через пересоздание данных перед каждым тестом

## Запуск тестов

```bash
# Запуск всех тестов уведомлений
go test -v ./server/handlers -run TestNotificationSuite

# Запуск конкретного теста
go test -v ./server/handlers -run TestNotificationSuite/TestNotification_Create_Success

# С покрытием
go test -v -cover ./server/handlers -run TestNotificationSuite
```

## Проверяемые сценарии

### Позитивные сценарии
- ✅ Создание уведомления с полными данными
- ✅ Получение всех уведомлений
- ✅ Фильтрация по различным параметрам
- ✅ Пометка как прочитанного
- ✅ Массовая пометка с фильтрацией
- ✅ Подсчет непрочитанных
- ✅ Удаление уведомления

### Негативные сценарии
- ✅ Создание с невалидными данными
- ✅ Пометка несуществующего уведомления
- ✅ Удаление несуществующего уведомления

### Граничные случаи
- ✅ Синхронизация между БД и памятью
- ✅ Персистентность при перезапуске
- ✅ Fallback на память при недоступности БД

## Статус

✅ **Все тесты реализованы и готовы к запуску**

Тесты покрывают:
- Все CRUD-операции
- Фильтрацию и агрегацию
- Синхронизацию и персистентность
- Fallback-логику

## Следующие шаги

1. Запустить тесты и убедиться, что все проходят
2. При необходимости добавить дополнительные тесты для edge cases
3. Интегрировать в CI/CD pipeline
4. Добавить тесты производительности при необходимости

