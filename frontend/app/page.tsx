'use client'

import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import Link from "next/link"
import { Play, BarChart3, Database, ArrowRight, RefreshCw, Download, FileCode } from "lucide-react"
import { StatCard } from "@/components/common/stat-card"
import { LoadingState } from "@/components/common/loading-state"

interface DatabaseInfo {
  name: string
  status: string
  stats: {
    uploads_count: number
    catalogs_count: number
    items_count: number
  }
}

interface NormalizationStats {
  total_processed: number
  total_groups: number
  total_merged: number
}

export default function HomePage() {
  const [dbInfo, setDbInfo] = useState<DatabaseInfo | null>(null)
  const [normStats, setNormStats] = useState<NormalizationStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [downloadingXML, setDownloadingXML] = useState(false)

  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch database info
        const dbResponse = await fetch('/api/database/info')
        if (dbResponse.ok) {
          const dbData = await dbResponse.json()
          setDbInfo(dbData)
        }

        // Fetch normalization stats
        const normResponse = await fetch('/api/normalization/stats')
        if (normResponse.ok) {
          const normData = await normResponse.json()
          setNormStats(normData)
        }
      } catch (error) {
        console.error('Error fetching data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
  }, [])

  const handleDownloadXML = async () => {
    setDownloadingXML(true)
    try {
      const response = await fetch('/api/1c/processing/xml')
      
      if (!response.ok) {
        throw new Error('Не удалось загрузить XML файл')
      }

      // Получаем имя файла из заголовка Content-Disposition
      const contentDisposition = response.headers.get('Content-Disposition')
      let filename = `1c_processing_export_${new Date().toISOString().split('T')[0].replace(/-/g, '')}.xml`

      if (contentDisposition) {
        const filenameMatch = contentDisposition.match(/filename="?([^"]+)"?/)
        if (filenameMatch) {
          filename = filenameMatch[1]
        }
      }

      // Скачиваем файл
      const blob = await response.blob()
      const url = window.URL.createObjectURL(blob)
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      window.URL.revokeObjectURL(url)
    } catch (error) {
      console.error('Failed to download XML:', error)
      alert('Ошибка при скачивании XML файла. Проверьте подключение к серверу.')
    } finally {
      setDownloadingXML(false)
    }
  }

  return (
    <div className="container mx-auto p-6 space-y-8">
      {/* Hero Section */}
      <div className="text-center space-y-4 py-8">
        <h1 className="text-4xl font-bold tracking-tight">
          Нормализатор данных 1С
        </h1>
        <p className="text-xl text-muted-foreground max-w-2xl mx-auto">
          Автоматизированная система для нормализации и унификации справочных данных из 1С
        </p>
      </div>

      {/* Quick Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {loading ? (
          <>
            <LoadingState message="Загрузка..." size="sm" />
            <LoadingState message="Загрузка..." size="sm" />
            <LoadingState message="Загрузка..." size="sm" />
          </>
        ) : (
          <>
            <StatCard
              title="Записей в БД"
              value={dbInfo?.stats?.items_count || 0}
              description="Всего записей номенклатуры"
              icon={Database}
              formatValue={(val: number | string) => {
                const numVal = typeof val === 'number' ? val : Number(val);
                return numVal.toLocaleString('ru-RU');
              }}
            />

            <StatCard
              title="Текущая база данных"
              value={dbInfo?.status === 'connected' ? 'Подключена' : 'Отключена'}
              description={dbInfo?.name || 'Неизвестно'}
              variant={dbInfo?.status === 'connected' ? 'success' : 'default'}
            />

            <StatCard
              title="Версия системы"
              value="Stable"
              description="Производственная версия"
            />
          </>
        )}
      </div>

      {/* Main Actions */}
      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Normalization Card */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 rounded-full -mr-16 -mt-16"></div>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
                <Play className="h-5 w-5 text-primary" />
              </div>
              <div>
                <CardTitle>Нормализация данных</CardTitle>
                <CardDescription>
                  Запуск процесса нормализации
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <ul className="space-y-2 text-sm text-muted-foreground">
              <li className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4" />
                Автоматическая группировка записей
              </li>
              <li className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4" />
                Категоризация товаров
              </li>
              <li className="flex items-center gap-2">
                <ArrowRight className="h-4 w-4" />
                Мониторинг в реальном времени
              </li>
            </ul>
            <Button asChild className="w-full">
              <Link href="/processes?tab=normalization">
                <Play className="mr-2 h-4 w-4" />
                Управление нормализацией
              </Link>
            </Button>
          </CardContent>
        </Card>

        {/* Results Card */}
        <Card className="relative overflow-hidden">
          <div className="absolute top-0 right-0 w-32 h-32 bg-secondary/5 rounded-full -mr-16 -mt-16"></div>
          <CardHeader>
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-secondary/10">
                <BarChart3 className="h-5 w-5 text-secondary-foreground" />
              </div>
              <div>
                <CardTitle>Результаты и аналитика</CardTitle>
                <CardDescription>
                  Просмотр статистики нормализации
                </CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            {loading ? (
              <div className="flex items-center justify-center py-6">
                <RefreshCw className="h-5 w-5 animate-spin text-muted-foreground" />
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Обработано записей:</span>
                  <span className="font-semibold">{(normStats?.total_processed || 0).toLocaleString('ru-RU')}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Создано групп:</span>
                  <span className="font-semibold">{(normStats?.total_groups || 0).toLocaleString('ru-RU')}</span>
                </div>
                <div className="flex justify-between text-sm">
                  <span className="text-muted-foreground">Объединено записей:</span>
                  <span className="font-semibold">{(normStats?.total_merged || 0).toLocaleString('ru-RU')}</span>
                </div>
              </div>
            )}
            <Button asChild variant="outline" className="w-full">
              <Link href="/results">
                <BarChart3 className="mr-2 h-4 w-4" />
                Просмотр результатов
              </Link>
            </Button>
          </CardContent>
        </Card>
      </div>

      {/* 1C Processing XML Download Card */}
      <Card className="relative overflow-hidden border-primary/20">
        <div className="absolute top-0 right-0 w-32 h-32 bg-primary/5 rounded-full -mr-16 -mt-16"></div>
        <CardHeader>
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary/10">
              <FileCode className="h-5 w-5 text-primary" />
            </div>
            <div>
              <CardTitle>Обработка 1С</CardTitle>
              <CardDescription>
                Скачать актуальный XML файл обработки для импорта в конфигуратор
              </CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              Получите всегда актуальную версию XML файла обработки 1С, объединяющую все модули и расширения.
              Файл готов к импорту в конфигуратор 1С.
            </p>
            <Button 
              onClick={handleDownloadXML}
              disabled={downloadingXML}
              className="w-full"
            >
              {downloadingXML ? (
                <>
                  <RefreshCw className="mr-2 h-4 w-4 animate-spin" />
                  Генерация XML...
                </>
              ) : (
                <>
                  <Download className="mr-2 h-4 w-4" />
                  Скачать XML обработки
                </>
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Info Section */}
      <Card>
        <CardHeader>
          <CardTitle>О системе</CardTitle>
          <CardDescription>
            Нормализатор данных 1С - инструмент для автоматизации обработки справочных данных
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <div className="space-y-2">
              <h3 className="font-semibold text-sm">Возможности</h3>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Автоматическая нормализация наименований</li>
                <li>• Группировка похожих записей</li>
                <li>• Категоризация товаров</li>
                <li>• Экспорт результатов</li>
              </ul>
            </div>
            <div className="space-y-2">
              <h3 className="font-semibold text-sm">Поддерживаемые данные</h3>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Номенклатура</li>
                <li>• Контрагенты (в разработке)</li>
                <li>• Склады (в разработке)</li>
                <li>• Прочие справочники</li>
              </ul>
            </div>
            <div className="space-y-2">
              <h3 className="font-semibold text-sm">Технологии</h3>
              <ul className="text-sm text-muted-foreground space-y-1">
                <li>• Go backend</li>
                <li>• Next.js 16 frontend</li>
                <li>• SQLite database</li>
                <li>• Real-time processing</li>
              </ul>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
