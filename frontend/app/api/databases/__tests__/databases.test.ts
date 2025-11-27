/**
 * Тесты для API routes баз данных
 * 
 * Эти тесты проверяют корректность работы всех API endpoints,
 * связанных с базами данных.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest'
import { NextRequest } from 'next/server'

// Мокаем getBackendUrl
vi.mock('@/lib/api-config', () => ({
  getBackendUrl: vi.fn(() => 'http://localhost:9999'),
}))

// Мокаем logger
vi.mock('@/lib/logger', () => ({
  logger: {
    debug: vi.fn(),
    info: vi.fn(),
    warn: vi.fn(),
    error: vi.fn(),
    logRequest: vi.fn(),
    logResponse: vi.fn(),
    logBackendError: vi.fn(),
    logApiError: vi.fn(),
    logApiSuccess: vi.fn(),
  },
  createApiContext: vi.fn((route, method, params, query) => ({
    route,
    method,
    ...(params && { params }),
    ...(query && { query }),
  })),
  withLogging: vi.fn((operation, fn) => fn()),
}))

// Мокаем error-handler
vi.mock('@/lib/error-handler', async () => {
  const actual = await vi.importActual('@/lib/error-handler')
  
  // Создаем класс ValidationError для мока
  class ValidationError extends Error {
    constructor(message: string, public field?: string, public details?: unknown) {
      super(message)
      this.name = 'ValidationError'
    }
    status = 400
    code = 'VALIDATION_ERROR'
  }
  
  return {
    ...actual,
    ValidationError,
    handleBackendResponse: vi.fn(async (response, endpoint, context, options) => {
      if (!response.ok) {
        if (response.status === 404 && options?.allow404) {
          return new Response(JSON.stringify(options.defaultData || null), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          })
        }
        const errorText = await response.text().catch(() => '')
        return new Response(JSON.stringify({ error: errorText || 'Backend error' }), {
          status: response.status,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      const data = await response.json()
      return new Response(JSON.stringify(data), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
    handleFetchError: vi.fn((error, endpoint, context) => {
      return new Response(JSON.stringify({ error: 'Connection failed' }), {
        status: 503,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
    handleError: vi.fn((error, context) => {
      if (error instanceof ValidationError) {
        return new Response(JSON.stringify({ 
          error: error.message, 
          code: error.code 
        }), {
          status: error.status,
          headers: { 'Content-Type': 'application/json' },
        })
      }
      return new Response(JSON.stringify({ error: 'Internal server error' }), {
        status: 500,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
    validateRequired: vi.fn((value, paramName, context) => {
      if (!value || (typeof value === 'string' && value.trim() === '')) {
        throw new ValidationError(`Required parameter "${paramName}" is missing or empty`, paramName)
      }
    }),
  }
})

// Мокаем fetch
global.fetch = vi.fn()

describe('Databases API Routes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('GET /api/databases/list', () => {
    it('должен возвращать список баз данных', async () => {
      const mockDatabases = [
        { id: 1, name: 'test.db', path: '/path/to/test.db' },
        { id: 2, name: 'test2.db', path: '/path/to/test2.db' },
      ]

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockDatabases,
      })

      const { GET } = await import('../list/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockDatabases)
      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:9999/api/databases/list',
        expect.objectContaining({
          cache: 'no-store',
        })
      )
    })

    it('должен возвращать пустой список при 404', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 404,
        text: async () => 'Not found',
      })

      const { GET } = await import('../list/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual([])
    })

    it('должен обрабатывать ошибки сети', async () => {
      ;(global.fetch as any).mockRejectedValueOnce(
        new Error('Network error')
      )

      const { GET } = await import('../list/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual([])
    })
  })

  describe('GET /api/databases/files', () => {
    it('должен возвращать список файлов баз данных', async () => {
      const mockFiles = {
        success: true,
        total: 2,
        files: [
          { path: '/path/to/test.db', name: 'test.db', type: 'main' },
          { path: '/path/to/test2.db', name: 'test2.db', type: 'uploaded' },
        ],
        grouped: {
          main: [{ path: '/path/to/test.db', name: 'test.db', type: 'main' }],
          service: [],
          uploaded: [{ path: '/path/to/test2.db', name: 'test2.db', type: 'uploaded' }],
          other: [],
        },
        summary: {
          main: 1,
          service: 0,
          uploaded: 1,
          other: 0,
        },
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockFiles,
      })

      const { GET } = await import('../files/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockFiles)
    })

    it('должен возвращать структурированный ответ при ошибке', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 500,
      })

      const { GET } = await import('../files/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(500)
      expect(data).toHaveProperty('success', false)
      expect(data).toHaveProperty('files', [])
      expect(data).toHaveProperty('grouped')
      expect(data).toHaveProperty('summary')
    })
  })

  describe('GET /api/databases/backups', () => {
    it('должен возвращать список бэкапов', async () => {
      const mockBackups = {
        backups: [
          { filename: 'backup1.db', size: 1024, modified_at: '2024-01-01T00:00:00Z' },
          { filename: 'backup2.db', size: 2048, modified_at: '2024-01-02T00:00:00Z' },
        ],
        total: 2,
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockBackups,
      })

      const { GET } = await import('../backups/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockBackups)
    })

    it('должен возвращать пустой список при 404', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 404,
      })

      const { GET } = await import('../backups/route')
      const response = await GET()
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual({ backups: [], total: 0 })
    })
  })

  describe('GET /api/databases/backups/[filename]', () => {
    it('должен скачивать бэкап', async () => {
      const mockBlob = new Blob(['test data'], { type: 'application/octet-stream' })

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        blob: async () => mockBlob,
        headers: new Headers({
          'Content-Type': 'application/octet-stream',
          'Content-Disposition': 'attachment; filename="backup.db"',
        }),
      })

      const { GET } = await import('../backups/[filename]/route')
      const request = new NextRequest('http://localhost:3000/api/databases/backups/test.db')
      const response = await GET(request, {
        params: Promise.resolve({ filename: 'test.db' }),
      })

      expect(response.status).toBe(200)
      expect(response.headers.get('Content-Type')).toBe('application/octet-stream')
    })
  })

  describe('GET /api/databases/analytics', () => {
    it('должен возвращать аналитику базы данных', async () => {
      const mockAnalytics = {
        file_path: '/path/to/test.db',
        database_type: 'sqlite',
        total_size: 1024000,
        total_size_mb: 1,
        table_count: 5,
        total_rows: 1000,
        table_stats: [],
        top_tables: [],
        analyzed_at: '2024-01-01T00:00:00Z',
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockAnalytics,
      })

      const { GET } = await import('../analytics/route')
      const request = new NextRequest('http://localhost:3000/api/databases/analytics?path=/path/to/test.db')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockAnalytics)
    })

    it('должен требовать параметр path', async () => {
      const { GET } = await import('../analytics/route')
      const request = new NextRequest('http://localhost:3000/api/databases/analytics')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(400)
      expect(data.error).toContain('path')
    })
  })

  describe('GET /api/databases/history', () => {
    it('должен возвращать историю базы данных', async () => {
      const mockHistory = {
        history: [
          { timestamp: '2024-01-01T00:00:00Z', action: 'created' },
          { timestamp: '2024-01-02T00:00:00Z', action: 'updated' },
        ],
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockHistory,
      })

      const { GET } = await import('../history/route')
      const request = new NextRequest('http://localhost:3000/api/databases/history?path=/path/to/test.db')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockHistory)
    })

    it('должен возвращать пустую историю при 404', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 404,
      })

      const { GET } = await import('../history/route')
      const request = new NextRequest('http://localhost:3000/api/databases/history?path=/path/to/test.db')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual({ history: [] })
    })
  })

  describe('GET /api/databases/pending', () => {
    it('должен возвращать список ожидающих баз данных', async () => {
      const mockPending = {
        databases: [
          { id: 1, file_name: 'pending.db', status: 'pending' },
        ],
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockPending,
      })

      const { GET } = await import('../pending/route')
      const request = new NextRequest('http://localhost:3000/api/databases/pending')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockPending)
    })

    it('должен фильтровать по статусу', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => ({ databases: [] }),
      })

      const { GET } = await import('../pending/route')
      const request = new NextRequest('http://localhost:3000/api/databases/pending?status=pending')
      await GET(request)

      expect(global.fetch).toHaveBeenCalledWith(
        'http://localhost:9999/api/databases/pending?status=pending',
        expect.any(Object)
      )
    })
  })

  describe('POST /api/databases/scan', () => {
    it('должен запускать сканирование баз данных', async () => {
      const mockResult = {
        success: true,
        scanned: 5,
        found: 3,
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockResult,
      })

      const { POST } = await import('../scan/route')
      const request = new NextRequest('http://localhost:3000/api/databases/scan', {
        method: 'POST',
        body: JSON.stringify({ path: '/scan/path' }),
      })
      const response = await POST(request)
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockResult)
    })
  })

  describe('GET /api/databases/find-project', () => {
    it('должен находить проект по пути к базе данных', async () => {
      const mockProject = {
        client_id: 1,
        project_id: 1,
        project_name: 'Test Project',
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockProject,
      })

      const { GET } = await import('../find-project/route')
      const request = new NextRequest('http://localhost:3000/api/databases/find-project?file_path=/path/to/test.db')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockProject)
    })

    it('должен требовать параметр file_path', async () => {
      const { GET } = await import('../find-project/route')
      const request = new NextRequest('http://localhost:3000/api/databases/find-project')
      const response = await GET(request)
      const data = await response.json()

      expect(response.status).toBe(400)
      expect(data.error).toContain('file_path')
    })
  })

  describe('GET /api/clients/[clientId]/databases', () => {
    it('должен возвращать базы данных клиента', async () => {
      const mockDatabases = [
        { id: 1, name: 'db1.db', client_id: 1 },
        { id: 2, name: 'db2.db', client_id: 1 },
      ]

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockDatabases,
      })

      const { GET } = await import('../../clients/[clientId]/databases/route')
      const request = new NextRequest('http://localhost:3000/api/clients/1/databases')
      const response = await GET(request, {
        params: Promise.resolve({ clientId: '1' }),
      })
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockDatabases)
    })
  })

  describe('GET /api/clients/[clientId]/projects/[projectId]/databases', () => {
    it('должен возвращать базы данных проекта', async () => {
      const mockDatabases = {
        databases: [
          { id: 1, name: 'db1.db', project_id: 1 },
        ],
        total: 1,
      }

      ;(global.fetch as any).mockResolvedValueOnce({
        ok: true,
        json: async () => mockDatabases,
      })

      const { GET } = await import('../../clients/[clientId]/projects/[projectId]/databases/route')
      const request = new NextRequest('http://localhost:3000/api/clients/1/projects/1/databases')
      const response = await GET(request, {
        params: Promise.resolve({ clientId: '1', projectId: '1' }),
      })
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual(mockDatabases)
    })

    it('должен возвращать пустой список при 404', async () => {
      ;(global.fetch as any).mockResolvedValueOnce({
        ok: false,
        status: 404,
      })

      const { GET } = await import('../../clients/[clientId]/projects/[projectId]/databases/route')
      const request = new NextRequest('http://localhost:3000/api/clients/1/projects/1/databases')
      const response = await GET(request, {
        params: Promise.resolve({ clientId: '1', projectId: '1' }),
      })
      const data = await response.json()

      expect(response.status).toBe(200)
      expect(data).toEqual({ databases: [], total: 0 })
    })
  })
})

