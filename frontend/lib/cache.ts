// Simple client-side cache with expiration

interface CacheEntry<T> {
  data: T
  timestamp: number
  expiresIn: number // milliseconds
}

const CACHE_PREFIX = 'app_cache_'

export class ClientCache {
  /**
   * Set item in cache with expiration time
   * @param key Cache key
   * @param data Data to cache
   * @param expiresIn Expiration time in milliseconds (default: 5 minutes)
   */
  static set<T>(key: string, data: T, expiresIn: number = 5 * 60 * 1000): void {
    if (typeof window === 'undefined') return

    const entry: CacheEntry<T> = {
      data,
      timestamp: Date.now(),
      expiresIn,
    }

    try {
      localStorage.setItem(`${CACHE_PREFIX}${key}`, JSON.stringify(entry))
    } catch (error) {
      console.warn('Failed to set cache:', error)
    }
  }

  /**
   * Get item from cache if not expired
   * @param key Cache key
   * @returns Cached data or null if expired/not found
   */
  static get<T>(key: string): T | null {
    if (typeof window === 'undefined') return null

    try {
      const item = localStorage.getItem(`${CACHE_PREFIX}${key}`)
      if (!item) return null

      const entry: CacheEntry<T> = JSON.parse(item)
      const isExpired = Date.now() - entry.timestamp > entry.expiresIn

      if (isExpired) {
        this.remove(key)
        return null
      }

      return entry.data
    } catch (error) {
      console.warn('Failed to get cache:', error)
      return null
    }
  }

  /**
   * Remove item from cache
   * @param key Cache key
   */
  static remove(key: string): void {
    if (typeof window === 'undefined') return

    try {
      localStorage.removeItem(`${CACHE_PREFIX}${key}`)
    } catch (error) {
      console.warn('Failed to remove cache:', error)
    }
  }

  /**
   * Clear all app cache
   */
  static clear(): void {
    if (typeof window === 'undefined') return

    try {
      const keys = Object.keys(localStorage)
      keys.forEach(key => {
        if (key.startsWith(CACHE_PREFIX)) {
          localStorage.removeItem(key)
        }
      })
    } catch (error) {
      console.warn('Failed to clear cache:', error)
    }
  }

  /**
   * Check if cache entry exists and is valid
   * @param key Cache key
   * @returns true if cache is valid, false otherwise
   */
  static has(key: string): boolean {
    return this.get(key) !== null
  }
}
