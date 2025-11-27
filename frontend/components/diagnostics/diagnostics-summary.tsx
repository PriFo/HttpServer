'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { AlertCircle, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface DiagnosticsSummaryProps {
  databases: {
    total: number
    healthy: number
    issues: number
  }
  uploads: {
    total: number
    valid: number
    missing: number
    invalid: number
  }
  extraction: {
    total: number
    extracted: number
    notExtracted: number
  }
  normalization: {
    hasData: boolean
    recordsCount: number
    sessionsCount: number
  }
}

export function DiagnosticsSummary({ databases, uploads, extraction, normalization }: DiagnosticsSummaryProps) {
  const getOverallStatus = () => {
    const hasIssues = 
      databases.issues > 0 ||
      uploads.missing > 0 ||
      uploads.invalid > 0 ||
      extraction.notExtracted > 0 ||
      !normalization.hasData

    if (hasIssues) {
      return {
        status: 'warning',
        icon: AlertTriangle,
        message: 'Обнаружены проблемы',
        color: 'text-yellow-600 dark:text-yellow-400'
      }
    }
    return {
      status: 'ok',
      icon: CheckCircle2,
      message: 'Все проверки пройдены',
      color: 'text-green-600 dark:text-green-400'
    }
  }

  const overall = getOverallStatus()
  const Icon = overall.icon

  return (
    <Card className="border-l-4 border-l-blue-500">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Icon className={`h-5 w-5 ${overall.color}`} />
          Общая сводка диагностики
        </CardTitle>
        <CardDescription>
          Статус всех компонентов цепочки данных
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {/* Базы данных */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">Базы данных</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{databases.total}</div>
              <div className="flex gap-2 mt-2">
                <Badge variant="default" className="bg-green-600">
                  OK: {databases.healthy}
                </Badge>
                {databases.issues > 0 && (
                  <Badge variant="destructive">
                    Проблемы: {databases.issues}
                  </Badge>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Upload записи */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">Upload записи</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{uploads.total}</div>
              <div className="flex flex-wrap gap-2 mt-2">
                <Badge variant="default" className="bg-green-600">
                  Валидных: {uploads.valid}
                </Badge>
                {uploads.missing > 0 && (
                  <Badge variant="secondary">
                    Отсутствует: {uploads.missing}
                  </Badge>
                )}
                {uploads.invalid > 0 && (
                  <Badge variant="destructive">
                    Невалидных: {uploads.invalid}
                  </Badge>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Извлечение данных */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">Извлечение данных</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{extraction.total}</div>
              <div className="flex gap-2 mt-2">
                <Badge variant="default" className="bg-green-600">
                  Извлечено: {extraction.extracted}
                </Badge>
                {extraction.notExtracted > 0 && (
                  <Badge variant="destructive">
                    Не извлечено: {extraction.notExtracted}
                  </Badge>
                )}
              </div>
            </CardContent>
          </Card>

          {/* Нормализация */}
          <Card>
            <CardHeader className="pb-2">
              <CardTitle className="text-sm text-muted-foreground">Нормализация</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">
                {normalization.recordsCount.toLocaleString('ru-RU')}
              </div>
              <div className="flex gap-2 mt-2">
                {normalization.hasData ? (
                  <Badge variant="default" className="bg-green-600">
                    Данные есть
                  </Badge>
                ) : (
                  <Badge variant="destructive">
                    Нет данных
                  </Badge>
                )}
                <Badge variant="outline">
                  Сессий: {normalization.sessionsCount}
                </Badge>
              </div>
            </CardContent>
          </Card>
        </div>

        {overall.status === 'warning' && (
          <Alert variant="destructive" className="mt-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Обнаружены проблемы в цепочке данных. Используйте соответствующие вкладки для детальной диагностики и исправления.
            </AlertDescription>
          </Alert>
        )}
      </CardContent>
    </Card>
  )
}

