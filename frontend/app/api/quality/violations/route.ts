import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    
    // Получаем параметры из query string
    const database = searchParams.get('database')
    const project = searchParams.get('project')
    const limit = searchParams.get('limit')
    const offset = searchParams.get('offset')
    const severity = searchParams.get('severity')
    const category = searchParams.get('category')

    // Формируем URL для бэкенда
    const params = new URLSearchParams()
    if (database) params.append('database', database)
    if (project) params.append('project', project)
    if (limit) params.append('limit', limit)
    if (offset) params.append('offset', offset)
    if (severity) params.append('severity', severity)
    if (category) params.append('category', category)

    const backendUrl = `${BACKEND_URL}/api/quality/violations?${params.toString()}`
    
    console.log('Fetching violations from backend:', backendUrl)

    const data = await fetchJsonServer(backendUrl, {
      timeout: QUALITY_TIMEOUTS.STANDARD,
      cache: 'no-store',
      headers: {
        'Content-Type': 'application/json',
      },
    })

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error in quality violations API route:', error)
    
    const errorMessage = getServerErrorMessage(error, 'Failed to connect to backend')
    const status = getServerErrorStatus(error, 500)

    return NextResponse.json(
      { 
        error: errorMessage,
        details: errorMessage
      },
      { status }
    )
  }
  }

