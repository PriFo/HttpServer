import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ sessionId: string }> }
) {
  try {
    const { sessionId } = await params

    if (!sessionId) {
      return NextResponse.json(
        { error: 'session_id is required' },
        { status: 400 }
      )
    }

    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 7000)

    // Пока используем заглушку, так как реальный endpoint может отличаться
    const response = await fetch(`${API_BASE_URL}/api/normalization/session/${sessionId}`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
      signal: controller.signal,
    })

    clearTimeout(timeoutId)

    if (!response.ok) {
      // Если endpoint не существует, возвращаем заглушку
      if (response.status === 404) {
        return NextResponse.json({
          id: parseInt(sessionId, 10),
          status: 'completed',
          created_at: new Date().toISOString(),
          finished_at: new Date().toISOString(),
          processed_count: 0,
          success_count: 0,
          error_count: 0,
        })
      }

      const errorText = await response.text().catch(() => 'Unknown error')
      return NextResponse.json(
        { error: errorText },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      return NextResponse.json(
        { error: 'Превышено время ожидания ответа от сервера' },
        { status: 504 }
      )
    }
    console.error('Error fetching session details:', error)
    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : 'Unknown error',
      },
      { status: 500 }
    )
  }
}

