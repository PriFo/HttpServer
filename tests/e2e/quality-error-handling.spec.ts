/**
 * 📋 E2E ТЕСТЫ ДЛЯ ОБРАБОТКИ ОШИБОК НА СТРАНИЦЕ КАЧЕСТВА ДАННЫХ
 * 
 * Этот тестовый набор проверяет обработку ошибок на странице /quality:
 * - Обработка таймаутов
 * - Обработка сетевых ошибок
 * - Обработка ошибок сервера
 * - Функциональность кнопки "Повторить"
 * - Отображение ErrorState компонентов
 * 
 * Prerequisites:
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 */

import { test, expect } from '@playwright/test'

const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'

test.describe('Страница качества данных - Обработка ошибок', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto(`${FRONTEND_URL}/quality`)
    // Ждем загрузки страницы
    await page.waitForLoadState('networkidle')
  })

  test('должен отображать все вкладки и карточки', async ({ page }) => {
    // Проверяем наличие вкладок
    await expect(page.locator('text=Обзор').or(page.locator('[role="tab"]:has-text("Обзор"))')).toBeVisible()
    await expect(page.locator('text=Дубликаты').or(page.locator('[role="tab"]:has-text("Дубликаты"))')).toBeVisible()
    await expect(page.locator('text=Нарушения').or(page.locator('[role="tab"]:has-text("Нарушения"))')).toBeVisible()
    await expect(page.locator('text=Предложения').or(page.locator('[role="tab"]:has-text("Предложения"))')).toBeVisible()
    await expect(page.locator('text=Отчёт').or(page.locator('[role="tab"]:has-text("Отчёт"))')).toBeVisible()
  })

  test('должен обрабатывать ошибку сети и показывать ее пользователю', async ({ page }) => {
    // Мокируем сбой сети для всех API запросов
    await page.route('**/api/quality/**', (route) => route.abort('failed'))

    // Выбираем базу данных (если есть селектор)
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
      await page.waitForTimeout(2000)
    }

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Не удалось подключиться|Ошибка подключения|Network error/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('должен позволять пользователю повторить операцию после ошибки', async ({ page }) => {
    let requestCount = 0

    // Мокируем сбой, затем успех
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
    if (await dbSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
      await page.waitForTimeout(2000)
    }

    // Проверяем наличие сообщения об ошибке
    await expect(page.locator('text=/Ошибка|Error/i').first()).toBeVisible({ timeout: 5000 })

    // Находим кнопку "Повторить" и нажимаем ее
    const retryButton = page.locator('button:has-text("Повторить")').first()
    if (await retryButton.isVisible({ timeout: 5000 }).catch(() => false)) {
      await retryButton.click()
      await page.waitForTimeout(3000)

      // Проверяем, что данные начали загружаться снова
      expect(requestCount).toBeGreaterThan(1)
    }
  })

  test('должен обрабатывать таймаут запроса', async ({ page }) => {
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

    // Выбираем базу данных
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
    }

    // Ждем появления сообщения об ошибке таймаута
    await page.waitForTimeout(12000)

    // Проверяем наличие сообщения об ошибке таймаута
    const timeoutMessage = page.locator('text=/Превышено время ожидания|Таймаут|Timeout/i')
    await expect(timeoutMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('должен обрабатывать ошибку 503 (Service Unavailable)', async ({ page }) => {
    await page.route('**/api/quality/**', (route) => {
      route.fulfill({
        status: 503,
        contentType: 'application/json',
        body: JSON.stringify({ 
          error: 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен.' 
        }),
      })
    })

    // Выбираем базу данных
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
      await page.waitForTimeout(2000)
    }

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Не удалось подключиться|Service Unavailable|Сервер недоступен/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('должен обрабатывать ошибку 404 на вкладке дубликатов', async ({ page }) => {
    await page.route('**/api/quality/duplicates**', (route) => {
      route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Дубликаты не найдены' }),
      })
    })

    // Переходим на вкладку дубликатов
    await page.click('text=Дубликаты')
    await page.waitForTimeout(2000)

    // Проверяем наличие сообщения об ошибке или пустого состояния
    const errorOrEmpty = page.locator('text=/не найдены|Дубликатов не найдено|Not found/i')
    await expect(errorOrEmpty.first()).toBeVisible({ timeout: 5000 })
  })

  test('должен обрабатывать ошибку на вкладке нарушений', async ({ page }) => {
    await page.route('**/api/quality/violations**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Ошибка сервера. Попробуйте позже' }),
      })
    })

    // Переходим на вкладку нарушений
    await page.click('text=Нарушения')
    await page.waitForTimeout(2000)

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Ошибка загрузки|Не удалось загрузить|Error/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })

    // Проверяем наличие кнопки "Повторить"
    const retryButton = page.locator('button:has-text("Повторить")').first()
    await expect(retryButton).toBeVisible({ timeout: 5000 })
  })

  test('должен обрабатывать ошибку на вкладке предложений', async ({ page }) => {
    await page.route('**/api/quality/suggestions**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Ошибка сервера. Попробуйте позже' }),
      })
    })

    // Переходим на вкладку предложений
    await page.click('text=Предложения')
    await page.waitForTimeout(2000)

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Ошибка загрузки|Не удалось загрузить|Error/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })

    // Проверяем наличие кнопки "Повторить"
    const retryButton = page.locator('button:has-text("Повторить")').first()
    await expect(retryButton).toBeVisible({ timeout: 5000 })
  })

  test('должен обрабатывать ошибку на вкладке отчета', async ({ page }) => {
    await page.route('**/api/quality/report**', (route) => {
      route.fulfill({
        status: 404,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Отчёт не найден' }),
      })
    })

    // Переходим на вкладку отчета
    await page.click('text=Отчёт')
    await page.waitForTimeout(2000)

    // Проверяем наличие сообщения об ошибке
    const errorMessage = page.locator('text=/Ошибка загрузки|не найден|Not found/i')
    await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
  })

  test('должен показывать счетчик попыток при повторных ошибках', async ({ page }) => {
    let requestCount = 0

    // Всегда возвращаем ошибку
    await page.route('**/api/quality/stats**', async (route) => {
      requestCount++
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      })
    })

    // Выбираем базу данных
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
      await page.waitForTimeout(2000)
    }

    // Нажимаем "Повторить" несколько раз
    const retryButton = page.locator('button:has-text("Повторить")').first()
    if (await retryButton.isVisible({ timeout: 5000 }).catch(() => false)) {
      for (let i = 0; i < 2; i++) {
        await retryButton.click()
        await page.waitForTimeout(2000)
      }

      // Проверяем, что было сделано несколько попыток
      expect(requestCount).toBeGreaterThan(1)
    }
  })

  test('должен корректно обрабатывать одновременные ошибки на разных вкладках', async ({ page }) => {
    // Мокируем ошибки для всех API
    await page.route('**/api/quality/**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      })
    })

    // Переключаемся между вкладками
    const tabs = ['Дубликаты', 'Нарушения', 'Предложения', 'Отчёт']
    
    for (const tab of tabs) {
      await page.click(`text=${tab}`)
      await page.waitForTimeout(2000)

      // Проверяем наличие сообщения об ошибке
      const errorMessage = page.locator('text=/Ошибка|Error/i')
      await expect(errorMessage.first()).toBeVisible({ timeout: 5000 })
    }
  })

  test('должен корректно отображать ErrorState компонент', async ({ page }) => {
    await page.route('**/api/quality/stats**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      })
    })

    // Выбираем базу данных
    const dbSelector = page.locator('[data-testid="database-selector"], select[name="database"]').first()
    if (await dbSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
      await dbSelector.selectOption({ index: 0 })
      await page.waitForTimeout(2000)
    }

    // Проверяем наличие ErrorState компонента
    const errorState = page.locator('[role="alert"]').or(
      page.locator('.error-state')
    ).first()
    
    await expect(errorState).toBeVisible({ timeout: 5000 })

    // Проверяем наличие кнопки "Повторить" в ErrorState
    const retryButton = errorState.locator('button:has-text("Повторить")')
    await expect(retryButton).toBeVisible({ timeout: 5000 })
  })
})

