# Исправление ошибки компиляции

## Дата: 21 ноября 2025

## Проблема

Ошибка компиляции в `frontend/app/clients/[clientId]/components/counterparties-tab.tsx`:
```
the name `hasClientFilters` is defined multiple times
```

Переменная `hasClientFilters` была объявлена дважды:
1. На строке 118 внутри функции `fetchCounterparties` (useCallback)
2. На строке 386 внутри useMemo

## Решение

Вынес вычисление `hasClientFilters` в отдельный `useMemo` на уровне компонента, чтобы избежать дублирования:

```typescript
// Вычисляем наличие клиентских фильтров один раз
const hasClientFilters = useMemo(() => {
  return selectedSource || selectedCountry || qualityFilter !== 'all'
}, [selectedSource, selectedCountry, qualityFilter])
```

Теперь эта переменная используется в обоих местах:
- В `fetchCounterparties` для определения стратегии загрузки данных
- В `filteredTotalPages` useMemo для пересчета пагинации

## Изменения

1. ✅ Добавлен `useMemo` для вычисления `hasClientFilters` на уровне компонента
2. ✅ Удалено дублирующее объявление из `fetchCounterparties`
3. ✅ Удалено дублирующее объявление из `filteredTotalPages` useMemo
4. ✅ Обновлены зависимости в `useCallback` и `useMemo`

## Результат

✅ **Ошибка компиляции исправлена**

- Нет дублирования переменных
- Код более эффективен (вычисление выполняется один раз)
- Все зависимости правильно указаны

**Статус:** ✅ Готово

