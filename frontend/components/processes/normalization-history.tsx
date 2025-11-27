'use client'

import { useState, useEffect } from 'react'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { History, Calendar, Clock, CheckCircle2, XCircle, AlertCircle, Loader2, Eye, Search, Filter, Download, RefreshCw } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale/ru'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { SessionDetailDialog } from './session-detail-dialog'

interface NormalizationSession {
  id: number
  project_database_id?: number
  database_name?: string
  status: string
  created_at: string
  finished_at?: string | null
  processed_count?: number
  success_count?: number
  error_count?: number
}

interface NormalizationHistoryProps {
  type: 'nomenclature' | 'counterparties'
  clientId?: number | null
  projectId?: number | null
  limit?: number
}

export function NormalizationHistory({ type, clientId, projectId, limit = 10 }: NormalizationHistoryProps) {
  const [sessions, setSessions] = useState<NormalizationSession[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedSessionId, setSelectedSessionId] = useState<number | null>(null)
  const [isDialogOpen, setIsDialogOpen] = useState(false)
  
  // Фильтры и поиск
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'date' | 'status' | 'processed'>('date')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')

  useEffect(() => {
    const fetchHistory = async () => {
      setLoading(true)
      setError(null)

      try {
        // Определяем endpoint в зависимости от типа и наличия clientId/projectId
        let endpoint = `/api/normalization/stats?type=${type}`
        if (clientId && projectId) {
          if (type === 'nomenclature') {
            endpoint = `/api/clients/${clientId}/projects/${projectId}/normalization/stats`
          } else {
            endpoint = `/api/counterparties/normalized/stats?client_id=${clientId}&project_id=${projectId}`
          }
        }
        
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут
        
        const response = await fetch(endpoint, {
          cache: 'no-store',
          signal: controller.signal,
        })
        
        clearTimeout(timeoutId)

        if (!response.ok) {
          throw new Error('Не удалось загрузить историю')
        }

        const data = await response.json()
        
        // Если в ответе есть sessions, используем их
        if (data.sessions && Array.isArray(data.sessions)) {
          setSessions(data.sessions.slice(0, limit))
        } else {
          // Иначе создаем пустой массив
          setSessions([])
        }
      } catch (err) {
        if (err instanceof Error) {
          // Улучшаем сообщения об ошибках
          let errorMessage = err.message
          if (err.name === 'AbortError') {
            errorMessage = 'Превышено время ожидания ответа от сервера'
          } else if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
            errorMessage = 'Не удалось подключиться к серверу. Проверьте подключение к backend серверу на порту 9999'
          }
          setError(errorMessage)
        } else {
          setError('Неизвестная ошибка при загрузке истории')
        }
      } finally {
        setLoading(false)
      }
    }

    fetchHistory()
  }, [type, clientId, projectId, limit])

  const getStatusBadge = (status: string) => {
    switch (status.toLowerCase()) {
      case 'completed':
      case 'finished':
        return (
          <Badge variant="default" className="bg-green-500">
            <CheckCircle2 className="h-3 w-3 mr-1" />
            Завершено
          </Badge>
        )
      case 'failed':
      case 'error':
        return (
          <Badge variant="destructive">
            <XCircle className="h-3 w-3 mr-1" />
            Ошибка
          </Badge>
        )
      case 'running':
      case 'in_progress':
        return (
          <Badge variant="default" className="bg-blue-500">
            <Loader2 className="h-3 w-3 mr-1 animate-spin" />
            Выполняется
          </Badge>
        )
      case 'stopped':
        return (
          <Badge variant="secondary">
            <AlertCircle className="h-3 w-3 mr-1" />
            Остановлено
          </Badge>
        )
      default:
        return (
          <Badge variant="outline">
            {status}
          </Badge>
        )
    }
  }

  const formatDate = (dateString: string) => {
    try {
      const date = new Date(dateString)
      return formatDistanceToNow(date, { addSuffix: true, locale: ru })
    } catch {
      return dateString
    }
  }

  const getDuration = (start: string, end?: string | null) => {
    if (!end) return '-'
    try {
      const startDate = new Date(start)
      const endDate = new Date(end)
      const diff = endDate.getTime() - startDate.getTime()
      const minutes = Math.floor(diff / 60000)
      const seconds = Math.floor((diff % 60000) / 1000)
      
      if (minutes > 0) {
        return `${minutes}м ${seconds}с`
      }
      return `${seconds}с`
    } catch {
      return '-'
    }
  }

  // Фильтрация и сортировка
  const filteredAndSortedSessions = sessions
    .filter((session) => {
      // Поиск по ID, базе данных
      const matchesSearch =
        searchQuery === '' ||
        session.id.toString().includes(searchQuery) ||
        (session.database_name || '').toLowerCase().includes(searchQuery.toLowerCase()) ||
        (session.project_database_id?.toString() || '').includes(searchQuery)

      // Фильтр по статусу
      const matchesStatus =
        statusFilter === 'all' ||
        session.status.toLowerCase() === statusFilter.toLowerCase()

      return matchesSearch && matchesStatus
    })
    .sort((a, b) => {
      let comparison = 0

      switch (sortBy) {
        case 'date':
          comparison =
            new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
          break
        case 'status':
          comparison = a.status.localeCompare(b.status)
          break
        case 'processed':
          comparison =
            (a.processed_count || 0) - (b.processed_count || 0)
          break
      }

      return sortOrder === 'asc' ? comparison : -comparison
    })

  // Экспорт данных
  const handleExport = () => {
    const csvContent = [
      ['ID', 'База данных', 'Статус', 'Создано', 'Завершено', 'Длительность', 'Обработано', 'Успешно', 'Ошибок'].join(','),
      ...filteredAndSortedSessions.map((session) =>
        [
          session.id,
          session.database_name || `БД #${session.project_database_id || 'N/A'}`,
          session.status,
          session.created_at,
          session.finished_at || '',
          getDuration(session.created_at, session.finished_at),
          session.processed_count || 0,
          session.success_count || 0,
          session.error_count || 0,
        ].join(',')
      ),
    ].join('\n')

    const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
    const link = document.createElement('a')
    const url = URL.createObjectURL(blob)
    link.setAttribute('href', url)
    link.setAttribute('download', `normalization-history-${type}-${new Date().toISOString().split('T')[0]}.csv`)
    link.style.visibility = 'hidden'
    document.body.appendChild(link)
    link.click()
    document.body.removeChild(link)
  }

  if (loading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            История процессов
          </CardTitle>
          <CardDescription>
            Последние {limit} сессий нормализации {type === 'nomenclature' ? 'номенклатуры' : 'контрагентов'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {[...Array(3)].map((_, i) => (
              <Skeleton key={i} className="h-16 w-full" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  const retryFetch = async () => {
    setError(null)
    setLoading(true)
    try {
      let endpoint = `/api/normalization/stats?type=${type}`
      if (clientId && projectId) {
        if (type === 'nomenclature') {
          endpoint = `/api/clients/${clientId}/projects/${projectId}/normalization/stats`
        } else {
          endpoint = `/api/counterparties/normalized/stats?client_id=${clientId}&project_id=${projectId}`
        }
      }
      const response = await fetch(endpoint, { cache: 'no-store' })
      if (!response.ok) throw new Error('Не удалось загрузить историю')
      const data = await response.json()
      if (data.sessions && Array.isArray(data.sessions)) {
        setSessions(data.sessions.slice(0, limit))
      } else {
        setSessions([])
      }
    } catch (err) {
      if (err instanceof Error) {
        let errorMessage = err.message
        if (err.name === 'AbortError') {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (err.message.includes('Failed to fetch') || err.message.includes('NetworkError')) {
          errorMessage = 'Не удалось подключиться к серверу. Проверьте подключение к backend серверу на порту 9999'
        }
        setError(errorMessage)
      } else {
        setError('Неизвестная ошибка при загрузке истории')
      }
    } finally {
      setLoading(false)
    }
  }

  if (error) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            История процессов
          </CardTitle>
          <CardDescription>
            Последние {limit} сессий нормализации {type === 'nomenclature' ? 'номенклатуры' : 'контрагентов'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            <AlertCircle className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p className="mb-4">{error}</p>
            <Button
              variant="outline"
              size="sm"
              onClick={retryFetch}
              disabled={loading}
              className="flex items-center gap-2"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              Повторить
            </Button>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (sessions.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            История процессов
          </CardTitle>
          <CardDescription>
            Последние {limit} сессий нормализации {type === 'nomenclature' ? 'номенклатуры' : 'контрагентов'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-muted-foreground">
            <History className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>История процессов пока пуста</p>
            <p className="text-sm mt-2">Запустите процесс нормализации, чтобы увидеть историю</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <History className="h-5 w-5" />
              История процессов
            </CardTitle>
            <CardDescription>
              Последние {limit} сессий нормализации {type === 'nomenclature' ? 'номенклатуры' : 'контрагентов'}
            </CardDescription>
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={handleExport}
            className="flex items-center gap-2"
            disabled={filteredAndSortedSessions.length === 0}
          >
            <Download className="h-4 w-4" />
            Экспорт
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {/* Фильтры и поиск */}
        <div className="mb-4 space-y-3">
          <div className="flex flex-col sm:flex-row gap-3">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Поиск по ID или базе данных..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-full sm:w-[180px]">
                <Filter className="h-4 w-4 mr-2" />
                <SelectValue placeholder="Статус" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все статусы</SelectItem>
                <SelectItem value="completed">Завершено</SelectItem>
                <SelectItem value="running">Выполняется</SelectItem>
                <SelectItem value="failed">Ошибка</SelectItem>
                <SelectItem value="stopped">Остановлено</SelectItem>
              </SelectContent>
            </Select>
            <Select value={`${sortBy}-${sortOrder}`} onValueChange={(value) => {
              const [by, order] = value.split('-')
              setSortBy(by as 'date' | 'status' | 'processed')
              setSortOrder(order as 'asc' | 'desc')
            }}>
              <SelectTrigger className="w-full sm:w-[200px]">
                <SelectValue placeholder="Сортировка" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="date-desc">Дата (новые)</SelectItem>
                <SelectItem value="date-asc">Дата (старые)</SelectItem>
                <SelectItem value="status-asc">Статус (А-Я)</SelectItem>
                <SelectItem value="status-desc">Статус (Я-А)</SelectItem>
                <SelectItem value="processed-desc">Обработано (больше)</SelectItem>
                <SelectItem value="processed-asc">Обработано (меньше)</SelectItem>
              </SelectContent>
            </Select>
          </div>
          {filteredAndSortedSessions.length !== sessions.length && (
            <div className="text-sm text-muted-foreground">
              Показано {filteredAndSortedSessions.length} из {sessions.length} сессий
            </div>
          )}
        </div>
        <div className="overflow-x-auto">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ID</TableHead>
                <TableHead>База данных</TableHead>
                <TableHead>Статус</TableHead>
                <TableHead>Создано</TableHead>
                <TableHead>Длительность</TableHead>
                <TableHead>Обработано</TableHead>
                <TableHead>Действия</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredAndSortedSessions.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                    {sessions.length === 0
                      ? 'История процессов пока пуста'
                      : 'Не найдено сессий по заданным фильтрам'}
                  </TableCell>
                </TableRow>
              ) : (
                filteredAndSortedSessions.map((session) => (
                <TableRow key={session.id}>
                  <TableCell className="font-mono text-sm">#{session.id}</TableCell>
                  <TableCell>
                    {session.database_name || `БД #${session.project_database_id || 'N/A'}`}
                  </TableCell>
                  <TableCell>{getStatusBadge(session.status)}</TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    <div className="flex items-center gap-1">
                      <Calendar className="h-3 w-3" />
                      {formatDate(session.created_at)}
                    </div>
                  </TableCell>
                  <TableCell className="text-sm text-muted-foreground">
                    <div className="flex items-center gap-1">
                      <Clock className="h-3 w-3" />
                      {getDuration(session.created_at, session.finished_at)}
                    </div>
                  </TableCell>
                  <TableCell>
                    {session.processed_count !== undefined ? (
                      <div className="flex flex-col gap-1">
                        <span className="text-sm font-medium">{session.processed_count}</span>
                        {session.success_count !== undefined && session.error_count !== undefined && (
                          <div className="flex gap-2 text-xs text-muted-foreground">
                            <span className="text-green-600">✓ {session.success_count}</span>
                            {session.error_count > 0 && (
                              <span className="text-red-600">✗ {session.error_count}</span>
                            )}
                          </div>
                        )}
                      </div>
                    ) : (
                      <span className="text-muted-foreground">-</span>
                    )}
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setSelectedSessionId(session.id)
                        setIsDialogOpen(true)
                      }}
                      className="flex items-center gap-1"
                    >
                      <Eye className="h-4 w-4" />
                      Детали
                    </Button>
                  </TableCell>
                </TableRow>
              ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>

      <SessionDetailDialog
        sessionId={selectedSessionId}
        open={isDialogOpen}
        onOpenChange={setIsDialogOpen}
        type={type}
        clientId={clientId}
        projectId={projectId}
      />
    </Card>
  )
}

