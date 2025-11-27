import type { Metadata } from 'next'

const baseUrl = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'
const siteName = 'Нормализатор данных 1С'

export interface PageSEOConfig {
  title: string
  description: string
  keywords?: string[]
  path?: string
  noindex?: boolean
  type?: 'website' | 'article'
  image?: string
  structuredData?: Record<string, any>
  publishedTime?: string
  modifiedTime?: string
  author?: string
  section?: string
}

export function generateMetadata(config: PageSEOConfig): Metadata {
  const {
    title,
    description,
    keywords = [],
    path = '/',
    noindex = false,
    type = 'website',
    image = '/og-image.png',
    structuredData,
    publishedTime,
    modifiedTime,
    author = 'HttpServer Team',
    section,
  } = config

  const fullTitle = `${title} | ${siteName}`
  const url = `${baseUrl}${path}`

  const defaultKeywords = [
    '1С',
    'нормализация данных',
    'номенклатура',
    'контрагенты',
    'унификация',
    'качество данных',
    'обработка данных',
    'справочники 1С',
  ]

  const allKeywords = [...defaultKeywords, ...keywords]

  // Базовые Open Graph настройки
  const openGraphBase = {
    type,
    locale: 'ru_RU',
    url,
    siteName,
    title: fullTitle,
    description,
    images: [
      {
        url: image,
        width: 1200,
        height: 630,
        alt: title,
      },
    ],
  }

  // Дополнительные поля для article типа
  const openGraphArticle = type === 'article' ? {
    publishedTime,
    modifiedTime,
    authors: author ? [author] : undefined,
    section,
  } : {}

  return {
    title: {
      default: fullTitle,
      template: `%s | ${siteName}`,
    },
    description,
    keywords: allKeywords,
    authors: [{ name: author }],
    creator: 'HttpServer',
    publisher: 'HttpServer',
    metadataBase: new URL(baseUrl),
    alternates: {
      canonical: url,
    },
    openGraph: {
      ...openGraphBase,
      ...openGraphArticle,
    },
    twitter: {
      card: 'summary_large_image',
      title: fullTitle,
      description,
      images: [image],
      creator: author,
    },
    robots: {
      index: !noindex,
      follow: !noindex,
      googleBot: {
        index: !noindex,
        follow: !noindex,
        'max-video-preview': -1,
        'max-image-preview': 'large',
        'max-snippet': -1,
      },
    },
  }
}

export function generateStructuredData(
  type: string,
  data: Record<string, any>
): Record<string, any> {
  return {
    '@context': 'https://schema.org',
    '@type': type,
    ...data,
  }
}

// Предустановленные конфигурации для типичных страниц
export const seoConfigs = {
  home: {
    title: 'Главная',
    description:
      'Автоматизированная система для нормализации и унификации справочных данных из 1С. Управление номенклатурой, контрагентами и качеством данных.',
    keywords: ['панель управления', 'дашборд', 'статистика'],
    path: '/',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'SoftwareApplication',
      name: siteName,
      description:
        'Автоматизированная система для нормализации и унификации справочных данных',
      applicationCategory: 'BusinessApplication',
      operatingSystem: 'Web',
      offers: {
        '@type': 'Offer',
        price: '0',
        priceCurrency: 'RUB',
      },
    },
  },
  clients: {
    title: 'Клиенты',
    description:
      'Управление клиентами и проектами. Просмотр статистики, управление базами данных и настройка процессов нормализации для каждого клиента.',
    keywords: ['клиенты', 'проекты', 'управление'],
    path: '/clients',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Управление клиентами',
      description:
        'Управление клиентами и проектами. Просмотр статистики, управление базами данных и настройка процессов нормализации.',
      mainEntity: {
        '@type': 'Organization',
        name: 'Клиенты',
        description: 'Управление клиентами и их проектами',
      },
    },
  },
  processes: {
    title: 'Процессы обработки',
    description:
      'Запуск и мониторинг процессов нормализации и переклассификации данных. Настройка параметров обработки, выбор моделей AI и отслеживание прогресса.',
    keywords: ['нормализация', 'переклассификация', 'обработка данных', 'AI'],
    path: '/processes',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Процессы обработки данных',
      description:
        'Запуск и мониторинг процессов нормализации и переклассификации данных',
      mainEntity: {
        '@type': 'SoftwareApplication',
        name: 'Процессы обработки',
        applicationCategory: 'DataProcessingApplication',
      },
    },
  },
  quality: {
    title: 'Качество данных',
    description:
      'Анализ качества нормализованных данных. Поиск дубликатов, выявление нарушений и получение предложений по улучшению данных.',
    keywords: ['качество данных', 'дубликаты', 'нарушения', 'предложения'],
    path: '/quality',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Качество данных',
      description:
        'Анализ качества нормализованных данных. Поиск дубликатов, выявление нарушений и получение предложений по улучшению данных.',
      mainEntity: {
        '@type': 'DataCatalog',
        name: 'Качество данных',
        description: 'Анализ и контроль качества данных',
      },
    },
  },
  results: {
    title: 'Результаты нормализации',
    description:
      'Просмотр результатов нормализации данных. Группировка по категориям, фильтрация и экспорт нормализованных данных.',
    keywords: ['результаты', 'нормализация', 'группы данных'],
    path: '/results',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Результаты нормализации',
      description:
        'Просмотр результатов нормализации данных. Группировка по категориям, фильтрация и экспорт нормализованных данных.',
      mainEntity: {
        '@type': 'Dataset',
        name: 'Нормализованные данные',
        description: 'Результаты нормализации справочных данных',
      },
    },
  },
  databases: {
    title: 'Базы данных',
    description:
      'Управление базами данных. Просмотр списка баз, переключение между базами, загрузка новых баз и просмотр ожидающих обработки баз.',
    keywords: ['базы данных', 'SQLite', 'управление БД'],
    path: '/databases',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Управление базами данных',
      description:
        'Управление базами данных для нормализации. Просмотр списка баз, переключение между базами, загрузка новых баз.',
      mainEntity: {
        '@type': 'Database',
        name: 'Базы данных нормализации',
        description: 'Управление базами данных системы нормализации',
      },
    },
  },
  classifiers: {
    title: 'Классификаторы',
    description:
      'Просмотр и управление классификаторами КПВЭД и ОКПД2. Поиск кодов, просмотр иерархии и статистики использования.',
    keywords: ['КПВЭД', 'ОКПД2', 'классификаторы', 'коды'],
    path: '/classifiers',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Классификаторы КПВЭД и ОКПД2',
      description:
        'Просмотр и управление классификаторами КПВЭД и ОКПД2. Поиск кодов, просмотр иерархии и статистики использования.',
      mainEntity: {
        '@type': 'DataCatalog',
        name: 'Классификаторы',
        description: 'Классификаторы КПВЭД и ОКПД2 для категоризации товаров',
      },
    },
  },
  gosts: {
    title: 'ГОСТы',
    description:
      'Просмотр и поиск ГОСТов из 50 источников Росстандарта. Национальные и межгосударственные стандарты. Импорт и управление базой ГОСТов.',
    keywords: ['ГОСТ', 'стандарты', 'Росстандарт', 'нормативные документы', 'национальные стандарты'],
    path: '/gosts',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'CollectionPage',
      name: 'ГОСТы',
      description: 'База данных ГОСТов из открытых данных Росстандарта. 50 источников данных.',
      mainEntity: {
        '@type': 'DataCatalog',
        name: 'ГОСТы',
        description: 'База данных российских ГОСТов из открытых данных Росстандарта',
        keywords: 'ГОСТ, стандарты, Росстандарт, нормативные документы',
      },
    },
  },
  monitoring: {
    title: 'Мониторинг системы',
    description:
      'Мониторинг работы системы. Просмотр метрик, событий, истории обработки и состояния воркеров.',
    keywords: ['мониторинг', 'метрики', 'события', 'логи'],
    path: '/monitoring',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Мониторинг системы нормализации',
      description:
        'Реальные метрики работы системы нормализации данных. Производительность, качество обработки, статистика AI и кеширования.',
      mainEntity: {
        '@type': 'MonitoringService',
        name: 'Мониторинг производительности',
        description: 'Система мониторинга метрик нормализации данных',
        serviceType: 'PerformanceMonitoring',
      },
    },
  },
  workers: {
    title: 'Воркеры',
    description:
      'Управление воркерами обработки данных. Настройка конфигурации, просмотр статуса провайдеров AI и управление моделями.',
    keywords: ['воркеры', 'AI', 'модели', 'провайдеры'],
    path: '/workers',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Управление воркерами',
      description:
        'Управление воркерами обработки данных. Настройка конфигурации, просмотр статуса провайдеров AI и управление моделями.',
      mainEntity: {
        '@type': 'SoftwareApplication',
        name: 'Воркеры обработки',
        applicationCategory: 'DataProcessingApplication',
      },
    },
  },
  pipelineStages: {
    title: 'Этапы обработки',
    description:
      'Просмотр этапов обработки данных в пайплайне. Статистика по каждому этапу, визуализация воронки обработки.',
    keywords: ['пайплайн', 'этапы обработки', 'воронка'],
    path: '/pipeline-stages',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Этапы обработки данных',
      description:
        'Просмотр этапов обработки данных в пайплайне. Статистика по каждому этапу, визуализация воронки обработки.',
      mainEntity: {
        '@type': 'DataProcessingPipeline',
        name: 'Пайплайн обработки',
        description: 'Этапы обработки данных в системе нормализации',
      },
    },
  },
  benchmark: {
    title: 'Бенчмарк моделей',
    description:
      'Сравнение производительности различных моделей AI для нормализации данных. Метрики качества и скорости обработки.',
    keywords: ['бенчмарк', 'модели AI', 'производительность'],
    path: '/models/benchmark',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Бенчмарк моделей AI',
      description:
        'Сравнение производительности различных моделей AI для нормализации данных',
      mainEntity: {
        '@type': 'BenchmarkTest',
        name: 'Бенчмарк моделей',
        description: 'Тестирование производительности AI моделей',
      },
    },
  },
  pendingDatabases: {
    title: 'Ожидающие базы данных',
    description:
      'Базы данных, ожидающие обработки. Управление очередью загрузки и обработки баз данных.',
    keywords: ['ожидающие БД', 'очередь', 'загрузка баз данных'],
    path: '/databases/pending',
  },
  monitoringHistory: {
    title: 'История мониторинга',
    description:
      'Исторические данные мониторинга системы. Графики метрик производительности, AI статистики и кеширования за различные периоды времени.',
    keywords: ['история мониторинга', 'графики метрик', 'аналитика'],
    path: '/monitoring/history',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'История мониторинга',
      description:
        'Исторические данные и графики метрик производительности системы нормализации',
      mainEntity: {
        '@type': 'DataVisualization',
        name: 'История метрик',
        description: 'Визуализация исторических данных мониторинга',
      },
    },
  },
  reports: {
    title: 'Отчеты',
    description:
      'Генерация комплексных PDF-отчетов по нормализации и качеству данных. Детальная статистика обработки, анализ качества и рекомендации по улучшению.',
    keywords: ['отчеты', 'PDF', 'качество данных', 'статистика нормализации'],
    path: '/reports',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Отчеты по нормализации и качеству данных',
      description:
        'Генерация комплексных PDF-отчетов по нормализации и качеству данных',
      mainEntity: {
        '@type': 'Report',
        name: 'Отчеты системы нормализации',
        description: 'Отчеты о нормализации и качестве данных',
      },
    },
  },
  processesNomenclature: {
    title: 'Процессы нормализации номенклатуры',
    description:
      'Управление процессами нормализации номенклатуры и переклассификации. Запуск нормализации, мониторинг прогресса, настройка AI-провайдеров (OpenRouter, Hugging Face, Arliai, Eden AI, DaData, Adata.kz).',
    keywords: ['нормализация номенклатуры', 'процессы обработки', 'AI провайдеры', 'переклассификация'],
    path: '/processes/nomenclature',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Процессы нормализации номенклатуры',
      description:
        'Управление процессами нормализации номенклатуры с использованием мульти-провайдерной архитектуры AI',
      mainEntity: {
        '@type': 'SoftwareApplication',
        name: 'Нормализация номенклатуры',
        applicationCategory: 'DataProcessingApplication',
        featureList: [
          'Нормализация названий товаров',
          'Мульти-провайдерная обработка (DaData, Adata.kz, OpenRouter, Hugging Face, Arliai, Eden AI)',
          'Переклассификация по КПВЭД и ОКПД2',
        ],
      },
    },
  },
  processesCounterparties: {
    title: 'Процессы нормализации контрагентов',
    description:
      'Управление процессами нормализации контрагентов. Запуск нормализации, мониторинг прогресса, использование AI-провайдеров (OpenRouter, Hugging Face, Arliai, Eden AI, DaData, Adata.kz) для обогащения данных.',
    keywords: ['нормализация контрагентов', 'процессы обработки', 'AI провайдеры', 'обогащение данных'],
    path: '/processes/counterparties',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Процессы нормализации контрагентов',
      description:
        'Управление процессами нормализации контрагентов с использованием мульти-провайдерной архитектуры AI',
      mainEntity: {
        '@type': 'SoftwareApplication',
        name: 'Нормализация контрагентов',
        applicationCategory: 'DataProcessingApplication',
        featureList: [
          'Нормализация названий организаций',
          'Мульти-провайдерная обработка (DaData, Adata.kz, OpenRouter, Hugging Face, Arliai, Eden AI)',
          'Обогащение данных контрагентов',
        ],
      },
    },
  },
  about: {
    title: 'Нормализатор данных 1С: Технологии, ML модели и AI алгоритмы',
    description:
      'Калибруемые ML модели, 50+ классификаторов ГОСТ и топовые AI алгоритмы для нормализации данных 1С. Платформа с Next.js 16 и Go бэкенд.',
    keywords: [
      'ML модели',
      'AI нормализация',
      'классификаторы ГОСТ',
      'Next.js 16',
      'Go бэкенд',
      'GLM-4.5',
      'калибруемые модели',
      'нормализация номенклатуры',
      '1С интеграция',
      'shadcn',
      'TypeScript',
      '50+ классификаторов',
      'OpenRouter',
      'Arliai',
      'Hugging Face',
      'машинное обучение',
      'нейронные сети',
      'обработка естественного языка',
      'унификация данных',
      'стандартизация',
      'категоризация',
      'ERP системы',
      'CRM системы',
      'верификация данных',
      'обогащение данных',
      'дедупликация',
    ],
    path: '/about',
    image: '/og-about-image.jpg',
    type: 'article' as const,
    publishedTime: '2024-01-01T00:00:00Z',
    modifiedTime: new Date().toISOString(),
    author: 'HttpServer Team',
    section: 'Технологии',
    structuredData: {
      '@context': 'https://schema.org',
      '@type': 'WebPage',
      name: 'Технологии Нормализатора данных 1С',
      description:
        'Передовые технологии для автоматической нормализации данных: калибруемые ML модели, 50+ классификаторов ГОСТ, топовые AI алгоритмы',
      mainEntity: {
        '@type': 'SoftwareApplication',
        name: 'Нормализатор данных 1С',
        applicationCategory: 'BusinessApplication',
        operatingSystem: 'Web',
        programmingLanguage: ['TypeScript', 'Go', 'JavaScript'],
        featureList: [
          'Калибруемые ML модели под каждого клиента',
          'Интеграция с 50+ классификаторами ГОСТ',
          'Топовые AI модели (GLM-4.5, OpenRouter, Arliai)',
          'Мульти-провайдерная архитектура',
          'Real-time мониторинг процессов',
        ],
      },
    },
  },
}

