'use client'

import React from 'react'

interface StateDebuggerProps {
  clientId: string | number | null
  projectId: string | number | null
  componentName: string
  enabled?: boolean
}

/**
 * Компонент для отладки состояния компонентов
 * Показывает информацию о монтировании/размонтировании компонентов
 * и изменении clientId/projectId
 */
export const StateDebugger: React.FC<StateDebuggerProps> = ({
  clientId,
  projectId,
  componentName,
  enabled = process.env.NODE_ENV === 'development',
}) => {
  const [mountCount, setMountCount] = React.useState(0)
  const [lastUpdate, setLastUpdate] = React.useState<Date | null>(null)

  React.useEffect(() => {
    if (!enabled) return

    setMountCount(prev => prev + 1)
    setLastUpdate(new Date())
    
    const projectKey = clientId && projectId ? `${clientId}:${projectId}` : 'none'
    console.log(`[${componentName}] Mounted/Updated:`, {
      clientId,
      projectId,
      projectKey,
      mountCount: mountCount + 1,
      timestamp: new Date().toISOString(),
    })

    return () => {
      console.log(`[${componentName}] Unmounted:`, {
        clientId,
        projectId,
        projectKey,
        timestamp: new Date().toISOString(),
      })
    }
  }, [clientId, projectId, componentName, enabled])

  if (!enabled) {
    return null
  }

  return (
    <div
      style={{
        position: 'fixed',
        top: 10,
        right: 10,
        background: 'rgba(0, 0, 0, 0.85)',
        color: 'white',
        padding: '12px 16px',
        fontSize: '11px',
        fontFamily: 'monospace',
        zIndex: 10000,
        borderRadius: '6px',
        border: '1px solid rgba(255, 255, 255, 0.2)',
        maxWidth: '300px',
        lineHeight: '1.5',
      }}
    >
      <div style={{ fontWeight: 'bold', marginBottom: '8px', borderBottom: '1px solid rgba(255,255,255,0.2)', paddingBottom: '4px' }}>
        🔍 {componentName}
      </div>
      <div>Client: {clientId || '—'}</div>
      <div>Project: {projectId || '—'}</div>
      <div>Mounts: {mountCount}</div>
      {lastUpdate && (
        <div style={{ fontSize: '10px', opacity: 0.7, marginTop: '4px' }}>
          Updated: {lastUpdate.toLocaleTimeString()}
        </div>
      )}
    </div>
  )
}

