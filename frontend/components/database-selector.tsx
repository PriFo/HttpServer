'use client'

import { useState, useEffect, useRef, useMemo } from 'react'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Database, RefreshCw, CheckCircle2 } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface DatabaseInfo {
  name: string
  path?: string
  isCurrent: boolean
}

interface DatabaseSelectorProps {
  value?: string
  onChange: (database: string) => void
  placeholder?: string
  className?: string
  cacheKey?: string
  cacheTTL?: number // Time to live in milliseconds
  onRefresh?: () => void
  projectFilter?: string // Формат: "clientId:projectId"
}

// Глобальный кеш для всех экземпляров компонента
const databaseCache = new Map<string, { data: DatabaseInfo[]; timestamp: number }>()

const DEFAULT_CACHE_KEY = 'databases-list'
const DEFAULT_CACHE_TTL = 5 * 60 * 1000 // 5 минут

export function DatabaseSelector({
  value,
  onChange,
  placeholder = "Выберите базу данных",
  className,
  cacheKey = DEFAULT_CACHE_KEY,
  cacheTTL = DEFAULT_CACHE_TTL,
  onRefresh,
  projectFilter,
}: DatabaseSelectorProps) {
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const isInitialMount = useRef(true)

  useEffect(() => {
    // Загружаем список баз данных при монтировании компонента или изменении фильтра проекта
    fetchDatabases(true) // Принудительно обновляем при изменении фильтра
    
    /**
     * Обработчик события фокуса окна
     * Автоматически обновляет список баз данных при возврате на страницу,
     * если кеш устарел (старше cacheTTL)
     */
    const handleFocus = () => {
      const cached = databaseCache.get(cacheKey)
      if (!cached) {
        // Если кеша нет, загружаем данные
        fetchDatabases(false)
      } else {
        const now = Date.now()
        // Если кеш устарел, обновляем данные
        if (now - cached.timestamp > cacheTTL) {
          fetchDatabases(false)
        }
      }
    }
    
    window.addEventListener('focus', handleFocus)
    
    return () => {
      window.removeEventListener('focus', handleFocus)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectFilter])

  const getCachedData = (): DatabaseInfo[] | null => {
    const cached = databaseCache.get(cacheKey)
    if (!cached) return null

    const now = Date.now()
    if (now - cached.timestamp > cacheTTL) {
      databaseCache.delete(cacheKey)
      return null
    }

    return cached.data
  }

  const setCachedData = (data: DatabaseInfo[]) => {
    databaseCache.set(cacheKey, {
      data,
      timestamp: Date.now(),
    })
  }

  /**
   * Загружает список баз данных с сервера
   * @param forceRefresh - если true, игнорирует кеш и принудительно обновляет данные
   */
  const fetchDatabases = async (forceRefresh = false) => {
    try {
      // Очищаем кеш при принудительном обновлении
      if (forceRefresh) {
        databaseCache.delete(cacheKey)
      } else {
        // Проверяем кеш, если не принудительное обновление
        const cached = getCachedData()
        if (cached) {
          setDatabases(cached)
          setLoading(false)
          
          // Автоматически выбираем текущую базу данных, если значение не задано
          if (!value && cached.length > 0) {
            const current = cached.find((db) => db.isCurrent)
            if (current && isInitialMount.current) {
              onChange(current.name)
              isInitialMount.current = false
            }
          }
          return
        }
      }

      setLoading(true)
      setError(null)

      // Если указан фильтр проекта, загружаем базы данных проекта
      let apiUrl = '/api/databases/list'
      if (projectFilter) {
        const parts = projectFilter.split(':')
        if (parts.length === 2) {
          const clientId = parts[0]
          const projectId = parts[1]
          apiUrl = `/api/clients/${clientId}/projects/${projectId}/databases?active_only=true`
        }
      }

      // Используем cache: 'no-store' для предотвращения кеширования на уровне браузера
      // Это гарантирует, что мы всегда получаем актуальные данные с сервера
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут
      
      const response = await fetch(apiUrl, {
        cache: 'no-store',
        headers: {
          'Cache-Control': 'no-cache',
        },
        signal: controller.signal,
      })
      
      clearTimeout(timeoutId)
      if (!response.ok) {
        let errorMessage = 'Не удалось загрузить список баз данных'
        
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          // Если ответ не JSON, пытаемся получить текст
          const errorText = await response.text().catch(() => 'Unknown error')
          console.error('Failed to fetch databases:', errorText)
          
          // Более информативные сообщения об ошибках
          if (response.status === 404) {
            errorMessage = 'Сервер не найден. Проверьте подключение к backend'
          } else if (response.status === 500) {
            errorMessage = 'Ошибка сервера. Попробуйте позже'
          } else if (response.status >= 400 && response.status < 500) {
            errorMessage = `Ошибка запроса (${response.status})`
          }
        }
        
        throw new Error(errorMessage)
      }

      const data = await response.json()
      let databasesList: DatabaseInfo[] = []
      
      if (projectFilter) {
        // Если фильтр по проекту, преобразуем формат ответа
        const projectDatabases = data.databases || []
        databasesList = projectDatabases.map((db: any) => ({
          name: db.file_path || db.name || db.path || '',
          path: db.file_path || db.path,
          isCurrent: false, // Для баз проекта не определяем текущую
        }))
      } else {
        databasesList = data.databases || []
      }
      
      setDatabases(databasesList)
      setCachedData(databasesList)

      // Auto-select current database if no value provided
      if (!value && databasesList.length > 0) {
        const current = databasesList.find((db: DatabaseInfo) => db.isCurrent)
        if (current && isInitialMount.current) {
          onChange(current.name)
          isInitialMount.current = false
        }
      }

      onRefresh?.()
    } catch (err) {
      console.error('Error fetching databases:', err)
      
      // Обрабатываем разные типы ошибок
      let errorMessage = 'Не удалось загрузить список баз данных'
      if (err instanceof Error) {
        if (err.name === 'AbortError') {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
          errorMessage = 'Ошибка сети. Проверьте подключение к backend серверу на порту 9999'
        } else {
          errorMessage = err.message || errorMessage
        }
      }
      
      setError(errorMessage)
      
      // Если есть кэш, показываем его данные
      const cached = getCachedData()
      if (cached && cached.length > 0) {
        setDatabases(cached)
      }
    } finally {
      setLoading(false)
    }
  }

  // Метод для очистки кеша (можно вызвать извне)
  const clearCache = () => {
    databaseCache.delete(cacheKey)
  }

  // Экспортируем метод для использования в других компонентах
  useEffect(() => {
    // Добавляем метод в window для глобального доступа (опционально)
    if (typeof window !== 'undefined') {
      ;(window as any).clearDatabaseCache = clearCache
    }
  }, [])

  if (error) {
    return (
      <div className={cn("flex items-center gap-2", className)}>
        <Select
          value={value}
          onValueChange={onChange}
          disabled={true}
        >
          <SelectTrigger className="w-full min-w-[200px]">
            <div className="flex items-center gap-2">
              <Database className="h-4 w-4" />
              <SelectValue placeholder={placeholder} />
            </div>
          </SelectTrigger>
        </Select>
        <Button
          variant="outline"
          size="icon"
          onClick={() => {
            setError(null)
            fetchDatabases(true)
          }}
          disabled={loading}
          className="h-10 w-10"
          title="Повторить попытку"
        >
          <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
        </Button>
        <Badge variant="destructive" className="text-xs whitespace-nowrap">
          {error}
        </Badge>
      </div>
    )
  }

  // Оптимизация: мемоизируем обработку списка баз данных
  const processedDatabases = useMemo(() => {
    if (databases.length === 0) {
      return []
    }

    // Убираем дубликаты по имени, предпочитая текущую базу данных
    return databases.reduce((acc, db) => {
      const existing = acc.find((d) => d.name === db.name)
      if (!existing) {
        acc.push(db)
      } else if (db.isCurrent && !existing.isCurrent) {
        // Заменяем существующую на текущую, если она есть
        const index = acc.indexOf(existing)
        acc[index] = db
      }
      return acc
    }, [] as DatabaseInfo[])
  }, [databases])

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Select
        value={value}
        onValueChange={onChange}
        disabled={loading}
      >
        <SelectTrigger className="w-full min-w-[200px]">
          <div className="flex items-center gap-2">
            <Database className={cn("h-4 w-4", loading && "animate-pulse")} />
            <SelectValue placeholder={loading ? "Загрузка..." : placeholder} />
          </div>
        </SelectTrigger>
        <SelectContent>
          {processedDatabases.length === 0 ? (
            <SelectItem value="no-database" disabled>
              <div className="flex items-center gap-2 text-muted-foreground">
                <Database className="h-4 w-4" />
                <span>Нет доступных баз данных</span>
              </div>
            </SelectItem>
          ) : (
            processedDatabases.map((db, index) => {
              // Создаем уникальный ключ, всегда включая индекс для гарантии уникальности
              // Это предотвращает ошибки React о дублирующихся ключах
              const uniqueKey = db.path ? `${db.path}-${index}` : `${db.name}-${index}`
              return (
                <SelectItem key={uniqueKey} value={db.name}>
                  <div className="flex items-center justify-between gap-3 w-full">
                    <div className="flex items-center gap-2 flex-1 min-w-0">
                      <Database className="h-4 w-4 text-muted-foreground shrink-0" />
                      <span className="truncate">{db.name}</span>
                    </div>
                    {db.isCurrent && (
                      <Badge variant="default" className="text-xs flex items-center gap-1 shrink-0">
                        <CheckCircle2 className="h-3 w-3" />
                        Текущая
                      </Badge>
                    )}
                  </div>
                </SelectItem>
              )
            })
          )}
        </SelectContent>
      </Select>
      <Button
        variant="outline"
        size="icon"
        onClick={() => fetchDatabases(true)}
        disabled={loading}
        className="h-10 w-10"
        title="Обновить список баз данных"
      >
        <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
      </Button>
    </div>
  )
}
