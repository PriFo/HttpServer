import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError, validateRequired } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function GET(request: NextRequest) {
  const url = new URL(request.url)
  const dbPath = url.searchParams.get('path')
  const context = createApiContext('/api/databases/history', 'GET', undefined, { path: dbPath || undefined })
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/history',
    async () => {
      try {
        // Валидация параметров
        validateRequired(dbPath, 'path', context)

        const BACKEND_URL = getBackendUrl()
        
        // Проксируем запрос к бэкенду с путем в URL
        // Бэкенд ожидает имя БД в пути: /api/databases/history/{dbName}
        // Извлекаем имя файла из пути, если путь полный
        const dbName = dbPath!.includes('/') || dbPath!.includes('\\') 
          ? dbPath!.split(/[/\\]/).pop() || dbPath! 
          : dbPath!
        
        const endpoint = `${BACKEND_URL}/api/databases/history/${encodeURIComponent(dbName)}`

        logger.logRequest('GET', '/api/databases/history', { ...context, dbName })

        const response = await fetch(endpoint, {
          cache: 'no-store',
          headers: {
            'Accept': 'application/json',
          },
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/databases/history', response.status, duration, context)

        // Для 404 возвращаем пустую историю
        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            allow404: true,
            defaultData: { history: [] },
            errorMessage: 'Failed to fetch database history',
          }
        )
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, `${getBackendUrl()}/api/databases/history`, {
          ...context,
          duration,
        })
      }
    },
    context
  )
}

