'use client'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Users, Building2, Mail, Phone, Globe, FileText } from "lucide-react"

interface CounterpartyItem {
  id: number
  name: string
  normalized_name: string
  tax_id?: string
  kpp?: string
  bin?: string
  type?: string
  status: string
  quality_score?: number
  country?: string
  contact_email?: string
  contact_phone?: string
  contact_person?: string
  legal_address?: string
  postal_address?: string
  legal_name?: string
  address?: string
  description?: string
  project_name?: string
  database_name?: string
  source_reference?: string
  source_name?: string
  source_databases?: Array<{
    database_id: number
    database_name: string
    source_reference?: string
    source_name?: string
  }>
}

interface CounterpartyDetailDialogProps {
  item: CounterpartyItem
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CounterpartyDetailDialog({ item, open, onOpenChange }: CounterpartyDetailDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            {item.normalized_name || item.name || 'Контрагент'}
          </DialogTitle>
          <DialogDescription>
            {item.name && item.name !== item.normalized_name && (
              <span className="text-xs text-muted-foreground">
                Исходное название: {item.name}
              </span>
            )}
            {!item.name || item.name === item.normalized_name ? 'Детальная информация о контрагенте' : ''}
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          {/* Основная информация */}
          <Card>
            <CardHeader>
              <CardTitle className="text-base">Основная информация</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3">
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Название:</span>
                <span className="text-sm font-medium">{item.name}</span>
              </div>
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Нормализованное название:</span>
                <span className="text-sm font-medium">{item.normalized_name}</span>
              </div>
              {item.legal_name && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Юридическое название:</span>
                  <span className="text-sm font-medium">{item.legal_name}</span>
                </div>
              )}
              {(item.tax_id || item.bin) && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground flex items-center gap-1">
                    <FileText className="h-3 w-3" />
                    {item.tax_id ? 'ИНН' : 'БИН'}:
                  </span>
                  <span className="text-sm font-mono font-medium">{item.tax_id || item.bin}</span>
                </div>
              )}
              {item.kpp && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground flex items-center gap-1">
                    <FileText className="h-3 w-3" />
                    КПП:
                  </span>
                  <span className="text-sm font-mono font-medium">{item.kpp}</span>
                </div>
              )}
              {item.type && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground flex items-center gap-1">
                    <Building2 className="h-3 w-3" />
                    Источник:
                  </span>
                  <Badge variant={item.type === 'Нормализованный' || item.type === 'normalized' ? 'default' : 'secondary'}>
                    {item.type === 'Нормализованный' || item.type === 'normalized' ? 'Нормализован' : item.type === 'База данных' || item.type === 'database' ? 'База данных' : item.type}
                  </Badge>
                </div>
              )}
              {item.database_name && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">База данных:</span>
                  <span className="text-sm font-medium max-w-[60%] truncate" title={item.database_name}>
                    {item.database_name.split(/[/\\]/).pop() || item.database_name}
                  </span>
                </div>
              )}
              {item.source_databases && item.source_databases.length > 0 && (
                <div className="flex flex-col gap-2">
                  <span className="text-sm text-muted-foreground">Связанные базы данных ({item.source_databases.length}):</span>
                  <div className="flex flex-col gap-1 pl-2 border-l-2 border-muted">
                    {item.source_databases.map((db, idx) => (
                      <div key={idx} className="flex justify-between items-start gap-2">
                        <div className="flex flex-col gap-0.5 flex-1">
                          <span className="text-sm font-medium">
                            📁 {db.database_name}
                          </span>
                          {db.source_reference && (
                            <span className="text-xs text-muted-foreground font-mono">
                              Ссылка: {db.source_reference}
                            </span>
                          )}
                          {db.source_name && db.source_name !== item.name && (
                            <span className="text-xs text-muted-foreground">
                              Название: {db.source_name}
                            </span>
                          )}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
              {item.project_name && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Проект:</span>
                  <span className="text-sm font-medium">{item.project_name}</span>
                </div>
              )}
              {item.source_reference && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Ссылка на источник:</span>
                  <span className="text-sm font-mono text-xs max-w-[60%] truncate" title={item.source_reference}>
                    {item.source_reference}
                  </span>
                </div>
              )}
              {item.source_name && item.source_name !== item.name && item.source_name !== item.normalized_name && (
                <div className="flex justify-between">
                  <span className="text-sm text-muted-foreground">Исходное название:</span>
                  <span className="text-sm font-medium max-w-[60%] truncate" title={item.source_name}>
                    {item.source_name}
                  </span>
                </div>
              )}
              <div className="flex justify-between">
                <span className="text-sm text-muted-foreground">Статус:</span>
                <Badge variant={item.status === 'active' ? 'default' : 'secondary'}>
                  {item.status}
                </Badge>
              </div>
              {item.description && (
                <div>
                  <span className="text-sm text-muted-foreground">Описание:</span>
                  <div className="mt-1 text-sm">{item.description}</div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Контактная информация */}
          {(item.contact_email || item.contact_phone || item.contact_person || item.legal_address || item.postal_address || item.address) && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Контактная информация</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {item.contact_person && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground flex items-center gap-1">
                      <Users className="h-3 w-3" />
                      Контактное лицо:
                    </span>
                    <span className="text-sm font-medium">{item.contact_person}</span>
                  </div>
                )}
                {item.contact_email && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground flex items-center gap-1">
                      <Mail className="h-3 w-3" />
                      Email:
                    </span>
                    <a href={`mailto:${item.contact_email}`} className="text-sm font-medium text-primary hover:underline">
                      {item.contact_email}
                    </a>
                  </div>
                )}
                {item.contact_phone && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground flex items-center gap-1">
                      <Phone className="h-3 w-3" />
                      Телефон:
                    </span>
                    <a href={`tel:${item.contact_phone}`} className="text-sm font-medium text-primary hover:underline">
                      {item.contact_phone}
                    </a>
                  </div>
                )}
                {item.legal_address && (
                  <div>
                    <span className="text-sm text-muted-foreground">Юридический адрес:</span>
                    <div className="mt-1 text-sm font-medium">{item.legal_address}</div>
                  </div>
                )}
                {item.postal_address && item.postal_address !== item.legal_address && (
                  <div>
                    <span className="text-sm text-muted-foreground">Почтовый адрес:</span>
                    <div className="mt-1 text-sm font-medium">{item.postal_address}</div>
                  </div>
                )}
                {item.address && !item.legal_address && !item.postal_address && (
                  <div>
                    <span className="text-sm text-muted-foreground">Адрес:</span>
                    <div className="mt-1 text-sm font-medium">{item.address}</div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Дополнительная информация */}
          {(item.country || item.quality_score !== undefined) && (
            <Card>
              <CardHeader>
                <CardTitle className="text-base">Дополнительная информация</CardTitle>
              </CardHeader>
              <CardContent className="space-y-3">
                {item.country && (
                  <div className="flex justify-between">
                    <span className="text-sm text-muted-foreground flex items-center gap-1">
                      <Globe className="h-3 w-3" />
                      Страна:
                    </span>
                    <span className="text-sm font-medium">{item.country}</span>
                  </div>
                )}
                {item.quality_score !== undefined && item.quality_score !== null && (
                  <div className="space-y-2">
                    <div className="flex justify-between items-center">
                      <span className="text-sm text-muted-foreground">Оценка качества:</span>
                      <Badge variant={item.quality_score >= 0.9 ? 'default' : item.quality_score >= 0.7 ? 'secondary' : 'destructive'}>
                        {Math.round(item.quality_score * 100)}%
                      </Badge>
                    </div>
                    <div className="w-full bg-muted rounded-full h-2 overflow-hidden">
                      <div 
                        className={`h-full transition-all ${
                          item.quality_score >= 0.9 ? 'bg-primary' : 
                          item.quality_score >= 0.7 ? 'bg-yellow-500' : 
                          'bg-destructive'
                        }`}
                        style={{ width: `${Math.round(item.quality_score * 100)}%` }}
                      />
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {item.quality_score >= 0.9 ? 'Высокое качество данных' : 
                       item.quality_score >= 0.7 ? 'Среднее качество данных' : 
                       'Низкое качество данных'}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

