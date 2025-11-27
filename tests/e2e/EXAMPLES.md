# Примеры использования E2E тестов

## Быстрый старт

### 1. Подготовка окружения

```bash
# Запустить бэкенд
go run main.go

# В другом терминале запустить фронтенд
cd frontend
npm run dev
```

### 2. Запуск всех тестов

```bash
# Из корня проекта
npm run test:e2e

# Или из директории frontend
cd frontend
npm run test:e2e
```

## Примеры запуска конкретных тестов

### Управление данными

```bash
# Запустить только тесты управления данными
npx playwright test tests/e2e/data-management.spec.ts

# Запустить конкретный тест
npx playwright test tests/e2e/data-management.spec.ts -g "Полный жизненный цикл"
```

### Управление качеством

```bash
# Запустить тесты качества
npx playwright test tests/e2e/quality-management.spec.ts

# Запустить тест слияния дубликатов
npx playwright test tests/e2e/quality-management.spec.ts -g "Ручное слияние"
```

### Обработка ошибок

```bash
# Запустить тесты обработки ошибок
npx playwright test tests/e2e/error-handling.spec.ts

# Запустить тест с таймаутом
npx playwright test tests/e2e/error-handling.spec.ts -g "Таймаут"
```

### Доступность

```bash
# Запустить тесты доступности
npx playwright test tests/e2e/accessibility.spec.ts

# Запустить только проверку axe-core
npx playwright test tests/e2e/accessibility.spec.ts -g "Автоматическая проверка"
```

## Режимы запуска

### UI режим (интерактивный)

```bash
npm run test:e2e:ui
```

Позволяет:
- Видеть тесты в реальном времени
- Запускать тесты выборочно
- Просматривать трассировку выполнения

### Режим отладки

```bash
npm run test:e2e:debug
```

Позволяет:
- Пошаговое выполнение
- Инспектировать элементы страницы
- Использовать консоль браузера

### Просмотр отчета

```bash
npm run test:e2e:report
```

Открывает HTML отчет с результатами всех тестов.

## Примеры использования утилит API

### Создание тестового клиента и проекта

```typescript
import { createTestClient, createTestProject, cleanupTestData } from '../../utils/api-testing'

test('Пример использования утилит', async () => {
  // Создаем клиента
  const client = await createTestClient({
    name: 'Test Client',
    contact_email: 'test@example.com'
  })
  
  // Создаем проект
  const project = await createTestProject(client.id, {
    name: 'Test Project',
    project_type: 'normalization'
  })
  
  // Очищаем после теста
  await cleanupTestData(client.id, project.id)
})
```

### Работа с бэкапами

```typescript
import { createBackup, listBackups, restoreBackup } from '../../utils/api-testing'

test('Пример работы с бэкапами', async () => {
  // Создаем бэкап
  const backup = await createBackup({
    format: 'zip',
    includeMain: true,
    includeUploads: true
  })
  
  // Получаем список бэкапов
  const backups = await listBackups()
  console.log(`Найдено бэкапов: ${backups.length}`)
  
  // Восстанавливаем из бэкапа
  if (backups.length > 0) {
    await restoreBackup(backups[0].name)
  }
})
```

### Работа с качеством данных

```typescript
import { getQualityDuplicates, mergeDuplicates, getQualityMetrics } from '../../utils/api-testing'

test('Пример работы с качеством', async () => {
  // Получаем дубликаты
  const duplicates = await getQualityDuplicates(undefined, {
    unmerged: true,
    limit: 10
  })
  
  // Получаем метрики качества
  const metrics = await getQualityMetrics()
  
  // Объединяем дубликаты
  if (duplicates.groups && duplicates.groups.length > 0) {
    const group = duplicates.groups[0]
    await mergeDuplicates(
      group.id,
      group.master_id,
      group.duplicate_ids
    )
  }
})
```

### Работа с тестовыми базами данных

```typescript
import { 
  createTestDatabase, 
  checkDatabaseIntegrity, 
  getDatabaseStats,
  copyDatabase,
  isValidSQLiteFile 
} from '../../utils/database-testing'

test('Пример работы с тестовыми БД', async () => {
  // Создаем тестовую БД (использует шаблон, если доступен)
  await createTestDatabase('test-db.sqlite')
  
  // Проверяем целостность
  const isIntact = await checkDatabaseIntegrity('test-db.sqlite')
  expect(isIntact).toBe(true)
  
  // Получаем статистику
  const stats = await getDatabaseStats('test-db.sqlite')
  console.log(`Размер БД: ${stats.fileSize} байт`)
  
  // Копируем БД
  await copyDatabase('test-db.sqlite', 'test-db-copy.sqlite')
  
  // Проверяем валидность файла
  const isValid = isValidSQLiteFile('test-db.sqlite')
  expect(isValid).toBe(true)
})
```

## Настройка переменных окружения

Создайте файл `.env` в корне проекта:

```env
BACKEND_URL=http://127.0.0.1:9999
NEXT_PUBLIC_BASE_URL=http://localhost:3000
```

Или установите переменные перед запуском:

```bash
# Windows PowerShell
$env:BACKEND_URL="http://127.0.0.1:9999"
$env:NEXT_PUBLIC_BASE_URL="http://localhost:3000"
npm run test:e2e

# Linux/Mac
BACKEND_URL=http://127.0.0.1:9999 NEXT_PUBLIC_BASE_URL=http://localhost:3000 npm run test:e2e
```

## Фильтрация тестов

### По тегам

```bash
# Запустить только тесты с определенным тегом
npx playwright test --grep "@smoke"
```

### По имени файла

```bash
# Запустить все тесты в определенной директории
npx playwright test tests/e2e/
```

### По паттерну

```bash
# Запустить тесты, содержащие "quality" в названии
npx playwright test -g "quality"
```

## Параллельный запуск

По умолчанию тесты запускаются параллельно. Для последовательного запуска:

```bash
npx playwright test --workers=1
```

## Пропуск тестов

Тесты автоматически пропускаются, если:
- Не найдена тестовая база данных
- Бэкенд недоступен
- Используется `test.skip()` в коде

## Отладка проблем

### Тест не находит элементы

Увеличьте таймауты в `playwright.config.ts` или используйте:

```typescript
await page.waitForSelector('selector', { timeout: 30000 })
```

### Тест падает из-за таймаута

Проверьте, что:
1. Бэкенд запущен и доступен
2. Фронтенд запущен и доступен
3. Тестовая БД существует

### Проблемы с доступностью

Проверьте отчет axe-core в консоли. Критичные нарушения будут показаны с деталями.

## CI/CD интеграция

### GitHub Actions пример

```yaml
name: E2E Tests

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-node@v3
      - name: Install dependencies
        run: |
          npm install
          cd frontend && npm install
      - name: Install Playwright
        run: npx playwright install --with-deps
      - name: Run E2E tests
        run: npm run test:e2e
        env:
          BACKEND_URL: http://localhost:9999
          NEXT_PUBLIC_BASE_URL: http://localhost:3000
```

## Лучшие практики

1. **Используйте beforeAll/afterAll** для подготовки и очистки данных
2. **Используйте test.skip()** для условного пропуска тестов
3. **Добавляйте логирование** через `console.log()` для отладки
4. **Используйте ожидания** вместо фиксированных `waitForTimeout()`
5. **Очищайте данные** после каждого теста для изоляции

## Дополнительные ресурсы

- [Документация Playwright](https://playwright.dev)
- [Документация axe-core](https://www.deque.com/axe/core-documentation/)
- [Лучшие практики E2E тестирования](https://playwright.dev/docs/best-practices)

