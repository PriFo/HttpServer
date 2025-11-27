'use client'

import { useState, useEffect } from 'react'
import { Badge } from '@/components/ui/badge'
import { Database } from 'lucide-react'
import Link from 'next/link'

interface DatabaseInfo {
  name: string
  status: string
}

export function DatabaseInfo() {
  const [dbInfo, setDbInfo] = useState<DatabaseInfo | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchDatabaseInfo = async () => {
      try {
        const response = await fetch('/api/database/info')
        if (response.ok) {
          try {
            const data = await response.json()
            setDbInfo(data)
          } catch (parseError) {
            // Если не удалось распарсить JSON, используем дефолтные значения
            setDbInfo({
              name: 'Сервисная БД',
              status: 'disconnected'
            })
          }
        } else {
          // Если ошибка, устанавливаем дефолтные значения для сервисной БД
          setDbInfo({
            name: 'Сервисная БД',
            status: 'disconnected'
          })
        }
      } catch (error) {
        // Логируем ошибку только если она содержит полезную информацию
        if (error instanceof Error && error.message) {
          console.error('Error fetching database info:', error.message)
        } else if (error && typeof error === 'object' && Object.keys(error).length > 0) {
          console.error('Error fetching database info:', error)
        }
        // При ошибке устанавливаем дефолтные значения для сервисной БД
        setDbInfo({
          name: 'Сервисная БД',
          status: 'disconnected'
        })
      } finally {
        setLoading(false)
      }
    }

    fetchDatabaseInfo()

    // Refresh database info every 30 seconds
    const interval = setInterval(fetchDatabaseInfo, 30000)
    return () => clearInterval(interval)
  }, [])

  if (loading || !dbInfo) {
    return (
      <Badge variant="outline" className="hidden sm:flex items-center gap-2">
        <div className="h-2 w-2 rounded-full bg-gray-400 animate-pulse"></div>
        <span className="text-xs">Загрузка...</span>
      </Badge>
    )
  }

  const isConnected = dbInfo.status === 'connected'

  return (
    <Link href="/databases">
      <Badge
        variant="outline"
        className="hidden sm:flex items-center gap-2 hover:bg-accent transition-colors cursor-pointer"
      >
        <div className={`h-2 w-2 rounded-full ${isConnected ? 'bg-green-500' : 'bg-red-500'}`}></div>
        <Database className="h-3 w-3" />
        <span className="text-xs">{dbInfo.name}</span>
      </Badge>
    </Link>
  )
}
