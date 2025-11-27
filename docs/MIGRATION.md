# MIGRATION.md — Legacy handlers

## 1. Цель
- собрать все `server_*.go` в `server/handlers/legacy/` и соответствующие сервисы в `server/services/legacy/`;
- сократить `server/server.go` до функций конфигурации;
- обеспечить повторяемую стратегию миграции для фаз 7–8.

## 2. Очередность групп

| Группа | Файлы | Риск | Ответственные | Критерии завершения |
| --- | --- | --- | --- | --- |
| Benchmarks / Versions | `server_benchmarks.go`, `server_versions.go`, `server_okpd2.go`, `server_patterns.go` | Низкий | backend-team | ✓ файлы перенесены в `server/handlers/legacy/`; ✓ smoke `/api/legacy/benchmarks`, `/versions`; ✓ `server.go` без `handle*Benchmark`. |
| Similarity | `server_similarity_*.go` (5 шт.) | Средний | backend-team | ✓ поддиректория `server/handlers/legacy/similarity`; ✓ единый `SimilarityHandler`; ✓ регрессионные тесты learning/export. |
| Export & Groups | `server_export.go`, `server_groups.go` | Средний | backend-team | ✓ сервисы вынесены в `server/services/legacy/export.go`; ✓ маршруты зарегистрированы через `legacy_routes.go`. |
| Classification & Models | `server_classification_management.go`, `server_classifiers.go`, `server_models_benchmark.go` | Высокий | backend-team + ML | ✓ сервисы разделены; ✓ CI для моделей проходит. |
| Complex Handlers | `server_gisp_nomenclatures.go`, `server_kpved_reclassify.go`, `server_pipeline.go`, `server_reclassification.go`, `server_duplicate_detection_api.go` | Критический | backend-team + domain experts | ✓ smoke-тесты для KPI; ✓ документация обновлена. |

## 3. Чеклист для каждого файла
1. `scripts/migrate_server_file.sh --dry-run server/server_patterns.go server/handlers/legacy`.
2. `scripts/migrate_server_file.sh server/server_patterns.go server/handlers/legacy`.
3. Обновить импорты и зависимости (см. TODO из файла).
4. Добавить регистрацию в `internal/api/routes/legacy_routes.go`.
5. Обновить `docs/MIGRATION.md` статусом (пример ниже).
6. Запустить `go build ./server/...` и smoke-тест.

## 4. Лог прогресса

| Дата | Файл | Действие | Ответственный | Комментарии |
| --- | --- | --- | --- | --- |
| 2025-11-24 | `server/server_benchmarks.go` | Подготовлено к миграции, dry-run + TODO | backend-team | Ожидает обновления зависимостей. |
| 2025-01-21 | Группа Similarity (5 файлов) | ✅ Мигрированы и переименованы | backend-team | Файлы переименованы: `similarity_api.go`, `similarity_learning_api.go`, `similarity_export_api.go`, `similarity_analysis_api.go`, `similarity_performance.go`. Все 17 обработчиков зарегистрированы в `legacy_routes_adapter.go`. |
| 2025-01-21 | Группа Export & Groups | ✅ Мигрирован export | backend-team | `server_export.go` → `server/export.go`. 4 обработчика зарегистрированы: `/api/legacy/export/data`, `/api/legacy/export/report`, `/api/legacy/export/statistics`, `/api/legacy/export/stages/progress`. `server_groups.go` содержит только типы, оставлен как есть. |
| 2025-01-21 | Группа Classification & Models | ✅ Мигрированы | backend-team | `server_classifiers.go` → `server/classifiers.go` (2 обработчика: `/api/legacy/classification/classifiers`, `/api/legacy/classification/classifiers/by-project-type`), `server_classification_management.go` → `server/classification_management.go` (2 обработчика: `/api/legacy/classification/reset`, `/api/legacy/classification/reset-all`), `server_models_benchmark.go` → `server/models_benchmark.go` (1 обработчик: `/api/legacy/models/benchmark`). Все зарегистрированы в `legacy_routes_adapter.go`. |
| 2025-01-21 | Группа Complex Handlers | ✅ Мигрированы | backend-team | `server_gisp_nomenclatures.go` → `server/gisp_nomenclatures.go` (6 обработчиков), `server_pipeline.go` → `server/pipeline.go` (3 обработчика), `server_reclassification.go` → `server/reclassification.go` (4 обработчика), `server_duplicate_detection_api.go` → `server/duplicate_detection_api.go` (2 обработчика), `server_kpved_reclassify.go` → `server/kpved_reclassify.go` (2 обработчика). Всего 17 обработчиков зарегистрированы в `legacy_routes_adapter.go` под группами `/api/legacy/gisp`, `/api/legacy/pipeline`, `/api/legacy/reclassification`, `/api/legacy/duplicate-detection`, `/api/legacy/kpved-reclassify`. |
| 2025-01-21 | Группа Benchmarks & Versions | ✅ Мигрированы | backend-team | `server_benchmarks.go` → `server/benchmarks.go` (1 обработчик: `/api/legacy/benchmarks/manufacturers/import`), `server_versions.go` → `server/versions.go` (5 обработчиков: `/api/legacy/versions/start`, `/api/legacy/versions/apply-patterns`, `/api/legacy/versions/apply-ai`, `/api/legacy/versions/session-history`, `/api/legacy/versions/revert-stage`), `server_okpd2.go` → `server/okpd2.go` (5 обработчиков: `/api/legacy/okpd2/hierarchy`, `/api/legacy/okpd2/search`, `/api/legacy/okpd2/stats`, `/api/legacy/okpd2/load`, `/api/legacy/okpd2/clear`). Всего 11 обработчиков зарегистрированы в `legacy_routes_adapter.go`. |
| 2025-01-21 | Исправление циклического импорта | ✅ Исправлено | backend-team | Устранен циклический импорт между `server` и `server/handlers` путем добавления метода `SetDatabaseHelperFunctions` в `ClientHandler`. Функции `FindMatchingProjectForDatabase` и `ParseDatabaseFileInfo` передаются как зависимости. Проект успешно компилируется. |

## 5. Дополнительные артефакты
- Smoke-тесты: `tests/regression/smoke_test.go`.
- CI: `.github/workflows/refactoring-checks.yml`.
- Документация: `docs/ENTERPRISE_REFACTORING_PHASES.md`, `README.md`, `docs/architecture.md`.
- Регистрация: `internal/api/routes/legacy_routes.go` + `server/legacy_routes_adapter.go`.

