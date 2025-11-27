import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function POST(request: Request) {
  try {
    const BACKEND_URL = getBackendUrl()
    const formData = await request.formData()
    
    const file = formData.get('file') as File
    const sourceType = formData.get('source_type') as string
    const sourceUrl = formData.get('source_url') as string

    if (!file) {
      return NextResponse.json(
        { error: 'File is required' },
        { status: 400 }
      )
    }

    if (!sourceType) {
      return NextResponse.json(
        { error: 'Source type is required' },
        { status: 400 }
      )
    }

    // Создаем новый FormData для передачи в backend
    const backendFormData = new FormData()
    backendFormData.append('file', file)
    backendFormData.append('source_type', sourceType)
    if (sourceUrl) {
      backendFormData.append('source_url', sourceUrl)
    }

    const url = `${BACKEND_URL}/api/gosts/import`

    const response = await fetch(url, {
      method: 'POST',
      body: backendFormData,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to import GOSTs' }))
      return NextResponse.json(
        errorData,
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error importing GOSTs:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

