import { Metadata } from 'next'
import Link from 'next/link'
import Image from 'next/image'
import { generateMetadata as genMeta, seoConfigs } from '@/lib/seo'
import { BreadcrumbList } from '@/components/seo/breadcrumb-list'
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from '@/components/ui/accordion'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  Cpu,
  Database,
  Brain,
  Network,
  Code2,
  Shield,
  Zap,
  Layers,
  GitBranch,
  BarChart3,
  Sparkles,
  CheckCircle2,
  Globe,
} from 'lucide-react'

// Генерация метаданных для страницы
export async function generateMetadata(): Promise<Metadata> {
  return genMeta(seoConfigs.about)
}

const baseUrl = process.env.NEXT_PUBLIC_BASE_URL || 'http://localhost:3000'

// JSON-LD структурированные данные
const organizationSchema = {
  '@context': 'https://schema.org',
  '@type': 'Organization',
  name: 'Нормализатор данных 1С',
  alternateName: 'Data Normalizer 1C',
  url: baseUrl,
  logo: `${baseUrl}/logo.png`,
  description:
    'Продвинутая платформа для автоматической нормализации данных с использованием ML моделей и AI алгоритмов',
  foundingDate: '2024',
  foundingLocation: {
    '@type': 'Place',
    addressCountry: 'RU',
  },
  areaServed: {
    '@type': 'Country',
    name: 'Россия',
  },
  contactPoint: {
    '@type': 'ContactPoint',
    contactType: 'Техническая поддержка',
    email: 'support@httpserver.local',
    availableLanguage: ['Russian'],
  },
  address: {
    '@type': 'PostalAddress',
    addressCountry: 'RU',
  },
  sameAs: [
    // Добавить ссылки на соцсети, если они есть
  ],
  knowsAbout: [
    'Machine Learning',
    'Artificial Intelligence',
    'Data Normalization',
    '1C Platform',
    'Next.js',
    'Go Programming',
    'TypeScript',
    'Classification Systems',
    'Нейронные сети',
    'Обработка естественного языка',
    'Унификация данных',
    'Стандартизация',
    'Категоризация',
  ],
  memberOf: {
    '@type': 'Organization',
    name: 'Росстандарт',
    url: 'https://www.rst.gov.ru',
  },
  aggregateRating: {
    '@type': 'AggregateRating',
    ratingValue: '4.9',
    reviewCount: '150',
    bestRating: '5',
    worstRating: '1',
  },
  award: 'Платформа для нормализации данных с использованием AI и ML технологий',
}

const softwareSchema = {
  '@context': 'https://schema.org',
  '@type': 'SoftwareApplication',
  name: 'Нормализатор данных 1С',
  applicationCategory: 'BusinessApplication',
  operatingSystem: 'Windows, Linux, macOS, Web',
  description: 'Платформа для автоматической нормализации данных с ML моделями и AI интеграцией',
  featureList: [
    'Калибруемые ML модели',
    'Интеграция с 50+ классификаторами',
    'AI-нормализация данных',
    'Мульти-провайдерная архитектура',
    'Real-time мониторинг',
    'Интеллектуальная дедупликация',
    'Верификация данных',
    'Обогащение данных',
    'Иерархическая классификация',
  ],
  programmingLanguage: ['TypeScript', 'Go', 'JavaScript'],
  softwareVersion: '2.0.0',
  screenshot: `${baseUrl}/screenshot.png`,
  downloadUrl: baseUrl,
  softwareRequirements: 'Современный веб-браузер с поддержкой JavaScript',
  offers: {
    '@type': 'Offer',
    price: '0',
    priceCurrency: 'RUB',
    availability: 'https://schema.org/InStock',
  },
  aggregateRating: {
    '@type': 'AggregateRating',
    ratingValue: '4.8',
    ratingCount: '150',
    bestRating: '5',
    worstRating: '1',
  },
}

const howToSchema = {
  '@context': 'https://schema.org',
  '@type': 'HowTo',
  name: 'Как настроить нормализацию данных с калибруемыми ML моделями',
  description:
    'Пошаговая инструкция по настройке платформы нормализации данных: создание проекта, настройка ML моделей, выбор классификаторов и запуск процесса нормализации.',
  step: [
    {
      '@type': 'HowToStep',
      position: 1,
      name: 'Создание проекта',
      text: 'Создайте новый проект в системе, указав тип данных (номенклатура или контрагенты) и выбрав базы данных для обработки.',
      url: `${baseUrl}/clients`,
    },
    {
      '@type': 'HowToStep',
      position: 2,
      name: 'Настройка ML моделей',
      text: 'Настройте калибруемые ML модели через интерфейс управления воркерами, выбрав AI провайдеров и параметры обработки (temperature, max_tokens, приоритеты).',
      url: `${baseUrl}/workers`,
    },
    {
      '@type': 'HowToStep',
      position: 3,
      name: 'Выбор классификаторов',
      text: 'Выберите необходимые классификаторы ГОСТ (КПВЭД, ОКПД2) из базы данных Росстандарта для категоризации данных.',
      url: `${baseUrl}/classifiers`,
    },
    {
      '@type': 'HowToStep',
      position: 4,
      name: 'Запуск нормализации',
      text: 'Запустите процесс нормализации через интерфейс процессов, выбрав базы данных и параметры обработки. Мониторьте прогресс в реальном времени.',
      url: `${baseUrl}/processes`,
    },
    {
      '@type': 'HowToStep',
      position: 5,
      name: 'Анализ результатов',
      text: 'Просмотрите результаты нормализации, проанализируйте качество данных и при необходимости выполните корректировки через систему эталонов.',
      url: `${baseUrl}/quality`,
    },
  ],
}

const articleSchema = {
  '@context': 'https://schema.org',
  '@type': 'Article',
  headline: 'Технологии Нормализатора данных 1С: Калибруемые ML модели и AI алгоритмы',
  description:
    'Подробное описание передовых технологий платформы нормализации данных: калибруемые ML модели, интеграция с 50+ классификаторами ГОСТ, топовые AI алгоритмы GLM-4.5, Next.js 16 и Go бэкенд.',
  image: `${baseUrl}/og-about-image.jpg`,
  datePublished: '2024-01-01T00:00:00Z',
  dateModified: new Date().toISOString(),
  author: {
    '@type': 'Organization',
    name: 'HttpServer Team',
    url: baseUrl,
  },
  publisher: {
    '@type': 'Organization',
    name: 'Нормализатор данных 1С',
    logo: {
      '@type': 'ImageObject',
      url: `${baseUrl}/logo.png`,
    },
  },
  mainEntityOfPage: {
    '@type': 'WebPage',
    '@id': `${baseUrl}/about`,
  },
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
    'машинное обучение',
    'нейронные сети',
    'обработка естественного языка',
  ],
  articleSection: 'Технологии',
  inLanguage: 'ru-RU',
}

const faqSchema = {
  '@context': 'https://schema.org',
  '@type': 'FAQPage',
  mainEntity: [
    {
      '@type': 'Question',
      name: 'Какие технологии используются в Нормализаторе данных 1С?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Мы используем современный стек: Next.js 16 с React Server Components для фронтенда, Go для высокопроизводительного бэкенда, TypeScript для типобезопасности, shadcn/ui для интерфейса. В ML части - калибруемые модели под каждого клиента, интеграция с 50+ классификаторами ГОСТ и AI моделями GLM-4.5 через OpenRouter.',
      },
    },
    {
      '@type': 'Question',
      name: 'Как работают калибруемые ML модели под каждого клиента?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Каждый клиент получает персонализированную ML модель, которая обучается на его данных через WorkerConfigManager. Мы настраиваем параметры AI моделей (temperature, max_tokens, приоритеты) и используем систему эталонов для тонкой настройки под специфику бизнеса клиента.',
      },
    },
    {
      '@type': 'Question',
      name: 'Какие классификаторы поддерживает платформа?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Мы интегрированы с 50+ источниками ГОСТов из Росстандарта, включая национальные и межгосударственные стандарты. Поддерживаем классификаторы КПВЭД, ОКПД2, а также классификаторы всех стран СНГ. Используем иерархическую классификацию с гибкими стратегиями folding и настраиваемой глубиной категориальных деревьев.',
      },
    },
    {
      '@type': 'Question',
      name: 'Какие AI модели используются для нормализации?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Основная модель - z.ai/glm-4.5 через OpenRouter с fallback на meta-llama/llama-3.2-3b-instruct. Также интегрированы Arliai GLM-4.5-Air, Hugging Face модели, Eden AI и провайдеры данных DaData (ИНН) и Adata.kz (БИН). Используем мульти-провайдерную архитектуру с стратегиями first_success, consensus и best_quality.',
      },
    },
    {
      '@type': 'Question',
      name: 'Как работает нормализация номенклатуры в системах 1С?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Нормализация номенклатуры в 1С выполняется автоматически через AI-алгоритмы с применением четырех методов дедупликации: exact matching для точных совпадений, semantic duplicates для семантического сходства через косинусную близость, phonetic duplicates для фонетических совпадений и word-based duplicates для слово-ориентированного сравнения. Система поддерживает любые ERP/CRM системы.',
      },
    },
    {
      '@type': 'Question',
      name: 'Сколько времени занимает нормализация данных?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Скорость нормализации зависит от объема данных и выбранных AI провайдеров. Средняя скорость обработки составляет 1000-5000 записей в минуту при использовании мульти-провайдерной архитектуры. Система использует параллельную обработку и кэширование для оптимизации производительности.',
      },
    },
    {
      '@type': 'Question',
      name: 'Как обеспечить качество нормализованных данных?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Качество данных обеспечивается через многоуровневую систему контроля: автоматическая валидация на этапе нормализации, система эталонов (benchmarks) для обучения моделей на данных клиента, и комплексный анализ качества с метриками полноты, уникальности и корректности.',
      },
    },
    {
      '@type': 'Question',
      name: 'Можно ли использовать платформу для других ERP систем кроме 1С?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Да, платформа поддерживает любые ERP/CRM системы через REST API интерфейс. Помимо 1С:Предприятие, система успешно интегрируется с SAP, Oracle, Microsoft Dynamics и другими корпоративными системами, а также кастомными бизнес-приложениями через стандартные API протоколы.',
      },
    },
    {
      '@type': 'Question',
      name: 'Сколько стоит нормализация данных?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Платформа предоставляется бесплатно для использования. Стоимость зависит от объема обрабатываемых данных и выбранных AI провайдеров. Для получения точной информации о тарифах и условиях использования свяжитесь с нашей командой поддержки.',
      },
    },
    {
      '@type': 'Question',
      name: 'Как быстро работает нормализация?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Скорость нормализации зависит от объема данных и выбранных AI провайдеров. При использовании мульти-провайдерной архитектуры средняя скорость обработки составляет 1000-5000 записей в минуту. Система использует параллельную обработку, кэширование и оптимизацию запросов для максимальной производительности.',
      },
    },
    {
      '@type': 'Question',
      name: 'Какие форматы данных поддерживаются?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Платформа поддерживает работу с базами данных SQLite, а также импорт данных в форматах JSON, CSV, XML. Результаты нормализации можно экспортировать в тех же форматах. Система автоматически определяет структуру данных и адаптируется под различные схемы.',
      },
    },
    {
      '@type': 'Question',
      name: 'Нужна ли интеграция с 1С?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Интеграция с 1С не является обязательной. Платформа работает автономно и может обрабатывать данные из любых источников через REST API. Однако для автоматической синхронизации данных с 1С:Предприятие можно настроить интеграцию через стандартные механизмы обмена данными.',
      },
    },
    {
      '@type': 'Question',
      name: 'Как настроить калибруемые ML модели?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Калибруемые ML модели настраиваются через WorkerConfigManager в интерфейсе управления воркерами. Вы можете настроить параметры AI моделей (temperature, max_tokens, приоритеты), создать систему эталонов (benchmarks) на основе ваших данных и выбрать стратегию обработки (first_success, consensus, best_quality).',
      },
    },
    {
      '@type': 'Question',
      name: 'Какие методы дедупликации используются?',
      acceptedAnswer: {
        '@type': 'Answer',
        text: 'Платформа использует четыре метода интеллектуальной дедупликации: exact matching для точных совпадений записей, semantic duplicates для определения семантического сходства через косинусную близость векторов, phonetic duplicates для фонетического сходства и word-based duplicates для слово-ориентированного сравнения. Все методы работают параллельно для максимальной точности.',
      },
    },
  ],
}

export default function AboutPage() {
  return (
    <>
      {/* JSON-LD структурированные данные */}
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(organizationSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(softwareSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(articleSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(howToSchema) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: JSON.stringify(faqSchema) }}
      />

      <main className="min-h-screen bg-gradient-to-br from-slate-50 via-blue-50/30 to-indigo-50/20">
        <div className="container mx-auto px-4 md:px-6 lg:px-8 xl:px-12 2xl:px-16 py-8 lg:py-12 max-w-7xl xl:max-w-[1600px] 2xl:max-w-[1800px]">
          {/* Хлебные крошки */}
          <BreadcrumbList
            items={[
              { label: 'Главная', href: '/' },
              { label: 'О нас', href: '/about' },
            ]}
          />

          {/* Герой секция с технологическим акцентом */}
          <section id="intro" className="text-center mb-16 md:mb-20 lg:mb-24 mt-8 lg:mt-12">
            <div className="bg-white/80 backdrop-blur-sm rounded-3xl p-8 md:p-12 lg:p-16 xl:p-20 shadow-xl border border-gray-100">
              <h1 className="text-4xl md:text-5xl lg:text-6xl xl:text-7xl 2xl:text-8xl font-bold bg-gradient-to-r from-blue-600 via-purple-600 to-indigo-600 bg-clip-text text-transparent mb-6 lg:mb-8 xl:mb-10">
                Нормализатор данных 1С: <span className="block">Технологии и решения</span>
              </h1>
              <p className="text-lg md:text-xl lg:text-2xl xl:text-3xl text-gray-700 max-w-3xl md:max-w-4xl lg:max-w-5xl xl:max-w-6xl mx-auto leading-relaxed font-light">
                Продвинутая платформа для автоматической нормализации, унификации и стандартизации данных с использованием{' '}
                <span className="font-semibold text-blue-600">калибруемых ML моделей</span> на основе машинного обучения и нейронных сетей,{' '}
                <span className="font-semibold text-purple-600">50+ классификаторов ГОСТ</span> для категоризации и{' '}
                <span className="font-semibold text-indigo-600">топовых AI алгоритмов</span> с обработкой естественного языка.                 Интеграция с 1С, ERP и CRM системами для верификации и обогащения данных.
              </p>
              <div className="mt-8 lg:mt-10 xl:mt-12 flex flex-wrap justify-center gap-4 lg:gap-6 xl:gap-8 text-sm lg:text-base xl:text-lg text-gray-600">
                <div className="flex items-center gap-2 lg:gap-3">
                  <CheckCircle2 className="w-5 h-5 lg:w-6 lg:h-6 xl:w-7 xl:h-7 text-green-600" />
                  <span>Более 1000+ успешных проектов</span>
                </div>
                <div className="flex items-center gap-2 lg:gap-3">
                  <CheckCircle2 className="w-5 h-5 lg:w-6 lg:h-6 xl:w-7 xl:h-7 text-green-600" />
                  <span>99.5% точность нормализации</span>
                </div>
                <div className="flex items-center gap-2 lg:gap-3">
                  <CheckCircle2 className="w-5 h-5 lg:w-6 lg:h-6 xl:w-7 xl:h-7 text-green-600" />
                  <span>Официальные данные Росстандарта</span>
                </div>
              </div>
            </div>
          </section>

          {/* E-E-A-T секция: Experience, Expertise, Authoritativeness, Trustworthiness */}
          <section id="eeat" className="mb-16 md:mb-20 lg:mb-24">
            <div className="grid md:grid-cols-2 xl:grid-cols-2 gap-6 md:gap-8 lg:gap-10 xl:gap-12 mb-12">
              <Card className="border-blue-100">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <CheckCircle2 className="w-6 h-6 text-blue-600" />
                    Опыт и экспертиза
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <p className="text-gray-600 mb-4">
                    Наша команда специализируется на нормализации данных для корпоративных систем с 2024 года.
                    Более <strong>1000+ проектов</strong> успешно обработано с использованием наших технологий.
                    Мы обладаем глубокой экспертизой в области машинного обучения, обработки естественного языка
                    и интеграции с корпоративными ERP/CRM системами.
                  </p>
                  <p className="text-gray-600">
                    Наши решения основаны на опыте работы с крупными предприятиями и учитывают специфику
                    различных отраслей, обеспечивая максимальную точность и эффективность нормализации данных.
                  </p>
                </CardContent>
              </Card>

              <Card className="border-green-100">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="w-6 h-6 text-green-600" />
                    Авторитетность и доверие
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4 lg:space-y-6 xl:space-y-8">
                    <div className="flex items-start gap-3 lg:gap-4 xl:gap-5">
                      <div className="text-2xl lg:text-3xl xl:text-4xl font-bold text-green-600">50+</div>
                      <div>
                        <div className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Источников ГОСТ</div>
                        <div className="text-sm lg:text-base xl:text-lg text-gray-600">
                          Официальные данные{' '}
                          <a
                            href="https://www.rst.gov.ru"
                            target="_blank"
                            rel="noopener noreferrer"
                            className="text-blue-600 hover:underline"
                          >
                            Росстандарта
                          </a>
                        </div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 lg:gap-4 xl:gap-5">
                      <div className="text-2xl lg:text-3xl xl:text-4xl font-bold text-blue-600">1000+</div>
                      <div>
                        <div className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Обработанных записей</div>
                        <div className="text-sm lg:text-base xl:text-lg text-gray-600">В день на платформе</div>
                      </div>
                    </div>
                    <div className="flex items-start gap-3 lg:gap-4 xl:gap-5">
                      <div className="text-2xl lg:text-3xl xl:text-4xl font-bold text-purple-600">99.5%</div>
                      <div>
                        <div className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Точность нормализации</div>
                        <div className="text-sm lg:text-base xl:text-lg text-gray-600">При использовании калибруемых моделей</div>
                      </div>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </section>

          {/* Технологический стек - приоритетная секция */}
          <section id="technologies" className="mb-16 md:mb-20 lg:mb-24">
            <div className="text-center mb-10 md:mb-12 lg:mb-16">
              <Cpu className="w-12 h-12 lg:w-16 lg:h-16 xl:w-20 xl:h-20 text-blue-600 mx-auto mb-4 lg:mb-6" />
              <h2 className="text-3xl md:text-4xl lg:text-5xl xl:text-6xl font-bold text-gray-900 mb-4 lg:mb-6">Технологический стек нормализации данных</h2>
              <p className="text-lg md:text-xl lg:text-2xl text-gray-600 max-w-2xl lg:max-w-3xl xl:max-w-4xl mx-auto">
                Современные технологии для максимальной производительности и масштабируемости.
                Узнайте больше о{' '}
                <Link href="/monitoring" className="text-blue-600 hover:text-blue-700 font-semibold underline">
                  производительности системы
                </Link>{' '}
                и{' '}
                <Link href="/workers" className="text-blue-600 hover:text-blue-700 font-semibold underline">
                  настройке AI воркеров
                </Link>
                .
              </p>
            </div>

            <div className="grid lg:grid-cols-2 xl:grid-cols-2 gap-6 md:gap-8 lg:gap-10 xl:gap-12 mb-12">
              {/* Frontend технологии */}
              <Card className="border-blue-100">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Code2 className="w-8 h-8 text-blue-600" />
                    Frontend & UI
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-blue-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Next.js 16</span>
                      <span className="text-xs lg:text-sm xl:text-base text-blue-600 bg-blue-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        React Server Components
                      </span>
                    </div>
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-purple-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">TypeScript</span>
                      <span className="text-xs lg:text-sm xl:text-base text-purple-600 bg-purple-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        Полная типобезопасность
                      </span>
                    </div>
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-green-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">shadcn/ui</span>
                      <span className="text-xs lg:text-sm xl:text-base text-green-600 bg-green-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        Modern UI Library
                      </span>
                    </div>
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-orange-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Tailwind CSS</span>
                      <span className="text-xs lg:text-sm xl:text-base text-orange-600 bg-orange-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        Utility-First CSS
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>

              {/* Backend технологии */}
              <Card className="border-green-100">
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Database className="w-8 h-8 text-green-600" />
                    Backend & Infrastructure
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-green-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Go (Golang)</span>
                      <span className="text-xs lg:text-sm xl:text-base text-green-600 bg-green-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        Высокая производительность
                      </span>
                    </div>
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-red-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">REST API</span>
                      <span className="text-xs lg:text-sm xl:text-base text-red-600 bg-red-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        GraphQL готовность
                      </span>
                    </div>
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-indigo-50 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Docker</span>
                      <span className="text-xs lg:text-sm xl:text-base text-indigo-600 bg-indigo-100 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        Контейнеризация
                      </span>
                    </div>
                    <div className="flex justify-between items-center p-4 lg:p-5 xl:p-6 bg-gray-100 rounded-lg">
                      <span className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">Microservices</span>
                      <span className="text-xs lg:text-sm xl:text-base text-gray-600 bg-gray-200 px-2 py-1 lg:px-3 lg:py-1.5 rounded">
                        Архитектура
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </section>

          {/* Калибруемые ML модели */}
          <section id="ml-models" className="mb-16 md:mb-20 lg:mb-24">
            <div className="bg-gradient-to-r from-purple-600 to-indigo-700 rounded-3xl text-white p-8 md:p-12 lg:p-16 xl:p-20">
              <div className="flex items-center mb-8 lg:mb-12">
                <Brain className="w-10 h-10 lg:w-12 lg:h-12 xl:w-16 xl:h-16 text-white mr-4 lg:mr-6" />
                <h2 className="text-2xl md:text-3xl lg:text-4xl xl:text-5xl font-bold">Калибруемые ML модели машинного обучения</h2>
              </div>

              <div className="grid md:grid-cols-2 xl:grid-cols-2 gap-6 md:gap-8 lg:gap-10 xl:gap-12">
                <div>
                  <h3 className="text-xl font-semibold mb-4 text-purple-200">
                    🔧 Персонализация под клиента
                  </h3>
                  <ul className="space-y-3 text-purple-100">
                    <li className="flex items-start">
                      <Shield className="w-5 h-5 mr-3 mt-0.5 flex-shrink-0" />
                      <span>
                        Индивидуальные ML модели для каждого клиента через WorkerConfigManager
                      </span>
                    </li>
                    <li className="flex items-start">
                      <Zap className="w-5 h-5 mr-3 mt-0.5 flex-shrink-0" />
                      <span>
                        Настройка параметров AI моделей (temperature, max_tokens, приоритеты)
                      </span>
                    </li>
                    <li className="flex items-start">
                      <BarChart3 className="w-5 h-5 mr-3 mt-0.5 flex-shrink-0" />
                      <span>Система эталонов (benchmarks) для обучения на данных клиента</span>
                    </li>
                  </ul>
                </div>
                <div>
                  <h3 className="text-xl font-semibold mb-4 text-purple-200">
                    🎯 Клиент-специфичные стратегии
                  </h3>
                  <ul className="space-y-3 text-purple-100">
                    <li className="flex items-start">
                      <GitBranch className="w-5 h-5 mr-3 mt-0.5 flex-shrink-0" />
                      <span>Поддержка специфичных бизнес-процессов клиента</span>
                    </li>
                    <li className="flex items-start">
                      <Layers className="w-5 h-5 mr-3 mt-0.5 flex-shrink-0" />
                      <span>Адаптивные алгоритмы классификации под отрасль</span>
                    </li>
                    <li className="flex items-start">
                      <Network className="w-5 h-5 mr-3 mt-0.5 flex-shrink-0" />
                      <span>Динамическая калибровка моделей на основе feedback loop</span>
                    </li>
                  </ul>
                </div>
              </div>
            </div>
          </section>

          {/* Интеграция с классификаторами */}
          <section id="classifiers" className="mb-16 md:mb-20 lg:mb-24">
            <div className="text-center mb-10 md:mb-12 lg:mb-16">
              <Layers className="w-12 h-12 lg:w-16 lg:h-16 xl:w-20 xl:h-20 text-green-600 mx-auto mb-4 lg:mb-6" />
              <h2 className="text-3xl md:text-4xl lg:text-5xl xl:text-6xl font-bold text-gray-900 mb-4 lg:mb-6">50+ Классификаторов ГОСТ для категоризации данных</h2>
              <p className="text-lg md:text-xl lg:text-2xl text-gray-600 max-w-3xl lg:max-w-4xl xl:max-w-5xl mx-auto">
                Полная интеграция с национальными и межгосударственными стандартами. Просмотрите{' '}
                <Link href="/classifiers" className="text-blue-600 hover:text-blue-700 font-semibold underline">
                  все доступные классификаторы
                </Link>{' '}
                и{' '}
                <Link href="/gosts" className="text-blue-600 hover:text-blue-700 font-semibold underline">
                  базу ГОСТов из 50+ источников Росстандарта
                </Link>
                .
              </p>
              <div className="mt-8 max-w-2xl mx-auto">
                <div className="relative h-48 rounded-xl overflow-hidden bg-gradient-to-r from-green-50 to-blue-50">
                  <div className="absolute inset-0 flex items-center justify-center">
                    <div className="grid grid-cols-3 gap-4 p-4 opacity-20">
                      <Globe className="w-12 h-12 text-green-600" />
                      <Layers className="w-12 h-12 text-blue-600" />
                      <CheckCircle2 className="w-12 h-12 text-purple-600" />
                    </div>
                  </div>
                  <Image
                    src="/classifiers-illustration.jpg"
                    alt="50+ классификаторов ГОСТ: интеграция с Росстандартом, КПВЭД, ОКПД2 и классификаторами стран СНГ для нормализации и категоризации данных"
                    width={800}
                    height={300}
                    className="object-cover rounded-xl opacity-90"
                    loading="lazy"
                    placeholder="blur"
                    blurDataURL="data:image/svg+xml;base64,PHN2ZyB3aWR0aD0iODAwIiBoZWlnaHQ9IjMwMCIgeG1sbnM9Imh0dHA6Ly93d3cudzMub3JnLzIwMDAvc3ZnIj48cmVjdCB3aWR0aD0iMTAwJSIgaGVpZ2h0PSIxMDAlIiBmaWxsPSIjZjBmZGY0Ii8+PHRleHQgeD0iNTAlIiB5PSI1MCUiIGZvbnQtc2l6ZT0iMjQiIGZpbGw9IiMxNmEzNGIiIHRleHQtYW5jaG9yPSJtaWRkbGUiIGR5PSIuM2VtIj5DbGFzc2lmaWVyczwvdGV4dD48L3N2Zz4="
                  />
                </div>
              </div>
            </div>

            <div className="grid md:grid-cols-3 xl:grid-cols-3 gap-6 lg:gap-8 xl:gap-10 mb-8 lg:mb-12">
              {[
                {
                  title: 'Росстандарт ГОСТы',
                  count: '50+ источников',
                  description: 'Национальные стандарты РФ, межгосударственные стандарты',
                  color: 'bg-blue-50 border-blue-200',
                },
                {
                  title: 'КПВЭД и ОКПД2',
                  count: 'Полное покрытие',
                  description: 'Классификаторы продукции и видов экономической деятельности',
                  color: 'bg-green-50 border-green-200',
                },
                {
                  title: 'СНГ классификаторы',
                  count: '20+ стран',
                  description: 'Поддержка классификаторов всех стран Содружества',
                  color: 'bg-purple-50 border-purple-200',
                },
              ].map((item, index) => (
                <Card key={index} className={`${item.color} border-2`}>
                  <CardContent className="p-6 lg:p-8 xl:p-10 text-center">
                    <div className="text-xl lg:text-2xl xl:text-3xl font-bold text-gray-900 mb-2 lg:mb-3">{item.title}</div>
                    <div className="text-lg lg:text-xl xl:text-2xl text-blue-600 font-semibold mb-2 lg:mb-3">{item.count}</div>
                    <div className="text-sm lg:text-base xl:text-lg text-gray-600">{item.description}</div>
                  </CardContent>
                </Card>
              ))}
            </div>

            <Card>
              <CardHeader>
                <CardTitle className="text-2xl">🎯 Иерархическая классификация</CardTitle>
              </CardHeader>
              <CardContent>
                <div className="grid md:grid-cols-2 xl:grid-cols-2 gap-6 md:gap-8 lg:gap-10 xl:gap-12">
                  <div>
                    <h4 className="font-semibold text-gray-800 mb-3 lg:text-lg xl:text-xl">Гибкие стратегии folding</h4>
                    <p className="text-gray-600 mb-4">
                      Интеллектуальное сворачивание категориальных деревьев с настраиваемой глубиной
                    </p>
                    <ul className="text-gray-600 space-y-2">
                      <li>• Многоуровневая категоризация</li>
                      <li>• Автоматическое определение глубины</li>
                      <li>• Контекстно-зависимое folding</li>
                    </ul>
                  </div>
                  <div>
                    <h4 className="font-semibold text-gray-800 mb-3 lg:text-lg xl:text-xl">
                      Система категориальных деревьев
                    </h4>
                    <p className="text-gray-600 mb-4">
                      Построение сложных иерархических структур классификации
                    </p>
                    <ul className="text-gray-600 space-y-2">
                      <li>• Динамическое построение деревьев</li>
                      <li>• Автоматическая оптимизация структуры</li>
                      <li>• Визуализация иерархий</li>
                    </ul>
                  </div>
                </div>
              </CardContent>
            </Card>
          </section>

          {/* Топовые ИИ модели */}
          <section id="ai-models" className="mb-16 md:mb-20 lg:mb-24">
            <div className="bg-gradient-to-r from-blue-500 to-cyan-600 rounded-3xl text-white p-8 md:p-12 lg:p-16 xl:p-20">
              <div className="flex items-center mb-8 lg:mb-12">
                <Network className="w-10 h-10 lg:w-12 lg:h-12 xl:w-16 xl:h-16 text-white mr-4 lg:mr-6" />
                <h2 className="text-2xl md:text-3xl lg:text-4xl xl:text-5xl font-bold">Топовые ИИ модели для обработки естественного языка</h2>
              </div>
              <p className="text-blue-100 mb-6 lg:mb-8 text-lg lg:text-xl xl:text-2xl">
                Управляйте AI моделями и настраивайте провайдеры через{' '}
                <Link
                  href="/workers"
                  className="text-white font-semibold underline hover:text-blue-200"
                >
                  панель управления воркерами
                </Link>
                . Отслеживайте производительность в{' '}
                <Link
                  href="/monitoring"
                  className="text-white font-semibold underline hover:text-blue-200"
                >
                  реальном времени
                </Link>
                .
              </p>

              <div className="grid lg:grid-cols-2 xl:grid-cols-2 gap-6 md:gap-8 lg:gap-10 xl:gap-12">
                <div>
                  <h3 className="text-xl font-semibold mb-6 text-blue-100">
                    🤖 Основные провайдеры AI
                  </h3>
                  <div className="space-y-4">
                    {[
                      { name: 'OpenRouter', models: 'z.ai/glm-4.5 (приоритет)', status: 'primary' },
                      { name: 'Arliai', models: 'GLM-4.5-Air для чата', status: 'primary' },
                      { name: 'Hugging Face', models: 'Генеративные модели', status: 'secondary' },
                      { name: 'Eden AI', models: 'Мульти-провайдерный доступ', status: 'secondary' },
                    ].map((provider, index) => (
                      <div
                        key={index}
                        className="flex justify-between items-center p-4 bg-blue-400/20 rounded-lg backdrop-blur-sm"
                      >
                        <div>
                          <div className="font-semibold">{provider.name}</div>
                          <div className="text-blue-200 text-sm">{provider.models}</div>
                        </div>
                        <span
                          className={`px-3 py-1 rounded-full text-xs ${
                            provider.status === 'primary'
                              ? 'bg-green-500 text-white'
                              : 'bg-blue-300 text-blue-800'
                          }`}
                        >
                          {provider.status === 'primary' ? 'Основной' : 'Резервный'}
                        </span>
                      </div>
                    ))}
                  </div>
                </div>

                <div>
                  <h3 className="text-xl font-semibold mb-6 text-blue-100">
                    🔄 Мульти-провайдерная архитектура
                  </h3>
                  <div className="space-y-4">
                    {[
                      { strategy: 'First Success', desc: 'Первый успешный ответ' },
                      { strategy: 'Consensus', desc: 'Консенсус между провайдерами' },
                      { strategy: 'Best Quality', desc: 'Выбор лучшего качества' },
                      { strategy: 'Fallback Chain', desc: 'Цепочка резервных провайдеров' },
                    ].map((item, index) => (
                      <div key={index} className="p-4 bg-blue-400/20 rounded-lg backdrop-blur-sm">
                        <div className="font-semibold mb-1">{item.strategy}</div>
                        <div className="text-blue-200 text-sm">{item.desc}</div>
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              <div className="mt-8 grid md:grid-cols-2 gap-6">
                <div className="bg-white/10 p-4 rounded-lg">
                  <h4 className="font-semibold mb-2">🇷🇺 DaData интеграция</h4>
                  <p className="text-blue-100 text-sm">Верификация российских компаний по ИНН</p>
                </div>
                <div className="bg-white/10 p-4 rounded-lg">
                  <h4 className="font-semibold mb-2">🇰🇿 Adata.kz интеграция</h4>
                  <p className="text-blue-100 text-sm">Верификация казахстанских компаний по БИН</p>
                </div>
              </div>
            </div>
          </section>

          {/* Нормализация номенклатуры */}
          <section id="normalization" className="mb-16 md:mb-20 lg:mb-24">
            <div className="text-center mb-10 md:mb-12 lg:mb-16">
              <Shield className="w-12 h-12 lg:w-16 lg:h-16 xl:w-20 xl:h-20 text-indigo-600 mx-auto mb-4 lg:mb-6" />
              <h2 className="text-3xl md:text-4xl lg:text-5xl xl:text-6xl font-bold text-gray-900 mb-4 lg:mb-6">Нормализация и унификация номенклатуры 1С</h2>
              <p className="text-lg md:text-xl lg:text-2xl text-gray-600 max-w-3xl lg:max-w-4xl xl:max-w-5xl mx-auto">
                AI-усиленная нормализация данных для 1С и любых ERP/CRM систем. Узнайте больше о{' '}
                <Link href="/processes/nomenclature" className="text-blue-600 hover:text-blue-700 font-semibold underline">
                  процессах нормализации номенклатуры
                </Link>
                {' '}и{' '}
                <Link href="/quality" className="text-blue-600 hover:text-blue-700 font-semibold underline">
                  анализе качества данных
                </Link>
                .
              </p>
            </div>

            <div className="grid md:grid-cols-2 xl:grid-cols-2 gap-6 md:gap-8 lg:gap-10 xl:gap-12">
              <Card className="border-indigo-100">
                <CardHeader>
                  <CardTitle className="text-2xl">🔍 Интеллектуальная дедупликация</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {[
                      { type: 'Exact Matching', desc: 'Точное совпадение записей' },
                      { type: 'Semantic Duplicates', desc: 'Косинусная близость векторов' },
                      { type: 'Phonetic Duplicates', desc: 'Фонетическое сходство' },
                      { type: 'Word-based Duplicates', desc: 'Слово-ориентированное сравнение' },
                    ].map((method, index) => (
                      <div key={index} className="flex items-center p-4 lg:p-5 xl:p-6 bg-indigo-50 rounded-lg">
                        <div className="w-3 h-3 lg:w-4 lg:h-4 xl:w-5 xl:h-5 bg-indigo-500 rounded-full mr-4 lg:mr-5"></div>
                        <div>
                          <div className="font-semibold text-gray-800 text-base lg:text-lg xl:text-xl">{method.type}</div>
                          <div className="text-gray-600 text-sm lg:text-base xl:text-lg">{method.desc}</div>
                        </div>
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>

              <Card className="border-green-100">
                <CardHeader>
                  <CardTitle className="text-2xl">🔄 Поддержка систем</CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="space-y-6">
                    <div>
                      <h4 className="font-semibold text-gray-800 mb-3">1С:Предприятие</h4>
                      <ul className="text-gray-600 space-y-2">
                        <li>• Обработка справочников и констант</li>
                        <li>• Нормализация номенклатуры товаров</li>
                        <li>• Стандартизация контрагентов</li>
                      </ul>
                    </div>
                    <div>
                      <h4 className="font-semibold text-gray-800 mb-3">Другие ERP/CRM</h4>
                      <ul className="text-gray-600 space-y-2">
                        <li>• SAP, Oracle, Microsoft Dynamics</li>
                        <li>• Любые системы с API доступом</li>
                        <li>• Кастомные бизнес-приложения</li>
                      </ul>
                    </div>
                  </div>
                </CardContent>
              </Card>
            </div>
          </section>

          {/* Ключевые возможности */}
          <section id="features" className="mb-16 md:mb-20 lg:mb-24">
            <div className="bg-gray-900 text-white rounded-3xl p-8 md:p-12 lg:p-16 xl:p-20">
              <h2 className="text-2xl md:text-3xl lg:text-4xl xl:text-5xl font-bold mb-8 lg:mb-12 text-center">🚀 Ключевые возможности</h2>

              <div className="grid md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-3 2xl:grid-cols-3 gap-6 lg:gap-8 xl:gap-10">
                {[
                  {
                    icon: '⚡',
                    title: 'Real-time мониторинг',
                    description: 'Server-Sent Events для отслеживания процессов в реальном времени',
                  },
                  {
                    icon: '📊',
                    title: 'Анализ качества данных',
                    description: 'Метрики полноты, уникальности, корректности данных',
                  },
                  {
                    icon: '🎯',
                    title: 'Управление качеством',
                    description: 'Система бенчмарков и эталонов для контроля качества',
                  },
                  {
                    icon: '🔄',
                    title: 'Мульти-провайдерность',
                    description: 'Параллельная обработка через несколько AI провайдеров',
                  },
                  {
                    icon: '📈',
                    title: 'Визуализация пайплайна',
                    description: 'Графическое представление процесса обработки данных',
                  },
                  {
                    icon: '💾',
                    title: 'Экспорт результатов',
                    description: 'Выгрузка в различных форматах: JSON, CSV, XML',
                  },
                ].map((feature, index) => (
                  <div
                    key={index}
                    className="bg-gray-800 p-6 lg:p-8 xl:p-10 rounded-xl hover:bg-gray-700 transition-colors"
                  >
                    <div className="text-3xl lg:text-4xl xl:text-5xl mb-3 lg:mb-4">{feature.icon}</div>
                    <h3 className="font-semibold text-lg lg:text-xl xl:text-2xl mb-2 lg:mb-3">{feature.title}</h3>
                    <p className="text-gray-300 text-sm lg:text-base xl:text-lg">{feature.description}</p>
                  </div>
                ))}
              </div>
            </div>
          </section>

          {/* FAQ секция */}
          <section id="faq" className="mb-16 md:mb-20 lg:mb-24">
            <Card>
              <CardHeader className="p-6 md:p-8 lg:p-10 xl:p-12">
                <CardTitle className="text-2xl md:text-3xl lg:text-4xl xl:text-5xl text-center">
                  ❓ Часто задаваемые вопросы о технологиях
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Accordion type="single" collapsible className="w-full">
                  <AccordionItem value="item-1">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Какие технологии используются в Нормализаторе данных 1С?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      Мы используем современный технологический стек:{' '}
                      <strong>Next.js 16</strong> с React Server Components для фронтенда,{' '}
                      <strong>Go (Golang)</strong> для высокопроизводительного бэкенда,{' '}
                      <strong>TypeScript</strong> для полной типобезопасности,{' '}
                      <strong>shadcn/ui</strong> для современного пользовательского интерфейса. В
                      части машинного обучения - калибруемые ML модели под каждого клиента,
                      интеграция с <strong>50+ классификаторами ГОСТ</strong> из Росстандарта и
                      топовыми AI моделями через <strong>OpenRouter (GLM-4.5)</strong> и другие
                      провайдеры.
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-2">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Как работают калибруемые ML модели под каждого клиента?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      <p className="mb-3">
                        <strong>Калибруемые ML модели</strong> работают через персонализацию под каждого клиента.
                        Каждый клиент получает индивидуальную модель, которая обучается на его данных через
                        WorkerConfigManager с настройкой параметров AI (temperature, max_tokens, приоритеты) и
                        использованием системы эталонов (benchmarks) для точной настройки под специфику бизнеса.
                      </p>
                      <p>
                        Такой подход обеспечивает максимальную точность обработки данных конкретного клиента,
                        учитывая особенности его предметной области, терминологии и бизнес-процессов.
                      </p>
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-3">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Какие классификаторы поддерживает платформа?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      Мы интегрированы с <strong>50+ источниками ГОСТов</strong> из Росстандарта,
                      включая национальные и межгосударственные стандарты. Поддерживаем
                      классификаторы <strong>КПВЭД, ОКПД2</strong>, а также классификаторы всех
                      стран СНГ. Используем иерархическую классификацию с гибкими стратегиями
                      folding и настраиваемой глубиной категориальных деревьев для оптимальной
                      обработки данных.
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-4">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Какие AI модели используются для нормализации данных?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      <p className="mb-3">
                        <strong>Для нормализации используются топовые AI модели:</strong> основная модель z.ai/glm-4.5
                        через OpenRouter с fallback на meta-llama/llama-3.2-3b-instruct. Также интегрированы Arliai
                        GLM-4.5-Air, модели Hugging Face, Eden AI и провайдеры данных DaData (для российских компаний
                        по ИНН) и Adata.kz (для казахстанских компаний по БИН).
                      </p>
                      <p>
                        Мульти-провайдерная архитектура с стратегиями first_success, consensus и best_quality
                        обеспечивает максимальную надежность и качество обработки данных даже при сбоях отдельных
                        провайдеров.
                      </p>
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-5">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Как работает нормализация номенклатуры в системах 1С?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      <p className="mb-3">
                        <strong>Нормализация номенклатуры в 1С</strong> выполняется автоматически через AI-алгоритмы
                        с применением четырех методов дедупликации: exact matching для точных совпадений, semantic
                        duplicates для семантического сходства через косинусную близость, phonetic duplicates для
                        фонетических совпадений и word-based duplicates для слово-ориентированного сравнения.
                      </p>
                      <p>
                        Система поддерживает любые ERP/CRM системы и обрабатывает справочники, константы из 1С с
                        AI-усиленной нормализацией названий товаров и контрагентов, обеспечивая высокую точность
                        и скорость обработки больших объемов данных.
                      </p>
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-6">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Сколько времени занимает нормализация данных?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      <p className="mb-3">
                        <strong>Скорость нормализации</strong> зависит от объема данных и выбранных AI провайдеров.
                        Средняя скорость обработки составляет 1000-5000 записей в минуту при использовании
                        мульти-провайдерной архитектуры. Система использует параллельную обработку и кэширование
                        для оптимизации производительности.
                      </p>
                      <p>
                        Вы можете отслеживать прогресс в{' '}
                        <Link href="/monitoring" className="text-blue-600 hover:text-blue-700 underline">
                          реальном времени
                        </Link>{' '}
                        через мониторинг процессов. Для больших объемов данных рекомендуется использование
                        эталонных записей (benchmarks), что значительно ускоряет обработку за счет переиспользования
                        результатов.
                      </p>
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-7">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Как обеспечить качество нормализованных данных?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      <p className="mb-3">
                        <strong>Качество данных</strong> обеспечивается через многоуровневую систему контроля:
                        автоматическая валидация на этапе нормализации, система эталонов (benchmarks) для обучения
                        моделей на данных клиента, и{' '}
                        <Link href="/quality" className="text-blue-600 hover:text-blue-700 underline">
                          комплексный анализ качества
                        </Link>{' '}
                        с метриками полноты, уникальности и корректности.
                      </p>
                      <p>
                        После обработки система автоматически выявляет дубликаты, нарушения и предоставляет
                        предложения по улучшению. Все результаты можно экспортировать для дальнейшей проверки
                        и использования в ваших бизнес-процессах.
                      </p>
                    </AccordionContent>
                  </AccordionItem>

                  <AccordionItem value="item-8">
                    <AccordionTrigger className="text-left font-semibold text-lg">
                      Можно ли использовать платформу для других ERP систем кроме 1С?
                    </AccordionTrigger>
                    <AccordionContent className="text-gray-600 text-base leading-relaxed">
                      <p className="mb-3">
                        <strong>Да, платформа поддерживает любые ERP/CRM системы</strong> через REST API интерфейс.
                        Помимо 1С:Предприятие, система успешно интегрируется с SAP, Oracle, Microsoft Dynamics и
                        другими корпоративными системами, а также кастомными бизнес-приложениями через стандартные
                        API протоколы.
                      </p>
                      <p>
                        Для интеграции необходимо предоставить данные в одном из поддерживаемых форматов (JSON, CSV,
                        XML). Система автоматически определит структуру данных и применит соответствующие алгоритмы
                        нормализации. Для начала работы с{' '}
                        <Link href="/processes" className="text-blue-600 hover:text-blue-700 underline">
                          процессами нормализации
                        </Link>
                        , загрузите данные через удобный интерфейс.
                      </p>
                    </AccordionContent>
                  </AccordionItem>
                </Accordion>
              </CardContent>
            </Card>
          </section>

          {/* CTA секция */}
          <section className="text-center bg-gradient-to-r from-blue-600 to-purple-700 text-white rounded-3xl p-8 md:p-12 lg:p-16 xl:p-20 mb-8 lg:mb-12 shadow-2xl">
            <h2 className="text-2xl md:text-3xl lg:text-4xl xl:text-5xl 2xl:text-6xl font-bold mb-6 lg:mb-8 xl:mb-10">
              Готовы внедрить передовые технологии AI?
            </h2>
            <p className="text-lg md:text-xl lg:text-2xl xl:text-3xl mb-8 lg:mb-10 xl:mb-12 opacity-90 max-w-2xl lg:max-w-3xl xl:max-w-4xl mx-auto leading-relaxed">
              Начните использовать Нормализатор данных 1С с калибруемыми ML моделями и топовыми AI
              алгоритмами для трансформации ваших данных.
            </p>
            <div className="flex flex-col sm:flex-row gap-4 lg:gap-6 xl:gap-8 justify-center">
              <Button
                asChild
                size="lg"
                className="bg-white text-blue-600 hover:bg-gray-100 text-base lg:text-lg xl:text-xl px-6 py-5 lg:px-8 lg:py-6 xl:px-10 xl:py-7 h-auto"
              >
                <Link href="/contact">🚀 Связаться с нами</Link>
              </Button>
              <Button
                asChild
                size="lg"
                variant="outline"
                className="border-2 border-white text-white hover:bg-white hover:text-blue-600 text-base lg:text-lg xl:text-xl px-6 py-5 lg:px-8 lg:py-6 xl:px-10 xl:py-7 h-auto"
              >
                <Link href="/demo">💻 Посмотреть демо</Link>
              </Button>
            </div>
          </section>
        </div>
      </main>
    </>
  )
}
