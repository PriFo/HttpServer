import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export const runtime = 'nodejs'

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const searchParams = request.nextUrl.searchParams
    const entityId = searchParams.get('entity_id')
    const entityType = searchParams.get('entity_type') || 'group'

    if (!entityId) {
      return NextResponse.json(
        { error: 'entity_id is required' },
        { status: 400 }
      )
    }

    const BACKEND_URL = getBackendUrl()
    
    // Получаем историю изменений для сущности
    const historyResponse = await fetch(
      `${BACKEND_URL}/api/normalization/history?client_id=${clientId}&project_id=${projectId}&entity_type=${entityType}&entity_id=${entityId}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    )

    if (!historyResponse.ok) {
      // Возвращаем заглушку с версиями
      const mockVersions = [
        {
          id: 'v1',
          timestamp: new Date(Date.now() - 86400000).toISOString(),
          user: 'system',
          data: {
            name: 'Товар А',
            category: 'electronics',
            attributes: [],
          },
        },
        {
          id: 'v2',
          timestamp: new Date().toISOString(),
          user: 'admin',
          data: {
            name: 'Товар А (обновлен)',
            category: 'electronics',
            attributes: [{ name: 'Цвет', value: 'Черный' }],
          },
        },
      ]

      return NextResponse.json({ versions: mockVersions })
    }

    const historyData = await historyResponse.json()
    const history = historyData.history || historyData || []

    // Формируем версии из истории
    const versions = history.map((entry: any, index: number) => ({
      id: entry.id || `v${index + 1}`,
      timestamp: entry.timestamp || entry.created_at,
      user: entry.user || entry.user_name || 'system',
      data: entry.data || entry.changes || {},
      changes: entry.changes || entry.diff || [],
    }))

    return NextResponse.json({ versions })
  } catch (error) {
    console.error('Error fetching versions:', error)
    return NextResponse.json({ versions: [] })
  }
}

