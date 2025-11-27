'use client'

import { Badge } from '@/components/ui/badge'
import { Database, Server, Upload, Award, Layers, HelpCircle } from 'lucide-react'

interface DatabaseTypeBadgeProps {
  type: string
  className?: string
}

const typeConfig: Record<string, { label: string; variant: 'default' | 'secondary' | 'destructive' | 'outline'; icon: typeof Database }> = {
  service: {
    label: 'Сервисная',
    variant: 'default',
    icon: Server,
  },
  uploads: {
    label: 'Выгрузки',
    variant: 'secondary',
    icon: Upload,
  },
  benchmarks: {
    label: 'Эталоны',
    variant: 'outline',
    icon: Award,
  },
  combined: {
    label: 'Комбинированная',
    variant: 'default',
    icon: Layers,
  },
  unknown: {
    label: 'Неизвестная',
    variant: 'outline',
    icon: HelpCircle,
  },
}

export function DatabaseTypeBadge({ type, className }: DatabaseTypeBadgeProps) {
  const config = typeConfig[type] || typeConfig.unknown
  const Icon = config.icon

  return (
    <Badge variant={config.variant} className={`gap-1 ${className || ''}`}>
      <Icon className="h-3 w-3" />
      {config.label}
    </Badge>
  )
}

