'use client'

import { useRef, useCallback } from 'react'

interface CacheEntry<T> {
  data: T
  timestamp: number
  projectKey: string
}

interface UseProjectCacheOptions {
  ttl?: number // Time to live в миллисекундах
}

/**
 * Хук для управления кэшем данных проекта
 * Автоматически очищает кэш при смене проекта
 */
export function useProjectCache<T>(
  clientId: string | number | null,
  projectId: string | number | null,
  options: UseProjectCacheOptions = {}
) {
  const { ttl = 5 * 60 * 1000 } = options // По умолчанию 5 минут
  const cacheRef = useRef<Map<string, CacheEntry<T>>>(new Map())
  const currentProjectRef = useRef<string | null>(null)

  const projectKey = clientId && projectId ? `${clientId}:${projectId}` : null

  // Очистка кэша при смене проекта
  if (currentProjectRef.current !== projectKey) {
    cacheRef.current.clear()
    currentProjectRef.current = projectKey
  }

  const get = useCallback(
    (key: string): T | null => {
      const entry = cacheRef.current.get(key)
      if (!entry) return null

      // Проверяем, что данные для текущего проекта
      if (entry.projectKey !== projectKey) {
        cacheRef.current.delete(key)
        return null
      }

      // Проверяем TTL
      const now = Date.now()
      if (now - entry.timestamp > ttl) {
        cacheRef.current.delete(key)
        return null
      }

      return entry.data
    },
    [projectKey, ttl]
  )

  const set = useCallback(
    (key: string, data: T) => {
      if (!projectKey) return

      cacheRef.current.set(key, {
        data,
        timestamp: Date.now(),
        projectKey,
      })
    },
    [projectKey]
  )

  const invalidate = useCallback((key?: string) => {
    if (key) {
      cacheRef.current.delete(key)
    } else {
      cacheRef.current.clear()
    }
  }, [])

  const clear = useCallback(() => {
    cacheRef.current.clear()
  }, [])

  return {
    get,
    set,
    invalidate,
    clear,
  }
}

