'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { 
  ArrowLeft,
  Building2 
} from "lucide-react"
import Link from 'next/link'

export default function NewClientPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [formData, setFormData] = useState({
    name: '',
    legal_name: '',
    description: '',
    contact_email: '',
    contact_phone: '',
    tax_id: ''
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsLoading(true)
    try {
      const response = await fetch('/api/clients', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      })

      if (response.ok) {
        const newClient = await response.json()
        router.push(`/clients/${newClient.id}`)
      } else {
        const errorData = await response.json()
        throw new Error(errorData.error || 'Failed to create client')
      }
    } catch (error) {
      console.error('Error creating client:', error)
      alert(error instanceof Error ? error.message : 'Ошибка при создании клиента')
    } finally {
      setIsLoading(false)
    }
  }

  const handleChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>) => {
    setFormData(prev => ({
      ...prev,
      [e.target.name]: e.target.value
    }))
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" asChild>
          <Link href="/clients">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div>
          <h1 className="text-3xl font-bold">Новый клиент</h1>
          <p className="text-muted-foreground">
            Добавление нового юридического лица или проекта
          </p>
        </div>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Building2 className="h-5 w-5" />
            Информация о клиенте
          </CardTitle>
          <CardDescription>
            Заполните основную информацию о клиенте
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="space-y-2">
                <Label htmlFor="name">Название *</Label>
                <Input
                  id="name"
                  name="name"
                  value={formData.name}
                  onChange={handleChange}
                  placeholder="ООО Ромашка"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="legal_name">Юридическое название</Label>
                <Input
                  id="legal_name"
                  name="legal_name"
                  value={formData.legal_name}
                  onChange={handleChange}
                  placeholder="Общество с ограниченной ответственностью 'Ромашка'"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="tax_id">ИНН</Label>
                <Input
                  id="tax_id"
                  name="tax_id"
                  value={formData.tax_id}
                  onChange={handleChange}
                  placeholder="1234567890"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="contact_email">Контактный email</Label>
                <Input
                  id="contact_email"
                  name="contact_email"
                  type="email"
                  value={formData.contact_email}
                  onChange={handleChange}
                  placeholder="contact@example.com"
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
                placeholder="Описание клиента и целей нормализации..."
                rows={3}
                className="flex min-h-[80px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
              />
            </div>
            <div className="flex gap-4 pt-4">
              <Button type="button" variant="outline" asChild>
                <Link href="/clients">
                  Отмена
                </Link>
              </Button>
              <Button type="submit" disabled={isLoading}>
                {isLoading ? 'Создание...' : 'Создать клиента'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}

