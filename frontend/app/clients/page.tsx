'use client'

import { useState, useEffect, useMemo, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { formatDate } from '@/lib/locale'
import { getCountryByCode, getSortedCountries } from '@/lib/countries'
import { exportClientsToCSV, exportClientsToJSON, exportClientsToExcel, exportClientsToPDF, exportClientsToWord } from '@/lib/export-clients'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { 
  Plus, 
  Building2,
  Target,
  Calendar,
  Globe,
  Download,
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { ClientsPageSkeleton } from "@/components/common/clients-skeleton"
import { Skeleton } from "@/components/ui/skeleton"
import { FilterBar, type FilterConfig } from "@/components/common/filter-bar"
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { motion } from "framer-motion"
import { FadeIn } from "@/components/animations/fade-in"
import { StaggerContainer, StaggerItem } from "@/components/animations/stagger-container"
import { useApiClient } from '@/hooks/useApiClient'
import { checkBackendHealth } from '@/lib/api-config'
import type { Client } from '@/types'

export default function ClientsPage() {
  const router = useRouter()
  const { get } = useApiClient()
  const [clients, setClients] = useState<Client[]>([])
  const [searchTerm, setSearchTerm] = useState('')
  const [debouncedSearchTerm, setDebouncedSearchTerm] = useState('')
  const [selectedCountry, setSelectedCountry] = useState<string>('')
  const [selectedIndustry, setSelectedIndustry] = useState<string>('')
  const [selectedStatus, setSelectedStatus] = useState<string>('')
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [backendUnavailable, setBackendUnavailable] = useState(false)
  const [backendCheckDone, setBackendCheckDone] = useState(false)
  
  const countries = getSortedCountries()

  useEffect(() => {
    fetchClients()
  }, [])

  // Debounce поиска
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchTerm(searchTerm)
    }, 300)

    return () => clearTimeout(timer)
  }, [searchTerm])

  const fetchClients = async () => {
    setIsLoading(true)
    setError(null)
    setBackendUnavailable(false)
    setBackendCheckDone(false)
    
    try {
      // Сначала проверяем доступность бэкенда
      const isBackendHealthy = await checkBackendHealth()
      setBackendCheckDone(true)
      
      if (!isBackendHealthy) {
        setBackendUnavailable(true)
        setClients([])
        setIsLoading(false)
        return
      }
      
      const data = await get<Client[]>('/api/clients', { skipErrorHandler: true })
      setClients(data || [])
      setBackendUnavailable(false)
    } catch (error) {
      setBackendCheckDone(true)
      const errorMessage = error instanceof Error ? error.message : 'Не удалось загрузить клиентов'
      setError(errorMessage)
      setClients([])
      
      // Определяем, является ли это ошибкой подключения к бэкенду
      if (errorMessage.includes('backend') || 
          errorMessage.includes('9999') || 
          errorMessage.includes('Failed to fetch') ||
          errorMessage.includes('ECONNREFUSED') ||
          errorMessage.includes('NetworkError')) {
        setBackendUnavailable(true)
      }
    } finally {
      setIsLoading(false)
    }
  }

  const filteredClients = useMemo(() => {
    if (!clients) return []

    let filtered = clients

    // Фильтр по поисковому запросу (расширенный поиск по всем полям)
    const searchLower = debouncedSearchTerm.toLowerCase()
    if (searchLower) {
      filtered = filtered.filter(client => {
        // Основные поля
        if (client.name.toLowerCase().includes(searchLower) ||
            (client.legal_name && client.legal_name.toLowerCase().includes(searchLower)) ||
            (client.tax_id && client.tax_id.toLowerCase().includes(searchLower)) ||
            (client.description && client.description.toLowerCase().includes(searchLower))) {
          return true
        }
        
        // Бизнес-информация
        if ((client.industry && client.industry.toLowerCase().includes(searchLower)) ||
            (client.company_size && client.company_size.toLowerCase().includes(searchLower)) ||
            (client.legal_form && client.legal_form.toLowerCase().includes(searchLower))) {
          return true
        }
        
        // Контакты
        if ((client.contact_person && client.contact_person.toLowerCase().includes(searchLower)) ||
            (client.contact_email && client.contact_email.toLowerCase().includes(searchLower)) ||
            (client.contact_phone && client.contact_phone.toLowerCase().includes(searchLower)) ||
            (client.website && client.website.toLowerCase().includes(searchLower))) {
          return true
        }
        
        // Юридические данные
        if ((client.ogrn && client.ogrn.toLowerCase().includes(searchLower)) ||
            (client.kpp && client.kpp.toLowerCase().includes(searchLower)) ||
            (client.legal_address && client.legal_address.toLowerCase().includes(searchLower)) ||
            (client.bank_name && client.bank_name.toLowerCase().includes(searchLower))) {
          return true
        }
        
        // Договорные данные
        if ((client.contract_number && client.contract_number.toLowerCase().includes(searchLower))) {
          return true
        }
        
        return false
      })
    }

    // Фильтр по стране
    if (selectedCountry) {
      filtered = filtered.filter(client => client.country === selectedCountry)
    }

    // Фильтр по отрасли
    if (selectedIndustry) {
      filtered = filtered.filter(client => client.industry === selectedIndustry)
    }

    // Фильтр по статусу
    if (selectedStatus) {
      filtered = filtered.filter(client => client.status === selectedStatus)
    }

    return filtered
  }, [clients, debouncedSearchTerm, selectedCountry, selectedIndustry, selectedStatus])

  // Получаем уникальные значения для фильтров
  const uniqueIndustries = useMemo(() => {
    if (!clients) return []
    const industries = clients
      .map(c => c.industry)
      .filter((ind): ind is string => Boolean(ind && ind.trim()))
    return Array.from(new Set(industries)).sort()
  }, [clients])

  const getStatusVariant = useCallback((status: string) => {
    switch (status) {
      case 'active': return 'default'
      case 'inactive': return 'secondary'
      case 'suspended': return 'destructive'
      default: return 'outline'
    }
  }, [])

  const getStatusLabel = useCallback((status: string) => {
    switch (status) {
      case 'active': return 'Активен'
      case 'inactive': return 'Неактивен'
      case 'suspended': return 'Приостановлен'
      default: return status
    }
  }, [])

  const breadcrumbItems = [
    { label: 'Клиенты', href: '/clients', icon: Building2 },
  ]

  if (isLoading) {
    return (
      <div className="container-wide mx-auto px-4 py-6 sm:py-8">
        <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
        <div className="mb-4">
          <Breadcrumb items={breadcrumbItems} />
        </div>
        <ClientsPageSkeleton />
      </div>
    )
  }

  return (
    <div className="container-wide mx-auto px-4 py-6 sm:py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      {/* Header */}
      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
          className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"
        >
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl sm:text-3xl font-bold flex items-center gap-2 sm:gap-3">
            <div className="p-2 rounded-lg bg-primary/10 flex-shrink-0">
              <Building2 className="h-5 w-5 sm:h-6 sm:w-6 text-primary" />
            </div>
            <span>Клиенты</span>
          </h1>
          <p className="text-sm sm:text-base text-muted-foreground mt-1 sm:mt-2">
            Управление юридическими лицами и проектами нормализации
          </p>
        </div>
          <motion.div 
            className="flex gap-2 flex-wrap"
            initial={{ opacity: 0, scale: 0.9 }}
            animate={{ opacity: 1, scale: 1 }}
            transition={{ duration: 0.3, delay: 0.2 }}
          >
            {filteredClients.length > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="outline" size="sm">
                    <Download className="mr-2 h-4 w-4" />
                    <span className="hidden sm:inline">Экспорт</span>
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem onClick={() => {
                    const exportData = filteredClients.map(client => ({
                      ...client,
                      legal_name: client.legal_name || '',
                      description: client.description || '',
                      contact_email: '',
                      contact_phone: '',
                      tax_id: client.tax_id || '',
                      country: client.country || '',
                      project_count: client.project_count || 0,
                      benchmark_count: client.benchmark_count || 0,
                      last_activity: client.last_activity || '',
                    }))
                    exportClientsToCSV(exportData)
                  }}>
                    Экспорт в CSV
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => {
                    const exportData = filteredClients.map(client => ({
                      ...client,
                      legal_name: client.legal_name || '',
                      description: client.description || '',
                      contact_email: '',
                      contact_phone: '',
                      tax_id: client.tax_id || '',
                      country: client.country || '',
                      project_count: client.project_count || 0,
                      benchmark_count: client.benchmark_count || 0,
                      last_activity: client.last_activity || '',
                    }))
                    exportClientsToJSON(exportData)
                  }}>
                    Экспорт в JSON
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => {
                    const exportData = filteredClients.map(client => ({
                      ...client,
                      legal_name: client.legal_name || '',
                      description: client.description || '',
                      contact_email: '',
                      contact_phone: '',
                      tax_id: client.tax_id || '',
                      country: client.country || '',
                      project_count: client.project_count || 0,
                      benchmark_count: client.benchmark_count || 0,
                      last_activity: client.last_activity || '',
                    }))
                    exportClientsToExcel(exportData)
                  }}>
                    Экспорт в Excel
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={() => {
                    const exportData = filteredClients.map(client => ({
                      ...client,
                      legal_name: client.legal_name || '',
                      description: client.description || '',
                      contact_email: '',
                      contact_phone: '',
                      tax_id: client.tax_id || '',
                      country: client.country || '',
                      project_count: client.project_count || 0,
                      benchmark_count: client.benchmark_count || 0,
                      last_activity: client.last_activity || '',
                    }))
                    exportClientsToPDF(exportData)
                  }}>
                    Экспорт в PDF
                  </DropdownMenuItem>
                  <DropdownMenuItem onClick={async () => {
                    const exportData = filteredClients.map(client => ({
                      ...client,
                      legal_name: client.legal_name || '',
                      description: client.description || '',
                      contact_email: '',
                      contact_phone: '',
                      tax_id: client.tax_id || '',
                      country: client.country || '',
                      project_count: client.project_count || 0,
                      benchmark_count: client.benchmark_count || 0,
                      last_activity: client.last_activity || '',
                    }))
                    await exportClientsToWord(exportData)
                  }}>
                    Экспорт в Word
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
            <Button asChild size="sm">
              <Link href="/clients/new">
                <Plus className="mr-2 h-4 w-4" />
                <span className="hidden sm:inline">Добавить клиента</span>
                <span className="sm:hidden">Добавить</span>
              </Link>
            </Button>
          </motion.div>
        </motion.div>
      </FadeIn>

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
                placeholder: 'Поиск по названию, ИНН, ОГРН, контактам, отрасли...',
              },
              {
                type: 'select',
                key: 'country',
                label: 'Страна',
                placeholder: 'Все страны',
                options: [
                  { value: '', label: 'Все страны' },
                  ...countries.map(c => ({ value: c.code, label: c.name }))
                ],
              },
              {
                type: 'select',
                key: 'industry',
                label: 'Отрасль',
                placeholder: 'Все отрасли',
                options: [
                  { value: '', label: 'Все отрасли' },
                  ...uniqueIndustries.map(ind => ({ value: ind, label: ind }))
                ],
              },
              {
                type: 'select',
                key: 'status',
                label: 'Статус',
                placeholder: 'Все статусы',
                options: [
                  { value: '', label: 'Все статусы' },
                  { value: 'active', label: 'Активен' },
                  { value: 'inactive', label: 'Неактивен' },
                  { value: 'suspended', label: 'Приостановлен' },
                ],
              },
            ]}
            values={{ search: searchTerm, country: selectedCountry, industry: selectedIndustry, status: selectedStatus }}
            onChange={(values) => {
              setSearchTerm(values.search || '')
              setSelectedCountry(values.country || '')
              setSelectedIndustry(values.industry || '')
              setSelectedStatus(values.status || '')
            }}
            onReset={() => {
              setSearchTerm('')
              setSelectedCountry('')
              setSelectedIndustry('')
              setSelectedStatus('')
            }}
          />
        </CardContent>
      </Card>

      {/* Error Alert */}
      {error && (
        <Alert variant="destructive">
          <AlertDescription>
            {error}
            <Button
              variant="outline"
              size="sm"
              className="ml-4"
              onClick={fetchClients}
            >
              Повторить
            </Button>
          </AlertDescription>
        </Alert>
      )}

      {/* Список клиентов */}
      {isLoading ? (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {[...Array(6)].map((_, i) => (
            <Card key={i}>
              <CardHeader>
                <Skeleton className="h-6 w-3/4 mb-2" />
                <Skeleton className="h-4 w-1/2" />
              </CardHeader>
              <CardContent className="space-y-4">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-2/3" />
                <div className="flex gap-2 pt-2">
                  <Skeleton className="h-9 flex-1" />
                  <Skeleton className="h-9 flex-1" />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {filteredClients.map((client) => (
            <Card 
              key={client.id} 
              className="hover:shadow-lg transition-shadow cursor-pointer"
              onClick={() => router.push(`/clients/${client.id}`)}
            >
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
              <CardContent className="space-y-3">
                {client.industry && (
                  <div className="text-sm text-muted-foreground">
                    <span className="font-medium">Отрасль:</span> {client.industry}
                  </div>
                )}
                {client.country && (
                  <div className="flex items-center gap-1 text-sm text-muted-foreground">
                    <Globe className="h-3 w-3" />
                    <span>{getCountryByCode(client.country)?.name || client.country}</span>
                  </div>
                )}
                {client.contact_person && (
                  <div className="text-sm text-muted-foreground">
                    <span className="font-medium">Контакт:</span> {client.contact_person}
                    {client.contact_position && ` (${client.contact_position})`}
                  </div>
                )}
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
                  <span>{formatDate(client.last_activity)}</span>
                </div>
                
                <div className="flex gap-2 pt-2" onClick={(e) => e.stopPropagation()}>
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

      {filteredClients.length === 0 && !isLoading && backendCheckDone && (
        <Card>
          <CardContent className="pt-6">
            {backendUnavailable && !searchTerm ? (
              <EmptyState
                icon={Building2}
                title="Backend сервер недоступен"
                description="Не удалось подключиться к backend серверу. Убедитесь, что сервер запущен на порту 9999. Используйте скрипт start-backend-exe.bat для запуска."
                action={{
                  label: 'Повторить попытку',
                  onClick: fetchClients,
                }}
              />
            ) : (
              <EmptyState
                icon={Building2}
                title="Клиенты не найдены"
                description={
                  searchTerm || selectedCountry
                    ? 'Попробуйте изменить условия поиска или сбросить фильтры' 
                    : 'Добавьте первого клиента, чтобы начать работу'
                }
                action={
                  searchTerm || selectedCountry
                    ? {
                        label: 'Сбросить фильтры',
                        onClick: () => {
                          setSearchTerm('')
                          setSelectedCountry('')
                        }
                      }
                    : {
                        label: 'Добавить клиента',
                        onClick: () => router.push('/clients/new'),
                      }
                }
              />
            )}
          </CardContent>
        </Card>
      )}
    </div>
  )
}

