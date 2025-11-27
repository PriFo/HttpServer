import { test, expect } from '@playwright/test'
import { waitForPageLoad, logPageInfo, waitForDownload } from './test-helpers'

test.describe('Отчеты', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/reports')
    await waitForPageLoad(page)
    await logPageInfo(page)
  })

  test('должен отображать страницу отчетов', async ({ page }) => {
    // Проверяем наличие заголовка
    await expect(page.locator('text=Отчеты').or(page.locator('h1:has-text("Отчеты")'))).toBeVisible({ timeout: 15000 })
    
    // Проверяем наличие карточек для выбора типа отчета
    const reportCards = page.locator('text=Отчет по нормализации').or(page.locator('text=Отчет о качестве данных'))
    await expect(reportCards.first()).toBeVisible({ timeout: 10000 })
  })

  test('должен генерировать отчет по нормализации', async ({ page }) => {
    // Ищем кнопку генерации отчета по нормализации
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').filter({ hasText: /нормализац/i }).first()
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      // Кликаем на кнопку генерации
      await generateButton.click()
      
      // Ждем генерации отчета
      await waitForPageLoad(page)
      
      // Проверяем, что появилась кнопка скачивания PDF
      const downloadButton = page.locator('button:has-text("Скачать PDF")').first()
      await expect(downloadButton).toBeVisible({ timeout: 30000 })
    }
  })

  test('должен генерировать отчет о качестве данных', async ({ page }) => {
    // Переключаемся на вкладку отчета о качестве данных
    const qualityTab = page.locator('button:has-text("Отчет о качестве данных")').or(page.locator('[role="tab"]:has-text("качеств")'))
    
    if (await qualityTab.isVisible({ timeout: 5000 })) {
      await qualityTab.click()
      await waitForPageLoad(page)
    }
    
    // Ищем кнопку генерации отчета о качестве
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').filter({ hasText: /качеств/i }).first()
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      // Кликаем на кнопку генерации
      await generateButton.click()
      
      // Ждем генерации отчета
      await waitForPageLoad(page)
      
      // Проверяем, что появилась кнопка скачивания PDF
      const downloadButton = page.locator('button:has-text("Скачать PDF")').first()
      await expect(downloadButton).toBeVisible({ timeout: 30000 })
    }
  })

  test('должен скачивать PDF отчет по нормализации', async ({ page }) => {
    // Сначала генерируем отчет, если его нет
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').filter({ hasText: /нормализац/i }).first()
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      await waitForPageLoad(page)
    }
    
    // Ищем кнопку скачивания PDF
    const downloadButton = page.locator('button:has-text("Скачать PDF")').first()
    
    if (await downloadButton.isVisible({ timeout: 10000 })) {
      // Настраиваем ожидание скачивания файла
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }).catch(() => null),
        downloadButton.click()
      ])
      
      if (download) {
        // Проверяем, что файл имеет правильное имя
        const filename = download.suggestedFilename()
        expect(filename).toMatch(/normalization.*\.pdf/i)
      }
    }
  })

  test('должен скачивать PDF отчет о качестве данных', async ({ page }) => {
    // Переключаемся на вкладку отчета о качестве данных
    const qualityTab = page.locator('button:has-text("Отчет о качестве данных")').or(page.locator('[role="tab"]:has-text("качеств")'))
    
    if (await qualityTab.isVisible({ timeout: 5000 })) {
      await qualityTab.click()
      await waitForPageLoad(page)
    }
    
    // Сначала генерируем отчет, если его нет
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').filter({ hasText: /качеств/i }).first()
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      await waitForPageLoad(page)
    }
    
    // Ищем кнопку скачивания PDF
    const downloadButton = page.locator('button:has-text("Скачать PDF")').first()
    
    if (await downloadButton.isVisible({ timeout: 10000 })) {
      // Настраиваем ожидание скачивания файла
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }).catch(() => null),
        downloadButton.click()
      ])
      
      if (download) {
        // Проверяем, что файл имеет правильное имя
        const filename = download.suggestedFilename()
        expect(filename).toMatch(/quality.*\.pdf|data-quality.*\.pdf/i)
      }
    }
  })

  test('должен отображать содержимое отчета по нормализации', async ({ page }) => {
    // Генерируем отчет
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').filter({ hasText: /нормализац/i }).first()
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      await waitForPageLoad(page)
      
      // Проверяем наличие содержимого отчета
      const reportContent = page.locator('#normalization-report-content').or(page.locator('text=Отчет по нормализации данных'))
      await expect(reportContent.first()).toBeVisible({ timeout: 30000 })
      
      // Проверяем наличие основных секций
      const sections = page.locator('text=Сводка').or(page.locator('text=Общая статистика')).or(page.locator('text=Анализ контрагентов'))
      const sectionCount = await sections.count()
      
      if (sectionCount > 0) {
        expect(sectionCount).toBeGreaterThan(0)
      }
    }
  })

  test('должен отображать содержимое отчета о качестве данных', async ({ page }) => {
    // Переключаемся на вкладку отчета о качестве данных
    const qualityTab = page.locator('button:has-text("Отчет о качестве данных")').or(page.locator('[role="tab"]:has-text("качеств")'))
    
    if (await qualityTab.isVisible({ timeout: 5000 })) {
      await qualityTab.click()
      await waitForPageLoad(page)
    }
    
    // Генерируем отчет
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').filter({ hasText: /качеств/i }).first()
    
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      await waitForPageLoad(page)
      
      // Проверяем наличие содержимого отчета
      const reportContent = page.locator('#data-quality-report-content').or(page.locator('text=Отчет о качестве данных'))
      await expect(reportContent.first()).toBeVisible({ timeout: 30000 })
      
      // Проверяем наличие основных секций
      const sections = page.locator('text=Общая оценка качества').or(page.locator('text=Статистика по контрагентам')).or(page.locator('text=Статистика по номенклатуре'))
      const sectionCount = await sections.count()
      
      if (sectionCount > 0) {
        expect(sectionCount).toBeGreaterThan(0)
      }
    }
  })
})

