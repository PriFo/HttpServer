# Примеры использования API получения всех контрагентов

**Endpoint:** `/api/counterparties/all`  
**Базовый URL:** http://localhost:9999

## Базовые примеры

### 1. Получить все контрагенты клиента

```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1"
```

**Ответ:**
```json
{
  "counterparties": [...],
  "projects": [...],
  "total": 13477,
  "offset": 0,
  "limit": 100,
  "stats": {
    "total_from_database": 13477,
    "total_normalized": 0,
    "databases_processed": 4,
    "projects_processed": 2,
    "processing_time_ms": 136
  }
}
```

### 2. С пагинацией

```bash
# Первая страница (10 записей)
curl "http://localhost:9999/api/counterparties/all?client_id=1&offset=0&limit=10"

# Вторая страница
curl "http://localhost:9999/api/counterparties/all?client_id=1&offset=10&limit=10"
```

### 3. Фильтр по источнику

```bash
# Только из баз данных
curl "http://localhost:9999/api/counterparties/all?client_id=1&source=database"

# Только нормализованные
curl "http://localhost:9999/api/counterparties/all?client_id=1&source=normalized"

# Все (по умолчанию)
curl "http://localhost:9999/api/counterparties/all?client_id=1"
```

### 4. Поиск

```bash
# Поиск по имени, ИНН или БИН
curl "http://localhost:9999/api/counterparties/all?client_id=1&search=ООО"

# Поиск по ИНН
curl "http://localhost:9999/api/counterparties/all?client_id=1&search=1234567890"
```

### 5. Сортировка

```bash
# По имени (по возрастанию)
curl "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=name&order=asc"

# По качеству (по убыванию)
curl "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=quality&order=desc"

# По источнику
curl "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=source&order=asc"

# По ID
curl "http://localhost:9999/api/counterparties/all?client_id=1&sort_by=id&order=asc"
```

### 6. Фильтр по проекту

```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&project_id=3"
```

### 7. Фильтр по качеству

```bash
# Минимальное качество
curl "http://localhost:9999/api/counterparties/all?client_id=1&min_quality=0.8"

# Максимальное качество
curl "http://localhost:9999/api/counterparties/all?client_id=1&max_quality=0.9"

# Диапазон качества
curl "http://localhost:9999/api/counterparties/all?client_id=1&min_quality=0.5&max_quality=0.9"
```

## Комбинированные примеры

### Поиск с фильтрацией и сортировкой

```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&search=ООО&source=database&sort_by=name&order=asc&limit=20"
```

### Конкретный проект с фильтрами

```bash
curl "http://localhost:9999/api/counterparties/all?client_id=1&project_id=3&source=database&min_quality=0.7&sort_by=quality&order=desc"
```

## Экспорт данных

### Экспорт в JSON

```bash
curl "http://localhost:9999/api/counterparties/all/export?client_id=1&format=json" -o counterparties.json
```

### Экспорт в CSV

```bash
curl "http://localhost:9999/api/counterparties/all/export?client_id=1&format=csv" -o counterparties.csv
```

### Экспорт с фильтрами

```bash
# Экспорт только нормализованных контрагентов
curl "http://localhost:9999/api/counterparties/all/export?client_id=1&source=normalized&format=csv" -o normalized.csv

# Экспорт с поиском
curl "http://localhost:9999/api/counterparties/all/export?client_id=1&search=ООО&format=json" -o search_results.json
```

## Примеры для PowerShell

### Базовый запрос

```powershell
$response = Invoke-RestMethod -Uri "http://localhost:9999/api/counterparties/all?client_id=1&limit=10"
$response.counterparties | Format-Table id, name, source, project_name
```

### Получение статистики

```powershell
$stats = Invoke-RestMethod -Uri "http://localhost:9999/api/counterparties/all?client_id=1&limit=0"
Write-Host "Всего контрагентов: $($stats.total)"
Write-Host "Из баз данных: $($stats.stats.total_from_database)"
Write-Host "Нормализованных: $($stats.stats.total_normalized)"
Write-Host "Обработано баз: $($stats.stats.databases_processed)"
```

### Поиск и фильтрация

```powershell
$search = "ООО"
$result = Invoke-RestMethod -Uri "http://localhost:9999/api/counterparties/all?client_id=1&search=$search&limit=10"
$result.counterparties | Select-Object id, name, source, tax_id | Format-Table
```

## Примеры для JavaScript/TypeScript

### Fetch API

```javascript
// Базовый запрос
const response = await fetch('http://localhost:9999/api/counterparties/all?client_id=1&limit=10');
const data = await response.json();

console.log(`Всего: ${data.total}`);
console.log(`Возвращено: ${data.counterparties.length}`);
console.log(`Статистика:`, data.stats);
```

### С фильтрами

```javascript
const params = new URLSearchParams({
  client_id: 1,
  source: 'database',
  search: 'ООО',
  sort_by: 'name',
  order: 'asc',
  limit: 20
});

const response = await fetch(`http://localhost:9999/api/counterparties/all?${params}`);
const data = await response.json();
```

### Экспорт

```javascript
const exportUrl = 'http://localhost:9999/api/counterparties/all/export?client_id=1&format=json';
const response = await fetch(exportUrl);
const blob = await response.blob();

// Скачать файл
const url = window.URL.createObjectURL(blob);
const a = document.createElement('a');
a.href = url;
a.download = 'counterparties.json';
a.click();
```

## Примеры для Python

### Requests

```python
import requests

# Базовый запрос
response = requests.get('http://localhost:9999/api/counterparties/all', params={
    'client_id': 1,
    'limit': 10
})
data = response.json()

print(f"Всего: {data['total']}")
print(f"Возвращено: {len(data['counterparties'])}")
print(f"Статистика: {data['stats']}")
```

### С фильтрами

```python
params = {
    'client_id': 1,
    'source': 'database',
    'search': 'ООО',
    'sort_by': 'name',
    'order': 'asc',
    'limit': 20
}

response = requests.get('http://localhost:9999/api/counterparties/all', params=params)
data = response.json()

for cp in data['counterparties']:
    print(f"{cp['id']}: {cp['name']} ({cp['source']})")
```

## Структура ответа

### UnifiedCounterparty

```json
{
  "id": 695,
  "name": "ООО Пример",
  "source": "database",
  "project_id": 3,
  "project_name": "mdm aitas",
  "database_id": 2,
  "database_name": "База данных",
  "reference": "xxx-xxx-xxx",
  "code": "001",
  "tax_id": "1234567890",
  "kpp": "123456789",
  "bin": "",
  "legal_address": "г. Москва, ул. Примерная, д. 1",
  "postal_address": "г. Москва, ул. Примерная, д. 1",
  "contact_phone": "+7 (495) 123-45-67",
  "contact_email": "info@example.com",
  "contact_person": "Иванов Иван Иванович",
  "quality_score": null
}
```

### Статистика

```json
{
  "stats": {
    "total_from_database": 13477,
    "total_normalized": 0,
    "total_with_quality": 0,
    "average_quality": 0.0,
    "databases_processed": 4,
    "projects_processed": 2,
    "processing_time_ms": 136
  }
}
```

## Коды ответов

- **200 OK** - Успешный запрос
- **400 Bad Request** - Ошибка валидации (например, отсутствует client_id)
- **500 Internal Server Error** - Внутренняя ошибка сервера

## Обработка ошибок

### Пример с обработкой ошибок (JavaScript)

```javascript
try {
  const response = await fetch('http://localhost:9999/api/counterparties/all?client_id=1');
  
  if (!response.ok) {
    if (response.status === 400) {
      const error = await response.json();
      console.error('Ошибка валидации:', error);
    } else {
      console.error('Ошибка сервера:', response.status);
    }
    return;
  }
  
  const data = await response.json();
  // Обработка данных
} catch (error) {
  console.error('Ошибка сети:', error);
}
```

## Производительность

### Рекомендации

- Используйте пагинацию для больших наборов данных (limit=20-100)
- Для получения только статистики используйте `limit=0`
- Используйте фильтры для уменьшения объема данных
- Кэшируйте результаты на клиенте при необходимости

### Пример оптимизации

```javascript
// Получить только статистику без данных
const statsResponse = await fetch('http://localhost:9999/api/counterparties/all?client_id=1&limit=0');
const stats = await statsResponse.json();

// Если нужно, загрузить данные с пагинацией
if (stats.total > 0) {
  const dataResponse = await fetch('http://localhost:9999/api/counterparties/all?client_id=1&limit=50');
  const data = await dataResponse.json();
}
```

## Дополнительные ресурсы

- Полная документация: `api_tests/COUNTERPARTIES_ALL_API.md`
- Отчет о верификации: `COUNTERPARTIES_ALL_API_VERIFICATION_REPORT.md`
- Отчет о тестировании: `COUNTERPARTIES_ALL_API_TEST_REPORT.md`
- Финальный отчет: `COUNTERPARTIES_ALL_API_FINAL_TEST_REPORT.md`

