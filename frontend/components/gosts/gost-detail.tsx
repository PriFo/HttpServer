'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { 
  Calendar, 
  ExternalLink, 
  FileText, 
  Download,
  AlertCircle,
  Copy,
  Check,
  Upload
} from 'lucide-react'
import { format } from 'date-fns'
import { ru } from 'date-fns/locale/ru'
import Link from 'next/link'
import { GostDocumentUploadDialog } from './gost-document-upload-dialog'
import { toast } from 'sonner'
import { useState as useStateDetail, useEffect as useEffectDetail } from 'react'
import { isFavorite, toggleFavorite } from '@/lib/gost-favorites'
import { Star, Share2 } from 'lucide-react'

interface GostDetailProps {
  gost: {
    id: number
    gost_number: string
    title: string
    adoption_date?: string
    effective_date?: string
    status?: string
    source_type?: string
    source_url?: string
    description?: string
    keywords?: string
    documents?: Array<{
      id: number
      file_path: string
      file_type: string
      file_size: number
      uploaded_at: string
    }>
    created_at?: string
    updated_at?: string
  }
}

export function GostDetail({ gost }: GostDetailProps) {
  const [copied, setCopied] = useState(false)
  const [showUploadDialog, setShowUploadDialog] = useState(false)
  const [refreshTrigger, setRefreshTrigger] = useState(0)
  const [favorite, setFavorite] = useStateDetail(false)

  useEffectDetail(() => {
    setFavorite(isFavorite(gost.id))
  }, [gost.id])

  const handleToggleFavorite = () => {
    const newFavorite = toggleFavorite(gost)
    setFavorite(newFavorite)
    toast.success(newFavorite ? 'Добавлено в избранное' : 'Удалено из избранного', {
      description: `ГОСТ ${gost.gost_number}`,
      duration: 2000,
    })
  }

  const handleShare = async () => {
    const url = `${window.location.origin}/gosts/${gost.id}`
    try {
      if (navigator.share) {
        await navigator.share({
          title: `ГОСТ ${gost.gost_number}`,
          text: gost.title,
          url: url,
        })
        toast.success('Ссылка скопирована')
      } else {
        await navigator.clipboard.writeText(url)
        toast.success('Ссылка скопирована', {
          description: 'Ссылка на ГОСТ скопирована в буфер обмена',
          duration: 2000,
        })
      }
    } catch (err) {
      // Пользователь отменил шаринг или произошла ошибка
      if (err instanceof Error && err.name !== 'AbortError') {
        toast.error('Ошибка', {
          description: 'Не удалось скопировать ссылку',
          duration: 3000,
        })
      }
    }
  }

  const handleCopyNumber = async () => {
    try {
      await navigator.clipboard.writeText(gost.gost_number)
      setCopied(true)
      toast.success('Номер скопирован', {
        description: `ГОСТ ${gost.gost_number} скопирован в буфер обмена`,
        duration: 2000,
      })
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
      toast.error('Ошибка копирования', {
        description: 'Не удалось скопировать номер ГОСТа',
        duration: 3000,
      })
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return null
    try {
      const date = new Date(dateStr)
      return format(date, 'dd.MM.yyyy', { locale: ru })
    } catch {
      return dateStr
    }
  }

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} Б`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} КБ`
    return `${(bytes / (1024 * 1024)).toFixed(2)} МБ`
  }

  const getStatusColor = (status?: string) => {
    if (!status) return 'secondary'
    const statusLower = status.toLowerCase()
    if (statusLower.includes('действующий') || statusLower.includes('действует')) {
      return 'default'
    }
    if (statusLower.includes('отменен') || statusLower.includes('отменён')) {
      return 'destructive'
    }
    if (statusLower.includes('заменен') || statusLower.includes('заменён')) {
      return 'outline'
    }
    return 'secondary'
  }

  const handleDownloadDocument = async (gostId: number, docId?: number) => {
    try {
      const url = `/api/gosts/${gostId}/document${docId ? `?doc_id=${docId}` : ''}`
      const response = await fetch(url)
      
      if (!response.ok) {
        throw new Error('Ошибка загрузки документа')
      }
      
      const blob = await response.blob()
      const downloadUrl = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = downloadUrl
      link.download = `${gost.gost_number}.${blob.type.includes('pdf') ? 'pdf' : 'doc'}`
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(downloadUrl)
      
      toast.success('Документ скачивается', {
        description: `ГОСТ ${gost.gost_number}`,
        duration: 2000,
      })
    } catch (error) {
      console.error('Error downloading document:', error)
      toast.error('Ошибка скачивания', {
        description: error instanceof Error ? error.message : 'Не удалось скачать документ',
        duration: 3000,
      })
    }
  }

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-4">
            <div className="flex-1">
              <div className="flex items-center gap-2 mb-2 flex-wrap">
                <CardTitle className="text-2xl font-mono">
                  {gost.gost_number}
                </CardTitle>
                <div className="flex items-center gap-1">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleCopyNumber}
                    className="h-8 w-8 p-0"
                    title="Копировать номер ГОСТа"
                  >
                    {copied ? (
                      <Check className="h-4 w-4 text-green-600" />
                    ) : (
                      <Copy className="h-4 w-4" />
                    )}
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleToggleFavorite}
                    className={`h-8 w-8 p-0 ${favorite ? 'text-yellow-500 hover:text-yellow-600' : ''}`}
                    title={favorite ? 'Удалить из избранного' : 'Добавить в избранное'}
                  >
                    <Star className={`h-4 w-4 ${favorite ? 'fill-current' : ''}`} />
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleShare}
                    className="h-8 w-8 p-0"
                    title="Поделиться"
                  >
                    <Share2 className="h-4 w-4" />
                  </Button>
                </div>
              </div>
              <CardDescription className="text-base">
                {gost.title}
              </CardDescription>
            </div>
            {gost.status && (
              <Badge variant={getStatusColor(gost.status)} className="shrink-0">
                {gost.status}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {gost.adoption_date && (
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">Дата принятия</p>
                  <p className="text-sm text-muted-foreground">
                    {formatDate(gost.adoption_date)}
                  </p>
                </div>
              </div>
            )}
            {gost.effective_date && (
              <div className="flex items-center gap-2">
                <Calendar className="h-4 w-4 text-muted-foreground" />
                <div>
                  <p className="text-sm font-medium">Дата вступления в силу</p>
                  <p className="text-sm text-muted-foreground">
                    {formatDate(gost.effective_date)}
                  </p>
                </div>
              </div>
            )}
          </div>

          {gost.source_type && (
            <div className="flex items-center gap-2">
              <Badge variant="outline">Тип источника: {gost.source_type}</Badge>
              {gost.source_url && (
                <a
                  href={gost.source_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-primary hover:underline flex items-center gap-1"
                >
                  <ExternalLink className="h-4 w-4" />
                  Открыть источник
                </a>
              )}
            </div>
          )}

          {gost.description && (
            <div>
              <p className="text-sm font-medium mb-2">Описание</p>
              <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                {gost.description}
              </p>
            </div>
          )}

          {gost.keywords && (
            <div>
              <p className="text-sm font-medium mb-2">Ключевые слова</p>
              <div className="flex flex-wrap gap-2">
                {gost.keywords.split(',').map((keyword, idx) => (
                  <Badge key={idx} variant="secondary" className="text-xs">
                    {keyword.trim()}
                  </Badge>
                ))}
              </div>
            </div>
          )}

          {(gost.created_at || gost.updated_at) && (
            <div className="pt-4 border-t">
              <div className="text-xs text-muted-foreground space-y-1">
                {gost.created_at && (
                  <p>Создан: {formatDate(gost.created_at)}</p>
                )}
                {gost.updated_at && (
                  <p>Обновлен: {formatDate(gost.updated_at)}</p>
                )}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Документы
              </CardTitle>
              <CardDescription>
                Полные тексты стандарта
              </CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowUploadDialog(true)}
            >
              <Upload className="h-4 w-4 mr-2" />
              Загрузить документ
            </Button>
          </div>
        </CardHeader>
        {gost.documents && gost.documents.length > 0 ? (
          <CardContent>
            <div className="space-y-2">
              {gost.documents.map((doc) => (
                <div
                  key={doc.id}
                  className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted transition-colors"
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    <FileText className="h-5 w-5 text-muted-foreground shrink-0" />
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {gost.gost_number}.{doc.file_type}
                      </p>
                      <p className="text-xs text-muted-foreground">
                        {formatFileSize(doc.file_size)} • {formatDate(doc.uploaded_at)}
                      </p>
                    </div>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleDownloadDocument(gost.id, doc.id)}
                    className="shrink-0 gap-2"
                  >
                    <Download className="h-4 w-4" />
                    Скачать
                  </Button>
                </div>
              ))}
            </div>
          </CardContent>
        ) : (
          <CardContent className="pt-6">
            <div className="flex flex-col items-center gap-3 text-center py-4">
              <AlertCircle className="h-8 w-8 text-muted-foreground" />
              <div>
                <p className="text-sm font-medium mb-1">Документы для данного ГОСТа не найдены</p>
                <p className="text-xs text-muted-foreground">
                  Загрузите полный текст стандарта
                </p>
              </div>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowUploadDialog(true)}
                className="mt-2"
              >
                <Upload className="h-4 w-4 mr-2" />
                Загрузить документ
              </Button>
            </div>
          </CardContent>
        )}
      </Card>

      <GostDocumentUploadDialog
        open={showUploadDialog}
        onOpenChange={setShowUploadDialog}
        gostId={gost.id}
        gostNumber={gost.gost_number}
        onUploadSuccess={() => {
          setRefreshTrigger(prev => prev + 1)
          // Перезагружаем страницу для обновления данных
          window.location.reload()
        }}
      />
    </div>
  )
}

