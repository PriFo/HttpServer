import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function POST(request: Request) {
  try {
    const contentType = request.headers.get('content-type') || ''
    
    // Если это multipart/form-data, передаем как есть
    if (contentType.includes('multipart/form-data')) {
      const formData = await request.formData()
      
      const response = await fetch(`${BACKEND_URL}/api/okpd2/load-from-file`, {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const errorText = await response.text()
        return NextResponse.json(
          { error: errorText || 'Failed to load OKPD2' },
          { status: response.status }
        )
      }

      const data = await response.json()
      return NextResponse.json(data)
    }
    
    // Если это JSON, передаем как есть
    const body = await request.json()
    
    const response = await fetch(`${BACKEND_URL}/api/okpd2/load-from-file`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to load OKPD2' }))
      return NextResponse.json(
        { error: errorData.error || 'Failed to load OKPD2' },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error loading OKPD2:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

