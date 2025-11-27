import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

interface RouteCheckConfig {
  path: string
  method?: string
  label: string
  critical?: boolean
  timeoutMs?: number
}

interface RouteCheckResult extends RouteCheckConfig {
  url: string
  ok: boolean
  status: number | null
  statusText?: string
  durationMs: number
  error?: string
  isTimeout?: boolean
}

const ROUTES_TO_CHECK: RouteCheckConfig[] = [
  {
    path: '/health',
    label: 'Health endpoint',
    critical: true,
    timeoutMs: 3000,
  },
  {
    path: '/api/dashboard/stats',
    label: 'Dashboard stats',
    critical: true,
  },
  {
    path: '/api/dashboard/normalization-status',
    label: 'Dashboard normalization status',
  },
  {
    path: '/api/monitoring/metrics',
    label: 'Monitoring metrics',
    critical: true,
  },
  {
    path: '/api/monitoring/providers',
    label: 'Monitoring providers',
  },
  {
    path: '/api/workers/config',
    label: 'Worker config',
    critical: true,
  },
  {
    path: '/api/quality/metrics',
    label: 'Quality metrics',
  },
  {
    path: '/api/clients',
    label: 'Clients list',
  },
  {
    path: '/api/gosts/statistics',
    label: 'GOST statistics',
  },
  {
    path: '/api/okpd2/stats',
    label: 'OKPD2 statistics',
  },
]

async function checkRoute(
  backendUrl: string,
  route: RouteCheckConfig,
): Promise<RouteCheckResult> {
  const url = `${backendUrl}${route.path}`
  const controller = new AbortController()
  const timeoutMs = route.timeoutMs ?? 7000
  const timeout = setTimeout(() => controller.abort(), timeoutMs)
  const startedAt = performance.now()

  try {
    const response = await fetch(url, {
      method: route.method ?? 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      signal: controller.signal,
    })
    clearTimeout(timeout)

    return {
      ...route,
      url,
      ok: response.ok,
      status: response.status,
      statusText: response.statusText,
      durationMs: Math.round(performance.now() - startedAt),
    }
  } catch (error) {
    clearTimeout(timeout)
    const err = error as Error
    const isTimeout = err.name === 'AbortError'

    return {
      ...route,
      url,
      ok: false,
      status: null,
      statusText: undefined,
      durationMs: Math.round(performance.now() - startedAt),
      error: err.message || 'Unknown error',
      isTimeout,
    }
  }
}

export async function GET() {
  const backendUrl = getBackendUrl()

  const results = await Promise.all(
    ROUTES_TO_CHECK.map((route) => checkRoute(backendUrl, route)),
  )

  const summary = {
    checkedAt: new Date().toISOString(),
    backendUrl,
    total: results.length,
    healthy: results.filter((result) => result.ok).length,
    degraded: results.filter((result) => !result.ok && !result.critical).length,
    failed: results.filter((result) => !result.ok && result.critical).length,
    hasCriticalFailure: results.some(
      (result) => result.critical && !result.ok,
    ),
  }

  return NextResponse.json({ summary, results })
}

