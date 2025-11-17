import { Separator } from '@/components/ui/separator'

export function Footer() {
  const currentYear = new Date().getFullYear()

  return (
    <footer className="border-t bg-background">
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col items-center justify-between gap-4 py-6 md:flex-row">
          <div className="text-center text-sm text-muted-foreground md:text-left">
            <p>
              Нормализатор 1С{' '}
              <span className="font-semibold">v1.0</span>
            </p>
            <p className="mt-1">
              © {currentYear} Отдел разработки. Все права защищены.
            </p>
          </div>

          <div className="flex items-center gap-4 text-sm text-muted-foreground">
            <a
              href="#"
              className="hover:text-foreground transition-colors"
            >
              Документация
            </a>
            <Separator orientation="vertical" className="h-4" />
            <a
              href="#"
              className="hover:text-foreground transition-colors"
            >
              Поддержка
            </a>
            <Separator orientation="vertical" className="h-4" />
            <a
              href="#"
              className="hover:text-foreground transition-colors"
            >
              О проекте
            </a>
          </div>
        </div>
      </div>
    </footer>
  )
}
