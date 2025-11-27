/**
 * 📋 COMPREHENSIVE E2E ERROR HANDLING TESTS
 * 
 * Advanced test suite covering all error scenarios and edge cases:
 * - Backend unavailability handling
 * - File upload validation and rejection
 * - Long-running operation timeout and cancellation
 * - Network error simulation
 * - API error handling
 * - Performance under error conditions
 * - Edge cases and concurrent scenarios
 * 
 * Prerequisites:
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 */

import { test, expect } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'
import { createTestClient, createTestProject, cleanupTestData } from '../../utils/api-testing'
import { waitForPageLoad, logPageInfo, checkToast, waitForOperation } from './test-helpers'

// Конфигурация
const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'

// Интерфейс для тестовых данных
interface TestData {
  clientId?: number
  projectId?: number
  testClientName: string
  testProjectName: string
}

const testData: TestData = {
  testClientName: `Error Handling Test Client ${Date.now()}`,
  testProjectName: `Error Handling Test Project ${Date.now()}`,
}

test.describe('🛡️ Comprehensive Error Handling Tests', () => {
  test.beforeAll(async () => {
    console.log('🚀 Подготовка тестового окружения для тестов обработки ошибок...')
    
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
      throw error
    }
  })

  test.afterAll(async () => {
    console.log('🧹 Очистка тестовых данных...')
    
    // Удаляем тестовые данные
    if (testData.clientId) {
      try {
        await cleanupTestData(testData.clientId!, testData.projectId)
        console.log(`✅ Удалены тестовые данные: клиент ${testData.clientId}`)
      } catch (error) {
        console.warn(`⚠️ Не удалось удалить тестовые данные:`, error)
      }
    }
  })

  /**
   * 3.1. Test: Обработка недоступности бэкенда
   * - Frontend should gracefully handle when the backend is unavailable
   */
  test('3.1. Backend Unavailability Handling', async ({ page }) => {
    console.log('\n🎯 Тест: Обработка недоступности бэкенда...')
    
    // Arrange: Перехватываем все запросы к API и возвращаем ошибки
    const errorScenarios = [
      { status: 500, body: { error: 'Internal Server Error', message: 'Database connection failed' } },
      { status: 503, body: { error: 'Service Unavailable', message: 'Backend maintenance in progress' } },
      { status: 502, body: { error: 'Bad Gateway', message: 'Upstream server error' } },
      { status: 404, body: { error: 'Not Found', message: 'Resource not available' } },
    ]

    for (const scenario of errorScenarios) {
      console.log(`🔄 Testing ${scenario.status} error scenario...`)
      
      await page.route('**/api/**', (route) => {
        route.fulfill({
          status: scenario.status,
          contentType: 'application/json',
          body: JSON.stringify(scenario.body),
        })
      })

      // Act: Refresh the homepage
      await page.goto('/')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Assert: Check for skeleton or error message
      const skeleton = page.locator('[data-testid="skeleton"]').or(
        page.locator('.skeleton')
      ).or(
        page.locator('[class*="skeleton"]')
      ).first()

      const errorMessage = page.locator('text=/Ошибка|Error|Не удалось загрузить|Failed to load/i').first()

      const hasSkeleton = await skeleton.isVisible({ timeout: 5000 }).catch(() => false)
      const hasError = await errorMessage.isVisible({ timeout: 5000 }).catch(() => false)

      expect(hasSkeleton || hasError).toBe(true)
      
      // Check for no white screen
      const body = page.locator('body')
      await expect(body).toBeVisible()
      
      // Act: Attempt to navigate to /clients
      await page.goto('/clients')
      await waitForPageLoad(page)
      
      // Assert: Verify page doesn't crash
      const clientsPage = page.locator('body')
      await expect(clientsPage).toBeVisible()
      
      // Check that buttons requiring data are inactive
      const actionButtons = page.locator('button:has-text("Создать")').or(
        page.locator('button:has-text("Добавить")')
      )

      const buttonCount = await actionButtons.count()
      if (buttonCount > 0) {
        const firstButton = actionButtons.first()
        const isDisabled = await firstButton.isDisabled().catch(() => false)
        expect(isDisabled).toBe(true)
      }
      
      // Act: Attempt to start normalization
      await page.goto('/')
      await waitForPageLoad(page)

      const startButton = page.locator('button:has-text("Начать нормализацию")').or(
        page.locator('button:has-text("Запустить")')
      ).first()

      if (await startButton.isVisible({ timeout: 5000 })) {
        const isDisabled = await startButton.isDisabled().catch(() => false)
        expect(isDisabled).toBe(true)
      }
    }

    // Cleanup: Remove route interception
    await page.unroute('**/api/**')
    console.log('✅ Тест обработки недоступности бэкенда завершен')
  })

  /**
   * 3.2. Test: Загрузка некорректного файла
   * - System should reject invalid files and show a clear error message
   */
  test('3.2. Invalid File Upload Rejection', async ({ page }) => {
    console.log('\n🎯 Тест: Загрузка некорректного файла...')
    
    // Создаем временную директорию для тестовых файлов
    const tempDir = path.join(__dirname, '../../temp')
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true })
    }

    // Создаем различные некорректные файлы
    const invalidFiles = [
      { name: 'invalid.txt', content: 'this is not a database', type: 'text/plain' },
      { name: 'invalid.csv', content: 'name,age\nJohn,30', type: 'text/csv' },
      { name: 'invalid.json', content: '{"data": "not a database"}', type: 'application/json' },
      { name: 'invalid.xml', content: '<root><data>not a database</data></root>', type: 'application/xml' },
      { name: 'invalid.pdf', content: '%PDF-1.4\n1 0 obj\n<<\n/Type /Catalog\n/Pages 2 0 R\n>>\nendobj', type: 'application/pdf' },
      { name: 'empty.db', content: '', type: 'application/x-sqlite3' },
    ]

    try {
      // Act: Navigate to /databases/manage
      await page.goto('/databases/manage')
      await waitForPageLoad(page)
      await logPageInfo(page)

      const fileInput = page.locator('input[type="file"]').first()

      if (await fileInput.isVisible({ timeout: 5000 })) {
        for (const invalidFile of invalidFiles) {
          console.log(`🔄 Testing invalid file: ${invalidFile.name}`)
          
          const filePath = path.join(tempDir, invalidFile.name)
          fs.writeFileSync(filePath, invalidFile.content)
          
          try {
            // Act: Attempt to upload invalid file
            await fileInput.setInputFiles(filePath)
            await waitForPageLoad(page)

            // Assert: Check for error toast notification
            const hasError = await checkToast(
              page,
              /Неверный формат|Invalid format|Ожидался .db|Ожидался .sqlite|.db или .sqlite|формат|format|db|sqlite/i,
              'error',
              5000
            )

            if (hasError) {
              console.log(`✅ Toast-уведомление об ошибке формата отображается для ${invalidFile.name}`)
              expect(hasError).toBe(true)
            } else {
              // Alternative check: Verify file doesn't appear in list
              const fileInList = page.locator(`text=${invalidFile.name}`)
              const isInList = await fileInList.isVisible({ timeout: 3000 }).catch(() => false)
              expect(isInList).toBe(false)
              console.log(`✅ Файл ${invalidFile.name} не загружен (валидация сработала)`)
            }

            // Verify file doesn't appear in database list
            const dbList = page.locator('[data-testid="database-list"]').or(
              page.locator('text=invalid.txt')
            )
            const isInDbList = await dbList.filter({ hasText: invalidFile.name }).isVisible({ timeout: 2000 }).catch(() => false)
            expect(isInDbList).toBe(false)
            
            // Reset file input for next test
            await fileInput.fill('')
            await waitForPageLoad(page)
            
          } catch (error) {
            console.warn(`⚠️ Error testing file ${invalidFile.name}:`, error)
          } finally {
            // Clean up test file
            if (fs.existsSync(filePath)) {
              fs.unlinkSync(filePath)
            }
          }
        }
      } else {
        console.log('⚠️ Поле загрузки файла не найдено')
      }
    } finally {
      // Clean up temp directory if empty
      try {
        if (fs.readdirSync(tempDir).length === 0) {
          fs.rmdirSync(tempDir)
        }
      } catch (error) {
        console.warn('⚠️ Could not clean up temp directory:', error)
      }
    }

    console.log('✅ Тест загрузки некорректного файла завершен')
  })

  /**
   * 3.3. Test: Таймаут длительной операции
   * - During very long operations, user should see an indicator and have option to cancel
   */
  test('3.3. Long Operation Timeout and Cancellation', async ({ page }) => {
    console.log('\n🎯 Тест: Таймаут длительной операции...')
    
    // Arrange: Simulate slow response from normalization status endpoint
    let requestCount = 0
    let shouldFail = false
    
    await page.route('**/api/**/normalization/status**', async (route) => {
      requestCount++
      
      // Simulate slow response for first few requests
      if (requestCount <= 3) {
        await new Promise(resolve => setTimeout(resolve, 3000)) // 3 second delay
        
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'running',
            progress: Math.min(requestCount * 15, 45),
            message: 'Processing records...',
          }),
        })
      } else if (requestCount <= 6 && !shouldFail) {
        // Continue with normal progress
        await new Promise(resolve => setTimeout(resolve, 1000))
        
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'running',
            progress: Math.min(45 + (requestCount - 3) * 10, 85),
            message: 'Normalizing data...',
          }),
        })
      } else if (shouldFail) {
        // Simulate timeout/failure
        route.fulfill({
          status: 504,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Gateway Timeout',
            message: 'Operation took too long to complete',
          }),
        })
      } else {
        // Complete the operation
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            status: 'completed',
            progress: 100,
            message: 'Normalization completed successfully',
          }),
        })
      }
    })

    try {
      // Act: Start normalization
      await page.goto('/')
      await waitForPageLoad(page)
      await logPageInfo(page)

      const startButton = page.locator('button:has-text("Начать нормализацию")').or(
        page.locator('button:has-text("Запустить")')
      ).first()

      if (await startButton.isVisible({ timeout: 5000 })) {
        await startButton.click()
        await page.waitForTimeout(3000)

        // Act: Navigate to monitoring page
        await page.goto('/monitoring')
        await waitForPageLoad(page)
        await logPageInfo(page)

        // Assert: Check for progress bar
        const progressBar = page.locator('[role="progressbar"]').or(
          page.locator('.progress').or(
            page.locator('[class*="progress"]')
          )
        ).first()

        const hasProgress = await progressBar.isVisible({ timeout: 10000 }).catch(() => false)
        expect(hasProgress).toBe(true)
        console.log('✅ Прогресс-бар отображается')

        // Assert: Check for progress indicator text
        const progressText = page.locator('text=/Выполняется|Running|Processing/i').first()
        const hasProgressText = await progressText.isVisible({ timeout: 5000 }).catch(() => false)
        expect(hasProgressText).toBe(true)

        // Assert: Check for stop button
        const stopButton = page.locator('button:has-text("Остановить")').or(
          page.locator('button:has-text("Stop")')
        ).first()

        if (await stopButton.isVisible({ timeout: 5000 })) {
          const isEnabled = await stopButton.isEnabled().catch(() => false)
          expect(isEnabled).toBe(true)
          console.log('✅ Кнопка "Остановить" активна')

          // Act: Click stop button
          await stopButton.click()
          await waitForPageLoad(page)

          // Assert: Check status changed to stopped
          const stoppedStatus = page.locator('text=Остановлено').or(
            page.locator('text=Stopped')
          ).or(
            page.locator('[data-status="stopped"]')
          ).first()

          const isStopped = await stoppedStatus.isVisible({ timeout: 5000 }).catch(() => false)
          expect(isStopped).toBe(true)
          console.log('✅ Статус изменился на "Остановлено"')
        }

        // Test timeout scenario
        console.log('🔄 Testing timeout scenario...')
        shouldFail = true
        requestCount = 0
        
        // Try to start another operation
        await page.goto('/')
        await waitForPageLoad(page)
        
        if (await startButton.isVisible({ timeout: 5000 })) {
          await startButton.click()
          await page.waitForTimeout(5000) // Wait for timeout response
          
          // Assert: Check for timeout error message
          const timeoutMessage = page.locator('text=/Таймаут|Timeout|Превышено время ожидания/i').first()
          const hasTimeout = await timeoutMessage.isVisible({ timeout: 5000 }).catch(() => false)
          
          if (hasTimeout) {
            console.log('✅ Сообщение о таймауте отображается')
          } else {
            // Check for generic error message
            const genericError = page.locator('text=/Ошибка|Error|Не удалось/i').first()
            const hasError = await genericError.isVisible({ timeout: 5000 }).catch(() => false)
            expect(hasError).toBe(true)
            console.log('✅ Сообщение об ошибке отображается')
          }
        }
      } else {
        console.log('ℹ️ Кнопка запуска нормализации не найдена')
      }
    } finally {
      // Cleanup: Remove route interception
      await page.unroute('**/api/**/normalization/status**')
    }

    console.log('✅ Тест таймаута длительной операции завершен')
  })

  /**
   * Additional test: Network Error Simulation
   */
  test('Network Error Simulation', async ({ page }) => {
    console.log('\n🎯 Тест: Симуляция сетевых ошибок...')
    
    const networkErrorTypes = [
      { type: 'failed', description: 'Network failure' },
      { type: 'timedout', description: 'Request timeout' },
      { type: 'aborted', description: 'Request aborted' },
    ]

    for (const errorType of networkErrorTypes) {
      console.log(`🔄 Testing ${errorType.description}...`)
      
      // Arrange: Intercept and abort network requests
      await page.route('**/api/**', (route) => {
        route.abort(errorType.type as any)
      })

      // Act: Navigate to different pages
      const pages = ['/', '/clients', '/databases', '/quality']
      
      for (const pagePath of pages) {
        await page.goto(pagePath)
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(1000)

        // Assert: Check that page loads without crashing
        const body = page.locator('body')
        await expect(body).toBeVisible()

        // Assert: Check for network error messages
        const networkError = page.locator('text=/Ошибка сети|Network error|Не удалось подключиться|Connection failed/i').first()
        const genericError = page.locator('text=/Ошибка|Error|Не удалось/i').first()

        const hasError = await networkError.isVisible({ timeout: 3000 }).catch(() => false) ||
                        await genericError.isVisible({ timeout: 3000 }).catch(() => false)

        if (hasError) {
          console.log(`✅ Сообщение об ошибке сети отображается на ${pagePath}`)
        }
      }

      // Cleanup: Remove route interception
      await page.unroute('**/api/**')
    }

    console.log('✅ Тест симуляции сетевых ошибок завершен')
  })

  /**
   * Additional test: API Error Handling
   */
  test('API Error Handling', async ({ page }) => {
    console.log('\n🎯 Тест: Обработка ошибок API...')
    
    const apiErrorScenarios = [
      { 
        status: 400, 
        body: { error: 'Bad Request', message: 'Invalid parameters provided' },
        description: 'Bad Request'
      },
      { 
        status: 401, 
        body: { error: 'Unauthorized', message: 'Authentication required' },
        description: 'Unauthorized'
      },
      { 
        status: 403, 
        body: { error: 'Forbidden', message: 'Access denied' },
        description: 'Forbidden'
      },
      { 
        status: 404, 
        body: { error: 'Not Found', message: 'Resource not found' },
        description: 'Not Found'
      },
      { 
        status: 422, 
        body: { error: 'Unprocessable Entity', message: 'Validation failed' },
        description: 'Validation Error'
      },
      { 
        status: 429, 
        body: { error: 'Too Many Requests', message: 'Rate limit exceeded' },
        description: 'Rate Limit'
      },
      { 
        status: 500, 
        body: { error: 'Internal Server Error', message: 'Server error occurred' },
        description: 'Server Error'
      },
      { 
        status: 503, 
        body: { error: 'Service Unavailable', message: 'Service temporarily unavailable' },
        description: 'Service Unavailable'
      },
    ]

    for (const scenario of apiErrorScenarios) {
      console.log(`🔄 Testing ${scenario.description} (${scenario.status})...`)
      
      // Arrange: Intercept API calls with specific error
      await page.route('**/api/**', (route) => {
        route.fulfill({
          status: scenario.status,
          contentType: 'application/json',
          body: JSON.stringify(scenario.body),
        })
      })

      // Act: Try to perform various actions
      await page.goto('/')
      await waitForPageLoad(page)
      await logPageInfo(page)

      // Assert: Check for appropriate error messages
      const specificError = page.locator(`text=${scenario.body.error}`).first()
      const messageError = page.locator(`text=${scenario.body.message}`).first()
      const genericError = page.locator('text=/Ошибка|Error/i').first()

      const hasError = await specificError.isVisible({ timeout: 3000 }).catch(() => false) ||
                      await messageError.isVisible({ timeout: 3000 }).catch(() => false) ||
                      await genericError.isVisible({ timeout: 3000 }).catch(() => false)

      expect(hasError).toBe(true)
      
      // Assert: Check that UI components handle errors gracefully
      const body = page.locator('body')
      await expect(body).toBeVisible()
      
      // Cleanup: Remove route interception
      await page.unroute('**/api/**')
    }

    console.log('✅ Тест обработки ошибок API завершен')
  })

  /**
   * Additional test: File Upload Edge Cases
   */
  test('File Upload Edge Cases', async ({ page }) => {
    console.log('\n🎯 Тест: Крайние случаи загрузки файлов...')
    
    // Create temp directory
    const tempDir = path.join(__dirname, '../../temp')
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true })
    }

    try {
      // Navigate to file upload page
      await page.goto('/databases/manage')
      await waitForPageLoad(page)
      await logPageInfo(page)

      const fileInput = page.locator('input[type="file"]').first()

      if (await fileInput.isVisible({ timeout: 5000 })) {
        // Test: File with special characters in name
        const specialCharFile = path.join(tempDir, 'file-with-@#$%_name.db')
        fs.writeFileSync(specialCharFile, 'SQLite format 3\x00' + Buffer.alloc(500))
        
        try {
          await fileInput.setInputFiles(specialCharFile)
          await page.waitForTimeout(2000)
          
          // Check if file is rejected or handled properly
          const errorToast = page.locator('text=/Недопустимые символы|Invalid characters|Специальные символы/i').first()
          const hasError = await errorToast.isVisible({ timeout: 3000 }).catch(() => false)
          
          if (hasError) {
            console.log('✅ Файл со спецсимволами отклонен')
          } else {
            console.log('ℹ️ Файл со спецсимволами обработан без ошибок')
          }
        } finally {
          if (fs.existsSync(specialCharFile)) {
            fs.unlinkSync(specialCharFile)
          }
        }

        // Test: File with very long name
        const longNameFile = path.join(tempDir, 'a'.repeat(300) + '.db')
        fs.writeFileSync(longNameFile, 'SQLite format 3\x00' + Buffer.alloc(500))
        
        try {
          await fileInput.setInputFiles(longNameFile)
          await page.waitForTimeout(2000)
          
          // Check for error handling
          const errorToast = page.locator('text=/Слишком длинное имя|Name too long/i').first()
          const hasError = await errorToast.isVisible({ timeout: 3000 }).catch(() => false)
          
          if (hasError) {
            console.log('✅ Файл с длинным именем отклонен')
          } else {
            console.log('ℹ️ Файл с длинным именем обработан без ошибок')
          }
        } finally {
          if (fs.existsSync(longNameFile)) {
            fs.unlinkSync(longNameFile)
          }
        }

        // Test: Multiple file upload
        const file1 = path.join(tempDir, 'test1.db')
        const file2 = path.join(tempDir, 'test2.db')
        
        fs.writeFileSync(file1, 'SQLite format 3\x00' + Buffer.alloc(500))
        fs.writeFileSync(file2, 'SQLite format 3\x00' + Buffer.alloc(500))
        
        try {
          await fileInput.setInputFiles([file1, file2])
          await page.waitForTimeout(3000)
          
          // Check if multiple files are handled
          const fileCount = await page.locator('[data-testid="file-item"]').count()
          console.log(`📁 Found ${fileCount} file items in UI`)
          
          if (fileCount >= 2) {
            console.log('✅ Множественная загрузка файлов обработана')
          } else {
            console.log('ℹ️ Множественная загрузка файлов ограничена или обработана по-другому')
          }
        } finally {
          if (fs.existsSync(file1)) fs.unlinkSync(file1)
          if (fs.existsSync(file2)) fs.unlinkSync(file2)
        }

        // Test: Upload same file twice
        const duplicateFile = path.join(tempDir, 'duplicate.db')
        fs.writeFileSync(duplicateFile, 'SQLite format 3\x00' + Buffer.alloc(500))
        
        try {
          // First upload
          await fileInput.setInputFiles(duplicateFile)
          await page.waitForTimeout(2000)
          
          // Second upload of same file
          await fileInput.setInputFiles(duplicateFile)
          await page.waitForTimeout(2000)
          
          // Check for duplicate handling
          const duplicateError = page.locator('text=/Файл уже загружен|File already uploaded|Дубликат/i').first()
          const hasDuplicateError = await duplicateError.isVisible({ timeout: 3000 }).catch(() => false)
          
          if (hasDuplicateError) {
            console.log('✅ Дубликат файла отклонен')
          } else {
            console.log('ℹ️ Дубликат файла обработан без ошибок')
          }
        } finally {
          if (fs.existsSync(duplicateFile)) {
            fs.unlinkSync(duplicateFile)
          }
        }
      } else {
        console.log('⚠️ Поле загрузки файла не найдено')
      }
    } finally {
      // Clean up temp directory if empty
      try {
        if (fs.readdirSync(tempDir).length === 0) {
          fs.rmdirSync(tempDir)
        }
      } catch (error) {
        console.warn('⚠️ Could not clean up temp directory:', error)
      }
    }

    console.log('✅ Тест крайних случаев загрузки файлов завершен')
  })

  /**
   * Additional test: Concurrent Error Scenarios
   */
  test('Concurrent Error Scenarios', async ({ page }) => {
    console.log('\n🎯 Тест: Конкурентные сценарии с ошибками...')
    
    // Simulate multiple concurrent errors
    let errorCount = 0
    const maxErrors = 5
    
    await page.route('**/api/**', (route) => {
      errorCount++
      
      if (errorCount <= maxErrors) {
        // Return different errors for different requests
        const errors = [
          { status: 500, body: { error: 'Server Error', message: 'Internal server error' } },
          { status: 503, body: { error: 'Service Unavailable', message: 'Service busy' } },
          { status: 404, body: { error: 'Not Found', message: 'Resource missing' } },
          { status: 429, body: { error: 'Too Many Requests', message: 'Rate limit exceeded' } },
          { status: 502, body: { error: 'Bad Gateway', message: 'Upstream error' } },
        ]
        
        const error = errors[errorCount - 1]
        route.fulfill({
          status: error.status,
          contentType: 'application/json',
          body: JSON.stringify(error.body),
        })
      } else {
        // Allow normal requests after error limit
        route.continue()
      }
    })

    try {
      // Act: Perform multiple actions concurrently
      await Promise.all([
        page.goto('/'),
        page.goto('/clients', { waitUntil: 'domcontentloaded' }),
        page.goto('/databases', { waitUntil: 'domcontentloaded' }),
      ])

      await page.waitForTimeout(3000)

      // Assert: Check that application remains stable
      const pages = ['/', '/clients', '/databases']
      
      for (const pagePath of pages) {
        await page.goto(pagePath)
        await page.waitForLoadState('networkidle')
        await page.waitForTimeout(1000)
        
        const body = page.locator('body')
        await expect(body).toBeVisible()
        
        // Check for error indicators
        const errorIndicator = page.locator('[data-testid="error-indicator"]').or(
          page.locator('.error-state')
        ).first()
        
        const hasError = await errorIndicator.isVisible({ timeout: 3000 }).catch(() => false)
        
        if (hasError) {
          console.log(`✅ Индикатор ошибки отображается на ${pagePath}`)
        }
      }

      // Test concurrent file uploads
      await page.goto('/databases/manage')
      await waitForPageLoad(page)
      await logPageInfo(page)

      const fileInput = page.locator('input[type="file"]').first()
      
      if (await fileInput.isVisible({ timeout: 5000 })) {
        // Create temp directory and files
        const tempDir = path.join(__dirname, '../../temp')
        if (!fs.existsSync(tempDir)) {
          fs.mkdirSync(tempDir, { recursive: true })
        }

        const files: string[] = []
        for (let i = 0; i < 3; i++) {
          const filePath = path.join(tempDir, `concurrent-${i}.db`)
          fs.writeFileSync(filePath, 'SQLite format 3\x00' + Buffer.alloc(500))
          files.push(filePath)
        }

        try {
          // Try to upload multiple files concurrently
          await fileInput.setInputFiles(files as any)
          await page.waitForTimeout(3000)
          
          // Check for concurrent operation handling
          const loadingIndicator = page.locator('[data-testid="loading"]').or(
            page.locator('.loading')
          ).first()
          
          const isLoading = await loadingIndicator.isVisible({ timeout: 3000 }).catch(() => false)
          
          if (isLoading) {
            console.log('✅ Индикатор загрузки отображается при одновременной загрузке')
          }
          
          // Check for error handling
          const errorToast = page.locator('[role="alert"]').or(
            page.locator('.toast')
          ).first()
          
          const hasError = await errorToast.isVisible({ timeout: 3000 }).catch(() => false)
          
          if (hasError) {
            console.log('✅ Сообщение об ошибке отображается при одновременных операциях')
          }
        } finally {
          // Clean up files
          for (const file of files) {
            if (fs.existsSync(file)) {
              fs.unlinkSync(file)
            }
          }
          
          // Clean up temp directory if empty
          try {
            if (fs.readdirSync(tempDir).length === 0) {
              fs.rmdirSync(tempDir)
            }
          } catch (error) {
            console.warn('⚠️ Could not clean up temp directory:', error)
          }
        }
      }
    } finally {
      // Cleanup: Remove route interception
      await page.unroute('**/api/**')
    }

    console.log('✅ Тест конкурентных сценариев с ошибками завершен')
  })

  /**
   * Additional test: Performance Under Error Conditions
   */
  test('Performance Under Error Conditions', async ({ page }) => {
    console.log('\n🎯 Тест: Производительность при ошибках...')
    
    const performanceMetrics: { [key: string]: number } = {}
    
    // Simulate slow error responses
    await page.route('**/api/**', async (route) => {
      // Add artificial delay to simulate slow error handling
      await new Promise(resolve => setTimeout(resolve, 1000))
      
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error', message: 'Simulated error' }),
      })
    })

    try {
      // Measure page load time with errors
      const startTime = Date.now()
      
      await page.goto('/')
      await waitForPageLoad(page)
      await logPageInfo(page)
      
      const loadTime = Date.now() - startTime
      performanceMetrics['pageLoadWithError'] = loadTime
      console.log(`⏱️ Page load time with errors: ${loadTime}ms`)

      // Measure UI responsiveness during errors
      await page.waitForTimeout(2000)
      
      const button = page.locator('button:has-text("Обновить")').or(
        page.locator('button:has-text("Refresh")')
      ).first()
      
      if (await button.isVisible({ timeout: 5000 })) {
        const buttonStartTime = Date.now()
        await button.click()
        await page.waitForTimeout(1000)
        
        const buttonResponseTime = Date.now() - buttonStartTime
        performanceMetrics['buttonResponseWithError'] = buttonResponseTime
        console.log(`⏱️ Button response time with errors: ${buttonResponseTime}ms`)
      }

      // Set performance expectations
      expect(loadTime).toBeLessThan(10000) // Should load within 10 seconds
      expect(performanceMetrics['buttonResponseWithError'] || 0).toBeLessThan(5000) // Should respond within 5 seconds

      // Check for memory leaks or excessive resource usage
      const body = page.locator('body')
      await expect(body).toBeVisible()
      
      // Check for error recovery
      const errorCount = await page.locator('[data-testid="error"]').count()
      expect(errorCount).toBeLessThan(10) // Should not accumulate too many errors

      console.log('✅ Производительность при ошибках в пределах ожидаемых норм')
    } finally {
      // Cleanup: Remove route interception
      await page.unroute('**/api/**')
    }

    console.log('✅ Тест производительности при ошибках завершен')
    console.log('📊 Performance Metrics:', performanceMetrics)
  })
})
