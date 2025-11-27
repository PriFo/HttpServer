/**
 * Утилиты для работы с нормализацией данных
 * 
 * Примечание: Эти утилиты дополняют функции из @/utils/normalization-helpers.ts
 * и фокусируются на форматировании и вычислениях для UI компонентов.
 */

/**
 * Форматирует число для отображения
 */
export function formatNumber(num: number): string {
  return new Intl.NumberFormat('ru-RU').format(num)
}

/**
 * Форматирует процентное значение
 */
export function formatPercent(value: number, decimals: number = 1): string {
  return `${(value * 100).toFixed(decimals)}%`
}

/**
 * Форматирует время в секундах в читаемый формат
 */
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${Math.round(seconds)}с`
  }
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.round(seconds % 60)
  if (minutes < 60) {
    return `${minutes}м ${remainingSeconds}с`
  }
  const hours = Math.floor(minutes / 60)
  const remainingMinutes = minutes % 60
  return `${hours}ч ${remainingMinutes}м`
}

/**
 * Вычисляет процент прогресса
 */
export function calculateProgress(processed: number, total: number): number {
  if (total === 0) return 0
  return Math.min(100, Math.round((processed / total) * 100))
}

/**
 * Вычисляет скорость обработки (записей в секунду)
 */
export function calculateSpeed(processed: number, durationSeconds: number): number {
  if (durationSeconds === 0) return 0
  return processed / durationSeconds
}

/**
 * Вычисляет оставшееся время
 */
export function calculateRemainingTime(total: number, processed: number, speed: number): number {
  if (speed === 0) return 0
  return Math.round((total - processed) / speed)
}

/**
 * Получает цвет для индикатора качества
 */
export function getQualityColor(score: number): string {
  if (score >= 0.9) return 'text-green-600'
  if (score >= 0.7) return 'text-yellow-600'
  return 'text-red-600'
}

/**
 * Получает вариант badge для качества
 */
export function getQualityVariant(score: number): 'default' | 'secondary' | 'destructive' {
  if (score >= 0.9) return 'default'
  if (score >= 0.7) return 'secondary'
  return 'destructive'
}

/**
 * Парсит проект из строки формата "clientId:projectId"
 */
export function parseProject(project: string | null | undefined): { clientId: number | null; projectId: number | null } {
  if (!project) {
    return { clientId: null, projectId: null }
  }
  
  const parts = project.split(':')
  if (parts.length !== 2) {
    return { clientId: null, projectId: null }
  }
  
  const clientId = parseInt(parts[0], 10)
  const projectId = parseInt(parts[1], 10)
  
  if (isNaN(clientId) || isNaN(projectId)) {
    return { clientId: null, projectId: null }
  }
  
  return { clientId, projectId }
}

/**
 * Форматирует проект в строку "clientId:projectId"
 */
export function formatProject(clientId: number | string, projectId: number | string): string {
  return `${clientId}:${projectId}`
}

/**
 * Создает ключ проекта для отслеживания изменений
 */
export function createProjectKey(
  project: string | null | undefined,
  database: string | null | undefined,
  clientId: number | string | null,
  projectId: number | string | null
): string {
  if (project) return project
  if (database) return `db:${database}`
  if (clientId && projectId) return formatProject(clientId, projectId)
  return 'none'
}

/**
 * Проверяет, изменился ли проект
 */
export function hasProjectChanged(
  currentKey: string | null,
  newKey: string
): boolean {
  return currentKey !== null && currentKey !== newKey
}

/**
 * Дебаунс функция
 */
export function debounce<T extends (...args: any[]) => any>(
  func: T,
  wait: number
): (...args: Parameters<T>) => void {
  let timeout: NodeJS.Timeout | null = null
  
  return function executedFunction(...args: Parameters<T>) {
    const later = () => {
      timeout = null
      func(...args)
    }
    
    if (timeout) {
      clearTimeout(timeout)
    }
    timeout = setTimeout(later, wait)
  }
}

/**
 * Форматирует размер файла
 */
export function formatFileSize(bytes: number): string {
  if (bytes === 0) return '0 Б'
  
  const k = 1024
  const sizes = ['Б', 'КБ', 'МБ', 'ГБ', 'ТБ']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

/**
 * Форматирует дату для отображения
 */
export function formatDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  return new Intl.DateTimeFormat('ru-RU', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(d)
}

/**
 * Форматирует относительное время
 */
export function formatRelativeTime(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const diffSec = Math.floor(diffMs / 1000)
  const diffMin = Math.floor(diffSec / 60)
  const diffHour = Math.floor(diffMin / 60)
  const diffDay = Math.floor(diffHour / 24)
  
  if (diffSec < 60) return 'только что'
  if (diffMin < 60) return `${diffMin} мин назад`
  if (diffHour < 24) return `${diffHour} ч назад`
  if (diffDay < 7) return `${diffDay} дн назад`
  
  return formatDate(d)
}

