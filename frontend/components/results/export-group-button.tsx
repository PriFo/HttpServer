'use client'

import { useState } from 'react'
import { Button } from "@/components/ui/button"
import { DownloadIcon } from "@radix-ui/react-icons"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { toast } from 'sonner'

interface ExportGroupButtonProps {
  normalizedName: string
  category: string
  variant?: "default" | "destructive" | "outline" | "secondary" | "ghost" | "link"
  size?: "default" | "sm" | "lg" | "icon"
  className?: string
  children?: React.ReactNode
  'aria-label'?: string
}

export function ExportGroupButton({
  normalizedName,
  category,
  variant = "default",
  size = "default",
  className,
  children,
  'aria-label': ariaLabel
}: ExportGroupButtonProps) {
  const [exporting, setExporting] = useState(false)

  const handleExport = async (format: 'csv' | 'json') => {
    setExporting(true)
    const formatName = format.toUpperCase()

    try {
      const params = new URLSearchParams({
        normalized_name: normalizedName,
        category: category,
        format: format
      })

      const response = await fetch(`/api/normalization/export-group?${params}`)

      if (!response.ok) {
        throw new Error('Export failed')
      }

      // Получаем имя файла из заголовка Content-Disposition
      const contentDisposition = response.headers.get('Content-Disposition')
      let filename = `group_export_${new Date().toISOString().split('T')[0]}.${format}`

      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename=(.+)/)
        if (filenameMatch) {
          filename = filenameMatch[1].replace(/['"]/g, '')
        }
      }

      // Скачиваем файл
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)

      // Показываем успешное уведомление
      toast.success(`Экспорт выполнен успешно`, {
        description: `Файл ${filename} загружен в формате ${formatName}`,
        duration: 4000,
      })
    } catch (error) {
      console.error('Export failed:', error)
      toast.error('Ошибка экспорта', {
        description: 'Не удалось экспортировать данные. Проверьте подключение к сети и попробуйте позже.',
        duration: 5000,
      })
    } finally {
      setExporting(false)
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant={variant}
          size={size}
          className={className}
          disabled={exporting}
          aria-label={ariaLabel || (exporting ? 'Экспортируются данные...' : 'Экспортировать данные группы')}
        >
          {children || (
            <>
              <DownloadIcon className="mr-2 h-4 w-4" />
              {exporting ? 'Экспорт...' : 'Экспорт'}
            </>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuItem
          onClick={() => handleExport('csv')}
          disabled={exporting}
          aria-label="Экспортировать данные в формате CSV"
        >
          Экспорт в CSV
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => handleExport('json')}
          disabled={exporting}
          aria-label="Экспортировать данные в формате JSON"
        >
          Экспорт в JSON
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
