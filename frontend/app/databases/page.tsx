'use client'

import { useState, useEffect } from 'react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Database, HardDrive, Calendar, CheckCircle2, AlertCircle, RefreshCw, BarChart3 } from 'lucide-react'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { StatCard } from '@/components/common/stat-card'
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
import { useRouter } from 'next/navigation'
import { DatabaseTypeBadge } from '@/components/database-type-badge'
import { DatabaseAnalyticsDialog } from '@/components/database-analytics-dialog'

interface DatabaseInfo {
  name: string
  path: string
  size: number
  modified_at: string
  is_current?: boolean
  type?: string
  table_count?: number
  total_rows?: number
  stats?: {
    total_uploads?: number
    uploads_count?: number
    total_catalogs?: number
    catalogs_count?: number
    total_items?: number
    items_count?: number
  }
}

interface CurrentDatabaseInfo extends DatabaseInfo {
  status: string
}

export default function DatabasesPage() {
  const router = useRouter()
  const [currentDB, setCurrentDB] = useState<CurrentDatabaseInfo | null>(null)
  const [databases, setDatabases] = useState<DatabaseInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [switching, setSwitching] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selectedDB, setSelectedDB] = useState<string | null>(null)
  const [showConfirmDialog, setShowConfirmDialog] = useState(false)
  const [showAnalyticsDialog, setShowAnalyticsDialog] = useState(false)
  const [analyticsDB, setAnalyticsDB] = useState<{ name: string; path: string } | null>(null)

  const fetchData = async () => {
    setLoading(true)
    setError(null)

    try {
      // Fetch current database info
      const infoResponse = await fetch('/api/database/info')
      if (!infoResponse.ok) throw new Error('Failed to fetch current database info')
      const infoData = await infoResponse.json()
      setCurrentDB(infoData)

      // Fetch list of databases
      const listResponse = await fetch('/api/databases/list')
      if (!listResponse.ok) throw new Error('Failed to fetch databases list')
      const listData = await listResponse.json()
      setDatabases(listData.databases || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error occurred')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleSwitchDatabase = async () => {
    if (!selectedDB) return

    setSwitching(true)
    setError(null)

    try {
      const response = await fetch('/api/database/switch', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({ path: selectedDB })
      })

      if (!response.ok) {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to switch database')
      }

      // Refresh data after successful switch
      await fetchData()
      setShowConfirmDialog(false)
      setSelectedDB(null)

      // Redirect to home page
      router.push('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to switch database')
    } finally {
      setSwitching(false)
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  const formatDate = (dateString: string) => {
    if (!dateString) return 'Неизвестно'
    const date = new Date(dateString)
    return date.toLocaleDateString('ru-RU', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  if (loading) {
    return (
      <div className="container mx-auto px-4 py-8">
        <LoadingState message="Загрузка информации о базах данных..." size="lg" fullScreen />
      </div>
    )
  }

  return (
    <div className="container mx-auto px-4 py-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold mb-2">Управление базами данных</h1>
        <p className="text-muted-foreground">
          Просмотр и переключение между базами данных 1С
        </p>
      </div>

      {error && (
        <ErrorState
          message={error}
          action={{
            label: 'Повторить',
            onClick: fetchData,
          }}
          variant="destructive"
          className="mb-6"
        />
      )}

      {/* Current Database Info */}
      {currentDB && (
        <Card className="mb-8">
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Database className="h-5 w-5" />
                  Текущая база данных
                </CardTitle>
                <CardDescription>Активная база данных для работы</CardDescription>
              </div>
              <Badge variant="outline" className="gap-2">
                <CheckCircle2 className="h-3 w-3 text-green-500" />
                Подключено
              </Badge>
            </div>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div>
                <div className="space-y-4">
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">Имя файла</p>
                    <p className="text-lg font-semibold">{currentDB.name}</p>
                  </div>
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-1">Путь</p>
                    <p className="text-sm font-mono bg-muted px-2 py-1 rounded">
                      {currentDB.path}
                    </p>
                  </div>
                  <div className="flex gap-6">
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-1">Размер</p>
                      <p className="text-sm">{formatFileSize(currentDB.size)}</p>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-muted-foreground mb-1">Изменено</p>
                      <p className="text-sm">{formatDate(currentDB.modified_at)}</p>
                    </div>
                  </div>
                </div>
              </div>

              {currentDB.stats && (
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-3">Статистика</p>
                  <div className="grid grid-cols-1 gap-3">
                    <StatCard
                      title="Выгрузок"
                      value={currentDB.stats.total_uploads ?? currentDB.stats.uploads_count ?? 0}
                      variant="default"
                      className="p-3"
                    />
                    <StatCard
                      title="Справочников"
                      value={currentDB.stats.total_catalogs ?? currentDB.stats.catalogs_count ?? 0}
                      variant="default"
                      className="p-3"
                    />
                    <StatCard
                      title="Записей"
                      value={currentDB.stats.total_items ?? currentDB.stats.items_count ?? 0}
                      variant="primary"
                      className="p-3"
                      formatValue={(val) => val.toLocaleString('ru-RU')}
                    />
                  </div>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Available Databases */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <HardDrive className="h-5 w-5" />
            Доступные базы данных
          </CardTitle>
          <CardDescription>
            Список всех баз данных в текущей директории ({databases.length})
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {databases.map((db) => {
              const isCurrent = db.path === currentDB?.path

              return (
                <div
                  key={db.path}
                  className={`flex items-center justify-between p-4 border rounded-lg ${
                    isCurrent ? 'bg-primary/5 border-primary' : 'bg-card'
                  }`}
                >
                  <div className="flex items-center gap-4 flex-1">
                    <Database className={`h-5 w-5 ${isCurrent ? 'text-primary' : 'text-muted-foreground'}`} />
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <p className="font-medium">{db.name}</p>
                        {isCurrent && (
                          <Badge variant="default" className="text-xs">
                            Текущая
                          </Badge>
                        )}
                        {db.type && <DatabaseTypeBadge type={db.type} />}
                      </div>
                      <div className="flex items-center gap-4 text-sm text-muted-foreground">
                        <span className="flex items-center gap-1">
                          <HardDrive className="h-3 w-3" />
                          {formatFileSize(db.size)}
                        </span>
                        <span className="flex items-center gap-1">
                          <Calendar className="h-3 w-3" />
                          {formatDate(db.modified_at)}
                        </span>
                        {db.table_count !== undefined && (
                          <span className="flex items-center gap-1">
                            <Database className="h-3 w-3" />
                            {db.table_count} таблиц
                          </span>
                        )}
                        {db.total_rows !== undefined && (
                          <span>
                            {db.total_rows.toLocaleString('ru-RU')} записей
                          </span>
                        )}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setAnalyticsDB({ name: db.name, path: db.path })
                        setShowAnalyticsDialog(true)
                      }}
                    >
                      <BarChart3 className="h-4 w-4 mr-2" />
                      Аналитика
                    </Button>
                    {!isCurrent && (
                      <Button
                        variant="outline"
                        onClick={() => {
                          setSelectedDB(db.path)
                          setShowConfirmDialog(true)
                        }}
                        disabled={switching}
                      >
                        Переключить
                      </Button>
                    )}
                  </div>
                </div>
              )
            })}

            {databases.length === 0 && (
              <EmptyState
                icon={Database}
                title="Не найдено доступных баз данных"
                description="В текущей директории нет доступных баз данных"
              />
            )}
          </div>
        </CardContent>
      </Card>

      {/* Confirmation Dialog */}
      <AlertDialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Подтвердите переключение базы данных</AlertDialogTitle>
            <AlertDialogDescription>
              Вы уверены, что хотите переключиться на базу данных{' '}
              <span className="font-semibold">{selectedDB}</span>?
              <br />
              <br />
              Это действие закроет текущее подключение и переключит все операции на выбранную базу данных.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={switching}>Отмена</AlertDialogCancel>
            <AlertDialogAction onClick={handleSwitchDatabase} disabled={switching}>
              {switching ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Переключение...
                </>
              ) : (
                'Переключить'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      {/* Analytics Dialog */}
      {analyticsDB && (
        <DatabaseAnalyticsDialog
          open={showAnalyticsDialog}
          onOpenChange={setShowAnalyticsDialog}
          databaseName={analyticsDB.name}
          databasePath={analyticsDB.path}
        />
      )}
    </div>
  )
}
