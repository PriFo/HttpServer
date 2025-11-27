'use client'

import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { apiPost } from '@/lib/api-client'

interface DuplicateItem {
  id: string
  name: string
  code?: string
  category?: string
}

interface CreateBenchmarkDialogProps {
  isOpen: boolean
  onClose: () => void
  uploadId: string
  duplicateItems: DuplicateItem[]
  onSuccess?: () => void
}

export function CreateBenchmarkDialog({
  isOpen,
  onClose,
  uploadId,
  duplicateItems,
  onSuccess,
}: CreateBenchmarkDialogProps) {
  const [entityType, setEntityType] = useState<'counterparty' | 'nomenclature'>('counterparty')
  const [selectedItems, setSelectedItems] = useState<string[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (selectedItems.length === 0) {
      setError('Выберите хотя бы один элемент')
      return
    }

    setLoading(true)
    setError(null)

    try {
      await apiPost('/api/benchmarks/from-upload', {
        upload_id: uploadId,
        item_ids: selectedItems,
        entity_type: entityType,
      })

      toast.success('Эталон успешно создан')
      onClose()
      if (onSuccess) {
        onSuccess()
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Ошибка создания эталона'
      setError(message)
      toast.error(message)
    } finally {
      setLoading(false)
    }
  }

  const handleItemChange = (itemId: string, checked: boolean) => {
    if (checked) {
      setSelectedItems(prev => [...prev, itemId])
    } else {
      setSelectedItems(prev => prev.filter(id => id !== itemId))
    }
  }

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedItems(duplicateItems.map(item => item.id))
    } else {
      setSelectedItems([])
    }
  }

  const isAllSelected = duplicateItems.length > 0 && selectedItems.length === duplicateItems.length

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Создать эталон</DialogTitle>
          <DialogDescription>
            Выберите элементы из группы дубликатов для создания эталона
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="entityType">Тип сущности</Label>
            <Select value={entityType} onValueChange={(value: 'counterparty' | 'nomenclature') => setEntityType(value)}>
              <SelectTrigger id="entityType">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="counterparty">Контрагент</SelectItem>
                <SelectItem value="nomenclature">Номенклатура</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <Label>Выберите элементы ({selectedItems.length} из {duplicateItems.length} выбрано)</Label>
              {duplicateItems.length > 0 && (
                <div className="flex items-center space-x-2">
                  <Checkbox
                    id="select-all"
                    checked={isAllSelected}
                    onCheckedChange={handleSelectAll}
                  />
                  <Label htmlFor="select-all" className="text-sm">Выбрать все</Label>
                </div>
              )}
            </div>
            <div className="border rounded-md p-4 max-h-[300px] overflow-y-auto space-y-2">
              {duplicateItems.length === 0 ? (
                <p className="text-sm text-muted-foreground">Нет элементов для выбора</p>
              ) : (
                duplicateItems.map((item) => (
                  <div
                    key={item.id}
                    className="flex items-start space-x-3 p-3 hover:bg-muted rounded cursor-pointer"
                  >
                    <Checkbox
                      id={`item-${item.id}`}
                      checked={selectedItems.includes(item.id)}
                      onCheckedChange={(checked) => handleItemChange(item.id, checked as boolean)}
                    />
                    <div className="flex-1 min-w-0">
                      <Label htmlFor={`item-${item.id}`} className="font-medium cursor-pointer">
                        {item.name}
                      </Label>
                      {item.code && (
                        <div className="text-sm text-muted-foreground">Код: {item.code}</div>
                      )}
                      {item.category && (
                        <div className="text-sm text-muted-foreground">Категория: {item.category}</div>
                      )}
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={onClose}
              disabled={loading}
            >
              Отмена
            </Button>
            <Button type="submit" disabled={loading || selectedItems.length === 0}>
              {loading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
              Создать эталон
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

