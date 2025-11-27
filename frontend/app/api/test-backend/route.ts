import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET() {
  const backendUrl = getBackendUrl()
  const testUrl = `${backendUrl}/health`
  
  const diagnostics = {
    backendUrl,
    testUrl,
    nodeEnv: process.env.NODE_ENV,
    hasBackendUrl: !!process.env.BACKEND_URL,
    hasNextPublicBackendUrl: !!process.env.NEXT_PUBLIC_BACKEND_URL,
    timestamp: new Date().toISOString(),
  }
  
  try {
    console.log('[Test Backend] Starting fetch to:', testUrl)
    
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 3000)
    
    const response = await fetch(testUrl, {
      method: 'GET',
      cache: 'no-store',
      signal: controller.signal,
      headers: {
        'Accept': 'application/json',
      },
    })
    
    clearTimeout(timeoutId)
    
    const responseText = await response.text()
    let responseData
    try {
      responseData = JSON.parse(responseText)
    } catch {
      responseData = responseText
    }
    
    return NextResponse.json({
      success: true,
      status: response.status,
      statusText: response.statusText,
      backendResponse: responseData,
      diagnostics,
    })
  } catch (error: any) {
    console.error('[Test Backend] Error:', error)
    
    return NextResponse.json({
      success: false,
      error: {
        name: error?.name,
        message: error?.message,
        stack: process.env.NODE_ENV === 'development' ? error?.stack : undefined,
      },
      diagnostics,
    }, { status: 500 })
  }
}

