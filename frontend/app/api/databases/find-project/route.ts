import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError, validateRequired } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function GET(request: NextRequest) {
  const searchParams = request.nextUrl.searchParams
  const filePath = searchParams.get('file_path')
  const context = createApiContext('/api/databases/find-project', 'GET', undefined, { file_path: filePath || undefined })
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/find-project',
    async () => {
      try {
        // Валидация параметров
        validateRequired(filePath, 'file_path', context)

        const BACKEND_URL = getBackendUrl()
        const backendUrl = new URL(`${BACKEND_URL}/api/databases/find-project`)
        backendUrl.searchParams.append('file_path', filePath!)

        logger.logRequest('GET', '/api/databases/find-project', { ...context, filePath })

        const response = await fetch(backendUrl.toString(), {
          cache: 'no-store',
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/databases/find-project', response.status, duration, context)

        return handleBackendResponse(
          response,
          backendUrl.toString(),
          context,
          {
            errorMessage: 'Failed to find project',
          }
        )
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, `${getBackendUrl()}/api/databases/find-project`, {
          ...context,
          duration,
        })
      }
    },
    context
  )
}

