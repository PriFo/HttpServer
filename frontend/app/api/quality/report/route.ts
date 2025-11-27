import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const API_BASE_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const database = searchParams.get('database')

  if (!database) {
    return NextResponse.json(
      { error: 'database parameter is required' },
      { status: 400 }
    )
  }

  try {
    const data = await fetchJsonServer(
      `${API_BASE_URL}/api/quality/report?database=${encodeURIComponent(database)}`,
      {
        timeout: QUALITY_TIMEOUTS.STANDARD,
        cache: 'no-store',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    )

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching quality report:', error)
    
    const errorMessage = getServerErrorMessage(error, 'Failed to connect to backend')
    const status = getServerErrorStatus(error, 500)

    return NextResponse.json(
      { error: errorMessage },
      { status }
    )
  }
}

