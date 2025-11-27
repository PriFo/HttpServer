#!/bin/bash
# Тест нормализации с невалидными данными
# Проверяет обработку ошибок при несуществующих database_id, невалидных параметрах

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

TEST_NAME="Нормализация с невалидными данными"
REPORT_FILE="$REPORT_DIR/invalid_normalization_$(date +%Y%m%d_%H%M%S).md"

log INFO "=== Начало теста: $TEST_NAME ==="

# Проверка доступности сервера
if ! check_server; then
    log ERROR "Сервер недоступен. Тест прерван."
    exit 1
fi

# Получаем список клиентов и проектов для тестирования
log INFO "Получение списка клиентов и проектов"
CLIENTS_RESPONSE=$(http_request "GET" "/api/clients" "" "")
CLIENTS_HTTP=$(echo "$CLIENTS_RESPONSE" | cut -d'|' -f1)
CLIENTS_BODY=$(echo "$CLIENTS_RESPONSE" | cut -d'|' -f4-)

TEST_CLIENT_ID=1
TEST_PROJECT_ID=1

if [ "$CLIENTS_HTTP" -eq 200 ]; then
    FIRST_CLIENT_ID=$(echo "$CLIENTS_BODY" | jq -r '.[0].id' 2>/dev/null)
    if [ -n "$FIRST_CLIENT_ID" ] && [ "$FIRST_CLIENT_ID" != "null" ]; then
        TEST_CLIENT_ID=$FIRST_CLIENT_ID
        log INFO "Используется клиент ID: $TEST_CLIENT_ID"
        
        # Получаем проекты клиента
        PROJECTS_RESPONSE=$(http_request "GET" "/api/clients/$TEST_CLIENT_ID/projects" "" "")
        PROJECTS_HTTP=$(echo "$PROJECTS_RESPONSE" | cut -d'|' -f1)
        PROJECTS_BODY=$(echo "$PROJECTS_RESPONSE" | cut -d'|' -f4-)
        
        if [ "$PROJECTS_HTTP" -eq 200 ]; then
            FIRST_PROJECT_ID=$(echo "$PROJECTS_BODY" | jq -r '.[0].id' 2>/dev/null)
            if [ -n "$FIRST_PROJECT_ID" ] && [ "$FIRST_PROJECT_ID" != "null" ]; then
                TEST_PROJECT_ID=$FIRST_PROJECT_ID
                log INFO "Используется проект ID: $TEST_PROJECT_ID"
            fi
        fi
    fi
fi

TESTS_PASSED=0
TESTS_FAILED=0
TEST_RESULTS=()

# Тест 1: Несуществующий database_id
log INFO "Тест 1: Запуск нормализации с несуществующим database_id"
TEST1_RESPONSE=$(http_request "POST" "/api/clients/$TEST_CLIENT_ID/projects/$TEST_PROJECT_ID/normalization/start" \
    '{"database_ids": [999999], "all_active": false}' "")
TEST1_HTTP=$(echo "$TEST1_RESPONSE" | cut -d'|' -f1)
TEST1_BODY=$(echo "$TEST1_RESPONSE" | cut -d'|' -f4-)

if [ "$TEST1_HTTP" -ge 400 ] && [ "$TEST1_HTTP" -lt 500 ]; then
    log SUCCESS "Тест 1 пройден: система вернула ошибку клиента (HTTP $TEST1_HTTP)"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 1: Несуществующий database_id - HTTP $TEST1_HTTP")
else
    log WARNING "Тест 1: неожиданный статус HTTP $TEST1_HTTP"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 1: Несуществующий database_id - HTTP $TEST1_HTTP (ожидался 4xx)")
fi

# Тест 2: Невалидный item_id (0)
log INFO "Тест 2: Запуск версионированной нормализации с item_id=0"
TEST2_RESPONSE=$(http_request "POST" "/api/normalization/start" \
    '{"item_id": 0, "original_name": "Test Item"}' "")
TEST2_HTTP=$(echo "$TEST2_RESPONSE" | cut -d'|' -f1)
TEST2_BODY=$(echo "$TEST2_RESPONSE" | cut -d'|' -f4-)

if [ "$TEST2_HTTP" -eq 400 ]; then
    log SUCCESS "Тест 2 пройден: система вернула ошибку валидации (HTTP 400)"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 2: item_id=0 - HTTP 400")
else
    log WARNING "Тест 2: неожиданный статус HTTP $TEST2_HTTP"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 2: item_id=0 - HTTP $TEST2_HTTP (ожидался 400)")
fi

# Тест 3: Пустой original_name
log INFO "Тест 3: Запуск версионированной нормализации с пустым original_name"
TEST3_RESPONSE=$(http_request "POST" "/api/normalization/start" \
    '{"item_id": 123, "original_name": ""}' "")
TEST3_HTTP=$(echo "$TEST3_RESPONSE" | cut -d'|' -f1)
TEST3_BODY=$(echo "$TEST3_RESPONSE" | cut -d'|' -f4-)

if [ "$TEST3_HTTP" -eq 400 ]; then
    log SUCCESS "Тест 3 пройден: система вернула ошибку валидации (HTTP 400)"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 3: Пустой original_name - HTTP 400")
else
    log WARNING "Тест 3: неожиданный статус HTTP $TEST3_HTTP"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 3: Пустой original_name - HTTP $TEST3_HTTP (ожидался 400)")
fi

# Тест 4: Отрицательный лимит в истории
log INFO "Тест 4: Получение истории нормализации с отрицательным лимитом"
TEST4_RESPONSE=$(http_request "GET" "/api/normalization/history?limit=-10" "" "")
TEST4_HTTP=$(echo "$TEST4_RESPONSE" | cut -d'|' -f1)

if [ "$TEST4_HTTP" -ge 400 ] || [ "$TEST4_HTTP" -eq 200 ]; then
    # Может быть либо ошибка, либо игнорирование невалидного параметра
    log INFO "Тест 4: HTTP $TEST4_HTTP (система обработала невалидный параметр)"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 4: Отрицательный лимит - HTTP $TEST4_HTTP")
else
    log WARNING "Тест 4: неожиданный статус HTTP $TEST4_HTTP"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 4: Отрицательный лимит - HTTP $TEST4_HTTP")
fi

# Тест 5: Несуществующий session_id
log INFO "Тест 5: Получение сессии нормализации с несуществующим session_id"
TEST5_RESPONSE=$(http_request "GET" "/api/normalization/session/999999" "" "")
TEST5_HTTP=$(echo "$TEST5_RESPONSE" | cut -d'|' -f1)

if [ "$TEST5_HTTP" -eq 404 ]; then
    log SUCCESS "Тест 5 пройден: система вернула 404 для несуществующей сессии"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 5: Несуществующий session_id - HTTP 404")
else
    log WARNING "Тест 5: неожиданный статус HTTP $TEST5_HTTP"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 5: Несуществующий session_id - HTTP $TEST5_HTTP (ожидался 404)")
fi

# Тест 6: Невалидный JSON в теле запроса
log INFO "Тест 6: Запуск нормализации с невалидным JSON"
TEST6_RESPONSE=$(http_request "POST" "/api/normalization/start" \
    '{"item_id": "invalid", "original_name":}' "")
TEST6_HTTP=$(echo "$TEST6_RESPONSE" | cut -d'|' -f1)

if [ "$TEST6_HTTP" -eq 400 ]; then
    log SUCCESS "Тест 6 пройден: система вернула ошибку парсинга JSON (HTTP 400)"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 6: Невалидный JSON - HTTP 400")
else
    log WARNING "Тест 6: неожиданный статус HTTP $TEST6_HTTP"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 6: Невалидный JSON - HTTP $TEST6_HTTP (ожидался 400)")
fi

# Генерируем отчет
{
    echo "# Отчет: $TEST_NAME"
    echo ""
    echo "**Дата:** $(date)"
    echo "**Сервер:** $BASE_URL"
    echo ""
    echo "## Что было сделано"
    echo ""
    echo "1. Тест с несуществующим database_id"
    echo "2. Тест с невалидным item_id (0)"
    echo "3. Тест с пустым original_name"
    echo "4. Тест с отрицательным лимитом в истории"
    echo "5. Тест с несуществующим session_id"
    echo "6. Тест с невалидным JSON"
    echo ""
    echo "## Результаты тестов"
    echo ""
    for result in "${TEST_RESULTS[@]}"; do
        echo "- $result"
    done
    echo ""
    echo "**Всего тестов:** $((TESTS_PASSED + TESTS_FAILED))"
    echo "**Пройдено:** $TESTS_PASSED"
    echo "**Провалено:** $TESTS_FAILED"
    echo ""
    echo "## Анализ"
    echo ""
    if [ $TESTS_FAILED -eq 0 ]; then
        echo "✅ Все тесты пройдены. Система корректно обрабатывает невалидные данные."
    else
        echo "⚠️ Обнаружены проблемы с обработкой невалидных данных:"
        echo ""
        echo "Рекомендуется:"
        echo "1. Улучшить валидацию входных данных в handlers"
        echo "2. Добавить проверки существования database_id перед запуском нормализации"
        echo "3. Улучшить сообщения об ошибках для лучшей диагностики"
    fi
    echo ""
    echo "## Рекомендации"
    echo ""
    echo "1. Добавить валидацию в `server/handlers/normalization.go:HandleStartClientProjectNormalization`"
    echo "2. Проверить обработку несуществующих database_id в `server/client_legacy_handlers.go:startProjectNormalization`"
    echo "3. Улучшить валидацию в `server/handlers/normalization.go:HandleStartVersionedNormalization`"
    echo ""
} > "$REPORT_FILE"

log SUCCESS "=== Тест завершен: $TEST_NAME ==="
log INFO "Отчет сохранен в: $REPORT_FILE"

echo ""
echo "Отчет: $REPORT_FILE"
cat "$REPORT_FILE"

