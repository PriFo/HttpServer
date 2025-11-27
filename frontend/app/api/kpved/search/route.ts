import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(request: Request) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { searchParams } = new URL(request.url)
    const q = searchParams.get('q')
    const limit = searchParams.get('limit')

    if (!q) {
      return NextResponse.json(
        { error: 'Search query (q) is required' },
        { status: 400 }
      )
    }

    const params = new URLSearchParams()
    params.append('q', q)
    if (limit) params.append('limit', limit)

    const url = `${BACKEND_URL}/api/kpved/search?${params.toString()}`

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      // Для 404 возвращаем пустые результаты вместо ошибки
      if (response.status === 404) {
        return NextResponse.json({
          results: [],
          total: 0,
        })
      }
      
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to search KPVED' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error searching KPVED:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}
