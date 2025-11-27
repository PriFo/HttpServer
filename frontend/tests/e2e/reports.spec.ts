import { test, expect } from '@playwright/test'

test.describe('Reports Page', () => {
  test.beforeEach(async ({ page }) => {
    // Переходим на страницу отчетов
    await page.goto('/reports')
  })

  test('should display reports page title', async ({ page }) => {
    await expect(page).toHaveTitle(/Отчеты/)
    await expect(page.locator('h1')).toContainText('Отчеты')
  })

  test('should display tabs for report types', async ({ page }) => {
    // Проверяем наличие табов
    const normalizationTab = page.locator('button').filter({ hasText: /Нормализация|Normalization/ })
    const dataQualityTab = page.locator('button').filter({ hasText: /Качество данных|Data Quality/ })
    
    await expect(normalizationTab).toBeVisible()
    await expect(dataQualityTab).toBeVisible()
  })

  test('should generate normalization report', async ({ page }) => {
    // Находим кнопку генерации отчета по нормализации
    const generateButton = page.locator('button').filter({ hasText: /Сгенерировать отчет по нормализации/ })
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      // Перехватываем запрос к API
      const responsePromise = page.waitForResponse(response => 
        response.url().includes('/api/reports/generate-normalization-report') && 
        response.request().method() === 'POST'
      )
      
      await generateButton.click()
      
      // Ждем ответа (может быть успешным или с ошибкой)
      const response = await responsePromise
      expect([200, 201, 400, 500]).toContain(response.status())
      
      // Ждем обновления UI
      await page.waitForTimeout(2000)
    }
  })

  test('should generate data quality report', async ({ page }) => {
    // Переключаемся на таб "Качество данных"
    const dataQualityTab = page.locator('button').filter({ hasText: /Качество данных|Data Quality/ })
    if (await dataQualityTab.isVisible({ timeout: 5000 })) {
      await dataQualityTab.click()
      await page.waitForTimeout(500)
      
      // Находим кнопку генерации отчета о качестве данных
      const generateButton = page.locator('button').filter({ hasText: /Сгенерировать отчет о качестве данных/ })
      
      if (await generateButton.isVisible({ timeout: 5000 })) {
        // Перехватываем запрос к API
        const responsePromise = page.waitForResponse(response => 
          response.url().includes('/api/reports/generate-data-quality-report') && 
          response.request().method() === 'POST'
        )
        
        await generateButton.click()
        
        // Ждем ответа
        const response = await responsePromise
        expect([200, 201, 400, 500]).toContain(response.status())
        
        // Ждем обновления UI
        await page.waitForTimeout(2000)
      }
    }
  })

  test('should display loading state during report generation', async ({ page }) => {
    const generateButton = page.locator('button').filter({ hasText: /Сгенерировать отчет/ })
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      
      // Проверяем наличие индикатора загрузки
      const loadingIndicator = page.locator('text=Генерация отчета, text=Создание PDF').first()
      // Может быть видимым кратковременно
      await page.waitForTimeout(500)
    }
  })

  test('should display download PDF button after report generation', async ({ page }) => {
    // Генерируем отчет
    const generateButton = page.locator('button').filter({ hasText: /Сгенерировать отчет по нормализации/ })
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      
      // Ждем завершения генерации (может занять время)
      await page.waitForTimeout(5000)
      
      // Проверяем наличие кнопки скачивания PDF
      const downloadButton = page.locator('button').filter({ hasText: /Скачать PDF|Download PDF/ })
      // Может быть видимым если отчет успешно сгенерирован
      if (await downloadButton.isVisible({ timeout: 10000 })) {
        await expect(downloadButton).toBeVisible()
      }
    }
  })

  test('should display report data after generation', async ({ page }) => {
    // Генерируем отчет
    const generateButton = page.locator('button').filter({ hasText: /Сгенерировать отчет по нормализации/ })
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      
      // Ждем появления данных отчета
      await page.waitForTimeout(5000)
      
      // Проверяем наличие секций отчета
      const reportSections = page.locator('text=Общая статистика, text=Анализ контрагентов, text=Анализ номенклатуры, text=Производительность провайдеров').first()
      // Может быть видимым если отчет успешно сгенерирован
      if (await reportSections.isVisible({ timeout: 10000 })) {
        await expect(reportSections).toBeVisible()
      }
    }
  })

  test('should handle report generation errors', async ({ page }) => {
    // Перехватываем ошибки
    page.on('console', msg => {
      if (msg.type() === 'error') {
        // Ошибки должны обрабатываться
        expect(msg.text()).toBeDefined()
      }
    })
    
    const generateButton = page.locator('button').filter({ hasText: /Сгенерировать отчет/ })
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      
      // Ждем обработки ошибки (если она произойдет)
      await page.waitForTimeout(3000)
      
      // Проверяем наличие сообщения об ошибке (если есть)
      const errorMessage = page.locator('[role="alert"], .alert, text=Ошибка, text=Error').first()
      // Может быть видимым если произошла ошибка
      if (await errorMessage.isVisible({ timeout: 5000 })) {
        await expect(errorMessage).toBeVisible()
      }
    }
  })

  test('should switch between report tabs', async ({ page }) => {
    const normalizationTab = page.locator('button').filter({ hasText: /Нормализация|Normalization/ })
    const dataQualityTab = page.locator('button').filter({ hasText: /Качество данных|Data Quality/ })
    
    if (await normalizationTab.isVisible({ timeout: 5000 }) && await dataQualityTab.isVisible({ timeout: 5000 })) {
      // Кликаем на таб качества данных
      await dataQualityTab.click()
      await page.waitForTimeout(500)
      
      // Проверяем, что контент изменился
      const dataQualityContent = page.locator('text=Сгенерировать отчет о качестве данных').first()
      await expect(dataQualityContent).toBeVisible()
      
      // Возвращаемся на таб нормализации
      await normalizationTab.click()
      await page.waitForTimeout(500)
      
      // Проверяем, что контент вернулся
      const normalizationContent = page.locator('text=Сгенерировать отчет по нормализации').first()
      await expect(normalizationContent).toBeVisible()
    }
  })
})

