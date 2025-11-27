'use client'

import { useState, useMemo } from 'react'
import { motion } from 'framer-motion'
import { 
  Home, 
  Search, 
  Bell, 
  Wifi, 
  WifiOff,
  X,
  Activity,
  Users,
  CheckCircle2,
  BarChart3,
  Database,
  FileText,
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Badge } from '@/components/ui/badge'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { formatDistanceToNow } from 'date-fns'
import { ru } from 'date-fns/locale'
import { ScrollArea } from '@/components/ui/scroll-area'
import { useDashboardStore } from '@/stores/dashboard-store'
import { SearchResults } from './SearchResults'
import { cn } from '@/lib/utils'

const searchPages = [
  { type: 'page' as const, title: 'Обзор', description: 'Главная панель управления', href: '/', icon: Home },
  { type: 'page' as const, title: 'Мониторинг', description: 'Мониторинг системы в реальном времени', href: '/monitoring', icon: Activity },
  { type: 'page' as const, title: 'Клиенты', description: 'Управление клиентами и проектами', href: '/clients', icon: Users },
  { type: 'page' as const, title: 'Качество данных', description: 'Анализ качества нормализации', href: '/quality', icon: CheckCircle2 },
  { type: 'page' as const, title: 'Результаты', description: 'Просмотр результатов нормализации', href: '/results', icon: BarChart3 },
  { type: 'page' as const, title: 'Базы данных', description: 'Управление базами данных', href: '/databases', icon: Database },
  { type: 'page' as const, title: 'Отчеты', description: 'Генерация отчетов', href: '/reports', icon: FileText },
]

export function DashboardHeader() {
  const { notifications, markNotificationAsRead, removeNotification, clearNotifications } = useDashboardStore()
  const [searchQuery, setSearchQuery] = useState('')
  const [isSearchOpen, setIsSearchOpen] = useState(false)
  const unreadCount = notifications.filter(n => !n.read).length

  const searchResults = useMemo(() => {
    if (!searchQuery.trim()) return []
    const query = searchQuery.toLowerCase()
    return searchPages.filter(page => 
      page.title.toLowerCase().includes(query) ||
      page.description.toLowerCase().includes(query)
    )
  }, [searchQuery])

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    // Поиск реализован через SearchResults компонент
    // При клике на результат происходит навигация
    if (searchResults.length > 0) {
      // Можно добавить логику для автоматической навигации к первому результату
      // или оставить выбор пользователю через SearchResults
    }
  }

  return (
    <motion.header
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60"
    >
      <div className="container flex h-16 items-center justify-between px-4">
        {/* Logo and Title */}
        <div className="flex items-center gap-3">
          <motion.div
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            className="flex items-center gap-2 cursor-pointer"
          >
            <motion.div 
              className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary text-primary-foreground shadow-lg"
              whileHover={{ rotate: 360 }}
              transition={{ duration: 0.6 }}
            >
              <Home className="h-5 w-5" />
            </motion.div>
            <div className="hidden sm:block">
              <motion.h1 
                className="text-lg font-bold"
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.1 }}
              >
                Нормализатор
              </motion.h1>
              <motion.p 
                className="text-xs text-muted-foreground"
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: 0.2 }}
              >
                Миссионный центр системы
              </motion.p>
            </div>
          </motion.div>
        </div>

        {/* Search Bar */}
        <div className="hidden md:flex flex-1 max-w-md mx-4 relative">
          <form onSubmit={handleSearch} className="w-full">
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="search"
                placeholder="Поиск по системе..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onFocus={() => setIsSearchOpen(true)}
                className="pl-9"
              />
              {searchQuery && (
                <button
                  type="button"
                  onClick={() => {
                    setSearchQuery('')
                    setIsSearchOpen(false)
                  }}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground"
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </form>
          {isSearchOpen && searchResults.length > 0 && (
            <SearchResults
              query={searchQuery}
              results={searchResults}
              onClose={() => setIsSearchOpen(false)}
            />
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2">
          {/* Real-time indicator */}
          <RealTimeIndicator />

          {/* Notifications */}
          <Popover>
            <PopoverTrigger asChild>
              <Button variant="ghost" size="icon" className="relative">
                <Bell className="h-5 w-5" />
                {unreadCount > 0 && (
                  <motion.div
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    className="absolute -top-1 -right-1 flex h-5 w-5 items-center justify-center rounded-full bg-destructive text-destructive-foreground text-xs font-bold"
                  >
                    {unreadCount > 9 ? '9+' : unreadCount}
                  </motion.div>
                )}
              </Button>
            </PopoverTrigger>
            <PopoverContent className="w-80 p-0" align="end">
              <div className="flex items-center justify-between p-4 border-b">
                <h3 className="font-semibold">Уведомления</h3>
                {notifications.length > 0 && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={clearNotifications}
                    className="h-8 text-xs"
                  >
                    Очистить все
                  </Button>
                )}
              </div>
              <ScrollArea className="h-[400px]">
                {notifications.length === 0 ? (
                  <div className="flex flex-col items-center justify-center p-8 text-center text-muted-foreground">
                    <Bell className="h-12 w-12 mb-2 opacity-50" />
                    <p>Нет уведомлений</p>
                  </div>
                ) : (
                  <div className="p-2">
                    {notifications.map((notification) => (
                      <motion.div
                        key={notification.id}
                        initial={{ opacity: 0, x: -10 }}
                        animate={{ opacity: 1, x: 0 }}
                        className={cn(
                          "flex items-start gap-3 p-3 rounded-lg cursor-pointer transition-colors",
                          !notification.read && "bg-muted"
                        )}
                        onClick={() => !notification.read && markNotificationAsRead(notification.id)}
                      >
                        <div className={cn(
                          "flex-1 min-w-0",
                          notification.read && "opacity-60"
                        )}>
                          <div className="flex items-center gap-2 mb-1">
                            <Badge variant={
                              notification.type === 'error' ? 'destructive' :
                              notification.type === 'success' ? 'default' :
                              notification.type === 'warning' ? 'secondary' : 'outline'
                            } className="text-xs">
                              {notification.type}
                            </Badge>
                            <span className="text-xs text-muted-foreground">
                              {formatDistanceToNow(notification.timestamp, { addSuffix: true, locale: ru })}
                            </span>
                          </div>
                          <p className="font-medium text-sm">{notification.title}</p>
                          <p className="text-sm text-muted-foreground">{notification.message}</p>
                        </div>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="h-6 w-6 flex-shrink-0"
                          onClick={(e) => {
                            e.stopPropagation()
                            removeNotification(notification.id)
                          }}
                        >
                          <X className="h-3 w-3" />
                        </Button>
                      </motion.div>
                    ))}
                  </div>
                )}
              </ScrollArea>
            </PopoverContent>
          </Popover>

          {/* Mobile search button */}
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden"
            onClick={() => setIsSearchOpen(!isSearchOpen)}
          >
            <Search className="h-5 w-5" />
          </Button>
        </div>
      </div>

      {/* Mobile search bar */}
      {isSearchOpen && (
        <motion.div
          initial={{ height: 0, opacity: 0 }}
          animate={{ height: 'auto', opacity: 1 }}
          exit={{ height: 0, opacity: 0 }}
          className="md:hidden border-t p-4"
        >
          <form onSubmit={handleSearch}>
            <div className="relative">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                type="search"
                placeholder="Поиск по системе..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
          </form>
        </motion.div>
      )}
    </motion.header>
  )
}

function RealTimeIndicator() {
  const { isRealTimeEnabled, toggleRealTime } = useDashboardStore()

  return (
    <motion.button
      onClick={toggleRealTime}
      className="flex items-center gap-2 px-2 py-1 rounded-md bg-muted hover:bg-muted/80 transition-colors"
      whileHover={{ scale: 1.05 }}
      whileTap={{ scale: 0.95 }}
      title={isRealTimeEnabled ? 'Отключить реальное время' : 'Включить реальное время'}
    >
      {isRealTimeEnabled ? (
        <>
          <motion.div
            animate={{ scale: [1, 1.2, 1] }}
            transition={{ duration: 2, repeat: Infinity }}
          >
            <Wifi className="h-4 w-4 text-green-600" />
          </motion.div>
          <span className="hidden sm:inline text-xs text-muted-foreground">Онлайн</span>
        </>
      ) : (
        <>
          <WifiOff className="h-4 w-4 text-muted-foreground" />
          <span className="hidden sm:inline text-xs text-muted-foreground">Офлайн</span>
        </>
      )}
    </motion.button>
  )
}

