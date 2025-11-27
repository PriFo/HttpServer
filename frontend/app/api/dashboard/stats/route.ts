import { NextResponse } from 'next/server';
import { getBackendUrl } from '@/lib/api-config';
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus, isNetworkError } from '@/lib/fetch-utils-server';
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants';

const API_BASE_URL = getBackendUrl()

const FALLBACK_STATS = {
  totalRecords: 0,
  totalDatabases: 0,
  processedRecords: 0,
  createdGroups: 0,
  mergedRecords: 0,
  systemVersion: 'unknown',
  currentDatabase: null,
  normalizationStatus: {
    status: 'idle',
    progress: 0,
    currentStage: 'Недоступно',
    startTime: null,
    endTime: null,
  },
  qualityMetrics: {
    overallQuality: 0,
    highConfidence: 0,
    mediumConfidence: 0,
    lowConfidence: 0,
    totalRecords: 0,
  },
}

function createFallbackResponse(reason: string) {
  return NextResponse.json(
    {
      ...FALLBACK_STATS,
      isFallback: true,
      fallbackReason: reason,
      timestamp: new Date().toISOString(),
    },
    { status: 200 }
  )
}

export async function GET() {
  try {
    const data = await fetchJsonServer(`${API_BASE_URL}/api/dashboard/stats`, {
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
      console.error('Error fetching dashboard stats:', error)
    }
    
    // Для сетевых ошибок возвращаем fallback данные
    if (isNetwork) {
      return createFallbackResponse('Backend сервер недоступен. Убедитесь, что сервер запущен на порту 9999.')
    }

    // Для других ошибок также возвращаем fallback для консистентности
    const errorMessage = getServerErrorMessage(error, 'Не удалось получить статистику дашборда')
    return createFallbackResponse(errorMessage)
  }
}
