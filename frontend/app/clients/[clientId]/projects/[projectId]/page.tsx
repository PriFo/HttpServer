'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  ArrowLeft,
  Target,
  BarChart3,
  Play,
  FileText,
  RefreshCw,
  Database,
  Plus,
  Trash2,
  AlertCircle
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { StatCard } from "@/components/common/stat-card"

interface ProjectDetail {
  project: {
    id: number
    name: string
    project_type: string
    description: string
    status: string
    created_at: string
  }
  benchmarks: Array<{
    id: number
    normalized_name: string
    category: string
    is_approved: boolean
  }>
  statistics: {
    total_benchmarks: number
    approved_benchmarks: number
    avg_quality_score: number
  }
}

interface ProjectDatabase {
  id: number
  client_project_id: number
  name: string
  file_path: string
  description: string
  is_active: boolean
  file_size: number
  created_at: string
  updated_at: string
}

export default function ProjectDetailPage() {
  const params = useParams()
  const clientId = params.clientId
  const projectId = params.projectId
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [databases, setDatabases] = useState<ProjectDatabase[]>([])
  const [showAddDatabase, setShowAddDatabase] = useState(false)
  const [newDatabase, setNewDatabase] = useState({ name: '', file_path: '', description: '' })
  const [databaseError, setDatabaseError] = useState<string | null>(null)
  const [isAddingDatabase, setIsAddingDatabase] = useState(false)

  useEffect(() => {
    if (clientId && projectId) {
      fetchProjectDetail(clientId as string, projectId as string)
      fetchDatabases()
    }
  }, [clientId, projectId])

  const fetchProjectDetail = async (clientId: string, projectId: string) => {
    setIsLoading(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}`)
      if (!response.ok) throw new Error('Failed to fetch project details')
      const data = await response.json()
      setProject(data)
    } catch (error) {
      console.error('Failed to fetch project details:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const fetchDatabases = async () => {
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`)
      if (!response.ok) throw new Error('Failed to fetch databases')
      const data = await response.json()
      setDatabases(data.databases || [])
    } catch (error) {
      console.error('Failed to fetch databases:', error)
    }
  }

  const handleAddDatabase = async () => {
    if (!newDatabase.name.trim() || !newDatabase.file_path.trim()) {
      setDatabaseError('Название и путь к файлу обязательны')
      return
    }

    setIsAddingDatabase(true)
    setDatabaseError(null)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newDatabase)
      })

      if (!response.ok) {
        const errorData = await response.json()
        setDatabaseError(errorData.error || 'Не удалось добавить базу данных')
        return
      }

      setNewDatabase({ name: '', file_path: '', description: '' })
      setShowAddDatabase(false)
      await fetchDatabases()
    } catch (error) {
      console.error('Failed to add database:', error)
      setDatabaseError('Ошибка подключения к серверу')
    } finally {
      setIsAddingDatabase(false)
    }
  }

  const handleDeleteDatabase = async (dbId: number) => {
    if (!confirm('Вы уверены, что хотите удалить эту базу данных?')) {
      return
    }

    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases/${dbId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        throw new Error('Failed to delete database')
      }

      await fetchDatabases()
    } catch (error) {
      console.error('Failed to delete database:', error)
    }
  }

  const getProjectTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      nomenclature: 'Номенклатура',
      counterparties: 'Контрагенты',
      mixed: 'Смешанный'
    }
    return labels[type] || type
  }

  if (isLoading) {
    return (
      <div className="container mx-auto p-6">
        <LoadingState message="Загрузка данных проекта..." size="lg" fullScreen />
      </div>
    )
  }

  if (!project) {
    return (
      <div className="container mx-auto p-6">
        <EmptyState
          icon={Target}
          title="Проект не найден"
          description="Проект не существует или был удален"
        />
      </div>
    )
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" asChild>
          <Link href={`/clients/${clientId}/projects`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">{project.project.name}</h1>
          <p className="text-muted-foreground">{project.project.description}</p>
        </div>
        <div className="flex gap-2">
          <Button asChild>
            <Link href={`/clients/${clientId}/projects/${projectId}/normalization`}>
              <Play className="mr-2 h-4 w-4" />
              Запустить нормализацию
            </Link>
          </Button>
        </div>
      </div>

      {/* Статистика */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        <StatCard
          title="Всего эталонов"
          value={project.statistics.total_benchmarks}
          description={`${project.statistics.approved_benchmarks} утверждено`}
          icon={FileText}
          variant="primary"
        />
        <StatCard
          title="Среднее качество"
          value={`${Math.round(project.statistics.avg_quality_score * 100)}%`}
          description="качество эталонов"
          variant={project.statistics.avg_quality_score >= 0.9 ? 'success' : project.statistics.avg_quality_score >= 0.7 ? 'warning' : 'danger'}
          progress={project.statistics.avg_quality_score * 100}
        />
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Тип проекта</CardTitle>
          </CardHeader>
          <CardContent>
            <Badge variant="outline" className="text-lg">
              {getProjectTypeLabel(project.project.project_type)}
            </Badge>
          </CardContent>
        </Card>
      </div>

      {/* Действия */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <FileText className="h-5 w-5" />
              Управление эталонами
            </CardTitle>
            <CardDescription>
              Просмотр и управление эталонными записями
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild className="w-full">
              <Link href={`/clients/${clientId}/projects/${projectId}/benchmarks`}>
                Открыть эталоны
              </Link>
            </Button>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5" />
              Нормализация
            </CardTitle>
            <CardDescription>
              Запуск процесса нормализации для этого проекта
            </CardDescription>
          </CardHeader>
          <CardContent>
            <Button asChild className="w-full">
              <Link href={`/clients/${clientId}/projects/${projectId}/normalization`}>
                <Play className="mr-2 h-4 w-4" />
                Запустить нормализацию
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* Базы данных */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="flex items-center gap-2">
                <Database className="h-5 w-5" />
                Базы данных проекта
              </CardTitle>
              <CardDescription>
                Управление базами данных для нормализации
              </CardDescription>
            </div>
            <Button onClick={() => setShowAddDatabase(!showAddDatabase)} size="sm">
              <Plus className="mr-2 h-4 w-4" />
              Добавить базу данных
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {showAddDatabase && (
            <Card className="border-2 border-primary/20">
              <CardHeader>
                <CardTitle className="text-base">Новая база данных</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="db-name">Название</Label>
                  <Input
                    id="db-name"
                    placeholder="Например: МПФ"
                    value={newDatabase.name}
                    onChange={(e) => setNewDatabase({ ...newDatabase, name: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="db-path">Путь к файлу</Label>
                  <Input
                    id="db-path"
                    placeholder="E:\HttpServer\1c_data.db"
                    value={newDatabase.file_path}
                    onChange={(e) => setNewDatabase({ ...newDatabase, file_path: e.target.value })}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="db-description">Описание (необязательно)</Label>
                  <Input
                    id="db-description"
                    placeholder="Описание базы данных"
                    value={newDatabase.description}
                    onChange={(e) => setNewDatabase({ ...newDatabase, description: e.target.value })}
                  />
                </div>
                {databaseError && (
                  <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>{databaseError}</AlertDescription>
                  </Alert>
                )}
                <div className="flex gap-2">
                  <Button
                    onClick={handleAddDatabase}
                    disabled={isAddingDatabase}
                    className="flex-1"
                  >
                    {isAddingDatabase ? 'Добавление...' : 'Добавить'}
                  </Button>
                  <Button
                    onClick={() => {
                      setShowAddDatabase(false)
                      setDatabaseError(null)
                      setNewDatabase({ name: '', file_path: '', description: '' })
                    }}
                    variant="outline"
                    className="flex-1"
                  >
                    Отмена
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {databases.length === 0 ? (
            <EmptyState
              icon={Database}
              title="Нет добавленных баз данных"
              description="Добавьте базу данных для начала работы"
            />
          ) : (
            <div className="space-y-2">
              {databases.map((db) => (
                <Card key={db.id} className="hover:shadow-md transition-shadow">
                  <CardContent className="pt-6">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <Database className="h-4 w-4 text-primary" />
                          <h4 className="font-semibold">{db.name}</h4>
                          {db.is_active && <Badge variant="default">Активна</Badge>}
                        </div>
                        <p className="text-sm text-muted-foreground mt-1 font-mono">
                          {db.file_path}
                        </p>
                        {db.description && (
                          <p className="text-sm text-muted-foreground mt-1">
                            {db.description}
                          </p>
                        )}
                        <p className="text-xs text-muted-foreground mt-2">
                          Добавлено: {new Date(db.created_at).toLocaleDateString('ru-RU')}
                        </p>
                      </div>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteDatabase(db.id)}
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

