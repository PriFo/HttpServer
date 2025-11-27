'use client'

import { useState, useEffect, useRef, useCallback } from 'react'
import { fetchNormalizationStats, fetchPipelineStatusApi } from '@/lib/mdm/api'
import type { NormalizationMetrics } from '@/types/normalization'

interface ProjectState<T> {
  data: T | null
  loading: boolean
  error: string | null
  lastUpdated: number | null
}

interface UseProjectStateOptions {
  refetchInterval?: number | null
  enabled?: boolean
  keepPreviousData?: boolean
}

export function useProjectState<T>(
  fetchFn: (clientId: string, projectId: string, signal?: AbortSignal) => Promise<T>,
  clientId: string | number | null,
  projectId: string | number | null,
  dependencies: any[] = [],
  options: UseProjectStateOptions = {}
): {
  data: T | null
  loading: boolean
  error: string | null
  lastUpdated: number | null
  refetch: () => void
} {
  const { refetchInterval = null, enabled = true, keepPreviousData = false } = options

  const [state, setState] = useState<ProjectState<T>>({
    data: null,
    loading: false,
    error: null,
    lastUpdated: null,
  })

  const currentProjectRef = useRef<string>('')
  const abortControllerRef = useRef<AbortController | null>(null)
  const intervalRef = useRef<NodeJS.Timeout | null>(null)

  const fetchData = useCallback(async () => {
    const clientIdStr = clientId?.toString() || ''
    const projectIdStr = projectId?.toString() || ''

    if (!enabled || !clientIdStr || !projectIdStr) {
      if (!keepPreviousData) {
        setState({ data: null, loading: false, error: null, lastUpdated: null })
        currentProjectRef.current = ''
      } else {
        setState(prev => ({ ...prev, loading: false }))
      }
      return
    }

    const currentProject = `${clientIdStr}:${projectIdStr}`

    // Если проект изменился, сбрасываем состояние (если не keepPreviousData)
    if (currentProjectRef.current !== currentProject && currentProjectRef.current !== '') {
      if (!keepPreviousData) {
        setState({ data: null, loading: false, error: null, lastUpdated: null })
      }
      currentProjectRef.current = currentProject
    } else if (currentProjectRef.current === '') {
      currentProjectRef.current = currentProject
    }

    // Отменяем предыдущий запрос
    if (abortControllerRef.current) {
      abortControllerRef.current.abort()
    }

    abortControllerRef.current = new AbortController()

    setState(prev => ({ ...prev, loading: true, error: null }))

    try {
      const data = await fetchFn(clientIdStr, projectIdStr, abortControllerRef.current.signal)

      // Проверяем, что данные все еще актуальны
      if (currentProjectRef.current === currentProject && !abortControllerRef.current.signal.aborted) {
        setState({
          data,
          loading: false,
          error: null,
          lastUpdated: Date.now(),
        })
      }
    } catch (error: any) {
      if (error.name === 'AbortError') {
        return // Игнорируем отмененные запросы
      }

      if (currentProjectRef.current === currentProject && !abortControllerRef.current.signal.aborted) {
        setState(prev => ({
          ...prev,
          loading: false,
          error: error.message || 'Failed to fetch data',
        }))
      }
    }
  }, [clientId, projectId, fetchFn, enabled, keepPreviousData, ...dependencies])

  // Сброс состояния при смене проекта
  useEffect(() => {
    const clientIdStr = clientId?.toString() || ''
    const projectIdStr = projectId?.toString() || ''
    const currentProject = clientIdStr && projectIdStr ? `${clientIdStr}:${projectIdStr}` : ''

    if (currentProjectRef.current !== currentProject && currentProjectRef.current !== '') {
      // Проект изменился - отменяем запросы и очищаем интервал
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [clientId, projectId])

  // Загрузка данных
  useEffect(() => {
    if (!enabled) {
      return
    }

    fetchData()

    return () => {
      if (abortControllerRef.current) {
        abortControllerRef.current.abort()
      }
    }
  }, [fetchData, enabled])

  // Автоматическое обновление
  useEffect(() => {
    if (!enabled || !refetchInterval) {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
      return
    }

    intervalRef.current = setInterval(() => {
      fetchData()
    }, refetchInterval)

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current)
        intervalRef.current = null
      }
    }
  }, [fetchData, refetchInterval, enabled])

  const refetch = useCallback(() => {
    fetchData()
  }, [fetchData])

  return { ...state, refetch }
}

/**
 * Специализированный хук для загрузки метрик нормализации
 * @param clientId - ID клиента
 * @param projectId - ID проекта
 * @param isProcessRunning - Запущен ли процесс нормализации
 * @returns Состояние с метриками нормализации
 */
export function useNormalizationMetrics(
  clientId: string | number | null,
  projectId: string | number | null,
  isProcessRunning: boolean = false
) {
  return useProjectState<NormalizationMetrics>(
    (cid, pid, signal) =>
      fetchNormalizationStats(
        cid,
        pid,
        { timeRange: '24h' },
        signal
      ),
    clientId,
    projectId,
    [],
    {
      refetchInterval: isProcessRunning ? 30000 : null, // Обновлять каждые 30 секунд если процесс запущен
      enabled: !!clientId && !!projectId,
    }
  )
}

/**
 * Специализированный хук для загрузки статуса пайплайна
 * @param clientId - ID клиента
 * @param projectId - ID проекта
 * @param activeProcess - Активный процесс (для определения необходимости автообновления)
 * @returns Состояние со статусом пайплайна
 */
export function usePipelineStatus(
  clientId: string | number | null,
  projectId: string | number | null,
  activeProcess?: string | null
) {
  return useProjectState(
    (cid, pid, signal) => fetchPipelineStatusApi(cid, pid, signal),
    clientId,
    projectId,
    [],
    {
      refetchInterval: activeProcess ? 10000 : null, // Обновлять каждые 10 секунд если процесс активен
      enabled: !!clientId && !!projectId,
    }
  )
}

/**
 * Специализированный хук для загрузки групп нормализации
 * @param clientId - ID клиента
 * @param projectId - ID проекта
 * @param page - Номер страницы
 * @param filters - Фильтры (search, category, sortKey, sortDirection)
 * @returns Состояние с группами нормализации
 */
export function useNormalizationGroups(
  clientId: string | number | null,
  projectId: string | number | null,
  page: number = 1,
  filters: {
    search?: string
    category?: string
    sortKey?: string | null
    sortDirection?: 'asc' | 'desc' | null
    limit?: number
  } = {}
) {
  return useProjectState(
    async (cid, pid, signal) => {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: (filters.limit || 10).toString(),
      })
      
      if (filters.search) {
        params.set('search', filters.search)
      }
      if (filters.category) {
        params.set('category', filters.category)
      }
      if (filters.sortKey && filters.sortDirection) {
        params.set('sortKey', filters.sortKey)
        params.set('sortDirection', filters.sortDirection)
      }
      
      const response = await fetch(
        `/api/clients/${cid}/projects/${pid}/normalization/groups?${params}`,
        { cache: 'no-store', signal }
      )
      if (!response.ok) {
        throw new Error(`Failed to fetch groups: ${response.status}`)
      }
      return response.json()
    },
    clientId,
    projectId,
    [page, filters.search, filters.category, filters.sortKey, filters.sortDirection, filters.limit],
    {
      enabled: !!clientId && !!projectId,
      // Не используем автообновление для групп, так как это может быть тяжелая операция
      refetchInterval: null,
    }
  )
}

/**
 * Специализированный хук для загрузки активности проекта
 * @param clientId - ID клиента
 * @param projectId - ID проекта
 * @param maxEvents - Максимальное количество событий
 * @returns Состояние с событиями активности
 */
export function useActivityEvents(
  clientId: string | number | null,
  projectId: string | number | null,
  maxEvents: number = 50,
  options: UseProjectStateOptions = {}
) {
  const {
    refetchInterval = 30000,
    enabled = !!clientId && !!projectId,
    keepPreviousData = true,
  } = options

  return useProjectState(
    async (cid, pid, signal) => {
      const response = await fetch(
        `/api/clients/${cid}/projects/${pid}/normalization/history?limit=${maxEvents}`,
        { cache: 'no-store', signal }
      )
      if (!response.ok) {
        if (response.status === 404) {
          return { events: [] }
        }
        throw new Error(`Failed to fetch activity events: ${response.status}`)
      }
      const data = await response.json()
      return {
        events: (data.history || []).map((entry: any) => ({
          id: entry.id,
          timestamp: entry.timestamp,
          user: entry.user || 'system',
          action: entry.action,
          type: entry.action === 'delete' ? 'error' as const :
                entry.action === 'create' ? 'success' as const :
                entry.action === 'update' ? 'info' as const : 'warning' as const,
          description: entry.description || `${entry.action} ${entry.entity_type}`,
          details: entry.changes,
        })),
      }
    },
    clientId,
    projectId,
    [maxEvents],
    {
      refetchInterval,
      enabled,
      keepPreviousData,
    }
  )
}

/**
 * Простой хук для загрузки базовых данных проекта (обратная совместимость)
 * @param clientId - ID клиента
 * @param projectId - ID проекта
 * @returns Состояние проекта с данными, загрузкой и ошибками
 */
export const useProjectStateSimple = (clientId: string | number | null, projectId: string | number | null) => {
  return useProjectState(
    async (cid, pid, signal) => {
      const response = await fetch(`/api/clients/${cid}/projects/${pid}`, {
        cache: 'no-store',
        signal,
      })
      if (!response.ok) {
        throw new Error(`Failed to fetch project: ${response.status}`)
      }
      return response.json()
    },
    clientId,
    projectId
  )
}
