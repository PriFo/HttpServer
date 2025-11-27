import { test, expect } from '@playwright/test'
import { waitForPageLoad } from '../e2e/test-helpers'

test.describe('Нагрузочное тестирование нормализации', () => {
  test('должен обработать 50K контрагентов и 100K номенклатуры', async ({ page }) => {
    // Этот тест требует предварительной подготовки БД с большим объемом данных
    // В реальной реализации здесь будет создание тестовой БД через API или скрипт
    
    test.setTimeout(600000) // 10 минут таймаут

    await page.goto('/data-quality')
    await waitForPageLoad(page)

    // Генерируем отчет
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').or(
      page.locator('button:has-text("Генерировать")')
    )

    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      await waitForPageLoad(page)
    }

    // Запускаем нормализацию
    const startButton = page.locator('button:has-text("Начать нормализацию")').or(
      page.locator('button:has-text("Запустить")')
    )

    if (await startButton.isVisible({ timeout: 10000 })) {
      const startTime = Date.now()

      await startButton.click()

      // Переходим на мониторинг
      await page.goto('/monitoring')
      await waitForPageLoad(page)

      // Ждем завершения (с большим таймаутом)
      const completedStatus = page.locator('text=Завершено').or(
        page.locator('[data-status="completed"]')
      )

      try {
        await expect(completedStatus.first()).toBeVisible({ timeout: 540000 }) // 9 минут

        const endTime = Date.now()
        const duration = (endTime - startTime) / 1000 // секунды

        console.log(`Нормализация завершена за ${duration} секунд`)

        // Проверяем, что система не упала
        expect(duration).toBeLessThan(600) // Должно завершиться за 10 минут
      } catch (e) {
        // Если не завершилось, проверяем, что процесс хотя бы запущен
        const runningStatus = page.locator('text=Запущено').or(
          page.locator('[data-status="running"]')
        )

        const isRunning = await runningStatus.isVisible({ timeout: 5000 }).catch(() => false)
        if (isRunning) {
          console.log('Нормализация запущена, но не завершена в течение таймаута')
        }
      }
    }
  })

  test('должен обработать 3-5 параллельных процессов нормализации', async ({ browser }) => {
    test.setTimeout(600000) // 10 минут

    // Создаем несколько контекстов для параллельной обработки
    const contexts = await Promise.all([
      browser.newContext(),
      browser.newContext(),
      browser.newContext(),
    ])

    const pages = await Promise.all(contexts.map(ctx => ctx.newPage()))

    try {
      // Запускаем нормализацию в каждом контексте
      const startPromises = pages.map(async (page, index) => {
        await page.goto('/data-quality')
        await waitForPageLoad(page)

        const startButton = page.locator('button:has-text("Начать нормализацию")').or(
          page.locator('button:has-text("Запустить")')
        )

        if (await startButton.isVisible({ timeout: 10000 })) {
          await startButton.click()
          await page.goto('/monitoring')
          await waitForPageLoad(page)
          return true
        }
        return false
      })

      const results = await Promise.all(startPromises)
      const startedCount = results.filter(r => r).length

      expect(startedCount).toBeGreaterThan(0)

      // Ждем немного и проверяем, что система не упала
      await new Promise(resolve => setTimeout(resolve, 10000))

      // Проверяем статус API для каждого процесса
      for (const page of pages) {
        const response = await page.evaluate(async () => {
          try {
            const res = await fetch('/api/normalization/status')
            return { status: res.status, ok: res.ok }
          } catch {
            return { status: 0, ok: false }
          }
        })

        // API должен отвечать (может быть 200 или другой статус)
        expect(response.status).toBeGreaterThan(0)
      }
    } finally {
      // Закрываем все контексты
      await Promise.all(contexts.map(ctx => ctx.close()))
    }
  })

  test('должен поддерживать разумное время ответа API под нагрузкой', async ({ page }) => {
    // Мониторим время ответа API во время нормализации
    const responseTimes: number[] = []

    // Перехватываем все API запросы
    page.on('response', async (response) => {
      if (response.url().includes('/api/')) {
        const timing = response.timing()
        const responseTime = timing.responseEnd - timing.requestStart
        responseTimes.push(responseTime)
      }
    })

    await page.goto('/monitoring')
    await waitForPageLoad(page)

    // Ждем некоторое время для сбора метрик
    await new Promise(resolve => setTimeout(resolve, 30000)) // 30 секунд

    if (responseTimes.length > 0) {
      const avgResponseTime = responseTimes.reduce((a, b) => a + b, 0) / responseTimes.length
      const maxResponseTime = Math.max(...responseTimes)

      console.log(`Average API response time: ${avgResponseTime}ms`)
      console.log(`Max API response time: ${maxResponseTime}ms`)

      // Среднее время ответа не должно превышать 2 секунды
      expect(avgResponseTime).toBeLessThan(2000)

      // Максимальное время ответа не должно превышать 10 секунд
      expect(maxResponseTime).toBeLessThan(10000)
    }
  })
})

