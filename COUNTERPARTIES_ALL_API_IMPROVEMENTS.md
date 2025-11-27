# Улучшения API получения всех контрагентов

## Добавлена сортировка результатов

### Описание
Добавлена интеллектуальная сортировка объединенного списка контрагентов перед применением пагинации.

### Логика сортировки

1. **Приоритет нормализованных записей**
   - Контрагенты с `quality_score` (нормализованные) располагаются выше записей без оценки качества
   - Это позволяет пользователю видеть более качественные данные в первую очередь

2. **Сортировка по качеству**
   - Если оба контрагента имеют `quality_score`, они сортируются по убыванию качества
   - Более качественные записи (с более высоким `quality_score`) идут первыми

3. **Алфавитная сортировка**
   - После сортировки по качеству применяется алфавитная сортировка по имени
   - Сортировка выполняется без учета регистра (`strings.ToLower`)

4. **Стабильность сортировки**
   - Если имена совпадают, используется сортировка по ID
   - Это обеспечивает стабильный порядок результатов при повторных запросах

### Реализация

```go
sort.Slice(allCounterparties, func(i, j int) bool {
    // 1. Приоритет нормализованным записям
    if allCounterparties[i].QualityScore != nil && allCounterparties[j].QualityScore == nil {
        return true
    }
    if allCounterparties[i].QualityScore == nil && allCounterparties[j].QualityScore != nil {
        return false
    }
    // 2. Сортировка по качеству
    if allCounterparties[i].QualityScore != nil && allCounterparties[j].QualityScore != nil {
        if *allCounterparties[i].QualityScore != *allCounterparties[j].QualityScore {
            return *allCounterparties[i].QualityScore > *allCounterparties[j].QualityScore
        }
    }
    // 3. Алфавитная сортировка
    nameI := strings.ToLower(allCounterparties[i].Name)
    nameJ := strings.ToLower(allCounterparties[j].Name)
    if nameI != nameJ {
        return nameI < nameJ
    }
    // 4. Стабильность по ID
    return allCounterparties[i].ID < allCounterparties[j].ID
})
```

### Преимущества

1. **Улучшенный UX**: Пользователи видят наиболее качественные данные первыми
2. **Предсказуемость**: Стабильный порядок результатов при повторных запросах
3. **Удобство навигации**: Алфавитная сортировка упрощает поиск нужных контрагентов
4. **Приоритет нормализованным данным**: Нормализованные записи (с обогащением и проверкой) отображаются выше исходных

### Измененные файлы

- `database/service_db.go`: Добавлен импорт `sort` и логика сортировки
- `api_tests/COUNTERPARTIES_ALL_API.md`: Обновлена документация
- `COUNTERPARTIES_ALL_API_IMPLEMENTATION.md`: Обновлена документация

### Проверка

✅ Код компилируется успешно  
✅ Линтер не выявил ошибок  
✅ Сортировка применяется перед пагинацией

