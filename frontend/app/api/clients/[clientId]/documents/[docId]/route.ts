import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(
  request: Request,
  { params }: { params: Promise<{ clientId: string; docId: string }> }
) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { clientId, docId } = await params

    const url = `${BACKEND_URL}/api/clients/${clientId}/documents/${docId}`

    const response = await fetch(url, {
      cache: 'no-store',
    })

    if (!response.ok) {
      const errorText = await response.text()
      return NextResponse.json(
        { error: errorText || 'Failed to fetch document' },
        { status: response.status }
      )
    }

    // Return file as blob
    const blob = await response.blob()
    const contentType = response.headers.get('content-type') || 'application/octet-stream'
    const contentDisposition = response.headers.get('content-disposition')

    return new NextResponse(blob, {
      headers: {
        'Content-Type': contentType,
        'Content-Disposition': contentDisposition || 'attachment',
      },
    })
  } catch (error) {
    console.error('Error downloading client document:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function DELETE(
  request: Request,
  { params }: { params: Promise<{ clientId: string; docId: string }> }
) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { clientId, docId } = await params

    const url = `${BACKEND_URL}/api/clients/${clientId}/documents/${docId}`

    const response = await fetch(url, {
      method: 'DELETE',
    })

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({ error: 'Failed to delete document' }))
      return NextResponse.json(
        errorData,
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error deleting client document:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

