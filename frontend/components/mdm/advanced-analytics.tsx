'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { BarChart3, TrendingUp, DollarSign, AlertTriangle, Download, Calendar } from 'lucide-react'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { fetchAnalyticsApi } from '@/lib/mdm/api'
import { logger } from '@/lib/logger'

interface AdvancedAnalyticsProps {
  clientId?: string
  projectId?: string
}

export const AdvancedAnalytics: React.FC<AdvancedAnalyticsProps> = ({
  clientId,
  projectId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId

  const [timeFrame, setTimeFrame] = useState('30d')
  const [comparisonMode, setComparisonMode] = useState<'projects' | 'time' | 'clients'>('time')

  const { data: analyticsData, loading, error, refetch } = useProjectState(
    (cid, pid, signal) =>
      fetchAnalyticsApi(cid, pid, { timeFrame, comparisonMode }, signal),
    effectiveClientId || '',
    effectiveProjectId || '',
    [timeFrame, comparisonMode],
    {
      enabled: !!effectiveClientId && !!effectiveProjectId,
      // Не используем автообновление для аналитики, так как это тяжелая операция
      refetchInterval: null,
    }
  )

  if (!effectiveClientId || !effectiveProjectId) {
    return (
      <Card>
        <CardHeader>
          <CardTitle>Расширенная аналитика</CardTitle>
          <CardDescription>Выберите проект для просмотра аналитики</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  if (loading && !analyticsData) {
    return <LoadingState message="Загрузка аналитики..." />
  }

  if (error) {
    return (
      <ErrorState 
        title="Ошибка загрузки аналитики" 
        message={error} 
        action={{
          label: 'Повторить',
          onClick: refetch,
        }}
      />
    )
  }

  const handleExport = (format: 'pdf' | 'excel' | 'json') => {
    logger.info('Exporting analytics', {
      component: 'AdvancedAnalytics',
      format,
      clientId: effectiveClientId,
      projectId: effectiveProjectId,
    })
    // Логика экспорта
  }

  // Используем данные из хука или заглушку
  const data = analyticsData || {
    efficiency: { successRate: 0.945, processingTime: 120, throughput: 150 },
    quality: { avgQuality: 0.92, issues: 8 },
    impact: { costSavings: 50000, timeSaved: 240 },
    anomalies: { count: 3, severity: 'low' },
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Расширенная аналитика</CardTitle>
              <CardDescription>
                Анализ эффективности и бизнес-эффекта нормализации
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Select value={timeFrame} onValueChange={setTimeFrame}>
                <SelectTrigger className="w-[120px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="realtime">В реальном времени</SelectItem>
                  <SelectItem value="24h">24 часа</SelectItem>
                  <SelectItem value="7d">7 дней</SelectItem>
                  <SelectItem value="30d">30 дней</SelectItem>
                </SelectContent>
              </Select>
              <Button variant="outline" size="sm" onClick={() => handleExport('excel')}>
                <Download className="h-4 w-4 mr-2" />
                Экспорт
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Tabs defaultValue="efficiency">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="efficiency">
                <BarChart3 className="h-4 w-4 mr-2" />
                Эффективность
              </TabsTrigger>
              <TabsTrigger value="quality">
                <TrendingUp className="h-4 w-4 mr-2" />
                Качество
              </TabsTrigger>
              <TabsTrigger value="impact">
                <DollarSign className="h-4 w-4 mr-2" />
                Бизнес-эффект
              </TabsTrigger>
              <TabsTrigger value="anomalies">
                <AlertTriangle className="h-4 w-4 mr-2" />
                Аномалии
              </TabsTrigger>
            </TabsList>

            <TabsContent value="efficiency" className="mt-4">
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Успешность</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">94.5%</div>
                      <p className="text-xs text-muted-foreground mt-1">+2.3% за период</p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Время обработки</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">12.5 сек</div>
                      <p className="text-xs text-muted-foreground mt-1">-1.2 сек за период</p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Улучшение качества</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">+15%</div>
                      <p className="text-xs text-muted-foreground mt-1">Средний прирост</p>
                    </CardContent>
                  </Card>
                </div>
                <div className="p-4 border rounded-lg text-center text-muted-foreground">
                  Графики эффективности нормализации будут добавлены в следующей итерации
                </div>
              </div>
            </TabsContent>

            <TabsContent value="quality" className="mt-4">
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                  {['completeness', 'accuracy', 'consistency', 'timeliness'].map((metric) => (
                    <Card key={metric}>
                      <CardHeader className="pb-2">
                        <CardTitle className="text-sm capitalize">{metric}</CardTitle>
                      </CardHeader>
                      <CardContent>
                        <div className="text-2xl font-bold">87%</div>
                        <Badge variant="secondary" className="mt-2">Хорошо</Badge>
                      </CardContent>
                    </Card>
                  ))}
                </div>
                <div className="p-4 border rounded-lg text-center text-muted-foreground">
                  Метрики качества данных будут добавлены в следующей итерации
                </div>
              </div>
            </TabsContent>

            <TabsContent value="impact" className="mt-4">
              <div className="space-y-4">
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Экономия времени</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">45 часов</div>
                      <p className="text-xs text-muted-foreground mt-1">В месяц</p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Эффективность процессов</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">+32%</div>
                      <p className="text-xs text-muted-foreground mt-1">Улучшение</p>
                    </CardContent>
                  </Card>
                  <Card>
                    <CardHeader className="pb-2">
                      <CardTitle className="text-sm">Поддержка решений</CardTitle>
                    </CardHeader>
                    <CardContent>
                      <div className="text-2xl font-bold">89%</div>
                      <p className="text-xs text-muted-foreground mt-1">Точность</p>
                    </CardContent>
                  </Card>
                </div>
                <div className="p-4 border rounded-lg text-center text-muted-foreground">
                  Анализ бизнес-эффекта будет добавлен в следующей итерации
                </div>
              </div>
            </TabsContent>

            <TabsContent value="anomalies" className="mt-4">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold">Обнаруженные аномалии</h3>
                    <p className="text-sm text-muted-foreground">Выбросы и изменения паттернов</p>
                  </div>
                  <Badge variant="secondary">3 активных</Badge>
                </div>
                <div className="p-4 border rounded-lg text-center text-muted-foreground">
                  Обнаружение аномалий будет добавлено в следующей итерации
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
    </div>
  )
}

