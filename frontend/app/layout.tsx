import type { Metadata } from "next";
import { Geist, Geist_Mono } from "next/font/google";
import "./globals.css";
import { ThemeProvider } from "@/components/providers/theme-provider";
import { AnimationProvider } from "@/providers/animation-provider";
import { Header } from "@/components/layout/header";
import { Footer } from "@/components/layout/footer";
import { Toaster } from "sonner";
import { ConsoleInterceptorProvider } from "@/components/console-interceptor-provider";
import { ErrorProviderWrapper } from "@/components/providers/error-provider-wrapper";

const geistSans = Geist({
  variable: "--font-geist-sans",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

import { generateMetadata as genMeta, seoConfigs } from "@/lib/seo"

export const metadata: Metadata = genMeta(seoConfigs.home)

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const structuredData = seoConfigs.home.structuredData || {
    "@context": "https://schema.org",
    "@type": "SoftwareApplication",
    name: "Нормализатор",
    description: "Автоматизированная система для нормализации и унификации справочных данных",
    applicationCategory: "BusinessApplication",
    operatingSystem: "Web",
    offers: {
      "@type": "Offer",
      price: "0",
      priceCurrency: "RUB",
    },
  };

  return (
    <html lang="ru" suppressHydrationWarning>
      <head>
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: JSON.stringify(structuredData) }}
        />
      </head>
      <body
        className={`${geistSans.variable} ${geistMono.variable} antialiased`}
      >
        <ThemeProvider
          attribute="class"
          defaultTheme="system"
          enableSystem
          disableTransitionOnChange
        >
          <AnimationProvider defaultVariant="gentle">
            <ErrorProviderWrapper>
              <ConsoleInterceptorProvider />
              <div className="relative flex min-h-screen flex-col">
                <div id="main-header">
                  <Header />
                </div>
                <main className="flex-1">{children}</main>
                <div id="main-footer">
                  <Footer />
                </div>
              </div>
              <Toaster position="bottom-right" richColors closeButton />
            </ErrorProviderWrapper>
          </AnimationProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
