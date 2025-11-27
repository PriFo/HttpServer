#!/bin/bash

# Тестирование API конфигурации
# Сервер должен быть запущен на http://localhost:9999

BASE_URL="http://localhost:9999"
REPORT_FILE="config_api_test_report.md"

echo "# Отчет о тестировании API конфигурации" > "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "Дата: $(date)" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"

# Функция для выполнения запроса и сохранения результата
test_request() {
    local test_name="$1"
    local method="$2"
    local url="$3"
    local data="$4"
    local expected_status="$5"
    
    echo "## Тест: $test_name" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    echo "**Запрос:**" >> "$REPORT_FILE"
    echo "- Метод: $method" >> "$REPORT_FILE"
    echo "- URL: $url" >> "$REPORT_FILE"
    if [ -n "$data" ]; then
        echo "- Body: \`$data\`" >> "$REPORT_FILE"
    fi
    echo "" >> "$REPORT_FILE"
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" "$url")
    elif [ "$method" = "PUT" ]; then
        response=$(curl -s -w "\n%{http_code}" -X PUT -H "Content-Type: application/json" -d "$data" "$url")
    fi
    
    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    echo "**Ответ:**" >> "$REPORT_FILE"
    echo "- Статус: $http_code" >> "$REPORT_FILE"
    echo "- Body:" >> "$REPORT_FILE"
    echo '```json' >> "$REPORT_FILE"
    echo "$body" | jq . 2>/dev/null || echo "$body" >> "$REPORT_FILE"
    echo '```' >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    if [ "$http_code" = "$expected_status" ]; then
        echo "**Результат:** ✅ PASS" >> "$REPORT_FILE"
    else
        echo "**Результат:** ❌ FAIL (ожидался статус $expected_status, получен $http_code)" >> "$REPORT_FILE"
    fi
    echo "" >> "$REPORT_FILE"
    echo "---" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
}

# Тест 1: GET /api/config (безопасная версия)
echo "Выполняю тест 1: GET /api/config..."
test_request "GET /api/config (безопасная версия)" \
    "GET" \
    "$BASE_URL/api/config" \
    "" \
    "200"

# Проверка отсутствия секретных полей
response=$(curl -s "$BASE_URL/api/config")
if echo "$response" | jq -e '.arliai_api_key' > /dev/null 2>&1; then
    echo "⚠️  ВНИМАНИЕ: В безопасной версии обнаружено поле arliai_api_key" >> "$REPORT_FILE"
else
    echo "✅ Поле arliai_api_key отсутствует в безопасной версии" >> "$REPORT_FILE"
fi

if echo "$response" | jq -e '.has_arliai_api_key' > /dev/null 2>&1; then
    echo "✅ Поле has_arliai_api_key присутствует" >> "$REPORT_FILE"
else
    echo "⚠️  Поле has_arliai_api_key отсутствует" >> "$REPORT_FILE"
fi

if echo "$response" | jq -e '.log_level' > /dev/null 2>&1; then
    log_level=$(echo "$response" | jq -r '.log_level')
    echo "✅ Поле log_level присутствует: $log_level" >> "$REPORT_FILE"
else
    echo "❌ Поле log_level отсутствует" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"

# Тест 2: GET /api/config/full (полная версия)
echo "Выполняю тест 2: GET /api/config/full..."
test_request "GET /api/config/full (полная версия)" \
    "GET" \
    "$BASE_URL/api/config/full" \
    "" \
    "200"

# Сохраняем текущую конфигурацию для последующего обновления
current_config=$(curl -s "$BASE_URL/api/config/full")
echo "$current_config" > /tmp/current_config.json

# Тест 3: PUT /api/config (обновление log_level)
echo "Выполняю тест 3: PUT /api/config..."
# Создаем обновленную конфигурацию с измененным log_level
updated_config=$(echo "$current_config" | jq '.log_level = "DEBUG"')
test_request "PUT /api/config (обновление log_level)" \
    "PUT" \
    "$BASE_URL/api/config?reason=QA test update" \
    "$updated_config" \
    "200"

# Проверяем, что log_level изменился
new_config=$(curl -s "$BASE_URL/api/config/full")
new_log_level=$(echo "$new_config" | jq -r '.log_level')
if [ "$new_log_level" = "DEBUG" ]; then
    echo "✅ log_level успешно обновлен на DEBUG" >> "$REPORT_FILE"
else
    echo "❌ log_level не обновлен (текущее значение: $new_log_level)" >> "$REPORT_FILE"
fi
echo "" >> "$REPORT_FILE"

# Тест 4: GET /api/config/history
echo "Выполняю тест 4: GET /api/config/history..."
test_request "GET /api/config/history" \
    "GET" \
    "$BASE_URL/api/config/history" \
    "" \
    "200"

# Проверка структуры истории
history_response=$(curl -s "$BASE_URL/api/config/history")
if echo "$history_response" | jq -e '.current_version' > /dev/null 2>&1; then
    current_version=$(echo "$history_response" | jq -r '.current_version')
    echo "✅ current_version: $current_version" >> "$REPORT_FILE"
else
    echo "❌ Поле current_version отсутствует" >> "$REPORT_FILE"
fi

if echo "$history_response" | jq -e '.history' > /dev/null 2>&1; then
    history_count=$(echo "$history_response" | jq '.history | length')
    echo "✅ Количество записей в истории: $history_count" >> "$REPORT_FILE"
    
    # Проверяем последнюю запись
    last_entry=$(echo "$history_response" | jq '.history[0]')
    if echo "$last_entry" | jq -e '.version' > /dev/null 2>&1; then
        version=$(echo "$last_entry" | jq -r '.version')
        echo "✅ Версия последней записи: $version" >> "$REPORT_FILE"
    fi
    
    if echo "$last_entry" | jq -e '.changed_by' > /dev/null 2>&1; then
        changed_by=$(echo "$last_entry" | jq -r '.changed_by')
        echo "✅ changed_by: $changed_by" >> "$REPORT_FILE"
    fi
    
    if echo "$last_entry" | jq -e '.change_reason' > /dev/null 2>&1; then
        change_reason=$(echo "$last_entry" | jq -r '.change_reason')
        if [ "$change_reason" = "QA test update" ]; then
            echo "✅ change_reason корректно сохранен: $change_reason" >> "$REPORT_FILE"
        else
            echo "⚠️  change_reason: $change_reason (ожидалось: QA test update)" >> "$REPORT_FILE"
        fi
    fi
    
    if echo "$last_entry" | jq -e '.created_at' > /dev/null 2>&1; then
        created_at=$(echo "$last_entry" | jq -r '.created_at')
        echo "✅ created_at: $created_at" >> "$REPORT_FILE"
    fi
else
    echo "❌ Поле history отсутствует" >> "$REPORT_FILE"
fi
echo "" >> "$REPORT_FILE"

# Тест 5: Невалидный JSON
echo "Выполняю тест 5: PUT /api/config с невалидным JSON..."
test_request "PUT /api/config (невалидный JSON)" \
    "PUT" \
    "$BASE_URL/api/config" \
    '{"log_level":}' \
    "400"

# Тест 6: Невалидное значение log_level
echo "Выполняю тест 6: PUT /api/config с невалидным log_level..."
invalid_config=$(echo "$current_config" | jq '.log_level = "INVALID"')
test_request "PUT /api/config (невалидный log_level)" \
    "PUT" \
    "$BASE_URL/api/config" \
    "$invalid_config" \
    "400"

# Тест 7: Некорректный параметр limit
echo "Выполняю тест 7: GET /api/config/history?limit=-1..."
test_request "GET /api/config/history?limit=-1" \
    "GET" \
    "$BASE_URL/api/config/history?limit=-1" \
    "" \
    "200"

# Тест 8: Слишком большое значение limit
echo "Выполняю тест 8: GET /api/config/history?limit=1000..."
test_request "GET /api/config/history?limit=1000" \
    "GET" \
    "$BASE_URL/api/config/history?limit=1000" \
    "" \
    "200"

# Проверяем, что limit ограничен
large_limit_response=$(curl -s "$BASE_URL/api/config/history?limit=1000")
large_limit_count=$(echo "$large_limit_response" | jq '.history | length')
if [ "$large_limit_count" -le 100 ]; then
    echo "✅ limit корректно ограничен максимумом (100): получено $large_limit_count записей" >> "$REPORT_FILE"
else
    echo "⚠️  limit не ограничен (получено $large_limit_count записей)" >> "$REPORT_FILE"
fi

echo "" >> "$REPORT_FILE"
echo "## Резюме" >> "$REPORT_FILE"
echo "" >> "$REPORT_FILE"
echo "Тестирование завершено. Результаты сохранены в $REPORT_FILE" >> "$REPORT_FILE"

echo ""
echo "Тестирование завершено. Отчет сохранен в $REPORT_FILE"

