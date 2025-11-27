'use client'

import React, { useCallback, useMemo, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Play, Square, RefreshCw, Download, Settings, Zap, Loader2 } from 'lucide-react'
import { toast } from 'sonner'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'

interface QuickActionsProps {
  clientId?: string
  projectId?: string
  isRunning?: boolean
  onStart?: () => void
  onStop?: () => void
  onRefresh?: () => void
}

export const QuickActions: React.FC<QuickActionsProps> = ({
  clientId,
  projectId,
  isRunning,
  onStart,
  onStop,
  onRefresh,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const effectiveIsRunning = useMemo(
    () => (typeof isRunning === 'boolean' ? isRunning : identifiers.isProcessRunning),
    [isRunning, identifiers.isProcessRunning]
  )
  if (!effectiveClientId || !effectiveProjectId) {
    return null
  }
  const [isStarting, setIsStarting] = useState(false)
  const [isStopping, setIsStopping] = useState(false)

  const handleStart = useCallback(async () => {
    if (onStart) {
      onStart()
      return
    }

    if (!effectiveClientId || !effectiveProjectId) {
      toast.error('Не указан проект')
      return
    }

    setIsStarting(true)
    try {
      const response = await fetch(`/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/start`, {
        method: 'POST',
      })

      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Неизвестная ошибка')
        throw new Error(`Не удалось запустить нормализацию: ${errorText}`)
      }

      toast.success('Нормализация запущена')
      
      // Вызываем обновление статуса
      const refreshFn = onRefresh || identifiers.refetchStatus
      if (refreshFn) {
        setTimeout(() => refreshFn(), 1000)
      }
    } catch (error: any) {
      console.error('Failed to start normalization:', error)
      toast.error(error.message || 'Ошибка при запуске нормализации')
    } finally {
      setIsStarting(false)
    }
  }, [effectiveClientId, effectiveProjectId, onStart, onRefresh, identifiers.refetchStatus])

  const handleStop = useCallback(async () => {
    if (onStop) {
      onStop()
      return
    }

    if (!effectiveClientId || !effectiveProjectId) {
      toast.error('Не указан проект')
      return
    }

    setIsStopping(true)
    try {
      const response = await fetch(`/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/stop`, {
        method: 'POST',
      })

      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Неизвестная ошибка')
        throw new Error(`Не удалось остановить нормализацию: ${errorText}`)
      }

      toast.success('Нормализация остановлена')
      
      // Вызываем обновление статуса
      const refreshFn = onRefresh || identifiers.refetchStatus
      if (refreshFn) {
        setTimeout(() => refreshFn(), 1000)
      }
    } catch (error: any) {
      console.error('Failed to stop normalization:', error)
      toast.error(error.message || 'Ошибка при остановке нормализации')
    } finally {
      setIsStopping(false)
    }
  }, [effectiveClientId, effectiveProjectId, onStop, onRefresh, identifiers.refetchStatus])

  const handleRefresh = useCallback(() => {
    if (onRefresh) {
      onRefresh()
    } else if (identifiers.refetchStatus) {
      identifiers.refetchStatus()
    } else {
      window.location.reload()
    }
  }, [onRefresh, identifiers])

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Быстрые действия</CardTitle>
        <CardDescription>Управление процессом нормализации</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex flex-wrap gap-2">
          {!effectiveIsRunning ? (
            <Button 
              onClick={handleStart} 
              className="flex-1 min-w-[120px]"
              disabled={isStarting}
            >
              {isStarting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Запуск...
                </>
              ) : (
                <>
                  <Play className="h-4 w-4 mr-2" />
                  Запустить
                </>
              )}
            </Button>
          ) : (
            <Button 
              onClick={handleStop} 
              variant="destructive" 
              className="flex-1 min-w-[120px]"
              disabled={isStopping}
            >
              {isStopping ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Остановка...
                </>
              ) : (
                <>
                  <Square className="h-4 w-4 mr-2" />
                  Остановить
                </>
              )}
            </Button>
          )}
          <Button variant="outline" onClick={handleRefresh}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Обновить
          </Button>
          <Button variant="outline">
            <Settings className="h-4 w-4 mr-2" />
            Настройки
          </Button>
          <Button variant="outline">
            <Zap className="h-4 w-4 mr-2" />
            Быстрая нормализация
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

