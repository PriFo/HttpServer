#!/bin/bash
# Скрипт для тестирования нормализации контрагентов на всех базах данных
# Использование: ./scripts/test_all_databases.sh

echo "=== Тестирование нормализации контрагентов на всех базах данных ==="

# Находим все базы данных с контрагентами
DB_PATHS=()

# Директории для поиска
SEARCH_DIRS=("." "data" "data/uploads")

for dir in "${SEARCH_DIRS[@]}"; do
    if [ -d "$dir" ]; then
        while IFS= read -r -d '' file; do
            basename=$(basename "$file")
            if [ "$basename" != "service.db" ] && [ "$basename" != "test.db" ]; then
                DB_PATHS+=("$file")
            fi
        done < <(find "$dir" -name "*.db" -type f -print0 2>/dev/null)
    fi
done

echo "Найдено баз данных: ${#DB_PATHS[@]}"

if [ ${#DB_PATHS[@]} -eq 0 ]; then
    echo "Базы данных не найдены!"
    exit 1
fi

# Выводим список найденных баз
echo ""
echo "Список баз данных для тестирования:"
for db_path in "${DB_PATHS[@]}"; do
    echo "  - $db_path"
done

# Запускаем тесты
echo ""
echo "Запуск интеграционных тестов..."
export TEST_ALL_DATABASES=true

# Запускаем тест с таймаутом для каждой базы
go test ./normalization -v -run "^TestCounterpartyNormalization_AllDatabases$" -timeout 5m -short=false

if [ $? -eq 0 ]; then
    echo ""
    echo "=== Все тесты пройдены успешно! ==="
else
    echo ""
    echo "=== Некоторые тесты провалились ==="
    exit 1
fi

