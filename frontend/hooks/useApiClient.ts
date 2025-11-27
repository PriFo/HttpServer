/**
 * Хук для упрощения использования API-клиента с автоматической обработкой ошибок
 */

import { useCallback } from 'react'
import { useError } from '@/contexts/ErrorContext'
import { apiClient, apiGet, apiPost, apiPut, apiDelete, ApiClientOptions } from '@/lib/api-client'
import { AppError } from '@/lib/errors'

/**
 * Хук для работы с API
 * Автоматически обрабатывает ошибки через ErrorContext
 * 
 * @example
 * ```tsx
 * const { get, post } = useApiClient()
 * 
 * try {
 *   const data = await get('/api/users')
 * } catch (error) {
 *   // Ошибка уже обработана через ErrorContext
 * }
 * ```
 */
export function useApiClient() {
  const { handleError } = useError()

  const request = useCallback(async (
    url: string,
    options?: ApiClientOptions
  ) => {
    try {
      return await apiClient(url, {
        ...options,
        onError: options?.onError || ((error) => {
          if (!options?.skipErrorHandler) {
            handleError(error)
          }
        }),
      })
    } catch (error) {
      if (!options?.skipErrorHandler) {
        handleError(error)
      }
      throw error
    }
  }, [handleError])

  const get = useCallback(async <T = unknown>(
    url: string,
    options?: ApiClientOptions
  ): Promise<T> => {
    try {
      return await apiGet<T>(url, {
        ...options,
        onError: options?.onError || ((error) => {
          if (!options?.skipErrorHandler) {
            handleError(error)
          }
        }),
      })
    } catch (error) {
      if (!options?.skipErrorHandler) {
        handleError(error)
      }
      throw error
    }
  }, [handleError])

  const post = useCallback(async <T = unknown>(
    url: string,
    data?: unknown,
    options?: ApiClientOptions
  ): Promise<T> => {
    try {
      return await apiPost<T>(url, data, {
        ...options,
        onError: options?.onError || ((error) => {
          if (!options?.skipErrorHandler) {
            handleError(error)
          }
        }),
      })
    } catch (error) {
      if (!options?.skipErrorHandler) {
        handleError(error)
      }
      throw error
    }
  }, [handleError])

  const put = useCallback(async <T = unknown>(
    url: string,
    data?: unknown,
    options?: ApiClientOptions
  ): Promise<T> => {
    try {
      return await apiPut<T>(url, data, {
        ...options,
        onError: options?.onError || ((error) => {
          if (!options?.skipErrorHandler) {
            handleError(error)
          }
        }),
      })
    } catch (error) {
      if (!options?.skipErrorHandler) {
        handleError(error)
      }
      throw error
    }
  }, [handleError])

  const del = useCallback(async <T = unknown>(
    url: string,
    options?: ApiClientOptions
  ): Promise<T> => {
    try {
      return await apiDelete<T>(url, {
        ...options,
        onError: options?.onError || ((error) => {
          if (!options?.skipErrorHandler) {
            handleError(error)
          }
        }),
      })
    } catch (error) {
      if (!options?.skipErrorHandler) {
        handleError(error)
      }
      throw error
    }
  }, [handleError])

  return {
    request,
    get,
    post,
    put,
    delete: del,
  }
}

