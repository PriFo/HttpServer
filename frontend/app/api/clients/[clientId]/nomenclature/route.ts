import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string }> }
) {
  try {
    const { clientId } = await params
    const { searchParams } = new URL(request.url)
    
    const page = searchParams.get('page') || '1'
    const limit = searchParams.get('limit') || '20'
    const search = searchParams.get('search')

    let url = `${BACKEND_URL}/api/clients/${clientId}/nomenclature?page=${page}&limit=${limit}`
    if (search) {
      url += `&search=${encodeURIComponent(search)}`
    }

    const response = await fetch(url, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text().catch(() => 'Unknown error')
      console.error(`Backend error (${response.status}):`, errorText)
      return NextResponse.json(
        { error: 'Failed to fetch nomenclature', details: errorText },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching nomenclature:', error)
    return NextResponse.json(
      { error: 'Failed to connect to backend', details: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    )
  }
}

