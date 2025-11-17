/**
 * Утилиты для работы с API
 */

export interface ApiError {
  message: string
  status?: number
  code?: string
}

/**
 * Обрабатывает ответ от API и извлекает сообщение об ошибке
 */
export async function handleApiError(response: Response): Promise<string> {
  let errorMessage = `Ошибка ${response.status}: ${response.statusText || 'Неизвестная ошибка'}`

  try {
    const contentType = response.headers.get('content-type')
    
    if (contentType && contentType.includes('application/json')) {
      const errorData = await response.json()
      errorMessage = errorData.error || errorData.message || errorMessage
    } else {
      const errorText = await response.text()
      if (errorText) {
        try {
          const errorJson = JSON.parse(errorText)
          errorMessage = errorJson.error || errorJson.message || errorText
        } catch {
          errorMessage = errorText || errorMessage
        }
      }
    }
  } catch (err) {
    console.error('Error parsing error response:', err)
  }

  return errorMessage
}

/**
 * Выполняет запрос к API с обработкой ошибок
 */
export async function apiRequest<T>(
  url: string,
  options?: RequestInit
): Promise<T> {
  try {
    const response = await fetch(url, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...options?.headers,
      },
    })

    if (!response.ok) {
      const errorMessage = await handleApiError(response)
      throw new Error(errorMessage)
    }

    const data = await response.json()
    return data as T
  } catch (error) {
    if (error instanceof Error) {
      throw error
    }
    throw new Error('Неизвестная ошибка при выполнении запроса')
  }
}

/**
 * Создает функцию для обработки ошибок в try-catch блоках
 */
export function createErrorHandler(
  setError: (error: string | null) => void,
  setLoading?: (loading: boolean) => void
) {
  return (error: unknown) => {
    const errorMessage = error instanceof Error 
      ? error.message 
      : 'Произошла неизвестная ошибка'
    
    console.error('API Error:', error)
    setError(errorMessage)
    
    if (setLoading) {
      setLoading(false)
    }
  }
}

/**
 * Форматирует ошибку для отображения пользователю
 */
export function formatError(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }
  
  if (typeof error === 'string') {
    return error
  }
  
  return 'Произошла неизвестная ошибка'
}

