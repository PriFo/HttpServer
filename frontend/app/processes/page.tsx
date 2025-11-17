'use client'

import { useState, useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'
// Неиспользуемые импорты удалены
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { DatabaseSelector } from '@/components/database-selector'
import dynamic from 'next/dynamic'

// Динамическая загрузка табов для уменьшения начального bundle
const NormalizationProcessTab = dynamic(
  () => import('@/components/processes/normalization-process-tab').then((mod) => ({ default: mod.NormalizationProcessTab })),
  { ssr: false }
)
const ReclassificationProcessTab = dynamic(
  () => import('@/components/processes/reclassification-process-tab').then((mod) => ({ default: mod.ReclassificationProcessTab })),
  { ssr: false }
)

export default function ProcessesPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  
  // Получаем значения из URL параметров
  const tabFromUrl = searchParams.get('tab') || 'normalization'
  const dbFromUrl = searchParams.get('database') || ''
  
  const [selectedDatabase, setSelectedDatabase] = useState<string>(dbFromUrl)
  const [activeTab, setActiveTab] = useState<string>(tabFromUrl)

  // Обновляем состояние при изменении URL параметров (асинхронно)
  useEffect(() => {
    const tab = searchParams.get('tab') || 'normalization'
    const db = searchParams.get('database') || ''
    
    // Обновляем состояние только если значения изменились
    if (tab !== activeTab) {
      // Используем requestAnimationFrame для асинхронного обновления
      requestAnimationFrame(() => {
        setActiveTab(tab)
      })
    }
    if (db !== selectedDatabase) {
      requestAnimationFrame(() => {
        setSelectedDatabase(db)
      })
    }
  }, [searchParams])

  const handleTabChange = (value: string) => {
    setActiveTab(value)
    // Обновляем URL без перезагрузки страницы
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', value)
    if (selectedDatabase) {
      params.set('database', selectedDatabase)
    }
    router.push(`/processes?${params.toString()}`, { scroll: false })
  }

  const handleDatabaseChange = (database: string) => {
    setSelectedDatabase(database)
    // Обновляем URL с новым database
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', activeTab)
    if (database) {
      params.set('database', database)
    } else {
      params.delete('database')
    }
    router.push(`/processes?${params.toString()}`, { scroll: false })
  }

  return (
    <div className="container mx-auto px-4 py-8">
      {/* Header with Database Selector */}
      <div className="mb-8 flex items-center justify-between gap-4">
        <div>
          <h1 className="text-3xl font-bold mb-2">Процессы обработки</h1>
          <p className="text-muted-foreground">
            Управление процессами нормализации и переклассификации данных
          </p>
        </div>
        <DatabaseSelector
          value={selectedDatabase}
          onChange={handleDatabaseChange}
          className="w-[300px]"
        />
      </div>

      {/* Tabs Navigation */}
      <Tabs value={activeTab} onValueChange={handleTabChange} className="space-y-6">
        <TabsList>
          <TabsTrigger value="normalization">Нормализация</TabsTrigger>
          <TabsTrigger value="reclassification">Переклассификация</TabsTrigger>
        </TabsList>

        <TabsContent value="normalization" className="space-y-6">
          <NormalizationProcessTab database={selectedDatabase} />
        </TabsContent>

        <TabsContent value="reclassification" className="space-y-6">
          <ReclassificationProcessTab database={selectedDatabase} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

