import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, isNetworkError } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function GET() {
  const context = createApiContext('/api/clients', 'GET')
  const startTime = Date.now()

  return withLogging(
    'GET /api/clients',
    async () => {
      const BACKEND_URL = getBackendUrl()
      const endpoint = `${BACKEND_URL}/api/clients`

      logger.logRequest('GET', '/api/clients', context)

      try {
        const data = await fetchJsonServer(endpoint, {
          timeout: QUALITY_TIMEOUTS.STANDARD,
          cache: 'no-store',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        // Обрабатываем ответ - он может быть массивом или объектом с полями clients/total
        let result: unknown[]
        if (Array.isArray(data)) {
          result = data
        } else if (data && typeof data === 'object' && 'clients' in data) {
          result = (data as { clients: unknown[] }).clients || []
        } else {
          result = []
        }

        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/clients', 200, duration, {
          ...context,
          count: result.length,
        })

        return NextResponse.json(result)
      } catch (error) {
        const duration = Date.now() - startTime
        const isNetwork = isNetworkError(error)
        
        if (!isNetwork) {
          logger.error('Error fetching clients', { ...context, duration }, error instanceof Error ? error : undefined)
        } else {
          logger.debug('Network error fetching clients (backend may be down)', { ...context, duration })
        }
        
        // Возвращаем пустой массив с информацией о fallback
        // Страница клиентов должна показать информативное сообщение
        return NextResponse.json([], { status: 200 })
      }
    },
    context
  )
}

export async function POST(request: Request) {
  const context = createApiContext('/api/clients', 'POST')
  const startTime = Date.now()

  return withLogging(
    'POST /api/clients',
    async () => {
      const API_BASE_URL = getBackendUrl()
      const endpoint = `${API_BASE_URL}/api/clients`
      
      let body: unknown = {}
      try {
        body = await request.json()
      } catch {
        // Body может быть пустым
      }

      logger.logRequest('POST', '/api/clients', { ...context, hasBody: !!body })

      try {
        const response = await fetch(endpoint, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(body),
        })

        const duration = Date.now() - startTime
        logger.logResponse('POST', '/api/clients', response.status, duration, context)

        if (!response.ok) {
          if (response.status === 404) {
            const errorMsg = 'Backend endpoint not found. Please restart the backend server to apply changes.'
            logger.error('Backend endpoint not found', { ...context, duration, endpoint })
            return NextResponse.json(
              { error: errorMsg },
              { status: 503 }
            )
          }
          
          const errorText = await response.text().catch(() => '')
          logger.logBackendError(endpoint, response.status, errorText, context)
          
          let errorData: { error?: string } = {}
          try {
            errorData = JSON.parse(errorText)
          } catch {
            errorData = { error: errorText }
          }
          
          return NextResponse.json(
            { error: errorData.error || errorText || `HTTP error! status: ${response.status}` },
            { status: response.status }
          )
        }

        const data = await response.json()
        return NextResponse.json(data, { status: 201 })
      } catch (error) {
        const duration = Date.now() - startTime
        return handleFetchError(error, endpoint, { ...context, duration })
      }
    },
    context
  )
}

