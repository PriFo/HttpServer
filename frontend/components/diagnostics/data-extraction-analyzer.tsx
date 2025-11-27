'use client'

import { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Loader2, RefreshCw, AlertCircle, CheckCircle2, Info } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface ExtractionStatus {
  database_id: number
  database_name: string
  source_tables: string[]
  catalog_items_count: number
  nomenclature_items_count: number
  extraction_method: 'auto' | 'manual' | 'none'
  last_extraction?: string
  issues: string[]
}

interface DataExtractionAnalyzerProps {
  projectId: number
  clientId: number
}

export function DataExtractionAnalyzer({ projectId, clientId }: DataExtractionAnalyzerProps) {
  const [extractionStatus, setExtractionStatus] = useState<ExtractionStatus[]>([])
  const [isAnalyzing, setIsAnalyzing] = useState(false)

  const analyzeExtraction = async () => {
    setIsAnalyzing(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/diagnostics/extraction`)
      if (!response.ok) {
        throw new Error('Не удалось проанализировать извлечение данных')
      }
      const data = await response.json()
      setExtractionStatus(data)
    } catch (err) {
      console.error('Error analyzing extraction:', err)
    } finally {
      setIsAnalyzing(false)
    }
  }

  const getMethodBadge = (method: string) => {
    switch (method) {
      case 'auto':
        return <Badge variant="default" className="bg-green-600">Автоматически</Badge>
      case 'manual':
        return <Badge variant="outline">Вручную</Badge>
      case 'none':
        return <Badge variant="destructive">Не извлечено</Badge>
      default:
        return <Badge variant="secondary">{method}</Badge>
    }
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="flex items-center gap-2">
              <RefreshCw className="h-5 w-5" />
              Анализатор извлечения данных
            </CardTitle>
            <CardDescription>
              Проверка наличия данных в catalog_items и nomenclature_items
            </CardDescription>
          </div>
          <Button onClick={analyzeExtraction} disabled={isAnalyzing}>
            {isAnalyzing ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Анализ...
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4 mr-2" />
                Анализировать
              </>
            )}
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {extractionStatus.length === 0 && !isAnalyzing && (
          <div className="text-center py-8 text-muted-foreground">
            <RefreshCw className="h-12 w-12 mx-auto mb-4 opacity-50" />
            <p>Нажмите "Анализировать" для проверки извлечения данных</p>
          </div>
        )}

        {extractionStatus.length > 0 && (
          <div className="space-y-4">
            {extractionStatus.map((status) => (
              <Card key={status.database_id} className="border-l-4 border-l-blue-500">
                <CardHeader className="pb-3">
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">{status.database_name}</CardTitle>
                    {getMethodBadge(status.extraction_method)}
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-4">
                    <div>
                      <div className="text-sm text-muted-foreground">catalog_items</div>
                      <div className="text-lg font-semibold">
                        {status.catalog_items_count.toLocaleString('ru-RU')}
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">nomenclature_items</div>
                      <div className="text-lg font-semibold">
                        {status.nomenclature_items_count.toLocaleString('ru-RU')}
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Исходных таблиц</div>
                      <div className="text-lg font-semibold">
                        {status.source_tables.length}
                      </div>
                    </div>
                    <div>
                      <div className="text-sm text-muted-foreground">Метод</div>
                      <div className="text-sm font-medium">
                        {status.extraction_method === 'auto' ? 'Автоматически' :
                         status.extraction_method === 'manual' ? 'Вручную' : 'Не извлечено'}
                      </div>
                    </div>
                  </div>

                  {status.issues.length > 0 && (
                    <Alert variant="destructive">
                      <AlertCircle className="h-4 w-4" />
                      <AlertDescription>
                        <div className="space-y-2">
                          <div className="space-y-1">
                            {status.issues.map((issue, idx) => (
                              <div key={idx}>• {issue}</div>
                            ))}
                          </div>
                          {status.extraction_method === 'none' && status.source_tables.length > 0 && (
                            <div className="mt-2 pt-2 border-t">
                              <div className="flex items-start gap-2 text-sm">
                                <Info className="h-4 w-4 mt-0.5 text-blue-600" />
                                <div>
                                  <p className="font-medium mb-1">Рекомендация:</p>
                                  <p className="text-muted-foreground">
                                    Для извлечения данных из исходных таблиц 1С используйте обработку 1С или загрузите данные через API upload.
                                  </p>
                                </div>
                              </div>
                            </div>
                          )}
                        </div>
                      </AlertDescription>
                    </Alert>
                  )}

                  {status.source_tables.length > 0 && (
                    <details className="mt-4 text-sm">
                      <summary className="cursor-pointer text-muted-foreground hover:text-foreground">
                        Показать исходные таблицы ({status.source_tables.length})
                      </summary>
                      <div className="mt-2 flex flex-wrap gap-1">
                        {status.source_tables.map((table) => (
                          <Badge key={table} variant="secondary" className="text-xs">
                            {table}
                          </Badge>
                        ))}
                      </div>
                    </details>
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

