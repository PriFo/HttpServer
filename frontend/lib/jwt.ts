/**
 * Утилиты для работы с JWT токенами
 * 
 * Предоставляет функции для валидации и извлечения информации из JWT токенов
 * Используется в proxy и API routes для аутентификации
 */

export interface JWTPayload {
  userId: string
  email?: string
  name?: string
  roles?: string[]
  exp?: number
  iat?: number
}

/**
 * Извлекает JWT токен из заголовка Authorization
 * Поддерживает форматы: "Bearer <token>" и просто "<token>"
 */
export function extractTokenFromHeader(authHeader: string | null): string | null {
  if (!authHeader) {
    return null
  }

  // Поддерживаем формат "Bearer <token>"
  if (authHeader.startsWith('Bearer ')) {
    return authHeader.substring(7)
  }

  // Также поддерживаем просто токен без префикса
  return authHeader
}

/**
 * Парсит JWT токен без валидации подписи (для извлечения payload)
 * 
 * ВАЖНО: Это базовая функция для извлечения данных из токена.
 * Для полной валидации токена (включая проверку подписи) используйте
 * библиотеку jsonwebtoken или jose в production.
 * 
 * @param token - JWT токен
 * @returns Payload токена или null если токен невалиден
 */
export function parseJWT(token: string): JWTPayload | null {
  try {
    const parts = token.split('.')
    if (parts.length !== 3) {
      return null
    }

    // Декодируем payload (вторая часть токена)
    const payload = parts[1]
    const decoded = Buffer.from(payload, 'base64url').toString('utf-8')
    return JSON.parse(decoded) as JWTPayload
  } catch (error) {
    console.error('Error parsing JWT:', error)
    return null
  }
}

/**
 * Валидирует JWT токен
 * 
 * Проверяет:
 * - Формат токена
 * - Срок действия (exp)
 * - Наличие обязательных полей (userId)
 * 
 * @param token - JWT токен
 * @returns true если токен валиден, false в противном случае
 */
export function validateJWT(token: string): boolean {
  const payload = parseJWT(token)
  if (!payload) {
    return false
  }

  // Проверяем наличие userId
  if (!payload.userId) {
    return false
  }

  // Проверяем срок действия (если есть exp)
  if (payload.exp) {
    const now = Math.floor(Date.now() / 1000)
    if (payload.exp < now) {
      return false
    }
  }

  return true
}

/**
 * Извлекает информацию о пользователе из JWT токена в запросе
 * 
 * @param request - NextRequest объект
 * @returns Payload токена или null если токен отсутствует/невалиден
 */
export function getUserFromRequest(request: Request): JWTPayload | null {
  const authHeader = request.headers.get('authorization')
  const token = extractTokenFromHeader(authHeader)

  if (!token) {
    return null
  }

  if (!validateJWT(token)) {
    return null
  }

  return parseJWT(token)
}

/**
 * Проверяет, имеет ли пользователь указанную роль
 * 
 * @param payload - JWT payload
 * @param role - Роль для проверки
 * @returns true если пользователь имеет роль, false в противном случае
 */
export function hasRole(payload: JWTPayload | null, role: string): boolean {
  if (!payload || !payload.roles) {
    return false
  }
  return payload.roles.includes(role)
}

/**
 * Проверяет, является ли пользователь администратором
 * 
 * @param payload - JWT payload
 * @returns true если пользователь администратор, false в противном случае
 */
export function isAdmin(payload: JWTPayload | null): boolean {
  return hasRole(payload, 'admin')
}


