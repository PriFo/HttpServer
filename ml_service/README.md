# Номенклатурный ML-сервис

Независимый Python-сервис, который обучает и обслуживает нейросеть (scikit-learn `MLPClassifier`) для классификации номенклатуры. Система поддерживает произвольные классы классификации (например, «Товар», «Услуга», «Тара», «Набор», «Работа» и т.д.), а также гибкую конфигурацию полей для обучения. Сервис включает контроль качества входных данных, feature store, мониторинг дрейфа, Explainable AI и управление метаданными модели.

## Основные возможности

- **REST API (FastAPI)** для обучения, инференса, контроля качества, мониторинга дрейфа и управления фичами/метаданными.
- **MLPClassifier** с архитектурой `512-256-128`, `relu`, `adam`, `batch_size=1024`, `learning_rate='adaptive'`, `warm_start=True`, `early_stopping=True`. Параметры подобраны для устойчивого обучения на десятках миллионов записей (поддерживается шардирование и переобучение на свежем датасете).
- **Feature store** на parquet-файлах с версионированием. Позволяет переиспользовать фичи, автоматически материализует производные признаки.
- **Data quality guard** – дешёвые проверки на пропуски, дубликаты, невалидные OKВЭД/HS коды перед запуском обучения/инференса.
- **Drift monitor** – расчёт PSI и Jensen–Shannon для числовых и категориальных признаков, хранение baseline и алерт при дрейфе.
- **Explainability** – гибридная система: глобальные permutation importances + локальные лексические объяснения («почему товар/услуга»).
- **Metadata store** – JSON-реестр версий модели, метрик и даты обучения; легко интегрируется с внутренними governance-инструментами.
- **Monitoring dashboard** – отдельное FastAPI-приложение (порт `6565`) на Jinja2/Plotly: показывает статус воркеров, метрики, БД `ml_store.db`, историю запросов и версий моделей; включает REST API (`GET /monitoring/workers/stats`, `GET /monitoring/db/status`, `GET /monitoring/requests/active`, `POST /monitoring/db/init`, `POST /monitoring/actions/*`), WebSocket `/ws/workers`, экспорт статистики и административные действия.

## Быстрый старт

```bash
cd ml_service
python -m venv .venv
.venv\Scripts\activate  # или source .venv/bin/activate
pip install -r requirements.txt
uvicorn ml_service.service:app --reload --port 8085
```

### Docker

Сервис можно запускать изолированно:

```bash
cd ml_service
docker compose up --build
```

```bash
docker build -t ml_service:test ./ml_service
docker run -d -p 6565:6565 -p 8085:8085 --name ml-service ml_service:test
```

`docker-compose.yml` резервирует 2 CPU и 8 ГБ RAM для контейнера, монтирует каталог `ml_artifacts` (артефакты модели) и, при необходимости, `data` для вспомогательных файлов, а также пробрасывает файл `ml_store.db` напрямую (`./ml_store.db:/app/ml_service/ml_store.db`). Внутри контейнера запускаются два процесса:

- `gunicorn ml_service.service:app` → основной ML API на `http://localhost:8085`
- `uvicorn ml_service.monitoring:app` → мониторинг/дашборд на `http://localhost:6565`

По умолчанию административные эндпоинты мониторинга защищены Basic Auth (`ML_MONITOR_ADMIN_USER`, `ML_MONITOR_ADMIN_PASSWORD`, дефолт `admin/admin`). UI автоматически обновляется каждые 10 секунд, дополнительно есть WebSocket `/ws/workers`.

В production можно масштабировать горизонтально (`docker compose up --scale ml-service=3`) или в Kubernetes, использовав `resources.requests/limits` и autoscaler.

#### SQLite-хранилище в контейнере

Схема находится в `sql/sqlite_schema.sql` и покрывает хранение датасетов, их версий, пользовательских правок и моделей. Файл `ml_store.db` уже лежит в корне репозитория и автоматически копируется/монтируется в контейнер (`./ml_store.db:/app/ml_service/ml_store.db`). При необходимости пересоздать схему:

```bash
cd ml_service
sqlite3 ml_store.db < sql/sqlite_schema.sql
# или
python -c "import sqlite3, pathlib; schema=pathlib.Path('sql/sqlite_schema.sql').read_text(); conn=sqlite3.connect('ml_store.db'); conn.executescript(schema); conn.close()"
```

- `datasets`, `dataset_versions`, `dataset_items` — полный слепок выгрузок и их версий.
- `user_updates` — правки от пользователей для последующего дообучения.
- `training_jobs` + `models` — история запусков обучения и пути к артефактам.
- `predictions_log` — журнал запросов/ответов, если потребуется трассировка.

Файл `ml_store.db` лежит рядом с кодом и переживает перезапуск контейнера; application-код обращается к нему как `sqlite:////app/ml_service/ml_store.db`.

### Пример вызовов

1. **Обучение**

```http
POST /train
{
  "version": "v1.0.0",
  "refresh_baseline": true,
  "target_field": "label",
  "feature_fields": ["name", "full_name", "kind", "unit", "okved_code", "hs_code"],
  "data": [
    {
      "name": "Поставка серверов",
      "full_name": "Поставка серверов Dell PowerEdge",
      "kind": "оборудование",
      "unit": "шт",
      "type_hint": "товар",
      "okved_code": "46.51",
      "hs_code": "8471",
      "label": "Товар"
    },
    {
      "name": "Услуги консультации",
      "full_name": "Консультационные услуги по внедрению",
      "kind": "услуга",
      "unit": "час",
      "label": "Услуга"
    },
    {
      "name": "Тара картонная",
      "full_name": "Картонная тара для упаковки",
      "kind": "упаковка",
      "unit": "шт",
      "label": "Тара"
    }
  ]
}
```

**Параметры обучения:**
- `target_field` (опционально, по умолчанию `"label"`) — поле в `NomenclatureItem`, которое используется как целевая переменная для обучения. Может быть любым строковым полем (например, `label`, `type_hint`, `kind`).
- `feature_fields` (опционально) — список полей, используемых как признаки для обучения. Если не указан, используются все доступные поля кроме `target_field`. Пример: `["name", "full_name", "kind", "unit", "okved_code", "hs_code"]`.
- `label` в данных может содержать произвольные строковые значения (например, `"Товар"`, `"Услуга"`, `"Тара"`, `"Набор"`, `"Работа"` и т.д.), а не только `"product"` и `"service"`.

Если поле `data` опущено (или пустое), сервис использует `ml_service/datasets/train_dataset.csv` как основной набор для переобучения. Перед первым запуском убедитесь, что файл существует; иначе `/train` вернёт:

```
"Нет данных для обучения, перед первым использованием необходимо загрузить в блок data датасет для обучения"
```

2. **Инференс**

```http
POST /predict
{
  "top_k": 2,
  "explain": true,
  "items": [
    {
      "name": "Услуги аутсорсинга ИТ",
      "full_name": "Пакет услуг по аутсорсингу ИТ-инфраструктуры",
      "kind": "услуга",
      "unit": "компл",
      "okved_code": "62.01"
    }
  ]
}
```

3. **Контроль качества**

```http
POST /quality
{ "items": [ ... ] }
```

4. **Дрейф**

```http
POST /drift/check
{ "items": [ ... ] }
```

## Структура

| Модуль | Описание |
| --- | --- |
| `config.py` | Pydantic-настройки сервиса и путей артефактов |
| `schemas.py` | Pydantic-схемы API |
| `model.py` | Обучение/инференс MLP, сохранение/загрузка |
| `feature_store.py` | Материализация признаков и версионирование |
| `data_quality.py` | Контроль качества входных данных |
| `drift_monitor.py` | Подготовка baseline и отчётов по дрейфу |
| `dataset_manager.py` | Управление эталонным датасетом и смешивание с клиентскими |
| `explainability.py` | Глобальные/локальные объяснения |
| `metadata.py` | Управление метаданными модели |
| `regional_validation.py` | Проверки кодировок, регионов, юрисдикций |
| `priority_scheduler.py` | Очередь с приоритетами для конкурирующих запросов |
| `monitoring/` | Веб-интерфейс и REST API мониторинга (FastAPI + Jinja2 + Plotly) |
| `service.py` | FastAPI-приложение |

## Дообучение и масштабирование

- **Warm start:** `MLPClassifier` сохраняет веса и допускает переобучение на полном датасете (через повторный вызов `/train`). Для сверхбольших данных используйте feature store + батчевую прогрузку (питание моделях через `settings.max_training_rows`).
- **Версионирование:** каждая итерация `/train` сохраняет `.joblib` под уникальной версией, плюс parquet features и baseline. Линейка версий доступна через `/metadata` и `/features`.
- **Monitoring:** `drift_monitor` сигнализирует, когда PSI>0.2 или Jensen-Shannon>0.3, что является поводом для переобучения.

## Метаданные и Explainability

- Метаданные (`metadata.json`) хранят историю метрик и позволяют прикреплять заметки (например, ссылка на тикет с обучением).
- Локальные объяснения – это комбинация:
  - конфиденса модели,
  - совпадений по лексическим словарям (товар/услуга),
  - ссылок на глобальную наиболее важную фичу (перестановочная важность).

## Данные и качество

- Сервис жестко требует наличие `name` и `full_name`.
- Перед обучением каждую выгрузку следует прогонять через `/quality`. При наличии критических ошибок ответ 422 с конкретными полями.
- Feature store и baseline-дрейф позволяют вернуться к старой версии данных и сравнить распределения.
- Папка `ml_service/datasets` хранит основной CSV `train_dataset.csv`. Любой вызов `/train` сначала валидирует и подмешивает этот файл, а затем — опциональный блок `data`. Все ошибки валидации возвращаются одним ответом 422.

### Сбор эталонного датасета

Каталог `ml_service/datasets` содержит разные CSV/Excel-выгрузки (train/valid/sample и т.д.). Нормализовать их в единый набор можно одной командой:

```
python -m ml_service.data.dataset_builder ^
  --dataset-dir ml_service/datasets ^
  --output-parquet ml_artifacts/reference_dataset.parquet ^
  --output-csv ml_service/datasets/nomenclature_master.csv
```

Скрипт:

- распознаёт кодировку и разделитель каждого файла;
- приводит данные к схеме `NomenclatureItem`, чистит пробелы и заполняет обязательные поля;
- нормализует метки (поддерживает произвольные классы: `Товар`, `Услуга`, `Тара`, `Набор`, `Работа` и т.д.) и удаляет дубликаты;
- сохраняет parquet/CSV и печатает JSON-сводку, где видно, сколько строк попало из каждого файла и какие выгрузки пришлось пропустить.

Полученный parquet-файл автоматически подхватывается `ReferenceDatasetManager` (см. `settings.reference_dataset_path`) и подходит для офлайн-обучения/валидации.

## Управление метаданными

- `GET /metadata` – текущее состояние + история.
- При обучении автоматически сохраняются метрики `accuracy`, `f1_macro`, `avg_confidence`.
- Можно добавлять дополнительные поля в `MetadataStore` (например, ответственный инженер, ссылка на датасет) – структура JSON расширяема.

## Обращение к модели и очереди запросов

1. **Лёгкие/синхронные кейсы:** внешние сервисы бьют напрямую по `POST /predict`. Благодаря `gunicorn` + нескольким воркерам сервис обрабатывает десятки запросов параллельно (по сути, по количеству воркеров). Для 2 CPU оптимально держать 3–4 воркера, 2–4 потока и включить keep-alive.
2. **Приоритетное обслуживание:** в самом FastAPI теперь есть встроенный приоритетный шедулер (`PriorityScheduler`). Каждый запрос к `/predict` получает вес в зависимости от полноты данных (наличие `full_name`, `kind`, кодов ОКВЭД/ТНВЭД, длины названия). Запросы с большим количеством информации попадают в начало очереди, а воркеры (числом `ML_PRIORITY_WORKERS`, по умолчанию 4) обрабатывают их первыми. Это гарантирует, что самые “богатые” записи классифицируются раньше, даже если одновременно пришло много запросов.
3. **Смешивание эталонных и клиентских данных:** `ReferenceDatasetManager` хранит канонический датасет в `ml_artifacts/reference_dataset.parquet`. На этапе `/train` входные данные перемешиваются с эталоном, при необходимости эталон реплицируется, чтобы занимать ≥10 % итоговой выборки. После успешного обучения свежие проверенные данные добавляются в эталон. Это защищает модель от размывания и сохраняет устойчивость к шуму.
4. **Региональная валидация и кодировки:** `RegionalValidator` проверяет ISO-коды стран, НС/ОКВЭД, а также допустимые кодировки (только UTF‑8). Ошибочные записи блокируются до обучения/инференса. При необходимости можно расширить списки стран/юрисдикций.
5. **Мониторинг очереди и воркеров:** `MonitoringStore` автоматически логирует каждый запрос (`train/predict/quality/drift`), фиксирует снапшоты воркеров (активные/очередь/ошибки + CPU/RAM), предоставляет API `GET /monitoring/workers/stats`, `GET /monitoring/requests/active`, WebSocket `/ws/workers` и UI на `http://localhost:6565`. Админ-панель даёт кнопку инициализации БД, остановку запросов, очистку логов и экспорт CSV/JSON.
6. **Пакетная нормализация / 100+ одновременных обращений:** рекомендуется ставить промежуточную очередь (RabbitMQ, Redis Streams, Kafka). Нормализационный сервис кладёт задания (id записи + полезная нагрузка) в очередь, `ml-service` поднимает пул воркеров (Celery/RQ/любой consumer), которые забирают задания, вызывают локальный FastAPI-инстанс (или напрямую `NomenclatureClassifier`) и сохраняют результаты в БД. Таким образом:
   - нагрузка распределяется, воркеры подтягивают задания в меру свободных CPU,
   - при падении приложения задания остаются в очереди и будут обработаны после перезапуска,
   - можно динамически увеличивать количество воркеров без изменения API.
7. **Downtime и надёжность:** используйте внешний персистентный брокер (RabbitMQ с durable очередями, Kafka, SQS). Это гарантирует, что при перезапуске контейнера никакой запрос не потеряется. Для гарантированной доставки включите подтверждения (`ack`) после успешной записи результата.
8. **Идемпотентность:** сохраняйте входные запросы/ответы в `service.db` (или другом персистентном хранилище) с idempotency key. При повторной доставке worker сверит, что ответ уже рассчитан.
9. **Backpressure:** лимитируйте размер очереди и/или используйте rate limiting на входном API, чтобы модель не «согнулась». В крайнем случае запросы можно временно парковать в брокере, а worker’ы будут по одной записи извлекать.

Таким образом, FastAPI остаётся удобной фронтовой обвязкой, но реальные батчи можно гонять через очередь + воркерный слой, обеспечивая устойчивость и отсутствие потерь при простойке.

## CLI для взаимодействия с моделью

Для ручных проверок и интеграции без Postman добавлен вспомогательный скрипт:

```
python -m ml_service.scripts.model_cli train --dataset ml_service/datasets/nomenclature_master.csv --version mlp_v1
python -m ml_service.scripts.model_cli predict --dataset ml_service/datasets/sample_dataset.csv --top-k 2 --explain
python -m ml_service.scripts.model_cli predict --json payload.json --base-url http://localhost:8085
```

Флаги `--limit`, `--refresh-baseline`, `--base-url` и `--timeout` позволяют управлять размером батча, обновлением drfit-baseline и адресом сервиса. Скрипт сам нормализует исходные данные к `NomenclatureItem`, поэтому ему можно скармливать и CSV, и заранее подготовленный JSON.

## Дальнейшие шаги

- Подключить централизованное хранилище (PostgreSQL/ClickHouse) вместо файловых артефактов.
- Завести пайплайн CI/CD для автоматического прогона `/quality` + `/train` в sandbox.
- Интегрировать сервис в существующий Go-бэкенд по HTTP/gRPC.

---

# Расширенная документация

## 1. REST API `ml_service.service:app`

- **Базовый URL:** `http://localhost:8085`
- **Формат:** JSON (`Content-Type: application/json`)
- **Аутентификация:** по умолчанию не требуется, но может быть добавлен API Key (`X-API-Key`), OAuth2 или mTLS — сервис воспринимает любые стандартные FastAPI зависимости, если их подключить в `build_app()`.
- **Заголовки, которые поддерживаются:**
  - `X-Request-ID` — пользовательский идентификатор запроса. Если передан, попадёт в `MonitoringStore`.
  - `X-User-IP` — если стоит прокси, можно принудительно указать IP клиента для логирования.
  - `X-Model-Key` — альтернативный способ указать ключ модели вместо поля `model_key` в теле (приоритет у поля).

### 1.1 `POST /train`
- **Назначение:** обучает модель на смеси эталонного и переданного датасета.
- **Тело:**
  - `version` (опц.) — желаемая версия датасета/модели.
  - `dataset_name` (опц.) — человекочитаемое название датасета.
  - `model_key` (опц., дефолт `nomenclature_classifier`) — имя модели; влияет на связку с БД.
  - `task_type` (опц., дефолт `classification`) — тип задачи. Любое иное значение пока отклоняется (HTTP 400).
  - `rewrite_dataset` (bool, дефолт false) — перезаписывать ли существующую версию датасета.
  - `refresh_baseline` (bool) — обновить baseline для дрейфа.
  - `target_field` (опц., дефолт `"label"`) — поле в `NomenclatureItem`, которое используется как целевая переменная для обучения. Может быть любым строковым полем (например, `label`, `type_hint`, `kind`).
  - `feature_fields` (опц.) — список полей, используемых как признаки для обучения. Если не указан, используются все доступные поля кроме `target_field`. Пример: `["name", "full_name", "kind", "unit", "okved_code", "hs_code"]`.
  - `items`/`data` — массив `NomenclatureItem`. Поле `label` (или указанное в `target_field`) может содержать произвольные строковые значения (например, `"Товар"`, `"Услуга"`, `"Тара"`, `"Набор"`, `"Работа"` и т.д.), а не только `"product"` и `"service"`.
- **Поведение:** 
  - Если `rewrite_dataset=false` и версия уже есть, вернётся 409.
  - Если метрики `accuracy >= 0.9` и `avg_confidence >= 0.75`, версия автоматически активируется и попадает в `models` + `metadata.json`.

### 1.2 `POST /predict`
- **Назначение:** инференс классификатора.
- **Тело:**
  - `items`: `List[NomenclatureItem]`.
  - `top_k` (1–2) — количество альтернатив.
  - `explain` (bool) — добавить объяснение.
  - `model_key` — ключ модели. Сейчас поддержан только `nomenclature_classifier`, иначе 400.
- **Ответ:** `PredictResponse` с результатами, версией модели и списком неожиданных записей.
- **Поведение:** 
  - Очередь с приоритетом (`PriorityScheduler`): чем больше информации в записи, тем быстрее попадёт на исполнение.
  - Таймаут берётся из `ML_PREDICT_TIMEOUT_SECONDS` (по умолчанию 120 секунд). При превышении возвращается 504.

### 1.3 `POST /quality`
- **Назначение:** лёгкий скрининг данных.
- **Тело:** `TrainRequest` (можно передавать только `items` или `data`).
- **Ответ:** `QualityReport` с флагами, сэмплами, списком дублей.
- **Особенности:** 
  - Дубликаты определяются по всему JSON (включая вложенные поля). 
  - Даже при ошибке запрос логируется (RequestTracker + `predictions_log`).

### 1.4 `POST /drift/check`
- **Назначение:** отчёт о дрейфе относительно baseline.
- **Тело:** `TrainRequest` (используются данные в `items`/`data`).
- **Ответ:** `DriftReport` с PSI, признаком `triggered`, версией baseline.
- **Ограничения:** если baseline отсутствует, 412 Precondition Failed.

### 1.5 `POST /drift/baseline`
- **Назначение:** принудительное обновление baseline.
- **Тело:** `TrainRequest` — данные, которые станут новым baseline. 
- **Ответ:** версия baseline и количество строк.

### 1.6 `GET /metadata`
- **Назначение:** JSON-история версий модели.
- **Ответ:** `MetadataEnvelope { current, history[] }`.
- **Использование:** UI на странице «Модели» строит график по этим данным.

### 1.7 `GET /features`
- **Назначение:** список версий фич в `feature_store`.
- **Ответ:** массив словарей `FeatureSetMeta`.

### 1.8 `POST /normalize`
- **Назначение:** предобработка текстов (нормализация, транслитерация, извлечение дат/чисел).
- **Тело:** `NormalizationRequest { items, transliterate_to, locale }`.
- **Ответ:** `NormalizationResponse` со списком нормализованных полей, комментариями, извлечёнными сущностями.
- **Параметры:**
  - `transliterate_to`: `latin`, `ascii`, `cyrillic` (любое другое значение → исходная строка без изменений).
  - `locale`: резерв, можно расширять для специфичных правил (например, `en`, `ru`).

### 1.9 Модельные утилиты (REST)
- `GET /models/{model_key}/history` — список версий модели из SQLite (`models`).
- `GET /models/{model_key}/responses` — последние записи из `predictions_log`.
- `GET /models/{model_key}/dataset/latest` — последняя информация о датасете.
- `GET /models/{model_key}/features/active` — документированный список фич (использует `FeatureBuilder.describe_features()`).
- `POST /models/{model_key}/activate` — ручная активация версии (тело `ModelActivationRequest { version, model_key? }`).

### 1.10 Загрузка датасетов через UI (`POST /datasets/upload`)
- **Назначение:** загрузка датасета через веб-интерфейс мониторинга (порт 6565).
- **Формат:** `multipart/form-data`.
- **Параметры:**
  - `file` (обязательный) — файл CSV или JSON.
  - `version_label` (обязательный) — версия датасета (например, `ds_2024_11_27`).
  - `dataset_name` (опционально, дефолт "Интерактивная загрузка") — человекочитаемое имя.
  - `model_key` (опционально, дефолт `nomenclature_classifier`) — ключ модели.
  - `task_type` (опционально, дефолт `classification`) — тип задачи.
  - `rewrite_dataset` (bool, дефолт false) — перезаписать существующую версию.
  - `trigger_training` (bool, дефолт false) — сразу запустить обучение после загрузки.
- **Поддерживаемые форматы:**
  - **CSV:** стандартный формат с заголовками (`name`, `full_name`, `unit`, `kind`, `label`, `type_hint`, `okved_code`, `hs_code`).
  - **JSON:** два варианта:
    1. Массив объектов: `[{ "name": "...", "full_name": "...", ... }, ...]`
    2. Объект с полем `data`: `{ "version": "...", "description": "...", "data": [{...}], "statistics": {...} }`
- **Метаданные JSON (опционально):**
  - `version` — автоматически заполняет поле `version_label` в форме.
  - `description` — автоматически заполняет поле `dataset_name`.
  - `created_date` — информационное поле (не используется для обучения).
  - `statistics` — объект со статистикой:
    - `total_items` — общее количество записей (отображается в UI, не используется для обучения).
    - `distribution` — распределение по категориям (например, `{"Товар": 3710, "Услуга": 3080}`).
- **Поведение:**
  - При загрузке JSON UI автоматически читает файл и заполняет поля формы (версия, имя).
  - Статистика из JSON отображается в отдельном блоке на странице загрузки.
  - Если `trigger_training=true`, после сохранения датасета автоматически вызывается `/train` с переданными данными.
  - При успешной загрузке происходит редирект на `/models` с flash-сообщением.

### 1.11 Внутренние заголовки (опции)
- `Accept-Language` — влияет только на будущие расширения UI/ответов (пока игнорируется).
- `X-Monitoring-Bypass` — можно использовать для отключения логирования (не реализовано по умолчанию, но легко добавить проверку в `RequestTracker`).

## 2. Структура сервиса (файл за файлом)

| Файл | Назначение / ключевые сущности |
|------|--------------------------------|
| `ml_service/__init__.py` | Помечает каталог как пакет, размещает метаданные (версия сервиса). |
| `ml_service/service.py` | Основное FastAPI-приложение. Функции: `build_app()`, обработчики `/health`, `/train`, `/predict`, `/quality`, `/drift/*`, `/normalize`, `/models/*`. Классы: `RequestTracker` — логирование и связь с MonitoringStore + `predictions_log`. Утилиты `compute_information_score`, `trim_payload`, `summarize_payload`, `build_audit_meta`. |
| `ml_service/config.py` | Pydantic Settings: пути к артефактам, лимиты, число воркеров, креды мониторинга. Поля доступны через `ML_` env vars. |
| `ml_service/schemas.py` | Все Pydantic-модели API: `NomenclatureItem`, `TrainRequest`, `PredictRequest`, `PredictResponse`, `QualityReport`, `MetadataRecord`, `NormalizationRequest/Response`. Содержит валидаторы, перечисление `NomenclatureType`. |
| `ml_service/model.py` | Класс `NomenclatureClassifier`: подготовка DataFrame, сбор pipeline (TF-IDF + OneHot + MLPClassifier), методы `train`, `predict`, `_persist_pipeline`, `load_latest`. Использует `ExplainabilityEngine`. |
| `ml_service/feature_store.py` | Класс `FeatureStore` для версионирования parquet-фич + `FeatureBuilder` (создаёт производные признаки), датакласс `FeatureSetMeta`. Метод `describe_features()` возвращает человекочитаемые описания для UI. |
| `ml_service/data_quality.py` | Класс `DataQualityGuard`: проверка обязательных полей, дублей (по JSON), длины, валидности кодов. Возвращает `QualityReport`. |
| `ml_service/dataset_manager.py` | `ReferenceDatasetManager`: хранение эталонного датасета (Parquet), смешивание клиентских данных, оверсемплинг эталона, формирование отчёта `MixingReport`. |
| `ml_service/drift_monitor.py` | `DriftMonitor`: управление baseline, расчёт PSI и Jensen–Shannon, формирование `DriftReport`. |
| `ml_service/metadata.py` | `MetadataStore`: JSON-хранилище истории метрик (`metadata.json`). Методы `_load`, `_save`, `add_record`, `get_envelope`. |
| `ml_service/repository.py` | `MlRepository`: обёртка над SQLite (`ml_store.db`). Методы: `persist_dataset`, `save_model_version`, `activate_model`, `log_request_start`, `finalize_request_log`, `latest_dataset_info`, `list_models`, `recent_responses`, `worker_usage`. Обрабатывает поля `model_key`, `task_type`, снапшоты, расширяет таблицы при необходимости (ALTER TABLE). |
| `ml_service/priority_scheduler.py` | Очередь задач с приоритетом. Класс `PriorityScheduler`: фоновые треды, приоритеты по величине информации. |
| `ml_service/monitoring/app.py` | Второе FastAPI-приложение (UI). Роуты: `/` (главная), `/workers`, `/db`, `/requests`, `/models`, `/datasets/upload` (поддерживает CSV и JSON с автозаполнением), `/models/activate`, REST `/monitoring/*`. Использует Jinja2 (`templates/`) и Plotly, валидацию загрузок, вызов `/train` через httpx. |
| `ml_service/monitoring/store.py` | `MonitoringStore`: связь с SQLite (таблицы `monitoring_requests`, `worker_snapshots`, `system_events`, `admin_logs`). Методы `start_request`, `update_request`, `record_event`, `worker_usage`, `db_summary`, `fetch_table_preview`, `initialize_database`, `export_statistics`. |
| `ml_service/templates/*.html` | UI слои (Jinja + PicoCSS + Plotly). `base.html` — макет, `overview.html`, `workers.html`, `requests.html`, `models.html`, `database.html`. |
| `ml_service/text_processing.py` | `TextNormalizer`: нормализация строк, извлечение дат, чисел, транслитерация (через `unidecode`). API `/normalize` возвращает `NormalizationResponse`. |
| `ml_service/regional_validation.py` | `RegionalValidator`: проверяет ISO-коды, кодировки и региональные ограничения. |
| `ml_service/feature_store.py` | см. выше; предоставляет `FeatureStore` и описания фич. |
| `ml_service/explainability.py` | `ExplainabilityEngine`: permutation importance + локальные объяснения (лексические маркеры, ОКВЭД). Методы `update_global_importance`, `explain_instance`. |
| `ml_service/data` / `datasets/` | Примеры данных (CSV, parquet). Служебные скрипты расположены в `ml_service/scripts`. |
| `ml_service/sql/sqlite_schema.sql` | Актуальная схема SQLite: таблицы датасетов, версий, пользовательских правок, моделей, запросов, метаданных мониторинга. Используется при инициализации/пересоздании `ml_store.db`. |
| `ml_service/README.md` | Текущий документ. |

> Примечание: файлы `tools/*.go` и прочие утилиты находятся за пределами папки `ml_service`; их работа описана в соответствующих README или комментариях в исходниках.

## 3. Frontend / мониторинг

- **Приложение:** `ml_service.monitoring.app` (порт 6565 по умолчанию).
- **Стек:** FastAPI + Jinja2 + PicoCSS + Plotly + WebSocket.
- **Навигация:** Главная → Воркеры → База данных → Запросы → Модели.

### 3.1 Главная (`overview.html`)
- Виджет активных моделей (карточки с точностью/F1/уверенностью).
- Статистика запросов за 24 часа (выполнено, ошибок, в очереди, в работе).
- Состояние актуального датасета (версия, строки, последнее использование).
- Лента событий и последние запросы (ID, тип, статус, IP, воркеры).
- Plotly-график активности (`requests_timeseries`), данные передаются через `<script type="application/json">`.

### 3.2 Воркеры (`workers.html`)
- Плитки с метриками (активные, очередь, завершено, ошибки, загрузка CPU/RAM).
- «Использование воркеров» — таблица последних записей из `predictions_log`.
- Блоки с очередью (`status=queued`) и текущими запросами (`status=running`), обновляются через MonitoringStore.
- История событий + график динамики запросов.

### 3.3 База данных (`database.html`)
- Карточки всех таблиц `ml_store.db`, отображает число столбцов и записей.
- Выбор таблицы → предпросмотр через «псевдотаблицу» (flex/grid, не `<table>`), чтобы соответствовать требованию «визуальные таблицы без `<table>`».
- Ошибки SQLite выводятся в алерте.

### 3.4 Запросы (`requests.html`)
- Фильтр по статусу/типу.
- Метрики окна (всего, выполнено, ошибок, в работе/очереди, запросы с дообучением).
- Plotly-график (bar). Данные прокидываются в JSON-скрипт.
- Список запросов (ID, тип, статус, приоритет, прогресс, старт, ошибка).

### 3.5 Модели (`models.html`)
- Текущая версия (плитки по метрикам).
- График истории метрик (Plotly line).
- Активные версии + форма ручной активации (POST `/models/activate`).
- История версий в формате «псевдотаблицы».
- Форма загрузки датасета (эндпойнт `/datasets/upload`): 
  - **Поддерживает CSV и JSON** (`.csv`, `.json`).
  - **JSON-формат:** может быть массивом объектов `[{...}]` или объектом с полем `data: [{...}]` и опциональными метаданными:
    - `version` — версия датасета (автозаполняет поле "Версия").
    - `description` — описание (автозаполняет "Имя датасета").
    - `created_date` — дата создания (информационное).
    - `statistics` — объект со статистикой (не используется для обучения, но отображается в UI):
      - `total_items` — общее количество записей.
      - `distribution` — распределение по категориям (например, `{"Товар": 3710, "Услуга": 3080}`).
  - При выборе JSON-файла JavaScript автоматически читает его и заполняет поля формы (версия, имя), а также отображает статистику в отдельном блоке.
  - Связывает датасет с моделью/версией, опционально запускает `/train`.
- Отчёты по запросам (журнал последних `predictions_log`).
- Список фич из `FeatureBuilder.describe_features()` с описанием.

### 3.6 Шаблоны и стили
- `base.html` подключает PicoCSS, определяет палитры светлой/тёмной темы, кастомные компоненты (псевдотаблицы, карточки, flash-сообщения).
- Все страницы наследуют `base.html`, вставляют контент в `{% block content %}`.
- Plotly и данные графиков передаются через JSON в `<script type="application/json">`, что делает HTML пригодным для линтеров.

### 3.7 Мониторинг API
- `/monitoring/workers/stats`, `/monitoring/db/status`, `/monitoring/requests/active` — REST-энндпоинты для внешнего доступа к статистике.
- `/ws/workers` — WebSocket, каждые ~15 сек отправляет снапшоты.
- Административные действия (`/monitoring/actions/*`) защищены Basic Auth (`ML_MONITOR_ADMIN_*`).

---

Документ теперь содержит полный справочник по API, структуре модулей и фронтенду. При изменении кода рекомендуется обновлять этот раздел, чтобы поддерживать соответствие документации фактическому поведению.*** End Patch

