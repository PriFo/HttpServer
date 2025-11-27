'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Trash2, Merge, Tag, Download, Upload, RefreshCw } from 'lucide-react'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'

interface BatchOperationsProps {
  clientId: string
  projectId: string
  selectedItems?: string[]
  onSelectionChange?: (items: string[]) => void
}

export const BatchOperations: React.FC<BatchOperationsProps> = ({
  clientId,
  projectId,
  selectedItems = [],
  onSelectionChange,
}) => {
  const [operation, setOperation] = useState<'merge' | 'delete' | 'tag' | 'export' | 'import' | null>(null)
  const [isProcessing, setIsProcessing] = useState(false)
  const [progress, setProgress] = useState(0)
  const [showConfirmDialog, setShowConfirmDialog] = useState(false)

  const handleOperation = async () => {
    if (selectedItems.length === 0) {
      toast.error('Выберите элементы для операции')
      return
    }

    if (!operation) {
      toast.error('Выберите операцию')
      return
    }

    setShowConfirmDialog(true)
  }

  const confirmOperation = async () => {
    setShowConfirmDialog(false)
    setIsProcessing(true)
    setProgress(0)

    try {
      // Имитация прогресса
      const progressInterval = setInterval(() => {
        setProgress(prev => {
          if (prev >= 90) {
            clearInterval(progressInterval)
            return prev
          }
          return prev + 10
        })
      }, 200)

      // Выполнение операции
      const response = await fetch(
        `/api/clients/${clientId}/projects/${projectId}/normalization/batch`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            operation,
            items: selectedItems,
          }),
        }
      )

      clearInterval(progressInterval)
      setProgress(100)

      if (response.ok) {
        toast.success(`Операция "${operation ? getOperationName(operation) : 'неизвестная'}" выполнена успешно`)
        if (onSelectionChange) {
          onSelectionChange([])
        }
      } else {
        throw new Error('Operation failed')
      }
    } catch (error) {
      console.error('Batch operation error:', error)
      toast.error('Ошибка при выполнении операции')
    } finally {
      setIsProcessing(false)
      setTimeout(() => setProgress(0), 1000)
    }
  }

  const getOperationName = (op: string) => {
    const names: Record<string, string> = {
      merge: 'Объединение',
      delete: 'Удаление',
      tag: 'Тегирование',
      export: 'Экспорт',
      import: 'Импорт',
    }
    return names[op] || op
  }

  const getOperationIcon = (op: string) => {
    switch (op) {
      case 'merge':
        return <Merge className="h-4 w-4" />
      case 'delete':
        return <Trash2 className="h-4 w-4" />
      case 'tag':
        return <Tag className="h-4 w-4" />
      case 'export':
        return <Download className="h-4 w-4" />
      case 'import':
        return <Upload className="h-4 w-4" />
      default:
        return null
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Массовые операции</CardTitle>
        <CardDescription>
          Выполнение операций над выбранными элементами ({selectedItems.length})
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-2">
          <Label>Выберите операцию</Label>
          <Select value={operation || undefined} onValueChange={(v: any) => setOperation(v)}>
            <SelectTrigger>
              <SelectValue placeholder="Выберите операцию" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="merge">
                <div className="flex items-center gap-2">
                  <Merge className="h-4 w-4" />
                  <span>Объединить</span>
                </div>
              </SelectItem>
              <SelectItem value="tag">
                <div className="flex items-center gap-2">
                  <Tag className="h-4 w-4" />
                  <span>Добавить теги</span>
                </div>
              </SelectItem>
              <SelectItem value="export">
                <div className="flex items-center gap-2">
                  <Download className="h-4 w-4" />
                  <span>Экспортировать</span>
                </div>
              </SelectItem>
              <SelectItem value="delete">
                <div className="flex items-center gap-2">
                  <Trash2 className="h-4 w-4" />
                  <span>Удалить</span>
                </div>
              </SelectItem>
            </SelectContent>
          </Select>
        </div>

        {selectedItems.length > 0 && (
          <div className="flex items-center justify-between p-3 bg-muted rounded-lg">
            <span className="text-sm font-medium">
              Выбрано элементов: {selectedItems.length}
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={() => onSelectionChange && onSelectionChange([])}
            >
              Очистить выбор
            </Button>
          </div>
        )}

        {isProcessing && (
          <div className="space-y-2">
            <div className="flex items-center justify-between text-sm">
              <span>Выполнение операции...</span>
              <span className="font-medium">{progress}%</span>
            </div>
            <Progress value={progress} />
          </div>
        )}

        <Button
          onClick={handleOperation}
          disabled={selectedItems.length === 0 || !operation || isProcessing}
          className="w-full"
        >
          {operation && getOperationIcon(operation)}
          <span className="ml-2">
            {operation ? `Выполнить: ${getOperationName(operation)}` : 'Выберите операцию'}
          </span>
        </Button>

        <Dialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Подтверждение операции</DialogTitle>
              <DialogDescription>
                Вы уверены, что хотите выполнить операцию "{getOperationName(operation || '')}" 
                над {selectedItems.length} элементами?
              </DialogDescription>
            </DialogHeader>
            <div className="flex gap-2 justify-end pt-4">
              <Button variant="outline" onClick={() => setShowConfirmDialog(false)}>
                Отмена
              </Button>
              <Button onClick={confirmOperation}>
                Подтвердить
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </CardContent>
    </Card>
  )
}

