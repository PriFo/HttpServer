#!/bin/bash

# Очистка решенных задач старше N дней
# Удаляет задачи со статусом RESOLVED, которые были решены более указанного количества дней назад

set -e

TODO_DB=".todos/tasks.json"
DAYS_OLD="${1:-30}"  # По умолчанию 30 дней

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🧹 Очистка решенных задач старше $DAYS_OLD дней...${NC}"

if [[ ! -f "$TODO_DB" ]]; then
    echo -e "${RED}✗${NC} Файл $TODO_DB не найден"
    exit 1
fi

# Получаем текущую дату в секундах
CURRENT_DATE=$(date +%s)
CUTOFF_DATE=$((CURRENT_DATE - DAYS_OLD * 86400))

# Подсчитываем задачи для удаления
RESOLVED_TASKS=$(jq '[.tasks[] | select(.status == "RESOLVED")]' "$TODO_DB")
TOTAL_RESOLVED=$(echo "$RESOLVED_TASKS" | jq 'length')

if [[ $TOTAL_RESOLVED -eq 0 ]]; then
    echo -e "${GREEN}✓${NC} Нет решенных задач для удаления"
    exit 0
fi

# Фильтруем задачи по дате обновления
TASKS_TO_REMOVE=0
TASK_IDS=()

while IFS= read -r task; do
    UPDATED_AT=$(echo "$task" | jq -r '.updatedAt // .createdAt')
    
    # Парсим ISO дату и конвертируем в секунды
    if [[ "$UPDATED_AT" != "null" && -n "$UPDATED_AT" ]]; then
        # Простая проверка (можно улучшить парсинг ISO даты)
        TASK_DATE=$(date -d "$UPDATED_AT" +%s 2>/dev/null || echo "0")
        
        if [[ $TASK_DATE -lt $CUTOFF_DATE ]]; then
            TASK_ID=$(echo "$task" | jq -r '.id')
            TASK_IDS+=("$TASK_ID")
            ((TASKS_TO_REMOVE++))
        fi
    fi
done < <(echo "$RESOLVED_TASKS" | jq -c '.[]')

if [[ $TASKS_TO_REMOVE -eq 0 ]]; then
    echo -e "${GREEN}✓${NC} Нет задач старше $DAYS_OLD дней для удаления"
    exit 0
fi

echo -e "${YELLOW}⚠${NC}  Найдено $TASKS_TO_REMOVE задач для удаления"

# Удаляем задачи
for task_id in "${TASK_IDS[@]}"; do
    jq ".tasks = (.tasks | map(select(.id != \"$task_id\")))" "$TODO_DB" > "${TODO_DB}.tmp" && mv "${TODO_DB}.tmp" "$TODO_DB"
done

echo -e "${GREEN}✓${NC} Удалено $TASKS_TO_REMOVE задач"
echo -e "${BLUE}📊${NC} Осталось задач: $(jq '.tasks | length' "$TODO_DB")"

