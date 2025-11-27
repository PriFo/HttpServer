import { NextRequest } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger } from '@/lib/logger'

export const runtime = 'nodejs'
export const dynamic = 'force-dynamic'

// Throttling для логирования ошибок подключения (не более 1 раза в 30 секунд)
let lastConnectionErrorLog = 0
const CONNECTION_ERROR_LOG_INTERVAL = 30000 // 30 секунд

/**
 * SSE (Server-Sent Events) прокси для /api/monitoring/providers/stream
 * Проксирует SSE стрим от бэкенда к клиенту
 */
export async function GET(request: NextRequest) {
  try {
    const backendUrl = getBackendUrl()
    const streamUrl = `${backendUrl}/api/monitoring/providers/stream`

    // Создаем ReadableStream для проксирования SSE
    const stream = new ReadableStream({
      async start(controller) {
        try {
          const response = await fetch(streamUrl, {
            method: 'GET',
            headers: {
              'Accept': 'text/event-stream',
              'Cache-Control': 'no-cache',
            },
            cache: 'no-store',
          })

          if (!response.ok) {
            controller.enqueue(
              new TextEncoder().encode(`data: ${JSON.stringify({ error: `Backend returned ${response.status}` })}\n\n`)
            )
            controller.close()
            return
          }

          if (!response.body) {
            controller.close()
            return
          }

          const reader = response.body.getReader()
          const decoder = new TextDecoder()

          try {
            while (true) {
              const { done, value } = await reader.read()
              
              if (done) {
                controller.close()
                break
              }

              // Передаем данные клиенту
              controller.enqueue(value)
            }
          } catch (streamError: any) {
            // "terminated" - это нормальное поведение при закрытии соединения, не логируем как ошибку
            const isTerminated = streamError?.message?.includes('terminated') || streamError?.name === 'AbortError'
            
            if (!isTerminated) {
              logger.warn('SSE stream error', {
                component: 'SSEStream',
                error: streamError?.message || String(streamError),
                url: streamUrl,
              })
              
              // Отправляем сообщение об ошибке перед закрытием только для реальных ошибок
              try {
                controller.enqueue(
                  new TextEncoder().encode(`data: ${JSON.stringify({ error: 'Stream error', details: streamError?.message })}\n\n`)
                )
              } catch {}
            }
            controller.close()
          } finally {
            reader.releaseLock()
          }
        } catch (fetchError: any) {
          // Проверяем, является ли это ошибкой подключения (ECONNREFUSED)
          const isConnectionError = fetchError?.code === 'ECONNREFUSED' || 
                                   fetchError?.message?.includes('ECONNREFUSED') ||
                                   fetchError?.message?.includes('fetch failed')
          
          // Логируем ошибки подключения с throttling (не более 1 раза в 30 секунд)
          const now = Date.now()
          if (isConnectionError) {
            if (now - lastConnectionErrorLog > CONNECTION_ERROR_LOG_INTERVAL) {
              logger.warn('SSE fetch error (backend unavailable)', {
                component: 'SSEStream',
                error: fetchError?.message || String(fetchError),
                url: streamUrl,
                note: 'Backend server appears to be unavailable. Will retry silently.',
              })
              lastConnectionErrorLog = now
            }
          } else {
            // Для других ошибок логируем всегда
            logger.error('SSE fetch error', {
              component: 'SSEStream',
              error: fetchError?.message || String(fetchError),
              url: streamUrl,
            })
          }
          
          try {
            controller.enqueue(
              new TextEncoder().encode(`data: ${JSON.stringify({ error: 'Connection error', details: fetchError?.message })}\n\n`)
            )
          } catch {}
          controller.close()
        }
      },
      cancel() {
        // Очистка при отмене запроса
      },
    })

    return new Response(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-transform',
        'Connection': 'keep-alive',
        'X-Accel-Buffering': 'no',
      },
    })
  } catch (error: any) {
    logger.error('SSE stream fatal error', {
      component: 'SSEStream',
      error: error?.message || String(error),
    })
    
    return new Response(
      `data: ${JSON.stringify({ error: 'Failed to establish stream', details: error?.message })}\n\n`,
      {
        status: 500,
        headers: {
          'Content-Type': 'text/event-stream',
        },
      }
    )
  }
}

