# Tools

Эта папка содержит вспомогательные утилиты для работы с базой данных.

## Использование

Каждый файл — это отдельная исполняемая программа с уникальным build-тегом. Это позволяет избежать конфликтов при сборке проекта.

### Запуск через go run (с указанием build-тега):

```bash
# Анализ структуры базы данных
go run -tags tool_analyze_database_structure ./tools/analyze_database_structure.go

# Анализ кэшированных метаданных
go run -tags tool_analyze_cached_metadata ./tools/analyze_cached_metadata.go

# Анализ баз контрагентов
go run -tags tool_analyze_counterparty_databases ./tools/analyze_counterparty_databases.go

# Применение миграций
go run -tags tool_apply_migration ./tools/apply_migration.go

# Проверка содержимого БД
go run -tags tool_check_db_content ./tools/check_db_content.go

# Проверка метаданных БД
go run -tags tool_check_database_metadata ./tools/check_database_metadata.go

# Проверка готовности нормализации
go run -tags tool_check_normalization_readiness ./tools/check_normalization_readiness.go

# Проверка баз данных проекта
go run -tags tool_check_project_databases ./tools/check_project_databases.go

# Проверка проектов
go run -tags tool_check_projects ./tools/check_projects.go

# Проверка service.db
go run -tags tool_check_service_db ./tools/check_service_db.go

# Детальный анализ структуры
go run -tags tool_detailed_structure_analysis ./tools/detailed_structure_analysis.go

# Список всех баз данных
go run -tags tool_list_all_databases ./tools/list_all_databases.go

# Список всех таблиц
go run -tags tool_list_all_tables ./tools/list_all_tables.go

# Ручное определение структуры
go run -tags tool_manual_detect ./tools/manual_detect.go

# Регистрация баз данных AITAS
go run -tags tool_register_aitas_databases ./tools/register_aitas_databases.go

# Прямой запуск нормализации
go run -tags tool_run_normalization_direct ./tools/run_normalization_direct.go

# Запуск нормализации проекта
go run -tags tool_start_normalization_project ./tools/start_normalization_project.go

# Мониторинг нормализации
go run -tags tool_monitor_normalization ./tools/monitor_normalization.go

# Проверка статуса API
go run -tags tool_check_api_status ./tools/check_api_status.go
```

### Сборка исполняемых файлов:

```bash
# Создать директорию для бинарников
mkdir -p bin

# Сборка конкретной утилиты
go build -tags tool_analyze_cached_metadata -o bin/analyze_cached_metadata.exe ./tools/analyze_cached_metadata.go

# Или использовать Makefile (см. ниже)
make build-tool TOOL=analyze_cached_metadata
```

## Примечание

Эти утилиты используют build-теги для предотвращения конфликтов при сборке. Каждая утилита имеет уникальный тег вида `tool_<имя_файла>`, что позволяет компилировать их независимо друг от друга.

**Важно:** При использовании `go build ./...` или `go run ./...` без указания тегов, эти файлы не будут включены в сборку, что предотвращает ошибки дублирования функции `main()`.

