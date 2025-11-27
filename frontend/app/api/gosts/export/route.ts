import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function GET(request: NextRequest) {
  try {
    // Получаем параметры запроса из URL
    const searchParams = request.nextUrl.searchParams
    
    // Формируем URL для бэкенда с теми же параметрами
    const backendUrl = new URL(`${BACKEND_URL}/api/gosts/export`)
    searchParams.forEach((value, key) => {
      backendUrl.searchParams.append(key, value)
    })

    // Выполняем запрос к бэкенду
    const response = await fetch(backendUrl.toString(), {
      method: 'GET',
      headers: {
        'Accept': 'text/csv',
      },
    })

    if (!response.ok) {
      const errorText = await response.text()
      let errorData
      try {
        errorData = JSON.parse(errorText)
      } catch {
        errorData = { error: errorText || 'Ошибка экспорта ГОСТов' }
      }
      
      return NextResponse.json(
        { error: errorData.error || 'Не удалось экспортировать ГОСТы' },
        { status: response.status }
      )
    }

    // Получаем CSV данные
    const csvData = await response.text()
    
    // Получаем имя файла из заголовка Content-Disposition или генерируем
    const contentDisposition = response.headers.get('Content-Disposition')
    let filename = `gosts_export_${new Date().toISOString().split('T')[0]}.csv`
    if (contentDisposition) {
      const filenameMatch = contentDisposition.match(/filename[^;=\n]*=((['"]).*?\2|[^;\n]*)/)
      if (filenameMatch && filenameMatch[1]) {
        filename = filenameMatch[1].replace(/['"]/g, '')
      }
    }

    // Возвращаем CSV файл
    return new NextResponse(csvData, {
      status: 200,
      headers: {
        'Content-Type': 'text/csv; charset=utf-8',
        'Content-Disposition': `attachment; filename="${filename}"`,
        'Content-Transfer-Encoding': 'binary',
      },
    })
  } catch (error) {
    console.error('[GOST Export API] Error:', error)
    return NextResponse.json(
      { error: error instanceof Error ? error.message : 'Внутренняя ошибка сервера' },
      { status: 500 }
    )
  }
}

