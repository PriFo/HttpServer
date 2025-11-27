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
    const format = searchParams.get('format') || 'excel'
    const includeAttributes = searchParams.get('include_attributes') === 'true'
    const includeMetadata = searchParams.get('include_metadata') === 'true'
    const dataType = searchParams.get('data_type') || 'groups'

    const BACKEND_URL = getBackendUrl()

    // Определяем endpoint в зависимости от типа данных
    let exportEndpoint = ''
    if (dataType === 'nomenclature') {
      exportEndpoint = `${BACKEND_URL}/api/clients/${clientId}/projects/${projectId}/nomenclature/export`
    } else if (dataType === 'counterparties') {
      exportEndpoint = `${BACKEND_URL}/api/counterparties/normalized/export?client_id=${clientId}&project_id=${projectId}`
    } else {
      exportEndpoint = `${BACKEND_URL}/api/normalization/export?client_id=${clientId}&project_id=${projectId}`
    }

    const exportResponse = await fetch(
      `${exportEndpoint}&format=${format}&include_attributes=${includeAttributes}&include_metadata=${includeMetadata}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    )

    if (!exportResponse.ok) {
      const errorText = await exportResponse.text()
      console.error('Backend export error:', errorText)
      return NextResponse.json(
        { error: 'Failed to export data' },
        { status: exportResponse.status }
      )
    }

    // Если это файл, возвращаем его напрямую
    const contentType = exportResponse.headers.get('content-type')
    if (contentType && contentType.includes('application/')) {
      const blob = await exportResponse.blob()
      return new NextResponse(blob, {
        headers: {
          'Content-Type': contentType,
          'Content-Disposition': `attachment; filename="export_${dataType}_${new Date().toISOString().split('T')[0]}.${format === 'json' ? 'json' : format === 'csv' ? 'csv' : 'xlsx'}"`,
        },
      })
    }

    const data = await exportResponse.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error exporting data:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

