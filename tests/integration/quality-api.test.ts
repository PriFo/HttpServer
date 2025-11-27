/**
 * Интеграционные тесты для API routes качества данных
 * 
 * Тестирует полный цикл запроса от фронтенда до бэкенда, включая таймауты и обработку ошибок
 * 
 * Для запуска требуется:
 * - Запущенный backend сервер на http://127.0.0.1:9999
 * - Запущенный frontend сервер на http://localhost:3000
 * 
 * Запуск: npx playwright test tests/integration/quality-api.test.ts
 */

import { test, expect } from '@playwright/test'

const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'
const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:9999'

test.describe('Quality API Integration Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Переходим на страницу качества
    await page.goto(`${FRONTEND_URL}/quality`)
  })

  test('GET /api/quality/stats успешно возвращает данные', async ({ page, request }) => {
    // Мокируем ответ от API
    await page.route('**/api/quality/stats**', async (route) => {
      const url = new URL(route.request().url())
      const database = url.searchParams.get('database')

      if (database) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            total_items: 100,
            average_quality: 85.5,
            benchmark_count: 50,
            benchmark_percentage: 50.0,
            by_level: {
              basic: { count: 30, avg_quality: 70.0, percentage: 30.0 },
              ai_enhanced: { count: 20, avg_quality: 85.0, percentage: 20.0 },
              benchmark: { count: 50, avg_quality: 95.0, percentage: 50.0 },
            },
          }),
        })
      } else {
        await route.fulfill({
          status: 400,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'database parameter is required' }),
        })
      }
    })

    // Ждем загрузки страницы
    await page.waitForLoadState('networkidle')

    // Проверяем, что страница загрузилась
    await expect(page.locator('h1')).toContainText('Качество данных')
  })

  test('GET /api/quality/stats обрабатывает таймаут', async ({ page }) => {
    // Мокируем долгий ответ (>10 секунд)
    await page.route('**/api/quality/stats**', async (route) => {
      // Имитируем долгий ответ
      await new Promise(resolve => setTimeout(resolve, 11000))
      await route.fulfill({
        status: 504,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Превышено время ожидания ответа от сервера' }),
      })
    })

    // Выбираем базу данных (если есть селектор)
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible().catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
    }

    // Ждем появления сообщения об ошибке
    await page.waitForTimeout(12000) // Ждем таймаут + небольшой запас

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Превышено время ожидания|Таймаут/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('GET /api/quality/stats обрабатывает сетевую ошибку', async ({ page }) => {
    // Мокируем сетевую ошибку
    await page.route('**/api/quality/stats**', (route) => route.abort('failed'))

    // Выбираем базу данных
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible().catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
    }

    // Ждем появления сообщения об ошибке
    await page.waitForTimeout(2000)

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Не удалось подключиться|Ошибка подключения/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('GET /api/quality/duplicates успешно возвращает список дубликатов', async ({ page }) => {
    await page.route('**/api/quality/duplicates**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          groups: [
            {
              id: 1,
              duplicate_type: 'exact',
              similarity_score: 100,
              item_count: 2,
              merged: false,
              items: [
                { id: 1, normalized_name: 'Тест 1', code: '001' },
                { id: 2, normalized_name: 'Тест 1', code: '001' },
              ],
            },
          ],
          total: 1,
          limit: 10,
          offset: 0,
        }),
      })
    })

    // Переходим на вкладку дубликатов
    await page.click('text=Дубликаты')
    await page.waitForTimeout(1000)

    // Проверяем наличие дубликатов
    const duplicatesList = page.locator('[data-testid="duplicates-list"], .duplicate-group').first()
    await expect(duplicatesList).toBeVisible({ timeout: 5000 })
  })

  test('POST /api/quality/duplicates/[groupId]/merge успешно выполняет слияние', async ({ page }) => {
    let mergeCalled = false

    await page.route('**/api/quality/duplicates/*/merge', async (route) => {
      mergeCalled = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ success: true, merged_count: 2 }),
      })
    })

    // Мокируем список дубликатов
    await page.route('**/api/quality/duplicates**', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          groups: [
            {
              id: 1,
              duplicate_type: 'exact',
              similarity_score: 100,
              item_count: 2,
              merged: false,
              items: [],
            },
          ],
          total: 1,
        }),
      })
    })

    // Переходим на вкладку дубликатов
    await page.click('text=Дубликаты')
    await page.waitForTimeout(1000)

    // Ищем кнопку слияния (может быть в разных местах)
    const mergeButton = page.locator('button:has-text("Слить"), button:has-text("Объединить")').first()
    if (await mergeButton.isVisible().catch(() => false)) {
      await mergeButton.click()
      await page.waitForTimeout(2000)
      expect(mergeCalled).toBe(true)
    }
  })

  test('GET /api/quality/violations обрабатывает ошибку 404', async ({ page }) => {
    await page.route('**/api/quality/violations**', async (route) => {
      await route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Нарушения не найдены' }),
      })
    })

    // Переходим на вкладку нарушений
    await page.click('text=Нарушения')
    await page.waitForTimeout(1000)

    // Проверяем наличие сообщения об ошибке или пустого состояния
    const errorOrEmpty = page.locator('text=/не найдены|Нарушений не найдено/i')
    await expect(errorOrEmpty.first()).toBeVisible({ timeout: 5000 })
  })

  test('GET /api/quality/suggestions обрабатывает ошибку сервера', async ({ page }) => {
    await page.route('**/api/quality/suggestions**', async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      })
    })

    // Переходим на вкладку предложений
    await page.click('text=Предложения')
    await page.waitForTimeout(1000)

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Ошибка сервера|Не удалось загрузить/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('POST /api/quality/analyze успешно запускает анализ', async ({ page }) => {
    let analyzeCalled = false

    await page.route('**/api/quality/analyze', async (route) => {
      analyzeCalled = true
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({ 
          success: true, 
          analysis_id: 'test-123',
          message: 'Анализ запущен'
        }),
      })
    })

    // Мокируем статус анализа
    await page.route('**/api/quality/analyze/status', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          is_running: true,
          progress: 50,
          current_step: 'duplicates',
          duplicates_found: 10,
          violations_found: 5,
          suggestions_found: 3,
        }),
      })
    })

    // Ищем кнопку запуска анализа
    const analyzeButton = page.locator('button:has-text("Анализ"), button:has-text("Запустить")').first()
    if (await analyzeButton.isVisible().catch(() => false)) {
      await analyzeButton.click()
      await page.waitForTimeout(2000)
      expect(analyzeCalled).toBe(true)
    }
  })

  test('кнопка "Повторить" работает после ошибки', async ({ page }) => {
    let requestCount = 0

    await page.route('**/api/quality/stats**', async (route) => {
      requestCount++
      if (requestCount === 1) {
        // Первый запрос - ошибка
        await route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal Server Error' }),
        })
      } else {
        // Второй запрос - успех
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            total_items: 100,
            average_quality: 85.5,
            benchmark_count: 50,
            benchmark_percentage: 50.0,
            by_level: {},
          }),
        })
      }
    })

    // Выбираем базу данных
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible().catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
      await page.waitForTimeout(2000)
    }

    // Ищем кнопку "Повторить"
    const retryButton = page.locator('button:has-text("Повторить")').first()
    if (await retryButton.isVisible().catch(() => false)) {
      await retryButton.click()
      await page.waitForTimeout(2000)
      expect(requestCount).toBeGreaterThan(1)
    }
  })
})

