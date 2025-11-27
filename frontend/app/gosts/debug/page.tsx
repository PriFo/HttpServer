'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Code, RefreshCw, Database, FileText, BarChart3 } from 'lucide-react'
import { toast } from 'sonner'
import Link from 'next/link'
import { ArrowLeft } from 'lucide-react'

/**
 * Страница для отладки и проверки данных ГОСТов
 * Показывает структуру ответов API и количество загруженных ГОСТов
 */
export default function GostsDebugPage() {
  const [apiResponse, setApiResponse] = useState<any>(null)
  const [statisticsResponse, setStatisticsResponse] = useState<any>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchData = async () => {
    setLoading(true)
    setError(null)

    try {
      // Получаем список ГОСТов
      const gostsResponse = await fetch('/api/gosts?limit=5&offset=0')
      if (!gostsResponse.ok) {
        throw new Error(`Failed to fetch GOSTs: ${gostsResponse.status}`)
      }
      const gostsData = await gostsResponse.json()
      setApiResponse(gostsData)

      // Получаем статистику
      const statsResponse = await fetch('/api/gosts/statistics')
      if (statsResponse.ok) {
        const statsData = await statsResponse.json()
        setStatisticsResponse(statsData)
      }

      toast.success('Данные загружены')
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Ошибка загрузки'
      setError(errorMessage)
      toast.error('Ошибка', {
        description: errorMessage,
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Link href="/gosts">
            <Button variant="outline" size="icon">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </Link>
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Database className="w-8 h-8" />
              Отладка ГОСТов
            </h1>
            <p className="text-muted-foreground mt-2">
              Проверка структуры данных и количества загруженных ГОСТов
            </p>
          </div>
        </div>
        <Button onClick={fetchData} disabled={loading}>
          <RefreshCw className={`h-4 w-4 mr-2 ${loading ? 'animate-spin' : ''}`} />
          Обновить
        </Button>
      </div>

      {error && (
        <Card className="border-destructive">
          <CardHeader>
            <CardTitle className="text-destructive">Ошибка</CardTitle>
          </CardHeader>
          <CardContent>
            <p>{error}</p>
          </CardContent>
        </Card>
      )}

      {/* Статистика */}
      {statisticsResponse && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <BarChart3 className="h-5 w-5" />
              Статистика базы данных
            </CardTitle>
            <CardDescription>
              Общая информация о загруженных ГОСТах
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                {statisticsResponse.total_gosts !== undefined && (
                  <div className="p-4 bg-muted rounded-lg border">
                    <div className="flex items-center gap-2 mb-2">
                      <FileText className="h-5 w-5 text-blue-600" />
                      <p className="text-sm text-muted-foreground">Всего ГОСТов</p>
                    </div>
                    <p className="text-3xl font-bold">{statisticsResponse.total_gosts.toLocaleString('ru-RU')}</p>
                  </div>
                )}
                {statisticsResponse.total_sources !== undefined && (
                  <div className="p-4 bg-muted rounded-lg border">
                    <div className="flex items-center gap-2 mb-2">
                      <Database className="h-5 w-5 text-green-600" />
                      <p className="text-sm text-muted-foreground">Источников</p>
                    </div>
                    <p className="text-3xl font-bold">{statisticsResponse.total_sources.toLocaleString('ru-RU')}</p>
                  </div>
                )}
                {statisticsResponse.total_documents !== undefined && (
                  <div className="p-4 bg-muted rounded-lg border">
                    <div className="flex items-center gap-2 mb-2">
                      <FileText className="h-5 w-5 text-purple-600" />
                      <p className="text-sm text-muted-foreground">Документов</p>
                    </div>
                    <p className="text-3xl font-bold">{statisticsResponse.total_documents.toLocaleString('ru-RU')}</p>
                  </div>
                )}
                {statisticsResponse.by_status && (
                  <div className="p-4 bg-muted rounded-lg border">
                    <div className="flex items-center gap-2 mb-2">
                      <BarChart3 className="h-5 w-5 text-orange-600" />
                      <p className="text-sm text-muted-foreground">Действующих</p>
                    </div>
                    <p className="text-3xl font-bold">
                      {(statisticsResponse.by_status['действующий'] || statisticsResponse.by_status['действует'] || 0).toLocaleString('ru-RU')}
                    </p>
                  </div>
                )}
              </div>

              {statisticsResponse.by_status && Object.keys(statisticsResponse.by_status).length > 0 && (
                <div className="mt-4 p-4 bg-muted rounded-lg">
                  <p className="text-sm font-medium mb-3">Распределение по статусам:</p>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                    {Object.entries(statisticsResponse.by_status)
                      .sort(([, a], [, b]) => (b as number) - (a as number))
                      .map(([status, count]) => (
                        <div key={status} className="flex justify-between items-center p-2 bg-background rounded">
                          <span className="text-sm">{status}</span>
                          <span className="font-bold">{count as number}</span>
                        </div>
                      ))}
                  </div>
                </div>
              )}

              {statisticsResponse.by_source_type && Object.keys(statisticsResponse.by_source_type).length > 0 && (
                <div className="mt-4 p-4 bg-muted rounded-lg">
                  <p className="text-sm font-medium mb-3">Распределение по источникам (топ-20):</p>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-2 max-h-96 overflow-y-auto">
                    {Object.entries(statisticsResponse.by_source_type)
                      .sort(([, a], [, b]) => (b as number) - (a as number))
                      .slice(0, 20)
                      .map(([source, count]) => (
                        <div key={source} className="flex justify-between items-center p-2 bg-background rounded">
                          <span className="text-sm truncate flex-1">{source}</span>
                          <span className="font-bold ml-2">{count as number}</span>
                        </div>
                      ))}
                  </div>
                  {Object.keys(statisticsResponse.by_source_type).length > 20 && (
                    <p className="text-xs text-muted-foreground mt-2">
                      И еще {Object.keys(statisticsResponse.by_source_type).length - 20} источников...
                    </p>
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Структура ответа API */}
      {apiResponse && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Code className="h-5 w-5" />
              Структура ответа API /api/gosts
            </CardTitle>
            <CardDescription>
              Как данные возвращаются с бэкенда
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                <div className="p-3 bg-muted rounded-lg border">
                  <p className="text-sm text-muted-foreground">Всего записей</p>
                  <p className="text-2xl font-bold">{apiResponse.total?.toLocaleString('ru-RU') || 0}</p>
                </div>
                <div className="p-3 bg-muted rounded-lg border">
                  <p className="text-sm text-muted-foreground">На странице</p>
                  <p className="text-2xl font-bold">{apiResponse.gosts?.length || 0}</p>
                </div>
                <div className="p-3 bg-muted rounded-lg border">
                  <p className="text-sm text-muted-foreground">Лимит</p>
                  <p className="text-2xl font-bold">{apiResponse.limit || 0}</p>
                </div>
                <div className="p-3 bg-muted rounded-lg border">
                  <p className="text-sm text-muted-foreground">Смещение</p>
                  <p className="text-2xl font-bold">{apiResponse.offset || 0}</p>
                </div>
              </div>

              {apiResponse.gosts && apiResponse.gosts.length > 0 && (
                <div className="mt-4">
                  <p className="text-sm font-medium mb-2">Пример первой записи:</p>
                  <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-xs border">
                    {JSON.stringify(apiResponse.gosts[0], null, 2)}
                  </pre>
                </div>
              )}

              <div className="mt-4">
                <p className="text-sm font-medium mb-2">Полная структура ответа:</p>
                <pre className="bg-muted p-4 rounded-lg overflow-x-auto text-xs max-h-96 overflow-y-auto border">
                  {JSON.stringify(apiResponse, null, 2)}
                </pre>
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      {loading && (
        <Card>
          <CardContent className="py-8 text-center">
            <RefreshCw className="h-8 w-8 animate-spin mx-auto mb-2" />
            <p className="text-muted-foreground">Загрузка данных...</p>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

