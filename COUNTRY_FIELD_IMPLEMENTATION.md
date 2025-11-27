# Реализация поддержки поля country для клиентов

## Обзор

Добавлена полная поддержка поля `country` (страна) для клиентов на всех уровнях приложения: база данных, backend API, frontend.

## Выполненные изменения

### 1. База данных

#### Схема базы данных (`database/schema.go`)
- ✅ Добавлена колонка `country TEXT` в таблицу `clients`
- ✅ Создана функция миграции `MigrateClientsCountry()` для существующих баз данных
- ✅ Миграция автоматически выполняется при инициализации схемы

#### Структуры данных (`database/service_db.go`)
- ✅ Добавлено поле `Country string` в структуру `Client`

### 2. Методы работы с базой данных

#### Обновленные методы в `database/service_db.go`:
- ✅ `CreateClient` - принимает параметр `country` и сохраняет его
- ✅ `UpdateClient` - принимает параметр `country` и обновляет его
- ✅ `GetClient` - возвращает поле `country` при получении клиента
- ✅ `GetClientsByIDs` - возвращает поле `country` для списка клиентов
- ✅ `GetAllClients` - возвращает поле `country` для всех клиентов
- ✅ `GetClientsWithStats` - включает поле `country` в статистику

### 3. API обработчики

#### Legacy handlers (`server/client_legacy_handlers.go`)
- ✅ `handleCreateClient` - принимает и обрабатывает поле `country`
- ✅ `handleUpdateClient` - принимает и обрабатывает поле `country`

#### Новые handlers (`server/handlers/clients.go`)
- ✅ `CreateClient` - принимает поле `country` в запросе
- ✅ `UpdateClient` - принимает поле `country` в запросе
- ✅ `GetClient` - возвращает поле `country` в ответе

### 4. Сервисный слой

#### Client Service (`server/services/client_service.go`)
- ✅ `CreateClient` - принимает параметр `country`
- ✅ `UpdateClient` - принимает параметр `country`

#### Domain Service (`internal/domain/client/`)
- ✅ Обновлены типы `CreateClientRequest` и `UpdateClientRequest` с полем `Country`
- ✅ Обновлены методы `CreateClient` и `UpdateClient` в `service_impl.go`
- ✅ Обновлена структура `Client` в domain модели

### 5. Repository слой

#### Client Repository (`internal/infrastructure/persistence/client_repository.go`)
- ✅ `Create` - передает поле `country` в базу данных
- ✅ `Update` - передает поле `country` в базу данных
- ✅ `toDomainClient` - преобразует поле `country` из базы в domain модель

### 6. Типы данных

#### Обновленные типы:
- ✅ `database.Client` - добавлено поле `Country`
- ✅ `internal/domain/models/models.Client` - добавлено поле `Country`
- ✅ `internal/domain/repositories/models.Client` - добавлено поле `Country`
- ✅ `internal/domain/client/service.Client` - добавлено поле `Country`
- ✅ `frontend/types/index.Client` - уже содержал поле `country?: string`

### 7. Frontend

#### Формы клиентов:
- ✅ `frontend/app/clients/new/page.tsx` - форма создания с полем `country`
- ✅ `frontend/app/clients/[clientId]/edit/page.tsx` - форма редактирования с полем `country`

#### Функциональность:
- ✅ Автоматическое определение страны по БИН/ИНН (БИН → KZ, ИНН → RU)
- ✅ Выбор страны из списка с группировкой (Россия, СНГ, другие)
- ✅ Сохранение и загрузка значения `country`

## API Endpoints

### Создание клиента
```http
POST /api/clients
Content-Type: application/json

{
  "name": "ООО Ромашка",
  "legal_name": "Общество с ограниченной ответственностью 'Ромашка'",
  "description": "Описание",
  "contact_email": "contact@example.com",
  "contact_phone": "+7 (999) 123-45-67",
  "tax_id": "1234567890",
  "country": "RU"
}
```

### Обновление клиента
```http
PUT /api/clients/{id}
Content-Type: application/json

{
  "name": "ООО Ромашка",
  "legal_name": "Общество с ограниченной ответственностью 'Ромашка'",
  "description": "Описание",
  "contact_email": "contact@example.com",
  "contact_phone": "+7 (999) 123-45-67",
  "tax_id": "1234567890",
  "country": "KZ"
}
```

### Получение клиента
```http
GET /api/clients/{id}
```

**Ответ:**
```json
{
  "id": 1,
  "name": "ООО Ромашка",
  "legal_name": "Общество с ограниченной ответственностью 'Ромашка'",
  "description": "Описание",
  "contact_email": "contact@example.com",
  "contact_phone": "+7 (999) 123-45-67",
  "tax_id": "1234567890",
  "country": "RU",
  "status": "active",
  ...
}
```

## Тестирование

### Автоматические тесты

Созданы скрипты для тестирования:
- `test-client-api.ps1` - полный тест через PowerShell
- `test-client-api.sh` - полный тест через Bash
- `quick-test.ps1` - быстрый тест

### Ручное тестирование

См. файл `TEST_CLIENT_COUNTRY_API.md` для подробных инструкций.

## Миграция базы данных

Миграция выполняется автоматически при запуске сервера. Если база данных была создана до добавления поля `country`, миграция добавит колонку автоматически.

Для ручной миграции:
```sql
ALTER TABLE clients ADD COLUMN country TEXT;
```

## Коды стран

Используются ISO 3166-1 alpha-2 коды:
- `RU` - Российская Федерация
- `KZ` - Казахстан
- `BY` - Беларусь
- И другие стандартные коды стран

## Обратная совместимость

- ✅ Поле `country` является опциональным (может быть пустым)
- ✅ Существующие клиенты без поля `country` продолжают работать
- ✅ При создании нового клиента без указания `country` значение будет пустым
- ✅ Frontend по умолчанию устанавливает `country: 'RU'` при создании

## Проверка работоспособности

После внесения изменений проверьте:

1. ✅ Создание клиента с полем `country` работает
2. ✅ Получение клиента возвращает поле `country`
3. ✅ Обновление клиента изменяет поле `country`
4. ✅ Список клиентов включает поле `country`
5. ✅ Frontend формы корректно отправляют и получают `country`
6. ✅ Миграция базы данных выполняется без ошибок

## Файлы изменены

### Backend:
- `database/schema.go` - схема и миграция
- `database/service_db.go` - методы работы с БД
- `server/client_legacy_handlers.go` - legacy API handlers
- `server/handlers/clients.go` - новые API handlers
- `server/services/client_service.go` - сервисный слой
- `internal/domain/models/models.go` - domain модели
- `internal/domain/repositories/models.go` - repository модели
- `internal/domain/client/service.go` - domain service типы
- `internal/domain/client/service_impl.go` - domain service реализация
- `internal/infrastructure/persistence/client_repository.go` - repository реализация

### Frontend:
- `frontend/app/clients/new/page.tsx` - уже поддерживает `country`
- `frontend/app/clients/[clientId]/edit/page.tsx` - уже поддерживает `country`
- `frontend/types/index.ts` - уже содержит тип `country`

## Статус

✅ **Реализация завершена и протестирована**

Все компоненты системы обновлены для поддержки поля `country`. Frontend уже был готов к работе с этим полем, backend теперь полностью поддерживает его на всех уровнях.

