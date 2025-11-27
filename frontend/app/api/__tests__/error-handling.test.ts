/**
 * Интеграционные тесты для обработки ошибок в API роутах
 */

import { describe, it, expect, vi, beforeEach } from 'vitest'
import { NextRequest } from 'next/server'
import { createErrorResponse, withErrorHandler } from '@/lib/errors'
import { AppError } from '@/lib/errors'

describe('API Error Handling', () => {
  describe('createErrorResponse', () => {
    it('should create error response for AppError', () => {
      const error = new AppError('User message', 'Technical details', 400, 'VALIDATION_ERROR')
      const response = createErrorResponse(error)

      expect(response.status).toBe(400)
      const json = response.json() as any
      expect(json.error).toBe('User message')
      expect(json.code).toBe('VALIDATION_ERROR')
    })

    it('should create error response for standard Error', () => {
      const error = new Error('Test error')
      const response = createErrorResponse(error)

      expect(response.status).toBe(500)
    })

    it('should include stack trace in development', () => {
      const originalEnv = process.env.NODE_ENV
      process.env.NODE_ENV = 'development'

      const error = new Error('Test error')
      const response = createErrorResponse(error, { includeStack: true })

      const json = response.json() as any
      expect(json.details).toBeDefined()

      process.env.NODE_ENV = originalEnv
    })

    it('should not include stack trace in production', () => {
      const originalEnv = process.env.NODE_ENV
      process.env.NODE_ENV = 'production'

      const error = new Error('Test error')
      const response = createErrorResponse(error, { includeStack: false })

      const json = response.json() as any
      expect(json.details).not.toHaveProperty('stack')

      process.env.NODE_ENV = originalEnv
    })

    it('should include path in error response', () => {
      const error = new AppError('Test error', undefined, 404, 'NOT_FOUND')
      const response = createErrorResponse(error, { path: '/api/test' })

      const json = response.json() as any
      expect(json.path).toBe('/api/test')
    })
  })

  describe('withErrorHandler', () => {
    it('should wrap handler and catch errors', async () => {
      const handler = vi.fn().mockRejectedValue(new Error('Handler error'))
      const wrappedHandler = withErrorHandler(handler)

      const request = new NextRequest('http://localhost:3000/api/test')
      const response = await wrappedHandler(request)

      expect(response.status).toBe(500)
      expect(handler).toHaveBeenCalled()
    })

    it('should pass through successful responses', async () => {
      const handler = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ success: true }), { status: 200 })
      )
      const wrappedHandler = withErrorHandler(handler)

      const request = new NextRequest('http://localhost:3000/api/test')
      const response = await wrappedHandler(request)

      expect(response.status).toBe(200)
    })

    it('should handle AppError correctly', async () => {
      const error = new AppError('User message', 'Technical', 400, 'VALIDATION_ERROR')
      const handler = vi.fn().mockRejectedValue(error)
      const wrappedHandler = withErrorHandler(handler)

      const request = new NextRequest('http://localhost:3000/api/test')
      const response = await wrappedHandler(request)

      expect(response.status).toBe(400)
    })
  })
})

