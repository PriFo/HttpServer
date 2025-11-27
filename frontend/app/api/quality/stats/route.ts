import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const database = searchParams.get('database')
    const project = searchParams.get('project')

    // Создаем URL для backend с параметрами database и project, если они указаны
    let backendUrl = `${BACKEND_URL}/api/quality/stats`
    const params = new URLSearchParams()
    if (database) {
      params.set('database', database)
    }
    if (project) {
      params.set('project', project)
    }
    if (params.toString()) {
      backendUrl += `?${params.toString()}`
    }

    const data = await fetchJsonServer(backendUrl, {
      timeout: QUALITY_TIMEOUTS.FAST,
      cache: 'no-store',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching quality stats:', error)
    
    const errorMessage = getServerErrorMessage(error, 'Failed to connect to backend')
    const status = getServerErrorStatus(error, 500)

    return NextResponse.json(
      { error: errorMessage },
      { status }
    )
  }
}
