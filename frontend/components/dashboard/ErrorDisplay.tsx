'use client'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { AlertCircle, RefreshCw } from 'lucide-react'
import { motion } from 'framer-motion'
import { useDashboardStore } from '@/stores/dashboard-store'

interface ErrorDisplayProps {
  error: string | null
  onRetry?: () => void
  className?: string
}

export function ErrorDisplay({ error, onRetry, className }: ErrorDisplayProps) {
  const { setError } = useDashboardStore()

  if (!error || typeof error !== 'string') return null

  const handleRetry = () => {
    try {
      setError(null)
      if (onRetry) {
        onRetry()
      }
    } catch {
      // Игнорируем ошибки установки состояния
    }
  }

  const handleClose = () => {
    try {
      setError(null)
    } catch {
      // Игнорируем ошибки установки состояния
    }
  }

  return (
    <motion.div
      initial={{ opacity: 0, y: -10, scale: 0.95 }}
      animate={{ opacity: 1, y: 0, scale: 1 }}
      exit={{ opacity: 0, y: -10, scale: 0.95 }}
      transition={{ 
        type: "spring",
        stiffness: 300,
        damping: 25
      }}
      className={className}
    >
      <Alert variant="destructive" className="relative overflow-hidden">
        <motion.div
          className="absolute inset-0 bg-destructive/10"
          animate={{ 
            x: ['-100%', '100%'],
          }}
          transition={{
            duration: 2,
            repeat: Infinity,
            repeatDelay: 3,
          }}
        />
        <AlertCircle className="h-4 w-4 relative z-10" />
        <AlertTitle className="relative z-10">Ошибка</AlertTitle>
        <AlertDescription className="flex items-center justify-between relative z-10">
          <span>{error}</span>
          <div className="flex gap-2">
            {onRetry && (
              <motion.div
                whileHover={{ scale: 1.05 }}
                whileTap={{ scale: 0.95 }}
              >
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleRetry}
                >
                  <motion.div
                    animate={{ rotate: 360 }}
                    transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                    style={{ display: 'inline-block' }}
                  >
                    <RefreshCw className="h-3 w-3 mr-1" />
                  </motion.div>
                  Повторить
                </Button>
              </motion.div>
            )}
            <motion.div
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClose}
              >
                Закрыть
              </Button>
            </motion.div>
          </div>
        </AlertDescription>
      </Alert>
    </motion.div>
  )
}

