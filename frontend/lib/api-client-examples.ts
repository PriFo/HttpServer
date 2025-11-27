/**
 * Примеры использования API-клиента
 * Этот файл служит как документация и может быть удален в production
 */

import { apiClient, apiGet, apiPost, apiPut, apiDelete } from './api-client'
import { useApiClient } from '../hooks/useApiClient'

// ============================================================================
// Пример 1: Базовое использование
// ============================================================================

export async function example1_BasicUsage() {
  try {
    const response = await apiClient('/api/users')
    const data = await response.json()
    return data
  } catch (error) {
    // error - это AppError
    console.error('Ошибка:', error instanceof Error ? error.message : String(error))
    throw error
  }
}

// ============================================================================
// Пример 2: Использование удобных функций
// ============================================================================

export async function example2_ConvenienceFunctions() {
  // GET запрос
  const users = await apiGet<User[]>('/api/users')
  
  // POST запрос
  const newUser = await apiPost<User>('/api/users', {
    name: 'John',
    email: 'john@example.com',
  })
  
  // PUT запрос
  const updatedUser = await apiPut<User>(`/api/users/${newUser.id}`, {
    name: 'John Doe',
  })
  
  // DELETE запрос
  await apiDelete(`/api/users/${newUser.id}`)
}

// ============================================================================
// Пример 3: Использование в React компоненте
// ============================================================================

/*
export function MyComponent() {
  const { get, post } = useApiClient()
  const [data, setData] = useState(null)
  
  useEffect(() => {
    const loadData = async () => {
      try {
        const result = await get<MyDataType>('/api/data')
        setData(result)
      } catch (error) {
        // Ошибка уже обработана через ErrorContext
        // Toast уже показан пользователю
      }
    }
    
    loadData()
  }, [get])
  
  const handleSubmit = async (formData: FormData) => {
    try {
      await post('/api/submit', formData)
      toast.success('Данные сохранены')
    } catch (error) {
      // Ошибка уже обработана
    }
  }
  
  // return ваш JSX компонент
}
*/

// ============================================================================
// Пример 4: Retry механизм
// ============================================================================

export async function example4_Retry() {
  // Повторит запрос 3 раза при сетевых ошибках или 5xx ошибках
  try {
    const data = await apiGet('/api/unstable-endpoint', {
      retries: 3,
      retryDelay: 1000, // 1 секунда между попытками
    })
    return data
  } catch (error) {
    // Ошибка после всех попыток
    console.error('Все попытки исчерпаны:', error)
    throw error
  }
}

// ============================================================================
// Пример 5: Кастомная обработка ошибок
// ============================================================================

export async function example5_CustomErrorHandling() {
  try {
    const data = await apiGet('/api/data', {
      skipErrorHandler: true, // Не показывать toast автоматически
      onError: (error) => {
        // Кастомная логика
        if (error.statusCode === 401) {
          // Перенаправить на страницу входа
          window.location.href = '/login'
          return
        }
        
        if (error.statusCode === 404) {
          // Специальная обработка для 404
          console.log('Ресурс не найден')
          return
        }
        
        // Для остальных ошибок используем стандартную обработку
        // Можно вызвать handleError из контекста
      }
    })
    return data
  } catch (error) {
    // Ошибка уже обработана в onError
  }
}

// ============================================================================
// Пример 6: Обработка ошибок валидации форм
// ============================================================================

/*
export function FormComponent() {
  const { post } = useApiClient()
  const [formErrors, setFormErrors] = useState<Record<string, string>>({})
  
  const handleSubmit = async (data: FormData) => {
    try {
      await post('/api/submit', data, { skipErrorHandler: true })
      toast.success('Форма отправлена')
    } catch (error) {
      if (error instanceof AppError && error.statusCode === 400) {
        // Ошибки валидации - показываем рядом с полями
        try {
          const validationErrors = JSON.parse(error.technicalDetails || '{}')
          setFormErrors(validationErrors)
        } catch {
          // Если не удалось распарсить, показываем общую ошибку
          handleError(error)
        }
      } else {
        // Другие ошибки обрабатываем стандартно
        handleError(error)
      }
    }
  }
  
  return (
    <form onSubmit={handleSubmit}>
      <input name="email" />
      {formErrors.email && <span>{formErrors.email}</span>}
      <button type="submit">Отправить</button>
    </form>
  )
}
*/

// ============================================================================
// Пример 7: Получение системной сводки
// ============================================================================

export async function example7_SystemSummary() {
  try {
    // Получаем сводную информацию по всем базам данных системы
    const summary = await apiGet<SystemSummary>('/api/system/summary')
    
    console.log('Всего баз данных:', summary.total_databases)
    console.log('Всего загрузок:', summary.total_uploads)
    console.log('Завершенных загрузок:', summary.completed_uploads)
    console.log('Проваленных загрузок:', summary.failed_uploads)
    console.log('В процессе:', summary.in_progress_uploads)
    console.log('Общее количество номенклатуры:', summary.total_nomenclature)
    console.log('Общее количество контрагентов:', summary.total_counterparties)
    console.log('Последняя активность:', summary.last_activity)
    
    // Просматриваем детали по каждой загрузке
    summary.upload_details.forEach((upload) => {
      console.log(`Загрузка ${upload.name}:`)
      console.log(`  - UUID: ${upload.upload_uuid}`)
      console.log(`  - Статус: ${upload.status}`)
      console.log(`  - Номенклатура: ${upload.nomenclature_count}`)
      console.log(`  - Контрагенты: ${upload.counterparty_count}`)
      console.log(`  - Файл БД: ${upload.database_file}`)
    })
    
    return summary
  } catch (error) {
    console.error('Ошибка получения системной сводки:', error)
    throw error
  }
}

// Пример 8: Управление кешем системной сводки
export async function example8_SystemSummaryCacheManagement() {
  try {
    // Получаем статистику кеша
    const stats = await apiGet<CacheStats>('/api/system/summary/cache/stats')
    
    console.log('Статистика кеша:')
    console.log(`  - Попаданий (hits): ${stats.hits}`)
    console.log(`  - Промахов (misses): ${stats.misses}`)
    console.log(`  - Процент попаданий: ${(stats.hit_rate * 100).toFixed(2)}%`)
    console.log(`  - Есть данные: ${stats.has_data}`)
    console.log(`  - Устарели: ${stats.is_stale}`)
    if (stats.time_to_expiry) {
      console.log(`  - Время до истечения: ${stats.time_to_expiry}`)
    }
    
    // Если кеш устарел, инвалидируем его
    if (stats.is_stale && stats.has_data) {
      console.log('\nКеш устарел, инвалидируем...')
      const invalidateResult = await apiPost('/api/system/summary/cache/invalidate', {}) as { message?: string }
      console.log('Результат инвалидации:', invalidateResult.message || 'OK')
      
      // Получаем обновленную статистику
      const newStats = await apiGet<CacheStats>('/api/system/summary/cache/stats')
      console.log('Новая статистика:', newStats)
    }
    
    // Пример полной очистки кеша (раскомментируйте при необходимости)
    // console.log('\nОчищаем кеш...')
    // const clearResult = await apiPost('/api/system/summary/cache/clear', {})
    // console.log('Результат очистки:', clearResult.message)
    
    return stats
  } catch (error) {
    console.error('Ошибка управления кешем:', error)
    throw error
  }
}

// Типы для примеров
interface User {
  id: number
  name: string
  email: string
}

interface MyDataType {
  // Определите структуру данных
}

// Типы для системной сводки
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

// Типы для управления кешем
interface CacheStats {
  hits: number
  misses: number
  hit_rate: number
  has_data: boolean
  is_stale: boolean
  expiry: string
  time_to_expiry?: string
}

