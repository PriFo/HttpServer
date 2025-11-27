'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { LogsPanel } from '@/components/normalization/logs-panel'
import { NormalizationResultsTable } from './normalization-results-table'
import { NormalizationPreviewStats } from './normalization-preview-stats'
import { NormalizationPreviewResults } from './normalization-preview-results'
import { FieldCompletenessAnalytics } from './field-completeness-analytics'
import { DataStructurePreview } from './data-structure-preview'
import { SmartProcessingRecommendations } from './smart-processing-recommendations'
import { InteractiveDataTree } from './interactive-data-tree'
import { DataQualityHeatmap } from './data-quality-heatmap'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Play, Square, RefreshCw, Clock, Zap, Loader2, Database, ChevronDown, ChevronUp, AlertTriangle, Search, ArrowUpDown, ArrowUp, ArrowDown, Sparkles, Settings, Package, Building2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { useError } from '@/contexts/ErrorContext'
import { apiClientJson } from '@/lib/api-client'
import type { NormalizationStatus, NormalizationType, PreviewStatsResponse } from '@/types/normalization'
import { motion } from 'framer-motion'
import { logger } from '@/lib/logger'
import { handleErrorWithDetails as handleErrorUtil } from '@/lib/error-handler'
import { toast } from 'sonner'

// Функция для оценки времени обработки проекта
function estimateProcessingTimeForProject(databaseCount: number): string {
  // Базовая оценка: ~5 минут на БД для номенклатуры, ~7 минут для контрагентов
  const avgTimePerDB = 6 // минут
  const totalMinutes = databaseCount * avgTimePerDB
  
  if (totalMinutes < 1) {
    return 'менее минуты'
  } else if (totalMinutes < 60) {
    return `${Math.ceil(totalMinutes)} мин`
  } else {
    const hours = Math.floor(totalMinutes / 60)
    const minutes = Math.ceil(totalMinutes % 60)
    if (minutes === 0) {
      return `${hours} ${hours === 1 ? 'час' : hours < 5 ? 'часа' : 'часов'}`
    }
    return `${hours} ${hours === 1 ? 'час' : hours < 5 ? 'часа' : 'часов'} ${minutes} мин`
  }
}

interface NormalizationProcessTabProps {
  database?: string
  project?: string // Формат: "clientId:projectId"
  normalizationType?: NormalizationType
}

export function NormalizationProcessTab({ database, project, normalizationType = 'both' }: NormalizationProcessTabProps) {
  const { handleError } = useError()
  const [status, setStatus] = useState<NormalizationStatus>({
    isRunning: false,
    progress: 0,
    processed: 0,
    total: 0,
    currentStep: 'Не запущено',
    logs: [],
  })
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [useKpved, setUseKpved] = useState(false)
  const [useOkpd2, setUseOkpd2] = useState(false)
  // Настройки для контрагентов
  const [verifyRequisites, setVerifyRequisites] = useState(false)
  const [normalizeAddresses, setNormalizeAddresses] = useState(false)
  const [standardizeLegalForms, setStandardizeLegalForms] = useState(false)

  const [clientId, setClientId] = useState<number | null>(null)
  const [projectId, setProjectId] = useState<number | null>(null)
  const [projectName, setProjectName] = useState<string | null>(null)
  const [projectType, setProjectType] = useState<string | null>(null)
  const [projectDatabasesCount, setProjectDatabasesCount] = useState<number | null>(null)
  const [projectDatabases, setProjectDatabases] = useState<Array<{ id: number; name: string; file_path: string; is_active?: boolean }>>([])
  const [loadingProjectInfo, setLoadingProjectInfo] = useState(false)
  const [showDatabasesList, setShowDatabasesList] = useState(false)
  const [databaseSearchQuery, setDatabaseSearchQuery] = useState('')
  const [databaseSortKey, setDatabaseSortKey] = useState<'name' | 'path' | null>(null)
  const [databaseSortDirection, setDatabaseSortDirection] = useState<'asc' | 'desc'>('asc')
  const [selectedDatabaseIds, setSelectedDatabaseIds] = useState<number[]>([])
  const projectInfoAbortRef = useRef<AbortController | null>(null)
  const [previewStats, setPreviewStats] = useState<PreviewStatsResponse | null>(null)
  const [dataTypeTab, setDataTypeTab] = useState<'nomenclature' | 'counterparties' | 'both'>('both')

  // Сброс состояния при изменении проекта
  useEffect(() => {
    // Сбрасываем состояние процесса при смене проекта
    setStatus({
      isRunning: false,
      progress: 0,
      processed: 0,
      total: 0,
      currentStep: 'Не запущено',
      logs: [],
    })
    setError(null)
    setIsLoading(false)
    setClientId(null)
    setProjectId(null)
    setProjectName(null)
    setProjectType(null)
    setProjectDatabasesCount(null)
    setProjectDatabases([])
    setSelectedDatabaseIds([])
  }, [project, database])

  const fetchProjectInfo = useCallback(
    async (clientIdNum: number, projectIdNum: number, options?: { preserveValues?: boolean }) => {
      const preserveValues = options?.preserveValues ?? false

      // Отменяем предыдущий запрос, если он еще выполняется
      if (projectInfoAbortRef.current) {
        projectInfoAbortRef.current.abort()
      }

      const controller = new AbortController()
      projectInfoAbortRef.current = controller
      const timeoutId = setTimeout(() => controller.abort(), 10000)

      setLoadingProjectInfo(true)
      if (!preserveValues) {
        setProjectName(null)
        setProjectDatabasesCount(null)
      }

      try {
        const [projectResponse, databasesResponse] = await Promise.allSettled([
          fetch(`/api/clients/${clientIdNum}/projects/${projectIdNum}`, {
            cache: 'no-store',
            signal: controller.signal,
          }),
          fetch(`/api/clients/${clientIdNum}/projects/${projectIdNum}/databases?active_only=true`, {
            cache: 'no-store',
            signal: controller.signal,
          }),
        ])

        if (controller.signal.aborted || projectInfoAbortRef.current !== controller) {
          return
        }

        if (projectResponse.status === 'fulfilled' && projectResponse.value.ok) {
          try {
            const projectData = await projectResponse.value.json()
            if (projectInfoAbortRef.current === controller && !controller.signal.aborted) {
              setProjectName(projectData.project?.name || null)
              setProjectType(projectData.project?.project_type || null)
            }
          } catch (err) {
            logger.error('Failed to parse project data', {
              component: 'NormalizationProcessTab',
              clientId,
              projectId
            }, err instanceof Error ? err : undefined)
          }
        } else if (projectResponse.status === 'rejected') {
          logger.error('Failed to fetch project', {
            component: 'NormalizationProcessTab',
            clientId,
            projectId
          }, projectResponse.reason instanceof Error ? projectResponse.reason : undefined)
        }

        if (databasesResponse.status === 'fulfilled' && databasesResponse.value.ok) {
          try {
            const databasesData = await databasesResponse.value.json()
            const dbList = databasesData.databases || []
            if (projectInfoAbortRef.current === controller && !controller.signal.aborted) {
              setProjectDatabasesCount(dbList.length)
              setProjectDatabases(
                dbList.map(
                  (db: { id: number; name?: string; file_path?: string; path?: string; is_active?: boolean }) => ({
                    id: db.id,
                    name: db.name || db.file_path?.split(/[/\\]/).pop() || 'БД',
                    file_path: db.file_path || db.path || '',
                    is_active: db.is_active !== undefined ? db.is_active : true,
                  })
                )
              )
            }
          } catch (err) {
            logger.error('Failed to parse databases data', {
              component: 'NormalizationProcessTab',
              clientId,
              projectId
            }, err instanceof Error ? err : undefined)
            setProjectDatabasesCount(0)
            setProjectDatabases([])
          }
        } else if (databasesResponse.status === 'rejected') {
          logger.error('Failed to fetch databases', {
            component: 'NormalizationProcessTab',
            clientId,
            projectId
          }, databasesResponse.reason instanceof Error ? databasesResponse.reason : undefined)
          setProjectDatabasesCount(0)
          setProjectDatabases([])
        } else if (databasesResponse.status === 'fulfilled' && !databasesResponse.value.ok) {
          setProjectDatabasesCount(0)
          setProjectDatabases([])
        }
      } catch (err) {
        const errorDetails = handleErrorUtil(
          err,
          'NormalizationProcessTab',
          'fetchProjectInfo',
          { clientId, projectId }
        )
        logger.error('Failed to fetch project info', {
          component: 'NormalizationProcessTab',
          clientId,
          projectId,
          errorMessage: errorDetails.message
        }, err instanceof Error ? err : undefined)
        handleError(err as Error)
      } finally {
        clearTimeout(timeoutId)
        if (projectInfoAbortRef.current === controller && !controller.signal.aborted) {
          setLoadingProjectInfo(false)
          projectInfoAbortRef.current = null
        }
      }
    },
    [handleError]
  )

  // Парсим проект из пропса и загружаем информацию о проекте
  useEffect(() => {
    if (project) {
      const parts = project.split(':')
      if (parts.length === 2) {
        const clientIdNum = parseInt(parts[0], 10)
        const projectIdNum = parseInt(parts[1], 10)
        if (!isNaN(clientIdNum) && !isNaN(projectIdNum)) {
          setClientId(clientIdNum)
          setProjectId(projectIdNum)
          fetchProjectInfo(clientIdNum, projectIdNum)
        }
      }
    } else {
      setClientId(null)
      setProjectId(null)
      setProjectName(null)
      setProjectType(null)
      setProjectDatabasesCount(null)
      setProjectDatabases([])
      setLoadingProjectInfo(false)
      setShowDatabasesList(false)
      projectInfoAbortRef.current?.abort()
    }
  }, [project, fetchProjectInfo])

  useEffect(() => {
    return () => {
      projectInfoAbortRef.current?.abort()
    }
  }, [])

  // Обновляем информацию о проекте после завершения нормализации
  useEffect(() => {
    if (project && clientId && projectId && !status.isRunning && projectDatabasesCount !== null) {
      // Обновляем количество БД и список после завершения нормализации
      const updateDatabasesInfo = async () => {
        try {
          const databasesResponse = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases?active_only=true`, {
            cache: 'no-store',
            signal: AbortSignal.timeout(5000),
          })
          if (databasesResponse.ok) {
            const databasesData = await databasesResponse.json()
            const dbList = databasesData.databases || []
            const newCount = dbList.length
            if (newCount !== projectDatabasesCount) {
              setProjectDatabasesCount(newCount)
              setProjectDatabases(dbList.map((db: { id: number; name?: string; file_path?: string; path?: string; is_active?: boolean }) => ({
                id: db.id,
                name: db.name || db.file_path?.split(/[/\\]/).pop() || 'БД',
                file_path: db.file_path || db.path || '',
                is_active: db.is_active !== undefined ? db.is_active : true
              })))
            }
          }
        } catch (err) {
          logger.error('Failed to update databases info', {
            component: 'NormalizationProcessTab',
            clientId,
            projectId,
            projectDatabasesCount
          }, err instanceof Error ? err : undefined)
        }
      }
      
      // Обновляем с небольшой задержкой, чтобы дать время серверу обновить данные
      const timeoutId = setTimeout(updateDatabasesInfo, 1000)
      return () => clearTimeout(timeoutId)
    }
  }, [project, clientId, projectId, status.isRunning, projectDatabasesCount])

  // Находим клиента и проект по базе данных при монтировании (только если проект не указан)
  useEffect(() => {
    if (database && !project) {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 секунд таймаут
      
      fetch(`/api/databases/find-project?file_path=${encodeURIComponent(database)}`, {
        signal: controller.signal,
        cache: 'no-store',
      })
        .then(res => {
          clearTimeout(timeoutId)
          if (!res.ok) {
            // Если 404 - база данных не найдена в проектах, это нормально
            if (res.status === 404) {
              logger.debug('Database not found in any project, using default status endpoint', {
                component: 'NormalizationProcessTab',
                database
              })
              return null
            }
            // Для других ошибок логируем
            logger.warn('Failed to find project', {
              component: 'NormalizationProcessTab',
              database,
              status: res.status,
              statusText: res.statusText
            })
            return null
          }
          return res.json()
        })
        .then(data => {
          if (data && data.client_id && data.project_id) {
            setClientId(data.client_id)
            setProjectId(data.project_id)
          }
        })
        .catch(err => {
          clearTimeout(timeoutId)
          if (err.name === 'AbortError') {
            logger.warn('Find project request timed out', {
              component: 'NormalizationProcessTab',
              database
            })
          } else {
            logger.error('Failed to find project', {
              component: 'NormalizationProcessTab',
              database
            }, err instanceof Error ? err : undefined)
          }
          // Не устанавливаем ошибку, так как это не критично - можно использовать общий эндпоинт
        })
      
      return () => {
        clearTimeout(timeoutId)
        controller.abort()
      }
    }
  }, [database, project])

  const fetchStatus = useCallback(async () => {
    try {
      // Если есть клиент и проект, используем их API
      let response: Response
      if (clientId && projectId) {
        // Определяем тип проекта и используем соответствующий endpoint
        const isCounterpartyProject = projectType === 'counterparty' || projectType === 'nomenclature_counterparties'
        
        if (isCounterpartyProject) {
          // Для контрагентов используем специальный endpoint
          const url = `/api/counterparties/normalization/status?client_id=${clientId}&project_id=${projectId}`
          try {
            response = await fetch(url, {
              cache: 'no-store',
              signal: AbortSignal.timeout(10000), // 10 секунд таймаут
            })
          } catch (fetchError) {
            // Если fetch упал (сеть, таймаут и т.д.), пробуем общий endpoint
            logger.warn('Failed to fetch counterparties status, trying general endpoint', {
              component: 'NormalizationProcessTab',
              clientId,
              projectId
            }, fetchError instanceof Error ? fetchError : undefined)
            response = await fetch('/api/counterparties/normalization/status', {
              cache: 'no-store',
              signal: AbortSignal.timeout(10000),
            })
          }
        } else {
          // Для номенклатуры используем общий endpoint
          const url = `/api/clients/${clientId}/projects/${projectId}/normalization/status`
          try {
            response = await fetch(url, {
              cache: 'no-store',
              signal: AbortSignal.timeout(10000), // 10 секунд таймаут
            })
          } catch (fetchError) {
            // Если fetch упал (сеть, таймаут и т.д.), пробуем старый endpoint
            logger.warn('Failed to fetch client-specific status, trying default endpoint', {
              component: 'NormalizationProcessTab',
              clientId,
              projectId
            }, fetchError instanceof Error ? fetchError : undefined)
            response = await fetch('/api/normalization/status', {
              cache: 'no-store',
              signal: AbortSignal.timeout(10000),
            })
          }
        }
      } else {
        response = await fetch('/api/normalization/status', {
          cache: 'no-store',
          signal: AbortSignal.timeout(10000),
        })
      }
      
      if (!response.ok) {
        // Если 500 ошибка, пробуем получить детали из ответа
        let errorMessage = 'Не удалось получить статус'
        let errorData: { error?: string; message?: string } | null = null
        try {
          errorData = await response.json()
          if (errorData && errorData.error) {
            errorMessage = errorData.error
          } else if (errorData && errorData.message) {
            errorMessage = errorData.message
          }
        } catch {
          // Если не удалось распарсить JSON, используем стандартное сообщение
          errorMessage = `Ошибка сервера: ${response.status} ${response.statusText}`
        }
        
        // Если это 404 или 400, и у нас есть clientId/projectId, пробуем старый endpoint
        if ((response.status === 404 || response.status === 400) && clientId && projectId) {
          logger.warn('Client-specific endpoint failed, trying default endpoint', {
            component: 'NormalizationProcessTab',
            clientId,
            projectId
          })
          try {
            const fallbackResponse = await fetch('/api/normalization/status', {
              cache: 'no-store',
              signal: AbortSignal.timeout(10000),
            })
            if (fallbackResponse.ok) {
              const fallbackData = await fallbackResponse.json()
              setStatus({
                isRunning: fallbackData.isRunning || fallbackData.is_running || false,
                progress: fallbackData.progress || 0,
                processed: fallbackData.processed || 0,
                total: fallbackData.total || 0,
                success: fallbackData.success,
                errors: fallbackData.errors,
                currentStep: fallbackData.currentStep || fallbackData.current_step || 'Не запущено',
                logs: fallbackData.logs || [],
                startTime: fallbackData.startTime || fallbackData.start_time,
                elapsedTime: fallbackData.elapsedTime || fallbackData.elapsed_time,
                rate: fallbackData.rate,
                kpvedClassified: fallbackData.kpvedClassified || fallbackData.kpved_classified,
                kpvedTotal: fallbackData.kpvedTotal || fallbackData.kpved_total,
                kpvedProgress: fallbackData.kpvedProgress || fallbackData.kpved_progress,
              })
              setError(null)
              return
            }
          } catch {
            // Игнорируем ошибки fallback
          }
        }
        
        throw new Error(errorMessage)
      }
      
      const data = await response.json()
      
      // Если в ответе есть ошибка, но статус 200, все равно показываем ошибку
      if (data.error && !data.is_running && !data.isRunning) {
        logger.warn('Status response contains error', {
          component: 'NormalizationProcessTab',
          clientId,
          projectId,
          error: data.error
        })
      }
      
      // Преобразуем формат ответа для единообразия
      setStatus({
        isRunning: data.isRunning || data.is_running || false,
        progress: data.progress || 0,
        processed: data.processed || 0,
        total: data.total || 0,
        success: data.success,
        errors: data.errors,
        currentStep: data.currentStep || data.current_step || 'Не запущено',
        logs: data.logs || [],
        startTime: data.startTime || data.start_time,
        elapsedTime: data.elapsedTime || data.elapsed_time,
        rate: data.rate,
        kpvedClassified: data.kpvedClassified || data.kpved_classified,
        kpvedTotal: data.kpvedTotal || data.kpved_total,
        kpvedProgress: data.kpvedProgress || data.kpved_progress,
        // Поля для нормализации контрагентов
        sessions: data.sessions || [],
        databases: data.databases || [],
        active_sessions_count: data.active_sessions_count || 0,
        total_databases_count: data.total_databases_count || 0,
      })
      setError(null)
    } catch (err) {
      const errorDetails = handleErrorUtil(
        err,
        'NormalizationProcessTab',
        'fetchStatus',
        { clientId, projectId, database, project }
      )
      logger.error('Error fetching normalization status', {
        component: 'NormalizationProcessTab',
        clientId,
        projectId,
        database,
        project,
        errorMessage: errorDetails.message
      }, err instanceof Error ? err : undefined)
      const errorMessage = err instanceof Error ? err.message : 'Не удалось подключиться к серверу'
      
      // Не показываем ошибку, если нормализация запущена (может быть временная проблема сети)
      if (!status.isRunning) {
        setError(errorMessage)
      }
    }
  }, [status.isRunning, clientId, projectId, projectType])

  useEffect(() => {
    // Первоначальная загрузка
    fetchStatus()

    // Автообновление статуса каждые 2 секунды, если процесс запущен
    // Не обновляем, если есть ошибка (кроме случаев, когда процесс запущен)
    const interval = setInterval(() => {
      if (status.isRunning) {
        fetchStatus()
      }
    }, 2000)

    return () => clearInterval(interval)
  }, [status.isRunning, fetchStatus, project, database])

  const handleStart = async () => {
    setIsLoading(true)
    setError(null)
    
    // Оптимистичное обновление UI
    const previousStatus = { ...status }
    setStatus(prev => ({
      ...prev,
      isRunning: true,
      currentStep: 'Запуск нормализации...',
      progress: 0,
      processed: 0,
      total: prev.total || 0,
    }))
    
    try {
      // Если выбран проект, используем его напрямую
      if (clientId && projectId) {
        // Проверяем, что в проекте есть активные базы данных
        if (projectDatabasesCount !== null && projectDatabasesCount === 0) {
          setError('В проекте нет активных баз данных для обработки')
          setIsLoading(false)
          return
        }
        
        try {
          // Определяем тип проекта и используем соответствующий endpoint
          const isCounterpartyProject = projectType === 'counterparty' || projectType === 'nomenclature_counterparties'
          
          // Определяем, использовать ли выбранные БД или все активные
          const useSelectedDatabases = selectedDatabaseIds.length > 0
          
          if (isCounterpartyProject) {
            // Для контрагентов используем специальный endpoint
            await apiClientJson(`/api/counterparties/normalization/start?client_id=${clientId}&project_id=${projectId}`, {
              method: 'POST',
              body: JSON.stringify({
                all_active: !useSelectedDatabases, // Запускаем для всех баз проекта или выбранных
                database_ids: useSelectedDatabases ? selectedDatabaseIds : undefined,
              }),
            })
          } else {
            // Для номенклатуры используем общий endpoint
            await apiClientJson(`/api/clients/${clientId}/projects/${projectId}/normalization/start`, {
              method: 'POST',
              body: JSON.stringify({
                all_active: !useSelectedDatabases, // Запускаем для всех баз проекта или выбранных
                database_ids: useSelectedDatabases ? selectedDatabaseIds : undefined,
                use_kpved: useKpved,
                use_okpd2: useOkpd2,
                // database_path не указываем, так как обрабатываем все базы или выбранные
              }),
            })
          }
          
          // Обновляем статус после запуска
          setTimeout(() => {
            fetchStatus()
          }, 500)
          return
        } catch (startError) {
          handleError(startError, 'Не удалось запустить нормализацию через проект')
          return
        }
      }

      // Если указана база данных, сначала находим клиента и проект
      let foundClientId: number | null = null
      let foundProjectId: number | null = null
      
      if (database) {
        try {
          const findData = await apiClientJson<{ client_id: number; project_id: number }>(
            `/api/databases/find-project?file_path=${encodeURIComponent(database)}`
          )
          foundClientId = findData.client_id
          foundProjectId = findData.project_id
        } catch {
          // Игнорируем ошибку поиска проекта, продолжаем со старым API
        }
      }

      // Если нашли клиента и проект, используем новый API
      if (foundClientId && foundProjectId) {
        try {
          await apiClientJson(`/api/clients/${foundClientId}/projects/${foundProjectId}/normalization/start`, {
            method: 'POST',
            body: JSON.stringify({
              database_path: database,
              all_active: false,
              use_kpved: useKpved,
              use_okpd2: useOkpd2,
            }),
          })
        } catch {
          // Если клиентский endpoint не работает, пробуем старый
          await apiClientJson('/api/normalization/start', {
            method: 'POST',
            body: JSON.stringify({
              use_ai: false,
              min_confidence: 0.8,
              database: database,
              use_kpved: useKpved,
              use_okpd2: useOkpd2,
            }),
          })
        }

        // Обновляем статус после запуска
        setTimeout(() => {
          fetchStatus()
        }, 500)
      } else {
        // Используем старый API, если не нашли клиента/проекта
        await apiClientJson('/api/normalization/start', {
          method: 'POST',
          body: JSON.stringify({
            use_ai: false,
            min_confidence: 0.8,
            database: database,
            use_kpved: useKpved,
            use_okpd2: useOkpd2,
          }),
        })

        // Обновляем статус после запуска
        setTimeout(() => {
          fetchStatus()
        }, 500)
      }
    } catch (err) {
      // Откатываем оптимистичное обновление при ошибке
      setStatus(previousStatus)
      handleError(err, 'Не удалось запустить нормализацию')
    } finally {
      setIsLoading(false)
    }
  }

  const handleStop = async () => {
    setIsLoading(true)
    setError(null)
    
    // Оптимистичное обновление UI
    const previousStatus = { ...status }
    setStatus(prev => ({
      ...prev,
      isRunning: false,
      currentStep: 'Остановка нормализации...',
    }))
    
    try {
      // Если есть клиент и проект, используем их API
      if (clientId && projectId) {
        try {
          // Определяем тип проекта и используем соответствующий endpoint
          const isCounterpartyProject = projectType === 'counterparty' || projectType === 'nomenclature_counterparties'
          
          if (isCounterpartyProject) {
            // Для контрагентов используем специальный endpoint
            await apiClientJson(`/api/counterparties/normalization/stop`, {
              method: 'POST',
            })
          } else {
            // Для номенклатуры используем общий endpoint
            await apiClientJson(`/api/clients/${clientId}/projects/${projectId}/normalization/stop`, {
              method: 'POST',
            })
          }
        } catch {
          // Если клиентский endpoint не работает, пробуем старый
          try {
            await apiClientJson('/api/counterparties/normalization/stop', {
              method: 'POST',
            })
          } catch {
            await apiClientJson('/api/normalization/stop', {
              method: 'POST',
            })
          }
        }
      } else {
        await apiClientJson('/api/normalization/stop', {
          method: 'POST',
        })
      }

      // Обновляем статус после остановки (несколько раз для надежности)
      setTimeout(() => {
        fetchStatus()
      }, 300)
      setTimeout(() => {
        fetchStatus()
      }, 1000)
      setTimeout(() => {
        fetchStatus()
      }, 2000)
    } catch (err) {
      // Откатываем оптимистичное обновление при ошибке
      setStatus(previousStatus)
      handleError(err, 'Не удалось остановить нормализацию')
    } finally {
      setIsLoading(false)
    }
  }

  const progressPercent = status.total > 0 
    ? Math.min(100, (status.processed / status.total) * 100)
    : status.progress

  return (
    <div className="space-y-6">
      {/* Статистика перед запуском - показывает готовность системы обрабатывать все БД */}
      {!status.isRunning && clientId && projectId && (
        <>
          <NormalizationPreviewStats 
            clientId={clientId} 
            projectId={projectId}
            normalizationType={normalizationType}
            selectedDatabaseIds={selectedDatabaseIds}
            onDatabasesSelected={setSelectedDatabaseIds}
            onFullStatsUpdate={(stats) => {
              setPreviewStats(stats)
            }}
          />
          
          {/* Улучшенная аналитика с табами */}
          {previewStats && previewStats.total_records > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              <Tabs value={dataTypeTab} onValueChange={(v) => setDataTypeTab(v as any)} className="space-y-6">
                <TabsList className="grid w-full grid-cols-3">
                  <TabsTrigger value="nomenclature" className="flex items-center gap-2">
                    <Package className="h-4 w-4" />
                    Номенклатура
                    {previewStats.total_nomenclature > 0 && (
                      <Badge variant="secondary" className="ml-1">
                        {previewStats.total_nomenclature.toLocaleString('ru-RU')}
                      </Badge>
                    )}
                  </TabsTrigger>
                  <TabsTrigger value="counterparties" className="flex items-center gap-2">
                    <Building2 className="h-4 w-4" />
                    Контрагенты
                    {previewStats.total_counterparties > 0 && (
                      <Badge variant="secondary" className="ml-1">
                        {previewStats.total_counterparties.toLocaleString('ru-RU')}
                      </Badge>
                    )}
                  </TabsTrigger>
                  <TabsTrigger value="both" className="flex items-center gap-2">
                    <Database className="h-4 w-4" />
                    Оба типа
                    {previewStats.total_records > 0 && (
                      <Badge variant="secondary" className="ml-1">
                        {previewStats.total_records.toLocaleString('ru-RU')}
                      </Badge>
                    )}
                  </TabsTrigger>
                </TabsList>

                <TabsContent value="nomenclature" className="space-y-6">
                  <FieldCompletenessAnalytics
                    completeness={previewStats.completeness_metrics}
                    normalizationType="nomenclature"
                    totalNomenclature={previewStats.total_nomenclature}
                    isLoading={false}
                  />
                  <DataStructurePreview
                    normalizationType="nomenclature"
                    isLoading={false}
                    onExport={(format) => {
                      toast.info(`Экспорт данных номенклатуры в формате ${format.toUpperCase()} будет доступен в следующей версии`)
                    }}
                  />
                  <DataQualityHeatmap
                    stats={previewStats}
                    normalizationType="nomenclature"
                    isLoading={false}
                  />
                </TabsContent>

                <TabsContent value="counterparties" className="space-y-6">
                  <FieldCompletenessAnalytics
                    completeness={previewStats.completeness_metrics}
                    normalizationType="counterparties"
                    totalCounterparties={previewStats.total_counterparties}
                    isLoading={false}
                  />
                  <DataStructurePreview
                    normalizationType="counterparties"
                    isLoading={false}
                    onExport={(format) => {
                      toast.info(`Экспорт данных контрагентов в формате ${format.toUpperCase()} будет доступен в следующей версии`)
                    }}
                  />
                  <DataQualityHeatmap
                    stats={previewStats}
                    normalizationType="counterparties"
                    isLoading={false}
                  />
                </TabsContent>

                <TabsContent value="both" className="space-y-6">
                  <InteractiveDataTree
                    stats={previewStats}
                    normalizationType="both"
                    isLoading={false}
                  />
                  <SmartProcessingRecommendations
                    stats={previewStats}
                    normalizationType="both"
                    onQuickStart={async (type) => {
                      if (type === 'problematic') {
                        // Запуск обработки проблемных зон
                        toast.info('Запуск обработки проблемных зон...')
                        await handleStart()
                      } else if (type === 'full') {
                        // Полная обработка всех данных
                        toast.info('Запуск полной обработки всех данных...')
                        await handleStart()
                      } else if (type === 'selective') {
                        // Выборочная обработка - открываем диалог выбора
                        toast.info('Выборочная обработка будет доступна в следующей версии')
                      }
                    }}
                  />
                  <FieldCompletenessAnalytics
                    completeness={previewStats.completeness_metrics}
                    normalizationType="both"
                    totalNomenclature={previewStats.total_nomenclature}
                    totalCounterparties={previewStats.total_counterparties}
                    isLoading={false}
                  />
                  <DataQualityHeatmap
                    stats={previewStats}
                    normalizationType="both"
                    isLoading={false}
                  />
                </TabsContent>
              </Tabs>
            </motion.div>
          )}
          
          {/* Сообщение, если нет данных для аналитики */}
          {previewStats && previewStats.total_records === 0 && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
            >
              <Card className="bg-muted/50 border-dashed">
                <CardContent className="pt-6">
                  <div className="text-center text-muted-foreground">
                    <Database className="h-8 w-8 mx-auto mb-2 opacity-50" />
                    <p className="font-medium">Нет данных для аналитики</p>
                    <p className="text-sm mt-1">
                      Выберите базы данных с записями для отображения детальной аналитики
                    </p>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          )}
        </>
      )}

      {/* Статус и управление */}
      <Card className="backdrop-blur-sm bg-card/95 border-border/50 shadow-lg">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Процесс нормализации</CardTitle>
              <CardDescription>
                <div className="space-y-2">
                  {project && clientId && projectId ? (
                    loadingProjectInfo ? (
                      <div className="flex items-center gap-2">
                        <Loader2 className="h-3 w-3 animate-spin text-muted-foreground" />
                        <span>Загрузка информации о проекте...</span>
                      </div>
                    ) : (
                      <>
                        <div>
                          {projectName 
                            ? `Проект "${projectName}"${projectType ? ` (${projectType === 'counterparty' ? 'Контрагенты' : projectType === 'nomenclature' ? 'Номенклатура' : projectType === 'nomenclature_counterparties' ? 'Номенклатура + Контрагенты' : projectType})` : ''}${projectDatabasesCount !== null ? ` (${projectDatabasesCount} ${projectDatabasesCount === 1 ? 'база данных' : projectDatabasesCount < 5 ? 'базы данных' : 'баз данных'})` : ''}: ${selectedDatabaseIds.length > 0 ? `обработка ${selectedDatabaseIds.length} выбранных баз данных` : 'обработка всех активных баз данных'}`
                            : `Проект (ID: ${projectId})${projectType ? ` (${projectType === 'counterparty' ? 'Контрагенты' : projectType === 'nomenclature' ? 'Номенклатура' : projectType === 'nomenclature_counterparties' ? 'Номенклатура + Контрагенты' : projectType})` : ''}: ${selectedDatabaseIds.length > 0 ? `обработка ${selectedDatabaseIds.length} выбранных баз данных` : 'обработка всех активных баз данных проекта'}`}
                        </div>
                        {projectDatabasesCount !== null && projectDatabasesCount > 0 && projectDatabases.length > 0 && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-auto p-0 text-xs text-muted-foreground hover:text-foreground"
                            onClick={() => setShowDatabasesList(!showDatabasesList)}
                          >
                            {showDatabasesList ? (
                              <>
                                <ChevronUp className="h-3 w-3 mr-1" />
                                Скрыть список баз данных
                              </>
                            ) : (
                              <>
                                <ChevronDown className="h-3 w-3 mr-1" />
                                Показать список баз данных ({projectDatabasesCount})
                              </>
                            )}
                          </Button>
                        )}
                      </>
                    )
                  ) : database 
                    ? `База данных: ${database.split(/[/\\]/).pop() || database}` 
                    : 'Выберите проект или базу данных для запуска нормализации'}
                </div>
              </CardDescription>
            </div>
            <div className="flex items-center gap-2">
              <Badge 
                variant={
                  status.isRunning 
                    ? 'default' 
                    : status.currentStep?.includes('остановлена') || status.currentStep?.includes('остановлен')
                      ? 'destructive'
                      : 'secondary'
                }
              >
                {status.isRunning 
                  ? 'Выполняется' 
                  : status.currentStep?.includes('остановлена') || status.currentStep?.includes('остановлен')
                    ? 'Остановлено пользователем'
                    : 'Остановлено'}
              </Badge>
              {project && clientId && projectId && (
                <Button
                  variant="outline"
                  size="icon"
                  onClick={async () => {
                    // Обновляем информацию о проекте
                    const parts = project.split(':')
                    if (parts.length === 2) {
                      const clientIdNum = parseInt(parts[0], 10)
                      const projectIdNum = parseInt(parts[1], 10)
                      if (!isNaN(clientIdNum) && !isNaN(projectIdNum)) {
                        setLoadingProjectInfo(true)
                        try {
                          const [projectResponse, databasesResponse] = await Promise.allSettled([
                            fetch(`/api/clients/${clientIdNum}/projects/${projectIdNum}`, {
                              cache: 'no-store',
                              signal: AbortSignal.timeout(10000),
                            }),
                            fetch(`/api/clients/${clientIdNum}/projects/${projectIdNum}/databases?active_only=true`, {
                              cache: 'no-store',
                              signal: AbortSignal.timeout(10000),
                            })
                          ])
                          
                          if (projectResponse.status === 'fulfilled' && projectResponse.value.ok) {
                            try {
                              const projectData = await projectResponse.value.json()
                              setProjectName(projectData.project?.name || null)
                            } catch (err) {
                              logger.error('Failed to parse project data during manual refresh', {
                                component: 'NormalizationProcessTab',
                                clientId,
                                projectId
                              }, err instanceof Error ? err : undefined)
                            }
                          }
                          
                          if (databasesResponse.status === 'fulfilled' && databasesResponse.value.ok) {
                            try {
                              const databasesData = await databasesResponse.value.json()
                              const dbList = databasesData.databases || []
                              setProjectDatabasesCount(dbList.length)
                              setProjectDatabases(dbList.map((db: { id: number; name?: string; file_path?: string; path?: string; is_active?: boolean }) => ({
                                id: db.id,
                                name: db.name || db.file_path?.split(/[/\\]/).pop() || 'БД',
                                file_path: db.file_path || db.path || '',
                                is_active: db.is_active !== undefined ? db.is_active : true
                              })))
                            } catch (err) {
                              logger.error('Failed to parse databases data during refresh', {
                                component: 'NormalizationProcessTab',
                                clientId,
                                projectId
                              }, err instanceof Error ? err : undefined)
                              setProjectDatabasesCount(0)
                              setProjectDatabases([])
                            }
                          } else {
                            setProjectDatabasesCount(0)
                            setProjectDatabases([])
                          }
                        } catch (err) {
                          const errorDetails = handleErrorUtil(
                            err,
                            'NormalizationProcessTab',
                            'refreshProjectInfo',
                            { clientId, projectId }
                          )
                          logger.error('Failed to refresh project info', {
                            component: 'NormalizationProcessTab',
                            clientId,
                            projectId,
                            errorMessage: errorDetails.message
                          }, err instanceof Error ? err : undefined)
                        } finally {
                          setLoadingProjectInfo(false)
                        }
                      }
                    }
                  }}
                  disabled={loadingProjectInfo}
                  title="Обновить информацию о проекте"
                >
                  <RefreshCw className={`h-4 w-4 ${loadingProjectInfo ? 'animate-spin' : ''}`} />
                </Button>
              )}
              <Button
                variant="outline"
                size="icon"
                onClick={fetchStatus}
                disabled={isLoading}
                title="Обновить статус нормализации"
              >
                <RefreshCw className={`h-4 w-4 ${isLoading ? 'animate-spin' : ''}`} />
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          {error && (
            <Alert variant="destructive">
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {/* Предупреждение при большом количестве БД */}
          {project && clientId && projectId && projectDatabasesCount !== null && projectDatabasesCount > 10 && !status.isRunning && (
            <Alert>
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                <div className="space-y-1">
                  <p className="font-medium">
                    Внимание: будет обработано {projectDatabasesCount} баз данных
                  </p>
                  <p className="text-sm text-muted-foreground">
                    Процесс нормализации может занять значительное время. Убедитесь, что у вас достаточно ресурсов для обработки всех баз данных.
                  </p>
                </div>
              </AlertDescription>
            </Alert>
          )}

          {/* Список баз данных проекта */}
          {project && clientId && projectId && showDatabasesList && projectDatabases.length > 0 && (
            <div className="space-y-2 p-4 bg-muted/50 rounded-lg border">
              <div className="flex items-center justify-between mb-3">
                <div className="text-sm font-medium flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  Базы данных, которые будут обработаны:
                </div>
              </div>
              
              {/* Поиск по базам данных */}
              {projectDatabases.length > 5 && (
                <div className="relative mb-3">
                  <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Поиск по названию или пути..."
                    value={databaseSearchQuery}
                    onChange={(e) => setDatabaseSearchQuery(e.target.value)}
                    className="pl-9"
                  />
                </div>
              )}
              
              {/* Заголовки с сортировкой */}
              {projectDatabases.length > 1 && (
                <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground pb-2 border-b">
                  <div className="flex-1 flex items-center gap-1">
                    <span>Название</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-4 w-4 p-0"
                      onClick={() => {
                        if (databaseSortKey === 'name') {
                          setDatabaseSortDirection(databaseSortDirection === 'asc' ? 'desc' : 'asc')
                        } else {
                          setDatabaseSortKey('name')
                          setDatabaseSortDirection('asc')
                        }
                      }}
                    >
                      {databaseSortKey === 'name' ? (
                        databaseSortDirection === 'asc' ? (
                          <ArrowUp className="h-3 w-3" />
                        ) : (
                          <ArrowDown className="h-3 w-3" />
                        )
                      ) : (
                        <ArrowUpDown className="h-3 w-3" />
                      )}
                    </Button>
                  </div>
                  <div className="w-[200px] flex items-center gap-1">
                    <span>Путь</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-4 w-4 p-0"
                      onClick={() => {
                        if (databaseSortKey === 'path') {
                          setDatabaseSortDirection(databaseSortDirection === 'asc' ? 'desc' : 'asc')
                        } else {
                          setDatabaseSortKey('path')
                          setDatabaseSortDirection('asc')
                        }
                      }}
                    >
                      {databaseSortKey === 'path' ? (
                        databaseSortDirection === 'asc' ? (
                          <ArrowUp className="h-3 w-3" />
                        ) : (
                          <ArrowDown className="h-3 w-3" />
                        )
                      ) : (
                        <ArrowUpDown className="h-3 w-3" />
                      )}
                    </Button>
                  </div>
                  <div className="w-20 text-center">Статус</div>
                </div>
              )}
              
              <div className="space-y-1.5 max-h-60 overflow-y-auto">
                {projectDatabases
                  .filter((db) => {
                    if (!databaseSearchQuery.trim()) return true
                    const query = databaseSearchQuery.toLowerCase()
                    return (
                      db.name.toLowerCase().includes(query) ||
                      db.file_path.toLowerCase().includes(query)
                    )
                  })
                  .sort((a, b) => {
                    if (!databaseSortKey) return 0
                    
                    let aValue: string
                    let bValue: string
                    
                    if (databaseSortKey === 'name') {
                      aValue = a.name.toLowerCase()
                      bValue = b.name.toLowerCase()
                    } else {
                      aValue = (a.file_path || '').toLowerCase()
                      bValue = (b.file_path || '').toLowerCase()
                    }
                    
                    const comparison = aValue.localeCompare(bValue, 'ru-RU', { numeric: true })
                    return databaseSortDirection === 'asc' ? comparison : -comparison
                  })
                  .map((db, index) => (
                  <div
                    key={`project-db-${db.id}-${index}`}
                    className="flex items-center gap-2 text-sm p-2 bg-background rounded border hover:bg-muted/50 transition-colors"
                  >
                    <Database className="h-3 w-3 text-muted-foreground shrink-0" />
                    <span className="font-medium flex-1 truncate" title={db.name}>{db.name}</span>
                    {db.file_path && (
                      <span className="text-xs text-muted-foreground font-mono truncate w-[200px]" title={db.file_path}>
                        {db.file_path.split(/[/\\]/).pop() || db.file_path}
                      </span>
                    )}
                    <div className="w-20 flex justify-center">
                      <Badge variant={db.is_active !== false ? 'default' : 'secondary'} className="text-xs">
                        {db.is_active !== false ? 'Активна' : 'Неактивна'}
                      </Badge>
                    </div>
                  </div>
                  ))}
              </div>
              {(() => {
                const filteredCount = projectDatabases.filter((db) => {
                  if (!databaseSearchQuery.trim()) return true
                  const query = databaseSearchQuery.toLowerCase()
                  return (
                    db.name.toLowerCase().includes(query) ||
                    db.file_path.toLowerCase().includes(query)
                  )
                }).length
                
                return (
                  <div className="text-xs text-muted-foreground mt-2 pt-2 border-t">
                    {databaseSearchQuery.trim() ? (
                      <>Показано {filteredCount} из {projectDatabases.length} баз данных</>
                    ) : (
                      <>Показано {projectDatabases.length} из {projectDatabasesCount} баз данных</>
                    )}
                  </div>
                )
              })()}
            </div>
          )}

          {/* Текущий шаг */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Текущий шаг:</span>
              <span className={`font-medium ${
                status.currentStep?.includes('остановлена') || status.currentStep?.includes('остановлен')
                  ? 'text-orange-600'
                  : ''
              }`}>
                {status.currentStep}
              </span>
            </div>
            {status.currentStep?.includes('остановлена') || status.currentStep?.includes('остановлен') ? (
              <div className="text-xs text-muted-foreground mt-1">
                Частично обработанные данные сохранены. Вы можете продолжить нормализацию позже.
              </div>
            ) : null}
          </div>

          {/* Прогресс */}
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span className="text-muted-foreground">Прогресс:</span>
              <span className="font-medium">
                {status.processed.toLocaleString()} / {status.total.toLocaleString()} 
                {status.total > 0 && ` (${progressPercent.toFixed(1)}%)`}
              </span>
            </div>
            <Progress value={progressPercent} className="h-2" />
          </div>

          {/* Статистика */}
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {(status.success !== undefined || status.errors !== undefined) && (
              <>
                {status.success !== undefined && (
                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground">Успешно</div>
                    <div className="text-2xl font-bold text-green-600">
                      {status.success.toLocaleString()}
                    </div>
                  </div>
                )}
                {status.errors !== undefined && (
                  <div className="space-y-1">
                    <div className="text-sm text-muted-foreground">Ошибок</div>
                    <div className="text-2xl font-bold text-red-600">
                      {status.errors.toLocaleString()}
                    </div>
                  </div>
                )}
              </>
            )}
            {(status.rate !== undefined && status.rate > 0) && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground flex items-center gap-1">
                  <Zap className="h-3 w-3" />
                  Скорость
                </div>
                <div className="text-2xl font-bold text-blue-600">
                  {status.rate >= 1 
                    ? `${status.rate.toFixed(1)}/сек`
                    : `${(status.rate * 60).toFixed(1)}/мин`
                  }
                </div>
                {status.processed > 0 && status.total > 0 && status.rate > 0 && (
                  <div className="text-xs text-muted-foreground">
                    {(() => {
                      const remaining = (status.total - status.processed) / status.rate
                      if (remaining < 60) {
                        return `~${Math.ceil(remaining)}с осталось`
                      } else if (remaining < 3600) {
                        return `~${Math.ceil(remaining / 60)}мин осталось`
                      } else {
                        return `~${Math.ceil(remaining / 3600)}ч осталось`
                      }
                    })()}
                  </div>
                )}
              </div>
            )}
            {status.elapsedTime && (
              <div className="space-y-1">
                <div className="text-sm text-muted-foreground flex items-center gap-1">
                  <Clock className="h-3 w-3" />
                  Время
                </div>
                <div className="text-2xl font-bold">
                  {status.elapsedTime}
                </div>
              </div>
            )}
          </div>

          {/* Метрики КПВЭД */}
          {status.kpvedTotal !== undefined && status.kpvedTotal > 0 && (
            <div className="space-y-2 pt-4 border-t">
              <div className="text-sm font-medium text-muted-foreground">Классификация КПВЭД</div>
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-muted-foreground">Классифицировано:</span>
                  <span className="font-medium">
                    {status.kpvedClassified?.toLocaleString() || 0} / {status.kpvedTotal.toLocaleString()}
                    {status.kpvedProgress !== undefined && ` (${status.kpvedProgress.toFixed(1)}%)`}
                  </span>
                </div>
                {status.kpvedProgress !== undefined && (
                  <Progress value={status.kpvedProgress} className="h-2" />
                )}
              </div>
            </div>
          )}

          {/* Информация о провайдерах */}
          {status.providers_used && status.providers_used.length > 0 && (
            <div className="space-y-3 pt-4 border-t">
              <div className="flex items-center justify-between">
                <div className="text-sm font-medium">Использованные провайдеры для нормализации</div>
                <Badge variant="secondary" className="text-xs">
                  {status.providers_used.length} {status.providers_used.length === 1 ? 'провайдер' : status.providers_used.length < 5 ? 'провайдера' : 'провайдеров'}
                </Badge>
              </div>
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                {status.providers_used.map((provider, idx) => {
                  const providerTypeLabels: Record<string, string> = {
                    'dadata': 'DaData',
                    'adata': 'Adata.kz',
                    'openrouter': 'OpenRouter',
                    'huggingface': 'Hugging Face',
                    'arliai': 'Arliai',
                    'edenai': 'Eden AI',
                  }
                  
                  const providerColors: Record<string, string> = {
                    'dadata': 'bg-red-100 text-red-800 border-red-200',
                    'adata': 'bg-pink-100 text-pink-800 border-pink-200',
                    'openrouter': 'bg-blue-100 text-blue-800 border-blue-200',
                    'huggingface': 'bg-green-100 text-green-800 border-green-200',
                    'arliai': 'bg-purple-100 text-purple-800 border-purple-200',
                    'edenai': 'bg-yellow-100 text-yellow-800 border-yellow-200',
                  }
                  
                  const colorClass = providerColors[provider.provider_type] || 'bg-gray-100 text-gray-800 border-gray-200'
                  
                  return (
                    <div key={idx} className="p-3 bg-card border rounded-lg hover:shadow-md transition-shadow">
                      <div className="flex items-center justify-between mb-2">
                        <Badge variant="outline" className={`text-xs ${colorClass}`}>
                          {providerTypeLabels[provider.provider_type] || provider.provider_type}
                        </Badge>
                        <span className={`text-xs font-medium ${
                          provider.success_rate >= 0.95 ? 'text-green-600' : 
                          provider.success_rate >= 0.8 ? 'text-yellow-600' : 
                          'text-red-600'
                        }`}>
                          {(provider.success_rate * 100).toFixed(1)}%
                        </span>
                      </div>
                      <div className="space-y-1 text-xs">
                        <div className="flex items-center justify-between text-muted-foreground">
                          <span>Запросов:</span>
                          <span className="font-medium text-foreground">{provider.requests_count.toLocaleString()}</span>
                        </div>
                        <div className="flex items-center justify-between text-muted-foreground">
                          <span>Успешность:</span>
                          <span className={`font-medium ${
                            provider.success_rate >= 0.95 ? 'text-green-600' : 
                            provider.success_rate >= 0.8 ? 'text-yellow-600' : 
                            'text-red-600'
                          }`}>
                            {provider.success_rate >= 0.95 ? 'Отлично' : 
                             provider.success_rate >= 0.8 ? 'Хорошо' : 
                             'Требует внимания'}
                          </span>
                        </div>
                      </div>
                    </div>
                  )
                })}
              </div>
              <p className="text-xs text-muted-foreground mt-2">
                Провайдеры использовались для нормализации данных в последней сессии. 
                {status.providers_used.some(p => p.provider_type === 'dadata' || p.provider_type === 'adata') && 
                  ' Использовались внешние сервисы (DaData, Adata.kz) для обогащения данных.'}
                {status.providers_used.some(p => p.provider_type === 'openrouter' || p.provider_type === 'huggingface' || p.provider_type === 'arliai' || p.provider_type === 'edenai') && 
                  ' Использовались генеративные AI модели для улучшения качества нормализации.'}
              </p>
            </div>
          )}

          {/* Кнопки управления */}
          <div className="flex items-center gap-2 pt-4 border-t">
            {!status.isRunning ? (
              <div className="flex-1 space-y-2">
                <Button
                  onClick={handleStart}
                  disabled={!!(isLoading || (!database && !project) || (project && projectDatabasesCount !== null && projectDatabasesCount === 0))}
                  className="w-full bg-linear-to-r from-primary to-primary/80 hover:from-primary/90 hover:to-primary/70 text-white font-semibold py-6 text-lg shadow-lg hover:shadow-xl transition-all duration-300 flex items-center justify-center gap-3"
                  title={project && projectDatabasesCount !== null && projectDatabasesCount === 0 ? 'В проекте нет активных баз данных' : undefined}
                  size="lg"
                >
                  <Zap className="h-5 w-5" />
                  {project && clientId && projectId 
                    ? selectedDatabaseIds.length > 0
                      ? normalizationType === 'counterparties'
                        ? `🚀 Запустить нормализацию контрагентов для ${selectedDatabaseIds.length} выбранных БД`
                        : normalizationType === 'nomenclature'
                        ? `🚀 Запустить нормализацию номенклатуры для ${selectedDatabaseIds.length} выбранных БД`
                        : `🚀 Запустить комплексную нормализацию для ${selectedDatabaseIds.length} выбранных БД`
                      : projectDatabasesCount !== null && projectDatabasesCount > 0
                        ? normalizationType === 'counterparties'
                          ? `🚀 Запустить нормализацию контрагентов для ${projectDatabasesCount} ${projectDatabasesCount === 1 ? 'БД' : 'БД'} проекта`
                          : normalizationType === 'nomenclature'
                          ? `🚀 Запустить нормализацию номенклатуры для ${projectDatabasesCount} ${projectDatabasesCount === 1 ? 'БД' : 'БД'} проекта`
                          : `🚀 Запустить комплексную нормализацию для ${projectDatabasesCount} ${projectDatabasesCount === 1 ? 'БД' : 'БД'} проекта`
                        : normalizationType === 'counterparties'
                          ? '🚀 Запустить нормализацию контрагентов для всех БД проекта'
                          : normalizationType === 'nomenclature'
                          ? '🚀 Запустить нормализацию номенклатуры для всех БД проекта'
                          : '🚀 Запустить комплексную нормализацию для всех БД проекта'
                    : '🚀 Запустить нормализацию'}
                </Button>
                {project && clientId && projectId && (
                  <div className="flex items-center gap-2 text-xs text-muted-foreground px-2">
                    <Clock className="h-3 w-3" />
                    <span>Примерное время выполнения: ~{estimateProcessingTimeForProject(selectedDatabaseIds.length > 0 ? selectedDatabaseIds.length : (projectDatabasesCount || 0))}</span>
                  </div>
                )}
              </div>
            ) : (
              <Button
                onClick={handleStop}
                disabled={isLoading}
                variant="destructive"
                className="flex items-center gap-2"
              >
                <Square className="h-4 w-4" />
                Остановить
              </Button>
            )}
          </div>
        </CardContent>
      </Card>

      {/* Панель классификации КПВЭД/ОКПД2 - отдельная секция */}
      {!status.isRunning && (normalizationType === 'nomenclature' || normalizationType === 'both') && projectType !== 'counterparty' && (
        <Card className="bg-linear-to-br from-purple-50/50 to-indigo-50/50 dark:from-purple-950/20 dark:to-indigo-950/20 border-purple-200/50 dark:border-purple-800/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Sparkles className="h-5 w-5 text-purple-600" />
              Панель классификации
            </CardTitle>
            <CardDescription>
              Дополнительные классификаторы для автоматической категоризации после нормализации
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="space-y-4">
              <div className="flex items-start space-x-3 p-4 bg-background/50 rounded-lg border border-purple-200/50 dark:border-purple-800/50">
                <Checkbox
                  id="use-kpved"
                  checked={useKpved}
                  onCheckedChange={(checked) => setUseKpved(checked === true)}
                  className="mt-1"
                />
                <div className="flex-1 space-y-1">
                  <Label
                    htmlFor="use-kpved"
                    className="text-sm font-semibold leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex items-center gap-2"
                  >
                    Классификация по КПВЭД
                    <Badge variant="outline" className="text-xs">AI</Badge>
                  </Label>
                  <p className="text-xs text-muted-foreground">
                    После нормализации выполнить автоматическую классификацию по классификатору КПВЭД с использованием AI
                  </p>
                </div>
              </div>
              
              <div className="flex items-start space-x-3 p-4 bg-background/50 rounded-lg border border-purple-200/50 dark:border-purple-800/50">
                <Checkbox
                  id="use-okpd2"
                  checked={useOkpd2}
                  onCheckedChange={(checked) => setUseOkpd2(checked === true)}
                  className="mt-1"
                />
                <div className="flex-1 space-y-1">
                  <Label
                    htmlFor="use-okpd2"
                    className="text-sm font-semibold leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex items-center gap-2"
                  >
                    Классификация по ОКПД2
                    <Badge variant="outline" className="text-xs">AI</Badge>
                  </Label>
                  <p className="text-xs text-muted-foreground">
                    После нормализации выполнить автоматическую классификацию по классификатору ОКПД2 с использованием AI
                  </p>
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Настройки для контрагентов */}
      {!status.isRunning && (normalizationType === 'counterparties' || normalizationType === 'both') && (
        <Card className="bg-linear-to-br from-green-50/50 to-green-100/50 dark:from-green-950/20 dark:to-green-900/20 border-green-200/50 dark:border-green-800/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Settings className="h-5 w-5 text-green-600" />
              Настройки нормализации контрагентов
            </CardTitle>
            <CardDescription>
              Дополнительные параметры обработки контрагентов
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex items-start space-x-3 p-4 bg-background/50 rounded-lg border border-green-200/50 dark:border-green-800/50">
              <Checkbox
                id="verify-requisites"
                checked={verifyRequisites}
                onCheckedChange={(checked) => setVerifyRequisites(checked === true)}
                className="mt-1"
              />
              <div className="flex-1 space-y-1">
                <Label
                  htmlFor="verify-requisites"
                  className="text-sm font-semibold leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex items-center gap-2"
                >
                  Верификация реквизитов через DaData/Adata.kz
                  <Badge variant="outline" className="text-xs">API</Badge>
                </Label>
                <p className="text-xs text-muted-foreground">
                  Проверка и дополнение реквизитов контрагентов через внешние сервисы
                </p>
              </div>
            </div>
            
            <div className="flex items-start space-x-3 p-4 bg-background/50 rounded-lg border border-green-200/50 dark:border-green-800/50">
              <Checkbox
                id="normalize-addresses"
                checked={normalizeAddresses}
                onCheckedChange={(checked) => setNormalizeAddresses(checked === true)}
                className="mt-1"
              />
              <div className="flex-1 space-y-1">
                <Label
                  htmlFor="normalize-addresses"
                  className="text-sm font-semibold leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex items-center gap-2"
                >
                  Нормализация адресов
                  <Badge variant="outline" className="text-xs">ФИАС</Badge>
                </Label>
                <p className="text-xs text-muted-foreground">
                  Стандартизация адресов по ФИАС/КЛАДР
                </p>
              </div>
            </div>

            <div className="flex items-start space-x-3 p-4 bg-background/50 rounded-lg border border-green-200/50 dark:border-green-800/50">
              <Checkbox
                id="standardize-legal-forms"
                checked={standardizeLegalForms}
                onCheckedChange={(checked) => setStandardizeLegalForms(checked === true)}
                className="mt-1"
              />
              <div className="flex-1 space-y-1">
                <Label
                  htmlFor="standardize-legal-forms"
                  className="text-sm font-semibold leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer flex items-center gap-2"
                >
                  Стандартизация юридических форм
                </Label>
                <p className="text-xs text-muted-foreground">
                  Приведение юридических форм к стандартному виду (ООО, ИП, АО и т.д.)
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Информация для контрагентов */}
      {!status.isRunning && projectType === 'counterparty' && (
        <Card className="bg-linear-to-br from-blue-50/50 to-cyan-50/50 dark:from-blue-950/20 dark:to-cyan-950/20 border-blue-200/50 dark:border-blue-800/50">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <Settings className="h-5 w-5 text-blue-600" />
              Процесс нормализации контрагентов
            </CardTitle>
            <CardDescription>
              Этапы обработки данных контрагентов
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm text-muted-foreground">
              <p>Нормализация контрагентов включает следующие этапы:</p>
              <ul className="list-disc list-inside space-y-1 ml-2">
                <li>Объединение дубликатов контрагентов из разных баз данных</li>
                <li>Обогащение данных из внешних источников (ИНН, БИН и т.д.)</li>
                <li>Проверка и улучшение качества данных</li>
                <li>Создание нормализованных записей с единой структурой</li>
                <li>Связывание с проектом и клиентом</li>
              </ul>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Запуск процесса - большая заметная кнопка */}
      {!status.isRunning && (
        <Card className="bg-linear-to-br from-primary/10 via-primary/5 to-background border-primary/30 shadow-lg">
          <CardContent className="pt-6">
            <div className="space-y-4">
              {/* Предупреждение о времени выполнения */}
              {project && clientId && projectId && (
                <Alert className="bg-orange-50/50 dark:bg-orange-950/20 border-orange-200 dark:border-orange-800">
                  <Clock className="h-4 w-4 text-orange-600" />
                  <AlertDescription className="text-sm">
                    <div className="space-y-1">
                      <p className="font-medium text-orange-900 dark:text-orange-100">
                        Оценка времени выполнения
                      </p>
                      <p className="text-orange-800 dark:text-orange-200">
                        Процесс нормализации может занять значительное время в зависимости от объема данных.
                        {projectDatabasesCount !== null && projectDatabasesCount > 0 && (
                          <> Будет обработано {projectDatabasesCount} {projectDatabasesCount === 1 ? 'база данных' : projectDatabasesCount < 5 ? 'базы данных' : 'баз данных'}.</>
                        )}
                      </p>
                    </div>
                  </AlertDescription>
                </Alert>
              )}
              
              {/* Кнопка запуска */}
              <Button
                onClick={handleStart}
                disabled={!!(isLoading || (!database && !project) || (project && projectDatabasesCount !== null && projectDatabasesCount === 0))}
                size="lg"
                className="w-full bg-linear-to-r from-primary to-primary/90 hover:from-primary/90 hover:to-primary text-white shadow-lg hover:shadow-xl transition-all duration-300 h-14 text-lg font-semibold"
                title={project && projectDatabasesCount !== null && projectDatabasesCount === 0 ? 'В проекте нет активных баз данных' : undefined}
              >
                <Play className="h-5 w-5 mr-2" />
                {project && clientId && projectId 
                  ? selectedDatabaseIds.length > 0
                    ? projectType === 'counterparty'
                      ? `🚀 Запустить нормализацию контрагентов для ${selectedDatabaseIds.length} выбранных БД`
                      : `🚀 Запустить нормализацию для ${selectedDatabaseIds.length} выбранных БД`
                    : projectDatabasesCount !== null && projectDatabasesCount > 0
                      ? projectType === 'counterparty'
                        ? `🚀 Запустить нормализацию контрагентов для ${projectDatabasesCount} ${projectDatabasesCount === 1 ? 'БД' : 'БД'} проекта`
                        : `🚀 Запустить нормализацию для ${projectDatabasesCount} ${projectDatabasesCount === 1 ? 'БД' : 'БД'} проекта`
                      : projectType === 'counterparty'
                        ? '🚀 Запустить нормализацию контрагентов для всех БД проекта'
                        : '🚀 Запустить нормализацию для всех БД проекта'
                  : '🚀 Запустить нормализацию'}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Кнопка остановки */}
      {status.isRunning && (
        <Card className="bg-linear-to-br from-red-50/50 to-orange-50/50 dark:from-red-950/20 dark:to-orange-950/20 border-red-200/50 dark:border-red-800/50">
          <CardContent className="pt-6">
            <Button
              onClick={handleStop}
              disabled={isLoading}
              variant="destructive"
              size="lg"
              className="w-full h-14 text-lg font-semibold shadow-lg"
            >
              <Square className="h-5 w-5 mr-2" />
              Остановить нормализацию
            </Button>
          </CardContent>
        </Card>
      )}

      {/* Preview результатов - показываем только если есть данные и процесс не запущен */}
      {!status.isRunning && project && clientId && projectId && projectType !== 'counterparty' && (
        <NormalizationPreviewResults
          clientId={clientId}
          projectId={projectId}
          isEnabled={true}
        />
      )}

      {/* Логи */}
      {status.logs && status.logs.length > 0 && (
        <LogsPanel
          logs={status.logs}
          title="Логи нормализации"
          description="Детальная информация о процессе нормализации"
        />
      )}

      {/* Результаты нормализации - показываем только после запуска или если есть данные */}
      {(status.isRunning || (status.processed > 0 && status.total > 0)) && (
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
        >
          <NormalizationResultsTable
            isRunning={status.isRunning}
            database={database}
            project={project}
            projectType={projectType}
            normalizationType={normalizationType}
            showOnlyWhenRunning={true}
          />
        </motion.div>
      )}
    </div>
  )
}
