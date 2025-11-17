'use client'

import { useEffect } from 'react'
import { useRouter, useSearchParams } from 'next/navigation'

// Редирект на новую страницу процессов
export default function ReclassificationPage() {
  const router = useRouter()
  const searchParams = useSearchParams()

  useEffect(() => {
    const params = new URLSearchParams(searchParams.toString())
    params.set('tab', 'reclassification')
    router.replace(`/processes?${params.toString()}`)
  }, [router, searchParams])

              return null
}
