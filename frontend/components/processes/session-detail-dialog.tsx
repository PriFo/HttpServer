'use client'

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'
import { 
  Calendar, 
  Clock, 
  CheckCircle2, 
  XCircle, 
  AlertCircle, 
  Loader2,
  Database,
  TrendingUp,
  FileText,
  RefreshCw
} from 'lucide-react'
import { formatDistanceToNow, format } from 'date-fns'
import { ru } from 'date-fns/locale/ru'

interface SessionDetail {
  id: number
  project_database_id?: number
  database_name?: string
  status: string
  created_at: string
  finished_at?: string | null
  processed_count?: number
  success_count?: number
  error_count?: number
  total_items?: number
  quality_score?: number
  duration_seconds?: number
}

interface SessionDetailDialogProps {
  sessionId: number | null
  open: boolean
  onOpenChange: (open: boolean) => void
  type: 'nomenclature' | 'counterparties'
  clientId?: number | null
  projectId?: number | null
}

export function SessionDetailDialog({
  sessionId,
  open,
  onOpenChange,
  type,
  clientId,
  projectId,
}: SessionDetailDialogProps) {
  const [session, setSession] = useState<SessionDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (open && sessionId) {
      fetchSessionDetail()
    } else {
      setSession(null)
      setError(null)
    }
  }, [open, sessionId])

  const fetchSessionDetail = async () => {
    if (!sessionId) return

    setLoading(true)
    setError(null)

    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут
      
      // Определяем endpoint в зависимости от наличия clientId/projectId
      let endpoint = `/api/normalization/session/${sessionId}`
      if (clientId && projectId) {
        // Используем endpoint для конкретного проекта, если доступен
        endpoint = `/api/clients/${clientId}/projects/${projectId}/normalization/sessions/${sessionId}`
      }
      
      const response = await fetch(endpoint, {
        cache: 'no-store',
        signal: controller.signal,
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        // Если 404 для проекта, пробуем общий endpoint
        if (response.status === 404 && clientId && projectId) {
          const fallbackResponse = await fetch(`/api/normalization/session/${sessionId}`, {
            cache: 'no-store',
            signal: AbortSignal.timeout(10000),
          })
          if (fallbackResponse.ok) {
            const data = await fallbackResponse.json()
            setSession(data)
            return
          }
        }
        
        let errorMessage = 'Не удалось загрузить детали сессии'
        if (response.status === 404) {
          errorMessage = 'Сессия не найдена'
        } else if (response.status === 503 || response.status === 504) {
          errorMessage = 'Сервер временно недоступен. Проверьте подключение к backend серверу на порту 9999'
        } else if (response.status >= 500) {
          errorMessage = `Ошибка сервера: ${response.status}`
        }
        throw new Error(errorMessage)
      }

      const data = await response.json()
      setSession(data)
      setError(null)
    } catch (err) {
      if (err instanceof Error) {
        let errorMessage = err.message
        if (err.name === 'AbortError') {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
          errorMessage = 'Не удалось подключиться к серверу. Проверьте подключение к backend серверу на порту 9999'
        }
        setError(errorMessage)
      } else {
        setError('Неизвестная ошибка при загрузке деталей сессии')
      }
    } finally {
      setLoading(false)
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
      case 'finished':
        return (
          <Badge variant="default" className="bg-green-500">
            <CheckCircle2 className="h-3 w-3 mr-1" />
            Завершено
          </Badge>
        )
      case 'failed':
      case 'error':
        return (
          <Badge variant="destructive">
            <XCircle className="h-3 w-3 mr-1" />
            Ошибка
          </Badge>
        )
      case 'running':
      case 'in_progress':
        return (
          <Badge variant="default" className="bg-blue-500">
            <Loader2 className="h-3 w-3 mr-1 animate-spin" />
            Выполняется
          </Badge>
        )
      case 'stopped':
        return (
          <Badge variant="secondary">
            <AlertCircle className="h-3 w-3 mr-1" />
            Остановлено
          </Badge>
        )
      default:
        return <Badge variant="outline">{status}</Badge>
    }
  }

  const formatDate = (dateString: string) => {
    try {
      const date = new Date(dateString)
      return format(date, 'dd.MM.yyyy HH:mm:ss', { locale: ru })
    } catch {
      return dateString
    }
  }

  const formatDuration = (seconds?: number) => {
    if (!seconds) return '-'
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    
    if (hours > 0) {
      return `${hours}ч ${minutes}м ${secs}с`
    } else if (minutes > 0) {
      return `${minutes}м ${secs}с`
    }
    return `${secs}с`
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh]">
        <DialogHeader>
          <DialogTitle>Детали сессии нормализации</DialogTitle>
          <DialogDescription>
            Подробная информация о сессии #{sessionId}
          </DialogDescription>
        </DialogHeader>

        <ScrollArea className="max-h-[60vh] pr-4">
          {loading ? (
            <div className="space-y-4">
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
              <Skeleton className="h-20 w-full" />
            </div>
          ) : error ? (
            <div className="text-center py-8 text-muted-foreground">
              <AlertCircle className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p className="mb-4">{error}</p>
              <Button
                variant="outline"
                size="sm"
                onClick={fetchSessionDetail}
                disabled={loading}
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
                Повторить
              </Button>
            </div>
          ) : session ? (
            <div className="space-y-6">
              {/* Основная информация */}
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="text-lg font-semibold">Основная информация</h3>
                  {getStatusBadge(session.status)}
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground flex items-center gap-1">
                      <Database className="h-4 w-4" />
                      База данных
                    </div>
                    <div className="font-medium">
                      {session.database_name || `БД #${session.project_database_id || 'N/A'}`}
                    </div>
                  </div>

                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground flex items-center gap-1">
                      <FileText className="h-4 w-4" />
                      ID сессии
                    </div>
                    <div className="font-mono font-medium">#{session.id}</div>
                  </div>
                </div>
              </div>

              <Separator />

              {/* Временные метки */}
              <div className="space-y-4">
                <h3 className="text-lg font-semibold">Временные метки</h3>

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground flex items-center gap-1">
                      <Calendar className="h-4 w-4" />
                      Создано
                    </div>
                    <div className="font-medium">{formatDate(session.created_at)}</div>
                    <div className="text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(session.created_at), { addSuffix: true, locale: ru })}
                    </div>
                  </div>

                  {session.finished_at && (
                    <div className="space-y-1">
                      <div className="text-sm text-muted-foreground flex items-center gap-1">
                        <CheckCircle2 className="h-4 w-4" />
                        Завершено
                      </div>
                      <div className="font-medium">{formatDate(session.finished_at)}</div>
                      <div className="text-xs text-muted-foreground">
                        {formatDistanceToNow(new Date(session.finished_at), { addSuffix: true, locale: ru })}
                      </div>
                    </div>
                  )}

                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground flex items-center gap-1">
                      <Clock className="h-4 w-4" />
                      Длительность
                    </div>
                    <div className="font-medium">
                      {session.duration_seconds
                        ? formatDuration(session.duration_seconds)
                        : session.finished_at
                        ? formatDuration(
                            Math.floor(
                              (new Date(session.finished_at).getTime() -
                                new Date(session.created_at).getTime()) /
                                1000
                            )
                          )
                        : 'В процессе...'}
                    </div>
                  </div>
                </div>
              </div>

              <Separator />

              {/* Статистика */}
              <div className="space-y-4">
                <h3 className="text-lg font-semibold flex items-center gap-2">
                  <TrendingUp className="h-5 w-5" />
                  Статистика обработки
                </h3>

                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground">Всего</div>
                    <div className="text-2xl font-bold">
                      {session.total_items || session.processed_count || 0}
                    </div>
                  </div>

                  {session.success_count !== undefined && (
                    <div className="space-y-1">
                      <div className="text-sm text-muted-foreground">Успешно</div>
                      <div className="text-2xl font-bold text-green-600">
                        {session.success_count}
                      </div>
                    </div>
                  )}

                  {session.error_count !== undefined && (
                    <div className="space-y-1">
                      <div className="text-sm text-muted-foreground">Ошибок</div>
                      <div className="text-2xl font-bold text-red-600">
                        {session.error_count}
                      </div>
                    </div>
                  )}

                  {session.quality_score !== undefined && (
                    <div className="space-y-1">
                      <div className="text-sm text-muted-foreground">Качество</div>
                      <div className="text-2xl font-bold">
                        {(session.quality_score * 100).toFixed(1)}%
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <AlertCircle className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>Сессия не найдена</p>
            </div>
          )}
        </ScrollArea>
      </DialogContent>
    </Dialog>
  )
}

