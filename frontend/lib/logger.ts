/**
 * Централизованная система логирования с уровнями и структурированными логами
 */

export enum LogLevel {
  DEBUG = 0,
  INFO = 1,
  WARN = 2,
  ERROR = 3,
  FATAL = 4,
}

export interface LogContext {
  [key: string]: unknown
}

export interface LogEntry {
  level: LogLevel
  message: string
  timestamp: string
  context?: LogContext
  error?: {
    name: string
    message: string
    stack?: string
  }
  userAgent?: string
  url?: string
}

class Logger {
  private minLevel: LogLevel = LogLevel.INFO
  private isDevelopment: boolean = false
  private logBuffer: LogEntry[] = []
  private maxBufferSize: number = 100

  constructor() {
    if (typeof window !== 'undefined') {
      this.isDevelopment = process.env.NODE_ENV === 'development'
      this.minLevel = this.isDevelopment ? LogLevel.DEBUG : LogLevel.INFO
    }
  }

  private shouldLog(level: LogLevel): boolean {
    return level >= this.minLevel
  }

  private formatMessage(level: LogLevel, message: string, context?: LogContext, error?: Error): string {
    const levelName = LogLevel[level]
    const timestamp = new Date().toISOString()
    const parts = [`[${timestamp}] [${levelName}]`, message]

    if (context && Object.keys(context).length > 0) {
      parts.push(`Context: ${JSON.stringify(context)}`)
    }

    if (error) {
      parts.push(`Error: ${error.name}: ${error.message}`)
      if (error.stack && this.isDevelopment) {
        parts.push(`Stack: ${error.stack}`)
      }
    }

    return parts.join(' | ')
  }

  private createLogEntry(
    level: LogLevel,
    message: string,
    context?: LogContext,
    error?: Error
  ): LogEntry {
    const entry: LogEntry = {
      level,
      message,
      timestamp: new Date().toISOString(),
      context,
    }

    if (error) {
      entry.error = {
        name: error.name,
        message: error.message,
        stack: error.stack,
      }
    }

    if (typeof window !== 'undefined') {
      entry.userAgent = navigator.userAgent
      entry.url = window.location.href
    }

    return entry
  }

  private log(level: LogLevel, message: string, context?: LogContext, error?: Error): void {
    if (!this.shouldLog(level)) {
      return
    }

    const entry = this.createLogEntry(level, message, context, error)
    
    // Добавляем в буфер
    this.logBuffer.push(entry)
    if (this.logBuffer.length > this.maxBufferSize) {
      this.logBuffer.shift()
    }

    // Выводим в консоль
    const formattedMessage = this.formatMessage(level, message, context, error)
    
    switch (level) {
      case LogLevel.DEBUG:
        console.debug(formattedMessage, context || '', error || '')
        break
      case LogLevel.INFO:
        console.info(formattedMessage, context || '')
        break
      case LogLevel.WARN:
        console.warn(formattedMessage, context || '', error || '')
        break
      case LogLevel.ERROR:
      case LogLevel.FATAL:
        console.error(formattedMessage, context || '', error || '')
        break
    }

    // В продакшене можно отправлять в систему мониторинга
    if (!this.isDevelopment && level >= LogLevel.ERROR) {
      this.sendToMonitoring(entry)
    }
  }

  private sendToMonitoring(entry: LogEntry): void {
    // Интеграция с Sentry, LogRocket и т.д.
    // if (typeof window !== 'undefined' && window.Sentry) {
    //   window.Sentry.captureMessage(entry.message, {
    //     level: LogLevel[entry.level].toLowerCase(),
    //     extra: entry.context,
    //     tags: { url: entry.url },
    //   })
    // }
  }

  debug(message: string, context?: LogContext): void {
    this.log(LogLevel.DEBUG, message, context)
  }

  info(message: string, context?: LogContext): void {
    this.log(LogLevel.INFO, message, context)
  }

  warn(message: string, context?: LogContext, error?: Error): void {
    this.log(LogLevel.WARN, message, context, error)
  }

  error(message: string, context?: LogContext, error?: Error): void {
    this.log(LogLevel.ERROR, message, context, error)
  }

  fatal(message: string, context?: LogContext, error?: Error): void {
    this.log(LogLevel.FATAL, message, context, error)
  }

  /**
   * Логирует ошибку API запроса
   */
  logApiError(
    url: string,
    method: string,
    status: number,
    error: Error,
    context?: LogContext
  ): void {
    this.error(
      `API Error: ${method} ${url} - ${status}`,
      {
        ...context,
        url,
        method,
        status,
        statusText: error.message,
      },
      error
    )
  }

  /**
   * Логирует успешный API запрос (только в dev режиме)
   */
  logApiSuccess(url: string, method: string, duration: number, context?: LogContext): void {
    if (this.isDevelopment) {
      this.debug(`API Success: ${method} ${url} - ${duration}ms`, context)
    }
  }

  /**
   * Получает последние логи из буфера
   */
  getRecentLogs(level?: LogLevel, limit: number = 50): LogEntry[] {
    let logs = this.logBuffer

    if (level !== undefined) {
      logs = logs.filter(log => log.level === level)
    }

    return logs.slice(-limit)
  }

  /**
   * Очищает буфер логов
   */
  clearBuffer(): void {
    this.logBuffer = []
  }

  /**
   * Устанавливает минимальный уровень логирования
   */
  setMinLevel(level: LogLevel): void {
    this.minLevel = level
  }

  /**
   * Логирует HTTP запрос
   */
  logRequest(method: string, url: string, context?: LogContext): void {
    this.debug(`HTTP ${method} ${url}`, {
      method,
      url,
      ...context,
    })
  }

  /**
   * Логирует HTTP ответ
   */
  logResponse(
    method: string,
    url: string,
    status: number,
    duration?: number,
    context?: LogContext
  ): void {
    const level = status >= 500 ? LogLevel.ERROR : status >= 400 ? LogLevel.WARN : LogLevel.INFO
    const message = `HTTP ${method} ${url} → ${status}${duration ? ` (${duration}ms)` : ''}`

    if (level === LogLevel.ERROR) {
      this.error(message, { status, duration, ...context })
    } else if (level === LogLevel.WARN) {
      this.warn(message, { status, duration, ...context })
    } else {
      this.info(message, { status, duration, ...context })
    }
  }

  /**
   * Логирует ошибку бэкенда
   */
  logBackendError(
    endpoint: string,
    status: number,
    errorText?: string,
    context?: LogContext
  ): void {
    const message = `Backend error: ${endpoint} returned ${status}`
    const level = status >= 500 ? LogLevel.ERROR : LogLevel.WARN

    const logContext: LogContext = {
      endpoint,
      status,
      ...context,
    }

    if (errorText) {
      logContext.errorText = errorText.length > 500
        ? errorText.substring(0, 500) + '...'
        : errorText
    }

    if (level === LogLevel.ERROR) {
      this.error(message, logContext)
    } else {
      this.warn(message, logContext)
    }
  }
}

// Экспортируем singleton экземпляр
export const logger = new Logger()

// Экспортируем удобные функции
export const logDebug = (message: string, context?: LogContext) => logger.debug(message, context)
export const logInfo = (message: string, context?: LogContext) => logger.info(message, context)
export const logWarn = (message: string, context?: LogContext, error?: Error) =>
  logger.warn(message, context, error)
export const logError = (message: string, context?: LogContext, error?: Error) =>
  logger.error(message, context, error)
export const logFatal = (message: string, context?: LogContext, error?: Error) =>
  logger.fatal(message, context, error)

/**
 * Создает контекст для логирования API route
 */
export function createApiContext(
  route: string,
  method: string,
  params?: Record<string, unknown>,
  query?: Record<string, unknown>
): LogContext {
  return {
    route,
    method,
    ...(params && Object.keys(params).length > 0 && { params }),
    ...(query && Object.keys(query).length > 0 && { query }),
  }
}

/**
 * Обертка для логирования выполнения функции
 */
export async function withLogging<T>(
  operation: string,
  fn: () => Promise<T>,
  context?: LogContext
): Promise<T> {
  const startTime = Date.now()
  logger.debug(`Starting: ${operation}`, context)

  try {
    const result = await fn()
    const duration = Date.now() - startTime
    logger.info(`Completed: ${operation}`, { ...context, duration })
    return result
  } catch (error) {
    const duration = Date.now() - startTime
    logger.error(`Failed: ${operation}`, { ...context, duration }, error instanceof Error ? error : undefined)
    throw error
  }
}
