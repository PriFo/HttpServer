import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ groupId: string }> }
) {
  try {
    const { groupId } = await params

    if (!groupId || isNaN(Number(groupId))) {
      return NextResponse.json(
        { error: 'Invalid group ID' },
        { status: 400 }
      )
    }

    const backendUrl = `${BACKEND_URL}/api/quality/duplicates/${groupId}/merge`
    
    console.log(`Proxying POST /api/quality/duplicates/${groupId}/merge to ${backendUrl}`)

    const data = await fetchJsonServer(backendUrl, {
      method: 'POST',
      timeout: QUALITY_TIMEOUTS.LONG,
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error in quality duplicates merge API route:', error)
    
    const errorMessage = getServerErrorMessage(error, 'Failed to connect to backend')
    const status = getServerErrorStatus(error, 500)

    return NextResponse.json(
      { 
        error: errorMessage,
        details: errorMessage
      },
      { status }
    )
  }
  }

