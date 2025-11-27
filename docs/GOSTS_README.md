# Модуль работы с ГОСТами

## Описание

Модуль для работы с базой российских ГОСТов из открытых данных Росстандарта. Поддерживает:

- Импорт метаданных ГОСТов из CSV файлов
- Хранение полных текстов стандартов (PDF/Word)
- Поиск и фильтрацию ГОСТов
- REST API для доступа к данным
- CLI утилиту для импорта данных

## Быстрый старт

### 1. Импорт данных через CLI

```bash
# Импорт из локального CSV файла
go run cmd/import_gosts/main.go \
  -file gosts.csv \
  -source-type nationalstandards \
  -source-url https://www.rst.gov.ru/opendata/7706406291-nationalstandards

# Автоматическая загрузка с Росстандарта
go run cmd/import_gosts/main.go \
  -download \
  -source-url https://www.rst.gov.ru/opendata/7706406291-nationalstandards \
  -source-type nationalstandards

# Импорт из всех доступных источников
go run cmd/import_gosts/main.go -all
```

### 2. Использование API

```bash
# Получить список ГОСТов
curl "http://localhost:8080/api/gosts?limit=10"

# Поиск ГОСТов
curl "http://localhost:8080/api/gosts/search?q=безопасность"

# Получить ГОСТ по номеру
curl "http://localhost:8080/api/gosts/number/ГОСТ%2012345-2020"

# Получить статистику
curl "http://localhost:8080/api/gosts/statistics"
```

## Структура проекта

```
database/
├── gosts_db.go          # Работа с БД ГОСТов
└── gosts_schema.go      # SQL схема

importer/
└── gost_parser.go       # Парсер CSV файлов

server/
├── services/
│   ├── gost_service.go       # Бизнес-логика
│   └── gost_service_test.go  # Тесты
└── handlers/
    └── gost_handler.go       # HTTP handlers

cmd/
└── import_gosts/
    └── main.go                # CLI утилита

docs/
├── GOSTS_API.md         # Документация API
└── GOSTS_README.md      # Этот файл
```

## База данных

База данных ГОСТов хранится в отдельном SQLite файле `gosts.db` в корне проекта.

### Таблицы

#### gosts
Метаданные ГОСТов:
- `id` - уникальный идентификатор
- `gost_number` - номер ГОСТа (уникальный)
- `title` - название стандарта
- `adoption_date` - дата принятия
- `effective_date` - дата вступления в силу
- `status` - статус (действующий, отменен, заменен)
- `source_type` - тип источника данных
- `source_id` - ссылка на источник
- `source_url` - URL источника
- `description` - описание
- `keywords` - ключевые слова для поиска

#### gost_documents
Полные тексты ГОСТов:
- `id` - уникальный идентификатор
- `gost_id` - ссылка на ГОСТ
- `file_path` - путь к файлу на диске
- `file_type` - тип файла (pdf, doc, docx)
- `file_size` - размер файла
- `uploaded_at` - дата загрузки

#### gost_sources
Источники данных:
- `id` - уникальный идентификатор
- `source_name` - название источника (уникальное)
- `source_url` - URL источника
- `last_sync_date` - дата последней синхронизации
- `records_count` - количество записей

## API Endpoints

Подробная документация API доступна в [GOSTS_API.md](./GOSTS_API.md).

### Основные endpoints:

- `GET /api/gosts` - список ГОСТов с фильтрацией
- `GET /api/gosts/:id` - детальная информация о ГОСТе
- `GET /api/gosts/search` - поиск ГОСТов
- `GET /api/gosts/number/:number` - получение по номеру
- `POST /api/gosts/import` - импорт из CSV
- `GET /api/gosts/statistics` - статистика
- `POST /api/gosts/:id/document` - загрузка документа
- `GET /api/gosts/:id/document` - получение документа

## Тестирование

### Запуск тестов сервиса

```bash
go test ./server/services -run TestGostService -v
```

### Покрытие тестами

```bash
go test ./server/services -run TestGostService -cover
```

## Источники данных

Модуль поддерживает работу с открытыми данными Росстандарта:

1. **Национальные стандарты**
   - URL: `https://www.rst.gov.ru/opendata/7706406291-nationalstandards`

2. **Межгосударственные стандарты**
   - URL: `https://www.rst.gov.ru/opendata/7706406291-interstatestandards`

3. **Список национальных стандартов**
   - URL: `https://www.rst.gov.ru/opendata/7706406291-listnationalstandarts`

Полный список из 50 источников доступен в документации проекта.

## Формат CSV

CSV файлы должны использовать разделитель точка с запятой (`;`) и содержать следующие колонки:

- Номер ГОСТа: `номер`, `номер стандарта`, `gost_number`
- Название: `название`, `наименование`, `title`
- Дата принятия: `дата принятия`, `adoption_date`
- Дата вступления: `дата вступления`, `effective_date`
- Статус: `статус`, `status`
- Описание: `описание`, `description`
- Ключевые слова: `ключевые слова`, `keywords`

## Примеры использования

### Импорт через CLI

```bash
# Базовый импорт
go run cmd/import_gosts/main.go \
  -file nationalstandards.csv \
  -source-type nationalstandards \
  -source-url https://www.rst.gov.ru/opendata/7706406291-nationalstandards \
  -verbose

# Импорт из всех источников
go run cmd/import_gosts/main.go -all -verbose
```

### Использование API

```bash
# Поиск ГОСТов
curl "http://localhost:8080/api/gosts/search?q=безопасность&limit=10"

# Импорт через API
curl -X POST http://localhost:8080/api/gosts/import \
  -F "file=@gosts.csv" \
  -F "source_type=nationalstandards" \
  -F "source_url=https://www.rst.gov.ru/opendata/7706406291-nationalstandards"

# Загрузка документа
curl -X POST http://localhost:8080/api/gosts/1/document \
  -F "file=@gost_document.pdf"
```

## Разработка

### Добавление новых функций

1. **Новые поля в БД**: обновите `database/gosts_schema.go`
2. **Новая бизнес-логика**: добавьте методы в `server/services/gost_service.go`
3. **Новые API endpoints**: добавьте handlers в `server/handlers/gost_handler.go`
4. **Тесты**: добавьте тесты в `server/services/gost_service_test.go`

### Запуск в режиме разработки

```bash
# Запуск сервера
go run main.go

# В другом терминале - импорт данных
go run cmd/import_gosts/main.go -all -verbose
```

## Лицензия

Модуль использует открытые данные Росстандарта, которые доступны для свободного использования.

