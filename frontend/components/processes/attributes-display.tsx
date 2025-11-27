'use client'

import React from 'react'
import { Badge } from '@/components/ui/badge'
import { 
  getConfidenceLevel, 
  getCategoryTitle, 
  formatAttributeValue,
  groupAttributesByCategory,
  getConfidenceColor,
  getConfidenceBgColor
} from '@/utils/normalization-helpers'
import { formatNumber, formatPercent } from '@/hooks/normalization-helpers'
import { Loader2 } from 'lucide-react'

interface Attribute {
  id?: number
  attribute_type?: string
  attribute_name?: string
  attribute_value: string
  unit?: string
  original_text?: string
  confidence?: number
  category?: string
}

interface AttributesDisplayProps {
  attributes: Attribute[]
  loading?: boolean
  compact?: boolean
}

export const AttributesDisplay: React.FC<AttributesDisplayProps> = ({
  attributes,
  loading = false,
  compact = false,
}) => {
  if (loading) {
    return (
      <div className="flex items-center gap-2 p-4 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>Загрузка реквизитов...</span>
      </div>
    )
  }

  if (!attributes || attributes.length === 0) {
    return (
      <div className="p-4 text-center text-sm text-muted-foreground italic">
        📝 Реквизиты не извлечены
      </div>
    )
  }

  // Группируем атрибуты по категориям
  const groupedAttributesRaw = groupAttributesByCategory(attributes)
  // Безопасная инициализация с проверкой типа
  const groupedAttributes = (groupedAttributesRaw && typeof groupedAttributesRaw === 'object' && !Array.isArray(groupedAttributesRaw))
    ? groupedAttributesRaw
    : {}

  if (compact) {
    // Компактный режим - просто список
    return (
      <div className="space-y-1">
        {attributes.map((attr, index) => (
          <div key={attr.id || index} className="flex items-center justify-between text-xs py-1">
            <span className="text-muted-foreground">
              {attr.attribute_name || attr.attribute_type || 'Атрибут'}:
            </span>
            <div className="flex items-center gap-2">
              <span className="font-medium">{formatAttributeValue(attr.attribute_value)}</span>
              {attr.unit && <span className="text-muted-foreground">{attr.unit}</span>}
              {attr.confidence !== undefined && attr.confidence < 1.0 && (
                <Badge 
                  variant="outline" 
                  className={`text-[10px] ${getConfidenceBgColor(attr.confidence * 100)}`}
                >
                  {(attr.confidence * 100).toFixed(0)}%
                </Badge>
              )}
            </div>
          </div>
        ))}
      </div>
    )
  }

  // Детальный режим - с группировкой по категориям
  return (
    <div className="space-y-4">
      {Object.entries(groupedAttributes).map(([category, categoryAttributes]) => (
        <div key={category} className="space-y-2">
          <h5 className="text-sm font-semibold text-foreground border-b pb-1">
            {getCategoryTitle(category)}
          </h5>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2">
            {categoryAttributes.map((attr, index) => (
              <div
                key={attr.id || index}
                className="bg-muted/50 border rounded-lg p-3 space-y-1"
              >
                <div className="flex items-center justify-between">
                  <span className="text-xs font-medium text-muted-foreground uppercase">
                    {attr.attribute_name || attr.attribute_type || 'Атрибут'}
                  </span>
                  {attr.confidence !== undefined && attr.confidence < 1.0 && (
                    <Badge
                      variant="outline"
                      className={`text-[10px] ${getConfidenceBgColor(attr.confidence * 100)}`}
                    >
                      {(attr.confidence * 100).toFixed(0)}%
                    </Badge>
                  )}
                </div>
                <div className="flex items-baseline gap-1">
                  <span className="font-semibold text-sm">
                    {formatAttributeValue(attr.attribute_value)}
                  </span>
                  {attr.unit && (
                    <span className="text-xs text-muted-foreground">{attr.unit}</span>
                  )}
                </div>
                {attr.original_text && (
                  <div className="text-xs text-muted-foreground mt-1 italic">
                    Из: &quot;{attr.original_text}&quot;
                  </div>
                )}
                {attr.attribute_type && (
                  <div className="mt-1">
                    <Badge variant="outline" className="text-[10px]">
                      {attr.attribute_type}
                    </Badge>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

