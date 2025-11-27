import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

export async function GET(
  request: Request,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const resolvedParams = await params
    const { clientId, projectId } = resolvedParams

    // Валидация параметров
    if (!clientId || !projectId) {
      console.error('Missing clientId or projectId:', { clientId, projectId })
      return NextResponse.json({
        total_processed: 0,
        total_groups: 0,
        benchmark_matches: 0,
        ai_enhanced: 0,
        basic_normalized: 0,
        is_running: false,
        error: 'Missing clientId or projectId'
      }, { status: 400 })
    }

    // Проверяем, что это числа
    const clientIdNum = parseInt(clientId, 10)
    const projectIdNum = parseInt(projectId, 10)
    
    if (isNaN(clientIdNum) || isNaN(projectIdNum)) {
      console.error('Invalid clientId or projectId:', { clientId, projectId })
      return NextResponse.json({
        total_processed: 0,
        total_groups: 0,
        benchmark_matches: 0,
        ai_enhanced: 0,
        basic_normalized: 0,
        is_running: false,
        error: 'Invalid clientId or projectId'
      }, { status: 400 })
    }

    const url = `${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/status`
    
    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
      signal: AbortSignal.timeout(10000), // 10 секунд таймаут
    })

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error')
      console.error(`Backend error (${response.status}):`, errorText)
      
      // Возвращаем дефолтные значения если endpoint не найден или ошибка
      return NextResponse.json({
        total_processed: 0,
        total_groups: 0,
        benchmark_matches: 0,
        ai_enhanced: 0,
        basic_normalized: 0,
        is_running: false,
        error: `Backend returned ${response.status}: ${errorText}`
      }, { status: response.status })
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching normalization status:', error)
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    
    return NextResponse.json({
      total_processed: 0,
      total_groups: 0,
      benchmark_matches: 0,
      ai_enhanced: 0,
      basic_normalized: 0,
      is_running: false,
      error: errorMessage
    }, { status: 500 })
  }
}

