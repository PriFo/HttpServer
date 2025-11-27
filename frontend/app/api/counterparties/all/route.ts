import { NextRequest, NextResponse } from 'next/server'
import { ApiErrorHandler } from '@/lib/api-error-handler'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext } from '@/lib/logger'
import { handleFetchError } from '@/lib/error-handler'

const BACKEND_URL = getBackendUrl()
const MAX_BACKEND_LIMIT = 100000
const HEAVY_THRESHOLD = 5000

export async function GET(request: NextRequest) {
  const endpoint = '/api/counterparties/all'
  const context = createApiContext(endpoint, 'GET')
  const startTime = Date.now()
  
  try {
    const { searchParams } = new URL(request.url)
    const clientId = searchParams.get('client_id')
    const projectId = searchParams.get('project_id')
    const offset = searchParams.get('offset') || '0'
    const limitParam = Number(searchParams.get('limit') || '20')
    let safeLimit = Number.isFinite(limitParam) ? limitParam : 20
    if (safeLimit < 1) {
      safeLimit = 1
    }
    const loadAllParam = searchParams.get('load_all')
    const forceLoadAll = loadAllParam === '1' || loadAllParam?.toLowerCase() === 'true'
    let clampedLimit = forceLoadAll ? MAX_BACKEND_LIMIT : Math.min(safeLimit, MAX_BACKEND_LIMIT)
    const search = searchParams.get('search')

    if (!clientId) {
      return NextResponse.json(
        { error: 'client_id is required' },
        { status: 400 }
      )
    }

    let url = `${BACKEND_URL}/api/counterparties/all?client_id=${encodeURIComponent(clientId)}&offset=${encodeURIComponent(offset)}&limit=${encodeURIComponent(clampedLimit)}`
    if (forceLoadAll) {
      url += `&load_all=1`
    }
    
    if (projectId) {
      url += `&project_id=${encodeURIComponent(projectId)}`
    }
    
    if (search) {
      url += `&search=${encodeURIComponent(search)}`
    }

    logger.logRequest('GET', endpoint, { ...context, clientId, projectId })

    try {
      const response = await fetch(url, {
        cache: 'no-store',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      const duration = Date.now() - startTime
      logger.logResponse('GET', endpoint, response.status, duration, { ...context, clientId, projectId })

      if (!response.ok) {
        // Для 404 возвращаем пустой список
        if (response.status === 404) {
          return NextResponse.json({
            counterparties: [],
            total: 0,
            offset: parseInt(offset),
            limit: clampedLimit,
          })
        }

        const apiError = await ApiErrorHandler.handleError(response)
        ApiErrorHandler.logError(endpoint, apiError, { ...context, clientId, projectId })
        
        return NextResponse.json(
          { error: apiError.error },
          { status: response.status }
        )
      }

      const isHeavy = forceLoadAll || clampedLimit >= HEAVY_THRESHOLD
      if (isHeavy) {
        const headers = new Headers(response.headers)
        return new NextResponse(response.body, {
          status: response.status,
          headers,
        })
      }

      const data = await response.json()
      if ((forceLoadAll || safeLimit > MAX_BACKEND_LIMIT) && data && typeof data === 'object' && !Array.isArray(data)) {
        ;(data as Record<string, unknown>).limit_clamped = true
        ;(data as Record<string, unknown>).limit_max = MAX_BACKEND_LIMIT
      }
      return NextResponse.json(data)
    } catch (fetchError) {
      const duration = Date.now() - startTime
      
      // Проверяем тип ошибки
      if (fetchError instanceof Error) {
        // Для сетевых ошибок и таймаутов возвращаем 503
        if (fetchError.name === 'AbortError' || fetchError.message.includes('fetch')) {
          logger.warn(`Network error in ${endpoint}`, { ...context, duration, clientId, projectId })
          return NextResponse.json(
            { error: 'Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен.' },
            { status: 503 }
          )
        }
      }
      
      // Используем централизованную обработку ошибок для остальных случаев
      return handleFetchError(fetchError, url, { ...context, duration, clientId, projectId })
    }
  } catch (error) {
    const duration = Date.now() - startTime
    logger.error(`Unexpected error in ${endpoint}`, { ...context, duration }, error instanceof Error ? error : undefined)
    
    // Для неожиданных ошибок возвращаем 500 с информативным сообщением
    const errorMessage = error instanceof Error ? error.message : 'Internal server error'
    return NextResponse.json(
      { error: `Ошибка сервера при загрузке контрагентов: ${errorMessage}` },
      { status: 500 }
    )
  }
}

