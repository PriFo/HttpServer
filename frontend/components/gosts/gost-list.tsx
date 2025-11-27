'use client'

import { memo } from 'react'
import { GostCard } from './gost-card'
import { GostSkeleton } from './gost-skeleton'
import { EmptyState } from '@/components/common/empty-state'
import { FileText } from 'lucide-react'

interface Gost {
  id: number
  gost_number: string
  title: string
  adoption_date?: string
  effective_date?: string
  status?: string
  source_type?: string
  source_url?: string
  description?: string
  keywords?: string
}

interface GostListProps {
  gosts: Gost[]
  loading: boolean
  error: string | null
  viewMode?: 'grid' | 'list'
}

export const GostList = memo(function GostList({ gosts, loading, error, viewMode = 'grid' }: GostListProps) {
  if (loading) {
    if (viewMode === 'list') {
      return (
        <div className="space-y-2">
          {[...Array(5)].map((_, i) => (
            <GostSkeleton key={i} compact />
          ))}
        </div>
      )
    }
    return (
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {[...Array(6)].map((_, i) => (
          <GostSkeleton key={i} />
        ))}
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-center py-8">
        <div className="bg-destructive/10 border border-destructive/20 rounded-lg p-4 max-w-md mx-auto">
          <p className="text-destructive font-medium mb-2">Ошибка загрузки</p>
          <p className="text-sm text-muted-foreground">{error}</p>
        </div>
      </div>
    )
  }

  if (gosts.length === 0) {
    return (
      <EmptyState
        icon={FileText}
        title="ГОСТы не найдены"
        description="Попробуйте изменить параметры поиска или фильтры"
      />
    )
  }

  if (viewMode === 'list') {
    return (
      <div className="space-y-2">
        {gosts.map((gost) => (
          <GostCard key={gost.id} gost={gost} compact />
        ))}
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
      {gosts.map((gost) => (
        <GostCard key={gost.id} gost={gost} />
      ))}
    </div>
  )
})

