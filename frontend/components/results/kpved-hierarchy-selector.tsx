'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { handleApiError } from '@/lib/errors'
import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from '@/components/ui/command'
import { Badge } from '@/components/ui/badge'
import { Check, ChevronRight, Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'

interface KpvedItem {
  code: string
  name: string
  level: number
  parent_code?: string
}

interface KpvedHierarchySelectorProps {
  value?: string
  onChange: (code: string | null) => void
  placeholder?: string
}

export function KpvedHierarchySelector({ value, onChange, placeholder = "Выберите КПВЭД код..." }: KpvedHierarchySelectorProps) {
  const [open, setOpen] = useState(false)
  const [searchMode, setSearchMode] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<KpvedItem[]>([])
  const [hierarchyItems, setHierarchyItems] = useState<KpvedItem[]>([])
  const [currentPath, setCurrentPath] = useState<KpvedItem[]>([])
  const [selectedItem, setSelectedItem] = useState<KpvedItem | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [lastOperation, setLastOperation] = useState<{ type: 'load' | 'search', param: string | null } | null>(null)

  // Загружаем начальный уровень при открытии
  useEffect(() => {
    if (open && !searchMode && currentPath.length === 0) {
      loadLevel(null)
    }
  }, [open, searchMode])

  // Загружаем уровень иерархии
  const loadLevel = async (parent: string | null) => {
    setLoading(true)
    setError(null)
    setLastOperation({ type: 'load', param: parent })
    try {
      const url = parent
        ? `/api/kpved/hierarchy?parent=${encodeURIComponent(parent)}`
        : '/api/kpved/hierarchy'

      const response = await fetch(url)
      if (!response.ok) {
        throw new Error(`Failed to fetch hierarchy: ${response.status}`)
      }
      const data = await response.json()
      setHierarchyItems(data)
    } catch (error) {
      console.error('Error loading KPVED level:', error)
      setError(handleApiError(error, 'LOAD_KPVED_ERROR'))
      setHierarchyItems([])
    } finally {
      setLoading(false)
    }
  }

  // Поиск по КПВЭД
  const searchKpved = async (query: string) => {
    if (!query.trim()) {
      setSearchResults([])
      return
    }

    setLoading(true)
    setError(null)
    setLastOperation({ type: 'search', param: query })
    try {
      const response = await fetch(`/api/kpved/search?q=${encodeURIComponent(query)}&limit=20`)
      if (!response.ok) {
        throw new Error(`Failed to search KPVED: ${response.status}`)
      }
      const data = await response.json()
      setSearchResults(data)
    } catch (error) {
      console.error('Error searching KPVED:', error)
      setError(handleApiError(error, 'SEARCH_ERROR'))
      setSearchResults([])
    } finally {
      setLoading(false)
    }
  }

  // Обработка поиска
  useEffect(() => {
    if (searchMode && searchQuery) {
      const timer = setTimeout(() => {
        searchKpved(searchQuery)
      }, 300)
      return () => clearTimeout(timer)
    } else {
      setSearchResults([])
    }
  }, [searchQuery, searchMode])

  // Выбор элемента
  const handleSelect = (item: KpvedItem) => {
    setSelectedItem(item)
    onChange(item.code)
    setOpen(false)
    setSearchMode(false)
    setSearchQuery('')
    setCurrentPath([])
  }

  // Переход в подуровень
  const navigateToChild = (item: KpvedItem) => {
    setCurrentPath([...currentPath, item])
    loadLevel(item.code)
  }

  // Назад по иерархии
  const navigateBack = () => {
    const newPath = currentPath.slice(0, -1)
    setCurrentPath(newPath)
    const parent = newPath.length > 0 ? newPath[newPath.length - 1].code : null
    loadLevel(parent)
  }

  // Очистить выбор
  const handleClear = () => {
    setSelectedItem(null)
    onChange(null)
    setCurrentPath([])
    setSearchMode(false)
    setSearchQuery('')
  }

  // Повторить последнюю операцию
  const handleRetry = () => {
    if (!lastOperation) return

    if (lastOperation.type === 'load') {
      loadLevel(lastOperation.param)
    } else if (lastOperation.type === 'search' && lastOperation.param) {
      searchKpved(lastOperation.param)
    }
  }

  return (
    <div className="flex items-center gap-2">
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            role="combobox"
            aria-expanded={open}
            className="w-full justify-between"
          >
            {selectedItem ? (
              <span className="flex items-center gap-2">
                <Badge variant="secondary" className="font-mono">
                  {selectedItem.code}
                </Badge>
                <span className="truncate">{selectedItem.name}</span>
              </span>
            ) : (
              placeholder
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-[500px] p-0" align="start">
          <div className="flex items-center border-b p-2 gap-2">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setSearchMode(!searchMode)
                setSearchQuery('')
              }}
            >
              <Search className="h-4 w-4" />
            </Button>
            {!searchMode && currentPath.length > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={navigateBack}
              >
                ← Назад
              </Button>
            )}
          </div>

          {error && (
            <Alert variant="destructive" className="m-2">
              <AlertDescription className="flex items-center justify-between">
                <span className="text-sm">{error}</span>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleRetry}
                  disabled={loading}
                >
                  Повторить
                </Button>
              </AlertDescription>
            </Alert>
          )}

          {searchMode ? (
            <Command>
              <CommandInput
                placeholder="Поиск по коду или названию..."
                value={searchQuery}
                onValueChange={setSearchQuery}
              />
              <CommandList>
                <CommandEmpty>
                  {loading ? 'Поиск...' : 'Ничего не найдено'}
                </CommandEmpty>
                <CommandGroup>
                  {searchResults.map((item) => (
                    <CommandItem
                      key={item.code}
                      onSelect={() => handleSelect(item)}
                    >
                      <Check
                        className={cn(
                          'mr-2 h-4 w-4',
                          value === item.code ? 'opacity-100' : 'opacity-0'
                        )}
                      />
                      <Badge variant="outline" className="mr-2 font-mono">
                        {item.code}
                      </Badge>
                      <span className="text-sm">{item.name}</span>
                    </CommandItem>
                  ))}
                </CommandGroup>
              </CommandList>
            </Command>
          ) : (
            <div className="max-h-[300px] overflow-y-auto p-2">
              {currentPath.length > 0 && (
                <div className="mb-2 flex items-center gap-1 text-xs text-muted-foreground">
                  {currentPath.map((item, idx) => (
                    <span key={item.code} className="flex items-center">
                      {idx > 0 && <ChevronRight className="h-3 w-3 mx-1" />}
                      <Badge variant="outline" className="font-mono text-xs">
                        {item.code}
                      </Badge>
                    </span>
                  ))}
                </div>
              )}
              {loading ? (
                <div className="py-4 text-center text-sm text-muted-foreground">
                  Загрузка...
                </div>
              ) : (
                <div className="space-y-1">
                  {hierarchyItems.map((item) => (
                    <div
                      key={item.code}
                      className="flex items-center justify-between rounded-md p-2 hover:bg-accent cursor-pointer"
                      onClick={() => {
                        // Если это не последний уровень, переходим в него
                        if (item.level < 4) {
                          navigateToChild(item)
                        } else {
                          // Если это последний уровень, выбираем
                          handleSelect(item)
                        }
                      }}
                    >
                      <div className="flex items-center gap-2 flex-1">
                        <Badge variant="outline" className="font-mono">
                          {item.code}
                        </Badge>
                        <span className="text-sm">{item.name}</span>
                      </div>
                      {item.level < 4 && (
                        <ChevronRight className="h-4 w-4 text-muted-foreground" />
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          )}
        </PopoverContent>
      </Popover>
      {selectedItem && (
        <Button
          variant="ghost"
          size="sm"
          onClick={handleClear}
        >
          <X className="h-4 w-4" />
        </Button>
      )}
    </div>
  )
}
