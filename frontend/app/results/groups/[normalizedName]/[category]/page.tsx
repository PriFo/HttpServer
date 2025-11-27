'use client'

import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { MagnifyingGlassIcon } from "@radix-ui/react-icons"
import { ArrowLeft, FileQuestion } from "lucide-react"
import { GroupItemsTable } from '@/components/results/group-items-table'
import { ExportGroupButton } from '@/components/results/export-group-button'
import { ConfidenceBadge } from '@/components/results/confidence-badge'
import { ProcessingLevelBadge } from '@/components/results/processing-level-badge'
import { KpvedBadge } from '@/components/results/kpved-badge'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { FilterBar, type FilterConfig } from '@/components/common/filter-bar'
import { StatCard } from '@/components/common/stat-card'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'
import { BarChart3 } from 'lucide-react'
import { Pagination } from '@/components/ui/pagination'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface ItemAttribute {
  id: number
  attribute_type: string
  attribute_name: string
  attribute_value: string
  unit?: string
  original_text?: string
  confidence?: number
}

interface GroupItem {
  id: number
  code: string
  source_name: string
  source_reference: string
  created_at: string
  ai_confidence?: number
  ai_reasoning?: string
  processing_level?: string
  kpved_code?: string
  kpved_name?: string
  kpved_confidence?: number
  attributes?: ItemAttribute[]
}

interface GroupDetails {
  normalized_name: string
  normalized_reference: string
  category: string
  merged_count: number
  items: GroupItem[]
  kpved_code?: string
  kpved_name?: string
  kpved_confidence?: number
  total?: number
  total_pages?: number
  page?: number
  limit?: number
}

export default function GroupDetailPage() {
  const params = useParams()
  const router = useRouter()
  const normalizedName = decodeURIComponent(params.normalizedName as string)
  const category = decodeURIComponent(params.category as string)

  const [groupDetails, setGroupDetails] = useState<GroupDetails | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')
  const [page, setPage] = useState(1)
  const [limit, setLimit] = useState(50)
  const [totalItems, setTotalItems] = useState(0)
  const [totalPages, setTotalPages] = useState(1)

  const fetchGroupData = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({
        normalized_name: normalizedName,
        category: category,
        include_ai: 'true',
        include_attributes: 'true',
        page: page.toString(),
        limit: limit.toString(),
        sort_by: 'created_at',
        sort_order: 'desc',
      })

      if (debouncedSearch) {
        params.set('search', debouncedSearch)
      }

      const response = await fetch(`/api/normalization/group-items?${params}`, {
        cache: 'no-store',
      })

      if (!response.ok) {
        throw new Error(`Failed to fetch: ${response.status}`)
      }

      const data = await response.json()
      setGroupDetails(data)
      setTotalItems(data.total ?? data.items?.length ?? 0)
      setTotalPages(data.total_pages ?? 1)
    } catch (error) {
      console.error('Failed to fetch group data:', error)
      setError(error instanceof Error ? error.message : 'Не удалось загрузить данные группы')
    } finally {
      setLoading(false)
    }
  }, [normalizedName, category, page, limit, debouncedSearch])

  useEffect(() => {
    fetchGroupData()
  }, [fetchGroupData])

  useEffect(() => {
    const handler = setTimeout(() => {
      setPage(1)
      setDebouncedSearch(searchTerm.trim())
    }, 400)
    return () => clearTimeout(handler)
  }, [searchTerm])

  // Оптимизированная фильтрация с useMemo
  const filteredItems = useMemo(() => groupDetails?.items ?? [], [groupDetails])

  // Оптимизированный расчет средней уверенности с useMemo
  const avgConfidence = useMemo(() => {
    if (!groupDetails?.items || groupDetails.items.length === 0) return undefined

    const sum = groupDetails.items.reduce((acc, item) => acc + (item.ai_confidence || 0), 0)
    return sum / groupDetails.items.length
  }, [groupDetails])

  // Определяем основной processing level
  const processingLevel = groupDetails?.items[0]?.processing_level

  if (loading) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <LoadingState message="Загрузка данных группы..." size="lg" fullScreen />
      </div>
    )
  }

  if (error || !groupDetails) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <EmptyState
          icon={FileQuestion}
          title={error || 'Группа не найдена'}
          description={error ? 'Попробуйте обновить страницу' : 'Группа не существует или была удалена'}
          action={{
            label: 'Назад',
            onClick: () => router.back(),
          }}
        />
      </div>
    )
  }

  const breadcrumbItems = [
    { label: 'Результаты', href: '/results', icon: BarChart3 },
    { label: groupDetails.normalized_name, href: '#', icon: FileQuestion },
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
          className="flex items-center justify-between"
        >
          <div className="flex items-center gap-4">
            <Button 
              variant="outline" 
              size="icon" 
              onClick={() => router.push('/results')} 
              aria-label="Вернуться к результатам"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-3xl font-bold flex items-center gap-2">
                <FileQuestion className="h-8 w-8 text-primary" />
                {groupDetails.normalized_name}
              </h1>
              <div className="flex items-center gap-2 mt-1">
                <Badge variant="secondary">{groupDetails.category}</Badge>
                <span className="text-sm text-muted-foreground">
                  {totalItems.toLocaleString('ru-RU')} элементов
                </span>
                {avgConfidence && avgConfidence > 0 && (
                  <>
                    <span className="text-sm text-muted-foreground">•</span>
                    <span className="text-sm text-muted-foreground">
                      Средняя уверенность: {(avgConfidence * 100).toFixed(1)}%
                    </span>
                  </>
                )}
              </div>
            </div>
          </div>
          <ExportGroupButton
            normalizedName={groupDetails.normalized_name}
            category={groupDetails.category}
          />
        </motion.div>
      </FadeIn>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Основная информация */}
        <div className="lg:col-span-2 space-y-6">
          {/* Поиск и таблица */}
          <Card>
            <CardHeader>
              <CardTitle>Элементы группы</CardTitle>
              <CardDescription>
                Все элементы, объединенные в эту группу
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex flex-col gap-4">
                <FilterBar
                  filters={[
                    {
                      type: 'search',
                      key: 'search',
                      label: 'Поиск',
                      placeholder: 'Поиск по коду, названию или reference...',
                    },
                  ]}
                  values={{ search: searchTerm }}
                  onChange={(values) => setSearchTerm(values.search || '')}
                  onReset={() => setSearchTerm('')}
                />
                <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-3">
                  <div className="text-sm text-muted-foreground" role="status" aria-live="polite">
                    Найдено: {filteredItems.length.toLocaleString('ru-RU')} из {totalItems.toLocaleString('ru-RU')}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">На странице:</span>
                    <Select
                      value={limit.toString()}
                      onValueChange={(value) => {
                        setPage(1)
                        setLimit(Number(value))
                      }}
                    >
                      <SelectTrigger className="w-[100px]">
                        <SelectValue placeholder="50" />
                      </SelectTrigger>
                      <SelectContent align="end">
                        {[25, 50, 100, 200].map((option) => (
                          <SelectItem key={option} value={option.toString()}>
                            {option}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </div>

              <GroupItemsTable items={filteredItems} loading={loading} />

              <div className="pt-4">
                <Pagination
                  currentPage={page}
                  totalPages={totalPages}
                  onPageChange={setPage}
                  itemsPerPage={limit}
                  totalItems={totalItems}
                />
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Боковая панель */}
        <div className="space-y-6">
          {/* Статистика группы */}
          <Card>
            <CardHeader>
              <CardTitle>Статистика группы</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <StatCard
                title="Всего элементов"
                value={totalItems}
                variant="primary"
                className="p-0"
              />
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Категория:</span>
                <Badge variant="outline">{groupDetails.category}</Badge>
              </div>
              {avgConfidence && avgConfidence > 0 && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Средняя уверенность:</span>
                  <ConfidenceBadge confidence={avgConfidence} size="sm" />
                </div>
              )}
              {processingLevel && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Уровень обработки:</span>
                  <ProcessingLevelBadge level={processingLevel} />
                </div>
              )}
              {groupDetails.kpved_code && (
                <div className="flex justify-between items-center">
                  <span className="text-sm text-muted-foreground">Код КПВЭД:</span>
                  <KpvedBadge
                    code={groupDetails.kpved_code}
                    name={groupDetails.kpved_name}
                    confidence={groupDetails.kpved_confidence}
                    showConfidence={true}
                  />
                </div>
              )}
            </CardContent>
          </Card>

          {/* КПВЭД информация */}
          {groupDetails.kpved_code && (
            <Card>
              <CardHeader>
                <CardTitle>Классификация КПВЭД</CardTitle>
                <CardDescription>Классификатор продукции по видам экономической деятельности</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3">
                <div>
                  <p className="text-xs text-muted-foreground mb-1">Код:</p>
                  <p className="text-sm font-mono bg-muted p-2 rounded">
                    {groupDetails.kpved_code}
                  </p>
                </div>
                <div>
                  <p className="text-xs text-muted-foreground mb-1">Наименование:</p>
                  <p className="text-sm bg-muted p-2 rounded">
                    {groupDetails.kpved_name || 'Не определено'}
                  </p>
                </div>
                {groupDetails.kpved_confidence !== undefined && (
                  <div>
                    <p className="text-xs text-muted-foreground mb-1">Уверенность классификации:</p>
                    <div className="flex items-center gap-2">
                      <div className="flex-1 bg-muted rounded-full h-2">
                        <div
                          className="bg-primary h-2 rounded-full transition-all"
                          style={{ width: `${groupDetails.kpved_confidence * 100}%` }}
                        />
                      </div>
                      <span className="text-sm font-medium">
                        {(groupDetails.kpved_confidence * 100).toFixed(1)}%
                      </span>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Reference информация */}
          <Card>
            <CardHeader>
              <CardTitle>Reference</CardTitle>
            </CardHeader>
            <CardContent>
              <div className="space-y-2">
                <div>
                  <p className="text-xs text-muted-foreground mb-1">Нормализованный:</p>
                  <p className="text-sm font-mono bg-muted p-2 rounded break-all">
                    {groupDetails.normalized_reference}
                  </p>
                </div>
              </div>
            </CardContent>
          </Card>

          {/* Быстрые действия */}
          <Card>
            <CardHeader>
              <CardTitle>Действия</CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              <Button variant="outline" className="w-full justify-start" asChild>
                <Link href="/results">
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  К списку групп
                </Link>
              </Button>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  )
}
