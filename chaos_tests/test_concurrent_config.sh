#!/bin/bash
# Тест конкурентных обновлений конфигурации
# Проверяет race conditions и блокировки БД при параллельных PUT /api/config запросах

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/utils.sh"

TEST_NAME="Конкурентные обновления конфигурации"
REPORT_FILE="$REPORT_DIR/concurrent_config_$(date +%Y%m%d_%H%M%S).md"

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

# Функция для генерации тестовой конфигурации
generate_test_config() {
    local index=$1
    local config_json=$(echo "$ORIGINAL_CONFIG" | jq -c --arg port "999$index" '.port = $port')
    echo "$config_json"
}

# Функция для выполнения одного запроса обновления конфигурации
update_config_request() {
    local index=$1
    local config_json=$(generate_test_config $index)
    local response=$(http_request "PUT" "/api/config" "$config_json" "")
    echo "$response"
}

# Запускаем параллельные запросы
log INFO "Запуск 10 параллельных обновлений конфигурации"
RESULTS_DIR=$(run_parallel_requests 10 update_config_request)

# Анализируем результаты
log INFO "Анализ результатов конкурентных обновлений"
ANALYSIS=$(analyze_parallel_results "$RESULTS_DIR")
SUCCESS=$(echo "$ANALYSIS" | cut -d'|' -f1)
FAILED=$(echo "$ANALYSIS" | cut -d'|' -f2)
DB_LOCKED=$(echo "$ANALYSIS" | cut -d'|' -f3)

# Проверяем историю конфигурации на пропуски версий
log INFO "Проверка истории конфигурации"
HISTORY_RESPONSE=$(http_request "GET" "/api/config/history?limit=20" "" "")
HISTORY_HTTP=$(echo "$HISTORY_RESPONSE" | cut -d'|' -f1)
HISTORY_BODY=$(echo "$HISTORY_RESPONSE" | cut -d'|' -f4-)

if [ "$HISTORY_HTTP" -eq 200 ]; then
    # Проверяем на пропуски версий
    VERSIONS=$(echo "$HISTORY_BODY" | jq -r '.history[]?.version' 2>/dev/null | grep -v '^null$' | sort -n)
    if [ -n "$VERSIONS" ] && [ "$VERSIONS" != "" ]; then
        local prev_version=0
        local gaps=0
        for version in $VERSIONS; do
            if [ -n "$version" ] && [ "$version" != "null" ] && [ "$version" != "" ]; then
                if [ "$prev_version" -gt 0 ] && [ $((version - prev_version)) -gt 1 ]; then
                    log WARNING "Обнаружен пропуск версий: $prev_version -> $version"
                    ((gaps++))
                fi
                prev_version=$version
            fi
        done
        if [ $gaps -eq 0 ]; then
            log SUCCESS "Пропусков версий не обнаружено"
        fi
    else
        log INFO "История конфигурации пуста или не удалось распарсить"
    fi
else
    log WARNING "Не удалось получить историю конфигурации: HTTP $HISTORY_HTTP"
fi

# Проверяем финальную конфигурацию
log INFO "Проверка финальной конфигурации"
FINAL_CONFIG=$(get_config)
if [ $? -eq 0 ]; then
    log SUCCESS "Финальная конфигурация доступна"
else
    log ERROR "Не удалось получить финальную конфигурацию"
fi

# Восстанавливаем исходную конфигурацию
log INFO "Восстановление исходной конфигурации"
save_config "$ORIGINAL_CONFIG"

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
    echo "2. Запущено 10 параллельных PUT /api/config запросов с разными данными"
    echo "3. Проанализированы результаты на наличие ошибок блокировки БД"
    echo "4. Проверена история конфигурации на пропуски версий"
    echo "5. Восстановлена исходная конфигурация"
    echo ""
    echo "## Результаты"
    echo ""
    echo "- **Успешных запросов:** $SUCCESS"
    echo "- **Неудачных запросов:** $FAILED"
    echo "- **Ошибок блокировки БД:** $DB_LOCKED"
    echo ""
    echo "## Ошибки"
    echo ""
    if [ "$DB_LOCKED" -gt 0 ]; then
        echo "⚠️ **Обнаружены ошибки блокировки БД:** $DB_LOCKED случаев"
        echo ""
        echo "Это указывает на возможные race conditions в `SaveAppConfigWithHistory`."
        echo "Рекомендуется добавить транзакции или механизм блокировок."
    else
        echo "✅ Ошибок блокировки БД не обнаружено"
    fi
    echo ""
    if [ "$FAILED" -gt 0 ]; then
        echo "⚠️ **Неудачных запросов:** $FAILED"
        echo ""
        echo "Детали ошибок можно найти в: $RESULTS_DIR"
    fi
    echo ""
    echo "## Рекомендации"
    echo ""
    if [ "$DB_LOCKED" -gt 0 ] || [ "$FAILED" -gt 0 ]; then
        echo "1. Добавить транзакции в `database/service_db.go:SaveAppConfigWithHistory`"
        echo "2. Реализовать retry механизм с экспоненциальной задержкой"
        echo "3. Добавить оптимистичную блокировку через версионирование"
        echo "4. Рассмотреть использование мьютексов на уровне приложения"
    else
        echo "✅ Система успешно обработала конкурентные обновления"
    fi
    echo ""
    echo "## Логи"
    echo ""
    echo "Детальные логи доступны в: $LOG_DIR"
    echo "Результаты запросов: $RESULTS_DIR"
} > "$REPORT_FILE"

log SUCCESS "=== Тест завершен: $TEST_NAME ==="
log INFO "Отчет сохранен в: $REPORT_FILE"

echo ""
echo "Отчет: $REPORT_FILE"
cat "$REPORT_FILE"

