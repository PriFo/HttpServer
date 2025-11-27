'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { 
  BookOpen, 
  Search, 
  ChevronRight, 
  ChevronDown,
  Folder,
  FileText,
  Loader2,
  Maximize2,
  Minimize2,
  Copy,
  Check,
  X,
  ArrowRight,
  BarChart3,
  Download,
  Filter,
  Home,
  Upload,
  AlertCircle,
  Database
} from 'lucide-react'
import Link from 'next/link'
import { DatabaseSelector } from '@/components/database-selector'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Label } from '@/components/ui/label'
import { LoadingState } from '@/components/common/loading-state'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { StatCard } from '@/components/common/stat-card'
// import dynamic from 'next/dynamic'

// const BackendStatusIndicator = dynamic(
//   () => import('@/components/common/backend-status-indicator').then(mod => ({ default: mod.BackendStatusIndicator })),
//   { ssr: false }
// )
import { FadeIn } from '@/components/animations/fade-in'
import { StaggerContainer, StaggerItem } from '@/components/animations/stagger-container'
import { motion } from 'framer-motion'
import { Progress } from '@/components/ui/progress'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

interface KPVEDNode {
  code: string
  name: string
  level: number
  children?: KPVEDNode[]
  has_children?: boolean
  item_count?: number
  parent_code?: string
}

interface SearchResult {
  code: string
  name: string
  level: number
  parent_code?: string
}

interface KPVEDHierarchyResponse {
  nodes: KPVEDNode[]
  total: number
}

export default function ClassifiersPage() {
  const [selectedDatabase, setSelectedDatabase] = useState<string | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [hierarchy, setHierarchy] = useState<KPVEDNode[]>([])
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set())
  const [loading, setLoading] = useState(false)
  const [loadingNodes, setLoadingNodes] = useState<Set<string>>(new Set())
  const [error, setError] = useState<string | null>(null)
  const [selectedClassifier, setSelectedClassifier] = useState<'kpved' | 'okpd2' | 'other'>('kpved')
  const [searchResults, setSearchResults] = useState<SearchResult[]>([])
  const [showSearchResults, setShowSearchResults] = useState(false)
  const [selectedNode, setSelectedNode] = useState<string | null>(null)
  const [copiedCode, setCopiedCode] = useState<string | null>(null)
  const [stats, setStats] = useState<{ total: number; levels: number } | null>(null)
  const [filterLevel, setFilterLevel] = useState<number | null>(null)
  const [nodePath, setNodePath] = useState<string[]>([])
  const [exporting, setExporting] = useState(false)
  const [loadingKpved, setLoadingKpved] = useState(false)
  const [kpvedFilePath, setKpvedFilePath] = useState('')
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [showLoadDialog, setShowLoadDialog] = useState(false)
  const [loadSuccess, setLoadSuccess] = useState<string | null>(null)
  
  // Защита от рендеринга объектов
  const safeRenderNumber = (value: unknown): string => {
    if (typeof value === 'number') {
      return value.toLocaleString('ru-RU')
    }
    if (typeof value === 'string') {
      const num = Number(value)
      return isNaN(num) ? '0' : num.toLocaleString('ru-RU')
    }
    return '0'
  }

  useEffect(() => {
    // Обновляем путь к файлу в зависимости от выбранного классификатора
    if (selectedClassifier === 'okpd2') {
      setKpvedFilePath('okpd2_data.txt')
    } else if (selectedClassifier === 'kpved') {
      setKpvedFilePath('КПВЭД.txt')
    }
  }, [selectedClassifier])

  useEffect(() => {
    // Для ОКПД2 и КПВЭД база данных не обязательна (данные в service.db)
    if (selectedClassifier === 'okpd2' || selectedClassifier === 'kpved') {
      fetchHierarchy()
      fetchStats()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedClassifier])

  const fetchStats = async () => {
    // Для ОКПД2 и КПВЭД база данных не нужна (данные в service.db)
    if (selectedClassifier !== 'okpd2' && selectedClassifier !== 'kpved') return
    
    try {
      const apiPath = selectedClassifier === 'okpd2' ? '/api/okpd2/stats' : '/api/kpved/stats'
      const response = await fetch(apiPath)
      if (response.ok) {
        const data = await response.json()
        // Убеждаемся, что данные в правильном формате
        const total = data.total_codes || data.total || 0
        const maxLevel = data.max_level || data.levels || 0
        setStats({
          total: Number(total),
          levels: Number(maxLevel)
        })
      } else if (response.status === 404) {
        // При 404 устанавливаем нулевые статистики
        setStats({
          total: 0,
          levels: 0
        })
      }
    } catch (err) {
      console.error('Error fetching stats:', err)
      // При ошибке устанавливаем нулевые статистики
      setStats({
        total: 0,
        levels: 0
      })
    }
  }

  const fetchHierarchy = async (parent?: string, level?: number) => {
    // Для ОКПД2 и КПВЭД база данных не нужна (данные в service.db)
    if (selectedClassifier !== 'okpd2' && selectedClassifier !== 'kpved') {
      setError('Выберите классификатор')
      return
    }

    // Используем setLoading только для корневых узлов (когда parent не указан)
    // Для дочерних узлов используем только loadingNodes
    const isRootLoad = !parent && level === undefined
    if (isRootLoad) {
      setLoading(true)
    }
    setError(null)

    try {
      const params = new URLSearchParams()
      if (parent) params.append('parent', parent)
      if (level !== undefined) params.append('level', level.toString())

      const apiPath = selectedClassifier === 'okpd2' ? '/api/okpd2/hierarchy' : '/api/kpved/hierarchy'
      const url = params.toString() ? `${apiPath}?${params.toString()}` : apiPath
      const response = await fetch(url)
      
      // Если ответ не OK, но это не критическая ошибка (таблица не существует), обрабатываем как пустой результат
      if (!response.ok) {
        // При 404 возвращаем пустую иерархию
        if (response.status === 404) {
          if (isRootLoad) {
            setHierarchy([])
          }
          return
        }
        
        const errorData = await response.json().catch(() => ({ error: 'Failed to fetch hierarchy' }))
        const errorMessage = errorData.error || 'Failed to fetch hierarchy'
        
        // Если таблица не существует или пуста - это нормально, просто показываем пустой список
        if (errorMessage.includes('no such table') || errorMessage.includes('kpved_classifier')) {
          if (isRootLoad) {
            setHierarchy([])
          }
          setError(null)
          return
        }
        
        // Для других ошибок показываем сообщение только если это не ошибка подключения
        if (!errorMessage.includes('ECONNREFUSED') && !errorMessage.includes('fetch')) {
          setError(errorMessage)
        } else {
          setError(null)
          if (isRootLoad) {
            setHierarchy([])
          }
        }
        return
      }

      const data: KPVEDHierarchyResponse = await response.json()
      
      // Проверяем формат ответа - может быть массив или объект с nodes
      const nodes = Array.isArray(data) ? data : (data.nodes || [])
      
      // Убеждаемся, что у всех узлов правильно установлен has_children
      // Обрабатываем разные форматы has_children (может быть boolean, string, number)
      const processedNodes = nodes.map(node => {
        let hasChildren = false
        if (node.has_children !== undefined && node.has_children !== null) {
          // Преобразуем в boolean, если пришло как строка или число
          if (typeof node.has_children === 'boolean') {
            hasChildren = node.has_children
          } else if (typeof node.has_children === 'string') {
            hasChildren = node.has_children === 'true' || node.has_children === '1'
          } else if (typeof node.has_children === 'number') {
            hasChildren = node.has_children > 0
          }
        }
        // Если has_children не указан, проверяем наличие children
        if (!hasChildren && node.children && node.children.length > 0) {
          hasChildren = true
        }
        // Если has_children не установлен явно, но узел существует,
        // предполагаем, что могут быть дочерние элементы (для возможности загрузки)
        // Это позволяет открывать узлы даже если has_children не был передан с сервера
        if (node.has_children === undefined && !hasChildren) {
          // Оставляем undefined, чтобы логика в toggleNode могла решить, загружать ли дочерние узлы
          return {
            ...node,
            has_children: undefined
          }
        }
        return {
          ...node,
          has_children: hasChildren
        }
      })
      
      if (parent) {
        // Обновляем конкретный узел - используем функциональное обновление для гарантии актуального состояния
        setHierarchy(prev => {
          const updated = updateNode(prev, parent, processedNodes)
          return updated
        })
      } else {
        // Устанавливаем корневые узлы
        setHierarchy(processedNodes)
      }
      setError(null)
    } catch (err) {
      // При ошибке подключения показываем информативное сообщение
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      
      // Определяем тип ошибки
      const isConnectionError = 
        errorMessage.includes('fetch') || 
        errorMessage.includes('ECONNREFUSED') ||
        errorMessage.includes('Failed to fetch') ||
        errorMessage.includes('NetworkError') ||
        errorMessage.includes('network')
      
      if (isConnectionError) {
        setError('Не удалось подключиться к серверу. Проверьте, что backend запущен и доступен.')
      } else if (!errorMessage.includes('no such table') && !errorMessage.includes('empty')) {
        setError(`Ошибка загрузки: ${errorMessage}`)
      } else {
        setError(null)
      }
      
      if (isRootLoad) {
        setHierarchy([])
      }
    } finally {
      if (isRootLoad) {
        setLoading(false)
      }
    }
  }

  const updateNode = (nodes: KPVEDNode[], code: string, children: KPVEDNode[]): KPVEDNode[] => {
    return nodes.map(node => {
      if (node.code === code) {
        // Найден узел - обновляем его children
        const processedChildren = children.map(child => {
          let hasChildren = false
          if (child.has_children !== undefined && child.has_children !== null) {
            if (typeof child.has_children === 'boolean') {
              hasChildren = child.has_children
            } else if (typeof child.has_children === 'string') {
              hasChildren = child.has_children === 'true' || child.has_children === '1'
            } else if (typeof child.has_children === 'number') {
              hasChildren = child.has_children > 0
            }
          }
          // Если has_children не установлен, но есть children, устанавливаем has_children = true
          if (!hasChildren && child.children && child.children.length > 0) {
            hasChildren = true
          }
          return {
            ...child,
            has_children: hasChildren
          }
        })
        // Обновляем узел с новыми children
        // Сохраняем has_children родителя явно, если он был установлен
        // Если не был установлен, проверяем наличие children
        const parentHasChildren = node.has_children !== undefined 
          ? node.has_children 
          : (processedChildren.length > 0)
        
        return { 
          ...node, 
          children: processedChildren,
          has_children: parentHasChildren
        }
      }
      // Рекурсивно ищем узел в дочерних элементах
      if (node.children && node.children.length > 0) {
        return { ...node, children: updateNode(node.children, code, children) }
      }
      return node
    })
  }

  const toggleNode = async (code: string) => {
    const newExpanded = new Set(expandedNodes)
    const isCurrentlyExpanded = newExpanded.has(code)
    
    if (isCurrentlyExpanded) {
      // Сворачиваем узел
      newExpanded.delete(code)
      setExpandedNodes(newExpanded)
    } else {
      // Раскрываем узел - сначала обновляем состояние
      newExpanded.add(code)
      setExpandedNodes(newExpanded)
      
      // Используем функциональное обновление для получения актуального состояния
      // и проверки необходимости загрузки дочерних узлов
      setHierarchy(currentHierarchy => {
        const node = findNode(currentHierarchy, code)
        
        // Упрощенная логика: загружаем дочерние узлы, если:
        // 1. Узел найден
        // 2. Дочерние узлы еще не загружены (отсутствуют или пустой массив)
        // 3. has_children === true (или не установлен, но узел существует)
        const needsLoad = !node?.children || node?.children?.length === 0
        const hasChildrenToLoad = node?.has_children === true || (node && needsLoad)
        
        if (node && needsLoad && hasChildrenToLoad) {
          // Устанавливаем состояние загрузки для конкретного узла
          setLoadingNodes(prev => new Set(prev).add(code))
          
          // Загружаем дочерние узлы асинхронно
          fetchHierarchy(code).catch(err => {
            console.error('Error loading children:', err)
            // При ошибке сворачиваем узел обратно
            setExpandedNodes(prev => {
              const updated = new Set(prev)
              updated.delete(code)
              return updated
            })
          }).finally(() => {
            // Убираем состояние загрузки
            setLoadingNodes(prev => {
              const newSet = new Set(prev)
              newSet.delete(code)
              return newSet
            })
          })
        }
        
        return currentHierarchy
      })
    }
  }

  const findNode = (nodes: KPVEDNode[], code: string): KPVEDNode | null => {
    for (const node of nodes) {
      if (node.code === code) return node
      if (node.children) {
        const found = findNode(node.children, code)
        if (found) return found
      }
    }
    return null
  }

  const searchKPVED = async () => {
    if (!searchQuery.trim()) {
      setShowSearchResults(false)
      setSearchResults([])
      return
    }
    // Для ОКПД2 и КПВЭД база данных не нужна (данные в service.db)
    if (selectedClassifier !== 'okpd2' && selectedClassifier !== 'kpved') {
      setError('Выберите классификатор')
      return
    }

    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams()
      params.append('q', searchQuery)
      params.append('limit', '50')
      
      const apiPath = selectedClassifier === 'okpd2' ? '/api/okpd2/search' : '/api/kpved/search'
      const response = await fetch(`${apiPath}?${params.toString()}`)
      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Failed to search' }))
        throw new Error(errorData.error || 'Failed to search')
      }

      const data = await response.json()
      const results = Array.isArray(data) ? data : []
      setSearchResults(results)
      setShowSearchResults(results.length > 0)
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Unknown error'
      
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
      setShowSearchResults(false)
    } finally {
      setLoading(false)
    }
  }

  const navigateToNode = async (code: string, parentCode?: string) => {
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
    setShowSearchResults(false)

    // Прокручиваем к выбранному узлу
    setTimeout(() => {
      const element = document.querySelector(`[data-node-code="${code}"]`)
      element?.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }, 300)
  }

  const expandAll = async () => {
    const allCodes = new Set<string>()
    const collectCodes = (nodes: KPVEDNode[]) => {
      nodes.forEach(node => {
        if (node.has_children) {
          allCodes.add(node.code)
        }
        if (node.children) {
          collectCodes(node.children)
        }
      })
    }
    collectCodes(hierarchy)
    setExpandedNodes(allCodes)
    
    // Загружаем все дочерние узлы
    for (const code of allCodes) {
      const node = findNode(hierarchy, code)
      if (node && !node.children && node.has_children) {
        await fetchHierarchy(code)
      }
    }
  }

  const collapseAll = () => {
    setExpandedNodes(new Set())
  }

  const copyToClipboard = async (text: string, code: string) => {
    try {
      await navigator.clipboard.writeText(text)
      setCopiedCode(code)
      setTimeout(() => setCopiedCode(null), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  const getLevelColor = (level: number): string => {
    const colors = [
      'text-blue-600', // level 0
      'text-green-600', // level 1
      'text-purple-600', // level 2
      'text-orange-600', // level 3
      'text-red-600', // level 4+
    ]
    return colors[Math.min(level, colors.length - 1)]
  }

  const getNodePath = (code: string, nodes: KPVEDNode[], path: string[] = []): string[] | null => {
    for (const node of nodes) {
      const currentPath = [...path, node.code]
      if (node.code === code) {
        return currentPath
      }
      if (node.children) {
        const found = getNodePath(code, node.children, currentPath)
        if (found) return found
      }
    }
    return null
  }

  const loadKpved = async () => {
    // Для ОКПД2 и КПВЭД база данных не нужна (данные в service.db)
    if (selectedClassifier !== 'okpd2' && selectedClassifier !== 'kpved') {
      setError('Выберите классификатор')
      return
    }

    // Проверяем, что указан либо путь к файлу, либо выбран файл
    if (!kpvedFilePath.trim() && !selectedFile) {
      const fileType = selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'
      setError(`Укажите путь к файлу ${fileType} или выберите файл для загрузки`)
      return
    }

    setLoadingKpved(true)
    setError(null)
    setLoadSuccess(null)

    try {
      const fileType = selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'
      
      // Используем API route фронтенда для проксирования запросов
      const apiPath = selectedClassifier === 'okpd2' ? '/api/okpd2/load' : '/api/kpved/load'
      
      let response: Response
      
      // Если выбран файл, загружаем через multipart/form-data
      if (selectedFile) {
        const formData = new FormData()
        formData.append('file', selectedFile)
        
        response = await fetch(apiPath, {
          method: 'POST',
          body: formData,
        })
      } else {
        // Используем JSON с путем к файлу
        const body: { file_path: string } = {
          file_path: kpvedFilePath,
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
        const errorData = await response.json().catch(() => ({ error: `Failed to load ${fileType}` }))
        throw new Error(errorData.error || `Ошибка загрузки ${fileType}`)
      }

      const data = await response.json()
      setLoadSuccess(`Классификатор успешно загружен! Загружено записей: ${data.total_codes || 0}`)
      setShowLoadDialog(false)
      setSelectedFile(null)
      setKpvedFilePath(selectedClassifier === 'okpd2' ? 'okpd2_data.txt' : 'КПВЭД.txt')
      
      // Обновляем статистику и иерархию
      await fetchStats()
      await fetchHierarchy()
      
      // Автоматически скрываем сообщение об успехе через 5 секунд
      setTimeout(() => {
        setLoadSuccess(null)
      }, 5000)
    } catch (err) {
      const fileType = selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'
      setError(err instanceof Error ? err.message : `Ошибка загрузки ${fileType}`)
    } finally {
      setLoadingKpved(false)
    }
  }

  const exportHierarchy = async (format: 'csv' | 'json') => {
    if (!hierarchy || hierarchy.length === 0) {
      setError('Нет данных для экспорта')
      return
    }

    setExporting(true)
    try {
      const flattenNodes = (nodes: KPVEDNode[], parentPath: string[] = []): Array<{code: string, name: string, level: number, path: string}> => {
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

      if (format === 'json') {
        const jsonData = JSON.stringify(flatData, null, 2)
        const blob = new Blob([jsonData], { type: 'application/json' })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `kpved_classifier_${new Date().toISOString().split('T')[0]}.json`
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
        link.download = `kpved_classifier_${new Date().toISOString().split('T')[0]}.csv`
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
  }

  const navigateToRoot = () => {
    setSelectedNode(null)
    setNodePath([])
    setExpandedNodes(new Set())
  }

  const navigateToParent = () => {
    if (nodePath.length > 1) {
      const newPath = nodePath.slice(0, -1)
      setNodePath(newPath)
      const parentCode = newPath[newPath.length - 1]
      setSelectedNode(parentCode)
      
      // Прокручиваем к родительскому узлу
      setTimeout(() => {
        const element = document.querySelector(`[data-node-code="${parentCode}"]`)
        element?.scrollIntoView({ behavior: 'smooth', block: 'center' })
      }, 100)
    } else {
      navigateToRoot()
    }
  }

  const filterByLevel = (level: number | null) => {
    setFilterLevel(level)
    if (level === null) {
      // Показываем все узлы
      return
    }
    // Фильтрация будет применена при рендеринге
  }

  const getFilteredNodes = (nodes: KPVEDNode[]): KPVEDNode[] => {
    if (filterLevel === null) return nodes
    
    const filterRecursive = (nodeList: KPVEDNode[]): KPVEDNode[] => {
      return nodeList
        .filter(node => node.level === filterLevel || (node.children && node.children.some(child => child.level === filterLevel)))
        .map(node => ({
          ...node,
          children: node.children ? filterRecursive(node.children) : undefined
        }))
    }
    
    return filterRecursive(nodes)
  }

  const renderNode = (node: KPVEDNode, depth: number = 0): React.ReactElement => {
    const isExpanded = expandedNodes.has(node.code)
    // Проверяем наличие дочерних элементов: либо они уже загружены, либо указано has_children === true
    const hasChildren = (node.children && node.children.length > 0) || node.has_children === true
    const isLoading = loadingNodes.has(node.code)
    const isSelected = selectedNode === node.code
    const levelColor = getLevelColor(node.level)

    return (
      <div key={node.code} data-node-code={node.code} className="select-none">
        <div
          className={`flex items-center gap-2 py-2 px-2 rounded transition-colors ${
            hasChildren ? 'cursor-pointer hover:bg-muted' : 'cursor-default'
          } ${isSelected ? 'bg-primary/10 border-l-2 border-primary' : ''} ${
            depth > 0 ? 'ml-4' : ''
          }`}
          onClick={(e) => {
            e.stopPropagation()
            if (hasChildren) {
              toggleNode(node.code)
            } else {
              // Если нет дочерних элементов, просто выбираем узел
              setSelectedNode(node.code)
              const path = getNodePath(node.code, hierarchy)
              if (path) {
                setNodePath(path)
              }
            }
          }}
        >
          {hasChildren ? (
            isLoading ? (
              <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
            ) : isExpanded ? (
              <ChevronDown className="h-4 w-4 text-muted-foreground" />
            ) : (
              <ChevronRight className="h-4 w-4 text-muted-foreground" />
            )
          ) : (
            <div className="w-4" />
          )}
          
          <Folder className={`h-4 w-4 ${levelColor}`} />
          
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 flex-wrap">
              <span className={`font-mono text-sm font-semibold ${levelColor}`}>
                {node.code}
              </span>
              <span className="text-sm truncate">{node.name}</span>
              <div className="flex items-center gap-1 ml-auto">
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0"
                  onClick={(e) => {
                    e.stopPropagation()
                    copyToClipboard(node.code, node.code)
                  }}
                  title="Копировать код"
                >
                  {copiedCode === node.code ? (
                    <Check className="h-3 w-3 text-green-600" />
                  ) : (
                    <Copy className="h-3 w-3" />
                  )}
                </Button>
              </div>
            </div>
            <div className="flex items-center gap-2 mt-1">
              {node.item_count !== undefined && (
                <Badge variant="outline" className="text-xs">
                  {node.item_count} записей
                </Badge>
              )}
              <Badge variant="secondary" className="text-xs">
                Уровень {node.level}
              </Badge>
            </div>
          </div>
        </div>

        {isExpanded && hasChildren && (
          <div className="ml-4 border-l-2 border-muted pl-2">
            {isLoading ? (
              <div className="py-2 px-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin inline mr-2" />
                Загрузка...
              </div>
            ) : node.children && node.children.length > 0 ? (
              node.children.map(child => renderNode(child, depth + 1))
            ) : node.children && node.children.length === 0 ? (
              <div className="py-2 px-2 text-sm text-muted-foreground">
                Нет дочерних элементов
              </div>
            ) : (
              <div className="py-2 px-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin inline mr-2" />
                Загрузка...
              </div>
            )}
          </div>
        )}
      </div>
    )
  }

  const breadcrumbItems = [
    { label: 'Классификаторы', href: '/classifiers', icon: BookOpen },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <div className="flex items-center justify-between flex-wrap gap-4">
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-2">
              <motion.h1 
                className="text-3xl font-bold flex items-center gap-2"
                initial={{ opacity: 0, y: -20 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ duration: 0.5 }}
              >
                <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/50">
                  <BookOpen className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                </div>
                Классификаторы
              </motion.h1>
              {/* {typeof window !== 'undefined' && <BackendStatusIndicator showLabel={true} />} */}
            </div>
            <motion.p 
              className="text-muted-foreground mt-2"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: 0.1 }}
            >
              Просмотр иерархии классификаторов (КПВЭД, ОКПД2 и другие)
            </motion.p>
          </div>
        </div>
      </FadeIn>

      {/* Classifier Selection */}
      <FadeIn>
        <Card>
          <CardHeader>
            <CardTitle>Выберите классификатор</CardTitle>
            <CardDescription>
              Перейдите на отдельную страницу классификатора для детального просмотра
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-2 flex-wrap">
              <Link href="/classifiers/kpved">
                <Button variant={selectedClassifier === 'kpved' ? 'default' : 'outline'} className="gap-2">
                  <FileText className="h-4 w-4" />
                  КПВЭД
                </Button>
              </Link>
              <Link href="/classifiers/okpd2">
                <Button variant={selectedClassifier === 'okpd2' ? 'default' : 'outline'} className="gap-2">
                  <FileText className="h-4 w-4" />
                  ОКПД2
                </Button>
              </Link>
            </div>
          </CardContent>
        </Card>
      </FadeIn>

      {(selectedClassifier === 'kpved' || selectedClassifier === 'okpd2') && (
        <>
          {(selectedClassifier === 'okpd2' || selectedClassifier === 'kpved') && (
            <Card className="border-blue-200 bg-blue-50 dark:bg-blue-950 dark:border-blue-800">
              <CardContent className="pt-6">
                <div className="flex items-start gap-3">
                  <AlertCircle className="h-5 w-5 text-blue-600 dark:text-blue-400 mt-0.5" />
                  <div>
                    <p className="text-sm font-medium text-blue-900 dark:text-blue-100">
                      Классификатор {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'}
                    </p>
                    <p className="text-sm text-blue-700 dark:text-blue-300 mt-1">
                      Данные {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'} хранятся в сервисной базе данных. Выбор базы данных не требуется.
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          )}
          {/* Search and Filters */}
          <Card>
            <CardHeader>
              <CardTitle>Поиск и фильтры</CardTitle>
              <CardDescription>
                Найдите код или название в классификаторе
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex gap-4 flex-wrap">
                  <div className="flex-1">
                    <div className="relative">
                      <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="Введите код или название..."
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-9"
                        aria-label="Поиск"
                      />
                    </div>
                  </div>
                  <div className="space-y-2">
                    <Label>Уровень</Label>
                    <Select
                      value={filterLevel === null ? 'all' : String(filterLevel)}
                      onValueChange={(value) => {
                        setFilterLevel(value === 'all' ? null : Number(value))
                      }}
                    >
                      <SelectTrigger aria-label="Уровень" className="w-[180px]">
                        <SelectValue placeholder="Все уровни" />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">Все уровни</SelectItem>
                        <SelectItem value="0">Уровень 0</SelectItem>
                        <SelectItem value="1">Уровень 1</SelectItem>
                        <SelectItem value="2">Уровень 2</SelectItem>
                        <SelectItem value="3">Уровень 3</SelectItem>
                        <SelectItem value="4">Уровень 4+</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
                {(searchQuery || filterLevel !== null) && (
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-sm text-muted-foreground">Активные фильтры:</span>
                    {searchQuery && (
                      <Badge variant="secondary" className="flex items-center gap-1">
                        Поиск: {searchQuery}
                        <button
                          className="ml-1 hover:text-destructive"
                          onClick={() => setSearchQuery('')}
                          aria-label="Удалить фильтр поиска"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    )}
                    {filterLevel !== null && (
                      <Badge variant="secondary" className="flex items-center gap-1">
                        Уровень: {filterLevel}
                        <button
                          className="ml-1 hover:text-destructive"
                          onClick={() => setFilterLevel(null)}
                          aria-label="Удалить фильтр уровня"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setSearchQuery('')
                        setFilterLevel(null)
                        setShowSearchResults(false)
                      }}
                      className="h-6 text-xs"
                    >
                      Сбросить все
                    </Button>
                  </div>
                )}
              </div>
              <div className="mt-2">
                <Button onClick={searchKPVED} disabled={!searchQuery.trim() || loading} size="sm">
                  <Search className="w-4 h-4 mr-2" />
                  Найти
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Load Classifier Button - для ОКПД2 и КПВЭД показываем всегда */}
          {(selectedClassifier === 'okpd2' || selectedClassifier === 'kpved') && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Upload className="h-5 w-5" />
                  Управление классификатором
                </CardTitle>
                <CardDescription>
                  Загрузите классификатор {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'} из файла в базу данных
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button
                  onClick={() => setShowLoadDialog(true)}
                  disabled={loadingKpved}
                  className="w-full sm:w-auto"
                >
                  {loadingKpved ? (
                    <>
                      <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                      Загрузка...
                    </>
                  ) : (
                    <>
                      <Upload className="h-4 w-4 mr-2" />
                      Загрузить {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'} из файла
                    </>
                  )}
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Statistics */}
          {stats && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <StatCard
                title="Всего элементов"
                value={stats.total}
                icon={BarChart3}
                formatValue={(val) => safeRenderNumber(val)}
              />
              <StatCard
                title="Уровней в иерархии"
                value={stats.levels}
                icon={BookOpen}
                variant="primary"
                formatValue={(val) => safeRenderNumber(val)}
              />
            </div>
          )}

          {/* Success Message */}
          {loadSuccess && (
            <Alert className="border-green-500 bg-green-50 dark:bg-green-950">
              <AlertCircle className="h-4 w-4 text-green-600" />
              <AlertDescription className="text-green-700 dark:text-green-300">
                {loadSuccess}
              </AlertDescription>
            </Alert>
          )}

          {/* Load Dialog */}
          <Dialog open={showLoadDialog} onOpenChange={setShowLoadDialog}>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Загрузка классификатора {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'}</DialogTitle>
                <DialogDescription>
                  Укажите путь к файлу {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД.txt'} для загрузки в базу данных
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
                          setError(`Файл слишком большой. Максимальный размер: ${(maxSize / 1024 / 1024).toFixed(0)}MB`)
                          setSelectedFile(null)
                          return
                        }
                        
                        // Валидация типа файла
                        const validExtensions = ['.txt', '.csv']
                        const fileExtension = '.' + file.name.split('.').pop()?.toLowerCase()
                        if (!validExtensions.includes(fileExtension)) {
                          setError(`Неподдерживаемый тип файла. Разрешенные форматы: ${validExtensions.join(', ')}`)
                          setSelectedFile(null)
                          return
                        }
                        
                        setError(null)
                        setSelectedFile(file)
                        setKpvedFilePath('')
                      } else {
                        setSelectedFile(null)
                      }
                    }}
                    disabled={loadingKpved}
                  />
                  <p className="text-xs text-muted-foreground">
                    Выберите файл {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'} для загрузки (максимум 100MB)
                  </p>
                </div>
                {selectedFile && (
                  <div className="p-3 bg-muted rounded-lg space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="font-medium">Выбранный файл:</span>
                      <span className="text-muted-foreground">{(selectedFile.size / 1024 / 1024).toFixed(2)} MB</span>
                    </div>
                    <p className="text-xs text-muted-foreground truncate" title={selectedFile.name}>
                      {selectedFile.name}
                    </p>
                  </div>
                )}
                <div className="relative">
                  <div className="absolute inset-0 flex items-center">
                    <span className="w-full border-t" />
                  </div>
                  <div className="relative flex justify-center text-xs uppercase">
                    <span className="bg-background px-2 text-muted-foreground">или</span>
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="kpved-file-path">Путь к файлу на сервере</Label>
                  <Input
                    id="kpved-file-path"
                    value={kpvedFilePath}
                    onChange={(e) => {
                      setKpvedFilePath(e.target.value)
                      if (e.target.value) {
                        setSelectedFile(null)
                      }
                    }}
                    placeholder={selectedClassifier === 'okpd2' ? 'okpd2_data.txt' : 'КПВЭД.txt'}
                    disabled={loadingKpved}
                  />
                  <p className="text-xs text-muted-foreground">
                    Укажите полный путь к файлу {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'} на сервере
                  </p>
                </div>
                {loadingKpved && (
                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-muted-foreground">Загрузка файла...</span>
                      <Loader2 className="h-4 w-4 animate-spin" />
                    </div>
                    <Progress value={undefined} className="h-2" />
                  </div>
                )}
                {error && (
                  <Alert variant="destructive">
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>{error}</AlertDescription>
                  </Alert>
                )}
              </div>
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => {
                    setShowLoadDialog(false)
                    setError(null)
                    setSelectedFile(null)
                  }}
                  disabled={loadingKpved}
                >
                  Отмена
                </Button>
                <Button
                  onClick={loadKpved}
                  disabled={
                    loadingKpved || 
                    (!kpvedFilePath.trim() && !selectedFile)
                  }
                >
                  {loadingKpved ? (
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

          {/* Search Results */}
          {showSearchResults && searchResults.length > 0 && (
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle>Результаты поиска</CardTitle>
                    <CardDescription>
                      Найдено: {searchResults.length} {searchResults.length === 1 ? 'элемент' : 'элементов'}
                    </CardDescription>
                  </div>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => setShowSearchResults(false)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              </CardHeader>
              <CardContent>
                <ScrollArea className="h-[300px]">
                  <div className="space-y-2">
                    {searchResults.map((result) => (
                      <div
                        key={result.code}
                        className="flex items-center justify-between p-3 border rounded hover:bg-muted cursor-pointer transition-colors"
                        onClick={() => navigateToNode(result.code, result.parent_code)}
                      >
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2">
                            <span className={`font-mono text-sm font-semibold ${getLevelColor(result.level)}`}>
                              {result.code}
                            </span>
                            <span className="text-sm truncate">{result.name}</span>
                          </div>
                          <Badge variant="secondary" className="mt-1 text-xs">
                            Уровень {result.level}
                          </Badge>
                        </div>
                        <ArrowRight className="h-4 w-4 text-muted-foreground ml-2" />
                      </div>
                    ))}
                  </div>
                </ScrollArea>
              </CardContent>
            </Card>
          )}

          {/* Hierarchy */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle>Иерархия {selectedClassifier === 'okpd2' ? 'ОКПД2' : 'КПВЭД'}</CardTitle>
                  <CardDescription>
                    Древовидная структура классификатора
                  </CardDescription>
                </div>
                {hierarchy && hierarchy.length > 0 && (
                  <div className="flex gap-2 flex-wrap">
                    {/* Breadcrumbs Navigation */}
                    {nodePath.length > 0 && (
                      <div className="flex items-center gap-1 mr-2">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={navigateToRoot}
                          title="Корень"
                        >
                          <Home className="h-4 w-4" />
                        </Button>
                        {nodePath.length > 1 && (
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={navigateToParent}
                            title="На уровень выше"
                          >
                            <ChevronRight className="h-4 w-4 rotate-180" />
                          </Button>
                        )}
                        <div className="flex items-center gap-1 text-sm text-muted-foreground px-2">
                          {nodePath.map((code, idx) => (
                            <span key={code}>
                              {code}
                              {idx < nodePath.length - 1 && <span className="mx-1">/</span>}
                            </span>
                          ))}
                        </div>
                      </div>
                    )}
                    
                    {/* Filter by Level */}
                    {stats && stats.levels > 0 && (
                      <div className="flex items-center gap-2">
                        <Filter className="h-4 w-4 text-muted-foreground" />
                        <select
                          value={filterLevel === null ? '' : filterLevel}
                          onChange={(e) => filterByLevel(e.target.value === '' ? null : Number(e.target.value))}
                          className="text-sm border rounded px-2 py-1"
                        >
                          <option value="">Все уровни</option>
                          {Array.from({ length: stats.levels }, (_, i) => (
                            <option key={i} value={i}>Уровень {i}</option>
                          ))}
                        </select>
                      </div>
                    )}
                    
                    {/* Export Buttons */}
                    <div className="flex gap-1">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => exportHierarchy('csv')}
                        disabled={exporting}
                        title="Экспорт в CSV"
                      >
                        <Download className="h-4 w-4 mr-2" />
                        CSV
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => exportHierarchy('json')}
                        disabled={exporting}
                        title="Экспорт в JSON"
                      >
                        <Download className="h-4 w-4 mr-2" />
                        JSON
                      </Button>
                    </div>
                    
                    {/* Expand/Collapse */}
                    <div className="flex gap-1">
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={expandAll}
                        title="Развернуть все"
                      >
                        <Maximize2 className="h-4 w-4 mr-2" />
                        Развернуть
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={collapseAll}
                        title="Свернуть все"
                      >
                        <Minimize2 className="h-4 w-4 mr-2" />
                        Свернуть
                      </Button>
                    </div>
                  </div>
                )}
              </div>
            </CardHeader>
            <CardContent>
              {error && (
                <Alert variant="destructive" className="mb-4">
                  <AlertCircle className="h-4 w-4" />
                  <AlertDescription className="flex items-center justify-between">
                    <span>{error}</span>
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => {
                        setError(null)
                        // Повторная попытка загрузки
                        if (selectedClassifier === 'okpd2' || selectedClassifier === 'kpved') {
                          fetchHierarchy()
                        }
                      }}
                      className="ml-2 h-6 text-xs"
                    >
                      Повторить
                    </Button>
                  </AlertDescription>
                </Alert>
              )}

              {loading && (!hierarchy || hierarchy.length === 0) ? (
                <LoadingState message="Загрузка иерархии..." size="lg" fullScreen />
              ) : !hierarchy || hierarchy.length === 0 ? (
                <EmptyState
                  icon={BookOpen}
                  title="Классификатор пуст"
                  description="Классификатор не загружен или не содержит данных. Загрузите классификатор из файла."
                />
              ) : (
                <div className="rounded-md border overflow-hidden">
                  <ScrollArea className="max-h-[60vh]">
                    <div className="space-y-1 p-4">
                      {filterLevel !== null ? (
                        getFilteredNodes(hierarchy).map(node => renderNode(node))
                      ) : (
                        hierarchy.map(node => renderNode(node))
                      )}
                    </div>
                  </ScrollArea>
                </div>
              )}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

