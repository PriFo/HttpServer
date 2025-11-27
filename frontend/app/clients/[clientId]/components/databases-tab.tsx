'use client'

import { useState, useEffect, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { 
  Database, 
  ChevronDown,
  ChevronRight,
  Eye,
  FileText,
  Layers,
  Link2,
  Zap,
  AlertCircle,
  Loader2,
  CheckCircle2,
  Unlink,
  Search,
  X,
  Filter,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Settings,
  CheckSquare,
  Square,
  Download,
  Table
} from "lucide-react"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { formatNumber } from "@/lib/locale"
import { LoadingState } from "@/components/common/loading-state"
import { ErrorState } from "@/components/common/error-state"
import { DatabaseDetailDialog } from "./database-detail-dialog"
import { formatDate } from "@/lib/locale"
import { Label } from "@/components/ui/label"
import { toast } from "sonner"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"

interface ProjectDatabase {
  id: number
  name: string
  path: string
  size?: number
  created_at: string
  status: string
  project_id: number
  project_name: string
  config_name?: string // Конфигурация 1С
  display_name?: string // Отображаемое название конфигурации
  stats?: {
    total_uploads?: number
    uploads_count?: number
    total_catalogs?: number
    catalogs_count?: number
    total_items?: number
    items_count?: number
    last_upload_date?: string
    avg_items_per_upload?: number
  }
  tables?: Array<{
    name: string
    row_count?: number
  }>
  statistics?: {
    total_tables: number
    total_rows: number
  }
}

interface DatabasesTabProps {
  clientId: string
  projects: Array<{
    id: number
    name: string
    project_type: string
    status: string
  }>
}

export function DatabasesTab({ clientId, projects }: DatabasesTabProps) {
  const [databases, setDatabases] = useState<ProjectDatabase[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null)
  const [expandedProjects, setExpandedProjects] = useState<Set<number>>(new Set())
  const [selectedDatabase, setSelectedDatabase] = useState<ProjectDatabase | null>(null)
  const [linkDialogOpen, setLinkDialogOpen] = useState(false)
  const [databaseToLink, setDatabaseToLink] = useState<ProjectDatabase | null>(null)
  const [selectedLinkProjectId, setSelectedLinkProjectId] = useState<number | null>(null)
  const [isLinking, setIsLinking] = useState(false)
  const [isAutoLinking, setIsAutoLinking] = useState(false)
  const [isUnlinking, setIsUnlinking] = useState(false)
  const [databaseToUnlink, setDatabaseToUnlink] = useState<ProjectDatabase | null>(null)
  const [unlinkDialogOpen, setUnlinkDialogOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [statusFilter, setStatusFilter] = useState<'all' | 'active' | 'inactive'>('all')
  const [configFilter, setConfigFilter] = useState<string>('all')
  const [sortBy, setSortBy] = useState<'name' | 'date' | 'size' | 'status'>('date')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [selectedDatabases, setSelectedDatabases] = useState<Set<number>>(new Set())
  const [isBulkLinking, setIsBulkLinking] = useState(false)
  const [isBulkUnlinking, setIsBulkUnlinking] = useState(false)
  const [bulkLinkDialogOpen, setBulkLinkDialogOpen] = useState(false)

  useEffect(() => {
    fetchDatabases()
  }, [clientId, selectedProjectId])

  const fetchDatabases = async () => {
    setIsLoading(true)
    setError(null)
    try {
      let url = `/api/clients/${clientId}/databases`
      if (selectedProjectId) {
        url = `/api/clients/${clientId}/projects/${selectedProjectId}/databases`
      }
      
      const response = await fetch(url)
      if (!response.ok) {
        throw new Error('Failed to fetch databases')
      }
      const data = await response.json()
      
      // Если это массив, используем его, иначе преобразуем
      const dbList = Array.isArray(data) ? data : (data.databases || [])
      setDatabases(dbList)
    } catch (error) {
      console.error('Failed to fetch databases:', error)
      setError(error instanceof Error ? error.message : 'Не удалось загрузить базы данных')
    } finally {
      setIsLoading(false)
    }
  }

  const toggleProject = (projectId: number) => {
    const newExpanded = new Set(expandedProjects)
    if (newExpanded.has(projectId)) {
      newExpanded.delete(projectId)
    } else {
      newExpanded.add(projectId)
    }
    setExpandedProjects(newExpanded)
  }

  const formatFileSize = (bytes?: number) => {
    if (!bytes) return 'Неизвестно'
    const kb = bytes / 1024
    const mb = kb / 1024
    const gb = mb / 1024
    if (gb >= 1) return `${gb.toFixed(2)} GB`
    if (mb >= 1) return `${mb.toFixed(2)} MB`
    return `${kb.toFixed(2)} KB`
  }

  const handleLinkDatabase = (db: ProjectDatabase) => {
    setDatabaseToLink(db)
    setSelectedLinkProjectId(null)
    setLinkDialogOpen(true)
  }

  const handleConfirmLink = async () => {
    if (!databaseToLink || !selectedLinkProjectId) return

    setIsLinking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/databases/${databaseToLink.id}/link`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ project_id: selectedLinkProjectId }),
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Не удалось привязать базу данных')
      }

      // Обновляем список баз данных
      await fetchDatabases()
      setLinkDialogOpen(false)
      setDatabaseToLink(null)
      setSelectedLinkProjectId(null)
      
      // Показываем успешное уведомление
      const selectedProject = projects.find(p => p.id === selectedLinkProjectId)
      toast.success('База данных привязана', {
        description: `База данных "${databaseToLink.name}" успешно привязана к проекту "${selectedProject?.name || 'неизвестный проект'}"`,
        duration: 4000,
      })
    } catch (error) {
      console.error('Failed to link database:', error)
      const errorMessage = error instanceof Error ? error.message : 'Не удалось привязать базу данных'
      setError(errorMessage)
      toast.error('Ошибка привязки', {
        description: errorMessage,
        duration: 5000,
      })
    } finally {
      setIsLinking(false)
    }
  }

  const handleAutoLink = async () => {
    setIsAutoLinking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/databases/auto-link`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Не удалось выполнить автоматическую привязку')
      }

      const result = await response.json()
      console.log('Auto-link result:', result)

      // Показываем результат пользователю через toast
      if (result.linked_count > 0) {
        const totalDatabases = result.total_databases || 0
        const unlinkedCount = result.unlinked_count || (totalDatabases - result.linked_count)
        
        if (result.errors && result.errors.length > 0) {
          toast.success('Автоматическая привязка завершена', {
            description: `Привязано баз: ${result.linked_count} из ${totalDatabases}. Осталось непривязанных: ${unlinkedCount}. Ошибок: ${result.errors.length}`,
            duration: 6000,
          })
          
          // Показываем детали ошибок в консоли или отдельном toast
          if (result.errors.length <= 3) {
            result.errors.forEach((error: string, index: number) => {
              setTimeout(() => {
                toast.error(`Ошибка ${index + 1}`, {
                  description: error,
                  duration: 5000,
                })
              }, 1000 * (index + 1))
            })
          } else {
            toast.warning('Множественные ошибки', {
              description: `Всего ошибок: ${result.errors.length}. Проверьте консоль для деталей.`,
              duration: 5000,
            })
            console.error('Ошибки привязки:', result.errors)
          }
        } else {
          toast.success('Автоматическая привязка завершена', {
            description: `Успешно привязано баз: ${result.linked_count} из ${totalDatabases}. Осталось непривязанных: ${unlinkedCount}`,
            duration: 5000,
          })
        }
      } else if (result.errors && result.errors.length > 0) {
        toast.error('Ошибка автоматической привязки', {
          description: `Не удалось привязать базы данных. Ошибок: ${result.errors.length}`,
          duration: 6000,
        })
        console.error('Ошибки привязки:', result.errors)
      } else {
        toast.info('Нет баз для привязки', {
          description: 'Все базы данных уже привязаны к проектам или нет подходящих баз для автоматической привязки.',
          duration: 4000,
        })
      }

      // Обновляем список баз данных
      await fetchDatabases()
    } catch (error) {
      console.error('Failed to auto-link databases:', error)
      setError(error instanceof Error ? error.message : 'Не удалось выполнить автоматическую привязку')
    } finally {
      setIsAutoLinking(false)
    }
  }

  const handleUnlinkDatabase = (db: ProjectDatabase) => {
    setDatabaseToUnlink(db)
    setUnlinkDialogOpen(true)
  }

  const handleConfirmUnlink = async () => {
    if (!databaseToUnlink) return

    setIsUnlinking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/databases/${databaseToUnlink.id}/link`, {
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
      })

      if (!response.ok) {
        const error = await response.json()
        throw new Error(error.error || 'Не удалось отвязать базу данных')
      }

      // Обновляем список баз данных
      await fetchDatabases()
      setUnlinkDialogOpen(false)
      setDatabaseToUnlink(null)
      
      // Показываем успешное уведомление
      toast.success('База данных отвязана', {
        description: `База данных "${databaseToUnlink.name}" успешно отвязана от проекта`,
        duration: 3000,
      })
    } catch (error) {
      console.error('Failed to unlink database:', error)
      const errorMessage = error instanceof Error ? error.message : 'Не удалось отвязать базу данных'
      setError(errorMessage)
      toast.error('Ошибка отвязки', {
        description: errorMessage,
        duration: 5000,
      })
    } finally {
      setIsUnlinking(false)
    }
  }

  // Массовые операции
  const toggleDatabaseSelection = (dbId: number) => {
    const newSelected = new Set(selectedDatabases)
    if (newSelected.has(dbId)) {
      newSelected.delete(dbId)
    } else {
      newSelected.add(dbId)
    }
    setSelectedDatabases(newSelected)
  }

  const handleBulkLink = () => {
    if (selectedDatabases.size === 0) {
      toast.warning('Не выбрано баз данных', {
        description: 'Выберите хотя бы одну базу данных для привязки',
        duration: 3000,
      })
      return
    }
    setBulkLinkDialogOpen(true)
  }

  const handleConfirmBulkLink = async () => {
    if (!selectedLinkProjectId || selectedDatabases.size === 0) return

    setIsBulkLinking(true)
    const errors: string[] = []
    let successCount = 0
    const selectedCount = selectedDatabases.size
    const databasesToLink = Array.from(selectedDatabases)
    const projectId = selectedLinkProjectId

    try {
      // Привязываем каждую базу данных
      for (const dbId of databasesToLink) {
        try {
          const response = await fetch(`/api/clients/${clientId}/databases/${dbId}/link`, {
            method: 'PUT',
            headers: {
              'Content-Type': 'application/json',
            },
            body: JSON.stringify({ project_id: projectId }),
          })

          if (!response.ok) {
            const error = await response.json()
            const db = databases.find(d => d.id === dbId)
            errors.push(`${db?.name || `БД #${dbId}`}: ${error.error || 'Ошибка привязки'}`)
          } else {
            successCount++
          }
        } catch (error) {
          const db = databases.find(d => d.id === dbId)
          errors.push(`${db?.name || `БД #${dbId}`}: ${error instanceof Error ? error.message : 'Неизвестная ошибка'}`)
        }
      }

      // Обновляем список баз данных
      await fetchDatabases()
      setBulkLinkDialogOpen(false)
      setSelectedDatabases(new Set())
      setSelectedLinkProjectId(null)

      // Показываем результат
      if (successCount > 0) {
        const selectedProject = projects.find(p => p.id === projectId)
        toast.success('Массовая привязка завершена', {
          description: `Успешно привязано: ${successCount} из ${selectedCount}. ${errors.length > 0 ? `Ошибок: ${errors.length}` : ''}`,
          duration: 5000,
        })
      }

      if (errors.length > 0) {
        if (errors.length <= 3) {
          errors.forEach((error, index) => {
            setTimeout(() => {
              toast.error(`Ошибка ${index + 1}`, {
                description: error,
                duration: 5000,
              })
            }, 1000 * (index + 1))
          })
        } else {
          toast.error('Множественные ошибки', {
            description: `Всего ошибок: ${errors.length}. Проверьте консоль для деталей.`,
            duration: 5000,
          })
          console.error('Ошибки массовой привязки:', errors)
        }
      }
    } catch (error) {
      console.error('Failed to bulk link databases:', error)
      toast.error('Ошибка массовой привязки', {
        description: error instanceof Error ? error.message : 'Не удалось выполнить массовую привязку',
        duration: 5000,
      })
    } finally {
      setIsBulkLinking(false)
    }
  }

  const handleExportDatabases = () => {
    try {
      // Подготавливаем данные для экспорта
      const exportData = filteredDatabases.map(db => ({
        'ID': db.id,
        'Название': db.name,
        'Путь': db.path,
        'Размер (байт)': db.size || 0,
        'Размер (MB)': db.size ? (db.size / 1024 / 1024).toFixed(2) : '0',
        'Статус': db.status,
        'ID проекта': db.project_id || '',
        'Название проекта': db.project_name || 'Непривязана',
        'Конфигурация 1С': db.display_name || db.config_name || '',
        'Техническое имя': db.config_name || '',
        'Дата создания': formatDate(db.created_at),
        'Выгрузок': db.stats?.total_uploads || db.stats?.uploads_count || 0,
        'Справочников': db.stats?.total_catalogs || db.stats?.catalogs_count || 0,
        'Записей': db.stats?.total_items || db.stats?.items_count || 0,
        'Последняя выгрузка': db.stats?.last_upload_date ? formatDate(db.stats.last_upload_date) : '',
      }))

      // Конвертируем в CSV
      const headers = Object.keys(exportData[0] || {})
      const csvRows = [
        headers.join(','),
        ...exportData.map(row => 
          headers.map(header => {
            const value = row[header as keyof typeof row]
            // Экранируем кавычки и запятые
            if (typeof value === 'string' && (value.includes(',') || value.includes('"') || value.includes('\n'))) {
              return `"${value.replace(/"/g, '""')}"`
            }
            return value
          }).join(',')
        )
      ]
      
      const csvContent = csvRows.join('\n')
      const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
      const url = URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = `databases_${clientId}_${new Date().toISOString().split('T')[0]}.csv`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)

      toast.success('Экспорт завершен', {
        description: `Экспортировано ${exportData.length} баз данных`,
        duration: 3000,
      })
    } catch (error) {
      console.error('Failed to export databases:', error)
      toast.error('Ошибка экспорта', {
        description: error instanceof Error ? error.message : 'Не удалось экспортировать данные',
        duration: 5000,
      })
    }
  }

  const handleBulkUnlink = async () => {
    if (selectedDatabases.size === 0) {
      toast.warning('Не выбрано баз данных', {
        description: 'Выберите хотя бы одну базу данных для отвязки',
        duration: 3000,
      })
      return
    }

    const selectedCount = selectedDatabases.size
    if (!confirm(`Вы уверены, что хотите отвязать ${selectedCount} баз данных от проектов?`)) {
      return
    }

    setIsBulkUnlinking(true)
    const errors: string[] = []
    let successCount = 0
    const databasesToUnlink = Array.from(selectedDatabases)

    try {
      // Отвязываем каждую базу данных
      for (const dbId of databasesToUnlink) {
        try {
          const response = await fetch(`/api/clients/${clientId}/databases/${dbId}/link`, {
            method: 'DELETE',
            headers: {
              'Content-Type': 'application/json',
            },
          })

          if (!response.ok) {
            const error = await response.json()
            const db = databases.find(d => d.id === dbId)
            errors.push(`${db?.name || `БД #${dbId}`}: ${error.error || 'Ошибка отвязки'}`)
          } else {
            successCount++
          }
        } catch (error) {
          const db = databases.find(d => d.id === dbId)
          errors.push(`${db?.name || `БД #${dbId}`}: ${error instanceof Error ? error.message : 'Неизвестная ошибка'}`)
        }
      }

      // Обновляем список баз данных
      await fetchDatabases()
      setSelectedDatabases(new Set())

      // Показываем результат
      if (successCount > 0) {
        toast.success('Массовая отвязка завершена', {
          description: `Успешно отвязано: ${successCount} из ${selectedCount}. ${errors.length > 0 ? `Ошибок: ${errors.length}` : ''}`,
          duration: 5000,
        })
      }

      if (errors.length > 0) {
        if (errors.length <= 3) {
          errors.forEach((error, index) => {
            setTimeout(() => {
              toast.error(`Ошибка ${index + 1}`, {
                description: error,
                duration: 5000,
              })
            }, 1000 * (index + 1))
          })
        } else {
          toast.error('Множественные ошибки', {
            description: `Всего ошибок: ${errors.length}. Проверьте консоль для деталей.`,
            duration: 5000,
          })
          console.error('Ошибки массовой отвязки:', errors)
        }
      }
    } catch (error) {
      console.error('Failed to bulk unlink databases:', error)
      toast.error('Ошибка массовой отвязки', {
        description: error instanceof Error ? error.message : 'Не удалось выполнить массовую отвязку',
        duration: 5000,
      })
    } finally {
      setIsBulkUnlinking(false)
    }
  }

  // Фильтрация баз данных
  const filteredDatabases = useMemo(() => {
    let filtered = [...databases]

    // Фильтр по поисковому запросу
    if (searchQuery.trim()) {
      const query = searchQuery.toLowerCase().trim()
      filtered = filtered.filter(db => 
        db.name.toLowerCase().includes(query) ||
        db.path.toLowerCase().includes(query) ||
        (db.config_name && db.config_name.toLowerCase().includes(query)) ||
        (db.display_name && db.display_name.toLowerCase().includes(query))
      )
    }

    // Фильтр по статусу
    if (statusFilter !== 'all') {
      filtered = filtered.filter(db => db.status === statusFilter)
    }

    // Фильтр по конфигурации
    if (configFilter !== 'all') {
      filtered = filtered.filter(db => 
        (db.config_name && db.config_name === configFilter) ||
        (db.display_name && db.display_name === configFilter)
      )
    }

    // Сортировка
    filtered.sort((a, b) => {
      let comparison = 0
      
      switch (sortBy) {
        case 'name':
          comparison = a.name.localeCompare(b.name, 'ru', { sensitivity: 'base' })
          break
        case 'date':
          comparison = new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
          break
        case 'size':
          comparison = (a.size || 0) - (b.size || 0)
          break
        case 'status':
          comparison = a.status.localeCompare(b.status, 'ru', { sensitivity: 'base' })
          break
        default:
          comparison = 0
      }
      
      return sortOrder === 'asc' ? comparison : -comparison
    })

    return filtered
  }, [databases, searchQuery, statusFilter, configFilter, sortBy, sortOrder])

  // Получаем непривязанные базы данных (исключаем их из группировки по проектам)
  const unlinkedDatabases = filteredDatabases.filter(db => !db.project_id || db.project_id === 0)
  const linkedDatabases = filteredDatabases.filter(db => db.project_id && db.project_id > 0)

  // Получаем уникальные конфигурации для фильтра
  const availableConfigs = useMemo(() => {
    const configs = new Set<string>()
    databases.forEach(db => {
      if (db.config_name) configs.add(db.config_name)
      if (db.display_name) configs.add(db.display_name)
    })
    return Array.from(configs).sort()
  }, [databases])

  // Функция для получения цвета конфигурации
  const getConfigColor = (configName?: string) => {
    if (!configName) return 'default'
    const name = configName.toLowerCase()
    if (name.includes('erp') || name.includes('управление')) return 'blue'
    if (name.includes('бух') || name.includes('accounting')) return 'green'
    if (name.includes('розница') || name.includes('retail')) return 'purple'
    if (name.includes('зарплата') || name.includes('salary')) return 'orange'
    return 'default'
  }

  // Функция для получения варианта Badge по цвету
  const getConfigBadgeVariant = (configName?: string): "default" | "secondary" | "destructive" | "outline" => {
    const color = getConfigColor(configName)
    if (color === 'default') return 'outline'
    return 'outline'
  }

  // Группировка баз данных по проектам
  const groupedDatabases = linkedDatabases.reduce((acc, db) => {
    const projectId = db.project_id || 0
    if (!acc[projectId]) {
      acc[projectId] = {
        projectId,
        projectName: db.project_name || 'Неизвестный проект',
        databases: []
      }
    }
    acc[projectId].databases.push(db)
    return acc
  }, {} as Record<number, { projectId: number; projectName: string; databases: ProjectDatabase[] }>)

  if (isLoading) {
    return <LoadingState message="Загрузка баз данных..." />
  }

  if (error) {
    return (
      <ErrorState
        title="Ошибка загрузки"
        message={error}
        action={{
          label: 'Повторить',
          onClick: fetchDatabases,
        }}
        variant="destructive"
      />
    )
  }

  return (
    <div className="space-y-4">
      {/* Фильтры и поиск */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <Filter className="h-4 w-4" />
            Фильтры и поиск
          </CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col sm:flex-row gap-4 items-start sm:items-center">
            {/* Поиск */}
            <div className="relative flex-1 w-full sm:w-auto">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Поиск по имени, пути или конфигурации..."
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

            {/* Фильтр по проекту */}
            <Select
              value={selectedProjectId?.toString() || "all"}
              onValueChange={(value) => setSelectedProjectId(value === "all" ? null : parseInt(value))}
            >
              <SelectTrigger className="w-full sm:w-[200px]">
                <SelectValue placeholder="Проект" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все проекты</SelectItem>
                {projects.map((project) => (
                  <SelectItem key={project.id} value={project.id.toString()}>
                    {project.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {/* Фильтр по статусу */}
            <Select
              value={statusFilter}
              onValueChange={(value) => setStatusFilter(value as typeof statusFilter)}
            >
              <SelectTrigger className="w-full sm:w-[150px]">
                <SelectValue placeholder="Статус" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">Все статусы</SelectItem>
                <SelectItem value="active">Активные</SelectItem>
                <SelectItem value="inactive">Неактивные</SelectItem>
              </SelectContent>
            </Select>

            {/* Фильтр по конфигурации */}
            {availableConfigs.length > 0 && (
              <Select
                value={configFilter}
                onValueChange={setConfigFilter}
              >
                <SelectTrigger className="w-full sm:w-[200px]">
                  <SelectValue placeholder="Конфигурация" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все конфигурации</SelectItem>
                  {availableConfigs.map((config) => (
                    <SelectItem key={config} value={config}>
                      {config}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}

            {/* Сортировка */}
            <div className="flex items-center gap-2">
              <Select
                value={sortBy}
                onValueChange={(value) => setSortBy(value as typeof sortBy)}
              >
                <SelectTrigger className="w-full sm:w-[140px]">
                  <SelectValue placeholder="Сортировать" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="date">По дате</SelectItem>
                  <SelectItem value="name">По имени</SelectItem>
                  <SelectItem value="size">По размеру</SelectItem>
                  <SelectItem value="status">По статусу</SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant="outline"
                size="icon"
                onClick={() => setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')}
                className="shrink-0"
              >
                {sortOrder === 'asc' ? (
                  <ArrowUp className="h-4 w-4" />
                ) : (
                  <ArrowDown className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>

          {/* Кнопка автоматической привязки и счетчики */}
          <div className="flex items-center justify-between flex-wrap gap-2">
            <div className="flex items-center gap-4 text-sm text-muted-foreground">
              <span>
                Всего: <span className="font-medium text-foreground">{databases.length}</span>
                {filteredDatabases.length !== databases.length && (
                  <> | Показано: <span className="font-medium text-foreground">{filteredDatabases.length}</span></>
                )}
              </span>
              {unlinkedDatabases.length > 0 && (
                <span>Непривязанных: <span className="font-medium text-orange-600 dark:text-orange-400">{unlinkedDatabases.length}</span></span>
              )}
              {selectedDatabases.size > 0 && (
                <span className="text-primary font-medium">
                  Выбрано: {selectedDatabases.size}
                </span>
              )}
            </div>
            <div className="flex items-center gap-2 flex-wrap">
              <Button
                onClick={handleExportDatabases}
                variant="outline"
                size="sm"
                disabled={filteredDatabases.length === 0}
              >
                <Download className="h-4 w-4 mr-2" />
                Экспорт CSV
              </Button>
              {selectedDatabases.size > 0 && (
                <>
                  <Button
                    onClick={handleBulkLink}
                    disabled={isBulkLinking || isBulkUnlinking}
                    variant="default"
                    size="sm"
                  >
                    {isBulkLinking ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Привязка...
                      </>
                    ) : (
                      <>
                        <Link2 className="h-4 w-4 mr-2" />
                        Привязать ({selectedDatabases.size})
                      </>
                    )}
                  </Button>
                  <Button
                    onClick={handleBulkUnlink}
                    disabled={isBulkLinking || isBulkUnlinking}
                    variant="outline"
                    size="sm"
                    className="text-orange-600 hover:text-orange-700 dark:text-orange-400"
                  >
                    {isBulkUnlinking ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Отвязка...
                      </>
                    ) : (
                      <>
                        <Unlink className="h-4 w-4 mr-2" />
                        Отвязать ({selectedDatabases.size})
                      </>
                    )}
                  </Button>
                  <Button
                    onClick={() => setSelectedDatabases(new Set())}
                    variant="ghost"
                    size="sm"
                  >
                    <X className="h-4 w-4 mr-2" />
                    Снять выбор
                  </Button>
                </>
              )}
              {unlinkedDatabases.length > 0 && (
                <Button
                  onClick={handleAutoLink}
                  disabled={isAutoLinking || isBulkLinking || isBulkUnlinking}
                  variant="outline"
                  className="w-full sm:w-auto"
                >
                  {isAutoLinking ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Привязка...
                    </>
                  ) : (
                    <>
                      <Link2 className="h-4 w-4 mr-2" />
                      Автоматическая привязка ({unlinkedDatabases.length})
                    </>
                  )}
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Непривязанные базы данных - отдельная секция */}
      {unlinkedDatabases.length > 0 && (
        <Card className="border-orange-200 dark:border-orange-900">
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5 text-orange-500" />
              <CardTitle className="text-base">Непривязанные базы данных</CardTitle>
              <Badge variant="outline" className="bg-orange-50 dark:bg-orange-950">
                {unlinkedDatabases.length}
              </Badge>
            </div>
            <div className="flex items-center gap-2">
              {unlinkedDatabases.length > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    const allSelected = unlinkedDatabases.every(db => selectedDatabases.has(db.id))
                    if (allSelected) {
                      const newSelected = new Set(selectedDatabases)
                      unlinkedDatabases.forEach(db => newSelected.delete(db.id))
                      setSelectedDatabases(newSelected)
                    } else {
                      const newSelected = new Set(selectedDatabases)
                      unlinkedDatabases.forEach(db => newSelected.add(db.id))
                      setSelectedDatabases(newSelected)
                    }
                  }}
                  className="text-xs"
                >
                  {unlinkedDatabases.every(db => selectedDatabases.has(db.id)) ? (
                    <>
                      <CheckSquare className="h-3.5 w-3.5 mr-1" />
                      Снять все
                    </>
                  ) : (
                    <>
                      <Square className="h-3.5 w-3.5 mr-1" />
                      Выбрать все
                    </>
                  )}
                </Button>
              )}
              <Button
                variant="outline"
                size="sm"
                onClick={handleAutoLink}
                disabled={isAutoLinking}
              >
                {isAutoLinking ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Привязка...
                  </>
                ) : (
                  <>
                    <Zap className="h-4 w-4 mr-2" />
                    Автоматическая привязка
                  </>
                )}
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-3">
              {unlinkedDatabases.map((db, index) => (
                <div
                  key={`unlinked-db-${db.id}-${index}`}
                  className={`flex items-start justify-between p-4 border rounded-lg hover:bg-muted/50 transition-colors bg-orange-50/50 dark:bg-orange-950/20 ${
                    selectedDatabases.has(db.id) ? 'ring-2 ring-primary' : ''
                  }`}
                >
                  <div className="flex items-start gap-3 flex-1">
                    <Checkbox
                      checked={selectedDatabases.has(db.id)}
                      onCheckedChange={() => toggleDatabaseSelection(db.id)}
                      className="mt-1"
                    />
                    <div className="flex-1">
                    <div className="font-medium flex items-center gap-2 mb-2">
                      <Database className="h-4 w-4 text-orange-600 dark:text-orange-400" />
                      <span>{db.name}</span>
                      {db.config_name && (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Badge 
                              variant={getConfigBadgeVariant(db.config_name)} 
                              className="ml-2 flex items-center gap-1"
                            >
                              <Settings className="h-3 w-3" />
                              {db.display_name || db.config_name}
                            </Badge>
                          </TooltipTrigger>
                          <TooltipContent>
                            <p>Конфигурация 1С: {db.display_name || db.config_name}</p>
                            {db.config_name !== db.display_name && (
                              <p className="text-xs text-muted-foreground mt-1">
                                Техническое имя: {db.config_name}
                              </p>
                            )}
                          </TooltipContent>
                        </Tooltip>
                      )}
                    </div>
                    <div className="text-sm text-muted-foreground space-y-1">
                      <div className="flex items-center gap-1">
                        <FileText className="h-3.5 w-3.5" />
                        <span className="font-mono text-xs truncate max-w-md">{db.path}</span>
                      </div>
                      <div className="flex items-center gap-4">
                        {db.size && (
                          <div className="flex items-center gap-1">
                            <span>Размер:</span>
                            <span className="font-medium">{formatFileSize(db.size)}</span>
                          </div>
                        )}
                        <div className="flex items-center gap-1">
                          <span>Создано:</span>
                          <span>{formatDate(db.created_at)}</span>
                        </div>
                      </div>
                      {db.stats && (db.stats.total_uploads !== undefined || db.stats.uploads_count !== undefined) && (
                        <div className="flex items-center gap-1.5 text-xs">
                          <Layers className="h-3.5 w-3.5" />
                          <span>{formatNumber(db.stats.total_uploads ?? db.stats.uploads_count ?? 0)} выгрузок</span>
                        </div>
                      )}
                      {db.statistics && (
                        <div className="flex items-center gap-1.5 text-xs">
                          <Table className="h-3.5 w-3.5" />
                          <span>
                            {db.statistics.total_tables} {db.statistics.total_tables === 1 ? 'таблица' : db.statistics.total_tables < 5 ? 'таблицы' : 'таблиц'}
                            {db.statistics.total_rows > 0 && (
                              <span className="ml-1">
                                • {formatNumber(db.statistics.total_rows)} {db.statistics.total_rows === 1 ? 'запись' : db.statistics.total_rows < 5 ? 'записи' : 'записей'}
                              </span>
                            )}
                          </span>
                        </div>
                      )}
                      {db.tables && db.tables.length > 0 && (
                        <div className="flex items-center gap-1.5 text-xs">
                          <Table className="h-3.5 w-3.5" />
                          <span>
                            {db.tables.length} {db.tables.length === 1 ? 'таблица' : db.tables.length < 5 ? 'таблицы' : 'таблиц'}
                            {db.tables.some(t => t.row_count !== undefined) && (
                              <span className="ml-1">
                                • {formatNumber(db.tables.reduce((sum, t) => sum + (t.row_count || 0), 0))} {db.tables.reduce((sum, t) => sum + (t.row_count || 0), 0) === 1 ? 'запись' : 'записей'}
                              </span>
                            )}
                          </span>
                        </div>
                      )}
                    </div>
                    </div>
                  </div>
                  <div className="flex items-center gap-2 ml-4">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleLinkDatabase(db)}
                      disabled={isLinking}
                    >
                      <Link2 className="h-4 w-4 mr-1" />
                      Привязать
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setSelectedDatabase(db)}
                    >
                      <Eye className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Список баз данных по проектам */}
      {Object.keys(groupedDatabases).length === 0 && unlinkedDatabases.length === 0 ? (
        <Card>
          <CardContent className="py-8 text-center text-muted-foreground">
            {searchQuery || statusFilter !== 'all' || configFilter !== 'all' ? (
              <div className="space-y-2">
                <p>Базы данных не найдены по заданным фильтрам</p>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setSearchQuery('')
                    setStatusFilter('all')
                    setConfigFilter('all')
                  }}
                >
                  <X className="h-4 w-4 mr-2" />
                  Очистить фильтры
                </Button>
              </div>
            ) : (
              'Базы данных не найдены'
            )}
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-4">
          {Object.values(groupedDatabases).map((group) => (
            <Card key={group.projectId}>
              <CardHeader>
                <CardTitle 
                  className="text-base flex items-center justify-between cursor-pointer"
                  onClick={() => toggleProject(group.projectId)}
                >
                  <div className="flex items-center gap-2">
                    {expandedProjects.has(group.projectId) ? (
                      <ChevronDown className="h-4 w-4" />
                    ) : (
                      <ChevronRight className="h-4 w-4" />
                    )}
                    <span>{group.projectName}</span>
                    <Badge variant="outline">{group.databases.length}</Badge>
                  </div>
                  {expandedProjects.has(group.projectId) && group.databases.length > 0 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        const allSelected = group.databases.every(db => selectedDatabases.has(db.id))
                        if (allSelected) {
                          const newSelected = new Set(selectedDatabases)
                          group.databases.forEach(db => newSelected.delete(db.id))
                          setSelectedDatabases(newSelected)
                        } else {
                          const newSelected = new Set(selectedDatabases)
                          group.databases.forEach(db => newSelected.add(db.id))
                          setSelectedDatabases(newSelected)
                        }
                      }}
                      className="text-xs"
                    >
                      {group.databases.every(db => selectedDatabases.has(db.id)) ? (
                        <>
                          <CheckSquare className="h-3.5 w-3.5 mr-1" />
                          Снять все
                        </>
                      ) : (
                        <>
                          <Square className="h-3.5 w-3.5 mr-1" />
                          Выбрать все
                        </>
                      )}
                    </Button>
                  )}
                </CardTitle>
              </CardHeader>
              {expandedProjects.has(group.projectId) && (
                <CardContent>
                  <div className="space-y-2">
                    {group.databases.map((db, index) => (
                      <div
                        key={`db-tab-${db.id}-${db.path || db.name}-${index}`}
                        className={`flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 ${
                          selectedDatabases.has(db.id) ? 'ring-2 ring-primary' : ''
                        }`}
                      >
                        <div className="flex items-center gap-3 flex-1">
                          <Checkbox
                            checked={selectedDatabases.has(db.id)}
                            onCheckedChange={() => toggleDatabaseSelection(db.id)}
                          />
                          <div className="flex-1">
                          <div className="font-medium flex items-center gap-2">
                            <Database className="h-4 w-4" />
                            {db.name}
                          </div>
                          <div className="text-sm text-muted-foreground mt-1 space-y-1">
                            <div>Путь: {db.path}</div>
                            {db.size && <div>Размер: {formatFileSize(db.size)}</div>}
                            {(db.display_name || db.config_name) && (
                              <div className="flex items-center gap-2">
                                <span className="font-medium text-foreground flex items-center gap-1">
                                  <Settings className="h-3.5 w-3.5" />
                                  Конфигурация:
                                </span>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Badge 
                                      variant={getConfigBadgeVariant(db.config_name)} 
                                      className="text-xs flex items-center gap-1"
                                    >
                                      <Settings className="h-3 w-3" />
                                      {db.display_name || db.config_name}
                                    </Badge>
                                  </TooltipTrigger>
                                  <TooltipContent>
                                    <p>Конфигурация 1С: {db.display_name || db.config_name}</p>
                                    {db.config_name !== db.display_name && (
                                      <p className="text-xs text-muted-foreground mt-1">
                                        Техническое имя: {db.config_name}
                                      </p>
                                    )}
                                  </TooltipContent>
                                </Tooltip>
                              </div>
                            )}
                            <div>Создано: {formatDate(db.created_at)}</div>
                            {db.stats && (db.stats.total_uploads !== undefined || db.stats.uploads_count !== undefined) && (
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <div className="flex items-center gap-1.5 hover:text-foreground transition-colors cursor-help">
                                    <Layers className="h-3.5 w-3.5" />
                                    <span>{formatNumber(db.stats.total_uploads ?? db.stats.uploads_count ?? 0)} выгрузок</span>
                                  </div>
                                </TooltipTrigger>
                                <TooltipContent>
                                  <div className="space-y-1">
                                    <p>Выгрузок: {formatNumber(db.stats.total_uploads ?? db.stats.uploads_count ?? 0)}</p>
                                    {db.stats.total_catalogs !== undefined && (
                                      <p>Справочников: {formatNumber(db.stats.total_catalogs ?? db.stats.catalogs_count ?? 0)}</p>
                                    )}
                                    {db.stats.total_items !== undefined && (
                                      <p>Записей: {formatNumber(db.stats.total_items ?? db.stats.items_count ?? 0)}</p>
                                    )}
                                    {db.stats.last_upload_date && (
                                      <p>Последняя выгрузка: {formatDate(new Date(db.stats.last_upload_date))}</p>
                                    )}
                                    {db.stats.avg_items_per_upload !== undefined && (
                                      <p>Среднее записей/выгрузка: {formatNumber(Math.round(db.stats.avg_items_per_upload))}</p>
                                    )}
                                  </div>
                                </TooltipContent>
                              </Tooltip>
                            )}
                            {db.statistics && (
                              <div className="flex items-center gap-1.5 text-xs">
                                <Table className="h-3.5 w-3.5" />
                                <span>
                                  {db.statistics.total_tables} {db.statistics.total_tables === 1 ? 'таблица' : db.statistics.total_tables < 5 ? 'таблицы' : 'таблиц'}
                                  {db.statistics.total_rows > 0 && (
                                    <span className="ml-1">
                                      • {formatNumber(db.statistics.total_rows)} {db.statistics.total_rows === 1 ? 'запись' : db.statistics.total_rows < 5 ? 'записи' : 'записей'}
                                    </span>
                                  )}
                                </span>
                              </div>
                            )}
                            {db.tables && db.tables.length > 0 && (
                              <div className="flex items-center gap-1.5 text-xs">
                                <Table className="h-3.5 w-3.5" />
                                <span>
                                  {db.tables.length} {db.tables.length === 1 ? 'таблица' : db.tables.length < 5 ? 'таблицы' : 'таблиц'}
                                  {db.tables.some(t => t.row_count !== undefined) && (
                                    <span className="ml-1">
                                      • {formatNumber(db.tables.reduce((sum, t) => sum + (t.row_count || 0), 0))} {db.tables.reduce((sum, t) => sum + (t.row_count || 0), 0) === 1 ? 'запись' : 'записей'}
                                    </span>
                                  )}
                                </span>
                              </div>
                            )}
                          </div>
                          </div>
                        </div>
                        <div className="flex items-center gap-2">
                          <Badge variant={db.status === 'active' ? 'default' : 'secondary'}>
                            {db.status}
                          </Badge>
                          {(!db.project_id || db.project_id === 0) ? (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleLinkDatabase(db)}
                              disabled={isLinking}
                            >
                              <Link2 className="h-4 w-4 mr-1" />
                              Привязать
                            </Button>
                          ) : (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleUnlinkDatabase(db)}
                              disabled={isUnlinking}
                              className="text-orange-600 hover:text-orange-700 dark:text-orange-400"
                            >
                              <Unlink className="h-4 w-4 mr-1" />
                              Отвязать
                            </Button>
                          )}
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setSelectedDatabase(db)}
                          >
                            <Eye className="h-4 w-4 mr-1" />
                            Детали
                          </Button>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              )}
            </Card>
          ))}
        </div>
      )}

      {selectedDatabase && (
        <DatabaseDetailDialog
          database={selectedDatabase}
          clientId={clientId}
          open={!!selectedDatabase}
          onOpenChange={(open) => !open && setSelectedDatabase(null)}
        />
      )}

      {/* Модальное окно для привязки базы данных */}
      <Dialog open={linkDialogOpen} onOpenChange={setLinkDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Привязать базу данных к проекту</DialogTitle>
            <DialogDescription>
              Выберите проект для базы данных: {databaseToLink?.name}
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Label htmlFor="project-select">Проект</Label>
            <Select
              value={selectedLinkProjectId?.toString() || undefined}
              onValueChange={(value) => setSelectedLinkProjectId(value ? parseInt(value) : null)}
            >
              <SelectTrigger id="project-select" className="mt-2">
                <SelectValue placeholder="Выберите проект" />
              </SelectTrigger>
              <SelectContent>
                {projects.map((project) => (
                  <SelectItem key={project.id} value={project.id.toString()}>
                    {project.name} ({project.project_type})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setLinkDialogOpen(false)
                setDatabaseToLink(null)
                setSelectedLinkProjectId(null)
              }}
            >
              Отмена
            </Button>
            <Button
              onClick={handleConfirmLink}
              disabled={!selectedLinkProjectId || isLinking}
            >
              {isLinking ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Привязка...
                </>
              ) : (
                'Привязать'
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Модальное окно для массовой привязки */}
      <Dialog open={bulkLinkDialogOpen} onOpenChange={setBulkLinkDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Массовая привязка баз данных</DialogTitle>
            <DialogDescription>
              Выберите проект для привязки {selectedDatabases.size} выбранных баз данных
            </DialogDescription>
          </DialogHeader>
          <div className="py-4">
            <Label htmlFor="bulk-project-select">Проект</Label>
            <Select
              value={selectedLinkProjectId?.toString() || undefined}
              onValueChange={(value) => setSelectedLinkProjectId(value ? parseInt(value) : null)}
            >
              <SelectTrigger id="bulk-project-select" className="mt-2">
                <SelectValue placeholder="Выберите проект" />
              </SelectTrigger>
              <SelectContent>
                {projects.map((project) => (
                  <SelectItem key={project.id} value={project.id.toString()}>
                    {project.name} ({project.project_type})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setBulkLinkDialogOpen(false)
                setSelectedLinkProjectId(null)
              }}
              disabled={isBulkLinking}
            >
              Отмена
            </Button>
            <Button
              onClick={handleConfirmBulkLink}
              disabled={!selectedLinkProjectId || isBulkLinking}
            >
              {isBulkLinking ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Привязка...
                </>
              ) : (
                `Привязать ${selectedDatabases.size} баз`
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Модальное окно для отвязки базы данных */}
      <Dialog open={unlinkDialogOpen} onOpenChange={setUnlinkDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Отвязать базу данных от проекта</DialogTitle>
            <DialogDescription>
              Вы уверены, что хотите отвязать базу данных "{databaseToUnlink?.name}" от проекта?
              <br />
              <span className="text-sm text-muted-foreground mt-2 block">
                База данных останется в системе, но не будет привязана к проекту.
              </span>
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setUnlinkDialogOpen(false)
                setDatabaseToUnlink(null)
              }}
              disabled={isUnlinking}
            >
              Отмена
            </Button>
            <Button
              onClick={handleConfirmUnlink}
              disabled={isUnlinking}
              variant="destructive"
            >
              {isUnlinking ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Отвязка...
                </>
              ) : (
                <>
                  <Unlink className="h-4 w-4 mr-2" />
                  Отвязать
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}

