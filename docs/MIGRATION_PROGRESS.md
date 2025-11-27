# Прогресс миграции Legacy Handlers

Обновлено: 2025-01-21

## Статистика миграции

### Мигрированные группы (✅ Завершено)

1. **Similarity** (5 файлов, 17 обработчиков)
   - `similarity_api.go`, `similarity_learning_api.go`, `similarity_export_api.go`, `similarity_analysis_api.go`, `similarity_performance.go`
   - Все зарегистрированы в `/api/legacy/similarity/*`

2. **Export & Groups** (1 файл, 4 обработчика)
   - `export.go`
   - Зарегистрированы в `/api/legacy/export/*`

3. **Classification & Models** (3 файла, 5 обработчиков)
   - `classifiers.go`, `classification_management.go`, `models_benchmark.go`
   - Зарегистрированы в `/api/legacy/classification/*` и `/api/legacy/models/*`

4. **Complex Handlers** (5 файлов, 17 обработчиков)
   - `gisp_nomenclatures.go`, `pipeline.go`, `reclassification.go`, `duplicate_detection_api.go`, `kpved_reclassify.go`
   - Зарегистрированы в `/api/legacy/gisp/*`, `/api/legacy/pipeline/*`, `/api/legacy/reclassification/*`, `/api/legacy/duplicate-detection/*`, `/api/legacy/kpved-reclassify/*`

5. **Benchmarks & Versions** (3 файла, 11 обработчиков)
   - `benchmarks.go`, `versions.go`, `okpd2.go`
   - Зарегистрированы в `/api/legacy/benchmarks/*`, `/api/legacy/versions/*`, `/api/legacy/okpd2/*`

6. **Patterns** (1 файл, 3 обработчика)
   - `patterns.go`
   - Зарегистрированы в `/api/legacy/patterns/*`

**Итого мигрировано:**
- 18 файлов из `server_*.go` формата
- 57 обработчиков
- Все зарегистрированы через `legacy_routes_adapter.go`

### Оставшиеся файлы `server_*.go`

- `server_groups.go` - содержит только типы, оставлен как есть
- `server_handler.go` - структура для доступа к методам Server, не требует миграции
- `server_worker_wrappers.go` - обертки для мониторинга, требует выноса в `internal/infrastructure`

### Оставшиеся обработчики в `server.go`

**✅ РЕФАКТОРИНГ ЗАВЕРШЕН!**

`server.go` сокращен с **6873 строк до 720 строк** (сокращение ~90%):

**Вынесенные файлы:**
- `server/server_new.go` (767 строк) - конструкторы `NewServer`, `NewServerWithConfig`
- `server/server_helpers.go` (192 строки) - вспомогательные методы (логирование, запись ответов)
- `server/server_init.go` (751 строка) - методы инициализации и построения функций
- `server/upload_normalized_handlers.go` - Upload и Normalized handlers (14 обработчиков)
- `server/kpved_handlers.go` - KPVED handlers (8 обработчиков)
- `server/snapshots_handlers.go` - Snapshots handlers (13 обработчиков)
- `server/counterparties_handlers.go` - Counterparties handlers (17 обработчиков)

**Оставшиеся в `server.go` (720 строк):**
- Структура `Server` и типы
- Методы жизненного цикла (`Start`, `Shutdown`, `ServeHTTP`)
- Вспомогательные методы для нормализации
- Несколько legacy handlers для обратной совместимости

**Историческая справка - изначально планировалось мигрировать:**

1. **Upload handlers** (8 обработчиков)
   - `handleGetUpload`, `handleGetUploadData`, `handleStreamUploadData`, `handleVerifyUpload`
   - `handleGetUploadNormalized`, `handleGetUploadDataNormalized`, `handleStreamUploadDataNormalized`, `handleVerifyUploadNormalized`

2. **Normalized handlers** (7 обработчиков)
   - `handleNormalizedHandshake`, `handleNormalizedMetadata`, `handleNormalizedConstant`
   - `handleNormalizedCatalogMeta`, `handleNormalizedCatalogItem`, `handleNormalizedComplete`

3. **KPVED handlers** (8 обработчиков)
   - `handleKpvedHierarchy`, `handleKpvedSearch`, `handleKpvedStats`, `handleKpvedLoad`
   - `handleKpvedClassifyTest`, `handleKpvedReclassify`, `handleKpvedClassifyHierarchical`, `handleKpvedCurrentTasks`

4. **Snapshots handlers** (12 обработчиков)
   - `handleSnapshotsRoutes`, `handleSnapshotRoutes`, `handleProjectSnapshotsRoutes`
   - `handleListSnapshots`, `handleCreateSnapshot`, `handleCreateAutoSnapshot`
   - `handleGetSnapshot`, `handleGetProjectSnapshots`, `handleDeleteSnapshot`
   - `handleNormalizeSnapshot`, `handleSnapshotComparison`, `handleSnapshotMetrics`, `handleSnapshotEvolution`

5. **Counterparties handlers** (15 обработчиков)
   - `handleNormalizedCounterparties`, `handleGetAllCounterparties`, `handleExportAllCounterparties`
   - `handleNormalizedCounterpartyRoutes`, `handleNormalizedCounterpartyStats`
   - `handleGetNormalizedCounterparty`, `handleUpdateNormalizedCounterparty`, `handleEnrichCounterparty`
   - `handleGetCounterpartyDuplicates`, `handleMergeCounterpartyDuplicates`, `handleExportCounterparties`
   - `handleBulkUpdateCounterparties`, `handleBulkDeleteCounterparties`, `handleBulkEnrichCounterparties`

6. **Прочие** (2 обработчика)
   - `handle1CProcessingXML`
   - `handleGetProjectPipelineStatsWithParams`

## ✅ Все миграции завершены!

Все обработчики успешно мигрированы:
1. ✅ **Upload handlers** → `server/upload_normalized_handlers.go` (14 обработчиков)
2. ✅ **Normalized handlers** → `server/upload_normalized_handlers.go` (вместе с Upload)
3. ✅ **KPVED handlers** → `server/kpved_handlers.go` (8 обработчиков)
4. ✅ **Snapshots handlers** → `server/snapshots_handlers.go` (13 обработчиков)
5. ✅ **Counterparties handlers** → `server/counterparties_handlers.go` (17 обработчиков)

**Дополнительно вынесено:**
- ✅ Конструкторы → `server/server_new.go` (767 строк)
- ✅ Вспомогательные методы → `server/server_helpers.go` (192 строки)
- ✅ Методы инициализации → `server/server_init.go` (755 строк)

## Регистрация маршрутов

Все мигрированные обработчики регистрируются через:
- `server/legacy_routes_adapter.go` - адаптер для Gin роутинга
- `internal/api/routes/legacy_routes.go` - интерфейс для регистрации legacy маршрутов

Маршруты доступны по префиксу `/api/legacy/*` для обратной совместимости.

