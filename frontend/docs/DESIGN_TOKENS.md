# Руководство по использованию токенов дизайна

Это руководство описывает использование системы дизайн-токенов в приложении.

## Импорт токенов

```tsx
import { tokens, getToken } from '@/styles/tokens'
```

## Цвета

### Primary (Основной)

```tsx
tokens.color.primary[500] // #3b82f6
tokens.color.primary[900] // #1e3a8a
```

### Gray (Серый)

```tsx
tokens.color.gray[500] // #6b7280
tokens.color.gray[900] // #111827
```

### Success (Успех)

```tsx
tokens.color.success[500] // #22c55e
```

### Error (Ошибка)

```tsx
tokens.color.error[500] // #ef4444
```

## Отступы (Spacing)

```tsx
tokens.spacing.xs  // 0.25rem (4px)
tokens.spacing.sm  // 0.5rem (8px)
tokens.spacing.md  // 1rem (16px)
tokens.spacing.lg  // 1.5rem (24px)
tokens.spacing.xl  // 2rem (32px)
```

### Использование в компонентах

```tsx
<div style={{ padding: tokens.spacing.md }}>
  Контент
</div>
```

## Радиусы скругления

```tsx
tokens.borderRadius.sm   // 0.25rem
tokens.borderRadius.md   // 0.5rem
tokens.borderRadius.lg   // 0.75rem
tokens.borderRadius.full // 9999px
```

## Тени

```tsx
tokens.shadows.sm   // Тень маленькая
tokens.shadows.md   // Тень средняя
tokens.shadows.lg   // Тень большая
tokens.shadows.xl   // Тень очень большая
```

### Использование

```tsx
<div style={{ boxShadow: tokens.shadows.md }}>
  Элемент с тенью
</div>
```

## Типографика

### Размеры шрифта

```tsx
tokens.typography.fontSize.xs    // 0.75rem
tokens.typography.fontSize.base  // 1rem
tokens.typography.fontSize.lg    // 1.125rem
tokens.typography.fontSize['2xl'] // 1.5rem
```

### Веса шрифта

```tsx
tokens.typography.fontWeight.normal  // 400
tokens.typography.fontWeight.medium  // 500
tokens.typography.fontWeight.bold    // 700
```

## CSS переменные

Токены также доступны как CSS переменные в `globals.css`:

```css
/* Отступы */
--token-spacing-xs: 0.25rem;
--token-spacing-md: 1rem;

/* Цвета */
--token-color-primary-500: #3b82f6;
--token-color-success-500: #22c55e;

/* Тени */
--token-shadow-md: 0 4px 6px -1px rgb(0 0 0 / 0.1);

/* Радиусы */
--token-radius-md: 0.5rem;
```

### Использование в CSS

```css
.my-component {
  padding: var(--token-spacing-md);
  border-radius: var(--token-radius-lg);
  box-shadow: var(--token-shadow-md);
  color: var(--token-color-primary-500);
}
```

## Функция getToken

Утилита для безопасного получения токенов:

```tsx
import { getToken } from '@/styles/tokens'

const primaryColor = getToken('color', 'primary', 500)
const spacing = getToken('spacing', 'md')
```

## Лучшие практики

1. Всегда используйте токены вместо хардкода значений
2. Используйте CSS переменные в стилях для лучшей производительности
3. При изменении токенов обновляются все компоненты автоматически
4. Следуйте существующей шкале (50, 100, 200... 900) для цветов

