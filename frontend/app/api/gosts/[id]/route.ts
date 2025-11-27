import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { id } = await params

    if (!id) {
      return NextResponse.json(
        { error: 'GOST ID is required' },
        { status: 400 }
      )
    }

    const url = `${BACKEND_URL}/api/gosts/${id}`

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to fetch GOST' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching GOST:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

