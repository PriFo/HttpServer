'use client'

import { useState, useEffect, useMemo } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  TableCaption,
} from '@/components/ui/table'
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
import { Progress } from '@/components/ui/progress'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { 
  Database, 
  Trash2, 
  Download, 
  RefreshCw, 
  Search, 
  Shield, 
  Link as LinkIcon,
  Package,
  HardDrive,
  Server,
  FileText,
  CheckCircle2,
  XCircle,
  Loader2,
  BarChart3
} from 'lucide-react'
import { toast } from 'sonner'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { useApiClient } from '@/hooks/useApiClient'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { formatDateTime, formatFileSize } from '@/lib/locale'
import { Tooltip, TooltipContent, TooltipTrigger, TooltipProvider } from '@/components/ui/tooltip'
import { FadeIn } from '@/components/animations/fade-in'
import { motion } from 'framer-motion'

interface DatabaseFileInfo {
  path: string
  name: string
  size: number
  modified_at: string
  type: 'main' | 'service' | 'uploaded' | 'other'
  is_protected: boolean
  linked_to_project: boolean
  client_id?: number
  project_id?: number
  project_name?: string
  database_id?: number
}

interface DatabasesFilesResponse {
  success: boolean
  total: number
  files: DatabaseFileInfo[]
  grouped: {
    main: DatabaseFileInfo[]
    service: DatabaseFileInfo[]
    uploaded: DatabaseFileInfo[]
    other: DatabaseFileInfo[]
  }
  summary: {
    main: number
    service: number
    uploaded: number
    other: number
  }
}

interface BackupInfo {
  filename: string
  size: number
  modified_at: string
}

export default function DatabasesManagePage() {
  const { get, post, request } = useApiClient()
  const [databases, setDatabases] = useState<DatabaseFileInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedPaths, setSelectedPaths] = useState<Set<string>>(new Set())
  const [filterType, setFilterType] = useState<string>('all')
  const [searchQuery, setSearchQuery] = useState('')
  const [showDeleteDialog, setShowDeleteDialog] = useState(false)
  const [showBackupDialog, setShowBackupDialog] = useState(false)
  const [deleting, setDeleting] = useState(false)
  const [backingUp, setBackingUp] = useState(false)
  const [backupProgress, setBackupProgress] = useState(0)
  const [backupStatus, setBackupStatus] = useState<string>('')
  const [deleteProgress, setDeleteProgress] = useState(0)
  const [deleteStatus, setDeleteStatus] = useState<string>('')
  const [backups, setBackups] = useState<BackupInfo[]>([])
  const [loadingBackups, setLoadingBackups] = useState(false)
  const [backupFormat, setBackupFormat] = useState<string>('both')
  const [includeMain, setIncludeMain] = useState(true)
  const [includeUploads, setIncludeUploads] = useState(true)
  const [includeService, setIncludeService] = useState(false)

  const fetchDatabases = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await get<DatabasesFilesResponse>('/api/databases/files', { skipErrorHandler: true })
      setDatabases(data.files || [])
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      setError('Не удалось загрузить список баз данных')
    } finally {
      setLoading(false)
    }
  }

  const fetchBackups = async () => {
    setLoadingBackups(true)
    try {
      const data = await get<{ backups?: BackupInfo[] } | BackupInfo[]>('/api/databases/backups', { skipErrorHandler: true })
      // Обрабатываем разные форматы ответа
      if (Array.isArray(data)) {
        setBackups(data)
      } else if (data && 'backups' in data && Array.isArray(data.backups)) {
        setBackups(data.backups)
      } else {
        setBackups([])
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Не удалось загрузить список резервных копий')
    } finally {
      setLoadingBackups(false)
    }
  }

  useEffect(() => {
    fetchDatabases()
    fetchBackups()
  }, [])

  const filteredDatabases = useMemo(() => {
    return databases.filter(db => {
      // Фильтр по типу
      if (filterType !== 'all' && db.type !== filterType) {
        return false
      }
      // Поиск
      if (searchQuery) {
        const query = searchQuery.toLowerCase()
        return (
          db.name.toLowerCase().includes(query) ||
          db.path.toLowerCase().includes(query) ||
          (db.project_name && db.project_name.toLowerCase().includes(query))
        )
      }
      return true
    })
  }, [databases, filterType, searchQuery])

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      const selectable = filteredDatabases
        .filter(db => !db.is_protected)
        .map(db => db.path)
      setSelectedPaths(new Set(selectable))
    } else {
      setSelectedPaths(new Set())
    }
  }

  const handleSelectOne = (path: string, checked: boolean) => {
    const newSelected = new Set(selectedPaths)
    if (checked) {
      newSelected.add(path)
    } else {
      newSelected.delete(path)
    }
    setSelectedPaths(newSelected)
  }

  const handleBulkDelete = async () => {
    if (selectedPaths.size === 0) {
      toast.error('Выберите базы данных для удаления')
      return
    }

    setDeleting(true)
    setDeleteProgress(0)
    setDeleteStatus('Подготовка к удалению...')
    
    let progressInterval: NodeJS.Timeout | null = null
    
    try {
      // Симуляция прогресса для лучшего UX
      progressInterval = setInterval(() => {
        setDeleteProgress(prev => {
          if (prev >= 90) {
            if (progressInterval) clearInterval(progressInterval)
            return prev
          }
          return prev + 10
        })
      }, 200)

      if (progressInterval) clearInterval(progressInterval)
      setDeleteProgress(50)
      setDeleteStatus('Обработка ответа...')

      const data = await post<{ message?: string; deleted?: number; success_count?: number; failed_count?: number }>('/api/databases/bulk-delete', {
        paths: Array.from(selectedPaths),
        confirm: true,
      }, { skipErrorHandler: true })

      setDeleteProgress(80)
      setDeleteStatus('Завершение операции...')

      const successCount = (data as any).success_count || 0
      const failedCount = (data as any).failed_count || 0

      setDeleteProgress(100)
      setDeleteStatus('Готово!')

      if (successCount > 0) {
        toast.success(`Удалено баз данных: ${successCount}`)
      }
      if (failedCount > 0) {
        toast.error(`Ошибок при удалении: ${failedCount}`)
      }

      setSelectedPaths(new Set())
      setShowDeleteDialog(false)
      
      // Небольшая задержка для визуализации завершения
      await new Promise(resolve => setTimeout(resolve, 300))
      await fetchDatabases()
    } catch (err) {
      if (progressInterval) clearInterval(progressInterval)
      setDeleteProgress(0)
      setDeleteStatus('Ошибка!')
      toast.error(err instanceof Error ? err.message : 'Ошибка удаления баз данных')
    } finally {
      // Гарантируем очистку интервала
      if (progressInterval) {
        clearInterval(progressInterval)
        progressInterval = null
      }
      setDeleting(false)
      // Небольшая задержка перед сбросом прогресса для визуализации ошибки
      setTimeout(() => {
        setDeleteProgress(0)
        setDeleteStatus('')
      }, 2000)
    }
  }

  const handleCreateBackup = async (includeMain: boolean, includeUploads: boolean, includeService: boolean, format: string = 'both') => {
    setBackingUp(true)
    setBackupProgress(0)
    setBackupStatus('Подготовка к созданию бэкапа...')
    
    let progressInterval: NodeJS.Timeout | null = null
    
    try {
      // Симуляция прогресса для лучшего UX
      progressInterval = setInterval(() => {
        setBackupProgress(prev => {
          if (prev >= 85) {
            if (progressInterval) clearInterval(progressInterval)
            return prev
          }
          return Math.min(prev + Math.random() * 15, 85)
        })
      }, 300)

      setBackupProgress(10)
      setBackupStatus('Сбор файлов для бэкапа...')

      if (progressInterval) clearInterval(progressInterval)
      setBackupProgress(70)
      setBackupStatus('Обработка ответа сервера...')

      const data = await post<{ backup?: { backup_file?: string; files_copy_dir?: string }; error?: string }>('/api/databases/backup', {
        include_main: includeMain,
        include_uploads: includeUploads,
        include_service: includeService,
        selected_files: selectedPaths.size > 0 ? Array.from(selectedPaths) : undefined,
        format: format, // "zip", "copy", "both"
      }, { skipErrorHandler: true })

      setBackupProgress(90)
      setBackupStatus('Финальная обработка...')

      const backupInfo = data.backup || {}
      let message = 'Бэкап создан успешно'
      if (backupInfo.backup_file) {
        message += `: ${backupInfo.backup_file}`
      }
      if (backupInfo.files_copy_dir) {
        message += ` (копии: ${backupInfo.files_copy_dir})`
      }

      setBackupProgress(100)
      setBackupStatus('Готово!')
      
      toast.success(message)
      
      // Небольшая задержка для визуализации завершения
      await new Promise(resolve => setTimeout(resolve, 500))
      
      setShowBackupDialog(false)
      await fetchBackups() // Обновляем список бэкапов
    } catch (err) {
      if (progressInterval) clearInterval(progressInterval)
      setBackupProgress(0)
      setBackupStatus('Ошибка!')
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
    } finally {
      // Гарантируем очистку интервала
      if (progressInterval) {
        clearInterval(progressInterval)
        progressInterval = null
      }
      setBackingUp(false)
      // Небольшая задержка перед сбросом прогресса для визуализации ошибки
      setTimeout(() => {
        setBackupProgress(0)
        setBackupStatus('')
      }, 2000)
    }
  }

  const handleDownloadBackup = async (filename: string) => {
    try {
      const response = await request(`/api/databases/backups/${filename}`, { skipErrorHandler: true })
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
      toast.success(`Бэкап ${filename} скачан`)
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Не удалось скачать резервную копию')
    }
  }

  const getTypeIcon = (type: string) => {
    switch (type) {
      case 'main':
        return <Database className="h-4 w-4" />
      case 'service':
        return <Server className="h-4 w-4" />
      case 'uploaded':
        return <Package className="h-4 w-4" />
      default:
        return <FileText className="h-4 w-4" />
    }
  }

  const getTypeLabel = (type: string) => {
    switch (type) {
      case 'main':
        return 'Основная'
      case 'service':
        return 'Сервисная'
      case 'uploaded':
        return 'Загруженная'
      default:
        return 'Другая'
    }
  }

  const breadcrumbItems = [
    { label: 'Базы данных', href: '/databases', icon: Database },
    { label: 'Управление', href: '/databases/manage', icon: HardDrive },
  ]

  const selectableCount = filteredDatabases.filter(db => !db.is_protected).length
  const allSelected = selectableCount > 0 && selectedPaths.size === selectableCount

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <div className="mb-8">
          <motion.h1
            className="text-3xl font-bold mb-2 flex items-center gap-2"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            <HardDrive className="h-8 w-8 text-primary" />
            Управление базами данных
          </motion.h1>
          <motion.p
            className="text-muted-foreground"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            Просмотр, удаление и создание бэкапов всех баз данных системы
          </motion.p>
        </div>
      </FadeIn>

      {/* Фильтры и действия */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>Фильтры и действия</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col md:flex-row gap-4 items-start md:items-center">
            <div className="flex-1 flex gap-2">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Поиск по имени или пути..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9"
                  aria-label="Поиск баз данных по имени или пути"
                  type="search"
                />
              </div>
              <Select value={filterType} onValueChange={setFilterType}>
                <SelectTrigger className="w-[180px]" aria-label="Фильтр по типу базы данных">
                  <SelectValue placeholder="Тип БД" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все типы</SelectItem>
                  <SelectItem value="main">Основные</SelectItem>
                  <SelectItem value="service">Сервисные</SelectItem>
                  <SelectItem value="uploaded">Загруженные</SelectItem>
                  <SelectItem value="other">Другие</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex gap-2 flex-wrap">
              <Button
                variant="outline"
                onClick={() => window.location.href = '/data-quality'}
                aria-label="Анализ качества данных"
                className="min-h-[44px] md:min-h-0"
              >
                <BarChart3 className="h-4 w-4 mr-2" aria-hidden="true" />
                Анализ качества данных
              </Button>
              <Button
                variant="outline"
                onClick={fetchDatabases}
                disabled={loading}
                aria-label="Обновить список баз данных"
                className="min-h-[44px] md:min-h-0"
              >
                <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} aria-hidden="true" />
                Обновить
              </Button>
              <Button
                variant="destructive"
                onClick={() => setShowDeleteDialog(true)}
                disabled={selectedPaths.size === 0 || deleting}
                aria-label={`Удалить ${selectedPaths.size} выбранных баз данных`}
                className="min-h-[44px] md:min-h-0"
              >
                <Trash2 className="h-4 w-4 mr-2" aria-hidden="true" />
                Удалить выбранные ({selectedPaths.size})
              </Button>
              <Button
                onClick={() => setShowBackupDialog(true)}
                disabled={backingUp}
                aria-label="Создать резервную копию баз данных"
                className="min-h-[44px] md:min-h-0"
              >
                <Download className="h-4 w-4 mr-2" aria-hidden="true" />
                Создать бэкап
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Статистика */}
      {!loading && databases.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Всего БД</p>
                  <p className="text-2xl font-bold">{databases.length}</p>
                </div>
                <Database className="h-8 w-8 text-muted-foreground" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Основные</p>
                  <p className="text-2xl font-bold">
                    {databases.filter(db => db.type === 'main').length}
                  </p>
                </div>
                <Database className="h-8 w-8 text-blue-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Загруженные</p>
                  <p className="text-2xl font-bold">
                    {databases.filter(db => db.type === 'uploaded').length}
                  </p>
                </div>
                <Package className="h-8 w-8 text-green-500" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm text-muted-foreground">Связано с проектами</p>
                  <p className="text-2xl font-bold">
                    {databases.filter(db => db.linked_to_project).length}
                  </p>
                </div>
                <LinkIcon className="h-8 w-8 text-purple-500" />
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Таблица */}
      {loading ? (
        <Card>
          <CardContent className="pt-6">
            <div className="flex items-center justify-center py-12">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          </CardContent>
        </Card>
      ) : error ? (
        <Alert variant="destructive">
          <AlertDescription>{error}</AlertDescription>
        </Alert>
      ) : filteredDatabases.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <div className="text-center py-12 text-muted-foreground">
              <Database className="h-16 w-16 mx-auto mb-4 opacity-50" />
              <p className="text-lg">Базы данных не найдены</p>
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle>Список баз данных ({filteredDatabases.length})</CardTitle>
                <CardDescription>
                  Выбрано: {selectedPaths.size} из {selectableCount} доступных
                </CardDescription>
              </div>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="select-all"
                  checked={allSelected}
                  onCheckedChange={handleSelectAll}
                />
                <label htmlFor="select-all" className="text-sm font-medium cursor-pointer">
                  Выбрать все
                </label>
              </div>
            </div>
          </CardHeader>
          <CardContent>
            <div className="rounded-md border overflow-x-auto -mx-4 px-4 md:mx-0 md:px-0">
              <Table>
                <TableCaption>Список всех баз данных в системе. Используйте фильтры и поиск для навигации.</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-12"></TableHead>
                    <TableHead>Имя файла</TableHead>
                    <TableHead>Тип</TableHead>
                    <TableHead>Размер</TableHead>
                    <TableHead>Изменен</TableHead>
                    <TableHead>Проект</TableHead>
                    <TableHead>Статус</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredDatabases.map((db) => {
                    const isSelected = selectedPaths.has(db.path)
                    const canSelect = !db.is_protected
                    return (
                      <TableRow key={db.path}>
                        <TableCell>
                          <Checkbox
                            checked={isSelected}
                            onCheckedChange={(checked) =>
                              handleSelectOne(db.path, checked as boolean)
                            }
                            disabled={!canSelect}
                            aria-label={canSelect ? `Выбрать базу данных ${db.name}` : `База данных ${db.name} защищена от удаления`}
                          />
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getTypeIcon(db.type)}
                            <div>
                              <div className="font-medium">{db.name}</div>
                              <div className="text-xs text-muted-foreground truncate max-w-md">
                                {db.path}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">{getTypeLabel(db.type)}</Badge>
                        </TableCell>
                        <TableCell>{formatFileSize(db.size)}</TableCell>
                        <TableCell>{formatDateTime(db.modified_at)}</TableCell>
                        <TableCell>
                          {db.linked_to_project ? (
                            <div className="flex items-center gap-1">
                              <LinkIcon className="h-3 w-3 text-green-500" />
                              <span className="text-sm">{db.project_name || 'Проект'}</span>
                            </div>
                          ) : (
                            <span className="text-muted-foreground text-sm">—</span>
                          )}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {db.is_protected && (
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger>
                                    <Shield className="h-4 w-4 text-yellow-500" />
                                  </TooltipTrigger>
                                  <TooltipContent>Защищенный файл</TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            )}
                            {db.linked_to_project ? (
                              <Badge variant="secondary" className="text-xs">
                                <LinkIcon className="h-3 w-3 mr-1" />
                                Связана
                              </Badge>
                            ) : (
                              <Badge variant="outline" className="text-xs">
                                Не связана
                              </Badge>
                            )}
                          </div>
                        </TableCell>
                      </TableRow>
                    )
                  })}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Диалог подтверждения удаления */}
      <AlertDialog 
        open={showDeleteDialog} 
        onOpenChange={(open) => {
          // Предотвращаем закрытие диалога во время операции удаления
          if (!deleting) {
            setShowDeleteDialog(open)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Подтверждение удаления</AlertDialogTitle>
            <AlertDialogDescription>
              Вы уверены, что хотите удалить {selectedPaths.size} баз данных?
              Это действие нельзя отменить. Физические файлы будут удалены с диска.
            </AlertDialogDescription>
          </AlertDialogHeader>
          {deleting && (
            <div className="space-y-2 mb-4" role="progressbar" aria-valuenow={deleteProgress} aria-valuemin={0} aria-valuemax={100} aria-label="Прогресс удаления баз данных">
              <Progress value={deleteProgress} className="w-full" />
              {deleteStatus && (
                <p className="text-sm text-muted-foreground text-center" aria-live="polite">
                  {deleteStatus}
                </p>
              )}
              <p className="text-xs text-muted-foreground text-center">
                {Math.round(deleteProgress)}% завершено
              </p>
            </div>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>Отмена</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleBulkDelete}
              disabled={deleting}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              aria-label={`Подтвердить удаление ${selectedPaths.size} баз данных`}
            >
              {deleting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" />
                  Удаление...
                </>
              ) : (
                <>
                  <Trash2 className="h-4 w-4 mr-2" aria-hidden="true" />
                  Удалить
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Диалог создания бэкапа */}
      <AlertDialog 
        open={showBackupDialog} 
        onOpenChange={(open) => {
          // Предотвращаем закрытие диалога во время операции бэкапа
          if (!backingUp) {
            setShowBackupDialog(open)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Создание бэкапа</AlertDialogTitle>
            <AlertDialogDescription>
              Выберите, какие базы данных включить в бэкап:
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="space-y-4 py-4">
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="backup-main" 
                checked={includeMain}
                onCheckedChange={(checked) => setIncludeMain(checked === true)}
                aria-label="Включить основные базы данных в бэкап"
              />
              <label htmlFor="backup-main" className="text-sm font-medium cursor-pointer">
                Включить основные БД
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="backup-uploads" 
                checked={includeUploads}
                onCheckedChange={(checked) => setIncludeUploads(checked === true)}
                aria-label="Включить загруженные базы данных в бэкап"
              />
              <label htmlFor="backup-uploads" className="text-sm font-medium cursor-pointer">
                Включить загруженные БД
              </label>
            </div>
            <div className="flex items-center space-x-2">
              <Checkbox 
                id="backup-service" 
                checked={includeService}
                onCheckedChange={(checked) => setIncludeService(checked === true)}
                aria-label="Включить service.db в бэкап"
              />
              <label htmlFor="backup-service" className="text-sm font-medium cursor-pointer">
                Включить service.db
              </label>
            </div>
            <div className="space-y-2">
              <label htmlFor="backup-format" className="text-sm font-medium">
                Формат бэкапа:
              </label>
              <Select value={backupFormat} onValueChange={setBackupFormat}>
                <SelectTrigger id="backup-format" aria-label="Выберите формат бэкапа">
                  <SelectValue placeholder="Выберите формат" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="both">ZIP архив и копии файлов</SelectItem>
                  <SelectItem value="zip">Только ZIP архив</SelectItem>
                  <SelectItem value="copy">Только копии файлов</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {selectedPaths.size > 0 && (
              <Alert>
                <AlertDescription>
                  Будет создан бэкап только выбранных файлов ({selectedPaths.size})
                </AlertDescription>
              </Alert>
            )}
            {backingUp && (
              <div className="space-y-2" role="progressbar" aria-valuenow={backupProgress} aria-valuemin={0} aria-valuemax={100} aria-label="Прогресс создания резервной копии">
                <Progress value={backupProgress} className="w-full" />
                {backupStatus && (
                  <p className="text-sm text-muted-foreground text-center" aria-live="polite">
                    {backupStatus}
                  </p>
                )}
                <p className="text-xs text-muted-foreground text-center">
                  {Math.round(backupProgress)}% завершено
                </p>
              </div>
            )}
          </div>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={backingUp}>Отмена</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                handleCreateBackup(includeMain, includeUploads, includeService, backupFormat)
              }}
              disabled={backingUp}
              aria-label="Создать резервную копию с выбранными параметрами"
            >
              {backingUp ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" aria-hidden="true" />
                  Создание...
                </>
              ) : (
                <>
                  <Download className="h-4 w-4 mr-2" aria-hidden="true" />
                  Создать бэкап
                </>
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Список последних бэкапов */}
      <Card className="mt-6">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Последние бэкапы</CardTitle>
              <CardDescription>
                Список созданных бэкапов для скачивания
              </CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={fetchBackups}
              disabled={loadingBackups}
              aria-label="Обновить список резервных копий"
            >
              <RefreshCw className={`h-4 w-4 mr-2 ${loadingBackups ? 'animate-spin' : ''}`} aria-hidden="true" />
              Обновить
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {loadingBackups ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            </div>
          ) : backups.length === 0 ? (
            <div className="text-center py-8 text-muted-foreground">
              <Package className="h-12 w-12 mx-auto mb-4 opacity-50" />
              <p>Бэкапы не найдены</p>
            </div>
          ) : (
            <div className="rounded-md border overflow-x-auto -mx-4 px-4 md:mx-0 md:px-0">
              <Table>
                <TableCaption>Список созданных резервных копий баз данных. Вы можете скачать любой бэкап.</TableCaption>
                <TableHeader>
                  <TableRow>
                    <TableHead>Имя файла</TableHead>
                    <TableHead>Размер</TableHead>
                    <TableHead>Дата создания</TableHead>
                    <TableHead className="text-right">Действия</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {backups.map((backup) => (
                    <TableRow key={backup.filename}>
                      <TableCell className="font-medium">{backup.filename}</TableCell>
                      <TableCell>{formatFileSize(backup.size)}</TableCell>
                      <TableCell>{formatDateTime(backup.modified_at)}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleDownloadBackup(backup.filename)}
                          aria-label={`Скачать резервную копию ${backup.filename}`}
                        >
                          <Download className="h-4 w-4 mr-2" aria-hidden="true" />
                          Скачать
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

