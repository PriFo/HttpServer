# Тестирование API клиентов с полем country

## Быстрый старт

### 1. Запуск сервера

```bash
# В корне проекта
go run main.go
```

Или если сервер уже скомпилирован:
```bash
./httpserver.exe
```

Сервер будет доступен на `http://127.0.0.1:9999`

### 2. Тестирование через PowerShell

```powershell
.\test-client-api.ps1
```

### 3. Тестирование через Bash/curl

```bash
bash test-client-api.sh
```

## Ручное тестирование через curl

### 1. Создание клиента с полем country

```bash
curl -X POST http://127.0.0.1:9999/api/clients \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Тестовый клиент",
    "legal_name": "ООО Тестовый клиент",
    "description": "Клиент для тестирования",
    "contact_email": "test@example.com",
    "contact_phone": "+7 (999) 123-45-67",
    "tax_id": "1234567890",
    "country": "RU"
  }'
```

**Ожидаемый ответ:**
```json
{
  "id": 1,
  "name": "Тестовый клиент",
  "legal_name": "ООО Тестовый клиент",
  "description": "Клиент для тестирования",
  "contact_email": "test@example.com",
  "contact_phone": "+7 (999) 123-45-67",
  "tax_id": "1234567890",
  "country": "RU",
  "status": "active",
  ...
}
```

### 2. Получение клиента

```bash
curl -X GET http://127.0.0.1:9999/api/clients/1 \
  -H "Content-Type: application/json"
```

**Проверьте, что в ответе есть поле `country`**

### 3. Обновление клиента с изменением country

```bash
curl -X PUT http://127.0.0.1:9999/api/clients/1 \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Тестовый клиент",
    "legal_name": "ООО Тестовый клиент",
    "description": "Клиент для тестирования",
    "contact_email": "test@example.com",
    "contact_phone": "+7 (999) 123-45-67",
    "tax_id": "1234567890",
    "country": "KZ"
  }'
```

**Ожидаемый ответ:**
```json
{
  "id": 1,
  ...
  "country": "KZ",
  ...
}
```

### 4. Проверка сохранения country

```bash
curl -X GET http://127.0.0.1:9999/api/clients/1 \
  -H "Content-Type: application/json"
```

**Убедитесь, что `country` изменился на `KZ`**

### 5. Получение списка клиентов

```bash
curl -X GET http://127.0.0.1:9999/api/clients \
  -H "Content-Type: application/json"
```

**Проверьте, что в списке клиентов есть поле `country`**

## Тестирование через PowerShell (ручные команды)

### 1. Создание клиента

```powershell
$body = @{
    name = "Тестовый клиент"
    legal_name = "ООО Тестовый клиент"
    description = "Клиент для тестирования"
    contact_email = "test@example.com"
    contact_phone = "+7 (999) 123-45-67"
    tax_id = "1234567890"
    country = "RU"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://127.0.0.1:9999/api/clients" `
    -Method POST `
    -Headers @{"Content-Type"="application/json"} `
    -Body $body
```

### 2. Получение клиента

```powershell
$clientId = 1
Invoke-RestMethod -Uri "http://127.0.0.1:9999/api/clients/$clientId" `
    -Method GET `
    -Headers @{"Content-Type"="application/json"}
```

### 3. Обновление клиента

```powershell
$body = @{
    name = "Тестовый клиент"
    legal_name = "ООО Тестовый клиент"
    description = "Клиент для тестирования"
    contact_email = "test@example.com"
    contact_phone = "+7 (999) 123-45-67"
    tax_id = "1234567890"
    country = "KZ"
} | ConvertTo-Json

Invoke-RestMethod -Uri "http://127.0.0.1:9999/api/clients/1" `
    -Method PUT `
    -Headers @{"Content-Type"="application/json"} `
    -Body $body
```

## Проверка результатов

После выполнения тестов убедитесь, что:

1. ✅ При создании клиента поле `country` сохраняется
2. ✅ При получении клиента поле `country` возвращается
3. ✅ При обновлении клиента поле `country` изменяется
4. ✅ В списке клиентов поле `country` присутствует
5. ✅ Значение `country` корректно сохраняется в базе данных

## Возможные проблемы

### Сервер не отвечает

Убедитесь, что сервер запущен:
```bash
# Проверка порта
netstat -an | findstr :9999
```

### Ошибка "column country does not exist"

Выполните миграцию базы данных. Миграция должна выполниться автоматически при запуске сервера, но если база данных была создана до добавления поля `country`, может потребоваться:

1. Удалить старую базу данных `service.db`
2. Перезапустить сервер (создастся новая база с полем `country`)

Или выполнить миграцию вручную:
```sql
ALTER TABLE clients ADD COLUMN country TEXT;
```

## Коды стран

Используются ISO 3166-1 alpha-2 коды:
- `RU` - Российская Федерация
- `KZ` - Казахстан
- `BY` - Беларусь
- и т.д.

