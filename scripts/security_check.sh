#!/bin/bash

# Скрипт для проверки безопасности API
# Проверяет основные уязвимости и проблемы безопасности

set -e

BASE_URL="${BASE_URL:-http://localhost:9999}"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Проверка безопасности API ===${NC}"
echo "Base URL: $BASE_URL"
echo ""

issues=0
warnings=0

# Функция для проверки
check_security() {
    local name=$1
    local check_func=$2
    
    echo -e "${YELLOW}Проверка: $name${NC}"
    if $check_func; then
        echo -e "${GREEN}  ✓ Пройдено${NC}"
    else
        echo -e "${RED}  ✗ Проблема обнаружена${NC}"
        issues=$((issues + 1))
    fi
    echo ""
}

# Проверка 1: HTTPS (если используется)
check_https() {
    if [[ "$BASE_URL" == https://* ]]; then
        return 0
    else
        echo -e "${YELLOW}  Предупреждение: используется HTTP вместо HTTPS${NC}"
        warnings=$((warnings + 1))
        return 0  # Не критично для локального тестирования
    fi
}

# Проверка 2: CORS заголовки
check_cors() {
    response=$(curl -s -I -X OPTIONS "$BASE_URL/api/system/summary" -H "Origin: http://evil.com" 2>&1)
    
    if echo "$response" | grep -qi "Access-Control-Allow-Origin"; then
        cors_header=$(echo "$response" | grep -i "Access-Control-Allow-Origin" | head -n 1)
        if echo "$cors_header" | grep -qi "\*"; then
            echo -e "${RED}  Обнаружен CORS с wildcard (*) - небезопасно${NC}"
            return 1
        fi
        return 0
    else
        echo -e "${YELLOW}  CORS заголовки не найдены (может быть нормально для внутреннего API)${NC}"
        warnings=$((warnings + 1))
        return 0
    fi
}

# Проверка 3: SQL Injection (базовая проверка)
check_sql_injection() {
    # Пробуем базовые SQL injection атаки
    test_payloads=("' OR '1'='1" "'; DROP TABLE--" "1' UNION SELECT NULL--")
    
    for payload in "${test_payloads[@]}"; do
        response=$(curl -s "$BASE_URL/api/databases/list?search=$payload" 2>&1)
        http_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/api/databases/list?search=$payload")
        
        # Если сервер возвращает 500, это может указывать на уязвимость
        if [ "$http_code" -eq 500 ]; then
            echo -e "${RED}  Возможная SQL injection уязвимость (HTTP 500 на payload: $payload)${NC}"
            return 1
        fi
        
        # Проверяем наличие SQL ошибок в ответе
        if echo "$response" | grep -qi "sql.*error\|syntax.*error\|database.*error"; then
            echo -e "${RED}  Обнаружена SQL ошибка в ответе (возможная уязвимость)${NC}"
            return 1
        fi
    done
    
    return 0
}

# Проверка 4: XSS (базовая проверка)
check_xss() {
    test_payload="<script>alert('XSS')</script>"
    response=$(curl -s "$BASE_URL/api/databases/list?search=$test_payload" 2>&1)
    
    # Проверяем, экранирован ли payload в ответе
    if echo "$response" | grep -q "$test_payload"; then
        # Проверяем, экранирован ли он правильно
        if echo "$response" | grep -q "&lt;script&gt;"; then
            return 0  # Правильно экранировано
        else
            echo -e "${RED}  Возможная XSS уязвимость (payload не экранирован)${NC}"
            return 1
        fi
    fi
    
    return 0
}

# Проверка 5: Заголовки безопасности
check_security_headers() {
    response=$(curl -s -I "$BASE_URL/health" 2>&1)
    
    missing_headers=()
    
    if ! echo "$response" | grep -qi "X-Content-Type-Options"; then
        missing_headers+=("X-Content-Type-Options")
    fi
    
    if ! echo "$response" | grep -qi "X-Frame-Options"; then
        missing_headers+=("X-Frame-Options")
    fi
    
    if ! echo "$response" | grep -qi "X-XSS-Protection"; then
        missing_headers+=("X-XSS-Protection")
    fi
    
    if [ ${#missing_headers[@]} -gt 0 ]; then
        echo -e "${YELLOW}  Отсутствуют заголовки безопасности: ${missing_headers[*]}${NC}"
        warnings=$((warnings + 1))
    fi
    
    return 0
}

# Проверка 6: Аутентификация (если требуется)
check_authentication() {
    # Проверяем endpoints, которые должны требовать аутентификацию
    protected_endpoints=("/api/system/summary" "/api/monitoring/metrics")
    
    for endpoint in "${protected_endpoints[@]}"; do
        http_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL$endpoint")
        
        # Если endpoint доступен без аутентификации (не 401/403), это может быть проблемой
        if [ "$http_code" -ne 401 ] && [ "$http_code" -ne 403 ]; then
            echo -e "${YELLOW}  Endpoint $endpoint доступен без аутентификации${NC}"
            warnings=$((warnings + 1))
        fi
    done
    
    return 0
}

# Проверка 7: Rate limiting
check_rate_limiting() {
    # Отправляем много запросов подряд
    success_count=0
    for i in {1..100}; do
        http_code=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/health")
        if [ "$http_code" -eq 200 ]; then
            success_count=$((success_count + 1))
        fi
    done
    
    if [ $success_count -eq 100 ]; then
        echo -e "${YELLOW}  Rate limiting не обнаружен (все 100 запросов прошли успешно)${NC}"
        warnings=$((warnings + 1))
    fi
    
    return 0
}

# Проверка 8: Информация о сервере в заголовках
check_server_info() {
    response=$(curl -s -I "$BASE_URL/health" 2>&1)
    
    if echo "$response" | grep -qi "Server:"; then
        server_header=$(echo "$response" | grep -i "Server:" | head -n 1)
        echo -e "${YELLOW}  Обнаружен заголовок Server: $server_header${NC}"
        echo -e "${YELLOW}  Рекомендуется скрыть информацию о сервере${NC}"
        warnings=$((warnings + 1))
    fi
    
    return 0
}

# Запускаем проверки
check_security "HTTPS" check_https
check_security "CORS" check_cors
check_security "SQL Injection" check_sql_injection
check_security "XSS" check_xss
check_security "Security Headers" check_security_headers
check_security "Authentication" check_authentication
check_security "Rate Limiting" check_rate_limiting
check_security "Server Information" check_server_info

# Итоговый отчет
echo -e "${GREEN}=== Итоговый отчет ===${NC}"
echo "Критических проблем: $issues"
echo "Предупреждений: $warnings"
echo ""

if [ $issues -eq 0 ] && [ $warnings -eq 0 ]; then
    echo -e "${GREEN}Все проверки пройдены успешно!${NC}"
    exit 0
elif [ $issues -eq 0 ]; then
    echo -e "${YELLOW}Есть предупреждения, но критических проблем не обнаружено${NC}"
    exit 0
else
    echo -e "${RED}Обнаружены критические проблемы безопасности!${NC}"
    exit 1
fi

