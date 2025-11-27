'use client'

import * as React from "react"
import { motion, AnimatePresence } from "framer-motion"
import { ChevronRight, Home } from "lucide-react"
import Link from "next/link"
import { cn } from "@/lib/utils"
import { useAnimationContext } from "@/providers/animation-provider"

export interface BreadcrumbItem {
  label: string
  href?: string
  icon?: React.ComponentType<{ className?: string }>
}

interface BreadcrumbProps extends Omit<React.HTMLAttributes<HTMLElement>, 'onAnimationStart' | 'onAnimationEnd' | 'onAnimationIteration' | 'onDragStart' | 'onDrag' | 'onDragEnd'> {
  items: BreadcrumbItem[]
  homeHref?: string
  homeLabel?: string
  showHome?: boolean
}

const Breadcrumb = React.forwardRef<HTMLElement, BreadcrumbProps>(
  ({ className, items, homeHref = "/", homeLabel = "Главная", showHome = true, ...props }, ref) => {
    const { getAnimationConfig } = useAnimationContext()
    const animationConfig = getAnimationConfig()
    const allItems = showHome
      ? [{ label: homeLabel, href: homeHref }, ...items]
      : items

    return (
      <motion.nav
        ref={ref}
        aria-label="Breadcrumb"
        className={cn("flex items-center space-x-1 text-sm", className)}
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ 
          duration: animationConfig.duration * 0.5, 
          ease: animationConfig.ease as [number, number, number, number]
        }}
        {...props}
      >
        <ol className="flex items-center space-x-1.5">
          <AnimatePresence mode="popLayout">
            {allItems.map((item, index) => {
            const isLast = index === allItems.length - 1
            return (
              <motion.li
                key={`${item.label}-${index}`}
                initial={{ opacity: 0, x: -10 }}
                animate={{ opacity: 1, x: 0 }}
                exit={{ opacity: 0, x: 10 }}
                transition={{ 
                  delay: index * 0.05, 
                  duration: animationConfig.duration * 0.5,
                  ease: animationConfig.ease as [number, number, number, number]
                }}
                className="flex items-center"
              >
                {index > 0 && (
                  <ChevronRight className="h-3.5 w-3.5 mx-1.5 text-muted-foreground/50" aria-hidden="true" />
                )}
                {item.href && !isLast ? (
                  <motion.div
                    whileHover={{ scale: 1.02 }}
                    whileTap={{ scale: 0.98 }}
                    transition={{ duration: animationConfig.duration * 0.3 }}
                  >
                    <Link
                      href={item.href}
                      className="hover:text-foreground transition-all flex items-center gap-1.5 group px-2 py-1 rounded-md hover:bg-accent/50"
                      aria-current={isLast ? "page" : undefined}
                    >
                    {item.icon && (
                      <item.icon className="h-4 w-4 text-muted-foreground group-hover:text-foreground transition-colors" />
                    )}
                      <span className="text-muted-foreground group-hover:text-foreground transition-colors">{item.label}</span>
                    </Link>
                  </motion.div>
                ) : (
                  <span
                    className={cn(
                      "font-semibold flex items-center gap-1.5 px-2 py-1 rounded-md bg-accent/30",
                      isLast && "text-foreground"
                    )}
                    aria-current={isLast ? "page" : undefined}
                  >
                    {item.icon && (
                      <item.icon className="h-4 w-4 text-primary" />
                    )}
                    <span>{item.label}</span>
                  </span>
                )}
              </motion.li>
            )
          })}
          </AnimatePresence>
        </ol>
      </motion.nav>
    )
  }
)
Breadcrumb.displayName = "Breadcrumb"

export { Breadcrumb }

