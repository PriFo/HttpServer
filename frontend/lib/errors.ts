// Централизованные error messages для consistency

export const ERROR_MESSAGES = {
  // Network errors
  NETWORK_ERROR: 'Не удалось выполнить запрос. Проверьте подключение к сети.',
  NETWORK_TIMEOUT: 'Время ожидания истекло. Проверьте подключение к сети и попробуйте позже.',
  SERVER_ERROR: 'Ошибка сервера. Попробуйте позже.',

  // Data loading errors
  LOAD_GROUPS_ERROR: 'Не удалось загрузить группы. Попробуйте еще раз.',
  LOAD_DETAILS_ERROR: 'Не удалось загрузить детали группы. Попробуйте еще раз.',
  LOAD_STATS_ERROR: 'Не удалось загрузить статистику.',
  LOAD_KPVED_ERROR: 'Не удалось загрузить данные КПВЭД. Проверьте подключение к сети.',

  // Navigation errors
  NAVIGATION_ERROR: 'Не удалось перейти к детальной странице. Попробуйте еще раз.',
  URL_TOO_LONG: 'URL слишком длинный, возможны проблемы в некоторых браузерах.',

  // Export errors
  EXPORT_ERROR: 'Не удалось экспортировать данные. Проверьте подключение к сети и попробуйте позже.',

  // Search errors
  SEARCH_ERROR: 'Не удалось выполнить поиск. Попробуйте еще раз.',

  // Generic
  UNKNOWN_ERROR: 'Произошла неизвестная ошибка. Попробуйте еще раз.',
  TRY_AGAIN: 'Попробуйте еще раз позже.',
} as const

export type ErrorMessageKey = keyof typeof ERROR_MESSAGES

export function getErrorMessage(key: ErrorMessageKey, customMessage?: string): string {
  return customMessage || ERROR_MESSAGES[key]
}

export function handleApiError(error: unknown, fallbackKey: ErrorMessageKey = 'UNKNOWN_ERROR'): string {
  if (error instanceof Error) {
    // Check for specific error types
    if (error.message.includes('NetworkError') || error.message.includes('Failed to fetch')) {
      return ERROR_MESSAGES.NETWORK_ERROR
    }
    if (error.message.includes('timeout')) {
      return ERROR_MESSAGES.NETWORK_TIMEOUT
    }
    if (error.message.includes('500') || error.message.includes('502') || error.message.includes('503')) {
      return ERROR_MESSAGES.SERVER_ERROR
    }
  }

  return ERROR_MESSAGES[fallbackKey]
}
