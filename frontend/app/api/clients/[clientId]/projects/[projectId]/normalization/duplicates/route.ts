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
    const similarityThreshold = parseFloat(searchParams.get('similarity_threshold') || '0.85')
    const maxClusterSize = parseInt(searchParams.get('max_cluster_size') || '10')

    const BACKEND_URL = getBackendUrl()

    // Получаем группы нормализации
    const groupsResponse = await fetch(
      `${BACKEND_URL}/api/clients/${clientId}/projects/${projectId}/normalization/groups`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    )

    if (!groupsResponse.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch groups' },
        { status: groupsResponse.status }
      )
    }

    const groupsData = await groupsResponse.json()
    const groups = groupsData.groups || []

    // Формируем кластеры дубликатов
    const clusters: any[] = []

    for (const group of groups) {
      if (group.merged_count > 1) {
        // Получаем элементы группы
        const itemsResponse = await fetch(
          `${BACKEND_URL}/api/normalization/group-items?group_reference=${encodeURIComponent(group.normalized_reference)}`,
          {
            method: 'GET',
            headers: {
              'Content-Type': 'application/json',
            },
            cache: 'no-store',
          }
        )

        if (itemsResponse.ok) {
          const itemsData = await itemsResponse.json()
          const items = itemsData.items || []

          if (items.length >= 2 && items.length <= maxClusterSize) {
            // Вычисляем среднюю схожесть
            const avgSimilarity = group.avg_confidence || 0.85

            if (avgSimilarity >= similarityThreshold) {
              clusters.push({
                id: `cluster-${group.normalized_reference}`,
                name: group.normalized_name,
                items: items.map((item: any) => ({
                  id: item.source_reference,
                  name: item.source_name,
                  code: item.code,
                  source_reference: item.source_reference,
                  attributes: item.attributes || [],
                  similarity: avgSimilarity,
                  source: item.source_name,
                })),
                similarity: avgSimilarity,
              })
            }
          }
        }
      }
    }

    return NextResponse.json({ clusters })
  } catch (error) {
    console.error('Error fetching duplicates:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const body = await request.json()
    const { action, clusterId, selectedAttributes } = body

    if (action === 'merge') {
      // Логика слияния дубликатов
      // В реальном приложении здесь будет вызов бэкенда для слияния
      return NextResponse.json({ success: true, message: 'Duplicates merged successfully' })
    } else if (action === 'separate') {
      // Логика разделения группы
      return NextResponse.json({ success: true, message: 'Group separated successfully' })
    }

    return NextResponse.json(
      { error: 'Invalid action' },
      { status: 400 }
    )
  } catch (error) {
    console.error('Error processing duplicate action:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

