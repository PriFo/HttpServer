import { NextResponse } from 'next/server'

export async function GET() {
  try {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9999'}/api/errors/by-code`, {
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error('Failed to fetch errors by code')
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching errors by code:', error)
    return NextResponse.json(
      { error: 'Failed to fetch errors by code' },
      { status: 500 }
    )
  }
}

