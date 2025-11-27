import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export const runtime = 'nodejs'

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const body = await request.json()
    const { operation, items } = body

    if (!operation || !items || !Array.isArray(items) || items.length === 0) {
      return NextResponse.json(
        { error: 'Invalid request: operation and items array required' },
        { status: 400 }
      )
    }

    const BACKEND_URL = getBackendUrl()

    // В зависимости от операции вызываем соответствующий endpoint
    let endpoint = ''
    let method = 'POST'

    switch (operation) {
      case 'merge':
        // Объединение групп
        endpoint = `${BACKEND_URL}/api/normalization/groups/merge`
        break
      case 'delete':
        // Удаление групп
        endpoint = `${BACKEND_URL}/api/normalization/groups/delete`
        method = 'DELETE'
        break
      case 'tag':
        // Тегирование
        endpoint = `${BACKEND_URL}/api/normalization/groups/tag`
        break
      case 'export':
        // Экспорт
        endpoint = `${BACKEND_URL}/api/normalization/export`
        method = 'GET'
        break
      default:
        return NextResponse.json(
          { error: `Unsupported operation: ${operation}` },
          { status: 400 }
        )
    }

    const response = await fetch(endpoint, {
      method,
      headers: {
        'Content-Type': 'application/json',
      },
      body: method !== 'GET' ? JSON.stringify({
        client_id: parseInt(clientId),
        project_id: parseInt(projectId),
        items,
      }) : undefined,
    })

    if (!response.ok) {
      const errorText = await response.text()
      console.error('Backend batch operation error:', errorText)
      return NextResponse.json(
        { error: 'Failed to perform batch operation' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json({
      success: true,
      operation,
      processed: items.length,
      result: data,
    })
  } catch (error) {
    console.error('Error performing batch operation:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

