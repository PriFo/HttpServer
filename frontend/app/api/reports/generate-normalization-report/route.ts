import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

export async function POST(request: NextRequest) {
  try {
    const body = await request.json().catch(() => ({}))
    
    const backendResponse = await fetch(
      `${API_BASE_URL}/api/reports/generate-normalization-report`,
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(body),
      }
    )

    if (!backendResponse.ok) {
      const errorData = await backendResponse.json().catch(() => ({ error: 'Failed to generate normalization report' }))
      return NextResponse.json(
        { error: errorData.error || 'Failed to generate normalization report' },
        { status: backendResponse.status }
      )
    }

    const data = await backendResponse.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error generating normalization report:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

