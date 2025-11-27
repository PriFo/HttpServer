'use client'

import { motion } from 'framer-motion'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Activity, CheckCircle2, AlertCircle, XCircle } from 'lucide-react'
import { useDashboardStore } from '@/stores/dashboard-store'
import { cn } from '@/lib/utils'

export function SystemHealth() {
  const { providerMetrics, monitoringSystemStats, isRealTimeEnabled } = useDashboardStore()

  const activeProviders = providerMetrics?.filter(p => p?.status === 'active').length || 0
  const totalProviders = providerMetrics?.length || 0
  const errorProviders = providerMetrics?.filter(p => p?.status === 'error').length || 0

  const healthStatus = totalProviders === 0
    ? 'healthy' // Если нет провайдеров, считаем систему здоровой
    : errorProviders === 0 
    ? 'healthy' 
    : errorProviders < totalProviders / 2 
    ? 'degraded' 
    : 'critical'

  const statusConfig = {
    healthy: {
      label: 'Здоров',
      color: 'bg-green-500',
      icon: CheckCircle2,
      textColor: 'text-green-600',
    },
    degraded: {
      label: 'Снижена',
      color: 'bg-yellow-500',
      icon: AlertCircle,
      textColor: 'text-yellow-600',
    },
    critical: {
      label: 'Критично',
      color: 'bg-red-500',
      icon: XCircle,
      textColor: 'text-red-600',
    },
  }

  const config = statusConfig[healthStatus]
  const Icon = config.icon

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center justify-between">
          <span>Состояние системы</span>
          <Badge
            variant={healthStatus === 'healthy' ? 'default' : healthStatus === 'degraded' ? 'secondary' : 'destructive'}
            className="flex items-center gap-1"
          >
            <Icon className="h-3 w-3" />
            {config.label}
          </Badge>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Провайдеры</span>
            <span className="font-semibold">
              {activeProviders} / {totalProviders} активны
            </span>
          </div>
          <div className="h-2 bg-secondary rounded-full overflow-hidden">
            <motion.div
              className={cn("h-full", config.color)}
              initial={{ width: 0 }}
              animate={{ width: `${totalProviders > 0 ? (activeProviders / totalProviders) * 100 : 0}%` }}
              transition={{ duration: 0.5 }}
            />
          </div>
        </div>

        {monitoringSystemStats && (
          <div className="pt-2 border-t space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">CPU</span>
              <span className="font-semibold">
                {typeof monitoringSystemStats.cpu_usage === 'number' 
                  ? Math.max(0, Math.min(100, monitoringSystemStats.cpu_usage)).toFixed(1) 
                  : '0.0'}%
              </span>
            </div>
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Память</span>
              <span className="font-semibold">
                {typeof monitoringSystemStats.memory_usage === 'number' 
                  ? Math.max(0, Math.min(100, monitoringSystemStats.memory_usage)).toFixed(1) 
                  : '0.0'}%
              </span>
            </div>
          </div>
        )}

        <div className="pt-2 border-t">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <Activity className="h-3 w-3" />
            <span>Реальное время: {isRealTimeEnabled ? 'Включено' : 'Выключено'}</span>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

