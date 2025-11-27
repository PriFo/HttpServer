import { NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

export async function GET(request: Request) {
  try {
    const BACKEND_URL = getBackendUrl()
    const { searchParams } = new URL(request.url)
    const parent = searchParams.get('parent')
    const level = searchParams.get('level')

    let url = `${BACKEND_URL}/api/kpved/hierarchy`
    const params = new URLSearchParams()

    if (parent) params.append('parent', parent)
    if (level) params.append('level', level)

    if (params.toString()) {
      url += `?${params.toString()}`
    }

    console.log(`[KPVED API] Fetching from backend: ${url}`)
    const response = await fetch(url, {
      cache: 'no-store',
    })

    console.log(`[KPVED API] Backend response status: ${response.status}`)

    if (!response.ok) {
      // Для 404 возвращаем пустые данные вместо ошибки
      if (response.status === 404) {
        console.log('[KPVED API] 404 - returning empty array')
        return NextResponse.json({
          nodes: [],
          total: 0,
        })
      }
      
      let errorMessage = 'Failed to fetch KPVED hierarchy'
      try {
        const errorData = await response.json()
        errorMessage = errorData.error || errorMessage
        console.error(`[KPVED API] Backend error: ${errorMessage}`)
      } catch {
        const errorText = await response.text()
        if (errorText) {
          errorMessage = errorText
          console.error(`[KPVED API] Backend error (text): ${errorText}`)
        }
      }
      
      // Если таблица не существует, возвращаем пустой массив
      if (response.status === 500 && (errorMessage.includes('no such table') || errorMessage.includes('kpved_classifier'))) {
        console.log('[KPVED API] Table does not exist - returning empty array')
        return NextResponse.json({
          nodes: [],
          total: 0,
        })
      }
      
      return NextResponse.json(
        { error: errorMessage },
        { status: response.status }
      )
    }

    const data = await response.json()
    console.log(`[KPVED API] Success: received ${data.nodes?.length || 0} nodes, total: ${data.total || 0}`)
    return NextResponse.json(data)
  } catch (error) {
    console.error('[KPVED API] Error fetching KPVED hierarchy:', error)
    // При ошибке подключения возвращаем пустой массив вместо ошибки
    return NextResponse.json({
      nodes: [],
      total: 0,
    })
  }
}
