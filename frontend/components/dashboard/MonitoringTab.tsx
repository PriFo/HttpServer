'use client'

import { memo } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Activity, Zap, Clock, AlertCircle, CheckCircle2 } from 'lucide-react'
import { useDashboardStore } from '@/stores/dashboard-store'
import { useRealTimeData } from '@/hooks/useRealTimeData'
import { EmptyState } from './EmptyState'
import { ProviderStatusBadge } from './ProviderStatusBadge'
import { MetricCard } from './MetricCard'
import { cn } from '@/lib/utils'

const PROVIDER_COLORS: Record<string, string> = {
  openrouter: '#3b82f6',
  huggingface: '#10b981',
  arliai: '#8b5cf6',
  edenai: '#f59e0b',
  dadata: '#ef4444',
  'adata.kz': '#ec4899',
}

function MonitoringTabComponent() {
  const { providerMetrics, monitoringSystemStats, isRealTimeEnabled } = useDashboardStore()
  
  // Подключаемся к реальному времени
  useRealTimeData()

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-500'
      case 'idle':
        return 'bg-gray-400'
      case 'error':
        return 'bg-red-500'
      default:
        return 'bg-gray-400'
    }
  }

  // Status icon function kept for backward compatibility if needed elsewhere
  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'active':
        return CheckCircle2
      case 'idle':
        return Clock
      case 'error':
        return AlertCircle
      default:
        return Clock
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold flex items-center gap-2">
            <Activity className="h-6 w-6" />
            Мониторинг провайдеров
          </h2>
          <p className="text-muted-foreground mt-1">
            Отслеживание производительности в реальном времени
          </p>
        </div>
        <Badge variant={isRealTimeEnabled ? 'default' : 'secondary'}>
          {isRealTimeEnabled ? 'Онлайн' : 'Офлайн'}
        </Badge>
      </div>

      {/* System Stats */}
      {monitoringSystemStats && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="grid grid-cols-1 md:grid-cols-4 gap-4"
        >
          <MetricCard
            title="Всего провайдеров"
            value={typeof monitoringSystemStats.total_providers === 'number' ? monitoringSystemStats.total_providers : 0}
            icon={Activity}
            color="blue"
          />
          <MetricCard
            title="Активных"
            value={typeof monitoringSystemStats.active_providers === 'number' ? monitoringSystemStats.active_providers : 0}
            icon={CheckCircle2}
            color="green"
          />
          <MetricCard
            title="Всего запросов"
            value={typeof monitoringSystemStats.total_requests === 'number' ? monitoringSystemStats.total_requests : 0}
            icon={Zap}
            color="purple"
            format={(v) => (typeof v === 'number' && !isNaN(v) ? v.toLocaleString('ru-RU') : '0')}
          />
          <MetricCard
            title="RPS системы"
            value={typeof monitoringSystemStats.system_requests_per_second === 'number' ? monitoringSystemStats.system_requests_per_second : 0}
            icon={Activity}
            color="yellow"
            format={(v) => (typeof v === 'number' && !isNaN(v) && isFinite(v) ? v.toFixed(2) : '0.00')}
          />
        </motion.div>
      )}

      {/* Provider Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {!Array.isArray(providerMetrics) || providerMetrics.length === 0 ? (
          <div className="col-span-full">
            <EmptyState
              icon={Activity}
              title="Нет данных о провайдерах"
              description="Данные о провайдерах появятся здесь после подключения к системе мониторинга"
            />
          </div>
        ) : (
          providerMetrics.filter(p => p && p.id).map((provider, index) => {
            const isActive = provider?.status === 'active'
            
            return (
              <motion.div
                key={provider.id}
                initial={{ opacity: 0, y: 20, scale: 0.95 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                transition={{ 
                  delay: index * 0.1,
                  type: "spring",
                  stiffness: 300,
                  damping: 20
                }}
                whileHover={{ 
                  scale: 1.03,
                  y: -4,
                  transition: { duration: 0.2 }
                }}
                whileTap={{ scale: 0.98 }}
              >
                <Card className={cn(
                  "relative overflow-hidden",
                  isActive && "border-primary/50 shadow-lg"
                )}>
                  {/* Pulsing indicator for active providers */}
                  {isActive && (
                    <motion.div
                      className="absolute top-2 right-2"
                      animate={{
                        scale: [1, 1.3, 1],
                        opacity: [1, 0.6, 1],
                      }}
                      transition={{
                        duration: 1.5,
                        repeat: Infinity,
                        ease: "easeInOut",
                      }}
                    >
                      <div className={cn(
                        "h-3 w-3 rounded-full shadow-lg",
                        getStatusColor(provider.status)
                      )} />
                      {/* Pulse ring effect */}
                      <motion.div
                        className={cn(
                          "absolute inset-0 h-3 w-3 rounded-full",
                          getStatusColor(provider.status)
                        )}
                        animate={{
                          scale: [1, 2, 2],
                          opacity: [0.6, 0, 0],
                        }}
                        transition={{
                          duration: 1.5,
                          repeat: Infinity,
                          ease: "easeOut",
                        }}
                      />
                    </motion.div>
                  )}

                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle className="text-lg">{provider.name}</CardTitle>
                      <ProviderStatusBadge status={provider.status} />
                    </div>
                    <CardDescription>ID: {provider.id}</CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    {/* RPS */}
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground flex items-center gap-1">
                        <Zap className="h-4 w-4" />
                        RPS
                      </span>
                      <motion.span
                        key={provider.requests_per_second}
                        initial={{ scale: 1.2, color: '#3b82f6' }}
                        animate={{ scale: 1, color: 'inherit' }}
                        className="text-lg font-bold"
                      >
                        {provider.requests_per_second.toFixed(2)}
                      </motion.span>
                    </div>

                    {/* Average Latency */}
                    <div className="flex items-center justify-between">
                      <span className="text-sm text-muted-foreground flex items-center gap-1">
                        <Clock className="h-4 w-4" />
                        Задержка
                      </span>
                      <motion.span
                        key={provider.average_latency_ms}
                        initial={{ scale: 1.2, color: '#3b82f6' }}
                        animate={{ scale: 1, color: 'inherit' }}
                        className="text-lg font-bold"
                      >
                        {provider.average_latency_ms.toFixed(0)}ms
                      </motion.span>
                    </div>

                    {/* Stats */}
                    <div className="grid grid-cols-2 gap-2 pt-2 border-t">
                      <div>
                        <div className="text-xs text-muted-foreground">Всего</div>
                        <div className="text-sm font-semibold">
                          {provider.total_requests.toLocaleString('ru-RU')}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">Успешно</div>
                        <div className="text-sm font-semibold text-green-600">
                          {provider.successful_requests.toLocaleString('ru-RU')}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">Ошибки</div>
                        <div className="text-sm font-semibold text-red-600">
                          {provider.failed_requests.toLocaleString('ru-RU')}
                        </div>
                      </div>
                      <div>
                        <div className="text-xs text-muted-foreground">Каналы</div>
                        <div className="text-sm font-semibold">
                          {provider.active_channels}
                        </div>
                      </div>
                    </div>

                    {/* Success Rate */}
                    {provider.total_requests > 0 && (
                      <div className="pt-2 border-t">
                        <div className="flex items-center justify-between mb-1">
                          <span className="text-xs text-muted-foreground">Успешность</span>
                          <motion.span
                            key={`${provider.id}-success-${provider.successful_requests}`}
                            initial={{ scale: 1.2, color: '#10b981' }}
                            animate={{ scale: 1, color: 'inherit' }}
                            className="text-xs font-semibold"
                          >
                            {((provider.successful_requests / provider.total_requests) * 100).toFixed(1)}%
                          </motion.span>
                        </div>
                        <div className="h-2 bg-secondary rounded-full overflow-hidden">
                          <motion.div
                            className="h-full bg-green-500"
                            initial={{ width: 0 }}
                            animate={{
                              width: `${(provider.successful_requests / provider.total_requests) * 100}%`,
                            }}
                            transition={{ duration: 0.5, ease: 'easeOut' }}
                          />
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </motion.div>
            )
          })
        )}
      </div>

      {/* Charts */}
      {providerMetrics.length > 0 && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {providerMetrics.slice(0, 4).map((provider) => (
              <Card key={provider.id}>
                <CardHeader>
                  <CardTitle className="text-lg">{provider.name}</CardTitle>
                  <CardDescription>RPS: {provider.requests_per_second.toFixed(2)}</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Задержка</span>
                      <span className="font-semibold">{provider.average_latency_ms.toFixed(0)}ms</span>
                    </div>
                    <div className="flex justify-between text-sm">
                      <span className="text-muted-foreground">Успешность</span>
                      <span className="font-semibold text-green-600">
                        {provider.total_requests > 0
                          ? ((provider.successful_requests / provider.total_requests) * 100).toFixed(1)
                          : 0}%
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        </motion.div>
      )}
    </div>
  )
}

export const MonitoringTab = memo(MonitoringTabComponent)

