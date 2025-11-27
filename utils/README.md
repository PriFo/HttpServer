# Утилиты для тестирования

Этот каталог содержит вспомогательные утилиты для E2E тестирования.

## Файлы

### `api-testing.ts`

Утилиты для работы с API бэкенда:

- **Создание тестовых данных:**
  - `createTestClient()` - создание тестового клиента
  - `createTestProject()` - создание тестового проекта
  - `uploadDatabaseFile()` - загрузка базы данных
  - `cleanupTestData()` - очистка тестовых данных

- **Работа с бэкапами:**
  - `createBackup()` - создание бэкапа
  - `listBackups()` - список бэкапов
  - `restoreBackup()` - восстановление из бэкапа

- **Работа с качеством:**
  - `getQualityDuplicates()` - получение дубликатов
  - `mergeDuplicates()` - слияние дубликатов
  - `getQualityMetrics()` - метрики качества

- **Нормализация:**
  - `startNormalization()` - запуск нормализации
  - `stopNormalization()` - остановка нормализации
  - `getNormalizationStatus()` - статус нормализации

- **Работа с бенчмарками:**
  - `listBenchmarks()` - получение списка эталонов
  - `getBenchmarkById()` - получение эталона по ID
  - `searchBenchmarks()` - поиск эталонов
  - `createBenchmarkFromUpload()` - создание эталона из загрузки
  - `createBenchmark()` - создание нового эталона
  - `updateBenchmark()` - обновление эталона
  - `deleteBenchmark()` - удаление эталона

- **Классификация КПВЭД:**
  - `getKpvedHierarchy()` - получение иерархии КПВЭД
  - `searchKpved()` - поиск по КПВЭД
  - `getKpvedStats()` - статистика КПВЭД
  - `testClassification()` - тестирование классификации
  - `classifyHierarchical()` - иерархическая классификация
  - `resetClassification()` - сброс классификации
  - `markClassificationIncorrect()` - пометка неправильной классификации
  - `markClassificationCorrect()` - пометка правильной классификации

- **Вспомогательные:**
  - `findTestDatabase()` - поиск тестовой БД в стандартных местах

### `database-testing.ts`

Утилиты для работы с тестовыми базами данных (без нативных зависимостей):

- `createTestDatabase()` - создание тестовой БД (использует шаблон, если доступен)
- `checkDatabaseIntegrity()` - проверка целостности БД (проверка заголовка SQLite)
- `getDatabaseStats()` - получение статистики БД (размер, дата изменения)
- `copyDatabase()` - копирование БД
- `deleteDatabase()` - удаление БД
- `isValidSQLiteFile()` - проверка валидности SQLite файла

### `auth-testing.ts`

Утилиты для работы с аутентификацией (если требуется):

- `createAdminToken()` - создание токена администратора
- `createManagerToken()` - создание токена менеджера
- `createViewerToken()` - создание токена наблюдателя
- `addAuthHeader()` - добавление заголовка авторизации
- `isAccessDeniedError()` - проверка ошибки доступа

## Использование

```typescript
import { 
  createTestClient, 
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData 
} from '../../utils/api-testing'

import {
  createTestDatabase,
  checkDatabaseIntegrity,
  copyDatabase
} from '../../utils/database-testing'

// Создание тестовых данных
const client = await createTestClient({ name: 'Test Client' })
const project = await createTestProject(client.id, { name: 'Test Project' })

// Работа с БД
const dbPath = 'test-db.sqlite'
await createTestDatabase(dbPath)
const isIntact = await checkDatabaseIntegrity(dbPath)

// Очистка
await cleanupTestData(client.id, project.id)
```

## Конфигурация

Все утилиты используют переменные окружения:

- `BACKEND_URL` - URL бэкенда (по умолчанию: `http://127.0.0.1:9999`)
- `NEXT_PUBLIC_BASE_URL` - URL фронтенда (по умолчанию: `http://localhost:3000`)

## Примечания

- Утилиты `database-testing.ts` не требуют нативных зависимостей (sqlite3)
- Для полноценной работы с БД рекомендуется использовать существующую тестовую БД
- Все функции автоматически обрабатывают ошибки и логируют действия
