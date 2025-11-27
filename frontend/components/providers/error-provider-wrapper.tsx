'use client'

import { ErrorProvider } from '@/contexts/ErrorContext'
import { ErrorBoundary } from '@/components/ErrorBoundary'
import { GlobalErrorHandlers } from '@/components/GlobalErrorHandlers'

export function ErrorProviderWrapper({ children }: { children: React.ReactNode }) {
  return (
    <ErrorProvider>
      <ErrorBoundary>
        <GlobalErrorHandlers />
        {children}
      </ErrorBoundary>
    </ErrorProvider>
  )
}

