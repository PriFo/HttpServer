'use client'

import { useState } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { 
  ArrowLeft,
  Target 
} from "lucide-react"
import Link from 'next/link'

export default function NewProjectPage() {
  const params = useParams()
  const router = useRouter()
  const clientId = params.clientId
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    project_type: 'nomenclature',
    description: '',
    source_system: '',
    target_quality_score: 0.9
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      })

      if (response.ok) {
        const newProject = await response.json()
        router.push(`/clients/${clientId}/projects/${newProject.id}`)
      } else {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create project')
      }
    } catch (error) {
      console.error('Error creating project:', error)
      alert(error instanceof Error ? error.message : 'Ошибка при создании проекта')
    } finally {
      setIsLoading(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const value = e.target.type === 'number' ? parseFloat(e.target.value) : e.target.value
    setFormData(prev => ({
      ...prev,
      [e.target.name]: value
    }))
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" asChild>
          <Link href={`/clients/${clientId}/projects`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold">Новый проект</h1>
          <p className="text-muted-foreground">
            Создание нового проекта нормализации
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Target className="h-5 w-5" />
            Информация о проекте
          </CardTitle>
          <CardDescription>
            Заполните основную информацию о проекте
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="name">Название проекта *</Label>
                <Input
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="Номенклатура 2024"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="project_type">Тип проекта *</Label>
                <select
                  id="project_type"
                  name="project_type"
                  value={formData.project_type}
                  onChange={handleChange}
                  className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  required
                >
                  <option value="nomenclature">Номенклатура</option>
                  <option value="counterparties">Контрагенты</option>
                  <option value="mixed">Смешанный</option>
                </select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="source_system">Исходная система</Label>
                <Input
                  id="source_system"
                  name="source_system"
                  value={formData.source_system}
                  onChange={handleChange}
                  placeholder="1C:Торговля 11.4"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="target_quality_score">Целевое качество (0.0-1.0)</Label>
                <Input
                  id="target_quality_score"
                  name="target_quality_score"
                  type="number"
                  min="0"
                  max="1"
                  step="0.1"
                  value={formData.target_quality_score}
                  onChange={handleChange}
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Описание</Label>
              <textarea
                id="description"
                name="description"
                value={formData.description}
                onChange={handleChange}
                placeholder="Описание проекта и целей нормализации..."
                rows={3}
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            <div className="flex gap-4 pt-4">
              <Button type="button" variant="outline" asChild>
                <Link href={`/clients/${clientId}/projects`}>
                  Отмена
                </Link>
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Создание...' : 'Создать проект'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

