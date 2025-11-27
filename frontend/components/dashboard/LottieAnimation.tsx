'use client'

import { Player } from '@lottiefiles/react-lottie-player'
import { useEffect, useRef, useState } from 'react'
import { motion, AnimatePresence } from 'framer-motion'

interface LottieAnimationProps {
  src: string
  loop?: boolean
  autoplay?: boolean
  className?: string
  onComplete?: () => void
  fallback?: React.ReactNode
}

export function LottieAnimation({
  src,
  loop = false,
  autoplay = true,
  className,
  onComplete,
  fallback,
}: LottieAnimationProps) {
  const playerRef = useRef<Player>(null)
  const [error, setError] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (playerRef.current && autoplay && !error) {
      playerRef.current.play()
    }
  }, [autoplay, error])

  // Fallback для случаев, когда Lottie файл не загрузился
  if (error && fallback) {
    return <>{fallback}</>
  }

  // Если это URL, используем его напрямую, иначе предполагаем локальный файл
  const animationSrc = src.startsWith('http') ? src : src

  return (
    <AnimatePresence>
      {loading && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="absolute inset-0 flex items-center justify-center"
        >
          <div className="h-8 w-8 border-2 border-primary border-t-transparent rounded-full animate-spin" />
        </motion.div>
      )}
      <Player
        ref={playerRef}
        src={animationSrc}
        loop={loop}
        autoplay={autoplay}
        className={className}
        onEvent={(event) => {
          if (event === 'complete' && onComplete) {
            onComplete()
          }
          if (event === 'load') {
            setLoading(false)
          }
          if (event === 'error') {
            setError(true)
            setLoading(false)
          }
        }}
      />
    </AnimatePresence>
  )
}

