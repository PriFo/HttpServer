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
    // Бэкенд ожидает формат: /api/databases/history?path=...
    const response = await fetch(`${BACKEND_URL}/api/databases/history?path=${encodeURIComponent(finalPath)}`, {
      cache: 'no-store',
      headers: {
        'Accept': 'application/json',
      },
    })

    if (!response.ok) {
      // Для 404 возвращаем пустую историю вместо ошибки
      if (response.status === 404) {
        return NextResponse.json({ history: [] })
      }
      
      let errorMessage = 'Failed to fetch database history'
      const contentType = response.headers.get('content-type')
      
      try {
        const responseText = await response.text()
        
        if (contentType?.includes('application/json')) {
          try {
            const errorData = JSON.parse(responseText)
            errorMessage = errorData.error || errorMessage
          } catch {
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
          component: 'DatabaseHistoryApi',
          dbname: finalPath,
          error: parseError instanceof Error ? parseError.message : String(parseError),
        })
      }
      
      return NextResponse.json(
        { error: errorMessage },
        { status: response.status }
      )
    }

    const data = await response.json()
    const duration = Date.now() - startTime
    logger.logApiSuccess(`${BACKEND_URL}/api/databases/history`, 'GET', duration, {
      endpoint: '/api/databases/history/[dbname]',
      dbname: finalPath,
    })
    
    return NextResponse.json(data)
  } catch (error) {
    const duration = Date.now() - startTime
    const { dbname } = await params
    const url = new URL(request.url)
    const dbPath = url.searchParams.get('path')
    const finalPath = dbPath || dbname || 'unknown'
    
    logger.logApiError(`${BACKEND_URL}/api/databases/history`, 'GET', 0, error as Error, {
      endpoint: '/api/databases/history/[dbname]',
      dbname: finalPath,
      duration,
    })
    throw error
  }
})

