import { NextRequest, NextResponse } from 'next/server'
import { violationResolveSchema, validateRequest, formatValidationError } from '@/lib/validation'
import { getBackendUrl } from '@/lib/api-config'
import { fetchJsonServer, getServerErrorMessage, getServerErrorStatus } from '@/lib/fetch-utils-server'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'

const BACKEND_URL = getBackendUrl()

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ violationId: string }> }
) {
  try {
    const { violationId } = await params

    if (!violationId || isNaN(Number(violationId))) {
      return NextResponse.json(
        { error: 'Invalid violation ID' },
        { status: 400 }
      )
    }

    const body = await request.json().catch(() => ({}))

    // Validate request body
    const validation = validateRequest(violationResolveSchema, body)
    if (!validation.success) {
      return NextResponse.json(
        {
          error: 'Validation failed',
          details: formatValidationError(validation.details)
        },
        { status: 400 }
      )
    }
    
    const backendUrl = `${BACKEND_URL}/api/quality/violations/${violationId}`
    
    console.log('Resolving violation:', backendUrl)

    const data = await fetchJsonServer(backendUrl, {
      method: 'POST',
      timeout: QUALITY_TIMEOUTS.LONG,
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(validation.data),
    })

    return NextResponse.json(data)
  } catch (error) {
    console.error('Error in quality violations resolve API route:', error)
    
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

