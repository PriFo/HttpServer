import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface ConfidenceBadgeProps {
  confidence?: number
  showTooltip?: boolean
  size?: 'sm' | 'md' | 'lg'
}

export function ConfidenceBadge({
  confidence,
  showTooltip = true,
  size = 'md'
}: ConfidenceBadgeProps) {
  if (confidence === undefined || confidence === null) {
    return <Badge variant="outline">N/A</Badge>
  }

  const percentage = Math.round(confidence * 100)

  // Определяем цвет и вариант бейджа с улучшенными порогами
  const getVariant = (conf: number) => {
    if (conf >= 0.9) return "default"      // Excellent
    if (conf >= 0.75) return "default"     // Good
    if (conf >= 0.5) return "secondary"    // Fair
    if (conf >= 0.3) return "secondary"    // Poor
    return "destructive"                   // Very Low
  }

  const getColorClass = (conf: number) => {
    if (conf >= 0.9) return "text-emerald-600 dark:text-emerald-400"
    if (conf >= 0.75) return "text-green-600 dark:text-green-400"
    if (conf >= 0.5) return "text-amber-600 dark:text-amber-400"
    if (conf >= 0.3) return "text-orange-600 dark:text-orange-400"
    return "text-red-600 dark:text-red-400"
  }

  const getLabel = (conf: number) => {
    if (conf >= 0.9) return "Отличная уверенность"
    if (conf >= 0.75) return "Хорошая уверенность"
    if (conf >= 0.5) return "Средняя уверенность"
    if (conf >= 0.3) return "Низкая уверенность"
    return "Очень низкая уверенность"
  }

  const getDescription = (conf: number) => {
    if (conf >= 0.9) return "AI очень уверен в классификации"
    if (conf >= 0.75) return "AI уверен в классификации"
    if (conf >= 0.5) return "AI умеренно уверен, возможны неточности"
    if (conf >= 0.3) return "AI слабо уверен, требуется проверка"
    return "AI не уверен, требуется ручная проверка"
  }

  const badgeContent = (
    <Badge variant={getVariant(confidence)} className={size === 'sm' ? 'text-xs' : size === 'lg' ? 'text-base' : ''}>
      <div className={`flex items-center gap-1 ${getColorClass(confidence)}`}>
        <div className="w-2 h-2 rounded-full bg-current" />
        {percentage}%
      </div>
    </Badge>
  )

  if (!showTooltip) {
    return badgeContent
  }

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          {badgeContent}
        </TooltipTrigger>
        <TooltipContent>
          <div className="space-y-2 min-w-[200px]">
            <p className="font-medium">{getLabel(confidence)}</p>
            <p className="text-xs text-muted-foreground">
              {getDescription(confidence)}
            </p>
            <div className="space-y-1">
              <div className="flex justify-between text-xs">
                <span>AI уверенность:</span>
                <span className="font-mono font-medium">{percentage}%</span>
              </div>
              <div className="w-full bg-secondary rounded-full h-1.5">
                <div
                  className={`h-1.5 rounded-full transition-all ${
                    confidence >= 0.9 ? 'bg-emerald-500' :
                    confidence >= 0.75 ? 'bg-green-500' :
                    confidence >= 0.5 ? 'bg-amber-500' :
                    confidence >= 0.3 ? 'bg-orange-500' :
                    'bg-red-500'
                  }`}
                  style={{ width: `${percentage}%` }}
                />
              </div>
            </div>
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
