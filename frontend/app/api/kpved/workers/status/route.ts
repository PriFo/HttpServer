import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { withErrorHandler } from '@/lib/errors'
import { logger } from '@/lib/logger'

// Ensure Node.js runtime for API routes
export const runtime = 'nodejs'

export const GET = withErrorHandler(async (request: NextRequest) => {
  const startTime = Date.now()
  const backendUrl = getBackendUrl()
  const url = `${backendUrl}/api/kpved/workers/status`
  
  try {
    const response = await fetch(url, {
      cache: 'no-store',
      headers: {
        'Accept': 'application/json',
      },
    })

    const duration = Date.now() - startTime

    if (!response.ok) {
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        logger.info('Workers status endpoint not found, returning empty data', {
          endpoint: '/api/kpved/workers/status',
          duration,
        })
        return NextResponse.json({
          workers: [],
          total: 0,
          active: 0,
        })
      }
      
      logger.logApiError(url, 'GET', response.status, new Error('Failed to fetch workers status'), {
        endpoint: '/api/kpved/workers/status',
        duration,
      })
      
      return NextResponse.json(
        { error: 'Failed to fetch workers status', status: response.status },
        { status: response.status }
      )
    }

    const data = await response.json()
    
    logger.logApiSuccess(url, 'GET', duration, {
      endpoint: '/api/kpved/workers/status',
    })
    
    return NextResponse.json(data)
  } catch (error) {
    const duration = Date.now() - startTime
    
    logger.logApiError(url, 'GET', 0, error as Error, {
      endpoint: '/api/kpved/workers/status',
      duration,
    })
    
    // Возвращаем пустые данные вместо ошибки для лучшего UX
    return NextResponse.json({
      workers: [],
      total: 0,
      active: 0,
    })
  }
})

