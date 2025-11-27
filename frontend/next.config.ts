import type { NextConfig } from "next";

// Bundle Analyzer
const withBundleAnalyzer = require('@next/bundle-analyzer')({
  enabled: process.env.ANALYZE === 'true',
})

type ExtendedNextConfig = NextConfig & {
  turbopack?: {
    root?: string
  }
}

const nextConfig: ExtendedNextConfig = {
  turbopack: {
    root: __dirname,
  },

  // Временно отключаем standalone для устранения проблем с prerendering
  // output: 'standalone',
  
  // Настройки для работы в контейнере
  // outputFileTracingRoot: require('path').join(__dirname, '../../'),
  
  // Включаем строгий режим React для лучшей работы с DevTools
  reactStrictMode: true,
  
  // Оптимизация импортов для уменьшения размера бандла
  experimental: {
    optimizePackageImports: ['lucide-react', '@radix-ui/react-icons'],
    // Временно отключаем Turbopack для устранения проблем с proxy
    // turbo: false,
  },
  
  // Отключаем генерацию статических страниц в dev режиме
  output: undefined,
};

export default withBundleAnalyzer(nextConfig);
