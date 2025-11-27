import type { Metadata } from "next"
import { generateMetadata as genMeta, seoConfigs } from "@/lib/seo"

export const metadata: Metadata = genMeta(seoConfigs.reports)

export default function ReportsLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const structuredData = seoConfigs.reports.structuredData

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

