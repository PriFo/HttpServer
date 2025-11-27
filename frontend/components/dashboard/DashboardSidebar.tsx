'use client'

import { motion } from 'framer-motion'
import {
  LayoutDashboard,
  Activity,
  PlayCircle,
  CheckCircle2,
  Users,
  Menu,
  X
} from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { useDashboardStore, type TabType } from '@/stores/dashboard-store'
import { useState } from 'react'

interface TabItem {
  id: TabType
  label: string
  icon: typeof LayoutDashboard
}

const tabs: TabItem[] = [
  { id: 'overview', label: 'Обзор', icon: LayoutDashboard },
  { id: 'monitoring', label: 'Мониторинг', icon: Activity },
  { id: 'processes', label: 'Процессы', icon: PlayCircle },
  { id: 'quality', label: 'Качество', icon: CheckCircle2 },
  { id: 'clients', label: 'Клиенты', icon: Users },
]

export function DashboardSidebar() {
  const { activeTab, setActiveTab } = useDashboardStore()
  const [isMobileOpen, setIsMobileOpen] = useState(false)

  const handleTabChange = (tab: TabType) => {
    setActiveTab(tab)
    setIsMobileOpen(false)
  }

  return (
    <>
      {/* Mobile menu button */}
      <div className="lg:hidden fixed top-16 left-0 right-0 z-40 border-b bg-background px-4 py-2">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => setIsMobileOpen(!isMobileOpen)}
          className="w-full justify-start"
        >
          <Menu className="h-4 w-4 mr-2" />
          {tabs.find(t => t.id === activeTab)?.label || 'Меню'}
        </Button>
      </div>

      {/* Mobile sidebar overlay */}
      {isMobileOpen && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="lg:hidden fixed inset-0 z-50 bg-background/80 backdrop-blur-sm"
          onClick={() => setIsMobileOpen(false)}
        >
          <motion.div
            initial={{ x: -300 }}
            animate={{ x: 0 }}
            exit={{ x: -300 }}
            transition={{ type: 'spring', damping: 25, stiffness: 200 }}
            className="h-full w-64 border-r bg-background shadow-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-center justify-between p-4 border-b">
              <h2 className="font-semibold">Навигация</h2>
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setIsMobileOpen(false)}
              >
                <X className="h-4 w-4" />
              </Button>
            </div>
            <nav className="p-2">
              {tabs.map((tab) => {
                const Icon = tab.icon
                const isActive = activeTab === tab.id
                return (
                  <motion.button
                    key={tab.id}
                    onClick={() => handleTabChange(tab.id)}
                    className={cn(
                      "w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                      isActive
                        ? "bg-primary text-primary-foreground"
                        : "text-muted-foreground hover:bg-muted hover:text-foreground"
                    )}
                    whileHover={{ x: 4 }}
                    whileTap={{ scale: 0.98 }}
                  >
                    <Icon className="h-5 w-5" />
                    {tab.label}
                  </motion.button>
                )
              })}
            </nav>
          </motion.div>
        </motion.div>
      )}

      {/* Desktop sidebar */}
      <aside className="hidden lg:block fixed left-0 top-16 h-[calc(100vh-4rem)] w-64 border-r bg-background overflow-y-auto">
        <nav className="p-4 space-y-1">
            <TooltipProvider>
              {tabs.map((tab, index) => {
                const Icon = tab.icon
                const isActive = activeTab === tab.id
                return (
                  <Tooltip key={tab.id}>
                    <TooltipTrigger asChild>
                      <motion.button
                        onClick={() => handleTabChange(tab.id)}
                        className={cn(
                          "w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-sm font-medium transition-all relative",
                          isActive
                            ? "bg-primary text-primary-foreground shadow-md"
                            : "text-muted-foreground hover:bg-muted hover:text-foreground"
                        )}
                        initial={{ opacity: 0, x: -20 }}
                        animate={{ opacity: 1, x: 0 }}
                        transition={{ delay: index * 0.05 }}
                        whileHover={{ x: 4, scale: 1.02 }}
                        whileTap={{ scale: 0.98 }}
                      >
                        {isActive && (
                          <motion.div
                            layoutId="activeTab"
                            className="absolute inset-0 bg-primary rounded-lg shadow-md"
                            initial={false}
                            transition={{ type: 'spring', stiffness: 500, damping: 30 }}
                          />
                        )}
                        <motion.div
                          className="relative z-10"
                          animate={isActive ? { rotate: [0, -10, 10, 0] } : {}}
                          transition={{ duration: 0.5 }}
                        >
                          <Icon className={cn("h-5 w-5", isActive && "text-primary-foreground")} />
                        </motion.div>
                        <span className="relative z-10 font-medium">{tab.label}</span>
                        <motion.span 
                          className="ml-auto text-xs text-muted-foreground/50 relative z-10"
                          animate={isActive ? { scale: [1, 1.2, 1] } : {}}
                          transition={{ duration: 0.3 }}
                        >
                          {index + 1}
                        </motion.span>
                      </motion.button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>{tab.label} (Нажмите {index + 1})</p>
                    </TooltipContent>
                  </Tooltip>
                )
              })}
            </TooltipProvider>
        </nav>
      </aside>
    </>
  )
}

