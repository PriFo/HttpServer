# Инвентаризация HTTP-маршрутов нормализации

## Обзор

Данный отчет содержит полную инвентаризацию всех HTTP-маршрутов, связанных с функциональностью нормализации данных в веб-приложении на Gin.

**Дата анализа:** 2024

**Файлы анализа:**
- `server/handlers/normalization.go` - обработчики
- `server/server_start_shutdown.go` - регистрация Gin роутов
- `internal/api/routes/normalization_routes.go` - legacy роуты

---

## Таблица маршрутов

| HTTP Метод | Путь | Функция-обработчик | Статус Gin | Статус Legacy | Статус Swagger |
|------------|------|-------------------|------------|---------------|----------------|
| GET | `/api/normalization/events` | `HandleNormalizationEvents` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/status` | `HandleNormalizationStatus` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| POST | `/api/clients/:clientId/projects/:projectId/normalization/start` | `HandleStartClientProjectNormalization` | ✅ Зарегистрирован | ❌ Отсутствует | ✅ Документирован |
| GET | `/api/clients/:clientId/projects/:projectId/normalization/status` | `HandleGetClientProjectNormalizationStatus` | ✅ Зарегистрирован | ❌ Отсутствует | ✅ Документирован |
| POST | `/api/normalization/stop` | `HandleNormalizationStop` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/stats` | `HandleNormalizationStats` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/groups` | `HandleNormalizationGroups` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/group-items` | `HandleNormalizationGroupItems` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| POST | `/api/normalization/start` | `HandleStartVersionedNormalization` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| POST | `/api/normalization/apply-patterns` | `HandleApplyPatterns` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| POST | `/api/normalization/apply-ai` | `HandleApplyAI` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| GET | `/api/normalization/history` | `HandleGetSessionHistory` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| POST | `/api/normalization/revert` | `HandleRevertStage` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| POST | `/api/normalization/apply-categorization` | `HandleApplyCategorization` | ✅ Зарегистрирован | ✅ Зарегистрирован | ✅ Документирован |
| POST | `/api/normalize/start` | `HandleNormalizeStart` | ❌ Отсутствует | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/item-attributes/:id` | `HandleNormalizationItemAttributes` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/export-group` | `HandleNormalizationExportGroup` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/pipeline/stats` | `HandlePipelineStats` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/pipeline/stage-details` | `HandleStageDetails` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/export` | `HandleExport` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET/PUT/POST | `/api/normalization/config` | `HandleNormalizationConfig` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/databases` | `HandleNormalizationDatabases` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/tables` | `HandleNormalizationTables` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/normalization/columns` | `HandleNormalizationColumns` | ✅ Зарегистрирован | ✅ Зарегистрирован | ❌ Отсутствует |
| GET | `/api/clients/:clientId/projects/:projectId/normalization/preview-stats` | `HandleGetClientProjectNormalizationPreviewStats` | ✅ Зарегистрирован | ❌ Отсутствует | ✅ Документирован |

---

## Статистика

- **Всего обработчиков:** 25
- **Зарегистрировано в Gin:** 25 (100%) ✅
- **Зарегистрировано в Legacy:** 22 (88%)
- **Имеют Swagger-документацию:** 8 (32%)
- **Полностью зарегистрированы (Gin + Legacy):** 22 (88%)
- **Отсутствуют в Gin router:** 0 (0%) ✅
- **Отсутствуют в Legacy routes:** 3 (12%)

**Обновлено:** 2025-11-25 - Все роуты нормализации теперь зарегистрированы в Gin router

---

## Замечания и рекомендации

### 1. Отсутствующие роуты в Gin router

✅ **ИСПРАВЛЕНО:** Все обработчики нормализации теперь зарегистрированы в Gin router.

Добавленные роуты (2025-11-25):
- ✅ `POST /api/normalization/stop` - `HandleNormalizationStop`
- ✅ `GET /api/normalization/pipeline/stage-details` - `HandleStageDetails`
- ✅ `GET /api/normalization/export` - `HandleExport`
- ✅ `GET/PUT/POST /api/normalization/config` - `HandleNormalizationConfig`
- ✅ `GET /api/normalization/databases` - `HandleNormalizationDatabases`
- ✅ `GET /api/normalization/tables` - `HandleNormalizationTables`
- ✅ `GET /api/normalization/columns` - `HandleNormalizationColumns`

#### 1.1. `HandleNormalizeStart` (legacy wrapper)
- **Путь:** `POST /api/normalize/start`
- **Примечание:** Это legacy wrapper, который перенаправляет на `HandleStartClientProjectNormalization`
- **Статус:** Оставлен только в legacy routes для обратной совместимости

### 2. Отсутствующие роуты в Legacy routes

Следующие обработчики зарегистрированы только в Gin router, но отсутствуют в legacy routes:

#### 2.1. `HandleStartClientProjectNormalization`
- **Путь:** `POST /api/clients/:clientId/projects/:projectId/normalization/start`
- **Примечание:** Это новый API для работы с проектами клиентов. Legacy routes не поддерживают параметризованные пути.
- **Рекомендация:** Если требуется обратная совместимость, можно добавить альтернативный путь в legacy routes.

#### 2.2. `HandleGetClientProjectNormalizationStatus`
- **Путь:** `GET /api/clients/:clientId/projects/:projectId/normalization/status`
- **Рекомендация:** Аналогично предыдущему пункту.

#### 2.3. `HandleGetClientProjectNormalizationPreviewStats`
- **Путь:** `GET /api/clients/:clientId/projects/:projectId/normalization/preview-stats`
- **Рекомендация:** Аналогично предыдущему пункту.

### 3. Отсутствующая Swagger-документация

Следующие обработчики не имеют Swagger-аннотаций:

1. `HandleNormalizationEvents` - GET `/api/normalization/events`
2. `HandleNormalizationStop` - POST `/api/normalization/stop`
3. `HandleNormalizationStats` - GET `/api/normalization/stats`
4. `HandleNormalizationGroups` - GET `/api/normalization/groups`
5. `HandleNormalizationGroupItems` - GET `/api/normalization/group-items`
6. `HandleNormalizeStart` - POST `/api/normalize/start`
7. `HandleNormalizationItemAttributes` - GET `/api/normalization/item-attributes/:id`
8. `HandleNormalizationExportGroup` - GET `/api/normalization/export-group`
9. `HandlePipelineStats` - GET `/api/normalization/pipeline/stats`
10. `HandleStageDetails` - GET `/api/normalization/pipeline/stage-details`
11. `HandleExport` - GET `/api/normalization/export`
12. `HandleNormalizationConfig` - GET/PUT/POST `/api/normalization/config`
13. `HandleNormalizationDatabases` - GET `/api/normalization/databases`
14. `HandleNormalizationTables` - GET `/api/normalization/tables`
15. `HandleNormalizationColumns` - GET `/api/normalization/columns`

**Рекомендация:** Добавить Swagger-аннотации для всех публичных API-эндпоинтов. Пример структуры:
```go
// @Summary Краткое описание
// @Description Подробное описание
// @Tags normalization
// @Accept json (если требуется)
// @Produce json
// @Param param_name param_type param_location required "Описание параметра"
// @Success 200 {object} ResponseType "Описание успешного ответа"
// @Failure 400 {object} ErrorResponse "Описание ошибки"
// @Router /api/normalization/path [method]
```

### 4. Несоответствия в путях

#### 4.1. Различие в формате параметров
- **Gin router:** Использует формат `:id` (например, `/item-attributes/:id`)
- **Legacy routes:** Использует формат с завершающим слешем (например, `/item-attributes/`)
- **Рекомендация:** Унифицировать формат. Предпочтительно использовать формат Gin router (`:id`), так как он более явный.

### 5. Конфликты и дубликаты

#### 5.1. Дублирование путей `/api/normalization/start`
- **Gin router:** `HandleStartVersionedNormalization` - POST `/api/normalization/start`
- **Legacy routes:** `HandleNormalizeStart` - POST `/api/normalize/start` (разные пути, конфликта нет)
- **Статус:** Конфликта нет, пути разные.

#### 5.2. Дублирование путей `/api/normalization/status`
- **Gin router:** `HandleNormalizationStatus` - GET `/api/normalization/status`
- **Legacy routes:** `HandleNormalizationStatus` - GET `/api/normalization/status`
- **Статус:** Оба используют один и тот же обработчик, конфликта нет.

### 6. Особые случаи

#### 6.1. `HandleNormalizationConfig` - поддержка нескольких HTTP-методов
- Обработчик поддерживает GET, PUT и POST методы
- В legacy routes зарегистрирован один путь для всех методов
- **Рекомендация:** При добавлении в Gin router зарегистрировать отдельно для каждого метода (см. раздел 1.5).

#### 6.2. `HandleNormalizeStart` - legacy wrapper
- Это обертка, которая перенаправляет на `HandleStartClientProjectNormalization`
- Используется для обратной совместимости
- **Рекомендация:** Оставить только в legacy routes, если не требуется регистрация в Gin router.

---

## Приоритетные действия

### Высокий приоритет
1. ✅ **ВЫПОЛНЕНО:** Добавить отсутствующие роуты в Gin router (8 обработчиков)
2. ⏳ Добавить Swagger-документацию для всех публичных API (15 обработчиков)

### Средний приоритет
3. Рассмотреть необходимость регистрации client-project роутов в legacy routes
4. Унифицировать формат параметров в путях

### Низкий приоритет
5. Документировать legacy wrapper `HandleNormalizeStart`
6. Проверить использование всех legacy routes и возможность их миграции в Gin router

---

## Заключение

Анализ показал, что большинство обработчиков нормализации зарегистрированы в legacy routes (88%), но только 68% зарегистрированы в Gin router. Это указывает на необходимость миграции legacy роутов в современный Gin router.

Также выявлен значительный пробел в Swagger-документации: только 32% обработчиков имеют аннотации. Рекомендуется добавить документацию для всех публичных API-эндпоинтов.

Основные проблемы:
- 8 обработчиков отсутствуют в Gin router
- 15 обработчиков не имеют Swagger-документации
- Несоответствие в формате параметров между Gin и legacy routes

После устранения выявленных проблем API нормализации будет полностью документирован и единообразно зарегистрирован в обоих системах роутинга.
