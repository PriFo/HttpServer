import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { withErrorHandler } from '@/lib/errors'
import { logger } from '@/lib/logger'

export const runtime = 'nodejs'

export const GET = withErrorHandler(async (
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string }> }
) => {
  const startTime = Date.now()
  const { clientId } = await params
  const BACKEND_URL = getBackendUrl()
  const endpoint = `${BACKEND_URL}/api/clients/${clientId}/databases`
  
  try {
    const response = await fetch(endpoint, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error')
      
      logger.logApiError(
        endpoint,
        'GET',
        response.status,
        new Error(errorText),
        {
          endpoint: '/api/clients/[clientId]/databases',
          clientId,
        }
      )
      
      return NextResponse.json(
        { error: 'Failed to fetch databases', details: errorText },
        { status: response.status }
      )
    }

    const data = await response.json()
    const duration = Date.now() - startTime
    
    logger.logApiSuccess(
      endpoint,
      'GET',
      duration,
      {
        endpoint: '/api/clients/[clientId]/databases',
        clientId,
      }
    )
    
    // Бэкенд возвращает массив напрямую, возвращаем как есть
    // Фронтенд уже обрабатывает оба формата (массив и объект с databases)
    return NextResponse.json(data)
  } catch (error) {
    const duration = Date.now() - startTime
    
    logger.logApiError(
      endpoint,
      'GET',
      0,
      error as Error,
      {
        endpoint: '/api/clients/[clientId]/databases',
        clientId,
        duration,
      }
    )
    
    throw error
  }
})

