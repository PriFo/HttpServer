import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext } from '@/lib/logger'

const API_BASE_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  const context = createApiContext('/api/counterparties/normalization/status', 'GET')
  const startTime = Date.now()

  try {
    // Получаем параметры из query string
    const { searchParams } = new URL(request.url)
    const clientId = searchParams.get('client_id')
    const projectId = searchParams.get('project_id')

    // Если есть client_id и project_id, используем специальный эндпоинт
    if (clientId && projectId) {
      const clientIdNum = parseInt(clientId, 10)
      const projectIdNum = parseInt(projectId, 10)
      
      // Валидация ID
      if (isNaN(clientIdNum) || clientIdNum <= 0) {
        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/counterparties/normalization/status', 400, duration, context)
        return NextResponse.json(
          {
            is_running: false,
            processed: 0,
            total: 0,
            error: 'Invalid client_id: must be a positive integer',
          },
          { status: 400 }
        )
      }
      
      if (isNaN(projectIdNum) || projectIdNum <= 0) {
        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/counterparties/normalization/status', 400, duration, context)
        return NextResponse.json(
          {
            is_running: false,
            processed: 0,
            total: 0,
            error: 'Invalid project_id: must be a positive integer',
          },
          { status: 400 }
        )
      }

      const url = `${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/status`
      
      try {
        const response = await fetch(url, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          cache: 'no-store',
          signal: AbortSignal.timeout(10000),
        })

        if (!response.ok) {
          let errorText = 'Unknown error'
          try {
            const errorData = await response.json()
            errorText = errorData.error || errorData.message || JSON.stringify(errorData)
          } catch {
            errorText = await response.text().catch(() => 'Unknown error')
          }
          
          const duration = Date.now() - startTime
          logger.logResponse('GET', '/api/counterparties/normalization/status', response.status, duration, context)
          return NextResponse.json(
            {
              is_running: false,
              processed: 0,
              total: 0,
              error: errorText,
              status: response.status,
            },
            { status: response.status }
          )
        }

        const data = await response.json()
        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/counterparties/normalization/status', 200, duration, context)
        return NextResponse.json(data)
      } catch (fetchError) {
        const isConnectionError = fetchError instanceof Error && (
          fetchError.name === 'AbortError' ||
          fetchError.message.includes('fetch failed') ||
          fetchError.message.includes('ECONNREFUSED') ||
          (fetchError as any)?.code === 'ECONNREFUSED'
        )

        if (fetchError instanceof Error && fetchError.name === 'AbortError') {
          const duration = Date.now() - startTime
          logger.warn('Counterparties normalization status: timeout', {
            ...context,
            clientId: clientIdNum,
            projectId: projectIdNum,
            duration,
          })
          logger.logResponse('GET', '/api/counterparties/normalization/status', 504, duration, context)
          return NextResponse.json(
            {
              is_running: false,
              processed: 0,
              total: 0,
              error: 'Превышено время ожидания ответа от сервера (10 секунд)',
            },
            { status: 504 }
          )
        }

        if (isConnectionError) {
          const duration = Date.now() - startTime
          // Логируем как предупреждение, не как ошибку
          logger.warn('Counterparties normalization status: backend unavailable', {
            ...context,
            clientId: clientIdNum,
            projectId: projectIdNum,
            error: fetchError instanceof Error ? fetchError.message : 'Unknown error',
            backendUrl: API_BASE_URL,
            note: 'Backend server appears to be unavailable. This is expected if the server is not running.',
          })
          logger.logResponse('GET', '/api/counterparties/normalization/status', 503, duration, context)
          return NextResponse.json(
            {
              is_running: false,
              processed: 0,
              total: 0,
              error: 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен.',
              backendUrl: API_BASE_URL,
            },
            { status: 503 }
          )
        }

        throw fetchError
      }
    }

    // Fallback: возвращаем пустой статус, если нет параметров
    const duration = Date.now() - startTime
    logger.logResponse('GET', '/api/counterparties/normalization/status', 200, duration, context)
    return NextResponse.json({
      is_running: false,
      processed: 0,
      total: 0,
      currentStep: 'Ожидание запуска',
      progress: 0,
    })
  } catch (error) {
    const duration = Date.now() - startTime
    logger.error('Counterparties normalization status: unexpected error', {
      ...context,
      error: error instanceof Error ? error.message : 'Unknown error',
      duration,
    }, error instanceof Error ? error : undefined)
    logger.logResponse('GET', '/api/counterparties/normalization/status', 500, duration, context)
    return NextResponse.json(
      {
        is_running: false,
        processed: 0,
        total: 0,
        error: error instanceof Error ? error.message : 'Unknown error',
      },
      { status: 500 }
    )
  }
}

