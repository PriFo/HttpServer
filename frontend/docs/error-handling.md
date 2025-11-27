# Система обработки ошибок и логирования

## Обзор

Проект использует централизованную систему обработки ошибок и логирования, которая обеспечивает:
- Единообразную обработку ошибок во всем приложении
- Структурированное логирование с уровнями
- Автоматическое логирование ошибок API
- Error Boundaries для React компонентов
- Понятные сообщения для пользователей

## Компоненты системы

### 1. Logger (`lib/logger.ts`)

Централизованная система логирования с уровнями и буферизацией.

```typescript
import { logger, logError, logInfo } from '@/lib/logger'

// Простое логирование
logger.info('Пользователь выполнил действие', { userId: '123' })
logger.error('Ошибка загрузки данных', { url: '/api/data' }, error)

// Или используйте удобные функции
logInfo('Информационное сообщение')
logError('Ошибка', { context: 'value' }, error)

// Логирование API запросов
logger.logApiError('/api/users', 'GET', 500, error, { userId: '123' })
logger.logApiSuccess('/api/users', 'GET', 150, { userId: '123' })
```

**Уровни логирования:**
- `DEBUG` - отладочная информация (только в development)
- `INFO` - информационные сообщения
- `WARN` - предупреждения
- `ERROR` - ошибки
- `FATAL` - критические ошибки

### 2. Error Handler (`lib/error-handler.ts`)

Централизованная обработка ошибок с автоматическим логированием и уведомлениями.

```typescript
import { handleError, withErrorHandling, createErrorHandler } from '@/lib/error-handler'

// Обработка ошибки
const userMessage = handleError(error, {
  logError: true,
  showToast: true,
  context: { userId: '123' },
  fallbackMessage: 'Произошла ошибка'
})

// Обертка для async функций
const safeFunction = withErrorHandling(
  async (userId: string) => {
    // Ваш код
  },
  { logError: true, showToast: true }
)

// Обработчик для React компонентов
const errorHandler = createErrorHandler('MyComponent', {
  logError: true,
  showToast: false
})
```

### 3. Error Boundary (`components/common/error-boundary.tsx`)

React компонент для перехвата ошибок в дочерних компонентах.

```typescript
import { ErrorBoundary } from '@/components/common/error-boundary'

function App() {
  return (
    <ErrorBoundary
      onError={(error, errorInfo) => {
        // Кастомная обработка
      }}
      resetKeys={[userId]} // Сброс при изменении userId
    >
      <YourComponent />
    </ErrorBoundary>
  )
}
```

### 4. AppError (`lib/errors.ts`)

Класс для типизированных ошибок приложения.

```typescript
import { AppError, createNetworkError, createValidationError } from '@/lib/errors'

// Создание ошибки
throw new AppError(
  'Сообщение для пользователя',
  'Технические детали',
  400,
  'VALIDATION_ERROR'
)

// Или используйте фабрики
throw createNetworkError('Не удалось подключиться', 503)
throw createValidationError('Неверные данные')
```

## Использование в компонентах

### React компоненты

```typescript
'use client'

import { useEffect } from 'react'
import { handleError } from '@/lib/error-handler'
import { logger } from '@/lib/logger'

export function MyComponent() {
  useEffect(() => {
    async function loadData() {
      try {
        const response = await fetch('/api/data')
        if (!response.ok) throw new Error('Failed to load')
        // ...
      } catch (error) {
        handleError(error, {
          context: { component: 'MyComponent' },
          showToast: true
        })
      }
    }
    loadData()
  }, [])

  return <div>...</div>
}
```

### API роуты

```typescript
import { NextRequest, NextResponse } from 'next/server'
import { withErrorHandler, createErrorResponse } from '@/lib/errors'
import { AppError } from '@/lib/errors'

export const GET = withErrorHandler(async (request: NextRequest) => {
  try {
    // Ваш код
    return NextResponse.json({ data: 'success' })
  } catch (error) {
    // Ошибки автоматически обрабатываются withErrorHandler
    throw error
  }
})
```

## Интеграция с мониторингом

Система готова к интеграции с системами мониторинга (Sentry, LogRocket и т.д.):

```typescript
// В lib/logger.ts раскомментируйте и настройте:
if (!this.isDevelopment && level >= LogLevel.ERROR) {
  if (typeof window !== 'undefined' && window.Sentry) {
    window.Sentry.captureMessage(entry.message, {
      level: LogLevel[entry.level].toLowerCase(),
      extra: entry.context,
    })
  }
}
```

## Тестирование

Все компоненты системы покрыты тестами:

```bash
# Запуск unit тестов
npm run test

# Запуск с покрытием
npm run test:coverage

# Запуск в watch режиме
npm run test:watch
```

## Best Practices

1. **Всегда используйте handleError для обработки ошибок**
2. **Логируйте контекст** - добавляйте полезную информацию в context
3. **Используйте Error Boundaries** для критических компонентов
4. **Не логируйте чувствительные данные** (пароли, токены)
5. **Используйте правильные уровни логирования**
6. **Тестируйте обработку ошибок** в ваших компонентах

