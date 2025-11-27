'use client'

import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { TrendingUp, TrendingDown, AlertCircle, CheckCircle2 } from 'lucide-react'
import { formatPercent, formatNumber } from '@/utils/normalization-helpers'

interface DataQualityMetricsCardProps {
  dimension: 'completeness' | 'accuracy' | 'consistency' | 'timeliness'
  score: number
  trend?: number
  issues?: number
  total?: number
}

const dimensionConfig = {
  completeness: {
    title: 'Полнота',
    description: 'Процент заполненности обязательных полей',
    icon: CheckCircle2,
    color: 'blue',
  },
  accuracy: {
    title: 'Точность',
    description: 'Процент корректных значений',
    icon: CheckCircle2,
    color: 'green',
  },
  consistency: {
    title: 'Согласованность',
    description: 'Процент согласованных данных между источниками',
    icon: AlertCircle,
    color: 'yellow',
  },
  timeliness: {
    title: 'Актуальность',
    description: 'Процент актуальных данных',
    icon: TrendingUp,
    color: 'purple',
  },
}

export const DataQualityMetricsCard: React.FC<DataQualityMetricsCardProps> = ({
  dimension,
  score,
  trend,
  issues,
  total,
}) => {
  const config = dimensionConfig[dimension]
  const Icon = config.icon

  const getScoreColor = (score: number) => {
    if (score >= 0.9) return 'text-green-600'
    if (score >= 0.7) return 'text-yellow-600'
    return 'text-red-600'
  }

  const getScoreBadgeVariant = (score: number): 'default' | 'secondary' | 'destructive' => {
    if (score >= 0.9) return 'default'
    if (score >= 0.7) return 'secondary'
    return 'destructive'
  }

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Icon className={`h-5 w-5 text-${config.color}-600`} />
            <CardTitle className="text-base">{config.title}</CardTitle>
          </div>
          <Badge variant={getScoreBadgeVariant(score)}>
            {Math.round(score * 100)}%
          </Badge>
        </div>
        <CardDescription className="text-xs">
          {config.description}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">Оценка качества</span>
            <span className={`font-semibold ${getScoreColor(score)}`}>
              {formatPercent(score, 0)}
            </span>
          </div>
          <Progress value={score * 100} className="h-2" />
        </div>

        {trend !== undefined && (
          <div className="flex items-center gap-2 text-xs">
            {trend > 0 ? (
              <>
                <TrendingUp className="h-3 w-3 text-green-600" />
                <span className="text-green-600">+{formatPercent(trend, 0)}</span>
              </>
            ) : trend < 0 ? (
              <>
                <TrendingDown className="h-3 w-3 text-red-600" />
                <span className="text-red-600">{formatPercent(trend, 0)}</span>
              </>
            ) : (
              <span className="text-muted-foreground">Без изменений</span>
            )}
            <span className="text-muted-foreground">за последний период</span>
          </div>
        )}

        {issues !== undefined && total !== undefined && (
          <div className="pt-2 border-t text-xs">
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Проблемные записи</span>
              <span className="font-medium">
                {formatNumber(issues)} из {formatNumber(total)}
              </span>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

