/**
 * Утилиты для работы с избранными ГОСТами
 */

const FAVORITES_KEY = 'gost_favorites'
const HISTORY_KEY = 'gost_view_history'
const MAX_HISTORY_ITEMS = 50

export interface FavoriteGost {
  id: number
  gost_number: string
  title: string
  addedAt: string
}

export interface HistoryGost {
  id: number
  gost_number: string
  title: string
  viewedAt: string
}

/**
 * Получить список избранных ГОСТов
 */
export function getFavorites(): FavoriteGost[] {
  if (typeof window === 'undefined') return []
  
  try {
    const stored = localStorage.getItem(FAVORITES_KEY)
    return stored ? JSON.parse(stored) : []
  } catch {
    return []
  }
}

/**
 * Проверить, находится ли ГОСТ в избранном
 */
export function isFavorite(gostId: number): boolean {
  const favorites = getFavorites()
  return favorites.some(fav => fav.id === gostId)
}

/**
 * Добавить ГОСТ в избранное
 */
export function addToFavorites(gost: { id: number; gost_number: string; title: string }): void {
  if (typeof window === 'undefined') return
  
  try {
    const favorites = getFavorites()
    if (!favorites.some(fav => fav.id === gost.id)) {
      favorites.push({
        id: gost.id,
        gost_number: gost.gost_number,
        title: gost.title,
        addedAt: new Date().toISOString(),
      })
      localStorage.setItem(FAVORITES_KEY, JSON.stringify(favorites))
    }
  } catch (err) {
    console.error('Failed to add to favorites:', err)
  }
}

/**
 * Удалить ГОСТ из избранного
 */
export function removeFromFavorites(gostId: number): void {
  if (typeof window === 'undefined') return
  
  try {
    const favorites = getFavorites()
    const filtered = favorites.filter(fav => fav.id !== gostId)
    localStorage.setItem(FAVORITES_KEY, JSON.stringify(filtered))
  } catch (err) {
    console.error('Failed to remove from favorites:', err)
  }
}

/**
 * Переключить статус избранного
 */
export function toggleFavorite(gost: { id: number; gost_number: string; title: string }): boolean {
  if (isFavorite(gost.id)) {
    removeFromFavorites(gost.id)
    return false
  } else {
    addToFavorites(gost)
    return true
  }
}

/**
 * Добавить ГОСТ в историю просмотров
 */
export function addToHistory(gost: { id: number; gost_number: string; title: string }): void {
  if (typeof window === 'undefined') return
  
  try {
    const history = getHistory()
    // Удаляем дубликаты
    const filtered = history.filter(item => item.id !== gost.id)
    // Добавляем в начало
    filtered.unshift({
      id: gost.id,
      gost_number: gost.gost_number,
      title: gost.title,
      viewedAt: new Date().toISOString(),
    })
    // Ограничиваем размер истории
    const limited = filtered.slice(0, MAX_HISTORY_ITEMS)
    localStorage.setItem(HISTORY_KEY, JSON.stringify(limited))
  } catch (err) {
    console.error('Failed to add to history:', err)
  }
}

/**
 * Получить историю просмотров
 */
export function getHistory(): HistoryGost[] {
  if (typeof window === 'undefined') return []
  
  try {
    const stored = localStorage.getItem(HISTORY_KEY)
    return stored ? JSON.parse(stored) : []
  } catch {
    return []
  }
}

/**
 * Очистить историю просмотров
 */
export function clearHistory(): void {
  if (typeof window === 'undefined') return
  
  try {
    localStorage.removeItem(HISTORY_KEY)
  } catch (err) {
    console.error('Failed to clear history:', err)
  }
}

