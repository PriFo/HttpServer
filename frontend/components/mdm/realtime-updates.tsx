'use client'

import React, { useState, useCallback, useEffect, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Label } from '@/components/ui/label'
import { Wifi, WifiOff, RefreshCw, Activity } from 'lucide-react'
import { toast } from 'sonner'
import { useProjectState } from '@/hooks/useProjectState'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'

type UpdateInterval = '1s' | '5s' | '10s' | '30s'

interface RealtimeUpdatesProps {
  clientId?: string
  projectId?: string
  onUpdate?: (data: Record<string, unknown>) => void
}

async function fetchStatus(clientId: string, projectId: string, signal?: AbortSignal) {
  const response = await fetch(
    `/api/clients/${clientId}/projects/${projectId}/normalization/status`,
    {
      method: 'GET',
      headers: { 'Content-Type': 'application/json' },
      cache: 'no-store',
      signal,
    }
  )
  if (!response.ok) throw new Error('Failed to fetch status')
  return response.json()
}

export const RealtimeUpdates: React.FC<RealtimeUpdatesProps> = ({
  clientId,
  projectId,
  onUpdate,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const hasIdentifiers = Boolean(effectiveClientId && effectiveProjectId)
  const refreshStatus = identifiers.refetchStatus
  const [isConnected, setIsConnected] = useState(false)
  const [updateInterval, setUpdateInterval] = useState<UpdateInterval>('5s')
  const [adaptiveInterval, setAdaptiveInterval] = useState<number | null>(null)
  const [connectionStats, setConnectionStats] = useState({
    successes: 0,
    failures: 0,
    lastError: null as string | null,
  })

  const getIntervalMs = useCallback((interval: string) => {
    switch (interval) {
      case '1s': return 1000
      case '5s': return 5000
      case '10s': return 10000
      case '30s': return 30000
      default: return 5000
    }
  }, [])

  const baseInterval = useMemo(() => getIntervalMs(updateInterval), [getIntervalMs, updateInterval])
  const effectiveInterval = adaptiveInterval ?? baseInterval

  const fetchStatusWithMetrics = useCallback(
    async (cid: string, pid: string, signal?: AbortSignal) => {
      try {
        const result = await fetchStatus(cid, pid, signal)
        setConnectionStats(prev => ({
          successes: prev.successes + 1,
          failures: prev.failures,
          lastError: null,
        }))
        setAdaptiveInterval(null)
        return result
      } catch (err) {
        setConnectionStats(prev => ({
          successes: prev.successes,
          failures: prev.failures + 1,
          lastError: err instanceof Error ? err.message : 'Ошибка соединения',
        }))
        setAdaptiveInterval(prev => {
          const next = Math.min((prev ?? baseInterval) * 2, 60000)
          return next === baseInterval ? null : next
        })
        throw err
      }
    },
    [baseInterval]
  )

  const { data, loading, error, refetch, lastUpdated } = useProjectState(
    fetchStatusWithMetrics,
    effectiveClientId || '',
    effectiveProjectId || '',
    [],
    {
      refetchInterval: isConnected ? effectiveInterval : null,
      enabled: isConnected && hasIdentifiers,
    }
  )

  useEffect(() => {
    if (data && isConnected && onUpdate) {
      onUpdate(data)
    }
  }, [data, isConnected, onUpdate])

  const handleToggle = useCallback(() => {
    if (!hasIdentifiers) {
      toast.error('Не выбран проект')
      return
    }
    setIsConnected(prev => {
      const next = !prev
      if (!next) {
        setAdaptiveInterval(null)
        setConnectionStats({ successes: 0, failures: 0, lastError: null })
      }
      return next
    })
  }, [hasIdentifiers])

  const handleManualRefresh = useCallback(async () => {
    if (!hasIdentifiers) {
      toast.error('Не выбран проект')
      return
    }
    try {
      await refetch()
      toast.success('Данные обновлены')
      refreshStatus?.()
      setConnectionStats(prev => ({
        successes: prev.successes + 1,
        failures: prev.failures,
        lastError: null,
      }))
      setAdaptiveInterval(null)
    } catch (error) {
      console.error('Failed to refresh:', error)
      toast.error('Ошибка обновления данных')
      setConnectionStats(prev => ({
        successes: prev.successes,
        failures: prev.failures + 1,
        lastError: 'Ошибка обновления данных',
      }))
    }
  }, [refetch, hasIdentifiers, refreshStatus])

  const lastUpdateDisplay = lastUpdated
    ? new Date(lastUpdated).toLocaleTimeString('ru-RU')
    : '—'

  if (!hasIdentifiers) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <WifiOff className="h-5 w-5 text-muted-foreground" />
            Обновления в реальном времени
          </CardTitle>
          <CardDescription>Выберите проект для подключения к обновлениям</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {isConnected ? (
              <Wifi className="h-5 w-5 text-green-600" />
            ) : (
              <WifiOff className="h-5 w-5 text-muted-foreground" />
            )}
            <CardTitle className="text-base">Обновления в реальном времени</CardTitle>
          </div>
          <Badge variant={isConnected ? 'default' : 'secondary'}>
            {isConnected ? 'Активно' : 'Неактивно'}
          </Badge>
        </div>
        <CardDescription>
          Автоматическое обновление данных о процессе нормализации
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {error && (
          <div className="flex items-center justify-between rounded-md bg-destructive/10 px-3 py-2 text-sm text-destructive">
            <span>{typeof error === 'string' ? error : 'Сбой подключения'}</span>
            <Button variant="ghost" size="sm" onClick={handleManualRefresh} className="text-destructive">
              Повторить
            </Button>
          </div>
        )}
        {connectionStats.failures > 0 && (
          <div className="rounded-md border border-border/50 bg-muted/40 p-2 text-xs text-muted-foreground space-y-1">
            <div className="flex items-center justify-between">
              <span>Успехов: {connectionStats.successes}</span>
              <span>Ошибок: {connectionStats.failures}</span>
            </div>
            {connectionStats.lastError && <div>Последняя ошибка: {connectionStats.lastError}</div>}
            {adaptiveInterval && adaptiveInterval !== baseInterval && (
              <div>Интервал увеличен до {Math.round(adaptiveInterval / 1000)} c (автовосстановление)</div>
            )}
          </div>
        )}
        <div className="flex items-center justify-between">
          <Button
            variant={isConnected ? 'destructive' : 'default'}
            size="sm"
            onClick={handleToggle}
          >
            {isConnected ? 'Остановить' : 'Запустить'}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={handleManualRefresh}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Обновить сейчас
          </Button>
        </div>

        {isConnected && (
          <>
            <div className="space-y-2">
              <Label>Интервал обновления</Label>
              <RadioGroup
                value={updateInterval}
                onValueChange={(v: UpdateInterval) => setUpdateInterval(v)}
              >
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="1s" id="1s" />
                  <Label htmlFor="1s" className="cursor-pointer">1 секунда</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="5s" id="5s" />
                  <Label htmlFor="5s" className="cursor-pointer">5 секунд</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="10s" id="10s" />
                  <Label htmlFor="10s" className="cursor-pointer">10 секунд</Label>
                </div>
                <div className="flex items-center space-x-2">
                  <RadioGroupItem value="30s" id="30s" />
                  <Label htmlFor="30s" className="cursor-pointer">30 секунд</Label>
                </div>
              </RadioGroup>
            </div>

            <div className="pt-4 border-t space-y-2 text-sm">
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Последнее обновление:</span>
                <span className="font-medium">{lastUpdateDisplay}</span>
              </div>
              {loading && (
                <div className="flex items-center gap-2 text-sm text-muted-foreground">
                  <Activity className="h-4 w-4 animate-pulse" />
                  Обновление...
                </div>
              )}
            </div>
          </>
        )}
      </CardContent>
    </Card>
  )
}

