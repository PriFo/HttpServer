#!/bin/bash
# Тест устойчивости к сбоям AI сервиса
# Проверяет обработку ошибок при невалидном API ключе и сбоях AI сервиса

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

TEST_NAME="Устойчивость к сбоям AI сервиса"
REPORT_FILE="$REPORT_DIR/ai_failure_$(date +%Y%m%d_%H%M%S).md"

log INFO "=== Начало теста: $TEST_NAME ==="

# Проверка доступности сервера
if ! check_server; then
    log ERROR "Сервер недоступен. Тест прерван."
    exit 1
fi

# Сохраняем исходную конфигурацию
log INFO "Сохранение исходной конфигурации"
ORIGINAL_CONFIG=$(get_config)
if [ $? -ne 0 ]; then
    log ERROR "Не удалось получить исходную конфигурацию"
    exit 1
fi

ORIGINAL_API_KEY=$(echo "$ORIGINAL_CONFIG" | jq -r '.arliai_api_key' 2>/dev/null || echo "")

# Создаем конфигурацию с невалидным API ключом
log INFO "Установка невалидного API ключа"
INVALID_CONFIG=$(echo "$ORIGINAL_CONFIG" | jq -c '.arliai_api_key = "invalid_key_12345"')
save_config "$INVALID_CONFIG"
sleep 2

TESTS_PASSED=0
TESTS_FAILED=0
TEST_RESULTS=()

# Тест 1: Запуск нормализации с невалидным API ключом
log INFO "Тест 1: Запуск версионированной нормализации с невалидным API ключом"
TEST1_RESPONSE=$(http_request "POST" "/api/normalization/start" \
    '{"item_id": 1, "original_name": "Тестовый товар для проверки AI"}' "")
TEST1_HTTP=$(echo "$TEST1_RESPONSE" | cut -d'|' -f1)
TEST1_BODY=$(echo "$TEST1_RESPONSE" | cut -d'|' -f4-)

if [ "$TEST1_HTTP" -eq 200 ]; then
    SESSION_ID=$(echo "$TEST1_BODY" | jq -r '.session_id' 2>/dev/null)
    log INFO "Сессия создана: $SESSION_ID"
    
    # Тест 2: Применение паттернов (должно пройти без AI)
    log INFO "Тест 2: Применение паттернов к сессии $SESSION_ID"
    sleep 1
    TEST2_RESPONSE=$(http_request "POST" "/api/normalization/apply-patterns" \
        "{\"session_id\": $SESSION_ID}" "")
    TEST2_HTTP=$(echo "$TEST2_RESPONSE" | cut -d'|' -f1)
    TEST2_BODY=$(echo "$TEST2_RESPONSE" | cut -d'|' -f4-)
    
    if [ "$TEST2_HTTP" -eq 200 ]; then
        log SUCCESS "Тест 2 пройден: паттерны применены (HTTP 200)"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✅ Тест 2: Применение паттернов - HTTP 200")
    else
        log WARNING "Тест 2: неожиданный статус HTTP $TEST2_HTTP"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("❌ Тест 2: Применение паттернов - HTTP $TEST2_HTTP")
    fi
    
    # Тест 3: Применение AI коррекции (должно вернуть ошибку)
    log INFO "Тест 3: Применение AI коррекции с невалидным API ключом"
    sleep 1
    TEST3_RESPONSE=$(http_request "POST" "/api/normalization/apply-ai" \
        "{\"session_id\": $SESSION_ID, \"use_chat\": false}" "")
    TEST3_HTTP=$(echo "$TEST3_RESPONSE" | cut -d'|' -f1)
    TEST3_BODY=$(echo "$TEST3_RESPONSE" | cut -d'|' -f4-)
    
    if [ "$TEST3_HTTP" -ge 400 ] && [ "$TEST3_HTTP" -lt 500 ]; then
        log SUCCESS "Тест 3 пройден: система вернула ошибку для невалидного API ключа (HTTP $TEST3_HTTP)"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✅ Тест 3: AI коррекция с невалидным ключом - HTTP $TEST3_HTTP")
        
        # Проверяем, что в ответе есть информация об ошибке
        if echo "$TEST3_BODY" | grep -i "error\|unauthorized\|invalid\|key" > /dev/null; then
            log SUCCESS "В ответе присутствует информация об ошибке"
        fi
    else
        log WARNING "Тест 3: неожиданный статус HTTP $TEST3_HTTP"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("❌ Тест 3: AI коррекция - HTTP $TEST3_HTTP (ожидался 4xx)")
    fi
    
    # Тест 4: Проверка статуса сессии после ошибки AI
    log INFO "Тест 4: Проверка статуса сессии после ошибки AI"
    sleep 1
    TEST4_RESPONSE=$(http_request "GET" "/api/normalization/session/$SESSION_ID" "" "")
    TEST4_HTTP=$(echo "$TEST4_RESPONSE" | cut -d'|' -f1)
    TEST4_BODY=$(echo "$TEST4_RESPONSE" | cut -d'|' -f4-)
    
    if [ "$TEST4_HTTP" -eq 200 ]; then
        SESSION_STATUS=$(echo "$TEST4_BODY" | jq -r '.status' 2>/dev/null)
        if [ "$SESSION_STATUS" = "failed" ] || [ "$SESSION_STATUS" = "error" ]; then
            log SUCCESS "Тест 4 пройден: сессия имеет статус '$SESSION_STATUS'"
            ((TESTS_PASSED++))
            TEST_RESULTS+=("✅ Тест 4: Статус сессии после ошибки - $SESSION_STATUS")
        else
            log WARNING "Тест 4: сессия имеет статус '$SESSION_STATUS' (ожидался 'failed' или 'error')"
            ((TESTS_FAILED++))
            TEST_RESULTS+=("⚠️ Тест 4: Статус сессии - $SESSION_STATUS (ожидался failed/error)")
        fi
    else
        log WARNING "Тест 4: не удалось получить статус сессии (HTTP $TEST4_HTTP)"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("❌ Тест 4: Получение статуса сессии - HTTP $TEST4_HTTP")
    fi
    
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 1: Создание сессии с невалидным ключом - HTTP 200")
else
    log WARNING "Тест 1: не удалось создать сессию (HTTP $TEST1_HTTP)"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("❌ Тест 1: Создание сессии - HTTP $TEST1_HTTP")
fi

# Тест 5: Проверка обработки различных HTTP статусов от AI API
log INFO "Тест 5: Проверка обработки ошибок в логах"
sleep 2

# Проверяем логи на наличие ошибок AI
ERROR_PATTERNS=("unauthorized" "401" "403" "429" "rate limit" "quota" "invalid.*key")
FOUND_ERRORS=()

for pattern in "${ERROR_PATTERNS[@]}"; do
    if check_logs_for_error "$pattern"; then
        FOUND_ERRORS+=("$pattern")
    fi
done

if [ ${#FOUND_ERRORS[@]} -gt 0 ]; then
    log INFO "Найдены ошибки в логах: ${FOUND_ERRORS[*]}"
    ((TESTS_PASSED++))
    TEST_RESULTS+=("✅ Тест 5: Ошибки AI зафиксированы в логах")
else
    log WARNING "Тест 5: ошибки AI не найдены в логах"
    ((TESTS_FAILED++))
    TEST_RESULTS+=("⚠️ Тест 5: Ошибки AI не найдены в логах")
fi

# Восстанавливаем исходную конфигурацию
log INFO "Восстановление исходной конфигурации"
if [ -n "$ORIGINAL_API_KEY" ]; then
    RESTORED_CONFIG=$(echo "$ORIGINAL_CONFIG" | jq -c --arg key "$ORIGINAL_API_KEY" '.arliai_api_key = $key')
else
    RESTORED_CONFIG=$(echo "$ORIGINAL_CONFIG" | jq -c 'del(.arliai_api_key)')
fi
save_config "$RESTORED_CONFIG"
sleep 2

# Тест 6: Проверка восстановления после восстановления API ключа
log INFO "Тест 6: Проверка работы после восстановления валидного API ключа"
if [ -n "$ORIGINAL_API_KEY" ] && [ "$ORIGINAL_API_KEY" != "null" ] && [ "$ORIGINAL_API_KEY" != "" ]; then
    TEST6_RESPONSE=$(http_request "POST" "/api/normalization/start" \
        '{"item_id": 2, "original_name": "Тестовый товар после восстановления"}' "")
    TEST6_HTTP=$(echo "$TEST6_RESPONSE" | cut -d'|' -f1)
    
    if [ "$TEST6_HTTP" -eq 200 ]; then
        log SUCCESS "Тест 6 пройден: система работает после восстановления API ключа"
        ((TESTS_PASSED++))
        TEST_RESULTS+=("✅ Тест 6: Восстановление после валидного ключа - HTTP 200")
    else
        log WARNING "Тест 6: неожиданный статус HTTP $TEST6_HTTP"
        ((TESTS_FAILED++))
        TEST_RESULTS+=("❌ Тест 6: Восстановление - HTTP $TEST6_HTTP")
    fi
else
    log INFO "Тест 6 пропущен: исходный API ключ не был установлен"
    TEST_RESULTS+=("⏭️ Тест 6: Пропущен (нет исходного API ключа)")
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
    echo "1. Сохранена исходная конфигурация"
    echo "2. Установлен невалидный ARLIAI_API_KEY"
    echo "3. Запущен полный цикл нормализации (start -> apply-patterns -> apply-ai)"
    echo "4. Проверен статус сессии после ошибки AI"
    echo "5. Проверены логи на наличие ошибок (401, 403, 429, 500)"
    echo "6. Восстановлен валидный API ключ"
    echo "7. Проверена работа после восстановления"
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
        echo "✅ Все тесты пройдены. Система корректно обрабатывает сбои AI сервиса."
    else
        echo "⚠️ Обнаружены проблемы с обработкой сбоев AI:"
        echo ""
        echo "Рекомендуется:"
        echo "1. Убедиться, что сессии корректно помечаются как 'failed' при ошибках AI"
        echo "2. Улучшить логирование ошибок AI для диагностики"
        echo "3. Добавить retry механизм с экспоненциальной задержкой"
    fi
    echo ""
    echo "## Рекомендации"
    echo ""
    echo "1. Проверить обработку ошибок в `server/services/normalization_service.go:ApplyAI`"
    echo "2. Убедиться, что сессии получают статус 'failed' при ошибках AI"
    echo "3. Улучшить обработку различных HTTP статусов в `nomenclature/ai_client.go`"
    echo "4. Проверить retry логику в `internal/infrastructure/ai/arliai.go`"
    echo "5. Добавить понятные сообщения об ошибках для пользователей"
    echo ""
    if [ ${#FOUND_ERRORS[@]} -gt 0 ]; then
        echo "## Обнаруженные ошибки в логах"
        echo ""
        for error in "${FOUND_ERRORS[@]}"; do
            echo "- $error"
        done
    fi
    echo ""
} > "$REPORT_FILE"

log SUCCESS "=== Тест завершен: $TEST_NAME ==="
log INFO "Отчет сохранен в: $REPORT_FILE"

echo ""
echo "Отчет: $REPORT_FILE"
cat "$REPORT_FILE"

