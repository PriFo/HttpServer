'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { Badge } from '@/components/ui/badge'
import { Loader2, Wifi, WifiOff } from 'lucide-react'
import { checkBackendHealth } from '@/lib/api-config'
import { cn } from '@/lib/utils'

interface BackendStatusIndicatorProps {
  className?: string
  showLabel?: boolean
  checkInterval?: number // в миллисекундах
}

export function BackendStatusIndicator({ 
  className,
  showLabel = true,
  checkInterval = 30000 // 30 секунд по умолчанию
}: BackendStatusIndicatorProps) {
  const [isConnected, setIsConnected] = useState<boolean | null>(null)
  const [isChecking, setIsChecking] = useState(true)
  const [lastCheck, setLastCheck] = useState<Date | null>(null)
  const retryTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const intervalRef = useRef<NodeJS.Timeout | null>(null)

  const checkStatus = useCallback(async (retryAttempt = 0) => {
    // Проверяем, что мы на клиенте
    if (typeof window === 'undefined') {
      return
    }
    
    setIsChecking(true)
    try {
      const healthy = await checkBackendHealth()
      setIsConnected(healthy)
      setLastCheck(new Date())
      // Если подключение восстановлено, сбрасываем счетчик попыток
      if (healthy && retryAttempt > 0) {
        console.log('[BackendStatus] Connection restored')
      }
    } catch (error) {
      console.error('Backend health check failed:', error)
      setIsConnected(false)
      setLastCheck(new Date())
      
      // Автоматическое переподключение с экспоненциальной задержкой
      if (retryAttempt < 5) {
        const delay = Math.min(1000 * Math.pow(2, retryAttempt), 30000) // Максимум 30 секунд
        console.log(`[BackendStatus] Retrying in ${delay}ms (attempt ${retryAttempt + 1}/5)`)
        
        if (retryTimeoutRef.current) {
          clearTimeout(retryTimeoutRef.current)
        }
        retryTimeoutRef.current = setTimeout(() => {
          checkStatus(retryAttempt + 1)
        }, delay)
      } else {
        console.error('[BackendStatus] Max retry attempts reached')
      }
    } finally {
      setIsChecking(false)
    }
  }, [])

  useEffect(() => {
    // Проверяем, что мы на клиенте
    if (typeof window === 'undefined') {
      return
    }
    
    // Первая проверка сразу
    checkStatus(0)

    // Периодическая проверка
    if (intervalRef.current) {
      clearInterval(intervalRef.current)
    }
    intervalRef.current = setInterval(() => {
      checkStatus(0)
    }, checkInterval)

    // Проверка при возврате фокуса на окно
    const handleFocus = () => {
      checkStatus(0)
    }
    window.addEventListener('focus', handleFocus)

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
      }
      if (retryTimeoutRef.current) {
        clearTimeout(retryTimeoutRef.current)
      }
      window.removeEventListener('focus', handleFocus)
    }
  }, [checkInterval, checkStatus])

  if (isChecking && isConnected === null) {
    return (
      <Badge 
        variant="outline" 
        className={cn("flex items-center gap-1.5", className)}
      >
        <Loader2 className="h-3 w-3 animate-spin" />
        {showLabel && <span className="text-xs">Проверка...</span>}
      </Badge>
    )
  }

  const statusColor = isConnected 
    ? "bg-green-500/10 text-green-600 dark:text-green-400 border-green-500/20"
    : "bg-red-500/10 text-red-600 dark:text-red-400 border-red-500/20"

  const StatusIcon = isConnected ? Wifi : WifiOff

  return (
    <Badge 
      variant="outline" 
      className={cn(
        "flex items-center gap-1.5 cursor-help",
        statusColor,
        className
      )}
      title={
        isConnected 
          ? `Backend подключен${lastCheck ? ` (проверено: ${lastCheck.toLocaleTimeString('ru-RU')})` : ''}`
          : `Backend недоступен${lastCheck ? ` (проверено: ${lastCheck.toLocaleTimeString('ru-RU')})` : ''}`
      }
    >
      <StatusIcon className="h-3 w-3" />
      {showLabel && (
        <span className="text-xs">
          {isConnected ? 'Backend' : 'Нет связи'}
        </span>
      )}
    </Badge>
  )
}

