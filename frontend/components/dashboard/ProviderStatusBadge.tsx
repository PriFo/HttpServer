'use client'

import { motion } from 'framer-motion'
import { Badge } from '@/components/ui/badge'
import { CheckCircle2, AlertCircle, XCircle, Clock } from 'lucide-react'
import { cn } from '@/lib/utils'

interface ProviderStatusBadgeProps {
  status: 'active' | 'idle' | 'error'
  className?: string
}

const statusConfig = {
  active: {
    label: 'Активен',
    icon: CheckCircle2,
    variant: 'default' as const,
    className: 'bg-green-500 text-white',
  },
  idle: {
    label: 'Ожидание',
    icon: Clock,
    variant: 'secondary' as const,
    className: 'bg-gray-400 text-white',
  },
  error: {
    label: 'Ошибка',
    icon: XCircle,
    variant: 'destructive' as const,
    className: 'bg-red-500 text-white',
  },
}

export function ProviderStatusBadge({ status, className }: ProviderStatusBadgeProps) {
  const config = statusConfig[status]
  const Icon = config.icon

  return (
    <motion.div
      initial={{ scale: 0.8, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      transition={{ duration: 0.2 }}
      whileHover={{ scale: 1.05 }}
      whileTap={{ scale: 0.95 }}
    >
      <Badge
        variant={config.variant}
        className={cn('flex items-center gap-1 transition-all', config.className, className)}
      >
        <motion.div
          animate={status === 'active' ? { 
            rotate: [0, 10, -10, 0],
            scale: [1, 1.1, 1]
          } : {}}
          transition={{ 
            duration: 2, 
            repeat: status === 'active' ? Infinity : 0,
            repeatDelay: 1
          }}
        >
          <Icon className="h-3 w-3" />
        </motion.div>
        {config.label}
      </Badge>
    </motion.div>
  )
}

