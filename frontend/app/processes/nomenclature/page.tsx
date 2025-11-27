'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { DatabaseSelector } from '@/components/database-selector'
import { ProjectSelector } from '@/components/project-selector'
import { NormalizationPreviewStats } from '@/components/processes/normalization-preview-stats'
import { NormalizationTypeSelector } from '@/components/processes/normalization-type-selector'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import dynamic from 'next/dynamic'
import { FadeIn } from '@/components/animations/fade-in'
import { motion } from 'framer-motion'
import { PlayCircle, Package, Database, RefreshCw, Download, Clock, Info, Layers } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { NormalizationType } from '@/types/normalization'

// Динамическая загрузка компонента нормализации
const NormalizationProcessTab = dynamic(
  () => import('@/components/processes/normalization-process-tab').then((mod) => ({ default: mod.NormalizationProcessTab })),
  { ssr: false }
)

export default function NomenclatureProcessesPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  
  const dbFromUrl = searchParams.get('database') || ''
  const projectFromUrl = searchParams.get('project') || ''
  const [selectedDatabase, setSelectedDatabase] = useState<string>(dbFromUrl)
  const [selectedProject, setSelectedProject] = useState<string>(projectFromUrl)
  const [mode, setMode] = useState<'database' | 'project'>(projectFromUrl ? 'project' : 'database')
  const [projectType, setProjectType] = useState<string | null>(null)
  const [filteredDatabases, setFilteredDatabases] = useState<string[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [clientId, setClientId] = useState<number | null>(null)
  const [projectId, setProjectId] = useState<number | null>(null)
  const [projectName, setProjectName] = useState<string | null>(null)
  const [projectCode, setProjectCode] = useState<string | null>(null)
  const [normalizationType, setNormalizationType] = useState<NormalizationType>('both')
  const [statsData, setStatsData] = useState<{ nomenclatureCount?: number; counterpartyCount?: number; totalRecords?: number } | null>(null)
  
  // Обновляем clientId и projectId при изменении selectedProject
  useEffect(() => {
    if (selectedProject) {
      const parts = selectedProject.split(':')
      if (parts.length === 2) {
        const cId = parseInt(parts[0], 10)
        const pId = parseInt(parts[1], 10)
        if (!isNaN(cId) && !isNaN(pId)) {
          setClientId(cId)
          setProjectId(pId)
          setMode('project')
          // Очищаем выбор базы данных при выборе проекта
          setSelectedDatabase('')
          
          // Загружаем информацию о проекте
          fetch(`/api/clients/${cId}/projects/${pId}`, { cache: 'no-store' })
            .then(res => res.ok ? res.json() : null)
            .then(data => {
              if (data?.project) {
                setProjectName(data.project.name || null)
                setProjectCode(data.project.code || null)
              }
            })
            .catch(() => {
              setProjectName(null)
              setProjectCode(null)
            })
        } else {
          setClientId(null)
          setProjectId(null)
          setProjectName(null)
          setProjectCode(null)
        }
      } else {
        setClientId(null)
        setProjectId(null)
        setProjectName(null)
        setProjectCode(null)
      }
    } else {
      setClientId(null)
      setProjectId(null)
      setProjectName(null)
      setProjectCode(null)
    }
  }, [selectedProject])
  
  // Очищаем выбор проекта при выборе базы данных
  useEffect(() => {
    if (selectedDatabase) {
      setMode('database')
      setSelectedProject('')
    }
  }, [selectedDatabase])

  // Обновляем состояние при изменении URL параметров
  useEffect(() => {
    const db = searchParams.get('database') || ''
    if (db !== selectedDatabase) {
      setSelectedDatabase(db)
    }
  }, [searchParams])

  // Фильтруем базы данных по типу проекта (nomenclature)
  useEffect(() => {
    const filterDatabases = async () => {
      if (!selectedDatabase) {
        setFilteredDatabases([])
        return
      }

      setIsLoading(true)
      try {
        // Проверяем тип проекта для выбранной базы данных
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут
        
        const response = await fetch(`/api/databases/find-project?file_path=${encodeURIComponent(selectedDatabase)}`, {
          cache: 'no-store',
          signal: controller.signal,
        })
        
        clearTimeout(timeoutId)
        
        if (response.ok) {
          const data = await response.json()
          const pt = data.project_type || ''
          
          // Проверяем, что это проект номенклатуры
          if (pt === 'nomenclature' || pt === 'normalization' || pt === 'nomenclature_counterparties') {
            setProjectType(pt)
            setFilteredDatabases([selectedDatabase])
          } else {
            // Если тип не подходит, очищаем выбор
            setProjectType(null)
            setFilteredDatabases([])
          }
        } else {
          // Если проект не найден, разрешаем использовать базу (для обратной совместимости)
          setProjectType(null)
          setFilteredDatabases([selectedDatabase])
        }
      } catch (error) {
        console.error('Failed to filter databases:', error)
        
        // Обрабатываем разные типы ошибок
        if (error instanceof Error) {
          if (error.name === 'AbortError') {
            console.warn('Request timeout while filtering databases')
          } else if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
            console.warn('Network error while filtering databases')
          }
        }
        
        // В случае ошибки разрешаем использовать базу (для обратной совместимости)
        setProjectType(null)
        setFilteredDatabases([selectedDatabase])
      } finally {
        setIsLoading(false)
      }
    }

    filterDatabases()
  }, [selectedDatabase])

  const handleDatabaseChange = (database: string) => {
    setSelectedDatabase(database)
    // Обновляем URL с новым database
    const params = new URLSearchParams(searchParams.toString())
    if (database) {
      params.set('database', database)
    } else {
      params.delete('database')
    }
    router.push(`/processes/nomenclature?${params.toString()}`, { scroll: false })
  }

  const breadcrumbItems = [
    { label: 'Процессы', href: '/processes', icon: PlayCircle },
    { label: 'Номенклатура', href: '/processes/nomenclature', icon: Package },
  ]

  // Проверяем, подходит ли выбранная база данных для номенклатуры
  const isValidDatabase = !selectedDatabase || filteredDatabases.includes(selectedDatabase) || projectType === null

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>
      
      {/* Шапка процесса */}
      <FadeIn>
        <div className="mb-8">
          <div className="flex flex-col md:flex-row md:items-start md:justify-between gap-4 mb-4">
            <div className="flex-1">
              <motion.h1 
                className="text-2xl md:text-3xl font-bold mb-2 flex items-center gap-2 bg-gradient-to-r from-primary via-primary/80 to-primary bg-clip-text text-transparent"
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
              >
                <Layers className="h-6 w-6 md:h-8 md:w-8 text-primary" />
                Запуск нормализации номенклатуры и контрагентов
              </motion.h1>
              <motion.p 
                className="text-sm md:text-base text-muted-foreground"
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5, delay: 0.1 }}
              >
                Предварительная аналитика и управление процессами нормализации номенклатуры
              </motion.p>
            </div>
            {mode === 'project' && clientId && projectId && (
              <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    // Используем функцию из window, если она доступна
                    const refreshFn = (window as any)[`refreshStats_${clientId}_${projectId}`]
                    if (refreshFn && typeof refreshFn === 'function') {
                      refreshFn()
                    } else {
                      // Fallback: перезагрузка страницы
                      window.location.reload()
                    }
                  }}
                  className="w-full sm:w-auto"
                >
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Обновить данные
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    // Экспорт доступен через dropdown в компоненте статистики
                    // Здесь просто показываем подсказку
                    const exportFn = (window as any)[`exportStats_${clientId}_${projectId}`]
                    if (exportFn && typeof exportFn === 'function') {
                      exportFn('csv')
                    } else {
                      // Показываем сообщение, что экспорт доступен в компоненте
                      alert('Используйте кнопку "Экспорт" в компоненте статистики для экспорта данных')
                    }
                  }}
                  className="w-full sm:w-auto"
                >
                  <Download className="h-4 w-4 mr-2" />
                  Экспорт отчета
                </Button>
              </div>
            )}
          </div>
          
          {/* Индикатор текущего проекта */}
          {mode === 'project' && clientId && projectId && (
            <motion.div
              className="mb-4"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.2 }}
            >
              <Card className="backdrop-blur-md bg-gradient-to-r from-primary/5 via-primary/10 to-primary/5 border-primary/20 shadow-lg bg-card/80">
                <CardContent className="pt-4 pb-4">
                  <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
                    <div className="flex items-center gap-3">
                      <div className="p-2 bg-primary/10 rounded-lg">
                        <Info className="h-5 w-5 text-primary" />
                      </div>
                      <div>
                        <div className="text-xs sm:text-sm font-medium text-muted-foreground">Текущий проект</div>
                        <div className="text-base sm:text-lg font-bold flex flex-wrap items-center gap-2">
                          {projectName || `Проект #${projectId}`}
                          {projectCode && (
                            <Badge variant="outline" className="text-xs">
                              {projectCode}
                            </Badge>
                          )}
                        </div>
                      </div>
                    </div>
                    <div className="text-left sm:text-right">
                      <div className="text-xs text-muted-foreground">Клиент ID</div>
                      <div className="text-sm font-medium">{clientId}</div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          )}
        </div>
      </FadeIn>

      {/* Выбор типа нормализации */}
      {mode === 'project' && clientId && projectId && (
        <FadeIn delay={0.1}>
          <NormalizationTypeSelector
            value={normalizationType}
            onChange={setNormalizationType}
            nomenclatureCount={statsData?.nomenclatureCount}
            counterpartyCount={statsData?.counterpartyCount}
            totalRecords={statsData?.totalRecords}
            className="mb-6"
          />
        </FadeIn>
      )}

      {/* Выбор режима работы */}
      <FadeIn delay={0.15}>
        <Card className="mb-6 backdrop-blur-sm bg-card/95 border-border/50 shadow-md">
          <CardHeader>
            <CardTitle>Выбор источника данных</CardTitle>
            <CardDescription>
              Выберите режим работы: обработка конкретной базы данных или всего проекта
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <Tabs value={mode} onValueChange={(v) => setMode(v as 'database' | 'project')}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="database" className="flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  База данных
                </TabsTrigger>
                <TabsTrigger value="project" className="flex items-center gap-2">
                  <Package className="h-4 w-4" />
                  Проект
                </TabsTrigger>
              </TabsList>
              
              <TabsContent value="database" className="mt-4">
                <DatabaseSelector
                  value={selectedDatabase}
                  onChange={handleDatabaseChange}
                  className="w-full"
                  placeholder="Выберите базу данных"
                />
              </TabsContent>
              
              <TabsContent value="project" className="mt-4">
                <ProjectSelector
                  value={selectedProject}
                  onChange={setSelectedProject}
                  placeholder="Выберите проект"
                  className="w-full"
                />
              </TabsContent>
            </Tabs>
          </CardContent>
        </Card>
      </FadeIn>

      {/* Предупреждение, если база данных не подходит для номенклатуры */}
      {selectedDatabase && !isValidDatabase && !isLoading && (
        <FadeIn delay={0.15}>
          <Alert variant="destructive" className="mb-6">
            <AlertDescription>
              Выбранная база данных не относится к проекту номенклатуры. 
              Пожалуйста, выберите базу данных из проекта типа "nomenclature" или "normalization".
            </AlertDescription>
          </Alert>
        </FadeIn>
      )}

      {/* Секция 3-5: Панель быстрой аналитики, детальная аналитика и таблица БД (только для проекта) */}
      {mode === 'project' && clientId && projectId && (
        <FadeIn delay={0.2}>
          <NormalizationPreviewStats
            clientId={clientId}
            projectId={projectId}
            normalizationType={normalizationType}
            onStatsUpdate={(stats) => {
              setStatsData(stats)
            }}
            onRefresh={() => {
              // Callback для обновления - можно использовать для дополнительных действий
            }}
            onExport={(format) => {
              // Callback для экспорта - можно использовать для дополнительных действий
              console.log(`Export completed: ${format}`)
            }}
          />
        </FadeIn>
      )}

      {/* Секция 6-7: Панель классификации и запуск процесса */}
      {((mode === 'project' && clientId && projectId) || (mode === 'database' && isValidDatabase && selectedDatabase)) && (
        <FadeIn delay={0.3}>
          <div className="space-y-6">
            {mode === 'database' && isValidDatabase && selectedDatabase && (
              <NormalizationProcessTab database={selectedDatabase} />
            )}
            {mode === 'project' && clientId && projectId && (
              <NormalizationProcessTab 
                project={`${clientId}:${projectId}`}
                normalizationType={normalizationType}
              />
            )}
          </div>
        </FadeIn>
      )}

      {mode === 'database' && !selectedDatabase && (
        <div className="text-center py-12 text-muted-foreground">
          <Database className="h-16 w-16 mx-auto mb-4 opacity-50" />
          <p className="text-lg">Выберите базу данных для просмотра процессов нормализации номенклатуры</p>
        </div>
      )}

      {mode === 'project' && (!clientId || !projectId) && (
        <div className="text-center py-12 text-muted-foreground">
          <Package className="h-16 w-16 mx-auto mb-4 opacity-50" />
          <p className="text-lg">Выберите проект для просмотра процессов нормализации номенклатуры</p>
        </div>
      )}
    </div>
  )
}

