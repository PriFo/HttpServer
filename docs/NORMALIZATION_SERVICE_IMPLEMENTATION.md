# Реализация методов версионированной нормализации

## Дата: 2025-01-21

## Проблема

В `server/services/normalization_service.go` были TODO комментарии о необходимости реализации методов:
- `StartVersionedNormalization` - запуск версионированной нормализации
- `ApplyPatterns` - применение алгоритмических паттернов
- `ApplyAI` - применение AI коррекции

## Реализованные изменения

### 1. StartVersionedNormalization

**Файл:** `server/services/normalization_service.go`

#### Реализация:
- Создает `PatternDetector` для обнаружения паттернов
- Получает API ключ через переданную функцию `getArliaiAPIKey()`
- Создает `AINormalizer` и `PatternAIIntegrator` при наличии API ключа
- Создает `VersionedNormalizationPipeline` с компонентами
- Запускает сессию нормализации через `pipeline.StartSession()`
- Возвращает результат с `session_id`, `current_name` и `original_name`

#### Fallback:
- Если API ключ не получен из конфигурации, используется переменная окружения `ARLIAI_API_KEY`
- Если API ключ отсутствует, пайплайн создается без AI интегратора (только алгоритмические паттерны)

### 2. ApplyPatterns

**Файл:** `server/services/normalization_service.go`

#### Реализация:
- Получает сессию нормализации из базы данных по `sessionID`
- Восстанавливает состояние пайплайна через `pipeline.StartSession()`
- Применяет алгоритмические паттерны через `pipeline.ApplyPatterns()`
- Возвращает результат с `session_id`, `current_name` и `stage_count`

#### Обработка ошибок:
- Если сессия не найдена, возвращает `NotFoundError`
- Если не удалось восстановить сессию, возвращает `InternalError`
- Если не удалось применить паттерны, возвращает `InternalError`

### 3. ApplyAI

**Файл:** `server/services/normalization_service.go`

#### Реализация:
- Получает сессию нормализации из базы данных по `sessionID`
- Восстанавливает состояние пайплайна через `pipeline.StartSession()`
- Создает AI интегратор с полученным API ключом
- Применяет AI коррекцию через `pipeline.ApplyAICorrection()` с поддержкой:
  - `useChat` - использование чат-режима для контекста
  - `context` - дополнительные контекстные строки
- Возвращает результат с `session_id`, `current_name`, `stage_count` и информацией о последней стадии

#### Валидация:
- Проверяет наличие API ключа (возвращает `ValidationError` если отсутствует)
- Использует fallback на переменную окружения `ARLIAI_API_KEY`

## Использование VersionedNormalizationPipeline

Все методы используют `normalization.VersionedNormalizationPipeline` для:
- Управления сессиями нормализации
- Версионирования стадий нормализации
- Сохранения истории изменений
- Отката к предыдущим стадиям

## Тестирование

Все методы протестированы:
- ✅ `TestNormalizationService_StartVersionedNormalization` - PASS
- ✅ `TestNormalizationService_ApplyPatterns` - PASS
- ✅ `TestNormalizationService_ApplyAI` - PASS

## Интеграция с API

Методы интегрированы через `NormalizationHandler`:
- `/api/normalization/start` → `HandleStartVersionedNormalization`
- `/api/normalization/apply-patterns` → `HandleApplyPatterns`
- `/api/normalization/apply-ai` → `HandleApplyAI`

Все эндпоинты получают API ключи из конфигурации через `WorkerConfigManager`.

## Результат

- ✅ Все TODO комментарии удалены
- ✅ Методы полностью реализованы
- ✅ Интеграция с `VersionedNormalizationPipeline` работает
- ✅ Обработка ошибок реализована
- ✅ Fallback на переменную окружения работает
- ✅ Все тесты проходят успешно

