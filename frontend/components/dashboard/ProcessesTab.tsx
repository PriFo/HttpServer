'use client'

import { useEffect, useState } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { PlayCircle, Square, RefreshCw, Activity } from 'lucide-react'
import { useDashboardStore } from '@/stores/dashboard-store'
import { apiClientJson } from '@/lib/api-client'
import { Skeleton } from '@/components/ui/skeleton'
import { ProcessMonitor } from '@/components/process-monitor'
import { cn } from '@/lib/utils'
import { ProjectSelector } from '@/components/project-selector'
import { NormalizationPreviewStats } from '@/components/processes/normalization-preview-stats'
import { toast } from 'sonner'

interface NormalizationStatus {
  status: 'idle' | 'running' | 'completed' | 'error'
  progress: number
  currentStage: string
  startTime: string | null
  endTime: string | null
}

export function ProcessesTab() {
  const { systemStats, setSystemStats, isLoading, setLoading } = useDashboardStore()
  // Удалены дублирующиеся запросы - ProcessMonitor уже делает запросы к statusEndpoint
  // Статусы теперь управляются только через ProcessMonitor компоненты
  const [nomenclatureStatus, setNomenclatureStatus] = useState<NormalizationStatus>({
    status: 'idle',
    progress: 0,
    currentStage: 'Ожидание запуска',
    startTime: null,
    endTime: null,
  })
  const [counterpartiesStatus, setCounterpartiesStatus] = useState<NormalizationStatus>({
    status: 'idle',
    progress: 0,
    currentStage: 'Ожидание запуска',
    startTime: null,
    endTime: null,
  })
  
  // Состояние для выбора клиента и проекта
  const [selectedProject, setSelectedProject] = useState<string>('')
  const [clientId, setClientId] = useState<number | null>(null)
  const [projectId, setProjectId] = useState<number | null>(null)
  
  // Обновляем clientId и projectId при изменении selectedProject
  useEffect(() => {
    if (selectedProject) {
      const parts = selectedProject.split(':')
      if (parts.length === 2) {
        const cId = parseInt(parts[0], 10)
        const pId = parseInt(parts[1], 10)
        if (!isNaN(cId) && !isNaN(pId)) {
          setClientId(cId)
          setProjectId(pId)
        } else {
          setClientId(null)
          setProjectId(null)
        }
      } else {
        setClientId(null)
        setProjectId(null)
      }
    } else {
      setClientId(null)
      setProjectId(null)
    }
  }, [selectedProject])

  // Убраны автоматические запросы - ProcessMonitor делает запросы самостоятельно
  // Это предотвращает дублирование запросов и бесконечный спам при 404 ошибках

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-blue-500'
      case 'completed':
        return 'bg-green-500'
      case 'error':
        return 'bg-red-500'
      default:
        return 'bg-gray-400'
    }
  }

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'running':
        return <Badge variant="default">Выполняется</Badge>
      case 'completed':
        return <Badge variant="default" className="bg-green-600">Завершено</Badge>
      case 'error':
        return <Badge variant="destructive">Ошибка</Badge>
      default:
        return <Badge variant="secondary">Ожидание</Badge>
    }
  }

  // Функции для запуска процессов с client_id и project_id
  const handleStartNomenclature = async () => {
    if (!clientId || !projectId) {
      toast.error('Выберите клиента и проект', {
        description: 'Для запуска процесса нормализации необходимо выбрать клиента и проект',
      })
      return
    }
    
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/normalization/start`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      })
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Не удалось запустить процесс' }))
        throw new Error(errorData.error || 'Не удалось запустить процесс')
      }
      
      toast.success('Процесс запущен', {
        description: 'Нормализация номенклатуры успешно запущена',
      })
      
      // Обновляем статус после запуска
      setTimeout(() => {
        window.location.reload()
      }, 1000)
    } catch (err) {
      console.error('Error starting nomenclature normalization:', err)
      toast.error('Ошибка запуска процесса', {
        description: err instanceof Error ? err.message : 'Не удалось запустить процесс',
      })
    }
  }
  
  const handleStartCounterparties = async () => {
    if (!clientId || !projectId) {
      toast.error('Выберите клиента и проект', {
        description: 'Для запуска процесса нормализации необходимо выбрать клиента и проект',
      })
      return
    }
    
    try {
      const response = await fetch(`/api/counterparties/normalization/start?client_id=${clientId}&project_id=${projectId}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({}),
      })
      
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Не удалось запустить процесс' }))
        throw new Error(errorData.error || 'Не удалось запустить процесс')
      }
      
      toast.success('Процесс запущен', {
        description: 'Нормализация контрагентов успешно запущена',
      })
      
      // Обновляем статус после запуска
      setTimeout(() => {
        window.location.reload()
      }, 1000)
    } catch (err) {
      console.error('Error starting counterparties normalization:', err)
      toast.error('Ошибка запуска процесса', {
        description: err instanceof Error ? err.message : 'Не удалось запустить процесс',
      })
    }
  }
  
  const handleStopNomenclature = async () => {
    try {
      const response = await fetch('/api/normalization/stop', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })
      
      if (!response.ok) {
        throw new Error('Не удалось остановить процесс')
      }
      
      toast.success('Процесс остановлен', {
        description: 'Нормализация номенклатуры остановлена',
      })
      
      setTimeout(() => {
        window.location.reload()
      }, 1000)
    } catch (err) {
      console.error('Error stopping nomenclature normalization:', err)
      toast.error('Ошибка остановки процесса', {
        description: err instanceof Error ? err.message : 'Не удалось остановить процесс',
      })
    }
  }
  
  const handleStopCounterparties = async () => {
    try {
      const response = await fetch('/api/counterparties/normalization/stop', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
      })
      
      if (!response.ok) {
        throw new Error('Не удалось остановить процесс')
      }
      
      toast.success('Процесс остановлен', {
        description: 'Нормализация контрагентов остановлена',
      })
      
      setTimeout(() => {
        window.location.reload()
      }, 1000)
    } catch (err) {
      console.error('Error stopping counterparties normalization:', err)
      toast.error('Ошибка остановки процесса', {
        description: err instanceof Error ? err.message : 'Не удалось остановить процесс',
      })
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      
      <div>
        <h2 className="text-2xl font-bold flex items-center gap-2">
          <Activity className="h-6 w-6" />
          Процессы нормализации
        </h2>
        <p className="text-muted-foreground mt-1">
          Управление и мониторинг процессов нормализации данных
        </p>
      </div>
      
      {/* Выбор клиента и проекта */}
      <Card>
        <CardHeader>
          <CardTitle>Выбор проекта</CardTitle>
          <CardDescription>
            Выберите клиента и проект для запуска процессов нормализации
          </CardDescription>
        </CardHeader>
        <CardContent>
          <ProjectSelector
            value={selectedProject}
            onChange={setSelectedProject}
            placeholder="Выберите проект"
            className="w-full"
          />
        </CardContent>
      </Card>
      
      {/* Предварительная статистика */}
      {clientId && projectId && (
        <NormalizationPreviewStats
          clientId={clientId}
          projectId={projectId}
        />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Nomenclature Process */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
        >
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Нормализация номенклатуры</CardTitle>
                {getStatusBadge(nomenclatureStatus.status)}
              </div>
              <CardDescription>
                Обработка и нормализация товаров и услуг
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <ProcessMonitor
                title="Номенклатура"
                statusEndpoint={clientId && projectId 
                  ? `/api/clients/${clientId}/projects/${projectId}/normalization/status`
                  : "/api/normalization/status"}
                startEndpoint={clientId && projectId ? undefined : "/api/normalization/start"}
                stopEndpoint="/api/normalization/stop"
                eventsEndpoint={clientId && projectId
                  ? `/api/clients/${clientId}/projects/${projectId}/normalization/events`
                  : "/api/normalization/events"}
                onStart={clientId && projectId ? handleStartNomenclature : undefined}
                onStop={handleStopNomenclature}
              />
              {!clientId || !projectId ? (
                <div className="text-sm text-muted-foreground p-2 bg-yellow-50 dark:bg-yellow-950 rounded">
                  Для запуска процесса выберите клиента и проект выше
                </div>
              ) : null}
            </CardContent>
          </Card>
        </motion.div>

        {/* Counterparties Process */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Нормализация контрагентов</CardTitle>
                {getStatusBadge(counterpartiesStatus.status)}
              </div>
              <CardDescription>
                Обработка и нормализация данных контрагентов
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <ProcessMonitor
                title="Контрагенты"
                statusEndpoint={clientId && projectId
                  ? `/api/counterparties/normalization/status?client_id=${clientId}&project_id=${projectId}`
                  : "/api/counterparties/normalization/status"}
                startEndpoint={clientId && projectId ? undefined : "/api/counterparties/normalization/start"}
                stopEndpoint="/api/counterparties/normalization/stop"
                eventsEndpoint={clientId && projectId
                  ? `/api/counterparties/normalization/events?client_id=${clientId}&project_id=${projectId}`
                  : "/api/counterparties/normalization/events"}
                onStart={clientId && projectId ? handleStartCounterparties : undefined}
                onStop={handleStopCounterparties}
              />
              {!clientId || !projectId ? (
                <div className="text-sm text-muted-foreground p-2 bg-yellow-50 dark:bg-yellow-950 rounded">
                  Для запуска процесса выберите клиента и проект выше
                </div>
              ) : null}
            </CardContent>
          </Card>
        </motion.div>
      </div>

      {/* Quick Actions */}
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ delay: 0.2 }}
      >
        <Card>
          <CardHeader>
            <CardTitle>Быстрые действия</CardTitle>
            <CardDescription>
              Переход к детальным страницам процессов
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <Button variant="outline" asChild>
                <a href="/processes/nomenclature">
                  <Activity className="h-4 w-4 mr-2" />
                  Детали нормализации номенклатуры
                </a>
              </Button>
              <Button variant="outline" asChild>
                <a href="/processes/counterparties">
                  <Activity className="h-4 w-4 mr-2" />
                  Детали нормализации контрагентов
                </a>
              </Button>
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}

