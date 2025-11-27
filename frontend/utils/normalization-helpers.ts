/**
 * Утилиты для работы с нормализацией НСИ
 */

/**
 * Определяет уровень уверенности на основе числового значения
 * @param confidence - Значение уверенности (0-100 или 0-1)
 * @returns Уровень уверенности: 'high', 'medium', 'low'
 */
export const getConfidenceLevel = (confidence: number): 'high' | 'medium' | 'low' => {
  // Нормализуем значение к диапазону 0-100
  const normalizedConfidence = confidence > 1 ? confidence : confidence * 100

  if (normalizedConfidence >= 90) return 'high'
  if (normalizedConfidence >= 70) return 'medium'
  return 'low'
}

/**
 * Получает локализованное название категории атрибута
 * @param category - Код категории
 * @returns Локализованное название категории
 */
export const getCategoryTitle = (category: string): string => {
  const titles: Record<string, string> = {
    'general': 'Основные реквизиты',
    'technical': 'Технические характеристики',
    'commercial': 'Коммерческие данные',
    'classification': 'Классификация',
    'dimensions': 'Габариты',
    'weight': 'Вес',
    'material': 'Материал',
    'color': 'Цвет',
    'brand': 'Бренд',
    'model': 'Модель',
    'manufacturer': 'Производитель',
    'country': 'Страна производства',
    'certificate': 'Сертификаты',
    'warranty': 'Гарантия',
    'price': 'Цена',
    'currency': 'Валюта',
  }

  return titles[category] || category
}

/**
 * Форматирует значение атрибута для отображения
 * @param value - Значение атрибута
 * @returns Отформатированное значение
 */
export const formatAttributeValue = (value: any): string => {
  if (value === null || value === undefined) {
    return '—'
  }

  if (typeof value === 'boolean') {
    return value ? 'Да' : 'Нет'
  }

  if (typeof value === 'number') {
    // Форматируем большие числа с разделителями
    if (value >= 1000) {
      return value.toLocaleString('ru-RU')
    }
    return value.toString()
  }

  if (typeof value === 'string') {
    // Обрезаем слишком длинные строки
    if (value.length > 100) {
      return value.substring(0, 100) + '...'
    }
    return value
  }

  if (Array.isArray(value)) {
    return value.join(', ')
  }

  if (typeof value === 'object') {
    return JSON.stringify(value, null, 2)
  }

  return String(value)
}

/**
 * Вычисляет схожесть между двумя строками (упрощенный алгоритм)
 * @param a - Первая строка
 * @param b - Вторая строка
 * @returns Коэффициент схожести от 0 до 1
 */
export const calculateSimilarity = (a: string, b: string): number => {
  if (!a || !b) return 0
  if (a === b) return 1

  // Нормализуем строки
  const normalize = (str: string) => str.toLowerCase().trim().replace(/\s+/g, ' ')
  const normalizedA = normalize(a)
  const normalizedB = normalize(b)

  if (normalizedA === normalizedB) return 1

  // Простой алгоритм схожести на основе общих подстрок
  const longer = normalizedA.length > normalizedB.length ? normalizedA : normalizedB
  const shorter = normalizedA.length > normalizedB.length ? normalizedB : normalizedA

  if (longer.length === 0) return 1

  // Проверяем, содержит ли длинная строка короткую
  if (longer.includes(shorter)) {
    return shorter.length / longer.length
  }

  // Вычисляем количество общих символов
  let matches = 0
  const shorterChars = shorter.split('')
  const longerChars = longer.split('')

  shorterChars.forEach(char => {
    const index = longerChars.indexOf(char)
    if (index !== -1) {
      matches++
      longerChars.splice(index, 1)
    }
  })

  return matches / longer.length
}

/**
 * Группирует атрибуты по категориям
 * @param attributes - Массив атрибутов
 * @returns Объект с атрибутами, сгруппированными по категориям
 */
export const groupAttributesByCategory = (attributes: any[]): Record<string, any[]> => {
  return attributes.reduce((groups, attr) => {
    const category = attr.category || attr.attribute_type || 'general'
    if (!groups[category]) {
      groups[category] = []
    }
    groups[category].push(attr)
    return groups
  }, {} as Record<string, any[]>)
}

/**
 * Получает цвет для индикатора уверенности
 * @param confidence - Значение уверенности
 * @returns CSS класс для цвета
 */
export const getConfidenceColor = (confidence: number): string => {
  const level = getConfidenceLevel(confidence)
  switch (level) {
    case 'high':
      return 'text-green-600 dark:text-green-400'
    case 'medium':
      return 'text-yellow-600 dark:text-yellow-400'
    case 'low':
      return 'text-red-600 dark:text-red-400'
    default:
      return 'text-gray-600 dark:text-gray-400'
  }
}

/**
 * Получает цвет фона для бейджа уверенности
 * @param confidence - Значение уверенности
 * @returns CSS класс для фона
 */
export const getConfidenceBgColor = (confidence: number): string => {
  const level = getConfidenceLevel(confidence)
  switch (level) {
    case 'high':
      return 'bg-green-100 dark:bg-green-900/30'
    case 'medium':
      return 'bg-yellow-100 dark:bg-yellow-900/30'
    case 'low':
      return 'bg-red-100 dark:bg-red-900/30'
    default:
      return 'bg-gray-100 dark:bg-gray-900/30'
  }
}

/**
 * Получает общий статус пайплайна на основе статусов всех этапов
 * @param stages - Массив этапов пайплайна
 * @returns Общий статус: 'completed', 'processing', 'error', 'pending'
 */
export function getOverallStatus(stages: Array<{ status: string }>): string {
  if (!stages || stages.length === 0) {
    return 'pending'
  }

  if (stages.every(stage => stage.status === 'completed')) {
    return 'completed'
  }

  if (stages.some(stage => stage.status === 'processing' || stage.status === 'active')) {
    return 'processing'
  }

  if (stages.some(stage => stage.status === 'error' || stage.status === 'failed')) {
    return 'error'
  }

  return 'pending'
}

/**
 * Получает локализованный текст статуса
 * @param status - Статус этапа или процесса
 * @returns Локализованный текст статуса
 */
export function getStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    pending: 'Ожидание',
    processing: 'В процессе',
    active: 'В работе',
    completed: 'Завершено',
    finished: 'Завершено',
    error: 'Ошибка',
    failed: 'Ошибка',
    cancelled: 'Отменено',
    stopped: 'Остановлено',
  }

  return statusMap[status] || status
}

/**
 * Получает цвет для статуса
 * @param status - Статус этапа или процесса
 * @returns CSS класс для цвета
 */
export function getStatusColor(status: string): string {
  const colorMap: Record<string, string> = {
    pending: 'text-muted-foreground',
    processing: 'text-blue-600 dark:text-blue-400',
    active: 'text-blue-600 dark:text-blue-400',
    completed: 'text-green-600 dark:text-green-400',
    finished: 'text-green-600 dark:text-green-400',
    error: 'text-red-600 dark:text-red-400',
    failed: 'text-red-600 dark:text-red-400',
    cancelled: 'text-yellow-600 dark:text-yellow-400',
    stopped: 'text-yellow-600 dark:text-yellow-400',
  }

  return colorMap[status] || 'text-muted-foreground'
}

/**
 * Получает вариант badge для статуса
 * @param status - Статус этапа или процесса
 * @returns Вариант badge
 */
export function getStatusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  const variantMap: Record<string, 'default' | 'secondary' | 'destructive' | 'outline'> = {
    pending: 'outline',
    processing: 'default',
    active: 'default',
    completed: 'default',
    finished: 'default',
    error: 'destructive',
    failed: 'destructive',
    cancelled: 'secondary',
    stopped: 'secondary',
  }

  return variantMap[status] || 'outline'
}

/**
 * Форматирует число с разделителями тысяч
 * @param value - Число для форматирования
 * @param decimals - Количество знаков после запятой (по умолчанию 0)
 * @returns Отформатированная строка
 */
export function formatNumber(value: number, decimals: number = 0): string {
  if (isNaN(value) || !isFinite(value)) {
    return '—'
  }
  return value.toLocaleString('ru-RU', {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })
}

/**
 * Форматирует процентное значение
 * @param value - Значение от 0 до 1 или от 0 до 100
 * @param decimals - Количество знаков после запятой (по умолчанию 1)
 * @returns Отформатированная строка с символом %
 */
export function formatPercent(value: number, decimals: number = 1): string {
  if (isNaN(value) || !isFinite(value)) {
    return '—'
  }
  // Нормализуем значение к диапазону 0-100
  const normalizedValue = value > 1 ? value : value * 100
  return `${normalizedValue.toFixed(decimals)}%`
}

/**
 * Форматирует длительность в секундах в читаемый формат
 * @param seconds - Количество секунд
 * @returns Отформатированная строка (например, "1ч 30м" или "45с")
 */
export function formatDuration(seconds: number): string {
  if (isNaN(seconds) || !isFinite(seconds) || seconds < 0) {
    return '—'
  }

  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  const parts: string[] = []
  if (hours > 0) {
    parts.push(`${hours}ч`)
  }
  if (minutes > 0) {
    parts.push(`${minutes}м`)
  }
  if (secs > 0 && hours === 0) {
    // Показываем секунды только если нет часов
    parts.push(`${secs}с`)
  }

  return parts.length > 0 ? parts.join(' ') : '0с'
}

/**
 * Вычисляет прогресс в процентах
 * @param current - Текущее значение
 * @param total - Общее значение
 * @returns Прогресс от 0 до 100
 */
export function calculateProgress(current: number, total: number): number {
  if (!total || total === 0) {
    return 0
  }
  return Math.min(100, Math.max(0, (current / total) * 100))
}

/**
 * Вычисляет скорость обработки (записей в секунду)
 * @param processed - Количество обработанных записей
 * @param elapsedSeconds - Прошедшее время в секундах
 * @returns Скорость обработки
 */
export function calculateSpeed(processed: number, elapsedSeconds: number): number {
  if (!elapsedSeconds || elapsedSeconds === 0) {
    return 0
  }
  return processed / elapsedSeconds
}

/**
 * Вычисляет оставшееся время обработки
 * @param remaining - Оставшееся количество записей
 * @param speed - Скорость обработки (записей/сек)
 * @returns Оставшееся время в секундах
 */
export function calculateRemainingTime(remaining: number, speed: number): number {
  if (!speed || speed === 0) {
    return 0
  }
  return Math.ceil(remaining / speed)
}

