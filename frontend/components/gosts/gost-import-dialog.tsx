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
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import { Upload, Loader2, AlertCircle, Check } from 'lucide-react'
import { toast } from 'sonner'

interface GostImportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onImportSuccess?: () => void
}

export function GostImportDialog({
  open,
  onOpenChange,
  onImportSuccess,
}: GostImportDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [sourceType, setSourceType] = useState('')
  const [sourceUrl, setSourceUrl] = useState('')
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const maxSize = 100 * 1024 * 1024 // 100MB
      if (file.size > maxSize) {
        setError(`Файл слишком большой. Максимальный размер: ${(maxSize / 1024 / 1024).toFixed(0)}MB`)
        setSelectedFile(null)
        return
      }

      const validExtensions = ['.csv', '.txt']
      const fileExtension = '.' + file.name.split('.').pop()?.toLowerCase()
      if (!validExtensions.includes(fileExtension)) {
        setError(`Неподдерживаемый тип файла. Разрешенные форматы: ${validExtensions.join(', ')}`)
        setSelectedFile(null)
        return
      }

      setError(null)
      setSelectedFile(file)
    } else {
      setSelectedFile(null)
    }
  }

  const handleImport = async () => {
    if (!selectedFile || !sourceType) {
      setError('Выберите файл и укажите тип источника')
      return
    }

    setLoading(true)
    setError(null)
    setSuccess(null)

    try {
      const formData = new FormData()
      formData.append('file', selectedFile)
      formData.append('source_type', sourceType)
      if (sourceUrl) {
        formData.append('source_url', sourceUrl)
      }

      const response = await fetch('/api/gosts/import', {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Ошибка импорта' }))
        throw new Error(errorData.error || 'Ошибка импорта ГОСТов')
      }

      const data = await response.json()
      const successMessage = `Успешно импортировано: ${data.success || 0} записей. ` +
        `Создано: ${data.created || 0}, обновлено: ${data.updated || 0}`
      
      setSuccess(successMessage)
      
      // Показываем toast уведомление
      toast.success('Импорт завершен', {
        description: successMessage,
        duration: 5000,
      })

      // Очищаем форму
      setSelectedFile(null)
      setSourceType('')
      setSourceUrl('')

      // Вызываем callback
      if (onImportSuccess) {
        setTimeout(() => {
          onImportSuccess()
        }, 1000)
      }

      // Закрываем диалог через 3 секунды
      setTimeout(() => {
        onOpenChange(false)
        setSuccess(null)
      }, 3000)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Ошибка импорта ГОСТов'
      setError(errorMessage)
      toast.error('Ошибка импорта', {
        description: errorMessage,
        duration: 5000,
      })
    } finally {
      setLoading(false)
    }
  }

  const handleClose = () => {
    if (!loading) {
      setSelectedFile(null)
      setSourceType('')
      setSourceUrl('')
      setError(null)
      setSuccess(null)
      onOpenChange(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Импорт ГОСТов из CSV</DialogTitle>
          <DialogDescription>
            Загрузите CSV файл с ГОСТами для импорта в базу данных
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="file">CSV файл</Label>
            <Input
              id="file"
              type="file"
              accept=".csv,.txt"
              onChange={handleFileChange}
              disabled={loading}
            />
            {selectedFile && (
              <div className="p-3 bg-muted rounded-lg">
                <p className="text-sm font-medium">{selectedFile.name}</p>
                <p className="text-xs text-muted-foreground">
                  {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                </p>
              </div>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="source-type">Тип источника *</Label>
            <Input
              id="source-type"
              value={sourceType}
              onChange={(e) => setSourceType(e.target.value)}
              placeholder="например: nationalstandards"
              disabled={loading}
              required
            />
            <p className="text-xs text-muted-foreground">
              Укажите тип источника данных (например: nationalstandards, interstatestandards)
            </p>
          </div>

          <div className="space-y-2">
            <Label htmlFor="source-url">URL источника (необязательно)</Label>
            <Input
              id="source-url"
              value={sourceUrl}
              onChange={(e) => setSourceUrl(e.target.value)}
              placeholder="https://www.rst.gov.ru/opendata/..."
              disabled={loading}
            />
          </div>

          {loading && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Импорт файла...</span>
                <Loader2 className="h-4 w-4 animate-spin" />
              </div>
              <Progress value={undefined} className="h-2" />
            </div>
          )}

          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          {success && (
            <Alert className="border-green-500 bg-green-50 dark:bg-green-950">
              <Check className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-700 dark:text-green-300">
                {success}
              </AlertDescription>
            </Alert>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleClose}
            disabled={loading}
          >
            Отмена
          </Button>
          <Button
            onClick={handleImport}
            disabled={loading || !selectedFile || !sourceType}
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Импорт...
              </>
            ) : (
              <>
                <Upload className="h-4 w-4 mr-2" />
                Импортировать
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

