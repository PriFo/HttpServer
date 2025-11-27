'use client'

import { memo, useState, useEffect, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Calendar, ExternalLink, ArrowRight, Star } from 'lucide-react'
import Link from 'next/link'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale/ru'
import { isFavorite, toggleFavorite } from '@/lib/gost-favorites'
import { toast } from 'sonner'

interface GostCardProps {
  gost: {
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
  compact?: boolean
}

export const GostCard = memo(function GostCard({ gost, compact = false }: GostCardProps) {
  const [favorite, setFavorite] = useState(false)

  useEffect(() => {
    setFavorite(isFavorite(gost.id))
  }, [gost.id])

  const handleToggleFavorite = useCallback((e: React.MouseEvent) => {
    e.preventDefault()
    e.stopPropagation()
    const newFavorite = toggleFavorite(gost)
    setFavorite(newFavorite)
    toast.success(newFavorite ? 'Добавлено в избранное' : 'Удалено из избранного', {
      description: `ГОСТ ${gost.gost_number}`,
      duration: 2000,
    })
  }, [gost])

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return null
    try {
      const date = new Date(dateStr)
      return format(date, 'dd.MM.yyyy', { locale: ru })
    } catch {
      return dateStr
    }
  }

  const getStatusColor = useCallback((status?: string) => {
    if (!status) return 'secondary'
    const statusLower = status.toLowerCase()
    if (statusLower.includes('действующий') || statusLower.includes('действует')) {
      return 'default'
    }
    if (statusLower.includes('отменен') || statusLower.includes('отменён')) {
      return 'destructive'
    }
    if (statusLower.includes('заменен') || statusLower.includes('заменён')) {
      return 'outline'
    }
    return 'secondary'
  }, [])

  if (compact) {
    return (
      <Card className="hover:shadow-md transition-shadow">
        <CardContent className="p-4">
          <div className="flex items-center justify-between gap-4">
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-3">
                <CardTitle className="text-base font-mono wrap-break-word">
                  {gost.gost_number}
                </CardTitle>
                {gost.status && (
                  <Badge variant={getStatusColor(gost.status)} className="shrink-0 text-xs">
                    {gost.status}
                  </Badge>
                )}
              </div>
              <CardDescription className="mt-1 line-clamp-1">
                {gost.title}
              </CardDescription>
              <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                {gost.adoption_date && (
                  <div className="flex items-center gap-1">
                    <Calendar className="h-3 w-3" />
                    <span>Принят: {formatDate(gost.adoption_date)}</span>
                  </div>
                )}
                {gost.source_type && (
                  <Badge variant="outline" className="text-xs">
                    {gost.source_type}
                  </Badge>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Link href={`/gosts/${gost.id}`}>
                <Button variant="outline" size="sm" className="gap-2 shrink-0">
                  Подробнее
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </Link>
              <Button
                variant="ghost"
                size="sm"
                onClick={(e) => {
                  e.preventDefault()
                  e.stopPropagation()
                  const newFavorite = toggleFavorite(gost)
                  setFavorite(newFavorite)
                  toast.success(newFavorite ? 'Добавлено в избранное' : 'Удалено из избранного', {
                    description: `ГОСТ ${gost.gost_number}`,
                    duration: 2000,
                  })
                }}
                className={`gap-2 shrink-0 ${favorite ? 'text-yellow-500 hover:text-yellow-600' : ''}`}
                title={favorite ? 'Удалить из избранного' : 'Добавить в избранное'}
              >
                <Star className={`h-4 w-4 ${favorite ? 'fill-current' : ''}`} />
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader>
        <div className="flex items-start justify-between gap-4">
          <div className="flex-1 min-w-0">
            <CardTitle className="text-lg font-mono mb-2 wrap-break-word">
              {gost.gost_number}
            </CardTitle>
            <CardDescription className="line-clamp-2">
              {gost.title}
            </CardDescription>
          </div>
          {gost.status && (
            <Badge variant={getStatusColor(gost.status)} className="shrink-0">
              {gost.status}
            </Badge>
          )}
        </div>
      </CardHeader>
      <CardContent>
        <div className="space-y-3">
          {(gost.adoption_date || gost.effective_date) && (
            <div className="flex flex-wrap gap-4 text-sm text-muted-foreground">
              {gost.adoption_date && (
                <div className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  <span>Принят: {formatDate(gost.adoption_date)}</span>
                </div>
              )}
              {gost.effective_date && (
                <div className="flex items-center gap-1">
                  <Calendar className="h-4 w-4" />
                  <span>Вступил: {formatDate(gost.effective_date)}</span>
                </div>
              )}
            </div>
          )}

          {gost.source_type && (
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="text-xs">
                {gost.source_type}
              </Badge>
              {gost.source_url && (
                <a
                  href={gost.source_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs text-muted-foreground hover:text-primary flex items-center gap-1"
                >
                  <ExternalLink className="h-3 w-3" />
                  Источник
                </a>
              )}
            </div>
          )}

          {gost.keywords && (
            <div className="flex flex-wrap gap-1">
              {gost.keywords.split(',').slice(0, 3).map((keyword, idx) => (
                <Badge key={idx} variant="secondary" className="text-xs">
                  {keyword.trim()}
                </Badge>
              ))}
              {gost.keywords.split(',').length > 3 && (
                <Badge variant="secondary" className="text-xs">
                  +{gost.keywords.split(',').length - 3}
                </Badge>
              )}
            </div>
          )}

          <div className="flex items-center justify-between pt-2">
            <Link href={`/gosts/${gost.id}`}>
              <Button variant="outline" size="sm" className="gap-2">
                Подробнее
                <ArrowRight className="h-4 w-4" />
              </Button>
            </Link>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleToggleFavorite}
              className={`gap-2 ${favorite ? 'text-yellow-500 hover:text-yellow-600' : ''}`}
              title={favorite ? 'Удалить из избранного' : 'Добавить в избранное'}
            >
              <Star className={`h-4 w-4 ${favorite ? 'fill-current' : ''}`} />
            </Button>
          </div>
        </div>
      </CardContent>
    </Card>
  )
})


