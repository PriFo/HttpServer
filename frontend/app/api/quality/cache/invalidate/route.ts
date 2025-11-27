import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { getServerErrorMessage, getServerErrorStatus, isNetworkError } from '@/lib/fetch-utils-server'

const BACKEND_URL = getBackendUrl()

export async function POST(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const projectId = searchParams.get('project_id') ?? searchParams.get('projectId')

    const backendUrl = new URL(`${BACKEND_URL}/api/quality/cache/invalidate`)
    if (projectId) {
      backendUrl.searchParams.set('project_id', projectId)
    }

    const response = await fetch(backendUrl.toString(), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    })

    let payload: unknown = null
    try {
      payload = await response.json()
    } catch (jsonError) {
      // Не логируем отсутствие JSON payload - это может быть нормальным ответом
    }

    if (!response.ok) {
      const message = (payload as { error?: string } | null)?.error || 'Failed to invalidate cache'
      return NextResponse.json({ error: message }, { status: response.status })
    }

    return NextResponse.json(payload ?? { message: 'Cache invalidated' }, { status: response.status })
  } catch (error) {
    // Не логируем сетевые ошибки подключения - они ожидаемы, если бэкенд не запущен
    const isNetwork = isNetworkError(error)
    if (!isNetwork) {
      console.error('Error invalidating quality cache:', error)
    }
    
    const errorMessage = getServerErrorMessage(error, 'Не удалось инвалидировать кеш')
    const status = getServerErrorStatus(error, isNetwork ? 503 : 500)

    return NextResponse.json({ error: errorMessage }, { status })
  }
}


