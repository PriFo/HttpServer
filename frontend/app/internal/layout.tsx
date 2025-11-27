'use client'

import { usePathname } from 'next/navigation'
import Link from 'next/link'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Activity, BarChart3, Settings } from 'lucide-react'
import { motion } from 'framer-motion'

/**
 * Layout для внутренних роутов (/internal/*)
 * 
 * Проверка доступа к admin роли выполняется в proxy.ts (строки 116-134)
 * Пользователи без прав admin автоматически перенаправляются на главную страницу
 */
export default function InternalLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const pathname = usePathname()
  
  // Определяем активную вкладку на основе пути
  const activeTab = pathname?.includes('/worker-trace') ? 'worker-trace' : 'metrics'

  return (
    <div className="min-h-screen bg-background">
      <div className="container mx-auto py-6 px-4">
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
          className="mb-6"
        >
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Settings className="h-8 w-8 text-primary" />
            Внутренние инструменты
          </h1>
          <p className="text-muted-foreground mt-2">
            Инструменты для разработчиков и администраторов
          </p>
        </motion.div>

        {/* Навигация по внутренним страницам */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
          className="mb-6"
        >
          <Tabs value={activeTab} className="w-full">
            <TabsList className="grid w-full max-w-md grid-cols-2">
              <TabsTrigger value="metrics" asChild>
                <Link href="/internal/metrics" className="flex items-center gap-2">
                  <BarChart3 className="h-4 w-4" />
                  Метрики
                </Link>
              </TabsTrigger>
              <TabsTrigger value="worker-trace" asChild>
                <Link href="/internal/worker-trace" className="flex items-center gap-2">
                  <Activity className="h-4 w-4" />
                  Трассировка
                </Link>
              </TabsTrigger>
            </TabsList>
          </Tabs>
        </motion.div>

        {children}
      </div>
    </div>
  )
}

