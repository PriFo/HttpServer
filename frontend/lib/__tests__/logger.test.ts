/**
 * Тесты для системы логирования
 */

import { describe, it, expect, beforeEach, vi } from 'vitest'
import { logger, LogLevel, logDebug, logInfo, logWarn, logError, logFatal } from '../logger'

describe('Logger', () => {
  beforeEach(() => {
    // Очищаем буфер перед каждым тестом
    logger.clearBuffer()
    // Сбрасываем уровень логирования
    logger.setMinLevel(LogLevel.DEBUG)
  })

  describe('Logging levels', () => {
    it('should log debug messages in development', () => {
      const consoleSpy = vi.spyOn(console, 'debug').mockImplementation(() => {})
      logger.setMinLevel(LogLevel.DEBUG)
      logger.debug('Test debug message')
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should log info messages', () => {
      const consoleSpy = vi.spyOn(console, 'info').mockImplementation(() => {})
      logger.info('Test info message')
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should log warn messages', () => {
      const consoleSpy = vi.spyOn(console, 'warn').mockImplementation(() => {})
      logger.warn('Test warn message')
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should log error messages', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const error = new Error('Test error')
      logger.error('Test error message', undefined, error)
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should log fatal messages', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const error = new Error('Fatal error')
      logger.fatal('Test fatal message', undefined, error)
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('Log filtering', () => {
    it('should not log messages below minimum level', () => {
      const consoleSpy = vi.spyOn(console, 'debug').mockImplementation(() => {})
      logger.setMinLevel(LogLevel.INFO)
      logger.debug('This should not be logged')
      expect(consoleSpy).not.toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should log messages at or above minimum level', () => {
      const consoleSpy = vi.spyOn(console, 'info').mockImplementation(() => {})
      logger.setMinLevel(LogLevel.INFO)
      logger.info('This should be logged')
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('Log buffer', () => {
    it('should store logs in buffer', () => {
      logger.info('Test message 1')
      logger.warn('Test message 2')
      logger.error('Test message 3')

      const logs = logger.getRecentLogs()
      expect(logs.length).toBeGreaterThanOrEqual(3)
    })

    it('should filter logs by level', () => {
      logger.info('Info message')
      logger.error('Error message')
      logger.warn('Warn message')

      const errorLogs = logger.getRecentLogs(LogLevel.ERROR)
      expect(errorLogs.every(log => log.level === LogLevel.ERROR)).toBe(true)
    })

    it('should limit buffer size', () => {
      // Заполняем буфер больше максимального размера
      for (let i = 0; i < 150; i++) {
        logger.info(`Message ${i}`)
      }

      const logs = logger.getRecentLogs()
      expect(logs.length).toBeLessThanOrEqual(100)
    })

    it('should clear buffer', () => {
      logger.info('Test message')
      logger.clearBuffer()
      const logs = logger.getRecentLogs()
      expect(logs.length).toBe(0)
    })
  })

  describe('API logging', () => {
    it('should log API errors', () => {
      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {})
      const error = new Error('API Error')
      logger.logApiError('/api/test', 'GET', 500, error, { userId: '123' })
      
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })

    it('should log API success in development', () => {
      const consoleSpy = vi.spyOn(console, 'debug').mockImplementation(() => {})
      logger.logApiSuccess('/api/test', 'GET', 150, { userId: '123' })
      
      // В dev режиме должен логироваться
      if (process.env.NODE_ENV === 'development') {
        expect(consoleSpy).toHaveBeenCalled()
      }
      consoleSpy.mockRestore()
    })
  })

  describe('Convenience functions', () => {
    it('should export convenience functions', () => {
      expect(typeof logDebug).toBe('function')
      expect(typeof logInfo).toBe('function')
      expect(typeof logWarn).toBe('function')
      expect(typeof logError).toBe('function')
      expect(typeof logFatal).toBe('function')
    })

    it('should work with convenience functions', () => {
      const consoleSpy = vi.spyOn(console, 'info').mockImplementation(() => {})
      logInfo('Test message', { key: 'value' })
      expect(consoleSpy).toHaveBeenCalled()
      consoleSpy.mockRestore()
    })
  })

  describe('Log entry structure', () => {
    it('should create structured log entries', () => {
      logger.info('Test message', { contextKey: 'contextValue' })
      const logs = logger.getRecentLogs(LogLevel.INFO)
      
      expect(logs.length).toBeGreaterThan(0)
      const log = logs[logs.length - 1]
      
      expect(log).toHaveProperty('level')
      expect(log).toHaveProperty('message')
      expect(log).toHaveProperty('timestamp')
      expect(log.message).toBe('Test message')
      expect(log.context).toEqual({ contextKey: 'contextValue' })
    })

    it('should include error details in log entry', () => {
      const error = new Error('Test error')
      logger.error('Error occurred', { userId: '123' }, error)
      const logs = logger.getRecentLogs(LogLevel.ERROR)
      
      expect(logs.length).toBeGreaterThan(0)
      const log = logs[logs.length - 1]
      
      expect(log.error).toBeDefined()
      expect(log.error?.name).toBe('Error')
      expect(log.error?.message).toBe('Test error')
    })
  })
})

