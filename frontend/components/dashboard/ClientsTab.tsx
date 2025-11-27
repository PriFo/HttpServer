'use client'

import { useEffect, useState, useMemo } from 'react'
import { motion } from 'framer-motion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import { Users, Search, Plus, Building2, TrendingUp } from 'lucide-react'
import { apiClientJson } from '@/lib/api-client'
import { Skeleton } from '@/components/ui/skeleton'
import { EmptyState } from './EmptyState'
import Link from 'next/link'
import { useError } from '@/contexts/ErrorContext'

interface Client {
  id: number
  name: string
  legal_name?: string
  tax_id?: string
  country?: string
  created_at?: string
}

export function ClientsTab() {
  const { handleError } = useError()
  const [clients, setClients] = useState<Client[]>([])
  const [searchQuery, setSearchQuery] = useState('')
  const [isLoading, setIsLoading] = useState(true)

  useEffect(() => {
    loadClients()
  }, [])

  const loadClients = async () => {
    try {
      setIsLoading(true)
      const data = await apiClientJson<Client[]>('/api/clients', { skipErrorHandler: true })
      setClients(data || [])
    } catch (error) {
      handleError(error, 'Не удалось загрузить список клиентов')
      setClients([])
    } finally {
      setIsLoading(false)
    }
  }

  const filteredClients = useMemo(() => {
    if (!searchQuery) return clients
    const query = searchQuery.toLowerCase()
    return clients.filter(
      (client) =>
        client.name.toLowerCase().includes(query) ||
        client.legal_name?.toLowerCase().includes(query) ||
        client.tax_id?.toLowerCase().includes(query)
    )
  }, [clients, searchQuery])

  if (isLoading) {
    return (
      <div className="container mx-auto p-6 space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {[1, 2, 3, 4, 5, 6].map((i) => (
            <Skeleton key={i} className="h-32" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold flex items-center gap-2">
            <Users className="h-6 w-6" />
            Клиенты
          </h2>
          <p className="text-muted-foreground mt-1">
            Управление клиентами и проектами
          </p>
        </div>
        <Button asChild>
          <Link href="/clients/new">
            <Plus className="h-4 w-4 mr-2" />
            Новый клиент
          </Link>
        </Button>
      </div>

      {/* Search */}
      <div className="relative">
        <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
        <Input
          type="search"
          placeholder="Поиск клиентов..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="pl-9"
        />
      </div>

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Всего клиентов</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{clients.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">Найдено</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{filteredClients.length}</div>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <TrendingUp className="h-4 w-4" />
              Активность
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">—</div>
          </CardContent>
        </Card>
      </div>

      {/* Clients List */}
      {filteredClients.length === 0 ? (
        <EmptyState
          icon={Users}
          title={searchQuery ? 'Клиенты не найдены' : 'Нет клиентов'}
          description={searchQuery ? 'Попробуйте изменить поисковый запрос' : 'Создайте первого клиента для начала работы'}
          action={!searchQuery ? {
            label: 'Создать клиента',
            onClick: () => window.location.href = '/clients/new'
          } : undefined}
        />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredClients.map((client, index) => (
            <motion.div
              key={client.id}
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: index * 0.05 }}
              whileHover={{ scale: 1.02 }}
            >
              <Card className="cursor-pointer hover:shadow-lg transition-shadow">
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <CardTitle className="text-lg">{client.name}</CardTitle>
                    <Building2 className="h-5 w-5 text-muted-foreground" />
                  </div>
                  {client.legal_name && (
                    <CardDescription>{client.legal_name}</CardDescription>
                  )}
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {client.tax_id && (
                      <div className="text-sm">
                        <span className="text-muted-foreground">ИНН: </span>
                        {client.tax_id}
                      </div>
                    )}
                    {client.country && (
                      <Badge variant="outline">{client.country}</Badge>
                    )}
                    <Button variant="outline" size="sm" className="w-full mt-2" asChild>
                      <Link href={`/clients/${client.id}`}>
                        Открыть проекты
                      </Link>
                    </Button>
                  </div>
                </CardContent>
              </Card>
            </motion.div>
          ))}
        </div>
      )}
    </div>
  )
}

