import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(request: Request) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { searchParams } = new URL(request.url)
    const q = searchParams.get('q')
    const limit = searchParams.get('limit')
    const offset = searchParams.get('offset')

    if (!q) {
      return NextResponse.json(
        { error: 'Search query (q) is required' },
        { status: 400 }
      )
    }

    const params = new URLSearchParams()
    params.append('q', q)
    if (limit) params.append('limit', limit)
    if (offset) params.append('offset', offset)

    const url = `${BACKEND_URL}/api/gosts/search?${params.toString()}`

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to search GOSTs' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error searching GOSTs:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

