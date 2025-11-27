import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus, isNetworkError } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const API_BASE_URL = getBackendUrl()

const FALLBACK_QUALITY_METRICS = {
  overallQuality: 0,
  highConfidence: 0,
  mediumConfidence: 0,
  lowConfidence: 0,
  totalRecords: 0,
}

function createFallbackResponse(reason: string) {
  return NextResponse.json(
    {
      ...FALLBACK_QUALITY_METRICS,
      isFallback: true,
      fallbackReason: reason,
      timestamp: new Date().toISOString(),
    },
    { status: 200 }
  )
}

export async function GET(request: NextRequest) {
  try {
    const data = await fetchJsonServer(`${API_BASE_URL}/api/quality/metrics`, {
      timeout: QUALITY_TIMEOUTS.FAST,
      cache: 'no-store',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    // Не логируем сетевые ошибки подключения - они ожидаемы, если бэкенд не запущен
    const isNetwork = isNetworkError(error)
    const errorMessage = getServerErrorMessage(error, 'Internal server error')
    const status = getServerErrorStatus(error, 500)

    // Для сетевых ошибок и 503/404 всегда возвращаем fallback
    if (isNetwork || status === 404 || status === 503) {
      if (!isNetwork) {
        // Логируем только не-сетевые ошибки
        console.warn('[quality/metrics] Backend unavailable, returning fallback metrics:', errorMessage)
      }
      return createFallbackResponse(isNetwork 
        ? 'Backend сервер недоступен. Убедитесь, что сервер запущен на порту 9999.'
        : errorMessage
      )
    }

    // Для других ошибок возвращаем fallback для консистентности
    if (!isNetwork) {
      console.error('Error fetching quality metrics:', error)
    }
    return createFallbackResponse(errorMessage)
  }
}
