'use client'

import * as React from 'react'
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
import { Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Badge } from '@/components/ui/badge'

export type FilterType = 'search' | 'select' | 'multiselect' | 'date' | 'checkbox'

export interface FilterOption {
  value: string
  label: string
}

export interface FilterConfig {
  type: FilterType
  key: string
  label: string
  placeholder?: string
  options?: FilterOption[]
  defaultValue?: string
  className?: string
}

export interface FilterBarProps {
  filters: FilterConfig[]
  values: Record<string, any>
  onChange: (values: Record<string, any>) => void
  onReset?: () => void
  showActiveFilters?: boolean
  className?: string
  searchPlaceholder?: string
}

export function FilterBar({
  filters,
  values,
  onChange,
  onReset,
  showActiveFilters = true,
  className,
  searchPlaceholder = 'Поиск...',
}: FilterBarProps) {
  const handleFilterChange = (key: string, value: any) => {
    onChange({
      ...values,
      [key]: value,
    })
  }

  const handleRemoveFilter = (key: string) => {
    const newValues = { ...values }
    delete newValues[key]
    onChange(newValues)
  }

  const activeFilters = Object.entries(values).filter(
    ([key, value]) => value !== '' && value !== null && value !== undefined
  )

  const hasActiveFilters = activeFilters.length > 0

  const renderFilter = (filter: FilterConfig) => {
    switch (filter.type) {
      case 'search':
        return (
          <div key={filter.key} className={cn('flex-1', filter.className)}>
            <div className="relative">
              <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={filter.placeholder || searchPlaceholder}
                value={values[filter.key] || ''}
                onChange={(e) => handleFilterChange(filter.key, e.target.value)}
                className="pl-9"
                aria-label={filter.label}
              />
            </div>
          </div>
        )

      case 'select':
        return (
          <div key={filter.key} className={cn('space-y-2', filter.className)}>
            {filter.label && <Label>{filter.label}</Label>}
            <Select
              value={values[filter.key] || filter.defaultValue || ''}
              onValueChange={(value) => handleFilterChange(filter.key, value)}
            >
              <SelectTrigger aria-label={filter.label}>
                <SelectValue placeholder={filter.placeholder || filter.label} />
              </SelectTrigger>
              <SelectContent>
                {filter.options
                  ?.filter((option) => {
                    // Фильтруем опции с пустыми значениями
                    const value = option.value !== null && option.value !== undefined 
                      ? String(option.value).trim() 
                      : ''
                    return value.length > 0
                  })
                  .map((option) => {
                    const value = String(option.value)
                    return (
                      <SelectItem key={value} value={value}>
                        {option.label}
                      </SelectItem>
                    )
                  })}
              </SelectContent>
            </Select>
          </div>
        )

      case 'checkbox':
        return (
          <div key={filter.key} className={cn('flex items-center space-x-2', filter.className)}>
            <input
              type="checkbox"
              id={filter.key}
              checked={values[filter.key] || false}
              onChange={(e) => handleFilterChange(filter.key, e.target.checked)}
              className="h-4 w-4 rounded border-gray-300"
            />
            {filter.label && (
              <Label htmlFor={filter.key} className="cursor-pointer">
                {filter.label}
              </Label>
            )}
          </div>
        )

      default:
        return null
    }
  }

  return (
    <div className={cn('space-y-4', className)}>
      <div className="flex gap-4 flex-wrap">
        {filters.map(renderFilter)}
      </div>

      {showActiveFilters && hasActiveFilters && (
        <div className="flex items-center gap-2 flex-wrap">
          <span className="text-sm text-muted-foreground">Активные фильтры:</span>
          {activeFilters.map(([key, value]) => {
            const filter = filters.find((f) => f.key === key)
            if (!filter) return null

            const displayValue =
              filter.type === 'select'
                ? filter.options?.find((opt) => opt.value === value)?.label || value
                : value

            return (
              <Badge
                key={key}
                variant="secondary"
                className="flex items-center gap-1"
              >
                {filter.label}: {displayValue}
                <button
                  className="ml-1 hover:text-destructive"
                  onClick={() => handleRemoveFilter(key)}
                  aria-label={`Удалить фильтр ${filter.label}`}
                >
                  <X className="h-3 w-3" />
                </button>
              </Badge>
            )
          })}
          {onReset && (
            <Button
              variant="ghost"
              size="sm"
              onClick={onReset}
              className="h-6 text-xs"
            >
              Сбросить все
            </Button>
          )}
        </div>
      )}
    </div>
  )
}

