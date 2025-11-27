import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()
const SERVICE_DB_NAME = process.env.SERVICE_DB_NAME || 'Сервисная БД'

export async function GET() {
  try {
    // Проверяем доступность backend с таймаутом
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 5000) // 5 секунд таймаут
    
    let isBackendAvailable = false
    try {
      const healthResponse = await fetch(`${BACKEND_URL}/health`, {
        cache: 'no-store',
        signal: controller.signal,
      })
      isBackendAvailable = healthResponse.ok
      clearTimeout(timeoutId)
    } catch {
      clearTimeout(timeoutId)
      // Игнорируем ошибки health check
      isBackendAvailable = false
    }
    
    // Если backend доступен, пытаемся получить информацию о БД
    if (isBackendAvailable) {
      const dbController = new AbortController()
      const dbTimeoutId = setTimeout(() => dbController.abort(), 5000) // 5 секунд таймаут
      
      try {
        const response = await fetch(`${BACKEND_URL}/api/database/info`, {
          cache: 'no-store',
          signal: dbController.signal,
        })
        clearTimeout(dbTimeoutId)

        if (response.ok) {
          const responseData = await response.json()
          // Новый handler возвращает { info: {...} }, legacy возвращает напрямую объект
          const data = responseData.info || responseData
          
          // Извлекаем путь к БД
          const dbPath = data.current_db_path || data.path || ''
          
          // Получаем базовое имя файла
          let name = data.name
          if (!name && dbPath) {
            // Извлекаем имя файла из пути
            const pathParts = dbPath.replace(/\\/g, '/').split('/')
            name = pathParts[pathParts.length - 1] || SERVICE_DB_NAME
          }
          
          // Получаем размер и дату модификации из файловой системы (если путь доступен)
          let size = data.size || 0
          let modified_at = data.modified_at || ''
          
          // Если размер и дата не указаны, но есть путь, пытаемся получить их
          // (но это на сервере Next.js, поэтому мы не можем получить доступ к файлам)
          // Используем значения из ответа бэкенда или устанавливаем по умолчанию
          
          // Преобразуем структуру ответа для совместимости с фронтендом
          // Убеждаемся, что все числовые значения валидны (не NaN, не undefined)
          const safeSize = (typeof size === 'number' && !isNaN(size) && size >= 0) ? size : 0
          const safeModifiedAt = modified_at || ''
          
          const result = {
            name: name || SERVICE_DB_NAME,
            path: dbPath || data.path || '',
            size: safeSize,
            modified_at: safeModifiedAt,
            status: data.status || (dbPath ? 'connected' : 'disconnected'),
            stats: data.stats || {
              uploads_count: data.upload_stats?.total_uploads || data.upload_stats?.uploads_count || 0,
              catalogs_count: data.upload_stats?.total_catalogs || data.upload_stats?.catalogs_count || 0,
              items_count: data.upload_stats?.total_items || data.upload_stats?.items_count || 0
            },
            upload_stats: data.upload_stats,
            // Сохраняем дополнительные поля для обратной совместимости
            current_db_path: dbPath,
            current_normalized_db_path: data.current_normalized_db_path || ''
          }
          
          return NextResponse.json(result)
        }
      } catch (dbError) {
        clearTimeout(dbTimeoutId)
        // Игнорируем ошибки получения информации о БД
        // Логируем только если есть полезная информация
        if (dbError instanceof Error) {
          console.warn('Error fetching database info from backend:', dbError.message)
        }
      }
    }
    
    // Если backend недоступен или endpoint не найден, возвращаем информацию о сервисной БД
    return NextResponse.json({
      name: SERVICE_DB_NAME,
      status: isBackendAvailable ? 'connected' : 'disconnected',
      path: 'service.db',
      size: 0,
      modified_at: '',
      stats: {
        uploads_count: 0,
        catalogs_count: 0,
        items_count: 0
      }
    })
  } catch (error) {
    // Логируем ошибку только если она содержит полезную информацию
    if (error instanceof Error && error.message) {
      console.error('Error fetching database info:', error.message)
    } else if (error && typeof error === 'object' && Object.keys(error).length > 0) {
      console.error('Error fetching database info:', error)
    }
    // Возвращаем информацию о сервисной БД при ошибке
    return NextResponse.json({
      name: SERVICE_DB_NAME,
      status: 'disconnected',
      path: 'service.db',
      size: 0,
      modified_at: '',
      stats: {
        uploads_count: 0,
        catalogs_count: 0,
        items_count: 0
      }
    })
  }
}
