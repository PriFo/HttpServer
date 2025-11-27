'use client'

import Link from 'next/link'
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { ArrowRightIcon, DownloadIcon } from "@radix-ui/react-icons"
import { ConfidenceBadge } from './confidence-badge'
import { ProcessingLevelBadge } from './processing-level-badge'
import { ExportGroupButton } from './export-group-button'

interface Group {
  normalized_name: string
  normalized_reference: string
  category: string
  merged_count: number
  avg_confidence?: number
  processing_level?: string
}

interface QuickViewModalProps {
  group: Group | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function QuickViewModal({ group, open, onOpenChange }: QuickViewModalProps) {
  if (!group) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="text-xl">{group.normalized_name}</DialogTitle>
          <DialogDescription>
            Быстрый просмотр информации о группе
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-6">
          {/* Основная информация */}
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-3">
              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-1">Категория</h4>
                {group.category ? (
                  <Badge variant="secondary">{group.category}</Badge>
                ) : (
                  <span className="text-sm text-muted-foreground">Не указана</span>
                )}
              </div>

              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-1">Количество элементов</h4>
                <p className="text-2xl font-bold">{group.merged_count || 0}</p>
              </div>
            </div>

            <div className="space-y-3">
              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-1">AI уверенность</h4>
                <ConfidenceBadge confidence={group.avg_confidence} size="lg" />
              </div>

              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-1">Уровень обработки</h4>
                {group.processing_level ? (
                  <ProcessingLevelBadge level={group.processing_level} />
                ) : (
                  <Badge variant="outline">Не определен</Badge>
                )}
              </div>
            </div>
          </div>

          {/* Reference */}
          <div>
            <h4 className="text-sm font-medium text-muted-foreground mb-1">Нормализованный reference</h4>
            <p className="text-sm font-mono bg-muted p-2 rounded">
              {group.normalized_reference}
            </p>
          </div>

          {/* Действия */}
          <div className="flex gap-2 pt-4 border-t">
            <Button asChild className="flex-1">
              <Link
                href={`/results/groups/${encodeURIComponent(group.normalized_name)}/${encodeURIComponent(group.category)}`}
                onClick={() => onOpenChange(false)}
                aria-label={`Перейти к детальной информации о группе ${group.normalized_name}`}
              >
                <ArrowRightIcon className="mr-2 h-4 w-4" />
                Подробнее
              </Link>
            </Button>

            <ExportGroupButton
              normalizedName={group.normalized_name}
              category={group.category}
              variant="outline"
              aria-label={`Экспортировать данные группы ${group.normalized_name}`}
            >
              <DownloadIcon className="mr-2 h-4 w-4" />
              Экспорт
            </ExportGroupButton>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
