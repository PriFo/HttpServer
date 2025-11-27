'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Loader2, Upload, CheckCircle2, XCircle, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'

interface UploadStatus {
  database_id: number
  file_name: string
  upload_id: number | null
  client_id: number | null
  project_id: number | null
  status: 'missing' | 'invalid' | 'valid' | 'error'
  created_at?: string
  records_count?: number
  issues?: string[]
}

interface UploadRecordsMonitorProps {
  projectId: number
  clientId: number
}

export function UploadRecordsMonitor({ projectId, clientId }: UploadRecordsMonitorProps) {
  const [uploadStatus, setUploadStatus] = useState<UploadStatus[]>([])
  const [isChecking, setIsChecking] = useState(false)
  const [isFixing, setIsFixing] = useState(false)

  const checkUploads = async () => {
    setIsChecking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/uploads`)
      if (!response.ok) {
        throw new Error('Не удалось проверить upload записи')
      }
      const data = await response.json()
      setUploadStatus(data)
    } catch (err) {
      toast.error('Ошибка', {
        description: err instanceof Error ? err.message : 'Не удалось проверить upload записи',
      })
    } finally {
      setIsChecking(false)
    }
  }

  const createMissingUploads = async () => {
    setIsFixing(true)
    const fixToast = toast.loading('Создание upload записей...', {
      description: 'Пожалуйста, подождите',
    })
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/uploads/fix`, {
        method: 'POST',
      })
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Не удалось создать upload записи' }))
        throw new Error(errorData.error || 'Не удалось создать upload записи')
      }
      const data = await response.json()
      toast.dismiss(fixToast)
      toast.success('Успешно', {
        description: `Создано ${data.fixed_count} upload записей`,
      })
      await checkUploads()
    } catch (err) {
      toast.dismiss(fixToast)
      toast.error('Ошибка', {
        description: err instanceof Error ? err.message : 'Не удалось создать upload записи',
      })
    } finally {
      setIsFixing(false)
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'valid':
        return <Badge variant="default" className="bg-green-600">Валидна</Badge>
      case 'invalid':
        return <Badge variant="destructive">Невалидна</Badge>
      case 'missing':
        return <Badge variant="secondary">Отсутствует</Badge>
      default:
        return <Badge variant="outline">Ошибка</Badge>
    }
  }

  const missingCount = uploadStatus.filter(u => u.status === 'missing').length
  const invalidCount = uploadStatus.filter(u => u.status === 'invalid').length

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Upload className="h-5 w-5" />
              Мониторинг Upload записей
            </CardTitle>
            <CardDescription>
              Проверка наличия и валидности upload записей для баз данных проекта
            </CardDescription>
          </div>
          <div className="flex gap-2">
            <Button onClick={checkUploads} disabled={isChecking} variant="outline">
              {isChecking ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Проверка...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4 mr-2" />
                  Проверить
                </>
              )}
            </Button>
            {(missingCount > 0 || invalidCount > 0) && (
              <Button onClick={createMissingUploads} disabled={isFixing || isChecking}>
                {isFixing ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Создание...
                  </>
                ) : (
                  <>
                    <CheckCircle2 className="h-4 w-4 mr-2" />
                    Создать недостающие ({missingCount + invalidCount})
                  </>
                )}
              </Button>
            )}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        {uploadStatus.length === 0 && !isChecking && (
          <div className="text-center py-8 text-muted-foreground">
            <Upload className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>Нажмите "Проверить" для проверки upload записей</p>
          </div>
        )}

        {uploadStatus.length > 0 && (
          <div className="space-y-4">
            <div className="flex gap-4 text-sm">
              <div>
                Всего: <span className="font-semibold">{uploadStatus.length}</span>
              </div>
              <div className="text-green-600">
                Валидных: <span className="font-semibold">{uploadStatus.filter(u => u.status === 'valid').length}</span>
              </div>
              <div className="text-yellow-600">
                Отсутствует: <span className="font-semibold">{missingCount}</span>
              </div>
              <div className="text-red-600">
                Невалидных: <span className="font-semibold">{invalidCount}</span>
              </div>
            </div>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>База данных</TableHead>
                  <TableHead>Upload ID</TableHead>
                  <TableHead>Статус</TableHead>
                  <TableHead>Записей</TableHead>
                  <TableHead>Создана</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {uploadStatus.map((upload) => (
                  <TableRow key={upload.database_id}>
                    <TableCell className="font-medium">{upload.file_name}</TableCell>
                    <TableCell>
                      {upload.upload_id ? (
                        <span className="font-mono text-sm">{upload.upload_id}</span>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>{getStatusBadge(upload.status)}</TableCell>
                    <TableCell>
                      {upload.records_count !== undefined ? (
                        upload.records_count.toLocaleString('ru-RU')
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell>
                      {upload.created_at ? (
                        <span className="text-sm text-muted-foreground">
                          {new Date(upload.created_at).toLocaleDateString('ru-RU')}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">—</span>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>

            {uploadStatus.some(u => u.issues && u.issues.length > 0) && (
              <div className="space-y-2">
                <h4 className="text-sm font-semibold flex items-center gap-2">
                  <AlertCircle className="h-4 w-4" />
                  Проблемы:
                </h4>
                {uploadStatus.map((upload) => (
                  upload.issues && upload.issues.length > 0 && (
                    <div key={upload.database_id} className="text-sm p-2 bg-yellow-50 dark:bg-yellow-950/20 rounded border border-yellow-200 dark:border-yellow-800">
                      <div className="font-medium mb-1">{upload.file_name}:</div>
                      <ul className="list-disc list-inside space-y-1 text-muted-foreground">
                        {upload.issues.map((issue, idx) => (
                          <li key={idx}>{issue}</li>
                        ))}
                      </ul>
                    </div>
                  )
                ))}
              </div>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

