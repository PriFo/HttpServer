/**
 * Утилиты для кэширования данных
 */

interface CacheEntry<T> {
  data: T
  timestamp: number
  expiresAt: number
}

class MemoryCache {
  private cache = new Map<string, CacheEntry<any>>()

  /**
   * Сохраняет данные в кэш
   */
  set<T>(key: string, data: T, ttl: number = 5 * 60 * 1000): void {
    const now = Date.now()
    this.cache.set(key, {
      data,
      timestamp: now,
      expiresAt: now + ttl,
    })
  }

  /**
   * Получает данные из кэша
   */
  get<T>(key: string): T | null {
    const entry = this.cache.get(key)
    if (!entry) {
      return null
    }

    // Проверяем срок действия
    if (Date.now() > entry.expiresAt) {
      this.cache.delete(key)
      return null
    }

    return entry.data as T
  }

  /**
   * Проверяет наличие данных в кэше
   */
  has(key: string): boolean {
    const entry = this.cache.get(key)
    if (!entry) {
      return false
    }

    if (Date.now() > entry.expiresAt) {
      this.cache.delete(key)
      return false
    }

    return true
  }

  /**
   * Удаляет данные из кэша
   */
  delete(key: string): void {
    this.cache.delete(key)
  }

  /**
   * Очищает весь кэш
   */
  clear(): void {
    this.cache.clear()
  }

  /**
   * Очищает истекшие записи
   */
  cleanup(): void {
    const now = Date.now()
    for (const [key, entry] of this.cache.entries()) {
      if (now > entry.expiresAt) {
        this.cache.delete(key)
      }
    }
  }

  /**
   * Получает размер кэша
   */
  size(): number {
    return this.cache.size
  }
}

// Глобальный экземпляр кэша
const cache = new MemoryCache()

// Периодическая очистка истекших записей (каждые 5 минут)
if (typeof window !== 'undefined') {
  setInterval(() => {
    cache.cleanup()
  }, 5 * 60 * 1000)
}

/**
 * Кэширует результат функции
 */
export async function cachedFetch<T>(
  key: string,
  fetcher: () => Promise<T>,
  ttl: number = 5 * 60 * 1000
): Promise<T> {
  // Проверяем кэш
  const cached = cache.get<T>(key)
  if (cached !== null) {
    return cached
  }

  // Выполняем запрос
  const data = await fetcher()
  
  // Сохраняем в кэш
  cache.set(key, data, ttl)
  
  return data
}

/**
 * Создает ключ кэша из параметров
 */
export function createCacheKey(prefix: string, params: Record<string, any>): string {
  const sortedParams = Object.keys(params)
    .sort()
    .map(key => `${key}=${JSON.stringify(params[key])}`)
    .join('&')
  return `${prefix}:${sortedParams}`
}

export { cache, MemoryCache }
