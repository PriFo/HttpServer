import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, isTimeoutError, isNetworkError } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

const DEFAULT_STATUS = {
  is_running: false,
  progress: 0,
  processed: 0,
  total: 0,
  current_step: 'idle',
  duplicates_found: 0,
  violations_found: 0,
  suggestions_found: 0,
}

export async function GET() {
  try {
    console.log('Proxying GET /api/quality/analyze/status to backend')
    
    const data = await fetchJsonServer(`${BACKEND_URL}/api/quality/analyze/status`, {
      timeout: QUALITY_TIMEOUTS.FAST,
      cache: 'no-store',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching analysis status:', error)
    
    // Возвращаем дефолтный статус вместо ошибки, чтобы фронтенд мог работать
    const errorMessage = isTimeoutError(error) 
      ? 'Превышено время ожидания ответа от сервера'
      : isNetworkError(error)
      ? 'Не удалось подключиться к серверу'
      : 'Backend unavailable'
    
    return NextResponse.json({
      ...DEFAULT_STATUS,
      error: errorMessage,
      details: getServerErrorMessage(error, 'Unknown error')
    })
  }
}

