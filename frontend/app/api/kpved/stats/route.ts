import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { withErrorHandler, createErrorResponse } from '@/lib/errors'
import { logger } from '@/lib/logger'

export const GET = withErrorHandler(async (request: NextRequest) => {
  const BACKEND_URL = getBackendUrl()
  const url = `${BACKEND_URL}/api/kpved/stats`

  const startTime = Date.now()
  
  try {
    const response = await fetch(url, {
      cache: 'no-store',
    })

    const duration = Date.now() - startTime
    logger.logApiSuccess(url, 'GET', duration, { endpoint: '/api/kpved/stats' })

    if (!response.ok) {
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        return NextResponse.json({
          total_codes: 0,
          max_level: 0,
        })
      }
      
      const errorText = await response.text()
      const error = new Error(errorText || 'Failed to fetch KPVED stats')
      logger.logApiError(url, 'GET', response.status, error, { endpoint: '/api/kpved/stats' })
      
      return NextResponse.json(
        { error: errorText || 'Failed to fetch KPVED stats' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    logger.logApiError(url, 'GET', 0, error as Error, { endpoint: '/api/kpved/stats' })
    throw error
  }
})
