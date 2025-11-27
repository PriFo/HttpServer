import { useEffect, useRef, useState } from 'react'

interface SSEMetrics {
  type: string
  timestamp: string
  uptime_seconds: number
  throughput: number
  ai_success_rate: number
  cache_hit_rate: number
  batch_queue_size: number
  circuit_breaker_state: string
  checkpoint_progress: number
}

export function useMonitoringSSE(enabled: boolean = true) {
  const [metrics, setMetrics] = useState<SSEMetrics | null>(null)
  const [connected, setConnected] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)

  useEffect(() => {
    if (!enabled) {
      // Disconnect if disabled
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
        setConnected(false)
      }
      return
    }

    const connect = () => {
      try {
        // Close existing connection if any
        if (eventSourceRef.current) {
          eventSourceRef.current.close()
        }

        // Create new EventSource
        const eventSource = new EventSource('/api/monitoring/events')
        eventSourceRef.current = eventSource

        eventSource.onopen = () => {
          try {
            setConnected(true)
            setError(null)
            reconnectAttemptsRef.current = 0
          } catch {
            // Игнорируем ошибки установки состояния
          }
        }

        eventSource.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)

            if (data.type === 'connected') {
              // Игнорируем сообщение о подключении
            } else if (data.type === 'metrics') {
              try {
                setMetrics(data)
              } catch {
                // Игнорируем ошибки установки метрик
              }
            }
          } catch (err) {
            // Безопасная обработка ошибок парсинга
            try {
              setError('Ошибка обработки данных мониторинга')
            } catch {
              // Игнорируем ошибки установки состояния
            }
          }
        }

        eventSource.onerror = (err) => {
          // Проверяем состояние соединения перед логированием ошибки
          if (eventSource.readyState === EventSource.CLOSED) {
            // Соединение закрыто - это нормально при переподключении
            setConnected(false)
            eventSource.close()
            eventSourceRef.current = null

            // Reconnect with exponential backoff
            reconnectAttemptsRef.current += 1
            const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)

            if (reconnectAttemptsRef.current <= 5) {
              try {
                setError(`Соединение потеряно. Переподключение через ${(delay / 1000).toFixed(0)}с...`)
              } catch {
                // Игнорируем ошибки установки состояния
              }

              reconnectTimeoutRef.current = setTimeout(() => {
                connect()
              }, delay)
            } else {
              try {
                setError('Не удалось подключиться после нескольких попыток')
              } catch {
                // Игнорируем ошибки установки состояния
              }
            }
          }
        }
      } catch (err) {
        try {
          setError('Не удалось установить соединение')
        } catch {
          // Игнорируем ошибки установки состояния
        }
      }
    }

    // Initial connection
    connect()

    // Cleanup
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
      }
    }
  }, [enabled])

  const reconnect = () => {
    reconnectAttemptsRef.current = 0
    setError(null)
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
  }

  return {
    metrics,
    connected,
    error,
    reconnect,
  }
}
