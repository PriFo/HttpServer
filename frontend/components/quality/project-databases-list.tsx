'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Database, ChevronRight, BarChart3, ArrowUp, ArrowDown, Search, X, Clock } from 'lucide-react'
import { DatabaseStat } from '@/app/quality/page'
import { cn } from '@/lib/utils'
import { normalizePercentage } from '@/lib/locale'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'

interface ProjectDatabasesListProps {
  databases: DatabaseStat[]
  selectedDatabase?: string
  onDatabaseSelect: (databasePath: string) => void
  loading?: boolean
}

type SortField = 'name' | 'quality' | 'items' | 'benchmarks' | 'activity'
type SortOrder = 'asc' | 'desc'
type LevelSummary = {
  count?: number
  avg_quality?: number
  percentage?: number
}

const getActivityTimestamp = (db: DatabaseStat) => {
  const source = db.last_activity || db.last_upload_at || db.last_used_at
  if (!source) return 0
  const time = new Date(source).getTime()
  return Number.isNaN(time) ? 0 : time
}

const formatRelativeTime = (value?: string) => {
  if (!value) return '—'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '—'
  return formatDistanceToNow(date, { addSuffix: true, locale: ru })
}

const formatAbsoluteTime = (value?: string) => {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleString('ru-RU')
}

export function ProjectDatabasesList({
  databases,
  selectedDatabase,
  onDatabaseSelect,
  loading = false,
}: ProjectDatabasesListProps) {
  const [sortField, setSortField] = useState<SortField>('name')
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc')
  const [searchQuery, setSearchQuery] = useState('')
  const [qualityFilter, setQualityFilter] = useState<'all' | 'high' | 'medium' | 'low'>('all')

  // Фильтрация и сортировка баз данных
  const filteredAndSortedDatabases = useMemo(() => {
    let filtered = [...databases]

    // Фильтрация по поисковому запросу
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase().trim()
      filtered = filtered.filter(db => 
        db.database_name.toLowerCase().includes(query) ||
        db.database_path.toLowerCase().includes(query)
      )
    }

    // Фильтрация по качеству
    if (qualityFilter !== 'all') {
      filtered = filtered.filter(db => {
        const quality = normalizePercentage(db.stats.average_quality || 0)
        switch (qualityFilter) {
          case 'high':
            return quality >= 90
          case 'medium':
            return quality >= 70 && quality < 90
          case 'low':
            return quality < 70
          default:
            return true
        }
      })
    }

    // Сортировка
    filtered.sort((a, b) => {
      let comparison = 0
      
      switch (sortField) {
        case 'name':
          comparison = a.database_name.localeCompare(b.database_name)
          break
        case 'quality':
          comparison = (a.stats.average_quality || 0) - (b.stats.average_quality || 0)
          break
        case 'items':
          comparison = (a.stats.total_items || 0) - (b.stats.total_items || 0)
          break
        case 'benchmarks':
          comparison = (a.stats.benchmark_count || 0) - (b.stats.benchmark_count || 0)
          break
        case 'activity':
          comparison = getActivityTimestamp(a) - getActivityTimestamp(b)
          break
      }
      
      return sortOrder === 'asc' ? comparison : -comparison
    })
    return filtered
  }, [databases, sortField, sortOrder, searchQuery, qualityFilter])

  const handleSortChange = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortOrder('asc')
    }
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5 animate-pulse" />
            Базы данных проекта
          </CardTitle>
          <CardDescription>
            <span className="inline-flex items-center gap-2">
              <span className="animate-pulse">Загрузка статистики...</span>
            </span>
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="h-24 bg-muted/50 animate-pulse rounded-lg border border-muted" />
            ))}
          </div>
          <div className="mt-4 text-center text-sm text-muted-foreground">
            Обработка баз данных проекта...
          </div>
        </CardContent>
      </Card>
    )
  }

  if (databases.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            Базы данных проекта
          </CardTitle>
          <CardDescription>В проекте нет активных баз данных</CardDescription>
        </CardHeader>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="space-y-4">
          <div className="flex items-center justify-between gap-4">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Database className="h-5 w-5" />
                Базы данных проекта
              </CardTitle>
              <CardDescription>
                {filteredAndSortedDatabases.length} из {databases.length} {databases.length === 1 ? 'база данных' : 'баз данных'}
                {(searchQuery || qualityFilter !== 'all') && ` (отфильтровано)`}
              </CardDescription>
            </div>
            <Select
              value={sortField}
              onValueChange={(value) => handleSortChange(value as SortField)}
            >
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Сортировать по..." />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="name">
                  <div className="flex items-center gap-2">
                    <span>По имени</span>
                    {sortField === 'name' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="quality">
                  <div className="flex items-center gap-2">
                    <span>По качеству</span>
                    {sortField === 'quality' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="items">
                  <div className="flex items-center gap-2">
                    <span>По элементам</span>
                    {sortField === 'items' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="benchmarks">
                  <div className="flex items-center gap-2">
                    <span>По эталонам</span>
                    {sortField === 'benchmarks' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
                <SelectItem value="activity">
                  <div className="flex items-center gap-2">
                    <span>По активности</span>
                    {sortField === 'activity' && (sortOrder === 'asc' ? <ArrowUp className="h-3 w-3" /> : <ArrowDown className="h-3 w-3" />)}
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="flex items-center gap-3">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Поиск по имени или пути..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 pr-9"
              />
              {searchQuery && (
                <Button
                  variant="ghost"
                  size="icon"
                  className="absolute right-1 top-1/2 transform -translate-y-1/2 h-6 w-6"
                  onClick={() => setSearchQuery('')}
                >
                  <X className="h-4 w-4" />
                </Button>
              )}
            </div>
            <Select value={qualityFilter} onValueChange={(value) => setQualityFilter(value as typeof qualityFilter)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все</SelectItem>
                <SelectItem value="high">Высокое (≥90%)</SelectItem>
                <SelectItem value="medium">Среднее (70-90%)</SelectItem>
                <SelectItem value="low">Низкое (&lt;70%)</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {filteredAndSortedDatabases.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            <Database className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>Базы данных не найдены</p>
            {(searchQuery || qualityFilter !== 'all') && (
              <Button
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={() => {
                  setSearchQuery('')
                  setQualityFilter('all')
                }}
              >
                Очистить фильтры
              </Button>
            )}
          </div>
        ) : (
          <div className="space-y-3">
            {filteredAndSortedDatabases.map((db) => {
              const stats = db.stats
              const isSelected = selectedDatabase === db.database_path
              const qualityPercentage = normalizePercentage(stats.average_quality || 0)
              const activitySource = db.last_activity || db.last_upload_at || db.last_used_at
              const activityLabel = activitySource ? formatRelativeTime(activitySource) : null
              const absoluteActivity = activitySource ? formatAbsoluteTime(activitySource) : ''
              const activityTag = activitySource
                ? db.last_upload_at && db.last_upload_at === activitySource
                  ? 'Сканирование'
                  : db.last_used_at && db.last_used_at === activitySource
                    ? 'Использование'
                    : 'Активность'
                : null

              return (
                <Card
                  key={db.database_id}
                  className={cn("cursor-pointer transition-all hover:shadow-md", isSelected && "ring-2 ring-primary")}
                  onClick={() => onDatabaseSelect(db.database_path)}
                >
                  <CardContent className="pt-4">
                    <div className="flex items-start justify-between gap-4">
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-2">
                          <Database className="h-4 w-4 text-muted-foreground shrink-0" />
                          <h4 className="font-semibold truncate">{db.database_name}</h4>
                          {isSelected && (
                            <Badge variant="default" className="shrink-0">
                              Выбрана
                            </Badge>
                          )}
                        </div>
                        <p className="text-sm text-muted-foreground truncate mb-3">
                          {db.database_path}
                        </p>
                        <div className="flex items-center gap-4 flex-wrap">
                          <div className="flex items-center gap-2">
                            <BarChart3 className="h-4 w-4 text-muted-foreground" />
                            <span className="text-sm">
                              <span className="font-medium">{stats.total_items || 0}</span>
                              <span className="text-muted-foreground ml-1">элементов</span>
                            </span>
                          </div>
                          <Badge
                            variant={qualityPercentage >= 90 ? "default" : qualityPercentage >= 70 ? "outline" : "destructive"}
                            className="text-xs"
                          >
                            {qualityPercentage.toFixed(1)}% качество
                          </Badge>
                          {stats.benchmark_count > 0 && (
                            <Badge variant="outline" className="text-xs">
                              {stats.benchmark_count} эталонов
                            </Badge>
                          )}
                          {stats.by_level && Object.keys(stats.by_level).length > 0 && (
                            <div className="flex items-center gap-2 text-xs text-muted-foreground">
                              {Object.entries(stats.by_level).slice(0, 3).map(([level, levelData]: [string, LevelSummary]) => (
                                <span key={level} className="flex items-center gap-1">
                                  <span className="font-medium">{levelData.count || 0}</span>
                                  <span className="text-muted-foreground">
                                    {level === 'basic' ? 'баз.' : level === 'ai_enhanced' ? 'AI' : level === 'benchmark' ? 'этал.' : level}
                                  </span>
                                </span>
                              ))}
                            </div>
                          )}
                        </div>
                        {activityLabel && (
                          <div className="flex items-center justify-between text-xs text-muted-foreground mt-3">
                            <div className="flex items-center gap-1">
                              <Clock className="h-3.5 w-3.5" />
                              <span>Последняя активность</span>
                              {activityTag && (
                                <Badge variant="outline" className="text-[10px] ml-2">
                                  {activityTag}
                                </Badge>
                              )}
                            </div>
                            <span className="font-medium" title={absoluteActivity}>
                              {activityLabel}
                            </span>
                          </div>
                        )}
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="shrink-0"
                        onClick={(e) => {
                          e.stopPropagation()
                          onDatabaseSelect(db.database_path)
                        }}
                      >
                        <ChevronRight className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

