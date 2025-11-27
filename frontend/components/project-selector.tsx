'use client'

import { useState, useEffect } from 'react'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Button } from '@/components/ui/button'
import { RefreshCw } from 'lucide-react'
import { cn } from '@/lib/utils'
import { apiRequest } from '@/lib/api-utils'

interface Client {
  id: number
  name: string
}

interface Project {
  id: number
  name: string
  client_id: number
}

interface ProjectSelectorProps {
  value?: string // Формат: "clientId:projectId" или пустая строка
  onChange: (value: string) => void
  placeholder?: string
  className?: string
  onRefresh?: () => void
}

export function ProjectSelector({
  value,
  onChange,
  placeholder = "Выберите проект",
  className,
  onRefresh,
}: ProjectSelectorProps) {
  const [clients, setClients] = useState<Client[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [selectedClientId, setSelectedClientId] = useState<number | null>(null)
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  // Парсим значение из пропса
  useEffect(() => {
    if (value) {
      const parts = value.split(':')
      if (parts.length === 2) {
        const clientId = parseInt(parts[0], 10)
        const projectId = parseInt(parts[1], 10)
        if (!isNaN(clientId) && !isNaN(projectId)) {
          setSelectedClientId(clientId)
          setSelectedProjectId(projectId)
        }
      }
    } else {
      setSelectedClientId(null)
      setSelectedProjectId(null)
    }
  }, [value])

  // Загружаем клиентов
  const fetchClients = async (force = false) => {
    if (loading && !force) return
    
    setLoading(true)
    setError(null)
    
    try {
      const data = await apiRequest<Client[]>('/api/clients')
      setClients(data || [])
      setError(null)
    } catch (err) {
      console.error('Failed to fetch clients:', err)
      let errorMessage = 'Не удалось загрузить клиентов'
      if (err instanceof Error) {
        if (err.message.includes('Превышено время ожидания')) {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (err.message.includes('Не удалось подключиться')) {
          errorMessage = 'Не удалось подключиться к backend серверу. Проверьте подключение на порту 9999'
        } else {
          errorMessage = err.message || errorMessage
        }
      }
      setError(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  // Загружаем проекты клиента
  const fetchProjects = async (clientId: number, force = false) => {
    if (loading && !force) return
    
    setLoading(true)
    setError(null)
    
    try {
      const data = await apiRequest<Project[] | { projects: Project[] }>(`/api/clients/${clientId}/projects`)
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
      setError(null)
    } catch (err) {
      console.error('Failed to fetch projects:', err)
      let errorMessage = 'Не удалось загрузить проекты'
      if (err instanceof Error) {
        if (err.message.includes('Превышено время ожидания')) {
          errorMessage = 'Превышено время ожидания ответа от сервера'
        } else if (err.message.includes('Не удалось подключиться')) {
          errorMessage = 'Не удалось подключиться к backend серверу. Проверьте подключение на порту 9999'
        } else {
          errorMessage = err.message || errorMessage
        }
      }
      setError(errorMessage)
      setProjects([])
    } finally {
      setLoading(false)
    }
  }

  // Загружаем клиентов при монтировании
  useEffect(() => {
    fetchClients()
  }, [])

  // Загружаем проекты при выборе клиента
  useEffect(() => {
    if (selectedClientId) {
      fetchProjects(selectedClientId)
    } else {
      setProjects([])
      setSelectedProjectId(null)
    }
  }, [selectedClientId])

  const handleClientChange = (clientIdStr: string) => {
    if (clientIdStr === 'all' || clientIdStr === '') {
      setSelectedClientId(null)
      setSelectedProjectId(null)
      onChange('')
      return
    }
    
    const clientId = parseInt(clientIdStr, 10)
    if (!isNaN(clientId)) {
      setSelectedClientId(clientId)
      setSelectedProjectId(null)
      onChange('')
    }
  }

  const handleProjectChange = (projectIdStr: string) => {
    if (projectIdStr === 'all' || projectIdStr === '' || !selectedClientId) {
      setSelectedProjectId(null)
      onChange('')
      return
    }
    
    const projectId = parseInt(projectIdStr, 10)
    if (!isNaN(projectId)) {
      setSelectedProjectId(projectId)
      onChange(`${selectedClientId}:${projectId}`)
    }
  }

  const handleRefresh = () => {
    if (selectedClientId) {
      fetchProjects(selectedClientId, true)
    } else {
      fetchClients(true)
    }
    onRefresh?.()
  }

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <Select
        value={selectedClientId?.toString() || 'all'}
        onValueChange={handleClientChange}
        disabled={loading}
      >
        <SelectTrigger className="w-[200px]">
          <SelectValue placeholder="Выберите клиента" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Все клиенты</SelectItem>
          {clients.map((client) => (
            <SelectItem key={client.id} value={client.id.toString()}>
              {client.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Select
        value={selectedProjectId?.toString() || 'all'}
        onValueChange={handleProjectChange}
        disabled={loading || !selectedClientId}
      >
        <SelectTrigger className="w-[250px]">
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="all">Все проекты</SelectItem>
          {projects.map((project) => (
            <SelectItem key={project.id} value={project.id.toString()}>
              {project.name}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>

      <Button
        variant="outline"
        size="icon"
        onClick={handleRefresh}
        disabled={loading}
        className="h-10 w-10"
        title="Обновить список"
      >
        <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
      </Button>
    </div>
  )
}

