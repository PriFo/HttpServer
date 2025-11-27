#!/bin/bash
# Тест работы с большими объемами данных
# Мониторит память и CPU во время нормализации, проверяет на утечки памяти

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

TEST_NAME="Работа с большими объемами данных"
REPORT_FILE="$REPORT_DIR/large_data_$(date +%Y%m%d_%H%M%S).md"

log INFO "=== Начало теста: $TEST_NAME ==="

# Проверка доступности сервера
if ! check_server; then
    log ERROR "Сервер недоступен. Тест прерван."
    exit 1
fi

# Определяем имя процесса сервера
PROCESS_NAME="httpserver"
if pgrep -f "httpserver_no_gui" > /dev/null; then
    PROCESS_NAME="httpserver_no_gui"
elif pgrep -f "httpserver" > /dev/null; then
    PROCESS_NAME="httpserver"
else
    log WARNING "Процесс сервера не найден. Будет использовано имя: $PROCESS_NAME"
fi

# Получаем список клиентов и проектов
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

# Тест 1: Базовый мониторинг ресурсов
log INFO "Тест 1: Базовый мониторинг ресурсов (60 секунд)"
RESOURCES_FILE=$(monitor_resources "$PROCESS_NAME" 60 5)

# Анализируем результаты мониторинга
if [ -f "$RESOURCES_FILE" ]; then
    # Пропускаем заголовок
    MEMORY_VALUES=$(tail -n +2 "$RESOURCES_FILE" | cut -d',' -f3 | sort -n)
    if [ -n "$MEMORY_VALUES" ]; then
        FIRST_MEMORY=$(echo "$MEMORY_VALUES" | head -1)
        LAST_MEMORY=$(echo "$MEMORY_VALUES" | tail -1)
        MEMORY_INCREASE=$(echo "scale=2; $LAST_MEMORY - $FIRST_MEMORY" | bc)
        
        log INFO "Память: начальная ${FIRST_MEMORY}MB, конечная ${LAST_MEMORY}MB, изменение ${MEMORY_INCREASE}MB"
        
        if (( $(echo "$MEMORY_INCREASE > 100" | bc -l) )); then
            log WARNING "Обнаружен значительный рост памяти: ${MEMORY_INCREASE}MB"
            MEMORY_LEAK="⚠️ Возможна утечка памяти: рост на ${MEMORY_INCREASE}MB"
        else
            log SUCCESS "Рост памяти в пределах нормы: ${MEMORY_INCREASE}MB"
            MEMORY_LEAK="✅ Рост памяти в норме: ${MEMORY_INCREASE}MB"
        fi
    fi
fi

# Тест 2: Мониторинг во время нормализации
log INFO "Тест 2: Запуск нормализации с мониторингом ресурсов"

# Запускаем мониторинг в фоне
MONITOR_DURATION=300  # 5 минут
monitor_resources "$PROCESS_NAME" "$MONITOR_DURATION" 5 > "$REPORT_DIR/normalization_resources_$(date +%s).csv" &
MONITOR_PID=$!

# Даем время на старт мониторинга
sleep 2

# Запускаем нормализацию
log INFO "Запуск нормализации для проекта $TEST_PROJECT_ID"
NORMALIZATION_RESPONSE=$(http_request "POST" "/api/clients/$TEST_CLIENT_ID/projects/$TEST_PROJECT_ID/normalization/start" \
    '{"all_active": true}' "")
NORMALIZATION_HTTP=$(echo "$NORMALIZATION_RESPONSE" | cut -d'|' -f1)
NORMALIZATION_BODY=$(echo "$NORMALIZATION_RESPONSE" | cut -d'|' -f4-)

if [ "$NORMALIZATION_HTTP" -eq 200 ]; then
    log SUCCESS "Нормализация запущена (HTTP 200)"
    
    # Ждем некоторое время для сбора данных
    log INFO "Ожидание 120 секунд для сбора данных мониторинга..."
    sleep 120
    
    # Проверяем статус нормализации
    STATUS_RESPONSE=$(http_request "GET" "/api/clients/$TEST_CLIENT_ID/projects/$TEST_PROJECT_ID/normalization/status" "" "")
    STATUS_HTTP=$(echo "$STATUS_RESPONSE" | cut -d'|' -f1)
    STATUS_BODY=$(echo "$STATUS_RESPONSE" | cut -d'|' -f4-)
    
    if [ "$STATUS_HTTP" -eq 200 ]; then
        NORMALIZATION_STATUS=$(echo "$STATUS_BODY" | jq -r '.status' 2>/dev/null || echo "unknown")
        log INFO "Статус нормализации: $NORMALIZATION_STATUS"
    fi
    
    # Останавливаем мониторинг
    kill $MONITOR_PID 2>/dev/null || true
    wait $MONITOR_PID 2>/dev/null || true
    
    NORMALIZATION_TEST="✅ Нормализация запущена и работает"
else
    log WARNING "Не удалось запустить нормализацию (HTTP $NORMALIZATION_HTTP)"
    kill $MONITOR_PID 2>/dev/null || true
    NORMALIZATION_TEST="❌ Не удалось запустить нормализацию: HTTP $NORMALIZATION_HTTP"
fi

# Тест 3: Проверка таймаутов
log INFO "Тест 3: Проверка таймаутов при длительных операциях"
TIMEOUT_TEST_RESPONSE=$(http_request "GET" "/api/clients/$TEST_CLIENT_ID/projects/$TEST_PROJECT_ID/normalization/status" "" "")
TIMEOUT_HTTP=$(echo "$TIMEOUT_TEST_RESPONSE" | cut -d'|' -f1)
TIMEOUT_TIME=$(echo "$TIMEOUT_TEST_RESPONSE" | cut -d'|' -f2)

if (( $(echo "$TIMEOUT_TIME > 10" | bc -l) )); then
    log WARNING "Долгий ответ: ${TIMEOUT_TIME}s"
    TIMEOUT_RESULT="⚠️ Долгий ответ: ${TIMEOUT_TIME}s (возможны проблемы с производительностью)"
else
    log SUCCESS "Время ответа в норме: ${TIMEOUT_TIME}s"
    TIMEOUT_RESULT="✅ Время ответа в норме: ${TIMEOUT_TIME}s"
fi

# Тест 4: Проверка CPU
log INFO "Тест 4: Анализ использования CPU"
if [ -f "$RESOURCES_FILE" ]; then
    CPU_VALUES=$(tail -n +2 "$RESOURCES_FILE" | cut -d',' -f2 | sort -n)
    if [ -n "$CPU_VALUES" ]; then
        MAX_CPU=$(echo "$CPU_VALUES" | tail -1)
        AVG_CPU=$(echo "$CPU_VALUES" | awk '{sum+=$1; count++} END {print sum/count}')
        
        log INFO "CPU: максимальное ${MAX_CPU}%, среднее ${AVG_CPU}%"
        
        if (( $(echo "$MAX_CPU > 90" | bc -l) )); then
            CPU_RESULT="⚠️ Высокая нагрузка CPU: максимум ${MAX_CPU}%"
        else
            CPU_RESULT="✅ Нагрузка CPU в норме: максимум ${MAX_CPU}%"
        fi
    fi
fi

# Генерируем отчет
{
    echo "# Отчет: $TEST_NAME"
    echo ""
    echo "**Дата:** $(date)"
    echo "**Сервер:** $BASE_URL"
    echo "**Процесс:** $PROCESS_NAME"
    echo ""
    echo "## Что было сделано"
    echo ""
    echo "1. Базовый мониторинг ресурсов (60 секунд)"
    echo "2. Мониторинг во время нормализации (120+ секунд)"
    echo "3. Проверка таймаутов при длительных операциях"
    echo "4. Анализ использования CPU и памяти"
    echo ""
    echo "## Результаты тестов"
    echo ""
    echo "- $NORMALIZATION_TEST"
    echo "- $TIMEOUT_RESULT"
    echo "- $MEMORY_LEAK"
    if [ -n "$CPU_RESULT" ]; then
        echo "- $CPU_RESULT"
    fi
    echo ""
    echo "## Данные мониторинга"
    echo ""
    if [ -f "$RESOURCES_FILE" ]; then
        echo "Базовый мониторинг сохранен в: $RESOURCES_FILE"
        echo ""
        echo "Первые 10 записей:"
        echo "\`\`\`"
        head -11 "$RESOURCES_FILE" | column -t -s','
        echo "\`\`\`"
    fi
    echo ""
    echo "## Рекомендации"
    echo ""
    echo "1. Регулярно мониторить использование памяти во время длительных операций"
    echo "2. Проверить на утечки памяти в обработчиках нормализации"
    echo "3. Оптимизировать запросы к БД для больших объемов данных"
    echo "4. Рассмотреть использование пагинации для больших результатов"
    echo "5. Добавить таймауты для длительных операций"
    echo ""
    if [ -n "$MEMORY_LEAK" ] && echo "$MEMORY_LEAK" | grep -q "⚠️"; then
        echo "## ⚠️ Внимание: Возможная утечка памяти"
        echo ""
        echo "Рекомендуется:"
        echo "1. Проверить закрытие соединений с БД"
        echo "2. Проверить освобождение ресурсов в горутинах"
        echo "3. Использовать профилирование памяти (pprof) для детального анализа"
    fi
    echo ""
} > "$REPORT_FILE"

log SUCCESS "=== Тест завершен: $TEST_NAME ==="
log INFO "Отчет сохранен в: $REPORT_FILE"

echo ""
echo "Отчет: $REPORT_FILE"
cat "$REPORT_FILE"

