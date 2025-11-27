import { NextRequest } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

/**
 * SSE эндпоинт для стриминга трассировки воркеров
 * 
 * Проксирует SSE соединение к Go бэкенду и транслирует события трассировки
 * в реальном времени через Server-Sent Events
 */
export async function GET(request: NextRequest) {
  const { searchParams } = new URL(request.url)
  const traceId = searchParams.get('trace_id')

  if (!traceId) {
    return new Response(
      JSON.stringify({ error: 'trace_id is required' }),
      {
        status: 400,
        headers: { 'Content-Type': 'application/json' },
      }
    )
  }

  const backendUrl = getBackendUrl()
  const backendSSEUrl = `${backendUrl}/api/internal/worker-trace/stream?trace_id=${encodeURIComponent(traceId)}`

  try {
    // Создаем ReadableStream для проксирования SSE
    const stream = new ReadableStream({
      async start(controller) {
        const encoder = new TextEncoder()
        
        // Функция для отправки события SSE
        const sendEvent = (event: string, data: Record<string, unknown>) => {
          try {
            const message = `event: ${event}\ndata: ${JSON.stringify(data)}\n\n`
            controller.enqueue(encoder.encode(message))
          } catch (error) {
            console.error('Error encoding SSE event:', error)
          }
        }

        let reader: ReadableStreamDefaultReader<Uint8Array> | null = null;

        try {
          // Проксируем SSE соединение к бэкенду
          // В серверном окружении используем fetch с ReadableStream
          const response = await fetch(backendSSEUrl, {
            headers: {
              'Accept': 'text/event-stream',
              'Cache-Control': 'no-cache',
            },
            signal: request.signal, // Поддержка отмены при отключении клиента
          });

          if (!response.ok) {
            sendEvent('error', { 
              message: `Backend error: ${response.status} ${response.statusText}` 
            });
            controller.close();
            return;
          }

          reader = response.body?.getReader() || null;
          const decoder = new TextDecoder();

          if (!reader) {
            sendEvent('error', { message: 'Failed to get response reader' });
            controller.close();
            return;
          }

          // Читаем поток данных от бэкенда и проксируем его клиенту
          while (true) {
            const { done, value } = await reader.read();

            if (done) {
              sendEvent('finished', { message: 'Stream ended' });
              controller.close();
              break;
            }

            // Декодируем данные и отправляем клиенту
            const chunk = decoder.decode(value, { stream: true });
            controller.enqueue(encoder.encode(chunk));
          }

        } catch (error: unknown) {
          console.error('Error in worker trace stream proxy:', error);
          
          // Если это не ошибка отмены, отправляем ошибку клиенту
          const err = error as { name?: string; message?: string };
          if (err.name !== 'AbortError') {
            sendEvent('error', { 
              message: err.message || 'Connection error',
              details: String(error),
            });
          }
          
          controller.close();
        } finally {
          // Очистка ресурсов
          if (reader) {
            try {
              reader.releaseLock();
            } catch {
              // Игнорируем ошибки при освобождении
            }
          }
        }
      },
      
      cancel() {
        // Вызывается при отмене клиентом
        console.log('SSE stream cancelled by client')
      },
    })

    return new Response(stream, {
      headers: {
        'Content-Type': 'text/event-stream',
        'Cache-Control': 'no-cache, no-transform',
        'Connection': 'keep-alive',
        'X-Accel-Buffering': 'no', // Отключаем буферизацию в nginx
        'Access-Control-Allow-Origin': '*',
        'Access-Control-Allow-Headers': 'Cache-Control',
      },
    })
  } catch (error: unknown) {
    console.error('Error creating SSE stream:', error)
    const err = error as { message?: string };
    return new Response(
      JSON.stringify({ error: 'Failed to create SSE stream', message: err.message || String(error) }),
      {
        status: 500,
        headers: { 'Content-Type': 'application/json' },
      }
    )
  }
}

