import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { withErrorHandler } from '@/lib/errors'
import { logger } from '@/lib/logger'

export const runtime = 'nodejs'

export const GET = withErrorHandler(async (
  request: NextRequest,
  { params }: { params: Promise<{ dbname: string }> }
) => {
  const startTime = Date.now()
  const BACKEND_URL = getBackendUrl()
  
  try {
    const { dbname } = await params
    if (!dbname) {
      return NextResponse.json(
        { error: 'Database name is required' },
        { status: 400 }
      )
    }

    // Получаем путь из query параметра
    const url = new URL(request.url)
    const dbPath = url.searchParams.get('path')
    
    // Если путь не указан в query, используем имя из параметра
    const finalPath = dbPath || dbname
    
    if (!finalPath) {
      return NextResponse.json(
        { error: 'Database path is required' },
        { status: 400 }
      )
    }
    
    // Используем путь из query параметра или имя файла
    const response = await fetch(`${BACKEND_URL}/api/databases/analytics?path=${encodeURIComponent(finalPath)}`, {
      cache: 'no-store',
      headers: {
        'Accept': 'application/json',
      },
    })

    if (!response.ok) {
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        return NextResponse.json({
          total_records: 0,
          total_size: 0,
          tables: [],
          last_analyzed: null,
        })
      }
      
      let errorMessage = 'Failed to fetch database analytics'
      const contentType = response.headers.get('content-type')
      
      try {
        // Пытаемся получить текст ответа (можно прочитать только один раз)
        const responseText = await response.text()
        
        // Пытаемся распарсить как JSON
        if (contentType?.includes('application/json')) {
          try {
            const errorData = JSON.parse(responseText)
            errorMessage = errorData.error || errorMessage
          } catch {
            // Если не JSON, используем текст как есть
            if (responseText) {
              errorMessage = responseText.length > 200 
                ? responseText.substring(0, 200) + '...' 
                : responseText
            }
          }
        } else if (responseText) {
          errorMessage = responseText.length > 200 
            ? responseText.substring(0, 200) + '...' 
            : responseText
        }
      } catch (parseError) {
        logger.warn('Error parsing error response', {
          component: 'DatabaseAnalyticsApi',
          dbname: finalPath,
          error: parseError instanceof Error ? parseError.message : String(parseError),
        })
      }
      
      logger.logApiError(`${BACKEND_URL}/api/databases/analytics`, 'GET', response.status, new Error(errorMessage), {
        endpoint: '/api/databases/analytics/[dbname]',
        dbname: finalPath,
        statusText: response.statusText,
      })
      
      return NextResponse.json(
        { error: errorMessage },
        { status: response.status }
      )
    }

    const data = await response.json()
    const duration = Date.now() - startTime
    logger.logApiSuccess(`${BACKEND_URL}/api/databases/analytics`, 'GET', duration, {
      endpoint: '/api/databases/analytics/[dbname]',
      dbname: finalPath,
    })
    
    return NextResponse.json(data)
  } catch (error) {
    const duration = Date.now() - startTime
    const { dbname } = await params
    const url = new URL(request.url)
    const dbPath = url.searchParams.get('path')
    const finalPath = dbPath || dbname || 'unknown'
    
    logger.logApiError(`${BACKEND_URL}/api/databases/analytics`, 'GET', 0, error as Error, {
      endpoint: '/api/databases/analytics/[dbname]',
      dbname: finalPath,
      duration,
    })
    throw error
  }
})

