'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Edit, Save, X, Plus } from 'lucide-react'
import { toast } from 'sonner'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'

interface BulkEditProps {
  clientId: string
  projectId: string
  selectedItems: string[]
  onSave?: (updates: Record<string, any>) => void
}

export const BulkEdit: React.FC<BulkEditProps> = ({
  clientId,
  projectId,
  selectedItems,
  onSave,
}) => {
  const [edits, setEdits] = useState<Record<string, any>>({})
  const [isProcessing, setIsProcessing] = useState(false)
  const [progress, setProgress] = useState(0)
  const [showDialog, setShowDialog] = useState(false)

  const handleFieldChange = (field: string, value: any) => {
    setEdits(prev => ({
      ...prev,
      [field]: value,
    }))
  }

  const handleSave = async () => {
    if (selectedItems.length === 0) {
      toast.error('Выберите элементы для редактирования')
      return
    }

    if (Object.keys(edits).length === 0) {
      toast.error('Внесите изменения перед сохранением')
      return
    }

    setShowDialog(true)
  }

  const confirmSave = async () => {
    setShowDialog(false)
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

      const response = await fetch(
        `/api/clients/${clientId}/projects/${projectId}/normalization/bulk-edit`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          body: JSON.stringify({
            items: selectedItems,
            updates: edits,
          }),
        }
      )

      clearInterval(progressInterval)
      setProgress(100)

      if (response.ok) {
        toast.success(`Изменения применены к ${selectedItems.length} элементам`)
        setEdits({})
        if (onSave) {
          onSave(edits)
        }
      } else {
        throw new Error('Save failed')
      }
    } catch (error) {
      console.error('Bulk edit error:', error)
      toast.error('Ошибка при сохранении изменений')
    } finally {
      setIsProcessing(false)
      setTimeout(() => setProgress(0), 1000)
    }
  }

  const commonFields = [
    { key: 'category', label: 'Категория', type: 'select', options: ['electronics', 'machinery', 'materials', 'services'] },
    { key: 'kpved_code', label: 'Код КПВЭД', type: 'text' },
    { key: 'tags', label: 'Теги', type: 'text', placeholder: 'Через запятую' },
    { key: 'notes', label: 'Примечания', type: 'textarea' },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center gap-2">
          <Edit className="h-5 w-5" />
          Массовое редактирование
        </CardTitle>
        <CardDescription>
          Редактирование общих полей для выбранных элементов ({selectedItems.length})
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {selectedItems.length === 0 ? (
          <div className="text-center py-8 text-muted-foreground">
            Выберите элементы для редактирования
          </div>
        ) : (
          <>
            <div className="space-y-4">
              {commonFields.map((field) => (
                <div key={field.key} className="space-y-2">
                  <Label>{field.label}</Label>
                  {field.type === 'select' ? (
                    <Select
                      value={edits[field.key] || undefined}
                      onValueChange={(v) => handleFieldChange(field.key, v)}
                    >
                      <SelectTrigger>
                        <SelectValue placeholder={`Выберите ${field.label.toLowerCase()}`} />
                      </SelectTrigger>
                      <SelectContent>
                        {field.options?.map((option) => (
                          <SelectItem key={option} value={option}>
                            {option}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : field.type === 'textarea' ? (
                    <Textarea
                      value={edits[field.key] || ''}
                      onChange={(e) => handleFieldChange(field.key, e.target.value)}
                      placeholder={field.placeholder}
                      rows={3}
                    />
                  ) : (
                    <Input
                      type="text"
                      value={edits[field.key] || ''}
                      onChange={(e) => handleFieldChange(field.key, e.target.value)}
                      placeholder={field.placeholder || `Введите ${field.label.toLowerCase()}`}
                    />
                  )}
                </div>
              ))}
            </div>

            {Object.keys(edits).length > 0 && (
              <div className="p-3 bg-muted rounded-lg">
                <div className="flex items-center justify-between mb-2">
                  <span className="text-sm font-medium">Изменения:</span>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setEdits({})}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
                <div className="space-y-1">
                  {Object.entries(edits).map(([key, value]) => (
                    <div key={key} className="text-xs flex items-center justify-between">
                      <span className="text-muted-foreground">{key}:</span>
                      <Badge variant="outline">{String(value)}</Badge>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {isProcessing && (
              <div className="space-y-2">
                <div className="flex items-center justify-between text-sm">
                  <span>Применение изменений...</span>
                  <span className="font-medium">{progress}%</span>
                </div>
                <Progress value={progress} />
              </div>
            )}

            <Button
              onClick={handleSave}
              disabled={selectedItems.length === 0 || Object.keys(edits).length === 0 || isProcessing}
              className="w-full"
            >
              <Save className="h-4 w-4 mr-2" />
              Применить к {selectedItems.length} элементам
            </Button>

            <Dialog open={showDialog} onOpenChange={setShowDialog}>
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>Подтверждение изменений</DialogTitle>
                  <DialogDescription>
                    Вы уверены, что хотите применить изменения к {selectedItems.length} элементам?
                    Это действие нельзя отменить.
                  </DialogDescription>
                </DialogHeader>
                <div className="flex gap-2 justify-end pt-4">
                  <Button variant="outline" onClick={() => setShowDialog(false)}>
                    Отмена
                  </Button>
                  <Button onClick={confirmSave}>
                    Применить
                  </Button>
                </div>
              </DialogContent>
            </Dialog>
          </>
        )}
      </CardContent>
    </Card>
  )
}

