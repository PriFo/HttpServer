'use client'

import { useEffect } from 'react'
import { useRouter } from 'next/navigation'

// Редирект на новую страницу процессов
export default function NormalizationPage() {
  const router = useRouter()

  useEffect(() => {
    router.replace('/processes?tab=normalization')
  }, [router])

  return null
}
