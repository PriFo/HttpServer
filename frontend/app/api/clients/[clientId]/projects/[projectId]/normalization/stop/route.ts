import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

export async function POST(
  request: Request,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const resolvedParams = await params
    const { clientId, projectId } = resolvedParams

    // Валидация параметров
    if (!clientId || !projectId) {
      console.error('Missing clientId or projectId:', { clientId, projectId })
      return NextResponse.json(
        { error: 'Missing clientId or projectId' },
        { status: 400 }
      )
    }

    // Проверяем, что это числа
    const clientIdNum = parseInt(clientId, 10)
    const projectIdNum = parseInt(projectId, 10)
    
    if (isNaN(clientIdNum) || isNaN(projectIdNum)) {
      console.error('Invalid clientId or projectId:', { clientId, projectId })
      return NextResponse.json(
        { error: 'Invalid clientId or projectId' },
        { status: 400 }
      )
    }

    const url = `${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/stop`
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      signal: AbortSignal.timeout(10000), // 10 секунд таймаут
    })

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error')
      let errorData: { error?: string } = {}
      try {
        errorData = JSON.parse(errorText)
      } catch {
        // Если не JSON, используем текст как есть
      }
      
      console.error(`Backend error (${response.status}):`, errorText)
      return NextResponse.json(
        { error: errorData.error || errorText || 'Failed to stop normalization' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error stopping normalization:', error)
    const errorMessage = error instanceof Error ? error.message : 'Unknown error'
    return NextResponse.json(
      { error: `Failed to stop normalization: ${errorMessage}` },
      { status: 500 }
    )
  }
}

