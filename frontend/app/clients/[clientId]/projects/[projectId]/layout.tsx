import type { Metadata } from 'next'
import { generateMetadata as genMeta } from '@/lib/seo'
import { getBackendUrl } from '@/lib/api-config'

type Props = {
  params: Promise<{ clientId: string; projectId: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { clientId, projectId } = await params
  
  // Пытаемся получить имя проекта для более информативного SEO
  let projectName: string | null = null
  let clientName: string | null = null
  try {
    const backendUrl = getBackendUrl()
    const response = await fetch(`${backendUrl}/api/clients/${clientId}/projects/${projectId}`, {
      next: { revalidate: 3600 }, // Кешируем на 1 час для SEO
      signal: AbortSignal.timeout(3000), // Таймаут 3 секунды
    })
    if (response.ok) {
      const data = await response.json()
      projectName = data.project?.name || null
      clientName = data.client?.name || null
    }
  } catch (error) {
    // Игнорируем ошибки, используем fallback
    console.warn(`Failed to fetch project name for SEO: ${error}`)
  }
  
  const title = projectName 
    ? `Проект: ${projectName}${clientName ? ` (${clientName})` : ''}`
    : `Проект #${projectId}`
  const description = projectName
    ? `Информация о проекте "${projectName}"${clientName ? ` клиента "${clientName}"` : ''}. Управление нормализацией, базами данных и статистикой.`
    : `Информация о проекте #${projectId} клиента #${clientId}. Управление нормализацией, базами данных и статистикой.`
  
  return genMeta({
    title,
    description,
    keywords: ['проект', 'нормализация', 'базы данных', 'статистика', 'качество данных'],
    path: `/clients/${clientId}/projects/${projectId}`,
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'Project',
      name: projectName || `Проект #${projectId}`,
      description: `Информация о проекте нормализации данных`,
      identifier: projectId,
      ...(clientName && {
        parentOrganization: {
          '@type': 'Organization',
          name: clientName,
        },
      }),
    },
  })
}

export default function ProjectLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return <>{children}</>
}

