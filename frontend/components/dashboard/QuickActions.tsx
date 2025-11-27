'use client'

import { motion } from 'framer-motion'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { 
  Play, 
  BarChart3, 
  Activity, 
  CheckCircle2,
  Database,
  FileText
} from 'lucide-react'
import Link from 'next/link'

interface QuickAction {
  label: string
  description: string
  icon: typeof Play
  href: string
  variant?: 'default' | 'outline' | 'secondary'
}

const quickActions: QuickAction[] = [
  {
    label: 'Мониторинг',
    description: 'Реальное время',
    icon: Activity,
    href: '/monitoring',
    variant: 'outline',
  },
  {
    label: 'Результаты',
    description: 'Просмотр данных',
    icon: BarChart3,
    href: '/results',
    variant: 'outline',
  },
  {
    label: 'Качество',
    description: 'Анализ данных',
    icon: CheckCircle2,
    href: '/quality',
    variant: 'outline',
  },
  {
    label: 'Базы данных',
    description: 'Управление БД',
    icon: Database,
    href: '/databases',
    variant: 'outline',
  },
  {
    label: 'Отчеты',
    description: 'Генерация отчетов',
    icon: FileText,
    href: '/reports',
    variant: 'outline',
  },
]

export function QuickActions() {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Быстрые действия</CardTitle>
        <CardDescription>
          Переход к основным разделам системы
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          {quickActions.map((action, index) => {
            const Icon = action.icon
            return (
              <motion.div
                key={action.href}
                initial={{ opacity: 0, y: 10, scale: 0.95 }}
                animate={{ opacity: 1, y: 0, scale: 1 }}
                transition={{ 
                  delay: index * 0.05,
                  type: "spring",
                  stiffness: 200,
                  damping: 15
                }}
                whileHover={{ 
                  scale: 1.05, 
                  y: -4,
                  transition: { duration: 0.2 }
                }}
                whileTap={{ scale: 0.95 }}
                className="relative group"
              >
                <Button
                  variant={action.variant || 'outline'}
                  asChild
                  className="w-full h-auto py-4 flex flex-col items-start gap-2 relative overflow-hidden transition-all duration-300 group-hover:shadow-lg"
                >
                  <Link href={action.href}>
                    <motion.div
                      className="absolute inset-0 bg-gradient-to-r from-primary/10 to-transparent"
                      initial={{ x: '-100%' }}
                      whileHover={{ x: '100%' }}
                      transition={{ duration: 0.5 }}
                    />
                    <motion.div
                      animate={{ rotate: [0, 10, -10, 0] }}
                      transition={{ duration: 0.5, delay: index * 0.1 }}
                    >
                      <Icon className="h-5 w-5 relative z-10" />
                    </motion.div>
                    <div className="text-left relative z-10">
                      <div className="font-semibold">{action.label}</div>
                      <div className="text-xs text-muted-foreground">{action.description}</div>
                    </div>
                  </Link>
                </Button>
              </motion.div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}

