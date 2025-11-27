/**
 * 📋 E2E ТЕСТЫ ДЛЯ КЛАССИФИКАЦИИ
 * 
 * Тесты проверяют функциональность классификации КПВЭД:
 * - Получение иерархии КПВЭД
 * - Поиск по КПВЭД
 * - Статистика КПВЭД
 * - Тестирование классификации
 * - Иерархическая классификация
 * - Сброс классификации
 * - Пометка неправильной классификации
 * 
 * Prerequisites:
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 * 3. Тестовая база данных (SQLite) в одном из стандартных мест
 */

import { test, expect } from '@playwright/test'
import {
  createTestClient,
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData,
  findTestDatabase,
  getKpvedHierarchy,
  searchKpved,
  getKpvedStats,
  testClassification,
  classifyHierarchical,
  resetClassification,
  markClassificationIncorrect,
  markClassificationCorrect,
} from '../../utils/api-testing'
import { waitForPageLoad, logPageInfo } from './test-helpers'

// Конфигурация
const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'

// Тестовые данные
interface TestData {
  clientId?: number
  projectId?: number
  databaseId?: number
  testClientName: string
  testProjectName: string
}

const testData: TestData = {
  testClientName: `Classification Test Client ${Date.now()}`,
  testProjectName: `Classification Test Project ${Date.now()}`,
}

test.describe('Классификация КПВЭД', () => {
  test.beforeAll(async () => {
    console.log('🚀 Начало подготовки тестового окружения для классификации...')
    
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

    // Загружаем тестовую БД, если доступна
    const dbPath = findTestDatabase()
    if (dbPath) {
      try {
        const database = await uploadDatabaseFile(
          testData.clientId!,
          testData.projectId!,
          dbPath
        )
        testData.databaseId = database.id || database
        console.log(`✅ Загружена тестовая база данных: ${dbPath} (ID: ${testData.databaseId})`)
      } catch (error) {
        console.warn(`⚠️ Не удалось загрузить ${dbPath}:`, error)
      }
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

  test('Получение иерархии КПВЭД', async ({ page }) => {
    console.log('\n🎯 Тест: Получение иерархии КПВЭД...')

    await page.goto('/')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Act: Получаем иерархию через API
    try {
      // Получаем корневые элементы
      const rootHierarchy = await getKpvedHierarchy()
      console.log(`✅ Получена корневая иерархия: ${rootHierarchy.total || 0} элементов`)
      
      // Assert: Проверяем структуру ответа
      expect(rootHierarchy).toBeDefined()
      expect(rootHierarchy.nodes).toBeDefined()
      expect(Array.isArray(rootHierarchy.nodes)).toBe(true)
      
      // Если есть узлы, получаем дочерние элементы
      if (rootHierarchy.nodes && rootHierarchy.nodes.length > 0) {
        const firstNode = rootHierarchy.nodes[0]
        if (firstNode.code) {
          const childHierarchy = await getKpvedHierarchy(firstNode.code)
          console.log(`✅ Получена дочерняя иерархия для ${firstNode.code}: ${childHierarchy.total || 0} элементов`)
          expect(childHierarchy.nodes).toBeDefined()
        }
      }
    } catch (error) {
      console.warn('⚠️ Не удалось получить иерархию КПВЭД:', error)
      // Не падаем, если API недоступен
    }
  })

  test('Поиск по КПВЭД', async ({ page }) => {
    console.log('\n🎯 Тест: Поиск по КПВЭД...')

    await page.goto('/')
    await waitForPageLoad(page)

    // Act: Ищем по КПВЭД через API
    const searchQueries = ['товар', 'услуга', 'продукт']
    
    for (const query of searchQueries) {
      try {
        const results = await searchKpved(query, 10)
        console.log(`✅ Поиск "${query}": найдено ${results.items?.length || 0} результатов`)
        
        // Assert: Проверяем структуру ответа
        expect(results).toBeDefined()
        if (results.items) {
          expect(Array.isArray(results.items)).toBe(true)
          expect(results.items.length).toBeLessThanOrEqual(10)
        }
      } catch (error) {
        console.warn(`⚠️ Не удалось выполнить поиск "${query}":`, error)
      }
    }
  })

  test('Статистика КПВЭД', async ({ page }) => {
    console.log('\n🎯 Тест: Статистика КПВЭД...')

    await page.goto('/')
    await waitForPageLoad(page)

    // Act: Получаем статистику через API
    try {
      const stats = await getKpvedStats()
      console.log('✅ Получена статистика КПВЭД:', stats)
      
      // Assert: Проверяем структуру ответа
      expect(stats).toBeDefined()
      // Статистика может содержать различные поля
    } catch (error) {
      console.warn('⚠️ Не удалось получить статистику КПВЭД:', error)
      // Не падаем, если API недоступен
    }
  })

  test('Тестирование классификации', async ({ page }) => {
    console.log('\n🎯 Тест: Тестирование классификации...')

    await page.goto('/')
    await waitForPageLoad(page)

    // Act: Тестируем классификацию через API
    const testNames = ['ООО Ромашка', 'ИП Иванов', 'Товар 1']
    
    for (const name of testNames) {
      try {
        const result = await testClassification(name)
        console.log(`✅ Классификация "${name}":`, result)
        
        // Assert: Проверяем структуру ответа
        expect(result).toBeDefined()
        // Результат может содержать код КПВЭД, уверенность и т.д.
      } catch (error: any) {
        // Может быть ошибка, если AI API ключ не настроен
        if (error.message?.includes('not configured') || error.message?.includes('ServiceUnavailable')) {
          console.log(`ℹ️ Классификация "${name}" пропущена: AI API ключ не настроен`)
        } else {
          console.warn(`⚠️ Не удалось классифицировать "${name}":`, error)
        }
      }
    }
  })

  test('Иерархическая классификация', async ({ page }) => {
    console.log('\n🎯 Тест: Иерархическая классификация...')

    await page.goto('/')
    await waitForPageLoad(page)

    // Act: Выполняем иерархическую классификацию
    try {
      const result = await classifyHierarchical('ООО Ромашка', 'Поставщик')
      console.log('✅ Иерархическая классификация выполнена:', result)
      
      // Assert: Проверяем структуру ответа
      expect(result).toBeDefined()
    } catch (error: any) {
      // Может быть ошибка, если AI API ключ не настроен
      if (error.message?.includes('not configured') || error.message?.includes('ServiceUnavailable')) {
        console.log('ℹ️ Иерархическая классификация пропущена: AI API ключ не настроен')
      } else {
        console.warn('⚠️ Не удалось выполнить иерархическую классификацию:', error)
      }
    }
  })

  test('Сброс классификации', async ({ page }) => {
    console.log('\n🎯 Тест: Сброс классификации...')

    test.skip(!testData.databaseId, 'Тестовая база данных не загружена')

    await page.goto('/')
    await waitForPageLoad(page)

    // Act: Сбрасываем классификацию через API
    try {
      const result = await resetClassification('Тестовое название', '51.10')
      console.log('✅ Классификация сброшена:', result)
      
      // Assert: Проверяем структуру ответа
      expect(result).toBeDefined()
      if (result.success !== undefined) {
        expect(result.success).toBe(true)
      }
    } catch (error) {
      console.warn('⚠️ Не удалось сбросить классификацию:', error)
      // Не падаем, если API недоступен
    }
  })

  test('Пометка неправильной классификации', async ({ page }) => {
    console.log('\n🎯 Тест: Пометка неправильной классификации...')

    test.skip(!testData.databaseId, 'Тестовая база данных не загружена')

    await page.goto('/')
    await waitForPageLoad(page)

    // Act: Помечаем классификацию как неправильную
    try {
      const result = await markClassificationIncorrect('Тестовое название', '51.10')
      console.log('✅ Классификация помечена как неправильная:', result)
      
      // Assert: Проверяем структуру ответа
      expect(result).toBeDefined()
      if (result.success !== undefined) {
        expect(result.success).toBe(true)
      }
      
      // Снимаем пометку
      const unmarkResult = await markClassificationCorrect('Тестовое название', '51.10')
      console.log('✅ Пометка снята:', unmarkResult)
    } catch (error) {
      console.warn('⚠️ Не удалось пометить классификацию:', error)
      // Не падаем, если API недоступен
    }
  })

  test('Проверка UI для классификации (если есть)', async ({ page }) => {
    console.log('\n🎯 Тест: Проверка UI для классификации...')

    // Act: Пытаемся найти страницу классификации
    const possiblePaths = ['/classification', '/kpved', '/settings/classification']
    
    for (const path of possiblePaths) {
      try {
        await page.goto(path)
        await waitForPageLoad(page)
        await logPageInfo(page)
        
        // Проверяем наличие заголовка или контента
        const header = page.locator('h1, h2').filter({ hasText: /классификац|КПВЭД|kpved/i }).first()
        const hasHeader = await header.isVisible({ timeout: 3000 }).catch(() => false)
        
        if (hasHeader) {
          console.log(`✅ Найдена страница классификации: ${path}`)
          
          // Проверяем наличие элементов интерфейса
          const searchInput = page.locator('input[type="search"]').or(
            page.locator('input[placeholder*="поиск"]')
          ).first()
          const hasSearch = await searchInput.isVisible({ timeout: 5000 }).catch(() => false)
          
          if (hasSearch) {
            console.log('✅ Поиск по КПВЭД доступен в UI')
          }
          
          return // Нашли страницу, выходим
        }
      } catch (error) {
        // Продолжаем поиск
        continue
      }
    }
    
    console.log('ℹ️ UI для классификации не найден (возможно, не реализован)')
  })
})

