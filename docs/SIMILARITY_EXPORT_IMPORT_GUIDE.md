# Руководство по экспорту и импорту данных

## Обзор

Система поддерживает экспорт результатов анализа в различных форматах и импорт обучающих данных для обучения алгоритмов.

## Экспорт результатов

### API эндпоинт

**POST** `/api/similarity/export`

Экспортирует результаты анализа в указанном формате.

#### Запрос

```json
{
  "pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "Рога и Копыта ООО"
    }
  ],
  "format": "json",
  "threshold": 0.75,
  "weights": {
    "jaro_winkler": 0.3,
    "lcs": 0.2,
    "phonetic": 0.2,
    "ngram": 0.2,
    "jaccard": 0.1
  }
}
```

#### Поддерживаемые форматы

- **json** - JSON формат с полной информацией
- **csv** - CSV формат для Excel/Google Sheets
- **tsv** - TSV формат (табуляция)
- **report** - Markdown отчет с метриками

#### Ответ

```json
{
  "filepath": "exports/similarity_export_20250120_153045.json",
  "format": "json",
  "count": 1,
  "message": "Export completed successfully"
}
```

### Примеры использования

#### Экспорт в JSON

```bash
curl -X POST http://localhost:9999/api/similarity/export \
  -H "Content-Type: application/json" \
  -d '{
    "pairs": [
      {"s1": "ООО Рога и Копыта", "s2": "Рога и Копыта ООО"}
    ],
    "format": "json"
  }'
```

#### Экспорт в CSV

```bash
curl -X POST http://localhost:9999/api/similarity/export \
  -H "Content-Type: application/json" \
  -d '{
    "pairs": [...],
    "format": "csv"
  }'
```

#### Экспорт отчета

```bash
curl -X POST http://localhost:9999/api/similarity/export \
  -H "Content-Type: application/json" \
  -d '{
    "pairs": [...],
    "format": "report"
  }'
```

### Формат CSV

CSV файл содержит следующие колонки:

- `String1` - Первая строка
- `String2` - Вторая строка
- `Similarity` - Общая схожесть
- `IsDuplicate` - Является ли дубликатом
- `Confidence` - Уверенность в результате
- `JaroWinkler` - Схожесть по Jaro-Winkler
- `LCS` - Схожесть по LCS
- `Phonetic` - Схожесть по фонетическим алгоритмам
- `Ngram` - Схожесть по N-граммам
- `Jaccard` - Схожесть по Jaccard

### Формат отчета (Markdown)

Отчет включает:
- Статистику анализа
- Рекомендации
- Таблицу с деталями всех пар

## Импорт обучающих данных

### API эндпоинт

**POST** `/api/similarity/import`

Импортирует обучающие пары из файла.

#### Запрос (multipart/form-data)

```
POST /api/similarity/import
Content-Type: multipart/form-data

file: [файл .json или .csv]
```

#### Поддерживаемые форматы

- **JSON** - Массив объектов `{s1, s2, is_duplicate}`
- **CSV** - Файл с колонками `String1,String2,IsDuplicate`

#### Формат JSON

```json
[
  {
    "s1": "ООО Рога и Копыта",
    "s2": "ООО Рога и Копыта",
    "is_duplicate": true
  },
  {
    "s1": "ООО Рога и Копыта",
    "s2": "ООО Другая Компания",
    "is_duplicate": false
  }
]
```

#### Формат CSV

```csv
String1,String2,IsDuplicate
ООО Рога и Копыта,ООО Рога и Копыта,true
ООО Рога и Копыта,ООО Другая Компания,false
```

#### Ответ

```json
{
  "pairs_imported": 2,
  "pairs": [
    {
      "s1": "ООО Рога и Копыта",
      "s2": "ООО Рога и Копыта",
      "is_duplicate": true
    }
  ],
  "message": "Import completed successfully"
}
```

### Примеры использования

#### Импорт из JSON

```bash
curl -X POST http://localhost:9999/api/similarity/import \
  -F "file=@training_data.json"
```

#### Импорт из CSV

```bash
curl -X POST http://localhost:9999/api/similarity/import \
  -F "file=@training_data.csv"
```

#### JavaScript (fetch)

```javascript
const formData = new FormData();
formData.append('file', fileInput.files[0]);

const response = await fetch('http://localhost:9999/api/similarity/import', {
  method: 'POST',
  body: formData
});

const result = await response.json();
console.log('Imported pairs:', result.pairs);
```

## Программное использование

### Экспорт

```go
import "httpserver/normalization/algorithms"

// Анализируем пары
analyzer := algorithms.NewSimilarityAnalyzer(weights)
result := analyzer.AnalyzePairs(pairs, 0.75)

// Экспортируем
exporter := algorithms.NewSimilarityExporter(result)
exporter.Export("results.json", algorithms.ExportFormatJSON)
exporter.ExportReport("report.md")
```

### Импорт

```go
// Импортируем обучающие пары
pairs, err := algorithms.ImportTrainingPairs("training.json", algorithms.ExportFormatJSON)
if err != nil {
    log.Fatal(err)
}

// Используем для обучения
learner := algorithms.NewSimilarityLearner()
learner.AddTrainingPairs(pairs)
weights, err := learner.OptimizeWeights(100, 0.01)
```

## Рекомендации

1. **Для анализа в Excel/Google Sheets** используйте формат CSV
2. **Для интеграции с другими системами** используйте JSON
3. **Для документирования результатов** используйте формат отчета (Markdown)
4. **Для больших объемов данных** используйте TSV (быстрее парсится)

## Ограничения

- Максимальный размер импортируемого файла: 10MB
- Максимальное количество пар для экспорта: 500
- Поддерживаются только UTF-8 кодировка

## См. также

- [SIMILARITY_API.md](./SIMILARITY_API.md) - Полная документация API
- [SIMILARITY_ANALYSIS_GUIDE.md](./SIMILARITY_ANALYSIS_GUIDE.md) - Руководство по анализу
- [SIMILARITY_LEARNING_GUIDE.md](./SIMILARITY_LEARNING_GUIDE.md) - Руководство по обучению

