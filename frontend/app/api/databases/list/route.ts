import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function GET() {
  const context = createApiContext('/api/databases/list', 'GET')
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/list',
    async () => {
      const BACKEND_URL = getBackendUrl()
      const endpoint = `${BACKEND_URL}/api/databases/list`
      
      logger.logRequest('GET', '/api/databases/list', context)

      // Создаем контроллер для таймаута (7 секунд)
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 7000)

      try {
        const response = await fetch(endpoint, {
          cache: 'no-store',
          headers: {
            'Content-Type': 'application/json',
          },
          signal: controller.signal,
        })

        clearTimeout(timeoutId)
        const duration = Date.now() - startTime

        logger.logResponse('GET', '/api/databases/list', response.status, duration, context)

        // Для этого endpoint возвращаем пустой список при ошибках, чтобы UI не ломался
        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            allow404: true,
            defaultData: [],
            errorMessage: 'Failed to fetch databases list',
          }
        )
      } catch (fetchError) {
        clearTimeout(timeoutId)
        const duration = Date.now() - startTime

        // Для этого endpoint возвращаем пустой список при ошибках сети
        if (fetchError instanceof Error && fetchError.name === 'AbortError') {
          logger.warn('Request timeout while fetching databases list', {
            ...context,
            duration,
            endpoint,
          })
          return NextResponse.json([])
        }

        // Для сетевых ошибок также возвращаем пустой список
        const errorResponse = handleFetchError(fetchError, endpoint, { ...context, duration })
        // Но для этого endpoint мы возвращаем пустой список вместо ошибки
        if (errorResponse.status >= 500) {
          logger.warn('Returning empty list due to backend error', {
            ...context,
            duration,
            endpoint,
          })
          return NextResponse.json([])
        }
        return errorResponse
      }
    },
    context
  ).catch((error) => {
    // Финальная обработка - всегда возвращаем пустой список для этого endpoint
    logger.error('Unexpected error in GET /api/databases/list, returning empty list', context, error)
    return NextResponse.json([])
  })
}
