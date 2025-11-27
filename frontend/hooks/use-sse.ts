/**
 * Хук для работы с Server-Sent Events (SSE)
 * 
 * Упрощает подключение к SSE потокам с автоматическим переподключением
 * и обработкой ошибок
 */

'use client'

import { useState, useEffect, useRef, useCallback } from 'react'

interface UseSSEOptions {
  autoConnect?: boolean
  onMessage?: (data: any) => void
  onError?: (error: Event) => void
  onOpen?: () => void
  onClose?: () => void
  reconnectInterval?: number
  maxReconnectAttempts?: number
}

/**
 * Хук для работы с SSE
 * 
 * @param url - URL SSE эндпоинта
 * @param options - Опции подключения
 * @returns Объект с состоянием и методами управления
 * 
 * @example
 * ```tsx
 * const { isConnected, error, connect, disconnect } = useSSE(
 *   '/api/stream',
 *   {
 *     onMessage: (data) => console.log(data),
 *     autoConnect: true,
 *   }
 * )
 * ```
 */
export function useSSE(url: string | null, options: UseSSEOptions = {}) {
  const {
    autoConnect = false,
    onMessage,
    onError,
    onOpen,
    onClose,
    reconnectInterval = 3000,
    maxReconnectAttempts = 5,
  } = options

  const [isConnected, setIsConnected] = useState(false)
  const [isConnecting, setIsConnecting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [reconnectAttempts, setReconnectAttempts] = useState(0)

  const eventSourceRef = useRef<EventSource | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const urlRef = useRef(url)

  // Обновляем URL при изменении
  useEffect(() => {
    urlRef.current = url
  }, [url])

  const cleanup = useCallback(() => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
  }, [])

  const connect = useCallback(() => {
    if (!urlRef.current) {
      setError('URL не указан')
      return
    }

    if (eventSourceRef.current) {
      cleanup()
    }

    setIsConnecting(true)
    setError(null)

    try {
      const eventSource = new EventSource(urlRef.current)
      eventSourceRef.current = eventSource

      eventSource.onopen = () => {
        setIsConnected(true)
        setIsConnecting(false)
        setReconnectAttempts(0)
        onOpen?.()
      }

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data)
          onMessage?.(data)
        } catch (err) {
          // Если не JSON, передаем как есть
          onMessage?.(event.data)
        }
      }

      eventSource.onerror = (err) => {
        // Проверяем, не является ли это нормальным закрытием соединения
        const errorMessage = err instanceof Error ? err.message : String(err)
        const isNormalClose = errorMessage.includes('terminated') || 
                              errorMessage.includes('Stream error') ||
                              eventSource.readyState === EventSource.CONNECTING
        
        if (!isNormalClose) {
          setIsConnected(false)
          setIsConnecting(false)
          onError?.(err)

          // Пытаемся переподключиться
          if (reconnectAttempts < maxReconnectAttempts) {
            setReconnectAttempts((prev) => prev + 1)
            reconnectTimeoutRef.current = setTimeout(() => {
              connect()
            }, reconnectInterval)
          } else {
            setError(`Не удалось подключиться после ${maxReconnectAttempts} попыток`)
            cleanup()
          }
        }
      }
    } catch (err) {
      setIsConnecting(false)
      setError(err instanceof Error ? err.message : 'Ошибка подключения')
      onError?.(err as Event)
    }
  }, [onMessage, onError, onOpen, reconnectInterval, maxReconnectAttempts, reconnectAttempts, cleanup])

  const disconnect = useCallback(() => {
    cleanup()
    setIsConnected(false)
    setIsConnecting(false)
    setReconnectAttempts(0)
    setError(null)
    onClose?.()
  }, [cleanup, onClose])

  // Автоподключение
  useEffect(() => {
    if (autoConnect && url) {
      connect()
    }

    return () => {
      cleanup()
    }
  }, [autoConnect, url, connect, cleanup])

  // Очистка при размонтировании
  useEffect(() => {
    return () => {
      cleanup()
    }
  }, [cleanup])

  return {
    isConnected,
    isConnecting,
    error,
    connect,
    disconnect,
    reconnectAttempts,
  }
}

