'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { 
  ArrowLeft,
  Plus,
  Target,
  FileText,
  Calendar,
  RefreshCw,
  Building2
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { ErrorState } from "@/components/common/error-state"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { AlertCircle } from "lucide-react"
import { formatDate } from '@/lib/locale'
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { motion } from "framer-motion"
import { FadeIn } from "@/components/animations/fade-in"
import { useRouter } from "next/navigation"

interface Project {
  id: number
  name: string
  project_type: string
  description: string
  status: string
  created_at: string
  updated_at: string
}

export default function ClientProjectsPage() {
  const params = useParams()
  const router = useRouter()
  const clientId = params.clientId
  const [projects, setProjects] = useState<Project[]>([])
  const [clientName, setClientName] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (clientId) {
      fetchProjects(clientId as string)
      fetchClientName(clientId as string)
    }
  }, [clientId])

  const fetchClientName = async (id: string) => {
    try {
      const response = await fetch(`/api/clients/${id}`)
      if (response.ok) {
        const data = await response.json()
        setClientName(data.client?.name || null)
      }
    } catch (error) {
      console.error('Failed to fetch client name:', error)
    }
  }

  const fetchProjects = async (id: string) => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await fetch(`/api/clients/${id}/projects`)
      if (!response.ok) {
        const errorText = await response.text().catch(() => 'Failed to fetch projects')
        throw new Error(errorText || 'Не удалось загрузить проекты')
      }
      const data = await response.json()
      // Обрабатываем как массив, так и объект с полем projects
      if (Array.isArray(data)) {
        setProjects(data)
      } else if (data && typeof data === 'object' && 'projects' in data) {
        setProjects(Array.isArray(data.projects) ? data.projects : [])
      } else {
        setProjects([])
      }
    } catch (error) {
      console.error('Failed to fetch projects:', error)
      setError(error instanceof Error ? error.message : 'Не удалось загрузить проекты')
      setProjects([])
    } finally {
      setIsLoading(false)
    }
  }

  const getProjectTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      nomenclature: 'Номенклатура',
      counterparties: 'Контрагенты',
      nomenclature_counterparties: 'Номенклатура + Контрагенты',
      mixed: 'Смешанный'
    }
    return labels[type] || type
  }

  const breadcrumbItems = [
    { label: 'Клиенты', href: '/clients', icon: Building2 },
    { label: clientName || 'Клиент', href: `/clients/${clientId}`, icon: Building2 },
    { label: 'Проекты', href: `/clients/${clientId}/projects`, icon: Target },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="flex items-center gap-4"
        >
          <Button 
            variant="outline" 
            size="icon"
            onClick={() => router.push(`/clients/${clientId}`)}
            aria-label="Назад к клиенту"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Target className="h-8 w-8 text-primary" />
              Проекты клиента
            </h1>
            <p className="text-muted-foreground mt-1">
              Управление проектами нормализации
            </p>
          </div>
          <Button asChild>
            <Link href={`/clients/${clientId}/projects/new`}>
              <Plus className="mr-2 h-4 w-4" />
              Новый проект
            </Link>
          </Button>
        </motion.div>
      </FadeIn>

      {error && (
        <Alert variant="destructive">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            {error}
            <Button
              variant="outline"
              size="sm"
              className="ml-4"
              onClick={() => clientId && fetchProjects(clientId as string)}
            >
              Повторить
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {isLoading ? (
        <LoadingState message="Загрузка проектов..." size="lg" fullScreen />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {projects.map((project) => (
            <Card key={project.id} className="hover:shadow-lg transition-shadow">
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Target className="h-5 w-5" />
                  {project.name}
                </CardTitle>
                <CardDescription className="line-clamp-2">
                  {project.description || 'Нет описания'}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Тип:</span>
                  <Badge variant="outline">{getProjectTypeLabel(project.project_type)}</Badge>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Статус:</span>
                  <Badge variant={project.status === 'active' ? 'default' : 'secondary'}>
                    {project.status === 'active' ? 'Активен' : project.status}
                  </Badge>
                </div>
                <div className="flex justify-between text-sm text-muted-foreground">
                  <span>Создан:</span>
                  <span>{formatDate(project.created_at)}</span>
                </div>
                <div className="flex gap-2 pt-2">
                  <Button asChild variant="outline" className="flex-1">
                    <Link href={`/clients/${clientId}/projects/${project.id}`}>
                      Открыть
                    </Link>
                  </Button>
                  <Button asChild className="flex-1">
                    <Link href={`/clients/${clientId}/projects/${project.id}/benchmarks`}>
                      Эталоны
                    </Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {projects.length === 0 && !isLoading && (
        <EmptyState
          icon={Target}
          title="Проекты не найдены"
          description="Создайте первый проект для этого клиента"
        />
      )}
    </div>
  )
}

