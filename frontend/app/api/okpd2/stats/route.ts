import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { withErrorHandler } from '@/lib/errors'
import { logger } from '@/lib/logger'

export const GET = withErrorHandler(async (request: NextRequest) => {
  const BACKEND_URL = getBackendUrl()
  const url = `${BACKEND_URL}/api/okpd2/stats`

  const startTime = Date.now()
  
  try {
    const response = await fetch(url, {
      cache: 'no-store',
    })

    const duration = Date.now() - startTime
    logger.logApiSuccess(url, 'GET', duration, { endpoint: '/api/okpd2/stats' })

    if (!response.ok) {
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        return NextResponse.json({
          total_codes: 0,
          max_level: 0,
        })
      }
      
      const error = new Error('Failed to fetch OKPD2 stats')
      logger.logApiError(url, 'GET', response.status, error, { endpoint: '/api/okpd2/stats' })
      
      return NextResponse.json(
        { error: 'Failed to fetch OKPD2 stats' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    logger.logApiError(url, 'GET', 0, error as Error, { endpoint: '/api/okpd2/stats' })
    throw error
  }
})

