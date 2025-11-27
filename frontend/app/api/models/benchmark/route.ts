import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext } from '@/lib/logger'

const API_BASE_URL = getBackendUrl()

export async function GET(request: Request) {
  const context = createApiContext('/api/models/benchmark', 'GET')
  const startTime = Date.now()

  try {
    const { searchParams } = new URL(request.url)
    const history = searchParams.get('history')
    const limit = searchParams.get('limit')
    const model = searchParams.get('model')

    const queryParams = new URLSearchParams()
    if (history === 'true') queryParams.append('history', 'true')
    if (limit) queryParams.append('limit', limit)
    if (model) queryParams.append('model', model)

    const url = `${API_BASE_URL}/api/models/benchmark${queryParams.toString() ? `?${queryParams.toString()}` : ''}`
    
    logger.logRequest('GET', '/api/models/benchmark', context)
    
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    })

    const duration = Date.now() - startTime

    if (!response.ok) {
      const errorText = await response.text().catch(() => '')
      logger.error('Benchmark API GET error', {
        ...context,
        status: response.status,
        error: errorText,
        duration,
      })
      
      // Если это 404, возвращаем пустой ответ с сообщением
      if (response.status === 404) {
        logger.logResponse('GET', '/api/models/benchmark', 200, duration, context)
        return NextResponse.json(
          { 
            models: [], 
            total: 0, 
            test_count: 0, 
            timestamp: new Date().toISOString(),
            message: "Use POST to run benchmark or ?history=true to get history"
          },
          { status: 200 } // Возвращаем 200, чтобы фронтенд не показывал ошибку
        )
      }
      
      logger.logResponse('GET', '/api/models/benchmark', response.status, duration, context)
      return NextResponse.json(
        { error: errorText || `HTTP error! status: ${response.status}`, models: [], total: 0, test_count: 0, timestamp: new Date().toISOString() },
        { status: response.status }
      )
    }

    const data = await response.json()
    logger.logResponse('GET', '/api/models/benchmark', 200, duration, context)
    return NextResponse.json(data)
  } catch (error) {
    const duration = Date.now() - startTime
    logger.error('Benchmark API GET error', {
      ...context,
      error: error instanceof Error ? error.message : 'Unknown error',
      duration,
    })
    logger.logResponse('GET', '/api/models/benchmark', 500, duration, context)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to fetch benchmarks', models: [], total: 0, test_count: 0, timestamp: new Date().toISOString() },
      { status: 500 }
    )
  }
}

export async function POST(request: Request) {
  const context = createApiContext('/api/models/benchmark', 'POST')
  const startTime = Date.now()

  try {
    const body = await request.json().catch(() => ({}))
    
    logger.logRequest('POST', '/api/models/benchmark', context)
    
    // Добавляем таймаут для запроса к бэкенду
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 60000) // 60 секунд для бенчмарка (может тестировать много моделей)
    
    try {
      const response = await fetch(`${API_BASE_URL}/api/models/benchmark`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
        cache: 'no-store',
        signal: controller.signal,
      })

      clearTimeout(timeoutId)
      const duration = Date.now() - startTime

      if (!response.ok) {
        let errorMessage = `HTTP error! status: ${response.status}`
        
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorData.message || errorMessage
          
          // Улучшаем сообщения об ошибках для пользователя
          if (errorMessage.includes("ARLIAI_API_KEY") || errorMessage.includes("API key")) {
            errorMessage = "API ключ Arliai не настроен. Настройте его в разделе 'Воркеры' или установите переменную окружения ARLIAI_API_KEY"
          } else if (errorMessage.includes("No models available")) {
            errorMessage = "Нет доступных моделей для тестирования. Проверьте конфигурацию воркеров"
          } else if (errorMessage.includes("Failed to get models")) {
            errorMessage = "Не удалось получить список моделей. Проверьте конфигурацию"
          } else if (errorMessage.includes("timeout") || errorMessage.includes("Timeout")) {
            errorMessage = "Превышено время ожидания ответа от сервера. Сервер может быть перегружен. Попробуйте позже."
          }
        } catch {
          // Если не удалось распарсить JSON, используем статус код
          const errorText = await response.text().catch(() => '')
          if (errorText) {
            errorMessage = errorText
          } else if (response.status === 503) {
            errorMessage = "Сервис временно недоступен. Проверьте настройки API ключа"
          } else if (response.status === 404) {
            errorMessage = "Эндпоинт не найден. Проверьте версию API"
          } else if (response.status === 500) {
            errorMessage = "Внутренняя ошибка сервера. Проверьте логи сервера"
          } else if (response.status === 408 || response.status === 504) {
            errorMessage = "Превышено время ожидания ответа от сервера. Попробуйте позже."
          }
        }
        
        logger.error('Benchmark API POST error', {
          ...context,
          status: response.status,
          error: errorMessage,
          duration,
        })
        logger.logResponse('POST', '/api/models/benchmark', response.status, duration, context)
        
        return NextResponse.json(
          { error: errorMessage },
          { status: response.status }
        )
      }

      const data = await response.json()
      logger.logResponse('POST', '/api/models/benchmark', 200, duration, context)
      return NextResponse.json(data)
    } catch (fetchError: any) {
      clearTimeout(timeoutId)
      const duration = Date.now() - startTime
      
      if (fetchError.name === 'AbortError') {
        logger.warn('Benchmark API POST timeout after 60s', {
          ...context,
          duration,
        })
        logger.logResponse('POST', '/api/models/benchmark', 408, duration, context)
        return NextResponse.json(
          { error: 'Превышено время ожидания ответа от сервера (60 секунд). Бэнчмарк может тестировать много моделей. Попробуйте позже или уменьшите количество моделей.' },
          { status: 408 }
        )
      }
      
      // Проверяем различные типы ошибок подключения
      const isConnectionError = 
        fetchError.message?.includes('fetch failed') ||
        fetchError.message?.includes('ECONNREFUSED') ||
        fetchError.message?.includes('ENOTFOUND') ||
        fetchError.message?.includes('ETIMEDOUT') ||
        fetchError.message?.includes('network') ||
        fetchError.code === 'ECONNREFUSED' ||
        fetchError.code === 'ENOTFOUND' ||
        fetchError.code === 'ETIMEDOUT'
      
      if (isConnectionError) {
        const backendUrl = API_BASE_URL
        logger.error('Benchmark API POST connection failed', {
          ...context,
          error: fetchError.message,
          backendUrl,
          duration,
        })
        logger.logResponse('POST', '/api/models/benchmark', 503, duration, context)
        return NextResponse.json(
          { 
            error: `Не удалось подключиться к серверу (${backendUrl}). Проверьте, что бэкенд запущен и доступен.`,
            backendUrl,
            suggestion: 'Убедитесь, что Go сервер запущен на порту 9999 или проверьте настройки BACKEND_URL в переменных окружения.'
          },
          { status: 503 }
        )
      }
      
      throw fetchError
    }
  } catch (error) {
    const duration = Date.now() - startTime
    logger.error('Benchmark API POST error', {
      ...context,
      error: error instanceof Error ? error.message : 'Unknown error',
      duration,
    })
    logger.logResponse('POST', '/api/models/benchmark', 500, duration, context)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to run benchmark' },
      { status: 500 }
    )
  }
}

