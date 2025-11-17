'use client'

import { useState, useMemo, ReactNode } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-react'
import { cn } from '@/lib/utils'

export type SortDirection = 'asc' | 'desc' | null

export interface Column<T> {
  key: string
  header: string
  accessor?: (row: T) => any
  render?: (row: T) => ReactNode
  sortable?: boolean
  align?: 'left' | 'center' | 'right'
  className?: string
  headerClassName?: string
}

export interface DataTableProps<T> {
  data: T[]
  columns: Column<T>[]
  loading?: boolean
  emptyMessage?: string
  emptyIcon?: ReactNode
  onRowClick?: (row: T) => void
  rowClassName?: (row: T) => string
  keyExtractor?: (row: T, index: number) => string
  sortable?: boolean
  defaultSort?: {
    key: string
    direction: SortDirection
  }
  onSortChange?: (key: string, direction: SortDirection) => void
  selectable?: boolean
  selectedRows?: T[]
  onSelectionChange?: (rows: T[]) => void
  getRowId?: (row: T) => string | number
  className?: string
}

export function DataTable<T extends Record<string, any>>({
  data,
  columns,
  loading = false,
  emptyMessage = 'Нет данных для отображения',
  emptyIcon,
  onRowClick,
  rowClassName,
  keyExtractor,
  sortable = true,
  defaultSort,
  onSortChange,
  selectable = false,
  selectedRows = [],
  onSelectionChange,
  getRowId,
  className,
}: DataTableProps<T>) {
  const [sortKey, setSortKey] = useState<string | null>(defaultSort?.key || null)
  const [sortDirection, setSortDirection] = useState<SortDirection>(defaultSort?.direction || null)

  const handleSort = (key: string) => {
    if (!sortable) return

    const column = columns.find((col) => col.key === key)
    if (!column?.sortable) return

    let newDirection: SortDirection = 'asc'
    if (sortKey === key) {
      if (sortDirection === 'asc') {
        newDirection = 'desc'
      } else if (sortDirection === 'desc') {
        newDirection = null
      }
    }

    setSortKey(newDirection ? key : null)
    setSortDirection(newDirection)

    if (onSortChange) {
      onSortChange(key, newDirection)
    }
  }

  const sortedData = useMemo(() => {
    if (!sortKey || !sortDirection) return data

    const column = columns.find((col) => col.key === sortKey)
    if (!column) return data

    return [...data].sort((a, b) => {
      const aValue = column.accessor ? column.accessor(a) : a[sortKey]
      const bValue = column.accessor ? column.accessor(b) : b[sortKey]

      // Handle null/undefined values
      if (aValue == null && bValue == null) return 0
      if (aValue == null) return 1
      if (bValue == null) return -1

      // Compare values
      let comparison = 0
      if (typeof aValue === 'string' && typeof bValue === 'string') {
        comparison = aValue.localeCompare(bValue, 'ru-RU', { numeric: true, sensitivity: 'base' })
      } else if (typeof aValue === 'number' && typeof bValue === 'number') {
        comparison = aValue - bValue
      } else {
        comparison = String(aValue).localeCompare(String(bValue), 'ru-RU', { numeric: true })
      }

      return sortDirection === 'asc' ? comparison : -comparison
    })
  }, [data, sortKey, sortDirection, columns])

  const handleRowSelection = (row: T, checked: boolean) => {
    if (!selectable || !onSelectionChange || !getRowId) return

    const rowId = getRowId(row)
    if (checked) {
      onSelectionChange([...selectedRows, row])
    } else {
      onSelectionChange(selectedRows.filter((r) => getRowId(r) !== rowId))
    }
  }

  const isRowSelected = (row: T): boolean => {
    if (!selectable || !getRowId) return false
    const rowId = getRowId(row)
    return selectedRows.some((r) => getRowId(r) === rowId)
  }

  const getSortIcon = (key: string) => {
    const column = columns.find((col) => col.key === key)
    if (!column?.sortable) return null

    if (sortKey !== key) {
      return <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
    }

    if (sortDirection === 'asc') {
      return <ArrowUp className="ml-2 h-4 w-4" />
    }

    if (sortDirection === 'desc') {
      return <ArrowDown className="ml-2 h-4 w-4" />
    }

    return <ArrowUpDown className="ml-2 h-4 w-4 opacity-50" />
  }

  const getRowKey = (row: T, index: number): string => {
    if (keyExtractor) {
      return keyExtractor(row, index)
    }
    if (getRowId) {
      return String(getRowId(row))
    }
    return `row-${index}`
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-center">
          <div className="h-8 w-8 border-4 border-primary border-t-transparent rounded-full animate-spin mx-auto mb-4" />
          <p className="text-sm text-muted-foreground">Загрузка данных...</p>
        </div>
      </div>
    )
  }

  if (sortedData.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        {emptyIcon || (
          <div className="h-12 w-12 rounded-full bg-muted flex items-center justify-center mb-4">
            <svg
              className="h-6 w-6 text-muted-foreground"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
              />
            </svg>
          </div>
        )}
        <h3 className="text-lg font-semibold mb-1">{emptyMessage}</h3>
      </div>
    )
  }

  return (
    <div className={cn('rounded-md border', className)}>
      <Table>
        <TableHeader>
          <TableRow>
            {selectable && (
              <TableHead className="w-12">
                <input
                  type="checkbox"
                  className="rounded border-gray-300"
                  checked={selectedRows.length === sortedData.length && sortedData.length > 0}
                  onChange={(e) => {
                    if (onSelectionChange) {
                      onSelectionChange(e.target.checked ? sortedData : [])
                    }
                  }}
                  aria-label="Выбрать все"
                />
              </TableHead>
            )}
            {columns.map((column) => (
              <TableHead
                key={column.key}
                className={cn(
                  column.align === 'right' && 'text-right',
                  column.align === 'center' && 'text-center',
                  column.headerClassName
                )}
              >
                {column.sortable !== false && sortable ? (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-8 -ml-3 hover:bg-transparent"
                    onClick={() => handleSort(column.key)}
                  >
                    <span className="font-medium">{column.header}</span>
                    {getSortIcon(column.key)}
                  </Button>
                ) : (
                  <span className="font-medium">{column.header}</span>
                )}
              </TableHead>
            ))}
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedData.map((row, index) => {
            const isSelected = isRowSelected(row)
            const rowKey = getRowKey(row, index)
            const customClassName = rowClassName ? rowClassName(row) : ''

            return (
              <TableRow
                key={rowKey}
                className={cn(
                  onRowClick && 'cursor-pointer hover:bg-muted/50',
                  isSelected && 'bg-muted',
                  customClassName
                )}
                onClick={() => onRowClick?.(row)}
                onKeyDown={(e) => {
                  if (onRowClick && (e.key === 'Enter' || e.key === ' ')) {
                    e.preventDefault()
                    onRowClick(row)
                  }
                }}
                tabIndex={onRowClick ? 0 : undefined}
                role={onRowClick ? 'button' : undefined}
              >
                {selectable && (
                  <TableCell>
                    <input
                      type="checkbox"
                      className="rounded border-gray-300"
                      checked={isSelected}
                      onChange={(e) => handleRowSelection(row, e.target.checked)}
                      onClick={(e) => e.stopPropagation()}
                      aria-label={`Выбрать строку ${index + 1}`}
                    />
                  </TableCell>
                )}
                {columns.map((column) => {
                  const value = column.accessor ? column.accessor(row) : row[column.key]
                  const content = column.render ? column.render(row) : value

                  return (
                    <TableCell
                      key={column.key}
                      className={cn(
                        column.align === 'right' && 'text-right',
                        column.align === 'center' && 'text-center',
                        column.className
                      )}
                    >
                      {content}
                    </TableCell>
                  )
                })}
              </TableRow>
            )
          })}
        </TableBody>
      </Table>
    </div>
  )
}

