#!/bin/bash

# Тестирование исправленных эндпоинтов
# Использование: ./test_fixed_endpoints.sh [PORT]
# По умолчанию: PORT=9999

PORT=${1:-9999}
BASE_URL="http://localhost:${PORT}"

echo "=========================================="
echo "Тестирование исправленных эндпоинтов"
echo "=========================================="
echo ""

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для тестирования эндпоинта
test_endpoint() {
    local name=$1
    local url=$2
    local expected_status=${3:-200}
    
    echo -n "Тест: $name ... "
    
    response=$(curl -s -w "\n%{http_code}" "$url" --max-time 7)
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "$expected_status" ]; then
        echo -e "${GREEN}✓ Success ${http_code}${NC}"
        if [ -n "$body" ] && [ "$body" != "null" ]; then
            echo "$body" | head -c 200
            echo ""
        fi
        return 0
    else
        echo -e "${RED}✗ Failed ${http_code} (expected ${expected_status})${NC}"
        if [ -n "$body" ]; then
            echo "Response: $body"
        fi
        return 1
    fi
}

# Тест 1: GET /api/databases/find-project
echo "1. Testing GET /api/databases/find-project"
test_endpoint "Find project by database path" \
    "${BASE_URL}/api/databases/find-project?file_path=Выгрузка_Номенклатура_ERPWE_Unknown_Unknown_2025_11_20_10_18_55.db" \
    200
echo ""

# Тест 2: GET /api/databases/find-project (несуществующая БД)
echo "2. Testing GET /api/databases/find-project (non-existent DB)"
test_endpoint "Find project by non-existent database path" \
    "${BASE_URL}/api/databases/find-project?file_path=non_existent.db" \
    404
echo ""

# Тест 3: GET /api/databases/find-project (без параметра)
echo "3. Testing GET /api/databases/find-project (missing parameter)"
test_endpoint "Find project without file_path parameter" \
    "${BASE_URL}/api/databases/find-project" \
    400
echo ""

# Тест 4: GET /api/normalization/status
echo "4. Testing GET /api/normalization/status"
test_endpoint "Get normalization status" \
    "${BASE_URL}/api/normalization/status" \
    200
echo ""

# Тест 5: GET /api/dashboard/normalization-status
echo "5. Testing GET /api/dashboard/normalization-status"
test_endpoint "Get dashboard normalization status" \
    "${BASE_URL}/api/dashboard/normalization-status" \
    200
echo ""

echo "=========================================="
echo "Тестирование завершено"
echo "=========================================="

