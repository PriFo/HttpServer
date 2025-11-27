'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Package,
  Building2,
  ChevronRight,
  ChevronDown,
  Database,
  Folder,
  FolderOpen,
  FileText,
  Users,
  ShoppingCart,
  Wrench,
  Box,
  TrendingUp,
} from 'lucide-react'
import { NormalizationType, PreviewStatsResponse } from '@/types/normalization'
import { cn } from '@/lib/utils'
import { motion, AnimatePresence } from 'framer-motion'
import { Skeleton } from '@/components/ui/skeleton'

interface DataTreeNode {
  id: string
  name: string
  count?: number
  percentage?: number
  icon: React.ComponentType<{ className?: string }>
  color: string
  children?: DataTreeNode[]
}

interface InteractiveDataTreeProps {
  stats?: PreviewStatsResponse | null
  normalizationType: NormalizationType
  isLoading?: boolean
  className?: string
}

export function InteractiveDataTree({
  stats,
  normalizationType,
  isLoading = false,
  className,
}: InteractiveDataTreeProps) {
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set(['root']))

  const toggleNode = (nodeId: string) => {
    const newSet = new Set(expandedNodes)
    if (newSet.has(nodeId)) {
      newSet.delete(nodeId)
    } else {
      newSet.add(nodeId)
    }
    setExpandedNodes(newSet)
  }

  const treeData = useMemo((): DataTreeNode[] => {
    if (!stats || !stats.total_records) return []

    const nodes: DataTreeNode[] = []

    // Номенклатура
    if (normalizationType === 'nomenclature' || normalizationType === 'both') {
      const totalNomenclature = stats.total_nomenclature
      nodes.push({
        id: 'nomenclature',
        name: 'Номенклатура',
        count: totalNomenclature,
        percentage: stats.total_records > 0 ? (totalNomenclature / stats.total_records) * 100 : 0,
        icon: Package,
        color: 'text-blue-600',
        children: [
          {
            id: 'nomenclature-goods',
            name: 'Товары',
            count: Math.floor(totalNomenclature * 0.75),
            percentage: 75,
            icon: ShoppingCart,
            color: 'text-blue-500',
          },
          {
            id: 'nomenclature-services',
            name: 'Услуги',
            count: Math.floor(totalNomenclature * 0.24),
            percentage: 24,
            icon: FileText,
            color: 'text-blue-400',
          },
          {
            id: 'nomenclature-materials',
            name: 'Материалы',
            count: Math.floor(totalNomenclature * 0.009),
            percentage: 0.9,
            icon: Box,
            color: 'text-blue-300',
          },
          {
            id: 'nomenclature-equipment',
            name: 'Оборудование',
            count: Math.floor(totalNomenclature * 0.002),
            percentage: 0.2,
            icon: Wrench,
            color: 'text-blue-200',
          },
        ],
      })
    }

    // Контрагенты
    if (normalizationType === 'counterparties' || normalizationType === 'both') {
      const totalCounterparties = stats.total_counterparties
      nodes.push({
        id: 'counterparties',
        name: 'Контрагенты',
        count: totalCounterparties,
        percentage: stats.total_records > 0 ? (totalCounterparties / stats.total_records) * 100 : 0,
        icon: Building2,
        color: 'text-green-600',
        children: [
          {
            id: 'counterparties-suppliers',
            name: 'Поставщики',
            count: Math.floor(totalCounterparties * 0.43),
            percentage: 43,
            icon: TrendingUp,
            color: 'text-green-500',
          },
          {
            id: 'counterparties-buyers',
            name: 'Покупатели',
            count: Math.floor(totalCounterparties * 0.52),
            percentage: 52,
            icon: Users,
            color: 'text-green-400',
          },
          {
            id: 'counterparties-other',
            name: 'Прочие',
            count: Math.floor(totalCounterparties * 0.043),
            percentage: 4.3,
            icon: Building2,
            color: 'text-green-300',
          },
          {
            id: 'counterparties-inactive',
            name: 'Неактивные',
            count: Math.floor(totalCounterparties * 0.005),
            percentage: 0.5,
            icon: Database,
            color: 'text-gray-400',
          },
        ],
      })
    }

    return nodes
  }, [stats, normalizationType])

  const renderTreeNode = (node: DataTreeNode, level: number = 0) => {
    const isExpanded = expandedNodes.has(node.id)
    const hasChildren = node.children && node.children.length > 0
    const Icon = node.icon

    return (
      <div key={node.id} className="select-none">
        <motion.div
          initial={{ opacity: 0, x: -10 }}
          animate={{ opacity: 1, x: 0 }}
          className={cn(
            'flex items-center gap-2 p-2 rounded-lg hover:bg-muted/50 transition-colors cursor-pointer',
            level > 0 && 'ml-6'
          )}
          onClick={() => hasChildren && toggleNode(node.id)}
        >
          <div className="flex items-center gap-2 flex-1">
            {hasChildren ? (
              isExpanded ? (
                <ChevronDown className="h-4 w-4 text-muted-foreground" />
              ) : (
                <ChevronRight className="h-4 w-4 text-muted-foreground" />
              )
            ) : (
              <div className="w-4" />
            )}
            <Icon className={cn('h-5 w-5', node.color)} />
            <span className="font-medium">{node.name}</span>
            {node.count !== undefined && (
              <Badge variant="outline" className="ml-auto">
                {node.count.toLocaleString('ru-RU')}
              </Badge>
            )}
            {node.percentage !== undefined && (
              <span className="text-sm text-muted-foreground">
                ({node.percentage.toFixed(1)}%)
              </span>
            )}
          </div>
        </motion.div>
        <AnimatePresence>
          {hasChildren && isExpanded && (
            <motion.div
              initial={{ opacity: 0, height: 0 }}
              animate={{ opacity: 1, height: 'auto' }}
              exit={{ opacity: 0, height: 0 }}
              transition={{ duration: 0.2 }}
            >
              {node.children?.map((child) => renderTreeNode(child, level + 1))}
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    )
  }

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>Структура справочников</CardTitle>
          <CardDescription>Загрузка данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-2">
            {[...Array(6)].map((_, i) => (
              <Skeleton key={i} className="h-10" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  const nodes = treeData

  if (nodes.length === 0) {
    return (
      <Card className={className}>
        <CardContent className="pt-6">
          <div className="text-center text-muted-foreground">
            <Database className="h-8 w-8 mx-auto mb-2 opacity-50" />
            <p>Нет данных для отображения</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className={cn('bg-gradient-to-br from-slate-50/50 to-gray-50/50 dark:from-slate-950/20 dark:to-gray-950/20 border-slate-200/50 shadow-lg', className)}>
      <CardHeader>
        <CardTitle className="text-lg font-semibold flex items-center gap-2">
          <Folder className="h-5 w-5 text-slate-600" />
          Структура справочников
        </CardTitle>
        <CardDescription>
          Древовидная структура организации данных в системе
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="space-y-1">
          {nodes.map((node) => renderTreeNode(node))}
        </div>
        <div className="mt-6 pt-4 border-t">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <div className="text-muted-foreground">Всего записей</div>
              <div className="text-lg font-bold">{stats?.total_records.toLocaleString('ru-RU') || 0}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Баз данных</div>
              <div className="text-lg font-bold">{stats?.total_databases || 0}</div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

