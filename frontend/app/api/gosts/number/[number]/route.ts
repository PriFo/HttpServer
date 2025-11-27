import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ number: string }> }
) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { number: numberParam } = await params
    const number = decodeURIComponent(numberParam)

    if (!number) {
      return NextResponse.json(
        { error: 'GOST number is required' },
        { status: 400 }
      )
    }

    const url = `${BACKEND_URL}/api/gosts/number/${encodeURIComponent(number)}`

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to fetch GOST by number' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching GOST by number:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

