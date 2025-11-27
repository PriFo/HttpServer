#!/bin/bash
# Pre-commit hook для проверки критических TODO
# Использование: cp .todos/scripts/pre-commit-check.sh .git/hooks/pre-commit

PROJECT_ROOT="$(git rev-parse --show-toplevel)"
TODO_DB="$PROJECT_ROOT/.todos/tasks.json"

if [ ! -f "$TODO_DB" ]; then
    exit 0
fi

# Проверка наличия jq
if ! command -v jq &> /dev/null; then
    # Если jq нет, используем Python
    if command -v python3 &> /dev/null; then
        CRITICAL_COUNT=$(python3 -c "
import json
import sys
try:
    with open('$TODO_DB', 'r') as f:
        data = json.load(f)
    critical = [t for t in data.get('tasks', []) if t.get('priority') == 'CRITICAL' and t.get('status') == 'OPEN']
    print(len(critical))
except:
    print(0)
")
    else
        exit 0
    fi
else
    CRITICAL_COUNT=$(jq '[.tasks[] | select(.priority == "CRITICAL" and .status == "OPEN")] | length' "$TODO_DB")
fi

if [ "$CRITICAL_COUNT" -gt 0 ]; then
    echo "🚨 Найдено $CRITICAL_COUNT критических TODO. Пожалуйста, исправьте перед коммитом."
    echo "Используйте 'git commit --no-verify' чтобы обойти проверку."
    exit 1
fi

exit 0


