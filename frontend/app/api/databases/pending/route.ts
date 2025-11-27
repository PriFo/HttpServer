import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams
  const status = searchParams.get('status')
  const context = createApiContext('/api/databases/pending', 'GET', undefined, { status: status || undefined })
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/pending',
    async () => {
      const API_BASE = getBackendUrl()
      const url = status
        ? `${API_BASE}/api/databases/pending?status=${status}`
        : `${API_BASE}/api/databases/pending`

      logger.logRequest('GET', '/api/databases/pending', context)

      try {
        const response = await fetch(url, {
          cache: 'no-store',
          headers: {
            'Accept': 'application/json',
          },
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/databases/pending', response.status, duration, context)

        return handleBackendResponse(
          response,
          url,
          context,
          {
            errorMessage: 'Failed to fetch pending databases',
          }
        )
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, url, { ...context, duration })
      }
    },
    context
  )
}

