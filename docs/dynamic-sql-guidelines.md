# Guidelines for Dynamic SQL Queries

## Обзор

Этот документ описывает best practices для работы с динамическими SQL запросами в проекте, чтобы предотвратить SQL injection и обеспечить безопасность.

## Проблема

Динамические SQL запросы, построенные с использованием конкатенации строк, могут быть уязвимы к SQL injection атакам, если имена таблиц или колонок приходят из пользовательского ввода.

## Решение

### 1. Валидация имен таблиц и колонок

Все динамические SQL запросы должны использовать валидацию через функции из `database/validation.go`:

```go
import "httpserver/database"

// ✅ ПРАВИЛЬНО: Валидация перед использованием
func (s *Service) GetData(tableName, columnName string) error {
    if !database.IsValidTableName(tableName) {
        return fmt.Errorf("invalid table name: %s", tableName)
    }
    if !database.IsValidColumnName(columnName) {
        return fmt.Errorf("invalid column name: %s", columnName)
    }
    
    query := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", tableName, columnName)
    // ... выполнение запроса
}
```

### 2. Использование whitelist для таблиц

Все допустимые имена таблиц должны быть добавлены в `allowedTableNames` в `database/validation.go`:

```go
var allowedTableNames = map[string]bool{
    "uploads": true,
    "clients": true,
    // ... другие таблицы
}
```

### 3. Безопасная конкатенация условий WHERE

При построении WHERE условий используйте `strings.Join` вместо прямой конкатенации:

```go
// ✅ ПРАВИЛЬНО: Использование strings.Join
conditions := []string{}
args := []interface{}{}

if name != "" {
    conditions = append(conditions, "name = ?")
    args = append(args, name)
}
if status != "" {
    conditions = append(conditions, "status = ?")
    args = append(args, status)
}

if len(conditions) == 0 {
    return fmt.Errorf("no criteria specified")
}

whereClause := strings.Join(conditions, " AND ")
query := fmt.Sprintf("SELECT * FROM table WHERE %s", whereClause)
```

```go
// ❌ НЕПРАВИЛЬНО: Прямая конкатенация
query := fmt.Sprintf("SELECT * FROM table WHERE name = '%s' AND status = '%s'", name, status)
```

### 4. Использование параметризованных запросов

Всегда используйте параметризованные запросы для значений:

```go
// ✅ ПРАВИЛЬНО: Параметризованный запрос
query := "SELECT * FROM users WHERE id = ?"
rows, err := db.Query(query, userID)

// ❌ НЕПРАВИЛЬНО: Конкатенация значений
query := fmt.Sprintf("SELECT * FROM users WHERE id = %d", userID)
rows, err := db.Query(query)
```

### 5. Валидация для IN запросов

При использовании `IN` запросов создавайте плейсхолдеры безопасно:

```go
// ✅ ПРАВИЛЬНО: Безопасное создание IN запроса
ids := []int{1, 2, 3}
placeholders := make([]string, len(ids))
args := make([]interface{}, len(ids))
for i, id := range ids {
    placeholders[i] = "?"
    args[i] = id
}
query := fmt.Sprintf("SELECT * FROM table WHERE id IN (%s)", strings.Join(placeholders, ","))
rows, err := db.Query(query, args...)
```

## Примеры из кодовой базы

### Пример 1: ClassificationService.ResetClassification

```go
func (cs *ClassificationService) ResetClassification(...) (int64, error) {
    var conditions []string
    var args []interface{}
    
    // Добавление условий
    if normalizedName != "" {
        conditions = append(conditions, "normalized_name = ?")
        args = append(args, normalizedName)
    }
    
    // Безопасная конкатенация
    whereClause := strings.Join(conditions, " AND ")
    query := fmt.Sprintf(`UPDATE normalized_data 
        SET kpved_code = NULL 
        WHERE kpved_code IS NOT NULL AND %s`, whereClause)
    
    result, err := cs.db.Exec(query, args...)
    // ...
}
```

### Пример 2: TableAnalyzer с валидацией

```go
func (ta *TableAnalyzer) AnalyzeTableForDuplicates(
    tableName, codeColumn, nameColumn string,
) error {
    // Валидация имен
    if !database.IsValidTableName(tableName) {
        return fmt.Errorf("invalid table name: %s", tableName)
    }
    if !database.IsValidColumnName(codeColumn) {
        return fmt.Errorf("invalid column name: %s", codeColumn)
    }
    
    // Безопасное использование
    query := fmt.Sprintf("SELECT %s, %s FROM %s", codeColumn, nameColumn, tableName)
    // ...
}
```

## Чеклист для проверки

При написании динамических SQL запросов убедитесь:

- [ ] Все имена таблиц валидируются через `database.IsValidTableName()`
- [ ] Все имена колонок валидируются через `database.IsValidColumnName()`
- [ ] WHERE условия собираются через `strings.Join()`, а не конкатенацию
- [ ] Все значения передаются как параметры (`?`), а не встраиваются в строку
- [ ] IN запросы создаются с использованием плейсхолдеров
- [ ] Добавлены тесты для проверки валидации

## Дополнительные ресурсы

- [OWASP SQL Injection Prevention](https://cheatsheetseries.owasp.org/cheatsheets/SQL_Injection_Prevention_Cheat_Sheet.html)
- [Go database/sql Best Practices](https://go.dev/doc/database/sql-injection)

