'use client'

import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Home, ArrowLeft, Search } from 'lucide-react'
import { EmptyState } from '@/components/common/empty-state'

export default function NotFound() {
  const router = useRouter()

  return (
    <div className="container-wide mx-auto px-4 py-16">
      <div className="max-w-2xl mx-auto text-center">
        <EmptyState
          icon={Search}
          title="404 - Страница не найдена"
          description="Запрашиваемая страница не существует или была перемещена. Проверьте правильность URL или вернитесь на главную страницу."
          action={{
            label: 'На главную',
            onClick: () => router.push('/'),
          }}
        />
        
        <div className="mt-8 flex flex-col sm:flex-row gap-4 justify-center">
          <Button
            variant="outline"
            onClick={() => router.back()}
            className="flex items-center gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Назад
          </Button>
          <Button
            onClick={() => router.push('/')}
            className="flex items-center gap-2"
          >
            <Home className="h-4 w-4" />
            На главную
          </Button>
        </div>

        <div className="mt-12 text-sm text-muted-foreground">
          <p className="mb-2">Возможные причины:</p>
          <ul className="list-disc list-inside space-y-1 text-left max-w-md mx-auto">
            <li>Страница была удалена или перемещена</li>
            <li>Неправильный URL адрес</li>
            <li>Страница находится в разработке</li>
          </ul>
        </div>
      </div>
    </div>
  )
}

