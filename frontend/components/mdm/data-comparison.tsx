'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ArrowLeftRight, Plus, Minus, Equal } from 'lucide-react'

interface DataComparisonProps {
  clientId: string
  projectId: string
  item1?: any
  item2?: any
  onItemSelect?: (item: any, position: 1 | 2) => void
}

export const DataComparison: React.FC<DataComparisonProps> = ({
  clientId,
  projectId,
  item1,
  item2,
  onItemSelect,
}) => {
  const [comparisonMode, setComparisonMode] = useState<'side-by-side' | 'unified'>('side-by-side')

  const getDifferences = () => {
    if (!item1 || !item2) return []

    const differences: Array<{
      field: string
      value1: any
      value2: any
      type: 'added' | 'removed' | 'changed' | 'unchanged'
    }> = []

    const allFields = new Set([
      ...Object.keys(item1),
      ...Object.keys(item2),
    ])

    allFields.forEach(field => {
      const val1 = item1[field]
      const val2 = item2[field]

      if (val1 === undefined && val2 !== undefined) {
        differences.push({ field, value1: null, value2: val2, type: 'added' })
      } else if (val1 !== undefined && val2 === undefined) {
        differences.push({ field, value1: val1, value2: null, type: 'removed' })
      } else if (val1 !== val2) {
        differences.push({ field, value1: val1, value2: val2, type: 'changed' })
      } else {
        differences.push({ field, value1: val1, value2: val2, type: 'unchanged' })
      }
    })

    return differences
  }

  const differences = getDifferences()
  const changedCount = differences.filter(d => d.type === 'changed').length
  const addedCount = differences.filter(d => d.type === 'added').length
  const removedCount = differences.filter(d => d.type === 'removed').length

  if (!item1 || !item2) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base flex items-center gap-2">
            <ArrowLeftRight className="h-5 w-5" />
            Сравнение данных
          </CardTitle>
          <CardDescription>
            Выберите два элемента для сравнения
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-2 gap-4">
            <div className="p-4 border-2 border-dashed rounded-lg text-center">
              <p className="text-sm text-muted-foreground mb-2">Элемент 1</p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onItemSelect && onItemSelect(null, 1)}
              >
                Выбрать элемент
              </Button>
            </div>
            <div className="p-4 border-2 border-dashed rounded-lg text-center">
              <p className="text-sm text-muted-foreground mb-2">Элемент 2</p>
              <Button
                variant="outline"
                size="sm"
                onClick={() => onItemSelect && onItemSelect(null, 2)}
              >
                Выбрать элемент
              </Button>
            </div>
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
            <CardTitle className="text-base flex items-center gap-2">
              <ArrowLeftRight className="h-5 w-5" />
              Сравнение данных
            </CardTitle>
            <CardDescription>
              Детальное сравнение двух элементов
            </CardDescription>
          </div>
          <div className="flex items-center gap-2">
            <Badge variant="outline">{changedCount} изменений</Badge>
            {addedCount > 0 && <Badge variant="default">+{addedCount}</Badge>}
            {removedCount > 0 && <Badge variant="destructive">-{removedCount}</Badge>}
          </div>
        </div>
      </CardHeader>
      <CardContent>
        <Tabs value={comparisonMode} onValueChange={(v: any) => setComparisonMode(v)}>
          <TabsList>
            <TabsTrigger value="side-by-side">Рядом</TabsTrigger>
            <TabsTrigger value="unified">Объединенный вид</TabsTrigger>
          </TabsList>

          <TabsContent value="side-by-side" className="mt-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <h4 className="font-semibold mb-3">Элемент 1</h4>
                <div className="space-y-2">
                  {Object.entries(item1).map(([key, value]) => {
                    const diff = differences.find(d => d.field === key)
                    return (
                      <div
                        key={key}
                        className={`p-2 border rounded ${
                          diff?.type === 'changed' ? 'bg-yellow-50 border-yellow-200' :
                          diff?.type === 'removed' ? 'bg-red-50 border-red-200' :
                          'bg-muted/30'
                        }`}
                      >
                        <div className="text-xs font-medium text-muted-foreground">{key}</div>
                        <div className="text-sm mt-1">
                          {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
              <div>
                <h4 className="font-semibold mb-3">Элемент 2</h4>
                <div className="space-y-2">
                  {Object.entries(item2).map(([key, value]) => {
                    const diff = differences.find(d => d.field === key)
                    return (
                      <div
                        key={key}
                        className={`p-2 border rounded ${
                          diff?.type === 'changed' ? 'bg-yellow-50 border-yellow-200' :
                          diff?.type === 'added' ? 'bg-green-50 border-green-200' :
                          'bg-muted/30'
                        }`}
                      >
                        <div className="text-xs font-medium text-muted-foreground">{key}</div>
                        <div className="text-sm mt-1">
                          {typeof value === 'object' ? JSON.stringify(value) : String(value)}
                        </div>
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          </TabsContent>

          <TabsContent value="unified" className="mt-4">
            <div className="space-y-2">
              {differences.map((diff) => (
                <div
                  key={diff.field}
                  className={`p-3 border rounded-lg ${
                    diff.type === 'changed' ? 'bg-yellow-50 border-yellow-200' :
                    diff.type === 'added' ? 'bg-green-50 border-green-200' :
                    diff.type === 'removed' ? 'bg-red-50 border-red-200' :
                    'bg-muted/30'
                  }`}
                >
                  <div className="flex items-center justify-between mb-2">
                    <span className="font-medium text-sm">{diff.field}</span>
                    {diff.type === 'changed' && (
                      <Badge variant="outline" className="text-xs">
                        <ArrowLeftRight className="h-3 w-3 mr-1" />
                        Изменено
                      </Badge>
                    )}
                    {diff.type === 'added' && (
                      <Badge variant="default" className="text-xs">
                        <Plus className="h-3 w-3 mr-1" />
                        Добавлено
                      </Badge>
                    )}
                    {diff.type === 'removed' && (
                      <Badge variant="destructive" className="text-xs">
                        <Minus className="h-3 w-3 mr-1" />
                        Удалено
                      </Badge>
                    )}
                    {diff.type === 'unchanged' && (
                      <Badge variant="outline" className="text-xs">
                        <Equal className="h-3 w-3 mr-1" />
                        Без изменений
                      </Badge>
                    )}
                  </div>
                  {diff.type !== 'unchanged' && (
                    <div className="grid grid-cols-2 gap-2 text-xs">
                      <div>
                        <div className="text-muted-foreground mb-1">Было:</div>
                        <div className="line-through text-red-600">
                          {diff.value1 !== null ? String(diff.value1) : '—'}
                        </div>
                      </div>
                      <div>
                        <div className="text-muted-foreground mb-1">Стало:</div>
                        <div className="font-medium text-green-600">
                          {diff.value2 !== null ? String(diff.value2) : '—'}
                        </div>
                      </div>
                    </div>
                  )}
                  {diff.type === 'unchanged' && (
                    <div className="text-sm text-muted-foreground">
                      {String(diff.value1)}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

