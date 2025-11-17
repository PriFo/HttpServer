import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Временно отключаем standalone для устранения проблем с prerendering
  // output: 'standalone',
  
  // Настройки для работы в контейнере
  // outputFileTracingRoot: require('path').join(__dirname, '../../'),
};

export default nextConfig;
