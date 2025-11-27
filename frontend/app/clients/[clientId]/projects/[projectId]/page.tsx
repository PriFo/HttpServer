'use client'

import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useSearchParams, useRouter } from 'next/navigation'
import Link from 'next/link'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Alert, AlertDescription } from "@/components/ui/alert"
import {
  ArrowLeft,
  Target,
  BarChart3,
  Play,
  FileText,
  RefreshCw,
  Database,
  Plus,
  Trash2,
  AlertCircle,
  Upload,
  X,
  Building2,
  BookOpen,
  Clock,
  Gauge,
  CheckCircle2,
  Activity,
  Eye,
  Table,
  Wrench
} from "lucide-react"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Progress } from "@/components/ui/progress"
import { PipelineStagesTab } from "./components/PipelineStagesTab"
import { LoadingState } from "@/components/common/loading-state"
import { EmptyState } from "@/components/common/empty-state"
import { normalizePercentage } from "@/lib/locale"
import { StatCard } from "@/components/common/stat-card"
import { UploadSpeedChart } from "@/components/upload/UploadSpeedChart"
import { Breadcrumb } from "@/components/ui/breadcrumb"
import { BreadcrumbList } from "@/components/seo/breadcrumb-list"
import { motion } from "framer-motion"
import { FadeIn } from "@/components/animations/fade-in"
import { toast } from 'sonner'
import { DatabaseDetailDialog } from "../../components/database-detail-dialog"

interface ProjectDetail {
  project: {
    id: number
    name: string
    project_type: string
    description: string
    status: string
    created_at: string
  }
  client_name?: string
  benchmarks: Array<{
    id: number
    normalized_name: string
    category: string
    is_approved: boolean
  }>
  statistics: {
    total_benchmarks: number
    approved_benchmarks: number
    avg_quality_score: number
  }
}

interface ProjectDatabase {
  id: number
  client_project_id: number
  name: string
  file_path: string
  description: string
  is_active: boolean
  file_size: number
  created_at: string
  updated_at: string
  tables?: Array<{
    name: string
    row_count?: number
  }>
  statistics?: {
    total_tables: number
    total_rows: number
  }
}

export default function ProjectDetailPage() {
  const params = useParams()
  const searchParams = useSearchParams()
  const router = useRouter()
  const clientId = params.clientId
  const projectId = params.projectId
  const [project, setProject] = useState<ProjectDetail | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [databases, setDatabases] = useState<ProjectDatabase[]>([])
  const [showAddDatabase, setShowAddDatabase] = useState(false)
  const [newDatabase, setNewDatabase] = useState({ name: '', file_path: '', description: '' })
  const [databaseError, setDatabaseError] = useState<string | null>(null)
  const [isAddingDatabase, setIsAddingDatabase] = useState(false)
  const [pendingDatabases, setPendingDatabases] = useState<Array<{ id: number; file_name: string; file_path: string }>>([])
  const [showPendingSelector, setShowPendingSelector] = useState(false)
  const [useCustomPath, setUseCustomPath] = useState(false)
  const [isDragging, setIsDragging] = useState(false)
  const [uploadedFile, setUploadedFile] = useState<{ file: File; suggestedName: string; filePath: string; nameRequired?: boolean } | null>(null)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadMetrics, setUploadMetrics] = useState<{
    startTime: string
    duration: number
    speed: number
    fileSize: number
  } | null>(null)
  const [uploadSpeedHistory, setUploadSpeedHistory] = useState<Array<{
    second: number
    speed: number
    bytesUploaded: number
  }>>([])
  const [multipleUploadProgress, setMultipleUploadProgress] = useState<{
    total: number
    completed: number
    current: string
    errors: Array<{ fileName: string; error: string }>
  } | null>(null)
  const [selectedDatabaseForDetail, setSelectedDatabaseForDetail] = useState<ProjectDatabase | null>(null)
  
  // Ref для debouncing fetchDatabases
  const fetchDatabasesTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const fetchDatabasesRetryCountRef = useRef<number>(0)
  const MAX_RETRY_ATTEMPTS = 3
  
  // Инициализируем активную вкладку из URL параметра или по умолчанию 'overview'
  const [activeTab, setActiveTab] = useState(() => {
    const tabFromUrl = searchParams?.get('tab') || 'overview'
    return tabFromUrl
  })

  // Обновляем активную вкладку при изменении URL параметра
  useEffect(() => {
    const tabFromUrl = searchParams?.get('tab')
    if (tabFromUrl && tabFromUrl !== activeTab) {
      setActiveTab(tabFromUrl)
    }
  }, [searchParams, activeTab])

  // Отслеживаем завершение множественной загрузки
  useEffect(() => {
    if (multipleUploadProgress && multipleUploadProgress.completed >= multipleUploadProgress.total) {
      const successCount = multipleUploadProgress.completed - multipleUploadProgress.errors.length
      if (successCount > 0) {
        toast.success(`Успешно загружено файлов: ${successCount} из ${multipleUploadProgress.total}`)
      }
      if (multipleUploadProgress.errors.length > 0) {
        toast.error(`Ошибок при загрузке: ${multipleUploadProgress.errors.length}`)
      }
      // Очищаем прогресс через 2 секунды
      const timer = setTimeout(() => {
        setMultipleUploadProgress(null)
      }, 2000)
      return () => clearTimeout(timer)
    }
  }, [multipleUploadProgress])

  const fetchProjectDetail = async (clientId: string, projectId: string) => {
    setIsLoading(true)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}`)
      if (!response.ok) throw new Error('Failed to fetch project details')
      const data = await response.json()
      setProject(data)
    } catch (error) {
      console.error('Failed to fetch project details:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const fetchDatabases = useCallback(async (immediate: boolean = false) => {
    if (!clientId || !projectId) return
    
    // Очищаем предыдущий таймер
    if (fetchDatabasesTimeoutRef.current) {
      clearTimeout(fetchDatabasesTimeoutRef.current)
      fetchDatabasesTimeoutRef.current = null
    }
    
    const doFetch = async () => {
      try {
        const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`)
        if (!response.ok) {
          if (response.status === 429) {
            // Проверяем количество попыток ПЕРЕД увеличением счетчика
            if (fetchDatabasesRetryCountRef.current >= MAX_RETRY_ATTEMPTS) {
              console.error('Max retry attempts reached for fetchDatabases')
              fetchDatabasesRetryCountRef.current = 0
              return
            }
            
            // Получаем задержку из заголовка Retry-After или используем по умолчанию
            const retryAfter = response.headers.get('Retry-After')
            const delay = retryAfter ? parseInt(retryAfter, 10) * 1000 : 2000
            
            fetchDatabasesRetryCountRef.current++
            console.warn(`Rate limit exceeded, retrying after ${delay}ms (attempt ${fetchDatabasesRetryCountRef.current}/${MAX_RETRY_ATTEMPTS})...`)
            
            // Повторяем попытку через указанное время
            fetchDatabasesTimeoutRef.current = setTimeout(() => {
              fetchDatabases(true)
            }, delay)
            return
          }
          // Сбрасываем счетчик при других ошибках
          fetchDatabasesRetryCountRef.current = 0
          throw new Error('Failed to fetch databases')
        }
        // Сбрасываем счетчик при успешном запросе
        fetchDatabasesRetryCountRef.current = 0
        const data = await response.json()
        setDatabases(data.databases || [])
      } catch (error) {
        // Сбрасываем счетчик при ошибке
        fetchDatabasesRetryCountRef.current = 0
        console.error('Failed to fetch databases:', error)
      }
    }
    
    if (immediate) {
      await doFetch()
    } else {
      // Debounce: ждем 500ms перед выполнением запроса
      fetchDatabasesTimeoutRef.current = setTimeout(doFetch, 500)
    }
  }, [clientId, projectId])

  const fetchPendingDatabases = async () => {
    try {
      const response = await fetch('/api/databases/pending?status=pending')
      if (response.ok) {
        const data = await response.json()
        setPendingDatabases((data.databases || []).map((db: { id: number; file_name: string; file_path: string }) => ({
          id: db.id,
          file_name: db.file_name,
          file_path: db.file_path,
        })))
      } else {
        // Не критичная ошибка - просто не показываем pending databases
        console.warn('Failed to fetch pending databases:', response.status)
      }
    } catch (error) {
      // Не критичная ошибка - просто не показываем pending databases
      console.warn('Failed to fetch pending databases:', error)
    }
  }

  useEffect(() => {
    if (clientId && projectId) {
      fetchProjectDetail(clientId as string, projectId as string)
      fetchDatabases()
      fetchPendingDatabases()
    }
    
    // Cleanup: очищаем таймер при размонтировании
    return () => {
      if (fetchDatabasesTimeoutRef.current) {
        clearTimeout(fetchDatabasesTimeoutRef.current)
        fetchDatabasesTimeoutRef.current = null
      }
    }
  }, [clientId, projectId, fetchDatabases])

  const handleAddDatabase = async () => {
    if (!newDatabase.name.trim() || !newDatabase.file_path.trim()) {
      setDatabaseError('Название и путь к файлу обязательны')
      return
    }

    setIsAddingDatabase(true)
    setDatabaseError(null)
    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(newDatabase)
      })

      if (!response.ok) {
        let errorMessage = 'Не удалось добавить базу данных'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          const errorText = await response.text().catch(() => '')
          errorMessage = errorText || `Ошибка сервера: ${response.status}`
        }
        setDatabaseError(errorMessage)
        return
      }

      setNewDatabase({ name: '', file_path: '', description: '' })
      setShowAddDatabase(false)
      setShowPendingSelector(false)
      setUseCustomPath(false)
      await fetchDatabases(true) // Немедленное обновление после добавления
      await fetchPendingDatabases()
    } catch (error) {
      console.error('Failed to add database:', error)
      setDatabaseError('Ошибка подключения к серверу')
    } finally {
      setIsAddingDatabase(false)
    }
  }

  const handleSelectPendingDatabase = (pendingDb: { id: number; file_name: string; file_path: string }) => {
    setNewDatabase({
      name: pendingDb.file_name,
      file_path: pendingDb.file_path,
      description: 'Автоматически добавлена из pending databases'
    })
    setShowPendingSelector(false)
    setUseCustomPath(true) // Делаем поле доступным для редактирования
  }

  const handleFileUpload = useCallback(async (file: File, autoConfirm: boolean = false) => {
    let metricsInterval: NodeJS.Timeout | undefined = undefined
    
    try {
      setIsUploading(true)
      setDatabaseError(null)
      setUploadMetrics(null) // Сбрасываем предыдущие метрики
      setUploadSpeedHistory([]) // Сбрасываем историю загрузки

      // Логируем информацию о файле для диагностики
      console.log('[Frontend] handleFileUpload: Начало обработки файла:', {
        name: file.name,
        size: file.size,
        type: file.type,
        lastModified: new Date(file.lastModified).toISOString(),
        nameLength: file.name.length,
        nameBytes: new TextEncoder().encode(file.name).length
      })

      // Валидация размера файла (максимум 500MB)
      const maxSize = 500 * 1024 * 1024 // 500MB
      const minSize = 1024 // Минимум 1KB
      if (file.size > maxSize) {
        const errorMsg = `Файл слишком большой (${(file.size / 1024 / 1024).toFixed(2)}MB). Максимальный размер: ${(maxSize / 1024 / 1024).toFixed(0)}MB`
        setDatabaseError(errorMsg)
        toast.error(errorMsg)
        setIsUploading(false)
        return
      }
      if (file.size < minSize) {
        const errorMsg = `Файл слишком маленький (${(file.size / 1024).toFixed(2)}KB). Минимальный размер: ${(minSize / 1024).toFixed(0)}KB`
        setDatabaseError(errorMsg)
        toast.error(errorMsg)
        setIsUploading(false)
        return
      }

      // Валидация типа файла
      const allowedExtensions = ['.db', '.sqlite', '.sqlite3']
      const fileExtension = file.name.toLowerCase().substring(file.name.lastIndexOf('.'))
      if (!allowedExtensions.includes(fileExtension)) {
        const errorMsg = `Неподдерживаемый тип файла. Разрешены: ${allowedExtensions.join(', ')}`
        setDatabaseError(errorMsg)
        toast.error(errorMsg)
        setIsUploading(false)
        return
      }

      // Валидация имени файла (избегаем проблемных символов)
      const invalidChars = /[<>:"|?*\x00-\x1f]/
      if (invalidChars.test(file.name)) {
        const errorMsg = 'Имя файла содержит недопустимые символы'
        setDatabaseError(errorMsg)
        toast.error(errorMsg)
        setIsUploading(false)
        return
      }
      
      // Дополнительная проверка: проверяем первые байты файла на клиенте (опционально)
      // Это можно сделать через FileReader, но для больших файлов это может быть медленно
      // Поэтому оставляем основную проверку на сервере

      const formData = new FormData()
      formData.append('file', file)
      // Если autoConfirm === true, создаем БД автоматически при загрузке
      // Если autoConfirm === false, показываем форму для подтверждения
      formData.append('auto_create', autoConfirm ? 'true' : 'false')

      const uploadStartTime = Date.now()
      const fileSizeMB = (file.size / 1024 / 1024).toFixed(2)
      console.log(`[Frontend] 📤 Начало загрузки файла: ${file.name} (${fileSizeMB} MB, ${file.size} байт)`)
      
      // Устанавливаем начальные метрики для отображения во время загрузки
      const startTimeISO = new Date(uploadStartTime).toISOString()
      setUploadMetrics({
        startTime: startTimeISO,
        duration: 0,
        speed: 0,
        fileSize: file.size
      })
      
      // Обновляем метрики в реальном времени во время загрузки и собираем историю по секундам
      let lastSecond = -1
      
      metricsInterval = setInterval(() => {
        const elapsed = (Date.now() - uploadStartTime) / 1000
        const currentSecond = Math.floor(elapsed)
        
        if (elapsed > 0) {
          const currentSpeed = parseFloat(fileSizeMB) / elapsed
          setUploadMetrics({
            startTime: startTimeISO,
            duration: elapsed,
            speed: currentSpeed,
            fileSize: file.size
          })
          
          // Собираем данные по секундам для графика
          if (currentSecond !== lastSecond && currentSecond > 0) {
            // Вычисляем приблизительное количество загруженных байт на основе времени и скорости
            // Используем более точную формулу: байты = скорость * время
            const estimatedBytesUploaded = Math.min(
              (currentSpeed * 1024 * 1024) * elapsed, // скорость в байтах/сек * время
              file.size
            )
            
            setUploadSpeedHistory(prev => {
              const newHistory = [...prev]
              // Обновляем или добавляем запись для текущей секунды
              const existingIndex = newHistory.findIndex(h => h.second === currentSecond)
              const historyEntry = {
                second: currentSecond,
                speed: currentSpeed,
                bytesUploaded: estimatedBytesUploaded
              }
              
              if (existingIndex >= 0) {
                newHistory[existingIndex] = historyEntry
              } else {
                newHistory.push(historyEntry)
              }
              
              return newHistory.sort((a, b) => a.second - b.second)
            })
            
            lastSecond = currentSecond
          }
        }
      }, 100) // Обновляем каждые 100мс

      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`, {
        method: 'POST',
        body: formData,
      })
      
      // Останавливаем обновление метрик после получения ответа
      if (metricsInterval) {
        clearInterval(metricsInterval)
      }

      const uploadDuration = ((Date.now() - uploadStartTime) / 1000).toFixed(2)
      console.log(`[Frontend] 📥 Получен ответ от сервера: статус ${response.status} (время: ${uploadDuration}s)`)

        if (!response.ok) {
          let errorMessage = 'Не удалось загрузить файл'
          let errorDetails = ''
          
          try {
            const errorData = await response.json()
            errorMessage = errorData.error || errorData.message || errorMessage
            errorDetails = errorData.details || errorData.detail || ''
          } catch {
            try {
              const errorText = await response.text()
              if (errorText) {
                try {
                  const errorJson = JSON.parse(errorText)
                  errorMessage = errorJson.error || errorJson.message || errorText
                  errorDetails = errorJson.details || errorJson.detail || ''
                } catch {
                  errorMessage = errorText || `Ошибка сервера: ${response.status}`
                }
              } else {
                errorMessage = `Ошибка сервера: ${response.status} ${response.statusText}`
              }
            } catch {
              errorMessage = `Ошибка сервера: ${response.status} ${response.statusText}`
            }
          }
          
          // Формируем полное сообщение об ошибке
          const fullErrorMessage = errorDetails 
            ? `${errorMessage}${errorDetails ? ` (${errorDetails})` : ''}`
            : errorMessage
          
          setDatabaseError(fullErrorMessage)
          toast.error('Ошибка загрузки файла', {
            description: fullErrorMessage,
            duration: 5000
          })
          setUploadMetrics(null) // Сбрасываем метрики при ошибке
          setUploadSpeedHistory([]) // Сбрасываем историю при ошибке
          // Останавливаем обновление метрик при ошибке
          if (metricsInterval) {
            clearInterval(metricsInterval)
          }
          setIsUploading(false)
          return
        }

        const data = await response.json()
        const totalDuration = ((Date.now() - uploadStartTime) / 1000).toFixed(2)
        const speedMBps = (parseFloat(fileSizeMB) / parseFloat(totalDuration)).toFixed(2)
        console.log(`[Frontend] ✅ Файл успешно загружен за ${totalDuration}s (скорость: ${speedMBps} MB/s):`, { 
          suggested_name: data.suggested_name, 
          file_path: data.file_path,
          file_size_mb: fileSizeMB
        })
        
        // Сохраняем метрики загрузки из ответа сервера или вычисляем на клиенте
        if (data.upload_metrics) {
          setUploadMetrics({
            startTime: data.upload_metrics.start_time || new Date(uploadStartTime).toISOString(),
            duration: data.upload_metrics.duration_sec || parseFloat(totalDuration),
            speed: data.upload_metrics.speed_mbps || parseFloat(speedMBps),
            fileSize: data.upload_metrics.file_size_bytes || file.size
          })
        } else {
          // Fallback: вычисляем метрики на клиенте
          setUploadMetrics({
            startTime: new Date(uploadStartTime).toISOString(),
            duration: parseFloat(totalDuration),
            speed: parseFloat(speedMBps),
            fileSize: file.size
          })
        }
        
        // Добавляем финальную точку в историю загрузки
        const finalSecond = Math.floor(parseFloat(totalDuration))
        if (finalSecond >= 0) {
          setUploadSpeedHistory(prev => {
            const newHistory = [...prev]
            const finalEntry = {
              second: finalSecond,
              speed: parseFloat(speedMBps),
              bytesUploaded: file.size
            }
            
            const existingIndex = newHistory.findIndex(h => h.second === finalSecond)
            if (existingIndex >= 0) {
              newHistory[existingIndex] = finalEntry
            } else {
              newHistory.push(finalEntry)
            }
            
            const sorted = newHistory.sort((a, b) => a.second - b.second)
            console.log(`[Frontend] 📊 История загрузки собрана: ${sorted.length} точек данных`, sorted)
            return sorted
          })
        }
        
        // Если auto_create был 'true', БД уже создана на backend
        // Проверяем наличие database в ответе
        if (data.database) {
          // БД успешно создана автоматически при загрузке
          console.log(`[Frontend] ✅ База данных автоматически создана: ID=${data.database.id}, название='${data.database.name}'`)
          // Очищаем ошибки при успешной загрузке
          setDatabaseError(null)
          // Обновляем список баз данных только если не множественная загрузка
          // (при множественной загрузке autoConfirm === true, обновление происходит один раз в конце)
          if (!autoConfirm) {
            await fetchDatabases(true)
            toast.success('База данных успешно добавлена', {
              description: `"${data.database.name}" добавлена в проект`
            })
          }
          // При множественной загрузке (autoConfirm === true) обновление и уведомления
          // происходят один раз в конце в handleDrop/handleFileInput
        } else {
          // БД не была создана автоматически, требуется подтверждение
          // Проверяем, требуется ли ввод имени от пользователя
          const nameRequired = data.name_required === true || !data.suggested_name || data.suggested_name === file.name.replace('.db', '')
          
          // Если autoConfirm и имя не требуется, автоматически подтверждаем загрузку
          if (autoConfirm && !nameRequired) {
            // Автоматически создаем базу данных с предложенным именем
            try {
              const confirmResponse = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`, {
                method: 'POST',
                headers: {
                  'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                  name: data.suggested_name || file.name.replace('.db', ''),
                  file_path: data.file_path,
                  description: data.description || ''
                })
              })

              if (confirmResponse.ok) {
                // Успешно добавлено
                const confirmData = await confirmResponse.json()
                console.log(`[Frontend] ✅ База данных создана после подтверждения: ID=${confirmData.id}, название='${confirmData.name}'`)
                // Очищаем ошибки при успешной загрузке
                setDatabaseError(null)
                // Обновляем список баз данных только если не множественная загрузка
                if (!autoConfirm) {
                  await fetchDatabases(true)
                  toast.success('База данных успешно добавлена', {
                    description: `"${confirmData.name}" добавлена в проект`
                  })
                }
                // При множественной загрузке обновление происходит один раз в конце
              } else {
                const errorText = await confirmResponse.text().catch(() => 'Неизвестная ошибка')
                console.error('Failed to auto-confirm database:', errorText)
                // Пробрасываем ошибку для обработки в множественной загрузке
                throw new Error(`Не удалось создать базу данных: ${errorText}`)
              }
            } catch (error) {
              console.error('Error auto-confirming database:', error)
              // Пробрасываем ошибку дальше для обработки в множественной загрузке
              throw error
            }
          } else {
          // Показываем форму с предложенным названием для ручного подтверждения
          // Это происходит если: !autoConfirm или nameRequired
          setUploadedFile({
            file,
            suggestedName: data.suggested_name || file.name.replace('.db', ''),
            filePath: data.file_path,
            nameRequired: nameRequired
          })
          setNewDatabase({
            name: nameRequired ? '' : (data.suggested_name || file.name.replace('.db', '')),
            file_path: data.file_path,
            description: data.description || ''
          })
          setShowAddDatabase(true)
          setUseCustomPath(true)
        }
      }
    } catch (error) {
      // Останавливаем обновление метрик при ошибке
      if (typeof metricsInterval !== 'undefined' && metricsInterval) {
        clearInterval(metricsInterval)
      }
      
      console.error('[Frontend] Error uploading file:', error)
      let errorMessage = 'Не удалось загрузить файл. Проверьте подключение к серверу.'
      
      if (error instanceof Error) {
        // Проверяем тип ошибки
        if (error.message.includes('Failed to fetch') || error.message.includes('NetworkError')) {
          errorMessage = 'Ошибка сети. Проверьте подключение к серверу и попробуйте снова.'
        } else if (error.message.includes('timeout') || error.message.includes('aborted')) {
          errorMessage = 'Время ожидания истекло. Файл может быть слишком большим. Попробуйте еще раз.'
        } else {
          errorMessage = error.message
        }
      }
      
      setDatabaseError(errorMessage)
      setUploadMetrics(null) // Сбрасываем метрики при ошибке
      setUploadSpeedHistory([]) // Сбрасываем историю при ошибке
    } finally {
      // Убеждаемся, что интервал очищен
      if (typeof metricsInterval !== 'undefined' && metricsInterval) {
        clearInterval(metricsInterval)
      }
      setIsUploading(false)
    }
  }, [clientId, projectId, fetchDatabases, fetchPendingDatabases])

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(true)
  }, [])

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }, [])

  const handleDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)

    try {
      const files = Array.from(e.dataTransfer.files)
      console.log('[Frontend] handleDrop: Получены файлы:', files.map(f => ({
        name: f.name,
        size: f.size,
        type: f.type
      })))
      
      if (files.length === 0) {
        setDatabaseError('Не удалось получить файлы. Попробуйте еще раз.')
        return
      }

      // Фильтруем только .db файлы
      const dbFiles = files.filter(file => file.name.endsWith('.db'))

      if (dbFiles.length === 0) {
        setDatabaseError('Пожалуйста, перетащите файлы базы данных (.db)')
        return
      }

      // Загружаем все .db файлы последовательно с автоматическим подтверждением
      const isMultiple = dbFiles.length > 1
      
      if (isMultiple) {
        // Инициализируем прогресс для множественной загрузки
        setMultipleUploadProgress({
          total: dbFiles.length,
          completed: 0,
          current: dbFiles[0].name,
          errors: []
        })
      }
      
      for (let i = 0; i < dbFiles.length; i++) {
        const dbFile = dbFiles[i]
        try {
          if (isMultiple) {
            setMultipleUploadProgress(prev => prev ? {
              ...prev,
              current: dbFile.name,
              completed: i
            } : null)
          }
          await handleFileUpload(dbFile, isMultiple) // autoConfirm = true для множественной загрузки
          
          if (isMultiple) {
            setMultipleUploadProgress(prev => prev ? {
              ...prev,
              completed: i + 1
            } : null)
          }
        } catch (error) {
          if (isMultiple) {
            setMultipleUploadProgress(prev => prev ? {
              ...prev,
              errors: [...(prev.errors || []), {
                fileName: dbFile.name,
                error: error instanceof Error ? error.message : 'Неизвестная ошибка'
              }]
            } : null)
          }
        }
      }
      
      // Обновляем список баз данных один раз после завершения всех загрузок
      if (isMultiple) {
        await fetchDatabases(true) // Обновляем список после всех загрузок
        await fetchPendingDatabases()
      }
      
      // Очищаем прогресс после завершения всех загрузок
      if (isMultiple) {
        setTimeout(() => {
          setMultipleUploadProgress(prev => {
            if (prev && prev.completed >= prev.total) {
              const successCount = prev.completed - prev.errors.length
              if (successCount > 0) {
                toast.success(`Успешно загружено файлов: ${successCount} из ${prev.total}`)
              }
              if (prev.errors.length > 0) {
                toast.error(`Ошибок при загрузке: ${prev.errors.length}`)
              }
              return null
            }
            return prev
          })
        }, 500)
      }
    } catch (error) {
      console.error('[Frontend] handleDrop: Ошибка при обработке файла:', error)
      setDatabaseError(`Ошибка при перетаскивании файла: ${error instanceof Error ? error.message : 'Неизвестная ошибка'}`)
    }
  }, [handleFileUpload])

  const handleFileInput = useCallback(async (e: React.ChangeEvent<HTMLInputElement>) => {
    try {
      const files = e.target.files
      if (!files || files.length === 0) {
        console.log('[Frontend] handleFileInput: Нет файлов')
        return
      }

      // Фильтруем только .db файлы
      const dbFiles = Array.from(files).filter(file => file.name.endsWith('.db'))

      if (dbFiles.length === 0) {
        setDatabaseError('Пожалуйста, выберите файлы базы данных (.db)')
        return
      }

      // Валидация размера всех файлов перед началом загрузки
      const maxSize = 500 * 1024 * 1024 // 500MB
      const oversizedFiles = dbFiles.filter(file => file.size > maxSize)
      if (oversizedFiles.length > 0) {
        setDatabaseError(
          `Следующие файлы слишком большие (максимум ${(maxSize / 1024 / 1024).toFixed(0)}MB): ${oversizedFiles.map(f => f.name).join(', ')}`
        )
        return
      }

      console.log('[Frontend] handleFileInput: Выбрано файлов:', dbFiles.length, dbFiles.map(f => ({
        name: f.name,
        size: f.size,
        type: f.type,
        lastModified: new Date(f.lastModified).toISOString()
      })))

      // Загружаем все .db файлы последовательно с автоматическим подтверждением
      const isMultiple = dbFiles.length > 1
      
      if (isMultiple) {
        // Инициализируем прогресс для множественной загрузки
        setMultipleUploadProgress({
          total: dbFiles.length,
          completed: 0,
          current: dbFiles[0].name,
          errors: []
        })
      }
      
      for (let i = 0; i < dbFiles.length; i++) {
        const dbFile = dbFiles[i]
        try {
          if (isMultiple) {
            setMultipleUploadProgress(prev => prev ? {
              ...prev,
              current: dbFile.name,
              completed: i
            } : null)
          }
          await handleFileUpload(dbFile, isMultiple) // autoConfirm = true для множественной загрузки
          
          if (isMultiple) {
            setMultipleUploadProgress(prev => prev ? {
              ...prev,
              completed: i + 1
            } : null)
          }
        } catch (error) {
          if (isMultiple) {
            setMultipleUploadProgress(prev => prev ? {
              ...prev,
              errors: [...(prev.errors || []), {
                fileName: dbFile.name,
                error: error instanceof Error ? error.message : 'Неизвестная ошибка'
              }]
            } : null)
          }
        }
      }
      
      // Обновляем список баз данных один раз после завершения всех загрузок
      if (isMultiple) {
        await fetchDatabases(true) // Обновляем список после всех загрузок
        await fetchPendingDatabases()
      }
      
      // Очищаем прогресс после завершения всех загрузок
      if (isMultiple) {
        setTimeout(() => {
          setMultipleUploadProgress(prev => {
            if (prev && prev.completed >= prev.total) {
              const successCount = prev.completed - prev.errors.length
              if (successCount > 0) {
                toast.success(`Успешно загружено файлов: ${successCount} из ${prev.total}`)
              }
              if (prev.errors.length > 0) {
                toast.error(`Ошибок при загрузке: ${prev.errors.length}`)
              }
              return null
            }
            return prev
          })
        }, 500)
      }
    } catch (error) {
      console.error('[Frontend] handleFileInput: Ошибка при обработке файла:', error)
      setDatabaseError(`Ошибка при выборе файла: ${error instanceof Error ? error.message : 'Неизвестная ошибка'}`)
    } finally {
      // Сбрасываем значение input, чтобы можно было выбрать тот же файл снова
      if (e.target) {
        e.target.value = ''
      }
    }
  }, [handleFileUpload])

  const handleConfirmUpload = async () => {
    if (!uploadedFile) return

    const finalName = newDatabase.name.trim() || (uploadedFile.nameRequired ? '' : uploadedFile.suggestedName)
    if (!finalName) {
      setDatabaseError('Название базы данных обязательно. Пожалуйста, введите название базы данных.')
      return
    }

    setIsAddingDatabase(true)
    setDatabaseError(null)

    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          name: finalName,
          file_path: uploadedFile.filePath,
          description: newDatabase.description
        })
      })

      if (!response.ok) {
        let errorMessage = 'Не удалось добавить базу данных'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          const errorText = await response.text().catch(() => '')
          errorMessage = errorText || `Ошибка сервера: ${response.status}`
        }
        setDatabaseError(errorMessage)
        return
      }

      // Успешно добавлено
      setUploadedFile(null)
      setNewDatabase({ name: '', file_path: '', description: '' })
      setShowAddDatabase(false)
      setShowPendingSelector(false)
      setUseCustomPath(false)
      await fetchDatabases(true) // Немедленное обновление после подтверждения загрузки
      await fetchPendingDatabases()
    } catch (error) {
      console.error('Failed to add database:', error)
      setDatabaseError('Ошибка подключения к серверу')
    } finally {
      setIsAddingDatabase(false)
    }
  }

  const handleDeleteDatabase = async (dbId: number) => {
    if (!confirm('Вы уверены, что хотите удалить эту базу данных?')) {
      return
    }

    try {
      const response = await fetch(`/api/clients/${clientId}/projects/${projectId}/databases/${dbId}`, {
        method: 'DELETE'
      })

      if (!response.ok) {
        let errorMessage = 'Не удалось удалить базу данных'
        try {
          const errorData = await response.json()
          errorMessage = errorData.error || errorMessage
        } catch {
          const errorText = await response.text().catch(() => '')
          errorMessage = errorText || `Ошибка сервера: ${response.status}`
        }
        alert(errorMessage)
        return
      }

      await fetchDatabases(true) // Немедленное обновление после удаления
    } catch (error) {
      console.error('Failed to delete database:', error)
      alert('Ошибка подключения к серверу')
    }
  }

  const getProjectTypeLabel = (type: string) => {
    const labels: Record<string, string> = {
      nomenclature: 'Номенклатура',
      counterparties: 'Контрагенты',
      nomenclature_counterparties: 'Номенклатура + Контрагенты',
      mixed: 'Смешанный'
    }
    return labels[type] || type
  }

const projectInfo = project?.project
const projectType = projectInfo?.project_type ?? ''

  // Состояние для классификаторов проекта
  const [projectClassifiers, setProjectClassifiers] = useState<Array<{ id: number; name: string; description: string }>>([])
  const [loadingClassifiers, setLoadingClassifiers] = useState(false)

  // Загружаем классификаторы для типа проекта
  useEffect(() => {
    if (projectType === 'nomenclature_counterparties') {
      setLoadingClassifiers(true)
      fetch(`/api/classification/classifiers/by-project-type?project_type=${projectType}`)
        .then(res => res.ok ? res.json() : null)
        .then(data => {
          if (data) {
            setProjectClassifiers(data.classifiers || [])
          }
        })
        .catch(err => console.error('Failed to fetch classifiers:', err))
        .finally(() => setLoadingClassifiers(false))
    }
  }, [projectType])

  if (isLoading) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <LoadingState message="Загрузка данных проекта..." size="lg" fullScreen />
      </div>
    )
  }

  if (!project) {
    const breadcrumbItems = [
      { label: 'Клиенты', href: '/clients', icon: Building2 },
      { label: 'Проекты', href: `/clients/${clientId}/projects`, icon: Target },
    ]

    return (
      <div className="container-wide mx-auto px-4 py-8">
        <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
        <div className="mb-4">
          <Breadcrumb items={breadcrumbItems} />
        </div>
        <EmptyState
          icon={Target}
          title="Проект не найден"
          description="Проект не существует или был удален"
        />
      </div>
    )
  }

  if (!projectInfo) {
    const breadcrumbItems = [
      { label: 'Клиенты', href: '/clients', icon: Building2 },
      { label: 'Проекты', href: `/clients/${clientId}/projects`, icon: Target },
    ]

    return (
      <div className="container-wide mx-auto px-4 py-8">
        <BreadcrumbList items={breadcrumbItems.map(item => ({ label: item.label, href: item.href || '#' }))} />
        <div className="mb-4">
          <Breadcrumb items={breadcrumbItems} />
        </div>
        <EmptyState
          icon={Target}
          title="Данные проекта недоступны"
          description="Бэкенд вернул некорректный ответ. Попробуйте обновить страницу или повторить позже."
        />
      </div>
    )
  }

const breadcrumbItems = [
  { label: 'Клиенты', href: '/clients', icon: Building2 },
  { label: project.client_name || 'Клиент', href: `/clients/${clientId}`, icon: Building2 },
  { label: projectInfo.name, href: `#`, icon: Target },
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
          className="flex items-center gap-4"
        >
          <Button 
            variant="outline" 
            size="icon"
            onClick={() => router.push(`/clients/${clientId}`)}
            aria-label="Назад к клиенту"
          >
            <ArrowLeft className="h-4 w-4" />
          </Button>
          <div className="flex-1">
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Target className="h-8 w-8 text-primary" />
              {projectInfo.name}
            </h1>
            <p className="text-muted-foreground mt-1">{projectInfo.description}</p>
          </div>
        <div className="flex gap-2">
          <Button asChild>
            <Link href={`/clients/${clientId}/projects/${projectId}/normalization`}>
              <Play className="mr-2 h-4 w-4" />
              Запустить нормализацию
            </Link>
          </Button>
          <Button variant="outline" asChild>
            <Link href={`/clients/${clientId}/projects/${projectId}/diagnostics`}>
              <Wrench className="mr-2 h-4 w-4" />
              Диагностика
            </Link>
          </Button>
        </div>
      </motion.div>
      </FadeIn>

      {/* Tabs для разных разделов проекта */}
      <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-6">
        <TabsList>
          <TabsTrigger value="overview">Обзор</TabsTrigger>
          <TabsTrigger value="databases">Базы данных</TabsTrigger>
          {(projectType === 'nomenclature' || projectType === 'normalization' || projectType === 'nomenclature_counterparties') && (
            <TabsTrigger value="pipeline-stages">Этапы обработки</TabsTrigger>
          )}
        </TabsList>

        <TabsContent value="overview" className="space-y-6">
          {/* Статистика */}
          <div className={`grid gap-6 ${projectType === 'nomenclature_counterparties' ? 'grid-cols-1 md:grid-cols-4' : 'grid-cols-1 md:grid-cols-3'}`}>
            <StatCard
              title="Всего эталонов"
              value={project.statistics.total_benchmarks}
              description={`${project.statistics.approved_benchmarks} утверждено`}
              icon={FileText}
              variant="primary"
            />
            <StatCard
              title="Среднее качество"
              value={`${Math.round(normalizePercentage(project.statistics.avg_quality_score))}%`}
              description="качество эталонов"
              variant={(() => {
                const normalized = normalizePercentage(project.statistics.avg_quality_score)
                return normalized >= 90 ? 'success' : normalized >= 70 ? 'warning' : 'danger'
              })()}
              progress={normalizePercentage(project.statistics.avg_quality_score)}
            />
            <Card>
              <CardHeader className="pb-2">
                <CardTitle className="text-sm font-medium">Тип проекта</CardTitle>
              </CardHeader>
              <CardContent>
                <Badge variant="outline" className="text-lg">
                  {getProjectTypeLabel(projectType || '')}
                </Badge>
              </CardContent>
            </Card>
            {projectType === 'nomenclature_counterparties' && (
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium flex items-center gap-2">
                    <BookOpen className="h-4 w-4" />
                    Доступные классификаторы
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  {loadingClassifiers ? (
                    <div className="text-sm text-muted-foreground">Загрузка...</div>
                  ) : projectClassifiers.length > 0 ? (
                    <div className="flex flex-wrap gap-2">
                      {projectClassifiers.map((classifier) => (
                        <Badge key={classifier.id} variant="secondary" className="text-xs">
                          {classifier.name}
                        </Badge>
                      ))}
                    </div>
                  ) : (
                    <div className="text-sm text-muted-foreground">Классификаторы не найдены</div>
                  )}
                </CardContent>
              </Card>
            )}
          </div>

          {/* Действия */}
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <FileText className="h-5 w-5" />
                  Управление эталонами
                </CardTitle>
                <CardDescription>
                  Просмотр и управление эталонными записями
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild className="w-full">
                  <Link href={`/clients/${clientId}/projects/${projectId}/benchmarks`}>
                    Открыть эталоны
                  </Link>
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <BarChart3 className="h-5 w-5" />
                  Нормализация
                </CardTitle>
                <CardDescription>
                  Запуск процесса нормализации для этого проекта
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild className="w-full">
                  <Link href={`/clients/${clientId}/projects/${projectId}/normalization`}>
                    <Play className="mr-2 h-4 w-4" />
                    Запустить нормализацию
                  </Link>
                </Button>
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Building2 className="h-5 w-5" />
                  Контрагенты
                </CardTitle>
                <CardDescription>
                  Просмотр и управление контрагентами проекта
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button asChild className="w-full">
                  <Link href={`/clients/${clientId}/projects/${projectId}/counterparties`}>
                    Открыть контрагенты
                  </Link>
                </Button>
              </CardContent>
            </Card>
          </div>
        </TabsContent>

        <TabsContent value="databases" className="space-y-6">
          {/* Базы данных */}
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Database className="h-5 w-5" />
                    Базы данных проекта
                  </CardTitle>
                  <CardDescription>
                    Управление базами данных для нормализации
                  </CardDescription>
                </div>
                <Button onClick={() => setShowAddDatabase(!showAddDatabase)} size="sm">
                  <Plus className="mr-2 h-4 w-4" />
                  Добавить базу данных
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Drag & Drop зона */}
              <div
                onDragOver={handleDragOver}
                onDragLeave={handleDragLeave}
                onDrop={handleDrop}
                className={`
                  relative border-2 border-dashed rounded-lg p-8 text-center transition-colors
                  ${isDragging 
                    ? 'border-primary bg-primary/5' 
                    : 'border-muted-foreground/25 hover:border-primary/50'
                  }
                  ${isUploading ? 'opacity-50 pointer-events-none' : ''}
                `}
              >
                <input
                  type="file"
                  id="file-upload"
                  accept=".db"
                  multiple
                  onChange={handleFileInput}
                  onClick={(e) => {
                    // Сбрасываем значение при клике, чтобы можно было выбрать тот же файл снова
                    const target = e.target as HTMLInputElement
                    if (target) {
                      target.value = ''
                    }
                  }}
                  className="hidden"
                  disabled={isUploading}
                />
                <label
                  htmlFor="file-upload"
                  className="cursor-pointer flex flex-col items-center gap-4"
                >
                  <div className={`
                    rounded-full p-4
                    ${isDragging ? 'bg-primary text-primary-foreground' : 'bg-muted'}
                  `}>
                    <Upload className={`h-8 w-8 ${isDragging ? 'text-primary-foreground' : ''}`} />
                  </div>
                  <div>
                    <p className="text-sm font-medium">
                      {isDragging 
                        ? 'Отпустите файлы для загрузки' 
                        : 'Перетащите файлы базы данных сюда или нажмите для выбора'
                      }
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Поддерживаются только файлы .db (можно выбрать несколько)
                    </p>
                  </div>
                </label>
                {(isUploading || multipleUploadProgress) && (
                  <div className="absolute inset-0 flex items-center justify-center bg-background/90 rounded-lg backdrop-blur-sm z-10">
                    <div className="flex flex-col items-center gap-4 p-6 bg-card rounded-lg border shadow-xl min-w-[280px] max-w-[400px]">
                      {multipleUploadProgress ? (
                        <>
                          <div className="flex items-center gap-3">
                            <RefreshCw className="h-6 w-6 animate-spin text-primary" />
                            <p className="text-base font-semibold">
                              Загрузка файлов ({multipleUploadProgress.completed}/{multipleUploadProgress.total})
                            </p>
                          </div>
                          <div className="w-full space-y-2">
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-muted-foreground truncate max-w-[200px]" title={multipleUploadProgress.current}>
                                {multipleUploadProgress.current}
                              </span>
                              <span className="font-medium">
                                {Math.round((multipleUploadProgress.completed / multipleUploadProgress.total) * 100)}%
                              </span>
                            </div>
                            <Progress 
                              value={(multipleUploadProgress.completed / multipleUploadProgress.total) * 100}
                              className="h-2"
                            />
                            {multipleUploadProgress.errors.length > 0 && (
                              <div className="text-xs text-destructive mt-2 space-y-1">
                                <div className="font-semibold">Ошибок: {multipleUploadProgress.errors.length}</div>
                                <details className="text-xs">
                                  <summary className="cursor-pointer hover:underline">Детали ошибок</summary>
                                  <div className="mt-2 space-y-1 max-h-32 overflow-y-auto">
                                    {multipleUploadProgress.errors.map((err, idx) => (
                                      <div key={idx} className="pl-2 border-l-2 border-destructive/50">
                                        <div className="font-medium">{err.fileName}</div>
                                        <div className="text-destructive/80">{err.error}</div>
                                      </div>
                                    ))}
                                  </div>
                                </details>
                              </div>
                            )}
                          </div>
                        </>
                      ) : (
                        <>
                          <div className="flex items-center gap-3">
                            <RefreshCw className="h-6 w-6 animate-spin text-primary" />
                            <p className="text-base font-semibold">Загрузка файла...</p>
                          </div>
                        </>
                      )}
                      {uploadMetrics && (
                        <>
                          {/* Прогресс-бар */}
                          <div className="w-full space-y-2">
                            <div className="flex items-center justify-between text-xs">
                              <span className="text-muted-foreground">Прогресс загрузки</span>
                              <span className="font-medium">
                                {uploadMetrics.duration > 0 
                                  ? Math.min(100, ((uploadMetrics.speed * uploadMetrics.duration) / (uploadMetrics.fileSize / (1024 * 1024))) * 100).toFixed(1)
                                  : 0
                                }%
                              </span>
                            </div>
                            <Progress 
                              value={uploadMetrics.duration > 0 && uploadMetrics.speed > 0
                                ? Math.min(100, Math.max(0, ((uploadMetrics.speed * uploadMetrics.duration) / (uploadMetrics.fileSize / (1024 * 1024))) * 100))
                                : 0
                              } 
                              className="h-2"
                            />
                          </div>
                          
                          {/* Метрики в сетке */}
                          <div className="grid grid-cols-2 gap-3 w-full">
                            <div className="flex items-center gap-2 p-2 bg-muted/50 rounded">
                              <Clock className="h-4 w-4 text-muted-foreground" />
                              <div className="flex-1 min-w-0">
                                <div className="text-[10px] text-muted-foreground">Время</div>
                                <div className="text-sm font-semibold truncate">{uploadMetrics.duration.toFixed(1)} сек</div>
                              </div>
                            </div>
                            <div className="flex items-center gap-2 p-2 bg-muted/50 rounded">
                              <Gauge className="h-4 w-4 text-muted-foreground" />
                              <div className="flex-1 min-w-0">
                                <div className="text-[10px] text-muted-foreground">Скорость</div>
                                <div className="text-sm font-semibold truncate">
                                  {uploadMetrics.speed > 0 ? uploadMetrics.speed.toFixed(2) : '...'} MB/s
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-2 p-2 bg-muted/50 rounded">
                              <Database className="h-4 w-4 text-muted-foreground" />
                              <div className="flex-1 min-w-0">
                                <div className="text-[10px] text-muted-foreground">Размер</div>
                                <div className="text-sm font-semibold truncate">
                                  {(uploadMetrics.fileSize / 1024 / 1024).toFixed(2)} MB
                                </div>
                              </div>
                            </div>
                            <div className="flex items-center gap-2 p-2 bg-muted/50 rounded">
                              <Activity className="h-4 w-4 text-muted-foreground" />
                              <div className="flex-1 min-w-0">
                                <div className="text-[10px] text-muted-foreground">Осталось</div>
                                <div className="text-sm font-semibold truncate">
                                  {uploadMetrics.speed > 0 
                                    ? Math.max(0, ((uploadMetrics.fileSize / (1024 * 1024) - uploadMetrics.speed * uploadMetrics.duration) / uploadMetrics.speed)).toFixed(1)
                                    : '...'
                                  } сек
                                </div>
                              </div>
                            </div>
                          </div>
                          
                          {uploadMetrics.startTime && (
                            <div className="text-[10px] text-muted-foreground w-full pt-2 border-t text-center">
                              Начало: {new Date(uploadMetrics.startTime).toLocaleTimeString('ru-RU')}
                            </div>
                          )}
                        </>
                      )}
                    </div>
                  </div>
                )}
              </div>

              {showAddDatabase && (
            <Card className="border-2 border-primary/20">
              <CardHeader>
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">Новая база данных</CardTitle>
                  {uploadMetrics && (
                    <Badge variant="outline" className="flex items-center gap-1">
                      <CheckCircle2 className="h-3 w-3 text-green-600" />
                      <span>Загружено</span>
                    </Badge>
                  )}
                </div>
                {uploadMetrics && (
                  <CardDescription className="pt-2">
                    <div className="grid grid-cols-3 gap-4 text-xs">
                      <div className="flex items-center gap-1.5">
                        <Clock className="h-3 w-3 text-muted-foreground" />
                        <div>
                          <div className="font-medium">Время загрузки</div>
                          <div className="text-muted-foreground">{uploadMetrics.duration.toFixed(2)} сек</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <Gauge className="h-3 w-3 text-muted-foreground" />
                        <div>
                          <div className="font-medium">Скорость</div>
                          <div className="text-muted-foreground">{uploadMetrics.speed.toFixed(2)} MB/s</div>
                        </div>
                      </div>
                      <div className="flex items-center gap-1.5">
                        <Database className="h-3 w-3 text-muted-foreground" />
                        <div>
                          <div className="font-medium">Размер</div>
                          <div className="text-muted-foreground">{(uploadMetrics.fileSize / 1024 / 1024).toFixed(2)} MB</div>
                        </div>
                      </div>
                    </div>
                    {uploadMetrics && (
                      <div className="mt-2 text-xs text-muted-foreground">
                        Начало загрузки: {new Date(uploadMetrics.startTime).toLocaleString('ru-RU', {
                          day: '2-digit',
                          month: '2-digit',
                          year: 'numeric',
                          hour: '2-digit',
                          minute: '2-digit',
                          second: '2-digit'
                        })}
                      </div>
                    )}
                  </CardDescription>
                )}
              </CardHeader>
              <CardContent className="space-y-4">
                {/* График скорости загрузки */}
                {uploadSpeedHistory.length > 0 && (
                  <UploadSpeedChart 
                    data={uploadSpeedHistory} 
                    totalSize={uploadMetrics?.fileSize || uploadedFile?.file.size || 0}
                  />
                )}
                {!showPendingSelector && (
                  <div className="space-y-2">
                    <Button
                      onClick={() => setShowPendingSelector(true)}
                      variant="outline"
                      className="w-full"
                    >
                      Выбрать из ожидающих баз данных
                    </Button>
                    <div className="text-center text-sm text-muted-foreground">или</div>
                  </div>
                )}

                {showPendingSelector && (
                  <div className="space-y-2">
                    <Label>Выберите из ожидающих баз данных</Label>
                    <div className="space-y-2 max-h-48 overflow-y-auto border rounded p-2">
                      {pendingDatabases.length === 0 ? (
                        <p className="text-sm text-muted-foreground text-center py-4">
                          Нет доступных ожидающих баз данных
                        </p>
                      ) : (
                        pendingDatabases.map((db, index) => (
                          <div
                            key={`pending-db-${db.id}-${db.file_path}-${index}`}
                            className="flex items-center justify-between p-2 hover:bg-muted rounded cursor-pointer"
                            onClick={() => handleSelectPendingDatabase(db)}
                          >
                            <div>
                              <div className="font-medium">{db.file_name}</div>
                              <div className="text-xs text-muted-foreground font-mono">
                                {db.file_path}
                              </div>
                            </div>
                            <Button size="sm" variant="ghost">Выбрать</Button>
                          </div>
                        ))
                      )}
                    </div>
                    <Button
                      onClick={() => {
                        setShowPendingSelector(false)
                        setUseCustomPath(true)
                      }}
                      variant="outline"
                      className="w-full"
                    >
                      Ввести путь вручную
                    </Button>
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="db-name">
                    Название
                    {uploadedFile?.nameRequired && (
                      <span className="text-destructive ml-1">*</span>
                    )}
                  </Label>
                  {uploadedFile?.nameRequired && (
                    <div className="text-sm text-amber-600 bg-amber-50 dark:bg-amber-950 dark:text-amber-400 p-2 rounded border border-amber-200 dark:border-amber-800">
                      <AlertCircle className="h-4 w-4 inline mr-1" />
                      Не удалось автоматически определить название базы данных из имени файла. Пожалуйста, введите название вручную.
                    </div>
                  )}
                  <Input
                    id="db-name"
                    placeholder={uploadedFile?.nameRequired ? "Введите название базы данных (обязательно)" : "Например: МПФ"}
                    value={newDatabase.name}
                    onChange={(e) => setNewDatabase({ ...newDatabase, name: e.target.value })}
                    required={uploadedFile?.nameRequired}
                    className={uploadedFile?.nameRequired && !newDatabase.name.trim() ? "border-destructive" : ""}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="db-path">Путь к файлу</Label>
                  <Input
                    id="db-path"
                    placeholder="E:\HttpServer\1c_data.db или оставьте пустым для перемещения в data/uploads/"
                    value={newDatabase.file_path}
                    onChange={(e) => setNewDatabase({ ...newDatabase, file_path: e.target.value })}
                    disabled={!showPendingSelector && !useCustomPath && !uploadedFile}
                  />
                  <p className="text-xs text-muted-foreground">
                    {uploadedFile 
                      ? 'Файл загружен на сервер. Путь указан автоматически.'
                      : 'Если путь не указан, файл будет автоматически перемещен в data/uploads/'
                    }
                  </p>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="db-description">Описание (необязательно)</Label>
                  <Input
                    id="db-description"
                    placeholder="Описание базы данных"
                    value={newDatabase.description}
                    onChange={(e) => setNewDatabase({ ...newDatabase, description: e.target.value })}
                  />
                </div>
                {uploadedFile && (
                  <Alert>
                    <AlertCircle className="h-4 w-4" />
                    <AlertDescription>
                      <div className="flex items-center justify-between">
                        <span>Файл загружен: {uploadedFile.file.name}</span>
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => {
                            setUploadedFile(null)
                            setNewDatabase({ name: '', file_path: '', description: '' })
                          }}
                        >
                          <X className="h-4 w-4" />
                        </Button>
                      </div>
                    </AlertDescription>
                  </Alert>
                )}
          {databaseError && (
            <Alert variant="destructive" className="mt-4">
              <AlertCircle className="h-4 w-4" />
              <AlertDescription className="flex items-center justify-between">
                <span>{databaseError}</span>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setDatabaseError(null)}
                  className="h-6 w-6 p-0"
                >
                  <X className="h-4 w-4" />
                </Button>
              </AlertDescription>
            </Alert>
          )}
                <div className="flex gap-2">
                  <Button
                    onClick={uploadedFile ? handleConfirmUpload : handleAddDatabase}
                    disabled={isAddingDatabase}
                    className="flex-1"
                  >
                    {isAddingDatabase ? 'Добавление...' : uploadedFile ? 'Подтвердить и добавить' : 'Добавить'}
                  </Button>
                  <Button
                    onClick={() => {
                      setShowAddDatabase(false)
                      setShowPendingSelector(false)
                      setUseCustomPath(false)
                      setDatabaseError(null)
                      setUploadedFile(null)
                      setNewDatabase({ name: '', file_path: '', description: '' })
                    }}
                    variant="outline"
                    className="flex-1"
                  >
                    Отмена
                  </Button>
                </div>
              </CardContent>
            </Card>
          )}

          {databases.length === 0 ? (
            <EmptyState
              icon={Database}
              title="Нет добавленных баз данных"
              description="Добавьте базу данных для начала работы"
            />
          ) : (
            <div className="space-y-2">
              {databases.map((db, index) => (
                <Card key={`db-${db.id}-${db.file_path}-${index}`} className="hover:shadow-md transition-shadow">
                  <CardContent className="pt-6">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-2">
                          <Database className="h-4 w-4 text-primary" />
                          <h4 className="font-semibold">{db.name}</h4>
                          {db.is_active && <Badge variant="default">Активна</Badge>}
                        </div>
                        <p className="text-sm text-muted-foreground mt-1 font-mono">
                          {db.file_path}
                        </p>
                        {db.description && (
                          <p className="text-sm text-muted-foreground mt-1">
                            {db.description}
                          </p>
                        )}
                        <div className="flex items-center gap-4 mt-2 text-xs text-muted-foreground">
                          <span>Добавлено: {new Date(db.created_at).toLocaleDateString('ru-RU')}</span>
                          {db.statistics && (
                            <span className="flex items-center gap-1">
                              <Table className="h-3.5 w-3.5" />
                              {db.statistics.total_tables} {db.statistics.total_tables === 1 ? 'таблица' : db.statistics.total_tables < 5 ? 'таблицы' : 'таблиц'}
                              {db.statistics.total_rows > 0 && (
                                <span className="ml-1">
                                  • {db.statistics.total_rows.toLocaleString('ru-RU')} {db.statistics.total_rows === 1 ? 'запись' : db.statistics.total_rows < 5 ? 'записи' : 'записей'}
                                </span>
                              )}
                            </span>
                          )}
                          {db.tables && db.tables.length > 0 && !db.statistics && (
                            <span className="flex items-center gap-1">
                              <Table className="h-3.5 w-3.5" />
                              {db.tables.length} {db.tables.length === 1 ? 'таблица' : db.tables.length < 5 ? 'таблицы' : 'таблиц'}
                              {db.tables.some(t => t.row_count !== undefined) && (
                                <span className="ml-1">
                                  • {db.tables.reduce((sum, t) => sum + (t.row_count || 0), 0).toLocaleString('ru-RU')} {db.tables.reduce((sum, t) => sum + (t.row_count || 0), 0) === 1 ? 'запись' : 'записей'}
                                </span>
                              )}
                            </span>
                          )}
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <Button 
                          size="sm" 
                          variant="outline"
                          onClick={() => setSelectedDatabaseForDetail(db)}
                        >
                          <Eye className="h-4 w-4 mr-2" />
                          Детали
                        </Button>
                        <Button asChild size="sm" variant="outline">
                          <Link href={`/clients/${clientId}/projects/${projectId}/databases/${db.id}`}>
                            Открыть
                          </Link>
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => handleDeleteDatabase(db.id)}
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
            </CardContent>
          </Card>
        </TabsContent>

        {(projectType === 'nomenclature' || projectType === 'normalization' || projectType === 'nomenclature_counterparties') && (
          <TabsContent value="pipeline-stages" className="space-y-6">
            <PipelineStagesTab clientId={clientId as string} projectId={projectId as string} />
          </TabsContent>
        )}
      </Tabs>

      {/* Диалог детальной информации о базе данных */}
      {selectedDatabaseForDetail && (
        <DatabaseDetailDialog
          database={{
            id: selectedDatabaseForDetail.id,
            name: selectedDatabaseForDetail.name,
            path: selectedDatabaseForDetail.file_path,
            size: selectedDatabaseForDetail.file_size,
            created_at: selectedDatabaseForDetail.created_at,
            status: selectedDatabaseForDetail.is_active ? 'active' : 'inactive',
            project_id: parseInt(projectId as string),
            project_name: project?.project.name || ''
          }}
          clientId={clientId as string}
          open={!!selectedDatabaseForDetail}
          onOpenChange={(open) => {
            if (!open) {
              setSelectedDatabaseForDetail(null)
            }
          }}
        />
      )}
    </div>
  )
}

