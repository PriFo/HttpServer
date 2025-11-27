import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus, isNetworkError } from '@/lib/fetch-utils-server'

const BACKEND_URL = getBackendUrl()

// Fallback данные для статистики кеша
const FALLBACK_CACHE_STATS = {
  size: 0,
  entries: 0,
  hits: 0,
  misses: 0,
  hitRate: 0,
  isFallback: true,
  fallbackReason: 'Backend сервер недоступен',
  timestamp: new Date().toISOString(),
}

function createFallbackResponse(reason: string) {
  return NextResponse.json(
    {
      ...FALLBACK_CACHE_STATS,
      fallbackReason: reason,
    },
    { status: 200 }
  )
}

export async function GET() {
  try {
    const data = await fetchJsonServer(`${BACKEND_URL}/api/quality/cache/stats`, {
      timeout: 10000,
      cache: 'no-store',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    // Не логируем сетевые ошибки подключения - они ожидаемы, если бэкенд не запущен
    const isNetwork = isNetworkError(error)
    if (!isNetwork) {
      console.error('Error fetching quality cache stats:', error)
    }
    
    // Для сетевых ошибок возвращаем fallback данные
    if (isNetwork) {
      return createFallbackResponse('Backend сервер недоступен. Убедитесь, что сервер запущен на порту 9999.')
    }

    // Для других ошибок также возвращаем fallback
    const errorMessage = getServerErrorMessage(error, 'Не удалось получить статистику кеша')
    return createFallbackResponse(errorMessage)
  }
}

