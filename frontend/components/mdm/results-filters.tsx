'use client'

import React, { useState } from 'react'
import { Card, CardContent } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { X, Filter } from 'lucide-react'

interface FilterState {
  category?: string
  confidence?: number
  hasAttributes?: boolean
  kpvedCode?: string
  search?: string
}

interface ResultsFiltersProps {
  filters: FilterState
  onFiltersChange: (filters: FilterState) => void
}

export const ResultsFilters: React.FC<ResultsFiltersProps> = ({
  filters,
  onFiltersChange,
}) => {
  const [localFilters, setLocalFilters] = useState<FilterState>(filters)
  const [isExpanded, setIsExpanded] = useState(false)

  const activeFiltersCount = Object.values(localFilters).filter(v => v !== undefined && v !== '').length

  const handleFilterChange = (key: keyof FilterState, value: any) => {
    const newFilters = { ...localFilters, [key]: value }
    setLocalFilters(newFilters)
    onFiltersChange(newFilters)
  }

  const clearFilter = (key: keyof FilterState) => {
    const newFilters = { ...localFilters, [key]: undefined }
    setLocalFilters(newFilters)
    onFiltersChange(newFilters)
  }

  const clearAll = () => {
    const emptyFilters: FilterState = {}
    setLocalFilters(emptyFilters)
    onFiltersChange(emptyFilters)
  }

  return (
    <Card>
      <CardContent className="p-4">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-2">
            <Filter className="h-4 w-4" />
            <span className="font-medium">Фильтры</span>
            {activeFiltersCount > 0 && (
              <Badge variant="secondary">{activeFiltersCount}</Badge>
            )}
          </div>
          <div className="flex gap-2">
            {activeFiltersCount > 0 && (
              <Button variant="ghost" size="sm" onClick={clearAll}>
                <X className="h-4 w-4 mr-1" />
                Очистить все
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={() => setIsExpanded(!isExpanded)}
            >
              {isExpanded ? 'Свернуть' : 'Развернуть'}
            </Button>
          </div>
        </div>

        {isExpanded && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div className="space-y-2">
              <Label>Поиск</Label>
              <Input
                placeholder="Название, код..."
                value={localFilters.search || ''}
                onChange={(e) => handleFilterChange('search', e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <Label>Категория</Label>
              <Select
                value={localFilters.category || 'all'}
                onValueChange={(v) => handleFilterChange('category', v === 'all' ? undefined : v)}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Все категории" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все категории</SelectItem>
                  <SelectItem value="electronics">Электроника</SelectItem>
                  <SelectItem value="machinery">Оборудование</SelectItem>
                  <SelectItem value="materials">Материалы</SelectItem>
                  <SelectItem value="services">Услуги</SelectItem>
                </SelectContent>
              </Select>
              {localFilters.category && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => clearFilter('category')}
                  className="h-6 text-xs"
                >
                  <X className="h-3 w-3 mr-1" />
                  Очистить
                </Button>
              )}
            </div>

            <div className="space-y-2">
              <Label>Уверенность (мин. %)</Label>
              <Input
                type="number"
                min="0"
                max="100"
                value={localFilters.confidence || ''}
                onChange={(e) => handleFilterChange('confidence', e.target.value ? parseInt(e.target.value) : undefined)}
                placeholder="0-100"
              />
              {localFilters.confidence !== undefined && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => clearFilter('confidence')}
                  className="h-6 text-xs"
                >
                  <X className="h-3 w-3 mr-1" />
                  Очистить
                </Button>
              )}
            </div>

            <div className="space-y-2">
              <Label>КПВЭД</Label>
              <Input
                placeholder="Код КПВЭД"
                value={localFilters.kpvedCode || ''}
                onChange={(e) => handleFilterChange('kpvedCode', e.target.value || undefined)}
              />
              {localFilters.kpvedCode && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => clearFilter('kpvedCode')}
                  className="h-6 text-xs"
                >
                  <X className="h-3 w-3 mr-1" />
                  Очистить
                </Button>
              )}
            </div>
          </div>
        )}

        {/* Активные фильтры */}
        {activeFiltersCount > 0 && (
          <div className="mt-4 pt-4 border-t">
            <div className="flex flex-wrap gap-2">
              {localFilters.search && (
                <Badge variant="secondary" className="flex items-center gap-1">
                  Поиск: {localFilters.search}
                  <X
                    className="h-3 w-3 cursor-pointer"
                    onClick={() => clearFilter('search')}
                  />
                </Badge>
              )}
              {localFilters.category && (
                <Badge variant="secondary" className="flex items-center gap-1">
                  Категория: {localFilters.category}
                  <X
                    className="h-3 w-3 cursor-pointer"
                    onClick={() => clearFilter('category')}
                  />
                </Badge>
              )}
              {localFilters.confidence !== undefined && (
                <Badge variant="secondary" className="flex items-center gap-1">
                  Уверенность: ≥{localFilters.confidence}%
                  <X
                    className="h-3 w-3 cursor-pointer"
                    onClick={() => clearFilter('confidence')}
                  />
                </Badge>
              )}
              {localFilters.kpvedCode && (
                <Badge variant="secondary" className="flex items-center gap-1">
                  КПВЭД: {localFilters.kpvedCode}
                  <X
                    className="h-3 w-3 cursor-pointer"
                    onClick={() => clearFilter('kpvedCode')}
                  />
                </Badge>
              )}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

