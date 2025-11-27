'use client'

import { useState, useEffect, useRef } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { 
  Settings, 
  Cpu, 
  Zap, 
  CheckCircle2, 
  XCircle, 
  Save,
  RefreshCw,
  AlertCircle,
  Loader2,
  ChevronDown,
  ChevronUp,
  Info,
  Key,
  TestTube,
  Download,
  Award,
  Activity,
  TrendingUp,
  Clock,
  BarChart3,
  Play
} from 'lucide-react'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { WorkersPageSkeleton } from '@/components/common/workers-skeleton'
import { StatCard } from '@/components/common/stat-card'
import { Breadcrumb } from '@/components/ui/breadcrumb'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { motion } from 'framer-motion'
import { FadeIn } from '@/components/animations/fade-in'
import { Gauge } from 'lucide-react'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'
import { useApiClient } from '@/hooks/useApiClient'
import { BenchmarkResultsTable } from '@/components/workers/benchmark-results-table'
import { BenchmarkRadarChart } from '@/components/workers/benchmark-radar-chart'
import { PerformanceHeatmap } from '@/components/workers/performance-heatmap'
import { ModelRecommendationEngine } from '@/components/workers/model-recommendation-engine'
import { BenchmarkTimeSeries } from '@/components/workers/benchmark-time-series'

interface ModelConfig {
  name: string
  provider: string
  enabled: boolean
  priority: number
  max_tokens?: number
  temperature?: number
  speed?: string
  quality?: string
  cost_per_token?: number
}

interface ProviderConfig {
  name: string
  base_url: string
  enabled: boolean
  priority: number
  max_workers: number
  rate_limit: number
  timeout: string
  models: ModelConfig[]
  api_key?: string // API ключ (опционально, может быть скрыт)
  has_api_key?: boolean // Флаг наличия API ключа на сервере
}

interface WorkerConfig {
  providers: Record<string, ProviderConfig>
  default_provider: string
  default_model: string
  global_max_workers: number
  isFallback?: boolean
  fallbackReason?: string
  lastSync?: string
}

interface BenchmarkResult {
  // Существующие поля
  model: string
  priority: number
  speed: number
  avg_response_time_ms: number
  median_response_time_ms?: number
  p75_response_time_ms?: number
  p90_response_time_ms?: number
  p95_response_time_ms?: number
  p99_response_time_ms?: number
  min_response_time_ms?: number
  max_response_time_ms?: number
  success_count: number
  error_count: number
  total_requests: number
  success_rate: number
  status: string
  
  // Новые поля для качества и стабильности
  avg_confidence?: number
  min_confidence?: number
  max_confidence?: number
  avg_ai_calls_count?: number
  avg_retries?: number
  coefficient_of_variation?: number
  throughput_items_per_sec?: number
  error_breakdown?: {
    quota_exceeded: number
    rate_limit: number
    timeout: number
    network: number
    auth: number
    other: number
  }
}

interface BenchmarkResponse {
  models: BenchmarkResult[]
  total: number
  test_count?: number // Опциональное, так как API может не возвращать это поле
  timestamp: string
  message?: string
}

export default function WorkersPage() {
  const { get, post } = useApiClient()
  const [config, setConfig] = useState<WorkerConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)
  const [expandedProviders, setExpandedProviders] = useState<Set<string>>(new Set())
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({}) // Локальное хранение API ключей
  const [apiKeyStatus, setApiKeyStatus] = useState<Record<string, { connected: boolean; testing: boolean; error?: string }>>({})
  const [refreshingModels, setRefreshingModels] = useState<Record<string, boolean>>({})
  const [savingApiKey, setSavingApiKey] = useState<Record<string, boolean>>({})
  const successTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const refreshingRef = useRef<Record<string, boolean>>({}) // Ref для отслеживания активных обновлений
  const [workerMetrics, setWorkerMetrics] = useState<any>(null)
  const [loadingMetrics, setLoadingMetrics] = useState(false)
  const [runningBenchmark, setRunningBenchmark] = useState<Record<string, boolean>>({})
  const [benchmarkResults, setBenchmarkResults] = useState<Record<string, BenchmarkResponse>>({})
  const [benchmarkProgress, setBenchmarkProgress] = useState<Record<string, { current: number; total: number }>>({})

  useEffect(() => {
    fetchConfig()
    fetchWorkerMetrics()
    
    // Обновляем метрики каждые 30 секунд
    const metricsInterval = setInterval(fetchWorkerMetrics, 30000)
    
    // Cleanup timeout при размонтировании
    return () => {
      if (successTimeoutRef.current) {
        clearTimeout(successTimeoutRef.current)
        successTimeoutRef.current = null
      }
      clearInterval(metricsInterval)
    }
  }, [])

  const fetchWorkerMetrics = async () => {
    setLoadingMetrics(true)
    try {
      const data = await get<any>('/api/monitoring/metrics', {
        skipErrorHandler: true,
        timeoutMs: 15000,
      })
      setWorkerMetrics(data)
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      console.error('Error fetching worker metrics:', err)
    } finally {
      setLoadingMetrics(false)
    }
  }

  const ensureBackendAvailable = (actionDescription?: string) => {
    if (config?.isFallback) {
      const reason = config.fallbackReason
        ? `Причина: ${config.fallbackReason}`
        : 'Backend конфигурации (порт 9999) недоступен.'
      const message = actionDescription ? `${actionDescription}. ${reason}` : reason
      toast.error('Backend недоступен', {
        description: message,
      })
      return false
    }
    return true
  }

  const testAPIKey = async (providerName: string) => {
    if (!config) {
      setError('Конфигурация не загружена')
      return
    }
    
    const provider = config.providers[providerName]
    if (!provider) {
      setError(`Провайдер ${providerName} не найден`)
      return
    }

    if (!ensureBackendAvailable('Невозможно проверить API ключ пока backend недоступен')) {
      return
    }

    // Устанавливаем статус тестирования
    setApiKeyStatus(prev => ({ ...prev, [providerName]: { connected: false, testing: true } }))

    try {
      let endpoint: string | null = null
      if (providerName === 'arliai') {
        // Используем специальный эндпоинт для Arliai
        endpoint = '/api/workers/arliai/status'
      } else if (providerName === 'openrouter') {
        // Используем специальный эндпоинт для OpenRouter
        endpoint = '/api/workers/openrouter/status'
      } else if (providerName === 'huggingface') {
        // Используем специальный эндпоинт для Hugging Face
        endpoint = '/api/workers/huggingface/status'
      } else {
        // Для других провайдеров можно добавить аналогичные эндпоинты
        setApiKeyStatus(prev => ({ 
          ...prev, 
          [providerName]: { 
            connected: false, 
            testing: false, 
            error: 'Проверка подключения не поддерживается для этого провайдера' 
          } 
        }))
        return
      }

      const responseData = await get<{ data?: { connected: boolean; error?: string; has_api_key?: boolean }; connected?: boolean; error?: string; has_api_key?: boolean }>(endpoint, {
        skipErrorHandler: true,
        timeoutMs: 20000,
      })
      const data = responseData.data || responseData

      // Определяем сообщение об ошибке
      let errorMessage: string | undefined
      if (!data.connected) {
        if (data.error) {
          errorMessage = data.error
        } else if (!data.has_api_key) {
          errorMessage = 'API ключ не установлен'
        } else {
          errorMessage = 'Подключение не установлено. Проверьте API ключ и доступность сервиса'
        }
      }
      
      setApiKeyStatus(prev => ({ 
        ...prev, 
        [providerName]: { 
          connected: data.connected || false, 
          testing: false,
          error: errorMessage
        } 
      }))
      
      if (data.connected) {
        toast.success('Подключение установлено', {
          description: `Провайдер ${providerName} успешно подключен`,
        })
      } else {
        toast.error('Ошибка подключения', {
          description: errorMessage || `Не удалось подключиться к провайдеру ${providerName}`,
        })
      }
      
      // Если подключение успешно, автоматически обновляем список моделей
      // Только если еще не обновляется
      if (data.connected && (providerName === 'arliai' || providerName === 'openrouter' || providerName === 'huggingface') && !refreshingRef.current[providerName]) {
        setTimeout(() => {
          refreshModels(providerName)
        }, 500)
      }
    } catch (err) {
      let errorMessage = 'Ошибка проверки подключения'
      
      if (err instanceof Error) {
        if (err.message.includes('timeout') || err.message.includes('Превышено время ожидания')) {
          errorMessage = `Таймаут при проверке подключения для ${providerName}. Сервер может быть перегружен.`
        } else if (err.message.includes('Failed to fetch') || err.message.includes('network') || err.message.includes('ECONNREFUSED')) {
          errorMessage = `Не удалось подключиться к серверу для ${providerName}. Проверьте, что бэкенд запущен.`
        } else if (err.message.includes('API key') || err.message.includes('401') || err.message.includes('403')) {
          errorMessage = `Неверный API ключ для ${providerName}. Проверьте правильность ключа.`
        } else {
          errorMessage = err.message
        }
      } else if (typeof err === 'string') {
        errorMessage = err
      } else if (err && typeof err === 'object' && 'message' in err) {
        errorMessage = String((err as { message: string }).message)
      }
      
      toast.error('Ошибка проверки подключения', {
        description: errorMessage,
        duration: 5000,
      })
      
      setApiKeyStatus(prev => ({ 
        ...prev, 
        [providerName]: { 
          connected: false, 
          testing: false,
          error: errorMessage
        } 
      }))
    }
  }

  const refreshModels = async (providerName: string) => {
    // Защита от повторных вызовов
    if (refreshingRef.current[providerName]) {
      return
    }
    
    if (!config) {
      setError('Конфигурация не загружена')
      return
    }
    
    const provider = config.providers[providerName]
    if (!provider) {
      setError(`Провайдер ${providerName} не найден`)
      return
    }

    if (!ensureBackendAvailable('Невозможно обновить список моделей пока backend недоступен')) {
      return
    }

    refreshingRef.current[providerName] = true
    setRefreshingModels(prev => ({ ...prev, [providerName]: true }))
    setError(null)

    try {
      // Запрашиваем список моделей из API с указанием провайдера и принудительным обновлением
      const data = await get<{ 
        success: boolean; 
        data?: { 
          models?: any[]; 
          api_error?: string; 
          api_available?: boolean;
          error_type?: string;
          error_message?: string;
          api_models_count?: number;
          local_models_count?: number;
        } 
      }>(`/api/workers/models?provider=${providerName}&refresh=1`, {
        skipErrorHandler: true,
        timeoutMs: 30000,
      })
      
      if (data.success) {
        // Обновляем конфигурацию напрямую, не вызывая fetchConfig (чтобы избежать бесконечного цикла)
        setConfig((prev) => {
          if (!prev) return prev
          return {
            ...prev,
            providers: {
              ...prev.providers,
              [providerName]: {
                ...prev.providers[providerName],
                available_models: data.data?.models || [],
              },
            },
          }
        })
        
        // Проверяем наличие ошибки API
        if (data.data?.api_error) {
          const errorType = data.data.error_type as string
          const errorMessage = data.data.error_message as string || data.data.api_error
          
          let toastTitle = 'Ошибка загрузки моделей'
          let toastVariant: 'error' | 'warning' = 'error'
          
          // Определяем тип ошибки и соответствующее сообщение
          if (errorType === 'missing_api_key') {
            toastTitle = 'API ключ не установлен'
            toastVariant = 'warning'
          } else if (errorType === 'unauthorized' || errorType === 'forbidden') {
            toastTitle = 'Ошибка авторизации'
            toastVariant = 'error'
          } else if (errorType === 'timeout') {
            toastTitle = 'Таймаут запроса'
            toastVariant = 'warning'
          }
          
          const errorMsg = errorMessage || `Ошибка при загрузке моделей: ${data.data.api_error}`
          
          if (toastVariant === 'error') {
            toast.error(toastTitle, {
              description: errorMsg,
              duration: 6000,
            })
          } else {
            toast.warning(toastTitle, {
              description: errorMsg,
              duration: 6000,
            })
          }
          setSuccess(errorMsg)
        } else if (data.data?.models && data.data.models.length > 0) {
          const message = `Список моделей обновлен (найдено моделей: ${data.data.models.length})`
          toast.success('Модели загружены', {
            description: message,
          })
          setSuccess(message)
        } else {
          // Проверяем, доступен ли API
          if (data.data?.api_available === false) {
            const message = 'Список моделей обновлен (API недоступен. Проверьте API ключ и подключение)'
            toast.warning('API недоступен', {
              description: message,
              duration: 5000,
            })
            setSuccess(message)
          } else {
            const message = 'Список моделей обновлен (модели не найдены. Возможно, API ключ не установлен или неверен)'
            toast.info('Модели не найдены', {
              description: message,
              duration: 5000,
            })
            setSuccess(message)
          }
        }
        
        if (successTimeoutRef.current) {
          clearTimeout(successTimeoutRef.current)
        }
        successTimeoutRef.current = setTimeout(() => {
          setSuccess(null)
          successTimeoutRef.current = null
        }, 3000)
      } else {
        throw new Error('Не удалось обновить список моделей')
      }
    } catch (err) {
      let errorMessage = 'Ошибка обновления списка моделей'
      
      if (err instanceof Error) {
        if (err.message.includes('timeout') || err.message.includes('Превышено время ожидания')) {
          errorMessage = `Таймаут при загрузке моделей для ${providerName}. Попробуйте позже.`
        } else if (err.message.includes('Failed to fetch') || err.message.includes('network') || err.message.includes('ECONNREFUSED')) {
          errorMessage = `Не удалось подключиться к серверу для ${providerName}. Проверьте подключение.`
        } else if (err.message.includes('API key') || err.message.includes('401') || err.message.includes('403')) {
          errorMessage = `API ключ для ${providerName} не настроен или неверен. Проверьте настройки.`
        } else {
          errorMessage = err.message
        }
      } else if (typeof err === 'string') {
        errorMessage = err
      } else if (err && typeof err === 'object' && 'message' in err) {
        errorMessage = String((err as { message: string }).message)
      }
      
      console.error('[Workers] Error refreshing models:', {
        provider: providerName,
        error: err,
        message: errorMessage,
      })
      
      // Если это ошибка из-за отсутствия подключения, не показываем критическую ошибку
      if (errorMessage.includes('API key') || errorMessage.includes('подключ') || errorMessage.includes('401') || errorMessage.includes('403')) {
        // Это не критическая ошибка, просто не удалось обновить модели
        // Статус подключения уже установлен в testAPIKey
        toast.warning('Не удалось загрузить модели', {
          description: errorMessage,
          duration: 4000,
        })
      } else {
        toast.error('Ошибка загрузки моделей', {
          description: errorMessage,
          duration: 5000,
        })
        setError(errorMessage)
      }
    } finally {
      refreshingRef.current[providerName] = false
      setRefreshingModels(prev => ({ ...prev, [providerName]: false }))
    }
  }

  const fetchConfig = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await get<WorkerConfig>('/api/workers/config', {
        skipErrorHandler: true,
        timeoutMs: 15000,
      })
      
      // Проверяем, что данные валидны
      if (!data || typeof data !== 'object') {
        throw new Error('Invalid config data received')
      }
      
      setConfig(data)
      // Очищаем ошибку после успешной загрузки
      setError(null)
      // Разворачиваем все провайдеры по умолчанию
      setExpandedProviders(new Set(Object.keys(data.providers || {})))
      // Сохраняем уже введенные API ключи, не сбрасываем их при обновлении конфигурации
      // (бэкенд скрывает API ключи в ответе, поэтому мы сохраняем локальные значения)
      setApiKeys(prev => {
        const keys: Record<string, string> = { ...prev }
        Object.keys(data.providers || {}).forEach(name => {
          // Инициализируем только если ключа еще нет
          if (!keys[name]) {
            keys[name] = ''
          }
        })
        return keys
      })
      // Обновляем статус только для новых провайдеров
      setApiKeyStatus(prev => {
        const status: Record<string, { connected: boolean; testing: boolean; error?: string }> = { ...prev }
        Object.keys(data.providers || {}).forEach(name => {
          if (!status[name]) {
            status[name] = { connected: false, testing: false }
          }
        })
        return status
      })
      
      // Проверяем статус подключения для Arliai после небольшой задержки
      // Только если еще не проверяли (избегаем бесконечных циклов)
      if (data.providers?.arliai && !apiKeyStatus['arliai']?.testing && !apiKeyStatus['arliai']?.connected) {
        setTimeout(() => {
          // Вызываем testAPIKey после того, как config установлен
          // Используем setTimeout для гарантии, что состояние обновлено
          testAPIKey('arliai')
        }, 200)
      }
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }

  const updateConfig = async (action: string, data: Record<string, unknown>) => {
    if (!ensureBackendAvailable('Невозможно обновить конфигурацию воркеров')) {
      return
    }
    setSaving(true)
    setError(null)
    setSuccess(null)
    
    // Очищаем предыдущий timeout, если он есть
    if (successTimeoutRef.current) {
      clearTimeout(successTimeoutRef.current)
      successTimeoutRef.current = null
    }
    
    try {
      const responseData = await post<{ message?: string }>('/api/workers/config', { action, data }, {
        skipErrorHandler: true,
        timeoutMs: 20000,
      })
      const message = responseData.message || 'Конфигурация обновлена успешно'
      
      toast.success('Конфигурация сохранена', {
        description: message,
      })
      setSuccess(message)
      await fetchConfig()
      successTimeoutRef.current = setTimeout(() => {
        setSuccess(null)
        successTimeoutRef.current = null
      }, 3000)
    } catch (err) {
      // Ошибка уже обработана через ErrorContext, если не skipErrorHandler
      const errorMessage = err instanceof Error 
        ? err.message 
        : 'Неизвестная ошибка при обновлении конфигурации'
      setError(errorMessage)
    } finally {
      setSaving(false)
    }
  }

  const toggleProvider = (providerName: string) => {
    if (!config) return
    const provider = config.providers[providerName]
    if (!provider) return

    // Исключаем timeout из данных, так как он может быть строкой, а бэкенд ожидает time.Duration
    const { timeout, ...providerWithoutTimeout } = provider
    updateConfig('update_provider', {
      ...providerWithoutTimeout,
      name: providerName,
      enabled: !provider.enabled,
    })
  }

  const updateProviderPriority = (providerName: string, priority: number) => {
    if (!config) return
    if (priority < 1) {
      setError('Приоритет должен быть больше 0')
      return
    }
    const provider = config.providers[providerName]
    if (!provider) return

    // Исключаем timeout из данных, так как он может быть строкой, а бэкенд ожидает time.Duration
    const { timeout, ...providerWithoutTimeout } = provider
    updateConfig('update_provider', {
      ...providerWithoutTimeout,
      name: providerName,
      priority,
    })
  }

  const updateProviderWorkers = (providerName: string, maxWorkers: number) => {
    if (!config) return
    if (maxWorkers < 1 || maxWorkers > 100) {
      setError('Количество воркеров должно быть от 1 до 100')
      return
    }
    const provider = config.providers[providerName]
    if (!provider) return

    // Исключаем timeout из данных, так как он может быть строкой, а бэкенд ожидает time.Duration
    const { timeout, ...providerWithoutTimeout } = provider
    updateConfig('update_provider', {
      ...providerWithoutTimeout,
      name: providerName,
      max_workers: maxWorkers,
    })
  }

  const saveProviderAPIKey = async (providerName: string) => {
    if (!config) {
      setError('Конфигурация не загружена')
      return
    }
    const provider = config.providers[providerName]
    if (!provider) {
      setError(`Провайдер ${providerName} не найден`)
      return
    }

    if (!ensureBackendAvailable('Невозможно сохранить API ключ, сервер недоступен')) {
      return
    }

    const apiKey = apiKeys[providerName] || ''
    setSavingApiKey(prev => ({ ...prev, [providerName]: true }))
    setError(null)
    setSuccess(null)

    try {
      // Отправляем обновление на сервер
      // Исключаем timeout из данных, так как он может быть строкой, а бэкенд ожидает time.Duration
      const { timeout, ...providerWithoutTimeout } = provider
      await updateConfig('update_provider', {
        ...providerWithoutTimeout,
        name: providerName,
        api_key: apiKey.trim(),
      })
      
      setSuccess('API ключ сохранен')
      if (successTimeoutRef.current) {
        clearTimeout(successTimeoutRef.current)
      }
      successTimeoutRef.current = setTimeout(() => {
        setSuccess(null)
        successTimeoutRef.current = null
      }, 3000)

      // После сохранения проверяем подключение и обновляем модели
      if ((providerName === 'arliai' || providerName === 'openrouter' || providerName === 'huggingface') && apiKey.trim() !== '') {
        // Увеличиваем задержку до 1000ms для надежности
        setTimeout(async () => {
          try {
            // Проверяем подключение
            await testAPIKey(providerName)
            
            // Дополнительная задержка для установки статуса connected
            await new Promise(resolve => setTimeout(resolve, 300))
            
            // Проверяем статус подключения перед обновлением моделей
            // Используем функциональное обновление для получения актуального значения
            setApiKeyStatus(prev => {
              const status = prev[providerName]
              if (status?.connected && !refreshingRef.current[providerName]) {
                // Явно вызываем обновление моделей после успешной проверки
                // Только если еще не обновляется
                refreshModels(providerName).catch(err => {
                  console.error('Error refreshing models after API key save:', err)
                })
              }
              return prev // Не изменяем состояние, только проверяем
            })
          } catch (err) {
            // Ошибка уже обработана в testAPIKey, просто логируем
            console.error('Error during API key test or model refresh:', err)
          }
        }, 1000)
      } else if (apiKey.trim() === '') {
        // Очищаем статус, если ключ удален
        setApiKeyStatus(prev => ({ ...prev, [providerName]: { connected: false, testing: false } }))
      }
    } catch (err) {
      const errorMessage = err instanceof Error 
        ? err.message 
        : 'Ошибка сохранения API ключа'
      setError(errorMessage)
    } finally {
      setSavingApiKey(prev => ({ ...prev, [providerName]: false }))
    }
  }

  const updateProviderAPIKey = (providerName: string, apiKey: string) => {
    // Обновляем только локальное состояние, не сохраняем автоматически
    setApiKeys(prev => ({ ...prev, [providerName]: apiKey }))
  }

  const updateGlobalWorkers = (maxWorkers: number) => {
    if (maxWorkers < 1 || maxWorkers > 100) {
      setError('Количество воркеров должно быть от 1 до 100')
      return
    }
    updateConfig('set_max_workers', { max_workers: maxWorkers })
  }

  const toggleModel = (providerName: string, modelName: string) => {
    if (!config) return
    const provider = config.providers[providerName]
    if (!provider) return
    const model = provider.models.find(m => m.name === modelName)
    if (!model) return

    updateConfig('update_model', {
      ...model,
      provider: providerName,
      name: modelName,
      enabled: !model.enabled,
    })
  }

  const updateModelPriority = (providerName: string, modelName: string, priority: number) => {
    if (!config) return
    if (priority < 1) {
      setError('Приоритет должен быть больше 0')
      return
    }
    const provider = config.providers[providerName]
    if (!provider) return
    const model = provider.models.find(m => m.name === modelName)
    if (!model) return

    updateConfig('update_model', {
      ...model,
      provider: providerName,
      name: modelName,
      priority,
    })
  }

  const setDefaultProvider = (providerName: string) => {
    updateConfig('set_default_provider', { provider: providerName })
  }

  const setDefaultModel = (providerName: string, modelName: string) => {
    updateConfig('set_default_model', { provider: providerName, model: modelName })
  }

  const toggleProviderExpanded = (providerName: string) => {
    setExpandedProviders(prev => {
      const newSet = new Set(prev)
      if (newSet.has(providerName)) {
        newSet.delete(providerName)
      } else {
        newSet.add(providerName)
      }
      return newSet
    })
  }

  const runBenchmark = async (providerName: string) => {
    if (!config) {
      const errorMsg = 'Конфигурация не загружена'
      console.error('[Benchmark] Cannot run benchmark:', {
        provider: providerName,
        error: errorMsg,
      })
      setError(errorMsg)
      return
    }

    const provider = config.providers[providerName]
    if (!provider) {
      const errorMsg = `Провайдер ${providerName} не найден`
      console.error('[Benchmark] Provider not found:', {
        provider: providerName,
        availableProviders: Object.keys(config.providers),
      })
      setError(errorMsg)
      return
    }

    if (!ensureBackendAvailable('Невозможно запустить бэнчмарк, сервер недоступен')) {
      console.error('[Benchmark] Backend unavailable:', {
        provider: providerName,
      })
      return
    }

    // Проверяем наличие API ключа
    if (!provider.has_api_key && !apiKeys[providerName]) {
      console.warn('[Benchmark] API key not set:', {
        provider: providerName,
        hasApiKey: provider.has_api_key,
      })
      toast.error('API ключ не установлен', {
        description: 'Пожалуйста, сохраните API ключ перед запуском бэнчмарка',
      })
      return
    }

    setRunningBenchmark(prev => ({ ...prev, [providerName]: true }))
    setBenchmarkProgress(prev => ({ ...prev, [providerName]: { current: 0, total: 0 } }))
    setError(null)

    try {
      // Получаем датасет из нормализации с таймаутом
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 15000) // 15 секунд для получения датасета

      let testProducts: string[] = []
      try {
        const datasetResponse = await fetch('/api/normalization/benchmark-dataset?limit=50', {
          signal: controller.signal,
        })
        clearTimeout(timeoutId)

        // Обрабатываем как успешные, так и неуспешные ответы
        // API route возвращает пустой массив при ошибках, поэтому всегда парсим JSON
        try {
          const datasetData = await datasetResponse.json()
          if (datasetData?.data && Array.isArray(datasetData.data)) {
            testProducts = datasetData.data
              .filter((item: unknown): item is string => typeof item === 'string' && item.trim() !== '')
              .slice(0, 50) // Ограничиваем максимум 50 элементами
          }
          if (!datasetResponse.ok && datasetData?.message) {
            console.warn(`[Benchmark] Dataset fetch warning: ${datasetData.message}`, {
              status: datasetResponse.status,
              provider: providerName,
            })
          }
        } catch (parseError) {
          console.warn('[Benchmark] Failed to parse dataset response, using empty array', {
            status: datasetResponse.status,
            provider: providerName,
            error: parseError,
          })
        }
      } catch (datasetError: any) {
        clearTimeout(timeoutId)
        if (datasetError.name === 'AbortError') {
          console.warn('[Benchmark] Dataset fetch timeout, using empty array (backend will use default data)', {
            provider: providerName,
          })
        } else {
          console.warn('[Benchmark] Error fetching dataset, using empty array (backend will use default data):', {
            error: datasetError,
            message: datasetError?.message,
            provider: providerName,
          })
        }
      }

      // Если не получили датасет, используем пустой массив (бэкенд использует дефолтные)
      // Это нормально, бэкенд имеет fallback на дефолтные данные

      // Запускаем бэнчмарк
      const benchmarkResponse = await post<{
        models?: Array<{
          model: string
          priority: number
          speed: number
          avg_response_time_ms: number
          success_count: number
          error_count: number
          success_rate: number
          status: string
        }>
        total?: number
        timestamp?: string
        message?: string
      }>('/api/models/benchmark', {
        provider: providerName,
        use_normalization_dataset: true,
        test_products: testProducts,
        max_retries: 5,
        retry_delay_ms: 200,
        auto_update_priorities: false,
      }, {
        timeoutMs: 60000,
      })

      if (benchmarkResponse?.models && Array.isArray(benchmarkResponse.models) && benchmarkResponse.models.length > 0) {
        // Преобразуем ответ API в BenchmarkResponse с обязательными полями
        const formattedModels: BenchmarkResult[] = benchmarkResponse.models.map((model: any): BenchmarkResult => ({
          model: model.model,
          priority: model.priority,
          speed: model.speed,
          avg_response_time_ms: model.avg_response_time_ms,
          success_count: model.success_count,
          error_count: model.error_count,
          success_rate: model.success_rate,
          status: model.status,
          // Вычисляем total_requests из success_count + error_count
          total_requests: model.total_requests || (model.success_count + model.error_count),
          // Перцентили времени ответа
          median_response_time_ms: model.median_response_time_ms,
          p75_response_time_ms: model.p75_response_time_ms,
          p90_response_time_ms: model.p90_response_time_ms,
          p95_response_time_ms: model.p95_response_time_ms,
          p99_response_time_ms: model.p99_response_time_ms,
          min_response_time_ms: model.min_response_time_ms,
          max_response_time_ms: model.max_response_time_ms,
          // Метрики качества классификации
          avg_confidence: model.avg_confidence,
          min_confidence: model.min_confidence,
          max_confidence: model.max_confidence,
          avg_ai_calls_count: model.avg_ai_calls_count,
          // Метрики надежности
          avg_retries: model.avg_retries,
          coefficient_of_variation: model.coefficient_of_variation,
          // Дополнительные метрики
          throughput_items_per_sec: model.throughput_items_per_sec || model.speed,
          // Детальная статистика ошибок
          error_breakdown: model.error_breakdown,
        }))
        
        const formattedResponse: BenchmarkResponse = {
          models: formattedModels,
          total: benchmarkResponse.total ?? benchmarkResponse.models.length,
          test_count: benchmarkResponse.total ?? benchmarkResponse.models.length, // Используем total как test_count если доступно
          timestamp: benchmarkResponse.timestamp ?? new Date().toISOString(),
          message: benchmarkResponse.message,
        }
        // Явно типизируем обновление state для избежания проблем с типами
        setBenchmarkResults((prev: Record<string, BenchmarkResponse>) => ({
          ...prev,
          [providerName]: formattedResponse,
        }))
        console.log(`[Benchmark] Completed for ${providerName}`, {
          models: formattedResponse.models.length,
          timestamp: formattedResponse.timestamp,
        })
        toast.success(`Бэнчмарк для ${providerName} завершен`, {
          description: `Протестировано ${formattedResponse.models.length} моделей`,
        })
      } else {
        const errorMsg = benchmarkResponse?.message || 'Неверный формат ответа от сервера'
        console.error('[Benchmark] Invalid response format', {
          provider: providerName,
          response: benchmarkResponse,
          message: errorMsg,
        })
        throw new Error(errorMsg)
      }
    } catch (err) {
      let errorMessage = 'Ошибка запуска бэнчмарка'
      
      if (err instanceof Error) {
        // Улучшаем сообщения об ошибках для пользователя
        if (err.message.includes('timeout') || err.message.includes('Превышено время ожидания')) {
          errorMessage = `Таймаут при запуске бэнчмарка для ${providerName}. Сервер может быть перегружен. Попробуйте позже.`
        } else if (err.message.includes('Failed to fetch') || err.message.includes('network') || err.message.includes('ECONNREFUSED')) {
          errorMessage = `Не удалось подключиться к серверу для ${providerName}. Проверьте, что бэкенд запущен и доступен.`
        } else if (err.message.includes('API key') || err.message.includes('ARLIAI_API_KEY')) {
          errorMessage = `API ключ для ${providerName} не настроен. Настройте его в разделе 'Воркеры'.`
        } else if (err.message.includes('No models available') || err.message.includes('нет доступных моделей')) {
          errorMessage = `Нет доступных моделей для ${providerName}. Проверьте конфигурацию воркеров.`
        } else {
          errorMessage = err.message
        }
      } else if (typeof err === 'string') {
        errorMessage = err
      } else if (err && typeof err === 'object' && 'message' in err) {
        errorMessage = String((err as { message: string }).message)
      }
      
      console.error('[Benchmark] Failed to run benchmark', {
        provider: providerName,
        error: err,
        message: errorMessage,
        errorType: err instanceof Error ? err.constructor.name : typeof err,
      })
      
      setError(errorMessage)
      toast.error('Ошибка бэнчмарка', {
        description: errorMessage,
        duration: 7000, // Увеличиваем время показа для важных ошибок
      })
    } finally {
      setRunningBenchmark(prev => ({ ...prev, [providerName]: false }))
      setBenchmarkProgress(prev => ({ ...prev, [providerName]: { current: 0, total: 0 } }))
    }
  }

  if (loading) {
    return (
      <div className="container-wide mx-auto px-4 py-8">
        <LoadingState message="Загрузка конфигурации воркеров..." size="lg" fullScreen />
      </div>
    )
  }

  if (!config && !loading) {
    return (
      <div className="container-wide mx-auto px-4 py-8 space-y-4">
        <ErrorState
          title="Ошибка загрузки конфигурации"
          message={error || 'Не удалось загрузить конфигурацию'}
          action={{
            label: 'Попробовать снова',
            onClick: fetchConfig,
          }}
          variant="destructive"
        />
        <Card>
          <CardHeader>
            <CardTitle>Возможные причины:</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2 text-sm">
            <p>• Сервер не запущен или недоступен</p>
            <p>• Эндпоинт /api/workers/config не зарегистрирован (требуется перезапуск сервера)</p>
            <p>• Проблемы с сетью или CORS</p>
            <p className="mt-4 font-semibold">Решение: Перезапустите Go сервер, чтобы загрузить новые эндпоинты</p>
          </CardContent>
        </Card>
      </div>
    )
  }

  const providers = Object.entries(config?.providers || {})
  const isFallbackConfig = Boolean(config?.isFallback)

  const breadcrumbItems = [
    { label: 'Воркеры', href: '/workers', icon: Cpu },
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
          className="flex items-center justify-between"
        >
          <div>
            <h1 className="text-3xl font-bold flex items-center gap-2">
              <Cpu className="h-8 w-8 text-primary" />
              Управление воркерами и моделями
            </h1>
            <p className="text-muted-foreground mt-2">
              Настройка провайдеров AI, моделей и количества воркеров для нормализации
            </p>
          </div>
          <Button onClick={fetchConfig} variant="outline" size="sm" disabled={loading}>
            <RefreshCw className={cn("h-4 w-4 mr-2", loading && "animate-spin")} />
            Обновить
          </Button>
        </motion.div>
      </FadeIn>

      {error && error !== 'Конфигурация не загружена' && (
        <ErrorState
          title="Ошибка"
          message={error}
          variant="destructive"
          dismissible
          onDismiss={() => setError(null)}
          className="mb-4"
        />
      )}

      {isFallbackConfig && (
        <Alert className="mb-4 border-amber-500 bg-amber-50 dark:bg-amber-950">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-4 w-4 text-amber-600" />
            <AlertTitle>Backend конфигурации недоступен</AlertTitle>
          </div>
          <AlertDescription className="mt-2 space-y-1 text-sm">
            <p>Показаны данные по умолчанию. Изменения будут доступны после восстановления соединения с сервером.</p>
            <p className="text-muted-foreground">
              {config?.fallbackReason || 'Перезапустите Go-сервер на порту 9999 и попробуйте снова.'}
            </p>
            {config?.lastSync && (
              <p className="text-xs text-muted-foreground">
                Последняя попытка подключения: {new Date(config.lastSync).toLocaleTimeString('ru-RU')}
              </p>
            )}
          </AlertDescription>
        </Alert>
      )}

      {success && (
        <Alert className="mb-4 border-green-500 bg-green-50 dark:bg-green-950">
          <CheckCircle2 className="h-4 w-4 text-green-600" />
          <AlertDescription>
            <div className="flex items-center justify-between">
              <span className="text-green-800 dark:text-green-200">{success}</span>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSuccess(null)}
                className="h-6 px-2 text-green-800 dark:text-green-200 hover:text-green-900 dark:hover:text-green-100"
              >
                <XCircle className="h-3 w-3" />
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      <Tabs defaultValue="providers" className="space-y-4">
        <TabsList>
          <TabsTrigger value="providers">Провайдеры и модели</TabsTrigger>
          <TabsTrigger value="workers">Воркеры</TabsTrigger>
          <TabsTrigger value="monitoring">Мониторинг</TabsTrigger>
          <TabsTrigger value="defaults">Настройки по умолчанию</TabsTrigger>
        </TabsList>

        <TabsContent value="providers" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Провайдеры AI</CardTitle>
              <CardDescription>
                Управление провайдерами и их моделями. Приоритет: меньше = выше приоритет.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {providers.map(([name, provider]) => (
                <Card key={name} className="border-2">
                  <CardHeader className="pb-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => toggleProviderExpanded(name)}
                        >
                          {expandedProviders.has(name) ? (
                            <ChevronUp className="h-4 w-4" />
                          ) : (
                            <ChevronDown className="h-4 w-4" />
                          )}
                        </Button>
                        <CardTitle className="text-lg">
                          {name === 'arliai' ? 'Arli AI' :
                           name === 'openrouter' ? 'OpenRouter' :
                           name === 'huggingface' ? 'Hugging Face' :
                           name}
                        </CardTitle>
                        {provider.enabled ? (
                          <Badge variant="default">Включен</Badge>
                        ) : (
                          <Badge variant="secondary">Выключен</Badge>
                        )}
                        {config?.default_provider === name && (
                          <Badge variant="outline">По умолчанию</Badge>
                        )}
                      </div>
                      <div className="flex items-center gap-2">
                        <Button
                          variant={provider.enabled ? "destructive" : "default"}
                          size="sm"
                          onClick={() => toggleProvider(name)}
                          disabled={saving}
                        >
                          {provider.enabled ? 'Выключить' : 'Включить'}
                        </Button>
                        {!provider.enabled && (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setDefaultProvider(name)}
                            disabled={saving}
                          >
                            Установить по умолчанию
                          </Button>
                        )}
                      </div>
                    </div>
                  </CardHeader>
                  {expandedProviders.has(name) && (
                    <CardContent className="space-y-4 pt-0">
                      <div className="space-y-4">
                        <div>
                          <div className="flex items-center justify-between mb-2">
                            <Label htmlFor={`api-key-${name}`} className="flex items-center gap-2">
                              <Key className="h-4 w-4" />
                              API Ключ
                            </Label>
                            {(name === 'arliai' || name === 'openrouter' || name === 'huggingface') && (
                              <div className="flex gap-2">
                                <Button
                                  type="button"
                                  variant="outline"
                                  size="sm"
                                  onClick={() => testAPIKey(name)}
                                  disabled={saving || apiKeyStatus[name]?.testing}
                                  className="h-8"
                                >
                                  {apiKeyStatus[name]?.testing ? (
                                    <>
                                      <Loader2 className="h-3 w-3 mr-2 animate-spin" />
                                      Проверка...
                                    </>
                                  ) : (
                                    <>
                                      <TestTube className="h-3 w-3 mr-2" />
                                      Проверить
                                    </>
                                  )}
                                </Button>
                                <Button
                                  type="button"
                                  variant="outline"
                                  size="sm"
                                  onClick={() => refreshModels(name)}
                                  disabled={saving || refreshingModels[name] || !apiKeyStatus[name]?.connected}
                                  className="h-8"
                                  title="Обновить список доступных моделей из API"
                                >
                                  {refreshingModels[name] ? (
                                    <>
                                      <Loader2 className="h-3 w-3 mr-2 animate-spin" />
                                      Обновление...
                                    </>
                                  ) : (
                                    <>
                                      <Download className="h-3 w-3 mr-2" />
                                      Обновить модели
                                    </>
                                  )}
                                </Button>
                                <Button
                                  type="button"
                                  variant="outline"
                                  size="sm"
                                  onClick={() => runBenchmark(name)}
                                  disabled={saving || runningBenchmark[name] || !apiKeyStatus[name]?.connected}
                                  className="h-8"
                                  title="Запустить бэнчмарк моделей провайдера с датасетом из нормализации"
                                >
                                  {runningBenchmark[name] ? (
                                    <>
                                      <Loader2 className="h-3 w-3 mr-2 animate-spin" />
                                      Бэнчмарк...
                                    </>
                                  ) : (
                                    <>
                                      <BarChart3 className="h-3 w-3 mr-2" />
                                      Бэнчмарк
                                    </>
                                  )}
                                </Button>
                              </div>
                            )}
                          </div>
                          <div className="space-y-2">
                            <div className="flex gap-2">
                              <Input
                                id={`api-key-${name}`}
                                type="password"
                                value={apiKeys[name] || ''}
                                onChange={(e) => updateProviderAPIKey(name, e.target.value)}
                                placeholder={provider.has_api_key ? '••••••••••••••••••••••••••••••••••••' : 'Введите API ключ'}
                                disabled={saving || savingApiKey[name]}
                                className="flex-1 font-mono text-sm"
                                onKeyDown={(e) => {
                                  if (e.key === 'Enter' && !saving && !savingApiKey[name]) {
                                    saveProviderAPIKey(name)
                                  }
                                }}
                              />
                              <Button
                                type="button"
                                variant="default"
                                size="sm"
                                onClick={() => saveProviderAPIKey(name)}
                                disabled={saving || savingApiKey[name]}
                                className="h-10"
                              >
                                {savingApiKey[name] ? (
                                  <>
                                    <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                                    Сохранение...
                                  </>
                                ) : (
                                  <>
                                    <Save className="h-4 w-4 mr-2" />
                                    Сохранить
                                  </>
                                )}
                              </Button>
                            </div>
                            {apiKeyStatus[name] && (
                              <div className="flex items-center gap-2 text-xs">
                                {apiKeyStatus[name].testing ? (
                                  <div className="flex items-center gap-2 text-muted-foreground">
                                    <Loader2 className="h-3 w-3 animate-spin" />
                                    <span>Проверка подключения...</span>
                                  </div>
                                ) : apiKeyStatus[name].connected ? (
                                  <div className="flex items-center gap-2 text-green-600 dark:text-green-400">
                                    <CheckCircle2 className="h-3 w-3" />
                                    <span>Подключение установлено</span>
                                  </div>
                                ) : apiKeyStatus[name].error ? (
                                  <div className="flex items-center gap-2 text-red-600 dark:text-red-400">
                                    <XCircle className="h-3 w-3" />
                                    <span>{apiKeyStatus[name].error}</span>
                                  </div>
                                ) : null}
                              </div>
                            )}
                            {provider.has_api_key && !apiKeyStatus[name] && (
                              <div className="flex items-center gap-2 text-xs text-blue-600 dark:text-blue-400">
                                <CheckCircle2 className="h-3 w-3" />
                                <span>API ключ установлен на сервере</span>
                              </div>
                            )}
                          </div>
                            <div className="space-y-1">
                              <p className="text-xs text-muted-foreground">
                                API ключ сохраняется на сервере. Оставьте пустым, чтобы использовать из переменных окружения.
                              </p>
                              {apiKeyStatus[name]?.connected && (
                                <div className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
                                  <Info className="h-3 w-3" />
                                  <span>После сохранения ключа список моделей обновится автоматически</span>
                                </div>
                              )}
                            </div>
                        </div>
                        <div className="grid grid-cols-2 gap-4">
                          <div>
                            <Label>Приоритет (меньше = выше)</Label>
                            <Input
                              type="number"
                              value={provider.priority}
                              onChange={(e) => {
                                const val = parseInt(e.target.value)
                                if (!isNaN(val) && val > 0) {
                                  updateProviderPriority(name, val)
                                }
                              }}
                              min="1"
                              disabled={saving}
                              className="w-full"
                            />
                            <p className="text-xs text-muted-foreground mt-1">
                              Провайдер с меньшим приоритетом выбирается первым
                            </p>
                          </div>
                          <div>
                            <Label>Макс. воркеров</Label>
                            <Input
                              type="number"
                              value={provider.max_workers}
                              onChange={(e) => {
                                const val = parseInt(e.target.value)
                                if (!isNaN(val) && val >= 1 && val <= 100) {
                                  updateProviderWorkers(name, val)
                                }
                              }}
                              min="1"
                              max="100"
                              disabled={saving}
                              className="w-full"
                            />
                            <p className="text-xs text-muted-foreground mt-1">
                              Максимальное количество параллельных запросов
                            </p>
                          </div>
                          <div>
                            <Label>Rate Limit (запросов/мин)</Label>
                            <Input
                              type="number"
                              value={provider.rate_limit}
                              disabled
                              className="bg-muted"
                            />
                          </div>
                          <div>
                            <Label>Timeout</Label>
                            <Input
                              value={provider.timeout}
                              disabled
                              className="bg-muted"
                            />
                          </div>
                        </div>
                      </div>

                      {/* Результаты бэнчмарка */}
                      {benchmarkResults[name] && benchmarkResults[name].models && Array.isArray(benchmarkResults[name].models) && benchmarkResults[name].models.length > 0 && (
                        <div className="space-y-4">
                          <Tabs defaultValue="table" className="w-full">
                            <TabsList className="grid w-full grid-cols-5">
                              <TabsTrigger value="table">Таблица</TabsTrigger>
                              <TabsTrigger value="radar">Радар</TabsTrigger>
                              <TabsTrigger value="heatmap">Тепловая карта</TabsTrigger>
                              <TabsTrigger value="timeseries">Временные ряды</TabsTrigger>
                              <TabsTrigger value="recommendations">Рекомендации</TabsTrigger>
                            </TabsList>
                            
                            <TabsContent value="table" className="space-y-4">
                              <BenchmarkResultsTable
                                results={benchmarkResults[name].models}
                                timestamp={benchmarkResults[name].timestamp}
                                onClose={() => setBenchmarkResults(prev => {
                                  const newResults = { ...prev }
                                  delete newResults[name]
                                  return newResults
                                })}
                              />
                            </TabsContent>
                            
                            <TabsContent value="radar" className="space-y-4">
                              <BenchmarkRadarChart
                                results={benchmarkResults[name].models}
                                maxModels={5}
                              />
                            </TabsContent>
                            
                            <TabsContent value="heatmap" className="space-y-4">
                              <PerformanceHeatmap
                                results={benchmarkResults[name].models}
                                metric="speed"
                              />
                            </TabsContent>
                            
                            <TabsContent value="timeseries" className="space-y-4">
                              <BenchmarkTimeSeries
                                provider={name}
                                timeRange="7d"
                              />
                            </TabsContent>
                            
                            <TabsContent value="recommendations" className="space-y-4">
                              <ModelRecommendationEngine
                                results={benchmarkResults[name].models}
                                priority="balanced"
                                scenario="realtime"
                              />
                            </TabsContent>
                          </Tabs>
                        </div>
                      )}

                      <div>
                        <div className="flex items-center justify-between mb-2">
                          <Label className="block">
                            Модели ({provider.models.length})
                            {provider.models.length > 0 && (
                              <span className="ml-2 text-xs text-muted-foreground font-normal">
                                (прокрутите вниз, чтобы увидеть все)
                              </span>
                            )}
                          </Label>
                          {provider.models.length === 0 && (
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => refreshModels(name)}
                              disabled={saving || refreshingModels[name] || !apiKeyStatus[name]?.connected}
                              className="h-8"
                            >
                              {refreshingModels[name] ? (
                                <>
                                  <Loader2 className="h-3 w-3 mr-2 animate-spin" />
                                  Обновление...
                                </>
                              ) : (
                                <>
                                  <Download className="h-3 w-3 mr-2" />
                                  Загрузить модели
                                </>
                              )}
                            </Button>
                          )}
                        </div>
                        {provider.models.length === 0 ? (
                          <div className="text-center py-6 border rounded-lg bg-muted/50">
                            <Cpu className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
                            <p className="text-sm text-muted-foreground mb-2">Модели не загружены</p>
                            <p className="text-xs text-muted-foreground">
                              {apiKeyStatus[name]?.connected 
                                ? 'Нажмите "Загрузить модели" для получения списка доступных моделей'
                                : 'Сохраните и проверьте API ключ, затем обновите список моделей'}
                            </p>
                          </div>
                        ) : (
                          <div className="space-y-2 max-h-[600px] overflow-y-auto overflow-x-hidden pr-2">
                            <div className="sticky top-0 bg-background z-10 pb-2 mb-2 border-b">
                              <p className="text-xs text-muted-foreground">
                                Всего моделей: {provider.models.length} | 
                                Включено: {provider.models.filter(m => m.enabled).length} | 
                                Выключено: {provider.models.filter(m => !m.enabled).length}
                              </p>
                            </div>
                            {provider.models.map((model) => (
                              <Card key={model.name} className="p-3 hover:shadow-md transition-shadow">
                                <div className="flex items-center justify-between">
                                  <div className="flex items-center gap-3 flex-1">
                                    <div className="flex-1">
                                      <div className="flex items-center gap-2 flex-wrap">
                                        <span className="font-medium">{model.name}</span>
                                        {model.enabled ? (
                                          <Badge variant="default" className="text-xs">Включена</Badge>
                                        ) : (
                                          <Badge variant="secondary" className="text-xs">Выключена</Badge>
                                        )}
                                        {config?.default_model === model.name && (
                                          <Badge variant="outline" className="text-xs">По умолчанию</Badge>
                                        )}
                                      </div>
                                      <div className="flex items-center gap-3 text-sm text-muted-foreground mt-1 flex-wrap">
                                        <span>Приоритет: <strong>{model.priority}</strong></span>
                                        {model.speed && (
                                          <span className="flex items-center gap-1">
                                            <Zap className="h-3 w-3" />
                                            {model.speed}
                                          </span>
                                        )}
                                        {model.quality && (
                                          <span className="flex items-center gap-1">
                                            <Award className="h-3 w-3" />
                                            {model.quality}
                                          </span>
                                        )}
                                        {model.max_tokens && (
                                          <span className="text-xs">Max tokens: {model.max_tokens.toLocaleString('ru-RU')}</span>
                                        )}
                                      </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                      <div className="flex flex-col items-end gap-1">
                                        <Label className="text-xs text-muted-foreground">Приоритет</Label>
                                        <Input
                                          type="number"
                                          value={model.priority}
                                          onChange={(e) => {
                                            const val = parseInt(e.target.value)
                                            if (!isNaN(val) && val > 0) {
                                              updateModelPriority(name, model.name, val)
                                            }
                                          }}
                                          min="1"
                                          className="w-20"
                                          disabled={saving}
                                        />
                                      </div>
                                      <div className="flex flex-col gap-1">
                                        <Button
                                          variant={model.enabled ? "destructive" : "default"}
                                          size="sm"
                                          onClick={() => toggleModel(name, model.name)}
                                          disabled={saving}
                                          className="w-16"
                                        >
                                          {model.enabled ? 'Выкл' : 'Вкл'}
                                        </Button>
                                        {model.enabled && (
                                          <Button
                                            variant="outline"
                                            size="sm"
                                            onClick={() => setDefaultModel(name, model.name)}
                                            disabled={saving || config?.default_model === model.name}
                                            className="w-16 text-xs"
                                          >
                                            {config?.default_model === model.name ? 'По умолчанию' : 'По умолчанию'}
                                          </Button>
                                        )}
                                      </div>
                                    </div>
                                  </div>
                                </div>
                              </Card>
                            ))}
                          </div>
                        )}
                      </div>
                    </CardContent>
                  )}
                </Card>
              ))}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="monitoring" className="space-y-4">
          <Card>
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Activity className="h-5 w-5" />
                    Мониторинг производительности
                  </CardTitle>
                  <CardDescription>
                    Метрики работы AI воркеров и провайдеров
                  </CardDescription>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={fetchWorkerMetrics}
                  disabled={loadingMetrics}
                >
                  <RefreshCw className={`h-4 w-4 mr-2 ${loadingMetrics ? 'animate-spin' : ''}`} />
                  Обновить
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              {loadingMetrics && !workerMetrics ? (
                <LoadingState message="Загрузка метрик..." />
              ) : workerMetrics?.ai ? (
                <div className="space-y-6">
                  {/* Общая статистика */}
                  <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                    <StatCard
                      title="Всего запросов"
                      value={workerMetrics.ai.total_requests.toLocaleString('ru-RU')}
                      description="За все время"
                      icon={Activity}
                      variant="default"
                    />
                    <StatCard
                      title="Успешных"
                      value={workerMetrics.ai.successful.toLocaleString('ru-RU')}
                      description={`${(workerMetrics.ai.success_rate * 100).toFixed(1)}% успешности`}
                      icon={CheckCircle2}
                      variant="success"
                      progress={workerMetrics.ai.success_rate * 100}
                    />
                    <StatCard
                      title="Ошибок"
                      value={workerMetrics.ai.failed.toLocaleString('ru-RU')}
                      description={`${((1 - workerMetrics.ai.success_rate) * 100).toFixed(1)}% ошибок`}
                      icon={XCircle}
                      variant={workerMetrics.ai.failed > 0 ? 'danger' : 'default'}
                    />
                    <StatCard
                      title="Средняя латентность"
                      value={`${workerMetrics.ai.average_latency_ms.toFixed(0)}ms`}
                      description="Время ответа"
                      icon={Clock}
                      variant="primary"
                    />
                  </div>

                  {/* Детальная статистика */}
                  <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <TrendingUp className="h-5 w-5" />
                          Успешность запросов
                        </CardTitle>
                        <CardDescription>Процент успешных AI запросов</CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-4">
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium">Success Rate</span>
                            <span className="text-2xl font-bold">
                              {(workerMetrics.ai.success_rate * 100).toFixed(1)}%
                            </span>
                          </div>
                          <div className="h-3 bg-secondary rounded-full overflow-hidden">
                            <div
                              className="h-full bg-green-500 transition-all"
                              style={{ width: `${workerMetrics.ai.success_rate * 100}%` }}
                            />
                          </div>
                          <div className="grid grid-cols-2 gap-4 pt-2">
                            <div>
                              <p className="text-sm text-muted-foreground">Успешно</p>
                              <p className="text-lg font-semibold text-green-600">
                                {workerMetrics.ai.successful.toLocaleString('ru-RU')}
                              </p>
                            </div>
                            <div>
                              <p className="text-sm text-muted-foreground">Ошибки</p>
                              <p className="text-lg font-semibold text-red-600">
                                {workerMetrics.ai.failed.toLocaleString('ru-RU')}
                              </p>
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>

                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <BarChart3 className="h-5 w-5" />
                          Производительность
                        </CardTitle>
                        <CardDescription>Метрики производительности системы</CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="space-y-4">
                          <div className="space-y-2">
                            <div className="flex items-center justify-between">
                              <span className="text-sm font-medium">Пропускная способность</span>
                              <span className="text-lg font-bold">
                                {workerMetrics.throughput_items_per_second.toFixed(2)} записей/сек
                              </span>
                            </div>
                            <div className="flex items-center justify-between">
                              <span className="text-sm font-medium">Время работы</span>
                              <span className="text-lg font-bold">
                                {Math.floor(workerMetrics.uptime_seconds / 3600)}ч{' '}
                                {Math.floor((workerMetrics.uptime_seconds % 3600) / 60)}м
                              </span>
                            </div>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  </div>

                  {/* Кеширование */}
                  {workerMetrics.cache && (
                    <Card>
                      <CardHeader>
                        <CardTitle className="flex items-center gap-2">
                          <Zap className="h-5 w-5" />
                          Кеширование
                        </CardTitle>
                        <CardDescription>Эффективность кеша AI результатов</CardDescription>
                      </CardHeader>
                      <CardContent>
                        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                          <div>
                            <p className="text-sm text-muted-foreground mb-1">Hit Rate</p>
                            <p className="text-2xl font-bold">
                              {(workerMetrics.cache.hit_rate * 100).toFixed(1)}%
                            </p>
                          </div>
                          <div>
                            <p className="text-sm text-muted-foreground mb-1">Попадания</p>
                            <p className="text-2xl font-bold text-green-600">
                              {workerMetrics.cache.hits.toLocaleString('ru-RU')}
                            </p>
                          </div>
                          <div>
                            <p className="text-sm text-muted-foreground mb-1">Размер кеша</p>
                            <p className="text-2xl font-bold">
                              {workerMetrics.cache.size.toLocaleString('ru-RU')}
                            </p>
                          </div>
                        </div>
                      </CardContent>
                    </Card>
                  )}
                </div>
              ) : (
                <div className="text-center py-8">
                  <AlertCircle className="h-12 w-12 text-muted-foreground mx-auto mb-4" />
                  <p className="text-muted-foreground">Метрики недоступны</p>
                  <p className="text-sm text-muted-foreground mt-2">
                    Убедитесь, что система мониторинга активна
                  </p>
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="workers" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Настройки воркеров</CardTitle>
              <CardDescription>
                Глобальные настройки количества воркеров для обработки данных
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label htmlFor="global-workers">Глобальный максимум воркеров</Label>
                <div className="flex items-center gap-4 mt-2">
                  <Input
                    id="global-workers"
                    type="number"
                    value={config?.global_max_workers}
                    onChange={(e) => {
                      const val = parseInt(e.target.value)
                      if (!isNaN(val) && val >= 1 && val <= 100) {
                        updateGlobalWorkers(val)
                      }
                    }}
                    min="1"
                    max="100"
                    className="w-32"
                    disabled={saving}
                  />
                  <div className="flex-1">
                    <p className="text-sm text-muted-foreground">
                      Фактическое количество будет минимумом из глобального и провайдерского лимита
                    </p>
                    <p className="text-xs text-muted-foreground mt-1">
                      Текущее ограничение: {Math.min(
                        config?.global_max_workers || 0,
                        ...providers.map(([_, p]) => p.max_workers)
                      )} воркеров
                    </p>
                  </div>
                </div>
              </div>

              <div className="pt-4 border-t">
                <h3 className="font-semibold mb-3">Текущие настройки по провайдерам:</h3>
                <div className="space-y-2">
                  {providers.map(([name, provider]) => (
                    <div key={name} className="flex items-center justify-between p-2 bg-muted rounded">
                      <span className="font-medium">{name}</span>
                      <Badge>{provider.max_workers} воркеров</Badge>
                    </div>
                  ))}
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="defaults" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Настройки по умолчанию</CardTitle>
              <CardDescription>
                Выбор провайдера и модели, которые будут использоваться по умолчанию
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div>
                <Label>Провайдер по умолчанию</Label>
                <Select
                  value={config?.default_provider || undefined}
                  onValueChange={(providerName) => {
                    if (!providerName) return
                    setDefaultProvider(providerName)
                  }}
                  disabled={saving}
                >
                  <SelectTrigger className="mt-2">
                    <SelectValue placeholder="Выберите провайдера" />
                  </SelectTrigger>
                  <SelectContent>
                    {providers.length > 0 ? (
                      providers.map(([name, provider]) => (
                        <SelectItem key={name} value={name || `provider-${name}`} disabled={!provider.enabled}>
                          {name} {!provider.enabled && '(выключен)'}
                        </SelectItem>
                      ))
                    ) : (
                      <SelectItem value="no-providers" disabled>
                        Нет доступных провайдеров
                      </SelectItem>
                    )}
                  </SelectContent>
                </Select>
              </div>

              <div>
                <Label>Модель по умолчанию</Label>
                <Select
                  value={config?.default_model || undefined}
                  onValueChange={(modelName) => {
                    if (!modelName) return
                    const provider = providers.find(([_, p]) => 
                      p.models.some(m => m.name === modelName)
                    )
                    if (provider) {
                      setDefaultModel(provider[0], modelName)
                    }
                  }}
                  disabled={saving}
                >
                  <SelectTrigger className="mt-2">
                    <SelectValue placeholder="Выберите модель" />
                  </SelectTrigger>
                  <SelectContent>
                    {providers.length > 0 && providers.some(([_, p]) => p.models.some(m => m.enabled)) ? (
                      providers.map(([providerName, provider]) =>
                        provider.models
                          .filter(m => m.enabled && m.name)
                          .map((model) => (
                            <SelectItem key={`${providerName}-${model.name}`} value={model.name || `model-${providerName}-${model.name}`}>
                              {model.name} ({providerName})
                            </SelectItem>
                          ))
                      )
                    ) : (
                      <SelectItem value="no-models" disabled>
                        Нет доступных моделей
                      </SelectItem>
                    )}
                  </SelectContent>
                </Select>
              </div>

              <Alert>
                <Info className="h-4 w-4" />
                <AlertDescription>
                  <p className="font-semibold mb-2">Как работает выбор провайдера и модели:</p>
                  <ul className="list-disc list-inside space-y-1 text-sm">
                    <li>Система автоматически выбирает провайдера с наименьшим приоритетом среди включенных</li>
                    <li>Затем выбирается модель с наименьшим приоритетом среди включенных моделей этого провайдера</li>
                    <li>Настройки по умолчанию используются, если нет активных провайдеров/моделей</li>
                    <li>Приоритет: меньшее число = выше приоритет (1 &lt; 2 &lt; 3)</li>
                  </ul>
                </AlertDescription>
              </Alert>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

