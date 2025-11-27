# API для работы с ГОСТами

## Обзор

API для работы с базой российских ГОСТов из открытых данных Росстандарта. Поддерживает импорт метаданных из CSV файлов, загрузку полных текстов стандартов и поиск по базе.

## База данных

База данных ГОСТов хранится в отдельном SQLite файле `gosts.db` в корне проекта.

### Структура таблиц

- `gosts` - метаданные ГОСТов (номер, название, дата принятия, статус и т.д.)
- `gost_documents` - полные тексты ГОСТов (PDF/Word файлы)
- `gost_sources` - источники данных (национальные, межгосударственные и т.д.)

## API Endpoints

### GET /api/gosts

Получить список ГОСТов с фильтрацией и пагинацией.

**Параметры запроса:**
- `limit` (int, optional) - количество записей на странице (по умолчанию: 50)
- `offset` (int, optional) - смещение для пагинации (по умолчанию: 0)
- `status` (string, optional) - фильтр по статусу (например: "действующий", "отменен")
- `source_type` (string, optional) - фильтр по типу источника (например: "nationalstandards")
- `search` (string, optional) - поисковый запрос (поиск по номеру, названию, ключевым словам)

**Пример запроса:**
```bash
GET /api/gosts?limit=20&offset=0&status=действующий&search=безопасность
```

**Пример ответа:**
```json
{
  "gosts": [
    {
      "id": 1,
      "gost_number": "ГОСТ 12345-2020",
      "title": "Название стандарта",
      "adoption_date": "2020-01-01",
      "effective_date": "2020-07-01",
      "status": "действующий",
      "source_type": "nationalstandards",
      "source_url": "https://www.rst.gov.ru/...",
      "description": "Описание стандарта",
      "keywords": "безопасность, качество",
      "created_at": "2025-01-27T10:00:00Z",
      "updated_at": "2025-01-27T10:00:00Z"
    }
  ],
  "total": 150,
  "limit": 20,
  "offset": 0
}
```

### GET /api/gosts/:id

Получить детальную информацию о ГОСТе по ID.

**Пример запроса:**
```bash
GET /api/gosts/1
```

**Пример ответа:**
```json
{
  "id": 1,
  "gost_number": "ГОСТ 12345-2020",
  "title": "Название стандарта",
  "adoption_date": "2020-01-01",
  "effective_date": "2020-07-01",
  "status": "действующий",
  "source_type": "nationalstandards",
  "source_url": "https://www.rst.gov.ru/...",
  "description": "Описание стандарта",
  "keywords": "безопасность, качество",
  "documents": [
    {
      "id": 1,
      "file_path": "data/gosts/documents/1_1234567890.pdf",
      "file_type": "pdf",
      "file_size": 1024000,
      "uploaded_at": "2025-01-27T10:00:00Z"
    }
  ],
  "created_at": "2025-01-27T10:00:00Z",
  "updated_at": "2025-01-27T10:00:00Z"
}
```

### GET /api/gosts/search

Поиск ГОСТов по номеру, названию или ключевым словам.

**Параметры запроса:**
- `q` (string, required) - поисковый запрос
- `limit` (int, optional) - количество записей (по умолчанию: 50)
- `offset` (int, optional) - смещение (по умолчанию: 0)

**Пример запроса:**
```bash
GET /api/gosts/search?q=безопасность&limit=10
```

### GET /api/gosts/number/:number

Получить ГОСТ по номеру.

**Пример запроса:**
```bash
GET /api/gosts/number/ГОСТ%2012345-2020
```

### POST /api/gosts/import

Импортировать ГОСТы из CSV файла.

**Параметры формы:**
- `file` (file, required) - CSV файл с ГОСТами
- `source_type` (string, required) - тип источника данных
- `source_url` (string, optional) - URL источника данных

**Пример запроса:**
```bash
curl -X POST http://localhost:8080/api/gosts/import \
  -F "file=@gosts.csv" \
  -F "source_type=nationalstandards" \
  -F "source_url=https://www.rst.gov.ru/opendata/7706406291-nationalstandards"
```

**Пример ответа:**
```json
{
  "success": 150,
  "total": 150,
  "errors": 0,
  "error_list": [],
  "source_id": 1
}
```

### GET /api/gosts/statistics

Получить статистику по базе ГОСТов.

**Пример запроса:**
```bash
GET /api/gosts/statistics
```

**Пример ответа:**
```json
{
  "total_gosts": 1500,
  "by_status": {
    "действующий": 1200,
    "отменен": 200,
    "заменен": 100
  },
  "by_source_type": {
    "nationalstandards": 1000,
    "interstatestandards": 500
  },
  "total_documents": 800,
  "total_sources": 2
}
```

### POST /api/gosts/:id/document

Загрузить полный текст ГОСТа (PDF/Word).

**Параметры формы:**
- `file` (file, required) - файл документа (PDF, DOC или DOCX)

**Пример запроса:**
```bash
curl -X POST http://localhost:8080/api/gosts/1/document \
  -F "file=@gost_12345.pdf"
```

**Пример ответа:**
```json
{
  "id": 1,
  "gost_id": 1,
  "file_path": "data/gosts/documents/1_1234567890.pdf",
  "file_type": "pdf",
  "file_size": 1024000,
  "uploaded_at": "2025-01-27T10:00:00Z"
}
```

### GET /api/gosts/:id/document

Получить файл документа ГОСТа.

**Параметры запроса:**
- `doc_id` (int, optional) - ID документа (если не указан, возвращается последний)

**Пример запроса:**
```bash
GET /api/gosts/1/document
```

Возвращает файл с соответствующим Content-Type.

## CLI Утилита

### Импорт из локального файла

```bash
go run cmd/import_gosts/main.go \
  -file gosts.csv \
  -source-type nationalstandards \
  -source-url https://www.rst.gov.ru/opendata/7706406291-nationalstandards \
  -verbose
```

### Автоматическая загрузка с Росстандарта

```bash
go run cmd/import_gosts/main.go \
  -download \
  -source-url https://www.rst.gov.ru/opendata/7706406291-nationalstandards \
  -source-type nationalstandards \
  -verbose
```

### Импорт из всех источников

```bash
go run cmd/import_gosts/main.go -all -verbose
```

## Источники данных

Доступные источники данных Росстандарта:

1. **nationalstandards** - Национальные стандарты
   - URL: `https://www.rst.gov.ru/opendata/7706406291-nationalstandards`

2. **interstatestandards** - Межгосударственные стандарты
   - URL: `https://www.rst.gov.ru/opendata/7706406291-interstatestandards`

3. **listnationalstandarts** - Список национальных стандартов
   - URL: `https://www.rst.gov.ru/opendata/7706406291-listnationalstandarts`

Полный список из 50 источников доступен в документации проекта.

## Формат CSV файла

CSV файлы должны содержать следующие колонки (названия могут варьироваться):

- `номер` / `номер стандарта` / `gost_number` - номер ГОСТа
- `название` / `наименование` / `title` - название стандарта
- `дата принятия` / `adoption_date` - дата принятия
- `дата вступления` / `effective_date` - дата вступления в силу
- `статус` / `status` - статус стандарта
- `описание` / `description` - описание
- `ключевые слова` / `keywords` - ключевые слова

Разделитель: точка с запятой (`;`)

## Обработка ошибок

Все ошибки возвращаются в формате:

```json
{
  "error": true,
  "message": "Описание ошибки"
}
```

HTTP статус коды:
- `200` - успешный запрос
- `400` - неверный запрос (неверные параметры)
- `404` - ресурс не найден
- `500` - внутренняя ошибка сервера

## Примеры использования

### Поиск ГОСТов по ключевому слову

```bash
curl "http://localhost:8080/api/gosts/search?q=безопасность&limit=10"
```

### Получение ГОСТа по номеру

```bash
curl "http://localhost:8080/api/gosts/number/ГОСТ%2012345-2020"
```

### Импорт ГОСТов из CSV

```bash
curl -X POST http://localhost:8080/api/gosts/import \
  -F "file=@nationalstandards.csv" \
  -F "source_type=nationalstandards" \
  -F "source_url=https://www.rst.gov.ru/opendata/7706406291-nationalstandards"
```

### Загрузка документа

```bash
curl -X POST http://localhost:8080/api/gosts/1/document \
  -F "file=@gost_document.pdf"
```

### Получение статистики

```bash
curl "http://localhost:8080/api/gosts/statistics"
```

