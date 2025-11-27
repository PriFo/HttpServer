/**
 * Токены дизайн-системы
 * 
 * Единая система дизайн-токенов для обеспечения консистентности
 * во всем приложении.
 */

export const tokens = {
  // Цветовая палитра
  color: {
    // Основной цвет (Primary)
    primary: {
      50: '#eff6ff',
      100: '#dbeafe',
      200: '#bfdbfe',
      300: '#93c5fd',
      400: '#60a5fa',
      500: '#3b82f6',
      600: '#2563eb',
      700: '#1d4ed8',
      800: '#1e40af',
      900: '#1e3a8a',
      950: '#172554',
    },
    // Серый (Gray/Neutral)
    gray: {
      50: '#f9fafb',
      100: '#f3f4f6',
      200: '#e5e7eb',
      300: '#d1d5db',
      400: '#9ca3af',
      500: '#6b7280',
      600: '#4b5563',
      700: '#374151',
      800: '#1f2937',
      900: '#111827',
      950: '#030712',
    },
    // Успех (Success)
    success: {
      50: '#f0fdf4',
      100: '#dcfce7',
      200: '#bbf7d0',
      300: '#86efac',
      400: '#4ade80',
      500: '#22c55e',
      600: '#16a34a',
      700: '#15803d',
      800: '#166534',
      900: '#14532d',
      950: '#052e16',
    },
    // Ошибка (Error/Destructive)
    error: {
      50: '#fef2f2',
      100: '#fee2e2',
      200: '#fecaca',
      300: '#fca5a5',
      400: '#f87171',
      500: '#ef4444',
      600: '#dc2626',
      700: '#b91c1c',
      800: '#991b1b',
      900: '#7f1d1d',
      950: '#450a0a',
    },
    // Предупреждение (Warning)
    warning: {
      50: '#fffbeb',
      100: '#fef3c7',
      200: '#fde68a',
      300: '#fcd34d',
      400: '#fbbf24',
      500: '#f59e0b',
      600: '#d97706',
      700: '#b45309',
      800: '#92400e',
      900: '#78350f',
      950: '#451a03',
    },
    // Информация (Info)
    info: {
      50: '#eff6ff',
      100: '#dbeafe',
      200: '#bfdbfe',
      300: '#93c5fd',
      400: '#60a5fa',
      500: '#3b82f6',
      600: '#2563eb',
      700: '#1d4ed8',
      800: '#1e40af',
      900: '#1e3a8a',
      950: '#172554',
    },
  },

  // Система отступов (Spacing)
  spacing: {
    xs: '0.25rem',   // 4px
    sm: '0.5rem',    // 8px
    md: '1rem',      // 16px
    lg: '1.5rem',    // 24px
    xl: '2rem',      // 32px
    '2xl': '3rem',   // 48px
    '3xl': '4rem',   // 64px
    '4xl': '6rem',   // 96px
  },

  // Радиусы скругления (Border Radius)
  borderRadius: {
    none: '0',
    sm: '0.25rem',   // 4px
    md: '0.5rem',    // 8px
    lg: '0.75rem',   // 12px
    xl: '1rem',      // 16px
    '2xl': '1.5rem', // 24px
    full: '9999px',
  },

  // Тени (Shadows)
  shadows: {
    sm: '0 1px 2px 0 rgb(0 0 0 / 0.05)',
    md: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)',
    lg: '0 10px 15px -3px rgb(0 0 0 / 0.1), 0 4px 6px -4px rgb(0 0 0 / 0.1)',
    xl: '0 20px 25px -5px rgb(0 0 0 / 0.1), 0 8px 10px -6px rgb(0 0 0 / 0.1)',
    '2xl': '0 25px 50px -12px rgb(0 0 0 / 0.25)',
    inner: 'inset 0 2px 4px 0 rgb(0 0 0 / 0.05)',
    none: 'none',
  },

  // Типографика (Typography)
  typography: {
    // Размеры шрифта
    fontSize: {
      xs: '0.75rem',    // 12px
      sm: '0.875rem',   // 14px
      base: '1rem',     // 16px
      lg: '1.125rem',   // 18px
      xl: '1.25rem',    // 20px
      '2xl': '1.5rem',  // 24px
      '3xl': '1.875rem', // 30px
      '4xl': '2.25rem', // 36px
      '5xl': '3rem',    // 48px
      '6xl': '3.75rem', // 60px
    },
    // Веса шрифта
    fontWeight: {
      thin: '100',
      extralight: '200',
      light: '300',
      normal: '400',
      medium: '500',
      semibold: '600',
      bold: '700',
      extrabold: '800',
      black: '900',
    },
    // Высота строки (Line Height)
    lineHeight: {
      none: '1',
      tight: '1.25',
      snug: '1.375',
      normal: '1.5',
      relaxed: '1.625',
      loose: '2',
    },
    // Межбуквенное расстояние (Letter Spacing)
    letterSpacing: {
      tighter: '-0.05em',
      tight: '-0.025em',
      normal: '0em',
      wide: '0.025em',
      wider: '0.05em',
      widest: '0.1em',
    },
  },

  // Z-index слои
  zIndex: {
    hide: -1,
    auto: 'auto',
    base: 0,
    docked: 10,
    dropdown: 1000,
    sticky: 1100,
    banner: 1200,
    overlay: 1300,
    modal: 1400,
    popover: 1500,
    skipLink: 1600,
    toast: 1700,
    tooltip: 1800,
  },

  // Переходы (Transitions)
  transitions: {
    duration: {
      fast: '150ms',
      base: '200ms',
      slow: '300ms',
      slower: '400ms',
    },
    easing: {
      linear: 'linear',
      easeIn: 'cubic-bezier(0.4, 0, 1, 1)',
      easeOut: 'cubic-bezier(0, 0, 0.2, 1)',
      easeInOut: 'cubic-bezier(0.4, 0, 0.2, 1)',
    },
  },

  // Брейкпоинты для медиа-запросов (для использования в JS)
  breakpoints: {
    sm: '640px',
    md: '768px',
    lg: '1024px',
    xl: '1280px',
    '2xl': '1536px',
  },
} as const

// Типы для TypeScript
export type TokenColorScale = 50 | 100 | 200 | 300 | 400 | 500 | 600 | 700 | 800 | 900 | 950
export type TokenSpacing = keyof typeof tokens.spacing
export type TokenBorderRadius = keyof typeof tokens.borderRadius
export type TokenShadow = keyof typeof tokens.shadows
export type TokenFontSize = keyof typeof tokens.typography.fontSize
export type TokenFontWeight = keyof typeof tokens.typography.fontWeight
export type TokenLineHeight = keyof typeof tokens.typography.lineHeight
export type TokenZIndex = keyof typeof tokens.zIndex

/**
 * Вспомогательная функция для получения значения токена с проверкой типа
 */
export function getToken<K extends keyof typeof tokens>(
  category: K,
  key: keyof typeof tokens[K]
): string {
  return (tokens[category] as any)[key]
}

/**
 * Функция для создания CSS переменных из токенов
 */
export function tokensToCSSVars(prefix = 'token'): Record<string, string> {
  const vars: Record<string, string> = {}
  
  // Цвета
  Object.entries(tokens.color).forEach(([colorName, scale]) => {
    Object.entries(scale).forEach(([shade, value]) => {
      vars[`--${prefix}-color-${colorName}-${shade}`] = value
    })
  })
  
  // Отступы
  Object.entries(tokens.spacing).forEach(([key, value]) => {
    vars[`--${prefix}-spacing-${key}`] = value
  })
  
  // Радиусы
  Object.entries(tokens.borderRadius).forEach(([key, value]) => {
    vars[`--${prefix}-radius-${key}`] = value
  })
  
  // Тени
  Object.entries(tokens.shadows).forEach(([key, value]) => {
    vars[`--${prefix}-shadow-${key}`] = value
  })
  
  // Типографика
  Object.entries(tokens.typography.fontSize).forEach(([key, value]) => {
    vars[`--${prefix}-font-size-${key}`] = value
  })
  
  Object.entries(tokens.typography.fontWeight).forEach(([key, value]) => {
    vars[`--${prefix}-font-weight-${key}`] = value
  })
  
  return vars
}

