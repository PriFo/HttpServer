'use client'

import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Search, X, Filter } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface GostFiltersProps {
  searchQuery: string
  onSearchChange: (value: string) => void
  statusFilter: string
  onStatusFilterChange: (value: string) => void
  sourceTypeFilter: string
  onSourceTypeFilterChange: (value: string) => void
  sourceTypes: string[]
  onClearFilters: () => void
  hasActiveFilters: boolean
  adoptionFrom: string
  adoptionTo: string
  onAdoptionFromChange: (value: string) => void
  onAdoptionToChange: (value: string) => void
  effectiveFrom: string
  effectiveTo: string
  onEffectiveFromChange: (value: string) => void
  onEffectiveToChange: (value: string) => void
}

export function GostFilters({
  searchQuery,
  onSearchChange,
  statusFilter,
  onStatusFilterChange,
  sourceTypeFilter,
  onSourceTypeFilterChange,
  sourceTypes,
  onClearFilters,
  hasActiveFilters,
  adoptionFrom,
  adoptionTo,
  onAdoptionFromChange,
  onAdoptionToChange,
  effectiveFrom,
  effectiveTo,
  onEffectiveFromChange,
  onEffectiveToChange,
}: GostFiltersProps) {
  const statusOptions = [
    { value: 'all', label: 'Все статусы' },
    { value: 'действующий', label: 'Действующий' },
    { value: 'отменен', label: 'Отменен' },
    { value: 'заменен', label: 'Заменен' },
  ]

  return (
    <div className="space-y-4">
      <div className="flex flex-col sm:flex-row gap-4">
        <div className="flex-1">
          <Label htmlFor="search">Поиск</Label>
          <div className="relative mt-1">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
            <Input
              id="search"
              placeholder="Поиск по номеру, названию, ключевым словам..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
      </div>

      <div className="flex flex-col sm:flex-row gap-4">
        <div className="flex-1">
          <Label htmlFor="status">Статус</Label>
          <Select value={statusFilter || 'all'} onValueChange={(v) => onStatusFilterChange(v === 'all' ? '' : v)}>
            <SelectTrigger id="status" className="mt-1">
              <SelectValue placeholder="Все статусы" />
            </SelectTrigger>
            <SelectContent>
              {statusOptions.map((option) => (
                <SelectItem key={option.value || 'all'} value={option.value || 'all'}>
                  {option.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex-1">
          <Label htmlFor="source-type">Тип источника</Label>
          <Select value={sourceTypeFilter || 'all'} onValueChange={(v) => onSourceTypeFilterChange(v === 'all' ? '' : v)}>
            <SelectTrigger id="source-type" className="mt-1">
              <SelectValue placeholder="Все источники" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">Все источники</SelectItem>
              {sourceTypes
                .filter((sourceType) => sourceType && sourceType.trim() !== '')
                .map((sourceType) => (
                  <SelectItem key={sourceType} value={sourceType}>
                    {sourceType}
                  </SelectItem>
                ))}
            </SelectContent>
          </Select>
        </div>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <div className="space-y-2">
          <Label>Дата принятия (ГГГГ-ММ-ДД)</Label>
          <div className="grid gap-2 sm:grid-cols-2">
            <Input
              type="date"
              value={adoptionFrom}
              onChange={(e) => onAdoptionFromChange(e.target.value)}
              placeholder="С"
            />
            <Input
              type="date"
              value={adoptionTo}
              onChange={(e) => onAdoptionToChange(e.target.value)}
              placeholder="По"
            />
          </div>
        </div>
        <div className="space-y-2">
          <Label>Дата вступления (ГГГГ-ММ-ДД)</Label>
          <div className="grid gap-2 sm:grid-cols-2">
            <Input
              type="date"
              value={effectiveFrom}
              onChange={(e) => onEffectiveFromChange(e.target.value)}
              placeholder="С"
            />
            <Input
              type="date"
              value={effectiveTo}
              onChange={(e) => onEffectiveToChange(e.target.value)}
              placeholder="По"
            />
          </div>
        </div>
      </div>

      {hasActiveFilters && (
        <div className="flex items-center gap-2 flex-wrap">
          <Filter className="h-4 w-4 text-muted-foreground" />
          <span className="text-sm text-muted-foreground">Активные фильтры:</span>
          {searchQuery && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Поиск: {searchQuery}
              <button
                onClick={() => onSearchChange('')}
                className="ml-1 hover:text-destructive"
                aria-label="Удалить фильтр поиска"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {statusFilter && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Статус: {statusOptions.find((o) => o.value === statusFilter)?.label}
              <button
                onClick={() => onStatusFilterChange('')}
                className="ml-1 hover:text-destructive"
                aria-label="Удалить фильтр статуса"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {sourceTypeFilter && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Источник: {sourceTypeFilter}
              <button
                onClick={() => onSourceTypeFilterChange('all')}
                className="ml-1 hover:text-destructive"
                aria-label="Удалить фильтр источника"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {adoptionFrom && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Принятия с: {adoptionFrom}
              <button
                onClick={() => onAdoptionFromChange('')}
                className="ml-1 hover:text-destructive"
                aria-label="Очистить начало периода принятия"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {adoptionTo && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Принятия до: {adoptionTo}
              <button
                onClick={() => onAdoptionToChange('')}
                className="ml-1 hover:text-destructive"
                aria-label="Очистить конец периода принятия"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {effectiveFrom && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Вступления с: {effectiveFrom}
              <button
                onClick={() => onEffectiveFromChange('')}
                className="ml-1 hover:text-destructive"
                aria-label="Очистить начало периода вступления"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          {effectiveTo && (
            <Badge variant="secondary" className="flex items-center gap-1">
              Вступления до: {effectiveTo}
              <button
                onClick={() => onEffectiveToChange('')}
                className="ml-1 hover:text-destructive"
                aria-label="Очистить конец периода вступления"
              >
                <X className="h-3 w-3" />
              </button>
            </Badge>
          )}
          <Button
            variant="ghost"
            size="sm"
            onClick={onClearFilters}
            className="h-6 text-xs"
          >
            Сбросить все
          </Button>
        </div>
      )}
    </div>
  )
}

