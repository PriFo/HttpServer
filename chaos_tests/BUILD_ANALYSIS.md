# Анализ сборки Go Backend сервера

## Дата анализа
2025-01-21

## Текущее состояние

### ✅ Что работает
1. **Сборка без GUI**: `go build -tags no_gui -o httpserver_no_gui.exe main_no_gui.go` - **работает**
2. **Go версия**: `go1.25.4 windows/amd64` - установлена и работает
3. **Существующие бинарники**: 
   - `httpserver_no_gui.exe` - существует
   - `httpserver.exe` - существует
   - `httpserver_fixed.exe` - существует

### ❌ Проблемы

## 1. Makefile - неправильные пути

**Проблема**: Makefile ссылается на несуществующий файл `./main.go`

```makefile
build: swagger
	@echo "Building application..."
	go build -o ./bin/httpserver ./main.go  # ❌ Файл не существует
```

**Фактическая структура**:
- `main_no_gui.go` - точка входа без GUI (в корне)
- `cmd/server/main.go` - точка входа с GUI
- `main_docker.go` - точка входа для Docker

**Решение**:
```makefile
# Сборка без GUI (для production)
build-no-gui:
	@echo "Building application (no GUI)..."
	go build -tags no_gui -o ./bin/httpserver_no_gui.exe main_no_gui.go

# Сборка с GUI
build-gui:
	@echo "Building application (with GUI)..."
	go build -o ./bin/httpserver.exe ./cmd/server/main.go

# Сборка по умолчанию (без GUI)
build: build-no-gui
```

## 2. CGO и SQLite зависимости

**Проблема**: SQLite требует CGO, который требует GCC компилятор

**Текущая ситуация**:
- `start_server.bat` проверяет наличие GCC
- Если GCC не найден, сборка может не работать
- SQLite драйвер `github.com/mattn/go-sqlite3` требует CGO

**Проверка CGO**:
```powershell
$env:CGO_ENABLED="1"
go build -tags no_gui -o httpserver_no_gui.exe main_no_gui.go
```

**Если CGO не работает**, возможные причины:
1. GCC не установлен (MinGW-w64)
2. CGO_ENABLED=0 (отключен)
3. Проблемы с путями к GCC

**Решение**:
- Установить MinGW-w64: https://www.mingw-w64.org/
- Или использовать TDM-GCC: https://jmeubank.github.io/tdm-gcc/
- Добавить GCC в PATH

## 3. Множественные точки входа

**Проблема**: Разные скрипты используют разные точки входа

| Скрипт | Точка входа | Build tags |
|--------|-------------|------------|
| `start_server.bat` | `main_no_gui.go` или `cmd/server/main.go` | `no_gui` или без тегов |
| `.air.toml` | `main_no_gui.go` | `no_gui` |
| `Dockerfile` | `main_no_gui.go` | `no_gui` |
| `Makefile` | `./main.go` ❌ | нет |

**Рекомендация**: Стандартизировать использование `main_no_gui.go` для production сборок.

## 4. SQLite Connection Pooling

**Проблема**: Возможные блокировки БД при конкурентных запросах

**Текущие настройки** (из кода):
```go
// database/db.go
conn.SetMaxOpenConns(25)  // Максимум 25 соединений
conn.SetMaxIdleConns(5)   // 5 простаивающих соединений
conn.SetConnMaxLifetime(5 * time.Minute)
```

**Потенциальные проблемы**:
1. **"database is locked"** - при конкурентных записях
2. **Connection pool exhaustion** - при большом количестве запросов
3. **WAL mode не включен** - SQLite по умолчанию использует журнальный режим

**Рекомендации**:
1. Включить WAL mode для лучшей конкурентности:
```go
conn.Exec("PRAGMA journal_mode = WAL")
conn.Exec("PRAGMA synchronous = NORMAL")
conn.Exec("PRAGMA busy_timeout = 5000")
```

2. Уменьшить `MaxOpenConns` для SQLite (SQLite не очень хорошо работает с большим количеством соединений):
```go
conn.SetMaxOpenConns(1)  // SQLite лучше работает с одним соединением
```

3. Использовать транзакции для конкурентных операций

## 5. Зависимости Go модулей

**Проверка зависимостей**:
```bash
go mod download
go mod tidy
go mod verify
```

**Критические зависимости**:
- `github.com/mattn/go-sqlite3` - требует CGO
- `fyne.io/fyne/v2` - GUI библиотека (только для версии с GUI)
- `github.com/gin-gonic/gin` - HTTP фреймворк

## 6. Переменные окружения

**Требуемые переменные** (из `main_no_gui.go`):
- `ARLIAI_API_KEY` - API ключ для AI сервиса
- `DATABASE_PATH` - путь к основной БД (опционально)
- `NORMALIZED_DATABASE_PATH` - путь к нормализованной БД (опционально)
- `SERVICE_DATABASE_PATH` - путь к сервисной БД (опционально)
- `PORT` - порт сервера (по умолчанию 9999)

**Проверка**:
```powershell
$env:ARLIAI_API_KEY
$env:PORT
```

## 7. Проблемы запуска сервера

### 7.1 Порт 9999 занят
```powershell
netstat -ano | findstr :9999
# Если порт занят, найти процесс и завершить его
```

### 7.2 База данных заблокирована
- Проверить, не открыта ли БД в другом процессе
- Проверить наличие `.db-shm` и `.db-wal` файлов (WAL mode)
- Удалить lock файлы, если сервер упал

### 7.3 Ошибки подключения к БД
- Проверить права доступа к файлам БД
- Проверить существование директорий для БД
- Проверить свободное место на диске

## Рекомендации по исправлению

### Приоритет 1 (Критично)
1. ✅ **Исправить Makefile** - обновить пути к правильным точкам входа
2. ✅ **Проверить CGO** - убедиться, что GCC установлен и работает
3. ✅ **Стандартизировать сборку** - использовать единый подход для всех скриптов

### Приоритет 2 (Важно)
4. ✅ **Оптимизировать SQLite** - включить WAL mode, настроить connection pooling
5. ✅ **Добавить проверки** - диагностические скрипты для проверки окружения
6. ✅ **Документировать** - создать инструкции по сборке и запуску

### Приоритет 3 (Желательно)
7. ✅ **Автоматизация** - скрипты для автоматической сборки и проверки
8. ✅ **Мониторинг** - логирование проблем сборки и запуска
9. ✅ **Тестирование** - автоматические тесты сборки

## Скрипты для диагностики

### Проверка окружения
```powershell
# Проверка Go
go version

# Проверка GCC
gcc --version

# Проверка CGO
$env:CGO_ENABLED="1"
go env CGO_ENABLED

# Проверка зависимостей
go mod verify
```

### Проверка сборки
```powershell
# Сборка без GUI
$env:CGO_ENABLED="1"
go build -tags no_gui -o httpserver_no_gui.exe main_no_gui.go

# Проверка бинарника
Test-Path httpserver_no_gui.exe
```

### Проверка запуска
```powershell
# Проверка порта
netstat -ano | findstr :9999

# Запуск сервера
$env:ARLIAI_API_KEY="your-key"
.\httpserver_no_gui.exe
```

## Выводы

1. **Сборка работает** - сервер успешно компилируется
2. **Makefile устарел** - требует обновления путей
3. **CGO критичен** - необходим для SQLite, требует GCC
4. **SQLite настройки** - могут быть оптимизированы для конкурентности
5. **Документация** - нужны четкие инструкции по сборке и запуску

## Следующие шаги

1. Исправить Makefile
2. Создать универсальный скрипт сборки
3. Добавить проверки окружения в скрипты запуска
4. Оптимизировать настройки SQLite
5. Протестировать сборку и запуск на чистой системе

