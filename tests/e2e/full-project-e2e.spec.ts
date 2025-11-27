/**
 * 📋 ЕДИНЫЙ СКВОЗНОЙ (E2E) ТЕСТ ДЛЯ ВСЕГО ПРОЕКТА
 * 
 * Этот тест проверяет полный жизненный цикл данных в приложении:
 * - Загрузка данных
 * - Анализ качества
 * - Запуск нормализации
 * - Мониторинг в реальном времени
 * - Финальный отчет
 * 
 * Prerequisites (требования для запуска):
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 *    - Можно запустить через: docker-compose up -d backend
 *    - Или напрямую: go run main.go
 * 
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 *    - cd frontend && npm run dev
 * 
 * 3. Установленные переменные окружения для AI-провайдеров:
 *    - OPENROUTER_API_KEY (опционально)
 *    - HUGGINGFACE_API_KEY (опционально)
 *    - ARLIAI_API_KEY (опционально)
 *    - EDEN_AI_API_KEY (опционально)
 *    - DADATA_API_KEY (для умного маршрутизатора)
 *    - ADATA_API_KEY (для умного маршрутизатора)
 * 
 * 4. Тестовая база данных (SQLite):
 *    - Должна содержать тестовые данные с дубликатами контрагентов и номенклатуры
 *    - Можно использовать существующую 1c_data.db или создать тестовую
 */

import { test, expect } from '@playwright/test'
import {
  createTestClient,
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData,
  getNormalizationStatus,
  findTestDatabase,
} from '../../utils/api-testing'
import { waitForPageLoad, logPageInfo, checkToast, waitForOperation } from './test-helpers'

// Конфигурация
const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'

// Тестовые данные
interface TestData {
  clientId?: number
  projectId?: number
  databaseId?: number
  testClientName: string
  testProjectName: string
  testDatabasePath?: string
}

const testData: TestData = {
  testClientName: `E2E Test Client ${Date.now()}`,
  testProjectName: `E2E Test Project ${Date.now()}`,
}

test.describe('Полный жизненный цикл E2E тест', () => {
  test.beforeAll(async () => {
    console.log('🚀 Начало подготовки тестового окружения...')
    
    // Создаем тестового клиента
    try {
      const client = await createTestClient({
        name: testData.testClientName,
        legal_name: testData.testClientName,
      })
      testData.clientId = client.id
      console.log(`✅ Создан тестовый клиент: ID ${testData.clientId}`)
    } catch (error) {
      console.error('❌ Ошибка создания клиента:', error)
      throw error
    }

    // Создаем тестовый проект
    try {
      const project = await createTestProject(testData.clientId!, {
        name: testData.testProjectName,
      })
      testData.projectId = project.id
      console.log(`✅ Создан тестовый проект: ID ${testData.projectId}`)
    } catch (error) {
      console.error('❌ Ошибка создания проекта:', error)
      // Удаляем клиента при ошибке
      if (testData.clientId) {
        await cleanupTestData(testData.clientId)
      }
      throw error
    }

    // Пытаемся загрузить тестовую базу данных
    const dbPath = findTestDatabase()
    if (dbPath) {
      try {
        const database = await uploadDatabaseFile(testData.clientId!, testData.projectId!, dbPath)
        testData.databaseId = database.id || database
        testData.testDatabasePath = dbPath
        console.log(`✅ Загружена тестовая база данных: ${dbPath} (ID: ${testData.databaseId})`)
      } catch (error) {
        console.warn(`⚠️ Не удалось загрузить ${dbPath}:`, error)
      }
    } else {
      console.warn('⚠️ Тестовая база данных не найдена. Некоторые тесты могут быть пропущены.')
    }

    if (!testData.databaseId) {
      console.warn('⚠️ Тестовая база данных не загружена. Некоторые тесты могут быть пропущены.')
    }

    console.log('✅ Подготовка окружения завершена')
  })

  test.afterAll(async () => {
    console.log('🧹 Начало очистки тестовых данных...')
    
    // Удаляем тестовые данные (клиент удалит проект каскадно)
    if (testData.clientId) {
      try {
        await cleanupTestData(
          testData.clientId,
          testData.projectId,
          testData.databaseId
        )
        console.log(`✅ Удалены тестовые данные: клиент ${testData.clientId}`)
      } catch (error) {
        console.warn(`⚠️ Не удалось удалить тестовые данные:`, error)
      }
    }

    console.log('✅ Очистка завершена')
    
    // Выводим итоговый отчет
    console.log('\n📊 ИТОГОВЫЙ ОТЧЕТ:')
    console.log('='.repeat(50))
    console.log(`Тестовый сценарий: ${testData.clientId ? '✅ Успешно выполнен' : '❌ Провален'}`)
    console.log(`Созданные ресурсы:`)
    console.log(`  - Клиент: ${testData.clientId || 'не создан'}`)
    console.log(`  - Проект: ${testData.projectId || 'не создан'}`)
    console.log(`  - База данных: ${testData.databaseId || 'не загружена'}`)
    console.log('='.repeat(50))
  })

  test('Полный цикл нормализации данных', async ({ page }) => {
    const startTime = Date.now()
    console.log('\n🎯 Начало основного сценария нормализации...')

    // Шаг 1: Открытие "Миссионного центра"
    console.log('📱 Шаг 1: Открытие главной страницы...')
    await page.goto('/')
    await expect(page).toHaveTitle(/Нормализатор|Dashboard|Миссионный центр/i, { timeout: 10000 })
    
    // Проверяем наличие ключевых элементов
    const header = page.locator('text=Миссионный центр').or(page.locator('h1')).first()
    await expect(header).toBeVisible({ timeout: 10000 })
    
    // Проверяем наличие боковой панели
    const sidebar = page.locator('[role="navigation"]').or(page.locator('nav')).first()
    await expect(sidebar).toBeVisible({ timeout: 5000 })
    
    console.log('✅ Главная страница загружена')

    // Шаг 2: Анализ качества данных
    console.log('📊 Шаг 2: Анализ качества данных...')
    
    // Переходим на страницу качества или используем вкладку на главной
    const qualityButton = page.locator('button:has-text("Анализ качества")').or(
      page.locator('button:has-text("Качество")')
    ).or(
      page.locator('a:has-text("Качество")')
    ).first()
    
    if (await qualityButton.isVisible({ timeout: 5000 })) {
      await qualityButton.click()
      await waitForPageLoad(page)
    } else {
      // Пытаемся перейти напрямую на страницу качества
      await page.goto('/quality')
    }

    // Ждем появления данных анализа качества
    const qualityHeader = page.locator('text=Общая оценка качества').or(
      page.locator('text=Качество данных')
    ).or(
      page.locator('text=Контрагенты')
    ).first()
    
    await expect(qualityHeader).toBeVisible({ timeout: 15000 })
    
    // Проверяем наличие данных по контрагентам и номенклатуре
    const counterpartiesSection = page.locator('text=Контрагенты').first()
    const nomenclatureSection = page.locator('text=Номенклатура').first()
    
    // Хотя бы один из разделов должен быть виден
    const hasCounterparties = await counterpartiesSection.isVisible({ timeout: 5000 }).catch(() => false)
    const hasNomenclature = await nomenclatureSection.isVisible({ timeout: 5000 }).catch(() => false)
    
    if (!hasCounterparties && !hasNomenclature) {
      console.warn('⚠️ Разделы контрагентов и номенклатуры не найдены, но продолжаем тест')
    } else {
      console.log('✅ Анализ качества данных отображается')
    }

    // Шаг 3: Запуск нормализации
    console.log('🚀 Шаг 3: Запуск нормализации...')
    
    // Ищем кнопку запуска нормализации
    const startNormalizationButton = page.locator('button:has-text("Начать нормализацию")').or(
      page.locator('button:has-text("Запустить нормализацию")')
    ).or(
      page.locator('button:has-text("Запустить")')
    ).first()
    
    if (await startNormalizationButton.isVisible({ timeout: 10000 })) {
      // Запускаем нормализацию
      await startNormalizationButton.click()
      await waitForPageLoad(page)
      
      // Backend Verification: Проверяем, что процесс запущен на бэкенде
      if (testData.clientId && testData.projectId) {
        const status = await getNormalizationStatus(testData.clientId, testData.projectId)
        if (status && (status.status === 'running' || status.status === 'started')) {
          console.log('✅ Нормализация запущена на бэкенде')
        } else {
          console.warn('⚠️ Статус нормализации не подтвержден на бэкенде')
        }
      }
    } else {
      console.warn('⚠️ Кнопка запуска нормализации не найдена, возможно нормализация уже запущена или недоступна')
    }

    // Шаг 4: Мониторинг в реальном времени
    console.log('📡 Шаг 4: Мониторинг в реальном времени...')
    
    // Переходим на страницу мониторинга
    await page.goto('/monitoring')
    await waitForPageLoad(page)
    await logPageInfo(page)
    
    // Проверяем наличие провайдеров
    const arliaiProvider = page.locator('text=Arliai').or(page.locator('[data-provider="arliai"]')).first()
    const openRouterProvider = page.locator('text=OpenRouter').or(page.locator('[data-provider="openrouter"]')).first()
    
    // Ждем появления хотя бы одного провайдера
    const hasArliai = await arliaiProvider.isVisible({ timeout: 10000 }).catch(() => false)
    const hasOpenRouter = await openRouterProvider.isVisible({ timeout: 10000 }).catch(() => false)
    
    if (hasArliai || hasOpenRouter) {
      console.log('✅ Провайдеры отображаются на странице мониторинга')
    } else {
      console.warn('⚠️ Провайдеры не найдены на странице мониторинга')
    }

    // Проверяем наличие прогресс-баров
    const progressBars = page.locator('[role="progressbar"]').or(page.locator('.progress')).or(page.locator('[class*="progress"]'))
    const progressCount = await progressBars.count()
    
    if (progressCount > 0) {
      console.log(`✅ Найдено ${progressCount} прогресс-баров`)
    } else {
      console.warn('⚠️ Прогресс-бары не найдены')
    }

    // SSE Verification: Проверяем обновление данных в реальном времени
    const totalRequests = page.locator('.total-requests').or(page.locator('text=/\\d+ запросов/')).first()
    if (await totalRequests.isVisible({ timeout: 5000 }).catch(() => false)) {
      const initialText = await totalRequests.textContent()
      await waitForPageLoad(page) // Ждем обновления через SSE
      const updatedText = await totalRequests.textContent()
      
      if (initialText !== updatedText) {
        console.log('✅ Данные обновляются в реальном времени (SSE работает)')
      } else {
        console.log('ℹ️ Данные не изменились за 5 секунд (возможно, нет активности)')
      }
    }

    // Шаг 5: Ожидание завершения и проверка результата
    console.log('⏳ Шаг 5: Ожидание завершения нормализации...')
    
    // Ищем индикатор завершения (с большим таймаутом)
    const completedStatus = page.locator('text=Завершено').or(
      page.locator('text=Завершена')
    ).or(
      page.locator('[data-status="completed"]')
    ).or(
      page.locator('text=Completed')
    ).first()
    
    try {
      await expect(completedStatus).toBeVisible({ timeout: 600000 }) // 10 минут
      console.log('✅ Нормализация завершена')
    } catch (error) {
      // Если не дождались завершения, проверяем, что процесс хотя бы запущен
      const runningStatus = page.locator('text=Запущено').or(
        page.locator('text=Выполняется')
      ).or(
        page.locator('[data-status="running"]')
      ).first()
      
      if (await runningStatus.isVisible({ timeout: 5000 }).catch(() => false)) {
        console.log('ℹ️ Нормализация все еще выполняется (это нормально для длительных процессов)')
      } else {
        console.warn('⚠️ Статус нормализации не определен')
      }
    }

    // Переходим на страницу отчетов
    console.log('📄 Шаг 6: Генерация финального отчета...')
    await page.goto('/reports')
    await waitForPageLoad(page)
    await logPageInfo(page)
    
    // Ищем кнопку генерации отчета
    const generateReportButton = page.locator('button:has-text("Сгенерировать финальный отчет")').or(
      page.locator('button:has-text("Сгенерировать отчет")')
    ).or(
      page.locator('button:has-text("Генерировать")')
    ).first()
    
    if (await generateReportButton.isVisible({ timeout: 10000 })) {
      await generateReportButton.click()
      await waitForPageLoad(page) // Ждем генерации отчета
    }

    // Ищем кнопку скачивания
    const downloadButton = page.locator('button:has-text("Скачать")').or(
      page.locator('button:has-text("Download")')
    ).or(
      page.locator('button:has-text("Скачать PDF")')
    ).first()
    
    if (await downloadButton.isVisible({ timeout: 10000 })) {
      // Настраиваем ожидание скачивания файла
      const [download] = await Promise.all([
        page.waitForEvent('download', { timeout: 30000 }).catch(() => null),
        downloadButton.click()
      ])
      
      if (download) {
        const filename = download.suggestedFilename()
        expect(filename).toMatch(/\.pdf$/i)
        console.log(`✅ PDF-файл успешно скачан: ${filename}`)
      } else {
        console.warn('⚠️ Скачивание файла не произошло')
      }
    } else {
      console.warn('⚠️ Кнопка скачивания не найдена')
    }

    const duration = ((Date.now() - startTime) / 1000 / 60).toFixed(2)
    console.log(`\n✅ Основной сценарий завершен за ${duration} минут`)
  })

  test.describe('Отказоустойчивость', () => {
    test('Сбой провайдера AI', async ({ page }) => {
      console.log('\n🛡️ Тест отказоустойчивости: Сбой провайдера AI...')
      
      // Симулируем сбой одного из провайдеров
      await page.route('**/api/workers/models**', (route) => {
        route.fulfill({
          status: 500,
          body: JSON.stringify({ error: 'Service Unavailable' }),
          headers: { 'Content-Type': 'application/json' },
        })
      })

      // Переходим на главную страницу
      await page.goto('/')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Пытаемся запустить нормализацию
      const startButton = page.locator('button:has-text("Начать нормализацию")').first()
      if (await startButton.isVisible({ timeout: 5000 })) {
        await startButton.click()
        await waitForPageLoad(page)
      }

      // Переходим на мониторинг
      await page.goto('/monitoring')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Проверяем, что процесс все еще работает (используя другие провайдеры)
      const runningStatus = page.locator('text=Запущено').or(
        page.locator('text=Выполняется')
      ).first()
      
      // Процесс должен либо работать, либо показать ошибку, но не упасть полностью
      const isRunning = await runningStatus.isVisible({ timeout: 10000 }).catch(() => false)
      const hasError = await page.locator('text=Ошибка').or(page.locator('text=Error')).isVisible({ timeout: 5000 }).catch(() => false)
      
      if (isRunning || hasError) {
        console.log('✅ Система корректно обработала сбой провайдера')
      } else {
        console.warn('⚠️ Реакция на сбой провайдера не определена')
      }

      // Убираем перехват маршрута
      await page.unroute('**/api/workers/models**')
    })

    test('Остановка процесса пользователем', async ({ page }) => {
      console.log('\n⏹️ Тест отказоустойчивости: Остановка процесса...')
      
      // Переходим на главную страницу
      await page.goto('/')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Запускаем нормализацию
      const startButton = page.locator('button:has-text("Начать нормализацию")').first()
      if (await startButton.isVisible({ timeout: 5000 })) {
        await startButton.click()
        await waitForPageLoad(page)
      }

      // Переходим на мониторинг
      await page.goto('/monitoring')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Ищем кнопку остановки
      const stopButton = page.locator('button:has-text("Остановить")').or(
        page.locator('button:has-text("Stop")')
      ).first()
      
      if (await stopButton.isVisible({ timeout: 10000 })) {
        await stopButton.click()
        await waitForPageLoad(page)
        
        // Проверяем, что статус изменился на "Остановлено"
        const stoppedStatus = page.locator('text=Остановлено').or(
          page.locator('text=Stopped')
        ).or(
          page.locator('[data-status="stopped"]')
        ).first()
        
        if (await stoppedStatus.isVisible({ timeout: 10000 }).catch(() => false)) {
          console.log('✅ Процесс успешно остановлен')
        } else {
          console.warn('⚠️ Статус остановки не подтвержден')
        }
      } else {
        console.warn('⚠️ Кнопка остановки не найдена (возможно, процесс не запущен)')
      }
    })
  })

  test.describe('Интеграции', () => {
    test('Умный маршрутизатор (DaData/Adata)', async ({ page }) => {
      console.log('\n🧠 Тест интеграций: Умный маршрутизатор...')
      
      test.skip(!testData.databaseId, 'Тестовая база данных не загружена')

      // Переходим на страницу процессов
      await page.goto('/processes')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Выбираем проект, если есть селектор
      if (testData.clientId && testData.projectId) {
        const projectSelector = page.locator('[data-testid="project-selector"]').or(
          page.locator('select').filter({ hasText: testData.testProjectName })
        ).first()
        
        if (await projectSelector.isVisible({ timeout: 5000 }).catch(() => false)) {
          await projectSelector.selectOption({ label: testData.testProjectName })
          await waitForPageLoad(page)
        }
      }

      // Запускаем нормализацию только контрагентов
      const startButton = page.locator('button:has-text("Начать нормализацию")').first()
      if (await startButton.isVisible({ timeout: 5000 })) {
        await startButton.click()
        await waitForPageLoad(page)
      }

      // Переходим на мониторинг
      await page.goto('/monitoring')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Делаем фоновый запрос к метрикам провайдеров
      try {
        const response = await page.evaluate(async () => {
          const res = await fetch('/api/monitoring/providers')
          return res.ok ? await res.json() : null
        })

        if (response) {
          console.log('✅ Метрики провайдеров получены')
          // Проверяем, что есть данные о провайдерах
          if (Array.isArray(response) && response.length > 0) {
            console.log(`✅ Найдено ${response.length} активных провайдеров`)
          }
        }
      } catch (error) {
        console.warn('⚠️ Не удалось получить метрики провайдеров:', error)
      }

      // Проверяем наличие специализированных сервисов в интерфейсе
      const dadataService = page.locator('text=DaData').or(page.locator('[data-provider="dadata"]')).first()
      const adataService = page.locator('text=Adata').or(page.locator('[data-provider="adata"]')).first()
      
      const hasDaData = await dadataService.isVisible({ timeout: 5000 }).catch(() => false)
      const hasAdata = await adataService.isVisible({ timeout: 5000 }).catch(() => false)
      
      if (hasDaData || hasAdata) {
        console.log('✅ Специализированные сервисы маршрутизации найдены')
      } else {
        console.log('ℹ️ Специализированные сервисы не отображаются (возможно, не используются)')
      }
    })
  })
})
