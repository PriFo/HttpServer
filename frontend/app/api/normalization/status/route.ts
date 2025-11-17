import { NextRequest, NextResponse } from 'next/server'

const API_BASE = process.env.BACKEND_URL || 'http://localhost:9999'

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function GET(_request: NextRequest) {
  try {
    const response = await fetch(`${API_BASE}/api/normalization/status`, {
      method: 'GET',
      headers: {
        'Content-Type': 'application/json',
      },
      // Таймаут обрабатывается через catch блок
    })

    if (!response.ok) {
      // Если бэкенд недоступен, возвращаем дефолтный статус
      if (response.status === 0 || response.status >= 500) {
        return NextResponse.json({
          isRunning: false,
          progress: 0,
          processed: 0,
          total: 15973,
          currentStep: 'Бэкенд недоступен',
          logs: [],
        })
      }
      return NextResponse.json(
        { error: `Backend responded with ${response.status}` },
        { status: response.status }
      )
    }

    const data = await response.json()
    return NextResponse.json(data)
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
  } catch (_error) {
    // Бэкенд недоступен - возвращаем дефолтный статус
    return NextResponse.json({
      isRunning: false,
      progress: 0,
      processed: 0,
      total: 15973,
      currentStep: 'Бэкенд недоступен',
      logs: [],
    })
  }
}

