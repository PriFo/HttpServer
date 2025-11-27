'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import {
  Package,
  Building2,
  Database,
  TrendingUp,
  TrendingDown,
  AlertCircle,
} from 'lucide-react'
import { NormalizationType, PreviewStatsResponse } from '@/types/normalization'
import { cn } from '@/lib/utils'
import { motion } from 'framer-motion'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { Skeleton } from '@/components/ui/skeleton'

interface HeatmapCell {
  databaseName: string
  databaseId: number
  nomenclatureQuality: number
  counterpartyQuality: number
  overallQuality: number
  recordCount: number
}

interface DataQualityHeatmapProps {
  stats?: PreviewStatsResponse | null
  normalizationType: NormalizationType
  isLoading?: boolean
  className?: string
}

export function DataQualityHeatmap({
  stats,
  normalizationType,
  isLoading = false,
  className,
}: DataQualityHeatmapProps) {
  const heatmapData = useMemo<HeatmapCell[]>(() => {
    if (!stats?.databases || !Array.isArray(stats.databases)) return []

    return stats.databases
      .filter(db => db && db.database_id && db.database_name) // Фильтруем некорректные данные
      .map((db) => {
        // Вычисляем качество данных на основе заполненности
        const nomenclatureQuality = db.completeness?.nomenclature_completeness?.overall_completeness || 0
        const counterpartyQuality = db.completeness?.counterparty_completeness?.overall_completeness || 0
        
        // Общее качество - среднее взвешенное по количеству записей
        const totalRecords = (db.nomenclature_count || 0) + (db.counterparty_count || 0)
        const overallQuality = totalRecords > 0
          ? (nomenclatureQuality * (db.nomenclature_count || 0) + counterpartyQuality * (db.counterparty_count || 0)) / totalRecords
          : 0

        return {
          databaseName: db.database_name || 'Неизвестная БД',
          databaseId: db.database_id,
          nomenclatureQuality,
          counterpartyQuality,
          overallQuality: Math.max(0, Math.min(100, overallQuality)), // Ограничиваем диапазон 0-100
          recordCount: totalRecords,
        }
      })
      .sort((a, b) => b.overallQuality - a.overallQuality)
  }, [stats])

  const getQualityColor = (quality: number) => {
    if (quality >= 90) return 'bg-green-500 hover:bg-green-600'
    if (quality >= 70) return 'bg-yellow-500 hover:bg-yellow-600'
    if (quality >= 50) return 'bg-orange-500 hover:bg-orange-600'
    return 'bg-red-500 hover:bg-red-600'
  }

  const getQualityTextColor = (quality: number) => {
    if (quality >= 90) return 'text-green-700 dark:text-green-300'
    if (quality >= 70) return 'text-yellow-700 dark:text-yellow-300'
    if (quality >= 50) return 'text-orange-700 dark:text-orange-300'
    return 'text-red-700 dark:text-red-300'
  }

  const getQualityLabel = (quality: number) => {
    if (quality >= 90) return 'Отлично'
    if (quality >= 70) return 'Хорошо'
    if (quality >= 50) return 'Удовлетворительно'
    return 'Требует внимания'
  }

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>Тепловая карта качества данных</CardTitle>
          <CardDescription>Загрузка данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {[...Array(8)].map((_, i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!stats || heatmapData.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground">
            <Database className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>Нет данных для отображения</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  const showNomenclature = normalizationType === 'nomenclature' || normalizationType === 'both'
  const showCounterparties = normalizationType === 'counterparties' || normalizationType === 'both'

  return (
    <Card className={cn('bg-gradient-to-br from-purple-50/50 to-pink-50/50 dark:from-purple-950/20 dark:to-pink-950/20 border-purple-200/50 shadow-lg', className)}>
      <CardHeader>
        <CardTitle className="text-lg font-semibold flex items-center gap-2">
          <TrendingUp className="h-5 w-5 text-purple-600" />
          Тепловая карта качества данных
        </CardTitle>
        <CardDescription>
          Визуализация качества данных по базам данных
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-6">
          {/* Легенда */}
          <div className="flex items-center gap-4 flex-wrap">
            <span className="text-sm font-medium">Качество:</span>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-green-500 rounded" />
              <span className="text-xs">≥90%</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-yellow-500 rounded" />
              <span className="text-xs">70-89%</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-orange-500 rounded" />
              <span className="text-xs">50-69%</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-4 h-4 bg-red-500 rounded" />
              <span className="text-xs">&lt;50%</span>
            </div>
          </div>

          {/* Тепловая карта */}
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
            <TooltipProvider>
              {heatmapData.map((cell, index) => (
                <motion.div
                  key={cell.databaseId}
                  initial={{ opacity: 0, scale: 0.9 }}
                  animate={{ opacity: 1, scale: 1 }}
                  transition={{ duration: 0.3, delay: index * 0.05 }}
                >
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Card className={cn(
                        'cursor-pointer transition-all hover:shadow-lg',
                        getQualityColor(cell.overallQuality)
                      )}>
                        <CardContent className="pt-4">
                          <div className="space-y-3">
                            <div className="flex items-center justify-between">
                              <span className="text-sm font-semibold text-white truncate flex-1">
                                {cell.databaseName}
                              </span>
                              <Badge variant="secondary" className="text-xs bg-white/20 text-white border-white/30">
                                {cell.recordCount.toLocaleString('ru-RU')}
                              </Badge>
                            </div>
                            <div className="space-y-2">
                              {showNomenclature && (
                                <div className="flex items-center justify-between text-xs text-white/90">
                                  <span className="flex items-center gap-1">
                                    <Package className="h-3 w-3" />
                                    Номенклатура
                                  </span>
                                  <span className="font-semibold">
                                    {cell.nomenclatureQuality.toFixed(1)}%
                                  </span>
                                </div>
                              )}
                              {showCounterparties && (
                                <div className="flex items-center justify-between text-xs text-white/90">
                                  <span className="flex items-center gap-1">
                                    <Building2 className="h-3 w-3" />
                                    Контрагенты
                                  </span>
                                  <span className="font-semibold">
                                    {cell.counterpartyQuality.toFixed(1)}%
                                  </span>
                                </div>
                              )}
                            </div>
                            <div className="pt-2 border-t border-white/20">
                              <div className="flex items-center justify-between">
                                <span className="text-xs text-white/80">Общее качество</span>
                                <span className={cn('text-lg font-bold', getQualityTextColor(cell.overallQuality))}>
                                  {cell.overallQuality.toFixed(1)}%
                                </span>
                              </div>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    </TooltipTrigger>
                    <TooltipContent className="max-w-xs">
                      <div className="space-y-2">
                        <p className="font-semibold">{cell.databaseName}</p>
                        <div className="text-sm space-y-1">
                          {showNomenclature && (
                            <p>Номенклатура: {cell.nomenclatureQuality.toFixed(1)}%</p>
                          )}
                          {showCounterparties && (
                            <p>Контрагенты: {cell.counterpartyQuality.toFixed(1)}%</p>
                          )}
                          <p>Общее качество: {cell.overallQuality.toFixed(1)}%</p>
                          <p>Записей: {cell.recordCount.toLocaleString('ru-RU')}</p>
                          <p className="text-xs text-muted-foreground mt-2">
                            Статус: {getQualityLabel(cell.overallQuality)}
                          </p>
                        </div>
                      </div>
                    </TooltipContent>
                  </Tooltip>
                </motion.div>
              ))}
            </TooltipProvider>
          </div>

          {/* Статистика */}
          {heatmapData.length > 0 && (
            <div className="grid grid-cols-2 md:grid-cols-4 gap-4 pt-4 border-t">
              <div className="text-center p-3 bg-background/50 rounded-lg">
                <div className="text-2xl font-bold text-green-600">
                  {heatmapData.filter(c => c.overallQuality >= 90).length}
                </div>
                <div className="text-xs text-muted-foreground mt-1">Отличное качество</div>
              </div>
              <div className="text-center p-3 bg-background/50 rounded-lg">
                <div className="text-2xl font-bold text-yellow-600">
                  {heatmapData.filter(c => c.overallQuality >= 70 && c.overallQuality < 90).length}
                </div>
                <div className="text-xs text-muted-foreground mt-1">Хорошее качество</div>
              </div>
              <div className="text-center p-3 bg-background/50 rounded-lg">
                <div className="text-2xl font-bold text-orange-600">
                  {heatmapData.filter(c => c.overallQuality >= 50 && c.overallQuality < 70).length}
                </div>
                <div className="text-xs text-muted-foreground mt-1">Удовлетворительно</div>
              </div>
              <div className="text-center p-3 bg-background/50 rounded-lg">
                <div className="text-2xl font-bold text-red-600">
                  {heatmapData.filter(c => c.overallQuality < 50).length}
                </div>
                <div className="text-xs text-muted-foreground mt-1">Требует внимания</div>
              </div>
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

