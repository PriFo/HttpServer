'use client'

import { useState, useEffect, useMemo } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'
import { 
  DynamicLineChart, 
  DynamicLine, 
  DynamicBarChart, 
  DynamicBar, 
  DynamicPieChart, 
  DynamicPie, 
  DynamicCell, 
  DynamicAreaChart,
  DynamicArea,
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  Legend, 
  ResponsiveContainer,
} from '@/lib/recharts-dynamic'
import { TrendingUp, Activity, CheckCircle2, XCircle, Clock, BarChart3 } from 'lucide-react'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale/ru'

interface PerformanceDataPoint {
  timestamp: string
  processed: number
  success: number
  errors: number
  progress: number
  rate?: number // записей в секунду
}

interface SessionStat {
  id: number
  date: string
  processed: number
  success: number
  errors: number
  duration: number
  status: string
}

interface NormalizationPerformanceChartsProps {
  type: 'nomenclature' | 'counterparties'
  clientId?: number | null
  projectId?: number | null
  currentStatus?: {
    processed: number
    total: number
    success?: number
    errors?: number
    startTime?: string
  }
}

const COLORS = ['#10b981', '#ef4444', '#3b82f6', '#f59e0b', '#8b5cf6']

export function NormalizationPerformanceCharts({ 
  type, 
  clientId,
  projectId,
  currentStatus 
}: NormalizationPerformanceChartsProps) {
  const [performanceData, setPerformanceData] = useState<PerformanceDataPoint[]>([])
  const [sessionStats, setSessionStats] = useState<SessionStat[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      setLoading(true)
      try {
        // Определяем endpoint в зависимости от типа и наличия clientId/projectId
        let endpoint = `/api/normalization/stats?type=${type}`
        if (clientId && projectId) {
          if (type === 'nomenclature') {
            endpoint = `/api/clients/${clientId}/projects/${projectId}/normalization/stats`
          } else {
            endpoint = `/api/counterparties/normalized/stats?client_id=${clientId}&project_id=${projectId}`
          }
        }
        
        // Получаем историю сессий для статистики
        const response = await fetch(endpoint, {
          cache: 'no-store',
        })

        if (response.ok) {
          const data = await response.json()
          
          // Преобразуем данные сессий в формат для графиков
          if (data.sessions && Array.isArray(data.sessions)) {
            const stats: SessionStat[] = data.sessions.slice(0, 10).map((session: any) => ({
              id: session.id,
              date: format(new Date(session.created_at), 'dd.MM', { locale: ru }),
              processed: session.processed_count || 0,
              success: session.success_count || 0,
              errors: session.error_count || 0,
              duration: session.finished_at 
                ? Math.floor((new Date(session.finished_at).getTime() - new Date(session.created_at).getTime()) / 1000)
                : 0,
              status: session.status,
            }))
            setSessionStats(stats)
          }
        }

        // Генерируем данные производительности на основе текущего статуса
        if (currentStatus && currentStatus.startTime) {
          const startTime = new Date(currentStatus.startTime)
          const now = new Date()
          const dataPoints: PerformanceDataPoint[] = []
          
          // Создаем точки данных за последние 30 минут (если процесс запущен)
          for (let i = 29; i >= 0; i--) {
            const timestamp = new Date(now.getTime() - i * 60000)
            const elapsed = (timestamp.getTime() - startTime.getTime()) / 1000
            
            if (elapsed > 0) {
              // Симулируем прогресс (в реальности это должно приходить с бэкенда)
              const progress = Math.min(
                (currentStatus.processed / currentStatus.total) * 100,
                100
              )
              
              dataPoints.push({
                timestamp: format(timestamp, 'HH:mm', { locale: ru }),
                processed: Math.floor((currentStatus.processed / 30) * (30 - i)),
                success: Math.floor(((currentStatus.success || 0) / 30) * (30 - i)),
                errors: Math.floor(((currentStatus.errors || 0) / 30) * (30 - i)),
                progress,
                rate: currentStatus.processed > 0 && elapsed > 0 
                  ? Math.floor(currentStatus.processed / elapsed) 
                  : 0,
              })
            }
          }
          
          setPerformanceData(dataPoints)
        }
      } catch (err) {
        console.error('Error fetching performance data:', err)
        // Не показываем ошибку пользователю, так как это не критично
        // Графики просто не будут обновляться
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    
    // Обновляем данные каждые 5 секунд, если процесс запущен
    const interval = setInterval(() => {
      if (currentStatus) {
        fetchData()
      }
    }, 5000)

    return () => clearInterval(interval)
  }, [type, clientId, projectId, currentStatus])

  // Данные для графика успешности по сессиям
  const successRateData = useMemo(() => {
    return sessionStats.map(session => ({
      name: session.date,
      success: session.success,
      errors: session.errors,
      successRate: session.processed > 0 
        ? ((session.success / session.processed) * 100).toFixed(1)
        : 0,
    }))
  }, [sessionStats])

  // Данные для круговой диаграммы статусов
  const statusDistribution = useMemo(() => {
    const statusCounts: Record<string, number> = {}
    sessionStats.forEach(session => {
      statusCounts[session.status] = (statusCounts[session.status] || 0) + 1
    })
    
    return Object.entries(statusCounts).map(([status, count]) => ({
      name: status === 'completed' ? 'Завершено' : 
            status === 'running' ? 'Выполняется' :
            status === 'failed' ? 'Ошибка' : status,
      value: count,
    }))
  }, [sessionStats])

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Графики производительности
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-64 w-full" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BarChart3 className="h-5 w-5" />
          Графики производительности
        </CardTitle>
        <CardDescription>
          Визуализация производительности процессов нормализации {type === 'nomenclature' ? 'номенклатуры' : 'контрагентов'}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="progress" className="w-full">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="progress">Прогресс</TabsTrigger>
            <TabsTrigger value="success">Успешность</TabsTrigger>
            <TabsTrigger value="sessions">Сессии</TabsTrigger>
            <TabsTrigger value="status">Статусы</TabsTrigger>
          </TabsList>

          <TabsContent value="progress" className="space-y-4">
            {performanceData.length > 0 ? (
              <>
                <div>
                  <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                    <TrendingUp className="h-4 w-4" />
                    Прогресс обработки
                  </h3>
                  <ResponsiveContainer width="100%" height={300}>
                    <DynamicAreaChart data={performanceData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="timestamp" />
                      <YAxis yAxisId="left" />
                      <YAxis yAxisId="right" orientation="right" />
                      <Tooltip />
                      <Legend />
                      <DynamicArea
                        yAxisId="left"
                        type="monotone"
                        dataKey="processed"
                        stroke="#3b82f6"
                        fill="#3b82f6"
                        fillOpacity={0.6}
                        name="Обработано"
                      />
                      <DynamicArea
                        yAxisId="left"
                        type="monotone"
                        dataKey="success"
                        stroke="#10b981"
                        fill="#10b981"
                        fillOpacity={0.6}
                        name="Успешно"
                      />
                      <DynamicArea
                        yAxisId="left"
                        type="monotone"
                        dataKey="errors"
                        stroke="#ef4444"
                        fill="#ef4444"
                        fillOpacity={0.6}
                        name="Ошибок"
                      />
                    </DynamicAreaChart>
                  </ResponsiveContainer>
                </div>

                {performanceData.some(d => d.rate !== undefined && d.rate > 0) && (
                  <div>
                    <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                      <Activity className="h-4 w-4" />
                      Скорость обработки (записей/сек)
                    </h3>
                    <ResponsiveContainer width="100%" height={250}>
                      <DynamicLineChart data={performanceData}>
                        <CartesianGrid strokeDasharray="3 3" />
                        <XAxis dataKey="timestamp" />
                        <YAxis />
                        <Tooltip />
                        <Legend />
                        <DynamicLine
                          type="monotone"
                          dataKey="rate"
                          stroke="#8b5cf6"
                          strokeWidth={2}
                          name="Записей/сек"
                          dot={{ r: 4 }}
                        />
                      </DynamicLineChart>
                    </ResponsiveContainer>
                  </div>
                )}
              </>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <Clock className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Нет данных о производительности</p>
                <p className="text-sm mt-2">Запустите процесс нормализации для отображения графиков</p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="success" className="space-y-4">
            {successRateData.length > 0 ? (
              <>
                <div>
                  <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4" />
                    Успешность по сессиям
                  </h3>
                  <ResponsiveContainer width="100%" height={300}>
                    <DynamicBarChart data={successRateData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis />
                      <Tooltip />
                      <Legend />
                      <DynamicBar dataKey="success" fill="#10b981" name="Успешно" />
                      <DynamicBar dataKey="errors" fill="#ef4444" name="Ошибок" />
                    </DynamicBarChart>
                  </ResponsiveContainer>
                </div>

                <div>
                  <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                    <TrendingUp className="h-4 w-4" />
                    Процент успешности
                  </h3>
                  <ResponsiveContainer width="100%" height={250}>
                    <DynamicLineChart data={successRateData}>
                      <CartesianGrid strokeDasharray="3 3" />
                      <XAxis dataKey="name" />
                      <YAxis domain={[0, 100]} />
                      <Tooltip formatter={(value: any) => `${value}%`} />
                      <Legend />
                      <DynamicLine
                        type="monotone"
                        dataKey="successRate"
                        stroke="#10b981"
                        strokeWidth={2}
                        name="Успешность (%)"
                        dot={{ r: 4 }}
                      />
                    </DynamicLineChart>
                  </ResponsiveContainer>
                </div>
              </>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <BarChart3 className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Нет данных о сессиях</p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="sessions" className="space-y-4">
            {sessionStats.length > 0 ? (
              <div>
                <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                  <Activity className="h-4 w-4" />
                  Производительность сессий
                </h3>
                <ResponsiveContainer width="100%" height={300}>
                  <DynamicBarChart data={sessionStats}>
                    <CartesianGrid strokeDasharray="3 3" />
                    <XAxis dataKey="date" />
                    <YAxis yAxisId="left" />
                    <YAxis yAxisId="right" orientation="right" />
                    <Tooltip />
                    <Legend />
                    <DynamicBar yAxisId="left" dataKey="processed" fill="#3b82f6" name="Обработано" />
                    <DynamicBar yAxisId="right" dataKey="duration" fill="#f59e0b" name="Длительность (сек)" />
                  </DynamicBarChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <Activity className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Нет данных о сессиях</p>
              </div>
            )}
          </TabsContent>

          <TabsContent value="status" className="space-y-4">
            {statusDistribution.length > 0 ? (
              <div>
                <h3 className="text-sm font-medium mb-2 flex items-center gap-2">
                  <BarChart3 className="h-4 w-4" />
                  Распределение по статусам
                </h3>
                <ResponsiveContainer width="100%" height={300}>
                  <DynamicPieChart>
                    <DynamicPie
                      data={statusDistribution}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(0)}%`}
                      outerRadius={100}
                      fill="#8884d8"
                      dataKey="value"
                    >
                      {statusDistribution.map((entry, index) => (
                        <DynamicCell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                      ))}
                    </DynamicPie>
                    <Tooltip />
                    <Legend />
                  </DynamicPieChart>
                </ResponsiveContainer>
              </div>
            ) : (
              <div className="text-center py-8 text-muted-foreground">
                <BarChart3 className="h-12 w-12 mx-auto mb-4 opacity-50" />
                <p>Нет данных о статусах</p>
              </div>
            )}
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

