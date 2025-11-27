import { NextResponse } from 'next/server'

export async function POST() {
  try {
    const response = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:9999'}/api/errors/reset`, {
      method: 'POST',
      cache: 'no-store',
    })

    if (!response.ok) {
      throw new Error('Failed to reset error metrics')
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error resetting error metrics:', error)
    return NextResponse.json(
      { error: 'Failed to reset error metrics' },
      { status: 500 }
    )
  }
}

