# Аудит обработки ошибок и логирования на фронтенде

## Статус миграции

### ✅ Полностью мигрированы (используют новую систему)

1. `frontend/app/workers/page.tsx` - использует `useApiClient`
2. `frontend/app/clients/[clientId]/projects/[projectId]/normalization/page.tsx` - использует `useApiClient`
3. `frontend/app/data-quality/page.tsx` - использует `useApiClient`
4. `frontend/app/monitoring/page.tsx` - использует `useError`

### ⚠️ Частично мигрированы (нужна доработка)

1. `frontend/app/page.tsx` (Dashboard)
   - Использует `apiClientJson` (не существует)
   - Использует `useError` для обработки
   - Нужно: заменить на `useApiClient`

### ❌ Не мигрированы (используют старый подход)

1. `frontend/app/clients/page.tsx`
   - Использует `fetch` напрямую
   - Использует `console.error`
   - Нужно: заменить на `useApiClient`

2. `frontend/app/databases/page.tsx`
   - Использует `fetch` напрямую
   - Использует `toast.error`
   - Нужно: заменить на `useApiClient`

3. `frontend/app/results/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

4. `frontend/app/databases/manage/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

5. `frontend/app/databases/pending/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

6. `frontend/app/clients/[clientId]/projects/[projectId]/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

7. `frontend/app/clients/[clientId]/projects/[projectId]/counterparties/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

8. `frontend/app/models/benchmark/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

9. `frontend/app/normalization/benchmark/page.tsx`
   - Использует `toast.error`
   - Нужно: проверить и мигрировать

## Проблемы, обнаруженные при аудите

### 1. Несуществующая функция `apiClientJson`

В `frontend/app/page.tsx` используется `apiClientJson`, которой нет в `api-client.ts`. 
Нужно заменить на `useApiClient` или добавить функцию.

### 2. Прямое использование `console.error`

Много компонентов используют `console.error` напрямую вместо `logError` из `lib/errors.ts`.
Это не критично, но лучше использовать структурированное логирование.

### 3. Прямое использование `toast.error`

Много компонентов используют `toast.error` напрямую вместо `handleError` из контекста.
Это приводит к несогласованности в обработке ошибок.

### 4. Прямое использование `fetch`

Много компонентов используют `fetch` напрямую вместо `useApiClient`.
Это означает, что они не получают:
- Автоматическую обработку ошибок
- Retry механизм
- Структурированные ошибки

## Рекомендации по миграции

### Приоритет 1 (Критично)
1. Исправить `apiClientJson` в `page.tsx`
2. Мигрировать основные страницы: `clients/page.tsx`, `databases/page.tsx`

### Приоритет 2 (Важно)
3. Мигрировать страницы с формами и действиями
4. Заменить все `toast.error` на `handleError`

### Приоритет 3 (Желательно)
5. Заменить `console.error` на `logError` где возможно
6. Добавить retry для критичных запросов

## Статистика

- Всего файлов с `console.error`: 109
- Всего файлов с `toast.error`: 10
- Всего файлов с `catch`: 325 (в 130 файлах)
- Файлов с `useApiClient`: 8
- Файлов с `useError`: 2

## План действий

1. ✅ Создана система обработки ошибок
2. ✅ Мигрированы 3 ключевых компонента
3. ⏳ Исправить `apiClientJson` в dashboard
4. ⏳ Мигрировать основные страницы списков
5. ⏳ Мигрировать страницы с формами
6. ⏳ Заменить все `toast.error` на `handleError`
7. ⏳ Опционально: заменить `console.error` на `logError`

