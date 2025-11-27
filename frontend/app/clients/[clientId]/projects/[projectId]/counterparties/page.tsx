'use client'

import { useState, useEffect, useMemo, useCallback } from 'react'
import { useParams, useRouter, useSearchParams, usePathname } from 'next/navigation'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  ArrowLeft,
  Search,
  Filter,
  Edit,
  Save,
  X,
  Calendar,
  Building2,
  Mail,
  Phone,
  MapPin,
  CreditCard,
  CheckCircle2,
  AlertCircle,
  RefreshCw,
  ChevronLeft,
  ChevronRight,
  Factory,
  Sparkles,
  Copy,
  Download,
  Users,
  Loader2,
  ArrowUpDown,
  ArrowUp,
  ArrowDown,
  Eye,
  ExternalLink,
} from "lucide-react"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { Textarea } from "@/components/ui/textarea"
import { Progress } from "@/components/ui/progress"
import { FadeIn } from "@/components/animations/fade-in"
import { StaggerContainer, StaggerItem } from "@/components/animations/stagger-container"
import { motion } from "framer-motion"
import { CounterpartyAnalytics } from "@/components/counterparties/CounterpartyAnalytics"
import { apiRequest, handleApiError, formatError } from "@/lib/api-utils"
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { Target } from "lucide-react"
import { toast } from "sonner"

interface NormalizedCounterparty {
  id: number
  client_project_id: number
  source_reference: string
  source_name: string
  normalized_name: string
  tax_id: string
  kpp: string
  bin: string
  legal_address: string
  postal_address: string
  contact_phone: string
  contact_email: string
  contact_person: string
  legal_form: string
  bank_name: string
  bank_account: string
  correspondent_account: string
  bik: string
  benchmark_id?: number
  quality_score: number
  enrichment_applied: boolean
  source_enrichment: string
  source_database: string
  subcategory: string
  created_at: string
  updated_at: string
}

interface Project {
  id: number
  name: string
}

export default function CounterpartiesPage() {
  const params = useParams()
  const router = useRouter()
  const searchParams = useSearchParams()
  const pathname = usePathname()
  const clientId = params.clientId as string
  const projectId = params.projectId as string

  // Инициализация из URL параметров
  const [counterparties, setCounterparties] = useState<NormalizedCounterparty[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState(() => searchParams.get('search') || '')
  const [debouncedSearchQuery, setDebouncedSearchQuery] = useState(() => searchParams.get('search') || '')
  const [selectedProject, setSelectedProject] = useState<string>(() => searchParams.get('project') || projectId || 'all')
  const [selectedEnrichment, setSelectedEnrichment] = useState<string>(() => searchParams.get('enrichment') || 'all')
  const [selectedSubcategory, setSelectedSubcategory] = useState<string>(() => searchParams.get('subcategory') || 'all')
  const [selectedQuality, setSelectedQuality] = useState<string>(() => searchParams.get('quality') || 'all')
  const [selectedDateRange, setSelectedDateRange] = useState<string>(() => searchParams.get('dateRange') || 'all')
  const [selectedIds, setSelectedIds] = useState<Set<number>>(new Set())
  const [isBulkEnriching, setIsBulkEnriching] = useState(false)
  const [bulkEnrichProgress, setBulkEnrichProgress] = useState({ current: 0, total: 0 })
  const [currentPage, setCurrentPage] = useState(() => {
    const page = searchParams.get('page')
    return page ? parseInt(page, 10) : 1
  })
  const [totalCount, setTotalCount] = useState(0)
  const [limit, setLimit] = useState(() => {
    const limitParam = searchParams.get('limit')
    return limitParam ? parseInt(limitParam, 10) : 20
  })
  const [editingCounterparty, setEditingCounterparty] = useState<NormalizedCounterparty | null>(null)
  const [editForm, setEditForm] = useState<Partial<NormalizedCounterparty>>({})
  const [isSaving, setIsSaving] = useState(false)
  const [editErrors, setEditErrors] = useState<Record<string, string>>({})
  const [isEnriching, setIsEnriching] = useState<number | null>(null)
  const [isExporting, setIsExporting] = useState(false)
  const [showBulkEnrichConfirm, setShowBulkEnrichConfirm] = useState(false)
  const [showDuplicates, setShowDuplicates] = useState(false)
  const [duplicates, setDuplicates] = useState<any[]>([])
  const [isLoadingDuplicates, setIsLoadingDuplicates] = useState(false)
  const [isLoadingStats, setIsLoadingStats] = useState(false)
  const [stats, setStats] = useState<{
    total_count?: number
    manufacturers_count?: number
    with_benchmark?: number
    enriched?: number
    subcategory_stats?: Record<string, number>
  }>({})
  const [sortKey, setSortKey] = useState<string | null>(null)
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc')
  const [viewingCounterparty, setViewingCounterparty] = useState<NormalizedCounterparty | null>(null)

  // Функция для обновления URL параметров
  const updateURL = useCallback((updates: Record<string, string | null>) => {
    const currentParams = searchParams.toString()
    const params = new URLSearchParams(currentParams)
    
    Object.entries(updates).forEach(([key, value]) => {
      if (value && value !== 'all' && value !== '1') {
        params.set(key, value)
      } else {
        params.delete(key)
      }
    })
    
    router.replace(`${pathname}?${params.toString()}`, { scroll: false })
  }, [searchParams, pathname, router])

  // Синхронизация selectedProject с projectId из URL
  useEffect(() => {
    if (projectId && projectId !== selectedProject) {
      setSelectedProject(projectId)
      updateURL({ project: projectId })
    } else if (!projectId && selectedProject !== 'all') {
      setSelectedProject('all')
      updateURL({ project: null })
    }
  }, [projectId, selectedProject, updateURL])

  // Debounce для поиска - задержка 500мс
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedSearchQuery(searchQuery)
      // Сбрасываем страницу при изменении поиска
      if (searchQuery !== debouncedSearchQuery) {
        setCurrentPage(1)
      }
    }, 500)

    return () => clearTimeout(timer)
  }, [searchQuery])


  // Обновление URL при изменении фильтров
  useEffect(() => {
    updateURL({
      search: debouncedSearchQuery || null,
      project: selectedProject !== 'all' ? selectedProject : null,
      enrichment: selectedEnrichment !== 'all' ? selectedEnrichment : null,
      subcategory: selectedSubcategory !== 'all' ? selectedSubcategory : null,
      quality: selectedQuality !== 'all' ? selectedQuality : null,
      dateRange: selectedDateRange !== 'all' ? selectedDateRange : null,
      page: currentPage > 1 ? currentPage.toString() : null,
      limit: limit !== 20 ? limit.toString() : null,
    })
  }, [debouncedSearchQuery, selectedProject, selectedEnrichment, selectedSubcategory, selectedQuality, selectedDateRange, currentPage, limit, updateURL])

  useEffect(() => {
    if (clientId) {
      fetchCounterparties()
      if (selectedProject !== 'all') {
        fetchStats()
      }
    }
  }, [clientId, selectedProject, currentPage, debouncedSearchQuery, selectedEnrichment, selectedSubcategory, selectedQuality, selectedDateRange, limit])

  const fetchCounterparties = async () => {
    setIsLoading(true)
    setError(null)

    try {
      const offset = (currentPage - 1) * limit
      let url = `/api/counterparties/normalized?client_id=${clientId}&offset=${offset}&limit=${limit}`
      
      if (selectedProject !== 'all') {
        url += `&project_id=${selectedProject}`
      }

      if (debouncedSearchQuery) {
        url += `&search=${encodeURIComponent(debouncedSearchQuery)}`
      }

      // Передаем фильтры на сервер
      if (selectedEnrichment !== 'all') {
        url += `&enrichment=${encodeURIComponent(selectedEnrichment)}`
      }

      if (selectedSubcategory !== 'all') {
        url += `&subcategory=${encodeURIComponent(selectedSubcategory)}`
      }

      const data = await apiRequest<{
        counterparties: NormalizedCounterparty[]
        projects: Project[]
        total: number
      }>(url)
      
      // Данные уже отфильтрованы на сервере
      setCounterparties(data.counterparties || [])
      setProjects(data.projects || [])
      // totalCount теперь учитывает все фильтры
      setTotalCount(data.total || 0)
    } catch (err) {
      const errorMessage = formatError(err)
      setError(errorMessage)
      console.error('Failed to fetch counterparties:', err)
      // Сбрасываем данные при ошибке
      setCounterparties([])
      setTotalCount(0)
    } finally {
      setIsLoading(false)
    }
  }

  const fetchStats = async () => {
    if (selectedProject === 'all') {
      setStats({})
      return
    }

    setIsLoadingStats(true)
    try {
      const response = await fetch(`/api/counterparties/normalized/stats?project_id=${selectedProject}`)
      if (response.ok) {
        const data = await response.json()
        setStats(data)
      } else {
        console.error('Failed to fetch stats:', response.statusText)
        setStats({})
      }
    } catch (err) {
      console.error('Failed to fetch stats:', err)
      setStats({})
    } finally {
      setIsLoadingStats(false)
    }
  }

  const handleEdit = (counterparty: NormalizedCounterparty) => {
    setEditingCounterparty(counterparty)
    setEditForm({
      normalized_name: counterparty.normalized_name,
      tax_id: counterparty.tax_id,
      kpp: counterparty.kpp,
      bin: counterparty.bin,
      legal_address: counterparty.legal_address,
      postal_address: counterparty.postal_address,
      contact_phone: counterparty.contact_phone,
      contact_email: counterparty.contact_email,
      contact_person: counterparty.contact_person,
      legal_form: counterparty.legal_form,
      bank_name: counterparty.bank_name,
      bank_account: counterparty.bank_account,
      correspondent_account: counterparty.correspondent_account,
      bik: counterparty.bik,
      quality_score: counterparty.quality_score,
      source_enrichment: counterparty.source_enrichment,
      subcategory: counterparty.subcategory,
    })
  }

  // Валидация полей формы
  const validateEditForm = (): boolean => {
    const errors: Record<string, string> = {}
    
    // Валидация названия
    if (!editForm.normalized_name || editForm.normalized_name.trim() === '') {
      errors.normalized_name = 'Название обязательно для заполнения'
    }
    
    // Валидация ИНН (10 или 12 цифр)
    if (editForm.tax_id && editForm.tax_id.trim() !== '') {
      const innRegex = /^\d{10}$|^\d{12}$/
      if (!innRegex.test(editForm.tax_id)) {
        errors.tax_id = 'ИНН должен содержать 10 или 12 цифр'
      }
    }
    
    // Валидация БИН (12 цифр)
    if (editForm.bin && editForm.bin.trim() !== '') {
      const binRegex = /^\d{12}$/
      if (!binRegex.test(editForm.bin)) {
        errors.bin = 'БИН должен содержать 12 цифр'
      }
    }
    
    // Валидация КПП (9 символов)
    if (editForm.kpp && editForm.kpp.trim() !== '') {
      const kppRegex = /^\d{9}$/
      if (!kppRegex.test(editForm.kpp)) {
        errors.kpp = 'КПП должен содержать 9 цифр'
      }
    }
    
    // Валидация email
    if (editForm.contact_email && editForm.contact_email.trim() !== '') {
      const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
      if (!emailRegex.test(editForm.contact_email)) {
        errors.contact_email = 'Некорректный формат email'
      }
    }
    
    // Валидация телефона (базовая проверка)
    if (editForm.contact_phone && editForm.contact_phone.trim() !== '') {
      const phoneRegex = /^[\d\s\-\+\(\)]+$/
      if (!phoneRegex.test(editForm.contact_phone) || editForm.contact_phone.replace(/\D/g, '').length < 10) {
        errors.contact_phone = 'Некорректный формат телефона'
      }
    }
    
    // Валидация БИК (9 цифр)
    if (editForm.bik && editForm.bik.trim() !== '') {
      const bikRegex = /^\d{9}$/
      if (!bikRegex.test(editForm.bik)) {
        errors.bik = 'БИК должен содержать 9 цифр'
      }
    }
    
    // Валидация качества (0-1)
    if (editForm.quality_score !== undefined && editForm.quality_score !== null) {
      if (editForm.quality_score < 0 || editForm.quality_score > 1) {
        errors.quality_score = 'Оценка качества должна быть от 0 до 1'
      }
    }
    
    setEditErrors(errors)
    return Object.keys(errors).length === 0
  }

  const handleSave = async () => {
    if (!editingCounterparty) return

    // Валидация перед сохранением
    if (!validateEditForm()) {
      toast.error('Исправьте ошибки в форме', {
        description: 'Проверьте выделенные поля',
      })
      return
    }

    setIsSaving(true)
    setError(null)
    setEditErrors({})
    try {
      await apiRequest(`/api/counterparties/normalized/${editingCounterparty.id}`, {
        method: 'PUT',
        body: JSON.stringify(editForm),
      })

      await fetchCounterparties()
      if (selectedProject !== 'all') {
        await fetchStats()
      }
      setEditingCounterparty(null)
      setEditForm({})
      setEditErrors({})
      toast.success('Контрагент успешно обновлен')
    } catch (err) {
      const errorMessage = formatError(err)
      setError(errorMessage)
      toast.error('Не удалось обновить контрагента', {
        description: errorMessage,
      })
    } finally {
      setIsSaving(false)
    }
  }

  const getEnrichmentBadge = (source: string) => {
    if (!source || source === '') {
      return <Badge variant="outline">Не нормализован</Badge>
    }
    
    const colors: Record<string, string> = {
      'Adata.kz': 'bg-blue-500',
      'Dadata.ru': 'bg-green-500',
      'gisp.gov.ru': 'bg-purple-500',
    }
    
    return (
      <Badge className={colors[source] || 'bg-gray-500'}>
        {source}
      </Badge>
    )
  }

  const handleBulkEnrichClick = useCallback(() => {
    if (selectedIds.size === 0) return
    setShowBulkEnrichConfirm(true)
  }, [selectedIds.size])

  const handleBulkEnrich = useCallback(async () => {
    if (selectedIds.size === 0) return
    
    setShowBulkEnrichConfirm(false)
    setIsBulkEnriching(true)
    setError(null)
    setBulkEnrichProgress({ current: 0, total: selectedIds.size })
    
    try {
      const selectedCounterparties = counterparties.filter(cp => selectedIds.has(cp.id))
      const total = selectedCounterparties.length
      let processed = 0
      let successCount = 0
      let failedCount = 0
      
      // Обрабатываем последовательно для отслеживания прогресса
      for (const cp of selectedCounterparties) {
        try {
          await apiRequest('/api/counterparties/normalized/enrich', {
            method: 'POST',
            body: JSON.stringify({
              counterparty_id: cp.id,
              inn: cp.tax_id,
              bin: cp.bin,
            }),
          })
          successCount++
        } catch (err) {
          failedCount++
          console.error(`Failed to enrich counterparty ${cp.id}:`, err)
        }
        
        processed++
        setBulkEnrichProgress({ current: processed, total })
      }
      
      if (failedCount > 0) {
        const errorMsg = `Обогащено: ${successCount} из ${total}, ошибок: ${failedCount}`
        setError(errorMsg)
        toast.warning('Массовое обогащение завершено с ошибками', {
          description: errorMsg,
        })
      } else {
        setError(null)
        toast.success(`Успешно обогащено ${successCount} контрагентов`)
      }
      
      await fetchCounterparties()
      if (selectedProject !== 'all') {
        await fetchStats()
      }
      setSelectedIds(new Set())
      setBulkEnrichProgress({ current: 0, total: 0 })
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Ошибка массового обогащения'
      setError(errorMessage)
      toast.error('Ошибка массового обогащения', {
        description: errorMessage,
      })
      setBulkEnrichProgress({ current: 0, total: 0 })
    } finally {
      setIsBulkEnriching(false)
    }
  }, [selectedIds, counterparties, selectedProject])

  const handleEnrich = async (counterparty: NormalizedCounterparty) => {
    setIsEnriching(counterparty.id)
    setError(null)
    
    try {
      const data = await apiRequest<{ success: boolean; message?: string }>('/api/counterparties/normalized/enrich', {
        method: 'POST',
        body: JSON.stringify({
          counterparty_id: counterparty.id,
          inn: counterparty.tax_id,
          bin: counterparty.bin,
        }),
      })

      if (data.success) {
        await fetchCounterparties()
        if (selectedProject !== 'all') {
          await fetchStats()
        }
        toast.success('Данные контрагента успешно обогащены')
      } else {
        throw new Error(data.message || 'Enrichment failed')
      }
    } catch (err) {
      const errorMessage = formatError(err)
      setError(errorMessage)
      toast.error('Не удалось обогатить данные', {
        description: errorMessage,
      })
    } finally {
      setIsEnriching(null)
    }
  }

  const handleExport = async (format: 'json' | 'csv' | 'xml' = 'json') => {
    if (selectedProject === 'all') {
      setError('Выберите проект для экспорта')
      return
    }

    setIsExporting(true)
    setError(null)

    try {
      const response = await fetch('/api/counterparties/normalized/export', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          project_id: parseInt(selectedProject),
          format: format,
        }),
      })

      if (!response.ok) {
        const errorMessage = await handleApiError(response)
        throw new Error(errorMessage)
      }

      if (format === 'json') {
        const data = await response.json()
        const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `counterparties_${selectedProject}_${new Date().toISOString().split('T')[0]}.json`
        a.click()
        URL.revokeObjectURL(url)
      } else {
        const blob = await response.blob()
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = `counterparties_${selectedProject}_${new Date().toISOString().split('T')[0]}.${format}`
        a.click()
        URL.revokeObjectURL(url)
        toast.success(`Экспорт в ${format.toUpperCase()} выполнен успешно`)
      }
    } catch (err) {
      const errorMessage = formatError(err)
      setError(errorMessage)
      toast.error('Не удалось экспортировать данные', {
        description: errorMessage,
      })
    } finally {
      setIsExporting(false)
    }
  }

  const handleLoadDuplicates = async () => {
    if (selectedProject === 'all') {
      setError('Выберите проект для просмотра дубликатов')
      return
    }

    setIsLoadingDuplicates(true)
    setError(null)

    try {
      const data = await apiRequest<{ groups: any[] }>(`/api/counterparties/normalized/duplicates?project_id=${selectedProject}`)
      setDuplicates(data.groups || [])
      setShowDuplicates(true)
    } catch (err) {
      setError(formatError(err))
    } finally {
      setIsLoadingDuplicates(false)
    }
  }

  const handleMergeDuplicates = async (groupId: string, masterId: number, mergeIds: number[]) => {
    setError(null)
    try {
      await apiRequest(`/api/counterparties/normalized/duplicates/${groupId}/merge`, {
        method: 'POST',
        body: JSON.stringify({
          master_id: masterId,
          merge_ids: mergeIds,
        }),
      })

      await handleLoadDuplicates()
      await fetchCounterparties()
      toast.success('Дубликаты успешно объединены')
    } catch (err) {
      const errorMessage = formatError(err)
      setError(errorMessage)
      toast.error('Не удалось объединить дубликаты', {
        description: errorMessage,
      })
    }
  }

  const totalPages = useMemo(() => Math.ceil(totalCount / limit), [totalCount, limit])
  
  // Фильтрация по качеству данных
  const filteredByQuality = useMemo(() => {
    if (selectedQuality === 'all') return counterparties

    return counterparties.filter(cp => {
      if (selectedQuality === 'no-quality') {
        return cp.quality_score === undefined || cp.quality_score === null
      }
      
      const score = cp.quality_score ?? 0
      switch (selectedQuality) {
        case 'excellent':
          return score >= 0.9
        case 'good':
          return score >= 0.7 && score < 0.9
        case 'fair':
          return score >= 0.5 && score < 0.7
        case 'poor':
          return score < 0.5
        default:
          return true
      }
    })
  }, [counterparties, selectedQuality])

  // Фильтрация по дате обновления
  const filteredByDate = useMemo(() => {
    if (selectedDateRange === 'all') return filteredByQuality

    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    
    return filteredByQuality.filter(cp => {
      const updatedDate = new Date(cp.updated_at)
      const updatedDay = new Date(updatedDate.getFullYear(), updatedDate.getMonth(), updatedDate.getDate())
      
      switch (selectedDateRange) {
        case 'today':
          return updatedDay.getTime() === today.getTime()
        case 'week':
          const weekAgo = new Date(today)
          weekAgo.setDate(weekAgo.getDate() - 7)
          return updatedDate >= weekAgo
        case 'month':
          const monthAgo = new Date(today)
          monthAgo.setMonth(monthAgo.getMonth() - 1)
          return updatedDate >= monthAgo
        case 'quarter':
          const quarterAgo = new Date(today)
          quarterAgo.setMonth(quarterAgo.getMonth() - 3)
          return updatedDate >= quarterAgo
        case 'year':
          const yearAgo = new Date(today)
          yearAgo.setFullYear(yearAgo.getFullYear() - 1)
          return updatedDate >= yearAgo
        default:
          return true
      }
    })
  }, [filteredByQuality, selectedDateRange])

  // Мемоизация обработчиков для оптимизации (перемещено сюда для использования в useEffect)
  const handleSelectAll = useCallback((checked: boolean) => {
    if (checked) {
      setSelectedIds(new Set(filteredByDate.map(cp => cp.id)))
    } else {
      setSelectedIds(new Set())
    }
  }, [filteredByDate])

  const handleToggleSelection = useCallback((id: number, checked: boolean) => {
    setSelectedIds(prev => {
      const newSelected = new Set(prev)
      if (checked) {
        newSelected.add(id)
      } else {
        newSelected.delete(id)
      }
      return newSelected
    })
  }, [])

  // Горячие клавиши (перемещено сюда, чтобы использовать handleSelectAll)
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // Ctrl/Cmd + K для фокуса на поиск
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault()
        const searchInput = document.querySelector('input[placeholder*="Поиск"]') as HTMLInputElement
        if (searchInput) {
          searchInput.focus()
          searchInput.select()
        }
      }
      // Escape для закрытия диалогов
      if (e.key === 'Escape') {
        if (viewingCounterparty) {
          setViewingCounterparty(null)
        }
        if (editingCounterparty) {
          setEditingCounterparty(null)
        }
        if (showDuplicates) {
          setShowDuplicates(false)
        }
      }
      // Ctrl/Cmd + A для выбора всех (только если не в input)
      if ((e.ctrlKey || e.metaKey) && e.key === 'a' && e.target instanceof HTMLInputElement === false && e.target instanceof HTMLTextAreaElement === false) {
        e.preventDefault()
        handleSelectAll(true)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [viewingCounterparty, editingCounterparty, showDuplicates, handleSelectAll])

  // Функция копирования в буфер обмена
  const handleCopyToClipboard = useCallback(async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success(`${label} скопирован в буфер обмена`, {
        description: text.length > 50 ? `${text.substring(0, 50)}...` : text,
        duration: 2000,
      })
    } catch (err) {
      console.error('Failed to copy to clipboard:', err)
      toast.error('Не удалось скопировать в буфер обмена')
    }
  }, [])

  // Обработчик сортировки
  const handleSort = useCallback((key: string) => {
    if (sortKey === key) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc')
    } else {
      setSortKey(key)
      setSortDirection('asc')
    }
  }, [sortKey, sortDirection])

  // Функция получения иконки сортировки
  const getSortIcon = useCallback((key: string) => {
    if (sortKey !== key) {
      return <ArrowUpDown className="h-4 w-4 text-muted-foreground" />
    }
    return sortDirection === 'asc' 
      ? <ArrowUp className="h-4 w-4 text-primary" />
      : <ArrowDown className="h-4 w-4 text-primary" />
  }, [sortKey, sortDirection])

  // Сортировка контрагентов
  const sortedCounterparties = useMemo(() => {
    if (!sortKey) return filteredByDate

    return [...filteredByDate].sort((a, b) => {
      let aValue: any
      let bValue: any

      switch (sortKey) {
        case 'name':
          aValue = a.normalized_name || ''
          bValue = b.normalized_name || ''
          break
        case 'tax_id':
          aValue = a.tax_id || a.bin || ''
          bValue = b.tax_id || b.bin || ''
          break
        case 'address':
          aValue = a.legal_address || ''
          bValue = b.legal_address || ''
          break
        case 'source':
          aValue = a.source_enrichment || ''
          bValue = b.source_enrichment || ''
          break
        case 'updated':
          aValue = new Date(a.updated_at).getTime()
          bValue = new Date(b.updated_at).getTime()
          break
        default:
          return 0
      }

      // Обработка null/undefined
      if (aValue == null && bValue == null) return 0
      if (aValue == null) return 1
      if (bValue == null) return -1

      // Сравнение значений
      let comparison = 0
      if (typeof aValue === 'string' && typeof bValue === 'string') {
        comparison = aValue.localeCompare(bValue, 'ru-RU', { numeric: true, sensitivity: 'base' })
      } else if (typeof aValue === 'number' && typeof bValue === 'number') {
        comparison = aValue - bValue
      } else {
        comparison = String(aValue).localeCompare(String(bValue), 'ru-RU', { numeric: true })
      }

      return sortDirection === 'asc' ? comparison : -comparison
    })
  }, [filteredByDate, sortKey, sortDirection])

  const breadcrumbItems = [
    { label: 'Клиенты', href: '/clients', icon: Building2 },
    { label: 'Проекты', href: `/clients/${clientId}/projects`, icon: Target },
    { label: 'Контрагенты', href: `#`, icon: Building2 },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>
      {/* Header */}
      <FadeIn>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4">
            <Button 
              variant="ghost" 
              size="icon"
              onClick={() => router.push(`/clients/${clientId}/projects/${projectId}`)}
              aria-label="Назад к проекту"
            >
              <motion.div whileHover={{ scale: 1.1 }} whileTap={{ scale: 0.9 }}>
                <ArrowLeft className="h-4 w-4" />
              </motion.div>
            </Button>
            <div>
              <h1 className="text-3xl font-bold">Контрагенты</h1>
              <p className="text-muted-foreground">Просмотр и управление контрагентами проекта</p>
            </div>
          </div>
        </div>
      </FadeIn>

      {/* Filters */}
      <FadeIn delay={0.1}>
        <Card>
        <CardHeader>
          <CardTitle>Фильтры</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-7 gap-4">
            <div className="space-y-2">
              <Label>Поиск</Label>
              <div className="relative">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="Поиск по названию, ИНН, адресу, email..."
                  value={searchQuery}
                  onChange={(e) => {
                    setSearchQuery(e.target.value)
                    setCurrentPage(1)
                  }}
                  className="pl-8"
                />
              </div>
            </div>
            <div className="space-y-2">
              <Label>Проект</Label>
              <Select value={selectedProject} onValueChange={(value) => {
                setSelectedProject(value)
                setCurrentPage(1)
              }}>
                <SelectTrigger>
                  <SelectValue placeholder="Все проекты" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все проекты</SelectItem>
                  {projects.map((p) => (
                    <SelectItem key={p.id} value={p.id.toString()}>
                      {p.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Источник нормализации</Label>
              <Select value={selectedEnrichment} onValueChange={(value) => {
                setSelectedEnrichment(value)
                setCurrentPage(1)
              }}>
                <SelectTrigger>
                  <SelectValue placeholder="Все источники" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все источники</SelectItem>
                  <SelectItem value="Adata.kz">Adata.kz</SelectItem>
                  <SelectItem value="Dadata.ru">Dadata.ru</SelectItem>
                  <SelectItem value="gisp.gov.ru">gisp.gov.ru</SelectItem>
                  <SelectItem value="none">Не нормализован</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Категория</Label>
              <Select value={selectedSubcategory} onValueChange={(value) => {
                setSelectedSubcategory(value)
                setCurrentPage(1)
              }}>
                <SelectTrigger>
                  <SelectValue placeholder="Все категории" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все категории</SelectItem>
                  <SelectItem value="manufacturer">Производители</SelectItem>
                  <SelectItem value="none">Без категории</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Качество данных</Label>
              <Select value={selectedQuality} onValueChange={(value) => {
                setSelectedQuality(value)
                setCurrentPage(1)
              }}>
                <SelectTrigger>
                  <SelectValue placeholder="Все уровни" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все уровни</SelectItem>
                  <SelectItem value="excellent">Отличное (≥90%)</SelectItem>
                  <SelectItem value="good">Хорошее (70-89%)</SelectItem>
                  <SelectItem value="fair">Удовлетворительное (50-69%)</SelectItem>
                  <SelectItem value="poor">Низкое (&lt;50%)</SelectItem>
                  <SelectItem value="no-quality">Без оценки</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Дата обновления</Label>
              <Select value={selectedDateRange} onValueChange={(value) => {
                setSelectedDateRange(value)
                setCurrentPage(1)
              }}>
                <SelectTrigger>
                  <SelectValue placeholder="Все даты" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Все даты</SelectItem>
                  <SelectItem value="today">Сегодня</SelectItem>
                  <SelectItem value="week">За последнюю неделю</SelectItem>
                  <SelectItem value="month">За последний месяц</SelectItem>
                  <SelectItem value="quarter">За последний квартал</SelectItem>
                  <SelectItem value="year">За последний год</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Действия</Label>
              <div className="space-y-2">
                <Button 
                  onClick={() => {
                    fetchCounterparties()
                    if (selectedProject !== 'all') {
                      fetchStats()
                    }
                  }} 
                  variant="outline" 
                  className="w-full"
                  disabled={isLoading || isLoadingStats}
                >
                  {isLoading || isLoadingStats ? (
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  ) : (
                    <RefreshCw className="h-4 w-4 mr-2" />
                  )}
                  Обновить
                </Button>
                {selectedProject !== 'all' && (
                  <>
                    <Button 
                      onClick={handleLoadDuplicates} 
                      variant="outline" 
                      className="w-full"
                      disabled={isLoadingDuplicates}
                    >
                      {isLoadingDuplicates ? (
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      ) : (
                        <Users className="h-4 w-4 mr-2" />
                      )}
                      Дубликаты
                    </Button>
                    {selectedIds.size > 0 && (
                      <div className="space-y-2">
                        <Button 
                          onClick={handleBulkEnrichClick} 
                          variant="default" 
                          className="w-full"
                          disabled={isBulkEnriching}
                        >
                          {isBulkEnriching ? (
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                          ) : (
                            <Sparkles className="h-4 w-4 mr-2" />
                          )}
                          Обогатить выбранные ({selectedIds.size})
                        </Button>
                        {isBulkEnriching && bulkEnrichProgress.total > 0 && (
                          <div className="space-y-1">
                            <div className="flex items-center justify-between text-xs text-muted-foreground">
                              <span>Обработка контрагентов...</span>
                              <span>
                                {bulkEnrichProgress.current} / {bulkEnrichProgress.total}
                              </span>
                            </div>
                            <Progress 
                              value={(bulkEnrichProgress.current / bulkEnrichProgress.total) * 100} 
                              className="h-2"
                            />
                          </div>
                        )}
                      </div>
                    )}
                    <div className="space-y-2">
                      <div className="flex gap-2">
                        <Button 
                          onClick={() => handleExport('json')} 
                          variant="outline" 
                          size="sm"
                          className="flex-1"
                          disabled={isExporting}
                        >
                          {isExporting ? (
                            <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                          ) : (
                            <Download className="h-4 w-4 mr-2" />
                          )}
                          JSON
                        </Button>
                        <Button 
                          onClick={() => handleExport('csv')} 
                          variant="outline" 
                          size="sm"
                          className="flex-1"
                          disabled={isExporting}
                        >
                          <Download className="h-4 w-4 mr-2" />
                          CSV
                        </Button>
                      </div>
                      {selectedIds.size > 0 && (
                        <Button 
                          onClick={async () => {
                            const selected = sortedCounterparties.filter(cp => selectedIds.has(cp.id))
                            const dataStr = JSON.stringify(selected, null, 2)
                            const blob = new Blob([dataStr], { type: 'application/json' })
                            const url = URL.createObjectURL(blob)
                            const a = document.createElement('a')
                            a.href = url
                            a.download = `counterparties_selected_${new Date().toISOString().split('T')[0]}.json`
                            a.click()
                            URL.revokeObjectURL(url)
                            toast.success(`Экспортировано ${selectedIds.size} контрагентов`)
                          }}
                          variant="outline" 
                          size="sm"
                          className="w-full"
                        >
                          <Download className="h-4 w-4 mr-2" />
                          Экспорт выбранных ({selectedIds.size})
                        </Button>
                      )}
                    </div>
                  </>
                )}
              </div>
            </div>
          </div>
        </CardContent>
        </Card>
      </FadeIn>

      {/* Selected Counterparties Info */}
      {selectedIds.size > 0 && (
        <Alert className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
          <div className="flex items-start sm:items-center gap-2 flex-1">
            <AlertCircle className="h-4 w-4 mt-0.5 sm:mt-0" />
            <AlertDescription className="flex-1 text-sm sm:text-base">
              <div className="flex flex-col sm:flex-row sm:items-center gap-1 sm:gap-2">
                <span>Выбрано контрагентов: <strong>{selectedIds.size}</strong></span>
                {sortedCounterparties.filter(cp => selectedIds.has(cp.id)).length > 0 && (
                  <span className="hidden sm:inline">•</span>
                )}
                {sortedCounterparties.filter(cp => selectedIds.has(cp.id)).length > 0 && (
                  <div className="flex flex-wrap gap-x-2 gap-y-1 text-xs sm:text-sm">
                    <span>Производителей: {sortedCounterparties.filter(cp => selectedIds.has(cp.id) && cp.subcategory === 'производитель').length}</span>
                    <span>•</span>
                    <span>С ИНН: {sortedCounterparties.filter(cp => selectedIds.has(cp.id) && cp.tax_id && cp.tax_id !== '').length}</span>
                    <span>•</span>
                    <span>Нормализовано: {sortedCounterparties.filter(cp => selectedIds.has(cp.id) && cp.source_enrichment && cp.source_enrichment !== '').length}</span>
                  </div>
                )}
              </div>
            </AlertDescription>
          </div>
          <div className="flex gap-2 w-full sm:w-auto">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setSelectedIds(new Set())}
              className="flex-1 sm:flex-initial"
            >
              Снять выбор
            </Button>
          </div>
        </Alert>
      )}

      {/* Statistics */}
      <StaggerContainer className="grid grid-cols-1 md:grid-cols-5 gap-4">
        <StaggerItem>
          <motion.div whileHover={{ scale: 1.05 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardContent className="pt-6">
                {isLoading ? (
                  <div className="flex items-center gap-2">
                    <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    <span className="text-sm text-muted-foreground">Загрузка...</span>
                  </div>
                ) : (
                  <>
                    <div className="text-2xl font-bold">{totalCount}</div>
                    <p className="text-xs text-muted-foreground">Всего контрагентов</p>
                  </>
                )}
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>
        <StaggerItem>
          <motion.div whileHover={{ scale: 1.05 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardContent className="pt-6">
                {isLoadingStats && selectedProject !== 'all' ? (
                  <div className="flex items-center gap-2">
                    <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    <span className="text-sm text-muted-foreground">Загрузка...</span>
                  </div>
                ) : (
                  <>
                    <div className="text-2xl font-bold flex items-center gap-2">
                      <Factory className="h-5 w-5 text-orange-500" />
                      {selectedProject !== 'all' && stats.manufacturers_count !== undefined
                        ? stats.manufacturers_count
                        : counterparties.filter(cp => cp.subcategory === 'производитель').length}
                    </div>
                    <p className="text-xs text-muted-foreground">Производителей</p>
                  </>
                )}
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>
        <StaggerItem>
          <motion.div whileHover={{ scale: 1.05 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardContent className="pt-6">
                {isLoadingStats && selectedProject !== 'all' ? (
                  <div className="flex items-center gap-2">
                    <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
                    <span className="text-sm text-muted-foreground">Загрузка...</span>
                  </div>
                ) : (
                  <>
                    <div className="text-2xl font-bold">
                      {selectedProject !== 'all' && stats.enriched !== undefined
                        ? stats.enriched
                        : counterparties.filter(cp => cp.enrichment_applied).length}
                    </div>
                    <p className="text-xs text-muted-foreground">С дозаполнением</p>
                  </>
                )}
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>
        <StaggerItem>
          <motion.div whileHover={{ scale: 1.05 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardContent className="pt-6">
                <div className="text-2xl font-bold">
                  {counterparties.filter(cp => cp.tax_id && cp.tax_id !== '').length}
                </div>
                <p className="text-xs text-muted-foreground">С ИНН</p>
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>
        <StaggerItem>
          <motion.div whileHover={{ scale: 1.05 }} transition={{ type: "spring", stiffness: 300 }}>
            <Card>
              <CardContent className="pt-6">
                <div className="text-2xl font-bold">
                  {counterparties.filter(cp => cp.source_enrichment && cp.source_enrichment !== '').length}
                </div>
                <p className="text-xs text-muted-foreground">Нормализовано</p>
              </CardContent>
            </Card>
          </motion.div>
        </StaggerItem>
      </StaggerContainer>

      {/* Analytics */}
      {selectedProject !== 'all' && (
        <CounterpartyAnalytics stats={stats} isLoading={isLoadingStats} />
      )}

      {/* Error */}
      {error && (
        <Alert variant="destructive" className="relative">
          <AlertCircle className="h-4 w-4" />
          <AlertDescription className="pr-8">{error}</AlertDescription>
          <Button
            variant="ghost"
            size="icon"
            className="absolute top-2 right-2 h-6 w-6"
            onClick={() => setError(null)}
          >
            <X className="h-4 w-4" />
          </Button>
        </Alert>
      )}

      {/* Table */}
      {isLoading ? (
        <LoadingState />
      ) : sortedCounterparties.length === 0 ? (
        <Card>
          <CardContent className="pt-6">
            <EmptyState
              icon={Building2}
              title="Контрагенты не найдены"
              description={
                debouncedSearchQuery || selectedEnrichment !== 'all' || selectedSubcategory !== 'all' || selectedQuality !== 'all' || selectedDateRange !== 'all'
                  ? 'Попробуйте изменить фильтры поиска или очистить их'
                  : selectedProject === 'all'
                  ? 'Выберите проект для просмотра контрагентов'
                  : 'В выбранном проекте пока нет контрагентов. Загрузите базу данных или запустите нормализацию.'
              }
              action={
                debouncedSearchQuery || selectedEnrichment !== 'all' || selectedSubcategory !== 'all' || selectedQuality !== 'all' || selectedDateRange !== 'all'
                  ? {
                      label: 'Очистить фильтры',
                      onClick: () => {
                        setSearchQuery('')
                        setSelectedEnrichment('all')
                        setSelectedSubcategory('all')
                        setSelectedQuality('all')
                        setSelectedDateRange('all')
                        setCurrentPage(1)
                      },
                    }
                  : undefined
              }
            />
          </CardContent>
        </Card>
      ) : (
        <>
          <Card>
            <CardContent className="p-0">
              {/* Desktop Table View */}
              <div className="hidden md:block overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr className="border-b">
                      <th className="text-left p-4 font-medium w-12">
                        <input
                          type="checkbox"
                          checked={sortedCounterparties.length > 0 && selectedIds.size === sortedCounterparties.length}
                          onChange={(e) => handleSelectAll(e.target.checked)}
                          className="rounded"
                        />
                      </th>
                      <th 
                        className="text-left p-4 font-medium cursor-pointer hover:bg-muted/50 transition-colors select-none"
                        onClick={() => handleSort('name')}
                      >
                        <div className="flex items-center gap-2">
                          Название
                          {getSortIcon('name')}
                        </div>
                      </th>
                      <th 
                        className="text-left p-4 font-medium cursor-pointer hover:bg-muted/50 transition-colors select-none"
                        onClick={() => handleSort('tax_id')}
                      >
                        <div className="flex items-center gap-2">
                          ИНН/БИН
                          {getSortIcon('tax_id')}
                        </div>
                      </th>
                      <th 
                        className="text-left p-4 font-medium cursor-pointer hover:bg-muted/50 transition-colors select-none"
                        onClick={() => handleSort('address')}
                      >
                        <div className="flex items-center gap-2">
                          Адрес
                          {getSortIcon('address')}
                        </div>
                      </th>
                      <th className="text-left p-4 font-medium">Контакты</th>
                      <th 
                        className="text-left p-4 font-medium cursor-pointer hover:bg-muted/50 transition-colors select-none"
                        onClick={() => handleSort('source')}
                      >
                        <div className="flex items-center gap-2">
                          Источник / Качество
                          {getSortIcon('source')}
                        </div>
                      </th>
                      <th 
                        className="text-left p-4 font-medium cursor-pointer hover:bg-muted/50 transition-colors select-none"
                        onClick={() => handleSort('updated')}
                      >
                        <div className="flex items-center gap-2">
                          Обновлено
                          {getSortIcon('updated')}
                        </div>
                      </th>
                      <th className="text-left p-4 font-medium">Действия</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sortedCounterparties.map((cp, index) => (
                      <motion.tr
                        key={cp.id}
                        initial={{ opacity: 0, y: 10 }}
                        animate={{ opacity: 1, y: 0 }}
                        transition={{ delay: Math.min(index * 0.02, 0.5), duration: 0.2 }}
                        className="border-b hover:bg-muted/50 cursor-pointer"
                        onClick={(e) => {
                          // Не открываем просмотр при клике на чекбокс или кнопки действий
                          const target = e.target as HTMLElement
                          if (!target.closest('input[type="checkbox"]') && !target.closest('button')) {
                            setViewingCounterparty(cp)
                          }
                        }}
                      >
                        <td className="p-4">
                          <input
                            type="checkbox"
                            checked={selectedIds.has(cp.id)}
                            onChange={(e) => handleToggleSelection(cp.id, e.target.checked)}
                            className="rounded"
                          />
                        </td>
                        <td className="p-4">
                          <div className="flex items-center gap-2">
                            <div className="font-medium">{cp.normalized_name}</div>
                            {cp.subcategory === 'производитель' && (
                              <Badge variant="secondary" className="bg-orange-100 text-orange-800 border-orange-300">
                                <Factory className="h-3 w-3 mr-1" />
                                Производитель
                              </Badge>
                            )}
                          </div>
                          {cp.source_name !== cp.normalized_name && (
                            <div className="text-sm text-muted-foreground">
                              {cp.source_name}
                            </div>
                          )}
                        </td>
                        <td className="p-4">
                          <div className="space-y-1">
                            {cp.tax_id && (
                              <div className="flex items-center gap-2 group">
                                <span>ИНН: {cp.tax_id}</span>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity"
                                  onClick={() => handleCopyToClipboard(cp.tax_id, 'ИНН')}
                                  title="Копировать ИНН"
                                >
                                  <Copy className="h-3 w-3" />
                                </Button>
                              </div>
                            )}
                            {cp.bin && (
                              <div className="flex items-center gap-2 group">
                                <span>БИН: {cp.bin}</span>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity"
                                  onClick={() => handleCopyToClipboard(cp.bin, 'БИН')}
                                  title="Копировать БИН"
                                >
                                  <Copy className="h-3 w-3" />
                                </Button>
                              </div>
                            )}
                            {cp.kpp && <div className="text-sm text-muted-foreground">КПП: {cp.kpp}</div>}
                          </div>
                        </td>
                        <td className="p-4">
                          <div className="space-y-1">
                            {cp.legal_address && (
                              <div className="flex items-start gap-2 group">
                                <span className="text-sm flex-1">{cp.legal_address}</span>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity flex-shrink-0"
                                  onClick={() => handleCopyToClipboard(cp.legal_address, 'Адрес')}
                                  title="Копировать адрес"
                                >
                                  <Copy className="h-3 w-3" />
                                </Button>
                              </div>
                            )}
                            {cp.postal_address && cp.postal_address !== cp.legal_address && (
                              <div className="text-sm text-muted-foreground">
                                Почтовый: {cp.postal_address}
                              </div>
                            )}
                          </div>
                        </td>
                        <td className="p-4">
                          <div className="space-y-1">
                            {cp.contact_phone && (
                              <div className="text-sm flex items-center gap-1 group">
                                <Phone className="h-3 w-3" />
                                <span className="flex-1">{cp.contact_phone}</span>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity"
                                  onClick={() => handleCopyToClipboard(cp.contact_phone, 'Телефон')}
                                  title="Копировать телефон"
                                >
                                  <Copy className="h-3 w-3" />
                                </Button>
                              </div>
                            )}
                            {cp.contact_email && (
                              <div className="text-sm flex items-center gap-1 group">
                                <Mail className="h-3 w-3" />
                                <span className="flex-1">{cp.contact_email}</span>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-5 w-5 opacity-0 group-hover:opacity-100 transition-opacity"
                                  onClick={() => handleCopyToClipboard(cp.contact_email, 'Email')}
                                  title="Копировать email"
                                >
                                  <Copy className="h-3 w-3" />
                                </Button>
                              </div>
                            )}
                            {cp.contact_person && (
                              <div className="text-sm">{cp.contact_person}</div>
                            )}
                          </div>
                        </td>
                        <td className="p-4">
                          <div className="space-y-2">
                            {getEnrichmentBadge(cp.source_enrichment)}
                            {cp.quality_score !== undefined && cp.quality_score !== null && (
                              <div className="flex items-center gap-2">
                                <div className="flex-1 bg-muted rounded-full h-2 relative overflow-hidden">
                                  <div
                                    className={`h-2 rounded-full transition-all ${
                                      cp.quality_score >= 0.9
                                        ? 'bg-green-500'
                                        : cp.quality_score >= 0.7
                                        ? 'bg-blue-500'
                                        : cp.quality_score >= 0.5
                                        ? 'bg-yellow-500'
                                        : 'bg-red-500'
                                    }`}
                                    style={{ width: `${cp.quality_score * 100}%` }}
                                  />
                                </div>
                                <span className="text-xs text-muted-foreground min-w-[3rem] text-right">
                                  {Math.round(cp.quality_score * 100)}%
                                </span>
                              </div>
                            )}
                          </div>
                        </td>
                        <td className="p-4">
                          <div className="text-sm" title={new Date(cp.updated_at).toLocaleString('ru-RU')}>
                            {new Date(cp.updated_at).toLocaleDateString('ru-RU', {
                              day: '2-digit',
                              month: '2-digit',
                              year: 'numeric'
                            })}
                          </div>
                          <div className="text-xs text-muted-foreground">
                            {new Date(cp.updated_at).toLocaleTimeString('ru-RU', {
                              hour: '2-digit',
                              minute: '2-digit'
                            })}
                          </div>
                        </td>
                        <td className="p-4">
                          <div className="flex items-center gap-2">
                            <motion.div whileHover={{ scale: 1.1 }} whileTap={{ scale: 0.9 }}>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => setViewingCounterparty(cp)}
                                title="Быстрый просмотр"
                              >
                                <Eye className="h-4 w-4" />
                              </Button>
                            </motion.div>
                            <motion.div whileHover={{ scale: 1.1 }} whileTap={{ scale: 0.9 }}>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleEdit(cp)}
                                title="Редактировать"
                              >
                                <Edit className="h-4 w-4" />
                              </Button>
                            </motion.div>
                            {(cp.tax_id || cp.bin) && (
                              <motion.div whileHover={{ scale: 1.1 }} whileTap={{ scale: 0.9 }}>
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => handleEnrich(cp)}
                                  disabled={isEnriching === cp.id}
                                  title="Обогатить данные"
                                >
                                  {isEnriching === cp.id ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  ) : (
                                    <Sparkles className="h-4 w-4" />
                                  )}
                                </Button>
                              </motion.div>
                            )}
                          </div>
                        </td>
                      </motion.tr>
                    ))}
                  </tbody>
                </table>
              </div>
              
              {/* Mobile Card View */}
              <div className="md:hidden space-y-4 p-4">
                {sortedCounterparties.map((cp, index) => (
                  <motion.div
                    key={cp.id}
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: Math.min(index * 0.02, 0.5), duration: 0.2 }}
                    className="border rounded-lg p-4 space-y-3"
                    onClick={() => setViewingCounterparty(cp)}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-1">
                          <input
                            type="checkbox"
                            checked={selectedIds.has(cp.id)}
                            onChange={(e) => {
                              e.stopPropagation()
                              handleToggleSelection(cp.id, e.target.checked)
                            }}
                            className="rounded"
                            onClick={(e) => e.stopPropagation()}
                          />
                          <div className="font-medium">{cp.normalized_name}</div>
                          {cp.subcategory === 'производитель' && (
                            <Badge variant="secondary" className="bg-orange-100 text-orange-800 border-orange-300">
                              <Factory className="h-3 w-3 mr-1" />
                              Производитель
                            </Badge>
                          )}
                        </div>
                        {cp.source_name !== cp.normalized_name && (
                          <div className="text-sm text-muted-foreground ml-6">
                            {cp.source_name}
                          </div>
                        )}
                      </div>
                      <div className="flex gap-1" onClick={(e) => e.stopPropagation()}>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => setViewingCounterparty(cp)}
                          title="Быстрый просмотр"
                        >
                          <Eye className="h-4 w-4" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEdit(cp)}
                          title="Редактировать"
                        >
                          <Edit className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                    
                    <div className="grid grid-cols-2 gap-2 text-sm ml-6">
                      {cp.tax_id && (
                        <div className="flex items-center gap-1">
                          <span className="text-muted-foreground">ИНН:</span>
                          <span>{cp.tax_id}</span>
                        </div>
                      )}
                      {cp.bin && (
                        <div className="flex items-center gap-1">
                          <span className="text-muted-foreground">БИН:</span>
                          <span>{cp.bin}</span>
                        </div>
                      )}
                      {cp.legal_address && (
                        <div className="col-span-2 flex items-start gap-1">
                          <MapPin className="h-3 w-3 mt-0.5 text-muted-foreground" />
                          <span className="line-clamp-1">{cp.legal_address}</span>
                        </div>
                      )}
                      {cp.quality_score !== undefined && cp.quality_score !== null && (
                        <div className="col-span-2 flex items-center gap-2">
                          <span className="text-muted-foreground">Качество:</span>
                          <div className="flex-1 bg-muted rounded-full h-2">
                            <div
                              className={`h-2 rounded-full ${
                                cp.quality_score >= 0.9
                                  ? 'bg-green-500'
                                  : cp.quality_score >= 0.7
                                  ? 'bg-blue-500'
                                  : cp.quality_score >= 0.5
                                  ? 'bg-yellow-500'
                                  : 'bg-red-500'
                              }`}
                              style={{ width: `${cp.quality_score * 100}%` }}
                            />
                          </div>
                          <span className="text-xs font-medium min-w-[3rem] text-right">
                            {Math.round(cp.quality_score * 100)}%
                          </span>
                        </div>
                      )}
                    </div>
                  </motion.div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Pagination */}
          {totalPages > 1 && (
            <div className="flex flex-col sm:flex-row items-center justify-between gap-4">
              <div className="flex items-center gap-4">
                <div className="text-sm text-muted-foreground">
                  Показано {(currentPage - 1) * limit + 1} - {Math.min(currentPage * limit, totalCount)} из {totalCount}
                </div>
                <div className="flex items-center gap-2">
                  <Label className="text-xs">На странице:</Label>
                  <Select value={limit.toString()} onValueChange={(value) => {
                    setLimit(parseInt(value, 10))
                    setCurrentPage(1)
                  }}>
                    <SelectTrigger className="w-20 h-8">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="10">10</SelectItem>
                      <SelectItem value="20">20</SelectItem>
                      <SelectItem value="50">50</SelectItem>
                      <SelectItem value="100">100</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                  disabled={currentPage === 1}
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <div className="flex items-center gap-1">
                  {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                    let pageNum: number
                    if (totalPages <= 5) {
                      pageNum = i + 1
                    } else if (currentPage <= 3) {
                      pageNum = i + 1
                    } else if (currentPage >= totalPages - 2) {
                      pageNum = totalPages - 4 + i
                    } else {
                      pageNum = currentPage - 2 + i
                    }
                    return (
                      <Button
                        key={pageNum}
                        variant={currentPage === pageNum ? "default" : "outline"}
                        size="sm"
                        onClick={() => setCurrentPage(pageNum)}
                      >
                        {pageNum}
                      </Button>
                    )
                  })}
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                  disabled={currentPage === totalPages}
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          )}
        </>
      )}

      {/* Edit Dialog */}
      <Dialog open={!!editingCounterparty} onOpenChange={(open) => {
        if (!open) {
          setEditingCounterparty(null)
          setEditForm({})
        }
      }}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Редактирование контрагента</DialogTitle>
            <DialogDescription>
              Измените данные контрагента и нажмите "Сохранить"
            </DialogDescription>
          </DialogHeader>
          <div className="grid grid-cols-2 gap-4 py-4">
            <div className="space-y-2">
              <Label>Название</Label>
              <Input
                value={editForm.normalized_name || ''}
                onChange={(e) => setEditForm({ ...editForm, normalized_name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>ИНН</Label>
              <Input
                value={editForm.tax_id || ''}
                onChange={(e) => setEditForm({ ...editForm, tax_id: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>КПП</Label>
              <Input
                value={editForm.kpp || ''}
                onChange={(e) => setEditForm({ ...editForm, kpp: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>БИН</Label>
              <Input
                value={editForm.bin || ''}
                onChange={(e) => setEditForm({ ...editForm, bin: e.target.value })}
              />
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Юридический адрес</Label>
              <Textarea
                value={editForm.legal_address || ''}
                onChange={(e) => setEditForm({ ...editForm, legal_address: e.target.value })}
                rows={2}
              />
            </div>
            <div className="space-y-2 col-span-2">
              <Label>Почтовый адрес</Label>
              <Textarea
                value={editForm.postal_address || ''}
                onChange={(e) => setEditForm({ ...editForm, postal_address: e.target.value })}
                rows={2}
              />
            </div>
            <div className="space-y-2">
              <Label>Телефон</Label>
              <Input
                value={editForm.contact_phone || ''}
                onChange={(e) => setEditForm({ ...editForm, contact_phone: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Email</Label>
              <Input
                type="email"
                value={editForm.contact_email || ''}
                onChange={(e) => setEditForm({ ...editForm, contact_email: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Контактное лицо</Label>
              <Input
                value={editForm.contact_person || ''}
                onChange={(e) => setEditForm({ ...editForm, contact_person: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Организационно-правовая форма</Label>
              <Input
                value={editForm.legal_form || ''}
                onChange={(e) => setEditForm({ ...editForm, legal_form: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Банк</Label>
              <Input
                value={editForm.bank_name || ''}
                onChange={(e) => setEditForm({ ...editForm, bank_name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Расчетный счет</Label>
              <Input
                value={editForm.bank_account || ''}
                onChange={(e) => setEditForm({ ...editForm, bank_account: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Корреспондентский счет</Label>
              <Input
                value={editForm.correspondent_account || ''}
                onChange={(e) => setEditForm({ ...editForm, correspondent_account: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>БИК</Label>
              <Input
                value={editForm.bik || ''}
                onChange={(e) => setEditForm({ ...editForm, bik: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <Label>Оценка качества</Label>
              <Input
                type="number"
                step="0.01"
                min="0"
                max="1"
                value={editForm.quality_score || 0}
                onChange={(e) => setEditForm({ ...editForm, quality_score: parseFloat(e.target.value) || 0 })}
              />
            </div>
            <div className="space-y-2">
              <Label>Источник нормализации</Label>
              <Select
                value={editForm.source_enrichment || 'none'}
                onValueChange={(value) => setEditForm({ ...editForm, source_enrichment: value === 'none' ? '' : value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Выберите источник" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Не нормализован</SelectItem>
                  <SelectItem value="Adata.kz">Adata.kz</SelectItem>
                  <SelectItem value="Dadata.ru">Dadata.ru</SelectItem>
                  <SelectItem value="gisp.gov.ru">gisp.gov.ru</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Подкатегория</Label>
              <Select
                value={editForm.subcategory || 'none'}
                onValueChange={(value) => setEditForm({ ...editForm, subcategory: value === 'none' ? '' : value })}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Выберите подкатегорию" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">Без подкатегории</SelectItem>
                  <SelectItem value="производитель">Производитель</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => {
              setEditingCounterparty(null)
              setEditForm({})
            }}>
              Отмена
            </Button>
            <Button onClick={handleSave} disabled={isSaving}>
              {isSaving ? (
                <>
                  <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
                  Сохранение...
                </>
              ) : (
                <>
                  <Save className="h-4 w-4 mr-2" />
                  Сохранить
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Duplicates Dialog */}
      <Dialog open={showDuplicates} onOpenChange={setShowDuplicates}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>Дубликаты контрагентов</DialogTitle>
            <DialogDescription>
              Найдено {duplicates.length} групп дубликатов. Выберите мастер-запись и объедините дубликаты.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            {duplicates.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">
                Дубликаты не найдены
              </div>
            ) : (
              duplicates.map((group: any, groupIndex: number) => (
                <Card key={groupIndex}>
                  <CardHeader>
                    <CardTitle className="text-lg">
                      Группа {groupIndex + 1}: ИНН/БИН {group.tax_id} ({group.count} записей)
                    </CardTitle>
                  </CardHeader>
                  <CardContent>
                    <div className="space-y-3">
                      {group.items.map((item: NormalizedCounterparty, itemIndex: number) => (
                        <div
                          key={item.id}
                          className="flex items-start gap-4 p-3 border rounded-lg hover:bg-muted/50"
                        >
                          <input
                            type="radio"
                            name={`master-${groupIndex}`}
                            id={`master-${groupIndex}-${item.id}`}
                            value={item.id}
                            defaultChecked={itemIndex === 0}
                            className="mt-1"
                          />
                          <label
                            htmlFor={`master-${groupIndex}-${item.id}`}
                            className="flex-1 cursor-pointer"
                          >
                            <div className="font-medium">{item.normalized_name}</div>
                            <div className="text-sm text-muted-foreground mt-1">
                              {item.legal_address && <div>Адрес: {item.legal_address}</div>}
                              {item.contact_phone && <div>Телефон: {item.contact_phone}</div>}
                              {item.contact_email && <div>Email: {item.contact_email}</div>}
                            </div>
                          </label>
                        </div>
                      ))}
                      <Button
                        onClick={() => {
                          const masterRadio = document.querySelector(
                            `input[name="master-${groupIndex}"]:checked`
                          ) as HTMLInputElement
                          if (masterRadio) {
                            const masterId = parseInt(masterRadio.value)
                            const mergeIds = group.items
                              .map((item: NormalizedCounterparty) => item.id)
                              .filter((id: number) => id !== masterId)
                            handleMergeDuplicates(group.tax_id, masterId, mergeIds)
                          }
                        }}
                        className="w-full"
                        variant="default"
                      >
                        <Copy className="h-4 w-4 mr-2" />
                        Объединить дубликаты
                      </Button>
                    </div>
                  </CardContent>
                </Card>
              ))
            )}
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowDuplicates(false)}>
              Закрыть
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Диалог быстрого просмотра контрагента */}
      <Dialog open={!!viewingCounterparty} onOpenChange={(open) => !open && setViewingCounterparty(null)}>
        <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Building2 className="h-5 w-5" />
              {viewingCounterparty?.normalized_name}
            </DialogTitle>
            <DialogDescription>
              Детальная информация о контрагенте
            </DialogDescription>
          </DialogHeader>
          {viewingCounterparty && (
            <div className="space-y-4">
              {/* Основная информация */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Основная информация</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label className="text-xs text-muted-foreground">Название</Label>
                      <p className="text-sm font-medium">{viewingCounterparty.normalized_name}</p>
                      {viewingCounterparty.source_name !== viewingCounterparty.normalized_name && (
                        <p className="text-xs text-muted-foreground mt-1">
                          Исходное: {viewingCounterparty.source_name}
                        </p>
                      )}
                    </div>
                    {viewingCounterparty.subcategory && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Категория</Label>
                        <div className="mt-1">
                          <Badge variant="secondary" className="bg-orange-100 text-orange-800 border-orange-300">
                            <Factory className="h-3 w-3 mr-1" />
                            {viewingCounterparty.subcategory}
                          </Badge>
                        </div>
                      </div>
                    )}
                  </div>
                  <div className="grid grid-cols-2 gap-4">
                    {viewingCounterparty.tax_id && (
                      <div>
                        <Label className="text-xs text-muted-foreground">ИНН</Label>
                        <p className="text-sm font-medium">{viewingCounterparty.tax_id}</p>
                      </div>
                    )}
                    {viewingCounterparty.bin && (
                      <div>
                        <Label className="text-xs text-muted-foreground">БИН</Label>
                        <p className="text-sm font-medium">{viewingCounterparty.bin}</p>
                      </div>
                    )}
                    {viewingCounterparty.kpp && (
                      <div>
                        <Label className="text-xs text-muted-foreground">КПП</Label>
                        <p className="text-sm font-medium">{viewingCounterparty.kpp}</p>
                      </div>
                    )}
                    {viewingCounterparty.legal_form && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Организационно-правовая форма</Label>
                        <p className="text-sm font-medium">{viewingCounterparty.legal_form}</p>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* Адреса */}
              {(viewingCounterparty.legal_address || viewingCounterparty.postal_address) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <MapPin className="h-4 w-4" />
                      Адреса
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {viewingCounterparty.legal_address && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Юридический адрес</Label>
                        <p className="text-sm">{viewingCounterparty.legal_address}</p>
                      </div>
                    )}
                    {viewingCounterparty.postal_address && viewingCounterparty.postal_address !== viewingCounterparty.legal_address && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Почтовый адрес</Label>
                        <p className="text-sm">{viewingCounterparty.postal_address}</p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}

              {/* Контакты */}
              {(viewingCounterparty.contact_phone || viewingCounterparty.contact_email || viewingCounterparty.contact_person) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <Users className="h-4 w-4" />
                      Контакты
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    {viewingCounterparty.contact_phone && (
                      <div className="flex items-center gap-2">
                        <Phone className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm">{viewingCounterparty.contact_phone}</span>
                      </div>
                    )}
                    {viewingCounterparty.contact_email && (
                      <div className="flex items-center gap-2">
                        <Mail className="h-4 w-4 text-muted-foreground" />
                        <span className="text-sm">{viewingCounterparty.contact_email}</span>
                      </div>
                    )}
                    {viewingCounterparty.contact_person && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Контактное лицо</Label>
                        <p className="text-sm">{viewingCounterparty.contact_person}</p>
                      </div>
                    )}
                  </CardContent>
                </Card>
              )}

              {/* Банковские реквизиты */}
              {(viewingCounterparty.bank_name || viewingCounterparty.bank_account || viewingCounterparty.bik) && (
                <Card>
                  <CardHeader>
                    <CardTitle className="text-base flex items-center gap-2">
                      <CreditCard className="h-4 w-4" />
                      Банковские реквизиты
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="space-y-3">
                    <div className="grid grid-cols-2 gap-4">
                      {viewingCounterparty.bank_name && (
                        <div>
                          <Label className="text-xs text-muted-foreground">Банк</Label>
                          <p className="text-sm">{viewingCounterparty.bank_name}</p>
                        </div>
                      )}
                      {viewingCounterparty.bik && (
                        <div>
                          <Label className="text-xs text-muted-foreground">БИК</Label>
                          <p className="text-sm">{viewingCounterparty.bik}</p>
                        </div>
                      )}
                      {viewingCounterparty.bank_account && (
                        <div>
                          <Label className="text-xs text-muted-foreground">Расчетный счет</Label>
                          <p className="text-sm">{viewingCounterparty.bank_account}</p>
                        </div>
                      )}
                      {viewingCounterparty.correspondent_account && (
                        <div>
                          <Label className="text-xs text-muted-foreground">Корреспондентский счет</Label>
                          <p className="text-sm">{viewingCounterparty.correspondent_account}</p>
                        </div>
                      )}
                    </div>
                  </CardContent>
                </Card>
              )}

              {/* Метаданные */}
              <Card>
                <CardHeader>
                  <CardTitle className="text-base">Метаданные</CardTitle>
                </CardHeader>
                <CardContent className="space-y-3">
                  <div className="grid grid-cols-2 gap-4">
                    <div>
                      <Label className="text-xs text-muted-foreground">Источник нормализации</Label>
                      <div className="mt-1">
                        {getEnrichmentBadge(viewingCounterparty.source_enrichment)}
                      </div>
                    </div>
                    {viewingCounterparty.quality_score !== undefined && viewingCounterparty.quality_score !== null && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Оценка качества</Label>
                        <div className="mt-1 flex items-center gap-2">
                          <div className="flex-1 bg-muted rounded-full h-2">
                            <div
                              className={`h-2 rounded-full ${
                                viewingCounterparty.quality_score >= 0.9
                                  ? 'bg-green-500'
                                  : viewingCounterparty.quality_score >= 0.7
                                  ? 'bg-blue-500'
                                  : viewingCounterparty.quality_score >= 0.5
                                  ? 'bg-yellow-500'
                                  : 'bg-red-500'
                              }`}
                              style={{ width: `${viewingCounterparty.quality_score * 100}%` }}
                            />
                          </div>
                          <span className="text-sm font-medium min-w-[3rem] text-right">
                            {Math.round(viewingCounterparty.quality_score * 100)}%
                          </span>
                        </div>
                      </div>
                    )}
                    <div>
                      <Label className="text-xs text-muted-foreground">Создано</Label>
                      <p className="text-sm">
                        {new Date(viewingCounterparty.created_at).toLocaleString('ru-RU')}
                      </p>
                    </div>
                    <div>
                      <Label className="text-xs text-muted-foreground">Обновлено</Label>
                      <p className="text-sm">
                        {new Date(viewingCounterparty.updated_at).toLocaleString('ru-RU')}
                      </p>
                    </div>
                    {viewingCounterparty.source_reference && (
                      <div>
                        <Label className="text-xs text-muted-foreground">Ссылка на источник</Label>
                        <p className="text-sm font-mono text-xs break-all">{viewingCounterparty.source_reference}</p>
                      </div>
                    )}
                    {viewingCounterparty.source_database && (
                      <div>
                        <Label className="text-xs text-muted-foreground">База данных</Label>
                        <p className="text-sm">{viewingCounterparty.source_database}</p>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setViewingCounterparty(null)}>
              Закрыть
            </Button>
            {viewingCounterparty && (
              <Button onClick={() => {
                setViewingCounterparty(null)
                handleEdit(viewingCounterparty)
              }}>
                <Edit className="h-4 w-4 mr-2" />
                Редактировать
              </Button>
            )}
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Диалог подтверждения массового обогащения */}
      <AlertDialog open={showBulkEnrichConfirm} onOpenChange={setShowBulkEnrichConfirm}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Подтвердите массовое обогащение</AlertDialogTitle>
            <AlertDialogDescription>
              Вы собираетесь обогатить данные для <strong>{selectedIds.size}</strong> контрагентов.
              <br />
              <br />
              Это действие может занять некоторое время. Продолжить?
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isBulkEnriching}>Отмена</AlertDialogCancel>
            <AlertDialogAction onClick={handleBulkEnrich} disabled={isBulkEnriching}>
              {isBulkEnriching ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Обработка...
                </>
              ) : (
                'Обогатить'
              )}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

