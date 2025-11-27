/**
 * Утилиты для экспорта данных мониторинга
 */

import type { MonitoringData, ProviderMetrics } from '@/types/monitoring'

/**
 * Экспорт данных мониторинга в CSV
 */
export function exportMonitoringToCSV(data: MonitoringData): void {
  const headers = [
    'Провайдер',
    'Статус',
    'Активные каналы',
    'Текущие запросы',
    'Всего запросов',
    'Успешных',
    'Неудачных',
    'Запросов/сек',
    'Средняя задержка (мс)',
    'Последний запрос',
  ]

  const rows = data.providers.map(provider => [
    provider.name,
    provider.status === 'active' ? 'Активен' : provider.status === 'idle' ? 'Ожидание' : 'Ошибка',
    provider.active_channels.toString(),
    provider.current_requests.toString(),
    provider.total_requests.toString(),
    provider.successful_requests.toString(),
    provider.failed_requests.toString(),
    provider.requests_per_second.toFixed(2),
    provider.average_latency_ms.toFixed(0),
    provider.last_request_time || 'N/A',
  ])

  // Добавляем системную статистику
  rows.push([])
  rows.push(['Системная статистика', '', '', '', '', '', '', '', '', ''])
  rows.push([
    'Всего провайдеров',
    data.system.total_providers.toString(),
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])
  rows.push([
    'Активных провайдеров',
    data.system.active_providers.toString(),
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])
  rows.push([
    'Всего запросов',
    data.system.total_requests.toString(),
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])
  rows.push([
    'Успешных запросов',
    data.system.total_successful.toString(),
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])
  rows.push([
    'Неудачных запросов',
    data.system.total_failed.toString(),
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])
  rows.push([
    'Запросов/сек (система)',
    data.system.system_requests_per_second.toFixed(2),
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])
  rows.push([
    'Время обновления',
    data.system.timestamp,
    '',
    '',
    '',
    '',
    '',
    '',
    '',
    '',
  ])

  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${cell}"`).join(',')),
  ].join('\n')

  const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `monitoring-${new Date().toISOString().split('T')[0]}.csv`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

/**
 * Экспорт данных мониторинга в JSON
 */
export function exportMonitoringToJSON(data: MonitoringData): void {
  const jsonContent = JSON.stringify(data, null, 2)
  const blob = new Blob([jsonContent], { type: 'application/json' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `monitoring-${new Date().toISOString().split('T')[0]}.json`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

/**
 * Экспорт метрик провайдера в CSV
 */
export function exportProviderMetricsToCSV(provider: ProviderMetrics): void {
  const headers = [
    'Метрика',
    'Значение',
  ]

  const rows = [
    ['Провайдер', provider.name],
    ['ID', provider.id],
    ['Статус', provider.status === 'active' ? 'Активен' : provider.status === 'idle' ? 'Ожидание' : 'Ошибка'],
    ['Активные каналы', provider.active_channels.toString()],
    ['Текущие запросы', provider.current_requests.toString()],
    ['Всего запросов', provider.total_requests.toString()],
    ['Успешных запросов', provider.successful_requests.toString()],
    ['Неудачных запросов', provider.failed_requests.toString()],
    ['Запросов в секунду', provider.requests_per_second.toFixed(2)],
    ['Средняя задержка (мс)', provider.average_latency_ms.toFixed(0)],
    ['Последний запрос', provider.last_request_time || 'N/A'],
    ['Процент успешности', provider.total_requests > 0 
      ? ((provider.successful_requests / provider.total_requests) * 100).toFixed(2) + '%'
      : '0%'],
  ]

  const csvContent = [
    headers.join(','),
    ...rows.map(row => row.map(cell => `"${cell}"`).join(',')),
  ].join('\n')

  const blob = new Blob(['\uFEFF' + csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', `provider-${provider.id}-${new Date().toISOString().split('T')[0]}.csv`)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}

