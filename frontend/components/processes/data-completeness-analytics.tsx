'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { Package, Building2, FileText, Ruler, Hash, MapPin, Phone, Mail, AlertCircle, CheckCircle2 } from 'lucide-react'
import { CompletenessMetrics, NormalizationType } from '@/types/normalization'
import { cn } from '@/lib/utils'
import { motion } from 'framer-motion'

interface DataCompletenessAnalyticsProps {
  completeness?: CompletenessMetrics
  normalizationType: NormalizationType
  isLoading?: boolean
  className?: string
}

export function DataCompletenessAnalytics({
  completeness,
  normalizationType,
  isLoading = false,
  className,
}: DataCompletenessAnalyticsProps) {
  if (isLoading) {
    return (
      <Card className={cn('bg-gradient-to-br from-blue-50/50 to-purple-50/50 dark:from-blue-950/20 dark:to-purple-950/20', className)}>
        <CardHeader>
          <CardTitle className="text-sm font-medium">Аналитика заполнения справочников</CardTitle>
          <CardDescription className="text-xs">Загрузка данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <div className="h-4 bg-muted animate-pulse rounded" />
            <div className="h-4 bg-muted animate-pulse rounded" />
            <div className="h-4 bg-muted animate-pulse rounded" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!completeness) {
    return (
      <Card className={cn('bg-muted/50 border-dashed', className)}>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground text-sm">
            <AlertCircle className="h-5 w-5 mx-auto mb-2 opacity-50" />
            <p>Данные о заполненности недоступны</p>
            <p className="text-xs mt-1">Метрики будут доступны после обработки данных</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  const showNomenclature = normalizationType === 'nomenclature' || normalizationType === 'both'
  const showCounterparties = normalizationType === 'counterparties' || normalizationType === 'both'

  return (
    <div className={cn('space-y-4', className)}>
      {/* Номенклатура */}
      {showNomenclature && completeness.nomenclature_completeness && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <Card className="bg-gradient-to-br from-blue-50/50 to-blue-100/50 dark:from-blue-950/20 dark:to-blue-900/20 border-blue-200/50 dark:border-blue-800/50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Package className="h-4 w-4 text-blue-600" />
                Заполненность номенклатуры
              </CardTitle>
              <CardDescription className="text-xs">
                Анализ полноты данных по товарам и услугам
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Общая заполненность */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Общая заполненность</span>
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={completeness.nomenclature_completeness.overall_completeness >= 80 ? 'default' : 'secondary'}
                      className={cn(
                        completeness.nomenclature_completeness.overall_completeness >= 80
                          ? 'bg-green-600'
                          : completeness.nomenclature_completeness.overall_completeness >= 60
                          ? 'bg-yellow-600'
                          : 'bg-red-600'
                      )}
                    >
                      {completeness.nomenclature_completeness.overall_completeness.toFixed(1)}%
                    </Badge>
                    {completeness.nomenclature_completeness.overall_completeness >= 80 ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600" />
                    ) : (
                      <AlertCircle className="h-4 w-4 text-yellow-600" />
                    )}
                  </div>
                </div>
                <Progress
                  value={completeness.nomenclature_completeness.overall_completeness}
                  className="h-2"
                />
              </div>

              {/* Детальные метрики */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-2 border-t">
                {/* Артикулы */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Hash className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">Артикулы</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-lg font-bold">
                      {completeness.nomenclature_completeness.articles_percent.toFixed(1)}%
                    </span>
                  </div>
                  <Progress
                    value={completeness.nomenclature_completeness.articles_percent}
                    className="h-1.5"
                  />
                </div>

                {/* Единицы измерения */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Ruler className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">Единицы измерения</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-lg font-bold">
                      {completeness.nomenclature_completeness.units_percent.toFixed(1)}%
                    </span>
                  </div>
                  <Progress
                    value={completeness.nomenclature_completeness.units_percent}
                    className="h-1.5"
                  />
                </div>

                {/* Описания */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <FileText className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">Описания</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-lg font-bold">
                      {completeness.nomenclature_completeness.descriptions_percent.toFixed(1)}%
                    </span>
                  </div>
                  <Progress
                    value={completeness.nomenclature_completeness.descriptions_percent}
                    className="h-1.5"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Контрагенты */}
      {showCounterparties && completeness.counterparty_completeness && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
        >
          <Card className="bg-gradient-to-br from-green-50/50 to-green-100/50 dark:from-green-950/20 dark:to-green-900/20 border-green-200/50 dark:border-green-800/50">
            <CardHeader className="pb-3">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <Building2 className="h-4 w-4 text-green-600" />
                Заполненность контрагентов
              </CardTitle>
              <CardDescription className="text-xs">
                Анализ полноты данных по контрагентам
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Общая заполненность */}
              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <span className="text-sm font-medium">Общая заполненность</span>
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={completeness.counterparty_completeness.overall_completeness >= 80 ? 'default' : 'secondary'}
                      className={cn(
                        completeness.counterparty_completeness.overall_completeness >= 80
                          ? 'bg-green-600'
                          : completeness.counterparty_completeness.overall_completeness >= 60
                          ? 'bg-yellow-600'
                          : 'bg-red-600'
                      )}
                    >
                      {completeness.counterparty_completeness.overall_completeness.toFixed(1)}%
                    </Badge>
                    {completeness.counterparty_completeness.overall_completeness >= 80 ? (
                      <CheckCircle2 className="h-4 w-4 text-green-600" />
                    ) : (
                      <AlertCircle className="h-4 w-4 text-yellow-600" />
                    )}
                  </div>
                </div>
                <Progress
                  value={completeness.counterparty_completeness.overall_completeness}
                  className="h-2"
                />
              </div>

              {/* Детальные метрики */}
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-2 border-t">
                {/* ИНН/БИН */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Hash className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">ИНН/БИН</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-lg font-bold">
                      {completeness.counterparty_completeness.inn_percent.toFixed(1)}%
                    </span>
                  </div>
                  <Progress
                    value={completeness.counterparty_completeness.inn_percent}
                    className="h-1.5"
                  />
                </div>

                {/* Адреса */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <MapPin className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">Адреса</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-lg font-bold">
                      {completeness.counterparty_completeness.address_percent.toFixed(1)}%
                    </span>
                  </div>
                  <Progress
                    value={completeness.counterparty_completeness.address_percent}
                    className="h-1.5"
                  />
                </div>

                {/* Контакты */}
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <Phone className="h-4 w-4 text-muted-foreground" />
                    <span className="text-xs font-medium">Контакты</span>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-lg font-bold">
                      {completeness.counterparty_completeness.contacts_percent.toFixed(1)}%
                    </span>
                  </div>
                  <Progress
                    value={completeness.counterparty_completeness.contacts_percent}
                    className="h-1.5"
                  />
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Сообщение, если нет данных */}
      {!showNomenclature && !showCounterparties && (
        <Card>
          <CardContent className="pt-6">
            <div className="text-center text-muted-foreground text-sm">
              Выберите тип нормализации для отображения аналитики заполненности
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

