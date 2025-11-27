'use client'

import { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Badge } from '@/components/ui/badge'
import { Loader2, CheckCircle2, AlertCircle, GitMerge, AlertTriangle, Lightbulb, Activity, Clock } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { motion, AnimatePresence } from 'framer-motion'
import { toast } from 'sonner'
import { useError } from '@/contexts/ErrorContext'
import { QUALITY_TIMEOUTS } from '@/lib/quality-constants'
import { fetchJson, getErrorMessage, isTimeoutError, isNetworkError } from '@/lib/fetch-utils'

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
  start_time?: string
}

interface QualityAnalysisProgressProps {
  onComplete?: () => void
}

export function QualityAnalysisProgress({ onComplete }: QualityAnalysisProgressProps) {
  const { handleError } = useError()
  const [status, setStatus] = useState<QualityAnalysisStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [wasRunning, setWasRunning] = useState(false)

  const fetchStatus = async () => {
    try {
      const data = await fetchJson<QualityAnalysisStatus>(
        '/api/quality/analyze/status',
        {
          timeout: QUALITY_TIMEOUTS.FAST,
          cache: 'no-store',
        }
      );

      const prevRunning = status?.is_running || false;
      
      setStatus(data);
      setLoading(false);

      // Показываем уведомление при завершении анализа
      if (wasRunning && !data.is_running && data.current_step === 'completed') {
        toast.success('Анализ качества завершен', {
          description: `Найдено: ${data.duplicates_found} дубликатов, ${data.violations_found} нарушений, ${data.suggestions_found} предложений`,
        });
        setWasRunning(false);
      }
      
      // Обновляем флаг выполнения
      if (data.is_running && !wasRunning) {
        setWasRunning(true);
      }

      // Если анализ завершен, вызываем callback
      if (!data.is_running && data.current_step === 'completed' && onComplete) {
        onComplete();
      }
    } catch (fetchError) {
      // Игнорируем таймауты и сетевые ошибки - не критично для статуса
      if (isTimeoutError(fetchError) || isNetworkError(fetchError)) {
        console.log(`[QualityAnalysisProgress] Request error (ignored): ${getErrorMessage(fetchError)}`);
        return;
      }
      
      // Игнорируем ошибки сети, если статус уже был загружен
      if (status) {
        return;
      }
      
      handleError(fetchError, 'Не удалось получить статус анализа качества');
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchStatus();
    
    // Оптимизация: обновляем чаще только когда анализ запущен
    // Если анализ не запущен - обновляем реже (каждые 5 секунд)
    let interval: NodeJS.Timeout | null = null;
    
    const setupInterval = () => {
      if (interval) {
        clearInterval(interval);
      }
      
      const intervalTime = status?.is_running ? 2000 : 5000; // 2 сек если запущен, 5 сек если нет
      
      interval = setInterval(() => {
        fetchStatus();
      }, intervalTime);
    };
    
    setupInterval();
    
    return () => {
      if (interval) {
        clearInterval(interval);
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [status?.is_running]);

  if (loading && !status) {
    return (
      <Card className="border-primary/20 shadow-lg">
        <CardContent className="pt-6">
          <div className="flex flex-col items-center justify-center py-8 space-y-4">
            <div className="relative">
                <div className="absolute inset-0 rounded-full bg-primary/20 animate-ping"></div>
                <Loader2 className="h-8 w-8 animate-spin text-primary relative z-10" />
            </div>
            <p className="text-sm text-muted-foreground animate-pulse">Подключение к серверу...</p>
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
      case 'initializing': return 'Инициализация'
      case 'duplicates': return 'Поиск дубликатов'
      case 'violations': return 'Проверка нарушений'
      case 'suggestions': return 'Генерация предложений'
      case 'completed': return 'Завершено'
      default: return step
    }
  }

  const getStepIcon = (step: string) => {
    switch (step) {
      case 'duplicates': return GitMerge
      case 'violations': return AlertTriangle
      case 'suggestions': return Lightbulb
      case 'completed': return CheckCircle2
      default: return Activity
    }
  }

  const StepIcon = getStepIcon(status.current_step)

  // Calculate estimated time remaining or elapsed time if needed
  // For now, simpler is better

  return (
    <AnimatePresence>
        <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: -20 }}
            transition={{ duration: 0.3 }}
        >
            <Card className={`border-l-4 transition-all duration-300 ${status.is_running ? 'border-l-blue-500 shadow-md' : 'border-l-green-500 shadow-sm'}`}>
            <CardHeader>
                <div className="flex items-center justify-between">
                <div className="space-y-1">
                    <CardTitle className="flex items-center gap-2">
                        Анализ качества данных
                        {status.is_running && (
                            <span className="relative flex h-2 w-2 ml-1">
                            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-blue-400 opacity-75"></span>
                            <span className="relative inline-flex rounded-full h-2 w-2 bg-blue-500"></span>
                            </span>
                        )}
                    </CardTitle>
                    <CardDescription>Выполняется анализ выбранной таблицы</CardDescription>
                </div>
                <Badge variant={status.is_running ? 'default' : 'secondary'} className={`px-3 py-1 ${status.is_running ? 'bg-blue-500 hover:bg-blue-600' : 'bg-green-500 hover:bg-green-600 text-white'}`}>
                    {status.is_running ? (
                    <span className="flex items-center gap-1.5">
                        <Loader2 className="h-3 w-3 animate-spin" />
                        Выполняется
                    </span>
                    ) : (
                    <span className="flex items-center gap-1.5">
                        <CheckCircle2 className="h-3 w-3" />
                        Завершено
                    </span>
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

                <div className="space-y-3">
                    <div className="flex items-center justify-between text-sm">
                        <div className="flex items-center gap-2.5 text-primary font-medium">
                            <div className={`p-1.5 rounded-full ${status.is_running ? 'bg-blue-100 text-blue-600' : 'bg-green-100 text-green-600'}`}>
                                <StepIcon className={`h-4 w-4 ${status.is_running && status.current_step !== 'completed' ? 'animate-spin-slow' : ''}`} />
                            </div>
                            <span>{getStepLabel(status.current_step)}</span>
                        </div>
                        <span className="text-muted-foreground font-mono text-xs bg-muted px-2 py-1 rounded">
                            {status.processed.toLocaleString()} / {status.total.toLocaleString()}
                        </span>
                    </div>
                    
                    <div className="relative">
                        <Progress value={isNaN(status.progress) ? 0 : Math.max(0, Math.min(100, status.progress))} className="h-3 rounded-full w-full transition-all duration-500" />
                    </div>
                    
                    <div className="flex justify-end">
                        <span className="text-xs font-bold text-muted-foreground">
                            {(isNaN(status.progress) ? 0 : status.progress).toFixed(1)}%
                        </span>
                    </div>
                </div>

                {(!status.is_running && status.current_step === 'completed') || (status.progress > 0) ? (
                <motion.div 
                    className="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t"
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    transition={{ delay: 0.2 }}
                >
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-blue-50/50 border border-blue-100 hover:bg-blue-50 transition-colors">
                        <div className="p-2 bg-blue-100 rounded-full text-blue-600">
                            <GitMerge className="h-5 w-5" />
                        </div>
                        <div>
                            <div className="text-xs font-medium text-blue-900 uppercase tracking-wider">Дубликаты</div>
                            <div className="text-2xl font-bold text-blue-700">{status.duplicates_found.toLocaleString()}</div>
                        </div>
                    </div>
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-orange-50/50 border border-orange-100 hover:bg-orange-50 transition-colors">
                        <div className="p-2 bg-orange-100 rounded-full text-orange-600">
                            <AlertTriangle className="h-5 w-5" />
                        </div>
                        <div>
                            <div className="text-xs font-medium text-orange-900 uppercase tracking-wider">Нарушения</div>
                            <div className="text-2xl font-bold text-orange-700">{status.violations_found.toLocaleString()}</div>
                        </div>
                    </div>
                    <div className="flex items-center gap-3 p-3 rounded-lg bg-yellow-50/50 border border-yellow-100 hover:bg-yellow-50 transition-colors">
                         <div className="p-2 bg-yellow-100 rounded-full text-yellow-600">
                            <Lightbulb className="h-5 w-5" />
                        </div>
                        <div>
                            <div className="text-xs font-medium text-yellow-900 uppercase tracking-wider">Предложения</div>
                            <div className="text-2xl font-bold text-yellow-700">{status.suggestions_found.toLocaleString()}</div>
                        </div>
                    </div>
                </motion.div>
                ) : null}
            </CardContent>
            </Card>
        </motion.div>
    </AnimatePresence>
  )
}
