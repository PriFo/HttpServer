'use client'

import React, { useEffect, useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { ArrowLeftRight, Plus, Minus } from 'lucide-react'
import { formatDate } from '@/lib/locale'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'

interface VersionComparisonProps {
  clientId: string
  projectId: string
  entityId: string
  entityType: 'group' | 'item'
}

interface Version {
  id: string
  timestamp: string
  user: string
  data: any
  changes?: {
    field: string
    old_value: any
    new_value: any
  }[]
}

export const VersionComparison: React.FC<VersionComparisonProps> = ({
  clientId,
  projectId,
  entityId,
  entityType,
}) => {
  const [selectedVersion1, setSelectedVersion1] = useState<string | null>(null)
  const [selectedVersion2, setSelectedVersion2] = useState<string | null>(null)
  const { data, loading, error, refetch } = useProjectState(
    async (cid, pid, signal) => {
      if (!entityId) {
        return { versions: [] }
      }
      const params = new URLSearchParams({
        entity_id: entityId,
        entity_type: entityType,
      })
      const response = await fetch(
        `/api/clients/${cid}/projects/${pid}/normalization/versions?${params.toString()}`,
        { cache: 'no-store', signal }
      )
      if (!response.ok) {
        // Для 404 возвращаем пустые данные вместо ошибки
        if (response.status === 404) {
          return { versions: [] }
        }
        throw new Error(`Не удалось загрузить версии: ${response.status}`)
      }
      return response.json()
    },
    clientId,
    projectId,
    [entityId, entityType],
    {
      enabled: Boolean(clientId && projectId && entityId),
      refetchInterval: null,
    }
  )

  const versions: Version[] = (data?.versions as Version[]) || []

  useEffect(() => {
    if (versions.length >= 2) {
      setSelectedVersion1(versions[0].id)
      setSelectedVersion2(versions[1].id)
    } else if (versions.length === 1) {
      setSelectedVersion1(versions[0].id)
      setSelectedVersion2(null)
    } else {
      setSelectedVersion1(null)
      setSelectedVersion2(null)
    }
  }, [versions])

  const version1 = versions.find(v => v.id === selectedVersion1)
  const version2 = versions.find(v => v.id === selectedVersion2)

  const getDiff = (field: string) => {
    if (!version1 || !version2) return null

    const val1 = version1.data[field]
    const val2 = version2.data[field]

    if (val1 === val2) return { type: 'unchanged', value: val1 }

    return {
      type: 'changed',
      old: val1,
      new: val2,
    }
  }

  if (loading) {
    return <LoadingState message="Загрузка версий..." />
  }

  if (error) {
    return (
      <ErrorState
        title="Ошибка загрузки версий"
        message={error}
        action={{
          label: 'Повторить',
          onClick: () => refetch(),
        }}
      />
    )
  }

  if (versions.length < 2) {
    return (
      <Card>
        <CardContent className="py-8">
          <div className="text-center text-muted-foreground">
            Недостаточно версий для сравнения
          </div>
        </CardContent>
      </Card>
    )
  }

  const fields = version1 && version2
    ? Object.keys({ ...version1.data, ...version2.data })
    : []

  return (
    <Card>
      <CardHeader>
        <CardTitle>Сравнение версий</CardTitle>
        <CardDescription>
          Сравнение различных версий записи
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Версия 1</label>
            <Select value={selectedVersion1 || ''} onValueChange={setSelectedVersion1}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {versions.map((v: Version) => (
                  <SelectItem key={v.id} value={v.id}>
                    {formatDate(v.timestamp)} - {v.user}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {version1 && (
              <p className="text-xs text-muted-foreground">
                {formatDate(version1.timestamp)} • {version1.user}
              </p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">Версия 2</label>
            <Select value={selectedVersion2 || ''} onValueChange={setSelectedVersion2}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {versions.map((v: Version) => (
                  <SelectItem key={v.id} value={v.id}>
                    {formatDate(v.timestamp)} - {v.user}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {version2 && (
              <p className="text-xs text-muted-foreground">
                {formatDate(version2.timestamp)} • {version2.user}
              </p>
            )}
          </div>
        </div>

        {version1 && version2 && (
          <div className="border rounded-lg p-4 space-y-3">
            <div className="flex items-center gap-2 mb-4">
              <ArrowLeftRight className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium">Изменения</span>
            </div>

            {fields.map(field => {
              const diff = getDiff(field)
              if (!diff) return null

              if (diff.type === 'unchanged') {
                return (
                  <div key={field} className="flex items-center justify-between p-2 bg-muted/30 rounded">
                    <span className="text-sm font-medium">{field}:</span>
                    <span className="text-sm">{String(diff.value || '—')}</span>
                  </div>
                )
              }

              return (
                <div key={field} className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium">{field}:</span>
                    <Badge variant="outline" className="text-xs">Изменено</Badge>
                  </div>
                  <div className="grid grid-cols-2 gap-2 pl-4">
                    <div className="p-2 bg-red-50 dark:bg-red-950/20 rounded border border-red-200 dark:border-red-900">
                      <div className="flex items-center gap-1 mb-1">
                        <Minus className="h-3 w-3 text-red-600" />
                        <span className="text-xs font-medium text-red-600">Было</span>
                      </div>
                      <p className="text-sm line-through">{String(diff.old || '—')}</p>
                    </div>
                    <div className="p-2 bg-green-50 dark:bg-green-950/20 rounded border border-green-200 dark:border-green-900">
                      <div className="flex items-center gap-1 mb-1">
                        <Plus className="h-3 w-3 text-green-600" />
                        <span className="text-xs font-medium text-green-600">Стало</span>
                      </div>
                      <p className="text-sm font-medium">{String(diff.new || '—')}</p>
                    </div>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

