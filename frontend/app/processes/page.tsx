'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
// Неиспользуемые импорты удалены
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { DatabaseSelector } from '@/components/database-selector'
import { ProjectSelector } from '@/components/project-selector'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { EmptyState } from '@/components/common/empty-state'
import { Skeleton } from '@/components/ui/skeleton'
import { RefreshCw, PlayCircle } from 'lucide-react'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import dynamic from 'next/dynamic'
import { FadeIn } from '@/components/animations/fade-in'
import { motion, AnimatePresence } from 'framer-motion'

// Динамическая загрузка табов для уменьшения начального bundle
const NormalizationProcessTab = dynamic(
  () => import('@/components/processes/normalization-process-tab').then((mod) => ({ default: mod.NormalizationProcessTab })),
  { ssr: false }
)
const ReclassificationProcessTab = dynamic(
  () => import('@/components/processes/reclassification-process-tab').then((mod) => ({ default: mod.ReclassificationProcessTab })),
  { ssr: false }
)
const PipelineOverview = dynamic(
  () => import('@/components/pipeline/PipelineOverview').then((mod) => ({ default: mod.PipelineOverview })),
  { ssr: false }
)
const PipelineFunnelChart = dynamic(
  () => import('@/components/pipeline/PipelineFunnelChart').then((mod) => ({ default: mod.PipelineFunnelChart })),
  { ssr: false }
)
// Новые компоненты для улучшенной аналитики
const FieldCompletenessAnalytics = dynamic(
  () => import('@/components/processes/field-completeness-analytics').then((mod) => ({ default: mod.FieldCompletenessAnalytics })),
  { ssr: false }
)
const DataStructurePreview = dynamic(
  () => import('@/components/processes/data-structure-preview').then((mod) => ({ default: mod.DataStructurePreview })),
  { ssr: false }
)
const SmartProcessingRecommendations = dynamic(
  () => import('@/components/processes/smart-processing-recommendations').then((mod) => ({ default: mod.SmartProcessingRecommendations })),
  { ssr: false }
)
const InteractiveDataTree = dynamic(
  () => import('@/components/processes/interactive-data-tree').then((mod) => ({ default: mod.InteractiveDataTree })),
  { ssr: false }
)
const DataQualityHeatmap = dynamic(
  () => import('@/components/processes/data-quality-heatmap').then((mod) => ({ default: mod.DataQualityHeatmap })),
  { ssr: false }
)
const NormalizationPreviewStats = dynamic(
  () => import('@/components/processes/normalization-preview-stats').then((mod) => ({ default: mod.NormalizationPreviewStats })),
  { ssr: false }
)

export default function ProcessesPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  
  // Получаем значения из URL параметров
  const tabFromUrl = searchParams.get('tab') || 'normalization'
  const dbFromUrl = searchParams.get('database') || ''
  const projectFromUrl = searchParams.get('project') || '' // Формат: "clientId:projectId"
  
  const [selectedDatabase, setSelectedDatabase] = useState<string>(dbFromUrl)
  const [selectedProject, setSelectedProject] = useState<string>(projectFromUrl)
  const [activeTab, setActiveTab] = useState<string>(tabFromUrl)
  const [pipelineStats, setPipelineStats] = useState<any>(null)
  const [loadingPipeline, setLoadingPipeline] = useState(false)

  // Обновляем состояние при изменении URL параметров (асинхронно)
  useEffect(() => {
    const tab = searchParams.get('tab') || 'normalization'
    const db = searchParams.get('database') || ''
    const project = searchParams.get('project') || ''
    
    // Обновляем состояние только если значения изменились
    if (tab !== activeTab) {
      // Используем requestAnimationFrame для асинхронного обновления
      requestAnimationFrame(() => {
        setActiveTab(tab)
      })
    }
    if (db !== selectedDatabase) {
      requestAnimationFrame(() => {
        setSelectedDatabase(db)
      })
    }
    if (project !== selectedProject) {
      requestAnimationFrame(() => {
        setSelectedProject(project)
      })
    }
  }, [searchParams])

  const [pipelineError, setPipelineError] = useState<string | null>(null)

  // Fetch pipeline stats when pipeline tab is active
  useEffect(() => {
    if (activeTab === 'pipeline') {
      const fetchPipelineStats = async () => {
        setLoadingPipeline(true)
        setPipelineError(null)
        try {
          const response = await fetch('/api/normalization/pipeline/stats', {
            cache: 'no-store'
          })
          if (response.ok) {
            const data = await response.json()
            setPipelineStats(data)
          } else {
            const errorText = await response.text().catch(() => 'Failed to fetch pipeline stats')
            setPipelineError(errorText)
          }
        } catch (error) {
          console.error('Failed to fetch pipeline stats:', error)
          setPipelineError(error instanceof Error ? error.message : 'Не удалось загрузить статистику')
        } finally {
          setLoadingPipeline(false)
        }
      }
      fetchPipelineStats()
    }
  }, [activeTab])

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    // Обновляем URL без перезагрузки страницы
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', value)
    if (selectedDatabase) {
      params.set('database', selectedDatabase)
    }
    if (selectedProject) {
      params.set('project', selectedProject)
    }
    router.push(`/processes?${params.toString()}`, { scroll: false })
  }

  const handleDatabaseChange = (database: string) => {
    setSelectedDatabase(database)
    // Обновляем URL с новым database
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', activeTab)
    if (database) {
      params.set('database', database)
      // При выборе базы данных очищаем выбор проекта
      params.delete('project')
      setSelectedProject('')
    } else {
      params.delete('database')
    }
    if (selectedProject && !database) {
      params.set('project', selectedProject)
    }
    router.push(`/processes?${params.toString()}`, { scroll: false })
  }

  const handleProjectChange = (project: string) => {
    setSelectedProject(project)
    // Обновляем URL с новым project
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', activeTab)
    if (project) {
      params.set('project', project)
      // При выборе проекта очищаем выбор базы данных
      params.delete('database')
      setSelectedDatabase('')
    } else {
      params.delete('project')
    }
    if (selectedDatabase && !project) {
      params.set('database', selectedDatabase)
    }
    router.push(`/processes?${params.toString()}`, { scroll: false })
  }

  const breadcrumbItems = [
    { label: 'Процессы', href: '/processes', icon: PlayCircle },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>
      {/* Header with Database Selector */}
      <FadeIn>
        <div className="mb-8 flex items-center justify-between gap-4">
          <div>
            <motion.h1 
              className="text-3xl font-bold mb-2"
              initial={{ opacity: 0, y: -20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5 }}
            >
              Процессы обработки
            </motion.h1>
            <motion.p 
              className="text-muted-foreground"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.1 }}
            >
              Управление процессами нормализации и переклассификации данных
            </motion.p>
          </div>
          <div className="flex items-center gap-4">
            <ProjectSelector
              value={selectedProject}
              onChange={handleProjectChange}
              placeholder="Выберите проект (все БД)"
              className="flex-shrink-0"
            />
            <DatabaseSelector
              value={selectedDatabase}
              onChange={handleDatabaseChange}
              className="w-[300px]"
              placeholder={selectedProject ? "Или выберите конкретную БД" : "Выберите базу данных"}
            />
          </div>
        </div>
      </FadeIn>

      {/* Tabs Navigation */}
      <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList>
          <TabsTrigger value="normalization">Нормализация</TabsTrigger>
          <TabsTrigger value="reclassification">Переклассификация</TabsTrigger>
          <TabsTrigger value="pipeline">Этапы</TabsTrigger>
        </TabsList>

        <TabsContent value="normalization" className="space-y-6">
          <AnimatePresence mode="wait">
            <motion.div
              key="normalization"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
              <NormalizationProcessTab 
                database={selectedDatabase} 
                project={selectedProject}
              />
            </motion.div>
          </AnimatePresence>
        </TabsContent>

        <TabsContent value="reclassification" className="space-y-6">
          <AnimatePresence mode="wait">
            <motion.div
              key="reclassification"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
              <ReclassificationProcessTab 
                database={selectedDatabase} 
                project={selectedProject}
              />
            </motion.div>
          </AnimatePresence>
        </TabsContent>

        <TabsContent value="pipeline" className="space-y-6">
          <AnimatePresence mode="wait">
            <motion.div
              key="pipeline"
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: -20 }}
              transition={{ duration: 0.3 }}
            >
          {loadingPipeline ? (
            <div className="space-y-6">
              <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                {[...Array(4)].map((_, i) => (
                  <Skeleton key={i} className="h-24" />
                ))}
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5 gap-4">
                {[...Array(15)].map((_, i) => (
                  <Skeleton key={i} className="h-32" />
                ))}
              </div>
            </div>
          ) : pipelineError ? (
            <ErrorState
              title="Ошибка загрузки статистики"
              message={pipelineError}
              action={{
                label: 'Повторить',
                onClick: async () => {
                  setPipelineError(null)
                  setLoadingPipeline(true)
                  try {
                    const response = await fetch('/api/normalization/pipeline/stats', {
                      cache: 'no-store'
                    })
                    if (response.ok) {
                      const data = await response.json()
                      setPipelineStats(data)
                      setPipelineError(null)
                    } else {
                      const errorText = await response.text().catch(() => 'Failed to fetch pipeline stats')
                      setPipelineError(errorText)
                    }
                  } catch (error) {
                    setPipelineError(error instanceof Error ? error.message : 'Не удалось загрузить статистику')
                  } finally {
                    setLoadingPipeline(false)
                  }
                },
              }}
              variant="destructive"
            />
          ) : pipelineStats ? (
            <>
              <PipelineOverview data={pipelineStats} />
              <PipelineFunnelChart data={pipelineStats.stage_stats} />
            </>
          ) : (
            <EmptyState
              icon={RefreshCw}
              title="Нет данных для отображения"
              description="Статистика этапов обработки пока недоступна"
            />
          )}
            </motion.div>
          </AnimatePresence>
        </TabsContent>
      </Tabs>
    </div>
  )
}

