/**
 * Утилиты для локализации с поддержкой России и Казахстана
 */

export type Locale = 'ru-RU' | 'kz-KZ'

/**
 * Определяет локаль на основе ИНН/БИН или других признаков
 * БИН (Казахстан) - всегда 12 цифр
 * ИНН (Россия) - 10 или 12 цифр
 */
export function detectLocale(taxId?: string): Locale {
  if (!taxId) {
    // По умолчанию используем ru-RU, но можно настроить через настройки пользователя
    return 'ru-RU'
  }
  
  const cleaned = taxId.replace(/\s/g, '')
  // Если ровно 12 цифр, это может быть БИН (Казахстан)
  // Для точного определения нужна проверка контрольной суммы на бэкенде
  // Пока используем эвристику: если 12 цифр и начинается с определенных префиксов - Казахстан
  if (cleaned.length === 12) {
    // БИН Казахстана обычно начинается с определенных цифр
    // Но для простоты считаем, что если пользователь ввел 12 цифр, это может быть БИН
    // Точная валидация будет на бэкенде
    return 'kz-KZ'
  }
  
  return 'ru-RU'
}

/**
 * Форматирует дату с учетом локали
 */
export function formatDate(
  date: Date | string | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locale?: Locale
): string {
  if (!date || date === '') {
    return 'Не указано'
  }
  
  const dateObj = typeof date === 'string' ? new Date(date) : date
  
  // Проверяем, что dateObj существует и является валидной датой
  if (!dateObj || !(dateObj instanceof Date) || isNaN(dateObj.getTime())) {
    return 'Неверная дата'
  }
  
  const loc = locale || 'ru-RU'
  
  return dateObj.toLocaleDateString(loc, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    ...options,
  })
}

/**
 * Форматирует дату и время с учетом локали
 */
export function formatDateTime(
  date: Date | string | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locale?: Locale
): string {
  if (!date || date === '') {
    return 'Не указано'
  }
  
  const dateObj = typeof date === 'string' ? new Date(date) : date
  
  // Проверяем, что dateObj существует и является валидной датой
  if (!dateObj || !(dateObj instanceof Date) || isNaN(dateObj.getTime())) {
    return 'Неверная дата'
  }
  
  const loc = locale || 'ru-RU'
  
  return dateObj.toLocaleString(loc, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  })
}

/**
 * Форматирует время с учетом локали
 */
export function formatTime(
  date: Date | string | null | undefined,
  options?: Intl.DateTimeFormatOptions,
  locale?: Locale
): string {
  if (!date || date === '') {
    return 'Не указано'
  }
  
  const dateObj = typeof date === 'string' ? new Date(date) : date
  
  // Проверяем, что dateObj существует и является валидной датой
  if (!dateObj || !(dateObj instanceof Date) || isNaN(dateObj.getTime())) {
    return 'Неверная дата'
  }
  
  const loc = locale || 'ru-RU'
  
  return dateObj.toLocaleTimeString(loc, {
    hour: '2-digit',
    minute: '2-digit',
    ...options,
  })
}

/**
 * Форматирует число с учетом локали
 */
export function formatNumber(
  value: number | string | null | undefined,
  options?: Intl.NumberFormatOptions,
  locale?: Locale
): string {
  if (value === null || value === undefined || value === '') {
    return '0'
  }
  
  const num = typeof value === 'string' ? parseFloat(value) : value
  
  // Проверяем, что число валидно
  if (isNaN(num)) {
    return '0'
  }
  
  const loc = locale || 'ru-RU'
  
  return num.toLocaleString(loc, options)
}

/**
 * Нормализует значение качества/процента: если > 1, считаем что это уже проценты, иначе - доли (0-1)
 * Возвращает значение в диапазоне 0-100
 * Обрабатывает NaN, null, undefined и возвращает 0
 */
export function normalizePercentage(value: number | null | undefined): number {
  // Обрабатываем null, undefined и NaN
  if (value === null || value === undefined || isNaN(value)) {
    return 0
  }
  
  if (value > 1) {
    // Уже в процентах, возвращаем как есть (ограничиваем диапазоном 0-100)
    return Math.min(100, Math.max(0, value))
  }
  // В долях (0-1), конвертируем в проценты (0-100)
  return Math.min(100, Math.max(0, value * 100))
}

/**
 * Определяет тип налогового идентификатора (ИНН или БИН)
 */
export function detectTaxIdType(taxId: string): 'inn' | 'bin' | null {
  const cleaned = taxId.replace(/\s/g, '')
  
  if (!/^\d+$/.test(cleaned)) {
    return null
  }
  
  if (cleaned.length === 12) {
    // 12 цифр может быть как ИНН (Россия), так и БИН (Казахстан)
    // Точное определение требует проверки контрольной суммы на бэкенде
    // Пока считаем потенциальным БИН
    return 'bin'
  }
  
  if (cleaned.length === 10) {
    return 'inn'
  }
  
  return null
}

/**
 * Получает метку для налогового идентификатора
 */
export function getTaxIdLabel(taxId?: string): string {
  if (!taxId) {
    return 'ИНН / БИН'
  }
  
  const type = detectTaxIdType(taxId)
  if (type === 'bin') {
    return 'БИН (Казахстан)'
  }
  if (type === 'inn') {
    return 'ИНН (Россия)'
  }
  
  return 'ИНН / БИН'
}

/**
 * Получает placeholder для поля налогового идентификатора
 */
export function getTaxIdPlaceholder(taxId?: string): string {
  if (!taxId) {
    return 'ИНН: 10 или 12 цифр, БИН: 12 цифр'
  }
  
  const type = detectTaxIdType(taxId)
  if (type === 'bin') {
    return 'БИН: 12 цифр (Казахстан)'
  }
  if (type === 'inn') {
    return 'ИНН: 10 или 12 цифр (Россия)'
  }
  
  return 'ИНН: 10 или 12 цифр, БИН: 12 цифр'
}

/**
 * Форматирует размер файла в читаемый вид
 */
export function formatFileSize(bytes: number | null | undefined): string {
  if (bytes === null || bytes === undefined || bytes === 0 || isNaN(bytes)) {
    return '0 B'
  }
  
  if (bytes < 0) {
    return '0 B'
  }
  
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  
  return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`
}

