'use client'

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Database, Table, FileText, Calendar, Layers, Upload, BarChart3 } from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { formatDateTime, formatNumber } from "@/lib/locale"

interface ProjectDatabase {
  id: number
  name: string
  path: string
  size?: number
  created_at: string
  status: string
  project_id: number
  project_name: string
}

interface DatabaseDetail {
  id: number
  name: string
  path: string
  size?: number
  created_at: string
  updated_at?: string
  status: string
  tables?: Array<{
    name: string
    row_count?: number
    size?: number
  }>
  statistics?: {
    total_tables: number
    total_rows: number
    total_size: number
  }
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
}

interface DatabaseDetailDialogProps {
  database: ProjectDatabase
  clientId?: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DatabaseDetailDialog({ database, clientId, open, onOpenChange }: DatabaseDetailDialogProps) {
  const [details, setDetails] = useState<DatabaseDetail | null>(null)
  const [isLoading, setIsLoading] = useState(false)

  useEffect(() => {
    if (open && database) {
      fetchDatabaseDetails()
    }
  }, [open, database])

  const fetchDatabaseDetails = async () => {
    setIsLoading(true)
    try {
      // Используем clientId из props, если он передан, иначе используем project_id как fallback
      const clientIdToUse = clientId || database.project_id?.toString()
      if (!clientIdToUse) {
        throw new Error('Client ID is required')
      }
      const response = await fetch(`/api/clients/${clientIdToUse}/projects/${database.project_id}/databases/${database.id}`)
      if (response.ok) {
        const data = await response.json()
        setDetails(data)
      } else {
        console.error('Failed to fetch database details:', response.statusText)
      }
    } catch (error) {
      console.error('Failed to fetch database details:', error)
    } finally {
      setIsLoading(false)
    }
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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Database className="h-5 w-5" />
            {database.name}
          </DialogTitle>
          <DialogDescription>
            Детальная информация о базе данных
          </DialogDescription>
        </DialogHeader>

        {isLoading ? (
          <LoadingState message="Загрузка деталей..." />
        ) : (
          <div className="space-y-4">
            {/* Основная информация */}
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Основная информация</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Название:</span>
                  <span className="text-sm font-medium">{database.name}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Путь:</span>
                  <span className="text-sm font-medium break-all">{database.path}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Размер:</span>
                  <span className="text-sm font-medium">
                    {formatFileSize(details?.size || database.size)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Статус:</span>
                  <Badge variant={database.status === 'active' ? 'default' : 'secondary'}>
                    {database.status}
                  </Badge>
                </div>
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Создано:</span>
                  <span className="text-sm font-medium">
                    {formatDateTime(database.created_at)}
                  </span>
                </div>
                {details?.updated_at && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">Обновлено:</span>
                    <span className="text-sm font-medium">
                      {formatDateTime(details.updated_at)}
                    </span>
                  </div>
                )}
              </CardContent>
            </Card>

            {/* Статистика базы данных */}
            {details?.statistics && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Статистика базы данных</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <div className="text-sm text-muted-foreground">Таблиц</div>
                      <div className="text-2xl font-bold">{details.statistics.total_tables}</div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Строк</div>
                      <div className="text-2xl font-bold">
                        {details.statistics.total_rows.toLocaleString('ru-RU')}
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Размер</div>
                      <div className="text-2xl font-bold">
                        {formatFileSize(details.statistics.total_size)}
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Статистика из выгрузок */}
            {details?.stats && (details.stats.total_uploads !== undefined || details.stats.uploads_count !== undefined) && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base flex items-center gap-2">
                    <Upload className="h-4 w-4" />
                    Статистика выгрузок
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-3 gap-4">
                    <div>
                      <div className="text-sm text-muted-foreground flex items-center gap-1">
                        <Layers className="h-3.5 w-3.5" />
                        Выгрузок
                      </div>
                      <div className="text-2xl font-bold">
                        {formatNumber(details.stats.total_uploads ?? details.stats.uploads_count ?? 0)}
                      </div>
                    </div>
                    {details.stats.total_catalogs !== undefined && (
                      <div>
                        <div className="text-sm text-muted-foreground flex items-center gap-1">
                          <FileText className="h-3.5 w-3.5" />
                          Справочников
                        </div>
                        <div className="text-2xl font-bold">
                          {formatNumber(details.stats.total_catalogs ?? details.stats.catalogs_count ?? 0)}
                        </div>
                      </div>
                    )}
                    {details.stats.total_items !== undefined && (
                      <div>
                        <div className="text-sm text-muted-foreground flex items-center gap-1">
                          <Table className="h-3.5 w-3.5" />
                          Записей
                        </div>
                        <div className="text-2xl font-bold">
                          {formatNumber(details.stats.total_items ?? details.stats.items_count ?? 0)}
                        </div>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            )}

            {/* Таблицы - показываем всегда, если есть данные */}
            {details?.tables && details.tables.length > 0 && (
              <Card>
                <CardHeader>
                  <CardTitle className="text-base flex items-center gap-2">
                    <Table className="h-4 w-4" />
                    Таблицы ({details.tables.length})
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {details.tables.map((table, index) => (
                      <div
                        key={index}
                        className="flex items-center justify-between p-2 border rounded hover:bg-muted/50 transition-colors"
                      >
                        <div className="flex items-center gap-2">
                          <FileText className="h-4 w-4 text-muted-foreground" />
                          <span className="font-medium">{table.name}</span>
                        </div>
                        <div className="flex items-center gap-4 text-sm text-muted-foreground">
                          {table.row_count !== undefined && (
                            <span className="font-medium">
                              {table.row_count.toLocaleString('ru-RU')} {table.row_count === 1 ? 'запись' : table.row_count < 5 ? 'записи' : 'записей'}
                            </span>
                          )}
                          {table.size !== undefined && (
                            <span>Размер: {formatFileSize(table.size)}</span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

