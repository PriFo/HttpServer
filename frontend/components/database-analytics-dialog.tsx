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
import { RefreshCw, Database, TrendingUp, BarChart3, History, Upload, FileText, Layers, Calendar } from 'lucide-react'
import {
  DynamicLineChart,
  DynamicLine,
  DynamicBarChart,
  DynamicBar,
  DynamicPieChart,
  DynamicPie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from '@/lib/recharts-dynamic'
import { DatabaseTypeBadge } from './database-type-badge'
import { formatDateTime, formatDate as formatDateSafe, formatNumber, formatFileSize } from '@/lib/locale'
import { fetchJson, getErrorMessage } from '@/lib/fetch-utils'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { toast } from 'sonner'

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
  upload_stats?: {
    total_uploads?: number
    uploads_count?: number
    total_catalogs?: number
    catalogs_count?: number
    total_items?: number
    items_count?: number
    last_upload_date?: string
    avg_items_per_upload?: number
  }
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
      
      // Используем формат без dbname в пути, только query параметр path
      const url = `/api/databases/analytics?path=${encodeURIComponent(dbIdentifier)}`
      
      const data = await fetchJson<DatabaseAnalytics>(url, {
        timeout: QUALITY_TIMEOUTS.STANDARD,
        cache: 'no-store',
      })
      
      // Обеспечиваем, что все массивы инициализированы
      const safeData: DatabaseAnalytics = {
        ...data,
        table_stats: data.table_stats || [],
        top_tables: data.top_tables || [],
      }
      
      setAnalytics(safeData)
    } catch (err) {
      const errorMessage = getErrorMessage(err, 'Не удалось загрузить аналитику базы данных')
      setError(errorMessage)
      // Показываем toast уведомление для критических ошибок
      toast.error('Ошибка загрузки аналитики', {
        description: errorMessage,
        duration: 5000,
      })
    } finally {
      setLoading(false)
    }
  }, [databaseName, databasePath])

  const fetchHistory = useCallback(async () => {
    try {
      // Используем путь к базе данных, если он доступен, иначе имя
      const dbIdentifier = databasePath || databaseName
      if (!dbIdentifier) {
        // История не критична, просто пропускаем загрузку
        return
      }
      
      // Бэкенд ожидает имя БД в пути: /api/databases/history/{dbName}
      // Извлекаем имя файла из пути, если путь полный
      const dbName = dbIdentifier.includes('/') || dbIdentifier.includes('\\')
        ? dbIdentifier.split(/[/\\]/).pop() || dbIdentifier
        : dbIdentifier
      const url = `/api/databases/history/${encodeURIComponent(dbName)}`
      
      const data = await fetchJson<{ history: HistoryEntry[] }>(
        url,
        {
          timeout: QUALITY_TIMEOUTS.STANDARD,
          cache: 'no-store',
        }
      )
      setHistory(data.history || [])
    } catch (err) {
      // История не критична для работы компонента, поэтому не показываем ошибку пользователю
      // Только логируем для отладки (в dev режиме)
      if (process.env.NODE_ENV === 'development') {
        console.debug('Failed to fetch history (non-critical):', err)
      }
      // Устанавливаем пустую историю, чтобы показать соответствующее сообщение в UI
      setHistory([])
    }
  }, [databaseName, databasePath])

  useEffect(() => {
    if (open && databaseName) {
      fetchAnalytics()
      fetchHistory()
    }
  }, [open, databaseName, fetchAnalytics, fetchHistory])

  // Используем безопасную функцию форматирования размера файла из lib/locale

  // Используем безопасную функцию форматирования из lib/locale
  const formatDate = (dateString: string | null | undefined) => {
    return formatDateTime(dateString, {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  // Подготовка данных для графиков
  const historyChartData = history.map((entry) => {
    return {
      date: formatDateSafe(entry.timestamp, { month: 'short', day: 'numeric' }),
      size: entry.size_mb,
      rows: entry.row_count,
    }
  })

  const topTablesData = (analytics?.top_tables || []).slice(0, 10).map((table) => ({
    name: table.name.length > 20 ? table.name.substring(0, 20) + '...' : table.name,
    size: table.size_mb,
    rows: table.row_count,
  }))

  const pieData = (analytics?.top_tables || []).slice(0, 8).map((table) => ({
    name: table.name,
    value: table.size_mb,
  }))

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
                      {formatNumber(analytics.total_rows)}
                    </p>
                  </CardContent>
                </Card>
              </div>

              {/* Статистика из выгрузок */}
              {analytics.upload_stats && (analytics.upload_stats.total_uploads !== undefined || analytics.upload_stats.uploads_count !== undefined) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                      <Upload className="h-4 w-4" />
                      Статистика выгрузок
                    </CardTitle>
                    <CardDescription>
                      Информация из таблицы uploads
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                      <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                        <div className="p-2 rounded-lg bg-primary/10">
                          <Layers className="h-4 w-4 text-primary" />
                        </div>
                        <div>
                          <p className="text-sm text-muted-foreground">Выгрузок</p>
                          <p className="text-xl font-bold">
                            {formatNumber(analytics.upload_stats.total_uploads ?? analytics.upload_stats.uploads_count ?? 0)}
                          </p>
                        </div>
                      </div>
                      {analytics.upload_stats.total_catalogs !== undefined && (
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                          <div className="p-2 rounded-lg bg-primary/10">
                            <FileText className="h-4 w-4 text-primary" />
                          </div>
                          <div>
                            <p className="text-sm text-muted-foreground">Справочников</p>
                            <p className="text-xl font-bold">
                              {formatNumber(analytics.upload_stats.total_catalogs ?? analytics.upload_stats.catalogs_count ?? 0)}
                            </p>
                          </div>
                        </div>
                      )}
                      {analytics.upload_stats.total_items !== undefined && (
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                          <div className="p-2 rounded-lg bg-primary/10">
                            <Database className="h-4 w-4 text-primary" />
                          </div>
                          <div>
                            <p className="text-sm text-muted-foreground">Записей</p>
                            <p className="text-xl font-bold">
                              {formatNumber(analytics.upload_stats.total_items ?? analytics.upload_stats.items_count ?? 0)}
                            </p>
                          </div>
                        </div>
                      )}
                      {analytics.upload_stats.last_upload_date && (
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                          <div className="p-2 rounded-lg bg-primary/10">
                            <Calendar className="h-4 w-4 text-primary" />
                          </div>
                          <div>
                            <p className="text-sm text-muted-foreground">Последняя выгрузка</p>
                            <p className="text-xl font-bold">
                              {formatDateTime(new Date(analytics.upload_stats.last_upload_date))}
                            </p>
                          </div>
                        </div>
                      )}
                      {analytics.upload_stats.avg_items_per_upload !== undefined && (
                        <div className="flex items-center gap-3 p-3 bg-muted/50 rounded-lg">
                          <div className="p-2 rounded-lg bg-primary/10">
                            <TrendingUp className="h-4 w-4 text-primary" />
                          </div>
                          <div>
                            <p className="text-sm text-muted-foreground">Среднее записей/выгрузка</p>
                            <p className="text-xl font-bold">
                              {formatNumber(Math.round(analytics.upload_stats.avg_items_per_upload))}
                            </p>
                          </div>
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              )}

              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <BarChart3 className="h-4 w-4" />
                    Топ таблиц по размеру
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {(analytics.top_tables || []).slice(0, 5).map((table, index) => (
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
                            {formatNumber(table.row_count)} записей
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
                        {(analytics.table_stats || []).map((table, index) => (
                          <tr 
                            key={table.name} 
                            className={`border-b hover:bg-muted/50 transition-colors ${
                              index % 2 === 0 ? 'bg-background' : 'bg-muted/30'
                            }`}
                          >
                            <td className="p-3 font-mono text-xs break-all">{table.name}</td>
                            <td className="text-right p-3">
                              {formatNumber(table.row_count)}
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
                        <DynamicLineChart data={historyChartData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="date" />
                          <YAxis label={{ value: 'Размер (MB)', angle: -90, position: 'insideLeft' }} />
                          <Tooltip />
                          <Legend />
                          <DynamicLine
                            type="monotone"
                            dataKey="size"
                            stroke="#3b82f6"
                            name="Размер (MB)"
                            strokeWidth={2}
                          />
                        </DynamicLineChart>
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
                        <DynamicBarChart data={topTablesData}>
                          <CartesianGrid strokeDasharray="3 3" />
                          <XAxis dataKey="name" angle={-45} textAnchor="end" height={100} />
                          <YAxis label={{ value: 'Размер (MB)', angle: -90, position: 'insideLeft' }} />
                          <Tooltip />
                          <Legend />
                          <DynamicBar dataKey="size" fill="#3b82f6" name="Размер (MB)" />
                        </DynamicBarChart>
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
                        <DynamicPieChart>
                          <DynamicPie
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
                          </DynamicPie>
                          <Tooltip />
                        </DynamicPieChart>
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
                              {formatNumber(entry.row_count)} записей
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

