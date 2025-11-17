'use client'

import { useState, useEffect } from 'react'
import { useParams } from 'next/navigation'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { 
  ArrowLeft,
  Search,
  CheckCircle2,
  XCircle,
  RefreshCw
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { FilterBar } from "@/components/common/filter-bar"

interface Benchmark {
  id: number
  original_name: string
  normalized_name: string
  category: string
  subcategory: string
  quality_score: number
  is_approved: boolean
  usage_count: number
  created_at: string
}

export default function BenchmarksPage() {
  const params = useParams()
  const clientId = params.clientId
  const projectId = params.projectId
  const [benchmarks, setBenchmarks] = useState<Benchmark[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [searchTerm, setSearchTerm] = useState('')
  const [categoryFilter, setCategoryFilter] = useState('')
  const [approvedOnly, setApprovedOnly] = useState(false)

  useEffect(() => {
    if (clientId && projectId) {
      fetchBenchmarks(clientId as string, projectId as string)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clientId, projectId, categoryFilter, approvedOnly])

  const fetchBenchmarks = async (clientId: string, projectId: string) => {
    setIsLoading(true)
    try {
      const params = new URLSearchParams()
      if (categoryFilter) params.append('category', categoryFilter)
      if (approvedOnly) params.append('approved_only', 'true')
      
      const url = `/api/clients/${clientId}/projects/${projectId}/benchmarks${params.toString() ? `?${params.toString()}` : ''}`
      const response = await fetch(url)
      if (!response.ok) throw new Error('Failed to fetch benchmarks')
      const data = await response.json()
      setBenchmarks(data.benchmarks || data || [])
    } catch (error) {
      console.error('Failed to fetch benchmarks:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const filteredBenchmarks = benchmarks.filter(benchmark => {
    const matchesSearch = !searchTerm || 
      benchmark.original_name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      benchmark.normalized_name.toLowerCase().includes(searchTerm.toLowerCase())
    const matchesCategory = !categoryFilter || benchmark.category === categoryFilter
    const matchesApproved = !approvedOnly || benchmark.is_approved
    return matchesSearch && matchesCategory && matchesApproved
  })

  const categories = Array.from(new Set(benchmarks.map(b => b.category)))

  return (
    <div className="container mx-auto p-6 space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="outline" size="icon" asChild>
          <Link href={`/clients/${clientId}/projects/${projectId}`}>
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <div className="flex-1">
          <h1 className="text-3xl font-bold">Эталонные записи</h1>
          <p className="text-muted-foreground">
            Управление эталонными записями проекта
          </p>
        </div>
      </div>

      {/* Фильтры */}
      <Card>
        <CardContent className="pt-6">
          <FilterBar
            filters={[
              {
                type: 'search',
                key: 'search',
                label: 'Поиск',
                placeholder: 'Поиск по названию...',
              },
              {
                type: 'select',
                key: 'category',
                label: 'Категория',
                options: [
                  { value: '', label: 'Все категории' },
                  ...categories.map(cat => ({ value: cat, label: cat })),
                ],
              },
              {
                type: 'checkbox',
                key: 'approvedOnly',
                label: 'Только утвержденные',
              },
            ]}
            values={{
              search: searchTerm,
              category: categoryFilter,
              approvedOnly: approvedOnly,
            }}
            onChange={(values) => {
              setSearchTerm(values.search || '')
              setCategoryFilter(values.category || '')
              setApprovedOnly(values.approvedOnly === true)
            }}
            onReset={() => {
              setSearchTerm('')
              setCategoryFilter('')
              setApprovedOnly(false)
            }}
          />
        </CardContent>
      </Card>

      {/* Таблица эталонов */}
      <Card>
        <CardHeader>
          <CardTitle>Эталонные записи ({filteredBenchmarks.length})</CardTitle>
          <CardDescription>
            Список всех эталонных записей для этого проекта
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <LoadingState message="Загрузка эталонов..." size="lg" fullScreen />
          ) : (
            <div className="rounded-md border">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Оригинальное название</TableHead>
                    <TableHead>Нормализованное</TableHead>
                    <TableHead>Категория</TableHead>
                    <TableHead>Качество</TableHead>
                    <TableHead>Статус</TableHead>
                    <TableHead>Использований</TableHead>
                    <TableHead>Действия</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {filteredBenchmarks.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="text-center py-8">
                        <EmptyState
                          icon={CheckCircle2}
                          title="Эталонные записи не найдены"
                          description="Создайте первую эталонную запись для этого проекта"
                        />
                      </TableCell>
                    </TableRow>
                  ) : (
                    filteredBenchmarks.map((benchmark) => (
                      <TableRow key={benchmark.id}>
                        <TableCell className="font-medium">{benchmark.original_name}</TableCell>
                        <TableCell>{benchmark.normalized_name}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{benchmark.category}</Badge>
                        </TableCell>
                        <TableCell>
                          {Math.round(benchmark.quality_score * 100)}%
                        </TableCell>
                        <TableCell>
                          {benchmark.is_approved ? (
                            <Badge variant="default" className="gap-1">
                              <CheckCircle2 className="h-3 w-3" />
                              Утвержден
                            </Badge>
                          ) : (
                            <Badge variant="secondary">На проверке</Badge>
                          )}
                        </TableCell>
                        <TableCell>{benchmark.usage_count}</TableCell>
                        <TableCell>
                          <div className="flex gap-2">
                            {!benchmark.is_approved && (
                              <Button size="sm" variant="outline">
                                Утвердить
                              </Button>
                            )}
                            <Button size="sm" variant="outline">
                              Редактировать
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

