import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { id } = await params
    const { searchParams } = new URL(request.url)
    const docId = searchParams.get('doc_id')

    if (!id) {
      return NextResponse.json(
        { error: 'GOST ID is required' },
        { status: 400 }
      )
    }

    const url = `${BACKEND_URL}/api/gosts/${id}/document${docId ? `?doc_id=${docId}` : ''}`

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to fetch GOST document' },
        { status: response.status }
      )
    }

    // Если это файл, возвращаем его напрямую
    const contentType = response.headers.get('content-type')
    if (contentType && (contentType.startsWith('application/pdf') || contentType.startsWith('application/msword') || contentType.startsWith('application/vnd.openxmlformats'))) {
      const blob = await response.blob()
      return new NextResponse(blob, {
        headers: {
          'Content-Type': contentType,
        },
      })
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error fetching GOST document:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function POST(
  request: Request,
  { params }: { params: Promise<{ id: string }> }
) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { id } = await params

    if (!id) {
      return NextResponse.json(
        { error: 'GOST ID is required' },
        { status: 400 }
      )
    }

    const formData = await request.formData()
    const file = formData.get('file') as File

    if (!file) {
      return NextResponse.json(
        { error: 'File is required' },
        { status: 400 }
      )
    }

    // Создаем новый FormData для передачи в backend
    const backendFormData = new FormData()
    backendFormData.append('file', file)

    const url = `${BACKEND_URL}/api/gosts/${id}/document`

    const response = await fetch(url, {
      method: 'POST',
      body: backendFormData,
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to upload GOST document' }))
      return NextResponse.json(
        errorData,
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error uploading GOST document:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

