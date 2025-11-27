'use client'

import React, { useState, useMemo, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Bell, CheckCircle2, AlertCircle, Info, X } from 'lucide-react'
import { formatDate } from '@/lib/locale'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { logger } from '@/lib/logger'
import { handleError } from '@/lib/error-handler'
import { fetchNotificationsApi, type NotificationResponse } from '@/lib/mdm/api'

interface Notification {
  id: string
  type: 'success' | 'error' | 'info' | 'warning'
  title: string
  message: string
  timestamp: string
  read: boolean
  action?: {
    label: string
    onClick: () => void
  }
}

interface NotificationCenterProps {
  clientId?: string
  projectId?: string
}

export const NotificationCenter: React.FC<NotificationCenterProps> = ({
  clientId,
  projectId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId

  const { data, loading, error, refetch } = useProjectState<NotificationResponse>(
    (cid, pid, signal) => fetchNotificationsApi(cid, pid, signal),
    effectiveClientId || '',
    effectiveProjectId || '',
    [],
    {
      enabled: !!effectiveClientId && !!effectiveProjectId,
      refetchInterval: 30000, // Обновляем каждые 30 секунд
    }
  )

  const notifications = data?.notifications || []
  
  const unreadCount = useMemo(() => {
    return notifications.filter((n: Notification) => !n.read).length
  }, [notifications])

  const markAsRead = useCallback(async (id: string) => {
    if (!effectiveClientId || !effectiveProjectId) return
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/notifications/${id}/read`,
        { method: 'POST' }
      )
      if (response.ok) {
        await refetch()
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'NotificationCenter',
          action: 'markAsRead',
          notificationId: id,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось отметить уведомление как прочитанное',
        showToast: false, // Не показываем toast для некритичных действий
      })
    }
  }, [effectiveClientId, effectiveProjectId, refetch])

  const markAllAsRead = useCallback(async () => {
    if (!effectiveClientId || !effectiveProjectId) return
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/notifications/read-all`,
        { method: 'POST' }
      )
      if (response.ok) {
        await refetch()
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'NotificationCenter',
          action: 'markAllAsRead',
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось отметить все уведомления как прочитанные',
        showToast: false,
      })
    }
  }, [effectiveClientId, effectiveProjectId, refetch])

  const removeNotification = useCallback(async (id: string) => {
    if (!effectiveClientId || !effectiveProjectId) return
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/notifications/${id}`,
        { method: 'DELETE' }
      )
      if (response.ok) {
        await refetch()
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'NotificationCenter',
          action: 'remove',
          notificationId: id,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось удалить уведомление',
        showToast: false,
      })
    }
  }, [effectiveClientId, effectiveProjectId, refetch])

  const getIcon = (type: string) => {
    switch (type) {
      case 'success':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'error':
        return <AlertCircle className="h-4 w-4 text-red-600" />
      case 'warning':
        return <AlertCircle className="h-4 w-4 text-yellow-600" />
      default:
        return <Info className="h-4 w-4 text-blue-600" />
    }
  }

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Уведомления</CardTitle>
          <CardDescription>Выберите проект, чтобы просмотреть уведомления</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (loading && notifications.length === 0) {
    return <LoadingState message="Загрузка уведомлений..." />
  }

  if (error) {
    return (
      <ErrorState 
        title="Ошибка загрузки уведомлений" 
        message={error} 
        action={{
          label: 'Повторить',
          onClick: refetch,
        }}
      />
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Bell className="h-5 w-5" />
            <CardTitle>Уведомления</CardTitle>
            {unreadCount > 0 && (
              <Badge variant="default">{unreadCount}</Badge>
            )}
          </div>
          {unreadCount > 0 && (
            <Button variant="ghost" size="sm" onClick={markAllAsRead}>
              Отметить все как прочитанные
            </Button>
          )}
        </div>
        <CardDescription>События и обновления процесса нормализации</CardDescription>
      </CardHeader>
      <CardContent>
        {notifications.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            Нет уведомлений
          </div>
        ) : (
          <ScrollArea className="h-[300px]">
            <div className="space-y-2">
              {notifications.map((notification: any) => (
                <div
                  key={notification.id}
                  className={`p-3 border rounded-lg ${
                    !notification.read ? 'bg-primary/5 border-primary/20' : 'bg-muted/30'
                  }`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-start gap-2 flex-1">
                      {getIcon(notification.type)}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <p className="font-medium text-sm">{notification.title}</p>
                          {!notification.read && (
                            <Badge variant="outline" className="text-xs">
                              Новое
                            </Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground">
                          {notification.message}
                        </p>
                        <p className="text-xs text-muted-foreground mt-1">
                          {formatDate(notification.timestamp)}
                        </p>
                        {notification.action && (
                          <Button
                            variant="link"
                            size="sm"
                            className="h-auto p-0 mt-1"
                            onClick={() => {
                              notification.action?.onClick()
                              markAsRead(notification.id)
                            }}
                          >
                            {notification.action.label}
                          </Button>
                        )}
                      </div>
                    </div>
                    <div className="flex items-center gap-1">
                      {!notification.read && (
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => markAsRead(notification.id)}
                          className="h-6 w-6 p-0"
                        >
                          <CheckCircle2 className="h-3 w-3" />
                        </Button>
                      )}
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => removeNotification(notification.id)}
                        className="h-6 w-6 p-0"
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  )
}

