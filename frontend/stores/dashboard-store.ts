import { create } from 'zustand'
import type { ProviderMetrics, SystemStats } from '@/types/monitoring'

export type TabType = 'overview' | 'monitoring' | 'processes' | 'quality' | 'clients'

export interface Notification {
  id: string
  type: 'info' | 'success' | 'warning' | 'error'
  title: string
  message: string
  timestamp: Date
  read: boolean
}

export interface DashboardSystemStats {
  totalRecords: number
  totalDatabases: number
  processedRecords: number
  createdGroups: number
  mergedRecords: number
  systemVersion: string
  currentDatabase: {
    name: string
    path: string
    status: 'connected' | 'disconnected' | 'unknown'
    lastUpdate: string
  } | null
  normalizationStatus: {
    status: 'idle' | 'running' | 'completed' | 'error'
    progress: number
    currentStage: string
    startTime: string | null
    endTime: string | null
  }
  qualityMetrics: {
    overallQuality: number
    highConfidence: number
    mediumConfidence: number
    lowConfidence: number
    totalRecords?: number
  }
}

export interface BackendFallbackState {
  isActive: boolean
  reasons: string[]
  timestamp: string
}

interface DashboardState {
  // Tab management
  activeTab: TabType
  setActiveTab: (tab: TabType) => void

  // Real-time monitoring
  isRealTimeEnabled: boolean
  toggleRealTime: () => void
  setRealTimeEnabled: (enabled: boolean) => void

  // Provider metrics from SSE
  providerMetrics: ProviderMetrics[]
  setProviderMetrics: (metrics: ProviderMetrics[]) => void

  // System statistics
  systemStats: DashboardSystemStats | null
  setSystemStats: (stats: Partial<DashboardSystemStats>) => void

  // Monitoring system stats
  monitoringSystemStats: SystemStats | null
  setMonitoringSystemStats: (stats: SystemStats | null) => void

  // Notifications
  notifications: Notification[]
  addNotification: (notification: Omit<Notification, 'id' | 'timestamp' | 'read'>) => void
  markNotificationAsRead: (id: string) => void
  removeNotification: (id: string) => void
  clearNotifications: () => void

  // Loading states
  isLoading: boolean
  setLoading: (loading: boolean) => void

  // Error state
  error: string | null
  setError: (error: string | null) => void

  // Backend fallback state
  backendFallback: BackendFallbackState | null
  setBackendFallback: (fallback: BackendFallbackState | null) => void
}

export const useDashboardStore = create<DashboardState>((set, get) => ({
  // Tab management
  activeTab: 'overview',
  setActiveTab: (tab) => set({ activeTab: tab }),

  // Real-time monitoring
  isRealTimeEnabled: true,
  toggleRealTime: () => set((state) => ({ isRealTimeEnabled: !state.isRealTimeEnabled })),
  setRealTimeEnabled: (enabled) => set({ isRealTimeEnabled: enabled }),

  // Provider metrics
  providerMetrics: [],
  setProviderMetrics: (metrics) => set({ providerMetrics: metrics }),

  // System statistics
  systemStats: null,
  setSystemStats: (stats) =>
    set((state) => ({
      systemStats: state.systemStats ? { ...state.systemStats, ...stats } : (stats as DashboardSystemStats),
    })),

  // Monitoring system stats
  monitoringSystemStats: null,
  setMonitoringSystemStats: (stats) => set({ monitoringSystemStats: stats }),

  // Notifications
  notifications: [],
  addNotification: (notification) => {
    const newNotification: Notification = {
      ...notification,
      id: `notif-${Date.now()}-${Math.random()}`,
      timestamp: new Date(),
      read: false,
    }
    set((state) => ({
      notifications: [newNotification, ...state.notifications].slice(0, 50), // Keep last 50
    }))
  },
  markNotificationAsRead: (id) =>
    set((state) => ({
      notifications: state.notifications.map((n) => (n.id === id ? { ...n, read: true } : n)),
    })),
  removeNotification: (id) =>
    set((state) => ({
      notifications: state.notifications.filter((n) => n.id !== id),
    })),
  clearNotifications: () => set({ notifications: [] }),

  // Loading states
  isLoading: false,
  setLoading: (loading) => set({ isLoading: loading }),

  // Error state
  error: null,
  setError: (error) => set({ error }),
  
  // Backend fallback
  backendFallback: null,
  setBackendFallback: (fallback) => set({ backendFallback: fallback }),
}))

