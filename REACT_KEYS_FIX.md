# Исправление проблем с дублирующимися ключами React

## Дата: 2025-01-XX

## Проблема

В приложении возникала ошибка React:
```
Encountered two children with the same key, `normalized_data.db`. 
Keys should be unique so that components maintain their identity across updates.
```

Эта ошибка появлялась в компоненте `DatabaseSelector` на странице `/results` и других страницах, где использовался этот компонент.

## Причина

Проблема возникала, когда несколько баз данных имели одинаковое имя (например, `normalized_data.db` в разных директориях). React требует уникальные ключи для каждого элемента в списке, чтобы правильно отслеживать изменения и обновлять DOM.

## Исправления

### 1. `frontend/components/database-selector.tsx`

**Проблема:** Ключ формировался как `db.path || db.name`, что могло приводить к дубликатам.

**Решение:** Добавлен индекс в ключ для гарантированной уникальности.

```typescript
// До:
<SelectItem key={db.path || `${db.name}-${index}`} value={db.name}>

// После:
.map((db, index) => {
  const uniqueKey = db.path ? `${db.path}-${index}` : `${db.name}-${index}`
  return (
    <SelectItem key={uniqueKey} value={db.name}>
```

**Строки:** 274-277

### 2. `frontend/components/processes/normalization-results-table.tsx`

**Проблема:** Ключ формировался как `${group.normalized_name}|${group.category}`, что могло дублироваться.

**Решение:** Добавлен индекс к React-ключу, при этом `groupKey` сохранен для внутренней логики (Map структур).

```typescript
// До:
{sortedGroups.map((group) => {
  const groupKey = `${group.normalized_name}|${group.category}`
  return (
    <GroupRow key={groupKey} ...>

// После:
{sortedGroups.map((group, index) => {
  const groupKey = `${group.normalized_name}|${group.category}`
  const reactKey = `${groupKey}-${index}`
  return (
    <GroupRow key={reactKey} ...>
```

**Строки:** 723-730

### 3. `frontend/components/quality/quality-report-tab.tsx`

**Проблема:** В трех местах использовались ключи без индекса, что могло приводить к дубликатам.

**Решение:** Добавлен индекс ко всем ключам в таблицах duplicates, completeness и consistency.

#### 3.1. Таблица duplicates (строка 448)
```typescript
// До:
key={duplicate.id || duplicate.group_id}

// После:
key={duplicate.id ? `duplicate-${duplicate.id}-${index}` : `duplicate-${duplicate.group_id}-${index}`}
```

#### 3.2. Таблица completeness (строка 554)
```typescript
// До:
key={item.id || `completeness-${index}`}

// После:
key={item.id ? `completeness-${item.id}-${index}` : `completeness-${index}`}
```

#### 3.3. Таблица consistency (строка 595)
```typescript
// До:
key={item.id || `consistency-${index}`}

// После:
key={item.id ? `consistency-${item.id}-${index}` : `consistency-${index}`}
```

## Оптимизация производительности

### `frontend/components/database-selector.tsx`

Добавлена оптимизация с использованием `useMemo` для обработки списка баз данных:

```typescript
// Оптимизация: мемоизируем обработку списка баз данных
const processedDatabases = useMemo(() => {
  // Убираем дубликаты по имени, предпочитая текущую базу данных
  return databases.reduce((acc, db) => {
    // ... логика обработки
  }, [] as DatabaseInfo[])
}, [databases])
```

Это предотвращает повторные вычисления при каждом рендере компонента, улучшая производительность.

**Строки:** 239-253

## Результаты

✅ Ошибка "Encountered two children with the same key" полностью устранена  
✅ Все компоненты используют уникальные ключи  
✅ Линтер не выявил ошибок  
✅ Код стал более устойчивым к дубликатам данных  
✅ Приложение работает без предупреждений React  
✅ Добавлена оптимизация производительности с `useMemo`

## Проверенные компоненты (без изменений)

Следующие компоненты уже использовали правильные ключи и не требовали изменений:

- `frontend/components/normalization/data-source-selector.tsx` - использует индекс
- `frontend/app/clients/[clientId]/components/counterparties-tab.tsx` - использует `uniqueKey` с индексом
- `frontend/app/databases/pending/page.tsx` - использует комбинацию с индексом
- `frontend/app/clients/[clientId]/components/databases-tab.tsx` - использует комбинацию с индексом
- `frontend/app/results/page.tsx` - использует `keyExtractor` с индексом

## Рекомендации

1. **Всегда используйте индекс в ключах** при работе с `.map()` в React, даже если данные кажутся уникальными
2. **Комбинируйте несколько полей** с индексом для создания уникальных ключей
3. **Сохраняйте логические ключи** для внутренней логики (Map, Set), но используйте уникальные ключи для React

## Тестирование

Исправления протестированы на:
- Странице `/results` - основная страница с ошибкой
- Консоли браузера - отсутствие ошибок React
- Линтере - отсутствие ошибок TypeScript/ESLint

## Связанные файлы

- `frontend/components/database-selector.tsx`
- `frontend/components/processes/normalization-results-table.tsx`
- `frontend/components/quality/quality-report-tab.tsx`

