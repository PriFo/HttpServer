'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  Database,
  RefreshCw,
  Play,
  Link as LinkIcon,
  Trash2,
  AlertCircle,
  CheckCircle2,
  Clock,
  XCircle,
  FolderOpen,
} from 'lucide-react'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'

interface PendingDatabase {
  id: number
  file_path: string
  file_name: string
  file_size: number
  detected_at: string
  indexing_status: 'pending' | 'indexing' | 'completed' | 'failed'
  indexing_started_at?: string
  indexing_completed_at?: string
  error_message?: string
  client_id?: number
  project_id?: number
  moved_to_uploads: boolean
  original_path?: string
}

interface Client {
  id: number
  name: string
}

interface Project {
  id: number
  name: string
  client_id: number
}

export default function PendingDatabasesPage() {
  const [databases, setDatabases] = useState<PendingDatabase[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [scanning, setScanning] = useState(false)
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deleteTargetId, setDeleteTargetId] = useState<number | null>(null)
  const [bindDialogOpen, setBindDialogOpen] = useState(false)
  const [bindTarget, setBindTarget] = useState<PendingDatabase | null>(null)
  const [clients, setClients] = useState<Client[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedClientId, setSelectedClientId] = useState<number | ''>('')
  const [selectedProjectId, setSelectedProjectId] = useState<number | ''>('')
  const [customPath, setCustomPath] = useState('')
  const [useCustomPath, setUseCustomPath] = useState(false)

  const fetchDatabases = async () => {
    setRefreshing(true)
    try {
      const url = statusFilter && statusFilter !== 'all'
        ? `/api/databases/pending?status=${statusFilter}`
        : '/api/databases/pending'
      const response = await fetch(url)
      if (!response.ok) {
        // Если 404 или другая ошибка, просто возвращаем пустой массив
        if (response.status === 404) {
          setDatabases([])
          return
        }
        throw new Error(`Failed to fetch pending databases: ${response.status}`)
      }
      const data = await response.json()
      setDatabases(data.databases || [])
    } catch (error) {
      console.error('Failed to fetch pending databases:', error)
      // Устанавливаем пустой массив при ошибке, чтобы не показывать ошибку пользователю
      setDatabases([])
    } finally {
      setLoading(false)
      setRefreshing(false)
    }
  }

  const fetchClients = async () => {
    try {
      const response = await fetch('/api/clients')
      if (response.ok) {
        const data = await response.json()
        setClients(data || [])
      }
    } catch (error) {
      console.error('Failed to fetch clients:', error)
    }
  }

  const fetchProjects = async (clientId: number) => {
    try {
      const response = await fetch(`/api/clients/${clientId}/projects`)
      if (response.ok) {
        const data = await response.json()
        setProjects(data.projects || [])
      }
    } catch (error) {
      console.error('Failed to fetch projects:', error)
    }
  }

  useEffect(() => {
    fetchDatabases()
    fetchClients()
    // Оптимизация: увеличиваем интервал до 10 секунд для снижения нагрузки
    const interval = setInterval(fetchDatabases, 10000) // Автообновление каждые 10 секунд
    return () => clearInterval(interval)
  }, [statusFilter])

  useEffect(() => {
    if (selectedClientId) {
      fetchProjects(Number(selectedClientId))
      setSelectedProjectId('')
    } else {
      setProjects([])
    }
  }, [selectedClientId])

  const handleStartIndexing = async (id: number) => {
    try {
      const response = await fetch(`/api/databases/pending/${id}/index`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || 'Не удалось запустить индексацию')
      }
      toast.success('Индексация запущена', {
        description: 'База данных поставлена в очередь на индексацию',
      })
      await fetchDatabases()
    } catch (error) {
      console.error('Failed to start indexing:', error)
      toast.error('Ошибка запуска индексации', {
        description: error instanceof Error ? error.message : 'Не удалось запустить индексацию',
      })
    }
  }

  const handleBind = async () => {
    if (!bindTarget || !selectedClientId || !selectedProjectId) return

    try {
      const body: any = {
        client_id: Number(selectedClientId),
        project_id: Number(selectedProjectId),
      }

      if (useCustomPath && customPath) {
        body.custom_path = customPath
      }

      const response = await fetch(`/api/databases/pending/${bindTarget.id}/bind`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Не удалось привязать базу данных')
      }

      const selectedClient = clients.find(c => c.id === Number(selectedClientId))
      const selectedProject = projects.find(p => p.id === Number(selectedProjectId))

      toast.success('База данных привязана', {
        description: `База данных успешно привязана к проекту "${selectedProject?.name}" клиента "${selectedClient?.name}"`,
      })

      setBindDialogOpen(false)
      setBindTarget(null)
      setSelectedClientId('')
      setSelectedProjectId('')
      setCustomPath('')
      setUseCustomPath(false)
      await fetchDatabases()
    } catch (error) {
      console.error('Failed to bind database:', error)
      toast.error('Ошибка привязки базы данных', {
        description: error instanceof Error ? error.message : 'Не удалось привязать базу данных',
      })
    }
  }

  const handleDelete = async () => {
    if (!deleteTargetId) return

    try {
      const response = await fetch(`/api/databases/pending/${deleteTargetId}`, {
        method: 'DELETE',
      })
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({}))
        throw new Error(errorData.error || 'Не удалось удалить базу данных')
      }
      toast.success('База данных удалена', {
        description: 'База данных удалена из списка ожидающих',
      })
      setDeleteDialogOpen(false)
      setDeleteTargetId(null)
      await fetchDatabases()
    } catch (error) {
      console.error('Failed to delete:', error)
      toast.error('Ошибка удаления', {
        description: error instanceof Error ? error.message : 'Не удалось удалить базу данных',
      })
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'pending':
        return <Badge variant="outline"><Clock className="h-3 w-3 mr-1" />Ожидание</Badge>
      case 'indexing':
        return <Badge variant="default"><RefreshCw className="h-3 w-3 mr-1 animate-spin" />Индексация</Badge>
      case 'completed':
        return <Badge variant="default" className="bg-green-600"><CheckCircle2 className="h-3 w-3 mr-1" />Завершено</Badge>
      case 'failed':
        return <Badge variant="destructive"><XCircle className="h-3 w-3 mr-1" />Ошибка</Badge>
      default:
        return <Badge variant="outline">{status}</Badge>
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} B`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} KB`
    return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
  }

  if (loading) {
    return <LoadingState message="Загрузка ожидающих баз данных..." size="lg" fullScreen />
  }

  const breadcrumbItems = [
    { label: 'Базы данных', href: '/databases', icon: Database },
    { label: 'Ожидающие', href: '/databases/pending', icon: Clock },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="flex items-center justify-between"
        >
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Clock className="h-8 w-8 text-primary" />
              Ожидающие базы данных
            </h1>
            <p className="text-muted-foreground mt-2">
              Управление базами данных, ожидающими индексации и привязки к проектам
            </p>
          </div>
          <div className="flex gap-2">
          <Button
            onClick={async () => {
              setScanning(true)
              try {
                const response = await fetch('/api/databases/scan', {
                  method: 'POST',
                  headers: { 'Content-Type': 'application/json' },
                  body: JSON.stringify({ paths: ['.', 'data/uploads'] }),
                })
                
                if (!response.ok) {
                  const errorData = await response.json().catch(() => ({}))
                  throw new Error(errorData.error || 'Ошибка при сканировании')
                }

                const data = await response.json()
                const foundCount = data.found_files || 0
                
                if (foundCount > 0) {
                  toast.success('Сканирование завершено', {
                    description: `Найдено ${foundCount} ${foundCount === 1 ? 'файл' : foundCount < 5 ? 'файла' : 'файлов'}`,
                  })
                } else {
                  toast.info('Сканирование завершено', {
                    description: 'Новых файлов не найдено',
                  })
                }
                
                await fetchDatabases()
              } catch (error) {
                console.error('Failed to scan:', error)
                toast.error('Ошибка сканирования', {
                  description: error instanceof Error ? error.message : 'Не удалось выполнить сканирование',
                })
              } finally {
                setScanning(false)
              }
            }}
            variant="outline"
            disabled={scanning}
          >
            <FolderOpen className={`h-4 w-4 mr-2 ${scanning ? 'animate-pulse' : ''}`} />
            {scanning ? 'Сканирование...' : 'Сканировать файлы'}
          </Button>
          <Button onClick={fetchDatabases} variant="outline" disabled={refreshing || scanning}>
            <RefreshCw className={`h-4 w-4 mr-2 ${refreshing ? 'animate-spin' : ''}`} />
            Обновить
          </Button>
        </div>
      </motion.div>
      </FadeIn>

      {/* Фильтр по статусу */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <Label>Фильтр по статусу:</Label>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[200px]">
                <SelectValue placeholder="Все статусы" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все статусы</SelectItem>
                <SelectItem value="pending">Ожидание</SelectItem>
                <SelectItem value="indexing">Индексация</SelectItem>
                <SelectItem value="completed">Завершено</SelectItem>
                <SelectItem value="failed">Ошибка</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </CardContent>
      </Card>

      {/* Список баз данных */}
      {databases.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <EmptyState
              icon={Database}
              title="Нет ожидающих баз данных"
              description="Все базы данных обработаны или еще не обнаружены"
            />
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4">
          {databases.map((db, index) => (
            <Card key={`pending-db-${db.id}-${db.file_path || db.file_name}-${index}`}>
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div className="flex-1">
                    <CardTitle className="flex items-center gap-2">
                      <Database className="h-5 w-5" />
                      {db.file_name}
                    </CardTitle>
                    <CardDescription className="mt-2 font-mono text-xs">
                      {db.file_path}
                    </CardDescription>
                  </div>
                  {getStatusBadge(db.indexing_status)}
                </div>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
                  <div>
                    <div className="text-muted-foreground">Размер</div>
                    <div className="font-medium">{formatFileSize(db.file_size)}</div>
                  </div>
                  <div>
                    <div className="text-muted-foreground">Обнаружено</div>
                    <div className="font-medium">
                      {new Date(db.detected_at).toLocaleString('ru-RU')}
                    </div>
                  </div>
                  {db.indexing_started_at && (
                    <div>
                      <div className="text-muted-foreground">Начало индексации</div>
                      <div className="font-medium">
                        {new Date(db.indexing_started_at).toLocaleString('ru-RU')}
                      </div>
                    </div>
                  )}
                  {db.moved_to_uploads && (
                    <div>
                      <div className="text-muted-foreground">Перемещено</div>
                      <div className="font-medium text-green-600">В uploads</div>
                    </div>
                  )}
                </div>

                {db.error_message && (
                  <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>{db.error_message}</AlertDescription>
                  </Alert>
                )}

                <div className="flex gap-2">
                  {db.indexing_status === 'pending' && (
                    <Button
                      onClick={() => handleStartIndexing(db.id)}
                      size="sm"
                      variant="outline"
                    >
                      <Play className="h-4 w-4 mr-2" />
                      Начать индексацию
                    </Button>
                  )}
                  <Button
                    onClick={() => {
                      setBindTarget(db)
                      setBindDialogOpen(true)
                    }}
                    size="sm"
                  >
                    <LinkIcon className="h-4 w-4 mr-2" />
                    Привязать к проекту
                  </Button>
                  <Button
                    onClick={() => {
                      setDeleteTargetId(db.id)
                      setDeleteDialogOpen(true)
                    }}
                    size="sm"
                    variant="destructive"
                  >
                    <Trash2 className="h-4 w-4 mr-2" />
                    Удалить
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {/* Диалог удаления */}
      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Удалить базу данных?</AlertDialogTitle>
            <AlertDialogDescription>
              Это действие нельзя отменить. База данных будет удалена из списка ожидающих.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Отмена</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} className="bg-destructive">
              Удалить
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Диалог привязки */}
      <AlertDialog open={bindDialogOpen} onOpenChange={setBindDialogOpen}>
        <AlertDialogContent className="max-w-2xl">
          <AlertDialogHeader>
            <AlertDialogTitle>Привязать базу данных к проекту</AlertDialogTitle>
            <AlertDialogDescription>
              Выберите клиента и проект для привязки базы данных.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Клиент *</Label>
              <Select
                value={selectedClientId.toString()}
                onValueChange={(v) => setSelectedClientId(v ? Number(v) : '')}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Выберите клиента" />
                </SelectTrigger>
                <SelectContent>
                  {clients.map((client) => (
                    <SelectItem key={client.id} value={client.id.toString()}>
                      {client.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>Проект *</Label>
              <Select
                value={selectedProjectId.toString()}
                onValueChange={(v) => setSelectedProjectId(v ? Number(v) : '')}
                disabled={!selectedClientId}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Выберите проект" />
                </SelectTrigger>
                <SelectContent>
                  {projects.map((project) => (
                    <SelectItem key={project.id} value={project.id.toString()}>
                      {project.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                <input
                  type="checkbox"
                  id="use-custom-path"
                  checked={useCustomPath}
                  onChange={(e) => setUseCustomPath(e.target.checked)}
                  className="rounded"
                />
                <Label htmlFor="use-custom-path" className="cursor-pointer">
                  Указать свой путь к файлу
                </Label>
              </div>
              {useCustomPath && (
                <div className="space-y-2">
                  <Label htmlFor="custom-path">Путь к файлу</Label>
                  <Input
                    id="custom-path"
                    value={customPath}
                    onChange={(e) => setCustomPath(e.target.value)}
                    placeholder="E:\HttpServer\data\custom\file.db"
                  />
                  <p className="text-xs text-muted-foreground">
                    Если не указано, файл будет перемещен в data/uploads/
                  </p>
                </div>
              )}
            </div>
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel>Отмена</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleBind}
              disabled={!selectedClientId || !selectedProjectId}
            >
              Привязать
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

