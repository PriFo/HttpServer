import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError, validateRequired } from '@/lib/error-handler'

export const runtime = 'nodejs'

const DEFAULT_EMPTY_ANALYTICS = {
  file_path: '',
  database_type: 'unknown',
  total_size: 0,
  total_size_mb: 0,
  table_count: 0,
  total_rows: 0,
  table_stats: [],
  top_tables: [],
  analyzed_at: new Date().toISOString(),
}

export async function GET(request: NextRequest) {
  const url = new URL(request.url)
  const dbPath = url.searchParams.get('path')
  const context = createApiContext('/api/databases/analytics', 'GET', undefined, { path: dbPath || undefined })
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/analytics',
    async () => {
      try {
        // Валидация параметров
        validateRequired(dbPath, 'path', context)

        // Нормализуем путь: заменяем обратные слеши на прямые для кроссплатформенности
        const normalizedPath = dbPath!.replace(/\\/g, '/')
        
        const BACKEND_URL = getBackendUrl()
        const endpoint = `${BACKEND_URL}/api/databases/analytics?path=${encodeURIComponent(normalizedPath)}`

        logger.logRequest('GET', '/api/databases/analytics', { ...context, dbPath })

        const response = await fetch(endpoint, {
          cache: 'no-store',
          headers: {
            'Accept': 'application/json',
          },
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/databases/analytics', response.status, duration, context)

        // Для 404 возвращаем пустые данные
        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            allow404: true,
            defaultData: { ...DEFAULT_EMPTY_ANALYTICS, file_path: dbPath! },
            errorMessage: 'Failed to fetch database analytics',
          }
        )
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, `${getBackendUrl()}/api/databases/analytics`, {
          ...context,
          duration,
        })
      }
    },
    context
  )
}

