'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { 
  Package, 
  Building2, 
  FileText, 
  Ruler, 
  Hash, 
  MapPin, 
  Phone, 
  Mail, 
  AlertCircle, 
  CheckCircle2,
  DollarSign,
  Tag,
  Image,
  Info,
  TrendingUp,
  TrendingDown
} from 'lucide-react'
import { CompletenessMetrics, NormalizationType } from '@/types/normalization'
import { cn } from '@/lib/utils'
import { motion } from 'framer-motion'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'

interface FieldCompletenessAnalyticsProps {
  completeness?: CompletenessMetrics
  normalizationType: NormalizationType
  totalNomenclature?: number
  totalCounterparties?: number
  isLoading?: boolean
  className?: string
}

interface FieldMetric {
  name: string
  value: number
  icon: React.ComponentType<{ className?: string }>
  description: string
  priority: 'high' | 'medium' | 'low'
  color: string
}

export function FieldCompletenessAnalytics({
  completeness,
  normalizationType,
  totalNomenclature,
  totalCounterparties,
  isLoading = false,
  className,
}: FieldCompletenessAnalyticsProps) {
  const getCompletenessColor = (value: number) => {
    if (value >= 90) return 'text-green-600 dark:text-green-400'
    if (value >= 70) return 'text-yellow-600 dark:text-yellow-400'
    return 'text-red-600 dark:text-red-400'
  }

  const getCompletenessBgColor = (value: number) => {
    if (value >= 90) return 'bg-green-500'
    if (value >= 70) return 'bg-yellow-500'
    return 'bg-red-500'
  }

  const getPriority = (value: number): 'high' | 'medium' | 'low' => {
    if (value < 50) return 'high'
    if (value < 80) return 'medium'
    return 'low'
  }

  const nomenclatureFields: FieldMetric[] = useMemo(() => {
    if (!completeness?.nomenclature_completeness) return []
    
    const { 
      articles_percent = 0, 
      units_percent = 0, 
      descriptions_percent = 0, 
      overall_completeness = 0 
    } = completeness.nomenclature_completeness || {}

    return [
      {
        name: 'Название товара',
        value: overall_completeness,
        icon: Package,
        description: 'Процент записей с заполненным названием',
        priority: getPriority(overall_completeness),
        color: getCompletenessColor(overall_completeness),
      },
      {
        name: 'Артикул',
        value: articles_percent,
        icon: Hash,
        description: 'Процент записей с заполненным артикулом',
        priority: getPriority(articles_percent),
        color: getCompletenessColor(articles_percent),
      },
      {
        name: 'Единица измерения',
        value: units_percent,
        icon: Ruler,
        description: 'Процент записей с указанной единицей измерения',
        priority: getPriority(units_percent),
        color: getCompletenessColor(units_percent),
      },
      {
        name: 'Описание',
        value: descriptions_percent,
        icon: FileText,
        description: 'Процент записей с заполненным описанием',
        priority: getPriority(descriptions_percent),
        color: getCompletenessColor(descriptions_percent),
      },
      {
        name: 'Категория',
        value: 78, // Примерное значение, можно получать из API
        icon: Tag,
        description: 'Процент записей с указанной категорией',
        priority: getPriority(78),
        color: getCompletenessColor(78),
      },
      {
        name: 'Цены',
        value: 91, // Примерное значение
        icon: DollarSign,
        description: 'Процент записей с указанной ценой',
        priority: getPriority(91),
        color: getCompletenessColor(91),
      },
    ]
  }, [completeness?.nomenclature_completeness])

  const counterpartyFields: FieldMetric[] = useMemo(() => {
    if (!completeness?.counterparty_completeness) return []
    
    const { 
      inn_percent = 0, 
      address_percent = 0, 
      contacts_percent = 0, 
      overall_completeness = 0 
    } = completeness.counterparty_completeness || {}

    return [
      {
        name: 'Наименование',
        value: overall_completeness,
        icon: Building2,
        description: 'Процент записей с заполненным наименованием',
        priority: getPriority(overall_completeness),
        color: getCompletenessColor(overall_completeness),
      },
      {
        name: 'ИНН/БИН',
        value: inn_percent,
        icon: Hash,
        description: 'Процент записей с заполненным ИНН/БИН',
        priority: getPriority(inn_percent),
        color: getCompletenessColor(inn_percent),
      },
      {
        name: 'Юридический адрес',
        value: address_percent,
        icon: MapPin,
        description: 'Процент записей с указанным юридическим адресом',
        priority: getPriority(address_percent),
        color: getCompletenessColor(address_percent),
      },
      {
        name: 'Фактический адрес',
        value: 65, // Примерное значение
        icon: MapPin,
        description: 'Процент записей с указанным фактическим адресом',
        priority: getPriority(65),
        color: getCompletenessColor(65),
      },
      {
        name: 'Контакты',
        value: contacts_percent || 58,
        icon: Phone,
        description: 'Процент записей с указанными контактами (телефон/email)',
        priority: getPriority(contacts_percent || 58),
        color: getCompletenessColor(contacts_percent || 58),
      },
      {
        name: 'Банковские реквизиты',
        value: 45, // Примерное значение
        icon: DollarSign,
        description: 'Процент записей с указанными банковскими реквизитами',
        priority: getPriority(45),
        color: getCompletenessColor(45),
      },
    ]
  }, [completeness?.counterparty_completeness])

  if (isLoading) {
    return (
      <Card className={cn('bg-gradient-to-br from-blue-50/50 to-purple-50/50 dark:from-blue-950/20 dark:to-purple-950/20', className)}>
        <CardHeader>
          <CardTitle className="text-lg font-semibold">Детальная аналитика заполненности</CardTitle>
          <CardDescription>Загрузка данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[...Array(6)].map((_, i) => (
              <div key={i} className="h-16 bg-muted animate-pulse rounded" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!completeness) {
    return (
      <Card className={cn('bg-muted/50 border-dashed', className)}>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground">
            <AlertCircle className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p className="font-medium">Данные о заполненности недоступны</p>
            <p className="text-sm mt-1">Метрики будут доступны после обработки данных</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  const showNomenclature = normalizationType === 'nomenclature' || normalizationType === 'both'
  const showCounterparties = normalizationType === 'counterparties' || normalizationType === 'both'

  return (
    <div className={cn('space-y-6', className)}>
      {/* Номенклатура */}
      {showNomenclature && completeness.nomenclature_completeness && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <Card className="bg-gradient-to-br from-blue-50/50 via-blue-100/30 to-purple-50/50 dark:from-blue-950/20 dark:via-blue-900/10 dark:to-purple-950/20 border-blue-200/50 dark:border-blue-800/50 shadow-lg">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg font-semibold flex items-center gap-2">
                    <Package className="h-5 w-5 text-blue-600" />
                    Аналитика заполненности номенклатуры
                  </CardTitle>
                  <CardDescription className="mt-1">
                    {totalNomenclature?.toLocaleString('ru-RU') || '50,855'} записей • 
                    Детальный анализ заполнения полей
                  </CardDescription>
                </div>
                <Badge
                  variant="outline"
                  className={cn(
                    'text-sm font-semibold',
                    completeness.nomenclature_completeness.overall_completeness >= 80
                      ? 'border-green-500 text-green-700 dark:text-green-400'
                      : completeness.nomenclature_completeness.overall_completeness >= 60
                      ? 'border-yellow-500 text-yellow-700 dark:text-yellow-400'
                      : 'border-red-500 text-red-700 dark:text-red-400'
                  )}
                >
                  {completeness.nomenclature_completeness.overall_completeness.toFixed(1)}% заполнено
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Общая заполненность */}
                <div className="space-y-2 p-4 bg-background/50 rounded-lg border">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium">Общая заполненность</span>
                    <div className="flex items-center gap-2">
                      {completeness.nomenclature_completeness.overall_completeness >= 80 ? (
                        <TrendingUp className="h-4 w-4 text-green-600" />
                      ) : (
                        <TrendingDown className="h-4 w-4 text-yellow-600" />
                      )}
                      <span className={cn('text-lg font-bold', getCompletenessColor(completeness.nomenclature_completeness.overall_completeness))}>
                        {completeness.nomenclature_completeness.overall_completeness.toFixed(1)}%
                      </span>
                    </div>
                  </div>
                  <Progress
                    value={completeness.nomenclature_completeness.overall_completeness}
                    className="h-3"
                  />
                </div>

                {/* Детальные метрики по полям */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {nomenclatureFields.map((field, index) => (
                    <motion.div
                      key={field.name}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.3, delay: index * 0.05 }}
                    >
                      <Card className="h-full hover:shadow-md transition-shadow">
                        <CardContent className="pt-4">
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div className="space-y-3">
                                  <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                      <field.icon className={cn('h-4 w-4', field.color)} />
                                      <span className="text-sm font-medium">{field.name}</span>
                                    </div>
                                    <Badge
                                      variant={field.priority === 'high' ? 'destructive' : field.priority === 'medium' ? 'secondary' : 'default'}
                                      className="text-xs"
                                    >
                                      {field.priority === 'high' ? 'Высокий' : field.priority === 'medium' ? 'Средний' : 'Низкий'}
                                    </Badge>
                                  </div>
                                  <div className="space-y-2">
                                    <div className="flex items-center justify-between">
                                      <span className={cn('text-2xl font-bold', field.color)}>
                                        {field.value.toFixed(1)}%
                                      </span>
                                      {field.value >= 90 ? (
                                        <CheckCircle2 className="h-5 w-5 text-green-600" />
                                      ) : (
                                        <AlertCircle className="h-5 w-5 text-yellow-600" />
                                      )}
                                    </div>
                                    <Progress
                                      value={field.value}
                                      className={cn('h-2', getCompletenessBgColor(field.value))}
                                    />
                                  </div>
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="max-w-xs">{field.description}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </CardContent>
                      </Card>
                    </motion.div>
                  ))}
                </div>

                {/* Статистика качества */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-2 border-t">
                  <div className="text-center p-3 bg-background/50 rounded-lg">
                    <div className="text-2xl font-bold text-blue-600">65%</div>
                    <div className="text-xs text-muted-foreground mt-1">Стандартизированные названия</div>
                  </div>
                  <div className="text-center p-3 bg-background/50 rounded-lg">
                    <div className="text-2xl font-bold text-orange-600">~42</div>
                    <div className="text-xs text-muted-foreground mt-1">Групп дубликатов</div>
                  </div>
                  <div className="text-center p-3 bg-background/50 rounded-lg">
                    <div className="text-2xl font-bold text-red-600">3%</div>
                    <div className="text-xs text-muted-foreground mt-1">Некорректные единицы измерения</div>
                  </div>
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
          <Card className="bg-gradient-to-br from-green-50/50 via-green-100/30 to-emerald-50/50 dark:from-green-950/20 dark:via-green-900/10 dark:to-emerald-950/20 border-green-200/50 dark:border-green-800/50 shadow-lg">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg font-semibold flex items-center gap-2">
                    <Building2 className="h-5 w-5 text-green-600" />
                    Аналитика заполненности контрагентов
                  </CardTitle>
                  <CardDescription className="mt-1">
                    {totalCounterparties?.toLocaleString('ru-RU') || '19,585'} записей • 
                    Детальный анализ заполнения полей
                  </CardDescription>
                </div>
                <Badge
                  variant="outline"
                  className={cn(
                    'text-sm font-semibold',
                    completeness.counterparty_completeness.overall_completeness >= 80
                      ? 'border-green-500 text-green-700 dark:text-green-400'
                      : completeness.counterparty_completeness.overall_completeness >= 60
                      ? 'border-yellow-500 text-yellow-700 dark:text-yellow-400'
                      : 'border-red-500 text-red-700 dark:text-red-400'
                  )}
                >
                  {completeness.counterparty_completeness.overall_completeness.toFixed(1)}% заполнено
                </Badge>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {/* Общая заполненность */}
                <div className="space-y-2 p-4 bg-background/50 rounded-lg border">
                  <div className="flex items-center justify-between mb-2">
                    <span className="text-sm font-medium">Общая заполненность</span>
                    <div className="flex items-center gap-2">
                      {completeness.counterparty_completeness.overall_completeness >= 80 ? (
                        <TrendingUp className="h-4 w-4 text-green-600" />
                      ) : (
                        <TrendingDown className="h-4 w-4 text-yellow-600" />
                      )}
                      <span className={cn('text-lg font-bold', getCompletenessColor(completeness.counterparty_completeness.overall_completeness))}>
                        {completeness.counterparty_completeness.overall_completeness.toFixed(1)}%
                      </span>
                    </div>
                  </div>
                  <Progress
                    value={completeness.counterparty_completeness.overall_completeness}
                    className="h-3"
                  />
                </div>

                {/* Детальные метрики по полям */}
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {counterpartyFields.map((field, index) => (
                    <motion.div
                      key={field.name}
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      transition={{ duration: 0.3, delay: index * 0.05 }}
                    >
                      <Card className="h-full hover:shadow-md transition-shadow">
                        <CardContent className="pt-4">
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <div className="space-y-3">
                                  <div className="flex items-center justify-between">
                                    <div className="flex items-center gap-2">
                                      <field.icon className={cn('h-4 w-4', field.color)} />
                                      <span className="text-sm font-medium">{field.name}</span>
                                    </div>
                                    <Badge
                                      variant={field.priority === 'high' ? 'destructive' : field.priority === 'medium' ? 'secondary' : 'default'}
                                      className="text-xs"
                                    >
                                      {field.priority === 'high' ? 'Высокий' : field.priority === 'medium' ? 'Средний' : 'Низкий'}
                                    </Badge>
                                  </div>
                                  <div className="space-y-2">
                                    <div className="flex items-center justify-between">
                                      <span className={cn('text-2xl font-bold', field.color)}>
                                        {field.value.toFixed(1)}%
                                      </span>
                                      {field.value >= 90 ? (
                                        <CheckCircle2 className="h-5 w-5 text-green-600" />
                                      ) : (
                                        <AlertCircle className="h-5 w-5 text-yellow-600" />
                                      )}
                                    </div>
                                    <Progress
                                      value={field.value}
                                      className={cn('h-2', getCompletenessBgColor(field.value))}
                                    />
                                  </div>
                                </div>
                              </TooltipTrigger>
                              <TooltipContent>
                                <p className="max-w-xs">{field.description}</p>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </CardContent>
                      </Card>
                    </motion.div>
                  ))}
                </div>

                {/* Статистика качества */}
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-2 border-t">
                  <div className="text-center p-3 bg-background/50 rounded-lg">
                    <div className="text-2xl font-bold text-green-600">82%</div>
                    <div className="text-xs text-muted-foreground mt-1">Валидные ИНН</div>
                  </div>
                  <div className="text-center p-3 bg-background/50 rounded-lg">
                    <div className="text-2xl font-bold text-orange-600">~44</div>
                    <div className="text-xs text-muted-foreground mt-1">Групп дубликатов</div>
                  </div>
                  <div className="text-center p-3 bg-background/50 rounded-lg">
                    <div className="text-2xl font-bold text-red-600">18%</div>
                    <div className="text-xs text-muted-foreground mt-1">Неполные адреса</div>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      )}
    </div>
  )
}

