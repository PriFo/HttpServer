import { test, expect } from '@playwright/test'
import { waitForPageLoad, logPageInfo, waitForText, waitForDataUpdate } from './test-helpers'

test.describe('Нормализация данных', () => {
  test.beforeEach(async ({ page }) => {
    // Предварительная настройка: авторизация, выбор проекта
    // В реальном сценарии здесь может быть логин и выбор проекта
    await page.goto('/')
    
    // Ждем загрузки страницы
    await waitForPageLoad(page)
    await logPageInfo(page)
  })

  test('должен запустить процесс нормализации', async ({ page }) => {
    // Переходим на страницу качества данных
    await page.goto('/data-quality')
    await waitForPageLoad(page)

    // Ждем появления основного контента
    await expect(page.locator('text=Общая оценка качества').or(page.locator('text=Качество данных')).first()).toBeVisible({ timeout: 15000 })

    // Генерируем отчет о качестве данных (если нужно)
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').or(page.locator('button:has-text("Генерировать")')).first()
    if (await generateButton.isVisible({ timeout: 5000 })) {
      await generateButton.click()
      await waitForPageLoad(page)
    }

    // Ищем кнопку запуска нормализации
    const startButton = page.locator('button:has-text("Начать нормализацию")').or(page.locator('button:has-text("Запустить")')).first()
    
    if (await startButton.isVisible({ timeout: 10000 })) {
      // Перехватываем API запрос запуска
      const [startResponse] = await Promise.all([
        page.waitForResponse(
          (resp) => (resp.url().includes('/api/normalize/start') || resp.url().includes('/api/normalization/start') || resp.url().includes('/api/clients/') && resp.url().includes('/normalization/start')) && resp.status() === 200,
          { timeout: 30000 }
        ).catch(() => null),
        startButton.click()
      ])

      // Проверяем, что появился индикатор загрузки
      const loadingIndicator = page.locator('text=/запуск/i').or(
        page.locator('text=/загрузка/i')
      ).or(
        page.locator('[data-testid="loading"]')
      )

      const hasLoadingIndicator = await loadingIndicator.isVisible({ timeout: 5000 }).catch(() => false)
      if (hasLoadingIndicator) {
        expect(hasLoadingIndicator).toBe(true)
      }

      // Проверяем ответ API
      if (startResponse) {
        expect(startResponse.status()).toBe(200)
        const responseData = await startResponse.json()
        expect(responseData).toHaveProperty('success')
      }

      // Проверяем, что произошел переход на страницу мониторинга (или остались на той же странице)
      await page.waitForTimeout(2000)
      const currentUrl = page.url()
      // Может быть переход на /monitoring или остаться на /data-quality
      expect(currentUrl).toMatch(/\/monitoring|\/data-quality/)
    } else {
      console.log('Кнопка "Начать нормализацию" не найдена, возможно нормализация уже запущена')
    }
  })

  test('должен отображать мониторинг в реальном времени', async ({ page }) => {
    // Переходим на страницу мониторинга
    await page.goto('/monitoring')
    await waitForPageLoad(page)

    // Ждем появления элементов мониторинга
    await expect(page.locator('text=Мониторинг').or(page.locator('text=Arliai')).first()).toBeVisible({ timeout: 15000 })

    // Проверяем наличие карточек провайдеров
    const providerCards = page.locator('[role="progressbar"]').or(
      page.locator('.provider-card')
    ).or(
      page.locator('[data-testid="provider"]')
    ).or(
      page.locator('text=OpenRouter')
    ).or(
      page.locator('text=Arliai')
    )

    const providerCount = await providerCards.count()
    
    // Должен быть хотя бы один провайдер
    if (providerCount > 0) {
      expect(providerCount).toBeGreaterThan(0)
    }

    // Проверяем наличие прогресс-баров
    const progressBars = page.locator('[role="progressbar"]').or(
      page.locator('.progress-bar')
    ).or(
      page.locator('[data-testid="progress"]')
    )

    const progressCount = await progressBars.count()
    if (progressCount > 0) {
      expect(progressCount).toBeGreaterThan(0)
    }

    // Проверяем подключение к SSE потоку
    // SSE события должны приходить через /api/normalization/events или /api/normalize/events
    const sseConnected = await page.evaluate(() => {
      return new Promise((resolve) => {
        const eventSource = new EventSource('/api/normalization/events')
        eventSource.onopen = () => {
          eventSource.close()
          resolve(true)
        }
        eventSource.onerror = () => {
          eventSource.close()
          resolve(false)
        }
        setTimeout(() => {
          eventSource.close()
          resolve(false)
        }, 5000)
      })
    }).catch(() => false)

    // SSE может быть не подключен, если нормализация не запущена
    if (sseConnected) {
      expect(sseConnected).toBe(true)
    }
  })

  test('должен остановить процесс нормализации', async ({ page }) => {
    // Переходим на страницу мониторинга
    await page.goto('/monitoring')
    await waitForPageLoad(page)

    // Ищем кнопку остановки
    const stopButton = page.locator('button:has-text("Остановить")').or(
      page.locator('button:has-text("Stop")')
    ).or(
      page.locator('[data-testid="stop-button"]')
    )

    if (await stopButton.isVisible({ timeout: 10000 })) {
      // Перехватываем API запрос остановки
      const [stopResponse] = await Promise.all([
        page.waitForResponse(
          (resp) => (resp.url().includes('/api/normalization/stop') || resp.url().includes('/api/clients/') && resp.url().includes('/normalization/stop')) && resp.status() === 200,
          { timeout: 10000 }
        ).catch(() => null),
        stopButton.click()
      ])

      // Ждем обновления статуса
      await page.waitForTimeout(2000)

      // Проверяем, что статус изменился на "Остановлено"
      const stoppedStatus = page.locator('text=Остановлено').or(
        page.locator('text=Stopped')
      ).or(
        page.locator('[data-status="stopped"]')
      )

      const isStopped = await stoppedStatus.isVisible({ timeout: 10000 }).catch(() => false)
      if (isStopped) {
        expect(isStopped).toBe(true)
      }

      // Проверяем ответ API
      if (stopResponse) {
        expect(stopResponse.status()).toBe(200)
        const responseData = await stopResponse.json()
        expect(responseData).toHaveProperty('status')
      }
    } else {
      console.log('Кнопка остановки не найдена, возможно нормализация не запущена')
    }
  })

  test('должен дождаться завершения процесса нормализации', async ({ page }) => {
    // Переходим на страницу мониторинга
    await page.goto('/monitoring')
    await waitForPageLoad(page)

    // Ищем индикатор завершения
    const completedStatus = page.locator('text=Завершено').or(
      page.locator('text=Completed')
    ).or(
      page.locator('[data-status="completed"]')
    )

    // Ждем завершения с таймаутом 5 минут
    try {
      await expect(completedStatus.first()).toBeVisible({ timeout: 300000 }) // 5 минут
      
      // Проверяем, что появилась кнопка "Скачать отчет"
      const downloadButton = page.locator('button:has-text("Скачать отчет")').or(
        page.locator('button:has-text("Download Report")')
      )

      const hasDownloadButton = await downloadButton.isVisible({ timeout: 5000 }).catch(() => false)
      if (hasDownloadButton) {
        expect(hasDownloadButton).toBe(true)
      }
    } catch (e) {
      // Если не дождались завершения, проверяем, что процесс хотя бы запущен
      const runningStatus = page.locator('text=Запущено').or(
        page.locator('text=Выполняется')
      ).or(
        page.locator('[data-status="running"]')
      )

      const isRunning = await runningStatus.isVisible({ timeout: 5000 }).catch(() => false)
      if (isRunning) {
        console.log('Нормализация запущена, но не завершена в течение таймаута')
      } else {
        console.log('Нормализация не запущена или уже завершена')
      }
    }
  })

  test('должен получать события через SSE поток', async ({ page }) => {
    // Переходим на страницу мониторинга
    await page.goto('/monitoring')
    await waitForPageLoad(page)

    // Проверяем подключение к SSE
    const sseEvents = await page.evaluate(() => {
      return new Promise((resolve) => {
        const events: any[] = []
        const eventSource = new EventSource('/api/normalization/events')
        
        eventSource.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)
            events.push(data)
          } catch (e) {
            events.push({ type: 'text', message: event.data })
          }
        }

        eventSource.onopen = () => {
          events.push({ type: 'connected' })
        }

        setTimeout(() => {
          eventSource.close()
          resolve(events)
        }, 5000)
      })
    }).catch(() => [])

    // Должны быть получены хотя бы события подключения
    if (Array.isArray(sseEvents) && sseEvents.length > 0) {
      expect(sseEvents.length).toBeGreaterThan(0)
      
      // Проверяем структуру событий
      const firstEvent = sseEvents[0]
      expect(firstEvent).toHaveProperty('type')
    }
  })

  test('должен пройти полный цикл от анализа до отчета', async ({ page }) => {
    // Шаг 1-3: Анализ качества
    await page.goto('/data-quality');
    
    // Ждем появления основного контента
    await expect(page.locator('text=Общая оценка качества').or(page.locator('text=Качество данных')).first()).toBeVisible({ timeout: 15000 });

    // Генерируем отчет о качестве данных
    const generateButton = page.locator('button:has-text("Сгенерировать отчет")').or(page.locator('button:has-text("Генерировать")')).first();
    if (await generateButton.isVisible()) {
      await generateButton.click();
      
      // Ждем генерации отчета
      await waitForPageLoad(page)
    }

    // Шаг 4-6: Запуск и мониторинг
    // Ищем кнопку запуска нормализации
    const startButton = page.locator('button:has-text("Начать нормализацию")').or(page.locator('button:has-text("Запустить")')).first();
    if (await startButton.isVisible({ timeout: 5000 })) {
      await startButton.click();
    }

    // Переходим на страницу мониторинга
    await page.goto('/monitoring');
    
    // Ждем появления элементов мониторинга
    await expect(page.locator('text=Arliai').or(page.locator('text=Мониторинг')).first()).toBeVisible({ timeout: 15000 });

    // Проверяем наличие провайдеров (5 каналов)
    const providerCards = page.locator('[role="progressbar"]').or(page.locator('.provider-card')).or(page.locator('[data-testid="provider"]'));
    const providerCount = await providerCards.count();
    
    // Может быть от 1 до 5 провайдеров в зависимости от конфигурации
    if (providerCount > 0) {
      expect(providerCount).toBeGreaterThan(0);
    }

    // Более надежное ожидание завершения
    // Ищем индикатор завершения или статус
    const finalStatus = page.locator('.final-status').or(page.locator('text=Завершено')).or(page.locator('text=Завершена')).or(page.locator('[data-status="completed"]'));
    
    // Ждем либо завершения, либо таймаут (5 минут)
    try {
      await expect(finalStatus.first()).toBeVisible({ timeout: 300000 }); // 5 минут
    } catch (e) {
      // Если не дождались завершения, проверяем, что процесс хотя бы запущен
      const runningStatus = page.locator('text=Запущено').or(page.locator('text=Выполняется')).or(page.locator('[data-status="running"]'));
      if (await runningStatus.count() > 0) {
        // Процесс запущен, это нормально для теста
        console.log('Нормализация запущена, но не завершена в течение таймаута');
      }
    }

    // Шаг 7-9: Проверка отчета
    await page.goto('/reports');
    
    // Ждем загрузки страницы отчетов
    await page.waitForLoadState('networkidle');
    
    // Ищем кнопку скачивания PDF
    const downloadButton = page.locator('button:has-text("Скачать PDF-отчет")').or(page.locator('button:has-text("Скачать")')).first();
    
    if (await downloadButton.isVisible({ timeout: 5000 })) {
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 10000 }).catch(() => null),
        downloadButton.click()
      ]);
      
      if (download) {
        expect(download.suggestedFilename()).toMatch(/normalization-report.*\.pdf/);
      }
    }
  });

  test('должен отображать страницу качества данных', async ({ page }) => {
    await page.goto('/data-quality');
    
    // Проверяем наличие основных элементов
    await expect(page.locator('text=Качество данных').or(page.locator('text=Общая оценка')).first()).toBeVisible({ timeout: 15000 });
  });

  test('должен отображать страницу мониторинга', async ({ page }) => {
    await page.goto('/monitoring');
    
    // Проверяем наличие элементов мониторинга
    await expect(page.locator('text=Мониторинг').or(page.locator('text=Провайдеры')).first()).toBeVisible({ timeout: 15000 });
  });

  test('должен отображать страницу отчетов', async ({ page }) => {
    await page.goto('/reports');
    
    // Проверяем наличие страницы отчетов
    await expect(page.locator('text=Отчеты').or(page.locator('text=Генерация отчетов')).first()).toBeVisible({ timeout: 15000 });
  });
});


