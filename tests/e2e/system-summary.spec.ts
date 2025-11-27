/**
 * 📋 E2E ТЕСТЫ ДЛЯ СИСТЕМНОЙ СВОДКИ
 * 
 * Тесты покрывают:
 * - Получение сводной информации по всем базам данных системы
 * - Проверку корректности подсчета метрик
 * - Валидацию структуры ответа
 * 
 * Prerequisites:
 * 1. Запущенный Go-бэкенд на http://127.0.0.1:9999
 * 2. Наличие хотя бы одной базы данных в системе (желательно)
 */

import { test, expect } from '@playwright/test'
import { logPageInfo } from './test-helpers'

// Конфигурация
const BACKEND_URL = process.env.BACKEND_URL || 'http://127.0.0.1:9999'
const API_BASE_URL = `${BACKEND_URL}/api`

// Интерфейсы для типизации ответа
interface SystemSummary {
  total_databases: number
  total_uploads: number
  completed_uploads: number
  failed_uploads: number
  in_progress_uploads: number
  last_activity: string
  total_nomenclature: number
  total_counterparties: number
  upload_details: UploadSummary[]
  // Метрики производительности (опционально)
  scan_duration?: string
  databases_processed: number
  databases_skipped?: number
}

interface UploadSummary {
  id: string
  upload_uuid: string
  name: string
  status: string
  created_at: string
  completed_at?: string
  nomenclature_count: number
  counterparty_count: number
  database_file: string
  database_id?: number
  client_id?: number
  project_id?: number
}

test.describe('System Summary API', () => {
  test('должен возвращать сводную информацию о системе', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)

    expect(response.ok()).toBeTruthy()
    expect(response.status()).toBe(200)

    const summary: SystemSummary = await response.json()

    // Проверяем основную структуру ответа
    expect(summary).toHaveProperty('total_databases')
    expect(summary).toHaveProperty('total_uploads')
    expect(summary).toHaveProperty('completed_uploads')
    expect(summary).toHaveProperty('failed_uploads')
    expect(summary).toHaveProperty('in_progress_uploads')
    expect(summary).toHaveProperty('last_activity')
    expect(summary).toHaveProperty('total_nomenclature')
    expect(summary).toHaveProperty('total_counterparties')
    expect(summary).toHaveProperty('upload_details')

    // Проверяем типы данных
    expect(typeof summary.total_databases).toBe('number')
    expect(typeof summary.total_uploads).toBe('number')
    expect(typeof summary.completed_uploads).toBe('number')
    expect(typeof summary.failed_uploads).toBe('number')
    expect(typeof summary.in_progress_uploads).toBe('number')
    expect(typeof summary.total_nomenclature).toBe('number')
    expect(typeof summary.total_counterparties).toBe('number')
    expect(Array.isArray(summary.upload_details)).toBe(true)

    // Проверяем, что значения не отрицательные
    expect(summary.total_databases).toBeGreaterThanOrEqual(0)
    expect(summary.total_uploads).toBeGreaterThanOrEqual(0)
    expect(summary.completed_uploads).toBeGreaterThanOrEqual(0)
    expect(summary.failed_uploads).toBeGreaterThanOrEqual(0)
    expect(summary.in_progress_uploads).toBeGreaterThanOrEqual(0)
    expect(summary.total_nomenclature).toBeGreaterThanOrEqual(0)
    expect(summary.total_counterparties).toBeGreaterThanOrEqual(0)
  })

  test('должен возвращать корректную структуру деталей загрузок', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Проверяем структуру деталей загрузок
    if (summary.upload_details.length > 0) {
      const upload = summary.upload_details[0]

      expect(upload).toHaveProperty('id')
      expect(upload).toHaveProperty('upload_uuid')
      expect(upload).toHaveProperty('name')
      expect(upload).toHaveProperty('status')
      expect(upload).toHaveProperty('created_at')
      expect(upload).toHaveProperty('nomenclature_count')
      expect(upload).toHaveProperty('counterparty_count')
      expect(upload).toHaveProperty('database_file')

      // Проверяем типы
      expect(typeof upload.id).toBe('string')
      expect(typeof upload.upload_uuid).toBe('string')
      expect(typeof upload.name).toBe('string')
      expect(typeof upload.status).toBe('string')
      expect(typeof upload.nomenclature_count).toBe('number')
      expect(typeof upload.counterparty_count).toBe('number')
      expect(typeof upload.database_file).toBe('string')

      // Проверяем валидность статуса
      expect(['completed', 'failed', 'in_progress']).toContain(upload.status)

      // Проверяем, что счетчики не отрицательные
      expect(upload.nomenclature_count).toBeGreaterThanOrEqual(0)
      expect(upload.counterparty_count).toBeGreaterThanOrEqual(0)
    }
  })

  test('должен возвращать корректное количество баз данных', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Количество уникальных БД должно соответствовать количеству загрузок с database_id
    const uniqueDatabases = new Set(
      summary.upload_details
        .filter(u => u.database_id !== undefined && u.database_id !== null)
        .map(u => u.database_id)
    )

    // Это не строгая проверка, так как uploads может быть больше чем баз данных
    // (несколько uploads могут ссылаться на одну БД)
    expect(summary.total_databases).toBeGreaterThanOrEqual(uniqueDatabases.size)
  })

  test('должен корректно суммировать статистику по загрузкам', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Проверяем, что общее количество загрузок соответствует сумме по статусам
    const expectedTotal = 
      summary.completed_uploads + 
      summary.failed_uploads + 
      summary.in_progress_uploads

    // Может быть небольшое расхождение из-за других статусов
    expect(summary.total_uploads).toBeGreaterThanOrEqual(expectedTotal)

    // Проверяем, что общее количество номенклатуры и контрагентов соответствует сумме
    const totalNomenclature = summary.upload_details.reduce(
      (sum, u) => sum + u.nomenclature_count, 
      0
    )
    const totalCounterparties = summary.upload_details.reduce(
      (sum, u) => sum + u.counterparty_count, 
      0
    )

    expect(summary.total_nomenclature).toBe(totalNomenclature)
    expect(summary.total_counterparties).toBe(totalCounterparties)
  })

  test('должен возвращать корректную дату последней активности', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    if (summary.upload_details.length > 0) {
      // Проверяем, что last_activity - валидная дата
      const lastActivity = new Date(summary.last_activity)
      expect(lastActivity.getTime()).not.toBeNaN()

      // Проверяем, что last_activity не в будущем
      expect(lastActivity.getTime()).toBeLessThanOrEqual(Date.now())

      // Проверяем, что last_activity соответствует самой поздней загрузке
      const latestUpload = summary.upload_details
        .map(u => {
          const completed = u.completed_at ? new Date(u.completed_at) : null
          const created = new Date(u.created_at)
          return completed && completed > created ? completed : created
        })
        .sort((a, b) => b.getTime() - a.getTime())[0]

      if (latestUpload) {
        expect(lastActivity.getTime()).toBeGreaterThanOrEqual(latestUpload.getTime())
      }
    }
  })

  test('должен обрабатывать запрос без ошибок при пустой системе', async ({ request }) => {
    // Этот тест проверяет, что API корректно обрабатывает случай,
    // когда в системе нет загрузок
    const response = await request.get(`${API_BASE_URL}/system/summary`)

    expect(response.ok()).toBeTruthy()
    expect(response.status()).toBe(200)

    const summary: SystemSummary = await response.json()

    // Даже при пустой системе должен вернуться валидный ответ
    expect(summary.total_uploads).toBe(0)
    expect(summary.total_databases).toBe(0)
    expect(summary.completed_uploads).toBe(0)
    expect(summary.failed_uploads).toBe(0)
    expect(summary.in_progress_uploads).toBe(0)
    expect(summary.total_nomenclature).toBe(0)
    expect(summary.total_counterparties).toBe(0)
    expect(summary.upload_details).toEqual([])
  })

  test('должен возвращать ошибку 405 для неподдерживаемых HTTP методов', async ({ request }) => {
    // Тестируем POST метод (должен быть только GET)
    const postResponse = await request.post(`${API_BASE_URL}/system/summary`)
    expect(postResponse.status()).toBe(405)

    // Тестируем PUT метод
    const putResponse = await request.put(`${API_BASE_URL}/system/summary`)
    expect(putResponse.status()).toBe(405)

    // Тестируем DELETE метод
    const deleteResponse = await request.delete(`${API_BASE_URL}/system/summary`)
    expect(deleteResponse.status()).toBe(405)
  })

  test('должен иметь правильный Content-Type в ответе', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response.ok()).toBeTruthy()

    const contentType = response.headers()['content-type']
    expect(contentType).toContain('application/json')
  })

  test('должен возвращать метрики производительности сканирования', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Проверяем наличие метрик производительности
    expect(summary).toHaveProperty('databases_processed')
    expect(typeof summary.databases_processed).toBe('number')
    expect(summary.databases_processed).toBeGreaterThanOrEqual(0)

    // scan_duration может отсутствовать если сканирование было очень быстрым
    if (summary.scan_duration) {
      expect(typeof summary.scan_duration).toBe('string')
      // Проверяем формат длительности (например, "123ms", "1.23s")
      expect(summary.scan_duration).toMatch(/^\d+\.?\d*(ns|us|µs|ms|s|m|h)$/)
    }

    // databases_skipped может отсутствовать если все БД обработаны успешно
    if (summary.databases_skipped !== undefined) {
      expect(typeof summary.databases_skipped).toBe('number')
      expect(summary.databases_skipped).toBeGreaterThanOrEqual(0)
    }

    // Проверяем логику: databases_processed + databases_skipped <= total_databases (или близко к этому)
    const totalDatabases = summary.databases_processed + (summary.databases_skipped || 0)
    // Это не строгая проверка, так как total_databases считается по уникальным database_id
    expect(totalDatabases).toBeLessThanOrEqual(summary.total_databases * 2) // допускаем небольшое расхождение
  })

  test('должен возвращать корректные метрики производительности при повторных запросах', async ({ request }) => {
    // Делаем первый запрос
    const response1 = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response1.ok()).toBeTruthy()
    const summary1: SystemSummary = await response1.json()

    // Делаем второй запрос
    const response2 = await request.get(`${API_BASE_URL}/system/summary`)
    expect(response2.ok()).toBeTruthy()
    const summary2: SystemSummary = await response2.json()

    // Проверяем, что основные метрики совпадают (если данные не изменились)
    expect(summary2.total_uploads).toBe(summary1.total_uploads)
    expect(summary2.total_databases).toBe(summary1.total_databases)

    // Метрики производительности могут отличаться из-за кеширования или изменений
    // Но они должны быть валидными
    if (summary2.scan_duration) {
      expect(typeof summary2.scan_duration).toBe('string')
    }
    expect(summary2.databases_processed).toBeGreaterThanOrEqual(0)
  })
})

test.describe('System Summary Cache Management API', () => {
  test('должен возвращать статистику кеша', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary/cache/stats`)
    
    expect(response.ok()).toBeTruthy()
    expect(response.status()).toBe(200)

    const stats = await response.json()

    // Проверяем структуру статистики
    expect(stats).toHaveProperty('hits')
    expect(stats).toHaveProperty('misses')
    expect(stats).toHaveProperty('hit_rate')
    expect(stats).toHaveProperty('has_data')
    expect(stats).toHaveProperty('is_stale')

    // Проверяем типы
    expect(typeof stats.hits).toBe('number')
    expect(typeof stats.misses).toBe('number')
    expect(typeof stats.hit_rate).toBe('number')
    expect(typeof stats.has_data).toBe('boolean')
    expect(typeof stats.is_stale).toBe('boolean')

    // Проверяем, что значения валидны
    expect(stats.hits).toBeGreaterThanOrEqual(0)
    expect(stats.misses).toBeGreaterThanOrEqual(0)
    expect(stats.hit_rate).toBeGreaterThanOrEqual(0)
    expect(stats.hit_rate).toBeLessThanOrEqual(1)
  })

  test('должен инвалидировать кеш', async ({ request }) => {
    // Делаем запрос, чтобы заполнить кеш
    await request.get(`${API_BASE_URL}/system/summary`)

    // Получаем статистику до инвалидации
    const statsBefore = await request.get(`${API_BASE_URL}/system/summary/cache/stats`)
    const statsBeforeData = await statsBefore.json()

    // Инвалидируем кеш
    const response = await request.post(`${API_BASE_URL}/system/summary/cache/invalidate`)
    expect(response.ok()).toBeTruthy()
    expect(response.status()).toBe(200)

    const result = await response.json()
    expect(result).toHaveProperty('message')
    expect(result.message).toContain('инвалидирован')
    expect(result).toHaveProperty('stats')

    // Проверяем, что кеш помечен как устаревший
    const statsAfter = await request.get(`${API_BASE_URL}/system/summary/cache/stats`)
    const statsAfterData = await statsAfter.json()
    
    // После инвалидации is_stale должен быть true (если есть данные)
    if (statsAfterData.has_data) {
      expect(statsAfterData.is_stale).toBe(true)
    }
  })

  test('должен очищать кеш', async ({ request }) => {
    // Делаем запрос, чтобы заполнить кеш
    await request.get(`${API_BASE_URL}/system/summary`)

    // Очищаем кеш
    const response = await request.post(`${API_BASE_URL}/system/summary/cache/clear`)
    expect(response.ok()).toBeTruthy()
    expect(response.status()).toBe(200)

    const result = await response.json()
    expect(result).toHaveProperty('message')
    expect(result.message).toContain('очищен')

    // Проверяем, что кеш пуст
    const statsAfter = await request.get(`${API_BASE_URL}/system/summary/cache/stats`)
    const statsAfterData = await statsAfter.json()
    
    expect(statsAfterData.has_data).toBe(false)
  })

  test('должен возвращать ошибку 405 для неподдерживаемых HTTP методов на /cache/stats', async ({ request }) => {
    const postResponse = await request.post(`${API_BASE_URL}/system/summary/cache/stats`)
    expect(postResponse.status()).toBe(405)
  })

  test('должен возвращать ошибку 405 для неподдерживаемых HTTP методов на /cache/invalidate', async ({ request }) => {
    const getResponse = await request.get(`${API_BASE_URL}/system/summary/cache/invalidate`)
    expect(getResponse.status()).toBe(405)

    const putResponse = await request.put(`${API_BASE_URL}/system/summary/cache/invalidate`)
    expect(putResponse.status()).toBe(405)
  })

  test('должен возвращать ошибку 405 для неподдерживаемых HTTP методов на /cache/clear', async ({ request }) => {
    const getResponse = await request.get(`${API_BASE_URL}/system/summary/cache/clear`)
    expect(getResponse.status()).toBe(405)

    const putResponse = await request.put(`${API_BASE_URL}/system/summary/cache/clear`)
    expect(putResponse.status()).toBe(405)
  })
})

test.describe('System Summary Filtering and Sorting', () => {
  test('должен фильтровать по статусу', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary?status=completed`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Проверяем, что все загрузки имеют статус completed
    summary.upload_details.forEach((upload) => {
      expect(upload.status).toBe('completed')
    })
  })

  test('должен фильтровать по нескольким статусам', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary?status=completed,failed`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    summary.upload_details.forEach((upload) => {
      expect(['completed', 'failed']).toContain(upload.status)
    })
  })

  test('должен искать по имени загрузки', async ({ request }) => {
    // Сначала получаем все загрузки
    const allResponse = await request.get(`${API_BASE_URL}/system/summary`)
    const allSummary: SystemSummary = await allResponse.json()

    if (allSummary.upload_details.length > 0) {
      const searchTerm = allSummary.upload_details[0].name.substring(0, 5)

      const response = await request.get(`${API_BASE_URL}/system/summary?search=${encodeURIComponent(searchTerm)}`)
      expect(response.ok()).toBeTruthy()

      const summary: SystemSummary = await response.json()

      // Проверяем, что все результаты содержат поисковый термин
      summary.upload_details.forEach((upload) => {
        const nameLower = upload.name.toLowerCase()
        const uuidLower = upload.upload_uuid.toLowerCase()
        const searchLower = searchTerm.toLowerCase()
        expect(nameLower.includes(searchLower) || uuidLower.includes(searchLower)).toBe(true)
      })
    }
  })

  test('должен сортировать по имени', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary?sort_by=name&order=asc`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    if (summary.upload_details.length > 1) {
      for (let i = 1; i < summary.upload_details.length; i++) {
        const prev = summary.upload_details[i - 1].name.toLowerCase()
        const curr = summary.upload_details[i].name.toLowerCase()
        expect(prev <= curr).toBe(true)
      }
    }
  })

  test('должен применять пагинацию', async ({ request }) => {
    const response = await request.get(`${API_BASE_URL}/system/summary?limit=5&page=1`)
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Проверяем, что количество результатов не превышает лимит
    expect(summary.upload_details.length).toBeLessThanOrEqual(5)
  })

  test('должен комбинировать фильтры', async ({ request }) => {
    const response = await request.get(
      `${API_BASE_URL}/system/summary?status=completed&sort_by=created_at&order=desc&limit=10`
    )
    expect(response.ok()).toBeTruthy()

    const summary: SystemSummary = await response.json()

    // Проверяем статус
    summary.upload_details.forEach((upload) => {
      expect(upload.status).toBe('completed')
    })

    // Проверяем лимит
    expect(summary.upload_details.length).toBeLessThanOrEqual(10)
  })
})
