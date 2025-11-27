#!/bin/bash

# Тестовый скрипт для проверки системы TODO

set -e

PROJECT_DIR="${1:-.}"
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🧪 Тестирование системы TODO...${NC}"
echo ""

# Проверка наличия необходимых инструментов
echo -e "${BLUE}Проверка зависимостей:${NC}"

check_command() {
    if command -v "$1" &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} $1 установлен"
        return 0
    else
        echo -e "  ${RED}✗${NC} $1 не установлен"
        return 1
    fi
}

ERRORS=0

check_command "bash" || ((ERRORS++))
check_command "jq" || ((ERRORS++))
check_command "node" || ((ERRORS++))
check_command "python3" || echo -e "  ${YELLOW}⚠${NC} python3 не установлен (опционально для автоназначения)"

echo ""

# Проверка структуры файлов
echo -e "${BLUE}Проверка структуры:${NC}"

check_file() {
    if [[ -f "$1" ]]; then
        echo -e "  ${GREEN}✓${NC} $1 существует"
        return 0
    else
        echo -e "  ${RED}✗${NC} $1 не найден"
        return 1
    fi
}

check_file "${PROJECT_DIR}/.todos/tasks.json" || ((ERRORS++))
check_file "${PROJECT_DIR}/.todos/team.json" || ((ERRORS++))
check_file "${PROJECT_DIR}/.todos/config.json" || ((ERRORS++))
check_file "${PROJECT_DIR}/.todos/scripts/scan-todos.sh" || ((ERRORS++))
check_file "${PROJECT_DIR}/.todos/scripts/generate-report.js" || ((ERRORS++))
check_file "${PROJECT_DIR}/.todos/dashboard.html" || ((ERRORS++))

echo ""

# Проверка JSON валидности
echo -e "${BLUE}Проверка валидности JSON:${NC}"

check_json() {
    if jq empty "$1" 2>/dev/null; then
        echo -e "  ${GREEN}✓${NC} $1 валиден"
        return 0
    else
        echo -e "  ${RED}✗${NC} $1 содержит ошибки"
        return 1
    fi
}

check_json "${PROJECT_DIR}/.todos/tasks.json" || ((ERRORS++))
check_json "${PROJECT_DIR}/.todos/team.json" || ((ERRORS++))
check_json "${PROJECT_DIR}/.todos/config.json" || ((ERRORS++))

echo ""

# Итоги
if [[ $ERRORS -eq 0 ]]; then
    echo -e "${GREEN}✅ Все проверки пройдены!${NC}"
    echo ""
    echo -e "${BLUE}Следующие шаги:${NC}"
    echo "  1. Запустите сканирование: npm run todos:scan"
    echo "  2. Откройте дашборд: .todos/dashboard.html"
    echo "  3. Сгенерируйте отчет: npm run todos:report"
    exit 0
else
    echo -e "${RED}❌ Найдено $ERRORS ошибок${NC}"
    exit 1
fi

