'use client'

import { Suspense, useState, useEffect } from 'react'
import { useSearchParams, useRouter, usePathname } from 'next/navigation'
import { DatabaseSelector } from '@/components/database-selector'
import { QualityViolationsTab } from '@/components/quality/quality-violations-tab'
import { AlertTriangle, CheckCircle2 } from 'lucide-react'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'

function ViolationsPageContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const pathname = usePathname()
  const [selectedDatabase, setSelectedDatabase] = useState<string>(searchParams.get('database') || '')

  // Update URL when database changes
  const handleDatabaseChange = (db: string) => {
    setSelectedDatabase(db)
    const params = new URLSearchParams(searchParams)
    if (db) {
      params.set('database', db)
    } else {
      params.delete('database')
    }
    router.replace(`${pathname}?${params.toString()}`)
  }

  useEffect(() => {
    const dbParam = searchParams.get('database')
    if (dbParam && dbParam !== selectedDatabase) {
      setSelectedDatabase(dbParam)
    }
  }, [searchParams])

  const breadcrumbItems = [
    { label: 'Качество', href: '/quality', icon: CheckCircle2 },
    { label: 'Нарушения', href: '/quality/violations', icon: AlertTriangle },
  ]

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <FadeIn>
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
          className="flex flex-col md:flex-row md:items-center justify-between gap-4"
        >
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <AlertTriangle className="w-8 h-8 text-orange-500" />
              Нарушения качества
            </h1>
            <p className="text-muted-foreground mt-1">
              Просмотр и управление нарушениями правил качества данных
            </p>
          </div>
        <div className="w-full md:w-[300px]">
            <DatabaseSelector
            value={selectedDatabase}
            onChange={handleDatabaseChange}
            />
        </div>
      </motion.div>
      </FadeIn>

      <QualityViolationsTab database={selectedDatabase} />
    </div>
  )
}

export default function ViolationsPage() {
  return (
    <Suspense fallback={<div className="container-wide mx-auto px-4 py-8">Загрузка...</div>}>
      <ViolationsPageContent />
    </Suspense>
  )
}
