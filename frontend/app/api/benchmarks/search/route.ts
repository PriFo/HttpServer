import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  try {
    const searchParams = request.nextUrl.searchParams
    const name = searchParams.get('name')
    const type = searchParams.get('type')

    if (!name || !type) {
      return NextResponse.json(
        { error: 'name and type parameters are required' },
        { status: 400 }
      )
    }

    const params = new URLSearchParams({ name, type })
    const response = await fetch(`${BACKEND_URL}/api/benchmarks/search?${params.toString()}`, {
      cache: 'no-store',
    })

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to search benchmarks' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error searching benchmarks:', error)
    return NextResponse.json(
      { error: 'Failed to connect to backend' },
      { status: 500 }
    )
  }
}

