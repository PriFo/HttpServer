# Улучшения обработки ошибок и логирования

## Внесенные изменения

### 1. Создана централизованная система логирования (`frontend/lib/logger.ts`)

**Возможности:**
- Поддержка уровней логирования: `debug`, `info`, `warn`, `error`
- Автоматическое форматирование сообщений с контекстом
- История логов (до 100 записей) для отладки
- Измерение времени выполнения операций через `logger.time()`
- Автоматическое отключение debug логов в production

**Использование:**
```typescript
import { logger } from '@/lib/logger'

logger.debug('Debug message', { context }, 'ComponentName')
logger.info('Info message', { context }, 'ComponentName')
logger.warn('Warning message', { context }, 'ComponentName')
logger.error('Error message', error, { context }, 'ComponentName')

// Измерение времени выполнения
await logger.time('operation name', async () => {
  // выполнение операции
}, { context }, 'ComponentName')
```

### 2. Создана централизованная обработка ошибок (`frontend/lib/error-handler.ts`)

**Возможности:**
- Автоматический анализ различных типов ошибок
- Преобразование ошибок в понятные сообщения для пользователя
- Определение возможности повторной попытки
- Определение HTTP статус-кодов
- Класс `AppError` для типизированных ошибок

**Типы ошибок, которые обрабатываются:**
- `TimeoutError` - превышено время ожидания
- `AbortError` - запрос прерван
- `NetworkError` - ошибки сети
- `SyntaxError` (JSON parsing) - ошибки парсинга
- HTTP ошибки (400, 401, 403, 404, 429, 500, 502, 503, 504)
- Неизвестные ошибки

**Использование:**
```typescript
import { handleError, analyzeError } from '@/lib/error-handler'

try {
  // код
} catch (error) {
  const errorDetails = handleError(error, 'ComponentName', 'operation', { context })
  // errorDetails содержит:
  // - message: понятное сообщение для пользователя
  // - code: код ошибки
  // - statusCode: HTTP статус (если есть)
  // - isRetryable: можно ли повторить
  // - retryAfter: через сколько секунд можно повторить
}
```

### 3. Обновлены компоненты нормализации

**Обновленные файлы:**

1. **`normalization-preview-stats.tsx`**:
   - Заменены все `console.log/warn/error` на `logger`
   - Добавлена централизованная обработка ошибок через `handleError`
   - Улучшены сообщения об ошибках для пользователя
   - Добавлено логирование всех операций (загрузка, обновление, экспорт, кэширование)

2. **`normalization-process-tab.tsx`**:
   - Заменены все `console.log/warn/error` на `logger`
   - Добавлена обработка ошибок через `handleErrorUtil`
   - Улучшено логирование операций с проектами и базами данных

3. **`normalization-results-table.tsx`**:
   - Заменены все `console.log/warn/error` на `logger`
   - Добавлена обработка ошибок при загрузке групп
   - Улучшено логирование поиска проектов

4. **`normalization-preview-results.tsx`**:
   - Заменен `console.debug` на `logger.debug`
   - Добавлена обработка ошибок при загрузке превью

5. **API Route `preview-stats/route.ts`**:
   - Добавлено логирование всех этапов запроса
   - Измерение времени выполнения запросов
   - Улучшена обработка ошибок с детальным логированием
   - Добавлена информация о производительности

### 4. Улучшения в обработке ошибок

**До:**
```typescript
catch (err) {
  console.error('Error:', err)
  setError('Произошла ошибка')
}
```

**После:**
```typescript
catch (err) {
  const errorDetails = handleError(err, 'ComponentName', 'operation', { context })
  setError(errorDetails.message) // Понятное сообщение для пользователя
  // Автоматически логируется с полным контекстом
}
```

### 5. Контекст логирования

Все логи теперь включают:
- Имя компонента
- Название операции
- Параметры запроса (clientId, projectId, normalizationType и т.д.)
- Время выполнения (где применимо)
- Детали ошибки (статус, сообщение, stack trace)

### 6. Примеры логов

**Успешная операция:**
```
2024-01-15T10:30:45.123Z INFO [NormalizationPreviewStats] Preview stats fetched successfully {
  "clientId": 1,
  "projectId": 2,
  "normalizationType": "both",
  "totalDatabases": 8,
  "totalRecords": 70440,
  "duration": "1234.56ms"
}
```

**Ошибка:**
```
2024-01-15T10:30:45.123Z ERROR [NormalizationPreviewStats] Error in NormalizationPreviewStats.fetchStats: Превышено время ожидания ответа от сервера {
  "clientId": 1,
  "projectId": 2,
  "normalizationType": "both",
  "url": "...",
  "component": "NormalizationPreviewStats",
  "operation": "fetchStats"
}
Error: timeout
Stack trace: ...
```

## Преимущества

1. **Единообразие**: Все логи в одном формате с контекстом
2. **Отладка**: Легко найти проблему по контексту и времени
3. **Производительность**: Измерение времени выполнения операций
4. **Пользовательский опыт**: Понятные сообщения об ошибках
5. **История**: Сохранение последних 100 логов для анализа
6. **Production-ready**: Debug логи автоматически отключены в production

## Рекомендации

1. Используйте `logger` вместо `console.log/warn/error`
2. Используйте `handleError` для обработки ошибок
3. Всегда передавайте контекст (clientId, projectId и т.д.)
4. Используйте `logger.time()` для измерения производительности
5. Проверяйте `errorDetails.isRetryable` для автоматических повторов

