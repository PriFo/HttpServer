# Примеры использования промпта для генерации интеграционных тестов CRUD API

Практические примеры замены placeholders и использования промпта для разных сущностей.

## 📋 Пример 1: Client (Клиент)

### Подготовка placeholders

| Placeholder | Значение |
|------------|----------|
| `{entity}` | `client` |
| `{Entity}` | `Client` |
| `{table}` | `clients` |
| `{entities}` | `clients` |

### Структура сущности

```go
type Client struct {
    ID          int       `json:"id"`
    Name        string    `json:"name"`
    LegalName   string    `json:"legal_name"`
    Description string    `json:"description"`
    ContactEmail string   `json:"contact_email"`
    ContactPhone string   `json:"contact_phone"`
    TaxID       string    `json:"tax_id"`
    Country     string    `json:"country"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Ожидаемые тесты

После генерации будут созданы тесты:
- `TestClient_Create_Success`
- `TestClient_GetByID_Success`
- `TestClient_GetAll_Success`
- `TestClient_Update_Success`
- `TestClient_Delete_Success`
- `TestClient_Create_InvalidData`
- `TestClient_GetByID_NotFound`
- `TestClient_Update_NotFound`
- `TestClient_Update_InvalidData`
- `TestClient_Delete_NotFound`

### Пример URL endpoints

- `POST /api/v2/clients` - создание
- `GET /api/v2/clients/{id}` - получение по ID
- `GET /api/v2/clients` - получение всех
- `PUT /api/v2/clients/{id}` - обновление
- `DELETE /api/v2/clients/{id}` - удаление

## 📋 Пример 2: Project (Проект)

### Подготовка placeholders

| Placeholder | Значение |
|------------|----------|
| `{entity}` | `project` |
| `{Entity}` | `Project` |
| `{table}` | `projects` |
| `{entities}` | `projects` |

### Структура сущности

```go
type Project struct {
    ID          int       `json:"id"`
    ClientID    int       `json:"client_id"`
    Name        string    `json:"name"`
    Description string   `json:"description"`
    Status      string   `json:"status"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Ожидаемые тесты

- `TestProject_Create_Success`
- `TestProject_GetByID_Success`
- `TestProject_GetAll_Success`
- `TestProject_Update_Success`
- `TestProject_Delete_Success`
- И негативные сценарии...

### Пример URL endpoints

- `POST /api/v2/projects`
- `GET /api/v2/projects/{id}`
- `GET /api/v2/projects`
- `PUT /api/v2/projects/{id}`
- `DELETE /api/v2/projects/{id}`

## 📋 Пример 3: Database (База данных)

### Подготовка placeholders

| Placeholder | Значение |
|------------|----------|
| `{entity}` | `database` |
| `{Entity}` | `Database` |
| `{table}` | `databases` |
| `{entities}` | `databases` |

### Структура сущности

```go
type Database struct {
    ID          int       `json:"id"`
    ClientID    int       `json:"client_id"`
    ProjectID   int       `json:"project_id"`
    Name        string    `json:"name"`
    Path        string    `json:"path"`
    Type        string    `json:"type"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}
```

### Ожидаемые тесты

- `TestDatabase_Create_Success`
- `TestDatabase_GetByID_Success`
- `TestDatabase_GetAll_Success`
- `TestDatabase_Update_Success`
- `TestDatabase_Delete_Success`
- И негативные сценарии...

## 🔄 Процесс замены placeholders

### Шаг 1: Откройте файл

```
CRUD_CHAIN_TEST_PROMPT_COPY.txt
```

### Шаг 2: Найдите и замените

**В VS Code / Cursor:**

1. Нажмите `Ctrl+H` (или `Cmd+H`)
2. В поле "Найти" введите: `{entity}`
3. В поле "Заменить" введите: `client` (или ваше значение)
4. Нажмите "Заменить все" (или `Alt+A`)
5. Повторите для всех placeholders

**Или используйте множественную замену:**

```
{entity}    → client
{Entity}    → Client
{table}     → clients
{entities}  → clients
```

### Шаг 3: Проверьте результат

Убедитесь, что:
- ✅ Нет оставшихся `{entity}`, `{Entity}`, `{table}`, `{entities}`
- ✅ Все замены выполнены корректно
- ✅ Текст промпта читается естественно

## 📝 Пример фрагмента промпта ДО замены

```
Тестируемый ресурс: {entity} (например, Client, Project, Database) с полями:
- id (int) - уникальный идентификатор
- name (string) - название
```

## 📝 Пример фрагмента промпта ПОСЛЕ замены (для Client)

```
Тестируемый ресурс: client (например, Client, Project, Database) с полями:
- id (int) - уникальный идентификатор
- name (string) - название
```

**Примечание:** В этом примере видно, что нужно заменить только технические placeholders, а примеры в скобках можно оставить или удалить.

## 🎯 Полный пример использования

### Для сущности Client:

1. **Откройте:** `CRUD_CHAIN_TEST_PROMPT_COPY.txt`

2. **Замените:**
   ```
   {entity}   → client
   {Entity}    → Client
   {table}     → clients
   {entities}  → clients
   ```

3. **Скопируйте весь текст**

4. **Вставьте в AI чат**

5. **Результат:** AI сгенерирует файл `client_service_integration_test.go`

### Пример сгенерированного кода (фрагмент):

```go
package client

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "httpserver/database"
    clientapp "httpserver/internal/application/client"
    "httpserver/internal/api/handlers/client"
    "httpserver/internal/domain/repositories"
    "httpserver/internal/infrastructure/persistence"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/suite"
)

type ClientIntegrationTestSuite struct {
    suite.Suite
    serviceDB    *database.ServiceDB
    tx           *sql.Tx
    router       http.Handler
    server       *httptest.Server
    handler      *client.Handler
    useCase      *clientapp.UseCase
    repository   repositories.ClientRepository
}

func (suite *ClientIntegrationTestSuite) TestClient_Create_Success() {
    // Тест создания клиента
    // ...
}
```

## ⚠️ Важные замечания

### 1. Множественное число

Убедитесь, что множественное число корректно:
- `client` → `clients` ✅
- `project` → `projects` ✅
- `database` → `databases` ✅
- `category` → `categories` ✅ (не `categorys`)

### 2. Имена таблиц

Проверьте, что имя таблицы соответствует реальной схеме БД:
- Может быть `clients` или `client` (в зависимости от схемы)
- Может быть с префиксом: `service_clients`

### 3. URL endpoints

Убедитесь, что URL соответствуют реальным маршрутам:
- Может быть `/api/v2/clients`
- Может быть `/api/clients`
- Может быть `/clients`

После генерации проверьте и при необходимости скорректируйте.

## 🚀 Следующие шаги

После генерации тестов:

1. **Проверьте компиляцию:**
   ```bash
   go build ./path/to/client_service_integration_test.go
   ```

2. **Запустите тесты:**
   ```bash
   go test -v ./path/to/client_service_integration_test.go
   ```

3. **Проверьте покрытие:**
   ```bash
   go test -cover ./path/to/
   ```

4. **Адаптируйте под свои нужды:**
   - Измените URL endpoints если нужно
   - Добавьте дополнительные поля в тесты
   - Модифицируйте проверки под специфику вашей БД

---

**💡 Совет:** Начните с простой сущности (например, Client), чтобы понять процесс, затем примените к более сложным.

