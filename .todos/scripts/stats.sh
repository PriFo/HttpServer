#!/bin/bash

# Быстрая статистика по задачам

set -e

TODO_DB=".todos/tasks.json"

if [[ ! -f "$TODO_DB" ]]; then
    echo "❌ Файл $TODO_DB не найден"
    exit 1
fi

echo "📊 Статистика TODO задач"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Общая статистика
TOTAL=$(jq '.tasks | length' "$TODO_DB")
OPEN=$(jq '[.tasks[] | select(.status == "OPEN")] | length' "$TODO_DB")
IN_PROGRESS=$(jq '[.tasks[] | select(.status == "IN_PROGRESS")] | length' "$TODO_DB")
RESOLVED=$(jq '[.tasks[] | select(.status == "RESOLVED")] | length' "$TODO_DB")

echo ""
echo "📈 Общая статистика:"
echo "   Всего задач: $TOTAL"
echo "   Открытых: $OPEN"
echo "   В работе: $IN_PROGRESS"
echo "   Решено: $RESOLVED"

# По приоритетам
CRITICAL=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' "$TODO_DB")
HIGH=$(jq '[.tasks[] | select(.priority == "HIGH" and .status == "OPEN")] | length' "$TODO_DB")
MEDIUM=$(jq '[.tasks[] | select(.priority == "MEDIUM" and .status == "OPEN")] | length' "$TODO_DB")
LOW=$(jq '[.tasks[] | select(.priority == "LOW" and .status == "OPEN")] | length' "$TODO_DB")

echo ""
echo "🎯 По приоритетам (открытые):"
echo "   🔴 CRITICAL: $CRITICAL"
echo "   🟠 HIGH: $HIGH"
echo "   🟡 MEDIUM: $MEDIUM"
echo "   🟢 LOW: $LOW"

# По типам
TODO_COUNT=$(jq '[.tasks[] | select(.type == "TODO")] | length' "$TODO_DB")
FIXME_COUNT=$(jq '[.tasks[] | select(.type == "FIXME")] | length' "$TODO_DB")
HACK_COUNT=$(jq '[.tasks[] | select(.type == "HACK")] | length' "$TODO_DB")
REFACTOR_COUNT=$(jq '[.tasks[] | select(.type == "REFACTOR")] | length' "$TODO_DB")

echo ""
echo "📝 По типам:"
echo "   TODO: $TODO_COUNT"
echo "   FIXME: $FIXME_COUNT"
echo "   HACK: $HACK_COUNT"
echo "   REFACTOR: $REFACTOR_COUNT"

# По типам файлов
BACKEND=$(jq '[.tasks[] | select(.fileType == "backend")] | length' "$TODO_DB")
FRONTEND=$(jq '[.tasks[] | select(.fileType == "frontend")] | length' "$TODO_DB")
SCRIPT=$(jq '[.tasks[] | select(.fileType == "script")] | length' "$TODO_DB")
OTHER=$(jq '[.tasks[] | select(.fileType == "other")] | length' "$TODO_DB")

echo ""
echo "💻 По типам файлов:"
echo "   Backend: $BACKEND"
echo "   Frontend: $FRONTEND"
echo "   Scripts: $SCRIPT"
echo "   Other: $OTHER"

# Последнее сканирование
LAST_SCAN=$(jq -r '.metadata.lastScan // "Never"' "$TODO_DB")
TOTAL_SCANS=$(jq -r '.metadata.totalScans // 0' "$TODO_DB")

echo ""
echo "🕐 Сканирование:"
echo "   Последнее: $LAST_SCAN"
echo "   Всего сканирований: $TOTAL_SCANS"

# Критические задачи
if [[ $CRITICAL -gt 0 ]]; then
    echo ""
    echo "🚨 Критические задачи:"
    jq -r '.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN") | "   - \(.file):\(.line) - \(.description)"' "$TODO_DB" | head -5
    if [[ $CRITICAL -gt 5 ]]; then
        echo "   ... и еще $((CRITICAL - 5)) задач"
    fi
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

