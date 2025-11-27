'use client'

import { useEffect, useState, useRef } from 'react'
import { motion } from 'framer-motion'
import {
  Database,
  Package,
  TrendingUp,
  CheckCircle2,
  Play,
  BarChart3,
  Zap,
  Activity,
  RefreshCw
} from 'lucide-react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { StatCard } from '@/components/common/stat-card'
import { useDashboardStore } from '@/stores/dashboard-store'
import { apiClientJson } from '@/lib/api-client'
import { Skeleton } from '@/components/ui/skeleton'
import { NormalizationModal } from './NormalizationModal'
import { useRealTimeData } from '@/hooks/useRealTimeData'
import { ConfettiEffect } from './ConfettiEffect'
import { LottieAnimation } from './LottieAnimation'
import { AnimatedNumber } from './AnimatedNumber'
import { QuickActions } from './QuickActions'
import { SystemHealth } from './SystemHealth'
import Link from 'next/link'
import { formatNumber } from '@/lib/locale'

const containerVariants = {
  hidden: { opacity: 0 },
  visible: {
    opacity: 1,
    transition: {
      staggerChildren: 0.1,
    },
  },
}

const itemVariants = {
  hidden: { opacity: 0, y: 20 },
  visible: {
    opacity: 1,
    y: 0,
    transition: {
      duration: 0.3,
    },
  },
}

export function OverviewTab() {
  const {
    systemStats,
    setSystemStats,
    isLoading,
    setLoading,
    monitoringSystemStats,
    setBackendFallback,
  } = useDashboardStore()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [confettiTrigger, setConfettiTrigger] = useState(false)
  const [showSuccessAnimation, setShowSuccessAnimation] = useState(false)
  const prevProcessedRef = useRef(0)
  
  // Подключаемся к реальному времени
  useRealTimeData()

  // Функция загрузки данных - объявлена до использования
  const normalizeFallbackReason = (reason?: string) => {
    if (!reason || reason.trim().length === 0) {
      return 'Данные временно недоступны. Проверьте состояние backend сервиса.'
    }
    const lower = reason.toLowerCase()
    if (lower.includes('body is unusable')) {
      return 'Backend вернул пустой ответ. Перезапустите сервис и проверьте /api/dashboard/stats.'
    }
    if (lower.includes('fetch failed') || lower.includes('failed to fetch')) {
      return 'Не удалось подключиться к backend серверу.'
    }
    return reason
  }

  const loadOverviewData = async () => {
    try {
      setLoading(true)
      const fallbackReasons: string[] = []
      
      const [statsData, qualityData, monitoringData] = await Promise.allSettled([
        apiClientJson<any>('/api/dashboard/stats', { skipErrorHandler: true }),
        apiClientJson<any>('/api/quality/metrics', { skipErrorHandler: true }),
        apiClientJson<any>('/api/monitoring/metrics', { skipErrorHandler: true }),
      ])

      // Обрабатываем основную статистику
      if (statsData.status === 'fulfilled') {
        const stats = statsData.value
        if (stats?.isFallback) {
          fallbackReasons.push(normalizeFallbackReason(stats.fallbackReason || 'Статистика дашборда недоступна.'))
        }
        setSystemStats({
          totalRecords: stats.totalRecords || 0,
          totalDatabases: stats.totalDatabases || 0,
          processedRecords: stats.processedRecords || 0,
          createdGroups: stats.createdGroups || 0,
          mergedRecords: stats.mergedRecords || 0,
          systemVersion: stats.systemVersion || '1.0.0',
          currentDatabase: stats.currentDatabase || null,
          normalizationStatus: stats.normalizationStatus || {
            status: 'idle',
            progress: 0,
            currentStage: 'Ожидание запуска',
            startTime: null,
            endTime: null,
          },
          qualityMetrics: (() => {
            if (qualityData.status === 'fulfilled') {
              const qualityPayload = qualityData.value || {}
              if (qualityPayload?.isFallback) {
                fallbackReasons.push(normalizeFallbackReason(qualityPayload.fallbackReason || 'Метрики качества недоступны.'))
              }
              return {
                overallQuality: qualityPayload.overallQuality || 0,
                highConfidence: qualityPayload.highConfidence || 0,
                mediumConfidence: qualityPayload.mediumConfidence || 0,
                lowConfidence: qualityPayload.lowConfidence || 0,
                totalRecords: qualityPayload.totalRecords || 0,
              }
            }
            if (qualityData.status === 'rejected') {
              fallbackReasons.push('Не удалось загрузить метрики качества.')
            }
            return {
              overallQuality: 0,
              highConfidence: 0,
              mediumConfidence: 0,
              lowConfidence: 0,
              totalRecords: 0,
            }
          })(),
        })
      } else if (statsData.status === 'rejected') {
        // Обрабатываем ошибки загрузки основной статистики
        const error = statsData.reason
        if (error && typeof error === 'object' && 'message' in error) {
          const errorMessage = error.message as string
          // Проверяем, является ли это ошибкой подключения к backend
          if (errorMessage.includes('подключиться к backend') || 
              errorMessage.includes('503') || 
              errorMessage.includes('Service Unavailable')) {
            useDashboardStore.getState().setError(`Не удалось загрузить статистику: ${errorMessage}`)
          } else {
            // Для других ошибок также показываем сообщение
            useDashboardStore.getState().setError(`Не удалось загрузить статистику: ${errorMessage}`)
          }
          fallbackReasons.push(normalizeFallbackReason(errorMessage))
        }
      }

      // Ошибки для qualityData и monitoringData не критичны - они необязательные данные
      // Не логируем их, чтобы не засорять консоль когда бэкенд не запущен
      // Данные будут заменены на fallback значения автоматически

      if (fallbackReasons.length > 0) {
        setBackendFallback({
          isActive: true,
          reasons: Array.from(new Set(fallbackReasons.map(normalizeFallbackReason))),
          timestamp: new Date().toISOString(),
        })
      } else {
        setBackendFallback(null)
      }
    } catch (error) {
      // Используем безопасное логирование ошибок
      try {
        const errorMessage = error instanceof Error ? error.message : String(error)
        useDashboardStore.getState().setError(`Ошибка загрузки данных: ${errorMessage}`)
        setBackendFallback({
          isActive: true,
          reasons: [normalizeFallbackReason(errorMessage)],
          timestamp: new Date().toISOString(),
        })
      } catch {
        // Игнорируем ошибки логирования, чтобы не сломать приложение
      }
    } finally {
      setLoading(false)
    }
  }

  // Проверка milestone для confetti
  useEffect(() => {
    const processed = systemStats?.processedRecords || 0
    const prevProcessed = prevProcessedRef.current

    // Milestone: 100,000 записей
    if (prevProcessed < 100000 && processed >= 100000) {
      setConfettiTrigger(true)
      setShowSuccessAnimation(true)
      setTimeout(() => {
        setConfettiTrigger(false)
        setShowSuccessAnimation(false)
      }, 3000)
      useDashboardStore.getState().addNotification({
        type: 'success',
        title: 'Milestone достигнут!',
        message: 'Обработано 100,000 записей! 🎉',
      })
    }

    // Milestone: высокое качество (>95%)
    const quality = Math.max(0, Math.min(1, systemStats?.qualityMetrics?.overallQuality || 0))
    if (!isNaN(quality) && quality > 0.95 && prevProcessed < processed && processed > 0) {
      setConfettiTrigger(true)
      setTimeout(() => setConfettiTrigger(false), 100)
      useDashboardStore.getState().addNotification({
        type: 'success',
        title: 'Отличное качество!',
        message: `Качество данных превысило 95%! 🚀`,
      })
    }

    prevProcessedRef.current = processed
  }, [systemStats])

  useEffect(() => {
    loadOverviewData()
    
    // Автоматическое обновление каждые 30 секунд
    const interval = setInterval(loadOverviewData, 30000)
    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (isLoading && !systemStats) {
    return (
      <div className="container mx-auto p-6 space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  const stats = systemStats || {
    totalRecords: 0,
    totalDatabases: 0,
    processedRecords: 0,
    createdGroups: 0,
    mergedRecords: 0,
    systemVersion: '1.0.0',
    currentDatabase: null,
    normalizationStatus: {
      status: 'idle' as const,
      progress: 0,
      currentStage: 'Ожидание запуска',
      startTime: null,
      endTime: null,
    },
    qualityMetrics: {
      overallQuality: 0,
      highConfidence: 0,
      mediumConfidence: 0,
      lowConfidence: 0,
      totalRecords: 0,
    },
  }

  const handleRetry = () => {
    loadOverviewData()
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      
      <motion.div
        variants={containerVariants}
        initial="hidden"
        animate="visible"
        className="space-y-6"
      >
        {/* Key Metrics */}
        <motion.div variants={itemVariants} className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          <StatCard
            title="Записей в БД"
            value={<AnimatedNumber value={stats.totalRecords} duration={0.8} />}
            description="Всего записей номенклатуры"
            icon={Database}
            variant="primary"
          />
          <StatCard
            title="Обработано"
            value={<AnimatedNumber value={stats.processedRecords} duration={0.8} />}
            description="Записей обработано"
            icon={CheckCircle2}
            variant="success"
          />
          <StatCard
            title="Создано групп"
            value={<AnimatedNumber value={stats.createdGroups} duration={0.8} />}
            description="Групп нормализации"
            icon={Package}
            variant="default"
          />
          <StatCard
            title="Качество данных"
            value={<AnimatedNumber 
              value={Math.max(0, Math.min(100, (stats.qualityMetrics.overallQuality || 0) * 100))} 
              duration={0.8} 
              decimals={0} 
              suffix="%" 
            />}
            description="Общее качество"
            icon={TrendingUp}
            variant={
              (stats.qualityMetrics.overallQuality || 0) > 0.9 ? 'success' : 
              (stats.qualityMetrics.overallQuality || 0) > 0.7 ? 'warning' : 
              'danger'
            }
            progress={Math.max(0, Math.min(100, (stats.qualityMetrics.overallQuality || 0) * 100))}
          />
        </motion.div>

        {/* Main Action Card and System Health */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
          <motion.div variants={itemVariants} className="lg:col-span-2">
          <Card className="border-2 border-primary/20 bg-gradient-to-br from-primary/5 to-background">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-2xl">
                <Zap className="h-6 w-6 text-primary" />
                Быстрые действия
              </CardTitle>
              <CardDescription>
                Запустите нормализацию данных или перейдите к другим разделам системы
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <motion.div
                whileHover={{ scale: 1.02, y: -2 }}
                whileTap={{ scale: 0.98 }}
                className="relative"
              >
                <Button
                  size="lg"
                  className="w-full h-20 text-lg relative overflow-hidden group"
                  onClick={() => setIsModalOpen(true)}
                  disabled={stats.normalizationStatus.status === 'running'}
                >
                  <motion.div
                    className="absolute inset-0 bg-gradient-to-r from-primary/20 to-primary/0"
                    initial={{ x: '-100%' }}
                    whileHover={{ x: '100%' }}
                    transition={{ duration: 0.6 }}
                  />
                  <Play className="h-6 w-6 mr-2 relative z-10" />
                  <span className="relative z-10">
                    {stats.normalizationStatus.status === 'running' 
                      ? 'Нормализация выполняется...' 
                      : 'Запустить нормализацию'}
                  </span>
                </Button>
              </motion.div>

              <QuickActions />
            </CardContent>
          </Card>
          </motion.div>
          
          <motion.div variants={itemVariants}>
            <SystemHealth />
          </motion.div>
        </div>

        {/* System Status */}
        {stats.normalizationStatus.status === 'running' && (
          <motion.div variants={itemVariants}>
            <Card>
              <CardHeader>
                <CardTitle>Статус нормализации</CardTitle>
                <CardDescription>{stats.normalizationStatus.currentStage}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span>Прогресс</span>
                    <span className="font-semibold">{stats.normalizationStatus.progress.toFixed(1)}%</span>
                  </div>
                  <div className="h-2 bg-secondary rounded-full overflow-hidden">
                    <motion.div
                      className="h-full bg-primary"
                      initial={{ width: 0 }}
                      animate={{ width: `${stats.normalizationStatus.progress}%` }}
                      transition={{ duration: 0.3 }}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          </motion.div>
        )}

        {/* Monitoring Stats */}
        {monitoringSystemStats && (
          <motion.div variants={itemVariants} className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Всего запросов</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{formatNumber(monitoringSystemStats.total_requests)}</div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Успешных</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-600">
                  {formatNumber(monitoringSystemStats.total_successful)}
                </div>
              </CardContent>
            </Card>
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">RPS системы</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">
                  {monitoringSystemStats.system_requests_per_second.toFixed(2)}
                </div>
              </CardContent>
            </Card>
          </motion.div>
        )}
      </motion.div>

      <NormalizationModal open={isModalOpen} onOpenChange={setIsModalOpen} />
      <ConfettiEffect trigger={confettiTrigger} type="milestone" />
      
      {/* Success Animation Overlay */}
      {showSuccessAnimation && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
          onClick={() => setShowSuccessAnimation(false)}
        >
          <motion.div
            initial={{ scale: 0.5, rotate: -180 }}
            animate={{ scale: 1, rotate: 0 }}
            exit={{ scale: 0.5, rotate: 180 }}
            transition={{ 
              type: "spring",
              stiffness: 200,
              damping: 15
            }}
            className="w-64 h-64"
            onClick={(e) => e.stopPropagation()}
          >
            <LottieAnimation
              src="https://assets5.lottiefiles.com/packages/lf20_jcikwtux.json"
              loop={false}
              autoplay={true}
              onComplete={() => setShowSuccessAnimation(false)}
              fallback={
                <div className="flex flex-col items-center justify-center h-full">
                  <motion.div
                    animate={{ scale: [1, 1.2, 1], rotate: [0, 180, 360] }}
                    transition={{ duration: 1, repeat: Infinity }}
                    className="text-6xl"
                  >
                    ✅
                  </motion.div>
                  <motion.p
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    className="mt-4 text-lg font-semibold"
                  >
                    Успех!
                  </motion.p>
                </div>
              }
            />
          </motion.div>
        </motion.div>
      )}
    </div>
  )
}

