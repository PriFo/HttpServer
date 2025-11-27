'use client'

import * as React from 'react'
import { Button } from '@/components/ui/button'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { cn } from '@/lib/utils'

export interface PaginationProps {
  currentPage: number
  totalPages: number
  onPageChange: (page: number) => void
  itemsPerPage?: number
  totalItems?: number
  showInfo?: boolean
  maxVisiblePages?: number
  className?: string
}

export function Pagination({
  currentPage,
  totalPages,
  onPageChange,
  itemsPerPage,
  totalItems,
  showInfo = true,
  maxVisiblePages = 5,
  className,
}: PaginationProps) {
  if (totalPages <= 1) {
    return null
  }

  const getVisiblePages = () => {
    if (totalPages <= maxVisiblePages) {
      return Array.from({ length: totalPages }, (_, i) => i + 1)
    }

    if (currentPage <= 3) {
      return Array.from({ length: maxVisiblePages }, (_, i) => i + 1)
    }

    if (currentPage >= totalPages - 2) {
      return Array.from(
        { length: maxVisiblePages },
        (_, i) => totalPages - maxVisiblePages + i + 1
      )
    }

    return Array.from(
      { length: maxVisiblePages },
      (_, i) => currentPage - Math.floor(maxVisiblePages / 2) + i
    )
  }

  const visiblePages = getVisiblePages()
  const startItem = itemsPerPage && totalItems 
    ? (currentPage - 1) * itemsPerPage + 1 
    : null
  const endItem = itemsPerPage && totalItems
    ? Math.min(currentPage * itemsPerPage, totalItems)
    : null

  return (
    <div className={cn('flex items-center justify-between', className)}>
      {showInfo && startItem !== null && endItem !== null ? (
        <div className="text-sm text-muted-foreground" role="status" aria-live="polite">
          Показаны записи {startItem.toLocaleString('ru-RU')} - {endItem.toLocaleString('ru-RU')}
          {totalItems && ` из ${totalItems.toLocaleString('ru-RU')}`}
        </div>
      ) : (
        <div />
      )}
      
      <nav className="flex gap-2" aria-label="Навигация по страницам">
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(Math.max(1, currentPage - 1))}
          disabled={currentPage === 1}
          aria-label="Предыдущая страница"
        >
          <ChevronLeft className="h-4 w-4" />
          Назад
        </Button>
        
        <div className="flex items-center gap-1">
          {visiblePages.map((pageNum) => (
            <Button
              key={pageNum}
              variant={currentPage === pageNum ? 'default' : 'outline'}
              size="sm"
              onClick={() => onPageChange(pageNum)}
              aria-label={`Страница ${pageNum}`}
              aria-current={currentPage === pageNum ? 'page' : undefined}
            >
              {pageNum}
            </Button>
          ))}
        </div>
        
        <Button
          variant="outline"
          size="sm"
          onClick={() => onPageChange(Math.min(totalPages, currentPage + 1))}
          disabled={currentPage === totalPages}
          aria-label="Следующая страница"
        >
          Вперед
          <ChevronRight className="h-4 w-4" />
        </Button>
      </nav>
    </div>
  )
}

