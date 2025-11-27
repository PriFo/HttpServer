/**
 * Переиспользуемый компонент для экспорта данных
 */

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from '@/components/ui/dropdown-menu'
import { Download, FileText, Loader2 } from 'lucide-react'
import { toast } from 'sonner'

export interface ExportOption {
  label: string
  icon?: React.ReactNode
  onClick: () => void | Promise<void>
  disabled?: boolean
}

interface ExportButtonProps {
  options: ExportOption[]
  disabled?: boolean
  loading?: boolean
  variant?: 'default' | 'outline' | 'ghost' | 'link' | 'destructive' | 'secondary'
  size?: 'default' | 'sm' | 'lg' | 'icon'
  label?: string
  'aria-label'?: string
}

export function ExportButton({
  options,
  disabled = false,
  loading = false,
  variant = 'outline',
  size = 'lg',
  label = 'Экспорт',
  'aria-label': ariaLabel = 'Экспорт данных',
}: ExportButtonProps) {
  if (options.length === 0) {
    return null
  }

  // Если только одна опция, показываем обычную кнопку
  if (options.length === 1) {
    const option = options[0]
    return (
      <Button
        onClick={option.onClick}
        disabled={disabled || loading || option.disabled}
        variant={variant}
        size={size}
        aria-label={ariaLabel}
        aria-busy={loading}
      >
        {loading ? (
          <>
            <Loader2 className="mr-2 h-4 w-4 animate-spin" />
            Экспорт...
          </>
        ) : (
          <>
            {option.icon || <Download className="mr-2 h-4 w-4" />}
            {option.label}
          </>
        )}
      </Button>
    )
  }

  // Если несколько опций, показываем dropdown
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          disabled={disabled || loading}
          variant={variant}
          size={size}
          aria-label={ariaLabel}
          aria-busy={loading}
        >
          {loading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Экспорт...
            </>
          ) : (
            <>
              <Download className="mr-2 h-4 w-4" />
              {label}
            </>
          )}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {options.map((option, index) => {
          const isLast = index === options.length - 1
          const showSeparator = index > 0 && option.label.includes('PDF')
          
          return (
            <div key={index}>
              {showSeparator && <DropdownMenuSeparator />}
              <DropdownMenuItem
                onClick={async () => {
                  try {
                    await option.onClick()
                  } catch (error) {
                    toast.error(`Ошибка при экспорте: ${error instanceof Error ? error.message : 'Неизвестная ошибка'}`)
                  }
                }}
                disabled={option.disabled}
              >
                {option.icon || <FileText className="mr-2 h-4 w-4" />}
                {option.label}
              </DropdownMenuItem>
            </div>
          )
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

