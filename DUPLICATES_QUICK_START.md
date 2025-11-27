# 🚀 Быстрый старт: Использование системы обнаружения дублей

## Дата: 2025-01-20

---

## ⚡ За 5 минут

### 1. Проверка готовности

```bash
# Проверить, что система работает
curl http://localhost:8080/api/quality/duplicates?database=test.db
```

**Ожидаемый результат**: JSON с группами дубликатов или пустой массив

---

### 2. Поиск дублей через API

#### Для номенклатуры:

```bash
# Получить все группы дубликатов
curl "http://localhost:8080/api/quality/duplicates?database=test.db&limit=10"

# Только необъединенные
curl "http://localhost:8080/api/quality/duplicates?database=test.db&unmerged=true"

# С пагинацией
curl "http://localhost:8080/api/quality/duplicates?database=test.db&limit=20&offset=0"
```

#### Для контрагентов:

```bash
# Получить дубли контрагентов
curl "http://localhost:8080/api/counterparties/duplicates?project_id=1"
```

---

### 3. Объединение дубликатов

```bash
# Объединить группу дубликатов номенклатуры
curl -X POST "http://localhost:8080/api/quality/duplicates/123/merge" \
  -H "Content-Type: application/json"

# Объединить дубли контрагентов
curl -X POST "http://localhost:8080/api/counterparties/duplicates/456/merge" \
  -H "Content-Type: application/json"
```

---

## 📝 Примеры использования в коде

### Go (Backend)

```go
package main

import (
    "fmt"
    "github.com/yourproject/normalization"
)

func main() {
    // Создаем нормализатор
    nsi := normalization.NewNSINormalizer()
    
    // Подготавливаем данные
    items := []normalization.DuplicateItem{
        {
            ID:   1,
            Code: "001",
            Name: "Масло сливочное",
        },
        {
            ID:   2,
            Code: "002",
            Name: "масло сливочное", // Дубликат
        },
    }
    
    // Конфигурация
    config := normalization.DuplicateDetectionConfig{
        UseExactMatching:  true,
        UseFuzzyMatching:  true,
        MinConfidence:     0.8,
        MergeOverlapping:  true,
    }
    
    // Ищем дубликаты
    groups := nsi.FindDuplicates(items, config)
    
    // Выводим результаты
    for _, group := range groups {
        fmt.Printf("Группа: %d элементов, уверенность: %.2f\n", 
            len(group.Items), group.Confidence)
    }
}
```

---

### JavaScript/TypeScript (Frontend)

```typescript
// Получение дубликатов
async function getDuplicates(database: string) {
  const response = await fetch(
    `/api/quality/duplicates?database=${database}&limit=50`
  );
  const data = await response.json();
  return data.groups;
}

// Объединение дубликатов
async function mergeDuplicates(groupId: number) {
  const response = await fetch(
    `/api/quality/duplicates/${groupId}/merge`,
    { method: 'POST' }
  );
  return await response.json();
}

// Использование
const duplicates = await getDuplicates('test.db');
console.log('Найдено групп:', duplicates.length);

// Объединить первую группу
if (duplicates.length > 0) {
  await mergeDuplicates(duplicates[0].id);
}
```

---

## 🎯 Типичные сценарии

### Сценарий 1: Проверка новых данных

```bash
# 1. Загрузить данные в БД
# 2. Запустить нормализацию
# 3. Проверить дубликаты
curl "http://localhost:8080/api/quality/duplicates?database=new_data.db"
```

---

### Сценарий 2: Ежедневная проверка

```bash
# Создать скрипт для автоматической проверки
#!/bin/bash
DATABASE="production.db"
RESULT=$(curl -s "http://localhost:8080/api/quality/duplicates?database=$DATABASE&unmerged=true")
COUNT=$(echo $RESULT | jq '.total_groups')

if [ "$COUNT" -gt 0 ]; then
    echo "Найдено $COUNT групп дубликатов!"
    # Отправить уведомление
fi
```

---

### Сценарий 3: Массовое объединение

```typescript
// Объединить все группы с высокой уверенностью
async function mergeHighConfidenceDuplicates() {
  const duplicates = await getDuplicates('test.db');
  
  for (const group of duplicates) {
    if (group.confidence >= 0.9) {
      await mergeDuplicates(group.id);
      console.log(`Объединена группа ${group.id}`);
    }
  }
}
```

---

## ⚙️ Конфигурация

### Параметры поиска дубликатов

```go
config := normalization.DuplicateDetectionConfig{
    // Exact matching
    UseExactMatching: true,
    
    // Fuzzy matching
    UseFuzzyMatching: true,
    FuzzyThreshold:   0.8,
    
    // Минимальная уверенность
    MinConfidence: 0.7,
    
    // Объединение пересекающихся групп
    MergeOverlapping: true,
    
    // Веса алгоритмов
    SimilarityWeights: normalization.SimilarityWeights{
        Levenshtein:     0.3,
        NGram:           0.2,
        Jaccard:         0.2,
        Phonetic:        0.15,
        Semantic:        0.15,
    },
}
```

---

## 📊 Интерпретация результатов

### Структура ответа API

```json
{
  "groups": [
    {
      "id": 123,
      "confidence": 0.95,
      "items": [
        {
          "id": 1,
          "code": "001",
          "name": "Масло сливочное",
          "normalized_name": "масло сливочное"
        },
        {
          "id": 2,
          "code": "002",
          "name": "масло сливочное",
          "normalized_name": "масло сливочное"
        }
      ],
      "master_id": 1,
      "merged": false
    }
  ],
  "total_groups": 1,
  "total_duplicates": 2
}
```

### Поля ответа

- **id**: Уникальный идентификатор группы
- **confidence**: Уверенность (0.0 - 1.0)
- **items**: Список элементов в группе
- **master_id**: ID мастер-записи
- **merged**: Объединена ли группа

---

## 🔍 Отладка

### Проверка логов

```bash
# Включить детальное логирование
export LOG_LEVEL=debug

# Проверить работу алгоритмов
curl -v "http://localhost:8080/api/quality/duplicates?database=test.db"
```

---

### Тестирование алгоритмов

```go
// Тест конкретного алгоритма
fuzzy := normalization.NewFuzzyAlgorithms()
similarity := fuzzy.LevenshteinSimilarity("масло", "масло")
fmt.Printf("Схожесть: %.2f\n", similarity) // Ожидается: 1.0
```

---

## ⚠️ Частые проблемы

### Проблема 1: Нет результатов

**Причина**: Данные не нормализованы

**Решение**: Запустить нормализацию перед поиском дубликатов

---

### Проблема 2: Много ложных срабатываний

**Причина**: Слишком низкий порог

**Решение**: Увеличить `MinConfidence` до 0.8-0.9

---

### Проблема 3: Пропущены дубликаты

**Причина**: Слишком высокий порог

**Решение**: Уменьшить `MinConfidence` до 0.6-0.7

---

## 📚 Дополнительная документация

- **Детальная документация**: `DUPLICATES_README.md`
- **Примеры кода**: `DUPLICATES_PRACTICAL_EXAMPLES.md`
- **API документация**: `DUPLICATES_API_USAGE_ANALYSIS.md`
- **Тестовые сценарии**: `DUPLICATES_TEST_SCENARIOS.md`

---

## ✅ Чек-лист быстрого старта

- [ ] Система запущена и доступна
- [ ] Проверен API endpoint `/api/quality/duplicates`
- [ ] Получены первые результаты
- [ ] Протестировано объединение дубликатов
- [ ] Настроена конфигурация под ваши данные

---

**Готово! Система готова к использованию! 🚀**

