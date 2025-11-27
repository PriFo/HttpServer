'use client'

import React, { useState } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Checkbox } from '@/components/ui/checkbox'
import { Download, FileSpreadsheet, FileCode, FileJson } from 'lucide-react'
import { toast } from 'sonner'

interface ExportDialogProps {
  clientId: string
  projectId: string
  dataType: 'nomenclature' | 'counterparties' | 'groups' | 'analytics'
  trigger?: React.ReactNode
  onExportStart?: () => void
  onExportComplete?: (success: boolean) => void
}

export const ExportDialog: React.FC<ExportDialogProps> = ({
  clientId,
  projectId,
  dataType,
  trigger,
  onExportStart,
  onExportComplete,
}) => {
  const [open, setOpen] = useState(false)
  const [format, setFormat] = useState<'excel' | 'csv' | 'json'>('excel')
  const [includeAttributes, setIncludeAttributes] = useState(true)
  const [includeMetadata, setIncludeMetadata] = useState(false)
  const [exporting, setExporting] = useState(false)

  const handleExport = async () => {
    setExporting(true)
    onExportStart?.()
    try {
      // Формируем параметры экспорта
      const params = new URLSearchParams({
        format,
        include_attributes: includeAttributes.toString(),
        include_metadata: includeMetadata.toString(),
      })

      let endpoint = ''
      switch (dataType) {
        case 'nomenclature':
          endpoint = `/api/clients/${clientId}/projects/${projectId}/nomenclature/export`
          break
        case 'counterparties':
          endpoint = `/api/clients/${clientId}/projects/${projectId}/counterparties/export`
          break
        case 'groups':
          endpoint = `/api/clients/${clientId}/projects/${projectId}/normalization/groups/export`
          break
        case 'analytics':
          endpoint = `/api/clients/${clientId}/projects/${projectId}/analytics/export`
          break
      }

      const response = await fetch(`${endpoint}?${params}`, {
        method: 'GET',
      })

      if (!response.ok) {
        throw new Error('Ошибка экспорта данных')
      }

      // Создаем blob и скачиваем файл
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `export_${dataType}_${new Date().toISOString().split('T')[0]}.${format === 'json' ? 'json' : format === 'csv' ? 'csv' : 'xlsx'}`
      document.body.appendChild(a)
      a.click()
      window.URL.revokeObjectURL(url)
      document.body.removeChild(a)

      toast.success('Данные успешно экспортированы')
      setOpen(false)
      onExportComplete?.(true)
    } catch (error) {
      console.error('Export error:', error)
      toast.error('Ошибка при экспорте данных')
      onExportComplete?.(false)
    } finally {
      setExporting(false)
    }
  }

  const getFormatIcon = (fmt: string) => {
    switch (fmt) {
      case 'excel':
        return <FileSpreadsheet className="h-4 w-4" />
      case 'csv':
        return <FileCode className="h-4 w-4" />
      case 'json':
        return <FileJson className="h-4 w-4" />
      default:
        return <Download className="h-4 w-4" />
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {trigger || (
          <Button variant="outline" size="sm">
            <Download className="h-4 w-4 mr-2" />
            Экспорт
          </Button>
        )}
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Экспорт данных</DialogTitle>
          <DialogDescription>
            Выберите параметры экспорта для {dataType === 'nomenclature' && 'номенклатуры'}
            {dataType === 'counterparties' && 'контрагентов'}
            {dataType === 'groups' && 'групп нормализации'}
            {dataType === 'analytics' && 'аналитики'}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label>Формат файла</Label>
            <Select value={format} onValueChange={(v: any) => setFormat(v)}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="excel">
                  <div className="flex items-center gap-2">
                    {getFormatIcon('excel')}
                    <span>Excel (.xlsx)</span>
                  </div>
                </SelectItem>
                <SelectItem value="csv">
                  <div className="flex items-center gap-2">
                    {getFormatIcon('csv')}
                    <span>CSV (.csv)</span>
                  </div>
                </SelectItem>
                <SelectItem value="json">
                  <div className="flex items-center gap-2">
                    {getFormatIcon('json')}
                    <span>JSON (.json)</span>
                  </div>
                </SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-3">
            <div className="flex items-center space-x-2">
              <Checkbox
                id="include-attributes"
                checked={includeAttributes}
                onCheckedChange={(checked) => setIncludeAttributes(checked as boolean)}
              />
              <Label htmlFor="include-attributes" className="cursor-pointer">
                Включить атрибуты
              </Label>
            </div>

            <div className="flex items-center space-x-2">
              <Checkbox
                id="include-metadata"
                checked={includeMetadata}
                onCheckedChange={(checked) => setIncludeMetadata(checked as boolean)}
              />
              <Label htmlFor="include-metadata" className="cursor-pointer">
                Включить метаданные (даты, источники, статусы)
              </Label>
            </div>
          </div>

          <div className="flex gap-2 pt-4">
            <Button
              onClick={handleExport}
              disabled={exporting}
              className="flex-1"
            >
              <Download className="h-4 w-4 mr-2" />
              {exporting ? 'Экспорт...' : 'Экспортировать'}
            </Button>
            <Button
              variant="outline"
              onClick={() => setOpen(false)}
              disabled={exporting}
            >
              Отмена
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}

