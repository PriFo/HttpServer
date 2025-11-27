'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { motion } from "framer-motion"
import { FadeIn } from "@/components/animations/fade-in"
import { toast } from 'sonner'
import { useApiClient } from '@/hooks/useApiClient'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Progress } from "@/components/ui/progress"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Checkbox } from "@/components/ui/checkbox"
import {
  ArrowLeft,
  Play,
  Square,
  BarChart3,
  Target,
  Database,
  AlertCircle
} from "lucide-react"

interface NormalizationStats {
  total_processed: number
  total_groups: number
  benchmark_matches: number
  ai_enhanced: number
  basic_normalized: number
  is_running: boolean
  // Поля для нормализации контрагентов
  processed?: number
  total?: number
  progress?: number
  currentStep?: string
  current_step?: string // Поддержка snake_case из бэкенда
  sessions?: Array<{
    id: number
    project_database_id: number
    database_name: string
    status: string
    created_at: string
    finished_at?: string
  }>
  databases?: Array<{
    id: number
    name: string
    file_path: string
    is_active: boolean
  }>
  active_sessions_count?: number
  total_databases_count?: number
  // Поля для номенклатуры (КПВЭД)
  kpvedClassified?: number
  kpvedTotal?: number
  kpvedProgress?: number
  // Поддержка snake_case из бэкенда
  kpved_classified?: number
  kpved_total?: number
  kpved_progress?: number
}

interface ProjectDatabase {
  id: number
  client_project_id: number
  name: string
  file_path: string
  description: string
  is_active: boolean
  file_size: number
  created_at: string
  updated_at: string
}

interface Project {
  project: {
    id: number
    name: string
    project_type: string
  }
}

export default function ClientNormalizationPage() {
  const params = useParams()
  const router = useRouter()
  const { get, post } = useApiClient()
  const clientId = params.clientId
  const projectId = params.projectId
  const [stats, setStats] = useState<NormalizationStats | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [selectedDatabaseId, setSelectedDatabaseId] = useState('')
  const [databases, setDatabases] = useState<ProjectDatabase[]>([])
  const [error, setError] = useState<string | null>(null)
  const [processingMode, setProcessingMode] = useState<'single' | 'all'>('single')
  const [useKpved, setUseKpved] = useState(false)
  const [project, setProject] = useState<Project | null>(null)
  const [isLoadingProject, setIsLoadingProject] = useState(true)
  
  // Определяем, является ли проект проектом контрагентов
  const isCounterpartyProject = project?.project.project_type === 'counterparty' || 
                                 project?.project.project_type === 'nomenclature_counterparties'
  
  // Определяем, является ли проект проектом номенклатуры
  const isNomenclatureProject = project?.project.project_type === 'nomenclature' || 
                                project?.project.project_type === 'normalization' ||
                                project?.project.project_type === 'nomenclature_counterparties'

  useEffect(() => {
    if (clientId && projectId) {
      fetchProject()
      fetchStats()
      fetchDatabases()
      // Обновляем статус каждые 2 секунды только если процесс запущен
      const interval = setInterval(() => {
        fetchStats()
      }, 2000)
      return () => clearInterval(interval)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clientId, projectId])

  const fetchProject = async () => {
    setIsLoadingProject(true)
    try {
      const data = await get<Project>(`/api/clients/${clientId}/projects/${projectId}`, { skipErrorHandler: true })
      setProject(data)
    } catch (error) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      console.error('Failed to fetch project:', error)
    } finally {
      setIsLoadingProject(false)
    }
  }

  const fetchStats = async () => {
    try {
      const data = await get<NormalizationStats>(`/api/clients/${clientId}/projects/${projectId}/normalization/status`, { skipErrorHandler: true })
      // Преобразуем формат ответа для единообразия
      setStats({
        total_processed: data.total_processed || data.processed || 0,
        total_groups: data.total_groups || 0,
        benchmark_matches: data.benchmark_matches || 0,
        ai_enhanced: data.ai_enhanced || 0,
        basic_normalized: data.basic_normalized || 0,
        is_running: data.is_running || false,
        processed: data.processed || 0,
        total: data.total || 0,
        progress: data.progress || 0,
        currentStep: (data.currentStep ?? data.current_step) || 'Не запущено',
        sessions: data.sessions || [],
        databases: data.databases || [],
        active_sessions_count: data.active_sessions_count || 0,
        total_databases_count: data.total_databases_count || 0,
        // Поля для номенклатуры (КПВЭД)
        kpvedClassified: data.kpvedClassified ?? data.kpved_classified,
        kpvedTotal: data.kpvedTotal ?? data.kpved_total,
        kpvedProgress: data.kpvedProgress ?? data.kpved_progress,
      })
    } catch (error) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      console.error('Failed to fetch normalization stats:', error)
    }
  }

  const fetchDatabases = async () => {
    try {
      const data = await get<{ databases: ProjectDatabase[] }>(`/api/clients/${clientId}/projects/${projectId}/databases?active_only=true`, { skipErrorHandler: true })
      setDatabases(data.databases || [])
    } catch (error) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      console.error('Failed to fetch databases:', error)
    }
  }

  const handleStart = async () => {
    if (processingMode === 'single' && !selectedDatabaseId) {
      setError('Пожалуйста, выберите базу данных')
      return
    }

    if (processingMode === 'single') {
      const selectedDb = databases.find(db => db.id.toString() === selectedDatabaseId)
      if (!selectedDb) {
        setError('Выбранная база данных не найдена')
        return
      }
    }

    setIsLoading(true)
    setError(null)
    try {
      const requestBody: any = {
        use_kpved: useKpved,
      }

      if (processingMode === 'all') {
        requestBody.all_active = true
      } else {
        const selectedDb = databases.find(db => db.id.toString() === selectedDatabaseId)
        if (selectedDb) {
          requestBody.database_path = selectedDb.file_path
        }
      }

      const result = await post<{ message?: string }>(`/api/clients/${clientId}/projects/${projectId}/normalization/start`, requestBody, { skipErrorHandler: true })
      console.log('Normalization started:', result)
      
      // Показываем успешное сообщение, если есть
      if (result.message) {
        toast.success('Нормализация запущена', {
          description: result.message,
        })
      } else {
        toast.success('Нормализация запущена', {
          description: 'Процесс нормализации успешно запущен',
        })
      }

      await fetchStats()
    } catch (error) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      const errorMessage = error instanceof Error ? error.message : 'Ошибка подключения к серверу'
      setError(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  const handleStop = async () => {
    setIsLoading(true)
    try {
      const result = await post<{ message?: string }>(`/api/clients/${clientId}/projects/${projectId}/normalization/stop`, {}, { skipErrorHandler: true })
      
      toast.success('Нормализация остановлена', {
        description: result.message || 'Процесс нормализации успешно остановлен',
      })
      
      // Обновляем статус несколько раз для надежности
      setTimeout(() => fetchStats(), 300)
      setTimeout(() => fetchStats(), 1000)
      setTimeout(() => fetchStats(), 2000)
    } catch (error) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      const errorMessage = error instanceof Error ? error.message : 'Ошибка подключения к серверу'
      setError(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  const breadcrumbItems = [
    { label: 'Клиенты', href: '/clients', icon: Database },
    { label: 'Проекты', href: `/clients/${clientId}/projects`, icon: Database },
    { label: 'Нормализация', href: `#`, icon: Play },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>
      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="flex items-center gap-4"
        >
          <Button 
            variant="outline" 
            size="icon"
            onClick={() => router.push(`/clients/${clientId}/projects/${projectId}`)}
            aria-label="Назад к проекту"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Play className="h-8 w-8 text-primary" />
              {isCounterpartyProject ? 'Нормализация контрагентов' : 'Нормализация для проекта'}
            </h1>
            <p className="text-muted-foreground mt-1">
              {isCounterpartyProject 
                ? 'Параллельная обработка контрагентов с использованием эталонов из базы данных. Внешние источники обогащения не используются.'
                : 'Запуск процесса нормализации с использованием эталонов клиента'}
            </p>
          </div>
        <div className="flex gap-2">
          {stats?.is_running ? (
            <Button onClick={handleStop} variant="destructive">
              <Square className="mr-2 h-4 w-4" />
              Остановить
            </Button>
          ) : (
            <Button 
              onClick={handleStart} 
              disabled={isLoading || (processingMode === 'single' && !selectedDatabaseId)}
            >
              <Play className="mr-2 h-4 w-4" />
              {isLoading ? 'Запуск...' : 'Запустить нормализацию'}
            </Button>
            )}
          </div>
        </motion.div>
      </FadeIn>

      {/* Выбор базы данных */}
      {!stats?.is_running && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              База данных источника
            </CardTitle>
            <CardDescription>
              {isCounterpartyProject 
                ? 'Выберите режим обработки: конкретная база данных или все активные базы проекта. Базы данных будут обработаны параллельно с использованием пула воркеров.'
                : 'Выберите режим обработки: конкретная база данных или все активные базы проекта'}
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            {databases.length === 0 ? (
              <Alert>
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  Нет доступных баз данных. Пожалуйста, добавьте базу данных на странице проекта.
                </AlertDescription>
              </Alert>
            ) : (
              <>
                <div className="space-y-3">
                  <Label>Режим обработки</Label>
                  <RadioGroup value={processingMode} onValueChange={(v) => setProcessingMode(v as 'single' | 'all')}>
                    <div className="flex items-center space-x-2">
                      <RadioGroupItem value="single" id="mode-single" />
                      <Label htmlFor="mode-single" className="cursor-pointer font-normal">
                        Обработать конкретную базу данных
                      </Label>
                    </div>
                    <div className="flex items-center space-x-2">
                      <RadioGroupItem value="all" id="mode-all" />
                      <Label htmlFor="mode-all" className="cursor-pointer font-normal">
                        Обработать все активные базы данных проекта ({databases.length})
                      </Label>
                    </div>
                  </RadioGroup>
                </div>

                {processingMode === 'single' && (
                  <div className="space-y-2">
                    <Label htmlFor="database-select">База данных</Label>
                    <Select value={selectedDatabaseId} onValueChange={setSelectedDatabaseId}>
                      <SelectTrigger id="database-select">
                        <SelectValue placeholder="Выберите базу данных" />
                      </SelectTrigger>
                      <SelectContent>
                        {databases.map((db, index) => (
                          <SelectItem key={`db-select-${db.id}-${db.file_path}-${index}`} value={db.id.toString()}>
                            <div className="flex flex-col">
                              <span className="font-medium">{db.name}</span>
                              <span className="text-xs text-muted-foreground font-mono">{db.file_path}</span>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {selectedDatabaseId && databases.find(db => db.id.toString() === selectedDatabaseId) && (
                      <div className="p-3 bg-muted rounded-md">
                        <p className="text-sm font-medium">
                          {databases.find(db => db.id.toString() === selectedDatabaseId)?.name}
                        </p>
                        <p className="text-xs text-muted-foreground font-mono mt-1">
                          {databases.find(db => db.id.toString() === selectedDatabaseId)?.file_path}
                        </p>
                        {databases.find(db => db.id.toString() === selectedDatabaseId)?.description && (
                          <p className="text-xs text-muted-foreground mt-1">
                            {databases.find(db => db.id.toString() === selectedDatabaseId)?.description}
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                )}

                {processingMode === 'all' && (
                  <div className="p-3 bg-muted rounded-md">
                    <p className="text-sm font-medium mb-2">Будут обработаны следующие базы данных:</p>
                    <ul className="space-y-1">
                      {databases.map((db, index) => (
                        <li key={`db-list-${db.id}-${db.file_path}-${index}`} className="text-sm flex items-center gap-2">
                          <Database className="h-3 w-3" />
                          <span className="font-medium">{db.name}</span>
                          <span className="text-xs text-muted-foreground font-mono">({db.file_path})</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {!isCounterpartyProject && (
                  <div className="flex items-center space-x-2 pt-2 border-t">
                    <Checkbox
                      id="use-kpved"
                      checked={useKpved}
                      onCheckedChange={(checked) => setUseKpved(checked === true)}
                    />
                    <Label htmlFor="use-kpved" className="cursor-pointer font-normal">
                      Классификация по КПВЭД после нормализации
                    </Label>
                  </div>
                )}
                {isCounterpartyProject && (
                  <div className="p-3 bg-blue-50 dark:bg-blue-950 rounded-md border border-blue-200 dark:border-blue-800">
                    <p className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-1">
                      Параллельная обработка контрагентов
                    </p>
                    <p className="text-xs text-blue-700 dark:text-blue-300">
                      Базы данных будут обработаны параллельно с использованием пула воркеров для ускорения процесса нормализации.
                    </p>
                  </div>
                )}
              </>
            )}
            {error && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
          </CardContent>
        </Card>
      )}

      {/* Статистика */}
      {stats && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Обработано</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total_processed}</div>
              <p className="text-xs text-muted-foreground">записей</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Создано групп</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.total_groups}</div>
              <p className="text-xs text-muted-foreground">уникальных групп</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">Совпадений с эталонами</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.benchmark_matches}</div>
              <p className="text-xs text-muted-foreground">использовано эталонов</p>
            </CardContent>
          </Card>
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm font-medium">AI улучшено</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{stats.ai_enhanced}</div>
              <p className="text-xs text-muted-foreground">записей</p>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Статус */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Статус нормализации
          </CardTitle>
        </CardHeader>
        <CardContent>
          {stats?.is_running ? (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-green-500 animate-pulse"></div>
                <span className="font-medium">Нормализация выполняется...</span>
              </div>
              <Progress value={stats.total_processed > 0 ? 50 : 0} className="h-2" />
            </div>
          ) : (
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <div className="h-2 w-2 rounded-full bg-gray-400"></div>
                <span className="font-medium">Нормализация не запущена</span>
              </div>
              <p className="text-sm text-muted-foreground">
                Нажмите &quot;Запустить нормализацию&quot; для начала процесса
              </p>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Информация о процессе */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Target className="h-5 w-5" />
            Процесс нормализации
          </CardTitle>
          <CardDescription>
            {isCounterpartyProject 
              ? 'Нормализация контрагентов использует только эталонные записи из базы данных. Внешние источники обогащения (dadata, adata, gisp) не используются.'
              : 'Нормализация использует эталонные записи клиента для улучшения качества'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">1. Проверка эталонов:</span>
              <Badge variant="outline">{stats?.benchmark_matches || 0} совпадений</Badge>
            </div>
            {!isCounterpartyProject && (
              <>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">2. Базовая нормализация:</span>
                  <Badge variant="outline">{stats?.basic_normalized || 0} записей</Badge>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">3. AI улучшение:</span>
                  <Badge variant="outline">{stats?.ai_enhanced || 0} записей</Badge>
                </div>
              </>
            )}
            {isCounterpartyProject && (
              <div className="flex justify-between text-sm">
                <span className="text-muted-foreground">2. Параллельная обработка БД:</span>
                <Badge variant="outline">Воркеры</Badge>
              </div>
            )}
            <div className="flex justify-between text-sm">
              <span className="text-muted-foreground">{isCounterpartyProject ? '3' : '4'}. Создание новых эталонов:</span>
              <Badge variant="outline">Автоматически</Badge>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

