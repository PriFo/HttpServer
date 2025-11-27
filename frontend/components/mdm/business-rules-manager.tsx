'use client'

import React, { useState } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { FileText, Settings, CheckCircle2, AlertCircle, Plus, Edit, Trash2 } from 'lucide-react'
import { RuleEditor } from './rule-editor'
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { useProjectState } from '@/hooks/useProjectState'
import { LoadingState } from '@/components/common/loading-state'
import { ErrorState } from '@/components/common/error-state'
import { useNormalizationIdentifiers } from '@/context/NormalizationContext'
import { logger } from '@/lib/logger'
import { handleError } from '@/lib/error-handler'
import { fetchBusinessRulesApi, type BusinessRulesResponse } from '@/lib/mdm/api'

interface BusinessRulesManagerProps {
  clientId?: string
  projectId?: string
}

const ruleTypes = [
  { id: 'naming', name: 'Правила наименования', icon: FileText },
  { id: 'categorization', name: 'Правила категоризации', icon: Settings },
  { id: 'validation', name: 'Правила валидации', icon: CheckCircle2 },
  { id: 'enrichment', name: 'Правила обогащения', icon: AlertCircle },
]

export const BusinessRulesManager: React.FC<BusinessRulesManagerProps> = ({
  clientId,
  projectId,
}) => {
  const identifiers = useNormalizationIdentifiers(clientId, projectId)
  const effectiveClientId = identifiers.clientId
  const effectiveProjectId = identifiers.projectId
  const [activeRuleType, setActiveRuleType] = useState(ruleTypes[0].id)
  const [editingRule, setEditingRule] = useState<any | null>(null)
  const [showEditor, setShowEditor] = useState(false)

  const { data: rulesData, loading, error, refetch } = useProjectState<BusinessRulesResponse>(
    (cid, pid, signal) => fetchBusinessRulesApi(cid, pid, activeRuleType, signal),
    effectiveClientId || '',
    effectiveProjectId || '',
    [activeRuleType],
    {
      enabled: !!effectiveClientId && !!effectiveProjectId,
      // Не используем автообновление для правил, так как они меняются редко
      refetchInterval: null,
    }
  )

  const rules = rulesData?.rules || []

  const handleSaveRule = async (rule: any) => {
    try {
      const method = rule.id ? 'PUT' : 'POST'
      if (!effectiveClientId || !effectiveProjectId) return
      const url = `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/rules${rule.id ? `?ruleId=${rule.id}` : ''}`

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(rule),
      })

      if (response.ok) {
        // Обновляем данные после сохранения
        await refetch()
        setShowEditor(false)
        setEditingRule(null)
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'BusinessRulesManager',
          action: 'save',
          ruleId: rule.id,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось сохранить правило',
      })
    }
  }

  const handleDeleteRule = async (ruleId: string) => {
    if (!effectiveClientId || !effectiveProjectId) return
    try {
      const response = await fetch(
        `/api/clients/${effectiveClientId}/projects/${effectiveProjectId}/normalization/rules?ruleId=${ruleId}`,
        {
          method: 'DELETE',
        }
      )

      if (response.ok) {
        // Обновляем данные после удаления
        await refetch()
      }
    } catch (error) {
      handleError(error, {
        context: {
          component: 'BusinessRulesManager',
          action: 'delete',
          ruleId,
          clientId: effectiveClientId,
          projectId: effectiveProjectId,
        },
        fallbackMessage: 'Не удалось удалить правило',
      })
    }
  }

  const handleTestRule = async (rule: any) => {
    // Заглушка для тестирования правила
    return {
      matched: 10,
      transformed: 10,
      examples: [
        { original: 'ООО Компания', transformed: 'Общество с ограниченной ответственностью Компания' },
      ],
    }
  }

  const activeRules = rules.filter((r: any) => r.type === activeRuleType)

  return (
    <div className="space-y-4">
      {!effectiveClientId || !effectiveProjectId ? (
        <Card>
          <CardHeader>
            <CardTitle>Управление бизнес-правилами</CardTitle>
            <CardDescription>Выберите проект для настройки правил</CardDescription>
          </CardHeader>
        </Card>
      ) : loading && !rulesData ? (
        <LoadingState message="Загрузка бизнес-правил..." />
      ) : error ? (
        <ErrorState
          title="Ошибка загрузки правил"
          message={error}
          action={{ label: 'Повторить', onClick: refetch }}
        />
      ) : (
      <Card>
        <CardHeader>
          <CardTitle>Управление бизнес-правилами</CardTitle>
          <CardDescription>
            Настройка правил нормализации данных
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Tabs value={activeRuleType} onValueChange={setActiveRuleType}>
            <TabsList>
              {ruleTypes.map((type) => (
                <TabsTrigger key={type.id} value={type.id}>
                  <type.icon className="h-4 w-4 mr-2" />
                  {type.name}
                </TabsTrigger>
              ))}
            </TabsList>

            <TabsContent value={activeRuleType} className="mt-4">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div>
                    <h3 className="font-semibold">
                      {ruleTypes.find(t => t.id === activeRuleType)?.name}
                    </h3>
                    <p className="text-sm text-muted-foreground">
                      Настройка правил для {activeRuleType === 'naming' && 'наименования'}
                      {activeRuleType === 'categorization' && 'категоризации'}
                      {activeRuleType === 'validation' && 'валидации'}
                      {activeRuleType === 'enrichment' && 'обогащения'}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Dialog open={showEditor} onOpenChange={setShowEditor}>
                      <DialogTrigger asChild>
                        <Button size="sm" onClick={() => {
                          setEditingRule(null)
                          setShowEditor(true)
                        }}>
                          <Plus className="h-4 w-4 mr-2" />
                          Создать правило
                        </Button>
                      </DialogTrigger>
                      <DialogContent className="max-w-3xl max-h-[90vh] overflow-y-auto">
                        <DialogHeader>
                          <DialogTitle>
                            {editingRule ? 'Редактирование правила' : 'Создание нового правила'}
                          </DialogTitle>
                          <DialogDescription>
                            Настройка правила для автоматической нормализации
                          </DialogDescription>
                        </DialogHeader>
                        <RuleEditor
                          rule={editingRule}
                          onSave={handleSaveRule}
                          onCancel={() => {
                            setShowEditor(false)
                            setEditingRule(null)
                          }}
                          onTest={handleTestRule}
                        />
                      </DialogContent>
                    </Dialog>
                    <Button size="sm" variant="outline">Импортировать</Button>
                  </div>
                </div>

                <div className="p-4 border rounded-lg space-y-3">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm font-medium">Активных правил</p>
                      <p className="text-xs text-muted-foreground">Применяются к новым данным</p>
                    </div>
                    <Badge variant="default">{activeRules.filter((r: any) => r.enabled).length}</Badge>
                  </div>
                </div>

                {/* Список правил */}
                {activeRules.length > 0 ? (
                  <div className="space-y-2">
                    {activeRules.map((rule: any) => (
                      <Card key={rule.id}>
                        <CardContent className="p-4">
                          <div className="flex items-center justify-between">
                            <div className="flex-1">
                              <div className="flex items-center gap-2 mb-1">
                                <span className="font-medium">{rule.name}</span>
                                <Badge variant={rule.enabled ? 'default' : 'secondary'}>
                                  {rule.enabled ? 'Активно' : 'Неактивно'}
                                </Badge>
                                <Badge variant="outline">Приоритет: {rule.priority}</Badge>
                              </div>
                              {rule.pattern && (
                                <p className="text-xs text-muted-foreground font-mono">
                                  {rule.pattern}
                                </p>
                              )}
                            </div>
                            <div className="flex gap-2">
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => {
                                  setEditingRule(rule)
                                  setShowEditor(true)
                                }}
                              >
                                <Edit className="h-4 w-4" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => handleDeleteRule(rule.id)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
                          </div>
                        </CardContent>
                      </Card>
                    ))}
                  </div>
                ) : (
                  <div className="p-4 border rounded-lg text-center text-muted-foreground">
                    <p className="mb-2">Правила не созданы</p>
                    <p className="text-xs">Создайте первое правило для автоматической нормализации данных</p>
                  </div>
                )}
              </div>
            </TabsContent>
          </Tabs>
        </CardContent>
      </Card>
      )}
    </div>
  )
}

