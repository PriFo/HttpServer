import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'
import { logger, createApiContext, withLogging } from '@/lib/logger'
import { createErrorResponse } from '@/lib/errors'
import { BackendConnectionError } from '@/lib/errors'

export const runtime = 'nodejs'

export async function GET() {
  const context = createApiContext('/api/health', 'GET')
  const startTime = Date.now()

  return withLogging(
    'GET /api/health',
    async () => {
      const backendUrl = getBackendUrl()
      // Use 127.0.0.1 instead of localhost for better compatibility
      const healthUrl = backendUrl.replace('localhost', '127.0.0.1').replace('http://', 'http://') + '/health'
      
      logger.debug('Attempting to connect to backend health endpoint', {
        ...context,
        healthUrl,
        backendUrl,
      })
      
      // Try using node's http module as fallback if fetch fails
      let response: Response
      try {
        const controller = new AbortController()
        const timeoutId = setTimeout(() => controller.abort(), 5000)
        
        response = await fetch(healthUrl, {
          method: 'GET',
          cache: 'no-store',
          signal: controller.signal,
          headers: {
            'Accept': 'application/json',
            'Connection': 'keep-alive',
          },
        })
        
        clearTimeout(timeoutId)
      } catch (fetchError: any) {
        logger.warn('Fetch failed, trying with http module', {
          ...context,
          error: fetchError?.message,
        })
        
        // Fallback to node http module
        const http = await import('http')
        const url = new URL(healthUrl)
        
        return new Promise((resolve) => {
          const req = http.request({
            hostname: url.hostname,
            port: url.port || 9999,
            path: url.pathname,
            method: 'GET',
            timeout: 5000,
          }, (res) => {
            let data = ''
            res.on('data', (chunk) => { data += chunk })
            res.on('end', () => {
              try {
                const jsonData = JSON.parse(data)
                logger.info('Successfully connected to backend (via http module)', {
                  ...context,
                  duration: Date.now() - startTime,
                })
                resolve(NextResponse.json({ ok: true, ...jsonData }))
              } catch {
                resolve(NextResponse.json({ ok: res.statusCode === 200, data }, { status: res.statusCode || 503 }))
              }
            })
          })
          
          req.on('error', (err) => {
            logger.error('HTTP module error', { ...context, endpoint: healthUrl }, err)
            resolve(NextResponse.json(
              { ok: false, error: 'Backend unavailable', details: err.message },
              { status: 503 }
            ))
          })
          
          req.on('timeout', () => {
            logger.warn('Backend health check timeout', { ...context, endpoint: healthUrl })
            req.destroy()
            resolve(NextResponse.json(
              { ok: false, error: 'Backend timeout' },
              { status: 503 }
            ))
          })
          
          req.end()
        }) as Promise<NextResponse>
      }
      
      const duration = Date.now() - startTime
      logger.logResponse('GET', '/api/health', response.status, duration, context)
      
      if (!response.ok) {
        logger.logBackendError(healthUrl, response.status, undefined, context)
        return NextResponse.json(
          { ok: false, status: response.status, url: healthUrl },
          { status: 503 }
        )
      }
      
      const data = await response.json()
      logger.info('Successfully connected to backend', {
        ...context,
        duration,
      })
      return NextResponse.json({ ok: true, ...data })
    },
    context
  ).catch((error) => {
    const duration = Date.now() - startTime
    logger.error('Health check failed', { ...context, duration }, error)
    return createErrorResponse(
      error instanceof BackendConnectionError 
        ? error 
        : new BackendConnectionError('Backend unavailable', { endpoint: '/health' }),
      { path: '/api/health' }
    )
  })
}

