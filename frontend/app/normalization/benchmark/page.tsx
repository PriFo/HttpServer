'use client'

import { useState, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { 
  Upload, 
  FileText, 
  BarChart3, 
  TrendingUp, 
  Clock, 
  Zap,
  MemoryStick,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Download,
  RefreshCw,
  List,
  FileJson
} from "lucide-react"
import { toast } from "sonner"
import { apiClientJson, apiClient } from '@/lib/api-client'
import { DynamicLineChart, DynamicLine, DynamicBarChart, DynamicBar, DynamicCell, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend } from '@/lib/recharts-dynamic'
import { FadeIn } from "@/components/animations/fade-in"
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { useEffect } from "react"
import { useApiClient } from '@/hooks/useApiClient'

interface BenchmarkResult {
  stage: string
  record_count: number
  duration_ms: number
  records_per_second: number
  memory_used_mb?: number
  duplicate_groups?: number
  total_duplicates?: number
  processed_count?: number
  benchmark_matches?: number
  enriched_count?: number
  created_benchmarks?: number
  error_count?: number
  stopped?: boolean
}

interface BenchmarkReport {
  timestamp: string
  test_name: string
  record_count: number
  duplicate_rate: number
  workers: number
  results: BenchmarkResult[]
  total_duration_ms: number
  average_speed_records_per_sec: number
  summary: Record<string, any>
}

const COLORS = ['#4CAF50', '#2196F3', '#FF9800', '#F44336', '#9C27B0', '#00BCD4']

interface BottleneckAnalysis {
  stage: string
  duration_ms: number
  percentage: number
  records_per_second: number
  memory_used_mb: number
  recommendations: string[]
  severity: 'critical' | 'high' | 'medium' | 'low'
}

interface ComparisonData {
  baseline: any
  current: any
  comparisons: Array<{
    stage: string
    baseline: BenchmarkResult
    current: BenchmarkResult
    speed_change_percent: number
    duration_change_percent: number
    memory_change_percent: number
    improvement: boolean
  }>
  summary: {
    speed_change: number
    duration_change: number
    improvements: number
    regressions: number
    no_changes: number
  }
}

export default function NormalizationBenchmarkPage() {
  const [report, setReport] = useState<BenchmarkReport | null>(null)
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [benchmarksList, setBenchmarksList] = useState<any[]>([])
  const [loadingList, setLoadingList] = useState(false)
  const [bottlenecks, setBottlenecks] = useState<BottleneckAnalysis[]>([])
  const [loadingAnalysis, setLoadingAnalysis] = useState(false)
  const [comparisonData, setComparisonData] = useState<ComparisonData | null>(null)
  const [loadingComparison, setLoadingComparison] = useState(false)
  const [selectedBaseline, setSelectedBaseline] = useState<string>('')
  const [selectedCurrent, setSelectedCurrent] = useState<string>('')
  const fileInputRef = useRef<HTMLInputElement>(null)


  // Загрузка списка бенчмарков
  const fetchBenchmarksList = async () => {
    setLoadingList(true)
    try {
      const data = await apiClientJson<{ benchmarks: any[] }>('/api/normalization/benchmark?list=true')
      setBenchmarksList(data.benchmarks || [])
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Не удалось загрузить список бенчмарков')
    } finally {
      setLoadingList(false)
    }
  }

  // Загрузка конкретного бенчмарка
  const loadBenchmark = async (id: string) => {
    setLoading(true)
    try {
      const data = await apiClientJson<BenchmarkReport>(`/api/normalization/benchmark?id=${id}`)
      setReport(data)
      toast.success('Бенчмарк загружен')
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Не удалось загрузить бенчмарк')
    } finally {
      setLoading(false)
    }
  }

  const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (!file) return

    if (!file.name.endsWith('.json')) {
      toast.error('Файл должен быть в формате JSON')
      return
    }

    setUploading(true)
    try {
      const formData = new FormData()
      formData.append('file', file)

      // Для FormData используем apiClient напрямую, так как apiClientJson не поддерживает FormData
      const response = await apiClient('/api/normalization/benchmark', {
        method: 'POST',
        body: formData,
        headers: {}, // Не устанавливаем Content-Type для FormData, браузер сделает это сам
      })

      const data = await response.json()
      
      if (data.data) {
        setReport(data.data)
        toast.success('Бенчмарк успешно загружен')
        // Обновляем список после загрузки
        fetchBenchmarksList()
      } else {
        throw new Error('Неверный формат ответа')
      }
    } catch (error) {
      toast.error(error instanceof Error ? error.message : 'Не удалось загрузить бенчмарк')
    } finally {
      setUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const handleDownloadExample = () => {
    const exampleReport: BenchmarkReport = {
      timestamp: new Date().toISOString(),
      test_name: "Example Normalization Benchmark",
      record_count: 1000,
      duplicate_rate: 0.2,
      workers: 10,
      results: [
        {
          stage: "Data Extraction",
          record_count: 1000,
          duration_ms: 500,
          records_per_second: 2000.0,
          memory_used_mb: 50.5,
          processed_count: 1000
        },
        {
          stage: "Duplicate Detection",
          record_count: 1000,
          duration_ms: 1200,
          records_per_second: 833.33,
          memory_used_mb: 120.3,
          duplicate_groups: 50,
          total_duplicates: 200
        },
        {
          stage: "Full Normalization",
          record_count: 1000,
          duration_ms: 5000,
          records_per_second: 200.0,
          memory_used_mb: 350.8,
          processed_count: 1000,
          duplicate_groups: 50,
          total_duplicates: 200,
          benchmark_matches: 150,
          enriched_count: 100,
          created_benchmarks: 50,
          error_count: 0
        }
      ],
      total_duration_ms: 6700,
      average_speed_records_per_sec: 149.25,
      summary: {
        total_stages: 3,
        fastest_stage: "Data Extraction",
        slowest_stage: "Full Normalization"
      }
    }

    const blob = new Blob([JSON.stringify(exampleReport, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `normalization_benchmark_example_${new Date().toISOString().split('T')[0]}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    toast.success('Пример файла загружен')
  }

  const formatDuration = (ms: number) => {
    if (ms < 1000) return `${ms.toFixed(0)} мс`
    if (ms < 60000) return `${(ms / 1000).toFixed(2)} сек`
    return `${(ms / 60000).toFixed(2)} мин`
  }

  const getSeverityColor = (percentage: number) => {
    if (percentage > 50) return 'destructive'
    if (percentage > 30) return 'default'
    if (percentage > 15) return 'secondary'
    return 'outline'
  }

  // Анализ узких мест
  const analyzeBottlenecks = async () => {
    if (!report) return

    setLoadingAnalysis(true)
    try {
      const response = await fetch('/api/normalization/benchmark', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'analyze', report }),
      })

      if (!response.ok) {
        throw new Error('Ошибка анализа')
      }

      const data = await response.json()
      setBottlenecks(data.bottlenecks || [])
      toast.success('Анализ завершен')
    } catch (error: any) {
      console.error('Error analyzing:', error)
      toast.error(error.message || 'Ошибка при анализе')
    } finally {
      setLoadingAnalysis(false)
    }
  }

  // Сравнение бенчмарков
  const compareBenchmarks = async () => {
    if (!selectedBaseline || !selectedCurrent) {
      toast.error('Выберите оба бенчмарка для сравнения')
      return
    }

    setLoadingComparison(true)
    try {
      const response = await fetch('/api/normalization/benchmark', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          action: 'compare',
          baseline_id: selectedBaseline,
          current_id: selectedCurrent,
        }),
      })

      if (!response.ok) {
        throw new Error('Ошибка сравнения')
      }

      const data = await response.json()
      setComparisonData(data)
      toast.success('Сравнение завершено')
    } catch (error: any) {
      console.error('Error comparing:', error)
      toast.error(error.message || 'Ошибка при сравнении')
    } finally {
      setLoadingComparison(false)
    }
  }

  // Загружаем список бенчмарков при монтировании
  useEffect(() => {
    fetchBenchmarksList()
  }, [])

  // Автоматически анализируем при загрузке отчета
  useEffect(() => {
    if (report) {
      analyzeBottlenecks()
    }
  }, [report])

  return (
    <div className="container mx-auto py-6 space-y-6">
      <Breadcrumb
        items={[
          { label: 'Главная', href: '/' },
          { label: 'Нормализация', href: '/normalization' },
          { label: 'Бенчмарк', href: '/normalization/benchmark' }
        ]} />

      <FadeIn>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold">Бенчмарк нормализации</h1>
            <p className="text-muted-foreground mt-2">
              Загрузите результаты бенчмарка нормализации для анализа производительности
            </p>
          </div>
        </div>
      </FadeIn>

      <FadeIn delay={0.1}>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <Card>
            <CardHeader>
              <CardTitle>Загрузка результатов бенчмарка</CardTitle>
              <CardDescription>
                Загрузите JSON файл с результатами бенчмарка, созданный утилитой test_normalization_benchmark.go
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex items-center gap-4">
                <div className="flex-1">
                  <Label htmlFor="file-upload">JSON файл бенчмарка</Label>
                  <Input
                    id="file-upload"
                    type="file"
                    accept=".json"
                    ref={fileInputRef}
                    onChange={handleFileUpload}
                    disabled={uploading}
                    className="mt-2"
                  />
                </div>
                <Button
                  onClick={handleDownloadExample}
                  variant="outline"
                  disabled={uploading}
                >
                  <Download className="mr-2 h-4 w-4" />
                  Пример
                </Button>
              </div>
              {uploading && (
                <Alert>
                  <RefreshCw className="h-4 w-4 animate-spin" />
                  <AlertDescription>Загрузка файла...</AlertDescription>
                </Alert>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Сохраненные бенчмарки</CardTitle>
                  <CardDescription>
                    Выберите ранее загруженный бенчмарк для просмотра
                  </CardDescription>
                </div>
                <Button
                  onClick={fetchBenchmarksList}
                  variant="outline"
                  size="sm"
                  disabled={loadingList}
                >
                  <RefreshCw className={`h-4 w-4 ${loadingList ? 'animate-spin' : ''}`} />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {loadingList ? (
                <div className="flex items-center justify-center py-8">
                  <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : benchmarksList.length === 0 ? (
                <div className="text-center py-8 text-muted-foreground">
                  <FileJson className="h-8 w-8 mx-auto mb-2 opacity-50" />
                  <p className="text-sm">Нет сохраненных бенчмарков</p>
                </div>
              ) : (
                <div className="space-y-2 max-h-64 overflow-y-auto">
                  {benchmarksList.map((benchmark: any) => (
                    <div
                      key={benchmark.id}
                      className="flex items-center justify-between p-3 border rounded-lg hover:bg-accent cursor-pointer transition-colors"
                      onClick={() => loadBenchmark(benchmark.id)}
                    >
                      <div className="flex-1 min-w-0">
                        <p className="font-medium text-sm truncate">{benchmark.test_name || 'Бенчмарк'}</p>
                        <p className="text-xs text-muted-foreground">
                          {new Date(benchmark.timestamp).toLocaleString('ru-RU')}
                        </p>
                        <div className="flex items-center gap-2 mt-1">
                          <Badge variant="outline" className="text-xs">
                            {benchmark.record_count} записей
                          </Badge>
                          <Badge variant="outline" className="text-xs">
                            {benchmark.average_speed?.toFixed(0)}/сек
                          </Badge>
                        </div>
                      </div>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={(e) => {
                          e.stopPropagation()
                          loadBenchmark(benchmark.id)
                        }}
                      >
                        <FileText className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </FadeIn>

      {report && (
        <>
          <FadeIn delay={0.2}>
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription>Записей</CardDescription>
                  <CardTitle className="text-2xl">{report.record_count.toLocaleString()}</CardTitle>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription>Средняя скорость</CardDescription>
                  <CardTitle className="text-2xl">
                    {report.average_speed_records_per_sec.toFixed(2)}/сек
                  </CardTitle>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription>Общее время</CardDescription>
                  <CardTitle className="text-2xl">
                    {formatDuration(report.total_duration_ms)}
                  </CardTitle>
                </CardHeader>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardDescription>Дубликатов</CardDescription>
                  <CardTitle className="text-2xl">
                    {(report.duplicate_rate * 100).toFixed(1)}%
                  </CardTitle>
                </CardHeader>
              </Card>
            </div>
          </FadeIn>

          <FadeIn delay={0.3}>
            <Tabs defaultValue="results" className="space-y-4">
              <TabsList>
                <TabsTrigger value="results">Результаты</TabsTrigger>
                <TabsTrigger value="charts">Графики</TabsTrigger>
                <TabsTrigger value="summary">Сводка</TabsTrigger>
                <TabsTrigger value="analysis">Анализ</TabsTrigger>
                <TabsTrigger value="compare">Сравнение</TabsTrigger>
              </TabsList>

              <TabsContent value="results" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle>Результаты по этапам</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <Table>
                      <TableHeader>
                        <TableRow>
                          <TableHead>Этап</TableHead>
                          <TableHead>Записей</TableHead>
                          <TableHead>Время</TableHead>
                          <TableHead>Скорость</TableHead>
                          <TableHead>Память</TableHead>
                          <TableHead>Дубликаты</TableHead>
                          <TableHead>Обработано</TableHead>
                          <TableHead>Статус</TableHead>
                        </TableRow>
                      </TableHeader>
                      <TableBody>
                        {report.results.map((result, index) => {
                          const percentage = (result.duration_ms / report.total_duration_ms) * 100
                          return (
                            <TableRow key={index}>
                              <TableCell className="font-medium">{result.stage}</TableCell>
                              <TableCell>{result.record_count.toLocaleString()}</TableCell>
                              <TableCell>{formatDuration(result.duration_ms)}</TableCell>
                              <TableCell>
                                <div className="flex items-center gap-2">
                                  <Zap className="h-4 w-4 text-yellow-500" />
                                  {result.records_per_second.toFixed(2)}/сек
                                </div>
                              </TableCell>
                              <TableCell>
                                {result.memory_used_mb ? (
                                  <div className="flex items-center gap-2">
                                    <MemoryStick className="h-4 w-4 text-blue-500" />
                                    {result.memory_used_mb.toFixed(2)} МБ
                                  </div>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {result.duplicate_groups ? (
                                  <Badge variant="outline">
                                    {result.duplicate_groups} групп
                                  </Badge>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {result.processed_count ? (
                                  <Badge variant="secondary">
                                    {result.processed_count}
                                  </Badge>
                                ) : (
                                  '-'
                                )}
                              </TableCell>
                              <TableCell>
                                {(result.error_count || 0) > 0 ? (
                                  <Badge variant="destructive">
                                    <XCircle className="h-3 w-3 mr-1" />
                                    {result.error_count} ошибок
                                  </Badge>
                                ) : result.stopped ? (
                                  <Badge variant="outline">
                                    <AlertCircle className="h-3 w-3 mr-1" />
                                    Остановлено
                                  </Badge>
                                ) : (
                                  <Badge variant="default">
                                    <CheckCircle2 className="h-3 w-3 mr-1" />
                                    OK
                                  </Badge>
                                )}
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="charts" className="space-y-4">
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                  <Card>
                    <CardHeader>
                      <CardTitle>Время выполнения по этапам</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ResponsiveContainer width="100%" height={300}>
                        <DynamicBarChart data={report.results.map(r => ({
                          name: r.stage,
                          time: r.duration_ms,
                          percentage: ((r.duration_ms / report.total_duration_ms) * 100).toFixed(1)
                        }))}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                          <YAxis />
                          <Tooltip formatter={(value: any) => `${formatDuration(value)} (${((value / report.total_duration_ms) * 100).toFixed(1)}%)`} />
                          <DynamicBar dataKey="time" fill="#4CAF50">
                            {report.results.map((_, index) => (
                              <DynamicCell key={index} fill={COLORS[index % COLORS.length]} />
                            ))}
                          </DynamicBar>
                        </DynamicBarChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>

                  <Card>
                    <CardHeader>
                      <CardTitle>Скорость обработки</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ResponsiveContainer width="100%" height={300}>
                        <DynamicBarChart data={report.results.map(r => ({
                          name: r.stage,
                          speed: r.records_per_second
                        }))}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                          <YAxis />
                          <Tooltip formatter={(value: any) => `${value.toFixed(2)} записей/сек`} />
                          <DynamicBar dataKey="speed" fill="#2196F3">
                            {report.results.map((_, index) => (
                              <DynamicCell key={index} fill={COLORS[index % COLORS.length]} />
                            ))}
                          </DynamicBar>
                        </DynamicBarChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>

                  {report.results.some(r => r.memory_used_mb) && (
                    <Card>
                      <CardHeader>
                        <CardTitle>Использование памяти</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <ResponsiveContainer width="100%" height={300}>
                          <DynamicBarChart data={report.results.filter(r => r.memory_used_mb).map(r => ({
                            name: r.stage,
                            memory: r.memory_used_mb
                          }))}>
                            <CartesianGrid strokeDasharray="3 3" />
                            <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                            <YAxis />
                            <Tooltip formatter={(value: any) => `${value.toFixed(2)} МБ`} />
                            <DynamicBar dataKey="memory" fill="#FF9800">
                              {report.results.filter(r => r.memory_used_mb).map((_, index) => (
                                <DynamicCell key={index} fill={COLORS[index % COLORS.length]} />
                              ))}
                            </DynamicBar>
                          </DynamicBarChart>
                        </ResponsiveContainer>
                      </CardContent>
                    </Card>
                  )}

                  <Card>
                    <CardHeader>
                      <CardTitle>Распределение времени</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ResponsiveContainer width="100%" height={300}>
                        <DynamicBarChart data={report.results.map(r => ({
                          name: r.stage,
                          percentage: (r.duration_ms / report.total_duration_ms) * 100
                        }))}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                          <YAxis />
                          <Tooltip formatter={(value: any) => `${value.toFixed(1)}%`} />
                          <DynamicBar dataKey="percentage" fill="#9C27B0">
                            {report.results.map((_, index) => (
                              <DynamicCell key={index} fill={COLORS[index % COLORS.length]} />
                            ))}
                          </DynamicBar>
                        </DynamicBarChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                </div>
              </TabsContent>

              <TabsContent value="summary" className="space-y-4">
                <Card>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle>Сводка результатов</CardTitle>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => {
                          if (!report) return
                          const jsonStr = JSON.stringify(report, null, 2)
                          const blob = new Blob([jsonStr], { type: 'application/json' })
                          const url = URL.createObjectURL(blob)
                          const a = document.createElement('a')
                          a.href = url
                          a.download = `normalization_benchmark_${report.timestamp.replace(/[:.]/g, '-')}.json`
                          document.body.appendChild(a)
                          a.click()
                          document.body.removeChild(a)
                          URL.revokeObjectURL(url)
                          toast.success('Результаты экспортированы')
                        }}
                      >
                        <Download className="mr-2 h-4 w-4" />
                        Экспорт JSON
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label className="text-sm font-medium text-muted-foreground">Тест</Label>
                        <p className="text-lg font-semibold">{report.test_name}</p>
                      </div>
                      <div>
                        <Label className="text-sm font-medium text-muted-foreground">Дата</Label>
                        <p className="text-lg font-semibold">
                          {new Date(report.timestamp).toLocaleString('ru-RU')}
                        </p>
                      </div>
                      <div>
                        <Label className="text-sm font-medium text-muted-foreground">Воркеров</Label>
                        <p className="text-lg font-semibold">{report.workers}</p>
                      </div>
                      <div>
                        <Label className="text-sm font-medium text-muted-foreground">Этапов</Label>
                        <p className="text-lg font-semibold">{report.results.length}</p>
                      </div>
                    </div>

                    {report.summary && (
                      <div className="mt-4 space-y-2">
                        <Label className="text-sm font-medium">Детали</Label>
                        <div className="space-y-1">
                          {Object.entries(report.summary).map(([key, value]) => (
                            <div key={key} className="flex justify-between text-sm">
                              <span className="text-muted-foreground">{key}:</span>
                              <span className="font-medium">{String(value)}</span>
                            </div>
                          ))}
                        </div>
                      </div>
                    )}

                    {report.results.some(r => r.benchmark_matches || r.enriched_count || r.created_benchmarks) && (
                      <div className="mt-4 space-y-2">
                        <Label className="text-sm font-medium">Статистика эталонов</Label>
                        <div className="grid grid-cols-3 gap-4">
                          {report.results.some(r => r.benchmark_matches) && (
                            <div>
                              <Label className="text-xs text-muted-foreground">Совпадений</Label>
                              <p className="text-lg font-semibold">
                                {report.results.reduce((sum, r) => sum + (r.benchmark_matches || 0), 0)}
                              </p>
                            </div>
                          )}
                          {report.results.some(r => r.enriched_count) && (
                            <div>
                              <Label className="text-xs text-muted-foreground">Обогащено</Label>
                              <p className="text-lg font-semibold">
                                {report.results.reduce((sum, r) => sum + (r.enriched_count || 0), 0)}
                              </p>
                            </div>
                          )}
                          {report.results.some(r => r.created_benchmarks) && (
                            <div>
                              <Label className="text-xs text-muted-foreground">Создано эталонов</Label>
                              <p className="text-lg font-semibold">
                                {report.results.reduce((sum, r) => sum + (r.created_benchmarks || 0), 0)}
                              </p>
                            </div>
                          )}
                        </div>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="analysis" className="space-y-4">
                <Card>
                  <CardHeader>
                    <div className="flex items-center justify-between">
                      <CardTitle>Анализ узких мест</CardTitle>
                      <Button
                        onClick={analyzeBottlenecks}
                        disabled={loadingAnalysis || !report}
                        variant="outline"
                        size="sm"
                      >
                        {loadingAnalysis ? (
                          <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                        ) : (
                          <BarChart3 className="mr-2 h-4 w-4" />
                        )}
                        Обновить анализ
                      </Button>
                    </div>
                  </CardHeader>
                  <CardContent>
                    {loadingAnalysis ? (
                      <div className="flex items-center justify-center py-8">
                        <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
                      </div>
                    ) : bottlenecks.length === 0 ? (
                      <div className="text-center py-8 text-muted-foreground">
                        <AlertCircle className="h-8 w-8 mx-auto mb-2 opacity-50" />
                        <p>Нет данных для анализа</p>
                      </div>
                    ) : (
                      <div className="space-y-4">
                        {bottlenecks.map((b, index) => {
                          const severityColors = {
                            critical: 'destructive',
                            high: 'default',
                            medium: 'secondary',
                            low: 'outline'
                          } as const
                          const severityIcons = {
                            critical: '🔴',
                            high: '🟠',
                            medium: '🟡',
                            low: '✓'
                          }

                          return (
                            <Card key={index} className={b.severity === 'critical' ? 'border-red-500' : ''}>
                              <CardHeader>
                                <div className="flex items-center justify-between">
                                  <CardTitle className="text-lg">{b.stage}</CardTitle>
                                  <Badge variant={severityColors[b.severity]}>
                                    {severityIcons[b.severity]} {b.severity}
                                  </Badge>
                                </div>
                              </CardHeader>
                              <CardContent className="space-y-3">
                                <div className="grid grid-cols-3 gap-4 text-sm">
                                  <div>
                                    <Label className="text-xs text-muted-foreground">Время</Label>
                                    <p className="font-semibold">{formatDuration(b.duration_ms)}</p>
                                  </div>
                                  <div>
                                    <Label className="text-xs text-muted-foreground">% от общего</Label>
                                    <p className="font-semibold">{b.percentage.toFixed(1)}%</p>
                                  </div>
                                  <div>
                                    <Label className="text-xs text-muted-foreground">Скорость</Label>
                                    <p className="font-semibold">{b.records_per_second.toFixed(2)}/сек</p>
                                  </div>
                                </div>
                                {b.recommendations.length > 0 && (
                                  <div>
                                    <Label className="text-sm font-medium mb-2 block">Рекомендации:</Label>
                                    <ul className="space-y-1">
                                      {b.recommendations.map((rec, i) => (
                                        <li key={i} className="text-sm text-muted-foreground flex items-start gap-2">
                                          <span className="text-primary">•</span>
                                          <span>{rec}</span>
                                        </li>
                                      ))}
                                    </ul>
                                  </div>
                                )}
                              </CardContent>
                            </Card>
                          )
                        })}
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>

              <TabsContent value="compare" className="space-y-4">
                <Card>
                  <CardHeader>
                    <CardTitle>Сравнение бенчмарков</CardTitle>
                    <CardDescription>
                      Выберите два бенчмарка для сравнения производительности
                    </CardDescription>
                  </CardHeader>
                  <CardContent className="space-y-4">
                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label>Базовый бенчмарк</Label>
                        <Select value={selectedBaseline} onValueChange={setSelectedBaseline}>
                          <SelectTrigger className="mt-2">
                            <SelectValue placeholder="Выберите бенчмарк..." />
                          </SelectTrigger>
                          <SelectContent>
                            {benchmarksList.map((b) => (
                              <SelectItem key={b.id} value={b.id}>
                                {b.test_name || 'Бенчмарк'} - {new Date(b.timestamp).toLocaleDateString('ru-RU')}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                      <div>
                        <Label>Текущий бенчмарк</Label>
                        <Select value={selectedCurrent} onValueChange={setSelectedCurrent}>
                          <SelectTrigger className="mt-2">
                            <SelectValue placeholder="Выберите бенчмарк..." />
                          </SelectTrigger>
                          <SelectContent>
                            {benchmarksList.map((b) => (
                              <SelectItem key={b.id} value={b.id}>
                                {b.test_name || 'Бенчмарк'} - {new Date(b.timestamp).toLocaleDateString('ru-RU')}
                              </SelectItem>
                            ))}
                          </SelectContent>
                        </Select>
                      </div>
                    </div>
                    <Button
                      onClick={compareBenchmarks}
                      disabled={loadingComparison || !selectedBaseline || !selectedCurrent}
                      className="w-full"
                    >
                      {loadingComparison ? (
                        <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                      ) : (
                        <TrendingUp className="mr-2 h-4 w-4" />
                      )}
                      Сравнить
                    </Button>

                    {comparisonData && (
                      <div className="mt-6 space-y-4">
                        <Card>
                          <CardHeader>
                            <CardTitle>Общее сравнение</CardTitle>
                          </CardHeader>
                          <CardContent>
                            <div className="grid grid-cols-2 gap-4">
                              <div>
                                <Label className="text-xs text-muted-foreground">Скорость</Label>
                                <p className={`text-lg font-semibold ${comparisonData.summary.speed_change > 0 ? 'text-green-600' : comparisonData.summary.speed_change < 0 ? 'text-red-600' : ''}`}>
                                  {comparisonData.summary.speed_change > 0 ? '+' : ''}
                                  {comparisonData.summary.speed_change.toFixed(2)}%
                                </p>
                              </div>
                              <div>
                                <Label className="text-xs text-muted-foreground">Время</Label>
                                <p className={`text-lg font-semibold ${comparisonData.summary.duration_change < 0 ? 'text-green-600' : comparisonData.summary.duration_change > 0 ? 'text-red-600' : ''}`}>
                                  {comparisonData.summary.duration_change > 0 ? '+' : ''}
                                  {comparisonData.summary.duration_change.toFixed(2)}%
                                </p>
                              </div>
                            </div>
                            <div className="mt-4 grid grid-cols-3 gap-4">
                              <div>
                                <Label className="text-xs text-muted-foreground">Улучшений</Label>
                                <p className="text-lg font-semibold text-green-600">{comparisonData.summary.improvements}</p>
                              </div>
                              <div>
                                <Label className="text-xs text-muted-foreground">Ухудшений</Label>
                                <p className="text-lg font-semibold text-red-600">{comparisonData.summary.regressions}</p>
                              </div>
                              <div>
                                <Label className="text-xs text-muted-foreground">Без изменений</Label>
                                <p className="text-lg font-semibold">{comparisonData.summary.no_changes}</p>
                              </div>
                            </div>
                          </CardContent>
                        </Card>

                        <Card>
                          <CardHeader>
                            <CardTitle>Детальное сравнение по этапам</CardTitle>
                          </CardHeader>
                          <CardContent>
                            <Table>
                              <TableHeader>
                                <TableRow>
                                  <TableHead>Этап</TableHead>
                                  <TableHead>Скорость</TableHead>
                                  <TableHead>Время</TableHead>
                                  <TableHead>Память</TableHead>
                                  <TableHead>Статус</TableHead>
                                </TableRow>
                              </TableHeader>
                              <TableBody>
                                {comparisonData.comparisons.map((comp, index) => (
                                  <TableRow key={index}>
                                    <TableCell className="font-medium">{comp.stage}</TableCell>
                                    <TableCell>
                                      <span className={comp.speed_change_percent > 0 ? 'text-green-600' : comp.speed_change_percent < 0 ? 'text-red-600' : ''}>
                                        {comp.speed_change_percent > 0 ? '+' : ''}
                                        {comp.speed_change_percent.toFixed(2)}%
                                      </span>
                                    </TableCell>
                                    <TableCell>
                                      <span className={comp.duration_change_percent < 0 ? 'text-green-600' : comp.duration_change_percent > 0 ? 'text-red-600' : ''}>
                                        {comp.duration_change_percent > 0 ? '+' : ''}
                                        {comp.duration_change_percent.toFixed(2)}%
                                      </span>
                                    </TableCell>
                                    <TableCell>
                                      {comp.memory_change_percent !== 0 && (
                                        <span className={comp.memory_change_percent < 0 ? 'text-green-600' : 'text-red-600'}>
                                          {comp.memory_change_percent > 0 ? '+' : ''}
                                          {comp.memory_change_percent.toFixed(2)}%
                                        </span>
                                      )}
                                    </TableCell>
                                    <TableCell>
                                      {comp.improvement ? (
                                        <Badge variant="default">
                                          <CheckCircle2 className="h-3 w-3 mr-1" />
                                          Улучшение
                                        </Badge>
                                      ) : (
                                        <Badge variant="destructive">
                                          <XCircle className="h-3 w-3 mr-1" />
                                          Ухудшение
                                        </Badge>
                                      )}
                                    </TableCell>
                                  </TableRow>
                                ))}
                              </TableBody>
                            </Table>
                          </CardContent>
                        </Card>
                      </div>
                    )}
                  </CardContent>
                </Card>
              </TabsContent>
            </Tabs>
          </FadeIn>
        </>
      )}

      {!report && !loading && (
        <FadeIn delay={0.2}>
          <Card>
            <CardContent className="py-12 text-center">
              <FileText className="h-12 w-12 mx-auto text-muted-foreground mb-4" />
              <h3 className="text-lg font-semibold mb-2">Нет загруженных результатов</h3>
              <p className="text-muted-foreground mb-4">
                Загрузите JSON файл с результатами бенчмарка для анализа или выберите из сохраненных
              </p>
              <div className="flex items-center justify-center gap-2">
                <Button onClick={() => fileInputRef.current?.click()}>
                  <Upload className="mr-2 h-4 w-4" />
                  Загрузить файл
                </Button>
                {benchmarksList.length > 0 && (
                  <Button variant="outline" onClick={() => loadBenchmark(benchmarksList[0].id)}>
                    <List className="mr-2 h-4 w-4" />
                    Открыть последний
                  </Button>
                )}
              </div>
            </CardContent>
          </Card>
        </FadeIn>
      )}
    </div>
  )
}

