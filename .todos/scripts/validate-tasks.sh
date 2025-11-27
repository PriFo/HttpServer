#!/bin/bash

# Валидация задач в tasks.json
# Проверяет корректность структуры данных

set -e

TODO_DB=".todos/tasks.json"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🔍 Валидация задач...${NC}"

if [[ ! -f "$TODO_DB" ]]; then
    echo -e "${RED}✗${NC} Файл $TODO_DB не найден"
    exit 1
fi

# Проверка валидности JSON
if ! jq empty "$TODO_DB" 2>/dev/null; then
    echo -e "${RED}✗${NC} JSON содержит ошибки"
    jq . "$TODO_DB" 2>&1 | head -5
    exit 1
fi

echo -e "${GREEN}✓${NC} JSON валиден"

# Проверка структуры
if ! jq -e '.tasks' "$TODO_DB" > /dev/null 2>&1; then
    echo -e "${RED}✗${NC} Отсутствует поле 'tasks'"
    exit 1
fi

if ! jq -e '.metadata' "$TODO_DB" > /dev/null 2>&1; then
    echo -e "${YELLOW}⚠${NC} Отсутствует поле 'metadata' (старая структура)"
    echo "   Рекомендуется запустить сканирование для миграции"
fi

# Статистика
TOTAL=$(jq '.tasks | length' "$TODO_DB")
OPEN=$(jq '[.tasks[] | select(.status == "OPEN")] | length' "$TODO_DB")
CRITICAL=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' "$TODO_DB")

echo ""
echo -e "${BLUE}📊 Статистика:${NC}"
echo "  Всего задач: $TOTAL"
echo "  Открытых: $OPEN"
echo "  Критических: $CRITICAL"

# Проверка обязательных полей в задачах
echo ""
echo -e "${BLUE}Проверка структуры задач...${NC}"

REQUIRED_FIELDS=("id" "file" "line" "description" "type" "priority" "status")
ERRORS=0

while IFS= read -r task; do
    for field in "${REQUIRED_FIELDS[@]}"; do
        if ! echo "$task" | jq -e ".$field" > /dev/null 2>&1; then
            TASK_ID=$(echo "$task" | jq -r '.id // "unknown"')
            echo -e "${RED}✗${NC} Задача $TASK_ID: отсутствует поле '$field'"
            ((ERRORS++))
        fi
    done
done < <(jq -c '.tasks[]' "$TODO_DB")

if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}✓${NC} Все задачи имеют обязательные поля"
else
    echo -e "${RED}✗${NC} Найдено $ERRORS ошибок в структуре задач"
    exit 1
fi

# Проверка уникальности ID
DUPLICATES=$(jq -r '.tasks[].id' "$TODO_DB" | sort | uniq -d | wc -l)
if [[ $DUPLICATES -gt 0 ]]; then
    echo -e "${RED}✗${NC} Найдено дублирующихся ID: $DUPLICATES"
    exit 1
else
    echo -e "${GREEN}✓${NC} Все ID уникальны"
fi

echo ""
echo -e "${GREEN}✅ Валидация завершена успешно!${NC}"

