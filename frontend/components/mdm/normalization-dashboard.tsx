'use client'

import React, { useState, useEffect } from 'react'
import '@/styles/normalization-enhanced.css'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { useProjectState, useProjectStateSimple } from '@/hooks/useProjectState'
import { fetchNormalizationStatusApi } from '@/lib/mdm/api'
import type { NormalizationStatus } from '@/types/normalization'
import { BarChart3, Package, Users, CheckCircle2, AlertTriangle, Loader2 } from 'lucide-react'
import { NormalizationResultsTable } from '@/components/processes/normalization-results-table'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { ProcessMonitoringWorkspace } from './process-monitoring-workspace'
import { PipelineVisualization } from './pipeline-visualization'
import { DataQualityWorkspace } from './data-quality-workspace'
import { IntelligentDeduplication } from './intelligent-deduplication'
import { BusinessRulesManager } from './business-rules-manager'
import { AdvancedAnalytics } from './advanced-analytics'
import { ExportDialog } from './export-dialog'
import { ChangeHistory } from './change-history'
import { GroupAnalysis } from './group-analysis'
import { ResultsFilters } from './results-filters'
import { QuickActions } from './quick-actions'
import { NotificationCenter } from './notification-center'
import { GlobalSearch } from './global-search'
import { NormalizationStatsSummary } from './normalization-stats-summary'
import { HelpPanel } from './help-panel'
import { KeyboardShortcuts } from './keyboard-shortcuts'
import { RealtimeUpdates } from './realtime-updates'
import { BatchOperations } from './batch-operations'
import { DataPreview } from './data-preview'
import { PerformanceMonitor } from './performance-monitor'
import { ExportManager } from './export-manager'
import { ActivityTimeline } from './activity-timeline'
import { SettingsPanel } from './settings-panel'
import { DataComparison } from './data-comparison'
import { BulkEdit } from './bulk-edit'
import { QuickStats } from './quick-stats'
import { NormalizationProvider } from '@/context/NormalizationContext'
import { ErrorBoundary } from '@/components/common/error-boundary'

interface NormalizationDashboardProps {
  clientId: string
  projectId: string
  projectType?: string | null
}

export const NormalizationDashboard: React.FC<NormalizationDashboardProps> = ({
  clientId,
  projectId,
  projectType,
}) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'nomenclature' | 'counterparties' | 'quality' | 'classification'>('overview')
  const [searchOpen, setSearchOpen] = useState(false)

  // Загрузка статуса нормализации с использованием хука
  // Используем отдельное состояние для отслеживания запущенного процесса
  const [isProcessRunning, setIsProcessRunning] = useState(false)
  
  const { data: normalizationStatus, loading: loadingStatus, refetch: refetchStatus } = useProjectState<NormalizationStatus>(
    (cid, pid, signal) => fetchNormalizationStatusApi(cid, pid, signal),
    clientId,
    projectId,
    [],
    {
      // Автоматическое обновление каждые 5 секунд, если процесс запущен
      refetchInterval: isProcessRunning ? 5000 : null,
      enabled: !!clientId && !!projectId,
    }
  )

  // Обновляем состояние запуска процесса
  useEffect(() => {
    setIsProcessRunning(normalizationStatus?.isRunning || false)
  }, [normalizationStatus?.isRunning])

  // Сброс активного таба и состояния при смене проекта
  useEffect(() => {
    setActiveTab('overview')
  }, [clientId, projectId])

  // Загрузка информации о проекте
  const projectState = useProjectStateSimple(clientId, projectId)

  // Обработчик обновления статуса после действий
  const handleStatusUpdate = React.useCallback(() => {
    refetchStatus()
  }, [refetchStatus])

  if (projectState.loading) {
    return <LoadingState message="Загрузка информации о проекте..." />
  }

  if (projectState.error) {
    return (
      <ErrorState
        title="Ошибка загрузки проекта"
        message={projectState.error}
        variant="destructive"
      />
    )
  }

  const project = projectState.data?.project

  const normalizationContextValue = React.useMemo(
    () => ({
      clientId,
      projectId,
      projectType,
      normalizationStatus,
      isProcessRunning,
      setIsProcessRunning,
      refetchStatus: handleStatusUpdate,
    }),
    [
      clientId,
      projectId,
      projectType,
      normalizationStatus,
      isProcessRunning,
      handleStatusUpdate,
    ]
  )

  return (
    <ErrorBoundary
      resetKeys={[clientId, projectId]}
      onError={(error, errorInfo) => {
        // Логирование уже происходит в ErrorBoundary
      }}
    >
      <NormalizationProvider value={normalizationContextValue}>
        <KeyboardShortcuts
        onSearch={() => setSearchOpen(true)}
        onExport={() => {
          // Открыть диалог экспорта
        }}
        onRefresh={handleStatusUpdate}
        onTabChange={(tab) => setActiveTab(tab as any)}
      />
      <div className="space-y-6">
        {/* Заголовок и контекст */}
        <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold">Нормализация НСИ</h2>
          <p className="text-muted-foreground">
            {project?.name ? `Проект: ${project.name}` : `Клиент: ${clientId}, Проект: ${projectId}`}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setSearchOpen(true)}
            className="gap-2"
          >
            <BarChart3 className="h-4 w-4" />
            Поиск
          </Button>
          <GlobalSearch
            clientId={clientId}
            projectId={projectId}
            open={searchOpen}
            onOpenChange={setSearchOpen}
          />
          <ExportDialog
            clientId={clientId}
            projectId={projectId}
            dataType="groups"
          />
          {normalizationStatus?.isRunning && (
            <Badge variant="default" className="animate-pulse">
              <Loader2 className="h-3 w-3 mr-2 animate-spin" />
              В процессе
            </Badge>
          )}
        </div>
      </div>

      {/* Навигационные табы */}
      <Tabs value={activeTab} onValueChange={(value) => setActiveTab(value as any)}>
        <TabsList className="grid w-full grid-cols-5">
          <TabsTrigger value="overview">
            <BarChart3 className="h-4 w-4 mr-2" />
            Обзор
          </TabsTrigger>
          <TabsTrigger value="nomenclature">
            <Package className="h-4 w-4 mr-2" />
            Номенклатура
          </TabsTrigger>
          <TabsTrigger value="counterparties">
            <Users className="h-4 w-4 mr-2" />
            Контрагенты
          </TabsTrigger>
          <TabsTrigger value="quality">
            <CheckCircle2 className="h-4 w-4 mr-2" />
            Качество
          </TabsTrigger>
          <TabsTrigger value="classification">
            <AlertTriangle className="h-4 w-4 mr-2" />
            Классификация
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-4">
          <OverviewTab
            clientId={clientId}
            projectId={projectId}
            projectType={projectType}
            normalizationStatus={normalizationStatus}
            onStatusUpdate={handleStatusUpdate}
          />
        </TabsContent>

        <TabsContent value="nomenclature" className="space-y-4">
          <NomenclatureWorkspace
            clientId={clientId}
            projectId={projectId}
            projectType={projectType}
          />
        </TabsContent>

        <TabsContent value="counterparties" className="space-y-4">
          <CounterpartiesWorkspace
            clientId={clientId}
            projectId={projectId}
            projectType={projectType}
          />
        </TabsContent>

        <TabsContent value="quality" className="space-y-4">
          <DataQualityWorkspace
            clientId={clientId}
            projectId={projectId}
          />
        </TabsContent>

        <TabsContent value="classification" className="space-y-4">
          <div className="space-y-4">
            <Card>
              <CardHeader>
                <CardTitle>Классификация данных</CardTitle>
                <CardDescription>
                  Управление классификацией нормализованных данных
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className="space-y-4">
                  <PipelineVisualization
                    clientId={clientId}
                    projectId={projectId}
                    activeProcess={normalizationStatus?.currentStep || null}
                  />
                  <IntelligentDeduplication
                    clientId={clientId}
                    projectId={projectId}
                  />
                </div>
              </CardContent>
            </Card>
          </div>
        </TabsContent>
      </Tabs>
      </div>
      </NormalizationProvider>
    </ErrorBoundary>
  )
}

// Компонент обзора
const OverviewTab: React.FC<{
  clientId: string
  projectId: string
  projectType?: string | null
  normalizationStatus: any
  onStatusUpdate?: () => void
}> = ({ clientId, projectId, projectType, normalizationStatus, onStatusUpdate }) => {
  const [filters, setFilters] = useState<any>({})
  const [selectedItems, setSelectedItems] = useState<string[]>([])
  const [previewData, setPreviewData] = useState<any>(null)
  const [comparisonItems, setComparisonItems] = useState<{ item1: any; item2: any }>({ item1: null, item2: null })

  return (
    <div className="space-y-4">
      {/* Быстрая статистика */}
      <QuickStats
        clientId={clientId}
        projectId={projectId}
      />

      {/* Сводная статистика */}
      <NormalizationStatsSummary
        clientId={clientId}
        projectId={projectId}
        isProcessRunning={normalizationStatus?.isRunning || false}
      />

      {/* Быстрые действия */}
      <QuickActions
        clientId={clientId}
        projectId={projectId}
        isRunning={normalizationStatus?.isRunning || false}
        onRefresh={onStatusUpdate}
      />

      {/* Мониторинг процесса */}
      <ProcessMonitoringWorkspace
        clientId={clientId}
        projectId={projectId}
        timeRange="realtime"
      />

      {/* Визуализация пайплайна */}
      <PipelineVisualization
        clientId={clientId}
        projectId={projectId}
        activeProcess={normalizationStatus?.currentStep || null}
      />

      {/* Фильтры результатов */}
      <ResultsFilters
        filters={filters}
        onFiltersChange={setFilters}
      />

      {/* Результаты нормализации */}
      <Card>
        <CardHeader>
          <CardTitle>Результаты нормализации</CardTitle>
          <CardDescription>
            Группы нормализованных данных
          </CardDescription>
        </CardHeader>
        <CardContent>
          <NormalizationResultsTable
            isRunning={normalizationStatus?.isRunning || false}
            project={`${clientId}:${projectId}`}
            projectType={projectType}
          />
        </CardContent>
      </Card>

      {/* Аналитика */}
      <AdvancedAnalytics
        clientId={clientId}
        projectId={projectId}
      />

      {/* История изменений */}
      <ChangeHistory
        clientId={clientId}
        projectId={projectId}
      />

      {/* Центр уведомлений и помощь */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <NotificationCenter
          clientId={clientId}
          projectId={projectId}
        />
        <HelpPanel section="overview" />
      </div>

      {/* Обновления в реальном времени */}
      {/* Обновления в реальном времени и производительность */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <RealtimeUpdates
          clientId={clientId}
          projectId={projectId}
          onUpdate={onStatusUpdate}
        />
        <PerformanceMonitor
          clientId={clientId}
          projectId={projectId}
        />
      </div>

      {/* Массовые операции и редактирование */}
      {selectedItems.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <BatchOperations
            clientId={clientId}
            projectId={projectId}
            selectedItems={selectedItems}
            onSelectionChange={setSelectedItems}
          />
          <BulkEdit
            clientId={clientId}
            projectId={projectId}
            selectedItems={selectedItems}
            onSave={() => {
              // Обновление данных после сохранения
              setSelectedItems([])
            }}
          />
        </div>
      )}

      {/* Сравнение данных */}
      {(comparisonItems.item1 || comparisonItems.item2) && (
        <DataComparison
          clientId={clientId}
          projectId={projectId}
          item1={comparisonItems.item1}
          item2={comparisonItems.item2}
          onItemSelect={(item, position) => {
            setComparisonItems(prev => ({
              ...prev,
              [position === 1 ? 'item1' : 'item2']: item,
            }))
          }}
        />
      )}

      {/* Предпросмотр данных */}
      {previewData && (
        <DataPreview
          data={previewData.data}
          type={previewData.type}
        />
      )}

      {/* Управление экспортом */}
  <ExportManager
        clientId={clientId}
        projectId={projectId}
      />

      {/* Лента активности и настройки */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <ActivityTimeline
          clientId={clientId}
          projectId={projectId}
          maxEvents={50}
        />
        <SettingsPanel
          clientId={clientId}
          projectId={projectId}
        />
      </div>
    </div>
  )
}

// Рабочее пространство номенклатуры
const NomenclatureWorkspace: React.FC<{
  clientId: string
  projectId: string
  projectType?: string | null
}> = ({ clientId, projectId, projectType }) => {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Номенклатура</CardTitle>
          <CardDescription>
            Работа с нормализованной номенклатурой
          </CardDescription>
        </CardHeader>
        <CardContent>
          <NormalizationResultsTable
            isRunning={false}
            project={`${clientId}:${projectId}`}
            projectType={projectType === 'counterparty' ? null : projectType}
          />
        </CardContent>
      </Card>

      {/* Дедупликация для номенклатуры */}
      <IntelligentDeduplication
        clientId={clientId}
        projectId={projectId}
      />

      {/* Правила для номенклатуры */}
      <BusinessRulesManager
        clientId={clientId}
        projectId={projectId}
      />
    </div>
  )
}

// Рабочее пространство контрагентов
const CounterpartiesWorkspace: React.FC<{
  clientId: string
  projectId: string
  projectType?: string | null
}> = ({ clientId, projectId, projectType }) => {
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <CardTitle>Контрагенты</CardTitle>
          <CardDescription>
            Работа с нормализованными контрагентами
          </CardDescription>
        </CardHeader>
        <CardContent>
          <NormalizationResultsTable
            isRunning={false}
            project={`${clientId}:${projectId}`}
            projectType={projectType === 'counterparty' ? 'counterparty' : 'nomenclature_counterparties'}
          />
        </CardContent>
      </Card>

      {/* Дедупликация для контрагентов */}
      <IntelligentDeduplication
        clientId={clientId}
        projectId={projectId}
      />
    </div>
  )
}

