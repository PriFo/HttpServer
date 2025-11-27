'use client'

import React, { createContext, useContext, PropsWithChildren, useMemo } from 'react'

interface NormalizationContextValue {
  clientId: string
  projectId: string
  projectType?: string | null
  normalizationStatus?: any
  isProcessRunning: boolean
  setIsProcessRunning: (value: boolean) => void
  refetchStatus?: () => void
}

const NormalizationContext = createContext<NormalizationContextValue | null>(null)

interface NormalizationProviderProps extends PropsWithChildren {
  value: NormalizationContextValue
}

export const NormalizationProvider: React.FC<NormalizationProviderProps> = ({ value, children }) => {
  const memoizedValue = useMemo(
    () => ({
      ...value,
    }),
    [value]
  )

  return (
    <NormalizationContext.Provider value={memoizedValue}>
      {children}
    </NormalizationContext.Provider>
  )
}

export const useNormalizationContext = () => {
  return useContext(NormalizationContext)
}

export const useNormalizationIdentifiers = (
  clientId?: string | null,
  projectId?: string | null
) => {
  const context = useNormalizationContext()

  return {
    clientId: clientId ?? context?.clientId ?? null,
    projectId: projectId ?? context?.projectId ?? null,
    projectType: context?.projectType,
    isProcessRunning: context?.isProcessRunning ?? false,
    normalizationStatus: context?.normalizationStatus,
    refetchStatus: context?.refetchStatus,
  }
}

