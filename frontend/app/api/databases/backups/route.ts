import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

const DEFAULT_EMPTY_RESPONSE = {
  backups: [] as unknown[],
  total: 0,
}

export async function GET() {
  const context = createApiContext('/api/databases/backups', 'GET')
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/backups',
    async () => {
      const backendUrl = getBackendUrl()
      const endpoint = `${backendUrl}/api/databases/backups`

      logger.logRequest('GET', '/api/databases/backups', context)

      try {
        const response = await fetch(endpoint, {
          method: 'GET',
          headers: {
            'Accept': 'application/json',
          },
          cache: 'no-store',
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/databases/backups', response.status, duration, context)

        // Для 404 возвращаем пустой список (бэкапов может не быть)
        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            allow404: true,
            defaultData: DEFAULT_EMPTY_RESPONSE,
            errorMessage: 'Failed to fetch backups',
          }
        )
      } catch (error) {
        const duration = Date.now() - startTime
        const errorResponse = handleFetchError(error, endpoint, { ...context, duration })
        
        // Возвращаем пустой список при ошибке, чтобы не ломать UI
        logger.warn('Returning empty backups list due to error', {
          ...context,
          duration,
          endpoint,
        })
        return NextResponse.json(DEFAULT_EMPTY_RESPONSE, { status: 200 })
      }
    },
    context
  )
}

