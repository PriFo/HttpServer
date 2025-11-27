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
import { Upload, Loader2, AlertCircle, Check, FileText } from 'lucide-react'
import { toast } from 'sonner'

interface GostDocumentUploadDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  gostId: number
  gostNumber: string
  onUploadSuccess?: () => void
}

export function GostDocumentUploadDialog({
  open,
  onOpenChange,
  gostId,
  gostNumber,
  onUploadSuccess,
}: GostDocumentUploadDialogProps) {
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      const maxSize = 50 * 1024 * 1024 // 50MB для документов
      if (file.size > maxSize) {
        setError(`Файл слишком большой. Максимальный размер: ${(maxSize / 1024 / 1024).toFixed(0)}MB`)
        setSelectedFile(null)
        return
      }

      const validExtensions = ['.pdf', '.doc', '.docx']
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

  const handleUpload = async () => {
    if (!selectedFile) {
      setError('Выберите файл для загрузки')
      return
    }

    setLoading(true)
    setError(null)
    setSuccess(null)

    try {
      const formData = new FormData()
      formData.append('file', selectedFile)

      const response = await fetch(`/api/gosts/${gostId}/document`, {
        method: 'POST',
        body: formData,
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Ошибка загрузки документа' }))
        throw new Error(errorData.error || 'Ошибка загрузки документа')
      }

      const data = await response.json()
      const successMessage = `Документ успешно загружен! Размер: ${(data.file_size / 1024 / 1024).toFixed(2)} MB`
      setSuccess(successMessage)
      
      // Показываем toast уведомление
      toast.success('Документ загружен', {
        description: successMessage,
        duration: 5000,
      })

      // Очищаем форму
      setSelectedFile(null)

      // Вызываем callback
      if (onUploadSuccess) {
        setTimeout(() => {
          onUploadSuccess()
        }, 1000)
      }

      // Закрываем диалог через 3 секунды
      setTimeout(() => {
        onOpenChange(false)
        setSuccess(null)
      }, 3000)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Ошибка загрузки документа'
      setError(errorMessage)
      toast.error('Ошибка загрузки', {
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
      setError(null)
      setSuccess(null)
      onOpenChange(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Загрузка документа для {gostNumber}</DialogTitle>
          <DialogDescription>
            Загрузите полный текст стандарта (PDF, DOC или DOCX)
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="document-file">Файл документа</Label>
            <Input
              id="document-file"
              type="file"
              accept=".pdf,.doc,.docx"
              onChange={handleFileChange}
              disabled={loading}
            />
            {selectedFile && (
              <div className="p-3 bg-muted rounded-lg">
                <div className="flex items-center gap-2">
                  <FileText className="h-4 w-4 text-muted-foreground" />
                  <div className="flex-1">
                    <p className="text-sm font-medium">{selectedFile.name}</p>
                    <p className="text-xs text-muted-foreground">
                      {(selectedFile.size / 1024 / 1024).toFixed(2)} MB
                    </p>
                  </div>
                </div>
              </div>
            )}
            <p className="text-xs text-muted-foreground">
              Поддерживаемые форматы: PDF, DOC, DOCX (максимум 50MB)
            </p>
          </div>

          {loading && (
            <div className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">Загрузка документа...</span>
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
            onClick={handleUpload}
            disabled={loading || !selectedFile}
          >
            {loading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Загрузка...
              </>
            ) : (
              <>
                <Upload className="h-4 w-4 mr-2" />
                Загрузить
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

