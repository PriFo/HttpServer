'use client'

import { useState, useEffect, useMemo } from 'react'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { 
  Plus, 
  Building2,
  Target,
  Calendar,
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { FilterBar, type FilterConfig } from "@/components/common/filter-bar"

interface Client {
  id: number
  name: string
  legal_name: string
  description: string
  status: string
  project_count: number
  benchmark_count: number
  last_activity: string
}

export default function ClientsPage() {
  const [clients, setClients] = useState<Client[]>([])
  const [searchTerm, setSearchTerm] = useState('')
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    fetchClients()
  }, [])

  const fetchClients = async () => {
    setIsLoading(true)
    try {
      const response = await fetch('/api/clients')
      if (!response.ok) throw new Error('Failed to fetch clients')
      const data = await response.json()
      setClients(data || [])
    } catch (error) {
      console.error('Failed to fetch clients:', error)
      setClients([])
    } finally {
      setIsLoading(false)
    }
  }

  const filteredClients = useMemo(() => {
    if (!clients) return []

    return clients.filter(client =>
      client.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      (client.legal_name && client.legal_name.toLowerCase().includes(searchTerm.toLowerCase()))
    )
  }, [clients, searchTerm])

  const getStatusVariant = (status: string) => {
    switch (status) {
      case 'active': return 'default'
      case 'inactive': return 'secondary'
      case 'suspended': return 'destructive'
      default: return 'outline'
    }
  }

  const getStatusLabel = (status: string) => {
    switch (status) {
      case 'active': return 'Активен'
      case 'inactive': return 'Неактивен'
      case 'suspended': return 'Приостановлен'
      default: return status
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold">Клиенты</h1>
          <p className="text-muted-foreground">
            Управление юридическими лицами и проектами нормализации
          </p>
        </div>
        <Button asChild>
          <Link href="/clients/new">
            <Plus className="mr-2 h-4 w-4" />
            Добавить клиента
          </Link>
        </Button>
      </div>

      {/* Поиск */}
      <Card>
        <CardHeader>
          <CardTitle>Поиск и фильтрация</CardTitle>
        </CardHeader>
        <CardContent>
          <FilterBar
            filters={[
              {
                type: 'search',
                key: 'search',
                label: 'Поиск',
                placeholder: 'Поиск клиентов по названию или ИНН...',
              },
            ]}
            values={{ search: searchTerm }}
            onChange={(values) => setSearchTerm(values.search || '')}
            onReset={() => setSearchTerm('')}
          />
        </CardContent>
      </Card>

      {/* Список клиентов */}
      {isLoading ? (
        <LoadingState message="Загрузка клиентов..." size="lg" fullScreen />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredClients.map((client) => (
            <Card key={client.id} className="hover:shadow-lg transition-shadow">
              <CardHeader>
                <CardTitle className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Building2 className="h-5 w-5" />
                    <span className="truncate">{client.name}</span>
                  </div>
                  <Badge variant={getStatusVariant(client.status)}>
                    {getStatusLabel(client.status)}
                  </Badge>
                </CardTitle>
                <CardDescription className="line-clamp-2">
                  {client.description || 'Нет описания'}
                </CardDescription>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="flex justify-between text-sm">
                  <div className="flex items-center gap-1">
                    <Target className="h-4 w-4" />
                    <span>Проектов:</span>
                  </div>
                  <Badge variant="secondary">{client.project_count}</Badge>
                </div>
                
                <div className="flex justify-between text-sm">
                  <div className="flex items-center gap-1">
                    <Calendar className="h-4 w-4" />
                    <span>Эталонов:</span>
                  </div>
                  <Badge variant="outline">{client.benchmark_count}</Badge>
                </div>
                
                <div className="flex justify-between text-sm text-muted-foreground">
                  <span>Последняя активность:</span>
                  <span>{new Date(client.last_activity).toLocaleDateString('ru-RU')}</span>
                </div>
                
                <div className="flex gap-2 pt-2">
                  <Button asChild variant="outline" className="flex-1">
                    <Link href={`/clients/${client.id}`}>
                      Профиль
                    </Link>
                  </Button>
                  <Button asChild className="flex-1">
                    <Link href={`/clients/${client.id}/projects`}>
                      Проекты
                    </Link>
                  </Button>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      {filteredClients.length === 0 && !isLoading && (
        <Card>
          <CardContent className="pt-6">
            <EmptyState
              icon={Building2}
              title="Клиенты не найдены"
              description={
                searchTerm 
                  ? 'Попробуйте изменить условия поиска' 
                  : 'Добавьте первого клиента'
              }
              action={
                searchTerm
                  ? undefined
                  : {
                      label: 'Добавить клиента',
                      onClick: () => {},
                    }
              }
            />
          </CardContent>
        </Card>
      )}
    </div>
  )
}

