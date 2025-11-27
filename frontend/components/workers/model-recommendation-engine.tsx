'use client'

import { useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { 
  Award, 
  Zap, 
  Shield, 
  DollarSign, 
  TrendingUp,
  CheckCircle2,
  AlertCircle,
  Info
} from 'lucide-react'
import { cn } from '@/lib/utils'

interface BenchmarkResult {
  model: string
  priority: number
  speed: number
  avg_response_time_ms: number
  success_count: number
  error_count: number
  total_requests: number
  success_rate: number
  status: string
}

interface ModelRecommendationEngineProps {
  results: BenchmarkResult[]
  priority?: 'speed' | 'quality' | 'reliability' | 'balanced'
  scenario?: 'realtime' | 'critical' | 'data_processing' | 'cost_saving'
  onSelectModel?: (model: string) => void
}

type Recommendation = {
  model: string
  score: number
  reasons: string[]
  priority: 'high' | 'medium' | 'low'
  scenario: string
}

// Вычисление рекомендаций на основе приоритетов
function calculateRecommendations(
  results: BenchmarkResult[],
  priority: string,
  scenario: string
): Recommendation[] {
  if (!results || results.length === 0) return []

  const recommendations: Recommendation[] = results.map(result => {
    let score = 0
    const reasons: string[] = []
    
    // Базовые метрики
    const speed = result.speed || 0
    const successRate = result.success_rate || 0
    const errorRate = (result.error_count / (result.total_requests || 1)) * 100
    const responseTime = result.avg_response_time_ms || 0
    const totalRequests = result.total_requests || 0

    // Приоритет: скорость
    if (priority === 'speed') {
      score = speed * 0.7 + successRate * 0.3
      if (speed > 10) reasons.push('Высокая скорость обработки')
      if (responseTime < 2000) reasons.push('Быстрое время отклика')
    }
    
    // Приоритет: качество
    else if (priority === 'quality') {
      score = successRate * 0.6 + (100 - errorRate) * 0.4
      if (successRate > 90) reasons.push('Высокая успешность запросов')
      if (errorRate < 5) reasons.push('Низкий процент ошибок')
    }
    
    // Приоритет: надежность
    else if (priority === 'reliability') {
      score = successRate * 0.5 + (100 - errorRate) * 0.3 + (totalRequests > 50 ? 20 : 0)
      if (successRate > 95) reasons.push('Очень высокая надежность')
      if (totalRequests > 50) reasons.push('Протестировано на большом объеме')
    }
    
    // Приоритет: сбалансированный
    else {
      score = (speed / 10) * 0.3 + successRate * 0.4 + (100 - errorRate) * 0.3
      reasons.push('Сбалансированная производительность')
    }

    // Сценарии использования
    if (scenario === 'realtime') {
      if (responseTime < 1500) {
        score += 20
        reasons.push('Оптимален для реального времени')
      }
    } else if (scenario === 'critical') {
      if (successRate > 95 && errorRate < 2) {
        score += 30
        reasons.push('Подходит для критических задач')
      }
    } else if (scenario === 'data_processing') {
      if (speed > 5 && totalRequests > 30) {
        score += 15
        reasons.push('Эффективен для обработки данных')
      }
    } else if (scenario === 'cost_saving') {
      if (result.priority && result.priority <= 3) {
        score += 10
        reasons.push('Экономически выгоден')
      }
    }

    // Штрафы
    if (errorRate > 20) {
      score -= 30
      reasons.push('⚠️ Высокий процент ошибок')
    }
    if (totalRequests < 10) {
      score -= 15
      reasons.push('⚠️ Мало тестовых запросов')
    }
    if (responseTime > 10000) {
      score -= 20
      reasons.push('⚠️ Медленное время отклика')
    }

    // Определяем приоритет рекомендации
    let recommendationPriority: 'high' | 'medium' | 'low' = 'medium'
    if (score > 80) recommendationPriority = 'high'
    else if (score < 40) recommendationPriority = 'low'

    return {
      model: result.model,
      score: Math.max(0, Math.min(100, score)),
      reasons,
      priority: recommendationPriority,
      scenario: scenario || 'balanced',
    }
  })

  // Сортируем по score
  return recommendations
    .sort((a, b) => b.score - a.score)
    .slice(0, 5) // Топ-5 рекомендаций
}

export function ModelRecommendationEngine({
  results,
  priority = 'balanced',
  scenario = 'realtime',
  onSelectModel,
}: ModelRecommendationEngineProps) {
  const recommendations = useMemo(
    () => calculateRecommendations(results, priority, scenario),
    [results, priority, scenario]
  )

  const priorityLabels = {
    speed: 'Скорость',
    quality: 'Качество',
    reliability: 'Надежность',
    balanced: 'Сбалансированный',
  }

  const scenarioLabels = {
    realtime: 'Реальное время',
    critical: 'Критические задачи',
    data_processing: 'Обработка данных',
    cost_saving: 'Экономия средств',
  }

  if (recommendations.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Award className="h-5 w-5" />
            Рекомендации моделей
          </CardTitle>
          <CardDescription>
            Недостаточно данных для рекомендаций
          </CardDescription>
        </CardHeader>
      </Card>
    )
  }

  const topRecommendation = recommendations[0]

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Award className="h-5 w-5" />
              Рекомендации моделей
            </CardTitle>
            <CardDescription>
              Приоритет: {priorityLabels[priority]} | Сценарий: {scenarioLabels[scenario]}
            </CardDescription>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Топ рекомендация */}
        {topRecommendation && (
          <div className="p-4 bg-primary/5 rounded-lg border-2 border-primary/20">
            <div className="flex items-start justify-between mb-2">
              <div className="flex items-center gap-2">
                <Badge variant="default" className="bg-primary">
                  <Award className="h-3 w-3 mr-1" />
                  Лучший выбор
                </Badge>
                <span className="font-semibold text-lg">{topRecommendation.model}</span>
              </div>
              <div className="text-right">
                <div className="text-2xl font-bold text-primary">
                  {topRecommendation.score.toFixed(0)}
                </div>
                <div className="text-xs text-muted-foreground">баллов</div>
              </div>
            </div>
            <div className="space-y-1 mt-3">
              {topRecommendation.reasons.slice(0, 3).map((reason, idx) => (
                <div key={idx} className="flex items-center gap-2 text-sm">
                  <CheckCircle2 className="h-3 w-3 text-green-600" />
                  <span>{reason}</span>
                </div>
              ))}
            </div>
            {onSelectModel && (
              <Button
                size="sm"
                className="mt-3 w-full"
                onClick={() => onSelectModel(topRecommendation.model)}
              >
                Выбрать эту модель
              </Button>
            )}
          </div>
        )}

        {/* Остальные рекомендации */}
        <div className="space-y-2">
          <h4 className="text-sm font-semibold text-muted-foreground">Альтернативные варианты:</h4>
          {recommendations.slice(1).map((rec, idx) => (
            <div
              key={rec.model}
              className={cn(
                'p-3 rounded-lg border transition-all hover:bg-muted/50',
                rec.priority === 'high' && 'border-green-200 bg-green-50/50 dark:bg-green-950/10',
                rec.priority === 'medium' && 'border-yellow-200 bg-yellow-50/50 dark:bg-yellow-950/10',
                rec.priority === 'low' && 'border-gray-200'
              )}
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="font-medium text-sm">{rec.model}</span>
                  <Badge
                    variant="outline"
                    className={cn(
                      'text-xs',
                      rec.priority === 'high' && 'border-green-500 text-green-700',
                      rec.priority === 'medium' && 'border-yellow-500 text-yellow-700',
                      rec.priority === 'low' && 'border-gray-500 text-gray-700'
                    )}
                  >
                    {rec.score.toFixed(0)}
                  </Badge>
                </div>
                {onSelectModel && (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onSelectModel(rec.model)}
                  >
                    Выбрать
                  </Button>
                )}
              </div>
              <div className="space-y-1">
                {rec.reasons.slice(0, 2).map((reason, reasonIdx) => (
                  <div key={reasonIdx} className="flex items-start gap-2 text-xs text-muted-foreground">
                    {reason.startsWith('⚠️') ? (
                      <AlertCircle className="h-3 w-3 text-amber-500 mt-0.5" />
                    ) : (
                      <Info className="h-3 w-3 text-blue-500 mt-0.5" />
                    )}
                    <span>{reason}</span>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>

        {/* Информация о метриках */}
        <div className="pt-4 mt-4 border-t text-xs text-muted-foreground space-y-1">
          <p className="font-semibold">Как вычисляется оценка:</p>
          <ul className="list-disc list-inside space-y-0.5 ml-2">
            {priority === 'speed' && (
              <>
                <li>Скорость (70%) + Успешность (30%)</li>
                <li>Бонус за быстрое время отклика</li>
              </>
            )}
            {priority === 'quality' && (
              <>
                <li>Успешность (60%) + Низкий процент ошибок (40%)</li>
                <li>Штраф за высокий процент ошибок</li>
              </>
            )}
            {priority === 'reliability' && (
              <>
                <li>Успешность (50%) + Низкий процент ошибок (30%)</li>
                <li>Бонус за большой объем тестирования</li>
              </>
            )}
            {priority === 'balanced' && (
              <>
                <li>Скорость (30%) + Успешность (40%) + Низкий процент ошибок (30%)</li>
                <li>Сбалансированная оценка всех метрик</li>
              </>
            )}
          </ul>
        </div>
      </CardContent>
    </Card>
  )
}

