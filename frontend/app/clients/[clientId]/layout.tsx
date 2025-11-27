import type { Metadata } from 'next'
import { generateMetadata as genMeta } from '@/lib/seo'
import { getBackendUrl } from '@/lib/api-config'

type Props = {
  params: Promise<{ clientId: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { clientId } = await params
  
  // Пытаемся получить имя клиента для более информативного SEO
  let clientName: string | null = null
  try {
    const backendUrl = getBackendUrl()
    const response = await fetch(`${backendUrl}/api/clients/${clientId}`, {
      next: { revalidate: 3600 }, // Кешируем на 1 час для SEO
      signal: AbortSignal.timeout(3000), // Таймаут 3 секунды
    })
    if (response.ok) {
      const data = await response.json()
      clientName = data.client?.name || null
    }
  } catch (error) {
    // Игнорируем ошибки, используем fallback
    console.warn(`Failed to fetch client name for SEO: ${error}`)
  }
  
  const title = clientName ? `Клиент: ${clientName}` : `Клиент #${clientId}`
  const description = clientName 
    ? `Информация о клиенте "${clientName}". Управление проектами, базами данных и процессами нормализации.`
    : `Информация о клиенте #${clientId}. Управление проектами, базами данных и процессами нормализации.`
  
  return genMeta({
    title,
    description,
    keywords: ['клиент', 'проекты', 'нормализация', 'базы данных', 'управление данными'],
    path: `/clients/${clientId}`,
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'Organization',
      name: clientName || `Клиент #${clientId}`,
      description: `Информация о клиенте и его проектах нормализации данных`,
      identifier: clientId,
    },
  })
}

export default function ClientLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return <>{children}</>
}

