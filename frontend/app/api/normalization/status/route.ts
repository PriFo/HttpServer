import { NextRequest, NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/api-config';
import { logger, createApiContext } from '@/lib/logger';

const API_BASE_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  const context = createApiContext('/api/normalization/status', 'GET')
  const startTime = Date.now()

  try {
    // Создаем контроллер для таймаута (7 секунд)
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 7000)

    try {
      // Проксируем запрос к существующему эндпоинту нормализации на бэкенде
      const backendResponse = await fetch(`${API_BASE_URL}/api/normalization/status`, {
        cache: 'no-store',
        signal: controller.signal,
        headers: {
          'Content-Type': 'application/json',
        },
      });

      clearTimeout(timeoutId)
      
      if (!backendResponse.ok) {
        const duration = Date.now() - startTime
        logger.logResponse('GET', '/api/normalization/status', backendResponse.status, duration, context)
        return NextResponse.json(
          { error: 'Failed to fetch normalization status' },
          { status: backendResponse.status }
        );
      }
      
      const data = await backendResponse.json();
      const duration = Date.now() - startTime
      logger.logResponse('GET', '/api/normalization/status', 200, duration, context)
      return NextResponse.json(data);
    } catch (fetchError) {
      clearTimeout(timeoutId)
      
      // Обработка сетевых ошибок
      if (fetchError instanceof Error) {
        const isConnectionError = fetchError.message.includes('fetch failed') || 
            fetchError.message.includes('Failed to fetch') ||
            fetchError.message.includes('NetworkError') ||
            fetchError.message.includes('ECONNREFUSED') ||
            (fetchError as any)?.code === 'ECONNREFUSED'

        if (fetchError.name === 'AbortError') {
          const duration = Date.now() - startTime
          logger.warn('Normalization status timeout', {
            ...context,
            duration,
          })
          logger.logResponse('GET', '/api/normalization/status', 504, duration, context)
          return NextResponse.json(
            { error: 'Превышено время ожидания ответа от сервера' },
            { status: 504 }
          )
        }
        
        if (isConnectionError) {
          const duration = Date.now() - startTime
          // Логируем как предупреждение, не как ошибку, так как бэкенд может быть просто не запущен
          logger.warn('Normalization status: backend unavailable', {
            ...context,
            error: fetchError.message,
            backendUrl: API_BASE_URL,
            note: 'Backend server appears to be unavailable. This is expected if the server is not running.',
          })
          logger.logResponse('GET', '/api/normalization/status', 503, duration, context)
          return NextResponse.json(
            { 
              error: 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999.',
              backendUrl: API_BASE_URL,
            },
            { status: 503 }
          )
        }
      }
      throw fetchError
    }
  } catch (error) {
    const duration = Date.now() - startTime
    logger.error('Normalization status: unexpected error', {
      ...context,
      error: error instanceof Error ? error.message : 'Unknown error',
      duration,
    }, error instanceof Error ? error : undefined)
    logger.logResponse('GET', '/api/normalization/status', 500, duration, context)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Internal server error' },
      { status: 500 }
    );
  }
}
