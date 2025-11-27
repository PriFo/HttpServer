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
    const timeRange = searchParams.get('timeRange') || '24h'

    const BACKEND_URL = getBackendUrl()

    // Получаем статистику нормализации
    const statsResponse = await fetch(
      `${BACKEND_URL}/api/clients/${clientId}/projects/${projectId}/normalization/stats?timeRange=${timeRange}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    )

    if (!statsResponse.ok) {
      const errorText = await statsResponse.text()
      console.error('Backend stats error:', errorText)
      return NextResponse.json(
        { error: 'Failed to fetch normalization stats' },
        { status: statsResponse.status }
      )
    }

    const statsData = await statsResponse.json()

    // Получаем группы для подсчета
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

    let totalGroups = 0
    let avgQuality = 0
    let totalProcessed = 0
    let issuesCount = 0

    if (groupsResponse.ok) {
      const groupsData = await groupsResponse.json()
      totalGroups = groupsData.groups?.length || 0
      
      // Вычисляем среднее качество
      if (groupsData.groups && groupsData.groups.length > 0) {
        const qualitySum = groupsData.groups.reduce((sum: number, group: any) => {
          return sum + (group.avg_confidence || 0.85)
        }, 0)
        avgQuality = qualitySum / groupsData.groups.length
      }

      // Подсчитываем обработанные записи
      totalProcessed = groupsData.groups?.reduce((sum: number, group: any) => {
        return sum + (group.merged_count || 0)
      }, 0) || 0

      // Подсчитываем проблемы (группы с низким качеством)
      issuesCount = groupsData.groups?.filter((group: any) => {
        return (group.avg_confidence || 0.85) < 0.7
      }).length || 0
    }

    // Объединяем данные
    const result = {
      ...statsData,
      totalGroups,
      avgQuality,
      totalProcessed,
      issuesCount,
      groupsTrend: statsData.groupsTrend || 0,
      processedTrend: statsData.processedTrend || 0,
    }

    return NextResponse.json(result)
  } catch (error) {
    console.error('Error fetching normalization stats:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

