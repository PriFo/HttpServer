/**
 * Утилита для получения конфигурации API
 * Унифицирует получение BACKEND_URL из переменных окружения
 */

const DEFAULT_LOCAL_BACKEND = 'http://localhost:9999'
const DEFAULT_HOST_BACKEND = 'http://host.docker.internal:9999'

function resolveDefaultBackendUrl(): string {
  const isWSL =
    typeof process !== 'undefined' &&
    process.platform === 'linux' &&
    !!process.env.WSL_DISTRO_NAME

  // В WSL2 сервисы, запущенные на Windows, недоступны по 127.0.0.1 из Linux,
  // поэтому используем host.docker.internal, который указывает на Windows-хост
  if (isWSL) {
    return DEFAULT_HOST_BACKEND
  }

  return DEFAULT_LOCAL_BACKEND
}

export function getBackendUrl(): string {
  const defaultUrl = resolveDefaultBackendUrl()

  // На клиенте доступны только NEXT_PUBLIC_* переменные
  if (typeof window !== 'undefined') {
    return process.env.NEXT_PUBLIC_BACKEND_URL || defaultUrl
  }

  // На сервере (Next.js API routes) доступны все переменные окружения
  return (
    process.env.BACKEND_URL ||
    process.env.NEXT_PUBLIC_BACKEND_URL ||
    defaultUrl
  )
}

/**
 * Создает полный URL для API эндпоинта
 */
export function getApiUrl(endpoint: string): string {
  const baseUrl = getBackendUrl()
  // Убираем ведущий слэш, если есть
  const cleanEndpoint = endpoint.startsWith('/') ? endpoint.slice(1) : endpoint
  return `${baseUrl}/${cleanEndpoint}`
}

/**
 * Проверяет, доступен ли бэкенд
 * Работает только на клиенте
 */
export async function checkBackendHealth(): Promise<boolean> {
  // Проверяем, что мы на клиенте
  if (typeof window === 'undefined') {
    return false
  }
  
  try {
    const controller = new AbortController()
    const timeoutId = setTimeout(() => controller.abort(), 5000) // 5 секунд таймаут
    
    // Use Next.js API route instead of direct backend access
    const response = await fetch('/api/health', {
      method: 'GET',
      cache: 'no-store',
      signal: controller.signal,
    })
    
    clearTimeout(timeoutId)
    return response.ok
  } catch {
    return false
  }
}

