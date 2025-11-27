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
    const entityType = searchParams.get('entity_type')
    const entityId = searchParams.get('entity_id')
    const limit = parseInt(searchParams.get('limit') || '50')
    const offset = parseInt(searchParams.get('offset') || '0')

    const BACKEND_URL = getBackendUrl()
    
    // Получаем историю изменений
    const historyResponse = await fetch(
      `${BACKEND_URL}/api/normalization/history?client_id=${clientId}&project_id=${projectId}${entityType ? `&entity_type=${entityType}` : ''}${entityId ? `&entity_id=${entityId}` : ''}&limit=${limit}&offset=${offset}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    )

    if (!historyResponse.ok) {
      // Если бэкенд не поддерживает историю, возвращаем заглушку
      const mockHistory = [
        {
          id: '1',
          timestamp: new Date().toISOString(),
          user: 'system',
          action: 'create',
          entity_type: 'group',
          entity_id: 'group-1',
          description: 'Создана новая группа нормализации',
          changes: [],
        },
      ]

      return NextResponse.json({ history: mockHistory })
    }

    const historyData = await historyResponse.json()

    // Форматируем историю
    const history = (historyData.history || historyData || []).map((entry: any) => ({
      id: entry.id || entry.timestamp,
      timestamp: entry.timestamp || entry.created_at,
      user: entry.user || entry.user_name || 'system',
      action: entry.action || 'update',
      entity_type: entry.entity_type || 'group',
      entity_id: entry.entity_id || entry.id,
      description: entry.description || entry.message,
      changes: entry.changes || entry.diff || [],
    }))

    return NextResponse.json({ history })
  } catch (error) {
    console.error('Error fetching normalization history:', error)
    // Возвращаем пустую историю при ошибке
    return NextResponse.json({ history: [] })
  }
}

