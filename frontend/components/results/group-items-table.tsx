'use client'

import { useState, useEffect, useRef } from 'react'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Button } from "@/components/ui/button"
import { ScrollArea } from "@/components/ui/scroll-area"
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible"
import { CopyIcon, CheckIcon, ChevronDownIcon, ChevronUpIcon } from "@radix-ui/react-icons"
import { Badge } from "@/components/ui/badge"
import { ConfidenceBadge } from './confidence-badge'
import { ProcessingLevelBadge } from './processing-level-badge'

interface ItemAttribute {
  id: number
  attribute_type: string
  attribute_name: string
  attribute_value: string
  unit?: string
  original_text?: string
  confidence?: number
}

interface GroupItem {
  id: number
  code: string
  source_name: string
  source_reference: string
  created_at: string
  ai_confidence?: number
  ai_reasoning?: string
  processing_level?: string
  attributes?: ItemAttribute[]
}

interface GroupItemsTableProps {
  items: GroupItem[]
  loading?: boolean
}

export function GroupItemsTable({ items, loading }: GroupItemsTableProps) {
  const [copiedCode, setCopiedCode] = useState<string | null>(null)
  const [sortField, setSortField] = useState<'code' | 'source_name' | 'created_at'>('code')
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [expandedRows, setExpandedRows] = useState<Set<number>>(new Set())
  const timeoutsRef = useRef<Set<NodeJS.Timeout>>(new Set())

  // Cleanup all timeouts on unmount
  useEffect(() => {
    return () => {
      timeoutsRef.current.forEach(clearTimeout)
    }
  }, [])

  const copyToClipboard = async (text: string, code: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedCode(code)
      const timeoutId = setTimeout(() => setCopiedCode(null), 2000)
      timeoutsRef.current.add(timeoutId)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  const toggleRowExpansion = (id: number) => {
    const newExpanded = new Set(expandedRows)
    if (newExpanded.has(id)) {
      newExpanded.delete(id)
    } else {
      newExpanded.add(id)
    }
    setExpandedRows(newExpanded)
  }

  const sortedItems = [...items].sort((a, b) => {
    let aValue: string | number = a[sortField]
    let bValue: string | number = b[sortField]

    if (sortField === 'created_at') {
      aValue = new Date(aValue).getTime()
      bValue = new Date(bValue).getTime()
    }

    if (aValue < bValue) return sortDirection === 'asc' ? -1 : 1
    if (aValue > bValue) return sortDirection === 'asc' ? 1 : -1
    return 0
  })

  const handleSort = (field: 'code' | 'source_name' | 'created_at') => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortDirection('asc')
    }
  }

  const getSortIcon = (field: string) => {
    if (sortField !== field) return null
    return sortDirection === 'asc' ? '↑' : '↓'
  }

  if (loading) {
    return (
      <div className="text-center py-8" role="status" aria-live="polite">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 mx-auto" aria-hidden="true"></div>
        <p className="mt-4 text-muted-foreground">Загрузка данных...</p>
      </div>
    )
  }

  if (items.length === 0) {
    return (
      <div className="text-center py-8 text-muted-foreground" role="status">
        Элементы не найдены
      </div>
    )
  }

  return (
    <div className="rounded-md border overflow-hidden">
      <ScrollArea className="max-h-[60vh]">
        <Table>
          <TableHeader className="sticky top-0 bg-background z-10">
          <TableRow>
            <TableHead className="w-[50px]"></TableHead>
            <TableHead
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => handleSort('code')}
            >
              Код {getSortIcon('code')}
            </TableHead>
            <TableHead
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => handleSort('source_name')}
            >
              Исходное название {getSortIcon('source_name')}
            </TableHead>
            <TableHead>Reference</TableHead>
            <TableHead>AI Confidence</TableHead>
            <TableHead>Processing</TableHead>
            <TableHead
              className="cursor-pointer hover:bg-muted/50"
              onClick={() => handleSort('created_at')}
            >
              Дата {getSortIcon('created_at')}
            </TableHead>
            <TableHead className="w-[100px]">Действия</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {sortedItems.map((item) => {
            const isExpanded = expandedRows.has(item.id)
            const hasReasoning = item.ai_reasoning && item.ai_reasoning.trim().length > 0
            const hasAttributes = item.attributes && item.attributes.length > 0
            const canExpand = hasReasoning || hasAttributes

            return (
              <>
                <TableRow key={item.id}>
                  <TableCell>
                    {canExpand && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => toggleRowExpansion(item.id)}
                        aria-label={isExpanded ? 'Скрыть детали' : 'Показать детали'}
                        aria-expanded={isExpanded}
                      >
                        {isExpanded ? (
                          <ChevronUpIcon className="h-4 w-4" />
                        ) : (
                          <ChevronDownIcon className="h-4 w-4" />
                        )}
                      </Button>
                    )}
                  </TableCell>
                  <TableCell className="font-mono text-sm">
                    <div className="flex items-center gap-2">
                      {item.code}
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-6 w-6"
                        onClick={() => copyToClipboard(item.code, item.code)}
                        aria-label={copiedCode === item.code ? 'Код скопирован' : `Копировать код ${item.code}`}
                      >
                        {copiedCode === item.code ? (
                          <CheckIcon className="h-3 w-3 text-green-600" />
                        ) : (
                          <CopyIcon className="h-3 w-3" />
                        )}
                      </Button>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="max-w-[300px] truncate" title={item.source_name}>
                      {item.source_name}
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="max-w-[200px] text-sm truncate text-muted-foreground" title={item.source_reference}>
                      {item.source_reference}
                    </div>
                  </TableCell>
                  <TableCell>
                    <ConfidenceBadge confidence={item.ai_confidence} size="sm" />
                  </TableCell>
                  <TableCell>
                    <ProcessingLevelBadge level={item.processing_level} />
                  </TableCell>
                  <TableCell>
                    <div className="text-sm text-muted-foreground">
                      {new Date(item.created_at).toLocaleDateString()}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => copyToClipboard(item.source_reference, `ref-${item.id}`)}
                      aria-label={copiedCode === `ref-${item.id}` ? 'Reference скопирован' : 'Копировать reference'}
                    >
                      {copiedCode === `ref-${item.id}` ? (
                        <CheckIcon className="h-4 w-4 text-green-600" />
                      ) : (
                        <span className="text-xs">Copy Ref</span>
                      )}
                    </Button>
                  </TableCell>
                </TableRow>

                {/* Expandable Details Row */}
                {isExpanded && (hasReasoning || hasAttributes) && (
                  <TableRow key={`${item.id}-details`}>
                    <TableCell colSpan={8} className="bg-muted/30">
                      <div className="p-4 space-y-4">
                        {/* AI Reasoning */}
                        {hasReasoning && (
                          <div className="space-y-2">
                            <h4 className="text-sm font-medium">AI обоснование:</h4>
                            <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                              {item.ai_reasoning}
                            </p>
                          </div>
                        )}
                        
                        {/* Attributes */}
                        {hasAttributes && (
                          <div className="space-y-2">
                            <h4 className="text-sm font-medium">Извлеченные реквизиты:</h4>
                            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
                              {item.attributes!.map((attr) => (
                                <div
                                  key={attr.id}
                                  className="bg-background border rounded p-2 text-sm"
                                >
                                  <div className="flex items-center justify-between mb-1">
                                    <span className="font-medium text-xs text-muted-foreground uppercase">
                                      {attr.attribute_name}
                                    </span>
                                    {attr.confidence !== undefined && attr.confidence < 1.0 && (
                                      <span className="text-xs text-muted-foreground">
                                        {(attr.confidence * 100).toFixed(0)}%
                                      </span>
                                    )}
                                  </div>
                                  <div className="flex items-baseline gap-1">
                                    <span className="font-semibold">{attr.attribute_value}</span>
                                    {attr.unit && (
                                      <span className="text-xs text-muted-foreground">{attr.unit}</span>
                                    )}
                                  </div>
                                  {attr.original_text && (
                                    <div className="text-xs text-muted-foreground mt-1">
                                      Из: "{attr.original_text}"
                                    </div>
                                  )}
                                  <div className="text-xs text-muted-foreground mt-1">
                                    <Badge variant="outline" className="text-xs">
                                      {attr.attribute_type}
                                    </Badge>
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </>
            )
          })}
        </TableBody>
      </Table>
      </ScrollArea>
    </div>
  )
}
