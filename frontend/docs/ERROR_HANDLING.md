# Руководство по обработке ошибок

## Обзор

В проекте реализована централизованная система обработки ошибок, аналогичная бэкенду. Все ошибки обрабатываются единообразно, с понятными сообщениями для пользователей и структурированным логированием для разработчиков.

## Основные компоненты

### 1. ErrorContext

Глобальный контекст для обработки ошибок во всем приложении.

```tsx
import { useError } from '@/contexts/ErrorContext'

function MyComponent() {
  const { handleError } = useError()
  
  const fetchData = async () => {
    try {
      // ваш код
    } catch (error) {
      handleError(error, 'Не удалось загрузить данные')
    }
  }
}
```

### 2. useApiClient

Хук для выполнения API-запросов с автоматической обработкой ошибок.

```tsx
import { useApiClient } from '@/hooks/useApiClient'

function MyComponent() {
  const { get, post, put, delete: del } = useApiClient()
  
  // GET запрос
  const loadData = async () => {
    try {
      const data = await get<MyDataType>('/api/data')
      // данные автоматически обработаны
    } catch (error) {
      // ошибка уже показана пользователю через toast
      // можно добавить дополнительную логику
    }
  }
  
  // POST запрос
  const saveData = async (data: MyDataType) => {
    try {
      const result = await post('/api/data', data)
      toast.success('Данные сохранены')
    } catch (error) {
      // ошибка уже обработана
    }
  }
}
```

### 3. AppError

Структурированный класс ошибок с разделением на пользовательские и технические сообщения.

```tsx
import { AppError, createNetworkError } from '@/lib/errors'

// Создание ошибки
const error = createNetworkError(
  'Не удалось подключиться к серверу', // для пользователя
  0, // статус код
  'ECONNREFUSED: connection refused' // технические детали
)

// Использование
throw error
```

### 4. ErrorBoundary

Компонент для обработки ошибок рендеринга React.

```tsx
import { ErrorBoundary } from '@/components/ErrorBoundary'

<ErrorBoundary>
  <YourComponent />
</ErrorBoundary>
```

## Миграция со старого кода

### До (старый подход)

```tsx
const fetchData = async () => {
  try {
    const response = await fetch('/api/data')
    if (!response.ok) {
      const errorData = await response.json()
      toast.error(errorData.error || 'Ошибка')
      return
    }
    const data = await response.json()
    setData(data)
  } catch (error) {
    console.error(error)
    toast.error('Ошибка загрузки данных')
  }
}
```

### После (новый подход)

```tsx
const { get } = useApiClient()

const fetchData = async () => {
  try {
    const data = await get<DataType>('/api/data')
    setData(data)
  } catch (error) {
    // Ошибка уже обработана через ErrorContext
    // Toast уже показан пользователю
  }
}
```

## Продвинутые сценарии

### Пропуск автоматической обработки ошибок

Если нужно обработать ошибку вручную:

```tsx
const { get } = useApiClient()

try {
  const data = await get('/api/data', { skipErrorHandler: true })
} catch (error) {
  // Обрабатываем ошибку вручную
  if (error instanceof AppError && error.statusCode === 404) {
    // Специальная обработка для 404
  } else {
    handleError(error)
  }
}
```

### Кастомный обработчик ошибок

```tsx
const { get } = useApiClient()

try {
  const data = await get('/api/data', {
    onError: (error) => {
      // Кастомная логика перед стандартной обработкой
      if (error.statusCode === 401) {
        router.push('/login')
        return
      }
      // Стандартная обработка
      handleError(error)
    }
  })
} catch (error) {
  // Ошибка уже обработана
}
```

### Retry механизм

Для автоматического повтора запросов при сетевых ошибках и 5xx ошибках:

```tsx
const { get } = useApiClient()

try {
  // Повторит запрос 3 раза с задержкой 1 секунда между попытками
  const data = await get('/api/data', {
    retries: 3,
    retryDelay: 1000, // миллисекунды
  })
} catch (error) {
  // Ошибка после всех попыток
}
```

Retry работает только для:
- Сетевых ошибок (statusCode === 0)
- 5xx ошибок сервера
- 408 Timeout ошибок

4xx ошибки (клиентские) не повторяются автоматически.

### Обработка ошибок в формах

Для ошибок валидации форм используйте inline-отображение:

```tsx
const [formErrors, setFormErrors] = useState<Record<string, string>>({})

const submitForm = async (data: FormData) => {
  try {
    await post('/api/submit', data)
  } catch (error) {
    if (error instanceof AppError && error.statusCode === 400) {
      // Ошибки валидации показываем рядом с полями
      const validationErrors = JSON.parse(error.technicalDetails || '{}')
      setFormErrors(validationErrors)
    } else {
      // Другие ошибки обрабатываем стандартно
      handleError(error)
    }
  }
}
```

## Глобальные обработчики

Глобальные обработчики автоматически установлены в `GlobalErrorHandlers`:

- Необработанные промисы (`unhandledrejection`)
- Глобальные JS ошибки (`error`)

Все они автоматически показывают toast пользователю и логируют ошибку в консоль.

## Логирование

Все ошибки логируются структурированно через `logError`:

```tsx
import { logError } from '@/lib/errors'

logError(error, {
  context: 'MyComponent',
  action: 'fetchData',
  userId: currentUser?.id,
})
```

В будущем можно легко добавить интеграцию с Sentry:

```tsx
// В lib/errors.ts уже есть заготовка:
// if (typeof window !== 'undefined' && window.Sentry) {
//   window.Sentry.captureException(error, { contexts: { custom: context } })
// }
```

## Best Practices

1. **Всегда используйте `useApiClient`** для API-запросов
2. **Не используйте `toast.error` напрямую** - используйте `handleError` из контекста
3. **Не используйте `console.error` для пользовательских ошибок** - используйте `logError`
4. **Разделяйте пользовательские и технические сообщения** - используйте `AppError`
5. **Обрабатывайте ошибки валидации отдельно** - показывайте их рядом с полями формы
6. **Используйте `skipErrorHandler`** только когда действительно нужна кастомная обработка

## Примеры использования

См. отрефакторенные компоненты:
- `frontend/app/workers/page.tsx`
- `frontend/app/clients/[clientId]/projects/[projectId]/normalization/page.tsx`
- `frontend/app/data-quality/page.tsx`

