'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { BarChart3, Target, Award, Zap, ArrowUpRight, ArrowDownRight, Clock } from 'lucide-react'
import { StatCard } from '@/components/common/stat-card'
import { DynamicPieChart, DynamicPie, DynamicCell, DynamicBarChart, DynamicBar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer, Cell } from '@/lib/recharts-dynamic'
import { Skeleton } from "@/components/ui/skeleton"
import { normalizePercentage } from '@/lib/locale'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'
import { EmptyState } from '@/components/common/empty-state'

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
  last_activity?: string
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

const normalizeQuality = normalizePercentage

const formatRelativeTime = (value?: string) => {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null
  return formatDistanceToNow(date, { addSuffix: true, locale: ru })
}

const formatAbsoluteTime = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('ru-RU')
}

interface QualityOverviewTabProps {
  stats: QualityStats | null
  loading: boolean
}

export function QualityOverviewTab({ stats, loading }: QualityOverviewTabProps) {
  if (loading) {
    return <QualityOverviewSkeleton />
  }

  if (!stats) {
    return (
      <EmptyState
        title="Нет данных для отображения"
        description="Выберите проект или базу данных, чтобы увидеть показатели качества."
      />
    )
  }

  // Безопасная инициализация by_level с проверкой типа
  const byLevel = (stats?.by_level && typeof stats.by_level === 'object' && !Array.isArray(stats.by_level))
    ? stats.by_level
    : {}

  const pieData = Object.entries(byLevel).map(([level, data]) => ({
    name: LEVEL_NAMES[level] || level,
    value: data.count || 0,
    percentage: isNaN(data.percentage) ? 0 : data.percentage,
  }))

  const barData = Object.entries(byLevel).map(([level, data]) => ({
    name: LEVEL_NAMES[level] || level,
    quality: normalizeQuality(data.avg_quality || 0).toFixed(1),
    count: data.count || 0,
  }))

  return (
    <div className="space-y-6">
      {/* Overview Cards */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 xl:grid-cols-5 gap-6">
        <StatCard
          title="Всего записей"
          value={stats.total_items}
          description="Обработано"
          icon={BarChart3}
          formatValue={(val) => val.toLocaleString('ru-RU')}
        />

        <StatCard
          title="Средняя оценка"
          value={`${normalizeQuality(stats.average_quality || 0).toFixed(1)}%`}
          icon={Target}
          progress={normalizeQuality(stats.average_quality || 0)}
        />

        <StatCard
          title="Эталонное качество"
          value={stats.benchmark_count || 0}
          description={`${(isNaN(stats.benchmark_percentage) ? 0 : stats.benchmark_percentage).toFixed(1)}% от общего числа`}
          icon={Award}
          variant="success"
          formatValue={(val) => val.toLocaleString('ru-RU')}
        />

        <StatCard
          title="AI обработано"
          value={byLevel['ai_enhanced']?.count || 0}
          description={`${(isNaN(byLevel['ai_enhanced']?.percentage) ? 0 : (byLevel['ai_enhanced']?.percentage || 0)).toFixed(1)}% записей`}
          icon={Zap}
          variant="primary"
          formatValue={(val) => val.toLocaleString('ru-RU')}
        />
        {stats.last_activity && (
          <StatCard
            title="Последняя активность"
            value={formatRelativeTime(stats.last_activity) || '—'}
            description={formatAbsoluteTime(stats.last_activity)}
            icon={Clock}
          />
        )}
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle>Распределение по уровням</CardTitle>
            <CardDescription>
              Количество записей на каждом уровне обработки
            </CardDescription>
          </CardHeader>
          <CardContent className="h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
                <DynamicPieChart>
                <DynamicPie
                  data={pieData}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={80}
                  paddingAngle={5}
                  dataKey="value"
                >
                  {pieData.map((entry, index) => {
                    const level = Object.keys(LEVEL_NAMES).find(
                      (k) => LEVEL_NAMES[k] === entry.name
                    ) || ''
                    return <DynamicCell key={`cell-${index}`} fill={LEVEL_COLORS[level]} strokeWidth={0} />
                  })}
                </DynamicPie>
                <Tooltip 
                  formatter={(value: unknown) => {
                    const numeric = typeof value === 'number' ? value : Number(value)
                    return [Number.isNaN(numeric) ? '0' : numeric.toLocaleString('ru-RU'), 'Записей']
                  }}
                  contentStyle={{ borderRadius: 'var(--radius)', border: '1px solid var(--border)' }}
                />
                <Legend verticalAlign="bottom" height={36} iconType="circle" />
              </DynamicPieChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Средняя оценка по уровням</CardTitle>
            <CardDescription>
              Средний показатель качества для каждого уровня
            </CardDescription>
          </CardHeader>
          <CardContent className="h-[300px]">
            <ResponsiveContainer width="100%" height="100%">
              <DynamicBarChart data={barData} margin={{ top: 20, right: 30, left: 20, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" vertical={false} opacity={0.3} />
                <XAxis 
                  dataKey="name" 
                  axisLine={false} 
                  tickLine={false} 
                  tick={{ fontSize: 12 }} 
                  dy={10}
                />
                <YAxis 
                  domain={[0, 100]} 
                  axisLine={false} 
                  tickLine={false} 
                  tickFormatter={(value) => `${value}%`}
                  tick={{ fontSize: 12 }} 
                />
                <Tooltip 
                  cursor={{ fill: 'transparent' }}
                  contentStyle={{ borderRadius: 'var(--radius)', border: '1px solid var(--border)' }}
                  formatter={(value: unknown) => {
                    const numeric = typeof value === 'number' ? value : Number(value)
                    const display = Number.isNaN(numeric) ? '0' : numeric.toString()
                    return [`${display}%`, 'Качество']
                  }} 
                />
                <DynamicBar 
                  dataKey="quality" 
                  fill="#3b82f6" 
                  radius={[4, 4, 0, 0]} 
                  barSize={40}
                >
                  {barData.map((entry, index) => {
                    const level = Object.keys(LEVEL_NAMES).find(
                      (k) => LEVEL_NAMES[k] === entry.name
                    ) || ''
                    return <Cell key={`cell-${index}`} fill={LEVEL_COLORS[level]} />
                  })}
                </DynamicBar>
              </DynamicBarChart>
            </ResponsiveContainer>
          </CardContent>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Level Details Table */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>Детализация по уровням</CardTitle>
            <CardDescription>
              Подробная информация о распределении и качестве
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              {Object.entries(byLevel).map(([level, data]: [string, LevelStat]) => (
                <div key={level} className="flex items-center justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors">
                  <div className="flex items-center gap-4">
                    <div className="w-3 h-3 rounded-full shadow-sm" style={{ backgroundColor: LEVEL_COLORS[level] }}></div>
                    <div>
                      <p className="font-medium">{LEVEL_NAMES[level] || level}</p>
                      <p className="text-sm text-muted-foreground">
                        {data.count.toLocaleString('ru-RU')} записей
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-6">
                    <div className="text-right">
                        <span className="text-xs text-muted-foreground block">Доля</span>
                        <span className="font-medium">{(isNaN(data.percentage) ? 0 : data.percentage).toFixed(1)}%</span>
                    </div>
                    <div className="text-right w-24">
                      <span className="text-xs text-muted-foreground block mb-1">Качество</span>
                      <Badge variant="outline" className="w-full justify-center bg-background">
                        {normalizeQuality(data.avg_quality || 0).toFixed(1)}%
                      </Badge>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Benchmark Quality Section */}
        <Card className="bg-gradient-to-br from-card to-muted/30">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Award className="h-5 w-5 text-primary" />
              Эталонное качество
            </CardTitle>
            <CardDescription>
              Записи с оценкой качества ≥ 90%
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-6">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Прогресс</span>
                  <span className="font-bold text-primary">{(isNaN(stats.benchmark_percentage) ? 0 : stats.benchmark_percentage).toFixed(1)}%</span>
                </div>
                <Progress value={isNaN(stats.benchmark_percentage) ? 0 : stats.benchmark_percentage} className="h-2" />
              </div>
              
              <div className="grid grid-cols-1 gap-4">
                <div className="p-4 bg-background/50 border rounded-lg space-y-1">
                  <div className="flex items-center gap-2 text-green-600">
                    <ArrowUpRight className="h-4 w-4" />
                    <span className="text-sm font-medium">Достигли эталона</span>
                  </div>
                  <p className="text-2xl font-bold">{stats.benchmark_count.toLocaleString('ru-RU')}</p>
                  <p className="text-xs text-muted-foreground">записей соответствуют стандартам</p>
                </div>
                
                <div className="p-4 bg-background/50 border rounded-lg space-y-1">
                  <div className="flex items-center gap-2 text-amber-600">
                    <ArrowDownRight className="h-4 w-4" />
                    <span className="text-sm font-medium">Требуют улучшения</span>
                  </div>
                  <p className="text-2xl font-bold">{(stats.total_items - stats.benchmark_count).toLocaleString('ru-RU')}</p>
                  <p className="text-xs text-muted-foreground">записей нуждаются в доработке</p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function QualityOverviewSkeleton() {
  return (
    <div className="space-y-6">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {[...Array(4)].map((_, i) => (
          <Card key={i}>
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <Skeleton className="h-4 w-[100px]" />
              <Skeleton className="h-4 w-4 rounded-full" />
            </CardHeader>
            <CardContent>
              <Skeleton className="h-8 w-[60px] mb-2" />
              <Skeleton className="h-3 w-[120px]" />
            </CardContent>
          </Card>
        ))}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-[200px] mb-2" />
            <Skeleton className="h-4 w-[300px]" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[300px] w-full rounded-lg" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <Skeleton className="h-6 w-[200px] mb-2" />
            <Skeleton className="h-4 w-[300px]" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[300px] w-full rounded-lg" />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
