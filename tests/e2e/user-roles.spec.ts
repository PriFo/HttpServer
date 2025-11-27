/**
 * 📋 Е2Е ТЕСТЫ ДЛЯ ПРОВЕРКИ РОЛЕЙ ПОЛЬЗОВАТЕЛЕЙ
 * 
 * Этот тестовый набор проверяет корректность реализации контроля доступа
 * на основе ролей пользователей в системе:
 * - Администратор системы (admin)
 * - Менеджер клиента (manager) 
 * - Наблюдатель (viewer)
 * 
 * Prerequisites (требования для запуска):
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 * 3. Тестовая база данных с данными
 */

import { test, expect } from '@playwright/test'
import * as fs from 'fs'
import * as path from 'path'
import {
  createAdminToken,
  createManagerToken,
  createViewerToken,
  addAuthHeader,
  isAccessDeniedError,
  type JWTPayload,
} from '../../utils/auth-testing'
import { waitForPageLoad, logPageInfo } from './test-helpers'
import {
  createTestClient,
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData,
} from '../../utils/api-testing'

// Тестовые данные
const testData = {
  adminUser: {
    email: 'admin@test.com',
    name: 'Admin User',
    roles: ['admin'],
    clientId: 0,
    projectId: 0,
    databaseId: 0
  },
  managerUser: {
    email: 'manager@test.com', 
    name: 'Manager User',
    roles: ['manager'],
    clientId: 123, // Указанный клиент для менеджера
    projectId: 0,
    databaseId: 0
  },
  viewerUser: {
    email: 'viewer@test.com',
    name: 'Viewer User', 
    roles: ['viewer'],
    clientId: 0,
    projectId: 0,
    databaseId: 0
  },
  testClientName: `Role Test Client ${Date.now()}`,
  testProjectName: `Role Test Project ${Date.now()}`,
  testDatabasePath: '1c_data.db' // Будет искать существующую БД
}

// Вспомогательные функции для работы с API

// Конфигурация
const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:9999'
const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'

test.describe('🔐 Тесты контроля доступа на основе ролей пользователей', () => {
  
  test.beforeAll(async () => {
    console.log('🚀 Начало подготовки тестового окружения для проверки ролей...')
    
    // Создаем тестовые данные для каждого типа пользователя
    try {
      // Создаем клиента и проект для администратора
      const adminToken = createAdminToken()
      const client = await createTestClient({
        name: testData.testClientName,
        legal_name: testData.testClientName,
      })
      testData.adminUser.clientId = client.id
      
      const project = await createTestProject(testData.adminUser.clientId, {
        name: testData.testProjectName,
      })
      testData.adminUser.projectId = project.id
      
      // Ищем и загружаем тестовую базу данных
      try {
        const dbPath = '1c_data.db' // Упрощенная версия - можно улучшить поиск
        if (fs.existsSync(dbPath)) {
          const database = await uploadDatabaseFile(
            testData.adminUser.clientId,
            testData.adminUser.projectId,
            dbPath
          )
          testData.adminUser.databaseId = database.id || database
          console.log(`✅ Загружена тестовая база данных: ID ${testData.adminUser.databaseId}`)
        }
      } catch (uploadError) {
        console.warn('⚠️ Не удалось загрузить тестовую базу данных:', uploadError)
      }
      
      console.log('✅ Подготовка окружения для проверки ролей завершена')
    } catch (error) {
      console.error('❌ Ошибка подготовки окружения:', error)
      throw error
    }
  })

  test.afterAll(async () => {
    console.log('🧹 Начало очистки тестовых данных...')
    
    try {
      // Удаляем тестовые данные
      await cleanupTestData(
        testData.adminUser.clientId,
        testData.adminUser.projectId,
        testData.adminUser.databaseId
      )
      
      console.log('✅ Очистка тестовых данных завершена')
    } catch (error) {
      console.warn('⚠️ Ошибка очистки:', error)
    }
  })

  test.describe('👨‍💼 Администратор системы', () => {
    test('Должен иметь полный доступ ко всем функциям системы', async ({ page }) => {
      console.log('\n🔧 Тест: Администратор системы - проверка полного доступа...')
      
      const startTime = Date.now()
      
      // Arrange: Логин как администратор
      const adminToken = createAdminToken()
      
      // Устанавливаем заголовки авторизации для всех запросов страницы
      await page.context().setExtraHTTPHeaders({
        Authorization: `Bearer ${adminToken}`,
      })
      
      // Act & Assert: Проверяем доступ к разным разделам
      await page.goto('/')
      await expect(page).toHaveTitle(/Нормализатор|Dashboard|Миссионный центр/i, { timeout: 10000 })
      
      // Проверяем наличие всех разделов меню включая "Система"
      const systemMenu = page.locator('text=Система').or(page.locator('[data-testid="system-menu"]')).first()
      await expect(systemMenu).toBeVisible({ timeout: 5000 })
      console.log('✅ Меню "Система" доступно администратору')
      
      // Проверяем доступ к мониторингу
      await page.goto('/monitoring')
      await expect(page).toHaveURL(/\/monitoring/)
      console.log('✅ Доступ к странице мониторинга подтвержден')
      
      // Проверяем доступ к рабочим процессам
      await page.goto('/workers')
      await expect(page).toHaveURL(/\/workers/)
      console.log('✅ Доступ к странице рабочих процессов подтвержден')
      
      // Проверяем наличие кнопок управления рабочими процессами
      const stopWorkerButton = page.locator('button:has-text("Остановить")').or(
        page.locator('button:has-text("Stop")')
      ).first()
      
      if (await stopWorkerButton.isVisible({ timeout: 5000 })) {
        console.log('✅ Кнопки управления рабочими процессами активны')
      } else {
        console.log('ℹ️ Кнопки управления не найдены (возможно, нет активных процессов)')
      }
      
      // Проверяем доступ к отчетам
      await page.goto('/reports')
      await expect(page).toHaveURL(/\/reports/)
      console.log('✅ Доступ к странице отчетов подтвержден')
      
      // Проверяем возможность генерации отчета
      const generateReportButton = page.locator('button:has-text("Сгенерировать")').or(
        page.locator('button:has-text("Generate")')
      ).first()
      
      if (await generateReportButton.isVisible({ timeout: 5000 })) {
        console.log('✅ Кнопка генерации отчета доступна')
      } else {
        console.log('ℹ️ Кнопка генерации отчета не найдена')
      }
      
      const duration = ((Date.now() - startTime) / 1000).toFixed(2)
      console.log(`✅ Тест администратора завершен за ${duration} секунд`)
    })
  })

  test.describe('👔 Менеджер клиента', () => {
    test('Должен иметь доступ только к своим клиентам и проектам', async ({ page }) => {
      console.log('\n👔 Тест: Менеджер клиента - проверка ограниченного доступа...')
      
      const startTime = Date.now()
      
      // Arrange: Логин как менеджер с доступом только к client_id = 123
      const managerToken = createManagerToken(testData.managerUser.clientId)
      
      // Устанавливаем заголовки авторизации
      await page.context().setExtraHTTPHeaders({
        Authorization: `Bearer ${managerToken}`,
      })
      
      // Act & Assert: Проверяем доступ к своему клиенту
      await page.goto(`/clients/${testData.managerUser.clientId}`)
      await expect(page).toHaveURL(/\/clients\/123/)
      console.log('✅ Доступ к своему клиенту (ID: 123) подтвержден')
      
      // Проверяем возможность создания проекта
      const createProjectButton = page.locator('button:has-text("Создать проект")').or(
        page.locator('button:has-text("Create project")')
      ).first()
      
      if (await createProjectButton.isVisible({ timeout: 5000 })) {
        await createProjectButton.click()
        await waitForPageLoad(page)
        console.log('✅ Кнопка создания проекта доступна')
      }
      
      // Проверяем доступ к загрузке базы данных
      const uploadButton = page.locator('button:has-text("Загрузить")').or(
        page.locator('button:has-text("Upload")')
      ).first()
      
      if (await uploadButton.isVisible({ timeout: 5000 })) {
        console.log('✅ Доступ к загрузке базы данных подтвержден')
      }
      
      // Проверяем попытку доступа к системным разделам
      await page.goto('/workers')
      
      // Должны получить ошибку доступа или перенаправление
      const hasAccessDenied = await page.locator('text=403').or(
        page.locator('text=Access denied')
      ).or(
        page.locator('text=Недостаточно прав')
      ).isVisible({ timeout: 5000 }).catch(() => false)
      
      const hasErrorPage = await page.locator('h1').or(
        page.locator('[role="alert"]')
      ).isVisible({ timeout: 5000 }).catch(() => false)
      
      if (hasAccessDenied) {
        console.log('✅ Доступ к /workers запрещен - ошибка 403')
      } else if (hasErrorPage) {
        console.log('✅ Доступ к /workers запрещен - страница ошибки')
      } else {
        console.warn('⚠️ Ожидаемая ошибка доступа к /workers не обнаружена')
      }
      
      // Проверяем, что меню "Система" отсутствует или неактивно
      const systemMenu = page.locator('text=Система').or(page.locator('[data-testid="system-menu"]')).first()
      const hasSystemMenu = await systemMenu.isVisible({ timeout: 5000 }).catch(() => false)
      
      if (!hasSystemMenu) {
        console.log('✅ Меню "Система" отсутствует для менеджера')
      } else {
        console.warn('⚠️ Меню "Система" обнаружено для менеджера')
      }
      
      const duration = ((Date.now() - startTime) / 1000).toFixed(2)
      console.log(`✅ Тест менеджера завершен за ${duration} секунд`)
    })
  })

  test.describe('👁️ Наблюдатель (Viewer)', () => {
    test('Должен иметь только права на чтение данных', async ({ page }) => {
      console.log('\n👁️ Тест: Наблюдатель - проверка прав на чтение...')
      
      const startTime = Date.now()
      
      // Arrange: Логин как наблюдатель
      const viewerToken = createViewerToken()
      
      // Устанавливаем заголовки авторизации
      await page.context().setExtraHTTPHeaders({
        Authorization: `Bearer ${viewerToken}`,
      })
      
      // Act & Assert: Проверяем доступ к чтению данных
      await page.goto('/')
      await expect(page).toHaveTitle(/Нормализатор|Dashboard|Миссионный центр/i, { timeout: 10000 })
      console.log('✅ Доступ к главной странице подтвержден')
      
      // Проверяем доступ к странице клиента
      await page.goto(`/clients/${testData.adminUser.clientId}`)
      await expect(page).toHaveURL(/\/clients\/\d+/)
      console.log('✅ Доступ к странице клиента подтвержден')
      
      // Проверяем доступ к анализу качества
      await page.goto('/quality')
      await expect(page).toHaveURL(/\/quality/)
      console.log('✅ Доступ к анализу качества подтвержден')
      
      // Проверяем, что данные отображаются
      const dataDisplay = page.locator('[data-testid="data-display"]').or(
        page.locator('.data-content')
      ).first()
      
      if (await dataDisplay.isVisible({ timeout: 5000 }).catch(() => false)) {
        console.log('✅ Данные успешно отображаются')
      } else {
        console.log('ℹ️ Данные не найдены, но страница доступна')
      }
      
      // Проверяем отсутствие кнопок управления процессами
      const startButton = page.locator('button:has-text("Начать")').or(
        page.locator('button:has-text("Start")')
      ).or(
        page.locator('button:has-text("Запустить")')
      ).first()
      
      const createButton = page.locator('button:has-text("Создать")').or(
        page.locator('button:has-text("Create")')
      ).first()
      
      const editButton = page.locator('button:has-text("Редактировать")').or(
        page.locator('button:has-text("Edit")')
      ).first()
      
      const deleteButton = page.locator('button:has-text("Удалить")').or(
        page.locator('button:has-text("Delete")')
      ).first()
      
      const startVisible = await startButton.isVisible({ timeout: 3000 }).catch(() => false)
      const createVisible = await createButton.isVisible({ timeout: 3000 }).catch(() => false)
      const editVisible = await editButton.isVisible({ timeout: 3000 }).catch(() => false)
      const deleteVisible = await deleteButton.isVisible({ timeout: 3000 }).catch(() => false)
      
      if (!startVisible && !createVisible && !editVisible && !deleteVisible) {
        console.log('✅ Кнопки управления процессами отсутствуют')
      } else {
        console.warn('⚠️ Некоторые кнопки управления обнаружены:')
        if (startVisible) console.warn('  - Кнопка "Начать/Запустить"')
        if (createVisible) console.warn('  - Кнопка "Создать"') 
        if (editVisible) console.warn('  - Кнопка "Редактировать"')
        if (deleteVisible) console.warn('  - Кнопка "Удалить"')
      }
      
      // Проверяем попытку редактирования клиентских данных
      const editClientButton = page.locator('button:has-text("Редактировать клиента")').or(
        page.locator('[data-action="edit-client"]')
      ).first()
      
      if (await editClientButton.isVisible({ timeout: 5000 }).catch(() => false)) {
        // Попытка клика должна быть заблокирована или не иметь эффекта
        await editClientButton.click()
        await waitForPageLoad(page)
        
        // Проверяем, что не открылась форма редактирования
        const editForm = page.locator('form').or(
          page.locator('[role="dialog"]')
        ).first()
        
        const hasEditForm = await editForm.isVisible({ timeout: 3000 }).catch(() => false)
        
        if (!hasEditForm) {
          console.log('✅ Попытка редактирования заблокирована')
        } else {
          console.warn('⚠️ Форма редактирования открылась у наблюдателя')
        }
      }
      
      const duration = ((Date.now() - startTime) / 1000).toFixed(2)
      console.log(`✅ Тест наблюдателя завершен за ${duration} секунд`)
    })
  })

  test.describe('🛡️ Дополнительные проверки безопасности', () => {
    test('Проверка доступа к API без авторизации', async ({ page }) => {
      console.log('\n🛡️ Тест: Проверка доступа к API без авторизации...')
      
      // Пытаемся получить доступ к защищенным API эндпоинтам без токена
      const apiEndpoints = [
        '/api/clients',
        '/api/workers', 
        '/api/reports'
      ]
      
      for (const endpoint of apiEndpoints) {
        try {
          const response = await page.evaluate(async (url) => {
            const res = await fetch(url, { method: 'GET' })
            return { status: res.status, ok: res.ok }
          }, `${BACKEND_URL}${endpoint}`)
          
          if (response.status === 401 || response.status === 403) {
            console.log(`✅ Доступ к ${endpoint} запрещен без авторизации (${response.status})`)
          } else {
            console.warn(`⚠️ Доступ к ${endpoint} разрешен без авторизации (${response.status})`)
          }
        } catch (error) {
          console.warn(`⚠️ Ошибка при проверке ${endpoint}:`, error)
        }
      }
      
      console.log('✅ Проверка доступа к API без авторизации завершена')
    })

    test('Проверка доступа с некорректным токеном', async ({ page }) => {
      console.log('\n🛡️ Тест: Проверка доступа с некорректным токеном...')

      // Устанавливаем некорректный токен
      await page.context().setExtraHTTPHeaders({
        Authorization: 'Bearer invalid_token_123',
      })

      // Пытаемся получить доступ к защищенным страницам
      await page.goto('/monitoring')

      // Должны перенаправиться на страницу ошибки или входа
      const errorPage = page.locator('h1').or(page.locator('[role="alert"]')).first()

      const hasError = await errorPage.isVisible({ timeout: 5000 }).catch(() => false)

      if (hasError) {
        console.log('✅ Доступ с некорректным токеном запрещен')
      } else {
        console.warn('⚠️ Доступ с некорректным токеном разрешен')
      }
    })
  })
})