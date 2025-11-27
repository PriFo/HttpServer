'use client'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Package, Code, Tag, TrendingUp, Users, Database } from "lucide-react"

interface NomenclatureItem {
  id: number
  code: string
  name: string
  normalized_name: string
  category: string
  quality_score: number
  status: string
  merged_count: number
  kpved_code?: string
  kpved_name?: string
  source_reference?: string
  source_name?: string
  ai_confidence?: number
  ai_reasoning?: string
  processing_level?: string
  source_database?: string
  source_type?: string
  project_id?: number
  project_name?: string
}

interface NomenclatureDetailDialogProps {
  item: NomenclatureItem
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NomenclatureDetailDialog({ item, open, onOpenChange }: NomenclatureDetailDialogProps) {
  const getQualityBadgeVariant = (score: number) => {
    if (score >= 0.9) return 'default'
    if (score >= 0.7) return 'secondary'
    return 'destructive'
  }

  const getQualityLabel = (score: number) => {
    if (score >= 0.9) return 'Высокое'
    if (score >= 0.7) return 'Среднее'
    return 'Низкое'
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Package className="h-5 w-5" />
            {item.name}
          </DialogTitle>
          <DialogDescription>
            Детальная информация о номенклатуре
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Основная информация */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Основная информация</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-1">
                  <Code className="h-3 w-3" />
                  Код:
                </span>
                <span className="text-sm font-mono font-medium">{item.code}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Исходное название:</span>
                <span className="text-sm font-medium">{item.source_name || item.name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Нормализованное название:</span>
                <span className="text-sm font-medium">{item.normalized_name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground flex items-center gap-1">
                  <Tag className="h-3 w-3" />
                  Категория:
                </span>
                <Badge variant="outline">{item.category}</Badge>
              </div>
              {item.source_reference && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Исходная ссылка:</span>
                  <span className="text-sm font-medium">{item.source_reference}</span>
                </div>
              )}
            </CardContent>
          </Card>

          {/* КПВЭД */}
          {item.kpved_code && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Классификация КПВЭД</CardTitle>
              </CardHeader>
              <CardContent className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Код КПВЭД:</span>
                  <span className="text-sm font-mono font-medium">{item.kpved_code}</span>
                </div>
                {item.kpved_name && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">Название:</span>
                    <span className="text-sm font-medium">{item.kpved_name}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Качество и обработка */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <TrendingUp className="h-4 w-4" />
                Качество и обработка
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Оценка качества:</span>
                <Badge variant={getQualityBadgeVariant(item.quality_score)}>
                  {getQualityLabel(item.quality_score)} ({Math.round(item.quality_score * 100)}%)
                </Badge>
              </div>
              {item.processing_level && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Уровень обработки:</span>
                  <Badge variant="outline">{item.processing_level}</Badge>
                </div>
              )}
              {item.ai_confidence !== undefined && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">AI уверенность:</span>
                  <span className="text-sm font-medium">{Math.round(item.ai_confidence * 100)}%</span>
                </div>
              )}
              {item.ai_reasoning && (
                <div>
                  <span className="text-sm text-muted-foreground">AI обоснование:</span>
                  <div className="mt-1 text-sm bg-muted p-2 rounded">{item.ai_reasoning}</div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Статистика */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base flex items-center gap-2">
                <Users className="h-4 w-4" />
                Статистика
              </CardTitle>
            </CardHeader>
            <CardContent>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Объединено записей:</span>
                <Badge variant="secondary">{item.merged_count}</Badge>
              </div>
            </CardContent>
          </Card>

          {/* Источник данных */}
          {item.source_type && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Database className="h-4 w-4" />
                  Источник данных
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Тип источника:</span>
                  <Badge variant={item.source_type === 'normalized' ? 'default' : 'outline'}>
                    {item.source_type === 'normalized' ? 'Нормализованная база' : 'Основная база'}
                  </Badge>
                </div>
                {item.source_database && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">База данных:</span>
                    <span className="text-sm font-mono font-medium max-w-[60%] truncate" title={item.source_database}>
                      {item.source_database.split(/[/\\]/).pop() || item.source_database}
                    </span>
                  </div>
                )}
                {item.project_name && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">Проект:</span>
                    <span className="text-sm font-medium">{item.project_name}</span>
                  </div>
                )}
                {item.project_id && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">ID проекта:</span>
                    <span className="text-sm font-mono font-medium">{item.project_id}</span>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

