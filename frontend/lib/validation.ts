/**
 * Утилиты для валидации данных
 */

/**
 * Валидирует email адрес
 */
export function isValidEmail(email: string): boolean {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  return emailRegex.test(email)
}

/**
 * Валидирует ИНН (10 или 12 цифр)
 */
export function isValidINN(inn: string): boolean {
  const cleaned = inn.replace(/\s/g, '')
  return /^\d{10}$|^\d{12}$/.test(cleaned)
}

/**
 * Валидирует БИН (12 цифр)
 */
export function isValidBIN(bin: string): boolean {
  const cleaned = bin.replace(/\s/g, '')
  return /^\d{12}$/.test(cleaned)
}

/**
 * Валидирует ОГРН (13 цифр для юридических лиц, 15 для ИП)
 */
export function isValidOGRN(ogrn: string): boolean {
  const cleaned = ogrn.replace(/\s/g, '')
  return /^\d{13}$|^\d{15}$/.test(cleaned)
}

/**
 * Валидирует КПП (9 цифр)
 */
export function isValidKPP(kpp: string): boolean {
  const cleaned = kpp.replace(/\s/g, '')
  return /^\d{9}$/.test(cleaned)
}

/**
 * Валидирует БИК (9 цифр)
 */
export function isValidBIK(bik: string): boolean {
  const cleaned = bik.replace(/\s/g, '')
  return /^\d{9}$/.test(cleaned)
}

/**
 * Валидирует телефонный номер (базовая проверка)
 */
export function isValidPhone(phone: string): boolean {
  const cleaned = phone.replace(/\s|-|\(|\)/g, '')
  return /^\+?\d{10,15}$/.test(cleaned)
}

/**
 * Валидирует URL
 */
export function isValidURL(url: string): boolean {
  try {
    new URL(url)
    return true
  } catch {
    return false
  }
}

/**
 * Валидирует, что значение не пустое
 */
export function isNotEmpty(value: string | null | undefined): boolean {
  return value !== null && value !== undefined && value.trim().length > 0
}

/**
 * Валидирует длину строки
 */
export function isValidLength(
  value: string,
  min: number,
  max?: number
): boolean {
  const length = value.length
  if (max !== undefined) {
    return length >= min && length <= max
  }
  return length >= min
}

/**
 * Валидирует числовое значение
 */
export function isValidNumber(
  value: string | number,
  min?: number,
  max?: number
): boolean {
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) {
    return false
  }
  if (min !== undefined && num < min) {
    return false
  }
  if (max !== undefined && num > max) {
    return false
  }
  return true
}

/**
 * Валидирует дату
 */
export function isValidDate(date: string | Date): boolean {
  const d = typeof date === 'string' ? new Date(date) : date
  return !isNaN(d.getTime())
}

/**
 * Валидирует объект отчета
 */
export function isValidReportMetadata(metadata: any): boolean {
  return (
    metadata &&
    typeof metadata === 'object' &&
    typeof metadata.generated_at === 'string' &&
    typeof metadata.report_version === 'string' &&
    typeof metadata.total_databases === 'number' &&
    typeof metadata.total_projects === 'number'
  )
}

/**
 * Валидирует данные мониторинга
 */
export function isValidMonitoringData(data: any): boolean {
  return (
    data &&
    typeof data === 'object' &&
    Array.isArray(data.providers) &&
    data.system &&
    typeof data.system.total_providers === 'number' &&
    typeof data.system.active_providers === 'number'
  )
}

/**
 * Схемы валидации для запросов
 */
export const kpvedLoadSchema = {
  type: 'object',
  properties: {
    file_path: { type: 'string' },
    // database_id removed - KPVED always uses serviceDB
  },
  required: ['file_path'],
}

export const kpvedReclassifySchema = {
  type: 'object',
  properties: {
    database_id: { type: 'number' },
    project_id: { type: 'number' },
    client_id: { type: 'number' },
  },
  required: [],
}

export const qualityAnalyzeSchema = {
  type: 'object',
  properties: {
    database_id: { type: 'number' },
    project_id: { type: 'number' },
    client_id: { type: 'number' },
  },
  required: [],
}

export const violationResolveSchema = {
  type: 'object',
  properties: {
    resolution: { type: 'string' },
    comment: { type: 'string' },
  },
  required: [],
}

/**
 * Результат валидации
 */
export interface ValidationResult<T> {
  success: boolean
  data?: T
  details?: any
}

/**
 * Валидирует запрос по схеме
 */
export function validateRequest<T>(schema: any, data: any): ValidationResult<T> {
  // Простая валидация - можно расширить с помощью библиотеки типа zod или yup
  if (!data || typeof data !== 'object') {
    return {
      success: false,
      details: { message: 'Invalid request data' },
    }
  }

  // Проверяем обязательные поля
  if (schema.required) {
    for (const field of schema.required) {
      if (!(field in data) || data[field] === null || data[field] === undefined) {
        return {
          success: false,
          details: { 
            message: `Missing required field: ${field}`,
            field,
          },
        }
      }
    }
  }

  return {
    success: true,
    data: data as T,
  }
}

/**
 * Форматирует ошибки валидации для ответа
 */
export function formatValidationError(details: any): string | Record<string, any> {
  if (typeof details === 'string') {
    return details
  }
  
  if (details?.message) {
    return details.message
  }
  
  if (details?.field) {
    return `Поле "${details.field}" обязательно для заполнения`
  }
  
  return 'Ошибка валидации данных'
}