'use client'

import { Suspense, lazy, memo } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import { useDashboardStore } from '@/stores/dashboard-store'
import { ErrorDisplay } from './ErrorDisplay'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { Skeleton } from '@/components/ui/skeleton'
import { useAnimationContext } from '@/providers/animation-provider'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { RefreshCw } from 'lucide-react'

// Lazy loading для оптимизации производительности с обработкой ошибок
const OverviewTab = lazy(() => 
  import('./OverviewTab')
    .then(m => ({ default: m.OverviewTab }))
    .catch(() => {
      return { default: () => <ErrorDisplay error="Не удалось загрузить компонент обзора" /> }
    })
)
const MonitoringTabError = memo(() => <ErrorDisplay error="Не удалось загрузить компонент мониторинга" />)
MonitoringTabError.displayName = 'MonitoringTabError'

const MonitoringTab = lazy(() => 
  import('./MonitoringTab')
    .then(m => ({ default: m.MonitoringTab }))
    .catch(() => {
      return { default: MonitoringTabError }
    })
)
const ProcessesTab = lazy(() => 
  import('./ProcessesTab')
    .then(m => ({ default: m.ProcessesTab }))
    .catch(() => {
      return { default: () => <ErrorDisplay error="Не удалось загрузить компонент процессов" /> }
    })
)
const QualityTab = lazy(() => 
  import('./QualityTab')
    .then(m => ({ default: m.QualityTab }))
    .catch(() => {
      return { default: () => <ErrorDisplay error="Не удалось загрузить компонент качества" /> }
    })
)
const ClientsTab = lazy(() => 
  import('./ClientsTab')
    .then(m => ({ default: m.ClientsTab }))
    .catch(() => {
      return { default: () => <ErrorDisplay error="Не удалось загрузить компонент клиентов" /> }
    })
)

const tabComponents = {
  overview: OverviewTab,
  monitoring: MonitoringTab,
  processes: ProcessesTab,
  quality: QualityTab,
  clients: ClientsTab,
}

function TabSkeleton() {
  return (
    <motion.div 
      className="container mx-auto p-6 space-y-6"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.2 }}
    >
      <div className="space-y-4">
        <Skeleton className="h-8 w-64" />
        <Skeleton className="h-4 w-96" />
      </div>
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <motion.div
            key={i}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: i * 0.1 }}
          >
            <Skeleton className="h-32" />
          </motion.div>
        ))}
      </div>
    </motion.div>
  )
}

export function MainContentArea() {
  const { activeTab, error, setError, backendFallback, setBackendFallback } = useDashboardStore()
  const { getAnimationConfig } = useAnimationContext()
  const animationConfig = getAnimationConfig()
  const TabComponent = tabComponents[activeTab]
  const handleFallbackRetry = () => {
    setBackendFallback(null)
    setError(null)
    if (typeof window !== 'undefined') {
      window.location.reload()
    }
  }

  // Проверяем, что компонент существует
  if (!TabComponent) {
    return (
      <div className="flex-1 lg:ml-64 mt-16 lg:mt-0 min-h-[calc(100vh-4rem)] pb-6">
        <ErrorDisplay error={`Неизвестная вкладка: ${activeTab}`} />
      </div>
    )
  }

  return (
    <div className="flex-1 lg:ml-64 mt-16 lg:mt-0 min-h-[calc(100vh-4rem)] pb-6">
      {backendFallback?.isActive && (
        <div className="container mx-auto px-6 pt-6">
          <Alert variant="destructive">
            <AlertTitle>Данные дашборда недоступны</AlertTitle>
            <AlertDescription className="mt-2 space-y-2">
              <div className="text-sm text-muted-foreground space-y-1">
                {backendFallback.reasons.map((reason, idx) => (
                  <div key={idx}>• {reason}</div>
                ))}
              </div>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleFallbackRetry}
                  className="flex items-center gap-2"
                >
                  <RefreshCw className="h-3 w-3" />
                  Повторить попытку
                </Button>
                <span className="text-xs text-muted-foreground">
                  Обновлено: {new Date(backendFallback.timestamp).toLocaleTimeString('ru-RU')}
                </span>
              </div>
            </AlertDescription>
          </Alert>
        </div>
      )}
      {error && <ErrorDisplay error={error} className="container mx-auto px-6 pt-6" onRetry={() => setError(null)} />}
      <AnimatePresence mode="wait">
        <motion.div
          key={activeTab}
          initial={{ opacity: 0, y: 20, scale: 0.98 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          exit={{ opacity: 0, y: -20, scale: 0.98 }}
          transition={{ 
            duration: animationConfig.duration,
            ease: animationConfig.ease as [number, number, number, number],
            type: "spring",
            stiffness: 300,
            damping: 30
          }}
          className="h-full"
        >
          <Suspense fallback={<TabSkeleton />}>
            <ErrorBoundary
              onError={(error) => {
                try {
                  setError(`Ошибка компонента: ${error.message}`)
                } catch {
                  // Игнорируем ошибки установки состояния
                }
              }}
            >
              <TabComponent />
            </ErrorBoundary>
          </Suspense>
        </motion.div>
      </AnimatePresence>
    </div>
  )
}

