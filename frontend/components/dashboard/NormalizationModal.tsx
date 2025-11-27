'use client'

import { useState, useEffect } from 'react'
import { motion, AnimatePresence } from 'framer-motion'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Label } from '@/components/ui/label'
import { Checkbox } from '@/components/ui/checkbox'
import { Loader2, Play, AlertCircle, CheckCircle2 } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { apiClientJson } from '@/lib/api-client'
import { useError } from '@/contexts/ErrorContext'
import { useDashboardStore } from '@/stores/dashboard-store'
import { ConfettiEffect } from './ConfettiEffect'
import { LottieAnimation } from './LottieAnimation'

interface Client {
  id: number
  name: string
}

interface Project {
  id: number
  name: string
  client_id: number
}

interface NormalizationModalProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function NormalizationModal({ open, onOpenChange }: NormalizationModalProps) {
  const { handleError } = useError()
  const { addNotification } = useDashboardStore()
  const [clients, setClients] = useState<Client[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedClientId, setSelectedClientId] = useState<string | undefined>(undefined)
  const [selectedProjectId, setSelectedProjectId] = useState<string | undefined>(undefined)
  const [useKpved, setUseKpved] = useState(false)
  const [useOkpd2, setUseOkpd2] = useState(false)
  const [isLoading, setIsLoading] = useState(false)
  const [isLoadingProjects, setIsLoadingProjects] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showSuccess, setShowSuccess] = useState(false)
  const [confettiTrigger, setConfettiTrigger] = useState(false)

  useEffect(() => {
    if (open) {
      loadClients()
    }
  }, [open])

  useEffect(() => {
    if (selectedClientId) {
      loadProjects(parseInt(selectedClientId))
    } else {
      setProjects([])
      setSelectedProjectId(undefined)
    }
  }, [selectedClientId])

  const loadClients = async () => {
    try {
      const data = await apiClientJson<Client[]>('/api/clients', { skipErrorHandler: true })
      setClients(Array.isArray(data) ? data : [])
    } catch (err) {
      try {
        handleError(err, 'Не удалось загрузить список клиентов')
      } catch {
        // Игнорируем ошибки обработки ошибок
      }
      setClients([])
    }
  }

  const loadProjects = async (clientId: number) => {
    if (!clientId || isNaN(clientId) || !isFinite(clientId)) {
      setProjects([])
      return
    }
    
    try {
      setIsLoadingProjects(true)
      const data = await apiClientJson<Project[] | { projects: Project[] }>(
        `/api/clients/${clientId}/projects`,
        { skipErrorHandler: true }
      )
      // Обрабатываем как массив, так и объект с полем projects
      // Legacy handler возвращает массив напрямую, новый handler возвращает объект
      if (Array.isArray(data)) {
        setProjects(data)
      } else if (data && typeof data === 'object' && 'projects' in data) {
        const projectsData = (data as { projects?: Project[] }).projects
        setProjects(Array.isArray(projectsData) ? projectsData : [])
      } else {
        setProjects([])
      }
    } catch (err) {
      try {
        handleError(err, 'Не удалось загрузить список проектов')
      } catch {
        // Игнорируем ошибки обработки ошибок
      }
      setProjects([])
    } finally {
      setIsLoadingProjects(false)
    }
  }

  const handleStart = async () => {
    if (!selectedClientId || !selectedProjectId) {
      setError('Пожалуйста, выберите клиента и проект')
      return
    }

    setIsLoading(true)
    setError(null)

    try {
      await apiClientJson(
        `/api/clients/${selectedClientId}/projects/${selectedProjectId}/normalization/start`,
        {
          method: 'POST',
          body: JSON.stringify({
            all_active: true,
            use_kpved: useKpved,
            use_okpd2: useOkpd2,
          }),
        }
      )

      // Показываем успешную анимацию
      setShowSuccess(true)
      setConfettiTrigger(true)
      
      addNotification({
        type: 'success',
        title: 'Нормализация запущена',
        message: 'Процесс нормализации успешно запущен',
      })

      // Закрываем модальное окно через 2 секунды после анимации
      setTimeout(() => {
        onOpenChange(false)
        setShowSuccess(false)
        setConfettiTrigger(false)
        
        // Сброс формы
        setSelectedClientId(undefined)
        setSelectedProjectId(undefined)
        setUseKpved(false)
        setUseOkpd2(false)
      }, 2000)
    } catch (err) {
      const errorMessage = 'Не удалось запустить нормализацию'
      setError(errorMessage)
      handleError(err, errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <>
      <ConfettiEffect trigger={confettiTrigger} type="success" />
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="sm:max-w-[500px]">
          <AnimatePresence mode="wait">
            {showSuccess ? (
              <motion.div
                key="success"
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                exit={{ opacity: 0, scale: 0.8 }}
                className="flex flex-col items-center justify-center py-8"
              >
                <motion.div
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  transition={{ type: "spring", stiffness: 200, damping: 15 }}
                  className="mb-4"
                >
                  <LottieAnimation
                    src="https://assets5.lottiefiles.com/packages/lf20_jcikwtux.json"
                    loop={false}
                    autoplay={true}
                    fallback={
                      <div className="w-32 h-32 flex items-center justify-center">
                        <motion.div
                          animate={{ scale: [1, 1.2, 1], rotate: [0, 180, 360] }}
                          transition={{ duration: 1 }}
                        >
                          <CheckCircle2 className="h-16 w-16 text-green-500" />
                        </motion.div>
                      </div>
                    }
                  />
                </motion.div>
                <motion.h3
                  initial={{ opacity: 0, y: 10 }}
                  animate={{ opacity: 1, y: 0 }}
                  transition={{ delay: 0.2 }}
                  className="text-xl font-semibold mb-2"
                >
                  Нормализация запущена!
                </motion.h3>
                <motion.p
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  transition={{ delay: 0.3 }}
                  className="text-muted-foreground text-center"
                >
                  Процесс нормализации успешно запущен
                </motion.p>
              </motion.div>
            ) : (
              <motion.div
                key="form"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                exit={{ opacity: 0 }}
              >
                <DialogHeader>
                  <DialogTitle className="flex items-center gap-2">
                    <motion.div
                      animate={{ rotate: [0, 10, -10, 0] }}
                      transition={{ duration: 0.5, repeat: Infinity, repeatDelay: 2 }}
                    >
                      <Play className="h-5 w-5" />
                    </motion.div>
                    Запуск нормализации
                  </DialogTitle>
                  <DialogDescription>
                    Выберите клиента и проект для запуска процесса нормализации данных
                  </DialogDescription>
                </DialogHeader>

        <div className="space-y-4 py-4">
          {error && (
            <Alert variant="destructive">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription>{error}</AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="client">Клиент</Label>
            <Select value={selectedClientId || undefined} onValueChange={setSelectedClientId}>
              <SelectTrigger id="client">
                <SelectValue placeholder="Выберите клиента" />
              </SelectTrigger>
              <SelectContent>
                {Array.isArray(clients) && clients.map((client) => (
                  client && client.id ? (
                    <SelectItem key={client.id} value={client.id.toString()}>
                      {client.name || `Клиент ${client.id}`}
                    </SelectItem>
                  ) : null
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="project">Проект</Label>
            <Select
              value={selectedProjectId || undefined}
              onValueChange={setSelectedProjectId}
              disabled={!selectedClientId || isLoadingProjects}
            >
              <SelectTrigger id="project">
                <SelectValue
                  placeholder={
                    isLoadingProjects
                      ? 'Загрузка...'
                      : !selectedClientId
                      ? 'Сначала выберите клиента'
                      : 'Выберите проект'
                  }
                />
              </SelectTrigger>
              <SelectContent>
                {Array.isArray(projects) && projects.map((project) => (
                  project && project.id ? (
                    <SelectItem key={project.id} value={project.id.toString()}>
                      {project.name || `Проект ${project.id}`}
                    </SelectItem>
                  ) : null
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-3 pt-2">
            <Label>Опции классификации</Label>
            <div className="space-y-2">
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="kpved"
                  checked={useKpved}
                  onCheckedChange={(checked) => setUseKpved(checked === true)}
                />
                <Label
                  htmlFor="kpved"
                  className="text-sm font-normal cursor-pointer"
                >
                  Использовать КПВЭД
                </Label>
              </div>
              <div className="flex items-center space-x-2">
                <Checkbox
                  id="okpd2"
                  checked={useOkpd2}
                  onCheckedChange={(checked) => setUseOkpd2(checked === true)}
                />
                <Label
                  htmlFor="okpd2"
                  className="text-sm font-normal cursor-pointer"
                >
                  Использовать ОКПД2
                </Label>
              </div>
            </div>
          </div>
        </div>

                <div className="flex justify-end gap-2">
                  <motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                    <Button variant="outline" onClick={() => onOpenChange(false)}>
                      Отмена
                    </Button>
                  </motion.div>
                  <motion.div whileHover={{ scale: 1.02 }} whileTap={{ scale: 0.98 }}>
                    <Button onClick={handleStart} disabled={isLoading || !selectedClientId || !selectedProjectId}>
                      {isLoading ? (
                        <>
                          <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                          Запуск...
                        </>
                      ) : (
                        <>
                          <Play className="h-4 w-4 mr-2" />
                          Запустить
                        </>
                      )}
                    </Button>
                  </motion.div>
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </DialogContent>
      </Dialog>
    </>
  )
}

