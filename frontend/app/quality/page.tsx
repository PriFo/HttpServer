'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { ErrorState } from "@/components/common/error-state"
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { motion } from "framer-motion"
import { RefreshCw, Database, CheckCircle2, BarChart3, Target, Award } from "lucide-react"
import { QualityOverviewTab } from "@/components/quality/quality-overview-tab"
import { QualityAnalysisProgress } from "@/components/quality/quality-analysis-progress"
import { QualityHeader } from "@/components/quality/quality-header"
import { QualityAnalysisDialog, AnalysisParams } from "@/components/quality/quality-analysis-dialog"
import { CreateBenchmarkDialog } from "@/components/quality/CreateBenchmarkDialog"
import { ProjectDatabasesList } from "@/components/quality/project-databases-list"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import Link from 'next/link'
import dynamic from 'next/dynamic'
import { FadeIn } from "@/components/animations/fade-in"
import { toast } from 'sonner'
import { useError } from '@/contexts/ErrorContext'
import { fetchJson, getErrorMessage } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { normalizePercentage } from '@/lib/locale'
import { useProjectState } from '@/hooks/useProjectState'

// Dynamically load tabs to reduce initial bundle size
const QualityDuplicatesTab = dynamic(
  () => import('@/components/quality/quality-duplicates-tab').then((mod) => ({ default: mod.QualityDuplicatesTab })),
  { ssr: false, loading: () => <TabSkeleton /> }
)
const QualityViolationsTab = dynamic(
  () => import('@/components/quality/quality-violations-tab').then((mod) => ({ default: mod.QualityViolationsTab })),
  { ssr: false, loading: () => <TabSkeleton /> }
)
const QualitySuggestionsTab = dynamic(
  () => import('@/components/quality/quality-suggestions-tab').then((mod) => ({ default: mod.QualitySuggestionsTab })),
  { ssr: false, loading: () => <TabSkeleton /> }
)
const QualityReportTab = dynamic(
  () => import('@/components/quality/quality-report-tab').then((mod) => ({ default: mod.QualityReportTab })),
  { ssr: false, loading: () => <TabSkeleton /> }
)

const QualityCacheTab = dynamic(
  () => import('@/components/quality/quality-cache-tab').then((mod) => ({ default: mod.QualityCacheTab })),
  { ssr: false, loading: () => <TabSkeleton /> }
)

// Skeleton for tab content loading
function TabSkeleton() {
  return (
    <div className="space-y-4 animate-pulse">
      <div className="h-32 bg-muted rounded-lg w-full" />
      <div className="h-64 bg-muted rounded-lg w-full" />
    </div>
  )
}

interface LevelStat {
  count: number
  avg_quality: number
  percentage: number
}

export interface DatabaseStat {
  database_id: number
  database_name: string
  database_path: string
  stats: QualityStats
  last_activity?: string
  last_upload_at?: string
  last_used_at?: string
}

export interface QualityStats {
  total_items: number
  by_level: {
    [key: string]: LevelStat
  }
  average_quality: number
  benchmark_count: number
  benchmark_percentage: number
  databases?: DatabaseStat[] // Массив баз данных проекта (если выбран проект)
  databases_count?: number // Количество баз данных в проекте
  last_activity?: string
}

const PROJECT_KEY_PREFIX = 'project:'
const DATABASE_KEY_PREFIX = 'database:'
const EMPTY_SELECTION_KEY = '__none__'

const buildProjectKey = (project?: string) =>
  project ? `${PROJECT_KEY_PREFIX}${project}` : EMPTY_SELECTION_KEY

const buildDatabaseKey = (database?: string) =>
  database ? `${DATABASE_KEY_PREFIX}${database}` : EMPTY_SELECTION_KEY

const fetchQualityStatsData = async (
  projectKey: string,
  databaseKey: string,
  signal?: AbortSignal
): Promise<QualityStats | null> => {
  if (projectKey === EMPTY_SELECTION_KEY && databaseKey === EMPTY_SELECTION_KEY) {
    return null
  }

  const params = new URLSearchParams()
  let isProjectSelection = false

  if (projectKey !== EMPTY_SELECTION_KEY && projectKey.startsWith(PROJECT_KEY_PREFIX)) {
    params.set('project', projectKey.slice(PROJECT_KEY_PREFIX.length))
    isProjectSelection = true
  }

  if (databaseKey !== EMPTY_SELECTION_KEY && databaseKey.startsWith(DATABASE_KEY_PREFIX)) {
    params.set('database', databaseKey.slice(DATABASE_KEY_PREFIX.length))
  }

  const timeout = isProjectSelection ? QUALITY_TIMEOUTS.VERY_LONG : QUALITY_TIMEOUTS.FAST
  const cache: RequestCache = isProjectSelection ? 'default' : 'no-store'
  const query = params.toString()
  const url = query ? `/api/quality/stats?${query}` : '/api/quality/stats'

  return fetchJson<QualityStats>(url, {
    timeout,
    cache,
    signal,
  })
}

export default function QualityPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { handleError } = useError()
  
  // State
  const [selectedDatabase, setSelectedDatabase] = useState<string>('')
  const [selectedProject, setSelectedProject] = useState<string>('')
  const [activeTab, setActiveTab] = useState<string>('overview')
  const [showAnalyzeDialog, setShowAnalyzeDialog] = useState(false)
  const [analyzing, setAnalyzing] = useState(false)
  const [showProgress, setShowProgress] = useState(false)
  const [showCreateBenchmarkDialog, setShowCreateBenchmarkDialog] = useState(false)
  const [selectedDuplicateItems, setSelectedDuplicateItems] = useState<Array<{id: string, name: string, code?: string, category?: string}>>([])
  const [benchmarkUploadId, setBenchmarkUploadId] = useState<string>('')

  const projectKey = buildProjectKey(selectedProject)
  const databaseKey = buildDatabaseKey(selectedDatabase)
  const hasSelection = Boolean(selectedDatabase || selectedProject)

  const {
    data: statsData,
    loading: statsLoading,
    error: statsError,
    refetch: refetchStats,
  } = useProjectState<QualityStats | null>(
    fetchQualityStatsData,
    projectKey,
    databaseKey,
    [projectKey, databaseKey],
    {
      enabled: hasSelection,
      keepPreviousData: true,
      refetchInterval: hasSelection ? 30000 : null,
    }
  )

  const stats = statsData ?? null

  // Initialize from URL
  useEffect(() => {
    const tab = searchParams.get('tab') || 'overview'
    const db = searchParams.get('database')
    const project = searchParams.get('project') || ''
    
    setActiveTab(tab)
    if (db) {
      setSelectedDatabase(db)
    } else if (!selectedDatabase) {
      // If no DB in URL and none selected, default to empty string
      // In a real app, we might want to auto-select the last used DB from localStorage
      setSelectedDatabase('')
    }
    if (project) {
      setSelectedProject(project)
    } else if (!selectedProject) {
      setSelectedProject('')
    }
  }, [searchParams, selectedDatabase, selectedProject])

  useEffect(() => {
    if (!statsError || !hasSelection) {
      return
    }

    const contextMessage = selectedProject
      ? `Ошибка загрузки статистики качества для проекта: ${selectedProject}`
      : 'Ошибка загрузки статистики качества'

    handleError(new Error(statsError), contextMessage)

    if (
      selectedProject &&
      (statsError.toLowerCase().includes('timeout') ||
        statsError.toLowerCase().includes('fetch') ||
        statsError.toLowerCase().includes('сеть'))
    ) {
      toast.error('Ошибка загрузки', {
        description: 'Не удалось загрузить статистику проекта. Попробуйте обновить страницу.',
        duration: 5000,
      })
    }
  }, [statsError, hasSelection, selectedProject, handleError])

  // Handlers
  const handleDatabaseChange = async (database: string) => {
    setSelectedDatabase(database)
    
    // Если база данных выбрана, но проект не выбран, пытаемся найти проект для этой базы
    let projectToSet = selectedProject
    if (database && !selectedProject) {
      try {
        const projectData = await fetchJson<{ client_id: number; project_id: number }>(
          `/api/databases/find-project?file_path=${encodeURIComponent(database)}`,
          {
            timeout: 5000,
            cache: 'no-store',
          }
        )
        if (projectData && projectData.client_id && projectData.project_id) {
          projectToSet = `${projectData.client_id}:${projectData.project_id}`
          setSelectedProject(projectToSet)
        }
      } catch (err) {
        // Игнорируем ошибки поиска проекта - это не критично
        console.debug('Could not find project for database:', err)
      }
    }
    
    // Update URL
    const params = new URLSearchParams(searchParams.toString())
    if (database) {
      params.set('database', database)
    } else {
      params.delete('database')
    }
    // Сохраняем проект в URL
    if (projectToSet) {
      params.set('project', projectToSet)
    } else {
      params.delete('project')
    }
    router.push(`/quality?${params.toString()}`, { scroll: false })
  }

  const handleProjectChange = (project: string) => {
    setSelectedProject(project)
    // Update URL
    const params = new URLSearchParams(searchParams.toString())
    if (project) {
      params.set('project', project)
    } else {
      params.delete('project')
    }
    // Сохраняем базу в URL, если она выбрана
    if (selectedDatabase) {
      params.set('database', selectedDatabase)
    }
    router.push(`/quality?${params.toString()}`, { scroll: false })
  }

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', value)
    if (selectedDatabase) {
      params.set('database', selectedDatabase)
    }
    if (selectedProject) {
      params.set('project', selectedProject)
    }
    router.push(`/quality?${params.toString()}`, { scroll: false })
  }

  const handleStartAnalysis = async (params: AnalysisParams) => {
    if (!selectedDatabase && !selectedProject) {
      toast.error('Выберите базу данных или проект', {
        description: 'Для запуска анализа необходимо выбрать базу данных или проект',
      })
      return
    }

    // Если выбран проект, но не выбрана конкретная база, нужно выбрать базу из проекта
    if (selectedProject && !selectedDatabase) {
      toast.error('Выберите базу данных', {
        description: 'Для запуска анализа необходимо выбрать конкретную базу данных из проекта',
      })
      return
    }

    setAnalyzing(true)
    setShowAnalyzeDialog(false)

    try {
      await fetchJson('/api/quality/analyze', {
        method: 'POST',
        timeout: QUALITY_TIMEOUTS.VERY_LONG,
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          database: selectedDatabase,
          table: params.table,
          code_column: params.codeColumn,
          name_column: params.nameColumn,
        }),
      })

      setShowProgress(true)
      setAnalyzing(false)
      toast.success('Анализ качества запущен', {
        description: `Анализ таблицы "${params.table}" начат. Результаты появятся после завершения.`,
      })
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Ошибка подключения к серверу')
      handleError(err, 'Не удалось запустить анализ качества')
      toast.error('Ошибка подключения', {
        description: errorMessage,
      })
      setAnalyzing(false)
    }
  }

  const handleAnalysisComplete = () => {
    setShowProgress(false)
    if (selectedDatabase || selectedProject) {
      refetchStats()
      // Trigger tab refresh via key/state update if needed
      // Currently, tabs fetch their own data based on 'database' prop change or internal logic
      // We might need to force a refresh if they cache data aggressively
    }
  }

  // Determine Content State
  const renderContent = () => {
    if (!selectedDatabase && !selectedProject) {
      return (
        <Card className="border-dashed mt-8">
          <CardContent className="pt-6">
            <EmptyState
              icon={Database}
              title="Выберите базу данных или проект"
              description="Для просмотра статистики качества нормализации необходимо выбрать базу данных или проект в выпадающем списке выше"
            />
          </CardContent>
        </Card>
      )
    }

    // Initial loading for stats
    if (statsLoading && !stats) {
      return <LoadingState message="Загрузка статистики качества..." size="lg" className="mt-12" />
    }

    if (statsError && !stats) {
      return (
        <ErrorState
          title="Ошибка загрузки статистики"
          message={statsError}
          action={{
            label: 'Повторить',
            onClick: () => refetchStats(),
          }}
          variant="destructive"
          className="mt-8"
        />
      )
    }

    // Empty stats state (DB not processed)
    if (stats && stats.total_items === 0) {
      const isProject = !!selectedProject && !selectedDatabase
      return (
        <Card className="border-amber-200 bg-amber-50/50 mt-8">
          <CardContent className="pt-6">
            <div className="flex items-start gap-4">
              <div className="rounded-full bg-amber-100 p-2">
                <RefreshCw className="h-5 w-5 text-amber-600 animate-spin" />
              </div>
              <div className="flex-1">
                <h3 className="font-semibold text-amber-900 mb-1">
                  {isProject ? 'Проект не был обработан' : 'База данных не была обработана'}
                </h3>
                <p className="text-sm text-amber-800 mb-4">
                  {isProject 
                    ? 'По базам данных проекта еще не было обработано элементов. Пожалуйста, запустите нормализацию для баз проекта и ожидайте завершения обработки.'
                    : 'По выбранной базе данных еще не было обработано элементов. Пожалуйста, запустите нормализацию и ожидайте завершения обработки.'
                  }
                </p>
                {!isProject && selectedDatabase && (
                  <Button asChild>
                    <Link href={`/processes?tab=normalization&database=${encodeURIComponent(selectedDatabase)}`}>
                      Запустить нормализацию
                    </Link>
                  </Button>
                )}
                {isProject && selectedProject && (
                  <Button asChild>
                    <Link href={`/processes?tab=normalization&project=${encodeURIComponent(selectedProject)}`}>
                      Запустить нормализацию для проекта
                    </Link>
                  </Button>
                )}
              </div>
            </div>
          </CardContent>
        </Card>
      )
    }

    // Main Content
    return (
      <div className="space-y-6 mt-8">
        {/* Показываем список баз проекта, если выбран проект */}
        {selectedProject && stats?.databases && (
          <ProjectDatabasesList
            databases={stats.databases}
            selectedDatabase={selectedDatabase}
            onDatabaseSelect={(dbPath) => {
              // При выборе базы из списка проекта, обновляем URL
              // Проект остается выбранным
              handleDatabaseChange(dbPath)
            }}
            loading={statsLoading}
          />
        )}
        {/* Показываем сводную статистику проекта, если выбрано только проект без конкретной базы */}
        {selectedProject && !selectedDatabase && stats && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Card className="border-blue-200 bg-blue-50/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-blue-900 flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  Базы данных
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-blue-900">
                  {stats.databases?.length || stats.databases_count || 0}
                </div>
                <p className="text-xs text-blue-700 mt-1">
                  {stats.databases?.length === 1 ? 'база данных' : 'баз данных'}
                </p>
              </CardContent>
            </Card>
            <Card className="border-green-200 bg-green-50/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-green-900 flex items-center gap-2">
                  <BarChart3 className="h-4 w-4" />
                  Всего элементов
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-green-900">
                  {stats.total_items?.toLocaleString() || 0}
                </div>
                <p className="text-xs text-green-700 mt-1">элементов обработано</p>
              </CardContent>
            </Card>
            <Card className="border-purple-200 bg-purple-50/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-purple-900 flex items-center gap-2">
                  <Target className="h-4 w-4" />
                  Среднее качество
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-purple-900">
                  {normalizePercentage(stats.average_quality || 0).toFixed(1)}%
                </div>
                <p className="text-xs text-purple-700 mt-1">качество нормализации</p>
              </CardContent>
            </Card>
            <Card className="border-amber-200 bg-amber-50/50">
              <CardHeader className="pb-3">
                <CardTitle className="text-sm font-medium text-amber-900 flex items-center gap-2">
                  <Award className="h-4 w-4" />
                  Эталонов
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold text-amber-900">
                  {stats.benchmark_count?.toLocaleString() || 0}
                </div>
                <p className="text-xs text-amber-700 mt-1">
                  {normalizePercentage(stats.benchmark_percentage || 0).toFixed(1)}% от общего
                </p>
              </CardContent>
            </Card>
          </div>
        )}
        <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList className="grid w-full grid-cols-2 lg:w-auto lg:inline-flex lg:grid-cols-none">
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="duplicates">Дубликаты</TabsTrigger>
          <TabsTrigger value="violations">Нарушения</TabsTrigger>
          <TabsTrigger value="suggestions">Предложения</TabsTrigger>
          <TabsTrigger value="report">Отчёт</TabsTrigger>
          <TabsTrigger value="cache">Кэш</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6 min-h-[400px]">
          {stats ? (
            <QualityOverviewTab stats={stats} loading={statsLoading} />
          ) : (
            <LoadingState message="Подготовка обзора..." />
          )}
        </TabsContent>

        <TabsContent value="duplicates" className="space-y-6 min-h-[400px]">
          <QualityDuplicatesTab database={selectedDatabase} project={selectedProject} />
        </TabsContent>

        <TabsContent value="violations" className="space-y-6 min-h-[400px]">
          <QualityViolationsTab database={selectedDatabase} project={selectedProject} />
        </TabsContent>

        <TabsContent value="suggestions" className="space-y-6 min-h-[400px]">
          <QualitySuggestionsTab database={selectedDatabase} project={selectedProject} />
        </TabsContent>

        <TabsContent value="report" className="space-y-6 min-h-[400px]">
          <QualityReportTab database={selectedDatabase} project={selectedProject} stats={stats} />
        </TabsContent>

        <TabsContent value="cache" className="space-y-6 min-h-[400px]">
          <QualityCacheTab />
        </TabsContent>
      </Tabs>
      </div>
    )
  }

  const breadcrumbItems = [
    { label: 'Качество', href: '/quality', icon: CheckCircle2 },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="mb-6"
        >
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold flex items-center gap-2">
                <CheckCircle2 className="h-8 w-8 text-primary" />
                Качество данных
              </h1>
              <p className="text-muted-foreground mt-2">
                Анализ качества нормализации и выявление проблем в данных
              </p>
            </div>
            {activeTab === 'duplicates' && stats && stats.total_items > 0 && (
              <Button
                onClick={() => {
                  // Generate a unique ID for the benchmark
                  const uploadId = `benchmark-${Date.now()}`
                  setBenchmarkUploadId(uploadId)
                  setShowCreateBenchmarkDialog(true)
                }}
                className="bg-blue-600 hover:bg-blue-700 text-white"
              >
                Создать эталон
              </Button>
            )}
          </div>
        </motion.div>
      </FadeIn>

      <QualityHeader
        selectedDatabase={selectedDatabase}
        selectedProject={selectedProject}
        onDatabaseChange={handleDatabaseChange}
        onProjectChange={handleProjectChange}
        onRefresh={refetchStats}
        onAnalyze={() => setShowAnalyzeDialog(true)}
        analyzing={analyzing}
        loading={statsLoading}
      />

      {showProgress && (
        <QualityAnalysisProgress onComplete={handleAnalysisComplete} />
      )}

      <QualityAnalysisDialog
        open={showAnalyzeDialog}
        onOpenChange={setShowAnalyzeDialog}
        selectedDatabase={selectedDatabase}
        onStartAnalysis={handleStartAnalysis}
        analyzing={analyzing}
      />

      <CreateBenchmarkDialog
        isOpen={showCreateBenchmarkDialog}
        onClose={() => {
          setShowCreateBenchmarkDialog(false)
          setSelectedDuplicateItems([])
          setBenchmarkUploadId('')
        }}
        uploadId={benchmarkUploadId}
        duplicateItems={selectedDuplicateItems}
        onSuccess={() => {
          // Refresh the current tab content if needed
          if (activeTab === 'duplicates' && (selectedDatabase || selectedProject)) {
            refetchStats()
          }
        }}
      />

      {renderContent()}
    </div>
  )
}
