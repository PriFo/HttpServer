import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus, isNetworkError } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

// Fallback данные для ГОСТов
const FALLBACK_GOSTS = {
  items: [],
  total: 0,
  isFallback: true,
  fallbackReason: 'Backend сервер недоступен',
  timestamp: new Date().toISOString(),
}

function createFallbackResponse(reason: string) {
  return NextResponse.json(
    {
      ...FALLBACK_GOSTS,
      fallbackReason: reason,
    },
    { status: 200 }
  )
}

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url)
    
    const limit = searchParams.get('limit')
    const offset = searchParams.get('offset')
    const status = searchParams.get('status')
    const sourceType = searchParams.get('source_type')
    const search = searchParams.get('search')
    const adoptionFrom = searchParams.get('adoption_from')
    const adoptionTo = searchParams.get('adoption_to')
    const effectiveFrom = searchParams.get('effective_from')
    const effectiveTo = searchParams.get('effective_to')

    const params = new URLSearchParams()
    if (limit) params.append('limit', limit)
    if (offset) params.append('offset', offset)
    if (status) params.append('status', status)
    if (sourceType) params.append('source_type', sourceType)
    if (search) params.append('search', search)
    if (adoptionFrom) params.append('adoption_from', adoptionFrom)
    if (adoptionTo) params.append('adoption_to', adoptionTo)
    if (effectiveFrom) params.append('effective_from', effectiveFrom)
    if (effectiveTo) params.append('effective_to', effectiveTo)

    const url = `${BACKEND_URL}/api/gosts${params.toString() ? `?${params.toString()}` : ''}`

    const data = await fetchJsonServer(url, {
      timeout: QUALITY_TIMEOUTS.STANDARD,
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
      console.error('Error fetching GOSTs:', error)
    }
    
    // Для сетевых ошибок возвращаем fallback данные
    if (isNetwork) {
      return createFallbackResponse('Backend сервер недоступен. Убедитесь, что сервер запущен на порту 9999.')
    }

    // Для других ошибок также возвращаем fallback для консистентности
    const errorMessage = getServerErrorMessage(error, 'Не удалось получить данные о ГОСТах')
    return createFallbackResponse(errorMessage)
  }
}

