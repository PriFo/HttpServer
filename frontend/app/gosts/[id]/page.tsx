'use client'

import { useState, useEffect } from 'react'
import { useParams, useRouter } from 'next/navigation'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { GostDetail } from '@/components/gosts/gost-detail'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { ArrowLeft, FileText } from 'lucide-react'
import Link from 'next/link'
import { motion } from 'framer-motion'
import { addToHistory } from '@/lib/gost-favorites'

interface Gost {
  id: number
  gost_number: string
  title: string
  adoption_date?: string
  effective_date?: string
  status?: string
  source_type?: string
  source_url?: string
  description?: string
  keywords?: string
  documents?: Array<{
    id: number
    file_path: string
    file_type: string
    file_size: number
    uploaded_at: string
  }>
  created_at?: string
  updated_at?: string
}

export default function GostDetailPage() {
  const params = useParams()
  const router = useRouter()
  const id = params.id as string

  const [gost, setGost] = useState<Gost | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) {
      setError('ID ГОСТа не указан')
      setLoading(false)
      return
    }

    const fetchGost = async () => {
      setLoading(true)
      setError(null)

      try {
        const response = await fetch(`/api/gosts/${id}`)
        
        if (!response.ok) {
          if (response.status === 404) {
            throw new Error('ГОСТ не найден')
          }
          const errorData = await response.json().catch(() => ({ error: 'Ошибка загрузки ГОСТа' }))
          throw new Error(errorData.error || 'Ошибка загрузки ГОСТа')
        }

        const data = await response.json()
        setGost(data)
        
        addToHistory({
          id: data.id,
          gost_number: data.gost_number,
          title: data.title,
        })
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Ошибка загрузки ГОСТа')
      } finally {
        setLoading(false)
      }
    }

    fetchGost()
  }, [id])

  const breadcrumbItems = [
    { label: 'ГОСТы', href: '/gosts', icon: FileText },
    { label: gost?.gost_number || 'Загрузка...', href: `/gosts/${id}`, icon: FileText },
  ]

  if (loading) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <LoadingState message="Загрузка ГОСТа..." size="lg" fullScreen />
      </div>
    )
  }

  if (error || !gost) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
        <div className="mb-4">
          <Breadcrumb items={breadcrumbItems} />
        </div>
        <ErrorState
          title="Ошибка загрузки"
          message={error || 'ГОСТ не найден'}
          action={{
            label: 'Вернуться к списку',
            onClick: () => router.push('/gosts'),
          }}
        />
      </div>
    )
  }

  return (
    <div className="container-wide mx-auto px-4 py-8 space-y-6">
      <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
      <div className="mb-4">
        <Breadcrumb items={breadcrumbItems} />
      </div>

      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.3 }}
        className="flex items-center gap-4"
      >
        <Link href="/gosts">
          <Button variant="outline" size="icon">
            <ArrowLeft className="h-4 w-4" />
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-bold">{gost.gost_number}</h1>
          <p className="text-muted-foreground mt-1">{gost.title}</p>
        </div>
      </motion.div>

      <GostDetail gost={gost} />
    </div>
  )
}

