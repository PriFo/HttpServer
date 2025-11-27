'use client'

import React, { useState, useEffect, useMemo, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Download, FileSpreadsheet, FileCode, FileJson, History, Trash2 } from 'lucide-react'
import { toast } from 'sonner'
import { ExportDialog } from './export-dialog'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import type { ExportHistory } from '@/types/normalization'
import { handleError } from '@/lib/error-handler'

interface ExportManagerProps {
  clientId: string
  projectId: string
}

async function fetchExportHistory(
  clientId: string,
  projectId: string,
  signal?: AbortSignal
): Promise<{ history: ExportHistory[] }> {
  const response = await fetch(
    `/api/clients/${clientId}/projects/${projectId}/normalization/export/history`,
    { cache: 'no-store', signal }
  )

  if (!response.ok) {
    if (response.status === 404) {
      return { history: [] }
    }
    throw new Error(`Failed to fetch export history: ${response.status}`)
  }

  return response.json()
}

export const ExportManager: React.FC<ExportManagerProps> = ({
  clientId,
  projectId,
}) => {
  const [exportHistory, setExportHistory] = useState<ExportHistory[]>([])
  const [isExporting, setIsExporting] = useState(false)
  const [exportProgress, setExportProgress] = useState(0)

  const { data: historyData, loading: historyLoading, error: historyError, refetch: refetchHistory } = useProjectState(
    fetchExportHistory,
    clientId,
    projectId,
    [],
    {
      refetchInterval: 60000,
      enabled: !!clientId && !!projectId,
    }
  )

  useEffect(() => {
    if (historyData?.history) {
      setExportHistory(historyData.history)
    }
  }, [historyData])

  useEffect(() => {
    if (!isExporting) {
      return
    }
    setExportProgress(10)
    const intervalId = setInterval(() => {
      setExportProgress(prev => {
        if (prev >= 90) {
          clearInterval(intervalId)
          return prev
        }
        return prev + 5
      })
    }, 300)
    return () => clearInterval(intervalId)
  }, [isExporting])

  const handleExportStart = useCallback(() => {
    setIsExporting(true)
    setExportProgress(10)
  }, [])

  const handleExportComplete = useCallback(async (success: boolean) => {
    setIsExporting(false)
    if (success) {
      setExportProgress(100)
      await refetchHistory()
    }
    setTimeout(() => setExportProgress(0), 800)
  }, [refetchHistory])

  const handleDeleteHistory = useCallback(async (id: string) => {
    try {
      const response = await fetch(
        `/api/clients/${clientId}/projects/${projectId}/normalization/export/history/${id}`,
        { method: 'DELETE' }
      )

      if (!response.ok) {
        throw new Error('Failed to delete history')
      }

      setExportHistory(prev => prev.filter(item => item.id !== id))
      toast.success('Запись удалена из истории')
    } catch (error) {
      handleError(error, {
        context: {
          component: 'ExportManager',
          action: 'deleteHistory',
          historyId: id,
          clientId,
          projectId,
        },
        fallbackMessage: 'Не удалось удалить запись истории',
      })
    }
  }, [clientId, projectId])

  const summary = useMemo(() => {
    if (exportHistory.length === 0) {
      return { totalExports: 0, completed: 0, failed: 0 }
    }
    const completed = exportHistory.filter(item => item.status === 'completed').length
    const failed = exportHistory.filter(item => item.status === 'failed').length
    return {
      totalExports: exportHistory.length,
      completed,
      failed,
    }
  }, [exportHistory])

  const getFormatIcon = (format: string) => {
    switch (format) {
      case 'excel':
        return <FileSpreadsheet className="h-4 w-4" />
      case 'csv':
        return <FileCode className="h-4 w-4" />
      case 'json':
        return <FileJson className="h-4 w-4" />
      default:
        return <Download className="h-4 w-4" />
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base flex items-center gap-2">
              <Download className="h-5 w-5" />
              Управление экспортом
            </CardTitle>
            <CardDescription>
              Экспорт данных и история экспортов
            </CardDescription>
          </div>
          <div className="flex items-center gap-3 text-xs text-muted-foreground">
            <span>Всего: {summary.totalExports}</span>
            <Badge variant="outline">Успехов: {summary.completed}</Badge>
            <Badge variant={summary.failed ? 'destructive' : 'secondary'}>Ошибок: {summary.failed}</Badge>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <ExportDialog
            clientId={clientId}
            projectId={projectId}
            dataType="groups"
            onExportStart={handleExportStart}
            onExportComplete={handleExportComplete}
            trigger={
              <Button variant="outline" className="w-full" disabled={isExporting}>
                <FileSpreadsheet className="h-4 w-4 mr-2" />
                Экспорт групп
              </Button>
            }
          />
          <ExportDialog
            clientId={clientId}
            projectId={projectId}
            dataType="nomenclature"
            onExportStart={handleExportStart}
            onExportComplete={handleExportComplete}
            trigger={
              <Button variant="outline" className="w-full" disabled={isExporting}>
                <FileCode className="h-4 w-4 mr-2" />
                Экспорт номенклатуры
              </Button>
            }
          />
          <ExportDialog
            clientId={clientId}
            projectId={projectId}
            dataType="counterparties"
            onExportStart={handleExportStart}
            onExportComplete={handleExportComplete}
            trigger={
              <Button variant="outline" className="w-full" disabled={isExporting}>
                <FileJson className="h-4 w-4 mr-2" />
                Экспорт контрагентов
              </Button>
            }
          />
        </div>

        {isExporting && (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span>Экспорт данных...</span>
              <span className="font-medium">{exportProgress}%</span>
            </div>
            <Progress value={exportProgress} />
          </div>
        )}

        {historyError && (
          <ErrorState
            title="Ошибка загрузки истории"
            message={historyError}
            action={{
              label: 'Повторить',
              onClick: refetchHistory,
            }}
          />
        )}

        {historyLoading && exportHistory.length === 0 && (
          <LoadingState message="Загрузка истории экспортов..." />
        )}

        {exportHistory.length > 0 && (
          <div className="pt-4 border-t">
            <div className="flex items-center justify-between mb-3">
              <h4 className="font-semibold text-sm flex items-center gap-2">
                <History className="h-4 w-4" />
                История экспортов
              </h4>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setExportHistory([])}
              >
                Очистить историю
              </Button>
            </div>
            <div className="space-y-2 max-h-[300px] overflow-y-auto">
              {exportHistory.map((exportItem) => (
                <div
                  key={exportItem.id}
                  className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50"
                >
                  <div className="flex items-center gap-3 flex-1">
                    {getFormatIcon(exportItem.format)}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <span className="font-medium text-sm">{exportItem.dataType}</span>
                        <Badge variant="outline" className="text-xs">
                          {exportItem.format.toUpperCase()}
                        </Badge>
                        <Badge
                          variant={
                            exportItem.status === 'completed' ? 'default' :
                            exportItem.status === 'failed' ? 'destructive' : 'secondary'
                          }
                          className="text-xs"
                        >
                          {exportItem.status === 'completed' ? 'Завершен' :
                           exportItem.status === 'failed' ? 'Ошибка' : 'В процессе'}
                        </Badge>
                      </div>
                      <p className="text-xs text-muted-foreground mt-1">
                        {new Date(exportItem.timestamp).toLocaleString('ru-RU')}
                        {exportItem.recordCount > 0 && ` • ${exportItem.recordCount} записей`}
                      </p>
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => handleDeleteHistory(exportItem.id)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

