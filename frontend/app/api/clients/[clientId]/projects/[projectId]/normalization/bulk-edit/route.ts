import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const body = await request.json()
    const { items, updates } = body

    if (!items || !Array.isArray(items) || items.length === 0) {
      return NextResponse.json(
        { error: 'Invalid request: items array required' },
        { status: 400 }
      )
    }

    if (!updates || Object.keys(updates).length === 0) {
      return NextResponse.json(
        { error: 'Invalid request: updates object required' },
        { status: 400 }
      )
    }

    // Выполняем массовое обновление
    const response = await fetch(
      `${BACKEND_URL}/api/normalization/groups/bulk-update`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          client_id: parseInt(clientId),
          project_id: parseInt(projectId),
          items,
          updates,
        }),
      }
    )

    if (!response.ok) {
      const errorText = await response.text()
      console.error('Backend bulk edit error:', errorText)
      return NextResponse.json(
        { error: 'Failed to perform bulk edit' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json({
      success: true,
      processed: items.length,
      result: data,
    })
  } catch (error) {
    console.error('Error performing bulk edit:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

