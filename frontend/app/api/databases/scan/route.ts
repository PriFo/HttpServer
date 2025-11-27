import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function POST(request: NextRequest) {
  const context = createApiContext('/api/databases/scan', 'POST')
  const startTime = Date.now()

  return withLogging(
    'POST /api/databases/scan',
    async () => {
      let body: unknown = {}
      try {
        body = await request.json()
      } catch {
        // Body может быть пустым
        logger.debug('Empty or invalid request body, using empty object', context)
      }

      const API_BASE = getBackendUrl()
      const endpoint = `${API_BASE}/api/databases/scan`

      logger.logRequest('POST', '/api/databases/scan', { ...context, hasBody: !!body })

      try {
        const response = await fetch(endpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(body),
        })

        const duration = Date.now() - startTime
        logger.logResponse('POST', '/api/databases/scan', response.status, duration, context)

        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            errorMessage: 'Failed to scan databases',
          }
        )
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, endpoint, { ...context, duration })
      }
    },
    context
  )
}

