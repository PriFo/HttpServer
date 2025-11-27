#!/bin/bash

# Скрипт для запуска полного набора тестов нормализации контрагентов

set -e

echo "=========================================="
echo "Запуск тестов нормализации контрагентов"
echo "=========================================="
echo ""

# Цвета для вывода
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Функция для вывода результата
print_result() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓${NC} $2"
    else
        echo -e "${RED}✗${NC} $2"
    fi
}

# 1. Unit-тесты для CounterpartyNormalizer
echo "1. Запуск unit-тестов для CounterpartyNormalizer..."
go test ./normalization -run TestProcessNormalization_StopCheck -v
UNIT_TEST_RESULT=$?
print_result $UNIT_TEST_RESULT "Unit-тесты остановки"

echo ""

# 2. Все тесты нормализации контрагентов
echo "2. Запуск всех тестов нормализации контрагентов..."
go test ./normalization -run TestCounterparty -v
COUNTERPARTY_TEST_RESULT=$?
print_result $COUNTERPARTY_TEST_RESULT "Тесты нормализации контрагентов"

echo ""

# 3. Интеграционные тесты API
echo "3. Запуск интеграционных тестов API..."
go test ./server -run TestStop -v
API_TEST_RESULT=$?
print_result $API_TEST_RESULT "Интеграционные тесты API"

echo ""

# 4. E2E тесты
echo "4. Запуск E2E тестов..."
go test ./server -run TestCounterpartyNormalizationE2E -v
E2E_TEST_RESULT=$?
print_result $E2E_TEST_RESULT "E2E тесты"

echo ""

# 5. Benchmark тесты производительности
echo "5. Запуск benchmark тестов производительности..."
go test ./normalization -bench=BenchmarkStopCheck -benchmem -benchtime=5s
BENCHMARK_RESULT=$?
print_result $BENCHMARK_RESULT "Benchmark тесты"

echo ""

# Итоговый результат
echo "=========================================="
echo "Итоговые результаты:"
echo "=========================================="

TOTAL_FAILED=0

if [ $UNIT_TEST_RESULT -ne 0 ]; then
    echo -e "${RED}✗${NC} Unit-тесты: ПРОВАЛЕНЫ"
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
else
    echo -e "${GREEN}✓${NC} Unit-тесты: ПРОЙДЕНЫ"
fi

if [ $COUNTERPARTY_TEST_RESULT -ne 0 ]; then
    echo -e "${RED}✗${NC} Тесты нормализации: ПРОВАЛЕНЫ"
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
else
    echo -e "${GREEN}✓${NC} Тесты нормализации: ПРОЙДЕНЫ"
fi

if [ $API_TEST_RESULT -ne 0 ]; then
    echo -e "${RED}✗${NC} Интеграционные тесты: ПРОВАЛЕНЫ"
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
else
    echo -e "${GREEN}✓${NC} Интеграционные тесты: ПРОЙДЕНЫ"
fi

if [ $E2E_TEST_RESULT -ne 0 ]; then
    echo -e "${RED}✗${NC} E2E тесты: ПРОВАЛЕНЫ"
    TOTAL_FAILED=$((TOTAL_FAILED + 1))
else
    echo -e "${GREEN}✓${NC} E2E тесты: ПРОЙДЕНЫ"
fi

if [ $BENCHMARK_RESULT -ne 0 ]; then
    echo -e "${YELLOW}⚠${NC} Benchmark тесты: ОШИБКА (не критично)"
else
    echo -e "${GREEN}✓${NC} Benchmark тесты: ЗАВЕРШЕНЫ"
fi

echo ""

if [ $TOTAL_FAILED -eq 0 ]; then
    echo -e "${GREEN}=========================================="
    echo "Все тесты пройдены успешно!"
    echo "==========================================${NC}"
    exit 0
else
    echo -e "${RED}=========================================="
    echo "Обнаружены ошибки в $TOTAL_FAILED группах тестов"
    echo "==========================================${NC}"
    exit 1
fi

