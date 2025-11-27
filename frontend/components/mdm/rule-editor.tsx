'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Code, FileText, Settings, TestTube, Play, Save, X } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface Rule {
  id?: string
  name: string
  type: 'naming' | 'categorization' | 'validation' | 'enrichment'
  pattern?: string
  replacement?: string
  stopWords?: string[]
  synonyms?: Record<string, string>
  enabled: boolean
  priority: number
}

interface RuleEditorProps {
  rule?: Rule
  onSave: (rule: Rule) => void
  onCancel: () => void
  onTest?: (rule: Rule) => Promise<any>
}

export const RuleEditor: React.FC<RuleEditorProps> = ({
  rule,
  onSave,
  onCancel,
  onTest,
}) => {
  const [editedRule, setEditedRule] = useState<Rule>(
    rule || {
      name: '',
      type: 'naming',
      enabled: true,
      priority: 0,
      stopWords: [],
      synonyms: {},
    }
  )
  const [testResult, setTestResult] = useState<any>(null)
  const [testing, setTesting] = useState(false)
  const [errors, setErrors] = useState<string[]>([])

  const validateRule = (): string[] => {
    const errs: string[] = []
    if (!editedRule.name.trim()) {
      errs.push('Название правила обязательно')
    }
    if (editedRule.type === 'naming' && !editedRule.pattern) {
      errs.push('Паттерн обязателен для правил наименования')
    }
    return errs
  }

  const handleSave = () => {
    const validationErrors = validateRule()
    if (validationErrors.length > 0) {
      setErrors(validationErrors)
      return
    }
    setErrors([])
    onSave(editedRule)
  }

  const handleTest = async () => {
    if (!onTest) return

    const validationErrors = validateRule()
    if (validationErrors.length > 0) {
      setErrors(validationErrors)
      return
    }

    setTesting(true)
    setErrors([])
    try {
      const result = await onTest(editedRule)
      setTestResult(result)
    } catch (error) {
      setErrors([error instanceof Error ? error.message : 'Ошибка тестирования'])
    } finally {
      setTesting(false)
    }
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>
                {rule ? 'Редактирование правила' : 'Создание нового правила'}
              </CardTitle>
              <CardDescription>
                Настройка правила для автоматической нормализации
              </CardDescription>
            </div>
            <Badge variant={editedRule.enabled ? 'default' : 'secondary'}>
              {editedRule.enabled ? 'Активно' : 'Неактивно'}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {errors.length > 0 && (
            <Alert variant="destructive">
              <AlertDescription>
                <ul className="list-disc list-inside">
                  {errors.map((error, idx) => (
                    <li key={idx}>{error}</li>
                  ))}
                </ul>
              </AlertDescription>
            </Alert>
          )}

          <div className="space-y-2">
            <Label htmlFor="rule-name">Название правила *</Label>
            <Input
              id="rule-name"
              value={editedRule.name}
              onChange={(e) => setEditedRule({ ...editedRule, name: e.target.value })}
              placeholder="Например: Замена ООО на полное название"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label htmlFor="rule-type">Тип правила</Label>
              <select
                id="rule-type"
                value={editedRule.type}
                onChange={(e) => setEditedRule({ ...editedRule, type: e.target.value as any })}
                className="flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm"
              >
                <option value="naming">Наименование</option>
                <option value="categorization">Категоризация</option>
                <option value="validation">Валидация</option>
                <option value="enrichment">Обогащение</option>
              </select>
            </div>

            <div className="space-y-2">
              <Label htmlFor="rule-priority">Приоритет</Label>
              <Input
                id="rule-priority"
                type="number"
                min="0"
                max="100"
                value={editedRule.priority}
                onChange={(e) => setEditedRule({ ...editedRule, priority: parseInt(e.target.value) || 0 })}
              />
            </div>
          </div>

          <Tabs defaultValue="pattern">
            <TabsList>
              <TabsTrigger value="pattern">
                <Code className="h-4 w-4 mr-2" />
                Паттерн
              </TabsTrigger>
              <TabsTrigger value="stopwords">
                <FileText className="h-4 w-4 mr-2" />
                Стоп-слова
              </TabsTrigger>
              <TabsTrigger value="synonyms">
                <Settings className="h-4 w-4 mr-2" />
                Синонимы
              </TabsTrigger>
            </TabsList>

            <TabsContent value="pattern" className="space-y-4 mt-4">
              <div className="space-y-2">
                <Label htmlFor="rule-pattern">Регулярное выражение</Label>
                <Input
                  id="rule-pattern"
                  value={editedRule.pattern || ''}
                  onChange={(e) => setEditedRule({ ...editedRule, pattern: e.target.value })}
                  placeholder="Например: ^ООО\\s+(.+)$"
                />
                <p className="text-xs text-muted-foreground">
                  Используйте регулярные выражения для поиска паттернов
                </p>
              </div>

              {editedRule.type === 'naming' && (
                <div className="space-y-2">
                  <Label htmlFor="rule-replacement">Замена</Label>
                  <Input
                    id="rule-replacement"
                    value={editedRule.replacement || ''}
                    onChange={(e) => setEditedRule({ ...editedRule, replacement: e.target.value })}
                    placeholder="Например: Общество с ограниченной ответственностью $1"
                  />
                  <p className="text-xs text-muted-foreground">
                    Используйте $1, $2 и т.д. для ссылок на группы из паттерна
                  </p>
                </div>
              )}
            </TabsContent>

            <TabsContent value="stopwords" className="space-y-4 mt-4">
              <div className="space-y-2">
                <Label>Стоп-слова (по одному на строку)</Label>
                <Textarea
                  value={editedRule.stopWords?.join('\n') || ''}
                  onChange={(e) => {
                    const words = e.target.value.split('\n').filter(w => w.trim())
                    setEditedRule({ ...editedRule, stopWords: words })
                  }}
                  placeholder="ООО&#10;ЗАО&#10;ОАО"
                  rows={6}
                />
              </div>
            </TabsContent>

            <TabsContent value="synonyms" className="space-y-4 mt-4">
              <div className="space-y-2">
                <Label>Синонимы (формат: ключ=значение, по одному на строку)</Label>
                <Textarea
                  value={Object.entries(editedRule.synonyms || {}).map(([k, v]) => `${k}=${v}`).join('\n')}
                  onChange={(e) => {
                    const synonyms: Record<string, string> = {}
                    e.target.value.split('\n').forEach(line => {
                      const [key, value] = line.split('=')
                      if (key && value) {
                        synonyms[key.trim()] = value.trim()
                      }
                    })
                    setEditedRule({ ...editedRule, synonyms })
                  }}
                  placeholder="ООО=Общество с ограниченной ответственностью&#10;ЗАО=Закрытое акционерное общество"
                  rows={6}
                />
              </div>
            </TabsContent>
          </Tabs>

          <div className="flex items-center space-x-2">
            <input
              type="checkbox"
              id="rule-enabled"
              checked={editedRule.enabled}
              onChange={(e) => setEditedRule({ ...editedRule, enabled: e.target.checked })}
              className="h-4 w-4"
            />
            <Label htmlFor="rule-enabled" className="cursor-pointer">
              Правило активно
            </Label>
          </div>

          {testResult && (
            <Alert>
              <TestTube className="h-4 w-4" />
              <AlertDescription>
                <div className="space-y-2">
                  <p className="font-medium">Результаты тестирования:</p>
                  <pre className="text-xs bg-muted p-2 rounded overflow-auto">
                    {JSON.stringify(testResult, null, 2)}
                  </pre>
                </div>
              </AlertDescription>
            </Alert>
          )}

          <div className="flex gap-2 pt-4 border-t">
            <Button onClick={handleSave} className="flex-1">
              <Save className="h-4 w-4 mr-2" />
              Сохранить
            </Button>
            {onTest && (
              <Button variant="outline" onClick={handleTest} disabled={testing}>
                <Play className="h-4 w-4 mr-2" />
                {testing ? 'Тестирование...' : 'Тестировать'}
              </Button>
            )}
            <Button variant="outline" onClick={onCancel}>
              <X className="h-4 w-4 mr-2" />
              Отмена
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

