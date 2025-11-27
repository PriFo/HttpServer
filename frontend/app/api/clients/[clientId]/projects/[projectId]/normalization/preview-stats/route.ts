import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger } from '@/lib/logger'
import { handleErrorWithDetails as handleError } from '@/lib/error-handler'

const API_BASE_URL = getBackendUrl()

export async function GET(
  request: Request,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  const startTime = performance.now()
  
  try {
    const resolvedParams = await params
    const { clientId, projectId } = resolvedParams

    // Валидация параметров
    if (!clientId || !projectId) {
      logger.error('Missing clientId or projectId in preview-stats route', {
        component: 'PreviewStatsAPI',
        clientId,
        projectId
      })
      
      return NextResponse.json(
        { error: 'Missing clientId or projectId' },
        { status: 400 }
      )
    }

    // Проверяем, что это числа
    const clientIdNum = parseInt(clientId, 10)
    const projectIdNum = parseInt(projectId, 10)
    
    if (isNaN(clientIdNum) || isNaN(projectIdNum)) {
      logger.error('Invalid clientId or projectId format', {
        component: 'PreviewStatsAPI',
        clientId,
        projectId,
        clientIdNum,
        projectIdNum
      })
      
      return NextResponse.json(
        { error: 'Invalid clientId or projectId. Expected numeric values.' },
        { status: 400 }
      )
    }

    // Получаем query параметры
    const { searchParams } = new URL(request.url)
    const normalizationType = searchParams.get('normalization_type') || 'both'

    logger.debug('Fetching preview stats from backend', {
      component: 'PreviewStatsAPI',
      clientId: clientIdNum,
      projectId: projectIdNum,
      normalizationType,
      backendUrl: API_BASE_URL
    })

    // Формируем URL с query параметрами
    const url = new URL(`${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/preview-stats`)
    url.searchParams.set('normalization_type', normalizationType)

    const backendStartTime = performance.now()
    const response = await fetch(url.toString(), {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      signal: AbortSignal.timeout(10000), // 10 секунд таймаут
    })
    const backendDuration = performance.now() - backendStartTime

    if (!response.ok) {
      let errorText = 'Unknown error'
      let errorData: { error?: string; message?: string } = {}
      
      try {
        errorText = await response.text()
        try {
          errorData = JSON.parse(errorText)
        } catch {
          // Если не JSON, используем текст
        }
      } catch (err) {
      logger.warn('Failed to read error response from backend', {
        component: 'PreviewStatsAPI',
        status: response.status,
        statusText: response.statusText
      })
      }

      const errorMessage = errorData.error || errorData.message || errorText || `HTTP ${response.status}`
      
      logger.error('Backend returned error in preview-stats', {
        component: 'PreviewStatsAPI',
        clientId: clientIdNum,
        projectId: projectIdNum,
        normalizationType,
        status: response.status,
        statusText: response.statusText,
        error: errorMessage,
        backendDuration: `${backendDuration.toFixed(2)}ms`
      })

      return NextResponse.json(
        { error: errorMessage },
        { status: response.status }
      )
    }

    const data = await response.json()
    const totalDuration = performance.now() - startTime

    logger.info('Preview stats fetched successfully', {
      component: 'PreviewStatsAPI',
      clientId: clientIdNum,
      projectId: projectIdNum,
      normalizationType,
      totalDatabases: data.total_databases,
      totalRecords: data.total_records,
      backendDuration: `${backendDuration.toFixed(2)}ms`,
      totalDuration: `${totalDuration.toFixed(2)}ms`
    })

    return NextResponse.json(data)
  } catch (error) {
    const duration = performance.now() - startTime
    const errorDetails = handleError(
      error,
      'PreviewStatsAPI',
      'GET',
      { duration: `${duration.toFixed(2)}ms` }
    )

    return NextResponse.json(
      { 
        error: errorDetails.message,
        code: errorDetails.code,
        ...(process.env.NODE_ENV === 'development' && errorDetails.context)
      },
      { status: errorDetails.statusCode || 500 }
    )
  }
}

