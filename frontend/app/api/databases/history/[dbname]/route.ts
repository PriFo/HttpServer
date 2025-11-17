import { NextRequest, NextResponse } from 'next/server'

const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:9999'

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ dbname: string }> }
) {
  try {
    const { dbname } = await params
    const response = await fetch(`${BACKEND_URL}/api/databases/history/${dbname}`, {
      cache: 'no-store'
    })

    if (!response.ok) {
      return NextResponse.json(
        { error: 'Failed to fetch database history' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching database history:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

