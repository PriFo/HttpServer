import { NextResponse } from 'next/server'

const API_BASE_URL = process.env.BACKEND_URL || 'http://localhost:9999'

export async function POST(
  request: Request,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const response = await fetch(`${API_BASE_URL}/api/clients/${clientId}/projects/${projectId}/normalization/start`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}))
      return NextResponse.json(
        { error: errorData.error || 'Failed to start normalization' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error starting normalization:', error)
    return NextResponse.json(
      { error: 'Failed to start normalization' },
      { status: 500 }
    )
  }
}

