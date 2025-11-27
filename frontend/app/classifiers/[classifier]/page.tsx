'use client'

import { useState, useEffect, useCallback, useMemo } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { DatabaseSelector } from '@/components/database-selector'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ClassifierPageSkeleton, ClassifierNodeSkeleton } from '@/components/common/classifier-skeleton'
// import { BackendStatusIndicator } from '@/components/common/backend-status-indicator'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { 
  FileText, 
  Database, 
  ArrowLeft,
  Search,
  BookOpen,
  ChevronRight,
  Home,
  Loader2,
  X,
  Filter,
  Download,
  Maximize2,
  Minimize2,
  Upload,
  AlertCircle,
  Check
} from 'lucide-react'
import Link from 'next/link'
import { Input } from '@/components/ui/input'
import { Separator } from '@/components/ui/separator'
import { motion, AnimatePresence } from 'framer-motion'
import { cn } from '@/lib/utils'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Progress } from '@/components/ui/progress'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

interface ClassifierNode {
  code: string
  name: string
  level: number
  children?: ClassifierNode[]
  has_children?: boolean
  item_count?: number
  parent_code?: string
}

// eslint-disable-next-line @typescript-eslint/no-empty-object-type
type ClassifierDetailPageProps = Record<string, never>

export default function ClassifierDetailPage(_props: ClassifierDetailPageProps) {
  const params = useParams()
  const router = useRouter()
  const classifierType = params.classifier as string
  
  const [selectedDatabase, setSelectedDatabase] = useState<string>('')
  const [hierarchy, setHierarchy] = useState<ClassifierNode[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [searchResults, setSearchResults] = useState<ClassifierNode[]>([])
  const [searching, setSearching] = useState(false)
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [filterLevel, setFilterLevel] = useState<number | null>(null)
  const [loadingNodes, setLoadingNodes] = useState<Set<string>>(new Set())
  const [showLoadDialog, setShowLoadDialog] = useState(false)
  const [loadingFile, setLoadingFile] = useState(false)
  const [filePath, setFilePath] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [loadError, setLoadError] = useState<string | null>(null)
  const [loadSuccess, setLoadSuccess] = useState<string | null>(null)
  const [stats, setStats] = useState<{ total_codes: number; max_level: number } | null>(null)
  const [exporting, setExporting] = useState(false)
  const [copiedCode, setCopiedCode] = useState<string | null>(null)
  const [showClearDialog, setShowClearDialog] = useState(false)
  const [clearing, setClearing] = useState(false)
  const [filePreview, setFilePreview] = useState<string | null>(null)
  const [showFilePreview, setShowFilePreview] = useState(false)
  const [searchInputValue, setSearchInputValue] = useState('')

  const classifierNames: Record<string, { name: string; fullName: string }> = {
    kpved: { name: 'КПВЭД', fullName: 'Классификатор продукции по видам экономической деятельности' },
    okpd2: { name: 'ОКПД2', fullName: 'Общероссийский классификатор продукции по видам экономической деятельности' },
  }

  const classifierInfo = classifierNames[classifierType] || { 
    name: classifierType.toUpperCase(), 
    fullName: `Классификатор ${classifierType.toUpperCase()}` 
  }

  const isValidClassifier = classifierType === 'kpved' || classifierType === 'okpd2'

  useEffect(() => {
    if (!isValidClassifier) {
      router.push('/classifiers')
      return
    }

    // Для ОКПД2 и КПВЭД база данных не обязательна (данные в service.db)
    if (classifierType === 'okpd2' || classifierType === 'kpved') {
      fetchHierarchy()
      fetchStats()
    }
  }, [classifierType, isValidClassifier, router])

  const fetchStats = useCallback(async () => {
    if (classifierType !== 'okpd2' && classifierType !== 'kpved') return

    try {
      const endpoint = classifierType === 'okpd2' 
        ? '/api/okpd2/stats'
        : '/api/kpved/stats'
      
      const response = await fetch(endpoint)
      if (response.ok) {
        const data = await response.json()
        setStats({
          total_codes: data.total_codes || 0,
          max_level: data.max_level || 0,
        })
      } else if (response.status === 404) {
        // При 404 устанавливаем нулевые статистики
        setStats({
          total_codes: 0,
          max_level: 0,
        })
      }
    } catch (err) {
      console.error('Error fetching stats:', err)
      // При ошибке устанавливаем нулевые статистики
      setStats({
        total_codes: 0,
        max_level: 0,
      })
    }
  }, [classifierType])

  const exportHierarchy = useCallback(async (format: 'csv' | 'json') => {
    if (!hierarchy || hierarchy.length === 0) {
      setError('Нет данных для экспорта')
      return
    }

    setExporting(true)
    try {
      const flattenNodes = (nodes: ClassifierNode[], parentPath: string[] = []): Array<{code: string, name: string, level: number, path: string}> => {
        const result: Array<{code: string, name: string, level: number, path: string}> = []
        for (const node of nodes) {
          const currentPath = [...parentPath, node.code]
          result.push({
            code: node.code,
            name: node.name,
            level: node.level,
            path: currentPath.join(' > ')
          })
          if (node.children) {
            result.push(...flattenNodes(node.children, currentPath))
          }
        }
        return result
      }

      const flatData = flattenNodes(hierarchy)
      const fileName = `${classifierType}_classifier_${new Date().toISOString().split('T')[0]}`

      if (format === 'json') {
        const jsonData = JSON.stringify(flatData, null, 2)
        const blob = new Blob([jsonData], { type: 'application/json' })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `${fileName}.json`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
      } else {
        // CSV format
        const headers = ['Код', 'Название', 'Уровень', 'Путь']
        const csvRows = [
          headers.join(','),
          ...flatData.map(row => 
            [
              `"${row.code}"`,
              `"${row.name.replace(/"/g, '""')}"`,
              row.level,
              `"${row.path.replace(/"/g, '""')}"`
            ].join(',')
          )
        ]
        const csvContent = csvRows.join('\n')
        const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8;' })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `${fileName}.csv`
        document.body.appendChild(link)
        link.click()
        document.body.removeChild(link)
        window.URL.revokeObjectURL(url)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка экспорта')
    } finally {
      setExporting(false)
    }
  }, [hierarchy, classifierType])

  const fetchHierarchy = useCallback(async (parentCode?: string) => {
    if (!classifierType) return

    // Для ОКПД2 и КПВЭД база данных не нужна (данные в service.db)
    if (classifierType !== 'okpd2' && classifierType !== 'kpved') return

    setLoading(!parentCode)
    if (parentCode) {
      setLoadingNodes(prev => new Set(prev).add(parentCode))
    }
    setError(null)

    try {
      const endpoint = classifierType === 'okpd2' 
        ? '/api/okpd2/hierarchy'
        : '/api/kpved/hierarchy'
      
      if (parentCode) {
        const urlWithParent = `${endpoint}?parent=${encodeURIComponent(parentCode)}`
        
        const response = await fetch(urlWithParent)
        if (!response.ok) {
          // При 404 просто возвращаем пустой массив дочерних узлов
          if (response.status === 404) {
            setHierarchy(prev => updateNodeChildren(prev, parentCode, []))
            return
          }
          throw new Error('Не удалось загрузить дочерние узлы')
        }
        
        const data = await response.json()
        const children = data.nodes || []
        
        // Обновляем иерархию с новыми дочерними узлами
        setHierarchy(prev => updateNodeChildren(prev, parentCode, children))
      } else {
        const response = await fetch(endpoint)
        console.log(`[Classifier] Fetching hierarchy from ${endpoint}, status: ${response.status}`)
        
        if (!response.ok) {
          // При 404 устанавливаем пустую иерархию вместо ошибки
          if (response.status === 404) {
            console.log('[Classifier] 404 - setting empty hierarchy')
            setHierarchy([])
            return
          }
          
          // Пытаемся получить текст ошибки
          let errorText = ''
          try {
            const errorData = await response.json()
            errorText = errorData.error || JSON.stringify(errorData)
          } catch {
            errorText = await response.text()
          }
          console.error(`[Classifier] Error fetching hierarchy: ${response.status} - ${errorText}`)
          throw new Error(`Не удалось загрузить иерархию классификатора: ${errorText || response.statusText}`)
        }

        const data = await response.json()
        console.log(`[Classifier] Received data:`, { nodes: data.nodes?.length || 0, total: data.total })
        setHierarchy(data.nodes || [])
        
        // Если nodes пустой, но нет ошибки - логируем это
        if (!data.nodes || data.nodes.length === 0) {
          console.warn(`[Classifier] Received empty nodes array. Total in response: ${data.total}`)
        }
      }
    } catch (err) {
      console.error('Error fetching hierarchy:', err)
      const errorMessage = err instanceof Error ? err.message : 'Ошибка загрузки данных'
      
      // Определяем тип ошибки
      const isConnectionError = 
        errorMessage.includes('fetch') || 
        errorMessage.includes('ECONNREFUSED') ||
        errorMessage.includes('Failed to fetch') ||
        errorMessage.includes('NetworkError') ||
        errorMessage.includes('network')
      
      if (isConnectionError) {
        setError('Не удалось подключиться к серверу. Проверьте, что backend запущен и доступен.')
      } else {
        setError(errorMessage)
      }
    } finally {
      setLoading(false)
      if (parentCode) {
        setLoadingNodes(prev => {
          const newSet = new Set(prev)
          newSet.delete(parentCode)
          return newSet
        })
      }
    }
  }, [classifierType])

  // Debounce для поиска - автоматический поиск при вводе с задержкой
  useEffect(() => {
    const timer = setTimeout(() => {
      if (searchInputValue.trim() && searchInputValue !== searchQuery) {
        setSearchQuery(searchInputValue)
        handleSearch(searchInputValue)
      } else if (!searchInputValue.trim()) {
        setSearchQuery('')
        setSearchResults([])
      }
    }, 500) // 500ms debounce delay

    return () => clearTimeout(timer)
  }, [searchInputValue]) // Используем только searchInputValue, чтобы избежать лишних вызовов

  const updateNodeChildren = (nodes: ClassifierNode[], parentCode: string, children: ClassifierNode[]): ClassifierNode[] => {
    return nodes.map(node => {
      if (node.code === parentCode) {
        return { ...node, children }
      }
      if (node.children) {
        return { ...node, children: updateNodeChildren(node.children, parentCode, children) }
      }
      return node
    })
  }

  const handleSearch = useCallback(async (query?: string) => {
    const searchValue = query ?? searchQuery
    if (!searchValue.trim()) {
      setSearchResults([])
      return
    }

    if (classifierType !== 'okpd2' && classifierType !== 'kpved') {
      setError('Выберите классификатор для поиска')
      return
    }

    setSearching(true)
    setError(null)

    try {
      const endpoint = classifierType === 'okpd2'
        ? '/api/okpd2/search'
        : '/api/kpved/search'
      
      const url = `${endpoint}?q=${encodeURIComponent(searchValue)}`
      
      const response = await fetch(url)

      if (!response.ok) {
        // При 404 возвращаем пустые результаты вместо ошибки
        if (response.status === 404) {
          setSearchResults([])
          return
        }
        throw new Error('Ошибка поиска')
      }

      const data = await response.json()
      setSearchResults(data.results || data || [])
    } catch (err) {
      console.error('Search error:', err)
      // При ошибке устанавливаем пустые результаты
      setSearchResults([])
      const errorMessage = err instanceof Error ? err.message : 'Ошибка поиска'
      
      // Определяем тип ошибки
      const isConnectionError = 
        errorMessage.includes('fetch') || 
        errorMessage.includes('ECONNREFUSED') ||
        errorMessage.includes('Failed to fetch') ||
        errorMessage.includes('NetworkError')
      
      if (isConnectionError) {
        setError('Не удалось подключиться к серверу. Проверьте, что backend запущен и доступен.')
      } else {
        setError(`Ошибка поиска: ${errorMessage}`)
      }
      
      setSearchResults([])
    } finally {
      setSearching(false)
    }
  }, [searchQuery, classifierType])

  const toggleNode = useCallback((code: string, node: ClassifierNode) => {
    setExpandedNodes(prev => {
      const newSet = new Set(prev)
      if (newSet.has(code)) {
        newSet.delete(code)
      } else {
        newSet.add(code)
        // Загружаем дочерние узлы, если их еще нет
        if (node.has_children && (!node.children || node.children.length === 0)) {
          fetchHierarchy(code)
        }
      }
      return newSet
    })
    setSelectedNode(code)
  }, [fetchHierarchy])

  const expandAll = useCallback(() => {
    const allCodes = new Set<string>()
    const collectCodes = (nodes: ClassifierNode[]) => {
      nodes.forEach(node => {
        if (node.has_children) {
          allCodes.add(node.code)
          if (node.children) {
            collectCodes(node.children)
          }
        }
      })
    }
    collectCodes(hierarchy)
    setExpandedNodes(allCodes)
  }, [hierarchy])

  const collapseAll = useCallback(() => {
    setExpandedNodes(new Set())
  }, [])

  // Функция для поиска узла в иерархии
  const findNode = useCallback((nodes: ClassifierNode[], code: string): ClassifierNode | null => {
    for (const node of nodes) {
      if (node.code === code) {
        return node
      }
      if (node.children) {
        const found = findNode(node.children, code)
        if (found) return found
      }
    }
    return null
  }, [])

  // Функция для навигации к узлу и раскрытия иерархии
  const navigateToNode = useCallback(async (code: string, parentCode?: string) => {
    if (!code) return

    // Находим путь к узлу
    const path: string[] = []
    let currentCode: string | undefined = code
    
    // Собираем путь от узла к корню
    while (currentCode) {
      path.unshift(currentCode)
      const node = findNode(hierarchy, currentCode)
      if (node?.parent_code) {
        currentCode = node.parent_code
      } else if (parentCode && currentCode === code) {
        currentCode = parentCode
      } else {
        break
      }
    }

    // Раскрываем все узлы в пути
    const newExpanded = new Set(expandedNodes)
    for (const pathCode of path) {
      newExpanded.add(pathCode)
      // Загружаем дочерние узлы, если нужно
      const node = findNode(hierarchy, pathCode)
      if (node && !node.children && node.has_children) {
        await fetchHierarchy(pathCode)
      }
    }
    setExpandedNodes(newExpanded)
    setSelectedNode(code)
    setSearchResults([])

    // Прокручиваем к выбранному узлу
    setTimeout(() => {
      const element = document.querySelector(`[data-node-code="${code}"]`)
      element?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 300)
  }, [hierarchy, expandedNodes, findNode, fetchHierarchy])

  const getLevelColor = (level: number): string => {
    const colors = [
      'text-blue-600 dark:text-blue-400',
      'text-green-600 dark:text-green-400',
      'text-purple-600 dark:text-purple-400',
      'text-orange-600 dark:text-orange-400',
      'text-red-600 dark:text-red-400',
    ]
    return colors[level] || 'text-muted-foreground'
  }

  const renderNode = useCallback((node: ClassifierNode, level: number = 0): React.ReactNode => {
    const isExpanded = expandedNodes.has(node.code)
    const hasChildren = node.has_children || (node.children && node.children.length > 0)
    const isLoading = loadingNodes.has(node.code)
    const isSelected = selectedNode === node.code

    return (
      <motion.div
        key={node.code}
        initial={{ opacity: 0, x: -10 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ duration: 0.2 }}
        className="select-none"
      >
        <div
          role="button"
          tabIndex={0}
          aria-expanded={hasChildren ? isExpanded : undefined}
          aria-label={`${node.code} ${node.name}`}
          className={cn(
            "flex items-center gap-2 p-2 rounded-md transition-all cursor-pointer group",
            "hover:bg-accent focus:bg-accent focus:outline-none focus:ring-2 focus:ring-ring",
            isSelected && "bg-primary/10 border-l-2 border-primary",
            level > 0 && "ml-4"
          )}
          style={{ paddingLeft: `${level * 1.5}rem` }}
          onClick={() => hasChildren && toggleNode(node.code, node)}
          onKeyDown={(e) => {
            if ((e.key === 'Enter' || e.key === ' ') && hasChildren) {
              e.preventDefault()
              toggleNode(node.code, node)
            }
          }}
        >
          {hasChildren ? (
            isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            ) : (
              <ChevronRight
                className={cn(
                  "h-4 w-4 transition-transform text-muted-foreground group-hover:text-foreground",
                  isExpanded && "rotate-90"
                )}
              />
            )
          ) : (
            <div className="w-4" />
          )}
          <Badge 
            variant="outline" 
            className={cn("font-mono text-xs cursor-pointer hover:bg-accent transition-colors", getLevelColor(node.level))}
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(node.code)
              setCopiedCode(node.code)
              setTimeout(() => setCopiedCode(null), 2000)
            }}
            title="Нажмите, чтобы скопировать код"
          >
            {copiedCode === node.code ? (
              <>
                <Check className="h-3 w-3 mr-1 inline" />
                Скопировано
              </>
            ) : (
              node.code
            )}
          </Badge>
          <span className="flex-1 text-sm truncate">{node.name}</span>
          {node.item_count !== undefined && (
            <Badge variant="secondary" className="text-xs">
              {node.item_count.toLocaleString('ru-RU')}
            </Badge>
          )}
        </div>
        <AnimatePresence>
          {isExpanded && hasChildren && (
            <motion.div
              initial={{ height: 0, opacity: 0 }}
              animate={{ height: 'auto', opacity: 1 }}
              exit={{ height: 0, opacity: 0 }}
              transition={{ duration: 0.2 }}
              className="overflow-hidden"
            >
              {isLoading ? (
                <div className="ml-8 space-y-1">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <ClassifierNodeSkeleton key={i} />
                  ))}
                </div>
              ) : node.children && node.children.length > 0 ? (
                <div className="ml-4 border-l-2 border-muted pl-2">
                  {node.children.map(child => renderNode(child, level + 1))}
                </div>
              ) : (
                <div className="ml-8 p-2 text-sm text-muted-foreground">
                  Нет дочерних элементов
                </div>
              )}
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    )
  }, [expandedNodes, loadingNodes, selectedNode, toggleNode])

  const filteredHierarchy = useMemo(() => {
    if (filterLevel === null) return hierarchy
    
    const filterByLevel = (nodes: ClassifierNode[]): ClassifierNode[] => {
      return nodes
        .filter(node => node.level === filterLevel)
        .map(node => ({
          ...node,
          children: node.children ? filterByLevel(node.children) : undefined
        }))
    }
    
    return filterByLevel(hierarchy)
  }, [hierarchy, filterLevel])

  const breadcrumbItems = [
    { label: 'Классификаторы', href: '/classifiers', icon: FileText },
    { label: classifierInfo.name, href: `/classifiers/${classifierType}`, icon: BookOpen },
  ]

  if (!isValidClassifier) {
    return <ClassifierPageSkeleton />
  }

  return (
    <div className="container-wide mx-auto px-4 py-6 sm:py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      {/* Header */}
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4"
      >
        <div className="flex items-center gap-4 flex-1 min-w-0">
          <Link href="/classifiers" aria-label="Вернуться к списку классификаторов">
            <Button variant="outline" size="icon" className="flex-shrink-0">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 sm:gap-3 flex-wrap">
              <h1 className="text-2xl sm:text-3xl font-bold flex items-center gap-2 sm:gap-3 flex-wrap">
                <BookOpen className="h-6 w-6 sm:h-8 sm:w-8 text-primary flex-shrink-0" />
                <span className="truncate">{classifierInfo.name}</span>
              </h1>
              {/* <BackendStatusIndicator showLabel={true} /> */}
            </div>
            <p className="text-sm sm:text-base text-muted-foreground mt-1 line-clamp-2">
              {classifierInfo.fullName}
            </p>
          </div>
        </div>
        <div className="flex flex-col sm:flex-row gap-3 w-full sm:w-auto">
          <div className="flex flex-col sm:flex-row gap-2">
            {hierarchy.length > 0 && (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className="w-full sm:w-auto"
                    disabled={exporting}
                  >
                    {exporting ? (
                      <>
                        <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                        Экспорт...
                      </>
                    ) : (
                      <>
                        <Download className="h-4 w-4 mr-2" />
                        Экспорт
                      </>
                    )}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuItem
                    onClick={() => exportHierarchy('csv')}
                    disabled={exporting}
                  >
                    <FileText className="h-4 w-4 mr-2" />
                    Экспорт в CSV
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => exportHierarchy('json')}
                    disabled={exporting}
                  >
                    <FileText className="h-4 w-4 mr-2" />
                    Экспорт в JSON
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            )}
            {(classifierType === 'okpd2' || classifierType === 'kpved') && (
              <>
                <Button
                  onClick={() => setShowLoadDialog(true)}
                  variant="outline"
                  className="w-full sm:w-auto"
                >
                  <Upload className="h-4 w-4 mr-2" />
                  Загрузить
                </Button>
                {classifierType === 'okpd2' && hierarchy.length > 0 && (
                  <Button
                    onClick={() => setShowClearDialog(true)}
                    variant="outline"
                    className="w-full sm:w-auto text-destructive hover:text-destructive"
                  >
                    <X className="h-4 w-4 mr-2" />
                    Очистить
                  </Button>
                )}
              </>
            )}
          </div>
        </div>
      </motion.div>

      {/* Info Message */}
      {(classifierType === 'okpd2' || classifierType === 'kpved') && (
        <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950 dark:border-blue-800">
          <CardContent className="pt-6">
            <div className="flex items-start gap-3">
              <AlertCircle className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" />
              <div>
                <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                  Классификатор {classifierInfo.name}
                </p>
                <p className="text-sm text-blue-700 dark:text-blue-300 mt-1">
                  Данные {classifierInfo.name} хранятся в сервисной базе данных. Выбор базы данных не требуется.
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Statistics */}
      {stats && stats.total_codes > 0 && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Всего элементов</p>
                  <p className="text-2xl font-bold">{stats.total_codes.toLocaleString('ru-RU')}</p>
                </div>
                <BookOpen className="h-8 w-8 text-muted-foreground" />
              </div>
            </CardContent>
          </Card>
          <Card>
            <CardContent className="pt-6">
              <div className="flex items-center justify-between">
                <div>
                  <p className="text-sm font-medium text-muted-foreground">Максимальный уровень</p>
                  <p className="text-2xl font-bold">{stats.max_level}</p>
                </div>
                <Filter className="h-8 w-8 text-muted-foreground" />
              </div>
            </CardContent>
          </Card>
        </div>
      )}

      {/* Search */}
      <Card>
        <CardHeader>
          <CardTitle>Поиск по классификатору</CardTitle>
          <CardDescription>
            Найдите код или название в классификаторе {classifierInfo.name}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="flex flex-col sm:flex-row gap-2">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Введите код или название... (Ctrl+K)"
                value={searchInputValue}
                onChange={(e) => setSearchInputValue(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    setSearchQuery(searchInputValue)
                    handleSearch(searchInputValue)
                  }
                  if (e.key === 'Escape') {
                    setSearchInputValue('')
                    setSearchQuery('')
                    setSearchResults([])
                  }
                }}
                className="pl-10"
                aria-label="Поиск по классификатору"
              />
            </div>
            <Button 
              onClick={() => handleSearch()} 
              disabled={searching || !searchQuery.trim()}
              className="w-full sm:w-auto"
            >
              {searching ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Поиск...
                </>
              ) : (
                <>
                  <Search className="h-4 w-4 mr-2" />
                  Найти
                </>
              )}
            </Button>
          </div>
          
          <AnimatePresence>
            {searchResults.length > 0 && (
              <motion.div
                initial={{ opacity: 0, height: 0 }}
                animate={{ opacity: 1, height: 'auto' }}
                exit={{ opacity: 0, height: 0 }}
                className="space-y-2"
              >
                <div className="flex items-center justify-between">
                  <p className="text-sm text-muted-foreground">
                    Найдено: <span className="font-semibold text-foreground">{searchResults.length}</span>{' '}
                    {searchResults.length === 1 ? 'результат' : 'результатов'}
                  </p>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      setSearchInputValue('')
                      setSearchQuery('')
                      setSearchResults([])
                    }}
                    aria-label="Очистить результаты поиска"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
                <div className="space-y-2 max-h-[300px] overflow-y-auto">
                  {searchResults.map((result) => (
                    <motion.div
                      key={result.code}
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      className="p-3 border rounded-md hover:bg-accent cursor-pointer transition-colors"
                      onClick={async () => {
                        setSelectedNode(result.code)
                        setSearchInputValue('')
                        setSearchQuery('')
                        setSearchResults([])
                        // Находим путь к узлу и раскрываем иерархию
                        await navigateToNode(result.code, result.parent_code)
                      }}
                      role="button"
                      tabIndex={0}
                      onKeyDown={async (e) => {
                        if (e.key === 'Enter' || e.key === ' ') {
                          e.preventDefault()
                          setSelectedNode(result.code)
                          setSearchInputValue('')
                          setSearchQuery('')
                          setSearchResults([])
                          await navigateToNode(result.code, result.parent_code)
                        }
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="font-mono">
                          {result.code}
                        </Badge>
                        <span className="text-sm flex-1">{result.name}</span>
                        <Badge variant="secondary" className="text-xs">
                          Уровень {result.level}
                        </Badge>
                      </div>
                    </motion.div>
                  ))}
                </div>
              </motion.div>
            )}
          </AnimatePresence>
        </CardContent>
      </Card>

      {/* Hierarchy */}
      <Card>
        <CardHeader>
          <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div>
              <CardTitle>Иерархия классификатора</CardTitle>
              <CardDescription>
                Древовидная структура {classifierInfo.name}
              </CardDescription>
            </div>
            {hierarchy.length > 0 && (
              <div className="flex flex-wrap gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={expandAll}
                  aria-label="Развернуть все узлы"
                >
                  <Maximize2 className="h-4 w-4 mr-2" />
                  Развернуть
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={collapseAll}
                  aria-label="Свернуть все узлы"
                >
                  <Minimize2 className="h-4 w-4 mr-2" />
                  Свернуть
                </Button>
                <Separator orientation="vertical" className="h-6" />
                <div className="flex items-center gap-2">
                  <Filter className="h-4 w-4 text-muted-foreground" />
                  <select
                    value={filterLevel === null ? '' : filterLevel}
                    onChange={(e) => setFilterLevel(e.target.value === '' ? null : Number(e.target.value))}
                    className="text-sm border rounded-md px-2 py-1 bg-background"
                    aria-label="Фильтр по уровню"
                  >
                    <option value="">Все уровни</option>
                    {Array.from({ length: 5 }, (_, i) => (
                      <option key={i} value={i}>
                        Уровень {i}
                      </option>
                    ))}
                  </select>
                </div>
              </div>
            )}
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <LoadingState message="Загрузка иерархии..." size="lg" />
          ) : error ? (
            <ErrorState
              title="Ошибка загрузки"
              message={error}
              action={{
                label: 'Повторить',
                onClick: () => fetchHierarchy(),
              }}
            />
          ) : hierarchy.length === 0 ? (
            <div className="space-y-4">
              <EmptyState
                icon={BookOpen}
                title="Классификатор пуст"
                description={
                  classifierType === 'okpd2'
                    ? "Классификатор ОКПД2 не загружен. Загрузите данные классификатора через API или используйте команду загрузки."
                    : "Классификатор не загружен или не содержит данных"
                }
              />
              {classifierType === 'okpd2' && (
                <div className="mt-4 space-y-3">
                  <div className="p-4 bg-muted rounded-lg">
                    <p className="text-sm text-muted-foreground mb-2">
                      Для загрузки классификатора ОКПД2 используйте:
                    </p>
                    <ul className="text-sm text-muted-foreground space-y-1 list-disc list-inside mb-3">
                      <li>API endpoint: POST /api/okpd2/load</li>
                      <li>Команду: load_okpd2.exe -file путь_к_файлу.txt</li>
                    </ul>
                  </div>
                  <Button
                    onClick={() => setShowLoadDialog(true)}
                    className="w-full"
                  >
                    <Upload className="h-4 w-4 mr-2" />
                    Загрузить классификатор ОКПД2
                  </Button>
                </div>
              )}
            </div>
          ) : (
            <div className="rounded-md border overflow-hidden">
              <div className="max-h-[60vh] overflow-y-auto p-4 space-y-1">
                {(filterLevel !== null ? filteredHierarchy : hierarchy).map(node => renderNode(node))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Load Dialog */}
      <Dialog open={showLoadDialog} onOpenChange={setShowLoadDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Загрузка классификатора {classifierInfo.name}</DialogTitle>
            <DialogDescription>
              Загрузите файл {classifierInfo.name} для загрузки в базу данных
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="classifier-file-upload">Загрузить файл с компьютера</Label>
              <Input
                id="classifier-file-upload"
                type="file"
                accept=".txt,.csv"
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (file) {
                    // Валидация размера файла (максимум 100MB для классификаторов)
                    const maxSize = 100 * 1024 * 1024 // 100MB
                    if (file.size > maxSize) {
                      setLoadError(`Файл слишком большой. Максимальный размер: ${(maxSize / 1024 / 1024).toFixed(0)}MB`)
                      setSelectedFile(null)
                      return
                    }
                    
                    // Валидация типа файла
                    const validExtensions = ['.txt', '.csv']
                    const fileExtension = '.' + file.name.split('.').pop()?.toLowerCase()
                    if (!validExtensions.includes(fileExtension)) {
                      setLoadError(`Неподдерживаемый тип файла. Разрешенные форматы: ${validExtensions.join(', ')}`)
                      setSelectedFile(null)
                      return
                    }
                    
                    setLoadError(null)
                    setSelectedFile(file)
                    setFilePath('')
                    
                    // Предпросмотр файла (первые 10 строк)
                    if (file.size < 1024 * 1024) { // Только для файлов меньше 1MB
                      const reader = new FileReader()
                      reader.onload = (e) => {
                        const text = e.target?.result as string
                        const lines = text.split('\n').slice(0, 10)
                        setFilePreview(lines.join('\n'))
                      }
                      reader.readAsText(file)
                    } else {
                      setFilePreview(null)
                    }
                  } else {
                    setSelectedFile(null)
                    setFilePreview(null)
                  }
                }}
                disabled={loadingFile}
              />
              <p className="text-xs text-muted-foreground">
                Выберите файл {classifierInfo.name} для загрузки
              </p>
              {selectedFile && filePreview && (
                <div className="mt-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowFilePreview(!showFilePreview)}
                    className="text-xs"
                  >
                    {showFilePreview ? 'Скрыть' : 'Показать'} предпросмотр файла
                  </Button>
                  {showFilePreview && (
                    <div className="mt-2 p-3 bg-muted rounded-md max-h-40 overflow-auto">
                      <pre className="text-xs font-mono whitespace-pre-wrap break-words">
                        {filePreview}
                        {filePreview.split('\n').length >= 10 && '\n...'}
                      </pre>
                    </div>
                  )}
                </div>
              )}
            </div>
            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-background px-2 text-muted-foreground">или</span>
              </div>
            </div>
            <div className="space-y-2">
              <Label htmlFor="classifier-file-path">Путь к файлу на сервере</Label>
              <Input
                id="classifier-file-path"
                value={filePath}
                onChange={(e) => {
                  setFilePath(e.target.value)
                  if (e.target.value) {
                    setSelectedFile(null)
                  }
                }}
                placeholder={classifierType === 'okpd2' ? 'okpd2_data.txt' : 'КПВЭД.txt'}
                disabled={loadingFile}
              />
              <p className="text-xs text-muted-foreground">
                Укажите полный путь к файлу {classifierInfo.name} на сервере
              </p>
            </div>
            {loadError && (
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>{loadError}</AlertDescription>
              </Alert>
            )}
            {loadSuccess && (
              <Alert className="border-green-500 bg-green-50 dark:bg-green-950">
                <AlertCircle className="h-4 w-4 text-green-600" />
                <AlertDescription className="text-green-700 dark:text-green-300">
                  {loadSuccess}
                </AlertDescription>
              </Alert>
            )}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                setShowLoadDialog(false)
                setLoadError(null)
                setLoadSuccess(null)
                setSelectedFile(null)
                setFilePath('')
                setFilePreview(null)
                setShowFilePreview(false)
              }}
              disabled={loadingFile}
            >
              Отмена
            </Button>
            <Button
              onClick={async () => {
                if (!filePath.trim() && !selectedFile) {
                  setLoadError('Укажите путь к файлу или выберите файл для загрузки')
                  return
                }

                setLoadingFile(true)
                setLoadError(null)
                setLoadSuccess(null)

                try {
                  const apiPath = classifierType === 'okpd2' ? '/api/okpd2/load' : '/api/kpved/load'
                  
                  let response: Response
                  
                  if (selectedFile) {
                    const formData = new FormData()
                    formData.append('file', selectedFile)
                    
                    response = await fetch(apiPath, {
                      method: 'POST',
                      body: formData,
                    })
                  } else {
                    const body: { file_path: string } = {
                      file_path: filePath,
                    }
                    
                    response = await fetch(apiPath, {
                      method: 'POST',
                      headers: {
                        'Content-Type': 'application/json',
                      },
                      body: JSON.stringify(body),
                    })
                  }

                  if (!response.ok) {
                    const errorData = await response.json().catch(() => ({ error: 'Ошибка загрузки' }))
                    throw new Error(errorData.error || 'Ошибка загрузки классификатора')
                  }

                  const data = await response.json()
                  setLoadSuccess(`Классификатор успешно загружен! Загружено записей: ${data.total_codes || 0}`)
                  
                  // Обновляем иерархию и статистику
                  await fetchHierarchy()
                  await fetchStats()
                  
                  // Очищаем предпросмотр
                  setFilePreview(null)
                  setShowFilePreview(false)
                  
                  // Автоматически закрываем диалог через 2 секунды
                  setTimeout(() => {
                    setShowLoadDialog(false)
                    setLoadSuccess(null)
                    setSelectedFile(null)
                    setFilePath('')
                  }, 2000)
                } catch (err) {
                  setLoadError(err instanceof Error ? err.message : 'Ошибка загрузки классификатора')
                } finally {
                  setLoadingFile(false)
                }
              }}
              disabled={
                loadingFile || 
                (!filePath.trim() && !selectedFile)
              }
            >
              {loadingFile ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Загрузка...
                </>
              ) : (
                <>
                  <Upload className="h-4 w-4 mr-2" />
                  Загрузить
                </>
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Clear Dialog */}
      {classifierType === 'okpd2' && (
        <Dialog open={showClearDialog} onOpenChange={setShowClearDialog}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Очистка классификатора {classifierInfo.name}</DialogTitle>
              <DialogDescription>
                Вы уверены, что хотите удалить все данные классификатора? Это действие нельзя отменить.
              </DialogDescription>
            </DialogHeader>
            <div className="py-4">
              <Alert variant="destructive">
                <AlertCircle className="h-4 w-4" />
                <AlertDescription>
                  Будет удалено {stats?.total_codes || 0} записей классификатора.
                </AlertDescription>
              </Alert>
            </div>
            <DialogFooter>
              <Button
                variant="outline"
                onClick={() => setShowClearDialog(false)}
                disabled={clearing}
              >
                Отмена
              </Button>
              <Button
                variant="destructive"
                onClick={async () => {
                  setClearing(true)
                  setError(null)

                  try {
                    const response = await fetch('/api/okpd2/clear', {
                      method: 'POST',
                    })

                    if (!response.ok) {
                      const errorData = await response.json().catch(() => ({ error: 'Ошибка очистки' }))
                      throw new Error(errorData.error || 'Ошибка очистки классификатора')
                    }

                    const data = await response.json()
                    setShowClearDialog(false)
                    
                    // Обновляем иерархию и статистику
                    setHierarchy([])
                    await fetchStats()
                    
                    // Показываем сообщение об успехе
                    setLoadSuccess(`Классификатор успешно очищен. Удалено записей: ${data.deleted_count || 0}`)
                    setTimeout(() => {
                      setLoadSuccess(null)
                    }, 5000)
                  } catch (err) {
                    setError(err instanceof Error ? err.message : 'Ошибка очистки классификатора')
                  } finally {
                    setClearing(false)
                  }
                }}
                disabled={clearing}
              >
                {clearing ? (
                  <>
                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                    Очистка...
                  </>
                ) : (
                  <>
                    <X className="h-4 w-4 mr-2" />
                    Очистить
                  </>
                )}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </div>
  )
}
