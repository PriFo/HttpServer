import { NextRequest, NextResponse } from 'next/server'

const API_BASE = process.env.NEXT_PUBLIC_BACKEND_URL || 'http://localhost:9999'

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    
    const response = await fetch(`${API_BASE}/api/kpved/load`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      const errorText = await response.text()
      console.error('Error loading KPVED:', errorText)
      return new NextResponse(errorText, { status: response.status })
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error loading KPVED:', error)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Failed to load KPVED' },
      { status: 500 }
    )
  }
}

