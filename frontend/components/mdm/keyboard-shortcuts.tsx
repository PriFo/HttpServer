'use client'

import React, { useEffect, useState, useCallback } from 'react'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Keyboard, Command } from 'lucide-react'

interface KeyboardShortcutsProps {
  children?: React.ReactNode
  onSearch?: () => void
  onExport?: () => void
  onRefresh?: () => void
  onTabChange?: (tab: string) => void
}

interface Shortcut {
  keys: string[]
  description: string
  action?: () => void
}

export const KeyboardShortcuts: React.FC<KeyboardShortcutsProps> = ({
  children,
  onSearch,
  onExport,
  onRefresh,
  onTabChange,
}) => {
  const [open, setOpen] = useState(false)

  const shortcuts: Shortcut[] = [
    { keys: ['Ctrl', 'K'], description: 'Открыть поиск', action: onSearch },
    { keys: ['Ctrl', 'E'], description: 'Экспорт данных', action: onExport },
    { keys: ['Ctrl', 'R'], description: 'Обновить данные', action: onRefresh },
    { keys: ['Ctrl', '1'], description: 'Перейти к обзору', action: () => onTabChange?.('overview') },
    { keys: ['Ctrl', '2'], description: 'Перейти к номенклатуре', action: () => onTabChange?.('nomenclature') },
    { keys: ['Ctrl', '3'], description: 'Перейти к контрагентам', action: () => onTabChange?.('counterparties') },
    { keys: ['Ctrl', '4'], description: 'Перейти к качеству', action: () => onTabChange?.('quality') },
    { keys: ['Ctrl', '5'], description: 'Перейти к классификации', action: () => onTabChange?.('classification') },
    { keys: ['Esc'], description: 'Закрыть диалог/модальное окно', action: () => setOpen(false) },
  ]

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    const isMac = navigator.platform.includes('Mac')
    const ctrlKey = isMac ? e.metaKey : e.ctrlKey

    // Ctrl+K или Cmd+K для открытия справки
    if (ctrlKey && e.key === 'k') {
      e.preventDefault()
      setOpen(prev => !prev)
      return
    }

    // Обработка других горячих клавиш
    if (ctrlKey && e.key === 'e') {
      e.preventDefault()
      onExport?.()
      return
    }

    if (ctrlKey && e.key === 'r') {
      e.preventDefault()
      onRefresh?.()
      return
    }

    if (ctrlKey && ['1', '2', '3', '4', '5'].includes(e.key)) {
      e.preventDefault()
      const tabMap: Record<string, string> = {
        '1': 'overview',
        '2': 'nomenclature',
        '3': 'counterparties',
        '4': 'quality',
        '5': 'classification',
      }
      onTabChange?.(tabMap[e.key])
      return
    }

    // Ctrl+K для поиска (если не открыта справка)
    if (ctrlKey && e.key === 'k' && !open) {
      e.preventDefault()
      onSearch?.()
      return
    }

    // Esc для закрытия
    if (e.key === 'Escape') {
      setOpen(false)
    }
  }, [onSearch, onExport, onRefresh, onTabChange, open])

  useEffect(() => {
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleKeyDown])

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      {children && <DialogTrigger asChild>{children}</DialogTrigger>}
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Keyboard className="h-5 w-5" />
            Горячие клавиши
          </DialogTitle>
          <DialogDescription>
            Клавиатурные сокращения для быстрой навигации
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          {shortcuts.map((shortcut, index) => (
            <div
              key={index}
              className="flex items-center justify-between p-3 border rounded-lg"
            >
              <span className="text-sm">{shortcut.description}</span>
              <div className="flex items-center gap-1">
                {shortcut.keys.map((key, keyIndex) => (
                  <React.Fragment key={keyIndex}>
                    {keyIndex > 0 && <span className="text-muted-foreground">+</span>}
                    <Badge variant="outline" className="font-mono">
                      {key === 'Ctrl' && navigator.platform.includes('Mac') ? '⌘' : key}
                    </Badge>
                  </React.Fragment>
                ))}
              </div>
            </div>
          ))}
        </div>
        <div className="mt-4 pt-4 border-t">
          <p className="text-xs text-muted-foreground">
            Нажмите Ctrl+K в любое время для открытия этого списка
          </p>
        </div>
      </DialogContent>
    </Dialog>
  )
}

