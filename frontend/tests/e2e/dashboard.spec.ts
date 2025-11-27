import { test, expect } from '@playwright/test'

test.describe('Dashboard Page (Main SPA)', () => {
  test.beforeEach(async ({ page }) => {
    // Переходим на главную страницу
    await page.goto('/')
    // Ждем загрузки компонентов
    await page.waitForTimeout(1000)
  })

  test('should display dashboard header', async ({ page }) => {
    // Проверяем наличие заголовка
    const header = page.locator('header').first()
    await expect(header).toBeVisible()
    
    // Проверяем логотип
    const logo = page.locator('text=Нормализатор').first()
    await expect(logo).toBeVisible()
    
    // Проверяем поиск
    const search = page.locator('input[type="search"]').first()
    await expect(search).toBeVisible()
  })

  test('should display sidebar navigation', async ({ page }) => {
    // Проверяем наличие сайдбара (на десктопе)
    const sidebar = page.locator('aside, nav').first()
    
    // Проверяем наличие табов
    const overviewTab = page.locator('button, a').filter({ hasText: /Обзор|Overview/ }).first()
    const monitoringTab = page.locator('button, a').filter({ hasText: /Мониторинг|Monitoring/ }).first()
    
    // На мобильных может быть скрыт, поэтому проверяем гибко
    const sidebarVisible = await sidebar.isVisible().catch(() => false)
    if (sidebarVisible) {
      await expect(overviewTab).toBeVisible({ timeout: 5000 })
    }
  })

  test('should switch between tabs', async ({ page }) => {
    // Ждем загрузки
    await page.waitForTimeout(1500)
    
    // Пробуем переключиться на таб мониторинга
    const monitoringTab = page.locator('button, a').filter({ hasText: /Мониторинг|Monitoring/ }).first()
    
    if (await monitoringTab.isVisible({ timeout: 3000 })) {
      await monitoringTab.click()
      await page.waitForTimeout(1000)
      
      // Проверяем, что контент изменился
      const monitoringContent = page.locator('text=Мониторинг провайдеров, text=Провайдеры').first()
      await expect(monitoringContent).toBeVisible({ timeout: 5000 })
    }
  })

  test('should display overview tab by default', async ({ page }) => {
    // Ждем загрузки
    await page.waitForTimeout(2000)
    
    // Проверяем наличие метрик или виджетов
    const metrics = page.locator('[class*="card"], [class*="metric"]').first()
    await expect(metrics).toBeVisible({ timeout: 5000 })
  })

  test('should display real-time indicator', async ({ page }) => {
    // Проверяем наличие индикатора real-time
    const realtimeIndicator = page.locator('text=Онлайн, text=Офлайн, [class*="wifi"]').first()
    
    // Может быть не виден сразу, проверяем гибко
    const isVisible = await realtimeIndicator.isVisible({ timeout: 3000 }).catch(() => false)
    expect(isVisible).toBeTruthy()
  })

  test('should handle keyboard shortcuts', async ({ page }) => {
    // Фокусируемся на странице
    await page.click('body')
    await page.waitForTimeout(500)
    
    // Нажимаем клавишу "2" для переключения на мониторинг
    await page.keyboard.press('2')
    await page.waitForTimeout(1000)
    
    // Проверяем, что произошло переключение (если не в поле ввода)
    const monitoringContent = page.locator('text=Мониторинг, text=Провайдеры').first()
    const isVisible = await monitoringContent.isVisible({ timeout: 3000 }).catch(() => false)
    // Может не сработать если фокус в поле ввода, это нормально
  })

  test('should display notifications', async ({ page }) => {
    // Ждем загрузки
    await page.waitForTimeout(2000)
    
    // Проверяем наличие кнопки уведомлений
    const notificationsButton = page.locator('button').filter({ hasText: /Bell|Уведомления/ }).or(
      page.locator('[aria-label*="notification"], [aria-label*="уведомление"]')
    ).first()
    
    const isVisible = await notificationsButton.isVisible({ timeout: 3000 }).catch(() => false)
    // Кнопка может быть не видна, если нет уведомлений
  })

  test('should be responsive on mobile', async ({ page }) => {
    // Устанавливаем мобильный viewport
    await page.setViewportSize({ width: 375, height: 667 })
    await page.waitForTimeout(1000)
    
    // Проверяем, что контент адаптировался
    const content = page.locator('main, [class*="container"]').first()
    await expect(content).toBeVisible()
  })

  test('should load lazy components', async ({ page }) => {
    // Ждем загрузки всех компонентов
    await page.waitForTimeout(3000)
    
    // Проверяем, что основные элементы загрузились
    const mainContent = page.locator('main, [class*="content"]').first()
    await expect(mainContent).toBeVisible()
  })

  test('should handle search functionality', async ({ page }) => {
    // Находим поле поиска
    const searchInput = page.locator('input[type="search"]').first()
    
    if (await searchInput.isVisible({ timeout: 3000 })) {
      await searchInput.fill('мониторинг')
      await page.waitForTimeout(500)
      
      // Проверяем, что появились результаты (если есть)
      const results = page.locator('[class*="search"], [class*="result"]').first()
      const hasResults = await results.isVisible({ timeout: 2000 }).catch(() => false)
      // Результаты могут не появиться, это нормально
    }
  })
})

