import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

interface KpvedBadgeProps {
  code?: string | null
  name?: string | null
  confidence?: number | null
  showConfidence?: boolean
  className?: string
}

export function KpvedBadge({ code, name, confidence, showConfidence = true, className }: KpvedBadgeProps) {
  if (!code) {
    return (
      <Badge variant="outline" className={className} aria-label="КПВЭД не классифицирован">
        Не классифицировано
      </Badge>
    )
  }

  // Улучшенная цветовая схема, согласованная с confidence badge
  const getConfidenceColor = (conf?: number | null) => {
    if (!conf) return ''
    if (conf >= 0.9) return 'text-emerald-600 dark:text-emerald-400'
    if (conf >= 0.75) return 'text-green-600 dark:text-green-400'
    if (conf >= 0.5) return 'text-amber-600 dark:text-amber-400'
    if (conf >= 0.3) return 'text-orange-600 dark:text-orange-400'
    return 'text-red-600 dark:text-red-400'
  }

  const getConfidenceLabel = (conf?: number | null) => {
    if (!conf) return null
    if (conf >= 0.9) return 'Отличная классификация'
    if (conf >= 0.75) return 'Хорошая классификация'
    if (conf >= 0.5) return 'Средняя классификация'
    if (conf >= 0.3) return 'Низкая классификация'
    return 'Очень низкая классификация'
  }

  const percentage = confidence ? Math.round(confidence * 100) : null
  const ariaLabel = `КПВЭД код ${code}${name ? `: ${name}` : ''}${percentage ? `, уверенность ${percentage}%` : ''}`

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge variant="secondary" className={className} aria-label={ariaLabel}>
            <span className="font-mono font-semibold">{code}</span>
            {showConfidence && confidence !== null && confidence !== undefined && (
              <span className={`ml-1.5 text-xs ${getConfidenceColor(confidence)}`}>
                ({percentage}%)
              </span>
            )}
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <div className="space-y-2 min-w-[250px]">
            <div>
              <p className="font-semibold text-sm">Код КПВЭД</p>
              <p className="font-mono text-xs text-muted-foreground">{code}</p>
            </div>
            <div>
              <p className="text-sm font-medium">{name || 'Название не найдено'}</p>
            </div>
            {confidence !== null && confidence !== undefined && (
              <div className="space-y-1 pt-1 border-t">
                <div className="flex justify-between items-center text-xs">
                  <span className="text-muted-foreground">Уверенность классификации</span>
                  <span className={`font-medium ${getConfidenceColor(confidence)}`}>
                    {getConfidenceLabel(confidence)}
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <div className="flex-1 bg-secondary rounded-full h-1.5">
                    <div
                      className={`h-1.5 rounded-full transition-all ${
                        confidence >= 0.9 ? 'bg-emerald-500' :
                        confidence >= 0.75 ? 'bg-green-500' :
                        confidence >= 0.5 ? 'bg-amber-500' :
                        confidence >= 0.3 ? 'bg-orange-500' :
                        'bg-red-500'
                      }`}
                      style={{ width: `${percentage}%` }}
                      aria-hidden="true"
                    />
                  </div>
                  <span className="text-xs font-mono font-medium min-w-[3ch]">{percentage}%</span>
                </div>
              </div>
            )}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
