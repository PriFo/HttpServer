/**
 * Утилиты для работы с аутентификацией
 * 
 * В данный момент система аутентификации не реализована.
 * Эта утилита предоставляет заглушку, которую можно легко расширить
 * при добавлении системы аутентификации (NextAuth, Clerk, и т.д.)
 */

/**
 * Получает текущего пользователя
 * 
 * @returns Имя текущего пользователя или 'System' если пользователь не аутентифицирован
 * 
 * @example
 * // Текущая реализация (без аутентификации)
 * const user = getCurrentUser() // 'System'
 * 
 * // Будущая реализация (с NextAuth)
 * // const session = await getServerSession()
 * // return session?.user?.name || 'System'
 * 
 * // Будущая реализация (с Clerk)
 * // const { userId } = auth()
 * // const user = await clerkClient.users.getUser(userId)
 * // return user.fullName || 'System'
 */
export function getCurrentUser(): string {
  // Текущая реализация использует sessionStorage для хранения пользователя
  // В будущем можно расширить для интеграции с NextAuth, Clerk и т.д.
  // Варианты расширения:
  // 1. NextAuth: const session = await getServerSession(); return session?.user?.name
  // 2. Clerk: const { userId } = auth(); const user = await clerkClient.users.getUser(userId)
  // 3. Custom: const user = await fetch('/api/user/current'); return user.name
  
  if (typeof window !== 'undefined') {
    const storedUser = sessionStorage.getItem('currentUser')
    if (storedUser) {
      return storedUser
    }
  }
  
  return 'System'
}

/**
 * Получает ID текущего пользователя
 * 
 * @returns ID пользователя или null если пользователь не аутентифицирован
 */
export function getCurrentUserId(): string | null {
  // Текущая реализация использует sessionStorage для хранения ID пользователя
  // В будущем можно расширить для интеграции с системами аутентификации
  if (typeof window !== 'undefined') {
    const storedUserId = sessionStorage.getItem('currentUserId')
    if (storedUserId) {
      return storedUserId
    }
  }
  
  return null
}

/**
 * Проверяет, аутентифицирован ли пользователь
 * 
 * @returns true если пользователь аутентифицирован, false в противном случае
 */
export function isAuthenticated(): boolean {
  // Текущая реализация проверяет наличие пользователя в sessionStorage
  // В будущем можно расширить для интеграции с системами аутентификации
  if (typeof window !== 'undefined') {
    return sessionStorage.getItem('currentUser') !== null
  }
  
  return false
}

