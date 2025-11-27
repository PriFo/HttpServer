'use client'

import { useMemo } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Loader2, RefreshCw, AlertCircle } from 'lucide-react'
import { formatNumber } from '@/lib/locale'

export interface DuplicateCounterparty {
  id: number
  name?: string
  normalized_name?: string
  tax_id?: string
  kpp?: string
  bin?: string
  legal_address?: string
  contact_phone?: string
  contact_email?: string
  quality_score?: number
  database_name?: string
  project_name?: string
  source_reference?: string
  source_databases?: Array<{
    DatabaseName: string
    SourceReference?: string
  }>
}

export interface DuplicateGroup {
  tax_id?: string
  key?: string
  key_type?: string
  count: number
  items: DuplicateCounterparty[]
}

interface CounterpartyDuplicatesDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  groups: DuplicateGroup[] | null
  isLoading: boolean
  error?: string | null
  onRefresh?: () => void
}

export function CounterpartyDuplicatesDialog({
  open,
  onOpenChange,
  groups,
  isLoading,
  error,
  onRefresh,
}: CounterpartyDuplicatesDialogProps) {
  const totalDuplicates = useMemo(() => {
    if (!groups) return 0
    return groups.reduce((sum, group) => sum + group.count, 0)
  }, [groups])

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogHeader>
          <DialogTitle>Группы дубликатов контрагентов</DialogTitle>
          <DialogDescription>
            Найдено {groups?.length || 0} групп дубликатов • {totalDuplicates} записей
          </DialogDescription>
        </DialogHeader>

        {error && (
          <Alert variant="destructive" className="mb-2">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}

        <div className="flex items-center justify-between pb-2">
          <div className="text-sm text-muted-foreground">
            {isLoading
              ? 'Загрузка дубликатов...'
              : groups && groups.length > 0
              ? 'Выберите группу, чтобы сравнить данные'
              : 'Дубликаты не найдены'}
          </div>
          <Button
            variant="outline"
            size="sm"
            onClick={onRefresh}
            disabled={isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Обновление...
              </>
            ) : (
              <>
                <RefreshCw className="h-4 w-4 mr-2" />
                Обновить
              </>
            )}
          </Button>
        </div>

        <ScrollArea className="max-h-[60vh] border rounded-md">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>ИНН/БИН</TableHead>
                <TableHead>Количество</TableHead>
                <TableHead>Записи</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={3} className="text-center py-6 text-muted-foreground">
                    <Loader2 className="h-4 w-4 mr-2 animate-spin inline" />
                    Загружаем данные о дубликатах...
                  </TableCell>
                </TableRow>
              ) : groups && groups.length > 0 ? (
                groups.map(group => (
                  <TableRow key={`${group.tax_id || group.key}-${group.count}`}>
                    <TableCell>
                      <div className="flex flex-col gap-1">
                        <span className="font-mono font-medium">
                          {group.tax_id || group.key || '—'}
                        </span>
                        {group.key_type && (
                          <Badge variant="outline" className="w-fit text-xs">
                            {group.key_type === 'bin' ? 'БИН' : 'ИНН/КПП'}
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary">
                        {formatNumber(group.count)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="space-y-2">
                        {group.items.slice(0, 3).map(item => (
                          <div
                            key={item.id}
                            className="p-2 border rounded-md bg-muted/30"
                          >
                            <div className="font-medium">
                              {item.normalized_name || item.name || 'Без названия'}
                            </div>
                            <div className="text-xs text-muted-foreground flex flex-wrap gap-2 mt-1">
                              {item.tax_id && <span>ИНН: {item.tax_id}</span>}
                              {item.bin && <span>БИН: {item.bin}</span>}
                              {item.quality_score !== undefined && (
                                <span>Качество: {Math.round((item.quality_score || 0) * 100)}%</span>
                              )}
                              {item.database_name && (
                                <span>База: {item.database_name.split(/[/\\]/).pop()}</span>
                              )}
                            </div>
                          </div>
                        ))}
                        {group.items.length > 3 && (
                          <div className="text-xs text-muted-foreground">
                            + ещё {group.items.length - 3} записей
                          </div>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              ) : (
                <TableRow>
                  <TableCell colSpan={3} className="text-center py-6 text-muted-foreground">
                    Дубликаты не найдены
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </ScrollArea>

        <DialogFooter>
          <Button onClick={() => onOpenChange(false)}>Закрыть</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

