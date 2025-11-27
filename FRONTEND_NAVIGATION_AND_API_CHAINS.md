# Навигация и цепочки вызовов на фронтенде

## Общая архитектура

Фронтенд построен на **Next.js 16** с использованием **App Router**. Архитектура состоит из следующих слоев:

1. **UI компоненты** (React компоненты)
2. **Next.js API Routes** (`/app/api/*`) - прокси-слой
3. **Backend Go сервер** (порт 9999)

## Структура навигации

### Главный Layout

```1:74:frontend/app/layout.tsx
import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { Header } from "@/components/layout/header";
import { Footer } from "@/components/layout/footer";
import { Toaster } from "sonner";
import { ConsoleInterceptorProvider } from "@/components/console-interceptor-provider";
import { ErrorProviderWrapper } from "@/components/providers/error-provider-wrapper";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

import { generateMetadata as genMeta, seoConfigs } from "@/lib/seo"

export const metadata: Metadata = genMeta(seoConfigs.home)

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const structuredData = seoConfigs.home.structuredData || {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "Нормализатор",
    description: "Автоматизированная система для нормализации и унификации справочных данных",
    applicationCategory: "BusinessApplication",
    operatingSystem: "Web",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "RUB",
    },
  };

  return (
    <html lang="ru" suppressHydrationWarning>
      <head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <ErrorProviderWrapper>
            <ConsoleInterceptorProvider />
            <div className="relative flex min-h-screen flex-col">
              <Header />
              <main className="flex-1">{children}</main>
              <Footer />
            </div>
            <Toaster position="bottom-right" richColors closeButton />
          </ErrorProviderWrapper>
        </ThemeProvider>
      </body>
    </html>
  );
}
```

### Header компонент - структура навигации

Навигация определена в `frontend/components/layout/header.tsx`:

#### Основные разделы навигации:

1. **Главная** (`/`) - Панель управления и общая статистика
2. **Клиенты** (`/clients`) - Управление клиентами и проектами
3. **Процессы** (выпадающее меню):
   - Номенклатура (`/processes/nomenclature`)
   - Контрагенты (`/processes/counterparties`)
   - Бенчмарк нормализации (`/normalization/benchmark`)
4. **Качество** (`/quality`) - Анализ качества данных и дубликаты
5. **Результаты** (`/results`) - Просмотр результатов нормализации

#### Группа "Данные" (выпадающее меню):
- Базы данных (`/databases`)
- Управление БД (`/databases/manage`)
- Ожидающие БД (`/databases/pending`)
- Классификаторы (`/classifiers`)
- КПВЭД (`/classifiers/kpved`)
- ОКПД2 (`/classifiers/okpd2`)
- Страны (`/countries`)

#### Группа "Система" (выпадающее меню):
- Мониторинг (`/monitoring`)
- Этапы обработки (`/pipeline-stages`)
- Воркеры (`/workers`)
- Бенчмарк моделей (`/models/benchmark`)
- Отчеты (`/reports`)
- Качество данных (`/data-quality`)

## Цепочки вызовов API

### Архитектура вызовов

```
[React Component] 
    ↓
[useApiClient hook / apiClientJson]
    ↓
[Next.js API Route] (/app/api/*/route.ts)
    ↓
[Backend Go Server] (http://127.0.0.1:9999)
```

### API Client

Централизованный API клиент находится в `frontend/lib/api-client.ts`:

**Основные функции:**
- `apiClient(url, options)` - базовый клиент с обработкой ошибок
- `apiGet<T>(url, options)` - GET запрос с парсингом JSON
- `apiPost<T>(url, data, options)` - POST запрос
- `apiPut<T>(url, data, options)` - PUT запрос
- `apiDelete<T>(url, options)` - DELETE запрос
- `apiClientJson<T>(url, options)` - устаревшая функция для обратной совместимости

**Особенности:**
- Таймаут 7 секунд на все запросы
- Автоматическая обработка ошибок через ErrorContext
- Поддержка retry для сетевых ошибок и 5xx
- Различение Next.js API routes (`/api/*`) и прямых запросов к backend

**Конфигурация:**
```1:21:frontend/lib/api-config.ts
/**
 * Утилита для получения конфигурации API
 * Унифицирует получение BACKEND_URL из переменных окружения
 */

export function getBackendUrl(): string {
  // На клиенте доступны только NEXT_PUBLIC_* переменные
  if (typeof window !== 'undefined') {
    return (
      process.env.NEXT_PUBLIC_BACKEND_URL ||
      'http://127.0.0.1:9999'
    )
  }
  // На сервере доступны все переменные
  // Используем 127.0.0.1 вместо localhost для избежания проблем с DNS resolution
  return (
    process.env.BACKEND_URL ||
    process.env.NEXT_PUBLIC_BACKEND_URL ||
    'http://127.0.0.1:9999'
  )
}
```

### Next.js API Routes (прокси-слой)

Все API routes находятся в `frontend/app/api/*/route.ts` и проксируют запросы к backend Go серверу.

**Пример структуры:**
```1:74:frontend/app/api/clients/route.ts
import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET() {
  try {
    // Получаем URL backend при каждом запросе
    const API_BASE_URL = getBackendUrl()
    
    // Создаем контроллер для таймаута (7 секунд)
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 7000)

    try {
      const response = await fetch(`${API_BASE_URL}/api/clients`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Если backend недоступен или возвращает 404, возвращаем пустой список
        if (response.status === 404) {
          console.warn('Backend endpoint /api/clients not found. Backend may need to be restarted.')
          return NextResponse.json([])
        }
        const errorText = await response.text().catch(() => 'Unknown error')
        console.error(`Backend error (${response.status}):`, errorText)
        // Возвращаем пустой список вместо ошибки для 5xx ошибок
        if (response.status >= 500) {
          return NextResponse.json([])
        }
        throw new Error(`HTTP error! status: ${response.status}`)
      }

      const data = await response.json()
      return NextResponse.json(data)
    } catch (fetchError) {
      clearTimeout(timeoutId)
      
      // Обработка сетевых ошибок
      if (fetchError instanceof Error) {
        if (fetchError.name === 'AbortError') {
          console.error('Request timeout while fetching clients')
          return NextResponse.json([])
        }
        if (fetchError.message.includes('fetch failed') || 
            fetchError.message.includes('Failed to fetch') ||
            fetchError.message.includes('NetworkError') ||
            fetchError.message.includes('ECONNREFUSED')) {
          console.error('Backend connection failed:', fetchError.message)
          return NextResponse.json([])
        }
      }
      throw fetchError
    }
  } catch (error) {
    console.error('Error fetching clients:', error)
    // Возвращаем пустой список вместо ошибки, чтобы UI не ломался
    // Логируем детали ошибки для отладки
    if (error instanceof Error) {
      console.error('Error details:', {
        message: error.message,
        name: error.name,
        stack: error.stack
      })
    }
    return NextResponse.json([], { status: 200 })
  }
}
```

## Примеры цепочек вызовов

### 1. Главная страница (Dashboard)

**Маршрут:** `/`  
**Файл:** `frontend/app/page.tsx`

**Цепочка вызовов при загрузке:**

1. Компонент монтируется → `useEffect` вызывает `loadDashboardData()`
2. Параллельные запросы:
   ```typescript
   const [statsData, statusData, qualityData, metricsData] = await Promise.allSettled([
     apiClientJson<Partial<DashboardStats>>('/api/dashboard/stats', { skipErrorHandler: true }),
     apiClientJson('/api/normalization/status', { skipErrorHandler: true }),
     apiClientJson('/api/quality/metrics', { skipErrorHandler: true }),
     apiClientJson('/api/monitoring/metrics', { skipErrorHandler: true })
   ])
   ```

3. **API Route:** `/app/api/dashboard/stats/route.ts`
   - Проксирует запрос к `http://127.0.0.1:9999/api/dashboard/stats`
   - Таймаут 7 секунд
   - Обработка ошибок

4. **Backend:** Go сервер обрабатывает запрос и возвращает статистику

**Автообновление:** Интервал 30 секунд для обновления данных

**Действия пользователя:**
- Запуск нормализации → `POST /api/normalization/start`
- Скачивание XML → `GET /api/1c/processing/xml`

### 2. Страница клиентов

**Маршрут:** `/clients`  
**Файл:** `frontend/app/clients/page.tsx`

**Цепочка вызовов:**

1. Компонент монтируется → `useEffect` вызывает `fetchClients()`
2. **Запрос:**
   ```typescript
   const data = await get<Client[]>('/api/clients', { skipErrorHandler: true })
   ```

3. **API Route:** `/app/api/clients/route.ts`
   - `GET` метод проксирует к `http://127.0.0.1:9999/api/clients`
   - При ошибке возвращает пустой массив `[]`

4. **Backend:** Возвращает список клиентов

**Фильтрация:** Локальная фильтрация на клиенте по поисковому запросу и стране

**Навигация:**
- Клик по карточке клиента → переход на `/clients/[clientId]`
- Кнопка "Проекты" → переход на `/clients/[clientId]/projects`

### 3. Детальная страница клиента

**Маршрут:** `/clients/[clientId]`  
**Файл:** `frontend/app/clients/[clientId]/page.tsx`

**Цепочка вызовов:**

1. Компонент получает `clientId` из `useParams()`
2. **Запрос:**
   ```typescript
   const response = await fetch(`/api/clients/${id}`)
   const data = await response.json()
   ```

3. **API Route:** `/app/api/clients/[clientId]/route.ts`
   - Проксирует к `http://127.0.0.1:9999/api/clients/{id}`

4. **Backend:** Возвращает детальную информацию о клиенте, проекты, статистику

**Табы:**
- **Обзор** - показывает общую информацию
- **Номенклатура** - компонент `NomenclatureTab`:
  - Запрос: `/api/clients/[clientId]/nomenclature`
  - Или: `/api/clients/[clientId]/projects/[projectId]/nomenclature`
- **Контрагенты** - компонент `CounterpartiesTab`:
  - Запрос: `/api/clients/[clientId]/counterparties`
- **Базы данных** - компонент `DatabasesTab`:
  - Запрос: `/api/clients/[clientId]/databases`
- **Статистика** - компонент `StatisticsTab`:
  - Запрос: `/api/clients/[clientId]/statistics`

### 4. Таб номенклатуры

**Компонент:** `frontend/app/clients/[clientId]/components/nomenclature-tab.tsx`

**Цепочка вызовов:**

1. При изменении `selectedProjectId`, `debouncedSearchQuery`, `currentPage` → `useEffect` вызывает `fetchNomenclature()`
2. **Запрос:**
   ```typescript
   let url = `/api/clients/${clientId}/nomenclature?page=${currentPage}&limit=${itemsPerPage}`
   if (selectedProjectId) {
     url = `/api/clients/${clientId}/projects/${selectedProjectId}/nomenclature?page=${currentPage}&limit=${itemsPerPage}`
   }
   if (debouncedSearchQuery) {
     url += `&search=${encodeURIComponent(debouncedSearchQuery)}`
   }
   const response = await fetch(url)
   ```

3. **API Route:** `/app/api/clients/[clientId]/nomenclature/route.ts`
   - Проксирует к backend с параметрами пагинации и поиска

4. **Backend:** Возвращает список номенклатуры с пагинацией

**Fallback:** Если endpoint возвращает 404, пробует `/api/normalized/uploads`

**Дополнительные действия:**
- Открытие деталей → `NomenclatureDetailDialog`
- Экспорт данных → локальная обработка
- Сортировка → локальная обработка

### 5. Процессы нормализации

**Маршрут:** `/processes/nomenclature` или `/processes/counterparties`

**Цепочка вызовов:**

1. Загрузка статуса процесса:
   - `GET /api/normalization/status` (для номенклатуры)
   - `GET /api/counterparties/normalization/status` (для контрагентов)

2. Запуск процесса:
   - `POST /api/normalization/start`
   - `POST /api/counterparties/normalization/start`

3. Остановка процесса:
   - `POST /api/normalization/stop`
   - `POST /api/counterparties/normalization/stop`

4. Получение статистики:
   - `GET /api/normalization/stats`
   - `GET /api/counterparties/normalized/stats`

### 6. Качество данных

**Маршрут:** `/quality`

**Подразделы:**
- `/quality/duplicates` - дубликаты
- `/quality/violations` - нарушения
- `/quality/suggestions` - предложения

**Цепочка вызовов:**

1. Загрузка дубликатов:
   - `GET /api/quality/duplicates?page=1&limit=20`

2. Объединение дубликатов:
   - `POST /api/quality/duplicates/[groupId]/merge`

3. Загрузка нарушений:
   - `GET /api/quality/violations?page=1&limit=20`

4. Загрузка предложений:
   - `GET /api/quality/suggestions?page=1&limit=20`

5. Применение предложения:
   - `POST /api/quality/suggestions/[suggestionId]/apply`

### 7. Мониторинг

**Маршрут:** `/monitoring`

**Цепочка вызовов:**

1. Метрики в реальном времени:
   - `GET /api/monitoring/metrics` - общие метрики
   - `GET /api/monitoring/providers` - метрики провайдеров
   - `GET /api/monitoring/events` - события

2. История:
   - `GET /api/monitoring/history?from=...&to=...`

**Автообновление:** Polling каждые несколько секунд для real-time метрик

## Обработка ошибок

### ErrorContext

Все ошибки обрабатываются через `ErrorContext` (`frontend/contexts/ErrorContext.tsx`):

1. API клиент перехватывает ошибки
2. Если не `skipErrorHandler`, ошибка передается в `ErrorContext`
3. `ErrorContext` показывает уведомление пользователю
4. Логирование ошибки в консоль

### Стратегии обработки ошибок в API Routes

1. **Возврат пустого массива** (для списков):
   - Если backend недоступен → `[]`
   - Если 404 → `[]`
   - Если 5xx → `[]`

2. **Возврат ошибки** (для критичных операций):
   - Возврат `{ error: "message" }` с соответствующим статусом

3. **Таймауты:**
   - Все запросы имеют таймаут 7 секунд
   - При таймауте возвращается соответствующая ошибка

## Роутинг Next.js App Router

### Структура маршрутов

```
app/
├── layout.tsx              # Корневой layout
├── page.tsx                 # Главная страница (/)
├── clients/
│   ├── layout.tsx           # Layout для раздела клиентов
│   ├── page.tsx             # Список клиентов (/clients)
│   ├── new/
│   │   └── page.tsx         # Создание клиента (/clients/new)
│   └── [clientId]/
│       ├── page.tsx         # Детали клиента (/clients/[clientId])
│       ├── edit/
│       │   └── page.tsx     # Редактирование (/clients/[clientId]/edit)
│       └── projects/
│           ├── page.tsx     # Список проектов
│           ├── new/
│           │   └── page.tsx # Создание проекта
│           └── [projectId]/
│               └── page.tsx # Детали проекта
├── processes/
│   ├── layout.tsx
│   ├── nomenclature/
│   │   └── page.tsx         # Процессы номенклатуры
│   └── counterparties/
│       └── page.tsx         # Процессы контрагентов
├── quality/
│   ├── layout.tsx
│   ├── page.tsx             # Обзор качества
│   ├── duplicates/
│   │   ├── layout.tsx
│   │   └── page.tsx         # Дубликаты
│   ├── violations/
│   │   └── page.tsx         # Нарушения
│   └── suggestions/
│       └── page.tsx         # Предложения
└── api/                     # Next.js API Routes
    ├── clients/
    │   ├── route.ts         # GET, POST /api/clients
    │   └── [clientId]/
    │       └── route.ts     # GET, PUT, DELETE /api/clients/[id]
    ├── dashboard/
    │   └── stats/
    │       └── route.ts     # GET /api/dashboard/stats
    └── ...
```

### Динамические маршруты

- `[clientId]` - динамический параметр клиента
- `[projectId]` - динамический параметр проекта
- `[dbId]` - динамический параметр базы данных
- `[groupId]` - динамический параметр группы

### Layout файлы

Layout файлы применяются ко всем дочерним маршрутам:
- `app/layout.tsx` - применяется ко всему приложению
- `app/clients/layout.tsx` - применяется ко всем маршрутам `/clients/*`
- `app/quality/layout.tsx` - применяется ко всем маршрутам `/quality/*`

## Хуки и утилиты

### useApiClient

Хук для упрощения работы с API (`frontend/hooks/useApiClient.ts`):

```typescript
const { get, post, put, delete: del } = useApiClient()

// Автоматическая обработка ошибок через ErrorContext
const data = await get<Client[]>('/api/clients')
```

### useError

Хук для доступа к ErrorContext:

```typescript
const { handleError } = useError()
```

## Особенности реализации

### 1. Debounce для поиска

Многие компоненты используют debounce для поисковых запросов (300-500мс задержка) для уменьшения количества запросов.

### 2. Локальная фильтрация

Некоторые страницы (например, список клиентов) используют локальную фильтрацию вместо серверной для лучшей производительности.

### 3. Пагинация

Большинство списков используют пагинацию:
- Параметры: `?page=1&limit=20`
- Локальное состояние для текущей страницы

### 4. Автообновление

Некоторые страницы (Dashboard, Monitoring) используют интервалы для автоматического обновления данных.

### 5. Skeleton loading

Используются skeleton компоненты для улучшения UX во время загрузки:
- `DashboardSkeleton`
- `ClientsPageSkeleton`
- `StatCardSkeleton`

## Заключение

Фронтенд использует следующую архитектуру:

1. **Навигация** - централизована в Header компоненте с выпадающими меню
2. **API вызовы** - через централизованный API клиент с обработкой ошибок
3. **Прокси-слой** - Next.js API Routes проксируют запросы к backend
4. **Backend** - Go сервер на порту 9999 обрабатывает бизнес-логику

Все запросы имеют таймаут 7 секунд, автоматическую обработку ошибок и поддержку retry для сетевых ошибок.

