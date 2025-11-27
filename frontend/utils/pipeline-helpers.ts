/**
 * Утилиты для работы с пайплайном нормализации
 */

import { getOverallStatus, getStatusText, getStatusColor, getStatusVariant } from './normalization-helpers'

/**
 * Получает статус этапа на основе данных из API или fallback логики
 */
export function getStageStatus(
  stageId: string,
  index: number,
  pipelineData: any,
  activeProcess?: string | null
): string {
  // Используем данные из API если доступны
  const stageData = pipelineData?.stages?.find((s: any) => s.id === stageId)
  if (stageData?.status) {
    return stageData.status
  }

  // Fallback логика на основе activeProcess
  if (activeProcess === stageId) return 'active'
  const activeIndex = pipelineData?.stages?.findIndex((s: any) => s.id === activeProcess) ?? -1
  if (activeIndex === -1) return 'completed'
  return index < activeIndex ? 'completed' : 'pending'
}

/**
 * Получает метрики для этапа
 */
export function getStageMetrics(stageId: string, pipelineData: any): any {
  const stageData = pipelineData?.stages?.find((s: any) => s.id === stageId)
  return stageData?.metrics || null
}

/**
 * Получает общий статус пайплайна
 */
export function getPipelineOverallStatus(pipelineData: any): string {
  if (!pipelineData?.stages || pipelineData.stages.length === 0) {
    return 'pending'
  }

  return getOverallStatus(
    pipelineData.stages.map((s: any) => ({ status: s.status || 'pending' }))
  )
}

/**
 * Получает локализованный текст статуса пайплайна
 */
export function getPipelineStatusText(status: string): string {
  return getStatusText(status)
}

/**
 * Получает цвет для статуса пайплайна
 */
export function getPipelineStatusColor(status: string): string {
  return getStatusColor(status)
}

/**
 * Получает вариант badge для статуса пайплайна
 */
export function getPipelineStatusVariant(status: string): 'default' | 'secondary' | 'destructive' | 'outline' {
  return getStatusVariant(status)
}

/**
 * Проверяет, активен ли этап
 */
export function isStageActive(status: string): boolean {
  return status === 'active' || status === 'processing'
}

/**
 * Проверяет, завершен ли этап
 */
export function isStageCompleted(status: string): boolean {
  return status === 'completed' || status === 'finished'
}

/**
 * Проверяет, есть ли ошибка в этапе
 */
export function isStageError(status: string): boolean {
  return status === 'error' || status === 'failed'
}

