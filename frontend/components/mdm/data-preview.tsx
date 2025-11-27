'use client'

import React, { useState, useMemo, useCallback } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Eye, Code, Table } from 'lucide-react'
import { AttributesDisplay } from '@/components/processes/attributes-display'
import { toast } from 'sonner'
import { formatPercent, formatNumber } from '@/utils/normalization-helpers'

interface DataPreviewProps {
  data: any
  type?: 'group' | 'item' | 'attribute'
}

export const DataPreview: React.FC<DataPreviewProps> = ({ data, type = 'group' }) => {
  const [viewMode, setViewMode] = useState<'formatted' | 'json' | 'table'>('formatted')

  const formattedView = useMemo(() => {
    if (!data) return null

    if (type === 'group') {
      return (
        <div className="space-y-4">
          <div>
            <h4 className="font-semibold mb-2">Нормализованное название</h4>
            <p className="text-lg">{data.normalized_name || data.name}</p>
          </div>
          
          {data.category && (
            <div>
              <h4 className="font-semibold mb-2">Категория</h4>
              <Badge variant="outline">{data.category}</Badge>
            </div>
          )}

          {data.merged_count && (
            <div>
              <h4 className="font-semibold mb-2">Объединено записей</h4>
              <p>{formatNumber(data.merged_count)}</p>
            </div>
          )}

          {typeof data.avg_confidence === 'number' && (
            <div>
              <h4 className="font-semibold mb-2">Средняя уверенность</h4>
              <Badge variant={data.avg_confidence >= 0.9 ? 'default' : 'secondary'}>
                {formatPercent(data.avg_confidence, 0)}
              </Badge>
            </div>
          )}

          {data.kpved_code && (
            <div>
              <h4 className="font-semibold mb-2">КПВЭД</h4>
              <p className="font-mono">{data.kpved_code}</p>
              {data.kpved_name && (
                <p className="text-sm text-muted-foreground">{data.kpved_name}</p>
              )}
            </div>
          )}

          {data.attributes && data.attributes.length > 0 && (
            <div>
              <h4 className="font-semibold mb-2">Атрибуты</h4>
              <AttributesDisplay attributes={data.attributes} />
            </div>
          )}
        </div>
      )
    }

    if (type === 'item') {
      return (
        <div className="space-y-4">
          <div>
            <h4 className="font-semibold mb-2">Название</h4>
            <p className="text-lg">{data.source_name || data.name}</p>
          </div>

          {data.code && (
            <div>
              <h4 className="font-semibold mb-2">Код</h4>
              <p className="font-mono">{data.code}</p>
            </div>
          )}

          {data.source_reference && (
            <div>
              <h4 className="font-semibold mb-2">Reference</h4>
              <p className="font-mono text-sm">{data.source_reference}</p>
            </div>
          )}

          {data.attributes && data.attributes.length > 0 && (
            <div>
              <h4 className="font-semibold mb-2">Атрибуты</h4>
              <AttributesDisplay attributes={data.attributes} />
            </div>
          )}
        </div>
      )
    }

    return (
      <pre className="text-xs bg-muted p-4 rounded-lg overflow-auto">
        {JSON.stringify(data, null, 2)}
      </pre>
    )
  }, [data, type])

  const tableEntries = useMemo(() => {
    if (!data) return []
    return Object.entries(data).filter(([key]) => !['attributes', 'items'].includes(key))
  }, [data])

  const jsonString = useMemo(() => (data ? JSON.stringify(data, null, 2) : ''), [data])

  const handleCopyJson = useCallback(() => {
    if (!jsonString) return
    navigator.clipboard?.writeText(jsonString)
      .then(() => toast.success('JSON скопирован в буфер обмена'))
      .catch(() => toast.error('Не удалось скопировать JSON'))
  }, [jsonString])

  if (!data) {
    return (
      <Card>
        <CardContent className="py-8">
          <div className="text-center text-muted-foreground">
            Нет данных для отображения
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-base">Предпросмотр данных</CardTitle>
            <CardDescription>
              Просмотр данных в различных форматах
            </CardDescription>
          </div>
          <Badge variant="outline">{type}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        <Tabs value={viewMode} onValueChange={(v: any) => setViewMode(v)}>
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="formatted">
              <Eye className="h-4 w-4 mr-2" />
              Форматированный
            </TabsTrigger>
            <TabsTrigger value="table">
              <Table className="h-4 w-4 mr-2" />
              Таблица
            </TabsTrigger>
            <TabsTrigger value="json">
              <Code className="h-4 w-4 mr-2" />
              JSON
            </TabsTrigger>
          </TabsList>

          <TabsContent value="formatted" className="mt-4">
            <ScrollArea className="h-[400px]">
              {formattedView}
            </ScrollArea>
          </TabsContent>

          <TabsContent value="table" className="mt-4">
            <ScrollArea className="h-[400px]">
              <div className="overflow-x-auto">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left p-2 font-semibold">Поле</th>
                      <th className="text-left p-2 font-semibold">Значение</th>
                    </tr>
                  </thead>
                  <tbody>
                    {tableEntries.map(([key, value]) => (
                      <tr key={key} className="border-b">
                        <td className="p-2 font-medium">{key}</td>
                        <td className="p-2">
                          {typeof value === 'object' ? (
                            <pre className="text-xs whitespace-pre-wrap break-all">{JSON.stringify(value, null, 2)}</pre>
                          ) : (
                            String(value)
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </ScrollArea>
          </TabsContent>

          <TabsContent value="json" className="mt-4">
            <div className="flex items-center justify-between mb-2">
              <span className="text-sm text-muted-foreground">JSON представление</span>
              <Button variant="outline" size="sm" onClick={handleCopyJson}>
                Скопировать
              </Button>
            </div>
            <ScrollArea className="h-[400px]">
              <pre className="text-xs bg-muted p-4 rounded-lg overflow-auto">
                {jsonString}
              </pre>
            </ScrollArea>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

