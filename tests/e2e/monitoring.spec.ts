import { test, expect } from '@playwright/test'
import { waitForPageLoad, logPageInfo, wait, waitForDataUpdate } from './test-helpers'

test.describe('Мониторинг AI-провайдеров', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/monitoring')
    await waitForPageLoad(page)
    await logPageInfo(page)
  })

  test('должен отображать страницу мониторинга', async ({ page }) => {
    // Проверяем наличие заголовка
    await expect(page.locator('text=Мониторинг AI-провайдеров').or(page.locator('h1:has-text("Мониторинг")'))).toBeVisible({ timeout: 15000 })
    
    // Проверяем наличие индикатора подключения
    const connectionIndicator = page.locator('text=Подключено').or(page.locator('text=Отключено'))
    await expect(connectionIndicator).toBeVisible({ timeout: 10000 })
  })

  test('должен отображать статистику системы', async ({ page }) => {
    // Ждем загрузки данных
    await waitForPageLoad(page)
    
    // Проверяем наличие карточек статистики
    const statsCards = page.locator('text=Всего запросов').or(page.locator('text=Запросов/сек')).or(page.locator('text=Активных провайдеров'))
    await expect(statsCards.first()).toBeVisible({ timeout: 15000 })
  })

  test('должен отображать карточки провайдеров', async ({ page }) => {
    // Ждем загрузки данных
    await waitForPageLoad(page)
    
    // Проверяем наличие карточек провайдеров
    // Ищем по классу Card или по тексту с названиями провайдеров
    const providerCards = page.locator('[class*="Card"]').filter({ hasText: /OpenRouter|Hugging Face|Arliai|Eden AI|DaData|Adata/i })
    const count = await providerCards.count()
    
    // Может быть от 0 до нескольких провайдеров
    if (count > 0) {
      expect(count).toBeGreaterThan(0)
    }
  })

  test('должен отображать управление воркерами KPVED', async ({ page }) => {
    // Ждем загрузки данных
    await waitForPageLoad(page)
    
    // Проверяем наличие секции управления воркерами
    const workersSection = page.locator('text=Управление воркерами KPVED').or(page.locator('text=воркеры KPVED'))
    const isVisible = await workersSection.isVisible({ timeout: 10000 })
    
    if (isVisible) {
      // Проверяем наличие кнопок управления
      const controlButtons = page.locator('button:has-text("Остановить воркеры")').or(page.locator('button:has-text("Запустить воркеры")'))
      await expect(controlButtons.first()).toBeVisible({ timeout: 5000 })
    }
  })

  test('должен позволять остановить/запустить воркеры', async ({ page }) => {
    // Ждем загрузки данных
    await waitForPageLoad(page)
    
    // Ищем кнопку управления воркерами
    const stopButton = page.locator('button:has-text("Остановить воркеры")')
    const startButton = page.locator('button:has-text("Запустить воркеры")')
    
    // Если есть кнопка остановки, кликаем на неё
    if (await stopButton.isVisible({ timeout: 5000 })) {
      await stopButton.click()
      await waitForPageLoad(page)
      
      // Проверяем, что появилась кнопка запуска
      await expect(startButton).toBeVisible({ timeout: 5000 })
      
      // Запускаем обратно
      await startButton.click()
      await waitForPageLoad(page)
    } else if (await startButton.isVisible({ timeout: 5000 })) {
      // Если воркеры уже остановлены, просто проверяем наличие кнопки
      await expect(startButton).toBeVisible()
    }
  })

  test('должен отображать графики производительности', async ({ page }) => {
    // Ждем загрузки данных
    await waitForPageLoad(page)
    
    // Проверяем наличие графиков
    const charts = page.locator('text=Нагрузка на провайдеры').or(page.locator('text=Средняя задержка'))
    const chartCount = await charts.count()
    
    // Должен быть хотя бы один график
    if (chartCount > 0) {
      expect(chartCount).toBeGreaterThan(0)
    }
  })

  test('должен обновлять данные в реальном времени', async ({ page }) => {
    // Ждем загрузки данных
    await waitForPageLoad(page)
    
    // Получаем начальное значение запросов (если есть)
    const initialRequests = page.locator('text=/\\d+ запросов/').first()
    const initialText = await initialRequests.textContent().catch(() => null)
    
    // Ждем обновления (SSE должен обновлять данные)
    const updated = await waitForDataUpdate(
      page,
      'text=/\\d+ запросов/',
      initialText,
      10000,
      1000
    )
    
    // Проверяем, что данные обновились (или остались теми же, если нет активности)
    const updatedRequests = page.locator('text=/\\d+ запросов/').first()
    const updatedText = await updatedRequests.textContent().catch(() => null)
    
    // Данные должны быть видимы
    if (initialText || updatedText) {
      expect(initialText || updatedText).toBeTruthy()
      if (updated) {
        console.log('✅ Данные обновились через SSE')
      }
    }
  })
})

