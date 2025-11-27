import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(request: Request) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { searchParams } = new URL(request.url)
    const parent = searchParams.get('parent')
    const level = searchParams.get('level')

    let url = `${BACKEND_URL}/api/okpd2/hierarchy`
    const params = new URLSearchParams()

    if (parent) params.append('parent', parent)
    if (level) params.append('level', level)

    if (params.toString()) {
      url += `?${params.toString()}`
    }

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        return NextResponse.json({
          nodes: [],
          total: 0,
        })
      }
      
      return NextResponse.json(
        { error: 'Failed to fetch OKPD2 hierarchy' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching OKPD2 hierarchy:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

