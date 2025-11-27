'use client'

import { useMemo } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Skeleton } from '@/components/ui/skeleton'
import {
  DynamicPieChart,
  DynamicPie,
  DynamicCell,
  DynamicBarChart,
  DynamicBar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from '@/lib/recharts-dynamic'
import { Package, Building2, Database, Copy, AlertCircle, CheckCircle2 } from 'lucide-react'
import { PreviewStatsResponse, NormalizationType } from '@/types/normalization'

interface NormalizationAnalyticsChartsProps {
  stats: PreviewStatsResponse | null
  isLoading?: boolean
  normalizationType?: NormalizationType
}

const COLORS = {
  nomenclature: '#3b82f6',
  counterparties: '#10b981',
  ready: '#10b981',
  problematic: '#ef4444',
  small: '#3b82f6',
  medium: '#f59e0b',
  large: '#ef4444',
}

export function NormalizationAnalyticsCharts({ stats, isLoading = false, normalizationType = 'both' }: NormalizationAnalyticsChartsProps) {
  // Данные для круговой диаграммы: Номенклатура vs Контрагенты
  const recordsDistribution = useMemo(() => {
    if (!stats || stats.total_records === 0) return null

    return [
      {
        name: 'Номенклатура',
        value: stats.total_nomenclature,
        percentage: (stats.total_nomenclature / stats.total_records) * 100,
        color: COLORS.nomenclature,
      },
      {
        name: 'Контрагенты',
        value: stats.total_counterparties,
        percentage: (stats.total_counterparties / stats.total_records) * 100,
        color: COLORS.counterparties,
      },
    ]
  }, [stats])

  // Функция для определения размера БД
  const getDatabaseSizeCategory = (size: number): 'small' | 'medium' | 'large' => {
    if (size < 10 * 1024 * 1024) return 'small' // до 10 МБ
    if (size < 100 * 1024 * 1024) return 'medium' // 10-100 МБ
    return 'large' // свыше 100 МБ
  }

  // Функция для определения категории по количеству записей
  const getRecordsCategory = (records: number): 'small' | 'medium' | 'large' => {
    if (records < 10000) return 'small' // до 10,000 записей
    if (records < 100000) return 'medium' // 10,000 - 100,000 записей
    return 'large' // свыше 100,000 записей
  }

  // Данные для bar chart: Распределение БД по размерам
  const sizeDistribution = useMemo(() => {
    if (!stats || !stats.databases || stats.databases.length === 0) return null

    const distribution = {
      small: 0,
      medium: 0,
      large: 0,
    }

    stats.databases.forEach(db => {
      const category = getDatabaseSizeCategory(db.database_size)
      distribution[category]++
    })

    return [
      { name: 'Малые (<10 МБ)', value: distribution.small, color: COLORS.small },
      { name: 'Средние (10-100 МБ)', value: distribution.medium, color: COLORS.medium },
      { name: 'Большие (>100 МБ)', value: distribution.large, color: COLORS.large },
    ]
  }, [stats])

  // Данные для bar chart: Распределение БД по количеству записей
  const recordsDistributionByDB = useMemo(() => {
    if (!stats || !stats.databases || stats.databases.length === 0) return null

    const distribution = {
      small: 0,
      medium: 0,
      large: 0,
    }

    stats.databases.forEach(db => {
      const category = getRecordsCategory(db.total_records)
      distribution[category]++
    })

    return [
      { name: 'Малые (<10K)', value: distribution.small, color: COLORS.small },
      { name: 'Средние (10K-100K)', value: distribution.medium, color: COLORS.medium },
      { name: 'Большие (>100K)', value: distribution.large, color: COLORS.large },
    ]
  }, [stats])

  // Статус готовности БД
  const readinessStats = useMemo(() => {
    if (!stats || !stats.databases || stats.databases.length === 0) return null

    const ready = stats.databases.filter(
      db => db.is_valid === true && db.is_accessible === true && !db.error
    ).length
    const problematic = stats.databases.length - ready

    return {
      ready,
      problematic,
      total: stats.databases.length,
      readyPercentage: (ready / stats.databases.length) * 100,
    }
  }, [stats])

  // Данные для мини-графика дубликатов
  const duplicatesData = useMemo(() => {
    if (!stats || stats.total_records === 0) return null

    const duplicatesPercentage = (stats.estimated_duplicates / stats.total_records) * 100
    const uniqueRecords = stats.total_records - stats.estimated_duplicates

    return [
      {
        name: 'Уникальные',
        value: uniqueRecords,
        percentage: (uniqueRecords / stats.total_records) * 100,
        color: COLORS.ready,
      },
      {
        name: 'Дубликаты',
        value: stats.estimated_duplicates,
        percentage: duplicatesPercentage,
        color: COLORS.problematic,
      },
    ]
  }, [stats])

  if (isLoading) {
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Распределение записей</CardTitle>
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[250px] w-full" />
          </CardContent>
        </Card>
        <Card>
          <CardHeader>
            <CardTitle className="text-sm">Распределение по размерам</CardTitle>
          </CardHeader>
          <CardContent>
            <Skeleton className="h-[250px] w-full" />
          </CardContent>
        </Card>
      </div>
    )
  }

  if (!stats) {
    return null
  }

  const showNomenclature = normalizationType === 'nomenclature' || normalizationType === 'both'
  const showCounterparties = normalizationType === 'counterparties' || normalizationType === 'both'

  return (
    <div className="space-y-6">
      {/* Распределение записей: Номенклатура vs Контрагенты */}
      {recordsDistribution && (showNomenclature || showCounterparties) && (
        <Card className="backdrop-blur-sm bg-gradient-to-br from-blue-50/50 to-green-50/50 dark:from-blue-950/20 dark:to-green-950/20 border-blue-200/50 dark:border-blue-800/50 shadow-lg">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Package className="h-4 w-4 text-blue-600" />
              Распределение записей
            </CardTitle>
            <CardDescription className="text-xs">
              Номенклатура и контрагенты в общем объеме данных
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Круговая диаграмма */}
              <div>
                <ResponsiveContainer width="100%" height={250}>
                  <DynamicPieChart>
                    <DynamicPie
                      data={recordsDistribution}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(1)}%`}
                      outerRadius={80}
                      fill="#8884d8"
                      dataKey="value"
                    >
                      {recordsDistribution.map((entry, index) => (
                        <DynamicCell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </DynamicPie>
                    <Tooltip
                      formatter={(value: number, name: string, props: any) => [
                        `${value.toLocaleString()} (${props.payload.percentage.toFixed(1)}%)`,
                        name,
                      ]}
                    />
                    <Legend />
                  </DynamicPieChart>
                </ResponsiveContainer>
              </div>
              {/* Текстовые метрики */}
              <div className="flex flex-col justify-center space-y-4">
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Package className="h-4 w-4 text-blue-600" />
                      <span className="text-sm font-medium">Номенклатура</span>
                    </div>
                    <div className="text-right">
                      <div className="text-lg font-bold text-blue-600">
                        {stats.total_nomenclature.toLocaleString()}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {recordsDistribution[0].percentage.toFixed(1)}%
                      </div>
                    </div>
                  </div>
                  <Progress value={recordsDistribution[0].percentage} className="h-2" />
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <Building2 className="h-4 w-4 text-green-600" />
                      <span className="text-sm font-medium">Контрагенты</span>
                    </div>
                    <div className="text-right">
                      <div className="text-lg font-bold text-green-600">
                        {stats.total_counterparties.toLocaleString()}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {recordsDistribution[1].percentage.toFixed(1)}%
                      </div>
                    </div>
                  </div>
                  <Progress value={recordsDistribution[1].percentage} className="h-2" />
                </div>
                <div className="pt-2 border-t">
                  <div className="flex items-center justify-between text-sm">
                    <span className="text-muted-foreground">Всего записей:</span>
                    <span className="font-bold">{stats.total_records.toLocaleString()}</span>
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Распределение БД по размерам и количеству записей */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
        {/* Распределение по размерам */}
        {sizeDistribution && (
          <Card className="backdrop-blur-sm bg-card/95 border-border/50 shadow-md">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Database className="h-4 w-4" />
                Распределение по размерам БД
              </CardTitle>
              <CardDescription className="text-xs">
                Классификация баз данных по размеру файла
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={200}>
                <DynamicBarChart data={sizeDistribution}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <DynamicBar dataKey="value" fill="#3b82f6" radius={[8, 8, 0, 0]}>
                    {sizeDistribution.map((entry, index) => (
                      <DynamicCell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </DynamicBar>
                </DynamicBarChart>
              </ResponsiveContainer>
              <div className="mt-4 grid grid-cols-3 gap-2 text-center">
                {sizeDistribution.map((item, index) => (
                  <div key={index} className="space-y-1">
                    <div className="text-2xl font-bold" style={{ color: item.color }}>
                      {item.value}
                    </div>
                    <div className="text-xs text-muted-foreground">{item.name.split(' ')[0]}</div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}

        {/* Распределение по количеству записей */}
        {recordsDistributionByDB && (
          <Card className="backdrop-blur-sm bg-card/95 border-border/50 shadow-md">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Copy className="h-4 w-4" />
                Распределение по объему записей
              </CardTitle>
              <CardDescription className="text-xs">
                Классификация баз данных по количеству записей
              </CardDescription>
            </CardHeader>
            <CardContent>
              <ResponsiveContainer width="100%" height={200}>
                <DynamicBarChart data={recordsDistributionByDB}>
                  <CartesianGrid strokeDasharray="3 3" />
                  <XAxis dataKey="name" tick={{ fontSize: 12 }} />
                  <YAxis />
                  <Tooltip />
                  <Legend />
                  <DynamicBar dataKey="value" fill="#10b981" radius={[8, 8, 0, 0]}>
                    {recordsDistributionByDB.map((entry, index) => (
                      <DynamicCell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </DynamicBar>
                </DynamicBarChart>
              </ResponsiveContainer>
              <div className="mt-4 grid grid-cols-3 gap-2 text-center">
                {recordsDistributionByDB.map((item, index) => (
                  <div key={index} className="space-y-1">
                    <div className="text-2xl font-bold" style={{ color: item.color }}>
                      {item.value}
                    </div>
                    <div className="text-xs text-muted-foreground">{item.name.split(' ')[0]}</div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Статус готовности БД */}
      {readinessStats && (
        <Card className="backdrop-blur-sm bg-gradient-to-br from-green-50/50 to-red-50/50 dark:from-green-950/20 dark:to-red-950/20 border-green-200/50 dark:border-green-800/50 shadow-lg">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <CheckCircle2 className="h-4 w-4 text-green-600" />
              Статус готовности баз данных
            </CardTitle>
            <CardDescription className="text-xs">
              Количество готовых и проблемных баз данных
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <CheckCircle2 className="h-4 w-4 text-green-600" />
                    <span className="text-sm font-medium">Готовы к обработке</span>
                  </div>
                  <div className="text-right">
                    <div className="text-lg font-bold text-green-600">
                      {readinessStats.ready} / {readinessStats.total}
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {readinessStats.readyPercentage.toFixed(1)}%
                    </div>
                  </div>
                </div>
                <Progress value={readinessStats.readyPercentage} className="h-2" />
              </div>
              {readinessStats.problematic > 0 && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <AlertCircle className="h-4 w-4 text-red-600" />
                      <span className="text-sm font-medium">Проблемные</span>
                    </div>
                    <div className="text-right">
                      <div className="text-lg font-bold text-red-600">
                        {readinessStats.problematic}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {((readinessStats.problematic / readinessStats.total) * 100).toFixed(1)}%
                      </div>
                    </div>
                  </div>
                  <Progress
                    value={(readinessStats.problematic / readinessStats.total) * 100}
                    className="h-2"
                  />
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Мини-график дубликатов */}
      {duplicatesData && stats.estimated_duplicates > 0 && (
        <Card className="backdrop-blur-sm bg-gradient-to-br from-orange-50/50 to-yellow-50/50 dark:from-orange-950/20 dark:to-yellow-950/20 border-orange-200/50 dark:border-orange-800/50 shadow-lg">
          <CardHeader className="pb-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <Copy className="h-4 w-4 text-orange-600" />
              Анализ дубликатов
            </CardTitle>
            <CardDescription className="text-xs">
              Потенциальные дубликаты в данных
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {/* Круговая диаграмма */}
              <div>
                <ResponsiveContainer width="100%" height={200}>
                  <DynamicPieChart>
                    <DynamicPie
                      data={duplicatesData}
                      cx="50%"
                      cy="50%"
                      labelLine={false}
                      label={({ name, percent }) => `${name}: ${((percent || 0) * 100).toFixed(1)}%`}
                      outerRadius={70}
                      fill="#8884d8"
                      dataKey="value"
                    >
                      {duplicatesData.map((entry, index) => (
                        <DynamicCell key={`cell-${index}`} fill={entry.color} />
                      ))}
                    </DynamicPie>
                    <Tooltip
                      formatter={(value: number, name: string, props: any) => [
                        `${value.toLocaleString()} (${props.payload.percentage.toFixed(1)}%)`,
                        name,
                      ]}
                    />
                    <Legend />
                  </DynamicPieChart>
                </ResponsiveContainer>
              </div>
              {/* Текстовые метрики */}
              <div className="flex flex-col justify-center space-y-3">
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium">Потенциальные дубликаты:</span>
                    <span className="text-lg font-bold text-orange-600">
                      {stats.estimated_duplicates.toLocaleString()}
                    </span>
                  </div>
                  <div className="text-xs text-muted-foreground">
                    {duplicatesData[1].percentage.toFixed(2)}% от общего объема
                  </div>
                </div>
                {stats.duplicate_groups && stats.duplicate_groups > 0 && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <span className="text-sm font-medium">Групп дубликатов:</span>
                      <span className="text-lg font-bold">
                        {stats.duplicate_groups.toLocaleString()}
                      </span>
                    </div>
                  </div>
                )}
                <div className="pt-2 border-t">
                  <div className="text-xs text-muted-foreground">
                    Уникальных записей: {duplicatesData[0].value.toLocaleString()}
                  </div>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

