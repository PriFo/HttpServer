'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { 
  FileText, 
  Upload, 
  Download,
  Trash2,
  File,
  FileSpreadsheet,
  FileCode2
} from "lucide-react"
import type { ClientDocument } from '@/types'
import { formatDate } from "@/lib/locale"
import { DocumentUploadDialog } from "./document-upload-dialog"
import { toast } from "sonner"

interface DocumentsSectionProps {
  clientId: string
  documents: ClientDocument[]
  onDocumentsChange: () => void
}

export function DocumentsSection({ clientId, documents, onDocumentsChange }: DocumentsSectionProps) {
  const [showUploadDialog, setShowUploadDialog] = useState(false)
  const [deletingDocId, setDeletingDocId] = useState<number | null>(null)

  const formatFileSize = (bytes: number) => {
    if (bytes < 1024) return `${bytes} Б`
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(2)} КБ`
    return `${(bytes / (1024 * 1024)).toFixed(2)} МБ`
  }

  const getFileIcon = (fileType: string) => {
    if (fileType.includes('spreadsheet') || fileType.includes('excel')) {
      return <FileSpreadsheet className="h-5 w-5 text-green-600" />
    }
    if (fileType.includes('word') || fileType.includes('document')) {
      return <FileCode2 className="h-5 w-5 text-blue-600" />
    }
    if (fileType.includes('pdf')) {
      return <FileText className="h-5 w-5 text-red-600" />
    }
    return <File className="h-5 w-5 text-muted-foreground" />
  }

  const handleDownload = async (docId: number, fileName: string) => {
    try {
      const response = await fetch(`/api/clients/${clientId}/documents/${docId}`)
      if (!response.ok) {
        throw new Error('Failed to download document')
      }

      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = fileName
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)
      toast.success('Документ загружен')
    } catch (error) {
      console.error('Download error:', error)
      toast.error('Ошибка загрузки документа')
    }
  }

  const handleDelete = async (docId: number) => {
    if (!confirm('Вы уверены, что хотите удалить этот документ?')) {
      return
    }

    setDeletingDocId(docId)
    try {
      const response = await fetch(`/api/clients/${clientId}/documents/${docId}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to delete document')
      }

      toast.success('Документ удален')
      onDocumentsChange()
    } catch (error) {
      console.error('Delete error:', error)
      toast.error('Ошибка удаления документа')
    } finally {
      setDeletingDocId(null)
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <FileText className="h-5 w-5" />
                Документы
              </CardTitle>
              <CardDescription>
                Техническая документация и материалы клиента
              </CardDescription>
            </div>
            <Button onClick={() => setShowUploadDialog(true)}>
              <Upload className="h-4 w-4 mr-2" />
              Загрузить документ
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {documents.length > 0 ? (
            <div className="space-y-2">
              {documents.map((doc) => (
                <div
                  key={doc.id}
                  className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted transition-colors"
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
                    {getFileIcon(doc.file_type)}
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {doc.file_name}
                      </p>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <span>{formatFileSize(doc.file_size)}</span>
                        <span>•</span>
                        <span>{formatDate(doc.uploaded_at)}</span>
                        {doc.category && doc.category !== 'technical' && (
                          <>
                            <span>•</span>
                            <Badge variant="outline" className="text-xs">
                              {doc.category}
                            </Badge>
                          </>
                        )}
                      </div>
                      {doc.description && (
                        <p className="text-xs text-muted-foreground mt-1">
                          {doc.description}
                        </p>
                      )}
                    </div>
                  </div>
                  <div className="flex items-center gap-2 shrink-0">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleDownload(doc.id, doc.file_name)}
                      className="gap-2"
                    >
                      <Download className="h-4 w-4" />
                      Скачать
                    </Button>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDelete(doc.id)}
                      disabled={deletingDocId === doc.id}
                      className="gap-2 text-destructive hover:text-destructive"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="text-center py-8 text-muted-foreground">
              <FileText className="h-12 w-12 mx-auto mb-3 text-muted-foreground/50" />
              <p className="text-sm">Документы отсутствуют</p>
              <p className="text-xs mt-1">Загрузите техническую документацию и материалы клиента</p>
            </div>
          )}
        </CardContent>
      </Card>

      <DocumentUploadDialog
        open={showUploadDialog}
        onOpenChange={setShowUploadDialog}
        clientId={clientId}
        onUploadSuccess={() => {
          onDocumentsChange()
          setShowUploadDialog(false)
        }}
      />
    </>
  )
}

