// Типы для мониторинга провайдеров

export interface ProviderMetrics {
  id: string
  name: string
  active_channels: number
  current_requests: number
  total_requests: number
  successful_requests: number
  failed_requests: number
  average_latency_ms: number
  last_request_time: string
  status: 'active' | 'idle' | 'error'
  requests_per_second: number
}

export interface SystemStats {
  total_providers: number
  active_providers: number
  total_requests: number
  total_successful: number
  total_failed: number
  system_requests_per_second: number
  timestamp: string
  cpu_usage?: number
  memory_usage?: number
}

export interface MonitoringData {
  providers: ProviderMetrics[]
  system: SystemStats
}

export interface WorkerStatus {
  is_running: boolean
  workers_count: number
  stopped: boolean
  current_tasks: Array<{
    worker_id: number
    normalized_name: string
    category: string
    merged_count: number
    index: number
  }>
}

