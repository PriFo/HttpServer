'use client'

import { motion, AnimatePresence } from 'framer-motion'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Search, FileText, Database, Users, Activity } from 'lucide-react'
import Link from 'next/link'
import { cn } from '@/lib/utils'

interface SearchResult {
  type: 'page' | 'client' | 'database' | 'process'
  title: string
  description: string
  href: string
  icon: typeof Search
}

interface SearchResultsProps {
  query: string
  results: SearchResult[]
  onClose: () => void
}

const typeColors = {
  page: 'bg-blue-100 text-blue-700 dark:bg-blue-900 dark:text-blue-300',
  client: 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300',
  database: 'bg-purple-100 text-purple-700 dark:bg-purple-900 dark:text-purple-300',
  process: 'bg-orange-100 text-orange-700 dark:bg-orange-900 dark:text-orange-300',
}

export function SearchResults({ query, results, onClose }: SearchResultsProps) {
  if (!query.trim() || results.length === 0) {
    return null
  }

  return (
    <AnimatePresence>
      <motion.div
        initial={{ opacity: 0, y: -10 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -10 }}
        className="absolute top-full left-0 right-0 mt-2 z-50"
      >
        <Card className="shadow-lg border-2">
          <CardContent className="p-2 max-h-[400px] overflow-y-auto">
            <div className="space-y-1">
              {results.map((result, index) => {
                const Icon = result.icon
                return (
                  <motion.div
                    key={result.href}
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.05 }}
                  >
                    <Link
                      href={result.href}
                      onClick={onClose}
                      className={cn(
                        "flex items-center gap-3 p-3 rounded-lg transition-colors",
                        "hover:bg-muted cursor-pointer"
                      )}
                    >
                      <div className={cn(
                        "p-2 rounded-lg",
                        typeColors[result.type]
                      )}>
                        <Icon className="h-4 w-4" />
                      </div>
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm">{result.title}</div>
                        <div className="text-xs text-muted-foreground truncate">
                          {result.description}
                        </div>
                      </div>
                      <Badge variant="outline" className="text-xs">
                        {result.type}
                      </Badge>
                    </Link>
                  </motion.div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </AnimatePresence>
  )
}

