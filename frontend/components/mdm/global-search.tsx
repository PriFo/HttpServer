'use client'

import React, { useState, useCallback, useMemo } from 'react'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Badge } from '@/components/ui/badge'
import { Search, Package, Users, FileText, Loader2 } from 'lucide-react'
import { useRouter } from 'next/navigation'
import { useProjectState } from '@/hooks/useProjectState'
import { SkeletonLoader } from '@/components/common/skeleton-loader'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'

interface GlobalSearchProps {
  clientId: string
  projectId: string
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

interface SearchResult {
  id: string
  type: 'group' | 'item' | 'nomenclature' | 'counterparty'
  title: string
  description?: string
  category?: string
  url: string
}

async function fetchSearchResults(
  clientId: string,
  projectId: string,
  query: string,
  signal?: AbortSignal
): Promise<SearchResult[]> {
  if (!query.trim()) return []

  const response = await fetch(
    `/api/clients/${clientId}/projects/${projectId}/normalization/groups?search=${encodeURIComponent(query)}&limit=10`,
    { cache: 'no-store', signal }
  )

  if (!response.ok) {
    if (response.status === 404) return []
    throw new Error(`Search failed: ${response.status}`)
  }

  const data = await response.json()
  const results: SearchResult[] = []

  if (data.groups) {
    data.groups.forEach((group: any) => {
      results.push({
        id: `group-${group.normalized_reference}`,
        type: 'group',
        title: group.normalized_name,
        description: group.category,
        category: group.category,
        url: `/clients/${clientId}?tab=normalization&group=${encodeURIComponent(group.normalized_reference)}`,
      })
    })
  }

  return results
}

export const GlobalSearch: React.FC<GlobalSearchProps> = ({
  clientId,
  projectId,
  open: controlledOpen,
  onOpenChange,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const [internalOpen, setInternalOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const router = useRouter()

  if (!effectiveClientId || !effectiveProjectId) {
    return null
  }

  const isOpen = controlledOpen !== undefined ? controlledOpen : internalOpen
  const setIsOpen = onOpenChange || setInternalOpen

  const { data: results, loading } = useProjectState(
    (cid, pid, signal) => fetchSearchResults(cid, pid, searchQuery, signal),
    effectiveClientId || '',
    effectiveProjectId || '',
    [searchQuery],
    {
      enabled: isOpen && !!searchQuery.trim() && !!effectiveClientId && !!effectiveProjectId,
      refetchInterval: null,
    }
  )

  const handleSelect = useCallback((result: SearchResult) => {
    router.push(result.url)
    setIsOpen(false)
    setSearchQuery('')
  }, [router, setIsOpen])

  const searchResults = useMemo(() => results || [], [results])

  const getIcon = (type: string) => {
    switch (type) {
      case 'group':
      case 'nomenclature':
        return <Package className="h-4 w-4" />
      case 'counterparty':
        return <Users className="h-4 w-4" />
      default:
        return <FileText className="h-4 w-4" />
    }
  }

  return (
    <Dialog open={isOpen} onOpenChange={setIsOpen}>
      <DialogContent className="p-0 max-w-2xl">
        <Command shouldFilter={false}>
          <CommandInput
            placeholder="Поиск групп, номенклатуры, контрагентов..."
            value={searchQuery}
            onValueChange={setSearchQuery}
          />
          <CommandList>
            {loading && searchQuery ? (
              <div className="p-4 text-center text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin mx-auto mb-2" />
                Поиск...
              </div>
            ) : searchResults.length === 0 && searchQuery ? (
              <CommandEmpty>Ничего не найдено</CommandEmpty>
            ) : searchResults.length > 0 ? (
              <CommandGroup heading="Результаты">
                {searchResults.map((result) => (
                  <CommandItem
                    key={result.id}
                    value={result.id}
                    onSelect={() => handleSelect(result)}
                    className="cursor-pointer"
                  >
                    <div className="flex items-center gap-3 flex-1">
                      {getIcon(result.type)}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2">
                          <span className="font-medium truncate">{result.title}</span>
                          {result.category && (
                            <Badge variant="outline" className="text-xs">
                              {result.category}
                            </Badge>
                          )}
                        </div>
                        {result.description && (
                          <p className="text-xs text-muted-foreground truncate">
                            {result.description}
                          </p>
                        )}
                      </div>
                    </div>
                  </CommandItem>
                ))}
              </CommandGroup>
            ) : null}
          </CommandList>
        </Command>
      </DialogContent>
    </Dialog>
  )
}

