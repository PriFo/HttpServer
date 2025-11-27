'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, Database, AlertCircle, CheckCircle2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface NormalizationSession {
  id: number
  status: string
  records_processed: number
  created_at: string
  finished_at?: string
}

interface ProjectDatabaseLink {
  database_id: number
  normalized_records: number
}

interface NormalizationStatus {
  project_id: number
  normalized_records_count: number
  normalization_sessions: NormalizationSession[]
  project_database_links: ProjectDatabaseLink[]
  issues: string[]
}

interface NormalizationStatusProps {
  projectId: number
  clientId: number
}

export function NormalizationStatusChecker({ projectId, clientId }: NormalizationStatusProps) {
  const [status, setStatus] = useState<NormalizationStatus | null>(null)
  const [isChecking, setIsChecking] = useState(false)

  const checkNormalization = async () => {
    setIsChecking(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/normalization`)
      if (!response.ok) {
        throw new Error('Не удалось проверить статус нормализации')
      }
      const data = await response.json()
      setStatus(data)
    } catch (err) {
      console.error('Error checking normalization:', err)
    } finally {
      setIsChecking(false)
    }
  }

  const getStatusBadge = (sessionStatus: string) => {
    switch (sessionStatus) {
      case 'completed':
        return <Badge variant="default" className="bg-green-600">Завершена</Badge>
      case 'running':
        return <Badge variant="default" className="bg-blue-600">Выполняется</Badge>
      case 'failed':
        return <Badge variant="destructive">Ошибка</Badge>
      default:
        return <Badge variant="outline">{sessionStatus}</Badge>
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              Диагностика нормализации
            </CardTitle>
            <CardDescription>
              Проверка статуса нормализации и нормализованных данных
            </CardDescription>
          </div>
          <Button onClick={checkNormalization} disabled={isChecking}>
            {isChecking ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Проверка...
              </>
            ) : (
              <>
                <Database className="h-4 w-4 mr-2" />
                Проверить
              </>
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {!status && !isChecking && (
          <div className="text-center py-8 text-muted-foreground">
            <Database className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>Нажмите "Проверить" для проверки статуса нормализации</p>
          </div>
        )}

        {status && (
          <div className="space-y-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm text-muted-foreground">Нормализованных записей</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-3xl font-bold">
                    {status.normalized_records_count.toLocaleString('ru-RU')}
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm text-muted-foreground">Сессий нормализации</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-3xl font-bold">
                    {status.normalization_sessions.length}
                  </div>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm text-muted-foreground">Связанных БД</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-3xl font-bold">
                    {status.project_database_links.length}
                  </div>
                </CardContent>
              </Card>
            </div>

            {status.normalization_sessions.length > 0 && (
              <div>
                <h4 className="text-sm font-semibold mb-2">Сессии нормализации:</h4>
                <div className="space-y-2">
                  {status.normalization_sessions.map((session) => (
                    <Card key={session.id} className="border-l-4 border-l-blue-500">
                      <CardContent className="pt-4">
                        <div className="flex items-center justify-between">
                          <div>
                            <div className="font-medium">Сессия #{session.id}</div>
                            <div className="text-sm text-muted-foreground">
                              Создана: {new Date(session.created_at).toLocaleString('ru-RU')}
                              {session.finished_at && (
                                <> • Завершена: {new Date(session.finished_at).toLocaleString('ru-RU')}</>
                              )}
                            </div>
                            <div className="text-sm mt-1">
                              Обработано записей: <span className="font-semibold">{session.records_processed.toLocaleString('ru-RU')}</span>
                            </div>
                          </div>
                          {getStatusBadge(session.status)}
                        </div>
                      </CardContent>
                    </Card>
                  ))}
                </div>
              </div>
            )}

            {status.project_database_links.length > 0 && (
              <div>
                <h4 className="text-sm font-semibold mb-2">Связи с базами данных:</h4>
                <div className="space-y-2">
                  {status.project_database_links.map((link) => (
                    <div key={link.database_id} className="flex items-center justify-between p-2 border rounded">
                      <span className="text-sm">База данных #{link.database_id}</span>
                      <Badge variant="outline">
                        {link.normalized_records.toLocaleString('ru-RU')} записей
                      </Badge>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {status.issues.length > 0 && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  <div className="space-y-1">
                    {status.issues.map((issue, idx) => (
                      <div key={idx}>• {issue}</div>
                    ))}
                  </div>
                </AlertDescription>
              </Alert>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

