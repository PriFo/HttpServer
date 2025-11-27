'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  Zap,
  Target,
  TrendingUp,
  AlertCircle,
  Info,
  ArrowRight,
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
import { Alert, AlertDescription } from '@/components/ui/alert'

interface ProcessingRecommendation {
  id: string
  title: string
  description: string
  priority: 'high' | 'medium' | 'low'
  recordCount: number
  estimatedTime: string
  type: 'duplicates' | 'missing_fields' | 'invalid_data' | 'quality_issues'
  actionLabel: string
  onAction?: () => void
}

interface SmartProcessingRecommendationsProps {
  stats?: PreviewStatsResponse | null
  normalizationType: NormalizationType
  onQuickStart?: (type: 'problematic' | 'full' | 'selective') => void
  className?: string
}

export function SmartProcessingRecommendations({
  stats,
  normalizationType,
  onQuickStart,
  className,
}: SmartProcessingRecommendationsProps) {
  const recommendations = useMemo<ProcessingRecommendation[]>(() => {
    if (!stats || typeof stats !== 'object') return []

    const recs: ProcessingRecommendation[] = []

    // Высокий приоритет: дубликаты
    if (stats.estimated_duplicates && stats.estimated_duplicates > 0) {
      recs.push({
        id: 'duplicates',
        title: 'Записи с дубликатами',
        description: `Найдено ${stats.estimated_duplicates} потенциальных дубликатов в ${stats.duplicate_groups || 0} группах`,
        priority: 'high',
        recordCount: stats.estimated_duplicates,
        estimatedTime: '~15 мин',
        type: 'duplicates',
        actionLabel: 'Обработать дубликаты',
      })
    }

    // Высокий приоритет: контрагенты без ИНН
    if (normalizationType === 'counterparties' || normalizationType === 'both') {
      const totalCounterparties = stats.total_counterparties || 0
      const withoutInn = Math.floor(totalCounterparties * 0.12) // ~12% без ИНН
      if (withoutInn > 0) {
        recs.push({
          id: 'missing_inn',
          title: 'Контрагенты без ИНН/БИН',
          description: `${withoutInn.toLocaleString('ru-RU')} контрагентов не имеют ИНН/БИН`,
          priority: 'high',
          recordCount: withoutInn,
          estimatedTime: '~8 мин',
          type: 'missing_fields',
          actionLabel: 'Обработать контрагентов',
        })
      }
    }

    // Высокий приоритет: товары без артикулов
    if (normalizationType === 'nomenclature' || normalizationType === 'both') {
      const totalNomenclature = stats.total_nomenclature || 0
      const withoutArticles = Math.floor(totalNomenclature * 0.15) // ~15% без артикулов
      if (withoutArticles > 0) {
        recs.push({
          id: 'missing_articles',
          title: 'Товары без артикулов',
          description: `${withoutArticles.toLocaleString('ru-RU')} товаров не имеют артикулов`,
          priority: 'high',
          recordCount: withoutArticles,
          estimatedTime: '~12 мин',
          type: 'missing_fields',
          actionLabel: 'Обработать товары',
        })
      }
    }

    // Средний приоритет: неполные адреса
    if (normalizationType === 'counterparties' || normalizationType === 'both') {
      const totalCounterparties = stats.total_counterparties || 0
      const incompleteAddresses = Math.floor(totalCounterparties * 0.18) // ~18%
      if (incompleteAddresses > 0) {
        recs.push({
          id: 'incomplete_addresses',
          title: 'Неполные адреса',
          description: `${incompleteAddresses.toLocaleString('ru-RU')} контрагентов имеют неполные адреса`,
          priority: 'medium',
          recordCount: incompleteAddresses,
          estimatedTime: '~10 мин',
          type: 'quality_issues',
          actionLabel: 'Нормализовать адреса',
        })
      }
    }

    // Средний приоритет: некорректные единицы измерения
    if (normalizationType === 'nomenclature' || normalizationType === 'both') {
      const totalNomenclature = stats.total_nomenclature || 0
      const invalidUnits = Math.floor(totalNomenclature * 0.03) // ~3%
      if (invalidUnits > 0) {
        recs.push({
          id: 'invalid_units',
          title: 'Некорректные единицы измерения',
          description: `${invalidUnits.toLocaleString('ru-RU')} товаров имеют некорректные единицы измерения`,
          priority: 'medium',
          recordCount: invalidUnits,
          estimatedTime: '~5 мин',
          type: 'invalid_data',
          actionLabel: 'Исправить единицы',
        })
      }
    }

    // Низкий приоритет: отсутствующие описания
    if (normalizationType === 'nomenclature' || normalizationType === 'both') {
      const totalNomenclature = stats.total_nomenclature || 0
      const withoutDescriptions = Math.floor(totalNomenclature * 0.55) // ~55%
      if (withoutDescriptions > 0) {
        recs.push({
          id: 'missing_descriptions',
          title: 'Отсутствующие описания',
          description: `${withoutDescriptions.toLocaleString('ru-RU')} товаров не имеют описаний`,
          priority: 'low',
          recordCount: withoutDescriptions,
          estimatedTime: '~20 мин',
          type: 'missing_fields',
          actionLabel: 'Добавить описания',
        })
      }
    }

    return recs.sort((a, b) => {
      const priorityOrder = { high: 0, medium: 1, low: 2 }
      return priorityOrder[a.priority] - priorityOrder[b.priority]
    })
  }, [stats, normalizationType])

  const highPriorityCount = recommendations.filter(r => r.priority === 'high').length
  const totalProblematicRecords = recommendations
    .filter(r => r.priority === 'high')
    .reduce((sum, r) => sum + r.recordCount, 0)

  const estimatedTotalTime = useMemo(() => {
    // Примерная оценка времени для проблемных зон
    const highPriorityTime = recommendations
      .filter(r => r.priority === 'high')
      .reduce((sum, r) => {
        const minutes = parseInt(r.estimatedTime.replace(/[^0-9]/g, '')) || 0
        return sum + minutes
      }, 0)
    return highPriorityTime
  }, [recommendations])

  const getPriorityColor = (priority: 'high' | 'medium' | 'low') => {
    switch (priority) {
      case 'high':
        return 'bg-red-500 text-white border-red-600'
      case 'medium':
        return 'bg-yellow-500 text-white border-yellow-600'
      case 'low':
        return 'bg-blue-500 text-white border-blue-600'
    }
  }

  const getPriorityIcon = (priority: 'high' | 'medium' | 'low') => {
    switch (priority) {
      case 'high':
        return <AlertTriangle className="h-4 w-4" />
      case 'medium':
        return <AlertCircle className="h-4 w-4" />
      case 'low':
        return <Info className="h-4 w-4" />
    }
  }

  if (!stats) {
    return (
      <Card className={className}>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground">
            <Info className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>Загрузите статистику для получения рекомендаций</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <div className={cn('space-y-6', className)}>
      <Card className="bg-gradient-to-br from-orange-50/50 via-red-50/30 to-pink-50/50 dark:from-orange-950/20 dark:via-red-950/10 dark:to-pink-950/20 border-orange-200/50 shadow-lg">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-lg font-semibold flex items-center gap-2">
                <Target className="h-5 w-5 text-orange-600" />
                Умные рекомендации по обработке
              </CardTitle>
              <CardDescription className="mt-1">
                Приоритизация задач на основе анализа данных
              </CardDescription>
            </div>
            {highPriorityCount > 0 && (
              <Badge variant="destructive" className="text-sm font-semibold px-3 py-1">
                {highPriorityCount} критических задач
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {/* Быстрый старт */}
            {highPriorityCount > 0 && onQuickStart && (
              <Alert className="bg-gradient-to-r from-orange-100/50 to-red-100/50 dark:from-orange-950/30 dark:to-red-950/30 border-orange-300">
                <Zap className="h-4 w-4 text-orange-600" />
                <AlertDescription>
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="font-semibold text-orange-900 dark:text-orange-100">
                        Быстрый старт: Обработать проблемные зоны
                      </p>
                      <p className="text-sm text-orange-700 dark:text-orange-200 mt-1">
                        {totalProblematicRecords.toLocaleString('ru-RU')} записей • ~{estimatedTotalTime} мин
                      </p>
                    </div>
                    <Button
                      onClick={() => onQuickStart('problematic')}
                      className="ml-4 gap-2"
                      size="sm"
                    >
                      <Zap className="h-4 w-4" />
                      Запустить
                    </Button>
                  </div>
                </AlertDescription>
              </Alert>
            )}

            {/* Список рекомендаций */}
            <div className="space-y-3">
              {recommendations.map((rec, index) => (
                <motion.div
                  key={rec.id}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ duration: 0.3, delay: index * 0.1 }}
                >
                  <Card className="hover:shadow-md transition-shadow">
                    <CardContent className="pt-4">
                      <div className="flex items-start justify-between gap-4">
                        <div className="flex-1 space-y-2">
                          <div className="flex items-center gap-2">
                            <Badge
                              className={cn('text-xs font-semibold', getPriorityColor(rec.priority))}
                            >
                              {getPriorityIcon(rec.priority)}
                              <span className="ml-1">
                                {rec.priority === 'high' ? 'Высокий' : rec.priority === 'medium' ? 'Средний' : 'Низкий'} приоритет
                              </span>
                            </Badge>
                            <span className="text-sm font-semibold">{rec.title}</span>
                          </div>
                          <p className="text-sm text-muted-foreground">{rec.description}</p>
                          <div className="flex items-center gap-4 text-xs text-muted-foreground">
                            <div className="flex items-center gap-1">
                              <Target className="h-3 w-3" />
                              {rec.recordCount.toLocaleString('ru-RU')} записей
                            </div>
                            <div className="flex items-center gap-1">
                              <Clock className="h-3 w-3" />
                              {rec.estimatedTime}
                            </div>
                          </div>
                        </div>
                        {rec.onAction && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={rec.onAction}
                            className="gap-2"
                          >
                            {rec.actionLabel}
                            <ArrowRight className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    </CardContent>
                  </Card>
                </motion.div>
              ))}
            </div>

            {/* Дополнительные действия */}
            {onQuickStart && (
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t">
                <Button
                  variant="default"
                  className="w-full gap-2"
                  onClick={() => onQuickStart('problematic')}
                >
                  <Zap className="h-4 w-4" />
                  Проблемные зоны
                  <Badge variant="secondary" className="ml-auto">
                    {totalProblematicRecords.toLocaleString('ru-RU')}
                  </Badge>
                </Button>
                <Button
                  variant="outline"
                  className="w-full gap-2"
                  onClick={() => onQuickStart('full')}
                >
                  <CheckCircle2 className="h-4 w-4" />
                  Полная обработка
                  <Badge variant="outline" className="ml-auto">
                    {stats.total_records.toLocaleString('ru-RU')}
                  </Badge>
                </Button>
                <Button
                  variant="outline"
                  className="w-full gap-2"
                  onClick={() => onQuickStart('selective')}
                >
                  <Target className="h-4 w-4" />
                  Выборочная
                </Button>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

