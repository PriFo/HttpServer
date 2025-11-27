# Тестирование

Этот каталог содержит тесты для фронтенда.

## Структура

```
tests/
├── e2e/              # E2E тесты (Playwright)
│   ├── monitoring.spec.ts
│   └── reports.spec.ts
└── components/       # Компонентные тесты (Jest + React Testing Library)
    ├── monitoring-provider-card.test.tsx
    └── reports-page.test.tsx
```

## E2E тесты (Playwright)

### Установка

```bash
npm install --save-dev @playwright/test
npx playwright install
```

### Запуск

```bash
# Все тесты
npm run test:e2e

# С UI
npm run test:e2e:ui

# В режиме отладки
npm run test:e2e:debug

# Просмотр отчета
npm run test:e2e:report
```

### Написание новых тестов

Создайте файл в `tests/e2e/` с расширением `.spec.ts`:

```typescript
import { test, expect } from '@playwright/test'

test.describe('My Feature', () => {
  test('should work correctly', async ({ page }) => {
    await page.goto('/my-page')
    await expect(page.locator('h1')).toContainText('My Page')
  })
})
```

## Компонентные тесты (Jest)

### Установка

```bash
npm install --save-dev @testing-library/react @testing-library/jest-dom jest jest-environment-jsdom @types/jest
```

### Настройка Jest

Создайте файл `jest.config.js`:

```javascript
module.exports = {
  testEnvironment: 'jsdom',
  setupFilesAfterEnv: ['<rootDir>/jest.setup.js'],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/$1',
  },
  testMatch: ['**/tests/components/**/*.test.tsx'],
}
```

Создайте файл `jest.setup.js`:

```javascript
import '@testing-library/jest-dom'
```

### Запуск

```bash
npm test
```

### Написание новых тестов

Создайте файл в `tests/components/` с расширением `.test.tsx`:

```typescript
import { render, screen } from '@testing-library/react'
import { MyComponent } from '@/components/MyComponent'

describe('MyComponent', () => {
  it('should render correctly', () => {
    render(<MyComponent />)
    expect(screen.getByText('Hello')).toBeInTheDocument()
  })
})
```

## Покрытие кода

Для проверки покрытия кода тестами:

```bash
# E2E (Playwright)
npx playwright test --coverage

# Компонентные (Jest)
npm test -- --coverage
```

## CI/CD

Тесты автоматически запускаются в CI/CD пайплайне. Убедитесь, что:
- Все тесты проходят локально перед коммитом
- Тесты не зависят от внешних сервисов (используйте моки)
- Тесты выполняются быстро (< 30 секунд для E2E)

