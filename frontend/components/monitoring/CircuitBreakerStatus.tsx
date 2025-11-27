'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Shield, CheckCircle, XCircle, AlertTriangle } from 'lucide-react'

interface CircuitBreakerState {
  enabled: boolean
  state: string // "closed", "open", "half_open"
  can_proceed: boolean
  failure_count: number
  success_count?: number
  last_failure_time?: string
}

interface CircuitBreakerStatusProps {
  data: CircuitBreakerState
}

export function CircuitBreakerStatus({ data }: CircuitBreakerStatusProps) {
  const getStateColor = (state: string): { variant: "default" | "destructive" | "outline" | "secondary", className?: string } => {
    switch (state) {
      case 'closed':
        return { variant: 'default', className: 'bg-green-500 hover:bg-green-600' }
      case 'open':
        return { variant: 'destructive' }
      case 'half_open':
        return { variant: 'default', className: 'bg-yellow-500 hover:bg-yellow-600' }
      default:
        return { variant: 'secondary' }
    }
  }

  const getStateIcon = (state: string) => {
    switch (state) {
      case 'closed':
        return <CheckCircle className="h-4 w-4" />
      case 'open':
        return <XCircle className="h-4 w-4" />
      case 'half_open':
        return <AlertTriangle className="h-4 w-4" />
      default:
        return <Shield className="h-4 w-4" />
    }
  }

  const getStateLabel = (state: string) => {
    switch (state) {
      case 'closed':
        return 'Закрыт (норма)'
      case 'open':
        return 'Открыт (ошибка)'
      case 'half_open':
        return 'Полуоткрыт (тест)'
      default:
        return 'Неизвестно'
    }
  }

  const getStateDescription = (state: string) => {
    switch (state) {
      case 'closed':
        return 'AI запросы обрабатываются нормально'
      case 'open':
        return 'AI запросы заблокированы из-за множественных ошибок'
      case 'half_open':
        return 'Идет проверка восстановления AI сервиса'
      default:
        return 'Состояние неопределено'
    }
  }

  if (!data.enabled) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Circuit Breaker
          </CardTitle>
          <CardDescription>Защита от перегрузки AI API</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-4">
            <Badge variant="secondary">Отключен</Badge>
            <p className="text-sm text-muted-foreground mt-2">
              Circuit Breaker не активирован
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={data.state === 'open' ? 'border-red-500' : ''}>
      <CardHeader>
        <CardTitle className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Shield className="h-5 w-5" />
            Circuit Breaker
          </div>
          <Badge
            variant={getStateColor(data.state).variant}
            className={`flex items-center gap-1 ${getStateColor(data.state).className || ''}`}
          >
            {getStateIcon(data.state)}
            {getStateLabel(data.state)}
          </Badge>
        </CardTitle>
        <CardDescription>{getStateDescription(data.state)}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-1">
            <p className="text-sm font-medium text-muted-foreground">Счетчик ошибок</p>
            <p className={`text-2xl font-bold ${data.failure_count > 5 ? 'text-red-500' : ''}`}>
              {data.failure_count}
            </p>
          </div>
          {data.success_count !== undefined && (
            <div className="space-y-1">
              <p className="text-sm font-medium text-muted-foreground">Успешных</p>
              <p className="text-2xl font-bold text-green-500">{data.success_count}</p>
            </div>
          )}
        </div>

        {data.last_failure_time && (
          <div className="pt-2 border-t">
            <p className="text-xs text-muted-foreground">
              Последняя ошибка: {new Date(data.last_failure_time).toLocaleString('ru-RU')}
            </p>
          </div>
        )}

        <div className="pt-2">
          <div className="flex items-center justify-between text-sm">
            <span>Разрешить запросы</span>
            <Badge
              variant="default"
              className={data.can_proceed ? 'bg-green-500 hover:bg-green-600' : 'bg-red-500 hover:bg-red-600'}
            >
              {data.can_proceed ? 'ДА' : 'НЕТ'}
            </Badge>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}
