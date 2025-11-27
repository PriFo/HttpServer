import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus, isNetworkError } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

// Fallback данные для мониторинга
const FALLBACK_MONITORING_METRICS = {
  providers: [],
  system: {
    total_providers: 0,
    active_providers: 0,
    total_requests: 0,
    total_successful: 0,
    total_failed: 0,
    system_requests_per_second: 0,
    timestamp: new Date().toISOString(),
  },
  isFallback: true,
  fallbackReason: 'Backend сервер недоступен',
  timestamp: new Date().toISOString(),
}

function createFallbackResponse(reason: string) {
  return NextResponse.json(
    {
      ...FALLBACK_MONITORING_METRICS,
      fallbackReason: reason,
    },
    { status: 200 }
  )
}

export async function GET() {
  try {
    const data = await fetchJsonServer(`${BACKEND_URL}/api/monitoring/metrics`, {
      timeout: QUALITY_TIMEOUTS.FAST,
      cache: 'no-store',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    // Не логируем ошибки подключения в консоль - они ожидаемы, если бэкенд не запущен
    const isNetwork = isNetworkError(error)
    if (!isNetwork) {
      console.error('Error fetching monitoring metrics:', error)
    }
    
    // Для сетевых ошибок возвращаем fallback данные вместо 503
    if (isNetwork) {
      return createFallbackResponse('Backend сервер недоступен. Убедитесь, что сервер запущен на порту 9999.')
    }

    // Для других ошибок возвращаем ошибку, но также можем вернуть fallback
    const errorMessage = getServerErrorMessage(error, 'Не удалось получить метрики мониторинга')
    
    // Возвращаем fallback для любых ошибок, чтобы фронтенд не ломался
    return createFallbackResponse(errorMessage)
  }
}
