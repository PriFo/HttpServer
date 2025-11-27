import type { Metadata } from 'next'
import { generateMetadata as genMeta, seoConfigs } from '@/lib/seo'

export const metadata: Metadata = genMeta(seoConfigs.gosts)

export default function GostsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const structuredData = seoConfigs.gosts.structuredData

  return (
    <>
      {structuredData && (
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
      )}
      {children}
    </>
  )
}

