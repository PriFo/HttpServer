import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const API_BASE_URL = getBackendUrl()

export async function POST(request: NextRequest) {
  try {
    // Получаем параметры из query string или body
    const { searchParams } = new URL(request.url)
    const clientId = searchParams.get('client_id')
    const projectId = searchParams.get('project_id')

    let body: Record<string, unknown> = {}
    try {
      body = await request.json()
    } catch {
      // Body может быть пустым
    }

    // Если есть client_id и project_id в query или body, используем специальный эндпоинт
    const finalClientId = clientId || body.client_id || body.clientId
    const finalProjectId = projectId || body.project_id || body.projectId

    if (finalClientId && finalProjectId) {
      const clientIdNum = parseInt(String(finalClientId), 10)
      const projectIdNum = parseInt(String(finalProjectId), 10)
      
      // Валидация ID
      if (isNaN(clientIdNum) || clientIdNum <= 0) {
        return NextResponse.json(
          { error: 'Invalid client_id: must be a positive integer' },
          { status: 400 }
        )
      }
      
      if (isNaN(projectIdNum) || projectIdNum <= 0) {
        return NextResponse.json(
          { error: 'Invalid project_id: must be a positive integer' },
          { status: 400 }
        )
      }

      const url = `${API_BASE_URL}/api/clients/${clientIdNum}/projects/${projectIdNum}/normalization/stop`
      
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000) // Увеличено до 10 секунд

      try {
        const response = await fetch(url, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          signal: controller.signal,
        })

        clearTimeout(timeoutId)

        if (!response.ok) {
          let errorText = 'Unknown error'
          try {
            const errorData = await response.json()
            errorText = errorData.error || errorData.message || JSON.stringify(errorData)
          } catch {
            errorText = await response.text().catch(() => 'Unknown error')
          }
          
          return NextResponse.json(
            { 
              error: errorText,
              status: response.status,
              statusText: response.statusText
            },
            { status: response.status }
          )
        }

        const data = await response.json()
        return NextResponse.json(data)
      } catch (fetchError) {
        clearTimeout(timeoutId)
        if (fetchError instanceof Error && fetchError.name === 'AbortError') {
          return NextResponse.json(
            { error: 'Превышено время ожидания ответа от сервера (10 секунд)' },
            { status: 504 }
          )
        }
        throw fetchError
      }
    }

    // Fallback: возвращаем ошибку, если нет параметров
    return NextResponse.json(
      { error: 'client_id and project_id are required' },
      { status: 400 }
    )
  } catch (error) {
    if (error instanceof Error && error.name === 'AbortError') {
      return NextResponse.json(
        { error: 'Превышено время ожидания ответа от сервера' },
        { status: 504 }
      )
    }
    console.error('Error stopping counterparties normalization:', error)
    return NextResponse.json(
      {
        error: error instanceof Error ? error.message : 'Unknown error',
      },
      { status: 500 }
    )
  }
}

