'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { Loader2, CheckCircle2, AlertCircle, GitMerge, AlertTriangle, Lightbulb } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface QualityAnalysisStatus {
  is_running: boolean
  progress: number
  processed: number
  total: number
  current_step: string
  duplicates_found: number
  violations_found: number
  suggestions_found: number
  error?: string
}

interface QualityAnalysisProgressProps {
  onComplete?: () => void
}

export function QualityAnalysisProgress({ onComplete }: QualityAnalysisProgressProps) {
  const [status, setStatus] = useState<QualityAnalysisStatus | null>(null)
  const [loading, setLoading] = useState(true)

  const fetchStatus = async () => {
    try {
      const response = await fetch('/api/quality/analyze/status')
      if (response.ok) {
        const data = await response.json()
        setStatus(data)
        setLoading(false)

        // Если анализ завершен, вызываем callback
        if (!data.is_running && data.current_step === 'completed' && onComplete) {
          onComplete()
        }
      }
    } catch (error) {
      console.error('Error fetching analysis status:', error)
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 2000) // Опрашиваем каждые 2 секунды
    return () => clearInterval(interval)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (loading && !status) {
    return (
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (!status) {
    return null
  }

  const getStepLabel = (step: string) => {
    switch (step) {
      case 'initializing':
        return 'Инициализация'
      case 'duplicates':
        return 'Поиск дубликатов'
      case 'violations':
        return 'Проверка нарушений'
      case 'suggestions':
        return 'Генерация предложений'
      case 'completed':
        return 'Завершено'
      default:
        return step
    }
  }

  const getStepIcon = (step: string) => {
    switch (step) {
      case 'duplicates':
        return GitMerge
      case 'violations':
        return AlertTriangle
      case 'suggestions':
        return Lightbulb
      case 'completed':
        return CheckCircle2
      default:
        return Loader2
    }
  }

  const StepIcon = getStepIcon(status.current_step)

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>Анализ качества данных</CardTitle>
            <CardDescription>Выполняется анализ выбранной таблицы</CardDescription>
          </div>
          <Badge variant={status.is_running ? 'default' : 'secondary'}>
            {status.is_running ? (
              <>
                <Loader2 className="mr-2 h-3 w-3 animate-spin" />
                Выполняется
              </>
            ) : (
              <>
                <CheckCircle2 className="mr-2 h-3 w-3" />
                Завершено
              </>
            )}
          </Badge>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {status.error && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{status.error}</AlertDescription>
          </Alert>
        )}

        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <div className="flex items-center gap-2">
              <StepIcon className={`h-4 w-4 ${status.is_running ? 'animate-spin' : ''}`} />
              <span className="font-medium">{getStepLabel(status.current_step)}</span>
            </div>
            <span className="text-muted-foreground">
              {status.processed} / {status.total}
            </span>
          </div>
          <Progress value={status.progress} className="w-full" />
          <div className="text-xs text-muted-foreground text-right">
            {status.progress.toFixed(1)}%
          </div>
        </div>

        {!status.is_running && status.current_step === 'completed' && (
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t">
            <div className="flex items-center gap-2">
              <GitMerge className="h-5 w-5 text-blue-500" />
              <div>
                <div className="text-sm font-medium">Дубликаты</div>
                <div className="text-2xl font-bold">{status.duplicates_found}</div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-orange-500" />
              <div>
                <div className="text-sm font-medium">Нарушения</div>
                <div className="text-2xl font-bold">{status.violations_found}</div>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Lightbulb className="h-5 w-5 text-yellow-500" />
              <div>
                <div className="text-sm font-medium">Предложения</div>
                <div className="text-2xl font-bold">{status.suggestions_found}</div>
              </div>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

