'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, Database, AlertCircle, CheckCircle2, XCircle } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface DatabaseDiagnostic {
  file_path: string
  database_id: number
  database_name: string
  exists: boolean
  tables: string[]
  record_counts: { [table: string]: number }
  has_catalog_items: boolean
  has_nomenclature_items: boolean
  has_1c_tables: boolean
  issues: string[]
  is_active: boolean
}

interface DatabaseHealthCheckerProps {
  projectId: number
  clientId: number
}

export function DatabaseHealthChecker({ projectId, clientId }: DatabaseHealthCheckerProps) {
  const [diagnostics, setDiagnostics] = useState<DatabaseDiagnostic[]>([])
  const [isScanning, setIsScanning] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const scanDatabases = async () => {
    setIsScanning(true)
    setError(null)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/databases`)
      if (!response.ok) {
        throw new Error('Не удалось получить диагностику баз данных')
      }
      const data = await response.json()
      setDiagnostics(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка при сканировании')
    } finally {
      setIsScanning(false)
    }
  }

  const getStatusBadge = (diagnostic: DatabaseDiagnostic) => {
    if (!diagnostic.exists) {
      return <Badge variant="destructive">Файл не найден</Badge>
    }
    if (diagnostic.issues.length > 0) {
      return <Badge variant="destructive">Проблемы обнаружены</Badge>
    }
    return <Badge variant="default" className="bg-green-600">OK</Badge>
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <Database className="h-5 w-5" />
              Диагностика баз данных
            </CardTitle>
            <CardDescription>
              Проверка наличия файлов, таблиц и данных в базах данных проекта
            </CardDescription>
          </div>
          <Button onClick={scanDatabases} disabled={isScanning}>
            {isScanning ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Сканирование...
              </>
            ) : (
              <>
                <Database className="h-4 w-4 mr-2" />
                Запустить диагностику
              </>
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {error && (
          <Alert variant="destructive" className="mb-4">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        {diagnostics.length === 0 && !isScanning && (
          <div className="text-center py-8 text-muted-foreground">
            <Database className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>Нажмите "Запустить диагностику" для проверки баз данных</p>
          </div>
        )}

        {diagnostics.length > 0 && (
          <div className="space-y-4">
            {diagnostics.map((db) => (
              <Card key={db.database_id} className="border-l-4 border-l-blue-500">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <CardTitle className="text-lg flex items-center gap-2">
                        {db.database_name}
                        {!db.is_active && (
                          <Badge variant="outline" className="text-xs">Неактивна</Badge>
                        )}
                      </CardTitle>
                      <CardDescription className="text-xs font-mono mt-1">
                        {db.file_path}
                      </CardDescription>
                    </div>
                    {getStatusBadge(db)}
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                    <div className="flex items-center gap-2">
                      {db.exists ? (
                        <CheckCircle2 className="h-4 w-4 text-green-600" />
                      ) : (
                        <XCircle className="h-4 w-4 text-red-600" />
                      )}
                      <span className="text-sm">
                        {db.exists ? 'Файл существует' : 'Файл не найден'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      {db.has_catalog_items ? (
                        <CheckCircle2 className="h-4 w-4 text-green-600" />
                      ) : (
                        <XCircle className="h-4 w-4 text-yellow-600" />
                      )}
                      <span className="text-sm">
                        catalog_items: {db.has_catalog_items ? 'Да' : 'Нет'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      {db.has_nomenclature_items ? (
                        <CheckCircle2 className="h-4 w-4 text-green-600" />
                      ) : (
                        <XCircle className="h-4 w-4 text-yellow-600" />
                      )}
                      <span className="text-sm">
                        nomenclature_items: {db.has_nomenclature_items ? 'Да' : 'Нет'}
                      </span>
                    </div>
                    <div className="flex items-center gap-2">
                      {db.has_1c_tables ? (
                        <CheckCircle2 className="h-4 w-4 text-blue-600" />
                      ) : (
                        <XCircle className="h-4 w-4 text-gray-400" />
                      )}
                      <span className="text-sm">
                        Таблицы 1С: {db.has_1c_tables ? 'Да' : 'Нет'}
                      </span>
                    </div>
                  </div>

                  {Object.keys(db.record_counts).length > 0 && (
                    <div className="mb-4">
                      <h4 className="text-sm font-semibold mb-2">Количество записей:</h4>
                      <div className="flex flex-wrap gap-2">
                        {Object.entries(db.record_counts).map(([table, count]) => (
                          <Badge key={table} variant="outline">
                            {table}: {count.toLocaleString('ru-RU')}
                          </Badge>
                        ))}
                      </div>
                    </div>
                  )}

                  {db.tables.length > 0 && (
                    <div className="mb-4">
                      <details className="text-sm">
                        <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
                          Показать все таблицы ({db.tables.length})
                        </summary>
                        <div className="mt-2 flex flex-wrap gap-1">
                          {db.tables.map((table) => (
                            <Badge key={table} variant="secondary" className="text-xs">
                              {table}
                            </Badge>
                          ))}
                        </div>
                      </details>
                    </div>
                  )}

                  {db.issues.length > 0 && (
                    <Alert variant="destructive">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription>
                        <div className="space-y-2">
                          <div className="space-y-1">
                            {db.issues.map((issue, idx) => (
                              <div key={idx}>• {issue}</div>
                            ))}
                          </div>
                          {!db.exists && (
                            <div className="mt-2 pt-2 border-t text-sm text-muted-foreground">
                              Проверьте путь к файлу базы данных и убедитесь, что файл доступен.
                            </div>
                          )}
                          {db.exists && !db.has_catalog_items && !db.has_nomenclature_items && !db.has_1c_tables && (
                            <div className="mt-2 pt-2 border-t text-sm text-muted-foreground">
                              База данных не содержит ожидаемых таблиц. Убедитесь, что это правильная база данных проекта.
                            </div>
                          )}
                        </div>
                      </AlertDescription>
                    </Alert>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

