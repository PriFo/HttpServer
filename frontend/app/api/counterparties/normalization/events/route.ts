import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

// SSE endpoint для контрагентов - проксирует к backend
export async function GET(request: NextRequest) {
  try {
    // Получаем параметры из query string
    const { searchParams } = new URL(request.url)
    const clientId = searchParams.get('client_id')
    const projectId = searchParams.get('project_id')

    // Если есть client_id и project_id, используем специальный эндпоинт
    if (clientId && projectId) {
      const clientIdNum = parseInt(clientId, 10)
      const projectIdNum = parseInt(projectId, 10)
      
      if (!isNaN(clientIdNum) && !isNaN(projectIdNum)) {
        const backendUrl = `${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/events`
        
        // Проксируем SSE запрос к backend
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
      }
    }

    // Fallback: используем общий endpoint для контрагентов
    const backendUrl = `${API_BASE_URL}/api/counterparties/normalization/events`
    
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
    console.error('Error in counterparties events route:', error)
    return new Response(
      `Failed to connect to events stream: ${error instanceof Error ? error.message : 'Unknown error'}`,
      { status: 500 }
    )
  }
}

