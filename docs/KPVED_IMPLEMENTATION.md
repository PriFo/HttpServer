# Реализация автоматической загрузки и хранения КПВЭД

## Обзор

Реализована система автоматической загрузки и хранения классификатора КПВЭД в базе данных. Данные загружаются автоматически при старте сервера, если их нет в базе.

## Компоненты

### 1. API Endpoints

Зарегистрированы следующие endpoints для работы с КПВЭД:

- **GET `/api/kpved/stats`** - Получение статистики по классификатору КПВЭД
  - Возвращает общее количество кодов, максимальный уровень и распределение по уровням

- **POST `/api/kpved/load`** - Загрузка классификатора из файла по пути
  - Тело запроса: `{"file_path": "путь/к/файлу/КПВЭД.txt"}`
  - Загружает данные из указанного файла в базу данных

- **POST `/api/kpved/load-from-file`** - Загрузка классификатора через multipart/form-data
  - Параметр формы: `file` (файл КПВЭД.txt)
  - Загружает данные из загруженного файла в базу данных

### 2. Автоматическая загрузка

Функция `ensureKpvedLoaded()` в `server/kpved_auto_load.go`:

- Проверяет наличие данных в таблице `kpved_classifier` при старте сервера
- Если данных нет, автоматически ищет файл `КПВЭД.txt` в стандартных местах:
  - `КПВЭД.txt` (корень проекта)
  - `kpved.txt`
  - `KPVED.txt`
  - `./КПВЭД.txt`
  - `../КПВЭД.txt`
- Автоматически загружает данные, если файл найден
- Логирует процесс загрузки

### 3. Структура данных

Таблица `kpved_classifier` в базе данных `service.db`:

```sql
CREATE TABLE kpved_classifier (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    code TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    parent_code TEXT,
    level INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)
```

Индексы:
- `idx_kpved_code` - по коду
- `idx_kpved_parent` - по родителю
- `idx_kpved_level` - по уровню

### 4. Статистика данных

После загрузки в базе данных:
- **Всего записей**: 5,426
- **Уровень 0** (секции): 21 запись
- **Уровень 1** (разделы): 88 записей
- **Уровень 2** (группы): 835 записей
- **Уровень 3** (детальные коды): 4,482 записи
- **Максимальный уровень**: 3

## Использование

### Автоматическая загрузка

При запуске сервера данные КПВЭД автоматически проверяются и загружаются, если их нет в базе. Никаких дополнительных действий не требуется.

### Ручная загрузка через API

Если нужно перезагрузить данные:

```bash
# Через путь к файлу
curl -X POST http://localhost:9999/api/kpved/load \
  -H "Content-Type: application/json" \
  -d '{"file_path": "КПВЭД.txt"}'

# Через загрузку файла
curl -X POST http://localhost:9999/api/kpved/load-from-file \
  -F "file=@КПВЭД.txt"
```

### Получение статистики

```bash
curl http://localhost:9999/api/kpved/stats
```

Ответ:
```json
{
  "total": 5426,
  "levels": 4,
  "levels_distribution": [
    {"level": 0, "count": 21},
    {"level": 1, "count": 88},
    {"level": 2, "count": 835},
    {"level": 3, "count": 4482}
  ]
}
```

## Утилиты

### Проверка данных в базе

```bash
go run tools/check_kpved.go service.db
```

Показывает:
- Общее количество записей
- Распределение по уровням
- Примеры записей
- Количество секций

### Детальная проверка

```bash
go run tools/check_kpved_detailed.go service.db
```

Показывает:
- Структуру таблицы
- Индексы
- Примеры записей по уровням
- Проверку целостности данных

### Загрузка данных

```bash
go run tools/load_kpved.go -file КПВЭД.txt -db service.db
```

### Тестирование endpoints

```powershell
# PowerShell
.\tools\test_kpved_endpoints.ps1

# Bash
bash tools/test_kpved_endpoints.sh
```

## Файлы

- `server/kpved_handlers.go` - HTTP handlers для работы с КПВЭД
- `server/kpved_auto_load.go` - Автоматическая загрузка при старте
- `server/legacy_routes_adapter.go` - Регистрация endpoints
- `server/server_start_shutdown.go` - Вызов автоматической загрузки
- `database/kpved_loader.go` - Парсер и загрузчик данных
- `tools/check_kpved.go` - Утилита проверки данных
- `tools/check_kpved_detailed.go` - Детальная проверка
- `tools/load_kpved.go` - Утилита загрузки
- `tools/test_kpved_endpoints.ps1` - Тестирование endpoints (PowerShell)
- `tools/test_kpved_endpoints.sh` - Тестирование endpoints (Bash)

## Проверка целостности

Данные проверены на:
- ✅ Наличие всех записей
- ✅ Корректность иерархии (все записи уровней > 0 имеют родителя)
- ✅ Отсутствие дубликатов кодов
- ✅ Корректность уровней
- ✅ Наличие индексов

## Примечания

- Данные хранятся в `service.db` (service database)
- Автоматическая загрузка происходит только при отсутствии данных
- Если файл не найден, загрузка пропускается с предупреждением в логах
- Все операции логируются с префиксом `[KPVED]`

