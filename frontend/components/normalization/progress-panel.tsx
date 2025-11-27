"use client"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"

interface ProgressPanelProps {
  status: {
    currentStep: string
    processed: number
    total: number
    progress: number
  }
}

export function ProgressPanel({ status }: ProgressPanelProps) {
  const steps = [
    { id: 'cleaning', name: 'Очистка данных', description: 'Удаление старых записей' },
    { id: 'normalizing', name: 'Нормализация имен', description: 'Обработка наименований' },
    { id: 'categorizing', name: 'Категоризация', description: 'Распределение по категориям' },
    { id: 'grouping', name: 'Группировка', description: 'Объединение похожих товаров' },
    { id: 'saving', name: 'Сохранение', description: 'Запись в базу данных' },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>Ход выполнения</CardTitle>
        <CardDescription>
          Детальная информация о текущем этапе
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {steps.map((step, index) => (
          <div key={step.id} className="flex items-center space-x-4">
            <div className={`w-3 h-3 rounded-full ${
              status.currentStep.includes(step.name) 
                ? 'bg-blue-500 animate-pulse' 
                : 'bg-gray-300'
            }`} />
            <div className="flex-1">
              <div className="font-medium">{step.name}</div>
              <div className="text-sm text-muted-foreground">{step.description}</div>
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

