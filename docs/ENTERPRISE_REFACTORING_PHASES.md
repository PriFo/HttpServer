# Enterprise рефакторинг — оставшиеся фазы

Актуализировано: 24.11.2025

## Текущее состояние
- ✅ `internal/` развернут (config, domain, infrastructure, api/routing)
- ✅ Конфигурация и модели вынесены (`internal/config`, `internal/domain/models`)
- ✅ Кэши и AI-провайдеры перенесены в `internal/infrastructure/{cache,ai}`
- ✅ DI-контейнер подключён через `server/container.go`, роуты лежат в `internal/api/routes`
- ✅ `server.go` сокращён до 720 строк (целевое значение < 1000) - **ДОСТИГНУТО!**
- ⚠️ 23 файлов формата `server_*.go` всё ещё находятся в корне `server/` и содержат legacy-обработчики
- ✅ Менеджеры и адаптеры перенесены в `internal/infrastructure` (см. фазу 6)
- ✅ `main.go` расположен в `cmd/server/main.go` и инициализирует контейнер

## Фаза 6: Выделение менеджеров и адаптеров (Статус: ✅)

### 6.1 Менеджеры → `internal/infrastructure`
Созданы директории `internal/infrastructure/workers`, `internal/infrastructure/monitoring` и расширен `internal/infrastructure/cache/`. Файлы перенесены и переименованы:
- `server/worker_config.go` → `internal/infrastructure/workers/config_manager.go`
- `server/provider_orchestrator.go` → `internal/infrastructure/ai/orchestrator.go`
- `server/system_scanner_history.go` → `internal/infrastructure/cache/scan_history.go`
- `server/system_scanner_incremental.go` → `internal/infrastructure/cache/db_tracker.go`
- `server/monitoring.go` → `internal/infrastructure/monitoring/manager.go`

DI-контейнер уже ссылается на новые пакеты; все импорты обновлены, кэш/мониторинг доступны вне `server`. Следующий контрольный шаг — удалить остаточные ссылочные типы из `server` после завершения фазы 7.

### 6.2 Адаптеры → `internal/infrastructure`
- `server/server_normalizer_adapter.go` → `internal/infrastructure/normalization/adapter.go`
- `server/server_worker_adapter.go` → `internal/infrastructure/workers/adapter.go`

Адаптеры сейчас используются контейнером и слоями `internal/api`. Дополнительно стоит покрыть их unit-тестами после миграции хендлеров (см. фазу 7).

## Фаза 7: Рефакторинг server_*.go файлов

### 7.1 Актуальная классификация (23 файла)

**Обработчики → `server/handlers/legacy/`**
- `server/server_benchmarks.go` → `server/handlers/legacy/benchmarks_legacy.go`
- `server/server_duplicate_detection_api.go` → `server/handlers/legacy/duplicate_detection_legacy.go`
- `server/server_export.go` → `server/handlers/legacy/export_legacy.go`
- `server/server_gisp_nomenclatures.go` → `server/handlers/legacy/gisp_nomenclatures_legacy.go`
- `server/server_groups.go` → `server/handlers/legacy/groups_legacy.go`
- `server/server_kpved_reclassify.go` → `server/handlers/legacy/kpved_reclassify_legacy.go`
- `server/server_models_benchmark.go` → `server/handlers/legacy/models_benchmark_legacy.go`
- `server/server_okpd2.go` → `server/handlers/legacy/okpd2_legacy.go`
- `server/server_patterns.go` → `server/handlers/legacy/patterns_legacy.go`
- `server/server_pipeline.go` → `server/handlers/legacy/pipeline_legacy.go`
- `server/server_reclassification.go` → `server/handlers/legacy/reclassification_legacy.go`
- `server/server_versions.go` → `server/handlers/legacy/versions_legacy.go`

**Similarity API → `server/handlers/legacy/` (можно группировать поддиректорией `similarity/`)**
- `server/server_similarity_api.go` → `server/handlers/legacy/similarity_api_legacy.go`
- `server/server_similarity_analysis_api.go` → `server/handlers/legacy/similarity_analysis_legacy.go`
- `server/server_similarity_export_api.go` → `server/handlers/legacy/similarity_export_legacy.go`
- `server/server_similarity_learning_api.go` → `server/handlers/legacy/similarity_learning_legacy.go`
- `server/server_similarity_performance.go` → `server/handlers/legacy/similarity_performance_legacy.go`

**Benchmarks/модели → `server/handlers/legacy/` (при необходимости с поддиректориями)**
- `server/server_benchmarks.go` (см. выше)
- `server/server_models_benchmark.go` (см. выше)

**Сервисы/логика (в `server/services/legacy/` или `internal/domain/services/`)**
- `server/server_classification_management.go` → выделить сервисные методы в `server/services/classification_management_legacy.go`, а HTTP-обработчики — в `server/handlers/legacy/classification_management_legacy.go`
- `server/server_classifiers.go` → `server/services/classifiers_legacy.go` + `server/handlers/legacy/classifiers_legacy.go`
- `server/server_handler.go` → заменить вызовами `internal/api/routes`, оставив только glue-код для DI
- `server/server_worker_wrappers.go` → вынести обёртки в `internal/infrastructure/monitoring`/`workers` и оставить в `server` только вызов контейнера

**Тесты (оставить рядом с новыми пакетами)**
- `server/server_backup_restore_test.go` — переместить к соответствующим handler'ам или оставить в `server/` до стабилизации
- `server/server_handlers_validation_test.go` — перенести в `server/handlers/` после релокации основного кода

### 7.2 План перемещения
1. Переносить группами (Benchmarks → Similarity → Export/Groups → Classification/Models → Complex), после каждой группы запускать `go build ./server/...`.
2. Для каждого файла:
   - изменить `package` на целевой (`package legacy` в `server/handlers/legacy`)
   - обновить импорты на `internal/domain`/`server/services`
   - зарегистрировать маршруты через `internal/api/routes` вместо прямых вызовов из `server.go`
3. В `server/container.go` создать фабрики для legacy-хендлеров и отдавать их в router (до тех пор, пока полный переход на Clean Architecture не завершён).
4. После миграции удалить неиспользуемые вспомогательные функции из `server.go` и пересобрать проект.

### 7.3 Очередность групп (обновлено)
| Группа | Риск | Файлы | Действия и критерии готовности |
| --- | --- | --- | --- |
| Benchmarks & Versions | Низкий | `server_benchmarks.go`, `server_versions.go`, `server_okpd2.go`, `server_patterns.go` | ✅ Все 4 файла мигрированы: `benchmarks.go`, `versions.go`, `okpd2.go`, `patterns.go`; ✅ 11 обработчиков зарегистрированы в `legacy_routes_adapter.go` под группами `/api/legacy/benchmarks`, `/api/legacy/versions`, `/api/legacy/okpd2`, `/api/legacy/patterns`; ✅ `server.go` очищен от оригинальных файлов. |
| Similarity | Средний | `server_similarity_api.go`, `server_similarity_analysis_api.go`, `server_similarity_export_api.go`, `server_similarity_learning_api.go`, `server_similarity_performance.go` | ✅ Файлы переименованы: `similarity_api.go`, `similarity_learning_api.go`, `similarity_export_api.go`, `similarity_analysis_api.go`, `similarity_performance.go`; ✅ все 17 обработчиков зарегистрированы в `legacy_routes_adapter.go`; ✅ `server.go` очищен от оригинальных файлов. |
| Export & Groups | Средний | `server_export.go`, `server_groups.go` | ✅ `server_export.go` переименован в `export.go`; ✅ 4 обработчика зарегистрированы в `legacy_routes_adapter.go`; ✅ `server_groups.go` содержит только типы, оставлен как есть. |
| Classification & Models | Высокий | `server_classification_management.go`, `server_classifiers.go`, `server_models_benchmark.go` | ✅ Файлы переименованы: `classifiers.go`, `classification_management.go`, `models_benchmark.go`; ✅ 5 обработчиков зарегистрированы в `legacy_routes_adapter.go`: `/api/legacy/classification/classifiers`, `/api/legacy/classification/classifiers/by-project-type`, `/api/legacy/classification/reset`, `/api/legacy/classification/reset-all`, `/api/legacy/models/benchmark`. |
| Complex Handlers | Критический | `server_gisp_nomenclatures.go`, `server_kpved_reclassify.go`, `server_pipeline.go`, `server_reclassification.go`, `server_duplicate_detection_api.go` | ✅ Все 5 файлов мигрированы: `gisp_nomenclatures.go`, `pipeline.go`, `reclassification.go`, `duplicate_detection_api.go`, `kpved_reclassify.go`; ✅ 17 обработчиков зарегистрированы в `legacy_routes_adapter.go` под группами `/api/legacy/gisp`, `/api/legacy/pipeline`, `/api/legacy/reclassification`, `/api/legacy/duplicate-detection`, `/api/legacy/kpved-reclassify`. |

**Контрольные точки**
- После каждой группы: `go build ./...` + `npm run todos:scan`.
- Журналировать прогресс в `docs/MIGRATION.md` (см. раздел документации).

### 7.5 Единая регистрация legacy-маршрутов
- Добавлен `internal/api/routes/legacy_routes.go` + адаптер `server/legacy_routes_adapter.go`.
- Сейчас покрыты группы `benchmarks` и `similarity` (эндпоинты доступны по `/api/legacy/...` помимо старых `mux`-маршрутов). Группа `similarity` включает 17 обработчиков: compare, batch, weights, evaluate, stats, cache/clear, learn, optimal-threshold, cross-validate, analyze, find-similar, compare-weights, breakdown, export, import, performance, performance/reset.
- При переносе следующей группы добавляйте методы `register<Domain>` в адаптер, чтобы исключить дублирование в `server.go`.

### 7.4 Скрипт миграции
- `scripts/migrate_server_file.sh --dry-run server/server_patterns.go server/handlers/legacy`
- Скрипт автоматически:
  - копирует файл в целевую директорию и переименовывает в `*_legacy.go`;
  - меняет `package` на `legacy`;
  - правит относительные импорты (`../internal` → `../../internal`);
  - добавляет `// TODO:legacy-migration` для ручной доводки зависимостей.
- После копирования обязательно:
  - зарегистрировать маршруты в `internal/api/routes/legacy_routes.go`;
  - обновить smoke-тесты и TODO Dashboard.

## Фаза 8: Финальная структуризация

### 8.1 Перемещение main.go (Статус: ✅)
- `cmd/server/main.go` уже использует контейнер (`internal/container`). Проверить, что скрипты сборки и документация ссылаются на новый путь (Makefile/README/CI).

### 8.2 Очистка `server.go` (в процессе)
- Сократить файл с 6809 строк (см. `docs/server_reduction.md`) до < 1000:
  - оставить только `Server` struct, `NewServerWithConfig`, `Start`, `Shutdown`, `setupRouter` и минимальные вспомогательные методы (логгер, middleware биндинг)
  - удалить все `handle*` функции после миграции в `server/handlers`
  - перенести авторизацию/валидацию в `internal/api/middleware` и `server/handlers`
  - убедиться, что `server/server_handler.go` больше не содержит логики маршрутизации — источником правды становится `internal/api/routes`

### 8.3 Проверки и регрессии
- `go mod tidy`
- `go build ./...`
- `go test ./...`
- smoke-тесты основных endpoints: upload, normalization, similarity, monitoring
- убедиться в отсутствии циклических зависимостей (особенно между `server`, `internal/api`, `internal/container`, `internal/infrastructure`)
- статический контроль сложности: `gocyclo -over 15 server/ internal/`
- обновить `.todos` / TODO Dashboard (`npm run todos:all`) перед финальным PR
- GitHub Actions: `.github/workflows/refactoring-checks.yml` (лимиты на размер `server.go` и количество `server_*.go`; значения настраиваются через `repository variables`)

## Критерии успеха (обновлено)
- `server.go` < 1000 строк и не содержит legacy-хендлеров
- все бывшие `server_*.go` расположены в `server/handlers/legacy` или `server/services`
- менеджеры/адаптеры доступны из `internal/infrastructure`
- `cmd/server/main.go` – единственная точка входа
- `go build ./...` и `go test ./...` проходят без ошибок

