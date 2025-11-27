'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { Sparkles, TrendingUp, Package, CheckCircle2, AlertCircle } from 'lucide-react'
import { motion } from 'framer-motion'
import { logger } from '@/lib/logger'
import { handleError } from '@/lib/error-handler'

interface PreviewGroup {
  normalized_name: string
  category: string
  merged_count: number
  kpved_code?: string
  kpved_name?: string
  kpved_confidence?: number
  avg_confidence?: number
}

interface NormalizationPreviewResultsProps {
  clientId: number
  projectId: number
  isEnabled?: boolean
}

export function NormalizationPreviewResults({ 
  clientId, 
  projectId,
  isEnabled = true 
}: NormalizationPreviewResultsProps) {
  const [previewGroups, setPreviewGroups] = useState<PreviewGroup[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [totalGroups, setTotalGroups] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!isEnabled || !clientId || !projectId) return

    const fetchPreview = async () => {
      setIsLoading(true)
      setError(null)
      
      try {
        // Получаем первые несколько групп для превью
        const response = await fetch(
          `/api/normalization/groups?page=1&limit=5&include_ai=true`,
          { cache: 'no-store', signal: AbortSignal.timeout(5000) }
        )

        if (response.ok) {
          const data = await response.json()
          setPreviewGroups(data.groups || [])
          setTotalGroups(data.total || 0)
        } else {
          // Если нет данных (еще не запускали нормализацию), это нормально
          if (response.status === 404) {
            setPreviewGroups([])
            setTotalGroups(0)
          } else {
            throw new Error('Не удалось загрузить превью')
          }
        }
      } catch (err) {
        // Игнорируем ошибки - превью не критично, но логируем для отладки
        logger.debug('Preview results not available', {
          component: 'NormalizationPreviewResults',
          clientId,
          projectId,
          isEnabled,
          error: err instanceof Error ? err.message : String(err)
        })
        setPreviewGroups([])
        setTotalGroups(null)
      } finally {
        setIsLoading(false)
      }
    }

    fetchPreview()
  }, [clientId, projectId, isEnabled])

  if (!isEnabled) return null

  if (isLoading) {
    return (
      <Card className="backdrop-blur-sm bg-gradient-to-br from-purple-50/50 to-indigo-50/50 dark:from-purple-950/20 dark:to-indigo-950/20 border-purple-200/50 dark:border-purple-800/50 shadow-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Sparkles className="h-5 w-5 text-purple-600" />
            Preview результатов
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Skeleton className="h-32 w-full" />
        </CardContent>
      </Card>
    )
  }

  // Если нет данных, показываем информационное сообщение
  if (previewGroups.length === 0 && totalGroups === 0) {
    return (
      <Card className="backdrop-blur-sm bg-gradient-to-br from-blue-50/50 to-cyan-50/50 dark:from-blue-950/20 dark:to-cyan-950/20 border-blue-200/50 dark:border-blue-800/50 shadow-md">
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <Sparkles className="h-5 w-5 text-blue-600" />
            Preview результатов
          </CardTitle>
          <CardDescription>
            Примеры нормализованных данных появятся после первого запуска нормализации
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-6 text-muted-foreground">
            <Package className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p className="text-sm">
              Запустите процесс нормализации, чтобы увидеть примеры нормализованных групп товаров
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (previewGroups.length === 0) {
    return null
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.5 }}
    >
      <Card className="backdrop-blur-sm bg-gradient-to-br from-purple-50/50 to-indigo-50/50 dark:from-purple-950/20 dark:to-indigo-950/20 border-purple-200/50 dark:border-purple-800/50 shadow-lg">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2 text-lg">
                <Sparkles className="h-5 w-5 text-purple-600" />
                Preview результатов
              </CardTitle>
              <CardDescription>
                Примеры нормализованных групп товаров
                {totalGroups !== null && totalGroups > 0 && (
                  <span className="ml-2">
                    (всего групп: {totalGroups.toLocaleString()})
                  </span>
                )}
              </CardDescription>
            </div>
            {totalGroups !== null && totalGroups > 0 && (
              <Badge variant="outline" className="text-sm">
                <TrendingUp className="h-3 w-3 mr-1" />
                {totalGroups.toLocaleString()} групп
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {previewGroups.map((group, index) => (
              <motion.div
                key={`${group.normalized_name}-${group.category}-${index}`}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ duration: 0.3, delay: index * 0.1 }}
                className="p-4 bg-background/60 rounded-lg border border-purple-200/30 dark:border-purple-800/30 hover:bg-background/80 transition-colors"
              >
                <div className="space-y-2">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex-1">
                      <div className="font-semibold text-base mb-1">
                        {group.normalized_name}
                      </div>
                      <div className="flex items-center gap-2 flex-wrap">
                        <Badge variant="outline" className="text-xs">
                          {group.category}
                        </Badge>
                        {group.merged_count > 0 && (
                          <Badge variant="secondary" className="text-xs">
                            {group.merged_count} {group.merged_count === 1 ? 'запись' : group.merged_count < 5 ? 'записи' : 'записей'}
                          </Badge>
                        )}
                        {group.avg_confidence !== undefined && group.avg_confidence > 0 && (
                          <Badge 
                            variant={group.avg_confidence >= 0.9 ? 'default' : 'secondary'} 
                            className="text-xs"
                          >
                            Уверенность: {(group.avg_confidence * 100).toFixed(0)}%
                          </Badge>
                        )}
                      </div>
                    </div>
                    {group.kpved_code && (
                      <div className="text-right">
                        <div className="text-xs text-muted-foreground mb-1">КПВЭД</div>
                        <div className="font-mono text-sm font-semibold text-purple-600">
                          {group.kpved_code}
                        </div>
                        {group.kpved_confidence !== undefined && group.kpved_confidence > 0 && (
                          <div className="text-xs text-muted-foreground mt-1">
                            {(group.kpved_confidence * 100).toFixed(0)}%
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                  {group.kpved_name && (
                    <div className="text-xs text-muted-foreground pt-2 border-t border-purple-200/30 dark:border-purple-800/30">
                      {group.kpved_name}
                    </div>
                  )}
                </div>
              </motion.div>
            ))}
            
            {totalGroups !== null && totalGroups > previewGroups.length && (
              <div className="text-center pt-2 border-t border-purple-200/30 dark:border-purple-800/30">
                <p className="text-xs text-muted-foreground">
                  Показано {previewGroups.length} из {totalGroups.toLocaleString()} групп
                </p>
                <p className="text-xs text-muted-foreground mt-1">
                  Полный список будет доступен после запуска нормализации
                </p>
              </div>
            )}
          </div>
        </CardContent>
      </Card>
    </motion.div>
  )
}

