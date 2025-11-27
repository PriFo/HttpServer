import { Badge } from "@/components/ui/badge"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface ProcessingLevelBadgeProps {
  level?: string
  showTooltip?: boolean
}

export function ProcessingLevelBadge({ level, showTooltip = true }: ProcessingLevelBadgeProps) {
  if (!level) {
    return <Badge variant="outline">N/A</Badge>
  }

  const levelConfig = {
    basic: {
      label: "База",
      variant: "secondary" as const,
      description: "Базовая нормализация без AI",
      color: "bg-gray-500"
    },
    ai_enhanced: {
      label: "AI",
      variant: "default" as const,
      description: "Обработано с использованием AI",
      color: "bg-blue-500"
    },
    benchmark: {
      label: "Эталон",
      variant: "default" as const,
      description: "Эталонное качество данных",
      color: "bg-purple-500"
    }
  }

  const config = levelConfig[level as keyof typeof levelConfig] || {
    label: level,
    variant: "outline" as const,
    description: "Неизвестный уровень обработки",
    color: "bg-gray-400"
  }

  const badgeContent = (
    <Badge variant={config.variant} className="text-xs">
      <div className="flex items-center gap-1">
        <div className={`w-2 h-2 rounded-full ${config.color}`} />
        {config.label}
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
          <p className="text-sm">{config.description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
