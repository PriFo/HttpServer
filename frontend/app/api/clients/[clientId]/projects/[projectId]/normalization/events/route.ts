import { NextRequest } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

// SSE endpoint - проксирует к backend
export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const resolvedParams = await params
    const { clientId, projectId } = resolvedParams

    // Валидация параметров
    if (!clientId || !projectId) {
      return new Response('Missing clientId or projectId', { status: 400 })
    }

    // Проверяем, что это числа
    const clientIdNum = parseInt(clientId, 10)
    const projectIdNum = parseInt(projectId, 10)
    
    if (isNaN(clientIdNum) || isNaN(projectIdNum)) {
      return new Response('Invalid clientId or projectId', { status: 400 })
    }

    // Проксируем SSE запрос к backend
    const backendUrl = `${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/events`
    
    // Для SSE нужно использовать прямой прокси через fetch с stream
    const backendResponse = await fetch(backendUrl, {
      method: 'GET',
      headers: {
        'Cache-Control': 'no-cache',
      },
      cache: 'no-store',
    })

    if (!backendResponse.ok) {
      return new Response('Failed to connect to events stream', { 
        status: backendResponse.status 
      })
    }

    // Создаем ReadableStream для проксирования SSE
    const stream = new ReadableStream({
      async start(controller) {
        const reader = backendResponse.body?.getReader()
        const decoder = new TextDecoder()

        if (!reader) {
          controller.close()
          return
        }

        try {
          while (true) {
            const { done, value } = await reader.read()
            
            if (done) {
              controller.close()
              break
            }

            const chunk = decoder.decode(value, { stream: true })
            controller.enqueue(new TextEncoder().encode(chunk))
          }
        } catch (error) {
          console.error('Error proxying SSE stream:', error)
          controller.error(error)
        }
      },
    })

    // Возвращаем поток с правильными заголовками для SSE
    return new Response(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache',
        'Connection': 'keep-alive',
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Cache-Control',
      },
    })
  } catch (error) {
    console.error('Error in events route:', error)
    return new Response(
      `Failed to connect to events stream: ${error instanceof Error ? error.message : 'Unknown error'}`,
      { status: 500 }
    )
  }
}

