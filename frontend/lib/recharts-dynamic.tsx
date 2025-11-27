'use client'

import dynamic from 'next/dynamic'
import { Skeleton } from '@/components/ui/skeleton'

// Динамические импорты компонентов Recharts для уменьшения размера бандла
export const DynamicBarChart = dynamic(
  () => import('recharts').then((mod) => mod.BarChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
)

export const DynamicBar = dynamic(
  () => import('recharts').then((mod) => mod.Bar),
  { ssr: false }
)

export const DynamicLineChart = dynamic(
  () => import('recharts').then((mod) => mod.LineChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
)

export const DynamicLine = dynamic(
  () => import('recharts').then((mod) => mod.Line),
  { ssr: false }
)

export const DynamicPieChart = dynamic(
  () => import('recharts').then((mod) => mod.PieChart),
  { ssr: false, loading: () => <Skeleton className="h-[150px] w-full" /> }
)

export const DynamicPie = dynamic(
  () => import('recharts').then((mod) => mod.Pie),
  { ssr: false }
)

export const DynamicAreaChart = dynamic(
  () => import('recharts').then((mod) => mod.AreaChart),
  { ssr: false, loading: () => <Skeleton className="h-[300px] w-full" /> }
)

export const DynamicArea = dynamic(
  () => import('recharts').then((mod) => mod.Area),
  { ssr: false }
)

export const DynamicCell = dynamic(
  () => import('recharts').then((mod) => mod.Cell),
  { ssr: false }
)

// Экспортируем Cell напрямую для использования в map
export { Cell } from 'recharts'

// Легкие компоненты можно импортировать напрямую
export {
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts'

