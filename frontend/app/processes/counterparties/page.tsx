'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { DatabaseSelector } from '@/components/database-selector'
import { ProjectSelector } from '@/components/project-selector'
import { NormalizationPreviewStats } from '@/components/processes/normalization-preview-stats'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import dynamic from 'next/dynamic'
import { FadeIn } from '@/components/animations/fade-in'
import { motion } from 'framer-motion'
import { PlayCircle, Building2, Database } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

// Динамическая загрузка компонента нормализации
const NormalizationProcessTab = dynamic(
  () => import('@/components/processes/normalization-process-tab').then((mod) => ({ default: mod.NormalizationProcessTab })),
  { ssr: false }
)

export default function CounterpartiesProcessesPage() {
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
        } else {
          setClientId(null)
          setProjectId(null)
        }
      } else {
        setClientId(null)
        setProjectId(null)
      }
    } else {
      setClientId(null)
      setProjectId(null)
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

  // Фильтруем базы данных по типу проекта (counterparty)
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
          
          // Проверяем, что это проект контрагентов
          if (pt === 'counterparty' || pt === 'nomenclature_counterparties') {
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
    router.push(`/processes/counterparties?${params.toString()}`, { scroll: false })
  }

  const breadcrumbItems = [
    { label: 'Процессы', href: '/processes', icon: PlayCircle },
    { label: 'Контрагенты', href: '/processes/counterparties', icon: Building2 },
  ]

  // Проверяем, подходит ли выбранная база данных для контрагентов
  const isValidDatabase = !selectedDatabase || filteredDatabases.includes(selectedDatabase) || projectType === null

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>
      
      {/* Header */}
      <FadeIn>
        <div className="mb-8">
          <motion.h1 
            className="text-3xl font-bold mb-2 flex items-center gap-2"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <Building2 className="h-8 w-8 text-primary" />
            Процессы нормализации контрагентов
          </motion.h1>
          <motion.p 
            className="text-muted-foreground"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            Управление процессами нормализации контрагентов
          </motion.p>
        </div>
      </FadeIn>

      {/* Выбор режима работы */}
      <FadeIn delay={0.15}>
        <Card className="mb-6">
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
                  <Building2 className="h-4 w-4" />
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

      {/* Предварительная статистика для проекта */}
      {mode === 'project' && clientId && projectId && (
        <FadeIn delay={0.2}>
          <NormalizationPreviewStats
            clientId={clientId}
            projectId={projectId}
          />
        </FadeIn>
      )}

      {/* Предупреждение, если база данных не подходит для контрагентов */}
      {selectedDatabase && !isValidDatabase && !isLoading && (
        <Alert variant="destructive" className="mb-6">
          <AlertDescription>
            Выбранная база данных не относится к проекту контрагентов. 
            Пожалуйста, выберите базу данных из проекта типа "counterparty".
          </AlertDescription>
        </Alert>
      )}

      {/* Основной контент */}
      {mode === 'database' && isValidDatabase && selectedDatabase && (
        <div className="space-y-6">
          <NormalizationProcessTab database={selectedDatabase} />
        </div>
      )}

      {mode === 'project' && clientId && projectId && (
        <div className="space-y-6">
          <NormalizationProcessTab project={`${clientId}:${projectId}`} />
        </div>
      )}

      {mode === 'database' && !selectedDatabase && (
        <div className="text-center py-12 text-muted-foreground">
          <Database className="h-16 w-16 mx-auto mb-4 opacity-50" />
          <p className="text-lg">Выберите базу данных для просмотра процессов нормализации контрагентов</p>
        </div>
      )}

      {mode === 'project' && (!clientId || !projectId) && (
        <div className="text-center py-12 text-muted-foreground">
          <Building2 className="h-16 w-16 mx-auto mb-4 opacity-50" />
          <p className="text-lg">Выберите проект для просмотра процессов нормализации контрагентов</p>
        </div>
      )}
    </div>
  )
}

