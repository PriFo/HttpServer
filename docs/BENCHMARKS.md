# Система управления эталонами (Benchmarks)

## Обзор

Система эталонов позволяет хранить проверенные и подтвержденные данные, которые используются как приоритетный источник для нормализации перед обращением к внешним AI-сервисам.

## Архитектура

### База данных

Эталоны хранятся в отдельной базе данных `data/benchmarks.db` с двумя основными таблицами:

- **benchmarks** - основные эталонные записи
- **benchmark_variations** - вариации названий для улучшения поиска

### Компоненты

1. **Backend (Go)**
   - `database/benchmarks_db.go` - работа с БД эталонов
   - `server/services/benchmark_service.go` - бизнес-логика
   - `server/handlers/benchmark.go` - HTTP обработчики
   - `normalization/counterparty_normalizer.go` - интеграция в нормализацию контрагентов
   - `normalization/normalizer.go` - интеграция в нормализацию номенклатуры

2. **Frontend (Next.js/React)**
   - `frontend/app/benchmarks/page.tsx` - страница управления
   - `frontend/components/quality/CreateBenchmarkDialog.tsx` - диалог создания
   - `frontend/app/api/benchmarks/*` - API routes

## API Endpoints

### Создание эталона

```http
POST /api/benchmarks
Content-Type: application/json

{
  "entity_type": "counterparty",
  "name": "ООО Ромашка",
  "data": {
    "inn": "1234567890",
    "address": "Москва, ул. Тестовая, 1"
  },
  "variations": ["Ромашка ООО", "ООО Ромашка"]
}
```

### Создание эталона из загрузки

```http
POST /api/benchmarks/from-upload
Content-Type: application/json

{
  "upload_id": "uuid-загрузки",
  "item_ids": ["1", "2", "3"],
  "entity_type": "counterparty"
}
```

### Поиск эталона

```http
GET /api/benchmarks/search?name=Ромашка ООО&type=counterparty
```

### Получение списка эталонов

```http
GET /api/benchmarks?type=counterparty&active=true&limit=20&offset=0
```

### Получение эталона по ID

```http
GET /api/benchmarks/{id}
```

### Обновление эталона

```http
PUT /api/benchmarks/{id}
Content-Type: application/json

{
  "name": "ООО Обновленная Ромашка",
  "data": {
    "inn": "0987654321"
  },
  "is_active": true
}
```

### Удаление эталона (мягкое)

```http
DELETE /api/benchmarks/{id}
```

## Использование в нормализации

### Приоритет поиска

1. **Эталоны** - сначала проверяются эталоны
2. **AI-сервисы** - если эталон не найден, используется AI

### Интеграция

Эталоны автоматически используются в:
- Нормализации контрагентов (`CounterpartyNormalizer`)
- Нормализации номенклатуры (`Normalizer`)

### Пример работы

```go
// В CounterpartyNormalizer.ProcessNormalization
if cn.benchmarkFinder != nil {
    normalized, found, err := cn.benchmarkFinder.FindBestMatch(cp.Name, "counterparty")
    if err == nil && found {
        // Используем эталон, пропускаем AI
        normalizedName = normalized
        benchmarkFound = true
    }
}

// Если эталон не найден, используем AI
if !benchmarkFound && cn.nameNormalizer != nil {
    normalizedName, err = cn.nameNormalizer.NormalizeName(...)
}
```

## Создание эталонов

### Из дубликатов

1. Перейдите на страницу `/quality/duplicates`
2. Выберите группу дубликатов
3. Нажмите "Создать эталон"
4. Выберите элементы и подтвердите

### Вручную

1. Перейдите на страницу `/benchmarks`
2. Нажмите "Создать эталон"
3. Заполните форму:
   - Тип сущности (контрагент/номенклатура)
   - Название
   - Данные (JSON)
   - Вариации названий

## Управление эталонами

### Фильтрация

- По типу (counterparty/nomenclature)
- По статусу (активные/все)
- Поиск по названию

### Редактирование

- Изменение названия
- Обновление данных
- Добавление/удаление вариаций
- Активация/деактивация

### Удаление

Удаление является "мягким" - эталон деактивируется (`is_active = false`), но не удаляется из БД.

## Тестирование

### Unit тесты

```bash
go test ./server/services/... -v
go test ./server/handlers/... -v
```

### E2E тесты

```bash
npm run test:e2e -- benchmarks.spec.ts
```

## Производительность

- Индексы на `entity_type`, `name`, `is_active`
- Индекс на `benchmark_variations.variation` для быстрого поиска
- Connection pooling для БД эталонов

## Безопасность

- Валидация типов сущностей
- Проверка прав доступа (если реализовано)
- Санитизация входных данных

## Мониторинг

- Количество эталонов по типам
- Статистика использования (сколько раз эталон был использован)
- Метрики производительности поиска

