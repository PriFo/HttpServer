import { NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:9999'

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url)
    const database = searchParams.get('database')

    let url = `${BACKEND_URL}/api/kpved/stats`
    if (database) {
      url += `?database=${encodeURIComponent(database)}`
    }

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to fetch KPVED stats' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching KPVED stats:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}
