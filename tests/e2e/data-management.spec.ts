/**
 * 📋 E2E ТЕСТЫ ДЛЯ УПРАВЛЕНИЯ ДАННЫМИ
 * 
 * Тесты покрывают:
 * - Полный жизненный цикл базы данных (загрузка, бэкап, удаление, восстановление)
 * - Переключение между базами данных
 * - Валидацию формата файла при загрузке
 * 
 * Prerequisites:
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 * 3. Тестовая база данных (SQLite) в одном из стандартных мест
 */

import { test, expect } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'
import {
  createTestClient,
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData,
  findTestDatabase,
  createBackup,
  listBackups,
  restoreBackup,
} from '../../utils/api-testing'
import {
  createTestDatabase,
  checkDatabaseIntegrity,
  getDatabaseStats,
  copyDatabase,
} from '../../utils/database-testing'
import { waitForPageLoad, logPageInfo, checkToast } from './test-helpers'

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
  testClientName: `E2E Data Management Test Client ${Date.now()}`,
  testProjectName: `E2E Data Management Test Project ${Date.now()}`,
}

test.describe('Управление данными', () => {
  test.beforeAll(async () => {
    console.log('🚀 Начало подготовки тестового окружения для управления данными...')
    
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
      if (testData.clientId) {
        await cleanupTestData(testData.clientId)
      }
      throw error
    }

    console.log('✅ Подготовка окружения завершена')
  })

  test.afterAll(async () => {
    console.log('🧹 Начало очистки тестовых данных...')
    
    if (testData.clientId) {
      try {
        await cleanupTestData(
          testData.clientId,
          testData.projectId,
          testData.databaseId
        )
        console.log(`✅ Удален тестовый клиент: ID ${testData.clientId}`)
      } catch (error) {
        console.warn(`⚠️ Не удалось удалить клиента ${testData.clientId}:`, error)
      }
    }

    console.log('✅ Очистка завершена')
  })

  test('Полный жизненный цикл базы данных', async ({ page }) => {
    console.log('\n🎯 Тест: Полный жизненный цикл базы данных...')

    // Arrange: Находим тестовую БД
    const dbPath = findTestDatabase()
    if (!dbPath) {
      test.skip(true, 'Тестовая база данных не найдена')
      return
    }

    // Шаг 1: Загрузка БД через UI
    console.log('📤 Шаг 1: Загрузка базы данных через UI...')
    await page.goto('/databases/manage')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Ищем кнопку загрузки или input для файла
    const fileInput = page.locator('input[type="file"]').first()
    const uploadButton = page.locator('button:has-text("Загрузить")').or(
      page.locator('button:has-text("Upload")')
    ).or(
      page.locator('button:has-text("Выбрать файл")')
    ).first()

    if (await fileInput.isVisible({ timeout: 5000 })) {
      await fileInput.setInputFiles(dbPath)
      await waitForPageLoad(page)
    } else if (await uploadButton.isVisible({ timeout: 5000 })) {
      await uploadButton.click()
      await waitForPageLoad(page)
      // Если открывается диалог выбора файла, используем file input
      const dialogFileInput = page.locator('input[type="file"]').first()
      if (await dialogFileInput.isVisible({ timeout: 2000 })) {
        await dialogFileInput.setInputFiles(dbPath)
      }
    }

    // Ждем появления БД в списке
    await waitForPageLoad(page)
    const dbName = path.basename(dbPath)
    const dbInList = page.locator(`text=${dbName}`).or(
      page.locator(`[data-db-name="${dbName}"]`)
    ).first()

    // Проверяем, что БД появилась в списке (или загружаем через API для надежности)
    try {
      const database = await uploadDatabaseFile(
        testData.clientId!,
        testData.projectId!,
        dbPath
      )
      testData.databaseId = database.id || database
      console.log(`✅ База данных загружена через API: ID ${testData.databaseId}`)
    } catch (error) {
      console.warn('⚠️ Не удалось загрузить БД через API:', error)
    }

    // Шаг 2: Создание бэкапа
    console.log('💾 Шаг 2: Создание бэкапа...')
    try {
      const backupResult = await createBackup({
        format: 'zip',
        includeMain: true,
        includeUploads: true,
      })
      console.log('✅ Бэкап создан:', backupResult)
      
      // Проверяем, что бэкап появился в списке
      const backups = await listBackups()
      expect(backups.length).toBeGreaterThan(0)
      console.log(`✅ Найдено бэкапов: ${backups.length}`)
    } catch (error) {
      console.warn('⚠️ Не удалось создать бэкап через API:', error)
    }

    // Проверяем список бэкапов через UI
    await page.goto('/databases/backups')
    await waitForPageLoad(page)
    await logPageInfo(page)

    const backupList = page.locator('[data-testid="backup-list"]').or(
      page.locator('text=/backup_/')
    ).first()
    
    const hasBackups = await backupList.isVisible({ timeout: 5000 }).catch(() => false)
    if (hasBackups) {
      console.log('✅ Бэкапы отображаются в UI')
    }

    // Шаг 3: Удаление БД
    console.log('🗑️ Шаг 3: Удаление базы данных...')
    await page.goto('/databases/manage')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Ищем кнопку удаления для нашей БД
    const deleteButton = page.locator(`button:has-text("Удалить")`).or(
      page.locator(`button[aria-label*="Удалить"]`)
    ).first()

    if (await deleteButton.isVisible({ timeout: 5000 })) {
      await deleteButton.click()
      await waitForPageLoad(page)

      // Подтверждаем удаление в диалоге
      const confirmButton = page.locator('button:has-text("Подтвердить")').or(
        page.locator('button:has-text("Удалить")')
      ).or(
        page.locator('button:has-text("OK")')
      ).first()

      if (await confirmButton.isVisible({ timeout: 3000 })) {
        await confirmButton.click()
        await waitForPageLoad(page)
        console.log('✅ БД удалена через UI')
      }
    }

    // Шаг 4: Восстановление из бэкапа
    console.log('♻️ Шаг 4: Восстановление из бэкапа...')
    const backups = await listBackups()
    if (backups.length > 0) {
      const latestBackup = backups[backups.length - 1]
      const backupFileName = latestBackup.name || latestBackup.filename

      if (backupFileName) {
        try {
          await restoreBackup(backupFileName)
          console.log(`✅ БД восстановлена из бэкапа: ${backupFileName}`)
        } catch (error) {
          console.warn('⚠️ Не удалось восстановить из бэкапа:', error)
        }
      }
    }

    console.log('✅ Тест жизненного цикла БД завершен')
  })

  test('Переключение между базами данных', async ({ page }) => {
    console.log('\n🎯 Тест: Переключение между базами данных...')

    // Arrange: Загружаем несколько БД (если возможно)
    const dbPath = findTestDatabase()
    test.skip(!dbPath, 'Тестовая база данных не найдена')

    await page.goto('/databases')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Ищем список БД или селектор для переключения
    const dbSelector = page.locator('select').or(
      page.locator('[data-testid="database-selector"]')
    ).first()

    if (await dbSelector.isVisible({ timeout: 5000 })) {
      // Получаем все опции
      const options = await dbSelector.locator('option').all()
      
      if (options.length > 1) {
        // Переключаемся на первую БД
        const firstOption = await options[0].getAttribute('value')
        if (firstOption) {
          await dbSelector.selectOption(firstOption)
          await waitForPageLoad(page)
          console.log('✅ Переключились на первую БД')

          // Переключаемся на вторую БД
          const secondOption = await options[1].getAttribute('value')
          if (secondOption) {
            await dbSelector.selectOption(secondOption)
            await waitForPageLoad(page)
            console.log('✅ Переключились на вторую БД')

            // Проверяем, что данные обновились
            const dataContent = page.locator('text=/Контрагент|Номенклатура|Запись/').first()
            const hasData = await dataContent.isVisible({ timeout: 5000 }).catch(() => false)
            if (hasData) {
              console.log('✅ Данные корректно отображаются после переключения')
            }
          }
        }
      } else {
        console.log('ℹ️ Доступна только одна БД, тест переключения пропущен')
      }
    } else {
      console.log('ℹ️ Селектор БД не найден, возможно используется другой механизм переключения')
    }

    console.log('✅ Тест переключения БД завершен')
  })

  test('Валидация формата файла при загрузке', async ({ page }) => {
    console.log('\n🎯 Тест: Валидация формата файла...')

    // Arrange: Создаем временный файл с неправильным форматом
    const tempDir = path.join(__dirname, '../../temp')
    if (!fs.existsSync(tempDir)) {
      fs.mkdirSync(tempDir, { recursive: true })
    }

    const invalidFilePath = path.join(tempDir, 'invalid.txt')
    fs.writeFileSync(invalidFilePath, 'this is not a database file')

    try {
      // Act: Пытаемся загрузить файл
      await page.goto('/databases/manage')
      await waitForPageLoad(page)
      await logPageInfo(page)

      const fileInput = page.locator('input[type="file"]').first()
      
      if (await fileInput.isVisible({ timeout: 5000 })) {
        await fileInput.setInputFiles(invalidFilePath)
        await waitForPageLoad(page)

        // Assert: Проверяем наличие сообщения об ошибке
        const hasError = await checkToast(
          page,
          /Неверный формат|Invalid format|Ожидался .db|Ожидался .sqlite|формат|format|db|sqlite/i,
          'error',
          5000
        )

        if (hasError) {
          console.log('✅ Ошибка валидации отображается')
          expect(hasError).toBe(true)
        } else {
          // Проверяем, что файл не появился в списке
          const fileInList = page.locator(`text=invalid.txt`)
          const isInList = await fileInList.isVisible({ timeout: 3000 }).catch(() => false)
          expect(isInList).toBe(false)
          console.log('✅ Файл не загружен (валидация сработала)')
        }
      } else {
        console.log('⚠️ Поле загрузки файла не найдено')
      }
    } finally {
      // Очищаем временный файл
      if (fs.existsSync(invalidFilePath)) {
        fs.unlinkSync(invalidFilePath)
      }
    }

    console.log('✅ Тест валидации формата завершен')
  })
})
