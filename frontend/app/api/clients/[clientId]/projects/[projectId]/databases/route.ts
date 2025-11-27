import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { withErrorHandler } from '@/lib/errors'
import { logger } from '@/lib/logger'

export const runtime = 'nodejs'

export const GET = withErrorHandler(async (
  request: Request,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) => {
  const startTime = Date.now()
  const { clientId, projectId } = await params
  const { searchParams } = new URL(request.url)
  const activeOnly = searchParams.get('active_only') === 'true'

  const API_BASE_URL = getBackendUrl()

  const queryParams = new URLSearchParams()
  if (activeOnly) queryParams.append('active_only', 'true')

  const url = `${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
  
  try {
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    })

    if (!response.ok) {
      if (response.status === 404) {
        return NextResponse.json({ databases: [], total: 0 })
      }
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    const data = await response.json()
    const duration = Date.now() - startTime
    logger.logApiSuccess(url, 'GET', duration, {
      endpoint: '/api/clients/[clientId]/projects/[projectId]/databases',
      clientId,
      projectId,
      activeOnly,
    })
    
    return NextResponse.json(data)
  } catch (error) {
    const duration = Date.now() - startTime
    logger.logApiError(url, 'GET', 0, error as Error, {
      endpoint: '/api/clients/[clientId]/projects/[projectId]/databases',
      clientId,
      projectId,
      duration,
    })
    // Возвращаем пустые данные вместо ошибки для лучшего UX
    return NextResponse.json({ databases: [], total: 0 })
  }
})

export const POST = withErrorHandler(async (
  request: Request,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) => {
  const startTime = Date.now()
  const API_BASE_URL = getBackendUrl()
  
  try {
    const { clientId, projectId } = await params
    const contentType = request.headers.get('content-type') || ''
    const requestID = request.headers.get('x-request-id') || `req_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
    const clientIP = request.headers.get('x-forwarded-for')?.split(',')[0] || request.headers.get('x-real-ip') || 'unknown'

    // Проверяем, является ли запрос multipart/form-data (загрузка файла)
    if (contentType.includes('multipart/form-data')) {
      const uploadStartTime = Date.now()
      const contentLength = request.headers.get('content-length')
      const fileSizeMB = contentLength ? (parseInt(contentLength) / 1024 / 1024).toFixed(2) : 'unknown'
      
      logger.info('Proxying multipart/form-data request', {
        component: 'DatabaseUploadApi',
        requestID,
        clientId,
        projectId,
        clientIP,
        contentType,
        contentLength,
        fileSizeMB,
      })
      
      // Используем потоковую передачу тела запроса напрямую
      // Это сохраняет boundary и предотвращает ошибку "Unexpected end of multipart data"
      if (!request.body) {
        logger.error('Request body is null', {
          component: 'DatabaseUploadApi',
          requestID,
          clientId,
          projectId,
        })
        return NextResponse.json(
          { error: 'No request body received' },
          { status: 400 }
        )
      }
      
      const timeoutMs = contentLength && parseInt(contentLength) > 100 * 1024 * 1024 
        ? 15 * 60 * 1000 // 15 мин для файлов > 100MB
        : 10 * 60 * 1000 // 10 мин для остальных
      
      const controller = new AbortController()
      const timeoutId = setTimeout(() => {
        logger.warn('Request timeout', {
          component: 'DatabaseUploadApi',
          requestID,
          clientId,
          projectId,
          timeoutSeconds: timeoutMs / 1000,
        })
        controller.abort()
      }, timeoutMs)
      
      try {
        logger.debug('Streaming request body to backend', {
          component: 'DatabaseUploadApi',
          requestID,
          timeoutSeconds: timeoutMs / 1000,
        })
        
        // Передаем тело запроса напрямую с сохранением всех заголовков
        const response = await fetch(`${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`, {
          method: 'POST',
          body: request.body,
          signal: controller.signal,
          headers: {
            'Content-Type': contentType, // Сохраняем оригинальный Content-Type с boundary
            'Content-Length': contentLength || '',
            'X-Request-ID': requestID,
          },
          // Используем дуплекс для потоковой передачи
          duplex: 'half' as any,
        } as RequestInit)
        
        clearTimeout(timeoutId)
        const backendResponseTime = ((Date.now() - uploadStartTime) / 1000).toFixed(2)
        logger.info('Backend response received', {
          component: 'DatabaseUploadApi',
          requestID,
          status: response.status,
          responseTime: backendResponseTime,
        })

        if (!response.ok) {
          let errorData: { error?: string; message?: string } = {}
          let errorText = ''
          
          // Пытаемся прочитать тело ответа как текст
          try {
            errorText = await response.text()
            // Пытаемся распарсить как JSON
            try {
              errorData = JSON.parse(errorText)
            } catch {
              // Если не JSON, используем текст как есть
              if (errorText) {
                errorData = { error: errorText }
              }
            }
          } catch {
            errorText = `HTTP error! status: ${response.status}`
            errorData = { error: errorText }
          }
          
          const errorMessage = errorData.error || errorData.message || errorText || `HTTP error! status: ${response.status}`
          
          logger.logApiError(
            `${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`,
            'POST',
            response.status,
            new Error(errorMessage),
            {
              component: 'DatabaseUploadApi',
              requestID,
              clientId,
              projectId,
              responseTime: backendResponseTime,
              errorData,
              errorText: errorText.substring(0, 500),
            }
          )
          
          return NextResponse.json(
            { 
              error: errorMessage,
              status: response.status,
              details: response.status === 500 ? 'Internal server error. Please check server logs for details.' : undefined
            },
            { status: response.status }
          )
        }

        const data = await response.json()
        const uploadDuration = ((Date.now() - uploadStartTime) / 1000).toFixed(2)
        
        logger.logApiSuccess(
          `${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`,
          'POST',
          Date.now() - uploadStartTime,
          {
            component: 'DatabaseUploadApi',
            requestID,
            clientId,
            projectId,
            uploadDuration,
            suggested_name: data.suggested_name,
            file_path: data.file_path,
            file_size_mb: fileSizeMB,
          }
        )
        
        return NextResponse.json(data, { status: response.status })
      } catch (fetchError: unknown) {
        clearTimeout(timeoutId)
        const err = fetchError as { name?: string; message?: string };
        if (err.name === 'AbortError') {
          logger.warn('Request timeout', {
            component: 'DatabaseUploadApi',
            requestID,
            clientId,
            projectId,
            timeoutMinutes: 10,
          })
          return NextResponse.json(
            { error: 'Request timeout. The file may be too large or the server is not responding.' },
            { status: 408 }
          )
        }
        
        logger.logApiError(
          `${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`,
          'POST',
          0,
          fetchError as Error,
          {
            component: 'DatabaseUploadApi',
            requestID,
            clientId,
            projectId,
          }
        )
        
        const errorMessage = err.message || 'Failed to upload file to backend'
        return NextResponse.json(
          { error: errorMessage },
          { status: 500 }
        )
      }
    }

    // Обычный JSON запрос
    const body = await request.json()

    const response = await fetch(`${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      if (response.status === 404) {
        const errorMsg = 'Backend endpoint not found. Please restart the backend server.'
        return NextResponse.json(
          { error: errorMsg },
          { status: 503 }
        )
      }
      const errorData = await response.json().catch(() => ({}))
      const errorText = await response.text().catch(() => '')
      
      logger.logApiError(
        `${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`,
        'POST',
        response.status,
        new Error(errorData.error || errorText || `HTTP error! status: ${response.status}`),
        {
          component: 'DatabaseUploadApi',
          clientId,
          projectId,
          errorData,
          errorText,
        }
      )
      
      return NextResponse.json(
        { error: errorData.error || errorText || `HTTP error! status: ${response.status}` },
        { status: response.status }
      )
    }

    const data = await response.json()
    const duration = Date.now() - startTime
    
    logger.logApiSuccess(
      `${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/databases`,
      'POST',
      duration,
      {
        component: 'DatabaseUploadApi',
        clientId,
        projectId,
      }
    )
    
    return NextResponse.json(data, { status: 201 })
  } catch (error) {
    const duration = Date.now() - startTime
    
    logger.logApiError(
      `${getBackendUrl()}/api/clients/${(await params).clientId}/projects/${(await params).projectId}/databases`,
      'POST',
      0,
      error as Error,
      {
        component: 'DatabaseUploadApi',
        clientId: (await params).clientId,
        projectId: (await params).projectId,
        duration,
      }
    )
    
    throw error
  }
})


