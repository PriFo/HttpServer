import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params

    const response = await fetch(
      `${BACKEND_URL}/api/clients/${clientId}/projects/${projectId}/diagnostics/uploads/fix`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    if (!response.ok) {
      const errorText = await response.text()
      let errorData
      try {
        errorData = JSON.parse(errorText)
      } catch {
        errorData = { error: errorText || 'Ошибка создания upload записей' }
      }

      return NextResponse.json(
        { error: errorData.error || 'Не удалось создать upload записи' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('[Diagnostics Uploads Fix API] Error:', error)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Внутренняя ошибка сервера' },
      { status: 500 }
    )
  }
}

