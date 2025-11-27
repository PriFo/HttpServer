import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()
const REQUEST_TIMEOUT_MS = 7000

interface WorkerConfigFallback {
  providers: Record<string, any>
  default_provider: string
  default_model: string
  global_max_workers: number
  isFallback: true
  fallbackReason: string
  lastSync: string
}

const buildFallbackConfig = (reason: string): WorkerConfigFallback => {
  const timestamp = new Date().toISOString()

  return {
    providers: {
      arliai: {
        name: 'arliai',
        enabled: false,
        priority: 1,
        max_workers: 2,
        rate_limit: 30,
        timeout: '15s',
        base_url: 'https://api.arliai.com',
        has_api_key: false,
        models: [
          {
            name: 'glm-4.5-air',
            provider: 'arliai',
            enabled: false,
            priority: 1,
            speed: 'fast',
            quality: 'balanced',
            temperature: 0.2,
          },
        ],
      },
      openrouter: {
        name: 'openrouter',
        enabled: false,
        priority: 2,
        max_workers: 1,
        rate_limit: 15,
        timeout: '20s',
        base_url: 'https://openrouter.ai/api/v1',
        has_api_key: false,
        models: [
          {
            name: 'anthropic/claude-3.5-sonnet',
            provider: 'openrouter',
            enabled: false,
            priority: 2,
            quality: 'premium',
            speed: 'standard',
          },
        ],
      },
      huggingface: {
        name: 'huggingface',
        enabled: false,
        priority: 3,
        max_workers: 1,
        rate_limit: 10,
        timeout: '25s',
        base_url: 'https://api-inference.huggingface.co',
        has_api_key: false,
        models: [
          {
            name: 'meta-llama/Llama-3.1-8B-Instruct',
            provider: 'huggingface',
            enabled: false,
            priority: 3,
            quality: 'experimental',
            speed: 'slow',
          },
        ],
      },
    },
    default_provider: 'arliai',
    default_model: 'glm-4.5-air',
    global_max_workers: 4,
    isFallback: true,
    fallbackReason: reason,
    lastSync: timestamp,
  }
}

export async function GET() {
  const controller = new AbortController()
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS)

  try {
    const response = await fetch(`${BACKEND_URL}/api/workers/config`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      cache: 'no-store',
      signal: controller.signal,
    })

    clearTimeout(timeout)

    if (!response.ok) {
      const errorText = await response.text().catch(() => null)
      let errorMessage = `Backend responded with ${response.status}`

      if (errorText) {
        try {
          const parsed = JSON.parse(errorText)
          errorMessage = parsed.error || parsed.message || errorText
        } catch {
          errorMessage = errorText
        }
      }

      console.warn('[Workers API] Backend returned non-200 status:', errorMessage)
      return NextResponse.json(buildFallbackConfig(errorMessage))
    }

    const data = await response.json()
    return NextResponse.json({
      ...data,
      lastSync: new Date().toISOString(),
    })
  } catch (error) {
    clearTimeout(timeout)

    const errorMessage =
      error instanceof Error ? error.message : 'Failed to connect to backend'

    console.error('[Workers API] Error fetching worker config:', errorMessage)
    return NextResponse.json(
      buildFallbackConfig(
        `Не удалось подключиться к backend: ${errorMessage}. Проверьте, запущен ли сервер на порту 9999.`,
      ),
    )
  }
}

export async function POST(request: NextRequest) {
  try {
    const body = await request.json()

    const response = await fetch(`${BACKEND_URL}/api/workers/config/update`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(body),
    })

    if (!response.ok) {
      let errorMessage = 'Failed to update worker config'
      try {
        const contentType = response.headers.get('content-type')
        if (contentType && contentType.includes('application/json')) {
          const errorData = await response.json()
          errorMessage = errorData.error || errorData.message || errorMessage
        } else {
          const errorText = await response.text()
          if (errorText) {
            try {
              const errorJson = JSON.parse(errorText)
              errorMessage = errorJson.error || errorJson.message || errorText
            } catch {
              errorMessage = errorText || errorMessage
            }
          }
        }
      } catch (err) {
        console.error('Error parsing error response:', err)
        errorMessage = `Ошибка ${response.status}: ${response.statusText || 'Failed to update worker config'}`
      }
      
      return NextResponse.json(
        { error: errorMessage },
        { status: response.status || 500 }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
  } catch (error) {
    console.error('Error updating worker config:', error)
    return NextResponse.json(
      { error: 'Failed to update worker config' },
      { status: 500 }
    )
  }
}

