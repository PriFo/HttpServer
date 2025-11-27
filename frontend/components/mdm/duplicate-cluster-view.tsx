'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Users, Merge, X, CheckCircle2, AlertCircle, Eye } from 'lucide-react'
import { AttributesDisplay } from '@/components/processes/attributes-display'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'

interface DuplicateItem {
  id: string
  name: string
  code?: string
  source_reference?: string
  attributes?: any[]
  similarity?: number
  source?: string
}

interface DuplicateCluster {
  id: string
  name: string
  items: DuplicateItem[]
  similarity: number
}

interface DuplicateClusterViewProps {
  cluster: DuplicateCluster
  onMerge: (clusterId: string, selectedAttributes: Record<string, any>) => void
  onSeparate: (clusterId: string) => void
  onExclude: (clusterId: string, itemId: string) => void
}

export const DuplicateClusterView: React.FC<DuplicateClusterViewProps> = ({
  cluster,
  onMerge,
  onSeparate,
  onExclude,
}) => {
  const [selectedAttributes, setSelectedAttributes] = useState<Record<string, Record<string, boolean>>>({})
  const [mergedPreview, setMergedPreview] = useState<any>(null)
  const [showPreview, setShowPreview] = useState(false)

  // Инициализация выбранных атрибутов
  React.useEffect(() => {
    const initial: Record<string, Record<string, boolean>> = {}
    cluster.items.forEach(item => {
      if (item.attributes) {
        initial[item.id] = {}
        item.attributes.forEach(attr => {
          initial[item.id][attr.id || attr.attribute_name] = false
        })
      }
    })
    setSelectedAttributes(initial)
  }, [cluster])

  const handleAttributeToggle = (itemId: string, attrId: string) => {
    setSelectedAttributes(prev => ({
      ...prev,
      [itemId]: {
        ...prev[itemId],
        [attrId]: !prev[itemId]?.[attrId],
      },
    }))
  }

  const generateMergedPreview = () => {
    // Генерация предпросмотра слияния на основе выбранных атрибутов
    const merged: any = {
      name: cluster.name,
      attributes: [],
    }

    cluster.items.forEach(item => {
      if (item.attributes && selectedAttributes[item.id]) {
        item.attributes.forEach(attr => {
          if (selectedAttributes[item.id][attr.id || attr.attribute_name]) {
            merged.attributes.push({
              ...attr,
              source: item.name,
            })
          }
        })
      }
    })

    setMergedPreview(merged)
    setShowPreview(true)
  }

  const handleMerge = () => {
    onMerge(cluster.id, selectedAttributes)
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Users className="h-5 w-5" />
                Группа дубликатов: {cluster.name}
              </CardTitle>
              <CardDescription>
                {cluster.items.length} записей со схожестью {Math.round(cluster.similarity * 100)}%
              </CardDescription>
            </div>
            <Badge variant={cluster.similarity >= 0.9 ? 'default' : 'secondary'}>
              {cluster.similarity >= 0.9 ? 'Высокая схожесть' : 'Средняя схожесть'}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Список исходных записей */}
          <div className="space-y-3">
            <h4 className="font-semibold text-sm">Исходные записи:</h4>
            {cluster.items.map((item, index) => (
              <Card key={item.id} className="border-2">
                <CardContent className="p-4">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-1">
                        <span className="font-medium">{item.name}</span>
                        {item.similarity && (
                          <Badge variant="outline" className="text-xs">
                            {Math.round(item.similarity * 100)}%
                          </Badge>
                        )}
                      </div>
                      {item.code && (
                        <p className="text-xs text-muted-foreground">Код: {item.code}</p>
                      )}
                      {item.source_reference && (
                        <p className="text-xs text-muted-foreground">Reference: {item.source_reference}</p>
                      )}
                      {item.source && (
                        <p className="text-xs text-muted-foreground">Источник: {item.source}</p>
                      )}
                    </div>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => onExclude(cluster.id, item.id)}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </div>

                  {/* Атрибуты записи */}
                  {item.attributes && item.attributes.length > 0 && (
                    <div className="mt-3 pt-3 border-t">
                      <Label className="text-xs font-medium mb-2 block">
                        Выберите атрибуты для сохранения:
                      </Label>
                      <div className="space-y-2">
                        {item.attributes.map((attr) => {
                          const attrId = attr.id || attr.attribute_name
                          const isSelected = selectedAttributes[item.id]?.[attrId] || false
                          return (
                            <div
                              key={attrId}
                              className="flex items-center space-x-2 p-2 border rounded hover:bg-muted/50"
                            >
                              <Checkbox
                                id={`${item.id}-${attrId}`}
                                checked={isSelected}
                                onCheckedChange={() => handleAttributeToggle(item.id, attrId)}
                              />
                              <Label
                                htmlFor={`${item.id}-${attrId}`}
                                className="flex-1 cursor-pointer"
                              >
                                <div className="flex items-center justify-between">
                                  <span className="text-sm">
                                    {attr.attribute_name || attr.attribute_type}: {attr.attribute_value}
                                  </span>
                                  {attr.confidence && attr.confidence < 1.0 && (
                                    <Badge variant="outline" className="text-xs ml-2">
                                      {Math.round(attr.confidence * 100)}%
                                    </Badge>
                                  )}
                                </div>
                              </Label>
                            </div>
                          )
                        })}
                      </div>
                    </div>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>

          <Separator />

          {/* Предпросмотр слияния */}
          <div className="flex items-center justify-between">
            <div>
              <h4 className="font-semibold text-sm mb-1">Предпросмотр слияния</h4>
              <p className="text-xs text-muted-foreground">
                Выберите атрибуты и нажмите "Предпросмотр" для просмотра результата
              </p>
            </div>
            <Dialog open={showPreview} onOpenChange={setShowPreview}>
              <DialogTrigger asChild>
                <Button variant="outline" size="sm" onClick={generateMergedPreview}>
                  <Eye className="h-4 w-4 mr-2" />
                  Предпросмотр
                </Button>
              </DialogTrigger>
              <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
                <DialogHeader>
                  <DialogTitle>Предпросмотр слияния</DialogTitle>
                  <DialogDescription>
                    Результат слияния записей с выбранными атрибутами
                  </DialogDescription>
                </DialogHeader>
                {mergedPreview && (
                  <div className="space-y-4">
                    <div>
                      <h4 className="font-semibold mb-2">Нормализованное название:</h4>
                      <p className="text-lg">{mergedPreview.name}</p>
                    </div>
                    {mergedPreview.attributes && mergedPreview.attributes.length > 0 && (
                      <div>
                        <h4 className="font-semibold mb-2">Выбранные атрибуты:</h4>
                        <AttributesDisplay
                          attributes={mergedPreview.attributes}
                          loading={false}
                          compact={false}
                        />
                      </div>
                    )}
                  </div>
                )}
              </DialogContent>
            </Dialog>
          </div>

          <Separator />

          {/* Действия */}
          <div className="flex gap-2">
            <Button
              onClick={handleMerge}
              className="flex-1"
              disabled={Object.keys(selectedAttributes).length === 0}
            >
              <Merge className="h-4 w-4 mr-2" />
              Принять слияние
            </Button>
            <Button
              variant="outline"
              onClick={() => onSeparate(cluster.id)}
              className="flex-1"
            >
              <X className="h-4 w-4 mr-2" />
              Разделить группу
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

