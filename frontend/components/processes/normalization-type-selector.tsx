'use client'

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Package, Building2, Layers, CheckCircle2 } from 'lucide-react'
import { motion } from 'framer-motion'
import { NormalizationType } from '@/types/normalization'
import { cn } from '@/lib/utils'

interface NormalizationTypeSelectorProps {
  value: NormalizationType
  onChange: (type: NormalizationType) => void
  nomenclatureCount?: number
  counterpartyCount?: number
  totalRecords?: number
  className?: string
}

export function NormalizationTypeSelector({
  value,
  onChange,
  nomenclatureCount = 0,
  counterpartyCount = 0,
  totalRecords = 0,
  className,
}: NormalizationTypeSelectorProps) {
  const bothCount = nomenclatureCount + counterpartyCount

  return (
    <Card className={cn('bg-gradient-to-br from-primary/5 to-primary/10 border-primary/20', className)}>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Layers className="h-5 w-5 text-primary" />
          Выбор типа нормализации
        </CardTitle>
        <CardDescription>
          Выберите тип данных для нормализации: номенклатура, контрагенты или комплексная обработка
        </CardDescription>
      </CardHeader>
      <CardContent>
        <Tabs value={value} onValueChange={(v) => onChange(v as NormalizationType)} className="w-full">
          <TabsList className="grid w-full grid-cols-3">
            <TabsTrigger value="nomenclature" className="flex flex-col items-center gap-2 py-3">
              <Package className="h-5 w-5" />
              <span className="text-sm font-medium">Номенклатура</span>
              {nomenclatureCount > 0 && (
                <Badge variant="secondary" className="text-xs">
                  {nomenclatureCount.toLocaleString()}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="counterparties" className="flex flex-col items-center gap-2 py-3">
              <Building2 className="h-5 w-5" />
              <span className="text-sm font-medium">Контрагенты</span>
              {counterpartyCount > 0 && (
                <Badge variant="secondary" className="text-xs">
                  {counterpartyCount.toLocaleString()}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="both" className="flex flex-col items-center gap-2 py-3">
              <Layers className="h-5 w-5" />
              <span className="text-sm font-medium">Комплексная</span>
              {bothCount > 0 && (
                <Badge variant="secondary" className="text-xs">
                  {bothCount.toLocaleString()}
                </Badge>
              )}
            </TabsTrigger>
          </TabsList>

          <TabsContent value="nomenclature" className="mt-4">
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-3"
            >
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                <span>Нормализация товаров и услуг</span>
              </div>
              <div className="text-sm">
                <p className="font-medium mb-1">Будет выполнено:</p>
                <ul className="list-disc list-inside space-y-1 text-muted-foreground">
                  <li>Нормализация названий товаров</li>
                  <li>Классификация по КПВЭД/ОКПД2</li>
                  <li>Стандартизация единиц измерения</li>
                  <li>Объединение дубликатов</li>
                </ul>
              </div>
            </motion.div>
          </TabsContent>

          <TabsContent value="counterparties" className="mt-4">
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-3"
            >
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                <span>Нормализация контрагентов</span>
              </div>
              <div className="text-sm">
                <p className="font-medium mb-1">Будет выполнено:</p>
                <ul className="list-disc list-inside space-y-1 text-muted-foreground">
                  <li>Верификация реквизитов (ИНН/БИН)</li>
                  <li>Стандартизация юридических форм</li>
                  <li>Нормализация адресов</li>
                  <li>Объединение дубликатов по реквизитам</li>
                </ul>
              </div>
            </motion.div>
          </TabsContent>

          <TabsContent value="both" className="mt-4">
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-3"
            >
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <CheckCircle2 className="h-4 w-4 text-green-600" />
                <span>Комплексная обработка всех данных</span>
              </div>
              <div className="text-sm">
                <p className="font-medium mb-1">Будет выполнено:</p>
                <ul className="list-disc list-inside space-y-1 text-muted-foreground">
                  <li>Нормализация номенклатуры и контрагентов</li>
                  <li>Все функции для обоих типов данных</li>
                  <li>Параллельная обработка</li>
                  <li>Объединение дубликатов</li>
                </ul>
              </div>
            </motion.div>
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

