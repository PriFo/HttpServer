'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'
import { LoadingState } from '@/components/common/loading-state'

/**
 * Страница моделей - редирект на бенчмарк
 * В будущем здесь может быть список моделей и другая информация
 */
export default function ModelsPage() {
  const router = useRouter()

  useEffect(() => {
    // Редирект на страницу бенчмарка
    router.replace('/models/benchmark')
  }, [router])

  return <LoadingState message="Перенаправление на страницу бенчмарка..." />
}

