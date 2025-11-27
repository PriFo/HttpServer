# Руководство по использованию анимаций

Это руководство описывает использование системы анимаций в приложении.

## AnimationProvider

`AnimationProvider` управляет глобальными настройками анимаций во всем приложении.

### Варианты анимаций

- **`gentle`** - Плавные анимации (по умолчанию, 0.4s)
- **`fast`** - Быстрые анимации (0.2s)
- **`none`** - Без анимаций (для доступности)

### Использование

```tsx
import { AnimationProvider } from '@/providers/animation-provider'

// В app/layout.tsx уже интегрирован
<AnimationProvider defaultVariant="gentle">
  {children}
</AnimationProvider>
```

### Получение настроек анимации в компонентах

```tsx
import { useAnimationContext } from '@/providers/animation-provider'

function MyComponent() {
  const { variant, setVariant, getAnimationConfig } = useAnimationContext()
  
  const config = getAnimationConfig()
  // config.duration - длительность анимации
  // config.ease - функция плавности
}
```

## useAnimatedState

Хук для плавной анимации изменений состояния.

### Пример использования

```tsx
import { useAnimatedState } from '@/hooks/use-animated-state'

function MyComponent() {
  const { state, setAnimatedState, isVisible } = useAnimatedState('initial')
  
  return (
    <AnimatePresence>
      {isVisible && <div>{state}</div>}
    </AnimatePresence>
  )
}
```

## useAnimatedList

Хук для анимации списков элементов.

### Пример использования

```tsx
import { useAnimatedList } from '@/hooks/use-animated-state'

function MyList() {
  const { items, setItems, isVisible } = useAnimatedList(['item1', 'item2'])
  
  return (
    <AnimatePresence>
      {isVisible && items.map(item => <Item key={item} item={item} />)}
    </AnimatePresence>
  )
}
```

## AnimatedCard

Готовый компонент анимированной карточки.

### Пример использования

```tsx
import { AnimatedCard } from '@/components/common/animated-card'

<AnimatedCard 
  title="Заголовок" 
  description="Описание"
  variant="hover"
  delay={0.1}
>
  Содержимое карточки
</AnimatedCard>
```

### Варианты

- **`default`** - Стандартная анимация появления
- **`hover`** - С эффектом при наведении
- **`scale`** - С масштабированием

## AnimatedList

Готовый компонент анимированного списка.

### Пример использования

```tsx
import { AnimatedList } from '@/components/common/animated-list'

<AnimatedList
  items={items}
  keyExtractor={(item) => item.id}
  renderItem={(item, index) => <Item key={item.id} item={item} />}
  staggerDelay={0.05}
  emptyMessage="Список пуст"
/>
```

## Доступность

Система автоматически учитывает настройки пользователя:

- Поддержка `prefers-reduced-motion`
- Автоматическое отключение анимаций при необходимости
- Минимальные анимации для доступности

## Лучшие практики

1. Используйте `AnimationProvider` для глобального управления
2. Используйте готовые компоненты (`AnimatedCard`, `AnimatedList`) когда возможно
3. Учитывайте производительность - не анимируйте слишком много элементов одновременно
4. Всегда тестируйте с `prefers-reduced-motion: reduce`

