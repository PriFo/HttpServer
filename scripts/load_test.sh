#!/bin/bash

# Скрипт для нагрузочного тестирования API
# Использует curl для отправки множественных запросов

set -e

BASE_URL="${BASE_URL:-http://localhost:9999}"
CONCURRENT_REQUESTS="${CONCURRENT_REQUESTS:-10}"
TOTAL_REQUESTS="${TOTAL_REQUESTS:-100}"
TEST_DURATION="${TEST_DURATION:-60}"

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}=== Нагрузочное тестирование API ===${NC}"
echo "Base URL: $BASE_URL"
echo "Concurrent requests: $CONCURRENT_REQUESTS"
echo "Total requests: $TOTAL_REQUESTS"
echo "Duration: ${TEST_DURATION}s"
echo ""

# Проверка доступности сервера
echo -e "${YELLOW}Проверка доступности сервера...${NC}"
if ! curl -s -f "$BASE_URL/health" > /dev/null; then
    echo -e "${RED}Ошибка: сервер недоступен на $BASE_URL${NC}"
    exit 1
fi
echo -e "${GREEN}Сервер доступен${NC}"
echo ""

# Функция для тестирования endpoint
test_endpoint() {
    local endpoint=$1
    local method=${2:-GET}
    local data=${3:-""}
    
    echo -e "${YELLOW}Тестирование: $method $endpoint${NC}"
    
    local success=0
    local failed=0
    local total_time=0
    local min_time=999999
    local max_time=0
    
    local start_time=$(date +%s)
    
    for i in $(seq 1 $TOTAL_REQUESTS); do
        local request_start=$(date +%s%N)
        
        if [ "$method" = "GET" ]; then
            response=$(curl -s -w "\n%{http_code}\n%{time_total}" -o /tmp/response_$$.json "$BASE_URL$endpoint" 2>&1)
        elif [ "$method" = "POST" ]; then
            response=$(curl -s -w "\n%{http_code}\n%{time_total}" -o /tmp/response_$$.json -X POST -H "Content-Type: application/json" -d "$data" "$BASE_URL$endpoint" 2>&1)
        fi
        
        local request_end=$(date +%s%N)
        local request_time=$(echo "scale=3; ($request_end - $request_start) / 1000000" | bc)
        
        http_code=$(echo "$response" | tail -n 2 | head -n 1)
        time_total=$(echo "$response" | tail -n 1)
        
        if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
            success=$((success + 1))
        else
            failed=$((failed + 1))
            echo -e "${RED}  Запрос $i: HTTP $http_code${NC}"
        fi
        
        total_time=$(echo "$total_time + $time_total" | bc)
        
        if (( $(echo "$time_total < $min_time" | bc -l) )); then
            min_time=$time_total
        fi
        
        if (( $(echo "$time_total > $max_time" | bc -l) )); then
            max_time=$time_total
        fi
        
        # Прогресс
        if [ $((i % 10)) -eq 0 ]; then
            echo -n "."
        fi
    done
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    local avg_time=$(echo "scale=3; $total_time / $TOTAL_REQUESTS" | bc)
    local success_rate=$(echo "scale=2; $success * 100 / $TOTAL_REQUESTS" | bc)
    local rps=$(echo "scale=2; $TOTAL_REQUESTS / $duration" | bc)
    
    echo ""
    echo -e "${GREEN}Результаты для $endpoint:${NC}"
    echo "  Успешных: $success / $TOTAL_REQUESTS ($success_rate%)"
    echo "  Неудачных: $failed"
    echo "  Среднее время ответа: ${avg_time}s"
    echo "  Минимальное время: ${min_time}s"
    echo "  Максимальное время: ${max_time}s"
    echo "  Запросов в секунду: $rps"
    echo "  Общее время: ${duration}s"
    echo ""
    
    # Сохраняем результаты
    echo "$endpoint,$method,$success,$failed,$avg_time,$min_time,$max_time,$rps" >> /tmp/load_test_results.csv
}

# Создаем файл результатов
echo "endpoint,method,success,failed,avg_time,min_time,max_time,rps" > /tmp/load_test_results.csv

# Тестируем основные endpoints
echo -e "${GREEN}=== Тест 1: Health Check ===${NC}"
test_endpoint "/health" "GET"

echo -e "${GREEN}=== Тест 2: System Summary ===${NC}"
test_endpoint "/api/system/summary" "GET"

echo -e "${GREEN}=== Тест 3: Monitoring Metrics ===${NC}"
test_endpoint "/api/monitoring/metrics" "GET"

echo -e "${GREEN}=== Тест 4: Performance Metrics ===${NC}"
test_endpoint "/api/monitoring/performance" "GET"

echo -e "${GREEN}=== Тест 5: Databases List ===${NC}"
test_endpoint "/api/databases/list" "GET"

# Генерируем отчет
echo -e "${GREEN}=== Генерация отчета ===${NC}"
report_file="load_test_report_$(date +%Y%m%d_%H%M%S).txt"
{
    echo "Отчет о нагрузочном тестировании"
    echo "Дата: $(date)"
    echo "Base URL: $BASE_URL"
    echo "Конкурентных запросов: $CONCURRENT_REQUESTS"
    echo "Всего запросов на endpoint: $TOTAL_REQUESTS"
    echo ""
    echo "Результаты:"
    echo "---"
    cat /tmp/load_test_results.csv | column -t -s','
} > "$report_file"

echo -e "${GREEN}Отчет сохранен в: $report_file${NC}"
echo ""

# Очистка
rm -f /tmp/response_$$.json /tmp/load_test_results.csv

echo -e "${GREEN}Нагрузочное тестирование завершено${NC}"

