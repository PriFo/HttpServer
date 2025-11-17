# API Документация - HTTP Server для работы с выгрузками из 1С

## Содержание

1. [Общая информация](#общая-информация)
2. [Архитектура системы](#архитектура-системы)
3. [Принцип работы](#принцип-работы)
4. [Структура базы данных](#структура-базы-данных)
5. [Форматы данных](#форматы-данных)
6. [Эндпоинты](#эндпоинты)
7. [Обработка ошибок](#обработка-ошибок)
8. [Примеры использования](#примеры-использования)

---

## Общая информация

HTTP Server для работы с выгрузками из 1С - это RESTful API сервер, написанный на Go, который предоставляет интерфейс для приема, хранения и получения данных из 1С:Предприятие. Сервер работает с двумя SQLite базами данных: основной (для исходных данных из 1С) и нормализованной (для обработанных данных, готовых к загрузке обратно в 1С).

**Базовый URL:** `http://localhost:9999`

**Формат данных:** 
- JSON - для списков выгрузок и детальной информации
- XML - для получения данных выгрузки (`/data` и `/stream`)

**Кодировка:** UTF-8

**CORS:** Поддерживается для всех источников

**Технологии:**
- Язык: Go (Golang)
- База данных: SQLite3
- Протокол: HTTP/1.1
- Форматы: JSON, XML

---

## Архитектура системы

### Компоненты системы

1. **HTTP Server** (`server/server.go`)
   - Обрабатывает HTTP запросы
   - Маршрутизирует запросы к соответствующим обработчикам
   - Управляет CORS заголовками
   - Логирует все операции

2. **Database Layer** (`database/db.go`, `database/schema.go`)
   - Управляет подключением к SQLite базам данных
   - Выполняет CRUD операции
   - Обеспечивает целостность данных через внешние ключи

3. **Models** (`server/models.go`)
   - Определяет структуры данных для запросов и ответов
   - Обеспечивает сериализацию/десериализацию XML и JSON

### Две базы данных

Система использует две независимые SQLite базы данных:

1. **Основная БД** (`data.db` или `1c_data.db`)
   - Хранит исходные данные, полученные напрямую из 1С
   - Используется для чтения исходных данных
   - Эндпоинты: `/api/uploads/*`

2. **Нормализованная БД** (`normalized_data.db`)
   - Хранит обработанные и нормализованные данные
   - Используется для записи обработанных данных и их последующей загрузки в 1С
   - Эндпоинты: `/api/normalized/uploads/*` и `/api/normalized/upload/*`

Обе базы данных имеют идентичную структуру, что позволяет легко переносить данные между ними.

### Поток данных

```
1С:Предприятие
    ↓ (XML через POST)
HTTP Server → Основная БД (data.db)
    ↓ (REST API)
Внешний обработчик
    ↓ (нормализация данных)
HTTP Server → Нормализованная БД (normalized_data.db)
    ↓ (REST API)
1С:Предприятие (загрузка обработанных данных)
```

---

## Принцип работы

### 1. Выгрузка данных из 1С

Процесс выгрузки данных из 1С состоит из следующих этапов:

#### Шаг 1: Handshake (Рукопожатие)
- **Эндпоинт:** `POST /handshake`
- **Назначение:** Создание новой выгрузки в основной БД
- **Процесс:**
  1. 1С отправляет XML с версией 1С и именем конфигурации
  2. Сервер генерирует уникальный UUID для выгрузки
  3. Создается запись в таблице `uploads` со статусом `in_progress`
  4. Сервер возвращает `upload_uuid` для использования в последующих запросах

#### Шаг 2: Metadata (Метаданные)
- **Эндпоинт:** `POST /metadata`
- **Назначение:** Подтверждение метаинформации о выгрузке
- **Процесс:**
  1. 1С отправляет метаданные с `upload_uuid`
  2. Сервер проверяет существование выгрузки
  3. Метаданные логируются (версия 1С, имя конфигурации)

#### Шаг 3: Constants (Константы)
- **Эндпоинт:** `POST /constant`
- **Назначение:** Добавление констант из 1С
- **Процесс:**
  1. 1С отправляет XML с данными константы
  2. Сервер парсит XML, извлекая:
     - `name` - имя константы
     - `synonym` - синоним
     - `type` - тип данных
     - `value` - значение (может содержать вложенный XML)
  3. Данные сохраняются в таблицу `constants`
  4. Счетчик `total_constants` в таблице `uploads` увеличивается
  5. Значение константы сохраняется как строка (может содержать XML)

**Особенность:** Поле `value` может содержать сложные XML структуры. Сервер использует кастомный парсер (`ConstantValue.UnmarshalXML`), который сохраняет все содержимое тега `<value>` как есть, включая вложенные XML элементы.

#### Шаг 4: Catalog Meta (Метаданные справочников)
- **Эндпоинт:** `POST /catalog/meta`
- **Назначение:** Регистрация справочников
- **Процесс:**
  1. 1С отправляет метаданные справочника (имя, синоним)
  2. Сервер создает запись в таблице `catalogs`
  3. Возвращается `catalog_id` для последующего использования
  4. Счетчик `total_catalogs` увеличивается

#### Шаг 5: Catalog Item (Элементы справочников)
- **Эндпоинт:** `POST /catalog/item`
- **Назначение:** Добавление элементов справочников
- **Процесс:**
  1. 1С отправляет XML с данными элемента справочника
  2. Сервер находит справочник по имени (`catalog_name`)
  3. Извлекаются данные:
     - `reference` - ссылка на элемент
     - `code` - код элемента
     - `name` - наименование
     - `attributes` - реквизиты (XML строка)
     - `table_parts` - табличные части (XML строка)
  4. Данные сохраняются в таблицу `catalog_items`
  5. Счетчик `total_items` увеличивается

**Важно:** Поля `attributes` и `table_parts` сохраняются как XML строки в том формате, в котором они приходят из 1С. Это позволяет сохранить всю структуру данных без потери информации.

#### Шаг 6: Complete (Завершение)
- **Эндпоинт:** `POST /complete`
- **Назначение:** Завершение выгрузки
- **Процесс:**
  1. 1С отправляет запрос на завершение
  2. Сервер обновляет статус выгрузки на `completed`
  3. Устанавливается `completed_at` (время завершения)
  4. Выгрузка становится доступной для чтения через API

### 2. Получение данных через API

После завершения выгрузки данные можно получить через REST API:

#### Список выгрузок
- **Эндпоинт:** `GET /api/uploads`
- **Процесс:**
  1. Сервер запрашивает все выгрузки из основной БД
  2. Для каждой выгрузки подсчитываются статистики (константы, справочники, элементы)
  3. Данные возвращаются в формате JSON

#### Детали выгрузки
- **Эндпоинт:** `GET /api/uploads/{uuid}`
- **Процесс:**
  1. Сервер находит выгрузку по UUID
  2. Получает список всех справочников с количеством элементов
  3. Получает список всех констант
  4. Формирует детальный ответ в JSON

#### Получение данных с фильтрацией
- **Эндпоинт:** `GET /api/uploads/{uuid}/data`
- **Параметры:**
  - `type` - тип данных: `all`, `constants`, `catalogs`
  - `catalog_names` - фильтр по именам справочников (через запятую)
  - `page` - номер страницы (начиная с 1)
  - `limit` - количество элементов на странице (максимум 1000)

**Процесс:**
1. Сервер определяет тип запроса (`constants`, `catalogs` или `all`)
2. Если `type=constants`:
   - Получает все константы выгрузки
   - Применяет пагинацию
   - Формирует XML для каждой константы
3. Если `type=catalogs`:
   - Применяет фильтр по именам справочников (если указан)
   - Получает элементы справочников с пагинацией
   - Для каждого элемента формирует XML с полными данными
4. Если `type=all`:
   - Получает все константы и элементы справочников
   - Объединяет их в один список
   - Применяет пагинацию к объединенному списку
   - Сначала идут константы, затем элементы справочников
5. Все данные возвращаются в формате XML

**Особенности:**
- Элементы возвращаются в порядке вставки в БД (по возрастанию ID)
- Поля `attributes_xml` и `table_parts_xml` вставляются как есть (innerXML)
- Поле `catalog_name` всегда присутствует в элементах справочников

#### Потоковая передача данных (SSE)
- **Эндпоинт:** `GET /api/uploads/{uuid}/stream`
- **Процесс:**
  1. Сервер устанавливает заголовки для Server-Sent Events (SSE)
  2. Открывает поток для записи
  3. Последовательно отправляет каждый элемент:
     - Константы (если `type=constants` или `type=all`)
     - Элементы справочников (если `type=catalogs` или `type=all`)
  4. Каждый элемент отправляется как отдельное событие SSE в формате `data: <item>...</item>\n\n`
  5. После отправки всех элементов отправляется завершающее событие `data: {"type":"complete"}\n\n`
  6. Поток закрывается

**Преимущества SSE:**
- Не требует загрузки всех данных в память
- Позволяет обрабатывать данные по мере поступления
- Подходит для больших объемов данных
- Автоматическое переподключение при обрыве соединения

#### Проверка передачи
- **Эндпоинт:** `POST /api/uploads/{uuid}/verify`
- **Процесс:**
  1. Клиент отправляет массив полученных ID элементов
  2. Сервер получает все ID элементов выгрузки (константы + элементы справочников)
  3. Сравнивает полученные ID с ожидаемыми
  4. Определяет отсутствующие элементы
  5. Возвращает результат проверки

### 3. Нормализация данных

Процесс нормализации данных позволяет обработать исходные данные из 1С и подготовить их для загрузки обратно в 1С.

#### Запись нормализованных данных

Процесс аналогичен выгрузке из 1С, но использует эндпоинты с префиксом `/api/normalized/upload/`:

1. **Handshake** - создание выгрузки в нормализованной БД
2. **Metadata** - отправка метаданных
3. **Constant** - добавление нормализованных констант
4. **Catalog Meta** - регистрация справочников
5. **Catalog Item** - добавление элементов справочников
6. **Complete** - завершение выгрузки

#### Получение нормализованных данных

После завершения выгрузки нормализованные данные доступны через эндпоинты с префиксом `/api/normalized/uploads/`:

- `GET /api/normalized/uploads` - список выгрузок
- `GET /api/normalized/uploads/{uuid}` - детали выгрузки
- `GET /api/normalized/uploads/{uuid}/data` - получение данных
- `GET /api/normalized/uploads/{uuid}/stream` - потоковая передача
- `POST /api/normalized/uploads/{uuid}/verify` - проверка передачи

Все эндпоинты работают аналогично основным, но используют нормализованную БД.

---

## Структура базы данных

### Таблица `uploads`

Хранит информацию о выгрузках.

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER | Первичный ключ (AUTOINCREMENT) |
| `upload_uuid` | TEXT | Уникальный идентификатор выгрузки (UNIQUE) |
| `started_at` | TIMESTAMP | Время начала выгрузки (DEFAULT CURRENT_TIMESTAMP) |
| `completed_at` | TIMESTAMP | Время завершения выгрузки (NULL для незавершенных) |
| `status` | TEXT | Статус: `in_progress` или `completed` |
| `version_1c` | TEXT | Версия 1С |
| `config_name` | TEXT | Имя конфигурации 1С |
| `total_constants` | INTEGER | Количество констант (DEFAULT 0) |
| `total_catalogs` | INTEGER | Количество справочников (DEFAULT 0) |
| `total_items` | INTEGER | Количество элементов справочников (DEFAULT 0) |

**Индексы:**
- `idx_uploads_uuid` на `upload_uuid` - для быстрого поиска по UUID

### Таблица `constants`

Хранит константы из 1С.

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER | Первичный ключ (AUTOINCREMENT) |
| `upload_id` | INTEGER | Внешний ключ на `uploads.id` |
| `name` | TEXT | Имя константы |
| `synonym` | TEXT | Синоним константы |
| `type` | TEXT | Тип данных константы |
| `value` | TEXT | Значение константы (может содержать XML) |
| `created_at` | TIMESTAMP | Время создания (DEFAULT CURRENT_TIMESTAMP) |

**Индексы:**
- `idx_constants_upload_id` на `upload_id` - для быстрого поиска констант выгрузки

**Внешние ключи:**
- `upload_id` → `uploads.id` ON DELETE CASCADE

### Таблица `catalogs`

Хранит метаданные справочников.

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER | Первичный ключ (AUTOINCREMENT) |
| `upload_id` | INTEGER | Внешний ключ на `uploads.id` |
| `name` | TEXT | Имя справочника |
| `synonym` | TEXT | Синоним справочника |
| `created_at` | TIMESTAMP | Время создания (DEFAULT CURRENT_TIMESTAMP) |

**Индексы:**
- `idx_catalogs_upload_id` на `upload_id` - для быстрого поиска справочников выгрузки

**Внешние ключи:**
- `upload_id` → `uploads.id` ON DELETE CASCADE

### Таблица `catalog_items`

Хранит элементы справочников.

| Колонка | Тип | Описание |
|---------|-----|----------|
| `id` | INTEGER | Первичный ключ (AUTOINCREMENT) |
| `catalog_id` | INTEGER | Внешний ключ на `catalogs.id` |
| `reference` | TEXT | Ссылка на элемент (уникальный идентификатор в 1С) |
| `code` | TEXT | Код элемента |
| `name` | TEXT | Наименование элемента |
| `attributes_xml` | TEXT | Реквизиты элемента в формате XML |
| `table_parts_xml` | TEXT | Табличные части элемента в формате XML |
| `created_at` | TIMESTAMP | Время создания (DEFAULT CURRENT_TIMESTAMP) |

**Индексы:**
- `idx_catalog_items_catalog_id` на `catalog_id` - для быстрого поиска элементов справочника
- `idx_catalog_items_reference` на `reference` - для быстрого поиска по ссылке

**Внешние ключи:**
- `catalog_id` → `catalogs.id` ON DELETE CASCADE

**Примечание:** В схеме БД поля называются `attributes` и `table_parts`, но в коде они используются как `attributes_xml` и `table_parts_xml` для ясности.

### Связи между таблицами

```
uploads (1) ──< (N) constants
uploads (1) ──< (N) catalogs
catalogs (1) ──< (N) catalog_items
```

При удалении выгрузки (`DELETE FROM uploads`) автоматически удаляются все связанные константы, справочники и элементы справочников благодаря CASCADE.

---

## Форматы данных

### JSON формат

Используется для:
- Списков выгрузок (`GET /api/uploads`)
- Детальной информации о выгрузке (`GET /api/uploads/{uuid}`)
- Проверки передачи (`POST /api/uploads/{uuid}/verify`)
- Ошибок API

**Пример JSON ответа:**
```json
{
  "upload_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "started_at": "2024-01-15T10:30:00Z",
  "status": "completed",
  "total_constants": 15,
  "total_items": 120
}
```

### XML формат

Используется для:
- Получения данных выгрузки (`GET /api/uploads/{uuid}/data`)
- Потоковой передачи (`GET /api/uploads/{uuid}/stream`)
- Приема данных из 1С (все POST эндпоинты)
- Записи нормализованных данных

**Структура XML ответа для `/data`:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<data_response>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <type>all</type>
  <page>1</page>
  <limit>100</limit>
  <total>135</total>
  <items>
    <item type="constant" id="1" created_at="2024-01-15T10:30:15Z">
      <constant>
        <id>1</id>
        <upload_id>1</upload_id>
        <name>Организация</name>
        <synonym>Организация</synonym>
        <type>Строка</type>
        <value>ООО Рога и Копыта</value>
        <created_at>2024-01-15T10:30:15Z</created_at>
      </constant>
    </item>
    <item type="catalog_item" id="5133" created_at="2024-01-15T10:33:43Z">
      <catalog_item>
        <id>5133</id>
        <catalog_id>509</catalog_id>
        <catalog_name>Валюты</catalog_name>
        <reference>BYN</reference>
        <code>933</code>
        <name>BYN</name>
        <attributes_xml>
          <ЗагружаетсяИзИнтернета>Нет</ЗагружаетсяИзИнтернета>
          <НаименованиеПолное>Белорусский рубль</НаименованиеПолное>
        </attributes_xml>
        <table_parts_xml>
          <Представления></Представления>
        </table_parts_xml>
        <created_at>2024-01-15T10:33:43Z</created_at>
      </catalog_item>
    </item>
  </items>
</data_response>
```

**Особенности XML:**
- Все текстовые значения экранируются (`, `&`, `"`, `'`)
- Поля `attributes_xml` и `table_parts_xml` вставляются как innerXML (без дополнительного экранирования)
- Временные метки в формате ISO 8601 (RFC3339)

### Server-Sent Events (SSE) формат

Используется для потоковой передачи данных (`GET /api/uploads/{uuid}/stream`).

**Формат события:**
```
data: <item type="constant" id="1" created_at="2024-01-15T10:30:15Z"><constant>...</constant></item>

data: <item type="catalog_item" id="5133" created_at="2024-01-15T10:33:43Z"><catalog_item>...</catalog_item></item>

data: {"type":"complete"}

```

**Особенности SSE:**
- Каждое событие начинается с префикса `data: `
- События разделяются двойным переводом строки (`\n\n`)
- Заголовки: `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
- Последнее событие `{"type":"complete"}` сигнализирует о завершении потока

---

## Эндпоинты

### 1. Список всех выгрузок

Получить список всех выгрузок с краткой информацией.

**Запрос:**
```
GET /api/uploads
```

**HTTP Метод:** `GET`

**Заголовки:**
- Не требуются

**Query параметры:**
- Отсутствуют

**Описание работы:**
1. Сервер выполняет SQL запрос `SELECT * FROM uploads ORDER BY started_at DESC`
2. Для каждой выгрузки извлекаются все поля из таблицы `uploads`
3. Статистики (`total_constants`, `total_catalogs`, `total_items`) уже хранятся в таблице и обновляются при добавлении данных
4. Результаты сортируются по времени начала выгрузки (новые первыми)
5. Формируется JSON ответ с массивом выгрузок и общим количеством

**Ответ:**
```json
{
  "uploads": [
    {
      "upload_uuid": "550e8400-e29b-41d4-a716-446655440000",
      "started_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:35:00Z",
      "status": "completed",
      "version_1c": "8.3.25",
      "config_name": "УправлениеТорговлей",
      "total_constants": 15,
      "total_catalogs": 5,
      "total_items": 120
    }
  ],
  "total": 1
}
```

**Поля ответа:**
- `uploads` - массив объектов выгрузок
  - `upload_uuid` - уникальный идентификатор выгрузки (UUID v4)
  - `started_at` - время начала выгрузки (ISO 8601, UTC)
  - `completed_at` - время завершения выгрузки (ISO 8601, UTC, может быть `null` для незавершенных выгрузок)
  - `status` - статус выгрузки:
    - `in_progress` - выгрузка в процессе
    - `completed` - выгрузка завершена
  - `version_1c` - версия 1С:Предприятие (например, "8.3.25")
  - `config_name` - имя конфигурации 1С (например, "УправлениеТорговлей")
  - `total_constants` - общее количество констант в выгрузке
  - `total_catalogs` - общее количество справочников в выгрузке
  - `total_items` - общее количество элементов справочников в выгрузке
- `total` - общее количество выгрузок в системе

**Коды ответа:**
- `200 OK` - успешный запрос
- `500 Internal Server Error` - ошибка при получении данных из БД

**Примечания:**
- Выгрузки сортируются по времени начала (новые первыми)
- Незавершенные выгрузки (`status=in_progress`) также включаются в список
- Статистики обновляются автоматически при добавлении данных через эндпоинты `/constant`, `/catalog/meta`, `/catalog/item`

---

### 2. Детальная информация о выгрузке

Получить полную информацию о конкретной выгрузке, включая список справочников и констант.

**Запрос:**
```
GET /api/uploads/{uuid}
```

**HTTP Метод:** `GET`

**Параметры пути:**
- `{uuid}` - UUID выгрузки (обязательный)

**Пример:**
```
GET /api/uploads/550e8400-e29b-41d4-a716-446655440000
```

**Описание работы:**
1. Сервер извлекает UUID из пути запроса
2. Выполняется SQL запрос для поиска выгрузки: `SELECT * FROM uploads WHERE upload_uuid = ?`
3. Если выгрузка не найдена, возвращается ошибка 404
4. Получаются все справочники выгрузки: `SELECT * FROM catalogs WHERE upload_id = ? ORDER BY name`
5. Для каждого справочника подсчитывается количество элементов: `SELECT COUNT(*) FROM catalog_items WHERE catalog_id = ?`
6. Получаются все константы выгрузки: `SELECT * FROM constants WHERE upload_id = ? ORDER BY id`
7. Формируется JSON ответ со всеми данными

**Ответ:**
```json
{
  "upload_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "started_at": "2024-01-15T10:30:00Z",
  "completed_at": "2024-01-15T10:35:00Z",
  "status": "completed",
  "version_1c": "8.3.25",
  "config_name": "УправлениеТорговлей",
  "total_constants": 15,
  "total_catalogs": 5,
  "total_items": 120,
  "catalogs": [
    {
      "id": 1,
      "name": "Номенклатура",
      "synonym": "Номенклатура",
      "item_count": 50,
      "created_at": "2024-01-15T10:31:00Z"
    },
    {
      "id": 2,
      "name": "Контрагенты",
      "synonym": "Контрагенты",
      "item_count": 30,
      "created_at": "2024-01-15T10:31:30Z"
    }
  ],
  "constants": [
    {
      "id": 1,
      "name": "Организация",
      "synonym": "Организация",
      "type": "Строка",
      "value": "ООО Рога и Копыта",
      "created_at": "2024-01-15T10:30:15Z"
    }
  ]
}
```

**Поля ответа:**
- Все поля из списка выгрузок (см. эндпоинт 1)
- `catalogs` - массив справочников:
  - `id` - ID справочника в БД
  - `name` - имя справочника
  - `synonym` - синоним справочника
  - `item_count` - количество элементов в справочнике
  - `created_at` - время создания справочника (ISO 8601, UTC)
- `constants` - массив констант:
  - `id` - ID константы в БД
  - `name` - имя константы
  - `synonym` - синоним константы
  - `type` - тип данных константы
  - `value` - значение константы (может содержать XML)
  - `created_at` - время создания константы (ISO 8601, UTC)

**Коды ответа:**
- `200 OK` - успешный запрос
- `404 Not Found` - выгрузка с указанным UUID не найдена
- `405 Method Not Allowed` - использован неверный HTTP метод
- `500 Internal Server Error` - ошибка при получении данных из БД

**Примечания:**
- Справочники сортируются по имени (`ORDER BY name`)
- Константы сортируются по ID (`ORDER BY id`) - сохраняется порядок вставки
- Поле `value` в константах может содержать сложные XML структуры
- Количество элементов справочника (`item_count`) вычисляется динамически через `COUNT(*)`

---

### 3. Получение данных выгрузки

Получить данные выгрузки с фильтрацией и пагинацией в формате XML.

**Запрос:**
```
GET /api/uploads/{uuid}/data
```

**HTTP Метод:** `GET`

**Параметры пути:**
- `{uuid}` - UUID выгрузки (обязательный)

**Query параметры:**
- `type` - тип данных (опциональный):
  - `all` - все данные (константы + элементы справочников) - **по умолчанию**
  - `constants` - только константы
  - `catalogs` - только элементы справочников
- `catalog_names` - список имен справочников через запятую (опциональный, только для `type=catalogs` или `type=all`)
  - Пример: `Номенклатура,Контрагенты`
  - Имена чувствительны к регистру
  - Если не указан, возвращаются элементы всех справочников
- `page` - номер страницы (опциональный, по умолчанию: `1`)
  - Минимальное значение: `1`
  - Используется для пагинации результатов
- `limit` - количество элементов на странице (опциональный, по умолчанию: `100`)
  - Минимальное значение: `1`
  - Максимальное значение: `1000`
  - Если указано значение больше 1000, автоматически устанавливается 1000

**Описание работы:**

**Шаг 1: Парсинг параметров**
1. Извлекается UUID из пути запроса
2. Парсятся query параметры:
   - `type` - если не указан, устанавливается `all`
   - `catalog_names` - разбивается по запятой, удаляются пробелы
   - `page` - если не указан или < 1, устанавливается 1
   - `limit` - если не указан или < 1, устанавливается 100; если > 1000, устанавливается 1000
3. Вычисляется `offset = (page - 1) * limit`

**Шаг 2: Получение данных в зависимости от типа**

**Если `type=constants`:**
1. Выполняется SQL: `SELECT * FROM constants WHERE upload_id = ? ORDER BY id`
2. Получаются все константы выгрузки
3. Применяется пагинация: берутся элементы с индексами `[offset, offset+limit)`
4. Для каждой константы формируется XML:
   ```xml
   <constant>
     <id>1</id>
     <upload_id>1</upload_id>
     <name>Организация</name>
     <synonym>Организация</synonym>
     <type>Строка</type>
     <value>ООО Рога и Копыта</value>
     <created_at>2024-01-15T10:30:15Z</created_at>
   </constant>
   ```
5. Все текстовые значения экранируются (`, `&`, `"`, `'`)

**Если `type=catalogs`:**
1. Строится SQL запрос с JOIN:
   ```sql
   SELECT ci.id, ci.catalog_id, c.name as catalog_name, 
          ci.reference, ci.code, ci.name, 
          COALESCE(ci.attributes_xml, '') as attributes, 
          COALESCE(ci.table_parts_xml, '') as table_parts, 
          ci.created_at
   FROM catalog_items ci
   INNER JOIN catalogs c ON ci.catalog_id = c.id
   WHERE c.upload_id = ?
   ```
2. Если указан `catalog_names`, добавляется условие: `AND c.name IN (?, ?, ...)`
3. Добавляется сортировка: `ORDER BY ci.id` (сохраняется порядок вставки)
4. Добавляется пагинация: `LIMIT ? OFFSET ?`
5. Выполняется запрос для получения общего количества (для поля `total`)
6. Для каждого элемента формируется XML:
   ```xml
   <catalog_item>
     <id>5133</id>
     <catalog_id>509</catalog_id>
     <catalog_name>Валюты</catalog_name>
     <reference>BYN</reference>
     <code>933</code>
     <name>BYN</name>
     <attributes_xml>...</attributes_xml>
     <table_parts_xml>...</table_parts_xml>
     <created_at>2024-01-15T10:33:43Z</created_at>
   </catalog_item>
   ```
7. Поля `attributes_xml` и `table_parts_xml` вставляются как innerXML (без дополнительного экранирования)
8. Остальные текстовые поля экранируются

**Если `type=all`:**
1. Получаются все константы: `SELECT * FROM constants WHERE upload_id = ? ORDER BY id`
2. Получаются все элементы справочников (с учетом фильтра `catalog_names`, если указан)
3. Объединяются в один список:
   - Сначала все константы (в порядке ID)
   - Затем все элементы справочников (в порядке ID)
4. Применяется пагинация к объединенному списку
5. Формируется XML для каждого элемента

**Шаг 3: Формирование XML ответа**
1. Создается корневой элемент `<data_response>`
2. Добавляются метаданные:
   - `<upload_uuid>` - UUID выгрузки
   - `<type>` - тип запрошенных данных
   - `<page>` - текущая страница
   - `<limit>` - лимит элементов
   - `<total>` - общее количество элементов (до пагинации)
3. Создается контейнер `<items>`
4. Каждый элемент оборачивается в `<item>` с атрибутами:
   - `type` - тип элемента (`constant` или `catalog_item`)
   - `id` - ID элемента в БД
   - `created_at` - время создания
5. Внутри `<item>` вставляется XML данных элемента (как innerXML)
6. Весь XML форматируется с отступами и возвращается клиенту

**Примеры:**

1. Получить все данные (константы + элементы справочников):
```
GET /api/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=all&page=1&limit=50
```

2. Получить только константы:
```
GET /api/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=constants
```

3. Получить элементы конкретных справочников:
```
GET /api/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=catalogs&catalog_names=Номенклатура,Контрагенты&page=1&limit=100
```

**Ответ (XML формат):**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<data_response>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <type>catalogs</type>
  <page>1</page>
  <limit>100</limit>
  <total>883</total>
  <items>
    <item type="catalog_item" id="5133" created_at="2025-11-09T10:33:43Z">
      <catalog_item>
        <id>5133</id>
        <catalog_id>509</catalog_id>
        <catalog_name>Валюты</catalog_name>
        <reference>BYN</reference>
        <code>933</code>
        <name>BYN</name>
        <attributes_xml>
          <ЗагружаетсяИзИнтернета>Нет</ЗагружаетсяИзИнтернета>
          <НаименованиеПолное>Белорусский рубль</НаименованиеПолное>
          <Наценка>0</Наценка>
        </attributes_xml>
        <table_parts_xml>
          <Представления></Представления>
        </table_parts_xml>
        <created_at>2025-11-09T10:33:43Z</created_at>
      </catalog_item>
    </item>
    <item type="constant" id="1" created_at="2025-11-09T10:30:15Z">
      <constant>
        <id>1</id>
        <upload_id>1</upload_id>
        <name>Организация</name>
        <synonym>Организация</synonym>
        <type>Строка</type>
        <value>ООО Рога и Копыта</value>
        <created_at>2025-11-09T10:30:15Z</created_at>
      </constant>
    </item>
  </items>
</data_response>
```

**Поля ответа:**
- `upload_uuid` - UUID выгрузки
- `type` - тип запрошенных данных (`all`, `constants`, `catalogs`)
- `page` - текущая страница
- `limit` - лимит элементов на странице
- `total` - общее количество элементов
- `items` - массив элементов данных
  - `item` - элемент данных с атрибутами:
    - `type` - тип элемента (`constant` или `catalog_item`)
    - `id` - ID элемента в базе данных
    - `created_at` - время создания
  - Внутри `item` содержится полная XML структура данных:
    - Для `constant`: `id`, `upload_id`, `name`, `synonym`, `type`, `value`, `created_at`
    - Для `catalog_item`: `id`, `catalog_id`, `catalog_name`, `reference`, `code`, `name`, `attributes_xml`, `table_parts_xml`, `created_at`

**Примечания:**
- Элементы возвращаются в порядке вставки в БД (по возрастанию ID)
- Для `type=all` сначала идут константы, затем элементы справочников (в порядке их ID)
- Поля `attributes_xml` и `table_parts_xml` содержат XML данные в том же формате, как они хранятся в БД
- Поле `catalog_name` всегда присутствует в элементах справочников и содержит имя справочника
- При `type=all` пагинация применяется к объединенному списку

---

### 4. Потоковая отправка данных (Server-Sent Events)

Получить данные выгрузки в режиме потоковой передачи через Server-Sent Events (SSE). Каждый элемент отправляется отдельным событием.

**Запрос:**
```
GET /api/uploads/{uuid}/stream
```

**Query параметры:**
- `type` - тип данных: `all`, `constants`, `catalogs`. По умолчанию: `all`
- `catalog_names` - список имен справочников через запятую

**Пример:**
```
GET /api/uploads/550e8400-e29b-41d4-a716-446655440000/stream?type=all&catalog_names=Номенклатура
```

**Ответ (SSE формат с XML данными):**
```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

data: <item type="catalog_item" id="5133" created_at="2025-11-09T10:33:43Z"><catalog_item><id>5133</id><catalog_id>509</catalog_id><catalog_name>Валюты</catalog_name><reference>BYN</reference><code>933</code><name>BYN</name><attributes_xml><ЗагружаетсяИзИнтернета>Нет</ЗагружаетсяИзИнтернета><НаименованиеПолное>Белорусский рубль</НаименованиеПолное></attributes_xml><table_parts_xml><Представления></Представления></table_parts_xml><created_at>2025-11-09T10:33:43Z</created_at></catalog_item></item>

data: <item type="constant" id="1" created_at="2025-11-09T10:30:15Z"><constant><id>1</id><upload_id>1</upload_id><name>Организация</name><synonym>Организация</synonym><type>Строка</type><value>ООО Рога и Копыта</value><created_at>2025-11-09T10:30:15Z</created_at></constant></item>

data: <item type="complete"></item>

```

**Использование в JavaScript:**
```javascript
const eventSource = new EventSource(
  'http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000/stream?type=all'
);

eventSource.onmessage = function(event) {
  const parser = new DOMParser();
  const xmlDoc = parser.parseFromString(event.data, 'text/xml');
  const item = xmlDoc.documentElement;
  
  if (item.getAttribute('type') === 'complete') {
    eventSource.close();
    console.log('Stream completed');
  } else {
    console.log('Received item:', item);
    // Обработка XML элемента
    const type = item.getAttribute('type');
    const id = item.getAttribute('id');
    // Парсинг внутренних данных элемента
  }
};

eventSource.onerror = function(error) {
  console.error('Stream error:', error);
  eventSource.close();
};
```

**Использование в Python:**
```python
import requests
from xml.etree import ElementTree as ET

url = 'http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000/stream?type=all'

with requests.get(url, stream=True) as response:
    for line in response.iter_lines():
        if line:
            line_str = line.decode('utf-8')
            if line_str.startswith('data: '):
                data_str = line_str[6:]  # Убираем префикс "data: "
                try:
                    item = ET.fromstring(data_str)
                    if item.get('type') == 'complete':
                        break
                    print(f"Received item: type={item.get('type')}, id={item.get('id')}")
                    # Обработка XML элемента
                except ET.ParseError as e:
                    print(f"XML parse error: {e}")
```

**Примечания:**
- Каждый элемент отправляется в формате `data: <item>...</item>\n\n` (XML)
- Элементы возвращаются в порядке вставки в БД (по возрастанию ID)
- В конце потока отправляется событие `<item type="complete"></item>`
- Поток автоматически закрывается после завершения
- Поле `catalog_name` всегда присутствует в элементах справочников
- Поля `attributes_xml` и `table_parts_xml` содержат XML данные в том же формате, как они хранятся в БД

---

### 5. Проверка успешной передачи

Проверить, что все элементы выгрузки были успешно получены.

**Запрос:**
```
POST /api/uploads/{uuid}/verify
Content-Type: application/json
```

**Тело запроса:**
```json
{
  "received_ids": [1, 2, 3, 10, 11, 12, 20, 21, 22]
}
```

**Параметры:**
- `received_ids` - массив ID элементов, которые были получены

**Ответ:**
```json
{
  "upload_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "expected_total": 135,
  "received_count": 9,
  "missing_ids": [4, 5, 6, 7, 8, 9, 13, 14, 15, 16, 17, 18, 19, 23, 24, 25],
  "is_complete": false,
  "message": "Received 9 of 135 items, 16 items missing"
}
```

**Успешный ответ (все получено):**
```json
{
  "upload_uuid": "550e8400-e29b-41d4-a716-446655440000",
  "expected_total": 135,
  "received_count": 135,
  "missing_ids": [],
  "is_complete": true,
  "message": "Received 135 of 135 items, all items received"
}
```

**Поля ответа:**
- `upload_uuid` - UUID выгрузки
- `expected_total` - общее количество элементов в выгрузке
- `received_count` - количество полученных элементов
- `missing_ids` - массив ID отсутствующих элементов (пустой, если все получено)
- `is_complete` - флаг, указывающий что все элементы получены
- `message` - текстовое сообщение о результате проверки

---

## Обработка ошибок

Все ошибки возвращаются в формате JSON с кодом состояния HTTP.

**Формат ошибки:**
```json
{
  "error": "Описание ошибки",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

**Коды состояния:**
- `200 OK` - успешный запрос
- `400 Bad Request` - неверный формат запроса
- `404 Not Found` - выгрузка не найдена
- `405 Method Not Allowed` - неверный HTTP метод
- `500 Internal Server Error` - внутренняя ошибка сервера

**Примеры ошибок:**

1. Выгрузка не найдена:
```json
{
  "error": "Upload not found",
  "timestamp": "2024-01-15T10:30:00Z"
}
```

2. Неверный метод:
```
HTTP/1.1 405 Method Not Allowed
Method not allowed
```

---

## Примеры использования

### Пример 1: Получить список выгрузок и выбрать одну

```bash
# Получить список выгрузок
curl http://localhost:9999/api/uploads

# Получить детали конкретной выгрузки
curl http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000
```

### Пример 2: Получить все константы выгрузки

```bash
curl "http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=constants"
```

### Пример 3: Получить элементы конкретных справочников с пагинацией

```bash
curl "http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=catalogs&catalog_names=Номенклатура,Контрагенты&page=1&limit=50"
```

### Пример 4: Потоковая загрузка всех данных

```bash
curl "http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000/stream?type=all"
```

### Пример 5: Проверка передачи данных

```bash
curl -X POST http://localhost:9999/api/uploads/550e8400-e29b-41d4-a716-446655440000/verify \
  -H "Content-Type: application/json" \
  -d '{"received_ids": [1, 2, 3, 10, 11, 12]}'
```

### Пример 6: Полный цикл работы с API (Python)

```python
import requests
import json

BASE_URL = "http://localhost:9999"

# 1. Получить список выгрузок
response = requests.get(f"{BASE_URL}/api/uploads")
uploads = response.json()["uploads"]
print(f"Найдено выгрузок: {len(uploads)}")

# 2. Выбрать первую выгрузку
upload_uuid = uploads[0]["upload_uuid"]
print(f"Выбранная выгрузка: {upload_uuid}")

# 3. Получить детали выгрузки
response = requests.get(f"{BASE_URL}/api/uploads/{upload_uuid}")
details = response.json()
print(f"Констант: {details['total_constants']}")
print(f"Справочников: {details['total_catalogs']}")
print(f"Элементов: {details['total_items']}")

# 4. Получить все константы (XML формат)
from xml.etree import ElementTree as ET

response = requests.get(f"{BASE_URL}/api/uploads/{upload_uuid}/data?type=constants")
xml_data = ET.fromstring(response.text)
items = xml_data.findall('.//item')
print(f"Получено констант: {len(items)}")

# 5. Получить элементы справочника "Номенклатура" с пагинацией (XML формат)
page = 1
limit = 50
all_items = []
while True:
    response = requests.get(
        f"{BASE_URL}/api/uploads/{upload_uuid}/data",
        params={
            "type": "catalogs",
            "catalog_names": "Номенклатура",
            "page": page,
            "limit": limit
        }
    )
    xml_data = ET.fromstring(response.text)
    items = xml_data.findall('.//item')
    all_items.extend(items)
    
    if len(items) < limit:
        break
    page += 1

print(f"Получено элементов справочника: {len(all_items)}")

# 6. Проверить передачу
received_ids = [int(item.get('id')) for item in all_items]
response = requests.post(
    f"{BASE_URL}/api/uploads/{upload_uuid}/verify",
    json={"received_ids": received_ids}
)
verify_result = response.json()
print(f"Проверка: {verify_result['message']}")
print(f"Получено: {verify_result['received_count']} из {verify_result['expected_total']}")
```

### Пример 7: Потоковая загрузка (JavaScript)

```javascript
async function streamUploadData(uploadUuid) {
  const url = `http://localhost:9999/api/uploads/${uploadUuid}/stream?type=all`;
  const eventSource = new EventSource(url);
  
  const receivedItems = [];
  
  return new Promise((resolve, reject) => {
    eventSource.onmessage = (event) => {
      const parser = new DOMParser();
      const xmlDoc = parser.parseFromString(event.data, 'text/xml');
      const item = xmlDoc.documentElement;
      
      const itemType = item.getAttribute('type');
      if (itemType === 'complete') {
        eventSource.close();
        resolve(receivedItems);
      } else {
        receivedItems.push(item);
        const itemId = item.getAttribute('id');
        console.log(`Получен элемент: ${itemType} #${itemId}`);
      }
    };
    
    eventSource.onerror = (error) => {
      eventSource.close();
      reject(error);
    };
  });
}

// Использование
streamUploadData('550e8400-e29b-41d4-a716-446655440000')
  .then(items => {
    console.log(`Всего получено элементов: ${items.length}`);
  })
  .catch(error => {
    console.error('Ошибка при потоковой загрузке:', error);
  });
```

---

## Нормализованная база данных

Сервер поддерживает работу с двумя базами данных:
- **Основная БД** (`1c_data.db` или `data.db`) - для хранения исходных данных из 1С
- **Нормализованная БД** (`normalized_data.db`) - для хранения нормализованных данных, готовых для загрузки обратно в 1С

Обе БД имеют идентичную структуру, что позволяет без проблем загружать данные из нормализованной БД обратно в 1С.

### Рабочий процесс

1. **Получение данных из основной БД** - внешний ресурс получает исходные данные через `/api/uploads/{uuid}/data` или `/stream`
2. **Нормализация данных** - внешний ресурс обрабатывает и нормализует данные
3. **Запись нормализованных данных** - нормализованные данные отправляются через `/api/normalized/upload/*`
4. **Получение нормализованных данных** - данные можно получить через `/api/normalized/uploads/{uuid}/data` для загрузки в 1С

---

## Эндпоинты для нормализованной БД

### 6. Список выгрузок из нормализованной БД

Получить список всех выгрузок из нормализованной БД.

**Запрос:**
```
GET /api/normalized/uploads
```

**Ответ:**
```json
{
  "uploads": [
    {
      "upload_uuid": "550e8400-e29b-41d4-a716-446655440000",
      "started_at": "2024-01-15T10:30:00Z",
      "completed_at": "2024-01-15T10:35:00Z",
      "status": "completed",
      "version_1c": "8.3.25",
      "config_name": "УправлениеТорговлей",
      "total_constants": 15,
      "total_catalogs": 5,
      "total_items": 120
    }
  ],
  "total": 1
}
```

**Поля ответа:** Аналогичны эндпоинту `/api/uploads`

---

### 7. Детальная информация о выгрузке из нормализованной БД

Получить полную информацию о конкретной выгрузке из нормализованной БД.

**Запрос:**
```
GET /api/normalized/uploads/{uuid}
```

**Пример:**
```
GET /api/normalized/uploads/550e8400-e29b-41d4-a716-446655440000
```

**Ответ:** Аналогичен эндпоинту `/api/uploads/{uuid}` (JSON формат)

---

### 8. Получение данных из нормализованной БД

Получить данные выгрузки из нормализованной БД с фильтрацией и пагинацией.

**Запрос:**
```
GET /api/normalized/uploads/{uuid}/data
```

**Query параметры:** Аналогичны эндпоинту `/api/uploads/{uuid}/data`

**Пример:**
```
GET /api/normalized/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=all&page=1&limit=50
```

**Ответ:** XML формат, аналогичен эндпоинту `/api/uploads/{uuid}/data`

---

### 9. Потоковая отправка данных из нормализованной БД

Получить данные выгрузки из нормализованной БД в режиме потоковой передачи.

**Запрос:**
```
GET /api/normalized/uploads/{uuid}/stream
```

**Query параметры:** Аналогичны эндпоинту `/api/uploads/{uuid}/stream`

**Пример:**
```
GET /api/normalized/uploads/550e8400-e29b-41d4-a716-446655440000/stream?type=all
```

**Ответ:** SSE формат с XML данными, аналогичен эндпоинту `/api/uploads/{uuid}/stream`

---

### 10. Проверка передачи данных из нормализованной БД

Проверить, что все элементы выгрузки из нормализованной БД были успешно получены.

**Запрос:**
```
POST /api/normalized/uploads/{uuid}/verify
Content-Type: application/json
```

**Тело запроса и ответ:** Аналогичны эндпоинту `/api/uploads/{uuid}/verify`

---

## Эндпоинты для записи нормализованных данных

### 11. Handshake (начало выгрузки нормализованных данных)

Создать новую выгрузку в нормализованной БД.

**Запрос:**
```
POST /api/normalized/upload/handshake
Content-Type: application/xml
```

**Тело запроса:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<handshake>
  <version_1c>8.3.25</version_1c>
  <config_name>УправлениеТорговлей</config_name>
  <timestamp>2024-01-15T10:30:00Z</timestamp>
</handshake>
```

**Ответ:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<handshake_response>
  <success>true</success>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <message>Normalized handshake successful</message>
  <timestamp>2024-01-15T10:30:00Z</timestamp>
</handshake_response>
```

**Важно:** Сохраните `upload_uuid` из ответа для последующих запросов.

---

### 12. Metadata (метаданные)

Отправить метаданные выгрузки.

**Запрос:**
```
POST /api/normalized/upload/metadata
Content-Type: application/xml
```

**Тело запроса:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <version_1c>8.3.25</version_1c>
  <config_name>УправлениеТорговлей</config_name>
  <timestamp>2024-01-15T10:30:00Z</timestamp>
</metadata>
```

**Ответ:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<metadata_response>
  <success>true</success>
  <message>Normalized metadata received successfully</message>
  <timestamp>2024-01-15T10:30:00Z</timestamp>
</metadata_response>
```

---

### 13. Constant (константа)

Добавить константу в нормализованную БД.

**Запрос:**
```
POST /api/normalized/upload/constant
Content-Type: application/xml
```

**Тело запроса:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<constant>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <name>Организация</name>
  <synonym>Организация</synonym>
  <type>Строка</type>
  <value>ООО Рога и Копыта</value>
  <timestamp>2024-01-15T10:30:15Z</timestamp>
</constant>
```

**Ответ:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<constant_response>
  <success>true</success>
  <message>Normalized constant added successfully</message>
  <timestamp>2024-01-15T10:30:15Z</timestamp>
</constant_response>
```

---

### 14. Catalog Meta (метаданные справочника)

Добавить метаданные справочника в нормализованную БД.

**Запрос:**
```
POST /api/normalized/upload/catalog/meta
Content-Type: application/xml
```

**Тело запроса:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<catalog_meta>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <name>Номенклатура</name>
  <synonym>Номенклатура</synonym>
  <timestamp>2024-01-15T10:31:00Z</timestamp>
</catalog_meta>
```

**Ответ:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<catalog_meta_response>
  <success>true</success>
  <catalog_id>1</catalog_id>
  <message>Normalized catalog metadata added successfully</message>
  <timestamp>2024-01-15T10:31:00Z</timestamp>
</catalog_meta_response>
```

---

### 15. Catalog Item (элемент справочника)

Добавить элемент справочника в нормализованную БД.

**Запрос:**
```
POST /api/normalized/upload/catalog/item
Content-Type: application/xml
```

**Тело запроса:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<catalog_item>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <catalog_name>Номенклатура</catalog_name>
  <reference>Справочник.Номенклатура.12345</reference>
  <code>00001</code>
  <name>Товар 1</name>
  <attributes>
    <ЕдиницаИзмерения>шт</ЕдиницаИзмерения>
    <Вес>10.5</Вес>
  </attributes>
  <table_parts>
    <ДополнительныеРеквизиты></ДополнительныеРеквизиты>
  </table_parts>
  <timestamp>2024-01-15T10:31:00Z</timestamp>
</catalog_item>
```

**Ответ:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<catalog_item_response>
  <success>true</success>
  <message>Normalized catalog item added successfully</message>
  <timestamp>2024-01-15T10:31:00Z</timestamp>
</catalog_item_response>
```

**Примечания:**
- Поля `attributes` и `table_parts` должны содержать XML данные в том же формате, как они хранятся в БД
- Поле `catalog_name` должно соответствовать имени справочника, созданного через `/catalog/meta`

---

### 16. Complete (завершение выгрузки)

Завершить выгрузку нормализованных данных.

**Запрос:**
```
POST /api/normalized/upload/complete
Content-Type: application/xml
```

**Тело запроса:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<complete>
  <upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid>
  <timestamp>2024-01-15T10:35:00Z</timestamp>
</complete>
```

**Ответ:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<complete_response>
  <success>true</success>
  <message>Normalized upload completed successfully</message>
  <timestamp>2024-01-15T10:35:00Z</timestamp>
</complete_response>
```

**Важно:** После завершения выгрузки данные готовы для получения через API и загрузки в 1С.

---

## Примеры использования нормализованной БД

### Пример 8: Полный цикл работы с нормализованной БД

```bash
# 1. Создать новую выгрузку (handshake)
curl -X POST http://localhost:9999/api/normalized/upload/handshake \
  -H "Content-Type: application/xml" \
  -d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><handshake><version_1c>8.3.25</version_1c><config_name>Тест</config_name><timestamp>2024-01-15T10:30:00Z</timestamp></handshake>"

# Сохраните upload_uuid из ответа (например: 550e8400-e29b-41d4-a716-446655440000)

# 2. Отправить метаданные
curl -X POST http://localhost:9999/api/normalized/upload/metadata \
  -H "Content-Type: application/xml" \
  -d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><metadata><upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid><version_1c>8.3.25</version_1c><config_name>Тест</config_name><timestamp>2024-01-15T10:30:00Z</timestamp></metadata>"

# 3. Добавить константу
curl -X POST http://localhost:9999/api/normalized/upload/constant \
  -H "Content-Type: application/xml" \
  -d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><constant><upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid><name>Организация</name><synonym>Организация</synonym><type>Строка</type><value>ООО Рога и Копыта</value><timestamp>2024-01-15T10:30:15Z</timestamp></constant>"

# 4. Добавить метаданные справочника
curl -X POST http://localhost:9999/api/normalized/upload/catalog/meta \
  -H "Content-Type: application/xml" \
  -d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><catalog_meta><upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid><name>Номенклатура</name><synonym>Номенклатура</synonym><timestamp>2024-01-15T10:31:00Z</timestamp></catalog_meta>"

# 5. Добавить элемент справочника
curl -X POST http://localhost:9999/api/normalized/upload/catalog/item \
  -H "Content-Type: application/xml" \
  -d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><catalog_item><upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid><catalog_name>Номенклатура</catalog_name><reference>Справочник.Номенклатура.12345</reference><code>00001</code><name>Товар 1</name><attributes><ЕдиницаИзмерения>шт</ЕдиницаИзмерения></attributes><table_parts></table_parts><timestamp>2024-01-15T10:31:00Z</timestamp></catalog_item>"

# 6. Завершить выгрузку
curl -X POST http://localhost:9999/api/normalized/upload/complete \
  -H "Content-Type: application/xml" \
  -d "<?xml version=\"1.0\" encoding=\"UTF-8\"?><complete><upload_uuid>550e8400-e29b-41d4-a716-446655440000</upload_uuid><timestamp>2024-01-15T10:35:00Z</timestamp></complete>"

# 7. Получить список выгрузок из нормализованной БД
curl http://localhost:9999/api/normalized/uploads

# 8. Получить данные из нормализованной БД
curl "http://localhost:9999/api/normalized/uploads/550e8400-e29b-41d4-a716-446655440000/data?type=all"
```

### Пример 9: Полный цикл с нормализацией данных (Python)

```python
import requests
from xml.etree import ElementTree as ET

BASE_URL = "http://localhost:9999"

# ===== ШАГ 1: Получение данных из основной БД =====
print("1. Получение данных из основной БД...")

# Получить список выгрузок
response = requests.get(f"{BASE_URL}/api/uploads")
uploads = response.json()["uploads"]
if not uploads:
    print("Нет выгрузок в основной БД")
    exit(1)

upload_uuid = uploads[0]["upload_uuid"]
print(f"   Выбрана выгрузка: {upload_uuid}")

# Получить данные
response = requests.get(f"{BASE_URL}/api/uploads/{upload_uuid}/data?type=all")
xml_data = ET.fromstring(response.text)
items = xml_data.findall('.//item')
print(f"   Получено элементов: {len(items)}")

# ===== ШАГ 2: Нормализация данных (пример) =====
print("\n2. Нормализация данных...")
# Здесь должна быть логика нормализации
# Для примера просто используем те же данные
normalized_items = items
print(f"   Нормализовано элементов: {len(normalized_items)}")

# ===== ШАГ 3: Запись нормализованных данных =====
print("\n3. Запись нормализованных данных...")

# Handshake
handshake_xml = """<?xml version="1.0" encoding="UTF-8"?>
<handshake>
  <version_1c>8.3.25</version_1c>
  <config_name>НормализованнаяКонфигурация</config_name>
  <timestamp>2024-01-15T10:30:00Z</timestamp>
</handshake>"""

response = requests.post(
    f"{BASE_URL}/api/normalized/upload/handshake",
    headers={"Content-Type": "application/xml"},
    data=handshake_xml.encode('utf-8')
)
handshake_response = ET.fromstring(response.text)
normalized_uuid = handshake_response.find('upload_uuid').text
print(f"   Создана выгрузка: {normalized_uuid}")

# Metadata
metadata_xml = f"""<?xml version="1.0" encoding="UTF-8"?>
<metadata>
  <upload_uuid>{normalized_uuid}</upload_uuid>
  <version_1c>8.3.25</version_1c>
  <config_name>НормализованнаяКонфигурация</config_name>
  <timestamp>2024-01-15T10:30:00Z</timestamp>
</metadata>"""

requests.post(
    f"{BASE_URL}/api/normalized/upload/metadata",
    headers={"Content-Type": "application/xml"},
    data=metadata_xml.encode('utf-8')
)

# Запись констант и элементов (упрощенный пример)
# В реальном сценарии нужно парсить XML и отправлять каждый элемент отдельно

# Complete
complete_xml = f"""<?xml version="1.0" encoding="UTF-8"?>
<complete>
  <upload_uuid>{normalized_uuid}</upload_uuid>
  <timestamp>2024-01-15T10:35:00Z</timestamp>
</complete>"""

requests.post(
    f"{BASE_URL}/api/normalized/upload/complete",
    headers={"Content-Type": "application/xml"},
    data=complete_xml.encode('utf-8')
)

print("   Выгрузка завершена")

# ===== ШАГ 4: Проверка нормализованных данных =====
print("\n4. Проверка нормализованных данных...")

response = requests.get(f"{BASE_URL}/api/normalized/uploads/{normalized_uuid}/data?type=all")
xml_data = ET.fromstring(response.text)
normalized_items = xml_data.findall('.//item')
print(f"   Получено элементов из нормализованной БД: {len(normalized_items)}")
print(f"   Данные готовы для загрузки в 1С")
```

### Пример 10: Использование тестового скрипта

Для быстрого тестирования используйте готовый PowerShell скрипт:

```powershell
.\test_normalized.ps1
```

Скрипт автоматически выполнит все шаги:
1. Создание выгрузки
2. Запись метаданных
3. Запись константы
4. Запись справочника
5. Запись элемента справочника
6. Завершение выгрузки
7. Проверка данных через API

---

## Примечания

1. **Идентификация элементов:** Все элементы (константы и элементы справочников) имеют уникальный `id` в базе данных, который используется для проверки передачи.

2. **Формат данных:** 
   - Эндпоинты `/api/uploads` и `/api/uploads/{uuid}` возвращают данные в формате JSON
   - Эндпоинты `/api/uploads/{uuid}/data` и `/api/uploads/{uuid}/stream` возвращают данные в формате XML
   - XML формат соответствует структуре данных в базе данных

3. **Порядок элементов:** Элементы возвращаются в порядке вставки в БД (по возрастанию ID), что гарантирует сохранение исходного порядка данных из 1С.

4. **Поле catalog_name:** В элементах справочников всегда присутствует поле `catalog_name`, которое содержит имя справочника. Это поле берется из таблицы `catalogs` и гарантированно заполнено для всех элементов.

5. **XML в атрибутах:** Поля `attributes_xml` и `table_parts_xml` в элементах справочников содержат XML данные в том же формате, как они хранятся в базе данных. Эти данные не требуют дополнительного парсинга и могут быть использованы напрямую.

6. **Пагинация:** При работе с большими объемами данных рекомендуется использовать пагинацию. Максимальный `limit` - 1000 элементов.

7. **Потоковая передача:** Для больших объемов данных рекомендуется использовать эндпоинт `/stream`, который отправляет данные поэлементно в формате XML через Server-Sent Events и не требует загрузки всего объема в память.

8. **Фильтрация справочников:** Параметр `catalog_names` принимает список имен справочников через запятую. Имена чувствительны к регистру и должны точно совпадать с именами в базе данных.

9. **Временные метки:** Все временные метки возвращаются в формате ISO 8601 (UTC).

10. **Нормализованная БД:** Сервер поддерживает работу с двумя базами данных:
    - Основная БД (`1c_data.db` или `data.db`) - для исходных данных из 1С
    - Нормализованная БД (`normalized_data.db`) - для нормализованных данных, готовых для загрузки в 1С
    - Обе БД имеют идентичную структуру, что позволяет загружать данные из нормализованной БД обратно в 1С
    - Эндпоинты для нормализованной БД работают аналогично основным эндпоинтам, но используют префикс `/api/normalized/`

11. **Запись нормализованных данных:** 
    - Все эндпоинты для записи нормализованных данных принимают XML формат
    - Порядок вызовов: `handshake` → `metadata` → `constant`/`catalog/meta`/`catalog/item` → `complete`
    - После завершения выгрузки (`complete`) данные доступны через API для получения
    - Поля `attributes` и `table_parts` в элементах справочников должны содержать XML данные в том же формате, как они хранятся в БД

---

## Технические детали

### Обработка XML

#### Парсинг вложенного XML в константах

Сервер использует кастомный парсер для обработки значения константы, которое может содержать сложные XML структуры. Парсер (`ConstantValue.UnmarshalXML`) работает следующим образом:

1. Отслеживает глубину вложенности XML элементов
2. Сохраняет все XML структуры как есть, включая:
   - Открывающие и закрывающие теги
   - Атрибуты элементов
   - Текстовое содержимое
3. Не выполняет дополнительного экранирования (данные уже в правильном формате)

**Пример сложного значения константы:**
```xml
<value>
  <Структура>
    <Поле1>Значение1</Поле1>
    <Поле2 Тип="Строка">Значение2</Поле2>
  </Структура>
</value>
```

Такое значение сохраняется в БД как строка и может быть извлечено без потери структуры.

#### Экранирование XML

При формировании XML ответов сервер экранирует специальные символы в текстовых полях:
- `&` → `&amp;`
- `<` → `&lt;`
- `>` → `&gt;`
- `"` → `&quot;`
- `'` → `&apos;`

**Исключение:** Поля `attributes_xml` и `table_parts_xml` в элементах справочников вставляются как innerXML без дополнительного экранирования, так как они уже содержат валидный XML.

### Работа с базой данных

#### Транзакции

Все операции записи выполняются в отдельных транзакциях:
- Каждая константа добавляется в отдельной транзакции
- Каждый элемент справочника добавляется в отдельной транзакции
- При ошибке транзакция откатывается, данные не сохраняются

#### Обновление счетчиков

Счетчики в таблице `uploads` обновляются автоматически:
- При добавлении константы: `total_constants = total_constants + 1`
- При добавлении справочника: `total_catalogs = total_catalogs + 1`
- При добавлении элемента справочника: `total_items = total_items + 1`

Обновление происходит в той же транзакции, что и добавление данных, что гарантирует консистентность.

#### Индексы

Для оптимизации запросов используются следующие индексы:
- `idx_uploads_uuid` - для быстрого поиска выгрузки по UUID
- `idx_constants_upload_id` - для быстрого получения констант выгрузки
- `idx_catalogs_upload_id` - для быстрого получения справочников выгрузки
- `idx_catalog_items_catalog_id` - для быстрого получения элементов справочника
- `idx_catalog_items_reference` - для быстрого поиска элемента по ссылке

### Потоковая передача (SSE)

#### Механизм работы

1. **Инициализация:**
   - Устанавливаются заголовки для SSE
   - Проверяется поддержка `http.Flusher` интерфейса
   - Открывается поток для записи

2. **Отправка данных:**
   - Данные получаются порциями (batch) для оптимизации
   - Каждый элемент сериализуется в XML
   - Отправляется через `fmt.Fprintf(w, "data: %s\n\n", xmlData)`
   - Вызывается `flusher.Flush()` для немедленной отправки

3. **Завершение:**
   - Отправляется завершающее событие `{"type":"complete"}`
   - Поток закрывается автоматически

#### Обработка ошибок

При ошибке во время потоковой передачи:
- Соединение закрывается
- Клиент получает событие ошибки (если поддерживается)
- Незавершенные данные не отправляются

### Производительность

#### Оптимизации

1. **Пагинация:**
   - Максимальный `limit` ограничен 1000 элементами
   - Используется SQL `LIMIT` и `OFFSET` для эффективной выборки

2. **JOIN запросы:**
   - Для получения элементов справочников используется `INNER JOIN` вместо множественных запросов
   - Имя справочника (`catalog_name`) получается через JOIN, а не отдельным запросом

3. **Индексы:**
   - Все внешние ключи индексированы
   - UUID выгрузки индексирован для быстрого поиска

4. **Память:**
   - При потоковой передаче данные не загружаются полностью в память
   - Используется batch обработка (100 элементов за раз)

#### Рекомендации

1. **Для больших выгрузок:**
   - Используйте потоковую передачу (`/stream`) вместо `/data`
   - Обрабатывайте данные по мере поступления

2. **Для фильтрации:**
   - Используйте параметр `catalog_names` для получения только нужных справочников
   - Используйте `type=constants` или `type=catalogs` вместо `type=all`, если нужен только один тип

3. **Для пагинации:**
   - Используйте разумный `limit` (100-500 элементов)
   - Не делайте слишком много запросов с маленьким `limit`

### Безопасность

#### Валидация входных данных

1. **UUID:**
   - Проверяется формат UUID перед использованием в запросах
   - Несуществующие UUID возвращают 404 ошибку

2. **Query параметры:**
   - Все числовые параметры валидируются (минимальные/максимальные значения)
   - Строковые параметры очищаются от пробелов

3. **SQL инъекции:**
   - Все параметры передаются через prepared statements
   - Используется параметризованные запросы

#### CORS

CORS настроен для разрешения запросов с любых источников:
- `Access-Control-Allow-Origin: *`
- Поддерживаются методы: `GET`, `POST`, `OPTIONS`
- Разрешены заголовки: `Content-Type`

**Внимание:** В production окружении рекомендуется ограничить CORS конкретными доменами.

### Логирование

Сервер логирует все операции:
- Создание выгрузок
- Добавление данных
- API запросы
- Ошибки

Логи включают:
- Временную метку
- Уровень (INFO, DEBUG, ERROR)
- Сообщение
- UUID выгрузки (если применимо)
- Эндпоинт

Логи доступны через канал `logChan` для интеграции с GUI или внешними системами логирования.

### Ограничения

1. **Размер данных:**
   - Максимальный размер одного запроса ограничен настройками HTTP сервера
   - Рекомендуется использовать потоковую передачу для больших объемов

2. **Количество элементов:**
   - Теоретически не ограничено
   - Практически ограничено размером SQLite БД (до нескольких ГБ)

3. **Параллельные запросы:**
   - SQLite поддерживает параллельное чтение
   - Запись выполняется последовательно (SQLite limitation)

---

## CORS

API поддерживает CORS для всех источников. Заголовки:
- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type`

**Внимание:** В production окружении рекомендуется ограничить CORS конкретными доменами для безопасности.

---

## Эндпоинты для обработки номенклатуры

### POST /api/nomenclature/process

Запускает многопоточную обработку номенклатуры с использованием ИИ (Arliai API) для нормализации наименований и классификации по КПВЭД.

**Описание:**
- Обрабатывает записи из таблицы `catalog_items` в нормализованной БД (`normalized_data.db`)
- Использует 2 параллельных потока (goroutine) для запросов к Arliai API
- Для каждой записи определяет нормализованное наименование и код КПВЭД
- Результаты сохраняются обратно в базу данных

**Метод:** `POST`

**Заголовки:**
- `Content-Type: application/json` (опционально)

**Тело запроса:**
Не требуется (все параметры берутся из конфигурации)

**Ответ:**

**Успешный ответ (200 OK):**
```json
{
  "status": "processing_started",
  "message": "Обработка номенклатуры запущена"
}
```

**Ошибки:**

**500 Internal Server Error** - если не установлена переменная окружения `ARLIAI_API_KEY`:
```json
{
  "error": "ARLIAI_API_KEY environment variable not set"
}
```

**500 Internal Server Error** - если не удалось создать процессор:
```json
{
  "error": "Failed to create processor: <описание ошибки>"
}
```

**Пример использования:**

```bash
curl -X POST http://localhost:9999/api/nomenclature/process
```

**Примечания:**
- Обработка запускается асинхронно в отдельной горутине
- Для работы требуется установленная переменная окружения `ARLIAI_API_KEY`
- Используется модель ИИ: `GLM-4.5-Air`
- Обрабатываются только записи со статусом `pending` или `NULL`
- После успешной обработки статус меняется на `completed`
- При ошибках статус меняется на `error`, увеличивается счетчик `processing_attempts`
- Максимальное количество попыток: 3 (настраивается в конфигурации)

**Поля, обновляемые в БД:**
- `normalized_name` - нормализованное наименование товара
- `kpved_code` - код КПВЭД
- `kpved_name` - наименование группы КПВЭД
- `processing_status` - статус обработки (`completed`, `error`, `pending`)
- `processed_at` - время успешной обработки
- `error_message` - сообщение об ошибке (если была ошибка)
- `ai_response_raw` - полный JSON ответ от ИИ
- `processing_attempts` - количество попыток обработки
- `last_processed_at` - время последней попытки обработки

---

### GET /api/nomenclature/status

Получает статус обработки номенклатуры (в разработке).

**Метод:** `GET`

**Ответ:**

**200 OK:**
```json
{
  "status": "not_implemented",
  "message": "Status endpoint is not yet implemented"
}
```

**Примечание:** Эндпоинт зарезервирован для будущей реализации мониторинга прогресса обработки.

---

## Версионирование

Текущая версия API: **v1**

Все эндпоинты находятся по пути `/api/`, что позволяет в будущем добавить версионирование (например, `/api/v1/uploads`).

При добавлении новых версий старые версии будут поддерживаться для обратной совместимости.

---

## Phase 4: API Endpoints для Многостадийного Версионирования и Классификации

### Обзор

Система многостадийного версионирования позволяет отслеживать процесс нормализации названий товаров через несколько стадий:
1. **Алгоритмические паттерны** - автоматическое исправление по правилам
2. **AI коррекция** - улучшение с помощью искусственного интеллекта
3. **Классификация** - определение категорий товаров

Каждая стадия сохраняется в истории, что позволяет откатываться к предыдущим версиям.

---

### Endpoints для Версионирования

#### POST /api/normalization/start

Начинает новую сессию нормализации для товара.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "item_id": 123,
  "original_name": "Товар для нормализации"
}
```

**Параметры:**
- `item_id` (int, обязательный) - ID товара в каталоге
- `original_name` (string, обязательный) - исходное название товара

**Ответ:**

**200 OK:**
```json
{
  "session_id": 1,
  "current_name": "Товар для нормализации",
  "original_name": "Товар для нормализации"
}
```

**400 Bad Request:**
```json
{
  "error": "item_id and original_name are required"
}
```

---

#### POST /api/normalization/apply-patterns

Применяет алгоритмические паттерны для исправления названия.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "session_id": 1,
  "stage_type": "algorithmic"
}
```

**Параметры:**
- `session_id` (int, обязательный) - ID сессии нормализации
- `stage_type` (string, опциональный) - тип стадии

**Ответ:**

**200 OK:**
```json
{
  "session_id": 1,
  "current_name": "Исправленное название",
  "stage_count": 1
}
```

**404 Not Found:**
```json
{
  "error": "Session not found"
}
```

---

#### POST /api/normalization/apply-ai

Применяет AI коррекцию с опцией использования чата.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "session_id": 1,
  "stage_type": "ai_correction",
  "use_chat": true,
  "context": ["дополнительный контекст"]
}
```

**Параметры:**
- `session_id` (int, обязательный) - ID сессии нормализации
- `use_chat` (bool, опциональный) - использовать чат режим для AI
- `context` (array of strings, опциональный) - дополнительный контекст для AI

**Ответ:**

**200 OK:**
```json
{
  "session_id": 1,
  "current_name": "Улучшенное название",
  "stage_count": 2,
  "confidence": 0.95
}
```

**400 Bad Request:**
```json
{
  "error": "ARLIAI_API_KEY not set"
}
```

---

#### GET /api/normalization/history

Получает полную историю стадий нормализации для сессии.

**Метод:** `GET`

**Query параметры:**
- `session_id` (int, обязательный) - ID сессии нормализации

**Пример запроса:**
```
GET /api/normalization/history?session_id=1
```

**Ответ:**

**200 OK:**
```json
{
  "session": {
    "id": 1,
    "catalog_item_id": 123,
    "original_name": "Товар для нормализации",
    "current_name": "Улучшенное название",
    "status": "in_progress",
    "stages_count": 2
  },
  "history": [
    {
      "id": 1,
      "stage_type": "algorithmic",
      "stage_name": "pattern_correction",
      "input_name": "Товар для нормализации",
      "output_name": "Исправленное название",
      "confidence": 0.85,
      "status": "applied",
      "created_at": "2024-01-15T10:00:00Z"
    },
    {
      "id": 2,
      "stage_type": "ai_correction",
      "stage_name": "ai_improvement",
      "input_name": "Исправленное название",
      "output_name": "Улучшенное название",
      "confidence": 0.95,
      "status": "applied",
      "created_at": "2024-01-15T10:01:00Z"
    }
  ]
}
```

---

#### POST /api/normalization/revert

Откатывает сессию к указанной стадии.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "session_id": 1,
  "target_stage": 1
}
```

**Параметры:**
- `session_id` (int, обязательный) - ID сессии нормализации
- `target_stage` (int, обязательный) - номер стадии для отката

**Ответ:**

**200 OK:**
```json
{
  "success": true,
  "session_id": 1,
  "reverted_to_stage": 1,
  "current_name": "Исправленное название"
}
```

---

### Endpoints для Классификации

#### POST /api/normalization/apply-categorization

Применяет стадию классификации к сессии нормализации.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "session_id": 1,
  "strategy_id": "top_priority",
  "context": {
    "additional_info": "value"
  }
}
```

**Параметры:**
- `session_id` (int, обязательный) - ID сессии нормализации
- `strategy_id` (string, опциональный) - ID стратегии свертки категорий (по умолчанию: "top_priority")
- `context` (object, опциональный) - дополнительный контекст

**Ответ:**

**200 OK:**
```json
{
  "session_id": 1,
  "original_category": ["Электроника", "Компьютеры", "Ноутбуки", "Игровые"],
  "folded_category": ["Электроника", "Компьютеры / Ноутбуки / Игровые"],
  "levels": {
    "level1": "Электроника",
    "level2": "Компьютеры / Ноутбуки / Игровые"
  },
  "confidence": 0.92,
  "strategy": "top_priority",
  "stage_applied": "categorization"
}
```

---

#### POST /api/classification/classify-item

Выполняет прямую классификацию элемента без создания сессии.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "item_name": "Игровой ноутбук ASUS",
  "item_code": "ASUS001",
  "strategy_id": "top_priority",
  "category": "общее",
  "context": {
    "brand": "ASUS",
    "type": "laptop"
  }
}
```

**Параметры:**
- `item_name` (string, обязательный) - название товара
- `item_code` (string, опциональный) - код товара
- `strategy_id` (string, опциональный) - ID стратегии свертки (по умолчанию: "top_priority")
- `category` (string, опциональный) - категория товара
- `context` (object, опциональный) - дополнительный контекст

**Ответ:**

**200 OK:**
```json
{
  "item_name": "Игровой ноутбук ASUS",
  "item_code": "ASUS001",
  "original_name": "Игровой ноутбук ASUS",
  "category": ["Электроника", "Компьютеры", "Ноутбуки", "Игровые"],
  "confidence": 0.95,
  "reasoning": "Товар относится к категории игровых ноутбуков",
  "strategy": "top_priority",
  "classification": {
    "category_path": ["Электроника", "Компьютеры", "Ноутбуки", "Игровые"],
    "confidence": 0.95,
    "reasoning": "Товар относится к категории игровых ноутбуков",
    "alternatives": [
      ["Электроника", "Компьютеры", "Ноутбуки"]
    ]
  }
}
```

---

#### GET /api/classification/strategies

Получает список всех доступных стратегий свертки категорий.

**Метод:** `GET`

**Ответ:**

**200 OK:**
```json
{
  "strategies": [
    {
      "id": "top_priority",
      "name": "Приоритет верхних уровней",
      "description": "Сохраняет верхние уровни, объединяя нижние",
      "max_depth": 2,
      "priority": ["0", "1"],
      "rules": [
        {
          "source_levels": [2, 3, 4, 5, 6],
          "target_level": 1,
          "separator": " / "
        }
      ]
    },
    {
      "id": "bottom_priority",
      "name": "Приоритет нижних уровней",
      "description": "Сохраняет нижние уровни, объединяя верхние",
      "max_depth": 2,
      "priority": ["-2", "-1"],
      "rules": []
    }
  ]
}
```

---

#### GET /api/classification/available

Получает доступные стратегии классификации с фильтрацией по клиенту.

**Метод:** `GET`

**Query параметры:**
- `client_id` (int, опциональный) - ID клиента для фильтрации

**Пример запроса:**
```
GET /api/classification/available?client_id=1
```

**Ответ:**

**200 OK:**
```json
{
  "strategies": [
    {
      "id": "top_priority",
      "name": "Приоритет верхних уровней",
      "description": "Сохраняет верхние уровни, объединяя нижние",
      "max_depth": 2
    }
  ],
  "total_count": 1,
  "client_filter": "1"
}
```

---

#### GET /api/classification/strategies/client

Получает стратегии для конкретного клиента из базы данных.

**Метод:** `GET`

**Query параметры:**
- `client_id` (int, обязательный) - ID клиента

**Пример запроса:**
```
GET /api/classification/strategies/client?client_id=1
```

**Ответ:**

**200 OK:**
```json
{
  "client_id": 1,
  "strategies": [
    {
      "id": 5,
      "name": "Кастомная стратегия клиента",
      "description": "Описание стратегии",
      "strategy_config": "{\"max_depth\":2,\"priority\":[\"0\",\"1\"]}",
      "client_id": 1,
      "is_default": false,
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "total_count": 1
}
```

---

#### POST /api/classification/strategies/create

Создает или обновляет стратегию свертки для клиента.

**Метод:** `POST`

**Тело запроса:**
```json
{
  "client_id": 1,
  "name": "Кастомная стратегия",
  "description": "Описание стратегии",
  "max_depth": 2,
  "priority": ["0", "1"],
  "rules": [
    {
      "source_levels": [2, 3, 4],
      "target_level": 1,
      "separator": " / ",
      "condition": ""
    }
  ]
}
```

**Параметры:**
- `client_id` (int, обязательный) - ID клиента
- `name` (string, обязательный) - название стратегии
- `description` (string, опциональный) - описание стратегии
- `max_depth` (int, опциональный) - максимальная глубина (по умолчанию: 2)
- `priority` (array of strings, опциональный) - приоритетные уровни
- `rules` (array of objects, опциональный) - правила свертки

**Ответ:**

**201 Created:**
```json
{
  "success": true,
  "strategy": {
    "id": 5,
    "name": "Кастомная стратегия",
    "description": "Описание стратегии",
    "strategy_config": "{\"id\":\"client_1_2_1234567890\",\"name\":\"Кастомная стратегия\",\"max_depth\":2,\"priority\":[\"0\",\"1\"],\"rules\":[]}",
    "client_id": 1,
    "is_default": false,
    "created_at": "2024-01-15T10:00:00Z",
    "updated_at": "2024-01-15T10:00:00Z"
  },
  "message": "Strategy created successfully",
  "strategy_id": 5
}
```

---

### Примеры использования

#### Полный цикл нормализации и классификации

```bash
# 1. Начинаем сессию нормализации
curl -X POST http://localhost:8080/api/normalization/start \
  -H "Content-Type: application/json" \
  -d '{
    "item_id": 123,
    "original_name": "Игровой ноутбук ASUS ROG"
  }'

# Ответ: {"session_id": 1, "current_name": "Игровой ноутбук ASUS ROG", ...}

# 2. Применяем алгоритмические паттерны
curl -X POST http://localhost:8080/api/normalization/apply-patterns \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": 1,
    "stage_type": "algorithmic"
  }'

# 3. Применяем AI коррекцию
curl -X POST http://localhost:8080/api/normalization/apply-ai \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": 1,
    "use_chat": true
  }'

# 4. Применяем классификацию
curl -X POST http://localhost:8080/api/normalization/apply-categorization \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": 1,
    "strategy_id": "top_priority"
  }'

# 5. Получаем историю
curl http://localhost:8080/api/normalization/history?session_id=1

# 6. При необходимости откатываемся
curl -X POST http://localhost:8080/api/normalization/revert \
  -H "Content-Type: application/json" \
  -d '{
    "session_id": 1,
    "target_stage": 1
  }'
```

---

### Коды ошибок

- **200 OK** - Успешный запрос
- **201 Created** - Ресурс успешно создан
- **400 Bad Request** - Неверный формат запроса или отсутствуют обязательные параметры
- **404 Not Found** - Сессия или ресурс не найдены
- **500 Internal Server Error** - Внутренняя ошибка сервера

---

### Примечания

1. **Сессии нормализации** сохраняются в базе данных и могут быть восстановлены после перезапуска сервера
2. **История стадий** позволяет отслеживать все изменения названия товара
3. **Стратегии классификации** могут быть настроены индивидуально для каждого клиента
4. **AI классификация** требует установленной переменной окружения `ARLIAI_API_KEY`
5. **Откат к стадии** удаляет все стадии после указанной, но сохраняет их в истории

