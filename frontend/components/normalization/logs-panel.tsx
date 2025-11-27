"use client"

import { useEffect, useRef, useState, useMemo, useCallback } from "react"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { AlertCircle, CheckCircle2, AlertTriangle, Info, ChevronDown } from "lucide-react"

interface LogsPanelProps {
  logs: string[]
  title?: string
  description?: string
}

function getLogIcon(log: string) {
  const lowerLog = log.toLowerCase()
  if (lowerLog.includes('ошибка') || lowerLog.includes('❌') || lowerLog.includes('error')) {
    return <AlertCircle className="h-3 w-3 text-red-500" />
  }
  if (lowerLog.includes('✅') || lowerLog.includes('завершена') || lowerLog.includes('завершен') || lowerLog.includes('успешно')) {
    return <CheckCircle2 className="h-3 w-3 text-green-500" />
  }
  if (lowerLog.includes('⚠') || lowerLog.includes('warning') || lowerLog.includes('предупреждение')) {
    return <AlertTriangle className="h-3 w-3 text-yellow-500" />
  }
  return <Info className="h-3 w-3 text-blue-500" />
}

function getLogColor(log: string) {
  const lowerLog = log.toLowerCase()
  if (lowerLog.includes('ошибка') || lowerLog.includes('❌') || lowerLog.includes('error')) {
    return 'text-red-600 dark:text-red-400'
  }
  if (lowerLog.includes('✅') || lowerLog.includes('завершена') || lowerLog.includes('завершен') || lowerLog.includes('успешно')) {
    return 'text-green-600 dark:text-green-400'
  }
  if (lowerLog.includes('⚠') || lowerLog.includes('warning') || lowerLog.includes('предупреждение')) {
    return 'text-yellow-600 dark:text-yellow-400'
  }
  return 'text-muted-foreground'
}

export function LogsPanel({ logs, title = "Логи выполнения", description = "Реальная информация о процессе нормализации" }: LogsPanelProps) {
  const scrollAreaRef = useRef<HTMLDivElement>(null)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const shouldAutoScrollRef = useRef(true)
  const [showScrollButton, setShowScrollButton] = useState(false)

  // Проверка, находится ли пользователь внизу области прокрутки
  const checkIfShouldAutoScroll = () => {
    if (!scrollAreaRef.current) return false
    
    // Находим внутренний элемент ScrollArea
    const scrollContainer = scrollAreaRef.current.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement
    if (!scrollContainer) return false
    
    const { scrollTop, scrollHeight, clientHeight } = scrollContainer
    // Прокручиваем, если пользователь внизу (с допуском 50px)
    return scrollHeight - scrollTop - clientHeight < 50
  }

  // Обработчик прокрутки - отслеживаем, когда пользователь прокручивает вверх
  useEffect(() => {
    const scrollContainer = scrollAreaRef.current?.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement
    if (!scrollContainer) return

    // Инициализируем состояние при монтировании
    shouldAutoScrollRef.current = checkIfShouldAutoScroll()

    const handleScroll = () => {
      const isAtBottom = checkIfShouldAutoScroll()
      shouldAutoScrollRef.current = isAtBottom
      setShowScrollButton(!isAtBottom && logs.length > 0)
    }

    scrollContainer.addEventListener('scroll', handleScroll)
    // Проверяем начальное состояние
    setTimeout(() => {
      const isAtBottom = checkIfShouldAutoScroll()
      setShowScrollButton(!isAtBottom && logs.length > 0)
    }, 100)
    
    return () => scrollContainer.removeEventListener('scroll', handleScroll)
  }, [logs.length])

  // Функция для прокрутки вниз (оптимизирована с useCallback)
  const scrollToBottom = useCallback(() => {
    const scrollContainer = scrollAreaRef.current?.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement
    if (!scrollContainer) return
    
    scrollContainer.scrollTo({
      top: scrollContainer.scrollHeight,
      behavior: "smooth"
    })
    shouldAutoScrollRef.current = true
    setShowScrollButton(false)
  }, [])

  // Автопрокрутка к последнему логу только если пользователь внизу
  useEffect(() => {
    if (logs.length === 0) {
      setShowScrollButton(false)
      return
    }
    
    // Проверяем, нужно ли прокручивать
    if (!shouldAutoScrollRef.current) {
      setShowScrollButton(true)
      return
    }
    
    const scrollContainer = scrollAreaRef.current?.querySelector('[data-radix-scroll-area-viewport]') as HTMLElement
    if (!scrollContainer || !logsEndRef.current) return
    
    // Небольшая задержка для корректной работы с обновлением DOM
    setTimeout(() => {
      // Повторно проверяем, что пользователь все еще внизу
      if (shouldAutoScrollRef.current && logsEndRef.current) {
        // Прокручиваем viewport напрямую вниз
        scrollContainer.scrollTo({
          top: scrollContainer.scrollHeight,
          behavior: "smooth"
        })
        setShowScrollButton(false)
      } else {
        setShowScrollButton(true)
      }
    }, 10)
  }, [logs])

  // Оптимизация: используем useMemo для подсчета статистики
  const { errorCount, successCount } = useMemo(() => {
    const errors = logs.filter(log => {
      const lowerLog = log.toLowerCase()
      return lowerLog.includes('ошибка') || lowerLog.includes('❌') || lowerLog.includes('error')
    }).length
    
    const successes = logs.filter(log => {
      const lowerLog = log.toLowerCase()
      return lowerLog.includes('✅') || lowerLog.includes('завершена') || lowerLog.includes('завершен') || lowerLog.includes('успешно')
    }).length
    
    return { errorCount: errors, successCount: successes }
  }, [logs])

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle>{title}</CardTitle>
            <CardDescription>
              {description}
            </CardDescription>
          </div>
          {logs.length > 0 && (
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="text-xs">
                {logs.length > 1000 ? `Показано: ${logs.length - 1000 + 1}-${logs.length} из ${logs.length}` : `Всего: ${logs.length}`}
              </Badge>
              {errorCount > 0 && (
                <Badge variant="destructive" className="text-xs">
                  {errorCount} ошибок
                </Badge>
              )}
              {successCount > 0 && (
                <Badge variant="default" className="text-xs bg-green-500">
                  {successCount} успешно
                </Badge>
              )}
            </div>
          )}
        </div>
      </CardHeader>
      <CardContent className="relative">
        {logs.length > 1000 && (
          <div className="mb-2 p-2 bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 rounded text-xs text-yellow-800 dark:text-yellow-200">
            ⚠ Показаны только последние 1000 логов из {logs.length} для оптимизации производительности
          </div>
        )}
        <ScrollArea className="h-96 rounded-md border p-4" ref={scrollAreaRef}>
          <div className="space-y-1">
            {logs.length === 0 ? (
              <div className="text-center text-muted-foreground py-8">
                Логи появятся здесь после запуска процесса
              </div>
            ) : (
              // Ограничиваем отображение последних 1000 логов для производительности
              (() => {
                const displayLogs = logs.length > 1000 ? logs.slice(-1000) : logs
                const startIndex = Math.max(0, logs.length - 1000)
                return displayLogs.map((log, index) => {
                  const actualIndex = startIndex + index
                  // Используем комбинацию индекса и хеша для уникального ключа
                  const logHash = log.length > 0 ? log.substring(0, 30).replace(/\s/g, '') : ''
                  return (
                    <div 
                      key={`log-${actualIndex}-${logHash}`} 
                      className={`flex items-start gap-2 py-1 text-sm font-mono ${getLogColor(log)}`}
                    >
                      <div className="mt-0.5 flex-shrink-0">
                        {getLogIcon(log)}
                      </div>
                      <span className="flex-1 break-words">{log}</span>
                    </div>
                  )
                })
              })()
            )}
            <div ref={logsEndRef} />
          </div>
        </ScrollArea>
        {showScrollButton && (
          <div className="absolute bottom-4 right-4">
            <Button
              size="sm"
              variant="outline"
              onClick={scrollToBottom}
              className="shadow-lg"
            >
              <ChevronDown className="h-4 w-4 mr-1" />
              Прокрутить вниз
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

