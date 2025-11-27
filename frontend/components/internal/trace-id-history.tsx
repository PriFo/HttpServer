'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Clock, Search, Trash2, History } from 'lucide-react'
import { motion, AnimatePresence } from 'framer-motion'

interface TraceIDHistoryProps {
  onSelectTraceId: (traceId: string) => void
}

const STORAGE_KEY = 'worker-trace-history'

export function TraceIDHistory({ onSelectTraceId }: TraceIDHistoryProps) {
  const [history, setHistory] = useState<string[]>([])
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    // Загружаем историю из localStorage
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        const parsed = JSON.parse(stored) as string[]
        setHistory(parsed)
      }
    } catch (error) {
      console.error('Error loading trace history:', error)
    }
  }, [])

  const addToHistory = (traceId: string) => {
    if (!traceId.trim()) return
    
    setHistory((prev) => {
      const updated = [traceId, ...prev.filter((id) => id !== traceId)].slice(0, 20) // Максимум 20 записей
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
      } catch (error) {
        console.error('Error saving trace history:', error)
      }
      return updated
    })
  }

  const removeFromHistory = (traceId: string, e: React.MouseEvent) => {
    e.stopPropagation()
    setHistory((prev) => {
      const updated = prev.filter((id) => id !== traceId)
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
      } catch (error) {
        console.error('Error saving trace history:', error)
      }
      return updated
    })
  }

  const clearHistory = () => {
    setHistory([])
    try {
      localStorage.removeItem(STORAGE_KEY)
    } catch (error) {
      console.error('Error clearing trace history:', error)
    }
  }

  const filteredHistory = history.filter((id) =>
    id.toLowerCase().includes(searchQuery.toLowerCase())
  )

  // Экспортируем функцию для добавления в историю
  useEffect(() => {
    // Сохраняем функцию в window для доступа из родительского компонента
    ;(window as any).__addTraceToHistory = addToHistory
    return () => {
      delete (window as any).__addTraceToHistory
    }
  }, [])

  if (history.length === 0) {
    return null
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <History className="h-5 w-5" />
          История trace_id
        </CardTitle>
        <CardDescription>
          Последние использованные trace_id для быстрого доступа
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {/* Поиск */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Поиск в истории..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-9"
            />
          </div>

          {/* Список истории */}
          <div className="space-y-2 max-h-[300px] overflow-y-auto">
            <AnimatePresence mode="popLayout">
              {filteredHistory.map((traceId, index) => (
                <motion.div
                  key={traceId}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 20 }}
                  transition={{ delay: index * 0.05 }}
                  className="flex items-center justify-between p-3 border rounded-lg hover:bg-accent/50 transition-colors cursor-pointer group"
                  onClick={() => onSelectTraceId(traceId)}
                >
                  <div className="flex-1 min-w-0">
                    <div className="font-mono text-sm truncate">{traceId}</div>
                    <div className="text-xs text-muted-foreground mt-1">
                      Нажмите для использования
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="opacity-0 group-hover:opacity-100 transition-opacity"
                    onClick={(e) => removeFromHistory(traceId, e)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </motion.div>
              ))}
            </AnimatePresence>
          </div>

          {/* Очистить историю */}
          {history.length > 0 && (
            <Button
              variant="outline"
              size="sm"
              className="w-full"
              onClick={clearHistory}
            >
              Очистить историю
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  )
}

// Хук для добавления trace_id в историю
export function useTraceHistory() {
  const addToHistory = (traceId: string) => {
    if (!traceId.trim()) return
    
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      const history = stored ? (JSON.parse(stored) as string[]) : []
      const updated = [traceId, ...history.filter((id) => id !== traceId)].slice(0, 20)
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated))
    } catch (error) {
      console.error('Error saving trace history:', error)
    }
  }

  return { addToHistory }
}

