'use client'

import { useState, useEffect, useCallback } from 'react'
import { useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'
import { Star, Search, Filter, Plus, Edit, Trash2, Building2, Package } from 'lucide-react'
import { Pagination } from '@/components/ui/pagination'
import { formatDate } from '@/lib/locale'
import { toast } from 'sonner'
import { DataTable } from '@/components/common/data-table'

interface Benchmark {
  id: string
  entity_type: 'counterparty' | 'nomenclature'
  name: string
  data: Record<string, unknown>
  source_upload_id?: number
  source_client_id?: number
  is_active: boolean
  created_at: string
  updated_at: string
  variations?: string[]
}

interface BenchmarkListResponse {
  benchmarks: Benchmark[]
  total: number
  limit: number
  offset: number
}

export default function BenchmarksPage() {
  const router = useRouter()
  const [benchmarks, setBenchmarks] = useState<Benchmark[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [currentPage, setCurrentPage] = useState(1)
  const [entityType, setEntityType] = useState<string>('all')
  const [activeOnly, setActiveOnly] = useState(true)
  const [searchTerm, setSearchTerm] = useState('')
  const itemsPerPage = 20

  const fetchBenchmarks = useCallback(async () => {
    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams({
        limit: itemsPerPage.toString(),
        offset: ((currentPage - 1) * itemsPerPage).toString(),
      })

      if (entityType && entityType !== 'all') {
        params.append('type', entityType)
      }
      if (activeOnly) {
        params.append('active', 'true')
      }

      const response = await fetch(`/api/benchmarks?${params.toString()}`)
      if (!response.ok) {
        throw new Error('Failed to fetch benchmarks')
      }

      const data: BenchmarkListResponse = await response.json()
      setBenchmarks(data.benchmarks || [])
      setTotal(data.total || 0)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки эталонов')
      setBenchmarks([])
    } finally {
      setLoading(false)
    }
  }, [currentPage, entityType, activeOnly])

  useEffect(() => {
    fetchBenchmarks()
  }, [fetchBenchmarks])

  const handleDelete = async (id: string) => {
    const confirmed = window.confirm('Вы уверены, что хотите удалить этот эталон? Это действие нельзя отменить.')
    if (!confirmed) {
      return
    }

    try {
      const response = await fetch(`/api/benchmarks/${id}`, {
        method: 'DELETE',
      })

      if (!response.ok) {
        throw new Error('Failed to delete benchmark')
      }

      toast.success('Эталон успешно удален')
      fetchBenchmarks()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : 'Ошибка удаления эталона')
    }
  }

  const handleCreateBenchmark = () => {
    router.push('/benchmarks/create')
  }

  const filteredBenchmarks = benchmarks.filter(b => {
    if (!searchTerm) return true
    const searchLower = searchTerm.toLowerCase()
    return (
      b.name.toLowerCase().includes(searchLower) ||
      b.id.toLowerCase().includes(searchLower) ||
      (b.variations?.some(v => v.toLowerCase().includes(searchLower)))
    )
  })

  const columns = [
    {
      key: 'entity_type',
      header: 'Тип',
      render: (row: Benchmark) => (
        <Badge variant={row.entity_type === 'counterparty' ? 'default' : 'secondary'}>
          {row.entity_type === 'counterparty' ? (
            <><Building2 className="w-3 h-3 mr-1" /> Контрагент</>
          ) : (
            <><Package className="w-3 h-3 mr-1" /> Номенклатура</>
          )}
        </Badge>
      ),
      sortable: true,
    },
    {
      key: 'name',
      header: 'Название',
      accessor: (row: Benchmark) => row.name,
      sortable: true,
    },
    {
      key: 'data',
      header: 'Данные',
      render: (row: Benchmark) => (
        <div className="max-w-xs truncate text-sm text-muted-foreground">
          {JSON.stringify(row.data).substring(0, 50)}...
        </div>
      ),
    },
    {
      key: 'variations',
      header: 'Вариации',
      render: (row: Benchmark) => (
        row.variations && row.variations.length > 0 ? (
          <Badge variant="outline">{row.variations.length}</Badge>
        ) : (
          <span className="text-muted-foreground">-</span>
        )
      ),
    },
    {
      key: 'is_active',
      header: 'Статус',
      render: (row: Benchmark) => (
        <Badge variant={row.is_active ? 'default' : 'secondary'}>
          {row.is_active ? 'Активен' : 'Неактивен'}
        </Badge>
      ),
      sortable: true,
    },
    {
      key: 'created_at',
      header: 'Создан',
      accessor: (row: Benchmark) => formatDate(row.created_at),
      sortable: true,
    },
    {
      key: 'actions',
      header: 'Действия',
      render: (row: Benchmark) => (
        <div className="flex gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => router.push(`/benchmarks/${row.id}`)}
          >
            <Edit className="w-4 h-4" />
          </Button>
          <Button
            variant="ghost"
            size="sm"
            onClick={() => handleDelete(row.id)}
          >
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      ),
    },
  ]

  const breadcrumbItems = [
    { label: 'Эталоны', href: '/benchmarks', icon: Star },
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
          className="flex flex-col md:flex-row md:items-center justify-between gap-4"
        >
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Star className="w-8 h-8 text-primary" />
              Управление эталонами
            </h1>
            <p className="text-muted-foreground mt-1">
              Библиотека проверенных и подтвержденных данных для нормализации
            </p>
          </div>
          <Button onClick={handleCreateBenchmark} className="flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Создать эталон
          </Button>
        </motion.div>
      </FadeIn>

      <Card>
        <CardHeader>
          <div className="flex flex-col md:flex-row gap-4 items-start md:items-center justify-between">
            <div>
              <CardTitle>Эталоны</CardTitle>
              <CardDescription>
                Всего: {total} | Показано: {filteredBenchmarks.length}
              </CardDescription>
            </div>
            <div className="flex flex-wrap gap-2">
              <div className="flex items-center gap-2">
                <Input
                  placeholder="Поиск по названию..."
                  value={searchTerm}
                  onChange={(e) => setSearchTerm(e.target.value)}
                  className="w-[200px]"
                />
                <Search className="w-4 h-4 text-muted-foreground" />
              </div>
              <Select value={entityType || 'all'} onValueChange={(v) => setEntityType(v === 'all' ? '' : v)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="Все типы" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все типы</SelectItem>
                  <SelectItem value="counterparty">Контрагенты</SelectItem>
                  <SelectItem value="nomenclature">Номенклатура</SelectItem>
                </SelectContent>
              </Select>
              <Button
                variant={activeOnly ? 'default' : 'outline'}
                onClick={() => setActiveOnly(!activeOnly)}
                size="sm"
              >
                <Filter className="w-4 h-4 mr-2" />
                {activeOnly ? 'Только активные' : 'Все'}
              </Button>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <LoadingState message="Загрузка эталонов..." />
          ) : error ? (
            <EmptyState
              icon={Star}
              title="Ошибка загрузки"
              description={error}
            />
          ) : filteredBenchmarks.length === 0 ? (
            <EmptyState
              icon={Star}
              title="Эталоны не найдены"
              description="Создайте первый эталон из загрузки или вручную"
              action={{
                label: 'Создать эталон',
                onClick: handleCreateBenchmark,
              }}
            />
          ) : (
            <>
              <DataTable
                data={filteredBenchmarks}
                columns={columns}
                loading={loading}
                emptyMessage="Эталоны не найдены"
                getRowId={(row) => row.id}
              />
              {total > itemsPerPage && (
                <div className="mt-4">
                  <Pagination
                    currentPage={currentPage}
                    totalPages={Math.ceil(total / itemsPerPage)}
                    onPageChange={setCurrentPage}
                  />
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

