import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { handleBackendResponse, handleFetchError } from '@/lib/error-handler'

export const runtime = 'nodejs'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ clientId: string }> }
) {
  const { clientId } = await params
  const context = createApiContext('/api/clients/[clientId]', 'GET', { clientId })
  const startTime = Date.now()

  return withLogging(
    'GET /api/clients/[clientId]',
    async () => {
      const API_BASE_URL = getBackendUrl()
      const endpoint = `${API_BASE_URL}/api/clients/${clientId}`

      logger.logRequest('GET', `/api/clients/${clientId}`, context)

      try {
        const response = await fetch(endpoint, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          cache: 'no-store',
        })

        const duration = Date.now() - startTime
        logger.logResponse('GET', `/api/clients/${clientId}`, response.status, duration, context)

        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            errorMessage: 'Failed to fetch client',
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

export async function PUT(
  request: Request,
  { params }: { params: Promise<{ clientId: string }> }
) {
  const { clientId } = await params
  const context = createApiContext('/api/clients/[clientId]', 'PUT', { clientId })
  const startTime = Date.now()

  return withLogging(
    'PUT /api/clients/[clientId]',
    async () => {
      const body = await request.json().catch(() => ({}))
      const API_BASE_URL = getBackendUrl()
      const endpoint = `${API_BASE_URL}/api/clients/${clientId}`

      logger.logRequest('PUT', `/api/clients/${clientId}`, { ...context, hasBody: !!body })

      try {
        const response = await fetch(endpoint, {
          method: 'PUT',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify(body),
        })

        const duration = Date.now() - startTime
        logger.logResponse('PUT', `/api/clients/${clientId}`, response.status, duration, context)

        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            errorMessage: 'Failed to update client',
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

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ clientId: string }> }
) {
  const { clientId } = await params
  const context = createApiContext('/api/clients/[clientId]', 'DELETE', { clientId })
  const startTime = Date.now()

  return withLogging(
    'DELETE /api/clients/[clientId]',
    async () => {
      const API_BASE_URL = getBackendUrl()
      const endpoint = `${API_BASE_URL}/api/clients/${clientId}`

      logger.logRequest('DELETE', `/api/clients/${clientId}`, context)

      try {
        const response = await fetch(endpoint, {
          method: 'DELETE',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        const duration = Date.now() - startTime
        logger.logResponse('DELETE', `/api/clients/${clientId}`, response.status, duration, context)

        return handleBackendResponse(
          response,
          endpoint,
          context,
          {
            errorMessage: 'Failed to delete client',
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

