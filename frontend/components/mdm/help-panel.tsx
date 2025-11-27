'use client'

import React, { useState, useMemo } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { HelpCircle, ChevronDown, ChevronUp, Book, Video, MessageCircle, Search } from 'lucide-react'
import { Input } from '@/components/ui/input'

interface HelpPanelProps {
  section?: 'overview' | 'nomenclature' | 'counterparties' | 'quality' | 'classification'
}

const helpTopics = {
  overview: [
    {
      id: 'getting-started',
      title: 'Начало работы',
      content: 'На этой вкладке вы можете видеть общий обзор процесса нормализации, мониторинг в реальном времени и результаты обработки данных.',
    },
    {
      id: 'monitoring',
      title: 'Мониторинг процесса',
      content: 'Панель мониторинга показывает метрики в реальном времени: количество обработанных записей, скорость обработки, прогресс и текущий этап.',
    },
    {
      id: 'pipeline',
      title: 'Пайплайн обработки',
      content: 'Визуализация показывает все этапы обработки данных от извлечения до публикации. Кликните на этап для просмотра деталей.',
    },
  ],
  nomenclature: [
    {
      id: 'nomenclature-basics',
      title: 'Работа с номенклатурой',
      content: 'На этой вкладке вы можете просматривать и управлять нормализованной номенклатурой, работать с дубликатами и настраивать правила.',
    },
    {
      id: 'deduplication',
      title: 'Дедупликация',
      content: 'Используйте интеллектуальную дедупликацию для объединения дубликатов. Настройте порог схожести и максимальный размер кластера.',
    },
    {
      id: 'rules',
      title: 'Бизнес-правила',
      content: 'Создавайте правила для автоматической нормализации данных. Правила применяются в порядке приоритета.',
    },
  ],
  quality: [
    {
      id: 'quality-metrics',
      title: 'Метрики качества',
      content: 'Отслеживайте качество данных по четырем измерениям: полнота, точность, согласованность и актуальность.',
    },
    {
      id: 'improvements',
      title: 'Улучшение качества',
      content: 'Используйте рекомендации системы для улучшения качества данных. Система автоматически выявляет проблемные места.',
    },
  ],
  counterparties: [
    {
      id: 'counterparties-basics',
      title: 'Работа с контрагентами',
      content: 'На этой вкладке вы можете просматривать и управлять нормализованными контрагентами, работать с дубликатами и настраивать правила.',
    },
    {
      id: 'counterparties-deduplication',
      title: 'Дедупликация контрагентов',
      content: 'Используйте интеллектуальную дедупликацию для объединения дубликатов контрагентов. Настройте порог схожести и максимальный размер кластера.',
    },
  ],
  classification: [
    {
      id: 'classification-basics',
      title: 'Классификация данных',
      content: 'На этой вкладке вы можете управлять классификацией нормализованных данных, настраивать правила классификации и просматривать результаты.',
    },
    {
      id: 'classification-rules',
      title: 'Правила классификации',
      content: 'Создавайте правила для автоматической классификации данных. Правила применяются в порядке приоритета.',
    },
  ],
}

export const HelpPanel: React.FC<HelpPanelProps> = ({ section = 'overview' }) => {
  const [expandedTopics, setExpandedTopics] = useState<Set<string>>(new Set())
  const [searchQuery, setSearchQuery] = useState('')

  const topics = helpTopics[section] || helpTopics.overview

  const filteredTopics = useMemo(() => {
    const query = searchQuery.trim().toLowerCase()
    if (!query) return topics
    return topics.filter(topic =>
      topic.title.toLowerCase().includes(query) ||
      topic.content.toLowerCase().includes(query)
    )
  }, [topics, searchQuery])

  const toggleTopic = (topicId: string) => {
    setExpandedTopics(prev => {
      const newSet = new Set(prev)
      if (newSet.has(topicId)) {
        newSet.delete(topicId)
      } else {
        newSet.add(topicId)
      }
      return newSet
    })
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <HelpCircle className="h-5 w-5" />
            <CardTitle>Справка</CardTitle>
          </div>
          <Badge variant="outline">Быстрая помощь</Badge>
        </div>
        <CardDescription>
          Полезная информация о работе с нормализацией НСИ
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center gap-2 mb-4">
          <Search className="h-4 w-4 text-muted-foreground" />
          <Input
            placeholder="Поиск по справке..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="h-8"
          />
        </div>
        <div className="space-y-2">
          {filteredTopics.length === 0 && (
            <div className="text-sm text-muted-foreground text-center py-4">
              Ничего не найдено. Попробуйте изменить запрос.
            </div>
          )}
          {filteredTopics.map((topic) => {
            const isExpanded = expandedTopics.has(topic.id)
            return (
              <Collapsible
                key={topic.id}
                open={isExpanded}
                onOpenChange={() => toggleTopic(topic.id)}
              >
                <CollapsibleTrigger asChild>
                  <Button
                    variant="ghost"
                    className="w-full justify-between"
                  >
                    <span className="font-medium">{topic.title}</span>
                    {isExpanded ? (
                      <ChevronUp className="h-4 w-4" />
                    ) : (
                      <ChevronDown className="h-4 w-4" />
                    )}
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent className="px-4 pb-2">
                  <p className="text-sm text-muted-foreground">{topic.content}</p>
                </CollapsibleContent>
              </Collapsible>
            )
          })}
        </div>

        <div className="mt-4 pt-4 border-t space-y-2">
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            onClick={() => window.open('https://docs.example.com/mdm', '_blank')}
          >
            <Book className="h-4 w-4 mr-2" />
            Полная документация
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            onClick={() => window.open('https://videos.example.com/mdm', '_blank')}
          >
            <Video className="h-4 w-4 mr-2" />
            Видео-руководства
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="w-full"
            onClick={() => window.open('mailto:support@example.com', '_blank')}
          >
            <MessageCircle className="h-4 w-4 mr-2" />
            Связаться с поддержкой
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

