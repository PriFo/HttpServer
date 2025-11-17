'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { RefreshCw, Database, AlertCircle, Play, Loader2 } from "lucide-react"
import { DatabaseSelector } from "@/components/database-selector"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { QualityOverviewTab } from "@/components/quality/quality-overview-tab"
import { QualityAnalysisProgress } from "@/components/quality/quality-analysis-progress"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import dynamic from 'next/dynamic'

// Динамическая загрузка табов для уменьшения начального bundle
const QualityDuplicatesTab = dynamic(
  () => import('@/components/quality/quality-duplicates-tab').then((mod) => ({ default: mod.QualityDuplicatesTab })),
  { ssr: false }
)
const QualityViolationsTab = dynamic(
  () => import('@/components/quality/quality-violations-tab').then((mod) => ({ default: mod.QualityViolationsTab })),
  { ssr: false }
)
const QualitySuggestionsTab = dynamic(
  () => import('@/components/quality/quality-suggestions-tab').then((mod) => ({ default: mod.QualitySuggestionsTab })),
  { ssr: false }
)

interface LevelStat {
  count: number
  avg_quality: number
  percentage: number
}

interface QualityStats {
  total_items: number
  by_level: {
    [key: string]: LevelStat
  }
  average_quality: number
  benchmark_count: number
  benchmark_percentage: number
}

const LEVEL_COLORS: {[key: string]: string} = {
  'basic': '#94a3b8',
  'ai_enhanced': '#3b82f6',
  'benchmark': '#10b981',
}

const LEVEL_NAMES: {[key: string]: string} = {
  'basic': 'Базовый',
  'ai_enhanced': 'AI улучшенный',
  'benchmark': 'Эталонный',
}

export default function QualityPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const [stats, setStats] = useState<QualityStats | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedDatabase, setSelectedDatabase] = useState<string>('')
  const [activeTab, setActiveTab] = useState<string>('overview')
  const [showAnalyzeDialog, setShowAnalyzeDialog] = useState(false)
  const [analyzeTable, setAnalyzeTable] = useState<string>('normalized_data')
  const [analyzeCodeColumn, setAnalyzeCodeColumn] = useState<string>('')
  const [analyzeNameColumn, setAnalyzeNameColumn] = useState<string>('')
  const [analyzing, setAnalyzing] = useState(false)
  const [showProgress, setShowProgress] = useState(false)

  // Получаем таб из URL или database из query параметров
  useEffect(() => {
    const tab = searchParams.get('tab') || 'overview'
    const db = searchParams.get('database')
    setActiveTab(tab)
    if (db) {
      setSelectedDatabase(db)
    } else if (!selectedDatabase) {
      // Если database не указан в URL и не выбран, пробуем получить из localStorage или использовать пустую строку
      setSelectedDatabase('')
    }
  }, [searchParams, selectedDatabase])

  const fetchStats = async (database: string) => {
    if (!database) {
      setStats(null)
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      setError(null)

      const response = await fetch(`/api/quality/stats?database=${encodeURIComponent(database)}`)
      if (response.ok) {
        const data = await response.json()
        setStats(data)
      } else {
        setError('Не удалось загрузить статистику качества')
      }
    } catch (err) {
      setError('Ошибка подключения к серверу')
      console.error('Error fetching quality stats:', err)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    if (selectedDatabase) {
      fetchStats(selectedDatabase)
      const interval = setInterval(() => fetchStats(selectedDatabase), 30000)
      return () => clearInterval(interval)
    }
  }, [selectedDatabase])

  const handleDatabaseChange = (database: string) => {
    setSelectedDatabase(database)
  }

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    // Обновляем URL без перезагрузки страницы
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', value)
    if (selectedDatabase) {
      params.set('database', selectedDatabase)
    }
    router.push(`/quality?${params.toString()}`, { scroll: false })
  }

  const handleStartAnalysis = async () => {
    if (!selectedDatabase) {
      setError('Выберите базу данных')
      return
    }

    setAnalyzing(true)
    setShowAnalyzeDialog(false)

    try {
      // Определяем колонки по умолчанию если не указаны
      let codeColumn = analyzeCodeColumn
      let nameColumn = analyzeNameColumn

      if (!codeColumn) {
        switch (analyzeTable) {
          case 'normalized_data':
            codeColumn = 'code'
            break
          case 'nomenclature_items':
            codeColumn = 'nomenclature_code'
            break
          case 'catalog_items':
            codeColumn = 'code'
            break
          default:
            codeColumn = 'code'
        }
      }

      if (!nameColumn) {
        switch (analyzeTable) {
          case 'normalized_data':
            nameColumn = 'normalized_name'
            break
          case 'nomenclature_items':
            nameColumn = 'nomenclature_name'
            break
          case 'catalog_items':
            nameColumn = 'name'
            break
          default:
            nameColumn = 'name'
        }
      }

      const response = await fetch('/api/quality/analyze', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          database: selectedDatabase,
          table: analyzeTable,
          code_column: codeColumn,
          name_column: nameColumn,
        }),
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Failed to start analysis' }))
        setError(errorData.error || 'Не удалось запустить анализ')
        setAnalyzing(false)
        return
      }

      setShowProgress(true)
      setAnalyzing(false)
    } catch (err) {
      setError('Ошибка подключения к серверу')
      setAnalyzing(false)
      console.error('Error starting analysis:', err)
    }
  }

  const handleAnalysisComplete = () => {
    setShowProgress(false)
    // Обновляем статистику и данные на странице
    if (selectedDatabase) {
      fetchStats(selectedDatabase)
      // Не перезагружаем страницу, просто обновим данные через небольшую задержку
      // чтобы дать время БД обновиться
      setTimeout(() => {
        // Триггерим обновление через изменение ключа или состояния
        setActiveTab(activeTab) // Это заставит табы перезагрузить данные
      }, 1000)
    }
  }

  const handleTableChange = (table: string) => {
    setAnalyzeTable(table)
    // Автозаполнение колонок
    switch (table) {
      case 'normalized_data':
        setAnalyzeCodeColumn('code')
        setAnalyzeNameColumn('normalized_name')
        break
      case 'nomenclature_items':
        setAnalyzeCodeColumn('nomenclature_code')
        setAnalyzeNameColumn('nomenclature_name')
        break
      case 'catalog_items':
        setAnalyzeCodeColumn('code')
        setAnalyzeNameColumn('name')
        break
      default:
        setAnalyzeCodeColumn('')
        setAnalyzeNameColumn('')
    }
  }

  // Empty state - no database selected
  if (!selectedDatabase) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold mb-2">Качество нормализации</h1>
            <p className="text-muted-foreground">
              Метрики качества обработки данных и управление качеством
            </p>
          </div>
          <DatabaseSelector
            value={selectedDatabase}
            onChange={handleDatabaseChange}
            className="w-[300px]"
          />
        </div>

        <Card className="border-dashed">
          <CardContent className="pt-6">
            <EmptyState
              icon={Database}
              title="Выберите базу данных"
              description="Для просмотра статистики качества нормализации необходимо выбрать базу данных в выпадающем списке выше"
            />
          </CardContent>
        </Card>
      </div>
    )
  }

  // Loading state
  if (loading && !stats) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold mb-2">Качество нормализации</h1>
            <p className="text-muted-foreground">
              Метрики качества обработки данных и управление качеством
            </p>
          </div>
          <DatabaseSelector
            value={selectedDatabase}
            onChange={handleDatabaseChange}
            className="w-[300px]"
          />
        </div>
        <LoadingState message="Загрузка статистики качества..." size="lg" fullScreen />
      </div>
    )
  }

  // Error state
  if (error && !stats) {
    return (
      <div className="container mx-auto px-4 py-8">
        <div className="mb-8 flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold mb-2">Качество нормализации</h1>
            <p className="text-muted-foreground">
              Метрики качества обработки данных и управление качеством
            </p>
          </div>
          <DatabaseSelector
            value={selectedDatabase}
            onChange={handleDatabaseChange}
            className="w-[300px]"
          />
        </div>
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription className="flex items-center justify-between">
            <span>{error || 'Нет данных для отображения'}</span>
            <Button
              variant="outline"
              size="sm"
              onClick={() => fetchStats(selectedDatabase)}
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Повторить
            </Button>
          </AlertDescription>
        </Alert>
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header with Database Selector */}
      <div className="mb-8 flex items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Качество нормализации</h1>
          <p className="text-muted-foreground">
            Метрики качества обработки данных и управление качеством
          </p>
        </div>
        <div className="flex items-center gap-3">
          <DatabaseSelector
            value={selectedDatabase}
            onChange={handleDatabaseChange}
            className="w-[300px]"
          />
          <Button
            variant="outline"
            size="icon"
            onClick={() => fetchStats(selectedDatabase)}
            title="Обновить данные"
          >
            <RefreshCw className="h-4 w-4" />
          </Button>
          <Button
            onClick={() => setShowAnalyzeDialog(true)}
            disabled={!selectedDatabase || analyzing}
            title="Запустить анализ качества"
          >
            {analyzing ? (
              <>
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                Запуск...
              </>
            ) : (
              <>
                <Play className="mr-2 h-4 w-4" />
                Запустить анализ
              </>
            )}
          </Button>
        </div>
      </div>

      {/* Progress Card */}
      {showProgress && (
        <QualityAnalysisProgress onComplete={handleAnalysisComplete} />
      )}

      {/* Analyze Dialog */}
      <Dialog open={showAnalyzeDialog} onOpenChange={setShowAnalyzeDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Запуск анализа качества</DialogTitle>
            <DialogDescription>
              Выберите таблицу для анализа качества данных. Анализ найдет дубликаты, нарушения правил и сгенерирует предложения по улучшению.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="table">Таблица</Label>
              <Select value={analyzeTable} onValueChange={handleTableChange}>
                <SelectTrigger id="table">
                  <SelectValue placeholder="Выберите таблицу" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="normalized_data">normalized_data</SelectItem>
                  <SelectItem value="nomenclature_items">nomenclature_items</SelectItem>
                  <SelectItem value="catalog_items">catalog_items</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="code-column">Колонка с кодом (опционально)</Label>
              <Input
                id="code-column"
                value={analyzeCodeColumn}
                onChange={(e) => setAnalyzeCodeColumn(e.target.value)}
                placeholder="Автозаполнение по таблице"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="name-column">Колонка с названием (опционально)</Label>
              <Input
                id="name-column"
                value={analyzeNameColumn}
                onChange={(e) => setAnalyzeNameColumn(e.target.value)}
                placeholder="Автозаполнение по таблице"
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowAnalyzeDialog(false)}>
              Отмена
            </Button>
            <Button onClick={handleStartAnalysis} disabled={!selectedDatabase || analyzing}>
              {analyzing ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Запуск...
                </>
              ) : (
                <>
                  <Play className="mr-2 h-4 w-4" />
                  Запустить
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Tabs Navigation */}
      <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="duplicates">Дубликаты</TabsTrigger>
          <TabsTrigger value="violations">Нарушения</TabsTrigger>
          <TabsTrigger value="suggestions">Предложения</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {stats && <QualityOverviewTab stats={stats} loading={loading} />}
        </TabsContent>

        <TabsContent value="duplicates" className="space-y-6">
          <QualityDuplicatesTab database={selectedDatabase} />
        </TabsContent>

        <TabsContent value="violations" className="space-y-6">
          <QualityViolationsTab database={selectedDatabase} />
        </TabsContent>

        <TabsContent value="suggestions" className="space-y-6">
          <QualitySuggestionsTab database={selectedDatabase} />
        </TabsContent>
      </Tabs>
    </div>
  )
}
