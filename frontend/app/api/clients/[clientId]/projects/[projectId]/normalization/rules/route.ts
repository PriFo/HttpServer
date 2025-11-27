import { NextRequest, NextResponse } from 'next/server'
import { getBackendUrl } from '@/lib/api-config'

const BACKEND_URL = getBackendUrl()

export async function GET(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const searchParams = request.nextUrl.searchParams
    const ruleType = searchParams.get('type')

    // Получаем правила из конфигурации нормализации
    const configResponse = await fetch(
      `${BACKEND_URL}/api/normalization/config?client_id=${clientId}&project_id=${projectId}`,
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
        cache: 'no-store',
      }
    )

    if (!configResponse.ok) {
      // Возвращаем пустой список правил
      return NextResponse.json({ rules: [] })
    }

    const configData = await configResponse.json()
    
    // Извлекаем правила из конфигурации
    const rules: any[] = []
    
    // Правила наименования из паттернов
    if (configData.patterns) {
      configData.patterns.forEach((pattern: any, index: number) => {
        if (!ruleType || ruleType === 'naming') {
          rules.push({
            id: `rule-naming-${index}`,
            type: 'naming',
            name: pattern.name || `Правило ${index + 1}`,
            pattern: pattern.pattern,
            replacement: pattern.replacement,
            enabled: pattern.enabled !== false,
            priority: pattern.priority || 0,
            definition: JSON.stringify(pattern),
          })
        }
      })
    }

    // Правила категоризации
    if (configData.categorization && (!ruleType || ruleType === 'categorization')) {
      rules.push({
        id: 'rule-categorization-1',
        type: 'categorization',
        name: 'Автоматическая категоризация',
        enabled: true,
        priority: 0,
        definition: JSON.stringify(configData.categorization),
      })
    }

    // Фильтруем по типу если указан
    const filteredRules = ruleType
      ? rules.filter(r => r.type === ruleType)
      : rules

    return NextResponse.json({ rules: filteredRules })
  } catch (error) {
    console.error('Error fetching rules:', error)
    return NextResponse.json({ rules: [] })
  }
}

export async function POST(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const body = await request.json()

    // Создание нового правила
    // В реальном приложении здесь будет сохранение в конфигурацию
    const newRule = {
      id: `rule-${Date.now()}`,
      ...body,
      created_at: new Date().toISOString(),
    }

    return NextResponse.json({ rule: newRule })
  } catch (error) {
    console.error('Error creating rule:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function PUT(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    const { clientId, projectId } = await params
    const body = await request.json()
    const { ruleId } = body

    // Обновление правила
    // В реальном приложении здесь будет обновление в конфигурации
    const updatedRule = {
      ...body,
      updated_at: new Date().toISOString(),
    }

    return NextResponse.json({ rule: updatedRule })
  } catch (error) {
    console.error('Error updating rule:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

export async function DELETE(
  request: NextRequest,
  { params }: { params: Promise<{ clientId: string; projectId: string }> }
) {
  try {
    await params // params не используется, но нужно await для соответствия типу
    const searchParams = request.nextUrl.searchParams
    const ruleId = searchParams.get('ruleId')

    if (!ruleId) {
      return NextResponse.json(
        { error: 'ruleId is required' },
        { status: 400 }
      )
    }

    // Удаление правила
    // В реальном приложении здесь будет удаление из конфигурации
    return NextResponse.json({ success: true })
  } catch (error) {
    console.error('Error deleting rule:', error)
    return NextResponse.json(
      { error: 'Internal server error' },
      { status: 500 }
    )
  }
}

