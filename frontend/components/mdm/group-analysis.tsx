'use client'

import React from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { BarChart3, TrendingUp, Users, CheckCircle2 } from 'lucide-react'

interface GroupAnalysisProps {
  group: {
    normalized_name: string
    category: string
    merged_count: number
    attributeCount?: number
    avg_confidence?: number
    kpved_code?: string
    kpved_name?: string
    kpved_confidence?: number
  }
  type: 'nomenclature' | 'counterparties'
}

export const GroupAnalysis: React.FC<GroupAnalysisProps> = ({ group, type }) => {
  const qualityScore = group.avg_confidence || 0.85
  const completeness = group.attributeCount && group.merged_count
    ? Math.min(1, group.attributeCount / (group.merged_count * 3))
    : 0.7

  return (
    <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <Users className="h-4 w-4" />
            Объединено записей
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-2xl font-bold">{group.merged_count}</div>
          <p className="text-xs text-muted-foreground mt-1">
            {type === 'nomenclature' ? 'позиций номенклатуры' : 'контрагентов'}
          </p>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <BarChart3 className="h-4 w-4" />
            Качество данных
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-2xl font-bold">{Math.round(qualityScore * 100)}%</span>
              <Badge variant={qualityScore >= 0.9 ? 'default' : qualityScore >= 0.7 ? 'secondary' : 'destructive'}>
                {qualityScore >= 0.9 ? 'Высокое' : qualityScore >= 0.7 ? 'Среднее' : 'Низкое'}
              </Badge>
            </div>
            <Progress value={qualityScore * 100} className="h-2" />
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-sm flex items-center gap-2">
            <CheckCircle2 className="h-4 w-4" />
            Полнота
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-2xl font-bold">{Math.round(completeness * 100)}%</span>
              <span className="text-xs text-muted-foreground">
                {group.attributeCount || 0} атрибутов
              </span>
            </div>
            <Progress value={completeness * 100} className="h-2" />
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

