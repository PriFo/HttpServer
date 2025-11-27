'use client'

import { Suspense } from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { DatabaseHealthChecker } from '@/components/diagnostics/database-health-checker'
import { UploadRecordsMonitor } from '@/components/diagnostics/upload-records-monitor'
import { DataExtractionAnalyzer } from '@/components/diagnostics/data-extraction-analyzer'
import { NormalizationStatusChecker } from '@/components/diagnostics/normalization-status'
import { DiagnosticsSummary } from '@/components/diagnostics/diagnostics-summary'
import { Wrench, Database, Upload, RefreshCw, CheckCircle2, AlertTriangle, Download, FileText } from 'lucide-react'
import { toast } from 'sonner'
import { useState, useEffect } from 'react'

interface DiagnosticsPageProps {
  params: {
    clientId: string
    projectId: string
  }
}

function DiagnosticsPageContent({ params }: DiagnosticsPageProps) {
  const clientId = parseInt(params.clientId)
  const projectId = parseInt(params.projectId)
  const [summary, setSummary] = useState<{
    databases: { total: number; healthy: number; issues: number }
    uploads: { total: number; valid: number; missing: number; invalid: number }
    extraction: { total: number; extracted: number; notExtracted: number }
    normalization: { hasData: boolean; recordsCount: number; sessionsCount: number }
  } | null>(null)

  const runAllDiagnostics = async () => {
    toast.info('Запуск полной диагностики...', {
      description: 'Проверка всех компонентов цепочки данных',
    })

    // Запускаем все проверки последовательно
    try {
      // 1. Проверка баз данных
      const dbResponse = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/databases`)
      if (!dbResponse.ok) throw new Error('Ошибка проверки баз данных')
      const dbData = await dbResponse.json()

      // 2. Проверка upload записей
      const uploadResponse = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/uploads`)
      if (!uploadResponse.ok) throw new Error('Ошибка проверки upload записей')
      const uploadData = await uploadResponse.json()

      // 3. Проверка извлечения
      const extractionResponse = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/extraction`)
      if (!extractionResponse.ok) throw new Error('Ошибка проверки извлечения данных')
      const extractionData = await extractionResponse.json()

      // 4. Проверка нормализации
      const normalizationResponse = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/normalization`)
      if (!normalizationResponse.ok) throw new Error('Ошибка проверки нормализации')
      const normalizationData = await normalizationResponse.json()

      // Обновляем сводку
      setSummary({
        databases: {
          total: dbData.length,
          healthy: dbData.filter((d: any) => d.exists && d.issues.length === 0).length,
          issues: dbData.filter((d: any) => !d.exists || d.issues.length > 0).length,
        },
        uploads: {
          total: uploadData.length,
          valid: uploadData.filter((u: any) => u.status === 'valid').length,
          missing: uploadData.filter((u: any) => u.status === 'missing').length,
          invalid: uploadData.filter((u: any) => u.status === 'invalid').length,
        },
        extraction: {
          total: extractionData.length,
          extracted: extractionData.filter((e: any) => e.extraction_method === 'auto' && (e.catalog_items_count > 0 || e.nomenclature_items_count > 0)).length,
          notExtracted: extractionData.filter((e: any) => e.extraction_method === 'none' || (e.catalog_items_count === 0 && e.nomenclature_items_count === 0)).length,
        },
        normalization: {
          hasData: normalizationData.normalized_records_count > 0,
          recordsCount: normalizationData.normalized_records_count || 0,
          sessionsCount: normalizationData.normalization_sessions?.length || 0,
        },
      })

      toast.success('Диагностика завершена', {
        description: 'Все проверки выполнены успешно',
      })
    } catch (err) {
      toast.error('Ошибка диагностики', {
        description: err instanceof Error ? err.message : 'Не удалось выполнить диагностику',
      })
    }
  }

  const fixAllIssues = async () => {
    toast.info('Исправление проблем...', {
      description: 'Создание недостающих upload записей',
    })

    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/uploads/fix`, {
        method: 'POST',
      })
      if (!response.ok) {
        throw new Error('Не удалось исправить проблемы')
      }
      const data = await response.json()
      toast.success('Проблемы исправлены', {
        description: `Создано ${data.fixed_count} upload записей`,
      })
      // Обновляем сводку после исправления
      await runAllDiagnostics()
    } catch (err) {
      toast.error('Ошибка исправления', {
        description: err instanceof Error ? err.message : 'Не удалось исправить проблемы',
      })
    }
  }

  const exportDiagnosticsReport = async () => {
    if (!summary) {
      toast.warning('Нет данных для экспорта', {
        description: 'Сначала запустите диагностику',
      })
      return
    }

    try {
      // Собираем все данные диагностики
      const [dbData, uploadData, extractionData, normalizationData] = await Promise.all([
        fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/databases`).then(r => r.json()),
        fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/uploads`).then(r => r.json()),
        fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/extraction`).then(r => r.json()),
        fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/normalization`).then(r => r.json()),
      ])

      const report = {
        timestamp: new Date().toISOString(),
        project_id: projectId,
        client_id: clientId,
        summary: summary,
        databases: dbData,
        uploads: uploadData,
        extraction: extractionData,
        normalization: normalizationData,
      }

      const jsonData = JSON.stringify(report, null, 2)
      const blob = new Blob([jsonData], { type: 'application/json' })
      const filename = `diagnostics_report_${projectId}_${new Date().toISOString().split('T')[0]}.json`

      if ('showSaveFilePicker' in window) {
        try {
          const fileHandle = await (window as any).showSaveFilePicker({
            suggestedName: filename,
            types: [{
              description: 'JSON файлы',
              accept: { 'application/json': ['.json'] },
            }],
          })
          const writable = await fileHandle.createWritable()
          await writable.write(blob)
          await writable.close()
          toast.success('Отчет экспортирован', {
            description: `Файл сохранен: ${filename}`,
          })
        } catch (saveErr: any) {
          if (saveErr.name !== 'AbortError') {
            // Fallback на стандартное скачивание
            const url = window.URL.createObjectURL(blob)
            const link = document.createElement('a')
            link.href = url
            link.download = filename
            document.body.appendChild(link)
            link.click()
            document.body.removeChild(link)
            window.URL.revokeObjectURL(url)
            toast.success('Отчет экспортирован', {
              description: `Файл скачан: ${filename}`,
            })
          }
        }
      } else {
        // Fallback для браузеров без File System Access API
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = filename
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
        toast.success('Отчет экспортирован', {
          description: `Файл скачан: ${filename}`,
        })
      }
    } catch (err) {
      toast.error('Ошибка экспорта', {
        description: err instanceof Error ? err.message : 'Не удалось экспортировать отчет',
      })
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold flex items-center gap-2">
            <Wrench className="h-8 w-8" />
            Диагностика цепочки данных
          </h1>
          <p className="text-muted-foreground mt-2">
            Проверка всех этапов извлечения данных: от исходных баз до отображения на фронтенде
          </p>
        </div>
        <div className="flex gap-2">
          <Button onClick={runAllDiagnostics} variant="outline">
            <CheckCircle2 className="h-4 w-4 mr-2" />
            Запустить все проверки
          </Button>
          <Button onClick={fixAllIssues} variant="default">
            <AlertTriangle className="h-4 w-4 mr-2" />
            Исправить проблемы
          </Button>
          {summary && (
            <Button onClick={exportDiagnosticsReport} variant="outline">
              <Download className="h-4 w-4 mr-2" />
              Экспорт отчета
            </Button>
          )}
        </div>
      </div>

      {summary && (
        <DiagnosticsSummary
          databases={summary.databases}
          uploads={summary.uploads}
          extraction={summary.extraction}
          normalization={summary.normalization}
        />
      )}

      <Card className="bg-blue-50 dark:bg-blue-950/20 border-blue-200 dark:border-blue-800">
        <CardHeader>
          <CardTitle className="text-lg flex items-center gap-2">
            <AlertTriangle className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            Информация о диагностике
          </CardTitle>
          <CardDescription>
            Диагностика проверяет каждый этап цепочки данных:
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ol className="list-decimal list-inside space-y-2 text-sm">
            <li><strong>Базы данных:</strong> Проверка наличия файлов, таблиц и данных</li>
            <li><strong>Upload записи:</strong> Проверка наличия и валидности записей в таблице uploads</li>
            <li><strong>Извлечение данных:</strong> Проверка наличия данных в catalog_items и nomenclature_items</li>
            <li><strong>Нормализация:</strong> Проверка нормализованных данных и сессий нормализации</li>
          </ol>
        </CardContent>
      </Card>

      <Tabs defaultValue="databases" className="space-y-4">
        <TabsList className="grid w-full grid-cols-4">
          <TabsTrigger value="databases">
            <Database className="h-4 w-4 mr-2" />
            Базы данных
          </TabsTrigger>
          <TabsTrigger value="uploads">
            <Upload className="h-4 w-4 mr-2" />
            Upload записи
          </TabsTrigger>
          <TabsTrigger value="extraction">
            <RefreshCw className="h-4 w-4 mr-2" />
            Извлечение
          </TabsTrigger>
          <TabsTrigger value="normalization">
            <CheckCircle2 className="h-4 w-4 mr-2" />
            Нормализация
          </TabsTrigger>
        </TabsList>

        <TabsContent value="databases" className="space-y-4">
          <DatabaseHealthChecker projectId={projectId} clientId={clientId} />
        </TabsContent>

        <TabsContent value="uploads" className="space-y-4">
          <UploadRecordsMonitor projectId={projectId} clientId={clientId} />
        </TabsContent>

        <TabsContent value="extraction" className="space-y-4">
          <DataExtractionAnalyzer projectId={projectId} clientId={clientId} />
        </TabsContent>

        <TabsContent value="normalization" className="space-y-4">
          <NormalizationStatusChecker projectId={projectId} clientId={clientId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default function DiagnosticsPage({ params }: DiagnosticsPageProps) {
  return (
    <Suspense fallback={<div>Загрузка...</div>}>
      <DiagnosticsPageContent params={params} />
    </Suspense>
  )
}

