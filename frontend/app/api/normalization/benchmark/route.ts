import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

// Интерфейсы для бенчмарка нормализации
interface BenchmarkResult {
  stage: string
  record_count: number
  duration_ms: number
  records_per_second: number
  memory_used_mb?: number
  duplicate_groups?: number
  total_duplicates?: number
  processed_count?: number
  benchmark_matches?: number
  enriched_count?: number
  created_benchmarks?: number
  error_count?: number
  stopped?: boolean
}

interface BenchmarkReport {
  timestamp: string
  test_name: string
  record_count: number
  duplicate_rate: number
  workers: number
  results: BenchmarkResult[]
  total_duration_ms: number
  average_speed_records_per_sec: number
  summary: Record<string, unknown>
}

// GET - получение списка доступных бенчмарков или конкретного бенчмарка
export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url)
    const id = searchParams.get('id')
    const list = searchParams.get('list') === 'true'

    // Если запрашивается список, пробуем получить через бэкенд
    if (list) {
      try {
        const response = await fetch(`${API_BASE_URL}/api/normalization/benchmark/list`, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          cache: 'no-store',
        })

        if (response.ok) {
          const data = await response.json()
          return NextResponse.json(data)
        }
      } catch {
        console.warn('[Benchmark API] Backend list endpoint not available, using file system')
      }

      // Fallback: ищем JSON файлы в текущей директории (для разработки)
      // В продакшене это должно быть через бэкенд
      return NextResponse.json({
        benchmarks: [],
        message: 'Use POST to upload benchmark results or configure backend endpoint'
      })
    }

    // Если запрашивается конкретный бенчмарк
    if (id) {
      try {
        const response = await fetch(`${API_BASE_URL}/api/normalization/benchmark/${id}`, {
          method: 'GET',
          headers: {
            'Content-Type': 'application/json',
          },
          cache: 'no-store',
        })

        if (response.ok) {
          const data = await response.json()
          return NextResponse.json(data)
        }
      } catch {
        console.warn('[Benchmark API] Backend get endpoint not available')
      }

      return NextResponse.json(
        { error: 'Benchmark not found' },
        { status: 404 }
      )
    }

    // По умолчанию возвращаем пустой список
    return NextResponse.json({
      benchmarks: [],
      message: 'Use ?list=true to get list or POST to upload'
    })
  } catch (error) {
    console.error('[Benchmark API] GET error:', error)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to fetch benchmarks' },
      { status: 500 }
    )
  }
}

// POST - загрузка результатов бенчмарка (JSON файл)
export async function POST(request: Request) {
  try {
    const formData = await request.formData()
    const file = formData.get('file') as File | null

    if (!file) {
      return NextResponse.json(
        { error: 'No file provided' },
        { status: 400 }
      )
    }

    // Проверяем тип файла
    if (!file.name.endsWith('.json')) {
      return NextResponse.json(
        { error: 'File must be a JSON file' },
        { status: 400 }
      )
    }

    // Читаем содержимое файла
    const fileContent = await file.text()
    let benchmarkData: BenchmarkReport

    try {
      benchmarkData = JSON.parse(fileContent)
    } catch {
      return NextResponse.json(
        { error: 'Invalid JSON file' },
        { status: 400 }
      )
    }

    // Валидация структуры
    if (!benchmarkData.timestamp || !benchmarkData.results || !Array.isArray(benchmarkData.results)) {
      return NextResponse.json(
        { error: 'Invalid benchmark report format' },
        { status: 400 }
      )
    }

    // Отправляем на бэкенд для сохранения
    try {
      const response = await fetch(`${API_BASE_URL}/api/normalization/benchmark/upload`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(benchmarkData),
        cache: 'no-store',
        signal: AbortSignal.timeout(10000), // 10 секунд таймаут
      })

      if (response.ok) {
        const data = await response.json()
        return NextResponse.json({
          success: true,
          message: 'Benchmark uploaded successfully',
          id: data.id || benchmarkData.timestamp,
          data: benchmarkData
        })
      } else {
        await response.text().catch(() => '')
        console.warn('[Benchmark API] Backend upload failed, returning data directly')
        // Если бэкенд недоступен, возвращаем данные напрямую
        return NextResponse.json({
          success: true,
          message: 'Benchmark processed (backend not available)',
          id: benchmarkData.timestamp,
          data: benchmarkData
        })
      }
    } catch (error) {
      // Если бэкенд недоступен, возвращаем данные напрямую
      console.warn('[Benchmark API] Backend not available, returning data directly:', error)
      return NextResponse.json({
        success: true,
        message: 'Benchmark processed (backend not available)',
        id: benchmarkData.timestamp,
        data: benchmarkData
      })
    }
  } catch (error) {
    console.error('[Benchmark API] POST error:', error)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to process benchmark' },
      { status: 500 }
    )
  }
}

// PUT - анализ узких мест
export async function PUT(request: Request) {
  try {
    const body = await request.json()
    const action = body.action

    if (action === 'analyze') {
      // Проксируем запрос на бэкенд
      const response = await fetch(`${API_BASE_URL}/api/normalization/benchmark/analyze`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body.report),
        cache: 'no-store',
      })

      if (!response.ok) {
        const errorText = await response.text().catch(() => '')
        return NextResponse.json(
          { error: errorText || `HTTP error! status: ${response.status}` },
          { status: response.status }
        )
      }

      const data = await response.json()
      return NextResponse.json(data)
    }

    if (action === 'compare') {
      // Проксируем запрос на бэкенд
      const response = await fetch(`${API_BASE_URL}/api/normalization/benchmark/compare`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          baseline_id: body.baseline_id,
          current_id: body.current_id,
        }),
        cache: 'no-store',
      })

      if (!response.ok) {
        const errorText = await response.text().catch(() => '')
        return NextResponse.json(
          { error: errorText || `HTTP error! status: ${response.status}` },
          { status: response.status }
        )
      }

      const data = await response.json()
      return NextResponse.json(data)
    }

    return NextResponse.json(
      { error: 'Unknown action' },
      { status: 400 }
    )
  } catch (error) {
    console.error('[Benchmark API] PUT error:', error)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to process request' },
      { status: 500 }
    )
  }
}

