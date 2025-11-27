# Исправление ошибки с NULL-значениями в базе данных

## Дата: 26.11.2025

## Проблема

При обращении к endpoint `/api/clients/:id` возникала ошибка:
```
ERROR: sql: Scan error on column index 7, name "country": 
converting NULL to string is unsupported
```

### Причина
Метод `GetClient()` в файле `database/service_db.go` пытался считать NULL-значения из БД напрямую в строковые поля (string), что не поддерживается в Go без использования `sql.NullString`.

### Симптомы
- Страница `http://localhost:3000/clients/1/projects/3` отображала ошибку "Бэкенд вернул некорректный ответ"
- GET `/api/clients/1` возвращал 404 Not Found
- В логах бэкенда: "Failed to get client: sql.Scan error"

## Решение

### Файл: `database/service_db.go`

**Изменено:** Метод `GetClient(id int)` (строка 341)

**До:**
```go
err := row.Scan(
    &client.ID, &client.Name, &client.LegalName, &client.Description,
    &client.ContactEmail, &client.ContactPhone, &client.TaxID, &client.Country,
    &client.Status, &client.CreatedBy, &client.CreatedAt, &client.UpdatedAt,
)
```

**После:**
```go
var (
    description  sql.NullString
    contactEmail sql.NullString
    contactPhone sql.NullString
    taxID        sql.NullString
    country      sql.NullString
    status       sql.NullString
    createdBy    sql.NullString
)

err := row.Scan(
    &client.ID,
    &client.Name,
    &client.LegalName,
    &description,
    &contactEmail,
    &contactPhone,
    &taxID,
    &country,
    &status,
    &createdBy,
    &client.CreatedAt,
    &client.UpdatedAt,
)

if err != nil {
    return nil, fmt.Errorf("failed to get client: %w", err)
}

client.Description = nullString(description)
client.ContactEmail = nullString(contactEmail)
client.ContactPhone = nullString(contactPhone)
client.TaxID = nullString(taxID)
client.Country = nullString(country)
client.Status = nullString(status)
client.CreatedBy = nullString(createdBy)
```

### Вспомогательная функция
Используется существующая функция `nullString()` для безопасного преобразования:
```go
func nullString(ns sql.NullString) string {
    if ns.Valid {
        return ns.String
    }
    return ""
}
```

## Проверка

### Тестирование
```powershell
# 1. Проверка проблемного клиента
curl http://localhost:9999/api/clients/1
# Результат: ✅ 200 OK, JSON с данными клиента

# 2. Проверка всех клиентов
curl http://localhost:9999/api/clients
# Результат: ✅ 8 клиентов, все загружаются без ошибок

# 3. Проверка страницы проекта
curl http://localhost:3000/clients/1/projects/3
# Результат: ✅ 200 OK, страница загружается
```

### Результаты
- ✅ Все 8 клиентов в БД загружаются без ошибок
- ✅ Страницы фронтенда работают корректно
- ✅ NULL-значения корректно преобразуются в пустые строки
- ✅ Linter ошибок нет

## Дополнительная информация

### Другие методы
Методы `GetClientsByIDs()` и `GetAllClients()` уже использовали правильную обработку NULL-значений через `sql.NullString`.

### Профилактика
При работе с базой данных SQLite всегда используйте:
- `sql.NullString` для строковых полей, которые могут быть NULL
- `sql.NullInt64` для числовых полей
- `sql.NullTime` для полей с датой/временем
- `sql.NullBool` для булевых полей

## Сервисы

### Backend (Go)
- **URL:** http://localhost:9999
- **Status:** ✅ Running (PID: 34792)

### Frontend (Next.js)
- **URL:** http://localhost:3000
- **Status:** ✅ Running

### База данных
- **Файл:** 1c_data.db
- **Клиентов:** 8
- **Проектов:** 6+

## Автор
AI Assistant (Claude Sonnet 4.5)

