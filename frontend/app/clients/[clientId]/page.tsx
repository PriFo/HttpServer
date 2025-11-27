'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { 
  Building2, 
  Plus,
  Target,
  Globe,
  ArrowLeft,
  Package,
  Users,
  Database,
  BarChart3,
  Phone,
  Mail,
  MapPin,
  FileText,
  Calendar as CalendarIcon,
  ChevronDown,
  ChevronUp
} from "lucide-react"
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { ErrorState } from "@/components/common/error-state"
import { StatCard } from "@/components/common/stat-card"
import { getCountryByCode } from '@/lib/countries'
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { motion } from "framer-motion"
import { FadeIn } from "@/components/animations/fade-in"
import { useRouter } from "next/navigation"
import { formatDate } from "@/lib/locale"
import type { Client } from '@/types'
import { StatisticsTab } from "./components/statistics-tab"
import { DatabasesTab } from "./components/databases-tab"
import { NomenclatureTab } from "./components/nomenclature-tab"
import { CounterpartiesTab } from "./components/counterparties-tab"
import { DocumentsSection } from "./components/documents-section"
import { EditClientDialog } from "./components/edit-client-dialog"
import { NormalizationDashboard } from "@/components/mdm/normalization-dashboard"

interface ClientDetail {
  client: {
    id: number
    name: string
    legal_name: string
    description: string
    contact_email: string
    contact_phone: string
    tax_id: string
    country?: string
    status: string
    created_at: string
    // Бизнес-информация
    industry?: string
    company_size?: string
    legal_form?: string
    // Расширенные контакты
    contact_person?: string
    contact_position?: string
    alternate_phone?: string
    website?: string
    // Юридические данные
    ogrn?: string
    kpp?: string
    legal_address?: string
    postal_address?: string
    bank_name?: string
    bank_account?: string
    correspondent_account?: string
    bik?: string
    // Договорные данные
    contract_number?: string
    contract_date?: string
    contract_terms?: string
    contract_expires_at?: string
  }
  projects: Array<{
    id: number
    name: string
    project_type: string
    status: string
    created_at: string
  }>
  statistics: {
    total_projects: number
    total_benchmarks: number
    active_sessions: number
    avg_quality_score: number
  }
  documents?: Array<{
    id: number
    client_id: number
    file_name: string
    file_path: string
    file_type: string
    file_size: number
    category: string
    description?: string
    uploaded_by?: string
    uploaded_at: string
  }>
}

export default function ClientDetailPage() {
  const params = useParams()
  const router = useRouter()
  const clientId = params.clientId
  const [client, setClient] = useState<ClientDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<string>("overview")
  const [selectedProjectId, setSelectedProjectId] = useState<number | null>(null)
  const [expandedSections, setExpandedSections] = useState({
    business: false,
    legal: false,
    contract: false,
  })
  const [showEditDialog, setShowEditDialog] = useState(false)

  const toggleSection = (section: keyof typeof expandedSections) => {
    setExpandedSections(prev => ({
      ...prev,
      [section]: !prev[section],
    }))
  }

  useEffect(() => {
    if (clientId) {
      fetchClientDetail(clientId as string)
    }
  }, [clientId])

  const fetchClientDetail = async (id: string) => {
    setIsLoading(true)
    setError(null)
    try {
      const response = await fetch(`/api/clients/${id}`)
      if (!response.ok) {
        throw new Error('Failed to fetch client details')
      }
      const data = await response.json()
      setClient(data)
    } catch (error) {
      console.error('Failed to fetch client details:', error)
      setError(error instanceof Error ? error.message : 'Не удалось загрузить данные клиента')
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

  if (isLoading) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <LoadingState message="Загрузка данных клиента..." size="lg" fullScreen />
      </div>
    )
  }

  if (error || !client) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <ErrorState
          title="Ошибка загрузки"
          message={error || 'Клиент не найден'}
          action={{
            label: 'Повторить',
            onClick: () => clientId && fetchClientDetail(clientId as string),
          }}
          variant="destructive"
        />
      </div>
    )
  }

  const breadcrumbItems = [
    { label: 'Клиенты', href: '/clients', icon: Building2 },
    { label: client.client.name, href: `/clients/${clientId}`, icon: Building2 },
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
            onClick={() => router.push('/clients')}
            aria-label="Назад к списку клиентов"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Building2 className="h-8 w-8 text-primary" />
              {client.client.name}
            </h1>
            <p className="text-muted-foreground mt-1">{client.client.description}</p>
            {client.client.legal_name && (
              <p className="text-sm text-muted-foreground mt-1">{client.client.legal_name}</p>
            )}
          </div>
        <div className="flex gap-2">
          <Button variant="outline" onClick={() => setShowEditDialog(true)}>
              Редактировать
          </Button>
          <Button asChild>
            <Link href={`/clients/${clientId}/projects/new`}>
              <Plus className="mr-2 h-4 w-4" />
              Новый проект
            </Link>
          </Button>
        </div>
      </motion.div>
      </FadeIn>

      {/* Табы для просмотра данных */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="grid w-full grid-cols-6">
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="normalization">Нормализация</TabsTrigger>
          <TabsTrigger value="nomenclature">Номенклатура</TabsTrigger>
          <TabsTrigger value="counterparties">Контрагенты</TabsTrigger>
          <TabsTrigger value="databases">Базы данных</TabsTrigger>
          <TabsTrigger value="statistics">Статистика</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {/* Статистика */}
          <div className="grid grid-cols-1 md:grid-cols-4 gap-6">
            <StatCard
              title="Проекты"
              value={client.statistics.total_projects}
              description="активных проектов"
              icon={Target}
              variant="primary"
            />
            <StatCard
              title="Эталоны"
              value={client.statistics.total_benchmarks}
              description="создано записей"
              icon={Building2}
              variant="success"
            />
            <StatCard
              title="Качество"
              value={`${Math.round(client.statistics.avg_quality_score * 100)}%`}
              description="среднее качество"
              variant={client.statistics.avg_quality_score >= 0.9 ? 'success' : client.statistics.avg_quality_score >= 0.7 ? 'warning' : 'danger'}
              progress={client.statistics.avg_quality_score * 100}
            />
            <StatCard
              title="Активность"
              value={client.statistics.active_sessions}
              description="активных сессий"
              variant="default"
            />
          </div>

          {/* Проекты клиента */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Target className="h-5 w-5" />
                Активные проекты
              </CardTitle>
              <CardDescription>
                Проекты нормализации для этого клиента
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-3">
                {client.projects.slice(0, 5).map((project) => (
                  <div key={project.id} className="flex items-center justify-between p-3 border rounded-lg">
                    <div>
                      <div className="font-medium">{project.name}</div>
                      <div className="text-sm text-muted-foreground">
                        {getProjectTypeLabel(project.project_type)} • {formatDate(project.created_at)}
                      </div>
                    </div>
                    <div className="flex items-center gap-2">
                      <Badge variant={project.status === 'active' ? 'default' : 'secondary'}>
                        {project.status}
                      </Badge>
                      <Button asChild size="sm">
                        <Link href={`/clients/${clientId}/projects/${project.id}`}>
                          Открыть
                        </Link>
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
              {client.projects.length > 5 && (
                <Button variant="outline" className="w-full mt-4" asChild>
                  <Link href={`/clients/${clientId}/projects`}>
                    Показать все проекты ({client.projects.length})
                  </Link>
                </Button>
              )}
            </CardContent>
          </Card>

          {/* Информация о клиенте */}
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* Основная информация */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Building2 className="h-5 w-5" />
                  Основная информация
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">Статус:</span>
                    <Badge variant={client.client.status === 'active' ? 'default' : 'secondary'}>
                      {client.client.status}
                    </Badge>
                  </div>
                  {client.client.tax_id && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">ИНН/БИН:</span>
                      <span className="text-sm font-medium">{client.client.tax_id}</span>
                    </div>
                  )}
                  {client.client.country && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground flex items-center gap-1">
                        <Globe className="h-3 w-3" />
                        Страна:
                      </span>
                      <span className="text-sm font-medium">
                        {getCountryByCode(client.client.country)?.name || client.client.country}
                      </span>
                    </div>
                  )}
                  {client.client.industry && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Отрасль:</span>
                      <span className="text-sm font-medium">{client.client.industry}</span>
                    </div>
                  )}
                  {client.client.company_size && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Размер компании:</span>
                      <span className="text-sm font-medium">{client.client.company_size}</span>
                    </div>
                  )}
                  {client.client.legal_form && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Правовая форма:</span>
                      <span className="text-sm font-medium">{client.client.legal_form}</span>
                    </div>
                  )}
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground">Создан:</span>
                    <span className="text-sm font-medium">
                      {formatDate(client.client.created_at)}
                    </span>
                  </div>
                </div>
              </CardContent>
            </Card>

            {/* Контактная информация */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Phone className="h-5 w-5" />
                  Контактная информация
                </CardTitle>
              </CardHeader>
              <CardContent>
                <div className="space-y-3">
                  {client.client.contact_person && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Контактное лицо:</span>
                      <span className="text-sm font-medium">{client.client.contact_person}</span>
                    </div>
                  )}
                  {client.client.contact_position && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Должность:</span>
                      <span className="text-sm font-medium">{client.client.contact_position}</span>
                    </div>
                  )}
                  {client.client.contact_email && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground flex items-center gap-1">
                        <Mail className="h-3 w-3" />
                        Email:
                      </span>
                      <span className="text-sm font-medium">{client.client.contact_email}</span>
                    </div>
                  )}
                  {client.client.contact_phone && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Телефон:</span>
                      <span className="text-sm font-medium">{client.client.contact_phone}</span>
                    </div>
                  )}
                  {client.client.alternate_phone && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Доп. телефон:</span>
                      <span className="text-sm font-medium">{client.client.alternate_phone}</span>
                    </div>
                  )}
                  {client.client.website && (
                    <div className="flex justify-between">
                      <span className="text-sm text-muted-foreground">Веб-сайт:</span>
                      <a href={client.client.website} target="_blank" rel="noopener noreferrer" className="text-sm font-medium text-primary hover:underline">
                        {client.client.website}
                      </a>
                    </div>
                  )}
                </div>
              </CardContent>
            </Card>
          </div>

          {/* Дополнительная информация (скрываемая) */}
          {(client.client.ogrn || client.client.kpp || client.client.legal_address || 
            client.client.postal_address || client.client.bank_name) && (
            <Card>
              <CardHeader className="cursor-pointer" onClick={() => toggleSection('legal')}>
                <CardTitle className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <MapPin className="h-5 w-5" />
                    Юридическая информация
                  </div>
                  {expandedSections.legal ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                </CardTitle>
              </CardHeader>
              {expandedSections.legal && (
                <CardContent>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {client.client.ogrn && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">ОГРН:</span>
                        <span className="text-sm font-medium">{client.client.ogrn}</span>
                      </div>
                    )}
                    {client.client.kpp && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">КПП:</span>
                        <span className="text-sm font-medium">{client.client.kpp}</span>
                      </div>
                    )}
                    {client.client.legal_address && (
                      <div className="col-span-2">
                        <span className="text-sm text-muted-foreground">Юридический адрес:</span>
                        <p className="text-sm font-medium mt-1">{client.client.legal_address}</p>
                      </div>
                    )}
                    {client.client.postal_address && (
                      <div className="col-span-2">
                        <span className="text-sm text-muted-foreground">Почтовый адрес:</span>
                        <p className="text-sm font-medium mt-1">{client.client.postal_address}</p>
                      </div>
                    )}
                    {client.client.bank_name && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">Банк:</span>
                        <span className="text-sm font-medium">{client.client.bank_name}</span>
                      </div>
                    )}
                    {client.client.bank_account && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">Расч. счет:</span>
                        <span className="text-sm font-medium">{client.client.bank_account}</span>
                      </div>
                    )}
                    {client.client.correspondent_account && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">Корр. счет:</span>
                        <span className="text-sm font-medium">{client.client.correspondent_account}</span>
                      </div>
                    )}
                    {client.client.bik && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">БИК:</span>
                        <span className="text-sm font-medium">{client.client.bik}</span>
                      </div>
                    )}
                  </div>
                </CardContent>
              )}
            </Card>
          )}

          {/* Договорная информация (скрываемая) */}
          {(client.client.contract_number || client.client.contract_date || client.client.contract_terms) && (
            <Card>
              <CardHeader className="cursor-pointer" onClick={() => toggleSection('contract')}>
                <CardTitle className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <FileText className="h-5 w-5" />
                    Договорная информация
                  </div>
                  {expandedSections.contract ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                </CardTitle>
              </CardHeader>
              {expandedSections.contract && (
                <CardContent>
                  <div className="space-y-3">
                    {client.client.contract_number && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground">Номер договора:</span>
                        <span className="text-sm font-medium">{client.client.contract_number}</span>
                      </div>
                    )}
                    {client.client.contract_date && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground flex items-center gap-1">
                          <CalendarIcon className="h-3 w-3" />
                          Дата договора:
                        </span>
                        <span className="text-sm font-medium">{formatDate(client.client.contract_date)}</span>
                      </div>
                    )}
                    {client.client.contract_expires_at && (
                      <div className="flex justify-between">
                        <span className="text-sm text-muted-foreground flex items-center gap-1">
                          <CalendarIcon className="h-3 w-3" />
                          Действует до:
                        </span>
                        <span className="text-sm font-medium">{formatDate(client.client.contract_expires_at)}</span>
                      </div>
                    )}
                    {client.client.contract_terms && (
                      <div>
                        <span className="text-sm text-muted-foreground">Условия договора:</span>
                        <p className="text-sm mt-1">{client.client.contract_terms}</p>
                      </div>
                    )}
                  </div>
                </CardContent>
              )}
            </Card>
          )}

          {/* Секция документов */}
          <DocumentsSection
            clientId={clientId as string}
            documents={client.documents || []}
            onDocumentsChange={() => fetchClientDetail(clientId as string)}
          />
        </TabsContent>

        {/* Диалог редактирования */}
        {client && (
          <EditClientDialog
            open={showEditDialog}
            onOpenChange={setShowEditDialog}
            client={client.client as Client}
            onSuccess={() => fetchClientDetail(clientId as string)}
          />
        )}

        <TabsContent value="normalization" className="space-y-4">
          {selectedProjectId ? (
            <NormalizationDashboard
              clientId={clientId as string}
              projectId={selectedProjectId.toString()}
              projectType={client.projects.find(p => p.id === selectedProjectId)?.project_type || null}
            />
          ) : (
            <Card>
              <CardContent className="py-8">
                <div className="text-center text-muted-foreground">
                  <p className="mb-4">Выберите проект для просмотра нормализации</p>
                  <div className="space-y-2">
                    {client.projects.map((project) => (
                      <Button
                        key={project.id}
                        variant="outline"
                        onClick={() => setSelectedProjectId(project.id)}
                        className="w-full"
                      >
                        {project.name}
                      </Button>
                    ))}
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
        </TabsContent>

        <TabsContent value="nomenclature">
          <NomenclatureTab clientId={clientId as string} projects={client.projects} />
        </TabsContent>

        <TabsContent value="counterparties">
          <CounterpartiesTab clientId={clientId as string} projects={client.projects} />
        </TabsContent>

        <TabsContent value="databases">
          <DatabasesTab clientId={clientId as string} projects={client.projects} />
        </TabsContent>

        <TabsContent value="statistics">
          <StatisticsTab clientId={clientId as string} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

