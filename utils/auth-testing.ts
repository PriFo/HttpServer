/**
 * Утилиты для тестирования аутентификации и авторизации
 * 
 * Предоставляет функции для создания тестовых JWT токенов
 * и работы с ролями пользователей в тестах
 */

import { APIRequestContext } from '@playwright/test'

export interface JWTPayload {
  userId: string
  email?: string
  name?: string
  roles?: string[]
  exp?: number
  iat?: number
  clientId?: number
}

/**
 * Создает простой JWT токен для тестирования
 * 
 * ВАЖНО: Это упрощенная версия для тестов. В production используйте
 * библиотеку jsonwebtoken или jose для создания подписанных токенов.
 * 
 * @param payload - Данные для токена
 * @returns JWT токен (без подписи для тестов)
 */
export function createTestJWTToken(payload: JWTPayload): string {
  const header = {
    alg: 'HS256',
    typ: 'JWT',
  }

  const now = Math.floor(Date.now() / 1000)
  const fullPayload: JWTPayload = {
    ...payload,
    iat: payload.iat || now,
    exp: payload.exp || now + 3600, // 1 час по умолчанию
  }

  // Кодируем header и payload в base64url
  const encodedHeader = Buffer.from(JSON.stringify(header))
    .toString('base64url')
    .replace(/=/g, '')
  const encodedPayload = Buffer.from(JSON.stringify(fullPayload))
    .toString('base64url')
    .replace(/=/g, '')

  // Для тестов не добавляем реальную подпись
  // В production здесь должна быть подпись с секретным ключом
  return `${encodedHeader}.${encodedPayload}.test_signature`
}

/**
 * Создает токен для администратора
 */
export function createAdminToken(userId?: string): string {
  return createTestJWTToken({
    userId: userId || `admin-${Date.now()}`,
    email: 'admin@test.com',
    name: 'Admin User',
    roles: ['admin'],
  })
}

/**
 * Создает токен для менеджера клиента
 */
export function createManagerToken(clientId: number, userId?: string): string {
  return createTestJWTToken({
    userId: userId || `manager-${Date.now()}`,
    email: 'manager@test.com',
    name: 'Manager User',
    roles: ['manager'],
    clientId,
  })
}

/**
 * Создает токен для наблюдателя
 */
export function createViewerToken(userId?: string): string {
  return createTestJWTToken({
    userId: userId || `viewer-${Date.now()}`,
    email: 'viewer@test.com',
    name: 'Viewer User',
    roles: ['viewer'],
  })
}

/**
 * Добавляет заголовок Authorization к запросу
 */
export function addAuthHeader(
  headers: Record<string, string>,
  token: string
): Record<string, string> {
  return {
    ...headers,
    Authorization: `Bearer ${token}`,
  }
}

/**
 * Устанавливает токен в контекст запроса Playwright
 */
export async function setAuthContext(
  context: APIRequestContext,
  token: string
): Promise<void> {
  await context.setExtraHTTPHeaders({
    Authorization: `Bearer ${token}`,
  })
}

/**
 * Проверяет, содержит ли текст сообщение об ошибке доступа
 */
export function isAccessDeniedError(text: string): boolean {
  const lowerText = text.toLowerCase()
  return (
    lowerText.includes('403') ||
    lowerText.includes('access denied') ||
    lowerText.includes('forbidden') ||
    lowerText.includes('недостаточно прав') ||
    lowerText.includes('unauthorized') ||
    lowerText.includes('401')
  )
}

/**
 * Проверяет, содержит ли ответ ошибку авторизации
 */
export function isUnauthorizedError(status: number, body?: string): boolean {
  if (status === 401 || status === 403) {
    return true
  }

  if (body && isAccessDeniedError(body)) {
    return true
  }

  return false
}

/**
 * Парсит JWT токен и возвращает payload
 */
export function parseJWTToken(token: string): JWTPayload | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) {
      return null
    }

    const payload = parts[1]
    const decoded = Buffer.from(payload, 'base64url').toString('utf-8')
    return JSON.parse(decoded) as JWTPayload
  } catch (error) {
    console.error('Error parsing JWT token:', error)
    return null
  }
}

/**
 * Проверяет, имеет ли пользователь указанную роль
 */
export function hasRole(payload: JWTPayload | null, role: string): boolean {
  if (!payload || !payload.roles) {
    return false
  }
  return payload.roles.includes(role)
}

/**
 * Проверяет, является ли пользователь администратором
 */
export function isAdmin(payload: JWTPayload | null): boolean {
  return hasRole(payload, 'admin')
}

/**
 * Проверяет, имеет ли пользователь доступ к клиенту
 */
export function hasClientAccess(
  payload: JWTPayload | null,
  clientId: number
): boolean {
  // Администратор имеет доступ ко всем клиентам
  if (isAdmin(payload)) {
    return true
  }

  // Менеджер имеет доступ только к своему клиенту
  if (payload?.clientId === clientId) {
    return true
  }

  return false
}

