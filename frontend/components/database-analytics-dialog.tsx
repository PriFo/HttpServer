'use client'

import { useState, useEffect, useCallback } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { RefreshCw, Database, TrendingUp, BarChart3, History } from 'lucide-react'
import {
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'
import { DatabaseTypeBadge } from './database-type-badge'

interface TableStat {
  name: string
  row_count: number
  size_bytes: number
  size_mb: number
}

interface HistoryEntry {
  timestamp: string
  size: number
  size_mb: number
  row_count: number
}

interface DatabaseAnalytics {
  file_path: string
  database_type: string
  total_size: number
  total_size_mb: number
  table_count: number
  total_rows: number
  table_stats: TableStat[]
  top_tables: TableStat[]
  analyzed_at: string
}

interface DatabaseAnalyticsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  databaseName: string
  databasePath: string
}

const COLORS = ['#3b82f6', '#8b5cf6', '#ec4899', '#f59e0b', '#10b981', '#ef4444', '#06b6d4', '#84cc16']

export function DatabaseAnalyticsDialog({
  open,
  onOpenChange,
  databaseName,
  databasePath,
}: DatabaseAnalyticsDialogProps) {
  const [analytics, setAnalytics] = useState<DatabaseAnalytics | null>(null)
  const [history, setHistory] = useState<HistoryEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchAnalytics = useCallback(async () => {
    setLoading(true)
    setError(null)

    try {
      // Используем путь к базе данных, если он доступен, иначе имя
      const dbIdentifier = databasePath || databaseName
      if (!dbIdentifier) {
        throw new Error('Database identifier is required')
      }
      
      // Кодируем имя базы данных для URL (убираем расширение .db из пути, если оно есть)
      const dbNameForUrl = databaseName.replace(/\.db$/i, '')
      const url = `/api/databases/analytics/${encodeURIComponent(dbNameForUrl)}?path=${encodeURIComponent(dbIdentifier)}`
      
      const response = await fetch(url)
      if (!response.ok) {
        let errorMessage = 'Failed to fetch analytics'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          // Если не удалось распарсить JSON, используем статус
          errorMessage = `HTTP ${response.status}: ${response.statusText}`
        }
        throw new Error(errorMessage)
      }
      const data = await response.json()
      setAnalytics(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [databaseName, databasePath])

  const fetchHistory = useCallback(async () => {
    try {
      if (!databaseName) {
        console.warn('Database name is not provided for history fetch')
        return
      }
      
      const response = await fetch(`/api/databases/history/${encodeURIComponent(databaseName)}`)
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Unknown error' }))
        throw new Error(errorData.error || 'Failed to fetch history')
      }
      const data = await response.json()
      setHistory(data.history || [])
    } catch (err) {
      console.error('Failed to fetch history:', err)
      // Не устанавливаем ошибку в state, так как история не критична
    }
  }, [databaseName])

  useEffect(() => {
    if (open && databaseName) {
      fetchAnalytics()
      fetchHistory()
    }
  }, [open, databaseName, fetchAnalytics, fetchHistory])

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    if (!dateString) return 'Неизвестно'
    const date = new Date(dateString)
    return date.toLocaleDateString('ru-RU', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  // Подготовка данных для графиков
  const historyChartData = history.map((entry) => ({
    date: new Date(entry.timestamp).toLocaleDateString('ru-RU', { month: 'short', day: 'numeric' }),
    size: entry.size_mb,
    rows: entry.row_count,
  }))

  const topTablesData = analytics?.top_tables.slice(0, 10).map((table) => ({
    name: table.name.length > 20 ? table.name.substring(0, 20) + '...' : table.name,
    size: table.size_mb,
    rows: table.row_count,
  })) || []

  const pieData = analytics?.top_tables.slice(0, 8).map((table) => ({
    name: table.name,
    value: table.size_mb,
  })) || []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-6xl max-h-[90vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Аналитика базы данных: {databaseName}
          </DialogTitle>
          <DialogDescription>
            Детальная информация и статистика по базе данных
          </DialogDescription>
        </DialogHeader>

        {loading && (
          <div className="flex items-center justify-center py-8">
            <RefreshCw className="h-8 w-8 animate-spin text-primary" />
          </div>
        )}

        {error && (
          <div className="p-4 bg-destructive/10 border border-destructive rounded-lg text-destructive">
            Ошибка: {error}
          </div>
        )}

        {analytics && !loading && (
          <Tabs defaultValue="overview" className="w-full flex-1 flex flex-col overflow-hidden">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="overview">Обзор</TabsTrigger>
              <TabsTrigger value="tables">Таблицы</TabsTrigger>
              <TabsTrigger value="charts">Графики</TabsTrigger>
              <TabsTrigger value="history">История</TabsTrigger>
            </TabsList>

            <TabsContent value="overview" className="space-y-4 overflow-y-auto flex-1">
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm font-medium">Тип базы данных</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <DatabaseTypeBadge type={analytics.database_type} />
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm font-medium">Общий размер</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">{formatFileSize(analytics.total_size)}</p>
                    <p className="text-sm text-muted-foreground">
                      {analytics.total_size_mb.toFixed(2)} MB
                    </p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm font-medium">Количество таблиц</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">{analytics.table_count}</p>
                  </CardContent>
                </Card>

                <Card>
                  <CardHeader>
                    <CardTitle className="text-sm font-medium">Всего записей</CardTitle>
                  </CardHeader>
                  <CardContent>
                    <p className="text-2xl font-bold">
                      {analytics.total_rows.toLocaleString('ru-RU')}
                    </p>
                  </CardContent>
                </Card>
              </div>

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <BarChart3 className="h-4 w-4" />
                    Топ таблиц по размеру
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {analytics.top_tables.slice(0, 5).map((table, index) => (
                      <div
                        key={table.name}
                        className="flex items-center justify-between p-2 bg-muted rounded-lg"
                      >
                        <div className="flex items-center gap-2">
                          <Badge variant="outline">{index + 1}</Badge>
                          <span className="font-medium">{table.name}</span>
                        </div>
                        <div className="flex items-center gap-4 text-sm">
                          <span className="text-muted-foreground">
                            {table.row_count.toLocaleString('ru-RU')} записей
                          </span>
                          <span className="font-semibold">{formatFileSize(table.size_bytes)}</span>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="tables" className="space-y-4 overflow-hidden flex flex-col">
              <Card className="flex-1 flex flex-col overflow-hidden">
                <CardHeader>
                  <CardTitle>Статистика по таблицам</CardTitle>
                  <CardDescription>
                    Детальная информация о всех таблицах базы данных
                  </CardDescription>
                </CardHeader>
                <CardContent className="p-0 flex-1 overflow-hidden">
                  <div className="overflow-auto h-full">
                    <table className="w-full text-sm border-collapse">
                      <thead className="sticky top-0 bg-background z-10 border-b">
                        <tr>
                          <th className="text-left p-3 font-semibold min-w-[200px]">Таблица</th>
                          <th className="text-right p-3 font-semibold w-[120px]">Записей</th>
                          <th className="text-right p-3 font-semibold w-[100px]">Размер</th>
                          <th className="text-right p-3 font-semibold w-[100px]">MB</th>
                        </tr>
                      </thead>
                      <tbody>
                        {analytics.table_stats.map((table, index) => (
                          <tr 
                            key={table.name} 
                            className={`border-b hover:bg-muted/50 transition-colors ${
                              index % 2 === 0 ? 'bg-background' : 'bg-muted/30'
                            }`}
                          >
                            <td className="p-3 font-mono text-xs break-all">{table.name}</td>
                            <td className="text-right p-3">
                              {table.row_count.toLocaleString('ru-RU')}
                            </td>
                            <td className="text-right p-3 text-muted-foreground">
                              {formatFileSize(table.size_bytes)}
                            </td>
                            <td className="text-right p-3 font-medium">
                              {table.size_mb.toFixed(2)}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="charts" className="space-y-4 overflow-y-auto flex-1">
              <div className="grid grid-cols-1 gap-4">
                {history.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <TrendingUp className="h-4 w-4" />
                        Рост размера базы данных
                      </CardTitle>
                      <CardDescription>Изменение размера за последние 30 дней</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ResponsiveContainer width="100%" height={300}>
                        <LineChart data={historyChartData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="date" />
                          <YAxis label={{ value: 'Размер (MB)', angle: -90, position: 'insideLeft' }} />
                          <Tooltip />
                          <Legend />
                          <Line
                            type="monotone"
                            dataKey="size"
                            stroke="#3b82f6"
                            name="Размер (MB)"
                            strokeWidth={2}
                          />
                        </LineChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                )}

                {topTablesData.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center gap-2">
                        <BarChart3 className="h-4 w-4" />
                        Топ таблиц по размеру
                      </CardTitle>
                      <CardDescription>10 самых больших таблиц</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ResponsiveContainer width="100%" height={300}>
                        <BarChart data={topTablesData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                          <YAxis label={{ value: 'Размер (MB)', angle: -90, position: 'insideLeft' }} />
                          <Tooltip />
                          <Legend />
                          <Bar dataKey="size" fill="#3b82f6" name="Размер (MB)" />
                        </BarChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                )}

                {pieData.length > 0 && (
                  <Card>
                    <CardHeader>
                      <CardTitle>Распределение по таблицам</CardTitle>
                      <CardDescription>Топ-8 таблиц по размеру</CardDescription>
                    </CardHeader>
                    <CardContent>
                      <ResponsiveContainer width="100%" height={300}>
                        <PieChart>
                          <Pie
                            data={pieData}
                            cx="50%"
                            cy="50%"
                            labelLine={false}
                            label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(0)}%`}
                            outerRadius={80}
                            fill="#8884d8"
                            dataKey="value"
                          >
                            {pieData.map((entry, index) => (
                              <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                            ))}
                          </Pie>
                          <Tooltip />
                        </PieChart>
                      </ResponsiveContainer>
                    </CardContent>
                  </Card>
                )}
              </div>
            </TabsContent>

            <TabsContent value="history" className="space-y-4 overflow-y-auto flex-1">
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <History className="h-4 w-4" />
                    История изменений
                  </CardTitle>
                  <CardDescription>
                    История изменения размера и количества записей за последние 30 дней
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {history.length === 0 ? (
                    <p className="text-center text-muted-foreground py-8">
                      История изменений пока недоступна
                    </p>
                  ) : (
                    <div className="space-y-2">
                      {history.map((entry, index) => (
                        <div
                          key={index}
                          className="flex items-center justify-between p-3 bg-muted rounded-lg"
                        >
                          <div>
                            <p className="font-medium">{formatDate(entry.timestamp)}</p>
                            <p className="text-sm text-muted-foreground">
                              {entry.row_count.toLocaleString('ru-RU')} записей
                            </p>
                          </div>
                          <div className="text-right">
                            <p className="font-semibold">{formatFileSize(entry.size)}</p>
                            <p className="text-sm text-muted-foreground">
                              {entry.size_mb.toFixed(2)} MB
                            </p>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        )}
      </DialogContent>
    </Dialog>
  )
}

