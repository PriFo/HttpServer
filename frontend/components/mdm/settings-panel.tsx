'use client'

import React, { useState, useEffect } from 'react'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Settings, Save, RefreshCw, Bell, Database, Zap } from 'lucide-react'
import { toast } from 'sonner'

interface SettingsPanelProps {
  clientId: string
  projectId: string
  onSettingsChange?: (settings: any) => void
}

export const SettingsPanel: React.FC<SettingsPanelProps> = ({
  clientId,
  projectId,
  onSettingsChange,
}) => {
  const [settings, setSettings] = useState({
    // Общие настройки
    autoRefresh: true,
    refreshInterval: 5,
    showNotifications: true,
    soundEnabled: false,

    // Настройки нормализации
    defaultSimilarityThreshold: 0.85,
    maxClusterSize: 10,
    autoMerge: false,
    useAI: true,

    // Настройки экспорта
    defaultExportFormat: 'excel' as 'excel' | 'csv' | 'json',
    includeAttributes: true,
    includeMetadata: false,

    // Настройки производительности
    cacheEnabled: true,
    batchSize: 100,
    maxConcurrentRequests: 5,
  })

  const [isSaving, setIsSaving] = useState(false)
  const [hasChanges, setHasChanges] = useState(false)

  // Загрузка настроек с использованием useProjectState
  const { data: loadedSettings, loading: loadingSettings } = useProjectState(
    async (cid, pid, signal) => {
      const response = await fetch(
        `/api/clients/${cid}/projects/${pid}/settings`,
        { cache: 'no-store', signal }
      )
      if (!response.ok) {
        if (response.status === 404) {
          // Возвращаем настройки по умолчанию если endpoint не существует
          return {
            autoRefresh: true,
            refreshInterval: 5,
            showNotifications: true,
            soundEnabled: false,
            defaultSimilarityThreshold: 0.85,
            maxClusterSize: 10,
            autoMerge: false,
            useAI: true,
            defaultExportFormat: 'excel',
            includeAttributes: true,
            includeMetadata: false,
            cacheEnabled: true,
            batchSize: 100,
            maxConcurrentRequests: 5,
          }
        }
        throw new Error(`Failed to load settings: ${response.status}`)
      }
      return response.json()
    },
    clientId,
    projectId,
    [],
    {
      enabled: !!clientId && !!projectId,
      refetchInterval: null,
    }
  )

  // Синхронизация загруженных настроек с локальным состоянием
  useEffect(() => {
    if (loadedSettings && !hasChanges) {
      setSettings(loadedSettings)
    }
  }, [loadedSettings])

  const handleSettingChange = (key: string, value: any) => {
    setSettings(prev => ({ ...prev, [key]: value }))
    setHasChanges(true)
  }

  const handleSave = async () => {
    setIsSaving(true)
    try {
      const response = await fetch(
        `/api/clients/${clientId}/projects/${projectId}/settings`,
        {
          method: 'PUT',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(settings),
        }
      )

      if (!response.ok) {
        throw new Error(`Failed to save settings: ${response.status}`)
      }

      setHasChanges(false)
      toast.success('Настройки сохранены')
      
      if (onSettingsChange) {
        onSettingsChange(settings)
      }
    } catch (error) {
      console.error('Failed to save settings:', error)
      toast.error('Ошибка при сохранении настроек')
    } finally {
      setIsSaving(false)
    }
  }

  const handleReset = () => {
    // Сброс к значениям по умолчанию
    setSettings({
      autoRefresh: true,
      refreshInterval: 5,
      showNotifications: true,
      soundEnabled: false,
      defaultSimilarityThreshold: 0.85,
      maxClusterSize: 10,
      autoMerge: false,
      useAI: true,
      defaultExportFormat: 'excel',
      includeAttributes: true,
      includeMetadata: false,
      cacheEnabled: true,
      batchSize: 100,
      maxConcurrentRequests: 5,
    })
    setHasChanges(true)
  }

  if (loadingSettings) {
    return <LoadingState message="Загрузка настроек..." />
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Settings className="h-5 w-5" />
            <CardTitle className="text-base">Настройки</CardTitle>
          </div>
          {hasChanges && (
            <Badge variant="outline">Несохраненные изменения</Badge>
          )}
        </div>
        <CardDescription>
          Настройка параметров нормализации и интерфейса
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs defaultValue="general">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="general">
              <Settings className="h-4 w-4 mr-2" />
              Общие
            </TabsTrigger>
            <TabsTrigger value="normalization">
              <Database className="h-4 w-4 mr-2" />
              Нормализация
            </TabsTrigger>
            <TabsTrigger value="export">
              <Zap className="h-4 w-4 mr-2" />
              Экспорт
            </TabsTrigger>
            <TabsTrigger value="performance">
              <Zap className="h-4 w-4 mr-2" />
              Производительность
            </TabsTrigger>
          </TabsList>

          <TabsContent value="general" className="space-y-4 mt-4">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Автообновление</Label>
                  <p className="text-xs text-muted-foreground">
                    Автоматическое обновление данных
                  </p>
                </div>
                <Switch
                  checked={settings.autoRefresh}
                  onCheckedChange={(checked) => handleSettingChange('autoRefresh', checked)}
                />
              </div>

              {settings.autoRefresh && (
                <div className="space-y-2">
                  <Label>Интервал обновления (секунды)</Label>
                  <Input
                    type="number"
                    min="1"
                    max="60"
                    value={settings.refreshInterval}
                    onChange={(e) => handleSettingChange('refreshInterval', parseInt(e.target.value) || 5)}
                  />
                </div>
              )}

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Уведомления</Label>
                  <p className="text-xs text-muted-foreground">
                    Показывать уведомления о событиях
                  </p>
                </div>
                <Switch
                  checked={settings.showNotifications}
                  onCheckedChange={(checked) => handleSettingChange('showNotifications', checked)}
                />
              </div>

              {settings.showNotifications && (
                <div className="flex items-center justify-between">
                  <div className="space-y-0.5">
                    <Label>Звуковые уведомления</Label>
                    <p className="text-xs text-muted-foreground">
                      Воспроизводить звук при уведомлениях
                    </p>
                  </div>
                  <Switch
                    checked={settings.soundEnabled}
                    onCheckedChange={(checked) => handleSettingChange('soundEnabled', checked)}
                  />
                </div>
              )}
            </div>
          </TabsContent>

          <TabsContent value="normalization" className="space-y-4 mt-4">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Порог схожести по умолчанию</Label>
                <Input
                  type="number"
                  min="0"
                  max="1"
                  step="0.01"
                  value={settings.defaultSimilarityThreshold}
                  onChange={(e) => handleSettingChange('defaultSimilarityThreshold', parseFloat(e.target.value) || 0.85)}
                />
              </div>

              <div className="space-y-2">
                <Label>Максимальный размер кластера</Label>
                <Input
                  type="number"
                  min="2"
                  max="50"
                  value={settings.maxClusterSize}
                  onChange={(e) => handleSettingChange('maxClusterSize', parseInt(e.target.value) || 10)}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Автоматическое объединение</Label>
                  <p className="text-xs text-muted-foreground">
                    Автоматически объединять дубликаты
                  </p>
                </div>
                <Switch
                  checked={settings.autoMerge}
                  onCheckedChange={(checked) => handleSettingChange('autoMerge', checked)}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Использовать AI</Label>
                  <p className="text-xs text-muted-foreground">
                    Использовать искусственный интеллект для улучшения нормализации
                  </p>
                </div>
                <Switch
                  checked={settings.useAI}
                  onCheckedChange={(checked) => handleSettingChange('useAI', checked)}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="export" className="space-y-4 mt-4">
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Формат экспорта по умолчанию</Label>
                <Select
                  value={settings.defaultExportFormat}
                  onValueChange={(v: any) => handleSettingChange('defaultExportFormat', v)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="excel">Excel (.xlsx)</SelectItem>
                    <SelectItem value="csv">CSV (.csv)</SelectItem>
                    <SelectItem value="json">JSON (.json)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Включать атрибуты</Label>
                  <p className="text-xs text-muted-foreground">
                    Включать атрибуты в экспорт по умолчанию
                  </p>
                </div>
                <Switch
                  checked={settings.includeAttributes}
                  onCheckedChange={(checked) => handleSettingChange('includeAttributes', checked)}
                />
              </div>

              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Включать метаданные</Label>
                  <p className="text-xs text-muted-foreground">
                    Включать метаданные в экспорт по умолчанию
                  </p>
                </div>
                <Switch
                  checked={settings.includeMetadata}
                  onCheckedChange={(checked) => handleSettingChange('includeMetadata', checked)}
                />
              </div>
            </div>
          </TabsContent>

          <TabsContent value="performance" className="space-y-4 mt-4">
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>Кэширование</Label>
                  <p className="text-xs text-muted-foreground">
                    Включить кэширование данных
                  </p>
                </div>
                <Switch
                  checked={settings.cacheEnabled}
                  onCheckedChange={(checked) => handleSettingChange('cacheEnabled', checked)}
                />
              </div>

              <div className="space-y-2">
                <Label>Размер пакета</Label>
                <Input
                  type="number"
                  min="10"
                  max="1000"
                  value={settings.batchSize}
                  onChange={(e) => handleSettingChange('batchSize', parseInt(e.target.value) || 100)}
                />
                <p className="text-xs text-muted-foreground">
                  Количество записей для обработки за один запрос
                </p>
              </div>

              <div className="space-y-2">
                <Label>Максимум одновременных запросов</Label>
                <Input
                  type="number"
                  min="1"
                  max="20"
                  value={settings.maxConcurrentRequests}
                  onChange={(e) => handleSettingChange('maxConcurrentRequests', parseInt(e.target.value) || 5)}
                />
                <p className="text-xs text-muted-foreground">
                  Максимальное количество параллельных запросов к API
                </p>
              </div>
            </div>
          </TabsContent>
        </Tabs>

        <div className="flex gap-2 pt-4 mt-4 border-t">
          <Button onClick={handleSave} disabled={!hasChanges || isSaving} className="flex-1">
            <Save className="h-4 w-4 mr-2" />
            {isSaving ? 'Сохранение...' : 'Сохранить'}
          </Button>
          <Button variant="outline" onClick={handleReset}>
            <RefreshCw className="h-4 w-4 mr-2" />
            Сбросить
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

