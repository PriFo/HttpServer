# 📚 Руководство по тестированию

## Обзор тестовой инфраструктуры

Проект использует Playwright для E2E тестирования с комплексной инфраструктурой утилит и вспомогательных функций.

## Структура тестов

```
tests/
├── e2e/                          # E2E тесты
│   ├── full-project-e2e.spec.ts  # Главный комплексный тест
│   ├── user-roles.spec.ts        # Тесты ролей пользователей
│   ├── quality-management.spec.ts # Тесты управления качеством
│   ├── data-management.spec.ts   # Тесты управления данными
│   ├── normalization.spec.ts     # Тесты нормализации
│   ├── monitoring.spec.ts        # Тесты мониторинга
│   ├── reports.spec.ts            # Тесты отчетов
│   ├── test-helpers.ts           # Вспомогательные функции
│   └── README.md                 # Документация
├── fixtures/                      # Тестовые данные
│   └── test-data.ts              # Fixtures и тестовые данные
├── data_integrity/               # Тесты целостности данных
├── performance/                  # Тесты производительности
└── resilience/                   # Тесты отказоустойчивости

utils/
├── api-testing.ts                # Утилиты для работы с API
├── auth-testing.ts               # Утилиты для аутентификации
└── README.md                     # Документация

scripts/
├── run-e2e-tests.sh             # Скрипт запуска (Linux/Mac)
└── run-e2e-tests.ps1            # Скрипт запуска (Windows)
```

## Быстрый старт

### 1. Установка зависимостей

```bash
cd frontend
npm install
npx playwright install
```

### 2. Запуск сервисов

```bash
# Бэкенд (в отдельном терминале)
docker-compose up -d backend
# или
go run main.go

# Фронтенд (в отдельном терминале)
cd frontend
npm run dev
```

### 3. Запуск тестов

```bash
# Все тесты
npx playwright test

# Конкретный тест
npx playwright test tests/e2e/full-project-e2e.spec.ts

# С UI
npx playwright test --ui

# В видимом режиме
npx playwright test --headed

# Через скрипт (Linux/Mac)
./scripts/run-e2e-tests.sh

# Через скрипт (Windows)
.\scripts\run-e2e-tests.ps1
```

## Использование утилит

### API Testing (`utils/api-testing.ts`)

```typescript
import {
  createTestClient,
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData,
} from '../../utils/api-testing'

// Создание тестовых данных
const client = await createTestClient({ name: 'Test Client' })
const project = await createTestProject(client.id, { name: 'Test Project' })

// Загрузка базы данных
const dbPath = findTestDatabase()
if (dbPath) {
  const database = await uploadDatabaseFile(client.id, project.id, dbPath)
}

// Очистка
await cleanupTestData(client.id, project.id, database.id)
```

### Auth Testing (`utils/auth-testing.ts`)

```typescript
import { createAdminToken, createManagerToken } from '../../utils/auth-testing'

// Создание токенов
const adminToken = createAdminToken()
const managerToken = createManagerToken(123) // clientId

// Использование в тестах
await page.context().setExtraHTTPHeaders({
  Authorization: `Bearer ${adminToken}`,
})
```

### Test Helpers (`tests/e2e/test-helpers.ts`)

```typescript
import { waitForText, clickIfVisible, checkToast } from './test-helpers'

// Ожидание текста
await waitForText(page, 'Завершено')

// Клик, если видно
await clickIfVisible(page, [
  'button:has-text("Начать")',
  'button:has-text("Start")',
])

// Проверка toast
await checkToast(page, /успешно/i, 'success')
```

### Test Fixtures (`tests/fixtures/test-data.ts`)

```typescript
import { getTestClient, getTestProject, findTestDatabase } from '../fixtures/test-data'

// Получение стандартных тестовых данных
const client = getTestClient({ name: 'Custom Name' })
const project = getTestProject({ project_type: 'custom' })

// Поиск тестовой БД
const dbPath = findTestDatabase()
```

## Лучшие практики

### 1. Использование beforeAll/afterAll

Всегда очищайте тестовые данные после тестов:

```typescript
test.beforeAll(async () => {
  // Создание тестовых данных
})

test.afterAll(async () => {
  // Очистка тестовых данных
  await cleanupTestData(clientId, projectId, databaseId)
})
```

### 2. Гибкие селекторы

Используйте множественные селекторы с fallback:

```typescript
const button = page.locator('button:has-text("Начать")').or(
  page.locator('button:has-text("Start")')
).first()
```

### 3. Обработка ошибок

Не падайте на необязательных проверках:

```typescript
const hasElement = await element.isVisible({ timeout: 5000 }).catch(() => false)
if (hasElement) {
  // Действие
} else {
  console.warn('⚠️ Элемент не найден, но продолжаем')
}
```

### 4. Логирование

Используйте подробное логирование:

```typescript
console.log('✅ Шаг выполнен успешно')
console.warn('⚠️ Предупреждение')
console.error('❌ Ошибка')
```

### 5. Проверка на бэкенде

Проверяйте статус не только в UI, но и через API:

```typescript
const status = await getNormalizationStatus(clientId, projectId)
if (status && status.status === 'running') {
  console.log('✅ Нормализация запущена на бэкенде')
}
```

## Отладка тестов

### Режим отладки

```bash
npx playwright test --debug
```

### UI режим

```bash
npx playwright test --ui
```

### Trace viewer

```bash
npx playwright show-trace trace.zip
```

### Скриншоты

Скриншоты автоматически создаются при ошибках в `test-results/`

## CI/CD

Тесты автоматически запускаются в GitHub Actions при push в main/develop.

См. `.github/workflows/e2e-tests.yml` для конфигурации.

## Troubleshooting

### Тест не находит элементы

1. Увеличьте таймауты
2. Проверьте, что страница полностью загрузилась
3. Используйте `page.waitForLoadState('networkidle')`

### Ошибки при создании данных

1. Проверьте, что бэкенд запущен
2. Проверьте переменные окружения
3. Проверьте логи бэкенда

### База данных не загружается

1. Убедитесь, что файл существует
2. Проверьте права доступа
3. Используйте `findTestDatabase()` для поиска

## Дополнительные ресурсы

- [Playwright Documentation](https://playwright.dev/)
- [API Documentation](../../API_DOCUMENTATION.md)
- [Frontend Navigation Guide](../../FRONTEND_NAVIGATION_AND_API_CHAINS.md)

