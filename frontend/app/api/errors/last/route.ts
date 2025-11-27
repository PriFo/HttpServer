import { NextResponse } from 'next/server'

export async function GET(request: Request) {
  try {
    const { searchParams } = new URL(request.url)
    const limit = searchParams.get('limit') || '50'
    
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9999'}/api/errors/last?limit=${limit}`, {
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error('Failed to fetch last errors')
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching last errors:', error)
    return NextResponse.json(
      { error: 'Failed to fetch last errors' },
      { status: 500 }
    )
  }
}

