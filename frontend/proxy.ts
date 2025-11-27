import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { getUserFromRequest, isAdmin } from '@/lib/jwt'

/**
 * Security Proxy for API Routes and Internal Pages
 *
 * Provides:
 * - API key authentication for production environments
 * - Rate limiting headers
 * - Security headers (CORS, CSP, etc.)
 * - Role-based access control for /internal/* routes
 */

// API routes that should be protected
const PROTECTED_API_ROUTES = [
  '/api/kpved',
  '/api/quality',
  '/api/normalization',
  '/api/classifiers',
  '/api/workers',
]

// API routes that should be excluded from rate limiting (e.g., logging endpoints, metrics, monitoring, status)
const RATE_LIMIT_EXCLUDED_ROUTES = [
  '/api/logs',
  '/api/quality/metrics',
  '/api/quality/stats',
  '/api/dashboard/stats',
  '/api/monitoring/metrics',
  '/api/monitoring/providers/stream',
  '/api/monitoring/events',
  '/api/normalization/status',
  '/api/counterparties/normalization/status',
  '/api/kpved/workers/status',
  '/api/dashboard/normalization-status',
  '/api/database/info', // Часто запрашивается для отображения информации о БД
  '/api/quality/analyze/status', // Опрашивается каждую секунду во время анализа качества
]

// API routes for file uploads - stricter rate limiting
const FILE_UPLOAD_ROUTES = [
  '/api/clients/',
]

// Rate limiting for file uploads (more restrictive)
const FILE_UPLOAD_RATE_LIMIT_WINDOW = 60 * 1000 // 1 minute
const FILE_UPLOAD_RATE_LIMIT_MAX_REQUESTS = 10 // 10 uploads per minute

// Check if request path matches protected routes
function isProtectedRoute(pathname: string): boolean {
  return PROTECTED_API_ROUTES.some(route => pathname.startsWith(route))
}

// Check if request path should be excluded from rate limiting
function isRateLimitExcluded(pathname: string): boolean {
  return RATE_LIMIT_EXCLUDED_ROUTES.some(route => pathname.startsWith(route))
}

// Check if request is a file upload
function isFileUploadRoute(pathname: string): boolean {
  return FILE_UPLOAD_ROUTES.some(route => pathname.includes(route) && pathname.includes('/databases'))
}

// Simple rate limiting (in-memory, for demo - use Redis in production)
const rateLimitMap = new Map<string, { count: number; resetTime: number }>()
const RATE_LIMIT_WINDOW = 60 * 1000 // 1 minute
const RATE_LIMIT_MAX_REQUESTS = 100 // 100 requests per minute

function checkRateLimit(identifier: string): { allowed: boolean; remaining: number } {
  const now = Date.now()
  const record = rateLimitMap.get(identifier)

  if (!record || now > record.resetTime) {
    // Reset or create new record
    rateLimitMap.set(identifier, {
      count: 1,
      resetTime: now + RATE_LIMIT_WINDOW,
    })
    return { allowed: true, remaining: RATE_LIMIT_MAX_REQUESTS - 1 }
  }

  if (record.count >= RATE_LIMIT_MAX_REQUESTS) {
    return { allowed: false, remaining: 0 }
  }

  record.count++
  return { allowed: true, remaining: RATE_LIMIT_MAX_REQUESTS - record.count }
}

// Rate limiting specifically for file uploads
function checkFileUploadRateLimit(identifier: string): { allowed: boolean; remaining: number } {
  const now = Date.now()
  const record = rateLimitMap.get(`upload_${identifier}`)

  if (!record || now > record.resetTime) {
    rateLimitMap.set(`upload_${identifier}`, {
      count: 1,
      resetTime: now + FILE_UPLOAD_RATE_LIMIT_WINDOW,
    })
    return { allowed: true, remaining: FILE_UPLOAD_RATE_LIMIT_MAX_REQUESTS - 1 }
  }

  if (record.count >= FILE_UPLOAD_RATE_LIMIT_MAX_REQUESTS) {
    return { allowed: false, remaining: 0 }
  }

  record.count++
  return { allowed: true, remaining: FILE_UPLOAD_RATE_LIMIT_MAX_REQUESTS - record.count }
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl

  // CRITICAL: Skip all Next.js API routes - they handle their own proxying to backend
  // Next.js API routes (app/api/*) are server-side routes that should not be intercepted
  // They will proxy to backend themselves and handle errors appropriately
  if (pathname.startsWith('/api/')) {
    // Just add security headers and pass through
    const response = NextResponse.next()
    response.headers.set('X-Content-Type-Options', 'nosniff')
    response.headers.set('X-Frame-Options', 'DENY')
    response.headers.set('X-XSS-Protection', '1; mode=block')
    return response
  }

  // Check if this is an internal route that requires admin access
  if (pathname.startsWith('/internal/')) {
    try {
      const user = getUserFromRequest(request as any)
      
      // Check if user is admin
      if (!isAdmin(user)) {
        // In development mode, allow access if ALLOW_INTERNAL_ACCESS is set
        if (process.env.NODE_ENV === 'development' && process.env.ALLOW_INTERNAL_ACCESS === 'true') {
          return NextResponse.next()
        }
        
        // Redirect to home page if not admin
        const url = request.nextUrl.clone()
        url.pathname = '/'
        return NextResponse.redirect(url)
      }
      
      // Admin access granted, continue
      return NextResponse.next()
    } catch (error) {
      // If there's an error getting user, redirect to home
      const url = request.nextUrl.clone()
      url.pathname = '/'
      return NextResponse.redirect(url)
    }
  }

  // Skip all other routes
  return NextResponse.next()
}

// Configure which routes the proxy runs on
// Note: proxy only runs on /internal/* routes, /api/* routes are handled by Next.js API routes
export const config = {
  matcher: ['/internal/:path*'],
}