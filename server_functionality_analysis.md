# Анализ функционала в server.go

## Общая информация
- **Размер файла**: 16033 строки
- **Основная структура**: HTTP сервер для приема и обработки данных из 1С
- **Архитектура**: Монолитный файл с большим количеством обработчиков и вспомогательных функций

---

## 1. ИНИЦИАЛИЗАЦИЯ И КОНФИГУРАЦИЯ

### Структура Server
Содержит более 30 полей, включая:
- **Базы данных**: `db`, `normalizedDB`, `serviceDB`
- **Процессоры**: `nomenclatureProcessor`, `normalizer`, `qualityAnalyzer`
- **AI клиенты**: `arliaiClient`, `openrouterClient`, `huggingfaceClient`, `multiProviderClient`
- **Сервисы**: `normalizationService`, `counterpartyService`, `uploadService`, `clientService`, `qualityService`, `classificationService`, `similarityService`
- **Handlers**: `uploadHandler`, `clientHandler`, `normalizationHandler`, `qualityHandler`, `classificationHandler`, `counterpartyHandler`, `similarityHandler`
- **Кэши и менеджеры**: `similarityCache`, `arliaiCache`, `dbInfoCache`, `workerConfigManager`, `monitoringManager`, `providerOrchestrator`
- **Классификаторы**: `hierarchicalClassifier` (KPVED)

### Функции инициализации
- `NewServer()` - устаревший конструктор
- `NewServerWithConfig()` - основной конструктор с полной конфигурацией
- `Start()` - запуск сервера и инициализация всех компонентов
- `Shutdown()` - graceful shutdown

---

## 2. ОБРАБОТКА ЗАГРУЗОК (UPLOADS)

### Старые эндпоинты (для обратной совместимости)
- `/handshake` - рукопожатие с 1С
- `/metadata` - получение метаданных
- `/constant` - получение констант
- `/catalog/meta` - метаданные каталога
- `/catalog/item` - один элемент каталога
- `/catalog/items` - пакет элементов каталога
- `/complete` - завершение загрузки
- `/stats` - статистика
- `/health` - проверка здоровья

### API v1 эндпоинты
- `/api/v1/upload/handshake`
- `/api/v1/upload/metadata`
- `/api/v1/upload/nomenclature/batch`

### Основные API эндпоинты
- `/api/uploads` - список загрузок
- `/api/uploads/{id}` - операции с конкретной загрузкой:
  - GET - получение информации
  - GET `/data` - получение данных
  - GET `/data/stream` - потоковая передача данных
  - POST `/verify` - верификация загрузки

### Нормализованные загрузки
- `/api/normalized/uploads` - список нормализованных загрузок
- `/api/normalized/uploads/{id}` - операции с нормализованной загрузкой
- `/api/normalized/upload/handshake`
- `/api/normalized/upload/metadata`
- `/api/normalized/upload/constant`
- `/api/normalized/upload/catalog/meta`
- `/api/normalized/upload/catalog/item`
- `/api/normalized/upload/complete`

### Обработчики
- `handleHandshake()` - обработка рукопожатия
- `handleMetadata()` - обработка метаданных
- `handleConstant()` - обработка констант
- `handleCatalogMeta()` - метаданные каталога
- `handleCatalogItem()` - один элемент
- `handleCatalogItems()` - пакет элементов
- `handleNomenclatureBatch()` - пакетная загрузка номенклатуры
- `handleComplete()` - завершение загрузки
- `handleListUploads()` - список загрузок
- `handleUploadRoutes()` - маршрутизация операций с загрузками
- `handleGetUpload()` - получение информации о загрузке
- `handleGetUploadData()` - получение данных загрузки
- `handleStreamUploadData()` - потоковая передача
- `handleVerifyUpload()` - верификация
- Аналогичные функции для нормализованных загрузок

---

## 3. ОБРАБОТКА НОМЕНКЛАТУРЫ

### Эндпоинты
- `/api/nomenclature/process` - запуск обработки номенклатуры
- `/api/nomenclature/status` - статус обработки
- `/api/nomenclature/recent` - недавние записи
- `/api/nomenclature/pending` - ожидающие обработки
- `/nomenclature/status` - HTML страница статуса

### Обработчики
- `startNomenclatureProcessing()` - запуск обработки
- `getNomenclatureStatus()` - получение статуса
- `getNomenclatureRecentRecords()` - недавние записи
- `getNomenclaturePendingRecords()` - ожидающие записи
- `serveNomenclatureStatusPage()` - HTML страница
- `getNomenclatureFromMainDB()` - получение из основной БД
- `getNomenclatureFromNormalizedDB()` - получение из нормализованной БД

---

## 4. НОРМАЛИЗАЦИЯ ДАННЫХ

### Основные эндпоинты
- `/api/normalize/start` - запуск нормализации (TODO: перенести в handler)
- `/api/normalize/events` - события нормализации (SSE)
- `/api/normalization/status` - статус нормализации
- `/api/normalization/stop` - остановка нормализации
- `/api/normalization/stats` - статистика
- `/api/normalization/groups` - группы нормализации
- `/api/normalization/group-items` - элементы группы
- `/api/normalization/item-attributes/{id}` - атрибуты элемента
- `/api/normalization/export-group` - экспорт группы

### Pipeline и версионирование
- `/api/normalization/pipeline/stats` - статистика pipeline
- `/api/normalization/pipeline/stage-details` - детали этапа
- `/api/normalization/export` - экспорт
- `/api/normalization/start` - запуск нормализации с версионированием
- `/api/normalization/apply-patterns` - применение паттернов
- `/api/normalization/apply-ai` - применение AI
- `/api/normalization/history` - история сессий
- `/api/normalization/revert` - откат этапа
- `/api/normalization/apply-categorization` - применение категоризации

### Конфигурация и метаданные
- `/api/normalization/config` - конфигурация нормализации
- `/api/normalization/databases` - список БД для нормализации
- `/api/normalization/tables` - таблицы
- `/api/normalization/columns` - колонки

### Бенчмарки
- `/api/normalization/benchmark/upload` - загрузка бенчмарка
- `/api/normalization/benchmark/list` - список бенчмарков
- `/api/normalization/benchmark/{id}` - получение бенчмарка

### Обработчики
- `handleNormalizeStart()` - запуск нормализации
- `handleNormalizationEvents()` - события (SSE)
- `handleNormalizationStatus()` - статус
- `handleNormalizationStop()` - остановка
- `handleNormalizationStats()` - статистика
- `handleNormalizationGroups()` - группы
- `handleNormalizationGroupItems()` - элементы группы
- `handleNormalizationItemAttributes()` - атрибуты
- `handleNormalizationExportGroup()` - экспорт группы
- `handlePipelineStats()` - статистика pipeline
- `handleStageDetails()` - детали этапа
- `handleExport()` - экспорт
- `handleNormalizationConfig()` - конфигурация
- `handleNormalizationDatabases()` - БД
- `handleNormalizationTables()` - таблицы
- `handleNormalizationColumns()` - колонки
- `handleStartNormalization()` - запуск с версионированием
- `handleApplyPatterns()` - применение паттернов
- `handleApplyAI()` - применение AI
- `handleGetSessionHistory()` - история
- `handleRevertStage()` - откат
- `handleApplyCategorization()` - категоризация
- `handleNormalizationBenchmarkUpload()` - загрузка бенчмарка
- `handleNormalizationBenchmarkList()` - список бенчмарков
- `handleNormalizationBenchmarkGet()` - получение бенчмарка

### Вспомогательные функции
- `shouldStopNormalization()` - проверка остановки
- `createStopCheckFunction()` - создание функции проверки
- `isNormalizationStopped()` - проверка остановки для результата
- `handleCounterpartyNormalizationResult()` - обработка результата нормализации контрагента
- `processCounterpartyDatabasesParallel()` - параллельная обработка БД контрагентов

---

## 5. КЛАССИФИКАЦИЯ (KPVED, OKPD2)

### KPVED эндпоинты
- `/api/kpved/hierarchy` - иерархия КПВЭД
- `/api/kpved/search` - поиск в КПВЭД
- `/api/kpved/stats` - статистика КПВЭД
- `/api/kpved/load` - загрузка классификатора
- `/api/kpved/load-from-file` - загрузка из файла
- `/api/kpved/classify-test` - тестовая классификация
- `/api/kpved/classify-hierarchical` - иерархическая классификация
- `/api/kpved/reclassify` - переклассификация
- `/api/kpved/reclassify-hierarchical` - иерархическая переклассификация
- `/api/kpved/current-tasks` - текущие задачи
- `/api/kpved/reset` - сброс классификации
- `/api/kpved/mark-incorrect` - пометка как неверной
- `/api/kpved/mark-correct` - пометка как верной
- `/api/kpved/reset-all` - сброс всей классификации
- `/api/kpved/reset-by-code` - сброс по коду
- `/api/kpved/reset-low-confidence` - сброс низкой уверенности
- `/api/kpved/workers/status` - статус воркеров
- `/api/kpved/workers/stop` - остановка воркеров
- `/api/kpved/workers/resume` - возобновление воркеров
- `/api/kpved/workers/start` - запуск воркеров
- `/api/kpved/stats/classification` - статистика классификации
- `/api/kpved/stats/by-category` - статистика по категориям
- `/api/kpved/stats/incorrect` - статистика неверных

### OKPD2 эндпоинты
- `/api/okpd2/hierarchy` - иерархия ОКПД2
- `/api/okpd2/search` - поиск в ОКПД2
- `/api/okpd2/stats` - статистика ОКПД2
- `/api/okpd2/load-from-file` - загрузка из файла
- `/api/okpd2/clear` - очистка

### Общие эндпоинты классификации
- `/api/classification/classify` - классификация элемента
- `/api/classification/classify-item` - прямая классификация
- `/api/classification/strategies` - стратегии классификации
- `/api/classification/strategies/configure` - конфигурация стратегии
- `/api/classification/strategies/client` - стратегии клиента
- `/api/classification/strategies/create` - создание/обновление стратегии
- `/api/classification/available` - доступные стратегии
- `/api/classification/classifiers` - классификаторы
- `/api/classification/classifiers/by-project-type` - классификаторы по типу проекта
- `/api/classification/optimization-stats` - статистика оптимизации

### Переклассификация
- `/api/reclassification/start` - запуск переклассификации
- `/api/reclassification/events` - события переклассификации
- `/api/reclassification/status` - статус
- `/api/reclassification/stop` - остановка

### Обработчики KPVED
- `handleKpvedHierarchy()` - иерархия
- `handleKpvedSearch()` - поиск
- `handleKpvedStats()` - статистика
- `handleKpvedLoad()` - загрузка
- `handleKpvedLoadFromFile()` - загрузка из файла
- `handleKpvedClassifyTest()` - тестовая классификация
- `handleKpvedClassifyHierarchical()` - иерархическая классификация
- `handleKpvedReclassify()` - переклассификация
- `handleKpvedReclassifyHierarchical()` - иерархическая переклассификация
- `handleKpvedCurrentTasks()` - текущие задачи
- `handleResetClassification()` - сброс
- `handleMarkIncorrect()` - пометка неверной
- `handleMarkCorrect()` - пометка верной
- `handleResetAllClassification()` - сброс всей
- `handleResetByCode()` - сброс по коду
- `handleResetLowConfidence()` - сброс низкой уверенности
- `handleKpvedWorkersStatus()` - статус воркеров
- `handleKpvedWorkersStop()` - остановка
- `handleKpvedWorkersResume()` - возобновление
- `handleKpvedStatsGeneral()` - общая статистика
- `handleKpvedStatsByCategory()` - статистика по категориям
- `handleKpvedStatsIncorrect()` - статистика неверных

### Обработчики OKPD2
- `handleOkpd2Hierarchy()` - иерархия
- `handleOkpd2Search()` - поиск
- `handleOkpd2Stats()` - статистика
- `handleOkpd2LoadFromFile()` - загрузка из файла
- `handleOkpd2Clear()` - очистка

### Обработчики классификации
- `handleClassifyItem()` - классификация элемента
- `handleClassifyItemDirect()` - прямая классификация
- `handleGetStrategies()` - получение стратегий
- `handleConfigureStrategy()` - конфигурация стратегии
- `handleGetClientStrategies()` - стратегии клиента
- `handleCreateOrUpdateClientStrategy()` - создание/обновление стратегии
- `handleGetAvailableStrategies()` - доступные стратегии
- `handleGetClassifiers()` - классификаторы
- `handleGetClassifiersByProjectType()` - классификаторы по типу проекта
- `handleClassificationOptimizationStats()` - статистика оптимизации

### Обработчики переклассификации
- `handleReclassificationStart()` - запуск
- `handleReclassificationEvents()` - события
- `handleReclassificationStatus()` - статус
- `handleReclassificationStop()` - остановка

### Дополнительные функции
- `handleModelsBenchmark()` - бенчмарк моделей
- `initDefaultProjectTypeClassifiers()` - инициализация классификаторов по умолчанию

---

## 6. КАЧЕСТВО ДАННЫХ

### Эндпоинты
- `/api/quality/stats` - статистика качества
- `/api/quality/report` - отчет о качестве
- `/api/quality/item/{id}` - детали элемента качества
- `/api/quality/violations` - нарушения
- `/api/quality/violations/{id}` - детали нарушения
- `/api/quality/suggestions` - предложения по улучшению
- `/api/quality/suggestions/{id}` - действие с предложением
- `/api/quality/duplicates` - дубликаты
- `/api/quality/duplicates/{id}` - действие с дубликатом
- `/api/quality/assess` - оценка качества
- `/api/quality/analyze` - запуск анализа качества
- `/api/quality/analyze/status` - статус анализа
- `/api/quality/metrics` - метрики качества

### API v1 эндпоинты
- `/api/v1/upload/{uuid}/quality/report` - отчет по загрузке
- `/api/v1/upload/{uuid}/quality/analysis` - анализ загрузки
- `/api/v1/databases/{id}/quality/dashboard` - дашборд качества БД
- `/api/v1/databases/{id}/quality/issues` - проблемы качества БД
- `/api/v1/databases/{id}/quality/trends` - тренды качества БД

### Обработчики
- `handleQualityStats()` - статистика
- `handleGetQualityReport()` - отчет
- `handleQualityItemDetail()` - детали элемента
- `handleQualityViolations()` - нарушения
- `handleQualityViolationDetail()` - детали нарушения
- `handleQualitySuggestions()` - предложения
- `handleQualitySuggestionAction()` - действие с предложением
- `handleQualityDuplicates()` - дубликаты
- `handleQualityDuplicateAction()` - действие с дубликатом
- `handleQualityAssess()` - оценка
- `handleQualityAnalyze()` - запуск анализа
- `handleQualityAnalyzeStatus()` - статус анализа
- `handleGetQualityMetrics()` - метрики
- `handleQualityUploadRoutes()` - маршрутизация для загрузок
- `handleQualityDatabaseRoutes()` - маршрутизация для БД
- `handleQualityReport()` - отчет по загрузке
- `handleQualityAnalysis()` - анализ загрузки
- `handleQualityDashboard()` - дашборд БД
- `handleQualityIssues()` - проблемы БД
- `handleQualityTrends()` - тренды БД

---

## 7. АЛГОРИТМЫ СХОЖЕСТИ (SIMILARITY)

### Эндпоинты
- `/api/similarity/compare` - сравнение двух строк
- `/api/similarity/batch` - пакетное сравнение
- `/api/similarity/weights` - веса алгоритма
- `/api/similarity/evaluate` - оценка алгоритма
- `/api/similarity/stats` - статистика
- `/api/similarity/cache/clear` - очистка кэша
- `/api/similarity/learn` - обучение алгоритма
- `/api/similarity/optimal-threshold` - оптимальный порог
- `/api/similarity/cross-validate` - кросс-валидация
- `/api/similarity/performance` - производительность
- `/api/similarity/performance/reset` - сброс метрик производительности
- `/api/similarity/analyze` - анализ схожести
- `/api/similarity/find-similar` - поиск похожих
- `/api/similarity/compare-weights` - сравнение весов
- `/api/similarity/breakdown` - разбивка по компонентам
- `/api/similarity/export` - экспорт конфигурации
- `/api/similarity/import` - импорт конфигурации

### Обработчики
- `handleSimilarityCompare()` - сравнение
- `handleSimilarityBatch()` - пакетное сравнение
- `handleSimilarityWeights()` - веса
- `handleSimilarityEvaluate()` - оценка
- `handleSimilarityStats()` - статистика
- `handleSimilarityClearCache()` - очистка кэша
- `handleSimilarityLearn()` - обучение
- `handleSimilarityOptimalThreshold()` - оптимальный порог
- `handleSimilarityCrossValidate()` - кросс-валидация
- `handleSimilarityPerformance()` - производительность
- `handleSimilarityPerformanceReset()` - сброс метрик
- `handleSimilarityAnalyze()` - анализ
- `handleSimilarityFindSimilar()` - поиск похожих
- `handleSimilarityCompareWeights()` - сравнение весов
- `handleSimilarityBreakdown()` - разбивка
- `handleSimilarityExport()` - экспорт
- `handleSimilarityImport()` - импорт

---

## 8. ОБНАРУЖЕНИЕ ДУБЛИКАТОВ И ПАТТЕРНОВ

### Эндпоинты
- `/api/duplicates/detect` - обнаружение дубликатов
- `/api/duplicates/detect/{id}` - статус обнаружения
- `/api/patterns/detect` - обнаружение паттернов
- `/api/patterns/suggest` - предложение паттернов
- `/api/patterns/test-batch` - тестирование паттернов

### Обработчики
- `handleDuplicateDetection()` - обнаружение дубликатов
- `handleDuplicateDetectionStatus()` - статус обнаружения
- `handlePatternDetect()` - обнаружение паттернов
- `handlePatternSuggest()` - предложение паттернов
- `handlePatternTestBatch()` - тестирование паттернов

---

## 9. РАБОТА С КЛИЕНТАМИ И ПРОЕКТАМИ

### Эндпоинты клиентов
- `/api/clients` - список клиентов (GET) или создание (POST)
- `/api/clients/{id}` - операции с клиентом:
  - GET - получение информации
  - PUT - обновление
  - DELETE - удаление
- `/api/clients/{id}/statistics` - статистика клиента
- `/api/clients/{id}/nomenclature` - номенклатура клиента
- `/api/clients/{id}/databases` - базы данных клиента

### Эндпоинты проектов
- `/api/clients/{id}/projects` - список проектов (GET) или создание (POST)
- `/api/clients/{id}/projects/{project_id}` - операции с проектом:
  - GET - получение информации
  - PUT - обновление
  - DELETE - удаление
- `/api/clients/{id}/projects/{project_id}/nomenclature` - номенклатура проекта
- `/api/clients/{id}/projects/{project_id}/databases` - базы данных проекта
- `/api/clients/{id}/projects/{project_id}/benchmarks` - эталоны проекта
- `/api/clients/{id}/projects/{project_id}/benchmarks` - создание эталона (POST)

### Эндпоинты баз данных проекта
- `/api/clients/{id}/projects/{project_id}/databases` - список БД (GET) или создание (POST)
- `/api/clients/{id}/projects/{project_id}/databases/{db_id}` - операции с БД:
  - GET - получение информации
  - PUT - обновление
  - DELETE - удаление
- `/api/clients/{id}/projects/{project_id}/databases/{db_id}/tables` - таблицы БД
- `/api/clients/{id}/projects/{project_id}/databases/{db_id}/tables/{table_name}/data` - данные таблицы
- `/api/clients/{id}/projects/{project_id}/databases/{db_id}/upload` - загрузка БД

### Эндпоинты нормализации клиента
- `/api/clients/{id}/projects/{project_id}/normalization/start` - запуск нормализации
- `/api/clients/{id}/projects/{project_id}/normalization/stop` - остановка нормализации
- `/api/clients/{id}/projects/{project_id}/normalization/status` - статус нормализации
- `/api/clients/{id}/projects/{project_id}/normalization/sessions` - сессии нормализации
- `/api/clients/{id}/projects/{project_id}/normalization/sessions/{session_id}/priority` - приоритет сессии
- `/api/clients/{id}/projects/{project_id}/normalization/sessions/{session_id}/stop` - остановка сессии
- `/api/clients/{id}/projects/{project_id}/normalization/sessions/{session_id}/resume` - возобновление сессии
- `/api/clients/{id}/projects/{project_id}/normalization/stats` - статистика нормализации
- `/api/clients/{id}/projects/{project_id}/normalization/groups` - группы нормализации

### Обработчики клиентов
- `handleClients()` - список/создание
- `handleClientRoutes()` - маршрутизация операций с клиентом
- `handleGetClients()` - получение списка
- `handleCreateClient()` - создание
- `handleGetClient()` - получение информации
- `handleGetClientStatistics()` - статистика
- `handleGetClientNomenclature()` - номенклатура
- `handleGetClientDatabases()` - базы данных
- `handleUpdateClient()` - обновление
- `handleDeleteClient()` - удаление

### Обработчики проектов
- `handleGetClientProjects()` - список проектов
- `handleCreateClientProject()` - создание проекта
- `handleGetClientProject()` - получение проекта
- `handleUpdateClientProject()` - обновление проекта
- `handleDeleteClientProject()` - удаление проекта
- `handleGetProjectNomenclature()` - номенклатура проекта
- `handleGetClientBenchmarks()` - эталоны проекта
- `handleCreateClientBenchmark()` - создание эталона

### Обработчики баз данных проекта
- `handleGetProjectDatabases()` - список БД
- `handleCreateProjectDatabase()` - создание БД
- `handleGetProjectDatabase()` - получение БД
- `handleUpdateProjectDatabase()` - обновление БД
- `handleDeleteProjectDatabase()` - удаление БД
- `handleGetProjectDatabaseTables()` - таблицы БД
- `handleGetProjectDatabaseTableData()` - данные таблицы
- `handleUploadProjectDatabase()` - загрузка БД

### Обработчики нормализации клиента
- `handleStartClientNormalization()` - запуск нормализации
- `handleStopClientNormalization()` - остановка нормализации
- `handleGetClientNormalizationStatus()` - статус нормализации
- `handleGetNormalizationSessions()` - сессии нормализации
- `handleUpdateSessionPriority()` - приоритет сессии
- `handleStopNormalizationSession()` - остановка сессии
- `handleResumeNormalizationSession()` - возобновление сессии
- `handleGetClientNormalizationStats()` - статистика нормализации
- `handleGetClientNormalizationGroups()` - группы нормализации

---

## 10. РАБОТА С БАЗАМИ ДАННЫХ

### Эндпоинты
- `/api/database/info` - информация о текущей БД
- `/api/databases/list` - список всех БД
- `/api/databases/find` - поиск БД
- `/api/databases/find-project` - поиск проекта по БД
- `/api/database/switch` - переключение БД
- `/api/databases/analytics` - аналитика БД
- `/api/databases/analytics/{id}` - аналитика конкретной БД
- `/api/databases/history/{id}` - история БД
- `/api/databases/pending` - ожидающие БД
- `/api/databases/pending/{id}` - операции с ожидающей БД
- `/api/databases/pending/cleanup` - очистка ожидающих БД
- `/api/databases/scan` - сканирование БД
- `/api/databases/files` - файлы БД
- `/api/databases/bulk-delete` - массовое удаление БД
- `/api/databases/backup` - создание резервной копии
- `/api/databases/backups` - список резервных копий
- `/api/databases/backups/{id}` - скачивание резервной копии
- `/api/databases/restore` - восстановление из резервной копии

### Обработчики
- `handleDatabaseInfo()` - информация о БД
- `handleDatabasesList()` - список БД
- `handleFindDatabase()` - поиск БД
- `handleFindProjectByDatabase()` - поиск проекта
- `handleDatabaseSwitch()` - переключение БД
- `handleDatabaseAnalytics()` - аналитика
- `handleDatabaseHistory()` - история
- `handlePendingDatabases()` - ожидающие БД
- `handlePendingDatabaseRoutes()` - маршрутизация ожидающих БД
- `handleCleanupPendingDatabases()` - очистка
- `handleScanDatabases()` - сканирование
- `handleDatabasesFiles()` - файлы БД
- `handleBulkDeleteDatabases()` - массовое удаление
- `handleBackupDatabases()` - создание резервной копии
- `handleListBackups()` - список резервных копий
- `handleDownloadBackup()` - скачивание
- `handleRestoreBackup()` - восстановление
- `handleDatabaseV1Routes()` - маршрутизация API v1

---

## 11. СРЕЗЫ ДАННЫХ (SNAPSHOTS)

### Эндпоинты
- `/api/snapshots` - список срезов (GET) или создание (POST)
- `/api/snapshots/auto` - автоматическое создание среза
- `/api/snapshots/{id}` - операции со срезом:
  - GET - получение информации
  - DELETE - удаление
- `/api/snapshots/{id}/normalize` - нормализация среза
- `/api/snapshots/{id}/comparison` - сравнение срезов
- `/api/snapshots/{id}/metrics` - метрики среза
- `/api/snapshots/{id}/evolution` - эволюция среза
- `/api/projects/{project_id}/snapshots` - срезы проекта

### Обработчики
- `handleSnapshotsRoutes()` - маршрутизация срезов
- `handleSnapshotRoutes()` - маршрутизация конкретного среза
- `handleProjectSnapshotsRoutes()` - маршрутизация срезов проекта
- `handleListSnapshots()` - список срезов
- `handleCreateSnapshot()` - создание среза
- `handleCreateAutoSnapshot()` - автоматическое создание
- `handleGetSnapshot()` - получение среза
- `handleDeleteSnapshot()` - удаление среза
- `handleNormalizeSnapshot()` - нормализация среза
- `handleSnapshotComparison()` - сравнение
- `handleSnapshotMetrics()` - метрики
- `handleSnapshotEvolution()` - эволюция
- `handleGetProjectSnapshots()` - срезы проекта

---

## 12. КОНТРАГЕНТЫ

### Эндпоинты
- `/api/counterparties/normalized` - список нормализованных контрагентов
- `/api/counterparties/normalized/{id}` - операции с нормализованным контрагентом
- `/api/counterparties/all` - все контрагенты (из всех БД и нормализованных)
- `/api/counterparties/all/export` - экспорт всех контрагентов
- `/api/counterparties/bulk/update` - массовое обновление
- `/api/counterparties/bulk/delete` - массовое удаление
- `/api/counterparties/bulk/enrich` - массовое обогащение
- `/api/counterparties/duplicates` - дубликаты контрагентов
- `/api/counterparties/duplicates/{id}` - операции с дубликатом

### Обработчики
- `handleNormalizedCounterparties()` - список нормализованных
- `handleNormalizedCounterpartyRoutes()` - маршрутизация нормализованных
- `handleGetAllCounterparties()` - все контрагенты
- `handleExportAllCounterparties()` - экспорт всех
- `handleBulkUpdateCounterparties()` - массовое обновление
- `handleBulkDeleteCounterparties()` - массовое удаление
- `handleBulkEnrichCounterparties()` - массовое обогащение
- `handleCounterpartyDuplicates()` - дубликаты
- `handleCounterpartyDuplicateRoutes()` - маршрутизация дубликатов

---

## 13. МОНИТОРИНГ И ПРОИЗВОДИТЕЛЬНОСТЬ

### Эндпоинты
- `/api/monitoring/metrics` - метрики мониторинга
- `/api/monitoring/cache` - состояние кэша
- `/api/monitoring/ai` - метрики AI
- `/api/monitoring/history` - история мониторинга
- `/api/monitoring/events` - события мониторинга
- `/api/monitoring/providers/stream` - поток событий провайдеров (SSE)
- `/api/monitoring/providers` - информация о провайдерах

### Обработчики
- `handleMonitoringMetrics()` - метрики
- `handleMonitoringCache()` - кэш
- `handleMonitoringAI()` - AI метрики
- `handleMonitoringHistory()` - история
- `handleMonitoringEvents()` - события
- `handleMonitoringProvidersStream()` - поток провайдеров
- `handleMonitoringProviders()` - провайдеры

### Вспомогательные функции
- `GetCircuitBreakerState()` - состояние circuit breaker
- `GetBatchProcessorStats()` - статистика batch процессора
- `GetCheckpointStatus()` - статус checkpoint
- `CollectMetricsSnapshot()` - сбор метрик

---

## 14. УПРАВЛЕНИЕ ВОРКЕРАМИ И МОДЕЛЯМИ

### Эндпоинты
- `/api/workers/config` - конфигурация воркеров
- `/api/workers/config/update` - обновление конфигурации
- `/api/workers/providers` - доступные провайдеры
- `/api/workers/arliai/status` - статус Arliai
- `/api/workers/openrouter/status` - статус OpenRouter
- `/api/workers/huggingface/status` - статус Hugging Face
- `/api/workers/models` - доступные модели
- `/api/workers/orchestrator/strategy` - стратегия оркестратора
- `/api/workers/orchestrator/stats` - статистика оркестратора

### Обработчики
- `handleGetWorkerConfig()` - получение конфигурации
- `handleUpdateWorkerConfig()` - обновление конфигурации
- `handleGetAvailableProviders()` - доступные провайдеры
- `handleCheckArliaiConnection()` - проверка Arliai
- `handleCheckOpenRouterConnection()` - проверка OpenRouter
- `handleCheckHuggingFaceConnection()` - проверка Hugging Face
- `handleGetModels()` - получение моделей
- `handleOrchestratorStrategy()` - стратегия оркестратора
- `handleOrchestratorStats()` - статистика оркестратора

---

## 15. ДАШБОРД И ОТЧЕТЫ

### Эндпоинты
- `/api/dashboard/stats` - статистика дашборда
- `/api/dashboard/normalization-status` - статус нормализации для дашборда
- `/api/reports/generate-normalization-report` - генерация отчета по нормализации
- `/api/reports/generate-data-quality-report` - генерация отчета по качеству данных

### Обработчики
- `handleGetDashboardStats()` - статистика дашборда
- `handleGetDashboardNormalizationStatus()` - статус нормализации
- `handleGenerateNormalizationReport()` - генерация отчета по нормализации
- `handleGenerateDataQualityReport()` - генерация отчета по качеству

---

## 16. ИНТЕГРАЦИИ И ВНЕШНИЕ СЕРВИСЫ

### GISP (gisp.gov.ru)
- `/api/gisp/nomenclatures/import` - импорт номенклатур из ГИСП
- `/api/gisp/nomenclatures` - список номенклатур ГИСП
- `/api/gisp/nomenclatures/{id}` - детали номенклатуры ГИСП
- `/api/gisp/reference-books` - справочники ГИСП
- `/api/gisp/reference-books/search` - поиск в справочниках ГИСП
- `/api/gisp/statistics` - статистика ГИСП

### Обработчики GISP
- `handleImportGISPNomenclatures()` - импорт номенклатур
- `handleGetGISPNomenclatures()` - получение номенклатур
- `handleGetGISPNomenclatureDetail()` - детали номенклатуры
- `handleGetGISPReferenceBooks()` - справочники
- `handleSearchGISPReferenceBook()` - поиск в справочниках
- `handleGetGISPStatistics()` - статистика

### Эталоны
- `/api/benchmarks/import-manufacturers` - импорт производителей

### Обработчики эталонов
- `handleImportManufacturers()` - импорт производителей

### 1С интеграция
- `/api/1c/processing/xml` - генерация XML для обработки 1С

### Обработчики 1С
- `handle1CProcessingXML()` - генерация XML

---

## 17. ВСПОМОГАТЕЛЬНЫЕ ФУНКЦИИ

### Логирование
- `log()` - отправка записи в лог
- `logError()` - логирование ошибки
- `logErrorf()` - логирование ошибки с форматированием
- `logWarn()` - логирование предупреждения
- `logWarnf()` - логирование предупреждения с форматированием
- `logInfo()` - логирование информации
- `logInfof()` - логирование информации с форматированием
- `GetLogChannel()` - получение канала логов

### HTTP утилиты
- `writeXMLResponse()` - запись XML ответа
- `writeJSONResponse()` - запись JSON ответа
- `writeJSONError()` - запись JSON ошибки
- `writeErrorResponse()` - запись ошибки
- `handleHTTPError()` - обработка HTTP ошибки

### Статистика и метрики
- `getNomenclatureDBStats()` - статистика БД номенклатуры
- `startSessionTimeoutChecker()` - проверка таймаутов сессий

---

## 18. MIDDLEWARE

### Применяемые middleware (в порядке применения)
1. `SecurityHeadersMiddleware()` - заголовки безопасности
2. `middleware.RequestIDMiddleware()` - ID запроса
3. `LoggingMiddleware()` - логирование запросов
4. `middleware.CORS()` - CORS заголовки
5. `middleware.RecoverMiddleware()` - обработка паник

---

## 19. СТАТИЧЕСКИЙ КОНТЕНТ

- `/static/` - статические файлы
- `/` - корневой путь (отдает статику, если не API запрос)

---

## 20. ТИПЫ ДАННЫХ

### Основные типы
- `Server` - основная структура сервера
- `QualityAnalysisStatus` - статус анализа качества
- `DatabaseInfoResponse` - информация о БД
- `NormalizationStatus` - статус нормализации
- `NomenclatureResult` - результат номенклатуры
- `DatabaseFileInfo` - информация о файле БД

---

## ВЫВОДЫ

### Проблемы архитектуры:
1. **Монолитный файл** - 16033 строки в одном файле
2. **Дублирование кода** - много fallback обработчиков для старых и новых handlers
3. **Смешанная ответственность** - сервер содержит и бизнес-логику, и обработку HTTP
4. **TODO комментарии** - есть незавершенные задачи (например, перенос `/api/normalize/start` в handler)

### Что уже вынесено в handlers:
- Upload обработка (частично)
- Client управление (частично)
- Normalization (частично)
- Quality (частично)
- Classification (частично)
- Counterparty (частично)
- Similarity (частично)

### Что осталось в server.go:
- Большое количество обработчиков для всех функций
- Вспомогательные функции
- Инициализация и конфигурация
- Маршрутизация
- Интеграции с внешними сервисами
- Работа с базами данных
- Срезы данных
- Мониторинг
- Управление воркерами

### Рекомендации:
1. Продолжить рефакторинг - вынести оставшиеся обработчики в отдельные handlers
2. Разделить на пакеты по доменам (uploads, normalization, classification, quality, etc.)
3. Создать отдельный пакет для интеграций
4. Вынести работу с БД в отдельный слой
5. Убрать fallback обработчики после полного перехода на новые handlers

