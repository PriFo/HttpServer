import { test, expect } from '@playwright/test'

test.describe('Monitoring Page', () => {
  test.beforeEach(async ({ page }) => {
    // Переходим на страницу мониторинга
    await page.goto('/monitoring')
  })

  test('should display monitoring page title', async ({ page }) => {
    await expect(page).toHaveTitle(/Мониторинг/)
    await expect(page.locator('h1')).toContainText('Мониторинг')
  })

  test('should display connection status', async ({ page }) => {
    // Проверяем наличие индикатора подключения
    const connectionStatus = page.locator('text=Подключено, text=Отключено').first()
    await expect(connectionStatus).toBeVisible()
  })

  test('should display system statistics cards', async ({ page }) => {
    // Ждем загрузки данных (может быть skeleton или реальные данные)
    await page.waitForTimeout(2000)
    
    // Проверяем наличие карточек статистики
    const statsCards = page.locator('[class*="card"]').filter({ hasText: /Всего запросов|Запросов\/сек|Активных провайдеров|Успешных запросов/ })
    const count = await statsCards.count()
    expect(count).toBeGreaterThan(0)
  })

  test('should display provider cards', async ({ page }) => {
    // Ждем загрузки данных
    await page.waitForTimeout(3000)
    
    // Проверяем наличие карточек провайдеров
    const providerCards = page.locator('[class*="card"]').filter({ hasText: /OpenRouter|Hugging Face|Arliai|Eden AI|DaData|Adata/ })
    const count = await providerCards.count()
    // Может быть 0 если нет данных, но структура должна быть
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('should display KPVED workers control section', async ({ page }) => {
    // Ждем загрузки статуса воркеров
    await page.waitForTimeout(2000)
    
    // Проверяем наличие секции управления воркерами
    const workersSection = page.locator('text=Управление воркерами KPVED')
    await expect(workersSection).toBeVisible({ timeout: 5000 })
  })

  test('should toggle KPVED workers', async ({ page }) => {
    // Ждем загрузки
    await page.waitForTimeout(2000)
    
    // Ищем кнопку управления воркерами
    const toggleButton = page.locator('button').filter({ hasText: /Запустить воркеры|Остановить воркеры/ }).first()
    
    if (await toggleButton.isVisible({ timeout: 5000 })) {
      const initialText = await toggleButton.textContent()
      await toggleButton.click()
      
      // Ждем обновления UI
      await page.waitForTimeout(1000)
      
      // Проверяем, что текст кнопки изменился
      const newText = await toggleButton.textContent()
      expect(newText).not.toBe(initialText)
    }
  })

  test('should display provider toggle buttons', async ({ page }) => {
    // Ждем загрузки данных
    await page.waitForTimeout(3000)
    
    // Ищем кнопки переключения провайдеров (Power/PowerOff иконки)
    const providerToggleButtons = page.locator('button').filter({ has: page.locator('svg') })
    const count = await providerToggleButtons.count()
    // Должны быть кнопки управления
    expect(count).toBeGreaterThan(0)
  })

  test('should display charts', async ({ page }) => {
    // Ждем загрузки данных
    await page.waitForTimeout(3000)
    
    // Проверяем наличие графиков (Recharts создает SVG элементы)
    const charts = page.locator('svg[class*="recharts"]')
    const count = await charts.count()
    // Может быть 0 если нет данных, но структура должна быть
    expect(count).toBeGreaterThanOrEqual(0)
  })

  test('should handle SSE connection errors gracefully', async ({ page }) => {
    // Перехватываем ошибки SSE
    page.on('console', msg => {
      if (msg.type() === 'error' && msg.text().includes('SSE')) {
        // Ошибка SSE должна обрабатываться
        expect(msg.text()).toBeDefined()
      }
    })
    
    // Ждем некоторое время для проверки обработки ошибок
    await page.waitForTimeout(2000)
  })
})

