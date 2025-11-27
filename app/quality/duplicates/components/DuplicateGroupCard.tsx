'use client'

import { memo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { CheckCircle, GitMerge, Star, TrendingUp } from 'lucide-react'

interface DuplicateItem {
  id: number
  normalized_name: string
  code: string
  category: string
  kpved_code: string
  quality_score: number
  processing_level: string
  merged_count: number
}

interface DuplicateGroup {
  id: number
  duplicate_type?: string
  detection_method?: string
  similarity_score: number
  suggested_master_id: number
  item_count?: number
  item_ids?: number[]
  items?: DuplicateItem[]
  merged: boolean
  merged_at: string | null
  created_at: string
  reason?: string
}

interface DuplicateGroupCardProps {
  group: DuplicateGroup
  mergingId: number | null
  onMerge: (groupId: number) => void
  getMethodBadge: (group: DuplicateGroup) => React.ReactNode
  getSimilarityBadge: (score: number) => React.ReactNode
  getQualityBadge: (score: number) => React.ReactNode
  getProcessingLevelBadge: (level: string) => React.ReactNode
}

const DuplicateGroupCard = memo<DuplicateGroupCardProps>(({
  group,
  mergingId,
  onMerge,
  getMethodBadge,
  getSimilarityBadge,
  getQualityBadge,
  getProcessingLevelBadge,
}) => {
  const masterItem = group.items?.find(item => item.id === group.suggested_master_id)

  return (
    <Card
      className={`border-l-4 ${
        group.merged ? 'border-green-500 opacity-60' : 'border-orange-500'
      }`}
    >
      <CardHeader>
        <div className="flex items-start justify-between">
          <div className="space-y-2 flex-1">
            <div className="flex items-center gap-2 flex-wrap">
              {getMethodBadge(group)}
              {getSimilarityBadge(group.similarity_score)}
              <Badge variant="outline">
                {group.item_count || group.items?.length || group.item_ids?.length || 0} записей
              </Badge>
              {group.reason && (
                <Badge variant="outline" className="text-xs">
                  {group.reason}
                </Badge>
              )}
              {group.merged && (
                <Badge className="bg-green-500 text-white">
                  <CheckCircle className="w-3 h-3 mr-1" />
                  Объединено
                </Badge>
              )}
            </div>
            <CardTitle className="text-lg">
              Группа дубликатов #{group.id}
            </CardTitle>
            <CardDescription>
              Создано: {new Date(group.created_at).toLocaleString('ru-RU')}
              {group.merged_at && ` • Объединено: ${new Date(group.merged_at).toLocaleString('ru-RU')}`}
            </CardDescription>
          </div>
          {!group.merged && (
            <Button
              size="sm"
              onClick={() => onMerge(group.id)}
              disabled={mergingId === group.id}
              className="bg-green-600 hover:bg-green-700"
            >
              <GitMerge className="w-4 h-4 mr-2" />
              {mergingId === group.id ? 'Объединение...' : 'Объединить'}
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Master Record */}
        {masterItem && (
          <div className="bg-yellow-50 border-2 border-yellow-300 rounded-lg p-4">
            <div className="flex items-center gap-2 mb-3">
              <Star className="w-5 h-5 text-yellow-600 fill-yellow-600" />
              <h4 className="font-semibold text-yellow-900">Рекомендуемая мастер-запись</h4>
            </div>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-sm">
              <div>
                <span className="text-muted-foreground">ID:</span>{' '}
                <span className="font-mono">#{masterItem.id}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Код:</span>{' '}
                <code className="bg-yellow-100 px-2 py-0.5 rounded">
                  {masterItem.code || 'N/A'}
                </code>
              </div>
              <div className="md:col-span-2">
                <span className="text-muted-foreground">Название:</span>{' '}
                <span className="font-medium">{masterItem.normalized_name}</span>
              </div>
              <div>
                <span className="text-muted-foreground">Категория:</span>{' '}
                <span>{masterItem.category}</span>
              </div>
              <div>
                <span className="text-muted-foreground">КПВЭД:</span>{' '}
                <code className="bg-yellow-100 px-2 py-0.5 rounded">
                  {masterItem.kpved_code || 'N/A'}
                </code>
              </div>
              <div className="flex items-center gap-2">
                {getQualityBadge(masterItem.quality_score)}
                {getProcessingLevelBadge(masterItem.processing_level)}
              </div>
              {masterItem.merged_count > 0 && (
                <div>
                  <Badge variant="outline" className="bg-blue-50">
                    <TrendingUp className="w-3 h-3 mr-1" />
                    {masterItem.merged_count} объединений
                  </Badge>
                </div>
              )}
            </div>
          </div>
        )}

        {/* Duplicate Items */}
        <div>
          <h4 className="font-semibold mb-3 text-sm text-muted-foreground">
            Все записи в группе:
          </h4>
          <div className="space-y-2">
            {group.items?.map((item) => (
              <div
                key={item.id}
                className={`border rounded-lg p-3 ${
                  item.id === group.suggested_master_id
                    ? 'bg-yellow-50 border-yellow-200'
                    : 'bg-white'
                }`}
              >
                <div className="grid grid-cols-1 md:grid-cols-3 gap-2 text-sm">
                  <div className="md:col-span-3 flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-mono text-xs text-muted-foreground">
                          #{item.id}
                        </span>
                        {item.id === group.suggested_master_id && (
                          <Star className="w-4 h-4 text-yellow-600 fill-yellow-600" />
                        )}
                      </div>
                      <div className="font-medium">{item.normalized_name}</div>
                    </div>
                  </div>
                  <div>
                    <span className="text-muted-foreground text-xs">Код:</span>{' '}
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                      {item.code || 'N/A'}
                    </code>
                  </div>
                  <div>
                    <span className="text-muted-foreground text-xs">КПВЭД:</span>{' '}
                    <code className="text-xs bg-muted px-1.5 py-0.5 rounded">
                      {item.kpved_code || 'N/A'}
                    </code>
                  </div>
                  <div className="flex items-center gap-2">
                    {getQualityBadge(item.quality_score)}
                    {getProcessingLevelBadge(item.processing_level)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      </CardContent>
    </Card>
  )
})

DuplicateGroupCard.displayName = 'DuplicateGroupCard'

export default DuplicateGroupCard


