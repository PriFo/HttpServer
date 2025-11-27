'use client'

import { useSearchParams, useRouter, usePathname } from 'next/navigation'
import { useEffect, useCallback, useMemo, useRef } from 'react'

interface UseProjectSearchParamsOptions {
  resetOnProjectChange?: boolean
}

export function useProjectSearchParams(
  clientId: string | number | null,
  projectId: string | number | null,
  options: UseProjectSearchParamsOptions = { resetOnProjectChange: true }
) {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()
  const { resetOnProjectChange = true } = options

  // Отслеживаем предыдущий проект для сброса параметров
  const prevProjectRef = useRef<string | null>(null)

  // Сброс параметров при смене проекта
  useEffect(() => {
    if (!resetOnProjectChange) return

    const currentProject = clientId && projectId ? `${clientId}:${projectId}` : null

    // Если проект изменился, сбрасываем параметры
    if (prevProjectRef.current !== null && prevProjectRef.current !== currentProject && currentProject) {
      // Проект изменился - сбрасываем параметры
      const params = new URLSearchParams()
      router.replace(`${pathname}?${params.toString()}`)
    }

    // Обновляем ref для следующего рендера
    prevProjectRef.current = currentProject
  }, [clientId, projectId, resetOnProjectChange, router, pathname])

  const setParam = useCallback(
    (key: string, value: string | number | null) => {
      const params = new URLSearchParams(searchParams.toString())
      if (value === null || value === '') {
        params.delete(key)
      } else {
        params.set(key, value.toString())
      }
      router.replace(`${pathname}?${params.toString()}`)
    },
    [searchParams, router, pathname]
  )

  const setParams = useCallback(
    (updates: Record<string, string | number | null>) => {
      const params = new URLSearchParams(searchParams.toString())
      Object.entries(updates).forEach(([key, value]) => {
        if (value === null || value === '') {
          params.delete(key)
        } else {
          params.set(key, value.toString())
        }
      })
      router.replace(`${pathname}?${params.toString()}`)
    },
    [searchParams, router, pathname]
  )

  const setMultipleParams = useCallback(
    (updates: Record<string, string | number | null>) => {
      setParams(updates)
    },
    [setParams]
  )

  const getParam = useCallback(
    (key: string, defaultValue: string = '') => {
      return searchParams.get(key) || defaultValue
    },
    [searchParams]
  )

  const resetParams = useCallback(() => {
    router.replace(pathname || '')
  }, [router, pathname])

  const params = useMemo(() => {
    return {
      get: getParam,
      getAll: (key: string) => searchParams.getAll(key),
      has: (key: string) => searchParams.has(key),
      toString: () => searchParams.toString(),
    }
  }, [searchParams, getParam])

  return {
    params,
    setParam,
    setParams,
    setMultipleParams,
    getParam,
    resetParams,
  }
}
