'use client'

import { useState, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Package,
  Building2,
  CheckCircle2,
  AlertCircle,
  XCircle,
  Search,
  Download,
  ChevronLeft,
  ChevronRight,
  FileText,
  Hash,
  Ruler,
  DollarSign,
  MapPin,
  Phone,
  Mail,
  Eye,
  EyeOff,
} from 'lucide-react'
import { NormalizationType } from '@/types/normalization'
import { cn } from '@/lib/utils'
import { motion, AnimatePresence } from 'framer-motion'
import { Skeleton } from '@/components/ui/skeleton'

interface PreviewRecord {
  id: string
  name: string
  fields: {
    [key: string]: {
      value: string | number | null
      filled: boolean
      quality?: 'good' | 'warning' | 'error'
    }
  }
  type?: 'nomenclature' | 'counterparty'
}

interface DataStructurePreviewProps {
  normalizationType: NormalizationType
  records?: PreviewRecord[]
  isLoading?: boolean
  onExport?: (format: 'json' | 'csv') => void
  className?: string
}

// Моковые данные для демонстрации
const mockNomenclatureRecords: PreviewRecord[] = [
  {
    id: '1',
    name: 'Панель ISOWALL BOX',
    type: 'nomenclature',
    fields: {
      name: { value: 'Панель ISOWALL BOX', filled: true, quality: 'good' },
      article: { value: 'IW-BOX-001', filled: true, quality: 'good' },
      unit: { value: 'шт', filled: true, quality: 'good' },
      description: { value: null, filled: false, quality: 'error' },
      price: { value: 1250, filled: true, quality: 'good' },
      category: { value: 'Строительные материалы', filled: true, quality: 'warning' },
    },
  },
  {
    id: '2',
    name: 'Болт М10х50',
    type: 'nomenclature',
    fields: {
      name: { value: 'Болт М10х50', filled: true, quality: 'good' },
      article: { value: 'BOLT-M10-50', filled: true, quality: 'good' },
      unit: { value: 'шт', filled: true, quality: 'good' },
      description: { value: 'Болт оцинкованный', filled: true, quality: 'good' },
      price: { value: 15, filled: true, quality: 'good' },
      category: { value: 'Крепеж', filled: true, quality: 'good' },
    },
  },
  {
    id: '3',
    name: 'Краска акриловая белая',
    type: 'nomenclature',
    fields: {
      name: { value: 'Краска акриловая белая', filled: true, quality: 'good' },
      article: { value: null, filled: false, quality: 'error' },
      unit: { value: 'л', filled: true, quality: 'good' },
      description: { value: 'Краска для внутренних работ', filled: true, quality: 'good' },
      price: { value: 450, filled: true, quality: 'good' },
      category: { value: 'Лакокрасочные материалы', filled: true, quality: 'good' },
    },
  },
]

const mockCounterpartyRecords: PreviewRecord[] = [
  {
    id: '1',
    name: 'ООО "СтройМатериалы"',
    type: 'counterparty',
    fields: {
      name: { value: 'ООО "СтройМатериалы"', filled: true, quality: 'good' },
      inn: { value: '7701234567', filled: true, quality: 'good' },
      legalAddress: { value: 'г. Москва, ул. Строителей, д. 1', filled: true, quality: 'good' },
      actualAddress: { value: 'г. Москва, ул. Строителей, д. 1', filled: true, quality: 'good' },
      phone: { value: '+7 (495) 123-45-67', filled: true, quality: 'good' },
      email: { value: 'info@stroymat.ru', filled: true, quality: 'good' },
      bankDetails: { value: null, filled: false, quality: 'error' },
    },
  },
  {
    id: '2',
    name: 'ИП Иванов Иван Иванович',
    type: 'counterparty',
    fields: {
      name: { value: 'ИП Иванов Иван Иванович', filled: true, quality: 'good' },
      inn: { value: null, filled: false, quality: 'error' },
      legalAddress: { value: 'г. Москва, ул. Примерная, д. 5', filled: true, quality: 'warning' },
      actualAddress: { value: null, filled: false, quality: 'error' },
      phone: { value: '+7 (495) 555-12-34', filled: true, quality: 'good' },
      email: { value: null, filled: false, quality: 'error' },
      bankDetails: { value: null, filled: false, quality: 'error' },
    },
  },
]

export function DataStructurePreview({
  normalizationType,
  records,
  isLoading = false,
  onExport,
  className,
}: DataStructurePreviewProps) {
  const [searchQuery, setSearchQuery] = useState('')
  const [currentPage, setCurrentPage] = useState(1)
  const [filterQuality, setFilterQuality] = useState<'all' | 'good' | 'warning' | 'error'>('all')
  const [showDetails, setShowDetails] = useState<Set<string>>(new Set())
  const recordsPerPage = 10

  // Используем моковые данные, если записи не переданы
  const allRecords = useMemo(() => {
    if (records) return records
    
    if (normalizationType === 'nomenclature' || normalizationType === 'both') {
      return mockNomenclatureRecords
    }
    if (normalizationType === 'counterparties') {
      return mockCounterpartyRecords
    }
    return []
  }, [records, normalizationType])

  const filteredRecords = useMemo(() => {
    let filtered = allRecords

    // Фильтр по поисковому запросу
    if (searchQuery) {
      filtered = filtered.filter(record =>
        record.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        Object.values(record.fields).some(field =>
          field.value?.toString().toLowerCase().includes(searchQuery.toLowerCase())
        )
      )
    }

    // Фильтр по качеству
    if (filterQuality !== 'all') {
      filtered = filtered.filter(record =>
        Object.values(record.fields).some(field => field.quality === filterQuality)
      )
    }

    return filtered
  }, [allRecords, searchQuery, filterQuality])

  const paginatedRecords = useMemo(() => {
    const start = (currentPage - 1) * recordsPerPage
    return filteredRecords.slice(start, start + recordsPerPage)
  }, [filteredRecords, currentPage])

  const totalPages = Math.ceil(filteredRecords.length / recordsPerPage)

  const toggleDetails = (id: string) => {
    const newSet = new Set(showDetails)
    if (newSet.has(id)) {
      newSet.delete(id)
    } else {
      newSet.add(id)
    }
    setShowDetails(newSet)
  }

  const getFieldIcon = (fieldName: string) => {
    const icons: { [key: string]: React.ComponentType<{ className?: string }> } = {
      name: Package,
      article: Hash,
      unit: Ruler,
      description: FileText,
      price: DollarSign,
      category: Package,
      inn: Hash,
      legalAddress: MapPin,
      actualAddress: MapPin,
      phone: Phone,
      email: Mail,
      bankDetails: DollarSign,
    }
    return icons[fieldName] || FileText
  }

  const getQualityBadge = (quality?: 'good' | 'warning' | 'error') => {
    switch (quality) {
      case 'good':
        return <CheckCircle2 className="h-4 w-4 text-green-600" />
      case 'warning':
        return <AlertCircle className="h-4 w-4 text-yellow-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      default:
        return null
    }
  }

  if (isLoading) {
    return (
      <Card className={className}>
        <CardHeader>
          <CardTitle>Предпросмотр структуры данных</CardTitle>
          <CardDescription>Загрузка данных...</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-4">
            {[...Array(5)].map((_, i) => (
              <Skeleton key={i} className="h-20" />
            ))}
          </div>
        </CardContent>
      </Card>
    )
  }

  const showNomenclature = normalizationType === 'nomenclature' || normalizationType === 'both'
  const showCounterparties = normalizationType === 'counterparties' || normalizationType === 'both'

  return (
    <div className={cn('space-y-6', className)}>
      {/* Номенклатура */}
      {showNomenclature && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <Card className="bg-gradient-to-br from-blue-50/50 to-purple-50/50 dark:from-blue-950/20 dark:to-purple-950/20 border-blue-200/50">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg font-semibold flex items-center gap-2">
                    <Package className="h-5 w-5 text-blue-600" />
                    Предпросмотр номенклатуры
                  </CardTitle>
                  <CardDescription className="mt-1">
                    Примеры записей товаров и услуг • {allRecords.length} записей
                  </CardDescription>
                </div>
                {onExport && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onExport('json')}
                    className="gap-2"
                  >
                    <Download className="h-4 w-4" />
                    Экспорт
                  </Button>
                )}
              </div>
            </CardHeader>
            <CardContent>
              {/* Фильтры и поиск */}
              <div className="flex items-center gap-4 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Поиск по названию, артикулу..."
                    value={searchQuery}
                    onChange={(e) => {
                      setSearchQuery(e.target.value)
                      setCurrentPage(1)
                    }}
                    className="pl-10"
                  />
                </div>
                <Select value={filterQuality} onValueChange={(value: any) => {
                  setFilterQuality(value)
                  setCurrentPage(1)
                }}>
                  <SelectTrigger className="w-[180px]">
                    <SelectValue placeholder="Фильтр по качеству" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Все записи</SelectItem>
                    <SelectItem value="good">Только заполненные</SelectItem>
                    <SelectItem value="warning">С предупреждениями</SelectItem>
                    <SelectItem value="error">С ошибками</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Таблица записей */}
              <div className="border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[50px]"></TableHead>
                      <TableHead>Название</TableHead>
                      <TableHead>Артикул</TableHead>
                      <TableHead>Единица</TableHead>
                      <TableHead>Цена</TableHead>
                      <TableHead>Качество</TableHead>
                      <TableHead className="w-[100px]">Действия</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <AnimatePresence>
                      {paginatedRecords
                        .filter(r => r.type === 'nomenclature' || !r.type)
                        .map((record) => (
                          <motion.tr
                            key={record.id}
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="border-b"
                          >
                            <TableCell>
                              <Package className="h-4 w-4 text-muted-foreground" />
                            </TableCell>
                            <TableCell className="font-medium">{record.name}</TableCell>
                            <TableCell>
                              {record.fields.article?.filled ? (
                                <Badge variant="outline">{record.fields.article.value}</Badge>
                              ) : (
                                <span className="text-muted-foreground text-sm">—</span>
                              )}
                            </TableCell>
                            <TableCell>
                              {record.fields.unit?.filled ? (
                                <span>{record.fields.unit.value}</span>
                              ) : (
                                <span className="text-muted-foreground text-sm">—</span>
                              )}
                            </TableCell>
                            <TableCell>
                              {record.fields.price?.filled ? (
                                <span className="font-medium">{record.fields.price.value} ₽</span>
                              ) : (
                                <span className="text-muted-foreground text-sm">—</span>
                              )}
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                {Object.values(record.fields).map((field, idx) => (
                                  <div key={idx}>{getQualityBadge(field.quality)}</div>
                                ))}
                              </div>
                            </TableCell>
                            <TableCell>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => toggleDetails(record.id)}
                                className="gap-1"
                              >
                                {showDetails.has(record.id) ? (
                                  <>
                                    <EyeOff className="h-4 w-4" />
                                    Скрыть
                                  </>
                                ) : (
                                  <>
                                    <Eye className="h-4 w-4" />
                                    Детали
                                  </>
                                )}
                              </Button>
                            </TableCell>
                          </motion.tr>
                        ))}
                    </AnimatePresence>
                  </TableBody>
                </Table>

                {/* Детали записи */}
                <AnimatePresence>
                  {paginatedRecords
                    .filter(r => showDetails.has(r.id))
                    .map((record) => (
                      <motion.tr
                        key={`details-${record.id}`}
                        initial={{ opacity: 0, height: 0 }}
                        animate={{ opacity: 1, height: 'auto' }}
                        exit={{ opacity: 0, height: 0 }}
                        className="bg-muted/30"
                      >
                        <TableCell colSpan={7} className="p-4">
                          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
                            {Object.entries(record.fields).map(([fieldName, field]) => {
                              const Icon = getFieldIcon(fieldName)
                              return (
                                <div key={fieldName} className="space-y-1">
                                  <div className="flex items-center gap-2 text-sm font-medium">
                                    <Icon className="h-4 w-4 text-muted-foreground" />
                                    <span className="capitalize">{fieldName}</span>
                                    {getQualityBadge(field.quality)}
                                  </div>
                                  <div className="text-sm text-muted-foreground">
                                    {field.filled ? (
                                      <span>{field.value}</span>
                                    ) : (
                                      <span className="italic">Не заполнено</span>
                                    )}
                                  </div>
                                </div>
                              )
                            })}
                          </div>
                        </TableCell>
                      </motion.tr>
                    ))}
                </AnimatePresence>
              </div>

              {/* Пагинация */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    Показано {((currentPage - 1) * recordsPerPage) + 1} - {Math.min(currentPage * recordsPerPage, filteredRecords.length)} из {filteredRecords.length}
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                      disabled={currentPage === 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>
                    <span className="text-sm font-medium">
                      Страница {currentPage} из {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                      disabled={currentPage === totalPages}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      )}

      {/* Контрагенты */}
      {showCounterparties && (
        <motion.div
          initial={{ opacity: 0, y: 10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3, delay: 0.1 }}
        >
          <Card className="bg-gradient-to-br from-green-50/50 to-emerald-50/50 dark:from-green-950/20 dark:to-emerald-950/20 border-green-200/50">
            <CardHeader>
              <div className="flex items-center justify-between">
                <div>
                  <CardTitle className="text-lg font-semibold flex items-center gap-2">
                    <Building2 className="h-5 w-5 text-green-600" />
                    Предпросмотр контрагентов
                  </CardTitle>
                  <CardDescription className="mt-1">
                    Примеры записей контрагентов • {allRecords.length} записей
                  </CardDescription>
                </div>
                {onExport && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => onExport('json')}
                    className="gap-2"
                  >
                    <Download className="h-4 w-4" />
                    Экспорт
                  </Button>
                )}
              </div>
            </CardHeader>
            <CardContent>
              {/* Фильтры и поиск */}
              <div className="flex items-center gap-4 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Поиск по названию, ИНН..."
                    value={searchQuery}
                    onChange={(e) => {
                      setSearchQuery(e.target.value)
                      setCurrentPage(1)
                    }}
                    className="pl-10"
                  />
                </div>
                <Select value={filterQuality} onValueChange={(value: any) => {
                  setFilterQuality(value)
                  setCurrentPage(1)
                }}>
                  <SelectTrigger className="w-[180px]">
                    <SelectValue placeholder="Фильтр по качеству" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Все записи</SelectItem>
                    <SelectItem value="good">Только заполненные</SelectItem>
                    <SelectItem value="warning">С предупреждениями</SelectItem>
                    <SelectItem value="error">С ошибками</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* Таблица записей */}
              <div className="border rounded-lg overflow-hidden">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="w-[50px]"></TableHead>
                      <TableHead>Наименование</TableHead>
                      <TableHead>ИНН/БИН</TableHead>
                      <TableHead>Адрес</TableHead>
                      <TableHead>Контакты</TableHead>
                      <TableHead>Качество</TableHead>
                      <TableHead className="w-[100px]">Действия</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    <AnimatePresence>
                      {paginatedRecords
                        .filter(r => r.type === 'counterparty' || (!r.type && showCounterparties))
                        .map((record) => (
                          <motion.tr
                            key={record.id}
                            initial={{ opacity: 0 }}
                            animate={{ opacity: 1 }}
                            exit={{ opacity: 0 }}
                            className="border-b"
                          >
                            <TableCell>
                              <Building2 className="h-4 w-4 text-muted-foreground" />
                            </TableCell>
                            <TableCell className="font-medium">{record.name}</TableCell>
                            <TableCell>
                              {record.fields.inn?.filled ? (
                                <Badge variant="outline">{record.fields.inn.value}</Badge>
                              ) : (
                                <span className="text-muted-foreground text-sm">—</span>
                              )}
                            </TableCell>
                            <TableCell>
                              {record.fields.legalAddress?.filled ? (
                                <span className="text-sm">{String(record.fields.legalAddress.value).substring(0, 30)}...</span>
                              ) : (
                                <span className="text-muted-foreground text-sm">—</span>
                              )}
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                {record.fields.phone?.filled && (
                                  <Phone className="h-3 w-3 text-muted-foreground" />
                                )}
                                {record.fields.email?.filled && (
                                  <Mail className="h-3 w-3 text-muted-foreground" />
                                )}
                                {!record.fields.phone?.filled && !record.fields.email?.filled && (
                                  <span className="text-muted-foreground text-sm">—</span>
                                )}
                              </div>
                            </TableCell>
                            <TableCell>
                              <div className="flex items-center gap-2">
                                {Object.values(record.fields).map((field, idx) => (
                                  <div key={idx}>{getQualityBadge(field.quality)}</div>
                                ))}
                              </div>
                            </TableCell>
                            <TableCell>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => toggleDetails(record.id)}
                                className="gap-1"
                              >
                                {showDetails.has(record.id) ? (
                                  <>
                                    <EyeOff className="h-4 w-4" />
                                    Скрыть
                                  </>
                                ) : (
                                  <>
                                    <Eye className="h-4 w-4" />
                                    Детали
                                  </>
                                )}
                              </Button>
                            </TableCell>
                          </motion.tr>
                        ))}
                    </AnimatePresence>
                  </TableBody>
                </Table>
              </div>

              {/* Пагинация */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between mt-4">
                  <div className="text-sm text-muted-foreground">
                    Показано {((currentPage - 1) * recordsPerPage) + 1} - {Math.min(currentPage * recordsPerPage, filteredRecords.length)} из {filteredRecords.length}
                  </div>
                  <div className="flex items-center gap-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                      disabled={currentPage === 1}
                    >
                      <ChevronLeft className="h-4 w-4" />
                    </Button>
                    <span className="text-sm font-medium">
                      Страница {currentPage} из {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                      disabled={currentPage === totalPages}
                    >
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>
        </motion.div>
      )}
    </div>
  )
}

