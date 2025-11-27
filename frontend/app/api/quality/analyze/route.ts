import { NextRequest, NextResponse } from 'next/server'
import { qualityAnalyzeSchema, validateRequest, formatValidationError } from '@/lib/validation'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()

    console.log('Proxying POST /api/quality/analyze to backend')

    // Validate request body
    const validation = validateRequest(qualityAnalyzeSchema, body)
    if (!validation.success) {
      return NextResponse.json(
        {
          error: 'Validation failed',
          details: formatValidationError(validation.details)
        },
        { status: 400 }
      )
    }

    const data = await fetchJsonServer(`${BACKEND_URL}/api/quality/analyze`, {
      method: 'POST',
      timeout: QUALITY_TIMEOUTS.VERY_LONG,
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(validation.data),
    })

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error starting quality analysis:', error)
    
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

