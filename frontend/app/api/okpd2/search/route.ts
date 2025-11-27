import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(request: Request) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { searchParams } = new URL(request.url)
    const query = searchParams.get('q')
    const limit = searchParams.get('limit') || '50'

    if (!query) {
      return NextResponse.json(
        { error: 'Query parameter q is required' },
        { status: 400 }
      )
    }

    const url = `${BACKEND_URL}/api/okpd2/search?q=${encodeURIComponent(query)}&limit=${limit}`

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
      
      return NextResponse.json(
        { error: 'Failed to search OKPD2' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data.results || data)
  } catch (error) {
    console.error('Error searching OKPD2:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

