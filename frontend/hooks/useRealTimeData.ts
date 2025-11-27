import { useEffect, useRef } from 'react'
import { useDashboardStore } from '@/stores/dashboard-store'
import { getBackendUrl } from '@/lib/api-config'
import type { MonitoringData } from '@/types/monitoring'

export function useRealTimeData() {
  const {
    isRealTimeEnabled,
    setProviderMetrics,
    setMonitoringSystemStats,
    setError,
    setRealTimeEnabled,
  } = useDashboardStore()
  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const availabilityCheckedRef = useRef(false)
  const availabilityFailedRef = useRef(false)

  useEffect(() => {
    if (!isRealTimeEnabled) {
      // Отключаемся, если реальное время отключено
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
      return
    }

    let cancelled = false

    const verifyAvailability = async () => {
      if (availabilityCheckedRef.current && availabilityFailedRef.current) {
        return false
      }

      if (availabilityCheckedRef.current && !availabilityFailedRef.current) {
        return true
      }

      availabilityCheckedRef.current = true
      const backendUrl = getBackendUrl()

      try {
        const response = await fetch(`${backendUrl}/api/monitoring/providers`, {
          cache: 'no-store',
          headers: {
            'Content-Type': 'application/json',
          },
        })

        if (!response.ok) {
          throw new Error(`Monitoring providers responded with status ${response.status}`)
        }

        availabilityFailedRef.current = false
        return true
      } catch (err) {
        availabilityFailedRef.current = true
        // Не логируем ошибки подключения - это ожидаемо, если бэкенд не запущен
        // Только обновляем состояние без шумных сообщений в консоли
        if (!cancelled) {
          setError('Сервер мониторинга недоступен. Реальное время отключено.')
          setRealTimeEnabled(false)
        }
        return false
      }
    }

    const connect = () => {
      try {
        // Закрываем существующее соединение, если есть
        if (eventSourceRef.current) {
          eventSourceRef.current.close()
        }

        // Используем Next.js API route для проксирования SSE
        const eventSource = new EventSource('/api/monitoring/providers/stream')
        eventSourceRef.current = eventSource

        eventSource.onopen = () => {
          try {
            setError(null)
            reconnectAttemptsRef.current = 0
          } catch {
            // Игнорируем ошибки установки состояния
          }
        }

        eventSource.onmessage = (event) => {
          try {
            const parsed = JSON.parse(event.data)
            
            // Пропускаем сообщение о подключении
            if (parsed.type === 'connected') {
              return
            }

            // Обрабатываем ошибки
            if (parsed.error) {
              try {
                const errorMessage = typeof parsed.error === 'string' ? parsed.error : 'Ошибка от сервера'
                setError(errorMessage)
              } catch {
                // Игнорируем ошибки установки состояния
              }
              return
            }

            const monitoringData = parsed as MonitoringData

            // Обновляем метрики провайдеров
            if (monitoringData.providers) {
              try {
                setProviderMetrics(monitoringData.providers)
              } catch {
                // Игнорируем ошибки обновления метрик
              }
            }

            // Обновляем системную статистику
            if (monitoringData.system) {
              try {
                setMonitoringSystemStats(monitoringData.system)
              } catch {
                // Игнорируем ошибки обновления статистики
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
          try {
            // Проверяем состояние соединения
            if (eventSource.readyState === EventSource.CLOSED) {
              // Соединение закрыто - пытаемся переподключиться
              // Не показываем ошибку сразу, только после нескольких попыток
              eventSource.close()
              eventSourceRef.current = null

              // Переподключение с экспоненциальной задержкой
              reconnectAttemptsRef.current += 1
              const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)

              if (reconnectAttemptsRef.current <= 5) {
                // Показываем сообщение только после 3 попыток
                if (reconnectAttemptsRef.current >= 3) {
                  setError('Попытка переподключения к серверу мониторинга...')
                }
                reconnectTimeoutRef.current = setTimeout(() => {
                  connect()
                }, delay)
              } else {
                // После 5 попыток показываем финальное сообщение
                setError('Не удалось подключиться к серверу мониторинга. Реальное время отключено.')
                setRealTimeEnabled(false)
              }
            } else if (eventSource.readyState === EventSource.CONNECTING) {
              // Соединение в процессе установки - не показываем ошибку
              return
            }
          } catch {
            // Игнорируем ошибки обработки
          }
        }
      } catch (err) {
        try {
          // Не логируем ошибки создания EventSource - они будут обработаны в onerror
          // Это предотвращает шум в консоли, когда бэкенд не запущен
        } catch {
          // Игнорируем ошибки установки состояния
        }
      }
    }

    const start = async () => {
      const available = await verifyAvailability()
      if (!available || cancelled) {
        return
      }
      connect()
    }

    start()

    // Cleanup
    return () => {
      cancelled = true
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
    }
  }, [isRealTimeEnabled, setProviderMetrics, setMonitoringSystemStats, setError, setRealTimeEnabled])

  return {
    reconnect: () => {
      reconnectAttemptsRef.current = 0
      availabilityCheckedRef.current = false
      availabilityFailedRef.current = false
      setError(null)
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
      }
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current)
        reconnectTimeoutRef.current = null
      }
    },
  }
}

