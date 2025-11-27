# Документация по использованию fetch-utils.ts

## Обзор

`fetch-utils.ts` предоставляет единообразные утилиты для выполнения HTTP-запросов с автоматической обработкой таймаутов, сетевых ошибок и повторных попыток.

## Основные функции

### `fetchJson<T>(url, options?)`

Выполняет fetch-запрос и автоматически парсит JSON ответ.

**Параметры:**
- `url: string` - URL для запроса
- `options?: FetchOptions` - Опции запроса (см. ниже)

**Возвращает:** `Promise<T>` - Распарсенные JSON данные

**Пример:**
```typescript
import { fetchJson } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

try {
  const data = await fetchJson<QualityStats>(
    `/api/quality/stats?database=${database}`,
    {
      timeout: QUALITY_TIMEOUTS.FAST,
      cache: 'no-store',
    }
  )
  setStats(data)
} catch (err) {
  const errorMessage = getErrorMessage(err, 'Не удалось загрузить статистику')
  setError(errorMessage)
}
```

### `fetchWithTimeout(url, options?)`

Выполняет fetch-запрос с таймаутом и обработкой ошибок. Возвращает `Response` объект.

**Параметры:**
- `url: string` - URL для запроса
- `options?: FetchOptions` - Опции запроса

**Возвращает:** `Promise<Response>`

**Пример:**
```typescript
import { fetchWithTimeout } from '@/lib/fetch-utils'

try {
  const response = await fetchWithTimeout('/api/data', {
    timeout: 10000,
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ data: 'test' })
  })
  const data = await response.json()
} catch (error) {
  if (error.isTimeout) {
    console.error('Таймаут запроса')
  } else if (error.isNetworkError) {
    console.error('Ошибка сети')
  }
}
```

### `getErrorMessage(error, fallback?)`

Получает понятное сообщение об ошибке для отображения пользователю.

**Параметры:**
- `error: unknown` - Ошибка (FetchError, Error или строка)
- `fallback?: string` - Сообщение по умолчанию (по умолчанию: 'Произошла ошибка')

**Возвращает:** `string` - Понятное сообщение на русском языке

**Пример:**
```typescript
import { getErrorMessage } from '@/lib/fetch-utils'

try {
  await fetchJson('/api/data')
} catch (err) {
  const message = getErrorMessage(err, 'Не удалось загрузить данные')
  setError(message)
}
```

### `isTimeoutError(error)`

Проверяет, является ли ошибка таймаутом.

**Параметры:**
- `error: unknown` - Ошибка для проверки

**Возвращает:** `boolean`

**Пример:**
```typescript
import { isTimeoutError } from '@/lib/fetch-utils'

try {
  await fetchJson('/api/data')
} catch (err) {
  if (isTimeoutError(err)) {
    console.log('Запрос превысил время ожидания')
  }
}
```

### `isNetworkError(error)`

Проверяет, является ли ошибка сетевой ошибкой.

**Параметры:**
- `error: unknown` - Ошибка для проверки

**Возвращает:** `boolean`

**Пример:**
```typescript
import { isNetworkError } from '@/lib/fetch-utils'

try {
  await fetchJson('/api/data')
} catch (err) {
  if (isNetworkError(err)) {
    console.log('Ошибка подключения к серверу')
  }
}
```

## Интерфейсы

### `FetchOptions`

Расширяет стандартный `RequestInit` с дополнительными опциями:

```typescript
interface FetchOptions extends RequestInit {
  timeout?: number      // Таймаут в миллисекундах (по умолчанию: QUALITY_TIMEOUTS.STANDARD = 10000)
  retryCount?: number   // Количество повторных попыток (по умолчанию: 1)
  retryDelay?: number   // Задержка между попытками в миллисекундах (по умолчанию: 1000)
}
```

### `FetchError`

Интерфейс ошибки, выбрасываемой утилитами:

```typescript
interface FetchError {
  message: string           // Понятное сообщение об ошибке
  status?: number          // HTTP статус код (если применимо)
  isTimeout: boolean       // Является ли ошибка таймаутом
  isNetworkError: boolean  // Является ли ошибка сетевой ошибкой
  retryCount?: number      // Количество выполненных попыток
}
```

## Константы таймаутов

Используйте константы из `quality-constants.ts` для единообразных таймаутов:

```typescript
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

QUALITY_TIMEOUTS.FAST      // 7000ms  - Быстрые запросы (статистика, статус)
QUALITY_TIMEOUTS.STANDARD  // 10000ms - Стандартные запросы (списки, отчеты)
QUALITY_TIMEOUTS.LONG      // 15000ms - Долгие операции (запуск анализа, слияние)
QUALITY_TIMEOUTS.VERY_LONG // 30000ms - Очень долгие операции
```

## Примеры использования

### Базовый GET запрос

```typescript
const data = await fetchJson<UserData>('/api/users/123', {
  timeout: QUALITY_TIMEOUTS.STANDARD,
  cache: 'no-store',
})
```

### POST запрос с телом

```typescript
await fetchJson('/api/users', {
  method: 'POST',
  timeout: QUALITY_TIMEOUTS.STANDARD,
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    name: 'John Doe',
    email: 'john@example.com'
  })
})
```

### Обработка ошибок в React компоненте

```typescript
const [data, setData] = useState<Data | null>(null)
const [error, setError] = useState<string | null>(null)
const [loading, setLoading] = useState(false)

const fetchData = useCallback(async () => {
  setLoading(true)
  setError(null)

  try {
    const result = await fetchJson<Data>('/api/data', {
      timeout: QUALITY_TIMEOUTS.STANDARD,
      cache: 'no-store',
    })
    setData(result)
  } catch (err) {
    const errorMessage = getErrorMessage(err, 'Не удалось загрузить данные')
    setError(errorMessage)
  } finally {
    setLoading(false)
  }
}, [])
```

### Игнорирование некритичных ошибок

```typescript
const fetchStatus = useCallback(async () => {
  try {
    const data = await fetchJson<Status>('/api/status', {
      timeout: QUALITY_TIMEOUTS.FAST,
      cache: 'no-store',
    })
    setStatus(data)
  } catch (err) {
    // Игнорируем ошибки, статус не критичен
    if (isTimeoutError(err) || isNetworkError(err)) {
      console.log('Статус недоступен, продолжаем работу')
      return
    }
    // Обрабатываем другие ошибки
    console.error('Ошибка получения статуса:', err)
  }
}, [])
```

## Автоматические повторные попытки

Утилиты автоматически повторяют запросы при сетевых ошибках:

- **Сетевые ошибки** (ECONNREFUSED, NetworkError, Failed to fetch) - повторяются автоматически
- **Таймауты** - не повторяются
- **HTTP ошибки** (4xx, 5xx) - не повторяются

Количество попыток и задержка настраиваются через опции:

```typescript
await fetchJson('/api/data', {
  timeout: 10000,
  retryCount: 2,    // Максимум 2 повторные попытки (всего 3 запроса)
  retryDelay: 500,  // Задержка 500ms между попытками
})
```

## Миграция с обычного fetch

### До (обычный fetch)

```typescript
try {
  const controller = new AbortController()
  const timeoutId = setTimeout(() => controller.abort(), 10000)

  try {
    const response = await fetch('/api/data', {
      signal: controller.signal,
      cache: 'no-store',
    })
    clearTimeout(timeoutId)

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed' }))
      throw new Error(errorData.error || 'Failed to fetch')
    }

    const data = await response.json()
    setData(data)
  } catch (fetchError) {
    clearTimeout(timeoutId)
    if (fetchError instanceof Error) {
      if (fetchError.name === 'AbortError') {
        setError('Превышено время ожидания')
      } else if (fetchError.message.includes('fetch failed')) {
        setError('Не удалось подключиться к серверу')
      } else {
        setError(fetchError.message)
      }
    }
  }
} catch (err) {
  setError('Ошибка подключения')
}
```

### После (с fetch-utils)

```typescript
try {
  const data = await fetchJson<Data>('/api/data', {
    timeout: QUALITY_TIMEOUTS.STANDARD,
    cache: 'no-store',
  })
  setData(data)
} catch (err) {
  const errorMessage = getErrorMessage(err, 'Не удалось загрузить данные')
  setError(errorMessage)
}
```

## Лучшие практики

1. **Всегда используйте типизацию** - указывайте тип данных в `fetchJson<T>`
2. **Используйте константы таймаутов** - не хардкодьте значения
3. **Обрабатывайте ошибки** - используйте `getErrorMessage` для пользовательских сообщений
4. **Игнорируйте некритичные ошибки** - используйте `isTimeoutError` и `isNetworkError` для фильтрации
5. **Используйте `cache: 'no-store'`** - для данных, которые должны быть актуальными

## Тестирование

Утилиты покрыты unit-тестами в `lib/__tests__/fetch-utils.test.ts`.

Запуск тестов:
```bash
npm run test lib/__tests__/fetch-utils.test.ts
```

