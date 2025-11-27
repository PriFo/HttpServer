# Тестирование получения моделей с API ключом из БД

## Описание

Тесты проверяют, что система корректно получает модели из различных провайдеров, используя API ключ из базы данных через `WorkerConfigManager`.

## Доступные тесты

### 1. TestFetchModelsWithAPIKeyFromDB

Интеграционный тест, который проверяет получение моделей Arliai с API ключом из БД.

**Запуск:**
```bash
# Установите API ключ в переменную окружения
$env:ARLIAI_API_KEY="your-api-key-here"

# Запустите тест
go test -v -run TestFetchModelsWithAPIKeyFromDB ./server -timeout 60s
```

**Что проверяет:**
- Сохранение API ключа в БД через `WorkerConfigManager`
- Получение провайдера из БД
- Загрузка моделей через `fetchProviderModels()`
- Вывод информации о полученных моделях

**Ожидаемый результат:**
- Тест получает список моделей из API
- Выводит информацию о количестве моделей и первых 10 моделях
- Показывает количество активных моделей

### 2. TestFetchModelsMultipleProviders

Тест проверяет получение моделей для нескольких провайдеров (Arliai, OpenRouter, HuggingFace).

**Запуск:**
```bash
# Установите API ключи для нужных провайдеров
$env:ARLIAI_API_KEY="your-arliai-key"
$env:OPENROUTER_API_KEY="your-openrouter-key"
$env:HUGGINGFACE_API_KEY="your-huggingface-key"

# Запустите тест
go test -v -run TestFetchModelsMultipleProviders ./server -timeout 60s
```

**Что проверяет:**
- Получение моделей для каждого провайдера с API ключом
- Корректная обработка провайдеров без API ключа (пропуск)
- Вывод информации о полученных моделях для каждого провайдера

### 3. TestFetchProviderModels_HuggingFace

Тест проверяет получение статического списка моделей HuggingFace (не требует API ключа).

**Запуск:**
```bash
go test -v -run TestFetchProviderModels_HuggingFace ./server
```

**Ожидаемый результат:**
- Возвращает 7 моделей из статического списка
- Не требует реального API ключа

## Как работает получение API ключа

1. **Приоритет 1: Из БД через WorkerConfigManager**
   - API ключ сохраняется в таблице `worker_config` в `service.db`
   - Загружается при создании `WorkerConfigManager` через `loadConfig()`
   - Доступен через `GetActiveProvider()` → `provider.APIKey`

2. **Приоритет 2: Из переменной окружения (fallback)**
   - Если ключ не найден в БД, используется переменная окружения
   - `ARLIAI_API_KEY`, `OPENROUTER_API_KEY`, `HUGGINGFACE_API_KEY`, `EDENAI_API_KEY`

## Пример вывода теста

```
=== RUN   TestFetchModelsWithAPIKeyFromDB
    worker_config_models_test.go:355: Testing with provider: arliai, API key set: true
    worker_config_models_test.go:380: Successfully fetched 15 models from API
    worker_config_models_test.go:385: First 10 models:
    worker_config_models_test.go:386:   - GLM-4.5-Air (ID: glm-4.5-air, Status: active)
    worker_config_models_test.go:386:   - GLM-4.5-Flash (ID: glm-4.5-flash, Status: active)
    ...
    worker_config_models_test.go:400: Found 15 active models
--- PASS: TestFetchModelsWithAPIKeyFromDB (2.45s)
```

## Обработка ошибок

Тесты корректно обрабатывают следующие ситуации:

- **Отсутствие API ключа**: Тест пропускается с сообщением
- **Неверный API ключ**: Выводится ошибка "Unauthorized"
- **Таймаут**: Выводится предупреждение о таймауте
- **Ошибка сети**: Выводится информация об ошибке

## Примечания

- Тесты требуют реального API ключа для полной проверки
- HuggingFace возвращает статический список (не требует API ключа)
- Тесты пропускаются в режиме `-short`
- Таймаут для интеграционных тестов: 30-60 секунд

