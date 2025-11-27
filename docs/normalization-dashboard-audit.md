# Аудит состояния компонентов дашборда нормализации

## 1. Подключение к `useProjectState`

| Группа | Файл | Статус |
| --- | --- | --- |
| MDM – основные виджеты | `frontend/components/mdm/realtime-updates.tsx`, `intelligent-deduplication.tsx`, `pipeline-visualization.tsx`, `global-search.tsx`, `process-monitoring-workspace.tsx`, `normalization-dashboard.tsx`, `export-manager.tsx`, `activity-timeline.tsx`, `business-rules-manager.tsx`, `normalization-stats-summary.tsx`, `quick-stats.tsx`, `change-history.tsx`, `data-quality-workspace.tsx`, `advanced-analytics.tsx`, `notification-center.tsx`, `performance-monitor.tsx`, `settings-panel.tsx` | ✅ Используют `useProjectState`/специализированные хуки |
| Качество данных | `frontend/components/quality/*.tsx` | ⛔ Локальный `useState/useEffect` с прямыми `fetch`-запросами. Требуется перевести на `useProjectState` (при переходе проектов состояние не сбрасывается). |
| Процессы | `frontend/components/processes/normalization-results-table.tsx`, `normalization-process-tab.tsx`, `normalization-process-card.tsx`, `normalization-preview-stats.tsx`, `normalization-history.tsx`, `normalization-performance-charts.tsx`, `session-detail-dialog.tsx`, `reclassification-process-tab.tsx` | ⛔ Собственная логика состояния, нет декларативных сбросов и отмены запросов. |
| Общие селекторы | `frontend/components/database-selector.tsx`, `normalization/data-source-selector.tsx`, `project-selector.tsx` | ⛔ Работают через локальный стейт и `fetch`, не сбрасывают состояние при смене проекта. |

**Вывод:** все основные компоненты MDM уже перешли на `useProjectState`, но блоки качества, процессов и селекторы по-прежнему используют ручную загрузку и нуждаются в рефакторинге.

## 2. Унификация утилит и типов

- `frontend/utils/normalization-helpers.ts` содержит форматирование чисел/атрибутов и функции для уверенности, но не описывает общие интерфейсы (`NormalizationStatus`, `PipelineStage`, `ActivityEvent` и т.д.).
- `frontend/utils/pipeline-helpers.ts` содержит только вычисление статуса; нет форматирования длительностей/скорости.
- Большинство компонентов объявляют локальные интерфейсы (например, `ActivityEvent`, `ExportHistory`, `PipelineStatsData`) вместо использования общих типов.

**Действия:** создать файл `frontend/types/normalization.ts` с базовыми интерфейсами и расширить утилиты (форматирование прогресса, времени, скоростей).

## 3. Консистентность UI/UX состояний

- `LoadingState`/`ErrorState` используются не везде (например, `quality-cache-tab.tsx`, `normalization-results-table.tsx` возвращают `null` или кастомные заглушки).
- Кнопка «Повторить» уже приведена к единому виду (через проп `action`) только в части компонентов.
- Нет стандартных пустых состояний для списков (кроме некоторых виджетов).

**Действия:** заменить кастомные блоки на `LoadingState`/`ErrorState`, добавить пустые заглушки по аналогии с `ActivityTimeline`.

## 4. URL-параметры и контекст проекта

- `useProjectSearchParams` задействован только в `NormalizationResultsTable`.
- Фильтры/страницы других таблиц (`quality-*`, `history`, `databases`) не синхронизированы с URL и не сбрасываются при смене проекта.

**Действия:** внедрить `useProjectSearchParams` в остальные таблицы и сбрасывать параметры при смене `clientId/projectId`.

## 5. Автообновление и производительность

- Компоненты мониторинга используют автообновление каждые 5–10 секунд, что соответствует требованиям.
- Тяжёлые панели (`advanced-analytics`, `intelligent-deduplication`) уже отключают автообновление.
- `quality-cache-tab.tsx` реализует свой цикл обновления без отмены запросов и не учитывает смену проектов.

**Действия:** перевести тяжёлые табы (качество, процессы) на `useProjectState` с продуманным `refetchInterval` и отменой запросов.

## 6. Совместимость с API

- Все новые `fetch`-вызовы используют `signal`, но модуль качества и процессы продолжают отправлять запросы без отмены и без fallback’а на 404.
- Некоторые старые компоненты (например, `normalization-results-table.tsx`) используют смешанный набор эндпоинтов и требуют унификации (одни и те же данные доступны через `/api/clients/...` и общие `/api/...`).

**Действия:** привести все запросы к `/api/clients/${clientId}/projects/${projectId}/…` с корректной обработкой 404 и fallback для старых серверов.

## 7. Рефакторинг структуры

- `NormalizationResultsTable` и `NormalizationProcessTab` всё ещё содержат крупные блоки логики, сложные `useEffect` и несколько responsibilities.
- Вкладки качества повторяют схожие блоки (карточки показателей, таблицы) без переиспользуемых компонентов.

**Действия:** вынести бизнес-логику в отдельные хуки (`useNormalizationResults`, `useProcessStatus`) и выделить переиспользуемые подкомпоненты для карточек, таблиц и фильтров.

## 8. Тестовый чеклист

Необходимо подготовить сценарии ручного тестирования:

1. Смена клиента/проекта – проверка сброса всех вкладок и URL-параметров.
2. Запущенный процесс – мониторинг, пайплайн, уведомления и автообновление.
3. Пустые данные (новый проект) – отображение заглушек.
4. Ошибки API (404, 500) – корректные `ErrorState` и возможность повторить.
5. Массовые действия (экспорт, дедупликация) – обновление состояний и историй.

После документирования сценариев пройтись по ним вручную и зафиксировать проблемы.

