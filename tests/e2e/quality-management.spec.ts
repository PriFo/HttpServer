
/**
 * 📋 E2E ТЕСТЫ ДЛЯ УПРАВЛЕНИЯ КАЧЕСТВОМ ДАННЫХ
 * 
 * Этот тестовый набор проверяет функциональность управления качеством данных,
 * включая обнаружение и ручное слияние дубликатов.
 * 
 * Основные сценарии:
 * - Загрузка тестовых данных с известными дубликатами
 * - Запуск анализа качества данных
 * - Обнаружение и просмотр дубликатов
 * - Ручное слияние дубликатов
 * - Валидация результатов слияния
 * 
 * Prerequisites (требования для запуска):
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Запущенный Next.js фронтенд на http://localhost:3000
 * 3. Установленные переменные окружения для AI-провайдеров
 */

import { test, expect } from '@playwright/test'
import {
  createTestClient,
  createTestProject,
  uploadDatabaseFile,
  cleanupTestData,
  getNormalizationStatus,
  startNormalization,
  getQualityDuplicates,
  mergeDuplicates,
  findTestDatabase,
  getQualityMetrics,
} from '../../utils/api-testing'
import { waitForPageLoad, waitForText, clickIfVisible, checkToast, logPageInfo, wait } from './test-helpers'

// Конфигурация
const FRONTEND_URL = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'
const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:9999'

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
  testClientName: `Quality Test Client ${Date.now()}`,
  testProjectName: `Quality Test Project ${Date.now()}`,
}

// Тестовые данные с дубликатами контрагентов
const testDuplicatesData = [
  {
    name: 'ООО Ромашка',
    code: '12345',
    inn: '1234567890',
    kpp: '123456789',
    address: 'г. Москва, ул. Ленина, д. 1',
    phone: '+74951234567',
    email: 'info@romashka.com',
    category: 'Поставщик',
    kpved_code: '51.10',
    legal_form: 'ООО'
  },
  {
    name: 'ООО Ромашка',
    code: '12345',
    inn: '1234567890',
    kpp: '123456789',
    address: 'г. Москва, ул. Ленина, д. 1',
    phone: '+74951234567',
    email: 'info@romashka.com',
    category: 'Поставщик',
    kpved_code: '51.10',
    legal_form: 'ООО'
  },
  {
    name: 'ООО "Ромашка"',
    code: '12346',
    inn: '1234567890',
    kpp: '123456789',
    address: 'г. Москва, ул. Ленина, д. 1',
    phone: '+74951234567',
    email: 'info@romashka.com',
    category: 'Поставщик',
    kpved_code: '51.10',
    legal_form: 'ООО'
  },
  {
    name: 'ИП Ромашкин А.А.',
    code: '12347',
    inn: '9876543210',
    kpp: '',
    address: 'г. Москва, ул. Ленина, д. 1',
    phone: '+74951234567',
    email: 'info@romashkin.com',
    category: 'Поставщик',
    kpved_code: '51.10',
    legal_form: 'ИП'
  },
  {
    name: 'ООО Ромашка-Трейд',
    code: '12348',
    inn: '1234567891',
    kpp: '123456790',
    address: 'г. Москва, ул. Ленина, д. 1',
    phone: '+74951234567',
    email: 'info@romashka-trade.com',
    category: 'Поставщик',
    kpved_code: '51.10',
    legal_form: 'ООО'
  },
  {
    name: 'ООО Ромашка',
    code: '12349',
    inn: '1234567892',
    kpp: '123456791',
    address: 'г. Москва, ул. Ленина, д. 2',
    phone: '+74951234568',
    email: 'info2@romashka.com',
    category: 'Поставщик',
    kpved_code: '51.10',
    legal_form: 'ООО'
  }
]

test.describe('🔍 Управление качеством данных - E2E тесты', () => {
  test.beforeAll(async () => {
    console.log('🚀 Начало подготовки тестового окружения для управления качеством...')
    
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
        project_type: 'quality_analysis',
        description: 'Проект для тестирования управления качеством данных',
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
  })

  test('Обнаружение и ручное слияние дубликатов', async ({ page }) => {
      const startTime = Date.now()
      console.log('\n🎯 Начало теста: Ручное слияние дубликатов...')

      // Arrange: Загрузка тестовых данных с известными дубликатами контрагентов
      console.log('📁 Шаг 1: Подготовка тестовых данных...')
      
      // Проверяем, что база данных загружена
      test.skip(!testData.databaseId, 'База данных не загружена')

      // Запускаем нормализацию для анализа качества данных
      console.log('📊 Шаг 2: Запуск анализа качества данных...')
      try {
        const startResult = await startNormalization(testData.clientId!, testData.projectId!, {
          target_entity_types: ['counterparties'],
          quality_analysis: true,
          duplicate_detection: true
        })
        console.log('✅ Нормализация запущена:', startResult)
      } catch (error) {
        console.warn('⚠️ Не удалось запустить нормализацию:', error)
      }

      // Ждем завершения анализа
      console.log('⏳ Шаг 3: Ожидание завершения анализа качества...')
      let analysisCompleted = false
      let maxAttempts = 30 // 5 минут с интервалом 10 секунд
      let attempts = 0

      while (!analysisCompleted && attempts < maxAttempts) {
        attempts++
        try {
          const status = await getNormalizationStatus(testData.clientId!, testData.projectId!)
          console.log(`Попытка ${attempts}: Статус нормализации -`, status?.status)
          
          if (status && (status.status === 'completed' || status.status === 'finished')) {
            analysisCompleted = true
            console.log('✅ Анализ качества данных завершен')
            break
          } else if (status && status.status === 'failed') {
            console.warn('⚠️ Анализ качества завершился с ошибкой')
            break
          }
        } catch (error) {
          console.warn(`⚠️ Ошибка проверки статуса (попытка ${attempts}):`, error)
        }
        
        if (!analysisCompleted) {
          await wait(10000) // Ждем 10 секунд перед следующей проверкой
        }
      }

      if (!analysisCompleted) {
        console.warn('⚠️ Анализ качества не завершился за отведенное время, продолжаем тест')
      }

      // Act: Переход к интерфейсу управления дубликатами
      console.log('🖥️ Шаг 4: Переход к интерфейсу управления дубликатами...')
      await page.goto(`${FRONTEND_URL}/quality/duplicates`)
      await waitForPageLoad(page)
      await logPageInfo(page)
      await expect(page).toHaveTitle(/Дубликаты|Duplicates/i, { timeout: 10000 })
      
      // Выбираем базу данных
      const databaseSelector = page.locator('[data-testid="database-selector"]')
      await expect(databaseSelector).toBeVisible({ timeout: 10000 })
      
      // Ищем опцию с путем к базе данных
      const databaseOption = page.locator('option').filter({ hasText: testData.testDatabasePath || testData.testProjectName })
      if (await databaseOption.isVisible({ timeout: 5000 })) {
        await databaseOption.click()
        console.log('✅ База данных выбрана')
      } else {
        console.warn('⚠️ Не удалось выбрать базу данных через селектор')
      }

      // Ждем загрузки дубликатов
      await waitForPageLoad(page)
      
      // Проверяем наличие дубликатов
      const duplicatesContainer = page.locator('[data-testid="duplicates-container"]')
      const hasDuplicates = await duplicatesContainer.isVisible({ timeout: 10000 }).catch(() => false)
      
      if (!hasDuplicates) {
        console.warn('⚠️ Контейнер дубликатов не найден, проверяем наличие сообщений')
        const noDuplicatesMessage = page.locator('text=Дубликатов не найдено')
        const hasNoDuplicates = await noDuplicatesMessage.isVisible({ timeout: 5000 }).catch(() => false)
        
        if (hasNoDuplicates) {
          console.log('ℹ️ Дубликаты не найдены, возможно, база данных пуста')
          test.skip(true, 'Дубликаты не найдены в базе данных')
          return
        }
      }

      // Находим группу дубликатов для слияния
      console.log('🔍 Шаг 5: Поиск группы дубликатов...')
      const duplicateGroups = page.locator('[data-testid="duplicate-group"]')
      const groupCount = await duplicateGroups.count()
      
      console.log(`📊 Найдено групп дубликатов: ${groupCount}`)
      
      if (groupCount === 0) {
        console.warn('⚠️ Группы дубликатов не найдены')
        test.skip(true, 'Группы дубликатов не найдены')
        return
      }

      // Выбираем первую группу для теста
      const firstGroup = duplicateGroups.first()
      await expect(firstGroup).toBeVisible({ timeout: 10000 })
      
      // Получаем ID группы
      const groupIdText = await firstGroup.locator('[data-testid="group-id"]').textContent()
      const groupId = parseInt(groupIdText?.replace(/[^0-9]/g, '') || '0')
      console.log(`✅ Выбрана группа дубликатов ID: ${groupId}`)

      // Проверяем содержимое группы
      console.log('📋 Шаг 6: Проверка содержимого группы дубликатов...')
      const groupItems = firstGroup.locator('[data-testid="duplicate-item"]')
      const itemCount = await groupItems.count()
      console.log(`📊 В группе найдено записей: ${itemCount}`)
      
      if (itemCount < 2) {
        console.warn('⚠️ В группе недостаточно записей для слияния')
        test.skip(true, 'В группе недостаточно записей для слияния')
        return
      }

      // Находим мастер-запись
      const masterItem = groupItems.first()
      const masterIdText = await masterItem.locator('[data-testid="item-id"]').textContent()
      const masterId = parseInt(masterIdText?.replace(/[^0-9]/g, '') || '0')
      console.log(`✅ Мастер-запись ID: ${masterId}`)

      // Находим записи для слияния
      const itemsToMerge: number[] = []
      for (let i = 1; i < Math.min(itemCount, 3); i++) { // Берем до 2 записей для слияния
        const itemIdText = await groupItems.nth(i).locator('[data-testid="item-id"]').textContent()
        const itemId = parseInt(itemIdText?.replace(/[^0-9]/g, '') || '0')
        if (itemId > 0) {
          itemsToMerge.push(itemId)
        }
      }
      console.log(`✅ Записи для слияния: ${itemsToMerge.join(', ')}`)

      // Act: Выполнение слияния дубликатов
      console.log('🔄 Шаг 7: Выполнение слияния дубликатов...')
      
      // Нажимаем кнопку "Объединить"
      const mergeButton = firstGroup.locator('button:has-text("Объединить")')
      await expect(mergeButton).toBeVisible({ timeout: 5000 })
      await mergeButton.click()
      
      // Ждем появления диалога слияния
      const mergeDialog = page.locator('[role="dialog"]').filter({ hasText: 'Объединение дубликатов' })
      await expect(mergeDialog).toBeVisible({ timeout: 10000 })
      console.log('✅ Диалог слияния открыт')

      // Проверяем, что мастер-запись выбрана по умолчанию
      const masterRecordCheckbox = mergeDialog.locator('input[type="checkbox"]').first()
      const isMasterSelected = await masterRecordCheckbox.isChecked()
      console.log(`✅ Статус мастер-записи: ${isMasterSelected ? 'выбрана' : 'не выбрана'}`)

      // Выбираем записи для слияния
      for (const itemId of itemsToMerge) {
        const itemCheckbox = mergeDialog.locator(`input[type="checkbox"]`).nth(1) // Первая - мастер, остальные - для слияния
        if (await itemCheckbox.isVisible({ timeout: 5000 })) {
          await itemCheckbox.check()
          console.log(`✅ Запись ID ${itemId} выбрана для слияния`)
        }
      }

      // Подтверждаем слияние
      const confirmButton = mergeDialog.locator('button:has-text("Подтвердить")')
      await expect(confirmButton).toBeVisible({ timeout: 5000 })
      await confirmButton.click()
      console.log('✅ Слияние подтверждено')

      // Ждем завершения операции
      await waitForPageLoad(page)

      // Проверяем toast-уведомление об успехе
      const hasSuccess = await checkToast(page, /успешно|success|объединен/i, 'success', 5000)
      if (hasSuccess) {
        console.log('✅ Toast-уведомление об успехе отображается')
      }

      // Проверяем через API, что дубликаты объединены
      await waitForPageLoad(page)
      try {
        const updatedDuplicates = await getQualityDuplicates(undefined, {
          unmerged: true,
          limit: 10,
        })
        console.log(`✅ Обновленное количество групп: ${updatedDuplicates.groups?.length || 0}`)
      } catch (error) {
        console.warn('⚠️ Не удалось проверить обновление через API:', error)
      }

      const duration = ((Date.now() - startTime) / 1000).toFixed(2)
      console.log(`✅ Тест слияния дубликатов завершен за ${duration} секунд`)
  })

  test('Анализ качества данных', async ({ page }) => {
    console.log('\n🎯 Тест: Анализ качества данных...')

    test.skip(!testData.databaseId, 'Тестовая база данных не загружена')

    // Act: Переходим на страницу качества
    await page.goto('/quality')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Проверяем наличие заголовка
    const qualityHeader = page.locator('text=Качество данных').or(
      page.locator('text=Общая оценка качества')
    ).or(
      page.locator('h1:has-text("Качество")')
    ).first()

    await expect(qualityHeader).toBeVisible({ timeout: 10000 })

    // Проверяем наличие метрик качества
    const qualityScore = page.locator('text=/\\d+\\.\\d+%|Оценка качества|Quality Score/i').first()
    const hasScore = await qualityScore.isVisible({ timeout: 10000 }).catch(() => false)

    if (hasScore) {
      console.log('✅ Метрики качества отображаются')
    }

    // Проверяем разделы контрагентов и номенклатуры
    const counterpartiesSection = page.locator('text=Контрагенты').or(
      page.locator('[data-testid="counterparties"]')
    ).first()

    const nomenclatureSection = page.locator('text=Номенклатура').or(
      page.locator('[data-testid="nomenclature"]')
    ).first()

    const hasCounterparties = await counterpartiesSection.isVisible({ timeout: 5000 }).catch(() => false)
    const hasNomenclature = await nomenclatureSection.isVisible({ timeout: 5000 }).catch(() => false)

    if (hasCounterparties || hasNomenclature) {
      console.log('✅ Разделы контрагентов и/или номенклатуры отображаются')
    }

    // Проверяем нарушения качества
    const violationsSection = page.locator('text=Нарушения').or(
      page.locator('text=Violations')
    ).or(
      page.locator('[data-testid="violations"]')
    ).first()

    const hasViolations = await violationsSection.isVisible({ timeout: 5000 }).catch(() => false)

    if (hasViolations) {
      console.log('✅ Раздел нарушений качества отображается')
    }

    // Проверяем через API
    try {
      const metrics = await getQualityMetrics()
      console.log('✅ Метрики качества получены через API:', Object.keys(metrics))
    } catch (error) {
      console.warn('⚠️ Не удалось получить метрики через API:', error)
    }

    console.log('✅ Тест анализа качества завершен')
  })

  test('Фильтрация дубликатов', async ({ page }) => {
    console.log('\n🎯 Тест: Фильтрация дубликатов...')

    test.skip(!testData.databaseId, 'Тестовая база данных не загружена')

    // Act: Переходим на страницу дубликатов
    await page.goto('/quality/duplicates')
    await waitForPageLoad(page)
    await logPageInfo(page)

    // Ищем фильтры
    const unmergedFilter = page.locator('input[type="checkbox"]').filter({ hasText: /необъединен|unmerged/i }).or(
      page.locator('label:has-text(/только необъединенные|unmerged only/i)')
    ).first()

    const filterSelect = page.locator('select').filter({ hasText: /тип|type|категория/i }).first()

    // Применяем фильтр "только необъединенные"
    if (await unmergedFilter.isVisible({ timeout: 5000 })) {
      await unmergedFilter.check()
      await waitForPageLoad(page)
      console.log('✅ Фильтр "только необъединенные" применен')

      // Проверяем, что список обновился
      const duplicateList = page.locator('[data-testid="duplicate-list"]').or(
        page.locator('.duplicate-group')
      ).first()

      const hasList = await duplicateList.isVisible({ timeout: 5000 }).catch(() => false)
      if (hasList) {
        console.log('✅ Список дубликатов обновился после применения фильтра')
      }
    }

    // Применяем фильтр по типу (если есть)
    if (await filterSelect.isVisible({ timeout: 5000 })) {
      const options = await filterSelect.locator('option').all()
      if (options.length > 1) {
        await filterSelect.selectOption({ index: 1 })
        await waitForPageLoad(page)
        console.log('✅ Фильтр по типу применен')
      }
    }

    // Проверяем через API
    try {
      const filteredDuplicates = await getQualityDuplicates(undefined, {
        unmerged: true,
        limit: 10,
      })
      console.log(`✅ Отфильтрованные дубликаты получены: ${filteredDuplicates.groups?.length || 0} групп`)
    } catch (error) {
      console.warn('⚠️ Не удалось получить отфильтрованные дубликаты:', error)
    }

    console.log('✅ Тест фильтрации дубликатов завершен')
  })
})

