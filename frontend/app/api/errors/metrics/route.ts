import { NextResponse } from 'next/server'

export async function GET() {
  try {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9999'}/api/errors/metrics`, {
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error('Failed to fetch error metrics')
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching error metrics:', error)
    return NextResponse.json(
      { error: 'Failed to fetch error metrics' },
      { status: 500 }
    )
  }
}

