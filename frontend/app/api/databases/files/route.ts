import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

const DEFAULT_EMPTY_RESPONSE = {
  success: false,
  files: [],
  total: 0,
  grouped: {
    main: [],
    service: [],
    uploaded: [],
    other: [],
  },
  summary: {
    main: 0,
    service: 0,
    uploaded: 0,
    other: 0,
  },
}

export async function GET() {
  const context = createApiContext('/api/databases/files', 'GET')
  const startTime = Date.now()

  return withLogging(
    'GET /api/databases/files',
    async () => {
      const backendUrl = getBackendUrl()
      const endpoint = `${backendUrl}/api/databases/files`

      logger.logRequest('GET', '/api/databases/files', context)

      try {
        const response = await fetch(endpoint, {
          method: 'GET',
          headers: {
            'Accept': 'application/json',
          },
          cache: 'no-store',
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/databases/files', response.status, duration, context)

        if (!response.ok) {
          logger.logBackendError(endpoint, response.status, undefined, context)
          // Если backend вернул 404, считаем что файлов просто нет
          if (response.status === 404) {
            return NextResponse.json(
              {
                ...DEFAULT_EMPTY_RESPONSE,
                success: true,
              },
              { status: 200 }
            )
          }

          return NextResponse.json(
            {
              ...DEFAULT_EMPTY_RESPONSE,
              error: 'Failed to fetch database files',
              status: response.status,
            },
            { status: response.status }
          )
        }

        const data = await response.json()
        return NextResponse.json(data)
      } catch (error) {
        const duration = Date.now() - startTime
        handleFetchError(error, endpoint, { ...context, duration })
        
        logger.warn('Returning empty database files list due to backend error', {
          ...context,
          duration,
          endpoint,
        })

        // Возвращаем структурированный ответ с пустыми данными и статусом 200,
        // чтобы UI мог продолжить работу даже без backend
        return NextResponse.json(
          {
            ...DEFAULT_EMPTY_RESPONSE,
            success: true,
            error: 'Failed to connect to backend',
            details: process.env.NODE_ENV === 'development' 
              ? (error instanceof Error ? error.message : String(error))
              : undefined,
          },
          { status: 200 }
        )
      }
    },
    context
  )
}

