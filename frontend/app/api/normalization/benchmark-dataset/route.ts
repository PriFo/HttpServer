import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE = getBackendUrl()

export async function GET(request: NextRequest) {
  try {
    // Получаем параметры из URL
    const { searchParams } = new URL(request.url)
    const limitParam = searchParams.get('limit')
    const database = searchParams.get('database')

    // Валидация и нормализация limit
    let limit = 50 // По умолчанию
    if (limitParam) {
      const parsedLimit = parseInt(limitParam, 10)
      if (!isNaN(parsedLimit) && parsedLimit > 0) {
        limit = Math.min(parsedLimit, 500) // Максимум 500 для безопасности
      }
    }

    // Валидация database path для предотвращения path traversal
    if (database && (database.includes('..') || database.includes('~'))) {
      return NextResponse.json(
        { error: 'Invalid database path' },
        { status: 400 }
      )
    }

    // Формируем query string
    const params = new URLSearchParams()
    params.append('limit', limit.toString())
    if (database) {
      params.append('database', database)
    }

    const queryString = params.toString()
    const url = `${API_BASE}/api/normalization/benchmark-dataset?${queryString}`

    // Создаем AbortController для таймаута
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут

    try {
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Если бэкенд недоступен или эндпоинт не найден, возвращаем пустой массив
        if (response.status === 0 || response.status === 404 || response.status >= 500) {
          console.warn(`[BenchmarkDataset] Backend error or endpoint not found: HTTP ${response.status}`, {
            status: response.status,
            statusText: response.statusText,
            url,
          })
          return NextResponse.json({
            data: [],
            count: 0,
            limit,
            source: 'normalization',
            message: response.status === 404 
              ? 'Benchmark dataset endpoint not available, using empty dataset'
              : 'Backend unavailable, using empty dataset',
          })
        }
        console.warn(`[BenchmarkDataset] Backend returned error status: HTTP ${response.status}`)
        // Для других ошибок (400, 401, 403) также возвращаем пустой массив, чтобы не блокировать бенчмарк
        return NextResponse.json({
          data: [],
          count: 0,
          limit,
          source: 'normalization',
          message: `Backend responded with ${response.status}, using empty dataset`,
        })
      }

      let data
      try {
        data = await response.json()
      } catch (parseError) {
        console.error('[BenchmarkDataset] Failed to parse response JSON', {
          error: parseError,
          url,
        })
        return NextResponse.json({
          data: [],
          count: 0,
          limit,
          source: 'normalization',
        })
      }
      return NextResponse.json(data)
    } catch (fetchError: any) {
      clearTimeout(timeoutId)
      if (fetchError.name === 'AbortError') {
        console.warn('[BenchmarkDataset] Request timeout, returning empty dataset', {
          timeout: '10s',
          url,
        })
      } else if (fetchError.message?.includes('fetch failed') || fetchError.message?.includes('ECONNREFUSED')) {
        console.warn('[BenchmarkDataset] Backend connection failed, returning empty dataset', {
          error: fetchError.message,
          url,
        })
      } else {
        console.error('[BenchmarkDataset] Unexpected error, returning empty dataset', {
          error: fetchError,
          message: fetchError?.message,
          url,
        })
      }
      // Всегда возвращаем пустой массив вместо ошибки, чтобы не блокировать бенчмарк
      return NextResponse.json({
        data: [],
        count: 0,
        limit,
        source: 'normalization',
        message: 'Backend unavailable or timeout, using empty dataset',
      })
    }
  } catch (error) {
    // Бэкенд недоступен - возвращаем пустые данные
    console.warn('[BenchmarkDataset] Error fetching benchmark dataset, returning empty dataset:', error)
    return NextResponse.json({
      data: [],
      count: 0,
      limit: 50,
      source: 'normalization',
      message: 'Backend unavailable, using empty dataset',
    })
  }
}

