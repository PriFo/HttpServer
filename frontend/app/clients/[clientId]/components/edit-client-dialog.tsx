'use client'

import { useState, useEffect } from 'react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Loader2, Save } from "lucide-react"
import { toast } from "sonner"
import type { Client } from '@/types'
import { isValidEmail, isValidURL, isValidINN, isValidOGRN, isValidKPP, isValidBIK, isValidPhone } from '@/lib/validation'

interface EditClientDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  client: Client | null
  onSuccess: () => void
}

export function EditClientDialog({
  open,
  onOpenChange,
  client,
  onSuccess,
}: EditClientDialogProps) {
  const [formData, setFormData] = useState<Partial<Client>>({})
  const [isSubmitting, setIsSubmitting] = useState(false)

  useEffect(() => {
    if (client) {
      setFormData({
        name: client.name || '',
        legal_name: client.legal_name || '',
        description: client.description || '',
        contact_email: client.contact_email || '',
        contact_phone: client.contact_phone || '',
        tax_id: client.tax_id || '',
        country: client.country || '',
        status: client.status || 'active',
        // Бизнес-информация
        industry: client.industry || '',
        company_size: client.company_size || '',
        legal_form: client.legal_form || '',
        // Расширенные контакты
        contact_person: client.contact_person || '',
        contact_position: client.contact_position || '',
        alternate_phone: client.alternate_phone || '',
        website: client.website || '',
        // Юридические данные
        ogrn: client.ogrn || '',
        kpp: client.kpp || '',
        legal_address: client.legal_address || '',
        postal_address: client.postal_address || '',
        bank_name: client.bank_name || '',
        bank_account: client.bank_account || '',
        correspondent_account: client.correspondent_account || '',
        bik: client.bik || '',
        // Договорные данные
        contract_number: client.contract_number || '',
        contract_date: client.contract_date || '',
        contract_terms: client.contract_terms || '',
        contract_expires_at: client.contract_expires_at || '',
      })
    }
  }, [client])

  const validateForm = (): string | null => {
    // Валидация обязательных полей
    if (!formData.name || formData.name.trim().length === 0) {
      return 'Название клиента обязательно для заполнения'
    }

    // Валидация email
    if (formData.contact_email && formData.contact_email.trim() && !isValidEmail(formData.contact_email)) {
      return 'Некорректный формат email адреса'
    }

    // Валидация телефона
    if (formData.contact_phone && formData.contact_phone.trim() && !isValidPhone(formData.contact_phone)) {
      return 'Некорректный формат телефона'
    }

    if (formData.alternate_phone && formData.alternate_phone.trim() && !isValidPhone(formData.alternate_phone)) {
      return 'Некорректный формат дополнительного телефона'
    }

    // Валидация ИНН
    if (formData.tax_id && formData.tax_id.trim() && !isValidINN(formData.tax_id)) {
      return 'ИНН должен содержать 10 или 12 цифр'
    }

    // Валидация ОГРН
    if (formData.ogrn && formData.ogrn.trim() && !isValidOGRN(formData.ogrn)) {
      return 'ОГРН должен содержать 13 или 15 цифр'
    }

    // Валидация КПП
    if (formData.kpp && formData.kpp.trim() && !isValidKPP(formData.kpp)) {
      return 'КПП должен содержать 9 цифр'
    }

    // Валидация БИК
    if (formData.bik && formData.bik.trim() && !isValidBIK(formData.bik)) {
      return 'БИК должен содержать 9 цифр'
    }

    // Валидация URL
    if (formData.website && formData.website.trim()) {
      // Добавляем протокол если его нет
      const urlToCheck = formData.website.startsWith('http://') || formData.website.startsWith('https://')
        ? formData.website
        : `https://${formData.website}`
      
      if (!isValidURL(urlToCheck)) {
        return 'Некорректный формат URL. Используйте формат: example.com или http://example.com'
      }
    }

    return null
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!client) return

    // Валидация формы
    const validationError = validateForm()
    if (validationError) {
      toast.error(validationError)
      return
    }

    setIsSubmitting(true)
    try {
      const response = await fetch(`/api/clients/${client.id}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(formData),
      })

      if (!response.ok) {
        const errorData = await response.json().catch(() => ({ error: 'Failed to update client' }))
        throw new Error(errorData.error || 'Failed to update client')
      }

      toast.success('Клиент успешно обновлен')
      onSuccess()
      onOpenChange(false)
    } catch (error) {
      console.error('Update error:', error)
      toast.error(error instanceof Error ? error.message : 'Ошибка обновления клиента')
    } finally {
      setIsSubmitting(false)
    }
  }

  const handleChange = (field: keyof Client, value: string) => {
    setFormData(prev => ({
      ...prev,
      [field]: value,
    }))
  }

  if (!client) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Редактирование клиента</DialogTitle>
          <DialogDescription>
            Обновите информацию о клиенте. Все поля опциональны.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit}>
          <Tabs defaultValue="basic" className="w-full">
            <TabsList className="grid w-full grid-cols-4">
              <TabsTrigger value="basic">Основное</TabsTrigger>
              <TabsTrigger value="contacts">Контакты</TabsTrigger>
              <TabsTrigger value="legal">Юридическое</TabsTrigger>
              <TabsTrigger value="contract">Договор</TabsTrigger>
            </TabsList>

            {/* Основная информация */}
            <TabsContent value="basic" className="space-y-4 mt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Название *</Label>
                  <Input
                    id="name"
                    value={formData.name || ''}
                    onChange={(e) => handleChange('name', e.target.value)}
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="legal_name">Юридическое название</Label>
                  <Input
                    id="legal_name"
                    value={formData.legal_name || ''}
                    onChange={(e) => handleChange('legal_name', e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="description">Описание</Label>
                <Textarea
                  id="description"
                  value={formData.description || ''}
                  onChange={(e) => handleChange('description', e.target.value)}
                  rows={3}
                />
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="tax_id">ИНН/БИН</Label>
                  <Input
                    id="tax_id"
                    value={formData.tax_id || ''}
                    onChange={(e) => handleChange('tax_id', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="country">Страна</Label>
                  <Input
                    id="country"
                    value={formData.country || ''}
                    onChange={(e) => handleChange('country', e.target.value)}
                    placeholder="RU, US, etc."
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="status">Статус</Label>
                  <Input
                    id="status"
                    value={formData.status || 'active'}
                    onChange={(e) => handleChange('status', e.target.value)}
                  />
                </div>
              </div>

              <div className="grid grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="industry">Отрасль</Label>
                  <Input
                    id="industry"
                    value={formData.industry || ''}
                    onChange={(e) => handleChange('industry', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="company_size">Размер компании</Label>
                  <Input
                    id="company_size"
                    value={formData.company_size || ''}
                    onChange={(e) => handleChange('company_size', e.target.value)}
                    placeholder="Малый, Средний, Крупный"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="legal_form">Правовая форма</Label>
                  <Input
                    id="legal_form"
                    value={formData.legal_form || ''}
                    onChange={(e) => handleChange('legal_form', e.target.value)}
                    placeholder="ООО, ИП, ЗАО"
                  />
                </div>
              </div>
            </TabsContent>

            {/* Контактная информация */}
            <TabsContent value="contacts" className="space-y-4 mt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="contact_person">Контактное лицо</Label>
                  <Input
                    id="contact_person"
                    value={formData.contact_person || ''}
                    onChange={(e) => handleChange('contact_person', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="contact_position">Должность</Label>
                  <Input
                    id="contact_position"
                    value={formData.contact_position || ''}
                    onChange={(e) => handleChange('contact_position', e.target.value)}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="contact_email">Email</Label>
                  <Input
                    id="contact_email"
                    type="email"
                    value={formData.contact_email || ''}
                    onChange={(e) => handleChange('contact_email', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="contact_phone">Телефон</Label>
                  <Input
                    id="contact_phone"
                    value={formData.contact_phone || ''}
                    onChange={(e) => handleChange('contact_phone', e.target.value)}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="alternate_phone">Доп. телефон</Label>
                  <Input
                    id="alternate_phone"
                    value={formData.alternate_phone || ''}
                    onChange={(e) => handleChange('alternate_phone', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="website">Веб-сайт</Label>
                  <Input
                    id="website"
                    type="url"
                    value={formData.website || ''}
                    onChange={(e) => handleChange('website', e.target.value)}
                    placeholder="https://example.com"
                  />
                </div>
              </div>
            </TabsContent>

            {/* Юридическая информация */}
            <TabsContent value="legal" className="space-y-4 mt-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="ogrn">ОГРН</Label>
                  <Input
                    id="ogrn"
                    value={formData.ogrn || ''}
                    onChange={(e) => handleChange('ogrn', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="kpp">КПП</Label>
                  <Input
                    id="kpp"
                    value={formData.kpp || ''}
                    onChange={(e) => handleChange('kpp', e.target.value)}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="legal_address">Юридический адрес</Label>
                <Textarea
                  id="legal_address"
                  value={formData.legal_address || ''}
                  onChange={(e) => handleChange('legal_address', e.target.value)}
                  rows={2}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="postal_address">Почтовый адрес</Label>
                <Textarea
                  id="postal_address"
                  value={formData.postal_address || ''}
                  onChange={(e) => handleChange('postal_address', e.target.value)}
                  rows={2}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="bank_name">Банк</Label>
                  <Input
                    id="bank_name"
                    value={formData.bank_name || ''}
                    onChange={(e) => handleChange('bank_name', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bank_account">Расчетный счет</Label>
                  <Input
                    id="bank_account"
                    value={formData.bank_account || ''}
                    onChange={(e) => handleChange('bank_account', e.target.value)}
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="correspondent_account">Корреспондентский счет</Label>
                  <Input
                    id="correspondent_account"
                    value={formData.correspondent_account || ''}
                    onChange={(e) => handleChange('correspondent_account', e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="bik">БИК</Label>
                  <Input
                    id="bik"
                    value={formData.bik || ''}
                    onChange={(e) => handleChange('bik', e.target.value)}
                  />
                </div>
              </div>
            </TabsContent>

            {/* Договорная информация */}
            <TabsContent value="contract" className="space-y-4 mt-4">
              <div className="space-y-2">
                <Label htmlFor="contract_number">Номер договора</Label>
                <Input
                  id="contract_number"
                  value={formData.contract_number || ''}
                  onChange={(e) => handleChange('contract_number', e.target.value)}
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="contract_date">Дата договора</Label>
                  <Input
                    id="contract_date"
                    type="date"
                    value={formData.contract_date ? formData.contract_date.split('T')[0] : ''}
                    onChange={(e) => handleChange('contract_date', e.target.value ? `${e.target.value}T00:00:00` : '')}
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="contract_expires_at">Действует до</Label>
                  <Input
                    id="contract_expires_at"
                    type="date"
                    value={formData.contract_expires_at ? formData.contract_expires_at.split('T')[0] : ''}
                    onChange={(e) => handleChange('contract_expires_at', e.target.value ? `${e.target.value}T00:00:00` : '')}
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="contract_terms">Условия договора</Label>
                <Textarea
                  id="contract_terms"
                  value={formData.contract_terms || ''}
                  onChange={(e) => handleChange('contract_terms', e.target.value)}
                  rows={4}
                />
              </div>
            </TabsContent>
          </Tabs>

          <DialogFooter className="mt-6">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              Отмена
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Сохранение...
                </>
              ) : (
                <>
                  <Save className="h-4 w-4 mr-2" />
                  Сохранить
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

