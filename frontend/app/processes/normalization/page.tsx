'use client'

import { useState, useEffect } from 'react'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { NormalizationProcessCard } from '@/components/processes/normalization-process-card'
import { NormalizationHistory } from '@/components/processes/normalization-history'
import { NormalizationStats } from '@/components/processes/normalization-stats'
import { NormalizationPerformanceCharts } from '@/components/processes/normalization-performance-charts'
import { NormalizationPreviewStats } from '@/components/processes/normalization-preview-stats'
import { ProjectSelector } from '@/components/project-selector'
import { Package, Building2, PlayCircle } from 'lucide-react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { FadeIn } from '@/components/animations/fade-in'
import { motion } from 'framer-motion'

export default function NormalizationProcessesPage() {
  const [activeTab, setActiveTab] = useState<'overview' | 'nomenclature' | 'counterparties'>('overview')
  const [selectedProject, setSelectedProject] = useState<string>('')
  const [clientId, setClientId] = useState<number | null>(null)
  const [projectId, setProjectId] = useState<number | null>(null)
  
  // Обновляем clientId и projectId при изменении selectedProject
  useEffect(() => {
    if (selectedProject) {
      const parts = selectedProject.split(':')
      if (parts.length === 2) {
        const cId = parseInt(parts[0], 10)
        const pId = parseInt(parts[1], 10)
        if (!isNaN(cId) && !isNaN(pId)) {
          setClientId(cId)
          setProjectId(pId)
        } else {
          setClientId(null)
          setProjectId(null)
        }
      } else {
        setClientId(null)
        setProjectId(null)
      }
    } else {
      setClientId(null)
      setProjectId(null)
    }
  }, [selectedProject])
  
  const breadcrumbItems = [
    { label: 'Процессы', href: '/processes', icon: PlayCircle },
    { label: 'Нормализация', href: '/processes/normalization', icon: PlayCircle },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <div className="mb-8">
          <motion.h1 
            className="text-3xl font-bold mb-2"
            initial={{ opacity: 0, y: -20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5 }}
          >
            Процессы нормализации
          </motion.h1>
          <motion.p 
            className="text-muted-foreground"
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.5, delay: 0.1 }}
          >
            Управление и мониторинг процессов нормализации данных
          </motion.p>
        </div>
      </FadeIn>

      {/* Выбор клиента и проекта */}
      <FadeIn delay={0.15}>
        <Card className="mb-6">
          <CardHeader>
            <CardTitle>Выбор проекта</CardTitle>
            <CardDescription>
              Выберите клиента и проект для запуска процессов нормализации
            </CardDescription>
          </CardHeader>
          <CardContent>
            <ProjectSelector
              value={selectedProject}
              onChange={setSelectedProject}
              placeholder="Выберите проект"
              className="w-full"
            />
          </CardContent>
        </Card>
      </FadeIn>

      {/* Предварительная статистика */}
      {clientId && projectId && (
        <FadeIn delay={0.2}>
          <NormalizationPreviewStats
            clientId={clientId}
            projectId={projectId}
          />
        </FadeIn>
      )}

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as any)} className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="nomenclature">Номенклатура</TabsTrigger>
          <TabsTrigger value="counterparties">Контрагенты</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {!clientId || !projectId ? (
            <div className="p-4 bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 rounded-lg">
              <p className="text-sm text-yellow-800 dark:text-yellow-200">
                <strong>Внимание:</strong> Для запуска процессов нормализации выберите клиента и проект выше.
                Без выбора проекта процессы могут не запуститься или работать некорректно.
              </p>
            </div>
          ) : null}
          
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <NormalizationProcessCard
              title="Нормализация номенклатуры"
              description="Обработка и нормализация товаров и услуг"
              statusEndpoint={clientId && projectId 
                ? `/api/clients/${clientId}/projects/${projectId}/normalization/status`
                : "/api/normalization/status"}
              startEndpoint={clientId && projectId
                ? `/api/clients/${clientId}/projects/${projectId}/normalization/start`
                : "/api/normalization/start"}
              stopEndpoint="/api/normalization/stop"
              detailPagePath="/processes/nomenclature"
              icon={<Package className="h-6 w-6" />}
              clientId={clientId}
              projectId={projectId}
            />

            <NormalizationProcessCard
              title="Нормализация контрагентов"
              description="Обработка и нормализация данных контрагентов"
              statusEndpoint={clientId && projectId
                ? `/api/counterparties/normalization/status?client_id=${clientId}&project_id=${projectId}`
                : "/api/counterparties/normalization/status"}
              startEndpoint={clientId && projectId
                ? `/api/counterparties/normalization/start?client_id=${clientId}&project_id=${projectId}`
                : "/api/counterparties/normalization/start"}
              stopEndpoint="/api/counterparties/normalization/stop"
              detailPagePath="/processes/counterparties"
              icon={<Building2 className="h-6 w-6" />}
              clientId={clientId}
              projectId={projectId}
            />
          </div>

          <FadeIn delay={0.3}>
            <div>
              <h2 className="text-xl font-semibold mb-4">Общая статистика</h2>
              <NormalizationStats 
                type="nomenclature" 
                clientId={clientId}
                projectId={projectId}
              />
            </div>
          </FadeIn>
        </TabsContent>

        <TabsContent value="nomenclature" className="space-y-6">
          <NormalizationProcessCard
            title="Нормализация номенклатуры"
            description="Обработка и нормализация товаров и услуг"
            statusEndpoint={clientId && projectId 
              ? `/api/clients/${clientId}/projects/${projectId}/normalization/status`
              : "/api/normalization/status"}
            startEndpoint={clientId && projectId
              ? `/api/clients/${clientId}/projects/${projectId}/normalization/start`
              : "/api/normalization/start"}
            stopEndpoint="/api/normalization/stop"
            detailPagePath="/processes/nomenclature"
            icon={<Package className="h-6 w-6" />}
            clientId={clientId}
            projectId={projectId}
          />

          <NormalizationStats 
            type="nomenclature" 
            clientId={clientId}
            projectId={projectId}
          />
          
          <NormalizationPerformanceCharts 
            type="nomenclature" 
            clientId={clientId}
            projectId={projectId}
          />
          
          <NormalizationHistory 
            type="nomenclature" 
            clientId={clientId}
            projectId={projectId}
            limit={10} 
          />
        </TabsContent>

        <TabsContent value="counterparties" className="space-y-6">
          <NormalizationProcessCard
            title="Нормализация контрагентов"
            description="Обработка и нормализация данных контрагентов"
            statusEndpoint={clientId && projectId
              ? `/api/counterparties/normalization/status?client_id=${clientId}&project_id=${projectId}`
              : "/api/counterparties/normalization/status"}
            startEndpoint={clientId && projectId
              ? `/api/counterparties/normalization/start?client_id=${clientId}&project_id=${projectId}`
              : "/api/counterparties/normalization/start"}
            stopEndpoint="/api/counterparties/normalization/stop"
            detailPagePath="/processes/counterparties"
            icon={<Building2 className="h-6 w-6" />}
            clientId={clientId}
            projectId={projectId}
          />

          <NormalizationStats 
            type="counterparties" 
            clientId={clientId}
            projectId={projectId}
          />
          
          <NormalizationPerformanceCharts 
            type="counterparties" 
            clientId={clientId}
            projectId={projectId}
          />
          
          <NormalizationHistory 
            type="counterparties" 
            clientId={clientId}
            projectId={projectId}
            limit={10} 
          />
        </TabsContent>
      </Tabs>

      <FadeIn delay={0.2}>
        <motion.div
          className="mt-8 p-6 bg-muted/50 rounded-lg border"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
        >
          <h2 className="text-lg font-semibold mb-4">Быстрые действия</h2>
          <p className="text-sm text-muted-foreground mb-4">
            Переход к детальным страницам процессов
          </p>
          <div className="flex flex-wrap gap-2">
            <a
              href="/processes/nomenclature"
              className="text-sm text-primary hover:underline"
            >
              Детали нормализации номенклатуры →
            </a>
            <span className="text-muted-foreground">|</span>
            <a
              href="/processes/counterparties"
              className="text-sm text-primary hover:underline"
            >
              Детали нормализации контрагентов →
            </a>
          </div>
        </motion.div>
      </FadeIn>
    </div>
  )
}

